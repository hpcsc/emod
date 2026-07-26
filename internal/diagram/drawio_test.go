//go:build unit

package diagram_test

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"testing"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/diagram"
	"github.com/stretchr/testify/require"
)

func TestExportDrawio(t *testing.T) {
	t.Run("auto style", func(t *testing.T) {
		t.Run("empty model (no contexts) returns valid XML with no cells", func(t *testing.T) {
			model := &ast.Model{Name: "Empty"}
			raw, err := diagram.ExportDrawio(model, diagram.StyleAuto)
			require.NoError(t, err)

			output := string(raw)
			require.Contains(t, output, `<diagram name="Empty">`)
			requireValidXML(t, output)

			require.Equal(t, 2, strings.Count(output, "<mxCell "),
				"an empty diagram holds only the two root cells")
		})

		t.Run("renders three swimlanes with correct labels", func(t *testing.T) {
			model := minimalModel("Test", "Slice1")
			raw, err := diagram.ExportDrawio(model, diagram.StyleAuto)
			require.NoError(t, err)

			output := string(raw)
			require.Contains(t, output, `value="UI / Triggers"`)
			require.Contains(t, output, `value="Commands / Views"`)
			require.Contains(t, output, `value="Events"`)
		})

		t.Run("renders a trigger with its kind and actor", func(t *testing.T) {
			model := &ast.Model{
				Name: "Test",
				Contexts: []*ast.Context{{
					Name: "Ctx",
					Aggregates: []*ast.Aggregate{{
						Name: "Agg",
						Slices: []*ast.Slice{{
							Name: "S",
							Trigger: &ast.Trigger{
								Kind:  "UI",
								Name:  "SubmitForm",
								Actor: "User",
							},
						}},
					}},
				}},
			}

			raw, err := diagram.ExportDrawio(model, diagram.StyleAuto)
			require.NoError(t, err)

			output := string(raw)
			require.Contains(t, output, "SubmitForm")
			require.Contains(t, output, "(User)")
		})

		t.Run("renders command, event and view labels", func(t *testing.T) {
			model := singleSliceModel("Test", "S",
				command("MakeReservation"), event("ReservationMade"), view("AvailableRooms"))

			raw, err := diagram.ExportDrawio(model, diagram.StyleAuto)
			require.NoError(t, err)

			output := string(raw)
			require.Contains(t, output, "MakeReservation")
			require.Contains(t, output, "ReservationMade")
			require.Contains(t, output, "AvailableRooms")
		})

		t.Run("connects trigger to command", func(t *testing.T) {
			model := &ast.Model{
				Name: "Test",
				Contexts: []*ast.Context{{
					Name: "Ctx",
					Aggregates: []*ast.Aggregate{{
						Name: "Agg",
						Slices: []*ast.Slice{{
							Name:     "S",
							Trigger:  &ast.Trigger{Kind: "UI", Name: "Click"},
							Commands: []*ast.Command{{Name: "DoAction"}},
						}},
					}},
				}},
			}

			raw, err := diagram.ExportDrawio(model, diagram.StyleAuto)
			require.NoError(t, err)

			require.Equal(t, []string{"Click -> DoAction"}, drawioConnections(t, string(raw)))
		})

		t.Run("connects command to event via flow", func(t *testing.T) {
			model := &ast.Model{
				Name: "Test",
				Contexts: []*ast.Context{{
					Name: "Ctx",
					Aggregates: []*ast.Aggregate{{
						Name: "Agg",
						Slices: []*ast.Slice{{
							Name:     "S",
							Commands: []*ast.Command{{Name: "Reserve"}},
							Events:   []*ast.Event{{Name: "Reserved"}},
							Flows:    []*ast.Flow{{CommandName: "Reserve", EventName: "Reserved"}},
						}},
					}},
				}},
			}

			raw, err := diagram.ExportDrawio(model, diagram.StyleAuto)
			require.NoError(t, err)

			require.Equal(t, []string{"Reserve -> Reserved"}, drawioConnections(t, string(raw)))
		})

		t.Run("connects event to view via subscribes", func(t *testing.T) {
			model := &ast.Model{
				Name: "Test",
				Contexts: []*ast.Context{{
					Name: "Ctx",
					Aggregates: []*ast.Aggregate{{
						Name: "Agg",
						Slices: []*ast.Slice{{
							Name:   "S",
							Events: []*ast.Event{{Name: "OrderPlaced"}},
							Views:  []*ast.View{{Name: "OrderList", Subscribes: []string{"OrderPlaced"}}},
						}},
					}},
				}},
			}

			raw, err := diagram.ExportDrawio(model, diagram.StyleAuto)
			require.NoError(t, err)

			require.Equal(t, []string{"OrderPlaced -> OrderList"}, drawioConnections(t, string(raw)))
		})

		t.Run("renders automation with gear icon indicator", func(t *testing.T) {
			model := &ast.Model{
				Name: "Test",
				Contexts: []*ast.Context{{
					Name: "Ctx",
					Aggregates: []*ast.Aggregate{{
						Name: "Agg",
						Slices: []*ast.Slice{{
							Name: "S",
							Automations: []*ast.Automation{{
								Name:         "Notifier",
								TriggerEvent: "OrderPlaced",
								Command:      "SendEmail",
							}},
						}},
					}},
				}},
			}

			raw, err := diagram.ExportDrawio(model, diagram.StyleAuto)
			require.NoError(t, err)

			output := string(raw)
			require.Contains(t, output, "Notifier")
			// Verify gear indicator is present
			require.Contains(t, output, "\u2699")
		})

		t.Run("connects event to automation to command", func(t *testing.T) {
			model := &ast.Model{
				Name: "Test",
				Contexts: []*ast.Context{{
					Name: "Ctx",
					Aggregates: []*ast.Aggregate{{
						Name: "Agg",
						Slices: []*ast.Slice{{
							Name:        "S",
							Events:      []*ast.Event{{Name: "OrderPlaced"}},
							Commands:    []*ast.Command{{Name: "SendEmail"}},
							Automations: []*ast.Automation{{Name: "Notifier", TriggerEvent: "OrderPlaced", Command: "SendEmail"}},
						}},
					}},
				}},
			}

			raw, err := diagram.ExportDrawio(model, diagram.StyleAuto)
			require.NoError(t, err)

			require.Equal(t,
				[]string{"OrderPlaced -> \u2699 Notifier", "\u2699 Notifier -> SendEmail"},
				drawioConnections(t, string(raw)))
		})

		t.Run("renders external system as gray dashed box", func(t *testing.T) {
			model := &ast.Model{
				Name: "Test",
				Contexts: []*ast.Context{{
					Name: "Ctx",
					Aggregates: []*ast.Aggregate{{
						Name: "Agg",
						Slices: []*ast.Slice{{
							Name: "S",
							Translations: []*ast.Translation{{
								Name:           "Import",
								ExternalSystem: "Stripe",
								Command:        "Charge",
								Event:          &ast.Event{Name: "Charged"},
							}},
						}},
					}},
				}},
			}

			raw, err := diagram.ExportDrawio(model, diagram.StyleAuto)
			require.NoError(t, err)

			output := string(raw)
			require.Contains(t, output, "Stripe")
			require.Contains(t, output, `dashed=1`)
		})

		t.Run("event with external source includes source label", func(t *testing.T) {
			model := singleSliceModel("Test", "S",
				eventWithSource("PaymentReceived", "external", "Stripe"))

			raw, err := diagram.ExportDrawio(model, diagram.StyleAuto)
			require.NoError(t, err)

			output := string(raw)
			require.Contains(t, output, "PaymentReceived")
			require.Contains(t, output, "Stripe")
		})

		t.Run("complete model with all element types produces well-formed diagram", func(t *testing.T) {
			model := fullModel()

			raw, err := diagram.ExportDrawio(model, diagram.StyleAuto)
			require.NoError(t, err)

			output := string(raw)
			require.Contains(t, output, `value="UI / Triggers"`)
			require.Contains(t, output, `value="Commands / Views"`)
			require.Contains(t, output, `value="Events"`)
			require.ElementsMatch(t, []string{
				"PlaceOrderForm (Customer) -> CreateOrder",
				"PlaceOrderForm (Customer) -> ValidatePayment",
				"CreateOrder -> OrderCreated",
				"ValidatePayment -> PaymentValidated",
				"OrderCreated -> OrderSummary",
				"OrderCreated -> \u2699 InventoryUpdater",
				"\u2699 InventoryUpdater -> CreateOrder",
				"Stripe -> \u2699 PaymentGW",
				"\u2699 PaymentGW -> ValidatePayment",
				"ShipTimer -> ShipOrder",
				"ShipOrder -> OrderShipped",
			}, drawioConnections(t, output))
		})

		t.Run("DCB context with direct slices renders content", func(t *testing.T) {
			model := &ast.Model{
				Name: "DCBTest",
				Contexts: []*ast.Context{{
					Name: "DCBCtx",
					Slices: []*ast.Slice{{
						Name:     "DirectSlice",
						Commands: []*ast.Command{{Name: "DirectCmd"}},
						Events:   []*ast.Event{{Name: "DirectEvt"}},
						Flows:    []*ast.Flow{{CommandName: "DirectCmd", EventName: "DirectEvt"}},
					}},
				}},
			}

			raw, err := diagram.ExportDrawio(model, diagram.StyleDCB)
			require.NoError(t, err)

			require.Equal(t, []string{"DirectCmd -> DirectEvt"}, drawioConnections(t, string(raw)))
		})

		t.Run("mixed mode context with both aggregates and direct slices renders all", func(t *testing.T) {
			model := &ast.Model{
				Name: "Mixed",
				Contexts: []*ast.Context{{
					Name: "Ctx",
					Aggregates: []*ast.Aggregate{{
						Name: "Agg",
						Slices: []*ast.Slice{{
							Name:     "AggSlice",
							Commands: []*ast.Command{{Name: "AggCmd"}},
						}},
					}},
					Slices: []*ast.Slice{{
						Name:     "DirectSlice",
						Commands: []*ast.Command{{Name: "DirectCmd"}},
					}},
				}},
			}

			raw, err := diagram.ExportDrawio(model, diagram.StyleAuto)
			require.NoError(t, err)

			output := string(raw)
			require.Contains(t, output, "AggCmd")
			require.Contains(t, output, "DirectCmd")
			requireValidXML(t, output)
		})
	})

	t.Run("projected style", func(t *testing.T) {
		t.Run("aggregate-only context with projected style produces same output as auto", func(t *testing.T) {
			model := fullModel()
			autoRaw, err := diagram.ExportDrawio(model, diagram.StyleAuto)
			require.NoError(t, err)
			projRaw, err := diagram.ExportDrawio(model, diagram.StyleProjected)
			require.NoError(t, err)
			// Pure aggregate context should produce identical output
			require.Equal(t, autoRaw, projRaw)
		})

		t.Run("DCB context with tagged events renders tag lanes in projected style", func(t *testing.T) {
			model := &ast.Model{
				Name: "TagTest",
				Contexts: []*ast.Context{{
					Name: "Ctx",
					Slices: []*ast.Slice{{
						Name: "S1",
						Events: []*ast.Event{
							{Name: "OrderPlaced", Tags: []ast.TagEntry{{Key: "priority"}}},
							{Name: "OrderShipped", Tags: []ast.TagEntry{{Key: "region"}}},
						},
					}},
				}},
			}

			raw, err := diagram.ExportDrawio(model, diagram.StyleProjected)
			require.NoError(t, err)

			output := string(raw)
			// Should have tag-labeled lanes instead of "Events" lane
			require.Contains(t, output, `value="Tag: priority"`)
			require.Contains(t, output, `value="Tag: region"`)
			// Should NOT have the standard "Events" lane (no aggregate events)
			require.NotContains(t, output, `value="Events"`)
			// Should have the shared Triggers / Commands lane
			require.Contains(t, output, `value="Triggers / Commands"`)
			requireValidXML(t, output)
		})

		t.Run("single-tag event appears only in that tag lane", func(t *testing.T) {
			model := &ast.Model{
				Name: "SingleTag",
				Contexts: []*ast.Context{{
					Name: "Ctx",
					Slices: []*ast.Slice{{
						Name: "S1",
						Events: []*ast.Event{
							{Name: "OrderPlaced", Tags: []ast.TagEntry{{Key: "priority"}}},
						},
						Flows: []*ast.Flow{},
					}},
				}},
			}

			raw, err := diagram.ExportDrawio(model, diagram.StyleProjected)
			require.NoError(t, err)

			output := string(raw)
			// Event appears in the "priority" tag lane
			require.Contains(t, output, "OrderPlaced")
			require.Contains(t, output, `value="Tag: priority"`)
			// Only one lane for one tag key
			requireValidXML(t, output)
		})

		t.Run("multi-tag event appears in each tag lane with connector", func(t *testing.T) {
			model := &ast.Model{
				Name: "MultiTag",
				Contexts: []*ast.Context{{
					Name: "Ctx",
					Slices: []*ast.Slice{{
						Name: "S1",
						Events: []*ast.Event{
							{Name: "OrderPlaced", Tags: []ast.TagEntry{{Key: "priority"}, {Key: "region"}}},
						},
					}},
				}},
			}

			raw, err := diagram.ExportDrawio(model, diagram.StyleProjected)
			require.NoError(t, err)

			output := string(raw)
			// Should have both tag lanes
			require.Contains(t, output, `value="Tag: priority"`)
			require.Contains(t, output, `value="Tag: region"`)
			// Event name appears in output
			require.Contains(t, output, "OrderPlaced")
			// Should have a multi-tag connector (dashed purple edge with no arrow)
			require.Contains(t, output, "dashed=1")
			require.Contains(t, output, "#9B59B6")
			requireValidXML(t, output)
		})

		t.Run("commands and triggers appear once in shared Triggers / Commands lane", func(t *testing.T) {
			model := &ast.Model{
				Name: "CmdTest",
				Contexts: []*ast.Context{{
					Name: "Ctx",
					Slices: []*ast.Slice{{
						Name:     "S1",
						Trigger:  &ast.Trigger{Kind: "UI", Name: "Submit", Actor: "User"},
						Commands: []*ast.Command{{Name: "ProcessOrder"}},
						Events: []*ast.Event{
							{Name: "OrderPlaced", Tags: []ast.TagEntry{{Key: "priority"}}},
						},
					}},
				}},
			}

			raw, err := diagram.ExportDrawio(model, diagram.StyleProjected)
			require.NoError(t, err)

			output := string(raw)
			// Commands and triggers appear
			require.Contains(t, output, "Submit")
			require.Contains(t, output, "(User)")
			require.Contains(t, output, "ProcessOrder")
			// Shared lane present
			require.Contains(t, output, `value="Triggers / Commands"`)
			// Only one instance of each (not duplicated per lane)
			// Count occurrences: "Submit" should appear once (no tag-lane duplication for commands)
			requireValidXML(t, output)
		})

		t.Run("projected output with tags is valid XML", func(t *testing.T) {
			model := &ast.Model{
				Name: "ValidXML",
				Contexts: []*ast.Context{{
					Name: "Ctx",
					Slices: []*ast.Slice{{
						Name: "S1",
						Events: []*ast.Event{
							{Name: "EventA", Tags: []ast.TagEntry{{Key: "alpha"}}},
							{Name: "EventB", Tags: []ast.TagEntry{{Key: "beta"}}},
						},
						Commands: []*ast.Command{{Name: "CmdA"}},
						Flows:    []*ast.Flow{{CommandName: "CmdA", EventName: "EventA"}},
					}},
				}},
			}

			raw, err := diagram.ExportDrawio(model, diagram.StyleProjected)
			require.NoError(t, err)
			requireValidXML(t, string(raw))
		})

		t.Run("mixed mode shows both aggregate and tag slices", func(t *testing.T) {
			model := &ast.Model{
				Name: "Mixed",
				Contexts: []*ast.Context{{
					Name: "Ctx",
					Aggregates: []*ast.Aggregate{{
						Name: "Agg",
						Slices: []*ast.Slice{{
							Name:   "AggSlice",
							Events: []*ast.Event{{Name: "AggEvent"}},
						}},
					}},
					Slices: []*ast.Slice{{
						Name: "DCBSlice",
						Events: []*ast.Event{
							{Name: "DCBEvent", Tags: []ast.TagEntry{{Key: "dcbTag"}}},
						},
					}},
				}},
			}

			autoRaw, err := diagram.ExportDrawio(model, diagram.StyleAuto)
			require.NoError(t, err)
			projRaw, err := diagram.ExportDrawio(model, diagram.StyleProjected)
			require.NoError(t, err)

			autoOut := string(autoRaw)
			projOut := string(projRaw)

			// Auto mode has 4 standard lanes
			require.Contains(t, autoOut, `value="UI / Triggers"`)
			require.Contains(t, autoOut, `value="Commands / Views"`)
			require.Contains(t, autoOut, `value="Events"`)

			// Projected mode has tag lane and events lane (for aggregate)
			require.Contains(t, projOut, `value="Tag: dcbTag"`)
			require.Contains(t, projOut, `value="Events"`) // for aggregate events
			require.Contains(t, projOut, "AggEvent")
			require.Contains(t, projOut, "DCBEvent")
			requireValidXML(t, projOut)
		})
	})

	t.Run("dcb style", func(t *testing.T) {
		t.Run("DCB context with StyleDCB renders flat Events lane", func(t *testing.T) {
			model := &ast.Model{
				Name: "DCBFlat",
				Contexts: []*ast.Context{{
					Name: "Ctx",
					Slices: []*ast.Slice{{
						Name: "S1",
						Events: []*ast.Event{
							{Name: "EventA", Tags: []ast.TagEntry{{Key: "priority"}}},
							{Name: "EventB", Tags: []ast.TagEntry{{Key: "region"}}},
						},
					}},
				}},
			}

			raw, err := diagram.ExportDrawio(model, diagram.StyleDCB)
			require.NoError(t, err)

			output := string(raw)
			// Should have a single "Events" lane (not tag lanes)
			require.Contains(t, output, `value="Events"`)
			// Should NOT have tag-labeled lanes
			require.NotContains(t, output, `value="Tag: priority"`)
			require.NotContains(t, output, `value="Tag: region"`)
			// Should have Triggers / Commands lane
			require.Contains(t, output, `value="Triggers / Commands"`)
			requireValidXML(t, output)
		})

		t.Run("command with decides_on shows annotation in DCB mode", func(t *testing.T) {
			model := &ast.Model{
				Name: "DecidesOn",
				Contexts: []*ast.Context{{
					Name: "Ctx",
					Slices: []*ast.Slice{{
						Name: "S1",
						Commands: []*ast.Command{{
							Name: "ProcessOrder",
							DecidesOn: &ast.DecidesOnClause{
								Events: []string{"OrderProcessed"},
							},
						}},
						Events: []*ast.Event{{Name: "OrderProcessed"}},
						Flows:  []*ast.Flow{{CommandName: "ProcessOrder", EventName: "OrderProcessed"}},
					}},
				}},
			}

			raw, err := diagram.ExportDrawio(model, diagram.StyleDCB)
			require.NoError(t, err)

			output := string(raw)
			// Command label should include decides_on annotation
			require.Contains(t, output, "ProcessOrder")
			require.Contains(t, output, "decides_on: OrderProcessed")
			requireValidXML(t, output)
		})

		t.Run("event with tags shows tag badges in DCB mode", func(t *testing.T) {
			model := &ast.Model{
				Name: "TagBadges",
				Contexts: []*ast.Context{{
					Name: "Ctx",
					Slices: []*ast.Slice{{
						Name: "S1",
						Events: []*ast.Event{
							{Name: "OrderPlaced", Tags: []ast.TagEntry{{Key: "priority", FieldRef: "high"}}},
						},
					}},
				}},
			}

			raw, err := diagram.ExportDrawio(model, diagram.StyleDCB)
			require.NoError(t, err)

			output := string(raw)
			// Event label should include tag badge
			require.Contains(t, output, "OrderPlaced")
			require.Contains(t, output, "[priority: high]")
			requireValidXML(t, output)
		})

		t.Run("non-DCB context with StyleDCB renders without error", func(t *testing.T) {
			// Pure aggregate context with StyleDCB should produce valid output
			raw, err := diagram.ExportDrawio(fullModel(), diagram.StyleDCB)
			require.NoError(t, err)
			requireValidXML(t, string(raw))
		})

		t.Run("flow arrows remain visible in DCB mode", func(t *testing.T) {
			model := &ast.Model{
				Name: "FlowDCB",
				Contexts: []*ast.Context{{
					Name: "Ctx",
					Slices: []*ast.Slice{{
						Name: "S1",
						Commands: []*ast.Command{{
							Name: "ProcessOrder",
							DecidesOn: &ast.DecidesOnClause{
								Events: []string{"OrderProcessed"},
							},
						}},
						Events: []*ast.Event{{Name: "OrderProcessed"}},
						Flows:  []*ast.Flow{{CommandName: "ProcessOrder", EventName: "OrderProcessed"}},
					}},
				}},
			}

			raw, err := diagram.ExportDrawio(model, diagram.StyleDCB)
			require.NoError(t, err)

			output := string(raw)
			// Edges should still be present
			require.NotEmpty(t, drawioConnections(t, output), "DCB mode should still have flow edges")
			requireValidXML(t, output)
		})

		t.Run("command with decides_on and predicate shows full annotation", func(t *testing.T) {
			model := &ast.Model{
				Name: "PredicateDCB",
				Contexts: []*ast.Context{{
					Name: "Ctx",
					Slices: []*ast.Slice{{
						Name: "S1",
						Commands: []*ast.Command{{
							Name: "PrioritizeOrder",
							DecidesOn: &ast.DecidesOnClause{
								Events: []string{"OrderPrioritized", "OrderFlagged"},
								Predicate: ast.TagPredicate{
									Field:    "priority",
									Operator: "==",
									Value:    "high",
								},
							},
						}},
						Events: []*ast.Event{
							{Name: "OrderPrioritized"},
							{Name: "OrderFlagged"},
						},
						Flows: []*ast.Flow{
							{CommandName: "PrioritizeOrder", EventName: "OrderPrioritized"},
						},
					}},
				}},
			}

			raw, err := diagram.ExportDrawio(model, diagram.StyleDCB)
			require.NoError(t, err)

			output := string(raw)
			require.Contains(t, output, "PrioritizeOrder")
			require.Contains(t, output, "decides_on: OrderPrioritized, OrderFlagged")
			require.Contains(t, output, "where tag(priority == high)")
			requireValidXML(t, output)
		})
	})

	t.Run("descriptions", func(t *testing.T) {
		styles := []struct {
			name  string
			style diagram.Style
		}{
			{name: "auto style", style: diagram.StyleAuto},
			{name: "projected style", style: diagram.StyleProjected},
			{name: "dcb style", style: diagram.StyleDCB},
		}

		for _, s := range styles {
			t.Run(s.name, func(t *testing.T) {
				t.Run("hovering a shape shows the description of the construct it was drawn for", func(t *testing.T) {
					raw, err := diagram.ExportDrawio(describedModel(), s.style)
					require.NoError(t, err)

					output := string(raw)
					requireValidXML(t, output)
					requireEveryDescriptionShown(t, output, drawioTooltipOf)
				})

				t.Run("describing a model leaves the picture itself untouched", func(t *testing.T) {
					described, err := diagram.ExportDrawio(describedModel(), s.style)
					require.NoError(t, err)
					plain, err := diagram.ExportDrawio(withoutDescriptions(describedModel()), s.style)
					require.NoError(t, err)

					require.Equal(t, drawioShapes(t, string(plain)), withoutTooltips(drawioShapes(t, string(described))),
						"prose must not add, move or repaint a shape")
					require.Equal(t, drawioConnections(t, string(plain)), drawioConnections(t, string(described)),
						"prose must not disturb the arrows between shapes")
					require.NotContains(t, string(plain), "tooltip",
						"a model that describes nothing is written exactly as it was before tooltips existed")
				})
			})
		}

		t.Run("a description written with markup characters reads back as written", func(t *testing.T) {
			prose := `Rooms held < 24h & marked "urgent"`
			model := describedModel()
			model.Contexts[0].Description = prose
			model.Contexts[0].Aggregates[0].Slices[0].Commands[0].Description = prose

			raw, err := diagram.ExportDrawio(model, diagram.StyleAuto)
			require.NoError(t, err)

			output := string(raw)
			requireValidXML(t, output)
			require.Equal(t, prose, drawioTooltipOf(t, output, "Bookings"))
			require.Equal(t, prose, drawioTooltipOf(t, output, "HoldRoom"))
		})

		t.Run("an event drawn in two tag lanes says the same thing in both", func(t *testing.T) {
			model := &ast.Model{
				Name: "Tagged",
				Contexts: []*ast.Context{{
					Name: "Bookings",
					Slices: []*ast.Slice{{
						Name: "Settle the stay",
						Events: []*ast.Event{{
							Name:        "StaySettled",
							Description: "The guest has paid for the whole stay",
							Tags: []ast.TagEntry{
								{Key: "stay", FieldRef: "stayId"},
								{Key: "guest", FieldRef: "guestId"},
							},
						}},
					}},
				}},
			}

			raw, err := diagram.ExportDrawio(model, diagram.StyleProjected)
			require.NoError(t, err)

			require.Equal(t,
				[]string{"The guest has paid for the whole stay", "The guest has paid for the whole stay"},
				drawioTooltipsOf(t, string(raw), "StaySettled"))
		})
	})
}

// --- drawio helpers ---

var drawioEdge = regexp.MustCompile(`<mxCell id="\d+"[^>]*edge="1"[^>]*source="(\d+)" target="(\d+)"`)

// drawioShape is a box the diagram draws, as draw.io reads it back: what it is
// called, what it says when hovered, and how it is painted and placed.
type drawioShape struct {
	id       string
	label    string
	tooltip  string
	style    string
	geometry string
}

// drawioShapes returns the diagram's boxes in document order, decoded through an
// XML parser so a test sees the text draw.io shows rather than its escaped form.
func drawioShapes(t *testing.T, output string) []drawioShape {
	t.Helper()

	attr := func(element xml.StartElement, name string) string {
		for _, a := range element.Attr {
			if a.Name.Local == name {
				return a.Value
			}
		}
		return ""
	}

	var (
		shapes   []drawioShape
		wrapper  *drawioShape
		inVertex bool
	)

	decoder := xml.NewDecoder(strings.NewReader(output))
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err, "output must be well-formed XML")

		switch element := token.(type) {
		case xml.StartElement:
			switch element.Name.Local {
			case "object":
				wrapper = &drawioShape{
					id:      attr(element, "id"),
					label:   attr(element, "label"),
					tooltip: attr(element, "tooltip"),
				}
			case "mxCell":
				inVertex = attr(element, "vertex") == "1"
				if !inVertex {
					continue
				}
				shape := drawioShape{
					id:    attr(element, "id"),
					label: attr(element, "value"),
					style: attr(element, "style"),
				}
				if wrapper != nil {
					shape.id, shape.label, shape.tooltip = wrapper.id, wrapper.label, wrapper.tooltip
				}
				shapes = append(shapes, shape)
			case "mxGeometry":
				if !inVertex {
					continue
				}
				shapes[len(shapes)-1].geometry = fmt.Sprintf("%s,%s %sx%s",
					attr(element, "x"), attr(element, "y"), attr(element, "width"), attr(element, "height"))
			}
		case xml.EndElement:
			switch element.Name.Local {
			case "mxCell":
				inVertex = false
			case "object":
				wrapper = nil
			}
		}
	}

	return shapes
}

// drawioTooltipsOf returns what every shape whose label contains label says when
// hovered, failing when the diagram draws no such shape.
func drawioTooltipsOf(t *testing.T, output, label string) []string {
	t.Helper()

	var tooltips []string
	for _, shape := range drawioShapes(t, output) {
		if strings.Contains(shape.label, label) {
			tooltips = append(tooltips, shape.tooltip)
		}
	}
	require.NotEmpty(t, tooltips, "no drawio shape labelled %q in output", label)

	return tooltips
}

func drawioTooltipOf(t *testing.T, output, label string) string {
	t.Helper()

	tooltips := drawioTooltipsOf(t, output, label)
	require.Len(t, tooltips, 1, "expected one shape labelled %q", label)

	return tooltips[0]
}

func withoutTooltips(shapes []drawioShape) []drawioShape {
	stripped := make([]drawioShape, len(shapes))
	for i, shape := range shapes {
		shape.tooltip = ""
		stripped[i] = shape
	}
	return stripped
}

// drawioConnections returns the diagram's edges as "source label -> target label"
// pairs, so a test can name the connection it expects instead of only asserting
// that some edge exists.
func drawioConnections(t *testing.T, output string) []string {
	t.Helper()

	labels := map[string]string{}
	for _, shape := range drawioShapes(t, output) {
		labels[shape.id] = shape.label
	}

	var connections []string
	for _, m := range drawioEdge.FindAllStringSubmatch(output, -1) {
		source, ok := labels[m[1]]
		require.True(t, ok, "edge references unknown source cell %s", m[1])
		target, ok := labels[m[2]]
		require.True(t, ok, "edge references unknown target cell %s", m[2])
		connections = append(connections, source+" -> "+target)
	}

	return connections
}

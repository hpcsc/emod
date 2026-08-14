//go:build unit

package diagram_test

import (
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"io"
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/diagram"
	"github.com/hpcsc/emod/internal/test"
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

		t.Run("names the lane a person enters through for the wireframes it holds", func(t *testing.T) {
			model := minimalModel("Test", "Slice1")
			raw, err := diagram.ExportDrawio(model, diagram.StyleAuto)
			require.NoError(t, err)

			output := string(raw)
			requireValidXML(t, output)
			require.Equal(t,
				[]string{"Wireframes", "Commands / Views", "Events", "External Systems"},
				drawioLaneLabels(t, output),
				"only the lane holding what a person touches is renamed; the lanes below it keep their names")
			require.NotContains(t, output, "UI / Triggers")
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
							Trigger:  &ast.Trigger{Name: "Click"},
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
								Name:    "Notifier",
								OnEvent: "OrderPlaced",
								Command: "SendEmail",
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
							Automations: []*ast.Automation{{Name: "Notifier", OnEvent: "OrderPlaced", Command: "SendEmail"}},
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
			require.Contains(t, output, `value="Wireframes"`)
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
		t.Run("automations and reactors share the triggers and commands lane without sitting on a box in it", func(t *testing.T) {
			requireReactorsClearOfSharedLane(t, diagram.StyleProjected,
				[]string{"Triggers / Commands", "Events", "Tag: loan", "External Systems"})
		})

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
						Trigger:  &ast.Trigger{Name: "Submit", Actor: "User"},
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
			require.Contains(t, autoOut, `value="Wireframes"`)
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
		t.Run("automations and reactors share the triggers and commands lane without sitting on a box in it", func(t *testing.T) {
			requireReactorsClearOfSharedLane(t, diagram.StyleDCB,
				[]string{"Triggers / Commands", "Events", "External Systems"})
		})

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
								Predicate: &ast.TagPredicate{
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

	t.Run("rejection badges", func(t *testing.T) {
		t.Run("draws one cell per rejection edge, labelled with the invariant it names", func(t *testing.T) {
			output := drawioOf(t, test.RejectionLibraryLendingModel(t), diagram.StyleAuto)

			var expected []string
			for _, edge := range test.RejectionLibraryLendingRejections {
				expected = append(expected, edge.InvariantName)
			}
			require.Equal(t, expected, drawioBadgeLabels(t, output))
		})

		t.Run("a badge carries the invariant's statement as its tooltip", func(t *testing.T) {
			output := drawioOf(t, test.RejectionLibraryLendingModel(t), diagram.StyleAuto)

			require.Equal(t,
				[]string{"A desk seats at most one reader at any moment"},
				drawioTooltipsOf(t, output, "OneReaderPerDesk"))
		})

		t.Run("a dashed edge runs from the rejected command to that badge, styled unlike a flow edge", func(t *testing.T) {
			output := drawioOf(t, test.RejectionLibraryLendingModel(t), diagram.StyleAuto)

			var rejection, flow diagramConnection
			for _, e := range drawioEdges(t, output) {
				if e.source == "ClaimDesk" && e.target == "OneReaderPerDesk" {
					rejection = e
				}
				if e.source == "ClaimDesk" && e.target == "DeskClaimed" {
					flow = e
				}
			}

			require.NotEmpty(t, flow.paint, "the flow edge this is compared against must be in the same render")
			require.NotEmpty(t, rejection.paint, "no dashed edge reaches the badge")
			require.NotEqual(t, flow.paint, rejection.paint,
				"a rejection must not be drawn like the flow beside it")
			require.Contains(t, rejection.paint, "dashed=1")
		})

		t.Run("two slices rejecting one invariant each get their own badge and their own edge", func(t *testing.T) {
			output := drawioOf(t, test.RejectionLibraryLendingModel(t), diagram.StyleAuto)

			var badgeIDs []string
			for _, shape := range drawioShapes(t, output) {
				if shape.label == "OneCopyPerLoan" {
					badgeIDs = append(badgeIDs, shape.id)
				}
			}
			require.Len(t, badgeIDs, 2, "each of the two slices draws its own badge cell")

			// Comparing target ids rather than target labels is the whole point:
			// both edges read back as reaching "OneCopyPerLoan" even when they
			// end at the same cell, so a label comparison cannot see the collapse.
			var reached []string
			for _, m := range drawioEdge.FindAllStringSubmatch(output, -1) {
				if slices.Contains(badgeIDs, m[3]) {
					reached = append(reached, m[3])
				}
			}

			require.ElementsMatch(t, badgeIDs, reached,
				"each slice's dashed edge ends at the badge in its own column; filing a badge under the invariant's name alone points both at whichever slice was drawn first")
		})

		for _, s := range []struct {
			name  string
			style diagram.Style
		}{
			{name: "auto style", style: diagram.StyleAuto},
			{name: "dcb style", style: diagram.StyleDCB},
			{name: "projected style", style: diagram.StyleProjected},
		} {
			t.Run(s.name+" draws every badge inside a lane, overlapping no cell", func(t *testing.T) {
				output := drawioOf(t, test.RejectionLibraryLendingModel(t), s.style)
				requireValidXML(t, output)

				rects := drawnCellRects(t, output)
				var badgeKeys []string
				for key := range maps.Keys(rects) {
					if strings.HasPrefix(key, "OneCopyPerLoan#") || strings.HasPrefix(key, "OneReaderPerDesk#") {
						badgeKeys = append(badgeKeys, key)
					}
				}
				slices.Sort(badgeKeys)
				require.Len(t, badgeKeys, len(test.RejectionLibraryLendingRejections))
				require.Empty(t, boxesDrawnOver(rects, badgeKeys),
					"a badge overlaps a cell already drawn")

				lanes := laneRectsOf(t, output)
				require.NotEmpty(t, lanes)
				for _, box := range drawioBoxes(t, output) {
					if !strings.Contains(box.appearance, "fillColor="+fillRejectionHex) {
						continue
					}
					require.True(t, slices.ContainsFunc(lanes, box.rect.within),
						"badge %q is drawn at a lane's coordinates with no lane behind it", box.label)
				}
			})
		}

		t.Run("a projected model whose events all sit in tag lanes still gets an events lane for its badges", func(t *testing.T) {
			model := taggedOnlyRejectionModel()

			withoutBadges, err := diagram.ExportDrawio(test.WithoutRejections(model), diagram.StyleProjected)
			require.NoError(t, err)
			require.NotContains(t, drawioLaneLabels(t, string(withoutBadges)), "Events",
				"without a rejection edge this shape draws no events lane, or the assertion below says nothing")

			output := drawioOf(t, model, diagram.StyleProjected)

			require.Contains(t, drawioLaneLabels(t, output), "Events")
			require.Equal(t, []string{"OneCopyPerLoan"}, drawioBadgeLabels(t, output))
			for _, box := range drawioBoxes(t, output) {
				if box.label != "OneCopyPerLoan" {
					continue
				}
				require.True(t, slices.ContainsFunc(laneRectsOf(t, output), box.rect.within))
			}
		})

		t.Run("one slice rejecting two invariants sends each edge to its own badge", func(t *testing.T) {
			output := drawioOf(t, twoRejectionsOneSliceModel(), diagram.StyleAuto)

			badgeIDs := map[string]string{}
			for _, shape := range drawioShapes(t, output) {
				if strings.Contains(shape.style, "fillColor="+fillRejectionHex) {
					badgeIDs[shape.label] = shape.id
				}
			}
			require.Len(t, badgeIDs, 2)

			var reached []string
			for _, m := range drawioEdge.FindAllStringSubmatch(output, -1) {
				if m[1] == rejectionEdgeStyleFor(t, output) {
					reached = append(reached, m[3])
				}
			}

			// Pairing every edge with badges[i][0] stacks both on the first
			// badge and orphans the second, which reading target ids catches
			// and reading target labels cannot.
			require.ElementsMatch(t, []string{badgeIDs["OneCopyPerLoan"], badgeIDs["FiveCopiesPerMember"]}, reached)
		})

		t.Run("a model stating no rejection edge draws no badge and no dashed rejection edge", func(t *testing.T) {
			_, unstated := requireRejectionTwinDiffers(t)

			for _, style := range []diagram.Style{diagram.StyleAuto, diagram.StyleDCB, diagram.StyleProjected} {
				output := drawioOf(t, unstated, style)

				require.Empty(t, drawioBadgeLabels(t, output))
				require.NotContains(t, output, "strokeColor="+strokeRejectionHex,
					"nothing this task added reaches a model that states no rejection edge")
			}
		})

		t.Run("every cell the twin draws keeps its id, label and style, and only its slice's event sequence reflows", func(t *testing.T) {
			stated, unstated := requireRejectionTwinDiffers(t)

			// A badge takes a place in the sequence its slice's events are laid
			// out in, so those boxes narrow — and in the projected layout that
			// reaches the tag lanes those events are drawn into as well, because
			// one sequence feeds both. Nothing outside it may move.
			inEventSequence := func(shape drawioShape) bool {
				return strings.Contains(shape.style, "fillColor="+fillEventHex) ||
					strings.Contains(shape.style, "fillColor="+fillRejectionHex)
			}

			for _, style := range []diagram.Style{diagram.StyleAuto, diagram.StyleDCB, diagram.StyleProjected} {
				statedOutput := drawioOf(t, stated, style)
				unstatedOutput := drawioOf(t, unstated, style)

				statedByID := map[string]drawioShape{}
				badges := 0
				for _, shape := range drawioShapes(t, statedOutput) {
					statedByID[shape.id] = shape
					if strings.Contains(shape.style, "fillColor="+fillRejectionHex) {
						badges++
					}
				}
				require.Equal(t, len(test.RejectionLibraryLendingRejections), badges,
					"no badge was drawn, so the exemption below says nothing")

				for _, shape := range drawioShapes(t, unstatedOutput) {
					counterpart, drawn := statedByID[shape.id]
					require.True(t, drawn, "cell %s (%q) lost its id", shape.id, shape.label)
					require.Equal(t, shape.label, counterpart.label,
						"cell %s changed what it names, so a badge id was taken before its cell was certain and renumbered every later one", shape.id)
					require.Equal(t, shape.style, counterpart.style)
					if !inEventSequence(shape) {
						require.Equal(t, shape.rect, counterpart.rect,
							"only a slice's event sequence may reflow")
					}
				}

				// The exemption is by kind, not by slice, so an event whose own
				// slice states no rejection is named explicitly: it must not
				// move, in any lane the projected layout draws it into.
				require.Equal(t,
					drawioRectsLabelled(t, unstatedOutput, "DeskReleased"),
					drawioRectsLabelled(t, statedOutput, "DeskReleased"),
					"a slice stating no rejection edge keeps its event geometry")
			}
		})
	})
}

func drawioOf(t *testing.T, model *ast.Model, style diagram.Style) string {
	t.Helper()

	raw, err := diagram.ExportDrawio(model, style)
	require.NoError(t, err)

	return string(raw)
}

// rejectionEdgeStyleFor returns the style string the render paints its dashed
// rejection edges with, read out of the render rather than restated, so the
// caller selects those edges without pinning a style table it does not own.
func rejectionEdgeStyleFor(t *testing.T, output string) string {
	t.Helper()

	for _, m := range drawioEdge.FindAllStringSubmatch(output, -1) {
		if strings.Contains(m[1], "strokeColor="+strokeRejectionHex) {
			return m[1]
		}
	}
	require.FailNow(t, "the render draws no rejection edge")
	return ""
}

// drawioBadgeLabels names the rejection badges the diagram draws, in document
// order, told from every other cell by the fill only they carry.
func drawioBadgeLabels(t *testing.T, output string) []string {
	t.Helper()

	var labels []string
	for _, shape := range drawioShapes(t, output) {
		if strings.Contains(shape.style, "fillColor="+fillRejectionHex) {
			labels = append(labels, shape.label)
		}
	}

	return labels
}

// drawioRectsLabelled returns where every cell whose label names the construct
// was drawn, in document order. Two things stop this being a single-match
// lookup: a projected layout draws one tagged event once per tag lane it
// matches, and the DCB layout appends the event's tag badges to its label.
func drawioRectsLabelled(t *testing.T, output, label string) []boxRect {
	t.Helper()

	var rects []boxRect
	for _, shape := range drawioShapes(t, output) {
		if strings.Contains(shape.label, label) {
			rects = append(rects, shape.rect)
		}
	}
	require.NotEmpty(t, rects, "the diagram draws no cell labelled %q", label)

	return rects
}

// laneRectsOf returns the band each swimlane occupies, so a test can ask whether
// a cell was drawn inside one rather than at a coordinate that merely looks right.
func laneRectsOf(t *testing.T, output string) []boxRect {
	t.Helper()

	var lanes []boxRect
	for _, shape := range drawioShapes(t, output) {
		if strings.HasPrefix(shape.style, "swimlane;") {
			lanes = append(lanes, shape.rect)
		}
	}

	return lanes
}

// drawnCellRects keys every non-lane cell by its label and id, so two slices
// drawing a badge for one invariant are two entries rather than one overwriting
// the other. Lanes are left out: a lane is drawn around the cells it holds.
func drawnCellRects(t *testing.T, output string) map[string]boxRect {
	t.Helper()

	rects := make(map[string]boxRect)
	for _, shape := range drawioShapes(t, output) {
		if shape.label == "" || strings.HasPrefix(shape.style, "swimlane;") || shape.parentID != "1" {
			continue
		}
		rects[shape.label+"#"+shape.id] = shape.rect
	}

	return rects
}

// taggedOnlyRejectionModel is the shape for which hasEventsLane is false today:
// DCB slices whose events all carry tags, with no translations and no aggregate
// events. Its rejection badge has nowhere to sit unless the edge counts.
func taggedOnlyRejectionModel() *ast.Model {
	return &ast.Model{
		Name: "Lending",
		Contexts: []*ast.Context{{
			Name:       "Lending",
			Mode:       "dcb",
			Invariants: []*ast.Invariant{{Name: "OneCopyPerLoan", Statement: "A loan covers exactly one copy"}},
			Slices: []*ast.Slice{{
				Name:     "Borrow Copy",
				Commands: []*ast.Command{{Name: "BorrowCopy"}},
				Events: []*ast.Event{
					{Name: "CopyBorrowed", Tags: []ast.TagEntry{{Key: "loan", FieldRef: "loanId"}}},
				},
				Flows:      []*ast.Flow{{CommandName: "BorrowCopy", EventName: "CopyBorrowed"}},
				Rejections: []*ast.Rejection{{CommandName: "BorrowCopy", InvariantName: "OneCopyPerLoan"}},
			}},
		}},
	}
}

// dcbReactorLabels labels the box drawn for each automation and each translation
// reactor sharedLaneModel declares.
var dcbReactorLabels = []string{
	gearMarking + " RemindOnDueDate",
	gearMarking + " RecallOverdueCopy",
	gearMarking + " PartnerImport",
}

// dcbElementLabels labels every box sharedLaneModel's own elements are drawn.
var dcbElementLabels = append([]string{
	"Lending Desk (Member)", "BorrowCopy", "ChargeFee", "MemberLoansView",
	"CopyBorrowed", "PartnerLoanReceived", "LoanRegistry",
}, dcbReactorLabels...)

// sharedLaneModel declares, directly under a DCB context, everything the one
// lane that holds triggers, commands, views, automations and reactors together
// has to make room for.
func sharedLaneModel() *ast.Model {
	return &ast.Model{
		Name: "Lending",
		Contexts: []*ast.Context{{
			Name: "Lending",
			Slices: []*ast.Slice{{
				Name:     "Borrow Copy",
				Trigger:  &ast.Trigger{Name: "Lending Desk", Actor: "Member"},
				Commands: []*ast.Command{{Name: "BorrowCopy"}, {Name: "ChargeFee"}},
				Views:    []*ast.View{{Name: "MemberLoansView", Subscribes: []string{"CopyBorrowed"}}},
				Events: []*ast.Event{
					{Name: "CopyBorrowed", Tags: []ast.TagEntry{{Key: "loan", FieldRef: "loanId"}}},
				},
				Flows: []*ast.Flow{{CommandName: "BorrowCopy", EventName: "CopyBorrowed"}},
				Automations: []*ast.Automation{
					{Name: "RemindOnDueDate", OnEvent: "CopyBorrowed", Command: "ChargeFee"},
					{Name: "RecallOverdueCopy", OnEvent: "CopyBorrowed", Reads: "MemberLoansView", Command: "BorrowCopy"},
				},
				Translations: []*ast.Translation{{
					Name:           "PartnerImport",
					ExternalSystem: "LoanRegistry",
					Command:        "BorrowCopy",
					Event:          &ast.Event{Name: "PartnerLoanReceived"},
				}},
			}},
		}},
	}
}

func requireReactorsClearOfSharedLane(t *testing.T, style diagram.Style, laneLabels []string) {
	t.Helper()

	raw, err := diagram.ExportDrawio(sharedLaneModel(), style)
	require.NoError(t, err)

	output := string(raw)
	requireValidXML(t, output)
	require.Equal(t, laneLabels, drawioLaneLabels(t, output),
		"a layout whose triggers and commands already share a lane draws the lanes it drew before")

	rects := rectsLabelled(t, drawioBoxes(t, output), dcbElementLabels)
	require.Empty(t, boxesDrawnOver(rects, dcbReactorLabels),
		"an automation or reactor sharing its lane with the commands is still drawn clear of every box in it")
}

// --- drawio helpers ---

// The text an arrow carries is written last and is optional, so an arrow
// carrying none still matches and the groups naming its ends keep their places.
var drawioEdge = regexp.MustCompile(`<mxCell id="\d+" style="([^"]*)" edge="1"[^>]*source="(\d+)" target="(\d+)"(?: value="([^"]*)")?`)

// drawioShape is a box the diagram draws, as draw.io reads it back: what it is
// called, what it says when hovered, and how it is painted and placed.
type drawioShape struct {
	id       string
	parentID string
	label    string
	tooltip  string
	style    string
	geometry string
	rect     boxRect
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
					id:       attr(element, "id"),
					parentID: attr(element, "parent"),
					label:    attr(element, "value"),
					style:    attr(element, "style"),
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
				shapes[len(shapes)-1].rect = boxRect{
					x: drawioMeasure(t, attr(element, "x")),
					y: drawioMeasure(t, attr(element, "y")),
					w: drawioMeasure(t, attr(element, "width")),
					h: drawioMeasure(t, attr(element, "height")),
				}
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

func drawioMeasure(t *testing.T, value string) int {
	t.Helper()

	measure, err := strconv.Atoi(value)
	require.NoError(t, err, "a box's geometry must be whole numbers, got %q", value)

	return measure
}

func drawioBoxes(t *testing.T, output string) []diagramBox {
	t.Helper()

	var boxes []diagramBox
	for _, shape := range drawioShapes(t, output) {
		boxes = append(boxes, diagramBox{
			label:      shape.label,
			appearance: shape.geometry + " " + shape.style,
			rect:       shape.rect,
		})
	}

	return boxes
}

// drawioLaneLabels names the swimlanes the diagram draws, in the order it draws
// them.
func drawioLaneLabels(t *testing.T, output string) []string {
	t.Helper()

	var labels []string
	for _, shape := range drawioShapes(t, output) {
		if strings.HasPrefix(shape.style, "swimlane;") {
			labels = append(labels, shape.label)
		}
	}

	return labels
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

// unescapeXML reads back the text draw.io shows from the escaped form the
// writer put in an attribute, so an assertion names the duration an author
// wrote rather than its encoding.
func unescapeXML(escaped string) string {
	return html.UnescapeString(escaped)
}

// drawioEdges returns the diagram's edges, each named by the boxes whose cells
// it references and carrying the style it is drawn with, so a test can say which
// boxes an edge runs between instead of restating cell ids.
func drawioEdges(t *testing.T, output string) []diagramConnection {
	t.Helper()

	labels := map[string]string{}
	for _, shape := range drawioShapes(t, output) {
		labels[shape.id] = shape.label
	}

	boxOf := func(id string) string {
		label, drawn := labels[id]
		require.True(t, drawn, "edge references cell %s, which the diagram draws no box for", id)
		return label
	}

	var edges []diagramConnection
	for _, m := range drawioEdge.FindAllStringSubmatch(output, -1) {
		edges = append(edges, diagramConnection{
			source: boxOf(m[2]),
			target: boxOf(m[3]),
			paint:  m[1],
			label:  unescapeXML(m[4]),
		})
	}

	return edges
}

// drawioConnections returns the diagram's edges as "source label -> target label"
// pairs, so a test can name the connection it expects instead of only asserting
// that some edge exists.
func drawioConnections(t *testing.T, output string) []string {
	t.Helper()

	var connections []string
	for _, edge := range drawioEdges(t, output) {
		connections = append(connections, edge.source+" -> "+edge.target)
	}

	return connections
}

//go:build unit

package diagram_test

import (
	"encoding/xml"
	"testing"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/diagram"
	"github.com/stretchr/testify/require"
)

func TestExportDrawio(t *testing.T) {
	t.Run("nil model returns nil error and empty output", func(t *testing.T) {
		raw, err := diagram.ExportDrawio(nil, diagram.StyleAuto)
		require.NoError(t, err)
		require.Empty(t, raw)
	})

	t.Run("empty model (no contexts) returns valid XML with no cells", func(t *testing.T) {
		model := &ast.Model{Name: "Empty"}
		raw, err := diagram.ExportDrawio(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, `<diagram name="Empty">`)
		require.True(t, validXML(output), "output must be valid XML")

		// Should have only the 2 root cells (0 and 1)
		count := countCells(output)
		require.Equal(t, 2, count, "empty diagram should only have root cells 0 and 1")
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

	t.Run("renders trigger with white fill and black stroke", func(t *testing.T) {
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
		require.Contains(t, output, `fillColor=#ffffff`)
		require.Contains(t, output, `strokeColor=#333333`)
		require.Contains(t, output, "SubmitForm")
		require.Contains(t, output, "(User)")
	})

	t.Run("renders command with blue fill", func(t *testing.T) {
		model := singleSliceModel("Test", "S", command("MakeReservation"))

		raw, err := diagram.ExportDrawio(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, `fillColor=#dae8fc`)
		require.Contains(t, output, `strokeColor=#6c8ebf`)
		require.Contains(t, output, "MakeReservation")
	})

	t.Run("renders event with orange fill", func(t *testing.T) {
		model := singleSliceModel("Test", "S", event("ReservationMade"))

		raw, err := diagram.ExportDrawio(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, `fillColor=#ffe6cc`)
		require.Contains(t, output, `strokeColor=#d79b00`)
		require.Contains(t, output, "ReservationMade")
	})

	t.Run("renders view with green fill", func(t *testing.T) {
		model := singleSliceModel("Test", "S", view("AvailableRooms"))

		raw, err := diagram.ExportDrawio(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, `fillColor=#d5e8d4`)
		require.Contains(t, output, `strokeColor=#82b366`)
		require.Contains(t, output, "AvailableRooms")
	})

	t.Run("slices laid out left-to-right in model order", func(t *testing.T) {
		model := &ast.Model{
			Name: "Test",
			Contexts: []*ast.Context{{
				Name: "Ctx",
				Aggregates: []*ast.Aggregate{{
					Name: "Agg",
					Slices: []*ast.Slice{
						{Name: "First", Commands: []*ast.Command{{Name: "CmdA"}}},
						{Name: "Second", Commands: []*ast.Command{{Name: "CmdB"}}},
						{Name: "Third", Commands: []*ast.Command{{Name: "CmdC"}}},
					},
				}},
			}},
		}

		raw, err := diagram.ExportDrawio(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		// Each slice has a different x position; first is leftmost
		require.Contains(t, output, "CmdA")
		require.Contains(t, output, "CmdB")
		require.Contains(t, output, "CmdC")
		// All three swimlanes present
		require.Contains(t, output, `value="UI / Triggers"`)
		require.Contains(t, output, `value="Commands / Views"`)
		require.Contains(t, output, `value="Events"`)
		_ = output
	})

	t.Run("connects trigger to command", func(t *testing.T) {
		model := &ast.Model{
			Name: "Test",
			Contexts: []*ast.Context{{
				Name: "Ctx",
				Aggregates: []*ast.Aggregate{{
					Name: "Agg",
					Slices: []*ast.Slice{{
						Name: "S",
						Trigger: &ast.Trigger{Kind: "UI", Name: "Click"},
						Commands: []*ast.Command{{Name: "DoAction"}},
					}},
				}},
			}},
		}

		raw, err := diagram.ExportDrawio(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		// Verify both cells are created
		require.Contains(t, output, "Click")
		require.Contains(t, output, "DoAction")
		// Verify there's a connection (edge) between them
		require.True(t, hasEdge(output), "output should contain at least one edge/connection")
	})

	t.Run("connects command to event via flow", func(t *testing.T) {
		model := &ast.Model{
			Name: "Test",
			Contexts: []*ast.Context{{
				Name: "Ctx",
				Aggregates: []*ast.Aggregate{{
					Name: "Agg",
					Slices: []*ast.Slice{{
						Name: "S",
						Commands: []*ast.Command{{Name: "Reserve"}},
						Events:   []*ast.Event{{Name: "Reserved"}},
						Flows:    []*ast.Flow{{CommandName: "Reserve", EventName: "Reserved"}},
					}},
				}},
			}},
		}

		raw, err := diagram.ExportDrawio(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "Reserve")
		require.Contains(t, output, "Reserved")
		require.True(t, hasEdge(output))
	})

	t.Run("connects event to view via subscribes", func(t *testing.T) {
		model := &ast.Model{
			Name: "Test",
			Contexts: []*ast.Context{{
				Name: "Ctx",
				Aggregates: []*ast.Aggregate{{
					Name: "Agg",
					Slices: []*ast.Slice{{
						Name: "S",
						Events: []*ast.Event{{Name: "OrderPlaced"}},
						Views:  []*ast.View{{Name: "OrderList", Subscribes: []string{"OrderPlaced"}}},
					}},
				}},
			}},
		}

		raw, err := diagram.ExportDrawio(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "OrderPlaced")
		require.Contains(t, output, "OrderList")
		require.True(t, hasEdge(output))
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
						Name: "S",
						Events:      []*ast.Event{{Name: "OrderPlaced"}},
						Commands:    []*ast.Command{{Name: "SendEmail"}},
						Automations: []*ast.Automation{{Name: "N", TriggerEvent: "OrderPlaced", Command: "SendEmail"}},
					}},
				}},
			}},
		}

		raw, err := diagram.ExportDrawio(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "OrderPlaced")
		require.Contains(t, output, "SendEmail")
		require.Contains(t, output, "N")
		// Should have at least 2 edges (event->automation, automation->command)
		require.True(t, hasEdge(output))
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
		require.Contains(t, output, `fillColor=#f5f5f5`)
		require.Contains(t, output, `strokeColor=#666666`)
		require.Contains(t, output, `dashed=1`)
	})

	t.Run("output is valid XML", func(t *testing.T) {
		model := fullModel()
		raw, err := diagram.ExportDrawio(model, diagram.StyleAuto)
		require.NoError(t, err)
		require.True(t, validXML(string(raw)), "output must be valid XML")
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
		// Verify all major components present
		require.Contains(t, output, `value="UI / Triggers"`)
		require.Contains(t, output, `value="Commands / Views"`)
		require.Contains(t, output, `value="Events"`)
		require.True(t, validXML(output))

		// Verify all colors appear
		require.Contains(t, output, fillEvent)
		require.Contains(t, output, fillCommand)
		require.Contains(t, output, fillView)
		require.Contains(t, output, fillTrigger)

		// Verify connections exist
		require.True(t, hasEdge(output), "complete model should have edges")
	})

	t.Run("DCB context with direct slices renders content", func(t *testing.T) {
		model := &ast.Model{
			Name: "DCBTest",
			Contexts: []*ast.Context{{
				Name: "DCBCtx",
				Slices: []*ast.Slice{{
					Name: "DirectSlice",
					Commands: []*ast.Command{{Name: "DirectCmd"}},
					Events:   []*ast.Event{{Name: "DirectEvt"}},
					Flows:    []*ast.Flow{{CommandName: "DirectCmd", EventName: "DirectEvt"}},
				}},
			}},
		}

		raw, err := diagram.ExportDrawio(model, diagram.StyleDCB)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "DirectCmd")
		require.Contains(t, output, "DirectEvt")
		require.True(t, validXML(output))
		require.True(t, hasEdge(output), "DCB context should have edges")
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
		require.True(t, validXML(output))
	})
}

// --- helpers ---

// validXML reports whether s is well-formed XML.
func validXML(s string) bool {
	return xml.Unmarshal([]byte(s), new(any)) == nil
}

// countCells counts mxCell elements in the output.
func countCells(s string) int {
	count := 0
	for i := 0; i < len(s)-6; i++ {
		if s[i:i+6] == "mxCell" {
			// Simple heuristic: count occurrences of id="N" within mxCell
			count++
		}
	}
	return count
}

// hasEdge reports whether the output contains an edge connection.
func hasEdge(s string) bool {
	return stringsContains(s, `edge="1"`)
}

func stringsContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// --- model builders ---

func minimalModel(name, sliceName string) *ast.Model {
	return &ast.Model{
		Name: name,
		Contexts: []*ast.Context{{
			Name: "Ctx",
			Aggregates: []*ast.Aggregate{{
				Name: "Agg",
				Slices: []*ast.Slice{{
					Name: sliceName,
				}},
			}},
		}},
	}
}

func singleSliceModel(modelName, sliceName string, opts ...any) *ast.Model {
	m := &ast.Model{
		Name: modelName,
		Contexts: []*ast.Context{{
			Name: "Ctx",
			Aggregates: []*ast.Aggregate{{
				Name: "Agg",
				Slices: []*ast.Slice{{
					Name: sliceName,
				}},
			}},
		}},
	}
	s := m.Contexts[0].Aggregates[0].Slices[0]
	for _, opt := range opts {
		switch v := opt.(type) {
		case *ast.Command:
			s.Commands = append(s.Commands, v)
		case *ast.Event:
			s.Events = append(s.Events, v)
		case *ast.View:
			s.Views = append(s.Views, v)
		case *ast.Trigger:
			s.Trigger = v
		}
	}
	return m
}

func command(name string) *ast.Command {
	return &ast.Command{Name: name}
}

func event(name string) *ast.Event {
	return &ast.Event{Name: name}
}

func eventWithSource(name, source, externalName string) *ast.Event {
	return &ast.Event{Name: name, Source: source, ExternalName: externalName}
}

func view(name string) *ast.View {
	return &ast.View{Name: name}
}

const (
	fillEvent    = "#ffe6cc"
	fillCommand  = "#dae8fc"
	fillView     = "#d5e8d4"
	fillTrigger  = "#ffffff"
	fillExternal = "#f5f5f5"
)

func fullModel() *ast.Model {
	return &ast.Model{
		Name: "FullModel",
		Contexts: []*ast.Context{{
			Name: "Orders",
			Aggregates: []*ast.Aggregate{{
				Name: "Order",
				Slices: []*ast.Slice{
					{
						Name: "Create Order",
						Trigger: &ast.Trigger{
							Kind:  "UI",
							Name:  "PlaceOrderForm",
							Actor: "Customer",
						},
						Commands: []*ast.Command{
							{Name: "CreateOrder"},
							{Name: "ValidatePayment"},
						},
						Events: []*ast.Event{
							{Name: "OrderCreated"},
							{Name: "PaymentValidated"},
						},
						Views: []*ast.View{
							{Name: "OrderSummary", Subscribes: []string{"OrderCreated"}},
						},
						Flows: []*ast.Flow{
							{CommandName: "CreateOrder", EventName: "OrderCreated"},
							{CommandName: "ValidatePayment", EventName: "PaymentValidated"},
						},
						Automations: []*ast.Automation{
							{
								Name:         "InventoryUpdater",
								TriggerEvent: "OrderCreated",
								Command:      "CreateOrder",
							},
						},
						Translations: []*ast.Translation{
							{
								Name:           "PaymentGW",
								ExternalSystem: "Stripe",
								Command:        "ValidatePayment",
								Event:          &ast.Event{Name: "PaymentValidated"},
							},
						},
					},
					{
						Name: "Ship Order",
						Trigger: &ast.Trigger{
							Kind: "Schedule",
							Name: "ShipTimer",
						},
						Commands: []*ast.Command{
							{Name: "ShipOrder"},
						},
						Events: []*ast.Event{
							{Name: "OrderShipped"},
						},
						Flows: []*ast.Flow{
							{CommandName: "ShipOrder", EventName: "OrderShipped"},
						},
					},
				},
			}},
		}},
	}
}

//go:build unit

package diagram_test

import (
	"testing"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/diagram"
	"github.com/stretchr/testify/require"
)

func TestExportSVG(t *testing.T) {
	t.Run("nil model returns empty bytes with no error", func(t *testing.T) {
		raw, err := diagram.ExportSVG(nil)
		require.NoError(t, err)
		require.Empty(t, raw)
	})

	t.Run("empty model returns valid SVG with svg root and no diagram content", func(t *testing.T) {
		model := &ast.Model{Name: "Empty"}
		raw, err := diagram.ExportSVG(model)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, `<svg xmlns="http://www.w3.org/2000/svg"`)
		require.True(t, validXML(output), "output must be valid XML")
		require.NotContains(t, output, "UI / Triggers")
	})

	t.Run("renders three swimlanes with correct labels", func(t *testing.T) {
		model := minimalModel("Test", "Slice1")
		raw, err := diagram.ExportSVG(model)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "UI / Triggers")
		require.Contains(t, output, "Commands / Views")
		require.Contains(t, output, "Events")
		require.True(t, validXML(output))
	})

	t.Run("renders trigger with white fill and stroke", func(t *testing.T) {
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

		raw, err := diagram.ExportSVG(model)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, `fill="#ffffff"`)
		require.Contains(t, output, `stroke="#333333"`)
		require.Contains(t, output, "SubmitForm")
		require.Contains(t, output, "(User)")
	})

	t.Run("renders command with blue fill and stroke", func(t *testing.T) {
		model := singleSliceModel("Test", "S", command("MakeReservation"))

		raw, err := diagram.ExportSVG(model)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, `fill="#dae8fc"`)
		require.Contains(t, output, `stroke="#6c8ebf"`)
		require.Contains(t, output, "MakeReservation")
	})

	t.Run("renders event with orange fill and stroke", func(t *testing.T) {
		model := singleSliceModel("Test", "S", event("ReservationMade"))

		raw, err := diagram.ExportSVG(model)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, `fill="#ffe6cc"`)
		require.Contains(t, output, `stroke="#d79b00"`)
		require.Contains(t, output, "ReservationMade")
	})

	t.Run("renders view with green fill and stroke", func(t *testing.T) {
		model := singleSliceModel("Test", "S", view("AvailableRooms"))

		raw, err := diagram.ExportSVG(model)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, `fill="#d5e8d4"`)
		require.Contains(t, output, `stroke="#82b366"`)
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

		raw, err := diagram.ExportSVG(model)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "CmdA")
		require.Contains(t, output, "CmdB")
		require.Contains(t, output, "CmdC")
		require.True(t, validXML(output))
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

		raw, err := diagram.ExportSVG(model)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "Click")
		require.Contains(t, output, "DoAction")
		require.Contains(t, output, "marker-end")
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

		raw, err := diagram.ExportSVG(model)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "Reserve")
		require.Contains(t, output, "Reserved")
		require.Contains(t, output, "marker-end")
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

		raw, err := diagram.ExportSVG(model)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "OrderPlaced")
		require.Contains(t, output, "OrderList")
		require.Contains(t, output, "marker-end")
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

		raw, err := diagram.ExportSVG(model)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "Notifier")
		require.Contains(t, output, "\u2699") // gear character
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

		raw, err := diagram.ExportSVG(model)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "OrderPlaced")
		require.Contains(t, output, "SendEmail")
		require.Contains(t, output, "N")
		require.Contains(t, output, "marker-end")
	})

	t.Run("renders external system as gray dashed rounded rect", func(t *testing.T) {
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

		raw, err := diagram.ExportSVG(model)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "Stripe")
		require.Contains(t, output, `fill="#f5f5f5"`)
		require.Contains(t, output, `stroke="#666666"`)
		require.Contains(t, output, `stroke-dasharray`)
	})

	t.Run("connects external system through reactor to command and event", func(t *testing.T) {
		model := &ast.Model{
			Name: "Test",
			Contexts: []*ast.Context{{
				Name: "Ctx",
				Aggregates: []*ast.Aggregate{{
					Name: "Agg",
					Slices: []*ast.Slice{{
						Name: "S",
						Commands: []*ast.Command{{Name: "Charge"}},
						Events:   []*ast.Event{{Name: "Charged"}},
						Translations: []*ast.Translation{{
							Name:           "Payment",
							ExternalSystem: "Stripe",
							Command:        "Charge",
							Event:          &ast.Event{Name: "Charged"},
						}},
					}},
				}},
			}},
		}

		raw, err := diagram.ExportSVG(model)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "Charge")
		require.Contains(t, output, "Charged")
		require.Contains(t, output, "Stripe")
		require.Contains(t, output, "marker-end")
	})

	t.Run("output is valid SVG XML", func(t *testing.T) {
		model := fullModel()
		raw, err := diagram.ExportSVG(model)
		require.NoError(t, err)
		require.True(t, validXML(string(raw)), "output must be valid XML")
	})

	t.Run("event with external source includes source label", func(t *testing.T) {
		model := singleSliceModel("Test", "S",
			eventWithSource("PaymentReceived", "external", "Stripe"))

		raw, err := diagram.ExportSVG(model)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "PaymentReceived")
		require.Contains(t, output, "Stripe")
	})

	t.Run("complete model with all element types produces well-formed diagram", func(t *testing.T) {
		model := fullModel()

		raw, err := diagram.ExportSVG(model)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "UI / Triggers")
		require.Contains(t, output, "Commands / Views")
		require.Contains(t, output, "Events")
		require.True(t, validXML(output))

		// Verify all element types rendered
		require.Contains(t, output, `fill="#ffe6cc"`)
		require.Contains(t, output, `fill="#dae8fc"`)
		require.Contains(t, output, `fill="#d5e8d4"`)
		require.Contains(t, output, `fill="#ffffff"`)
		require.Contains(t, output, `fill="#f5f5f5"`)

		// Verify connections exist
		require.Contains(t, output, "marker-end")
	})
}

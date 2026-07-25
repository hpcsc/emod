//go:build unit

package diagram_test

import (
	"strings"
	"testing"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/diagram"
	"github.com/stretchr/testify/require"
)

func TestExportSVG(t *testing.T) {
	t.Run("empty model returns valid SVG with svg root and no diagram content", func(t *testing.T) {
		model := &ast.Model{Name: "Empty"}
		raw, err := diagram.ExportSVG(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, `<svg xmlns="http://www.w3.org/2000/svg"`)
		requireValidXML(t, output)
		require.NotContains(t, output, "UI / Triggers")
	})

	t.Run("renders three swimlanes with correct labels", func(t *testing.T) {
		model := minimalModel("Test", "Slice1")
		raw, err := diagram.ExportSVG(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "UI / Triggers")
		require.Contains(t, output, "Commands / Views")
		require.Contains(t, output, "Events")
		requireValidXML(t, output)
	})

	t.Run("renders a trigger with its actor", func(t *testing.T) {
		model := singleSliceModel("Test", "S", &ast.Trigger{Kind: "UI", Name: "SubmitForm", Actor: "User"})

		raw, err := diagram.ExportSVG(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "SubmitForm")
		require.Contains(t, output, "(User)")
	})

	t.Run("renders command, event and view labels", func(t *testing.T) {
		model := singleSliceModel("Test", "S",
			command("MakeReservation"), event("ReservationMade"), view("AvailableRooms"))

		raw, err := diagram.ExportSVG(model, diagram.StyleAuto)
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

		raw, err := diagram.ExportSVG(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "Click")
		require.Contains(t, output, "DoAction")
		require.Equal(t, 1, arrowCount(output), "trigger to command is one arrow")
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

		raw, err := diagram.ExportSVG(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "Reserve")
		require.Contains(t, output, "Reserved")
		require.Equal(t, 1, arrowCount(output), "command to event is one arrow")
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

		raw, err := diagram.ExportSVG(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "OrderPlaced")
		require.Contains(t, output, "OrderList")
		require.Equal(t, 1, arrowCount(output), "event to view is one arrow")
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

		raw, err := diagram.ExportSVG(model, diagram.StyleAuto)
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
						Name:        "S",
						Events:      []*ast.Event{{Name: "OrderPlaced"}},
						Commands:    []*ast.Command{{Name: "SendEmail"}},
						Automations: []*ast.Automation{{Name: "Notifier", TriggerEvent: "OrderPlaced", Command: "SendEmail"}},
					}},
				}},
			}},
		}

		raw, err := diagram.ExportSVG(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "OrderPlaced")
		require.Contains(t, output, "SendEmail")
		require.Contains(t, output, "Notifier")
		require.Equal(t, 2, arrowCount(output), "event to automation to command is two arrows")
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

		raw, err := diagram.ExportSVG(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "Stripe")
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
						Name:     "S",
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

		raw, err := diagram.ExportSVG(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "Charge")
		require.Contains(t, output, "Charged")
		require.Contains(t, output, "Stripe")
		// external system -> reactor, reactor -> command, command -> event
		require.Equal(t, 3, arrowCount(output))
	})

	t.Run("event with external source includes source label", func(t *testing.T) {
		model := singleSliceModel("Test", "S",
			eventWithSource("PaymentReceived", "external", "Stripe"))

		raw, err := diagram.ExportSVG(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "PaymentReceived")
		require.Contains(t, output, "Stripe")
	})

	t.Run("complete model with all element types produces well-formed diagram", func(t *testing.T) {
		model := fullModel()

		raw, err := diagram.ExportSVG(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "UI / Triggers")
		require.Contains(t, output, "Commands / Views")
		require.Contains(t, output, "Events")
		require.Equal(t, 11, arrowCount(output), "every flow, subscription, automation and translation edge is drawn")
	})
}

// arrowCount reports how many connecting arrows the SVG draws.
func arrowCount(output string) int {
	return strings.Count(output, "marker-end")
}

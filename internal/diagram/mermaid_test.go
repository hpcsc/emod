//go:build unit

package diagram_test

import (
	"testing"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/diagram"
	"github.com/stretchr/testify/require"
)

func TestExportMermaid(t *testing.T) {
	t.Run("nil model returns empty bytes with no error", func(t *testing.T) {
		raw, err := diagram.ExportMermaid(nil, diagram.StyleAuto)
		require.NoError(t, err)
		require.Empty(t, raw)
	})

	t.Run("empty model (no contexts) returns output starting with eventmodeling and no timeframe entries", func(t *testing.T) {
		model := &ast.Model{Name: "Empty"}
		raw, err := diagram.ExportMermaid(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "eventmodeling")
		require.NotContains(t, output, "tf ")
	})

	t.Run("single slice with UI trigger renders as tf NN ui TriggerName", func(t *testing.T) {
		model := singleSliceModel("Test", "S1",
			&ast.Trigger{Kind: "UI", Name: "SubmitForm"})
		raw, err := diagram.ExportMermaid(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "tf 01 ui Ctx.SubmitForm")
	})

	t.Run("single slice with schedule trigger renders as tf NN pcr TriggerName", func(t *testing.T) {
		model := singleSliceModel("Test", "S1",
			&ast.Trigger{Kind: "Schedule", Name: "NightlyBatch"})
		raw, err := diagram.ExportMermaid(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "tf 01 pcr Ctx.NightlyBatch")
	})

	t.Run("single slice with processor trigger renders as tf NN pcr TriggerName", func(t *testing.T) {
		model := singleSliceModel("Test", "S1",
			&ast.Trigger{Kind: "Processor", Name: "FileWatcher"})
		raw, err := diagram.ExportMermaid(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "tf 01 pcr Ctx.FileWatcher")
	})

	t.Run("commands render as tf NN cmd CommandName", func(t *testing.T) {
		model := singleSliceModel("Test", "S1", command("CreateOrder"))
		raw, err := diagram.ExportMermaid(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "tf 01 cmd Ctx.CreateOrder")
	})

	t.Run("events render as tf NN evt EventName", func(t *testing.T) {
		model := singleSliceModel("Test", "S1", event("OrderCreated"))
		raw, err := diagram.ExportMermaid(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "tf 01 evt Ctx.OrderCreated")
	})

	t.Run("views render as tf NN rmo ViewName", func(t *testing.T) {
		model := singleSliceModel("Test", "S1", view("OrderSummary"))
		raw, err := diagram.ExportMermaid(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "tf 01 rmo Ctx.OrderSummary")
	})

	t.Run("automations render as tf NN pcr AutomationName", func(t *testing.T) {
		model := &ast.Model{
			Name: "Test",
			Contexts: []*ast.Context{{
				Name: "Ctx",
				Aggregates: []*ast.Aggregate{{
					Name: "Agg",
					Slices: []*ast.Slice{{
						Name: "S1",
						Automations: []*ast.Automation{{
							Name: "AutoNotifier",
						}},
					}},
				}},
			}},
		}
		raw, err := diagram.ExportMermaid(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "tf 01 pcr Ctx.AutoNotifier")
	})

	t.Run("context names are prepended with dot notation", func(t *testing.T) {
		model := &ast.Model{
			Name: "Test",
			Contexts: []*ast.Context{{
				Name: "Orders",
				Aggregates: []*ast.Aggregate{{
					Name: "Order",
					Slices: []*ast.Slice{{
						Name:     "CreateOrder",
						Commands: []*ast.Command{{Name: "CreateOrder"}},
					}},
				}},
			}},
		}
		raw, err := diagram.ExportMermaid(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "tf 01 cmd Orders.CreateOrder")
	})

	t.Run("automation with TargetContext uses the target context as the namespace prefix", func(t *testing.T) {
		model := &ast.Model{
			Name: "Test",
			Contexts: []*ast.Context{{
				Name: "Orders",
				Aggregates: []*ast.Aggregate{{
					Name: "Order",
					Slices: []*ast.Slice{{
						Name: "S1",
						Automations: []*ast.Automation{{
							Name:         "Notifier",
							TargetContext: "Billing",
						}},
					}},
				}},
			}},
		}
		raw, err := diagram.ExportMermaid(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "tf 01 pcr Billing.Notifier")
	})

	t.Run("sequential numbering increments across all slices", func(t *testing.T) {
		model := &ast.Model{
			Name: "Test",
			Contexts: []*ast.Context{{
				Name: "Ctx",
				Aggregates: []*ast.Aggregate{{
					Name: "Agg",
					Slices: []*ast.Slice{
						{
							Name:     "S1",
							Commands: []*ast.Command{{Name: "CmdA"}},
						},
						{
							Name:     "S2",
							Commands: []*ast.Command{{Name: "CmdB"}},
						},
					},
				}},
			}},
		}
		raw, err := diagram.ExportMermaid(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "tf 01 cmd Ctx.CmdA")
		require.Contains(t, output, "tf 02 cmd Ctx.CmdB")
	})

	t.Run("model name appears as a comment", func(t *testing.T) {
		model := &ast.Model{
			Name: "MyModel",
			Contexts: []*ast.Context{{
				Name: "Ctx",
				Aggregates: []*ast.Aggregate{{
					Name: "Agg",
					Slices: []*ast.Slice{{
						Name:     "S1",
						Commands: []*ast.Command{{Name: "Cmd"}},
					}},
				}},
			}},
		}
		raw, err := diagram.ExportMermaid(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "% MyModel")
	})

	t.Run("slice names appear as comments", func(t *testing.T) {
		model := minimalModel("Test", "InitOrder")
		raw, err := diagram.ExportMermaid(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "% Slice: InitOrder")
	})

	t.Run("complete model with all element types produces well-formed output", func(t *testing.T) {
		model := fullModel()
		raw, err := diagram.ExportMermaid(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "eventmodeling")
		require.Contains(t, output, "% FullModel")
		require.Contains(t, output, "% Slice: Create Order")
		require.Contains(t, output, "% Slice: Ship Order")

		// First slice: Create Order — UI trigger, commands, events, view, automation
		require.Contains(t, output, "tf 01 ui Orders.PlaceOrderForm")
		require.Contains(t, output, "tf 02 cmd Orders.CreateOrder")
		require.Contains(t, output, "tf 03 cmd Orders.ValidatePayment")
		require.Contains(t, output, "tf 04 evt Orders.OrderCreated")
		require.Contains(t, output, "tf 05 evt Orders.PaymentValidated")
		require.Contains(t, output, "tf 06 rmo Orders.OrderSummary")
		require.Contains(t, output, "tf 07 pcr Orders.InventoryUpdater")
		require.Contains(t, output, "tf 08 pcr Orders.PaymentGW")

		// Second slice: Ship Order — schedule trigger, command, event
		require.Contains(t, output, "tf 09 pcr Orders.ShipTimer")
		require.Contains(t, output, "tf 10 cmd Orders.ShipOrder")
		require.Contains(t, output, "tf 11 evt Orders.OrderShipped")
	})

	t.Run("DCB context with direct slices renders timeframe entries", func(t *testing.T) {
		model := &ast.Model{
			Name: "DCBTest",
			Contexts: []*ast.Context{{
				Name: "DCBCtx",
				Slices: []*ast.Slice{{
					Name:     "DirectSlice",
					Commands: []*ast.Command{{Name: "DirectCmd"}},
				}},
			}},
		}

		raw, err := diagram.ExportMermaid(model, diagram.StyleDCB)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "eventmodeling")
		require.Contains(t, output, "tf 01 cmd DCBCtx.DirectCmd")
	})
}

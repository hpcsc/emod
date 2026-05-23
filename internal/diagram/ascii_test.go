//go:build unit

package diagram_test

import (
	"testing"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/diagram"
	"github.com/stretchr/testify/require"
)

func TestExportASCII(t *testing.T) {
	t.Run("nil model returns empty bytes with no error", func(t *testing.T) {
		raw, err := diagram.ExportASCII(nil)
		require.NoError(t, err)
		require.Empty(t, raw)
	})

	t.Run("empty model (no contexts) returns model name only", func(t *testing.T) {
		model := &ast.Model{Name: "Empty"}
		raw, err := diagram.ExportASCII(model)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "Model: Empty")
	})

	t.Run("renders trigger with kind, name, and actor", func(t *testing.T) {
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

		raw, err := diagram.ExportASCII(model)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "<<UI: SubmitForm")
		require.Contains(t, output, "(User)")
	})

	t.Run("renders trigger without actor when actor is empty", func(t *testing.T) {
		model := singleSliceModel("Test", "S",
			&ast.Trigger{Kind: "Schedule", Name: "NightlyBatch"})

		raw, err := diagram.ExportASCII(model)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "<<Schedule: NightlyBatch>>")
		require.NotContains(t, output, "NightlyBatch (")
	})

	t.Run("renders command as [CommandName]", func(t *testing.T) {
		model := singleSliceModel("Test", "S", command("CreateOrder"))

		raw, err := diagram.ExportASCII(model)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "[CreateOrder]")
	})

	t.Run("renders event as (EventName)", func(t *testing.T) {
		model := singleSliceModel("Test", "S", event("OrderCreated"))

		raw, err := diagram.ExportASCII(model)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "(OrderCreated)")
	})

	t.Run("renders view as {ViewName}", func(t *testing.T) {
		model := singleSliceModel("Test", "S", view("OrderSummary"))

		raw, err := diagram.ExportASCII(model)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "{OrderSummary}")
	})

	t.Run("renders flow as command to event arrow", func(t *testing.T) {
		model := &ast.Model{
			Name: "Test",
			Contexts: []*ast.Context{{
				Name: "Ctx",
				Aggregates: []*ast.Aggregate{{
					Name: "Agg",
					Slices: []*ast.Slice{{
						Name:     "S",
						Commands: []*ast.Command{{Name: "CreateOrder"}},
						Events:   []*ast.Event{{Name: "OrderCreated"}},
						Flows:    []*ast.Flow{{CommandName: "CreateOrder", EventName: "OrderCreated"}},
					}},
				}},
			}},
		}

		raw, err := diagram.ExportASCII(model)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "->")
		require.Contains(t, output, "[CreateOrder]")
		require.Contains(t, output, "(OrderCreated)")
	})

	t.Run("renders trigger along with flow chain", func(t *testing.T) {
		model := &ast.Model{
			Name: "Test",
			Contexts: []*ast.Context{{
				Name: "Ctx",
				Aggregates: []*ast.Aggregate{{
					Name: "Agg",
					Slices: []*ast.Slice{{
						Name:    "S",
						Trigger: &ast.Trigger{Kind: "UI", Name: "Click"},
						Commands: []*ast.Command{
							{Name: "DoAction"},
						},
						Events: []*ast.Event{
							{Name: "ActionDone"},
						},
						Flows: []*ast.Flow{
							{CommandName: "DoAction", EventName: "ActionDone"},
						},
					}},
				}},
			}},
		}

		raw, err := diagram.ExportASCII(model)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "<<UI: Click>>")
		require.Contains(t, output, "[DoAction] -> (ActionDone)")
	})

	t.Run("renders automation as event to gear to command chain", func(t *testing.T) {
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

		raw, err := diagram.ExportASCII(model)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "(OrderPlaced) -> ⚙ Notifier -> [SendEmail]")
	})

	t.Run("renders translation as system to command to event chain", func(t *testing.T) {
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

		raw, err := diagram.ExportASCII(model)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "[Stripe] -> [Charge] -> (Charged)")
	})

	t.Run("view with subscribes lists subscribed events", func(t *testing.T) {
		model := &ast.Model{
			Name: "Test",
			Contexts: []*ast.Context{{
				Name: "Ctx",
				Aggregates: []*ast.Aggregate{{
					Name: "Agg",
					Slices: []*ast.Slice{{
						Name: "S",
						Views: []*ast.View{
							{Name: "OrderList", Subscribes: []string{"OrderPlaced", "OrderUpdated"}},
						},
					}},
				}},
			}},
		}

		raw, err := diagram.ExportASCII(model)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "{OrderList} [OrderPlaced, OrderUpdated]")
	})

	t.Run("multiple slices renders sequential headers", func(t *testing.T) {
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

		raw, err := diagram.ExportASCII(model)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "=== Slice: First ===")
		require.Contains(t, output, "=== Slice: Second ===")
		require.Contains(t, output, "=== Slice: Third ===")
		// First comes before Second comes before Third
		firstIdx := findIndex(output, "=== Slice: First ===")
		secondIdx := findIndex(output, "=== Slice: Second ===")
		thirdIdx := findIndex(output, "=== Slice: Third ===")
		require.True(t, firstIdx < secondIdx, "First slice should appear before Second")
		require.True(t, secondIdx < thirdIdx, "Second slice should appear before Third")
	})

	t.Run("model name appears at the top", func(t *testing.T) {
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

		raw, err := diagram.ExportASCII(model)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "Model: MyModel")
	})

	t.Run("complete model with all element types produces well-formed output", func(t *testing.T) {
		model := fullModel()

		raw, err := diagram.ExportASCII(model)
		require.NoError(t, err)

		output := string(raw)
		// Model name
		require.Contains(t, output, "Model: FullModel")

		// Slice headers
		require.Contains(t, output, "=== Slice: Create Order ===")
		require.Contains(t, output, "=== Slice: Ship Order ===")

		// Triggers
		require.Contains(t, output, "<<UI: PlaceOrderForm (Customer)>>")
		require.Contains(t, output, "<<Schedule: ShipTimer>>")

		// Flow chains
		require.Contains(t, output, "[CreateOrder] -> (OrderCreated)")
		require.Contains(t, output, "[ValidatePayment] -> (PaymentValidated)")
		require.Contains(t, output, "[ShipOrder] -> (OrderShipped)")

		// Views with subscribers
		require.Contains(t, output, "{OrderSummary} [OrderCreated]")

		// Automation
		require.Contains(t, output, "⚙ InventoryUpdater")
		require.Contains(t, output, "(OrderCreated) -> ⚙ InventoryUpdater -> [CreateOrder]")

		// Translation
		require.Contains(t, output, "[Stripe] -> [ValidatePayment] -> (PaymentValidated)")
	})

	t.Run("output is valid ASCII text", func(t *testing.T) {
		model := fullModel()

		raw, err := diagram.ExportASCII(model)
		require.NoError(t, err)

		output := string(raw)
		for i, r := range output {
			if r > 127 {
				// The gear character ⚙ (U+2699) is allowed — it's a visual marker
				if r != '⚙' {
					t.Fatalf("non-ASCII character %U at position %d", r, i)
				}
			}
		}
	})

	t.Run("automation uses gear symbol", func(t *testing.T) {
		model := &ast.Model{
			Name: "Test",
			Contexts: []*ast.Context{{
				Name: "Ctx",
				Aggregates: []*ast.Aggregate{{
					Name: "Agg",
					Slices: []*ast.Slice{{
						Name: "S",
						Automations: []*ast.Automation{{
							Name:         "N",
							TriggerEvent: "OrderPlaced",
							Command:      "SendEmail",
						}},
					}},
				}},
			}},
		}

		raw, err := diagram.ExportASCII(model)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "\u2699") // gear character
	})
}

// findIndex returns the byte index of the first occurrence of substr in s,
// or -1 if not found.
func findIndex(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

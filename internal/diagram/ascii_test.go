//go:build unit

package diagram_test

import (
	"strings"
	"testing"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/diagram"
	"github.com/hpcsc/emod/internal/test"
	"github.com/stretchr/testify/require"
)

// libraryLendingAutomationChains is the chain drawn for each automation of
// test.AutomationScheduleLibraryLending, both slice homes together and in
// declaration order: the ones running on a cadence start from it where the ones
// activating on an event start from that event.
var libraryLendingAutomationChains = []string{
	`every "0 9 * * *" -> ⚙ RemindMemberEachMorning -> [RemindMember]`,
	`(MemberReminded) -> ⚙ RecallOnSecondReminder -> [RecallCopy]`,
	`every "15m" -> ⚙ SweepOverdueLoans -> [RecallCopy]`,
	`every "0 22 * * *" -> ⚙ CloseDesksAtNight -> [ReleaseDesk]`,
	`(DeskReleased) -> ⚙ RemindReaderOfLoans -> [RemindMember]`,
	`every "45m" -> ⚙ SweepIdleDesks -> [ReleaseDesk]`,
}

func reactorChains(output string) []string {
	var chains []string
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "⚙") {
			chains = append(chains, strings.TrimSpace(line))
		}
	}
	return chains
}

func TestExportASCII(t *testing.T) {
	t.Run("empty model (no contexts) returns model name only", func(t *testing.T) {
		model := &ast.Model{Name: "Empty"}
		raw, err := diagram.ExportASCII(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "Model: Empty")
	})

	t.Run("renders trigger with name and actor, dropping any kind", func(t *testing.T) {
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

		raw, err := diagram.ExportASCII(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "<<SubmitForm (User)>>")
		require.NotContains(t, output, "UI:")
	})

	t.Run("renders trigger with name only when actor is empty", func(t *testing.T) {
		model := singleSliceModel("Test", "S",
			&ast.Trigger{Kind: "Schedule", Name: "NightlyBatch"})

		raw, err := diagram.ExportASCII(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "<<NightlyBatch>>")
		require.NotContains(t, output, "Schedule:")
		require.NotContains(t, output, "<<:")
		require.NotContains(t, output, "NightlyBatch (")
	})

	t.Run("trigger labels omit kind whether or not one is stated", func(t *testing.T) {
		model := &ast.Model{
			Name: "Test",
			Contexts: []*ast.Context{{
				Name: "Ctx",
				Aggregates: []*ast.Aggregate{{
					Name: "Agg",
					Slices: []*ast.Slice{
						{
							Name: "S1",
							Trigger: &ast.Trigger{
								Kind:  "UI",
								Name:  "SubmitForm",
								Actor: "User",
							},
						},
						{
							Name:    "S2",
							Trigger: &ast.Trigger{Name: "HeadlessBatch"},
						},
					},
				}},
			}},
		}

		raw, err := diagram.ExportASCII(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "<<SubmitForm (User)>>")
		require.Contains(t, output, "<<HeadlessBatch>>")
		require.NotContains(t, output, "UI:")
		require.NotContains(t, output, "<<:")
	})

	t.Run("renders command as [CommandName]", func(t *testing.T) {
		model := singleSliceModel("Test", "S", command("CreateOrder"))

		raw, err := diagram.ExportASCII(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "[CreateOrder]")
	})

	t.Run("renders event as (EventName)", func(t *testing.T) {
		model := singleSliceModel("Test", "S", event("OrderCreated"))

		raw, err := diagram.ExportASCII(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "(OrderCreated)")
	})

	t.Run("renders view as {ViewName}", func(t *testing.T) {
		model := singleSliceModel("Test", "S", view("OrderSummary"))

		raw, err := diagram.ExportASCII(model, diagram.StyleAuto)
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

		raw, err := diagram.ExportASCII(model, diagram.StyleAuto)
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

		raw, err := diagram.ExportASCII(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "<<Click>>")
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
							Name:    "Notifier",
							OnEvent: "OrderPlaced",
							Command: "SendEmail",
						}},
					}},
				}},
			}},
		}

		raw, err := diagram.ExportASCII(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "(OrderPlaced) -> ⚙ Notifier -> [SendEmail]")
	})

	t.Run("chains a scheduled automation from its cadence where an event-activated one starts from its event", func(t *testing.T) {
		raw, err := diagram.ExportASCII(test.AutomationScheduleLibraryLendingModel(t), diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Equal(t, libraryLendingAutomationChains, reactorChains(output))
		require.NotContains(t, output, "()", "a chain must not start from an empty source")
	})

	t.Run("lists an event nothing references beside a scheduled automation", func(t *testing.T) {
		model := singleSliceModel("Library Lending", "Chase Overdue Copy",
			command("RecallCopy"), event("CopyRecalled"),
			&ast.Automation{Name: "SweepOverdueLoans", Schedule: "15m", Command: "RecallCopy"})

		raw, err := diagram.ExportASCII(model, diagram.StyleAuto)
		require.NoError(t, err)

		require.Contains(t, string(raw), "\n  (CopyRecalled)\n")
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

		raw, err := diagram.ExportASCII(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "[Stripe] -> ⚙ Import")
		require.Contains(t, output, "⚙ Import -> [Charge]")
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

		raw, err := diagram.ExportASCII(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "(OrderPlaced) -> {OrderList}")
		require.Contains(t, output, "(OrderUpdated) -> {OrderList}")
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

		raw, err := diagram.ExportASCII(model, diagram.StyleAuto)
		require.NoError(t, err)

		headers := []string{"=== Slice: First ===", "=== Slice: Second ===", "=== Slice: Third ==="}

		require.Equal(t, headers, appearanceOrder(t, string(raw), headers...))
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

		raw, err := diagram.ExportASCII(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "Model: MyModel")
	})

	t.Run("complete model with all element types produces well-formed output", func(t *testing.T) {
		model := fullModel()

		raw, err := diagram.ExportASCII(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		// Model name
		require.Contains(t, output, "Model: FullModel")

		// Slice headers
		require.Contains(t, output, "=== Slice: Create Order ===")
		require.Contains(t, output, "=== Slice: Ship Order ===")

		// Triggers
		require.Contains(t, output, "<<PlaceOrderForm (Customer)>>")
		require.Contains(t, output, "<<ShipTimer>>")

		// Flow chains
		require.Contains(t, output, "[CreateOrder] -> (OrderCreated)")
		require.Contains(t, output, "[ValidatePayment] -> (PaymentValidated)")
		require.Contains(t, output, "[ShipOrder] -> (OrderShipped)")

		// Views with subscribers
		require.Contains(t, output, "(OrderCreated) -> {OrderSummary}")

		// Automation
		require.Contains(t, output, "⚙ InventoryUpdater")
		require.Contains(t, output, "(OrderCreated) -> ⚙ InventoryUpdater -> [CreateOrder]")

		// Translation
		require.Contains(t, output, "[Stripe] -> ⚙ PaymentGW")
		require.Contains(t, output, "⚙ PaymentGW -> [ValidatePayment]")
		require.Contains(t, output, "[ValidatePayment] -> (PaymentValidated)")
	})

	t.Run("output is valid ASCII text", func(t *testing.T) {
		models := []struct {
			name  string
			model func(t *testing.T) *ast.Model
		}{
			{name: "every element type", model: func(*testing.T) *ast.Model { return fullModel() }},
			{name: "automations running on a cadence", model: test.AutomationScheduleLibraryLendingModel},
		}

		for _, m := range models {
			t.Run(m.name, func(t *testing.T) {
				raw, err := diagram.ExportASCII(m.model(t), diagram.StyleAuto)
				require.NoError(t, err)

				output := string(raw)
				for i, r := range output {
					// The gear ⚙ (U+2699) is the one deliberate non-ASCII marker.
					require.True(t, r <= 127 || r == '⚙',
						"unexpected non-ASCII character %U at position %d", r, i)
				}
			})
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
							Name:    "N",
							OnEvent: "OrderPlaced",
							Command: "SendEmail",
						}},
					}},
				}},
			}},
		}

		raw, err := diagram.ExportASCII(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "\u2699") // gear character
	})

	t.Run("DCB context with direct slices renders ASCII content", func(t *testing.T) {
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

		raw, err := diagram.ExportASCII(model, diagram.StyleDCB)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "Model: DCBTest")
		require.Contains(t, output, "=== Slice: DirectSlice ===")
		require.Contains(t, output, "[DirectCmd]")
	})
}

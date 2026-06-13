//go:build unit

package linter_test

import (
	"testing"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/diagnostic"
	"github.com/hpcsc/emod/internal/linter"
	"github.com/stretchr/testify/require"
)

func TestLint(t *testing.T) {
	t.Run("nil model produces no warnings", func(t *testing.T) {
		diags := linter.Lint(nil)

		require.Empty(t, diags)
	})

	t.Run("state-obsession suffix Updated triggers a warning", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Aggregates: []*ast.Aggregate{
						{
							Name: "Order",
							Slices: []*ast.Slice{
								{
									Events: []*ast.Event{
										{
											Name: "OrderUpdated",
											NamePos: ast.Position{
												Filename: "orders.emod",
												Line:     5,
												Column:   3,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		diags := linter.Lint(model)

		require.Len(t, diags, 1)
		require.Equal(t, "state-obsession", diags[0].RuleName)
		require.Equal(t, "orders.emod", diags[0].Filename)
		require.Equal(t, 5, diags[0].Line)
		require.Contains(t, diags[0].Message, "OrderUpdated")
	})

	t.Run("state-obsession detects all three suffixes", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Aggregates: []*ast.Aggregate{
						{
							Name: "Order",
							Slices: []*ast.Slice{
								{
									Events: []*ast.Event{
										{
											Name: "OrderUpdated",
											NamePos: ast.Position{
												Filename: "orders.emod",
												Line:     5,
											},
										},
										{
											Name: "AccountModified",
											NamePos: ast.Position{
												Filename: "orders.emod",
												Line:     6,
											},
										},
										{
											Name: "CustomerChanged",
											NamePos: ast.Position{
												Filename: "orders.emod",
												Line:     7,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		diags := linter.Lint(model)

		require.Len(t, diags, 3)
		for _, d := range diags {
			require.Equal(t, "state-obsession", d.RuleName)
		}
		require.Equal(t, 5, diags[0].Line)
		require.Equal(t, 6, diags[1].Line)
		require.Equal(t, 7, diags[2].Line)
	})

	t.Run("property-sourcing detects entity-field-Changed names", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Aggregates: []*ast.Aggregate{
						{
							Name: "Order",
							Slices: []*ast.Slice{
								{
									Events: []*ast.Event{
										{
											Name: "OrderStatusChanged",
											NamePos: ast.Position{
												Filename: "orders.emod",
												Line:     10,
												Column:   3,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		diags := linter.Lint(model)

		require.Len(t, diags, 1)
		require.Equal(t, "property-sourcing", diags[0].RuleName)
		require.Equal(t, "orders.emod", diags[0].Filename)
		require.Equal(t, 10, diags[0].Line)
		require.Contains(t, diags[0].Message, "Order")
		require.Contains(t, diags[0].Message, "Status")
	})

	t.Run("property-sourcing with different entity-field combination", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Customers",
					Aggregates: []*ast.Aggregate{
						{
							Name: "Customer",
							Slices: []*ast.Slice{
								{
									Events: []*ast.Event{
										{
											Name: "CustomerAddressChanged",
											NamePos: ast.Position{
												Filename: "customers.emod",
												Line:     15,
												Column:   3,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		diags := linter.Lint(model)

		require.Len(t, diags, 1)
		require.Equal(t, "property-sourcing", diags[0].RuleName)
		require.Contains(t, diags[0].Message, "Customer")
		require.Contains(t, diags[0].Message, "Address")
	})

	t.Run("command-in-disguise suffix Initiated triggers a warning", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Payments",
					Aggregates: []*ast.Aggregate{
						{
							Name: "Payment",
							Slices: []*ast.Slice{
								{
									Events: []*ast.Event{
										{
											Name: "PaymentInitiated",
											NamePos: ast.Position{
												Filename: "payments.emod",
												Line:     20,
												Column:   3,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		diags := linter.Lint(model)

		require.Len(t, diags, 1)
		require.Equal(t, "command-in-disguise", diags[0].RuleName)
		require.Equal(t, "payments.emod", diags[0].Filename)
		require.Equal(t, 20, diags[0].Line)
		require.Contains(t, diags[0].Message, "PaymentInitiated")
	})

	t.Run("compliant event names produce no warnings", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Aggregates: []*ast.Aggregate{
						{
							Name: "Order",
							Slices: []*ast.Slice{
								{
									Events: []*ast.Event{
										{Name: "OrderPlaced"},
										{Name: "RoomReserved"},
									},
								},
							},
						},
					},
				},
			},
		}

		diags := linter.Lint(model)

		require.Empty(t, diags)
	})

	t.Run("inline events within translations are also checked", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Bookings",
					Aggregates: []*ast.Aggregate{
						{
							Name: "Booking",
							Slices: []*ast.Slice{
								{
									Translations: []*ast.Translation{
										{
											Name: "ImportBooking",
											Event: &ast.Event{
												Name: "BookingUpdated",
												NamePos: ast.Position{
													Filename: "bookings.emod",
													Line:     30,
													Column:   5,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		diags := linter.Lint(model)

		require.Len(t, diags, 1)
		require.Equal(t, "state-obsession", diags[0].RuleName)
		require.Equal(t, "bookings.emod", diags[0].Filename)
		require.Equal(t, 30, diags[0].Line)
		require.Contains(t, diags[0].Message, "BookingUpdated")
	})

	t.Run("past-tense command name produces a warning", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Aggregates: []*ast.Aggregate{
						{
							Name: "Order",
							Slices: []*ast.Slice{
								{
									Commands: []*ast.Command{
										{
											Name: "OrderPlaced",
											NamePos: ast.Position{
												Filename: "orders.emod",
												Line:     8,
												Column:   3,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		diags := linter.Lint(model)

		require.Len(t, diags, 1)
		require.Equal(t, "command-past-tense", diags[0].RuleName)
		require.Equal(t, diagnostic.Warning, diags[0].Severity)
		require.Equal(t, "orders.emod", diags[0].Filename)
		require.Equal(t, 8, diags[0].Line)
		require.Contains(t, diags[0].Message, "OrderPlaced")
	})

	t.Run("imperative command names produce no warning", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Aggregates: []*ast.Aggregate{
						{
							Name: "Order",
							Slices: []*ast.Slice{
								{
									Commands: []*ast.Command{
										{
											Name: "PlaceOrder",
											NamePos: ast.Position{
												Filename: "orders.emod",
												Line:     8,
											},
										},
										{
											Name: "CancelReservation",
											NamePos: ast.Position{
												Filename: "orders.emod",
												Line:     12,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		diags := linter.Lint(model)

		require.Empty(t, diags)
	})

	t.Run("commands ending with imperative verbs that naturally end in ed produce no warning", func(t *testing.T) {
		imperativeVerbs := []struct {
			commandName string
		}{
			{"Proceed"},
			{"ProceedWithPayment"},
			{"Exceed"},
			{"ExceedLimit"},
			{"Feed"},
			{"FeedInventory"},
			{"Embed"},
			{"EmbedDocument"},
			{"Speed"},
			{"SpeedUpDelivery"},
			{"Seed"},
			{"SeedDatabase"},
			{"Shred"},
			{"ShredDocument"},
			{"Succeed"},
			{"SucceedTask"},
			{"Need"},
			{"NeedApproval"},
		}

		var commands []*ast.Command
		for _, v := range imperativeVerbs {
			commands = append(commands, &ast.Command{
				Name: v.commandName,
				NamePos: ast.Position{
					Filename: "orders.emod",
					Line:     1,
				},
			})
		}

		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Aggregates: []*ast.Aggregate{
						{
							Name: "Order",
							Slices: []*ast.Slice{
								{
									Commands: commands,
								},
							},
						},
					},
				},
			},
		}

		diags := linter.Lint(model)

		require.Empty(t, diags)
	})

	t.Run("view without View suffix produces a warning", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Aggregates: []*ast.Aggregate{
						{
							Name: "Order",
							Slices: []*ast.Slice{
								{
									Views: []*ast.View{
										{
											Name: "OrderList",
											NamePos: ast.Position{
												Filename: "orders.emod",
												Line:     15,
												Column:   3,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		diags := linter.Lint(model)

		require.Len(t, diags, 1)
		require.Equal(t, "view-naming", diags[0].RuleName)
		require.Equal(t, diagnostic.Warning, diags[0].Severity)
		require.Equal(t, "orders.emod", diags[0].Filename)
		require.Equal(t, 15, diags[0].Line)
		require.Contains(t, diags[0].Message, "OrderList")
	})

	t.Run("view with View suffix produces no warning", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Aggregates: []*ast.Aggregate{
						{
							Name: "Order",
							Slices: []*ast.Slice{
								{
									Views: []*ast.View{
										{
											Name: "OrderListView",
											NamePos: ast.Position{
												Filename: "orders.emod",
												Line:     15,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		diags := linter.Lint(model)

		require.Empty(t, diags)
	})

	t.Run("left-chair fires for command with 3 or more flows", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Aggregates: []*ast.Aggregate{
						{
							Name: "Order",
							Slices: []*ast.Slice{
								{
									Commands: []*ast.Command{
										{
											Name: "PlaceOrder",
											NamePos: ast.Position{
												Filename: "orders.emod",
												Line:     10,
												Column:   3,
											},
										},
									},
									Flows: []*ast.Flow{
										{CommandName: "PlaceOrder"},
										{CommandName: "PlaceOrder"},
										{CommandName: "PlaceOrder"},
									},
								},
							},
						},
					},
				},
			},
		}

		diags := linter.Lint(model)

		require.Len(t, diags, 1)
		require.Equal(t, "left-chair", diags[0].RuleName)
		require.Equal(t, diagnostic.Error, diags[0].Severity)
		require.Equal(t, "orders.emod", diags[0].Filename)
		require.Equal(t, 10, diags[0].Line)
		require.Contains(t, diags[0].Message, "PlaceOrder")
	})

	t.Run("left-chair not fired for command with 2 flows", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Aggregates: []*ast.Aggregate{
						{
							Name: "Order",
							Slices: []*ast.Slice{
								{
									Commands: []*ast.Command{
										{
											Name: "PlaceOrder",
											NamePos: ast.Position{
												Filename: "orders.emod",
												Line:     10,
											},
										},
									},
									Flows: []*ast.Flow{
										{CommandName: "PlaceOrder"},
										{CommandName: "PlaceOrder"},
									},
								},
							},
						},
					},
				},
			},
		}

		diags := linter.Lint(model)

		require.Empty(t, diags)
	})

	t.Run("left-chair not fired for command with 0 flows", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Aggregates: []*ast.Aggregate{
						{
							Name: "Order",
							Slices: []*ast.Slice{
								{
									Commands: []*ast.Command{
										{
											Name: "PlaceOrder",
										},
									},
								},
							},
						},
					},
				},
			},
		}

		diags := linter.Lint(model)

		require.Empty(t, diags)
	})

	t.Run("left-chair counts flows across all slices", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Aggregates: []*ast.Aggregate{
						{
							Name: "Order",
							Slices: []*ast.Slice{
								{
									Commands: []*ast.Command{
										{
											Name: "PlaceOrder",
											NamePos: ast.Position{
												Filename: "orders.emod",
												Line:     10,
											},
										},
									},
									Flows: []*ast.Flow{
										{CommandName: "PlaceOrder"},
									},
								},
								{
									Flows: []*ast.Flow{
										{CommandName: "PlaceOrder"},
										{CommandName: "PlaceOrder"},
									},
								},
							},
						},
					},
				},
			},
		}

		diags := linter.Lint(model)

		require.Len(t, diags, 1)
		require.Equal(t, "left-chair", diags[0].RuleName)
	})

	t.Run("god-view fires for view subscribing to 5 or more events", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Aggregates: []*ast.Aggregate{
						{
							Name: "Order",
							Slices: []*ast.Slice{
								{
									Views: []*ast.View{
										{
											Name: "OrderSummaryView",
											NamePos: ast.Position{
												Filename: "orders.emod",
												Line:     20,
												Column:   3,
											},
											Subscribes: []string{
												"Event1", "Event2", "Event3", "Event4", "Event5",
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		diags := linter.Lint(model)

		require.Len(t, diags, 1)
		require.Equal(t, "god-view", diags[0].RuleName)
		require.Equal(t, diagnostic.Error, diags[0].Severity)
		require.Equal(t, "orders.emod", diags[0].Filename)
		require.Equal(t, 20, diags[0].Line)
		require.Contains(t, diags[0].Message, "OrderSummaryView")
	})

	t.Run("god-view not fired for view subscribing to 4 events", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Aggregates: []*ast.Aggregate{
						{
							Name: "Order",
							Slices: []*ast.Slice{
								{
									Views: []*ast.View{
										{
											Name: "OrderSummaryView",
											NamePos: ast.Position{
												Filename: "orders.emod",
												Line:     20,
											},
											Subscribes: []string{
												"Event1", "Event2", "Event3", "Event4",
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		diags := linter.Lint(model)

		require.Empty(t, diags)
	})

	t.Run("god-view not fired for view with no subscribes", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Aggregates: []*ast.Aggregate{
						{
							Name: "Order",
							Slices: []*ast.Slice{
								{
									Views: []*ast.View{
										{
											Name: "OrderSummaryView",
											NamePos: ast.Position{
												Filename: "orders.emod",
												Line:     20,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		diags := linter.Lint(model)

		require.Empty(t, diags)
	})

	t.Run("clickbait-event fires for event with single ID field", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Aggregates: []*ast.Aggregate{
						{
							Name: "Order",
							Slices: []*ast.Slice{
								{
									Events: []*ast.Event{
										{
											Name: "OrderItemSelected",
											NamePos: ast.Position{
												Filename: "orders.emod",
												Line:     30,
												Column:   3,
											},
											Fields: []*ast.Field{
												{Name: "OrderId"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		diags := linter.Lint(model)

		require.Len(t, diags, 1)
		require.Equal(t, "clickbait-event", diags[0].RuleName)
		require.Equal(t, diagnostic.Error, diags[0].Severity)
		require.Equal(t, "orders.emod", diags[0].Filename)
		require.Equal(t, 30, diags[0].Line)
		require.Contains(t, diags[0].Message, "OrderItemSelected")
		require.Contains(t, diags[0].Message, "OrderId")
	})

	t.Run("clickbait-event not fired for event with single non-ID field", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Aggregates: []*ast.Aggregate{
						{
							Name: "Order",
							Slices: []*ast.Slice{
								{
									Events: []*ast.Event{
										{
											Name: "OrderReady",
											NamePos: ast.Position{
												Filename: "orders.emod",
												Line:     30,
											},
											Fields: []*ast.Field{
												{Name: "ReadyTime"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		diags := linter.Lint(model)

		require.Empty(t, diags)
	})

	t.Run("clickbait-event not fired for event with multiple fields including ID", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Aggregates: []*ast.Aggregate{
						{
							Name: "Order",
							Slices: []*ast.Slice{
								{
									Events: []*ast.Event{
										{
											Name: "OrderItemSelected",
											NamePos: ast.Position{
												Filename: "orders.emod",
												Line:     30,
											},
											Fields: []*ast.Field{
												{Name: "OrderId"},
												{Name: "ItemName"},
											},
										},
									},
								},
							},
		},
					},
				},
			},
		}

		diags := linter.Lint(model)

		require.Empty(t, diags)
	})

	t.Run("clickbait-event fires for inline event in translation with single ID field", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Aggregates: []*ast.Aggregate{
						{
							Name: "Order",
							Slices: []*ast.Slice{
								{
									Translations: []*ast.Translation{
										{
											Name: "ImportOrder",
											Event: &ast.Event{
												Name: "OrderImported",
												NamePos: ast.Position{
													Filename: "orders.emod",
													Line:     40,
													Column:   5,
												},
												Fields: []*ast.Field{
													{Name: "OrderId"},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		diags := linter.Lint(model)

		require.Len(t, diags, 1)
		require.Equal(t, "clickbait-event", diags[0].RuleName)
		require.Equal(t, "orders.emod", diags[0].Filename)
		require.Equal(t, 40, diags[0].Line)
	})

	t.Run("all eight rules fire in a single Lint invocation", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Aggregates: []*ast.Aggregate{
						{
							Name: "Order",
							Slices: []*ast.Slice{
								{
									Events: []*ast.Event{
										{
											Name: "OrderUpdated",
											NamePos: ast.Position{
												Filename: "orders.emod",
												Line:     1,
											},
										},
										{
											Name: "OrderStatusChanged",
											NamePos: ast.Position{
												Filename: "orders.emod",
												Line:     2,
											},
										},
										{
											Name: "PaymentInitiated",
											NamePos: ast.Position{
												Filename: "orders.emod",
												Line:     3,
											},
										},
										{
											Name: "ItemSelected",
											NamePos: ast.Position{
												Filename: "orders.emod",
												Line:     8,
											},
											Fields: []*ast.Field{
												{Name: "ItemId"},
											},
										},
									},
									Commands: []*ast.Command{
										{
											Name: "ReservationCancelled",
											NamePos: ast.Position{
												Filename: "orders.emod",
												Line:     4,
											},
										},
										{
											Name: "PlaceOrder",
											NamePos: ast.Position{
												Filename: "orders.emod",
												Line:     6,
											},
										},
									},
									Views: []*ast.View{
										{
											Name: "OrderList",
											NamePos: ast.Position{
												Filename: "orders.emod",
												Line:     5,
											},
										},
										{
											Name: "OrderSummaryView",
											NamePos: ast.Position{
												Filename: "orders.emod",
												Line:     7,
											},
											Subscribes: []string{
												"E1", "E2", "E3", "E4", "E5",
											},
										},
									},
									Flows: []*ast.Flow{
										{CommandName: "PlaceOrder"},
										{CommandName: "PlaceOrder"},
										{CommandName: "PlaceOrder"},
									},
								},
							},
						},
					},
				},
			},
		}

		diags := linter.Lint(model)

		require.Len(t, diags, 8)

		ruleNames := make(map[string]bool)
		for _, d := range diags {
			ruleNames[d.RuleName] = true
		}
		require.Len(t, ruleNames, 8)
		require.True(t, ruleNames["state-obsession"])
		require.True(t, ruleNames["property-sourcing"])
		require.True(t, ruleNames["command-in-disguise"])
		require.True(t, ruleNames["command-past-tense"])
		require.True(t, ruleNames["view-naming"])
		require.True(t, ruleNames["left-chair"])
		require.True(t, ruleNames["god-view"])
		require.True(t, ruleNames["clickbait-event"])

		// Structural rules produce Error severity
		for _, d := range diags {
			switch d.RuleName {
			case "left-chair", "god-view", "clickbait-event":
				require.Equal(t, diagnostic.Error, d.Severity, "rule %q should have Error severity", d.RuleName)
			default:
				require.Equal(t, diagnostic.Warning, d.Severity, "rule %q should have Warning severity", d.RuleName)
			}
		}

		linesByRule := make(map[string]int)
		for _, d := range diags {
			linesByRule[d.RuleName] = d.Line
		}
		require.Equal(t, 1, linesByRule["state-obsession"])
		require.Equal(t, 2, linesByRule["property-sourcing"])
		require.Equal(t, 3, linesByRule["command-in-disguise"])
		require.Equal(t, 4, linesByRule["command-past-tense"])
		require.Equal(t, 5, linesByRule["view-naming"])
		require.Equal(t, 6, linesByRule["left-chair"])
		require.Equal(t, 7, linesByRule["god-view"])
		require.Equal(t, 8, linesByRule["clickbait-event"])
	})

	t.Run("dcb-in-aggregate-mode warns on event tags in default aggregate mode", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Aggregates: []*ast.Aggregate{
						{
							Name: "Order",
							Slices: []*ast.Slice{
								{
									Events: []*ast.Event{
										{
											Name: "OrderPlaced",
											NamePos: ast.Position{
												Filename: "orders.emod",
												Line:     10,
												Column:   3,
											},
											Tags: []ast.TagEntry{
												{Key: "priority", FieldRef: "OrderId"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		diags := linter.Lint(model)

		require.Len(t, diags, 1)
		require.Equal(t, "dcb-in-aggregate-mode", diags[0].RuleName)
		require.Equal(t, diagnostic.Warning, diags[0].Severity)
		require.Equal(t, "orders.emod", diags[0].Filename)
		require.Equal(t, 10, diags[0].Line)
		require.Contains(t, diags[0].Message, "OrderPlaced")
	})

	t.Run("dcb-in-aggregate-mode warns on command decides_on in default aggregate mode", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Aggregates: []*ast.Aggregate{
						{
							Name: "Order",
							Slices: []*ast.Slice{
								{
									Commands: []*ast.Command{
										{
											Name: "PlaceOrder",
											NamePos: ast.Position{
												Filename: "orders.emod",
												Line:     10,
												Column:   3,
											},
											DecidesOn: &ast.DecidesOnClause{
												Events: []string{"OrderPlaced"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		diags := linter.Lint(model)

		require.Len(t, diags, 1)
		require.Equal(t, "dcb-in-aggregate-mode", diags[0].RuleName)
		require.Equal(t, diagnostic.Warning, diags[0].Severity)
		require.Equal(t, "orders.emod", diags[0].Filename)
		require.Equal(t, 10, diags[0].Line)
		require.Contains(t, diags[0].Message, "PlaceOrder")
	})

	t.Run("dcb-in-aggregate-mode warns on context-level slices in default aggregate mode", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Slices: []*ast.Slice{
						{
							Name: "ProcessOrder",
							NamePos: ast.Position{
								Filename: "orders.emod",
								Line:     5,
								Column:   3,
							},
						},
					},
				},
			},
		}

		diags := linter.Lint(model)

		require.Len(t, diags, 1)
		require.Equal(t, "dcb-in-aggregate-mode", diags[0].RuleName)
		require.Equal(t, diagnostic.Warning, diags[0].Severity)
		require.Equal(t, "orders.emod", diags[0].Filename)
		require.Equal(t, 5, diags[0].Line)
		require.Contains(t, diags[0].Message, "ProcessOrder")
	})

	t.Run("dcb-in-aggregate-mode warns with explicit aggregate mode", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Mode: "aggregate",
					Aggregates: []*ast.Aggregate{
						{
							Name: "Order",
							Slices: []*ast.Slice{
								{
									Events: []*ast.Event{
										{
											Name: "OrderPlaced",
											NamePos: ast.Position{
												Filename: "orders.emod",
												Line:     10,
												Column:   3,
											},
											Tags: []ast.TagEntry{
												{Key: "priority", FieldRef: "OrderId"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		diags := linter.Lint(model)

		require.Len(t, diags, 1)
		require.Equal(t, "dcb-in-aggregate-mode", diags[0].RuleName)
	})

	t.Run("aggregate-in-dcb-mode warns on aggregate blocks in dcb mode", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Mode: "dcb",
					Aggregates: []*ast.Aggregate{
						{
							Name: "Order",
							NamePos: ast.Position{
								Filename: "orders.emod",
								Line:     10,
								Column:   3,
							},
						},
					},
				},
			},
		}

		diags := linter.Lint(model)

		require.Len(t, diags, 1)
		require.Equal(t, "aggregate-in-dcb-mode", diags[0].RuleName)
		require.Equal(t, diagnostic.Warning, diags[0].Severity)
		require.Equal(t, "orders.emod", diags[0].Filename)
		require.Equal(t, 10, diags[0].Line)
		require.Contains(t, diags[0].Message, "Order")
	})

	t.Run("dcb mode without aggregate blocks produces no aggregate-in-dcb-mode warning", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Mode: "dcb",
					Slices: []*ast.Slice{
						{Name: "ProcessOrder"},
					},
				},
			},
		}

		diags := linter.Lint(model)

		require.Empty(t, diags)
	})

	t.Run("mixed mode produces no mode warnings for DCB constructs", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Mode: "mixed",
					Aggregates: []*ast.Aggregate{
						{
							Name: "Order",
							Slices: []*ast.Slice{
								{
									Events: []*ast.Event{
										{
											Name: "OrderPlaced",
											NamePos: ast.Position{
												Filename: "orders.emod",
												Line:     10,
											},
											Tags: []ast.TagEntry{
												{Key: "priority", FieldRef: "OrderId"},
											},
										},
									},
								},
							},
						},
					},
					Slices: []*ast.Slice{
						{
							Name: "ProcessOrder",
							NamePos: ast.Position{
								Filename: "orders.emod",
								Line:     20,
							},
						},
					},
				},
			},
		}

		diags := linter.Lint(model)

		require.Empty(t, diags)
	})

	t.Run("mixed mode produces no mode warnings for aggregate blocks", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Mode: "mixed",
					Aggregates: []*ast.Aggregate{
						{
							Name: "Order",
							Slices: []*ast.Slice{
								{Name: "ProcessOrder"},
							},
						},
					},
				},
			},
		}

		diags := linter.Lint(model)

		require.Empty(t, diags)
	})

	t.Run("existing checks apply to context-level slices in dcb mode", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Mode: "dcb",
					Slices: []*ast.Slice{
						{
							Name: "ProcessOrder",
							Events: []*ast.Event{
								{
									Name: "OrderUpdated",
									NamePos: ast.Position{
										Filename: "orders.emod",
										Line:     10,
										Column:   5,
									},
								},
								{
									Name: "OrderStatusChanged",
									NamePos: ast.Position{
										Filename: "orders.emod",
										Line:     11,
										Column:   5,
									},
								},
								{
									Name: "PaymentInitiated",
									NamePos: ast.Position{
										Filename: "orders.emod",
										Line:     12,
										Column:   5,
									},
								},
								{
									Name: "ItemSelected",
									NamePos: ast.Position{
										Filename: "orders.emod",
										Line:     13,
										Column:   5,
									},
									Fields: []*ast.Field{
										{Name: "ItemId"},
									},
								},
							},
							Commands: []*ast.Command{
								{
									Name: "ReservationCancelled",
									NamePos: ast.Position{
										Filename: "orders.emod",
										Line:     14,
										Column:   5,
									},
								},
								{
									Name: "PlaceOrder",
									NamePos: ast.Position{
										Filename: "orders.emod",
										Line:     15,
										Column:   5,
									},
								},
							},
							Views: []*ast.View{
								{
									Name: "OrderList",
									NamePos: ast.Position{
										Filename: "orders.emod",
										Line:     16,
										Column:   5,
									},
								},
								{
									Name: "OrderSummaryView",
									NamePos: ast.Position{
										Filename: "orders.emod",
										Line:     17,
										Column:   5,
									},
									Subscribes: []string{
										"E1", "E2", "E3", "E4", "E5",
									},
								},
							},
							Flows: []*ast.Flow{
								{CommandName: "PlaceOrder"},
								{CommandName: "PlaceOrder"},
								{CommandName: "PlaceOrder"},
							},
						},
					},
				},
			},
		}

		diags := linter.Lint(model)

		ruleNames := make(map[string]bool)
		for _, d := range diags {
			ruleNames[d.RuleName] = true
		}
		require.True(t, ruleNames["state-obsession"], "state-obsession should fire for context-level slices")
		require.True(t, ruleNames["command-in-disguise"], "command-in-disguise should fire for context-level slices")
		require.True(t, ruleNames["clickbait-event"], "clickbait-event should fire for context-level slices")
		require.True(t, ruleNames["command-past-tense"], "command-past-tense should fire for context-level slices")
		require.True(t, ruleNames["view-naming"], "view-naming should fire for context-level slices")
		require.True(t, ruleNames["left-chair"], "left-chair should fire for context-level slices")
		require.True(t, ruleNames["god-view"], "god-view should fire for context-level slices")
		// Property-sourcing does not apply to context-level slices (no aggregate name)
		require.False(t, ruleNames["property-sourcing"], "property-sourcing should NOT fire for context-level slices")
		require.Equal(t, 7, len(ruleNames), "expected 7 distinct rule names (no property-sourcing for context-level slices)")
	})

	t.Run("left-chair counts flows from context-level slices", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Mode: "dcb",
					Slices: []*ast.Slice{
						{
							Name: "FirstSlice",
							Commands: []*ast.Command{
								{
									Name: "PlaceOrder",
									NamePos: ast.Position{
										Filename: "orders.emod",
										Line:     10,
										Column:   3,
									},
								},
							},
							Flows: []*ast.Flow{
								{CommandName: "PlaceOrder"},
							},
						},
						{
							Name: "SecondSlice",
							Flows: []*ast.Flow{
								{CommandName: "PlaceOrder"},
								{CommandName: "PlaceOrder"},
							},
						},
					},
				},
			},
		}

		diags := linter.Lint(model)

		require.Len(t, diags, 1)
		require.Equal(t, "left-chair", diags[0].RuleName)
		require.Contains(t, diags[0].Message, "PlaceOrder")
	})

	t.Run("tags on events in dcb mode context-level slice do not trigger dcb-in-aggregate-mode", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Mode: "dcb",
					Slices: []*ast.Slice{
						{
							Name: "ProcessOrder",
							Events: []*ast.Event{
								{
									Name: "OrderPlaced",
									NamePos: ast.Position{
										Filename: "orders.emod",
										Line:     10,
									},
									Tags: []ast.TagEntry{
										{Key: "priority", FieldRef: "OrderId"},
									},
								},
							},
						},
					},
				},
			},
		}

		diags := linter.Lint(model)

		// No dcb-in-aggregate-mode warning since mode is dcb, not aggregate
		// No naming issues either
		require.Empty(t, diags)
	})

	t.Run("event tags in inline translations are checked in aggregate mode", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Aggregates: []*ast.Aggregate{
						{
							Name: "Order",
							Slices: []*ast.Slice{
								{
									Translations: []*ast.Translation{
										{
											Name: "ImportOrder",
											Event: &ast.Event{
												Name: "OrderImported",
												NamePos: ast.Position{
													Filename: "orders.emod",
													Line:     30,
													Column:   5,
												},
												Tags: []ast.TagEntry{
													{Key: "source", FieldRef: "OrderId"},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		diags := linter.Lint(model)

		require.Len(t, diags, 1)
		require.Equal(t, "dcb-in-aggregate-mode", diags[0].RuleName)
		require.Contains(t, diags[0].Message, "OrderImported")
	})
}

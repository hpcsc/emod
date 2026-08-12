//go:build unit

package linter_test

import (
	"testing"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/diagnostic"
	"github.com/hpcsc/emod/internal/linter"
	"github.com/hpcsc/emod/internal/test"
	"github.com/stretchr/testify/require"
)

func TestLint(t *testing.T) {
	t.Run("nil model produces no warnings", func(t *testing.T) {
		diags := linter.Lint(nil)

		require.Empty(t, diags)
	})

	t.Run("event naming", func(t *testing.T) {
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
	})

	t.Run("command naming", func(t *testing.T) {
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
	})

	t.Run("view naming", func(t *testing.T) {
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
	})

	t.Run("coupling and cohesion", func(t *testing.T) {
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
	})

	t.Run("all rules together", func(t *testing.T) {
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
	})

	t.Run("context mode mismatches", func(t *testing.T) {
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
													{Key: "region", FieldRef: "WarehouseId"},
												},
											},
										},
										Commands: []*ast.Command{
											{
												Name: "PlaceOrder",
												DecidesOn: &ast.DecidesOnClause{
													Events: []string{"OrderPlaced"},
													Predicate: &ast.TagPredicate{
														Field: "priority", Operator: "=", Value: "high",
													},
												},
											},
											{
												Name: "ShipOrder",
												DecidesOn: &ast.DecidesOnClause{
													Events: []string{"OrderPlaced"},
													Predicate: &ast.TagPredicate{
														Field: "region", Operator: "=", Value: "us",
													},
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
										Tags: []ast.TagEntry{
											{Key: "type", FieldRef: "OrderId"},
										},
									},
									{
										Name: "OrderStatusChanged",
										NamePos: ast.Position{
											Filename: "orders.emod",
											Line:     11,
											Column:   5,
										},
										Tags: []ast.TagEntry{
											{Key: "type", FieldRef: "OrderId"},
										},
									},
									{
										Name: "PaymentInitiated",
										NamePos: ast.Position{
											Filename: "orders.emod",
											Line:     12,
											Column:   5,
										},
										Tags: []ast.TagEntry{
											{Key: "type", FieldRef: "OrderId"},
										},
									},
									{
										Name: "ItemSelected",
										NamePos: ast.Position{
											Filename: "orders.emod",
											Line:     13,
											Column:   5,
										},
										Tags: []ast.TagEntry{
											{Key: "type", FieldRef: "ItemId"},
											{Key: "category", FieldRef: "ItemCategory"},
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
										DecidesOn: &ast.DecidesOnClause{
											Events: []string{"OrderUpdated", "OrderStatusChanged", "PaymentInitiated", "ItemSelected"},
											Predicate: &ast.TagPredicate{
												Field: "type", Operator: "=", Value: "standard",
											},
										},
									},
									{
										Name: "CategorizeItem",
										NamePos: ast.Position{
											Filename: "orders.emod",
											Line:     16,
											Column:   5,
										},
										DecidesOn: &ast.DecidesOnClause{
											Events: []string{"ItemSelected"},
											Predicate: &ast.TagPredicate{
												Field: "category", Operator: "=", Value: "premium",
											},
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
											{Key: "region", FieldRef: "WarehouseId"},
										},
									},
								},
								Commands: []*ast.Command{
									{
										Name: "PlaceOrder",
										DecidesOn: &ast.DecidesOnClause{
											Events: []string{"OrderPlaced"},
											Predicate: &ast.TagPredicate{
												Field: "priority", Operator: "=", Value: "high",
											},
										},
									},
									{
										Name: "ShipOrder",
										DecidesOn: &ast.DecidesOnClause{
											Events: []string{"OrderPlaced"},
											Predicate: &ast.TagPredicate{
												Field: "region", Operator: "=", Value: "us",
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
	})

	t.Run("untagged events", func(t *testing.T) {
		t.Run("untagged-event fires for event without tags in dcb mode", func(t *testing.T) {
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
											Column:   3,
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
			require.Equal(t, "dcb/untagged-event", diags[0].RuleName)
			require.Equal(t, diagnostic.Error, diags[0].Severity)
			require.Equal(t, "orders.emod", diags[0].Filename)
			require.Equal(t, 10, diags[0].Line)
			require.Contains(t, diags[0].Message, "OrderPlaced")
		})

		t.Run("untagged-event fires for event without tags in mixed mode", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Orders",
						Mode: "mixed",
						Slices: []*ast.Slice{
							{
								Name: "ProcessOrder",
								Events: []*ast.Event{
									{
										Name: "OrderPlaced",
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
			}

			diags := linter.Lint(model)

			require.Len(t, diags, 1)
			require.Equal(t, "dcb/untagged-event", diags[0].RuleName)
			require.Equal(t, diagnostic.Error, diags[0].Severity)
			require.Equal(t, "orders.emod", diags[0].Filename)
			require.Equal(t, 10, diags[0].Line)
		})

		t.Run("untagged-event does not fire for event with tags in dcb mode", func(t *testing.T) {
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
											{Key: "region", FieldRef: "WarehouseId"},
										},
									},
								},
								Commands: []*ast.Command{
									{
										Name: "PlaceOrder",
										DecidesOn: &ast.DecidesOnClause{
											Events: []string{"OrderPlaced"},
											Predicate: &ast.TagPredicate{
												Field: "priority", Operator: "=", Value: "high",
											},
										},
									},
									{
										Name: "ShipOrder",
										DecidesOn: &ast.DecidesOnClause{
											Events: []string{"OrderPlaced"},
											Predicate: &ast.TagPredicate{
												Field: "region", Operator: "=", Value: "us",
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

		t.Run("untagged-event fires for inline event in translation without tags in dcb mode", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Orders",
						Mode: "dcb",
						Slices: []*ast.Slice{
							{
								Name: "ProcessOrder",
								Translations: []*ast.Translation{
									{
										Name: "ImportOrder",
										Event: &ast.Event{
											Name: "OrderImported",
											NamePos: ast.Position{
												Filename: "orders.emod",
												Line:     20,
												Column:   5,
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
			require.Equal(t, "dcb/untagged-event", diags[0].RuleName)
			require.Equal(t, "orders.emod", diags[0].Filename)
			require.Equal(t, 20, diags[0].Line)
			require.Contains(t, diags[0].Message, "OrderImported")
		})

		t.Run("untagged-event does not fire in default aggregate mode", func(t *testing.T) {
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

			// Should get zero dcb/untagged-event diagnostics (but may get other rule hits)
			for _, d := range diags {
				require.NotEqual(t, "dcb/untagged-event", d.RuleName)
			}
			require.Empty(t, diags)
		})

		t.Run("untagged-event does not fire in explicit aggregate mode", func(t *testing.T) {
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

			for _, d := range diags {
				require.NotEqual(t, "dcb/untagged-event", d.RuleName)
			}
			require.Empty(t, diags)
		})

		t.Run("untagged-event fires for multiple untagged events in dcb mode", func(t *testing.T) {
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
									},
									{
										Name: "OrderConfirmed",
										NamePos: ast.Position{
											Filename: "orders.emod",
											Line:     11,
										},
									},
								},
							},
						},
					},
				},
			}

			diags := linter.Lint(model)

			require.Len(t, diags, 2)
			for _, d := range diags {
				require.Equal(t, "dcb/untagged-event", d.RuleName)
			}
			require.Equal(t, 10, diags[0].Line)
			require.Equal(t, 11, diags[1].Line)
		})
	})

	t.Run("dcb/query-too-broad", func(t *testing.T) {
		t.Run("dcb/query-too-broad fires for 6+ events in dcb mode", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Orders",
						Mode: "dcb",
						Slices: []*ast.Slice{
							{
								Name: "ProcessOrder",
								Commands: []*ast.Command{
									{
										Name: "PlaceOrder",
										NamePos: ast.Position{
											Filename: "orders.emod",
											Line:     10,
											Column:   3,
										},
										DecidesOn: &ast.DecidesOnClause{
											Events: []string{"E1", "E2", "E3", "E4", "E5", "E6"},
											Predicate: &ast.TagPredicate{
												Field: "type", Operator: "=", Value: "standard",
											},
										},
									},
									{
										Name: "CancelOrder",
										DecidesOn: &ast.DecidesOnClause{
											Events: []string{"E1"},
											Predicate: &ast.TagPredicate{
												Field: "region", Operator: "=", Value: "us",
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
			require.Equal(t, "dcb/query-too-broad", diags[0].RuleName)
			require.Equal(t, diagnostic.Warning, diags[0].Severity)
			require.Equal(t, "orders.emod", diags[0].Filename)
			require.Equal(t, 10, diags[0].Line)
			require.Contains(t, diags[0].Message, "PlaceOrder")
			require.Contains(t, diags[0].Message, "6")
		})

		t.Run("dcb/query-too-broad does not fire for exactly 5 events in dcb mode", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Orders",
						Mode: "dcb",
						Slices: []*ast.Slice{
							{
								Name: "ProcessOrder",
								Commands: []*ast.Command{
									{
										Name: "PlaceOrder",
										NamePos: ast.Position{
											Filename: "orders.emod",
											Line:     10,
										},
										DecidesOn: &ast.DecidesOnClause{
											Events: []string{"E1", "E2", "E3", "E4", "E5"},
											Predicate: &ast.TagPredicate{
												Field: "type", Operator: "=", Value: "standard",
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

			var found bool
			for _, d := range diags {
				if d.RuleName == "dcb/query-too-broad" {
					found = true
				}
			}
			require.False(t, found, "dcb/query-too-broad should not fire for 5 events")
		})

		t.Run("dcb/query-too-broad fires for nil predicate in dcb mode", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Orders",
						Mode: "dcb",
						Slices: []*ast.Slice{
							{
								Name: "ProcessOrder",
								Commands: []*ast.Command{
									{
										Name: "PlaceOrder",
										NamePos: ast.Position{
											Filename: "orders.emod",
											Line:     10,
											Column:   3,
										},
										DecidesOn: &ast.DecidesOnClause{
											Events: []string{"E1", "E2"},
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
			require.Equal(t, "dcb/query-too-broad", diags[0].RuleName)
			require.Equal(t, diagnostic.Warning, diags[0].Severity)
			require.Contains(t, diags[0].Message, "PlaceOrder")
			require.Contains(t, diags[0].Message, "no where clause")
		})

		t.Run("dcb/query-too-broad does not fire for tag predicate in dcb mode", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Orders",
						Mode: "dcb",
						Slices: []*ast.Slice{
							{
								Name: "ProcessOrder",
								Commands: []*ast.Command{
									{
										Name: "PlaceOrder",
										NamePos: ast.Position{
											Filename: "orders.emod",
											Line:     10,
										},
										DecidesOn: &ast.DecidesOnClause{
											Events: []string{"E1", "E2"},
											Predicate: &ast.TagPredicate{
												Field: "type", Operator: "=", Value: "standard",
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

			var found bool
			for _, d := range diags {
				if d.RuleName == "dcb/query-too-broad" {
					found = true
				}
			}
			require.False(t, found, "dcb/query-too-broad should not fire with valid tag predicate")
		})

		t.Run("dcb/query-too-broad does not fire in aggregate mode", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Orders",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Order",
								Slices: []*ast.Slice{
									{
										Name: "ProcessOrder",
										Commands: []*ast.Command{
											{
												Name: "PlaceOrder",
												NamePos: ast.Position{
													Filename: "orders.emod",
													Line:     10,
												},
												DecidesOn: &ast.DecidesOnClause{
													Events: []string{"E1", "E2", "E3", "E4", "E5", "E6"},
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

			for _, d := range diags {
				require.NotEqual(t, "dcb/query-too-broad", d.RuleName)
			}
		})

		t.Run("dcb/query-too-broad fires in mixed mode", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Orders",
						Mode: "mixed",
						Slices: []*ast.Slice{
							{
								Name: "ProcessOrder",
								Commands: []*ast.Command{
									{
										Name: "PlaceOrder",
										NamePos: ast.Position{
											Filename: "orders.emod",
											Line:     10,
											Column:   3,
										},
										DecidesOn: &ast.DecidesOnClause{
											Events: []string{"E1", "E2", "E3", "E4", "E5", "E6"},
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
			require.Equal(t, "dcb/query-too-broad", diags[0].RuleName)
			require.Equal(t, diagnostic.Warning, diags[0].Severity)
			require.Contains(t, diags[0].Message, "PlaceOrder")
		})

		t.Run("dcb/query-too-broad does not fire for command without decides_on", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Orders",
						Mode: "dcb",
						Slices: []*ast.Slice{
							{
								Name: "ProcessOrder",
								Commands: []*ast.Command{
									{
										Name: "PlaceOrder",
										NamePos: ast.Position{
											Filename: "orders.emod",
											Line:     10,
										},
									},
								},
							},
						},
					},
				},
			}

			diags := linter.Lint(model)

			var found bool
			for _, d := range diags {
				if d.RuleName == "dcb/query-too-broad" {
					found = true
				}
			}
			require.False(t, found, "dcb/query-too-broad should not fire for command without decides_on")
		})

		t.Run("dcb/query-too-broad fires for command in aggregate-level slice in mixed mode", func(t *testing.T) {
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
										Commands: []*ast.Command{
											{
												Name: "PlaceOrder",
												NamePos: ast.Position{
													Filename: "orders.emod",
													Line:     10,
													Column:   3,
												},
												DecidesOn: &ast.DecidesOnClause{
													Events: []string{"E1", "E2", "E3", "E4", "E5", "E6"},
													Predicate: &ast.TagPredicate{
														Field: "type", Operator: "=", Value: "standard",
													},
												},
											},
											{
												Name: "CancelOrder",
												DecidesOn: &ast.DecidesOnClause{
													Events: []string{"E1"},
													Predicate: &ast.TagPredicate{
														Field: "region", Operator: "=", Value: "us",
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
			require.Equal(t, "dcb/query-too-broad", diags[0].RuleName)
		})
	})

	t.Run("dcb/single-tag-everywhere", func(t *testing.T) {
		t.Run("dcb/single-tag-everywhere fires when all commands use same tag key in dcb mode", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Orders",
						NamePos: ast.Position{
							Filename: "orders.emod",
							Line:     1,
							Column:   1,
						},
						Mode: "dcb",
						Slices: []*ast.Slice{
							{
								Name: "ProcessOrder",
								Commands: []*ast.Command{
									{
										Name: "PlaceOrder",
										DecidesOn: &ast.DecidesOnClause{
											Events: []string{"OrderPlaced"},
											Predicate: &ast.TagPredicate{
												Field: "priority",
											},
										},
									},
									{
										Name: "CancelOrder",
										DecidesOn: &ast.DecidesOnClause{
											Events: []string{"OrderPlaced"},
											Predicate: &ast.TagPredicate{
												Field: "priority",
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
			require.Equal(t, "dcb/single-tag-everywhere", diags[0].RuleName)
			require.Equal(t, diagnostic.Info, diags[0].Severity)
			require.Equal(t, "orders.emod", diags[0].Filename)
			require.Equal(t, 1, diags[0].Line)
			require.Contains(t, diags[0].Message, "Orders")
			require.Contains(t, diags[0].Message, "priority")
		})

		t.Run("dcb/single-tag-everywhere does not fire when commands use different tag keys in dcb mode", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Orders",
						NamePos: ast.Position{
							Filename: "orders.emod",
							Line:     1,
						},
						Mode: "dcb",
						Slices: []*ast.Slice{
							{
								Name: "ProcessOrder",
								Commands: []*ast.Command{
									{
										Name: "PlaceOrder",
										DecidesOn: &ast.DecidesOnClause{
											Events: []string{"OrderPlaced"},
											Predicate: &ast.TagPredicate{
												Field: "priority",
											},
										},
									},
									{
										Name: "CancelOrder",
										DecidesOn: &ast.DecidesOnClause{
											Events: []string{"OrderPlaced"},
											Predicate: &ast.TagPredicate{
												Field: "region",
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

			var found bool
			for _, d := range diags {
				if d.RuleName == "dcb/single-tag-everywhere" {
					found = true
				}
			}
			require.False(t, found, "dcb/single-tag-everywhere should not fire with multiple tag keys")
		})

		t.Run("dcb/single-tag-everywhere does not fire when there are no commands", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Orders",
						NamePos: ast.Position{
							Filename: "orders.emod",
							Line:     1,
						},
						Mode: "dcb",
						Slices: []*ast.Slice{
							{
								Name: "ProcessOrder",
							},
						},
					},
				},
			}

			diags := linter.Lint(model)

			var found bool
			for _, d := range diags {
				if d.RuleName == "dcb/single-tag-everywhere" {
					found = true
				}
			}
			require.False(t, found, "dcb/single-tag-everywhere should not fire with no commands")
		})

		t.Run("dcb/single-tag-everywhere does not fire in aggregate mode", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Orders",
						NamePos: ast.Position{
							Filename: "orders.emod",
							Line:     1,
						},
						Aggregates: []*ast.Aggregate{
							{
								Name: "Order",
								Slices: []*ast.Slice{
									{
										Name: "ProcessOrder",
										Commands: []*ast.Command{
											{
												Name: "PlaceOrder",
												DecidesOn: &ast.DecidesOnClause{
													Events: []string{"OrderPlaced"},
													Predicate: &ast.TagPredicate{
														Field: "priority",
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

			var found bool
			for _, d := range diags {
				if d.RuleName == "dcb/single-tag-everywhere" {
					found = true
				}
			}
			require.False(t, found, "dcb/single-tag-everywhere should not fire in aggregate mode")
		})

		t.Run("dcb/single-tag-everywhere fires in mixed mode with single tag key", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Orders",
						NamePos: ast.Position{
							Filename: "orders.emod",
							Line:     1,
							Column:   1,
						},
						Mode: "mixed",
						Slices: []*ast.Slice{
							{
								Name: "ProcessOrder",
								Commands: []*ast.Command{
									{
										Name: "PlaceOrder",
										DecidesOn: &ast.DecidesOnClause{
											Events: []string{"OrderPlaced"},
											Predicate: &ast.TagPredicate{
												Field: "priority",
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
			require.Equal(t, "dcb/single-tag-everywhere", diags[0].RuleName)
			require.Equal(t, diagnostic.Info, diags[0].Severity)
			require.Contains(t, diags[0].Message, "Orders")
			require.Contains(t, diags[0].Message, "priority")
			require.Contains(t, diags[0].Message, "mixed")
		})

		t.Run("dcb/single-tag-everywhere does not fire when no command has decides_on", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Orders",
						NamePos: ast.Position{
							Filename: "orders.emod",
							Line:     1,
						},
						Mode: "dcb",
						Slices: []*ast.Slice{
							{
								Name: "ProcessOrder",
								Commands: []*ast.Command{
									{Name: "PlaceOrder"},
									{Name: "CancelOrder"},
								},
							},
						},
					},
				},
			}

			diags := linter.Lint(model)

			var found bool
			for _, d := range diags {
				if d.RuleName == "dcb/single-tag-everywhere" {
					found = true
				}
			}
			require.False(t, found, "dcb/single-tag-everywhere should not fire with no decides_on")
		})

		t.Run("dcb/single-tag-everywhere fires for single command with single tag key", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Orders",
						NamePos: ast.Position{
							Filename: "orders.emod",
							Line:     1,
							Column:   1,
						},
						Mode: "dcb",
						Slices: []*ast.Slice{
							{
								Name: "ProcessOrder",
								Commands: []*ast.Command{
									{
										Name: "PlaceOrder",
										DecidesOn: &ast.DecidesOnClause{
											Events: []string{"OrderPlaced"},
											Predicate: &ast.TagPredicate{
												Field: "priority",
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
			require.Equal(t, "dcb/single-tag-everywhere", diags[0].RuleName)
		})

		t.Run("dcb/single-tag-everywhere fires with nested LogicalExpr using same tag key", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Orders",
						NamePos: ast.Position{
							Filename: "orders.emod",
							Line:     1,
							Column:   1,
						},
						Mode: "dcb",
						Slices: []*ast.Slice{
							{
								Name: "ProcessOrder",
								Commands: []*ast.Command{
									{
										Name: "PlaceOrder",
										DecidesOn: &ast.DecidesOnClause{
											Events: []string{"OrderPlaced"},
											Predicate: &ast.LogicalExpr{
												Left: &ast.TagPredicate{
													Field: "priority", Operator: "=", Value: "high",
												},
												Operator: "and",
												Right: &ast.TagPredicate{
													Field: "priority", Operator: "!=", Value: "low",
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
			require.Equal(t, "dcb/single-tag-everywhere", diags[0].RuleName)
		})

		t.Run("dcb/single-tag-everywhere does not fire with nested LogicalExpr using different tag keys", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Orders",
						NamePos: ast.Position{
							Filename: "orders.emod",
							Line:     1,
						},
						Mode: "dcb",
						Slices: []*ast.Slice{
							{
								Name: "ProcessOrder",
								Commands: []*ast.Command{
									{
										Name: "PlaceOrder",
										DecidesOn: &ast.DecidesOnClause{
											Events: []string{"OrderPlaced"},
											Predicate: &ast.LogicalExpr{
												Left: &ast.TagPredicate{
													Field: "priority", Operator: "=", Value: "high",
												},
												Operator: "and",
												Right: &ast.TagPredicate{
													Field: "region", Operator: "=", Value: "us",
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

			var found bool
			for _, d := range diags {
				if d.RuleName == "dcb/single-tag-everywhere" {
					found = true
				}
			}
			require.False(t, found, "dcb/single-tag-everywhere should not fire with different tag keys in nested expr")
		})

		t.Run("dcb/single-tag-everywhere collects keys across all slices including aggregate-level", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Orders",
						NamePos: ast.Position{
							Filename: "orders.emod",
							Line:     1,
							Column:   1,
						},
						Mode: "mixed",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Order",
								Slices: []*ast.Slice{
									{
										Name: "SliceInAggregate",
										Commands: []*ast.Command{
											{
												Name: "PlaceOrder",
												DecidesOn: &ast.DecidesOnClause{
													Events: []string{"OrderPlaced"},
													Predicate: &ast.TagPredicate{
														Field: "priority",
													},
												},
											},
										},
									},
								},
							},
						},
						Slices: []*ast.Slice{
							{
								Name: "SliceAtContextLevel",
								Commands: []*ast.Command{
									{
										Name: "CancelOrder",
										DecidesOn: &ast.DecidesOnClause{
											Events: []string{"OrderCancelled"},
											Predicate: &ast.TagPredicate{
												Field: "priority",
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
			require.Equal(t, "dcb/single-tag-everywhere", diags[0].RuleName)
		})

		t.Run("dcb/single-tag-everywhere fires with NotExpr using single tag key", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Orders",
						NamePos: ast.Position{
							Filename: "orders.emod",
							Line:     1,
							Column:   1,
						},
						Mode: "dcb",
						Slices: []*ast.Slice{
							{
								Name: "ProcessOrder",
								Commands: []*ast.Command{
									{
										Name: "PlaceOrder",
										DecidesOn: &ast.DecidesOnClause{
											Events: []string{"OrderPlaced"},
											Predicate: &ast.NotExpr{
												Expr: &ast.TagPredicate{
													Field: "priority", Operator: "=", Value: "low",
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
			require.Equal(t, "dcb/single-tag-everywhere", diags[0].RuleName)
		})

		t.Run("dcb/single-tag-everywhere does not fire for commands with nil predicate", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Orders",
						NamePos: ast.Position{
							Filename: "orders.emod",
							Line:     1,
						},
						Mode: "dcb",
						Slices: []*ast.Slice{
							{
								Name: "ProcessOrder",
								Commands: []*ast.Command{
									{
										Name: "PlaceOrder",
										DecidesOn: &ast.DecidesOnClause{
											Events: []string{"OrderPlaced"},
										},
									},
								},
							},
						},
					},
				},
			}

			diags := linter.Lint(model)

			var found bool
			for _, d := range diags {
				if d.RuleName == "dcb/single-tag-everywhere" {
					found = true
				}
			}
			require.False(t, found, "dcb/single-tag-everywhere should not fire with nil predicate")
		})

		t.Run("orphan-tag-key fires for tag key declared on events but not used in any predicate in dcb mode", func(t *testing.T) {
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
											Column:   3,
										},
										Tags: []ast.TagEntry{
											{Key: "priority", FieldRef: "OrderId"},
											{Key: "region", FieldRef: "WarehouseId"},
											{Key: "source", FieldRef: "Channel"},
										},
									},
								},
								Commands: []*ast.Command{
									{
										Name: "PlaceOrder",
										DecidesOn: &ast.DecidesOnClause{
											Events: []string{"OrderPlaced"},
											Predicate: &ast.TagPredicate{
												Field: "priority", Operator: "=", Value: "high",
											},
										},
									},
									{
										Name: "CancelOrder",
										DecidesOn: &ast.DecidesOnClause{
											Events: []string{"OrderPlaced"},
											Predicate: &ast.TagPredicate{
												Field: "source", Operator: "=", Value: "web",
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

			var orphanDiags []*diagnostic.Entry
			for _, d := range diags {
				if d.RuleName == "dcb/orphan-tag-key" {
					orphanDiags = append(orphanDiags, d)
				}
			}
			require.Len(t, orphanDiags, 1)
			require.Equal(t, diagnostic.Warning, orphanDiags[0].Severity)
			require.Equal(t, "orders.emod", orphanDiags[0].Filename)
			require.Equal(t, 10, orphanDiags[0].Line)
			require.Contains(t, orphanDiags[0].Message, "region")
			require.Contains(t, orphanDiags[0].Message, "Orders")
		})
	})

	t.Run("dcb/orphan-tag-key", func(t *testing.T) {
		t.Run("orphan-tag-key does not fire when all tag keys are used in predicates", func(t *testing.T) {
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
											{Key: "region", FieldRef: "WarehouseId"},
										},
									},
								},
								Commands: []*ast.Command{
									{
										Name: "PlaceOrder",
										DecidesOn: &ast.DecidesOnClause{
											Events: []string{"OrderPlaced"},
											Predicate: &ast.TagPredicate{
												Field: "priority", Operator: "=", Value: "high",
											},
										},
									},
									{
										Name: "ShipOrder",
										DecidesOn: &ast.DecidesOnClause{
											Events: []string{"OrderPlaced"},
											Predicate: &ast.TagPredicate{
												Field: "region", Operator: "=", Value: "us",
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

			var found bool
			for _, d := range diags {
				if d.RuleName == "dcb/orphan-tag-key" {
					found = true
				}
			}
			require.False(t, found, "dcb/orphan-tag-key should not fire when all tag keys are used in predicates")
		})

		t.Run("multiple orphan keys each produce separate diagnostics in declaration order", func(t *testing.T) {
			// "source" is declared after "region" in the alphabet but before it
			// in the file, so ordering by name instead of by declaration —
			// or leaving Go's randomised map order in place — fails here.
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
											Column:   3,
										},
										Tags: []ast.TagEntry{
											{Key: "priority", FieldRef: "OrderId"},
											{Key: "source", FieldRef: "Channel"},
										},
									},
									{
										Name: "OrderShipped",
										NamePos: ast.Position{
											Filename: "orders.emod",
											Line:     20,
											Column:   3,
										},
										Tags: []ast.TagEntry{
											{Key: "priority", FieldRef: "OrderId"},
											{Key: "region", FieldRef: "WarehouseId"},
										},
									},
								},
								Commands: []*ast.Command{
									{
										Name: "PlaceOrder",
										DecidesOn: &ast.DecidesOnClause{
											Events: []string{"OrderPlaced"},
											Predicate: &ast.TagPredicate{
												Field: "priority", Operator: "=", Value: "high",
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

			var orphans []*diagnostic.Entry
			for _, d := range diags {
				if d.RuleName == "dcb/orphan-tag-key" {
					orphans = append(orphans, d)
				}
			}
			require.Len(t, orphans, 2)
			require.Equal(t, 10, orphans[0].Line)
			require.Contains(t, orphans[0].Message, `tag key "source"`)
			require.Equal(t, 20, orphans[1].Line)
			require.Contains(t, orphans[1].Message, `tag key "region"`)
		})

		t.Run("orphan-tag-key does not fire in aggregate mode", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Orders",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Order",
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
					},
				},
			}

			diags := linter.Lint(model)

			var found bool
			for _, d := range diags {
				if d.RuleName == "dcb/orphan-tag-key" {
					found = true
				}
			}
			require.False(t, found, "dcb/orphan-tag-key should not fire in aggregate mode")
		})

		t.Run("orphan-tag-key fires in mixed mode", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Orders",
						Mode: "mixed",
						Slices: []*ast.Slice{
							{
								Name: "ProcessOrder",
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
											{Key: "region", FieldRef: "WarehouseId"},
											{Key: "source", FieldRef: "Channel"},
										},
									},
								},
								Commands: []*ast.Command{
									{
										Name: "PlaceOrder",
										DecidesOn: &ast.DecidesOnClause{
											Events: []string{"OrderPlaced"},
											Predicate: &ast.TagPredicate{
												Field: "priority", Operator: "=", Value: "high",
											},
										},
									},
									{
										Name: "CancelOrder",
										DecidesOn: &ast.DecidesOnClause{
											Events: []string{"OrderPlaced"},
											Predicate: &ast.TagPredicate{
												Field: "source", Operator: "=", Value: "web",
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

			var orphanDiags []*diagnostic.Entry
			for _, d := range diags {
				if d.RuleName == "dcb/orphan-tag-key" {
					orphanDiags = append(orphanDiags, d)
				}
			}
			require.Len(t, orphanDiags, 1)
			require.Contains(t, orphanDiags[0].Message, "mixed")
			require.Contains(t, orphanDiags[0].Message, "region")
		})

		t.Run("orphan-tag-key points at the first event that declares the orphan key", func(t *testing.T) {
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
											Column:   3,
										},
										Tags: []ast.TagEntry{
											{Key: "region", FieldRef: "WarehouseId"},
										},
									},
									{
										Name: "OrderShipped",
										NamePos: ast.Position{
											Filename: "orders.emod",
											Line:     20,
											Column:   3,
										},
										Tags: []ast.TagEntry{
											{Key: "region", FieldRef: "WarehouseId"},
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
			require.Equal(t, "dcb/orphan-tag-key", diags[0].RuleName)
			require.Equal(t, 10, diags[0].Line, "diagnostic should point at the first event that declares the orphan key")
		})

		t.Run("orphan-tag-key detects orphan key from inline event in translation", func(t *testing.T) {
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
								Translations: []*ast.Translation{
									{
										Name: "ImportOrder",
										Event: &ast.Event{
											Name: "OrderImported",
											NamePos: ast.Position{
												Filename: "orders.emod",
												Line:     15,
												Column:   5,
											},
											Tags: []ast.TagEntry{
												{Key: "source", FieldRef: "OrderId"},
											},
										},
									},
								},
								Commands: []*ast.Command{
									{
										Name: "ProcessImport",
										DecidesOn: &ast.DecidesOnClause{
											Events: []string{"OrderImported"},
											Predicate: &ast.TagPredicate{
												Field: "priority", Operator: "=", Value: "high",
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

			var orphanDiags []*diagnostic.Entry
			for _, d := range diags {
				if d.RuleName == "dcb/orphan-tag-key" {
					orphanDiags = append(orphanDiags, d)
				}
			}
			require.Len(t, orphanDiags, 1)
			require.Equal(t, 15, orphanDiags[0].Line)
			require.Contains(t, orphanDiags[0].Message, "source")
		})

		t.Run("orphan-tag-key handles events with no tags", func(t *testing.T) {
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
									},
								},
							},
						},
					},
				},
			}

			diags := linter.Lint(model)

			var found bool
			for _, d := range diags {
				if d.RuleName == "dcb/orphan-tag-key" {
					found = true
				}
			}
			require.False(t, found, "dcb/orphan-tag-key should not fire when there are no tags")
		})

		t.Run("orphan-tag-key collects predicate keys from NotExpr", func(t *testing.T) {
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
											{Key: "region", FieldRef: "WarehouseId"},
											{Key: "source", FieldRef: "Channel"},
										},
									},
								},
								Commands: []*ast.Command{
									{
										Name: "PlaceOrder",
										DecidesOn: &ast.DecidesOnClause{
											Events: []string{"OrderPlaced"},
											Predicate: &ast.NotExpr{
												Expr: &ast.TagPredicate{
													Field: "priority", Operator: "=", Value: "low",
												},
											},
										},
									},
									{
										Name: "CancelOrder",
										DecidesOn: &ast.DecidesOnClause{
											Events: []string{"OrderPlaced"},
											Predicate: &ast.TagPredicate{
												Field: "source", Operator: "=", Value: "web",
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

			var orphanDiags []*diagnostic.Entry
			for _, d := range diags {
				if d.RuleName == "dcb/orphan-tag-key" {
					orphanDiags = append(orphanDiags, d)
				}
			}
			require.Len(t, orphanDiags, 1)
			require.Contains(t, orphanDiags[0].Message, "region")
		})

		t.Run("orphan-tag-key collects predicate keys from LogicalExpr", func(t *testing.T) {
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
											{Key: "region", FieldRef: "WarehouseId"},
											{Key: "source", FieldRef: "Channel"},
										},
									},
								},
								Commands: []*ast.Command{
									{
										Name: "PlaceOrder",
										DecidesOn: &ast.DecidesOnClause{
											Events: []string{"OrderPlaced"},
											Predicate: &ast.LogicalExpr{
												Left: &ast.TagPredicate{
													Field: "priority", Operator: "=", Value: "high",
												},
												Operator: "and",
												Right: &ast.TagPredicate{
													Field: "region", Operator: "=", Value: "us",
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
			require.Equal(t, "dcb/orphan-tag-key", diags[0].RuleName)
			require.Contains(t, diags[0].Message, "source")
		})

		t.Run("orphan-tag-key fires when there are no commands with decides_on", func(t *testing.T) {
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
											Column:   3,
										},
										Tags: []ast.TagEntry{
											{Key: "priority", FieldRef: "OrderId"},
										},
									},
								},
								Commands: []*ast.Command{
									{Name: "PlaceOrder"},
								},
							},
						},
					},
				},
			}

			diags := linter.Lint(model)

			require.Len(t, diags, 1)
			require.Equal(t, "dcb/orphan-tag-key", diags[0].RuleName)
			require.Contains(t, diags[0].Message, "priority")
		})
	})

	t.Run("automation/missing-todo-list", func(t *testing.T) {
		t.Run("an automation activated by an event and reading no view is reported at its name", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Reservations",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Reservation",
								Slices: []*ast.Slice{
									{
										Name: "Auto Confirm Reservation",
										Automations: []*ast.Automation{
											{
												Name: "AutoConfirm",
												NamePos: ast.Position{
													Filename: "hotel.emod",
													Line:     42,
													Column:   18,
												},
												OnEvent: "ReservationMade",
												Command: "ConfirmReservation",
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

			require.Equal(t, []*diagnostic.Entry{
				{
					Filename: "hotel.emod",
					Line:     42,
					Column:   18,
					Severity: diagnostic.Warning,
					RuleName: "automation/missing-todo-list",
					Message:  `automation "AutoConfirm" reads no view, so nothing in the model shows what work is outstanding; project a view of pending work and read it`,
				},
			}, diags)
		})

		t.Run("the message names what the automation's activation clause leaves unrepresented", func(t *testing.T) {
			const eventActivated = `automation "SweepOverdueLoans" reads no view, so nothing in the model shows what work is outstanding; project a view of pending work and read it`
			const scheduled = `automation "SweepOverdueLoans" reads no view, so the model does not state what the processor acts on; project a view of pending work and read it`

			tests := []struct {
				name     string
				onEvent  string
				schedule string
				expected string
			}{
				{
					name:     "a schedule alone says the model does not state what the processor acts on",
					schedule: "15m",
					expected: scheduled,
				},
				{
					name:     "neither clause reads as an event-activated automation",
					expected: eventActivated,
				},
				{
					name:     "both clauses together read as a scheduled automation",
					onEvent:  "MemberReminded",
					schedule: "15m",
					expected: scheduled,
				},
			}

			for _, tc := range tests {
				t.Run(tc.name, func(t *testing.T) {
					model := &ast.Model{
						Contexts: []*ast.Context{
							{
								Name: "Lending",
								Aggregates: []*ast.Aggregate{
									{
										Name: "Loan",
										Slices: []*ast.Slice{
											{
												Name: "Chase Overdue Copy",
												Automations: []*ast.Automation{
													{
														Name: "SweepOverdueLoans",
														NamePos: ast.Position{
															Filename: "lending.emod",
															Line:     78,
															Column:   18,
														},
														OnEvent:  tc.onEvent,
														Schedule: tc.schedule,
														Command:  "RecallCopy",
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

					require.Equal(t, []*diagnostic.Entry{
						{
							Filename: "lending.emod",
							Line:     78,
							Column:   18,
							Severity: diagnostic.Warning,
							RuleName: "automation/missing-todo-list",
							Message:  tc.expected,
						},
					}, diags)
				})
			}
		})

		t.Run("an automation naming a view is silent, and so are a trigger and a translation naming none", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Lending",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Loan",
								Slices: []*ast.Slice{
									{
										Name: "Chase Overdue Copy",
										Trigger: &ast.Trigger{
											Name: "Overdue Report",
											NamePos: ast.Position{
												Filename: "lending.emod",
												Line:     10,
												Column:   7,
											},
											Actor: "Librarian",
										},
										Views: []*ast.View{
											{
												Name: "PendingRemindersView",
												NamePos: ast.Position{
													Filename: "lending.emod",
													Line:     14,
													Column:   7,
												},
												Subscribes: []string{"CopyBorrowed"},
											},
										},
										Automations: []*ast.Automation{
											{
												Name: "RecallOverdueCopy",
												NamePos: ast.Position{
													Filename: "lending.emod",
													Line:     22,
													Column:   18,
												},
												OnEvent: "CopyBorrowed",
												Reads:   "PendingRemindersView",
												Command: "RecallCopy",
											},
											{
												Name: "RemindOnDueDate",
												NamePos: ast.Position{
													Filename: "lending.emod",
													Line:     28,
													Column:   18,
												},
												OnEvent: "CopyBorrowed",
												Command: "RemindMember",
											},
										},
										Translations: []*ast.Translation{
											{
												Name: "ImportLoan",
												NamePos: ast.Position{
													Filename: "lending.emod",
													Line:     34,
													Column:   19,
												},
												ExternalSystem: "Legacy Catalogue",
												Command:        "BorrowCopy",
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

			require.Equal(t, []*diagnostic.Entry{
				{
					Filename: "lending.emod",
					Line:     28,
					Column:   18,
					Severity: diagnostic.Warning,
					RuleName: "automation/missing-todo-list",
					Message:  `automation "RemindOnDueDate" reads no view, so nothing in the model shows what work is outstanding; project a view of pending work and read it`,
				},
			}, diags)
		})

		t.Run("an automation in an aggregate's slice and one on a context's own slice are both reported, in declaration order", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Lending",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Loan",
								Slices: []*ast.Slice{
									{
										Name: "Chase Overdue Copy",
										Automations: []*ast.Automation{
											{
												Name: "RemindOnDueDate",
												NamePos: ast.Position{
													Filename: "lending.emod",
													Line:     30,
													Column:   18,
												},
												OnEvent: "CopyBorrowed",
												Command: "RemindMember",
											},
										},
									},
								},
							},
						},
					},
					{
						Name: "Reading Room",
						Mode: "dcb",
						Slices: []*ast.Slice{
							{
								Name: "Close Reading Room",
								Automations: []*ast.Automation{
									{
										Name: "SweepIdleDesks",
										NamePos: ast.Position{
											Filename: "lending.emod",
											Line:     61,
											Column:   16,
										},
										Schedule: "45m",
										Command:  "ReleaseDesk",
									},
								},
							},
						},
					},
				},
			}

			diags := linter.Lint(model)

			require.Equal(t, []string{
				`lending.emod:30: [automation/missing-todo-list] automation "RemindOnDueDate" reads no view, so nothing in the model shows what work is outstanding; project a view of pending work and read it`,
				`lending.emod:61: [automation/missing-todo-list] automation "SweepIdleDesks" reads no view, so the model does not state what the processor acts on; project a view of pending work and read it`,
			}, reportedLines(diags))
		})
	})

	t.Run("flow/rejection-without-spec", func(t *testing.T) {
		rejectionAt := func(command, invariant string, line int) *ast.Rejection {
			return &ast.Rejection{
				CommandName:   command,
				InvariantName: invariant,
				InvariantPos:  ast.Position{Filename: "lending.emod", Line: line, Column: 41},
			}
		}
		exercising := func(command, invariant string) *ast.Spec {
			return &ast.Spec{
				Name: "refuses it",
				When: &ast.SpecElement{Name: command},
				Then: &ast.ThenRejected{InvariantName: invariant},
			}
		}
		// The models below declare no invariant. This rule reads the edge's own
		// two names and never the declarations, and declaring one that no spec
		// rejects would additionally trip spec/invariant-never-exercised, which
		// is a different rule's subject.
		aggregateModel := func(slices ...*ast.Slice) *ast.Model {
			return &ast.Model{
				Contexts: []*ast.Context{{
					Name:       "Lending",
					Aggregates: []*ast.Aggregate{{Name: "Loan", Slices: slices}},
				}},
			}
		}

		t.Run("reports a rejection edge its slice states no exercising spec for", func(t *testing.T) {
			model := aggregateModel(&ast.Slice{
				Name:       "Borrow Copy",
				Rejections: []*ast.Rejection{rejectionAt("BorrowCopy", "OneCopyPerLoan", 12)},
			})

			diags := linter.Lint(model)

			require.Len(t, diags, 1)
			require.Equal(t, "flow/rejection-without-spec", diags[0].RuleName)
			require.Equal(t, diagnostic.Info, diags[0].Severity)
			require.Equal(t, ast.Position{Filename: "lending.emod", Line: 12, Column: 41},
				ast.Position{Filename: diags[0].Filename, Line: diags[0].Line, Column: diags[0].Column})
			require.Equal(t,
				`command "BorrowCopy" can be rejected by invariant "OneCopyPerLoan", but no spec on this slice exercises that rejection`,
				diags[0].Message)
		})

		t.Run("a matching spec on the same slice silences it", func(t *testing.T) {
			model := aggregateModel(&ast.Slice{
				Name:       "Borrow Copy",
				Rejections: []*ast.Rejection{rejectionAt("BorrowCopy", "OneCopyPerLoan", 12)},
				Specs:      []*ast.Spec{exercising("BorrowCopy", "OneCopyPerLoan")},
			})

			require.Empty(t, linter.Lint(model))
		})

		t.Run("matching is on both halves", func(t *testing.T) {
			t.Run("a spec rejecting that invariant from another command does not silence it", func(t *testing.T) {
				model := aggregateModel(&ast.Slice{
					Name:       "Borrow Copy",
					Rejections: []*ast.Rejection{rejectionAt("BorrowCopy", "OneCopyPerLoan", 12)},
					Specs:      []*ast.Spec{exercising("ReturnCopy", "OneCopyPerLoan")},
				})

				diags := linter.Lint(model)

				require.Len(t, diags, 1)
				require.Equal(t, "flow/rejection-without-spec", diags[0].RuleName)
			})

			t.Run("a spec naming that command but rejecting another invariant does not silence it", func(t *testing.T) {
				model := aggregateModel(&ast.Slice{
					Name:       "Borrow Copy",
					Rejections: []*ast.Rejection{rejectionAt("BorrowCopy", "OneCopyPerLoan", 12)},
					Specs:      []*ast.Spec{exercising("BorrowCopy", "FiveCopiesPerMember")},
				})

				diags := linter.Lint(model)

				require.Len(t, diags, 1)
				require.Equal(t, "flow/rejection-without-spec", diags[0].RuleName)
			})

			t.Run("a spec whose then is not a rejection does not silence it", func(t *testing.T) {
				model := aggregateModel(&ast.Slice{
					Name:       "Borrow Copy",
					Rejections: []*ast.Rejection{rejectionAt("BorrowCopy", "OneCopyPerLoan", 12)},
					Specs: []*ast.Spec{{
						Name: "borrows a free copy",
						When: &ast.SpecElement{Name: "BorrowCopy"},
						Then: &ast.ThenEvents{Events: []*ast.SpecElement{{Name: "CopyBorrowed"}}},
					}},
				})

				diags := linter.Lint(model)

				require.Len(t, diags, 1)
				require.Equal(t, "flow/rejection-without-spec", diags[0].RuleName)
			})
		})

		t.Run("the search is slice-local", func(t *testing.T) {
			t.Run("a matching spec in a sibling slice of the same aggregate does not silence it", func(t *testing.T) {
				model := aggregateModel(
					&ast.Slice{
						Name:       "Borrow Copy",
						Rejections: []*ast.Rejection{rejectionAt("BorrowCopy", "OneCopyPerLoan", 12)},
					},
					&ast.Slice{
						Name:  "Return Copy",
						Specs: []*ast.Spec{exercising("BorrowCopy", "OneCopyPerLoan")},
					},
				)

				diags := linter.Lint(model)

				require.Len(t, diags, 1)
				require.Equal(t, "flow/rejection-without-spec", diags[0].RuleName)
			})

			t.Run("a matching spec in another context does not silence it", func(t *testing.T) {
				model := aggregateModel(&ast.Slice{
					Name:       "Borrow Copy",
					Rejections: []*ast.Rejection{rejectionAt("BorrowCopy", "OneCopyPerLoan", 12)},
				})
				model.Contexts = append(model.Contexts, &ast.Context{
					Name: "Reading Room",
					Mode: "dcb",
					Slices: []*ast.Slice{{
						Name:  "Claim Desk",
						Specs: []*ast.Spec{exercising("BorrowCopy", "OneCopyPerLoan")},
					}},
				})

				diags := linter.Lint(model)

				require.Len(t, diags, 1)
				require.Equal(t, "flow/rejection-without-spec", diags[0].RuleName)
			})
		})

		t.Run("a slice stating no rejection edge reports nothing however many specs and flows it declares", func(t *testing.T) {
			model := aggregateModel(&ast.Slice{
				Name:  "Borrow Copy",
				Flows: []*ast.Flow{{CommandName: "BorrowCopy", EventName: "CopyBorrowed"}},
				Specs: []*ast.Spec{exercising("BorrowCopy", "OneCopyPerLoan")},
			})

			require.Empty(t, linter.Lint(model))
		})

		t.Run("reaches both slice homes and reports in declaration order", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Lending",
						Aggregates: []*ast.Aggregate{{
							Name: "Loan",
							Slices: []*ast.Slice{{
								Name: "Borrow Copy",
								Rejections: []*ast.Rejection{
									rejectionAt("BorrowCopy", "OneCopyPerLoan", 12),
									rejectionAt("BorrowCopy", "FiveCopiesPerMember", 13),
								},
							}},
						}},
					},
					{
						Name: "Reading Room",
						Mode: "dcb",
						Slices: []*ast.Slice{{
							Name:       "Claim Desk",
							Rejections: []*ast.Rejection{rejectionAt("ClaimDesk", "OneReaderPerDesk", 40)},
						}},
					},
				},
			}

			require.Equal(t, []string{
				`lending.emod:12: [flow/rejection-without-spec] command "BorrowCopy" can be rejected by invariant "OneCopyPerLoan", but no spec on this slice exercises that rejection`,
				`lending.emod:13: [flow/rejection-without-spec] command "BorrowCopy" can be rejected by invariant "FiveCopiesPerMember", but no spec on this slice exercises that rejection`,
				`lending.emod:40: [flow/rejection-without-spec] command "ClaimDesk" can be rejected by invariant "OneReaderPerDesk", but no spec on this slice exercises that rejection`,
			}, reportedLines(linter.Lint(model)))
		})

		t.Run("the shared rejection fixture exercises every edge it states", func(t *testing.T) {
			require.Empty(t, linter.Lint(test.RejectionLibraryLendingModel(t)))
		})
	})

	t.Run("spec/command-without-spec", func(t *testing.T) {
		t.Run("reports a command no spec exercises at info severity", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Lending",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Loan",
								Slices: []*ast.Slice{
									{
										Name: "Borrow Copy",
										Commands: []*ast.Command{
											{
												Name: "BorrowCopy",
												NamePos: ast.Position{
													Filename: "lending.emod",
													Line:     5,
													Column:   7,
												},
											},
										},
										Specs: []*ast.Spec{
											{
												Name: "borrows a copy no one holds",
												When: &ast.SpecElement{
													Name: "BorrowCopy",
												},
											},
											{
												Name: "refuses a copy already on loan",
												When: &ast.SpecElement{Name: "BorrowCopy"},
												Then: &ast.ThenRejected{InvariantName: "OneCopyPerLoan"},
											},
										},
									},
									{
										Name: "Return Copy",
										Commands: []*ast.Command{
											{
												Name: "ReturnCopy",
												NamePos: ast.Position{
													Filename: "lending.emod",
													Line:     15,
													Column:   7,
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
			require.Equal(t, "spec/command-without-spec", diags[0].RuleName)
			require.Equal(t, diagnostic.Info, diags[0].Severity)
			require.Equal(t, "lending.emod", diags[0].Filename)
			require.Equal(t, 15, diags[0].Line)
			require.Equal(t, `command "ReturnCopy" is not exercised by any spec`, diags[0].Message)
		})

		t.Run("reports nothing when no model-wide spec exists", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Lending",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Loan",
								Slices: []*ast.Slice{
									{
										Name: "Borrow Copy",
										Commands: []*ast.Command{
											{
												Name: "BorrowCopy",
												NamePos: ast.Position{
													Filename: "lending.emod",
													Line:     5,
													Column:   7,
												},
											},
										},
									},
									{
										Name: "Return Copy",
										Commands: []*ast.Command{
											{
												Name: "ReturnCopy",
												NamePos: ast.Position{
													Filename: "lending.emod",
													Line:     15,
													Column:   7,
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

		t.Run("reports nothing for a command exercised by any spec", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Lending",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Loan",
								Slices: []*ast.Slice{
									{
										Name: "Borrow Copy",
										Commands: []*ast.Command{
											{
												Name: "BorrowCopy",
												NamePos: ast.Position{
													Filename: "lending.emod",
													Line:     5,
													Column:   7,
												},
											},
										},
										Specs: []*ast.Spec{
											{
												Name: "borrows a copy no one holds",
												When: &ast.SpecElement{
													Name: "BorrowCopy",
												},
											},
											{
												Name: "refuses a copy already on loan",
												When: &ast.SpecElement{Name: "BorrowCopy"},
												Then: &ast.ThenRejected{InvariantName: "OneCopyPerLoan"},
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

		t.Run("coverage reaches across slices", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Lending",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Loan",
								Slices: []*ast.Slice{
									{
										Name: "Borrow Copy",
										Commands: []*ast.Command{
											{
												Name: "BorrowCopy",
												NamePos: ast.Position{
													Filename: "lending.emod",
													Line:     5,
													Column:   7,
												},
											},
										},
									},
									{
										Name: "Return Copy",
										Commands: []*ast.Command{
											{
												Name: "ReturnCopy",
												NamePos: ast.Position{
													Filename: "lending.emod",
													Line:     15,
													Column:   7,
												},
											},
										},
										Specs: []*ast.Spec{
											{
												Name: "returns a copy the member holds",
												When: &ast.SpecElement{
													Name: "ReturnCopy",
												},
											},
											{
												Name: "refuses to return an unborrowed copy",
												When: &ast.SpecElement{Name: "ReturnCopy"},
												Then: &ast.ThenRejected{InvariantName: "OneCopyPerLoan"},
											},
										},
									},
									{
										Name: "Review Member Loans",
										Specs: []*ast.Spec{
											{
												Name: "covers a command in an earlier slice",
												When: &ast.SpecElement{
													Name: "BorrowCopy",
												},
												Then: &ast.ThenRejected{InvariantName: "OneCopyPerLoan"},
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

		t.Run("a spec whose when is absent exercises no command", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Lending",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Loan",
								Slices: []*ast.Slice{
									{
										Name: "Borrow Copy",
										Commands: []*ast.Command{
											{
												Name: "BorrowCopy",
												NamePos: ast.Position{
													Filename: "lending.emod",
													Line:     5,
													Column:   7,
												},
											},
										},
										Specs: []*ast.Spec{
											{
												Name: "has no when clause",
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
			require.Equal(t, "spec/command-without-spec", diags[0].RuleName)
			require.Equal(t, `command "BorrowCopy" is not exercised by any spec`, diags[0].Message)
		})

		t.Run("reports uncovered commands from both slice homes in declaration order", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Lending",
						Mode: "mixed",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Loan",
								Slices: []*ast.Slice{
									{
										Name: "Borrow Copy",
										Commands: []*ast.Command{
											{
												Name: "BorrowCopy",
												NamePos: ast.Position{
													Filename: "lending.emod",
													Line:     5,
													Column:   7,
												},
											},
										},
									},
								},
							},
						},
						Slices: []*ast.Slice{
							{
								Name: "Release Desk",
								Commands: []*ast.Command{
									{
										Name: "ReleaseDesk",
										NamePos: ast.Position{
											Filename: "lending.emod",
											Line:     20,
											Column:   7,
										},
									},
								},
								Specs: []*ast.Spec{
									{
										Name: "has a spec but for a different command",
										When: &ast.SpecElement{
											Name: "ClaimDesk",
										},
									},
								},
							},
						},
					},
				},
			}

			diags := linter.Lint(model)

			require.Equal(t, []string{
				`lending.emod:5: [spec/command-without-spec] command "BorrowCopy" is not exercised by any spec`,
				`lending.emod:20: [spec/command-without-spec] command "ReleaseDesk" is not exercised by any spec`,
			}, reportedLines(diags))
		})
	})

	t.Run("spec/no-rejection-path", func(t *testing.T) {
		t.Run("reports a command exercised by specs but with no rejection", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Lending",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Loan",
								Slices: []*ast.Slice{
									{
										Name: "Borrow Copy",
										Commands: []*ast.Command{
											{
												Name: "BorrowCopy",
												NamePos: ast.Position{
													Filename: "lending.emod",
													Line:     5,
													Column:   7,
												},
											},
										},
										Specs: []*ast.Spec{
											{
												Name: "borrows a copy no one holds",
												When: &ast.SpecElement{Name: "BorrowCopy"},
												Then: &ast.ThenEvents{},
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
			require.Equal(t, "spec/no-rejection-path", diags[0].RuleName)
			require.Equal(t, diagnostic.Info, diags[0].Severity)
			require.Equal(t, `command "BorrowCopy" is exercised by specs but none states a rejection`, diags[0].Message)
		})

		t.Run("reports nothing when a command has a rejection spec", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Lending",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Loan",
								Slices: []*ast.Slice{
									{
										Name: "Borrow Copy",
										Commands: []*ast.Command{
											{
												Name: "BorrowCopy",
												NamePos: ast.Position{
													Filename: "lending.emod",
													Line:     5,
													Column:   7,
												},
											},
										},
										Specs: []*ast.Spec{
											{
												Name: "borrows a copy no one holds",
												When: &ast.SpecElement{Name: "BorrowCopy"},
												Then: &ast.ThenEvents{},
											},
											{
												Name:  "refuses a copy already on loan",
												Given: []*ast.SpecElement{{Name: "CopyBorrowed"}},
												When:  &ast.SpecElement{Name: "BorrowCopy"},
												Then:  &ast.ThenRejected{InvariantName: "OneCopyPerLoan"},
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

		t.Run("does not fire on a command no spec exercises", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Lending",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Loan",
								Slices: []*ast.Slice{
									{
										Name: "Borrow Copy",
										Commands: []*ast.Command{
											{
												Name: "BorrowCopy",
												NamePos: ast.Position{
													Filename: "lending.emod",
													Line:     5,
													Column:   7,
												},
											},
										},
										Specs: []*ast.Spec{
											{
												Name: "borrows a copy no one holds",
												When: &ast.SpecElement{Name: "BorrowCopy"},
												Then: &ast.ThenEvents{},
											},
											{
												Name: "refuses a copy already on loan",
												When: &ast.SpecElement{Name: "BorrowCopy"},
												Then: &ast.ThenRejected{InvariantName: "OneCopyPerLoan"},
											},
										},
									},
									{
										Name: "Return Copy",
										Commands: []*ast.Command{
											{
												Name: "ReturnCopy",
												NamePos: ast.Position{
													Filename: "lending.emod",
													Line:     15,
													Column:   7,
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
			require.Equal(t, "spec/command-without-spec", diags[0].RuleName)
			require.Contains(t, diags[0].Message, "ReturnCopy")
		})

		t.Run("rejection counts from a spec in a different slice", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Lending",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Loan",
								Slices: []*ast.Slice{
									{
										Name: "Borrow Copy",
										Commands: []*ast.Command{
											{
												Name: "BorrowCopy",
												NamePos: ast.Position{
													Filename: "lending.emod",
													Line:     5,
													Column:   7,
												},
											},
										},
									},
									{
										Name: "Return Copy",
										Specs: []*ast.Spec{
											{
												Name: "returns a copy no loan holds",
												When: &ast.SpecElement{Name: "BorrowCopy"},
												Then: &ast.ThenRejected{InvariantName: "OneCopyPerLoan"},
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

		t.Run("reports both commands in declaration order when neither has a rejection spec", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Lending",
						Mode: "mixed",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Loan",
								Slices: []*ast.Slice{
									{
										Name: "Borrow Copy",
										Commands: []*ast.Command{
											{
												Name: "BorrowCopy",
												NamePos: ast.Position{
													Filename: "lending.emod",
													Line:     5,
													Column:   7,
												},
											},
										},
										Specs: []*ast.Spec{
											{
												Name: "borrows a copy",
												When: &ast.SpecElement{Name: "BorrowCopy"},
												Then: &ast.ThenEvents{},
											},
										},
									},
								},
							},
						},
						Slices: []*ast.Slice{
							{
								Name: "Return Copy",
								Commands: []*ast.Command{
									{
										Name: "ReturnCopy",
										NamePos: ast.Position{
											Filename: "lending.emod",
											Line:     20,
											Column:   7,
										},
									},
								},
								Specs: []*ast.Spec{
									{
										Name: "returns a copy",
										When: &ast.SpecElement{Name: "ReturnCopy"},
										Then: &ast.ThenEvents{},
									},
								},
							},
						},
					},
				},
			}

			diags := linter.Lint(model)

			require.Equal(t, []string{
				`lending.emod:5: [spec/no-rejection-path] command "BorrowCopy" is exercised by specs but none states a rejection`,
				`lending.emod:20: [spec/no-rejection-path] command "ReturnCopy" is exercised by specs but none states a rejection`,
			}, reportedLines(diags))
		})
	})

	t.Run("spec/invariant-never-exercised", func(t *testing.T) {
		t.Run("reports an invariant no rejection references in an aggregate scope", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Lending",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Loan",
								Invariants: []*ast.Invariant{
									{
										Name: "OneCopyPerLoan",
										NamePos: ast.Position{
											Filename: "lending.emod",
											Line:     5,
											Column:   18,
										},
									},
								},
								Slices: []*ast.Slice{
									{
										Name: "Borrow Copy",
									},
								},
							},
						},
					},
				},
			}

			diags := linter.Lint(model)

			require.Len(t, diags, 1)
			require.Equal(t, "spec/invariant-never-exercised", diags[0].RuleName)
			require.Equal(t, diagnostic.Warning, diags[0].Severity)
			require.Equal(t, `invariant "OneCopyPerLoan" in aggregate "Loan" is not referenced by any rejection`, diags[0].Message)
		})

		t.Run("reports an invariant no rejection references in a context scope", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Reading Room",
						Mode: "dcb",
						Invariants: []*ast.Invariant{
							{
								Name: "OneReaderPerDesk",
								NamePos: ast.Position{
									Filename: "reading.emod",
									Line:     5,
									Column:   18,
								},
							},
						},
						Slices: []*ast.Slice{
							{
								Name: "Claim Desk",
							},
						},
					},
				},
			}

			diags := linter.Lint(model)

			require.Len(t, diags, 1)
			require.Equal(t, diagnostic.Warning, diags[0].Severity)
			require.Equal(t, `invariant "OneReaderPerDesk" in context "Reading Room" is not referenced by any rejection`, diags[0].Message)
		})

		t.Run("scope is not inherited: a context rejection does not exercise an aggregate invariant", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Reading Room",
						Mode: "mixed",
						Slices: []*ast.Slice{
							{
								Name: "Scan Badge",
								Specs: []*ast.Spec{
									{
										Name: "rejects an unknown badge",
										When: &ast.SpecElement{Name: "ScanBadge"},
										Then: &ast.ThenRejected{InvariantName: "OneCopyPerLoan"},
									},
								},
							},
						},
						Aggregates: []*ast.Aggregate{
							{
								Name: "Loan",
								Invariants: []*ast.Invariant{
									{
										Name: "OneCopyPerLoan",
										NamePos: ast.Position{
											Filename: "reading.emod",
											Line:     10,
											Column:   18,
										},
									},
								},
								Slices: []*ast.Slice{
									{
										Name: "Borrow Copy",
									},
								},
							},
						},
					},
				},
			}

			diags := linter.Lint(model)

			require.Len(t, diags, 1)
			require.Equal(t, `invariant "OneCopyPerLoan" in aggregate "Loan" is not referenced by any rejection`, diags[0].Message)
		})

		t.Run("scope is not inherited: an aggregate rejection does not exercise a context invariant", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Reading Room",
						Mode: "mixed",
						Invariants: []*ast.Invariant{
							{
								Name: "OneReaderPerDesk",
								NamePos: ast.Position{
									Filename: "reading.emod",
									Line:     5,
									Column:   18,
								},
							},
						},
						Slices: []*ast.Slice{
							{
								Name: "Scan Badge",
							},
						},
						Aggregates: []*ast.Aggregate{
							{
								Name: "Loan",
								Slices: []*ast.Slice{
									{
										Name: "Borrow Copy",
										Specs: []*ast.Spec{
											{
												Name: "rejects a duplicate loan",
												When: &ast.SpecElement{Name: "BorrowCopy"},
												Then: &ast.ThenRejected{InvariantName: "OneReaderPerDesk"},
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
			require.Equal(t, "spec/invariant-never-exercised", diags[0].RuleName)
			require.Equal(t, `invariant "OneReaderPerDesk" in context "Reading Room" is not referenced by any rejection`, diags[0].Message)
		})

		t.Run("an unresolved then rejected does not count as a reference", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Lending",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Loan",
								Invariants: []*ast.Invariant{
									{
										Name: "OneCopyPerLoan",
										NamePos: ast.Position{
											Filename: "lending.emod",
											Line:     5,
											Column:   18,
										},
									},
								},
								Slices: []*ast.Slice{
									{
										Name: "Borrow Copy",
										Specs: []*ast.Spec{
											{
												Name: "rejects with a typoed name",
												When: &ast.SpecElement{Name: "BorrowCopy"},
												Then: &ast.ThenRejected{InvariantName: "OneCpyPerLoan"},
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
			require.Equal(t, "spec/invariant-never-exercised", diags[0].RuleName)
			require.Equal(t, `invariant "OneCopyPerLoan" in aggregate "Loan" is not referenced by any rejection`, diags[0].Message)
		})

		t.Run("declaring no invariant produces nothing", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Lending",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Loan",
								Slices: []*ast.Slice{
									{
										Name: "Borrow Copy",
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

		t.Run("an invariant exercised by a rejection produces nothing", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Lending",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Loan",
								Invariants: []*ast.Invariant{
									{
										Name: "OneCopyPerLoan",
										NamePos: ast.Position{
											Filename: "lending.emod",
											Line:     5,
											Column:   18,
										},
									},
								},
								Slices: []*ast.Slice{
									{
										Name: "Borrow Copy",
										Specs: []*ast.Spec{
											{
												Name: "rejects a copy already on loan",
												When: &ast.SpecElement{Name: "BorrowCopy"},
												Then: &ast.ThenRejected{InvariantName: "OneCopyPerLoan"},
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

		t.Run("reports findings in declaration order across context and aggregate scopes", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Lending",
						Mode: "mixed",
						Invariants: []*ast.Invariant{
							{
								Name: "DeskFreeAtClosing",
								NamePos: ast.Position{
									Filename: "lending.emod",
									Line:     26,
									Column:   18,
								},
							},
						},
						Slices: []*ast.Slice{
							{
								Name: "Scan Badge",
							},
						},
						Aggregates: []*ast.Aggregate{
							{
								Name: "Loan",
								Invariants: []*ast.Invariant{
									{
										Name: "OneCopyPerLoan",
										NamePos: ast.Position{
											Filename: "lending.emod",
											Line:     5,
											Column:   18,
										},
									},
								},
								Slices: []*ast.Slice{
									{
										Name: "Borrow Copy",
									},
								},
							},
						},
					},
				},
			}

			diags := linter.Lint(model)

			require.Equal(t, []string{
				`lending.emod:5: [spec/invariant-never-exercised] invariant "OneCopyPerLoan" in aggregate "Loan" is not referenced by any rejection`,
				`lending.emod:26: [spec/invariant-never-exercised] invariant "DeskFreeAtClosing" in context "Lending" is not referenced by any rejection`,
			}, reportedLines(diags))
		})

		t.Run("a rejection edge references an invariant as much as a spec's rejection does", func(t *testing.T) {
			invariantAt := func(name string, line int) *ast.Invariant {
				return &ast.Invariant{
					Name:    name,
					NamePos: ast.Position{Filename: "lending.emod", Line: line, Column: 18},
				}
			}
			edge := func(command, invariant string) *ast.Rejection {
				return &ast.Rejection{
					CommandName:   command,
					InvariantName: invariant,
					InvariantPos:  ast.Position{Filename: "lending.emod", Line: 30, Column: 41},
				}
			}
			// No slice below pairs its edge with an exercising spec. A spec that
			// silenced flow/rejection-without-spec would silence this rule too
			// through the reference it is itself, leaving the edge doing nothing
			// and the leaf passing whether or not the collector reads edges at
			// all. So the edge is the only reference, and the noise the other
			// rule makes is filtered out by name.
			exercised := func(diags []*diagnostic.Entry) []string {
				return linesReportedBy(diags, "spec/invariant-never-exercised")
			}

			t.Run("an aggregate's invariant named only by an edge in its own slices is not reported", func(t *testing.T) {
				model := &ast.Model{
					Contexts: []*ast.Context{{
						Name: "Lending",
						Aggregates: []*ast.Aggregate{{
							Name:       "Loan",
							Invariants: []*ast.Invariant{invariantAt("OneCopyPerLoan", 5)},
							Slices: []*ast.Slice{{
								Name:       "Borrow Copy",
								Rejections: []*ast.Rejection{edge("BorrowCopy", "OneCopyPerLoan")},
							}},
						}},
					}},
				}

				require.Empty(t, exercised(linter.Lint(model)),
					"the model states no spec at all, which is the case the seam exists for and the one a spec gate would have broken")
			})

			t.Run("a dcb context's own invariant named only by an edge in its own slices is not reported", func(t *testing.T) {
				model := &ast.Model{
					Contexts: []*ast.Context{{
						Name:       "Reading Room",
						Mode:       "dcb",
						Invariants: []*ast.Invariant{invariantAt("OneReaderPerDesk", 5)},
						Slices: []*ast.Slice{{
							Name:       "Claim Desk",
							Rejections: []*ast.Rejection{edge("ClaimDesk", "OneReaderPerDesk")},
						}},
					}},
				}

				require.Empty(t, exercised(linter.Lint(model)))
			})

			t.Run("an edge in a sibling aggregate's slice does not exercise it", func(t *testing.T) {
				model := &ast.Model{
					Contexts: []*ast.Context{{
						Name: "Lending",
						Aggregates: []*ast.Aggregate{
							{
								Name:       "Loan",
								Invariants: []*ast.Invariant{invariantAt("OneCopyPerLoan", 5)},
								Slices:     []*ast.Slice{{Name: "Borrow Copy"}},
							},
							{
								Name: "Reservation",
								Slices: []*ast.Slice{{
									Name:       "Hold Copy",
									Rejections: []*ast.Rejection{edge("HoldCopy", "OneCopyPerLoan")},
								}},
							},
						},
					}},
				}

				require.Equal(t, []string{
					`lending.emod:5: [spec/invariant-never-exercised] invariant "OneCopyPerLoan" in aggregate "Loan" is not referenced by any rejection`,
				}, exercised(linter.Lint(model)))
			})

			t.Run("an edge in an aggregate's slice does not exercise the enclosing context's invariant", func(t *testing.T) {
				model := &ast.Model{
					Contexts: []*ast.Context{{
						Name:       "Lending",
						Mode:       "mixed",
						Invariants: []*ast.Invariant{invariantAt("DeskFreeAtClosing", 5)},
						Aggregates: []*ast.Aggregate{{
							Name: "Loan",
							Slices: []*ast.Slice{{
								Name:       "Borrow Copy",
								Rejections: []*ast.Rejection{edge("BorrowCopy", "DeskFreeAtClosing")},
							}},
						}},
					}},
				}

				require.Equal(t, []string{
					`lending.emod:5: [spec/invariant-never-exercised] invariant "DeskFreeAtClosing" in context "Lending" is not referenced by any rejection`,
				}, exercised(linter.Lint(model)))
			})

			t.Run("an edge naming an invariant no scope declares leaves the one it was written for reported", func(t *testing.T) {
				model := &ast.Model{
					Contexts: []*ast.Context{{
						Name: "Lending",
						Aggregates: []*ast.Aggregate{{
							Name:       "Loan",
							Invariants: []*ast.Invariant{invariantAt("OneCopyPerLoan", 5)},
							Slices: []*ast.Slice{{
								Name:       "Borrow Copy",
								Rejections: []*ast.Rejection{edge("BorrowCopy", "OneCopyPerLon")},
							}},
						}},
					}},
				}

				require.Equal(t, []string{
					`lending.emod:5: [spec/invariant-never-exercised] invariant "OneCopyPerLoan" in aggregate "Loan" is not referenced by any rejection`,
				}, exercised(linter.Lint(model)), "a typo must not silence the rule for the invariant it was meant to name")
			})

			t.Run("an invariant an edge and a spec both reference is silent while its unreferenced sibling is still reported once", func(t *testing.T) {
				model := &ast.Model{
					Contexts: []*ast.Context{{
						Name: "Lending",
						Aggregates: []*ast.Aggregate{{
							Name:       "Loan",
							Invariants: []*ast.Invariant{invariantAt("OneCopyPerLoan", 5), invariantAt("FiveCopiesPerMember", 6)},
							Slices: []*ast.Slice{{
								Name:       "Borrow Copy",
								Rejections: []*ast.Rejection{edge("BorrowCopy", "OneCopyPerLoan")},
								Specs: []*ast.Spec{{
									Name: "refuses a copy already on loan",
									When: &ast.SpecElement{Name: "BorrowCopy"},
									Then: &ast.ThenRejected{InvariantName: "OneCopyPerLoan"},
								}},
							}},
						}},
					}},
				}

				require.Equal(t, []string{
					`lending.emod:6: [spec/invariant-never-exercised] invariant "FiveCopiesPerMember" in aggregate "Loan" is not referenced by any rejection`,
				}, exercised(linter.Lint(model)))
			})

		})
	})

	t.Run("spec/given-outside-boundary", func(t *testing.T) {
		t.Run("aggregate arm: reports a given event from another aggregate", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Lending",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Loan",
								Slices: []*ast.Slice{
									{
										Name: "Borrow Copy",
										Events: []*ast.Event{
											{
												Name: "CopyBorrowed",
												NamePos: ast.Position{
													Filename: "lending.emod",
													Line:     12,
													Column:   9,
												},
											},
										},
									},
								},
							},
							{
								Name: "Reader",
								Slices: []*ast.Slice{
									{
										Name: "Claim Desk",
										Specs: []*ast.Spec{
											{
												Name: "claims a desk",
												When: &ast.SpecElement{Name: "ClaimDesk"},
												Given: []*ast.SpecElement{
													{
														Name: "CopyBorrowed",
														NamePos: ast.Position{
															Filename: "lending.emod",
															Line:     22,
															Column:   15,
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
				},
			}

			diags := linter.Lint(model)

			require.Len(t, diags, 1)
			require.Equal(t, "spec/given-outside-boundary", diags[0].RuleName)
			require.Equal(t, diagnostic.Warning, diags[0].Severity)
			require.Equal(t, `lending.emod:22: [spec/given-outside-boundary] given event "CopyBorrowed" names an event declared by aggregate "Loan" instead of aggregate "Reader"`, diags[0].String())
		})

		t.Run("aggregate arm: reports a given event declared in another context", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Lending",
						Mode: "mixed",
						Slices: []*ast.Slice{
							{
								Name: "Register Member",
								Commands: []*ast.Command{
									{
										Name: "RegisterMember",
										NamePos: ast.Position{
											Filename: "lending.emod",
											Line:     5,
											Column:   9,
										},
										DecidesOn: &ast.DecidesOnClause{
											Events: []string{"MemberRegistered"},
											Predicate: &ast.LogicalExpr{
												Left:  &ast.TagPredicate{Field: "role"},
												Right: &ast.TagPredicate{Field: "region"},
											},
										},
									},
								},
								Events: []*ast.Event{
									{
										Name: "MemberRegistered",
										NamePos: ast.Position{
											Filename: "lending.emod",
											Line:     12,
											Column:   9,
										},
										Tags: []ast.TagEntry{{Key: "role", FieldRef: "member"}, {Key: "region", FieldRef: "country"}},
									},
								},
								Specs: []*ast.Spec{
									{
										Name: "registers a new member",
										When: &ast.SpecElement{Name: "RegisterMember"},
										Then: &ast.ThenEvents{},
									},
									{
										Name: "refuses a duplicate",
										When: &ast.SpecElement{Name: "RegisterMember"},
										Then: &ast.ThenRejected{InvariantName: "OneMemberPerEmail"},
									},
								},
							},
						},
					},
					{
						Name: "Reading Room",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Desk",
								Slices: []*ast.Slice{
									{
										Name: "Claim Desk",
										Specs: []*ast.Spec{
											{
												Name: "claims a desk",
												When: &ast.SpecElement{Name: "ClaimDesk"},
												Given: []*ast.SpecElement{
													{
														Name: "MemberRegistered",
														NamePos: ast.Position{
															Filename: "lending.emod",
															Line:     18,
															Column:   15,
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
				},
			}

			diags := linter.Lint(model)

			require.Len(t, diags, 1)
			require.Equal(t, `given event "MemberRegistered" names an event declared by context "Lending" instead of aggregate "Desk"`, diags[0].Message)
		})

		t.Run("aggregate arm: reports nothing when the event is declared by a sibling slice of the same aggregate", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Lending",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Loan",
								Slices: []*ast.Slice{
									{
										Name: "Borrow Copy",
										Events: []*ast.Event{
											{
												Name: "CopyBorrowed",
												NamePos: ast.Position{
													Filename: "lending.emod",
													Line:     8,
													Column:   9,
												},
											},
										},
									},
									{
										Name: "Return Copy",
										Specs: []*ast.Spec{
											{
												Name: "returns a copy",
												When: &ast.SpecElement{Name: "ReturnCopy"},
												Given: []*ast.SpecElement{
													{
														Name: "CopyBorrowed",
														NamePos: ast.Position{
															Filename: "lending.emod",
															Line:     16,
															Column:   15,
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
				},
			}

			diags := linter.Lint(model)

			require.Empty(t, diags)
		})

		t.Run("aggregate arm: an event inside a translation counts as declared by the slice", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Lending",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Loan",
								Slices: []*ast.Slice{
									{
										Name: "Borrow Copy",
										Translations: []*ast.Translation{
											{
												Event: &ast.Event{
													Name: "CopyBorrowed",
													NamePos: ast.Position{
														Filename: "lending.emod",
														Line:     8,
														Column:   9,
													},
												},
											},
										},
									},
									{
										Name: "Return Copy",
										Specs: []*ast.Spec{
											{
												Name: "returns a copy",
												When: &ast.SpecElement{Name: "ReturnCopy"},
												Given: []*ast.SpecElement{
													{
														Name: "CopyBorrowed",
														NamePos: ast.Position{
															Filename: "lending.emod",
															Line:     16,
															Column:   15,
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
				},
			}

			diags := linter.Lint(model)

			require.Empty(t, diags)
		})

		t.Run("aggregate arm: an undeclared given event produces no diagnostic", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Lending",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Loan",
								Slices: []*ast.Slice{
									{
										Name: "Borrow Copy",
										Specs: []*ast.Spec{
											{
												Name: "borrows a copy",
												When: &ast.SpecElement{Name: "BorrowCopy"},
												Given: []*ast.SpecElement{
													{
														Name: "UnknownEvent",
														NamePos: ast.Position{
															Filename: "lending.emod",
															Line:     8,
															Column:   15,
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
				},
			}

			diags := linter.Lint(model)

			require.Empty(t, diags)
		})

		t.Run("aggregate arm: empty or missing given produces no diagnostic", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Lending",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Loan",
								Slices: []*ast.Slice{
									{
										Name: "Borrow Copy",
										Specs: []*ast.Spec{
											{
												Name:  "borrows a copy",
												When:  &ast.SpecElement{Name: "BorrowCopy"},
												Given: []*ast.SpecElement{},
											},
											{
												Name: "returns a copy",
												When: &ast.SpecElement{Name: "ReturnCopy"},
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

		t.Run("aggregate arm: reports every offending name in declaration order", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Lending",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Loan",
								Slices: []*ast.Slice{
									{
										Name: "Borrow Copy",
										Events: []*ast.Event{
											{
												Name: "CopyBorrowed",
												NamePos: ast.Position{
													Filename: "lending.emod",
													Line:     8,
													Column:   9,
												},
											},
											{
												Name: "CopyReturned",
												NamePos: ast.Position{
													Filename: "lending.emod",
													Line:     10,
													Column:   9,
												},
											},
										},
									},
								},
							},
							{
								Name: "Reader",
								Slices: []*ast.Slice{
									{
										Name: "Claim Desk",
										Specs: []*ast.Spec{
											{
												Name: "claims a desk",
												When: &ast.SpecElement{Name: "ClaimDesk"},
												Given: []*ast.SpecElement{
													{
														Name: "CopyBorrowed",
														NamePos: ast.Position{
															Filename: "lending.emod",
															Line:     22,
															Column:   15,
														},
													},
													{
														Name: "CopyReturned",
														NamePos: ast.Position{
															Filename: "lending.emod",
															Line:     23,
															Column:   15,
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
				},
			}

			diags := linter.Lint(model)

			require.Equal(t, []string{
				`lending.emod:22: [spec/given-outside-boundary] given event "CopyBorrowed" names an event declared by aggregate "Loan" instead of aggregate "Reader"`,
				`lending.emod:23: [spec/given-outside-boundary] given event "CopyReturned" names an event declared by aggregate "Loan" instead of aggregate "Reader"`,
			}, reportedLines(diags))
		})

		t.Run("aggregate arm: a DCB context-level spec produces no diagnostic from this arm", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Reading Room",
						Mode: "dcb",
						Slices: []*ast.Slice{
							{
								Name: "Claim Desk",
								Specs: []*ast.Spec{
									{
										Name: "claims a desk",
										When: &ast.SpecElement{Name: "ClaimDesk"},
										Given: []*ast.SpecElement{
											{
												Name: "DeskClaimed",
												NamePos: ast.Position{
													Filename: "reading.emod",
													Line:     5,
													Column:   15,
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

		t.Run("DCB arm: reports a given event not listed by the when command's decides_on", func(t *testing.T) {
			makeDCBModel := func() *ast.Model {
				return &ast.Model{
					Contexts: []*ast.Context{
						{
							Name: "Reading Room", Mode: "dcb",
							Invariants: []*ast.Invariant{
								{Name: "OneDeskPerReader", NamePos: ast.Position{Filename: "reading.emod", Line: 2, Column: 18}},
							},
							Slices: []*ast.Slice{{Name: "Claim Desk",
								Commands: []*ast.Command{{
									Name: "ClaimDesk", NamePos: ast.Position{Filename: "reading.emod", Line: 4, Column: 9},
									DecidesOn: &ast.DecidesOnClause{
										Events:    []string{"DeskClaimed"},
										Predicate: &ast.LogicalExpr{Left: &ast.TagPredicate{Field: "desk"}, Right: &ast.TagPredicate{Field: "region"}},
									},
								}},
								Events: []*ast.Event{
									{Name: "DeskClaimed", NamePos: ast.Position{Filename: "reading.emod", Line: 10, Column: 9}, Tags: []ast.TagEntry{{Key: "desk", FieldRef: "deskId"}, {Key: "region", FieldRef: "regionId"}}},
									{Name: "DeskReleased", NamePos: ast.Position{Filename: "reading.emod", Line: 12, Column: 9}, Tags: []ast.TagEntry{{Key: "desk", FieldRef: "deskId"}, {Key: "region", FieldRef: "regionId"}}},
								},
								Specs: []*ast.Spec{
									{Name: "claims a desk", When: &ast.SpecElement{Name: "ClaimDesk"}, Then: &ast.ThenEvents{}},
									{Name: "refuses when taken", When: &ast.SpecElement{Name: "ClaimDesk"}, Then: &ast.ThenRejected{InvariantName: "OneDeskPerReader"}},
								},
							}},
						},
					},
				}
			}

			model := makeDCBModel()
			model.Contexts[0].Slices[0].Specs = append(model.Contexts[0].Slices[0].Specs, &ast.Spec{
				Name: "claims a desk when released",
				When: &ast.SpecElement{Name: "ClaimDesk"},
				Given: []*ast.SpecElement{
					{Name: "DeskReleased", NamePos: ast.Position{Filename: "reading.emod", Line: 22, Column: 15}},
				},
				Then: &ast.ThenEvents{},
			})

			diags := linter.Lint(model)

			require.Len(t, diags, 1)
			require.Equal(t, "spec/given-outside-boundary", diags[0].RuleName)
			require.Equal(t, diagnostic.Warning, diags[0].Severity)
			require.Equal(t, `reading.emod:22: [spec/given-outside-boundary] given event "DeskReleased" names an event command "ClaimDesk"'s decides_on does not list`, diags[0].String())
		})

		t.Run("DCB arm: reports nothing when the given event is in decides_on", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Reading Room", Mode: "dcb",
						Invariants: []*ast.Invariant{
							{Name: "OneDeskPerReader", NamePos: ast.Position{Filename: "reading.emod", Line: 2, Column: 18}},
						},
						Slices: []*ast.Slice{{Name: "Claim Desk",
							Commands: []*ast.Command{{
								Name: "ClaimDesk", NamePos: ast.Position{Filename: "reading.emod", Line: 4, Column: 9},
								DecidesOn: &ast.DecidesOnClause{
									Events:    []string{"DeskClaimed", "DeskReleased"},
									Predicate: &ast.LogicalExpr{Left: &ast.TagPredicate{Field: "desk"}, Right: &ast.TagPredicate{Field: "region"}},
								},
							}},
							Events: []*ast.Event{
								{Name: "DeskClaimed", NamePos: ast.Position{Filename: "reading.emod", Line: 10, Column: 9}, Tags: []ast.TagEntry{{Key: "desk", FieldRef: "deskId"}, {Key: "region", FieldRef: "regionId"}}},
							},
							Specs: []*ast.Spec{
								{Name: "claims a desk", When: &ast.SpecElement{Name: "ClaimDesk"}, Then: &ast.ThenEvents{}},
								{Name: "refuses when taken", When: &ast.SpecElement{Name: "ClaimDesk"}, Then: &ast.ThenRejected{InvariantName: "OneDeskPerReader"}},
								{
									Name:  "claims a desk when released",
									When:  &ast.SpecElement{Name: "ClaimDesk"},
									Given: []*ast.SpecElement{{Name: "DeskClaimed", NamePos: ast.Position{Filename: "reading.emod", Line: 20, Column: 15}}},
									Then:  &ast.ThenEvents{},
								},
							},
						}},
					},
				},
			}

			diags := linter.Lint(model)
			require.Empty(t, diags)
		})

		t.Run("DCB arm: a command with no decides_on puts nothing outside the boundary", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Reading Room", Mode: "dcb",
						Invariants: []*ast.Invariant{
							{Name: "OneDeskPerReader", NamePos: ast.Position{Filename: "reading.emod", Line: 2, Column: 18}},
						},
						Slices: []*ast.Slice{{Name: "Claim Desk",
							Commands: []*ast.Command{{Name: "ClaimDesk", NamePos: ast.Position{Filename: "reading.emod", Line: 4, Column: 9}}},
							Specs: []*ast.Spec{
								{Name: "claims a desk", When: &ast.SpecElement{Name: "ClaimDesk"}, Then: &ast.ThenEvents{}},
								{Name: "refuses when taken", When: &ast.SpecElement{Name: "ClaimDesk"}, Then: &ast.ThenRejected{InvariantName: "OneDeskPerReader"}},
								{
									Name:  "claims a desk after release",
									When:  &ast.SpecElement{Name: "ClaimDesk"},
									Given: []*ast.SpecElement{{Name: "DeskClaimed", NamePos: ast.Position{Filename: "reading.emod", Line: 14, Column: 15}}},
									Then:  &ast.ThenEvents{},
								},
							},
						}},
					},
				},
			}

			diags := linter.Lint(model)
			require.Empty(t, diags)
		})

		t.Run("DCB arm: a spec whose when is absent produces no diagnostic", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Reading Room", Mode: "dcb",
						Slices: []*ast.Slice{{Name: "Claim Desk",
							Specs: []*ast.Spec{
								{Name: "claims a desk", Given: []*ast.SpecElement{{Name: "DeskReleased", NamePos: ast.Position{Filename: "reading.emod", Line: 5, Column: 15}}}},
							},
						}},
					},
				},
			}

			diags := linter.Lint(model)
			require.Empty(t, diags)
		})

		t.Run("DCB arm: an undeclared given event produces no diagnostic", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Reading Room", Mode: "dcb",
						Invariants: []*ast.Invariant{
							{Name: "OneDeskPerReader", NamePos: ast.Position{Filename: "reading.emod", Line: 2, Column: 18}},
						},
						Slices: []*ast.Slice{{Name: "Claim Desk",
							Commands: []*ast.Command{{
								Name: "ClaimDesk", NamePos: ast.Position{Filename: "reading.emod", Line: 4, Column: 9},
								DecidesOn: &ast.DecidesOnClause{
									Events:    []string{"DeskClaimed"},
									Predicate: &ast.LogicalExpr{Left: &ast.TagPredicate{Field: "desk"}, Right: &ast.TagPredicate{Field: "region"}},
								},
							}},
							Events: []*ast.Event{
								{Name: "DeskClaimed", NamePos: ast.Position{Filename: "reading.emod", Line: 10, Column: 9}, Tags: []ast.TagEntry{{Key: "desk", FieldRef: "deskId"}, {Key: "region", FieldRef: "regionId"}}},
							},
							Specs: []*ast.Spec{
								{Name: "claims a desk", When: &ast.SpecElement{Name: "ClaimDesk"}, Then: &ast.ThenEvents{}},
								{Name: "refuses when taken", When: &ast.SpecElement{Name: "ClaimDesk"}, Then: &ast.ThenRejected{InvariantName: "OneDeskPerReader"}},
								{
									Name: "unknown event", When: &ast.SpecElement{Name: "ClaimDesk"},
									Given: []*ast.SpecElement{{Name: "UnknownEvent", NamePos: ast.Position{Filename: "reading.emod", Line: 20, Column: 15}}},
									Then:  &ast.ThenEvents{},
								},
							},
						}},
					},
				},
			}

			diags := linter.Lint(model)
			require.Empty(t, diags)
		})

		t.Run("DCB arm: reports every offending name in declaration order", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Reading Room", Mode: "dcb",
						Invariants: []*ast.Invariant{
							{Name: "OneDeskPerReader", NamePos: ast.Position{Filename: "reading.emod", Line: 2, Column: 18}},
						},
						Slices: []*ast.Slice{{Name: "Claim Desk",
							Commands: []*ast.Command{{
								Name: "ClaimDesk", NamePos: ast.Position{Filename: "reading.emod", Line: 4, Column: 9},
								DecidesOn: &ast.DecidesOnClause{
									Events:    []string{"DeskClaimed"},
									Predicate: &ast.LogicalExpr{Left: &ast.TagPredicate{Field: "desk"}, Right: &ast.TagPredicate{Field: "region"}},
								},
							}},
							Events: []*ast.Event{
								{Name: "DeskClaimed", NamePos: ast.Position{Filename: "reading.emod", Line: 10, Column: 9}, Tags: []ast.TagEntry{{Key: "desk", FieldRef: "deskId"}, {Key: "region", FieldRef: "regionId"}}},
								{Name: "DeskReleased", NamePos: ast.Position{Filename: "reading.emod", Line: 12, Column: 9}, Tags: []ast.TagEntry{{Key: "desk", FieldRef: "deskId"}, {Key: "region", FieldRef: "regionId"}}},
							},
							Specs: []*ast.Spec{
								{Name: "claims a desk", When: &ast.SpecElement{Name: "ClaimDesk"}, Then: &ast.ThenEvents{}},
								{Name: "refuses when taken", When: &ast.SpecElement{Name: "ClaimDesk"}, Then: &ast.ThenRejected{InvariantName: "OneDeskPerReader"}},
								{
									Name: "claims a desk after events", When: &ast.SpecElement{Name: "ClaimDesk"},
									Given: []*ast.SpecElement{
										{Name: "DeskReleased", NamePos: ast.Position{Filename: "reading.emod", Line: 18, Column: 15}},
										{Name: "DeskClaimed", NamePos: ast.Position{Filename: "reading.emod", Line: 19, Column: 15}},
									},
									Then: &ast.ThenEvents{},
								},
							},
						}},
					},
				},
			}

			diags := linter.Lint(model)

			require.Equal(t, []string{
				`reading.emod:18: [spec/given-outside-boundary] given event "DeskReleased" names an event command "ClaimDesk"'s decides_on does not list`,
			}, reportedLines(diags))
		})

		t.Run("value arm: reports a given payload value the command's tag predicate excludes", func(t *testing.T) {
			spec := &ast.Spec{
				Name: "claims a desk another reader holds",
				Given: []*ast.SpecElement{
					specElementAt("DeskClaimed", 18,
						payloadValueAt("deskId", "D-4210", ast.StringLiteral, 18, 34)),
				},
				When: specElementAt("ClaimDesk", 19,
					payloadValueAt("deskId", "D-5817", ast.StringLiteral, 19, 30)),
				Then: &ast.ThenEvents{},
			}

			diags := linter.Lint(dcbValueModel(&ast.TagPredicate{Field: "desk", Value: "deskId"}, spec))

			require.Len(t, diags, 1)
			require.Equal(t, diagnostic.Warning, diags[0].Severity)
			require.Equal(t, "spec/given-outside-boundary", diags[0].RuleName)
			require.Equal(t, 18, diags[0].Line)
			require.Equal(t, 34, diags[0].Column,
				"the diagnostic sits on the value inside the given payload, not on the event name")
			require.Equal(t, `reading.emod:18: [spec/given-outside-boundary] given event "DeskClaimed" `+
				`states deskId "D-4210" while command "ClaimDesk"'s when payload states deskId "D-5817", `+
				`so tag "desk" excludes it from the query`, diags[0].String())
		})

		t.Run("value arm: payloads stating the tagged field alike report nothing", func(t *testing.T) {
			agreeing := disagreeingDeskSpec()
			agreeing.Given[0].Payload[0].Value = "D-5817"

			diags := linter.Lint(dcbValueModel(&ast.TagPredicate{Field: "desk", Value: "deskId"}, agreeing))

			require.Empty(t, diags)
		})

		t.Run("value arm: a given payload omitting the tagged field reports nothing", func(t *testing.T) {
			spec := disagreeingDeskSpec()
			spec.Given[0].Payload = []*ast.PayloadField{
				payloadValueAt("sessionId", "S-7712", ast.StringLiteral, 18, 34),
				payloadValueAt("quietZone", "true", ast.BooleanLiteral, 18, 56),
			}

			diags := linter.Lint(dcbValueModel(&ast.TagPredicate{Field: "desk", Value: "deskId"}, spec))

			require.Empty(t, diags)
		})

		t.Run("value arm: a when payload omitting the tagged field reports nothing", func(t *testing.T) {
			spec := disagreeingDeskSpec()
			spec.When.Payload = []*ast.PayloadField{
				payloadValueAt("memberId", "M-63204", ast.StringLiteral, 19, 30),
				payloadValueAt("preferredZone", "north gallery", ast.StringLiteral, 19, 55),
			}

			diags := linter.Lint(dcbValueModel(&ast.TagPredicate{Field: "desk", Value: "deskId"}, spec))

			require.Empty(t, diags)
		})

		t.Run("value arm: an element stating no payload at all reports nothing", func(t *testing.T) {
			t.Run("the given element", func(t *testing.T) {
				spec := disagreeingDeskSpec()
				spec.Given[0].Payload = nil

				diags := linter.Lint(dcbValueModel(&ast.TagPredicate{Field: "desk", Value: "deskId"}, spec))

				require.Empty(t, diags)
			})

			t.Run("the when element", func(t *testing.T) {
				spec := disagreeingDeskSpec()
				spec.When.Payload = nil

				diags := linter.Lint(dcbValueModel(&ast.TagPredicate{Field: "desk", Value: "deskId"}, spec))

				require.Empty(t, diags)
			})
		})

		t.Run("value arm: two literals of different kinds are not compared", func(t *testing.T) {
			pairs := []struct {
				name       string
				givenValue string
				givenKind  ast.LiteralKind
				whenValue  string
				whenKind   ast.LiteralKind
			}{
				{"a string against a number", "4210", ast.StringLiteral, "4211", ast.IntegerLiteral},
				{"a boolean against a string", "true", ast.BooleanLiteral, "yes", ast.StringLiteral},
				{"a boolean against a number", "true", ast.BooleanLiteral, "1", ast.IntegerLiteral},
			}
			for _, pair := range pairs {
				t.Run(pair.name, func(t *testing.T) {
					spec := disagreeingDeskSpec()
					spec.Given[0].Payload = []*ast.PayloadField{payloadValueAt("deskId", pair.givenValue, pair.givenKind, 18, 34)}
					spec.When.Payload = []*ast.PayloadField{payloadValueAt("deskId", pair.whenValue, pair.whenKind, 19, 30)}

					diags := linter.Lint(dcbValueModel(&ast.TagPredicate{Field: "desk", Value: "deskId"}, spec))

					require.Empty(t, diags)
				})
			}
		})

		t.Run("value arm: numbers compare as numbers, not as source text", func(t *testing.T) {
			pairs := []struct {
				name      string
				given     string
				givenKind ast.LiteralKind
				when      string
				whenKind  ast.LiteralKind
				reports   bool
			}{
				{name: "a trailing zero is not a different value", given: "12.50", givenKind: ast.DecimalLiteral, when: "12.5", whenKind: ast.DecimalLiteral},
				{name: "a leading zero is not a different value", given: "007", givenKind: ast.IntegerLiteral, when: "7", whenKind: ast.IntegerLiteral},
				{name: "an integer and a decimal of one value agree", given: "7", givenKind: ast.IntegerLiteral, when: "7.0", whenKind: ast.DecimalLiteral},
				{name: "an integer and a decimal of different value disagree", given: "7", givenKind: ast.IntegerLiteral, when: "7.5", whenKind: ast.DecimalLiteral, reports: true},
				{name: "two numbers of different value disagree", given: "12.50", givenKind: ast.DecimalLiteral, when: "12.75", whenKind: ast.DecimalLiteral, reports: true},
			}
			for _, pair := range pairs {
				t.Run(pair.name, func(t *testing.T) {
					spec := disagreeingDeskSpec()
					spec.Given[0].Payload = []*ast.PayloadField{payloadValueAt("deskId", pair.given, pair.givenKind, 18, 34)}
					spec.When.Payload = []*ast.PayloadField{payloadValueAt("deskId", pair.when, pair.whenKind, 19, 30)}

					diags := linter.Lint(dcbValueModel(&ast.TagPredicate{Field: "desk", Value: "deskId"}, spec))

					if !pair.reports {
						require.Empty(t, diags)
						return
					}
					require.Equal(t, []string{
						`reading.emod:18: [spec/given-outside-boundary] given event "DeskClaimed" states deskId ` + pair.given +
							` while command "ClaimDesk"'s when payload states deskId ` + pair.when +
							`, so tag "desk" excludes it from the query`,
					}, reportedLines(diags))
				})
			}
		})

		t.Run("value arm: booleans compare by their spelling", func(t *testing.T) {
			pairs := []struct {
				name    string
				given   string
				when    string
				reports bool
			}{
				{name: "one spelling on both sides agrees", given: "true", when: "true"},
				{name: "the two spellings disagree", given: "true", when: "false", reports: true},
			}
			for _, pair := range pairs {
				t.Run(pair.name, func(t *testing.T) {
					spec := disagreeingDeskSpec()
					spec.Given[0].Payload = []*ast.PayloadField{payloadValueAt("deskId", pair.given, ast.BooleanLiteral, 18, 34)}
					spec.When.Payload = []*ast.PayloadField{payloadValueAt("deskId", pair.when, ast.BooleanLiteral, 19, 30)}

					diags := linter.Lint(dcbValueModel(&ast.TagPredicate{Field: "desk", Value: "deskId"}, spec))

					if !pair.reports {
						require.Empty(t, diags)
						return
					}
					require.Equal(t, []string{
						`reading.emod:18: [spec/given-outside-boundary] given event "DeskClaimed" states deskId true ` +
							`while command "ClaimDesk"'s when payload states deskId false, so tag "desk" excludes it from the query`,
					}, reportedLines(diags))
				})
			}
		})

		t.Run("value arm: strings compare by their exact text", func(t *testing.T) {
			pairs := []struct {
				name  string
				given string
				when  string
			}{
				{"differing only in case", "d-5817", "D-5817"},
				{"differing by surrounding whitespace", " D-5817", "D-5817"},
			}
			for _, pair := range pairs {
				t.Run(pair.name, func(t *testing.T) {
					spec := disagreeingDeskSpec()
					spec.Given[0].Payload = []*ast.PayloadField{payloadValueAt("deskId", pair.given, ast.StringLiteral, 18, 34)}
					spec.When.Payload = []*ast.PayloadField{payloadValueAt("deskId", pair.when, ast.StringLiteral, 19, 30)}

					diags := linter.Lint(dcbValueModel(&ast.TagPredicate{Field: "desk", Value: "deskId"}, spec))

					require.Equal(t, []string{
						`reading.emod:18: [spec/given-outside-boundary] given event "DeskClaimed" states deskId "` + pair.given +
							`" while command "ClaimDesk"'s when payload states deskId "D-5817", so tag "desk" excludes it from the query`,
					}, reportedLines(diags))
				})
			}
		})

		t.Run("value arm: a payload stating the tagged field twice reports nothing", func(t *testing.T) {
			t.Run("the given side", func(t *testing.T) {
				spec := disagreeingDeskSpec()
				spec.Given[0].Payload = append(spec.Given[0].Payload,
					payloadValueAt("deskId", "D-9006", ast.StringLiteral, 18, 52))

				diags := linter.Lint(dcbValueModel(&ast.TagPredicate{Field: "desk", Value: "deskId"}, spec))

				require.Empty(t, diags,
					"neither entry is the value: taking the first or the last would report")
			})

			t.Run("the when side", func(t *testing.T) {
				spec := disagreeingDeskSpec()
				spec.When.Payload = append(spec.When.Payload,
					payloadValueAt("deskId", "D-0000", ast.StringLiteral, 19, 48))

				diags := linter.Lint(dcbValueModel(&ast.TagPredicate{Field: "desk", Value: "deskId"}, spec))

				require.Empty(t, diags,
					"neither entry is the value: taking the first or the last would report")
			})
		})

		t.Run("value arm: a command with no decides_on reports nothing however the payloads disagree", func(t *testing.T) {
			model := dcbValueModel(&ast.TagPredicate{Field: "desk", Value: "deskId"}, disagreeingDeskSpec())
			model.Contexts[0].Slices[0].Commands[0].DecidesOn = nil

			diags := linter.Lint(model)

			require.Equal(t, []string{
				`:0: [dcb/single-tag-everywhere] context "Reading Room" uses only tag key "region" across all decides_on predicates in dcb mode`,
				`reading.emod:10: [dcb/orphan-tag-key] tag key "desk" declared on events is never referenced in any command's decides_on predicate in dcb-mode context "Reading Room"`,
			}, reportedLines(diags),
				"a command that queries nothing states no boundary this rule could place the given outside of")
		})

		t.Run("value arm: a decides_on with no where predicate reports nothing", func(t *testing.T) {
			model := dcbValueModel(nil, disagreeingDeskSpec())

			diags := linter.Lint(model)

			require.Equal(t, []string{
				`:0: [dcb/single-tag-everywhere] context "Reading Room" uses only tag key "region" across all decides_on predicates in dcb mode`,
				`reading.emod:4: [dcb/query-too-broad] command "ClaimDesk" has a broad query: has no where clause`,
				`reading.emod:10: [dcb/orphan-tag-key] tag key "desk" declared on events is never referenced in any command's decides_on predicate in dcb-mode context "Reading Room"`,
			}, reportedLines(diags),
				"the missing predicate is reported by its own rule and leaves this one nothing to compare")
		})

		t.Run("value arm: a given event outside the type-level boundary reports only the type-level diagnostic", func(t *testing.T) {
			spec := disagreeingDeskSpec()
			spec.Given[0] = specElementAt("DeskReleased", 18,
				payloadValueAt("deskId", "D-0001", ast.StringLiteral, 18, 34))
			model := dcbValueModel(&ast.TagPredicate{Field: "desk", Value: "deskId"}, spec)
			model.Contexts[0].Slices[0].Events = append(model.Contexts[0].Slices[0].Events, &ast.Event{
				Name:    "DeskReleased",
				NamePos: ast.Position{Filename: "reading.emod", Line: 12, Column: 9},
				Tags:    []ast.TagEntry{{Key: "desk", FieldRef: "deskId"}, {Key: "region", FieldRef: "regionId"}},
			})

			diags := linter.Lint(model)

			require.Equal(t, []string{
				`reading.emod:18: [spec/given-outside-boundary] given event "DeskReleased" names an event command "ClaimDesk"'s decides_on does not list`,
			}, reportedLines(diags),
				"one mistake earns one diagnostic: the type arm owns a given the query never reaches")
		})

		t.Run("value arm: a when naming an event rather than a command reports nothing", func(t *testing.T) {
			spec := disagreeingDeskSpec()
			spec.When = specElementAt("DeskClaimed", 19,
				payloadValueAt("deskId", "D-5817", ast.StringLiteral, 19, 30))

			diags := linter.Lint(dcbValueModel(&ast.TagPredicate{Field: "desk", Value: "deskId"}, spec))

			require.Empty(t, diags)
		})

		t.Run("value arm: a spec in an aggregate-nested slice takes no value check", func(t *testing.T) {
			model := dcbValueModel(&ast.TagPredicate{Field: "desk", Value: "deskId"})
			nested := model.Contexts[0].Slices[0]
			nested.Specs = append(nested.Specs, disagreeingDeskSpec())
			model.Contexts[0].Slices = model.Contexts[0].Slices[1:]
			model.Contexts[0].Aggregates = []*ast.Aggregate{{Name: "Desk", Slices: []*ast.Slice{nested}}}

			diags := linter.Lint(model)

			require.Empty(t, linesReportedBy(diags, "spec/given-outside-boundary"),
				"arm selection is by slice home, so a nested slice takes the aggregate arm and no value check")
		})

		t.Run("value arm: a model carrying the shape in both slice homes reports for the context-level home only", func(t *testing.T) {
			model := dcbValueModel(&ast.TagPredicate{Field: "desk", Value: "deskId"}, disagreeingDeskSpec())
			nested := disagreeingDeskSpec()
			nested.Given[0] = specElementAt("DeskClaimed", 48,
				payloadValueAt("deskId", "D-4210", ast.StringLiteral, 48, 34))
			model.Contexts[0].Aggregates = []*ast.Aggregate{{
				Name: "Desk",
				Slices: []*ast.Slice{{
					Name:     "Claim Desk Again",
					Commands: model.Contexts[0].Slices[0].Commands,
					Events:   model.Contexts[0].Slices[0].Events,
					Specs:    []*ast.Spec{nested},
				}},
			}}

			diags := linter.Lint(model)

			require.Equal(t, []string{deskValueExcluded}, linesReportedBy(diags, "spec/given-outside-boundary"))
		})

		t.Run("value arm: a conjunction checks each predicate it states", func(t *testing.T) {
			t.Run("one field disagreeing reports that field alone", func(t *testing.T) {
				spec := twoTagDeskSpec()
				spec.Given[0].Payload[1].Value = "R-40817"

				diags := linter.Lint(dcbValueModel(deskAndRegion("and"), spec))

				require.Equal(t, []string{deskValueExcluded}, reportedLines(diags))
			})

			t.Run("both fields disagreeing report once each, at each field's own value", func(t *testing.T) {
				diags := linter.Lint(dcbValueModel(deskAndRegion("and"), twoTagDeskSpec()))

				require.Equal(t, []string{deskValueExcluded, regionValueExcluded}, reportedLines(diags))
				require.Equal(t, []int{34, 52}, reportedColumns(diags),
					"each diagnostic sits on its own field's value, both on the one payload line")
			})

			t.Run("both fields disagreeing come back in source order, not predicate order", func(t *testing.T) {
				spec := twoTagDeskSpec()
				spec.Given[0] = specElementAt("DeskClaimed", 18,
					payloadValueAt("regionId", "R-63204", ast.StringLiteral, 18, 34),
					payloadValueAt("deskId", "D-4210", ast.StringLiteral, 18, 56))

				diags := linter.Lint(dcbValueModel(deskAndRegion("and"), spec))

				require.Equal(t, []int{34, 56}, reportedColumns(diags),
					"the payload writes the second predicate's field first, so only a column tiebreak orders these")
				require.Equal(t, []string{
					`reading.emod:18: [spec/given-outside-boundary] given event "DeskClaimed" states regionId "R-63204" ` +
						`while command "ClaimDesk"'s when payload states regionId "R-40817", so tag "region" excludes it from the query`,
					`reading.emod:18: [spec/given-outside-boundary] given event "DeskClaimed" states deskId "D-4210" ` +
						`while command "ClaimDesk"'s when payload states deskId "D-5817", so tag "desk" excludes it from the query`,
				}, reportedLines(diags))
			})

			t.Run("three predicates joined by two ands are all checked", func(t *testing.T) {
				spec := twoTagDeskSpec()
				spec.Given[0].Payload = append(spec.Given[0].Payload,
					payloadValueAt("zoneId", "Z-11", ast.StringLiteral, 18, 70))
				spec.When.Payload = append(spec.When.Payload,
					payloadValueAt("zoneId", "Z-92", ast.StringLiteral, 19, 66))
				predicate := &ast.LogicalExpr{
					Left:     deskAndRegion("and"),
					Operator: "and",
					Right:    &ast.TagPredicate{Field: "zone", Value: "zoneId"},
				}
				model := dcbValueModel(predicate, spec)
				model.Contexts[0].Slices[0].Events[0].Tags = append(
					model.Contexts[0].Slices[0].Events[0].Tags, ast.TagEntry{Key: "zone", FieldRef: "zoneId"})

				diags := linter.Lint(model)

				require.Equal(t, []string{
					deskValueExcluded,
					regionValueExcluded,
					`reading.emod:18: [spec/given-outside-boundary] given event "DeskClaimed" states zoneId "Z-11" ` +
						`while command "ClaimDesk"'s when payload states zoneId "Z-92", so tag "zone" excludes it from the query`,
				}, reportedLines(diags))
			})

			t.Run("two keys the event binds to one field report once per predicate", func(t *testing.T) {
				predicate := &ast.LogicalExpr{
					Left:     &ast.TagPredicate{Field: "desk", Value: "deskId"},
					Operator: "and",
					Right:    &ast.TagPredicate{Field: "region", Value: "deskId"},
				}
				model := dcbValueModel(predicate, disagreeingDeskSpec())
				model.Contexts[0].Slices[0].Events[0].Tags = []ast.TagEntry{
					{Key: "desk", FieldRef: "deskId"},
					{Key: "region", FieldRef: "deskId"},
				}

				diags := linter.Lint(model)

				require.Equal(t, []string{
					deskValueExcluded,
					`reading.emod:18: [spec/given-outside-boundary] given event "DeskClaimed" states deskId "D-4210" ` +
						`while command "ClaimDesk"'s when payload states deskId "D-5817", so tag "region" excludes it from the query`,
				}, reportedLines(diags),
					"each predicate states a separate routing requirement, so each is separately fixable")
			})
		})

		t.Run("value arm: the given side is read through the field the event binds the tag key to", func(t *testing.T) {
			// tags { desk: seatId } routes the desk tag on seatId, while the
			// predicate names the command's own deskId — so the two sides are
			// looked up under different names.
			mapped := func(seat, deskOnGiven, deskOnWhen string) *ast.Model {
				spec := &ast.Spec{
					Name: "claims a desk another reader holds",
					Given: []*ast.SpecElement{
						specElementAt("DeskClaimed", 18,
							payloadValueAt("seatId", seat, ast.StringLiteral, 18, 34),
							payloadValueAt("deskId", deskOnGiven, ast.StringLiteral, 18, 56)),
					},
					When: specElementAt("ClaimDesk", 19,
						payloadValueAt("deskId", deskOnWhen, ast.StringLiteral, 19, 30)),
					Then: &ast.ThenEvents{},
				}
				model := dcbValueModel(&ast.TagPredicate{Field: "desk", Value: "deskId"}, spec)
				model.Contexts[0].Slices[0].Events[0].Tags = []ast.TagEntry{
					{Key: "desk", FieldRef: "seatId"},
					{Key: "region", FieldRef: "regionId"},
				}

				return model
			}

			t.Run("the bound field agreeing keeps the event inside the query", func(t *testing.T) {
				diags := linter.Lint(mapped("D-5817", "D-4210", "D-5817"))

				require.Empty(t, diags,
					"the desk tag reads seatId, and a same-named deskId the query never consults must not report")
			})

			t.Run("the bound field disagreeing puts the event outside it", func(t *testing.T) {
				diags := linter.Lint(mapped("S-99", "D-5817", "D-5817"))

				require.Equal(t, []string{
					`reading.emod:18: [spec/given-outside-boundary] given event "DeskClaimed" states seatId "S-99" ` +
						`while command "ClaimDesk"'s when payload states deskId "D-5817", so tag "desk" excludes it from the query`,
				}, reportedLines(diags))
			})

			t.Run("an event binding no such key leaves the comparison no basis", func(t *testing.T) {
				model := mapped("S-99", "D-4210", "D-5817")
				model.Contexts[0].Slices[0].Events[0].Tags = []ast.TagEntry{{Key: "region", FieldRef: "regionId"}}

				require.Empty(t, linesReportedBy(linter.Lint(model), "spec/given-outside-boundary"),
					"the payloads disagree on deskId too, so only the missing key can be keeping this silent")
			})

			t.Run("a same-named event in another context does not supply the mapping", func(t *testing.T) {
				model := mapped("D-5817", "D-4210", "D-5817")
				model.Contexts = append([]*ast.Context{{
					Name: "Archive",
					Mode: "dcb",
					Slices: []*ast.Slice{{
						Name: "Shelve Desk",
						Events: []*ast.Event{{
							Name:    "DeskClaimed",
							NamePos: ast.Position{Filename: "archive.emod", Line: 6, Column: 9},
							Tags:    []ast.TagEntry{{Key: "desk", FieldRef: "deskId"}},
						}},
					}},
				}}, model.Contexts...)

				require.Empty(t, linesReportedBy(linter.Lint(model), "spec/given-outside-boundary"),
					"the Reading Room event binds desk to seatId, and seatId agrees; reading Archive's mapping would report deskId")
			})

			t.Run("an event only another context declares binds nothing here", func(t *testing.T) {
				spec := disagreeingDeskSpec()
				spec.Given[0] = specElementAt("ShelvedDesk", 18,
					payloadValueAt("deskId", "D-4210", ast.StringLiteral, 18, 34))
				model := dcbValueModel(&ast.TagPredicate{Field: "desk", Value: "deskId"}, spec)
				model.Contexts[0].Slices[0].Commands[0].DecidesOn.Events = []string{"DeskClaimed", "ShelvedDesk"}
				model.Contexts = append(model.Contexts, &ast.Context{
					Name: "Archive",
					Slices: []*ast.Slice{{
						Name: "Shelve Desk",
						Events: []*ast.Event{{
							Name:    "ShelvedDesk",
							NamePos: ast.Position{Filename: "archive.emod", Line: 6, Column: 9},
							Tags:    []ast.TagEntry{{Key: "desk", FieldRef: "deskId"}},
						}},
					}},
				})

				require.Empty(t, linesReportedBy(linter.Lint(model), "spec/given-outside-boundary"),
					"the payloads disagree on deskId, so only the event being absent from this context keeps it silent")
			})

			t.Run("a same-named command in another context does not supply the predicate", func(t *testing.T) {
				spec := disagreeingDeskSpec()
				spec.When = specElementAt("ClaimDesk", 19,
					payloadValueAt("deskId", "D-4210", ast.StringLiteral, 19, 30),
					payloadValueAt("seatId", "S-9", ast.StringLiteral, 19, 52))
				model := dcbValueModel(&ast.TagPredicate{Field: "desk", Value: "deskId"}, spec)
				model.Contexts = append(model.Contexts, &ast.Context{
					Name: "Study Hall",
					Mode: "dcb",
					Slices: []*ast.Slice{{
						Name: "Claim Seat",
						Commands: []*ast.Command{{
							Name:    "ClaimDesk",
							NamePos: ast.Position{Filename: "study.emod", Line: 4, Column: 9},
							DecidesOn: &ast.DecidesOnClause{
								Events:    []string{"DeskClaimed"},
								Predicate: &ast.TagPredicate{Field: "desk", Value: "seatId"},
							},
						}},
					}},
				})

				require.Empty(t, linesReportedBy(linter.Lint(model), "spec/given-outside-boundary"),
					"Reading Room's own predicate names deskId, and both payloads state D-4210; "+
						"reading Study Hall's predicate would compare deskId against seatId")
			})

			t.Run("a mixed-mode context binds through an event its own aggregate declares", func(t *testing.T) {
				spec := disagreeingDeskSpec()
				model := dcbValueModel(&ast.TagPredicate{Field: "desk", Value: "deskId"}, spec)
				model.Contexts[0].Mode = "mixed"
				model.Contexts[0].Aggregates = []*ast.Aggregate{{
					Name:   "Desk",
					Slices: []*ast.Slice{{Name: "Record Desk", Events: model.Contexts[0].Slices[0].Events}},
				}}
				model.Contexts[0].Slices[0].Events = nil

				require.Equal(t, []string{deskValueExcluded},
					linesReportedBy(linter.Lint(model), "spec/given-outside-boundary"),
					"the event lives in an aggregate of this context, which still declares it")
			})

			t.Run("an event a translation declares in this context binds its own tags", func(t *testing.T) {
				spec := disagreeingDeskSpec()
				spec.Given[0] = specElementAt("DeskImported", 18,
					payloadValueAt("seatId", "S-99", ast.StringLiteral, 18, 34))
				model := dcbValueModel(&ast.TagPredicate{Field: "desk", Value: "deskId"}, spec)
				model.Contexts[0].Slices[0].Commands[0].DecidesOn.Events = []string{"DeskClaimed", "DeskImported"}
				model.Contexts[0].Slices[0].Translations = []*ast.Translation{{
					Name: "ImportDesk",
					Event: &ast.Event{
						Name:    "DeskImported",
						NamePos: ast.Position{Filename: "reading.emod", Line: 14, Column: 9},
						Tags:    []ast.TagEntry{{Key: "desk", FieldRef: "seatId"}},
					},
				}}

				require.Equal(t, []string{
					`reading.emod:18: [spec/given-outside-boundary] given event "DeskImported" states seatId "S-99" ` +
						`while command "ClaimDesk"'s when payload states deskId "D-5817", so tag "desk" excludes it from the query`,
				}, linesReportedBy(linter.Lint(model), "spec/given-outside-boundary"))
			})
		})

		t.Run("value arm: a predicate the query does not require is not checked", func(t *testing.T) {
			t.Run("two predicates joined by or", func(t *testing.T) {
				diags := linter.Lint(dcbValueModel(deskAndRegion("or"), twoTagDeskSpec()))

				require.Empty(t, diags,
					"under or a single disagreement does not put the event outside the query")
			})

			t.Run("a not at the root", func(t *testing.T) {
				predicate := &ast.NotExpr{Expr: &ast.TagPredicate{Field: "desk", Value: "deskId"}}

				diags := linter.Lint(dcbValueModel(predicate, disagreeingDeskSpec()))

				require.Empty(t, diags, "under not a disagreement argues the event is inside the query")
			})

			t.Run("a not as one operand of an and", func(t *testing.T) {
				predicate := &ast.LogicalExpr{
					Left:     &ast.NotExpr{Expr: &ast.TagPredicate{Field: "desk", Value: "deskId"}},
					Operator: "and",
					Right:    &ast.TagPredicate{Field: "region", Value: "regionId"},
				}
				spec := twoTagDeskSpec()

				diags := linter.Lint(dcbValueModel(predicate, spec))

				require.Equal(t, []string{regionValueExcluded}, reportedLines(diags),
					"the conjunctive operand is still checked; the negated one is not")
			})

			t.Run("an or nested inside an and", func(t *testing.T) {
				predicate := &ast.LogicalExpr{
					Left:     &ast.TagPredicate{Field: "desk", Value: "deskId"},
					Operator: "and",
					Right:    deskAndRegion("or"),
				}
				spec := twoTagDeskSpec()

				diags := linter.Lint(dcbValueModel(predicate, spec))

				require.Equal(t, []string{deskValueExcluded}, reportedLines(diags),
					"the and's other operand is checked and neither of the or's is")
			})
		})

		t.Run("value arm: every silence rule holds under a conjunction while the sibling predicate still reports", func(t *testing.T) {
			t.Run("a field the given payload omits", func(t *testing.T) {
				spec := twoTagDeskSpec()
				spec.Given[0].Payload = spec.Given[0].Payload[1:]

				diags := linter.Lint(dcbValueModel(deskAndRegion("and"), spec))

				require.Equal(t, []string{regionValueExcluded}, reportedLines(diags))
			})

			t.Run("a field the given payload states twice", func(t *testing.T) {
				spec := twoTagDeskSpec()
				spec.Given[0].Payload = append(spec.Given[0].Payload,
					payloadValueAt("deskId", "D-9006", ast.StringLiteral, 18, 70))

				diags := linter.Lint(dcbValueModel(deskAndRegion("and"), spec))

				require.Equal(t, []string{regionValueExcluded}, reportedLines(diags))
			})

			t.Run("a field the two payloads state at different literal kinds", func(t *testing.T) {
				spec := twoTagDeskSpec()
				spec.Given[0].Payload[0] = payloadValueAt("deskId", "4210", ast.IntegerLiteral, 18, 34)

				diags := linter.Lint(dcbValueModel(deskAndRegion("and"), spec))

				require.Equal(t, []string{regionValueExcluded}, reportedLines(diags))
			})
		})

		t.Run("value arm: reports every disagreeing given element in declaration order", func(t *testing.T) {
			spec := disagreeingDeskSpec()
			spec.Given = append(spec.Given, specElementAt("DeskClaimed", 20,
				payloadValueAt("deskId", "D-9006", ast.StringLiteral, 20, 34)))

			diags := linter.Lint(dcbValueModel(&ast.TagPredicate{Field: "desk", Value: "deskId"}, spec))

			require.Equal(t, []string{
				deskValueExcluded,
				`reading.emod:20: [spec/given-outside-boundary] given event "DeskClaimed" states deskId "D-9006" ` +
					`while command "ClaimDesk"'s when payload states deskId "D-5817", so tag "desk" excludes it from the query`,
			}, reportedLines(diags))
		})

		t.Run("value arm: a then payload is never read", func(t *testing.T) {
			spec := disagreeingDeskSpec()
			spec.Given[0].Payload[0].Value = "D-5817"
			spec.Then = &ast.ThenEvents{Events: []*ast.SpecElement{
				specElementAt("DeskClaimed", 21,
					payloadValueAt("deskId", "D-0000", ast.StringLiteral, 21, 34)),
			}}

			diags := linter.Lint(dcbValueModel(&ast.TagPredicate{Field: "desk", Value: "deskId"}, spec))

			require.Empty(t, diags,
				"a then outcome states what the command produces, not history the query has to reach")
		})

		t.Run("all three arms report under one rule name with three distinct texts", func(t *testing.T) {
			model := dcbValueModel(&ast.TagPredicate{Field: "desk", Value: "deskId"}, disagreeingDeskSpec())
			outsideType := disagreeingDeskSpec()
			outsideType.Name = "claims a desk after release"
			outsideType.Given = []*ast.SpecElement{specElementAt("DeskReleased", 24)}
			model.Contexts[0].Slices[0].Specs = append(model.Contexts[0].Slices[0].Specs, outsideType)
			model.Contexts[0].Slices[0].Events = append(model.Contexts[0].Slices[0].Events, &ast.Event{
				Name:    "DeskReleased",
				NamePos: ast.Position{Filename: "reading.emod", Line: 12, Column: 9},
				Tags:    []ast.TagEntry{{Key: "desk", FieldRef: "deskId"}, {Key: "region", FieldRef: "regionId"}},
			})
			model.Contexts = append(model.Contexts, &ast.Context{
				Name: "Lending",
				Aggregates: []*ast.Aggregate{
					{Name: "Loan", Slices: []*ast.Slice{{Name: "Borrow Copy", Events: []*ast.Event{
						{Name: "CopyBorrowed", NamePos: ast.Position{Filename: "lending.emod", Line: 6, Column: 9}},
					}}}},
					{Name: "Reader", Slices: []*ast.Slice{{Name: "Read Copy", Specs: []*ast.Spec{
						{Name: "reads a borrowed copy", When: &ast.SpecElement{Name: "ReadCopy"}, Given: []*ast.SpecElement{
							{Name: "CopyBorrowed", NamePos: ast.Position{Filename: "lending.emod", Line: 16, Column: 15}},
						}},
					}}}},
				},
			})

			diags := linter.Lint(model)

			require.Equal(t, []string{
				`lending.emod:16: [spec/given-outside-boundary] given event "CopyBorrowed" names an event declared by aggregate "Loan" instead of aggregate "Reader"`,
				deskValueExcluded,
				`reading.emod:24: [spec/given-outside-boundary] given event "DeskReleased" names an event command "ClaimDesk"'s decides_on does not list`,
			}, linesReportedBy(diags, "spec/given-outside-boundary"))
		})
	})
}

// linesReportedBy selects the formatted lines one rule produced, so a leaf about
// that rule can be written over a model another rule also has something to say
// about.
func linesReportedBy(diags []*diagnostic.Entry, rule string) []string {
	var lines []string
	for _, d := range diags {
		if d.RuleName == rule {
			lines = append(lines, d.String())
		}
	}

	return lines
}

func reportedLines(diags []*diagnostic.Entry) []string {
	lines := make([]string, 0, len(diags))
	for _, d := range diags {
		lines = append(lines, d.String())
	}

	return lines
}

// dcbValueModel builds a two-slice DCB model for the value-level arm of
// spec/given-outside-boundary. ClaimDesk decides on DeskClaimed under predicate
// and carries the specs passed in beside a happy-path one and a rejection;
// Release Desk routes on the context's other tag key and carries the same pair.
// That keeps dcb/single-tag-everywhere, dcb/orphan-tag-key, dcb/query-too-broad,
// spec/command-without-spec, spec/no-rejection-path and
// spec/invariant-never-exercised quiet, so any diagnostic over the returned
// model came from the specs the caller wrote.
func dcbValueModel(predicate ast.PredicateExpr, specs ...*ast.Spec) *ast.Model {
	claimDesk := &ast.Slice{
		Name: "Claim Desk",
		Commands: []*ast.Command{{
			Name:    "ClaimDesk",
			NamePos: ast.Position{Filename: "reading.emod", Line: 4, Column: 9},
			DecidesOn: &ast.DecidesOnClause{
				Events:    []string{"DeskClaimed"},
				Predicate: predicate,
			},
		}},
		Events: []*ast.Event{{
			Name:    "DeskClaimed",
			NamePos: ast.Position{Filename: "reading.emod", Line: 10, Column: 9},
			Tags:    []ast.TagEntry{{Key: "desk", FieldRef: "deskId"}, {Key: "region", FieldRef: "regionId"}},
		}},
		Specs: append([]*ast.Spec{
			{Name: "claims a free desk", When: &ast.SpecElement{Name: "ClaimDesk"}, Then: &ast.ThenEvents{}},
			{Name: "refuses a desk in use", When: &ast.SpecElement{Name: "ClaimDesk"}, Then: &ast.ThenRejected{InvariantName: "OneDeskPerReader"}},
		}, specs...),
	}

	releaseDesk := &ast.Slice{
		Name: "Release Desk",
		Commands: []*ast.Command{{
			Name:    "ReleaseDesk",
			NamePos: ast.Position{Filename: "reading.emod", Line: 30, Column: 9},
			DecidesOn: &ast.DecidesOnClause{
				Events:    []string{"DeskClaimed"},
				Predicate: &ast.TagPredicate{Field: "region", Value: "regionId"},
			},
		}},
		Specs: []*ast.Spec{
			{Name: "frees a desk in use", When: &ast.SpecElement{Name: "ReleaseDesk"}, Then: &ast.ThenEvents{}},
			{Name: "refuses to free a free desk", When: &ast.SpecElement{Name: "ReleaseDesk"}, Then: &ast.ThenRejected{InvariantName: "OneDeskPerReader"}},
		},
	}

	return &ast.Model{
		Contexts: []*ast.Context{{
			Name: "Reading Room",
			Mode: "dcb",
			Invariants: []*ast.Invariant{
				{Name: "OneDeskPerReader", NamePos: ast.Position{Filename: "reading.emod", Line: 2, Column: 18}},
			},
			Slices: []*ast.Slice{claimDesk, releaseDesk},
		}},
	}
}

func specElementAt(name string, line int, payload ...*ast.PayloadField) *ast.SpecElement {
	return &ast.SpecElement{
		Name:    name,
		NamePos: ast.Position{Filename: "reading.emod", Line: line, Column: 15},
		Payload: payload,
	}
}

// payloadValueAt positions the literal rather than the field name, because the
// value-level arm reports at the value an author would edit.
func payloadValueAt(field, value string, kind ast.LiteralKind, line, column int) *ast.PayloadField {
	return &ast.PayloadField{
		Name:     field,
		Value:    value,
		Kind:     kind,
		ValuePos: ast.Position{Filename: "reading.emod", Line: line, Column: column},
	}
}

// disagreeingDeskSpec is the shape the value arm exists to report: a given event
// the command's decides_on lists, stating the tagged field with a value the when
// payload contradicts.
func disagreeingDeskSpec() *ast.Spec {
	return &ast.Spec{
		Name: "claims a desk another reader holds",
		Given: []*ast.SpecElement{
			specElementAt("DeskClaimed", 18,
				payloadValueAt("deskId", "D-4210", ast.StringLiteral, 18, 34)),
		},
		When: specElementAt("ClaimDesk", 19,
			payloadValueAt("deskId", "D-5817", ast.StringLiteral, 19, 30)),
		Then: &ast.ThenEvents{},
	}
}

// twoTagDeskSpec disagrees on both fields a two-tag conjunction routes on, and
// writes each payload on one line, the way emod fmt writes it — so two
// diagnostics from one given element are told apart by column rather than by
// line.
func twoTagDeskSpec() *ast.Spec {
	return &ast.Spec{
		Name: "claims a desk another reader holds",
		Given: []*ast.SpecElement{
			specElementAt("DeskClaimed", 18,
				payloadValueAt("deskId", "D-4210", ast.StringLiteral, 18, 34),
				payloadValueAt("regionId", "R-63204", ast.StringLiteral, 18, 52)),
		},
		When: specElementAt("ClaimDesk", 19,
			payloadValueAt("deskId", "D-5817", ast.StringLiteral, 19, 30),
			payloadValueAt("regionId", "R-40817", ast.StringLiteral, 19, 48)),
		Then: &ast.ThenEvents{},
	}
}

// deskAndRegion names the fields DeskClaimed's own tags block binds each key to,
// which is what the given side is read under.
func deskAndRegion(operator string) *ast.LogicalExpr {
	return &ast.LogicalExpr{
		Left:     &ast.TagPredicate{Field: "desk", Value: "deskId"},
		Operator: operator,
		Right:    &ast.TagPredicate{Field: "region", Value: "regionId"},
	}
}

func reportedColumns(diags []*diagnostic.Entry) []int {
	columns := make([]int, 0, len(diags))
	for _, d := range diags {
		columns = append(columns, d.Column)
	}

	return columns
}

const deskValueExcluded = `reading.emod:18: [spec/given-outside-boundary] ` +
	`given event "DeskClaimed" states deskId "D-4210" while command "ClaimDesk"'s ` +
	`when payload states deskId "D-5817", so tag "desk" excludes it from the query`

const regionValueExcluded = `reading.emod:18: [spec/given-outside-boundary] ` +
	`given event "DeskClaimed" states regionId "R-63204" while command "ClaimDesk"'s ` +
	`when payload states regionId "R-40817", so tag "region" excludes it from the query`

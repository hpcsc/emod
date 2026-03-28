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

	t.Run("all five rules fire in a single Lint invocation", func(t *testing.T) {
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
									},
									Commands: []*ast.Command{
										{
											Name: "ReservationCancelled",
											NamePos: ast.Position{
												Filename: "orders.emod",
												Line:     4,
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
									},
								},
							},
						},
					},
				},
			},
		}

		diags := linter.Lint(model)

		require.Len(t, diags, 5)

		ruleNames := make(map[string]bool)
		for _, d := range diags {
			ruleNames[d.RuleName] = true
		}
		require.Len(t, ruleNames, 5)
		require.True(t, ruleNames["state-obsession"])
		require.True(t, ruleNames["property-sourcing"])
		require.True(t, ruleNames["command-in-disguise"])
		require.True(t, ruleNames["command-past-tense"])
		require.True(t, ruleNames["view-naming"])

		linesByRule := make(map[string]int)
		for _, d := range diags {
			linesByRule[d.RuleName] = d.Line
		}
		require.Equal(t, 1, linesByRule["state-obsession"])
		require.Equal(t, 2, linesByRule["property-sourcing"])
		require.Equal(t, 3, linesByRule["command-in-disguise"])
		require.Equal(t, 4, linesByRule["command-past-tense"])
		require.Equal(t, 5, linesByRule["view-naming"])
	})
}

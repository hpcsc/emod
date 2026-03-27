//go:build unit

package linter_test

import (
	"testing"

	"github.com/hpcsc/emod/internal/ast"
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
}

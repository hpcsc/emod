//go:build unit

package validator_test

import (
	"testing"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/validator"
	"github.com/stretchr/testify/require"
)

func TestValidate(t *testing.T) {
	t.Run("nil model produces no diagnostics", func(t *testing.T) {
		diags := validator.Validate(nil)

		require.Empty(t, diags)
	})

	t.Run("valid target context reference produces no diagnostics", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Aggregates: []*ast.Aggregate{
						{
							Slices: []*ast.Slice{
								{
									Automations: []*ast.Automation{
										{
											Name:          "NotifyOnOrder",
											TargetContext: "Notifications",
										},
									},
								},
							},
						},
					},
				},
				{
					Name: "Notifications",
				},
			},
		}

		diags := validator.Validate(model)

		require.Empty(t, diags)
	})

	t.Run("invalid target context reference produces one diagnostic", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Aggregates: []*ast.Aggregate{
						{
							Slices: []*ast.Slice{
								{
									Automations: []*ast.Automation{
										{
											Name:          "NotifyOnOrder",
											TargetContext: "NonExistent",
										},
									},
								},
							},
						},
					},
				},
			},
		}

		diags := validator.Validate(model)

		require.Len(t, diags, 1)
		require.Equal(t, `target context "NonExistent" does not exist`, diags[0].Message)
	})

	t.Run("automation without target context produces no diagnostics", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Aggregates: []*ast.Aggregate{
						{
							Slices: []*ast.Slice{
								{
									Automations: []*ast.Automation{
										{
											Name:          "AutoConfirm",
											TargetContext: "",
										},
									},
								},
							},
						},
					},
				},
			},
		}

		diags := validator.Validate(model)

		require.Empty(t, diags)
	})

	t.Run("diagnostic includes position from target context", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Aggregates: []*ast.Aggregate{
						{
							Slices: []*ast.Slice{
								{
									Automations: []*ast.Automation{
										{
											Name:          "NotifyOnOrder",
											TargetContext: "NonExistent",
											TargetContextPos: ast.Position{
												Filename: "test.emod",
												Line:     10,
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
		}

		diags := validator.Validate(model)

		require.Len(t, diags, 1)
		require.Equal(t, "test.emod", diags[0].Filename)
		require.Equal(t, 10, diags[0].Line)
		require.Equal(t, 5, diags[0].Column)
	})

	t.Run("multiple invalid references across contexts produce multiple diagnostics", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Aggregates: []*ast.Aggregate{
						{
							Slices: []*ast.Slice{
								{
									Automations: []*ast.Automation{
										{
											Name:          "NotifyOnOrder",
											TargetContext: "Ghost",
										},
									},
								},
							},
						},
					},
				},
				{
					Name: "Billing",
					Aggregates: []*ast.Aggregate{
						{
							Slices: []*ast.Slice{
								{
									Automations: []*ast.Automation{
										{
											Name:          "ChargeBilling",
											TargetContext: "Phantom",
										},
									},
								},
							},
						},
					},
				},
			},
		}

		diags := validator.Validate(model)

		require.Len(t, diags, 2)
		require.Equal(t, `target context "Ghost" does not exist`, diags[0].Message)
		require.Equal(t, `target context "Phantom" does not exist`, diags[1].Message)
	})
}

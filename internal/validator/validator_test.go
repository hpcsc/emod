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

	t.Run("automation referencing command in same context produces no diagnostics", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Aggregates: []*ast.Aggregate{
						{
							Slices: []*ast.Slice{
								{
									Commands: []*ast.Command{
										{Name: "PlaceOrder"},
									},
									Automations: []*ast.Automation{
										{
											Name:    "AutoConfirm",
											Command: "PlaceOrder",
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

	t.Run("automation referencing command in different context produces no diagnostics", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Aggregates: []*ast.Aggregate{
						{
							Slices: []*ast.Slice{
								{
									Commands: []*ast.Command{
										{Name: "PlaceOrder"},
									},
								},
							},
						},
					},
				},
				{
					Name: "Shipping",
					Aggregates: []*ast.Aggregate{
						{
							Slices: []*ast.Slice{
								{
									Automations: []*ast.Automation{
										{
											Name:    "ShipOnOrder",
											Command: "PlaceOrder",
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

	t.Run("automation referencing nonexistent command produces one diagnostic", func(t *testing.T) {
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
											Name:    "AutoConfirm",
											Command: "NonExistentCmd",
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
		require.Equal(t, `command "NonExistentCmd" does not exist`, diags[0].Message)
	})

	t.Run("translation referencing nonexistent command produces one diagnostic", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Aggregates: []*ast.Aggregate{
						{
							Slices: []*ast.Slice{
								{
									Translations: []*ast.Translation{
										{
											Name:    "ImportOrder",
											Command: "MissingCmd",
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
		require.Equal(t, `command "MissingCmd" does not exist`, diags[0].Message)
	})

	t.Run("command reference diagnostic includes position from CommandPos", func(t *testing.T) {
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
											Name:    "AutoConfirm",
											Command: "NonExistentCmd",
											CommandPos: ast.Position{
												Filename: "test.emod",
												Line:     15,
												Column:   9,
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
		require.Equal(t, 15, diags[0].Line)
		require.Equal(t, 9, diags[0].Column)
	})

	t.Run("automation with empty command field produces no diagnostics", func(t *testing.T) {
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
											Name:    "AutoConfirm",
											Command: "",
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

	t.Run("translation with empty command field produces no diagnostics", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Aggregates: []*ast.Aggregate{
						{
							Slices: []*ast.Slice{
								{
									Translations: []*ast.Translation{
										{
											Name:    "ImportOrder",
											Command: "",
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

	t.Run("translation command reference diagnostic includes position from CommandPos", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Aggregates: []*ast.Aggregate{
						{
							Slices: []*ast.Slice{
								{
									Translations: []*ast.Translation{
										{
											Name:    "ImportOrder",
											Command: "NonExistentCmd",
											CommandPos: ast.Position{
												Filename: "trans.emod",
												Line:     20,
												Column:   12,
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
		require.Equal(t, "trans.emod", diags[0].Filename)
		require.Equal(t, 20, diags[0].Line)
		require.Equal(t, 12, diags[0].Column)
	})

	t.Run("automation and translation each referencing nonexistent commands produce two diagnostics", func(t *testing.T) {
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
											Name:    "AutoConfirm",
											Command: "GhostCmd",
										},
									},
									Translations: []*ast.Translation{
										{
											Name:    "ImportOrder",
											Command: "PhantomCmd",
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
		require.Equal(t, `command "GhostCmd" does not exist`, diags[0].Message)
		require.Equal(t, `command "PhantomCmd" does not exist`, diags[1].Message)
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

//go:build unit

package validator_test

import (
	"testing"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/diagnostic"
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
									Flows: []*ast.Flow{
										{CommandName: "PlaceOrder"},
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
									Flows: []*ast.Flow{
										{CommandName: "PlaceOrder"},
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

	t.Run("automation trigger referencing existing event produces no diagnostics", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Aggregates: []*ast.Aggregate{
						{
							Slices: []*ast.Slice{
								{
									Events: []*ast.Event{
										{Name: "OrderPlaced", Source: "ExternalSystem"},
									},
									Automations: []*ast.Automation{
										{
											Name:         "NotifyOnOrder",
											TriggerEvent: "OrderPlaced",
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

	t.Run("automation trigger referencing non-existent event produces one diagnostic", func(t *testing.T) {
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
											Name:         "ShipOrder",
											TriggerEvent: "OrderShipped",
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
		require.Equal(t, `event "OrderShipped" does not exist`, diags[0].Message)
	})

	t.Run("automation trigger diagnostic includes position from TriggerEventPos", func(t *testing.T) {
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
											Name:         "ShipOrder",
											TriggerEvent: "OrderShipped",
											TriggerEventPos: ast.Position{
												Filename: "test.emod",
												Line:     8,
												Column:   17,
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
		require.Equal(t, 8, diags[0].Line)
		require.Equal(t, 17, diags[0].Column)
	})

	t.Run("empty automation trigger skipped", func(t *testing.T) {
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
											Name:         "AutoConfirm",
											TriggerEvent: "",
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

	t.Run("view subscribes non-existent event produces one diagnostic", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Aggregates: []*ast.Aggregate{
						{
							Slices: []*ast.Slice{
								{
									Views: []*ast.View{
										{
											Name:       "OrderList",
											Subscribes: []string{"NoSuchEvent"},
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
		require.Equal(t, `event "NoSuchEvent" does not exist`, diags[0].Message)
	})

	t.Run("view subscribes diagnostic uses View.NamePos", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Aggregates: []*ast.Aggregate{
						{
							Slices: []*ast.Slice{
								{
									Views: []*ast.View{
										{
											Name:       "OrderList",
											Subscribes: []string{"NoSuchEvent"},
											NamePos: ast.Position{
												Filename: "test.emod",
												Line:     12,
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

		diags := validator.Validate(model)

		require.Len(t, diags, 1)
		require.Equal(t, "test.emod", diags[0].Filename)
		require.Equal(t, 12, diags[0].Line)
		require.Equal(t, 7, diags[0].Column)
	})

	t.Run("view with multiple invalid subscribes produces diagnostics for each", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Aggregates: []*ast.Aggregate{
						{
							Slices: []*ast.Slice{
								{
									Views: []*ast.View{
										{
											Name:       "OrderList",
											Subscribes: []string{"EventA", "EventB"},
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
		require.Equal(t, `event "EventA" does not exist`, diags[0].Message)
		require.Equal(t, `event "EventB" does not exist`, diags[1].Message)
	})

	t.Run("flow event non-existent produces one diagnostic", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Aggregates: []*ast.Aggregate{
						{
							Slices: []*ast.Slice{
								{
									Flows: []*ast.Flow{
										{
											EventName: "MissingEvt",
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
		require.Equal(t, `event "MissingEvt" does not exist`, diags[0].Message)
	})

	t.Run("flow event diagnostic includes position from EventPos", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Aggregates: []*ast.Aggregate{
						{
							Slices: []*ast.Slice{
								{
									Flows: []*ast.Flow{
										{
											EventName: "MissingEvt",
											EventPos: ast.Position{
												Filename: "test.emod",
												Line:     25,
												Column:   30,
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
		require.Equal(t, 25, diags[0].Line)
		require.Equal(t, 30, diags[0].Column)
	})

	t.Run("event in different context found by automation trigger", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Aggregates: []*ast.Aggregate{
						{
							Slices: []*ast.Slice{
								{
									Events: []*ast.Event{
										{Name: "OrderPlaced", Source: "ExternalSystem"},
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
											Name:         "ShipOnOrder",
											TriggerEvent: "OrderPlaced",
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

	t.Run("inline Translation.Event included in event lookup", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Bookings",
					Aggregates: []*ast.Aggregate{
						{
							Slices: []*ast.Slice{
								{
									Translations: []*ast.Translation{
										{
											Name: "ImportBooking",
											Event: &ast.Event{
												Name: "BookingImported",
											},
										},
									},
									Automations: []*ast.Automation{
										{
											Name:         "ProcessBooking",
											TriggerEvent: "BookingImported",
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

	t.Run("empty flow event name skipped", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Aggregates: []*ast.Aggregate{
						{
							Slices: []*ast.Slice{
								{
									Flows: []*ast.Flow{
										{
											EventName: "",
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

	t.Run("command defined with no flow reference produces orphan diagnostic", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Aggregates: []*ast.Aggregate{
						{
							Slices: []*ast.Slice{
								{
									Commands: []*ast.Command{
										{Name: "PlaceOrder", NamePos: ast.Position{Filename: "test.emod", Line: 5, Column: 3}},
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
		require.Equal(t, diagnostic.Error, diags[0].Severity)
		require.Equal(t, "orphan-command", diags[0].RuleName)
		require.Contains(t, diags[0].Message, "PlaceOrder")
		require.Contains(t, diags[0].Message, "orphaned")
		require.Equal(t, "test.emod", diags[0].Filename)
		require.Equal(t, 5, diags[0].Line)
		require.Equal(t, 3, diags[0].Column)
	})

	t.Run("command referenced by flow produces no orphan diagnostic", func(t *testing.T) {
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
									Flows: []*ast.Flow{
										{CommandName: "PlaceOrder"},
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

	t.Run("orphan command diagnostic includes command definition position", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Aggregates: []*ast.Aggregate{
						{
							Slices: []*ast.Slice{
								{
									Commands: []*ast.Command{
										{Name: "ShipOrder", NamePos: ast.Position{Filename: "orders.emod", Line: 12, Column: 7}},
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
		require.Equal(t, "orders.emod", diags[0].Filename)
		require.Equal(t, 12, diags[0].Line)
		require.Equal(t, 7, diags[0].Column)
		require.Equal(t, "orphan-command", diags[0].RuleName)
	})

	t.Run("event defined with no producer produces orphan diagnostic", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Aggregates: []*ast.Aggregate{
						{
							Slices: []*ast.Slice{
								{
									Events: []*ast.Event{
										{Name: "OrderPlaced", NamePos: ast.Position{Filename: "test.emod", Line: 5, Column: 3}},
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
		require.Equal(t, diagnostic.Error, diags[0].Severity)
		require.Equal(t, "orphan-event", diags[0].RuleName)
		require.Contains(t, diags[0].Message, "OrderPlaced")
		require.Contains(t, diags[0].Message, "orphaned")
	})

	t.Run("event referenced by flow produces no orphan diagnostic", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Aggregates: []*ast.Aggregate{
						{
							Slices: []*ast.Slice{
								{
									Events: []*ast.Event{
										{Name: "OrderPlaced"},
									},
									Flows: []*ast.Flow{
										{EventName: "OrderPlaced"},
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

	t.Run("event with external source produces no orphan diagnostic", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Aggregates: []*ast.Aggregate{
						{
							Slices: []*ast.Slice{
								{
									Events: []*ast.Event{
										{Name: "OrderPlaced", Source: "ExternalSystem"},
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

	t.Run("event inside translation produces no orphan diagnostic", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Bookings",
					Aggregates: []*ast.Aggregate{
						{
							Slices: []*ast.Slice{
								{
									Translations: []*ast.Translation{
										{
											Name: "ImportBooking",
											Event: &ast.Event{
												Name: "BookingImported",
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

		require.Empty(t, diags)
	})

	t.Run("orphan event diagnostic includes event definition position", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Aggregates: []*ast.Aggregate{
						{
							Slices: []*ast.Slice{
								{
									Events: []*ast.Event{
										{Name: "OrderPlaced", NamePos: ast.Position{Filename: "orders.emod", Line: 8, Column: 5}},
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
		require.Equal(t, "orders.emod", diags[0].Filename)
		require.Equal(t, 8, diags[0].Line)
		require.Equal(t, 5, diags[0].Column)
		require.Equal(t, "orphan-event", diags[0].RuleName)
	})

	t.Run("event referenced by flow in different context produces no orphan diagnostic", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Aggregates: []*ast.Aggregate{
						{
							Slices: []*ast.Slice{
								{
									Events: []*ast.Event{
										{Name: "OrderPlaced"},
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
									Flows: []*ast.Flow{
										{EventName: "OrderPlaced"},
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

	t.Run("multiple orphan events produce multiple orphan diagnostics", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Aggregates: []*ast.Aggregate{
						{
							Slices: []*ast.Slice{
								{
									Events: []*ast.Event{
										{Name: "OrderPlaced", NamePos: ast.Position{Filename: "test.emod", Line: 3, Column: 5}},
										{Name: "OrderShipped", NamePos: ast.Position{Filename: "test.emod", Line: 7, Column: 5}},
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
		require.Equal(t, "orphan-event", diags[0].RuleName)
		require.Contains(t, diags[0].Message, "OrderPlaced")
		require.Equal(t, "orphan-event", diags[1].RuleName)
		require.Contains(t, diags[1].Message, "OrderShipped")
	})

	// ── DCB: tag field reference validation ──────────────────────────────

	t.Run("tag entry with valid fieldRef produces no diagnostics", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Slices: []*ast.Slice{
						{
							Events: []*ast.Event{
								{
									Name: "OrderPlaced",
									Fields: []*ast.Field{
										{Name: "orderId"},
										{Name: "customerId"},
									},
									Tags: []ast.TagEntry{
										{Key: "aggregate_id", FieldRef: "orderId"},
									},
								},
							},
							Flows: []*ast.Flow{
								{EventName: "OrderPlaced"},
							},
						},
					},
				},
			},
		}

		diags := validator.Validate(model)

		require.Empty(t, diags)
	})

	t.Run("tag entry with invalid fieldRef produces diagnostic at FieldRefPos", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Slices: []*ast.Slice{
						{
							Events: []*ast.Event{
								{
									Name: "OrderPlaced",
									Source: "external",
									Fields: []*ast.Field{
										{Name: "orderId"},
									},
									Tags: []ast.TagEntry{
										{
											Key:     "aggregate_id",
											FieldRef: "nonExistentField",
											FieldRefPos: ast.Position{
												Filename: "test.emod",
												Line:     10,
												Column:   25,
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
		require.Equal(t, `tag field reference "nonExistentField" does not match any field on event "OrderPlaced"`, diags[0].Message)
		require.Equal(t, "test.emod", diags[0].Filename)
		require.Equal(t, 10, diags[0].Line)
		require.Equal(t, 25, diags[0].Column)
	})

	t.Run("multiple tag entries with invalid fieldRefs produce multiple diagnostics", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Slices: []*ast.Slice{
						{
							Events: []*ast.Event{
								{
									Name: "OrderPlaced",
									Source: "external",
									Fields: []*ast.Field{
										{Name: "orderId"},
									},
									Tags: []ast.TagEntry{
										{Key: "key1", FieldRef: "badField1"},
										{Key: "key2", FieldRef: "badField2"},
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
		require.Contains(t, diags[0].Message, "badField1")
		require.Contains(t, diags[1].Message, "badField2")
	})

	t.Run("tag entry fieldRef validated for inline translation event", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Bookings",
					Slices: []*ast.Slice{
						{
							Translations: []*ast.Translation{
								{
									Name: "ImportBooking",
									Event: &ast.Event{
										Name: "BookingImported",
										Fields: []*ast.Field{
											{Name: "bookingRef"},
										},
										Tags: []ast.TagEntry{
											{Key: "aggregate_id", FieldRef: "missingField"},
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
		require.Contains(t, diags[0].Message, "missingField")
		require.Contains(t, diags[0].Message, "BookingImported")
	})

	t.Run("tag entry with empty fieldRef is skipped", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Slices: []*ast.Slice{
						{
							Events: []*ast.Event{
								{
									Name: "OrderPlaced",
									Source: "external",
									Fields: []*ast.Field{
										{Name: "orderId"},
									},
									Tags: []ast.TagEntry{
										{Key: "aggregate_id", FieldRef: ""},
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

	// ── DCB: decides_on event name validation ────────────────────────────

	t.Run("decides_on with valid event names produces no diagnostics", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Slices: []*ast.Slice{
						{
							Events: []*ast.Event{
								{
									Name: "OrderPlaced",
									Source: "external",
									Fields: []*ast.Field{
										{Name: "orderId"},
									},
									Tags: []ast.TagEntry{
										{Key: "aggregate_id", FieldRef: "orderId"},
									},
								},
							},
							Commands: []*ast.Command{
								{
									Name: "PlaceOrder",
									DecidesOn: &ast.DecidesOnClause{
										Events:    []string{"OrderPlaced"},
										EventsPos: []ast.Position{{}},
										Predicate: &ast.TagPredicate{
											Field: "aggregate_id", FieldPos: ast.Position{},
											Operator: "=", OpPos: ast.Position{},
											Value: "orderId", ValuePos: ast.Position{},
										},
									},
								},
							},
							Flows: []*ast.Flow{
								{CommandName: "PlaceOrder", EventName: "OrderPlaced"},
							},
						},
					},
				},
			},
		}

		diags := validator.Validate(model)

		require.Empty(t, diags)
	})

	t.Run("decides_on with non-existent event name produces diagnostic", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Slices: []*ast.Slice{
						{
							Commands: []*ast.Command{
								{
									Name: "PlaceOrder",
									DecidesOn: &ast.DecidesOnClause{
										Events:    []string{"NonExistentEvent"},
										EventsPos: []ast.Position{
											{Filename: "test.emod", Line: 15, Column: 20},
										},
									},
								},
							},
							Flows: []*ast.Flow{
								{CommandName: "PlaceOrder"},
							},
						},
					},
				},
			},
		}

		diags := validator.Validate(model)

		require.Len(t, diags, 1)
		require.Equal(t, `event "NonExistentEvent" in decides_on does not exist`, diags[0].Message)
		require.Equal(t, "test.emod", diags[0].Filename)
		require.Equal(t, 15, diags[0].Line)
		require.Equal(t, 20, diags[0].Column)
	})

	t.Run("decides_on with multiple non-existent events produces diagnostics for each", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Slices: []*ast.Slice{
						{
							Commands: []*ast.Command{
								{
									Name: "PlaceOrder",
									DecidesOn: &ast.DecidesOnClause{
										Events:    []string{"MissingA", "MissingB"},
										EventsPos: []ast.Position{{Filename: "a.emod", Line: 1, Column: 1}, {Filename: "b.emod", Line: 2, Column: 2}},
									},
								},
							},
							Flows: []*ast.Flow{
								{CommandName: "PlaceOrder"},
							},
						},
					},
				},
			},
		}

		diags := validator.Validate(model)

		require.Len(t, diags, 2)
		require.Contains(t, diags[0].Message, "MissingA")
		require.Contains(t, diags[1].Message, "MissingB")
	})

	t.Run("command without decides_on produces no decisions diagnostics", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Slices: []*ast.Slice{
						{
							Commands: []*ast.Command{
								{Name: "PlaceOrder"},
							},
							Flows: []*ast.Flow{
								{CommandName: "PlaceOrder"},
							},
						},
					},
				},
			},
		}

		diags := validator.Validate(model)

		require.Empty(t, diags)
	})

	// ── DCB: predicate tag key and field reference validation ────────────

	t.Run("predicate with valid tag key and fieldRef produces no diagnostics", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Slices: []*ast.Slice{
						{
							Events: []*ast.Event{
								{
									Name: "OrderPlaced",
									Source: "external",
									Fields: []*ast.Field{
										{Name: "orderId"},
									},
									Tags: []ast.TagEntry{
										{Key: "aggregate_id", FieldRef: "orderId"},
									},
								},
							},
							Commands: []*ast.Command{
								{
									Name: "PlaceOrder",
									DecidesOn: &ast.DecidesOnClause{
										Events:    []string{"OrderPlaced"},
										EventsPos: []ast.Position{{}},
										Predicate: &ast.TagPredicate{
											Field:    "aggregate_id",
											FieldPos: ast.Position{},
											Operator: "=",
											OpPos:    ast.Position{},
											Value:    "orderId",
											ValuePos: ast.Position{},
										},
									},
								},
							},
							Flows: []*ast.Flow{
								{CommandName: "PlaceOrder", EventName: "OrderPlaced"},
							},
						},
					},
				},
			},
		}

		diags := validator.Validate(model)

		require.Empty(t, diags)
	})

	t.Run("predicate with tag key not declared on any listed event produces diagnostic", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Slices: []*ast.Slice{
						{
							Events: []*ast.Event{
								{
									Name: "OrderPlaced",
									Source: "external",
									Fields: []*ast.Field{
										{Name: "orderId"},
									},
									Tags: []ast.TagEntry{
										{Key: "different_key", FieldRef: "orderId"},
									},
								},
							},
							Commands: []*ast.Command{
								{
									Name: "PlaceOrder",
									DecidesOn: &ast.DecidesOnClause{
										Events:    []string{"OrderPlaced"},
										EventsPos: []ast.Position{{}},
										Predicate: &ast.TagPredicate{
											Field:    "aggregate_id",
											FieldPos: ast.Position{Filename: "pred.emod", Line: 5, Column: 10},
											Operator: "=",
											OpPos:    ast.Position{},
											Value:    "orderId",
											ValuePos: ast.Position{},
										},
									},
								},
							},
							Flows: []*ast.Flow{
								{CommandName: "PlaceOrder", EventName: "OrderPlaced"},
							},
						},
					},
				},
			},
		}

		diags := validator.Validate(model)

		require.Len(t, diags, 1)
		require.Equal(t, `tag key "aggregate_id" is not declared on any event in decides_on`, diags[0].Message)
		require.Equal(t, "pred.emod", diags[0].Filename)
		require.Equal(t, 5, diags[0].Line)
		require.Equal(t, 10, diags[0].Column)
	})

	t.Run("predicate with fieldRef not declared on any listed event produces diagnostic", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Slices: []*ast.Slice{
						{
							Events: []*ast.Event{
								{
									Name: "OrderPlaced",
									Source: "external",
									Fields: []*ast.Field{
										{Name: "orderId"},
									},
									Tags: []ast.TagEntry{
										{Key: "aggregate_id", FieldRef: "orderId"},
									},
								},
							},
							Commands: []*ast.Command{
								{
									Name: "PlaceOrder",
									DecidesOn: &ast.DecidesOnClause{
										Events:    []string{"OrderPlaced"},
										EventsPos: []ast.Position{{}},
										Predicate: &ast.TagPredicate{
											Field:    "aggregate_id",
											FieldPos: ast.Position{},
											Operator: "=",
											OpPos:    ast.Position{},
											Value:    "nonExistentField",
											ValuePos: ast.Position{Filename: "pred.emod", Line: 8, Column: 30},
										},
									},
								},
							},
							Flows: []*ast.Flow{
								{CommandName: "PlaceOrder", EventName: "OrderPlaced"},
							},
						},
					},
				},
			},
		}

		diags := validator.Validate(model)

		require.Len(t, diags, 1)
		require.Equal(t, `field reference "nonExistentField" is not declared on any event in decides_on`, diags[0].Message)
		require.Equal(t, "pred.emod", diags[0].Filename)
		require.Equal(t, 8, diags[0].Line)
		require.Equal(t, 30, diags[0].Column)
	})

	t.Run("predicate with tag key valid on any one of multiple listed events produces no diagnostic", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Slices: []*ast.Slice{
						{
							Events: []*ast.Event{
								{
									Name: "OrderPlaced",
									Source: "external",
									Fields: []*ast.Field{
										{Name: "orderId"},
									},
									Tags: []ast.TagEntry{
										{Key: "aggregate_id", FieldRef: "orderId"},
									},
								},
								{
									Name: "OrderShipped",
									Source: "external",
									Fields: []*ast.Field{
										{Name: "orderId"},
									},
									// No tags on this one
								},
							},
							Commands: []*ast.Command{
								{
									Name: "PlaceOrder",
									DecidesOn: &ast.DecidesOnClause{
										Events:    []string{"OrderPlaced", "OrderShipped"},
										EventsPos: []ast.Position{{}, {}},
										Predicate: &ast.TagPredicate{
											Field:    "aggregate_id",
											FieldPos: ast.Position{},
											Operator: "=",
											OpPos:    ast.Position{},
											Value:    "orderId",
											ValuePos: ast.Position{},
										},
									},
								},
							},
							Flows: []*ast.Flow{
								{CommandName: "PlaceOrder", EventName: "OrderPlaced"},
							},
						},
					},
				},
			},
		}

		diags := validator.Validate(model)

		require.Empty(t, diags)
	})

	t.Run("complex predicate with and/or/not validates all leaf TagPredicates", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Slices: []*ast.Slice{
						{
							Events: []*ast.Event{
								{
									Name: "OrderPlaced",
									Source: "external",
									Tags: []ast.TagEntry{
										{Key: "aggregate_id", FieldRef: "orderId"},
									},
									Fields: []*ast.Field{
										{Name: "orderId"},
									},
								},
							},
							Commands: []*ast.Command{
								{
									Name: "PlaceOrder",
									DecidesOn: &ast.DecidesOnClause{
										Events:    []string{"OrderPlaced"},
										EventsPos: []ast.Position{{}},
										Predicate: &ast.LogicalExpr{
											Left: &ast.TagPredicate{
												Field:    "aggregate_id",
												FieldPos: ast.Position{},
												Operator: "=",
												OpPos:    ast.Position{},
												Value:    "orderId",
												ValuePos: ast.Position{},
											},
											Operator: "and",
											OpPos:    ast.Position{},
											Right: &ast.NotExpr{
												OpPos: ast.Position{},
												Expr: &ast.TagPredicate{
													Field:    "aggregate_id",
													FieldPos: ast.Position{},
													Operator: "=",
													OpPos:    ast.Position{},
													Value:    "orderId",
													ValuePos: ast.Position{},
												},
											},
										},
									},
								},
							},
							Flows: []*ast.Flow{
								{CommandName: "PlaceOrder", EventName: "OrderPlaced"},
							},
						},
					},
				},
			},
		}

		diags := validator.Validate(model)

		require.Empty(t, diags)
	})

	t.Run("complex predicate with invalid field refs in both branches produces multiple diagnostics", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Slices: []*ast.Slice{
						{
							Events: []*ast.Event{
								{
									Name: "OrderPlaced",
									Source: "external",
									Tags: []ast.TagEntry{
										{Key: "aggregate_id", FieldRef: "orderId"},
									},
									Fields: []*ast.Field{
										{Name: "orderId"},
									},
								},
							},
							Commands: []*ast.Command{
								{
									Name: "PlaceOrder",
									DecidesOn: &ast.DecidesOnClause{
										Events:    []string{"OrderPlaced"},
										EventsPos: []ast.Position{{}},
										Predicate: &ast.LogicalExpr{
											Left: &ast.TagPredicate{
												Field:    "aggregate_id",
												FieldPos: ast.Position{},
												Operator: "=",
												OpPos:    ast.Position{},
												Value:    "badField1",
												ValuePos: ast.Position{Filename: "test.emod", Line: 1, Column: 1},
											},
											Operator: "or",
											OpPos:    ast.Position{},
											Right: &ast.TagPredicate{
												Field:    "aggregate_id",
												FieldPos: ast.Position{},
												Operator: "=",
												OpPos:    ast.Position{},
												Value:    "badField2",
												ValuePos: ast.Position{Filename: "test.emod", Line: 2, Column: 2},
											},
										},
									},
								},
							},
							Flows: []*ast.Flow{
								{CommandName: "PlaceOrder", EventName: "OrderPlaced"},
							},
						},
					},
				},
			},
		}

		diags := validator.Validate(model)

		require.Len(t, diags, 2)
		require.Contains(t, diags[0].Message, "badField1")
		require.Contains(t, diags[1].Message, "badField2")
	})

	// ── DCB: valid references across both slice locations ────────────────

	t.Run("ctx.Slices events with valid tags produce no diagnostics", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Slices: []*ast.Slice{
						{
							Events: []*ast.Event{
								{
									Name: "OrderPlaced",
									Source: "external",
									Fields: []*ast.Field{
										{Name: "orderId"},
									},
									Tags: []ast.TagEntry{
										{Key: "aggregate_id", FieldRef: "orderId"},
									},
								},
							},
							Commands: []*ast.Command{
								{
									Name: "PlaceOrder",
									DecidesOn: &ast.DecidesOnClause{
										Events:    []string{"OrderPlaced"},
										EventsPos: []ast.Position{{}},
										Predicate: &ast.TagPredicate{
											Field:    "aggregate_id",
											FieldPos: ast.Position{},
											Operator: "=",
											OpPos:    ast.Position{},
											Value:    "orderId",
											ValuePos: ast.Position{},
										},
									},
								},
							},
							Flows: []*ast.Flow{
								{CommandName: "PlaceOrder", EventName: "OrderPlaced"},
							},
						},
					},
				},
			},
		}

		diags := validator.Validate(model)

		require.Empty(t, diags)
	})

	t.Run("ctx.Aggregates[].Slices events with valid tags produce no diagnostics", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Aggregates: []*ast.Aggregate{
						{
							Slices: []*ast.Slice{
								{
									Events: []*ast.Event{
										{
											Name: "OrderPlaced",
											Source: "external",
											Fields: []*ast.Field{
												{Name: "orderId"},
											},
											Tags: []ast.TagEntry{
												{Key: "aggregate_id", FieldRef: "orderId"},
											},
										},
									},
									Commands: []*ast.Command{
										{
											Name: "PlaceOrder",
											DecidesOn: &ast.DecidesOnClause{
												Events:    []string{"OrderPlaced"},
												EventsPos: []ast.Position{{}},
												Predicate: &ast.TagPredicate{
													Field:    "aggregate_id",
													FieldPos: ast.Position{},
													Operator: "=",
													OpPos:    ast.Position{},
													Value:    "orderId",
													ValuePos: ast.Position{},
												},
											},
										},
									},
									Flows: []*ast.Flow{
										{CommandName: "PlaceOrder", EventName: "OrderPlaced"},
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

	t.Run("existing aggregate events with no tags produce no new diagnostics", func(t *testing.T) {
		model := &ast.Model{
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Aggregates: []*ast.Aggregate{
						{
							Slices: []*ast.Slice{
								{
									Events: []*ast.Event{
										{Name: "OrderPlaced", Source: "ExternalSystem"},
									},
									Flows: []*ast.Flow{
										{EventName: "OrderPlaced"},
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
}

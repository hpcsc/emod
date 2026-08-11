//go:build unit

package validator_test

import (
	"fmt"
	"testing"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/diagnostic"
	"github.com/hpcsc/emod/internal/lexer"
	"github.com/hpcsc/emod/internal/parser"
	"github.com/hpcsc/emod/internal/test"
	"github.com/hpcsc/emod/internal/validator"
	"github.com/stretchr/testify/require"
)

func TestValidate(t *testing.T) {
	t.Run("nil model produces no diagnostics", func(t *testing.T) {
		diags := validator.Validate(nil)

		require.Empty(t, diags)
	})

	t.Run("target context references", func(t *testing.T) {
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
	})

	t.Run("command references", func(t *testing.T) {
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
	})

	t.Run("event references", func(t *testing.T) {
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
												Name:    "NotifyOnOrder",
												OnEvent: "OrderPlaced",
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
												Name:    "ShipOrder",
												OnEvent: "OrderShipped",
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

		t.Run("automation trigger diagnostic includes position from OnEventPos", func(t *testing.T) {
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
												Name:    "ShipOrder",
												OnEvent: "OrderShipped",
												OnEventPos: ast.Position{
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
												Name:    "AutoConfirm",
												OnEvent: "",
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
												Name:    "ShipOnOrder",
												OnEvent: "OrderPlaced",
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
												Name:    "ProcessBooking",
												OnEvent: "BookingImported",
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
	})

	t.Run("view references", func(t *testing.T) {
		t.Run("a read resolves against a view declared in another context and in another aggregate", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Availability",
						Mode: "dcb",
						Slices: []*ast.Slice{
							{
								Name:  "Room Availability",
								Views: []*ast.View{{Name: "RoomAvailabilityView"}},
							},
						},
					},
					{
						Name: "Reservations",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Reservation",
								Slices: []*ast.Slice{
									{
										Name:  "Reservation History",
										Views: []*ast.View{{Name: "ReservationHistoryView"}},
									},
								},
							},
							{
								Name: "Confirmation",
								Slices: []*ast.Slice{
									{
										Name: "Confirm Reservation",
										Automations: []*ast.Automation{
											{Name: "ConfirmOnPayment", Reads: "RoomAvailabilityView"},
											{Name: "ExpireStaleHolds", Reads: "ReservationHistoryView"},
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

		t.Run("a view no slice declares is reported on the reads entry, not on the automation name", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Reservations",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Reservation",
								Slices: []*ast.Slice{
									{
										Name:  "Confirm Reservation",
										Views: []*ast.View{{Name: "ReservationsView"}},
										Automations: []*ast.Automation{
											{
												Name:     "AutoConfirm",
												NamePos:  ast.Position{Filename: "reservations.emod", Line: 9, Column: 18},
												Reads:    "PendingConfirmationsView",
												ReadsPos: ast.Position{Filename: "reservations.emod", Line: 11, Column: 15},
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
			require.Equal(t, `reservations.emod:11: view "PendingConfirmationsView" does not exist`, diags[0].String())
			require.Equal(t, 15, diags[0].Column)
			require.Equal(t, diagnostic.Error, diags[0].Severity)
			require.Empty(t, diags[0].RuleName)
		})

		t.Run("an automation on a context's own slice is checked like one inside an aggregate", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Availability",
						Mode: "dcb",
						Slices: []*ast.Slice{
							{
								Name: "Release Held Rooms",
								Automations: []*ast.Automation{
									{
										Name:     "ReleaseHeldRooms",
										Reads:    "ExpiringHoldsView",
										ReadsPos: ast.Position{Filename: "availability.emod", Line: 7, Column: 15},
									},
								},
							},
						},
					},
					{
						Name: "Reservations",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Reservation",
								Slices: []*ast.Slice{
									{
										Name:  "View Reservations",
										Views: []*ast.View{{Name: "ReservationsView"}},
									},
								},
							},
						},
					},
				},
			}

			diags := validator.Validate(model)

			require.Len(t, diags, 1)
			require.Equal(t, `availability.emod:7: view "ExpiringHoldsView" does not exist`, diags[0].String())
		})

		t.Run("an automation without a reads entry produces no diagnostic while its sibling reading an undeclared view is reported", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Reservations",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Reservation",
								Slices: []*ast.Slice{
									{
										Name: "Confirm Reservation",
										Automations: []*ast.Automation{
											{
												Name:    "AutoConfirm",
												NamePos: ast.Position{Filename: "reservations.emod", Line: 6, Column: 18},
											},
											{
												Name:     "ExpireStaleHolds",
												NamePos:  ast.Position{Filename: "reservations.emod", Line: 10, Column: 18},
												Reads:    "ExpiringHoldsView",
												ReadsPos: ast.Position{Filename: "reservations.emod", Line: 11, Column: 15},
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
			require.Equal(t, `reservations.emod:11: view "ExpiringHoldsView" does not exist`, diags[0].String())
		})

		t.Run("a name declared only as an event does not resolve as a view", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Reservations",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Reservation",
								Slices: []*ast.Slice{
									{
										Name:     "Confirm Reservation",
										Commands: []*ast.Command{{Name: "ConfirmReservation"}},
										Events:   []*ast.Event{{Name: "ReservationMade"}},
										Flows:    []*ast.Flow{{CommandName: "ConfirmReservation", EventName: "ReservationMade"}},
										Automations: []*ast.Automation{
											{
												Name:     "AutoConfirm",
												Reads:    "ReservationMade",
												ReadsPos: ast.Position{Filename: "reservations.emod", Line: 14, Column: 15},
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
			require.Equal(t, `reservations.emod:14: view "ReservationMade" does not exist`, diags[0].String())
		})
	})

	t.Run("schedule expressions", func(t *testing.T) {
		const companionRejected = `reservations.emod:12: schedule expression "nightly" is neither a Go duration nor a five-field cron expression`

		t.Run("a fixed interval stated as a Go duration is accepted", func(t *testing.T) {
			for _, expression := range []string{"5m", "1h", "1h30m"} {
				t.Run(expression, func(t *testing.T) {
					require.Empty(t, validator.Validate(modelSchedulingEvery(expression)))
					require.Equal(t, []string{companionRejected},
						reportedLines(validator.Validate(modelSchedulingEvery(expression, "nightly"))))
				})
			}
		})

		t.Run("a wall-clock schedule stated as a five-field cron expression is accepted", func(t *testing.T) {
			for _, expression := range []string{"0 2 * * *", "*/15 * * * *", "0 0 1,15 * 1-5", "0 9 * * MON-FRI", "0 0 1 jan *"} {
				t.Run(expression, func(t *testing.T) {
					require.Empty(t, validator.Validate(modelSchedulingEvery(expression)))
					require.Equal(t, []string{companionRejected},
						reportedLines(validator.Validate(modelSchedulingEvery(expression, "nightly"))))
				})
			}
		})

		t.Run("a cron field outside the range its position allows is left to the scheduler", func(t *testing.T) {
			require.Empty(t, validator.Validate(modelSchedulingEvery("99 * * * *")))
			require.Equal(t, []string{companionRejected},
				reportedLines(validator.Validate(modelSchedulingEvery("99 * * * *", "nightly"))))
		})

		t.Run("an expression of neither form is reported on the every entry, not on the automation name", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Reservations",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Reservation",
								Slices: []*ast.Slice{
									{
										Name: "Expire Stale Holds",
										Automations: []*ast.Automation{
											{
												Name:        "StaleHoldExpirer",
												NamePos:     ast.Position{Filename: "reservations.emod", Line: 6, Column: 18},
												Schedule:    "nightly",
												SchedulePos: ast.Position{Filename: "reservations.emod", Line: 7, Column: 15},
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

			require.Equal(t, []string{
				`reservations.emod:7: schedule expression "nightly" is neither a Go duration nor a five-field cron expression`,
			}, reportedLines(diags))
			require.Equal(t, 15, diags[0].Column)
			require.Equal(t, diagnostic.Error, diags[0].Severity)
			require.Empty(t, diags[0].RuleName)
		})

		t.Run("a cron expression of any length other than five fields is reported", func(t *testing.T) {
			for _, tc := range []struct {
				expression string
				reported   string
			}{
				{
					expression: "0 2 * *",
					reported:   `reservations.emod:7: schedule expression "0 2 * *" is neither a Go duration nor a five-field cron expression`,
				},
				{
					expression: "0 2 * * * *",
					reported:   `reservations.emod:7: schedule expression "0 2 * * * *" is neither a Go duration nor a five-field cron expression`,
				},
			} {
				t.Run(tc.expression, func(t *testing.T) {
					require.Equal(t, []string{tc.reported},
						reportedLines(validator.Validate(modelSchedulingEvery(tc.expression))))
				})
			}
		})

		t.Run("an expression of neither form is reported while the duration beside it is not", func(t *testing.T) {
			model := modelSchedulingEvery("5 minutes", "5m")

			require.Equal(t, []string{
				`reservations.emod:7: schedule expression "5 minutes" is neither a Go duration nor a five-field cron expression`,
			}, reportedLines(validator.Validate(model)))
		})

		t.Run("an automation activated by an event states no expression to reject", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Reservations",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Reservation",
								Slices: []*ast.Slice{
									{
										Name:   "Expire Stale Holds",
										Events: []*ast.Event{{Name: "HoldPlaced", Source: "external"}},
										Automations: []*ast.Automation{
											{
												Name:       "HoldConfirmer",
												OnEvent:    "HoldPlaced",
												OnEventPos: ast.Position{Filename: "reservations.emod", Line: 7, Column: 12},
											},
											{
												Name:        "StaleHoldExpirer",
												Schedule:    "nightly",
												SchedulePos: ast.Position{Filename: "reservations.emod", Line: 12, Column: 15},
											},
										},
									},
								},
							},
						},
					},
				},
			}

			require.Equal(t, []string{companionRejected}, reportedLines(validator.Validate(model)))
		})

		t.Run("expressions in both slice homes are reported in declaration order", func(t *testing.T) {
			// The aggregate's slice is declared before the context's own, so a
			// walk that visits either collection first comes out reversed here.
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Availability",
						Mode: "dcb",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Hold",
								Slices: []*ast.Slice{
									{
										Name:    "Expire Stale Holds",
										NamePos: ast.Position{Filename: "availability.emod", Line: 10, Column: 9},
										Automations: []*ast.Automation{
											{
												Name:        "StaleHoldExpirer",
												Schedule:    "nightly",
												SchedulePos: ast.Position{Filename: "availability.emod", Line: 12, Column: 15},
											},
										},
									},
								},
							},
						},
						Slices: []*ast.Slice{
							{
								Name:    "Release Held Rooms",
								NamePos: ast.Position{Filename: "availability.emod", Line: 28, Column: 9},
								Automations: []*ast.Automation{
									{
										Name:        "HeldRoomReleaser",
										Schedule:    "0 2 * *",
										SchedulePos: ast.Position{Filename: "availability.emod", Line: 30, Column: 15},
									},
								},
							},
						},
					},
				},
			}

			require.Equal(t, []string{
				`availability.emod:12: schedule expression "nightly" is neither a Go duration nor a five-field cron expression`,
				`availability.emod:30: schedule expression "0 2 * *" is neither a Go duration nor a five-field cron expression`,
			}, reportedLines(validator.Validate(model)))
		})
	})

	t.Run("spec references", func(t *testing.T) {
		t.Run("an event a given names but no construct declares is reported on that reference", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Lending",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Loan",
								Slices: []*ast.Slice{
									{
										Name:     "Borrow Copy",
										Commands: []*ast.Command{{Name: "BorrowCopy"}},
										Events:   []*ast.Event{{Name: "CopyBorrowed"}},
										Flows:    []*ast.Flow{{CommandName: "BorrowCopy", EventName: "CopyBorrowed"}},
										Specs: []*ast.Spec{
											{
												Name: "borrows a copy the member returned",
												Given: []*ast.SpecElement{
													{Name: "CopyRetruned", NamePos: ast.Position{Filename: "lending.emod", Line: 12, Column: 16}},
												},
												When: &ast.SpecElement{Name: "BorrowCopy", NamePos: ast.Position{Filename: "lending.emod", Line: 13, Column: 14}},
												Then: &ast.ThenEvents{Events: []*ast.SpecElement{
													{Name: "CopyBorrowed", NamePos: ast.Position{Filename: "lending.emod", Line: 14, Column: 15}},
												}},
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

			require.Equal(t, []*diagnostic.Entry{
				{
					Filename: "lending.emod",
					Line:     12,
					Column:   16,
					Severity: diagnostic.Error,
					Message:  `event "CopyRetruned" does not exist`,
				},
			}, diags)
		})

		t.Run("an event a then names but no construct declares is reported on that reference", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Lending",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Loan",
								Slices: []*ast.Slice{
									{
										Name:     "Borrow Copy",
										Commands: []*ast.Command{{Name: "BorrowCopy"}},
										Events:   []*ast.Event{{Name: "CopyBorrowed"}},
										Flows:    []*ast.Flow{{CommandName: "BorrowCopy", EventName: "CopyBorrowed"}},
										Specs: []*ast.Spec{
											{
												Name: "borrows a copy no one holds",
												When: &ast.SpecElement{Name: "BorrowCopy", NamePos: ast.Position{Filename: "lending.emod", Line: 12, Column: 14}},
												Then: &ast.ThenEvents{Events: []*ast.SpecElement{
													{Name: "CopyBorrwed", NamePos: ast.Position{Filename: "lending.emod", Line: 13, Column: 15}},
												}},
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
			require.Equal(t, `lending.emod:13: event "CopyBorrwed" does not exist`, diags[0].String())
		})

		t.Run("a command a when names but no construct declares is reported as a missing command", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Lending",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Loan",
								Slices: []*ast.Slice{
									{
										Name:     "Borrow Copy",
										Commands: []*ast.Command{{Name: "BorrowCopy"}},
										Events:   []*ast.Event{{Name: "CopyBorrowed"}},
										Flows:    []*ast.Flow{{CommandName: "BorrowCopy", EventName: "CopyBorrowed"}},
										Specs: []*ast.Spec{
											{
												Name: "borrows a copy no one holds",
												When: &ast.SpecElement{Name: "BorowCopy", NamePos: ast.Position{Filename: "lending.emod", Line: 12, Column: 14}},
												Then: &ast.ThenEvents{Events: []*ast.SpecElement{
													{Name: "CopyBorrowed", NamePos: ast.Position{Filename: "lending.emod", Line: 13, Column: 15}},
												}},
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
			require.Equal(t, `lending.emod:12: command "BorowCopy" does not exist`, diags[0].String())
		})

		t.Run("a when naming an event the model declares is reported on nothing", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Reservations",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Reservation",
								Slices: []*ast.Slice{
									{
										Name:     "Reserve a Room",
										Commands: []*ast.Command{{Name: "ReserveRoom"}},
										Events:   []*ast.Event{{Name: "RoomReserved"}},
										Flows:    []*ast.Flow{{CommandName: "ReserveRoom", EventName: "RoomReserved"}},
									},
									{
										Name:     "Send Confirmation Email",
										Commands: []*ast.Command{{Name: "SendConfirmationEmail"}},
										Automations: []*ast.Automation{
											{Name: "ConfirmationMailer", OnEvent: "RoomReserved", Command: "SendConfirmationEmail"},
										},
										Specs: []*ast.Spec{
											{
												Name: "emails the guest once the room is reserved",
												When: &ast.SpecElement{Name: "RoomReserved", NamePos: ast.Position{Filename: "hotel.emod", Line: 18, Column: 12}},
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

		t.Run("references resolve against declarations anywhere in the model", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Lending",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Loan",
								Slices: []*ast.Slice{
									{
										Name:     "Borrow Copy",
										Commands: []*ast.Command{{Name: "BorrowCopy"}},
										Events:   []*ast.Event{{Name: "CopyBorrowed"}},
										Flows:    []*ast.Flow{{CommandName: "BorrowCopy", EventName: "CopyBorrowed"}},
										Specs: []*ast.Spec{
											{
												Name: "borrows an imported copy the member returned",
												Given: []*ast.SpecElement{
													{Name: "CopyImported"},
													{Name: "CopyReturned"},
												},
												When: &ast.SpecElement{Name: "BorrowCopy"},
												Then: &ast.ThenEvents{Events: []*ast.SpecElement{{Name: "CopyBorrowed"}}},
											},
										},
									},
								},
							},
						},
					},
					{
						Name: "Circulation",
						Slices: []*ast.Slice{
							{
								Name: "Circulate Copies",
								Commands: []*ast.Command{
									{Name: "BorrowCopy"},
									{Name: "ReturnCopy"},
								},
								Events: []*ast.Event{
									{Name: "CopyBorrowed"},
									{Name: "CopyReturned"},
								},
								Flows: []*ast.Flow{
									{CommandName: "BorrowCopy", EventName: "CopyBorrowed"},
									{CommandName: "ReturnCopy", EventName: "CopyReturned"},
								},
							},
						},
					},
					{
						Name: "Catalog",
						Slices: []*ast.Slice{
							{
								Name: "Import Catalog",
								Translations: []*ast.Translation{
									{Name: "ImportCopy", Event: &ast.Event{Name: "CopyImported"}},
								},
							},
						},
					},
				},
			}

			diags := validator.Validate(model)

			require.Empty(t, diags)
		})

		t.Run("a spec part naming nothing to resolve is reported on nothing", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name:       "Reading Room",
						Mode:       "dcb",
						Invariants: []*ast.Invariant{{Name: "OneReaderPerDesk"}},
						Slices: []*ast.Slice{
							{
								Name:     "Claim Desk",
								Commands: []*ast.Command{{Name: "ClaimDesk"}},
								Events:   []*ast.Event{{Name: "DeskClaimed"}},
								Flows:    []*ast.Flow{{CommandName: "ClaimDesk", EventName: "DeskClaimed"}},
								Specs: []*ast.Spec{
									{
										Name:  "seats a reader at a free desk",
										Given: []*ast.SpecElement{},
									},
									{
										Name:  "refuses a desk another reader is seated at",
										Given: []*ast.SpecElement{{Name: "DeskClaimed"}},
										When:  &ast.SpecElement{Name: "ClaimDesk"},
										Then:  &ast.ThenRejected{InvariantName: "OneReaderPerDesk"},
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

		t.Run("undefined references are reported once each in declaration order on every run", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Lending",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Loan",
								Slices: []*ast.Slice{
									{
										Name:     "Borrow Copy",
										Commands: []*ast.Command{{Name: "BorrowCopy"}},
										Events:   []*ast.Event{{Name: "CopyBorrowed"}},
										Flows:    []*ast.Flow{{CommandName: "BorrowCopy", EventName: "CopyBorrowed"}},
										Specs: []*ast.Spec{
											{
												Name: "borrows a copy the member returned",
												Given: []*ast.SpecElement{
													{Name: "CopyRetruned", NamePos: ast.Position{Filename: "lending.emod", Line: 10, Column: 16}},
												},
												When: &ast.SpecElement{Name: "BorowCopy", NamePos: ast.Position{Filename: "lending.emod", Line: 11, Column: 14}},
												Then: &ast.ThenEvents{Events: []*ast.SpecElement{
													{Name: "CopyBorrwed", NamePos: ast.Position{Filename: "lending.emod", Line: 12, Column: 15}},
												}},
											},
										},
									},
									{
										Name:     "Return Copy",
										Commands: []*ast.Command{{Name: "ReturnCopy"}},
										Events:   []*ast.Event{{Name: "CopyReturned"}},
										Flows:    []*ast.Flow{{CommandName: "ReturnCopy", EventName: "CopyReturned"}},
										Specs: []*ast.Spec{
											// This spec states its outcome before the command that
											// produces it, the order internal/test's lending model
											// uses, so the report cannot follow the order the AST
											// happens to hold the parts in.
											{
												Name: "returns a copy the member holds",
												Given: []*ast.SpecElement{
													{Name: "CopyBorrowed", NamePos: ast.Position{Filename: "lending.emod", Line: 24, Column: 16}},
												},
												Then: &ast.ThenEvents{Events: []*ast.SpecElement{
													{Name: "CopyRetrned", NamePos: ast.Position{Filename: "lending.emod", Line: 25, Column: 15}},
												}},
												When: &ast.SpecElement{Name: "ReturnCpy", NamePos: ast.Position{Filename: "lending.emod", Line: 26, Column: 14}},
											},
										},
									},
								},
							},
						},
					},
				},
			}

			runs := make([][]string, 0, 3)
			for range 3 {
				runs = append(runs, reportedLines(validator.Validate(model)))
			}

			require.Equal(t, []string{
				`lending.emod:10: event "CopyRetruned" does not exist`,
				`lending.emod:11: command "BorowCopy" does not exist`,
				`lending.emod:12: event "CopyBorrwed" does not exist`,
				`lending.emod:25: event "CopyRetrned" does not exist`,
				`lending.emod:26: command "ReturnCpy" does not exist`,
			}, runs[0])
			require.Equal(t, runs[0], runs[1])
			require.Equal(t, runs[0], runs[2])
		})

		t.Run("rejected invariant resolution", func(t *testing.T) {
			t.Run("a rejection resolves against the invariants of the scope that owns the slice", func(t *testing.T) {
				model := &ast.Model{
					Contexts: []*ast.Context{
						{
							Name: "Lending",
							Aggregates: []*ast.Aggregate{
								{
									Name:       "Loan",
									Invariants: []*ast.Invariant{{Name: "OneCopyPerLoan"}},
									Slices: []*ast.Slice{
										{
											Name:     "Borrow Copy",
											Commands: []*ast.Command{{Name: "BorrowCopy"}},
											Events:   []*ast.Event{{Name: "CopyBorrowed"}},
											Flows:    []*ast.Flow{{CommandName: "BorrowCopy", EventName: "CopyBorrowed"}},
											Specs: []*ast.Spec{
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
						{
							Name:       "Reading Room",
							Mode:       "dcb",
							Invariants: []*ast.Invariant{{Name: "OneReaderPerDesk"}},
							Slices: []*ast.Slice{
								{
									Name:     "Claim Desk",
									Commands: []*ast.Command{{Name: "ClaimDesk"}},
									Events:   []*ast.Event{{Name: "DeskClaimed"}},
									Flows:    []*ast.Flow{{CommandName: "ClaimDesk", EventName: "DeskClaimed"}},
									Specs: []*ast.Spec{
										{
											Name:  "refuses a desk another reader is seated at",
											Given: []*ast.SpecElement{{Name: "DeskClaimed"}},
											When:  &ast.SpecElement{Name: "ClaimDesk"},
											Then:  &ast.ThenRejected{InvariantName: "OneReaderPerDesk"},
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

			t.Run("a rejection naming an invariant no scope declares is reported on the reference", func(t *testing.T) {
				model := &ast.Model{
					Contexts: []*ast.Context{
						{
							Name: "Lending",
							Aggregates: []*ast.Aggregate{
								{
									Name:       "Loan",
									Invariants: []*ast.Invariant{{Name: "OneCopyPerLoan"}},
									Slices: []*ast.Slice{
										{
											Name:     "Borrow Copy",
											Commands: []*ast.Command{{Name: "BorrowCopy"}},
											Events:   []*ast.Event{{Name: "CopyBorrowed"}},
											Flows:    []*ast.Flow{{CommandName: "BorrowCopy", EventName: "CopyBorrowed"}},
											Specs: []*ast.Spec{
												{
													Name:  "refuses a copy already on loan",
													Given: []*ast.SpecElement{{Name: "CopyBorrowed"}},
													When:  &ast.SpecElement{Name: "BorrowCopy"},
													Then: &ast.ThenRejected{
														InvariantName: "OneCopyPerLon",
														InvariantPos:  ast.Position{Filename: "lending.emod", Line: 18, Column: 21},
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

				require.Equal(t, []*diagnostic.Entry{
					{
						Filename: "lending.emod",
						Line:     18,
						Column:   21,
						Severity: diagnostic.Error,
						Message:  `invariant "OneCopyPerLon" is not declared in aggregate "Loan"`,
					},
				}, diags)
			})

			t.Run("an invariant declared only on a sibling aggregate does not resolve", func(t *testing.T) {
				model := &ast.Model{
					Contexts: []*ast.Context{
						{
							Name: "Lending",
							Aggregates: []*ast.Aggregate{
								{
									Name: "Loan",
									Slices: []*ast.Slice{
										{
											Name:     "Borrow Copy",
											Commands: []*ast.Command{{Name: "BorrowCopy"}},
											Events:   []*ast.Event{{Name: "CopyBorrowed"}},
											Flows:    []*ast.Flow{{CommandName: "BorrowCopy", EventName: "CopyBorrowed"}},
											Specs: []*ast.Spec{
												{
													Name:  "refuses a copy already on loan",
													Given: []*ast.SpecElement{{Name: "CopyBorrowed"}},
													When:  &ast.SpecElement{Name: "BorrowCopy"},
													Then: &ast.ThenRejected{
														InvariantName: "OneHoldPerCopy",
														InvariantPos:  ast.Position{Filename: "lending.emod", Line: 15, Column: 21},
													},
												},
											},
										},
									},
								},
								{
									Name:       "Reservation",
									Invariants: []*ast.Invariant{{Name: "OneHoldPerCopy"}},
									Slices: []*ast.Slice{
										{
											Name:     "Hold Copy",
											Commands: []*ast.Command{{Name: "HoldCopy"}},
											Events:   []*ast.Event{{Name: "CopyHeld"}},
											Flows:    []*ast.Flow{{CommandName: "HoldCopy", EventName: "CopyHeld"}},
										},
									},
								},
							},
						},
					},
				}

				diags := validator.Validate(model)

				require.Len(t, diags, 1)
				require.Equal(t, `lending.emod:15: invariant "OneHoldPerCopy" is not declared in aggregate "Loan"`, diags[0].String())
			})

			t.Run("an aggregate and its enclosing context are separate scopes in both directions", func(t *testing.T) {
				model := &ast.Model{
					Contexts: []*ast.Context{
						{
							Name:       "Lending",
							Invariants: []*ast.Invariant{{Name: "FiveCopiesPerMember"}},
							Aggregates: []*ast.Aggregate{
								{
									Name:       "Loan",
									Invariants: []*ast.Invariant{{Name: "OneCopyPerLoan"}},
									Slices: []*ast.Slice{
										{
											Name:     "Borrow Copy",
											Commands: []*ast.Command{{Name: "BorrowCopy"}},
											Events:   []*ast.Event{{Name: "CopyBorrowed"}},
											Flows:    []*ast.Flow{{CommandName: "BorrowCopy", EventName: "CopyBorrowed"}},
											Specs: []*ast.Spec{
												{
													Name:  "refuses a member holding five copies",
													Given: []*ast.SpecElement{{Name: "CopyBorrowed"}},
													When:  &ast.SpecElement{Name: "BorrowCopy"},
													Then: &ast.ThenRejected{
														InvariantName: "FiveCopiesPerMember",
														InvariantPos:  ast.Position{Filename: "lending.emod", Line: 16, Column: 21},
													},
												},
											},
										},
									},
								},
							},
							Slices: []*ast.Slice{
								{
									Name:     "Waive Fine",
									Commands: []*ast.Command{{Name: "WaiveFine"}},
									Events:   []*ast.Event{{Name: "FineWaived"}},
									Flows:    []*ast.Flow{{CommandName: "WaiveFine", EventName: "FineWaived"}},
									Specs: []*ast.Spec{
										{
											Name: "refuses a waiver on a copy still on loan",
											When: &ast.SpecElement{Name: "WaiveFine"},
											Then: &ast.ThenRejected{
												InvariantName: "OneCopyPerLoan",
												InvariantPos:  ast.Position{Filename: "lending.emod", Line: 30, Column: 19},
											},
										},
									},
								},
							},
						},
					},
				}

				diags := validator.Validate(model)

				require.Equal(t, []string{
					`lending.emod:16: invariant "FiveCopiesPerMember" is not declared in aggregate "Loan"`,
					`lending.emod:30: invariant "OneCopyPerLoan" is not declared in context "Lending"`,
				}, reportedLines(diags))
			})

			t.Run("two scopes declaring one invariant name each resolve their own rejections", func(t *testing.T) {
				model := &ast.Model{
					Contexts: []*ast.Context{
						{
							Name: "Lending",
							Aggregates: []*ast.Aggregate{
								{
									Name:       "Loan",
									Invariants: []*ast.Invariant{{Name: "OneAtATime"}},
									Slices: []*ast.Slice{
										{
											Name:     "Borrow Copy",
											Commands: []*ast.Command{{Name: "BorrowCopy"}},
											Events:   []*ast.Event{{Name: "CopyBorrowed"}},
											Flows:    []*ast.Flow{{CommandName: "BorrowCopy", EventName: "CopyBorrowed"}},
											Specs: []*ast.Spec{
												{
													Name:  "refuses a copy already on loan",
													Given: []*ast.SpecElement{{Name: "CopyBorrowed"}},
													When:  &ast.SpecElement{Name: "BorrowCopy"},
													Then:  &ast.ThenRejected{InvariantName: "OneAtATime"},
												},
											},
										},
									},
								},
								{
									Name:       "Reservation",
									Invariants: []*ast.Invariant{{Name: "OneAtATime"}},
									Slices: []*ast.Slice{
										{
											Name:     "Hold Copy",
											Commands: []*ast.Command{{Name: "HoldCopy"}},
											Events:   []*ast.Event{{Name: "CopyHeld"}},
											Flows:    []*ast.Flow{{CommandName: "HoldCopy", EventName: "CopyHeld"}},
											Specs: []*ast.Spec{
												{
													Name:  "refuses a second hold on one copy",
													Given: []*ast.SpecElement{{Name: "CopyHeld"}},
													When:  &ast.SpecElement{Name: "HoldCopy"},
													Then:  &ast.ThenRejected{InvariantName: "OneAtATime"},
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

			t.Run("unresolved rejections follow declaration order across both slice homes", func(t *testing.T) {
				// "Lending" states one of its own slices ahead of its aggregate's and
				// one after, so neither walking a context ahead of its aggregates nor
				// the reverse yields source order.
				model := &ast.Model{
					Contexts: []*ast.Context{
						{
							Name:       "Lending",
							Invariants: []*ast.Invariant{{Name: "FiveCopiesPerMember"}},
							Aggregates: []*ast.Aggregate{
								{
									Name:       "Loan",
									Invariants: []*ast.Invariant{{Name: "OneCopyPerLoan"}},
									Slices: []*ast.Slice{
										{
											Name:     "Borrow Copy",
											Commands: []*ast.Command{{Name: "BorrowCopy"}},
											Events:   []*ast.Event{{Name: "CopyBorrowed"}},
											Flows:    []*ast.Flow{{CommandName: "BorrowCopy", EventName: "CopyBorrowed"}},
											Specs: []*ast.Spec{
												{
													Name: "refuses a copy already on loan",
													When: &ast.SpecElement{Name: "BorrowCopy"},
													Then: &ast.ThenRejected{
														InvariantName: "OneCopyPerLon",
														InvariantPos:  ast.Position{Filename: "shelf.emod", Line: 40, Column: 21},
													},
												},
											},
										},
									},
								},
							},
							Slices: []*ast.Slice{
								{
									Name:     "Waive Fine",
									Commands: []*ast.Command{{Name: "WaiveFine"}},
									Events:   []*ast.Event{{Name: "FineWaived"}},
									Flows:    []*ast.Flow{{CommandName: "WaiveFine", EventName: "FineWaived"}},
									Specs: []*ast.Spec{
										{
											Name: "refuses a waiver above the member's allowance",
											When: &ast.SpecElement{Name: "WaiveFine"},
											Then: &ast.ThenRejected{
												InvariantName: "FiveCopiesPerMemer",
												InvariantPos:  ast.Position{Filename: "shelf.emod", Line: 14, Column: 19},
											},
										},
									},
								},
								{
									Name:     "Renew Loan",
									Commands: []*ast.Command{{Name: "RenewLoan"}},
									Events:   []*ast.Event{{Name: "LoanRenewed"}},
									Flows:    []*ast.Flow{{CommandName: "RenewLoan", EventName: "LoanRenewed"}},
									Specs: []*ast.Spec{
										{
											Name: "refuses a renewal another member is waiting on",
											When: &ast.SpecElement{Name: "RenewLoan"},
											Then: &ast.ThenRejected{
												InvariantName: "NoQueueSkipping",
												InvariantPos:  ast.Position{Filename: "shelf.emod", Line: 70, Column: 19},
											},
										},
									},
								},
							},
						},
					},
				}

				require.Equal(t, []string{
					`shelf.emod:14: invariant "FiveCopiesPerMemer" is not declared in context "Lending"`,
					`shelf.emod:40: invariant "OneCopyPerLon" is not declared in aggregate "Loan"`,
					`shelf.emod:70: invariant "NoQueueSkipping" is not declared in context "Lending"`,
				}, reportedLines(validator.Validate(model)))
			})

			t.Run("a rejection edge on the timeline resolves exactly as a spec's rejection does", func(t *testing.T) {
				t.Run("an edge naming an invariant its aggregate does not declare reports the line a spec would", func(t *testing.T) {
					edgeModel := rejectionScopeModel(&ast.Slice{
						Name:     "Borrow Copy",
						Commands: []*ast.Command{{Name: "BorrowCopy"}},
						Events:   []*ast.Event{{Name: "CopyBorrowed"}},
						Flows:    []*ast.Flow{{CommandName: "BorrowCopy", EventName: "CopyBorrowed"}},
						Rejections: []*ast.Rejection{
							{
								CommandName:   "BorrowCopy",
								InvariantName: "OneHoldPerCopy",
								InvariantPos:  ast.Position{Filename: "lending.emod", Line: 15, Column: 43},
							},
						},
					})
					specModel := rejectionScopeModel(&ast.Slice{
						Name:     "Borrow Copy",
						Commands: []*ast.Command{{Name: "BorrowCopy"}},
						Events:   []*ast.Event{{Name: "CopyBorrowed"}},
						Flows:    []*ast.Flow{{CommandName: "BorrowCopy", EventName: "CopyBorrowed"}},
						Specs: []*ast.Spec{
							{
								Name: "refuses a copy already held",
								When: &ast.SpecElement{Name: "BorrowCopy"},
								Then: &ast.ThenRejected{
									InvariantName: "OneHoldPerCopy",
									InvariantPos:  ast.Position{Filename: "lending.emod", Line: 15, Column: 43},
								},
							},
						},
					})

					diags := validator.Validate(edgeModel)

					require.Len(t, diags, 1)
					require.Equal(t, diagnostic.Error, diags[0].Severity)
					require.Empty(t, diags[0].RuleName, "nothing configures this check, so lint --explain must not claim to describe it")
					require.Equal(t, reportedLines(validator.Validate(specModel)), reportedLines(diags))
				})

				t.Run("an edge on a dcb context's own slice resolves against that context's invariants", func(t *testing.T) {
					model := &ast.Model{
						Contexts: []*ast.Context{
							{
								Name:       "Reading Room",
								Mode:       "dcb",
								Invariants: []*ast.Invariant{{Name: "OneReaderPerDesk"}},
								Slices: []*ast.Slice{
									{
										Name:     "Claim Desk",
										Commands: []*ast.Command{{Name: "ClaimDesk"}},
										Events:   []*ast.Event{{Name: "DeskClaimed"}},
										Flows:    []*ast.Flow{{CommandName: "ClaimDesk", EventName: "DeskClaimed"}},
										Rejections: []*ast.Rejection{
											{CommandName: "ClaimDesk", InvariantName: "OneReaderPerDesk"},
											{
												CommandName:   "ClaimDesk",
												InvariantName: "OneDeskPerReader",
												InvariantPos:  ast.Position{Filename: "room.emod", Line: 22, Column: 41},
											},
										},
									},
								},
							},
						},
					}

					require.Equal(t, []string{
						`room.emod:22: invariant "OneDeskPerReader" is not declared in context "Reading Room"`,
					}, reportedLines(validator.Validate(model)))
				})

				t.Run("an edge inherits no scope in either direction", func(t *testing.T) {
					model := &ast.Model{
						Contexts: []*ast.Context{
							{
								Name:       "Lending",
								Invariants: []*ast.Invariant{{Name: "FiveCopiesPerMember"}},
								Aggregates: []*ast.Aggregate{
									{
										Name:       "Loan",
										Invariants: []*ast.Invariant{{Name: "OneCopyPerLoan"}},
										Slices: []*ast.Slice{
											{
												Name:     "Borrow Copy",
												Commands: []*ast.Command{{Name: "BorrowCopy"}},
												Events:   []*ast.Event{{Name: "CopyBorrowed"}},
												Flows:    []*ast.Flow{{CommandName: "BorrowCopy", EventName: "CopyBorrowed"}},
												Rejections: []*ast.Rejection{
													{
														CommandName:   "BorrowCopy",
														InvariantName: "FiveCopiesPerMember",
														InvariantPos:  ast.Position{Filename: "lending.emod", Line: 16, Column: 43},
													},
												},
											},
										},
									},
								},
								Slices: []*ast.Slice{
									{
										Name:     "Waive Fine",
										Commands: []*ast.Command{{Name: "WaiveFine"}},
										Events:   []*ast.Event{{Name: "FineWaived"}},
										Flows:    []*ast.Flow{{CommandName: "WaiveFine", EventName: "FineWaived"}},
										Rejections: []*ast.Rejection{
											{
												CommandName:   "WaiveFine",
												InvariantName: "OneCopyPerLoan",
												InvariantPos:  ast.Position{Filename: "lending.emod", Line: 30, Column: 41},
											},
										},
									},
								},
							},
						},
					}

					require.Equal(t, []string{
						`lending.emod:16: invariant "FiveCopiesPerMember" is not declared in aggregate "Loan"`,
						`lending.emod:30: invariant "OneCopyPerLoan" is not declared in context "Lending"`,
					}, reportedLines(validator.Validate(model)))
				})

				t.Run("findings from a scope's specs and its edges come back in declaration order", func(t *testing.T) {
					model := &ast.Model{
						Contexts: []*ast.Context{
							{
								Name: "Lending",
								Aggregates: []*ast.Aggregate{
									{
										Name:       "Loan",
										Invariants: []*ast.Invariant{{Name: "OneCopyPerLoan"}},
										Slices: []*ast.Slice{
											{
												Name:     "Borrow Copy",
												Commands: []*ast.Command{{Name: "BorrowCopy"}},
												Events:   []*ast.Event{{Name: "CopyBorrowed"}},
												Flows:    []*ast.Flow{{CommandName: "BorrowCopy", EventName: "CopyBorrowed"}},
												Rejections: []*ast.Rejection{
													{
														CommandName:   "BorrowCopy",
														InvariantName: "EdgeAbove",
														InvariantPos:  ast.Position{Filename: "lending.emod", Line: 12, Column: 43},
													},
													{
														CommandName:   "BorrowCopy",
														InvariantName: "EdgeBelow",
														InvariantPos:  ast.Position{Filename: "lending.emod", Line: 30, Column: 43},
													},
												},
												Specs: []*ast.Spec{
													{
														Name: "refuses a copy already on loan",
														When: &ast.SpecElement{Name: "BorrowCopy"},
														Then: &ast.ThenRejected{
															InvariantName: "SpecBetween",
															InvariantPos:  ast.Position{Filename: "lending.emod", Line: 20, Column: 21},
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

					require.Equal(t, []string{
						`lending.emod:12: invariant "EdgeAbove" is not declared in aggregate "Loan"`,
						`lending.emod:20: invariant "SpecBetween" is not declared in aggregate "Loan"`,
						`lending.emod:30: invariant "EdgeBelow" is not declared in aggregate "Loan"`,
					}, reportedLines(validator.Validate(model)))
				})

				t.Run("two scopes declaring one invariant name each resolve their own edges, and a third scope declaring it not at all is the only one reported", func(t *testing.T) {
					edgeSlice := func(name, command, event string, pos ast.Position) *ast.Slice {
						return &ast.Slice{
							Name:       name,
							Commands:   []*ast.Command{{Name: command}},
							Events:     []*ast.Event{{Name: event}},
							Flows:      []*ast.Flow{{CommandName: command, EventName: event}},
							Rejections: []*ast.Rejection{{CommandName: command, InvariantName: "OneCopyPerLoan", InvariantPos: pos}},
						}
					}
					model := &ast.Model{
						Contexts: []*ast.Context{
							{
								Name: "Lending",
								Aggregates: []*ast.Aggregate{
									{
										Name:       "Loan",
										Invariants: []*ast.Invariant{{Name: "OneCopyPerLoan"}},
										Slices:     []*ast.Slice{edgeSlice("Borrow Copy", "BorrowCopy", "CopyBorrowed", ast.Position{})},
									},
									{
										Name:       "Reservation",
										Invariants: []*ast.Invariant{{Name: "OneCopyPerLoan"}},
										Slices:     []*ast.Slice{edgeSlice("Hold Copy", "HoldCopy", "CopyHeld", ast.Position{})},
									},
								},
								Slices: []*ast.Slice{
									edgeSlice("Waive Fine", "WaiveFine", "FineWaived", ast.Position{Filename: "lending.emod", Line: 44, Column: 41}),
								},
							},
						},
					}

					require.Equal(t, []string{
						`lending.emod:44: invariant "OneCopyPerLoan" is not declared in context "Lending"`,
					}, reportedLines(validator.Validate(model)))
				})

				t.Run("an edge naming a command no slice declares is reported on nothing, as a flow's is not", func(t *testing.T) {
					model := rejectionScopeModel(&ast.Slice{
						Name:       "Borrow Copy",
						Commands:   []*ast.Command{{Name: "BorrowCopy"}},
						Events:     []*ast.Event{{Name: "CopyBorrowed"}},
						Flows:      []*ast.Flow{{CommandName: "BorrowCopy", EventName: "CopyBorrowed"}},
						Rejections: []*ast.Rejection{{CommandName: "ReturnCopy", InvariantName: "OneCopyPerLoan"}},
					})

					require.Empty(t, validator.Validate(model))
				})

				t.Run("the shared rejection fixture resolves every edge it states", func(t *testing.T) {
					require.Empty(t, validator.Validate(test.RejectionLibraryLendingModel(t)))
				})
			})
		})

		t.Run("outcome references", func(t *testing.T) {
			t.Run("a view outcome naming an undeclared view is reported on the view name", func(t *testing.T) {
				model := &ast.Model{
					Contexts: []*ast.Context{
						{
							Name: "Lending",
							Aggregates: []*ast.Aggregate{
								{
									Name: "Loan",
									Slices: []*ast.Slice{
										{
											Name:  "Review Member Loans",
											Views: []*ast.View{{Name: "MemberLoansView"}},
											Specs: []*ast.Spec{
												{
													Name: "lists loans no one holds",
													Then: &ast.ThenView{
														ViewName: "MissingView",
														ViewPos:  ast.Position{Filename: "lending.emod", Line: 12, Column: 18},
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

				require.Equal(t, []*diagnostic.Entry{
					{
						Filename: "lending.emod",
						Line:     12,
						Column:   18,
						Severity: diagnostic.Error,
						Message:  `view "MissingView" does not exist`,
					},
				}, diags)
			})

			t.Run("a command outcome naming an undeclared command is reported on the command name", func(t *testing.T) {
				model := &ast.Model{
					Contexts: []*ast.Context{
						{
							Name: "Lending",
							Aggregates: []*ast.Aggregate{
								{
									Name: "Loan",
									Slices: []*ast.Slice{
										{
											Name:     "Sweep Overdue Loans",
											Commands: []*ast.Command{{Name: "RecallCopy"}},
											Events:   []*ast.Event{{Name: "CopyRecalled"}},
											Flows:    []*ast.Flow{{CommandName: "RecallCopy", EventName: "CopyRecalled"}},
											Automations: []*ast.Automation{
												{Name: "RecallOverdueCopy", Command: "RecallCopy"},
											},
											Specs: []*ast.Spec{
												{
													Name: "recalls copies that are overdue",
													Then: &ast.ThenCommand{
														CommandName: "MissingCommand",
														CommandPos:  ast.Position{Filename: "lending.emod", Line: 15, Column: 20},
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

				require.Equal(t, []*diagnostic.Entry{
					{
						Filename: "lending.emod",
						Line:     15,
						Column:   20,
						Severity: diagnostic.Error,
						Message:  `command "MissingCommand" does not exist`,
					},
				}, diags)
			})

			t.Run("a view outcome naming a view in another aggregate produces no diagnostic", func(t *testing.T) {
				model := &ast.Model{
					Contexts: []*ast.Context{
						{
							Name: "Lending",
							Aggregates: []*ast.Aggregate{
								{
									Name: "Loan",
									Slices: []*ast.Slice{
										{
											Name:  "Review Member Loans",
											Views: []*ast.View{{Name: "MemberLoansView"}},
										},
										{
											Name:  "Chase Overdue Copy",
											Views: []*ast.View{{Name: "OverdueLoansView"}},
											Specs: []*ast.Spec{
												{
													Name: "lists loans that are overdue",
													Then: &ast.ThenView{ViewName: "MemberLoansView"},
												},
											},
										},
									},
								},
							},
						},
					},
				}

				require.Empty(t, validator.Validate(model))
			})

			t.Run("a view outcome naming a view on a DCB context produces no diagnostic", func(t *testing.T) {
				model := &ast.Model{
					Contexts: []*ast.Context{
						{
							Name: "Lending",
							Aggregates: []*ast.Aggregate{
								{
									Name: "Loan",
									Slices: []*ast.Slice{
										{
											Name:  "Chase Overdue Copy",
											Views: []*ast.View{{Name: "OverdueLoansView"}},
											Specs: []*ast.Spec{
												{
													Name: "lists loans that are overdue",
													Then: &ast.ThenView{ViewName: "DeskOccupancyView"},
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
									Name:  "Browse Desk Occupancy",
									Views: []*ast.View{{Name: "DeskOccupancyView"}},
								},
							},
						},
					},
				}

				require.Empty(t, validator.Validate(model))
			})

			t.Run("a command outcome naming a command in another context produces no diagnostic", func(t *testing.T) {
				model := &ast.Model{
					Contexts: []*ast.Context{
						{
							Name: "Lending",
							Aggregates: []*ast.Aggregate{
								{
									Name: "Loan",
									Slices: []*ast.Slice{
										{
											Name:     "Sweep Overdue Loans",
											Commands: []*ast.Command{{Name: "RecallCopy"}},
											Events:   []*ast.Event{{Name: "CopyRecalled"}},
											Flows:    []*ast.Flow{{CommandName: "RecallCopy", EventName: "CopyRecalled"}},
											Automations: []*ast.Automation{
												{Name: "RecallOverdueCopy", Command: "RecallCopy"},
											},
											Specs: []*ast.Spec{
												{
													Name: "recalls copies that are overdue",
													Then: &ast.ThenCommand{CommandName: "ReleaseDesk"},
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
									Name:     "Release Desk",
									Commands: []*ast.Command{{Name: "ReleaseDesk"}},
									Events:   []*ast.Event{{Name: "DeskReleased"}},
									Flows:    []*ast.Flow{{CommandName: "ReleaseDesk", EventName: "DeskReleased"}},
								},
							},
						},
					},
				}

				require.Empty(t, validator.Validate(model))
			})

			t.Run("a view outcome naming something declared only as a command is reported as a missing view", func(t *testing.T) {
				model := &ast.Model{
					Contexts: []*ast.Context{
						{
							Name: "Lending",
							Aggregates: []*ast.Aggregate{
								{
									Name: "Loan",
									Slices: []*ast.Slice{
										{
											Name:     "Sweep Overdue Loans",
											Views:    []*ast.View{{Name: "OverdueLoansView"}},
											Commands: []*ast.Command{{Name: "RecallCopy"}},
											Events:   []*ast.Event{{Name: "CopyRecalled"}},
											Flows:    []*ast.Flow{{CommandName: "RecallCopy", EventName: "CopyRecalled"}},
											Specs: []*ast.Spec{
												{
													Name: "recalls copies that are overdue",
													Then: &ast.ThenView{
														ViewName: "RecallCopy",
														ViewPos:  ast.Position{Filename: "lending.emod", Line: 12, Column: 16},
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

				require.Equal(t, []*diagnostic.Entry{
					{
						Filename: "lending.emod",
						Line:     12,
						Column:   16,
						Severity: diagnostic.Error,
						Message:  `view "RecallCopy" does not exist`,
					},
				}, diags)
			})

			t.Run("a command outcome naming something declared only as a view is reported as a missing command", func(t *testing.T) {
				model := &ast.Model{
					Contexts: []*ast.Context{
						{
							Name: "Lending",
							Aggregates: []*ast.Aggregate{
								{
									Name: "Loan",
									Slices: []*ast.Slice{
										{
											Name:     "Review Member Loans",
											Views:    []*ast.View{{Name: "MemberLoansView"}},
											Commands: []*ast.Command{{Name: "ListLoans"}},
											Events:   []*ast.Event{{Name: "LoansListed"}},
											Flows:    []*ast.Flow{{CommandName: "ListLoans", EventName: "LoansListed"}},
											Automations: []*ast.Automation{
												{Name: "ListLoansOnRequest", Command: "ListLoans"},
											},
											Specs: []*ast.Spec{
												{
													Name: "lists loans no one holds",
													Then: &ast.ThenCommand{
														CommandName: "MemberLoansView",
														CommandPos:  ast.Position{Filename: "lending.emod", Line: 12, Column: 18},
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

				require.Equal(t, []*diagnostic.Entry{
					{
						Filename: "lending.emod",
						Line:     12,
						Column:   18,
						Severity: diagnostic.Error,
						Message:  `command "MemberLoansView" does not exist`,
					},
				}, diags)
			})

			t.Run("multiple undefined view and command outcomes are reported in declaration order on every run", func(t *testing.T) {
				model := &ast.Model{
					Contexts: []*ast.Context{
						{
							Name: "Lending",
							Aggregates: []*ast.Aggregate{
								{
									Name: "Loan",
									Slices: []*ast.Slice{
										{
											Name:     "Sweep Overdue Loans",
											Commands: []*ast.Command{{Name: "RecallCopy"}},
											Events:   []*ast.Event{{Name: "CopyRecalled"}},
											Flows:    []*ast.Flow{{CommandName: "RecallCopy", EventName: "CopyRecalled"}},
											Automations: []*ast.Automation{
												{Name: "RecallOverdueCopy", Command: "RecallCopy"},
											},
											Specs: []*ast.Spec{
												{
													Name: "recalls copies that are overdue",
													Then: &ast.ThenCommand{
														CommandName: "MissingCommandA",
														CommandPos:  ast.Position{Filename: "lending.emod", Line: 10, Column: 20},
													},
												},
											},
										},
										{
											Name:  "Review Member Loans",
											Views: []*ast.View{{Name: "MemberLoansView"}},
											Specs: []*ast.Spec{
												{
													Name: "lists loans no one holds",
													Then: &ast.ThenView{
														ViewName: "MissingViewA",
														ViewPos:  ast.Position{Filename: "lending.emod", Line: 14, Column: 18},
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

				runs := make([][]string, 0, 3)
				for range 3 {
					runs = append(runs, reportedLines(validator.Validate(model)))
				}

				require.Equal(t, []string{
					`lending.emod:10: command "MissingCommandA" does not exist`,
					`lending.emod:14: view "MissingViewA" does not exist`,
				}, runs[0])
				require.Equal(t, runs[0], runs[1])
				require.Equal(t, runs[0], runs[2])
			})
		})
	})

	t.Run("payload field names", func(t *testing.T) {
		t.Run("a field the referenced construct does not declare is reported on the field name", func(t *testing.T) {
			tests := []struct {
				name  string
				specs string
				want  []string
			}{
				{
					name: "on a given element, against the event's fields",
					specs: `spec "borrows a copy the member returned" {
        given [CopyReturned { copyIdd: "C-93204" }]
        when BorrowCopy
        then [CopyBorrowed]
      }`,
					want: []string{`lending.emod:33: payload field "copyIdd" is not declared on event "CopyReturned"`},
				},
				{
					name: "on the when reference, against the command's fields",
					specs: `spec "borrows a copy no one holds" {
        when BorrowCopy { copyIdd: "C-93204" }
        then [CopyBorrowed]
      }`,
					want: []string{`lending.emod:33: payload field "copyIdd" is not declared on command "BorrowCopy"`},
				},
				{
					name: "on a then event, against the event's fields",
					specs: `spec "borrows a copy no one holds" {
        when BorrowCopy
        then [CopyBorrowed { loanIdd: "L-771" }]
      }`,
					want: []string{`lending.emod:34: payload field "loanIdd" is not declared on event "CopyBorrowed"`},
				},
				{
					name: "on a when naming an event rather than a command",
					specs: `spec "records a return the desk logged" {
        when CopyReturned { copyIdd: "C-93204" }
        then [CopyBorrowed]
      }`,
					want: []string{`lending.emod:33: payload field "copyIdd" is not declared on event "CopyReturned"`},
				},
				{
					name: "on a construct declaring no fields block at all, once per payload field",
					specs: `spec "shelves a copy no one holds" {
        when ShelveCopy { copyId: "C-93204", shelfMark: "AURELIA" }
        then [CopyBorrowed]
      }`,
					want: []string{
						`lending.emod:33: payload field "copyId" is not declared on command "ShelveCopy"`,
						`lending.emod:33: payload field "shelfMark" is not declared on command "ShelveCopy"`,
					},
				},
				{
					name: "on a field named after a DSL keyword the construct does not declare",
					specs: `spec "borrows a copy no one holds" {
        when BorrowCopy { given: "C-93204" }
        then [CopyBorrowed]
      }`,
					want: []string{`lending.emod:33: payload field "given" is not declared on command "BorrowCopy"`},
				},
			}

			for _, tc := range tests {
				t.Run(tc.name, func(t *testing.T) {
					diags := validateSource(t, lendingModelWithSpecs(tc.specs))

					require.Equal(t, tc.want, reportedLines(diags))
				})
			}
		})

		t.Run("a field the referenced construct declares is accepted", func(t *testing.T) {
			tests := []struct {
				name  string
				specs string
			}{
				{
					name: "on every reference a spec accepts",
					specs: `spec "borrows a copy the member returned" {
        given [CopyReturned { copyId: "C-93204" }]
        when BorrowCopy { copyId: "C-93204", shelfMark: "AURELIA" }
        then [CopyBorrowed { loanId: "L-771" }]
      }`,
				},
				{
					name: "when every required field but one is omitted, because a payload is partial",
					specs: `spec "borrows a copy no one holds" {
        when BorrowCopy { copyId: "C-93204" }
        then [CopyBorrowed]
      }`,
				},
				{
					name: "on a field named after a DSL keyword the construct declares",
					specs: `spec "borrows a copy no one holds" {
        when BorrowCopy { where: "AURELIA" }
        then [CopyBorrowed]
      }`,
				},
			}

			for _, tc := range tests {
				t.Run(tc.name, func(t *testing.T) {
					require.Empty(t, validateSource(t, lendingModelWithSpecs(tc.specs)))
				})
			}
		})

		t.Run("a payload on a reference no construct declares reports only the missing reference", func(t *testing.T) {
			diags := validateSource(t, lendingModelWithSpecs(`spec "borrows a copy no one holds" {
        when BorrwoCopy { copyIdd: "C-93204" }
        then [CopyBorrowed]
      }`))

			require.Equal(t, []string{`lending.emod:33: command "BorrwoCopy" does not exist`}, reportedLines(diags))
		})

		t.Run("several undeclared fields are reported in declaration order", func(t *testing.T) {
			diags := validateSource(t, lendingModelWithSpecs(`spec "borrows a copy the member returned" {
        given [CopyReturned { returnedAtt: "2024-07-19" }]
        when BorrowCopy { copyIdd: "C-93204", shelfMarkk: "AURELIA" }
        then [CopyBorrowed { loanIdd: "L-771" }]
      }`))

			require.Equal(t, []string{
				`lending.emod:33: payload field "returnedAtt" is not declared on event "CopyReturned"`,
				`lending.emod:34: payload field "copyIdd" is not declared on command "BorrowCopy"`,
				`lending.emod:34: payload field "shelfMarkk" is not declared on command "BorrowCopy"`,
				`lending.emod:35: payload field "loanIdd" is not declared on event "CopyBorrowed"`,
			}, reportedLines(diags))
		})

		t.Run("the diagnostic carries no rule name, so emod lint --explain has nothing to answer for", func(t *testing.T) {
			diags := validateSource(t, lendingModelWithSpecs(`spec "borrows a copy no one holds" {
        when BorrowCopy { copyIdd: "C-93204" }
        then [CopyBorrowed]
      }`))

			require.Len(t, diags, 1)
			require.Equal(t, diagnostic.Error, diags[0].Severity)
			require.Empty(t, diags[0].RuleName)
		})

		t.Run("two commands sharing a name declare between them every field a payload may state", func(t *testing.T) {
			source := `model "Library Lending"
context "Lending" {
  aggregate "Loan" {
    slice "Borrow Copy" {
      command BorrowCopy {
        fields {
          copyId string required
        }
      }
      event CopyBorrowed {
        fields {
          loanId string required
        }
      }
      spec "borrows a copy no one holds" {
        when BorrowCopy { copyId: "C-93204", dueOn: "2024-07-19" }
        then [CopyBorrowed]
      }
      flow {
        command -> event: BorrowCopy -> CopyBorrowed
      }
    }
    slice "Shelve Copy" {
      command BorrowCopy {
        fields {
          dueOn date required
        }
      }
      event CopyShelved {
        fields {
          shelfId string required
        }
      }
      flow {
        command -> event: BorrowCopy -> CopyShelved
      }
    }
  }
}`

			require.Empty(t, validateSource(t, source),
				"a second command of the same name has to add its fields to the first's, not replace them")
		})

		t.Run("a command and an event sharing a name keep their fields apart", func(t *testing.T) {
			source := `model "Library Lending"
context "Lending" {
  aggregate "Loan" {
    slice "Borrow Copy" {
      command Borrow {
        fields {
          copyId string required
        }
      }
      event Borrow {
        fields {
          loanId string required
        }
      }
      spec "borrows a copy no one holds" {
        when Borrow { loanId: "L-771" }
        then [Borrow { copyId: "C-93204" }]
      }
      flow {
        command -> event: Borrow -> Borrow
      }
    }
  }
}`

			require.Equal(t, []string{
				`lending.emod:16: payload field "loanId" is not declared on command "Borrow"`,
				`lending.emod:17: payload field "copyId" is not declared on event "Borrow"`,
			}, reportedLines(validateSource(t, source)),
				"a payload field has to be looked up on the kind its reference resolved to, not on both pooled together")
		})

		t.Run("the shared payload fixture states no field its constructs fail to declare", func(t *testing.T) {
			require.Empty(t, validator.Validate(test.PayloadLibraryLendingModel(t)))
		})
	})

	t.Run("payload literal kinds", func(t *testing.T) {
		t.Run("a literal the declared type admits is accepted", func(t *testing.T) {
			tests := []struct {
				name    string
				payload string
			}{
				{name: "a string for a string field", payload: `where: "AURELIA"`},
				{name: "the empty string for a string field", payload: `where: ""`},
				{name: "a YYYY-MM-DD value for a date field", payload: `dueOn: "2024-07-19"`},
				{name: "a domain-typed field taking a string", payload: `shelfMark: "AURELIA"`},
				{name: "a domain-typed field taking a number", payload: `shelfMark: 4821`},
				{name: "a domain-typed field taking a boolean", payload: `shelfMark: true`},
			}

			for _, tc := range tests {
				t.Run(tc.name, func(t *testing.T) {
					require.Empty(t, validateSource(t, lendingModelWithSpecs(fmt.Sprintf(`spec "borrows a copy no one holds" {
        when BorrowCopy { %s }
        then [CopyBorrowed]
      }`, tc.payload))))
				})
			}
		})

		t.Run("a literal the declared type admits is accepted on an event's fields", func(t *testing.T) {
			tests := []struct {
				name    string
				payload string
			}{
				{name: "an RFC 3339 value for a timestamp field", payload: `borrowedAt: "2024-07-05T14:32:00Z"`},
				{name: "an integer for an int field", payload: `renewals: 4821`},
				{name: "an integer for a decimal field", payload: `lateFee: 4821`},
				{name: "a fractional number for a decimal field", payload: `lateFee: 12.50`},
				{name: "true for a bool field", payload: `expedited: true`},
				{name: "false for a bool field", payload: `expedited: false`},
			}

			for _, tc := range tests {
				t.Run(tc.name, func(t *testing.T) {
					require.Empty(t, validateSource(t, lendingModelWithSpecs(fmt.Sprintf(`spec "borrows a copy no one holds" {
        when BorrowCopy
        then [CopyBorrowed { %s }]
      }`, tc.payload))))
				})
			}
		})

		t.Run("a canonical uuid is accepted in either case", func(t *testing.T) {
			for _, value := range []string{"7c9e6679-7425-40de-944b-e07fc1f90ae7", "7C9E6679-7425-40DE-944B-E07FC1F90AE7"} {
				t.Run(value, func(t *testing.T) {
					require.Empty(t, validateSource(t, lendingModelWithSpecs(fmt.Sprintf(`spec "returns a copy the member holds" {
        given [CopyReturned { returnId: "%s" }]
        when BorrowCopy
        then [CopyBorrowed]
      }`, value))))
				})
			}
		})

		t.Run("a literal the declared type does not admit is reported on the value", func(t *testing.T) {
			tests := []struct {
				name    string
				payload string
				want    string
			}{
				{
					name:    "a value that does not parse as a date",
					payload: `dueOn: "19-07-2024"`,
					want:    `payload value "19-07-2024" for field "dueOn" is not a valid date (expected YYYY-MM-DD)`,
				},
				{
					name:    "a date-only value for a date field naming a month that does not exist",
					payload: `dueOn: "2024-13-01"`,
					want:    `payload value "2024-13-01" for field "dueOn" is not a valid date (expected YYYY-MM-DD)`,
				},
				{
					name:    "a timestamp value for a date field",
					payload: `dueOn: "2024-07-05T14:32:00Z"`,
					want:    `payload value "2024-07-05T14:32:00Z" for field "dueOn" is not a valid date (expected YYYY-MM-DD)`,
				},
				{
					name:    "a number for a string field",
					payload: `where: 4821`,
					want:    `payload value 4821 for field "where" is not a valid string`,
				},
				{
					name:    "a boolean for a string field",
					payload: `where: true`,
					want:    `payload value true for field "where" is not a valid string`,
				},
			}

			for _, tc := range tests {
				t.Run(tc.name, func(t *testing.T) {
					diags := validateSource(t, lendingModelWithSpecs(fmt.Sprintf(`spec "borrows a copy no one holds" {
        when BorrowCopy { %s }
        then [CopyBorrowed]
      }`, tc.payload)))

					require.Equal(t, []string{"lending.emod:33: " + tc.want}, reportedLines(diags))
				})
			}
		})

		t.Run("a literal an event field's declared type does not admit is reported on the value", func(t *testing.T) {
			tests := []struct {
				name    string
				payload string
				want    string
			}{
				{
					name:    "a value that does not parse as a timestamp",
					payload: `borrowedAt: "yesterday"`,
					want:    `payload value "yesterday" for field "borrowedAt" is not a valid timestamp (expected an RFC 3339 timestamp)`,
				},
				{
					name:    "a date-only value for a timestamp field",
					payload: `borrowedAt: "2024-07-05"`,
					want:    `payload value "2024-07-05" for field "borrowedAt" is not a valid timestamp (expected an RFC 3339 timestamp)`,
				},
				{
					name:    "a fractional number for an int field",
					payload: `renewals: 12.50`,
					want:    `payload value 12.50 for field "renewals" is not a valid int`,
				},
				{
					name:    "a string for an int field",
					payload: `renewals: "4821"`,
					want:    `payload value "4821" for field "renewals" is not a valid int`,
				},
				{
					name:    "a string for a bool field",
					payload: `expedited: "yes"`,
					want:    `payload value "yes" for field "expedited" is not a valid bool`,
				},
				{
					name:    "a string for a decimal field",
					payload: `lateFee: "12.50"`,
					want:    `payload value "12.50" for field "lateFee" is not a valid decimal`,
				},
				{
					name:    "a boolean for a decimal field",
					payload: `lateFee: true`,
					want:    `payload value true for field "lateFee" is not a valid decimal`,
				},
			}

			for _, tc := range tests {
				t.Run(tc.name, func(t *testing.T) {
					diags := validateSource(t, lendingModelWithSpecs(fmt.Sprintf(`spec "borrows a copy no one holds" {
        when BorrowCopy
        then [CopyBorrowed { %s }]
      }`, tc.payload)))

					require.Equal(t, []string{"lending.emod:34: " + tc.want}, reportedLines(diags))
				})
			}
		})

		t.Run("a value that is not a canonical uuid is reported", func(t *testing.T) {
			diags := validateSource(t, lendingModelWithSpecs(`spec "returns a copy the member holds" {
        given [CopyReturned { returnId: "7c9e6679742540de944be07fc1f90ae7" }]
        when BorrowCopy
        then [CopyBorrowed]
      }`))

			require.Equal(t, []string{
				`lending.emod:33: payload value "7c9e6679742540de944be07fc1f90ae7" for field "returnId" is not a valid uuid (expected 8-4-4-4-12 hexadecimal digits)`,
			}, reportedLines(diags))
		})

		t.Run("a field whose declared type spells a DSL keyword is a domain type and takes any literal", func(t *testing.T) {
			for _, value := range []string{`"AURELIA"`, "4821", "true"} {
				t.Run(value, func(t *testing.T) {
					require.Empty(t, validateSource(t, lendingModelWithSpecs(fmt.Sprintf(`spec "returns a copy the member holds" {
        given [CopyReturned { shelf: %s }]
        when BorrowCopy
        then [CopyBorrowed]
      }`, value))))
				})
			}
		})

		t.Run("several mismatched literals are reported in declaration order", func(t *testing.T) {
			diags := validateSource(t, lendingModelWithSpecs(`spec "returns a copy the member holds" {
        given [CopyReturned { returnedAt: "yesterday" }]
        when BorrowCopy { dueOn: "19-07-2024", where: 4821 }
        then [CopyBorrowed { renewals: 12.50 }]
      }`))

			require.Equal(t, []string{
				`lending.emod:33: payload value "yesterday" for field "returnedAt" is not a valid timestamp (expected an RFC 3339 timestamp)`,
				`lending.emod:34: payload value "19-07-2024" for field "dueOn" is not a valid date (expected YYYY-MM-DD)`,
				`lending.emod:34: payload value 4821 for field "where" is not a valid string`,
				`lending.emod:35: payload value 12.50 for field "renewals" is not a valid int`,
			}, reportedLines(diags))
		})

		t.Run("the diagnostic carries no rule name, so emod lint --explain has nothing to answer for", func(t *testing.T) {
			diags := validateSource(t, lendingModelWithSpecs(`spec "borrows a copy no one holds" {
        when BorrowCopy { dueOn: "19-07-2024" }
        then [CopyBorrowed]
      }`))

			require.Len(t, diags, 1)
			require.Equal(t, diagnostic.Error, diags[0].Severity)
			require.Empty(t, diags[0].RuleName)
		})

		t.Run("stripping the payloads silences the check and leaves the rest of validation alone", func(t *testing.T) {
			stated := parseSource(t, lendingModelWithSpecs(`spec "borrows a copy no one holds" {
        when BorrowCopy { dueOn: "19-07-2024" }
        then [CopyBorrowed { renewals: 12.50 }]
      }`))

			require.Len(t, validator.Validate(stated), 2)
			require.Empty(t, validator.Validate(test.WithoutSpecPayloads(stated)))
		})
	})

	t.Run("outcome shape", func(t *testing.T) {
		t.Run("a view outcome in a slice declaring no view is reported on the view name", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Lending",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Loan",
								Slices: []*ast.Slice{
									{
										Name:  "Review Member Loans",
										Views: []*ast.View{{Name: "MemberLoansView"}},
									},
									{
										Name:     "Borrow Copy",
										Commands: []*ast.Command{{Name: "BorrowCopy"}},
										Events:   []*ast.Event{{Name: "CopyBorrowed"}},
										Flows:    []*ast.Flow{{CommandName: "BorrowCopy", EventName: "CopyBorrowed"}},
										Specs: []*ast.Spec{
											{
												Name: "lists loans no one holds",
												Then: &ast.ThenView{
													ViewName: "MemberLoansView",
													ViewPos:  ast.Position{Filename: "lending.emod", Line: 12, Column: 18},
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

			require.Equal(t, []*diagnostic.Entry{
				{
					Filename: "lending.emod",
					Line:     12,
					Column:   18,
					Severity: diagnostic.Error,
					Message:  `outcome "view" requires a view in this slice`,
				},
			}, diags)
		})

		t.Run("a command outcome in a slice declaring no automation is reported on the command name", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Lending",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Loan",
								Slices: []*ast.Slice{
									{
										Name:     "Borrow Copy",
										Commands: []*ast.Command{{Name: "BorrowCopy"}},
										Events:   []*ast.Event{{Name: "CopyBorrowed"}},
										Flows:    []*ast.Flow{{CommandName: "BorrowCopy", EventName: "CopyBorrowed"}},
										Specs: []*ast.Spec{
											{
												Name: "recalls copies that are overdue",
												Then: &ast.ThenCommand{
													CommandName: "BorrowCopy",
													CommandPos:  ast.Position{Filename: "lending.emod", Line: 12, Column: 20},
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

			require.Equal(t, []*diagnostic.Entry{
				{
					Filename: "lending.emod",
					Line:     12,
					Column:   20,
					Severity: diagnostic.Error,
					Message:  `outcome "command" requires an automation in this slice`,
				},
			}, diags)
		})

		t.Run("a rejection outcome in a slice declaring no command is reported on the invariant name", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Lending",
						Aggregates: []*ast.Aggregate{
							{
								Name:       "Loan",
								Invariants: []*ast.Invariant{{Name: "OneCopyPerLoan"}},
								Slices: []*ast.Slice{
									{
										Name:  "Review Member Loans",
										Views: []*ast.View{{Name: "MemberLoansView"}},
										Specs: []*ast.Spec{
											{
												Name: "refuses a copy already on loan",
												Then: &ast.ThenRejected{
													InvariantName: "OneCopyPerLoan",
													InvariantPos:  ast.Position{Filename: "lending.emod", Line: 12, Column: 21},
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

			require.Equal(t, []*diagnostic.Entry{
				{
					Filename: "lending.emod",
					Line:     12,
					Column:   21,
					Severity: diagnostic.Error,
					Message:  `outcome "rejected" requires a command in this slice`,
				},
			}, diags)
		})

		t.Run("an event list outcome in a slice declaring neither command nor translation is reported on the first event", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Lending",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Loan",
								Slices: []*ast.Slice{
									{
										Name:   "Review Member Loans",
										Views:  []*ast.View{{Name: "MemberLoansView"}},
										Events: []*ast.Event{{Name: "CopyBorrowed", Source: "external"}},
										Specs: []*ast.Spec{
											{
												Name: "lists loans no one holds",
												Then: &ast.ThenEvents{Events: []*ast.SpecElement{
													{Name: "CopyBorrowed", NamePos: ast.Position{Filename: "lending.emod", Line: 12, Column: 15}},
												}},
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

			require.Equal(t, []*diagnostic.Entry{
				{
					Filename: "lending.emod",
					Line:     12,
					Column:   15,
					Severity: diagnostic.Error,
					Message:  `outcome "events" requires a command or translation in this slice`,
				},
			}, diags)
		})

		t.Run("an empty event list outcome in a slice declaring neither command nor translation is reported on the spec name", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Lending",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Loan",
								Slices: []*ast.Slice{
									{
										Name:  "Review Member Loans",
										Views: []*ast.View{{Name: "MemberLoansView"}},
										Specs: []*ast.Spec{
											{
												Name:    "lists loans no one holds",
												NamePos: ast.Position{Filename: "lending.emod", Line: 10, Column: 12},
												Then:    &ast.ThenEvents{Events: []*ast.SpecElement{}},
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

			require.Equal(t, []*diagnostic.Entry{
				{
					Filename: "lending.emod",
					Line:     10,
					Column:   12,
					Severity: diagnostic.Error,
					Message:  `outcome "events" requires a command or translation in this slice`,
				},
			}, diags)
		})

		t.Run("a view outcome in a slice declaring a view produces no diagnostic", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Lending",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Loan",
								Slices: []*ast.Slice{
									{
										Name:  "Review Member Loans",
										Views: []*ast.View{{Name: "MemberLoansView"}},
										Specs: []*ast.Spec{
											{
												Name: "lists loans no one holds",
												Then: &ast.ThenView{ViewName: "MemberLoansView"},
											},
										},
									},
								},
							},
						},
					},
				},
			}

			require.Empty(t, validator.Validate(model))
		})

		t.Run("a command outcome in a slice declaring an automation produces no diagnostic", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Lending",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Loan",
								Slices: []*ast.Slice{
									{
										Name:     "Sweep Overdue Loans",
										Commands: []*ast.Command{{Name: "RecallCopy"}},
										Events:   []*ast.Event{{Name: "CopyRecalled"}},
										Flows:    []*ast.Flow{{CommandName: "RecallCopy", EventName: "CopyRecalled"}},
										Automations: []*ast.Automation{
											{Name: "RecallOverdueCopy", Command: "RecallCopy"},
										},
										Specs: []*ast.Spec{
											{
												Name: "recalls copies that are overdue",
												Then: &ast.ThenCommand{CommandName: "RecallCopy"},
											},
										},
									},
								},
							},
						},
					},
				},
			}

			require.Empty(t, validator.Validate(model))
		})

		t.Run("a rejection outcome in a slice declaring a command produces no diagnostic", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Lending",
						Aggregates: []*ast.Aggregate{
							{
								Name:       "Loan",
								Invariants: []*ast.Invariant{{Name: "OneCopyPerLoan"}},
								Slices: []*ast.Slice{
									{
										Name:     "Borrow Copy",
										Commands: []*ast.Command{{Name: "BorrowCopy"}},
										Events:   []*ast.Event{{Name: "CopyBorrowed"}},
										Flows:    []*ast.Flow{{CommandName: "BorrowCopy", EventName: "CopyBorrowed"}},
										Specs: []*ast.Spec{
											{
												Name: "refuses a copy already on loan",
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

			require.Empty(t, validator.Validate(model))
		})

		t.Run("an event list outcome in a slice declaring a command produces no diagnostic", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Lending",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Loan",
								Slices: []*ast.Slice{
									{
										Name:     "Borrow Copy",
										Commands: []*ast.Command{{Name: "BorrowCopy"}},
										Events:   []*ast.Event{{Name: "CopyBorrowed"}},
										Flows:    []*ast.Flow{{CommandName: "BorrowCopy", EventName: "CopyBorrowed"}},
										Specs: []*ast.Spec{
											{
												Name: "borrows a copy no one holds",
												Then: &ast.ThenEvents{Events: []*ast.SpecElement{{Name: "CopyBorrowed"}}},
											},
										},
									},
								},
							},
						},
					},
				},
			}

			require.Empty(t, validator.Validate(model))
		})

		t.Run("an event list outcome in a slice declaring a translation produces no diagnostic", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Reading Room",
						Mode: "dcb",
						Slices: []*ast.Slice{
							{
								Name: "Import External Desk Booking",
								Translations: []*ast.Translation{
									{
										Name:  "ExternalDeskBookingImport",
										Event: &ast.Event{Name: "ExternalDeskBookingImported"},
									},
								},
								Specs: []*ast.Spec{
									{
										Name: "imports a desk booking",
										Then: &ast.ThenEvents{Events: []*ast.SpecElement{{Name: "ExternalDeskBookingImported"}}},
									},
								},
							},
						},
					},
				},
			}

			require.Empty(t, validator.Validate(model))
		})

		t.Run("a slice declaring both a command and an automation accepts event-list and command outcomes", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Lending",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Loan",
								Slices: []*ast.Slice{
									{
										Name:     "Chase Overdue Copy",
										Commands: []*ast.Command{{Name: "RemindMember"}},
										Events:   []*ast.Event{{Name: "MemberReminded"}},
										Flows:    []*ast.Flow{{CommandName: "RemindMember", EventName: "MemberReminded"}},
										Automations: []*ast.Automation{
											{Name: "RemindOnDueDate", Command: "RemindMember"},
										},
										Specs: []*ast.Spec{
											{
												Name: "reminds a member when a copy becomes due",
												Then: &ast.ThenEvents{Events: []*ast.SpecElement{{Name: "MemberReminded"}}},
											},
											{
												Name: "recalls copies that are overdue",
												Then: &ast.ThenCommand{CommandName: "RemindMember"},
											},
										},
									},
								},
							},
						},
					},
				},
			}

			require.Empty(t, validator.Validate(model))
		})

		t.Run("a spec that omits when is accepted in every slice", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Lending",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Loan",
								Slices: []*ast.Slice{
									{
										Name:     "Borrow Copy",
										Commands: []*ast.Command{{Name: "BorrowCopy"}},
										Events:   []*ast.Event{{Name: "CopyBorrowed"}},
										Flows:    []*ast.Flow{{CommandName: "BorrowCopy", EventName: "CopyBorrowed"}},
										Specs: []*ast.Spec{
											{
												Name: "borrows a copy no one holds",
												Then: &ast.ThenEvents{Events: []*ast.SpecElement{{Name: "CopyBorrowed"}}},
											},
										},
									},
								},
							},
						},
					},
				},
			}

			require.Empty(t, validator.Validate(model))
		})

		t.Run("a when naming an event is accepted in every slice", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Lending",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Loan",
								Slices: []*ast.Slice{
									{
										Name:     "Borrow Copy",
										Commands: []*ast.Command{{Name: "BorrowCopy"}},
										Events:   []*ast.Event{{Name: "CopyBorrowed"}},
										Flows:    []*ast.Flow{{CommandName: "BorrowCopy", EventName: "CopyBorrowed"}},
										Specs: []*ast.Spec{
											{
												Name: "borrows a copy no one holds",
												When: &ast.SpecElement{Name: "CopyBorrowed"},
												Then: &ast.ThenEvents{Events: []*ast.SpecElement{{Name: "CopyBorrowed"}}},
											},
										},
									},
								},
							},
						},
					},
				},
			}

			require.Empty(t, validator.Validate(model))
		})

		t.Run("mismatched outcomes in both slice homes are reported in declaration order on every run", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name:       "Lending",
						Invariants: []*ast.Invariant{{Name: "OneReaderPerDesk"}},
						Aggregates: []*ast.Aggregate{
							{
								Name: "Loan",
								Slices: []*ast.Slice{
									{
										Name:     "Borrow Copy",
										NamePos:  ast.Position{Filename: "lending.emod", Line: 5, Column: 5},
										Commands: []*ast.Command{{Name: "BorrowCopy"}},
										Events:   []*ast.Event{{Name: "CopyBorrowed"}},
										Flows:    []*ast.Flow{{CommandName: "BorrowCopy", EventName: "CopyBorrowed"}},
										Specs: []*ast.Spec{
											{
												Name: "lists loans no one holds",
												Then: &ast.ThenView{
													ViewName: "MemberLoansView",
													ViewPos:  ast.Position{Filename: "lending.emod", Line: 10, Column: 18},
												},
											},
										},
									},
								},
							},
						},
						Slices: []*ast.Slice{
							{
								Name:    "Review Member Loans",
								NamePos: ast.Position{Filename: "lending.emod", Line: 25, Column: 5},
								Views:   []*ast.View{{Name: "MemberLoansView"}},
								Specs: []*ast.Spec{
									{
										Name: "refuses a desk another reader is seated at",
										Then: &ast.ThenRejected{
											InvariantName: "OneReaderPerDesk",
											InvariantPos:  ast.Position{Filename: "lending.emod", Line: 30, Column: 21},
										},
									},
								},
							},
						},
					},
				},
			}

			runs := make([][]string, 0, 3)
			for range 3 {
				runs = append(runs, reportedLines(validator.Validate(model)))
			}

			require.Equal(t, []string{
				`lending.emod:10: outcome "view" requires a view in this slice`,
				`lending.emod:30: outcome "rejected" requires a command in this slice`,
			}, runs[0])
			require.Equal(t, runs[0], runs[1])
			require.Equal(t, runs[0], runs[2])
		})
	})

	t.Run("orphan commands and events", func(t *testing.T) {
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

		t.Run("orphan diagnostics follow declaration order, not name order", func(t *testing.T) {
			// Declared last-to-first alphabetically, so a name-ordered
			// implementation would report these the other way round.
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Orders",
						Aggregates: []*ast.Aggregate{
							{
								Slices: []*ast.Slice{
									{
										Events: []*ast.Event{
											{Name: "Zulu", NamePos: ast.Position{Filename: "test.emod", Line: 3, Column: 5}},
											{Name: "Alpha", NamePos: ast.Position{Filename: "test.emod", Line: 7, Column: 5}},
											{Name: "Mike", NamePos: ast.Position{Filename: "test.emod", Line: 11, Column: 5}},
										},
									},
								},
							},
						},
					},
				},
			}

			messages := make([]string, 0, 3)
			for _, d := range validator.Validate(model) {
				messages = append(messages, d.Message)
			}

			require.Equal(t, []string{
				`event "Zulu" is orphaned (no flow, external source, or translation produces it)`,
				`event "Alpha" is orphaned (no flow, external source, or translation produces it)`,
				`event "Mike" is orphaned (no flow, external source, or translation produces it)`,
			}, messages)
		})
	})

	t.Run("duplicate invariant declarations", func(t *testing.T) {
		t.Run("an identifier declared twice in one aggregate is reported on the redeclaration", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Lending",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Loan",
								Invariants: []*ast.Invariant{
									{Name: "OneCopyPerLoan", NamePos: ast.Position{Filename: "lending.emod", Line: 4, Column: 5}},
									{Name: "OneCopyPerLoan", NamePos: ast.Position{Filename: "lending.emod", Line: 9, Column: 5}},
								},
							},
						},
					},
				},
			}

			diags := validator.Validate(model)

			require.Len(t, diags, 1)
			require.Equal(t, diagnostic.Error, diags[0].Severity)
			require.Contains(t, diags[0].Message, "OneCopyPerLoan")
			require.Contains(t, diags[0].Message, "Loan")
			require.Equal(t, "lending.emod", diags[0].Filename)
			require.Equal(t, 9, diags[0].Line)
			require.Equal(t, 5, diags[0].Column)
		})

		t.Run("an identifier declared twice on one context is reported on the redeclaration", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Reading Room",
						Mode: "dcb",
						Invariants: []*ast.Invariant{
							{Name: "OneReaderPerDesk", NamePos: ast.Position{Filename: "reading.emod", Line: 3, Column: 3}},
							{Name: "OneReaderPerDesk", NamePos: ast.Position{Filename: "reading.emod", Line: 11, Column: 3}},
						},
					},
				},
			}

			diags := validator.Validate(model)

			require.Len(t, diags, 1)
			require.Equal(t, diagnostic.Error, diags[0].Severity)
			require.Contains(t, diags[0].Message, "OneReaderPerDesk")
			require.Contains(t, diags[0].Message, "Reading Room")
			require.Equal(t, "reading.emod", diags[0].Filename)
			require.Equal(t, 11, diags[0].Line)
			require.Equal(t, 3, diags[0].Column)
		})

		t.Run("sibling aggregates may each declare the same identifier", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Lending",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Loan",
								Invariants: []*ast.Invariant{
									{Name: "OneAtATime", NamePos: ast.Position{Filename: "lending.emod", Line: 4, Column: 5}},
								},
							},
							{
								Name: "Reservation",
								Invariants: []*ast.Invariant{
									{Name: "OneAtATime", NamePos: ast.Position{Filename: "lending.emod", Line: 10, Column: 5}},
								},
							},
						},
					},
				},
			}

			diags := validator.Validate(model)

			require.Empty(t, diags)
		})

		t.Run("an aggregate and its enclosing context may each declare the same identifier", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Lending",
						Invariants: []*ast.Invariant{
							{Name: "OneAtATime", NamePos: ast.Position{Filename: "lending.emod", Line: 3, Column: 3}},
						},
						Aggregates: []*ast.Aggregate{
							{
								Name: "Loan",
								Invariants: []*ast.Invariant{
									{Name: "OneAtATime", NamePos: ast.Position{Filename: "lending.emod", Line: 6, Column: 5}},
								},
							},
						},
					},
				},
			}

			diags := validator.Validate(model)

			require.Empty(t, diags)
		})

		t.Run("three declarations of one identifier are reported once per redeclaration", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Lending",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Loan",
								Invariants: []*ast.Invariant{
									{Name: "OneCopyPerLoan", NamePos: ast.Position{Filename: "lending.emod", Line: 4, Column: 5}},
									{Name: "OneCopyPerLoan", NamePos: ast.Position{Filename: "lending.emod", Line: 8, Column: 5}},
									{Name: "OneCopyPerLoan", NamePos: ast.Position{Filename: "lending.emod", Line: 12, Column: 5}},
								},
							},
						},
					},
				},
			}

			diags := validator.Validate(model)

			require.Len(t, diags, 2)
			require.Equal(t, 8, diags[0].Line)
			require.Equal(t, 12, diags[1].Line)
		})

		t.Run("duplicates across scopes follow declaration order on every run", func(t *testing.T) {
			// "Lending" declares its own invariants after its aggregate and
			// "Billing" declares its own before, so neither walking a context
			// ahead of its aggregates nor the reverse yields source order.
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Lending",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Loan",
								Invariants: []*ast.Invariant{
									{Name: "Zulu", NamePos: ast.Position{Filename: "shelf.emod", Line: 5, Column: 5}},
									{Name: "Zulu", NamePos: ast.Position{Filename: "shelf.emod", Line: 12, Column: 5}},
								},
							},
						},
						Invariants: []*ast.Invariant{
							{Name: "Yankee", NamePos: ast.Position{Filename: "shelf.emod", Line: 20, Column: 3}},
							{Name: "Yankee", NamePos: ast.Position{Filename: "shelf.emod", Line: 30, Column: 3}},
						},
					},
					{
						Name: "Billing",
						Invariants: []*ast.Invariant{
							{Name: "Bravo", NamePos: ast.Position{Filename: "shelf.emod", Line: 40, Column: 3}},
							{Name: "Bravo", NamePos: ast.Position{Filename: "shelf.emod", Line: 50, Column: 3}},
						},
						Aggregates: []*ast.Aggregate{
							{
								Name: "Invoice",
								Invariants: []*ast.Invariant{
									{Name: "Alpha", NamePos: ast.Position{Filename: "shelf.emod", Line: 60, Column: 5}},
									{Name: "Alpha", NamePos: ast.Position{Filename: "shelf.emod", Line: 70, Column: 5}},
								},
							},
						},
					},
				},
			}

			runs := make([][]string, 0, 3)
			for range 3 {
				runs = append(runs, reportedLines(validator.Validate(model)))
			}

			require.Equal(t, []string{
				`shelf.emod:12: invariant "Zulu" is already declared in aggregate "Loan"`,
				`shelf.emod:30: invariant "Yankee" is already declared in context "Lending"`,
				`shelf.emod:50: invariant "Bravo" is already declared in context "Billing"`,
				`shelf.emod:70: invariant "Alpha" is already declared in aggregate "Invoice"`,
			}, runs[0])
			require.Equal(t, runs[0], runs[1])
			require.Equal(t, runs[0], runs[2])
		})
	})

	// ── DCB: tag field reference validation ──────────────────────────────

	t.Run("tag field references", func(t *testing.T) {
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
										Name:   "OrderPlaced",
										Source: "external",
										Fields: []*ast.Field{
											{Name: "orderId"},
										},
										Tags: []ast.TagEntry{
											{
												Key:      "aggregate_id",
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
										Name:   "OrderPlaced",
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
										Name:   "OrderPlaced",
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
	})

	// ── DCB: decides_on event name validation ────────────────────────────

	t.Run("decides_on event references", func(t *testing.T) {
		t.Run("decides_on with valid event names produces no diagnostics", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Orders",
						Slices: []*ast.Slice{
							{
								Events: []*ast.Event{
									{
										Name:   "OrderPlaced",
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
											Events: []string{"NonExistentEvent"},
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
	})

	// ── DCB: predicate tag key and field reference validation ────────────

	t.Run("decides_on predicates", func(t *testing.T) {
		t.Run("predicate with valid tag key and fieldRef produces no diagnostics", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Orders",
						Slices: []*ast.Slice{
							{
								Events: []*ast.Event{
									{
										Name:   "OrderPlaced",
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
										Name:   "OrderPlaced",
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
										Name:   "OrderPlaced",
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
										Name:   "OrderPlaced",
										Source: "external",
										Fields: []*ast.Field{
											{Name: "orderId"},
										},
										Tags: []ast.TagEntry{
											{Key: "aggregate_id", FieldRef: "orderId"},
										},
									},
									{
										Name:   "OrderShipped",
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
										Name:   "OrderPlaced",
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
										Name:   "OrderPlaced",
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
	})

	// ── DCB: valid references across both slice locations ────────────────

	t.Run("tags across context shapes", func(t *testing.T) {
		t.Run("ctx.Slices events with valid tags produce no diagnostics", func(t *testing.T) {
			model := &ast.Model{
				Contexts: []*ast.Context{
					{
						Name: "Orders",
						Slices: []*ast.Slice{
							{
								Events: []*ast.Event{
									{
										Name:   "OrderPlaced",
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
												Name:   "OrderPlaced",
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
	})
}

// modelSchedulingEvery gives each expression an automation of its own, five
// lines apart, so a reported line says which expression it is about.
func modelSchedulingEvery(expressions ...string) *ast.Model {
	automations := make([]*ast.Automation, 0, len(expressions))
	for i, expression := range expressions {
		line := 7 + i*5
		automations = append(automations, &ast.Automation{
			Name:        fmt.Sprintf("Processor%d", i+1),
			NamePos:     ast.Position{Filename: "reservations.emod", Line: line - 1, Column: 18},
			Schedule:    expression,
			SchedulePos: ast.Position{Filename: "reservations.emod", Line: line, Column: 15},
		})
	}

	return &ast.Model{
		Contexts: []*ast.Context{
			{
				Name: "Reservations",
				Aggregates: []*ast.Aggregate{
					{
						Name: "Reservation",
						Slices: []*ast.Slice{
							{
								Name:        "Expire Stale Holds",
								Automations: automations,
							},
						},
					},
				},
			},
		},
	}
}

// rejectionScopeModel puts slice under an aggregate declaring OneCopyPerLoan, so
// two models differing only in whether a spec or a rejection edge names an
// invariant can be validated against each other.
func rejectionScopeModel(slice *ast.Slice) *ast.Model {
	return &ast.Model{
		Contexts: []*ast.Context{
			{
				Name: "Lending",
				Aggregates: []*ast.Aggregate{
					{
						Name:       "Loan",
						Invariants: []*ast.Invariant{{Name: "OneCopyPerLoan"}},
						Slices:     []*ast.Slice{slice},
					},
				},
			},
		},
	}
}

// validateSource runs the validator over parsed source rather than a model built
// in Go, so the position a payload diagnostic reports is the one an author would
// see and a whole-line expectation can name it.
func validateSource(t *testing.T, source string) []*diagnostic.Entry {
	t.Helper()

	return validator.Validate(parseSource(t, source))
}

func parseSource(t *testing.T, source string) *ast.Model {
	t.Helper()

	tokens, scanDiags := lexer.Scan(source, "lending.emod")
	require.Empty(t, scanDiags)

	model, parseDiags := parser.New(tokens, "lending.emod").Parse()
	require.Empty(t, parseDiags)

	return model
}

// lendingModelWithSpecs wraps specs in a model whose commands and events declare
// fields of every type a payload literal is checked against. The spec block it
// wraps opens on line 32, which is what lets a caller name the line each
// diagnostic reports on.
func lendingModelWithSpecs(specs string) string {
	return fmt.Sprintf(`model "Library Lending"
context "Lending" {
  aggregate "Loan" {
    slice "Borrow Copy" {
      command BorrowCopy {
        fields {
          copyId    string required
          dueOn     date   required
          where     string
          shelfMark ShelfMark
        }
      }
      command ShelveCopy {
      }
      event CopyBorrowed {
        fields {
          loanId     string    required
          borrowedAt timestamp required
          renewals   int
          lateFee    decimal
          expedited  bool
        }
      }
      event CopyReturned {
        fields {
          copyId     string    required
          returnedAt timestamp required
          returnId   uuid      required
          shelf      events
        }
      }
      %s
      flow {
        command -> event: BorrowCopy -> CopyBorrowed
        command -> event: ShelveCopy -> CopyReturned
      }
    }
  }
}`, specs)
}

func reportedLines(diags []*diagnostic.Entry) []string {
	lines := make([]string, 0, len(diags))
	for _, d := range diags {
		lines = append(lines, d.String())
	}

	return lines
}

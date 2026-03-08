//go:build unit

package formatter_test

import (
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/formatter"
	"github.com/hpcsc/emod/internal/lexer"
	"github.com/hpcsc/emod/internal/parser"
	"github.com/hpcsc/emod/internal/test"
	"github.com/stretchr/testify/require"
)

func TestFormat(t *testing.T) {
	t.Run("formats model and actor declarations", func(t *testing.T) {
		model := &ast.Model{
			Name: "Test",
			Actors: []*ast.Actor{
				{Name: "Guest"},
			},
		}

		result := formatter.Format(model)

		expected := "model \"Test\"\n\nactor \"Guest\"\n"
		require.Equal(t, expected, result)
	})

	t.Run("formats context with aggregate and slice containing command, event, flow", func(t *testing.T) {
		model := &ast.Model{
			Name: "Hotel",
			Contexts: []*ast.Context{
				{
					Name: "Reservations",
					Aggregates: []*ast.Aggregate{
						{
							Name: "Reservation",
							Slices: []*ast.Slice{
								{
									Name: "Make Reservation",
									Commands: []*ast.Command{
										{
											Name: "MakeReservation",
											Fields: []*ast.Field{
												{Name: "guestId", Type: "string", Modifier: "required"},
												{Name: "roomType", Type: "string", Modifier: "required"},
											},
										},
									},
									Events: []*ast.Event{
										{
											Name: "ReservationMade",
											Fields: []*ast.Field{
												{Name: "reservationId", Type: "string", Modifier: "required"},
											},
										},
									},
									Flows: []*ast.Flow{
										{CommandName: "MakeReservation", EventName: "ReservationMade"},
									},
								},
							},
						},
					},
				},
			},
		}

		result := formatter.Format(model)

		expected := strings.Join([]string{
			`model "Hotel"`,
			``,
			`context "Reservations" {`,
			`  aggregate "Reservation" {`,
			`    slice "Make Reservation" {`,
			`      command MakeReservation {`,
			`        fields {`,
			`          guestId  string required`,
			`          roomType string required`,
			`        }`,
			`      }`,
			``,
			`      event ReservationMade {`,
			`        fields {`,
			`          reservationId string required`,
			`        }`,
			`      }`,
			``,
			`      flow {`,
			`        command -> event: MakeReservation -> ReservationMade`,
			`      }`,
			`    }`,
			`  }`,
			`}`,
			``,
		}, "\n")

		require.Equal(t, expected, result)
	})

	t.Run("formats trigger block", func(t *testing.T) {
		model := &ast.Model{
			Name: "Test",
			Contexts: []*ast.Context{
				{
					Name: "Ctx",
					Aggregates: []*ast.Aggregate{
						{
							Name: "Agg",
							Slices: []*ast.Slice{
								{
									Name: "My Slice",
									Trigger: &ast.Trigger{
										Kind:  "UI",
										Name:  "Form",
										Actor: "Guest",
										Reads: "MyView",
									},
								},
							},
						},
					},
				},
			},
		}

		result := formatter.Format(model)

		expected := strings.Join([]string{
			`model "Test"`,
			``,
			`context "Ctx" {`,
			`  aggregate "Agg" {`,
			`    slice "My Slice" {`,
			`      trigger UI "Form" {`,
			`        actor Guest`,
			`        reads MyView`,
			`      }`,
			`    }`,
			`  }`,
			`}`,
			``,
		}, "\n")

		require.Equal(t, expected, result)
	})

	t.Run("formats view with fields and subscribes", func(t *testing.T) {
		model := &ast.Model{
			Name: "Test",
			Contexts: []*ast.Context{
				{
					Name: "Ctx",
					Aggregates: []*ast.Aggregate{
						{
							Name: "Agg",
							Slices: []*ast.Slice{
								{
									Name: "My Slice",
									Views: []*ast.View{
										{
											Name: "RoomsView",
											Fields: []*ast.Field{
												{Name: "roomId", Type: "string", Modifier: "required"},
												{Name: "status", Type: "string", Modifier: "optional"},
											},
											Subscribes: []string{"RoomReserved", "GuestCheckedOut"},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		result := formatter.Format(model)

		expected := strings.Join([]string{
			`model "Test"`,
			``,
			`context "Ctx" {`,
			`  aggregate "Agg" {`,
			`    slice "My Slice" {`,
			`      view RoomsView {`,
			`        fields {`,
			`          roomId string required`,
			`          status string optional`,
			`        }`,
			`        subscribes [RoomReserved, GuestCheckedOut]`,
			`      }`,
			`    }`,
			`  }`,
			`}`,
			``,
		}, "\n")

		require.Equal(t, expected, result)
	})

	t.Run("formats automation block", func(t *testing.T) {
		model := &ast.Model{
			Name: "Test",
			Contexts: []*ast.Context{
				{
					Name: "Ctx",
					Aggregates: []*ast.Aggregate{
						{
							Name: "Agg",
							Slices: []*ast.Slice{
								{
									Name: "Notify",
									Automations: []*ast.Automation{
										{
											Name:          "OrderNotifier",
											TriggerEvent:  "OrderPlaced",
											Command:       "SendNotification",
											TargetContext: "Notifications",
										},
									},
								},
							},
						},
					},
				},
			},
		}

		result := formatter.Format(model)

		expected := strings.Join([]string{
			`model "Test"`,
			``,
			`context "Ctx" {`,
			`  aggregate "Agg" {`,
			`    slice "Notify" {`,
			`      automation OrderNotifier {`,
			`        trigger OrderPlaced`,
			`        command SendNotification`,
			`        target context Notifications`,
			`      }`,
			`    }`,
			`  }`,
			`}`,
			``,
		}, "\n")

		require.Equal(t, expected, result)
	})

	t.Run("formats translation block with nested event", func(t *testing.T) {
		model := &ast.Model{
			Name: "Test",
			Contexts: []*ast.Context{
				{
					Name: "Ctx",
					Aggregates: []*ast.Aggregate{
						{
							Name: "Agg",
							Slices: []*ast.Slice{
								{
									Name: "Import",
									Translations: []*ast.Translation{
										{
											Name:           "BookingImport",
											ExternalSystem: "Booking.com API",
											Reads:          "WebhookView",
											Command:        "ImportBooking",
											Event: &ast.Event{
												Name: "BookingImported",
												Fields: []*ast.Field{
													{Name: "bookingId", Type: "string", Modifier: "required"},
													{Name: "source", Type: "string", Modifier: "required"},
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

		result := formatter.Format(model)

		expected := strings.Join([]string{
			`model "Test"`,
			``,
			`context "Ctx" {`,
			`  aggregate "Agg" {`,
			`    slice "Import" {`,
			`      translation BookingImport {`,
			`        external_system "Booking.com API"`,
			`        reads WebhookView`,
			`        command ImportBooking`,
			`        event BookingImported {`,
			`          fields {`,
			`            bookingId string required`,
			`            source    string required`,
			`          }`,
			`        }`,
			`      }`,
			`    }`,
			`  }`,
			`}`,
			``,
		}, "\n")

		require.Equal(t, expected, result)
	})

	t.Run("formats event with source external", func(t *testing.T) {
		model := &ast.Model{
			Name: "Test",
			Contexts: []*ast.Context{
				{
					Name: "Ctx",
					Aggregates: []*ast.Aggregate{
						{
							Name: "Agg",
							Slices: []*ast.Slice{
								{
									Name: "Receive",
									Events: []*ast.Event{
										{
											Name:         "PaymentReceived",
											Source:       "external",
											ExternalName: "Stripe",
											Fields: []*ast.Field{
												{Name: "paymentId", Type: "string", Modifier: "required"},
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

		result := formatter.Format(model)

		expected := strings.Join([]string{
			`model "Test"`,
			``,
			`context "Ctx" {`,
			`  aggregate "Agg" {`,
			`    slice "Receive" {`,
			`      event PaymentReceived {`,
			`        source external "Stripe"`,
			`        fields {`,
			`          paymentId string required`,
			`        }`,
			`      }`,
			`    }`,
			`  }`,
			`}`,
			``,
		}, "\n")

		require.Equal(t, expected, result)
	})

	t.Run("round-trip: parse then format then re-parse produces equivalent AST", func(t *testing.T) {
		input := strings.Join([]string{
			`model "Hotel Reservation"`,
			``,
			`actor "Guest"`,
			``,
			`context "Reservations" {`,
			`  aggregate "Reservation" {`,
			`    slice "Make Reservation" {`,
			`      command MakeReservation {`,
			`        fields {`,
			`          guestId string required`,
			`          roomType string required`,
			`        }`,
			`      }`,
			``,
			`      event ReservationMade {`,
			`        fields {`,
			`          reservationId string required`,
			`        }`,
			`      }`,
			``,
			`      flow {`,
			`        command -> event: MakeReservation -> ReservationMade`,
			`      }`,
			`    }`,
			`  }`,
			`}`,
			``,
		}, "\n")

		tokens1, scanErrs1 := lexer.Scan(input, "test.emod")
		require.Empty(t, scanErrs1)
		p1 := parser.New(tokens1, "test.emod")
		ast1, parseErrs1 := p1.Parse()
		require.Empty(t, parseErrs1)

		formatted := formatter.Format(ast1)

		tokens2, scanErrs2 := lexer.Scan(formatted, "test.emod")
		require.Empty(t, scanErrs2)
		p2 := parser.New(tokens2, "test.emod")
		ast2, parseErrs2 := p2.Parse()
		require.Empty(t, parseErrs2)

		ignorePositionsAndComments := cmp.Options{
			cmpopts.IgnoreTypes(ast.Position{}),
			cmpopts.IgnoreFields(ast.Model{}, "Comments"),
			cmpopts.IgnoreFields(ast.Actor{}, "Comments"),
			cmpopts.IgnoreFields(ast.Context{}, "Comments"),
			cmpopts.IgnoreFields(ast.Aggregate{}, "Comments"),
			cmpopts.IgnoreFields(ast.Slice{}, "Comments"),
			cmpopts.IgnoreFields(ast.Command{}, "Comments"),
			cmpopts.IgnoreFields(ast.Event{}, "Comments"),
			cmpopts.IgnoreFields(ast.Flow{}, "Comments"),
			cmpopts.IgnoreFields(ast.Trigger{}, "Comments"),
			cmpopts.IgnoreFields(ast.View{}, "Comments"),
			cmpopts.IgnoreFields(ast.Automation{}, "Comments"),
			cmpopts.IgnoreFields(ast.Translation{}, "Comments"),
		}

		test.RequireEqual(t, ast1, ast2, ignorePositionsAndComments)
	})

	t.Run("blank line between sibling slices", func(t *testing.T) {
		model := &ast.Model{
			Name: "Test",
			Contexts: []*ast.Context{
				{
					Name: "Ctx",
					Aggregates: []*ast.Aggregate{
						{
							Name: "Agg",
							Slices: []*ast.Slice{
								{
									Name: "First Slice",
									Commands: []*ast.Command{
										{Name: "CmdA"},
									},
								},
								{
									Name: "Second Slice",
									Commands: []*ast.Command{
										{Name: "CmdB"},
									},
								},
							},
						},
					},
				},
			},
		}

		result := formatter.Format(model)

		expected := strings.Join([]string{
			`model "Test"`,
			``,
			`context "Ctx" {`,
			`  aggregate "Agg" {`,
			`    slice "First Slice" {`,
			`      command CmdA {`,
			`      }`,
			`    }`,
			``,
			`    slice "Second Slice" {`,
			`      command CmdB {`,
			`      }`,
			`    }`,
			`  }`,
			`}`,
			``,
		}, "\n")

		require.Equal(t, expected, result)
	})

	t.Run("formats multiple actors and contexts with blank lines between them", func(t *testing.T) {
		model := &ast.Model{
			Name: "Test",
			Actors: []*ast.Actor{
				{Name: "Guest"},
				{Name: "Admin"},
			},
			Contexts: []*ast.Context{
				{
					Name: "Orders",
					Aggregates: []*ast.Aggregate{
						{Name: "Order"},
					},
				},
				{
					Name: "Payments",
					Aggregates: []*ast.Aggregate{
						{Name: "Payment"},
					},
				},
			},
		}

		result := formatter.Format(model)

		expected := strings.Join([]string{
			`model "Test"`,
			``,
			`actor "Guest"`,
			``,
			`actor "Admin"`,
			``,
			`context "Orders" {`,
			`  aggregate "Order" {`,
			`  }`,
			`}`,
			``,
			`context "Payments" {`,
			`  aggregate "Payment" {`,
			`  }`,
			`}`,
			``,
		}, "\n")

		require.Equal(t, expected, result)
	})

	t.Run("fields without modifier omit trailing whitespace", func(t *testing.T) {
		model := &ast.Model{
			Name: "Test",
			Contexts: []*ast.Context{
				{
					Name: "Ctx",
					Aggregates: []*ast.Aggregate{
						{
							Name: "Agg",
							Slices: []*ast.Slice{
								{
									Name: "S",
									Commands: []*ast.Command{
										{
											Name: "Cmd",
											Fields: []*ast.Field{
												{Name: "name", Type: "string"},
												{Name: "age", Type: "int", Modifier: "optional"},
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

		result := formatter.Format(model)

		expected := strings.Join([]string{
			`model "Test"`,
			``,
			`context "Ctx" {`,
			`  aggregate "Agg" {`,
			`    slice "S" {`,
			`      command Cmd {`,
			`        fields {`,
			`          name string`,
			`          age  int    optional`,
			`        }`,
			`      }`,
			`    }`,
			`  }`,
			`}`,
			``,
		}, "\n")

		require.Equal(t, expected, result)
	})

	t.Run("element ordering inside slice follows canonical order", func(t *testing.T) {
		model := &ast.Model{
			Name: "Test",
			Contexts: []*ast.Context{
				{
					Name: "Ctx",
					Aggregates: []*ast.Aggregate{
						{
							Name: "Agg",
							Slices: []*ast.Slice{
								{
									Name: "Full Slice",
									Trigger: &ast.Trigger{
										Kind:  "UI",
										Name:  "Form",
										Actor: "User",
									},
									Commands: []*ast.Command{
										{Name: "DoThing"},
									},
									Events: []*ast.Event{
										{Name: "ThingDone"},
									},
									Views: []*ast.View{
										{
											Name:       "ThingView",
											Subscribes: []string{"ThingDone"},
										},
									},
									Automations: []*ast.Automation{
										{
											Name:         "Reactor",
											TriggerEvent: "ThingDone",
											Command:      "Notify",
										},
									},
									Flows: []*ast.Flow{
										{CommandName: "DoThing", EventName: "ThingDone"},
									},
								},
							},
						},
					},
				},
			},
		}

		result := formatter.Format(model)

		triggerIdx := strings.Index(result, "trigger UI")
		commandIdx := strings.Index(result, "command DoThing")
		eventIdx := strings.Index(result, "event ThingDone")
		viewIdx := strings.Index(result, "view ThingView")
		automationIdx := strings.Index(result, "automation Reactor")
		flowIdx := strings.Index(result, "flow {")

		require.Greater(t, commandIdx, triggerIdx, "command should come after trigger")
		require.Greater(t, eventIdx, commandIdx, "event should come after command")
		require.Greater(t, viewIdx, eventIdx, "view should come after event")
		require.Greater(t, automationIdx, viewIdx, "automation should come after view")
		require.Greater(t, flowIdx, automationIdx, "flow should come after automation")
	})

	t.Run("event with source external but empty provider omits source line", func(t *testing.T) {
		model := &ast.Model{
			Name: "Test",
			Contexts: []*ast.Context{
				{
					Name: "Ctx",
					Aggregates: []*ast.Aggregate{
						{
							Name: "Agg",
							Slices: []*ast.Slice{
								{
									Name: "S",
									Events: []*ast.Event{
										{
											Name:   "Evt",
											Source: "external",
											Fields: []*ast.Field{
												{Name: "id", Type: "string"},
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

		result := formatter.Format(model)

		expected := strings.Join([]string{
			`model "Test"`,
			``,
			`context "Ctx" {`,
			`  aggregate "Agg" {`,
			`    slice "S" {`,
			`      event Evt {`,
			`        fields {`,
			`          id string`,
			`        }`,
			`      }`,
			`    }`,
			`  }`,
			`}`,
			``,
		}, "\n")

		require.Equal(t, expected, result)
	})

	t.Run("event with no fields omits fields block", func(t *testing.T) {
		model := &ast.Model{
			Name: "Test",
			Contexts: []*ast.Context{
				{
					Name: "Ctx",
					Aggregates: []*ast.Aggregate{
						{
							Name: "Agg",
							Slices: []*ast.Slice{
								{
									Name: "S",
									Events: []*ast.Event{
										{Name: "ThingHappened"},
									},
								},
							},
						},
					},
				},
			},
		}

		result := formatter.Format(model)

		expected := strings.Join([]string{
			`model "Test"`,
			``,
			`context "Ctx" {`,
			`  aggregate "Agg" {`,
			`    slice "S" {`,
			`      event ThingHappened {`,
			`      }`,
			`    }`,
			`  }`,
			`}`,
			``,
		}, "\n")

		require.Equal(t, expected, result)
	})

	t.Run("view with subscribes but no fields omits fields block", func(t *testing.T) {
		model := &ast.Model{
			Name: "Test",
			Contexts: []*ast.Context{
				{
					Name: "Ctx",
					Aggregates: []*ast.Aggregate{
						{
							Name: "Agg",
							Slices: []*ast.Slice{
								{
									Name: "S",
									Views: []*ast.View{
										{
											Name:       "MyView",
											Subscribes: []string{"EventA"},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		result := formatter.Format(model)

		expected := strings.Join([]string{
			`model "Test"`,
			``,
			`context "Ctx" {`,
			`  aggregate "Agg" {`,
			`    slice "S" {`,
			`      view MyView {`,
			`        subscribes [EventA]`,
			`      }`,
			`    }`,
			`  }`,
			`}`,
			``,
		}, "\n")

		require.Equal(t, expected, result)
	})

	t.Run("field names padded to longest name width within a block", func(t *testing.T) {
		model := &ast.Model{
			Name: "Test",
			Contexts: []*ast.Context{
				{
					Name: "Ctx",
					Aggregates: []*ast.Aggregate{
						{
							Name: "Agg",
							Slices: []*ast.Slice{
								{
									Name: "S",
									Commands: []*ast.Command{
										{
											Name: "Cmd",
											Fields: []*ast.Field{
												{Name: "id", Type: "string", Modifier: "required"},
												{Name: "guestName", Type: "string", Modifier: "required"},
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

		result := formatter.Format(model)

		expected := strings.Join([]string{
			`model "Test"`,
			``,
			`context "Ctx" {`,
			`  aggregate "Agg" {`,
			`    slice "S" {`,
			`      command Cmd {`,
			`        fields {`,
			`          id        string required`,
			`          guestName string required`,
			`        }`,
			`      }`,
			`    }`,
			`  }`,
			`}`,
			``,
		}, "\n")

		require.Equal(t, expected, result)
	})

	t.Run("field types padded to longest type width within a block", func(t *testing.T) {
		model := &ast.Model{
			Name: "Test",
			Contexts: []*ast.Context{
				{
					Name: "Ctx",
					Aggregates: []*ast.Aggregate{
						{
							Name: "Agg",
							Slices: []*ast.Slice{
								{
									Name: "S",
									Events: []*ast.Event{
										{
											Name: "Evt",
											Fields: []*ast.Field{
												{Name: "checkIn", Type: "date", Modifier: "required"},
												{Name: "created", Type: "timestamp", Modifier: "required"},
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

		result := formatter.Format(model)

		expected := strings.Join([]string{
			`model "Test"`,
			``,
			`context "Ctx" {`,
			`  aggregate "Agg" {`,
			`    slice "S" {`,
			`      event Evt {`,
			`        fields {`,
			`          checkIn date      required`,
			`          created timestamp required`,
			`        }`,
			`      }`,
			`    }`,
			`  }`,
			`}`,
			``,
		}, "\n")

		require.Equal(t, expected, result)
	})

	t.Run("different fields blocks are aligned independently", func(t *testing.T) {
		model := &ast.Model{
			Name: "Test",
			Contexts: []*ast.Context{
				{
					Name: "Ctx",
					Aggregates: []*ast.Aggregate{
						{
							Name: "Agg",
							Slices: []*ast.Slice{
								{
									Name: "S",
									Commands: []*ast.Command{
										{
											Name: "Cmd",
											Fields: []*ast.Field{
												{Name: "id", Type: "string", Modifier: "required"},
												{Name: "name", Type: "string", Modifier: "required"},
											},
										},
									},
									Events: []*ast.Event{
										{
											Name: "Evt",
											Fields: []*ast.Field{
												{Name: "reservationId", Type: "string", Modifier: "required"},
												{Name: "at", Type: "timestamp", Modifier: "required"},
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

		result := formatter.Format(model)

		expected := strings.Join([]string{
			`model "Test"`,
			``,
			`context "Ctx" {`,
			`  aggregate "Agg" {`,
			`    slice "S" {`,
			`      command Cmd {`,
			`        fields {`,
			`          id   string required`,
			`          name string required`,
			`        }`,
			`      }`,
			``,
			`      event Evt {`,
			`        fields {`,
			`          reservationId string    required`,
			`          at            timestamp required`,
			`        }`,
			`      }`,
			`    }`,
			`  }`,
			`}`,
			``,
		}, "\n")

		require.Equal(t, expected, result)
	})

	t.Run("single-field block produces no extra padding", func(t *testing.T) {
		model := &ast.Model{
			Name: "Test",
			Contexts: []*ast.Context{
				{
					Name: "Ctx",
					Aggregates: []*ast.Aggregate{
						{
							Name: "Agg",
							Slices: []*ast.Slice{
								{
									Name: "S",
									Commands: []*ast.Command{
										{
											Name: "Cmd",
											Fields: []*ast.Field{
												{Name: "id", Type: "string", Modifier: "required"},
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

		result := formatter.Format(model)

		expected := strings.Join([]string{
			`model "Test"`,
			``,
			`context "Ctx" {`,
			`  aggregate "Agg" {`,
			`    slice "S" {`,
			`      command Cmd {`,
			`        fields {`,
			`          id string required`,
			`        }`,
			`      }`,
			`    }`,
			`  }`,
			`}`,
			``,
		}, "\n")

		require.Equal(t, expected, result)
	})

	t.Run("mixed modifier and no-modifier fields aligned correctly", func(t *testing.T) {
		model := &ast.Model{
			Name: "Test",
			Contexts: []*ast.Context{
				{
					Name: "Ctx",
					Aggregates: []*ast.Aggregate{
						{
							Name: "Agg",
							Slices: []*ast.Slice{
								{
									Name: "S",
									Commands: []*ast.Command{
										{
											Name: "Cmd",
											Fields: []*ast.Field{
												{Name: "firstName", Type: "string", Modifier: "required"},
												{Name: "age", Type: "int"},
												{Name: "email", Type: "string", Modifier: "optional"},
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

		result := formatter.Format(model)

		expected := strings.Join([]string{
			`model "Test"`,
			``,
			`context "Ctx" {`,
			`  aggregate "Agg" {`,
			`    slice "S" {`,
			`      command Cmd {`,
			`        fields {`,
			`          firstName string required`,
			`          age       int`,
			`          email     string optional`,
			`        }`,
			`      }`,
			`    }`,
			`  }`,
			`}`,
			``,
		}, "\n")

		require.Equal(t, expected, result)
	})

	t.Run("round-trip: all_patterns.emod field blocks match fixture alignment", func(t *testing.T) {
		raw, err := os.ReadFile("../../internal/parser/testdata/all_patterns.emod")
		require.NoError(t, err)
		input := string(raw)

		tokens, scanErrs := lexer.Scan(input, "all_patterns.emod")
		require.Empty(t, scanErrs)
		p := parser.New(tokens, "all_patterns.emod")
		model, parseErrs := p.Parse()
		require.Empty(t, parseErrs)

		formatted := formatter.Format(model)

		tokens2, scanErrs2 := lexer.Scan(formatted, "all_patterns.emod")
		require.Empty(t, scanErrs2)
		p2 := parser.New(tokens2, "all_patterns.emod")
		model2, parseErrs2 := p2.Parse()
		require.Empty(t, parseErrs2)

		ignorePositionsAndComments := cmp.Options{
			cmpopts.IgnoreTypes(ast.Position{}),
			cmpopts.IgnoreFields(ast.Model{}, "Comments"),
			cmpopts.IgnoreFields(ast.Actor{}, "Comments"),
			cmpopts.IgnoreFields(ast.Context{}, "Comments"),
			cmpopts.IgnoreFields(ast.Aggregate{}, "Comments"),
			cmpopts.IgnoreFields(ast.Slice{}, "Comments"),
			cmpopts.IgnoreFields(ast.Command{}, "Comments"),
			cmpopts.IgnoreFields(ast.Event{}, "Comments"),
			cmpopts.IgnoreFields(ast.Flow{}, "Comments"),
			cmpopts.IgnoreFields(ast.Trigger{}, "Comments"),
			cmpopts.IgnoreFields(ast.View{}, "Comments"),
			cmpopts.IgnoreFields(ast.Automation{}, "Comments"),
			cmpopts.IgnoreFields(ast.Translation{}, "Comments"),
		}

		test.RequireEqual(t, model, model2, ignorePositionsAndComments)
	})

	t.Run("view with fields but no subscribes omits subscribes line", func(t *testing.T) {
		model := &ast.Model{
			Name: "Test",
			Contexts: []*ast.Context{
				{
					Name: "Ctx",
					Aggregates: []*ast.Aggregate{
						{
							Name: "Agg",
							Slices: []*ast.Slice{
								{
									Name: "S",
									Views: []*ast.View{
										{
											Name: "MyView",
											Fields: []*ast.Field{
												{Name: "id", Type: "string"},
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

		result := formatter.Format(model)

		expected := strings.Join([]string{
			`model "Test"`,
			``,
			`context "Ctx" {`,
			`  aggregate "Agg" {`,
			`    slice "S" {`,
			`      view MyView {`,
			`        fields {`,
			`          id string`,
			`        }`,
			`      }`,
			`    }`,
			`  }`,
			`}`,
			``,
		}, "\n")

		require.Equal(t, expected, result)
	})
}

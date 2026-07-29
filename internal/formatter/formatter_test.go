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
	t.Run("version header", func(t *testing.T) {
		t.Run("a version the model declared is preserved instead of being moved to the supported version", func(t *testing.T) {
			model := &ast.Model{Version: 7, VersionDeclared: true, Name: "Hotel Reservation"}

			result := formatter.Format(model)

			require.Equal(t, "emod 7\nmodel \"Hotel Reservation\"\n", result)
		})
	})

	t.Run("element formatting", func(t *testing.T) {
		t.Run("formats model and actor declarations", func(t *testing.T) {
			model := &ast.Model{
				Name: "Test",
				Actors: []*ast.Actor{
					{Name: "Guest"},
				},
			}

			result := formatter.Format(model)

			expected := "emod 1\nmodel \"Test\"\n\nactor \"Guest\"\n"
			require.Equal(t, expected, result)
		})

		t.Run("a described model or actor takes the block form while an undescribed actor stays on one line", func(t *testing.T) {
			model := &ast.Model{
				Name:        "Hotel Reservation",
				Description: "How the hotel takes and confirms room bookings",
				Actors: []*ast.Actor{
					{Name: "Guest", Description: "A person booking a room"},
					{Name: "Clerk"},
				},
			}

			result := formatter.Format(model)

			expected := strings.Join([]string{
				`emod 1`,
				`model "Hotel Reservation" {`,
				`  description "How the hotel takes and confirms room bookings"`,
				`}`,
				``,
				`actor "Guest" {`,
				`  description "A person booking a room"`,
				`}`,
				``,
				`actor "Clerk"`,
				``,
			}, "\n")

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
											{
												Comments:    []*ast.Comment{{Text: "# main flow"}},
												CommandName: "MakeReservation",
												EventName:   "ReservationMade",
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
				`emod 1`,
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
				`        # main flow`,
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
				`emod 1`,
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
				`emod 1`,
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
				`emod 1`,
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
				`emod 1`,
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
				`emod 1`,
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
	})

	t.Run("round-trip through the parser", func(t *testing.T) {
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

			original := parseModel(t, input, "test.emod")

			reparsed := parseModel(t, formatter.Format(original), "test.emod")

			test.RequireEqual(t, original, reparsed, ignoreFormatterNormalizations)
			require.True(t, reparsed.VersionDeclared, "formatted output should pin the version")
		})

		t.Run("round-trip: a field without a modifier survives formatting", func(t *testing.T) {
			input := strings.Join([]string{
				`model "Hotel Reservation"`,
				``,
				`context "Reservations" {`,
				`  aggregate "Reservation" {`,
				`    slice "Make Reservation" {`,
				`      command MakeReservation {`,
				`        fields {`,
				`          roomType string`,
				`          guestId string required`,
				`        }`,
				`      }`,
				`    }`,
				`  }`,
				`}`,
				``,
			}, "\n")

			original := parseModel(t, input, "test.emod")
			firstFormat := formatter.Format(original)
			reparsed := parseModel(t, firstFormat, "formatted.emod")

			test.RequireEqual(t, original, reparsed, ignoreFormatterNormalizations)
			test.RequireEqual(t, []*ast.Field{
				{Name: "roomType", Type: "string"},
				{Name: "guestId", Type: "string", Modifier: "required"},
			}, reparsed.Contexts[0].Aggregates[0].Slices[0].Commands[0].Fields, ignoreFormatterNormalizations)
			require.Equal(t, firstFormat, formatter.Format(reparsed),
				"a second format run should produce identical bytes")
		})

		t.Run("round-trip: a description on every construct survives formatting", func(t *testing.T) {
			original := parseModel(t, test.DescribedHotelReservation, "described.emod")

			reparsed := parseModel(t, formatter.Format(original), "described.emod")

			test.RequireEqual(t, original, reparsed, ignoreFormatterNormalizations)
		})

		t.Run("round-trip: a description survives text that quoting could mangle", func(t *testing.T) {
			for _, testCase := range []struct {
				hazard      string
				description string
			}{
				{"backslash", `Runs on the clerk's laptop under C:\hotel\rates`},
				{"tab", "Two columns:\troom then rate"},
				{"newline", "Two lines:\nroom then rate"},
				{"percent", "Takes a 10% deposit up front"},
			} {
				t.Run(testCase.hazard, func(t *testing.T) {
					source := "model \"Hotel\" {\n  description \"" + testCase.description + "\"\n}\n"

					original := parseModel(t, source, "described.emod")
					formatted := formatter.Format(original)
					reparsed := parseModel(t, formatted, "described.emod")

					require.Equal(t, testCase.description, reparsed.Description)
					require.Equal(t, formatted, formatter.Format(reparsed),
						"a second format run should not re-encode the description")
				})
			}
		})

		t.Run("idempotency: format(format(described input)) equals format(described input)", func(t *testing.T) {
			original := parseModel(t, test.DescribedHotelReservation, "described.emod")

			firstFormat := formatter.Format(original)
			secondFormat := formatter.Format(parseModel(t, firstFormat, "formatted.emod"))

			require.Equal(t, firstFormat, secondFormat,
				"formatting the already-formatted output should produce identical bytes")
		})
	})

	t.Run("blank lines and ordering", func(t *testing.T) {
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
				`emod 1`,
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
				`emod 1`,
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
				`emod 1`,
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

		t.Run("canonical order puts description first inside every block and slice elements in pattern order", func(t *testing.T) {
			model := &ast.Model{
				Name: "Test",
				Contexts: []*ast.Context{
					{
						Name:        "Ctx",
						Description: "Where things get done",
						Mode:        "mixed",
						Aggregates: []*ast.Aggregate{
							{
								Name:        "Agg",
								Description: "One thing and its history",
								Slices: []*ast.Slice{
									{
										Name:        "Full Slice",
										Description: "A user does the thing",
										Trigger: &ast.Trigger{
											Kind:        "UI",
											Name:        "Form",
											Description: "The form the user fills in",
											Actor:       "User",
											Reads:       "ThingView",
										},
										Commands: []*ast.Command{
											{
												Name:        "DoThing",
												Description: "Ask for the thing to be done",
												DecidesOn: &ast.DecidesOnClause{
													Events:    []string{"ThingDone"},
													Predicate: &ast.TagPredicate{Field: "thing", Operator: "=", Value: "thingId"},
												},
												Fields: []*ast.Field{
													{Name: "thingId", Type: "string", Modifier: "required"},
												},
											},
										},
										Events: []*ast.Event{
											{
												Name:        "ThingDone",
												Description: "The thing was done",
												Tags: []ast.TagEntry{
													{Key: "thing", FieldRef: "thingId"},
												},
												Fields: []*ast.Field{
													{Name: "thingId", Type: "string", Modifier: "required"},
												},
											},
										},
										Views: []*ast.View{
											{
												Name:        "ThingView",
												Description: "Every thing and whether it is done",
												Fields: []*ast.Field{
													{Name: "thingId", Type: "string", Modifier: "required"},
												},
												Subscribes: []string{"ThingDone"},
											},
										},
										Automations: []*ast.Automation{
											{
												Name:          "Reactor",
												Description:   "Notifies whoever asked once the thing is done",
												TriggerEvent:  "ThingDone",
												Command:       "Notify",
												TargetContext: "Notifications",
											},
										},
										Translations: []*ast.Translation{
											{
												Name:           "Importer",
												Description:    "Restates a partner report as a thing",
												ExternalSystem: "Partner",
												Reads:          "ThingView",
												Command:        "DoThing",
												Event: &ast.Event{
													Name:        "ThingImported",
													Description: "A partner reported a thing",
													Tags: []ast.TagEntry{
														{Key: "thing", FieldRef: "thingId"},
													},
													Fields: []*ast.Field{
														{Name: "thingId", Type: "string", Modifier: "required"},
													},
												},
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
			translationIdx := strings.Index(result, "translation Importer")
			flowIdx := strings.Index(result, "flow {")

			require.Greater(t, commandIdx, triggerIdx, "command should come after trigger")
			require.Greater(t, eventIdx, commandIdx, "event should come after command")
			require.Greater(t, viewIdx, eventIdx, "view should come after event")
			require.Greater(t, automationIdx, viewIdx, "automation should come after view")
			require.Greater(t, translationIdx, automationIdx, "translation should come after automation")
			require.Greater(t, flowIdx, translationIdx, "flow should come after translation")

			require.Equal(t, `description "Where things get done"`, lineAfter(t, result, `context "Ctx" mode mixed {`))
			require.Equal(t, `description "One thing and its history"`, lineAfter(t, result, `aggregate "Agg" {`))
			require.Equal(t, `description "A user does the thing"`, lineAfter(t, result, `slice "Full Slice" {`))
			require.Equal(t, `description "The form the user fills in"`, lineAfter(t, result, `trigger UI "Form" {`))
			require.Equal(t, `description "Ask for the thing to be done"`, lineAfter(t, result, `command DoThing {`))
			require.Equal(t, `description "The thing was done"`, lineAfter(t, result, `event ThingDone {`))
			require.Equal(t, `description "Every thing and whether it is done"`, lineAfter(t, result, `view ThingView {`))
			require.Equal(t, `description "Notifies whoever asked once the thing is done"`, lineAfter(t, result, `automation Reactor {`))
			require.Equal(t, `description "Restates a partner report as a thing"`, lineAfter(t, result, `translation Importer {`))
			require.Equal(t, `description "A partner reported a thing"`, lineAfter(t, result, `event ThingImported {`))
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
				`emod 1`,
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
				`emod 1`,
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
				`emod 1`,
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
	})

	t.Run("field alignment", func(t *testing.T) {
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
				`emod 1`,
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
				`emod 1`,
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
				`emod 1`,
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
				`emod 1`,
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
				`emod 1`,
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
			original := parseFixture(t, "all_patterns.emod")

			reparsed := parseModel(t, formatter.Format(original), "all_patterns.emod")

			test.RequireEqual(t, original, reparsed, ignoreFormatterNormalizations)
		})
	})

	t.Run("comments", func(t *testing.T) {
		t.Run("single comment before model declaration at root level", func(t *testing.T) {
			model := &ast.Model{
				Comments: []*ast.Comment{{Text: "# System description"}},
				Name:     "Test",
			}

			result := formatter.Format(model)

			expected := "emod 1\n# System description\nmodel \"Test\"\n"
			require.Equal(t, expected, result)
		})

		t.Run("comment before slice at nested indentation level", func(t *testing.T) {
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
										Comments: []*ast.Comment{{Text: "# Important slice"}},
										Name:     "My Slice",
										Commands: []*ast.Command{
											{Name: "Cmd"},
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
				`emod 1`,
				`model "Test"`,
				``,
				`context "Ctx" {`,
				`  aggregate "Agg" {`,
				`    # Important slice`,
				`    slice "My Slice" {`,
				`      command Cmd {`,
				`      }`,
				`    }`,
				`  }`,
				`}`,
				``,
			}, "\n")

			require.Equal(t, expected, result)
		})

		t.Run("multiple consecutive comments before a single node", func(t *testing.T) {
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
										Comments: []*ast.Comment{
											{Text: "# First comment"},
											{Text: "# Second comment"},
											{Text: "# Third comment"},
										},
										Name: "My Slice",
										Commands: []*ast.Command{
											{Name: "Cmd"},
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
				`emod 1`,
				`model "Test"`,
				``,
				`context "Ctx" {`,
				`  aggregate "Agg" {`,
				`    # First comment`,
				`    # Second comment`,
				`    # Third comment`,
				`    slice "My Slice" {`,
				`      command Cmd {`,
				`      }`,
				`    }`,
				`  }`,
				`}`,
				``,
			}, "\n")

			require.Equal(t, expected, result)
		})

		t.Run("round-trip: formatting all_patterns.emod preserves all comments", func(t *testing.T) {
			original := parseFixture(t, "all_patterns.emod")

			reparsed := parseModel(t, formatter.Format(original), "all_patterns.emod")

			require.Equal(t,
				"# Hotel Reservation System — exercises all four slice patterns",
				reparsed.Comments[0].Text,
			)

			slices := reparsed.Contexts[0].Aggregates[0].Slices
			require.Equal(t, "# Slice 1: Command Pattern", slices[0].Comments[0].Text)
			require.Equal(t, "# Slice 2: View Pattern", slices[1].Comments[0].Text)
			require.Equal(t, "# Slice 3: Command Pattern — Check Out", slices[2].Comments[0].Text)
			require.Equal(t, "# Slice 4: Automation Pattern", slices[3].Comments[0].Text)
			require.Equal(t, "# Slice 5: Translation Pattern", slices[4].Comments[0].Text)
		})

		t.Run("idempotency: format(format(input)) equals format(input)", func(t *testing.T) {
			original := parseFixture(t, "all_patterns.emod")

			firstFormat := formatter.Format(original)
			secondFormat := formatter.Format(parseModel(t, firstFormat, "formatted.emod"))

			require.Equal(t, firstFormat, secondFormat,
				"formatting the already-formatted output should produce identical bytes")
		})

		t.Run("comment on deeply nested command inside slice", func(t *testing.T) {
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
										Commands: []*ast.Command{
											{
												Comments: []*ast.Comment{{Text: "# Command comment"}},
												Name:     "DoThing",
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
				`emod 1`,
				`model "Test"`,
				``,
				`context "Ctx" {`,
				`  aggregate "Agg" {`,
				`    slice "My Slice" {`,
				`      # Command comment`,
				`      command DoThing {`,
				`      }`,
				`    }`,
				`  }`,
				`}`,
				``,
			}, "\n")

			require.Equal(t, expected, result)
		})
	})

	t.Run("blank line normalisation", func(t *testing.T) {
		t.Run("zero blank lines between sibling slices are expanded to exactly one", func(t *testing.T) {
			input := strings.Join([]string{
				`model "Hotel"`,
				``,
				`context "Reservations" {`,
				`  aggregate "Reservation" {`,
				`    slice "Make Reservation" {`,
				`      command MakeReservation {`,
				`      }`,
				`    }`,
				`    slice "Cancel Reservation" {`,
				`      command CancelReservation {`,
				`      }`,
				`    }`,
				`  }`,
				`}`,
				``,
			}, "\n")

			expected := strings.Join([]string{
				`emod 1`,
				`model "Hotel"`,
				``,
				`context "Reservations" {`,
				`  aggregate "Reservation" {`,
				`    slice "Make Reservation" {`,
				`      command MakeReservation {`,
				`      }`,
				`    }`,
				``,
				`    slice "Cancel Reservation" {`,
				`      command CancelReservation {`,
				`      }`,
				`    }`,
				`  }`,
				`}`,
				``,
			}, "\n")

			model := parseModel(t, input, "test.emod")

			result := formatter.Format(model)

			require.Equal(t, expected, result)
		})

		t.Run("excessive blank lines between sibling slices are reduced to exactly one", func(t *testing.T) {
			input := strings.Join([]string{
				`model "Hotel"`,
				``,
				`context "Reservations" {`,
				`  aggregate "Reservation" {`,
				`    slice "Make Reservation" {`,
				`      command MakeReservation {`,
				`      }`,
				`    }`,
				``,
				``,
				``,
				``,
				`    slice "Cancel Reservation" {`,
				`      command CancelReservation {`,
				`      }`,
				`    }`,
				`  }`,
				`}`,
				``,
			}, "\n")

			expected := strings.Join([]string{
				`emod 1`,
				`model "Hotel"`,
				``,
				`context "Reservations" {`,
				`  aggregate "Reservation" {`,
				`    slice "Make Reservation" {`,
				`      command MakeReservation {`,
				`      }`,
				`    }`,
				``,
				`    slice "Cancel Reservation" {`,
				`      command CancelReservation {`,
				`      }`,
				`    }`,
				`  }`,
				`}`,
				``,
			}, "\n")

			model := parseModel(t, input, "test.emod")

			result := formatter.Format(model)

			require.Equal(t, expected, result)
		})

		t.Run("no leading blank line before the first slice in an aggregate", func(t *testing.T) {
			input := strings.Join([]string{
				`model "Hotel"`,
				``,
				`context "Reservations" {`,
				`  aggregate "Reservation" {`,
				``,
				``,
				`    slice "Make Reservation" {`,
				`      command MakeReservation {`,
				`      }`,
				`    }`,
				`  }`,
				`}`,
				``,
			}, "\n")

			expected := strings.Join([]string{
				`emod 1`,
				`model "Hotel"`,
				``,
				`context "Reservations" {`,
				`  aggregate "Reservation" {`,
				`    slice "Make Reservation" {`,
				`      command MakeReservation {`,
				`      }`,
				`    }`,
				`  }`,
				`}`,
				``,
			}, "\n")

			model := parseModel(t, input, "test.emod")

			result := formatter.Format(model)

			require.Equal(t, expected, result)
		})

		t.Run("no trailing blank line after the last slice in an aggregate", func(t *testing.T) {
			input := strings.Join([]string{
				`model "Hotel"`,
				``,
				`context "Reservations" {`,
				`  aggregate "Reservation" {`,
				`    slice "Make Reservation" {`,
				`      command MakeReservation {`,
				`      }`,
				`    }`,
				``,
				``,
				`  }`,
				`}`,
				``,
			}, "\n")

			expected := strings.Join([]string{
				`emod 1`,
				`model "Hotel"`,
				``,
				`context "Reservations" {`,
				`  aggregate "Reservation" {`,
				`    slice "Make Reservation" {`,
				`      command MakeReservation {`,
				`      }`,
				`    }`,
				`  }`,
				`}`,
				``,
			}, "\n")

			model := parseModel(t, input, "test.emod")

			result := formatter.Format(model)

			require.Equal(t, expected, result)
		})

		t.Run("zero blank lines between top-level declarations are expanded to exactly one", func(t *testing.T) {
			input := strings.Join([]string{
				`model "Hotel"`,
				`actor "Guest"`,
				`context "Reservations" {`,
				`  aggregate "Reservation" {`,
				`  }`,
				`}`,
				`context "Billing" {`,
				`  aggregate "Invoice" {`,
				`  }`,
				`}`,
				``,
			}, "\n")

			expected := strings.Join([]string{
				`emod 1`,
				`model "Hotel"`,
				``,
				`actor "Guest"`,
				``,
				`context "Reservations" {`,
				`  aggregate "Reservation" {`,
				`  }`,
				`}`,
				``,
				`context "Billing" {`,
				`  aggregate "Invoice" {`,
				`  }`,
				`}`,
				``,
			}, "\n")

			model := parseModel(t, input, "test.emod")

			result := formatter.Format(model)

			require.Equal(t, expected, result)
		})

		t.Run("excessive blank lines between top-level declarations are reduced to exactly one", func(t *testing.T) {
			input := strings.Join([]string{
				`model "Hotel"`,
				``,
				``,
				``,
				``,
				`actor "Guest"`,
				``,
				``,
				``,
				``,
				`context "Reservations" {`,
				`  aggregate "Reservation" {`,
				`  }`,
				`}`,
				``,
			}, "\n")

			expected := strings.Join([]string{
				`emod 1`,
				`model "Hotel"`,
				``,
				`actor "Guest"`,
				``,
				`context "Reservations" {`,
				`  aggregate "Reservation" {`,
				`  }`,
				`}`,
				``,
			}, "\n")

			model := parseModel(t, input, "test.emod")

			result := formatter.Format(model)

			require.Equal(t, expected, result)
		})

		t.Run("idempotency on already-normalized multi-slice input", func(t *testing.T) {
			input := strings.Join([]string{
				`emod 1`,
				`model "Hotel"`,
				``,
				`actor "Guest"`,
				``,
				`context "Reservations" {`,
				`  aggregate "Reservation" {`,
				`    slice "Make Reservation" {`,
				`      command MakeReservation {`,
				`      }`,
				`    }`,
				``,
				`    slice "Cancel Reservation" {`,
				`      command CancelReservation {`,
				`      }`,
				`    }`,
				`  }`,
				`}`,
				``,
			}, "\n")

			model := parseModel(t, input, "test.emod")

			result := formatter.Format(model)

			require.Equal(t, input, result)
		})
	})

	t.Run("context modes", func(t *testing.T) {
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
				`emod 1`,
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

		t.Run("context with mode dcb emits mode clause", func(t *testing.T) {
			model := &ast.Model{
				Name: "Test",
				Contexts: []*ast.Context{
					{
						Name: "Ctx",
						Mode: "dcb",
						Slices: []*ast.Slice{
							{Name: "My Slice"},
						},
					},
				},
			}

			result := formatter.Format(model)

			expected := strings.Join([]string{
				`emod 1`,
				`model "Test"`,
				``,
				`context "Ctx" mode dcb {`,
				`  slice "My Slice" {`,
				`  }`,
				`}`,
				``,
			}, "\n")
			require.Equal(t, expected, result)
		})

		t.Run("context with mode aggregate emits mode clause", func(t *testing.T) {
			model := &ast.Model{
				Name: "Test",
				Contexts: []*ast.Context{
					{
						Name: "Ctx",
						Mode: "aggregate",
						Aggregates: []*ast.Aggregate{
							{Name: "Agg"},
						},
					},
				},
			}

			result := formatter.Format(model)

			expected := strings.Join([]string{
				`emod 1`,
				`model "Test"`,
				``,
				`context "Ctx" mode aggregate {`,
				`  aggregate "Agg" {`,
				`  }`,
				`}`,
				``,
			}, "\n")
			require.Equal(t, expected, result)
		})

		t.Run("context with mode mixed emits mode clause with both aggregates and slices", func(t *testing.T) {
			model := &ast.Model{
				Name: "Test",
				Contexts: []*ast.Context{
					{
						Name: "Ctx",
						Mode: "mixed",
						Aggregates: []*ast.Aggregate{
							{Name: "Agg"},
						},
						Slices: []*ast.Slice{
							{Name: "Direct Slice"},
						},
					},
				},
			}

			result := formatter.Format(model)

			expected := strings.Join([]string{
				`emod 1`,
				`model "Test"`,
				``,
				`context "Ctx" mode mixed {`,
				`  aggregate "Agg" {`,
				`  }`,
				``,
				`  slice "Direct Slice" {`,
				`  }`,
				`}`,
				``,
			}, "\n")
			require.Equal(t, expected, result)
		})

		t.Run("context without mode omits mode clause (backward compatible)", func(t *testing.T) {
			model := &ast.Model{
				Name: "Test",
				Contexts: []*ast.Context{
					{
						Name: "Ctx",
						Aggregates: []*ast.Aggregate{
							{Name: "Agg"},
						},
					},
				},
			}

			result := formatter.Format(model)

			expected := strings.Join([]string{
				`emod 1`,
				`model "Test"`,
				``,
				`context "Ctx" {`,
				`  aggregate "Agg" {`,
				`  }`,
				`}`,
				``,
			}, "\n")
			require.Equal(t, expected, result)
		})

		t.Run("slices directly under context are formatted at context body level", func(t *testing.T) {
			model := &ast.Model{
				Name: "Test",
				Contexts: []*ast.Context{
					{
						Name: "Ctx",
						Mode: "dcb",
						Slices: []*ast.Slice{
							{
								Name: "First Slice",
								Commands: []*ast.Command{
									{Name: "DoThing"},
								},
							},
							{
								Name: "Second Slice",
								Commands: []*ast.Command{
									{Name: "DoOther"},
								},
							},
						},
					},
				},
			}

			result := formatter.Format(model)

			expected := strings.Join([]string{
				`emod 1`,
				`model "Test"`,
				``,
				`context "Ctx" mode dcb {`,
				`  slice "First Slice" {`,
				`    command DoThing {`,
				`    }`,
				`  }`,
				``,
				`  slice "Second Slice" {`,
				`    command DoOther {`,
				`    }`,
				`  }`,
				`}`,
				``,
			}, "\n")
			require.Equal(t, expected, result)
		})
	})

	t.Run("event tags", func(t *testing.T) {
		t.Run("event with tags block", func(t *testing.T) {
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
												Name: "TestEvent",
												Tags: []ast.TagEntry{
													{Key: "priority", FieldRef: "statusCode"},
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
				`emod 1`,
				`model "Test"`,
				``,
				`context "Ctx" {`,
				`  aggregate "Agg" {`,
				`    slice "S" {`,
				`      event TestEvent {`,
				`        tags {`,
				`          priority: statusCode`,
				`        }`,
				`      }`,
				`    }`,
				`  }`,
				`}`,
				``,
			}, "\n")
			require.Equal(t, expected, result)
		})

		t.Run("event with tags and fields", func(t *testing.T) {
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
												Name: "TestEvent",
												Tags: []ast.TagEntry{
													{Key: "priority", FieldRef: "statusCode"},
												},
												Fields: []*ast.Field{
													{Name: "statusCode", Type: "string"},
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
				`emod 1`,
				`model "Test"`,
				``,
				`context "Ctx" {`,
				`  aggregate "Agg" {`,
				`    slice "S" {`,
				`      event TestEvent {`,
				`        tags {`,
				`          priority: statusCode`,
				`        }`,
				`        fields {`,
				`          statusCode string`,
				`        }`,
				`      }`,
				`    }`,
				`  }`,
				`}`,
				``,
			}, "\n")
			require.Equal(t, expected, result)
		})

		t.Run("event with tags and source external", func(t *testing.T) {
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
												Name:         "WebhookEvent",
												Source:       "external",
												ExternalName: "SendGrid",
												Tags: []ast.TagEntry{
													{Key: "priority", FieldRef: "statusCode"},
												},
												Fields: []*ast.Field{
													{Name: "statusCode", Type: "string"},
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
				`emod 1`,
				`model "Test"`,
				``,
				`context "Ctx" {`,
				`  aggregate "Agg" {`,
				`    slice "S" {`,
				`      event WebhookEvent {`,
				`        tags {`,
				`          priority: statusCode`,
				`        }`,
				`        source external "SendGrid"`,
				`        fields {`,
				`          statusCode string`,
				`        }`,
				`      }`,
				`    }`,
				`  }`,
				`}`,
				``,
			}, "\n")
			require.Equal(t, expected, result)
		})

		t.Run("event with multiple tags aligned correctly", func(t *testing.T) {
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
												Name: "TestEvent",
												Tags: []ast.TagEntry{
													{Key: "priority", FieldRef: "statusCode"},
													{Key: "category", FieldRef: "eventType"},
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
				`emod 1`,
				`model "Test"`,
				``,
				`context "Ctx" {`,
				`  aggregate "Agg" {`,
				`    slice "S" {`,
				`      event TestEvent {`,
				`        tags {`,
				`          priority: statusCode`,
				`          category: eventType`,
				`        }`,
				`      }`,
				`    }`,
				`  }`,
				`}`,
				``,
			}, "\n")
			require.Equal(t, expected, result)
		})
	})

	t.Run("decides_on", func(t *testing.T) {
		t.Run("command with decides_on block", func(t *testing.T) {
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
												Name: "DoThing",
												DecidesOn: &ast.DecidesOnClause{
													Events:    []string{"ThingDone"},
													Predicate: &ast.TagPredicate{Field: "priority", Operator: "=", Value: "high"},
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
				`emod 1`,
				`model "Test"`,
				``,
				`context "Ctx" {`,
				`  aggregate "Agg" {`,
				`    slice "S" {`,
				`      command DoThing {`,
				`        decides_on {`,
				`          events [ThingDone]`,
				`          where tag(priority = high)`,
				`        }`,
				`      }`,
				`    }`,
				`  }`,
				`}`,
				``,
			}, "\n")
			require.Equal(t, expected, result)
		})

		t.Run("command with decides_on and fields", func(t *testing.T) {
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
												Name: "DoThing",
												DecidesOn: &ast.DecidesOnClause{
													Events:    []string{"ThingDone"},
													Predicate: &ast.TagPredicate{Field: "priority", Operator: "=", Value: "high"},
												},
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
				`emod 1`,
				`model "Test"`,
				``,
				`context "Ctx" {`,
				`  aggregate "Agg" {`,
				`    slice "S" {`,
				`      command DoThing {`,
				`        decides_on {`,
				`          events [ThingDone]`,
				`          where tag(priority = high)`,
				`        }`,
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

		t.Run("command with decides_on and multiple events", func(t *testing.T) {
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
												Name: "DoThing",
												DecidesOn: &ast.DecidesOnClause{
													Events:    []string{"EventA", "EventB", "EventC"},
													Predicate: &ast.TagPredicate{Field: "priority", Operator: "=", Value: "high"},
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
				`emod 1`,
				`model "Test"`,
				``,
				`context "Ctx" {`,
				`  aggregate "Agg" {`,
				`    slice "S" {`,
				`      command DoThing {`,
				`        decides_on {`,
				`          events [EventA, EventB, EventC]`,
				`          where tag(priority = high)`,
				`        }`,
				`      }`,
				`    }`,
				`  }`,
				`}`,
				``,
			}, "\n")
			require.Equal(t, expected, result)
		})

		t.Run("command with decides_on and compound predicate (and)", func(t *testing.T) {
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
												Name: "DoThing",
												DecidesOn: &ast.DecidesOnClause{
													Events: []string{"ThingDone"},
													Predicate: &ast.LogicalExpr{
														Left:     &ast.TagPredicate{Field: "priority", Operator: "=", Value: "high"},
														Operator: "and",
														Right:    &ast.TagPredicate{Field: "region", Operator: "=", Value: "us"},
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
				`emod 1`,
				`model "Test"`,
				``,
				`context "Ctx" {`,
				`  aggregate "Agg" {`,
				`    slice "S" {`,
				`      command DoThing {`,
				`        decides_on {`,
				`          events [ThingDone]`,
				`          where tag(priority = high) and tag(region = us)`,
				`        }`,
				`      }`,
				`    }`,
				`  }`,
				`}`,
				``,
			}, "\n")
			require.Equal(t, expected, result)
		})

		t.Run("command with decides_on and compound predicate (or)", func(t *testing.T) {
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
												Name: "DoThing",
												DecidesOn: &ast.DecidesOnClause{
													Events: []string{"ThingDone"},
													Predicate: &ast.LogicalExpr{
														Left:     &ast.TagPredicate{Field: "priority", Operator: "=", Value: "high"},
														Operator: "or",
														Right:    &ast.TagPredicate{Field: "priority", Operator: "=", Value: "low"},
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
				`emod 1`,
				`model "Test"`,
				``,
				`context "Ctx" {`,
				`  aggregate "Agg" {`,
				`    slice "S" {`,
				`      command DoThing {`,
				`        decides_on {`,
				`          events [ThingDone]`,
				`          where tag(priority = high) or tag(priority = low)`,
				`        }`,
				`      }`,
				`    }`,
				`  }`,
				`}`,
				``,
			}, "\n")
			require.Equal(t, expected, result)
		})

		t.Run("command with decides_on and not predicate", func(t *testing.T) {
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
												Name: "DoThing",
												DecidesOn: &ast.DecidesOnClause{
													Events: []string{"ThingDone"},
													Predicate: &ast.NotExpr{
														Expr: &ast.TagPredicate{Field: "priority", Operator: "=", Value: "high"},
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
				`emod 1`,
				`model "Test"`,
				``,
				`context "Ctx" {`,
				`  aggregate "Agg" {`,
				`    slice "S" {`,
				`      command DoThing {`,
				`        decides_on {`,
				`          events [ThingDone]`,
				`          where not tag(priority = high)`,
				`        }`,
				`      }`,
				`    }`,
				`  }`,
				`}`,
				``,
			}, "\n")
			require.Equal(t, expected, result)
		})

		t.Run("command with decides_on and parenthesised sub-expression", func(t *testing.T) {
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
												Name: "DoThing",
												DecidesOn: &ast.DecidesOnClause{
													Events: []string{"ThingDone"},
													Predicate: &ast.LogicalExpr{
														Left: &ast.LogicalExpr{
															Left:     &ast.TagPredicate{Field: "priority", Operator: "=", Value: "high"},
															Operator: "or",
															Right:    &ast.TagPredicate{Field: "priority", Operator: "=", Value: "low"},
														},
														Operator: "and",
														Right:    &ast.TagPredicate{Field: "region", Operator: "=", Value: "us"},
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
				`emod 1`,
				`model "Test"`,
				``,
				`context "Ctx" {`,
				`  aggregate "Agg" {`,
				`    slice "S" {`,
				`      command DoThing {`,
				`        decides_on {`,
				`          events [ThingDone]`,
				`          where (tag(priority = high) or tag(priority = low)) and tag(region = us)`,
				`        }`,
				`      }`,
				`    }`,
				`  }`,
				`}`,
				``,
			}, "\n")
			require.Equal(t, expected, result)
		})

		t.Run("command with decides_on and nested parentheses", func(t *testing.T) {
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
												Name: "DoThing",
												DecidesOn: &ast.DecidesOnClause{
													Events: []string{"ThingDone"},
													Predicate: &ast.LogicalExpr{
														Left:     &ast.TagPredicate{Field: "priority", Operator: "=", Value: "high"},
														Operator: "and",
														Right: &ast.LogicalExpr{
															Left:     &ast.TagPredicate{Field: "region", Operator: "=", Value: "us"},
															Operator: "or",
															Right:    &ast.TagPredicate{Field: "tier", Operator: "=", Value: "gold"},
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

			result := formatter.Format(model)

			expected := strings.Join([]string{
				`emod 1`,
				`model "Test"`,
				``,
				`context "Ctx" {`,
				`  aggregate "Agg" {`,
				`    slice "S" {`,
				`      command DoThing {`,
				`        decides_on {`,
				`          events [ThingDone]`,
				`          where tag(priority = high) and (tag(region = us) or tag(tier = gold))`,
				`        }`,
				`      }`,
				`    }`,
				`  }`,
				`}`,
				``,
			}, "\n")
			require.Equal(t, expected, result)
		})

		t.Run("command with decides_on and not with compound", func(t *testing.T) {
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
												Name: "DoThing",
												DecidesOn: &ast.DecidesOnClause{
													Events: []string{"ThingDone"},
													Predicate: &ast.NotExpr{
														Expr: &ast.LogicalExpr{
															Left:     &ast.TagPredicate{Field: "priority", Operator: "=", Value: "high"},
															Operator: "or",
															Right:    &ast.TagPredicate{Field: "priority", Operator: "=", Value: "low"},
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

			result := formatter.Format(model)

			expected := strings.Join([]string{
				`emod 1`,
				`model "Test"`,
				``,
				`context "Ctx" {`,
				`  aggregate "Agg" {`,
				`    slice "S" {`,
				`      command DoThing {`,
				`        decides_on {`,
				`          events [ThingDone]`,
				`          where not (tag(priority = high) or tag(priority = low))`,
				`        }`,
				`      }`,
				`    }`,
				`  }`,
				`}`,
				``,
			}, "\n")
			require.Equal(t, expected, result)
		})

		t.Run("command with decides_on and double not", func(t *testing.T) {
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
												Name: "DoThing",
												DecidesOn: &ast.DecidesOnClause{
													Events: []string{"ThingDone"},
													Predicate: &ast.NotExpr{
														Expr: &ast.NotExpr{
															Expr: &ast.TagPredicate{Field: "priority", Operator: "=", Value: "high"},
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

			result := formatter.Format(model)

			expected := strings.Join([]string{
				`emod 1`,
				`model "Test"`,
				``,
				`context "Ctx" {`,
				`  aggregate "Agg" {`,
				`    slice "S" {`,
				`      command DoThing {`,
				`        decides_on {`,
				`          events [ThingDone]`,
				`          where not not tag(priority = high)`,
				`        }`,
				`      }`,
				`    }`,
				`  }`,
				`}`,
				``,
			}, "\n")
			require.Equal(t, expected, result)
		})
	})

	t.Run("dcb regression", func(t *testing.T) {
		t.Run("existing aggregate-based contexts format identically (no regression)", func(t *testing.T) {
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
											{
												Comments:    []*ast.Comment{{Text: "# main flow"}},
												CommandName: "MakeReservation",
												EventName:   "ReservationMade",
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

			// Same expected output as the existing test "formats context with aggregate and slice"
			expected := strings.Join([]string{
				`emod 1`,
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
				`        # main flow`,
				`        command -> event: MakeReservation -> ReservationMade`,
				`      }`,
				`    }`,
				`  }`,
				`}`,
				``,
			}, "\n")
			require.Equal(t, expected, result)
		})

		t.Run("round-trip: DCB model with mode, direct slices, tags, decides_on", func(t *testing.T) {
			input := strings.Join([]string{
				`model "Test"`,
				``,
				`context "Ctx" mode dcb {`,
				`  slice "My Slice" {`,
				`    command DoThing {`,
				`      decides_on {`,
				`        events [ThingDone]`,
				`        where tag(priority = high)`,
				`      }`,
				`      fields {`,
				`        id string required`,
				`      }`,
				`    }`,
				``,
				`    event ThingDone {`,
				`      tags {`,
				`        priority: statusCode`,
				`      }`,
				`      fields {`,
				`        statusCode string`,
				`      }`,
				`    }`,
				``,
				`    flow {`,
				`      command -> event: DoThing -> ThingDone`,
				`    }`,
				`  }`,
				`}`,
				``,
			}, "\n")

			original := parseModel(t, input, "test.emod")

			reparsed := parseModel(t, formatter.Format(original), "test.emod")

			test.RequireEqual(t, original, reparsed, ignoreFormatterNormalizations)
		})

		t.Run("idempotency: DCB model format is idempotent", func(t *testing.T) {
			input := strings.Join([]string{
				`model "Test"`,
				``,
				`context "Ctx" mode dcb {`,
				`  slice "My Slice" {`,
				`    command DoThing {`,
				`      decides_on {`,
				`        events [ThingDone]`,
				`        where tag(priority = high)`,
				`      }`,
				`    }`,
				``,
				`    event ThingDone {`,
				`      tags {`,
				`        priority: statusCode`,
				`      }`,
				`    }`,
				`  }`,
				`}`,
				``,
			}, "\n")

			original := parseModel(t, input, "test.emod")

			firstFormat := formatter.Format(original)
			secondFormat := formatter.Format(parseModel(t, firstFormat, "formatted.emod"))

			require.Equal(t, firstFormat, secondFormat,
				"formatting the already-formatted DCB output should produce identical bytes")
		})
	})
}

func parseModel(t *testing.T, source, filename string) *ast.Model {
	t.Helper()

	tokens, scanErrs := lexer.Scan(source, filename)
	require.Empty(t, scanErrs)

	model, parseErrs := parser.New(tokens, filename).Parse()
	require.Empty(t, parseErrs)

	return model
}

func parseFixture(t *testing.T, filename string) *ast.Model {
	t.Helper()

	source, err := os.ReadFile("../parser/testdata/" + filename)
	require.NoError(t, err)

	return parseModel(t, string(source), filename)
}

func lineAfter(t *testing.T, formatted, blockHeader string) string {
	t.Helper()

	lines := strings.Split(formatted, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == blockHeader && i+1 < len(lines) {
			return strings.TrimSpace(lines[i+1])
		}
	}

	require.FailNowf(t, "block header not found in formatted output", "%q in:\n%s", blockHeader, formatted)
	return ""
}

var ignoreFormatterNormalizations = cmp.Options{
	cmpopts.IgnoreTypes(ast.Position{}),
	cmpopts.IgnoreFields(ast.Model{}, "Comments", "VersionDeclared"),
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

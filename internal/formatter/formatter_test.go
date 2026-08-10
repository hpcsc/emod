//go:build unit

package formatter_test

import (
	"os"
	"slices"
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

		t.Run("a flow block writes every event entry, then every rejection entry", func(t *testing.T) {
			model := &ast.Model{
				Name: "Library Lending",
				Contexts: []*ast.Context{
					{
						Name: "Lending",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Loan",
								Slices: []*ast.Slice{
									{
										Name: "Borrow a Copy",
										Flows: []*ast.Flow{
											{CommandName: "BorrowCopy", EventName: "CopyBorrowed"},
											{CommandName: "BorrowCopy", EventName: "LoanOpened"},
										},
										Rejections: []*ast.Rejection{
											{CommandName: "BorrowCopy", InvariantName: "OneCopyPerLoan"},
											{CommandName: "BorrowCopy", InvariantName: "FiveCopiesPerMember"},
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
				`model "Library Lending"`,
				``,
				`context "Lending" {`,
				`  aggregate "Loan" {`,
				`    slice "Borrow a Copy" {`,
				`      flow {`,
				`        command -> event: BorrowCopy -> CopyBorrowed`,
				`        command -> event: BorrowCopy -> LoanOpened`,
				`        command -> rejected: BorrowCopy -> OneCopyPerLoan`,
				`        command -> rejected: BorrowCopy -> FiveCopiesPerMember`,
				`      }`,
				`    }`,
				`  }`,
				`}`,
				``,
			}, "\n")

			require.Equal(t, expected, result)
		})

		t.Run("a slice stating only rejections still emits a flow block", func(t *testing.T) {
			model := &ast.Model{
				Name: "Library Lending",
				Contexts: []*ast.Context{
					{
						Name: "Lending",
						Aggregates: []*ast.Aggregate{
							{
								Name: "Loan",
								Slices: []*ast.Slice{
									{
										Name: "Borrow a Copy",
										Rejections: []*ast.Rejection{
											{CommandName: "BorrowCopy", InvariantName: "OneCopyPerLoan"},
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
				`model "Library Lending"`,
				``,
				`context "Lending" {`,
				`  aggregate "Loan" {`,
				`    slice "Borrow a Copy" {`,
				`      flow {`,
				`        command -> rejected: BorrowCopy -> OneCopyPerLoan`,
				`      }`,
				`    }`,
				`  }`,
				`}`,
				``,
			}, "\n")

			require.Equal(t, expected, result)
		})

		t.Run("a slice stating neither entry kind emits no flow block", func(t *testing.T) {
			model := &ast.Model{
				Name: "Library Lending",
				Contexts: []*ast.Context{
					{
						Name: "Lending",
						Aggregates: []*ast.Aggregate{
							{
								Name:   "Loan",
								Slices: []*ast.Slice{{Name: "Borrow a Copy"}},
							},
						},
					},
				},
			}

			result := formatter.Format(model)

			require.NotContains(t, result, "flow {")
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
				`      trigger "Form" {`,
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

		t.Run("formats kindless trigger block", func(t *testing.T) {
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
										Name: "Kindless",
										Trigger: &ast.Trigger{
											Name:  "Kindless Form",
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
				`    slice "Kindless" {`,
				`      trigger "Kindless Form" {`,
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

		t.Run("formats two triggers with one space between keyword and name", func(t *testing.T) {
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
										Name: "Kindless",
										Trigger: &ast.Trigger{
											Name:  "Kindless Form",
											Actor: "Guest",
											Reads: "KindlessView",
										},
									},
									{
										Name: "Kinded",
										Trigger: &ast.Trigger{
											Name:  "Kinded Form",
											Actor: "Guest",
											Reads: "KindedView",
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
				`    slice "Kindless" {`,
				`      trigger "Kindless Form" {`,
				`        actor Guest`,
				`        reads KindlessView`,
				`      }`,
				`    }`,
				``,
				`    slice "Kinded" {`,
				`      trigger "Kinded Form" {`,
				`        actor Guest`,
				`        reads KindedView`,
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
												OnEvent:       "OrderPlaced",
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
				`        on OrderPlaced`,
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

		t.Run("formats an automation that reads a view between its activation event and its command", func(t *testing.T) {
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
												OnEvent:       "OrderPlaced",
												Reads:         "OpenOrdersView",
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
				`        on OrderPlaced`,
				`        reads OpenOrdersView`,
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

		t.Run("an automation's entries are written in canonical order wherever the source put them", func(t *testing.T) {
			expected := strings.Join([]string{
				`emod 1`,
				`model "Order Fulfilment"`,
				``,
				`context "Fulfilment" {`,
				`  aggregate "Shipment" {`,
				`    slice "Notify Customer" {`,
				`      automation OrderNotifier {`,
				`        description "Tells the customer their parcel has left the depot"`,
				`        on OrderPlaced`,
				`        reads OpenOrdersView`,
				`        command SendNotification`,
				`        target context Notifications`,
				`      }`,
				`    }`,
				`  }`,
				`}`,
				``,
			}, "\n")

			for _, testCase := range []struct {
				placement string
				body      []string
			}{
				{"the view it reads is written as the block's first entry", []string{
					`        reads OpenOrdersView`,
					`        description "Tells the customer their parcel has left the depot"`,
					`        on OrderPlaced`,
					`        command SendNotification`,
					`        target context Notifications`,
				}},
				{"the view it reads is written after target context", []string{
					`        description "Tells the customer their parcel has left the depot"`,
					`        on OrderPlaced`,
					`        command SendNotification`,
					`        target context Notifications`,
					`        reads OpenOrdersView`,
				}},
				{"its description is written last", []string{
					`        on OrderPlaced`,
					`        reads OpenOrdersView`,
					`        command SendNotification`,
					`        target context Notifications`,
					`        description "Tells the customer their parcel has left the depot"`,
				}},
			} {
				t.Run(testCase.placement, func(t *testing.T) {
					source := orderNotifierSource(testCase.body...)

					result := formatter.Format(parseModel(t, source, "automation-entries.emod"))

					require.Equal(t, expected, result)
				})
			}
		})

		t.Run("a schedule is written under the description of the automation declaring it and invented for no sibling", func(t *testing.T) {
			source := expirySweepSliceSource(
				`      automation NightlyExpirySweep {`,
				`        description "Releases the holds nobody paid for overnight"`,
				`        reads ExpiringHoldsView`,
				`        command ReleaseExpiredHolds`,
				`        target context Inventory`,
				`            every "0 2 * * *"`,
				`      }`,
				``,
				`      automation LedgerSync {`,
				`        command SyncLedger`,
				`        every "15m"`,
				`      }`,
				``,
				`      automation OrderNotifier {`,
				`        on OrderPlaced`,
				`        command SendNotification`,
				`      }`,
			)

			result := formatter.Format(parseModel(t, source, "scheduled-automation.emod"))

			expected := strings.Join([]string{
				`emod 1`,
				`model "Order Fulfilment"`,
				``,
				`context "Fulfilment" {`,
				`  aggregate "Shipment" {`,
				`    slice "Sweep Expired Holds" {`,
				`      automation NightlyExpirySweep {`,
				`        description "Releases the holds nobody paid for overnight"`,
				`        every "0 2 * * *"`,
				`        reads ExpiringHoldsView`,
				`        command ReleaseExpiredHolds`,
				`        target context Inventory`,
				`      }`,
				``,
				`      automation LedgerSync {`,
				`        every "15m"`,
				`        command SyncLedger`,
				`      }`,
				``,
				`      automation OrderNotifier {`,
				`        on OrderPlaced`,
				`        command SendNotification`,
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
			reparsed := requireStableFormat(t, original)

			test.RequireEqual(t, original, reparsed, ignoreFormatterNormalizations)
			test.RequireEqual(t, []*ast.Field{
				{Name: "roomType", Type: "string"},
				{Name: "guestId", Type: "string", Modifier: "required"},
			}, reparsed.Contexts[0].Aggregates[0].Slices[0].Commands[0].Fields, ignoreFormatterNormalizations)
		})

		t.Run("round-trip: rejection edges in both slice homes survive formatting", func(t *testing.T) {
			input := strings.Join([]string{
				`model "Library Lending"`,
				``,
				`context "Lending" {`,
				`  aggregate "Loan" {`,
				`    invariant OneCopyPerLoan "A loan covers exactly one copy of one title"`,
				``,
				`    slice "Borrow a Copy" {`,
				`      flow {`,
				`        command -> event: BorrowCopy -> CopyBorrowed`,
				`        command -> rejected: BorrowCopy -> OneCopyPerLoan`,
				`        command -> event: BorrowCopy -> LoanOpened`,
				`      }`,
				`    }`,
				`  }`,
				`}`,
				``,
				`context "Reading Room" mode dcb {`,
				`  invariant OneReaderPerDesk "A desk seats at most one reader at any moment"`,
				``,
				`  slice "Claim a Desk" {`,
				`    flow {`,
				`      command -> rejected: ClaimDesk -> OneReaderPerDesk`,
				`    }`,
				`  }`,
				`}`,
				``,
			}, "\n")

			original := parseModel(t, input, "test.emod")
			reparsed := requireStableFormat(t, original)

			test.RequireEqual(t, original, reparsed, ignoreFormatterNormalizations)
			test.RequireEqual(t, []*ast.Rejection{
				{CommandName: "BorrowCopy", InvariantName: "OneCopyPerLoan"},
			}, reparsed.Contexts[0].Aggregates[0].Slices[0].Rejections, ignoreFormatterNormalizations)
			test.RequireEqual(t, []*ast.Rejection{
				{CommandName: "ClaimDesk", InvariantName: "OneReaderPerDesk"},
			}, reparsed.Contexts[1].Slices[0].Rejections, ignoreFormatterNormalizations)
		})

		t.Run("round-trip: fields named after keywords survive formatting", func(t *testing.T) {
			original := test.KeywordFieldSearchCatalogModel(t)

			reparsed := requireStableFormat(t, original)

			test.RequireEqual(t, original, reparsed, ignoreFormatterNormalizations)
		})

		t.Run("round-trip: a description on every construct survives formatting", func(t *testing.T) {
			original := test.DescribedHotelReservationModel(t)

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

					reparsed := requireStableFormat(t, parseModel(t, source, "described.emod"))

					require.Equal(t, testCase.description, reparsed.Description)
				})
			}
		})

		t.Run("round-trip: invariants declared on an aggregate and on a dcb context survive formatting", func(t *testing.T) {
			original := parseModel(t, test.InvariantLibraryLending, "library-lending.emod")

			reparsed := requireStableFormat(t, original)

			test.RequireEqual(t, original, reparsed, ignoreFormatterNormalizations)
			test.RequireEqual(t, []*ast.Invariant{
				{Name: "OneCopyPerLoan", Statement: "A loan covers exactly one copy of one title"},
				{Name: "FiveCopiesPerMember", Statement: "A member holds at most five copies at one time"},
			}, reparsed.Contexts[0].Aggregates[0].Invariants, ignoreFormatterNormalizations)
			test.RequireEqual(t, []*ast.Invariant{
				{Name: "OneReaderPerDesk", Statement: "A desk seats at most one reader at any moment"},
				{Name: "OneDeskPerReader", Statement: "A reader holds at most one desk for the length of a session"},
				{Name: "DeskFreeAtClosing", Statement: "No desk stays claimed past the closing hour"},
			}, reparsed.Contexts[1].Invariants, ignoreFormatterNormalizations)
		})

		t.Run("round-trip: an invariant statement survives text that quoting could mangle", func(t *testing.T) {
			for _, testCase := range []struct {
				hazard    string
				statement string
			}{
				{"backslash", `The shelf map lives in C:\library\stacks`},
				{"tab", "Two columns:\tcopy then due date"},
				{"newline", "Two lines:\ncopy then due date"},
				{"percent", "At most 10% of the stock is on loan at once"},
				{"non-ascii", "A member holds ≤5 copies"},
			} {
				t.Run(testCase.hazard, func(t *testing.T) {
					source := strings.Join([]string{
						`model "Library Lending"`,
						``,
						`context "Lending" {`,
						`  aggregate "Loan" {`,
						`    invariant CopyLimit "` + testCase.statement + `"`,
						`  }`,
						`}`,
						``,
					}, "\n")

					reparsed := requireStableFormat(t, parseModel(t, source, "library-lending.emod"))

					require.Equal(t, testCase.statement,
						reparsed.Contexts[0].Aggregates[0].Invariants[0].Statement)
				})
			}
		})

		t.Run("round-trip: an invariant named but not yet written down is still written back", func(t *testing.T) {
			source := strings.Join([]string{
				`model "Library Lending"`,
				``,
				`context "Lending" {`,
				`  aggregate "Loan" {`,
				`    invariant OneCopyPerLoan "A loan covers exactly one copy of one title"`,
				`    invariant OverdueFine ""`,
				`  }`,
				`}`,
				``,
			}, "\n")

			reparsed := requireStableFormat(t, parseModel(t, source, "library-lending.emod"))

			test.RequireEqual(t, []*ast.Invariant{
				{Name: "OneCopyPerLoan", Statement: "A loan covers exactly one copy of one title"},
				{Name: "OverdueFine", Statement: ""},
			}, reparsed.Contexts[0].Aggregates[0].Invariants, ignoreFormatterNormalizations)
		})

		t.Run("round-trip: specs declared in an aggregate slice and on a dcb context slice survive formatting", func(t *testing.T) {
			original := parseModel(t, test.SpecLibraryLending, "specs.emod")

			reparsed := requireStableFormat(t, original)

			test.RequireEqual(t, original, reparsed, ignoreFormatterNormalizations)
			test.RequireEqual(t, []*ast.Spec{
				{
					Name: "borrows a copy no one holds",
					When: &ast.SpecElement{Name: "BorrowCopy"},
					Then: &ast.ThenEvents{Events: []*ast.SpecElement{{Name: "CopyBorrowed"}}},
				},
				{
					Name:  "borrows a copy the member before returned",
					Given: []*ast.SpecElement{{Name: "CopyBorrowed"}, {Name: "CopyReturned"}},
					When:  &ast.SpecElement{Name: "BorrowCopy"},
					Then:  &ast.ThenEvents{Events: []*ast.SpecElement{{Name: "CopyBorrowed"}}},
				},
				{
					Name:  "refuses a copy already on loan",
					Given: []*ast.SpecElement{{Name: "CopyBorrowed"}},
					When:  &ast.SpecElement{Name: "BorrowCopy"},
					Then:  &ast.ThenRejected{InvariantName: "OneCopyPerLoan"},
				},
			}, reparsed.Contexts[0].Aggregates[0].Slices[0].Specs, ignoreFormatterNormalizations)
		})

		t.Run("round-trip: rejection edges declared in an aggregate slice and on a dcb context slice survive formatting", func(t *testing.T) {
			original := test.RejectionLibraryLendingModel(t)

			reparsed := requireStableFormat(t, original)

			test.RequireEqual(t, original, reparsed, ignoreFormatterNormalizations)
			require.NotEmpty(t, test.RejectionLibraryLendingRejections)
			require.Equal(t, test.RejectionLibraryLendingRejections, test.DeclaredRejections(reparsed))
		})

		t.Run("a flow block's leading comment is written at the top of the block, not behind its flow entries", func(t *testing.T) {
			input := strings.Join([]string{
				`model "Library Lending"`,
				``,
				`context "Lending" {`,
				`  aggregate "Loan" {`,
				`    invariant OneCopyPerLoan "A loan covers exactly one copy of one title"`,
				``,
				`    slice "Borrow a Copy" {`,
				`      # the copy may already be out`,
				`      flow {`,
				`        command -> rejected: BorrowCopy -> OneCopyPerLoan`,
				`        command -> event: BorrowCopy -> CopyBorrowed`,
				`      }`,
				`    }`,
				`  }`,
				`}`,
				``,
			}, "\n")

			// The block's comment is parsed onto its first entry and the
			// formatter writes every flow ahead of every rejection, so leaving
			// it on the rejection the author wrote first would sink the comment
			// below the entry it introduces.
			require.Contains(t, formatter.Format(parseModel(t, input, "test.emod")),
				strings.Join([]string{
					`      flow {`,
					`        # the copy may already be out`,
					`        command -> event: BorrowCopy -> CopyBorrowed`,
					`        command -> rejected: BorrowCopy -> OneCopyPerLoan`,
					`      }`,
				}, "\n"))
		})

		t.Run("round-trip: a comment above a rejection entry is not deleted", func(t *testing.T) {
			model := &ast.Model{
				Name: "Library Lending",
				Contexts: []*ast.Context{{
					Name: "Lending",
					Aggregates: []*ast.Aggregate{{
						Name: "Loan",
						Slices: []*ast.Slice{{
							Name: "Borrow a Copy",
							Rejections: []*ast.Rejection{{
								Comments:      []*ast.Comment{{Text: "# the copy may already be out"}},
								CommandName:   "BorrowCopy",
								InvariantName: "OneCopyPerLoan",
							}},
						}},
					}},
				}},
			}

			require.Contains(t, formatter.Format(model), "# the copy may already be out",
				"emod fmt must not silently delete a comment written above a rejection entry")
		})

		t.Run("round-trip: the rejection twin clears every edge in both slice homes and moves nothing else", func(t *testing.T) {
			stating := test.RejectionLibraryLendingModel(t)

			twin := test.WithoutRejections(stating)

			require.Empty(t, test.DeclaredRejections(twin))
			require.Equal(t, test.RejectionLibraryLendingRejections, test.DeclaredRejections(stating),
				"the twin must not write through to the model it was handed")
			test.RequireEqual(t, stating, twin, cmpopts.IgnoreFields(ast.Slice{}, "Rejections"))
			require.Empty(t, test.DeclaredRejections(requireStableFormat(t, twin)),
				"the twin states no rejection edge, so formatting it may not invent one")
		})

		t.Run("round-trip: the view and command outcomes in the slice-pattern fixture survive formatting", func(t *testing.T) {
			original := test.SlicePatternLibraryLendingModel(t)

			reparsed := requireStableFormat(t, original)

			test.RequireEqual(t, original, reparsed, ignoreFormatterNormalizations)
			require.Equal(t, test.SlicePatternLibraryLendingSpecNames, test.DeclaredSpecNames(reparsed))
			require.Equal(t, test.SlicePatternLibraryLendingOutcomeKinds, test.DeclaredSpecOutcomeKinds(reparsed))
		})

		t.Run("round-trip: an empty given history is written by leaving the given line out, however it was spelled", func(t *testing.T) {
			expected := strings.Join([]string{
				`emod 1`,
				`model "Reading Room"`,
				``,
				`context "Reading Room" mode dcb {`,
				`  slice "Claim Desk" {`,
				`    spec "seats a reader at a free desk" {`,
				`      when ClaimDesk`,
				`      then [DeskClaimed]`,
				`    }`,
				`  }`,
				`}`,
				``,
			}, "\n")

			for _, testCase := range []struct {
				spelling   string
				givenLines []string
			}{
				{"an empty given list", []string{`      given []`}},
				{"no given entry at all", nil},
			} {
				t.Run(testCase.spelling, func(t *testing.T) {
					source := strings.Join(slices.Concat(
						[]string{
							`model "Reading Room"`,
							``,
							`context "Reading Room" mode dcb {`,
							`  slice "Claim Desk" {`,
							`    spec "seats a reader at a free desk" {`,
						},
						testCase.givenLines,
						[]string{
							`      when ClaimDesk`,
							`      then [DeskClaimed]`,
							`    }`,
							`  }`,
							`}`,
							``,
						},
					), "\n")
					model := parseModel(t, source, "specs.emod")

					require.Equal(t, expected, formatter.Format(model))
					requireStableFormat(t, model)
				})
			}
		})

		t.Run("round-trip: a spec name survives text that quoting could mangle", func(t *testing.T) {
			for _, testCase := range []struct {
				hazard string
				name   string
			}{
				{"backslash", `borrows the copy shelved under C:\library\stacks`},
				{"tab", "two columns:\tcopy then due date"},
				{"newline", "two lines:\ncopy then due date"},
				{"percent", "borrows the last 10% of the stock"},
				{"non-ascii", "borrows while the member holds ≤5 copies"},
			} {
				t.Run(testCase.hazard, func(t *testing.T) {
					source := strings.Join([]string{
						`model "Library Lending"`,
						``,
						`context "Lending" {`,
						`  aggregate "Loan" {`,
						`    slice "Borrow Copy" {`,
						`      spec "` + testCase.name + `" {`,
						`        when BorrowCopy`,
						`        then [CopyBorrowed]`,
						`      }`,
						`    }`,
						`  }`,
						`}`,
						``,
					}, "\n")

					reparsed := requireStableFormat(t, parseModel(t, source, "specs.emod"))

					require.Equal(t, testCase.name,
						reparsed.Contexts[0].Aggregates[0].Slices[0].Specs[0].Name)
				})
			}
		})

		t.Run("round-trip: a spec named but with no command or outcome yet is still written back", func(t *testing.T) {
			source := strings.Join([]string{
				`model "Library Lending"`,
				``,
				`context "Lending" {`,
				`  aggregate "Loan" {`,
				`    slice "Borrow Copy" {`,
				`      spec "borrows a copy no one holds" {`,
				`      }`,
				`      spec "refuses a copy already on loan" {`,
				`        when BorrowCopy`,
				`      }`,
				`    }`,
				`  }`,
				`}`,
				``,
			}, "\n")

			reparsed := requireStableFormat(t, parseModel(t, source, "specs.emod"))

			test.RequireEqual(t, []*ast.Spec{
				{Name: "borrows a copy no one holds"},
				{Name: "refuses a copy already on loan", When: &ast.SpecElement{Name: "BorrowCopy"}},
			}, reparsed.Contexts[0].Aggregates[0].Slices[0].Specs, ignoreFormatterNormalizations)
		})

		t.Run("round-trip: the view an automation reads and the event it activates on survive formatting in both slice homes", func(t *testing.T) {
			reading := test.AutomationReadsLibraryLendingModel(t)
			unread := test.WithoutAutomationReads(reading)

			reparsed := requireStableFormat(t, reading)
			reparsedTwin := requireStableFormat(t, unread)

			test.RequireEqual(t, reading, reparsed, ignoreFormatterNormalizations)
			require.Equal(t, test.AutomationReadsLibraryLendingViewNames, test.DeclaredAutomationReads(reparsed))
			require.Equal(t, test.AutomationReadsLibraryLendingActivationEvents, test.DeclaredActivationEvents(reparsed))
			require.Empty(t, test.DeclaredAutomationReads(reparsedTwin),
				"the twin declares no automation reads, so formatting it may not invent one")

			formatted, formattedTwin := formatter.Format(reading), formatter.Format(unread)
			require.Contains(t, formattedTwin, "reads AvailableCopiesView",
				"what a trigger reads is not an automation's, so the twin keeps it")
			require.NotEqual(t, formatted, formattedTwin,
				"the twin has to read no view, or the comparison below says nothing")
			require.Equal(t, withoutReadsLines(formatted), withoutReadsLines(formattedTwin),
				"naming the view an automation reads moves no other line of the model")
		})

		t.Run("round-trip: a whole shared model survives formatting, the views its triggers and automations read and the schedule or event each automation activates on included", func(t *testing.T) {
			for _, testCase := range []struct {
				shape            string
				parse            func(*testing.T) *ast.Model
				activationEvents []string
				schedules        []string
				triggerReads     []string
				automationReads  []string
			}{
				{
					shape:            "automations in both slice homes, reading views across contexts",
					parse:            test.AutomationReadsLibraryLendingModel,
					activationEvents: test.AutomationReadsLibraryLendingActivationEvents,
					triggerReads:     []string{"AvailableCopiesView"},
					automationReads:  test.AutomationReadsLibraryLendingViewNames,
				},
				{
					shape:            "an automation among all four slice patterns",
					parse:            test.HotelReservationModel,
					activationEvents: []string{"ReservationMade"},
					triggerReads:     []string{"AvailableRoomsView"},
					automationReads:  []string{"ReservationsView"},
				},
				{
					shape:            "automations run on a schedule in both slice homes, beside automations activated by an event",
					parse:            test.AutomationScheduleLibraryLendingModel,
					activationEvents: test.AutomationScheduleLibraryLendingActivationEvents,
					schedules:        test.AutomationScheduleLibraryLendingSchedules,
					triggerReads:     []string{"AvailableCopiesView"},
					automationReads:  []string{"MemberLoansView", "DeskOccupancyView", "MemberLoansView"},
				},
				{
					shape:            "triggers in both slice homes, reading views the model declares in another slice and in another context",
					parse:            test.TriggerReadsLibraryLendingModel,
					activationEvents: []string{"CopyBorrowed", "CopyBorrowed", "DeskClaimed", "DeskReleased"},
					triggerReads:     test.TriggerReadsLibraryLendingTriggerViewNames,
					automationReads:  test.TriggerReadsLibraryLendingAutomationViewNames,
				},
			} {
				t.Run(testCase.shape, func(t *testing.T) {
					original := testCase.parse(t)

					reparsed := requireStableFormat(t, original)

					test.RequireEqual(t, original, reparsed, ignoreFormatterNormalizations)
					require.Equal(t, testCase.activationEvents, test.DeclaredActivationEvents(reparsed))
					require.Equal(t, testCase.schedules, test.DeclaredSchedules(reparsed))
					require.Equal(t, testCase.triggerReads, test.DeclaredTriggerReads(reparsed))
					require.Equal(t, testCase.automationReads, test.DeclaredAutomationReads(reparsed))
				})
			}
		})

		t.Run("round-trip: a reads twin clears the construct it names in both slice homes and leaves the model it was handed whole", func(t *testing.T) {
			for _, testCase := range []struct {
				construct          string
				twin               func(*ast.Model) *ast.Model
				clearedReads       func(*ast.Model) []string
				untouchedReads     func(*ast.Model) []string
				untouchedViewNames []string
			}{
				{
					construct:          "triggers",
					twin:               test.WithoutTriggerReads,
					clearedReads:       test.DeclaredTriggerReads,
					untouchedReads:     test.DeclaredAutomationReads,
					untouchedViewNames: test.TriggerReadsLibraryLendingAutomationViewNames,
				},
				{
					construct:          "automations",
					twin:               test.WithoutAutomationReads,
					clearedReads:       test.DeclaredAutomationReads,
					untouchedReads:     test.DeclaredTriggerReads,
					untouchedViewNames: test.TriggerReadsLibraryLendingTriggerViewNames,
				},
			} {
				t.Run(testCase.construct, func(t *testing.T) {
					require.NotEmpty(t, testCase.untouchedViewNames,
						"a twin leaving the other construct alone says nothing unless that construct reads something")
					reading := test.TriggerReadsLibraryLendingModel(t)

					twin := testCase.twin(reading)

					require.Empty(t, testCase.clearedReads(twin))
					require.Equal(t, testCase.untouchedViewNames, testCase.untouchedReads(twin))
					require.Equal(t, test.TriggerReadsLibraryLendingTriggerViewNames, test.DeclaredTriggerReads(reading))
					require.Equal(t, test.TriggerReadsLibraryLendingAutomationViewNames, test.DeclaredAutomationReads(reading))
					require.Empty(t, testCase.clearedReads(requireStableFormat(t, twin)),
						"the twin reads no view, so formatting it may not invent one")
				})
			}
		})

		t.Run("round-trip: the schedule one automation runs on and the event its sibling activates on both survive formatting", func(t *testing.T) {
			source := expirySweepSliceSource(
				`      automation NightlyExpirySweep {`,
				`        description "Releases the holds nobody paid for overnight"`,
				`        every "0 2 * * *"`,
				`        reads ExpiringHoldsView`,
				`        command ReleaseExpiredHolds`,
				`        target context Inventory`,
				`      }`,
				``,
				`      automation OrderNotifier {`,
				`        on OrderPlaced`,
				`        command SendNotification`,
				`      }`,
			)
			original := parseModel(t, source, "scheduled-automation.emod")

			reparsed := requireStableFormat(t, original)

			test.RequireEqual(t, original, reparsed, ignoreFormatterNormalizations)
			test.RequireEqual(t, []*ast.Automation{
				{
					Name:          "NightlyExpirySweep",
					Description:   "Releases the holds nobody paid for overnight",
					Schedule:      "0 2 * * *",
					Reads:         "ExpiringHoldsView",
					Command:       "ReleaseExpiredHolds",
					TargetContext: "Inventory",
				},
				{
					Name:    "OrderNotifier",
					OnEvent: "OrderPlaced",
					Command: "SendNotification",
				},
			}, reparsed.Contexts[0].Aggregates[0].Slices[0].Automations, ignoreFormatterNormalizations)
		})

		t.Run("round-trip: a schedule survives text that quoting could mangle", func(t *testing.T) {
			for _, testCase := range []struct {
				hazard     string
				expression string
			}{
				{"backslash", `0 2 * * * per C:\ops\nightly`},
				{"tab", "0 2 * * *\tnightly sweep"},
				{"newline", "0 2 * * *\nnightly sweep"},
				{"percent", "0 2 * * * while under 10% load"},
				{"non-ascii", "0 2 * * * ≤ once a night"},
			} {
				t.Run(testCase.hazard, func(t *testing.T) {
					source := expirySweepSliceSource(
						`      automation NightlyExpirySweep {`,
						`        every "`+testCase.expression+`"`,
						`        command ReleaseExpiredHolds`,
						`      }`,
					)

					reparsed := requireStableFormat(t, parseModel(t, source, "scheduled-automation.emod"))

					require.Equal(t, testCase.expression,
						reparsed.Contexts[0].Aggregates[0].Slices[0].Automations[0].Schedule)
				})
			}
		})

		t.Run("round-trip: a kindless trigger survives formatting", func(t *testing.T) {
			source := strings.Join([]string{
				`model "Test"`,
				``,
				`context "Ctx" {`,
				`  aggregate "Agg" {`,
				`    slice "Slice" {`,
				`      trigger "Reservation Form" {`,
				`        description "The booking form on the public site"`,
				`        actor Guest`,
				`        reads AvailableRoomsView`,
				`      }`,
				`    }`,
				`  }`,
				`}`,
				``,
			}, "\n")

			original := parseModel(t, source, "test.emod")

			reparsed := parseModel(t, formatter.Format(original), "test.emod")

			test.RequireEqual(t, original, reparsed, ignoreFormatterNormalizations)
			require.True(t, reparsed.VersionDeclared, "formatted output should pin the version")
		})

		t.Run("round-trip: a trigger name survives text that quoting could mangle", func(t *testing.T) {
			for _, testCase := range []struct {
				hazard string
				name   string
			}{
				{"backslash", `C:\hotel\rates`},
				{"tab", "Two columns:\troom then rate"},
				{"percent", "Form with 10% deposit"},
				{"non-ascii", "Form with ≤ deposit"},
			} {
				t.Run(testCase.hazard, func(t *testing.T) {
					source := strings.Join([]string{
						`model "Test"`,
						``,
						`context "Ctx" {`,
						`  aggregate "Agg" {`,
						`    slice "Slice" {`,
						`      trigger "` + testCase.name + `" {`,
						`        actor Guest`,
						`      }`,
						`    }`,
						`  }`,
						`}`,
						``,
					}, "\n")

					original := parseModel(t, source, "test.emod")
					reparsed := requireStableFormat(t, original)

					require.Equal(t, testCase.name, reparsed.Contexts[0].Aggregates[0].Slices[0].Trigger.Name)
				})
			}
		})

		t.Run("idempotency: format(format(described input)) equals format(described input)", func(t *testing.T) {
			requireStableFormat(t, test.DescribedHotelReservationModel(t))
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
												OnEvent:       "ThingDone",
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

			triggerIdx := strings.Index(result, `trigger "Form"`)
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
			require.Equal(t, `description "The form the user fills in"`, lineAfter(t, result, `trigger "Form" {`))
			require.Equal(t, `description "Ask for the thing to be done"`, lineAfter(t, result, `command DoThing {`))
			require.Equal(t, `description "The thing was done"`, lineAfter(t, result, `event ThingDone {`))
			require.Equal(t, `description "Every thing and whether it is done"`, lineAfter(t, result, `view ThingView {`))
			require.Equal(t, `description "Notifies whoever asked once the thing is done"`, lineAfter(t, result, `automation Reactor {`))
			require.Equal(t, `description "Restates a partner report as a thing"`, lineAfter(t, result, `translation Importer {`))
			require.Equal(t, `description "A partner reported a thing"`, lineAfter(t, result, `event ThingImported {`))
		})

		t.Run("canonical order gathers invariants after the description and ahead of the first aggregate or slice", func(t *testing.T) {
			input := strings.Join([]string{
				`model "Library Lending"`,
				``,
				`context "Lending" {`,
				`  description "How the library lends its copies"`,
				`  aggregate "Loan" {`,
				`    description "One member holding one copy over one loan period"`,
				`    slice "Borrow Copy" {`,
				`    }`,
				`    invariant OneCopyPerLoan "A loan covers exactly one copy of one title"`,
				`    invariant FiveCopiesPerMember "A member holds at most five copies at one time"`,
				`  }`,
				`  invariant EveryLoanHasABorrower "Every loan names the member who holds it"`,
				`  invariant CopiesReturnToTheShelf "A returned copy is back on the shelf the same day"`,
				`}`,
				``,
			}, "\n")

			result := formatter.Format(parseModel(t, input, "library-lending.emod"))

			require.Equal(t, `invariant EveryLoanHasABorrower "Every loan names the member who holds it"`,
				lineAfter(t, result, `description "How the library lends its copies"`))
			require.Equal(t, `invariant CopiesReturnToTheShelf "A returned copy is back on the shelf the same day"`,
				lineAfter(t, result, `invariant EveryLoanHasABorrower "Every loan names the member who holds it"`))
			require.Equal(t, `aggregate "Loan" {`,
				lineAfter(t, result, `invariant CopiesReturnToTheShelf "A returned copy is back on the shelf the same day"`))

			require.Equal(t, `invariant OneCopyPerLoan "A loan covers exactly one copy of one title"`,
				lineAfter(t, result, `description "One member holding one copy over one loan period"`))
			require.Equal(t, `invariant FiveCopiesPerMember "A member holds at most five copies at one time"`,
				lineAfter(t, result, `invariant OneCopyPerLoan "A loan covers exactly one copy of one title"`))
			require.Equal(t, `slice "Borrow Copy" {`,
				lineAfter(t, result, `invariant FiveCopiesPerMember "A member holds at most five copies at one time"`))
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

		t.Run("a field named after a keyword keeps its own line with the columns padded", func(t *testing.T) {
			formatted := formatter.Format(test.KeywordFieldSearchCatalogModel(t))

			require.True(t, strings.HasPrefix(formatted, "emod 1\n"), "formatted output should open with the version header")

			requireFieldsBlockAfter(t, formatted, "command DefineSavedSearch {",
				`        fields {`,
				`          model       string required`,
				`          source      string required`,
				`          where       string required`,
				`          and         string`,
				`          not         string`,
				`          fields      string required`,
				`          description string optional`,
				`        }`,
			)

			requireFieldsBlockAfter(t, formatted, "event SavedSearchDefined {",
				`        fields {`,
				`          searchId    string required`,
				`          model       string required`,
				`          source      string required`,
				`          where       string required`,
				`          events      string required`,
				`          tag         string required`,
				`          emod        string`,
				`          description string required`,
				`          definedAt   date   required`,
				`        }`,
			)

			requireFieldsBlockAfter(t, formatted, "view SavedSearchesView {",
				`        fields {`,
				`          searchId    string required`,
				`          description string required`,
				`          tag         string required`,
				`          model       string`,
				`          where       string required`,
				`          matches     int    required`,
				`        }`,
			)
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
			requireStableFormat(t, parseFixture(t, "all_patterns.emod"))
		})

		t.Run("comment above an invariant stays directly above it at the invariant's indentation", func(t *testing.T) {
			input := strings.Join([]string{
				`model "Library Lending"`,
				``,
				`context "Lending" {`,
				`  aggregate "Loan" {`,
				`    # A shelf at a time would let a member empty a series`,
				`    invariant OneCopyPerLoan "A loan covers exactly one copy of one title"`,
				`    slice "Borrow Copy" {`,
				`    }`,
				`  }`,
				`}`,
				``,
			}, "\n")

			result := formatter.Format(parseModel(t, input, "library-lending.emod"))

			expected := strings.Join([]string{
				`emod 1`,
				`model "Library Lending"`,
				``,
				`context "Lending" {`,
				`  aggregate "Loan" {`,
				`    # A shelf at a time would let a member empty a series`,
				`    invariant OneCopyPerLoan "A loan covers exactly one copy of one title"`,
				`    slice "Borrow Copy" {`,
				`    }`,
				`  }`,
				`}`,
				``,
			}, "\n")

			require.Equal(t, expected, result)
		})

		t.Run("comment above a spec stays directly above it at the spec's indentation", func(t *testing.T) {
			input := strings.Join([]string{
				`model "Library Lending"`,
				``,
				`context "Lending" {`,
				`  aggregate "Loan" {`,
				`    slice "Borrow Copy" {`,
				`      # The desk clerk walks new starters through this one`,
				`      spec "borrows a copy no one holds" {`,
				`        when BorrowCopy`,
				`        then [CopyBorrowed]`,
				`      }`,
				`    }`,
				`  }`,
				`}`,
				``,
			}, "\n")

			result := formatter.Format(parseModel(t, input, "specs.emod"))

			expected := strings.Join([]string{
				`emod 1`,
				`model "Library Lending"`,
				``,
				`context "Lending" {`,
				`  aggregate "Loan" {`,
				`    slice "Borrow Copy" {`,
				`      # The desk clerk walks new starters through this one`,
				`      spec "borrows a copy no one holds" {`,
				`        when BorrowCopy`,
				`        then [CopyBorrowed]`,
				`      }`,
				`    }`,
				`  }`,
				`}`,
				``,
			}, "\n")

			require.Equal(t, expected, result)
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

	t.Run("specs", func(t *testing.T) {
		t.Run("every spec block sits after the slice's other entries, in the order it was declared", func(t *testing.T) {
			input := strings.Join([]string{
				`model "Library Lending"`,
				``,
				`context "Lending" {`,
				`  aggregate "Loan" {`,
				`    slice "Borrow Copy" {`,
				`      spec "borrows a copy no one holds" {`,
				`        when BorrowCopy`,
				`        then [CopyBorrowed]`,
				`      }`,
				`      command BorrowCopy {`,
				`      }`,
				`      spec "refuses a copy already on loan" {`,
				`        given [CopyBorrowed]`,
				`        when BorrowCopy`,
				`        then rejected OneCopyPerLoan`,
				`      }`,
				`      event CopyBorrowed {`,
				`      }`,
				`      flow {`,
				`        command -> event: BorrowCopy -> CopyBorrowed`,
				`      }`,
				`    }`,
				`  }`,
				`}`,
				``,
			}, "\n")

			result := formatter.Format(parseModel(t, input, "specs.emod"))

			expected := strings.Join([]string{
				`emod 1`,
				`model "Library Lending"`,
				``,
				`context "Lending" {`,
				`  aggregate "Loan" {`,
				`    slice "Borrow Copy" {`,
				`      command BorrowCopy {`,
				`      }`,
				``,
				`      event CopyBorrowed {`,
				`      }`,
				``,
				`      flow {`,
				`        command -> event: BorrowCopy -> CopyBorrowed`,
				`      }`,
				``,
				`      spec "borrows a copy no one holds" {`,
				`        when BorrowCopy`,
				`        then [CopyBorrowed]`,
				`      }`,
				``,
				`      spec "refuses a copy already on loan" {`,
				`        given [CopyBorrowed]`,
				`        when BorrowCopy`,
				`        then rejected OneCopyPerLoan`,
				`      }`,
				`    }`,
				`  }`,
				`}`,
				``,
			}, "\n")

			require.Equal(t, expected, result)
		})

		t.Run("a spec whose entries are written out of order is emitted as given, when, then", func(t *testing.T) {
			input := strings.Join([]string{
				`model "Library Lending"`,
				``,
				`context "Lending" {`,
				`  aggregate "Loan" {`,
				`    slice "Return Copy" {`,
				`      spec "returns a copy the member holds" {`,
				`        given [CopyBorrowed]`,
				`        then [CopyReturned]`,
				`        when ReturnCopy`,
				`      }`,
				`    }`,
				`  }`,
				`}`,
				``,
			}, "\n")
			original := parseModel(t, input, "specs.emod")

			result := formatter.Format(original)

			expected := strings.Join([]string{
				`emod 1`,
				`model "Library Lending"`,
				``,
				`context "Lending" {`,
				`  aggregate "Loan" {`,
				`    slice "Return Copy" {`,
				`      spec "returns a copy the member holds" {`,
				`        given [CopyBorrowed]`,
				`        when ReturnCopy`,
				`        then [CopyReturned]`,
				`      }`,
				`    }`,
				`  }`,
				`}`,
				``,
			}, "\n")

			require.Equal(t, expected, result)
			test.RequireEqual(t, original, parseModel(t, result, "formatted.emod"), ignoreFormatterNormalizations)
		})

		t.Run("a spec that states no when omits the when line and round-trips", func(t *testing.T) {
			input := strings.Join([]string{
				`model "Library Lending"`,
				``,
				`context "Lending" {`,
				`  aggregate "Loan" {`,
				`    slice "Review Member Loans" {`,
				`      spec "lists the loans a member holds" {`,
				`        then view MemberLoansView`,
				`      }`,
				`    }`,
				`  }`,
				`}`,
				``,
			}, "\n")
			original := parseModel(t, input, "specs.emod")

			result := formatter.Format(original)

			expected := strings.Join([]string{
				`emod 1`,
				`model "Library Lending"`,
				``,
				`context "Lending" {`,
				`  aggregate "Loan" {`,
				`    slice "Review Member Loans" {`,
				`      spec "lists the loans a member holds" {`,
				`        then view MemberLoansView`,
				`      }`,
				`    }`,
				`  }`,
				`}`,
				``,
			}, "\n")

			require.Equal(t, expected, result)
			reparsed := parseModel(t, result, "formatted.emod")
			test.RequireEqual(t, original, reparsed, ignoreFormatterNormalizations)
			require.Nil(t, reparsed.Contexts[0].Aggregates[0].Slices[0].Specs[0].When)
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

			requireStableFormat(t, parseModel(t, input, "test.emod"))
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

func orderNotifierSource(automationBody ...string) string {
	return fulfilmentSliceSource("Notify Customer", slices.Concat(
		[]string{`      automation OrderNotifier {`},
		automationBody,
		[]string{`      }`},
	))
}

func expirySweepSliceSource(sliceBody ...string) string {
	return fulfilmentSliceSource("Sweep Expired Holds", sliceBody)
}

func fulfilmentSliceSource(sliceName string, sliceBody []string) string {
	return strings.Join(slices.Concat(
		[]string{
			`model "Order Fulfilment"`,
			``,
			`context "Fulfilment" {`,
			`  aggregate "Shipment" {`,
			`    slice "` + sliceName + `" {`,
		},
		sliceBody,
		[]string{
			`    }`,
			`  }`,
			`}`,
			``,
		},
	), "\n")
}

func requireStableFormat(t *testing.T, model *ast.Model) *ast.Model {
	t.Helper()

	formatted := formatter.Format(model)
	reparsed := parseModel(t, formatted, "formatted.emod")

	require.Equal(t, formatted, formatter.Format(reparsed),
		"formatting the already-formatted output should produce identical bytes")

	return reparsed
}

func withoutReadsLines(formatted string) string {
	kept := slices.DeleteFunc(strings.Split(formatted, "\n"), func(line string) bool {
		return strings.HasPrefix(strings.TrimSpace(line), "reads ")
	})

	return strings.Join(kept, "\n")
}

func lineAfter(t *testing.T, formatted, blockHeader string) string {
	t.Helper()

	lines := strings.Split(formatted, "\n")
	header := indexOfLine(t, lines, 0, blockHeader)
	require.Less(t, header+1, len(lines), "nothing follows %q in:\n%s", blockHeader, formatted)

	return strings.TrimSpace(lines[header+1])
}

func requireFieldsBlockAfter(t *testing.T, formatted, blockHeader string, expected ...string) {
	t.Helper()

	lines := strings.Split(formatted, "\n")
	header := indexOfLine(t, lines, 0, blockHeader)
	start := indexOfLine(t, lines, header+1, "fields {")
	end := indexOfLine(t, lines, start+1, "}")

	require.Equal(t, strings.Join(expected, "\n"), strings.Join(lines[start:end+1], "\n"))
}

func indexOfLine(t *testing.T, lines []string, from int, text string) int {
	t.Helper()

	for i := from; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == text {
			return i
		}
	}

	require.FailNowf(t, "line not found in formatted output", "%q at or after line %d in:\n%s",
		text, from+1, strings.Join(lines, "\n"))
	return -1
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
	cmpopts.IgnoreFields(ast.Rejection{}, "Comments"),
	cmpopts.IgnoreFields(ast.Trigger{}, "Comments"),
	cmpopts.IgnoreFields(ast.View{}, "Comments"),
	cmpopts.IgnoreFields(ast.Automation{}, "Comments"),
	cmpopts.IgnoreFields(ast.Translation{}, "Comments"),
}

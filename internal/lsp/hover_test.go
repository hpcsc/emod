//go:build unit

package lsp_test

import (
	"strings"
	"testing"

	"github.com/hpcsc/emod/internal/lsp"
	"github.com/hpcsc/emod/internal/test"
	"github.com/stretchr/testify/require"
)

func TestGetHover(t *testing.T) {
	const testDoc = `context "Orders" {
    aggregate "Sales" {
        slice "OrderSlice" {
            command SubmitOrder {
            }
            event OrderSubmitted {
                fields {
                    id String required
                    amount Number required
                }
            }
            event EmptyEvent {
            }
            view OrderView {
                subscribes [OrderSubmitted]
            }
            view NoSubscribeView {
            }
        }
    }
}
`

	const automationDoc = `context "Warehouse" {
    aggregate "Hold" {
        slice "Expire Holds" {
            event HoldSwept {
                fields {
                    every string required
                }
            }
            automation ReleaseOnPickup {
                on ItemPickedUp
                command ReleaseHold
            }
            automation SweepStaleHolds {
                every "0 * * * *"
                reads PendingExpiries
                command ExpireHold
            }
        }
    }
}`

	const everyDescription = `Sets the schedule that activates the automation: a duration such as "5m", or a five-field cron expression such as "0 2 * * *".`

	assertHover := func(t *testing.T, doc string, cLine, cChar int, expectedContent string) string {
		t.Helper()
		hover := lsp.GetHover(doc, cLine, cChar)
		require.NotNil(t, hover, "expected hover for cursor at (%d,%d)", cLine, cChar)
		require.Equal(t, lsp.Markdown, hover.Contents.Kind)
		require.Equal(t, expectedContent, hover.Contents.Value)
		return hover.Contents.Value
	}

	assertNil := func(t *testing.T, doc string, cLine, cChar int) {
		t.Helper()
		hover := lsp.GetHover(doc, cLine, cChar)
		require.Nil(t, hover, "expected nil for cursor at (%d,%d)", cLine, cChar)
	}

	t.Run("command name shows parent context and aggregate", func(t *testing.T) {
		cLine, cChar := posIn(t, testDoc, "command SubmitOrder", "SubmitOrder")
		assertHover(t, testDoc, cLine, cChar, "**Command** in Orders > Sales")
	})

	t.Run("event name shows parent context, aggregate, and fields", func(t *testing.T) {
		cLine, cChar := posIn(t, testDoc, "event OrderSubmitted", "OrderSubmitted")
		assertHover(t, testDoc, cLine, cChar, "**Event** in Orders > Sales\n\n**Fields:**\n- id String required\n- amount Number required")
	})

	t.Run("event without fields omits fields section", func(t *testing.T) {
		cLine, cChar := posIn(t, testDoc, "event EmptyEvent", "EmptyEvent")
		assertHover(t, testDoc, cLine, cChar, "**Event** in Orders > Sales")
	})

	t.Run("view name shows parent context, aggregate, and subscribed events", func(t *testing.T) {
		cLine, cChar := posIn(t, testDoc, "view OrderView", "OrderView")
		assertHover(t, testDoc, cLine, cChar, "**View** in Orders > Sales\n\n**Subscribes:**\n- OrderSubmitted")
	})

	t.Run("view without subscriptions omits subscribes section", func(t *testing.T) {
		cLine, cChar := posIn(t, testDoc, "view NoSubscribeView", "NoSubscribeView")
		assertHover(t, testDoc, cLine, cChar, "**View** in Orders > Sales")
	})

	t.Run("names the context alone for a slice hanging off a dcb context, and the aggregate too where there is one", func(t *testing.T) {
		doc := test.AutomationReadsLibraryLending

		for _, tc := range []struct {
			container string
			name      string
			expected  string
		}{
			{
				container: "command ClaimDesk",
				name:      "ClaimDesk",
				expected:  "**Command** in Reading Room",
			},
			{
				container: "event DeskClaimed",
				name:      "DeskClaimed",
				expected: "**Event** in Reading Room\n\n**Fields:**" +
					"\n- sessionId string required" +
					"\n- deskId string required" +
					"\n- memberId string required" +
					"\n- claimedAt timestamp required",
			},
			{
				container: "view DeskOccupancyView",
				name:      "DeskOccupancyView",
				expected:  "**View** in Reading Room\n\n**Subscribes:**\n- DeskClaimed\n- DeskReleased",
			},
			{
				container: "command BorrowCopy",
				name:      "BorrowCopy",
				expected:  "**Command** in Lending > Loan",
			},
			{
				container: "event CopyBorrowed",
				name:      "CopyBorrowed",
				expected: "**Event** in Lending > Loan\n\n**Fields:**" +
					"\n- loanId string required" +
					"\n- memberId string required" +
					"\n- copyId string required" +
					"\n- dueOn date required",
			},
			{
				container: "view MemberLoansView",
				name:      "MemberLoansView",
				expected:  "**View** in Lending > Loan\n\n**Subscribes:**\n- CopyBorrowed",
			},
			{
				container: `slice "Claim Desk"`,
				name:      "Claim Desk",
				expected:  "**Slice** in Reading Room",
			},
			{
				container: `slice "Borrow Copy"`,
				name:      "Borrow Copy",
				expected:  "**Slice** in Lending > Loan",
			},
		} {
			cLine, cChar := posIn(t, doc, tc.container, tc.name)
			hover := lsp.GetHover(doc, cLine, cChar)
			require.NotNil(t, hover, "expected hover on %s", tc.name)
			require.Equal(t, lsp.Markdown, hover.Contents.Kind)
			require.Equal(t, tc.expected, hover.Contents.Value, "hover on %s", tc.name)
		}
	})

	t.Run("a construct's description rides on its hover, and its undescribed twin's hover is the same without it", func(t *testing.T) {
		for _, tc := range []struct {
			construct   string
			container   string
			name        string
			undescribed string
			description string
		}{
			{
				construct:   "model",
				container:   `model "Hotel Reservation"`,
				name:        "Hotel Reservation",
				undescribed: "**Model**",
				description: "How the hotel takes, confirms and imports room bookings",
			},
			{
				construct:   "actor",
				container:   `actor "Guest"`,
				name:        "Guest",
				undescribed: "**Actor**",
				description: "A person booking a room, not necessarily the one staying in it",
			},
			{
				construct:   "context",
				container:   `context "Reservations"`,
				name:        "Reservations",
				undescribed: "**Context**",
				description: "Everything the hotel knows about a stay before the guest arrives",
			},
			{
				construct:   "aggregate",
				container:   `aggregate "Reservation"`,
				name:        "Reservation",
				undescribed: "**Aggregate** in Reservations",
				description: "One guest holding one room over one date range",
			},
			{
				construct:   "slice",
				container:   `slice "Make Reservation"`,
				name:        "Make Reservation",
				undescribed: "**Slice** in Reservations > Reservation",
				description: "A guest books a room from the public site",
			},
			{
				construct:   "trigger",
				container:   `trigger "Reservation Form"`,
				name:        "Reservation Form",
				undescribed: "**Trigger** in Reservations > Reservation",
				description: "The booking form on the public site",
			},
			{
				construct:   "command",
				container:   "command MakeReservation",
				name:        "MakeReservation",
				undescribed: "**Command** in Reservations > Reservation",
				description: "Ask the hotel to hold a room for a date range, 10% deposit taken up front",
			},
			{
				construct: "event",
				container: "event ReservationMade",
				name:      "ReservationMade",
				undescribed: "**Event** in Reservations > Reservation\n\n**Fields:**" +
					"\n- reservationId string required" +
					"\n- guestId string required" +
					"\n- roomType string required" +
					"\n- checkIn date required" +
					"\n- checkOut date required" +
					"\n- status string required",
				description: "A room is held for a guest",
			},
			{
				construct:   "view",
				container:   "view ReservationsView",
				name:        "ReservationsView",
				undescribed: "**View** in Reservations > Reservation\n\n**Subscribes:**\n- ReservationMade",
				description: "Every reservation with the stage it has reached",
			},
			{
				construct:   "automation",
				container:   "automation AutoConfirm",
				name:        "AutoConfirm",
				undescribed: "**Automation** in Reservations > Reservation",
				description: "Confirms every reservation the moment it is made",
			},
			{
				construct:   "translation",
				container:   "translation BookingImport",
				name:        "BookingImport",
				undescribed: "**Translation** in Reservations > Reservation",
				description: "Restates a partner webhook in the hotel's own language",
			},
			{
				construct: "event a translation declares",
				container: "event BookingImported",
				name:      "BookingImported",
				undescribed: "**Event** in Reservations > Reservation\n\n**Fields:**" +
					"\n- bookingId string required" +
					"\n- hotelName string required" +
					"\n- bookingRef string required",
				description: "A partner site reported a booking",
			},
		} {
			t.Run("a "+tc.construct+" name", func(t *testing.T) {
				plainLine, plainChar := posIn(t, test.HotelReservation, tc.container, tc.name)
				assertHover(t, test.HotelReservation, plainLine, plainChar, tc.undescribed)

				describedLine, describedChar := posIn(t, test.DescribedHotelReservation, tc.container, tc.name)
				assertHover(
					t, test.DescribedHotelReservation, describedLine, describedChar,
					describedHeading(tc.undescribed, tc.description),
				)
			})
		}
	})

	t.Run("a description keeps the characters that delimit code elsewhere", func(t *testing.T) {
		const doc = `context "Reservations" {
    description "A # and a // and a { brace }"
    aggregate "Reservation" {
    }
}`
		line, char := posIn(t, doc, `context "Reservations"`, "Reservations")
		assertHover(t, doc, line, char, "**Context**\n\nA # and a // and a { brace }")
	})

	t.Run("the description keyword still describes itself", func(t *testing.T) {
		doc := test.DescribedHotelReservation
		line, char := posIn(t, doc, `description "Everything the hotel knows`, "description")
		assertHover(t, doc, line, char, "Attaches a human-readable description to the enclosing declaration.")
	})

	t.Run("hover sees a translation's own event, and navigation still does not", func(t *testing.T) {
		const doc = `context "Reservations" {
    aggregate "Reservation" {
        slice "Import External Booking" {
            event ReservationMade {
            }
            view ImportsView {
                subscribes [ReservationMade, BookingImported]
            }
            translation BookingImport {
                external_system "Booking.com API"
                event BookingImported {
                }
            }
        }
    }
}`
		uri := "file:///imports.emod"

		madeLine, madeChar := posIn(t, doc, "subscribes [ReservationMade, BookingImported]", "ReservationMade")
		require.NotNil(t, lsp.GetDefinition(doc, madeLine, madeChar, uri), "the slice-declared event is the control")

		subLine, subChar := posIn(t, doc, "subscribes [ReservationMade, BookingImported]", "BookingImported")
		require.Nil(t, lsp.GetDefinition(doc, subLine, subChar, uri))

		declLine, declChar := posIn(t, doc, "event BookingImported", "BookingImported")
		require.NotNil(t, lsp.GetHover(doc, declLine, declChar), "hover answers where navigation does not")
		require.Nil(t, lsp.GetReferences(doc, declLine, declChar, uri))
	})

	t.Run("a construct name in a document the parser could not finish still hovers", func(t *testing.T) {
		const doc = `context "Reservations" {
    aggregate "Reservation" {
        slice
`
		line, char := posIn(t, doc, `context "Reservations"`, "Reservations")
		hover := lsp.GetHover(doc, line, char)

		require.NotNil(t, hover, "the context name is recovered even from the truncated document")
		require.Equal(t, "**Context**", hover.Contents.Value)

		sliceLine, sliceChar := posIn(t, doc, "        slice", "slice")
		assertHover(t, doc, sliceLine, sliceChar, "Defines a slice within an aggregate.")
	})

	t.Run("a quoted name resolves from either end and reports a range covering neither quote", func(t *testing.T) {
		doc := test.DescribedHotelReservation
		const name = "Make Reservation"

		quoteLine, quoteChar := posIn(t, doc, `slice "Make Reservation"`, `"`)
		nameStart := quoteChar + 1

		expected := &lsp.Range{
			Start: lsp.Position{Line: quoteLine, Character: nameStart},
			End:   lsp.Position{Line: quoteLine, Character: nameStart + len(name)},
		}

		first := lsp.GetHover(doc, quoteLine, nameStart)
		require.NotNil(t, first, "cursor on the name's first character")
		require.Equal(t, expected, first.Range)

		last := lsp.GetHover(doc, quoteLine, nameStart+len(name)-1)
		require.NotNil(t, last, "cursor on the name's last character")
		require.Equal(t, expected, last.Range)

		require.Nil(t, lsp.GetHover(doc, quoteLine, quoteChar), "cursor on the opening quote")
		require.Nil(t, lsp.GetHover(doc, quoteLine, nameStart+len(name)), "cursor on the closing quote")
	})

	t.Run("a quoted name holding non-ASCII text measures itself in characters", func(t *testing.T) {
		const doc = `context "Réservations" {
    aggregate "Séjour" {
    }
}`
		const name = "Réservations"

		_, quoteChar := posIn(t, doc, `context "`, `"`)
		nameStart := quoteChar + 1
		lastChar := nameStart + len([]rune(name)) - 1

		first := lsp.GetHover(doc, 0, nameStart)
		require.NotNil(t, first, "cursor on the name's first character")
		require.Equal(t, &lsp.Range{
			Start: lsp.Position{Line: 0, Character: nameStart},
			End:   lsp.Position{Line: 0, Character: lastChar + 1},
		}, first.Range)

		require.NotNil(t, lsp.GetHover(doc, 0, lastChar), "cursor on the name's last character")
		require.Nil(t, lsp.GetHover(doc, 0, lastChar+1), "cursor on the closing quote")
	})

	t.Run("cursor on keyword returns description", func(t *testing.T) {
		line, char := posIn(t, testDoc, "command SubmitOrder", "command")
		assertHover(t, testDoc, line, char, "Defines a command that can be sent to an aggregate.")
	})

	t.Run("a block keyword answers as a keyword while the name beside it answers as a declaration", func(t *testing.T) {
		doc := test.DescribedHotelReservation

		for _, tc := range []struct {
			keyword     string
			container   string
			name        string
			description string
		}{
			{keyword: "context", container: `context "Reservations"`, name: "Reservations", description: "Defines a bounded context."},
			{keyword: "aggregate", container: `aggregate "Reservation"`, name: "Reservation", description: "Defines an aggregate root."},
			{keyword: "slice", container: `slice "Make Reservation"`, name: "Make Reservation", description: "Defines a slice within an aggregate."},
		} {
			t.Run("the "+tc.keyword+" keyword", func(t *testing.T) {
				keywordLine, keywordChar := posIn(t, doc, tc.container, tc.keyword)
				assertHover(t, doc, keywordLine, keywordChar, tc.description)

				nameLine, nameChar := posIn(t, doc, tc.container, tc.name)
				nameHover := lsp.GetHover(doc, nameLine, nameChar)
				require.NotNil(t, nameHover)
				require.NotEqual(t, tc.description, nameHover.Contents.Value)
			})
		}
	})

	t.Run("on and every each describe how the automation they sit in is activated", func(t *testing.T) {
		onLine, onChar := posIn(t, automationDoc, "automation ReleaseOnPickup", "on ItemPickedUp")
		assertHover(t, automationDoc, onLine, onChar, "Names the event whose occurrence activates the automation.")

		everyLine, everyChar := posIn(t, automationDoc, "automation SweepStaleHolds", `every "0 * * * *"`)
		assertHover(t, automationDoc, everyLine, everyChar, everyDescription)
	})

	t.Run("automation names its pattern, both activation forms and the view it reads", func(t *testing.T) {
		line, char := posIn(t, automationDoc, "automation SweepStaleHolds", "automation")
		content := assertHover(
			t, automationDoc, line, char,
			"Defines an automation, the reactive processor of the Automation pattern: activated by an on event or an every schedule, optionally reads a view, and sends a command.",
		)

		require.Contains(t, content, "Automation pattern")
		require.Regexp(t, `\bon\b`, content)
		require.Regexp(t, `\bevery\b`, content)
		require.Contains(t, content, "reads a view")
	})

	t.Run("trigger names the Command pattern and no kind between the keyword and the name", func(t *testing.T) {
		doc := `context "Warehouse" {
    aggregate "Hold" {
        slice "Place Hold" {
            trigger "Hold Desk" {
                actor Member
                reads AvailableCopiesView
            }
            command PlaceHold {
            }
        }
    }
}`
		line, char := posIn(t, doc, `trigger "Hold Desk"`, "trigger")
		content := assertHover(
			t, doc, line, char,
			"Defines a trigger, the Command pattern's human entry point into a slice: the actor who acts and the view they read.",
		)

		require.Contains(t, content, "Command pattern")
		for _, retiredKind := range []string{"UI", "Schedule", "Processor"} {
			require.NotContains(t, content, retiredKind)
		}
	})

	t.Run("reads names every block that accepts it", func(t *testing.T) {
		line, char := posIn(t, automationDoc, "reads PendingExpiries", "reads")
		content := assertHover(
			t, automationDoc, line, char,
			"Defines the view a trigger, automation or translation reads from.",
		)

		for _, block := range []string{"trigger", "automation", "translation"} {
			require.Regexp(t, `\b`+block+`\b`, content)
		}
	})

	t.Run("a field named every hovers as the every keyword", func(t *testing.T) {
		fieldLine, fieldChar := posIn(t, automationDoc, "fields {", "every string required")
		assertHover(t, automationDoc, fieldLine, fieldChar, everyDescription)

		scheduleLine, scheduleChar := posIn(t, automationDoc, "automation SweepStaleHolds", `every "0 * * * *"`)
		assertHover(t, automationDoc, scheduleLine, scheduleChar, everyDescription)
	})

	// The story asks hover to answer on any construct, and a spec's quoted name is
	// a declaration like a slice's. It carries no description, so kind and scope
	// are the whole of it.
	t.Run("a spec name hovers as a spec, in both homes a slice has", func(t *testing.T) {
		doc := test.SpecLibraryLending

		aggLine, aggChar := posIn(t, doc, `spec "borrows a copy no one holds"`, "borrows a copy no one holds")
		assertHover(t, doc, aggLine, aggChar, "**Spec** in Lending > Loan")

		dcbLine, dcbChar := posIn(t, doc, `spec "seats a reader at a free desk"`, "seats a reader at a free desk")
		assertHover(t, doc, dcbLine, dcbChar, "**Spec** in Reading Room")
	})

	t.Run("an invariant states its prose where it is declared and where a spec rejects by it", func(t *testing.T) {
		doc := test.SpecLibraryLending

		for _, tc := range []struct {
			home      string
			invariant string
			expected  string
		}{
			{
				home:      "declared on an aggregate",
				invariant: "OneCopyPerLoan",
				expected:  "**Invariant** in Lending > Loan\n\nA loan covers exactly one copy of one title",
			},
			{
				home:      "declared directly on a mode dcb context",
				invariant: "OneReaderPerDesk",
				expected:  "**Invariant** in Reading Room\n\nA desk seats at most one reader at any moment",
			},
		} {
			t.Run(tc.home, func(t *testing.T) {
				declLine, declChar := posIn(t, doc, "invariant "+tc.invariant, tc.invariant)
				assertHover(t, doc, declLine, declChar, tc.expected)

				refLine, refChar := posIn(t, doc, "then rejected "+tc.invariant, tc.invariant)
				reference := lsp.GetHover(doc, refLine, refChar)
				require.NotNil(t, reference)
				require.Equal(t, tc.expected, reference.Contents.Value)

				// The prose comes from the declaration; the range must not, or
				// the editor would highlight a span the caret is nowhere near.
				require.Equal(t, &lsp.Range{
					Start: lsp.Position{Line: refLine, Character: refChar},
					End:   lsp.Position{Line: refLine, Character: refChar + len(tc.invariant)},
				}, reference.Range)
			})
		}
	})

	t.Run("two scopes declaring one invariant name each answer with their own statement", func(t *testing.T) {
		const doc = `context "Lending" {
    aggregate "Loan" {
        invariant OneAtATime "A loan covers exactly one copy of one title"
        slice "Borrow Copy" {
            command BorrowCopy {
            }
            spec "refuses a second copy of the title" {
                when BorrowCopy
                then rejected OneAtATime
            }
        }
    }
    aggregate "Hold" {
        invariant OneAtATime "A member holds at most one title back at a time"
        slice "Place Hold" {
            command PlaceHold {
            }
            spec "refuses a second hold" {
                when PlaceHold
                then rejected OneAtATime
            }
        }
    }
}`

		loanLine, loanChar := posIn(t, doc, `spec "refuses a second copy of the title"`, "OneAtATime")
		assertHover(t, doc, loanLine, loanChar, "**Invariant** in Lending > Loan\n\nA loan covers exactly one copy of one title")

		holdLine, holdChar := posIn(t, doc, `spec "refuses a second hold"`, "OneAtATime")
		assertHover(t, doc, holdLine, holdChar, "**Invariant** in Lending > Hold\n\nA member holds at most one title back at a time")
	})

	t.Run("an aggregate does not resolve against the invariants of the context enclosing it", func(t *testing.T) {
		const doc = `context "Lending" {
    invariant CardInGoodStanding "A member borrows only while their card is in good standing"
    aggregate "Loan" {
        invariant OneCopyPerLoan "A loan covers exactly one copy of one title"
        slice "Borrow Copy" {
            command BorrowCopy {
            }
            spec "refuses a copy already on loan" {
                when BorrowCopy
                then rejected OneCopyPerLoan
            }
            spec "refuses a member whose card has lapsed" {
                when BorrowCopy
                then rejected CardInGoodStanding
            }
        }
    }
}`

		ownLine, ownChar := posIn(t, doc, "then rejected OneCopyPerLoan", "OneCopyPerLoan")
		assertHover(t, doc, ownLine, ownChar, "**Invariant** in Lending > Loan\n\nA loan covers exactly one copy of one title")

		enclosingLine, enclosingChar := posIn(t, doc, "then rejected CardInGoodStanding", "CardInGoodStanding")
		assertNil(t, doc, enclosingLine, enclosingChar)
	})

	t.Run("a rejection naming an invariant of another scope hovers nothing", func(t *testing.T) {
		const doc = `context "Lending" {
    aggregate "Loan" {
        invariant OneCopyPerLoan "A loan covers exactly one copy of one title"
        slice "Borrow Copy" {
            command BorrowCopy {
            }
        }
    }
    aggregate "Hold" {
        slice "Place Hold" {
            command PlaceHold {
            }
            spec "refuses a second hold" {
                when PlaceHold
                then rejected OneCopyPerLoan
            }
        }
    }
}`

		declLine, declChar := posIn(t, doc, "invariant OneCopyPerLoan", "OneCopyPerLoan")
		assertHover(t, doc, declLine, declChar, "**Invariant** in Lending > Loan\n\nA loan covers exactly one copy of one title")

		refLine, refChar := posIn(t, doc, "then rejected OneCopyPerLoan", "OneCopyPerLoan")
		assertNil(t, doc, refLine, refChar)
	})

	t.Run("the rejected keyword and the invariant name beside it answer differently", func(t *testing.T) {
		doc := test.SpecLibraryLending

		keywordLine, keywordChar := posIn(t, doc, "then rejected OneCopyPerLoan", "rejected")
		assertHover(t, doc, keywordLine, keywordChar, "States that a spec's command is rejected by the named invariant.")

		nameLine, nameChar := posIn(t, doc, "then rejected OneCopyPerLoan", "OneCopyPerLoan")
		assertHover(t, doc, nameLine, nameChar, "**Invariant** in Lending > Loan\n\nA loan covers exactly one copy of one title")
	})

	t.Run("a flow rejection edge names its invariant with the same prose its declaration carries", func(t *testing.T) {
		doc := test.RejectionLibraryLending

		for _, tc := range []struct {
			home      string
			invariant string
			edge      string
		}{
			{
				home:      "declared on an aggregate",
				invariant: "OneCopyPerLoan",
				edge:      "command -> rejected: BorrowCopy -> OneCopyPerLoan",
			},
			{
				home:      "declared directly on a mode dcb context",
				invariant: "OneReaderPerDesk",
				edge:      "command -> rejected: ClaimDesk -> OneReaderPerDesk",
			},
		} {
			t.Run(tc.home, func(t *testing.T) {
				declLine, declChar := posIn(t, doc, "invariant "+tc.invariant, tc.invariant)
				declaration := lsp.GetHover(doc, declLine, declChar)
				require.NotNil(t, declaration)

				edgeLine, edgeChar := posIn(t, doc, tc.edge, tc.invariant)
				edge := lsp.GetHover(doc, edgeLine, edgeChar)
				require.NotNil(t, edge, "expected hover on the rejection edge's invariant")

				require.Equal(t, declaration.Contents, edge.Contents)
			})
		}
	})

	t.Run("a rejection edge naming an invariant of another scope hovers nothing", func(t *testing.T) {
		const doc = `context "Lending" {
    aggregate "Loan" {
        invariant OneCopyPerLoan "A loan covers exactly one copy of one title"
        slice "Borrow Copy" {
            command BorrowCopy {
            }
            event CopyBorrowed {
            }
            flow {
                command -> event: BorrowCopy -> CopyBorrowed
            }
        }
    }
    aggregate "Hold" {
        slice "Place Hold" {
            command PlaceHold {
            }
            event HoldPlaced {
            }
            flow {
                command -> rejected: PlaceHold -> OneCopyPerLoan
                command -> event: PlaceHold -> HoldPlaced
            }
        }
    }
}`
		line, char := posIn(t, doc, "command -> rejected: PlaceHold -> OneCopyPerLoan", "OneCopyPerLoan")
		assertNil(t, doc, line, char)
	})

	t.Run("cursor on non-resolvable token returns nil", func(t *testing.T) {
		// Cursor on the identifier "required" which is a field modifier,
		// not a resolvable definition name.
		line, char := posIn(t, testDoc, "id String required", "required")
		assertNil(t, testDoc, line, char)
	})

	t.Run("cursor on punctuation and whitespace returns nil", func(t *testing.T) {
		// Neither the brace that opens a block nor the space before it belongs to
		// any name or keyword. The quotes around a name are covered separately,
		// by the leaf that pins a quoted name's boundaries.
		_, braceChar := posIn(t, testDoc, `context "Orders" {`, `{`)
		assertNil(t, testDoc, 0, braceChar)
		assertNil(t, testDoc, 0, braceChar-1)
	})

	t.Run("empty document returns nil", func(t *testing.T) {
		assertNil(t, "", 0, 0)
	})

	t.Run("hover range covers definition name", func(t *testing.T) {
		cLine, cChar := posIn(t, testDoc, "command SubmitOrder", "SubmitOrder")
		hover := lsp.GetHover(testDoc, cLine, cChar)
		require.NotNil(t, hover)
		require.NotNil(t, hover.Range)
		require.Equal(t, cLine, hover.Range.Start.Line)
		require.Equal(t, cChar, hover.Range.Start.Character)
		require.Equal(t, cLine, hover.Range.End.Line)
		require.Equal(t, cChar+len("SubmitOrder"), hover.Range.End.Character)
	})
}

// describedHeading states what a description is expected to cost a hover: the
// same value the undescribed twin returns, with the description standing between
// the heading and whatever bullet sections follow it.
func describedHeading(undescribed, description string) string {
	heading, sections, hasSections := strings.Cut(undescribed, "\n\n")
	if !hasSections {
		return undescribed + "\n\n" + description
	}
	return heading + "\n\n" + description + "\n\n" + sections
}

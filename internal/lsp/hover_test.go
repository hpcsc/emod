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

	t.Run("every construct hovers as the kind it is", func(t *testing.T) {
		doc := test.DescribedHotelReservation

		for _, tc := range []struct {
			construct string
			container string
			name      string
			kind      string
		}{
			{construct: "model", container: `model "Hotel Reservation"`, name: "Hotel Reservation", kind: "**Model**"},
			{construct: "actor", container: `actor "Guest"`, name: "Guest", kind: "**Actor**"},
			{construct: "context", container: `context "Reservations"`, name: "Reservations", kind: "**Context**"},
			{construct: "aggregate", container: `aggregate "Reservation"`, name: "Reservation", kind: "**Aggregate** in Reservations"},
			{construct: "slice", container: `slice "Make Reservation"`, name: "Make Reservation", kind: "**Slice** in Reservations > Reservation"},
			{construct: "trigger", container: `trigger "Reservation Form"`, name: "Reservation Form", kind: "**Trigger** in Reservations > Reservation"},
			{construct: "automation", container: "automation AutoConfirm", name: "AutoConfirm", kind: "**Automation** in Reservations > Reservation"},
			{construct: "translation", container: "translation BookingImport", name: "BookingImport", kind: "**Translation** in Reservations > Reservation"},
		} {
			t.Run("a "+tc.construct+" name", func(t *testing.T) {
				line, char := posIn(t, doc, tc.container, tc.name)
				hover := lsp.GetHover(doc, line, char)

				require.NotNil(t, hover, "expected hover on the %s name", tc.construct)
				require.Equal(t, lsp.Markdown, hover.Contents.Kind)
				require.True(
					t, strings.HasPrefix(hover.Contents.Value, tc.kind),
					"hover on the %s name reads %q, which does not open with %q", tc.construct, hover.Contents.Value, tc.kind,
				)
			})
		}
	})

	t.Run("an event a translation declares inside itself hovers like an event declared on a slice", func(t *testing.T) {
		doc := test.DescribedHotelReservation

		line, char := posIn(t, doc, "translation BookingImport", "BookingImported")
		hover := lsp.GetHover(doc, line, char)

		require.NotNil(t, hover)
		require.True(
			t,
			strings.HasPrefix(hover.Contents.Value, "**Event** in Reservations > Reservation"),
			"hover on a translation's own event reads %q", hover.Contents.Value,
		)
		require.Contains(t, hover.Contents.Value, "**Fields:**\n- bookingId string required\n- hotelName string required\n- bookingRef string required")
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

	t.Run("cursor on non-resolvable token returns nil", func(t *testing.T) {
		// Cursor on the identifier "required" which is a field modifier,
		// not a resolvable definition name.
		line, char := posIn(t, testDoc, "id String required", "required")
		assertNil(t, testDoc, line, char)
	})

	t.Run("cursor not on any definition returns nil", func(t *testing.T) {
		// The closing quote of `context "Orders"` delimits a name rather than
		// belonging to it, so it resolves to no declaration and to no keyword.
		_, quoteChar := posIn(t, testDoc, `context "Orders"`, `"`)
		assertNil(t, testDoc, 0, quoteChar+1+len("Orders"))
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

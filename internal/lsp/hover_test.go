//go:build unit

package lsp_test

import (
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
		} {
			cLine, cChar := posIn(t, doc, tc.container, tc.name)
			hover := lsp.GetHover(doc, cLine, cChar)
			require.NotNil(t, hover, "expected hover on %s", tc.name)
			require.Equal(t, lsp.Markdown, hover.Contents.Kind)
			require.Equal(t, tc.expected, hover.Contents.Value, "hover on %s", tc.name)
		}
	})

	t.Run("cursor on keyword returns description", func(t *testing.T) {
		line, char := posIn(t, testDoc, "command SubmitOrder", "command")
		assertHover(t, testDoc, line, char, "Defines a command that can be sent to an aggregate.")
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
		// Cursor at (0, 9) is on the string literal "Orders", not a keyword or definition.
		assertNil(t, testDoc, 0, 9)
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

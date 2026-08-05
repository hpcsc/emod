//go:build unit

package lsp_test

import (
	"strings"
	"testing"

	"github.com/hpcsc/emod/internal/lsp"
	"github.com/hpcsc/emod/internal/test"
	"github.com/stretchr/testify/require"
)

func TestGetReferences(t *testing.T) {
	const testDoc = `context "Orders" {
    aggregate "Sales" {
        slice "OrderSlice" {
            command SubmitOrder {
            }
            event OrderSubmitted {
            }
            view OrderView {
                subscribes [OrderSubmitted]
            }
            automation AutoSubmit {
                on OrderSubmitted
                command SubmitOrder
                target context Orders
            }
            translation TransOrder {
                external_system "ERP"
                reads OrderView
                command SubmitOrder
            }
            trigger "MyTrigger" {
                actor user
                reads OrderView
            }
            flow {
                command -> event : SubmitOrder -> OrderSubmitted
            }
        }
    }
}
`
	uri := "file:///test.emod"

	// posIn returns 0-based (line, character) of substr within the first
	// occurrence of container in doc.
	posIn := func(t *testing.T, doc, container, substr string) (int, int) {
		t.Helper()
		cIdx := strings.Index(doc, container)
		require.GreaterOrEqual(t, cIdx, 0, "container %q not found", container)
		rel := strings.Index(doc[cIdx:], substr)
		require.GreaterOrEqual(t, rel, 0, "substr %q not found in container %q", substr, container)
		abs := cIdx + rel
		line := 0
		col := 0
		for i, ch := range doc {
			if i == abs {
				break
			}
			if ch == '\n' {
				line++
				col = 0
			} else {
				col++
			}
		}
		return line, col
	}

	// requireLocation checks that locs contains a Location matching the expected
	// line, character, and name. The range end is char+len(name).
	requireLocation := func(t *testing.T, locs []lsp.Location, expectedLine, expectedChar int, expectedName string) {
		t.Helper()
		found := false
		for _, loc := range locs {
			if loc.URI == uri &&
				loc.Range.Start.Line == expectedLine &&
				loc.Range.Start.Character == expectedChar &&
				loc.Range.End.Line == expectedLine &&
				loc.Range.End.Character == expectedChar+len(expectedName) {
				found = true
				break
			}
		}
		require.True(t, found, "expected location at line=%d char=%d name=%q not found in %d locations",
			expectedLine, expectedChar, expectedName, len(locs))
	}

	// locationOf builds the Location a reference at the given name inside
	// container should be reported as.
	locationOf := func(t *testing.T, doc, container, name string) lsp.Location {
		t.Helper()
		line, char := posIn(t, doc, container, name)
		return lsp.Location{
			URI: uri,
			Range: lsp.Range{
				Start: lsp.Position{Line: line, Character: char},
				End:   lsp.Position{Line: line, Character: char + len(name)},
			},
		}
	}

	t.Run("events", func(t *testing.T) {
		t.Run("cursor on event definition name returns all event references", func(t *testing.T) {
			cLine, cChar := posIn(t, testDoc, "event OrderSubmitted", "OrderSubmitted")
			locs := lsp.GetReferences(testDoc, cLine, cChar, uri)
			require.Len(t, locs, 4)

			dLine, dChar := posIn(t, testDoc, "event OrderSubmitted", "OrderSubmitted")
			requireLocation(t, locs, dLine, dChar, "OrderSubmitted")

			sLine, sChar := posIn(t, testDoc, "subscribes [OrderSubmitted]", "OrderSubmitted")
			requireLocation(t, locs, sLine, sChar, "OrderSubmitted")

			tLine, tChar := posIn(t, testDoc,
				"automation AutoSubmit {\n                on OrderSubmitted", "OrderSubmitted")
			requireLocation(t, locs, tLine, tChar, "OrderSubmitted")

			flowLine := "command -> event : SubmitOrder -> OrderSubmitted"
			fLine, fChar := posIn(t, testDoc, flowLine, "OrderSubmitted")
			requireLocation(t, locs, fLine, fChar, "OrderSubmitted")
		})

		t.Run("cursor on an event declared on a mode dcb context returns every site naming it", func(t *testing.T) {
			doc := test.AutomationReadsLibraryLending

			cLine, cChar := posIn(t, doc, "event DeskReleased", "DeskReleased")
			locs := lsp.GetReferences(doc, cLine, cChar, uri)

			require.Equal(t, []lsp.Location{
				locationOf(t, doc, "event DeskReleased", "DeskReleased"),
				locationOf(t, doc, "command -> event: ReleaseDesk -> DeskReleased", "DeskReleased"),
				locationOf(t, doc, "subscribes [DeskClaimed, DeskReleased]", "DeskReleased"),
				locationOf(t, doc, "automation RemindReaderOfLoans", "DeskReleased"),
			}, locs)
		})

		t.Run("cursor on event reference in subscribes returns all event references", func(t *testing.T) {
			cLine, cChar := posIn(t, testDoc, "subscribes [OrderSubmitted]", "OrderSubmitted")
			locs := lsp.GetReferences(testDoc, cLine, cChar, uri)
			require.Len(t, locs, 4)
		})

		t.Run("cursor on event reference in an automation activation event returns all event references", func(t *testing.T) {
			cLine, cChar := posIn(t, testDoc,
				"automation AutoSubmit {\n                on OrderSubmitted", "OrderSubmitted")
			locs := lsp.GetReferences(testDoc, cLine, cChar, uri)
			require.Len(t, locs, 4)
		})

		t.Run("cursor on event reference in flow event entry returns all event references", func(t *testing.T) {
			flowLine := "command -> event : SubmitOrder -> OrderSubmitted"
			cLine, cChar := posIn(t, testDoc, flowLine, "OrderSubmitted")
			locs := lsp.GetReferences(testDoc, cLine, cChar, uri)
			require.Len(t, locs, 4)
		})
	})

	t.Run("commands", func(t *testing.T) {
		t.Run("cursor on command definition name returns all command references", func(t *testing.T) {
			cLine, cChar := posIn(t, testDoc, "command SubmitOrder", "SubmitOrder")
			locs := lsp.GetReferences(testDoc, cLine, cChar, uri)
			require.Len(t, locs, 4)

			dLine, dChar := posIn(t, testDoc, "command SubmitOrder", "SubmitOrder")
			requireLocation(t, locs, dLine, dChar, "SubmitOrder")

			aLine, aChar := posIn(t, testDoc,
				"on OrderSubmitted\n                command SubmitOrder", "SubmitOrder")
			requireLocation(t, locs, aLine, aChar, "SubmitOrder")

			trLine, trChar := posIn(t, testDoc,
				"external_system \"ERP\"\n                reads OrderView\n                command SubmitOrder", "SubmitOrder")
			requireLocation(t, locs, trLine, trChar, "SubmitOrder")

			flowLine := "command -> event : SubmitOrder -> OrderSubmitted"
			colonIdx := strings.Index(flowLine, ": ")
			require.GreaterOrEqual(t, colonIdx, 0)
			fLine, fChar := posIn(t, testDoc, flowLine, flowLine[colonIdx:])
			fChar += 2 // skip ": " to point to 'S' of SubmitOrder
			requireLocation(t, locs, fLine, fChar, "SubmitOrder")
		})

		t.Run("cursor on a command an aggregate declares and a mode dcb context names returns both sites", func(t *testing.T) {
			doc := test.AutomationReadsLibraryLending

			cLine, cChar := posIn(t, doc, "command RemindMember {", "RemindMember")
			locs := lsp.GetReferences(doc, cLine, cChar, uri)

			require.Equal(t, []lsp.Location{
				locationOf(t, doc, "command RemindMember {", "RemindMember"),
				locationOf(t, doc, "automation RemindOnDueDate", "RemindMember"),
				locationOf(t, doc, "command -> event: RemindMember -> MemberReminded", "RemindMember"),
				locationOf(t, doc, "automation RemindReaderOfLoans", "RemindMember"),
			}, locs)
		})

		t.Run("sites in a context's own slices follow the sites in its aggregates", func(t *testing.T) {
			const doc = `context "Reading Room" mode dcb {
    aggregate "Desk" {
        slice "Claim Desk" {
            command ClaimDesk {
            }
            automation ClaimOnArrival {
                command ClaimDesk
            }
        }
    }
    slice "Close Reading Room" {
        automation ClaimAtOpening {
            command ClaimDesk
        }
    }
}
`

			cLine, cChar := posIn(t, doc, "command ClaimDesk {", "ClaimDesk")
			locs := lsp.GetReferences(doc, cLine, cChar, uri)

			require.Equal(t, []lsp.Location{
				locationOf(t, doc, "command ClaimDesk {", "ClaimDesk"),
				locationOf(t, doc, "automation ClaimOnArrival", "ClaimDesk"),
				locationOf(t, doc, "automation ClaimAtOpening", "ClaimDesk"),
			}, locs)
		})

		t.Run("cursor on command in automation returns all command references", func(t *testing.T) {
			cLine, cChar := posIn(t, testDoc,
				"on OrderSubmitted\n                command SubmitOrder", "SubmitOrder")
			locs := lsp.GetReferences(testDoc, cLine, cChar, uri)
			require.Len(t, locs, 4)
		})

		t.Run("cursor on command in translation returns all command references", func(t *testing.T) {
			cLine, cChar := posIn(t, testDoc,
				"external_system \"ERP\"\n                reads OrderView\n                command SubmitOrder", "SubmitOrder")
			locs := lsp.GetReferences(testDoc, cLine, cChar, uri)
			require.Len(t, locs, 4)
		})

		t.Run("cursor on command in flow returns all command references", func(t *testing.T) {
			flowLine := "command -> event : SubmitOrder -> OrderSubmitted"
			colonIdx := strings.Index(flowLine, ": ")
			require.GreaterOrEqual(t, colonIdx, 0)
			cLine, cChar := posIn(t, testDoc, flowLine, flowLine[colonIdx:])
			cChar += 2 // skip ": " to point to 'S' of SubmitOrder
			locs := lsp.GetReferences(testDoc, cLine, cChar, uri)
			require.Len(t, locs, 4)
		})
	})

	t.Run("views", func(t *testing.T) {
		t.Run("cursor on view definition name returns all view references", func(t *testing.T) {
			cLine, cChar := posIn(t, testDoc, "view OrderView", "OrderView")
			locs := lsp.GetReferences(testDoc, cLine, cChar, uri)
			require.Len(t, locs, 3)

			dLine, dChar := posIn(t, testDoc, "view OrderView", "OrderView")
			requireLocation(t, locs, dLine, dChar, "OrderView")

			trLine, trChar := posIn(t, testDoc, "reads OrderView", "OrderView")
			requireLocation(t, locs, trLine, trChar, "OrderView")

			triggerReadsLine, triggerReadsChar := posIn(t, testDoc,
				"trigger \"MyTrigger\" {\n                actor user\n                reads OrderView", "OrderView")
			requireLocation(t, locs, triggerReadsLine, triggerReadsChar, "OrderView")
		})

		t.Run("cursor on view in translation reads returns all view references", func(t *testing.T) {
			cLine, cChar := posIn(t, testDoc, "reads OrderView", "OrderView")
			locs := lsp.GetReferences(testDoc, cLine, cChar, uri)
			require.Len(t, locs, 3)
		})

		t.Run("cursor on view in trigger reads returns all view references", func(t *testing.T) {
			cLine, cChar := posIn(t, testDoc,
				"trigger \"MyTrigger\" {\n                actor user\n                reads OrderView", "OrderView")
			locs := lsp.GetReferences(testDoc, cLine, cChar, uri)
			require.Len(t, locs, 3)
		})
	})

	t.Run("nil returns", func(t *testing.T) {
		t.Run("cursor not on a resolvable name returns nil", func(t *testing.T) {
			t.Run("on a keyword", func(t *testing.T) {
				line, char := posIn(t, testDoc, "command SubmitOrder", "command")
				locs := lsp.GetReferences(testDoc, line, char, uri)
				require.Nil(t, locs)
			})

			t.Run("on a block keyword", func(t *testing.T) {
				line, char := posIn(t, testDoc, "view OrderView", "view")
				locs := lsp.GetReferences(testDoc, line, char, uri)
				require.Nil(t, locs)
			})
		})

		t.Run("cursor on reference to undefined name returns nil", func(t *testing.T) {
			t.Run("missing event in subscribes", func(t *testing.T) {
				doc := `context "Ctx" {
    aggregate "Agg" {
        slice "Slc" {
            view V {
                subscribes [NonExistentEvent]
            }
        }
    }
}`
				line, char := posIn(t, doc, "NonExistentEvent", "NonExistentEvent")
				locs := lsp.GetReferences(doc, line, char, uri)
				require.Nil(t, locs)
			})

			t.Run("missing command in automation", func(t *testing.T) {
				doc := `context "Ctx" {
    aggregate "Agg" {
        slice "Slc" {
            automation A {
                command NonExistentCmd
            }
        }
    }
}`
				line, char := posIn(t, doc, "NonExistentCmd", "NonExistentCmd")
				locs := lsp.GetReferences(doc, line, char, uri)
				require.Nil(t, locs)
			})
		})

		t.Run("empty document returns nil", func(t *testing.T) {
			locs := lsp.GetReferences("", 0, 0, uri)
			require.Nil(t, locs)
		})
	})
}

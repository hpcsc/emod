//go:build unit

package lsp_test

import (
	"strings"
	"testing"

	"github.com/hpcsc/emod/internal/lsp"
	"github.com/hpcsc/emod/internal/test"
	"github.com/stretchr/testify/require"
)

func TestGetDefinition(t *testing.T) {
	// testDoc is a valid emod document with definitions and references.
	// Context/aggregate/slice names are quoted strings; command/event/view
	// and other names are identifiers. Indentation uses 4 spaces per level.
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
                reads OrderView
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

	// assertDef checks that GetDefinition returns a Location pointing
	// to the definition name at the given expected position.
	assertDef := func(t *testing.T, doc string, cLine, cChar, dLine, dChar int, dName string) {
		t.Helper()
		loc := lsp.GetDefinition(doc, cLine, cChar, uri)
		require.NotNil(t, loc, "expected definition for cursor at (%d,%d)", cLine, cChar)
		require.Equal(t, uri, loc.URI)
		require.Equal(t, dLine, loc.Range.Start.Line, "start line mismatch")
		require.Equal(t, dChar, loc.Range.Start.Character, "start char mismatch")
		require.Equal(t, dLine, loc.Range.End.Line, "end line mismatch")
		require.Equal(t, dChar+len(dName), loc.Range.End.Character, "end char mismatch")
	}

	// assertNil checks that GetDefinition returns nil.
	assertNil := func(t *testing.T, doc string, cLine, cChar int) {
		t.Helper()
		loc := lsp.GetDefinition(doc, cLine, cChar, uri)
		require.Nil(t, loc, "expected nil for cursor at (%d,%d)", cLine, cChar)
	}

	t.Run("event reference in subscribes", func(t *testing.T) {
		cLine, cChar := posIn(t, testDoc, "subscribes [OrderSubmitted]", "OrderSubmitted")
		dLine, dChar := posIn(t, testDoc, "event OrderSubmitted", "OrderSubmitted")
		assertDef(t, testDoc, cLine, cChar, dLine, dChar, "OrderSubmitted")
	})

	t.Run("every name an automation block names resolves to its declaration", func(t *testing.T) {
		t.Run("event reference in automation activation event", func(t *testing.T) {
			cLine, cChar := posIn(t, testDoc, "automation AutoSubmit", "OrderSubmitted")
			dLine, dChar := posIn(t, testDoc, "event OrderSubmitted", "OrderSubmitted")
			assertDef(t, testDoc, cLine, cChar, dLine, dChar, "OrderSubmitted")
		})

		t.Run("view reference in automation reads", func(t *testing.T) {
			cLine, cChar := posIn(t, testDoc, "automation AutoSubmit", "OrderView")
			dLine, dChar := posIn(t, testDoc, "view OrderView", "OrderView")
			assertDef(t, testDoc, cLine, cChar, dLine, dChar, "OrderView")
		})

		t.Run("command reference in automation command", func(t *testing.T) {
			cLine, cChar := posIn(t, testDoc, "automation AutoSubmit", "SubmitOrder")
			dLine, dChar := posIn(t, testDoc, "command SubmitOrder", "SubmitOrder")
			assertDef(t, testDoc, cLine, cChar, dLine, dChar, "SubmitOrder")
		})
	})

	t.Run("context reference in automation target context", func(t *testing.T) {
		// target context Orders — "Orders" is an identifier here
		cLine, cChar := posIn(t, testDoc, "target context Orders", "Orders")
		// context "Orders" — definition NamePos points to the opening quote
		// The parser stores the string token position at the opening quote.
		// locationFor converts: Start.Character = NamePos.Column - 1.
		// posIn for "\"" finds the quote at its 0-based column (e.g. col 8),
		// which matches locationFor's result (since NamePos is 1-based col 9,
		// and 9 - 1 = 8).
		dLine, dChar := posIn(t, testDoc, "context \"Orders\"", "\"")
		assertDef(t, testDoc, cLine, cChar, dLine, dChar, "Orders")
	})

	t.Run("view reference in translation reads", func(t *testing.T) {
		cLine, cChar := posIn(t, testDoc, "translation TransOrder", "OrderView")
		dLine, dChar := posIn(t, testDoc, "view OrderView", "OrderView")
		assertDef(t, testDoc, cLine, cChar, dLine, dChar, "OrderView")
	})

	t.Run("command reference in translation command", func(t *testing.T) {
		cLine, cChar := posIn(t, testDoc, "translation TransOrder", "SubmitOrder")
		dLine, dChar := posIn(t, testDoc, "command SubmitOrder", "SubmitOrder")
		assertDef(t, testDoc, cLine, cChar, dLine, dChar, "SubmitOrder")
	})

	t.Run("view reference in trigger reads", func(t *testing.T) {
		cLine, cChar := posIn(t, testDoc, "trigger \"MyTrigger\"", "OrderView")
		dLine, dChar := posIn(t, testDoc, "view OrderView", "OrderView")
		assertDef(t, testDoc, cLine, cChar, dLine, dChar, "OrderView")
	})

	t.Run("command reference in flow command", func(t *testing.T) {
		flowLine := "command -> event : SubmitOrder -> OrderSubmitted"
		colonIdx := strings.Index(flowLine, ": ")
		require.GreaterOrEqual(t, colonIdx, 0)
		cLine, cChar := posIn(t, testDoc, flowLine, flowLine[colonIdx:])
		cChar += 2 // skip ": " to point to 'S' of SubmitOrder
		dLine, dChar := posIn(t, testDoc, "command SubmitOrder", "SubmitOrder")
		assertDef(t, testDoc, cLine, cChar, dLine, dChar, "SubmitOrder")
	})

	t.Run("event reference in flow event", func(t *testing.T) {
		flowLine := "command -> event : SubmitOrder -> OrderSubmitted"
		firstArrow := strings.Index(flowLine, "->")
		require.GreaterOrEqual(t, firstArrow, 0)
		rest := flowLine[firstArrow+2:]
		secondArrow := strings.Index(rest, "->")
		require.GreaterOrEqual(t, secondArrow, 0)
		eventPart := rest[secondArrow:] // "-> OrderSubmitted"
		cLine, cChar := posIn(t, testDoc, flowLine, eventPart)
		cChar += 3 // skip "-> " to point to 'O' of OrderSubmitted
		dLine, dChar := posIn(t, testDoc, "event OrderSubmitted", "OrderSubmitted")
		assertDef(t, testDoc, cLine, cChar, dLine, dChar, "OrderSubmitted")
	})

	t.Run("reference in a mode dcb context's own slice resolves to a sibling slice's declaration", func(t *testing.T) {
		doc := test.AutomationReadsLibraryLending

		t.Run("subscribes entry naming an event", func(t *testing.T) {
			cLine, cChar := posIn(t, doc, "subscribes [DeskClaimed, DeskReleased]", "DeskClaimed")
			dLine, dChar := posIn(t, doc, "event DeskClaimed", "DeskClaimed")
			assertDef(t, doc, cLine, cChar, dLine, dChar, "DeskClaimed")
		})

		t.Run("automation naming a command", func(t *testing.T) {
			cLine, cChar := posIn(t, doc, "automation FreeDeskAtClosing", "ReleaseDesk")
			dLine, dChar := posIn(t, doc, "command ReleaseDesk {", "ReleaseDesk")
			assertDef(t, doc, cLine, cChar, dLine, dChar, "ReleaseDesk")
		})
	})

	t.Run("an automation's reads resolves wherever the model declares the view", func(t *testing.T) {
		doc := test.AutomationReadsLibraryLending

		for _, tc := range []struct {
			name       string
			automation string
			view       string
		}{
			{
				name:       "view declared by a sibling slice of the same aggregate",
				automation: "automation RecallOverdueCopy",
				view:       "MemberLoansView",
			},
			{
				name:       "view declared by another context, read from a slice a mode dcb context declares directly",
				automation: "automation RemindReaderOfLoans",
				view:       "MemberLoansView",
			},
			{
				name:       "view declared by the same mode dcb context",
				automation: "automation FreeDeskAtClosing",
				view:       "DeskOccupancyView",
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				cLine, cChar := posIn(t, doc, tc.automation, tc.view)
				dLine, dChar := posIn(t, doc, "view "+tc.view, tc.view)
				assertDef(t, doc, cLine, cChar, dLine, dChar, tc.view)
			})
		}
	})

	t.Run("automation reads naming no declared view returns nil", func(t *testing.T) {
		doc := `context "Ctx" {
    aggregate "Agg" {
        slice "Slc" {
            view KnownView {
            }
            automation ReadUnresolved {
                reads NonExistentView
            }
            automation ReadResolved {
                reads KnownView
            }
        }
    }
}`

		t.Run("undeclared view name", func(t *testing.T) {
			line, char := posIn(t, doc, "automation ReadUnresolved", "NonExistentView")
			assertNil(t, doc, line, char)
		})

		t.Run("declared view name in the sibling automation", func(t *testing.T) {
			cLine, cChar := posIn(t, doc, "automation ReadResolved", "KnownView")
			dLine, dChar := posIn(t, doc, "view KnownView", "KnownView")
			assertDef(t, doc, cLine, cChar, dLine, dChar, "KnownView")
		})

		t.Run("cursor on the reads keyword rather than the view name", func(t *testing.T) {
			line, char := posIn(t, doc, "automation ReadResolved", "reads")
			assertNil(t, doc, line, char)
		})
	})

	t.Run("cursor not on a known reference returns nil", func(t *testing.T) {
		t.Run("on a keyword", func(t *testing.T) {
			line, char := posIn(t, testDoc, "command SubmitOrder", "command")
			assertNil(t, testDoc, line, char)
		})

		t.Run("on a block keyword", func(t *testing.T) {
			line, char := posIn(t, testDoc, "view OrderView", "view")
			assertNil(t, testDoc, line, char)
		})
	})

	t.Run("referenced name with no definition returns nil", func(t *testing.T) {
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
			assertNil(t, doc, line, char)
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
			assertNil(t, doc, line, char)
		})

		t.Run("missing context in automation target", func(t *testing.T) {
			doc := `context "Ctx" {
    aggregate "Agg" {
        slice "Slc" {
            automation A {
                target context NonExistentCtx
            }
        }
    }
}`
			line, char := posIn(t, doc, "NonExistentCtx", "NonExistentCtx")
			assertNil(t, doc, line, char)
		})
	})

	t.Run("empty document returns nil", func(t *testing.T) {
		assertNil(t, "", 0, 0)
	})

	t.Run("cursor on definition name itself returns nil", func(t *testing.T) {
		doc := `context "Ctx" {
    aggregate "Agg" {
        slice "Slc" {
            command Cmd {
            }
        }
    }
}`
		line, char := posIn(t, doc, "command Cmd", "Cmd")
		assertNil(t, doc, line, char)
	})
}

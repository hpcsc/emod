//go:build unit

package lsp_test

import (
	"strings"
	"testing"

	"github.com/hpcsc/emod/internal/lsp"
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
	// occurrence of container in doc. container must be found in doc,
	// and substr must be found within container.
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

	t.Run("event reference in automation activation event", func(t *testing.T) {
		cLine, cChar := posIn(t, testDoc,
			"automation AutoSubmit {\n                on OrderSubmitted",
			"OrderSubmitted")
		dLine, dChar := posIn(t, testDoc, "event OrderSubmitted", "OrderSubmitted")
		assertDef(t, testDoc, cLine, cChar, dLine, dChar, "OrderSubmitted")
	})

	t.Run("command reference in automation command", func(t *testing.T) {
		cLine, cChar := posIn(t, testDoc,
			"on OrderSubmitted\n                command SubmitOrder",
			"SubmitOrder")
		dLine, dChar := posIn(t, testDoc, "command SubmitOrder", "SubmitOrder")
		assertDef(t, testDoc, cLine, cChar, dLine, dChar, "SubmitOrder")
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
		cLine, cChar := posIn(t, testDoc, "reads OrderView", "OrderView")
		dLine, dChar := posIn(t, testDoc, "view OrderView", "OrderView")
		assertDef(t, testDoc, cLine, cChar, dLine, dChar, "OrderView")
	})

	t.Run("command reference in translation command", func(t *testing.T) {
		cLine, cChar := posIn(t, testDoc,
			"external_system \"ERP\"\n                reads OrderView\n                command SubmitOrder",
			"SubmitOrder")
		dLine, dChar := posIn(t, testDoc, "command SubmitOrder", "SubmitOrder")
		assertDef(t, testDoc, cLine, cChar, dLine, dChar, "SubmitOrder")
	})

	t.Run("view reference in trigger reads", func(t *testing.T) {
		cLine, cChar := posIn(t, testDoc,
			"trigger \"MyTrigger\" {\n                actor user\n                reads OrderView",
			"OrderView")
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

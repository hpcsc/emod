//go:build unit

package lsp_test

import (
	"strings"
	"testing"

	"github.com/hpcsc/emod/internal/lsp"
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

	assertHover := func(t *testing.T, doc string, cLine, cChar int, expectedContent string) {
		t.Helper()
		hover := lsp.GetHover(doc, cLine, cChar)
		require.NotNil(t, hover, "expected hover for cursor at (%d,%d)", cLine, cChar)
		require.Equal(t, lsp.Markdown, hover.Contents.Kind)
		require.Equal(t, expectedContent, hover.Contents.Value)
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

	t.Run("cursor on keyword returns nil", func(t *testing.T) {
		line, char := posIn(t, testDoc, "command SubmitOrder", "command")
		assertNil(t, testDoc, line, char)
	})

	t.Run("cursor not on any definition returns nil", func(t *testing.T) {
		assertNil(t, testDoc, 0, 0)
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

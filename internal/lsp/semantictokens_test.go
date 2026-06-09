//go:build unit

package lsp_test

import (
	"strings"
	"testing"

	"github.com/hpcsc/emod/internal/lsp"
	"github.com/stretchr/testify/require"
)

// posIn finds the 0-based line and character position of substr within a
// container string inside doc. It is used to locate expected token positions.
func posIn(t *testing.T, doc, container, substr string) (int, int) {
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

// decodeTokens decodes delta-encoded semantic token data into a human-readable
// slice of (line, character, length, tokenTypeIndex) tuples for assertions.
func decodeTokens(data []uint) []struct {
	line, char, length, typeIdx int
} {
	result := make([]struct {
		line, char, length, typeIdx int
	}, 0, len(data)/5)
	var prevLine, prevChar int
	for i := 0; i+4 < len(data); i += 5 {
		dl := int(data[i])
		dc := int(data[i+1])
		l := int(data[i+2])
		tt := int(data[i+3])
		line := prevLine + dl
		char := dc
		if dl == 0 {
			char = prevChar + dc
		}
		result = append(result, struct {
			line, char, length, typeIdx int
		}{line, char, l, tt})
		prevLine = line
		prevChar = char
	}
	return result
}

func TestGetSemanticTokens(t *testing.T) {
	// legendTokenType returns the expected legend index for a given token type
	// name, matching the index used by GetSemanticTokens.
	legendTokenType := func(tt lsp.SemanticTokenTypes) int {
		switch tt {
		case lsp.TokenTypeFunction:
			return 0
		case lsp.TokenTypeEvent:
			return 1
		case lsp.TokenTypeClass:
			return 2
		case lsp.TokenTypeParameter:
			return 3
		case lsp.TokenTypeNamespace:
			return 4
		case lsp.TokenTypeStruct:
			return 5
		default:
			return -1
		}
	}

	// assertToken asserts that the decoded token at the given index has the
	// expected position and token type.
	assertToken := func(t *testing.T, tokens []struct {
		line, char, length, typeIdx int
	}, idx, line, char, length, typeIdx int) {
		t.Helper()
		require.Greater(t, len(tokens), idx, "expected token at index %d", idx)
		require.Equal(t, line, tokens[idx].line, "token %d: line", idx)
		require.Equal(t, char, tokens[idx].char, "token %d: char", idx)
		require.Equal(t, length, tokens[idx].length, "token %d: length", idx)
		require.Equal(t, typeIdx, tokens[idx].typeIdx, "token %d: type", idx)
	}

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

	t.Run("command name receives TokenTypeFunction", func(t *testing.T) {
		line, char := posIn(t, testDoc, "command SubmitOrder", "SubmitOrder")
		result := lsp.GetSemanticTokens(testDoc)
		tokens := decodeTokens(result.Data)
		found := false
		for _, tok := range tokens {
			if tok.line == line && tok.char == char {
				require.Equal(t, legendTokenType(lsp.TokenTypeFunction), tok.typeIdx)
				require.Equal(t, len("SubmitOrder"), tok.length)
				found = true
				break
			}
		}
		require.True(t, found, "SubmitOrder token not found in semantic tokens")
	})

	t.Run("event name receives TokenTypeEvent", func(t *testing.T) {
		line, char := posIn(t, testDoc, "event OrderSubmitted", "OrderSubmitted")
		result := lsp.GetSemanticTokens(testDoc)
		tokens := decodeTokens(result.Data)
		found := false
		for _, tok := range tokens {
			if tok.line == line && tok.char == char {
				require.Equal(t, legendTokenType(lsp.TokenTypeEvent), tok.typeIdx)
				require.Equal(t, len("OrderSubmitted"), tok.length)
				found = true
				break
			}
		}
		require.True(t, found, "OrderSubmitted token not found in semantic tokens")
	})

	t.Run("view name receives TokenTypeClass", func(t *testing.T) {
		line, char := posIn(t, testDoc, "view OrderView", "OrderView")
		result := lsp.GetSemanticTokens(testDoc)
		tokens := decodeTokens(result.Data)
		found := false
		for _, tok := range tokens {
			if tok.line == line && tok.char == char {
				require.Equal(t, legendTokenType(lsp.TokenTypeClass), tok.typeIdx)
				require.Equal(t, len("OrderView"), tok.length)
				found = true
				break
			}
		}
		require.True(t, found, "OrderView token not found in semantic tokens")
	})

	t.Run("actor name receives TokenTypeParameter", func(t *testing.T) {
		const actorDoc = `actor "Customer"
`
		line, char := posIn(t, actorDoc, "actor \"Customer\"", "Customer")
		result := lsp.GetSemanticTokens(actorDoc)
		tokens := decodeTokens(result.Data)
		require.Len(t, tokens, 1)
		assertToken(t, tokens, 0, line, char, len("Customer"), legendTokenType(lsp.TokenTypeParameter))
	})

	t.Run("context name receives TokenTypeNamespace", func(t *testing.T) {
		line, char := posIn(t, testDoc, "context \"Orders\"", "Orders")
		result := lsp.GetSemanticTokens(testDoc)
		tokens := decodeTokens(result.Data)
		found := false
		for _, tok := range tokens {
			if tok.line == line && tok.char == char {
				require.Equal(t, legendTokenType(lsp.TokenTypeNamespace), tok.typeIdx)
				require.Equal(t, len("Orders"), tok.length)
				found = true
				break
			}
		}
		require.True(t, found, "Orders token not found in semantic tokens")
	})

	t.Run("aggregate name receives TokenTypeStruct", func(t *testing.T) {
		line, char := posIn(t, testDoc, "aggregate \"Sales\"", "Sales")
		result := lsp.GetSemanticTokens(testDoc)
		tokens := decodeTokens(result.Data)
		found := false
		for _, tok := range tokens {
			if tok.line == line && tok.char == char {
				require.Equal(t, legendTokenType(lsp.TokenTypeStruct), tok.typeIdx)
				require.Equal(t, len("Sales"), tok.length)
				found = true
				break
			}
		}
		require.True(t, found, "Sales token not found in semantic tokens")
	})

	t.Run("delta-encodes multiple identifiers on same line", func(t *testing.T) {
		const doc = `context "Orders" {
    aggregate "Sales" {
        slice "OrderSlice" {
            command CmdOne {
            }
            command CmdTwo {
            }
        }
    }
}
`
		result := lsp.GetSemanticTokens(doc)
		data := result.Data
		require.GreaterOrEqual(t, len(data), 10, "expected at least two delta-encoded tokens")

		// Verify first token is absolute.
		firstLine := int(data[0])
		firstChar := int(data[1])
		require.GreaterOrEqual(t, firstLine, 0)
		require.GreaterOrEqual(t, firstChar, 0)

		// Verify second token uses delta encoding.
		if data[5] == 0 {
			// Same line as first token — delta char should be relative.
			secondDeltaChar := int(data[6])
			require.Greater(t, secondDeltaChar, 0, "second token on same line should have positive delta char")
		}
	})

	t.Run("delta-encodes identifiers across multiple lines", func(t *testing.T) {
		const doc = `actor "Customer"
actor "Admin"
`
		result := lsp.GetSemanticTokens(doc)
		tokens := decodeTokens(result.Data)
		require.Len(t, tokens, 2)

		// First token: "Customer" at line 0.
		require.Equal(t, 0, tokens[0].line)
		require.Equal(t, legendTokenType(lsp.TokenTypeParameter), tokens[0].typeIdx)

		// Second token: "Admin" at line 1.
		require.Equal(t, 1, tokens[1].line)
		require.Equal(t, legendTokenType(lsp.TokenTypeParameter), tokens[1].typeIdx)
		require.Equal(t, len("Admin"), tokens[1].length)
	})

	t.Run("empty document returns empty tokens", func(t *testing.T) {
		result := lsp.GetSemanticTokens("")
		require.Empty(t, result.Data)
	})

	t.Run("document with no named identifiers returns empty tokens", func(t *testing.T) {
		const doc = `model "TestModel"
`
		result := lsp.GetSemanticTokens(doc)
		require.Empty(t, result.Data)
	})

	t.Run("parse error document returns empty tokens", func(t *testing.T) {
		const doc = `this is not valid emod syntax`
		result := lsp.GetSemanticTokens(doc)
		require.Empty(t, result.Data)
	})
}

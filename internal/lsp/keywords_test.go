//go:build unit

package lsp_test

import (
	"strings"
	"testing"

	"github.com/hpcsc/emod/internal/lexer"
	"github.com/hpcsc/emod/internal/lsp"
	"github.com/stretchr/testify/require"
)

// The hover descriptions and the completer's per-block lists are both written by
// hand and neither is derived from internal/lexer, so a keyword added to the
// language reaches them only if someone remembers. These are the assertions that
// remember.
func TestKeywordCoverage(t *testing.T) {
	t.Run("hover", func(t *testing.T) {
		require.NotEmpty(t, lexer.Keywords(), "the lexer reports no keywords at all")

		for _, keyword := range lexer.Keywords() {
			t.Run(keyword+" describes itself on hover", func(t *testing.T) {
				hover := lsp.GetHover(keyword, 0, 0)

				require.NotNil(t, hover, "the lexer defines %q but no hover text describes it", keyword)
				require.NotEmpty(t, hover.Contents.Value)
			})
		}

		t.Run("a spelling the lexer does not define describes nothing", func(t *testing.T) {
			require.Nil(t, lsp.GetHover("decides_upon", 0, 0))
		})

		t.Run("a keyword inside a real document describes itself where it stands", func(t *testing.T) {
			doc := `context "Fulfillment" mode dcb {
  slice "Authorize" {
    command Authorize {
      decides_on {
        events [OrderPlaced]
        where tag(entity = customerId)
      }
    }
  }
}`
			hover := lsp.GetHover(doc, 3, 6)

			require.NotNil(t, hover)
			require.NotEmpty(t, hover.Contents.Value)
			// The range, not the wording, is what says the hover resolved this
			// token: it spans exactly the ten characters of `decides_on`.
			require.Equal(t, &lsp.Range{
				Start: lsp.Position{Line: 3, Character: 6},
				End:   lsp.Position{Line: 3, Character: 16},
			}, hover.Range)
		})
	})

	t.Run("completion", func(t *testing.T) {
		for _, block := range keywordBlocks {
			t.Run(block.name+" offers only spellings the lexer defines", func(t *testing.T) {
				labels := completionLabels(t, block.doc, block.line)

				require.NotEmpty(t, labels, "%s offers nothing, so this leaf asserts nothing", block.name)
				for _, label := range labels {
					// `target context` is two keywords offered as one entry.
					for _, word := range strings.Fields(label) {
						require.Contains(
							t, lexer.Keywords(), word,
							"%s offers %q, which the lexer does not define", block.name, label,
						)
					}
				}
			})
		}

		// The counterpart of the rule above: the DSL reserves nothing, so the
		// types and modifiers a field line accepts must not be keywords.
		t.Run("a fields block offers types and modifiers, none of them keywords", func(t *testing.T) {
			labels := completionLabels(t, fieldsDocument, 4)

			require.NotEmpty(t, labels)
			for _, label := range labels {
				require.NotContains(t, lexer.Keywords(), label)
			}
		})

		t.Run("a tags block offers nothing, its keys being free identifiers", func(t *testing.T) {
			require.Empty(t, completionLabels(t, tagsDocument, 4))
		})
	})
}

// Each document parks the cursor on a blank line inside one block, which is the
// only way an external test can reach a blockContext: the enum is unexported.
type keywordBlock struct {
	name string
	doc  string
	line int
}

var keywordBlocks = []keywordBlock{
	{name: "the top level", doc: "", line: 0},
	{name: "a context body", doc: "context \"C\" {\n\n}", line: 1},
	{name: "an aggregate body", doc: "context \"C\" {\n  aggregate \"A\" {\n\n  }\n}", line: 2},
	{name: "a slice body", doc: "context \"C\" {\n  slice \"S\" {\n\n  }\n}", line: 2},
	{name: "a command body", doc: "context \"C\" {\n  slice \"S\" {\n    command Decide {\n\n    }\n  }\n}", line: 3},
	{name: "an event body", doc: "context \"C\" {\n  slice \"S\" {\n    event Happened {\n\n    }\n  }\n}", line: 3},
	{name: "an automation body", doc: "context \"C\" {\n  slice \"S\" {\n    automation Notify {\n\n    }\n  }\n}", line: 3},
	{name: "a decides_on body", doc: decidesOnDocument, line: 4},
}

const decidesOnDocument = `context "C" {
  slice "S" {
    command Decide {
      decides_on {

      }
    }
  }
}`

const tagsDocument = `context "C" {
  slice "S" {
    event Happened {
      tags {

      }
    }
  }
}`

const fieldsDocument = `context "C" {
  slice "S" {
    command Decide {
      fields {

      }
    }
  }
}`

func completionLabels(t *testing.T, doc string, line int) []string {
	t.Helper()

	result := lsp.GetCompletions(doc, line, 0)

	labels := make([]string, 0, len(result.Items))
	for _, item := range result.Items {
		labels = append(labels, item.Label)
	}

	return labels
}

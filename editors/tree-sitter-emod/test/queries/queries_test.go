//go:build grammar

package queries

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	foldsQuery       = "queries/folds.scm"
	indentsQuery     = "queries/indents.scm"
	textobjectsQuery = "queries/textobjects.scm"
)

type sampleBlock struct {
	name   string
	header string
}

var sampleBlocks = []sampleBlock{
	{name: "the kindless trigger", header: `trigger "Review orders"`},
	{name: "the automation stating on", header: "automation NotifyCustomer"},
	{name: "the automation stating every", header: "automation ArchiveOrders"},
	{name: "the fields block", header: "fields {"},
}

func TestEditorQueries(t *testing.T) {
	t.Run("folding", func(t *testing.T) {
		for _, block := range sampleBlocks {
			t.Run(block.name+" folds from its keyword through its closing brace", func(t *testing.T) {
				extent := locateBlock(t, block)

				fold := soleCaptureAt(t, runQuery(t, foldsQuery), "fold", extent.Start)

				require.Equal(t, extent.End, fold.End)
			})
		}

		t.Run("every folded block ends on its own closing delimiter", func(t *testing.T) {
			for _, fold := range capturesNamed(runQuery(t, foldsQuery), "fold") {
				require.Contains(t, []string{"}", "]"}, sampleCharacterBefore(t, fold.End))
			}
		})
	})

	t.Run("text objects", func(t *testing.T) {
		for _, block := range sampleBlocks {
			t.Run(block.name+" selects whole from its keyword through its closing brace", func(t *testing.T) {
				extent := locateBlock(t, block)

				outer := soleCaptureAt(t, runQuery(t, textobjectsQuery), "block.outer", extent.Start)

				require.Equal(t, extent.End, outer.End)
			})

			t.Run(block.name+" selects inside strictly between its own braces", func(t *testing.T) {
				extent := locateBlock(t, block)

				inner := soleMatchAt(t, runQuery(t, textobjectsQuery), "block.inner", extent.Start)

				start := inner.capture(t, "_start")
				end := inner.capture(t, "_end")
				require.Equal(t, "{", start.Text)
				require.Equal(t, "}", end.Text)
				require.Equal(t, extent.OpenBrace, start.Start)
				require.Equal(t, extent.CloseBrace, end.Start)
				require.True(t, precedes(extent.Start, start.End), "interior must begin after the block's keyword")
				require.True(t, precedes(start.End, end.Start), "interior must not be empty")
				require.True(t, precedes(end.Start, extent.End), "interior must end before the block's closing brace")
			})
		}

		t.Run("whole-block selection covers the same range as folding", func(t *testing.T) {
			outer := spans(capturesNamed(runQuery(t, textobjectsQuery), "block.outer"))
			folds := spans(capturesNamed(runQuery(t, foldsQuery), "fold"))

			require.NotEmpty(t, outer)
			require.Subset(t, folds, outer)
		})
	})

	t.Run("indentation", func(t *testing.T) {
		for _, block := range sampleBlocks {
			t.Run(block.name+" indents on its opening brace and dedents on its closing brace", func(t *testing.T) {
				extent := locateBlock(t, block)

				indented := soleMatchAt(t, runQuery(t, indentsQuery), "indent", extent.OpenBrace)

				require.Equal(t, capture{
					Name:  "indent",
					Start: extent.OpenBrace,
					End:   columnAfter(extent.OpenBrace),
					Text:  "{",
				}, indented.capture(t, "indent"))
				require.Equal(t, capture{
					Name:  "dedent",
					Start: extent.CloseBrace,
					End:   columnAfter(extent.CloseBrace),
					Text:  "}",
				}, indented.capture(t, "dedent"))
			})
		}
	})

	t.Run("compilation against the grammar", func(t *testing.T) {
		t.Run("a query naming a node type the grammar does not define is rejected", func(t *testing.T) {
			original := readGrammarFile(t, foldsQuery)
			require.Contains(t, original, "(trigger_definition)")
			renamed := strings.Replace(original, "(trigger_definition)", "(trigger_declaration)", 1)
			mutated := filepath.Join(t.TempDir(), "folds.scm")
			require.NoError(t, os.WriteFile(mutated, []byte(renamed), 0o600))

			_, err := queryMatches(mutated, samplePath)

			require.Error(t, err)
			require.Contains(t, err.Error(), "folds.scm")
			require.Contains(t, err.Error(), "trigger_declaration")
		})
	})
}

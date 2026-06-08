//go:build unit

package lsp_test

import (
	"testing"

	"github.com/hpcsc/emod/internal/lsp"
	"github.com/stretchr/testify/require"
)

func TestGetCompletions(t *testing.T) {
	t.Run("top level", func(t *testing.T) {
		t.Run("empty document returns model actor and context", func(t *testing.T) {
			result := lsp.GetCompletions("", 0, 0)
			require.False(t, result.IsIncomplete)
			require.Equal(t, []string{"model", "actor", "context"}, extractLabels(result.Items))
		})

		t.Run("cursor at beginning of document returns top level keywords", func(t *testing.T) {
			result := lsp.GetCompletions("context Foo {}", 0, 0)
			require.Equal(t, []string{"model", "actor", "context"}, extractLabels(result.Items))
		})

		t.Run("cursor after an unclosed keyword line returns top level context", func(t *testing.T) {
			result := lsp.GetCompletions("context", 0, 7)
			require.Equal(t, []string{"model", "actor", "context"}, extractLabels(result.Items))
		})
	})

	t.Run("context block", func(t *testing.T) {
		t.Run("inside context block returns aggregate", func(t *testing.T) {
			doc := `context MyContext {
	// cursor here
}`
			result := lsp.GetCompletions(doc, 1, 2)
			require.Equal(t, []string{"aggregate"}, extractLabels(result.Items))
		})

		t.Run("context block with opening brace on next line still works", func(t *testing.T) {
			doc := "context MyContext\n{\n}"
			// cursor inside the block
			result := lsp.GetCompletions(doc, 1, 1)
			require.Equal(t, []string{"aggregate"}, extractLabels(result.Items))
		})
	})

	t.Run("aggregate block", func(t *testing.T) {
		t.Run("inside aggregate block returns slice", func(t *testing.T) {
			doc := `context Ctx {
	aggregate Agg {
		// cursor here
	}
}`
			result := lsp.GetCompletions(doc, 2, 3)
			require.Equal(t, []string{"slice"}, extractLabels(result.Items))
		})
	})

	t.Run("slice block", func(t *testing.T) {
		t.Run("inside slice block returns all slice keywords", func(t *testing.T) {
			doc := `context Ctx {
	aggregate Agg {
		slice Slc {
			// cursor here
		}
	}
}`
			result := lsp.GetCompletions(doc, 3, 4)
			require.Equal(t, []string{"command", "event", "trigger", "view", "automation", "translation", "flow"}, extractLabels(result.Items))
		})
	})

	t.Run("command block", func(t *testing.T) {
		t.Run("inside command block returns fields", func(t *testing.T) {
			doc := `context Ctx {
	aggregate Agg {
		slice Slc {
			command Cmd {
				// cursor here
			}
		}
	}
}`
			result := lsp.GetCompletions(doc, 4, 5)
			require.Equal(t, []string{"fields"}, extractLabels(result.Items))
		})
	})

	t.Run("event block", func(t *testing.T) {
		t.Run("inside event block returns fields", func(t *testing.T) {
			doc := `context Ctx {
	aggregate Agg {
		slice Slc {
			event Evt {
				// cursor here
			}
		}
	}
}`
			result := lsp.GetCompletions(doc, 4, 5)
			require.Equal(t, []string{"fields"}, extractLabels(result.Items))
		})
	})

	t.Run("fields block", func(t *testing.T) {
		t.Run("inside fields block returns field types and modifiers", func(t *testing.T) {
			doc := `context Ctx {
	aggregate Agg {
		slice Slc {
			command Cmd {
				fields {
					// cursor here
				}
			}
		}
	}
}`
			result := lsp.GetCompletions(doc, 5, 5)
			require.Equal(t, []string{"string", "date", "timestamp", "int", "required", "optional"}, extractLabels(result.Items))
		})

		t.Run("fields inside event block returns same completions", func(t *testing.T) {
			doc := `command Cmd {
	fields {
		// cursor here
	}
}`
			result := lsp.GetCompletions(doc, 2, 3)
			require.Equal(t, []string{"string", "date", "timestamp", "int", "required", "optional"}, extractLabels(result.Items))
		})
	})

	t.Run("completion items use keyword kind", func(t *testing.T) {
		t.Run("all items have Kind set to KeywordCompletion", func(t *testing.T) {
			result := lsp.GetCompletions("", 0, 0)
			for _, item := range result.Items {
				require.Equal(t, lsp.KeywordCompletion, item.Kind, "item %q should be KeywordCompletion", item.Label)
			}
		})
	})
}

func extractLabels(items []lsp.CompletionItem) []string {
	labels := make([]string, len(items))
	for i, item := range items {
		labels[i] = item.Label
	}
	return labels
}

//go:build unit

package lsp_test

import (
	"testing"

	"github.com/hpcsc/emod/internal/lexer"
	"github.com/hpcsc/emod/internal/lsp"
	"github.com/hpcsc/emod/internal/parser"
	"github.com/hpcsc/emod/internal/test"
	"github.com/stretchr/testify/require"
)

func TestGetCompletions(t *testing.T) {
	automationEntries := []string{"on", "every", "reads", "command", "target context"}

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
		t.Run("inside context block returns aggregate, slice and invariant", func(t *testing.T) {
			doc := `context MyContext {
	// cursor here
}`
			result := lsp.GetCompletions(doc, 1, 2)
			require.Equal(t, []string{"aggregate", "slice", "invariant"}, extractLabels(result.Items))
		})

		t.Run("context block with opening brace on next line still works", func(t *testing.T) {
			doc := "context MyContext\n{\n}"
			// cursor inside the block
			result := lsp.GetCompletions(doc, 1, 1)
			require.Equal(t, []string{"aggregate", "slice", "invariant"}, extractLabels(result.Items))
		})

		t.Run("opening brace separated from the keyword by lines carrying no code still opens the block", func(t *testing.T) {
			for _, separator := range []string{"", "  ", "  # a note"} {
				doc := "context MyContext\n" + separator + "\n{\n}"
				result := lsp.GetCompletions(doc, 2, 1)
				require.Equal(t, []string{"aggregate", "slice", "invariant"}, extractLabels(result.Items), "separator %q", separator)
			}
		})
	})

	t.Run("aggregate block", func(t *testing.T) {
		t.Run("inside aggregate block returns slice and invariant", func(t *testing.T) {
			doc := `context Ctx {
	aggregate Agg {
		// cursor here
	}
}`
			result := lsp.GetCompletions(doc, 2, 3)
			require.Equal(t, []string{"slice", "invariant"}, extractLabels(result.Items))
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

	t.Run("automation block", func(t *testing.T) {
		t.Run("inside automation block returns the entries an automation accepts", func(t *testing.T) {
			doc := `context Ctx {
	aggregate Agg {
		slice Slc {
			automation Auto {
				// cursor here
			}
		}
	}
}`
			result := lsp.GetCompletions(doc, 4, 5)
			require.Equal(t, automationEntries, extractLabels(result.Items))
			requireItemKinds(t, result.Items, lsp.KeywordCompletion)
		})

		t.Run("below a braceless command reference still returns automation entries", func(t *testing.T) {
			doc := `context Ctx {
	aggregate Agg {
		slice Slc {
			automation Auto {
				on OrderPlaced
				command ShipOrder
				// cursor here
			}
		}
	}
}`
			result := lsp.GetCompletions(doc, 6, 5)
			require.Equal(t, automationEntries, extractLabels(result.Items))
		})

		t.Run("after a closed automation block returns the enclosing slice keywords", func(t *testing.T) {
			doc := `context Ctx {
	aggregate Agg {
		slice Slc {
			automation Auto {
				on OrderPlaced
				command ShipOrder
			}
			// cursor here
		}
	}
}`
			result := lsp.GetCompletions(doc, 7, 4)
			require.Equal(t, []string{"command", "event", "trigger", "view", "automation", "translation", "flow"}, extractLabels(result.Items))
		})

		// Sibling slice blocks own no entry list yet, so they fall through to the top
		// level keywords rather than borrowing the automation entries.
		for _, keyword := range []string{"trigger", "view", "translation"} {
			t.Run("inside a "+keyword+" block beside an automation returns the top level keywords", func(t *testing.T) {
				doc := `context Ctx {
	aggregate Agg {
		slice Slc {
			automation Auto {
				on OrderPlaced
				command ShipOrder
			}
			` + keyword + ` Sibling {
				// cursor here
			}
		}
	}
}`
				result := lsp.GetCompletions(doc, 8, 5)
				require.Equal(t, []string{"model", "actor", "context"}, extractLabels(result.Items))
			})
		}
	})

	t.Run("command block", func(t *testing.T) {
		t.Run("inside command block returns fields and decides_on", func(t *testing.T) {
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
			require.Equal(t, []string{"fields", "decides_on"}, extractLabels(result.Items))
		})
	})

	t.Run("event block", func(t *testing.T) {
		t.Run("inside event block returns fields and tags", func(t *testing.T) {
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
			require.Equal(t, []string{"fields", "tags"}, extractLabels(result.Items))
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

	t.Run("value position", func(t *testing.T) {
		declaredEvents := []string{"CopyBorrowed", "MemberReminded", "CopyRecalled", "DeskClaimed", "DeskReleased"}
		declaredViews := []string{"MemberLoansView", "DeskOccupancyView"}

		t.Run("after on returns the declared events of both slice homes in declaration order", func(t *testing.T) {
			doc := test.AutomationReadsLibraryLending
			line, character := posIn(t, doc, "automation RemindOnDueDate", "CopyBorrowed")

			result := lsp.GetCompletions(doc, line, character)

			require.Equal(t, declaredEvents, extractLabels(result.Items))
			requireItemKinds(t, result.Items, lsp.EventCompletion)
		})

		t.Run("after reads returns the declared views of both slice homes", func(t *testing.T) {
			translationDoc := `context "Shelving" {
	aggregate "Catalog" {
		slice "Browse Catalog" {
			view CatalogView {
				subscribes [BookShelved]
			}
		}
		slice "Import Shelf" {
			translation ShelfImport {
				external_system "Partner API"
				reads CatalogView
				command ShelveBook
			}
		}
	}
}`
			for _, tc := range []struct {
				block     string
				doc       string
				container string
				value     string
				expected  []string
			}{
				{
					block:     "automation",
					doc:       test.AutomationReadsLibraryLending,
					container: "automation RecallOverdueCopy",
					value:     "MemberLoansView",
					expected:  declaredViews,
				},
				{
					block:     "trigger",
					doc:       test.AutomationReadsLibraryLending,
					container: `trigger "Lending Desk"`,
					value:     "MemberLoansView",
					expected:  declaredViews,
				},
				{
					block:     "translation",
					doc:       translationDoc,
					container: "translation ShelfImport",
					value:     "CatalogView",
					expected:  []string{"CatalogView"},
				},
			} {
				t.Run("inside a "+tc.block+" block", func(t *testing.T) {
					line, character := posIn(t, tc.doc, tc.container, tc.value)

					result := lsp.GetCompletions(tc.doc, line, character)

					require.Equal(t, tc.expected, extractLabels(result.Items))
					requireItemKinds(t, result.Items, lsp.ClassCompletion)
				})
			}
		})

		t.Run("an on entry naming nothing yet still returns the events of a document that does not parse cleanly", func(t *testing.T) {
			doc := `context "Lending" {
	aggregate "Loan" {
		slice "Chase Overdue Copy" {
			event CopyBorrowed {
			}
			event MemberReminded {
			}
			automation RemindOnDueDate {
				on` + " " + `
			}
		}
	}
}`
			tokens, scanErrs := lexer.Scan(doc, "half-written.emod")
			require.Empty(t, scanErrs)
			_, parseErrs := parser.New(tokens, "half-written.emod").Parse()
			require.NotEmpty(t, parseErrs, "the document under test is expected to carry parse diagnostics")

			result := lsp.GetCompletions(doc, 8, 7)

			require.Equal(t, []string{"CopyBorrowed", "MemberReminded"}, extractLabels(result.Items))
		})

		t.Run("with the cursor immediately after on the half-typed keyword returns the automation entries", func(t *testing.T) {
			doc := test.AutomationReadsLibraryLending
			line, nameStart := posIn(t, doc, "automation RemindOnDueDate", "CopyBorrowed")

			result := lsp.GetCompletions(doc, line, nameStart-1)

			require.Equal(t, automationEntries, extractLabels(result.Items))
		})

		t.Run("a name already begun stays a value position", func(t *testing.T) {
			doc := test.AutomationReadsLibraryLending
			line, nameStart := posIn(t, doc, "automation RemindOnDueDate", "CopyBorrowed")

			for _, tc := range []struct {
				cursor    string
				character int
			}{
				{cursor: "three letters into the name", character: nameStart + 3},
				{cursor: "at the end of the name", character: nameStart + len("CopyBorrowed")},
			} {
				t.Run("with the cursor "+tc.cursor, func(t *testing.T) {
					result := lsp.GetCompletions(doc, line, tc.character)

					require.Equal(t, declaredEvents, extractLabels(result.Items))
				})
			}
		})

		t.Run("a field line spelling reads returns the field types and modifiers", func(t *testing.T) {
			doc := `context "Lending" {
	aggregate "Loan" {
		slice "Borrow Copy" {
			command BorrowCopy {
				fields {
					id reads required
				}
			}
			view MemberLoansView {
			}
		}
	}
}`
			result := lsp.GetCompletions(doc, 5, 14)

			require.Equal(t, []string{"string", "date", "timestamp", "int", "required", "optional"}, extractLabels(result.Items))
		})

		t.Run("of an automation's entries only on and reads name something the model declares", func(t *testing.T) {
			doc := `context "Lending" {
	aggregate "Loan" {
		slice "Borrow Copy" {
			event CopyBorrowed {
			}
			automation RemindOnDueDate {
				on CopyBorrowed
				command RemindMember
				target context Notifications
			}
		}
	}
}`
			for _, tc := range []struct {
				entry     string
				line      int
				character int
				expected  []string
			}{
				{entry: "on", line: 6, character: 7, expected: []string{"CopyBorrowed"}},
				{entry: "command", line: 7, character: 12, expected: automationEntries},
				{entry: "target context", line: 8, character: 19, expected: automationEntries},
			} {
				t.Run("after "+tc.entry, func(t *testing.T) {
					result := lsp.GetCompletions(doc, tc.line, tc.character)

					require.Equal(t, tc.expected, extractLabels(result.Items))
				})
			}
		})

		t.Run("a model declaring nothing of the kind an entry names returns an empty list", func(t *testing.T) {
			doc := `context "Reading Room" {
	aggregate "Desk" {
		slice "Claim Desk" {
			command ClaimDesk {
			}
			automation FreeDeskAtClosing {
				on DeskClaimed
				reads DeskOccupancyView
				command ClaimDesk
			}
		}
	}
}`
			for _, tc := range []struct {
				entry     string
				line      int
				character int
			}{
				{entry: "on", line: 6, character: 7},
				{entry: "reads", line: 7, character: 10},
			} {
				t.Run("after "+tc.entry, func(t *testing.T) {
					result := lsp.GetCompletions(doc, tc.line, tc.character)

					require.Equal(t, []lsp.CompletionItem{}, result.Items)
				})
			}
		})
	})

	t.Run("quoted strings", func(t *testing.T) {
		t.Run("string contents neither start a comment nor open or close a block", func(t *testing.T) {
			for _, description := range []string{"plain text", "a # b", "a { b", "a } b", "a // b"} {
				doc := `context Ctx {
	aggregate Agg {
		slice Slc {
			view V { description "` + description + `" }
		}
		// cursor here
	}
}`
				result := lsp.GetCompletions(doc, 5, 2)
				require.Equal(t, []string{"slice", "invariant"}, extractLabels(result.Items), "description %q", description)
			}
		})
	})

	t.Run("completion items use keyword kind", func(t *testing.T) {
		t.Run("all items have Kind set to KeywordCompletion", func(t *testing.T) {
			result := lsp.GetCompletions("", 0, 0)
			requireItemKinds(t, result.Items, lsp.KeywordCompletion)
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

func requireItemKinds(t *testing.T, items []lsp.CompletionItem, kind lsp.CompletionItemKind) {
	t.Helper()
	for _, item := range items {
		require.Equal(t, kind, item.Kind, "kind of item %q", item.Label)
	}
}

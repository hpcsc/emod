//go:build unit

package lsp_test

import (
	"strings"
	"testing"

	"github.com/hpcsc/emod/internal/formatter"

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

	t.Run("spec block", func(t *testing.T) {
		specEntries := []string{"given", "when", "then"}

		const doc = `context "Lending" {
	aggregate "Loan" {
		slice "Borrow Copy" {
			command BorrowCopy {
			}
			event CopyBorrowed {
			}
			event CopyReturned {
			}
			spec "borrows a copy the member before returned" {
				given [CopyBorrowed, CopyReturned]
				when BorrowCopy
				then [CopyBorrowed]
			}
		}
	}
}`

		declaredEventItems := []lsp.CompletionItem{
			{Label: "CopyBorrowed", Kind: lsp.EventCompletion},
			{Label: "CopyReturned", Kind: lsp.EventCompletion},
		}

		t.Run("inside a spec block returns the entries a spec accepts", func(t *testing.T) {
			blankLineDoc := `context "Lending" {
	aggregate "Loan" {
		slice "Borrow Copy" {
			spec "borrows a copy no one holds" {

			}
		}
	}
}`
			result := lsp.GetCompletions(blankLineDoc, 4, 4)

			// A spec accepts no description, which the whole-list equality says.
			require.Equal(t, specEntries, extractLabels(result.Items))
			requireItemKinds(t, result.Items, lsp.KeywordCompletion)
		})

		t.Run("a spec block with its opening brace on the next line still returns the spec entries", func(t *testing.T) {
			braceOnNextLineDoc := `context "Lending" {
	aggregate "Loan" {
		slice "Borrow Copy" {
			spec "borrows a copy no one holds"
			{

			}
		}
	}
}`
			result := lsp.GetCompletions(braceOnNextLineDoc, 5, 4)

			require.Equal(t, specEntries, extractLabels(result.Items))
		})

		t.Run("after a closed spec block returns the enclosing slice keywords", func(t *testing.T) {
			closedDoc := `context "Lending" {
	aggregate "Loan" {
		slice "Borrow Copy" {
			spec "borrows a copy no one holds" {
				when BorrowCopy
			}

		}
	}
}`
			result := lsp.GetCompletions(closedDoc, 6, 3)

			require.Equal(t, []string{"command", "event", "trigger", "view", "automation", "translation", "flow"}, extractLabels(result.Items))
		})

		t.Run("a given list offers the event names the model declares in declaration order", func(t *testing.T) {
			line, char := posIn(t, doc, "given [CopyBorrowed, CopyReturned]", "given [CopyBorrowed, CopyReturned]")

			for _, cursor := range []struct {
				name      string
				character int
			}{
				{name: "immediately after the opening bracket", character: char + len("given [")},
				{name: "after an element and its comma", character: char + len("given [CopyBorrowed, ")},
			} {
				t.Run(cursor.name, func(t *testing.T) {
					result := lsp.GetCompletions(doc, line, cursor.character)

					require.Equal(t, declaredEventItems, result.Items)
				})
			}
		})

		t.Run("a then event list offers the same event names", func(t *testing.T) {
			line, char := posIn(t, doc, "then [CopyBorrowed]", "then [CopyBorrowed]")

			result := lsp.GetCompletions(doc, line, char+len("then ["))

			require.Equal(t, declaredEventItems, result.Items)
		})

		// A spec's when resolves against commands and events both, so both are
		// offered, each under the kind GetSemanticTokens paints that name with.
		t.Run("a when offers the commands and the events the model declares", func(t *testing.T) {
			line, char := posIn(t, doc, "when BorrowCopy", "when BorrowCopy")

			result := lsp.GetCompletions(doc, line, char+len("when "))

			require.Equal(t, append(
				[]lsp.CompletionItem{{Label: "BorrowCopy", Kind: lsp.FunctionCompletion}},
				declaredEventItems...,
			), result.Items)
		})

		t.Run("a then naming a view or a command offers that kind alone", func(t *testing.T) {
			outcomeDoc := `context "Lending" {
	aggregate "Loan" {
		slice "Review Member Loans" {
			command BorrowCopy {
			}
			event CopyBorrowed {
			}
			view MemberLoansView {
			}
			spec "lists the loans a member holds" {
				then view MemberLoansView
			}
			spec "reminds a member when a copy becomes due" {
				then command BorrowCopy
			}
		}
	}
}`
			viewLine, viewChar := posIn(t, outcomeDoc, "then view MemberLoansView", "then view MemberLoansView")
			viewResult := lsp.GetCompletions(outcomeDoc, viewLine, viewChar+len("then view "))
			require.Equal(t, []lsp.CompletionItem{
				{Label: "MemberLoansView", Kind: lsp.ClassCompletion},
			}, viewResult.Items)

			cmdLine, cmdChar := posIn(t, outcomeDoc, "then command BorrowCopy", "then command BorrowCopy")
			cmdResult := lsp.GetCompletions(outcomeDoc, cmdLine, cmdChar+len("then command "))
			require.Equal(t, []lsp.CompletionItem{
				{Label: "BorrowCopy", Kind: lsp.FunctionCompletion},
			}, cmdResult.Items)
		})

		// then becoming an event slot must not outrank rejected, which the
		// backward scan reaches first.
		t.Run("a then rejected still offers the invariants in scope and no event names", func(t *testing.T) {
			rejectionDoc := `context "Lending" {
	aggregate "Loan" {
		invariant OneCopyPerLoan "A loan covers exactly one copy of one title"
		slice "Borrow Copy" {
			command BorrowCopy {
			}
			event CopyBorrowed {
			}
			spec "refuses a copy already on loan" {
				when BorrowCopy
				then rejected OneCopyPerLoan
			}
		}
	}
}`
			line, char := posIn(t, rejectionDoc, "then rejected OneCopyPerLoan", "then rejected OneCopyPerLoan")

			result := lsp.GetCompletions(rejectionDoc, line, char+len("then rejected "))

			require.Equal(t, []lsp.CompletionItem{
				{Label: "OneCopyPerLoan", Kind: lsp.ConstantCompletion},
			}, result.Items)
		})

		t.Run("a cursor still touching a half-typed entry offers the spec entries", func(t *testing.T) {
			for _, entry := range []string{"given", "when", "then"} {
				t.Run(entry, func(t *testing.T) {
					halfTypedDoc := `context "Lending" {
	aggregate "Loan" {
		slice "Borrow Copy" {
			event CopyBorrowed {
			}
			spec "borrows a copy no one holds" {
				` + entry + `
			}
		}
	}
}`
					line, char := posIn(t, halfTypedDoc, `spec "borrows a copy no one holds"`, entry)

					result := lsp.GetCompletions(halfTypedDoc, line, char+len(entry))

					require.Equal(t, specEntries, extractLabels(result.Items))
				})
			}
		})

		t.Run("a field named after a spec entry still offers field types and modifiers", func(t *testing.T) {
			for _, entry := range []string{"given", "when", "then"} {
				t.Run(entry, func(t *testing.T) {
					fieldDoc := `context "Lending" {
	aggregate "Loan" {
		slice "Borrow Copy" {
			event CopyBorrowed {
				fields {
					` + entry + ` string required
				}
			}
		}
	}
}`
					line, char := posIn(t, fieldDoc, "fields {", entry+" string")

					result := lsp.GetCompletions(fieldDoc, line, char+len(entry)+1)

					require.Equal(t, []string{"string", "date", "timestamp", "int", "required", "optional"}, extractLabels(result.Items))
				})
			}
		})
	})

	t.Run("invariant names after rejected", func(t *testing.T) {
		// Three scopes, so a model-wide list is visibly wrong rather than
		// coincidentally right: two aggregates of one context, and a second
		// context declaring its own invariants over a slice of its own.
		const threeScopeDoc = `context "Lending" {
	aggregate "Loan" {
		invariant OneCopyPerLoan "A loan covers exactly one copy of one title"
		invariant FiveCopiesPerMember "A member holds at most five copies at one time"
		slice "Borrow Copy" {
			command BorrowCopy {
			}
			event CopyBorrowed {
			}
			spec "borrows a copy no one holds" {
				when BorrowCopy
				then [CopyBorrowed]
			}
			spec "refuses a copy already on loan" {
				when BorrowCopy
				then rejected OneCopyPerLoan
			}
		}
	}
	aggregate "Hold" {
		invariant OneHoldPerTitle "A member holds at most one copy of a title back"
		slice "Place Hold" {
			command PlaceHold {
			}
			spec "refuses a second hold" {
				when PlaceHold
				then rejected OneHoldPerTitle
			}
		}
	}
}

context "Reading Room" mode dcb {
	invariant OneReaderPerDesk "A desk seats at most one reader at any moment"
	invariant DeskFreeAtClosing "No desk stays claimed past the closing hour"
	slice "Claim Desk" {
		command ClaimDesk {
		}
		spec "refuses a desk another reader is seated at" {
			when ClaimDesk
			then rejected OneReaderPerDesk
		}
	}
}`

		t.Run("offers exactly the invariants of the scope holding the spec", func(t *testing.T) {
			for _, tc := range []struct {
				scope     string
				container string
				invariant string
				expected  []string
			}{
				{
					scope:     "the aggregate holding the slice",
					container: `spec "refuses a copy already on loan"`,
					invariant: "OneCopyPerLoan",
					expected:  []string{"OneCopyPerLoan", "FiveCopiesPerMember"},
				},
				{
					scope:     "a sibling aggregate of the same context",
					container: `spec "refuses a second hold"`,
					invariant: "OneHoldPerTitle",
					expected:  []string{"OneHoldPerTitle"},
				},
				{
					scope:     "a mode dcb context declaring the slice directly",
					container: `spec "refuses a desk another reader is seated at"`,
					invariant: "OneReaderPerDesk",
					expected:  []string{"OneReaderPerDesk", "DeskFreeAtClosing"},
				},
			} {
				t.Run(tc.scope, func(t *testing.T) {
					line, character := posIn(t, threeScopeDoc, tc.container, "then rejected "+tc.invariant)

					result := lsp.GetCompletions(threeScopeDoc, line, character+len("then rejected "))

					require.Equal(t, tc.expected, extractLabels(result.Items))
					requireItemKinds(t, result.Items, lsp.ConstantCompletion)
				})
			}
		})

		t.Run("a then with no rejected on the line offers the event names, not the invariants", func(t *testing.T) {
			line, char := posIn(t, threeScopeDoc, `spec "borrows a copy no one holds"`, "then [CopyBorrowed]")

			result := lsp.GetCompletions(threeScopeDoc, line, char+len("then ["))

			require.Equal(t, []string{"CopyBorrowed"}, extractLabels(result.Items))
		})

		// A half-typed word after `then ` is indistinguishable from a half-typed
		// event name, so the then slot answers and the client filters `rejected`
		// out of it. What matters here is that no invariant is offered until the
		// keyword is finished and a space typed after it.
		t.Run("a cursor still touching a half-typed rejected offers no invariant names", func(t *testing.T) {
			line, char := posIn(t, threeScopeDoc, `spec "refuses a copy already on loan"`, "rejected OneCopyPerLoan")

			result := lsp.GetCompletions(threeScopeDoc, line, char+len("rejected"))

			require.Equal(t, []string{"CopyBorrowed"}, extractLabels(result.Items))
		})

		t.Run("a spec whose enclosing braces are not yet closed still offers the invariants in scope", func(t *testing.T) {
			const truncatedDoc = `context "Lending" {
	aggregate "Loan" {
		invariant OneCopyPerLoan "A loan covers exactly one copy of one title"
		invariant FiveCopiesPerMember "A member holds at most five copies at one time"
		slice "Borrow Copy" {
			command BorrowCopy {
			}
			spec "refuses a copy already on loan" {
				when BorrowCopy
				then rejected `
			line := strings.Count(truncatedDoc, "\n")

			result := lsp.GetCompletions(truncatedDoc, line, len("\t\t\t\tthen rejected "))

			require.Equal(t, []string{"OneCopyPerLoan", "FiveCopiesPerMember"}, extractLabels(result.Items))
		})

		t.Run("an aggregate is not offered the invariants of the context enclosing it", func(t *testing.T) {
			const nestedDoc = `context "Lending" {
	invariant CardInGoodStanding "A member borrows only while their card is in good standing"
	aggregate "Loan" {
		invariant OneCopyPerLoan "A loan covers exactly one copy of one title"
		slice "Borrow Copy" {
			command BorrowCopy {
			}
			spec "refuses a copy already on loan" {
				when BorrowCopy
				then rejected OneCopyPerLoan
			}
		}
	}
}`
			line, char := posIn(t, nestedDoc, "then rejected OneCopyPerLoan", "then rejected OneCopyPerLoan")

			result := lsp.GetCompletions(nestedDoc, line, char+len("then rejected "))

			require.Equal(t, []string{"OneCopyPerLoan"}, extractLabels(result.Items))
		})

		t.Run("a field named rejected still offers field types and modifiers", func(t *testing.T) {
			const doc = `context "Lending" {
	aggregate "Loan" {
		invariant OneCopyPerLoan "A loan covers exactly one copy of one title"
		slice "Borrow Copy" {
			command BorrowCopy {
				fields {
					rejected string required
				}
			}
		}
	}
}`
			line, char := posIn(t, doc, "fields {", "rejected string")

			result := lsp.GetCompletions(doc, line, char+len("rejected "))

			require.Equal(t, []string{"string", "date", "timestamp", "int", "required", "optional"}, extractLabels(result.Items))
		})

		// The identifier immediately after `rejected` on a rejection edge is a
		// command name and the invariant sits after a second arrow, so the line
		// keeps offering what its enclosing flow block offers -- which is the top
		// level list, a flow body having no entry list of its own -- whichever way
		// the author spaces the colon.
		t.Run("a rejection edge on a flow line offers no invariant names", func(t *testing.T) {
			for _, entry := range []string{
				"command -> rejected: BorrowCopy -> OneCopyPerLoan",
				"command -> rejected : BorrowCopy -> OneCopyPerLoan",
			} {
				doc := `context "Lending" {
	aggregate "Loan" {
		invariant OneCopyPerLoan "A loan covers exactly one copy of one title"
		slice "Borrow Copy" {
			command BorrowCopy {
			}
			event CopyBorrowed {
			}
			flow {
				` + entry + `
			}
		}
	}
}`
				line, char := posIn(t, doc, entry, entry)

				for _, cursor := range []struct {
					name      string
					character int
				}{
					{name: "at the end of the entry", character: char + len(entry)},
					{name: "in the invariant's own position", character: char + len(entry) - len("OneCopyPerLoan")},
				} {
					t.Run(cursor.name, func(t *testing.T) {
						result := lsp.GetCompletions(doc, line, cursor.character)

						require.Equal(t, []string{"model", "actor", "context"}, extractLabels(result.Items), "entry %q", entry)
					})
				}
			}
		})
	})

	t.Run("payload field names", func(t *testing.T) {
		// Two constructs declaring different field sets, so a list that ignores
		// the element the payload hangs off is visibly wrong.
		const doc = `context "Lending" {
	aggregate "Loan" {
		slice "Borrow Copy" {
			command BorrowCopy {
				fields {
					memberId string required
					copyId   string required
				}
			}
			event CopyBorrowed {
				fields {
					loanId     string    required
					borrowedAt timestamp required
					reads      string
				}
			}
			event CopyReturned {
				fields {
					returnedAt timestamp required
				}
			}
			spec "borrows a copy the member before returned" {
				given [CopyBorrowed { loanId: "L-1" }, CopyReturned { returnedAt: "2024-07-05T14:32:00Z" }]
				when BorrowCopy { memberId: "M-40817", copyId: "C-93204" }
				then [CopyBorrowed { loanId: "L-2" }]
			}
		}
	}
}`

		borrowCopyFields := []string{"memberId", "copyId"}
		// CopyBorrowed's third field is named after a DSL keyword: a payload's
		// labels are field names, not keywords, and the whole-list assertions
		// below are what say so.
		copyBorrowedFields := []string{"loanId", "borrowedAt", "reads"}

		t.Run("offers the fields of the construct the enclosing element names", func(t *testing.T) {
			for _, tc := range []struct {
				entry    string
				line     string
				after    string
				expected []string
			}{
				{
					entry:    "given",
					line:     `given [CopyBorrowed { loanId: "L-1" }, CopyReturned { returnedAt: "2024-07-05T14:32:00Z" }]`,
					after:    `given [CopyBorrowed { `,
					expected: copyBorrowedFields,
				},
				{
					entry:    "a second element of the same given list",
					line:     `given [CopyBorrowed { loanId: "L-1" }, CopyReturned { returnedAt: "2024-07-05T14:32:00Z" }]`,
					after:    `given [CopyBorrowed { loanId: "L-1" }, CopyReturned { `,
					expected: []string{"returnedAt"},
				},
				{
					entry:    "when",
					line:     `when BorrowCopy { memberId: "M-40817", copyId: "C-93204" }`,
					after:    `when BorrowCopy { `,
					expected: borrowCopyFields,
				},
				{
					entry:    "then",
					line:     `then [CopyBorrowed { loanId: "L-2" }]`,
					after:    `then [CopyBorrowed { `,
					expected: copyBorrowedFields,
				},
			} {
				t.Run("on a "+tc.entry+" element", func(t *testing.T) {
					line, char := posIn(t, doc, tc.line, tc.line)

					result := lsp.GetCompletions(doc, line, char+len(tc.after))

					require.Equal(t, tc.expected, extractLabels(result.Items))
					requireItemKinds(t, result.Items, lsp.FieldCompletion)
				})
			}
		})

		t.Run("a field name already written is still offered, the client filtering the list", func(t *testing.T) {
			const whenLine = `when BorrowCopy { memberId: "M-40817", copyId: "C-93204" }`
			line, char := posIn(t, doc, whenLine, whenLine)

			result := lsp.GetCompletions(doc, line, char+len(`when BorrowCopy { memberId: "M-40817", `))

			require.Equal(t, borrowCopyFields, extractLabels(result.Items))
		})

		t.Run("a payload spanning several lines offers the same names on a continuation line", func(t *testing.T) {
			const multiLineDoc = `context "Lending" {
	aggregate "Loan" {
		slice "Borrow Copy" {
			command BorrowCopy {
				fields {
					memberId string required
					copyId   string required
				}
			}
			spec "borrows a copy no one holds" {
				when BorrowCopy {
					memberId: "M-40817",

				}
			}
		}
	}
}`
			line, _ := posIn(t, multiLineDoc, `memberId: "M-40817",`, `memberId: "M-40817",`)

			result := lsp.GetCompletions(multiLineDoc, line+1, 5)

			require.Equal(t, borrowCopyFields, extractLabels(result.Items))
		})

		// emod fmt wraps a spec entry past its column budget, leaving `given [` on
		// one line and the element it qualifies on the next, so the entry keyword
		// has to outlive the line that stated it.
		t.Run("a payload on an element wrapped below its given keyword still offers that construct's fields", func(t *testing.T) {
			const wrappedDoc = `context "Lending" {
	aggregate "Loan" {
		slice "Borrow Copy" {
			command BorrowCopy {
			}
			event CopyBorrowed {
				fields {
					loanId     string    required
					borrowedAt timestamp required
				}
			}
			spec "borrows a copy no one holds" {
				given [
					CopyBorrowed {
						loanId: "L-1",

					}
				]
				when BorrowCopy
			}
		}
	}
}`
			line, _ := posIn(t, wrappedDoc, `loanId: "L-1",`, `loanId: "L-1",`)

			result := lsp.GetCompletions(wrappedDoc, line+1, 6)

			require.Equal(t, []string{"loanId", "borrowedAt"}, extractLabels(result.Items))
		})

		// An automation slice's when names the triggering event rather than a
		// command, so the payload on it resolves against events too.
		t.Run("a payload on a when naming an event offers that event's fields", func(t *testing.T) {
			const automationDoc = `context "Lending" {
	aggregate "Loan" {
		slice "Chase Overdue Copy" {
			event CopyBorrowed {
				fields {
					loanId string required
					dueOn  date   required
				}
			}
			command RemindMember {
				fields {
					memberId string required
				}
			}
			event MemberReminded {
			}
			automation RemindOnDueDate {
				on CopyBorrowed
				command RemindMember
			}
			spec "reminds a member when a copy becomes due" {
				when CopyBorrowed { loanId: "L-1" }
				then [MemberReminded]
			}
			flow {
				command -> event: RemindMember -> MemberReminded
			}
		}
	}
}`
			const whenLine = `when CopyBorrowed { loanId: "L-1" }`
			line, char := posIn(t, automationDoc, whenLine, whenLine)

			result := lsp.GetCompletions(automationDoc, line, char+len(`when CopyBorrowed { `))

			require.Equal(t, []string{"loanId", "dueOn"}, extractLabels(result.Items))
			requireItemKinds(t, result.Items, lsp.FieldCompletion)
		})

		// emod fmt puts each element of a wrapped list on its own line, so the
		// entry keyword has to outlive every line until the bracket that opened
		// the list closes — not just the line that stated it.
		t.Run("every element of a wrapped list resolves against its own construct", func(t *testing.T) {
			const wrappedDoc = `context "Lending" {
	aggregate "Loan" {
		slice "Borrow Copy" {
			command BorrowCopy {
			}
			event CopyBorrowed {
				fields {
					loanId     string    required
					borrowedAt timestamp required
				}
			}
			event CopyReturned {
				fields {
					returnedAt timestamp required
				}
			}
			spec "borrows a copy the member before returned" {
				given [
					# the copy this member had out before
					CopyBorrowed {
						loanId: "L-1",

					},
					CopyReturned {
						returnedAt: "2024-07-05T14:32:00Z",

					}
				]
				when BorrowCopy
			}
		}
	}
}`
			for _, tc := range []struct {
				element  string
				marker   string
				expected []string
			}{
				{element: "the first", marker: `loanId: "L-1",`, expected: []string{"loanId", "borrowedAt"}},
				{element: "the second", marker: `returnedAt: "2024-07-05T14:32:00Z",`, expected: []string{"returnedAt"}},
			} {
				t.Run(tc.element+" element, on a continuation line", func(t *testing.T) {
					line, _ := posIn(t, wrappedDoc, tc.marker, tc.marker)

					result := lsp.GetCompletions(wrappedDoc, line+1, 6)

					require.Equal(t, tc.expected, extractLabels(result.Items))
				})
			}

			t.Run("a comment between the bracket and the first element does not lose the entry", func(t *testing.T) {
				line, char := posIn(t, wrappedDoc, "# the copy this member had out before", "CopyBorrowed {")

				result := lsp.GetCompletions(wrappedDoc, line, char+len("CopyBorrowed { "))

				require.Equal(t, []string{"loanId", "borrowedAt"}, extractLabels(result.Items))
			})

			t.Run("a caret past one element's closing brace offers the event names the list accepts", func(t *testing.T) {
				line, _ := posIn(t, wrappedDoc, `loanId: "L-1",`, `loanId: "L-1",`)

				result := lsp.GetCompletions(wrappedDoc, line+2, len("\t\t\t\t\t},"))

				require.Equal(t, []string{"CopyBorrowed", "CopyReturned"}, extractLabels(result.Items))
			})
		})

		t.Run("a payload naming a construct the model does not declare offers nothing at all", func(t *testing.T) {
			const undeclaredDoc = `context "Lending" {
	aggregate "Loan" {
		slice "Borrow Copy" {
			command BorrowCopy {
				fields {
					memberId string required
				}
			}
			spec "borrows a copy no one holds" {
				given [CopyReturned { returnedAt: "2024-07-05" }]
				when BorrowCopy
			}
		}
	}
}`
			line, char := posIn(t, undeclaredDoc, `given [CopyReturned {`, `given [CopyReturned {`)

			result := lsp.GetCompletions(undeclaredDoc, line, char+len(`given [CopyReturned { `))

			require.Equal(t, []lsp.CompletionItem{}, result.Items)
		})

		// A payload's opening brace sits on the line of the reference it
		// qualifies, matching the parser, so a brace written on the next line
		// opens no payload.
		t.Run("a brace on the line after a spec element opens no payload", func(t *testing.T) {
			const braceBelowDoc = `context "Lending" {
	aggregate "Loan" {
		slice "Borrow Copy" {
			command BorrowCopy {
				fields {
					memberId string required
				}
			}
			spec "borrows a copy no one holds" {
				when BorrowCopy
				{

				}
			}
		}
	}
}`
			result := lsp.GetCompletions(braceBelowDoc, 11, 4)

			require.Equal(t, []string{"model", "actor", "context"}, extractLabels(result.Items))
		})

		// A bracket the author has not closed belongs to the block it was opened
		// in. Carrying it further would let one stray `[` claim every brace in
		// the rest of the file, so later blocks would offer nothing at all.
		t.Run("an unclosed list does not reach past the block that opened it", func(t *testing.T) {
			const strayDoc = `context "Lending" {
	aggregate "Loan" {
		slice "Borrow Copy" {
			spec "borrows a copy no one holds" {
				given [
			}
		}
		slice "Return Copy" {
			command ReturnCopy {

			}
		}
	}
	aggregate "Hold" {

	}
}

`
			for _, tc := range []struct {
				position string
				line     int
				expected []string
			}{
				{position: "a later sibling slice's command body", line: 10, expected: []string{"fields", "decides_on"}},
				{position: "a later sibling aggregate body", line: 14, expected: []string{"slice", "invariant"}},
				{position: "the top level below everything", line: 17, expected: []string{"model", "actor", "context"}},
			} {
				t.Run(tc.position, func(t *testing.T) {
					result := lsp.GetCompletions(strayDoc, tc.line, 0)

					require.Equal(t, tc.expected, extractLabels(result.Items))
				})
			}
		})

		t.Run("a payload field spelled like a spec entry does not reclaim the list", func(t *testing.T) {
			const hijackDoc = `context "Lending" {
	aggregate "Loan" {
		slice "Borrow Copy" {
			command BorrowCopy {
			}
			event CopyBorrowed {
				fields {
					when   string required
					loanId string required
				}
			}
			event CopyReturned {
			}
			spec "borrows a copy no one holds" {
				given [
					CopyBorrowed {
						when: "2024-07-05T14:32:00Z",
					}
				]
				when BorrowCopy
			}
		}
	}
}`
			line, _ := posIn(t, hijackDoc, `when: "2024-07-05T14:32:00Z",`, `when: "2024-07-05T14:32:00Z",`)

			result := lsp.GetCompletions(hijackDoc, line+1, len("\t\t\t\t\t}"))

			require.Equal(t, []string{"CopyBorrowed", "CopyReturned"}, extractLabels(result.Items))
		})

		// given and then accept events only, while when accepts either kind, so
		// a command named in a given list qualifies nothing.
		t.Run("a given element naming a declared command offers nothing", func(t *testing.T) {
			const kindDoc = `context "Lending" {
	aggregate "Loan" {
		slice "Borrow Copy" {
			command BorrowCopy {
				fields {
					memberId string required
				}
			}
			event CopyBorrowed {
				fields {
					loanId string required
				}
			}
			spec "borrows a copy no one holds" {
				given [BorrowCopy { memberId: "M-1" }]
				when BorrowCopy { memberId: "M-1" }
				then [CopyBorrowed { loanId: "L-1" }]
			}
		}
	}
}`
			givenLine, givenChar := posIn(t, kindDoc, `given [BorrowCopy { memberId: "M-1" }]`, `given [BorrowCopy { memberId: "M-1" }]`)
			givenResult := lsp.GetCompletions(kindDoc, givenLine, givenChar+len(`given [BorrowCopy { `))
			require.Equal(t, []lsp.CompletionItem{}, givenResult.Items)

			whenLine, whenChar := posIn(t, kindDoc, `when BorrowCopy { memberId: "M-1" }`, `when BorrowCopy { memberId: "M-1" }`)
			whenResult := lsp.GetCompletions(kindDoc, whenLine, whenChar+len(`when BorrowCopy { `))
			require.Equal(t, []string{"memberId"}, extractLabels(whenResult.Items))
		})

		// The shared fixture formatted by emod fmt is the shape a user's own file
		// has, wrapped lists and all, rather than one authored to suit the test.
		t.Run("the shared payload fixture completes at every line of a wrapped list", func(t *testing.T) {
			formatted := formatter.Format(test.PayloadLibraryLendingModel(t))
			lines := strings.Split(formatted, "\n")

			opening := -1
			for i, l := range lines {
				if strings.TrimSpace(l) == "given [" {
					opening = i
					break
				}
			}
			require.GreaterOrEqual(t, opening, 0, "emod fmt is expected to wrap a given list in this fixture")

			// The fixture's own source states the element and the fields it
			// declares; the formatter decides only where the lines break.
			require.Equal(t, "DeskClaimed {", strings.TrimSpace(lines[opening+1]))
			deskClaimedFields := []string{"sessionId", "deskId", "memberId", "claimedAt", "quietZone"}

			for _, offset := range []int{1, 2, 3} {
				require.Equal(
					t, deskClaimedFields,
					extractLabels(lsp.GetCompletions(formatted, opening+offset, len(lines[opening+offset])).Items),
					"line %d of the wrapped payload: %q", opening+offset, lines[opening+offset],
				)
			}
		})

		t.Run("outside a payload the surrounding lists are unchanged", func(t *testing.T) {
			for _, tc := range []struct {
				position string
				line     int
				char     int
				expected []string
			}{
				{position: "a blank line in the spec body", line: 22, char: 3, expected: []string{"given", "when", "then"}},
				{position: "a blank line in the fields block", line: 6, char: 5, expected: []string{"string", "date", "timestamp", "int", "required", "optional"}},
			} {
				t.Run(tc.position, func(t *testing.T) {
					lines := strings.Split(doc, "\n")
					blanked := append([]string{}, lines[:tc.line]...)
					blanked = append(blanked, "")
					blanked = append(blanked, lines[tc.line:]...)

					result := lsp.GetCompletions(strings.Join(blanked, "\n"), tc.line, tc.char)

					require.Equal(t, tc.expected, extractLabels(result.Items))
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

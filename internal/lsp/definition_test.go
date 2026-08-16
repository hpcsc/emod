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

	t.Run("every construct a spec names resolves to its declaration", func(t *testing.T) {
		t.Run("in a slice an aggregate holds", func(t *testing.T) {
			doc := test.SpecLibraryLending

			t.Run("an element of a given list", func(t *testing.T) {
				cLine, cChar := posIn(t, doc, `spec "refuses a copy already on loan"`, "CopyBorrowed")
				dLine, dChar := posIn(t, doc, "event CopyBorrowed", "CopyBorrowed")
				assertDef(t, doc, cLine, cChar, dLine, dChar, "CopyBorrowed")
			})

			t.Run("an element of a then event list", func(t *testing.T) {
				cLine, cChar := posIn(t, doc, "then [CopyReturned]", "CopyReturned")
				dLine, dChar := posIn(t, doc, "event CopyReturned", "CopyReturned")
				assertDef(t, doc, cLine, cChar, dLine, dChar, "CopyReturned")
			})
		})

		t.Run("in a slice a mode dcb context holds directly", func(t *testing.T) {
			doc := test.SpecLibraryLending

			t.Run("an element of a given list", func(t *testing.T) {
				cLine, cChar := posIn(t, doc, `spec "refuses a desk another reader is seated at"`, "DeskClaimed")
				dLine, dChar := posIn(t, doc, "event DeskClaimed", "DeskClaimed")
				assertDef(t, doc, cLine, cChar, dLine, dChar, "DeskClaimed")
			})

			t.Run("an element of a then event list", func(t *testing.T) {
				cLine, cChar := posIn(t, doc, "then [DeskReleased]", "DeskReleased")
				dLine, dChar := posIn(t, doc, "event DeskReleased", "DeskReleased")
				assertDef(t, doc, cLine, cChar, dLine, dChar, "DeskReleased")
			})
		})

		// A spec's when resolves against commands and events both, because a
		// command slice's when names a command while an automation slice's names
		// the triggering event.
		t.Run("a when naming a command, and a when naming an event", func(t *testing.T) {
			doc := test.SlicePatternLibraryLending

			cmdLine, cmdChar := posIn(t, doc, "when RemindMember", "RemindMember")
			cmdDeclLine, cmdDeclChar := posIn(t, doc, "command RemindMember", "RemindMember")
			assertDef(t, doc, cmdLine, cmdChar, cmdDeclLine, cmdDeclChar, "RemindMember")

			evtLine, evtChar := posIn(t, doc, "when CopyBorrowed", "CopyBorrowed")
			evtDeclLine, evtDeclChar := posIn(t, doc, "event CopyBorrowed", "CopyBorrowed")
			assertDef(t, doc, evtLine, evtChar, evtDeclLine, evtDeclChar, "CopyBorrowed")
		})

		t.Run("a then naming a view, and a then naming a command", func(t *testing.T) {
			doc := test.SlicePatternLibraryLending

			viewLine, viewChar := posIn(t, doc, "then view MemberLoansView", "MemberLoansView")
			viewDeclLine, viewDeclChar := posIn(t, doc, "view MemberLoansView", "MemberLoansView")
			assertDef(t, doc, viewLine, viewChar, viewDeclLine, viewDeclChar, "MemberLoansView")

			cmdLine, cmdChar := posIn(t, doc, "then command RecallCopy", "RecallCopy")
			cmdDeclLine, cmdDeclChar := posIn(t, doc, "command RecallCopy", "RecallCopy")
			assertDef(t, doc, cmdLine, cmdChar, cmdDeclLine, cmdDeclChar, "RecallCopy")
		})

		t.Run("a then rejected invariant name resolves in the scope that declares it", func(t *testing.T) {
			doc := test.SpecLibraryLending

			for _, tc := range []struct {
				home      string
				invariant string
			}{
				{home: "declared on an aggregate", invariant: "OneCopyPerLoan"},
				{home: "declared directly on a mode dcb context", invariant: "OneReaderPerDesk"},
			} {
				t.Run(tc.home, func(t *testing.T) {
					cLine, cChar := posIn(t, doc, "then rejected "+tc.invariant, tc.invariant)
					dLine, dChar := posIn(t, doc, "invariant "+tc.invariant, tc.invariant)
					assertDef(t, doc, cLine, cChar, dLine, dChar, tc.invariant)
				})
			}
		})

		t.Run("each of two scopes declaring one invariant name jumps to its own declaration", func(t *testing.T) {
			const doc = `context "Lending" {
    aggregate "Loan" {
        invariant OneAtATime "A loan covers exactly one copy of one title"
        slice "Borrow Copy" {
            command BorrowCopy {
            }
            spec "refuses a second copy of the title" {
                when BorrowCopy
                then rejected OneAtATime
            }
        }
    }
    aggregate "Hold" {
        invariant OneAtATime "A member holds at most one title back at a time"
        slice "Place Hold" {
            command PlaceHold {
            }
            spec "refuses a second hold" {
                when PlaceHold
                then rejected OneAtATime
            }
        }
    }
}`

			loanLine, loanChar := posIn(t, doc, `spec "refuses a second copy of the title"`, "OneAtATime")
			loanDeclLine, loanDeclChar := posIn(t, doc, `aggregate "Loan"`, "OneAtATime")
			assertDef(t, doc, loanLine, loanChar, loanDeclLine, loanDeclChar, "OneAtATime")

			holdLine, holdChar := posIn(t, doc, `spec "refuses a second hold"`, "OneAtATime")
			holdDeclLine, holdDeclChar := posIn(t, doc, `aggregate "Hold"`, "OneAtATime")
			assertDef(t, doc, holdLine, holdChar, holdDeclLine, holdDeclChar, "OneAtATime")
		})

		t.Run("an aggregate does not resolve against the invariants of the context enclosing it", func(t *testing.T) {
			const doc = `context "Lending" {
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
            spec "refuses a member whose card has lapsed" {
                when BorrowCopy
                then rejected CardInGoodStanding
            }
        }
    }
}`
			ownLine, ownChar := posIn(t, doc, "then rejected OneCopyPerLoan", "OneCopyPerLoan")
			declLine, declChar := posIn(t, doc, "invariant OneCopyPerLoan", "OneCopyPerLoan")
			assertDef(t, doc, ownLine, ownChar, declLine, declChar, "OneCopyPerLoan")

			enclosingLine, enclosingChar := posIn(t, doc, "then rejected CardInGoodStanding", "CardInGoodStanding")
			assertNil(t, doc, enclosingLine, enclosingChar)
		})

		// The command equivalent is pinned below; an invariant declaration must
		// not resolve as a reference to itself either.
		t.Run("a cursor on an invariant declaration returns nil", func(t *testing.T) {
			doc := test.SpecLibraryLending
			line, char := posIn(t, doc, "invariant OneCopyPerLoan", "OneCopyPerLoan")
			assertNil(t, doc, line, char)
		})

		t.Run("a then rejected naming an invariant of no enclosing scope returns nil", func(t *testing.T) {
			const doc = `context "Lending" {
    aggregate "Loan" {
        invariant OneCopyPerLoan "A loan covers exactly one copy of one title"
        slice "Borrow Copy" {
            command BorrowCopy {
            }
        }
    }
    aggregate "Hold" {
        slice "Place Hold" {
            command PlaceHold {
            }
            spec "refuses a second hold" {
                when PlaceHold
                then rejected OneCopyPerLoan
            }
        }
    }
}`
			line, char := posIn(t, doc, "then rejected OneCopyPerLoan", "OneCopyPerLoan")
			assertNil(t, doc, line, char)
		})

		t.Run("a spec naming a construct the model does not declare yields no jump", func(t *testing.T) {
			const doc = `context "Lending" {
    aggregate "Loan" {
        slice "Borrow Copy" {
            command BorrowCopy {
            }
            event CopyBorrowed {
            }
            spec "borrows a copy no one holds" {
                given [CopyReturned]
                when BorrowCopy
                then [CopyBorrowed]
            }
        }
    }
}`
			line, char := posIn(t, doc, "given [CopyReturned]", "CopyReturned")
			assertNil(t, doc, line, char)

			declaredLine, declaredChar := posIn(t, doc, "then [CopyBorrowed]", "CopyBorrowed")
			dLine, dChar := posIn(t, doc, "event CopyBorrowed", "CopyBorrowed")
			assertDef(t, doc, declaredLine, declaredChar, dLine, dChar, "CopyBorrowed")
		})
	})

	t.Run("both names a flow rejection edge carries resolve to their declarations", func(t *testing.T) {
		doc := test.RejectionLibraryLending

		for _, tc := range []struct {
			home      string
			edge      string
			command   string
			invariant string
		}{
			{
				home:      "in a slice an aggregate holds",
				edge:      "command -> rejected: BorrowCopy -> OneCopyPerLoan",
				command:   "BorrowCopy",
				invariant: "OneCopyPerLoan",
			},
			{
				home:      "in a slice a mode dcb context holds directly",
				edge:      "command -> rejected: ClaimDesk -> OneReaderPerDesk",
				command:   "ClaimDesk",
				invariant: "OneReaderPerDesk",
			},
		} {
			t.Run(tc.home, func(t *testing.T) {
				cmdLine, cmdChar := posIn(t, doc, tc.edge, tc.command)
				cmdDeclLine, cmdDeclChar := posIn(t, doc, "command "+tc.command+" {", tc.command)
				assertDef(t, doc, cmdLine, cmdChar, cmdDeclLine, cmdDeclChar, tc.command)

				invLine, invChar := posIn(t, doc, tc.edge, tc.invariant)
				invDeclLine, invDeclChar := posIn(t, doc, "invariant "+tc.invariant, tc.invariant)
				assertDef(t, doc, invLine, invChar, invDeclLine, invDeclChar, tc.invariant)
			})
		}
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

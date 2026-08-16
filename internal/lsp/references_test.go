//go:build unit

package lsp_test

import (
	"strings"
	"testing"

	"github.com/hpcsc/emod/internal/lsp"
	"github.com/hpcsc/emod/internal/test"
	"github.com/stretchr/testify/require"
)

func TestGetReferences(t *testing.T) {
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

	// requireLocation checks that locs contains a Location matching the expected
	// line, character, and name. The range end is char+len(name).
	requireLocation := func(t *testing.T, locs []lsp.Location, expectedLine, expectedChar int, expectedName string) {
		t.Helper()
		found := false
		for _, loc := range locs {
			if loc.URI == uri &&
				loc.Range.Start.Line == expectedLine &&
				loc.Range.Start.Character == expectedChar &&
				loc.Range.End.Line == expectedLine &&
				loc.Range.End.Character == expectedChar+len(expectedName) {
				found = true
				break
			}
		}
		require.True(t, found, "expected location at line=%d char=%d name=%q not found in %d locations",
			expectedLine, expectedChar, expectedName, len(locs))
	}

	// locationOf builds the Location a reference at the given name inside
	// container should be reported as.
	locationOf := func(t *testing.T, doc, container, name string) lsp.Location {
		t.Helper()
		line, char := posIn(t, doc, container, name)
		return lsp.Location{
			URI: uri,
			Range: lsp.Range{
				Start: lsp.Position{Line: line, Character: char},
				End:   lsp.Position{Line: line, Character: char + len(name)},
			},
		}
	}

	t.Run("events", func(t *testing.T) {
		t.Run("cursor on event definition name returns all event references", func(t *testing.T) {
			cLine, cChar := posIn(t, testDoc, "event OrderSubmitted", "OrderSubmitted")
			locs := lsp.GetReferences(testDoc, cLine, cChar, uri)
			require.Len(t, locs, 4)

			dLine, dChar := posIn(t, testDoc, "event OrderSubmitted", "OrderSubmitted")
			requireLocation(t, locs, dLine, dChar, "OrderSubmitted")

			sLine, sChar := posIn(t, testDoc, "subscribes [OrderSubmitted]", "OrderSubmitted")
			requireLocation(t, locs, sLine, sChar, "OrderSubmitted")

			tLine, tChar := posIn(t, testDoc,
				"automation AutoSubmit {\n                on OrderSubmitted", "OrderSubmitted")
			requireLocation(t, locs, tLine, tChar, "OrderSubmitted")

			flowLine := "command -> event : SubmitOrder -> OrderSubmitted"
			fLine, fChar := posIn(t, testDoc, flowLine, "OrderSubmitted")
			requireLocation(t, locs, fLine, fChar, "OrderSubmitted")
		})

		t.Run("cursor on an event declared on a mode dcb context returns every site naming it", func(t *testing.T) {
			doc := test.AutomationReadsLibraryLending

			cLine, cChar := posIn(t, doc, "event DeskReleased", "DeskReleased")
			locs := lsp.GetReferences(doc, cLine, cChar, uri)

			require.Equal(t, []lsp.Location{
				locationOf(t, doc, "event DeskReleased", "DeskReleased"),
				locationOf(t, doc, "command -> event: ReleaseDesk -> DeskReleased", "DeskReleased"),
				locationOf(t, doc, "subscribes [DeskClaimed, DeskReleased]", "DeskReleased"),
				locationOf(t, doc, "automation RemindReaderOfLoans", "DeskReleased"),
			}, locs)
		})

		t.Run("cursor on event reference in subscribes returns all event references", func(t *testing.T) {
			cLine, cChar := posIn(t, testDoc, "subscribes [OrderSubmitted]", "OrderSubmitted")
			locs := lsp.GetReferences(testDoc, cLine, cChar, uri)
			require.Len(t, locs, 4)
		})

		t.Run("cursor on event reference in an automation activation event returns all event references", func(t *testing.T) {
			cLine, cChar := posIn(t, testDoc,
				"automation AutoSubmit {\n                on OrderSubmitted", "OrderSubmitted")
			locs := lsp.GetReferences(testDoc, cLine, cChar, uri)
			require.Len(t, locs, 4)
		})

		t.Run("cursor on event reference in flow event entry returns all event references", func(t *testing.T) {
			flowLine := "command -> event : SubmitOrder -> OrderSubmitted"
			cLine, cChar := posIn(t, testDoc, flowLine, "OrderSubmitted")
			locs := lsp.GetReferences(testDoc, cLine, cChar, uri)
			require.Len(t, locs, 4)
		})
	})

	t.Run("commands", func(t *testing.T) {
		t.Run("cursor on command definition name returns all command references", func(t *testing.T) {
			cLine, cChar := posIn(t, testDoc, "command SubmitOrder", "SubmitOrder")
			locs := lsp.GetReferences(testDoc, cLine, cChar, uri)
			require.Len(t, locs, 4)

			dLine, dChar := posIn(t, testDoc, "command SubmitOrder", "SubmitOrder")
			requireLocation(t, locs, dLine, dChar, "SubmitOrder")

			aLine, aChar := posIn(t, testDoc, "automation AutoSubmit", "SubmitOrder")
			requireLocation(t, locs, aLine, aChar, "SubmitOrder")

			trLine, trChar := posIn(t, testDoc, "translation TransOrder", "SubmitOrder")
			requireLocation(t, locs, trLine, trChar, "SubmitOrder")

			flowLine := "command -> event : SubmitOrder -> OrderSubmitted"
			colonIdx := strings.Index(flowLine, ": ")
			require.GreaterOrEqual(t, colonIdx, 0)
			fLine, fChar := posIn(t, testDoc, flowLine, flowLine[colonIdx:])
			fChar += 2 // skip ": " to point to 'S' of SubmitOrder
			requireLocation(t, locs, fLine, fChar, "SubmitOrder")
		})

		t.Run("cursor on a command an aggregate declares and a mode dcb context names returns both sites", func(t *testing.T) {
			doc := test.AutomationReadsLibraryLending

			cLine, cChar := posIn(t, doc, "command RemindMember {", "RemindMember")
			locs := lsp.GetReferences(doc, cLine, cChar, uri)

			require.Equal(t, []lsp.Location{
				locationOf(t, doc, "command RemindMember {", "RemindMember"),
				locationOf(t, doc, "automation RemindOnDueDate", "RemindMember"),
				locationOf(t, doc, "command -> event: RemindMember -> MemberReminded", "RemindMember"),
				locationOf(t, doc, "automation RemindReaderOfLoans", "RemindMember"),
			}, locs)
		})

		t.Run("sites in a context's own slices follow the sites in its aggregates", func(t *testing.T) {
			const doc = `context "Reading Room" mode dcb {
    aggregate "Desk" {
        slice "Claim Desk" {
            command ClaimDesk {
            }
            automation ClaimOnArrival {
                command ClaimDesk
            }
        }
    }
    slice "Close Reading Room" {
        automation ClaimAtOpening {
            command ClaimDesk
        }
    }
}
`

			cLine, cChar := posIn(t, doc, "command ClaimDesk {", "ClaimDesk")
			locs := lsp.GetReferences(doc, cLine, cChar, uri)

			require.Equal(t, []lsp.Location{
				locationOf(t, doc, "command ClaimDesk {", "ClaimDesk"),
				locationOf(t, doc, "automation ClaimOnArrival", "ClaimDesk"),
				locationOf(t, doc, "automation ClaimAtOpening", "ClaimDesk"),
			}, locs)
		})

		t.Run("cursor on command in automation returns all command references", func(t *testing.T) {
			cLine, cChar := posIn(t, testDoc, "automation AutoSubmit", "SubmitOrder")
			locs := lsp.GetReferences(testDoc, cLine, cChar, uri)
			require.Len(t, locs, 4)
		})

		t.Run("cursor on command in translation returns all command references", func(t *testing.T) {
			cLine, cChar := posIn(t, testDoc, "translation TransOrder", "SubmitOrder")
			locs := lsp.GetReferences(testDoc, cLine, cChar, uri)
			require.Len(t, locs, 4)
		})

		t.Run("cursor on command in flow returns all command references", func(t *testing.T) {
			flowLine := "command -> event : SubmitOrder -> OrderSubmitted"
			colonIdx := strings.Index(flowLine, ": ")
			require.GreaterOrEqual(t, colonIdx, 0)
			cLine, cChar := posIn(t, testDoc, flowLine, flowLine[colonIdx:])
			cChar += 2 // skip ": " to point to 'S' of SubmitOrder
			locs := lsp.GetReferences(testDoc, cLine, cChar, uri)
			require.Len(t, locs, 4)
		})
	})

	t.Run("views", func(t *testing.T) {
		orderViewSites := func(t *testing.T) []lsp.Location {
			t.Helper()
			return []lsp.Location{
				locationOf(t, testDoc, "view OrderView", "OrderView"),
				locationOf(t, testDoc, "automation AutoSubmit", "OrderView"),
				locationOf(t, testDoc, "translation TransOrder", "OrderView"),
				locationOf(t, testDoc, "trigger \"MyTrigger\"", "OrderView"),
			}
		}

		t.Run("cursor on view definition name returns the declaration and every site reading it", func(t *testing.T) {
			cLine, cChar := posIn(t, testDoc, "view OrderView", "OrderView")
			locs := lsp.GetReferences(testDoc, cLine, cChar, uri)

			require.Equal(t, orderViewSites(t), locs)
		})

		t.Run("cursor on view in automation reads returns all view references", func(t *testing.T) {
			cLine, cChar := posIn(t, testDoc, "automation AutoSubmit", "OrderView")
			locs := lsp.GetReferences(testDoc, cLine, cChar, uri)

			require.Equal(t, orderViewSites(t), locs)
		})

		t.Run("cursor on view in translation reads returns all view references", func(t *testing.T) {
			cLine, cChar := posIn(t, testDoc, "translation TransOrder", "OrderView")
			locs := lsp.GetReferences(testDoc, cLine, cChar, uri)
			require.Len(t, locs, 4)
		})

		t.Run("cursor on view in trigger reads returns all view references", func(t *testing.T) {
			cLine, cChar := posIn(t, testDoc, "trigger \"MyTrigger\"", "OrderView")
			locs := lsp.GetReferences(testDoc, cLine, cChar, uri)
			require.Len(t, locs, 4)
		})

		t.Run("cursor on a view declaration lists every construct reading it across slice and context boundaries", func(t *testing.T) {
			doc := test.AutomationReadsLibraryLending

			mLine, mChar := posIn(t, doc, "view MemberLoansView", "MemberLoansView")
			require.Equal(t, []lsp.Location{
				locationOf(t, doc, "view MemberLoansView", "MemberLoansView"),
				locationOf(t, doc, `trigger "Lending Desk"`, "MemberLoansView"),
				locationOf(t, doc, "automation RecallOverdueCopy", "MemberLoansView"),
				locationOf(t, doc, "automation RemindReaderOfLoans", "MemberLoansView"),
			}, lsp.GetReferences(doc, mLine, mChar, uri))

			dLine, dChar := posIn(t, doc, "view DeskOccupancyView", "DeskOccupancyView")
			require.Equal(t, []lsp.Location{
				locationOf(t, doc, "view DeskOccupancyView", "DeskOccupancyView"),
				locationOf(t, doc, "automation FreeDeskAtClosing", "DeskOccupancyView"),
			}, lsp.GetReferences(doc, dLine, dChar, uri))
		})

		t.Run("cursor on a view no automation reads returns its declaration alone", func(t *testing.T) {
			const doc = `context "Orders" {
    aggregate "Sales" {
        slice "Fulfilment" {
            view PickListView {
            }
            view ArchiveView {
            }
            automation PickOnPayment {
                reads PickListView
            }
        }
    }
}
`

			rLine, rChar := posIn(t, doc, "view PickListView", "PickListView")
			require.Equal(t, []lsp.Location{
				locationOf(t, doc, "view PickListView", "PickListView"),
				locationOf(t, doc, "automation PickOnPayment", "PickListView"),
			}, lsp.GetReferences(doc, rLine, rChar, uri))

			uLine, uChar := posIn(t, doc, "view ArchiveView", "ArchiveView")
			require.Equal(t, []lsp.Location{
				locationOf(t, doc, "view ArchiveView", "ArchiveView"),
			}, lsp.GetReferences(doc, uLine, uChar, uri))
		})
	})

	t.Run("specs", func(t *testing.T) {
		doc := test.SpecLibraryLending

		t.Run("an event a mode dcb context's own slices name lists every spec site beside the sites it listed before", func(t *testing.T) {
			expected := []lsp.Location{
				locationOf(t, doc, "event DeskClaimed", "DeskClaimed"),
				locationOf(t, doc, "then [DeskClaimed]", "DeskClaimed"),
				locationOf(t, doc, `spec "refuses a desk another reader is seated at"`, "DeskClaimed"),
				locationOf(t, doc, "command -> event: ClaimDesk -> DeskClaimed", "DeskClaimed"),
				locationOf(t, doc, `spec "frees the desk its reader is seated at"`, "DeskClaimed"),
				locationOf(t, doc, `spec "refuses to free a desk already empty"`, "DeskClaimed"),
			}

			declLine, declChar := posIn(t, doc, "event DeskClaimed", "DeskClaimed")
			require.Equal(t, expected, lsp.GetReferences(doc, declLine, declChar, uri), "from the declaration")

			specLine, specChar := posIn(t, doc, `spec "frees the desk its reader is seated at"`, "DeskClaimed")
			require.Equal(t, expected, lsp.GetReferences(doc, specLine, specChar, uri), "from a spec's given element")
		})

		t.Run("an event an aggregate's slices name lists its spec sites in the order they were written", func(t *testing.T) {
			expected := []lsp.Location{
				locationOf(t, doc, "event CopyReturned", "CopyReturned"),
				locationOf(t, doc, "given [CopyBorrowed, CopyReturned]", "CopyReturned"),
				locationOf(t, doc, "then [CopyReturned]", "CopyReturned"),
				locationOf(t, doc, "command -> event: ReturnCopy -> CopyReturned", "CopyReturned"),
			}

			declLine, declChar := posIn(t, doc, "event CopyReturned", "CopyReturned")
			require.Equal(t, expected, lsp.GetReferences(doc, declLine, declChar, uri), "from the declaration")

			thenLine, thenChar := posIn(t, doc, "then [CopyReturned]", "CopyReturned")
			require.Equal(t, expected, lsp.GetReferences(doc, thenLine, thenChar, uri), "from a spec's then element")
		})

		t.Run("a command lists every spec when naming it beside its flow entry", func(t *testing.T) {
			expected := []lsp.Location{
				locationOf(t, doc, "command ReturnCopy {", "ReturnCopy"),
				locationOf(t, doc, `spec "returns a copy the member holds"`, "ReturnCopy"),
				locationOf(t, doc, `spec "refuses to return a copy the member no longer holds"`, "ReturnCopy"),
				locationOf(t, doc, "command -> event: ReturnCopy -> CopyReturned", "ReturnCopy"),
			}

			declLine, declChar := posIn(t, doc, "command ReturnCopy {", "ReturnCopy")
			require.Equal(t, expected, lsp.GetReferences(doc, declLine, declChar, uri), "from the declaration")

			whenLine, whenChar := posIn(t, doc, `spec "returns a copy the member holds"`, "ReturnCopy")
			require.Equal(t, expected, lsp.GetReferences(doc, whenLine, whenChar, uri), "from a spec's when")
		})

		t.Run("a spec naming a construct the model does not declare contributes no location", func(t *testing.T) {
			const undeclaredDoc = `context "Lending" {
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

			line, char := posIn(t, undeclaredDoc, "given [CopyReturned]", "CopyReturned")
			require.Nil(t, lsp.GetReferences(undeclaredDoc, line, char, uri))

			declLine, declChar := posIn(t, undeclaredDoc, "event CopyBorrowed", "CopyBorrowed")
			require.Equal(t, []lsp.Location{
				locationOf(t, undeclaredDoc, "event CopyBorrowed", "CopyBorrowed"),
				locationOf(t, undeclaredDoc, "then [CopyBorrowed]", "CopyBorrowed"),
			}, lsp.GetReferences(undeclaredDoc, declLine, declChar, uri))
		})

		t.Run("a then naming a view or a command lists that site too", func(t *testing.T) {
			const patternDoc = `context "Lending" {
    aggregate "Loan" {
        slice "Chase Overdue Copy" {
            command RecallCopy {
            }
            event CopyRecalled {
            }
            view OverdueLoansView {
                subscribes [CopyRecalled]
            }
            automation RecallOverdueCopy {
                on CopyRecalled
                reads OverdueLoansView
                command RecallCopy
            }
            spec "lists the copies now overdue" {
                then view OverdueLoansView
            }
            spec "recalls copies that are overdue" {
                then command RecallCopy
            }
            flow {
                command -> event: RecallCopy -> CopyRecalled
            }
        }
    }
}`

			viewLine, viewChar := posIn(t, patternDoc, "view OverdueLoansView {", "OverdueLoansView")
			require.Equal(t, []lsp.Location{
				locationOf(t, patternDoc, "view OverdueLoansView {", "OverdueLoansView"),
				locationOf(t, patternDoc, "automation RecallOverdueCopy", "OverdueLoansView"),
				locationOf(t, patternDoc, "then view OverdueLoansView", "OverdueLoansView"),
			}, lsp.GetReferences(patternDoc, viewLine, viewChar, uri))

			cmdLine, cmdChar := posIn(t, patternDoc, "command RecallCopy {", "RecallCopy")
			require.Equal(t, []lsp.Location{
				locationOf(t, patternDoc, "command RecallCopy {", "RecallCopy"),
				locationOf(t, patternDoc, "automation RecallOverdueCopy", "RecallCopy"),
				locationOf(t, patternDoc, "then command RecallCopy", "RecallCopy"),
				locationOf(t, patternDoc, "command -> event: RecallCopy -> CopyRecalled", "RecallCopy"),
			}, lsp.GetReferences(patternDoc, cmdLine, cmdChar, uri))
		})
	})

	t.Run("invariants", func(t *testing.T) {
		doc := test.SpecLibraryLending

		t.Run("an invariant lists the declaration and every spec in its scope that rejects by it", func(t *testing.T) {
			for _, tc := range []struct {
				home     string
				name     string
				expected []lsp.Location
			}{
				{
					home: "declared on an aggregate",
					name: "OneCopyPerLoan",
					expected: []lsp.Location{
						locationOf(t, doc, "invariant OneCopyPerLoan", "OneCopyPerLoan"),
						locationOf(t, doc, `spec "refuses a copy already on loan"`, "OneCopyPerLoan"),
						locationOf(t, doc, `spec "refuses to return a copy the member no longer holds"`, "OneCopyPerLoan"),
					},
				},
				{
					home: "declared directly on a mode dcb context",
					name: "OneReaderPerDesk",
					expected: []lsp.Location{
						locationOf(t, doc, "invariant OneReaderPerDesk", "OneReaderPerDesk"),
						locationOf(t, doc, `spec "refuses a desk another reader is seated at"`, "OneReaderPerDesk"),
						locationOf(t, doc, `spec "refuses to free a desk already empty"`, "OneReaderPerDesk"),
					},
				},
			} {
				t.Run(tc.home, func(t *testing.T) {
					declLine, declChar := posIn(t, doc, "invariant "+tc.name, tc.name)
					require.Equal(t, tc.expected, lsp.GetReferences(doc, declLine, declChar, uri), "from the declaration")

					refLine, refChar := posIn(t, doc, "then rejected "+tc.name, tc.name)
					require.Equal(t, tc.expected, lsp.GetReferences(doc, refLine, refChar, uri), "from a then rejected")
				})
			}
		})

		t.Run("two scopes declaring one invariant name list only their own spec sites", func(t *testing.T) {
			const twoScopeDoc = `context "Lending" {
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

			loanLine, loanChar := posIn(t, twoScopeDoc, `aggregate "Loan"`, "OneAtATime")
			require.Equal(t, []lsp.Location{
				locationOf(t, twoScopeDoc, `aggregate "Loan"`, "OneAtATime"),
				locationOf(t, twoScopeDoc, `spec "refuses a second copy of the title"`, "OneAtATime"),
			}, lsp.GetReferences(twoScopeDoc, loanLine, loanChar, uri))

			holdLine, holdChar := posIn(t, twoScopeDoc, `aggregate "Hold"`, "OneAtATime")
			require.Equal(t, []lsp.Location{
				locationOf(t, twoScopeDoc, `aggregate "Hold"`, "OneAtATime"),
				locationOf(t, twoScopeDoc, `spec "refuses a second hold"`, "OneAtATime"),
			}, lsp.GetReferences(twoScopeDoc, holdLine, holdChar, uri))
		})

		t.Run("an aggregate's list never reaches the invariants of the context enclosing it", func(t *testing.T) {
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

			ownLine, ownChar := posIn(t, doc, "invariant OneCopyPerLoan", "OneCopyPerLoan")
			require.Equal(t, []lsp.Location{
				locationOf(t, doc, "invariant OneCopyPerLoan", "OneCopyPerLoan"),
				locationOf(t, doc, `spec "refuses a copy already on loan"`, "OneCopyPerLoan"),
			}, lsp.GetReferences(doc, ownLine, ownChar, uri))

			// The context declares CardInGoodStanding but holds no slice of its
			// own, so the aggregate's spec that rejects by it resolves nowhere.
			enclosingLine, enclosingChar := posIn(t, doc, "invariant CardInGoodStanding", "CardInGoodStanding")
			require.Equal(t, []lsp.Location{
				locationOf(t, doc, "invariant CardInGoodStanding", "CardInGoodStanding"),
			}, lsp.GetReferences(doc, enclosingLine, enclosingChar, uri))

			refLine, refChar := posIn(t, doc, "then rejected CardInGoodStanding", "CardInGoodStanding")
			require.Nil(t, lsp.GetReferences(doc, refLine, refChar, uri))
		})

		t.Run("a then rejected naming an invariant of no enclosing scope lists nothing", func(t *testing.T) {
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
			require.Nil(t, lsp.GetReferences(doc, line, char, uri))

			declLine, declChar := posIn(t, doc, "invariant OneCopyPerLoan", "OneCopyPerLoan")
			require.Equal(t, []lsp.Location{
				locationOf(t, doc, "invariant OneCopyPerLoan", "OneCopyPerLoan"),
			}, lsp.GetReferences(doc, declLine, declChar, uri))
		})
	})

	t.Run("nil returns", func(t *testing.T) {
		t.Run("cursor not on a resolvable name returns nil", func(t *testing.T) {
			t.Run("on a keyword", func(t *testing.T) {
				line, char := posIn(t, testDoc, "command SubmitOrder", "command")
				locs := lsp.GetReferences(testDoc, line, char, uri)
				require.Nil(t, locs)
			})

			t.Run("on a block keyword", func(t *testing.T) {
				line, char := posIn(t, testDoc, "view OrderView", "view")
				locs := lsp.GetReferences(testDoc, line, char, uri)
				require.Nil(t, locs)
			})
		})

		t.Run("cursor on reference to undefined name returns nil", func(t *testing.T) {
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
				locs := lsp.GetReferences(doc, line, char, uri)
				require.Nil(t, locs)
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
				locs := lsp.GetReferences(doc, line, char, uri)
				require.Nil(t, locs)
			})
		})

		t.Run("empty document returns nil", func(t *testing.T) {
			locs := lsp.GetReferences("", 0, 0, uri)
			require.Nil(t, locs)
		})
	})
}

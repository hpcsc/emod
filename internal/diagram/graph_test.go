//go:build unit

package diagram_test

import (
	"testing"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/diagram"
	"github.com/hpcsc/emod/internal/test"
	"github.com/stretchr/testify/require"
)

// wiredSlice declares one of every relation SliceEdges derives, so a whole-list
// comparison against it fails when an edge appears, disappears or moves.
func wiredSlice() *ast.Slice {
	return &ast.Slice{
		Name:     "Borrow Copy",
		Trigger:  &ast.Trigger{Name: "Lending Desk", Reads: "AvailableCopiesView"},
		Commands: []*ast.Command{{Name: "BorrowCopy"}},
		Events:   []*ast.Event{{Name: "CopyBorrowed"}},
		Flows:    []*ast.Flow{{CommandName: "BorrowCopy", EventName: "CopyBorrowed"}},
		Views: []*ast.View{
			{Name: "MemberLoansView", Subscribes: []string{"CopyBorrowed"}},
		},
		Automations: []*ast.Automation{
			{Name: "RecallOnSecondReminder", OnEvent: "MemberReminded", Reads: "MemberLoansView", Command: "RecallCopy"},
		},
		Translations: []*ast.Translation{
			{
				Name:           "PartnerFeed",
				ExternalSystem: "PartnerLibrary",
				Reads:          "MemberLoansView",
				Command:        "ImportLoan",
				Event:          &ast.Event{Name: "LoanImported"},
			},
		},
	}
}

func wiredSliceEdges() []diagram.Edge {
	return []diagram.Edge{
		{Kind: diagram.EdgeFlow, From: "BorrowCopy", To: "CopyBorrowed"},
		{Kind: diagram.EdgeTriggerReads, From: "AvailableCopiesView", To: "Lending Desk"},
		{Kind: diagram.EdgeTriggerCommand, From: "Lending Desk", To: "BorrowCopy"},
		{Kind: diagram.EdgeSubscription, From: "CopyBorrowed", To: "MemberLoansView"},
		{Kind: diagram.EdgeAutomationTrigger, From: "MemberReminded", To: "RecallOnSecondReminder"},
		{Kind: diagram.EdgeAutomationReads, From: "MemberLoansView", To: "RecallOnSecondReminder"},
		{Kind: diagram.EdgeAutomationCommand, From: "RecallOnSecondReminder", To: "RecallCopy"},
		{Kind: diagram.EdgeTranslationReads, From: "MemberLoansView", To: "PartnerFeed"},
		{Kind: diagram.EdgeTranslationExternal, From: "PartnerLibrary", To: "PartnerFeed"},
		{Kind: diagram.EdgeTranslationCommand, From: "PartnerFeed", To: "ImportLoan"},
		{Kind: diagram.EdgeTranslationFlow, From: "ImportLoan", To: "LoanImported"},
	}
}

func TestSliceEdges(t *testing.T) {
	t.Run("derivation", func(t *testing.T) {
		t.Run("a slice stating no rejection entry derives exactly the edges it always has", func(t *testing.T) {
			require.Equal(t, wiredSliceEdges(), diagram.SliceEdges(wiredSlice()))
		})

		t.Run("a rejection entry derives one edge naming its command and its invariant", func(t *testing.T) {
			s := &ast.Slice{
				Name:     "Borrow Copy",
				Commands: []*ast.Command{{Name: "BorrowCopy"}},
				Rejections: []*ast.Rejection{
					{CommandName: "BorrowCopy", InvariantName: "OneCopyPerLoan"},
				},
			}

			require.Equal(t, []diagram.Edge{
				{Kind: diagram.EdgeRejection, From: "BorrowCopy", To: "OneCopyPerLoan"},
			}, diagram.SliceEdges(s))
		})

		t.Run("rejection edges come back in declaration order beside the slice's flow edges", func(t *testing.T) {
			s := wiredSlice()
			s.Rejections = []*ast.Rejection{
				{CommandName: "BorrowCopy", InvariantName: "OneCopyPerLoan"},
				{CommandName: "BorrowCopy", InvariantName: "FiveCopiesPerMember"},
			}

			expected := wiredSliceEdges()
			withRejections := append([]diagram.Edge{expected[0],
				{Kind: diagram.EdgeRejection, From: "BorrowCopy", To: "OneCopyPerLoan"},
				{Kind: diagram.EdgeRejection, From: "BorrowCopy", To: "FiveCopiesPerMember"},
			}, expected[1:]...)

			require.Equal(t, withRejections, diagram.SliceEdges(s))
		})

		t.Run("the shared rejection fixture derives one edge per transcribed entry", func(t *testing.T) {
			model := test.RejectionLibraryLendingModel(t)

			var derived []test.RejectionEdge
			for _, ref := range model.SliceRefs() {
				for _, edge := range diagram.SliceEdges(ref.Slice) {
					if edge.Kind == diagram.EdgeRejection {
						derived = append(derived, test.RejectionEdge{CommandName: edge.From, InvariantName: edge.To})
					}
				}
			}

			require.Equal(t, test.RejectionLibraryLendingRejections, derived)
		})
	})
}

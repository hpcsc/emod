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
		Trigger:  &ast.Trigger{Name: "Lending Desk", Reads: []string{"AvailableCopiesView"}},
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

		t.Run("a trigger reading two views derives one reads edge from each, in the order of its list", func(t *testing.T) {
			slice := &ast.Slice{
				Name:    "Apply Discount",
				Trigger: &ast.Trigger{Name: "Checkout Page", Reads: []string{"DiscountView", "CartView"}},
			}

			require.Equal(t, []diagram.Edge{
				{Kind: diagram.EdgeTriggerReads, From: "DiscountView", To: "Checkout Page"},
				{Kind: diagram.EdgeTriggerReads, From: "CartView", To: "Checkout Page"},
			}, diagram.SliceEdges(slice))
		})

		t.Run("each shows block derives a reads edge into its screen for each view and a command edge out of it for each command, in the order of its lists", func(t *testing.T) {
			slice := &ast.Slice{
				Name: "Chase Overdue Copy",
				Appearances: []*ast.Appearance{
					{Screen: "OverdueReport", Reads: []string{"MemberLoansView", "DeskOccupancyView"}, Commands: []string{"RecallCopy", "RemindMember"}},
					{Screen: "DeskKiosk", Reads: []string{"DeskOccupancyView"}},
				},
			}

			require.Equal(t, []diagram.Edge{
				{Kind: diagram.EdgeScreenReads, From: "MemberLoansView", To: "OverdueReport"},
				{Kind: diagram.EdgeScreenReads, From: "DeskOccupancyView", To: "OverdueReport"},
				{Kind: diagram.EdgeScreenCommand, From: "OverdueReport", To: "RecallCopy"},
				{Kind: diagram.EdgeScreenCommand, From: "OverdueReport", To: "RemindMember"},
				{Kind: diagram.EdgeScreenReads, From: "DeskOccupancyView", To: "DeskKiosk"},
			}, diagram.SliceEdges(slice))
		})

		t.Run("the shared screen fixture derives one edge for each view and each command that its shows blocks name", func(t *testing.T) {
			model := test.ScreenLibraryLendingModel(t)

			var reads, commands []string
			for _, ref := range model.SliceRefs() {
				for _, edge := range diagram.SliceEdges(ref.Slice) {
					switch edge.Kind {
					case diagram.EdgeScreenReads:
						reads = append(reads, edge.From)
					case diagram.EdgeScreenCommand:
						commands = append(commands, edge.To)
					}
				}
			}

			require.Equal(t, test.ScreenLibraryLendingAppearanceReads, reads)
			require.Equal(t, test.ScreenLibraryLendingAppearanceCommands, commands)
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

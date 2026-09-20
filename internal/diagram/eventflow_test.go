//go:build unit

package diagram_test

import (
	"testing"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/diagram"
	"github.com/hpcsc/emod/internal/test"
	"github.com/stretchr/testify/require"
)

// shapes states what each node draws, leaving the prose out, so a whole-list
// comparison reads as the picture rather than as the model it came from.
type shape struct {
	ID      string
	Kind    diagram.NodeKind
	Label   string
	Caption string
	DeadEnd bool
}

func shapes(flow diagram.EventFlow) []shape {
	drawn := make([]shape, 0, len(flow.Nodes))
	for _, node := range flow.Nodes {
		drawn = append(drawn, shape{
			ID:      node.ID,
			Kind:    node.Kind,
			Label:   node.Label,
			Caption: node.Caption,
			DeadEnd: node.DeadEnd,
		})
	}

	return drawn
}

func nodeByID(t *testing.T, flow diagram.EventFlow, id string) diagram.Node {
	t.Helper()

	for _, node := range flow.Nodes {
		if node.ID == id {
			return node
		}
	}
	require.Failf(t, "node not drawn", "no node has id %q", id)

	return diagram.Node{}
}

func TestCollapseEventFlow(t *testing.T) {
	t.Run("nodes", func(t *testing.T) {
		t.Run("every event the model declares gets a node of its own", func(t *testing.T) {
			model := singleSliceModel("Lending", "Borrow Copy", event("CopyBorrowed"), event("LoanOpened"))

			flow := diagram.CollapseEventFlow(model)

			require.Equal(t, []shape{
				{ID: "CopyBorrowed", Kind: diagram.NodeEvent, Label: "CopyBorrowed", DeadEnd: true},
				{ID: "LoanOpened", Kind: diagram.NodeEvent, Label: "LoanOpened", DeadEnd: true},
			}, shapes(flow))
		})

		t.Run("an automation and the reactor a translation declares are both automations", func(t *testing.T) {
			model := translationModel(false)
			slice := model.Contexts[0].Aggregates[0].Slices[0]
			slice.Automations = []*ast.Automation{{Name: "ReceiptSender", OnEvent: "Charged", Command: "SendReceipt"}}

			flow := diagram.CollapseEventFlow(model)

			require.Equal(t, diagram.NodeAutomation, nodeByID(t, flow, "ReceiptSender").Kind)
			require.Equal(t, diagram.NodeAutomation, nodeByID(t, flow, "StripeWebhook").Kind)
		})

		t.Run("a schedule-woken automation carries its cadence under its name", func(t *testing.T) {
			model := singleSliceModel("Lending", "Sweep", command("RecallCopy"), event("CopyRecalled"),
				&ast.Automation{Name: "RecallOverdueCopy", Schedule: "15m", Command: "RecallCopy"})

			flow := diagram.CollapseEventFlow(model)

			require.Equal(t, `every "15m"`, nodeByID(t, flow, "RecallOverdueCopy").Caption)
		})

		t.Run("an event-woken automation carries no caption", func(t *testing.T) {
			model := singleSliceModel("Lending", "Chase", command("RemindMember"), event("MemberReminded"),
				&ast.Automation{Name: "RemindOnDueDate", OnEvent: "CopyBorrowed", Command: "RemindMember"})

			flow := diagram.CollapseEventFlow(model)

			require.Empty(t, nodeByID(t, flow, "RemindOnDueDate").Caption)
		})

		t.Run("a trigger is an origin captioned by the actor who works it", func(t *testing.T) {
			model := singleSliceModel("Lending", "Borrow Copy",
				&ast.Trigger{Name: "Lending Desk", Actor: "Member"}, command("BorrowCopy"), event("CopyBorrowed"))
			model.Contexts[0].Aggregates[0].Slices[0].Flows = []*ast.Flow{
				{CommandName: "BorrowCopy", EventName: "CopyBorrowed"},
			}

			flow := diagram.CollapseEventFlow(model)

			require.Equal(t, diagram.Node{
				ID:      "Ctx.Lending Desk",
				Kind:    diagram.NodeOrigin,
				Label:   "Lending Desk",
				Caption: "Member",
			}, nodeByID(t, flow, "Ctx.Lending Desk"))
		})

		t.Run("a trigger naming no actor is captioned as a trigger", func(t *testing.T) {
			model := singleSliceModel("Lending", "Borrow Copy",
				&ast.Trigger{Name: "Lending Desk"}, command("BorrowCopy"), event("CopyBorrowed"))
			model.Contexts[0].Aggregates[0].Slices[0].Flows = []*ast.Flow{
				{CommandName: "BorrowCopy", EventName: "CopyBorrowed"},
			}

			flow := diagram.CollapseEventFlow(model)

			require.Equal(t, "trigger", nodeByID(t, flow, "Ctx.Lending Desk").Caption)
		})

		t.Run("an external system is an origin captioned as one", func(t *testing.T) {
			flow := diagram.CollapseEventFlow(translationModel(false))

			require.Equal(t, diagram.Node{
				ID:      "Stripe",
				Kind:    diagram.NodeOrigin,
				Label:   "Stripe",
				Caption: "external system",
			}, nodeByID(t, flow, "Stripe"))
		})

		t.Run("a command nothing issues stands as the origin of what it emits", func(t *testing.T) {
			model := singleSliceModel("Lending", "Return Copy", command("ReturnCopy"), event("CopyReturned"))
			model.Contexts[0].Aggregates[0].Slices[0].Flows = []*ast.Flow{
				{CommandName: "ReturnCopy", EventName: "CopyReturned"},
			}

			flow := diagram.CollapseEventFlow(model)

			require.Equal(t, diagram.Node{
				ID:      "ReturnCopy",
				Kind:    diagram.NodeOrigin,
				Label:   "ReturnCopy",
				Caption: "command",
			}, nodeByID(t, flow, "ReturnCopy"))
		})

		t.Run("an invariant that refuses a command is a node of its own", func(t *testing.T) {
			model := singleSliceModel("Lending", "Borrow Copy", command("BorrowCopy"), event("CopyBorrowed"))
			slice := model.Contexts[0].Aggregates[0].Slices[0]
			slice.Flows = []*ast.Flow{{CommandName: "BorrowCopy", EventName: "CopyBorrowed"}}
			slice.Rejections = []*ast.Rejection{{CommandName: "BorrowCopy", InvariantName: "OneCopyPerLoan"}}
			model.Contexts[0].Aggregates[0].Invariants = []*ast.Invariant{
				{Name: "OneCopyPerLoan", Statement: "A loan covers exactly one copy"},
			}

			flow := diagram.CollapseEventFlow(model)

			require.Equal(t, diagram.Node{
				ID:          "Ctx.OneCopyPerLoan",
				Kind:        diagram.NodeRejection,
				Label:       "OneCopyPerLoan",
				Description: "A loan covers exactly one copy",
			}, nodeByID(t, flow, "Ctx.OneCopyPerLoan"))
		})

		t.Run("two contexts declaring one invariant name get a node each", func(t *testing.T) {
			model := test.EveryConstructLibraryLendingModel(t)
			renamed := model.Contexts[1]
			renamed.Invariants[0].Name = "OneCopyPerLoan"
			for _, ref := range renamed.SliceRefs() {
				for _, rejection := range ref.Slice.Rejections {
					rejection.InvariantName = "OneCopyPerLoan"
				}
			}

			flow := diagram.CollapseEventFlow(model)

			require.Equal(t, "OneCopyPerLoan", nodeByID(t, flow, "Lending.OneCopyPerLoan").Label)
			require.Equal(t, "OneCopyPerLoan", nodeByID(t, flow, "Reading Room.OneCopyPerLoan").Label)
		})

		t.Run("the prose a construct states reaches the node drawn for it", func(t *testing.T) {
			model := test.EveryConstructLibraryLendingModel(t)

			flow := diagram.CollapseEventFlow(model)

			require.Equal(t, "A copy left the shelf with a member", nodeByID(t, flow, "CopyBorrowed").Description)
			require.Equal(t, "Waits out the grace period, then nudges", nodeByID(t, flow, "RemindOnDueDate").Description)
			require.Equal(t, "The counter a member borrows from", nodeByID(t, flow, "Lending.Lending Desk").Description)
		})
	})

	t.Run("hops", func(t *testing.T) {
		t.Run("an automation reaches the events its command emits, with the command left out", func(t *testing.T) {
			model := singleSliceModel("Lending", "Chase", command("RemindMember"), event("MemberReminded"),
				&ast.Automation{Name: "RemindOnDueDate", OnEvent: "CopyBorrowed", Command: "RemindMember"})
			model.Contexts[0].Aggregates[0].Slices[0].Flows = []*ast.Flow{
				{CommandName: "RemindMember", EventName: "MemberReminded"},
			}

			flow := diagram.CollapseEventFlow(model)

			require.Equal(t, []diagram.Hop{
				{From: "RemindOnDueDate", To: "MemberReminded"},
			}, flow.Hops)
			require.NotContains(t, shapes(flow), shape{ID: "RemindMember", Kind: diagram.NodeOrigin,
				Label: "RemindMember", Caption: "command"})
		})

		t.Run("the arrow waking an automation carries the delay it waits out", func(t *testing.T) {
			model := test.EveryConstructLibraryLendingModel(t)

			flow := diagram.CollapseEventFlow(model)

			require.Contains(t, flow.Hops, diagram.Hop{From: "CopyBorrowed", To: "RemindOnDueDate", Label: `after "72h"`})
		})

		t.Run("a schedule-woken automation is reached by no arrow", func(t *testing.T) {
			model := singleSliceModel("Lending", "Sweep", command("RecallCopy"), event("CopyRecalled"),
				&ast.Automation{Name: "RecallOverdueCopy", Schedule: "15m", Command: "RecallCopy"})
			model.Contexts[0].Aggregates[0].Slices[0].Flows = []*ast.Flow{
				{CommandName: "RecallCopy", EventName: "CopyRecalled"},
			}

			flow := diagram.CollapseEventFlow(model)

			require.Equal(t, []diagram.Hop{
				{From: "RecallOverdueCopy", To: "CopyRecalled"},
			}, flow.Hops)
		})

		t.Run("an external system reaches its reactor, and the reactor the event it nests", func(t *testing.T) {
			flow := diagram.CollapseEventFlow(translationModel(false))

			require.Equal(t, []diagram.Hop{
				{From: "Stripe", To: "StripeWebhook"},
				{From: "StripeWebhook", To: "Charged"},
			}, flow.Hops)
		})

		t.Run("a translation stating its flow draws the same arrows as one leaving it implied", func(t *testing.T) {
			implied := diagram.CollapseEventFlow(translationModel(false))
			stated := diagram.CollapseEventFlow(translationModel(true))

			require.Equal(t, implied.Hops, stated.Hops)
		})

		t.Run("a refused command reaches the invariant that refuses it", func(t *testing.T) {
			model := test.EveryConstructLibraryLendingModel(t)

			flow := diagram.CollapseEventFlow(model)

			require.Contains(t, flow.Hops, diagram.Hop{From: "Lending.Lending Desk", To: "Lending.OneCopyPerLoan"})
		})

		t.Run("an arrow to an event no slice declares is left out", func(t *testing.T) {
			model := singleSliceModel("Lending", "Borrow Copy", command("BorrowCopy"), event("CopyBorrowed"))
			model.Contexts[0].Aggregates[0].Slices[0].Flows = []*ast.Flow{
				{CommandName: "BorrowCopy", EventName: "CopyBorrowed"},
				{CommandName: "BorrowCopy", EventName: "CopyMislaid"},
			}

			flow := diagram.CollapseEventFlow(model)

			require.Equal(t, []diagram.Hop{
				{From: "BorrowCopy", To: "CopyBorrowed"},
			}, flow.Hops)
		})
	})

	t.Run("dead ends", func(t *testing.T) {
		t.Run("an event nothing reads is marked", func(t *testing.T) {
			model := singleSliceModel("Lending", "Borrow Copy", event("CopyBorrowed"))

			flow := diagram.CollapseEventFlow(model)

			require.True(t, nodeByID(t, flow, "CopyBorrowed").DeadEnd)
		})

		t.Run("an event a view subscribes to is not marked", func(t *testing.T) {
			model := singleSliceModel("Lending", "Borrow Copy", event("CopyBorrowed"),
				&ast.View{Name: "MemberLoansView", Subscribes: []string{"CopyBorrowed"}})

			flow := diagram.CollapseEventFlow(model)

			require.False(t, nodeByID(t, flow, "CopyBorrowed").DeadEnd)
		})

		t.Run("an event an automation activates on is not marked", func(t *testing.T) {
			model := singleSliceModel("Lending", "Borrow Copy", event("CopyBorrowed"),
				&ast.Automation{Name: "RemindOnDueDate", OnEvent: "CopyBorrowed", Command: "RemindMember"})

			flow := diagram.CollapseEventFlow(model)

			require.False(t, nodeByID(t, flow, "CopyBorrowed").DeadEnd)
		})

		t.Run("an event a command decides on is not marked", func(t *testing.T) {
			model := singleSliceModel("Reading Room", "Claim Desk", event("DeskClaimed"),
				&ast.Command{Name: "ClaimDesk", DecidesOn: &ast.DecidesOnClause{Events: []string{"DeskClaimed"}}})

			flow := diagram.CollapseEventFlow(model)

			require.False(t, nodeByID(t, flow, "DeskClaimed").DeadEnd)
		})
	})

	t.Run("whole model", func(t *testing.T) {
		t.Run("a model stating every construct collapses to its events, automations and origins", func(t *testing.T) {
			model := test.EveryConstructLibraryLendingModel(t)

			flow := diagram.CollapseEventFlow(model)

			require.Equal(t, []shape{
				{ID: "CopyBorrowed", Kind: diagram.NodeEvent, Label: "CopyBorrowed"},
				{ID: "LoanOpened", Kind: diagram.NodeEvent, Label: "LoanOpened", DeadEnd: true},
				{ID: "Lending.Lending Desk", Kind: diagram.NodeOrigin, Label: "Lending Desk", Caption: "Member"},
				{ID: "Lending.OneCopyPerLoan", Kind: diagram.NodeRejection, Label: "OneCopyPerLoan"},
				{ID: "RemindOnDueDate", Kind: diagram.NodeAutomation, Label: "RemindOnDueDate"},
				{ID: "MemberReminded", Kind: diagram.NodeEvent, Label: "MemberReminded", DeadEnd: true},
				{ID: "RecallOverdueCopy", Kind: diagram.NodeAutomation, Label: "RecallOverdueCopy", Caption: `every "15m"`},
				{ID: "CopyRecalled", Kind: diagram.NodeEvent, Label: "CopyRecalled", DeadEnd: true},
				{ID: "CopyReturned", Kind: diagram.NodeEvent, Label: "CopyReturned"},
				{ID: "ReturnCopy", Kind: diagram.NodeOrigin, Label: "ReturnCopy", Caption: "command"},
				{ID: "DeskClaimed", Kind: diagram.NodeEvent, Label: "DeskClaimed"},
				{ID: "ClaimDesk", Kind: diagram.NodeOrigin, Label: "ClaimDesk", Caption: "command"},
				{ID: "Reading Room.OneReaderPerDesk", Kind: diagram.NodeRejection, Label: "OneReaderPerDesk"},
				{ID: "ExternalDeskBookingImport", Kind: diagram.NodeAutomation, Label: "ExternalDeskBookingImport"},
				{ID: "Room Booking API", Kind: diagram.NodeOrigin, Label: "Room Booking API", Caption: "external system"},
				{ID: "ExternalDeskBookingImported", Kind: diagram.NodeEvent, Label: "ExternalDeskBookingImported", DeadEnd: true},
			}, shapes(flow))

			require.Equal(t, []diagram.Hop{
				{From: "Lending.Lending Desk", To: "CopyBorrowed"},
				{From: "Lending.Lending Desk", To: "LoanOpened"},
				{From: "Lending.Lending Desk", To: "Lending.OneCopyPerLoan"},
				{From: "CopyBorrowed", To: "RemindOnDueDate", Label: `after "72h"`},
				{From: "RemindOnDueDate", To: "MemberReminded"},
				{From: "RecallOverdueCopy", To: "CopyRecalled"},
				{From: "ReturnCopy", To: "CopyReturned"},
				{From: "ClaimDesk", To: "DeskClaimed"},
				{From: "ClaimDesk", To: "Reading Room.OneReaderPerDesk"},
				{From: "Room Booking API", To: "ExternalDeskBookingImport"},
				{From: "ExternalDeskBookingImport", To: "ExternalDeskBookingImported"},
			}, flow.Hops)
		})

		t.Run("a nil model collapses to an empty flow", func(t *testing.T) {
			flow := diagram.CollapseEventFlow(nil)

			require.Empty(t, flow.Nodes)
			require.Empty(t, flow.Hops)
		})
	})
}

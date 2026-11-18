//go:build unit

package diagram_test

import (
	"testing"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/diagram"
	"github.com/hpcsc/emod/internal/test"
	"github.com/stretchr/testify/require"
)

func TestElementCopies(t *testing.T) {
	t.Run("copy a subscribed event", func(t *testing.T) {
		t.Run("a view copies each event it subscribes to from two or more slices away, and none from the slice next to it", func(t *testing.T) {
			model := test.SubscribedEventsHotelStaysModel(t)
			copies := copiesBySliceName(model)

			require.Equal(t, []diagram.DeclaredElement{
				{Kind: diagram.ElementEvent, Name: "RoomReserved", Slice: sliceNamed(t, model, "Reserve Room")},
				{Kind: diagram.ElementEvent, Name: "GuestCheckedIn", Slice: sliceNamed(t, model, "Check In")},
				{Kind: diagram.ElementEvent, Name: "BreakfastOrdered", Slice: sliceNamed(t, model, "Import Breakfast Orders")},
			}, copies["See Stay History"])
		})

		t.Run("a view that subscribes to only three events copies each of them that is two or more slices away", func(t *testing.T) {
			model := test.SubscribedEventsHotelStaysModel(t)
			copies := copiesBySliceName(model)

			require.Equal(t, []diagram.DeclaredElement{
				{Kind: diagram.ElementEvent, Name: "RoomReserved", Slice: sliceNamed(t, model, "Reserve Room")},
				{Kind: diagram.ElementEvent, Name: "GuestCheckedIn", Slice: sliceNamed(t, model, "Check In")},
				{Kind: diagram.ElementEvent, Name: "GuestCheckedOut", Slice: sliceNamed(t, model, "Check Out")},
			}, copies["See Occupancy"])
		})

		t.Run("a view in a DCB context copies as one under an aggregate does", func(t *testing.T) {
			model := test.SubscribedEventsHotelStaysModel(t)
			copies := copiesBySliceName(model)

			require.Equal(t, []diagram.DeclaredElement{
				{Kind: diagram.ElementEvent, Name: "RoomReserved", Slice: sliceNamed(t, model, "Reserve Room")},
				{Kind: diagram.ElementEvent, Name: "GuestCheckedIn", Slice: sliceNamed(t, model, "Check In")},
				{Kind: diagram.ElementEvent, Name: "GuestCheckedOut", Slice: sliceNamed(t, model, "Check Out")},
			}, copies["Plan Cleaning"])
		})

		t.Run("the last slice of one context and the first of the next are next to each other", func(t *testing.T) {
			model := &ast.Model{
				Name: "Copies",
				Contexts: []*ast.Context{
					{Name: "Before", Aggregates: []*ast.Aggregate{{Name: "Agg", Slices: []*ast.Slice{
						declaringSlice("Far"), declaringSlice("Near"),
					}}}},
					{Name: "After", Aggregates: []*ast.Aggregate{{Name: "Agg", Slices: []*ast.Slice{
						{Name: "Watch", Views: []*ast.View{{Name: "WatchView", Subscribes: []string{"Far", "Near"}}}},
					}}}},
				},
			}

			require.Equal(t, map[string][]string{"Watch": {"Far"}}, namesCopiedBySliceName(model))
		})

		t.Run("an event the view's own slice declares is not copied, although another slice two away declares it first", func(t *testing.T) {
			model := slicesModel(
				declaringSlice("Own"), declaringSlice("Far"), declaringSlice("Between"),
				&ast.Slice{
					Name:   "Watch",
					Events: []*ast.Event{event("Own")},
					Views:  []*ast.View{{Name: "WatchView", Subscribes: []string{"Own", "Far"}}},
				},
			)

			require.Equal(t, map[string][]string{"Watch": {"Far"}}, namesCopiedBySliceName(model))
		})

		t.Run("an event no slice declares is not copied", func(t *testing.T) {
			model := slicesModel(
				declaringSlice("Far"), declaringSlice("Between"),
				&ast.Slice{Name: "Watch", Views: []*ast.View{{Name: "WatchView", Subscribes: []string{"Undeclared", "Far"}}}},
			)

			require.Equal(t, map[string][]string{"Watch": {"Far"}}, namesCopiedBySliceName(model))
		})

		t.Run("an event a view names twice is copied once", func(t *testing.T) {
			model := slicesModel(
				declaringSlice("Far"), declaringSlice("Between"),
				&ast.Slice{Name: "Watch", Views: []*ast.View{{Name: "WatchView", Subscribes: []string{"Far", "Far"}}}},
			)

			require.Equal(t, map[string][]string{"Watch": {"Far"}}, namesCopiedBySliceName(model))
		})

		t.Run("two views in one slice share one copy of an event they both subscribe to", func(t *testing.T) {
			model := slicesModel(
				declaringSlice("First"), declaringSlice("Second"), declaringSlice("Third"), declaringSlice("Between"),
				&ast.Slice{
					Name: "Watch",
					Views: []*ast.View{
						{Name: "WatchView", Subscribes: []string{"First", "Second"}},
						{Name: "AuditView", Subscribes: []string{"Second", "Third"}},
					},
				},
			)

			require.Equal(t, map[string][]string{"Watch": {"First", "Second", "Third"}}, namesCopiedBySliceName(model))
		})

		t.Run("an event a later slice declares is copied from two or more slices away, and not from the slice next to the view's", func(t *testing.T) {
			model := slicesModel(
				declaringSlice("Earlier"),
				&ast.Slice{Name: "Watch", Views: []*ast.View{{Name: "WatchView", Subscribes: []string{"Next", "Far"}}}},
				declaringSlice("Next"), declaringSlice("Between"), declaringSlice("Far"),
			)

			require.Equal(t, map[string][]string{"Watch": {"Far"}}, namesCopiedBySliceName(model))
		})

		t.Run("each event a slice declares is copied from it, not only its first", func(t *testing.T) {
			model := slicesModel(
				&ast.Slice{Name: "Declares Two", Events: []*ast.Event{event("First"), event("Second")}},
				declaringSlice("Between"),
				&ast.Slice{Name: "Watch", Views: []*ast.View{{Name: "WatchView", Subscribes: []string{"Second"}}}},
			)

			require.Equal(t, []diagram.DeclaredElement{
				{Kind: diagram.ElementEvent, Name: "Second", Slice: sliceNamed(t, model, "Declares Two")},
			}, copiesBySliceName(model)["Watch"])
		})

		t.Run("the event of each translation of a slice is copied from it, not only the first translation's", func(t *testing.T) {
			model := slicesModel(
				&ast.Slice{Name: "Import", Translations: []*ast.Translation{
					{Name: "FirstImport", ExternalSystem: "Partner", Event: event("FirstImported")},
					{Name: "SecondImport", ExternalSystem: "Partner", Event: event("SecondImported")},
				}},
				declaringSlice("Between"),
				&ast.Slice{Name: "Watch", Views: []*ast.View{{Name: "WatchView", Subscribes: []string{"SecondImported"}}}},
			)

			require.Equal(t, []diagram.DeclaredElement{
				{Kind: diagram.ElementEvent, Name: "SecondImported", Slice: sliceNamed(t, model, "Import")},
			}, copiesBySliceName(model)["Watch"])
		})

		t.Run("a translation the parser left empty declares no event, and the other translations of its slice still do", func(t *testing.T) {
			model := slicesModel(
				&ast.Slice{Name: "Import", Translations: []*ast.Translation{
					nil,
					{Name: "SecondImport", ExternalSystem: "Partner", Event: event("SecondImported")},
				}},
				declaringSlice("Between"),
				&ast.Slice{Name: "Watch", Views: []*ast.View{{Name: "WatchView", Subscribes: []string{"SecondImported"}}}},
			)

			require.Equal(t, map[string][]string{"Watch": {"SecondImported"}}, namesCopiedBySliceName(model))
		})

		t.Run("an arrow of a kind no copy stands in for draws no copy, although a slice two away declares an event with no name", func(t *testing.T) {
			model := slicesModel(
				declaringSlice("Far"),
				&ast.Slice{Name: "Unnamed", Events: []*ast.Event{event("")}},
				declaringSlice("Between"),
				&ast.Slice{
					Name:     "Order",
					Commands: []*ast.Command{{Name: "PlaceOrder"}},
					Events:   []*ast.Event{event("OrderPlaced")},
					Views:    []*ast.View{{Name: "OrderView", Subscribes: []string{"Far"}}},
					Flows:    []*ast.Flow{{CommandName: "PlaceOrder", EventName: "OrderPlaced"}},
				},
			)

			require.Equal(t, map[string][]string{"Order": {"Far"}}, namesCopiedBySliceName(model))
		})

		t.Run("an event two slices declare is copied from the first of them", func(t *testing.T) {
			model := slicesModel(
				&ast.Slice{Name: "Declares Fourth", Events: []*ast.Event{event("Fourth")}},
				&ast.Slice{Name: "Declares Fourth Again", Events: []*ast.Event{event("Fourth")}},
				declaringSlice("Between"),
				&ast.Slice{Name: "Watch", Views: []*ast.View{{Name: "WatchView", Subscribes: []string{"Fourth"}}}},
			)

			require.Equal(t, []diagram.DeclaredElement{
				{Kind: diagram.ElementEvent, Name: "Fourth", Slice: sliceNamed(t, model, "Declares Fourth")},
			}, copiesBySliceName(model)["Watch"])
		})
	})

	t.Run("copy an activating event", func(t *testing.T) {
		t.Run("an automation copies the event it activates on from two or more slices away, and none its own slice declares", func(t *testing.T) {
			model := test.AutomationDelayLibraryLendingModel(t)
			copies := copiesBySliceName(model)

			require.Equal(t, map[string][]diagram.DeclaredElement{
				"Chase Overdue Copy": {
					{Kind: diagram.ElementEvent, Name: "CopyBorrowed", Slice: sliceNamed(t, model, "Borrow Copy")},
				},
				"Close Reading Room": {
					{Kind: diagram.ElementEvent, Name: "DeskClaimed", Slice: sliceNamed(t, model, "Claim Desk")},
					{Kind: diagram.ElementEvent, Name: "DeskReleased", Slice: sliceNamed(t, model, "Release Desk")},
				},
			}, map[string][]diagram.DeclaredElement{
				"Chase Overdue Copy": copiesOfKind(copies["Chase Overdue Copy"], diagram.ElementEvent),
				"Close Reading Room": copiesOfKind(copies["Close Reading Room"], diagram.ElementEvent),
			})
		})

		t.Run("an automation's activating event in the slice next to it is not copied", func(t *testing.T) {
			model := slicesModel(
				declaringSlice("Far"), declaringSlice("Near"),
				&ast.Slice{Name: "React", Automations: []*ast.Automation{
					{Name: "OnNear", OnEvent: "Near"},
					{Name: "OnFar", OnEvent: "Far"},
				}},
			)

			require.Equal(t, map[string][]string{"React": {"Far"}}, namesCopiedBySliceName(model))
		})

		t.Run("an event that a view and an automation in one slice both read is copied once", func(t *testing.T) {
			model := slicesModel(
				declaringSlice("Far"), declaringSlice("Between"),
				&ast.Slice{
					Name:        "React",
					Views:       []*ast.View{{Name: "TodoView", Subscribes: []string{"Far"}}},
					Automations: []*ast.Automation{{Name: "OnFar", OnEvent: "Far"}},
				},
			)

			require.Equal(t, map[string][]string{"React": {"Far"}}, namesCopiedBySliceName(model))
		})
	})

	t.Run("copy a read view and an issued command", func(t *testing.T) {
		t.Run("a trigger and an automation copy the view they read from two or more slices away, and none from the slice next to them", func(t *testing.T) {
			model := test.TriggerReadsLibraryLendingModel(t)

			require.Equal(t, map[string][]diagram.DeclaredElement{
				"Chase Overdue Copy": {
					{Kind: diagram.ElementView, Name: "DeskOccupancyView", Slice: sliceNamed(t, model, "Browse Desk Occupancy")},
					{Kind: diagram.ElementView, Name: "MemberLoansView", Slice: sliceNamed(t, model, "Review Member Loans")},
				},
				"Claim Desk": {
					{Kind: diagram.ElementView, Name: "DeskOccupancyView", Slice: sliceNamed(t, model, "Browse Desk Occupancy")},
				},
				"Close Reading Room": {
					{Kind: diagram.ElementView, Name: "MemberLoansView", Slice: sliceNamed(t, model, "Review Member Loans")},
				},
			}, copiesOfKindBySliceName(model, diagram.ElementView))
		})

		t.Run("an automation copies the command it issues from two or more slices away, and none its own slice declares", func(t *testing.T) {
			model := test.TriggerReadsLibraryLendingModel(t)

			require.Equal(t, map[string][]diagram.DeclaredElement{
				"Close Reading Room": {
					{Kind: diagram.ElementCommand, Name: "ReleaseDesk", Slice: sliceNamed(t, model, "Release Desk")},
					{Kind: diagram.ElementCommand, Name: "RemindMember", Slice: sliceNamed(t, model, "Chase Overdue Copy")},
				},
			}, copiesOfKindBySliceName(model, diagram.ElementCommand))
		})

		t.Run("a shows block copies the view it reads from two or more slices away, and none from the slice next to it", func(t *testing.T) {
			model := test.ScreenLibraryLendingModel(t)

			require.Equal(t, map[string][]diagram.DeclaredElement{
				"Chase Overdue Copy": {
					{Kind: diagram.ElementView, Name: "MemberLoansView", Slice: sliceNamed(t, model, "Review Member Loans")},
					{Kind: diagram.ElementView, Name: "DeskOccupancyView", Slice: sliceNamed(t, model, "Browse Desk Occupancy")},
				},
			}, copiesOfKindBySliceName(model, diagram.ElementView))
		})

		t.Run("a shows block copies the command it names from two or more slices away, and none from the slice next to it", func(t *testing.T) {
			model := slicesModel(
				&ast.Slice{Name: "Cancel Order", Appearances: []*ast.Appearance{{Screen: "OrderPage", Commands: []string{"RefundOrder", "TrackOrder"}}}},
				&ast.Slice{Name: "Track Order", Commands: []*ast.Command{{Name: "TrackOrder"}}},
				&ast.Slice{Name: "Refund Order", Commands: []*ast.Command{{Name: "RefundOrder"}}},
			)

			require.Equal(t, map[string][]diagram.DeclaredElement{
				"Cancel Order": {
					{Kind: diagram.ElementCommand, Name: "RefundOrder", Slice: sliceNamed(t, model, "Refund Order")},
				},
			}, copiesOfKindBySliceName(model, diagram.ElementCommand))
		})

		t.Run("a translation copies the view it reads and the command it issues from two or more slices away", func(t *testing.T) {
			model := farTranslationModel()

			require.Equal(t, []diagram.DeclaredElement{
				{Kind: diagram.ElementView, Name: "PartnersView", Slice: sliceNamed(t, model, "Read Partners")},
				{Kind: diagram.ElementCommand, Name: "ImportBooking", Slice: sliceNamed(t, model, "Record Booking")},
			}, copiesBySliceName(model)["Import Booking"])
		})

		t.Run("a view or a command in the slice next to the reader's is not copied", func(t *testing.T) {
			model := slicesModel(
				&ast.Slice{Name: "Far", Views: []*ast.View{{Name: "FarView"}}},
				&ast.Slice{Name: "Near", Views: []*ast.View{{Name: "NearView"}}},
				&ast.Slice{Name: "Automate", Automations: []*ast.Automation{
					{Name: "ActOnNear", Reads: "NearView", Command: "NearCommand"},
					{Name: "ActOnFar", Reads: "FarView"},
				}},
				&ast.Slice{Name: "Write", Commands: []*ast.Command{{Name: "NearCommand"}}},
			)

			require.Equal(t, map[string][]string{"Automate": {"FarView"}}, namesCopiedBySliceName(model))
		})

		t.Run("each view and each command a slice declares is copied from it, not only its first", func(t *testing.T) {
			model := slicesModel(
				&ast.Slice{
					Name:     "Declares Two",
					Views:    []*ast.View{{Name: "FirstView"}, {Name: "SecondView"}},
					Commands: []*ast.Command{{Name: "FirstCommand"}, {Name: "SecondCommand"}},
				},
				&ast.Slice{Name: "Between"},
				&ast.Slice{Name: "Automate", Automations: []*ast.Automation{{Name: "Act", Reads: "SecondView", Command: "SecondCommand"}}},
			)

			require.Equal(t, []diagram.DeclaredElement{
				{Kind: diagram.ElementView, Name: "SecondView", Slice: sliceNamed(t, model, "Declares Two")},
				{Kind: diagram.ElementCommand, Name: "SecondCommand", Slice: sliceNamed(t, model, "Declares Two")},
			}, copiesBySliceName(model)["Automate"])
		})

		t.Run("an event and a view of one name are each copied", func(t *testing.T) {
			model := sharedNameModel()

			require.Equal(t, []diagram.DeclaredElement{
				{Kind: diagram.ElementEvent, Name: "Stock", Slice: sliceNamed(t, model, "Declare Event")},
				{Kind: diagram.ElementView, Name: "Stock", Slice: sliceNamed(t, model, "Declare View")},
			}, copiesBySliceName(model)["React"])
		})
	})

	t.Run("find the element an arrow joins", func(t *testing.T) {
		t.Run("an arrow into a view from a copied event joins the copy in the view's slice", func(t *testing.T) {
			model := test.SubscribedEventsHotelStaysModel(t)

			element, copied, found := diagram.NewElementCopies(model).Joined(sliceNamed(t, model, "See Stay History"),
				diagram.Edge{Kind: diagram.EdgeSubscription, From: "RoomReserved", To: "StayHistoryView"})

			require.Equal(t, []any{
				diagram.DeclaredElement{Kind: diagram.ElementEvent, Name: "RoomReserved", Slice: sliceNamed(t, model, "Reserve Room")}, true, true,
			}, []any{element, copied, found})
		})

		t.Run("an arrow into a view from an event in the slice next to it joins that event, and no copy", func(t *testing.T) {
			model := test.SubscribedEventsHotelStaysModel(t)

			element, copied, found := diagram.NewElementCopies(model).Joined(sliceNamed(t, model, "See Stay History"),
				diagram.Edge{Kind: diagram.EdgeSubscription, From: "GuestCheckedOut", To: "StayHistoryView"})

			require.Equal(t, []any{
				diagram.DeclaredElement{Kind: diagram.ElementEvent, Name: "GuestCheckedOut", Slice: sliceNamed(t, model, "Check Out")}, false, true,
			}, []any{element, copied, found})
		})

		t.Run("an arrow into an automation from a copied event joins the copy in the automation's slice", func(t *testing.T) {
			model := test.AutomationDelayLibraryLendingModel(t)

			element, copied, found := diagram.NewElementCopies(model).Joined(sliceNamed(t, model, "Chase Overdue Copy"),
				diagram.Edge{Kind: diagram.EdgeAutomationTrigger, From: "CopyBorrowed", To: "RemindAfterGracePeriod"})

			require.Equal(t, []any{
				diagram.DeclaredElement{Kind: diagram.ElementEvent, Name: "CopyBorrowed", Slice: sliceNamed(t, model, "Borrow Copy")}, true, true,
			}, []any{element, copied, found})
		})

		t.Run("a reads arrow from a copied view joins the copy in the reader's slice", func(t *testing.T) {
			model := test.TriggerReadsLibraryLendingModel(t)

			element, copied, found := diagram.NewElementCopies(model).Joined(sliceNamed(t, model, "Claim Desk"),
				diagram.Edge{Kind: diagram.EdgeTriggerReads, From: "DeskOccupancyView", To: "Desk Kiosk"})

			require.Equal(t, []any{
				diagram.DeclaredElement{Kind: diagram.ElementView, Name: "DeskOccupancyView", Slice: sliceNamed(t, model, "Browse Desk Occupancy")}, true, true,
			}, []any{element, copied, found})
		})

		t.Run("a reads arrow into a screen from a copied view joins the copy in the slice that shows the screen", func(t *testing.T) {
			model := test.ScreenLibraryLendingModel(t)

			element, copied, found := diagram.NewElementCopies(model).Joined(sliceNamed(t, model, "Chase Overdue Copy"),
				diagram.Edge{Kind: diagram.EdgeScreenReads, From: "DeskOccupancyView", To: "DeskKiosk"})

			require.Equal(t, []any{
				diagram.DeclaredElement{Kind: diagram.ElementView, Name: "DeskOccupancyView", Slice: sliceNamed(t, model, "Browse Desk Occupancy")}, true, true,
			}, []any{element, copied, found})
		})

		t.Run("a command arrow out of a screen joins the command that the slice showing the screen declares, not a copy", func(t *testing.T) {
			model := test.ScreenLibraryLendingModel(t)

			element, copied, found := diagram.NewElementCopies(model).Joined(sliceNamed(t, model, "Return Copy"),
				diagram.Edge{Kind: diagram.EdgeScreenCommand, From: "LendingDesk", To: "RenewLoan"})

			require.Equal(t, []any{
				diagram.DeclaredElement{Kind: diagram.ElementCommand, Name: "RenewLoan", Slice: sliceNamed(t, model, "Return Copy")}, false, true,
			}, []any{element, copied, found})
		})

		t.Run("a command arrow into a copied command joins the copy in the automation's slice", func(t *testing.T) {
			model := test.TriggerReadsLibraryLendingModel(t)

			element, copied, found := diagram.NewElementCopies(model).Joined(sliceNamed(t, model, "Close Reading Room"),
				diagram.Edge{Kind: diagram.EdgeAutomationCommand, From: "RemindReaderOfLoans", To: "RemindMember"})

			require.Equal(t, []any{
				diagram.DeclaredElement{Kind: diagram.ElementCommand, Name: "RemindMember", Slice: sliceNamed(t, model, "Chase Overdue Copy")}, true, true,
			}, []any{element, copied, found})
		})

		t.Run("the arrow a translation implies from its copied command joins the copy", func(t *testing.T) {
			model := farTranslationModel()

			element, copied, found := diagram.NewElementCopies(model).Joined(sliceNamed(t, model, "Import Booking"),
				diagram.Edge{Kind: diagram.EdgeTranslationFlow, From: "ImportBooking", To: "BookingImported"})

			require.Equal(t, []any{
				diagram.DeclaredElement{Kind: diagram.ElementCommand, Name: "ImportBooking", Slice: sliceNamed(t, model, "Record Booking")}, true, true,
			}, []any{element, copied, found})
		})

		t.Run("an arrow joins the copy of its own kind when an event and a view share a name", func(t *testing.T) {
			model := sharedNameModel()
			copies := diagram.NewElementCopies(model)
			react := sliceNamed(t, model, "React")

			fromEvent, eventCopied, _ := copies.Joined(react, diagram.Edge{Kind: diagram.EdgeAutomationTrigger, From: "Stock", To: "Restock"})
			fromView, viewCopied, _ := copies.Joined(react, diagram.Edge{Kind: diagram.EdgeAutomationReads, From: "Stock", To: "Restock"})

			require.Equal(t, []any{
				diagram.DeclaredElement{Kind: diagram.ElementEvent, Name: "Stock", Slice: sliceNamed(t, model, "Declare Event")}, true,
				diagram.DeclaredElement{Kind: diagram.ElementView, Name: "Stock", Slice: sliceNamed(t, model, "Declare View")}, true,
			}, []any{fromEvent, eventCopied, fromView, viewCopied})
		})

		t.Run("an arrow joins the element its own slice declares, though an earlier slice declares it too", func(t *testing.T) {
			model := slicesModel(
				declaringSlice("Own"), declaringSlice("Far"), declaringSlice("Between"),
				&ast.Slice{
					Name:   "Watch",
					Events: []*ast.Event{event("Own")},
					Views:  []*ast.View{{Name: "WatchView", Subscribes: []string{"Own", "Far"}}},
				},
			)
			watch := sliceNamed(t, model, "Watch")

			element, copied, found := diagram.NewElementCopies(model).Joined(watch,
				diagram.Edge{Kind: diagram.EdgeSubscription, From: "Own", To: "WatchView"})

			require.Equal(t, []any{diagram.DeclaredElement{Kind: diagram.ElementEvent, Name: "Own", Slice: watch}, false, true},
				[]any{element, copied, found})
		})

		t.Run("an arrow joins the element its own slice declares, though an earlier slice of the same name declares it too", func(t *testing.T) {
			first := &ast.Slice{Name: "Cancel", Events: []*ast.Event{event("OrderCancelled")}}
			second := &ast.Slice{
				Name:   "Cancel",
				Events: []*ast.Event{event("OrderCancelled")},
				Views:  []*ast.View{{Name: "CancelView", Subscribes: []string{"OrderCancelled"}}},
			}
			model := &ast.Model{
				Name: "Orders",
				Contexts: []*ast.Context{
					{Name: "Ordering", Aggregates: []*ast.Aggregate{{Name: "Order", Slices: []*ast.Slice{first, {Name: "A"}, {Name: "B"}}}}},
					{Name: "Refunds", Aggregates: []*ast.Aggregate{{Name: "Refund", Slices: []*ast.Slice{second}}}},
				},
			}

			element, copied, found := diagram.NewElementCopies(model).Joined(second,
				diagram.Edge{Kind: diagram.EdgeSubscription, From: "OrderCancelled", To: "CancelView"})

			require.Same(t, second, element.Slice)
			require.Equal(t, []any{diagram.ElementEvent, "OrderCancelled", false, true}, []any{element.Kind, element.Name, copied, found})
		})

		t.Run("an arrow joins the element the first slice that declares it declares, when its own slice does not", func(t *testing.T) {
			model := slicesModel(
				declaringSlice("Booked"),
				&ast.Slice{Name: "List", Views: []*ast.View{{Name: "RoomsView", Subscribes: []string{"Booked"}}}},
				&ast.Slice{Name: "Between"},
				&ast.Slice{Name: "Book Again", Events: []*ast.Event{event("Booked")}},
			)

			element, copied, found := diagram.NewElementCopies(model).Joined(sliceNamed(t, model, "List"),
				diagram.Edge{Kind: diagram.EdgeSubscription, From: "Booked", To: "RoomsView"})

			require.Equal(t, []any{
				diagram.DeclaredElement{Kind: diagram.ElementEvent, Name: "Booked", Slice: sliceNamed(t, model, "Declares Booked")}, false, true,
			}, []any{element, copied, found})
		})

		t.Run("an arrow that no copy can stand in for joins no element this way, although a slice declares an event with no name", func(t *testing.T) {
			model := slicesModel(
				&ast.Slice{Name: "Unnamed", Events: []*ast.Event{event("")}},
				&ast.Slice{
					Name:     "Place Order",
					Commands: []*ast.Command{{Name: "PlaceOrder"}},
					Events:   []*ast.Event{event("OrderPlaced")},
				},
			)

			element, copied, found := diagram.NewElementCopies(model).Joined(sliceNamed(t, model, "Place Order"),
				diagram.Edge{Kind: diagram.EdgeFlow, From: "PlaceOrder", To: "OrderPlaced"})

			require.Equal(t, []any{diagram.DeclaredElement{}, false, false}, []any{element, copied, found})
		})

		t.Run("an arrow from an event no slice declares joins no element", func(t *testing.T) {
			model := slicesModel(
				declaringSlice("Far"), declaringSlice("Between"),
				&ast.Slice{Name: "Watch", Views: []*ast.View{{Name: "WatchView", Subscribes: []string{"Undeclared", "Far"}}}},
			)

			element, copied, found := diagram.NewElementCopies(model).Joined(sliceNamed(t, model, "Watch"),
				diagram.Edge{Kind: diagram.EdgeSubscription, From: "Undeclared", To: "WatchView"})

			require.Equal(t, []any{diagram.DeclaredElement{}, false, false}, []any{element, copied, found})
		})
	})
}

// slicesModel declares the given slices, in order, under one aggregate.
func slicesModel(slices ...*ast.Slice) *ast.Model {
	return &ast.Model{
		Name: "Copies",
		Contexts: []*ast.Context{{
			Name:       "Ctx",
			Aggregates: []*ast.Aggregate{{Name: "Agg", Slices: slices}},
		}},
	}
}

// declaringSlice is a slice that declares the named event and nothing else, and
// is named after it.
func declaringSlice(eventName string) *ast.Slice {
	return &ast.Slice{Name: "Declares " + eventName, Events: []*ast.Event{event(eventName)}}
}

// farTranslationModel has a translation that reads a view three slices away and
// issues a command two slices away.
func farTranslationModel() *ast.Model {
	return slicesModel(
		&ast.Slice{Name: "Read Partners", Views: []*ast.View{{Name: "PartnersView"}}},
		&ast.Slice{Name: "Record Booking", Commands: []*ast.Command{{Name: "ImportBooking"}}},
		&ast.Slice{Name: "Between"},
		&ast.Slice{Name: "Import Booking", Translations: []*ast.Translation{{
			Name:           "PartnerFeed",
			ExternalSystem: "Partner API",
			Reads:          "PartnersView",
			Command:        "ImportBooking",
			Event:          event("BookingImported"),
		}}},
	)
}

// sharedNameModel has an automation that activates on the event Stock and reads
// the view Stock, each declared two or more slices away.
func sharedNameModel() *ast.Model {
	return slicesModel(
		&ast.Slice{Name: "Declare Event", Events: []*ast.Event{event("Stock")}},
		&ast.Slice{Name: "Declare View", Views: []*ast.View{{Name: "Stock"}}},
		&ast.Slice{Name: "Between"},
		&ast.Slice{Name: "React", Automations: []*ast.Automation{{Name: "Restock", OnEvent: "Stock", Reads: "Stock"}}},
	)
}

func sliceNamed(t *testing.T, model *ast.Model, name string) *ast.Slice {
	t.Helper()

	for _, s := range model.AllSlices() {
		if s.Name == name {
			return s
		}
	}
	require.Failf(t, "no slice", "the model declares no slice named %q", name)

	return nil
}

// copiesBySliceName states the copies of each slice that draws any.
func copiesBySliceName(model *ast.Model) map[string][]diagram.DeclaredElement {
	copies := diagram.NewElementCopies(model)
	byName := make(map[string][]diagram.DeclaredElement)
	for _, s := range model.AllSlices() {
		if in := copies.In(s); len(in) > 0 {
			byName[s.Name] = in
		}
	}

	return byName
}

func copiesOfKind(copies []diagram.DeclaredElement, kind diagram.ElementKind) []diagram.DeclaredElement {
	var ofKind []diagram.DeclaredElement
	for _, c := range copies {
		if c.Kind == kind {
			ofKind = append(ofKind, c)
		}
	}

	return ofKind
}

// copiesOfKindBySliceName states the copies of kind of each slice that draws
// any.
func copiesOfKindBySliceName(model *ast.Model, kind diagram.ElementKind) map[string][]diagram.DeclaredElement {
	byName := make(map[string][]diagram.DeclaredElement)
	for slice, copies := range copiesBySliceName(model) {
		if ofKind := copiesOfKind(copies, kind); len(ofKind) > 0 {
			byName[slice] = ofKind
		}
	}

	return byName
}

// namesCopiedBySliceName states the name of each copy of each slice that draws
// any.
func namesCopiedBySliceName(model *ast.Model) map[string][]string {
	byName := make(map[string][]string)
	for slice, copies := range copiesBySliceName(model) {
		for _, c := range copies {
			byName[slice] = append(byName[slice], c.Name)
		}
	}

	return byName
}

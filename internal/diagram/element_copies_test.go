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
			copies := copiesBySliceName(test.SubscribedEventsHotelStaysModel(t))

			require.Equal(t, []diagram.ElementCopy{
				{Kind: diagram.CopyOfEvent, Name: "RoomReserved", DeclaredIn: "Reserve Room"},
				{Kind: diagram.CopyOfEvent, Name: "GuestCheckedIn", DeclaredIn: "Check In"},
				{Kind: diagram.CopyOfEvent, Name: "BreakfastOrdered", DeclaredIn: "Import Breakfast Orders"},
			}, copies["See Stay History"])
		})

		t.Run("a view that subscribes to only three events copies each of them that is two or more slices away", func(t *testing.T) {
			copies := copiesBySliceName(test.SubscribedEventsHotelStaysModel(t))

			require.Equal(t, []diagram.ElementCopy{
				{Kind: diagram.CopyOfEvent, Name: "RoomReserved", DeclaredIn: "Reserve Room"},
				{Kind: diagram.CopyOfEvent, Name: "GuestCheckedIn", DeclaredIn: "Check In"},
				{Kind: diagram.CopyOfEvent, Name: "GuestCheckedOut", DeclaredIn: "Check Out"},
			}, copies["See Occupancy"])
		})

		t.Run("a view in a DCB context copies as one under an aggregate does", func(t *testing.T) {
			copies := copiesBySliceName(test.SubscribedEventsHotelStaysModel(t))

			require.Equal(t, []diagram.ElementCopy{
				{Kind: diagram.CopyOfEvent, Name: "RoomReserved", DeclaredIn: "Reserve Room"},
				{Kind: diagram.CopyOfEvent, Name: "GuestCheckedIn", DeclaredIn: "Check In"},
				{Kind: diagram.CopyOfEvent, Name: "GuestCheckedOut", DeclaredIn: "Check Out"},
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

			require.Equal(t, []diagram.ElementCopy{
				{Kind: diagram.CopyOfEvent, Name: "Second", DeclaredIn: "Declares Two"},
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

			require.Equal(t, []diagram.ElementCopy{
				{Kind: diagram.CopyOfEvent, Name: "SecondImported", DeclaredIn: "Import"},
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

			require.Equal(t, []diagram.ElementCopy{
				{Kind: diagram.CopyOfEvent, Name: "Fourth", DeclaredIn: "Declares Fourth"},
			}, copiesBySliceName(model)["Watch"])
		})
	})

	t.Run("copy an activating event", func(t *testing.T) {
		t.Run("an automation copies the event it activates on from two or more slices away, and none its own slice declares", func(t *testing.T) {
			copies := copiesBySliceName(test.AutomationDelayLibraryLendingModel(t))

			require.Equal(t, map[string][]diagram.ElementCopy{
				"Chase Overdue Copy": {
					{Kind: diagram.CopyOfEvent, Name: "CopyBorrowed", DeclaredIn: "Borrow Copy"},
				},
				"Close Reading Room": {
					{Kind: diagram.CopyOfEvent, Name: "DeskClaimed", DeclaredIn: "Claim Desk"},
					{Kind: diagram.CopyOfEvent, Name: "DeskReleased", DeclaredIn: "Release Desk"},
				},
			}, map[string][]diagram.ElementCopy{
				"Chase Overdue Copy": copiesOfKind(copies["Chase Overdue Copy"], diagram.CopyOfEvent),
				"Close Reading Room": copiesOfKind(copies["Close Reading Room"], diagram.CopyOfEvent),
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
			require.Equal(t, map[string][]diagram.ElementCopy{
				"Chase Overdue Copy": {
					{Kind: diagram.CopyOfView, Name: "DeskOccupancyView", DeclaredIn: "Browse Desk Occupancy"},
					{Kind: diagram.CopyOfView, Name: "MemberLoansView", DeclaredIn: "Review Member Loans"},
				},
				"Claim Desk": {
					{Kind: diagram.CopyOfView, Name: "DeskOccupancyView", DeclaredIn: "Browse Desk Occupancy"},
				},
				"Close Reading Room": {
					{Kind: diagram.CopyOfView, Name: "MemberLoansView", DeclaredIn: "Review Member Loans"},
				},
			}, copiesOfKindBySliceName(test.TriggerReadsLibraryLendingModel(t), diagram.CopyOfView))
		})

		t.Run("an automation copies the command it issues from two or more slices away, and none its own slice declares", func(t *testing.T) {
			require.Equal(t, map[string][]diagram.ElementCopy{
				"Close Reading Room": {
					{Kind: diagram.CopyOfCommand, Name: "ReleaseDesk", DeclaredIn: "Release Desk"},
					{Kind: diagram.CopyOfCommand, Name: "RemindMember", DeclaredIn: "Chase Overdue Copy"},
				},
			}, copiesOfKindBySliceName(test.TriggerReadsLibraryLendingModel(t), diagram.CopyOfCommand))
		})

		t.Run("a translation copies the view it reads and the command it issues from two or more slices away", func(t *testing.T) {
			require.Equal(t, []diagram.ElementCopy{
				{Kind: diagram.CopyOfView, Name: "PartnersView", DeclaredIn: "Read Partners"},
				{Kind: diagram.CopyOfCommand, Name: "ImportBooking", DeclaredIn: "Record Booking"},
			}, copiesBySliceName(farTranslationModel())["Import Booking"])
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

			require.Equal(t, []diagram.ElementCopy{
				{Kind: diagram.CopyOfView, Name: "SecondView", DeclaredIn: "Declares Two"},
				{Kind: diagram.CopyOfCommand, Name: "SecondCommand", DeclaredIn: "Declares Two"},
			}, copiesBySliceName(model)["Automate"])
		})

		t.Run("an event and a view of one name are each copied", func(t *testing.T) {
			require.Equal(t, []diagram.ElementCopy{
				{Kind: diagram.CopyOfEvent, Name: "Stock", DeclaredIn: "Declare Event"},
				{Kind: diagram.CopyOfView, Name: "Stock", DeclaredIn: "Declare View"},
			}, copiesBySliceName(sharedNameModel())["React"])
		})
	})

	t.Run("find the copy an arrow joins", func(t *testing.T) {
		t.Run("an arrow into a view from a copied event starts at the copy in the view's slice", func(t *testing.T) {
			model := test.SubscribedEventsHotelStaysModel(t)

			copied, found := diagram.NewElementCopies(model).At(sliceNamed(t, model, "See Stay History"),
				diagram.Edge{Kind: diagram.EdgeSubscription, From: "RoomReserved", To: "StayHistoryView"})

			require.True(t, found)
			require.Equal(t, diagram.ElementCopy{Kind: diagram.CopyOfEvent, Name: "RoomReserved", DeclaredIn: "Reserve Room"}, copied)
		})

		t.Run("an arrow into a view from an event its slice does not copy joins no copy", func(t *testing.T) {
			model := test.SubscribedEventsHotelStaysModel(t)

			_, found := diagram.NewElementCopies(model).At(sliceNamed(t, model, "See Stay History"),
				diagram.Edge{Kind: diagram.EdgeSubscription, From: "GuestCheckedOut", To: "StayHistoryView"})

			require.False(t, found)
		})

		t.Run("an arrow into an automation from a copied event starts at the copy in the automation's slice", func(t *testing.T) {
			model := test.AutomationDelayLibraryLendingModel(t)

			copied, found := diagram.NewElementCopies(model).At(sliceNamed(t, model, "Chase Overdue Copy"),
				diagram.Edge{Kind: diagram.EdgeAutomationTrigger, From: "CopyBorrowed", To: "RemindAfterGracePeriod"})

			require.True(t, found)
			require.Equal(t, diagram.ElementCopy{Kind: diagram.CopyOfEvent, Name: "CopyBorrowed", DeclaredIn: "Borrow Copy"}, copied)
		})

		t.Run("a reads arrow from a copied view starts at the copy in the reader's slice", func(t *testing.T) {
			model := test.TriggerReadsLibraryLendingModel(t)

			copied, found := diagram.NewElementCopies(model).At(sliceNamed(t, model, "Claim Desk"),
				diagram.Edge{Kind: diagram.EdgeTriggerReads, From: "DeskOccupancyView", To: "Desk Kiosk"})

			require.True(t, found)
			require.Equal(t, diagram.ElementCopy{Kind: diagram.CopyOfView, Name: "DeskOccupancyView", DeclaredIn: "Browse Desk Occupancy"}, copied)
		})

		t.Run("a command arrow into a copied command ends at the copy in the automation's slice", func(t *testing.T) {
			model := test.TriggerReadsLibraryLendingModel(t)

			copied, found := diagram.NewElementCopies(model).At(sliceNamed(t, model, "Close Reading Room"),
				diagram.Edge{Kind: diagram.EdgeAutomationCommand, From: "RemindReaderOfLoans", To: "RemindMember"})

			require.True(t, found)
			require.Equal(t, diagram.ElementCopy{Kind: diagram.CopyOfCommand, Name: "RemindMember", DeclaredIn: "Chase Overdue Copy"}, copied)
		})

		t.Run("the arrow a translation implies from its copied command starts at the copy", func(t *testing.T) {
			model := farTranslationModel()

			copied, found := diagram.NewElementCopies(model).At(sliceNamed(t, model, "Import Booking"),
				diagram.Edge{Kind: diagram.EdgeTranslationFlow, From: "ImportBooking", To: "BookingImported"})

			require.True(t, found)
			require.Equal(t, diagram.ElementCopy{Kind: diagram.CopyOfCommand, Name: "ImportBooking", DeclaredIn: "Record Booking"}, copied)
		})

		t.Run("an arrow joins the copy of its own kind when an event and a view share a name", func(t *testing.T) {
			model := sharedNameModel()
			copies := diagram.NewElementCopies(model)
			react := sliceNamed(t, model, "React")

			fromEvent, eventFound := copies.At(react, diagram.Edge{Kind: diagram.EdgeAutomationTrigger, From: "Stock", To: "Restock"})
			fromView, viewFound := copies.At(react, diagram.Edge{Kind: diagram.EdgeAutomationReads, From: "Stock", To: "Restock"})

			require.Equal(t, []any{
				diagram.ElementCopy{Kind: diagram.CopyOfEvent, Name: "Stock", DeclaredIn: "Declare Event"}, true,
				diagram.ElementCopy{Kind: diagram.CopyOfView, Name: "Stock", DeclaredIn: "Declare View"}, true,
			}, []any{fromEvent, eventFound, fromView, viewFound})
		})

		t.Run("a flow arrow into a copied event joins no copy", func(t *testing.T) {
			model := test.AutomationDelayLibraryLendingModel(t)

			_, found := diagram.NewElementCopies(model).At(sliceNamed(t, model, "Chase Overdue Copy"),
				diagram.Edge{Kind: diagram.EdgeFlow, From: "RemindMember", To: "CopyBorrowed"})

			require.False(t, found)
		})
	})

	t.Run("find the element an arrow joins with no copy", func(t *testing.T) {
		t.Run("an arrow joins the element its own slice declares, though an earlier slice declares it too", func(t *testing.T) {
			model := slicesModel(
				declaringSlice("Own"), declaringSlice("Far"), declaringSlice("Between"),
				&ast.Slice{
					Name:   "Watch",
					Events: []*ast.Event{event("Own")},
					Views:  []*ast.View{{Name: "WatchView", Subscribes: []string{"Own", "Far"}}},
				},
			)

			joined, found := diagram.NewElementCopies(model).Joined(sliceNamed(t, model, "Watch"),
				diagram.Edge{Kind: diagram.EdgeSubscription, From: "Own", To: "WatchView"})

			require.Equal(t, []any{diagram.ElementCopy{Kind: diagram.CopyOfEvent, Name: "Own", DeclaredIn: "Watch"}, true}, []any{joined, found})
		})

		t.Run("an arrow joins the element the first slice that declares it declares, when its own slice does not", func(t *testing.T) {
			model := slicesModel(
				declaringSlice("Booked"),
				&ast.Slice{Name: "List", Views: []*ast.View{{Name: "RoomsView", Subscribes: []string{"Booked"}}}},
				&ast.Slice{Name: "Between"},
				&ast.Slice{Name: "Book Again", Events: []*ast.Event{event("Booked")}},
			)

			joined, found := diagram.NewElementCopies(model).Joined(sliceNamed(t, model, "List"),
				diagram.Edge{Kind: diagram.EdgeSubscription, From: "Booked", To: "RoomsView"})

			require.Equal(t, []any{diagram.ElementCopy{Kind: diagram.CopyOfEvent, Name: "Booked", DeclaredIn: "Declares Booked"}, true}, []any{joined, found})
		})

		t.Run("an arrow that no copy can stand in for joins no element this way", func(t *testing.T) {
			model := slicesModel(&ast.Slice{
				Name:     "Place Order",
				Commands: []*ast.Command{{Name: "PlaceOrder"}},
				Events:   []*ast.Event{event("OrderPlaced")},
			})

			_, found := diagram.NewElementCopies(model).Joined(sliceNamed(t, model, "Place Order"),
				diagram.Edge{Kind: diagram.EdgeFlow, From: "PlaceOrder", To: "OrderPlaced"})

			require.False(t, found)
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
func copiesBySliceName(model *ast.Model) map[string][]diagram.ElementCopy {
	copies := diagram.NewElementCopies(model)
	byName := make(map[string][]diagram.ElementCopy)
	for _, s := range model.AllSlices() {
		if in := copies.In(s); len(in) > 0 {
			byName[s.Name] = in
		}
	}

	return byName
}

func copiesOfKind(copies []diagram.ElementCopy, kind diagram.CopyKind) []diagram.ElementCopy {
	var ofKind []diagram.ElementCopy
	for _, c := range copies {
		if c.Kind == kind {
			ofKind = append(ofKind, c)
		}
	}

	return ofKind
}

// copiesOfKindBySliceName states the copies of kind of each slice that draws
// any.
func copiesOfKindBySliceName(model *ast.Model, kind diagram.CopyKind) map[string][]diagram.ElementCopy {
	byName := make(map[string][]diagram.ElementCopy)
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

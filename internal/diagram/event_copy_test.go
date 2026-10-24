//go:build unit

package diagram_test

import (
	"testing"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/diagram"
	"github.com/hpcsc/emod/internal/test"
	"github.com/stretchr/testify/require"
)

func TestEventCopies(t *testing.T) {
	t.Run("copy", func(t *testing.T) {
		t.Run("a view subscribing to four events from other slices copies all four into its own slice", func(t *testing.T) {
			copies := copiesBySliceName(test.SubscribedEventsHotelStaysModel(t))

			require.Equal(t, []diagram.EventCopy{
				{Event: "RoomReserved", DeclaredIn: "Reserve Room"},
				{Event: "GuestCheckedIn", DeclaredIn: "Check In"},
				{Event: "BreakfastOrdered", DeclaredIn: "Import Breakfast Orders"},
				{Event: "GuestCheckedOut", DeclaredIn: "Check Out"},
			}, copies["See Stay History"])
		})

		t.Run("a view in a DCB context copies into its own slice as one under an aggregate does", func(t *testing.T) {
			copies := copiesBySliceName(test.SubscribedEventsHotelStaysModel(t))

			require.Equal(t, []diagram.EventCopy{
				{Event: "RoomReserved", DeclaredIn: "Reserve Room"},
				{Event: "GuestCheckedIn", DeclaredIn: "Check In"},
				{Event: "GuestCheckedOut", DeclaredIn: "Check Out"},
				{Event: "RoomCleaned", DeclaredIn: "Clean Room"},
			}, copies["Plan Cleaning"])
		})

		t.Run("a view subscribing to three events from other slices copies none", func(t *testing.T) {
			copies := copiesBySliceName(test.SubscribedEventsHotelStaysModel(t))

			require.NotContains(t, copies, "See Occupancy")
		})

		t.Run("an event the view's own slice declares does not count toward the threshold", func(t *testing.T) {
			model := slicesModel(
				&ast.Slice{
					Name:   "Watch",
					Events: []*ast.Event{event("Own")},
					Views:  []*ast.View{{Name: "WatchView", Subscribes: []string{"Own", "First", "Second", "Third"}}},
				},
				declaringSlice("First"), declaringSlice("Second"), declaringSlice("Third"),
			)

			require.Empty(t, diagram.EventCopies(model))
		})

		t.Run("an event no slice declares does not count toward the threshold", func(t *testing.T) {
			model := slicesModel(
				&ast.Slice{
					Name:  "Watch",
					Views: []*ast.View{{Name: "WatchView", Subscribes: []string{"First", "Second", "Third", "Undeclared"}}},
				},
				declaringSlice("First"), declaringSlice("Second"), declaringSlice("Third"),
			)

			require.Empty(t, diagram.EventCopies(model))
		})

		t.Run("an event a view names twice counts once toward the threshold", func(t *testing.T) {
			model := slicesModel(
				&ast.Slice{
					Name:  "Watch",
					Views: []*ast.View{{Name: "WatchView", Subscribes: []string{"First", "Second", "Third", "First"}}},
				},
				declaringSlice("First"), declaringSlice("Second"), declaringSlice("Third"),
			)

			require.Empty(t, diagram.EventCopies(model))
		})

		t.Run("two views in one slice share one copy of an event they both subscribe to", func(t *testing.T) {
			subscribed := []string{"First", "Second", "Third", "Fourth"}
			model := slicesModel(
				&ast.Slice{
					Name: "Watch",
					Views: []*ast.View{
						{Name: "WatchView", Subscribes: subscribed},
						{Name: "AuditView", Subscribes: []string{"Fourth", "Fifth", "First", "Second"}},
					},
				},
				declaringSlice("First"), declaringSlice("Second"), declaringSlice("Third"),
				declaringSlice("Fourth"), declaringSlice("Fifth"),
			)

			require.Equal(t, []string{"First", "Second", "Third", "Fourth", "Fifth"},
				eventsCopiedIn(copiesBySliceName(model)["Watch"]))
		})

		t.Run("an event two slices declare is copied from the first of them", func(t *testing.T) {
			model := slicesModel(
				&ast.Slice{
					Name:  "Watch",
					Views: []*ast.View{{Name: "WatchView", Subscribes: []string{"First", "Second", "Third", "Fourth"}}},
				},
				declaringSlice("First"), declaringSlice("Second"), declaringSlice("Third"),
				&ast.Slice{Name: "Declares Fourth", Events: []*ast.Event{event("Fourth")}},
				&ast.Slice{Name: "Declares Fourth Again", Events: []*ast.Event{event("Fourth")}},
			)

			require.Contains(t, copiesBySliceName(model)["Watch"],
				diagram.EventCopy{Event: "Fourth", DeclaredIn: "Declares Fourth"})
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

func copiesBySliceName(model *ast.Model) map[string][]diagram.EventCopy {
	byName := make(map[string][]diagram.EventCopy)
	for slice, copies := range diagram.EventCopies(model) {
		byName[slice.Name] = copies
	}

	return byName
}

func eventsCopiedIn(copies []diagram.EventCopy) []string {
	var events []string
	for _, c := range copies {
		events = append(events, c.Event)
	}

	return events
}

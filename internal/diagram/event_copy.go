package diagram

import (
	"slices"

	"github.com/hpcsc/emod/internal/ast"
)

// copyThreshold is the most events from other slices a view draws an arrow from
// each of. Past it, the view's slice draws every one of them again.
const copyThreshold = 3

// EventCopy is an event drawn again in the slice of a view that subscribes to it
// from another slice, so the arrow into the view stays inside that slice.
type EventCopy struct {
	Event      string
	DeclaredIn string
}

func (c EventCopy) labelLines() []string {
	return []string{c.Event, "from " + c.DeclaredIn}
}

// EventCopies returns the events each slice draws again, keyed by that slice. A
// slice copies each event one of its views subscribes to from another slice,
// once that view subscribes to more than three of them, and copies it once
// however many of its views subscribe to it.
func EventCopies(model *ast.Model) map[*ast.Slice][]EventCopy {
	declaredIn := declaringSlices(model)
	copies := make(map[*ast.Slice][]EventCopy)

	for _, s := range model.AllSlices() {
		for _, v := range s.Views {
			if v == nil {
				continue
			}
			elsewhere := subscribedElsewhere(s, v, declaredIn)
			if len(elsewhere) <= copyThreshold {
				continue
			}
			for _, c := range elsewhere {
				if !slices.Contains(copies[s], c) {
					copies[s] = append(copies[s], c)
				}
			}
		}
	}

	return copies
}

// subscribedElsewhere returns the events v subscribes to that a slice other than
// s declares. An event no slice declares draws no arrow, so it does not count.
func subscribedElsewhere(s *ast.Slice, v *ast.View, declaredIn map[string]*ast.Slice) []EventCopy {
	var elsewhere []EventCopy
	for _, name := range v.Subscribes {
		origin, declared := declaredIn[name]
		if !declared || slices.Contains(declaredEvents(s), name) {
			continue
		}
		c := EventCopy{Event: name, DeclaredIn: origin.Name}
		if !slices.Contains(elsewhere, c) {
			elsewhere = append(elsewhere, c)
		}
	}

	return elsewhere
}

// declaringSlices maps each event to the first slice that declares it.
func declaringSlices(model *ast.Model) map[string]*ast.Slice {
	declaredIn := make(map[string]*ast.Slice)
	for _, s := range model.AllSlices() {
		for _, name := range declaredEvents(s) {
			if _, seen := declaredIn[name]; !seen {
				declaredIn[name] = s
			}
		}
	}

	return declaredIn
}

// declaredEvents names the events a slice declares: its own, and the event each
// of its translations nests.
func declaredEvents(s *ast.Slice) []string {
	var names []string
	for _, evt := range s.Events {
		if evt != nil {
			names = append(names, evt.Name)
		}
	}
	for _, tr := range s.Translations {
		if tr != nil && tr.Event != nil && tr.Event.Name != "" {
			names = append(names, tr.Event.Name)
		}
	}

	return names
}

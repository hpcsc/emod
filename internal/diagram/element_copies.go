package diagram

import (
	"slices"

	"github.com/hpcsc/emod/internal/ast"
)

// slicesApartToCopy is how many slices apart, in layout order, the two ends of
// an arrow are before the slice at the far end draws a copy in place of the
// arrow.
const slicesApartToCopy = 2

// CopyKind is the element type a copy draws again.
type CopyKind int

const (
	CopyOfEvent CopyKind = iota
)

// ElementCopy is an element drawn again in a slice that one of its arrows joins
// to the element from a slice far away.
type ElementCopy struct {
	Kind       CopyKind
	Name       string
	DeclaredIn string
}

func (c ElementCopy) labelLines() []string {
	return []string{c.Name, "from " + c.DeclaredIn}
}

// ElementCopies is what each slice of a model draws again.
type ElementCopies struct {
	bySlice map[*ast.Slice][]ElementCopy
}

// NewElementCopies decides the copies of every slice of model. A slice copies an
// element when one of its arrows joins it to that element from a slice at least
// slicesApartToCopy slices away, and copies it once however many of its arrows
// do.
func NewElementCopies(model *ast.Model) ElementCopies {
	refs := model.SliceRefs()
	layoutOrder := make(map[*ast.Slice]int, len(refs))
	for i, ref := range refs {
		layoutOrder[ref.Slice] = i
	}
	eventsIn := declaringSlices(model)

	c := ElementCopies{bySlice: make(map[*ast.Slice][]ElementCopy)}
	for _, ref := range refs {
		s := ref.Slice
		for _, edge := range SliceEdges(s) {
			kind, name, copyable := c.copiedEnd(edge)
			if !copyable {
				continue
			}
			origin, declared := eventsIn[name]
			if !declared || slices.Contains(declaredEvents(s), name) {
				continue
			}
			if apart(layoutOrder[s], layoutOrder[origin]) < slicesApartToCopy {
				continue
			}
			copied := ElementCopy{Kind: kind, Name: name, DeclaredIn: origin.Name}
			if !slices.Contains(c.bySlice[s], copied) {
				c.bySlice[s] = append(c.bySlice[s], copied)
			}
		}
	}

	return c
}

// In returns the copies slice s draws, in the order its arrows first join them.
func (c ElementCopies) In(s *ast.Slice) []ElementCopy {
	return c.bySlice[s]
}

// At returns the copy that edge, an arrow of slice s, joins in place of the
// element it names, and whether s draws one.
func (c ElementCopies) At(s *ast.Slice, edge Edge) (ElementCopy, bool) {
	kind, name, copyable := c.copiedEnd(edge)
	if !copyable {
		return ElementCopy{}, false
	}
	for _, copied := range c.bySlice[s] {
		if copied.Kind == kind && copied.Name == name {
			return copied, true
		}
	}

	return ElementCopy{}, false
}

// copiedEnd names the end of edge that a copy can stand in for.
func (c ElementCopies) copiedEnd(edge Edge) (CopyKind, string, bool) {
	switch edge.Kind {
	case EdgeSubscription, EdgeAutomationTrigger:
		return CopyOfEvent, edge.From, true
	default:
		return 0, "", false
	}
}

func apart(a, b int) int {
	if a > b {
		return a - b
	}

	return b - a
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

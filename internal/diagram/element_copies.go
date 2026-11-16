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
	CopyOfView
	CopyOfCommand
)

// namesDeclaredIn names the elements of this kind that slice s declares.
func (k CopyKind) namesDeclaredIn(s *ast.Slice) []string {
	var names []string
	switch k {
	case CopyOfView:
		for _, v := range s.Views {
			if v != nil {
				names = append(names, v.Name)
			}
		}
	case CopyOfCommand:
		for _, cmd := range s.Commands {
			if cmd != nil {
				names = append(names, cmd.Name)
			}
		}
	default:
		names = declaredEvents(s)
	}

	return names
}

// ElementCopy is an element drawn again in a slice that one of its arrows joins
// to the element from a slice far away.
type ElementCopy struct {
	Kind       CopyKind
	Name       string
	DeclaredIn string
}

func (c ElementCopy) labelLines() []string {
	if c.Kind == CopyOfCommand {
		return []string{c.Name, "in " + c.DeclaredIn}
	}

	return []string{c.Name, "from " + c.DeclaredIn}
}

// ElementCopies is what each slice of a model draws again.
type ElementCopies struct {
	bySlice        map[*ast.Slice][]ElementCopy
	firstDeclaring map[CopyKind]map[string]*ast.Slice
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
	firstDeclaring := map[CopyKind]map[string]*ast.Slice{
		CopyOfEvent:   firstDeclaringSlices(model, CopyOfEvent),
		CopyOfView:    firstDeclaringSlices(model, CopyOfView),
		CopyOfCommand: firstDeclaringSlices(model, CopyOfCommand),
	}

	c := ElementCopies{bySlice: make(map[*ast.Slice][]ElementCopy), firstDeclaring: firstDeclaring}
	for _, ref := range refs {
		s := ref.Slice
		for _, edge := range SliceEdges(s) {
			kind, name, copyable := copiedEnd(edge)
			if !copyable {
				continue
			}
			first, declared := c.firstDeclaring[kind][name]
			if !declared || slices.Contains(kind.namesDeclaredIn(s), name) {
				continue
			}
			if abs(layoutOrder[s]-layoutOrder[first]) < slicesApartToCopy {
				continue
			}
			copied := ElementCopy{Kind: kind, Name: name, DeclaredIn: first.Name}
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
	kind, name, copyable := copiedEnd(edge)
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

// Joined returns the element that the end of edge, an arrow of slice s, that a
// copy can stand in for joins when s draws no copy of it: the element s declares
// itself, or else the element as the first slice that declares it declares it.
// It reports false when no slice declares the element, and for an edge that no
// copy can stand in for.
func (c ElementCopies) Joined(s *ast.Slice, edge Edge) (ElementCopy, bool) {
	kind, name, copyable := copiedEnd(edge)
	if !copyable {
		return ElementCopy{}, false
	}
	if slices.Contains(kind.namesDeclaredIn(s), name) {
		return ElementCopy{Kind: kind, Name: name, DeclaredIn: s.Name}, true
	}
	first, declared := c.firstDeclaring[kind][name]
	if !declared {
		return ElementCopy{}, false
	}

	return ElementCopy{Kind: kind, Name: name, DeclaredIn: first.Name}, true
}

// copiedEnd names the end of edge that a copy can stand in for.
func copiedEnd(edge Edge) (CopyKind, string, bool) {
	switch edge.Kind {
	case EdgeSubscription, EdgeAutomationTrigger:
		return CopyOfEvent, edge.From, true
	case EdgeTriggerReads, EdgeAutomationReads, EdgeTranslationReads:
		return CopyOfView, edge.From, true
	case EdgeAutomationCommand, EdgeTranslationCommand:
		return CopyOfCommand, edge.To, true
	case EdgeTranslationFlow:
		return CopyOfCommand, edge.From, true
	default:
		return 0, "", false
	}
}

// firstDeclaringSlices maps each element of kind to the first slice that
// declares it.
func firstDeclaringSlices(model *ast.Model, kind CopyKind) map[string]*ast.Slice {
	first := make(map[string]*ast.Slice)
	for _, s := range model.AllSlices() {
		for _, name := range kind.namesDeclaredIn(s) {
			if _, seen := first[name]; !seen {
				first[name] = s
			}
		}
	}

	return first
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

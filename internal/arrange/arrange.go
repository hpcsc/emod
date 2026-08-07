// Package arrange orders the slices inside a container so a model reads
// forward: as far as possible, every reference a slice makes points at a slice
// declared before it.
//
// The order is not a sort. A slice carries no key saying where it belongs; the
// order falls out of the references BETWEEN slices, which makes this a
// linearization of a directed graph rather than a comparison of items.
//
// Minimizing backward references alone is the wrong target. A model whose
// slices are ordered purely to that end reads as nonsense — the resolution of a
// thing before the thing exists — because it will happily put the whole
// lifecycle in reverse to save an arrow. So the process slices keep the order
// their author gave them, which is the story the model tells, and only view
// slices move. That is also where the freedom actually is: a view is a
// projection of events rather than a step in the process, so it has no place in
// the story of its own.
package arrange

import "github.com/hpcsc/emod/internal/ast"

// Reference kinds, named for the syntax that creates them.
const (
	KindSubscribes = "subscribes"
	KindReads      = "reads"
	KindOn         = "on"
	KindFlow       = "flow"
)

// Reference is one slice pointing at another inside a single container.
type Reference struct {
	From  *ast.Slice
	To    *ast.Slice
	Kind  string
	Label string
}

// Report describes what arranging a model did and what it could not fix.
type Report struct {
	Moved          int
	BackwardBefore int
	BackwardAfter  int
	Backward       []Reference
}

// Changed reports whether arranging moved any slice.
func (r Report) Changed() bool { return r.Moved > 0 }

// Model reorders the slices of every container in the model, in place, and
// reports what changed. Containers are arranged independently: slice order only
// means anything among siblings, and a reference reaching into another
// container cannot be fixed by moving either end.
//
// This walks ctx.Slices and agg.Slices directly where the rest of the codebase
// goes through the ast traversal helpers. Those helpers flatten every container
// into one list sorted by source position, which is the very thing reordering
// exists to change.
func Model(m *ast.Model) Report {
	var report Report
	if m == nil {
		return report
	}

	for _, ctx := range m.Contexts {
		for _, agg := range ctx.Aggregates {
			agg.Slices = arrangeContainer(agg.Slices, &report)
		}
		ctx.Slices = arrangeContainer(ctx.Slices, &report)
	}

	return report
}

func arrangeContainer(slices []*ast.Slice, report *Report) []*ast.Slice {
	if len(slices) < 2 {
		return slices
	}

	refs := references(slices)
	report.BackwardBefore += countBackward(slices, refs)

	arranged := bestOrder(slices, refs)

	// Count the views that changed slot rather than the slices sitting at a new
	// index: relocating one view shifts everything after it, which would report
	// a whole container as moved for a single change.
	before, after := slots(slices), slots(arranged)
	for s, was := range before {
		if after[s] != was {
			report.Moved++
		}
	}

	report.BackwardAfter += countBackward(arranged, refs)
	report.Backward = append(report.Backward, backward(arranged, refs)...)

	return arranged
}

// references collects every reference between two different slices of one
// container. Where a name resolves outside the container it is left out: no
// ordering of these slices can turn such a reference forward.
func references(slices []*ast.Slice) []Reference {
	eventHome := map[string]*ast.Slice{}
	viewHome := map[string]*ast.Slice{}
	for _, s := range slices {
		for _, e := range s.Events {
			eventHome[e.Name] = s
		}
		for _, v := range s.Views {
			viewHome[v.Name] = s
		}
		for _, t := range s.Translations {
			if t.Event != nil {
				eventHome[t.Event.Name] = s
			}
		}
	}

	var refs []Reference
	add := func(from, to *ast.Slice, kind, label string) {
		if from == nil || to == nil || from == to {
			return
		}
		refs = append(refs, Reference{From: from, To: to, Kind: kind, Label: label})
	}

	for _, s := range slices {
		for _, v := range s.Views {
			for _, e := range v.Subscribes {
				add(eventHome[e], s, KindSubscribes, e+" -> "+v.Name)
			}
		}
		if s.Trigger != nil && s.Trigger.Reads != "" {
			add(viewHome[s.Trigger.Reads], s, KindReads, s.Trigger.Reads+" -> trigger "+s.Trigger.Name)
		}
		for _, a := range s.Automations {
			add(viewHome[a.Reads], s, KindReads, a.Reads+" -> "+a.Name)
			add(eventHome[a.OnEvent], s, KindOn, a.OnEvent+" -> "+a.Name)
		}
		for _, t := range s.Translations {
			add(viewHome[t.Reads], s, KindReads, t.Reads+" -> "+t.Name)
		}
		for _, f := range s.Flows {
			add(s, eventHome[f.EventName], KindFlow, f.CommandName+" -> "+f.EventName)
		}
	}

	return refs
}

// movable reports whether a slice is a pure read model — one that declares
// views and nothing that writes. Such a slice states no step of the process, so
// moving it changes no story. A slice that declares a view alongside a command
// does state one, and stays where its author put it.
func movable(s *ast.Slice) bool {
	return len(s.Views) > 0 &&
		len(s.Commands) == 0 &&
		len(s.Events) == 0 &&
		len(s.Automations) == 0 &&
		len(s.Translations) == 0
}

// bestOrder holds the process slices in their authored order and searches for
// the slot for each view slice that leaves the fewest references pointing
// backward.
//
// Each view is placed against the others' current slots and the search repeats
// until a pass improves nothing, which handles the rare view that reads another
// view. Only a strict improvement moves a slice, so a model already arranged is
// left exactly as it is and running the command twice changes nothing.
func bestOrder(slices []*ast.Slice, refs []Reference) []*ast.Slice {
	var fixed, views []*ast.Slice
	for _, s := range slices {
		if movable(s) {
			views = append(views, s)
		} else {
			fixed = append(fixed, s)
		}
	}
	if len(views) == 0 || len(fixed) == 0 {
		return slices
	}

	// A slot is a gap in the fixed sequence: slot n sits after n fixed slices.
	slot := map[*ast.Slice]int{}
	seen := 0
	for _, s := range slices {
		if movable(s) {
			slot[s] = seen
			continue
		}
		seen++
	}

	assemble := func() []*ast.Slice {
		out := make([]*ast.Slice, 0, len(slices))
		for i := 0; i <= len(fixed); i++ {
			for _, v := range views {
				if slot[v] == i {
					out = append(out, v)
				}
			}
			if i < len(fixed) {
				out = append(out, fixed[i])
			}
		}
		return out
	}

	best := countBackward(assemble(), refs)
	for improved := true; improved; {
		improved = false
		for _, v := range views {
			start := slot[v]
			for candidate := 0; candidate <= len(fixed); candidate++ {
				if candidate == start {
					continue
				}
				slot[v] = candidate
				if got := countBackward(assemble(), refs); got < best {
					best, start, improved = got, candidate, true
				}
			}
			slot[v] = start
		}
	}

	return assemble()
}

// slots maps each movable slice to the number of process slices ahead of it,
// which is the position that actually says where a view sits in the story.
func slots(order []*ast.Slice) map[*ast.Slice]int {
	out := map[*ast.Slice]int{}
	seen := 0
	for _, s := range order {
		if movable(s) {
			out[s] = seen
			continue
		}
		seen++
	}
	return out
}

func positions(order []*ast.Slice) map[*ast.Slice]int {
	pos := make(map[*ast.Slice]int, len(order))
	for i, s := range order {
		pos[s] = i
	}
	return pos
}

func countBackward(order []*ast.Slice, refs []Reference) int {
	pos := positions(order)
	count := 0
	for _, r := range refs {
		if pos[r.From] > pos[r.To] {
			count++
		}
	}
	return count
}

func backward(order []*ast.Slice, refs []Reference) []Reference {
	pos := positions(order)
	var out []Reference
	for _, r := range refs {
		if pos[r.From] > pos[r.To] {
			out = append(out, r)
		}
	}
	return out
}

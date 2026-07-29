package validator

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/diagnostic"
)

func Validate(model *ast.Model) []*diagnostic.Entry {
	if model == nil {
		return nil
	}

	index := newModelIndex(model)

	var diags []*diagnostic.Entry
	for _, slice := range index.slices {
		diags = append(diags, referenceDiagnostics(slice, index)...)
	}
	for _, slice := range index.slices {
		diags = append(diags, tagFieldRefDiagnostics(slice, index)...)
	}
	for _, slice := range index.slices {
		diags = append(diags, decidesOnDiagnostics(slice, index)...)
	}
	diags = append(diags, redeclaredInvariantDiagnostics(model)...)
	diags = append(diags, orphanDiagnostics(index.commandPositions, index.referencedCommands,
		"orphan-command", "command %q is orphaned (no flow references it)")...)
	diags = append(diags, orphanDiagnostics(index.eventPositions, index.producedEvents,
		"orphan-event", "event %q is orphaned (no flow, external source, or translation produces it)")...)

	return diags
}

type modelIndex struct {
	slices             []*ast.Slice
	contextNames       map[string]bool
	commandNames       map[string]bool
	eventNames         map[string]bool
	commandPositions   map[string]ast.Position
	eventPositions     map[string]ast.Position
	referencedCommands map[string]bool
	producedEvents     map[string]bool
	eventFields        map[string][]string
	eventTagKeys       map[string][]string
}

func newModelIndex(model *ast.Model) *modelIndex {
	index := &modelIndex{
		contextNames:       make(map[string]bool, len(model.Contexts)),
		commandNames:       make(map[string]bool),
		eventNames:         make(map[string]bool),
		commandPositions:   make(map[string]ast.Position),
		eventPositions:     make(map[string]ast.Position),
		referencedCommands: make(map[string]bool),
		producedEvents:     make(map[string]bool),
		eventFields:        make(map[string][]string),
		eventTagKeys:       make(map[string][]string),
	}

	for _, ctx := range model.Contexts {
		index.contextNames[ctx.Name] = true
		index.slices = append(index.slices, ctx.Slices...)
		for _, agg := range ctx.Aggregates {
			index.slices = append(index.slices, agg.Slices...)
		}
	}

	for _, slice := range index.slices {
		index.collect(slice)
	}

	return index
}

func (i *modelIndex) collect(slice *ast.Slice) {
	for _, cmd := range slice.Commands {
		i.commandNames[cmd.Name] = true
		i.commandPositions[cmd.Name] = cmd.NamePos
	}
	for _, evt := range slice.Events {
		i.eventNames[evt.Name] = true
		i.eventPositions[evt.Name] = evt.NamePos
		if evt.Source != "" {
			i.producedEvents[evt.Name] = true
		}
		i.collectEventShape(evt)
	}
	for _, tr := range slice.Translations {
		if tr.Event != nil {
			i.eventNames[tr.Event.Name] = true
			i.producedEvents[tr.Event.Name] = true
			i.collectEventShape(tr.Event)
		}
		if tr.Command != "" {
			i.referencedCommands[tr.Command] = true
		}
	}
	for _, f := range slice.Flows {
		if f.CommandName != "" {
			i.referencedCommands[f.CommandName] = true
		}
		if f.EventName != "" {
			i.producedEvents[f.EventName] = true
		}
	}
	for _, auto := range slice.Automations {
		if auto.Command != "" {
			i.referencedCommands[auto.Command] = true
		}
	}
}

func (i *modelIndex) collectEventShape(evt *ast.Event) {
	for _, f := range evt.Fields {
		i.eventFields[evt.Name] = append(i.eventFields[evt.Name], f.Name)
	}
	for _, tag := range evt.Tags {
		i.eventTagKeys[evt.Name] = append(i.eventTagKeys[evt.Name], tag.Key)
	}
}

func anyEventDeclares(events []string, byEvent map[string][]string, name string) bool {
	for _, evt := range events {
		if slices.Contains(byEvent[evt], name) {
			return true
		}
	}

	return false
}

func errorAt(pos ast.Position, format string, args ...any) *diagnostic.Entry {
	return &diagnostic.Entry{
		Filename: pos.Filename,
		Line:     pos.Line,
		Column:   pos.Column,
		Severity: diagnostic.Error,
		Message:  fmt.Sprintf(format, args...),
	}
}

func orphanDiagnostics(positions map[string]ast.Position, referenced map[string]bool, ruleName, messageFormat string) []*diagnostic.Entry {
	var diags []*diagnostic.Entry
	for _, name := range orphanNames(positions, referenced) {
		entry := errorAt(positions[name], messageFormat, name)
		entry.RuleName = ruleName
		diags = append(diags, entry)
	}

	return diags
}

// orphanNames returns the names in positions that referenced does not cover,
// ordered by where they are declared. Ranging over the map directly would emit
// diagnostics in Go's randomised map order, so the same file would report the
// same problems in a different order on every run.
func orphanNames(positions map[string]ast.Position, referenced map[string]bool) []string {
	names := make([]string, 0, len(positions))
	for name := range positions {
		if !referenced[name] {
			names = append(names, name)
		}
	}

	slices.SortFunc(names, func(a, b string) int {
		if c := comparePositions(positions[a], positions[b]); c != 0 {
			return c
		}
		return cmp.Compare(a, b)
	})

	return names
}

func comparePositions(a, b ast.Position) int {
	if c := cmp.Compare(a.Filename, b.Filename); c != 0 {
		return c
	}
	if c := cmp.Compare(a.Line, b.Line); c != 0 {
		return c
	}
	return cmp.Compare(a.Column, b.Column)
}

type invariantRedeclaration struct {
	invariant *ast.Invariant
	scopeKind string
	scopeName string
}

// An aggregate block and a context's own invariants are separate resolution
// scopes, so one identifier may be declared in both, and in sibling aggregates,
// without either declaration hiding the other. The redeclarations are sorted
// because the AST holds a context's own invariants in a collection of their own,
// which loses whether they were written before or after its aggregates.
func redeclaredInvariantDiagnostics(model *ast.Model) []*diagnostic.Entry {
	var redeclared []invariantRedeclaration
	for _, ctx := range model.Contexts {
		redeclared = append(redeclared, redeclarationsIn(ctx.Invariants, "context", ctx.Name)...)
		for _, agg := range ctx.Aggregates {
			redeclared = append(redeclared, redeclarationsIn(agg.Invariants, "aggregate", agg.Name)...)
		}
	}

	slices.SortStableFunc(redeclared, func(a, b invariantRedeclaration) int {
		return comparePositions(a.invariant.NamePos, b.invariant.NamePos)
	})

	diags := make([]*diagnostic.Entry, 0, len(redeclared))
	for _, r := range redeclared {
		diags = append(diags, errorAt(r.invariant.NamePos,
			"invariant %q is already declared in %s %q", r.invariant.Name, r.scopeKind, r.scopeName))
	}

	return diags
}

func redeclarationsIn(invariants []*ast.Invariant, scopeKind, scopeName string) []invariantRedeclaration {
	declared := make(map[string]bool, len(invariants))
	var found []invariantRedeclaration
	for _, inv := range invariants {
		if declared[inv.Name] {
			found = append(found, invariantRedeclaration{invariant: inv, scopeKind: scopeKind, scopeName: scopeName})
			continue
		}
		declared[inv.Name] = true
	}

	return found
}

func referenceDiagnostics(slice *ast.Slice, index *modelIndex) []*diagnostic.Entry {
	var diags []*diagnostic.Entry
	for _, auto := range slice.Automations {
		if auto.TargetContext != "" && !index.contextNames[auto.TargetContext] {
			diags = append(diags, errorAt(auto.TargetContextPos, "target context %q does not exist", auto.TargetContext))
		}
		if auto.Command != "" && !index.commandNames[auto.Command] {
			diags = append(diags, errorAt(auto.CommandPos, "command %q does not exist", auto.Command))
		}
		if auto.TriggerEvent != "" && !index.eventNames[auto.TriggerEvent] {
			diags = append(diags, errorAt(auto.TriggerEventPos, "event %q does not exist", auto.TriggerEvent))
		}
	}
	for _, tr := range slice.Translations {
		if tr.Command != "" && !index.commandNames[tr.Command] {
			diags = append(diags, errorAt(tr.CommandPos, "command %q does not exist", tr.Command))
		}
	}
	for _, v := range slice.Views {
		for _, sub := range v.Subscribes {
			if sub != "" && !index.eventNames[sub] {
				diags = append(diags, errorAt(v.NamePos, "event %q does not exist", sub))
			}
		}
	}
	for _, f := range slice.Flows {
		if f.EventName != "" && !index.eventNames[f.EventName] {
			diags = append(diags, errorAt(f.EventPos, "event %q does not exist", f.EventName))
		}
	}

	return diags
}

func tagFieldRefDiagnostics(slice *ast.Slice, index *modelIndex) []*diagnostic.Entry {
	var diags []*diagnostic.Entry
	for _, evt := range slice.Events {
		diags = append(diags, eventTagFieldRefDiagnostics(evt, index)...)
	}
	for _, tr := range slice.Translations {
		if tr.Event != nil {
			diags = append(diags, eventTagFieldRefDiagnostics(tr.Event, index)...)
		}
	}

	return diags
}

func eventTagFieldRefDiagnostics(evt *ast.Event, index *modelIndex) []*diagnostic.Entry {
	var diags []*diagnostic.Entry
	for _, tag := range evt.Tags {
		if tag.FieldRef != "" && !slices.Contains(index.eventFields[evt.Name], tag.FieldRef) {
			diags = append(diags, errorAt(tag.FieldRefPos,
				"tag field reference %q does not match any field on event %q", tag.FieldRef, evt.Name))
		}
	}

	return diags
}

// decidesOnDiagnostics validates decides_on clauses on commands:
//   - Event names listed in events must exist in the model
//   - Predicate tag keys must be declared on at least one listed event
//   - Predicate field references must be declared on at least one listed event
func decidesOnDiagnostics(slice *ast.Slice, index *modelIndex) []*diagnostic.Entry {
	var diags []*diagnostic.Entry
	for _, cmd := range slice.Commands {
		if cmd.DecidesOn == nil {
			continue
		}
		clause := cmd.DecidesOn

		for i, evtName := range clause.Events {
			if !index.eventNames[evtName] {
				diags = append(diags, errorAt(clause.EventsPos[i], "event %q in decides_on does not exist", evtName))
			}
		}

		if clause.Predicate != nil {
			diags = append(diags, predicateDiagnostics(clause.Predicate, clause.Events, index)...)
		}
	}

	return diags
}

func predicateDiagnostics(expr ast.PredicateExpr, events []string, index *modelIndex) []*diagnostic.Entry {
	var diags []*diagnostic.Entry
	switch e := expr.(type) {
	case *ast.TagPredicate:
		if !anyEventDeclares(events, index.eventTagKeys, e.Field) {
			diags = append(diags, errorAt(e.FieldPos, "tag key %q is not declared on any event in decides_on", e.Field))
		}
		if !anyEventDeclares(events, index.eventFields, e.Value) {
			diags = append(diags, errorAt(e.ValuePos, "field reference %q is not declared on any event in decides_on", e.Value))
		}

	case *ast.LogicalExpr:
		diags = append(diags, predicateDiagnostics(e.Left, events, index)...)
		diags = append(diags, predicateDiagnostics(e.Right, events, index)...)

	case *ast.NotExpr:
		diags = append(diags, predicateDiagnostics(e.Expr, events, index)...)

	case *ast.FieldRef:
		// A predicate parses a bare field reference only as one side of a tag
		// comparison, which *ast.TagPredicate already covers.
	}

	return diags
}

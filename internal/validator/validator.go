package validator

import (
	"cmp"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

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
	diags = append(diags, scheduleExpressionDiagnostics(index.slices)...)
	diags = append(diags, redeclaredInvariantDiagnostics(model)...)
	diags = append(diags, unresolvedRejectionDiagnostics(model)...)
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
	viewNames          map[string]bool
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
		viewNames:          make(map[string]bool),
		commandPositions:   make(map[string]ast.Position),
		eventPositions:     make(map[string]ast.Position),
		referencedCommands: make(map[string]bool),
		producedEvents:     make(map[string]bool),
		eventFields:        make(map[string][]string),
		eventTagKeys:       make(map[string][]string),
	}

	for _, ctx := range model.Contexts {
		index.contextNames[ctx.Name] = true
	}
	index.slices = model.AllSlices()

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
	for _, v := range slice.Views {
		i.viewNames[v.Name] = true
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

// A spec's `when` names the command a command slice exercises, but the
// triggering event of an automation slice, so a name declared as either kind
// resolves; whether the kind suits the slice's pattern is a separate rule.
func (i *modelIndex) declaresCommandOrEvent(name string) bool {
	return i.commandNames[name] || i.eventNames[name]
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
		if c := positions[a].Compare(positions[b]); c != 0 {
			return c
		}
		return cmp.Compare(a, b)
	})

	return names
}

func sortInDeclarationOrder(diags []*diagnostic.Entry) {
	position := func(d *diagnostic.Entry) ast.Position {
		return ast.Position{Filename: d.Filename, Line: d.Line, Column: d.Column}
	}

	slices.SortStableFunc(diags, func(a, b *diagnostic.Entry) int {
		return position(a).Compare(position(b))
	})
}

type invariantScope struct {
	kind       string
	name       string
	invariants []*ast.Invariant
	slices     []*ast.Slice
}

type scopedInvariant struct {
	name  string
	pos   ast.Position
	scope invariantScope
}

// A context's own invariants and each of its aggregates' are separate resolution
// scopes: an identifier declared in one neither hides nor resolves against the
// same identifier in another, not even between an aggregate and the context
// enclosing it.
func invariantScopes(model *ast.Model) []invariantScope {
	var scopes []invariantScope
	for _, ctx := range model.Contexts {
		scopes = append(scopes, invariantScope{kind: "context", name: ctx.Name, invariants: ctx.Invariants, slices: ctx.Slices})
		for _, agg := range ctx.Aggregates {
			scopes = append(scopes, invariantScope{kind: "aggregate", name: agg.Name, invariants: agg.Invariants, slices: agg.Slices})
		}
	}

	return scopes
}

func (s invariantScope) redeclarations() []scopedInvariant {
	declared := make(map[string]bool, len(s.invariants))
	var found []scopedInvariant
	for _, inv := range s.invariants {
		if declared[inv.Name] {
			found = append(found, scopedInvariant{name: inv.Name, pos: inv.NamePos, scope: s})
			continue
		}
		declared[inv.Name] = true
	}

	return found
}

func (s invariantScope) unresolvedRejections() []scopedInvariant {
	declared := make(map[string]bool, len(s.invariants))
	for _, inv := range s.invariants {
		declared[inv.Name] = true
	}

	var found []scopedInvariant
	for _, slice := range s.slices {
		for _, spec := range slice.Specs {
			rejection, ok := spec.Then.(*ast.ThenRejected)
			if !ok || declared[rejection.InvariantName] {
				continue
			}
			found = append(found, scopedInvariant{name: rejection.InvariantName, pos: rejection.InvariantPos, scope: s})
		}
	}

	return found
}

func redeclaredInvariantDiagnostics(model *ast.Model) []*diagnostic.Entry {
	var redeclared []scopedInvariant
	for _, scope := range invariantScopes(model) {
		redeclared = append(redeclared, scope.redeclarations()...)
	}

	return scopedInvariantDiagnostics(redeclared, "invariant %q is already declared in %s %q")
}

func unresolvedRejectionDiagnostics(model *ast.Model) []*diagnostic.Entry {
	var unresolved []scopedInvariant
	for _, scope := range invariantScopes(model) {
		unresolved = append(unresolved, scope.unresolvedRejections()...)
	}

	return scopedInvariantDiagnostics(unresolved, "invariant %q is not declared in %s %q")
}

// The findings are sorted because a context holds its own invariants and its own
// slices in collections separate from its aggregates', which loses whether they
// were written before or after them.
func scopedInvariantDiagnostics(found []scopedInvariant, messageFormat string) []*diagnostic.Entry {
	diags := make([]*diagnostic.Entry, 0, len(found))
	for _, f := range found {
		diags = append(diags, errorAt(f.pos, messageFormat, f.name, f.scope.kind, f.scope.name))
	}
	sortInDeclarationOrder(diags)

	return diags
}

func referenceDiagnostics(slice *ast.Slice, index *modelIndex) []*diagnostic.Entry {
	var diags []*diagnostic.Entry
	for _, auto := range slice.Automations {
		diags = appendUndeclaredRef(diags, "target context", auto.TargetContext, auto.TargetContextPos, index.contextNames)
		diags = appendUndeclaredRef(diags, "command", auto.Command, auto.CommandPos, index.commandNames)
		diags = appendUndeclaredRef(diags, "event", auto.OnEvent, auto.OnEventPos, index.eventNames)
		// A trigger's and a translation's `reads` stay unchecked on purpose:
		// existing models name views in them that no slice declares.
		diags = appendUndeclaredRef(diags, "view", auto.Reads, auto.ReadsPos, index.viewNames)
	}
	for _, tr := range slice.Translations {
		diags = appendUndeclaredRef(diags, "command", tr.Command, tr.CommandPos, index.commandNames)
	}
	for _, v := range slice.Views {
		for _, sub := range v.Subscribes {
			diags = appendUndeclaredRef(diags, "event", sub, v.NamePos, index.eventNames)
		}
	}
	for _, f := range slice.Flows {
		diags = appendUndeclaredRef(diags, "event", f.EventName, f.EventPos, index.eventNames)
	}
	for _, spec := range slice.Specs {
		diags = append(diags, specDiagnostics(spec, index)...)
	}

	return diags
}

func appendUndeclaredRef(diags []*diagnostic.Entry, kind, name string, pos ast.Position, declared map[string]bool) []*diagnostic.Entry {
	if name == "" || declared[name] {
		return diags
	}

	return append(diags, errorAt(pos, "%s %q does not exist", kind, name))
}

type undeclaredSpecReference struct {
	element *ast.SpecElement
	kind    string
}

// The references are sorted because a spec's parts are written in any order and
// the AST holds each of them in a field of its own, which loses which part the
// author wrote first.
func specDiagnostics(spec *ast.Spec, index *modelIndex) []*diagnostic.Entry {
	undeclared := undeclaredSpecEvents(spec.Given, index)
	if ref := spec.When; ref != nil && !index.declaresCommandOrEvent(ref.Name) {
		undeclared = append(undeclared, undeclaredSpecReference{element: ref, kind: "command"})
	}
	if outcome, ok := spec.Then.(*ast.ThenEvents); ok {
		undeclared = append(undeclared, undeclaredSpecEvents(outcome.Events, index)...)
	}
	if outcome, ok := spec.Then.(*ast.ThenView); ok {
		if !index.viewNames[outcome.ViewName] {
			undeclared = append(undeclared, undeclaredSpecReference{
				element: &ast.SpecElement{Name: outcome.ViewName, NamePos: outcome.ViewPos},
				kind:    "view",
			})
		}
	}
	if outcome, ok := spec.Then.(*ast.ThenCommand); ok {
		if !index.commandNames[outcome.CommandName] {
			undeclared = append(undeclared, undeclaredSpecReference{
				element: &ast.SpecElement{Name: outcome.CommandName, NamePos: outcome.CommandPos},
				kind:    "command",
			})
		}
	}

	diags := make([]*diagnostic.Entry, 0, len(undeclared))
	for _, ref := range undeclared {
		diags = append(diags, errorAt(ref.element.NamePos, "%s %q does not exist", ref.kind, ref.element.Name))
	}
	sortInDeclarationOrder(diags)

	return diags
}

func undeclaredSpecEvents(refs []*ast.SpecElement, index *modelIndex) []undeclaredSpecReference {
	var undeclared []undeclaredSpecReference
	for _, ref := range refs {
		if !index.eventNames[ref.Name] {
			undeclared = append(undeclared, undeclaredSpecReference{element: ref, kind: "event"})
		}
	}

	return undeclared
}

func scheduleExpressionDiagnostics(modelSlices []*ast.Slice) []*diagnostic.Entry {
	var diags []*diagnostic.Entry
	for _, slice := range modelSlices {
		for _, auto := range slice.Automations {
			if auto.Schedule == "" || isWellFormedSchedule(auto.Schedule) {
				continue
			}
			diags = append(diags, errorAt(auto.SchedulePos,
				"schedule expression %q is neither a Go duration nor a five-field cron expression", auto.Schedule))
		}
	}

	return diags
}

// Only the shape is judged: a value the field's position does not allow — a
// minute of 99, a weekday spelled XYZ — is the scheduler's to reject, not the
// model's.
func isWellFormedSchedule(expression string) bool {
	return isGoDuration(expression) || isCronExpression(expression)
}

func isGoDuration(expression string) bool {
	_, err := time.ParseDuration(expression)

	return err == nil
}

const (
	cronFieldCount   = 5
	cronNumberOrName = `(?:\d+|[A-Za-z]{3})`
	cronFieldTerm    = `(?:\*|` + cronNumberOrName + `)(?:-` + cronNumberOrName + `)?(?:/\d+)?`
)

var cronFieldPattern = regexp.MustCompile(`^` + cronFieldTerm + `(?:,` + cronFieldTerm + `)*$`)

func isCronExpression(expression string) bool {
	fields := strings.Fields(expression)
	if len(fields) != cronFieldCount {
		return false
	}

	for _, field := range fields {
		if !cronFieldPattern.MatchString(field) {
			return false
		}
	}

	return true
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
	}

	return diags
}

package query

import (
	"fmt"
	"slices"
	"strings"

	"github.com/hpcsc/emod/internal/ast"
)

const (
	KindInvariant   Kind = "invariant"
	KindTrigger     Kind = "trigger"
	KindAutomation  Kind = "automation"
	KindTranslation Kind = "translation"
)

var ReferenceKinds = []Kind{KindCommand, KindEvent, KindView, KindInvariant}

type Group string

const (
	GroupProducedBy      Group = "produced by"
	GroupReadBy          Group = "read by"
	GroupIssuedBy        Group = "issued by"
	GroupProduces        Group = "produces"
	GroupRefusedBy       Group = "refused by"
	GroupBuiltFrom       Group = "built from"
	GroupRefusedCommands Group = "refused commands"
	GroupNamedInSpecs    Group = "named in specs"
)

func GroupsOf(kind Kind) []Group {
	switch kind {
	case KindEvent:
		return []Group{GroupProducedBy, GroupReadBy, GroupNamedInSpecs}
	case KindCommand:
		return []Group{GroupIssuedBy, GroupProduces, GroupRefusedBy, GroupNamedInSpecs}
	case KindView:
		return []Group{GroupBuiltFrom, GroupReadBy, GroupNamedInSpecs}
	case KindInvariant:
		return []Group{GroupRefusedCommands, GroupNamedInSpecs}
	}
	return nil
}

// An invariant has no slice: a context or an aggregate declares it.
type NameReferences struct {
	Kind       Kind        `json:"kind"`
	Name       string      `json:"name"`
	Context    string      `json:"context"`
	Aggregate  string      `json:"aggregate,omitempty"`
	Slice      string      `json:"slice,omitempty"`
	Line       int         `json:"line"`
	References []Reference `json:"references"`
}

// A trigger issues the commands of its slice with no DSL word, so that link has
// the keyword "issues".
type Reference struct {
	Group   Group    `json:"group"`
	Keyword string   `json:"keyword"`
	Line    int      `json:"line"`
	Element *Element `json:"element,omitempty"`
	Slice   string   `json:"slice"`
	Spec    string   `json:"spec,omitempty"`
}

type Element struct {
	Kind Kind   `json:"kind"`
	Name string `json:"name"`
}

type UnknownNameError struct {
	Name string
	// Kinds are the kinds that the caller looked in. No kind means each kind of
	// element.
	Kinds      []Kind
	DeclaredAs []Kind
	Closest    []string
}

func (e *UnknownNameError) Error() string {
	looked := "element"
	if len(e.Kinds) > 0 {
		looked = joinKinds(e.Kinds, "or")
	}
	if len(e.DeclaredAs) > 0 {
		return fmt.Sprintf("the model declares %q as %s, not as %s", e.Name, joinKinds(e.DeclaredAs, "and"), looked)
	}

	message := fmt.Sprintf("the model declares no %s named %q", looked, e.Name)
	if len(e.Closest) > 0 {
		message += "; the closest names are " + strings.Join(e.Closest, ", ")
	}
	return message
}

func joinKinds(kinds []Kind, last string) string {
	words := make([]string, len(kinds))
	for i, kind := range kinds {
		words[i] = string(kind)
	}
	if len(words) < 2 {
		return strings.Join(words, "")
	}
	return strings.Join(words[:len(words)-1], ", ") + " " + last + " " + words[len(words)-1]
}

// An empty kind finds the name as each kind in ReferenceKinds.
func References(model *ast.Model, name string, kind Kind) ([]NameReferences, error) {
	kinds := ReferenceKinds
	if kind != "" {
		kinds = []Kind{kind}
	}

	var found []NameReferences
	var foundPos []ast.Position
	add := func(result NameReferences, pos ast.Position) {
		found = append(found, result)
		foundPos = append(foundPos, pos)
	}

	for _, k := range kinds {
		if k == KindInvariant {
			for _, scope := range InvariantScopes(model) {
				if inv, ok := scope.DeclarationOf(name); ok {
					add(invariantReferences(scope, inv), inv.NamePos)
				}
			}
			continue
		}
		if decl, ok := lastDeclaration(model, k, name); ok {
			add(declarationReferences(model, decl), decl.Pos)
		}
	}

	if len(found) == 0 {
		return nil, unknownName(model, name, kind)
	}
	return inSourceOrder(found, foundPos), nil
}

// inSourceOrder sorts items by the position at the same index in pos.
func inSourceOrder[T any](items []T, pos []ast.Position) []T {
	order := make([]int, len(items))
	for i := range order {
		order[i] = i
	}
	slices.SortStableFunc(order, func(a, b int) int { return pos[a].Compare(pos[b]) })

	sorted := make([]T, len(items))
	for i, at := range order {
		sorted[i] = items[at]
	}
	return sorted
}

// lastDeclaration keeps the last of two declarations of one name, as
// DeclaredNames does, so the CLI and the editor resolve the same one.
func lastDeclaration(model *ast.Model, kind Kind, name string) (Declaration, bool) {
	var last Declaration
	found := false
	for _, decl := range Declarations(model) {
		if decl.Kind == kind && decl.Name == name {
			last, found = decl, true
		}
	}
	return last, found
}

type referenceList struct {
	refs []Reference
	pos  []ast.Position
}

func (l *referenceList) add(group Group, keyword string, pos ast.Position, element *Element, slice *ast.Slice) {
	l.refs = append(l.refs, Reference{Group: group, Keyword: keyword, Line: pos.Line, Element: element, Slice: slice.Name})
	l.pos = append(l.pos, pos)
}

func (l *referenceList) addSpec(keyword string, pos ast.Position, slice *ast.Slice, spec *ast.Spec) {
	l.refs = append(l.refs, Reference{Group: GroupNamedInSpecs, Keyword: keyword, Line: pos.Line, Slice: slice.Name, Spec: spec.Name})
	l.pos = append(l.pos, pos)
}

func element(kind Kind, name string) *Element {
	return &Element{Kind: kind, Name: name}
}

func declarationReferences(model *ast.Model, decl Declaration) NameReferences {
	var list referenceList
	for _, slice := range model.AllSlices() {
		switch decl.Kind {
		case KindEvent:
			addEventReferences(&list, slice, decl.Name)
		case KindCommand:
			addCommandReferences(&list, slice, decl.Name)
		case KindView:
			addViewReferences(&list, slice, decl.Name)
		}
	}

	return NameReferences{
		Kind:       decl.Kind,
		Name:       decl.Name,
		Context:    decl.Context,
		Aggregate:  decl.Aggregate,
		Slice:      decl.Slice,
		Line:       decl.Pos.Line,
		References: inSourceOrder(list.refs, list.pos),
	}
}

func addEventReferences(list *referenceList, slice *ast.Slice, name string) {
	for _, f := range slice.Flows {
		if f.EventName == name {
			list.add(GroupProducedBy, "flow", f.EventPos, element(KindCommand, f.CommandName), slice)
		}
	}
	for _, tr := range slice.Translations {
		if tr.Event != nil && tr.Event.Name == name && tr.Command != "" {
			list.add(GroupProducedBy, "translation", tr.CommandPos, element(KindCommand, tr.Command), slice)
		}
	}
	for _, v := range slice.Views {
		for i, sub := range v.Subscribes {
			if sub == name && i < len(v.SubscribesPos) {
				list.add(GroupReadBy, "subscribes", v.SubscribesPos[i], element(KindView, v.Name), slice)
			}
		}
	}
	for _, auto := range slice.Automations {
		if auto.OnEvent == name {
			list.add(GroupReadBy, "on", auto.OnEventPos, element(KindAutomation, auto.Name), slice)
		}
	}
	for _, spec := range slice.Specs {
		for _, given := range spec.Given {
			if given.Name == name {
				list.addSpec("given", given.NamePos, slice, spec)
			}
		}
		if spec.When != nil && spec.When.Name == name {
			list.addSpec("when", spec.When.NamePos, slice, spec)
		}
		if then, ok := spec.Then.(*ast.ThenEvents); ok {
			for _, evt := range then.Events {
				if evt.Name == name {
					list.addSpec("then", evt.NamePos, slice, spec)
				}
			}
		}
	}
}

func addCommandReferences(list *referenceList, slice *ast.Slice, name string) {
	if trigger := slice.Trigger; trigger != nil {
		for _, cmd := range slice.Commands {
			if cmd.Name == name {
				list.add(GroupIssuedBy, "issues", trigger.NamePos, element(KindTrigger, trigger.Name), slice)
			}
		}
	}
	for _, auto := range slice.Automations {
		if auto.Command == name {
			list.add(GroupIssuedBy, "command", auto.CommandPos, element(KindAutomation, auto.Name), slice)
		}
	}
	for _, tr := range slice.Translations {
		if tr.Command != name {
			continue
		}
		list.add(GroupIssuedBy, "command", tr.CommandPos, element(KindTranslation, tr.Name), slice)
		if tr.Event != nil {
			list.add(GroupProduces, "translation", tr.CommandPos, element(KindEvent, tr.Event.Name), slice)
		}
	}
	for _, f := range slice.Flows {
		if f.CommandName == name {
			list.add(GroupProduces, "flow", f.CommandPos, element(KindEvent, f.EventName), slice)
		}
	}
	for _, rejection := range slice.Rejections {
		if rejection.CommandName == name {
			list.add(GroupRefusedBy, "rejected", rejection.CommandPos, element(KindInvariant, rejection.InvariantName), slice)
		}
	}
	for _, spec := range slice.Specs {
		if spec.When != nil && spec.When.Name == name {
			list.addSpec("when", spec.When.NamePos, slice, spec)
		}
		if then, ok := spec.Then.(*ast.ThenCommand); ok && then.CommandName == name {
			list.addSpec("then", then.CommandPos, slice, spec)
		}
	}
}

func addViewReferences(list *referenceList, slice *ast.Slice, name string) {
	for _, v := range slice.Views {
		if v.Name != name {
			continue
		}
		for i, sub := range v.Subscribes {
			if i < len(v.SubscribesPos) {
				list.add(GroupBuiltFrom, "subscribes", v.SubscribesPos[i], element(KindEvent, sub), slice)
			}
		}
	}
	if trigger := slice.Trigger; trigger != nil {
		for i, view := range trigger.Reads {
			if view == name {
				list.add(GroupReadBy, "reads", trigger.ReadsPos[i], element(KindTrigger, trigger.Name), slice)
			}
		}
	}
	for _, auto := range slice.Automations {
		if auto.Reads == name {
			list.add(GroupReadBy, "reads", auto.ReadsPos, element(KindAutomation, auto.Name), slice)
		}
	}
	for _, tr := range slice.Translations {
		if tr.Reads == name {
			list.add(GroupReadBy, "reads", tr.ReadsPos, element(KindTranslation, tr.Name), slice)
		}
	}
	for _, spec := range slice.Specs {
		if then, ok := spec.Then.(*ast.ThenView); ok && then.ViewName == name {
			list.addSpec("then", then.ViewPos, slice, spec)
		}
	}
}

func invariantReferences(scope InvariantScope, inv *ast.Invariant) NameReferences {
	var list referenceList
	for _, slice := range scope.Slices {
		for _, rejection := range slice.Rejections {
			if rejection.InvariantName == inv.Name {
				list.add(GroupRefusedCommands, "rejected", rejection.InvariantPos, element(KindCommand, rejection.CommandName), slice)
			}
		}
		for _, spec := range slice.Specs {
			if then, ok := spec.Then.(*ast.ThenRejected); ok && then.InvariantName == inv.Name {
				list.addSpec("then", then.InvariantPos, slice, spec)
			}
		}
	}

	return NameReferences{
		Kind:       KindInvariant,
		Name:       inv.Name,
		Context:    scope.Context,
		Aggregate:  scope.Aggregate,
		Line:       inv.NamePos.Line,
		References: inSourceOrder(list.refs, list.pos),
	}
}

func unknownName(model *ast.Model, name string, kind Kind) error {
	unknown := &UnknownNameError{Name: name, Kinds: ReferenceKinds}
	if kind != "" {
		unknown.Kinds = []Kind{kind}
		for _, other := range ReferenceKinds {
			if other == kind {
				continue
			}
			if declares(model, other, name) {
				unknown.DeclaredAs = append(unknown.DeclaredAs, other)
			}
		}
		if len(unknown.DeclaredAs) > 0 {
			return unknown
		}
	}

	unknown.Closest = closestNames(name, namesOf(model, kind))
	return unknown
}

func declares(model *ast.Model, kind Kind, name string) bool {
	return slices.Contains(namesOf(model, kind), name)
}

func namesOf(model *ast.Model, kind Kind) []string {
	var names []string
	for _, decl := range Declarations(model) {
		if decl.Kind == KindContext || (kind != "" && decl.Kind != kind) {
			continue
		}
		names = append(names, decl.Name)
	}
	if kind == "" || kind == KindInvariant {
		for _, scope := range InvariantScopes(model) {
			for _, inv := range scope.Declared {
				names = append(names, inv.Name)
			}
		}
	}
	return names
}

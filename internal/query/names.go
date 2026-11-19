package query

import (
	"slices"

	"github.com/hpcsc/emod/internal/ast"
)

type Kind string

const (
	KindCommand Kind = "command"
	KindEvent   Kind = "event"
	KindView    Kind = "view"
	KindContext Kind = "context"
)

// Declaration is a name that the model resolves across the whole file. Pos
// points at the first character of the name, also where the AST stores the
// opening quote, so a caller measures the name and not its delimiters.
type Declaration struct {
	Kind      Kind
	Name      string
	Pos       ast.Position
	Context   string
	Aggregate string
	Slice     string
}

// An invariant resolves in one scope only, so InvariantScopes lists the
// invariants and Declarations does not.
func Declarations(model *ast.Model) []Declaration {
	var decls []Declaration
	for _, screen := range model.Screens {
		decls = append(decls, Declaration{Kind: KindScreen, Name: screen.Name, Pos: afterQuote(screen.NamePos)})
	}
	for _, ctx := range model.Contexts {
		decls = append(decls, Declaration{Kind: KindContext, Name: ctx.Name, Pos: afterQuote(ctx.NamePos), Context: ctx.Name})
	}

	for _, ref := range model.SliceRefs() {
		add := func(kind Kind, name string, pos ast.Position) {
			decls = append(decls, Declaration{
				Kind:      kind,
				Name:      name,
				Pos:       afterQuote(pos),
				Context:   ref.Context.Name,
				Aggregate: aggregateName(ref),
				Slice:     ref.Slice.Name,
			})
		}
		for _, cmd := range ref.Slice.Commands {
			add(KindCommand, cmd.Name, cmd.NamePos)
		}
		for _, evt := range ref.Slice.Events {
			add(KindEvent, evt.Name, evt.NamePos)
		}
		for _, tr := range ref.Slice.Translations {
			if tr.Event != nil {
				add(KindEvent, tr.Event.Name, tr.Event.NamePos)
			}
		}
		for _, v := range ref.Slice.Views {
			add(KindView, v.Name, v.NamePos)
		}
	}

	return decls
}

func aggregateName(ref ast.SliceRef) string {
	if ref.Aggregate == nil {
		return ""
	}
	return ref.Aggregate.Name
}

// afterQuote moves a position stored at a quoted name's opening quote onto the
// name's first character, which is where every measurement of the name starts.
func afterQuote(pos ast.Position) ast.Position {
	pos.Column++
	return pos
}

// A name that two declarations of one kind share keeps the last one.
type DeclaredNames map[Kind]map[string]ast.Position

func NewDeclaredNames(model *ast.Model) DeclaredNames {
	declared := DeclaredNames{
		KindCommand: {},
		KindEvent:   {},
		KindView:    {},
		KindScreen:  {},
		KindContext: {},
	}

	for _, decl := range Declarations(model) {
		declared[decl.Kind][decl.Name] = decl.Pos
	}

	return declared
}

func (d DeclaredNames) PositionOf(kind Kind, name string) (ast.Position, bool) {
	pos, ok := d[kind][name]
	return pos, ok
}

type NameSite struct {
	Kind Kind
	Name string
	Pos  ast.Position
}

func NameSites(model *ast.Model) []NameSite {
	var sites []NameSite
	add := func(kind Kind, name string, pos ast.Position) {
		sites = append(sites, NameSite{Kind: kind, Name: name, Pos: pos})
	}

	for _, slice := range model.AllSlices() {
		for _, v := range slice.Views {
			for i, sub := range v.Subscribes {
				if i < len(v.SubscribesPos) {
					add(KindEvent, sub, v.SubscribesPos[i])
				}
			}
		}
		for _, auto := range slice.Automations {
			add(KindEvent, auto.OnEvent, auto.OnEventPos)
			add(KindView, auto.Reads, auto.ReadsPos)
			add(KindCommand, auto.Command, auto.CommandPos)
			add(KindContext, auto.TargetContext, auto.TargetContextPos)
		}
		for _, tr := range slice.Translations {
			add(KindView, tr.Reads, tr.ReadsPos)
			add(KindCommand, tr.Command, tr.CommandPos)
		}
		for _, appearance := range slice.Appearances {
			add(KindScreen, appearance.Screen, appearance.ScreenPos)
			for i, view := range appearance.Reads {
				add(KindView, view, appearance.ReadsPos[i])
			}
			for i, command := range appearance.Commands {
				add(KindCommand, command, appearance.CommandsPos[i])
			}
		}
		for _, f := range slice.Flows {
			add(KindCommand, f.CommandName, f.CommandPos)
			add(KindEvent, f.EventName, f.EventPos)
		}
		for _, spec := range slice.Specs {
			addSpecSites(add, spec)
		}
		for _, rejection := range slice.Rejections {
			// Only the command. The other name a rejection edge carries is an
			// invariant, which resolves in one scope rather than model-wide and
			// so is stated by InvariantScopes instead.
			add(KindCommand, rejection.CommandName, rejection.CommandPos)
		}
	}

	// Every kind of site is gathered from a collection of its own, so the order
	// they were appended in is the AST's field order rather than the order an
	// author wrote them: a slice's flow block sits in the list ahead of a spec
	// written above it, and a spec's given, when and then are three more fields
	// admitting any order. Sorting recovers the document order find-references
	// and go-to-definition report in.
	slices.SortStableFunc(sites, func(a, b NameSite) int {
		return a.Pos.Compare(b.Pos)
	})

	return sites
}

func addSpecSites(add func(Kind, string, ast.Position), spec *ast.Spec) {
	for _, given := range spec.Given {
		add(KindEvent, given.Name, given.NamePos)
	}
	if spec.When != nil {
		// A command slice's when names a command while an automation slice's
		// names the triggering event, so the site is a site of both kinds
		// and resolves as whichever one the model declares.
		add(KindCommand, spec.When.Name, spec.When.NamePos)
		add(KindEvent, spec.When.Name, spec.When.NamePos)
	}
	switch then := spec.Then.(type) {
	case *ast.ThenEvents:
		for _, evt := range then.Events {
			add(KindEvent, evt.Name, evt.NamePos)
		}
	case *ast.ThenView:
		add(KindView, then.ViewName, then.ViewPos)
	case *ast.ThenCommand:
		add(KindCommand, then.CommandName, then.CommandPos)
	}
}

// InvariantScope is one resolution scope for invariant names. A context's own
// invariants and each of its aggregates' are separate scopes, exactly as
// internal/validator resolves them: a name declared in one neither hides nor
// resolves against the same name in another, not even between an aggregate and
// the context enclosing it. A flat model-wide map cannot express that — one
// declaration would silently win, and a caller would resolve a name that
// `emod validate` reports as unresolved.
type InvariantScope struct {
	Context   string
	Aggregate string
	Declared  []*ast.Invariant
	Sites     []InvariantSite
	Slices    []*ast.Slice
}

type InvariantSite struct {
	Name string
	Pos  ast.Position
}

func InvariantScopes(model *ast.Model) []InvariantScope {
	var scopes []InvariantScope
	for _, ctx := range model.Contexts {
		scopes = append(scopes, newInvariantScope(ctx.Name, "", ctx.Invariants, ctx.Slices))
		for _, agg := range ctx.Aggregates {
			scopes = append(scopes, newInvariantScope(ctx.Name, agg.Name, agg.Invariants, agg.Slices))
		}
	}

	return scopes
}

func newInvariantScope(context, aggregate string, invariants []*ast.Invariant, sliceList []*ast.Slice) InvariantScope {
	scope := InvariantScope{Context: context, Aggregate: aggregate, Declared: invariants, Slices: sliceList}
	add := func(name string, pos ast.Position) {
		scope.Sites = append(scope.Sites, InvariantSite{Name: name, Pos: pos})
	}

	for _, slice := range sliceList {
		for _, spec := range slice.Specs {
			if rejected, ok := spec.Then.(*ast.ThenRejected); ok {
				add(rejected.InvariantName, rejected.InvariantPos)
			}
		}
		for _, rejection := range slice.Rejections {
			add(rejection.InvariantName, rejection.InvariantPos)
		}
	}

	// A slice's specs and its rejection edges are separate collections, so the
	// order they were appended in is not the order they were written.
	slices.SortStableFunc(scope.Sites, func(a, b InvariantSite) int {
		return a.Pos.Compare(b.Pos)
	})

	return scope
}

func (s InvariantScope) Name() string {
	return ScopeName(s.Context, s.Aggregate)
}

func ScopeName(context, aggregate string) string {
	if aggregate == "" {
		return context
	}
	return context + " > " + aggregate
}

func (s InvariantScope) DeclarationOf(name string) (*ast.Invariant, bool) {
	for _, inv := range s.Declared {
		if inv.Name == name {
			return inv, true
		}
	}
	return nil, false
}

package lsp

import (
	"slices"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/lexer"
	"github.com/hpcsc/emod/internal/parser"
)

func parseModel(text, uri string) (*ast.Model, []*lexer.Token) {
	tokens, _ := lexer.Scan(text, uri)
	model, _ := parser.New(tokens, uri).Parse()
	return model, tokens
}

func declaredAggregates(model *ast.Model) []*ast.Aggregate {
	var aggregates []*ast.Aggregate
	for _, ctx := range model.Contexts {
		aggregates = append(aggregates, ctx.Aggregates...)
	}
	return aggregates
}

// constructDecl is one declaration the editor can describe: the kind it is, the
// enclosing constructs that scope it, and the parts a description of it draws on.
// namePos points at the first character of the name even where the AST stores the
// opening quote, so a caller measures the name and not its delimiters.
type constructDecl struct {
	kind        string
	name        string
	namePos     ast.Position
	scope       string
	description string
	fields      []*ast.Field
	subscribes  []string
}

func declaredConstructs(model *ast.Model) []constructDecl {
	var decls []constructDecl
	addQuoted := func(kind string, name string, pos ast.Position, scope, description string) {
		decls = append(decls, constructDecl{
			kind:        kind,
			name:        name,
			namePos:     afterQuote(pos),
			scope:       scope,
			description: description,
		})
	}

	if model.Name != "" {
		addQuoted("Model", model.Name, model.NamePos, "", model.Description)
	}
	for _, actor := range model.Actors {
		addQuoted("Actor", actor.Name, actor.NamePos, "", actor.Description)
	}
	for _, ctx := range model.Contexts {
		addQuoted("Context", ctx.Name, ctx.NamePos, "", ctx.Description)
		for _, agg := range ctx.Aggregates {
			addQuoted("Aggregate", agg.Name, agg.NamePos, ctx.Name, agg.Description)
		}
	}

	for _, scoped := range model.SliceRefs() {
		scope := scopeOf(scoped)
		addQuoted("Slice", scoped.Slice.Name, scoped.Slice.NamePos, scope, scoped.Slice.Description)
		if trigger := scoped.Slice.Trigger; trigger != nil {
			addQuoted("Trigger", trigger.Name, trigger.NamePos, scope, trigger.Description)
		}
		for _, cmd := range scoped.Slice.Commands {
			decls = append(decls, constructDecl{
				kind:        "Command",
				name:        cmd.Name,
				namePos:     cmd.NamePos,
				scope:       scope,
				description: cmd.Description,
			})
		}
		for _, evt := range scoped.Slice.Events {
			decls = append(decls, eventDecl(evt, scope))
		}
		for _, v := range scoped.Slice.Views {
			decls = append(decls, constructDecl{
				kind:        "View",
				name:        v.Name,
				namePos:     v.NamePos,
				scope:       scope,
				description: v.Description,
				subscribes:  v.Subscribes,
			})
		}
		for _, auto := range scoped.Slice.Automations {
			decls = append(decls, constructDecl{
				kind:        "Automation",
				name:        auto.Name,
				namePos:     auto.NamePos,
				scope:       scope,
				description: auto.Description,
			})
		}
		for _, tr := range scoped.Slice.Translations {
			decls = append(decls, constructDecl{
				kind:        "Translation",
				name:        tr.Name,
				namePos:     tr.NamePos,
				scope:       scope,
				description: tr.Description,
			})
			if tr.Event != nil {
				decls = append(decls, eventDecl(tr.Event, scope))
			}
		}
	}

	return decls
}

func eventDecl(evt *ast.Event, scope string) constructDecl {
	return constructDecl{
		kind:        "Event",
		name:        evt.Name,
		namePos:     evt.NamePos,
		scope:       scope,
		description: evt.Description,
		fields:      evt.Fields,
	}
}

// scopeOf names the constructs a slice sits inside, dropping the aggregate
// segment for the slices a `mode dcb` context declares directly.
func scopeOf(scoped ast.SliceRef) string {
	if scoped.Aggregate == nil {
		return scoped.Context.Name
	}
	return scoped.Context.Name + " > " + scoped.Aggregate.Name
}

// afterQuote moves a position stored at a quoted name's opening quote onto the
// name's first character, which is where every measurement of the name starts.
func afterQuote(pos ast.Position) ast.Position {
	pos.Column++
	return pos
}

// invariantScope is one resolution scope for invariant names. A context's own
// invariants and each of its aggregates' are separate scopes, exactly as
// internal/validator resolves them: a name declared in one neither hides nor
// resolves against the same name in another, not even between an aggregate and
// the context enclosing it. A flat model-wide map cannot express that — one
// declaration would silently win, and the editor would jump to a declaration
// `emod validate` reports as unresolved.
type invariantScope struct {
	name       string
	declared   []*ast.Invariant
	references []invariantRef
}

// invariantRef is one site naming an invariant rather than declaring it.
type invariantRef struct {
	name string
	pos  ast.Position
}

func invariantScopes(model *ast.Model) []invariantScope {
	var scopes []invariantScope
	for _, ctx := range model.Contexts {
		scopes = append(scopes, newInvariantScope(ctx.Name, ctx.Invariants, ctx.Slices))
		for _, agg := range ctx.Aggregates {
			scopes = append(scopes, newInvariantScope(ctx.Name+" > "+agg.Name, agg.Invariants, agg.Slices))
		}
	}

	return scopes
}

func newInvariantScope(name string, invariants []*ast.Invariant, slices []*ast.Slice) invariantScope {
	scope := invariantScope{name: name, declared: invariants}
	for _, slice := range slices {
		for _, spec := range slice.Specs {
			if rejected, ok := spec.Then.(*ast.ThenRejected); ok {
				scope.references = append(scope.references, invariantRef{
					name: rejected.InvariantName,
					pos:  rejected.InvariantPos,
				})
			}
		}
	}

	return scope
}

// referencedNameAt reports the invariant name a site in this scope names under
// the cursor. A cursor on the declaration itself is not a reference, which is
// what keeps go-to-definition returning nil where it already does.
func (s invariantScope) referencedNameAt(at cursor) (string, bool) {
	for _, ref := range s.references {
		if at.onName(ref.pos, ref.name) {
			return ref.name, true
		}
	}
	return "", false
}

func (s invariantScope) declaredNameAt(at cursor) (string, bool) {
	for _, inv := range s.declared {
		if at.onName(inv.NamePos, inv.Name) {
			return inv.Name, true
		}
	}
	return "", false
}

// invariantNamesInScopeAt names the invariants an identifier written on the
// given 1-based line resolves against: those of the aggregate holding the line,
// or of the context itself where the line sits outside every aggregate.
func invariantNamesInScopeAt(model *ast.Model, line int) []string {
	for _, ctx := range model.Contexts {
		if !blockHolds(ctx.OpenPos, ctx.ClosePos, line) {
			continue
		}
		for _, agg := range ctx.Aggregates {
			if blockHolds(agg.OpenPos, agg.ClosePos, line) {
				return invariantNames(agg.Invariants)
			}
		}
		return invariantNames(ctx.Invariants)
	}

	return nil
}

// blockHolds reports whether a line falls inside a block. A block the author has
// not closed yet carries no closing position, and while it is being typed the
// cursor is exactly what sits inside it, so it runs to the end of the document.
func blockHolds(open, close ast.Position, line int) bool {
	return open.Line <= line && (close.Line == 0 || line <= close.Line)
}

func invariantNames(invariants []*ast.Invariant) []string {
	names := make([]string, 0, len(invariants))
	for _, inv := range invariants {
		names = append(names, inv.Name)
	}
	return names
}

func (s invariantScope) declarationOf(name string) (*ast.Invariant, bool) {
	for _, inv := range s.declared {
		if inv.Name == name {
			return inv, true
		}
	}
	return nil, false
}

type nameKind int

const (
	commandName nameKind = iota
	eventName
	viewName
	contextName
)

type nameDecl struct {
	kind nameKind
	name string
	pos  ast.Position
}

func declarationsIn(model *ast.Model) []nameDecl {
	var decls []nameDecl
	add := func(kind nameKind, name string, pos ast.Position) {
		decls = append(decls, nameDecl{kind: kind, name: name, pos: pos})
	}

	for _, ctx := range model.Contexts {
		add(contextName, ctx.Name, ctx.NamePos)
	}
	for _, slice := range model.AllSlices() {
		for _, cmd := range slice.Commands {
			add(commandName, cmd.Name, cmd.NamePos)
		}
		for _, evt := range slice.Events {
			add(eventName, evt.Name, evt.NamePos)
		}
		for _, v := range slice.Views {
			add(viewName, v.Name, v.NamePos)
		}
	}

	return decls
}

func declaredNamesInOrder(model *ast.Model, kind nameKind) []string {
	var names []string
	for _, decl := range declarationsIn(model) {
		if decl.kind == kind {
			names = append(names, decl.name)
		}
	}
	return names
}

type declaredNames map[nameKind]map[string]ast.Position

func newDeclaredNames(model *ast.Model) declaredNames {
	declared := declaredNames{
		commandName: {},
		eventName:   {},
		viewName:    {},
		contextName: {},
	}

	for _, decl := range declarationsIn(model) {
		declared[decl.kind][decl.name] = decl.pos
	}

	return declared
}

func (d declaredNames) positionOf(kind nameKind, name string) (ast.Position, bool) {
	pos, ok := d[kind][name]
	return pos, ok
}

// nameRef is one site that mentions a name rather than declaring it.
type nameRef struct {
	kind nameKind
	name string
	pos  ast.Position
}

func referencesIn(model *ast.Model) []nameRef {
	var refs []nameRef
	add := func(kind nameKind, name string, pos ast.Position) {
		refs = append(refs, nameRef{kind: kind, name: name, pos: pos})
	}

	for _, slice := range model.AllSlices() {
		for _, v := range slice.Views {
			for i, sub := range v.Subscribes {
				if i < len(v.SubscribesPos) {
					add(eventName, sub, v.SubscribesPos[i])
				}
			}
		}
		for _, auto := range slice.Automations {
			add(eventName, auto.OnEvent, auto.OnEventPos)
			add(viewName, auto.Reads, auto.ReadsPos)
			add(commandName, auto.Command, auto.CommandPos)
			add(contextName, auto.TargetContext, auto.TargetContextPos)
		}
		for _, tr := range slice.Translations {
			add(viewName, tr.Reads, tr.ReadsPos)
			add(commandName, tr.Command, tr.CommandPos)
		}
		if slice.Trigger != nil {
			add(viewName, slice.Trigger.Reads, slice.Trigger.ReadsPos)
		}
		for _, f := range slice.Flows {
			add(commandName, f.CommandName, f.CommandPos)
			add(eventName, f.EventName, f.EventPos)
		}
		for _, spec := range slice.Specs {
			addSpecReferences(add, spec)
		}
	}

	// Every kind of site is gathered from a collection of its own, so the order
	// they were appended in is the AST's field order rather than the order an
	// author wrote them: a slice's flow block sits in the list ahead of a spec
	// written above it, and a spec's given, when and then are three more fields
	// admitting any order. Sorting recovers the document order find-references
	// and go-to-definition report in.
	slices.SortStableFunc(refs, func(a, b nameRef) int {
		return a.pos.Compare(b.pos)
	})

	return refs
}

func addSpecReferences(add func(nameKind, string, ast.Position), spec *ast.Spec) {
	for _, given := range spec.Given {
		add(eventName, given.Name, given.NamePos)
	}
	if spec.When != nil {
		// A command slice's when names a command while an automation slice's
		// names the triggering event, so the site is a reference of both kinds
		// and resolves as whichever one the model declares.
		add(commandName, spec.When.Name, spec.When.NamePos)
		add(eventName, spec.When.Name, spec.When.NamePos)
	}
	switch then := spec.Then.(type) {
	case *ast.ThenEvents:
		for _, evt := range then.Events {
			add(eventName, evt.Name, evt.NamePos)
		}
	case *ast.ThenView:
		add(viewName, then.ViewName, then.ViewPos)
	case *ast.ThenCommand:
		add(commandName, then.CommandName, then.CommandPos)
	}
}

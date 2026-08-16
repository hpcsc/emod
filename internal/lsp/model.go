package lsp

import (
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
	kind       string
	name       string
	namePos    ast.Position
	scope      string
	fields     []*ast.Field
	subscribes []string
}

func declaredConstructs(model *ast.Model) []constructDecl {
	var decls []constructDecl
	addQuoted := func(kind string, name string, pos ast.Position, scope string) {
		decls = append(decls, constructDecl{kind: kind, name: name, namePos: afterQuote(pos), scope: scope})
	}

	if model.Name != "" {
		addQuoted("Model", model.Name, model.NamePos, "")
	}
	for _, actor := range model.Actors {
		addQuoted("Actor", actor.Name, actor.NamePos, "")
	}
	for _, ctx := range model.Contexts {
		addQuoted("Context", ctx.Name, ctx.NamePos, "")
		for _, agg := range ctx.Aggregates {
			addQuoted("Aggregate", agg.Name, agg.NamePos, ctx.Name)
		}
	}

	for _, scoped := range model.SliceRefs() {
		scope := scopeOf(scoped)
		addQuoted("Slice", scoped.Slice.Name, scoped.Slice.NamePos, scope)
		if trigger := scoped.Slice.Trigger; trigger != nil {
			addQuoted("Trigger", trigger.Name, trigger.NamePos, scope)
		}
		for _, cmd := range scoped.Slice.Commands {
			decls = append(decls, constructDecl{kind: "Command", name: cmd.Name, namePos: cmd.NamePos, scope: scope})
		}
		for _, evt := range scoped.Slice.Events {
			decls = append(decls, eventDecl(evt, scope))
		}
		for _, v := range scoped.Slice.Views {
			decls = append(decls, constructDecl{
				kind:       "View",
				name:       v.Name,
				namePos:    v.NamePos,
				scope:      scope,
				subscribes: v.Subscribes,
			})
		}
		for _, auto := range scoped.Slice.Automations {
			decls = append(decls, constructDecl{kind: "Automation", name: auto.Name, namePos: auto.NamePos, scope: scope})
		}
		for _, tr := range scoped.Slice.Translations {
			decls = append(decls, constructDecl{kind: "Translation", name: tr.Name, namePos: tr.NamePos, scope: scope})
			if tr.Event != nil {
				decls = append(decls, eventDecl(tr.Event, scope))
			}
		}
	}

	return decls
}

func eventDecl(evt *ast.Event, scope string) constructDecl {
	return constructDecl{
		kind:    "Event",
		name:    evt.Name,
		namePos: evt.NamePos,
		scope:   scope,
		fields:  evt.Fields,
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
	}

	return refs
}

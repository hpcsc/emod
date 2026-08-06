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

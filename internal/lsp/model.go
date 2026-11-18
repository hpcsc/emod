package lsp

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/oracle"
	"github.com/hpcsc/emod/internal/query"
)

// word is one bare word in the source, which is how a keyword is written.
type word struct {
	text string
	pos  ast.Position
}

func parseModel(text, uri string) (*ast.Model, []word) {
	model, _ := oracle.Parse(text, uri)
	return model, wordsIn(text, uri)
}

func wordsIn(text, uri string) []word {
	tokens, _ := hclsyntax.LexConfig([]byte(text), uri, hcl.InitialPos)

	var words []word
	for _, token := range tokens {
		if token.Type != hclsyntax.TokenIdent {
			continue
		}
		words = append(words, word{
			text: string(token.Bytes),
			pos:  ast.Position{Line: token.Range.Start.Line, Column: token.Range.Start.Column},
		})
	}

	return words
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
				namePos:     afterQuote(cmd.NamePos),
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
				namePos:     afterQuote(v.NamePos),
				scope:       scope,
				description: v.Description,
				subscribes:  v.Subscribes,
			})
		}
		for _, auto := range scoped.Slice.Automations {
			decls = append(decls, constructDecl{
				kind:        "Automation",
				name:        auto.Name,
				namePos:     afterQuote(auto.NamePos),
				scope:       scope,
				description: auto.Description,
			})
		}
		for _, tr := range scoped.Slice.Translations {
			decls = append(decls, constructDecl{
				kind:        "Translation",
				name:        tr.Name,
				namePos:     afterQuote(tr.NamePos),
				scope:       scope,
				description: tr.Description,
			})
			if tr.Event != nil {
				decls = append(decls, eventDecl(tr.Event, scope))
			}
		}
		for _, spec := range scoped.Slice.Specs {
			// A spec accepts no description, so its hover is kind and scope
			// alone; it is a named declaration all the same.
			addQuoted("Spec", spec.Name, spec.NamePos, scope, "")
		}
	}

	return decls
}

func eventDecl(evt *ast.Event, scope string) constructDecl {
	return constructDecl{
		kind:        "Event",
		name:        evt.Name,
		namePos:     afterQuote(evt.NamePos),
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

// referencedNameAt reports the invariant name a site in the scope names under
// the cursor. A cursor on the declaration itself is not a reference, which is
// what keeps go-to-definition returning nil where it already does.
func referencedNameAt(s query.InvariantScope, at cursor) (string, bool) {
	for _, site := range s.Sites {
		if at.onName(site.Pos, site.Name) {
			return site.Name, true
		}
	}
	return "", false
}

func declaredNameAt(s query.InvariantScope, at cursor) (string, bool) {
	for _, inv := range s.Declared {
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

func declaredNamesInOrder(model *ast.Model, kind query.Kind) []string {
	var names []string
	for _, decl := range query.Declarations(model) {
		if decl.Kind == kind {
			names = append(names, decl.Name)
		}
	}
	return names
}

// declaredFieldNames names the fields the command or event called name declares.
// It reads the same declarations name resolution does, so a payload offers
// nothing for a name go-to-definition would not resolve either.
func declaredFieldNames(model *ast.Model, kind query.Kind, name string) []string {
	for _, slice := range model.AllSlices() {
		if kind == query.KindCommand {
			for _, cmd := range slice.Commands {
				if cmd.Name == name {
					return fieldNames(cmd.Fields)
				}
			}
			continue
		}
		for _, evt := range slice.Events {
			if evt.Name == name {
				return fieldNames(evt.Fields)
			}
		}
	}

	return nil
}

func fieldNames(fields []*ast.Field) []string {
	names := make([]string, 0, len(fields))
	for _, f := range fields {
		names = append(names, f.Name)
	}
	return names
}

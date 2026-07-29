// Package glossary turns a parsed model into the vocabulary it defines: every
// named construct paired with the description declared alongside it.
package glossary

import "github.com/hpcsc/emod/internal/ast"

type document struct {
	model    term
	contexts []section
}

type section struct {
	term       term
	aggregates []term
}

type term struct {
	name       string
	definition string
}

func newDocument(model *ast.Model) document {
	doc := document{model: term{name: model.Name, definition: model.Description}}
	for _, ctx := range model.Contexts {
		doc.contexts = append(doc.contexts, section{
			term:       term{name: ctx.Name, definition: ctx.Description},
			aggregates: aggregateTerms(ctx.Aggregates),
		})
	}
	return doc
}

func aggregateTerms(aggregates []*ast.Aggregate) []term {
	terms := make([]term, 0, len(aggregates))
	for _, agg := range aggregates {
		terms = append(terms, term{name: agg.Name, definition: agg.Description})
	}
	return terms
}

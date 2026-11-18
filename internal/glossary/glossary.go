// Package glossary turns a parsed model into the vocabulary it defines: every
// named construct paired with the description declared alongside it.
package glossary

import "github.com/hpcsc/emod/internal/ast"

type document struct {
	term
	Contexts []contextSection `json:"contexts,omitempty"`
}

type contextSection struct {
	term
	Invariants []term             `json:"invariants,omitempty"`
	Aggregates []aggregateSection `json:"aggregates,omitempty"`
	Commands   []term             `json:"commands,omitempty"`
	Events     []term             `json:"events,omitempty"`
	Views      []term             `json:"views,omitempty"`
}

type aggregateSection struct {
	term
	Invariants []term `json:"invariants,omitempty"`
}

type term struct {
	Name string `json:"name"`
	// Description takes no omitempty, unlike the collections above: an
	// undescribed term still carries the key, so a consumer reading the glossary
	// can tell a gap in the vocabulary from a field the document does not have.
	Description string `json:"description"`
}

func newDocument(model *ast.Model) document {
	return document{
		term:     term{Name: model.Name, Description: model.Description},
		Contexts: contextSections(model.Contexts),
	}
}

func contextSections(contexts []*ast.Context) []contextSection {
	var sections []contextSection
	for _, ctx := range contexts {
		slices := ctx.AllSlices()
		sections = append(sections, contextSection{
			term:       term{Name: ctx.Name, Description: ctx.Description},
			Invariants: invariantTerms(ctx.Invariants),
			Aggregates: aggregateSections(ctx.Aggregates),
			Commands:   commandTerms(slices),
			Events:     eventTerms(slices),
			Views:      viewTerms(slices),
		})
	}
	return sections
}

func aggregateSections(aggregates []*ast.Aggregate) []aggregateSection {
	var sections []aggregateSection
	for _, agg := range aggregates {
		sections = append(sections, aggregateSection{
			term:       term{Name: agg.Name, Description: agg.Description},
			Invariants: invariantTerms(agg.Invariants),
		})
	}
	return sections
}

func invariantTerms(invariants []*ast.Invariant) []term {
	var terms []term
	for _, invariant := range invariants {
		terms = append(terms, term{Name: invariant.Name, Description: invariant.Statement})
	}
	return terms
}

func commandTerms(slices []*ast.Slice) []term {
	var terms []term
	for _, slice := range slices {
		for _, cmd := range slice.Commands {
			terms = append(terms, term{Name: cmd.Name, Description: cmd.Description})
		}
	}
	return terms
}

func eventTerms(slices []*ast.Slice) []term {
	var terms []term
	for _, slice := range slices {
		for _, evt := range slice.Events {
			terms = append(terms, term{Name: evt.Name, Description: evt.Description})
		}
		for _, translation := range slice.Translations {
			if translation.Event != nil {
				terms = append(terms, term{Name: translation.Event.Name, Description: translation.Event.Description})
			}
		}
	}
	return terms
}

func viewTerms(slices []*ast.Slice) []term {
	var terms []term
	for _, slice := range slices {
		for _, view := range slice.Views {
			terms = append(terms, term{Name: view.Name, Description: view.Description})
		}
	}
	return terms
}

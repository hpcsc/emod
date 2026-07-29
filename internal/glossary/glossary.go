// Package glossary turns a parsed model into the vocabulary it defines: every
// named construct paired with the description declared alongside it.
package glossary

import "github.com/hpcsc/emod/internal/ast"

const (
	commandsHeading = "Commands"
	eventsHeading   = "Events"
	viewsHeading    = "Views"
	actorsHeading   = "Actors"
)

type document struct {
	model    term
	groups   []termGroup
	contexts []contextSection
}

type contextSection struct {
	term       term
	aggregates []aggregateSection
	groups     []termGroup
}

type aggregateSection struct {
	term term
}

type termGroup struct {
	heading string
	terms   []term
}

type term struct {
	name       string
	definition string
}

func newDocument(model *ast.Model) document {
	descriptions := actorDescriptions(model.Actors)

	doc := document{
		model: term{name: model.Name, definition: model.Description},
		groups: nonEmptyGroups(termGroup{
			heading: actorsHeading,
			terms:   unreferencedActorTerms(model),
		}),
	}

	for _, ctx := range model.Contexts {
		slices := allSlicesIn(ctx)
		doc.contexts = append(doc.contexts, contextSection{
			term:       term{name: ctx.Name, definition: ctx.Description},
			aggregates: aggregateSections(ctx.Aggregates),
			groups: nonEmptyGroups(
				termGroup{heading: commandsHeading, terms: commandTerms(slices)},
				termGroup{heading: eventsHeading, terms: eventTerms(slices)},
				termGroup{heading: viewsHeading, terms: viewTerms(slices)},
				termGroup{heading: actorsHeading, terms: actorTerms(triggerActorNames(slices), descriptions)},
			),
		})
	}

	return doc
}

func allSlicesIn(ctx *ast.Context) []*ast.Slice {
	var slices []*ast.Slice
	for _, agg := range ctx.Aggregates {
		slices = append(slices, agg.Slices...)
	}
	return append(slices, ctx.Slices...)
}

func aggregateSections(aggregates []*ast.Aggregate) []aggregateSection {
	sections := make([]aggregateSection, 0, len(aggregates))
	for _, agg := range aggregates {
		sections = append(sections, aggregateSection{term: term{name: agg.Name, definition: agg.Description}})
	}
	return sections
}

func commandTerms(slices []*ast.Slice) []term {
	var terms []term
	for _, slice := range slices {
		for _, cmd := range slice.Commands {
			terms = append(terms, term{name: cmd.Name, definition: cmd.Description})
		}
	}
	return terms
}

func eventTerms(slices []*ast.Slice) []term {
	var terms []term
	for _, slice := range slices {
		for _, evt := range slice.Events {
			terms = append(terms, term{name: evt.Name, definition: evt.Description})
		}
		for _, translation := range slice.Translations {
			if translation.Event != nil {
				terms = append(terms, term{name: translation.Event.Name, definition: translation.Event.Description})
			}
		}
	}
	return terms
}

func viewTerms(slices []*ast.Slice) []term {
	var terms []term
	for _, slice := range slices {
		for _, view := range slice.Views {
			terms = append(terms, term{name: view.Name, definition: view.Description})
		}
	}
	return terms
}

func triggerActorNames(slices []*ast.Slice) []string {
	var names []string
	seen := make(map[string]bool)
	for _, slice := range slices {
		if slice.Trigger == nil || slice.Trigger.Actor == "" || seen[slice.Trigger.Actor] {
			continue
		}
		seen[slice.Trigger.Actor] = true
		names = append(names, slice.Trigger.Actor)
	}
	return names
}

func actorTerms(names []string, descriptions map[string]string) []term {
	var terms []term
	for _, name := range names {
		terms = append(terms, term{name: name, definition: descriptions[name]})
	}
	return terms
}

func actorDescriptions(actors []*ast.Actor) map[string]string {
	descriptions := make(map[string]string, len(actors))
	for _, actor := range actors {
		descriptions[actor.Name] = actor.Description
	}
	return descriptions
}

func unreferencedActorTerms(model *ast.Model) []term {
	referenced := make(map[string]bool)
	for _, ctx := range model.Contexts {
		for _, name := range triggerActorNames(allSlicesIn(ctx)) {
			referenced[name] = true
		}
	}

	var terms []term
	for _, actor := range model.Actors {
		if referenced[actor.Name] {
			continue
		}
		terms = append(terms, term{name: actor.Name, definition: actor.Description})
	}
	return terms
}

func nonEmptyGroups(groups ...termGroup) []termGroup {
	var kept []termGroup
	for _, group := range groups {
		if len(group.terms) > 0 {
			kept = append(kept, group)
		}
	}
	return kept
}

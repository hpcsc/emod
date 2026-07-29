package glossary

import (
	"fmt"
	"strings"

	"github.com/hpcsc/emod/internal/ast"
)

const (
	modelLevel = iota + 1
	contextLevel
	aggregateLevel
)

const (
	commandsHeading = "Commands"
	eventsHeading   = "Events"
	viewsHeading    = "Views"
	actorsHeading   = "Actors"
)

type termGroup struct {
	heading string
	terms   []term
}

// RenderMarkdown writes the model's vocabulary as markdown, nesting each
// context under the model and, under each context, its aggregates followed by
// the commands, events, views and actors its slices declare. An actor no
// trigger references is listed under the model instead. A construct without a
// description keeps its heading and has no definition beneath it.
func RenderMarkdown(model *ast.Model) ([]byte, error) {
	if model == nil {
		return []byte{}, nil
	}

	doc := newDocument(model)

	var blocks []string
	blocks = appendTerm(blocks, modelLevel, doc.term)
	blocks = appendGroups(blocks, contextLevel, termGroup{heading: actorsHeading, terms: doc.Actors})
	for _, ctx := range doc.Contexts {
		blocks = appendTerm(blocks, contextLevel, ctx.term)
		for _, aggregate := range ctx.Aggregates {
			blocks = appendTerm(blocks, aggregateLevel, aggregate)
		}
		blocks = appendGroups(blocks, aggregateLevel,
			termGroup{heading: commandsHeading, terms: ctx.Commands},
			termGroup{heading: eventsHeading, terms: ctx.Events},
			termGroup{heading: viewsHeading, terms: ctx.Views},
			termGroup{heading: actorsHeading, terms: ctx.Actors},
		)
	}

	return []byte(strings.Join(blocks, "\n\n") + "\n"), nil
}

func appendGroups(blocks []string, level int, groups ...termGroup) []string {
	for _, group := range groups {
		if len(group.terms) == 0 {
			continue
		}
		blocks = append(blocks, headingLine(level, group.heading))
		for _, grouped := range group.terms {
			blocks = appendTerm(blocks, level+1, grouped)
		}
	}
	return blocks
}

func appendTerm(blocks []string, level int, t term) []string {
	blocks = append(blocks, headingLine(level, t.Name))
	if t.Description != "" {
		blocks = append(blocks, t.Description)
	}
	return blocks
}

func headingLine(level int, text string) string {
	return fmt.Sprintf("%s %s", strings.Repeat("#", level), text)
}

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
	blocks = appendTerm(blocks, modelLevel, doc.model)
	blocks = appendGroups(blocks, contextLevel, doc.groups)
	for _, ctx := range doc.contexts {
		blocks = appendTerm(blocks, contextLevel, ctx.term)
		for _, aggregate := range ctx.aggregates {
			blocks = appendTerm(blocks, aggregateLevel, aggregate.term)
		}
		blocks = appendGroups(blocks, aggregateLevel, ctx.groups)
	}

	return []byte(strings.Join(blocks, "\n\n") + "\n"), nil
}

func appendGroups(blocks []string, level int, groups []termGroup) []string {
	for _, group := range groups {
		blocks = append(blocks, headingLine(level, group.heading))
		for _, grouped := range group.terms {
			blocks = appendTerm(blocks, level+1, grouped)
		}
	}
	return blocks
}

func appendTerm(blocks []string, level int, t term) []string {
	blocks = append(blocks, headingLine(level, t.name))
	if t.definition != "" {
		blocks = append(blocks, t.definition)
	}
	return blocks
}

func headingLine(level int, text string) string {
	return fmt.Sprintf("%s %s", strings.Repeat("#", level), text)
}

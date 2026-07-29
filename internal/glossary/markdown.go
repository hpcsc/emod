package glossary

import (
	"fmt"
	"strings"

	"github.com/hpcsc/emod/internal/ast"
)

const (
	modelHeading = iota + 1
	contextHeading
	aggregateHeading
)

// RenderMarkdown writes the model's vocabulary as markdown, nesting each
// context under the model and each aggregate under its context. A construct
// without a description keeps its heading and has no definition beneath it.
func RenderMarkdown(model *ast.Model) ([]byte, error) {
	if model == nil {
		return []byte{}, nil
	}

	doc := newDocument(model)

	var blocks []string
	blocks = appendTerm(blocks, modelHeading, doc.model)
	for _, ctx := range doc.contexts {
		blocks = appendTerm(blocks, contextHeading, ctx.term)
		for _, aggregate := range ctx.aggregates {
			blocks = appendTerm(blocks, aggregateHeading, aggregate)
		}
	}

	return []byte(strings.Join(blocks, "\n\n") + "\n"), nil
}

func appendTerm(blocks []string, heading int, t term) []string {
	blocks = append(blocks, fmt.Sprintf("%s %s", strings.Repeat("#", heading), t.name))
	if t.definition != "" {
		blocks = append(blocks, t.definition)
	}
	return blocks
}

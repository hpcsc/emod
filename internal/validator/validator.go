package validator

import (
	"fmt"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/diagnostic"
)

func Validate(model *ast.Model) []*diagnostic.Entry {
	if model == nil {
		return nil
	}

	contextNames := make(map[string]bool, len(model.Contexts))
	for _, ctx := range model.Contexts {
		contextNames[ctx.Name] = true
	}

	var diags []*diagnostic.Entry

	for _, ctx := range model.Contexts {
		for _, agg := range ctx.Aggregates {
			for _, slice := range agg.Slices {
				for _, auto := range slice.Automations {
					if auto.TargetContext == "" {
						continue
					}
					if !contextNames[auto.TargetContext] {
						diags = append(diags, &diagnostic.Entry{
							Filename: auto.TargetContextPos.Filename,
							Line:     auto.TargetContextPos.Line,
							Column:   auto.TargetContextPos.Column,
							Message:  fmt.Sprintf("target context %q does not exist", auto.TargetContext),
						})
					}
				}
			}
		}
	}

	return diags
}

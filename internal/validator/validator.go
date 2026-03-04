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
	commandNames := make(map[string]bool)
	for _, ctx := range model.Contexts {
		contextNames[ctx.Name] = true
		for _, agg := range ctx.Aggregates {
			for _, slice := range agg.Slices {
				for _, cmd := range slice.Commands {
					commandNames[cmd.Name] = true
				}
			}
		}
	}

	var diags []*diagnostic.Entry

	for _, ctx := range model.Contexts {
		for _, agg := range ctx.Aggregates {
			for _, slice := range agg.Slices {
				for _, auto := range slice.Automations {
					if auto.TargetContext != "" && !contextNames[auto.TargetContext] {
						diags = append(diags, &diagnostic.Entry{
							Filename: auto.TargetContextPos.Filename,
							Line:     auto.TargetContextPos.Line,
							Column:   auto.TargetContextPos.Column,
							Message:  fmt.Sprintf("target context %q does not exist", auto.TargetContext),
						})
					}
					if auto.Command != "" && !commandNames[auto.Command] {
						diags = append(diags, &diagnostic.Entry{
							Filename: auto.CommandPos.Filename,
							Line:     auto.CommandPos.Line,
							Column:   auto.CommandPos.Column,
							Message:  fmt.Sprintf("command %q does not exist", auto.Command),
						})
					}
				}
				for _, tr := range slice.Translations {
					if tr.Command != "" && !commandNames[tr.Command] {
						diags = append(diags, &diagnostic.Entry{
							Filename: tr.CommandPos.Filename,
							Line:     tr.CommandPos.Line,
							Column:   tr.CommandPos.Column,
							Message:  fmt.Sprintf("command %q does not exist", tr.Command),
						})
					}
				}
			}
		}
	}

	return diags
}

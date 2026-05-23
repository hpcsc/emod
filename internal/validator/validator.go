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
	eventNames := make(map[string]bool)
	commandPositions := make(map[string]ast.Position)
	eventPositions := make(map[string]ast.Position)
	flowCmdNames := make(map[string]bool)
	producedEventNames := make(map[string]bool)
	for _, ctx := range model.Contexts {
		contextNames[ctx.Name] = true
		for _, agg := range ctx.Aggregates {
			for _, slice := range agg.Slices {
				for _, cmd := range slice.Commands {
					commandNames[cmd.Name] = true
					commandPositions[cmd.Name] = cmd.NamePos
				}
				for _, evt := range slice.Events {
					eventNames[evt.Name] = true
					eventPositions[evt.Name] = evt.NamePos
					if evt.Source != "" {
						producedEventNames[evt.Name] = true
					}
				}
				for _, tr := range slice.Translations {
					if tr.Event != nil {
						eventNames[tr.Event.Name] = true
						producedEventNames[tr.Event.Name] = true
					}
				}
				for _, f := range slice.Flows {
					if f.CommandName != "" {
						flowCmdNames[f.CommandName] = true
					}
					if f.EventName != "" {
						producedEventNames[f.EventName] = true
					}
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
					if auto.TriggerEvent != "" && !eventNames[auto.TriggerEvent] {
						diags = append(diags, &diagnostic.Entry{
							Filename: auto.TriggerEventPos.Filename,
							Line:     auto.TriggerEventPos.Line,
							Column:   auto.TriggerEventPos.Column,
							Message:  fmt.Sprintf("event %q does not exist", auto.TriggerEvent),
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
				for _, v := range slice.Views {
					for _, sub := range v.Subscribes {
						if sub != "" && !eventNames[sub] {
							diags = append(diags, &diagnostic.Entry{
								Filename: v.NamePos.Filename,
								Line:     v.NamePos.Line,
								Column:   v.NamePos.Column,
								Message:  fmt.Sprintf("event %q does not exist", sub),
							})
						}
					}
				}
				for _, f := range slice.Flows {
					if f.EventName != "" && !eventNames[f.EventName] {
						diags = append(diags, &diagnostic.Entry{
							Filename: f.EventPos.Filename,
							Line:     f.EventPos.Line,
							Column:   f.EventPos.Column,
							Message:  fmt.Sprintf("event %q does not exist", f.EventName),
						})
					}
				}
			}
		}
	}

	// orphan-command: commands defined but not referenced by any flow
	for name, pos := range commandPositions {
		if !flowCmdNames[name] {
			diags = append(diags, &diagnostic.Entry{
				Filename: pos.Filename,
				Line:     pos.Line,
				Column:   pos.Column,
				Message:  fmt.Sprintf("command %q is orphaned (no flow references it)", name),
				Severity: diagnostic.Error,
				RuleName: "orphan-command",
			})
		}
	}

	// orphan-event: events defined but not produced by any flow, external source, or translation
	for name, pos := range eventPositions {
		if !producedEventNames[name] {
			diags = append(diags, &diagnostic.Entry{
				Filename: pos.Filename,
				Line:     pos.Line,
				Column:   pos.Column,
				Message:  fmt.Sprintf("event %q is orphaned (no flow, external source, or translation produces it)", name),
				Severity: diagnostic.Error,
				RuleName: "orphan-event",
			})
		}
	}

	return diags
}

package linter

import (
	"fmt"
	"strings"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/diagnostic"
)

var stateObsessionSuffixes = []string{"Updated", "Changed", "Modified"}

func Lint(model *ast.Model) []*diagnostic.Entry {
	if model == nil {
		return nil
	}

	var diags []*diagnostic.Entry

	for _, ctx := range model.Contexts {
		for _, agg := range ctx.Aggregates {
			for _, slice := range agg.Slices {
				for _, evt := range slice.Events {
					diags = append(diags, checkEvent(evt, agg.Name)...)
				}
				for _, tr := range slice.Translations {
					if tr.Event != nil {
						diags = append(diags, checkEvent(tr.Event, agg.Name)...)
					}
				}
			}
		}
	}

	return diags
}

func checkEvent(evt *ast.Event, aggregateName string) []*diagnostic.Entry {
	if d := checkPropertySourcing(evt, aggregateName); d != nil {
		return []*diagnostic.Entry{d}
	}
	if d := checkStateObsession(evt); d != nil {
		return []*diagnostic.Entry{d}
	}
	if d := checkCommandInDisguise(evt); d != nil {
		return []*diagnostic.Entry{d}
	}
	return nil
}

func checkStateObsession(evt *ast.Event) *diagnostic.Entry {
	for _, suffix := range stateObsessionSuffixes {
		if strings.HasSuffix(evt.Name, suffix) {
			return &diagnostic.Entry{
				Filename: evt.NamePos.Filename,
				Line:     evt.NamePos.Line,
				Column:   evt.NamePos.Column,
				Severity: diagnostic.Warning,
				RuleName: "state-obsession",
				Message:  fmt.Sprintf("event %q uses a generic state-change suffix; prefer a name that describes a specific business fact", evt.Name),
			}
		}
	}
	return nil
}

// Property-sourcing fires when the event name is <AggregateName><Field>Changed.
// It is checked before state-obsession so the more specific rule wins.
func checkPropertySourcing(evt *ast.Event, aggregateName string) *diagnostic.Entry {
	if !strings.HasPrefix(evt.Name, aggregateName) || !strings.HasSuffix(evt.Name, "Changed") {
		return nil
	}

	field := strings.TrimSuffix(strings.TrimPrefix(evt.Name, aggregateName), "Changed")
	if field == "" {
		return nil
	}

	return &diagnostic.Entry{
		Filename: evt.NamePos.Filename,
		Line:     evt.NamePos.Line,
		Column:   evt.NamePos.Column,
		Severity: diagnostic.Warning,
		RuleName: "property-sourcing",
		Message:  fmt.Sprintf("event %q tracks a single property change on %s.%s; prefer an event that captures the business reason for the change", evt.Name, aggregateName, field),
	}
}

func checkCommandInDisguise(evt *ast.Event) *diagnostic.Entry {
	if strings.HasSuffix(evt.Name, "Initiated") {
		return &diagnostic.Entry{
			Filename: evt.NamePos.Filename,
			Line:     evt.NamePos.Line,
			Column:   evt.NamePos.Column,
			Severity: diagnostic.Warning,
			RuleName: "command-in-disguise",
			Message:  fmt.Sprintf("event %q sounds like a command; events should describe what happened, not what was requested", evt.Name),
		}
	}
	return nil
}

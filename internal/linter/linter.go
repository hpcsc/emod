package linter

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/diagnostic"
)

var stateObsessionSuffixes = []string{"Updated", "Changed", "Modified"}

func warning(pos ast.Position, rule, msg string) *diagnostic.Entry {
	return &diagnostic.Entry{
		Filename: pos.Filename,
		Line:     pos.Line,
		Column:   pos.Column,
		Severity: diagnostic.Warning,
		RuleName: rule,
		Message:  msg,
	}
}

func error(pos ast.Position, rule, msg string) *diagnostic.Entry {
	return &diagnostic.Entry{
		Filename: pos.Filename,
		Line:     pos.Line,
		Column:   pos.Column,
		Severity: diagnostic.Error,
		RuleName: rule,
		Message:  msg,
	}
}

func Lint(model *ast.Model) []*diagnostic.Entry {
	if model == nil {
		return nil
	}

	var diags []*diagnostic.Entry

	// Build flow count map for left-chair detection
	flowCount := make(map[string]int)
	for _, ctx := range model.Contexts {
		for _, agg := range ctx.Aggregates {
			for _, slice := range agg.Slices {
				for _, flow := range slice.Flows {
					flowCount[flow.CommandName]++
				}
			}
		}
	}

	for _, ctx := range model.Contexts {
		for _, agg := range ctx.Aggregates {
			for _, slice := range agg.Slices {
				for _, evt := range slice.Events {
					diags = append(diags, checkEvent(evt, agg.Name)...)
					if d := checkClickbaitEvent(evt); d != nil {
						diags = append(diags, d)
					}
				}
				for _, tr := range slice.Translations {
					if tr.Event != nil {
						diags = append(diags, checkEvent(tr.Event, agg.Name)...)
						if d := checkClickbaitEvent(tr.Event); d != nil {
							diags = append(diags, d)
						}
					}
				}
				for _, cmd := range slice.Commands {
					if d := checkCommandPastTense(cmd); d != nil {
						diags = append(diags, d)
					}
					if d := checkLeftChair(cmd, flowCount); d != nil {
						diags = append(diags, d)
					}
				}
				for _, view := range slice.Views {
					if d := checkViewNaming(view); d != nil {
						diags = append(diags, d)
					}
					if d := checkGodView(view); d != nil {
						diags = append(diags, d)
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
			return warning(evt.NamePos, "state-obsession", fmt.Sprintf("event %q uses a generic state-change suffix; prefer a name that describes a specific business fact", evt.Name))
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

	return warning(evt.NamePos, "property-sourcing", fmt.Sprintf("event %q tracks a single property change on %s.%s; prefer an event that captures the business reason for the change", evt.Name, aggregateName, field))
}

func checkCommandInDisguise(evt *ast.Event) *diagnostic.Entry {
	if strings.HasSuffix(evt.Name, "Initiated") {
		return warning(evt.NamePos, "command-in-disguise", fmt.Sprintf("event %q sounds like a command; events should describe what happened, not what was requested", evt.Name))
	}
	return nil
}

var imperativeVerbsEndingInEd = map[string]bool{
	"proceed": true,
	"exceed":  true,
	"feed":    true,
	"embed":   true,
	"speed":   true,
	"seed":    true,
	"shred":   true,
	"succeed": true,
	"need":    true,
	"bleed":   true,
	"breed":   true,
	"heed":    true,
	"weed":    true,
}

func lastPascalCaseWord(name string) string {
	lastUpper := -1
	for i, r := range name {
		if unicode.IsUpper(r) {
			lastUpper = i
		}
	}
	if lastUpper < 0 {
		return name
	}
	return name[lastUpper:]
}

func checkCommandPastTense(cmd *ast.Command) *diagnostic.Entry {
	if !strings.HasSuffix(cmd.Name, "ed") {
		return nil
	}

	lastWord := lastPascalCaseWord(cmd.Name)
	if imperativeVerbsEndingInEd[strings.ToLower(lastWord)] {
		return nil
	}

	return warning(cmd.NamePos, "command-past-tense", fmt.Sprintf("command %q appears to be past tense; commands should use imperative form (e.g. PlaceOrder, not OrderPlaced)", cmd.Name))
}

func checkViewNaming(view *ast.View) *diagnostic.Entry {
	if !strings.HasSuffix(view.Name, "View") {
		return warning(view.NamePos, "view-naming", fmt.Sprintf("view %q should end with \"View\" (e.g. %sView)", view.Name, view.Name))
	}
	return nil
}

func checkLeftChair(cmd *ast.Command, flowCount map[string]int) *diagnostic.Entry {
	if flowCount[cmd.Name] >= 3 {
		return error(cmd.NamePos, "left-chair", fmt.Sprintf("command %q is referenced by %d flows; consider splitting into specialized commands to reduce coupling", cmd.Name, flowCount[cmd.Name]))
	}
	return nil
}

func checkGodView(view *ast.View) *diagnostic.Entry {
	if len(view.Subscribes) >= 5 {
		return error(view.NamePos, "god-view", fmt.Sprintf("view %q subscribes to %d events; consider splitting into smaller focused views", view.Name, len(view.Subscribes)))
	}
	return nil
}

// isIDField returns true if the field name suggests it is an ID reference.
// In PascalCase/naming conventions, ID fields end with "Id" or "ID".
func isIDField(name string) bool {
	return strings.HasSuffix(name, "Id") || strings.HasSuffix(name, "ID")
}

func checkClickbaitEvent(evt *ast.Event) *diagnostic.Entry {
	if len(evt.Fields) == 1 && isIDField(evt.Fields[0].Name) {
		return error(evt.NamePos, "clickbait-event", fmt.Sprintf("event %q has a single ID field %q; consider adding domain-relevant fields or inlining the identifier", evt.Name, evt.Fields[0].Name))
	}
	return nil
}

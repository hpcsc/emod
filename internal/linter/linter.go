package linter

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/diagnostic"
)

var stateObsessionSuffixes = []string{"Updated", "Changed", "Modified"}

func info(pos ast.Position, rule, msg string) *diagnostic.Entry {
	return &diagnostic.Entry{
		Filename: pos.Filename,
		Line:     pos.Line,
		Column:   pos.Column,
		Severity: diagnostic.Info,
		RuleName: rule,
		Message:  msg,
	}
}

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

	// Build flow count map for left-chair detection across all slices
	flowCount := make(map[string]int)
	for _, ctx := range model.Contexts {
		for _, agg := range ctx.Aggregates {
			for _, slice := range agg.Slices {
				for _, flow := range slice.Flows {
					flowCount[flow.CommandName]++
				}
			}
		}
		for _, slice := range ctx.Slices {
			for _, flow := range slice.Flows {
				flowCount[flow.CommandName]++
			}
		}
	}

	for _, ctx := range model.Contexts {
		// Mode-aware checks
		if isAggregateMode(ctx.Mode) {
			diags = append(diags, checkDCBInAggregateMode(ctx)...)
		} else if isDCBMode(ctx.Mode) {
			diags = append(diags, checkAggregateInDCBMode(ctx)...)
		}
		// Mixed mode: no extra mode warnings

		// DCB and mixed modes require tags on all events
		if isDCBMode(ctx.Mode) || isMixedMode(ctx.Mode) {
			diags = append(diags, checkUntaggedEvents(ctx)...)
		}

		// DCB and mixed modes: check for broad queries on decides_on
		if isDCBMode(ctx.Mode) || isMixedMode(ctx.Mode) {
			diags = append(diags, checkQueryTooBroad(ctx)...)
		}

		// Existing checks on aggregate-level slices
		for _, agg := range ctx.Aggregates {
			for _, slice := range agg.Slices {
				diags = append(diags, checkSlice(slice, agg.Name, flowCount)...)
			}
		}

		// Existing checks on context-level slices
		for _, slice := range ctx.Slices {
			diags = append(diags, checkSlice(slice, "", flowCount)...)
		}
	}

	return diags
}

// checkSlice applies all existing lint checks to a single slice.
// aggregateName is used for property-sourcing detection; pass "" for context-level slices.
func checkSlice(slice *ast.Slice, aggregateName string, flowCount map[string]int) []*diagnostic.Entry {
	var diags []*diagnostic.Entry
	for _, evt := range slice.Events {
		diags = append(diags, checkEvent(evt, aggregateName)...)
		if d := checkClickbaitEvent(evt); d != nil {
			diags = append(diags, d)
		}
	}
	for _, tr := range slice.Translations {
		if tr.Event != nil {
			diags = append(diags, checkEvent(tr.Event, aggregateName)...)
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
	return diags
}

// Mode helpers

func isAggregateMode(mode string) bool {
	return mode == "" || mode == "aggregate"
}

func isDCBMode(mode string) bool {
	return mode == "dcb"
}

func isMixedMode(mode string) bool {
	return mode == "mixed"
}

// checkDCBInAggregateMode warns about DCB constructs found in an aggregate-mode context.
// DCB-only constructs are: tags on events, decides_on on commands, and slices directly under context.
func checkDCBInAggregateMode(ctx *ast.Context) []*diagnostic.Entry {
	var diags []*diagnostic.Entry

	// Check all slices (both aggregate-level and context-level) for DCB constructs
	allSlices := ctx.Slices
	for _, agg := range ctx.Aggregates {
		allSlices = append(allSlices, agg.Slices...)
	}

	for _, slice := range allSlices {
		for _, evt := range slice.Events {
			if len(evt.Tags) > 0 {
				diags = append(diags, warning(evt.NamePos, "dcb-in-aggregate-mode",
					fmt.Sprintf("event %q uses DCB-style tags in an aggregate-mode context %q", evt.Name, ctx.Name)))
			}
		}
		for _, tr := range slice.Translations {
			if tr.Event != nil && len(tr.Event.Tags) > 0 {
				diags = append(diags, warning(tr.Event.NamePos, "dcb-in-aggregate-mode",
					fmt.Sprintf("event %q uses DCB-style tags in an aggregate-mode context %q", tr.Event.Name, ctx.Name)))
			}
		}
		for _, cmd := range slice.Commands {
			if cmd.DecidesOn != nil {
				diags = append(diags, warning(cmd.NamePos, "dcb-in-aggregate-mode",
					fmt.Sprintf("command %q uses DCB-style decides_on in an aggregate-mode context %q", cmd.Name, ctx.Name)))
			}
		}
	}

	// Warn for context-level slices themselves being a DCB construct
	for _, slice := range ctx.Slices {
		diags = append(diags, warning(slice.NamePos, "dcb-in-aggregate-mode",
			fmt.Sprintf("slice %q is a DCB-style construct in an aggregate-mode context %q", slice.Name, ctx.Name)))
	}

	return diags
}

// checkAggregateInDCBMode warns about aggregate blocks found in a DCB-mode context.
func checkAggregateInDCBMode(ctx *ast.Context) []*diagnostic.Entry {
	var diags []*diagnostic.Entry
	for _, agg := range ctx.Aggregates {
		diags = append(diags, warning(agg.NamePos, "aggregate-in-dcb-mode",
			fmt.Sprintf("aggregate block %q is an aggregate-style construct in a DCB-mode context %q", agg.Name, ctx.Name)))
	}
	return diags
}

// checkUntaggedEvents fires when an event in a DCB or mixed-mode context lacks tags.
// Tags are required in these modes because the routing infrastructure relies on them.
func checkUntaggedEvents(ctx *ast.Context) []*diagnostic.Entry {
	var diags []*diagnostic.Entry

	allSlices := ctx.Slices
	for _, agg := range ctx.Aggregates {
		allSlices = append(allSlices, agg.Slices...)
	}

	for _, slice := range allSlices {
		for _, evt := range slice.Events {
			if len(evt.Tags) == 0 {
				diags = append(diags, error(evt.NamePos, "dcb/untagged-event",
					fmt.Sprintf("event %q is missing tags in %s-mode context %q", evt.Name, ctx.Mode, ctx.Name)))
			}
		}
		for _, tr := range slice.Translations {
			if tr.Event != nil && len(tr.Event.Tags) == 0 {
				diags = append(diags, error(tr.Event.NamePos, "dcb/untagged-event",
					fmt.Sprintf("event %q is missing tags in %s-mode context %q", tr.Event.Name, ctx.Mode, ctx.Name)))
			}
		}
	}

	return diags
}

// checkQueryTooBroad warns when a command's decides_on references more than 5 event types
// or has no predicate (missing where clause). This rule only fires in DCB and mixed modes.
func checkQueryTooBroad(ctx *ast.Context) []*diagnostic.Entry {
	var diags []*diagnostic.Entry

	allSlices := ctx.Slices
	for _, agg := range ctx.Aggregates {
		allSlices = append(allSlices, agg.Slices...)
	}

	for _, slice := range allSlices {
		for _, cmd := range slice.Commands {
			if cmd.DecidesOn == nil {
				continue
			}
			if len(cmd.DecidesOn.Events) > 5 || cmd.DecidesOn.Predicate == nil {
				var reasons []string
				if len(cmd.DecidesOn.Events) > 5 {
					reasons = append(reasons, fmt.Sprintf("references %d event types", len(cmd.DecidesOn.Events)))
				}
				if cmd.DecidesOn.Predicate == nil {
					reasons = append(reasons, "has no where clause")
				}
				msg := fmt.Sprintf("command %q has a broad query: %s", cmd.Name, strings.Join(reasons, " and "))
				diags = append(diags, warning(cmd.NamePos, "dcb/query-too-broad", msg))
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
// When aggregateName is empty (context-level slice), property-sourcing does not apply.
func checkPropertySourcing(evt *ast.Event, aggregateName string) *diagnostic.Entry {
	if aggregateName == "" {
		return nil
	}
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

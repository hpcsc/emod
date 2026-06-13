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

	// Stores per-event field names and tag keys for DCB cross-reference validation
	eventFields := make(map[string][]string)
	eventTagKeys := make(map[string][]string)

	for _, ctx := range model.Contexts {
		contextNames[ctx.Name] = true
		for _, slice := range ctx.Slices {
			collectNames(slice, commandNames, eventNames, commandPositions, eventPositions,
				flowCmdNames, producedEventNames, eventFields, eventTagKeys)
		}
		for _, agg := range ctx.Aggregates {
			for _, slice := range agg.Slices {
				collectNames(slice, commandNames, eventNames, commandPositions, eventPositions,
					flowCmdNames, producedEventNames, eventFields, eventTagKeys)
			}
		}
	}

	var diags []*diagnostic.Entry

	// Cross-reference validation for all slices (both direct ctx.Slices and aggregate slices)
	for _, ctx := range model.Contexts {
		for _, slice := range ctx.Slices {
			validateReferences(slice, contextNames, commandNames, eventNames, &diags)
		}
		for _, agg := range ctx.Aggregates {
			for _, slice := range agg.Slices {
				validateReferences(slice, contextNames, commandNames, eventNames, &diags)
			}
		}
	}

	// DCB tag field reference validation: tag key:fieldRef must match declared event fields
	for _, ctx := range model.Contexts {
		for _, slice := range ctx.Slices {
			validateTagFieldRefs(slice, eventFields, &diags)
		}
		for _, agg := range ctx.Aggregates {
			for _, slice := range agg.Slices {
				validateTagFieldRefs(slice, eventFields, &diags)
			}
		}
	}

	// DCB decides_on validation: event names must exist, predicate references must be valid
	for _, ctx := range model.Contexts {
		for _, slice := range ctx.Slices {
			validateDecidesOn(slice, eventNames, eventFields, eventTagKeys, &diags)
		}
		for _, agg := range ctx.Aggregates {
			for _, slice := range agg.Slices {
				validateDecidesOn(slice, eventNames, eventFields, eventTagKeys, &diags)
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

// collectNames gathers all names and per-event metadata from a single slice.
func collectNames(
	slice *ast.Slice,
	commandNames, eventNames map[string]bool,
	commandPositions, eventPositions map[string]ast.Position,
	flowCmdNames, producedEventNames map[string]bool,
	eventFields, eventTagKeys map[string][]string,
) {
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
		for _, f := range evt.Fields {
			eventFields[evt.Name] = append(eventFields[evt.Name], f.Name)
		}
		for _, tag := range evt.Tags {
			eventTagKeys[evt.Name] = append(eventTagKeys[evt.Name], tag.Key)
		}
	}
	for _, tr := range slice.Translations {
		if tr.Event != nil {
			eventNames[tr.Event.Name] = true
			producedEventNames[tr.Event.Name] = true
			for _, f := range tr.Event.Fields {
				eventFields[tr.Event.Name] = append(eventFields[tr.Event.Name], f.Name)
			}
			for _, tag := range tr.Event.Tags {
				eventTagKeys[tr.Event.Name] = append(eventTagKeys[tr.Event.Name], tag.Key)
			}
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
	for _, auto := range slice.Automations {
		if auto.Command != "" {
			flowCmdNames[auto.Command] = true
		}
	}
	for _, tr := range slice.Translations {
		if tr.Command != "" {
			flowCmdNames[tr.Command] = true
		}
	}
}

// validateReferences checks automations, translations, views, and flows for
// references to names that do not exist in the model.
func validateReferences(
	slice *ast.Slice,
	contextNames, commandNames, eventNames map[string]bool,
	diags *[]*diagnostic.Entry,
) {
	for _, auto := range slice.Automations {
		if auto.TargetContext != "" && !contextNames[auto.TargetContext] {
			*diags = append(*diags, &diagnostic.Entry{
				Filename: auto.TargetContextPos.Filename,
				Line:     auto.TargetContextPos.Line,
				Column:   auto.TargetContextPos.Column,
				Message:  fmt.Sprintf("target context %q does not exist", auto.TargetContext),
			})
		}
		if auto.Command != "" && !commandNames[auto.Command] {
			*diags = append(*diags, &diagnostic.Entry{
				Filename: auto.CommandPos.Filename,
				Line:     auto.CommandPos.Line,
				Column:   auto.CommandPos.Column,
				Message:  fmt.Sprintf("command %q does not exist", auto.Command),
			})
		}
		if auto.TriggerEvent != "" && !eventNames[auto.TriggerEvent] {
			*diags = append(*diags, &diagnostic.Entry{
				Filename: auto.TriggerEventPos.Filename,
				Line:     auto.TriggerEventPos.Line,
				Column:   auto.TriggerEventPos.Column,
				Message:  fmt.Sprintf("event %q does not exist", auto.TriggerEvent),
			})
		}
	}
	for _, tr := range slice.Translations {
		if tr.Command != "" && !commandNames[tr.Command] {
			*diags = append(*diags, &diagnostic.Entry{
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
				*diags = append(*diags, &diagnostic.Entry{
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
			*diags = append(*diags, &diagnostic.Entry{
				Filename: f.EventPos.Filename,
				Line:     f.EventPos.Line,
				Column:   f.EventPos.Column,
				Message:  fmt.Sprintf("event %q does not exist", f.EventName),
			})
		}
	}
}

// validateTagFieldRefs checks that every tag entry's field reference on an event
// matches a declared field name on that event.
func validateTagFieldRefs(slice *ast.Slice, eventFields map[string][]string, diags *[]*diagnostic.Entry) {
	for _, evt := range slice.Events {
		validateEventTagFieldRefs(evt, eventFields, diags)
	}
	for _, tr := range slice.Translations {
		if tr.Event != nil {
			validateEventTagFieldRefs(tr.Event, eventFields, diags)
		}
	}
}

func validateEventTagFieldRefs(evt *ast.Event, eventFields map[string][]string, diags *[]*diagnostic.Entry) {
	for _, tag := range evt.Tags {
		if tag.FieldRef != "" {
			fields := eventFields[evt.Name]
			if !contains(fields, tag.FieldRef) {
				*diags = append(*diags, &diagnostic.Entry{
					Filename: tag.FieldRefPos.Filename,
					Line:     tag.FieldRefPos.Line,
					Column:   tag.FieldRefPos.Column,
					Message:  fmt.Sprintf("tag field reference %q does not match any field on event %q", tag.FieldRef, evt.Name),
				})
			}
		}
	}
}

// validateDecidesOn validates decides_on clauses on commands:
//   - Event names listed in events must exist in the model
//   - Predicate tag keys must be declared on at least one listed event
//   - Predicate field references must be declared on at least one listed event
func validateDecidesOn(
	slice *ast.Slice,
	eventNames map[string]bool,
	eventFields, eventTagKeys map[string][]string,
	diags *[]*diagnostic.Entry,
) {
	for _, cmd := range slice.Commands {
		if cmd.DecidesOn == nil {
			continue
		}
		clause := cmd.DecidesOn

		// Check that each event name in decides_on.events exists
		for i, evtName := range clause.Events {
			if !eventNames[evtName] {
				pos := clause.EventsPos[i]
				*diags = append(*diags, &diagnostic.Entry{
					Filename: pos.Filename,
					Line:     pos.Line,
					Column:   pos.Column,
					Message:  fmt.Sprintf("event %q in decides_on does not exist", evtName),
				})
			}
		}

		// Validate predicate if present
		if clause.Predicate != nil {
			validatePredicateTagRefs(clause.Predicate, clause.Events, eventFields, eventTagKeys, diags)
		}
	}
}

// validatePredicateTagRefs recursively walks a predicate expression and validates
// that tag key references and field references are declared on at least one of
// the given events.
func validatePredicateTagRefs(
	expr ast.PredicateExpr,
	events []string,
	eventFields, eventTagKeys map[string][]string,
	diags *[]*diagnostic.Entry,
) {
	switch e := expr.(type) {
	case *ast.TagPredicate:
		// Check that the tag key (e.g., "aggregate_id") is declared on at least one
		// of the events listed in decides_on.events
		if !anyEventHasTagKey(events, eventTagKeys, e.Field) {
			*diags = append(*diags, &diagnostic.Entry{
				Filename: e.FieldPos.Filename,
				Line:     e.FieldPos.Line,
				Column:   e.FieldPos.Column,
				Message:  fmt.Sprintf("tag key %q is not declared on any event in decides_on", e.Field),
			})
		}
		// Check that the field reference (e.g., "orderId") is declared as a field on
		// at least one of the events listed in decides_on.events
		if !anyEventHasField(events, eventFields, e.Value) {
			*diags = append(*diags, &diagnostic.Entry{
				Filename: e.ValuePos.Filename,
				Line:     e.ValuePos.Line,
				Column:   e.ValuePos.Column,
				Message:  fmt.Sprintf("field reference %q is not declared on any event in decides_on", e.Value),
			})
		}

	case *ast.LogicalExpr:
		validatePredicateTagRefs(e.Left, events, eventFields, eventTagKeys, diags)
		validatePredicateTagRefs(e.Right, events, eventFields, eventTagKeys, diags)

	case *ast.NotExpr:
		validatePredicateTagRefs(e.Expr, events, eventFields, eventTagKeys, diags)

	case *ast.FieldRef:
		// Bare FieldRef not expected in DCB predicate context; skip validation.
	}
}

func contains(list []string, item string) bool {
	for _, s := range list {
		if s == item {
			return true
		}
	}
	return false
}

func anyEventHasTagKey(events []string, eventTagKeys map[string][]string, key string) bool {
	for _, evt := range events {
		if contains(eventTagKeys[evt], key) {
			return true
		}
	}
	return false
}

func anyEventHasField(events []string, eventFields map[string][]string, field string) bool {
	for _, evt := range events {
		if contains(eventFields[evt], field) {
			return true
		}
	}
	return false
}

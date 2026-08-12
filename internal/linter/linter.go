package linter

import (
	"cmp"
	"fmt"
	"math/big"
	"slices"
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

func errorEntry(pos ast.Position, rule, msg string) *diagnostic.Entry {
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
	for _, slice := range model.AllSlices() {
		for _, flow := range slice.Flows {
			flowCount[flow.CommandName]++
		}
	}

	hasSpec := false
	exercisedCommands := make(map[string]bool)
	commandsWithRejection := make(map[string]bool)
	for _, slice := range model.AllSlices() {
		for _, spec := range slice.Specs {
			hasSpec = true
			if spec.When != nil {
				exercisedCommands[spec.When.Name] = true
			}
			if _, ok := spec.Then.(*ast.ThenRejected); ok && spec.When != nil {
				commandsWithRejection[spec.When.Name] = true
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

		// DCB and mixed modes: check for single tag key across all decides_on predicates
		if isDCBMode(ctx.Mode) || isMixedMode(ctx.Mode) {
			diags = append(diags, checkSingleTagEverywhere(ctx)...)
		}

		// DCB and mixed modes: check for orphan tag keys (declared on events but never referenced in predicates)
		if isDCBMode(ctx.Mode) || isMixedMode(ctx.Mode) {
			diags = append(diags, checkOrphanTagKeys(ctx)...)
		}

		for _, ref := range ctx.SliceRefs() {
			aggregateName := ""
			if ref.Aggregate != nil {
				aggregateName = ref.Aggregate.Name
			}
			diags = append(diags, checkSlice(ref.Slice, aggregateName, flowCount, hasSpec, exercisedCommands, commandsWithRejection)...)
		}
	}

	diags = append(diags, checkInvariantNeverExercised(model)...)
	diags = append(diags, checkGivenOutsideBoundary(model)...)

	slices.SortFunc(diags, byPosition)

	return diags
}

// byPosition orders diagnostics as a reader meets them. Line alone is not a
// total order: one payload written the way emod fmt writes it puts every value
// on a single line, so several diagnostics share it and only the column tells
// them apart.
func byPosition(a, b *diagnostic.Entry) int {
	if line := cmp.Compare(a.Line, b.Line); line != 0 {
		return line
	}

	return cmp.Compare(a.Column, b.Column)
}

// checkSlice applies all existing lint checks to a single slice.
// aggregateName is used for property-sourcing detection; pass "" for context-level slices.
func checkSlice(slice *ast.Slice, aggregateName string, flowCount map[string]int, hasSpec bool, exercisedCommands map[string]bool, commandsWithRejection map[string]bool) []*diagnostic.Entry {
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
		if d := checkCommandWithoutSpec(cmd, hasSpec, exercisedCommands); d != nil {
			diags = append(diags, d)
		}
		if d := checkNoRejectionPath(cmd, exercisedCommands, commandsWithRejection); d != nil {
			diags = append(diags, d)
		}
	}
	diags = append(diags, checkRejectionsWithoutSpec(slice)...)
	for _, view := range slice.Views {
		if d := checkViewNaming(view); d != nil {
			diags = append(diags, d)
		}
		if d := checkGodView(view); d != nil {
			diags = append(diags, d)
		}
	}
	for _, auto := range slice.Automations {
		if d := checkMissingTodoList(auto); d != nil {
			diags = append(diags, d)
		}
	}
	return diags
}

// checkRejectionsWithoutSpec reports each rejection edge whose own slice states
// no spec exercising it. The search is slice-local on purpose: a rejection edge
// is declared inside one slice's flow block, so that slice is its scope and a
// spec written in an unrelated slice must not silence it.
func checkRejectionsWithoutSpec(slice *ast.Slice) []*diagnostic.Entry {
	var diags []*diagnostic.Entry
	for _, rejection := range slice.Rejections {
		if rejection == nil || exercisesRejection(slice, rejection) {
			continue
		}
		diags = append(diags, info(rejection.InvariantPos, "flow/rejection-without-spec",
			fmt.Sprintf("command %q can be rejected by invariant %q, but no spec on this slice exercises that rejection",
				rejection.CommandName, rejection.InvariantName)))
	}

	return diags
}

// exercisesRejection matches both halves of the edge. Matching the invariant
// alone would let one command's rejection spec silence another command's edge.
// The outcome is asked whether it is a rejection through an ok-guarded
// assertion, so a variant added to ThenClause later is neither counted as one
// nor treated as an error.
func exercisesRejection(slice *ast.Slice, rejection *ast.Rejection) bool {
	for _, spec := range slice.Specs {
		if spec == nil || spec.When == nil || spec.When.Name != rejection.CommandName {
			continue
		}
		if rejected, isRejection := spec.Then.(*ast.ThenRejected); isRejection &&
			rejected.InvariantName == rejection.InvariantName {
			return true
		}
	}

	return false
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
	allSlices := ctx.AllSlices()

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

	allSlices := ctx.AllSlices()

	for _, slice := range allSlices {
		for _, evt := range slice.Events {
			if len(evt.Tags) == 0 {
				diags = append(diags, errorEntry(evt.NamePos, "dcb/untagged-event",
					fmt.Sprintf("event %q is missing tags in %s-mode context %q", evt.Name, ctx.Mode, ctx.Name)))
			}
		}
		for _, tr := range slice.Translations {
			if tr.Event != nil && len(tr.Event.Tags) == 0 {
				diags = append(diags, errorEntry(tr.Event.NamePos, "dcb/untagged-event",
					fmt.Sprintf("event %q is missing tags in %s-mode context %q", tr.Event.Name, ctx.Mode, ctx.Name)))
			}
		}
	}

	return diags
}

// checkSingleTagEverywhere fires an info diagnostic when all commands in a DCB or
// mixed-mode context use only one distinct tag key across their decides_on predicates.
// Using many tag keys is an indicator of healthy domain diversity; a single key suggests
// the tag dimension is not being used to express different routing concerns.
func checkSingleTagEverywhere(ctx *ast.Context) []*diagnostic.Entry {
	var diags []*diagnostic.Entry

	allSlices := ctx.AllSlices()

	tagKeys := make(map[string]bool)
	for _, slice := range allSlices {
		for _, cmd := range slice.Commands {
			if cmd.DecidesOn == nil || cmd.DecidesOn.Predicate == nil {
				continue
			}
			keys := collectPredicateTagKeys(cmd.DecidesOn.Predicate)
			for _, k := range keys {
				tagKeys[k] = true
			}
		}
	}

	if len(tagKeys) == 1 {
		var key string
		for k := range tagKeys {
			key = k
		}
		diags = append(diags, info(ctx.NamePos, "dcb/single-tag-everywhere",
			fmt.Sprintf("context %q uses only tag key %q across all decides_on predicates in %s mode", ctx.Name, key, ctx.Mode)))
	}

	return diags
}

// checkOrphanTagKeys warns when a tag key declared on events is never referenced
// in any command's decides_on predicate in a DCB or mixed-mode context.
// Each orphan key produces its own diagnostic pointing at the first event that declares it.
func checkOrphanTagKeys(ctx *ast.Context) []*diagnostic.Entry {
	var diags []*diagnostic.Entry

	allSlices := ctx.AllSlices()

	// Collect all tag keys declared on events, tracking the first event for each key
	eventTagKeys := make(map[string]bool)
	firstEventForKey := make(map[string]ast.Position)

	for _, slice := range allSlices {
		for _, evt := range slice.Events {
			for _, tag := range evt.Tags {
				if !eventTagKeys[tag.Key] {
					eventTagKeys[tag.Key] = true
					firstEventForKey[tag.Key] = evt.NamePos
				}
			}
		}
		for _, tr := range slice.Translations {
			if tr.Event != nil {
				for _, tag := range tr.Event.Tags {
					if !eventTagKeys[tag.Key] {
						eventTagKeys[tag.Key] = true
						firstEventForKey[tag.Key] = tr.Event.NamePos
					}
				}
			}
		}
	}

	// If no events have tags, nothing to check
	if len(eventTagKeys) == 0 {
		return nil
	}

	// Collect all tag keys referenced in commands' decides_on predicates
	predicateTagKeys := make(map[string]bool)
	for _, slice := range allSlices {
		for _, cmd := range slice.Commands {
			if cmd.DecidesOn == nil || cmd.DecidesOn.Predicate == nil {
				continue
			}
			keys := collectPredicateTagKeys(cmd.DecidesOn.Predicate)
			for _, k := range keys {
				predicateTagKeys[k] = true
			}
		}
	}

	// Find orphan keys — declared on events but never referenced in predicates.
	// Sorted by declaration position: ranging over the map directly would emit
	// diagnostics in Go's randomised map order, so the same file would report
	// the same problems in a different order on every run.
	orphaned := make([]string, 0, len(eventTagKeys))
	for key := range eventTagKeys {
		if !predicateTagKeys[key] {
			orphaned = append(orphaned, key)
		}
	}
	slices.SortFunc(orphaned, func(a, b string) int {
		if c := firstEventForKey[a].Compare(firstEventForKey[b]); c != 0 {
			return c
		}
		return cmp.Compare(a, b)
	})

	for _, key := range orphaned {
		diags = append(diags, warning(firstEventForKey[key], "dcb/orphan-tag-key",
			fmt.Sprintf("tag key %q declared on events is never referenced in any command's decides_on predicate in %s-mode context %q", key, ctx.Mode, ctx.Name)))
	}

	return diags
}

// collectPredicateTagKeys recursively walks a predicate expression tree and
// returns all tag key field names referenced in TagPredicate nodes.
func collectPredicateTagKeys(pred ast.PredicateExpr) []string {
	if pred == nil {
		return nil
	}
	switch p := pred.(type) {
	case *ast.TagPredicate:
		return []string{p.Field}
	case *ast.LogicalExpr:
		var keys []string
		keys = append(keys, collectPredicateTagKeys(p.Left)...)
		keys = append(keys, collectPredicateTagKeys(p.Right)...)
		return keys
	case *ast.NotExpr:
		return collectPredicateTagKeys(p.Expr)
	}
	return nil
}

// checkQueryTooBroad warns when a command's decides_on references more than 5 event types
// or has no predicate (missing where clause). This rule only fires in DCB and mixed modes.
func checkQueryTooBroad(ctx *ast.Context) []*diagnostic.Entry {
	var diags []*diagnostic.Entry

	allSlices := ctx.AllSlices()

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
		return errorEntry(cmd.NamePos, "left-chair", fmt.Sprintf("command %q is referenced by %d flows; consider splitting into specialized commands to reduce coupling", cmd.Name, flowCount[cmd.Name]))
	}
	return nil
}

func checkCommandWithoutSpec(cmd *ast.Command, hasSpec bool, exercisedCommands map[string]bool) *diagnostic.Entry {
	if !hasSpec {
		return nil
	}
	if exercisedCommands[cmd.Name] {
		return nil
	}
	return info(cmd.NamePos, "spec/command-without-spec", fmt.Sprintf("command %q is not exercised by any spec", cmd.Name))
}

func checkNoRejectionPath(cmd *ast.Command, exercisedCommands map[string]bool, commandsWithRejection map[string]bool) *diagnostic.Entry {
	if !exercisedCommands[cmd.Name] {
		return nil
	}
	if commandsWithRejection[cmd.Name] {
		return nil
	}
	return info(cmd.NamePos, "spec/no-rejection-path", fmt.Sprintf("command %q is exercised by specs but none states a rejection", cmd.Name))
}

func checkMissingTodoList(auto *ast.Automation) *diagnostic.Entry {
	if auto.Reads != "" {
		return nil
	}

	consequence := "nothing in the model shows what work is outstanding"
	if auto.Schedule != "" {
		consequence = "the model does not state what the processor acts on"
	}

	return warning(auto.NamePos, "automation/missing-todo-list",
		fmt.Sprintf("automation %q reads no view, so %s; project a view of pending work and read it", auto.Name, consequence))
}

func checkGodView(view *ast.View) *diagnostic.Entry {
	if len(view.Subscribes) >= 5 {
		return errorEntry(view.NamePos, "god-view", fmt.Sprintf("view %q subscribes to %d events; consider splitting into smaller focused views", view.Name, len(view.Subscribes)))
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
		return errorEntry(evt.NamePos, "clickbait-event", fmt.Sprintf("event %q has a single ID field %q; consider adding domain-relevant fields or inlining the identifier", evt.Name, evt.Fields[0].Name))
	}
	return nil
}

func checkInvariantNeverExercised(model *ast.Model) []*diagnostic.Entry {
	var diags []*diagnostic.Entry

	for _, ctx := range model.Contexts {
		diags = append(diags, scopeUnexercisedInvariants(ctx.Invariants, ctx.Slices, "context", ctx.Name)...)
		for _, agg := range ctx.Aggregates {
			diags = append(diags, scopeUnexercisedInvariants(agg.Invariants, agg.Slices, "aggregate", agg.Name)...)
		}
	}

	slices.SortFunc(diags, func(a, b *diagnostic.Entry) int {
		return cmp.Compare(a.Line, b.Line)
	})

	return diags
}

func rejectionReferences(slices []*ast.Slice, invariants []*ast.Invariant) map[string]bool {
	declared := make(map[string]bool, len(invariants))
	for _, inv := range invariants {
		declared[inv.Name] = true
	}

	referenced := make(map[string]bool)
	for _, sl := range slices {
		for _, spec := range sl.Specs {
			rejection, ok := spec.Then.(*ast.ThenRejected)
			if !ok {
				continue
			}
			if declared[rejection.InvariantName] {
				referenced[rejection.InvariantName] = true
			}
		}
		for _, rejection := range sl.Rejections {
			if declared[rejection.InvariantName] {
				referenced[rejection.InvariantName] = true
			}
		}
	}
	return referenced
}

func scopeUnexercisedInvariants(invariants []*ast.Invariant, slices []*ast.Slice, scopeKind, scopeName string) []*diagnostic.Entry {
	if len(invariants) == 0 {
		return nil
	}

	referenced := rejectionReferences(slices, invariants)

	var diags []*diagnostic.Entry
	for _, inv := range invariants {
		if referenced[inv.Name] {
			continue
		}
		diags = append(diags, warning(inv.NamePos, "spec/invariant-never-exercised",
			fmt.Sprintf("invariant %q in %s %q is not referenced by any rejection", inv.Name, scopeKind, scopeName)))
	}

	return diags
}

type eventDeclaration struct {
	ownerName string
	kind      string
}

func eventHomeIndex(model *ast.Model) map[string][]eventDeclaration {
	index := make(map[string][]eventDeclaration)
	for _, ctx := range model.Contexts {
		for _, ref := range ctx.SliceRefs() {
			homeName := ctx.Name
			homeKind := "context"
			if ref.Aggregate != nil {
				homeName = ref.Aggregate.Name
				homeKind = "aggregate"
			}
			for _, evt := range ref.Slice.Events {
				index[evt.Name] = append(index[evt.Name], eventDeclaration{homeName, homeKind})
			}
			for _, tr := range ref.Slice.Translations {
				if tr.Event != nil {
					index[tr.Event.Name] = append(index[tr.Event.Name], eventDeclaration{homeName, homeKind})
				}
			}
		}
	}
	return index
}

// contextEventIndex names the events one context declares. A DCB query reads the
// history its own context holds, and event names are only unique within one, so
// resolving a given event's tags model-wide would read the tag mapping off a
// same-named event another context declares. An event this context does not
// declare is absent rather than borrowed from elsewhere.
func contextEventIndex(ctx *ast.Context) map[string]*ast.Event {
	index := make(map[string]*ast.Event)
	for _, slice := range ctx.AllSlices() {
		for _, evt := range slice.Events {
			if _, seen := index[evt.Name]; !seen {
				index[evt.Name] = evt
			}
		}
		for _, tr := range slice.Translations {
			if tr.Event == nil {
				continue
			}
			if _, seen := index[tr.Event.Name]; !seen {
				index[tr.Event.Name] = tr.Event
			}
		}
	}

	return index
}

func eventHomeLookup(index map[string][]eventDeclaration, eventName string) (eventDeclaration, bool) {
	homes, ok := index[eventName]
	if !ok || len(homes) == 0 {
		return eventDeclaration{}, false
	}
	return homes[0], true
}

func checkGivenOutsideBoundary(model *ast.Model) []*diagnostic.Entry {
	eventHome := eventHomeIndex(model)

	commandIndex := make(map[string]*ast.Command)
	for _, sl := range model.AllSlices() {
		for _, cmd := range sl.Commands {
			commandIndex[cmd.Name] = cmd
		}
	}

	var diags []*diagnostic.Entry
	for _, ctx := range model.Contexts {
		declaredHere := contextEventIndex(ctx)
		for _, ref := range ctx.SliceRefs() {
			if ref.Aggregate != nil {
				for _, spec := range ref.Slice.Specs {
					for _, given := range spec.Given {
						homes, ok := eventHome[given.Name]
						if !ok || len(homes) == 0 {
							continue
						}
						inBoundary := false
						for _, h := range homes {
							if h.ownerName == ref.Aggregate.Name {
								inBoundary = true
								break
							}
						}
						if inBoundary {
							continue
						}
						diags = append(diags, warning(given.NamePos, "spec/given-outside-boundary",
							fmt.Sprintf("given event %q names an event declared by %s %q instead of aggregate %q",
								given.Name, homes[0].kind, homes[0].ownerName, ref.Aggregate.Name)))
					}
				}
				continue
			}

			for _, spec := range ref.Slice.Specs {
				if spec.When == nil {
					continue
				}
				cmd, ok := commandIndex[spec.When.Name]
				if !ok {
					continue
				}
				if cmd.DecidesOn == nil {
					continue
				}
				decidesSet := make(map[string]bool, len(cmd.DecidesOn.Events))
				for _, e := range cmd.DecidesOn.Events {
					decidesSet[e] = true
				}
				for _, given := range spec.Given {
					if _, ok := eventHomeLookup(eventHome, given.Name); !ok {
						continue
					}
					if !decidesSet[given.Name] {
						diags = append(diags, warning(given.NamePos, "spec/given-outside-boundary",
							fmt.Sprintf("given event %q names an event command %q's decides_on does not list",
								given.Name, spec.When.Name)))
						continue
					}
					diags = append(diags, excludedPayloadValues(given, declaredHere[given.Name], spec.When, cmd)...)
				}
			}
		}
	}

	slices.SortFunc(diags, byPosition)

	return diags
}

// excludedPayloadValues reports each tagged field whose value the given payload
// states differently from the when payload, so the command's query would pass
// over the event the given names. Each predicate the query requires states a
// separate routing requirement and so is separately fixable.
//
// The two sides are looked up under different names. A tag key is an
// indirection — "tags { desk: seatId }" says the desk tag is carried by seatId —
// so the given event is read through the field its own tags block maps the key
// to, while the when payload is read under the field the predicate names on the
// command. Reading both under the predicate's name compares seatId against
// nothing whenever a model spells the two apart.
func excludedPayloadValues(given *ast.SpecElement, declared *ast.Event, when *ast.SpecElement, cmd *ast.Command) []*diagnostic.Entry {
	var diags []*diagnostic.Entry
	for _, predicate := range requiredTagPredicates(cmd.DecidesOn.Predicate) {
		givenField, tagged := taggedField(declared, predicate.Field)
		if !tagged {
			continue
		}
		givenValue := statedOnce(given.Payload, givenField)
		whenValue := statedOnce(when.Payload, predicate.Value)
		if givenValue == nil || whenValue == nil {
			continue
		}
		if compareLiterals(givenValue, whenValue) != literalsDiffer {
			continue
		}

		diags = append(diags, warning(givenValue.ValuePos, "spec/given-outside-boundary",
			fmt.Sprintf("given event %q states %s %s while command %q's when payload states %s %s, so tag %q excludes it from the query",
				given.Name, givenField, literalSource(givenValue),
				cmd.Name, predicate.Value, literalSource(whenValue), predicate.Field)))
	}

	return diags
}

// taggedField names the field an event's tags block binds a tag key to. An event
// carrying no such key is not routed on it at all, which leaves the comparison
// no basis rather than a disagreement.
func taggedField(declared *ast.Event, key string) (string, bool) {
	if declared == nil {
		return "", false
	}
	for _, tag := range declared.Tags {
		if tag.Key == key {
			return tag.FieldRef, true
		}
	}

	return "", false
}

// requiredTagPredicates selects the tag predicates a query requires to hold:
// those reachable from the root through "and" alone. Under "or" a single
// disagreement does not put the event outside the query, and under "not" it
// argues the opposite, so neither is crossed and every warning the arm emits
// names a predicate the command genuinely demands. Parentheses leave no node
// behind, so conjunctive position is decidable from the tree alone. Note the
// deliberate difference from collectPredicateTagKeys, which descends both
// operands whatever the operator and descends a not: this walk does neither.
func requiredTagPredicates(pred ast.PredicateExpr) []*ast.TagPredicate {
	switch p := pred.(type) {
	case *ast.TagPredicate:
		return []*ast.TagPredicate{p}
	case *ast.LogicalExpr:
		if p.Operator != "and" {
			return nil
		}

		return append(requiredTagPredicates(p.Left), requiredTagPredicates(p.Right)...)
	case *ast.NotExpr:
		return nil
	}

	return nil
}

// statedOnce returns the single field a payload states under name. A payload
// that omits it leaves nothing to compare — a spec is an example, not an
// instance — and one that states it twice states two values, so nothing here
// can say which one the author meant.
func statedOnce(payload []*ast.PayloadField, name string) *ast.PayloadField {
	var found *ast.PayloadField
	for _, field := range payload {
		if field.Name != name {
			continue
		}
		if found != nil {
			return nil
		}
		found = field
	}

	return found
}

// literalAgreement is how two payload literals stand to one another. Literals of
// different kinds are incomparable rather than unequal: nothing requires a
// command's field and an event's field of one name to declare the same type, so
// such a pair is a field typed two ways — and a check that cannot tell a
// different value from a different spelling has no business naming a boundary.
type literalAgreement int

const (
	literalsIncomparable literalAgreement = iota
	literalsEqual
	literalsDiffer
)

func compareLiterals(given, when *ast.PayloadField) literalAgreement {
	switch {
	case given.Kind == ast.StringLiteral && when.Kind == ast.StringLiteral,
		given.Kind == ast.BooleanLiteral && when.Kind == ast.BooleanLiteral:
		if given.Value == when.Value {
			return literalsEqual
		}

		return literalsDiffer
	case isNumberLiteral(given.Kind) && isNumberLiteral(when.Kind):
		return compareNumberLiterals(given.Value, when.Value)
	}

	return literalsIncomparable
}

func isNumberLiteral(kind ast.LiteralKind) bool {
	return kind == ast.IntegerLiteral || kind == ast.DecimalLiteral
}

// compareNumberLiterals reads both literals as exact rationals rather than as
// source text, so 12.50 and 12.5 are one value and 007 and 7 are one value: the
// AST keeps a number's spelling verbatim so emod fmt can write it back, and a
// formatting difference is not a boundary violation. float64 would equate two
// distinct integers past its precision.
func compareNumberLiterals(given, when string) literalAgreement {
	left, leftOK := new(big.Rat).SetString(given)
	right, rightOK := new(big.Rat).SetString(when)
	if !leftOK || !rightOK {
		return literalsIncomparable
	}
	if left.Cmp(right) == 0 {
		return literalsEqual
	}

	return literalsDiffer
}

// literalSource spells a literal the way its author wrote it, so a message
// echoes the source: a string keeps its quotes and 12.50 keeps its trailing
// zero, though the comparison above reads it as a number.
func literalSource(field *ast.PayloadField) string {
	if field.Kind == ast.StringLiteral {
		return `"` + field.Value + `"`
	}

	return field.Value
}

package formatter

import (
	"fmt"
	"strings"

	"github.com/hpcsc/emod/internal/ast"
)

func Format(model *ast.Model) string {
	var b strings.Builder
	f := &writer{b: &b}
	f.writeModel(model)
	return b.String()
}

type writer struct {
	b *strings.Builder
}

func (w *writer) line(level int, format string, args ...any) {
	fmt.Fprintf(w.b, "%s%s\n", indent(level), fmt.Sprintf(format, args...))
}

func (w *writer) blankLine() {
	w.b.WriteString("\n")
}

func (w *writer) blankLineBetweenBlocks() func() {
	first := true
	return func() {
		if !first {
			w.blankLine()
		}
		first = false
	}
}

func indent(level int) string {
	return strings.Repeat("  ", level)
}

// quoted renders text as an emod string literal, which runs verbatim from one
// quote to the next: the language has no escape sequences. %q would escape a
// backslash or tab that the lexer then reads back as the escape sequence itself,
// so the text would grow on every format run.
func quoted(text string) string {
	return `"` + text + `"`
}

func bracketed(names []string) string {
	return "[" + strings.Join(names, ", ") + "]"
}

func (w *writer) writeComments(comments []*ast.Comment, level int) {
	for _, c := range comments {
		w.line(level, "%s", c.Text)
	}
}

func (w *writer) writeDescription(description string, level int) {
	if description == "" {
		return
	}
	w.line(level, "description %s", quoted(description))
}

func (w *writer) writeInvariants(invariants []*ast.Invariant, level int) {
	for _, invariant := range invariants {
		w.writeComments(invariant.Comments, level)
		w.line(level, "invariant %s %s", invariant.Name, quoted(invariant.Statement))
	}
}

func (w *writer) writeDeclaration(keyword, name, description string) {
	if description == "" {
		w.line(0, "%s %s", keyword, quoted(name))
		return
	}
	w.line(0, "%s %s {", keyword, quoted(name))
	w.writeDescription(description, 1)
	w.line(0, "}")
}

func (w *writer) writeModel(model *ast.Model) {
	w.line(0, "emod %d", pinnedVersion(model))
	w.writeComments(model.Comments, 0)
	w.writeDeclaration("model", model.Name, model.Description)

	for _, actor := range model.Actors {
		w.blankLine()
		w.writeComments(actor.Comments, 0)
		w.writeDeclaration("actor", actor.Name, actor.Description)
	}

	for _, ctx := range model.Contexts {
		w.blankLine()
		w.writeContext(ctx, 0)
	}
}

func pinnedVersion(model *ast.Model) int {
	if model.VersionDeclared {
		return model.Version
	}
	return ast.SupportedVersion
}

func (w *writer) writeContext(ctx *ast.Context, level int) {
	w.writeComments(ctx.Comments, level)
	if ctx.Mode != "" {
		w.line(level, "context %s mode %s {", quoted(ctx.Name), ctx.Mode)
	} else {
		w.line(level, "context %s {", quoted(ctx.Name))
	}
	w.writeDescription(ctx.Description, level+1)
	w.writeInvariants(ctx.Invariants, level+1)

	separate := w.blankLineBetweenBlocks()
	for _, agg := range ctx.Aggregates {
		separate()
		w.writeAggregate(agg, level+1)
	}
	for _, slice := range ctx.Slices {
		separate()
		w.writeSlice(slice, level+1)
	}
	w.line(level, "}")
}

func (w *writer) writeAggregate(agg *ast.Aggregate, level int) {
	w.writeComments(agg.Comments, level)
	w.line(level, "aggregate %s {", quoted(agg.Name))
	w.writeDescription(agg.Description, level+1)
	w.writeInvariants(agg.Invariants, level+1)

	separate := w.blankLineBetweenBlocks()
	for _, slice := range agg.Slices {
		separate()
		w.writeSlice(slice, level+1)
	}
	w.line(level, "}")
}

func (w *writer) writeSlice(slice *ast.Slice, level int) {
	w.writeComments(slice.Comments, level)
	w.line(level, "slice %s {", quoted(slice.Name))

	inner := level + 1
	w.writeDescription(slice.Description, inner)

	separate := w.blankLineBetweenBlocks()

	if slice.Trigger != nil {
		separate()
		w.writeTrigger(slice.Trigger, inner)
	}

	for _, cmd := range slice.Commands {
		separate()
		w.writeCommand(cmd, inner)
	}

	for _, evt := range slice.Events {
		separate()
		w.writeEvent(evt, inner)
	}

	for _, view := range slice.Views {
		separate()
		w.writeView(view, inner)
	}

	for _, auto := range slice.Automations {
		separate()
		w.writeAutomation(auto, inner)
	}

	for _, trans := range slice.Translations {
		separate()
		w.writeTranslation(trans, inner)
	}

	if len(slice.Flows) > 0 {
		separate()
		w.writeFlows(slice.Flows, inner)
	}

	for _, spec := range slice.Specs {
		separate()
		w.writeSpec(spec, inner)
	}

	w.line(level, "}")
}

func (w *writer) writeTrigger(trigger *ast.Trigger, level int) {
	w.writeComments(trigger.Comments, level)
	w.line(level, "trigger %s %s {", trigger.Kind, quoted(trigger.Name))
	w.writeDescription(trigger.Description, level+1)
	if trigger.Actor != "" {
		w.line(level+1, "actor %s", trigger.Actor)
	}
	if trigger.Reads != "" {
		w.line(level+1, "reads %s", trigger.Reads)
	}
	w.line(level, "}")
}

func (w *writer) writeCommand(cmd *ast.Command, level int) {
	w.writeComments(cmd.Comments, level)
	w.line(level, "command %s {", cmd.Name)
	w.writeDescription(cmd.Description, level+1)
	if cmd.DecidesOn != nil {
		w.writeDecidesOn(cmd.DecidesOn, level+1)
	}
	if len(cmd.Fields) > 0 {
		w.writeFields(cmd.Fields, level+1)
	}
	w.line(level, "}")
}

func (w *writer) writeDecidesOn(d *ast.DecidesOnClause, level int) {
	w.writeComments(d.Comments, level)
	w.line(level, "decides_on {")
	if len(d.Events) > 0 {
		w.line(level+1, "events %s", bracketed(d.Events))
	}
	if d.Predicate != nil {
		w.line(level+1, "where %s", formatPredicate(d.Predicate))
	}
	w.line(level, "}")
}

func formatPredicate(expr ast.PredicateExpr) string {
	switch e := expr.(type) {
	case *ast.TagPredicate:
		return fmt.Sprintf("tag(%s %s %s)", e.Field, e.Operator, e.Value)
	case *ast.NotExpr:
		return fmt.Sprintf("not %s", formatPredicateParen(e.Expr, "not"))
	case *ast.LogicalExpr:
		left := formatPredicateParen(e.Left, e.Operator)
		right := formatPredicateParen(e.Right, e.Operator)
		return fmt.Sprintf("%s %s %s", left, e.Operator, right)
	default:
		return ""
	}
}

// formatPredicateParen wraps expr in parens when needed for correct precedence.
// parentOp is the operator of the parent expression ("and", "or", "not", or "").
func formatPredicateParen(expr ast.PredicateExpr, parentOp string) string {
	s := formatPredicate(expr)
	if logical, ok := expr.(*ast.LogicalExpr); ok {
		if parentOp == "not" || (parentOp == "and" && logical.Operator == "or") {
			return "(" + s + ")"
		}
	}
	return s
}

func (w *writer) writeEvent(evt *ast.Event, level int) {
	w.writeComments(evt.Comments, level)
	w.line(level, "event %s {", evt.Name)
	w.writeDescription(evt.Description, level+1)
	if len(evt.Tags) > 0 {
		w.writeTags(evt.Tags, level+1)
	}
	if evt.Source == "external" && evt.ExternalName != "" {
		w.line(level+1, "source external %s", quoted(evt.ExternalName))
	}
	if len(evt.Fields) > 0 {
		w.writeFields(evt.Fields, level+1)
	}
	w.line(level, "}")
}

func (w *writer) writeFields(fields []*ast.Field, level int) {
	nameWidth, typeWidth := fieldColumnWidths(fields)

	w.line(level, "fields {")
	for _, f := range fields {
		if f.Modifier != "" {
			w.line(level+1, "%-*s %-*s %s", nameWidth, f.Name, typeWidth, f.Type, f.Modifier)
		} else {
			w.line(level+1, "%-*s %s", nameWidth, f.Name, f.Type)
		}
	}
	w.line(level, "}")
}

func (w *writer) writeTags(tags []ast.TagEntry, level int) {
	maxKey := 0
	for _, t := range tags {
		if len(t.Key) > maxKey {
			maxKey = len(t.Key)
		}
	}
	w.line(level, "tags {")
	for _, t := range tags {
		w.line(level+1, "%-*s: %s", maxKey, t.Key, t.FieldRef)
	}
	w.line(level, "}")
}

func fieldColumnWidths(fields []*ast.Field) (nameWidth, typeWidth int) {
	for _, f := range fields {
		if len(f.Name) > nameWidth {
			nameWidth = len(f.Name)
		}
		if len(f.Type) > typeWidth {
			typeWidth = len(f.Type)
		}
	}
	return nameWidth, typeWidth
}

func (w *writer) writeFlows(flows []*ast.Flow, level int) {
	w.line(level, "flow {")
	for _, flow := range flows {
		w.writeComments(flow.Comments, level+1)
		w.line(level+1, "command -> event: %s -> %s", flow.CommandName, flow.EventName)
	}
	w.line(level, "}")
}

func (w *writer) writeView(view *ast.View, level int) {
	w.writeComments(view.Comments, level)
	w.line(level, "view %s {", view.Name)
	w.writeDescription(view.Description, level+1)
	if len(view.Fields) > 0 {
		w.writeFields(view.Fields, level+1)
	}
	if len(view.Subscribes) > 0 {
		w.line(level+1, "subscribes %s", bracketed(view.Subscribes))
	}
	w.line(level, "}")
}

func (w *writer) writeAutomation(auto *ast.Automation, level int) {
	w.writeComments(auto.Comments, level)
	w.line(level, "automation %s {", auto.Name)
	w.writeDescription(auto.Description, level+1)
	if auto.TriggerEvent != "" {
		w.line(level+1, "trigger %s", auto.TriggerEvent)
	}
	if auto.Reads != "" {
		w.line(level+1, "reads %s", auto.Reads)
	}
	if auto.Command != "" {
		w.line(level+1, "command %s", auto.Command)
	}
	if auto.TargetContext != "" {
		w.line(level+1, "target context %s", auto.TargetContext)
	}
	w.line(level, "}")
}

func (w *writer) writeTranslation(trans *ast.Translation, level int) {
	w.writeComments(trans.Comments, level)
	w.line(level, "translation %s {", trans.Name)
	w.writeDescription(trans.Description, level+1)
	if trans.ExternalSystem != "" {
		w.line(level+1, "external_system %s", quoted(trans.ExternalSystem))
	}
	if trans.Reads != "" {
		w.line(level+1, "reads %s", trans.Reads)
	}
	if trans.Command != "" {
		w.line(level+1, "command %s", trans.Command)
	}
	if trans.Event != nil {
		w.writeEvent(trans.Event, level+1)
	}
	w.line(level, "}")
}

func (w *writer) writeSpec(spec *ast.Spec, level int) {
	w.writeComments(spec.Comments, level)
	w.line(level, "spec %s {", quoted(spec.Name))
	if len(spec.Given) > 0 {
		w.line(level+1, "given %s", bracketed(specElementNames(spec.Given)))
	}
	if spec.When != nil {
		w.line(level+1, "when %s", spec.When.Name)
	}
	if outcome := formatOutcome(spec.Then); outcome != "" {
		w.line(level+1, "then %s", outcome)
	}
	w.line(level, "}")
}

func formatOutcome(then ast.ThenClause) string {
	switch t := then.(type) {
	case *ast.ThenEvents:
		return bracketed(specElementNames(t.Events))
	case *ast.ThenRejected:
		if t.InvariantName == "" {
			return ""
		}
		return fmt.Sprintf("rejected %s", t.InvariantName)
	default:
		return ""
	}
}

func specElementNames(elements []*ast.SpecElement) []string {
	names := make([]string, 0, len(elements))
	for _, element := range elements {
		names = append(names, element.Name)
	}
	return names
}

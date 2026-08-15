package formatter

import (
	"fmt"
	"strings"
	"unicode/utf8"

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

func (w *writer) lineIfSet(level int, format, value string) {
	if value == "" {
		return
	}
	w.line(level, format, value)
}

func (w *writer) quotedLineIfSet(level int, format, value string) {
	if value == "" {
		return
	}
	w.line(level, format, quoted(value))
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

// maxLineWidth is the column budget a rendered line is measured against, its
// leading indent counted. The language fixes no line width, so this is the one
// place the formatter states one.
const maxLineWidth = 100

// quoted renders text as an emod string literal, which runs verbatim from one
// quote to the next: the language has no escape sequences. %q would escape a
// backslash or tab that the lexer then reads back as the escape sequence itself,
// so the text would grow on every format run.
func quoted(text string) string {
	return `"` + text + `"`
}

func quotedIfSet(value string) string {
	if value == "" {
		return ""
	}

	return quoted(value)
}

// withDelay suffixes an already-rendered activation value with the delay
// qualifying it. It answers "" for an activation the automation does not state,
// so the caller's lineIfSet guard still tests the activation and a delay alone
// invents no line. The delay is written as a suffix on whichever activation
// form is stated, including the every the parser rejects it beside: dropping it
// there would delete a clause the author wrote rather than let them read the
// diagnostic and fix it.
func withDelay(activation, after string) string {
	if activation == "" || after == "" {
		return activation
	}

	return activation + " after " + quoted(after)
}

// delayUnless withholds the delay from the every line when the automation also
// states an on event. Such an automation is one the parser rejects, but it
// reaches the formatter through the wasm export path, and writing the suffix on
// both lines would hand the author back a second delay they never wrote.
func delayUnless(after, onEvent string) string {
	if onEvent != "" {
		return ""
	}

	return after
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
	w.quotedLineIfSet(level, "description %s", description)
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

	if len(slice.Flows) > 0 || len(slice.Rejections) > 0 {
		separate()
		w.writeFlowBlock(slice.Flows, slice.Rejections, inner)
	}

	for _, spec := range slice.Specs {
		separate()
		w.writeSpec(spec, inner)
	}

	w.line(level, "}")
}

func (w *writer) writeTrigger(trigger *ast.Trigger, level int) {
	w.writeComments(trigger.Comments, level)
	w.line(level, "trigger %s {", quoted(trigger.Name))
	w.writeDescription(trigger.Description, level+1)
	w.lineIfSet(level+1, "actor %s", trigger.Actor)
	w.lineIfSet(level+1, "reads %s", trigger.Reads)
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
	w.quotedLineIfSet(level+1, "type %s", evt.WireType)
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
	keyWidth := columnWidth(tags, func(t ast.TagEntry) string { return t.Key })

	w.line(level, "tags {")
	for _, t := range tags {
		w.line(level+1, "%-*s: %s", keyWidth, t.Key, t.FieldRef)
	}
	w.line(level, "}")
}

func fieldColumnWidths(fields []*ast.Field) (nameWidth, typeWidth int) {
	return columnWidth(fields, func(f *ast.Field) string { return f.Name }),
		columnWidth(fields, func(f *ast.Field) string { return f.Type })
}

// columnWidth answers the width every row's cell must be padded to for the
// column after it to start in one place. It is computed over the rows handed
// to it and nothing else, so a block states its own alignment rather than
// inheriting a width from its siblings.
func columnWidth[T any](rows []T, cell func(T) string) int {
	widest := 0
	for _, row := range rows {
		if len(cell(row)) > widest {
			widest = len(cell(row))
		}
	}
	return widest
}

type flowEntry struct {
	comments []*ast.Comment
	prefix   string
	edge     string
}

func (w *writer) writeFlowBlock(flows []*ast.Flow, rejections []*ast.Rejection, level int) {
	entries := flowEntries(flows, rejections)
	prefixWidth := columnWidth(entries, func(e flowEntry) string { return e.prefix })

	w.line(level, "flow {")
	for _, entry := range entries {
		w.writeComments(entry.comments, level+1)
		w.line(level+1, "%-*s %s", prefixWidth, entry.prefix, entry.edge)
	}
	w.line(level, "}")
}

// flowEntries lists a flow block's entries, in canonical order: every event
// entry, then every rejection entry. The prefix column is padded across this
// list alone, so a block holding one entry kind is written with no padding.
func flowEntries(flows []*ast.Flow, rejections []*ast.Rejection) []flowEntry {
	entries := make([]flowEntry, 0, len(flows)+len(rejections))
	for _, flow := range flows {
		entries = append(entries, flowEntry{
			comments: flow.Comments,
			prefix:   "command -> event:",
			edge:     fmt.Sprintf("%s -> %s", flow.CommandName, flow.EventName),
		})
	}
	for _, rejection := range rejections {
		entries = append(entries, flowEntry{
			comments: rejection.Comments,
			prefix:   "command -> rejected:",
			edge:     fmt.Sprintf("%s -> %s", rejection.CommandName, rejection.InvariantName),
		})
	}
	return entries
}

func (w *writer) writeView(view *ast.View, level int) {
	w.writeComments(view.Comments, level)
	w.line(level, "view %s {", view.Name)
	w.writeDescription(view.Description, level+1)
	if len(view.Subscribes) > 0 {
		w.line(level+1, "subscribes %s", bracketed(view.Subscribes))
	}
	if len(view.Fields) > 0 {
		w.writeFields(view.Fields, level+1)
	}
	w.line(level, "}")
}

func (w *writer) writeAutomation(auto *ast.Automation, level int) {
	w.writeComments(auto.Comments, level)
	w.line(level, "automation %s {", auto.Name)
	w.writeDescription(auto.Description, level+1)
	w.lineIfSet(level+1, "on %s", withDelay(auto.OnEvent, auto.After))
	w.lineIfSet(level+1, "every %s", withDelay(quotedIfSet(auto.Schedule), delayUnless(auto.After, auto.OnEvent)))
	w.lineIfSet(level+1, "reads %s", auto.Reads)
	w.lineIfSet(level+1, "command %s", auto.Command)
	w.lineIfSet(level+1, "target context %s", auto.TargetContext)
	w.line(level, "}")
}

func (w *writer) writeTranslation(trans *ast.Translation, level int) {
	w.writeComments(trans.Comments, level)
	w.line(level, "translation %s {", trans.Name)
	w.writeDescription(trans.Description, level+1)
	w.quotedLineIfSet(level+1, "external_system %s", trans.ExternalSystem)
	w.lineIfSet(level+1, "reads %s", trans.Reads)
	w.lineIfSet(level+1, "command %s", trans.Command)
	if trans.Event != nil {
		w.writeEvent(trans.Event, level+1)
	}
	w.line(level, "}")
}

type specEntry struct {
	keyword   string
	value     string
	elements  []*ast.SpecElement
	bracketed bool
}

func (w *writer) writeSpec(spec *ast.Spec, level int) {
	w.writeComments(spec.Comments, level)
	w.line(level, "spec %s {", quoted(spec.Name))

	entries := specEntries(spec)
	keywordWidth := columnWidth(entries, func(e specEntry) string { return e.keyword })
	for _, entry := range entries {
		w.writeSpecEntry(entry, keywordWidth, level+1)
	}

	w.line(level, "}")
}

// writeSpecEntry keeps the entry on one line while that line fits the budget.
// Past it, the payload that made the line overrun is written one field per
// line, and a list holding one goes one element per line so the brace block is
// not buried inside a comma list. An entry stating no payload is left alone
// however long it runs: it has nothing this rule knows how to break.
func (w *writer) writeSpecEntry(entry specEntry, keywordWidth, level int) {
	head := fmt.Sprintf("%-*s ", keywordWidth, entry.keyword)

	if fitsOnOneLine(level, head+entry.value) || !statesPayload(entry.elements) {
		w.line(level, "%s", head+entry.value)
		return
	}

	if !entry.bracketed {
		w.writeSpecElement(entry.elements[0], head, "", level)
		return
	}

	w.line(level, "%s[", head)
	for i, element := range entry.elements {
		separator := ","
		if i == len(entry.elements)-1 {
			separator = ""
		}
		w.writeSpecElement(element, "", separator, level+1)
	}
	w.line(level, "]")
}

// writeSpecElement writes one reference and the payload qualifying it, wrapping
// that payload one field per line when the reference does not fit the line it
// sits on. The opening brace stays on the reference's own line, which is where
// the parser requires it.
func (w *writer) writeSpecElement(element *ast.SpecElement, head, separator string, level int) {
	oneLine := head + specElementText(element) + separator
	if len(element.Payload) == 0 || fitsOnOneLine(level, oneLine) {
		w.line(level, "%s", oneLine)
		return
	}

	labelWidth := columnWidth(element.Payload, func(f *ast.PayloadField) string { return payloadLabel(f) })

	w.line(level, "%s%s {", head, element.Name)
	for _, field := range element.Payload {
		w.line(level+1, "%-*s %s", labelWidth, payloadLabel(field), payloadLiteral(field))
	}
	w.line(level, "}%s", separator)
}

// fitsOnOneLine counts characters rather than bytes, because the budget is a
// column count: a payload value holding an accented letter or an em dash would
// otherwise be charged two or three columns for one and wrap early.
func fitsOnOneLine(level int, text string) bool {
	return len(indent(level))+utf8.RuneCountInString(text) <= maxLineWidth
}

func statesPayload(elements []*ast.SpecElement) bool {
	for _, element := range elements {
		if len(element.Payload) > 0 {
			return true
		}
	}
	return false
}

// specEntries lists the lines a spec writes, in canonical order. The keyword
// column is padded across this list alone, so a spec that omits given — the
// widest of the three — is written with no padding at all.
func specEntries(spec *ast.Spec) []specEntry {
	var entries []specEntry
	if len(spec.Given) > 0 {
		entries = append(entries, specEntry{
			keyword:   "given",
			value:     bracketed(specElementTexts(spec.Given)),
			elements:  spec.Given,
			bracketed: true,
		})
	}
	if spec.When != nil {
		entries = append(entries, specEntry{
			keyword:  "when",
			value:    specElementText(spec.When),
			elements: []*ast.SpecElement{spec.When},
		})
	}
	if outcome := formatOutcome(spec.Then); outcome != "" {
		entries = append(entries, specEntry{
			keyword:   "then",
			value:     outcome,
			elements:  thenEvents(spec.Then),
			bracketed: true,
		})
	}
	return entries
}

// thenEvents answers the elements a then clause holds, which is none for the
// three outcome shapes that name a single construct rather than a list. Those
// carry no payload, so an over-long one is left on its line.
func thenEvents(then ast.ThenClause) []*ast.SpecElement {
	if events, ok := then.(*ast.ThenEvents); ok {
		return events.Events
	}
	return nil
}

func formatOutcome(then ast.ThenClause) string {
	switch t := then.(type) {
	case *ast.ThenEvents:
		return bracketed(specElementTexts(t.Events))
	case *ast.ThenRejected:
		if t.InvariantName == "" {
			return ""
		}
		return fmt.Sprintf("rejected %s", t.InvariantName)
	case *ast.ThenView:
		if t.ViewName == "" {
			return ""
		}
		return fmt.Sprintf("view %s", t.ViewName)
	case *ast.ThenCommand:
		if t.CommandName == "" {
			return ""
		}
		return fmt.Sprintf("command %s", t.CommandName)
	default:
		return ""
	}
}

func specElementTexts(elements []*ast.SpecElement) []string {
	texts := make([]string, 0, len(elements))
	for _, element := range elements {
		texts = append(texts, specElementText(element))
	}
	return texts
}

// specElementText writes a reference beside the example payload it states. A
// payload stating nothing is written as no braces at all, so {} and an omitted
// payload format to the same text.
func specElementText(element *ast.SpecElement) string {
	if len(element.Payload) == 0 {
		return element.Name
	}

	fields := make([]string, 0, len(element.Payload))
	for _, field := range element.Payload {
		fields = append(fields, fmt.Sprintf("%s %s", payloadLabel(field), payloadLiteral(field)))
	}

	return fmt.Sprintf("%s { %s }", element.Name, strings.Join(fields, ", "))
}

// payloadLabel is a payload field's first column: the name and the colon
// binding it to its value. They pad as one, so a field is spelled the same
// whether its payload was written on one line or wrapped over several.
func payloadLabel(field *ast.PayloadField) string {
	return field.Name + ":"
}

// payloadLiteral writes a number or a boolean as the source text it was read
// from, so 12.50 is written back whole rather than reduced to 12.5.
func payloadLiteral(field *ast.PayloadField) string {
	if field.Kind == ast.StringLiteral {
		return quoted(field.Value)
	}

	return field.Value
}

package parser

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/diagnostic"
	"github.com/zclconf/go-cty/cty"
)

// Parse reads a model. It reports every fault as a diagnostic and returns
// whatever it could recover.
func Parse(source string, filename string) (*ast.Model, []*diagnostic.Entry) {
	r := &hclReader{filename: filename, src: source, lines: strings.Split(source, "\n")}

	file, diags := hclsyntax.ParseConfig([]byte(source), filename, hcl.InitialPos)
	r.report(diags)

	tokens, lexDiags := hclsyntax.LexConfig([]byte(source), filename, hcl.InitialPos)
	r.report(lexDiags)
	r.collectComments(tokens)

	if file == nil {
		return &ast.Model{Version: ast.SupportedVersion}, r.diags
	}
	body, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return &ast.Model{Version: ast.SupportedVersion}, r.diags
	}
	return r.model(body), r.diags
}

type hclReader struct {
	filename string
	src      string
	lines    []string
	diags    []*diagnostic.Entry
	comments []*ast.Comment
	taken    int
}

func (r *hclReader) report(diags hcl.Diagnostics) {
	for _, d := range diags {
		if d.Severity != hcl.DiagError || d.Subject == nil {
			continue
		}
		r.fail(d.Subject.Start, "%s", strings.ToLower(strings.TrimSuffix(d.Summary, ".")))
	}
}

func (r *hclReader) fail(p hcl.Pos, format string, args ...any) {
	r.diags = append(r.diags, &diagnostic.Entry{
		Filename: r.filename,
		Line:     p.Line,
		Column:   p.Column,
		Message:  fmt.Sprintf(format, args...),
	})
}

func (r *hclReader) at(p hcl.Pos) ast.Position {
	return ast.Position{Filename: r.filename, Line: p.Line, Column: p.Column}
}

func (r *hclReader) collectComments(tokens hclsyntax.Tokens) {
	for _, t := range tokens {
		if t.Type != hclsyntax.TokenComment {
			continue
		}
		r.comments = append(r.comments, &ast.Comment{
			Text:     strings.TrimRight(string(t.Bytes), "\r\n"),
			Position: r.at(t.Range.Start),
		})
	}
}

// commentsBefore hands over every comment written above the given line, in the
// order they appear. A construct takes the comments that lead up to it, which
// is what the formatter writes back above it.
func (r *hclReader) commentsBefore(p hcl.Pos) []*ast.Comment {
	var taken []*ast.Comment
	for r.taken < len(r.comments) && r.comments[r.taken].Line < p.Line {
		taken = append(taken, r.comments[r.taken])
		r.taken++
	}
	return taken
}

func hclAttrs(body *hclsyntax.Body) []*hclsyntax.Attribute {
	ordered := make([]*hclsyntax.Attribute, 0, len(body.Attributes))
	for _, a := range body.Attributes {
		ordered = append(ordered, a)
	}
	sort.Slice(ordered, func(i, j int) bool {
		return ordered[i].SrcRange.Start.Byte < ordered[j].SrcRange.Start.Byte
	})
	return ordered
}

// text reads a quoted string.
func (r *hclReader) text(a *hclsyntax.Attribute) (string, ast.Position) {
	start := a.Expr.Range().Start
	value, diags := a.Expr.Value(nil)
	if diags.HasErrors() || value.IsNull() || value.Type() != cty.String {
		r.fail(start, "expected a quoted string after %s", a.Name)
		return "", r.at(start)
	}
	return value.AsString(), r.at(start)
}

// name reads an unquoted reference such as `reads = AvailableRoomsView`.
func (r *hclReader) name(expr hcl.Expression) (string, ast.Position) {
	start := expr.Range().Start
	traversal, diags := hcl.AbsTraversalForExpr(expr)
	if diags.HasErrors() || len(traversal) != 1 {
		r.fail(start, "expected a name")
		return "", r.at(start)
	}
	return traversal.RootName(), r.at(start)
}

// names reads a list of unquoted references.
func (r *hclReader) names(expr hcl.Expression) ([]string, []ast.Position) {
	items, diags := hcl.ExprList(expr)
	if diags.HasErrors() {
		r.fail(expr.Range().Start, "expected a list of names")
		return nil, nil
	}
	values := make([]string, 0, len(items))
	positions := make([]ast.Position, 0, len(items))
	for _, item := range items {
		value, position := r.name(item)
		values = append(values, value)
		positions = append(positions, position)
	}
	return values, positions
}

func hclCall(expr hcl.Expression) (*hclsyntax.FunctionCallExpr, bool) {
	call, ok := expr.(*hclsyntax.FunctionCallExpr)
	return call, ok
}

func (r *hclReader) block(b *hclsyntax.Block, labels int) (*hclsyntax.Body, bool) {
	if len(b.Labels) != labels {
		r.fail(b.TypeRange.Start, "%s takes %d name(s), not %d", b.Type, labels, len(b.Labels))
		return nil, false
	}
	return b.Body, true
}

func (r *hclReader) label(b *hclsyntax.Block) (string, ast.Position) {
	if len(b.Labels) == 0 {
		return "", r.at(b.TypeRange.Start)
	}
	return b.Labels[0], r.at(b.LabelRanges[0].Start)
}

// hclEntry is one entry of a block body: an attribute or a nested block. A
// body is read in source order so that a comment lands on the construct it was
// written above.
type hclEntry struct {
	attr  *hclsyntax.Attribute
	block *hclsyntax.Block
	start hcl.Pos
}

func hclEntries(body *hclsyntax.Body) []hclEntry {
	entries := make([]hclEntry, 0, len(body.Attributes)+len(body.Blocks))
	for _, a := range body.Attributes {
		entries = append(entries, hclEntry{attr: a, start: a.SrcRange.Start})
	}
	for _, b := range body.Blocks {
		entries = append(entries, hclEntry{block: b, start: b.TypeRange.Start})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].start.Byte < entries[j].start.Byte })
	return entries
}

func (r *hclReader) model(body *hclsyntax.Body) *ast.Model {
	model := &ast.Model{Version: ast.SupportedVersion}

	for _, entry := range hclEntries(body) {
		switch {
		case entry.attr != nil && entry.attr.Name == "emod":
			// A file this tool cannot read is reported on its header alone.
			// Reading on would bury that line under diagnostics about a
			// grammar the file was never written in.
			if !r.version(model, entry.attr) {
				return model
			}
		case entry.attr != nil:
			r.fail(entry.attr.NameRange.Start, "unexpected %s at the top level", entry.attr.Name)
		case entry.block.Type == "model":
			r.modelBlock(model, entry.block)
		case entry.block.Type == "actor":
			model.Actors = append(model.Actors, r.actor(entry.block))
		case entry.block.Type == "context":
			model.Contexts = append(model.Contexts, r.context(entry.block))
		default:
			r.fail(entry.block.TypeRange.Start,
				"unexpected %q block; expected one of: model, actor, context", entry.block.Type)
		}
	}
	return model
}

func (r *hclReader) version(model *ast.Model, a *hclsyntax.Attribute) bool {
	value, diags := a.Expr.Value(nil)
	if diags.HasErrors() || value.IsNull() || value.Type() != cty.Number {
		r.fail(a.Expr.Range().Start, "expected a whole number after emod")
		return false
	}
	declared, _ := value.AsBigFloat().Int64()
	model.Version = int(declared)
	model.VersionDeclared = true
	if model.Version != ast.SupportedVersion {
		r.fail(a.NameRange.Start, "unsupported version %d: this tool supports emod version %d",
			model.Version, ast.SupportedVersion)
		return false
	}
	return true
}

func (r *hclReader) modelBlock(model *ast.Model, b *hclsyntax.Block) {
	model.Comments = r.commentsBefore(b.TypeRange.Start)
	model.Name, model.NamePos = r.label(b)
	model.OpenPos = r.at(b.OpenBraceRange.Start)
	model.ClosePos = r.at(b.CloseBraceRange.Start)

	body, ok := r.block(b, 1)
	if !ok {
		return
	}
	for _, a := range hclAttrs(body) {
		if a.Name == "description" {
			model.Description, model.DescriptionPos = r.text(a)
			continue
		}
		r.fail(a.NameRange.Start, "unexpected %s in model", a.Name)
	}
}

func (r *hclReader) actor(b *hclsyntax.Block) *ast.Actor {
	actor := &ast.Actor{Comments: r.commentsBefore(b.TypeRange.Start)}
	actor.Name, actor.NamePos = r.label(b)
	actor.OpenPos = r.at(b.OpenBraceRange.Start)
	actor.ClosePos = r.at(b.CloseBraceRange.Start)

	body, ok := r.block(b, 1)
	if !ok {
		return actor
	}
	for _, a := range hclAttrs(body) {
		if a.Name == "description" {
			actor.Description, actor.DescriptionPos = r.text(a)
			continue
		}
		r.fail(a.NameRange.Start, "unexpected %s in actor", a.Name)
	}
	return actor
}

func (r *hclReader) context(b *hclsyntax.Block) *ast.Context {
	context := &ast.Context{Comments: r.commentsBefore(b.TypeRange.Start)}
	context.Name, context.NamePos = r.label(b)
	context.OpenPos = r.at(b.OpenBraceRange.Start)
	context.ClosePos = r.at(b.CloseBraceRange.Start)

	body, ok := r.block(b, 1)
	if !ok {
		return context
	}
	for _, entry := range hclEntries(body) {
		if a := entry.attr; a != nil {
			switch a.Name {
			case "description":
				context.Description, context.DescriptionPos = r.text(a)
			case "mode":
				context.Mode, context.ModePos = r.name(a.Expr)
			default:
				r.fail(a.NameRange.Start, "unexpected %s in context", a.Name)
			}
			continue
		}
		switch entry.block.Type {
		case "invariants":
			context.Invariants = append(context.Invariants, r.invariants(entry.block)...)
		case "aggregate":
			context.Aggregates = append(context.Aggregates, r.aggregate(entry.block))
		case "slice":
			context.Slices = append(context.Slices, r.slice(entry.block))
		default:
			r.fail(entry.block.TypeRange.Start, "unexpected %q block in context", entry.block.Type)
		}
	}
	return context
}

func (r *hclReader) aggregate(b *hclsyntax.Block) *ast.Aggregate {
	aggregate := &ast.Aggregate{Comments: r.commentsBefore(b.TypeRange.Start)}
	aggregate.Name, aggregate.NamePos = r.label(b)
	aggregate.OpenPos = r.at(b.OpenBraceRange.Start)
	aggregate.ClosePos = r.at(b.CloseBraceRange.Start)

	body, ok := r.block(b, 1)
	if !ok {
		return aggregate
	}
	for _, entry := range hclEntries(body) {
		if a := entry.attr; a != nil {
			if a.Name == "description" {
				aggregate.Description, aggregate.DescriptionPos = r.text(a)
				continue
			}
			r.fail(a.NameRange.Start, "unexpected %s in aggregate", a.Name)
			continue
		}
		switch entry.block.Type {
		case "invariants":
			aggregate.Invariants = append(aggregate.Invariants, r.invariants(entry.block)...)
		case "slice":
			aggregate.Slices = append(aggregate.Slices, r.slice(entry.block))
		default:
			r.fail(entry.block.TypeRange.Start, "unexpected %q block in aggregate", entry.block.Type)
		}
	}
	return aggregate
}

func (r *hclReader) invariants(b *hclsyntax.Block) []*ast.Invariant {
	leading := r.commentsBefore(b.TypeRange.Start)
	body, ok := r.block(b, 0)
	if !ok {
		return nil
	}
	var invariants []*ast.Invariant
	for _, a := range hclAttrs(body) {
		invariant := &ast.Invariant{
			Comments: append(leading, r.commentsBefore(a.NameRange.Start)...),
			Name:     a.Name,
			NamePos:  r.at(a.NameRange.Start),
		}
		leading = nil
		invariant.Statement, invariant.StatementPos = r.text(a)
		invariants = append(invariants, invariant)
	}
	return invariants
}

func (r *hclReader) slice(b *hclsyntax.Block) *ast.Slice {
	slice := &ast.Slice{Comments: r.commentsBefore(b.TypeRange.Start)}
	slice.Name, slice.NamePos = r.label(b)
	slice.OpenPos = r.at(b.OpenBraceRange.Start)
	slice.ClosePos = r.at(b.CloseBraceRange.Start)

	body, ok := r.block(b, 1)
	if !ok {
		return slice
	}
	for _, entry := range hclEntries(body) {
		if a := entry.attr; a != nil {
			switch a.Name {
			case "description":
				slice.Description, slice.DescriptionPos = r.text(a)
			case "flow":
				r.flow(slice, a)
			default:
				r.fail(a.NameRange.Start, "unexpected %s in slice", a.Name)
			}
			continue
		}
		switch entry.block.Type {
		case "trigger":
			slice.Trigger = r.trigger(entry.block)
		case "command":
			slice.Commands = append(slice.Commands, r.command(entry.block))
		case "event":
			slice.Events = append(slice.Events, r.event(entry.block))
		case "view":
			slice.Views = append(slice.Views, r.view(entry.block))
		case "automation":
			slice.Automations = append(slice.Automations, r.automation(entry.block))
		case "translation":
			slice.Translations = append(slice.Translations, r.translation(entry.block))
		case "spec":
			slice.Specs = append(slice.Specs, r.spec(entry.block))
		default:
			r.fail(entry.block.TypeRange.Start, "unexpected %q block in slice", entry.block.Type)
		}
	}
	return slice
}

// retiredTriggerKind names the kinds a trigger once carried ahead of its name,
// so a model written before they were retired is told what replaced them rather
// than that a trigger takes one name.
func retiredTriggerKind(kind string) string {
	if kind == "Schedule" || kind == "Processor" {
		return fmt.Sprintf("trigger %s is no longer supported: use an automation with every", kind)
	}
	return fmt.Sprintf("trigger %s is no longer supported: drop the word %s", kind, kind)
}

func (r *hclReader) trigger(b *hclsyntax.Block) *ast.Trigger {
	if len(b.Labels) == 2 {
		r.fail(b.LabelRanges[0].Start, "%s", retiredTriggerKind(b.Labels[0]))
		return &ast.Trigger{}
	}
	trigger := &ast.Trigger{Comments: r.commentsBefore(b.TypeRange.Start)}
	trigger.Name, trigger.NamePos = r.label(b)
	trigger.OpenPos = r.at(b.OpenBraceRange.Start)
	trigger.ClosePos = r.at(b.CloseBraceRange.Start)

	body, ok := r.block(b, 1)
	if !ok {
		return trigger
	}
	for _, a := range hclAttrs(body) {
		switch a.Name {
		case "description":
			trigger.Description, trigger.DescriptionPos = r.text(a)
		case "actor":
			trigger.Actor, trigger.ActorPos = r.name(a.Expr)
		case "reads":
			trigger.Reads, trigger.ReadsPos = r.name(a.Expr)
		default:
			r.fail(a.NameRange.Start, "unexpected %s in trigger", a.Name)
		}
	}
	return trigger
}

func (r *hclReader) command(b *hclsyntax.Block) *ast.Command {
	command := &ast.Command{Comments: r.commentsBefore(b.TypeRange.Start)}
	command.Name, command.NamePos = r.label(b)
	command.OpenPos = r.at(b.OpenBraceRange.Start)
	command.ClosePos = r.at(b.CloseBraceRange.Start)

	body, ok := r.block(b, 1)
	if !ok {
		return command
	}
	for _, entry := range hclEntries(body) {
		if a := entry.attr; a != nil {
			if a.Name == "description" {
				command.Description, command.DescriptionPos = r.text(a)
				continue
			}
			r.fail(a.NameRange.Start, "unexpected %s in command", a.Name)
			continue
		}
		switch entry.block.Type {
		case "fields":
			command.Fields = r.fields(entry.block)
		case "decides_on":
			command.DecidesOn = r.decidesOn(entry.block)
		default:
			r.fail(entry.block.TypeRange.Start, "unexpected %q block in command", entry.block.Type)
		}
	}
	return command
}

func (r *hclReader) event(b *hclsyntax.Block) *ast.Event {
	event := &ast.Event{Comments: r.commentsBefore(b.TypeRange.Start)}
	event.Name, event.NamePos = r.label(b)
	event.OpenPos = r.at(b.OpenBraceRange.Start)
	event.ClosePos = r.at(b.CloseBraceRange.Start)

	body, ok := r.block(b, 1)
	if !ok {
		return event
	}
	for _, entry := range hclEntries(body) {
		if a := entry.attr; a != nil {
			switch a.Name {
			case "description":
				event.Description, event.DescriptionPos = r.text(a)
			case "type":
				event.WireType, event.WireTypePos = r.text(a)
			case "source":
				r.source(event, a)
			default:
				r.fail(a.NameRange.Start, "unexpected %s in event", a.Name)
			}
			continue
		}
		switch entry.block.Type {
		case "fields":
			event.Fields = r.fields(entry.block)
		case "tags":
			event.Tags = r.tags(entry.block)
		default:
			r.fail(entry.block.TypeRange.Start, "unexpected %q block in event", entry.block.Type)
		}
	}
	return event
}

// source reads `source = external("Partner Booking")`.
func (r *hclReader) source(event *ast.Event, a *hclsyntax.Attribute) {
	event.SourcePos = r.at(a.NameRange.Start)
	call, ok := hclCall(a.Expr)
	if !ok || call.Name != "external" || len(call.Args) != 1 {
		r.fail(a.Expr.Range().Start, `expected external("<name>") after source in event`)
		return
	}
	event.Source = call.Name
	value, diags := call.Args[0].Value(nil)
	if diags.HasErrors() || value.IsNull() || value.Type() != cty.String {
		r.fail(call.Args[0].Range().Start, "expected a quoted string after source external in event")
		return
	}
	event.ExternalName = value.AsString()
	event.ExternalNamePos = r.at(call.Args[0].Range().Start)
}

func (r *hclReader) view(b *hclsyntax.Block) *ast.View {
	view := &ast.View{Comments: r.commentsBefore(b.TypeRange.Start)}
	view.Name, view.NamePos = r.label(b)
	view.OpenPos = r.at(b.OpenBraceRange.Start)
	view.ClosePos = r.at(b.CloseBraceRange.Start)

	body, ok := r.block(b, 1)
	if !ok {
		return view
	}
	for _, entry := range hclEntries(body) {
		if a := entry.attr; a != nil {
			switch a.Name {
			case "description":
				view.Description, view.DescriptionPos = r.text(a)
			case "subscribes":
				view.Subscribes, view.SubscribesPos = r.names(a.Expr)
			default:
				r.fail(a.NameRange.Start, "unexpected %s in view", a.Name)
			}
			continue
		}
		if entry.block.Type == "fields" {
			view.Fields = r.fields(entry.block)
			continue
		}
		r.fail(entry.block.TypeRange.Start, "unexpected %q block in view", entry.block.Type)
	}
	return view
}

func (r *hclReader) automation(b *hclsyntax.Block) *ast.Automation {
	automation := &ast.Automation{Comments: r.commentsBefore(b.TypeRange.Start)}
	automation.Name, automation.NamePos = r.label(b)
	automation.OpenPos = r.at(b.OpenBraceRange.Start)
	automation.ClosePos = r.at(b.CloseBraceRange.Start)

	body, ok := r.block(b, 1)
	if !ok {
		return automation
	}
	for _, entry := range hclEntries(body) {
		if a := entry.attr; a != nil {
			switch a.Name {
			case "description":
				automation.Description, automation.DescriptionPos = r.text(a)
			case "on":
				automation.OnEvent, automation.OnEventPos = r.name(a.Expr)
			case "every":
				automation.Schedule, automation.SchedulePos = r.text(a)
			case "after":
				automation.After, automation.AfterPos = r.text(a)
			case "reads":
				automation.Reads, automation.ReadsPos = r.name(a.Expr)
			case "command":
				automation.Command, automation.CommandPos = r.name(a.Expr)
			default:
				r.fail(a.NameRange.Start, "unexpected %s in automation", a.Name)
			}
			continue
		}
		if entry.block.Type == "target" {
			r.target(automation, entry.block)
			continue
		}
		r.fail(entry.block.TypeRange.Start, "unexpected %q block in automation", entry.block.Type)
	}
	return automation
}

func (r *hclReader) target(automation *ast.Automation, b *hclsyntax.Block) {
	body, ok := r.block(b, 0)
	if !ok {
		return
	}
	for _, a := range hclAttrs(body) {
		if a.Name == "context" {
			automation.TargetContext, automation.TargetContextPos = r.name(a.Expr)
			continue
		}
		r.fail(a.NameRange.Start, "unexpected %s in target", a.Name)
	}
}

func (r *hclReader) translation(b *hclsyntax.Block) *ast.Translation {
	translation := &ast.Translation{Comments: r.commentsBefore(b.TypeRange.Start)}
	translation.Name, translation.NamePos = r.label(b)
	translation.OpenPos = r.at(b.OpenBraceRange.Start)
	translation.ClosePos = r.at(b.CloseBraceRange.Start)

	body, ok := r.block(b, 1)
	if !ok {
		return translation
	}
	for _, entry := range hclEntries(body) {
		if a := entry.attr; a != nil {
			switch a.Name {
			case "description":
				translation.Description, translation.DescriptionPos = r.text(a)
			case "external_system":
				translation.ExternalSystem, translation.ExternalPos = r.text(a)
			case "reads":
				translation.Reads, translation.ReadsPos = r.name(a.Expr)
			case "command":
				translation.Command, translation.CommandPos = r.name(a.Expr)
			default:
				r.fail(a.NameRange.Start, "unexpected %s in translation", a.Name)
			}
			continue
		}
		if entry.block.Type == "event" {
			translation.Event = r.event(entry.block)
			continue
		}
		r.fail(entry.block.TypeRange.Start, "unexpected %q block in translation", entry.block.Type)
	}
	return translation
}

func (r *hclReader) fields(b *hclsyntax.Block) []*ast.Field {
	body, ok := r.block(b, 0)
	if !ok {
		return nil
	}
	var fields []*ast.Field
	for _, a := range hclAttrs(body) {
		field := &ast.Field{Name: a.Name, NamePos: r.at(a.NameRange.Start)}
		if call, isCall := hclCall(a.Expr); isCall {
			if call.Name != "required" && call.Name != "optional" {
				r.fail(call.NameRange.Start, "expected required() or optional() around the type of %s", a.Name)
				continue
			}
			if len(call.Args) != 1 {
				r.fail(call.NameRange.Start, "%s takes one type", call.Name)
				continue
			}
			field.Modifier = call.Name
			field.ModPos = r.at(call.NameRange.Start)
			field.Type, field.TypePos = r.name(call.Args[0])
		} else {
			field.Type, field.TypePos = r.name(a.Expr)
		}
		fields = append(fields, field)
	}
	return fields
}

func (r *hclReader) tags(b *hclsyntax.Block) []ast.TagEntry {
	body, ok := r.block(b, 0)
	if !ok {
		return nil
	}
	var tags []ast.TagEntry
	for _, a := range hclAttrs(body) {
		tag := ast.TagEntry{Key: a.Name, KeyPos: r.at(a.NameRange.Start)}
		tag.FieldRef, tag.FieldRefPos = r.name(a.Expr)
		tags = append(tags, tag)
	}
	return tags
}

func (r *hclReader) decidesOn(b *hclsyntax.Block) *ast.DecidesOnClause {
	clause := &ast.DecidesOnClause{
		Comments: r.commentsBefore(b.TypeRange.Start),
		OpenPos:  r.at(b.OpenBraceRange.Start),
		ClosePos: r.at(b.CloseBraceRange.Start),
	}
	body, ok := r.block(b, 0)
	if !ok {
		return clause
	}
	for _, a := range hclAttrs(body) {
		switch a.Name {
		case "events":
			clause.Events, clause.EventsPos = r.names(a.Expr)
		case "where":
			clause.Predicate = r.predicate(a.Expr)
		default:
			r.fail(a.NameRange.Start, "unexpected %s in decides_on", a.Name)
		}
	}
	return clause
}

// predicate reads a tag expression. HCL supplies the grouping and the
// precedence, so the shape below is the expression it already parsed.
func (r *hclReader) predicate(expr hcl.Expression) ast.PredicateExpr {
	switch e := expr.(type) {
	case *hclsyntax.ParenthesesExpr:
		return r.predicate(e.Expression)
	case *hclsyntax.UnaryOpExpr:
		if e.Op != hclsyntax.OpLogicalNot {
			r.fail(e.SymbolRange.Start, "expected ! before a tag expression")
			return nil
		}
		return &ast.NotExpr{OpPos: r.at(e.SymbolRange.Start), Expr: r.predicate(e.Val)}
	case *hclsyntax.BinaryOpExpr:
		operator := ""
		switch e.Op {
		case hclsyntax.OpLogicalAnd:
			operator = "and"
		case hclsyntax.OpLogicalOr:
			operator = "or"
		default:
			r.fail(e.SrcRange.Start, "expected && or || between tag expressions")
			return nil
		}
		return &ast.LogicalExpr{
			Left:     r.predicate(e.LHS),
			Operator: operator,
			OpPos:    r.at(e.LHS.Range().End),
			Right:    r.predicate(e.RHS),
		}
	case *hclsyntax.FunctionCallExpr:
		if e.Name != "tag" || len(e.Args) != 2 {
			r.fail(e.NameRange.Start, "expected tag(<key>, <field>)")
			return nil
		}
		key, keyPos := r.name(e.Args[0])
		value, valuePos := r.name(e.Args[1])
		return &ast.TagPredicate{
			Field:    key,
			FieldPos: keyPos,
			Operator: "=",
			OpPos:    keyPos,
			Value:    value,
			ValuePos: valuePos,
		}
	}
	r.fail(expr.Range().Start, "expected a tag expression")
	return nil
}

func (r *hclReader) spec(b *hclsyntax.Block) *ast.Spec {
	spec := &ast.Spec{Comments: r.commentsBefore(b.TypeRange.Start)}
	spec.Name, spec.NamePos = r.label(b)
	spec.OpenPos = r.at(b.OpenBraceRange.Start)
	spec.ClosePos = r.at(b.CloseBraceRange.Start)

	body, ok := r.block(b, 1)
	if !ok {
		return spec
	}
	for _, a := range hclAttrs(body) {
		switch a.Name {
		case "given":
			spec.Given = r.specElements(a.Expr)
		case "when":
			spec.When = r.specElement(a.Expr)
		case "then":
			spec.Then = r.then(a.Expr)
		default:
			r.fail(a.NameRange.Start, "unexpected %s in spec", a.Name)
		}
	}
	return spec
}

func (r *hclReader) then(expr hcl.Expression) ast.ThenClause {
	if call, ok := hclCall(expr); ok {
		if len(call.Args) != 1 {
			r.fail(call.NameRange.Start, "%s names one element", call.Name)
			return nil
		}
		name, position := r.name(call.Args[0])
		switch call.Name {
		case "rejected":
			return &ast.ThenRejected{InvariantName: name, InvariantPos: position}
		case "view":
			return &ast.ThenView{ViewName: name, ViewPos: position}
		case "command":
			return &ast.ThenCommand{CommandName: name, CommandPos: position}
		}
		r.fail(call.NameRange.Start, "expected a list of events, rejected(), view() or command() after then")
		return nil
	}
	return &ast.ThenEvents{Events: r.specElements(expr)}
}

func (r *hclReader) specElements(expr hcl.Expression) []*ast.SpecElement {
	items, diags := hcl.ExprList(expr)
	if diags.HasErrors() {
		r.fail(expr.Range().Start, "expected a list of events")
		return nil
	}
	elements := make([]*ast.SpecElement, 0, len(items))
	for _, item := range items {
		elements = append(elements, r.specElement(item))
	}
	return elements
}

// specElement reads a reference, on its own or qualified by an example payload
// such as ReserveRoom({ roomId = "R-204" }).
func (r *hclReader) specElement(expr hcl.Expression) *ast.SpecElement {
	call, ok := hclCall(expr)
	if !ok {
		name, position := r.name(expr)
		return &ast.SpecElement{Name: name, NamePos: position}
	}
	element := &ast.SpecElement{Name: call.Name, NamePos: r.at(call.NameRange.Start)}
	if len(call.Args) != 1 {
		r.fail(call.NameRange.Start, "an example payload is one set of field values")
		return element
	}
	element.Payload = r.payload(call.Args[0])
	return element
}

func (r *hclReader) payload(expr hcl.Expression) []*ast.PayloadField {
	pairs, diags := hcl.ExprMap(expr)
	if diags.HasErrors() {
		r.fail(expr.Range().Start, "expected a set of field values")
		return nil
	}
	var fields []*ast.PayloadField
	for _, pair := range pairs {
		name := hcl.ExprAsKeyword(pair.Key)
		if name == "" {
			r.fail(pair.Key.Range().Start, "expected a field name")
			continue
		}
		field := &ast.PayloadField{
			Name:     name,
			NamePos:  r.at(pair.Key.Range().Start),
			ValuePos: r.at(pair.Value.Range().Start),
		}
		field.Value, field.Kind = r.literal(pair.Value)
		fields = append(fields, field)
	}
	return fields
}

func (r *hclReader) literal(expr hcl.Expression) (string, ast.LiteralKind) {
	start := expr.Range().Start
	value, diags := expr.Value(nil)
	if diags.HasErrors() || value.IsNull() {
		r.fail(start, "expected a string, a number or a boolean")
		return "", 0
	}
	switch value.Type() {
	case cty.String:
		return value.AsString(), ast.StringLiteral
	case cty.Bool:
		if value.True() {
			return "true", ast.BooleanLiteral
		}
		return "false", ast.BooleanLiteral
	case cty.Number:
		// The digits are taken from the source so that a value reaches the
		// exports exactly as it was written.
		written := r.textAt(expr.Range())
		if strings.Contains(written, ".") {
			return written, ast.DecimalLiteral
		}
		return written, ast.IntegerLiteral
	}
	r.fail(start, "expected a string, a number or a boolean")
	return "", 0
}

func (r *hclReader) textAt(rng hcl.Range) string {
	if rng.Start.Byte < 0 || rng.End.Byte > len(r.src) {
		return ""
	}
	return r.src[rng.Start.Byte:rng.End.Byte]
}

// flow reads the heredoc holding a slice's flow entries. The arrows have no
// operator in HCL, so the lines are read as they are written.
func (r *hclReader) flow(slice *ast.Slice, a *hclsyntax.Attribute) {
	leading := r.commentsBefore(a.NameRange.Start)
	first := a.Expr.Range().Start.Line + 1
	last := a.Expr.Range().End.Line

	for number := first; number <= last && number <= len(r.lines); number++ {
		line := r.lines[number-1]
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || !strings.Contains(trimmed, "->") {
			continue
		}
		kind, rest, found := strings.Cut(trimmed, ":")
		if !found {
			r.fail(hcl.Pos{Line: number, Column: 1}, "expected command -> event: or command -> rejected: in flow")
			continue
		}
		left, right, split := strings.Cut(rest, "->")
		if !split {
			r.fail(hcl.Pos{Line: number, Column: 1}, "expected -> between the two names in flow")
			continue
		}
		from := strings.TrimSpace(left)
		to := strings.TrimSpace(right)
		fromPos := r.at(hcl.Pos{Line: number, Column: strings.Index(line, from) + 1})
		toPos := r.at(hcl.Pos{Line: number, Column: strings.LastIndex(line, to) + 1})

		switch strings.Join(strings.Fields(kind), " ") {
		case "command -> event":
			slice.Flows = append(slice.Flows, &ast.Flow{
				Comments:    leading,
				CommandName: from,
				CommandPos:  fromPos,
				EventName:   to,
				EventPos:    toPos,
			})
		case "command -> rejected":
			slice.Rejections = append(slice.Rejections, &ast.Rejection{
				Comments:      leading,
				CommandName:   from,
				CommandPos:    fromPos,
				InvariantName: to,
				InvariantPos:  toPos,
			})
		default:
			r.fail(hcl.Pos{Line: number, Column: 1}, "unexpected flow entry %q", kind)
			continue
		}
		leading = nil
	}
}

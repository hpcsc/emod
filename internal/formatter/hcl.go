package formatter

import (
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/hpcsc/emod/internal/ast"
	"github.com/zclconf/go-cty/cty"
)

// FormatHCL writes the model in HCL. hclwrite owns the layout, so the columns
// line up on the `=` rather than on the name and the type.
func FormatHCL(model *ast.Model) string {
	file := hclwrite.NewEmptyFile()
	body := file.Body()

	body.SetAttributeValue("emod", cty.NumberIntVal(int64(pinnedVersion(model))))
	body.AppendNewline()

	writeComments(body, model.Comments)
	block := body.AppendNewBlock("model", []string{model.Name})
	writeText(block.Body(), "description", model.Description)

	for _, actor := range model.Actors {
		gap(body)
		writeComments(body, actor.Comments)
		actorBlock := body.AppendNewBlock("actor", []string{actor.Name})
		writeText(actorBlock.Body(), "description", actor.Description)
	}

	for _, context := range model.Contexts {
		gap(body)
		writeComments(body, context.Comments)
		writeContext(body.AppendNewBlock("context", []string{context.Name}).Body(), context)
	}

	return indentHeredocs(string(hclwrite.Format(file.Bytes())))
}

func writeContext(body *hclwrite.Body, context *ast.Context) {
	writeText(body, "description", context.Description)
	if context.Mode != "" {
		writeRef(body, "mode", context.Mode)
	}
	writeInvariantsHCL(body, context.Invariants)
	for _, aggregate := range context.Aggregates {
		gap(body)
		writeComments(body, aggregate.Comments)
		writeAggregate(body.AppendNewBlock("aggregate", []string{aggregate.Name}).Body(), aggregate)
	}
	for _, slice := range context.Slices {
		gap(body)
		writeComments(body, slice.Comments)
		writeSlice(body.AppendNewBlock("slice", []string{slice.Name}).Body(), slice)
	}
}

func writeAggregate(body *hclwrite.Body, aggregate *ast.Aggregate) {
	writeText(body, "description", aggregate.Description)
	writeInvariantsHCL(body, aggregate.Invariants)
	for _, slice := range aggregate.Slices {
		gap(body)
		writeComments(body, slice.Comments)
		writeSlice(body.AppendNewBlock("slice", []string{slice.Name}).Body(), slice)
	}
}

func writeInvariantsHCL(body *hclwrite.Body, invariants []*ast.Invariant) {
	if len(invariants) == 0 {
		return
	}
	gap(body)
	for _, invariant := range invariants {
		writeComments(body, invariant.Comments)
	}
	block := body.AppendNewBlock("invariants", nil)
	for _, invariant := range invariants {
		block.Body().SetAttributeValue(invariant.Name, cty.StringVal(invariant.Statement))
	}
}

func writeSlice(body *hclwrite.Body, slice *ast.Slice) {
	writeText(body, "description", slice.Description)

	if slice.Trigger != nil {
		gap(body)
		writeComments(body, slice.Trigger.Comments)
		trigger := body.AppendNewBlock("trigger", []string{slice.Trigger.Name}).Body()
		writeText(trigger, "description", slice.Trigger.Description)
		writeRef(trigger, "actor", slice.Trigger.Actor)
		writeRef(trigger, "reads", slice.Trigger.Reads)
	}

	for _, command := range slice.Commands {
		gap(body)
		writeComments(body, command.Comments)
		writeCommand(body.AppendNewBlock("command", []string{command.Name}).Body(), command)
	}
	for _, event := range slice.Events {
		gap(body)
		writeComments(body, event.Comments)
		writeEvent(body.AppendNewBlock("event", []string{event.Name}).Body(), event)
	}
	for _, view := range slice.Views {
		gap(body)
		writeComments(body, view.Comments)
		writeView(body.AppendNewBlock("view", []string{view.Name}).Body(), view)
	}
	for _, automation := range slice.Automations {
		gap(body)
		writeComments(body, automation.Comments)
		writeAutomation(body.AppendNewBlock("automation", []string{automation.Name}).Body(), automation)
	}
	for _, translation := range slice.Translations {
		gap(body)
		writeComments(body, translation.Comments)
		writeTranslation(body.AppendNewBlock("translation", []string{translation.Name}).Body(), translation)
	}

	writeFlow(body, slice)

	for _, spec := range slice.Specs {
		gap(body)
		writeComments(body, spec.Comments)
		writeSpec(body.AppendNewBlock("spec", []string{spec.Name}).Body(), spec)
	}
}

func writeCommand(body *hclwrite.Body, command *ast.Command) {
	writeText(body, "description", command.Description)
	writeFieldsHCL(body, command.Fields)
	if command.DecidesOn == nil {
		return
	}
	gap(body)
	block := body.AppendNewBlock("decides_on", nil).Body()
	block.SetAttributeRaw("events", refListTokens(command.DecidesOn.Events))
	if command.DecidesOn.Predicate != nil {
		block.SetAttributeRaw("where", predicateTokens(command.DecidesOn.Predicate))
	}
}

func writeEvent(body *hclwrite.Body, event *ast.Event) {
	writeText(body, "description", event.Description)
	writeText(body, "type", event.WireType)
	if event.Source != "" {
		body.SetAttributeRaw("source", callTokens(event.Source, quotedTokens(event.ExternalName)))
	}
	if len(event.Tags) > 0 {
		gap(body)
		tags := body.AppendNewBlock("tags", nil).Body()
		for _, tag := range event.Tags {
			tags.SetAttributeTraversal(tag.Key, rootOf(tag.FieldRef))
		}
	}
	writeFieldsHCL(body, event.Fields)
}

func writeView(body *hclwrite.Body, view *ast.View) {
	writeText(body, "description", view.Description)
	if len(view.Subscribes) > 0 {
		body.SetAttributeRaw("subscribes", refListTokens(view.Subscribes))
	}
	writeFieldsHCL(body, view.Fields)
}

func writeAutomation(body *hclwrite.Body, automation *ast.Automation) {
	writeText(body, "description", automation.Description)
	writeRef(body, "on", automation.OnEvent)
	writeText(body, "every", automation.Schedule)
	writeText(body, "after", automation.After)
	writeRef(body, "reads", automation.Reads)
	writeRef(body, "command", automation.Command)
	if automation.TargetContext == "" {
		return
	}
	gap(body)
	target := body.AppendNewBlock("target", nil).Body()
	target.SetAttributeTraversal("context", rootOf(automation.TargetContext))
}

func writeTranslation(body *hclwrite.Body, translation *ast.Translation) {
	writeText(body, "description", translation.Description)
	writeText(body, "external_system", translation.ExternalSystem)
	writeRef(body, "reads", translation.Reads)
	writeRef(body, "command", translation.Command)
	if translation.Event == nil {
		return
	}
	gap(body)
	writeComments(body, translation.Event.Comments)
	writeEvent(body.AppendNewBlock("event", []string{translation.Event.Name}).Body(), translation.Event)
}

func writeFieldsHCL(body *hclwrite.Body, fields []*ast.Field) {
	if len(fields) == 0 {
		return
	}
	gap(body)
	block := body.AppendNewBlock("fields", nil).Body()
	for _, field := range fields {
		if field.Modifier == "" {
			block.SetAttributeTraversal(field.Name, rootOf(field.Type))
			continue
		}
		block.SetAttributeRaw(field.Name, callTokens(field.Modifier, identTokens(field.Type)))
	}
}

func writeSpec(body *hclwrite.Body, spec *ast.Spec) {
	if len(spec.Given) > 0 {
		body.SetAttributeRaw("given", elementListTokens(spec.Given))
	}
	if spec.When != nil {
		body.SetAttributeRaw("when", elementTokens(spec.When))
	}
	switch then := spec.Then.(type) {
	case *ast.ThenEvents:
		body.SetAttributeRaw("then", elementListTokens(then.Events))
	case *ast.ThenRejected:
		body.SetAttributeRaw("then", callTokens("rejected", identTokens(then.InvariantName)))
	case *ast.ThenView:
		body.SetAttributeRaw("then", callTokens("view", identTokens(then.ViewName)))
	case *ast.ThenCommand:
		body.SetAttributeRaw("then", callTokens("command", identTokens(then.CommandName)))
	}
}

// writeFlow writes the slice's flow entries as a heredoc, in the shape the
// reader expects: the event entries first, then the rejections.
func writeFlow(body *hclwrite.Body, slice *ast.Slice) {
	if len(slice.Flows) == 0 && len(slice.Rejections) == 0 {
		return
	}
	var lines []string
	for _, flow := range slice.Flows {
		lines = append(lines, "command -> event:    "+flow.CommandName+" -> "+flow.EventName)
	}
	for _, rejection := range slice.Rejections {
		lines = append(lines, "command -> rejected: "+rejection.CommandName+" -> "+rejection.InvariantName)
	}

	gap(body)
	for _, flow := range slice.Flows {
		writeComments(body, flow.Comments)
	}
	for _, rejection := range slice.Rejections {
		writeComments(body, rejection.Comments)
	}
	body.SetAttributeRaw("flow", heredocTokens("FLOW", lines))
}

func writeComments(body *hclwrite.Body, comments []*ast.Comment) {
	for _, comment := range comments {
		body.AppendUnstructuredTokens(hclwrite.Tokens{
			{Type: hclsyntax.TokenComment, Bytes: []byte(comment.Text + "\n")},
		})
	}
}

func writeText(body *hclwrite.Body, name, value string) {
	if value == "" {
		return
	}
	body.SetAttributeValue(name, cty.StringVal(value))
}

func writeRef(body *hclwrite.Body, name, value string) {
	if value == "" {
		return
	}
	body.SetAttributeTraversal(name, rootOf(value))
}

func rootOf(name string) hcl.Traversal {
	return hcl.Traversal{hcl.TraverseRoot{Name: name}}
}

func token(kind hclsyntax.TokenType, text string) *hclwrite.Token {
	return &hclwrite.Token{Type: kind, Bytes: []byte(text)}
}

func identTokens(name string) hclwrite.Tokens {
	return hclwrite.Tokens{token(hclsyntax.TokenIdent, name)}
}

func quotedTokens(value string) hclwrite.Tokens {
	return hclwrite.TokensForValue(cty.StringVal(value))
}

func callTokens(name string, argument hclwrite.Tokens) hclwrite.Tokens {
	tokens := hclwrite.Tokens{token(hclsyntax.TokenIdent, name), token(hclsyntax.TokenOParen, "(")}
	tokens = append(tokens, argument...)
	return append(tokens, token(hclsyntax.TokenCParen, ")"))
}

func listTokens(items []hclwrite.Tokens) hclwrite.Tokens {
	tokens := hclwrite.Tokens{token(hclsyntax.TokenOBrack, "[")}
	for i, item := range items {
		if i > 0 {
			tokens = append(tokens, token(hclsyntax.TokenComma, ","))
		}
		tokens = append(tokens, item...)
	}
	return append(tokens, token(hclsyntax.TokenCBrack, "]"))
}

func refListTokens(names []string) hclwrite.Tokens {
	items := make([]hclwrite.Tokens, 0, len(names))
	for _, name := range names {
		items = append(items, identTokens(name))
	}
	return listTokens(items)
}

func elementListTokens(elements []*ast.SpecElement) hclwrite.Tokens {
	items := make([]hclwrite.Tokens, 0, len(elements))
	for _, element := range elements {
		items = append(items, elementTokens(element))
	}
	return listTokens(items)
}

func elementTokens(element *ast.SpecElement) hclwrite.Tokens {
	if len(element.Payload) == 0 {
		return identTokens(element.Name)
	}
	return callTokens(element.Name, payloadTokens(element.Payload))
}

func payloadTokens(fields []*ast.PayloadField) hclwrite.Tokens {
	tokens := hclwrite.Tokens{token(hclsyntax.TokenOBrace, "{")}
	for i, field := range fields {
		if i > 0 {
			tokens = append(tokens, token(hclsyntax.TokenComma, ","))
		}
		tokens = append(tokens, token(hclsyntax.TokenIdent, field.Name), token(hclsyntax.TokenEqual, "="))
		tokens = append(tokens, literalTokens(field)...)
	}
	return append(tokens, token(hclsyntax.TokenCBrace, "}"))
}

func literalTokens(field *ast.PayloadField) hclwrite.Tokens {
	switch field.Kind {
	case ast.IntegerLiteral, ast.DecimalLiteral:
		return hclwrite.Tokens{token(hclsyntax.TokenNumberLit, field.Value)}
	case ast.BooleanLiteral:
		return identTokens(field.Value)
	default:
		return quotedTokens(field.Value)
	}
}

func predicateTokens(predicate ast.PredicateExpr) hclwrite.Tokens {
	switch p := predicate.(type) {
	case *ast.TagPredicate:
		argument := append(identTokens(p.Field), token(hclsyntax.TokenComma, ","))
		return callTokens("tag", append(argument, identTokens(p.Value)...))
	case *ast.NotExpr:
		return append(hclwrite.Tokens{token(hclsyntax.TokenBang, "!")}, operandTokens(p.Expr)...)
	case *ast.LogicalExpr:
		operator := "&&"
		if p.Operator == "or" {
			operator = "||"
		}
		tokens := operandTokens(p.Left)
		tokens = append(tokens, token(hclsyntax.TokenOr, operator))
		return append(tokens, operandTokens(p.Right)...)
	}
	return nil
}

// operandTokens brackets a nested combination, so that the expression HCL
// parses back groups the way the model states it.
func operandTokens(predicate ast.PredicateExpr) hclwrite.Tokens {
	if _, nested := predicate.(*ast.LogicalExpr); !nested {
		return predicateTokens(predicate)
	}
	tokens := hclwrite.Tokens{token(hclsyntax.TokenOParen, "(")}
	tokens = append(tokens, predicateTokens(predicate)...)
	return append(tokens, token(hclsyntax.TokenCParen, ")"))
}

func heredocTokens(marker string, lines []string) hclwrite.Tokens {
	tokens := hclwrite.Tokens{token(hclsyntax.TokenOHeredoc, "<<-"+marker+"\n")}
	for _, line := range lines {
		tokens = append(tokens, token(hclsyntax.TokenStringLit, "  "+line+"\n"))
	}
	return append(tokens, token(hclsyntax.TokenCHeredoc, marker))
}

// gap separates one entry from the one before it, and writes nothing at the
// head of a block.
func gap(body *hclwrite.Body) {
	if len(body.BuildTokens(nil)) > 0 {
		body.AppendNewline()
	}
}

// indentHeredocs lines a heredoc's content and its closing marker up with the
// attribute that opens it. hclwrite leaves both where they were written, and
// `<<-` strips the indentation again when the file is read.
func indentHeredocs(text string) string {
	lines := strings.Split(text, "\n")
	marker, indent := "", ""
	for i, line := range lines {
		if marker == "" {
			opener := strings.Index(line, "<<-")
			if opener < 0 {
				continue
			}
			marker = strings.TrimSpace(line[opener+3:])
			indent = line[:len(line)-len(strings.TrimLeft(line, " "))]
			continue
		}
		if strings.TrimSpace(line) == marker {
			lines[i] = indent + marker
			marker = ""
			continue
		}
		lines[i] = indent + "  " + strings.TrimSpace(line)
	}
	return strings.Join(lines, "\n")
}

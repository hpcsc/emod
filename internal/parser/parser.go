package parser

import (
	"fmt"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/diagnostic"
	"github.com/hpcsc/emod/internal/lexer"
)

type Instance struct {
	tokens      []*lexer.Token
	pos         int
	diagnostics []*diagnostic.Entry
	filename    string
}

func New(tokens []*lexer.Token, filename string) *Instance {
	return &Instance{
		tokens:   tokens,
		pos:      0,
		filename: filename,
	}
}

func (p *Instance) Parse() (*ast.Model, []*diagnostic.Entry) {
	model := &ast.Model{}

	for !p.isAtEnd() {
		if p.check(lexer.KeywordModel) {
			name, pos := p.parseModel()
			model.Name = name
			model.NamePos = pos
		} else if p.check(lexer.KeywordActor) {
			if actor := p.parseActor(); actor != nil {
				model.Actors = append(model.Actors, actor)
			}
		} else if p.check(lexer.KeywordContext) {
			if ctx := p.parseContext(); ctx != nil {
				model.Contexts = append(model.Contexts, ctx)
			}
		} else if p.check(lexer.EOF) {
			break
		} else {
			p.error(fmt.Sprintf("unrecognized keyword %q; expected one of: model, actor, context", p.peek().Value))
			p.advance()
		}
	}

	return model, p.diagnostics
}

func (p *Instance) parseModel() (string, ast.Position) {
	p.consume(lexer.KeywordModel, "expected model")
	if !p.check(lexer.String) {
		p.error(fmt.Sprintf("expected quoted string after \"model\", got %q", p.peek().Value))
		return "", ast.Position{}
	}

	tok := p.advance()
	return tok.Value, p.position(tok)
}

func (p *Instance) parseActor() *ast.Actor {
	p.consume(lexer.KeywordActor, "expected actor")
	if !p.check(lexer.String) {
		p.error(fmt.Sprintf("expected quoted string after \"actor\", got %q", p.peek().Value))
		return nil
	}

	tok := p.advance()
	return &ast.Actor{
		Name:    tok.Value,
		NamePos: p.position(tok),
	}
}

func (p *Instance) parseContext() *ast.Context {
	p.consume(lexer.KeywordContext, "expected context")
	if !p.check(lexer.String) {
		p.error(fmt.Sprintf("expected quoted string after \"context\", got %q", p.peek().Value))
		return nil
	}

	nameTok := p.advance()
	context := &ast.Context{
		Name:    nameTok.Value,
		NamePos: p.position(nameTok),
	}

	if !p.check(lexer.OpenBrace) {
		p.error("expected { after context name")
		return nil
	}
	openTok := p.advance()
	context.OpenPos = p.position(openTok)

	for !p.check(lexer.CloseBrace) && !p.isAtEnd() {
		if p.check(lexer.KeywordAggregate) {
			if agg := p.parseAggregate(); agg != nil {
				context.Aggregates = append(context.Aggregates, agg)
			}
		} else {
			p.error("expected aggregate in context")
			p.advance()
		}
	}

	if !p.check(lexer.CloseBrace) {
		p.error(fmt.Sprintf("unclosed brace for \"context\" block opened at line %d", context.OpenPos.Line))
		return context
	}
	closeTok := p.advance()
	context.ClosePos = p.position(closeTok)

	return context
}

func (p *Instance) parseAggregate() *ast.Aggregate {
	p.consume(lexer.KeywordAggregate, "expected aggregate")
	if !p.check(lexer.String) {
		p.error(fmt.Sprintf("expected quoted string after \"aggregate\", got %q", p.peek().Value))
		return nil
	}

	nameTok := p.advance()
	aggregate := &ast.Aggregate{
		Name:    nameTok.Value,
		NamePos: p.position(nameTok),
	}

	if !p.check(lexer.OpenBrace) {
		p.error("expected { after aggregate name")
		return nil
	}
	openTok := p.advance()
	aggregate.OpenPos = p.position(openTok)

	for !p.check(lexer.CloseBrace) && !p.isAtEnd() {
		if p.check(lexer.KeywordSlice) {
			if slice := p.parseSlice(); slice != nil {
				aggregate.Slices = append(aggregate.Slices, slice)
			}
		} else {
			p.error("expected slice in aggregate")
			p.advance()
		}
	}

	if !p.check(lexer.CloseBrace) {
		p.error(fmt.Sprintf("unclosed brace for \"aggregate\" block opened at line %d", aggregate.OpenPos.Line))
		return aggregate
	}
	closeTok := p.advance()
	aggregate.ClosePos = p.position(closeTok)

	return aggregate
}

func (p *Instance) parseSlice() *ast.Slice {
	p.consume(lexer.KeywordSlice, "expected slice")
	if !p.check(lexer.String) {
		p.error(fmt.Sprintf("expected quoted string after \"slice\", got %q", p.peek().Value))
		return nil
	}

	nameTok := p.advance()
	slice := &ast.Slice{
		Name:    nameTok.Value,
		NamePos: p.position(nameTok),
	}

	if !p.check(lexer.OpenBrace) {
		p.error("expected { after slice name")
		return nil
	}
	openTok := p.advance()
	slice.OpenPos = p.position(openTok)

	for !p.check(lexer.CloseBrace) && !p.isAtEnd() {
		if p.check(lexer.KeywordCommand) {
			if cmd := p.parseCommand(); cmd != nil {
				slice.Commands = append(slice.Commands, cmd)
			}
		} else if p.check(lexer.KeywordEvent) {
			if evt := p.parseEvent(); evt != nil {
				slice.Events = append(slice.Events, evt)
			}
		} else if p.check(lexer.KeywordFlow) {
			slice.Flows = append(slice.Flows, p.parseFlows()...)
		} else {
			p.error("expected command, event, or flow in slice")
			p.advance()
		}
	}

	if !p.check(lexer.CloseBrace) {
		p.error(fmt.Sprintf("unclosed brace for \"slice\" block opened at line %d", slice.OpenPos.Line))
		return slice
	}
	closeTok := p.advance()
	slice.ClosePos = p.position(closeTok)

	return slice
}

func (p *Instance) parseCommand() *ast.Command {
	p.consume(lexer.KeywordCommand, "expected command")
	if !p.check(lexer.Identifier) {
		p.error("expected identifier after command")
		return nil
	}

	nameTok := p.advance()
	command := &ast.Command{
		Name:    nameTok.Value,
		NamePos: p.position(nameTok),
	}

	if !p.check(lexer.OpenBrace) {
		p.error("expected { after command name")
		return nil
	}
	openTok := p.advance()
	command.OpenPos = p.position(openTok)

	for !p.check(lexer.CloseBrace) && !p.isAtEnd() {
		if p.check(lexer.KeywordFields) {
			command.Fields = p.parseFields()
		} else {
			p.error("expected fields in command")
			p.advance()
		}
	}

	if !p.check(lexer.CloseBrace) {
		p.error(fmt.Sprintf("unclosed brace for \"command\" block opened at line %d", command.OpenPos.Line))
		return command
	}
	closeTok := p.advance()
	command.ClosePos = p.position(closeTok)

	return command
}

func (p *Instance) parseEvent() *ast.Event {
	p.consume(lexer.KeywordEvent, "expected event")
	if !p.check(lexer.Identifier) {
		p.error("expected identifier after event")
		return nil
	}

	nameTok := p.advance()
	event := &ast.Event{
		Name:    nameTok.Value,
		NamePos: p.position(nameTok),
	}

	if !p.check(lexer.OpenBrace) {
		p.error("expected { after event name")
		return nil
	}
	openTok := p.advance()
	event.OpenPos = p.position(openTok)

	for !p.check(lexer.CloseBrace) && !p.isAtEnd() {
		if p.check(lexer.KeywordFields) {
			event.Fields = p.parseFields()
		} else {
			p.error("expected fields in event")
			p.advance()
		}
	}

	if !p.check(lexer.CloseBrace) {
		p.error(fmt.Sprintf("unclosed brace for \"event\" block opened at line %d", event.OpenPos.Line))
		return event
	}
	closeTok := p.advance()
	event.ClosePos = p.position(closeTok)

	return event
}

func (p *Instance) parseFields() []*ast.Field {
	var fields []*ast.Field
	p.consume(lexer.KeywordFields, "expected fields")
	if !p.check(lexer.OpenBrace) {
		p.error("expected { after fields")
		return fields
	}
	openTok := p.advance()
	openLine := openTok.Line

	for !p.check(lexer.CloseBrace) && !p.isAtEnd() {
		if p.check(lexer.Identifier) {
			if field := p.parseField(); field != nil {
				fields = append(fields, field)
			}
		} else {
			p.error("expected field definition")
			p.advance()
		}
	}

	if !p.check(lexer.CloseBrace) {
		p.error(fmt.Sprintf("unclosed brace for \"fields\" block opened at line %d", openLine))
		return fields
	}
	p.advance()

	return fields
}

func (p *Instance) parseField() *ast.Field {
	if !p.check(lexer.Identifier) {
		return nil
	}

	nameTok := p.advance()
	field := &ast.Field{
		Name:    nameTok.Value,
		NamePos: p.position(nameTok),
	}

	if !p.check(lexer.Identifier) {
		p.error("expected field type")
		return field
	}
	typeTok := p.advance()
	field.Type = typeTok.Value
	field.TypePos = p.position(typeTok)

	if p.check(lexer.Identifier) {
		modTok := p.advance()
		field.Modifier = modTok.Value
		field.ModPos = p.position(modTok)
	}

	return field
}

func (p *Instance) parseFlows() []*ast.Flow {
	var flows []*ast.Flow
	p.consume(lexer.KeywordFlow, "expected flow")
	if !p.check(lexer.OpenBrace) {
		p.error("expected { after flow")
		return flows
	}
	openTok := p.advance()
	openLine := openTok.Line

	for !p.check(lexer.CloseBrace) && !p.isAtEnd() {
		if p.check(lexer.KeywordCommand) {
			if flow := p.parseFlowEntry(); flow != nil {
				flows = append(flows, flow)
			}
		} else {
			p.error("expected command in flow")
			p.advance()
		}
	}

	if !p.check(lexer.CloseBrace) {
		p.error(fmt.Sprintf("unclosed brace for \"flow\" block opened at line %d", openLine))
		return flows
	}
	p.advance()

	return flows
}

func (p *Instance) parseFlowEntry() *ast.Flow {
	p.advance() // consume "command"
	if !p.check(lexer.Arrow) {
		p.error("expected -> after command in flow")
		p.advance()
		return nil
	}
	p.advance()
	if !p.check(lexer.KeywordEvent) {
		p.error("expected event after -> in flow")
		p.advance()
		return nil
	}
	p.advance()
	if !p.check(lexer.Colon) {
		p.error("expected : in flow")
		p.advance()
		return nil
	}
	p.advance()
	if !p.check(lexer.Identifier) {
		p.error("expected command identifier after : in flow")
		p.advance()
		return nil
	}
	cmdTok := p.advance()
	if !p.check(lexer.Arrow) {
		p.error("expected -> between command and event identifiers")
		p.advance()
		return nil
	}
	p.advance()
	if !p.check(lexer.Identifier) {
		p.error("expected event identifier")
		p.advance()
		return nil
	}
	evtTok := p.advance()

	return &ast.Flow{
		CommandName: cmdTok.Value,
		CommandPos:  p.position(cmdTok),
		EventName:   evtTok.Value,
		EventPos:    p.position(evtTok),
	}
}

func (p *Instance) position(tok *lexer.Token) ast.Position {
	return ast.Position{
		Filename: p.filename,
		Line:     tok.Line,
		Column:   tok.Column,
	}
}

func (p *Instance) peek() *lexer.Token {
	if p.pos >= len(p.tokens) {
		return &lexer.Token{Type: lexer.EOF}
	}
	return p.tokens[p.pos]
}

func (p *Instance) advance() *lexer.Token {
	tok := p.peek()
	if !p.isAtEnd() {
		p.pos++
	}
	return tok
}

func (p *Instance) check(typ lexer.Kind) bool {
	if p.isAtEnd() {
		return false
	}
	return p.tokens[p.pos].Type == typ
}

func (p *Instance) consume(typ lexer.Kind, msg string) {
	if !p.check(typ) {
		p.error(msg)
		return
	}
	p.advance()
}

func (p *Instance) isAtEnd() bool {
	return p.pos >= len(p.tokens) || p.tokens[p.pos].Type == lexer.EOF
}

func (p *Instance) error(msg string) {
	tok := p.peek()
	p.diagnostics = append(p.diagnostics, &diagnostic.Entry{
		Filename: p.filename,
		Line:     tok.Line,
		Column:   tok.Column,
		Message:  msg,
	})
}

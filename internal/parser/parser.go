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
			p.parseModel(model)
		} else if p.check(lexer.KeywordActor) {
			p.parseActor(model)
		} else if p.check(lexer.KeywordContext) {
			p.parseContext(model)
		} else if p.check(lexer.EOF) {
			break
		} else {
			p.error(fmt.Sprintf("unrecognized keyword %q; expected one of: model, actor, context", p.peek().Value))
			p.advance()
		}
	}

	return model, p.diagnostics
}

func (p *Instance) parseModel(model *ast.Model) {
	p.consume(lexer.KeywordModel, "expected model")
	if !p.check(lexer.String) {
		p.error(fmt.Sprintf("expected quoted string after \"model\", got %q", p.peek().Value))
		return
	}

	tok := p.advance()
	model.Name = tok.Value
	model.NamePos = ast.Position{
		Filename: p.filename,
		Line:     tok.Line,
		Column:   tok.Column,
	}
}

func (p *Instance) parseActor(model *ast.Model) {
	p.consume(lexer.KeywordActor, "expected actor")
	if !p.check(lexer.String) {
		p.error(fmt.Sprintf("expected quoted string after \"actor\", got %q", p.peek().Value))
		return
	}

	tok := p.advance()
	actor := &ast.Actor{
		Name: tok.Value,
		NamePos: ast.Position{
			Filename: p.filename,
			Line:     tok.Line,
			Column:   tok.Column,
		},
	}
	model.Actors = append(model.Actors, actor)
}

func (p *Instance) parseContext(model *ast.Model) {
	p.consume(lexer.KeywordContext, "expected context")
	if !p.check(lexer.String) {
		p.error(fmt.Sprintf("expected quoted string after \"context\", got %q", p.peek().Value))
		return
	}

	nameTok := p.advance()
	context := &ast.Context{
		Name: nameTok.Value,
		NamePos: ast.Position{
			Filename: p.filename,
			Line:     nameTok.Line,
			Column:   nameTok.Column,
		},
	}

	if !p.check(lexer.OpenBrace) {
		p.error("expected { after context name")
		return
	}
	openTok := p.advance()
	context.OpenPos = ast.Position{
		Filename: p.filename,
		Line:     openTok.Line,
		Column:   openTok.Column,
	}

	for !p.check(lexer.CloseBrace) && !p.isAtEnd() {
		if p.check(lexer.KeywordAggregate) {
			p.parseAggregate(context)
		} else {
			p.error("expected aggregate in context")
			p.advance()
		}
	}

	if !p.check(lexer.CloseBrace) {
		p.error(fmt.Sprintf("unclosed brace for \"context\" block opened at line %d", context.OpenPos.Line))
		return
	}
	closeTok := p.advance()
	context.ClosePos = ast.Position{
		Filename: p.filename,
		Line:     closeTok.Line,
		Column:   closeTok.Column,
	}

	model.Contexts = append(model.Contexts, context)
}

func (p *Instance) parseAggregate(context *ast.Context) {
	p.consume(lexer.KeywordAggregate, "expected aggregate")
	if !p.check(lexer.String) {
		p.error(fmt.Sprintf("expected quoted string after \"aggregate\", got %q", p.peek().Value))
		return
	}

	nameTok := p.advance()
	aggregate := &ast.Aggregate{
		Name: nameTok.Value,
		NamePos: ast.Position{
			Filename: p.filename,
			Line:     nameTok.Line,
			Column:   nameTok.Column,
		},
	}

	if !p.check(lexer.OpenBrace) {
		p.error("expected { after aggregate name")
		return
	}
	openTok := p.advance()
	aggregate.OpenPos = ast.Position{
		Filename: p.filename,
		Line:     openTok.Line,
		Column:   openTok.Column,
	}

	for !p.check(lexer.CloseBrace) && !p.isAtEnd() {
		if p.check(lexer.KeywordSlice) {
			p.parseSlice(aggregate)
		} else {
			p.error("expected slice in aggregate")
			p.advance()
		}
	}

	if !p.check(lexer.CloseBrace) {
		p.error(fmt.Sprintf("unclosed brace for \"aggregate\" block opened at line %d", aggregate.OpenPos.Line))
		return
	}
	closeTok := p.advance()
	aggregate.ClosePos = ast.Position{
		Filename: p.filename,
		Line:     closeTok.Line,
		Column:   closeTok.Column,
	}

	context.Aggregates = append(context.Aggregates, aggregate)
}

func (p *Instance) parseSlice(aggregate *ast.Aggregate) {
	p.consume(lexer.KeywordSlice, "expected slice")
	if !p.check(lexer.String) {
		p.error(fmt.Sprintf("expected quoted string after \"slice\", got %q", p.peek().Value))
		return
	}

	nameTok := p.advance()
	slice := &ast.Slice{
		Name: nameTok.Value,
		NamePos: ast.Position{
			Filename: p.filename,
			Line:     nameTok.Line,
			Column:   nameTok.Column,
		},
	}

	if !p.check(lexer.OpenBrace) {
		p.error("expected { after slice name")
		return
	}
	openTok := p.advance()
	slice.OpenPos = ast.Position{
		Filename: p.filename,
		Line:     openTok.Line,
		Column:   openTok.Column,
	}

	for !p.check(lexer.CloseBrace) && !p.isAtEnd() {
		if p.check(lexer.KeywordCommand) {
			p.parseCommand(slice)
		} else if p.check(lexer.KeywordEvent) {
			p.parseEvent(slice)
		} else if p.check(lexer.KeywordFlow) {
			p.parseFlow(slice)
		} else {
			p.error("expected command, event, or flow in slice")
			p.advance()
		}
	}

	if !p.check(lexer.CloseBrace) {
		p.error(fmt.Sprintf("unclosed brace for \"slice\" block opened at line %d", slice.OpenPos.Line))
		return
	}
	closeTok := p.advance()
	slice.ClosePos = ast.Position{
		Filename: p.filename,
		Line:     closeTok.Line,
		Column:   closeTok.Column,
	}

	aggregate.Slices = append(aggregate.Slices, slice)
}

func (p *Instance) parseCommand(slice *ast.Slice) {
	p.consume(lexer.KeywordCommand, "expected command")
	if !p.check(lexer.Identifier) {
		p.error("expected identifier after command")
		return
	}

	nameTok := p.advance()
	command := &ast.Command{
		Name: nameTok.Value,
		NamePos: ast.Position{
			Filename: p.filename,
			Line:     nameTok.Line,
			Column:   nameTok.Column,
		},
	}

	if !p.check(lexer.OpenBrace) {
		p.error("expected { after command name")
		return
	}
	openTok := p.advance()
	command.OpenPos = ast.Position{
		Filename: p.filename,
		Line:     openTok.Line,
		Column:   openTok.Column,
	}

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
		return
	}
	closeTok := p.advance()
	command.ClosePos = ast.Position{
		Filename: p.filename,
		Line:     closeTok.Line,
		Column:   closeTok.Column,
	}

	slice.Commands = append(slice.Commands, command)
}

func (p *Instance) parseEvent(slice *ast.Slice) {
	p.consume(lexer.KeywordEvent, "expected event")
	if !p.check(lexer.Identifier) {
		p.error("expected identifier after event")
		return
	}

	nameTok := p.advance()
	event := &ast.Event{
		Name: nameTok.Value,
		NamePos: ast.Position{
			Filename: p.filename,
			Line:     nameTok.Line,
			Column:   nameTok.Column,
		},
	}

	if !p.check(lexer.OpenBrace) {
		p.error("expected { after event name")
		return
	}
	openTok := p.advance()
	event.OpenPos = ast.Position{
		Filename: p.filename,
		Line:     openTok.Line,
		Column:   openTok.Column,
	}

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
		return
	}
	closeTok := p.advance()
	event.ClosePos = ast.Position{
		Filename: p.filename,
		Line:     closeTok.Line,
		Column:   closeTok.Column,
	}

	slice.Events = append(slice.Events, event)
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
			field := p.parseField()
			if field != nil {
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
		Name: nameTok.Value,
		NamePos: ast.Position{
			Filename: p.filename,
			Line:     nameTok.Line,
			Column:   nameTok.Column,
		},
	}

	if !p.check(lexer.Identifier) {
		p.error("expected field type")
		return field
	}
	typeTok := p.advance()
	field.Type = typeTok.Value
	field.TypePos = ast.Position{
		Filename: p.filename,
		Line:     typeTok.Line,
		Column:   typeTok.Column,
	}

	if p.check(lexer.Identifier) {
		modTok := p.advance()
		field.Modifier = modTok.Value
		field.ModPos = ast.Position{
			Filename: p.filename,
			Line:     modTok.Line,
			Column:   modTok.Column,
		}
	}

	return field
}

func (p *Instance) parseFlow(slice *ast.Slice) {
	p.consume(lexer.KeywordFlow, "expected flow")
	if !p.check(lexer.OpenBrace) {
		p.error("expected { after flow")
		return
	}
	openTok := p.advance()
	openLine := openTok.Line

	for !p.check(lexer.CloseBrace) && !p.isAtEnd() {
		if p.check(lexer.KeywordCommand) {
			p.advance()
			if !p.check(lexer.Arrow) {
				p.error("expected -> after command in flow")
				p.advance()
				continue
			}
			p.advance()
			if !p.check(lexer.KeywordEvent) {
				p.error("expected event after -> in flow")
				p.advance()
				continue
			}
			p.advance()
			if !p.check(lexer.Colon) {
				p.error("expected : in flow")
				p.advance()
				continue
			}
			p.advance()
			if !p.check(lexer.Identifier) {
				p.error("expected command identifier after : in flow")
				p.advance()
				continue
			}
			cmdTok := p.advance()
			if !p.check(lexer.Arrow) {
				p.error("expected -> between command and event identifiers")
				p.advance()
				continue
			}
			p.advance()
			if !p.check(lexer.Identifier) {
				p.error("expected event identifier")
				p.advance()
				continue
			}
			evtTok := p.advance()
			flow := &ast.Flow{
				CommandName: cmdTok.Value,
				CommandPos: ast.Position{
					Filename: p.filename,
					Line:     cmdTok.Line,
					Column:   cmdTok.Column,
				},
				EventName: evtTok.Value,
				EventPos: ast.Position{
					Filename: p.filename,
					Line:     evtTok.Line,
					Column:   evtTok.Column,
				},
			}
			slice.Flows = append(slice.Flows, flow)
		} else {
			p.error("expected command in flow")
			p.advance()
		}
	}

	if !p.check(lexer.CloseBrace) {
		p.error(fmt.Sprintf("unclosed brace for \"flow\" block opened at line %d", openLine))
		return
	}
	p.advance()
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
	diag := &diagnostic.Entry{
		Filename: p.filename,
		Line:     tok.Line,
		Column:   tok.Column,
		Message:  msg,
	}
	p.diagnostics = append(p.diagnostics, diag)
}

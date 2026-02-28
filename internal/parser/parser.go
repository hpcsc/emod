package parser

import (
	"fmt"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/diagnostic"
	"github.com/hpcsc/emod/internal/lexer"
)

// Parser consumes a token stream and produces an AST.
type Parser struct {
	tokens      []*lexer.Token
	pos         int
	diagnostics []*diagnostic.Diagnostic
	filename    string
}

// New creates a new parser for the given tokens.
func New(tokens []*lexer.Token, filename string) *Parser {
	return &Parser{
		tokens:   tokens,
		pos:      0,
		filename: filename,
	}
}

// Parse parses the token stream into an AST model.
func (p *Parser) Parse() (*ast.Model, []*diagnostic.Diagnostic) {
	model := &ast.Model{}

	for !p.isAtEnd() {
		if p.check(lexer.TokenKeywordModel) {
			p.parseModel(model)
		} else if p.check(lexer.TokenKeywordActor) {
			p.parseActor(model)
		} else if p.check(lexer.TokenKeywordContext) {
			p.parseContext(model)
		} else if p.check(lexer.TokenEOF) {
			break
		} else {
			p.error(fmt.Sprintf("unrecognized keyword %q; expected one of: model, actor, context", p.peek().Value))
			p.advance()
		}
	}

	return model, p.diagnostics
}

func (p *Parser) parseModel(model *ast.Model) {
	p.consume(lexer.TokenKeywordModel, "expected model")
	if !p.check(lexer.TokenString) {
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

func (p *Parser) parseActor(model *ast.Model) {
	p.consume(lexer.TokenKeywordActor, "expected actor")
	if !p.check(lexer.TokenString) {
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

func (p *Parser) parseContext(model *ast.Model) {
	p.consume(lexer.TokenKeywordContext, "expected context")
	if !p.check(lexer.TokenString) {
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

	if !p.check(lexer.TokenOpenBrace) {
		p.error("expected { after context name")
		return
	}
	openTok := p.advance()
	context.OpenPos = ast.Position{
		Filename: p.filename,
		Line:     openTok.Line,
		Column:   openTok.Column,
	}

	for !p.check(lexer.TokenCloseBrace) && !p.isAtEnd() {
		if p.check(lexer.TokenKeywordAggregate) {
			p.parseAggregate(context)
		} else {
			p.error("expected aggregate in context")
			p.advance()
		}
	}

	if !p.check(lexer.TokenCloseBrace) {
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

func (p *Parser) parseAggregate(context *ast.Context) {
	p.consume(lexer.TokenKeywordAggregate, "expected aggregate")
	if !p.check(lexer.TokenString) {
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

	if !p.check(lexer.TokenOpenBrace) {
		p.error("expected { after aggregate name")
		return
	}
	openTok := p.advance()
	aggregate.OpenPos = ast.Position{
		Filename: p.filename,
		Line:     openTok.Line,
		Column:   openTok.Column,
	}

	for !p.check(lexer.TokenCloseBrace) && !p.isAtEnd() {
		if p.check(lexer.TokenKeywordSlice) {
			p.parseSlice(aggregate)
		} else {
			p.error("expected slice in aggregate")
			p.advance()
		}
	}

	if !p.check(lexer.TokenCloseBrace) {
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

func (p *Parser) parseSlice(aggregate *ast.Aggregate) {
	p.consume(lexer.TokenKeywordSlice, "expected slice")
	if !p.check(lexer.TokenString) {
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

	if !p.check(lexer.TokenOpenBrace) {
		p.error("expected { after slice name")
		return
	}
	openTok := p.advance()
	slice.OpenPos = ast.Position{
		Filename: p.filename,
		Line:     openTok.Line,
		Column:   openTok.Column,
	}

	for !p.check(lexer.TokenCloseBrace) && !p.isAtEnd() {
		if p.check(lexer.TokenKeywordCommand) {
			p.parseCommand(slice)
		} else if p.check(lexer.TokenKeywordEvent) {
			p.parseEvent(slice)
		} else if p.check(lexer.TokenKeywordFlow) {
			p.parseFlow(slice)
		} else {
			p.error("expected command, event, or flow in slice")
			p.advance()
		}
	}

	if !p.check(lexer.TokenCloseBrace) {
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

func (p *Parser) parseCommand(slice *ast.Slice) {
	p.consume(lexer.TokenKeywordCommand, "expected command")
	if !p.check(lexer.TokenIdentifier) {
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

	if !p.check(lexer.TokenOpenBrace) {
		p.error("expected { after command name")
		return
	}
	openTok := p.advance()
	command.OpenPos = ast.Position{
		Filename: p.filename,
		Line:     openTok.Line,
		Column:   openTok.Column,
	}

	for !p.check(lexer.TokenCloseBrace) && !p.isAtEnd() {
		if p.check(lexer.TokenKeywordFields) {
			command.Fields = p.parseFields()
		} else {
			p.error("expected fields in command")
			p.advance()
		}
	}

	if !p.check(lexer.TokenCloseBrace) {
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

func (p *Parser) parseEvent(slice *ast.Slice) {
	p.consume(lexer.TokenKeywordEvent, "expected event")
	if !p.check(lexer.TokenIdentifier) {
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

	if !p.check(lexer.TokenOpenBrace) {
		p.error("expected { after event name")
		return
	}
	openTok := p.advance()
	event.OpenPos = ast.Position{
		Filename: p.filename,
		Line:     openTok.Line,
		Column:   openTok.Column,
	}

	for !p.check(lexer.TokenCloseBrace) && !p.isAtEnd() {
		if p.check(lexer.TokenKeywordFields) {
			event.Fields = p.parseFields()
		} else {
			p.error("expected fields in event")
			p.advance()
		}
	}

	if !p.check(lexer.TokenCloseBrace) {
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

func (p *Parser) parseFields() []*ast.Field {
	var fields []*ast.Field
	p.consume(lexer.TokenKeywordFields, "expected fields")
	if !p.check(lexer.TokenOpenBrace) {
		p.error("expected { after fields")
		return fields
	}
	openTok := p.advance()
	openLine := openTok.Line

	for !p.check(lexer.TokenCloseBrace) && !p.isAtEnd() {
		if p.check(lexer.TokenIdentifier) {
			field := p.parseField()
			if field != nil {
				fields = append(fields, field)
			}
		} else {
			p.error("expected field definition")
			p.advance()
		}
	}

	if !p.check(lexer.TokenCloseBrace) {
		p.error(fmt.Sprintf("unclosed brace for \"fields\" block opened at line %d", openLine))
		return fields
	}
	p.advance()

	return fields
}

func (p *Parser) parseField() *ast.Field {
	if !p.check(lexer.TokenIdentifier) {
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

	if !p.check(lexer.TokenIdentifier) {
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

	if p.check(lexer.TokenIdentifier) {
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

func (p *Parser) parseFlow(slice *ast.Slice) {
	p.consume(lexer.TokenKeywordFlow, "expected flow")
	if !p.check(lexer.TokenOpenBrace) {
		p.error("expected { after flow")
		return
	}
	openTok := p.advance()
	openLine := openTok.Line

	for !p.check(lexer.TokenCloseBrace) && !p.isAtEnd() {
		if p.check(lexer.TokenKeywordCommand) {
			p.advance()
			if !p.check(lexer.TokenArrow) {
				p.error("expected -> after command in flow")
				p.advance()
				continue
			}
			p.advance()
			if !p.check(lexer.TokenKeywordEvent) {
				p.error("expected event after -> in flow")
				p.advance()
				continue
			}
			p.advance()
			if !p.check(lexer.TokenColon) {
				p.error("expected : in flow")
				p.advance()
				continue
			}
			p.advance()
			if !p.check(lexer.TokenIdentifier) {
				p.error("expected command identifier after : in flow")
				p.advance()
				continue
			}
			cmdTok := p.advance()
			if !p.check(lexer.TokenArrow) {
				p.error("expected -> between command and event identifiers")
				p.advance()
				continue
			}
			p.advance()
			if !p.check(lexer.TokenIdentifier) {
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

	if !p.check(lexer.TokenCloseBrace) {
		p.error(fmt.Sprintf("unclosed brace for \"flow\" block opened at line %d", openLine))
		return
	}
	p.advance()
}

func (p *Parser) peek() *lexer.Token {
	if p.pos >= len(p.tokens) {
		return &lexer.Token{Type: lexer.TokenEOF}
	}
	return p.tokens[p.pos]
}

func (p *Parser) advance() *lexer.Token {
	tok := p.peek()
	if !p.isAtEnd() {
		p.pos++
	}
	return tok
}

func (p *Parser) check(typ lexer.TokenType) bool {
	if p.isAtEnd() {
		return false
	}
	return p.tokens[p.pos].Type == typ
}

func (p *Parser) consume(typ lexer.TokenType, msg string) {
	if !p.check(typ) {
		p.error(msg)
		return
	}
	p.advance()
}

func (p *Parser) isAtEnd() bool {
	return p.pos >= len(p.tokens) || p.tokens[p.pos].Type == lexer.TokenEOF
}

func (p *Parser) error(msg string) {
	tok := p.peek()
	diag := &diagnostic.Diagnostic{
		Filename: p.filename,
		Line:     tok.Line,
		Column:   tok.Column,
		Message:  msg,
	}
	p.diagnostics = append(p.diagnostics, diag)
}

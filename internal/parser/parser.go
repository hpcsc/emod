package parser

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/diagnostic"
	"github.com/hpcsc/emod/internal/lexer"
)

type topLevelHandler func(model *ast.Model)

type Instance struct {
	tokens      []*lexer.Token
	pos         int
	diagnostics []*diagnostic.Entry
	filename    string
	handlers    map[lexer.Kind]topLevelHandler
}

func New(tokens []*lexer.Token, filename string) *Instance {
	p := &Instance{
		tokens:   tokens,
		pos:      0,
		filename: filename,
	}
	p.handlers = map[lexer.Kind]topLevelHandler{
		lexer.KeywordModel: func(model *ast.Model) {
			name, pos := p.parseModel()
			model.Name = name
			model.NamePos = pos
		},
		lexer.KeywordActor: func(model *ast.Model) {
			if actor := p.parseActor(); actor != nil {
				model.Actors = append(model.Actors, actor)
			}
		},
		lexer.KeywordContext: func(model *ast.Model) {
			if ctx := p.parseContext(); ctx != nil {
				model.Contexts = append(model.Contexts, ctx)
			}
		},
	}
	return p
}

func (p *Instance) Parse() (*ast.Model, []*diagnostic.Entry) {
	model := &ast.Model{}

	for !p.isAtEnd() {
		if p.check(lexer.EOF) {
			break
		}

		if handler, ok := p.handlers[p.peek().Type]; ok {
			handler(model)
		} else {
			p.error(fmt.Sprintf("unrecognized keyword %q; expected one of: %s", p.peek().Value, p.expectedKeywords()))
			p.advance()
		}
	}

	return model, p.diagnostics
}

func (p *Instance) expectedKeywords() string {
	names := make([]string, 0, len(p.handlers))
	for kind := range p.handlers {
		names = append(names, kind.String())
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
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
		} else if p.check(lexer.KeywordTrigger) {
			if trigger := p.parseTrigger(); trigger != nil {
				slice.Trigger = trigger
			}
		} else if p.check(lexer.KeywordFlow) {
			slice.Flows = append(slice.Flows, p.parseFlows()...)
		} else if p.check(lexer.KeywordView) {
			if view := p.parseView(); view != nil {
				slice.Views = append(slice.Views, view)
			}
		} else if p.check(lexer.KeywordAutomation) {
			if automation := p.parseAutomation(); automation != nil {
				slice.Automations = append(slice.Automations, automation)
			}
		} else if p.check(lexer.KeywordTranslation) {
			if translation := p.parseTranslation(); translation != nil {
				slice.Translations = append(slice.Translations, translation)
			}
		} else {
			p.error("expected command, event, trigger, view, automation, translation, or flow in slice")
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

func (p *Instance) parseTrigger() *ast.Trigger {
	p.consume(lexer.KeywordTrigger, "expected trigger")
	if !p.check(lexer.Identifier) {
		p.error("expected identifier after trigger")
		return nil
	}

	kindTok := p.advance()
	trigger := &ast.Trigger{
		Kind:    kindTok.Value,
		KindPos: p.position(kindTok),
	}

	if !p.check(lexer.String) {
		p.error("expected quoted string after trigger kind")
		return nil
	}
	nameTok := p.advance()
	trigger.Name = nameTok.Value
	trigger.NamePos = p.position(nameTok)

	if !p.check(lexer.OpenBrace) {
		p.error("expected { after trigger name")
		return nil
	}
	openTok := p.advance()
	trigger.OpenPos = p.position(openTok)

	for !p.check(lexer.CloseBrace) && !p.isAtEnd() {
		if p.check(lexer.KeywordActor) {
			p.advance()
			if !p.check(lexer.Identifier) {
				p.error("expected identifier after actor in trigger")
				p.advance()
				continue
			}
			actorTok := p.advance()
			trigger.Actor = actorTok.Value
			trigger.ActorPos = p.position(actorTok)
		} else if p.check(lexer.KeywordReads) {
			p.advance()
			if !p.check(lexer.Identifier) {
				p.error("expected identifier after reads in trigger")
				p.advance()
				continue
			}
			readsTok := p.advance()
			trigger.Reads = readsTok.Value
			trigger.ReadsPos = p.position(readsTok)
		} else {
			p.error(fmt.Sprintf("expected actor or reads in trigger, got %q", p.peek().Value))
			p.advance()
		}
	}

	if !p.check(lexer.CloseBrace) {
		p.error(fmt.Sprintf("unclosed brace for \"trigger\" block opened at line %d", trigger.OpenPos.Line))
		return trigger
	}
	closeTok := p.advance()
	trigger.ClosePos = p.position(closeTok)

	return trigger
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

func (p *Instance) parseView() *ast.View {
	p.consume(lexer.KeywordView, "expected view")
	if !p.check(lexer.Identifier) {
		p.error("expected identifier after view")
		return nil
	}

	nameTok := p.advance()
	view := &ast.View{
		Name:    nameTok.Value,
		NamePos: p.position(nameTok),
	}

	if !p.check(lexer.OpenBrace) {
		p.error("expected { after view name")
		return nil
	}
	openTok := p.advance()
	view.OpenPos = p.position(openTok)

	for !p.check(lexer.CloseBrace) && !p.isAtEnd() {
		if p.check(lexer.KeywordFields) {
			view.Fields = p.parseFields()
		} else if p.check(lexer.KeywordSubscribes) {
			view.Subscribes = p.parseSubscribes()
		} else {
			p.error("expected fields or subscribes in view")
			p.advance()
		}
	}

	if !p.check(lexer.CloseBrace) {
		p.error(fmt.Sprintf("unclosed brace for \"view\" block opened at line %d", view.OpenPos.Line))
		return view
	}
	closeTok := p.advance()
	view.ClosePos = p.position(closeTok)

	if len(view.Fields) == 0 && len(view.Subscribes) == 0 {
		p.diagnostics = append(p.diagnostics, &diagnostic.Entry{
			Filename: view.NamePos.Filename,
			Line:     view.NamePos.Line,
			Column:   view.NamePos.Column,
			Message:  "view block requires fields or subscribes",
		})
	}

	return view
}

func (p *Instance) parseAutomation() *ast.Automation {
	p.consume(lexer.KeywordAutomation, "expected automation")
	if !p.check(lexer.Identifier) {
		p.error("expected identifier after automation")
		return nil
	}

	nameTok := p.advance()
	automation := &ast.Automation{
		Name:    nameTok.Value,
		NamePos: p.position(nameTok),
	}

	if !p.check(lexer.OpenBrace) {
		p.error("expected { after automation name")
		return nil
	}
	openTok := p.advance()
	automation.OpenPos = p.position(openTok)

	for !p.check(lexer.CloseBrace) && !p.isAtEnd() {
		if p.check(lexer.KeywordTrigger) {
			p.advance()
			if !p.check(lexer.Identifier) {
				p.error("expected identifier after trigger in automation")
				p.advance()
				continue
			}
			triggerTok := p.advance()
			automation.TriggerEvent = triggerTok.Value
			automation.TriggerEventPos = p.position(triggerTok)
		} else if p.check(lexer.KeywordCommand) {
			p.advance()
			if !p.check(lexer.Identifier) {
				p.error("expected identifier after command in automation")
				p.advance()
				continue
			}
			cmdTok := p.advance()
			automation.Command = cmdTok.Value
			automation.CommandPos = p.position(cmdTok)
		} else if p.check(lexer.KeywordTarget) {
			p.advance()
			if !p.check(lexer.KeywordContext) {
				p.error("expected context after target in automation")
				p.advance()
				continue
			}
			p.advance()
			if !p.check(lexer.Identifier) {
				p.error("expected identifier after target context in automation")
				p.advance()
				continue
			}
			ctxTok := p.advance()
			automation.TargetContext = ctxTok.Value
			automation.TargetContextPos = p.position(ctxTok)
		} else {
			p.error(fmt.Sprintf("expected trigger, command, or target in automation, got %q", p.peek().Value))
			p.advance()
		}
	}

	if !p.check(lexer.CloseBrace) {
		p.error(fmt.Sprintf("unclosed brace for \"automation\" block opened at line %d", automation.OpenPos.Line))
		return automation
	}
	closeTok := p.advance()
	automation.ClosePos = p.position(closeTok)

	if automation.TriggerEvent == "" {
		p.diagnostics = append(p.diagnostics, &diagnostic.Entry{
			Filename: automation.NamePos.Filename,
			Line:     automation.NamePos.Line,
			Column:   automation.NamePos.Column,
			Message:  "automation block requires a trigger event",
		})
	}
	if automation.Command == "" {
		p.diagnostics = append(p.diagnostics, &diagnostic.Entry{
			Filename: automation.NamePos.Filename,
			Line:     automation.NamePos.Line,
			Column:   automation.NamePos.Column,
			Message:  "automation block requires a command",
		})
	}

	return automation
}

func (p *Instance) parseTranslation() *ast.Translation {
	p.consume(lexer.KeywordTranslation, "expected translation")
	if !p.check(lexer.Identifier) {
		p.error("expected identifier after translation")
		return nil
	}

	nameTok := p.advance()
	translation := &ast.Translation{
		Name:    nameTok.Value,
		NamePos: p.position(nameTok),
	}

	if !p.check(lexer.OpenBrace) {
		p.error("expected { after translation name")
		return nil
	}
	openTok := p.advance()
	translation.OpenPos = p.position(openTok)

	for !p.check(lexer.CloseBrace) && !p.isAtEnd() {
		if p.check(lexer.KeywordExternalSystem) {
			p.advance()
			if !p.check(lexer.String) {
				p.error("expected quoted string after external_system in translation")
				p.advance()
				continue
			}
			extTok := p.advance()
			translation.ExternalSystem = extTok.Value
			translation.ExternalPos = p.position(extTok)
		} else if p.check(lexer.KeywordReads) {
			p.advance()
			if !p.check(lexer.Identifier) {
				p.error("expected identifier after reads in translation")
				p.advance()
				continue
			}
			readsTok := p.advance()
			translation.Reads = readsTok.Value
			translation.ReadsPos = p.position(readsTok)
		} else if p.check(lexer.KeywordCommand) {
			p.advance()
			if !p.check(lexer.Identifier) {
				p.error("expected identifier after command in translation")
				p.advance()
				continue
			}
			cmdTok := p.advance()
			translation.Command = cmdTok.Value
			translation.CommandPos = p.position(cmdTok)
		} else if p.check(lexer.KeywordEvent) {
			translation.Event = p.parseEvent()
		} else {
			p.error(fmt.Sprintf("expected external_system, reads, command, or event in translation, got %q", p.peek().Value))
			p.advance()
		}
	}

	if !p.check(lexer.CloseBrace) {
		p.error(fmt.Sprintf("unclosed brace for \"translation\" block opened at line %d", translation.OpenPos.Line))
		return translation
	}
	closeTok := p.advance()
	translation.ClosePos = p.position(closeTok)

	if translation.ExternalSystem == "" {
		p.diagnostics = append(p.diagnostics, &diagnostic.Entry{
			Filename: translation.NamePos.Filename,
			Line:     translation.NamePos.Line,
			Column:   translation.NamePos.Column,
			Message:  "translation block requires an external_system",
		})
	}
	if translation.Reads == "" {
		p.diagnostics = append(p.diagnostics, &diagnostic.Entry{
			Filename: translation.NamePos.Filename,
			Line:     translation.NamePos.Line,
			Column:   translation.NamePos.Column,
			Message:  "translation block requires a reads view",
		})
	}
	if translation.Command == "" {
		p.diagnostics = append(p.diagnostics, &diagnostic.Entry{
			Filename: translation.NamePos.Filename,
			Line:     translation.NamePos.Line,
			Column:   translation.NamePos.Column,
			Message:  "translation block requires a command",
		})
	}

	return translation
}

func (p *Instance) parseSubscribes() []string {
	var names []string
	p.consume(lexer.KeywordSubscribes, "expected subscribes")
	if !p.check(lexer.OpenBracket) {
		p.error("expected [ after subscribes")
		return names
	}
	p.advance()

	for !p.check(lexer.CloseBracket) && !p.isAtEnd() {
		if !p.check(lexer.Identifier) {
			p.error("expected identifier in subscribes list")
			p.advance()
			continue
		}
		names = append(names, p.advance().Value)

		if p.check(lexer.Comma) {
			p.advance()
		}
	}

	if !p.check(lexer.CloseBracket) {
		p.error("expected ] to close subscribes list")
		return names
	}
	p.advance()

	return names
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

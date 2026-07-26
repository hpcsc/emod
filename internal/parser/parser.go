package parser

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/diagnostic"
	"github.com/hpcsc/emod/internal/lexer"
)

const (
	impliedVersion         = 1
	expectedVersionInteger = `invalid version header: expected an integer after "emod"`
)

type topLevelHandler func(model *ast.Model)

type Instance struct {
	tokens      []*lexer.Token
	pos         int
	diagnostics []*diagnostic.Entry
	filename    string
	handlers    map[lexer.Kind]topLevelHandler
	pending     []*ast.Comment
}

func New(tokens []*lexer.Token, filename string) *Instance {
	p := &Instance{
		tokens:   tokens,
		pos:      0,
		filename: filename,
	}
	p.handlers = map[lexer.Kind]topLevelHandler{
		lexer.KeywordModel: func(model *ast.Model) {
			comments := p.takePendingComments()
			name, pos := p.parseModel()
			model.Comments = comments
			model.Name = name
			model.NamePos = pos
		},
		lexer.KeywordActor: func(model *ast.Model) {
			comments := p.takePendingComments()
			if actor := p.parseActor(); actor != nil {
				actor.Comments = comments
				model.Actors = append(model.Actors, actor)
			}
		},
		lexer.KeywordContext: func(model *ast.Model) {
			comments := p.takePendingComments()
			if ctx := p.parseContext(); ctx != nil {
				ctx.Comments = comments
				model.Contexts = append(model.Contexts, ctx)
			}
		},
	}
	return p
}

func (p *Instance) Parse() (*ast.Model, []*diagnostic.Entry) {
	header := p.parseVersionHeader()
	model := &ast.Model{Version: header.version, VersionDeclared: header.declared}

	if header.declaresUnsupportedVersion() {
		p.reportUnsupportedVersion(header)
		return model, p.diagnostics
	}

	for !p.isAtEnd() {
		if p.check(lexer.EOF) {
			break
		}

		if p.check(lexer.KeywordEmod) {
			p.reportMisplacedVersionHeader()
			continue
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

type versionHeader struct {
	version  int
	declared bool
	keyword  *lexer.Token
}

func (h versionHeader) declaresUnsupportedVersion() bool {
	return h.declared && h.version != ast.SupportedVersion
}

func (p *Instance) parseVersionHeader() versionHeader {
	implied := versionHeader{version: impliedVersion}

	if !p.check(lexer.KeywordEmod) {
		return implied
	}

	keywordTok := p.advance()
	if !p.checkSameLineAs(keywordTok) {
		p.errorAt(keywordTok, expectedVersionInteger)
		return implied
	}

	if !p.check(lexer.Integer) {
		offending := p.peek()
		p.errorAt(keywordTok, fmt.Sprintf("%s, got %q", expectedVersionInteger, offending.Value))
		if _, startsTopLevelDeclaration := p.handlers[offending.Type]; !startsTopLevelDeclaration {
			p.advance()
		}
		return implied
	}

	versionTok := p.advance()
	version, err := strconv.Atoi(versionTok.Value)
	if err != nil {
		p.errorAt(versionTok, fmt.Sprintf("invalid version header: version %q is out of range", versionTok.Value))
		return implied
	}

	return versionHeader{version: version, declared: true, keyword: keywordTok}
}

func (p *Instance) reportUnsupportedVersion(header versionHeader) {
	p.errorAt(header.keyword, fmt.Sprintf("unsupported version %d: this tool supports emod version %d", header.version, ast.SupportedVersion))
}

func (p *Instance) reportMisplacedVersionHeader() {
	keywordTok := p.advance()
	p.errorAt(keywordTok, `misplaced version header: "emod" must appear before the "model" declaration`)
	if p.checkSameLineAs(keywordTok) && p.check(lexer.Integer) {
		p.advance()
	}
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

	// Optional mode clause: mode dcb | mode aggregate | mode mixed
	if p.check(lexer.KeywordMode) {
		p.advance()
		if p.checkIdentifierLike() {
			modeTok := p.advance()
			context.Mode = modeTok.Value
			context.ModePos = p.position(modeTok)
		} else {
			p.error("expected mode value after 'mode'")
		}
	}

	if !p.check(lexer.OpenBrace) {
		p.error("expected { after context name")
		return nil
	}
	openTok := p.advance()
	context.OpenPos = p.position(openTok)

	for !p.check(lexer.CloseBrace) && !p.isAtEnd() {
		switch {
		case p.check(lexer.KeywordDescription):
			p.parseDescriptionInto("context", &context.Description, &context.DescriptionPos)
		case p.check(lexer.KeywordAggregate):
			if agg := p.parseAggregate(); agg != nil {
				context.Aggregates = append(context.Aggregates, agg)
			}
		case p.check(lexer.KeywordSlice):
			if slice := p.parseSlice(); slice != nil {
				context.Slices = append(context.Slices, slice)
			}
		default:
			p.error(fmt.Sprintf("expected description, aggregate or slice in context, got %q", p.peek().Value))
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
	comments := p.takePendingComments()
	p.consume(lexer.KeywordAggregate, "expected aggregate")
	if !p.check(lexer.String) {
		p.error(fmt.Sprintf("expected quoted string after \"aggregate\", got %q", p.peek().Value))
		return nil
	}

	nameTok := p.advance()
	aggregate := &ast.Aggregate{
		Comments: comments,
		Name:     nameTok.Value,
		NamePos:  p.position(nameTok),
	}

	if !p.check(lexer.OpenBrace) {
		p.error("expected { after aggregate name")
		return nil
	}
	openTok := p.advance()
	aggregate.OpenPos = p.position(openTok)

	for !p.check(lexer.CloseBrace) && !p.isAtEnd() {
		if p.check(lexer.KeywordDescription) {
			p.parseDescriptionInto("aggregate", &aggregate.Description, &aggregate.DescriptionPos)
		} else if p.check(lexer.KeywordSlice) {
			if slice := p.parseSlice(); slice != nil {
				aggregate.Slices = append(aggregate.Slices, slice)
			}
		} else {
			p.error("expected description or slice in aggregate")
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
	comments := p.takePendingComments()
	p.consume(lexer.KeywordSlice, "expected slice")
	if !p.check(lexer.String) {
		p.error(fmt.Sprintf("expected quoted string after \"slice\", got %q", p.peek().Value))
		return nil
	}

	nameTok := p.advance()
	slice := &ast.Slice{
		Comments: comments,
		Name:     nameTok.Value,
		NamePos:  p.position(nameTok),
	}

	if !p.check(lexer.OpenBrace) {
		p.error("expected { after slice name")
		return nil
	}
	openTok := p.advance()
	slice.OpenPos = p.position(openTok)

	for !p.check(lexer.CloseBrace) && !p.isAtEnd() {
		if p.check(lexer.KeywordDescription) {
			p.parseDescriptionInto("slice", &slice.Description, &slice.DescriptionPos)
		} else if p.check(lexer.KeywordCommand) {
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
			p.error("expected description, command, event, trigger, view, automation, translation, or flow in slice")
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
	comments := p.takePendingComments()
	p.consume(lexer.KeywordTrigger, "expected trigger")
	if !p.check(lexer.Identifier) {
		p.error("expected identifier after trigger")
		return nil
	}

	kindTok := p.advance()
	trigger := &ast.Trigger{
		Comments: comments,
		Kind:     kindTok.Value,
		KindPos:  p.position(kindTok),
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
		if p.check(lexer.KeywordDescription) {
			p.parseDescriptionInto("trigger", &trigger.Description, &trigger.DescriptionPos)
		} else if p.check(lexer.KeywordActor) {
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
			p.error(fmt.Sprintf("expected description, actor or reads in trigger, got %q", p.peek().Value))
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
	comments := p.takePendingComments()
	p.consume(lexer.KeywordCommand, "expected command")
	if !p.check(lexer.Identifier) {
		p.error("expected identifier after command")
		return nil
	}

	nameTok := p.advance()
	command := &ast.Command{
		Comments: comments,
		Name:     nameTok.Value,
		NamePos:  p.position(nameTok),
	}

	if !p.check(lexer.OpenBrace) {
		p.error("expected { after command name")
		return nil
	}
	openTok := p.advance()
	command.OpenPos = p.position(openTok)

	for !p.check(lexer.CloseBrace) && !p.isAtEnd() {
		if p.check(lexer.KeywordDescription) {
			p.parseDescriptionInto("command", &command.Description, &command.DescriptionPos)
		} else if p.check(lexer.KeywordFields) {
			command.Fields = p.parseFields()
		} else if p.check(lexer.KeywordDecidesOn) {
			command.DecidesOn = p.parseDecidesOn()
		} else {
			p.error("expected description, fields or decides_on in command")
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

func (p *Instance) parseDecidesOn() *ast.DecidesOnClause {
	comments := p.takePendingComments()
	p.consume(lexer.KeywordDecidesOn, "expected decides_on")

	if !p.check(lexer.OpenBrace) {
		p.error("expected { after decides_on")
		return nil
	}
	openTok := p.advance()

	clause := &ast.DecidesOnClause{
		Comments: comments,
		OpenPos:  p.position(openTok),
	}

	for !p.check(lexer.CloseBrace) && !p.isAtEnd() {
		if p.check(lexer.KeywordEvents) {
			clause.Events, clause.EventsPos = p.parseDecidesOnEvents()
		} else if p.check(lexer.KeywordWhere) {
			clause.Predicate = p.parsePredicate()
		} else {
			p.error("expected events or where in decides_on")
			p.advance()
		}
	}

	if !p.check(lexer.CloseBrace) {
		p.error(fmt.Sprintf("unclosed brace for \"decides_on\" block opened at line %d", clause.OpenPos.Line))
		return clause
	}
	closeTok := p.advance()
	clause.ClosePos = p.position(closeTok)

	if len(clause.Events) == 0 {
		p.diagnostics = append(p.diagnostics, &diagnostic.Entry{
			Filename: clause.OpenPos.Filename,
			Line:     clause.OpenPos.Line,
			Column:   clause.OpenPos.Column,
			Message:  "decides_on block requires an events clause",
		})
	}
	if clause.Predicate == nil {
		p.diagnostics = append(p.diagnostics, &diagnostic.Entry{
			Filename: clause.OpenPos.Filename,
			Line:     clause.OpenPos.Line,
			Column:   clause.OpenPos.Column,
			Message:  "decides_on block requires a where clause",
		})
	}

	return clause
}

func (p *Instance) parseDecidesOnEvents() ([]string, []ast.Position) {
	var names []string
	var positions []ast.Position
	p.consume(lexer.KeywordEvents, "expected events")
	if !p.check(lexer.OpenBracket) {
		p.error("expected [ after events")
		return names, positions
	}
	p.advance()

	for !p.check(lexer.CloseBracket) && !p.isAtEnd() {
		if !p.check(lexer.Identifier) {
			p.error("expected identifier in events list")
			p.advance()
			continue
		}
		tok := p.advance()
		names = append(names, tok.Value)
		positions = append(positions, p.position(tok))

		if p.check(lexer.Comma) {
			p.advance()
		}
	}

	if !p.check(lexer.CloseBracket) {
		p.error("expected ] to close events list")
		return names, positions
	}
	p.advance()

	return names, positions
}

func (p *Instance) parsePredicate() ast.PredicateExpr {
	p.consume(lexer.KeywordWhere, "expected where")
	return p.parseOrExpr()
}

func (p *Instance) parseOrExpr() ast.PredicateExpr {
	left := p.parseAndExpr()
	if left == nil {
		return nil
	}
	for p.check(lexer.KeywordOr) {
		opTok := p.advance()
		right := p.parseAndExpr()
		if right == nil {
			p.error("expected expression after 'or'")
			break
		}
		left = &ast.LogicalExpr{
			Left:     left,
			Operator: opTok.Value,
			OpPos:    p.position(opTok),
			Right:    right,
		}
	}
	return left
}

func (p *Instance) parseAndExpr() ast.PredicateExpr {
	left := p.parseNotExpr()
	if left == nil {
		return nil
	}
	for p.check(lexer.KeywordAnd) {
		opTok := p.advance()
		right := p.parseNotExpr()
		if right == nil {
			p.error("expected expression after 'and'")
			break
		}
		left = &ast.LogicalExpr{
			Left:     left,
			Operator: opTok.Value,
			OpPos:    p.position(opTok),
			Right:    right,
		}
	}
	return left
}

func (p *Instance) parseNotExpr() ast.PredicateExpr {
	if p.check(lexer.KeywordNot) {
		opTok := p.advance()
		expr := p.parseNotExpr()
		if expr == nil {
			p.error("expected expression after 'not'")
			return nil
		}
		return &ast.NotExpr{
			OpPos: p.position(opTok),
			Expr:  expr,
		}
	}
	return p.parsePrimary()
}

func (p *Instance) parsePrimary() ast.PredicateExpr {
	if p.check(lexer.CloseBrace) || p.check(lexer.EOF) {
		return nil
	}

	if p.check(lexer.OpenParen) {
		p.advance()
		expr := p.parseOrExpr()
		if !p.check(lexer.CloseParen) {
			p.error("expected ) after predicate sub-expression")
			return expr
		}
		p.advance()
		return expr
	}

	if p.check(lexer.KeywordTag) {
		return p.parseTagPredicate()
	}

	p.error("expected tag() or ( in predicate")
	return nil
}

func (p *Instance) parseTagPredicate() ast.PredicateExpr {
	p.consume(lexer.KeywordTag, "expected tag")
	if !p.check(lexer.OpenParen) {
		p.error("expected ( after tag")
		return nil
	}
	p.advance()

	if !p.checkIdentifierLike() {
		p.error("expected tag key in tag()")
		p.skipTo(lexer.CloseParen, lexer.CloseBrace)
		if p.check(lexer.CloseParen) {
			p.advance()
		}
		return nil
	}
	keyTok := p.advance()

	if !p.check(lexer.Equals) {
		p.error("expected = after tag key")
		p.skipTo(lexer.CloseParen, lexer.CloseBrace)
		if p.check(lexer.CloseParen) {
			p.advance()
		}
		return nil
	}
	opTok := p.advance()

	if !p.checkIdentifierLike() && !p.check(lexer.String) {
		p.error("expected value in tag()")
		p.skipTo(lexer.CloseParen, lexer.CloseBrace)
		if p.check(lexer.CloseParen) {
			p.advance()
		}
		return nil
	}
	valTok := p.advance()

	if !p.check(lexer.CloseParen) {
		p.error("expected ) after tag() arguments")
		return &ast.TagPredicate{
			Field:    keyTok.Value,
			FieldPos: p.position(keyTok),
			Operator: opTok.Value,
			OpPos:    p.position(opTok),
			Value:    valTok.Value,
			ValuePos: p.position(valTok),
		}
	}
	p.advance()

	return &ast.TagPredicate{
		Field:    keyTok.Value,
		FieldPos: p.position(keyTok),
		Operator: opTok.Value,
		OpPos:    p.position(opTok),
		Value:    valTok.Value,
		ValuePos: p.position(valTok),
	}
}

func (p *Instance) parseEvent() *ast.Event {
	comments := p.takePendingComments()
	p.consume(lexer.KeywordEvent, "expected event")
	if !p.check(lexer.Identifier) {
		p.error("expected identifier after event")
		return nil
	}

	nameTok := p.advance()
	event := &ast.Event{
		Comments: comments,
		Name:     nameTok.Value,
		NamePos:  p.position(nameTok),
	}

	if !p.check(lexer.OpenBrace) {
		p.error("expected { after event name")
		return nil
	}
	openTok := p.advance()
	event.OpenPos = p.position(openTok)

	for !p.check(lexer.CloseBrace) && !p.isAtEnd() {
		if p.check(lexer.KeywordDescription) {
			p.parseDescriptionInto("event", &event.Description, &event.DescriptionPos)
		} else if p.check(lexer.KeywordFields) {
			event.Fields = p.parseFields()
		} else if p.check(lexer.KeywordSource) {
			sourceTok := p.advance()
			event.SourcePos = p.position(sourceTok)
			if !p.check(lexer.KeywordExternal) {
				p.error("expected external after source in event")
				p.advance()
				continue
			}
			p.advance()
			event.Source = "external"
			if !p.check(lexer.String) {
				p.error("expected quoted string after source external in event")
				if !p.check(lexer.CloseBrace) && !p.check(lexer.KeywordFields) && !p.check(lexer.KeywordSource) && !p.check(lexer.KeywordTags) {
					p.advance()
				}
				continue
			}
			nameTok := p.advance()
			event.ExternalName = nameTok.Value
			event.ExternalNamePos = p.position(nameTok)
		} else if p.check(lexer.KeywordTags) {
			event.Tags = p.parseTags()
		} else {
			p.error("expected description, fields, source, or tags in event")
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
	comments := p.takePendingComments()
	p.consume(lexer.KeywordView, "expected view")
	if !p.check(lexer.Identifier) {
		p.error("expected identifier after view")
		return nil
	}

	nameTok := p.advance()
	view := &ast.View{
		Comments: comments,
		Name:     nameTok.Value,
		NamePos:  p.position(nameTok),
	}

	if !p.check(lexer.OpenBrace) {
		p.error("expected { after view name")
		return nil
	}
	openTok := p.advance()
	view.OpenPos = p.position(openTok)

	for !p.check(lexer.CloseBrace) && !p.isAtEnd() {
		if p.check(lexer.KeywordDescription) {
			p.parseDescriptionInto("view", &view.Description, &view.DescriptionPos)
		} else if p.check(lexer.KeywordFields) {
			view.Fields = p.parseFields()
		} else if p.check(lexer.KeywordSubscribes) {
			view.Subscribes, view.SubscribesPos = p.parseSubscribes()
		} else {
			p.error("expected description, fields or subscribes in view")
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
	comments := p.takePendingComments()
	p.consume(lexer.KeywordAutomation, "expected automation")
	if !p.check(lexer.Identifier) {
		p.error("expected identifier after automation")
		return nil
	}

	nameTok := p.advance()
	automation := &ast.Automation{
		Comments: comments,
		Name:     nameTok.Value,
		NamePos:  p.position(nameTok),
	}

	if !p.check(lexer.OpenBrace) {
		p.error("expected { after automation name")
		return nil
	}
	openTok := p.advance()
	automation.OpenPos = p.position(openTok)

	for !p.check(lexer.CloseBrace) && !p.isAtEnd() {
		if p.check(lexer.KeywordDescription) {
			p.parseDescriptionInto("automation", &automation.Description, &automation.DescriptionPos)
		} else if p.check(lexer.KeywordTrigger) {
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
			p.error(fmt.Sprintf("expected description, trigger, command, or target in automation, got %q", p.peek().Value))
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
	comments := p.takePendingComments()
	p.consume(lexer.KeywordTranslation, "expected translation")
	if !p.check(lexer.Identifier) {
		p.error("expected identifier after translation")
		return nil
	}

	nameTok := p.advance()
	translation := &ast.Translation{
		Comments: comments,
		Name:     nameTok.Value,
		NamePos:  p.position(nameTok),
	}

	if !p.check(lexer.OpenBrace) {
		p.error("expected { after translation name")
		return nil
	}
	openTok := p.advance()
	translation.OpenPos = p.position(openTok)

	for !p.check(lexer.CloseBrace) && !p.isAtEnd() {
		if p.check(lexer.KeywordDescription) {
			p.parseDescriptionInto("translation", &translation.Description, &translation.DescriptionPos)
		} else if p.check(lexer.KeywordExternalSystem) {
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
			p.error(fmt.Sprintf("expected description, external_system, reads, command, or event in translation, got %q", p.peek().Value))
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

func (p *Instance) parseSubscribes() ([]string, []ast.Position) {
	var names []string
	var positions []ast.Position
	p.consume(lexer.KeywordSubscribes, "expected subscribes")
	if !p.check(lexer.OpenBracket) {
		p.error("expected [ after subscribes")
		return names, positions
	}
	p.advance()

	for !p.check(lexer.CloseBracket) && !p.isAtEnd() {
		if !p.check(lexer.Identifier) {
			p.error("expected identifier in subscribes list")
			p.advance()
			continue
		}
		tok := p.advance()
		names = append(names, tok.Value)
		positions = append(positions, p.position(tok))

		if p.check(lexer.Comma) {
			p.advance()
		}
	}

	if !p.check(lexer.CloseBracket) {
		p.error("expected ] to close subscribes list")
		return names, positions
	}
	p.advance()

	return names, positions
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
		if p.checkIdentifierLike() {
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
	if !p.checkIdentifierLike() {
		return nil
	}

	nameTok := p.advance()
	field := &ast.Field{
		Name:    nameTok.Value,
		NamePos: p.position(nameTok),
	}

	if !p.checkIdentifierLike() {
		p.error("expected field type")
		return field
	}
	typeTok := p.advance()
	field.Type = typeTok.Value
	field.TypePos = p.position(typeTok)

	if p.checkIdentifierLike() {
		modTok := p.advance()
		field.Modifier = modTok.Value
		field.ModPos = p.position(modTok)
	}

	return field
}

func (p *Instance) parseTags() []ast.TagEntry {
	var tags []ast.TagEntry
	p.consume(lexer.KeywordTags, "expected tags")
	if !p.check(lexer.OpenBrace) {
		p.error("expected { after tags")
		return tags
	}
	openTok := p.advance()
	openLine := openTok.Line

	for !p.check(lexer.CloseBrace) && !p.isAtEnd() {
		if p.checkIdentifierLike() {
			if tag := p.parseTagEntry(); tag != nil {
				tags = append(tags, *tag)
			}
		} else {
			p.error("expected tag entry")
			p.advance()
		}
	}

	if !p.check(lexer.CloseBrace) {
		p.error(fmt.Sprintf("unclosed brace for \"tags\" block opened at line %d", openLine))
		return tags
	}
	p.advance()

	return tags
}

func (p *Instance) parseTagEntry() *ast.TagEntry {
	if !p.checkIdentifierLike() {
		return nil
	}

	keyTok := p.advance()

	if !p.check(lexer.Colon) {
		p.error("expected : after tag key")
		return nil
	}
	p.advance()

	if !p.checkIdentifierLike() {
		p.error("expected field reference after : in tag")
		p.advance()
		return nil
	}
	fieldRefTok := p.advance()

	return &ast.TagEntry{
		Key:         keyTok.Value,
		KeyPos:      p.position(keyTok),
		FieldRef:    fieldRefTok.Value,
		FieldRefPos: p.position(fieldRefTok),
	}
}

func (p *Instance) parseFlows() []*ast.Flow {
	comments := p.takePendingComments()
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

	if len(flows) > 0 {
		flows[0].Comments = comments
	} else {
		p.pending = comments
	}

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

func (p *Instance) parseDescriptionInto(construct string, description *string, position *ast.Position) {
	keywordTok := p.advance()
	if !p.check(lexer.String) {
		offending := p.peek()
		p.errorAt(offending, fmt.Sprintf("expected quoted string after description in %s, got %q", construct, offending.Value))
		for p.checkSameLineAs(keywordTok) && !p.check(lexer.CloseBrace) {
			p.advance()
		}
		return
	}

	tok := p.advance()
	*description, *position = tok.Value, p.position(tok)
}

func (p *Instance) position(tok *lexer.Token) ast.Position {
	return ast.Position{
		Filename: p.filename,
		Line:     tok.Line,
		Column:   tok.Column,
	}
}

func (p *Instance) peek() *lexer.Token {
	p.skipComments()
	if p.pos >= len(p.tokens) {
		return &lexer.Token{Type: lexer.EOF}
	}
	return p.tokens[p.pos]
}

func (p *Instance) skipComments() {
	for p.pos < len(p.tokens) && p.tokens[p.pos].Type == lexer.Comment {
		tok := p.tokens[p.pos]
		p.pending = append(p.pending, &ast.Comment{
			Text:     tok.Value,
			Position: ast.Position{Filename: p.filename, Line: tok.Line, Column: tok.Column},
		})
		p.pos++
	}
}

func (p *Instance) takePendingComments() []*ast.Comment {
	p.skipComments()
	comments := p.pending
	p.pending = nil
	return comments
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

func (p *Instance) checkSameLineAs(tok *lexer.Token) bool {
	return !p.isAtEnd() && p.peek().Line == tok.Line
}

// checkIdentifierLike returns true when the current token is an Identifier or
// any keyword. Keywords are valid as field names inside fields blocks.
func (p *Instance) checkIdentifierLike() bool {
	if p.isAtEnd() {
		return false
	}
	typ := p.tokens[p.pos].Type
	return typ == lexer.Identifier || typ < lexer.Identifier
}

func (p *Instance) consume(typ lexer.Kind, msg string) {
	if !p.check(typ) {
		p.error(msg)
		return
	}
	p.advance()
}

// skipTo advances tokens until one of the given types is found, or the end
// of input is reached. Used for error recovery.
func (p *Instance) skipTo(types ...lexer.Kind) {
	for !p.isAtEnd() {
		for _, typ := range types {
			if p.check(typ) {
				return
			}
		}
		p.advance()
	}
}

func (p *Instance) isAtEnd() bool {
	p.skipComments()
	return p.pos >= len(p.tokens) || p.tokens[p.pos].Type == lexer.EOF
}

func (p *Instance) error(msg string) {
	p.errorAt(p.peek(), msg)
}

func (p *Instance) errorAt(tok *lexer.Token, msg string) {
	p.diagnostics = append(p.diagnostics, &diagnostic.Entry{
		Filename: p.filename,
		Line:     tok.Line,
		Column:   tok.Column,
		Message:  msg,
	})
}

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
		lexer.KeywordModel:   p.parseModelInto,
		lexer.KeywordActor:   p.parseActorInto,
		lexer.KeywordContext: p.parseContextInto,
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

func (p *Instance) parseModelInto(model *ast.Model) {
	model.Comments = p.takePendingComments()
	decl := p.parseDeclaration(lexer.KeywordModel)
	if decl == nil {
		return
	}

	model.Name, model.NamePos = decl.name, decl.namePos
	model.Description, model.DescriptionPos = decl.description, decl.descriptionPos
	model.OpenPos, model.ClosePos = decl.openPos, decl.closePos
}

func (p *Instance) parseActorInto(model *ast.Model) {
	comments := p.takePendingComments()
	decl := p.parseDeclaration(lexer.KeywordActor)
	if decl == nil {
		return
	}

	model.Actors = append(model.Actors, &ast.Actor{
		Comments:       comments,
		Name:           decl.name,
		NamePos:        decl.namePos,
		Description:    decl.description,
		DescriptionPos: decl.descriptionPos,
		OpenPos:        decl.openPos,
		ClosePos:       decl.closePos,
	})
}

func (p *Instance) parseContextInto(model *ast.Model) {
	comments := p.takePendingComments()
	if context := p.parseContext(); context != nil {
		context.Comments = comments
		model.Contexts = append(model.Contexts, context)
	}
}

type declaration struct {
	name           string
	namePos        ast.Position
	description    string
	descriptionPos ast.Position
	openPos        ast.Position
	closePos       ast.Position
}

func (p *Instance) parseDeclaration(keyword lexer.Kind) *declaration {
	construct := keyword.String()
	p.consume(keyword, "expected "+construct)
	if !p.check(lexer.String) {
		p.error(fmt.Sprintf("expected quoted string after %q, got %q", construct, p.peek().Value))
		return nil
	}

	nameTok := p.advance()
	decl := &declaration{name: nameTok.Value, namePos: p.position(nameTok)}

	if !p.check(lexer.OpenBrace) {
		return decl
	}
	openTok := p.advance()
	decl.openPos = p.position(openTok)

	for !p.check(lexer.CloseBrace) && !p.isAtEnd() {
		if p.check(lexer.KeywordDescription) {
			p.parseQuotedEntryInto(construct, &decl.description, &decl.descriptionPos)
		} else {
			p.error(fmt.Sprintf("expected description in %s, got %q", construct, p.peek().Value))
			p.advance()
		}
	}

	if !p.check(lexer.CloseBrace) {
		p.error(fmt.Sprintf("unclosed brace for %q block opened at line %d", construct, decl.openPos.Line))
		return decl
	}
	closeTok := p.advance()
	decl.closePos = p.position(closeTok)

	return decl
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
			p.parseQuotedEntryInto("context", &context.Description, &context.DescriptionPos)
		case p.check(lexer.KeywordInvariant):
			if invariant := p.parseInvariant(); invariant != nil {
				context.Invariants = append(context.Invariants, invariant)
			}
		case p.check(lexer.KeywordAggregate):
			if agg := p.parseAggregate(); agg != nil {
				context.Aggregates = append(context.Aggregates, agg)
			}
		case p.check(lexer.KeywordSlice):
			if slice := p.parseSlice(); slice != nil {
				context.Slices = append(context.Slices, slice)
			}
		default:
			p.error(fmt.Sprintf("expected description, invariant, aggregate or slice in context, got %q", p.peek().Value))
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
			p.parseQuotedEntryInto("aggregate", &aggregate.Description, &aggregate.DescriptionPos)
		} else if p.check(lexer.KeywordInvariant) {
			if invariant := p.parseInvariant(); invariant != nil {
				aggregate.Invariants = append(aggregate.Invariants, invariant)
			}
		} else if p.check(lexer.KeywordSlice) {
			if slice := p.parseSlice(); slice != nil {
				aggregate.Slices = append(aggregate.Slices, slice)
			}
		} else {
			p.error("expected description, invariant or slice in aggregate")
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

func (p *Instance) parseInvariant() *ast.Invariant {
	comments := p.takePendingComments()
	keywordTok := p.advance()

	if !p.checkIdentifierLikeSameLineAs(keywordTok) {
		p.errorAt(keywordTok, "expected identifier after invariant")
		p.skipRestOfLineOrBlockEnd(keywordTok)
		return nil
	}
	nameTok := p.advance()

	if !p.checkSameLineAs(keywordTok) || !p.check(lexer.String) {
		p.errorAt(keywordTok, "expected quoted statement after invariant name")
		p.skipRestOfLineOrBlockEnd(keywordTok)
		return nil
	}
	statementTok := p.advance()

	return &ast.Invariant{
		Comments:     comments,
		Name:         nameTok.Value,
		NamePos:      p.position(nameTok),
		Statement:    statementTok.Value,
		StatementPos: p.position(statementTok),
	}
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
			p.parseQuotedEntryInto("slice", &slice.Description, &slice.DescriptionPos)
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
		} else if p.check(lexer.KeywordSpec) {
			if spec := p.parseSpec(); spec != nil {
				slice.Specs = append(slice.Specs, spec)
			}
		} else {
			p.error("expected description, command, event, trigger, view, automation, translation, spec, or flow in slice")
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

func (p *Instance) parseSpec() *ast.Spec {
	comments := p.takePendingComments()
	p.consume(lexer.KeywordSpec, "expected spec")
	if !p.check(lexer.String) {
		p.error(fmt.Sprintf("expected quoted string after \"spec\", got %q", p.peek().Value))
		return nil
	}

	nameTok := p.advance()
	spec := &ast.Spec{
		Comments: comments,
		Name:     nameTok.Value,
		NamePos:  p.position(nameTok),
	}

	if !p.check(lexer.OpenBrace) {
		p.error("expected { after spec name")
		return nil
	}
	openTok := p.advance()
	spec.OpenPos = p.position(openTok)

	for !p.check(lexer.CloseBrace) && !p.isAtEnd() {
		switch entryTok := p.peek(); entryTok.Type {
		case lexer.KeywordGiven:
			p.advance()
			if history, ok := p.parseSpecEventList(entryTok); ok {
				spec.Given = history
			}
		case lexer.KeywordWhen:
			p.advance()
			if command := p.parseSpecCommand(entryTok); command != nil {
				spec.When = command
			}
		case lexer.KeywordThen:
			p.advance()
			if outcome := p.parseSpecOutcome(entryTok); outcome != nil {
				spec.Then = outcome
			}
		default:
			p.error("expected given, when or then in spec")
			p.advance()
		}
	}

	if !p.check(lexer.CloseBrace) {
		p.error(fmt.Sprintf("unclosed brace for \"spec\" block opened at line %d", spec.OpenPos.Line))
		return spec
	}
	closeTok := p.advance()
	spec.ClosePos = p.position(closeTok)

	return spec
}

func (p *Instance) parseSpecCommand(keywordTok *lexer.Token) *ast.SpecElement {
	if !p.check(lexer.Identifier) {
		p.errorAt(keywordTok, "expected command identifier after when in spec")
		p.skipRestOfLineOrBlockEnd(keywordTok)
		return nil
	}

	nameTok := p.advance()

	return &ast.SpecElement{Name: nameTok.Value, NamePos: p.position(nameTok)}
}

func (p *Instance) parseSpecOutcome(keywordTok *lexer.Token) ast.ThenClause {
	switch {
	case p.check(lexer.OpenBracket):
		events, ok := p.parseSpecEventList(keywordTok)
		if !ok {
			return nil
		}
		return &ast.ThenEvents{Events: events}
	case p.check(lexer.KeywordRejected):
		rejectedTok := p.advance()
		if !p.checkIdentifierLikeSameLineAs(rejectedTok) {
			p.errorAt(keywordTok, "expected invariant name after rejected in spec")
			p.skipRestOfLineOrBlockEnd(rejectedTok)
			return nil
		}
		nameTok := p.advance()
		return &ast.ThenRejected{InvariantName: nameTok.Value, InvariantPos: p.position(nameTok)}
	case p.check(lexer.KeywordView):
		viewTok := p.advance()
		if !p.checkSameLineAs(viewTok) || !p.check(lexer.Identifier) {
			p.errorAt(keywordTok, "expected view name after view in spec")
			p.skipRestOfLineOrBlockEnd(viewTok)
			return nil
		}
		nameTok := p.advance()
		return &ast.ThenView{ViewName: nameTok.Value, ViewPos: p.position(nameTok)}
	case p.check(lexer.KeywordCommand):
		commandTok := p.advance()
		if !p.checkSameLineAs(commandTok) || !p.check(lexer.Identifier) {
			p.errorAt(keywordTok, "expected command name after command in spec")
			p.skipRestOfLineOrBlockEnd(commandTok)
			return nil
		}
		nameTok := p.advance()
		return &ast.ThenCommand{CommandName: nameTok.Value, CommandPos: p.position(nameTok)}
	default:
		p.errorAt(keywordTok, "expected an event list, rejected, view or command after then in spec")
		p.skipRestOfLineOrBlockEnd(keywordTok)
		return nil
	}
}

func (p *Instance) parseSpecEventList(keywordTok *lexer.Token) ([]*ast.SpecElement, bool) {
	entry := keywordTok.Value
	if !p.check(lexer.OpenBracket) {
		p.errorAt(keywordTok, fmt.Sprintf("expected [ after %s in spec", entry))
		p.skipRestOfLineOrBlockEnd(keywordTok)
		return nil, false
	}
	p.advance()

	identifiers := p.parseIdentifiersUntil(p.atSpecEventListEnd, fmt.Sprintf("expected event identifier in %s list of spec", entry))

	if !p.check(lexer.CloseBracket) {
		p.errorAt(keywordTok, fmt.Sprintf("expected ] to close %s list of spec", entry))
		return nil, false
	}
	p.advance()

	var events []*ast.SpecElement
	for _, tok := range identifiers {
		events = append(events, &ast.SpecElement{Name: tok.Value, NamePos: p.position(tok)})
	}

	return events, true
}

func (p *Instance) atSpecEventListEnd() bool {
	return p.isAtEnd() || p.checkAny(lexer.CloseBracket, lexer.CloseBrace, lexer.KeywordGiven, lexer.KeywordWhen, lexer.KeywordThen)
}

func (p *Instance) parseTrigger() *ast.Trigger {
	comments := p.takePendingComments()
	p.consume(lexer.KeywordTrigger, "expected trigger")
	if !p.check(lexer.Identifier) && !p.check(lexer.String) {
		p.errorAt(p.peek(), "expected quoted name after trigger")
		if p.check(lexer.OpenBrace) {
			p.advance()
			p.skipTo(lexer.CloseBrace)
			if p.check(lexer.CloseBrace) {
				p.advance()
			}
		}
		return nil
	}

	trigger := &ast.Trigger{
		Comments: comments,
	}

	if p.check(lexer.Identifier) {
		kindTok := p.advance()
		p.errorAt(kindTok, retiredTriggerKindMessage(kindTok.Value))
	}

	if !p.check(lexer.String) {
		p.errorAt(p.peek(), "expected quoted name after trigger")
		if p.check(lexer.OpenBrace) {
			p.advance()
			p.skipTo(lexer.CloseBrace)
			if p.check(lexer.CloseBrace) {
				p.advance()
			}
		}
		return trigger
	}
	nameTok := p.advance()
	trigger.Name = nameTok.Value
	trigger.NamePos = p.position(nameTok)

	if !p.check(lexer.OpenBrace) {
		p.error("expected { after trigger name")
		return trigger
	}
	openTok := p.advance()
	trigger.OpenPos = p.position(openTok)

	for !p.check(lexer.CloseBrace) && !p.isAtEnd() {
		if p.check(lexer.KeywordDescription) {
			p.parseQuotedEntryInto("trigger", &trigger.Description, &trigger.DescriptionPos)
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

func retiredTriggerKindMessage(kind string) string {
	if kind == "Schedule" || kind == "Processor" {
		return fmt.Sprintf("trigger %s is no longer supported: use an automation with every", kind)
	}
	return fmt.Sprintf("trigger %s is no longer supported: drop the word %s", kind, kind)
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
			p.parseQuotedEntryInto("command", &command.Description, &command.DescriptionPos)
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
			clause.Events, clause.EventsPos = p.parseIdentifierList(lexer.KeywordEvents)
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
		p.errorAtPosition(clause.OpenPos, "decides_on block requires an events clause")
	}
	if clause.Predicate == nil {
		p.errorAtPosition(clause.OpenPos, "decides_on block requires a where clause")
	}

	return clause
}

func (p *Instance) parseIdentifierList(keyword lexer.Kind) ([]string, []ast.Position) {
	entry := keyword.String()
	p.consume(keyword, "expected "+entry)
	if !p.check(lexer.OpenBracket) {
		p.error("expected [ after " + entry)
		return nil, nil
	}
	p.advance()

	identifiers := p.parseIdentifiersUntil(
		func() bool { return p.check(lexer.CloseBracket) || p.isAtEnd() },
		"expected identifier in "+entry+" list",
	)

	var names []string
	var positions []ast.Position
	for _, tok := range identifiers {
		names = append(names, tok.Value)
		positions = append(positions, p.position(tok))
	}

	if !p.check(lexer.CloseBracket) {
		p.error("expected ] to close " + entry + " list")
		return names, positions
	}
	p.advance()

	return names, positions
}

func (p *Instance) parseIdentifiersUntil(atEnd func() bool, invalidItemMsg string) []*lexer.Token {
	var identifiers []*lexer.Token
	for !atEnd() {
		if !p.check(lexer.Identifier) {
			p.error(invalidItemMsg)
			p.advance()
			continue
		}
		identifiers = append(identifiers, p.advance())

		if p.check(lexer.Comma) {
			p.advance()
		}
	}

	return identifiers
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
			p.parseQuotedEntryInto("event", &event.Description, &event.DescriptionPos)
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
			p.parseQuotedEntryInto("view", &view.Description, &view.DescriptionPos)
		} else if p.check(lexer.KeywordFields) {
			view.Fields = p.parseFields()
		} else if p.check(lexer.KeywordSubscribes) {
			view.Subscribes, view.SubscribesPos = p.parseIdentifierList(lexer.KeywordSubscribes)
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
		p.errorAtPosition(view.NamePos, "view block requires fields or subscribes")
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
		p.parseAutomationEntry(automation)
	}

	if !p.check(lexer.CloseBrace) {
		p.error(fmt.Sprintf("unclosed brace for \"automation\" block opened at line %d", automation.OpenPos.Line))
		return automation
	}
	closeTok := p.advance()
	automation.ClosePos = p.position(closeTok)

	switch {
	case automation.OnEvent != "" && automation.Schedule != "":
		p.errorAtPosition(automation.NamePos, "automation block cannot declare both on and every")
	case automation.OnEvent == "" && automation.Schedule == "":
		p.errorAtPosition(automation.NamePos, "automation block requires either an on event or an every schedule")
	}
	if automation.Command == "" {
		p.errorAtPosition(automation.NamePos, "automation block requires a command")
	}

	return automation
}

func (p *Instance) parseAutomationEntry(automation *ast.Automation) {
	switch p.peek().Type {
	case lexer.KeywordDescription:
		p.parseQuotedEntryInto("automation", &automation.Description, &automation.DescriptionPos)
	case lexer.KeywordOn:
		p.parseIdentifierEntryInto("automation", &automation.OnEvent, &automation.OnEventPos)
	case lexer.KeywordEvery:
		p.parseQuotedEntryInto("automation", &automation.Schedule, &automation.SchedulePos)
	case lexer.KeywordTrigger:
		triggerTok := p.advance()
		p.errorAt(triggerTok, "trigger is not an automation entry: name the activation event with on")
		p.skipRestOfLineOrBlockEnd(triggerTok)
	case lexer.KeywordReads:
		p.parseIdentifierEntryInto("automation", &automation.Reads, &automation.ReadsPos)
	case lexer.KeywordCommand:
		p.advance()
		if !p.check(lexer.Identifier) {
			p.error("expected identifier after command in automation")
			p.advance()
			return
		}
		cmdTok := p.advance()
		automation.Command = cmdTok.Value
		automation.CommandPos = p.position(cmdTok)
	case lexer.KeywordTarget:
		p.advance()
		if !p.check(lexer.KeywordContext) {
			p.error("expected context after target in automation")
			p.advance()
			return
		}
		p.advance()
		if !p.check(lexer.Identifier) {
			p.error("expected identifier after target context in automation")
			p.advance()
			return
		}
		ctxTok := p.advance()
		automation.TargetContext = ctxTok.Value
		automation.TargetContextPos = p.position(ctxTok)
	default:
		p.error(fmt.Sprintf("expected description, on, every, reads, command, or target in automation, got %q", p.peek().Value))
		p.advance()
	}
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
			p.parseQuotedEntryInto("translation", &translation.Description, &translation.DescriptionPos)
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
		p.errorAtPosition(translation.NamePos, "translation block requires an external_system")
	}
	if translation.Reads == "" {
		p.errorAtPosition(translation.NamePos, "translation block requires a reads view")
	}
	if translation.Command == "" {
		p.errorAtPosition(translation.NamePos, "translation block requires a command")
	}

	return translation
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

	if !p.checkIdentifierLikeSameLineAs(nameTok) {
		p.errorAt(nameTok, "expected field type")
		p.skipRestOfLineOrBlockEnd(nameTok)
		return field
	}
	typeTok := p.advance()
	field.Type = typeTok.Value
	field.TypePos = p.position(typeTok)

	if p.checkIdentifierLikeSameLineAs(nameTok) {
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

func (p *Instance) parseQuotedEntryInto(construct string, value *string, position *ast.Position) {
	keywordTok := p.advance()
	if !p.check(lexer.String) {
		offending := p.peek()
		p.errorAt(offending, fmt.Sprintf("expected quoted string after %s in %s, got %q", keywordTok.Value, construct, offending.Value))
		p.skipRestOfLineOrBlockEnd(keywordTok)
		return
	}

	tok := p.advance()
	*value, *position = tok.Value, p.position(tok)
}

func (p *Instance) parseIdentifierEntryInto(construct string, name *string, position *ast.Position) {
	keywordTok := p.advance()
	if !p.check(lexer.Identifier) {
		p.errorAt(keywordTok, fmt.Sprintf("expected identifier after %s in %s", keywordTok.Value, construct))
		p.skipRestOfLineOrBlockEnd(keywordTok)
		return
	}

	tok := p.advance()
	*name, *position = tok.Value, p.position(tok)
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

func (p *Instance) checkAny(types ...lexer.Kind) bool {
	for _, typ := range types {
		if p.check(typ) {
			return true
		}
	}
	return false
}

func (p *Instance) checkSameLineAs(tok *lexer.Token) bool {
	return !p.isAtEnd() && p.peek().Line == tok.Line
}

func (p *Instance) checkIdentifierLike() bool {
	if p.isAtEnd() {
		return false
	}
	typ := p.tokens[p.pos].Type
	return typ == lexer.Identifier || typ.IsKeyword()
}

func (p *Instance) checkIdentifierLikeSameLineAs(tok *lexer.Token) bool {
	return p.checkSameLineAs(tok) && p.checkIdentifierLike()
}

func (p *Instance) consume(typ lexer.Kind, msg string) {
	if !p.check(typ) {
		p.error(msg)
		return
	}
	p.advance()
}

func (p *Instance) skipRestOfLineOrBlockEnd(tok *lexer.Token) {
	for p.checkSameLineAs(tok) && !p.check(lexer.CloseBrace) {
		p.advance()
	}
}

// skipTo advances tokens until one of the given types is found, or the end
// of input is reached. Used for error recovery.
func (p *Instance) skipTo(types ...lexer.Kind) {
	for !p.isAtEnd() && !p.checkAny(types...) {
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
	p.errorAtPosition(p.position(tok), msg)
}

func (p *Instance) errorAtPosition(pos ast.Position, msg string) {
	p.diagnostics = append(p.diagnostics, &diagnostic.Entry{
		Filename: pos.Filename,
		Line:     pos.Line,
		Column:   pos.Column,
		Message:  msg,
	})
}

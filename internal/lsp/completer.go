package lsp

import (
	"strings"

	"github.com/hpcsc/emod/internal/ast"
)

// GetCompletions returns the completions available at the cursor. Where the
// cursor sits in the value position of an entry naming something the model
// declares, those names are offered; everywhere else it is the keywords the
// enclosing block accepts.
//
// Positions are 0-based LSP coordinates (line, character).
func GetCompletions(text string, line, character int) CompletionList {
	return CompletionList{
		IsIncomplete: false,
		Items:        completionsAt(text, line, character),
	}
}

func completionsAt(text string, line, character int) []CompletionItem {
	lines := strings.Split(text, "\n")
	block := enclosingBlock(lines, line)
	if slot, ok := valueSlotBefore(linePrefix(lines, line, character), block); ok {
		return valueCompletions(text, line, slot)
	}
	return keywordCompletions(block)
}

type blockContext int

const (
	ctxUnknown blockContext = iota
	ctxContext
	ctxAggregate
	ctxSlice
	ctxCommand
	ctxEvent
	ctxAutomation
	ctxDecidesOn
	ctxTags
	ctxFields
	ctxSpec
)

func enclosingBlock(lines []string, line int) blockContext {
	if line >= len(lines) {
		line = len(lines) - 1
	}

	var scanner blockScanner
	for i := 0; i <= line; i++ {
		scanner.consume(lines[i])
	}
	return scanner.innermost()
}

type blockScanner struct {
	blocks               []blockContext
	keywordAwaitingBrace blockContext
}

func (s *blockScanner) consume(line string) {
	code := codeOutsideStringsAndComments(line)
	keyword := findBlockKeyword(code)
	opens := strings.Count(code, "{")

	switch {
	case opens > 0:
		opener := keyword
		if opener == ctxUnknown {
			opener = s.keywordAwaitingBrace
		}
		s.blocks = append(s.blocks, opener)
		for i := 1; i < opens; i++ {
			s.blocks = append(s.blocks, ctxUnknown)
		}
		s.keywordAwaitingBrace = ctxUnknown
	case code != "":
		// A keyword holds its claim on an opening brace only until the next line that
		// carries code, so `command Ship` inside an automation stays a reference to a
		// command rather than opening a command block for the rest of the body.
		s.keywordAwaitingBrace = keyword
	}

	s.closeBlocks(strings.Count(code, "}"))
}

func (s *blockScanner) closeBlocks(braces int) {
	if braces > len(s.blocks) {
		braces = len(s.blocks)
	}
	s.blocks = s.blocks[:len(s.blocks)-braces]
}

func (s *blockScanner) innermost() blockContext {
	if len(s.blocks) == 0 {
		return ctxUnknown
	}
	return s.blocks[len(s.blocks)-1]
}

// A string literal's contents are not code: a `#` inside one starts no comment and a
// brace inside one delimits no block. The quotes themselves stay, so a line holding
// only a string still reads as carrying code.
func codeOutsideStringsAndComments(line string) string {
	var code strings.Builder
	inString := false
	for i := 0; i < len(line); i++ {
		switch ch := line[i]; {
		case inString:
			if ch == '"' {
				inString = false
				code.WriteByte(ch)
			}
		case ch == '"':
			inString = true
			code.WriteByte(ch)
		case ch == '#', ch == '/' && i+1 < len(line) && line[i+1] == '/':
			return strings.TrimSpace(code.String())
		default:
			code.WriteByte(ch)
		}
	}
	return strings.TrimSpace(code.String())
}

func findBlockKeyword(code string) blockContext {
	fields := strings.Fields(code)
	if len(fields) == 0 {
		return ctxUnknown
	}

	switch fields[0] {
	case "context":
		return ctxContext
	case "aggregate":
		return ctxAggregate
	case "slice":
		return ctxSlice
	case "command":
		return ctxCommand
	case "event":
		return ctxEvent
	case "automation":
		return ctxAutomation
	case "decides_on":
		return ctxDecidesOn
	case "tags":
		return ctxTags
	case "fields":
		return ctxFields
	case "spec":
		return ctxSpec
	}
	return ctxUnknown
}

func linePrefix(lines []string, line, character int) string {
	if line < 0 || line >= len(lines) {
		return ""
	}
	current := lines[line]
	if character > len(current) {
		character = len(current)
	}
	if character < 0 {
		character = 0
	}
	return current[:character]
}

// valueSlot names what an entry keyword accepts after it. A slot is handed the
// cursor's line as well as the model, because an invariant name resolves in the
// scope enclosing the line rather than model-wide.
type valueSlot struct {
	items func(model *ast.Model, line int) []CompletionItem
}

var valueSlots = map[string]valueSlot{
	"on":       {items: namesOfKind(eventName, EventCompletion)},
	"reads":    {items: namesOfKind(viewName, ClassCompletion)},
	"rejected": {items: invariantsInScope},
	"given":    {items: namesOfKind(eventName, EventCompletion)},
	"then":     {items: namesOfKind(eventName, EventCompletion)},
	"when":     {items: commandAndEventNames},
}

// A spec's then takes an event list, a rejection, a view or a command, so which
// names belong after it is decided by the word following it rather than by then
// alone.
var compoundValueSlots = map[string]valueSlot{
	"then view":    {items: namesOfKind(viewName, ClassCompletion)},
	"then command": {items: namesOfKind(commandName, FunctionCompletion)},
}

// The item kinds are the ones GetSemanticTokens gives the same names, so a name
// an editor paints as an event does not complete as something else.
func namesOfKind(kind nameKind, itemKind CompletionItemKind) func(*ast.Model, int) []CompletionItem {
	return func(model *ast.Model, _ int) []CompletionItem {
		return completionItems(declaredNamesInOrder(model, kind), itemKind)
	}
}

// GetSemanticTokens paints no invariant, so an invariant takes the one kind this
// story offers it under: a named rule stated once and referred to by name.
func invariantsInScope(model *ast.Model, line int) []CompletionItem {
	return completionItems(invariantNamesInScopeAt(model, line), ConstantCompletion)
}

// A spec's when resolves against commands and events both, because a command
// slice's when names a command while an automation slice's names the triggering
// event.
func commandAndEventNames(model *ast.Model, _ int) []CompletionItem {
	commands := completionItems(declaredNamesInOrder(model, commandName), FunctionCompletion)
	events := completionItems(declaredNamesInOrder(model, eventName), EventCompletion)
	return append(commands, events...)
}

func valueSlotBefore(prefix string, block blockContext) (valueSlot, bool) {
	// Keywords stay legal as field names, types and modifiers, so `id reads required`
	// is a field line and names no view.
	if block == ctxFields {
		return valueSlot{}, false
	}

	typed := strings.Fields(codeOutsideStringsAndComments(prefix))
	// The word the cursor still touches is half-typed rather than finished, so
	// `on` completes from the keyword list while `on ` opens the value slot.
	if len(typed) > 0 && strings.HasSuffix(prefix, typed[len(typed)-1]) {
		typed = typed[:len(typed)-1]
	}

	for i := len(typed) - 1; i >= 0; i-- {
		// An arrow makes the line a flow or rejection entry, whose parts are
		// positional rather than named by the keyword before them: the
		// identifier after `rejected` on `command -> rejected: X -> Y` is a
		// command, not the invariant the same word introduces inside a spec.
		if strings.Contains(typed[i], "->") {
			return valueSlot{}, false
		}
		if i > 0 {
			if slot, ok := compoundValueSlots[typed[i-1]+" "+typed[i]]; ok {
				return slot, true
			}
		}
		if slot, ok := valueSlots[typed[i]]; ok {
			return slot, true
		}
	}
	return valueSlot{}, false
}

func valueCompletions(text string, line int, slot valueSlot) []CompletionItem {
	model, _ := parseModel(text, "")
	if model == nil {
		return []CompletionItem{}
	}
	return slot.items(model, cursorAt(line, 0).line)
}

func keywordCompletions(block blockContext) []CompletionItem {
	return completionItems(keywordsFor(block), KeywordCompletion)
}

func completionItems(labels []string, kind CompletionItemKind) []CompletionItem {
	items := make([]CompletionItem, len(labels))
	for i, label := range labels {
		items[i] = CompletionItem{Label: label, Kind: kind}
	}
	return items
}

func keywordsFor(block blockContext) []string {
	switch block {
	case ctxUnknown:
		return []string{"model", "actor", "context"}
	case ctxContext:
		return []string{"aggregate", "slice", "invariant"}
	case ctxAggregate:
		return []string{"slice", "invariant"}
	case ctxSlice:
		return []string{"command", "event", "trigger", "view", "automation", "translation", "flow"}
	case ctxCommand:
		return []string{"fields", "decides_on"}
	case ctxEvent:
		return []string{"fields", "tags"}
	case ctxAutomation:
		return []string{"on", "every", "reads", "command", "target context"}
	case ctxDecidesOn:
		return []string{"events", "where"}
	case ctxTags:
		// A tag entry is `key: fieldRef`, both of them free identifiers, so the
		// block accepts no keyword of its own. Without an arm here the scanner
		// reads the body as unknown and offers the top-level list instead.
		return nil
	case ctxFields:
		return []string{"string", "date", "timestamp", "int", "required", "optional"}
	case ctxSpec:
		// The order emod fmt writes a spec's entries in. A spec accepts no
		// description, so none is offered.
		return []string{"given", "when", "then"}
	}
	return nil
}

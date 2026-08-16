package lsp

import (
	"strings"

	"github.com/hpcsc/emod/internal/ast"
)

// GetCompletions returns the completions available at the cursor. Inside a spec
// element's payload braces the declared field names of the construct that element
// names are offered, and a name the model does not declare offers nothing rather
// than falling back to the block around it. Where the cursor sits in the value
// position of an entry naming something the model declares, those names are
// offered; everywhere else it is the keywords the enclosing block accepts.
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
	at := enclosingBlock(lines, line, character)
	if at.block.payload.owner != "" {
		return payloadCompletions(text, at.block.payload)
	}
	if slot, ok := valueSlotBefore(linePrefix(lines, line, character), at.block.context); ok {
		return valueCompletions(text, line, slot)
	}
	// A caret on a continuation line of a wrapped list has no keyword of its own
	// to scan back to, so the entry whose brackets are still open decides.
	if slot, ok := valueSlots[at.openEntry]; ok {
		return valueCompletions(text, line, slot)
	}
	return keywordCompletions(at.block.context)
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

// cursorContext is what the line scan knows about where the caret sits: the
// innermost block open around it, and the spec entry whose bracketed list is
// still open, if any.
type cursorContext struct {
	block     block
	openEntry string
}

// enclosingBlock reads the document down to the cursor. The cursor line is read
// only as far as the cursor, because a payload is usually written whole on one
// line and reading past the caret would close the very block the caret sits in.
func enclosingBlock(lines []string, line, character int) cursorContext {
	if line >= len(lines) {
		line = len(lines) - 1
	}

	var scanner blockScanner
	for i := 0; i < line; i++ {
		scanner.consume(lines[i])
	}
	if line >= 0 {
		scanner.consume(linePrefix(lines, line, character))
	}
	return cursorContext{block: scanner.innermost(), openEntry: scanner.openEntry}
}

// block is one open pair of braces: the entries it accepts, and — where the
// braces are a spec element's payload — the construct whose fields they admit.
type block struct {
	context blockContext
	payload payloadRef
}

// payloadRef names the construct a payload's values belong to. A spec element's
// payload opens on the line of the reference it qualifies, so the construct is
// the identifier the brace follows and the kind is the spec entry that leads it.
type payloadRef struct {
	owner string
	kinds []nameKind
}

// payloadEntries maps the spec entries that accept a payload to the kinds of
// construct their element names, mirroring how each entry's name resolves: a
// command slice's when names a command while an automation slice's names the
// triggering event, so a when payload has to try both.
var payloadEntries = map[string][]nameKind{
	"when":  {commandName, eventName},
	"given": {eventName},
	"then":  {eventName},
}

type blockScanner struct {
	blocks               []block
	keywordAwaitingBrace blockContext
	// openEntry is the spec entry whose bracketed list is still open, and
	// listDepth how many brackets deep that list runs. emod fmt wraps an entry
	// past the column budget, putting `given [` on one line and each element it
	// qualifies on its own line below, so the entry has to outlive every line
	// until the bracket that opened the list is closed — not merely the one line
	// that stated it, which would lose the second element and any comment
	// between the bracket and the first.
	openEntry string
	listDepth int
	// listBlocks is how deep the block stack was when the open list started. A
	// list belongs to the block that opened it, so once that block closes an
	// unclosed bracket must stop claiming the lines below it — otherwise one
	// stray `[` would arm the entry for the whole rest of the document.
	listBlocks int
}

func (s *blockScanner) consume(line string) {
	code := codeOutsideStringsAndComments(line)
	keyword := findBlockKeyword(code)

	entry := s.openEntry
	var preceding string
	opened := false

	for _, token := range lineTokens(code) {
		switch token {
		case "{":
			s.blocks = append(s.blocks, s.opening(keyword, entry, preceding, opened))
			s.keywordAwaitingBrace = ctxUnknown
			opened = true
			preceding = ""
		case "}":
			s.closeBlocks(1)
			preceding = ""
		default:
			preceding = token
			// Only a token at statement position claims the list's entry. Inside
			// an element's braces the same spelling is a payload field label,
			// and letting it through would rewrite the list's own entry.
			if _, ok := payloadEntries[token]; ok && s.innermost().payload.owner == "" {
				entry = token
			}
		}
	}

	startingList := s.listDepth == 0
	s.listDepth += strings.Count(code, "[") - strings.Count(code, "]")
	if s.listDepth < 0 {
		s.listDepth = 0
	}
	if s.listDepth > 0 && startingList {
		s.listBlocks = len(s.blocks)
	}
	if s.listDepth == 0 || len(s.blocks) < s.listBlocks {
		s.listDepth = 0
		s.openEntry = ""
	} else {
		s.openEntry = entry
	}

	if !opened && code != "" {
		// A keyword holds its claim on an opening brace only until the next line that
		// carries code, so `command Ship` inside an automation stays a reference to a
		// command rather than opening a command block for the rest of the body.
		s.keywordAwaitingBrace = keyword
	}
}

func (s *blockScanner) opening(keyword blockContext, entry, preceding string, alreadyOpened bool) block {
	if kinds, ok := payloadEntries[entry]; ok && preceding != "" && preceding != entry {
		return block{payload: payloadRef{owner: preceding, kinds: kinds}}
	}
	if alreadyOpened {
		return block{}
	}
	if keyword != ctxUnknown {
		return block{context: keyword}
	}
	return block{context: s.keywordAwaitingBrace}
}

// lineTokens splits a line's code into its words and the braces between them, so
// a brace can be read against the word it follows. Bracket, comma, colon and
// quote separate words without being words themselves.
func lineTokens(code string) []string {
	var tokens []string
	var word strings.Builder
	flush := func() {
		if word.Len() > 0 {
			tokens = append(tokens, word.String())
			word.Reset()
		}
	}

	for i := 0; i < len(code); i++ {
		switch ch := code[i]; ch {
		case '{', '}':
			flush()
			tokens = append(tokens, string(ch))
		case ' ', '\t', '[', ']', ',', ':', '"':
			flush()
		default:
			word.WriteByte(ch)
		}
	}
	flush()

	return tokens
}

func (s *blockScanner) closeBlocks(braces int) {
	if braces > len(s.blocks) {
		braces = len(s.blocks)
	}
	s.blocks = s.blocks[:len(s.blocks)-braces]
}

func (s *blockScanner) innermost() block {
	if len(s.blocks) == 0 {
		return block{}
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

// payloadCompletions offers the fields of the construct the payload qualifies and
// nothing else: a payload's values belong to that construct or to no one, so a
// name the model does not declare offers an empty list rather than falling back
// to the keywords of the block around it.
func payloadCompletions(text string, payload payloadRef) []CompletionItem {
	model, _ := parseModel(text, "")
	if model == nil {
		return []CompletionItem{}
	}
	for _, kind := range payload.kinds {
		if names := declaredFieldNames(model, kind, payload.owner); names != nil {
			return completionItems(names, FieldCompletion)
		}
	}
	return []CompletionItem{}
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

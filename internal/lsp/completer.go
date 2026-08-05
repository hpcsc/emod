package lsp

import (
	"strings"
)

// GetCompletions returns context-appropriate keyword completions based on cursor position.
// It examines the document text to determine which block the cursor falls into,
// then returns the keywords that are valid in that context.
func GetCompletions(text string, line, character int) CompletionList {
	ctx := resolveContext(text, line, character)
	items := completionsFor(ctx)
	return CompletionList{
		IsIncomplete: false,
		Items:        items,
	}
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
	ctxFields
)

func resolveContext(text string, line, character int) blockContext {
	lines := strings.Split(text, "\n")
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
	case "fields":
		return ctxFields
	}
	return ctxUnknown
}

func completionsFor(ctx blockContext) []CompletionItem {
	labels := labelsFor(ctx)
	items := make([]CompletionItem, len(labels))
	for i, label := range labels {
		items[i] = CompletionItem{
			Label: label,
			Kind:  KeywordCompletion,
		}
	}
	return items
}

func labelsFor(ctx blockContext) []string {
	switch ctx {
	case ctxUnknown:
		return []string{"model", "actor", "context"}
	case ctxContext:
		return []string{"aggregate"}
	case ctxAggregate:
		return []string{"slice"}
	case ctxSlice:
		return []string{"command", "event", "trigger", "view", "automation", "translation", "flow"}
	case ctxCommand, ctxEvent:
		return []string{"fields"}
	case ctxAutomation:
		return []string{"on", "every", "reads", "command", "target context"}
	case ctxFields:
		return []string{"string", "date", "timestamp", "int", "required", "optional"}
	}
	return nil
}

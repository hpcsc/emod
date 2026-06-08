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
	ctxUnknown   blockContext = iota
	ctxContext
	ctxAggregate
	ctxSlice
	ctxCommand
	ctxEvent
	ctxFields
)

// resolveContext scans the document text up to the cursor position and determines
// which block context the cursor falls into by tracking brace nesting and keywords.
func resolveContext(text string, line, character int) blockContext {
	if text == "" {
		return ctxUnknown
	}

	lines := strings.Split(text, "\n")
	if line >= len(lines) {
		line = len(lines) - 1
	}

	stack := []blockContext{}
	// pendingKeyword tracks a keyword whose opening brace hasn't been reached yet
	// (e.g. keyword on one line, { on the next)
	pendingKeyword := ctxUnknown

	for i := 0; i <= line; i++ {
		l := lines[i]
		kw := findBlockKeyword(l)

		opens := strings.Count(l, "{")
		closes := strings.Count(l, "}")

		if kw != ctxUnknown {
			if opens > 0 {
				stack = append(stack, kw)
				for j := 1; j < opens; j++ {
					stack = append(stack, ctxUnknown)
				}
			} else {
				pendingKeyword = kw
			}
		} else if pendingKeyword != ctxUnknown && opens > 0 {
			stack = append(stack, pendingKeyword)
			pendingKeyword = ctxUnknown
			for j := 1; j < opens; j++ {
				stack = append(stack, ctxUnknown)
			}
		} else if opens > 0 {
			// Anonymous opening brace (e.g. "model {" or standalone "{")
			for j := 0; j < opens; j++ {
				stack = append(stack, ctxUnknown)
			}
		}

		if closes > 0 {
			pops := closes
			if pops > len(stack) {
				pops = len(stack)
			}
			stack = stack[:len(stack)-pops]
		}
	}

	// Cursor is on a keyword line before its opening brace — still in parent context
	if pendingKeyword != ctxUnknown {
		return ctxUnknown
	}

	if len(stack) > 0 {
		return stack[len(stack)-1]
	}
	return ctxUnknown
}

func findBlockKeyword(line string) blockContext {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return ctxUnknown
	}

	if idx := strings.Index(trimmed, "//"); idx >= 0 {
		trimmed = strings.TrimSpace(trimmed[:idx])
	}
	if trimmed == "" {
		return ctxUnknown
	}

	fields := strings.Fields(trimmed)
	if len(fields) == 0 {
		return ctxUnknown
	}
	keyword := fields[0]

	switch keyword {
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
	case "fields":
		return ctxFields
	}
	return ctxUnknown
}

func completionsFor(ctx blockContext) []CompletionItem {
	var labels []string
	switch ctx {
	case ctxUnknown:
		labels = []string{"model", "actor", "context"}
	case ctxContext:
		labels = []string{"aggregate"}
	case ctxAggregate:
		labels = []string{"slice"}
	case ctxSlice:
		labels = []string{"command", "event", "trigger", "view", "automation", "translation", "flow"}
	case ctxCommand, ctxEvent:
		labels = []string{"fields"}
	case ctxFields:
		labels = []string{"string", "date", "timestamp", "int", "required", "optional"}
	}

	items := make([]CompletionItem, len(labels))
	for i, label := range labels {
		items[i] = CompletionItem{
			Label: label,
			Kind:  KeywordCompletion,
		}
	}
	return items
}

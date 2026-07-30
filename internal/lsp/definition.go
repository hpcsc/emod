package lsp

import (
	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/lexer"
	"github.com/hpcsc/emod/internal/parser"
)

// GetDefinition finds the definition location for the reference at the given
// cursor position. If the cursor is not on a known reference, or the referenced
// name has no definition in the document, it returns nil.
//
// Positions are 0-based LSP coordinates (line, character).
func GetDefinition(text string, line, character int, uri string) *Location {
	if text == "" {
		return nil
	}

	tokens, _ := lexer.Scan(text, uri)
	p := parser.New(tokens, uri)
	model, _ := p.Parse()
	if model == nil {
		return nil
	}

	// Build definition position maps: name → definition position (1-based AST)
	commandDefs := make(map[string]ast.Position)
	eventDefs := make(map[string]ast.Position)
	viewDefs := make(map[string]ast.Position)
	contextDefs := make(map[string]ast.Position)

	for _, ctx := range model.Contexts {
		contextDefs[ctx.Name] = ctx.NamePos
		for _, agg := range ctx.Aggregates {
			for _, slice := range agg.Slices {
				for _, cmd := range slice.Commands {
					commandDefs[cmd.Name] = cmd.NamePos
				}
				for _, evt := range slice.Events {
					eventDefs[evt.Name] = evt.NamePos
				}
				for _, v := range slice.Views {
					viewDefs[v.Name] = v.NamePos
				}
			}
		}
	}

	// Convert cursor from 0-based LSP to 1-based AST coordinates.
	cursorLine := line + 1
	cursorChar := character + 1

	// Check each reference type by iterating through the model.

	for _, ctx := range model.Contexts {
		for _, agg := range ctx.Aggregates {
			for _, slice := range agg.Slices {

				// 1. View subscribes → event definitions
				for _, v := range slice.Views {
					for i, sub := range v.Subscribes {
						if i < len(v.SubscribesPos) {
							if cursorOnName(cursorLine, cursorChar, v.SubscribesPos[i], sub) {
								if defPos, ok := eventDefs[sub]; ok {
									return locationFor(uri, defPos, sub)
								}
							}
						}
					}
				}

				// 2. Automation references
				for _, auto := range slice.Automations {
					if cursorOnName(cursorLine, cursorChar, auto.OnEventPos, auto.OnEvent) {
						if defPos, ok := eventDefs[auto.OnEvent]; ok {
							return locationFor(uri, defPos, auto.OnEvent)
						}
					}
					if cursorOnName(cursorLine, cursorChar, auto.CommandPos, auto.Command) {
						if defPos, ok := commandDefs[auto.Command]; ok {
							return locationFor(uri, defPos, auto.Command)
						}
					}
					if cursorOnName(cursorLine, cursorChar, auto.TargetContextPos, auto.TargetContext) {
						if defPos, ok := contextDefs[auto.TargetContext]; ok {
							return locationFor(uri, defPos, auto.TargetContext)
						}
					}
				}

				// 3. Translation references
				for _, tr := range slice.Translations {
					if cursorOnName(cursorLine, cursorChar, tr.ReadsPos, tr.Reads) {
						if defPos, ok := viewDefs[tr.Reads]; ok {
							return locationFor(uri, defPos, tr.Reads)
						}
					}
					if cursorOnName(cursorLine, cursorChar, tr.CommandPos, tr.Command) {
						if defPos, ok := commandDefs[tr.Command]; ok {
							return locationFor(uri, defPos, tr.Command)
						}
					}
				}

				// 4. Trigger reads → view definitions
				if slice.Trigger != nil {
					if cursorOnName(cursorLine, cursorChar, slice.Trigger.ReadsPos, slice.Trigger.Reads) {
						if defPos, ok := viewDefs[slice.Trigger.Reads]; ok {
							return locationFor(uri, defPos, slice.Trigger.Reads)
						}
					}
				}

				// 5. Flow command/event references
				for _, f := range slice.Flows {
					if cursorOnName(cursorLine, cursorChar, f.CommandPos, f.CommandName) {
						if defPos, ok := commandDefs[f.CommandName]; ok {
							return locationFor(uri, defPos, f.CommandName)
						}
					}
					if cursorOnName(cursorLine, cursorChar, f.EventPos, f.EventName) {
						if defPos, ok := eventDefs[f.EventName]; ok {
							return locationFor(uri, defPos, f.EventName)
						}
					}
				}
			}
		}
	}

	return nil
}

// cursorOnName checks whether the cursor (1-based) falls within the text
// span of the given named token at the given AST position (1-based).
func cursorOnName(cursorLine, cursorChar int, pos ast.Position, name string) bool {
	if name == "" {
		return false
	}
	return cursorLine == pos.Line &&
		cursorChar >= pos.Column &&
		cursorChar < pos.Column+len(name)
}

// locationFor builds an LSP Location for the given definition position,
// converting from 1-based AST coordinates to 0-based LSP coordinates.
// The range covers the full definition name.
func locationFor(uri string, pos ast.Position, name string) *Location {
	if name == "" {
		return nil
	}
	return &Location{
		URI: uri,
		Range: Range{
			Start: Position{
				Line:      pos.Line - 1,
				Character: pos.Column - 1,
			},
			End: Position{
				Line:      pos.Line - 1,
				Character: pos.Column - 1 + len(name),
			},
		},
	}
}

package lsp

import (
	"fmt"
	"strings"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/lexer"
	"github.com/hpcsc/emod/internal/parser"
)

// keywordDescriptions maps EMOD keyword strings to their hover descriptions.
var keywordDescriptions = map[string]string{
	"model":           "Declares the domain model name.",
	"actor":           "Declares an actor in the domain.",
	"context":         "Defines a bounded context.",
	"aggregate":       "Defines an aggregate root.",
	"slice":           "Defines a slice within an aggregate.",
	"command":         "Defines a command that can be sent to an aggregate.",
	"event":           "Defines an event that represents a state change.",
	"fields":          "Defines the fields of an event.",
	"flow":            "Defines the flow between commands and events.",
	"trigger":         "Defines a manual trigger for a slice.",
	"view":            "Defines a read model that subscribes to events.",
	"automation":      "Defines an automation that triggers on an event and sends a command.",
	"translation":     "Defines a translation from a view to a command.",
	"subscribes":      "Defines the events a view subscribes to.",
	"target":          "Defines the target context for an automation.",
	"external_system": "Defines an external system for a translation.",
	"reads":           "Defines the view a trigger or translation reads from.",
	"source":          "Defines the source for an external system.",
	"external":        "Declares an external reference.",
}

// isKeyword returns true for recognized EMOD keyword token kinds.
func isKeyword(k lexer.Kind) bool {
	return k >= lexer.KeywordModel && k <= lexer.KeywordExternal
}

// GetHover returns hover information for the token at the given cursor position.
// If the cursor is on a command, event, or view definition name, it returns
// contextual hover text describing the element's parent context and aggregate,
// plus relevant details.
// If the cursor is on an EMOD keyword, it returns a brief description.
// For any unrecognized or non-resolvable token, it returns nil.
//
// Positions are 0-based LSP coordinates (line, character).
func GetHover(text string, line, character int) *Hover {
	if text == "" {
		return nil
	}

	tokens, _ := lexer.Scan(text, "")
	p := parser.New(tokens, "")
	model, _ := p.Parse()
	if model == nil {
		return nil
	}

	// Convert cursor from 0-based LSP to 1-based AST coordinates.
	cursorLine := line + 1
	cursorChar := character + 1

	for _, ctx := range model.Contexts {
		for _, agg := range ctx.Aggregates {
			for _, slice := range agg.Slices {
				for _, cmd := range slice.Commands {
					if cursorOnName(cursorLine, cursorChar, cmd.NamePos, cmd.Name) {
						return hoverForCommand(cmd, ctx, agg)
					}
				}
				for _, evt := range slice.Events {
					if cursorOnName(cursorLine, cursorChar, evt.NamePos, evt.Name) {
						return hoverForEvent(evt, ctx, agg)
					}
				}
				for _, v := range slice.Views {
					if cursorOnName(cursorLine, cursorChar, v.NamePos, v.Name) {
						return hoverForView(v, ctx, agg)
					}
				}
			}
		}
	}

	// Check for keywords — tokens with no AST definition name.
	for _, tok := range tokens {
		if isKeyword(tok.Type) {
			if cursorOnName(cursorLine, cursorChar, ast.Position{Line: tok.Line, Column: tok.Column}, tok.Value) {
				if desc, ok := keywordDescriptions[tok.Value]; ok {
					return &Hover{
						Contents: MarkupContent{
							Kind:  Markdown,
							Value: desc,
						},
						Range: nameRange(ast.Position{Line: tok.Line, Column: tok.Column}, tok.Value),
					}
				}
			}
		}
	}

	return nil
}

func hoverForCommand(cmd *ast.Command, ctx *ast.Context, agg *ast.Aggregate) *Hover {
	content := fmt.Sprintf("**Command** in %s > %s", ctx.Name, agg.Name)
	return &Hover{
		Contents: MarkupContent{
			Kind:  Markdown,
			Value: content,
		},
		Range: nameRange(cmd.NamePos, cmd.Name),
	}
}

func hoverForEvent(evt *ast.Event, ctx *ast.Context, agg *ast.Aggregate) *Hover {
	var b strings.Builder
	fmt.Fprintf(&b, "**Event** in %s > %s", ctx.Name, agg.Name)
	if len(evt.Fields) > 0 {
		b.WriteString("\n\n**Fields:**")
		for _, f := range evt.Fields {
			if f.Modifier != "" {
				fmt.Fprintf(&b, "\n- %s %s %s", f.Name, f.Type, f.Modifier)
			} else {
				fmt.Fprintf(&b, "\n- %s %s", f.Name, f.Type)
			}
		}
	}
	return &Hover{
		Contents: MarkupContent{
			Kind:  Markdown,
			Value: b.String(),
		},
		Range: nameRange(evt.NamePos, evt.Name),
	}
}

func hoverForView(v *ast.View, ctx *ast.Context, agg *ast.Aggregate) *Hover {
	var b strings.Builder
	fmt.Fprintf(&b, "**View** in %s > %s", ctx.Name, agg.Name)
	if len(v.Subscribes) > 0 {
		b.WriteString("\n\n**Subscribes:**")
		for _, sub := range v.Subscribes {
			fmt.Fprintf(&b, "\n- %s", sub)
		}
	}
	return &Hover{
		Contents: MarkupContent{
			Kind:  Markdown,
			Value: b.String(),
		},
		Range: nameRange(v.NamePos, v.Name),
	}
}

// nameRange builds an LSP Range for the given name position,
// converting from 1-based AST coordinates to 0-based LSP coordinates.
func nameRange(pos ast.Position, name string) *Range {
	if name == "" {
		return nil
	}
	return &Range{
		Start: Position{
			Line:      pos.Line - 1,
			Character: pos.Column - 1,
		},
		End: Position{
			Line:      pos.Line - 1,
			Character: pos.Column - 1 + len(name),
		},
	}
}

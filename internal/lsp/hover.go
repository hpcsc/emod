package lsp

import (
	"fmt"
	"strings"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/lexer"
	"github.com/hpcsc/emod/internal/parser"
)

// GetHover returns hover information for the token at the given cursor position.
// If the cursor is on a command, event, or view definition name, it returns
// contextual hover text describing the element's parent context and aggregate,
// plus relevant details.
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

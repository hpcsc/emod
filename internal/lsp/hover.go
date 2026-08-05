package lsp

import (
	"fmt"
	"strings"

	"github.com/hpcsc/emod/internal/ast"
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

// GetHover returns hover information for the token at the given cursor position.
// If the cursor is on a command, event, or view definition name, it returns
// contextual hover text describing the context the element was declared in, the
// aggregate where it has one, plus relevant details.
// If the cursor is on an EMOD keyword, it returns a brief description.
// For any unrecognized or non-resolvable token, it returns nil.
//
// Positions are 0-based LSP coordinates (line, character).
func GetHover(text string, line, character int) *Hover {
	if text == "" {
		return nil
	}

	model, tokens := parseModel(text, "")
	if model == nil {
		return nil
	}

	at := cursorAt(line, character)

	for _, scoped := range scopedSlices(model) {
		for _, cmd := range scoped.slice.Commands {
			if at.onName(cmd.NamePos, cmd.Name) {
				return hoverForCommand(cmd, declaredIn(scoped))
			}
		}
		for _, evt := range scoped.slice.Events {
			if at.onName(evt.NamePos, evt.Name) {
				return hoverForEvent(evt, declaredIn(scoped))
			}
		}
		for _, v := range scoped.slice.Views {
			if at.onName(v.NamePos, v.Name) {
				return hoverForView(v, declaredIn(scoped))
			}
		}
	}

	// Check for keywords — tokens with no AST definition name.
	for _, tok := range tokens {
		pos := ast.Position{Line: tok.Line, Column: tok.Column}
		if !tok.Type.IsKeyword() || !at.onName(pos, tok.Value) {
			continue
		}
		if desc, ok := keywordDescriptions[tok.Value]; ok {
			return hoverAt(desc, pos, tok.Value)
		}
	}

	return nil
}

func declaredIn(scoped scopedSlice) string {
	if scoped.aggregate == nil {
		return scoped.context.Name
	}
	return fmt.Sprintf("%s > %s", scoped.context.Name, scoped.aggregate.Name)
}

func hoverForCommand(cmd *ast.Command, scope string) *Hover {
	return hoverAt(fmt.Sprintf("**Command** in %s", scope), cmd.NamePos, cmd.Name)
}

func hoverForEvent(evt *ast.Event, scope string) *Hover {
	content := fmt.Sprintf("**Event** in %s", scope) + bulletList("Fields", fieldDescriptions(evt.Fields))
	return hoverAt(content, evt.NamePos, evt.Name)
}

func hoverForView(v *ast.View, scope string) *Hover {
	content := fmt.Sprintf("**View** in %s", scope) + bulletList("Subscribes", v.Subscribes)
	return hoverAt(content, v.NamePos, v.Name)
}

func hoverAt(content string, pos ast.Position, name string) *Hover {
	return &Hover{
		Contents: MarkupContent{
			Kind:  Markdown,
			Value: content,
		},
		Range: nameRange(pos, name),
	}
}

func fieldDescriptions(fields []*ast.Field) []string {
	var descriptions []string
	for _, f := range fields {
		if f.Modifier != "" {
			descriptions = append(descriptions, fmt.Sprintf("%s %s %s", f.Name, f.Type, f.Modifier))
			continue
		}
		descriptions = append(descriptions, fmt.Sprintf("%s %s", f.Name, f.Type))
	}
	return descriptions
}

func bulletList(title string, items []string) string {
	if len(items) == 0 {
		return ""
	}

	var b strings.Builder
	fmt.Fprintf(&b, "\n\n**%s:**", title)
	for _, item := range items {
		fmt.Fprintf(&b, "\n- %s", item)
	}
	return b.String()
}

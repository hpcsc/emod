package lsp

import (
	"fmt"
	"strings"

	"github.com/hpcsc/emod/internal/ast"
)

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
	"trigger":         "Defines a trigger, the Command pattern's human entry point into a slice: the actor who acts and the view they read.",
	"view":            "Defines a read model that subscribes to events.",
	"automation":      "Defines an automation, the reactive processor of the Automation pattern: activated by an on event or an every schedule, optionally reads a view, and sends a command.",
	"on":              "Names the event whose occurrence activates the automation.",
	"every":           `Sets the schedule that activates the automation: a duration such as "5m", or a five-field cron expression such as "0 2 * * *".`,
	"after":           `Delays the automation by the stated Go duration, such as "24h", measured from each occurrence of its on event. An every schedule is already absolute, so the two never combine.`,
	"translation":     "Defines a translation from a view to a command.",
	"subscribes":      "Defines the events a view subscribes to.",
	"target":          "Defines the target context for an automation.",
	"external_system": "Defines an external system for a translation.",
	"reads":           "Defines the view a trigger, automation or translation reads from.",
	"source":          "Defines the source for an external system.",
	"external":        "Declares an external reference.",
	"emod":            "Declares the emod language version the file is written in.",
	"description":     "Attaches a human-readable description to the enclosing declaration.",
	"invariant":       "Declares a named business rule the enclosing context or aggregate must uphold.",
	"spec":            "Defines a given/when/then scenario a slice must satisfy.",
	"given":           "Lists the events that have already occurred when a spec's scenario starts.",
	"when":            "Names the command or event a spec exercises.",
	"then":            "States a spec's outcome: the events produced, a rejection, a view, or a command.",
	"rejected":        "States that a spec's command is rejected by the named invariant.",
	"mode":            `Sets a context's modeling mode: "aggregate", "dcb", or "mixed".`,
	"tags":            "Defines the tag entries on an event, each mapping a tag key to a field.",
	"tag":             "References a tag key in a decides_on predicate, as tag(key = field).",
	"decides_on":      "Defines the events and predicate a command's decision is based on (DCB mode).",
	"events":          "Lists the event types a decides_on clause reads.",
	"where":           "Filters a decides_on clause with a tag predicate.",
	"and":             "Combines two decides_on predicates; both must hold.",
	"or":              "Combines two decides_on predicates; either may hold.",
	"not":             "Negates a decides_on predicate.",
	"type":            `Binds the event to the type a consumer outside the model routes by, such as "com.acme.reservations.room-reserved". Two events may not share one.`,
}

// GetHover returns hover information for the token at the given cursor position.
// On any declaration name it returns the construct's kind, the context and
// aggregate holding it where it has them, its description where it declares one,
// and the details its kind carries — an event's fields, a view's subscriptions.
// On an invariant it returns the statement, both at the declaration and at every
// site naming it, resolved in the one scope that declares it.
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

	for _, decl := range declaredConstructs(model) {
		if at.onName(decl.namePos, decl.name) {
			return hoverForConstruct(decl)
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

func hoverForConstruct(decl constructDecl) *Hover {
	content := headingFor(decl) +
		paragraph(decl.description) +
		bulletList("Fields", fieldDescriptions(decl.fields)) +
		bulletList("Subscribes", decl.subscribes)
	return hoverAt(content, decl.namePos, decl.name)
}

func headingFor(decl constructDecl) string {
	if decl.scope == "" {
		return fmt.Sprintf("**%s**", decl.kind)
	}
	return fmt.Sprintf("**%s** in %s", decl.kind, decl.scope)
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

func paragraph(text string) string {
	if text == "" {
		return ""
	}
	return "\n\n" + text
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

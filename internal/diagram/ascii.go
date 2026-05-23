// Package diagram renders AST models as diagrams.
package diagram

import (
	"fmt"
	"strings"

	"github.com/hpcsc/emod/internal/ast"
)

// ExportASCII renders a terminal-preview of the event model as human-readable ASCII.
// Elements are visually distinguished with markers:
//   - Commands as [Name]
//   - Events as (Name)
//   - Views as {Name}
//   - Triggers as <<Kind: Name>>
//   - Automations as ⚙ Name
//
// Flows are rendered with ──> arrows between connected elements.
func ExportASCII(model *ast.Model) ([]byte, error) {
	if model == nil {
		return []byte{}, nil
	}

	var b strings.Builder

	entries := collectSlices(model)
	if len(entries) == 0 {
		if model.Name != "" {
			fmt.Fprintf(&b, "Model: %s\n", model.Name)
		}
		return []byte(b.String()), nil
	}

	if model.Name != "" {
		fmt.Fprintf(&b, "Model: %s\n\n", model.Name)
	}

	for _, entry := range entries {
		s := entry.slice

		fmt.Fprintf(&b, "=== Slice: %s ===\n", s.Name)

		// Trigger
		if s.Trigger != nil {
			fmt.Fprintf(&b, "  %s\n", formatTrigger(s.Trigger))
		}

		// Standalone commands (not referenced in any flow)
		standaloneCmds := standaloneCommands(s)
		for _, cmd := range standaloneCmds {
			fmt.Fprintf(&b, "  [%s]\n", cmd.Name)
		}

		// Standalone events (not referenced in any flow, automation, or translation)
		standaloneEvts := standaloneEvents(s)
		for _, evt := range standaloneEvts {
			fmt.Fprintf(&b, "  (%s)\n", evt.Name)
		}

		// Flows (command → event)
		for _, flow := range s.Flows {
			fmt.Fprintf(&b, "  [%s] -> (%s)\n", flow.CommandName, flow.EventName)
		}

		// Views
		for _, view := range s.Views {
			if len(view.Subscribes) > 0 {
				fmt.Fprintf(&b, "  {%s} [%s]\n", view.Name, strings.Join(view.Subscribes, ", "))
			} else {
				fmt.Fprintf(&b, "  {%s}\n", view.Name)
			}
		}

		// Automations (event → ⚙ → command)
		for _, auto := range s.Automations {
			fmt.Fprintf(&b, "  (%s) -> ⚙ %s -> [%s]\n",
				auto.TriggerEvent, auto.Name, auto.Command)
		}

		// Translations (system → command → event)
		for _, tr := range s.Translations {
			evtName := ""
			if tr.Event != nil {
				evtName = tr.Event.Name
			}
			fmt.Fprintf(&b, "  [%s] -> [%s] -> (%s)\n",
				tr.ExternalSystem, tr.Command, evtName)
		}

		fmt.Fprintf(&b, "\n")
	}

	return []byte(b.String()), nil
}

func formatTrigger(t *ast.Trigger) string {
	label := fmt.Sprintf("<<%s: %s", t.Kind, t.Name)
	if t.Actor != "" {
		label += fmt.Sprintf(" (%s)", t.Actor)
	}
	label += ">>"
	return label
}

func standaloneCommands(s *ast.Slice) []*ast.Command {
	flowCmds := make(map[string]bool)
	for _, f := range s.Flows {
		flowCmds[f.CommandName] = true
	}
	for _, t := range s.Translations {
		if t.Command != "" {
			flowCmds[t.Command] = true
		}
	}
	for _, a := range s.Automations {
		flowCmds[a.Command] = true
	}

	var result []*ast.Command
	for _, cmd := range s.Commands {
		if !flowCmds[cmd.Name] {
			result = append(result, cmd)
		}
	}
	return result
}

func standaloneEvents(s *ast.Slice) []*ast.Event {
	flowEvts := make(map[string]bool)
	for _, f := range s.Flows {
		flowEvts[f.EventName] = true
	}
	for _, t := range s.Translations {
		if t.Event != nil && t.Event.Name != "" {
			flowEvts[t.Event.Name] = true
		}
	}
	for _, a := range s.Automations {
		flowEvts[a.TriggerEvent] = true
	}

	var result []*ast.Event
	for _, evt := range s.Events {
		if !flowEvts[evt.Name] {
			result = append(result, evt)
		}
	}
	return result
}

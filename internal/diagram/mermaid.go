package diagram

import (
	"fmt"
	"strings"

	"github.com/hpcsc/emod/internal/ast"
)

// ExportMermaid converts a parsed AST model into Mermaid event modeling diagram markup.
// The output uses the eventmodeling diagram type introduced in Mermaid v11.15.0+,
// with timeframe definitions for triggers, commands, events, views, and automations.
// Contexts are rendered as namespaces using dot notation, which groups entities into
// swimlanes per bounded context.
func ExportMermaid(model *ast.Model) ([]byte, error) {
	if model == nil {
		return []byte{}, nil
	}

	var b strings.Builder
	b.WriteString("eventmodeling\n\n")

	if model.Name != "" {
		b.WriteString(fmt.Sprintf("%% %s\n", model.Name))
		b.WriteString("\n")
	}

	entries := collectSlices(model)
	if len(entries) == 0 {
		return []byte(b.String()), nil
	}

	nextNum := 1
	for _, entry := range entries {
		s := entry.slice
		ns := entry.ctxName

		if s.Name != "" {
			b.WriteString(fmt.Sprintf("%% Slice: %s\n", s.Name))
		}

		if s.Trigger != nil {
			etype := "ui"
			if s.Trigger.Kind == "Schedule" || s.Trigger.Kind == "Processor" {
				etype = "pcr"
			}
			eid := s.Trigger.Name
			if ns != "" {
				eid = ns + "." + eid
			}
			b.WriteString(fmt.Sprintf("tf %02d %s %s\n", nextNum, etype, eid))
			nextNum++
		}

		for _, cmd := range s.Commands {
			eid := cmd.Name
			if ns != "" {
				eid = ns + "." + eid
			}
			b.WriteString(fmt.Sprintf("tf %02d cmd %s\n", nextNum, eid))
			nextNum++
		}

		for _, evt := range s.Events {
			eid := evt.Name
			if ns != "" {
				eid = ns + "." + eid
			}
			b.WriteString(fmt.Sprintf("tf %02d evt %s\n", nextNum, eid))
			nextNum++
		}

		for _, view := range s.Views {
			eid := view.Name
			if ns != "" {
				eid = ns + "." + eid
			}
			b.WriteString(fmt.Sprintf("tf %02d rmo %s\n", nextNum, eid))
			nextNum++
		}

		for _, auto := range s.Automations {
			targetNs := ns
			if auto.TargetContext != "" {
				targetNs = auto.TargetContext
			}
			eid := auto.Name
			if targetNs != "" {
				eid = targetNs + "." + eid
			}
			b.WriteString(fmt.Sprintf("tf %02d pcr %s\n", nextNum, eid))
			nextNum++
		}

		if s.Name != "" {
			b.WriteString("\n")
		}
	}

	return []byte(b.String()), nil
}

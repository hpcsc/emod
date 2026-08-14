package diagram

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hpcsc/emod/internal/ast"
)

// ExportMermaid converts a parsed AST model into Mermaid event modeling diagram markup.
// The output uses the eventmodeling diagram type introduced in Mermaid v11.15.0+,
// with timeframe definitions for triggers, commands, events, views, and automations.
// Contexts are rendered as namespaces using dot notation, which groups entities into
// swimlanes per bounded context.
// When StyleProjected is used with DCB contexts that have tagged events, events are
// grouped by tag key instead of by aggregate context.
func ExportMermaid(model *ast.Model, style Style) ([]byte, error) {
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

	// Determine if projected (tag-based) layout should be used
	hasDCB := false
	for _, e := range entries {
		if e.fromDCB {
			hasDCB = true
			break
		}
	}
	tagKeys := collectTagKeys(entries)
	useProjected := style == StyleProjected && hasDCB && len(tagKeys) > 0
	useDCB := style == StyleDCB && hasDCB

	if useDCB {
		exportMermaidDCB(&b, model.Name, entries)
	} else if useProjected {
		exportMermaidProjected(&b, model.Name, entries, tagKeys)
	} else {
		exportMermaidStandard(&b, model.Name, entries)
	}

	return []byte(b.String()), nil
}

func namespaced(ns, name string) string {
	if ns == "" {
		return name
	}

	return ns + "." + name
}

// exportMermaidStandard renders the standard aggregate-based layout.
// This is the original layout used for aggregate-mode contexts.
func exportMermaidStandard(b *strings.Builder, modelName string, entries []sliceEntry) {
	nextNum := 1
	for _, entry := range entries {
		s := entry.slice
		ns := entry.ctxName

		if s.Name != "" {
			b.WriteString(fmt.Sprintf("%% Slice: %s\n", s.Name))
		}

		if s.Trigger != nil {
			eid := namespaced(ns, s.Trigger.Name)
			b.WriteString(fmt.Sprintf("tf %02d ui %s\n", nextNum, eid))
			nextNum++
		}

		for _, cmd := range s.Commands {
			eid := namespaced(ns, cmd.Name)
			b.WriteString(fmt.Sprintf("tf %02d cmd %s\n", nextNum, eid))
			nextNum++
		}

		for _, evt := range s.Events {
			eid := namespaced(ns, evt.Name)
			b.WriteString(fmt.Sprintf("tf %02d evt %s\n", nextNum, eid))
			nextNum++
		}

		for _, view := range s.Views {
			eid := namespaced(ns, view.Name)
			b.WriteString(fmt.Sprintf("tf %02d rmo %s\n", nextNum, eid))
			nextNum++
			for _, sub := range view.Subscribes {
				b.WriteString(fmt.Sprintf("%%   subscribes to %s\n", sub))
			}
		}

		for _, auto := range s.Automations {
			b.WriteString(fmt.Sprintf("tf %02d pcr %s\n", nextNum, mermaidAutomationLabel(auto, ns)))
			nextNum++
		}

		for _, tr := range s.Translations {
			eid := namespaced(ns, tr.Name)
			b.WriteString(fmt.Sprintf("tf %02d pcr %s\n", nextNum, eid))
			nextNum++
			b.WriteString(fmt.Sprintf("%%   pcr -> cmd\n"))
			if tr.Command != "" && tr.Event != nil && tr.Event.Name != "" {
				b.WriteString(fmt.Sprintf("%%   cmd %s -> evt %s\n", tr.Command, tr.Event.Name))
			}
			if tr.Reads != "" {
				b.WriteString(fmt.Sprintf("%%   reads %s\n", tr.Reads))
			}
		}

		if s.Name != "" {
			b.WriteString("\n")
		}
	}
}

func mermaidAutomationLabel(auto *ast.Automation, ctxName string) string {
	ns := ctxName
	if auto.TargetContext != "" {
		ns = auto.TargetContext
	}
	eid := namespaced(ns, auto.Name)

	activation := auto.OnEvent
	if activation != "" {
		if delay := delayLabel(auto.After); delay != "" {
			activation += " " + delay
		}
	}
	if auto.Schedule != "" {
		activation = cadenceLabel(auto.Schedule)
	}
	if activation == "" || auto.Command == "" {
		return eid
	}

	return fmt.Sprintf("%s (%s → %s)", eid, activation, auto.Command)
}

// exportMermaidProjected renders the tag-key-based projected layout for DCB contexts.
// Events are grouped by tag key with one section per unique tag key.
// Commands, triggers, and untagged events appear in a general cross-cutting section.
// Multi-tag events appear in each matching tag section with a connector annotation.
func exportMermaidProjected(b *strings.Builder, modelName string, entries []sliceEntry, tagKeys []string) {
	// Sort tag keys for deterministic output
	sort.Strings(tagKeys)

	nextNum := 1

	// --- First pass: non-event elements from all slices ---
	b.WriteString("%% Commands / Triggers\n")
	for _, entry := range entries {
		s := entry.slice
		ns := entry.ctxName

		if s.Name != "" {
			b.WriteString(fmt.Sprintf("%% Slice: %s\n", s.Name))
		}

		// Triggers
		if s.Trigger != nil {
			eid := namespaced(ns, s.Trigger.Name)
			b.WriteString(fmt.Sprintf("tf %02d ui %s\n", nextNum, eid))
			nextNum++
		}

		// Commands
		for _, cmd := range s.Commands {
			eid := namespaced(ns, cmd.Name)
			b.WriteString(fmt.Sprintf("tf %02d cmd %s\n", nextNum, eid))
			nextNum++
		}

		// Aggregate events (not tag-grouped)
		if !entry.fromDCB {
			for _, evt := range s.Events {
				eid := namespaced(ns, evt.Name)
				b.WriteString(fmt.Sprintf("tf %02d evt %s\n", nextNum, eid))
				nextNum++
			}
		}

		// DCB events without tags (appear in general section)
		if entry.fromDCB {
			for _, evt := range s.Events {
				if len(evt.Tags) == 0 {
					eid := namespaced(ns, evt.Name)
					b.WriteString(fmt.Sprintf("tf %02d evt %s\n", nextNum, eid))
					nextNum++
				}
			}
		}

		// Views
		for _, view := range s.Views {
			eid := namespaced(ns, view.Name)
			b.WriteString(fmt.Sprintf("tf %02d rmo %s\n", nextNum, eid))
			nextNum++
			for _, sub := range view.Subscribes {
				b.WriteString(fmt.Sprintf("%%   subscribes to %s\n", sub))
			}
		}

		// Automations
		for _, auto := range s.Automations {
			b.WriteString(fmt.Sprintf("tf %02d pcr %s\n", nextNum, mermaidAutomationLabel(auto, ns)))
			nextNum++
		}

		// Translations
		for _, tr := range s.Translations {
			eid := namespaced(ns, tr.Name)
			b.WriteString(fmt.Sprintf("tf %02d pcr %s\n", nextNum, eid))
			nextNum++
			b.WriteString(fmt.Sprintf("%%   pcr -> cmd\n"))
			if tr.Command != "" && tr.Event != nil && tr.Event.Name != "" {
				b.WriteString(fmt.Sprintf("%%   cmd %s -> evt %s\n", tr.Command, tr.Event.Name))
			}
			if tr.Reads != "" {
				b.WriteString(fmt.Sprintf("%%   reads %s\n", tr.Reads))
			}
		}

		if s.Name != "" {
			b.WriteString("\n")
		}
	}

	// --- Second pass: events grouped by tag key ---
	for _, key := range tagKeys {
		b.WriteString(fmt.Sprintf("%% Tag: %s\n", key))
		for _, entry := range entries {
			s := entry.slice
			if !entry.fromDCB {
				continue // aggregate events were rendered in the general section
			}
			for _, evt := range s.Events {
				if !eventHasTag(evt, key) {
					continue
				}
				eid := key + "." + evt.Name
				b.WriteString(fmt.Sprintf("tf %02d evt %s\n", nextNum, eid))
				nextNum++

				// Multi-tag annotation
				if len(evt.Tags) > 1 {
					var otherTags []string
					for _, t := range evt.Tags {
						if t.Key != key {
							otherTags = append(otherTags, t.Key)
						}
					}
					if len(otherTags) > 0 {
						sort.Strings(otherTags)
						b.WriteString(fmt.Sprintf("%%   connector: also in %s\n", strings.Join(otherTags, ", ")))
					}
				}
			}
		}
		b.WriteString("\n")
	}
}

// eventHasTag reports whether an event has a tag with the given key.
func eventHasTag(evt *ast.Event, key string) bool {
	for _, tag := range evt.Tags {
		if tag.Key == key {
			return true
		}
	}
	return false
}

// exportMermaidDCB renders the query-lens (DCB) layout.
// All events appear in a single flat timeline with tag badges.
// Commands show decides_on annotations.
func exportMermaidDCB(b *strings.Builder, modelName string, entries []sliceEntry) {
	nextNum := 1

	b.WriteString("%% Commands / Triggers\n")
	for _, entry := range entries {
		s := entry.slice
		ns := entry.ctxName

		if s.Name != "" {
			b.WriteString(fmt.Sprintf("%% Slice: %s\n", s.Name))
		}

		// Triggers
		if s.Trigger != nil {
			eid := namespaced(ns, s.Trigger.Name)
			b.WriteString(fmt.Sprintf("tf %02d ui %s\n", nextNum, eid))
			nextNum++
		}

		// Commands with decides_on annotation
		for _, cmd := range s.Commands {
			eid := namespaced(ns, cmd.Name)
			b.WriteString(fmt.Sprintf("tf %02d cmd %s\n", nextNum, eid))
			nextNum++
			if cmd.DecidesOn != nil {
				ann := formatDecidesOnAnnotation(cmd.DecidesOn)
				if ann != "" {
					b.WriteString(fmt.Sprintf("%%   %s\n", ann))
				}
			}
		}

		// Views
		for _, view := range s.Views {
			eid := namespaced(ns, view.Name)
			b.WriteString(fmt.Sprintf("tf %02d rmo %s\n", nextNum, eid))
			nextNum++
			for _, sub := range view.Subscribes {
				b.WriteString(fmt.Sprintf("%%   subscribes to %s\n", sub))
			}
		}

		// Automations
		for _, auto := range s.Automations {
			b.WriteString(fmt.Sprintf("tf %02d pcr %s\n", nextNum, mermaidAutomationLabel(auto, ns)))
			nextNum++
		}

		// Translations (reactor part)
		for _, tr := range s.Translations {
			eid := namespaced(ns, tr.Name)
			b.WriteString(fmt.Sprintf("tf %02d pcr %s\n", nextNum, eid))
			nextNum++
			b.WriteString(fmt.Sprintf("%%   pcr -> cmd\n"))
			if tr.Reads != "" {
				b.WriteString(fmt.Sprintf("%%   reads %s\n", tr.Reads))
			}
		}

		if s.Name != "" {
			b.WriteString("\n")
		}
	}

	// --- Events section: flat timeline ---
	b.WriteString("%% Events\n")
	for _, entry := range entries {
		s := entry.slice
		ns := entry.ctxName

		if s.Name != "" {
			b.WriteString(fmt.Sprintf("%% Slice: %s\n", s.Name))
		}

		// Events with tag badges
		for _, evt := range s.Events {
			eid := namespaced(ns, evt.Name)
			b.WriteString(fmt.Sprintf("tf %02d evt %s\n", nextNum, eid))
			nextNum++
			if len(evt.Tags) > 0 {
				tagText := formatEventTagBadges(evt.Tags)
				if tagText != "" {
					b.WriteString(fmt.Sprintf("%%   tags: %s\n", tagText))
				}
			}
		}

		// Translation events
		for _, tr := range s.Translations {
			if tr.Event != nil && tr.Event.Name != "" {
				eid := namespaced(ns, tr.Event.Name)
				b.WriteString(fmt.Sprintf("tf %02d evt %s\n", nextNum, eid))
				nextNum++
			}
		}

		if s.Name != "" {
			b.WriteString("\n")
		}
	}
}

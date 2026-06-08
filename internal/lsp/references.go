package lsp

import (
	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/lexer"
	"github.com/hpcsc/emod/internal/parser"
)

// GetReferences finds all references to the command, event, or view name at the
// given cursor position. If the cursor is not on a resolvable name (definition
// or reference), it returns nil.
//
// Positions are 0-based LSP coordinates (line, character).
func GetReferences(text string, line, character int, uri string) []Location {
	if text == "" {
		return nil
	}

	tokens, _ := lexer.Scan(text, uri)
	p := parser.New(tokens, uri)
	model, _ := p.Parse()
	if model == nil {
		return nil
	}

	// Build definition position maps: name → definition position (1-based AST).
	commandDefs := make(map[string]ast.Position)
	eventDefs := make(map[string]ast.Position)
	viewDefs := make(map[string]ast.Position)

	for _, ctx := range model.Contexts {
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

	// Determine what name the cursor is on and its type.
	var targetName string
	var targetType string // "command", "event", or "view"

	// Check definitions first.
	for name := range commandDefs {
		if cursorOnName(cursorLine, cursorChar, commandDefs[name], name) {
			targetName = name
			targetType = "command"
			break
		}
	}
	if targetName == "" {
		for name := range eventDefs {
			if cursorOnName(cursorLine, cursorChar, eventDefs[name], name) {
				targetName = name
				targetType = "event"
				break
			}
		}
	}
	if targetName == "" {
		for name := range viewDefs {
			if cursorOnName(cursorLine, cursorChar, viewDefs[name], name) {
				targetName = name
				targetType = "view"
				break
			}
		}
	}

	// Check references if no definition hit.
	if targetName == "" {
		for _, ctx := range model.Contexts {
			for _, agg := range ctx.Aggregates {
				for _, slice := range agg.Slices {
					// View subscribes → event references
					for _, v := range slice.Views {
						for i, sub := range v.Subscribes {
							if i < len(v.SubscribesPos) {
								if cursorOnName(cursorLine, cursorChar, v.SubscribesPos[i], sub) {
									if _, ok := eventDefs[sub]; ok {
										targetName = sub
										targetType = "event"
									}
								}
							}
						}
					}

					if targetName != "" {
						break
					}

					// Automation references
					for _, auto := range slice.Automations {
						if cursorOnName(cursorLine, cursorChar, auto.TriggerEventPos, auto.TriggerEvent) {
							if _, ok := eventDefs[auto.TriggerEvent]; ok {
								targetName = auto.TriggerEvent
								targetType = "event"
							}
						}
						if cursorOnName(cursorLine, cursorChar, auto.CommandPos, auto.Command) {
							if _, ok := commandDefs[auto.Command]; ok {
								targetName = auto.Command
								targetType = "command"
							}
						}
					}

					if targetName != "" {
						break
					}

					// Translation references
					for _, tr := range slice.Translations {
						if cursorOnName(cursorLine, cursorChar, tr.ReadsPos, tr.Reads) {
							if _, ok := viewDefs[tr.Reads]; ok {
								targetName = tr.Reads
								targetType = "view"
							}
						}
						if cursorOnName(cursorLine, cursorChar, tr.CommandPos, tr.Command) {
							if _, ok := commandDefs[tr.Command]; ok {
								targetName = tr.Command
								targetType = "command"
							}
						}
					}

					if targetName != "" {
						break
					}

					// Trigger reads → view references
					if slice.Trigger != nil {
						if cursorOnName(cursorLine, cursorChar, slice.Trigger.ReadsPos, slice.Trigger.Reads) {
							if _, ok := viewDefs[slice.Trigger.Reads]; ok {
								targetName = slice.Trigger.Reads
								targetType = "view"
							}
						}
					}

					if targetName != "" {
						break
					}

					// Flow references
					for _, f := range slice.Flows {
						if cursorOnName(cursorLine, cursorChar, f.CommandPos, f.CommandName) {
							if _, ok := commandDefs[f.CommandName]; ok {
								targetName = f.CommandName
								targetType = "command"
							}
						}
						if cursorOnName(cursorLine, cursorChar, f.EventPos, f.EventName) {
							if _, ok := eventDefs[f.EventName]; ok {
								targetName = f.EventName
								targetType = "event"
							}
						}
					}
				}
			}
		}
	}

	if targetName == "" {
		return nil
	}

	// Collect all locations for the resolved name.
	var locations []Location

	// Add definition location.
	switch targetType {
	case "command":
		if pos, ok := commandDefs[targetName]; ok {
			locations = append(locations, *locationFor(uri, pos, targetName))
		}
	case "event":
		if pos, ok := eventDefs[targetName]; ok {
			locations = append(locations, *locationFor(uri, pos, targetName))
		}
	case "view":
		if pos, ok := viewDefs[targetName]; ok {
			locations = append(locations, *locationFor(uri, pos, targetName))
		}
	}

	// Walk all references in the model.
	for _, ctx := range model.Contexts {
		for _, agg := range ctx.Aggregates {
			for _, slice := range agg.Slices {
				// Subscribes → event references
				if targetType == "event" {
					for _, v := range slice.Views {
						for i, sub := range v.Subscribes {
							if sub == targetName && i < len(v.SubscribesPos) {
								locations = append(locations, *locationFor(uri, v.SubscribesPos[i], sub))
							}
						}
					}
				}

				// Automation references
				for _, auto := range slice.Automations {
					if targetType == "event" && auto.TriggerEvent == targetName {
						locations = append(locations, *locationFor(uri, auto.TriggerEventPos, auto.TriggerEvent))
					}
					if targetType == "command" && auto.Command == targetName {
						locations = append(locations, *locationFor(uri, auto.CommandPos, auto.Command))
					}
				}

				// Translation references
				for _, tr := range slice.Translations {
					if targetType == "view" && tr.Reads == targetName {
						locations = append(locations, *locationFor(uri, tr.ReadsPos, tr.Reads))
					}
					if targetType == "command" && tr.Command == targetName {
						locations = append(locations, *locationFor(uri, tr.CommandPos, tr.Command))
					}
				}

				// Trigger reads → view references
				if slice.Trigger != nil {
					if targetType == "view" && slice.Trigger.Reads == targetName {
						locations = append(locations, *locationFor(uri, slice.Trigger.ReadsPos, slice.Trigger.Reads))
					}
				}

				// Flow references
				for _, f := range slice.Flows {
					if targetType == "command" && f.CommandName == targetName {
						locations = append(locations, *locationFor(uri, f.CommandPos, f.CommandName))
					}
					if targetType == "event" && f.EventName == targetName {
						locations = append(locations, *locationFor(uri, f.EventPos, f.EventName))
					}
				}
			}
		}
	}

	return locations
}

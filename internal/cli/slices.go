package cli

import (
	"encoding/json"
	"fmt"

	"github.com/hpcsc/emod/internal/ast"
)

type sliceJSONEntry struct {
	Name        string `json:"name"`
	Pattern     string `json:"pattern"`
	Context     string `json:"context"`
	KeyElements string `json:"keyElements"`
}

func RunSlices(path, format string) error {
	if format != "text" && format != "json" {
		return &LintError{
			Message:  fmt.Sprintf("unsupported format %q; supported formats: text, json", format),
			ExitCode: 1,
			Cause:    ErrUnsupportedFormat,
		}
	}

	model, err := parseModelFile("slices", path)
	if err != nil {
		return err
	}

	entries := model.SliceRefs()

	if format == "json" {
		return formatSlicesJSON(entries)
	}

	if len(entries) == 0 {
		fmt.Println("No slices found.")
		return nil
	}

	type row struct {
		sliceName   string
		pattern     string
		ctxName     string
		keyElements string
	}

	rows := make([]row, 0, len(entries))
	maxSlice := len("SLICE")
	maxPattern := len("PATTERN")
	maxContext := len("CONTEXT")
	for _, e := range entries {
		pattern := detectPattern(e.Slice)
		keyElements := keyElementsForPattern(e.Slice, pattern)
		r := row{
			sliceName:   e.Slice.Name,
			pattern:     pattern,
			ctxName:     e.Context.Name,
			keyElements: keyElements,
		}
		rows = append(rows, r)
		if len(r.sliceName) > maxSlice {
			maxSlice = len(r.sliceName)
		}
		if len(r.pattern) > maxPattern {
			maxPattern = len(r.pattern)
		}
		if len(r.ctxName) > maxContext {
			maxContext = len(r.ctxName)
		}
	}

	// Print header
	fmt.Printf("%-*s  %-*s  %-*s  %s\n", maxSlice, "SLICE", maxPattern, "PATTERN", maxContext, "CONTEXT", "KEY ELEMENTS")

	// Print rows
	for _, r := range rows {
		fmt.Printf("%-*s  %-*s  %-*s  %s\n", maxSlice, r.sliceName, maxPattern, r.pattern, maxContext, r.ctxName, r.keyElements)
	}

	return nil
}

func formatSlicesJSON(entries []ast.SliceRef) error {
	jsonEntries := make([]sliceJSONEntry, 0, len(entries))
	for _, e := range entries {
		pattern := detectPattern(e.Slice)
		keyElements := keyElementsForPattern(e.Slice, pattern)
		jsonEntries = append(jsonEntries, sliceJSONEntry{
			Name:        e.Slice.Name,
			Pattern:     pattern,
			Context:     e.Context.Name,
			KeyElements: keyElements,
		})
	}

	out, err := json.Marshal(jsonEntries)
	if err != nil {
		return &LintError{
			Message:  fmt.Sprintf("json encoding: %s", err),
			ExitCode: 1,
		}
	}
	fmt.Println(string(out))
	return nil
}

func detectPattern(s *ast.Slice) string {
	if len(s.Translations) > 0 {
		return "translation"
	}
	if len(s.Automations) > 0 {
		return "automation"
	}
	if len(s.Views) > 0 {
		return "view"
	}
	if s.Trigger != nil && len(s.Commands) > 0 && (len(s.Events) > 0 || len(s.Flows) > 0) {
		return "command"
	}
	return "unknown"
}

func automationActivation(auto *ast.Automation) string {
	if auto.Schedule != "" {
		return `every "` + auto.Schedule + `"`
	}
	return auto.OnEvent
}

func keyElementsForPattern(s *ast.Slice, pattern string) string {
	switch pattern {
	case "command":
		if len(s.Flows) > 0 {
			return fmt.Sprintf("%s, %s", s.Flows[0].CommandName, s.Flows[0].EventName)
		}
		cmd := ""
		if len(s.Commands) > 0 {
			cmd = s.Commands[0].Name
		}
		evt := ""
		if len(s.Events) > 0 {
			evt = s.Events[0].Name
		}
		if cmd != "" && evt != "" {
			return fmt.Sprintf("%s, %s", cmd, evt)
		}
		if cmd != "" {
			return cmd
		}
		return evt
	case "view":
		if len(s.Views) > 0 {
			return s.Views[0].Name
		}
		return ""
	case "automation":
		if len(s.Automations) > 0 {
			auto := s.Automations[0]
			return fmt.Sprintf("%s, %s", automationActivation(auto), auto.Command)
		}
		return ""
	case "translation":
		if len(s.Translations) > 0 {
			tr := s.Translations[0]
			evt := ""
			if tr.Event != nil {
				evt = tr.Event.Name
			}
			if tr.Command != "" && evt != "" {
				return fmt.Sprintf("%s, %s", tr.Command, evt)
			}
			if tr.Command != "" {
				return tr.Command
			}
			return evt
		}
		return ""
	default:
		return ""
	}
}

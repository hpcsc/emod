package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/lexer"
	"github.com/hpcsc/emod/internal/parser"
)

func RunSlices(path string) error {
	if path == "" {
		return &LintError{
			Message:  "slices requires exactly one file argument",
			ExitCode: 1,
		}
	}

	source, err := os.ReadFile(path)
	if err != nil {
		return &LintError{
			Message:  fmt.Sprintf("reading %s: %s", path, err),
			ExitCode: 1,
		}
	}

	tokens, diagnostics := lexer.Scan(string(source), path)

	p := parser.New(tokens, path)
	model, parserDiags := p.Parse()
	diagnostics = append(diagnostics, parserDiags...)

	if len(diagnostics) > 0 {
		var sb strings.Builder
		for _, d := range diagnostics {
			fmt.Fprintln(&sb, d.String())
		}
		return &LintError{
			Message:  strings.TrimRight(sb.String(), "\n"),
			ExitCode: 1,
		}
	}

	entries := collectSliceEntries(model)
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
		pattern := detectPattern(e.slice)
		keyElements := keyElementsForPattern(e.slice, pattern)
		r := row{
			sliceName:   e.slice.Name,
			pattern:     pattern,
			ctxName:     e.ctxName,
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

type sliceEntry struct {
	slice   *ast.Slice
	ctxName string
}

func collectSliceEntries(model *ast.Model) []sliceEntry {
	var entries []sliceEntry
	if model == nil {
		return entries
	}
	for _, ctx := range model.Contexts {
		for _, agg := range ctx.Aggregates {
			for _, s := range agg.Slices {
				entries = append(entries, sliceEntry{slice: s, ctxName: ctx.Name})
			}
		}
	}
	return entries
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
			return fmt.Sprintf("%s, %s", auto.TriggerEvent, auto.Command)
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

package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/hpcsc/emod/internal/diagnostic"
	"github.com/hpcsc/emod/internal/lexer"
	"github.com/hpcsc/emod/internal/linter"
	"github.com/hpcsc/emod/internal/parser"
)

type LintError struct {
	Message  string
	ExitCode int
}

func (e *LintError) Error() string {
	return e.Message
}

type jsonEntry struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Rule     string `json:"rule,omitempty"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

func formatJSON(diagnostics []*diagnostic.Entry) error {
	entries := make([]jsonEntry, 0, len(diagnostics))
	hasErrors := false
	for _, d := range diagnostics {
		if d.Severity == diagnostic.Error {
			hasErrors = true
		}
		entries = append(entries, jsonEntry{
			File:     d.Filename,
			Line:     d.Line,
			Rule:     d.RuleName,
			Severity: d.Severity.String(),
			Message:  d.Message,
		})
	}
	out, err := json.Marshal(entries)
	if err != nil {
		return &LintError{
			Message:  fmt.Sprintf("json encoding: %s", err),
			ExitCode: 1,
		}
	}
	fmt.Println(string(out))
	if len(diagnostics) == 0 {
		return nil
	}
	exitCode := 1
	if hasErrors {
		exitCode = 2
	}
	return &LintError{
		Message:  "",
		ExitCode: exitCode,
	}
}

func formatText(diagnostics []*diagnostic.Entry) error {
	var sb strings.Builder
	for _, d := range diagnostics {
		fmt.Fprintln(&sb, d.String())
	}
	return &LintError{
		Message:  strings.TrimRight(sb.String(), "\n"),
		ExitCode: 1,
	}
}

func RunLintExplain(ruleName string) error {
	desc, ok := linter.RuleDescription(ruleName)
	if !ok {
		return &LintError{
			Message:  fmt.Sprintf("unknown rule %q", ruleName),
			ExitCode: 1,
		}
	}
	fmt.Println(desc)
	return nil
}

func RunLint(path, format string) error {
	if format != "text" && format != "json" {
		return &LintError{
			Message:  fmt.Sprintf("unsupported format %q; supported formats: text, json", format),
			ExitCode: 1,
		}
	}

	if path == "" {
		return &LintError{
			Message:  "lint requires exactly one file argument",
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

	if len(diagnostics) == 0 {
		diagnostics = linter.Lint(model)
	}

	if format == "json" {
		return formatJSON(diagnostics)
	}

	if len(diagnostics) > 0 {
		return formatText(diagnostics)
	}

	return nil
}

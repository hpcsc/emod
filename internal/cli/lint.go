package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/hpcsc/emod/internal/diagnostic"
	"github.com/hpcsc/emod/internal/linter"
	"github.com/hpcsc/emod/internal/oracle"
)

// Commands report these conditions so callers can branch on the cause rather
// than on the wording of the message shown to the user.
var (
	ErrMissingFileArgument = errors.New("requires exactly one file argument")
	ErrUnsupportedFormat   = errors.New("unsupported format")
	ErrUnknownRule         = errors.New("unknown rule")
)

type LintError struct {
	Message  string
	ExitCode int
	Cause    error
}

func (e *LintError) Error() string {
	return e.Message
}

func (e *LintError) Unwrap() error {
	return e.Cause
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
			Cause:    ErrUnknownRule,
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
			Cause:    ErrUnsupportedFormat,
		}
	}

	source, err := readSourceFile("lint", path)
	if err != nil {
		return err
	}

	diagnostics := oracle.Check(source, path)

	if format == "json" {
		return formatJSON(diagnostics)
	}

	if len(diagnostics) > 0 {
		return formatText(diagnostics)
	}

	return nil
}

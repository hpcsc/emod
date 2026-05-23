package cli

import (
	"fmt"
	"os"

	"github.com/hpcsc/emod/internal/diagnostic"
	"github.com/hpcsc/emod/internal/export"
	"github.com/hpcsc/emod/internal/lexer"
	"github.com/hpcsc/emod/internal/linter"
	"github.com/hpcsc/emod/internal/parser"
	"github.com/hpcsc/emod/internal/validator"
)

// RunExport reads the file at path, lexes and parses it, validates and lints,
// and if clean outputs the full model to stdout in the requested format.
// If diagnostics exist, they are written to stderr and a non-zero exit code is returned.
// The format parameter controls output format — "json" and "cue" are supported.
func RunExport(path, format string) error {
	if format != "json" && format != "cue" {
		return &LintError{
			Message:  fmt.Sprintf("unsupported format %q; supported formats: json, cue", format),
			ExitCode: 1,
		}
	}

	if path == "" {
		return &LintError{
			Message:  "export requires exactly one file argument",
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

	validatorDiags := validator.Validate(model)
	diagnostics = append(diagnostics, validatorDiags...)

	lintDiags := linter.Lint(model)
	diagnostics = append(diagnostics, lintDiags...)

	if len(diagnostics) > 0 {
		// Diagnostics go to stderr in text format with non-zero exit
		hasErrors := false
		for _, d := range diagnostics {
			if d.Severity == diagnostic.Error {
				hasErrors = true
			}
			fmt.Fprintln(os.Stderr, d.String())
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

	var output []byte
	switch format {
	case "cue":
		output, err = export.ExportCUE(model)
	default:
		output, err = export.ExportJSON(model)
	}
	if err != nil {
		return &LintError{
			Message:  fmt.Sprintf("export encoding: %s", err),
			ExitCode: 1,
		}
	}
	fmt.Println(string(output))
	return nil
}

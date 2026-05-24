package cli

import (
	"fmt"
	"os"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/diagnostic"
	"github.com/hpcsc/emod/internal/export"
	"github.com/hpcsc/emod/internal/lexer"
	"github.com/hpcsc/emod/internal/linter"
	"github.com/hpcsc/emod/internal/parser"
	"github.com/hpcsc/emod/internal/validator"
)

// RunExport reads the file at path, lexes and parses it, validates and lints,
// and outputs the model in the requested format.
// For JSON format, the output is always a complete JSON document on stdout
// wrapped with diagnostics, with exit code based on severity.
// For CUE format, diagnostics are written to stderr with non-zero exit.
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

	tokens, diags := lexer.Scan(string(source), path)

	p := parser.New(tokens, path)
	model, parserDiags := p.Parse()
	diags = append(diags, parserDiags...)

	validatorDiags := validator.Validate(model)
	diags = append(diags, validatorDiags...)

	lintDiags := linter.Lint(model)
	diags = append(diags, lintDiags...)

	switch format {
	case "json":
		return handleJSONExport(model, diags)
	default:
		return handleCUEExport(model, diags)
	}
}

// exitCodeForDiagnostics derives the exit code from a diagnostics slice:
// 0 for no diagnostics, 1 for warnings only, 2 if any error is present.
func exitCodeForDiagnostics(diagnostics []*diagnostic.Entry) int {
	if len(diagnostics) == 0 {
		return 0
	}
	for _, d := range diagnostics {
		if d.Severity == diagnostic.Error {
			return 2
		}
	}
	return 1
}

// handleJSONExport always outputs a complete JSON document to stdout with
// diagnostics included, and returns a LintError with exit code based on severity.
func handleJSONExport(model *ast.Model, diagnostics []*diagnostic.Entry) error {
	output, err := export.ExportJSONDiagnostics(model, diagnostics)
	if err != nil {
		return &LintError{
			Message:  fmt.Sprintf("export encoding: %s", err),
			ExitCode: 1,
		}
	}
	fmt.Println(string(output))

	exitCode := exitCodeForDiagnostics(diagnostics)
	if exitCode > 0 {
		return &LintError{
			Message:  "",
			ExitCode: exitCode,
		}
	}
	return nil
}

// handleCUEExport outputs CUE text to stdout if clean, or writes diagnostics
// to stderr and returns a non-zero exit code.
func handleCUEExport(model *ast.Model, diagnostics []*diagnostic.Entry) error {
	if len(diagnostics) > 0 {
		for _, d := range diagnostics {
			fmt.Fprintln(os.Stderr, d.String())
		}
		return &LintError{
			Message:  "",
			ExitCode: exitCodeForDiagnostics(diagnostics),
		}
	}

	output, err := export.ExportCUE(model)
	if err != nil {
		return &LintError{
			Message:  fmt.Sprintf("export encoding: %s", err),
			ExitCode: 1,
		}
	}
	fmt.Println(string(output))
	return nil
}



package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hpcsc/emod/internal/diagnostic"
	"github.com/hpcsc/emod/internal/diagram"
	"github.com/hpcsc/emod/internal/lexer"
	"github.com/hpcsc/emod/internal/linter"
	"github.com/hpcsc/emod/internal/parser"
	"github.com/hpcsc/emod/internal/validator"
)

// RunDiagram reads the file at path, lexes and parses it, validates and lints,
// generates a draw.io diagram, and writes it to a .drawio file.
// If outputPath is empty, it defaults to the input filename with .drawio extension.
// Errors produce diagnostics on stderr and a non-zero exit code.
// Lint warnings still produce the diagram but with exit code 1.
func RunDiagram(path, outputPath string) error {
	if path == "" {
		return &LintError{
			Message:  "diagram requires exactly one file argument",
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

	// Check for errors that prevent diagram generation
	hasErrors := false
	hasWarnings := false
	for _, d := range diagnostics {
		if d.Severity == diagnostic.Error {
			hasErrors = true
		} else {
			hasWarnings = true
		}
		fmt.Fprintln(os.Stderr, d.String())
	}

	// Errors: no diagram output, exit code 2
	if hasErrors {
		return &LintError{
			Message:  "",
			ExitCode: 2,
		}
	}

	// Generate diagram
	xmlBytes, err := diagram.ExportDrawio(model)
	if err != nil {
		return &LintError{
			Message:  fmt.Sprintf("diagram generation: %s", err),
			ExitCode: 1,
		}
	}

	if outputPath == "" {
		outputPath = defaultDrawioPath(path)
	}

	// Ensure directory exists
	dir := filepath.Dir(outputPath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return &LintError{
				Message:  fmt.Sprintf("creating directory %s: %s", dir, err),
				ExitCode: 1,
			}
		}
	}

	if err := os.WriteFile(outputPath, xmlBytes, 0o644); err != nil {
		return &LintError{
			Message:  fmt.Sprintf("writing %s: %s", outputPath, err),
			ExitCode: 1,
		}
	}

	// Warnings: diagram written, but non-zero exit
	if hasWarnings {
		return &LintError{
			Message:  "",
			ExitCode: 1,
		}
	}

	return nil
}

// defaultDrawioPath replaces the .emod extension with .drawio.
// If the file has no .emod extension, .drawio is appended.
func defaultDrawioPath(path string) string {
	if strings.HasSuffix(path, ".emod") {
		return path[:len(path)-len(".emod")] + ".drawio"
	}
	return path + ".drawio"
}

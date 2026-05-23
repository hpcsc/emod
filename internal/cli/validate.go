package cli

import (
	"fmt"
	"os"

	"github.com/hpcsc/emod/internal/lexer"
	"github.com/hpcsc/emod/internal/linter"
	"github.com/hpcsc/emod/internal/parser"
	"github.com/hpcsc/emod/internal/validator"
)

// RunValidate reads the file at path, lexes and parses it, and returns an error
// if there are any diagnostics. The format parameter controls output: "text"
// for human-readable diagnostics (default) or "json" for structured output.
// An empty path is treated as a missing argument.
func RunValidate(path, format string) error {
	if format != "text" && format != "json" {
		return &LintError{
			Message:  fmt.Sprintf("unsupported format %q; supported formats: text, json", format),
			ExitCode: 1,
		}
	}

	if path == "" {
		return &LintError{
			Message:  "validate requires exactly one file argument",
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

	if format == "json" {
		return formatJSON(diagnostics)
	}

	if len(diagnostics) > 0 {
		return formatText(diagnostics)
	}

	return nil
}

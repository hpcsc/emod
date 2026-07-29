package cli

import (
	"fmt"
	"os"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/lexer"
	"github.com/hpcsc/emod/internal/parser"
)

func parseModelFile(command, path string) (*ast.Model, error) {
	if path == "" {
		return nil, &LintError{
			Message:  fmt.Sprintf("%s requires exactly one file argument", command),
			ExitCode: 1,
			Cause:    ErrMissingFileArgument,
		}
	}

	source, err := os.ReadFile(path)
	if err != nil {
		return nil, &LintError{
			Message:  fmt.Sprintf("reading %s: %s", path, err),
			ExitCode: 1,
		}
	}

	tokens, diagnostics := lexer.Scan(string(source), path)

	model, parserDiags := parser.New(tokens, path).Parse()
	diagnostics = append(diagnostics, parserDiags...)

	if len(diagnostics) > 0 {
		return nil, formatText(diagnostics)
	}

	return model, nil
}

package cli

import (
	"fmt"
	"os"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/oracle"
)

// readSourceFile reads the model source at path, reporting a missing argument
// or read failure as a LintError named after the command.
func readSourceFile(command, path string) (string, error) {
	if path == "" {
		return "", &LintError{
			Message:  fmt.Sprintf("%s requires exactly one file argument", command),
			ExitCode: 1,
			Cause:    ErrMissingFileArgument,
		}
	}

	source, err := os.ReadFile(path)
	if err != nil {
		return "", &LintError{
			Message:  fmt.Sprintf("reading %s: %s", path, err),
			ExitCode: 1,
		}
	}

	return string(source), nil
}

func parseModelFile(command, path string) (*ast.Model, error) {
	source, err := readSourceFile(command, path)
	if err != nil {
		return nil, err
	}

	model, diagnostics := oracle.Parse(source, path)
	if len(diagnostics) > 0 {
		return nil, formatText(diagnostics)
	}

	return model, nil
}

package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/hpcsc/emod/internal/formatter"
	"github.com/hpcsc/emod/internal/oracle"
	"github.com/hpcsc/emod/internal/parser"
)

// RunFmt formats the file in the syntax it is written in. hcl converts a file
// written in the emod grammar to HCL; a file already in HCL stays in HCL.
func RunFmt(path string, check bool, hcl bool) error {
	if path == "" {
		return fmt.Errorf("fmt %w", ErrMissingFileArgument)
	}

	source, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}

	model, diagnostics := oracle.Parse(string(source), path)

	if len(diagnostics) > 0 {
		var sb strings.Builder
		for _, d := range diagnostics {
			fmt.Fprintln(&sb, d.String())
		}
		return errors.New(strings.TrimRight(sb.String(), "\n"))
	}

	formatted := formatter.Format(model)
	if hcl || parser.IsHCL(string(source)) {
		formatted = formatter.FormatHCL(model)
	}

	if check {
		if formatted != string(source) {
			return fmt.Errorf("%s is not formatted", path)
		}
		return nil
	}

	if formatted == string(source) {
		return nil
	}

	return os.WriteFile(path, []byte(formatted), 0o644)
}

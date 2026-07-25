package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/hpcsc/emod/internal/formatter"
	"github.com/hpcsc/emod/internal/lexer"
	"github.com/hpcsc/emod/internal/parser"
)

func RunFmt(path string, check bool) error {
	if path == "" {
		return fmt.Errorf("fmt %w", ErrMissingFileArgument)
	}

	source, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
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
		return errors.New(strings.TrimRight(sb.String(), "\n"))
	}

	formatted := formatter.Format(model)

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

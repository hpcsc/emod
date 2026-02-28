package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/hpcsc/emod/internal/diagnostic"
	"github.com/hpcsc/emod/internal/lexer"
	"github.com/hpcsc/emod/internal/parser"
)

// RunValidate reads the file at path, lexes and parses it, and returns an error
// if there are any diagnostics. An empty path is treated as a missing argument.
func RunValidate(path string) error {
	if path == "" {
		return errors.New("validate requires exactly one file argument")
	}

	source, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}

	l := lexer.New(string(source))
	tokens := l.Tokenize()

	var diagnostics []*diagnostic.Diagnostic
	for _, errTok := range l.Errors() {
		diagnostics = append(diagnostics, &diagnostic.Diagnostic{
			Filename: path,
			Line:     errTok.Line,
			Column:   errTok.Column,
			Message:  errTok.Value,
		})
	}

	p := parser.New(tokens, path)
	_, parserDiags := p.Parse()
	diagnostics = append(diagnostics, parserDiags...)

	if len(diagnostics) > 0 {
		var sb strings.Builder
		for _, d := range diagnostics {
			fmt.Fprintf(&sb, "%s:%d: %s\n", d.Filename, d.Line, d.Message)
		}
		return errors.New(strings.TrimRight(sb.String(), "\n"))
	}

	return nil
}

package oracle

import (
	"github.com/hpcsc/emod/internal/diagnostic"
	"github.com/hpcsc/emod/internal/lexer"
	"github.com/hpcsc/emod/internal/linter"
	"github.com/hpcsc/emod/internal/parser"
	"github.com/hpcsc/emod/internal/validator"
)

// Check returns the combined diagnostics from the lex/parse/validate/lint chain.
// A clean source yields a length-zero slice. Check never reads from disk or
// performs any I/O — the caller supplies the source text and filename.
func Check(source string, filename string) []*diagnostic.Entry {
	tokens, diagnostics := lexer.Scan(source, filename)

	p := parser.New(tokens, filename)
	model, parserDiags := p.Parse()
	diagnostics = append(diagnostics, parserDiags...)
	diagnostics = append(diagnostics, validator.Validate(model)...)
	diagnostics = append(diagnostics, linter.Lint(model)...)

	return diagnostics
}

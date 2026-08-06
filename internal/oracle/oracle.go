// Package oracle wires the pipeline stages together so every frontend — CLI,
// LSP, wasm — runs the same chain and reports the same diagnostics.
package oracle

import (
	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/diagnostic"
	"github.com/hpcsc/emod/internal/lexer"
	"github.com/hpcsc/emod/internal/linter"
	"github.com/hpcsc/emod/internal/parser"
	"github.com/hpcsc/emod/internal/validator"
)

// Parse runs the lex/parse chain and returns the model alongside any
// diagnostics. The model is best-effort: it is non-nil even when diagnostics
// are present, holding whatever the parser could recover.
func Parse(source string, filename string) (*ast.Model, []*diagnostic.Entry) {
	tokens, diagnostics := lexer.Scan(source, filename)
	model, parserDiags := parser.New(tokens, filename).Parse()
	return model, append(diagnostics, parserDiags...)
}

// Run runs the full lex/parse/validate/lint chain and returns the model
// alongside the combined diagnostics. Run never reads from disk or performs
// any I/O — the caller supplies the source text and filename.
func Run(source string, filename string) (*ast.Model, []*diagnostic.Entry) {
	model, diagnostics := Parse(source, filename)
	diagnostics = append(diagnostics, validator.Validate(model)...)
	diagnostics = append(diagnostics, linter.Lint(model)...)
	return model, diagnostics
}

// Check returns the combined diagnostics from the lex/parse/validate/lint
// chain. A clean source yields a length-zero slice.
func Check(source string, filename string) []*diagnostic.Entry {
	_, diagnostics := Run(source, filename)
	return diagnostics
}

// Package oracle wires the pipeline stages together so every frontend — CLI,
// LSP, wasm — runs the same chain and reports the same diagnostics. The CLI's
// lint and validate also run the rules that read the disk next to the file.
package oracle

import (
	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/diagnostic"
	"github.com/hpcsc/emod/internal/linter"
	"github.com/hpcsc/emod/internal/parser"
	"github.com/hpcsc/emod/internal/validator"
)

// Parse reads the source and returns the model alongside any diagnostics. The
// model is best-effort: it is non-nil even when diagnostics are present,
// holding whatever the parser could recover.
func Parse(source string, filename string) (*ast.Model, []*diagnostic.Entry) {
	return parser.Parse(source, filename)
}

// Run runs the full lex/parse/validate/lint chain and returns the model
// alongside the combined diagnostics. Run never reads from disk or performs
// any I/O — the caller supplies the source text and filename.
func Run(source string, filename string) (*ast.Model, []*diagnostic.Entry) {
	model, diagnostics, _ := RunParsed(source, filename)
	return model, diagnostics
}

// RunParsed runs the chain as Run does, and also reports whether the source
// parsed: whether lexing and parsing reported nothing.
func RunParsed(source string, filename string) (model *ast.Model, diagnostics []*diagnostic.Entry, parsed bool) {
	return run(source, filename, linter.Lint)
}

func run(source string, filename string, lint func(*ast.Model) []*diagnostic.Entry) (model *ast.Model, diagnostics []*diagnostic.Entry, parsed bool) {
	model, diagnostics = Parse(source, filename)
	parsed = len(diagnostics) == 0
	diagnostics = append(diagnostics, validator.Validate(model)...)
	diagnostics = append(diagnostics, lint(model)...)
	return model, diagnostics, parsed
}

// Check returns the combined diagnostics from the lex/parse/validate/lint
// chain. A clean source yields a length-zero slice.
func Check(source string, filename string) []*diagnostic.Entry {
	_, diagnostics := Run(source, filename)
	return diagnostics
}

// CheckFile is Check for a source read from the file at path. Unlike the rest of
// the chain, it reads the disk: it runs the rules that look next to the file.
func CheckFile(source string, path string) []*diagnostic.Entry {
	_, diagnostics, _ := run(source, path, func(model *ast.Model) []*diagnostic.Entry {
		return linter.LintFile(model, path)
	})
	return diagnostics
}

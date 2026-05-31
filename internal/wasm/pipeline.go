// Package wasm extracts the emod pipeline (lex → parse → validate → lint → export)
// into functions that accept and return standard Go types, enabling testing
// independent of syscall/js.
package wasm

import (
	"encoding/json"
	"fmt"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/diagnostic"
	"github.com/hpcsc/emod/internal/export"
	"github.com/hpcsc/emod/internal/lexer"
	"github.com/hpcsc/emod/internal/linter"
	"github.com/hpcsc/emod/internal/parser"
	"github.com/hpcsc/emod/internal/validator"
)

// exportFunc is a function that serializes a model and diagnostics into JSON.
type exportFunc func(*ast.Model, []*diagnostic.Entry) ([]byte, error)

// ExtractSource parses the input JSON and returns the source field.
func ExtractSource(input string) (string, error) {
	var req struct {
		Source string `json:"source"`
	}
	if err := json.Unmarshal([]byte(input), &req); err != nil {
		return "", fmt.Errorf("invalid JSON: %v", err)
	}
	if req.Source == "" {
		return "", fmt.Errorf("missing source field")
	}
	return req.Source, nil
}

// runPipeline runs the full emod pipeline (lex → parse → validate → lint)
// and invokes the given export function on the result.
func runPipeline(source string, fn exportFunc) (result []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("pipeline panic: %v", r)
		}
	}()

	tokens, diags := lexer.Scan(source, "input.emod")
	p := parser.New(tokens, "input.emod")
	model, parserDiags := p.Parse()
	diags = append(diags, parserDiags...)
	diags = append(diags, validator.Validate(model)...)
	diags = append(diags, linter.Lint(model)...)

	return fn(model, diags)
}

// RunPipelineExportDiagram runs the pipeline and wraps the result
// in the diagram JSON diagnostics envelope { diagnostics, diagram }.
func RunPipelineExportDiagram(source string) ([]byte, error) {
	return runPipeline(source, export.ExportDiagramJSONDiagnostics)
}

// RunPipelineExportJSON runs the pipeline and wraps the result
// in the model JSON diagnostics envelope { diagnostics, model }.
func RunPipelineExportJSON(source string) ([]byte, error) {
	return runPipeline(source, export.ExportJSONDiagnostics)
}

// ErrorJSON returns a JSON error string in the form {"error": "..."}.
func ErrorJSON(msg string) string {
	b, _ := json.Marshal(map[string]string{"error": msg})
	return string(b)
}

// Package pipeline runs the emod pipeline (lex → parse → validate → lint → export)
// behind functions that accept and return standard Go types, so every surface that
// drives a model — the browser shim, the desktop shell, a test — shares one
// orchestration and none of them imposes its transport on it.
package pipeline

import (
	"encoding/json"
	"fmt"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/diagnostic"
	"github.com/hpcsc/emod/internal/export"
	"github.com/hpcsc/emod/internal/formatter"
	"github.com/hpcsc/emod/internal/importer"
	"github.com/hpcsc/emod/internal/oracle"
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

	model, diags := oracle.Run(source, "input.emod")

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

// ExportEmod converts a diagram JSON document — the {model_name, nodes, edges}
// shape the viewer holds and edits — into formatted .emod text, so the viewer's
// export button and `emod fmt` share one writer.
func ExportEmod(diagramJSON string) (result []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("export panic: %v", r)
		}
	}()

	model, err := importer.ImportDiagram([]byte(diagramJSON))
	if err != nil {
		return nil, err
	}

	return []byte(formatter.Format(model)), nil
}

// ExportEmodJSON wraps ExportEmod in a {"emod": "..."} envelope, mirroring the
// {"error": "..."} shape ErrorJSON produces, so a JS caller can tell the two
// apart without guessing at the payload.
func ExportEmodJSON(diagramJSON string) string {
	result, err := ExportEmod(diagramJSON)
	if err != nil {
		return ErrorJSON(err.Error())
	}

	b, err := json.Marshal(map[string]string{"emod": string(result)})
	if err != nil {
		return ErrorJSON(err.Error())
	}

	return string(b)
}

// ErrorJSON returns a JSON error string in the form {"error": "..."}.
func ErrorJSON(msg string) string {
	b, _ := json.Marshal(map[string]string{"error": msg})
	return string(b)
}

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

// diagramAnswer is what the viewer runtimes read back from a parse: the
// document `emod export --format diagram-json` prints, and whether the source
// parsed.
type diagramAnswer struct {
	Diagnostics json.RawMessage `json:"diagnostics"`
	Diagram     json.RawMessage `json:"diagram"`
	Parsed      bool            `json:"parsed"`
}

// Request is the envelope both viewer runtimes hand across their boundary: the
// source to run, and the name of the file it came from for the diagnostics and
// node positions to report.
type Request struct {
	Source   string `json:"source"`
	Filename string `json:"filename"`
}

const pastedSourceFilename = "input.emod"

// ExtractRequest parses the input JSON into a Request.
func ExtractRequest(input string) (Request, error) {
	var req Request
	if err := json.Unmarshal([]byte(input), &req); err != nil {
		return Request{}, fmt.Errorf("invalid JSON: %v", err)
	}
	if req.Source == "" {
		return Request{}, fmt.Errorf("missing source field")
	}
	if req.Filename == "" {
		req.Filename = pastedSourceFilename
	}
	return req, nil
}

// runPipeline runs the full emod pipeline (lex → parse → validate → lint)
// and invokes the given export function on the result, answering beside it
// whether the source parsed.
func runPipeline(source, filename string, fn exportFunc) (result []byte, parsed bool, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("pipeline panic: %v", r)
		}
	}()

	model, diags, parsed := oracle.RunParsed(source, filename)
	result, err = fn(model, diags)

	return result, parsed, err
}

// RunPipelineExportDiagram runs the pipeline and answers the viewer's
// { diagnostics, diagram, parsed } document.
func RunPipelineExportDiagram(source, filename string) ([]byte, error) {
	document, parsed, err := runPipeline(source, filename, export.ExportDiagramJSONDiagnostics)
	if err != nil {
		return nil, err
	}

	var answer diagramAnswer
	if err := json.Unmarshal(document, &answer); err != nil {
		return nil, err
	}
	answer.Parsed = parsed

	return json.Marshal(answer)
}

// RunPipelineExportJSON runs the pipeline and wraps the result
// in the model JSON diagnostics envelope { diagnostics, model }.
func RunPipelineExportJSON(source, filename string) ([]byte, error) {
	document, _, err := runPipeline(source, filename, export.ExportJSONDiagnostics)

	return document, err
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

// RunOnSource unwraps the request envelope, runs one of the pipeline entry
// points over it, and answers either that entry point's bytes or the
// {"error": "..."} envelope. Both shells hand their frontend the same strings,
// so this sequencing is theirs to share rather than to spell twice.
func RunOnSource(request string, run func(source, filename string) ([]byte, error)) string {
	req, err := ExtractRequest(request)
	if err != nil {
		return ErrorJSON(err.Error())
	}

	result, err := run(req.Source, req.Filename)
	if err != nil {
		return ErrorJSON(err.Error())
	}

	return string(result)
}

// ErrorJSON returns a JSON error string in the form {"error": "..."}.
func ErrorJSON(msg string) string {
	b, _ := json.Marshal(map[string]string{"error": msg})
	return string(b)
}

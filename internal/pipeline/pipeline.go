// Package pipeline runs the emod pipeline (lex → parse → validate → lint → export)
// behind functions that accept and return standard Go types, so every surface that
// drives a model — the browser shim, the desktop shell, a test — shares one
// orchestration and none of them imposes its transport on it.
package pipeline

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/diagnostic"
	"github.com/hpcsc/emod/internal/export"
	"github.com/hpcsc/emod/internal/formatter"
	"github.com/hpcsc/emod/internal/importer"
	"github.com/hpcsc/emod/internal/oracle"
)

type exportFunc func(model *ast.Model, diagnostics []*diagnostic.Entry, parsed bool) ([]byte, error)

// Request is the source to run and the name of the file it came from, which
// the diagnostics and node positions report.
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
// and invokes the given export function on the result.
func runPipeline(source, filename string, fn exportFunc) (result []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("pipeline panic: %v", r)
		}
	}()

	model, diags, parsed := oracle.RunParsed(source, filename)

	return fn(model, diags, parsed)
}

// RunPipelineExportDiagram runs the pipeline and answers the
// { diagnostics, diagram, parsed } document.
func RunPipelineExportDiagram(source, filename string) ([]byte, error) {
	return runPipeline(source, filename, export.ExportDiagramJSONParsed)
}

// RunPipelineExportJSON runs the pipeline and wraps the result
// in the model JSON diagnostics envelope { diagnostics, model }.
func RunPipelineExportJSON(source, filename string) ([]byte, error) {
	return runPipeline(source, filename, func(model *ast.Model, diagnostics []*diagnostic.Entry, _ bool) ([]byte, error) {
		return export.ExportJSONDiagnostics(model, diagnostics)
	})
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

// ApplyDiagramJSON takes the {source, filename, diagram} envelope, where diagram
// is the document a parse of source drew, edited in the viewer. It answers the
// source with that edit applied, under emod, beside the diagnostics, diagram and
// parsed flag a parse of the edited source answers. reformatted says whether
// emod fmt would have changed the source it was given, and dropped_comments how
// many of that source's comments the emod fmt layout does not keep.
func ApplyDiagramJSON(request string) string {
	req, err := ExtractRequest(request)
	if err != nil {
		return ErrorJSON(err.Error())
	}

	var envelope struct {
		Diagram json.RawMessage `json:"diagram"`
	}
	if err := json.Unmarshal([]byte(request), &envelope); err != nil || len(envelope.Diagram) == 0 || string(envelope.Diagram) == "null" {
		return ErrorJSON("missing diagram field")
	}

	result, err := applyDiagram(req, envelope.Diagram)
	if err != nil {
		return ErrorJSON(err.Error())
	}

	return string(result)
}

func applyDiagram(req Request, diagram []byte) (result []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("apply panic: %v", r)
		}
	}()

	model, diagnostics := oracle.Parse(req.Source, req.Filename)
	if len(diagnostics) > 0 {
		return nil, fmt.Errorf("the diagram edit cannot be applied: %s does not parse", req.Filename)
	}
	laidOut := formatter.Format(model)

	if err := importer.ApplyDiagram(model, diagram); err != nil {
		return nil, err
	}
	edited := formatter.Format(model)

	parsed, err := RunPipelineExportDiagram(edited, req.Filename)
	if err != nil {
		return nil, err
	}
	var answer map[string]any
	if err := json.Unmarshal(parsed, &answer); err != nil {
		return nil, err
	}
	answer["emod"] = edited
	answer["reformatted"] = laidOut != req.Source
	answer["dropped_comments"] = commentsIn(req.Source) - commentsIn(laidOut)

	return json.Marshal(answer)
}

// commentsIn also counts the # lines of a flow block: HCL reads them as heredoc
// text, and the parser skips them.
func commentsIn(source string) int {
	tokens, _ := hclsyntax.LexConfig([]byte(source), "", hcl.InitialPos)

	count := 0
	inFlow := false
	for i, token := range tokens {
		switch token.Type {
		case hclsyntax.TokenComment:
			count++
		case hclsyntax.TokenOHeredoc:
			inFlow = string(tokens[i-2].Bytes) == "flow"
		case hclsyntax.TokenStringLit:
			if inFlow && strings.HasPrefix(strings.TrimSpace(string(token.Bytes)), "#") {
				count++
			}
		}
	}
	return count
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

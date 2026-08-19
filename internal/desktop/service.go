// Package desktop is what a native shell binds to: the emod pipeline, and the
// filesystem reads a shell needs to put a model in front of it. It imports no
// GUI framework and links no CGO, so it is testable and buildable everywhere
// the rest of the repository is, and the framework-specific shell stays a thin
// binding layer over it.
package desktop

import "github.com/hpcsc/emod/internal/pipeline"

// ModelService is the surface a native shell binds to its frontend. Its methods
// take and return the same JSON strings the browser shim hands across the
// WebAssembly boundary — including the {"error": "..."} envelope for failures —
// so both runtimes satisfy one contract and the shared frontend modules need no
// per-runtime knowledge.
type ModelService struct{}

// ParseEmod takes the {"source": "..."} envelope and answers the pipeline's
// {diagnostics, diagram} document. Source the pipeline reports on still yields a
// diagram beside the diagnostics; only a malformed envelope is an error.
func (s *ModelService) ParseEmod(request string) string {
	return pipeline.RunOnSource(request, pipeline.RunPipelineExportDiagram)
}

// ExportJSON takes the {"source": "..."} envelope and answers the pipeline's
// {diagnostics, model} document.
func (s *ModelService) ExportJSON(request string) string {
	return pipeline.RunOnSource(request, pipeline.RunPipelineExportJSON)
}

// ExportEmod takes the diagram document itself rather than the {"source": "..."}
// envelope the parse entry points use, because the frontend already holds that
// document as its state.
func (s *ModelService) ExportEmod(diagramJSON string) string {
	return pipeline.ExportEmodJSON(diagramJSON)
}

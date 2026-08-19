//go:build js && wasm

package main

import (
	"syscall/js"

	"github.com/hpcsc/emod/internal/pipeline"
)

func main() {
	js.Global().Set("parseEmod", js.FuncOf(jsParseEmod))
	js.Global().Set("exportJSON", js.FuncOf(jsExportJSON))
	js.Global().Set("exportEmod", js.FuncOf(jsExportEmod))

	select {}
}

func jsParseEmod(this js.Value, args []js.Value) any {
	return jsHandle(args, pipeline.RunPipelineExportDiagram)
}

func jsExportJSON(this js.Value, args []js.Value) any {
	return jsHandle(args, pipeline.RunPipelineExportJSON)
}

// jsExportEmod takes the diagram JSON document itself rather than the
// {"source": "..."} envelope the parse entry points use, because the viewer
// already holds that document as its state.
func jsExportEmod(this js.Value, args []js.Value) any {
	if len(args) != 1 {
		return pipeline.ErrorJSON("expected 1 argument")
	}

	return pipeline.ExportEmodJSON(args[0].String())
}

func jsHandle(args []js.Value, run func(string) ([]byte, error)) any {
	if len(args) != 1 {
		return pipeline.ErrorJSON("expected 1 argument")
	}

	source, err := pipeline.ExtractSource(args[0].String())
	if err != nil {
		return pipeline.ErrorJSON(err.Error())
	}

	result, err := run(source)
	if err != nil {
		return pipeline.ErrorJSON(err.Error())
	}

	return string(result)
}

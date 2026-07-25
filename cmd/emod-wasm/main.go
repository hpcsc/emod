package main

import (
	"syscall/js"

	"github.com/hpcsc/emod/internal/wasm"
)

func main() {
	js.Global().Set("parseEmod", js.FuncOf(jsParseEmod))
	js.Global().Set("exportJSON", js.FuncOf(jsExportJSON))
	js.Global().Set("exportEmod", js.FuncOf(jsExportEmod))

	select {}
}

func jsParseEmod(this js.Value, args []js.Value) any {
	return jsHandle(args, wasm.RunPipelineExportDiagram)
}

func jsExportJSON(this js.Value, args []js.Value) any {
	return jsHandle(args, wasm.RunPipelineExportJSON)
}

// jsExportEmod takes the diagram JSON document itself rather than the
// {"source": "..."} envelope the parse entry points use, because the viewer
// already holds that document as its state.
func jsExportEmod(this js.Value, args []js.Value) any {
	if len(args) != 1 {
		return wasm.ErrorJSON("expected 1 argument")
	}

	result, err := wasm.ExportEmod(args[0].String())
	if err != nil {
		return wasm.ErrorJSON(err.Error())
	}

	return string(result)
}

func jsHandle(args []js.Value, pipeline func(string) ([]byte, error)) any {
	if len(args) != 1 {
		return wasm.ErrorJSON("expected 1 argument")
	}

	source, err := wasm.ExtractSource(args[0].String())
	if err != nil {
		return wasm.ErrorJSON(err.Error())
	}

	result, err := pipeline(source)
	if err != nil {
		return wasm.ErrorJSON(err.Error())
	}

	return string(result)
}

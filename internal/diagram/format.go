package diagram

import (
	"fmt"
	"strings"

	"github.com/hpcsc/emod/internal/ast"
)

func Formats() []string {
	return []string{"svg", "mermaid", "ascii", "event-flow", "event-flow-mermaid", "event-flow-mermaid-no-icons"}
}

func DrawsSpecCards(format string) bool {
	return format == "svg"
}

func Export(model *ast.Model, format string, style Style, options ...Option) ([]byte, error) {
	if len(options) > 0 && !DrawsSpecCards(format) {
		return nil, fmt.Errorf("format %q takes no options; only svg draws spec cards", format)
	}
	switch format {
	case "svg":
		return ExportSVG(model, style, options...)
	case "mermaid":
		return ExportMermaid(model, style)
	case "ascii":
		return ExportASCII(model, style)
	case "event-flow":
		return ExportEventFlow(model, style)
	case "event-flow-mermaid":
		return ExportEventFlowMermaid(model, style)
	case "event-flow-mermaid-no-icons":
		return ExportEventFlowMermaidNoIcons(model, style)
	default:
		return nil, fmt.Errorf("unsupported format %q; supported formats: %s", format, strings.Join(Formats(), ", "))
	}
}

//go:build unit

package diagram_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/diagram"
	"github.com/hpcsc/emod/internal/parser"
)

func TestFormat(t *testing.T) {
	t.Run("draws spec cards", func(t *testing.T) {
		t.Run("says that only svg draws spec cards", func(t *testing.T) {
			drawing := map[string]bool{}
			for _, format := range diagram.Formats() {
				drawing[format] = diagram.DrawsSpecCards(format)
			}

			require.Equal(t, map[string]bool{
				"svg":                         true,
				"mermaid":                     false,
				"ascii":                       false,
				"event-flow":                  false,
				"event-flow-mermaid":          false,
				"event-flow-mermaid-no-icons": false,
			}, drawing)
		})
	})

	t.Run("export", func(t *testing.T) {
		t.Run("gives each format the output of its own renderer", func(t *testing.T) {
			model := fullModel()
			renderers := map[string]func(*ast.Model, diagram.Style) ([]byte, error){
				"svg":                         func(m *ast.Model, s diagram.Style) ([]byte, error) { return diagram.ExportSVG(m, s) },
				"mermaid":                     diagram.ExportMermaid,
				"ascii":                       diagram.ExportASCII,
				"event-flow":                  diagram.ExportEventFlow,
				"event-flow-mermaid":          diagram.ExportEventFlowMermaid,
				"event-flow-mermaid-no-icons": diagram.ExportEventFlowMermaidNoIcons,
			}
			outputs := map[string]string{}

			for _, format := range diagram.Formats() {
				drawn, err := diagram.Export(model, format, diagram.StyleAuto)
				require.NoError(t, err, "format %s", format)
				expected, err := renderers[format](model, diagram.StyleAuto)
				require.NoError(t, err, "format %s", format)

				require.Equal(t, string(expected), string(drawn), "format %s", format)
				outputs[format] = string(drawn)
			}
			distinct := map[string]bool{}
			for _, output := range outputs {
				distinct[output] = true
			}
			require.Len(t, distinct, len(outputs), "the fixture must draw a different output in each format, or a swap passes")
		})

		t.Run("gives the style to the renderer", func(t *testing.T) {
			model := dcbModel(t)

			projected, err := diagram.Export(model, "mermaid", diagram.StyleProjected)
			require.NoError(t, err)
			dcb, err := diagram.Export(model, "mermaid", diagram.StyleDCB)
			require.NoError(t, err)

			expected, err := diagram.ExportMermaid(model, diagram.StyleDCB)
			require.NoError(t, err)
			require.Equal(t, string(expected), string(dcb))
			require.NotEqual(t, string(projected), string(dcb), "the fixture must draw differently in the two styles")
		})

		t.Run("gives the options to svg, so that it draws the spec cards", func(t *testing.T) {
			model := specsModel(t)

			plain, err := diagram.Export(model, "svg", diagram.StyleAuto)
			require.NoError(t, err)
			cards, err := diagram.Export(model, "svg", diagram.StyleAuto, diagram.WithSpecs())
			require.NoError(t, err)

			expected, err := diagram.ExportSVG(model, diagram.StyleAuto, diagram.WithSpecs())
			require.NoError(t, err)
			require.Equal(t, string(expected), string(cards))
			require.NotEqual(t, string(plain), string(cards))
		})

		t.Run("refuses an option for a format that draws no spec cards, and does not drop it", func(t *testing.T) {
			for _, format := range diagram.Formats() {
				if diagram.DrawsSpecCards(format) {
					continue
				}

				drawn, err := diagram.Export(specsModel(t), format, diagram.StyleAuto, diagram.WithSpecs())

				require.EqualError(t, err, `format "`+format+`" takes no options; only svg draws spec cards`)
				require.Nil(t, drawn)
			}
		})

		t.Run("gives an error that names the unknown format and each format", func(t *testing.T) {
			_, err := diagram.Export(fullModel(), "png", diagram.StyleAuto)

			require.EqualError(t, err,
				`unsupported format "png"; supported formats: svg, mermaid, ascii, event-flow, event-flow-mermaid, event-flow-mermaid-no-icons`)
		})
	})
}

func dcbModel(t *testing.T) *ast.Model {
	t.Helper()

	return parsedExample(t, "../../examples/dcb_model.emod")
}

func specsModel(t *testing.T) *ast.Model {
	t.Helper()

	return parsedExample(t, "../../examples/specs_hotel.emod")
}

func parsedExample(t *testing.T, path string) *ast.Model {
	t.Helper()

	source, err := os.ReadFile(path)
	require.NoError(t, err)
	model, diagnostics := parser.Parse(string(source), path)
	require.Empty(t, diagnostics)
	return model
}

//go:build unit

package pipeline_test

import (
	"encoding/json"
	"testing"

	"github.com/hpcsc/emod/internal/pipeline"
	"github.com/stretchr/testify/require"
)

const billingModel = `emod 1
model "Billing"

context "Payments" {
  aggregate "Payment" {
    slice "Take Payment" {
      command TakePayment {
        fields {
          amount int required
        }
      }

      event PaymentTaken {
        fields {
          amount int required
        }
      }

      flow {
        command -> event: TakePayment -> PaymentTaken
      }
    }
  }
}
`

func TestPipeline(t *testing.T) {
	t.Run("extract source", func(t *testing.T) {
		t.Run("returns the source field", func(t *testing.T) {
			source, err := pipeline.ExtractSource(`{"source": "model MyModel"}`)

			require.NoError(t, err)
			require.Equal(t, "model MyModel", source)
		})

		t.Run("malformed JSON returns error", func(t *testing.T) {
			_, err := pipeline.ExtractSource(`not json`)

			require.Error(t, err)
			require.Contains(t, err.Error(), "invalid JSON")
		})

		t.Run("a missing or empty source field returns error", func(t *testing.T) {
			for _, input := range []string{`{}`, `{"source": ""}`} {
				_, err := pipeline.ExtractSource(input)

				require.Error(t, err, "input %s", input)
				require.Contains(t, err.Error(), "missing source field")
			}
		})
	})

	t.Run("export diagram", func(t *testing.T) {
		t.Run("returns the model's nodes and edges with no diagnostics", func(t *testing.T) {
			result, err := pipeline.RunPipelineExportDiagram(billingModel)
			require.NoError(t, err)

			envelope := decodeDiagramEnvelope(t, result)

			require.Empty(t, envelope.Diagnostics)
			require.Equal(t, "Billing", envelope.Diagram.ModelName)
			require.Equal(t, []string{"TakePayment", "PaymentTaken"},
				labelsOfType(envelope.Diagram.Nodes, "command", "event"))
			require.Equal(t, []edge{{Source: "command-1", Target: "event-1", Type: "flow"}},
				envelope.Diagram.Edges)
		})

		t.Run("reports diagnostics for unparseable source and still returns an envelope", func(t *testing.T) {
			result, err := pipeline.RunPipelineExportDiagram("foobar {\n}\n")
			require.NoError(t, err)

			envelope := decodeDiagramEnvelope(t, result)

			require.NotEmpty(t, envelope.Diagnostics, "an unparseable model must report why")
			require.Contains(t, envelope.Diagnostics[0].Message, "foobar")
		})

		t.Run("empty source yields an envelope with no nodes", func(t *testing.T) {
			result, err := pipeline.RunPipelineExportDiagram("")
			require.NoError(t, err)

			envelope := decodeDiagramEnvelope(t, result)

			require.Empty(t, envelope.Diagram.Nodes)
		})
	})

	t.Run("export model json", func(t *testing.T) {
		t.Run("returns the parsed model with no diagnostics", func(t *testing.T) {
			result, err := pipeline.RunPipelineExportJSON(billingModel)
			require.NoError(t, err)

			var envelope struct {
				Diagnostics []diagnostic `json:"diagnostics"`
				Model       struct {
					Name string `json:"name"`
				} `json:"model"`
			}
			require.NoError(t, json.Unmarshal(result, &envelope))

			require.Empty(t, envelope.Diagnostics)
			require.Equal(t, "Billing", envelope.Model.Name)
		})
	})

	t.Run("export emod", func(t *testing.T) {
		t.Run("diagram JSON round-trips back to the source it was parsed from", func(t *testing.T) {
			parsed, err := pipeline.RunPipelineExportDiagram(billingModel)
			require.NoError(t, err)

			var envelope struct {
				Diagram json.RawMessage `json:"diagram"`
			}
			require.NoError(t, json.Unmarshal(parsed, &envelope))

			result, err := pipeline.ExportEmod(string(envelope.Diagram))
			require.NoError(t, err)

			require.Equal(t, billingModel, string(result))
		})

		t.Run("malformed diagram JSON returns error", func(t *testing.T) {
			_, err := pipeline.ExportEmod(`not json`)

			require.Error(t, err)
			require.Contains(t, err.Error(), "invalid diagram JSON")
		})
	})

	t.Run("export emod as json", func(t *testing.T) {
		t.Run("wraps the formatted source in an emod field", func(t *testing.T) {
			parsed := decodeEmodEnvelope(t, pipeline.ExportEmodJSON(`{"model_name":"Billing","nodes":[],"edges":[]}`))

			require.Empty(t, parsed.Error)
			require.Equal(t, "emod 1\nmodel \"Billing\"\n", parsed.Emod)
		})

		t.Run("reports a failure in an error field instead of an emod field", func(t *testing.T) {
			parsed := decodeEmodEnvelope(t, pipeline.ExportEmodJSON(`not json`))

			require.Empty(t, parsed.Emod)
			require.Contains(t, parsed.Error, "invalid diagram JSON")
		})
	})

	t.Run("error json", func(t *testing.T) {
		t.Run("carries the message in an error field", func(t *testing.T) {
			var parsed struct {
				Error string `json:"error"`
			}
			require.NoError(t, json.Unmarshal([]byte(pipeline.ErrorJSON("something went wrong")), &parsed))

			require.Equal(t, "something went wrong", parsed.Error)
		})
	})
}

type diagnostic struct {
	Message  string `json:"message"`
	Severity string `json:"severity"`
}

type node struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Label string `json:"label"`
}

type edge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type"`
}

type diagramEnvelope struct {
	Diagnostics []diagnostic `json:"diagnostics"`
	Diagram     struct {
		ModelName string `json:"model_name"`
		Nodes     []node `json:"nodes"`
		Edges     []edge `json:"edges"`
	} `json:"diagram"`
}

func decodeDiagramEnvelope(t *testing.T, raw []byte) diagramEnvelope {
	t.Helper()

	var envelope diagramEnvelope
	require.NoError(t, json.Unmarshal(raw, &envelope), "pipeline must return valid JSON")

	return envelope
}

func decodeEmodEnvelope(t *testing.T, raw string) struct {
	Emod  string `json:"emod"`
	Error string `json:"error"`
} {
	t.Helper()

	var parsed struct {
		Emod  string `json:"emod"`
		Error string `json:"error"`
	}
	require.NoError(t, json.Unmarshal([]byte(raw), &parsed))

	return parsed
}

// labelsOfType returns the labels of nodes matching any of the given types, in
// document order.
func labelsOfType(nodes []node, types ...string) []string {
	wanted := make(map[string]bool, len(types))
	for _, t := range types {
		wanted[t] = true
	}

	var labels []string
	for _, n := range nodes {
		if wanted[n.Type] {
			labels = append(labels, n.Label)
		}
	}
	return labels
}

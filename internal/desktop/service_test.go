//go:build unit

package desktop_test

import (
	"encoding/json"
	"testing"

	"github.com/hpcsc/emod/internal/desktop"
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

func sourceEnvelope(t *testing.T, source string) string {
	t.Helper()

	b, err := json.Marshal(map[string]string{"source": source})
	require.NoError(t, err)

	return string(b)
}

func TestModelService(t *testing.T) {
	t.Run("parse emod", func(t *testing.T) {
		t.Run("answers exactly what the pipeline produces for the same source", func(t *testing.T) {
			service := &desktop.ModelService{}

			expected, err := pipeline.RunPipelineExportDiagram(billingModel)
			require.NoError(t, err)

			require.Equal(t, string(expected), service.ParseEmod(sourceEnvelope(t, billingModel)))
		})

		t.Run("returns a diagram alongside diagnostics rather than an error", func(t *testing.T) {
			service := &desktop.ModelService{}

			var envelope struct {
				Diagnostics []map[string]any `json:"diagnostics"`
				Diagram     map[string]any   `json:"diagram"`
			}
			answer := service.ParseEmod(sourceEnvelope(t, `emod 1
model "Billing"

context "Payments" {
  aggregate "Payment" {
    slice "Take Payment" {
      command TakePayment {
        fields {
          amount int required
        }
      }
    }
  }
}
`))
			require.NoError(t, json.Unmarshal([]byte(answer), &envelope))
			require.NotEmpty(t, envelope.Diagnostics, "a command no flow names is an orphan")
			require.NotEmpty(t, envelope.Diagram)
		})

		t.Run("reports a payload that is not the source envelope", func(t *testing.T) {
			service := &desktop.ModelService{}

			var envelope struct {
				Error string `json:"error"`
			}
			require.NoError(t, json.Unmarshal([]byte(service.ParseEmod("not json")), &envelope))
			require.Contains(t, envelope.Error, "invalid JSON")
		})

		t.Run("reports an envelope stating no source", func(t *testing.T) {
			service := &desktop.ModelService{}

			require.Equal(t, pipeline.ErrorJSON("missing source field"), service.ParseEmod(`{}`))
		})
	})

	t.Run("export json", func(t *testing.T) {
		t.Run("answers exactly what the pipeline produces for the same source", func(t *testing.T) {
			service := &desktop.ModelService{}

			var envelope struct {
				Diagnostics []map[string]any `json:"diagnostics"`
				Model       struct {
					Name string `json:"name"`
				} `json:"model"`
			}
			answer := service.ExportJSON(sourceEnvelope(t, billingModel))
			require.NoError(t, json.Unmarshal([]byte(answer), &envelope))
			require.Empty(t, envelope.Diagnostics)
			require.Equal(t, "Billing", envelope.Model.Name)
		})

		t.Run("reports an envelope stating no source", func(t *testing.T) {
			service := &desktop.ModelService{}

			require.Equal(t, pipeline.ErrorJSON("missing source field"), service.ExportJSON(`{}`))
		})
	})

	t.Run("export emod", func(t *testing.T) {
		t.Run("takes the diagram document itself, not the source envelope", func(t *testing.T) {
			service := &desktop.ModelService{}

			diagram := service.ParseEmod(sourceEnvelope(t, billingModel))
			var parsed struct {
				Diagram json.RawMessage `json:"diagram"`
			}
			require.NoError(t, json.Unmarshal([]byte(diagram), &parsed))

			var envelope struct {
				Emod  string `json:"emod"`
				Error string `json:"error"`
			}
			require.NoError(t, json.Unmarshal([]byte(service.ExportEmod(string(parsed.Diagram))), &envelope))
			require.Contains(t, envelope.Emod, `model "Billing"`)

			// Handed the envelope the parse entry points take, it reads the
			// document keys that envelope does not have and writes an empty
			// model — so this asserts which shape the method actually reads,
			// which an implementation forwarding either one would fail.
			var fromEnvelope struct {
				Emod string `json:"emod"`
			}
			require.NoError(t, json.Unmarshal(
				[]byte(service.ExportEmod(sourceEnvelope(t, billingModel))), &fromEnvelope))
			require.NotContains(t, fromEnvelope.Emod, "Billing")
		})

		t.Run("answers the emod envelope for a document it can import", func(t *testing.T) {
			service := &desktop.ModelService{}

			diagram := service.ParseEmod(sourceEnvelope(t, billingModel))
			var parsed struct {
				Diagram json.RawMessage `json:"diagram"`
			}
			require.NoError(t, json.Unmarshal([]byte(diagram), &parsed))

			var envelope struct {
				Emod  string `json:"emod"`
				Error string `json:"error"`
			}
			require.NoError(t, json.Unmarshal([]byte(service.ExportEmod(string(parsed.Diagram))), &envelope))
			require.Empty(t, envelope.Error)
			require.Contains(t, envelope.Emod, `model "Billing"`)
			require.Contains(t, envelope.Emod, "TakePayment")
		})

		t.Run("answers the error envelope for a document it cannot import", func(t *testing.T) {
			service := &desktop.ModelService{}

			answer := service.ExportEmod("not a diagram document")

			var envelope struct {
				Emod  string `json:"emod"`
				Error string `json:"error"`
			}
			require.NoError(t, json.Unmarshal([]byte(answer), &envelope))
			require.NotEmpty(t, envelope.Error)
			require.Empty(t, envelope.Emod)
		})
	})
}

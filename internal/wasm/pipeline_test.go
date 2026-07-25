package wasm_test

import (
	"encoding/json"
	"testing"

	"github.com/hpcsc/emod/internal/wasm"
	"github.com/stretchr/testify/require"
)

func TestExtractSource(t *testing.T) {
	t.Run("valid input returns source", func(t *testing.T) {
		source, err := wasm.ExtractSource(`{"source": "model MyModel"}`)
		require.NoError(t, err)
		require.Equal(t, "model MyModel", source)
	})

	t.Run("malformed JSON returns error", func(t *testing.T) {
		_, err := wasm.ExtractSource(`not json`)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid JSON")
	})

	t.Run("missing source field returns error", func(t *testing.T) {
		_, err := wasm.ExtractSource(`{}`)
		require.Error(t, err)
		require.Contains(t, err.Error(), "missing source field")
	})

	t.Run("empty source returns error", func(t *testing.T) {
		_, err := wasm.ExtractSource(`{"source": ""}`)
		require.Error(t, err)
		require.Contains(t, err.Error(), "missing source field")
	})
}

func TestRunPipelineExportDiagram(t *testing.T) {
	t.Run("valid source returns diagram diagnostics wrapper", func(t *testing.T) {
		result, err := wasm.RunPipelineExportDiagram("model MyModel")
		require.NoError(t, err)
		require.NotEmpty(t, result)

		var wrapper struct {
			Diagnostics []json.RawMessage `json:"diagnostics"`
			Diagram     json.RawMessage   `json:"diagram"`
		}
		err = json.Unmarshal(result, &wrapper)
		require.NoError(t, err, "result must be valid JSON")
		require.NotNil(t, wrapper.Diagnostics, "diagnostics field must be present")
		require.NotNil(t, wrapper.Diagram, "diagram field must be present")
	})

	t.Run("empty source returns valid envelope", func(t *testing.T) {
		result, err := wasm.RunPipelineExportDiagram("")
		require.NoError(t, err)
		require.NotEmpty(t, result)

		var wrapped struct {
			Diagnostics []json.RawMessage `json:"diagnostics"`
			Diagram     json.RawMessage   `json:"diagram"`
		}
		err = json.Unmarshal(result, &wrapped)
		require.NoError(t, err)
		require.NotNil(t, wrapped.Diagnostics)
	})
}

func TestRunPipelineExportJSON(t *testing.T) {
	t.Run("valid source returns model diagnostics wrapper", func(t *testing.T) {
		result, err := wasm.RunPipelineExportJSON("model MyModel")
		require.NoError(t, err)
		require.NotEmpty(t, result)

		var wrapper struct {
			Diagnostics []json.RawMessage `json:"diagnostics"`
			Model       json.RawMessage   `json:"model"`
		}
		err = json.Unmarshal(result, &wrapper)
		require.NoError(t, err, "result must be valid JSON")
		require.NotNil(t, wrapper.Diagnostics, "diagnostics field must be present")
		require.NotNil(t, wrapper.Model, "model field must be present")
	})
}

func TestExportEmod(t *testing.T) {
	t.Run("diagram JSON round-trips back to the source it was parsed from", func(t *testing.T) {
		source := `model "Billing"

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
		parsed, err := wasm.RunPipelineExportDiagram(source)
		require.NoError(t, err)

		var envelope struct {
			Diagram json.RawMessage `json:"diagram"`
		}
		require.NoError(t, json.Unmarshal(parsed, &envelope))

		result, err := wasm.ExportEmod(string(envelope.Diagram))
		require.NoError(t, err)
		require.Equal(t, source, string(result))
	})

	t.Run("malformed diagram JSON returns error", func(t *testing.T) {
		_, err := wasm.ExportEmod(`not json`)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid diagram JSON")
	})
}

func TestErrorJSON(t *testing.T) {
	t.Run("returns JSON with error field", func(t *testing.T) {
		result := wasm.ErrorJSON("something went wrong")
		require.NotEmpty(t, result)

		var parsed struct {
			Error string `json:"error"`
		}
		err := json.Unmarshal([]byte(result), &parsed)
		require.NoError(t, err, "must be valid JSON")
		require.Equal(t, "something went wrong", parsed.Error)
	})
}

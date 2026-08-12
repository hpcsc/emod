//go:build unit

package cli_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/hpcsc/emod/internal/cli"
	"github.com/hpcsc/emod/internal/test"
	"github.com/stretchr/testify/require"
)

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = w

	fn()

	err = w.Close()
	require.NoError(t, err)
	os.Stderr = old

	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)
	return buf.String()
}

func TestExport(t *testing.T) {
	// --- JSON format tests ---

	t.Run("valid file outputs wrapped JSON with empty diagnostics to stdout", func(t *testing.T) {
		path := writeTemp(t, "valid.emod", validEmod)

		output := captureStdout(t, func() {
			err := cli.RunExport(path, "json")
			require.NoError(t, err)
		})

		var doc map[string]interface{}
		err := json.Unmarshal([]byte(output), &doc)
		require.NoError(t, err)

		diags, ok := doc["diagnostics"].([]interface{})
		require.True(t, ok, "expected diagnostics in output")
		require.Empty(t, diags, "expected empty diagnostics for valid file")

		modelVal, ok := doc["model"].(map[string]interface{})
		require.True(t, ok, "expected model in output")
		require.Equal(t, "Hotel Reservation", modelVal["name"])
	})

	t.Run("valid file wrapped output includes actors and contexts under model key", func(t *testing.T) {
		path := writeTemp(t, "valid.emod", validEmod)

		output := captureStdout(t, func() {
			err := cli.RunExport(path, "json")
			require.NoError(t, err)
		})

		var doc map[string]interface{}
		err := json.Unmarshal([]byte(output), &doc)
		require.NoError(t, err)

		modelVal, ok := doc["model"].(map[string]interface{})
		require.True(t, ok, "expected model key in output")

		actors, ok := modelVal["actors"].([]interface{})
		require.True(t, ok, "expected actors in model output")
		require.Len(t, actors, 1)
		require.Equal(t, "Guest", actors[0].(map[string]interface{})["name"])

		_, ok = modelVal["contexts"].([]interface{})
		require.True(t, ok, "expected contexts in model output")
	})

	t.Run("described file's descriptions reach the model object of the JSON envelope", func(t *testing.T) {
		path := writeTemp(t, "described.emod", describedEmod)

		output := captureStdout(t, func() {
			err := cli.RunExport(path, "json")
			require.NoError(t, err)
		})

		var doc map[string]interface{}
		require.NoError(t, json.Unmarshal([]byte(output), &doc))

		require.Len(t, doc, 2, "envelope should carry diagnostics and model only")
		require.Empty(t, doc["diagnostics"])

		modelVal := doc["model"].(map[string]interface{})
		actor := modelVal["actors"].([]interface{})[0].(map[string]interface{})
		context := modelVal["contexts"].([]interface{})[0].(map[string]interface{})
		aggregate := context["aggregates"].([]interface{})[0].(map[string]interface{})
		slice := aggregate["slices"].([]interface{})[0].(map[string]interface{})
		command := slice["commands"].([]interface{})[0].(map[string]interface{})

		require.Equal(t, map[string]interface{}{
			"model":     "How the hotel takes, confirms and imports room bookings",
			"actor":     "A person booking a room, not necessarily the one staying in it",
			"context":   "Everything the hotel knows about a stay before the guest arrives",
			"aggregate": "One guest holding one room over one date range",
			"slice":     "A guest books a room from the public site",
			"command":   "Ask the hotel to hold a room for a date range, 10% deposit taken up front",
		}, map[string]interface{}{
			"model":     modelVal["description"],
			"actor":     actor["description"],
			"context":   context["description"],
			"aggregate": aggregate["description"],
			"slice":     slice["description"],
			"command":   command["description"],
		})
	})

	t.Run("a file binding wire types carries them into the model object of the JSON envelope", func(t *testing.T) {
		path := writeTemp(t, "wire-types.emod", test.WireTypeLibraryLending)

		output := captureStdout(t, func() {
			err := cli.RunExport(path, "json")
			require.NoError(t, err)
		})

		var doc map[string]interface{}
		require.NoError(t, json.Unmarshal([]byte(output), &doc))

		require.Len(t, doc, 2, "envelope should carry diagnostics and model only")
		require.Empty(t, doc["diagnostics"])

		modelVal := doc["model"].(map[string]interface{})
		context := modelVal["contexts"].([]interface{})[0].(map[string]interface{})
		aggregate := context["aggregates"].([]interface{})[0].(map[string]interface{})
		slice := aggregate["slices"].([]interface{})[0].(map[string]interface{})
		event := slice["events"].([]interface{})[0].(map[string]interface{})

		require.Equal(t, "com.library.lending.copy-borrowed", event["type"])
	})

	t.Run("a file naming its fields after keywords exports those names with no diagnostics", func(t *testing.T) {
		path := writeTemp(t, "keyword-fields.emod", keywordFieldEmod)

		output := captureStdout(t, func() {
			err := cli.RunExport(path, "json")
			require.NoError(t, err)
		})

		var doc map[string]interface{}
		require.NoError(t, json.Unmarshal([]byte(output), &doc))
		require.Empty(t, doc["diagnostics"])

		modelVal := doc["model"].(map[string]interface{})
		context := modelVal["contexts"].([]interface{})[0].(map[string]interface{})
		aggregate := context["aggregates"].([]interface{})[0].(map[string]interface{})
		slice := aggregate["slices"].([]interface{})[0].(map[string]interface{})
		command := slice["commands"].([]interface{})[0].(map[string]interface{})

		var names []string
		for _, field := range command["fields"].([]interface{}) {
			names = append(names, field.(map[string]interface{})["name"].(string))
		}
		require.Equal(t, []string{"model", "source", "where", "and", "not", "fields", "description"}, names)
	})

	t.Run("file with validation errors outputs JSON with diagnostics on stdout and empty stderr", func(t *testing.T) {
		path := writeTemp(t, "invalid.emod", invalidEmod)

		var err error
		var output string
		stderr := captureStderr(t, func() {
			output = captureStdout(t, func() {
				err = cli.RunExport(path, "json")
			})
		})

		require.Error(t, err)
		var lintErr *cli.LintError
		require.True(t, errors.As(err, &lintErr))
		require.Equal(t, "", lintErr.Message)
		require.Equal(t, 2, lintErr.ExitCode)

		// stderr should be empty for JSON format
		require.Empty(t, stderr, "JSON format should not write diagnostics to stderr")

		// stdout should contain diagnostics wrapper
		var doc map[string]interface{}
		err = json.Unmarshal([]byte(output), &doc)
		require.NoError(t, err)

		diags, ok := doc["diagnostics"].([]interface{})
		require.True(t, ok, "expected diagnostics in output")
		require.NotEmpty(t, diags, "expected non-empty diagnostics for invalid file")

		_, ok = doc["model"].(map[string]interface{})
		require.True(t, ok, "expected model key in output (partial model)")
	})

	t.Run("file with only lint warnings outputs JSON with diagnostics on stdout and empty stderr", func(t *testing.T) {
		input := `model "Test"
context "Orders" {
  aggregate "Order" {
    slice "Update Order" {
      command PlaceOrder {
        fields {
          orderId string required
          reason  string required
        }
      }
      event OrderUpdated {
        fields {
          orderId string required
          reason  string required
        }
      }
      flow {
        command -> event: PlaceOrder -> OrderUpdated
      }
    }
  }
}
`
		path := writeTemp(t, "warnings.emod", input)

		var err error
		var output string
		stderr := captureStderr(t, func() {
			output = captureStdout(t, func() {
				err = cli.RunExport(path, "json")
			})
		})

		require.Error(t, err)
		var lintErr *cli.LintError
		require.True(t, errors.As(err, &lintErr))
		require.Equal(t, 1, lintErr.ExitCode)
		require.Equal(t, "", lintErr.Message)

		// stderr should be empty for JSON format
		require.Empty(t, stderr, "JSON format should not write diagnostics to stderr")

		// stdout should contain diagnostics wrapper
		var doc map[string]interface{}
		err = json.Unmarshal([]byte(output), &doc)
		require.NoError(t, err)

		diags, ok := doc["diagnostics"].([]interface{})
		require.True(t, ok, "expected diagnostics in output")
		require.NotEmpty(t, diags, "expected non-empty diagnostics for file with warnings")
	})

	t.Run("unparseable file outputs JSON with diagnostics on stdout and empty stderr", func(t *testing.T) {
		path := writeTemp(t, "invalid.emod", invalidEmod)

		var err error
		var output string
		stderr := captureStderr(t, func() {
			output = captureStdout(t, func() {
				err = cli.RunExport(path, "json")
			})
		})

		require.Error(t, err)
		var lintErr *cli.LintError
		require.True(t, errors.As(err, &lintErr))
		require.Equal(t, "", lintErr.Message)
		require.Equal(t, 2, lintErr.ExitCode)

		// stderr should be empty for JSON format
		require.Empty(t, stderr, "JSON format should not write diagnostics to stderr")

		// stdout should contain diagnostics wrapper
		var doc map[string]interface{}
		err = json.Unmarshal([]byte(output), &doc)
		require.NoError(t, err)

		diags, ok := doc["diagnostics"].([]interface{})
		require.True(t, ok, "expected diagnostics in output")
		require.NotEmpty(t, diags, "expected non-empty diagnostics for unparseable file")

		_, ok = doc["model"].(map[string]interface{})
		require.True(t, ok, "expected model key in output (partial model)")
	})

	t.Run("missing file argument returns error", func(t *testing.T) {
		err := cli.RunExport("", "json")

		require.ErrorIs(t, err, cli.ErrMissingFileArgument)
	})

	t.Run("nonexistent file returns error naming the file", func(t *testing.T) {
		missing := filepath.Join(t.TempDir(), "nonexistent.emod")

		err := cli.RunExport(missing, "json")

		require.Error(t, err)
		require.Contains(t, err.Error(), missing)
	})

	t.Run("unsupported format returns error", func(t *testing.T) {
		path := writeTemp(t, "valid.emod", validEmod)

		err := cli.RunExport(path, "text")

		require.ErrorIs(t, err, cli.ErrUnsupportedFormat)
		require.Contains(t, err.Error(), "json")
		require.Contains(t, err.Error(), "cue")
		require.Contains(t, err.Error(), "diagram-json")
		var lintErr *cli.LintError
		require.True(t, errors.As(err, &lintErr))
		require.Equal(t, 1, lintErr.ExitCode)
	})

	t.Run("default format is json and produces wrapped valid JSON", func(t *testing.T) {
		path := writeTemp(t, "valid.emod", validEmod)

		output := captureStdout(t, func() {
			err := cli.RunExport(path, "json")
			require.NoError(t, err)
		})

		require.True(t, json.Valid([]byte(output)))

		var doc map[string]interface{}
		err := json.Unmarshal([]byte(output), &doc)
		require.NoError(t, err)

		_, hasDiagnostics := doc["diagnostics"]
		require.True(t, hasDiagnostics, "expected diagnostics key in JSON output")
		_, hasModel := doc["model"]
		require.True(t, hasModel, "expected model key in JSON output")
	})

	// --- diagram-json format tests ---

	t.Run("valid file outputs wrapped diagram JSON with empty diagnostics to stdout", func(t *testing.T) {
		path := writeTemp(t, "valid.emod", validEmod)

		output := captureStdout(t, func() {
			err := cli.RunExport(path, "diagram-json")
			require.NoError(t, err)
		})

		var doc map[string]interface{}
		err := json.Unmarshal([]byte(output), &doc)
		require.NoError(t, err)

		diags, ok := doc["diagnostics"].([]interface{})
		require.True(t, ok, "expected diagnostics in output")
		require.Empty(t, diags, "expected empty diagnostics for valid file")

		diagram, ok := doc["diagram"].(map[string]interface{})
		require.True(t, ok, "expected diagram in output")
		require.Equal(t, "Hotel Reservation", diagram["model_name"])

		nodes, ok := diagram["nodes"].([]interface{})
		require.True(t, ok, "expected nodes in diagram output")
		require.NotEmpty(t, nodes, "expected non-empty nodes for valid file")

		edges, ok := diagram["edges"].([]interface{})
		require.True(t, ok, "expected edges in diagram output")
		require.NotEmpty(t, edges, "expected non-empty edges for valid file")
	})

	t.Run("described file's descriptions reach the nodes of the diagram JSON envelope", func(t *testing.T) {
		path := writeTemp(t, "described.emod", describedEmod)

		output := captureStdout(t, func() {
			err := cli.RunExport(path, "diagram-json")
			require.NoError(t, err)
		})

		var doc map[string]interface{}
		require.NoError(t, json.Unmarshal([]byte(output), &doc))
		require.Empty(t, doc["diagnostics"])

		require.Equal(t, map[string]interface{}{
			"context": "Everything the hotel knows about a stay before the guest arrives",
			"slice":   "A guest books a room from the public site",
			"command": "Ask the hotel to hold a room for a date range, 10% deposit taken up front",
		}, map[string]interface{}{
			"context": findDiagramNodeByLabel(t, doc, "Reservations")["description"],
			"slice":   findDiagramNodeByLabel(t, doc, "Make Reservation")["description"],
			"command": findDiagramNodeByLabel(t, doc, "MakeReservation")["description"],
		})
	})

	t.Run("file with validation errors outputs diagram JSON with diagnostics on stdout and empty stderr", func(t *testing.T) {
		path := writeTemp(t, "invalid.emod", invalidEmod)

		var err error
		var output string
		stderr := captureStderr(t, func() {
			output = captureStdout(t, func() {
				err = cli.RunExport(path, "diagram-json")
			})
		})

		require.Error(t, err)
		var lintErr *cli.LintError
		require.True(t, errors.As(err, &lintErr))
		require.Equal(t, "", lintErr.Message)
		require.Equal(t, 2, lintErr.ExitCode)

		// stderr should be empty for diagram-json format
		require.Empty(t, stderr, "diagram-json format should not write diagnostics to stderr")

		// stdout should contain diagnostics wrapper
		var doc map[string]interface{}
		err = json.Unmarshal([]byte(output), &doc)
		require.NoError(t, err)

		diags, ok := doc["diagnostics"].([]interface{})
		require.True(t, ok, "expected diagnostics in output")
		require.NotEmpty(t, diags, "expected non-empty diagnostics for invalid file")

		diagram, ok := doc["diagram"].(map[string]interface{})
		require.True(t, ok, "expected diagram key in output (partial diagram)")

		_, ok = diagram["nodes"].([]interface{})
		require.True(t, ok, "expected nodes in diagram output")
	})

	// --- CUE format tests ---

	t.Run("valid file outputs CUE text to stdout with -f cue", func(t *testing.T) {
		path := writeTemp(t, "valid.emod", validEmod)

		output := captureStdout(t, func() {
			err := cli.RunExport(path, "cue")
			require.NoError(t, err)
		})

		require.Contains(t, output, "Hotel Reservation")
		require.Contains(t, output, "Guest")
	})

	t.Run("CUE format with diagnostics writes text to stderr and returns non-zero exit", func(t *testing.T) {
		path := writeTemp(t, "invalid.emod", invalidEmod)

		var err error
		output := captureStdout(t, func() {
			stderr := captureStderr(t, func() {
				err = cli.RunExport(path, "cue")
			})
			require.Contains(t, stderr, path)
			require.Contains(t, stderr, ":1:")
		})

		// stdout should be empty for CUE format with errors
		require.Empty(t, output, "CUE format with errors should not write to stdout")

		require.Error(t, err)
		var lintErr *cli.LintError
		require.True(t, errors.As(err, &lintErr))
		require.Equal(t, "", lintErr.Message)
		require.Equal(t, 2, lintErr.ExitCode)
	})

	t.Run("CUE format on clean file outputs text to stdout and no stderr", func(t *testing.T) {
		path := writeTemp(t, "valid.emod", validEmod)

		var stderrText string
		output := captureStdout(t, func() {
			stderrText = captureStderr(t, func() {
				err := cli.RunExport(path, "cue")
				require.NoError(t, err)
			})
		})

		require.Contains(t, output, "Hotel Reservation")
		require.Empty(t, stderrText, "CUE format on clean file should not write to stderr")
	})

	t.Run("CUE format on a file naming its fields after keywords prints those names and no stderr", func(t *testing.T) {
		path := writeTemp(t, "keyword-fields.emod", keywordFieldEmod)

		var stderrText string
		output := captureStdout(t, func() {
			stderrText = captureStderr(t, func() {
				err := cli.RunExport(path, "cue")
				require.NoError(t, err)
			})
		})

		for _, name := range []string{"model", "source", "where", "and", "not", "fields", "events", "tag", "emod", "description"} {
			require.Contains(t, output, `name: "`+name+`"`)
		}
		require.Empty(t, stderrText, "a model whose fields are named after keywords is legal, so nothing is reported")
	})
}

func TestExportAllPatterns(t *testing.T) {
	path := "../../examples/all_patterns.emod"

	t.Run("json export includes the trigger under its slice with actor and reads", func(t *testing.T) {
		output := captureStdout(t, func() {
			err := cli.RunExport(path, "json")
			require.NoError(t, err)
		})

		var doc map[string]any
		require.NoError(t, json.Unmarshal([]byte(output), &doc))
		require.Empty(t, doc["diagnostics"])

		reserveSlice := findSliceByName(t, doc["model"].(map[string]any), "Reserve a Room")
		trigger := reserveSlice["trigger"].(map[string]any)
		require.Equal(t, "Reservation Form", trigger["name"])
		require.Equal(t, "Guest", trigger["actor"])
		require.Equal(t, "AvailableRoomsView", trigger["reads"])
		_, hasKind := trigger["kind"]
		require.False(t, hasKind, "exported trigger must not carry kind")
		_, hasKindPosition := trigger["kind_position"]
		require.False(t, hasKindPosition, "exported trigger must not carry kind_position")
	})

	t.Run("cue export includes the trigger under its slice with actor and reads", func(t *testing.T) {
		var stderrText string
		output := captureStdout(t, func() {
			stderrText = captureStderr(t, func() {
				err := cli.RunExport(path, "cue")
				require.NoError(t, err)
			})
		})

		require.Empty(t, stderrText, "a clean file should not write to stderr")
		require.Contains(t, output, `trigger: {`)
		require.Contains(t, output, `name: "Reservation Form"`)
		require.Contains(t, output, `actor: "Guest"`)
		require.Contains(t, output, `reads: "AvailableRoomsView"`)
		require.NotContains(t, output, "kind: ", "exported CUE trigger must not carry kind")
	})
}

func findDiagramNodeByLabel(t *testing.T, doc map[string]interface{}, label string) map[string]interface{} {
	t.Helper()

	diagram := doc["diagram"].(map[string]interface{})
	for _, n := range diagram["nodes"].([]interface{}) {
		node := n.(map[string]interface{})
		if node["label"] == label {
			return node
		}
	}
	t.Fatalf("node labelled %q not found", label)
	return nil
}

func findSliceByName(t *testing.T, model map[string]any, name string) map[string]any {
	t.Helper()

	contexts := model["contexts"].([]any)
	for _, c := range contexts {
		context := c.(map[string]any)
		aggregates := context["aggregates"].([]any)
		for _, a := range aggregates {
			aggregate := a.(map[string]any)
			slices := aggregate["slices"].([]any)
			for _, s := range slices {
				slice := s.(map[string]any)
				if slice["name"] == name {
					return slice
				}
			}
		}
	}
	t.Fatalf("slice %q not found", name)
	return nil
}

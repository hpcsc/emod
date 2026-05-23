//go:build unit

package cli_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/hpcsc/emod/internal/cli"
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
	t.Run("valid file outputs model JSON to stdout", func(t *testing.T) {
		path := writeTemp(t, "valid.emod", validEmod)

		output := captureStdout(t, func() {
			err := cli.RunExport(path, "json")
			require.NoError(t, err)
		})

		var doc map[string]interface{}
		err := json.Unmarshal([]byte(output), &doc)
		require.NoError(t, err)
		require.Equal(t, "Hotel Reservation", doc["name"])
	})

	t.Run("valid file output includes actors and contexts", func(t *testing.T) {
		path := writeTemp(t, "valid.emod", validEmod)

		output := captureStdout(t, func() {
			err := cli.RunExport(path, "json")
			require.NoError(t, err)
		})

		var doc map[string]interface{}
		err := json.Unmarshal([]byte(output), &doc)
		require.NoError(t, err)

		actors, ok := doc["actors"].([]interface{})
		require.True(t, ok, "expected actors in output")
		require.Len(t, actors, 1)
		require.Equal(t, "Guest", actors[0].(map[string]interface{})["name"])

		_, ok = doc["contexts"].([]interface{})
		require.True(t, ok, "expected contexts in output")
	})

	t.Run("file with validation errors writes diagnostics to stderr", func(t *testing.T) {
		path := writeTemp(t, "invalid.emod", invalidEmod)

		var err error
		stderr := captureStderr(t, func() {
			err = cli.RunExport(path, "json")
		})

		require.Error(t, err)
		var lintErr *cli.LintError
		if errors.As(err, &lintErr) {
			require.Equal(t, "", lintErr.Message)
			require.Equal(t, 2, lintErr.ExitCode)
		}
		require.Contains(t, stderr, path)
		require.Contains(t, stderr, ":1:")
	})

	t.Run("file with only lint warnings writes diagnostics to stderr with non-zero exit", func(t *testing.T) {
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
		stderr := captureStderr(t, func() {
			err = cli.RunExport(path, "json")
		})

		require.Error(t, err)
		var lintErr *cli.LintError
		if errors.As(err, &lintErr) {
			require.Equal(t, 1, lintErr.ExitCode)
			require.Equal(t, "", lintErr.Message)
		}
		require.Contains(t, stderr, "state-obsession")
	})

	t.Run("unparseable file writes diagnostics to stderr with non-zero exit", func(t *testing.T) {
		path := writeTemp(t, "invalid.emod", invalidEmod)

		var err error
		stderr := captureStderr(t, func() {
			err = cli.RunExport(path, "json")
		})

		require.Error(t, err)
		var lintErr *cli.LintError
		if errors.As(err, &lintErr) {
			require.Equal(t, "", lintErr.Message)
			require.Equal(t, 2, lintErr.ExitCode)
		}
		require.Contains(t, stderr, path)
		require.Contains(t, stderr, ":1:")
	})

	t.Run("missing file argument returns error", func(t *testing.T) {
		err := cli.RunExport("", "json")

		require.Error(t, err)
		require.Equal(t, "export requires exactly one file argument", err.Error())
	})

	t.Run("nonexistent file returns error", func(t *testing.T) {
		err := cli.RunExport("/tmp/nonexistent-export-file-abc123.emod", "json")

		require.Error(t, err)
	})

	t.Run("unsupported format returns error", func(t *testing.T) {
		path := writeTemp(t, "valid.emod", validEmod)

		err := cli.RunExport(path, "text")

		require.Error(t, err)
		require.Contains(t, err.Error(), "unsupported format")
		var lintErr *cli.LintError
		if errors.As(err, &lintErr) {
			require.Equal(t, 1, lintErr.ExitCode)
		}
	})

	t.Run("default format is json and produces valid output", func(t *testing.T) {
		path := writeTemp(t, "valid.emod", validEmod)

		output := captureStdout(t, func() {
			err := cli.RunExport(path, "json")
			require.NoError(t, err)
		})

		require.True(t, json.Valid([]byte(output)))
	})
}

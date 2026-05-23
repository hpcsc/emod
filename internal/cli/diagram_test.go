//go:build unit

package cli_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hpcsc/emod/internal/cli"
	"github.com/stretchr/testify/require"
)

func TestDiagram(t *testing.T) {
	t.Run("valid file uses default output path", func(t *testing.T) {
		path := writeTemp(t, "valid.emod", validEmod)
		defaultOutput := path[:len(path)-len(".emod")] + ".drawio"

		err := cli.RunDiagram(path, "")
		require.NoError(t, err)

		_, statErr := os.Stat(defaultOutput)
		require.NoError(t, statErr, "expected .drawio file to exist at default path")
		_ = os.Remove(defaultOutput)
	})

	t.Run("valid file uses custom -o output path", func(t *testing.T) {
		path := writeTemp(t, "valid.emod", validEmod)
		customOutput := filepath.Join(t.TempDir(), "custom.drawio")

		err := cli.RunDiagram(path, customOutput)
		require.NoError(t, err)

		_, statErr := os.Stat(customOutput)
		require.NoError(t, statErr, "expected .drawio file to exist at custom path")
	})

	t.Run("validation errors produce no .drawio file and exit code 2", func(t *testing.T) {
		path := writeTemp(t, "invalid.emod", invalidEmod)
		defaultOutput := path[:len(path)-len(".emod")] + ".drawio"

		var err error
		stderr := captureStderr(t, func() {
			err = cli.RunDiagram(path, "")
		})

		require.Error(t, err)
		var lintErr *cli.LintError
		if errors.As(err, &lintErr) {
			require.Equal(t, 2, lintErr.ExitCode)
		}

		_, statErr := os.Stat(defaultOutput)
		require.True(t, os.IsNotExist(statErr), "expected no .drawio file to be created")
		require.Contains(t, stderr, path)
		require.Contains(t, stderr, ":1:")
	})

	t.Run("lint warnings produce .drawio with exit code 1", func(t *testing.T) {
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
		defaultOutput := path[:len(path)-len(".emod")] + ".drawio"

		var err error
		stderr := captureStderr(t, func() {
			err = cli.RunDiagram(path, "")
		})

		require.Error(t, err)
		var lintErr *cli.LintError
		if errors.As(err, &lintErr) {
			require.Equal(t, 1, lintErr.ExitCode)
		}

		_, statErr := os.Stat(defaultOutput)
		require.NoError(t, statErr, "expected .drawio to be created despite warnings")
		require.Contains(t, stderr, "state-obsession")
		_ = os.Remove(defaultOutput)
	})

	t.Run("missing file argument returns error", func(t *testing.T) {
		err := cli.RunDiagram("", "")

		require.Error(t, err)
		require.Equal(t, "diagram requires exactly one file argument", err.Error())
	})

	t.Run("nonexistent file returns error", func(t *testing.T) {
		err := cli.RunDiagram("/tmp/nonexistent-diagram-file-abc123.emod", "")

		require.Error(t, err)
	})

	t.Run("output file is well-formed draw.io XML", func(t *testing.T) {
		path := writeTemp(t, "valid.emod", validEmod)
		defaultOutput := path[:len(path)-len(".emod")] + ".drawio"
		defer os.Remove(defaultOutput)

		err := cli.RunDiagram(path, "")
		require.NoError(t, err)

		content, err := os.ReadFile(defaultOutput)
		require.NoError(t, err)

		xml := string(content)
		require.True(t, strings.HasPrefix(xml, `<?xml version="1.0"`), "expected XML declaration")
		require.Contains(t, xml, "<mxfile")
		require.Contains(t, xml, "</mxfile>")
	})

	t.Run("custom -o path with nested directories creates the directory", func(t *testing.T) {
		path := writeTemp(t, "valid.emod", validEmod)
		customOutput := filepath.Join(t.TempDir(), "nested", "dir", "out.drawio")

		err := cli.RunDiagram(path, customOutput)
		require.NoError(t, err)

		_, statErr := os.Stat(customOutput)
		require.NoError(t, statErr, "expected .drawio file to be created in nested directory")
	})
}

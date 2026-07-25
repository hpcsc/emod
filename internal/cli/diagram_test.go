//go:build unit

package cli_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hpcsc/emod/internal/cli"
	"github.com/hpcsc/emod/internal/diagram"
	"github.com/stretchr/testify/require"
)

func TestDiagram(t *testing.T) {
	t.Run("valid file uses default output path", func(t *testing.T) {
		path := writeTemp(t, "valid.emod", validEmod)
		defaultOutput := path[:len(path)-len(".emod")] + ".drawio"

		err := cli.RunDiagram(path, "", "drawio", diagram.StyleAuto)
		require.NoError(t, err)

		_, statErr := os.Stat(defaultOutput)
		require.NoError(t, statErr, "expected .drawio file to exist at default path")
		_ = os.Remove(defaultOutput)
	})

	t.Run("valid file uses custom -o output path", func(t *testing.T) {
		path := writeTemp(t, "valid.emod", validEmod)
		customOutput := filepath.Join(t.TempDir(), "custom.drawio")

		err := cli.RunDiagram(path, customOutput, "drawio", diagram.StyleAuto)
		require.NoError(t, err)

		_, statErr := os.Stat(customOutput)
		require.NoError(t, statErr, "expected .drawio file to exist at custom path")
	})

	t.Run("validation errors produce no .drawio file and exit code 2", func(t *testing.T) {
		path := writeTemp(t, "invalid.emod", invalidEmod)
		defaultOutput := path[:len(path)-len(".emod")] + ".drawio"

		var err error
		stderr := captureStderr(t, func() {
			err = cli.RunDiagram(path, "", "drawio", diagram.StyleAuto)
		})

		require.Error(t, err)
		var lintErr *cli.LintError
		require.True(t, errors.As(err, &lintErr))
		require.Equal(t, 2, lintErr.ExitCode)

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
			err = cli.RunDiagram(path, "", "drawio", diagram.StyleAuto)
		})

		require.Error(t, err)
		var lintErr *cli.LintError
		require.True(t, errors.As(err, &lintErr))
		require.Equal(t, 1, lintErr.ExitCode)

		_, statErr := os.Stat(defaultOutput)
		require.NoError(t, statErr, "expected .drawio to be created despite warnings")
		require.Contains(t, stderr, "state-obsession")
		_ = os.Remove(defaultOutput)
	})

	t.Run("missing file argument returns error", func(t *testing.T) {
		err := cli.RunDiagram("", "", "drawio", diagram.StyleAuto)

		require.ErrorIs(t, err, cli.ErrMissingFileArgument)
	})

	t.Run("nonexistent file returns error naming the file", func(t *testing.T) {
		missing := filepath.Join(t.TempDir(), "nonexistent.emod")

		err := cli.RunDiagram(missing, "", "drawio", diagram.StyleAuto)

		require.Error(t, err)
		require.Contains(t, err.Error(), missing)
	})

	t.Run("output file is well-formed draw.io XML", func(t *testing.T) {
		path := writeTemp(t, "valid.emod", validEmod)
		defaultOutput := path[:len(path)-len(".emod")] + ".drawio"
		defer os.Remove(defaultOutput)

		err := cli.RunDiagram(path, "", "drawio", diagram.StyleAuto)
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

		err := cli.RunDiagram(path, customOutput, "drawio", diagram.StyleAuto)
		require.NoError(t, err)

		_, statErr := os.Stat(customOutput)
		require.NoError(t, statErr, "expected .drawio file to be created in nested directory")
	})

	t.Run("mermaid output is printed to stdout and starts with eventmodeling", func(t *testing.T) {
		path := writeTemp(t, "valid.emod", validEmod)

		output := captureStdout(t, func() {
			err := cli.RunDiagram(path, "", "mermaid", diagram.StyleAuto)
			require.NoError(t, err)
		})

		require.True(t, strings.HasPrefix(output, "eventmodeling"), "mermaid output should start with eventmodeling")
	})

	t.Run("mermaid output can be written to a specific file", func(t *testing.T) {
		path := writeTemp(t, "valid.emod", validEmod)
		outputPath := filepath.Join(t.TempDir(), "output.mermaid")

		err := cli.RunDiagram(path, outputPath, "mermaid", diagram.StyleAuto)
		require.NoError(t, err)

		content, err := os.ReadFile(outputPath)
		require.NoError(t, err)
		require.True(t, strings.HasPrefix(string(content), "eventmodeling"))
	})

	t.Run("ascii output is printed to stdout and starts with Model header", func(t *testing.T) {
		path := writeTemp(t, "valid.emod", validEmod)

		output := captureStdout(t, func() {
			err := cli.RunDiagram(path, "", "ascii", diagram.StyleAuto)
			require.NoError(t, err)
		})

		require.True(t, strings.HasPrefix(output, "Model:"), "ascii output should start with Model header")
		require.Contains(t, output, "=== Slice:")
		require.Contains(t, output, "->")
	})

	t.Run("ascii output can be written to a specific file", func(t *testing.T) {
		path := writeTemp(t, "valid.emod", validEmod)
		outputPath := filepath.Join(t.TempDir(), "output.ascii")

		err := cli.RunDiagram(path, outputPath, "ascii", diagram.StyleAuto)
		require.NoError(t, err)

		content, err := os.ReadFile(outputPath)
		require.NoError(t, err)
		require.True(t, strings.HasPrefix(string(content), "Model:"))
	})

	t.Run("ascii validation errors produce exit code 2", func(t *testing.T) {
		path := writeTemp(t, "invalid.emod", invalidEmod)

		var err error
		stderr := captureStderr(t, func() {
			err = cli.RunDiagram(path, "", "ascii", diagram.StyleAuto)
		})

		require.Error(t, err)
		var lintErr *cli.LintError
		require.True(t, errors.As(err, &lintErr))
		require.Equal(t, 2, lintErr.ExitCode)

		require.Contains(t, stderr, path)
		require.Contains(t, stderr, ":1:")
	})

	t.Run("ascii lint warnings produce output with exit code 1", func(t *testing.T) {
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
		output := captureStdout(t, func() {
			err = cli.RunDiagram(path, "", "ascii", diagram.StyleAuto)
		})

		require.Error(t, err)
		var lintErr *cli.LintError
		require.True(t, errors.As(err, &lintErr))
		require.Equal(t, 1, lintErr.ExitCode)

		require.Contains(t, output, "Model:")
		require.Contains(t, output, "=== Slice:")
	})

	t.Run("unsupported diagram format returns error listing supported formats", func(t *testing.T) {
		path := writeTemp(t, "valid.emod", validEmod)

		err := cli.RunDiagram(path, "", "bogus", diagram.StyleAuto)
		require.ErrorIs(t, err, cli.ErrUnsupportedFormat)
		require.Contains(t, err.Error(), "drawio")
		require.Contains(t, err.Error(), "mermaid")
		require.Contains(t, err.Error(), "svg")
		require.Contains(t, err.Error(), "ascii")
	})

	t.Run("svg: valid file uses default .svg output path", func(t *testing.T) {
		path := writeTemp(t, "valid.emod", validEmod)
		defaultOutput := path[:len(path)-len(".emod")] + ".svg"

		err := cli.RunDiagram(path, "", "svg", diagram.StyleAuto)
		require.NoError(t, err)

		_, statErr := os.Stat(defaultOutput)
		require.NoError(t, statErr, "expected .svg file to exist at default path")
		_ = os.Remove(defaultOutput)
	})

	t.Run("svg: valid file uses custom -o output path", func(t *testing.T) {
		path := writeTemp(t, "valid.emod", validEmod)
		customOutput := filepath.Join(t.TempDir(), "custom.svg")

		err := cli.RunDiagram(path, customOutput, "svg", diagram.StyleAuto)
		require.NoError(t, err)

		_, statErr := os.Stat(customOutput)
		require.NoError(t, statErr, "expected .svg file to exist at custom path")
	})

	t.Run("svg: output file is well-formed SVG", func(t *testing.T) {
		path := writeTemp(t, "valid.emod", validEmod)
		defaultOutput := path[:len(path)-len(".emod")] + ".svg"
		defer os.Remove(defaultOutput)

		err := cli.RunDiagram(path, "", "svg", diagram.StyleAuto)
		require.NoError(t, err)

		content, err := os.ReadFile(defaultOutput)
		require.NoError(t, err)

		svg := string(content)
		require.True(t, strings.HasPrefix(svg, `<svg xmlns="http://www.w3.org/2000/svg"`), "expected SVG declaration")
		require.Contains(t, svg, "</svg>")
	})

	t.Run("svg: validation errors produce no .svg file and exit code 2", func(t *testing.T) {
		path := writeTemp(t, "invalid.emod", invalidEmod)
		defaultOutput := path[:len(path)-len(".emod")] + ".svg"

		var err error
		stderr := captureStderr(t, func() {
			err = cli.RunDiagram(path, "", "svg", diagram.StyleAuto)
		})

		require.Error(t, err)
		var lintErr *cli.LintError
		require.True(t, errors.As(err, &lintErr))
		require.Equal(t, 2, lintErr.ExitCode)

		_, statErr := os.Stat(defaultOutput)
		require.True(t, os.IsNotExist(statErr), "expected no .svg file to be created")
		require.Contains(t, stderr, path)
		require.Contains(t, stderr, ":1:")
	})

	t.Run("svg: lint warnings produce .svg with exit code 1", func(t *testing.T) {
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
		defaultOutput := path[:len(path)-len(".emod")] + ".svg"

		var err error
		stderr := captureStderr(t, func() {
			err = cli.RunDiagram(path, "", "svg", diagram.StyleAuto)
		})

		require.Error(t, err)
		var lintErr *cli.LintError
		require.True(t, errors.As(err, &lintErr))
		require.Equal(t, 1, lintErr.ExitCode)

		_, statErr := os.Stat(defaultOutput)
		require.NoError(t, statErr, "expected .svg to be created despite warnings")
		require.Contains(t, stderr, "state-obsession")
		_ = os.Remove(defaultOutput)
	})

	t.Run("svg: custom -o path with nested directories creates the directory", func(t *testing.T) {
		path := writeTemp(t, "valid.emod", validEmod)
		customOutput := filepath.Join(t.TempDir(), "nested", "dir", "out.svg")

		err := cli.RunDiagram(path, customOutput, "svg", diagram.StyleAuto)
		require.NoError(t, err)

		_, statErr := os.Stat(customOutput)
		require.NoError(t, statErr, "expected .svg file to be created in nested directory")
	})
	t.Run("serve", func(t *testing.T) {
		t.Run("serves the parsed model as initial viewer data", func(t *testing.T) {
			path := writeTemp(t, "valid.emod", validEmod)

			addr, output := startDiagramServe(t, path)

			body := getBody(t, addr)
			require.Contains(t, body, "window.INITIAL_DATA = ")
			require.Contains(t, body, "Hotel Reservation")
			require.Empty(t, output.stderr(), "a valid file should produce no diagnostics")
		})

		t.Run("serves the viewer without initial data when no file is given", func(t *testing.T) {
			addr, _ := startDiagramServe(t, "")

			body := getBody(t, addr)
			require.Contains(t, body, "<!DOCTYPE html>")
			require.NotContains(t, body, "window.INITIAL_DATA")
		})

		t.Run("reports diagnostics on stderr and still serves the viewer", func(t *testing.T) {
			path := writeTemp(t, "invalid.emod", invalidEmod)

			addr, output := startDiagramServe(t, path)

			require.Contains(t, output.stderr(), path)
			require.Contains(t, output.stderr(), ":1:")
			require.Contains(t, getBody(t, addr), "<!DOCTYPE html>")
		})
	})
}

var viewerURL = regexp.MustCompile(`http://127\.0\.0\.1:\d+`)

// capturedStreams accumulates what a still-running command writes to the
// standard streams, so tests can read a snapshot before it exits.
type capturedStreams struct {
	mu  sync.Mutex
	out strings.Builder
	err strings.Builder
}

func (c *capturedStreams) stdout() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.out.String()
}

func (c *capturedStreams) stderr() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.err.String()
}

func (c *capturedStreams) write(dst *strings.Builder, p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return dst.Write(p)
}

type streamWriter struct {
	captured *capturedStreams
	dst      *strings.Builder
}

func (w streamWriter) Write(p []byte) (int, error) { return w.captured.write(w.dst, p) }

// startDiagramServe runs the serve command in the background, waits for the
// viewer URL it prints, and cancels it during cleanup. Readiness comes from the
// command's own output rather than a fixed delay.
func startDiagramServe(t *testing.T, path string) (string, *capturedStreams) {
	t.Helper()

	captured := &capturedStreams{}
	restore := redirectStdStreams(t, captured)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- cli.RunDiagramServe(ctx, path, false) }()

	t.Cleanup(func() {
		cancel()
		select {
		case err := <-done:
			require.NoError(t, err)
		case <-time.After(5 * time.Second):
			t.Error("serve did not shut down within 5s of cancellation")
		}
		restore()
	})

	require.Eventually(t, func() bool {
		return viewerURL.MatchString(captured.stdout())
	}, 5*time.Second, 5*time.Millisecond, "serve never printed a viewer URL")

	stdout := captured.stdout()
	require.Contains(t, stdout, "Viewer available at")

	return viewerURL.FindString(stdout), captured
}

// redirectStdStreams points os.Stdout/os.Stderr at captured until the returned
// function restores them and the drain goroutines finish.
func redirectStdStreams(t *testing.T, captured *capturedStreams) func() {
	t.Helper()

	outR, outW, err := os.Pipe()
	require.NoError(t, err)
	errR, errW, err := os.Pipe()
	require.NoError(t, err)

	oldOut, oldErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = outW, errW

	var drained sync.WaitGroup
	drained.Add(2)
	go func() {
		defer drained.Done()
		io.Copy(streamWriter{captured, &captured.out}, outR)
	}()
	go func() {
		defer drained.Done()
		io.Copy(streamWriter{captured, &captured.err}, errR)
	}()

	return func() {
		os.Stdout, os.Stderr = oldOut, oldErr
		outW.Close()
		errW.Close()
		drained.Wait()
	}
}

func getBody(t *testing.T, addr string) string {
	t.Helper()

	resp, err := http.Get(addr)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return string(body)
}

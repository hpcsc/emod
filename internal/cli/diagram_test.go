//go:build unit

package cli_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
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
			err = cli.RunDiagram(path, "", "drawio", diagram.StyleAuto)
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
		err := cli.RunDiagram("", "", "drawio", diagram.StyleAuto)

		require.Error(t, err)
		require.Equal(t, "diagram requires exactly one file argument", err.Error())
	})

	t.Run("nonexistent file returns error", func(t *testing.T) {
		err := cli.RunDiagram("/tmp/nonexistent-diagram-file-abc123.emod", "", "drawio", diagram.StyleAuto)

		require.Error(t, err)
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
		if errors.As(err, &lintErr) {
			require.Equal(t, 2, lintErr.ExitCode)
		}

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
		if errors.As(err, &lintErr) {
			require.Equal(t, 1, lintErr.ExitCode)
		}

		require.Contains(t, output, "Model:")
		require.Contains(t, output, "=== Slice:")
	})

	t.Run("unsupported diagram format returns error listing supported formats", func(t *testing.T) {
		path := writeTemp(t, "valid.emod", validEmod)

		err := cli.RunDiagram(path, "", "bogus", diagram.StyleAuto)
		require.Error(t, err)
		require.Contains(t, err.Error(), "unsupported format")
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
		if errors.As(err, &lintErr) {
			require.Equal(t, 2, lintErr.ExitCode)
		}

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
		if errors.As(err, &lintErr) {
			require.Equal(t, 1, lintErr.ExitCode)
		}

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

	// --- Serve flag ---

	t.Run("serve with valid file starts server with diagram data", func(t *testing.T) {
		path := writeTemp(t, "valid.emod", validEmod)

		rOut, wOut, err := os.Pipe()
		require.NoError(t, err)
		oldOut := os.Stdout
		os.Stdout = wOut

		rErr, wErr, err := os.Pipe()
		require.NoError(t, err)
		oldErr := os.Stderr
		os.Stderr = wErr

		errCh := make(chan error, 1)
		go func() {
			errCh <- cli.RunDiagramServe(path, false)
		}()

		time.Sleep(200 * time.Millisecond)

		wOut.Close()
		os.Stdout = oldOut
		wErr.Close()
		os.Stderr = oldErr

		var stdout, stderr bytes.Buffer
		_, _ = io.Copy(&stdout, rOut)
		_, _ = io.Copy(&stderr, rErr)

		re := regexp.MustCompile(`http://127\.0\.0\.1:(\d+)`)
		matches := re.FindStringSubmatch(stdout.String())
		require.Len(t, matches, 2, "stdout should contain viewer URL")
		require.Contains(t, stdout.String(), "Viewer available at")

		addr := fmt.Sprintf("http://127.0.0.1:%s", matches[1])

		resp, getErr := http.Get(addr)
		require.NoError(t, getErr)
		body, readErr := io.ReadAll(resp.Body)
		require.NoError(t, readErr)
		resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.Contains(t, string(body), "window.INITIAL_DATA = ")

		// Stderr should be clean for a valid file
		require.Empty(t, stderr.String())

		// Signal shutdown
		syscall.Kill(syscall.Getpid(), syscall.SIGINT)

		select {
		case serveErr := <-errCh:
			require.NoError(t, serveErr)
		case <-time.After(2 * time.Second):
			t.Fatal("server did not shut down within 2s")
		}
	})

	t.Run("serve without file starts server without initial data", func(t *testing.T) {
		rOut, wOut, err := os.Pipe()
		require.NoError(t, err)
		oldOut := os.Stdout
		os.Stdout = wOut

		errCh := make(chan error, 1)
		go func() {
			errCh <- cli.RunDiagramServe("", false)
		}()

		time.Sleep(200 * time.Millisecond)

		wOut.Close()
		os.Stdout = oldOut

		var stdout bytes.Buffer
		_, _ = io.Copy(&stdout, rOut)

		re := regexp.MustCompile(`http://127\.0\.0\.1:(\d+)`)
		matches := re.FindStringSubmatch(stdout.String())
		require.Len(t, matches, 2, "stdout should contain viewer URL")

		addr := fmt.Sprintf("http://127.0.0.1:%s", matches[1])

		resp, getErr := http.Get(addr)
		require.NoError(t, getErr)
		body, readErr := io.ReadAll(resp.Body)
		require.NoError(t, readErr)
		resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.NotContains(t, string(body), "window.INITIAL_DATA")

		// Signal shutdown
		syscall.Kill(syscall.Getpid(), syscall.SIGINT)

		select {
		case serveErr := <-errCh:
			require.NoError(t, serveErr)
		case <-time.After(2 * time.Second):
			t.Fatal("server did not shut down within 2s")
		}
	})

	t.Run("serve with invalid file prints diagnostics and starts server", func(t *testing.T) {
		path := writeTemp(t, "invalid.emod", invalidEmod)

		rOut, wOut, err := os.Pipe()
		require.NoError(t, err)
		oldOut := os.Stdout
		os.Stdout = wOut

		rErr, wErr, err := os.Pipe()
		require.NoError(t, err)
		oldErr := os.Stderr
		os.Stderr = wErr

		errCh := make(chan error, 1)
		go func() {
			errCh <- cli.RunDiagramServe(path, false)
		}()

		time.Sleep(200 * time.Millisecond)

		wOut.Close()
		os.Stdout = oldOut
		wErr.Close()
		os.Stderr = oldErr

		var stdout, stderr bytes.Buffer
		_, _ = io.Copy(&stdout, rOut)
		_, _ = io.Copy(&stderr, rErr)

		// Stderr should have diagnostics for invalid file
		require.Contains(t, stderr.String(), path)

		// Server should still start
		re := regexp.MustCompile(`http://127\.0\.0\.1:(\d+)`)
		matches := re.FindStringSubmatch(stdout.String())
		require.Len(t, matches, 2, "stdout should contain viewer URL despite errors")

		addr := fmt.Sprintf("http://127.0.0.1:%s", matches[1])

		resp, getErr := http.Get(addr)
		require.NoError(t, getErr)
		body, readErr := io.ReadAll(resp.Body)
		require.NoError(t, readErr)
		resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		// Even with invalid data, the server should start (window.INITIAL_DATA may or may not be present)
		require.Contains(t, string(body), "<!DOCTYPE html>")

		// Signal shutdown
		syscall.Kill(syscall.Getpid(), syscall.SIGINT)

		select {
		case serveErr := <-errCh:
			require.NoError(t, serveErr)
		case <-time.After(2 * time.Second):
			t.Fatal("server did not shut down within 2s")
		}
	})
}

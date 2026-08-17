//go:build unit

package cli_test

import (
	"context"
	"errors"
	"fmt"
	"html"
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
	"github.com/hpcsc/emod/internal/formatter"
	"github.com/hpcsc/emod/internal/test"
	"github.com/stretchr/testify/require"
	urfave "github.com/urfave/cli/v2"
)

// warningEmod is a model the linter warns about and the validator accepts, so a
// command run over it still writes its output and still exits non-zero.
const warningEmod = `model "Test"
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

func TestDiagram(t *testing.T) {
	t.Run("valid file uses default output path", func(t *testing.T) {
		path := writeTemp(t, "valid.emod", validEmod)
		defaultOutput := path[:len(path)-len(".emod")] + ".drawio"

		err := cli.RunDiagram(path, "", "drawio", diagram.StyleAuto, false)
		require.NoError(t, err)

		_, statErr := os.Stat(defaultOutput)
		require.NoError(t, statErr, "expected .drawio file to exist at default path")
		_ = os.Remove(defaultOutput)
	})

	t.Run("valid file uses custom -o output path", func(t *testing.T) {
		path := writeTemp(t, "valid.emod", validEmod)
		customOutput := filepath.Join(t.TempDir(), "custom.drawio")

		err := cli.RunDiagram(path, customOutput, "drawio", diagram.StyleAuto, false)
		require.NoError(t, err)

		_, statErr := os.Stat(customOutput)
		require.NoError(t, statErr, "expected .drawio file to exist at custom path")
	})

	t.Run("validation errors produce no .drawio file and exit code 2", func(t *testing.T) {
		path := writeTemp(t, "invalid.emod", invalidEmod)
		defaultOutput := path[:len(path)-len(".emod")] + ".drawio"

		var err error
		stderr := captureStderr(t, func() {
			err = cli.RunDiagram(path, "", "drawio", diagram.StyleAuto, false)
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
		path := writeTemp(t, "warnings.emod", warningEmod)
		defaultOutput := path[:len(path)-len(".emod")] + ".drawio"

		var err error
		stderr := captureStderr(t, func() {
			err = cli.RunDiagram(path, "", "drawio", diagram.StyleAuto, false)
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
		err := cli.RunDiagram("", "", "drawio", diagram.StyleAuto, false)

		require.ErrorIs(t, err, cli.ErrMissingFileArgument)
	})

	t.Run("nonexistent file returns error naming the file", func(t *testing.T) {
		missing := filepath.Join(t.TempDir(), "nonexistent.emod")

		err := cli.RunDiagram(missing, "", "drawio", diagram.StyleAuto, false)

		require.Error(t, err)
		require.Contains(t, err.Error(), missing)
	})

	t.Run("output file is well-formed draw.io XML", func(t *testing.T) {
		path := writeTemp(t, "valid.emod", validEmod)
		defaultOutput := path[:len(path)-len(".emod")] + ".drawio"
		defer os.Remove(defaultOutput)

		err := cli.RunDiagram(path, "", "drawio", diagram.StyleAuto, false)
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

		err := cli.RunDiagram(path, customOutput, "drawio", diagram.StyleAuto, false)
		require.NoError(t, err)

		_, statErr := os.Stat(customOutput)
		require.NoError(t, statErr, "expected .drawio file to be created in nested directory")
	})

	t.Run("mermaid output is printed to stdout and starts with eventmodeling", func(t *testing.T) {
		path := writeTemp(t, "valid.emod", validEmod)

		output := captureStdout(t, func() {
			err := cli.RunDiagram(path, "", "mermaid", diagram.StyleAuto, false)
			require.NoError(t, err)
		})

		require.True(t, strings.HasPrefix(output, "eventmodeling"), "mermaid output should start with eventmodeling")
	})

	t.Run("mermaid output can be written to a specific file", func(t *testing.T) {
		path := writeTemp(t, "valid.emod", validEmod)
		outputPath := filepath.Join(t.TempDir(), "output.mermaid")

		err := cli.RunDiagram(path, outputPath, "mermaid", diagram.StyleAuto, false)
		require.NoError(t, err)

		content, err := os.ReadFile(outputPath)
		require.NoError(t, err)
		require.True(t, strings.HasPrefix(string(content), "eventmodeling"))
	})

	t.Run("ascii output is printed to stdout and starts with Model header", func(t *testing.T) {
		path := writeTemp(t, "valid.emod", validEmod)

		output := captureStdout(t, func() {
			err := cli.RunDiagram(path, "", "ascii", diagram.StyleAuto, false)
			require.NoError(t, err)
		})

		require.True(t, strings.HasPrefix(output, "Model:"), "ascii output should start with Model header")
		require.Contains(t, output, "=== Slice:")
		require.Contains(t, output, "->")
	})

	t.Run("ascii output can be written to a specific file", func(t *testing.T) {
		path := writeTemp(t, "valid.emod", validEmod)
		outputPath := filepath.Join(t.TempDir(), "output.ascii")

		err := cli.RunDiagram(path, outputPath, "ascii", diagram.StyleAuto, false)
		require.NoError(t, err)

		content, err := os.ReadFile(outputPath)
		require.NoError(t, err)
		require.True(t, strings.HasPrefix(string(content), "Model:"))
	})

	t.Run("ascii validation errors produce exit code 2", func(t *testing.T) {
		path := writeTemp(t, "invalid.emod", invalidEmod)

		var err error
		stderr := captureStderr(t, func() {
			err = cli.RunDiagram(path, "", "ascii", diagram.StyleAuto, false)
		})

		require.Error(t, err)
		var lintErr *cli.LintError
		require.True(t, errors.As(err, &lintErr))
		require.Equal(t, 2, lintErr.ExitCode)

		require.Contains(t, stderr, path)
		require.Contains(t, stderr, ":1:")
	})

	t.Run("ascii lint warnings produce output with exit code 1", func(t *testing.T) {
		path := writeTemp(t, "warnings.emod", warningEmod)

		var err error
		output := captureStdout(t, func() {
			err = cli.RunDiagram(path, "", "ascii", diagram.StyleAuto, false)
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

		err := cli.RunDiagram(path, "", "bogus", diagram.StyleAuto, false)
		require.ErrorIs(t, err, cli.ErrUnsupportedFormat)
		require.Contains(t, err.Error(), "drawio")
		require.Contains(t, err.Error(), "mermaid")
		require.Contains(t, err.Error(), "svg")
		require.Contains(t, err.Error(), "ascii")
	})

	t.Run("svg: valid file uses default .svg output path", func(t *testing.T) {
		path := writeTemp(t, "valid.emod", validEmod)
		defaultOutput := path[:len(path)-len(".emod")] + ".svg"

		err := cli.RunDiagram(path, "", "svg", diagram.StyleAuto, false)
		require.NoError(t, err)

		_, statErr := os.Stat(defaultOutput)
		require.NoError(t, statErr, "expected .svg file to exist at default path")
		_ = os.Remove(defaultOutput)
	})

	t.Run("svg: valid file uses custom -o output path", func(t *testing.T) {
		path := writeTemp(t, "valid.emod", validEmod)
		customOutput := filepath.Join(t.TempDir(), "custom.svg")

		err := cli.RunDiagram(path, customOutput, "svg", diagram.StyleAuto, false)
		require.NoError(t, err)

		_, statErr := os.Stat(customOutput)
		require.NoError(t, statErr, "expected .svg file to exist at custom path")
	})

	t.Run("svg: output file is well-formed SVG", func(t *testing.T) {
		path := writeTemp(t, "valid.emod", validEmod)
		defaultOutput := path[:len(path)-len(".emod")] + ".svg"
		defer os.Remove(defaultOutput)

		err := cli.RunDiagram(path, "", "svg", diagram.StyleAuto, false)
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
			err = cli.RunDiagram(path, "", "svg", diagram.StyleAuto, false)
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
		path := writeTemp(t, "warnings.emod", warningEmod)
		defaultOutput := path[:len(path)-len(".emod")] + ".svg"

		var err error
		stderr := captureStderr(t, func() {
			err = cli.RunDiagram(path, "", "svg", diagram.StyleAuto, false)
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

		err := cli.RunDiagram(path, customOutput, "svg", diagram.StyleAuto, false)
		require.NoError(t, err)

		_, statErr := os.Stat(customOutput)
		require.NoError(t, statErr, "expected .svg file to be created in nested directory")
	})
	t.Run("spec cards", func(t *testing.T) {
		t.Run("writes a draw.io file naming every scenario the model states", func(t *testing.T) {
			path := writeTemp(t, "specs.emod", specStatingEmod(t))
			output := path[:len(path)-len(".emod")] + ".drawio"
			defer os.Remove(output)

			require.NoError(t, cli.RunDiagram(path, "", "drawio", diagram.StyleAuto, true))

			requireNamesEveryScenario(t, readFileContent(t, output))
		})

		t.Run("writes an svg naming every scenario the model states", func(t *testing.T) {
			path := writeTemp(t, "specs.emod", specStatingEmod(t))
			output := path[:len(path)-len(".emod")] + ".svg"
			defer os.Remove(output)

			require.NoError(t, cli.RunDiagram(path, "", "svg", diagram.StyleAuto, true))

			requireNamesEveryScenario(t, readFileContent(t, output))
		})

		t.Run("without the flag, a model stating scenarios writes what its spec-less twin writes", func(t *testing.T) {
			stated, unstated := specStatingEmod(t), specLessEmod(t)
			require.NotEqual(t, stated, unstated,
				"the twin has to lose the specs, or the comparison below says nothing")

			for _, format := range []string{"drawio", "svg", "mermaid", "ascii"} {
				t.Run(format, func(t *testing.T) {
					require.Equal(t,
						diagramWritten(t, unstated, format),
						diagramWritten(t, stated, format),
						"a scenario reaches no diagram unless --specs asks for it")
				})
			}
		})

		for _, format := range []string{"mermaid", "ascii"} {
			t.Run("refuses "+format+", which draws no card, rather than writing one without them", func(t *testing.T) {
				path := writeTemp(t, "specs.emod", specStatingEmod(t))

				var err error
				printed := captureStdout(t, func() {
					err = cli.RunDiagram(path, "", format, diagram.StyleAuto, true)
				})

				require.ErrorIs(t, err, cli.ErrUnsupportedFormat)
				var lintErr *cli.LintError
				require.True(t, errors.As(err, &lintErr))
				require.Equal(t, 1, lintErr.ExitCode)
				require.Contains(t, err.Error(), "--specs")
				require.Contains(t, err.Error(), "drawio")
				require.Contains(t, err.Error(), "svg")
				require.Contains(t, err.Error(), format)

				require.Empty(t, printed, "a refused render prints no diagram")
			})
		}

		t.Run("refuses --serve, which draws no card, and starts no server", func(t *testing.T) {
			path := writeTemp(t, "specs.emod", specStatingEmod(t))

			// Returning promptly is the receipt that no server started, and it
			// is bounded rather than awaited: RunDiagramServe blocks until it is
			// signalled, so a run that reached it would hang the suite until the
			// whole binary timed out instead of failing here.
			var err error
			stderr := captureStderr(t, func() {
				returned := make(chan error, 1)
				go func() {
					returned <- runCommandLine(t, "emod", "diagram", "--specs", "--serve", path)
				}()

				select {
				case err = <-returned:
				case <-time.After(10 * time.Second):
					require.FailNow(t, "the command never returned",
						"--specs must be refused before any server is started")
				}
			})

			var exitErr urfave.ExitCoder
			require.True(t, errors.As(err, &exitErr), "%v", err)
			require.Equal(t, 1, exitErr.ExitCode())

			// reportExitError prints a LintError's wording and returns an
			// exit code carrying none, so stderr is where the refusal reads.
			require.Contains(t, stderr, "--specs")
			require.Contains(t, stderr, "drawio")
			require.Contains(t, stderr, "svg")
			require.Contains(t, stderr, "--serve")
		})

		for _, specs := range []bool{false, true} {
			t.Run(fmt.Sprintf("a model the pipeline rejects still exits 2 and writes nothing, specs=%t", specs), func(t *testing.T) {
				path := writeTemp(t, "invalid.emod", invalidEmod)

				// mermaid is the format --specs is refused for, so this is the
				// pairing that pins the order: the diagnostics gate runs first,
				// and a refusal hoisted above it would answer 1 instead of 2.
				var err error
				captureStderr(t, func() {
					err = cli.RunDiagram(path, "", "mermaid", diagram.StyleAuto, specs)
				})

				var lintErr *cli.LintError
				require.True(t, errors.As(err, &lintErr))
				require.Equal(t, 2, lintErr.ExitCode, "asking for cards must not turn a rejected model into a refusal")
				require.NotErrorIs(t, err, cli.ErrSpecCardsUnsupported,
					"a model the pipeline rejects is reported as rejected, not as a surface that draws no card")
				requireNoDiagramWritten(t, path)
			})

			t.Run(fmt.Sprintf("a model the linter warns about still writes its diagram and exits 1, specs=%t", specs), func(t *testing.T) {
				path := writeTemp(t, "warnings.emod", warningEmod)
				output := path[:len(path)-len(".emod")] + ".drawio"

				var err error
				captureStderr(t, func() {
					err = cli.RunDiagram(path, "", "drawio", diagram.StyleAuto, specs)
				})

				var lintErr *cli.LintError
				require.True(t, errors.As(err, &lintErr))
				require.Equal(t, 1, lintErr.ExitCode)
				require.FileExists(t, output, "a warning does not stop the diagram being written")
				_ = os.Remove(output)
			})
		}

		t.Run("the flag reaches the exporters through the command line, not only through RunDiagram", func(t *testing.T) {
			path := writeTemp(t, "specs.emod", specStatingEmod(t))
			output := path[:len(path)-len(".emod")] + ".drawio"
			defer os.Remove(output)

			// Driving the real command line is what pins the argument the
			// action passes: RunDiagram called directly is green either way.
			captureStderr(t, func() {
				_ = runCommandLine(t, "emod", "diagram", "--specs", path)
			})

			requireNamesEveryScenario(t, readFileContent(t, output))
		})

		t.Run("the flag is listed in the command's help, saying what it draws", func(t *testing.T) {
			help := captureStdout(t, func() {
				_ = runCommandLine(t, "emod", "diagram", "--help")
			})

			require.Contains(t, help, "--specs")
			require.Contains(t, help, "Given-When-Then card")
		})
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

// specStatingEmod and specLessEmod are one model written twice, differing only
// in whether its slices state their scenarios. Both go through the formatter so
// the pair differ by the spec blocks alone rather than by layout, which is what
// makes a byte comparison of their diagrams mean something.
func specStatingEmod(t *testing.T) string {
	t.Helper()

	model := test.SlicePatternLibraryLendingModel(t)
	require.Equal(t, test.SlicePatternLibraryLendingSpecNames, test.DeclaredSpecNames(model))

	return formatter.Format(model)
}

func specLessEmod(t *testing.T) string {
	t.Helper()

	unstated := test.WithoutSpecs(test.SlicePatternLibraryLendingModel(t))
	require.Empty(t, test.DeclaredSpecNames(unstated),
		"the twin has to lose the specs of both slice homes, or a comparison against it is answered by whichever home it kept")

	return formatter.Format(unstated)
}

// diagramWritten renders source in format and returns what the command wrote,
// from a file for the picture formats and from stdout for the text ones.
func diagramWritten(t *testing.T, source, format string) string {
	t.Helper()

	path := writeTemp(t, "model.emod", source)

	if format == "mermaid" || format == "ascii" {
		var err error
		printed := captureStdout(t, func() {
			err = cli.RunDiagram(path, "", format, diagram.StyleAuto, false)
		})
		requireDiagramWasWritten(t, err)
		return printed
	}

	output := filepath.Join(t.TempDir(), "diagram."+format)
	requireDiagramWasWritten(t, cli.RunDiagram(path, output, format, diagram.StyleAuto, false))

	return readFileContent(t, output)
}

// requireDiagramWasWritten accepts the warnings exit alongside a clean one. A
// model that loses its specs orphans the invariants it exercised only through
// them, so the spec-less twin reports spec/invariant-never-exercised and exits 1
// while still writing its diagram; only exit 2 means nothing was written.
func requireDiagramWasWritten(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		return
	}

	var lintErr *cli.LintError
	require.True(t, errors.As(err, &lintErr), "%v", err)
	require.Equal(t, 1, lintErr.ExitCode, "a diagram is written for a warning, never for an error")
}

var (
	diagramMarkup     = regexp.MustCompile(`<[^>]*>`)
	diagramWhitespace = regexp.MustCompile(`\s+`)
)

// requireNamesEveryScenario checks a written diagram names every scenario the
// fixture states. Markup, escaping and the line break each format inserts all
// become spaces first, so a name the writer had to wrap still reads as written.
func requireNamesEveryScenario(t *testing.T, written string) {
	t.Helper()

	text := written
	if strings.Contains(text, "<svg") {
		// SVG draws a card's text between tags, one tspan per line, so the tags
		// have to go for the lines to read as one string. draw.io writes the
		// same text inside an attribute, where stripping tags would take it too.
		text = diagramMarkup.ReplaceAllString(text, " ")
	}
	text = html.UnescapeString(text)
	text = diagramWhitespace.ReplaceAllString(strings.ReplaceAll(text, `\n`, " "), " ")

	for _, name := range test.SlicePatternLibraryLendingSpecNames {
		require.Contains(t, text, name, "the diagram has to name every scenario, wrapped or not")
	}
}

func readFileContent(t *testing.T, path string) string {
	t.Helper()

	raw, err := os.ReadFile(path)
	require.NoError(t, err)

	return string(raw)
}

// requireNoDiagramWritten fails when the command left a diagram beside the model
// it refused to render, in either of the two names it defaults to.
func requireNoDiagramWritten(t *testing.T, modelPath string) {
	t.Helper()

	stem := modelPath[:len(modelPath)-len(".emod")]
	for _, extension := range []string{".drawio", ".svg"} {
		require.NoFileExists(t, stem+extension)
	}
}

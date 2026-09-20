package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"syscall"

	"github.com/hpcsc/emod/internal/diagnostic"
	"github.com/hpcsc/emod/internal/diagram"
	"github.com/hpcsc/emod/internal/export"
	"github.com/hpcsc/emod/internal/oracle"
	"github.com/hpcsc/emod/internal/viewer"
)

// diagramFormats are the formats the command draws, and specCardFormats those
// of them that draw a spec card.
var (
	diagramFormats  = []string{"drawio", "mermaid", "svg", "ascii", "event-flow", "event-flow-mermaid"}
	specCardFormats = []string{"drawio", "svg"}
)

// stdoutFormats are the formats that print to stdout when no output path is
// given, the rest being pictures written to a file.
var stdoutFormats = []string{"mermaid", "ascii", "event-flow-mermaid"}

// RunDiagram reads the file at path, lexes and parses it, validates and lints,
// generates a diagram in the requested format, and writes it.
// Supported formats: "drawio" (default), "mermaid", "svg", "ascii",
// "event-flow" and "event-flow-mermaid".
// For drawio, svg and event-flow: output is written to a file; if outputPath is
// empty it defaults to .drawio, .svg or .event-flow.svg.
// For mermaid, ascii and event-flow-mermaid: output goes to stdout unless
// outputPath is specified.
// Errors produce diagnostics on stderr and a non-zero exit code.
// Lint warnings still produce the diagram but with exit code 1.
// style controls the layout strategy (auto, projected, dcb).
// specs draws each slice's scenarios as a Given-When-Then card; only drawio and
// svg render one, and asking for cards in another format is refused rather than
// ignored, so a script cannot quietly write a diagram missing what it asked for.
func RunDiagram(path, outputPath, format string, style diagram.Style, specs bool) error {
	source, err := readSourceFile("diagram", path)
	if err != nil {
		return err
	}

	model, diagnostics := oracle.Run(source, path)

	// Check for errors that prevent diagram generation
	hasErrors := false
	hasWarnings := false
	for _, d := range diagnostics {
		if d.Severity == diagnostic.Error {
			hasErrors = true
		} else {
			hasWarnings = true
		}
		fmt.Fprintln(os.Stderr, d.String())
	}

	// Errors: no diagram output, exit code 2
	if hasErrors {
		return &LintError{
			Message:  "",
			ExitCode: 2,
		}
	}

	// Validate format
	if !slices.Contains(diagramFormats, format) {
		return &LintError{
			Message: fmt.Sprintf("unsupported format %q; supported formats: %s",
				format, strings.Join(diagramFormats, ", ")),
			ExitCode: 1,
			Cause:    ErrUnsupportedFormat,
		}
	}

	if specs && !slices.Contains(specCardFormats, format) {
		return unsupportedSpecsSurface(fmt.Sprintf("format %q", format))
	}

	var options []diagram.Option
	if specs {
		options = append(options, diagram.WithSpecs())
	}

	// Generate diagram
	var output []byte
	switch format {
	case "mermaid":
		output, err = diagram.ExportMermaid(model, style)
	case "ascii":
		output, err = diagram.ExportASCII(model, style)
	case "svg":
		output, err = diagram.ExportSVG(model, style, options...)
	case "event-flow":
		output, err = diagram.ExportEventFlow(model, style)
	case "event-flow-mermaid":
		output, err = diagram.ExportEventFlowMermaid(model, style)
	default:
		output, err = diagram.ExportDrawio(model, style, options...)
	}
	if err != nil {
		return &LintError{
			Message:  fmt.Sprintf("diagram generation: %s", err),
			ExitCode: 1,
		}
	}

	if slices.Contains(stdoutFormats, format) && outputPath == "" {
		fmt.Println(string(output))
		return lintExit(hasWarnings)
	}

	if outputPath == "" {
		switch format {
		case "svg":
			outputPath = diagramPath(path, ".svg")
		case "event-flow":
			// Not .svg: a picture of the same model in the same extension would
			// overwrite whichever of the two was written first.
			outputPath = diagramPath(path, ".event-flow.svg")
		default:
			outputPath = diagramPath(path, ".drawio")
		}
	}

	dir := filepath.Dir(outputPath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return &LintError{
				Message:  fmt.Sprintf("creating directory %s: %s", dir, err),
				ExitCode: 1,
			}
		}
	}

	if err := os.WriteFile(outputPath, output, 0o644); err != nil {
		return &LintError{
			Message:  fmt.Sprintf("writing %s: %s", outputPath, err),
			ExitCode: 1,
		}
	}

	return lintExit(hasWarnings)
}

// unsupportedSpecsSurface refuses to draw spec cards where nothing draws them,
// naming the flag and the two formats that do. Turning this refusal into output
// later breaks no one, where turning today's silence into output would change
// what a working script writes.
func unsupportedSpecsSurface(surface string) error {
	return &LintError{
		Message:  fmt.Sprintf("--specs is not supported for %s; only drawio and svg draw spec cards", surface),
		ExitCode: 1,
		Cause:    ErrSpecCardsUnsupported,
	}
}

func lintExit(hasWarnings bool) error {
	if hasWarnings {
		return &LintError{Message: "", ExitCode: 1}
	}
	return nil
}

// diagramPath replaces the .emod extension with the format's own.
// If the file has no .emod extension, the extension is appended.
func diagramPath(path, extension string) string {
	if strings.HasSuffix(path, ".emod") {
		return path[:len(path)-len(".emod")] + extension
	}
	return path + extension
}

// RunDiagramServe parses the file at path (if provided), generates diagram JSON,
// starts the viewer server with that data, and blocks until SIGINT/SIGTERM or a
// cancelled ctx shuts the server down. If launchBrowser is true, the default
// browser is opened to the viewer URL.
func RunDiagramServe(ctx context.Context, path string, launchBrowser bool) error {
	var diagramJSON []byte

	if path != "" {
		source, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		} else {
			model, diagnostics := oracle.Run(string(source), path)

			for _, d := range diagnostics {
				fmt.Fprintln(os.Stderr, d.String())
			}

			json, exportErr := export.ExportDiagramJSONDiagnostics(model, diagnostics)
			if exportErr != nil {
				fmt.Fprintln(os.Stderr, exportErr)
			} else {
				diagramJSON = json
			}
		}
	}

	addr, shutdown, err := viewer.ServeViewer(0, diagramJSON)
	if err != nil {
		return err
	}
	defer shutdown()

	if launchBrowser {
		openBrowser(addr)
	}

	sigCtx, stopSignals := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stopSignals()
	<-sigCtx.Done()

	return nil
}

func openBrowser(url string) {
	var cmd string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
	case "linux":
		cmd = "xdg-open"
	default:
		return
	}
	exec.Command(cmd, url).Start()
}

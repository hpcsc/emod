package cli

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/hpcsc/emod/internal/diagnostic"
	"github.com/hpcsc/emod/internal/diagram"
	"github.com/hpcsc/emod/internal/export"
	"github.com/hpcsc/emod/internal/lexer"
	"github.com/hpcsc/emod/internal/linter"
	"github.com/hpcsc/emod/internal/parser"
	"github.com/hpcsc/emod/internal/validator"
	"github.com/hpcsc/emod/internal/viewer"
)

// RunDiagram reads the file at path, lexes and parses it, validates and lints,
// generates a diagram in the requested format, and writes it.
// Supported formats: "drawio" (default), "mermaid", "svg", and "ascii".
// For drawio and svg: output is written to a file; if outputPath is empty it defaults to .drawio or .svg.
// For mermaid and ascii: output goes to stdout unless outputPath is specified.
// Errors produce diagnostics on stderr and a non-zero exit code.
// Lint warnings still produce the diagram but with exit code 1.
func RunDiagram(path, outputPath, format string) error {
	if path == "" {
		return &LintError{
			Message:  "diagram requires exactly one file argument",
			ExitCode: 1,
		}
	}

	source, err := os.ReadFile(path)
	if err != nil {
		return &LintError{
			Message:  fmt.Sprintf("reading %s: %s", path, err),
			ExitCode: 1,
		}
	}

	tokens, diagnostics := lexer.Scan(string(source), path)

	p := parser.New(tokens, path)
	model, parserDiags := p.Parse()
	diagnostics = append(diagnostics, parserDiags...)

	validatorDiags := validator.Validate(model)
	diagnostics = append(diagnostics, validatorDiags...)

	lintDiags := linter.Lint(model)
	diagnostics = append(diagnostics, lintDiags...)

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
	if format != "drawio" && format != "mermaid" && format != "svg" && format != "ascii" {
		return &LintError{
			Message:  fmt.Sprintf("unsupported format %q; supported formats: drawio, mermaid, svg, ascii", format),
			ExitCode: 1,
		}
	}

	// Generate diagram
	var output []byte
	switch format {
	case "mermaid":
		output, err = diagram.ExportMermaid(model)
	case "ascii":
		output, err = diagram.ExportASCII(model)
	case "svg":
		output, err = diagram.ExportSVG(model)
	default:
		output, err = diagram.ExportDrawio(model)
	}
	if err != nil {
		return &LintError{
			Message:  fmt.Sprintf("diagram generation: %s", err),
			ExitCode: 1,
		}
	}

	if (format == "mermaid" || format == "ascii") && outputPath == "" {
		fmt.Println(string(output))
		return lintExit(hasWarnings)
	}

	if outputPath == "" {
		switch format {
		case "svg":
			outputPath = defaultSVGPath(path)
		default:
			outputPath = defaultDrawioPath(path)
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

func lintExit(hasWarnings bool) error {
	if hasWarnings {
		return &LintError{"", 1}
	}
	return nil
}

// defaultDrawioPath replaces the .emod extension with .drawio.
// If the file has no .emod extension, .drawio is appended.
func defaultDrawioPath(path string) string {
	if strings.HasSuffix(path, ".emod") {
		return path[:len(path)-len(".emod")] + ".drawio"
	}
	return path + ".drawio"
}

// defaultSVGPath replaces the .emod extension with .svg.
// If the file has no .emod extension, .svg is appended.
func defaultSVGPath(path string) string {
	if strings.HasSuffix(path, ".emod") {
		return path[:len(path)-len(".emod")] + ".svg"
	}
	return path + ".svg"
}

// RunDiagramServe parses the file at path (if provided), generates diagram JSON,
// starts the viewer server with that data, and blocks until SIGINT/SIGTERM
// shuts the server down. If launchBrowser is true, the default browser is opened
// to the viewer URL.
func RunDiagramServe(path string, launchBrowser bool) error {
	var diagramJSON []byte

	if path != "" {
		source, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		} else {
			tokens, diagnostics := lexer.Scan(string(source), path)

			p := parser.New(tokens, path)
			model, parserDiags := p.Parse()
			diagnostics = append(diagnostics, parserDiags...)

			diagnostics = append(diagnostics, validator.Validate(model)...)
			diagnostics = append(diagnostics, linter.Lint(model)...)

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

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)
	<-sigCh

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

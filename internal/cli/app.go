package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/hpcsc/emod/internal/diagram"
	urfave "github.com/urfave/cli/v2"
)

func reportExitError(err error) error {
	if err == nil {
		return nil
	}

	var lintErr *LintError
	if errors.As(err, &lintErr) {
		if lintErr.Message != "" {
			fmt.Fprintln(os.Stderr, lintErr.Message)
		}
		return urfave.Exit("", lintErr.ExitCode)
	}

	fmt.Fprintln(os.Stderr, err)
	return urfave.Exit("", 1)
}

func NewApp() *urfave.App {
	return &urfave.App{
		Name:  "emod",
		Usage: "Event modeling DSL tool",
		Commands: []*urfave.Command{
			{
				Name:      "validate",
				Usage:     "Validate an .emod file",
				ArgsUsage: "<file>",
				Flags: []urfave.Flag{
					&urfave.StringFlag{
						Name:  "format",
						Usage: "Output format (text|json)",
						Value: "text",
					},
				},
				Action: func(c *urfave.Context) error {
					path := c.Args().First()
					format := c.String("format")
					return reportExitError(RunValidate(path, format))
				},
			},
			{
				Name:      "fmt",
				Usage:     "Format an .emod file",
				ArgsUsage: "<file>",
				Flags: []urfave.Flag{
					&urfave.BoolFlag{
						Name:  "check",
						Usage: "Check if the file is already formatted (exit 1 if not)",
					},
				},
				Action: func(c *urfave.Context) error {
					path := c.Args().First()
					check := c.Bool("check")
					if err := RunFmt(path, check); err != nil {
						fmt.Fprintln(os.Stderr, err)
						return urfave.Exit("", 1)
					}
					return nil
				},
			},
			{
				Name:      "lint",
				Usage:     "Lint an .emod file for naming conventions",
				ArgsUsage: "<file>",
				Flags: []urfave.Flag{
					&urfave.StringFlag{
						Name:  "format",
						Usage: "Output format (text|json)",
						Value: "text",
					},
					&urfave.StringFlag{
						Name:  "explain",
						Usage: "Print a description of a lint rule and exit",
					},
				},
				Action: func(c *urfave.Context) error {
					if explain := c.String("explain"); explain != "" {
						return reportExitError(RunLintExplain(explain))
					}

					path := c.Args().First()
					format := c.String("format")
					return reportExitError(RunLint(path, format))
				},
			},
			{
				Name:      "export",
				Usage:     "Export an .emod file as JSON or CUE",
				ArgsUsage: "<file>",
				Flags: []urfave.Flag{
					&urfave.StringFlag{
						Name:  "format",
						Usage: "Output format (json|cue|diagram-json)",
						Value: "json",
					},
				},
				Action: func(c *urfave.Context) error {
					path := c.Args().First()
					format := c.String("format")
					return reportExitError(RunExport(path, format))
				},
			},
			{
				Name:      "diagram",
				Usage:     "Generate a diagram from an .emod file",
				ArgsUsage: "<file>",
				Flags: []urfave.Flag{
					&urfave.StringFlag{
						Name:  "format",
						Usage: "Output format (drawio|mermaid|svg|ascii)",
						Value: "drawio",
					},
					&urfave.StringFlag{
						Name:  "style",
						Usage: "Layout style (projected|dcb|auto); auto detects based on context mode",
						Value: "auto",
					},
					&urfave.StringFlag{
						Name:  "o",
						Usage: "Output path",
					},
					&urfave.BoolFlag{
						Name:  "serve",
						Usage: "Start viewer server with diagram data",
					},
				},
				Action: func(c *urfave.Context) error {
					path := c.Args().First()
					if c.Bool("serve") {
						return RunDiagramServe(c.Context, path, true)
					}
					format := c.String("format")
					outputPath := c.String("o")
					style, err := diagram.ParseStyle(c.String("style"))
					if err != nil {
						return urfave.Exit(err.Error(), 1)
					}
					return reportExitError(RunDiagram(path, outputPath, format, style))
				},
			},
			{
				Name:  "slices",
				Usage: "Inspect the slices in a model",
				Action: func(c *urfave.Context) error {
					if arg := c.Args().First(); arg != "" {
						return reportExitError(fmt.Errorf("unknown slices subcommand %q; to list a model's slices run: emod slices list %s", arg, arg))
					}
					return urfave.ShowSubcommandHelp(c)
				},
				Subcommands: []*urfave.Command{
					{
						Name:      "list",
						Usage:     "List all slices in a model with their pattern types",
						ArgsUsage: "<file>",
						Flags: []urfave.Flag{
							&urfave.StringFlag{
								Name:  "format",
								Usage: "Output format (text|json)",
								Value: "text",
							},
						},
						Action: func(c *urfave.Context) error {
							path := c.Args().First()
							format := c.String("format")
							return reportExitError(RunSlicesList(path, format))
						},
					},
				},
			},
			{
				Name:      "glossary",
				Usage:     "Render a glossary of the terms a model defines",
				ArgsUsage: "<file>",
				Flags: []urfave.Flag{
					&urfave.StringFlag{
						Name:    "format",
						Aliases: []string{"f"},
						Usage:   "Output format (markdown, json)",
						Value:   "markdown",
					},
				},
				Action: func(c *urfave.Context) error {
					path, format := glossaryPathAndFormat(c.Args().Slice(), c.String("format"))
					return reportExitError(RunGlossary(path, format))
				},
			},
			{
				Name:  "schema",
				Usage: "Print the bundled CUE schema definition",
				Flags: []urfave.Flag{
					&urfave.StringFlag{
						Name:  "format",
						Usage: "Output format (cue)",
						Value: "cue",
					},
				},
				Action: func(c *urfave.Context) error {
					format := c.String("format")
					return reportExitError(RunSchema(format))
				},
			},
			{
				Name:  "lsp",
				Usage: "Start the LSP server (stdin/stdout transport)",
				Action: func(c *urfave.Context) error {
					if err := RunLSP(); err != nil {
						fmt.Fprintln(os.Stderr, err)
						return urfave.Exit("", 1)
					}
					return nil
				},
			},
		},
	}
}

package cli

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/hpcsc/emod/internal/diagram"
	"github.com/hpcsc/emod/internal/llm/config"
	"github.com/hpcsc/emod/internal/release"
	"github.com/hpcsc/emod/internal/version"
	urfave "github.com/urfave/cli/v3"
)

// Run executes the emod CLI over args, which are the process arguments with the
// program name still at the front.
func Run(args []string) error {
	return RunApp(NewApp(), args)
}

// RunApp is Run over a caller-supplied app, so a test can substitute an exit
// handler in place of the one that ends the process.
func RunApp(app *urfave.Command, args []string) error {
	return app.Run(context.Background(), args)
}

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

func aiConfigLoader(cmd *urfave.Command) func() (config.Config, error) {
	return func() (config.Config, error) {
		return config.Load(config.Flags{
			Provider:   cmd.String("ai-provider"),
			Model:      cmd.String("ai-model"),
			CheapModel: cmd.String("ai-model-cheap"),
			Effort:     cmd.String("ai-effort"),
			Endpoint:   cmd.String("ai-endpoint"),
		})
	}
}

func NewApp() *urfave.Command {
	return &urfave.Command{
		Name:    "emod",
		Usage:   "Event modeling DSL tool",
		Version: version.Current(),
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
				Action: func(ctx context.Context, cmd *urfave.Command) error {
					path := cmd.Args().First()
					format := cmd.String("format")
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
				Action: func(ctx context.Context, cmd *urfave.Command) error {
					path := cmd.Args().First()
					check := cmd.Bool("check")
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
				Action: func(ctx context.Context, cmd *urfave.Command) error {
					if explain := cmd.String("explain"); explain != "" {
						return reportExitError(RunLintExplain(explain))
					}

					path := cmd.Args().First()
					format := cmd.String("format")
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
				Action: func(ctx context.Context, cmd *urfave.Command) error {
					path := cmd.Args().First()
					format := cmd.String("format")
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
						Usage: "Output format (svg|mermaid|ascii|event-flow|event-flow-mermaid|event-flow-mermaid-no-icons)",
						Value: "svg",
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
					&urfave.BoolFlag{
						Name:  "specs",
						Usage: "Draw each slice's specs as a Given-When-Then card (svg only)",
					},
				},
				Action: func(ctx context.Context, cmd *urfave.Command) error {
					path := cmd.Args().First()
					specs := cmd.Bool("specs")
					if cmd.Bool("serve") {
						if specs {
							return reportExitError(unsupportedSpecsSurface("--serve"))
						}
						return RunDiagramServe(ctx, path, true)
					}
					format := cmd.String("format")
					outputPath := cmd.String("o")
					style, err := diagram.ParseStyle(cmd.String("style"))
					if err != nil {
						return urfave.Exit(err.Error(), 1)
					}
					return reportExitError(RunDiagram(path, outputPath, format, style, specs))
				},
			},
			{
				Name:  "slices",
				Usage: "Inspect the slices in a model",
				Action: func(ctx context.Context, cmd *urfave.Command) error {
					if arg := cmd.Args().First(); arg != "" {
						return reportExitError(fmt.Errorf("unknown slices subcommand %q; to list a model's slices run: emod slices list %s", arg, arg))
					}
					return urfave.ShowSubcommandHelp(cmd)
				},
				Commands: []*urfave.Command{
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
						Action: func(ctx context.Context, cmd *urfave.Command) error {
							path := cmd.Args().First()
							format := cmd.String("format")
							return reportExitError(RunSlicesList(path, format))
						},
					},
					{
						Name:      "arrange",
						Usage:     "Reorder slices so the model reads forward, moving view slices only",
						ArgsUsage: "<file>",
						Flags: []urfave.Flag{
							&urfave.BoolFlag{
								Name:  "check",
								Usage: "Report whether the slices are already arranged (exit 1 if not)",
							},
						},
						Action: func(ctx context.Context, cmd *urfave.Command) error {
							path := cmd.Args().First()
							check := cmd.Bool("check")
							return reportExitError(RunSlicesArrange(path, check))
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
				Action: func(ctx context.Context, cmd *urfave.Command) error {
					path := cmd.Args().First()
					format := cmd.String("format")
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
				Action: func(ctx context.Context, cmd *urfave.Command) error {
					format := cmd.String("format")
					return reportExitError(RunSchema(format))
				},
			},
			{
				Name:  "version",
				Usage: "Print the tag emod was built from, or its commit when it has no tag",
				Action: func(ctx context.Context, cmd *urfave.Command) error {
					return reportExitError(RunVersion())
				},
			},
			{
				Name:  "update",
				Usage: "Replace emod with the latest release, or with the latest prerelease",
				Flags: []urfave.Flag{
					&urfave.BoolFlag{
						Name:  "prerelease",
						Usage: "Install the latest prerelease, a build of main, in place of the latest release",
					},
					&urfave.BoolFlag{
						Name:  "check",
						Usage: "Only report whether this build is the latest",
					},
					&urfave.BoolFlag{
						Name:  "force",
						Usage: "Replace a build from a commit, which is not a release or a prerelease",
					},
				},
				Action: func(ctx context.Context, cmd *urfave.Command) error {
					channel := release.Releases
					if cmd.Bool("prerelease") {
						channel = release.Prereleases
					}
					return reportExitError(RunUpdate(ctx, channel, cmd.Bool("check"), cmd.Bool("force")))
				},
			},
			{
				Name:  "ai",
				Usage: "Check that AI is configured and turns a prompt into an emod model that passes the checks",
				Flags: []urfave.Flag{
					&urfave.StringFlag{Name: "ai-provider", Usage: "AI provider: anthropic or openai (EMOD_AI_PROVIDER)"},
					&urfave.StringFlag{Name: "ai-model", Usage: "Model ID for AI work (EMOD_AI_MODEL)"},
					&urfave.StringFlag{Name: "ai-model-cheap", Usage: "Model ID for low-stakes AI work (EMOD_AI_MODEL_CHEAP)"},
					&urfave.StringFlag{Name: "ai-effort", Usage: "Effort level: low, medium, high or xhigh (EMOD_AI_EFFORT)"},
					&urfave.StringFlag{Name: "ai-endpoint", Usage: "Gateway URL to send model calls through (EMOD_AI_ENDPOINT)"},
				},
				Action: func(ctx context.Context, cmd *urfave.Command) error {
					return reportExitError(RunAI(ctx, aiConfigLoader(cmd), connectModel))
				},
				Commands: []*urfave.Command{
					{
						Name:  "status",
						Usage: "Show the AI settings and the source of the API key, but not the key, and call no model",
						Action: func(_ context.Context, cmd *urfave.Command) error {
							return reportExitError(RunAIStatus(aiConfigLoader(cmd)))
						},
					},
				},
			},
			{
				Name:  "lsp",
				Usage: "Start the LSP server (stdin/stdout transport)",
				Action: func(ctx context.Context, cmd *urfave.Command) error {
					if err := RunLSP(); err != nil {
						fmt.Fprintln(os.Stderr, err)
						return urfave.Exit("", 1)
					}
					return nil
				},
			},
			{
				Name:  "mcp",
				Usage: "Start the MCP server (stdin/stdout transport)",
				Flags: []urfave.Flag{
					&urfave.StringFlag{
						Name:  "root",
						Usage: "Read model files only under this directory (default: the working directory)",
					},
					&urfave.BoolFlag{
						Name:  "allow-write",
						Usage: "Let emod_fmt and emod_slices_arrange write the result to the file for a call with write: true",
					},
				},
				Action: func(ctx context.Context, cmd *urfave.Command) error {
					return reportExitError(RunMCP(ctx, cmd.String("root"), cmd.Bool("allow-write")))
				},
			},
		},
	}
}

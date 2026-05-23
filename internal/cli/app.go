package cli

import (
	"errors"
	"fmt"
	"os"

	urfave "github.com/urfave/cli/v2"
)

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
					if err := RunValidate(path, format); err != nil {
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
					return nil
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
				},
				Action: func(c *urfave.Context) error {
					path := c.Args().First()
					format := c.String("format")
					if err := RunLint(path, format); err != nil {
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
					return nil
				},
			},
			{
				Name:      "export",
				Usage:     "Export an .emod file as JSON or CUE",
				ArgsUsage: "<file>",
				Flags: []urfave.Flag{
					&urfave.StringFlag{
						Name:  "format",
						Usage: "Output format (json|cue)",
						Value: "json",
					},
				},
				Action: func(c *urfave.Context) error {
					path := c.Args().First()
					format := c.String("format")
					if err := RunExport(path, format); err != nil {
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
					return nil
				},
			},
			{
				Name:      "diagram",
				Usage:     "Generate a diagram from an .emod file",
				ArgsUsage: "<file>",
				Flags: []urfave.Flag{
					&urfave.StringFlag{
						Name:  "format",
						Usage: "Output format (drawio|mermaid|svg)",
						Value: "drawio",
					},
					&urfave.StringFlag{
						Name:  "o",
						Usage: "Output path",
					},
				},
				Action: func(c *urfave.Context) error {
					path := c.Args().First()
					format := c.String("format")
					outputPath := c.String("o")
					if err := RunDiagram(path, outputPath, format); err != nil {
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
					return nil
				},
			},
			{
				Name:      "slices",
				Usage:     "List all slices in a model with their pattern types",
				ArgsUsage: "<file>",
				Action: func(c *urfave.Context) error {
					path := c.Args().First()
					if err := RunSlices(path); err != nil {
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
					return nil
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
					if err := RunSchema(format); err != nil {
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
					return nil
				},
			},
		},
	}
}

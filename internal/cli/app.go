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
				Action: func(c *urfave.Context) error {
					path := c.Args().First()
					if err := RunValidate(path); err != nil {
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
		},
	}
}

package cli

import (
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
				Action: func(c *urfave.Context) error {
					path := c.Args().First()
					if err := RunLint(path); err != nil {
						fmt.Fprintln(os.Stderr, err)
						return urfave.Exit("", 1)
					}
					return nil
				},
			},
		},
	}
}

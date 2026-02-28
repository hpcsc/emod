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
		},
	}
}

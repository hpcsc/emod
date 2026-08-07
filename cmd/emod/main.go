package main

import (
	"os"

	"github.com/hpcsc/emod/internal/cli"
)

func main() {
	// Run reports usage errors itself and returns them without an exit code of
	// their own, so an unparseable command line would otherwise succeed.
	if err := cli.Run(os.Args); err != nil {
		os.Exit(1)
	}
}

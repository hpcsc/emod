package main

import (
	"os"

	"github.com/hpcsc/emod/internal/cli"
)

func main() {
	_ = cli.NewApp().Run(os.Args)
}

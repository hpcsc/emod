package cli

import (
	"context"
	"os"

	"github.com/hpcsc/emod/internal/mcp"
)

func RunMCP(ctx context.Context) error {
	root, err := mcp.NewRoot(".")
	if err != nil {
		return err
	}
	return mcp.NewServer(os.Stdin, os.Stdout, root).Run(ctx)
}

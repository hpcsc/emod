package cli

import (
	"context"
	"os"

	"github.com/hpcsc/emod/internal/mcp"
)

func RunMCP(ctx context.Context) error {
	return mcp.NewServer(os.Stdin, os.Stdout).Run(ctx)
}

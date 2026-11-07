package cli

import (
	"cmp"
	"context"
	"os"

	"github.com/hpcsc/emod/internal/mcp"
)

func RunMCP(ctx context.Context, rootDir string) error {
	root, err := mcp.NewRoot(cmp.Or(rootDir, "."))
	if err != nil {
		return err
	}
	return mcp.NewServer(os.Stdin, os.Stdout, root, false).Run(ctx)
}

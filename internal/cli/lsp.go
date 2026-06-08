package cli

import (
	"context"
	"os"

	"github.com/hpcsc/emod/internal/lsp"
)

// RunLSP starts the LSP server reading from stdin and writing to stdout.
func RunLSP() error {
	server := lsp.NewServer(os.Stdin, os.Stdout)
	return server.Run(context.Background())
}

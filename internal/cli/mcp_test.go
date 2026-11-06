//go:build unit

package cli_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"

	"github.com/hpcsc/emod/internal/diagnostic"
)

func TestMCP(t *testing.T) {
	t.Run("help", func(t *testing.T) {
		t.Run("lists mcp as the command that starts the MCP server on stdin and stdout", func(t *testing.T) {
			var err error
			stdout := captureStdout(t, func() {
				err = runCommandLine(t, "emod", "--help")
			})

			require.NoError(t, err)
			require.Regexp(t, `(?m)^\s+mcp\s+Start the MCP server \(stdin/stdout transport\)$`, stdout)
		})
	})

	t.Run("serve", func(t *testing.T) {
		t.Run("checks a draft that a host sends on stdin and answers on stdout", func(t *testing.T) {
			serverReads, hostWrites, err := os.Pipe()
			require.NoError(t, err)
			hostReads, serverWrites, err := os.Pipe()
			require.NoError(t, err)
			stdin, stdout := os.Stdin, os.Stdout
			os.Stdin, os.Stdout = serverReads, serverWrites
			defer func() { os.Stdin, os.Stdout = stdin, stdout }()

			served := make(chan error, 1)
			go func() { served <- runCommandLine(t, "emod", "mcp") }()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			client := sdk.NewClient(&sdk.Implementation{Name: "test-host", Version: "1"}, nil)
			session, err := client.Connect(ctx, &sdk.IOTransport{Reader: hostReads, Writer: hostWrites}, nil)
			require.NoError(t, err)
			result, err := session.CallTool(ctx, &sdk.CallToolParams{
				Name:      "emod_check",
				Arguments: map[string]any{"content": warningEmod},
			})
			require.NoError(t, err)
			require.NoError(t, session.Close())

			require.NoError(t, <-served)
			encoded, err := json.Marshal(result.StructuredContent)
			require.NoError(t, err)
			var decoded struct {
				Diagnostics []diagnostic.Record `json:"diagnostics"`
			}
			require.NoError(t, json.Unmarshal(encoded, &decoded))
			require.Len(t, decoded.Diagnostics, 1)
			require.Equal(t, "inline.emod", decoded.Diagnostics[0].File)
			require.Equal(t, "state-obsession", decoded.Diagnostics[0].Rule)
		})
	})
}

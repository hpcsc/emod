//go:build unit

package cli_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"

	"github.com/hpcsc/emod/internal/diagnostic"
)

func checkOverStdio(t *testing.T, commandLine []string, arguments map[string]any) []diagnostic.Record {
	t.Helper()

	serverReads, hostWrites, err := os.Pipe()
	require.NoError(t, err)
	hostReads, serverWrites, err := os.Pipe()
	require.NoError(t, err)
	stdin, stdout := os.Stdin, os.Stdout
	os.Stdin, os.Stdout = serverReads, serverWrites
	defer func() { os.Stdin, os.Stdout = stdin, stdout }()

	served := make(chan error, 1)
	go func() { served <- runCommandLine(t, commandLine...) }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client := sdk.NewClient(&sdk.Implementation{Name: "test-host", Version: "1"}, nil)
	session, err := client.Connect(ctx, &sdk.IOTransport{Reader: hostReads, Writer: hostWrites}, nil)
	require.NoError(t, err)
	result, err := session.CallTool(ctx, &sdk.CallToolParams{Name: "emod_check", Arguments: arguments})
	require.NoError(t, err)
	require.NoError(t, session.Close())
	require.NoError(t, <-served)
	require.False(t, result.IsError, "the call must succeed: %v", result.Content)

	encoded, err := json.Marshal(result.StructuredContent)
	require.NoError(t, err)
	var decoded struct {
		Diagnostics []diagnostic.Record `json:"diagnostics"`
	}
	require.NoError(t, json.Unmarshal(encoded, &decoded))
	return decoded.Diagnostics
}

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
			records := checkOverStdio(t, []string{"emod", "mcp"}, map[string]any{"content": warningEmod})

			require.Len(t, records, 1)
			require.Equal(t, "inline.emod", records[0].File)
			require.Equal(t, "state-obsession", records[0].Rule)
		})

		t.Run("reads a path under the working directory, which is the root", func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(dir, "ordering.emod"), []byte(warningEmod), 0o644))
			t.Chdir(dir)

			records := checkOverStdio(t, []string{"emod", "mcp"}, map[string]any{"path": "ordering.emod"})

			require.Len(t, records, 1)
			require.Equal(t, "ordering.emod", records[0].File)
			require.Equal(t, "state-obsession", records[0].Rule)
		})
	})
}

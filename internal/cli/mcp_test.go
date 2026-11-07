//go:build unit

package cli_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
	urfave "github.com/urfave/cli/v3"

	"github.com/hpcsc/emod/internal/cli"
	"github.com/hpcsc/emod/internal/diagnostic"
)

func callOverStdio(t *testing.T, commandLine []string, name string, arguments map[string]any) *sdk.CallToolResult {
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
	result, err := session.CallTool(ctx, &sdk.CallToolParams{Name: name, Arguments: arguments})
	require.NoError(t, err)
	require.NoError(t, session.Close())
	require.NoError(t, <-served)
	return result
}

func checkOverStdio(t *testing.T, commandLine []string, arguments map[string]any) []diagnostic.Record {
	t.Helper()

	result := callOverStdio(t, commandLine, "emod_check", arguments)
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

		t.Run("lists --root for mcp, with the working directory as its default", func(t *testing.T) {
			var err error
			stdout := captureStdout(t, func() {
				err = runCommandLine(t, "emod", "mcp", "--help")
			})

			require.NoError(t, err)
			require.Regexp(t, `(?m)^\s+--root string\s+Read model files only under this directory \(default: the working directory\)$`, stdout)
		})

		t.Run("lists --allow-write for mcp, with a usage that names the two tools that can write", func(t *testing.T) {
			var err error
			stdout := captureStdout(t, func() {
				err = runCommandLine(t, "emod", "mcp", "--help")
			})

			require.NoError(t, err)
			require.Regexp(t, `(?m)^\s+--allow-write\s+Let emod_fmt and emod_slices_arrange write the result to the file for a call with write: true$`, stdout)
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

		t.Run("reads an absolute path under the working directory when no --root is given", func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(dir, "ordering.emod"), []byte(warningEmod), 0o644))
			t.Chdir(dir)
			path := filepath.Join(dir, "ordering.emod")

			records := checkOverStdio(t, []string{"emod", "mcp"}, map[string]any{"path": path})

			require.Len(t, records, 1)
			require.Equal(t, path, records[0].File)
			require.Equal(t, "state-obsession", records[0].Rule)
		})

		t.Run("reads a path relative to the directory that --root names", func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.MkdirAll(filepath.Join(dir, "models"), 0o755))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "models", "ordering.emod"), []byte(warningEmod), 0o644))

			records := checkOverStdio(t, []string{"emod", "mcp", "--root", dir}, map[string]any{"path": "models/ordering.emod"})

			require.Len(t, records, 1)
			require.Equal(t, "models/ordering.emod", records[0].File)
			require.Equal(t, "state-obsession", records[0].Rule)
		})

		t.Run("with --allow-write, emod_fmt with write writes the text that emod fmt writes to the file", func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "hotel.emod")
			require.NoError(t, os.WriteFile(path, []byte(emodWithoutVersionHeader), 0o644))

			result := callOverStdio(t, []string{"emod", "mcp", "--root", dir, "--allow-write"}, "emod_fmt", map[string]any{"path": "hotel.emod", "write": true})

			require.False(t, result.IsError, "the call must succeed: %v", result.Content)
			written, err := os.ReadFile(path)
			require.NoError(t, err)
			require.Equal(t, formattedEmod, string(written))
		})

		t.Run("with --allow-write, emod_slices_arrange with write writes the text that emod slices arrange writes to the file", func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "lending.emod")
			require.NoError(t, os.WriteFile(path, []byte(unarrangedEmod), 0o644))
			arrangedByCLI := writeTemp(t, "lending.emod", unarrangedEmod)
			captureStdout(t, func() {
				require.NoError(t, cli.RunSlicesArrange(arrangedByCLI, false))
			})
			expected, err := os.ReadFile(arrangedByCLI)
			require.NoError(t, err)
			require.NotEqual(t, unarrangedEmod, string(expected), "the fixture must have a slice to move")

			result := callOverStdio(t, []string{"emod", "mcp", "--root", dir, "--allow-write"}, "emod_slices_arrange", map[string]any{"path": "lending.emod", "write": true})

			require.False(t, result.IsError, "the call must succeed: %v", result.Content)
			written, err := os.ReadFile(path)
			require.NoError(t, err)
			require.Equal(t, string(expected), string(written))
		})

		t.Run("without --allow-write, a call with write gives an error result that names the flag, and the file does not change", func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "hotel.emod")
			require.NoError(t, os.WriteFile(path, []byte(emodWithoutVersionHeader), 0o644))

			result := callOverStdio(t, []string{"emod", "mcp", "--root", dir}, "emod_fmt", map[string]any{"path": "hotel.emod", "write": true})

			require.True(t, result.IsError)
			text, ok := result.Content[0].(*sdk.TextContent)
			require.True(t, ok)
			require.Contains(t, text.Text, "emod mcp --allow-write")
			unchanged, err := os.ReadFile(path)
			require.NoError(t, err)
			require.Equal(t, emodWithoutVersionHeader, string(unchanged))
		})

		t.Run("a root that does not exist ends the command with exit code 1 and a message that names it", func(t *testing.T) {
			missing := filepath.Join(t.TempDir(), "missing")

			var err error
			stderr := captureStderr(t, func() {
				err = runCommandLine(t, "emod", "mcp", "--root", missing)
			})

			var exitErr urfave.ExitCoder
			require.True(t, errors.As(err, &exitErr))
			require.Equal(t, 1, exitErr.ExitCode())
			require.Contains(t, stderr, "root "+missing+": no such file or directory")
		})
	})
}

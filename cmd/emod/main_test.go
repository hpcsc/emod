//go:build integration

package main_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

var emodBinary string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "emod-binary-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	emodBinary = filepath.Join(dir, "emod")

	build := exec.Command("go", "build", "-o", emodBinary, ".")
	build.Stdout = os.Stderr
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "building emod:", err)
		_ = os.RemoveAll(dir)
		os.Exit(1)
	}

	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

func repoRoot(t *testing.T) string {
	t.Helper()

	root, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)
	return root
}

func withoutAISettings(t *testing.T) []string {
	t.Helper()

	var env []string
	for _, variable := range os.Environ() {
		key, _, _ := strings.Cut(variable, "=")
		if strings.HasPrefix(key, "EMOD_AI_") || key == "ANTHROPIC_API_KEY" || key == "OPENAI_API_KEY" ||
			key == "HOME" || key == "XDG_CONFIG_HOME" {
			continue
		}
		env = append(env, variable)
	}
	return append(env, "HOME="+t.TempDir())
}

func startServer(t *testing.T) (*sdk.ClientSession, *exec.Cmd) {
	t.Helper()

	server := exec.Command(emodBinary, "mcp")
	server.Dir = repoRoot(t)
	server.Env = withoutAISettings(t)
	server.Stderr = os.Stderr

	client := sdk.NewClient(&sdk.Implementation{Name: "test-host", Version: "1"}, nil)
	session, err := client.Connect(context.Background(), &sdk.CommandTransport{Command: server, TerminateDuration: 10 * time.Second}, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = session.Close() })
	return session, server
}

func validateJSON(t *testing.T, path string) string {
	t.Helper()

	validate := exec.Command(emodBinary, "validate", "--format", "json", path)
	validate.Dir = repoRoot(t)
	validate.Env = withoutAISettings(t)
	output, err := validate.Output()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		require.LessOrEqual(t, exitErr.ExitCode(), 2, "emod validate failed: %s", exitErr.Stderr)
	} else {
		require.NoError(t, err)
	}
	return string(output)
}

func TestMCP(t *testing.T) {
	t.Run("handshake", func(t *testing.T) {
		t.Run("the binary names itself emod, with the version that emod version prints", func(t *testing.T) {
			printed, err := exec.Command(emodBinary, "version").Output()
			require.NoError(t, err)
			session, _ := startServer(t)

			info := session.InitializeResult().ServerInfo

			require.Equal(t, &sdk.Implementation{Name: "emod", Version: strings.TrimSpace(string(printed))}, info)
		})

		t.Run("the binary lists emod_check", func(t *testing.T) {
			session, _ := startServer(t)

			listed, err := session.ListTools(context.Background(), nil)

			require.NoError(t, err)
			require.Len(t, listed.Tools, 1)
			require.Equal(t, "emod_check", listed.Tools[0].Name)
		})
	})

	t.Run("check", func(t *testing.T) {
		root := repoRoot(t)
		var paths []string
		for _, pattern := range []string{"examples/*.emod", "internal/parser/testdata/*.emod"} {
			matches, err := filepath.Glob(filepath.Join(root, pattern))
			require.NoError(t, err)
			for _, match := range matches {
				relative, err := filepath.Rel(root, match)
				require.NoError(t, err)
				paths = append(paths, filepath.ToSlash(relative))
			}
		}
		require.Contains(t, paths, "examples/error_diagnostics_test.emod", "the files must include one with diagnostics")

		for _, path := range paths {
			t.Run("gives the records of emod validate for "+path, func(t *testing.T) {
				source, err := os.ReadFile(filepath.Join(root, path))
				require.NoError(t, err)
				session, _ := startServer(t)

				result, err := session.CallTool(context.Background(), &sdk.CallToolParams{
					Name:      "emod_check",
					Arguments: map[string]any{"content": string(source), "name": path},
				})

				require.NoError(t, err)
				require.False(t, result.IsError)
				structured, ok := result.StructuredContent.(map[string]any)
				require.True(t, ok, "the structured result must be an object, but it is %T", result.StructuredContent)
				records, err := json.Marshal(structured["diagnostics"])
				require.NoError(t, err)
				require.JSONEq(t, validateJSON(t, path), string(records))
			})
		}
	})

	t.Run("shutdown", func(t *testing.T) {
		t.Run("the server ends with exit code 0 when the host closes its stdin", func(t *testing.T) {
			session, server := startServer(t)

			err := session.Close()

			require.NoError(t, err)
			require.Equal(t, 0, server.ProcessState.ExitCode())
		})
	})
}

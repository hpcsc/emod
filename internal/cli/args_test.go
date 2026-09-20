//go:build unit

package cli_test

import (
	"context"
	"os"
	"testing"

	"github.com/hpcsc/emod/internal/cli"
	"github.com/stretchr/testify/require"
	urfave "github.com/urfave/cli/v3"
)

func TestArgs(t *testing.T) {
	t.Run("flags after the file argument", func(t *testing.T) {
		t.Run("a long format written after the file selects that format", func(t *testing.T) {
			path := writeTemp(t, "model.emod", validEmod)

			output := captureStdout(t, func() {
				require.NoError(t, runCommandLine(t, "emod", "export", path, "--format", "cue"))
			})

			require.Contains(t, output, "name: ")
			require.NotContains(t, output, `"diagnostics"`)
		})

		t.Run("a short format written after the file selects that format", func(t *testing.T) {
			path := writeTemp(t, "model.emod", validEmod)

			output := captureStdout(t, func() {
				require.NoError(t, runCommandLine(t, "emod", "glossary", path, "-f", "json"))
			})

			require.Contains(t, output, `"contexts"`)
		})

		t.Run("a format joined to its flag by = selects that format", func(t *testing.T) {
			path := writeTemp(t, "model.emod", validEmod)

			output := captureStdout(t, func() {
				require.NoError(t, runCommandLine(t, "emod", "export", path, "--format=cue"))
			})

			require.Contains(t, output, "name: ")
		})

		// --check takes no value, and it reports rather than rewrites, so the
		// untouched file is what says the flag was read at all: without it the
		// same command line formats the file in place.
		t.Run("a valueless flag written after the file still applies", func(t *testing.T) {
			unformatted := "model \"M\"\n"
			path := writeTemp(t, "unformatted.emod", unformatted)

			require.Error(t, runCommandLine(t, "emod", "fmt", path, "--check"))

			after, err := os.ReadFile(path)
			require.NoError(t, err)
			require.Equal(t, unformatted, string(after))
		})

		t.Run("a subcommand's flag applies as a top-level command's does", func(t *testing.T) {
			path := writeTemp(t, "model.emod", validEmod)

			output := captureStdout(t, func() {
				require.NoError(t, runCommandLine(t, "emod", "slices", "list", path, "--format", "json"))
			})

			require.Contains(t, output, "[")
		})

		t.Run("a flag the command does not declare is still reported as undefined", func(t *testing.T) {
			path := writeTemp(t, "model.emod", validEmod)

			err := runCommandLine(t, "emod", "export", path, "-f", "cue")

			require.Error(t, err)
			require.Contains(t, err.Error(), "flag provided but not defined")
		})
	})

	t.Run("flags before the file argument", func(t *testing.T) {
		t.Run("a long format written before the file selects that format", func(t *testing.T) {
			path := writeTemp(t, "model.emod", validEmod)

			output := captureStdout(t, func() {
				require.NoError(t, runCommandLine(t, "emod", "export", "--format", "cue", path))
			})

			require.Contains(t, output, "name: ")
		})
	})

	t.Run("arguments after --", func(t *testing.T) {
		t.Run("a word that looks like a flag is read as the file", func(t *testing.T) {
			app := cli.NewApp()
			app.ExitErrHandler = func(context.Context, *urfave.Command, error) {}

			err := cli.RunApp(app, []string{"emod", "validate", "--", "--format"})

			require.Error(t, err)
			require.NotContains(t, err.Error(), "flag provided but not defined")
		})
	})
}

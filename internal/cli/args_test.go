//go:build unit

package cli_test

import (
	"os"
	"testing"

	"github.com/hpcsc/emod/internal/cli"
	"github.com/stretchr/testify/require"
	urfave "github.com/urfave/cli/v2"
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

		t.Run("a flag the command does not declare is still reported as undefined", func(t *testing.T) {
			path := writeTemp(t, "model.emod", validEmod)

			err := runCommandLine(t, "emod", "export", path, "-f", "cue")

			require.Error(t, err)
			require.Contains(t, err.Error(), "flag provided but not defined")
		})
	})

	t.Run("reordering", func(t *testing.T) {
		t.Run("leaves a command line whose flags already precede the file alone", func(t *testing.T) {
			path := writeTemp(t, "model.emod", validEmod)

			output := captureStdout(t, func() {
				require.NoError(t, runCommandLine(t, "emod", "export", "--format", "cue", path))
			})

			require.Contains(t, output, "name: ")
		})

		t.Run("reaches a subcommand's flags, not only a top-level command's", func(t *testing.T) {
			path := writeTemp(t, "model.emod", validEmod)

			output := captureStdout(t, func() {
				require.NoError(t, runCommandLine(t, "emod", "slices", "list", path, "--format", "json"))
			})

			require.Contains(t, output, "[")
		})

		t.Run("treats a word after -- as a file even when it looks like a flag", func(t *testing.T) {
			app := cli.NewApp()
			app.ExitErrHandler = func(*urfave.Context, error) {}

			err := cli.RunApp(app, []string{"emod", "validate", "--", "--format"})

			require.Error(t, err)
			require.NotContains(t, err.Error(), "flag provided but not defined")
		})
	})
}

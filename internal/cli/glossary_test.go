//go:build unit

package cli_test

import (
	"errors"
	"testing"

	"github.com/hpcsc/emod/internal/cli"
	"github.com/stretchr/testify/require"
	urfave "github.com/urfave/cli/v2"
)

// The app terminates the process on an exit-coded error, which would take the
// test binary with it, so the handler is replaced with one that lets Run return.
func runCommandLine(t *testing.T, args ...string) error {
	t.Helper()
	app := cli.NewApp()
	app.ExitErrHandler = func(*urfave.Context, error) {}
	return app.Run(args)
}

func TestGlossary(t *testing.T) {
	t.Run("markdown", func(t *testing.T) {
		t.Run("names the model, its context and its aggregate with empty definitions when the model describes none of them", func(t *testing.T) {
			path := writeTemp(t, "undescribed.emod", validEmod)

			output := captureStdout(t, func() {
				require.NoError(t, cli.RunGlossary(path, "markdown"))
			})

			require.Equal(t, `# Hotel Reservation

## Reservations

### Reservation
`, output)
		})

		t.Run("pairs the model, its context and its aggregate with the descriptions the model declares", func(t *testing.T) {
			path := writeTemp(t, "described.emod", describedEmod)

			output := captureStdout(t, func() {
				require.NoError(t, cli.RunGlossary(path, "markdown"))
			})

			require.Equal(t, `# Hotel Reservation

How the hotel takes, confirms and imports room bookings

## Reservations

Everything the hotel knows about a stay before the guest arrives

### Reservation

One guest holding one room over one date range
`, output)
		})
	})

	t.Run("rejected input", func(t *testing.T) {
		t.Run("missing file argument names the cause callers branch on", func(t *testing.T) {
			err := cli.RunGlossary("", "markdown")

			require.ErrorIs(t, err, cli.ErrMissingFileArgument)
			var lintErr *cli.LintError
			require.True(t, errors.As(err, &lintErr))
			require.Equal(t, 1, lintErr.ExitCode)
		})

		t.Run("unreadable path is reported with the path the user supplied", func(t *testing.T) {
			missing := "/tmp/nonexistent-emod-glossary-file-abc123.emod"

			err := cli.RunGlossary(missing, "markdown")

			require.Error(t, err)
			require.Contains(t, err.Error(), missing)
			var lintErr *cli.LintError
			require.True(t, errors.As(err, &lintErr))
			require.Equal(t, 1, lintErr.ExitCode)
		})

		t.Run("unparseable file reports the rejected token and the expected keywords instead of a glossary", func(t *testing.T) {
			path := writeTemp(t, "invalid.emod", invalidEmod)

			var err error
			output := captureStdout(t, func() {
				err = cli.RunGlossary(path, "markdown")
			})

			require.Empty(t, output)
			require.Error(t, err)
			require.Contains(t, err.Error(), `"foobar"`)
			require.Contains(t, err.Error(), "model")
			require.Contains(t, err.Error(), "actor")
			require.Contains(t, err.Error(), "context")
			var lintErr *cli.LintError
			require.True(t, errors.As(err, &lintErr))
			require.Equal(t, 1, lintErr.ExitCode)
		})

		t.Run("a format the command does not render is rejected by name", func(t *testing.T) {
			path := writeTemp(t, "valid.emod", validEmod)

			err := cli.RunGlossary(path, "xml")

			require.ErrorIs(t, err, cli.ErrUnsupportedFormat)
			require.Contains(t, err.Error(), "markdown")
			var lintErr *cli.LintError
			require.True(t, errors.As(err, &lintErr))
			require.Equal(t, 1, lintErr.ExitCode)
		})
	})

	t.Run("command line", func(t *testing.T) {
		t.Run("rejects a short-form format written after the file argument", func(t *testing.T) {
			path := writeTemp(t, "valid.emod", validEmod)

			var err error
			var stdout string
			stderr := captureStderr(t, func() {
				stdout = captureStdout(t, func() {
					err = runCommandLine(t, "emod", "glossary", path, "-f", "xml")
				})
			})

			require.Error(t, err)
			var exitErr urfave.ExitCoder
			require.True(t, errors.As(err, &exitErr))
			require.Equal(t, 1, exitErr.ExitCode())
			require.Contains(t, stderr, "xml")
			require.Contains(t, stderr, "markdown")
			require.NotContains(t, stdout, "Hotel Reservation")
		})

		t.Run("renders when the long-form format is written before the file argument", func(t *testing.T) {
			path := writeTemp(t, "valid.emod", validEmod)

			var err error
			stdout := captureStdout(t, func() {
				err = runCommandLine(t, "emod", "glossary", "--format", "markdown", path)
			})

			require.NoError(t, err)
			require.Contains(t, stdout, "Hotel Reservation")
			require.Contains(t, stdout, "Reservations")
		})

		t.Run("lists glossary among the commands the help offers", func(t *testing.T) {
			var err error
			stdout := captureStdout(t, func() {
				err = runCommandLine(t, "emod", "--help")
			})

			require.NoError(t, err)
			require.Contains(t, stdout, "glossary")
		})
	})
}

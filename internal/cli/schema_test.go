//go:build unit

package cli_test

import (
	"errors"
	"testing"

	"github.com/hpcsc/emod/internal/cli"
	"github.com/hpcsc/emod/internal/cue"
	"github.com/stretchr/testify/require"
)

func TestSchema(t *testing.T) {
	t.Run("prints CUE schema to stdout", func(t *testing.T) {
		output := captureStdout(t, func() {
			err := cli.RunSchema("cue")
			require.NoError(t, err)
		})

		require.Contains(t, output, "#Model:")
		require.Contains(t, output, "#Actor:")
		require.Contains(t, output, "#Slice:")
	})

	t.Run("prints a schema declaring the payload a spec element may state", func(t *testing.T) {
		output := captureStdout(t, func() {
			err := cli.RunSchema("cue")
			require.NoError(t, err)
		})

		require.Contains(t, output, "#SpecElement:")
		require.Contains(t, output, "#PayloadField:")
		require.Regexp(t, `payload\?:\s+\[\.\.\.#PayloadField\]`, output)
		require.Regexp(t, `value:\s+string \| number \| bool`, output)
	})

	t.Run("output matches embedded schema content", func(t *testing.T) {
		output := captureStdout(t, func() {
			err := cli.RunSchema("cue")
			require.NoError(t, err)
		})

		require.Equal(t, cue.Schema, output)
	})

	t.Run("unsupported format returns LintError", func(t *testing.T) {
		err := cli.RunSchema("json")

		require.ErrorIs(t, err, cli.ErrUnsupportedFormat)
		require.Contains(t, err.Error(), "cue")

		var lintErr *cli.LintError
		require.True(t, errors.As(err, &lintErr))
		require.Equal(t, 1, lintErr.ExitCode)
	})
}

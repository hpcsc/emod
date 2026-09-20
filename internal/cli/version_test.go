//go:build unit

package cli_test

import (
	"testing"

	"github.com/hpcsc/emod/internal/cli"
	"github.com/hpcsc/emod/internal/version"
	"github.com/stretchr/testify/require"
)

func TestVersion(t *testing.T) {
	t.Run("print", func(t *testing.T) {
		t.Run("prints the version of this build on one line of stdout", func(t *testing.T) {
			output := captureStdout(t, func() {
				require.NoError(t, cli.RunVersion())
			})

			require.Equal(t, version.Current()+"\n", output)
		})
	})
}

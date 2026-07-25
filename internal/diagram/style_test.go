//go:build unit

package diagram_test

import (
	"strings"
	"testing"

	"github.com/hpcsc/emod/internal/diagram"
	"github.com/stretchr/testify/require"
)

func TestStyle(t *testing.T) {
	styles := map[string]diagram.Style{
		"auto":      diagram.StyleAuto,
		"projected": diagram.StyleProjected,
		"dcb":       diagram.StyleDCB,
	}

	t.Run("parse", func(t *testing.T) {
		t.Run("accepts every style name the CLI documents", func(t *testing.T) {
			for name, want := range styles {
				got, err := diagram.ParseStyle(name)

				require.NoError(t, err, "parsing %q", name)
				require.Equal(t, want, got, "parsing %q", name)
			}
		})

		t.Run("ignores case", func(t *testing.T) {
			for _, name := range []string{"AUTO", "Projected", "DCB"} {
				lowered, err := diagram.ParseStyle(name)

				require.NoError(t, err, "parsing %q", name)
				require.Equal(t, styles[strings.ToLower(name)], lowered, "parsing %q", name)
			}
		})

		t.Run("rejects an unknown style", func(t *testing.T) {
			_, err := diagram.ParseStyle("invalid")

			require.Error(t, err)
			require.Contains(t, err.Error(), "unsupported style")
		})
	})

	t.Run("string", func(t *testing.T) {
		t.Run("round-trips back through parse", func(t *testing.T) {
			for name, style := range styles {
				require.Equal(t, name, style.String())

				parsed, err := diagram.ParseStyle(style.String())
				require.NoError(t, err)
				require.Equal(t, style, parsed)
			}
		})
	})
}

//go:build unit

package diagram_test

import (
	"testing"

	"github.com/hpcsc/emod/internal/diagram"
	"github.com/stretchr/testify/require"
)

func TestStyle(t *testing.T) {
	t.Run("ParseStyle returns StyleAuto for auto", func(t *testing.T) {
		s, err := diagram.ParseStyle("auto")
		require.NoError(t, err)
		require.Equal(t, diagram.StyleAuto, s)
	})

	t.Run("ParseStyle returns StyleProjected for projected", func(t *testing.T) {
		s, err := diagram.ParseStyle("projected")
		require.NoError(t, err)
		require.Equal(t, diagram.StyleProjected, s)
	})

	t.Run("ParseStyle returns StyleDCB for dcb", func(t *testing.T) {
		s, err := diagram.ParseStyle("dcb")
		require.NoError(t, err)
		require.Equal(t, diagram.StyleDCB, s)
	})

	t.Run("ParseStyle is case-insensitive", func(t *testing.T) {
		s, err := diagram.ParseStyle("AUTO")
		require.NoError(t, err)
		require.Equal(t, diagram.StyleAuto, s)

		s, err = diagram.ParseStyle("Projected")
		require.NoError(t, err)
		require.Equal(t, diagram.StyleProjected, s)

		s, err = diagram.ParseStyle("DCB")
		require.NoError(t, err)
		require.Equal(t, diagram.StyleDCB, s)
	})

	t.Run("ParseStyle returns error for invalid value", func(t *testing.T) {
		_, err := diagram.ParseStyle("invalid")
		require.Error(t, err)
		require.Contains(t, err.Error(), "unsupported style")
	})

	t.Run("StyleAuto.String returns auto", func(t *testing.T) {
		require.Equal(t, "auto", diagram.StyleAuto.String())
	})

	t.Run("StyleProjected.String returns projected", func(t *testing.T) {
		require.Equal(t, "projected", diagram.StyleProjected.String())
	})

	t.Run("StyleDCB.String returns dcb", func(t *testing.T) {
		require.Equal(t, "dcb", diagram.StyleDCB.String())
	})
}

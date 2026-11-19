//go:build unit

package docs_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/hpcsc/emod/docs"
)

func TestDSLReference(t *testing.T) {
	t.Run("read", func(t *testing.T) {
		t.Run("is the text of dsl-reference.md", func(t *testing.T) {
			text, err := os.ReadFile("dsl-reference.md")
			require.NoError(t, err)

			require.Equal(t, string(text), docs.DSLReference)
		})
	})

	t.Run("lint rules", func(t *testing.T) {
		t.Run("names the explain command of each rule about a screen", func(t *testing.T) {
			for _, rule := range []string{"screen/missing-image-asset", "bed"} {
				require.Contains(t, docs.DSLReference, "`emod lint --explain "+rule+"`")
			}
		})
	})
}

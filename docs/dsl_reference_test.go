//go:build unit

package docs_test

import (
	"os"
	"strings"
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

	t.Run("glossary", func(t *testing.T) {
		t.Run("says that a screen defines a glossary term, and does not list it among the constructs that define none", func(t *testing.T) {
			require.Contains(t, docs.DSLReference, "A screen defines a glossary term. `emod glossary` lists each screen")

			const noTerm = "**A construct that defines no term stays out of the glossary:**"
			start := strings.Index(docs.DSLReference, noTerm)
			require.NotEqual(t, -1, start)
			bullet, _, _ := strings.Cut(docs.DSLReference[start:], "\n")
			require.Contains(t, bullet, "`shows`")
			require.NotContains(t, bullet, "`screen`")
		})
	})

	t.Run("lint rules", func(t *testing.T) {
		t.Run("names the explain command of each rule about a screen", func(t *testing.T) {
			for _, rule := range []string{"screen/missing-image-asset", "bed", "screen/never-shown"} {
				require.Contains(t, docs.DSLReference, "`emod lint --explain "+rule+"`")
			}
		})
	})
}

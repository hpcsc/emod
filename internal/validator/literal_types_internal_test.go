//go:build unit

package validator

import (
	"maps"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestLiteralCheckers sits in the package rather than beside the rest of the
// validator's tests because literalCheckers is internal, and exporting it — or a
// reader of it — so an external test could reach it would widen the package's
// API for a test alone.
func TestLiteralCheckers(t *testing.T) {
	t.Run("knows exactly the seven types the language checks a payload literal against", func(t *testing.T) {
		require.Equal(t, []string{
			"bool", "date", "decimal", "int", "string", "timestamp", "uuid",
		}, slices.Sorted(maps.Keys(literalCheckers)))
	})
}

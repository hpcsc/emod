//go:build unit

package lsp_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// posIn returns the 0-based (line, character) of substr at or after the first
// occurrence of container in doc.
func posIn(t *testing.T, doc, container, substr string) (int, int) {
	t.Helper()
	cIdx := strings.Index(doc, container)
	require.GreaterOrEqual(t, cIdx, 0, "container %q not found", container)
	rel := strings.Index(doc[cIdx:], substr)
	require.GreaterOrEqual(t, rel, 0, "substr %q not found in container %q", substr, container)
	abs := cIdx + rel
	line := 0
	col := 0
	for i, ch := range doc {
		if i == abs {
			break
		}
		if ch == '\n' {
			line++
			col = 0
		} else {
			col++
		}
	}
	return line, col
}

//go:build unit

package linter

import (
	"testing"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/diagnostic"
	"github.com/stretchr/testify/require"
)

func TestInfoHelper(t *testing.T) {
	t.Run("returns entry with Info severity", func(t *testing.T) {
		entry := info(ast.Position{}, "test-rule", "test message")

		require.Equal(t, diagnostic.Info, entry.Severity)
	})

	t.Run("populates position fields", func(t *testing.T) {
		entry := info(ast.Position{
			Filename: "test.emod",
			Line:     5,
			Column:   3,
		}, "", "")

		require.Equal(t, "test.emod", entry.Filename)
		require.Equal(t, 5, entry.Line)
		require.Equal(t, 3, entry.Column)
	})

	t.Run("populates rule name and message", func(t *testing.T) {
		entry := info(ast.Position{}, "naming-rule", "something to note")

		require.Equal(t, "naming-rule", entry.RuleName)
		require.Equal(t, "something to note", entry.Message)
	})
}

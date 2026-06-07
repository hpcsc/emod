//go:build unit

package lsp_test

import (
	"testing"

	"github.com/hpcsc/emod/internal/diagnostic"
	"github.com/hpcsc/emod/internal/lsp"
	"github.com/stretchr/testify/require"
)

func TestConvertDiagnostics(t *testing.T) {
	t.Run("converts empty entries list to nil", func(t *testing.T) {
		result := lsp.ConvertDiagnostics("file:///test.emod", nil)
		require.Nil(t, result)

		result = lsp.ConvertDiagnostics("file:///test.emod", []*diagnostic.Entry{})
		require.Nil(t, result)
	})

	t.Run("error severity", func(t *testing.T) {
		t.Run("maps to severity 1", func(t *testing.T) {
			entries := []*diagnostic.Entry{
				{
					Filename: "test.emod",
					Line:     1,
					Column:   1,
					Message:  "syntax error",
					Severity: diagnostic.Error,
				},
			}

			result := lsp.ConvertDiagnostics("file:///test.emod", entries)
			require.Len(t, result, 1)
			require.Equal(t, lsp.DiagnosticSeverity(1), result[0].Severity)
		})

		t.Run("maps default severity (zero value) to 1", func(t *testing.T) {
			entries := []*diagnostic.Entry{
				{
					Filename: "test.emod",
					Line:     1,
					Column:   1,
					Message:  "syntax error",
				},
			}

			result := lsp.ConvertDiagnostics("file:///test.emod", entries)
			require.Len(t, result, 1)
			require.Equal(t, lsp.DiagnosticSeverity(1), result[0].Severity)
		})
	})

	t.Run("warning severity", func(t *testing.T) {
		t.Run("maps to severity 2", func(t *testing.T) {
			entries := []*diagnostic.Entry{
				{
					Filename: "test.emod",
					Line:     1,
					Column:   1,
					Message:  "unused variable",
					Severity: diagnostic.Warning,
				},
			}

			result := lsp.ConvertDiagnostics("file:///test.emod", entries)
			require.Len(t, result, 1)
			require.Equal(t, lsp.DiagnosticSeverity(2), result[0].Severity)
		})
	})

	t.Run("position mapping", func(t *testing.T) {
		t.Run("converts 1-based line to 0-based line", func(t *testing.T) {
			entries := []*diagnostic.Entry{
				{
					Filename: "test.emod",
					Line:     5,
					Column:   1,
					Message:  "error",
					Severity: diagnostic.Error,
				},
			}

			result := lsp.ConvertDiagnostics("file:///test.emod", entries)
			require.Equal(t, 4, result[0].Range.Start.Line)
			require.Equal(t, 4, result[0].Range.End.Line)
		})

		t.Run("converts 1-based column to 0-based column", func(t *testing.T) {
			entries := []*diagnostic.Entry{
				{
					Filename: "test.emod",
					Line:     3,
					Column:   10,
					Message:  "error",
					Severity: diagnostic.Error,
				},
			}

			result := lsp.ConvertDiagnostics("file:///test.emod", entries)
			require.Equal(t, 9, result[0].Range.Start.Character)
		})

		t.Run("end position is a single-character range", func(t *testing.T) {
			entries := []*diagnostic.Entry{
				{
					Filename: "test.emod",
					Line:     3,
					Column:   5,
					Message:  "error",
					Severity: diagnostic.Error,
				},
			}

			result := lsp.ConvertDiagnostics("file:///test.emod", entries)
			require.Equal(t, 2, result[0].Range.Start.Line)
			require.Equal(t, 4, result[0].Range.Start.Character)
			require.Equal(t, 2, result[0].Range.End.Line)
			require.Equal(t, 5, result[0].Range.End.Character)
		})
	})

	t.Run("message", func(t *testing.T) {
		t.Run("passes through entry message unchanged", func(t *testing.T) {
			entries := []*diagnostic.Entry{
				{
					Filename: "test.emod",
					Line:     1,
					Column:   1,
					Message:  `unrecognized keyword "foobar"; expected one of: model, actor, context`,
					Severity: diagnostic.Error,
				},
			}

			result := lsp.ConvertDiagnostics("file:///test.emod", entries)
			require.Len(t, result, 1)
			require.Equal(t, `unrecognized keyword "foobar"; expected one of: model, actor, context`, result[0].Message)
		})
	})

	t.Run("source", func(t *testing.T) {
		t.Run("is set to emod", func(t *testing.T) {
			entries := []*diagnostic.Entry{
				{
					Filename: "test.emod",
					Line:     1,
					Column:   1,
					Message:  "error",
					Severity: diagnostic.Error,
				},
			}

			result := lsp.ConvertDiagnostics("file:///test.emod", entries)
			require.Len(t, result, 1)
			require.Equal(t, "emod", result[0].Source)
		})
	})

	t.Run("converts multiple entries", func(t *testing.T) {
		entries := []*diagnostic.Entry{
			{
				Filename: "test.emod",
				Line:     3,
				Column:   5,
				Message:  "first error",
				Severity: diagnostic.Error,
			},
			{
				Filename: "test.emod",
				Line:     7,
				Column:   1,
				Message:  "first warning",
				Severity: diagnostic.Warning,
			},
			{
				Filename: "test.emod",
				Line:     10,
				Column:   15,
				Message:  "second error",
				Severity: diagnostic.Error,
			},
		}

		result := lsp.ConvertDiagnostics("file:///test.emod", entries)
		require.Len(t, result, 3)

		require.Equal(t, 2, result[0].Range.Start.Line)
		require.Equal(t, 4, result[0].Range.Start.Character)
		require.Equal(t, lsp.DiagnosticSeverity(1), result[0].Severity)
		require.Equal(t, "first error", result[0].Message)

		require.Equal(t, 6, result[1].Range.Start.Line)
		require.Equal(t, 0, result[1].Range.Start.Character)
		require.Equal(t, lsp.DiagnosticSeverity(2), result[1].Severity)
		require.Equal(t, "first warning", result[1].Message)

		require.Equal(t, 9, result[2].Range.Start.Line)
		require.Equal(t, 14, result[2].Range.Start.Character)
		require.Equal(t, lsp.DiagnosticSeverity(1), result[2].Severity)
		require.Equal(t, "second error", result[2].Message)
	})
}

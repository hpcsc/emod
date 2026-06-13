//go:build unit

package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/hpcsc/emod/internal/diagnostic"
	"github.com/stretchr/testify/require"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	fn()

	err = w.Close()
	require.NoError(t, err)
	os.Stdout = old

	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)
	return buf.String()
}

func TestFormatJSON(t *testing.T) {
	t.Run("info severity produces severity info in output and exit code 1", func(t *testing.T) {
		diags := []*diagnostic.Entry{
			{
				Filename: "test.emod",
				Line:     5,
				Column:   3,
				Severity: diagnostic.Info,
				RuleName: "naming-hint",
				Message:  "something to note",
			},
		}

		err := formatJSON(diags)

		var lintErr *LintError
		require.True(t, errors.As(err, &lintErr))
		require.Equal(t, 1, lintErr.ExitCode)

		output := captureStdout(t, func() {
			_ = formatJSON(diags)
		})

		var entries []map[string]interface{}
		err = json.Unmarshal([]byte(output), &entries)
		require.NoError(t, err)
		require.Len(t, entries, 1)
		require.Equal(t, "info", entries[0]["severity"])
		require.Equal(t, "naming-hint", entries[0]["rule"])
		require.Equal(t, "test.emod", entries[0]["file"])
		require.Equal(t, float64(5), entries[0]["line"])
		require.Equal(t, "something to note", entries[0]["message"])
	})

	t.Run("exit code 2 when info and error entries are both present", func(t *testing.T) {
		diags := []*diagnostic.Entry{
			{
				Filename: "test.emod",
				Line:     5,
				Severity: diagnostic.Info,
				RuleName: "naming-hint",
				Message:  "info message",
			},
			{
				Filename: "test.emod",
				Line:     10,
				Severity: diagnostic.Error,
				RuleName: "clickbait-event",
				Message:  "error message",
			},
		}

		err := formatJSON(diags)

		var lintErr *LintError
		require.True(t, errors.As(err, &lintErr))
		require.Equal(t, 2, lintErr.ExitCode)
	})
}

func TestFormatText(t *testing.T) {
	t.Run("info severity entry is formatted consistently with other severities", func(t *testing.T) {
		diags := []*diagnostic.Entry{
			{
				Filename: "test.emod",
				Line:     5,
				Column:   3,
				Severity: diagnostic.Info,
				RuleName: "naming-hint",
				Message:  "something to note",
			},
		}

		err := formatText(diags)

		var lintErr *LintError
		require.True(t, errors.As(err, &lintErr))
		require.Contains(t, lintErr.Message, "test.emod:5:")
		require.Contains(t, lintErr.Message, "naming-hint")
		require.Contains(t, lintErr.Message, "something to note")
	})
}

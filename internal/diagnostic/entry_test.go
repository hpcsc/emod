//go:build unit

package diagnostic_test

import (
	"testing"

	"github.com/hpcsc/emod/internal/diagnostic"
	"github.com/stretchr/testify/require"
)

func TestDiagnosticString(t *testing.T) {
	t.Run("formats as filename:line: message", func(t *testing.T) {
		d := &diagnostic.Entry{
			Filename: "minimal.emod",
			Line:     3,
			Column:   1,
			Message:  `unrecognized keyword "foobar"; expected one of: model, actor, context`,
		}

		require.Equal(t, `minimal.emod:3: unrecognized keyword "foobar"; expected one of: model, actor, context`, d.String())
	})

	t.Run("formats unclosed brace error", func(t *testing.T) {
		d := &diagnostic.Entry{
			Filename: "minimal.emod",
			Line:     5,
			Column:   1,
			Message:  `unclosed brace for "context" block opened at line 3`,
		}

		require.Equal(t, `minimal.emod:5: unclosed brace for "context" block opened at line 3`, d.String())
	})

	t.Run("includes rule name in brackets when present", func(t *testing.T) {
		d := &diagnostic.Entry{
			Filename: "file.emod",
			Line:     5,
			Column:   1,
			RuleName: "state-obsession",
			Message:  "message",
		}

		require.Equal(t, "file.emod:5: [state-obsession] message", d.String())
	})

	t.Run("produces original format when rule name is empty", func(t *testing.T) {
		d := &diagnostic.Entry{
			Filename: "file.emod",
			Line:     3,
			Column:   1,
			Message:  "message",
		}

		require.Equal(t, "file.emod:3: message", d.String())
	})

}

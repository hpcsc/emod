//go:build unit

package diagnostic_test

import (
	"testing"

	"github.com/hpcsc/emod/internal/diagnostic"
	"github.com/stretchr/testify/require"
)

func TestEntry(t *testing.T) {
	t.Run("severity", func(t *testing.T) {
		t.Run("each level reports the name tools match on", func(t *testing.T) {
			require.Equal(t, "info", diagnostic.Info.String())
			require.Equal(t, "warning", diagnostic.Warning.String())
			require.Equal(t, "error", diagnostic.Error.String())
		})
	})

	t.Run("string", func(t *testing.T) {
		t.Run("reads as filename:line: message", func(t *testing.T) {
			d := &diagnostic.Entry{
				Filename: "minimal.emod",
				Line:     3,
				Column:   1,
				Message:  `unrecognized keyword "foobar"; expected one of: model, actor, context`,
			}

			require.Equal(t,
				`minimal.emod:3: unrecognized keyword "foobar"; expected one of: model, actor, context`,
				d.String())
		})

		t.Run("names the rule in brackets when one produced the entry", func(t *testing.T) {
			d := &diagnostic.Entry{
				Filename: "file.emod",
				Line:     5,
				Column:   1,
				RuleName: "state-obsession",
				Message:  "message",
			}

			require.Equal(t, "file.emod:5: [state-obsession] message", d.String())
		})

		t.Run("omits the brackets when no rule produced the entry", func(t *testing.T) {
			d := &diagnostic.Entry{
				Filename: "file.emod",
				Line:     3,
				Column:   1,
				Message:  "message",
			}

			require.Equal(t, "file.emod:3: message", d.String())
		})
	})
}

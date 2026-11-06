//go:build unit

package diagnostic_test

import (
	"encoding/json"
	"testing"

	"github.com/hpcsc/emod/internal/diagnostic"
	"github.com/stretchr/testify/require"
)

func TestRecord(t *testing.T) {
	t.Run("records", func(t *testing.T) {
		t.Run("each entry becomes one record, in the order of the entries", func(t *testing.T) {
			entries := []*diagnostic.Entry{
				{
					Filename: "ordering.emod",
					Line:     22,
					Column:   7,
					Message:  `event "OrderUpdated" uses a generic state-change suffix`,
					Severity: diagnostic.Warning,
					RuleName: "state-obsession",
				},
				{
					Filename: "billing.emod",
					Line:     3,
					Column:   1,
					Message:  `event "InvoiceSent" does not exist`,
					Severity: diagnostic.Error,
				},
			}

			records := diagnostic.Records(entries)

			require.Equal(t, []diagnostic.Record{
				{
					File:     "ordering.emod",
					Line:     22,
					Rule:     "state-obsession",
					Severity: "warning",
					Message:  `event "OrderUpdated" uses a generic state-change suffix`,
				},
				{
					File:     "billing.emod",
					Line:     3,
					Severity: "error",
					Message:  `event "InvoiceSent" does not exist`,
				},
			}, records)
		})
	})

	t.Run("json", func(t *testing.T) {
		t.Run("no entries encode as an empty list, not null", func(t *testing.T) {
			out, err := json.Marshal(diagnostic.Records(nil))

			require.NoError(t, err)
			require.Equal(t, "[]", string(out))
		})

		t.Run("a record with a rule encodes its keys in the order file, line, rule, severity, message", func(t *testing.T) {
			out, err := json.Marshal(diagnostic.Records([]*diagnostic.Entry{{
				Filename: "examples/error_diagnostics_test.emod",
				Line:     64,
				Message:  `command "SendEmail" is orphaned (no flow references it)`,
				Severity: diagnostic.Error,
				RuleName: "orphan-command",
			}}))

			require.NoError(t, err)
			require.Equal(t,
				`[{"file":"examples/error_diagnostics_test.emod","line":64,"rule":"orphan-command","severity":"error","message":"command \"SendEmail\" is orphaned (no flow references it)"}]`,
				string(out))
		})

		t.Run("a record with no rule has no rule key", func(t *testing.T) {
			out, err := json.Marshal(diagnostic.Records([]*diagnostic.Entry{{
				Filename: "examples/error_diagnostics_test.emod",
				Line:     43,
				Message:  `event "GuestCheckedOut" does not exist`,
				Severity: diagnostic.Error,
			}}))

			require.NoError(t, err)
			require.Equal(t,
				`[{"file":"examples/error_diagnostics_test.emod","line":43,"severity":"error","message":"event \"GuestCheckedOut\" does not exist"}]`,
				string(out))
		})
	})
}

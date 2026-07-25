//go:build unit

package cue_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hpcsc/emod/internal/cue"
	"github.com/stretchr/testify/require"
)

// requireCue returns the cue binary, skipping when it is not installed. The
// schema is only meaningful to the real compiler, so a stand-in would not tell
// us whether the embedded copy still works.
func requireCue(t *testing.T) string {
	t.Helper()

	bin, err := exec.LookPath("cue")
	if err != nil {
		t.Skip("cue not installed; skipping schema conformance check")
	}

	return bin
}

// vetAgainstModel checks modelJSON against the embedded schema's #Model
// definition. The -d flag is what binds the data to the definition; without it
// cue only checks that both files parse.
func vetAgainstModel(t *testing.T, cueBin, modelJSON string) ([]byte, error) {
	t.Helper()

	dir := t.TempDir()
	schemaPath := filepath.Join(dir, "schema.cue")
	require.NoError(t, os.WriteFile(schemaPath, []byte(cue.Schema), 0o644))

	modelPath := filepath.Join(dir, "model.json")
	require.NoError(t, os.WriteFile(modelPath, []byte(modelJSON), 0o644))

	return exec.Command(cueBin, "vet", "-d", "#Model", schemaPath, modelPath).CombinedOutput()
}

func TestSchema(t *testing.T) {
	t.Run("is self-contained so the embedded copy needs no module fetch", func(t *testing.T) {
		for _, line := range strings.Split(cue.Schema, "\n") {
			require.False(t, strings.HasPrefix(strings.TrimSpace(line), "import "),
				"schema must not import anything, found: %s", line)
		}
	})

	t.Run("compiles", func(t *testing.T) {
		cueBin := requireCue(t)

		dir := t.TempDir()
		schemaPath := filepath.Join(dir, "schema.cue")
		require.NoError(t, os.WriteFile(schemaPath, []byte(cue.Schema), 0o644))

		output, err := exec.Command(cueBin, "vet", schemaPath).CombinedOutput()

		require.NoError(t, err, "cue vet failed: %s", output)
	})

	t.Run("accepts a model using every element the language offers", func(t *testing.T) {
		cueBin := requireCue(t)

		output, err := vetAgainstModel(t, cueBin, fullModelJSON)

		require.NoError(t, err, "schema rejected a valid model: %s", output)
	})

	t.Run("rejects a model with no name", func(t *testing.T) {
		cueBin := requireCue(t)

		output, err := vetAgainstModel(t, cueBin, `{"actors":[{"name":"Guest"}]}`)

		require.Error(t, err, "schema accepted a nameless model")
		require.Contains(t, string(output), "name")
	})

	t.Run("rejects a model whose name is not a string", func(t *testing.T) {
		cueBin := requireCue(t)

		output, err := vetAgainstModel(t, cueBin, `{"name":42}`)

		require.Error(t, err, "schema accepted a numeric model name")
		require.Contains(t, string(output), "name")
	})

	t.Run("rejects a command field that has no type", func(t *testing.T) {
		cueBin := requireCue(t)

		output, err := vetAgainstModel(t, cueBin, strings.Replace(fullModelJSON,
			`{"name":"guestId","type":"string","modifier":"required"}`,
			`{"name":"guestId","modifier":"required"}`, 1))

		require.Error(t, err, "schema accepted a field with no type")
		require.Contains(t, string(output), "type")
	})

	t.Run("rejects a flow that names only one side", func(t *testing.T) {
		cueBin := requireCue(t)

		output, err := vetAgainstModel(t, cueBin, strings.Replace(fullModelJSON,
			`{"command_name":"MakeReservation","event_name":"ReservationMade"}`,
			`{"command_name":"MakeReservation"}`, 1))

		require.Error(t, err, "schema accepted a flow with no event")
		require.Contains(t, string(output), "event_name")
	})
}

// fullModelJSON exercises every definition the schema declares, so a definition
// that gets dropped or renamed makes the acceptance test above fail.
const fullModelJSON = `{
  "comments": [{"text": "# Full model"}],
  "name": "Full",
  "actors": [{"name": "Guest"}],
  "contexts": [{
    "name": "Reservations",
    "aggregates": [{
      "name": "Reservation",
      "slices": [{
        "name": "Make Reservation",
        "trigger": {"kind": "UI", "name": "Form", "actor": "Guest", "reads": "RoomsView"},
        "commands": [{
          "name": "MakeReservation",
          "fields": [{"name":"guestId","type":"string","modifier":"required"}]
        }],
        "events": [{
          "name": "ReservationMade",
          "source": "external",
          "external_name": "Stripe",
          "fields": [{"name": "reservationId", "type": "int", "modifier": "optional"}]
        }],
        "views": [{
          "name": "RoomsView",
          "fields": [{"name": "roomId", "type": "string", "modifier": "required"}],
          "subscribes": ["ReservationMade"]
        }],
        "automations": [{
          "name": "Notifier",
          "trigger_event": "ReservationMade",
          "command": "SendEmail",
          "target_context": "Notifications"
        }],
        "translations": [{
          "name": "BookingImport",
          "external_system": "Booking.com",
          "reads": "RoomsView",
          "command": "MakeReservation",
          "event": {"name": "BookingImported"}
        }],
        "flows": [{"command_name":"MakeReservation","event_name":"ReservationMade"}]
      }]
    }]
  }]
}
`

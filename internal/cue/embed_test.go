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

	t.Run("rejects a description that is not prose", func(t *testing.T) {
		cueBin := requireCue(t)

		output, err := vetAgainstModel(t, cueBin, `{"name":"Full","description":42}`)

		require.Error(t, err, "schema accepted a numeric description")
		require.Contains(t, string(output), "description")
	})

	t.Run("rejects a command field that has no type", func(t *testing.T) {
		cueBin := requireCue(t)

		output, err := vetAgainstModel(t, cueBin, strings.Replace(fullModelJSON,
			`{"name":"guestId","type":"string","modifier":"required"}`,
			`{"name":"guestId","modifier":"required"}`, 1))

		require.Error(t, err, "schema accepted a field with no type")
		require.Contains(t, string(output), "type")
	})

	t.Run("rejects an automation whose activation event is spelled the retired way", func(t *testing.T) {
		cueBin := requireCue(t)

		output, err := vetAgainstModel(t, cueBin, strings.Replace(fullModelJSON,
			`"on_event": "ReservationMade"`,
			`"trigger_event": "ReservationMade"`, 1))

		require.Error(t, err, "schema accepted an automation keyed trigger_event")
		require.Contains(t, string(output), "trigger_event")
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
  "description": "How the hotel takes and imports bookings",
  "actors": [{"name": "Guest", "description": "A person booking a room"}],
  "contexts": [{
    "name": "Reservations",
    "description": "What the hotel knows before the guest arrives",
    "invariants": [{
      "comments": [{"text": "# The hotel never double-books"}],
      "name": "OneGuestPerRoom",
      "statement": "A room holds at most one guest for any night"
    }],
    "aggregates": [{
      "name": "Reservation",
      "description": "One guest holding one room",
      "invariants": [{
        "name": "OneRoomPerReservation",
        "statement": "A reservation covers exactly one room"
      }],
      "slices": [{
        "name": "Make Reservation",
        "description": "A guest books a room",
        "trigger": {
          "kind": "UI",
          "name": "Form",
          "description": "The booking form on the public site",
          "actor": "Guest",
          "reads": "RoomsView"
        },
        "commands": [{
          "name": "MakeReservation",
          "description": "Ask the hotel to hold a room",
          "fields": [{"name":"guestId","type":"string","modifier":"required"}]
        }],
        "events": [{
          "name": "ReservationMade",
          "description": "A room is held for a guest",
          "source": "external",
          "external_name": "Stripe",
          "fields": [{"name": "reservationId", "type": "int", "modifier": "optional"}]
        }],
        "views": [{
          "name": "RoomsView",
          "description": "Every room with the stage it has reached",
          "fields": [{"name": "roomId", "type": "string", "modifier": "required"}],
          "subscribes": ["ReservationMade"]
        }],
        "automations": [{
          "name": "Notifier",
          "description": "Tells the guest their room is held",
          "on_event": "ReservationMade",
          "reads": "RoomsView",
          "command": "SendEmail",
          "target_context": "Notifications"
        }],
        "translations": [{
          "name": "BookingImport",
          "description": "Restates a partner webhook in the hotel's language",
          "external_system": "Booking.com",
          "reads": "RoomsView",
          "command": "MakeReservation",
          "event": {"name": "BookingImported", "description": "A partner reported a booking"}
        }],
        "flows": [{"command_name":"MakeReservation","event_name":"ReservationMade"}],
        "specs": [{
          "comments": [{"text": "# The room has to be free"}],
          "name": "holds a room no guest holds",
          "when": "MakeReservation",
          "then": {"events": ["ReservationMade"]}
        }, {
          "name": "refuses a room another guest holds",
          "given": ["ReservationMade"],
          "when": "MakeReservation",
          "then": {"rejected": "OneRoomPerReservation"}
        }]
      }]
    }]
  }]
}
`

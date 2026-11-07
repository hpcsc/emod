//go:build unit

package cli_test

import (
	"errors"
	"testing"

	"github.com/hpcsc/emod/internal/cli"
	"github.com/hpcsc/emod/internal/query"
	"github.com/stretchr/testify/require"
)

const deskModel = `context "Desk" {
  aggregate "Stay" {
    slice "Check In" {
      trigger "Arrival Screen" {
        actor = Guest
      }

      command "CheckIn" {
      }

      event "CheckedIn" {
      }

      flow = <<-FLOW
        command -> event: CheckIn -> CheckedIn
      FLOW
    }

    slice "Log Check In" {
      event "CheckIn" {
      }
    }
  }
}
`

func TestReferences(t *testing.T) {
	t.Run("text", func(t *testing.T) {
		t.Run("prints the declaration, then each reference in the group for its role", func(t *testing.T) {
			output := captureStdout(t, func() {
				require.NoError(t, cli.RunReferences("../../examples/all_patterns.emod", "RoomReserved", "", "text"))
			})

			require.Equal(t, `event RoomReserved   Reservations > Reservation, slice "Reserve a Room"   :49

produced by
  :66   flow        ReserveRoom -> RoomReserved          "Reserve a Room"
read by
  :93   subscribes  view AvailableRoomsView              "View Available Rooms"
  :161  subscribes  view PendingConfirmationsView        "Send Confirmation Email"
  :172  on          automation ConfirmationEmailReactor  "Send Confirmation Email"
  :295  subscribes  view UnconfirmedReservationsView     "View Unconfirmed Reservations"
named in specs
  :72   then   "holds a room nobody has booked"
  :76   given  "refuses a room already held over the same nights"
  :105  given  "lists the rooms a guest can still book"
  :143  given  "closes a stay that is under way"
  :149  given  "refuses to close a stay that already ended"
  :183  when   "asks for a confirmation once a booking has settled"
  :272  given  "asks for a release on every overnight sweep"
  :277  given  "releases a room nobody confirmed"
  :283  given  "refuses to release a stay that already ended"
  :305  given  "lists the bookings the sweep has left to do"
`, output)
		})

		t.Run("a name that two kinds declare prints one block for each kind, and a group with no reference says none", func(t *testing.T) {
			path := writeTemp(t, "desk.emod", deskModel)

			output := captureStdout(t, func() {
				require.NoError(t, cli.RunReferences(path, "CheckIn", "", "text"))
			})

			require.Equal(t, `command CheckIn   Desk > Stay, slice "Check In"   :8

issued by
  :4   issues  trigger "Arrival Screen"  "Check In"
produces
  :15  flow    CheckIn -> CheckedIn      "Check In"
refused by
  none
named in specs
  none

event CheckIn   Desk > Stay, slice "Log Check In"   :20

produced by
  none
read by
  none
named in specs
  none
`, output)
		})

		t.Run("the kind prints only the declaration of that kind", func(t *testing.T) {
			path := writeTemp(t, "desk.emod", deskModel)

			output := captureStdout(t, func() {
				require.NoError(t, runCommandLine(t, "emod", "references", path, "CheckIn", "--kind", "event"))
			})

			require.Equal(t, `event CheckIn   Desk > Stay, slice "Log Check In"   :20

produced by
  none
read by
  none
named in specs
  none
`, output)
		})
	})

	t.Run("json", func(t *testing.T) {
		t.Run("gives each reference with its group, its keyword, its line, the other element and the slice", func(t *testing.T) {
			path := writeTemp(t, "desk.emod", deskModel)

			output := captureStdout(t, func() {
				require.NoError(t, cli.RunReferences(path, "CheckedIn", "", "json"))
			})

			require.JSONEq(t, `[{
				"kind": "event",
				"name": "CheckedIn",
				"context": "Desk",
				"aggregate": "Stay",
				"slice": "Check In",
				"line": 11,
				"references": [
					{"group": "produced by", "keyword": "flow", "line": 15, "element": {"kind": "command", "name": "CheckIn"}, "slice": "Check In"}
				]
			}]`, output)
		})
	})

	t.Run("errors", func(t *testing.T) {
		t.Run("a name that the model does not declare exits with code 1 and lists the closest names", func(t *testing.T) {
			path := writeTemp(t, "desk.emod", deskModel)

			err := cli.RunReferences(path, "CheckedOn", "", "text")

			var unknown *query.UnknownNameError
			require.True(t, errors.As(err, &unknown))
			require.EqualError(t, err, `the model declares no command, event, view or invariant named "CheckedOn"; the closest names are CheckedIn, CheckIn`)
			var lintErr *cli.LintError
			require.True(t, errors.As(err, &lintErr))
			require.Equal(t, 1, lintErr.ExitCode)
		})

		t.Run("a kind that the command does not answer for is refused", func(t *testing.T) {
			path := writeTemp(t, "desk.emod", deskModel)

			err := cli.RunReferences(path, "Arrival Screen", "trigger", "text")

			require.ErrorIs(t, err, cli.ErrUnsupportedKind)
			require.EqualError(t, err, `unsupported kind "trigger"; supported kinds: command, event, view, invariant`)
		})

		t.Run("a format other than text and json is refused", func(t *testing.T) {
			path := writeTemp(t, "desk.emod", deskModel)

			err := cli.RunReferences(path, "CheckIn", "", "xml")

			require.ErrorIs(t, err, cli.ErrUnsupportedFormat)
		})

		t.Run("a command line with no name is refused", func(t *testing.T) {
			path := writeTemp(t, "desk.emod", deskModel)

			err := cli.RunReferences(path, "", "", "text")

			require.ErrorIs(t, err, cli.ErrMissingName)
			require.EqualError(t, err, "references requires a file and a name: emod references <file> <name>")
		})

		t.Run("a file that does not parse stops with its parse diagnostics", func(t *testing.T) {
			path := writeTemp(t, "invalid.emod", invalidEmod)

			err := cli.RunReferences(path, "CheckIn", "", "text")

			require.ErrorContains(t, err, path+":1:")
			var lintErr *cli.LintError
			require.True(t, errors.As(err, &lintErr))
			require.Equal(t, 1, lintErr.ExitCode)
		})
	})
}

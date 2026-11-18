//go:build unit

package cli_test

import (
	"errors"
	"testing"

	"github.com/hpcsc/emod/internal/cli"
	"github.com/hpcsc/emod/internal/query"
	"github.com/stretchr/testify/require"
)

const deskModel = `screen "ArrivalScreen" {
}

context "Desk" {
  aggregate "Stay" {
    slice "Check In" {
      shows {
        screen  = ArrivalScreen
        command = CheckIn
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

const checkoutScreenModel = `screen "CheckoutPage" {
}

context "Shop" {
  slice "Show Cart" {
    view "CartView" {
    }
  }

  slice "Apply Discount" {
    shows {
      screen  = CheckoutPage
      reads   = CartView
      command = ApplyDiscount
    }

    command "ApplyDiscount" {
    }
  }
}
`

func TestReferences(t *testing.T) {
	t.Run("text", func(t *testing.T) {
		t.Run("a command lists the shows block that names it under issued by, and a view the shows block that reads it under read by", func(t *testing.T) {
			path := writeTemp(t, "checkout.emod", checkoutScreenModel)

			output := captureStdout(t, func() {
				require.NoError(t, cli.RunReferences(path, "ApplyDiscount", "", "text"))
				require.NoError(t, cli.RunReferences(path, "CartView", "", "text"))
			})

			require.Equal(t, `command ApplyDiscount   Shop, slice "Apply Discount"   :17

issued by
  :14  command  screen CheckoutPage  "Apply Discount"
produces
  none
refused by
  none
named in specs
  none
view CartView   Shop, slice "Show Cart"   :6

built from
  none
read by
  :13  reads  screen CheckoutPage  "Apply Discount"
named in specs
  none
`, output)
		})

		t.Run("prints the declaration, then each reference in the group for its role", func(t *testing.T) {
			output := captureStdout(t, func() {
				require.NoError(t, cli.RunReferences("../../examples/all_patterns.emod", "RoomReserved", "", "text"))
			})

			require.Equal(t, `event RoomReserved   Reservations > Reservation, slice "Reserve a Room"   :53

produced by
  :70   flow        ReserveRoom -> RoomReserved          "Reserve a Room"
read by
  :97   subscribes  view AvailableRoomsView              "View Available Rooms"
  :165  subscribes  view PendingConfirmationsView        "Send Confirmation Email"
  :176  on          automation ConfirmationEmailReactor  "Send Confirmation Email"
  :299  subscribes  view UnconfirmedReservationsView     "View Unconfirmed Reservations"
named in specs
  :76   then   "holds a room nobody has booked"
  :80   given  "refuses a room already held over the same nights"
  :109  given  "lists the rooms a guest can still book"
  :147  given  "closes a stay that is under way"
  :153  given  "refuses to close a stay that already ended"
  :187  when   "asks for a confirmation once a booking has settled"
  :276  given  "asks for a release on every overnight sweep"
  :281  given  "releases a room nobody confirmed"
  :287  given  "refuses to release a stay that already ended"
  :309  given  "lists the bookings the sweep has left to do"
`, output)
		})

		t.Run("a name that two kinds declare prints one block for each kind, and a group with no reference says none", func(t *testing.T) {
			path := writeTemp(t, "desk.emod", deskModel)

			output := captureStdout(t, func() {
				require.NoError(t, cli.RunReferences(path, "CheckIn", "", "text"))
			})

			require.Equal(t, `command CheckIn   Desk > Stay, slice "Check In"   :12

issued by
  :9   command  screen ArrivalScreen  "Check In"
produces
  :19  flow     CheckIn -> CheckedIn  "Check In"
refused by
  none
named in specs
  none

event CheckIn   Desk > Stay, slice "Log Check In"   :24

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

			require.Equal(t, `event CheckIn   Desk > Stay, slice "Log Check In"   :24

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
		t.Run("gives a shows block that issues the command with the keyword command, the line of the entry and the screen", func(t *testing.T) {
			path := writeTemp(t, "checkout.emod", checkoutScreenModel)

			output := captureStdout(t, func() {
				require.NoError(t, cli.RunReferences(path, "ApplyDiscount", "", "json"))
			})

			require.JSONEq(t, `[{
				"kind": "command",
				"name": "ApplyDiscount",
				"context": "Shop",
				"slice": "Apply Discount",
				"line": 17,
				"references": [
					{"group": "issued by", "keyword": "command", "line": 14, "element": {"kind": "screen", "name": "CheckoutPage"}, "slice": "Apply Discount"}
				]
			}]`, output)
		})

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
				"line": 15,
				"references": [
					{"group": "produced by", "keyword": "flow", "line": 19, "element": {"kind": "command", "name": "CheckIn"}, "slice": "Check In"}
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

			err := cli.RunReferences(path, "ArrivalScreen", "trigger", "text")

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

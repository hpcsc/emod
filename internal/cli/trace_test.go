//go:build unit

package cli_test

import (
	"errors"
	"testing"

	"github.com/hpcsc/emod/internal/cli"
	"github.com/hpcsc/emod/internal/query"
	"github.com/stretchr/testify/require"
)

const hotelTraceModel = `context "Hotel" {
  aggregate "Stay" {
    invariants {
      RoomFree = "A room holds one booking at a time"
    }

    slice "Book Room" {
      trigger "Booking Form" {
        reads = RoomsView
      }

      command "BookRoom" {
      }

      event "RoomBooked" {
      }

      flow = <<-FLOW
        command -> event:    BookRoom -> RoomBooked
        command -> rejected: BookRoom -> RoomFree
      FLOW
    }

    slice "Show Rooms" {
      view "RoomsView" {
        subscribes = [RoomBooked]
      }
    }

    slice "Confirm Booking" {
      automation "Confirmer" {
        on      = RoomBooked
        after   = "15m"
        command = SendConfirmation

        target {
          context = Mail
        }
      }
    }

    slice "Sweep Bookings" {
      automation "Sweeper" {
        every   = "0 2 * * *"
        reads   = RoomsView
        command = ReleaseRoom
      }

      command "ReleaseRoom" {
      }

      event "RoomReleased" {
      }

      flow = <<-FLOW
        command -> event: ReleaseRoom -> RoomReleased
      FLOW
    }

    slice "Import Booking" {
      translation "PartnerImport" {
        external_system = "Partner API"
        command         = BookRoom

        event "BookingImported" {
        }
      }
    }
  }
}

context "Mail" {
  aggregate "Message" {
    slice "Send Confirmation" {
      command "SendConfirmation" {
      }

      event "ConfirmationSent" {
      }

      flow = <<-FLOW
        command -> event: SendConfirmation -> ConfirmationSent
      FLOW
    }
  }
}
`

func TestTrace(t *testing.T) {
	t.Run("text", func(t *testing.T) {
		t.Run("prints the tree with the edge, the kind, the context, the slice and the line of each element", func(t *testing.T) {
			path := writeTemp(t, "hotel.emod", hotelTraceModel)

			output := captureStdout(t, func() {
				require.NoError(t, cli.RunTrace(path, "RoomBooked", "text", query.TraceOptions{NoViews: true}))
			})

			require.Equal(t, `3 elements follow from RoomBooked

RoomBooked                         event       Hotel  "Book Room"          :15
  -[on, after 15m]-> Confirmer     automation  Hotel  "Confirm Booking"    :31
    -[command]-> SendConfirmation  command     Mail   "Send Confirmation"  :75  (Hotel -> Mail)
      -[flow]-> ConfirmationSent   event       Mail   "Send Confirmation"  :78
`, output)
		})

		t.Run("an upstream trace draws each edge backward and names the contexts in the order that the work goes", func(t *testing.T) {
			path := writeTemp(t, "hotel.emod", hotelTraceModel)

			output := captureStdout(t, func() {
				require.NoError(t, cli.RunTrace(path, "SendConfirmation", "text", query.TraceOptions{Upstream: true, NoViews: true}))
			})

			require.Equal(t, `6 elements lead to SendConfirmation

SendConfirmation                    command          Mail   "Send Confirmation"  :75
  <-[command]- Confirmer            automation       Hotel  "Confirm Booking"    :31  (Hotel -> Mail)
    <-[on, after 15m]- RoomBooked   event            Hotel  "Book Room"          :15
      <-[flow]- BookRoom            command          Hotel  "Book Room"          :12
        <-[issues]- "Booking Form"  trigger          Hotel  "Book Room"          :8
        <-[command]- PartnerImport  translation      Hotel  "Import Booking"     :61
          <-[from]- "Partner API"   external_system  Hotel  "Import Booking"     :62
`, output)
		})

		t.Run("an element that nothing follows says so", func(t *testing.T) {
			path := writeTemp(t, "hotel.emod", hotelTraceModel)

			output := captureStdout(t, func() {
				require.NoError(t, cli.RunTrace(path, "ConfirmationSent", "text", query.TraceOptions{}))
			})

			require.Equal(t, `nothing follows from ConfirmationSent

ConfirmationSent  event  Mail  "Send Confirmation"  :78
`, output)
		})

		t.Run("the flags after the file and the element apply", func(t *testing.T) {
			path := writeTemp(t, "hotel.emod", hotelTraceModel)

			output := captureStdout(t, func() {
				require.NoError(t, runCommandLine(t, "emod", "trace", path, "ConfirmationSent", "--upstream", "--no-views", "--kind", "event"))
			})

			require.Contains(t, output, "7 elements lead to ConfirmationSent\n")
		})
	})

	t.Run("json", func(t *testing.T) {
		t.Run("gives the root, and each step with its label, its element, its place, its line and the element that reaches it", func(t *testing.T) {
			path := writeTemp(t, "hotel.emod", hotelTraceModel)

			output := captureStdout(t, func() {
				require.NoError(t, cli.RunTrace(path, "Confirmer", "json", query.TraceOptions{}))
			})

			require.JSONEq(t, `{
				"root": {"kind": "automation", "name": "Confirmer", "context": "Hotel", "aggregate": "Stay", "slice": "Confirm Booking", "line": 31},
				"steps": [
					{
						"label": "command",
						"kind": "command", "name": "SendConfirmation", "context": "Mail", "aggregate": "Message", "slice": "Send Confirmation", "line": 75,
						"from": {"kind": "automation", "name": "Confirmer"},
						"crosses_context": true,
						"depth": 1
					},
					{
						"label": "flow",
						"kind": "event", "name": "ConfirmationSent", "context": "Mail", "aggregate": "Message", "slice": "Send Confirmation", "line": 78,
						"from": {"kind": "command", "name": "SendConfirmation"},
						"depth": 2
					}
				]
			}`, output)
		})
	})

	t.Run("errors", func(t *testing.T) {
		t.Run("a name that two kinds declare exits with code 1, lists the kinds and asks for one", func(t *testing.T) {
			path := writeTemp(t, "desk.emod", deskModel)

			err := cli.RunTrace(path, "CheckIn", "text", query.TraceOptions{})

			var ambiguous *query.AmbiguousNameError
			require.True(t, errors.As(err, &ambiguous))
			require.EqualError(t, err, `the model declares "CheckIn" more than one time: as command in Desk > Stay, as event in Desk > Stay; pick one with --kind`)
			var lintErr *cli.LintError
			require.True(t, errors.As(err, &lintErr))
			require.Equal(t, 1, lintErr.ExitCode)
		})

		t.Run("a name that the model does not declare exits with code 1 and lists the closest names", func(t *testing.T) {
			path := writeTemp(t, "hotel.emod", hotelTraceModel)

			err := cli.RunTrace(path, "Confirmr", "text", query.TraceOptions{})

			require.EqualError(t, err, `the model declares no element named "Confirmr"; the closest names are Confirmer`)
			var lintErr *cli.LintError
			require.True(t, errors.As(err, &lintErr))
			require.Equal(t, 1, lintErr.ExitCode)
		})

		t.Run("a kind that no element has is refused", func(t *testing.T) {
			path := writeTemp(t, "hotel.emod", hotelTraceModel)

			err := cli.RunTrace(path, "RoomBooked", "text", query.TraceOptions{Kind: "screen"})

			require.ErrorIs(t, err, cli.ErrUnsupportedKind)
			require.EqualError(t, err, `unsupported kind "screen"; supported kinds: command, event, view, trigger, automation, translation, invariant, external_system`)
		})

		t.Run("a command line with no element is refused", func(t *testing.T) {
			path := writeTemp(t, "hotel.emod", hotelTraceModel)

			err := cli.RunTrace(path, "", "text", query.TraceOptions{})

			require.ErrorIs(t, err, cli.ErrMissingName)
			require.EqualError(t, err, "trace requires a file and an element: emod trace <file> <from>")
		})

		t.Run("a file that does not parse stops with its parse diagnostics", func(t *testing.T) {
			path := writeTemp(t, "invalid.emod", invalidEmod)

			err := cli.RunTrace(path, "RoomBooked", "text", query.TraceOptions{})

			require.ErrorContains(t, err, path+":1:")
			var lintErr *cli.LintError
			require.True(t, errors.As(err, &lintErr))
			require.Equal(t, 1, lintErr.ExitCode)
		})
	})
}

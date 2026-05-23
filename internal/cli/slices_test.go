//go:build unit

package cli_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/hpcsc/emod/internal/cli"
	"github.com/stretchr/testify/require"
)

func TestSlices(t *testing.T) {
	t.Run("prints table for valid file with all pattern types", func(t *testing.T) {
		path := writeTemp(t, "valid.emod", validEmod)

		output := captureStdout(t, func() {
			err := cli.RunSlices(path)
			require.NoError(t, err)
		})

		require.Contains(t, output, "SLICE")
		require.Contains(t, output, "PATTERN")
		require.Contains(t, output, "CONTEXT")
		require.Contains(t, output, "KEY ELEMENTS")
		require.Contains(t, output, "Make Reservation")
		require.Contains(t, output, "command")
		require.Contains(t, output, "Reservations")
		require.Contains(t, output, "MakeReservation, ReservationMade")
		require.Contains(t, output, "View Reservations")
		require.Contains(t, output, "view")
		require.Contains(t, output, "ReservationsView")
		require.Contains(t, output, "Auto Confirm Reservation")
		require.Contains(t, output, "automation")
		require.Contains(t, output, "ReservationMade, ConfirmReservation")
		require.Contains(t, output, "Import External Booking")
		require.Contains(t, output, "translation")
		require.Contains(t, output, "ImportBooking, BookingImported")
	})

	t.Run("lists slices in model order grouped by context", func(t *testing.T) {
		path := writeTemp(t, "valid.emod", validEmod)

		output := captureStdout(t, func() {
			err := cli.RunSlices(path)
			require.NoError(t, err)
		})

		lines := strings.Split(output, "\n")
		var sliceLines []string
		for _, line := range lines {
			if strings.Contains(line, "Make Reservation") ||
				strings.Contains(line, "View Reservations") ||
				strings.Contains(line, "Auto Confirm Reservation") ||
				strings.Contains(line, "Import External Booking") {
				sliceLines = append(sliceLines, line)
			}
		}
		require.Len(t, sliceLines, 4)
		require.Contains(t, sliceLines[0], "Make Reservation")
		require.Contains(t, sliceLines[1], "View Reservations")
		require.Contains(t, sliceLines[2], "Auto Confirm Reservation")
		require.Contains(t, sliceLines[3], "Import External Booking")
	})

	t.Run("fails with descriptive message for parse errors", func(t *testing.T) {
		path := writeTemp(t, "invalid.emod", invalidEmod)

		err := cli.RunSlices(path)

		require.Error(t, err)
		require.Contains(t, err.Error(), path)
		require.Contains(t, err.Error(), ":1:")
		var lintErr *cli.LintError
		require.True(t, errors.As(err, &lintErr))
		require.Equal(t, 1, lintErr.ExitCode)
	})

	t.Run("fails for missing file argument", func(t *testing.T) {
		err := cli.RunSlices("")

		require.Error(t, err)
		require.Equal(t, "slices requires exactly one file argument", err.Error())
		var lintErr *cli.LintError
		require.True(t, errors.As(err, &lintErr))
		require.Equal(t, 1, lintErr.ExitCode)
	})

	t.Run("fails for nonexistent file", func(t *testing.T) {
		err := cli.RunSlices("/tmp/nonexistent-emod-slices-file-abc123.emod")

		require.Error(t, err)
		require.Contains(t, err.Error(), "/tmp/nonexistent-emod-slices-file-abc123.emod")
		var lintErr *cli.LintError
		require.True(t, errors.As(err, &lintErr))
		require.Equal(t, 1, lintErr.ExitCode)
	})

	t.Run("pattern detection prefers most specific match", func(t *testing.T) {
		input := `model "Test"
context "Orders" {
  aggregate "Order" {
    slice "Mixed Pattern" {
      command PlaceOrder {
        fields {
          orderId string required
        }
      }
      event OrderPlaced {
        fields {
          orderId string required
        }
      }
      view OrderView {
        fields {
          orderId string required
        }
      }
      automation AutoProcess {
        trigger OrderPlaced
        command ProcessOrder
      }
      translation ExtIntegration {
        external_system "External API"
        reads OrderView
        command PlaceOrder
        event OrderPlaced {
          fields {
            orderId string required
          }
        }
      }
      flow {
        command -> event: PlaceOrder -> OrderPlaced
      }
    }
  }
}
`
		path := writeTemp(t, "mixed.emod", input)

		output := captureStdout(t, func() {
			err := cli.RunSlices(path)
			require.NoError(t, err)
		})

		require.Contains(t, output, "Mixed Pattern")
		require.Contains(t, output, "translation")
	})
}

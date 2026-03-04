//go:build unit

package cli_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hpcsc/emod/internal/cli"
	"github.com/stretchr/testify/require"
)

const validEmod = `# Hotel Reservation System
model "Hotel Reservation"

actor "Guest"

context "Reservations" {
  aggregate "Reservation" {
    slice "Make Reservation" {
      trigger UI "Reservation Form" {
        actor Guest
        reads AvailableRoomsView
      }
      command MakeReservation {
        fields {
          guestId     string required
          roomType    string required
          checkIn     date   required
          checkOut    date   required
        }
      }
      event ReservationMade {
        fields {
          reservationId string required
          guestId       string required
          roomType      string required
          checkIn       date   required
          checkOut      date   required
          status        string required
        }
      }
      flow {
        command -> event: MakeReservation -> ReservationMade
      }
    }
    slice "View Reservations" {
      view ReservationsView {
        fields {
          reservationId string required
          guestId       string required
          status        string required
        }
        subscribes [ReservationMade]
      }
    }
    slice "Auto Confirm Reservation" {
      automation AutoConfirm {
        trigger ReservationMade
        command ConfirmReservation
      }
    }
    slice "Import External Booking" {
      translation BookingImport {
        external_system "Booking.com API"
        reads BookingWebhookView
        command ImportBooking
        event BookingImported {
          fields {
            bookingId string required
          }
        }
      }
    }
  }
}
`

const invalidEmod = `foobar {
}
`

func TestValidate(t *testing.T) {
	t.Run("returns no error for valid input", func(t *testing.T) {
		path := writeTemp(t, "valid.emod", validEmod)

		err := cli.RunValidate(path)

		require.NoError(t, err)
	})

	t.Run("returns error for invalid input", func(t *testing.T) {
		path := writeTemp(t, "invalid.emod", invalidEmod)

		err := cli.RunValidate(path)

		require.Error(t, err)
		require.Contains(t, err.Error(), path)
		require.Contains(t, err.Error(), ":1:")
	})

	t.Run("returns error for nonexistent file", func(t *testing.T) {
		err := cli.RunValidate("/tmp/nonexistent-emod-file-abc123.emod")

		require.Error(t, err)
	})

	t.Run("returns error when no file argument given", func(t *testing.T) {
		err := cli.RunValidate("")

		require.Error(t, err)
		require.Equal(t, "validate requires exactly one file argument", err.Error())
	})
}

func writeTemp(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	err := os.WriteFile(path, []byte(content), 0o644)
	require.NoError(t, err)
	return path
}

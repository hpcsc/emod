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
      command ConfirmReservation {
        fields {
          reservationId string required
        }
      }
      automation AutoConfirm {
        trigger ReservationMade
        command ConfirmReservation
      }
    }
    slice "Import External Booking" {
      command ImportBooking {
        fields {
          bookingRef string required
        }
      }
      translation BookingImport {
        external_system "Booking.com API"
        reads BookingWebhookView
        command ImportBooking
        event BookingImported {
          fields {
            bookingId   string required
            hotelName   string required
            bookingRef  string required
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

	t.Run("returns semantic error for automation targeting nonexistent context", func(t *testing.T) {
		input := `model "Test"
context "Orders" {
  aggregate "Order" {
    slice "Process Order" {
      automation OrderNotifier {
        trigger OrderPlaced
        command NotifyCustomer
        target context NonExistent
      }
    }
  }
}
`
		path := writeTemp(t, "bad_target.emod", input)

		err := cli.RunValidate(path)

		require.Error(t, err)
		require.Contains(t, err.Error(), "NonExistent")
		require.Contains(t, err.Error(), "does not exist")
	})

	t.Run("returns error for automation trigger referencing nonexistent event", func(t *testing.T) {
		input := `model "Test"
context "Orders" {
  aggregate "Order" {
    slice "Process Order" {
      automation OrderNotifier {
        trigger NonExistentEvent
        command NotifyCustomer
      }
    }
  }
}
`
		path := writeTemp(t, "bad_trigger.emod", input)

		err := cli.RunValidate(path)

		require.Error(t, err)
		require.Contains(t, err.Error(), "NonExistentEvent")
		require.Contains(t, err.Error(), "does not exist")
	})

	t.Run("returns no error for valid multi-context model", func(t *testing.T) {
		input := `model "Multi Context Test"

context "Orders" {
  aggregate "Order" {
    slice "Place Order" {
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
      flow {
        command -> event: PlaceOrder -> OrderPlaced
      }
    }
    slice "Notify On Order" {
      automation OrderNotifier {
        trigger OrderPlaced
        command SendNotification
        target context Notifications
      }
    }
  }
}
context "Notifications" {
  aggregate "Notification" {
    slice "Send Notification" {
      command SendNotification {
        fields {
          message string required
        }
      }
      event NotificationReceived {
        source external "Email Provider"
        fields {
          notificationId string required
          receivedAt     timestamp required
        }
      }
    }
  }
}
`
		path := writeTemp(t, "multi_context.emod", input)

		err := cli.RunValidate(path)

		require.NoError(t, err)
	})

	t.Run("returns no error for automation targeting existing context", func(t *testing.T) {
		input := `model "Test"
context "Orders" {
  aggregate "Order" {
    slice "Place Order" {
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
    }
    slice "Notify On Order" {
      automation OrderNotifier {
        trigger OrderPlaced
        command SendNotification
        target context Notifications
      }
    }
  }
}
context "Notifications" {
  aggregate "Notification" {
    slice "Send Notification" {
      command SendNotification {
        fields {
          message string required
        }
      }
      command SendEmail {
        fields {
          to string required
        }
      }
      event NotificationRequested {
        fields {
          notificationId string required
        }
      }
      automation Sender {
        trigger NotificationRequested
        command SendEmail
      }
    }
  }
}
`
		path := writeTemp(t, "valid_target.emod", input)

		err := cli.RunValidate(path)

		require.NoError(t, err)
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

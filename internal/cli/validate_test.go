//go:build unit

package cli_test

import (
	"encoding/json"
	"errors"
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
      flow {
        command -> event: ConfirmReservation -> ReservationMade
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
      flow {
        command -> event: ImportBooking -> BookingImported
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

		err := cli.RunValidate(path, "text")

		require.NoError(t, err)
	})

	t.Run("returns error for invalid input", func(t *testing.T) {
		path := writeTemp(t, "invalid.emod", invalidEmod)

		err := cli.RunValidate(path, "text")

		require.Error(t, err)
		require.Contains(t, err.Error(), path)
		require.Contains(t, err.Error(), ":1:")
	})

	t.Run("returns error for nonexistent file", func(t *testing.T) {
		err := cli.RunValidate("/tmp/nonexistent-emod-file-abc123.emod", "text")

		require.Error(t, err)
	})

	t.Run("returns error when no file argument given", func(t *testing.T) {
		err := cli.RunValidate("", "text")

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

		err := cli.RunValidate(path, "text")

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

		err := cli.RunValidate(path, "text")

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
          orderId     string required
          totalAmount string required
        }
      }
      event OrderPlaced {
        fields {
          orderId     string required
          totalAmount string required
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
      flow {
        command -> event: SendNotification -> NotificationReceived
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

		err := cli.RunValidate(path, "text")

		require.NoError(t, err)
	})

	t.Run("returns no error for automation targeting existing context", func(t *testing.T) {
		input := `model "Test"
context "Orders" {
  aggregate "Order" {
    slice "Place Order" {
      command PlaceOrder {
        fields {
          orderId     string required
          totalAmount string required
        }
      }
      event OrderPlaced {
        fields {
          orderId     string required
          totalAmount string required
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
      command SendEmail {
        fields {
          to string required
        }
      }
      flow {
        command -> event: SendNotification -> NotificationRequested
      }
      flow {
        command -> event: SendEmail -> NotificationRequested
      }
      event NotificationRequested {
        fields {
          notificationId string required
          message        string required
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

		err := cli.RunValidate(path, "text")

		require.NoError(t, err)
	})

	t.Run("returns error for model with only lint warnings", func(t *testing.T) {
		input := `model "Test"
context "Test" {
  aggregate "Test" {
    slice "Test" {
      command OrderPlaced {}
      event OrderUpdated {}
      view OrderList {}
      flow {
        command -> event: OrderPlaced -> OrderUpdated
      }
    }
  }
}
`
		path := writeTemp(t, "lint_only.emod", input)

		err := cli.RunValidate(path, "text")

		require.Error(t, err)
		require.Contains(t, err.Error(), "command-past-tense")
		require.Contains(t, err.Error(), "OrderPlaced")
		require.Contains(t, err.Error(), "state-obsession")
		require.Contains(t, err.Error(), "OrderUpdated")
		require.Contains(t, err.Error(), "view-naming")
		require.Contains(t, err.Error(), "OrderList")
	})

	t.Run("returns both lint warnings and validation errors", func(t *testing.T) {
		input := `model "Test"
context "Orders" {
  aggregate "Order" {
    slice "Process Order" {
      command OrderPlaced {}
      event OrderUpdated {}
      view OrderList {}
      automation OrderNotifier {
        trigger OrderPlaced
        command NotifyCustomer
        target context NonExistent
      }
      flow {
        command -> event: OrderPlaced -> OrderUpdated
      }
    }
  }
}
`
		path := writeTemp(t, "combined.emod", input)

		err := cli.RunValidate(path, "text")

		require.Error(t, err)
		// lint warnings
		require.Contains(t, err.Error(), "command-past-tense")
		require.Contains(t, err.Error(), "state-obsession")
		require.Contains(t, err.Error(), "view-naming")
		// validation errors
		require.Contains(t, err.Error(), "NonExistent")
		require.Contains(t, err.Error(), "does not exist")
	})

	t.Run("json format on clean file outputs empty array", func(t *testing.T) {
		path := writeTemp(t, "clean.emod", validEmod)

		output := captureStdout(t, func() {
			err := cli.RunValidate(path, "json")
			require.NoError(t, err)
		})

		require.Equal(t, "[]\n", output)
	})

	t.Run("json format on invalid input outputs structured diagnostics", func(t *testing.T) {
		path := writeTemp(t, "invalid.emod", invalidEmod)

		output := captureStdout(t, func() {
			err := cli.RunValidate(path, "json")
			var lintErr *cli.LintError
			if errors.As(err, &lintErr) {
				require.Equal(t, 2, lintErr.ExitCode)
				require.Equal(t, "", lintErr.Message)
			} else {
				require.Error(t, err)
			}
		})

		var entries []map[string]interface{}
		err := json.Unmarshal([]byte(output), &entries)
		require.NoError(t, err)
		require.Greater(t, len(entries), 0)
	})

	t.Run("json entries contain file, line, severity, and message fields", func(t *testing.T) {
		path := writeTemp(t, "invalid.emod", invalidEmod)

		output := captureStdout(t, func() {
			_ = cli.RunValidate(path, "json")
		})

		var entries []map[string]interface{}
		err := json.Unmarshal([]byte(output), &entries)
		require.NoError(t, err)
		require.Greater(t, len(entries), 0)

		entry := entries[0]
		require.Equal(t, path, entry["file"])
		require.NotEqual(t, 0, entry["line"])
		require.NotEmpty(t, entry["severity"])
		require.NotEmpty(t, entry["message"])
	})

	t.Run("json format on warning-only file outputs warning severity and exit code 1", func(t *testing.T) {
		input := `model "Test"
context "Orders" {
  aggregate "Order" {
    slice "Update Order" {
      command PlaceOrder {
        fields {
          orderId string required
          reason  string required
        }
      }
      event OrderUpdated {
        fields {
          orderId string required
          reason  string required
        }
      }
      flow {
        command -> event: PlaceOrder -> OrderUpdated
      }
    }
  }
}
`
		path := writeTemp(t, "warnings.emod", input)

		var output string
		output = captureStdout(t, func() {
			err := cli.RunValidate(path, "json")
			var lintErr *cli.LintError
			if errors.As(err, &lintErr) {
				require.Equal(t, 1, lintErr.ExitCode)
				require.Equal(t, "", lintErr.Message)
			} else {
				require.Error(t, err)
			}
		})

		var entries []map[string]interface{}
		err := json.Unmarshal([]byte(output), &entries)
		require.NoError(t, err)
		require.Len(t, entries, 1)
		require.Equal(t, "warning", entries[0]["severity"])
		require.Equal(t, "state-obsession", entries[0]["rule"])
	})

	t.Run("json format on file with errors outputs error severity and exit code 2", func(t *testing.T) {
		input := `model "Test"
context "Orders" {
  aggregate "Order" {
    slice "Events" {
      command PlaceOrder {
        fields {
          orderId string required
        }
      }
      event SingleIdEvent {
        fields {
          orderId string required
        }
      }
      flow {
        command -> event: PlaceOrder -> SingleIdEvent
      }
    }
  }
}
`
		path := writeTemp(t, "errors.emod", input)

		var output string
		output = captureStdout(t, func() {
			err := cli.RunValidate(path, "json")
			var lintErr *cli.LintError
			if errors.As(err, &lintErr) {
				require.Equal(t, 2, lintErr.ExitCode)
				require.Equal(t, "", lintErr.Message)
			} else {
				require.Error(t, err)
			}
		})

		var entries []map[string]interface{}
		err := json.Unmarshal([]byte(output), &entries)
		require.NoError(t, err)
		require.Len(t, entries, 1)
		require.Equal(t, "error", entries[0]["severity"])
		require.Equal(t, "clickbait-event", entries[0]["rule"])
	})

	t.Run("json format reports all file and line fields", func(t *testing.T) {
		path := writeTemp(t, "invalid.emod", invalidEmod)

		output := captureStdout(t, func() {
			_ = cli.RunValidate(path, "json")
		})

		var entries []map[string]interface{}
		err := json.Unmarshal([]byte(output), &entries)
		require.NoError(t, err)
		require.Greater(t, len(entries), 0)

		entry := entries[0]
		require.Equal(t, path, entry["file"])
		require.NotEqual(t, 0, entry["line"])
		require.NotEmpty(t, entry["message"])
	})

	t.Run("unsupported format returns error", func(t *testing.T) {
		path := writeTemp(t, "clean.emod", validEmod)

		err := cli.RunValidate(path, "unknown")

		require.Error(t, err)
		require.Contains(t, err.Error(), "unsupported format")
		var lintErr *cli.LintError
		if errors.As(err, &lintErr) {
			require.Equal(t, 1, lintErr.ExitCode)
		}
	})

	t.Run("text format is the default and unchanged for existing behaviors", func(t *testing.T) {
		path := writeTemp(t, "invalid.emod", invalidEmod)

		err := cli.RunValidate(path, "text")

		require.Error(t, err)
		require.Contains(t, err.Error(), path)
		require.Contains(t, err.Error(), ":1:")
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

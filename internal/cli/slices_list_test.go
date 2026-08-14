//go:build unit

package cli_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/hpcsc/emod/internal/cli"
	"github.com/stretchr/testify/require"
)

// automationCadenceEmod holds, under one aggregate so the listing reaches them
// all, a slice whose automation runs on a cadence, one whose automation names
// the event it activates on, and one whose automation names a delay beside that
// event — the three shapes an automation has of stating when it runs, so no row
// is checked in isolation.
const automationCadenceEmod = `model "Library Lending"

context "Lending" {
  aggregate "Loan" {
    slice "Sweep Overdue Loans" {
      command RecallCopy {
        fields {
          loanId string required
        }
      }
      automation SweepOverdueLoans {
        every "15m"
        command RecallCopy
      }
    }
    slice "Chase Overdue Copy" {
      command RemindMember {
        fields {
          loanId string required
        }
      }
      event CopyBorrowed {
        fields {
          loanId string required
        }
      }
      automation RemindOnDueDate {
        on CopyBorrowed
        command RemindMember
      }
    }
    slice "Release Expired Hold" {
      command ReleaseHold {
        fields {
          holdId string required
        }
      }
      event RoomHeld {
        fields {
          holdId string required
        }
      }
      automation ExpiredHoldReleaser {
        on RoomHeld after "24h"
        command ReleaseHold
      }
    }
  }
}
`

func TestSlicesList(t *testing.T) {
	t.Run("text format", func(t *testing.T) {
		t.Run("prints table for valid file with all pattern types", func(t *testing.T) {
			path := writeTemp(t, "valid.emod", validEmod)

			output := captureStdout(t, func() {
				err := cli.RunSlicesList(path, "text")
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
				err := cli.RunSlicesList(path, "text")
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

		t.Run("lists slices a DCB context declares directly", func(t *testing.T) {
			input := `model "Orders"

context "Fulfillment" mode dcb {
  slice "Place Order" {
    trigger "Order Form" {
      actor Customer
    }
    command PlaceOrder {
      fields {
        customerId string required
      }
    }
    event OrderPlaced {
      tags {
        entity: customerId
      }
      fields {
        orderId    string required
        customerId string required
      }
    }
    flow {
      command -> event: PlaceOrder -> OrderPlaced
    }
  }
}
`
			path := writeTemp(t, "dcb.emod", input)

			output := captureStdout(t, func() {
				err := cli.RunSlicesList(path, "text")
				require.NoError(t, err)
			})

			require.Contains(t, output, "Place Order")
			require.Contains(t, output, "Fulfillment")
			require.NotContains(t, output, "No slices found.")
		})

		t.Run("names the cadence of a scheduled automation, the event of an event-activated one and the delay of a delayed one", func(t *testing.T) {
			path := writeTemp(t, "cadence.emod", automationCadenceEmod)

			output := captureStdout(t, func() {
				err := cli.RunSlicesList(path, "text")
				require.NoError(t, err)
			})

			require.Contains(t, output, `every "15m", RecallCopy`)
			require.Contains(t, output, "CopyBorrowed, RemindMember")
			require.Contains(t, output, `RoomHeld after "24h", ReleaseHold`)
			for _, line := range strings.Split(output, "\n") {
				require.NotContains(t, line, "  , ", "a row must not open its key elements with a comma")
			}
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
        on OrderPlaced
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
				err := cli.RunSlicesList(path, "text")
				require.NoError(t, err)
			})

			require.Contains(t, output, "Mixed Pattern")
			require.Contains(t, output, "translation")
		})
	})

	t.Run("json format", func(t *testing.T) {
		t.Run("outputs JSON array with name, pattern, context, and keyElements", func(t *testing.T) {
			path := writeTemp(t, "valid.emod", validEmod)

			output := captureStdout(t, func() {
				err := cli.RunSlicesList(path, "json")
				require.NoError(t, err)
			})

			require.True(t, json.Valid([]byte(output)))

			var entries []map[string]interface{}
			err := json.Unmarshal([]byte(output), &entries)
			require.NoError(t, err)
			require.Len(t, entries, 4)

			require.Equal(t, "Make Reservation", entries[0]["name"])
			require.Equal(t, "command", entries[0]["pattern"])
			require.Equal(t, "Reservations", entries[0]["context"])
			require.Equal(t, "MakeReservation, ReservationMade", entries[0]["keyElements"])

			require.Equal(t, "View Reservations", entries[1]["name"])
			require.Equal(t, "view", entries[1]["pattern"])
			require.Equal(t, "Reservations", entries[1]["context"])
			require.Equal(t, "ReservationsView", entries[1]["keyElements"])

			require.Equal(t, "Auto Confirm Reservation", entries[2]["name"])
			require.Equal(t, "automation", entries[2]["pattern"])
			require.Equal(t, "Reservations", entries[2]["context"])
			require.Equal(t, "ReservationMade, ConfirmReservation", entries[2]["keyElements"])

			require.Equal(t, "Import External Booking", entries[3]["name"])
			require.Equal(t, "translation", entries[3]["pattern"])
			require.Equal(t, "Reservations", entries[3]["context"])
			require.Equal(t, "ImportBooking, BookingImported", entries[3]["keyElements"])
		})

		t.Run("carries the same key elements as the text listing for a file stating all three timing shapes", func(t *testing.T) {
			path := writeTemp(t, "cadence.emod", automationCadenceEmod)

			output := captureStdout(t, func() {
				err := cli.RunSlicesList(path, "json")
				require.NoError(t, err)
			})

			var entries []map[string]interface{}
			require.NoError(t, json.Unmarshal([]byte(output), &entries))

			keyElements := make([]string, 0, len(entries))
			for _, entry := range entries {
				keyElements = append(keyElements, entry["keyElements"].(string))
			}

			require.Equal(t, []string{
				`every "15m", RecallCopy`,
				"CopyBorrowed, RemindMember",
				`RoomHeld after "24h", ReleaseHold`,
			}, keyElements)
		})

		t.Run("empty model outputs empty JSON array", func(t *testing.T) {
			input := `model "Empty"`
			path := writeTemp(t, "empty.emod", input)

			output := captureStdout(t, func() {
				err := cli.RunSlicesList(path, "json")
				require.NoError(t, err)
			})

			require.Equal(t, "[]\n", output)
		})
	})

	t.Run("fails with descriptive message for parse errors", func(t *testing.T) {
		path := writeTemp(t, "invalid.emod", invalidEmod)

		err := cli.RunSlicesList(path, "text")

		require.Error(t, err)
		require.Contains(t, err.Error(), path)
		require.Contains(t, err.Error(), ":1:")
		var lintErr *cli.LintError
		require.True(t, errors.As(err, &lintErr))
		require.Equal(t, 1, lintErr.ExitCode)
	})

	t.Run("fails for missing file argument", func(t *testing.T) {
		err := cli.RunSlicesList("", "text")

		require.ErrorIs(t, err, cli.ErrMissingFileArgument)
		var lintErr *cli.LintError
		require.True(t, errors.As(err, &lintErr))
		require.Equal(t, 1, lintErr.ExitCode)
	})

	t.Run("fails for nonexistent file", func(t *testing.T) {
		err := cli.RunSlicesList("/tmp/nonexistent-emod-slices-file-abc123.emod", "text")

		require.Error(t, err)
		require.Contains(t, err.Error(), "/tmp/nonexistent-emod-slices-file-abc123.emod")
		var lintErr *cli.LintError
		require.True(t, errors.As(err, &lintErr))
		require.Equal(t, 1, lintErr.ExitCode)
	})

	t.Run("unsupported format returns error", func(t *testing.T) {
		path := writeTemp(t, "valid.emod", validEmod)

		err := cli.RunSlicesList(path, "xml")

		require.ErrorIs(t, err, cli.ErrUnsupportedFormat)
		require.Contains(t, err.Error(), "text")
		require.Contains(t, err.Error(), "json")
		var lintErr *cli.LintError
		require.True(t, errors.As(err, &lintErr))
		require.Equal(t, 1, lintErr.ExitCode)
	})

	t.Run("fails with descriptive message for parse errors with json format", func(t *testing.T) {
		path := writeTemp(t, "invalid.emod", invalidEmod)

		err := cli.RunSlicesList(path, "json")

		require.Error(t, err)
		require.Contains(t, err.Error(), path)
		require.Contains(t, err.Error(), ":1:")
		var lintErr *cli.LintError
		require.True(t, errors.As(err, &lintErr))
		require.Equal(t, 1, lintErr.ExitCode)
	})
}

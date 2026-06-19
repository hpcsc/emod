//go:build unit

package oracle_test

import (
	"strings"
	"testing"

	"github.com/hpcsc/emod/internal/diagnostic"
	"github.com/hpcsc/emod/internal/oracle"
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
  }
}
`

const invalidEmod = `foobar {
}
`

func TestCheck(t *testing.T) {
	t.Run("clean input", func(t *testing.T) {
		t.Run("returns an empty diagnostic list for a fully valid model", func(t *testing.T) {
			diagnostics := oracle.Check(validEmod, "valid.emod")

			require.Empty(t, diagnostics)
		})
	})

	t.Run("unparseable input", func(t *testing.T) {
		t.Run("returns entries carrying position, severity, and message", func(t *testing.T) {
			diagnostics := oracle.Check(invalidEmod, "invalid.emod")

			require.NotEmpty(t, diagnostics)
			first := diagnostics[0]
			require.Equal(t, 1, first.Line)
			require.NotEmpty(t, first.Message)
			require.Equal(t, diagnostic.Error, first.Severity)
		})

		t.Run("propagates the passed-in filename into every entry", func(t *testing.T) {
			const filename = "my-model.emod"

			diagnostics := oracle.Check(invalidEmod, filename)

			require.NotEmpty(t, diagnostics)
			for _, d := range diagnostics {
				require.Equal(t, filename, d.Filename)
			}
		})
	})

	t.Run("validator faults", func(t *testing.T) {
		t.Run("surface when a parseable model targets a nonexistent context", func(t *testing.T) {
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

			diagnostics := oracle.Check(input, "bad_target.emod")

			var found *diagnostic.Entry
			for _, d := range diagnostics {
				if strings.Contains(d.Message, "NonExistent") {
					found = d
					break
				}
			}
			require.NotNil(t, found, "expected a validator diagnostic mentioning the missing context")
			require.Contains(t, found.Message, "NonExistent")
			require.Contains(t, found.Message, "does not exist")
		})
	})

	t.Run("linter faults", func(t *testing.T) {
		t.Run("surface on an otherwise valid model", func(t *testing.T) {
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

			diagnostics := oracle.Check(input, "lint_only.emod")

			require.True(t, hasRule(diagnostics, "command-past-tense"), "expected command-past-tense rule")
			require.True(t, hasRule(diagnostics, "state-obsession"), "expected state-obsession rule")
			require.True(t, hasRule(diagnostics, "view-naming"), "expected view-naming rule")
		})
	})

	t.Run("combined faults", func(t *testing.T) {
		t.Run("surface both a validator error and linter warnings together", func(t *testing.T) {
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

			diagnostics := oracle.Check(input, "combined.emod")

			var found *diagnostic.Entry
			for _, d := range diagnostics {
				if strings.Contains(d.Message, "NonExistent") {
					found = d
					break
				}
			}
			require.NotNil(t, found, "expected the validator diagnostic for the missing context")
			require.Contains(t, found.Message, "NonExistent")
			require.Contains(t, found.Message, "does not exist")
			require.True(t,
				hasRule(diagnostics, "command-past-tense") ||
					hasRule(diagnostics, "state-obsession") ||
					hasRule(diagnostics, "view-naming"),
				"expected at least one linter diagnostic")
		})
	})

	t.Run("severity", func(t *testing.T) {
		t.Run("reports a single-id event at error severity", func(t *testing.T) {
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

			diagnostics := oracle.Check(input, "errors.emod")

			var found *diagnostic.Entry
			for _, d := range diagnostics {
				if d.RuleName == "clickbait-event" {
					found = d
					break
				}
			}
			require.NotNil(t, found, "expected a clickbait-event diagnostic")
			require.Equal(t, diagnostic.Error, found.Severity)
		})
	})
}

func hasRule(diagnostics []*diagnostic.Entry, ruleName string) bool {
	for _, d := range diagnostics {
		if d.RuleName == ruleName {
			return true
		}
	}
	return false
}

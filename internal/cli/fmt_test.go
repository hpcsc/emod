//go:build unit

package cli_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hpcsc/emod/internal/cli"
	"github.com/stretchr/testify/require"
)

const emodWithoutVersionHeader = `model "Hotel Reservation"

actor "Guest"

context "Reservations" {
  aggregate "Reservation" {
    slice "Make Reservation" {
      command MakeReservation {
        fields {
          guestId  string required
          roomType string required
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
  }
}
`

const formattedEmod = "emod 1\n" + emodWithoutVersionHeader

const describedFormattedEmod = `emod 1
model "Hotel Reservation" {
  description "How the hotel takes and confirms room bookings"
}

actor "Guest" {
  description "A person booking a room"
}

context "Reservations" {
  description "Everything the hotel knows about a stay before the guest arrives"
  aggregate "Reservation" {
    description "One guest holding one room over one date range"
    slice "Make Reservation" {
      description "A guest books a room from the public site"
      command MakeReservation {
        description "Ask the hotel to hold a room for a date range"
        fields {
          guestId  string required
          roomType string required
        }
      }

      event ReservationMade {
        description "A room is held for a guest"
        fields {
          reservationId string required
          guestId       string required
          roomType      string required
        }
      }

      flow {
        command -> event: MakeReservation -> ReservationMade
      }
    }
  }
}
`

const unformattedEmod = `model "Hotel Reservation"
actor "Guest"
context "Reservations" {
  aggregate "Reservation" {
    slice "Make Reservation" {
      command MakeReservation {
        fields {
          guestId string required
          roomType string required
        }
      }
      event ReservationMade {
        fields {
          reservationId string required
          guestId string required
          roomType string required
          checkIn date required
          checkOut date required
          status string required
        }
      }
      flow {
        command -> event: MakeReservation -> ReservationMade
      }
    }
  }
}
`

const modifierlessFormattedEmod = `emod 1
model "Hotel Reservation"

context "Reservations" {
  aggregate "Reservation" {
    slice "Make Reservation" {
      command MakeReservation {
        fields {
          roomType string
          guestId  string required
        }
      }
    }
  }
}
`

const unparsableEmod = `foobar {
}
`

func TestFmt(t *testing.T) {
	t.Run("returns error when no file argument given", func(t *testing.T) {
		err := cli.RunFmt("", false)

		require.ErrorIs(t, err, cli.ErrMissingFileArgument)
	})

	t.Run("returns error naming the file when it does not exist", func(t *testing.T) {
		missing := filepath.Join(t.TempDir(), "nonexistent.emod")

		err := cli.RunFmt(missing, false)

		require.Error(t, err)
		require.Contains(t, err.Error(), missing)
	})

	t.Run("returns error and does not modify file with parse errors", func(t *testing.T) {
		path := writeTemp(t, "broken.emod", unparsableEmod)

		err := cli.RunFmt(path, false)

		require.Error(t, err)
		require.Contains(t, err.Error(), path)
		require.Contains(t, err.Error(), ":1:")
		require.Equal(t, unparsableEmod, readFile(t, path), "file should not be modified when there are parse errors")
	})

	t.Run("returns error and does not modify a file declaring an unsupported version", func(t *testing.T) {
		source := "emod 2\n" + emodWithoutVersionHeader
		path := writeTemp(t, "unsupported.emod", source)

		err := cli.RunFmt(path, false)

		require.Error(t, err)
		require.Contains(t, err.Error(), path)
		require.Contains(t, err.Error(), ":1:")
		require.Equal(t, source, readFile(t, path))
	})

	t.Run("rewrites file in-place with formatted content", func(t *testing.T) {
		path := writeTemp(t, "messy.emod", unformattedEmod)

		err := cli.RunFmt(path, false)

		require.NoError(t, err)
		require.Equal(t, formattedEmod, readFile(t, path))
	})

	t.Run("produces no changes when file is already formatted", func(t *testing.T) {
		path := writeTemp(t, "clean.emod", formattedEmod)
		info, statErr := os.Stat(path)
		require.NoError(t, statErr)
		modTimeBefore := info.ModTime()

		err := cli.RunFmt(path, false)

		require.NoError(t, err)
		require.Equal(t, formattedEmod, readFile(t, path))

		infoAfter, statErr := os.Stat(path)
		require.NoError(t, statErr)
		require.Equal(t, modTimeBefore, infoAfter.ModTime(), "file should not be rewritten when already formatted")
	})

	t.Run("leaves a formatted file whose field has no modifier untouched on every run", func(t *testing.T) {
		path := writeTemp(t, "modifierless.emod", modifierlessFormattedEmod)

		require.NoError(t, cli.RunFmt(path, false))
		require.Equal(t, modifierlessFormattedEmod, readFile(t, path))

		require.NoError(t, cli.RunFmt(path, false))
		require.Equal(t, modifierlessFormattedEmod, readFile(t, path), "a second run should not change the file")

		require.NoError(t, cli.RunFmt(path, true), "check mode should report nothing to change")
		require.Equal(t, modifierlessFormattedEmod, readFile(t, path))
	})

	t.Run("check mode returns nil when file is already formatted", func(t *testing.T) {
		path := writeTemp(t, "clean.emod", formattedEmod)

		err := cli.RunFmt(path, true)

		require.NoError(t, err)
		require.Equal(t, formattedEmod, readFile(t, path), "check mode should not modify the file")
	})

	t.Run("check mode returns nil when a file using descriptions is already formatted", func(t *testing.T) {
		path := writeTemp(t, "described.emod", describedFormattedEmod)

		err := cli.RunFmt(path, true)

		require.NoError(t, err)
		require.Equal(t, describedFormattedEmod, readFile(t, path), "check mode should not modify the file")
	})

	t.Run("check mode returns error when a file is canonical apart from a missing version header", func(t *testing.T) {
		path := writeTemp(t, "headerless.emod", emodWithoutVersionHeader)

		err := cli.RunFmt(path, true)

		require.Error(t, err)
		require.Contains(t, err.Error(), path)
		require.Equal(t, emodWithoutVersionHeader, readFile(t, path), "check mode should not modify the file")
	})

	t.Run("check mode returns error when file needs formatting", func(t *testing.T) {
		path := writeTemp(t, "messy.emod", unformattedEmod)

		err := cli.RunFmt(path, true)

		require.Error(t, err)
		require.Contains(t, err.Error(), path)
		require.Equal(t, unformattedEmod, readFile(t, path), "check mode should not modify the file")
	})
}

func readFile(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	require.NoError(t, err)

	return string(content)
}

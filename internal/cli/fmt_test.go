//go:build unit

package cli_test

import (
	"os"
	"testing"

	"github.com/hpcsc/emod/internal/cli"
	"github.com/stretchr/testify/require"
)

const formattedEmod = `model "Hotel Reservation"

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

const unparsableEmod = `foobar {
}
`

func TestFmt(t *testing.T) {
	t.Run("returns error when no file argument given", func(t *testing.T) {
		err := cli.RunFmt("", false)

		require.Error(t, err)
		require.Equal(t, "fmt requires exactly one file argument", err.Error())
	})

	t.Run("returns error for nonexistent file", func(t *testing.T) {
		dir := t.TempDir()
		err := cli.RunFmt(dir+"/nonexistent.emod", false)

		require.Error(t, err)
	})

	t.Run("returns error and does not modify file with parse errors", func(t *testing.T) {
		path := writeTemp(t, "broken.emod", unparsableEmod)
		originalBytes, readErr := os.ReadFile(path)
		require.NoError(t, readErr)

		err := cli.RunFmt(path, false)

		require.Error(t, err)
		require.Contains(t, err.Error(), path)
		require.Contains(t, err.Error(), ":1:")

		afterBytes, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		require.Equal(t, originalBytes, afterBytes, "file should not be modified when there are parse errors")
	})

	t.Run("rewrites file in-place with formatted content", func(t *testing.T) {
		path := writeTemp(t, "messy.emod", unformattedEmod)

		err := cli.RunFmt(path, false)

		require.NoError(t, err)
		afterBytes, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		require.Equal(t, formattedEmod, string(afterBytes))
	})

	t.Run("produces no changes when file is already formatted", func(t *testing.T) {
		path := writeTemp(t, "clean.emod", formattedEmod)
		info, statErr := os.Stat(path)
		require.NoError(t, statErr)
		modTimeBefore := info.ModTime()

		err := cli.RunFmt(path, false)

		require.NoError(t, err)
		afterBytes, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		require.Equal(t, formattedEmod, string(afterBytes))

		infoAfter, statErr := os.Stat(path)
		require.NoError(t, statErr)
		require.Equal(t, modTimeBefore, infoAfter.ModTime(), "file should not be rewritten when already formatted")
	})

	t.Run("check mode returns nil when file is already formatted", func(t *testing.T) {
		path := writeTemp(t, "clean.emod", formattedEmod)

		err := cli.RunFmt(path, true)

		require.NoError(t, err)
		afterBytes, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		require.Equal(t, formattedEmod, string(afterBytes), "check mode should not modify the file")
	})

	t.Run("check mode returns error when file needs formatting", func(t *testing.T) {
		path := writeTemp(t, "messy.emod", unformattedEmod)

		err := cli.RunFmt(path, true)

		require.Error(t, err)
		require.Contains(t, err.Error(), path)

		afterBytes, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		require.Equal(t, unformattedEmod, string(afterBytes), "check mode should not modify the file")
	})
}

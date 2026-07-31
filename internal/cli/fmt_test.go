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

const keywordFieldFormattedEmod = `emod 1
# Saved searches over a catalogue of emod models
model "Model Search Catalog"

actor "Analyst"

context "Discovery" {
  aggregate "Saved Search" {
    slice "Define Saved Search" {
      trigger UI "Search Builder" {
        actor Analyst
        reads SavedSearchesView
      }

      command DefineSavedSearch {
        fields {
          model       string required
          source      string required
          where       string required
          and         string
          not         string
          fields      string required
          description string optional
        }
      }

      event SavedSearchDefined {
        fields {
          searchId    string required
          model       string required
          source      string required
          where       string required
          events      string required
          tag         string required
          emod        string
          description string required
          definedAt   date   required
        }
      }

      flow {
        command -> event: DefineSavedSearch -> SavedSearchDefined
      }
    }

    slice "Browse Saved Searches" {
      view SavedSearchesView {
        fields {
          searchId    string required
          description string required
          tag         string required
          model       string
          where       string required
          matches     int    required
        }
        subscribes [SavedSearchDefined]
      }
    }

    slice "Auto Share Saved Search" {
      command ShareSavedSearch {
        fields {
          searchId string required
          tag      string required
        }
      }

      automation AutoShare {
        on SavedSearchDefined
        command ShareSavedSearch
      }

      flow {
        command -> event: ShareSavedSearch -> SavedSearchDefined
      }
    }

    slice "Import Vendor Search" {
      command ImportVendorSearch {
        fields {
          source string required
        }
      }

      translation VendorSearchImport {
        external_system "Metabase API"
        reads VendorSearchWebhookView
        command ImportVendorSearch
        event VendorSearchImported {
          fields {
            vendorSearchId string required
            source         string required
            emod           string required
            where          string required
            tag            string
            model          string required
          }
        }
      }

      flow {
        command -> event: ImportVendorSearch -> VendorSearchImported
      }
    }
  }
}
`

const specFormattedEmod = `emod 1
# Lending a library's copies and seating its readers, with the scenarios each slice must satisfy
model "Library Lending"

actor "Member"

context "Lending" {
  aggregate "Loan" {
    invariant OneCopyPerLoan "A loan covers exactly one copy of one title"
    slice "Borrow Copy" {
      trigger UI "Lending Desk" {
        actor Member
        reads AvailableCopiesView
      }

      command BorrowCopy {
        fields {
          memberId string required
          copyId   string required
          dueOn    date   required
        }
      }

      event CopyBorrowed {
        fields {
          loanId   string required
          memberId string required
          copyId   string required
          dueOn    date   required
        }
      }

      flow {
        command -> event: BorrowCopy -> CopyBorrowed
      }

      spec "borrows a copy no one holds" {
        when BorrowCopy
        then [CopyBorrowed]
      }

      spec "borrows a copy the member before returned" {
        given [CopyBorrowed, CopyReturned]
        when BorrowCopy
        then [CopyBorrowed]
      }

      spec "refuses a copy already on loan" {
        given [CopyBorrowed]
        when BorrowCopy
        then rejected OneCopyPerLoan
      }
    }

    slice "Return Copy" {
      command ReturnCopy {
        fields {
          loanId string required
          copyId string required
        }
      }

      event CopyReturned {
        fields {
          loanId     string    required
          copyId     string    required
          returnedAt timestamp required
        }
      }

      flow {
        command -> event: ReturnCopy -> CopyReturned
      }

      spec "returns a copy the member holds" {
        given [CopyBorrowed]
        when ReturnCopy
        then [CopyReturned]
      }
    }

    slice "Review Member Loans" {
      view MemberLoansView {
        fields {
          loanId   string required
          memberId string required
          dueOn    date   required
        }
        subscribes [CopyBorrowed]
      }
    }
  }
}

context "Reading Room" mode dcb {
  invariant OneReaderPerDesk "A desk seats at most one reader at any moment"
  slice "Claim Desk" {
    command ClaimDesk {
      fields {
        memberId string required
        deskId   string required
      }
    }

    event DeskClaimed {
      tags {
        desk  : deskId
        reader: memberId
      }
      fields {
        sessionId string    required
        deskId    string    required
        memberId  string    required
        claimedAt timestamp required
      }
    }

    flow {
      command -> event: ClaimDesk -> DeskClaimed
    }

    spec "seats a reader at a free desk" {
      when ClaimDesk
      then [DeskClaimed]
    }

    spec "refuses a desk another reader is seated at" {
      given [DeskClaimed]
      when ClaimDesk
      then rejected OneReaderPerDesk
    }
  }

  slice "Release Desk" {
    command ReleaseDesk {
      decides_on {
        events [DeskClaimed]
        where tag(desk = deskId) and tag(reader = memberId)
      }
      fields {
        sessionId string required
      }
    }

    event DeskReleased {
      tags {
        desk  : deskId
        reader: memberId
      }
      fields {
        sessionId  string    required
        deskId     string    required
        memberId   string    required
        releasedAt timestamp required
      }
    }

    flow {
      command -> event: ReleaseDesk -> DeskReleased
    }

    spec "frees the desk its reader is seated at" {
      given [DeskClaimed]
      when ReleaseDesk
      then [DeskReleased]
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

		requireFmtSettlesOn(t, path, modifierlessFormattedEmod)
	})

	t.Run("keeps every keyword-named field on its own line and settles after one run", func(t *testing.T) {
		path := writeTemp(t, "keyword-fields.emod", keywordFieldEmod)

		requireFmtSettlesOn(t, path, keywordFieldFormattedEmod)
	})

	t.Run("keeps every declared invariant and settles after one run", func(t *testing.T) {
		path := writeTemp(t, "library-lending.emod", invariantEmod)

		require.NoError(t, cli.RunFmt(path, false))

		formatted := readFile(t, path)
		for _, declaration := range []string{
			`invariant OneCopyPerLoan "A loan covers exactly one copy of one title"`,
			`invariant FiveCopiesPerMember "A member holds at most five copies at one time"`,
			`invariant OneReaderPerDesk "A desk seats at most one reader at any moment"`,
			`invariant OneDeskPerReader "A reader holds at most one desk for the length of a session"`,
			`invariant DeskFreeAtClosing "No desk stays claimed past the closing hour"`,
		} {
			require.Contains(t, formatted, declaration)
		}

		requireFmtSettlesOn(t, path, formatted)
	})

	t.Run("keeps every declared spec and settles after one run", func(t *testing.T) {
		path := writeTemp(t, "specs.emod", specEmod)

		requireFmtSettlesOn(t, path, specFormattedEmod)
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

func requireFmtSettlesOn(t *testing.T, path, formatted string) {
	t.Helper()

	require.NoError(t, cli.RunFmt(path, false))
	require.Equal(t, formatted, readFile(t, path))

	require.NoError(t, cli.RunFmt(path, false))
	require.Equal(t, formatted, readFile(t, path), "a second run should not change the file")

	require.NoError(t, cli.RunFmt(path, true), "check mode should report nothing to change")
	require.Equal(t, formatted, readFile(t, path))
}

func readFile(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	require.NoError(t, err)

	return string(content)
}

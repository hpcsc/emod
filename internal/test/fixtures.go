package test

import (
	"fmt"

	"github.com/hpcsc/emod/internal/ast"
)

// HotelReservation exercises all four slice patterns — command, view,
// automation and translation — and is valid input for every stage of the
// pipeline. Packages that need "a realistic model" share this one so a change
// to the language is reflected in a single place.
const HotelReservation = `# Hotel Reservation System
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

// DescribedHotelReservation mirrors HotelReservation and adds a description to
// every construct that accepts one, so packages that carry descriptions through
// the pipeline share one model.
const DescribedHotelReservation = `# Hotel Reservation System
model "Hotel Reservation" {
  description "How the hotel takes, confirms and imports room bookings"
}

actor "Guest" {
  description "A person booking a room, not necessarily the one staying in it"
}

context "Reservations" {
  description "Everything the hotel knows about a stay before the guest arrives"
  aggregate "Reservation" {
    description "One guest holding one room over one date range"
    slice "Make Reservation" {
      description "A guest books a room from the public site"
      trigger UI "Reservation Form" {
        description "The booking form on the public site"
        actor Guest
        reads AvailableRoomsView
      }
      command MakeReservation {
        description "Ask the hotel to hold a room for a date range, 10% deposit taken up front"
        fields {
          guestId     string required
          roomType    string required
          checkIn     date   required
          checkOut    date   required
        }
      }
      event ReservationMade {
        description "A room is held for a guest"
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
      description "A guest reviews the stays they have booked"
      view ReservationsView {
        description "Every reservation with the stage it has reached"
        fields {
          reservationId string required
          guestId       string required
          status        string required
        }
        subscribes [ReservationMade]
      }
    }
    slice "Auto Confirm Reservation" {
      description "The hotel confirms a held room without a clerk touching it"
      command ConfirmReservation {
        description "Turn a held room into a confirmed stay"
        fields {
          reservationId string required
        }
      }
      flow {
        command -> event: ConfirmReservation -> ReservationMade
      }
      automation AutoConfirm {
        description "Confirms every reservation the moment it is made"
        trigger ReservationMade
        command ConfirmReservation
      }
    }
    slice "Import External Booking" {
      description "A booking made on a partner site becomes a reservation here"
      command ImportBooking {
        description "Record a booking taken by a partner site"
        fields {
          bookingRef string required
        }
      }
      flow {
        command -> event: ImportBooking -> BookingImported
      }
      translation BookingImport {
        description "Restates a partner webhook in the hotel's own language"
        external_system "Booking.com API"
        reads BookingWebhookView
        command ImportBooking
        event BookingImported {
          description "A partner site reported a booking"
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

// KeywordFieldSearchCatalog names its fields after DSL keywords, so packages
// that must keep keywords usable as identifiers share one model. Fields
// carrying no modifier sit mid-block ahead of a further field on purpose: give
// every field a modifier and nothing here would catch the two running together
// onto one line.
const KeywordFieldSearchCatalog = `# Saved searches over a catalogue of emod models
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
      flow {
        command -> event: ShareSavedSearch -> SavedSearchDefined
      }
      automation AutoShare {
        trigger SavedSearchDefined
        command ShareSavedSearch
      }
    }
    slice "Import Vendor Search" {
      command ImportVendorSearch {
        fields {
          source string required
        }
      }
      flow {
        command -> event: ImportVendorSearch -> VendorSearchImported
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
    }
  }
}
`

// InvariantLibraryLending declares invariants in both homes the language gives
// them — on an aggregate, and directly on a DCB-mode context, which has no
// aggregate to hold them — so packages that carry invariants through the
// pipeline share one model. Each block puts an invariant ahead of a later
// entry on purpose: write them all last and nothing here would catch a
// declaration running on into what follows it.
const InvariantLibraryLending = `# Lending a library's copies, and seating its readers
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
    }
    invariant FiveCopiesPerMember "A member holds at most five copies at one time"
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
  invariant OneDeskPerReader "A reader holds at most one desk for the length of a session"
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
  }
  invariant DeskFreeAtClosing "No desk stays claimed past the closing hour"
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
  }
}
`

// SpecLibraryLending states the scenarios its slices must satisfy in both homes
// a slice has — nested in an aggregate, and directly on a DCB-mode context —
// and with both outcomes a spec's then accepts, so packages that carry specs
// through the pipeline share one model. A spec block sits mid-block ahead of a
// further entry on purpose, and one spec omits its given history entirely:
// write them all last and nothing here would catch a block running on into what
// follows it.
const SpecLibraryLending = `# Lending a library's copies and seating its readers, with the scenarios each slice must satisfy
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
      spec "borrows a copy no one holds" {
        when BorrowCopy
        then [CopyBorrowed]
      }
      spec "borrows a copy the member before returned" {
        given [CopyBorrowed, CopyReturned]
        when BorrowCopy
        then [CopyBorrowed]
      }
      event CopyBorrowed {
        fields {
          loanId   string required
          memberId string required
          copyId   string required
          dueOn    date   required
        }
      }
      spec "refuses a copy already on loan" {
        given [CopyBorrowed]
        when BorrowCopy
        then rejected OneCopyPerLoan
      }
      flow {
        command -> event: BorrowCopy -> CopyBorrowed
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
      spec "returns a copy the member holds" {
        given [CopyBorrowed]
        then [CopyReturned]
        when ReturnCopy
      }
      flow {
        command -> event: ReturnCopy -> CopyReturned
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
    spec "seats a reader at a free desk" {
      given []
      when ClaimDesk
      then [DeskClaimed]
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
    spec "refuses a desk another reader is seated at" {
      given [DeskClaimed]
      when ClaimDesk
      then rejected OneReaderPerDesk
    }
    flow {
      command -> event: ClaimDesk -> DeskClaimed
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
    spec "frees the desk its reader is seated at" {
      given [DeskClaimed]
      when ReleaseDesk
      then [DeskReleased]
    }
    flow {
      command -> event: ReleaseDesk -> DeskReleased
    }
  }
}
`

// WithOrdinaryFieldNames renames every field of model in place so that no name
// spells a DSL keyword, giving KeywordFieldSearchCatalog a twin that differs
// from it in nothing but the naming of its fields. Renaming the parsed model
// rather than its source keeps every recorded position identical, so a
// comparison between the two answers only the naming question.
func WithOrdinaryFieldNames(model *ast.Model) *ast.Model {
	if model == nil {
		return nil
	}
	for i, field := range declaredFields(model) {
		field.Name = fmt.Sprintf("attribute%d", i+1)
	}
	return model
}

func declaredFields(model *ast.Model) []*ast.Field {
	var fields []*ast.Field
	for _, ctx := range model.Contexts {
		for _, agg := range ctx.Aggregates {
			fields = append(fields, fieldsOfSlices(agg.Slices)...)
		}
		fields = append(fields, fieldsOfSlices(ctx.Slices)...)
	}
	return fields
}

func fieldsOfSlices(slices []*ast.Slice) []*ast.Field {
	var fields []*ast.Field
	for _, s := range slices {
		for _, cmd := range s.Commands {
			fields = append(fields, cmd.Fields...)
		}
		for _, evt := range s.Events {
			fields = append(fields, evt.Fields...)
		}
		for _, v := range s.Views {
			fields = append(fields, v.Fields...)
		}
		for _, tr := range s.Translations {
			if tr.Event != nil {
				fields = append(fields, tr.Event.Fields...)
			}
		}
	}
	return fields
}

// Unparseable starts with a keyword the language does not define, so the lexer
// and parser report on line 1.
const Unparseable = `foobar {
}
`

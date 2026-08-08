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
      trigger "Reservation Form" {
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
        on ReservationMade
        reads ReservationsView
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
      trigger "Reservation Form" {
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
        on ReservationMade
        reads ReservationsView
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
      trigger "Search Builder" {
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
        on SavedSearchDefined
        reads SavedSearchesView
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
      trigger "Lending Desk" {
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
      trigger "Lending Desk" {
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
      spec "refuses to return a copy the member no longer holds" {
        given [CopyBorrowed]
        when ReturnCopy
        then rejected OneCopyPerLoan
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
    spec "refuses to free a desk already empty" {
      given [DeskClaimed]
      when ReleaseDesk
      then rejected OneReaderPerDesk
    }
    flow {
      command -> event: ReleaseDesk -> DeskReleased
    }
  }
}
`

// SlicePatternLibraryLending states a spec for each slice pattern in both homes
// a slice has — nested in an aggregate, and declared directly on a DCB-mode
// context — and with every outcome a spec's then accepts, so packages that walk
// or strip specs share one model. The view, command and no-when outcomes only
// this fixture has sit in the aggregate home; the DCB home carries command and
// translation specs so both homes read back short against the transcriptions.
const SlicePatternLibraryLending = `# Lending a library's copies and seating its readers, with a spec for every slice pattern
model "Library Lending"

actor "Member"

context "Lending" {
  aggregate "Loan" {
    invariant OneCopyPerLoan "A loan covers exactly one copy of one title"

    slice "Borrow Copy" {
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
      spec "borrows a copy no one holds" {
        given []
        when BorrowCopy
        then [CopyBorrowed]
      }
      spec "refuses a copy already on loan" {
        when BorrowCopy
        then rejected OneCopyPerLoan
      }
      spec "borrows a copy the member before returned" {
        given [CopyBorrowed, CopyReturned]
        then [CopyBorrowed]
        when BorrowCopy
      }
      flow {
        command -> event: BorrowCopy -> CopyBorrowed
      }
    }

    slice "Review Member Loans" {
      view MemberLoansView {
        fields {
          loanId   string required
          memberId string required
          dueOn    date   required
        }
        subscribes [CopyBorrowed, CopyReturned]
      }
      spec "lists the loans a member holds" {
        then view MemberLoansView
      }
    }

    slice "Chase Overdue Copy" {
      view OverdueLoansView {
        fields {
          loanId   string required
          memberId string required
        }
        subscribes [CopyBorrowed, CopyReturned]
      }
      command RemindMember {
        fields {
          loanId   string required
          memberId string required
        }
      }
      event MemberReminded {
        fields {
          loanId     string required
          memberId   string required
          remindedAt timestamp required
        }
      }
      automation RemindOnDueDate {
        on CopyBorrowed
        reads OverdueLoansView
        command RemindMember
      }
      flow {
        command -> event: RemindMember -> MemberReminded
      }
      spec "reminds a member when a copy becomes due" {
        when CopyBorrowed
        then command RemindMember
      }
      spec "sanctions a member's loan" {
        when RemindMember
        then [MemberReminded]
      }
    }

    slice "Sweep Overdue Loans" {
      view OverdueLoansView {
        fields {
          loanId   string required
          memberId string required
        }
        subscribes [CopyBorrowed, CopyReturned]
      }
      command RecallCopy {
        fields {
          loanId string required
          copyId string required
        }
      }
      event CopyRecalled {
        fields {
          loanId     string required
          copyId     string required
          recalledAt timestamp required
        }
      }
      automation RecallOverdueCopy {
        every "15m"
        reads OverdueLoansView
        command RecallCopy
      }
      flow {
        command -> event: RecallCopy -> CopyRecalled
      }
      spec "recalls copies that are overdue" {
        then command RecallCopy
      }
      spec "calls in a copy before its due date" {
        when RecallCopy
        then [CopyRecalled]
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
          loanId     string required
          copyId     string required
          returnedAt timestamp required
        }
      }
      flow {
        command -> event: ReturnCopy -> CopyReturned
      }
      spec "returns a loaned copy to the library" {
        when ReturnCopy
        then [CopyReturned]
      }
    }
  }
}

context "Reading Room" mode dcb {
  invariant OneReaderPerDesk "A desk seats at most one reader at any moment"

  slice "Desk Occupancy" {
    view DeskOccupancyView {
      fields {
        deskId   string required
        memberId string required
      }
      subscribes [DeskClaimed]
    }
  }

  slice "Claim Desk" {
    command ClaimDesk {
      decides_on {
        events [DeskClaimed]
        where tag(desk = deskId) and tag(reader = memberId)
      }
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
    spec "seats a reader at a free desk" {
      given []
      when ClaimDesk
      then [DeskClaimed]
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

  slice "Import External Desk Booking" {
    command ImportExternalDeskBooking {
      fields {
        externalRef string required
        deskId      string required
        memberId    string required
      }
    }
    translation ExternalDeskBookingImport {
      external_system "Room Booking API"
      reads DeskOccupancyView
      command ImportExternalDeskBooking
      event ExternalDeskBookingImported {
        tags {
          desk  : deskId
          reader: memberId
        }
        fields {
          externalRef string    required
          deskId      string    required
          memberId    string    required
          importedAt  timestamp required
        }
      }
    }
    spec "imports a desk booking from an external system" {
      given [DeskClaimed]
      when ImportExternalDeskBooking
      then [ExternalDeskBookingImported]
    }
  }
}
`

// AutomationReadsLibraryLending names the view its automations read in both
// homes a slice has — nested in an aggregate, and declared directly on a
// DCB-mode context — and one of them reads a view another context declares, so
// packages that carry an automation's reads through the pipeline share one
// model. One automation leaves its reads out ahead of a further automation on
// purpose: write the omission last and nothing here would catch an automation
// running on into what follows it.
const AutomationReadsLibraryLending = `# Lending a library's copies and seating its readers, with the views its automations consult
model "Library Lending"

actor "Member"

context "Lending" {
  aggregate "Loan" {
    slice "Borrow Copy" {
      trigger "Lending Desk" {
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
    slice "Chase Overdue Copy" {
      command RemindMember {
        fields {
          loanId   string required
          memberId string required
        }
      }
      command RecallCopy {
        fields {
          loanId string required
          copyId string required
        }
      }
      event MemberReminded {
        fields {
          loanId     string    required
          memberId   string    required
          remindedAt timestamp required
        }
      }
      event CopyRecalled {
        fields {
          loanId     string    required
          copyId     string    required
          recalledAt timestamp required
        }
      }
      automation RemindOnDueDate {
        on CopyBorrowed
        command RemindMember
      }
      automation RecallOverdueCopy {
        on CopyBorrowed
        reads MemberLoansView
        command RecallCopy
      }
      flow {
        command -> event: RemindMember -> MemberReminded
        command -> event: RecallCopy -> CopyRecalled
      }
    }
  }
}

context "Reading Room" mode dcb {
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
  slice "Browse Desk Occupancy" {
    view DeskOccupancyView {
      fields {
        deskId    string    required
        memberId  string    required
        claimedAt timestamp required
      }
      subscribes [DeskClaimed, DeskReleased]
    }
  }
  slice "Close Reading Room" {
    automation FreeDeskAtClosing {
      on DeskClaimed
      reads DeskOccupancyView
      command ReleaseDesk
    }
    automation RemindReaderOfLoans {
      on DeskReleased
      reads MemberLoansView
      command RemindMember
    }
  }
}
`

// TriggerReadsLibraryLending names the view its triggers read in both homes a
// slice has — nested in an aggregate, and declared directly on a DCB-mode
// context — and every view named is one the model itself declares, one of them
// from another slice and one from another context, so packages that resolve a
// trigger's reads to the view it opens on share one model. A trigger and an
// automation each leave their reads out ahead of further declarations on
// purpose: write the omission last and nothing here would catch a declaration
// running on into what follows it.
const TriggerReadsLibraryLending = `# Lending a library's copies and seating its readers, with the views its triggers open on
model "Library Lending"

actor "Member"
actor "Librarian"

context "Lending" {
  aggregate "Loan" {
    slice "Borrow Copy" {
      trigger "Lending Desk" {
        actor Member
        reads MemberLoansView
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
    slice "Return Copy" {
      trigger "Returns Counter" {
        actor Member
      }
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
    }
    slice "Chase Overdue Copy" {
      trigger "Overdue Report" {
        actor Librarian
        reads DeskOccupancyView
      }
      command RemindMember {
        fields {
          loanId   string required
          memberId string required
        }
      }
      command RecallCopy {
        fields {
          loanId string required
          copyId string required
        }
      }
      event MemberReminded {
        fields {
          loanId     string    required
          memberId   string    required
          remindedAt timestamp required
        }
      }
      event CopyRecalled {
        fields {
          loanId     string    required
          copyId     string    required
          recalledAt timestamp required
        }
      }
      automation RemindOnDueDate {
        on CopyBorrowed
        command RemindMember
      }
      automation RecallOverdueCopy {
        on CopyBorrowed
        reads MemberLoansView
        command RecallCopy
      }
      flow {
        command -> event: RemindMember -> MemberReminded
        command -> event: RecallCopy -> CopyRecalled
      }
    }
  }
}

context "Reading Room" mode dcb {
  slice "Claim Desk" {
    trigger "Desk Kiosk" {
      actor Member
      reads DeskOccupancyView
    }
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
  slice "Browse Desk Occupancy" {
    view DeskOccupancyView {
      fields {
        deskId    string    required
        memberId  string    required
        claimedAt timestamp required
      }
      subscribes [DeskClaimed, DeskReleased]
    }
  }
  slice "Close Reading Room" {
    automation FreeDeskAtClosing {
      on DeskClaimed
      reads DeskOccupancyView
      command ReleaseDesk
    }
    automation RemindReaderOfLoans {
      on DeskReleased
      reads MemberLoansView
      command RemindMember
    }
  }
}
`

// AutomationScheduleLibraryLending runs automations on a schedule in both homes
// a slice has — nested in an aggregate, and declared directly on a DCB-mode
// context — stating a duration in one automation and a cron expression in
// another within each home, beside automations that name an activation event
// instead, so packages that carry a schedule through the pipeline share one
// model. An automation stating an event and no schedule sits mid-block ahead of
// a further automation on purpose: write the omission last and nothing here
// would catch an automation running on into what follows it.
const AutomationScheduleLibraryLending = `# Lending a library's copies and seating its readers, with the cadences its automations run on
model "Library Lending"

actor "Member"

context "Lending" {
  aggregate "Loan" {
    slice "Borrow Copy" {
      trigger "Lending Desk" {
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
    slice "Chase Overdue Copy" {
      command RemindMember {
        fields {
          loanId   string required
          memberId string required
        }
      }
      command RecallCopy {
        fields {
          loanId string required
          copyId string required
        }
      }
      event MemberReminded {
        fields {
          loanId     string    required
          memberId   string    required
          remindedAt timestamp required
        }
      }
      event CopyRecalled {
        fields {
          loanId     string    required
          copyId     string    required
          recalledAt timestamp required
        }
      }
      automation RemindMemberEachMorning {
        every "0 9 * * *"
        reads MemberLoansView
        command RemindMember
      }
      automation RecallOnSecondReminder {
        on MemberReminded
        command RecallCopy
      }
      automation SweepOverdueLoans {
        every "15m"
        command RecallCopy
      }
      flow {
        command -> event: RemindMember -> MemberReminded
        command -> event: RecallCopy -> CopyRecalled
      }
    }
  }
}

context "Reading Room" mode dcb {
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
  slice "Browse Desk Occupancy" {
    view DeskOccupancyView {
      fields {
        deskId    string    required
        memberId  string    required
        claimedAt timestamp required
      }
      subscribes [DeskClaimed, DeskReleased]
    }
  }
  slice "Close Reading Room" {
    automation CloseDesksAtNight {
      every "0 22 * * *"
      reads DeskOccupancyView
      command ReleaseDesk
    }
    automation RemindReaderOfLoans {
      on DeskReleased
      reads MemberLoansView
      command RemindMember
    }
    automation SweepIdleDesks {
      every "45m"
      command ReleaseDesk
    }
  }
}
`

// SpecLibraryLendingSpecNames transcribes the name of every scenario
// SpecLibraryLending states, both slice homes together and in declaration order,
// so a walk or a strip that reaches only one of the homes reads back short
// against it.
var SpecLibraryLendingSpecNames = []string{
	"borrows a copy no one holds",
	"borrows a copy the member before returned",
	"refuses a copy already on loan",
	"returns a copy the member holds",
	"refuses to return a copy the member no longer holds",
	"seats a reader at a free desk",
	"refuses a desk another reader is seated at",
	"frees the desk its reader is seated at",
	"refuses to free a desk already empty",
}

// SlicePatternLibraryLendingSpecNames transcribes the name of every scenario
// SlicePatternLibraryLending states, both slice homes together and in
// declaration order, so a walk or a strip that reaches only one of the homes
// reads back short against it.
var SlicePatternLibraryLendingSpecNames = []string{
	"borrows a copy no one holds",
	"refuses a copy already on loan",
	"borrows a copy the member before returned",
	"lists the loans a member holds",
	"reminds a member when a copy becomes due",
	"sanctions a member's loan",
	"recalls copies that are overdue",
	"calls in a copy before its due date",
	"returns a loaned copy to the library",
	"seats a reader at a free desk",
	"refuses a desk another reader is seated at",
	"imports a desk booking from an external system",
}

// SlicePatternLibraryLendingOutcomeKinds transcribes the outcome kind of every
// scenario SlicePatternLibraryLending states, both slice homes together and in
// declaration order, so a walk or a strip that collapses one outcome kind into
// another reads back short against it.
var SlicePatternLibraryLendingOutcomeKinds = []string{
	"events",
	"rejection",
	"events",
	"view",
	"command",
	"events",
	"command",
	"events",
	"events",
	"events",
	"rejection",
	"events",
}

// AutomationReadsLibraryLendingViewNames transcribes the view every automation
// of AutomationReadsLibraryLending reads, both slice homes together and in
// declaration order, so a walk or a strip that reaches only one of the homes
// reads back short against it. The automation that reads no view contributes
// nothing, and the view a trigger reads is not an automation's, so the list is
// shorter than either count.
var AutomationReadsLibraryLendingViewNames = []string{
	"MemberLoansView",
	"DeskOccupancyView",
	"MemberLoansView",
}

// AutomationReadsLibraryLendingActivationEvents transcribes the event every
// automation of AutomationReadsLibraryLending activates on, both slice homes
// together and in declaration order, so a walk or a rewrite that drops one reads
// back short against it. Every automation names one, so the list is as long as
// the automation count and longer than the views above.
var AutomationReadsLibraryLendingActivationEvents = []string{
	"CopyBorrowed",
	"CopyBorrowed",
	"DeskClaimed",
	"DeskReleased",
}

// TriggerReadsLibraryLendingTriggerViewNames transcribes the view every trigger
// of TriggerReadsLibraryLending reads, both slice homes together and in
// declaration order, so a walk or a strip that reaches only one of the homes
// reads back short against it. The trigger that reads no view contributes
// nothing, and neither does a slice carrying no trigger, so the list is shorter
// than either count.
var TriggerReadsLibraryLendingTriggerViewNames = []string{
	"MemberLoansView",
	"DeskOccupancyView",
	"DeskOccupancyView",
}

// TriggerReadsLibraryLendingAutomationViewNames transcribes the view every
// automation of TriggerReadsLibraryLending reads, both slice homes together and
// in declaration order, so the same fixture answers for both constructs that
// name a view they read and a strip reaching the wrong one reads back short.
var TriggerReadsLibraryLendingAutomationViewNames = []string{
	"MemberLoansView",
	"DeskOccupancyView",
	"MemberLoansView",
}

// AutomationScheduleLibraryLendingSchedules transcribes the cadence every
// automation of AutomationScheduleLibraryLending runs on, both slice homes
// together and in declaration order, so a walk or a rewrite that reaches only
// one of the homes reads back short against it. The automations naming an
// activation event instead contribute nothing, so the list is shorter than the
// automation count.
var AutomationScheduleLibraryLendingSchedules = []string{
	"0 9 * * *",
	"15m",
	"0 22 * * *",
	"45m",
}

// AutomationScheduleLibraryLendingActivationEvents transcribes the event the
// automations of AutomationScheduleLibraryLending that state one activate on,
// both slice homes together and in declaration order, so the same fixture is
// read back for both forms an automation has of stating when it runs.
var AutomationScheduleLibraryLendingActivationEvents = []string{
	"MemberReminded",
	"DeskReleased",
}

// DescribedHotelReservationDescriptions transcribes the description
// DescribedHotelReservation states for every construct declared under its model,
// filed under the name that construct carries, with the event a translation
// nests filed under the translation's own name followed by "event". The model's
// own description is left out: it belongs to the model rather than to a
// construct the model declares.
var DescribedHotelReservationDescriptions = map[string]string{
	"Guest":                    "A person booking a room, not necessarily the one staying in it",
	"Reservations":             "Everything the hotel knows about a stay before the guest arrives",
	"Reservation":              "One guest holding one room over one date range",
	"Make Reservation":         "A guest books a room from the public site",
	"Reservation Form":         "The booking form on the public site",
	"MakeReservation":          "Ask the hotel to hold a room for a date range, 10% deposit taken up front",
	"ReservationMade":          "A room is held for a guest",
	"View Reservations":        "A guest reviews the stays they have booked",
	"ReservationsView":         "Every reservation with the stage it has reached",
	"Auto Confirm Reservation": "The hotel confirms a held room without a clerk touching it",
	"ConfirmReservation":       "Turn a held room into a confirmed stay",
	"AutoConfirm":              "Confirms every reservation the moment it is made",
	"Import External Booking":  "A booking made on a partner site becomes a reservation here",
	"ImportBooking":            "Record a booking taken by a partner site",
	"BookingImport":            "Restates a partner webhook in the hotel's own language",
	"BookingImport event":      "A partner site reported a booking",
}

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

// WithoutSpecs returns a copy of model whose slices state no spec, in both homes
// a slice has — nested in an aggregate and declared directly on a context. The
// original keeps every spec it was written with, so a caller comparing the two
// is not comparing a model with itself.
func WithoutSpecs(model *ast.Model) *ast.Model {
	return copyWithEditedSlices(model, func(s *ast.Slice) {
		s.Specs = nil
	})
}

// WithoutAutomationReads returns a copy of model whose automations name no view
// they read, in both homes a slice has — nested in an aggregate and declared
// directly on a context. The original keeps every view it was written reading,
// so a caller comparing the two is not comparing a model with itself. What a
// trigger and a translation read is left alone: only an automation's reads is
// the subject of the comparison.
func WithoutAutomationReads(model *ast.Model) *ast.Model {
	return copyWithEditedSlices(model, func(s *ast.Slice) {
		s.Automations = editedCopies(s.Automations, func(auto *ast.Automation) {
			auto.Reads = ""
			auto.ReadsPos = ast.Position{}
		})
	})
}

// WithoutTriggerReads returns a copy of model whose triggers name no view they
// read, in both homes a slice has — nested in an aggregate and declared directly
// on a context. The original keeps every view it was written reading, so a
// caller comparing the two is not comparing a model with itself. What an
// automation and a translation read is left alone: only a trigger's reads is the
// subject of the comparison.
func WithoutTriggerReads(model *ast.Model) *ast.Model {
	return copyWithEditedSlices(model, func(s *ast.Slice) {
		s.Trigger = editedCopy(s.Trigger, func(trigger *ast.Trigger) {
			trigger.Reads = ""
			trigger.ReadsPos = ast.Position{}
		})
	})
}

// copyWithEditedSlices hands edit a copy of every slice in both homes — nested in
// an aggregate and declared directly on a context. A copied slice still points at
// the original's commands, events, views and automations, so an edit reaching
// inside one of those has to copy it too or it writes through to the model this
// was called with.
func copyWithEditedSlices(model *ast.Model, edit func(*ast.Slice)) *ast.Model {
	if model == nil {
		return nil
	}
	copied := *model
	copied.Contexts = editedCopies(model.Contexts, func(ctx *ast.Context) {
		ctx.Slices = editedCopies(ctx.Slices, edit)
		ctx.Aggregates = editedCopies(ctx.Aggregates, func(agg *ast.Aggregate) {
			agg.Slices = editedCopies(agg.Slices, edit)
		})
	})
	return &copied
}

// editedCopies applies edit to a copy of every item and leaves a nil list nil
// rather than preallocating an empty one: an empty list differs from the original
// all by itself, which is enough to satisfy a caller's "the twin differs" guard
// with nothing edited at all.
func editedCopies[T any](items []*T, edit func(*T)) []*T {
	var copies []*T
	for _, item := range items {
		copies = append(copies, editedCopy(item, edit))
	}
	return copies
}

// editedCopy edits a copy rather than the item it was handed, because a slice
// holds its trigger as a single pointer into the model the caller passed:
// clearing that trigger where it sits leaves the twin and the original reading
// alike.
func editedCopy[T any](item *T, edit func(*T)) *T {
	if item == nil {
		return nil
	}
	edited := *item
	edit(&edited)
	return &edited
}

// DeclaredSpecNames names every spec model states, both slice homes together and
// in declaration order, so a caller pairing it with a transcribed list reads back
// short when a strip or a walk reaches only one of the homes.
func DeclaredSpecNames(model *ast.Model) []string {
	var names []string
	for _, s := range declaredSlices(model) {
		for _, spec := range s.Specs {
			names = append(names, spec.Name)
		}
	}
	return names
}

// DeclaredDescriptions files the description every construct of model states
// under the name that construct carries, both slice homes together, with the
// event a translation nests filed under the translation's own name followed by
// "event", so a caller pairing it with a transcribed map reads back short when a
// walk reaches only one of the homes. The model's own description is left out,
// so the map holds constructs only.
func DeclaredDescriptions(model *ast.Model) map[string]string {
	described := make(map[string]string)
	describe := func(name, description string) {
		if description != "" {
			described[name] = description
		}
	}

	for _, a := range model.Actors {
		describe(a.Name, a.Description)
	}
	for _, ctx := range model.Contexts {
		describe(ctx.Name, ctx.Description)
		for _, agg := range ctx.Aggregates {
			describe(agg.Name, agg.Description)
		}
	}
	for _, s := range declaredSlices(model) {
		describe(s.Name, s.Description)
		if s.Trigger != nil {
			describe(s.Trigger.Name, s.Trigger.Description)
		}
		for _, cmd := range s.Commands {
			describe(cmd.Name, cmd.Description)
		}
		for _, evt := range s.Events {
			describe(evt.Name, evt.Description)
		}
		for _, v := range s.Views {
			describe(v.Name, v.Description)
		}
		for _, auto := range s.Automations {
			describe(auto.Name, auto.Description)
		}
		for _, tr := range s.Translations {
			describe(tr.Name, tr.Description)
			if tr.Event != nil {
				describe(tr.Name+" event", tr.Event.Description)
			}
		}
	}

	return described
}

// DeclaredSpecOutcomeKinds names the outcome kind of every spec model states,
// both slice homes together and in declaration order, so a caller pairing it
// with a transcribed list reads back short when a strip or a walk collapses one
// outcome kind into another.
func DeclaredSpecOutcomeKinds(model *ast.Model) []string {
	var kinds []string
	for _, s := range declaredSlices(model) {
		for _, spec := range s.Specs {
			kinds = append(kinds, specOutcomeKind(spec.Then))
		}
	}
	return kinds
}

func specOutcomeKind(then ast.ThenClause) string {
	switch then.(type) {
	case *ast.ThenEvents:
		return "events"
	case *ast.ThenRejected:
		return "rejection"
	case *ast.ThenView:
		return "view"
	case *ast.ThenCommand:
		return "command"
	default:
		return fmt.Sprintf("%T", then)
	}
}

// DeclaredAutomationReads names the view every automation of model reads, both
// slice homes together and in declaration order, so a caller pairing it with a
// transcribed list reads back short when a strip or a walk reaches only one of
// the homes.
func DeclaredAutomationReads(model *ast.Model) []string {
	return declaredAutomationEntries(model, func(auto *ast.Automation) string { return auto.Reads })
}

// DeclaredTriggerReads names the view every trigger of model reads, both slice
// homes together and in declaration order, so a caller pairing it with a
// transcribed list reads back short when a strip or a walk reaches only one of
// the homes. A slice with no trigger and a trigger naming no view contribute
// nothing, so the list counts what the model says rather than how many slices it
// declares.
func DeclaredTriggerReads(model *ast.Model) []string {
	var reads []string
	for _, s := range declaredSlices(model) {
		if s.Trigger != nil && s.Trigger.Reads != "" {
			reads = append(reads, s.Trigger.Reads)
		}
	}
	return reads
}

// DeclaredActivationEvents names the event every automation of model activates
// on, both slice homes together and in declaration order, so a caller pairing it
// with a transcribed list reads back short when an entry goes missing or a walk
// reaches only one of the homes.
func DeclaredActivationEvents(model *ast.Model) []string {
	return declaredAutomationEntries(model, func(auto *ast.Automation) string { return auto.OnEvent })
}

// DeclaredSchedules names the cadence every automation of model runs on, both
// slice homes together and in declaration order, so a caller pairing it with a
// transcribed list reads back short when an entry goes missing or a walk reaches
// only one of the homes.
func DeclaredSchedules(model *ast.Model) []string {
	return declaredAutomationEntries(model, func(auto *ast.Automation) string { return auto.Schedule })
}

// declaredAutomationEntries leaves out the automations stating no entry, so a
// list read back against a transcribed one counts what the model says rather
// than how many automations it declares.
func declaredAutomationEntries(model *ast.Model, entry func(*ast.Automation) string) []string {
	var entries []string
	for _, s := range declaredSlices(model) {
		for _, auto := range s.Automations {
			if stated := entry(auto); stated != "" {
				entries = append(entries, stated)
			}
		}
	}
	return entries
}

func declaredSlices(model *ast.Model) []*ast.Slice {
	var slices []*ast.Slice
	for _, ctx := range model.Contexts {
		for _, agg := range ctx.Aggregates {
			slices = append(slices, agg.Slices...)
		}
		slices = append(slices, ctx.Slices...)
	}
	return slices
}

func declaredFields(model *ast.Model) []*ast.Field {
	return fieldsOfSlices(declaredSlices(model))
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

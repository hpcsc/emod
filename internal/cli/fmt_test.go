//go:build unit

package cli_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hpcsc/emod/internal/cli"
	"github.com/hpcsc/emod/internal/test"
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
        subscribes [SavedSearchDefined]
        fields {
          searchId    string required
          description string required
          tag         string required
          model       string
          where       string required
          matches     int    required
        }
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
        reads SavedSearchesView
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
        reads SavedSearchesView
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

      spec "borrows a copy no one holds" {
        when BorrowCopy
        then [CopyBorrowed]
      }

      spec "borrows a copy the member before returned" {
        given [CopyBorrowed, CopyReturned]
        when  BorrowCopy
        then  [CopyBorrowed]
      }

      spec "refuses a copy already on loan" {
        given [CopyBorrowed]
        when  BorrowCopy
        then  rejected OneCopyPerLoan
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
        when  ReturnCopy
        then  [CopyReturned]
      }

      spec "refuses to return a copy the member no longer holds" {
        given [CopyBorrowed]
        when  ReturnCopy
        then  rejected OneCopyPerLoan
      }
    }

    slice "Review Member Loans" {
      view MemberLoansView {
        subscribes [CopyBorrowed]
        fields {
          loanId   string required
          memberId string required
          dueOn    date   required
        }
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
      when  ClaimDesk
      then  rejected OneReaderPerDesk
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
      when  ReleaseDesk
      then  [DeskReleased]
    }

    spec "refuses to free a desk already empty" {
      given [DeskClaimed]
      when  ReleaseDesk
      then  rejected OneReaderPerDesk
    }
  }
}
`

// payloadFormattedEmod is what emod fmt writes for test.PayloadLibraryLending:
// each payload on the line of the reference it qualifies, as one canonical
// comma-separated brace block, with 12.50 written back whole.
const payloadFormattedEmod = `emod 1
# Lending a library's copies and seating its readers, with example values on the scenarios each slice must satisfy
model "Library Lending"

actor "Member"

context "Lending" {
  aggregate "Loan" {
    invariant OneCopyPerLoan "A loan covers exactly one copy of one title"
    slice "Borrow Copy" {
      command BorrowCopy {
        fields {
          memberId  string    required
          copyId    string    required
          dueOn     date      required
          shelfMark ShelfMark
        }
      }

      event CopyBorrowed {
        fields {
          loanId          uuid      required
          memberId        string    required
          copyId          string    required
          dueOn           date      required
          borrowedAt      timestamp required
          catalogueNumber int
          lateFee         decimal
          expedited       bool
        }
      }

      flow {
        command -> event: BorrowCopy -> CopyBorrowed
      }

      spec "borrows a copy no one holds" {
        when BorrowCopy {
          memberId:  "M-40817"
          copyId:    "C-93204"
          dueOn:     "2024-07-19"
          shelfMark: "AURELIA"
        }
        then [
          CopyBorrowed {
            loanId:          "7c9e6679-7425-40de-944b-e07fc1f90ae7"
            borrowedAt:      "2024-07-05T14:32:00Z"
            catalogueNumber: 4821
            lateFee:         12.50
            expedited:       true
          }
        ]
      }

      spec "refuses a copy already on loan" {
        given [CopyBorrowed { copyId: "C-93204", expedited: false }, CopyReturned]
        when  BorrowCopy { copyId: "C-93204" }
        then  rejected OneCopyPerLoan
      }
    }

    slice "Return Copy" {
      command ReturnCopy {
        fields {
          loanId uuid   required
          copyId string required
        }
      }

      event CopyReturned {
        fields {
          loanId     uuid      required
          copyId     string    required
          returnedAt timestamp required
        }
      }

      flow {
        command -> event: ReturnCopy -> CopyReturned
      }

      spec "returns a copy the member holds" {
        given [CopyBorrowed]
        when  ReturnCopy
        then  [CopyReturned]
      }

      spec "refuses to return a copy the member no longer holds" {
        given [CopyBorrowed]
        when  ReturnCopy
        then  rejected OneCopyPerLoan
      }
    }

    slice "Review Member Loans" {
      view MemberLoansView {
        subscribes [CopyBorrowed]
        fields {
          loanId   uuid   required
          memberId string required
          dueOn    date   required
        }
      }
    }
  }
}

context "Reading Room" mode dcb {
  invariant OneReaderPerDesk "A desk seats at most one reader at any moment"
  slice "Claim Desk" {
    command ClaimDesk {
      decides_on {
        events [DeskClaimed, DeskReleased]
        where tag(desk = deskId)
      }
      fields {
        memberId      string required
        deskId        string required
        preferredZone string
      }
    }

    event DeskClaimed {
      tags {
        desk  : deskId
        reader: memberId
      }
      fields {
        sessionId uuid      required
        deskId    string    required
        memberId  string    required
        claimedAt timestamp required
        quietZone bool
      }
    }

    flow {
      command -> event: ClaimDesk -> DeskClaimed
    }

    spec "seats a reader at a desk its last reader released" {
      given [DeskReleased { deskId: "D-5817", releasedAt: "2024-07-05T08:50:00Z" }]
      when  ClaimDesk { memberId: "M-40817", preferredZone: "north gallery" }
      then  [
        DeskClaimed {
          sessionId: "b6f4a3d2-91c8-4e57-8f10-2d6a5c7e9b31"
          claimedAt: "2024-07-05T09:15:00Z"
          quietZone: true
        }
      ]
    }

    spec "refuses a desk another reader is seated at" {
      given [DeskClaimed { deskId: "D-5817", quietZone: false }]
      when  ClaimDesk { memberId: "M-63204", deskId: "D-5817" }
      then  rejected OneReaderPerDesk
    }
  }

  slice "Release Desk" {
    command ReleaseDesk {
      decides_on {
        events [DeskClaimed]
        where tag(desk = deskId) and tag(reader = memberId)
      }
      fields {
        sessionId uuid   required
        deskId    string required
        memberId  string required
      }
    }

    event DeskReleased {
      tags {
        desk  : deskId
        reader: memberId
      }
      fields {
        sessionId  uuid      required
        deskId     string    required
        memberId   string    required
        releasedAt timestamp required
        seatedFor  decimal
      }
    }

    flow {
      command -> event: ReleaseDesk -> DeskReleased
    }

    spec "frees the desk its reader is seated at" {
      given [DeskClaimed { deskId: "D-5817", memberId: "M-40817", quietZone: false }]
      when  ReleaseDesk {
        sessionId: "b6f4a3d2-91c8-4e57-8f10-2d6a5c7e9b31"
        deskId:    "D-5817"
        memberId:  "M-40817"
      }
      then  [DeskReleased { releasedAt: "2024-07-05T11:40:00Z", seatedFor: 145.25 }]
    }

    spec "refuses to free a desk already empty" {
      given [
        DeskClaimed {
          sessionId: "3f21c8a7-6d94-4b02-9e15-7c8a3d5f2b64"
          claimedAt: "2024-07-04T16:05:00Z"
          quietZone: true
        }
      ]
      when  ReleaseDesk {
        sessionId: "b6f4a3d2-91c8-4e57-8f10-2d6a5c7e9b31"
        deskId:    "D-5817"
        memberId:  "M-40817"
      }
      then  rejected OneReaderPerDesk
    }
  }
}
`

const scheduledAutomationEmod = `model "Order Fulfilment"

context "Fulfilment" {
  aggregate "Shipment" {
    slice "Sweep Expired Holds" {
      command ReleaseExpiredHolds {
        fields {
          holdId string required
        }
      }

      event ExpiredHoldsReleased {
        fields {
          holdId string required
          releasedAt timestamp required
        }
      }

      automation NightlyExpirySweep {
        description "Releases the holds nobody paid for overnight"
        command ReleaseExpiredHolds
            every "0 2 * * *"
      }

      flow {
        command -> event: ReleaseExpiredHolds -> ExpiredHoldsReleased
      }
    }
  }
}
`

const scheduledAutomationFormattedEmod = `emod 1
model "Order Fulfilment"

context "Fulfilment" {
  aggregate "Shipment" {
    slice "Sweep Expired Holds" {
      command ReleaseExpiredHolds {
        fields {
          holdId string required
        }
      }

      event ExpiredHoldsReleased {
        fields {
          holdId     string    required
          releasedAt timestamp required
        }
      }

      automation NightlyExpirySweep {
        description "Releases the holds nobody paid for overnight"
        every "0 2 * * *"
        command ReleaseExpiredHolds
      }

      flow {
        command -> event: ReleaseExpiredHolds -> ExpiredHoldsReleased
      }
    }
  }
}
`

const delayedAutomationEmod = `model "Order Fulfilment"

context "Fulfilment" {
  aggregate "Shipment" {
    slice "Sweep Expired Holds" {
      command ReleaseExpiredHolds {
        fields {
          holdId string required
        }
      }

      event ExpiredHoldsReleased {
        fields {
          holdId string required
          releasedAt timestamp required
        }
      }

      event RoomHeld {
        source external "Booking"
        fields {
          holdId string required
        }
      }

      automation ExpiredHoldReleaser {
        description "Releases the holds nobody paid for overnight"
        command ReleaseExpiredHolds
            on RoomHeld     after    "24h"
      }

      flow {
        command -> event: ReleaseExpiredHolds -> ExpiredHoldsReleased
      }
    }
  }
}
`

const delayedAutomationFormattedEmod = `emod 1
model "Order Fulfilment"

context "Fulfilment" {
  aggregate "Shipment" {
    slice "Sweep Expired Holds" {
      command ReleaseExpiredHolds {
        fields {
          holdId string required
        }
      }

      event ExpiredHoldsReleased {
        fields {
          holdId     string    required
          releasedAt timestamp required
        }
      }

      event RoomHeld {
        source external "Booking"
        fields {
          holdId string required
        }
      }

      automation ExpiredHoldReleaser {
        description "Releases the holds nobody paid for overnight"
        on RoomHeld after "24h"
        command ReleaseExpiredHolds
      }

      flow {
        command -> event: ReleaseExpiredHolds -> ExpiredHoldsReleased
      }
    }
  }
}
`

const wireTypeEmod = `model "Reservations"

context "Booking" {
  aggregate "Reservation" {
    slice "Reserve Room" {
      command ReserveRoom {
        fields {
          guestId string required
        }
      }

      event RoomReserved {
        fields {
          reservationId string required
          guestId string required
        }
            type "com.acme.reservations.room-reserved"
      }

      flow {
        command -> event: ReserveRoom -> RoomReserved
      }
    }
  }
}
`

// wireTypeFormattedEmod is what emod fmt writes for wireTypeEmod, not that
// fixture re-indented: the header is added, the wire type moves from below the
// fields block to the event's first line, and the field columns align.
const wireTypeFormattedEmod = `emod 1
model "Reservations"

context "Booking" {
  aggregate "Reservation" {
    slice "Reserve Room" {
      command ReserveRoom {
        fields {
          guestId string required
        }
      }

      event RoomReserved {
        type "com.acme.reservations.room-reserved"
        fields {
          reservationId string required
          guestId       string required
        }
      }

      flow {
        command -> event: ReserveRoom -> RoomReserved
      }
    }
  }
}
`

// rejectionFormattedEmod is what emod fmt writes for test.RejectionLibraryLending,
// not that fixture re-indented: the header is added, each slice's specs move below
// its flow block, an empty given history loses its line, and a flow block's two
// entry kinds are written in canonical order.
const rejectionFormattedEmod = `emod 1
# Lending a library's copies and seating its readers, with the rejections each command can meet
model "Library Lending"

actor "Member"

context "Lending" {
  aggregate "Loan" {
    invariant OneCopyPerLoan "A loan covers exactly one copy of one title"
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
        command -> event:    BorrowCopy -> CopyBorrowed
        command -> rejected: BorrowCopy -> OneCopyPerLoan
      }

      spec "borrows a copy no one holds" {
        when BorrowCopy
        then [CopyBorrowed]
      }

      spec "refuses a copy already on loan" {
        given [CopyBorrowed]
        when  BorrowCopy
        then  rejected OneCopyPerLoan
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
        command -> event:    ReturnCopy -> CopyReturned
        command -> rejected: ReturnCopy -> OneCopyPerLoan
      }

      spec "returns a copy the member holds" {
        given [CopyBorrowed]
        when  ReturnCopy
        then  [CopyReturned]
      }

      spec "refuses to return a copy the member no longer holds" {
        given [CopyBorrowed]
        when  ReturnCopy
        then  rejected OneCopyPerLoan
      }
    }

    slice "Review Member Loans" {
      view MemberLoansView {
        subscribes [CopyBorrowed]
        fields {
          loanId   string required
          memberId string required
          dueOn    date   required
        }
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
      command -> event:    ClaimDesk -> DeskClaimed
      command -> rejected: ClaimDesk -> OneReaderPerDesk
    }

    spec "seats a reader at a free desk" {
      when ClaimDesk
      then [DeskClaimed]
    }

    spec "refuses a desk another reader is seated at" {
      given [DeskClaimed]
      when  ClaimDesk
      then  rejected OneReaderPerDesk
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
      when  ReleaseDesk
      then  [DeskReleased]
    }

    spec "refuses to free a desk already empty" {
      given [DeskClaimed]
      when  ReleaseDesk
      then  rejected OneReaderPerDesk
    }
  }
}
`

const slicePatternFormattedEmod = `emod 1
# Lending a library's copies and seating its readers, with a spec for every slice pattern
model "Library Lending"

actor "Member"

context "Lending" {
  aggregate "Loan" {
    invariant OneCopyPerLoan "A loan covers exactly one copy of one title"
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

      spec "borrows a copy no one holds" {
        when BorrowCopy
        then [CopyBorrowed]
      }

      spec "refuses a copy already on loan" {
        when BorrowCopy
        then rejected OneCopyPerLoan
      }

      spec "borrows a copy the member before returned" {
        given [CopyBorrowed, CopyReturned]
        when  BorrowCopy
        then  [CopyBorrowed]
      }
    }

    slice "Review Member Loans" {
      view MemberLoansView {
        subscribes [CopyBorrowed, CopyReturned]
        fields {
          loanId   string required
          memberId string required
          dueOn    date   required
        }
      }

      spec "lists the loans a member holds" {
        then view MemberLoansView
      }
    }

    slice "Chase Overdue Copy" {
      command RemindMember {
        fields {
          loanId   string required
          memberId string required
        }
      }

      event MemberReminded {
        fields {
          loanId     string    required
          memberId   string    required
          remindedAt timestamp required
        }
      }

      view OverdueLoansView {
        subscribes [CopyBorrowed, CopyReturned]
        fields {
          loanId   string required
          memberId string required
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

      spec "refuses to remind a member with no overdue loans" {
        given [MemberReminded]
        when  RemindMember
        then  rejected OneCopyPerLoan
      }
    }

    slice "Sweep Overdue Loans" {
      command RecallCopy {
        fields {
          loanId string required
          copyId string required
        }
      }

      event CopyRecalled {
        fields {
          loanId     string    required
          copyId     string    required
          recalledAt timestamp required
        }
      }

      view OverdueLoansView {
        subscribes [CopyBorrowed, CopyReturned]
        fields {
          loanId   string required
          memberId string required
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

      spec "refuses to recall a copy already returned" {
        given [CopyReturned]
        when  RecallCopy
        then  rejected OneCopyPerLoan
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

      spec "returns a loaned copy to the library" {
        when ReturnCopy
        then [CopyReturned]
      }

      spec "refuses to return a copy already returned" {
        given [CopyReturned]
        when  ReturnCopy
        then  rejected OneCopyPerLoan
      }
    }
  }
}

context "Reading Room" mode dcb {
  invariant OneReaderPerDesk "A desk seats at most one reader at any moment"
  slice "Desk Occupancy" {
    view DeskOccupancyView {
      subscribes [DeskClaimed]
      fields {
        deskId   string required
        memberId string required
      }
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

    flow {
      command -> event: ClaimDesk -> DeskClaimed
    }

    spec "seats a reader at a free desk" {
      when ClaimDesk
      then [DeskClaimed]
    }

    spec "refuses a desk another reader is seated at" {
      given [DeskClaimed]
      when  ClaimDesk
      then  rejected OneReaderPerDesk
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
      when  ImportExternalDeskBooking
      then  [ExternalDeskBookingImported]
    }

    spec "refuses to import a booking for an occupied desk" {
      given [DeskClaimed]
      when  ImportExternalDeskBooking
      then  rejected OneReaderPerDesk
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

	t.Run("keeps every example payload in both slice homes and settles after one run", func(t *testing.T) {
		path := writeTemp(t, "payloads.emod", test.PayloadLibraryLending)

		requireFmtSettlesOn(t, path, payloadFormattedEmod)
	})

	t.Run("keeps every spec outcome including view and command and settles after one run", func(t *testing.T) {
		path := writeTemp(t, "slice-patterns.emod", test.SlicePatternLibraryLending)

		requireFmtSettlesOn(t, path, slicePatternFormattedEmod)
	})

	t.Run("keeps every rejection edge in both slice homes and settles after one run", func(t *testing.T) {
		path := writeTemp(t, "rejections.emod", test.RejectionLibraryLending)

		requireFmtSettlesOn(t, path, rejectionFormattedEmod)
	})

	t.Run("moves an automation's schedule to its canonical line and settles after one run", func(t *testing.T) {
		path := writeTemp(t, "scheduled-automation.emod", scheduledAutomationEmod)

		requireFmtSettlesOn(t, path, scheduledAutomationFormattedEmod)
	})

	t.Run("moves a delayed activation to its canonical line, single spaced, and settles after one run", func(t *testing.T) {
		path := writeTemp(t, "delayed-automation.emod", delayedAutomationEmod)

		requireFmtSettlesOn(t, path, delayedAutomationFormattedEmod)
	})

	t.Run("moves an event's wire type to its canonical line and settles after one run", func(t *testing.T) {
		path := writeTemp(t, "wire-types.emod", wireTypeEmod)

		requireFmtSettlesOn(t, path, wireTypeFormattedEmod)
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

package test

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
model "Hotel Reservation"

actor "Guest"

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
        description "Ask the hotel to hold a room for a date range"
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

// Unparseable starts with a keyword the language does not define, so the lexer
// and parser report on line 1.
const Unparseable = `foobar {
}
`

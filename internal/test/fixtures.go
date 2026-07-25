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

// Unparseable starts with a keyword the language does not define, so the lexer
// and parser report on line 1.
const Unparseable = `foobar {
}
`

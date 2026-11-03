---
title: Getting started
weight: 1
---

emod is an Event Modeling DSL and visualization CLI. You write `.emod` files that describe
your commands, events, views, automations and translations; the tool validates them against
event-modeling anti-patterns and renders them as diagrams.

## Install

```sh
go install github.com/hpcsc/emod/cmd/emod@latest
```

Or build from source:

```sh
git clone https://github.com/hpcsc/emod.git
cd emod
go build -o ./bin/emod ./cmd/emod
```

## Version

```sh
emod version   # print the tag this build came from
```

A release carries the tag it was built from. A build from source has no tag, so `emod version`
prints the commit instead, and marks it `-dirty` when the working tree held changes.

## Update

`emod update` replaces the running binary with the latest release on GitHub, after checking the
download against the release's `checksums.txt`:

```sh
emod update               # install the latest release
emod update --check       # say what an update installs, and install nothing
emod update --prerelease  # install the latest build of main
emod update --force       # replace a build from a commit
```

## Write a model

Create `reservation.emod`:

```emod
emod = 1

model "Hotel Reservation" {
}

actor "Guest" {
}

context "Reservations" {
  aggregate "Reservation" {
    slice "Reserve a Room" {
      trigger "Reservation Form" {
        actor = Guest
        reads = AvailableRoomsView
      }

      command "ReserveRoom" {
        fields {
          roomId    = required(string)
          guestName = required(string)
          checkIn   = required(date)
          checkOut  = required(date)
        }
      }

      event "RoomReserved" {
        fields {
          reservationId = required(string)
          roomId        = required(string)
          guestName     = required(string)
          reservedAt    = required(timestamp)
        }
      }

      flow = <<-FLOW
        command -> event:    ReserveRoom -> RoomReserved
      FLOW
    }

    slice "View Available Rooms" {
      view "AvailableRoomsView" {
        subscribes = [RoomReserved]

        fields {
          roomId     = required(string)
          roomNumber = required(string)
          status     = required(string)
        }
      }
    }

    slice "Send Confirmation Email" {
      view "PendingConfirmationsView" {
        subscribes = [RoomReserved]

        fields {
          reservationId = required(string)
          guestName     = required(string)
          reservedAt    = required(timestamp)
        }
      }

      automation "ConfirmationEmailReactor" {
        on      = RoomReserved
        reads   = PendingConfirmationsView
        command = SendConfirmationEmail

        target {
          context = Notifications
        }
      }
    }
  }
}

context "Notifications" {
  aggregate "Notification" {
    slice "Send Confirmation" {
      command "SendConfirmationEmail" {
        fields {
          reservationId = required(string)
          guestName     = required(string)
          email         = required(string)
        }
      }
    }
  }
}
```

## Validate

```sh
emod validate reservation.emod
```

## Lint for anti-patterns

```sh
emod lint reservation.emod
```

## Format

```sh
emod fmt reservation.emod          # format in place
emod fmt --check reservation.emod  # check only (CI)
```

## Next

- [DSL reference](/docs/dsl-reference/) — every construct and pattern.
- [Diagrams](/docs/diagrams/) — SVG, Mermaid, ASCII, and event-flow.
- [Editor setup](/docs/editor-setup/) — language server integration.
- [Web viewer](/docs/viewer/) — open and edit diagrams in the browser.

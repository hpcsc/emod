---
title: DSL reference
weight: 2
---

The emod DSL is a human-readable, version-controllable format for describing event-driven
architectures using Event Modeling concepts. A model is written in HCL and files use the
`.emod` extension.

The full reference lives in the repository at
[`docs/dsl-reference.md`](https://github.com/hpcsc/emod/blob/main/docs/dsl-reference.md).
This page covers the shape of the language.

## Version header

A file opens with a version attribute, before `model`. It pins the file to a revision of the
grammar.

```emod
emod = 1
model "Hotel Reservation" {}
```

The version is a single integer. Absence means version 1. `emod fmt` writes the header as the
first line, so formatted files are always pinned.

## Top-level constructs

### `model`

Required root declaration. Exactly one per file, naming the system being modeled.

```emod
model "Hotel Reservation" {
  description = "Rooms, reservations and confirmations"
}
```

### `actor`

Declares a persona or external system that interacts with the model. Actors are referenced by
triggers.

```emod
actor "Guest" {
  description = "Someone booking a room"
}
```

## Bounded contexts

### `context`

Groups aggregates (or slices in DCB mode) under a bounded context. A model must contain at
least one context.

```emod
context "Reservations" {
  aggregate "Reservation" { ... }
}
```

An optional `mode` attribute declares the consistency-boundary style:

| Mode | Description |
|------|-------------|
| `aggregate` (default) | Slices live inside an `aggregate` block. |
| `dcb` | Slices live directly under the context; events use `tags`, commands use `decides_on`. |
| `mixed` | Both aggregate-wrapped and direct slices are accepted. |

### `aggregate`

Groups slices within a context and represents a consistency boundary. Not used in `mode dcb`
contexts.

### `invariants`

Declares the business rules the enclosing scope keeps true:

```emod
invariants {
  OneCopyPerLoan = "A loan covers exactly one copy of one title"
}
```

In aggregate mode the block sits on the `aggregate`; a `mode dcb` context has no aggregate, so
its block sits on the context.

## Slices and patterns

A slice is an implementation unit representing one use case. All event-modeling primitives live
inside slices.

```emod
slice "Reserve a Room" {
  trigger     "Reservation Form" { ... }  # 0-1 trigger (Command Pattern)
  command     "ReserveRoom" { ... }        # 0+ commands
  event       "RoomReserved" { ... }       # 0+ events
  view        "AvailableRoomsView" { ... } # 0+ views (View Pattern)
  automation  "ConfirmationEmailReactor" { ... }  # 0+ automations
  translation "ExternalEvents" { ... }     # 0+ translations
  spec        "reserves a room" { ... }    # 0+ Given-When-Then specs
}
```

- A **trigger** names the actor who works the slice, and the view they read.
- A **command** states the input fields a use case accepts.
- An **event** states the facts the use case appends to the stream.
- A **view** projects events; it declares which events it subscribes to.
- An **automation** reacts to an event, reads a view, and issues a command (optionally into
  another context via `target`).
- A **translation** adapts events from outside the model into events the model owns.
- A **spec** is a Given-When-Then statement of behaviour.

### Fields

```emod
fields {
  reservationId = required(string)
  notes         = optional(string)
}
```

`required(<type>)` means the field must be present; `optional(<type>)` means it may be absent.

## DCB models

For cross-cutting consistency boundaries, `mode = dcb` declares slices directly under a context,
with tagged events and tag-scoped decision queries:

```emod
context "Fulfillment" {
  mode = dcb

  slice "Place Order" {
    command "PlaceOrder" { ... }

    event "OrderPlaced" {
      tags {
        entity = customerId
      }

      fields {
        orderId    = string
        customerId = string
      }
    }

    flow = <<-FLOW
      command -> event: PlaceOrder -> OrderPlaced
    FLOW
  }

  slice "Authorize Payment" {
    command "AuthorizePayment" {
      decides_on {
        events = [OrderPlaced]
        where  = tag(entity, customerId)
      }

      fields { ... }
    }
  }
}
```

See [`examples/dcb_model.emod`](https://github.com/hpcsc/emod/blob/main/examples/dcb_model.emod)
for a complete example.

## Further reading

- The complete reference: [`docs/dsl-reference.md`](https://github.com/hpcsc/emod/blob/main/docs/dsl-reference.md)
- How the repository fits together: [`docs/architecture.md`](https://github.com/hpcsc/emod/blob/main/docs/architecture.md)

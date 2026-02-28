# emod — Event Modeling DSL & Visualization Tool

## 1. Problem

Event modeling today is a manual, visual-first process (Miro/draw.io). This creates friction when:

- Agents need to produce or consume event models programmatically
- Models need to be version-controlled, diffed, reviewed in PRs
- Diagrams need to stay in sync with the textual model
- Models need to be validated against known anti-patterns

## 2. DSL Design: `.emod` files

The DSL is declarative, human-readable, and maps 1:1 to event modeling concepts.

Two options explored: a custom `.emod` syntax and a CUE-based alternative.

### Option A: Custom `.emod` DSL (readable, purpose-built)

```emod
model "Hotel Reservation" {

  # --- Actors ---
  actor Guest
  actor FrontDesk
  actor System

  # --- Bounded Context ---
  context Reservations {

    # --- Aggregates ---
    aggregate RoomReservation {
      stream "RoomReservation-{reservationId}"
    }

    # --- Slice 1: Reserve a Room (Command Pattern) ---
    slice "Reserve a Room" {
      trigger UI "Reservation Form" {
        actor Guest
        reads AvailableRoomsView
      }

      command ReserveRoom {
        actor Guest
        aggregate RoomReservation
        fields {
          roomId       RoomID       required
          guestName    String       required
          checkIn      Date         required
          checkOut     Date         required
          guestCount   Int          required
        }
      }

      event RoomReserved {
        aggregate RoomReservation
        fields {
          reservationId  ReservationID
          roomId         RoomID
          guestName      String
          checkIn        Date
          checkOut        Date
          guestCount     Int
          reservedAt     Timestamp
        }
      }

      command -> event: ReserveRoom -> RoomReserved
    }

    # --- Slice 2: View Reservations (View Pattern) ---
    slice "View Available Rooms" {
      view AvailableRoomsView {
        fields {
          roomId       RoomID
          roomNumber   String
          status       RoomStatus
          nextCheckIn  Date?
        }
        subscribes [RoomReserved, GuestCheckedOut]
      }
    }

    # --- Slice 3: Send Confirmation (Automation Pattern) ---
    slice "Send Confirmation Email" {
      automation ConfirmationEmailReactor {
        trigger   RoomReserved
        command   SendConfirmationEmail
        target    context Notifications
      }
    }

    # --- Slice 4: Import from Booking.com (Translation Pattern) ---
    slice "Import External Booking" {
      translation BookingComImport {
        external_system "Booking.com API"
        reads BookingComWebhookView
        command ImportExternalReservation
        event ExternalReservationImported {
          fields {
            reservationId   ReservationID
            externalRef     String
            source          "booking.com"
            roomId          RoomID
            guestName       String
            checkIn         Date
            checkOut        Date
          }
        }
      }
    }
  }

  context Notifications {
    aggregate CustomerEngagement {
      stream "CustomerEngagement-{engagementId}"
    }

    slice "Handle Confirmation Email" {
      command SendConfirmationEmail {
        aggregate CustomerEngagement
        fields {
          reservationId  ReservationID
          guestEmail     Email
        }
      }

      event ConfirmationEmailRequested {
        aggregate CustomerEngagement
        fields {
          engagementId   EngagementID
          reservationId  ReservationID
          guestEmail     Email
        }
      }

      # External provider callback
      event ConfirmationEmailDelivered {
        aggregate CustomerEngagement
        source external "SendGrid Webhook"
        fields {
          engagementId   EngagementID
          deliveredAt    Timestamp
        }
      }
    }
  }
}
```

### Option B: CUE-based DSL (leveraging existing tooling)

```cue
package emod

#Model: {
  name:     string
  actors:   [...#Actor]
  contexts: [string]: #BoundedContext
}

#Actor: {
  name: string
  type: "human" | "system" | "external"
}

#BoundedContext: {
  aggregates: [string]: #Aggregate
  slices:     [...#Slice]
}

#Aggregate: {
  stream_pattern: string  // e.g. "Order-{orderId}"
}

#Slice: {
  name:    string
  pattern: #CommandPattern | #ViewPattern | #AutomationPattern | #TranslationPattern
}

#CommandPattern: {
  type: "command"
  trigger: #Trigger
  command: #Command
  events:  [...#Event]
}

#ViewPattern: {
  type: "view"
  view: #View
}

#AutomationPattern: {
  type: "automation"
  reactor: #Reactor
}

#TranslationPattern: {
  type: "translation"
  external_system: string
  inbound_view:    string
  command:         #Command
  events:          [...#Event]
}

#Command: {
  name:      =~"^[A-Z][a-zA-Z]+$"           // PascalCase, imperative
  aggregate: string
  actor?:    string
  fields:    [string]: #Field
}

#Event: {
  name:      =~"^[A-Z][a-zA-Z]+(ed|en|te)$" // past tense hint
  aggregate: string
  source?:   "internal" | {external: string}
  fields:    [string]: #Field
}

#View: {
  name:       =~"^[A-Z][a-zA-Z]+View$"
  fields:     [string]: #Field
  subscribes: [...string]  // event names
}

#Trigger: {
  type:   "ui" | "api" | "cron" | "webhook"
  name:   string
  actor?: string
  reads?: [...string]  // view names
}

#Reactor: {
  name:           string
  listens_to:     [...string]  // event names
  emits_command:  string
  target_context?: string
}

#Field: {
  type:     string
  required: bool | *true
}
```

### Comparison

| Consideration | Custom `.emod` | Pure CUE |
|---|---|---|
| Readability for non-devs | High — domain-focused | Medium — schema syntax |
| Agent-friendliness | High — clear grammar | High — structured |
| Validation | Need to build | Built-in via CUE constraints |
| Extensibility | Full control | CUE ecosystem |
| Tooling cost | Parser needed | CUE tooling exists |

### Recommendation: Hybrid

Write models in `.emod` custom syntax, use CUE as the intermediate validated representation. The parser converts `.emod` -> CUE -> validated model -> diagram output.

## 3. Application Architecture

```
                   ┌─────────────────────────┐
                   │     .emod DSL files      │  ← Human / Agent authored
                   └────────────┬─────────────┘
                                │ parse
                   ┌────────────▼─────────────┐
                   │    AST / Internal Model   │  ← Structured representation
                   └────────────┬─────────────┘
                                │
             ┌──────────────────┼──────────────────┐
             │                  │                   │
    ┌────────▼────────┐ ┌──────▼───────┐  ┌───────▼────────┐
    │   Validator      │ │  Diagram Gen │  │  Code Gen      │
    │ (anti-patterns,  │ │              │  │  (future)      │
    │  completeness)   │ │ - draw.io    │  │ - Go structs   │
    └─────────────────┘ │ - SVG        │  │ - aggregate    │
                        │ - Mermaid    │  │   skeletons    │
                        │ - terminal   │  └────────────────┘
                        └──────────────┘
```

### CLI: `emod`

```
emod validate reservation.emod         # Validate model + anti-pattern check
emod diagram  reservation.emod         # Generate draw.io XML
emod diagram  reservation.emod -f svg  # Generate SVG
emod diagram  reservation.emod -f mermaid  # Mermaid for markdown
emod lint     reservation.emod         # Anti-pattern detection
emod fmt      reservation.emod         # Format/normalize
emod slices   reservation.emod         # List implementation slices
emod export   reservation.emod -f json # JSON for other tools
```

## 4. Built-in Validations

Derived from event modeling anti-pattern guidelines.

| Rule | Checks |
|---|---|
| No state obsession | Event names don't end in `Updated/Changed/Modified` |
| No property sourcing | No `[Entity][Field]Changed` pattern |
| No clickbait events | Event payloads have more than just an ID |
| No `...Initiated` events | Events don't end in `Initiated` |
| Single event per decision | Commands don't fan out to 3+ events (left chair) |
| No god views | Views don't subscribe to 5+ events (right chair) |
| No direct event chains | Events connect to views or reactors, never directly to events |
| Business stakeholder test | Warning for overly technical names |
| Past tense events | Events must use past tense |
| Imperative commands | Commands must use imperative form |

## 5. Implementation Language: Go

- Single binary distribution (CLI)
- Fast parsing/generation
- Good ecosystem for template rendering, graph layout
- Pairs well with CUE (CUE is written in Go, has Go SDK)

## 6. Phased Implementation

### Phase 1: Core DSL + Parser + Validator

- Define grammar (tree-sitter or hand-written recursive descent)
- Parse `.emod` -> internal model
- Anti-pattern lint rules
- `emod validate`, `emod lint`, `emod fmt`

### Phase 2: Diagram Generation

- Draw.io XML output (using existing templates from guidelines)
- Mermaid output (for embedding in docs/PRs)
- Terminal ASCII preview

### Phase 3: Agent Integration

- JSON/CUE import/export for agent consumption
- Claude skill that outputs `.emod` files directly
- Round-trip: agent reads `.emod`, proposes changes, writes `.emod`

### Phase 4 (future): Code Generation + Desktop App

- Generate aggregate/command/event Go structs from model
- Electron/Tauri desktop app for interactive editing with live diagram preview

## 7. Why Not Mermaid/PlantUML?

General diagramming tools lack:

- **Domain semantics** — they don't know what an "event" or "command" is
- **Validation** — they can't check anti-patterns
- **Structure** — they're layout languages, not modeling languages
- **Agent-friendliness** — too much visual noise for structured agent output

The `.emod` DSL is to event modeling what Terraform is to infrastructure — a domain-specific language that understands the domain and can validate, generate, and evolve models programmatically.

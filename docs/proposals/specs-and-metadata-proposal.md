# Specs, Invariants, and Model Metadata: Proposal

## Problem

An `.emod` file describes structure: which commands exist, which events they produce, which views project them. It cannot describe behaviour or meaning:

- **No specifications.** Event Modeling prescribes Given-When-Then specs per slice, but the DSL has no place for them. Teams keep specs in wiki pages or test files, where they drift from the model.
- **No business rules.** There is nowhere to state *why* a command would be rejected. The rule "a room can hold one active reservation" lives in comments or nowhere.
- **No error paths.** The timeline shows only success. A command that can be rejected renders identically to one that cannot, so the error handling discovered in workshops has nowhere to land.
- **No descriptions.** Constructs carry only a name. LSP hover, diagrams, and any future glossary have nothing to show beyond the identifier.
- **No version marker.** Files do not declare which revision of the grammar they target. The first breaking grammar change will fail on old files with a parse error instead of a clear versioning message.
- **No bridge to the wire.** Model events have no link to the event types actually published (CloudEvents types, schema registry subjects), so JSON/CUE exports stop at the model boundary.
- **No time-driven behaviour.** Automations react to events only. "Release the hold 24 hours after `RoomHeld`" cannot be expressed.

ESDM (esdm.io) addresses most of these in its YAML-based DSL: a Given-When-Then extension, named invariants, `description` on every kind, Kubernetes-style `apiVersion` pinning, CloudEvents type annotations, and process-manager timers. Emlang (emlang-project.github.io), another YAML-based Event Modeling DSL, supplies two further ideas: error outcomes drawn on the timeline as first-class elements, and Given-When-Then tests that carry example payload data. This proposal adopts the ideas, not the syntax: each lands as an addition to the existing `.emod` grammar.

## Goals

- Attach Given-When-Then specs to slices, covering all four slice patterns, with optional example payload data.
- Declare named invariants on aggregates and DCB contexts; let spec rejections reference them by name.
- Show invariant rejection paths in `flow` blocks and on diagrams.
- Add `description` to model constructs and generate a glossary from them.
- Pin files to a DSL version with a one-line header.
- Annotate events with their wire-level type.
- Trigger automations on elapsed time relative to an event.
- Keep every existing `.emod` file valid and unchanged in meaning.

## Non-Goals

- **Executing specs.** Specs are model artifacts. Exporting test skeletons is a later, separate feature; running them against an implementation is out of scope entirely.
- **Adopting ESDM's file layout.** YAML, one-artifact-per-file, and `scope` back-references are not adopted. The nested single-file grammar stays.
- **Structural DDD kinds.** No `entity`, `value-object`, `domain-service`, or `subdomain` constructs. emod models the Event Modeling timeline, not the object model.
- **Process-manager construct.** Event Modeling decomposes process managers into the view plus automation pattern, which the DSL already expresses. Only the timer trigger is adopted.
- **Context-mapping relationship types.** Labelling cross-context relationships (ACL, conformist, customer-supplier) is deferred until diagram grouping gives the labels somewhere useful to appear.

---

## 1. Version Header

A file may open with a version directive, before `model`:

```emod
emod 1
model "Hotel Reservation"
```

- The version is a single integer. Absence means `1`.
- `emod fmt` inserts the header, so formatted files are always pinned.
- Semantics follow the Kubernetes convention: additive grammar changes (new optional keywords) do not bump the version; breaking changes do. A parser that supports up to version N rejects `emod N+1` with a clear "unsupported version" diagnostic instead of a parse error deep in the file.

This is the cheapest item in the proposal and the most expensive to retrofit later.

## 2. Descriptions and Glossary

### `description` attribute

Block constructs accept an optional `description` string as their first attribute:

```emod
context "Reservations" {
  description "Owns the lifecycle of a reservation from booking to checkout."

  aggregate "Reservation" {
    slice "Reserve a Room" {
      command ReserveRoom {
        description "Guest books a specific room for a date range."
        fields { ... }
      }
    }
  }
}
```

Applies to: `context`, `aggregate`, `slice`, `command`, `event`, `view`, `automation`, `translation`, `trigger`.

Single-line declarations (`model`, `actor`) gain an optional block form for the same purpose:

```emod
actor "Guest" {
  description "A person booking a room. Not necessarily the person staying in it."
}
```

Field-level descriptions are deferred (see Open Questions).

### `emod glossary`

A new subcommand renders the ubiquitous language from the model:

```bash
emod glossary reservation.emod            # markdown to stdout
emod glossary reservation.emod -f json    # structured output
```

Output groups terms by context: aggregates, commands, events, views, and actors, each with its description. Constructs without a description appear with an empty definition, which makes gaps visible without a lint rule.

### Consumers

- **LSP hover** shows the description alongside the construct kind.
- **Diagrams** attach descriptions as tooltips (draw.io) and `<title>` elements (SVG).
- **Exports** (`json`, `cue`) carry descriptions through.

## 3. Named Invariants

Aggregates declare the rules they protect. Each invariant is an identifier plus a prose statement:

```emod
aggregate "Reservation" {
  invariant roomNotDoubleBooked "A room has at most one active reservation for any date range."
  invariant checkOutAfterCheckIn "Check-out must be after check-in."

  slice "Reserve a Room" { ... }
}
```

In DCB mode there is no aggregate, so invariants attach to the context:

```emod
context "Fulfillment" mode dcb {
  invariant onePaymentPerOrder "An order is authorized for payment at most once."
  slice "Authorize Payment" { ... }
}
```

Invariants do nothing on their own. They exist to be referenced: spec rejections name the invariant that fired (next section), and the glossary lists invariants under their aggregate. This mirrors the one structural insight worth copying from ESDM's Given-When-Then extension: a rejection is not free text, it is a reference to a declared rule, and the validator can check the reference.

## 4. Given-When-Then Specs

### Surface

A slice may contain any number of `spec` blocks:

```emod
slice "Reserve a Room" {
  command ReserveRoom { ... }
  event RoomReserved { ... }
  flow { command -> event: ReserveRoom -> RoomReserved }

  spec "reserves a free room" {
    given []
    when  ReserveRoom
    then  [RoomReserved]
  }

  spec "rejects a double booking" {
    given [RoomReserved]
    when  ReserveRoom
    then  rejected roomNotDoubleBooked
  }
}
```

- `given`: an ordered list of event names, the prior history. `[]` is the empty history and may be omitted entirely.
- `when`: the command under test.
- `then`: one of two outcomes.
  - `[EventA, EventB]`: the events appended on success.
  - `rejected <invariantName>`: the command fails; the name must resolve to an invariant on the enclosing aggregate (or context in DCB mode).

### Pattern variants

The four slice patterns each get a spec shape, following ESDM's four feature variants:

| Pattern | `given` | `when` | `then` |
|---|---|---|---|
| Command | event history | command | events or `rejected <invariant>` |
| View | event history | (omitted) | `view <ViewName>` |
| Automation | event history | event (the automation's trigger) | `command <CommandName>` |
| Translation | event history | command | events |

View and automation examples:

```emod
slice "View Available Rooms" {
  view AvailableRoomsView { ... }

  spec "a reserved room is not available" {
    given [RoomReserved]
    then  view AvailableRoomsView
  }
}

slice "Send Confirmation Email" {
  automation ConfirmationEmailReactor { ... }

  spec "reservation triggers a confirmation" {
    given []
    when  RoomReserved
    then  command SendConfirmationEmail
  }
}
```

### Payload literals

Any event or command reference in a spec may take a brace block of field/value pairs. Partial payloads are the point: a spec states only the fields the scenario turns on.

```emod
spec "rejects a double booking" {
  given [
    RoomReserved { roomId: "101", checkIn: "2026-08-01", checkOut: "2026-08-03" }
  ]
  when ReserveRoom { roomId: "101", checkIn: "2026-08-02", checkOut: "2026-08-04" }
  then rejected roomNotDoubleBooked
}
```

The repeated `roomId` and overlapping dates are what trip the invariant — the values carry the scenario's meaning, and the repetition is the documentation. Nothing links the values mechanically; there is no variable binding (see Open Questions).

`then` event lists accept payloads the same way: `then [RoomReserved { roomId: "101" }]`.

Literals come in three forms, checked against the referenced construct's field declarations:

| Literal | Example | Accepted by declared type |
|---|---|---|
| string | `"101"` | `string`; `date`, `timestamp`, `uuid` (the value must parse as that format) |
| number | `42`, `12.50` | `int` (no fractional part), `decimal` |
| bool | `true`, `false` | `bool` |

Domain types (`RoomID`) accept any literal unchecked — they are opaque to the model. There are no bare date literals: quoted strings validated by format give the same safety without new lexer states.

A names-only spec remains valid; payloads are additive per element reference.

### Validation

- Every event in `given` and `then` must be defined in the model.
- `when` must name a command (or event, for automation specs) defined in the model.
- `rejected <name>` must resolve to an invariant on the enclosing aggregate or DCB context.
- The `then` shape must match the slice pattern (a `view` outcome inside a command-pattern slice is an error).
- Payload field names must exist on the referenced construct's `fields` declaration.
- Payload literal kinds must match the declared field type per the table above.
- Required fields may be omitted from a payload — a spec is an example, not an instance.

### Linting

| Rule | Severity | Check |
|---|---|---|
| `spec/command-without-spec` | info | A command has no spec exercising it. |
| `spec/invariant-never-exercised` | warning | An invariant is declared but no spec ends in `rejected` with it. |
| `spec/no-rejection-path` | info | A command has specs but none exercises a rejection. Happy-path-only coverage. |
| `spec/given-outside-boundary` | warning | In aggregate mode, a `given` event belongs to a different aggregate. In DCB mode, a `given` event is not matched by the command's `decides_on`. |

`spec/given-outside-boundary` is the rule that pays for the feature: it catches specs that assume history the consistency boundary cannot see. Payload literals sharpen it in DCB mode: for a command with `where tag(entity = customerId)`, the rule can check not only that a `given` event's type is matched by `decides_on`, but that its tagged field value equals the `when` command's `customerId`. With payloads, specs stop being documentation and start being model-checking.

## 5. Rejection Paths in Flows

A `flow` block accepts a second entry kind: a rejection edge from a command to an invariant.

```emod
aggregate "Reservation" {
  invariant roomNotDoubleBooked "A room has at most one active reservation for any date range."

  slice "Reserve a Room" {
    command ReserveRoom { ... }
    event RoomReserved { ... }

    flow {
      command -> event:    ReserveRoom -> RoomReserved
      command -> rejected: ReserveRoom -> roomNotDoubleBooked
    }
  }
}
```

The entry reads like the existing production — left of `:` declares the kinds, right names them — except the target is an invariant, not an event.

The edge covers invariant rejections only: the command fails and nothing is appended. A failure the business cares about (a payment declined) is an event and already has a flow entry (`command -> event: AuthorizePayment -> PaymentDeclined`). Errors that carry meaning belong on the timeline as events; the rejection edge exists for the case where the timeline is otherwise silent.

- `emod validate` resolves the invariant name on the enclosing aggregate (or context in DCB mode), the same rule as `rejected` in specs.
- Diagrams render the edge dashed, ending in a rejection badge; the badge's tooltip (draw.io) or `<title>` (SVG) carries the invariant's prose.
- One lint rule accompanies the edge — `flow/rejection-without-spec` (info): the timeline claims a failure path no spec on the slice exercises. A rejection edge also counts as a reference for `spec/invariant-never-exercised`, which otherwise flags the invariant as unused.

## 6. Wire-Level Event Types

Events accept an optional `type` attribute binding the model event to its published representation:

```emod
event RoomReserved {
  type "com.acme.reservations.room-reserved"
  fields { ... }
}
```

- The value is an opaque string. A lint rule (`wire/type-format`, info) nudges toward reverse-DNS kebab-case, the CloudEvents convention, without enforcing it.
- `emod export -f json` and `-f cue` emit the wire type, which is the point: exports become usable as input to schema registries and code generation.
- `emod validate` errors on two events sharing the same wire type.

## 7. Timer Triggers for Automations

An automation's trigger accepts an optional `after` clause:

```emod
slice "Release Expired Hold" {
  automation ExpiredHoldReleaser {
    trigger RoomHeld after "24h"
    command ReleaseHold
  }
}
```

Reading: 24 hours after each `RoomHeld` occurrence, issue `ReleaseHold`.

- The duration is a string in Go duration syntax (`"30m"`, `"24h"`, `"72h"`).
- Without `after`, behaviour is unchanged: the automation reacts immediately.
- Diagrams render timed automations with a clock badge and the duration on the trigger edge.

How the timer is implemented (durable scheduling, delivery guarantees, idempotency) is a runtime concern and stays out of the model, the same line the DCB proposal drew for append-condition checking. Cron-style standalone schedules (`every day at 03:00`) are deferred (see Open Questions).

---

## Internal Representation

### AST (`internal/ast/ast.go`)

New nodes:

```go
type Spec struct {
    Comments []*Comment
    Name     string
    NamePos  Position
    Given    []*SpecElement
    When     *SpecElement
    Then     ThenClause
    OpenPos  Position
    ClosePos Position
}

// SpecElement references an event or command, with an optional example payload.
type SpecElement struct {
    Name    string
    NamePos Position
    Payload []*FieldValue // nil for a names-only reference
}

type FieldValue struct {
    Name     string
    NamePos  Position
    Value    Literal // one of: StringLit, NumberLit, BoolLit
    ValuePos Position
}

// ThenClause is one of: ThenEvents, ThenRejected, ThenView, ThenCommand.
// ThenEvents holds []*SpecElement, so success outcomes carry payloads too.
type ThenClause interface{ thenNode() }

type RejectionFlow struct {
    Comments      []*Comment
    CommandName   string
    CommandPos    Position
    InvariantName string
    InvariantPos  Position
}

type Invariant struct {
    Comments    []*Comment
    Name        string
    NamePos     Position
    Description string
    DescPos     Position
}
```

Extensions to existing nodes:

- `Model`: `Version int`, `VersionPos Position`.
- `Slice`: `Specs []*Spec`, `Rejections []*RejectionFlow`.
- `Aggregate`, `Context`: `Invariants []*Invariant`.
- `Event`: `WireType string`, `WireTypePos Position`.
- `Automation`: `TriggerAfter string`, `TriggerAfterPos Position`.
- `Description string` and `DescPos Position` on `Model`, `Actor`, `Context`, `Aggregate`, `Slice`, `Command`, `Event`, `View`, `Automation`, `Translation`, `Trigger`.

### Lexer (`internal/lexer`)

New keywords: `spec`, `given`, `when`, `then`, `rejected`, `invariant`, `description`, `type`, `after`, `emod`.

One new token kind: `Number` (`42`, `12.50`), for spec payload values. `true` and `false` are recognized in payload value position without becoming reserved words.

`type` and `description` are plausible field names. To avoid breaking `fields { type string required }`, the parser must accept keyword tokens in field-name position. This is a small, general fix: the field-row production takes any identifier-or-keyword token as the name. The same courtesy already-reserved words like `events` and `source` benefit retroactively.

### Parser (`internal/parser`)

- Optional `emod <int>` directive before `model`.
- Optional block form for `actor` and `model`.
- `description`, `invariant`, `spec`, `type`, `after` productions as described above.
- `then` disambiguates on its first token: `[` starts an event list, `rejected` an invariant reference, `view` a view outcome, `command` a command outcome.
- An optional `{ field: literal, ... }` payload block after any event or command reference inside a spec.
- A second `flow` entry production, disambiguated by the `rejected` keyword after the first `->`: `command -> rejected: <CommandName> -> <invariantName>`.

### Formatter (`internal/formatter` via `emod fmt`)

- Inserts the `emod 1` header when absent.
- Canonical attribute order inside blocks: `description`, then pattern-specific attributes, then `fields`, then `spec` blocks last.
- Aligns `given` / `when` / `then` keywords within a spec.
- Payloads stay on one line when they fit; otherwise one field per line with values aligned, the same convention as `fields` blocks.
- Aligns the `:` across `command -> event:` and `command -> rejected:` entries within a `flow` block.

### Validator and Linter

Covered per feature above. All new lint rules respect the existing severity and `--explain` machinery.

### LSP (`emod lsp`)

- Hover: descriptions on all constructs; invariant prose on `rejected <name>` references, in specs and in flow rejection edges alike.
- Completion: invariant names after `rejected`; event names inside `given [...]`; field names inside payload braces, scoped to the referenced construct's `fields`.
- Go-to-definition and find-references: spec event/command references, invariant references (including flow rejection edges).

### Diagrams

- Specs render as an optional GWT card under the slice (`--specs` flag; off by default to keep diagrams stable).
- Rejection edges render dashed, ending in a rejection badge whose tooltip/`<title>` carries the invariant's prose.
- Descriptions become tooltips (draw.io) and `<title>` elements (SVG).
- Timed automations get a clock badge.

---

## Worked Example

```emod
emod 1
model "Hotel Reservation"

actor "Guest" {
  description "A person booking a room."
}

context "Reservations" {
  description "Owns the reservation lifecycle from booking to checkout."

  aggregate "Reservation" {
    invariant roomNotDoubleBooked "A room has at most one active reservation for any date range."

    slice "Reserve a Room" {
      trigger UI "Reservation Form" {
        actor Guest
        reads AvailableRoomsView
      }

      command ReserveRoom {
        description "Guest books a specific room for a date range."
        fields {
          roomId    string required
          guestName string required
          checkIn   date   required
          checkOut  date   required
        }
      }

      event RoomReserved {
        type "com.acme.reservations.room-reserved"
        fields {
          reservationId string    required
          roomId        string    required
          checkIn       date      required
          checkOut      date      required
          reservedAt    timestamp required
        }
      }

      flow {
        command -> event:    ReserveRoom -> RoomReserved
        command -> rejected: ReserveRoom -> roomNotDoubleBooked
      }

      spec "reserves a free room" {
        when ReserveRoom { roomId: "204" }
        then [RoomReserved { roomId: "204" }]
      }

      spec "rejects a double booking" {
        given [
          RoomReserved { roomId: "101", checkIn: "2026-08-01", checkOut: "2026-08-03" }
        ]
        when ReserveRoom { roomId: "101", checkIn: "2026-08-02", checkOut: "2026-08-04" }
        then rejected roomNotDoubleBooked
      }
    }

    slice "Release Expired Hold" {
      automation ExpiredHoldReleaser {
        trigger RoomHeld after "24h"
        command ReleaseHold
      }

      spec "hold expires after a day" {
        given [RoomHeld]
        when  RoomHeld
        then  command ReleaseHold
      }
    }
  }
}
```

---

## Phased Implementation

### Phase 1: Metadata Foundation

Version header, `description` attribute, block forms for `actor` and `model`, keyword-as-field-name fix, `emod glossary`. Each piece is small and independently shippable; together they establish the attribute plumbing (AST fields, formatter ordering, export pass-through) that later phases reuse.

### Phase 2: Invariants and Specs

Grammar, AST, validation, and the four lint rules, with names-only specs. Rejection flow edges land here too: they need only the invariant resolution specs already require, and bring `flow/rejection-without-spec`. This is the bulk of the work and the bulk of the value.

### Phase 3: Payload Literals

The `Number` token, payload grammar, type checking against field declarations, formatter alignment, and the DCB value-check upgrade to `spec/given-outside-boundary`. Purely additive on Phase 2: every names-only spec stays valid.

### Phase 4: Wire Types and Timers

`type` on events with export support and uniqueness validation. `after` on automation triggers with diagram badge.

### Phase 5: Tooling and Docs

LSP hover/completion/references for the new constructs. `--specs` diagram flag. Tree-sitter grammar and VS Code highlighting for new keywords. Update `examples/all_patterns.emod`; add `examples/specs_hotel.emod`. Extend `docs/dsl-reference.md`.

---

## Risks and Open Questions

- **View-state payloads.** `then view AvailableRoomsView` names the view but cannot state its expected contents. Expected view state means modeling rows and collections — a much larger literal grammar than scalar payloads. Deferred until scalar payloads prove themselves.
- **Variable binding in specs.** Payload linkage is by repetition: the same `"101"` appears in `given` and `when`. A `let` binding would make the linkage mechanical, at the cost of a new scoping construct. At spec scale repetition reads fine; revisit only if real specs grow long enough for repetition to breed errors.
- **Spec bloat in large files.** Specs can dwarf the structure they describe, and payload literals raise the odds. If files grow unwieldy, an include mechanism (`specs "reservation_specs.emod"`) is the escape hatch. Not designed here; watch real usage first.
- **`description` versus doc comments.** `#` comments are already preserved in the AST, so a `## doc comment` convention was the alternative. The explicit attribute wins on discoverability, formatter canonicalisation, and export fidelity, at the cost of some ceremony. Revisit only if authors consistently reach for comments instead.
- **Keyword collisions.** Every new keyword is a word modellers might use as a field name. The keyword-as-field-name fix removes the worst of it, but names like a view called `Given` remain awkward. The lexer stays context-free; the parser absorbs the ambiguity.
- **Timer semantics stop at the model.** `after "24h"` says nothing about clock skew, missed timers, or redelivery. That is intentional, but teams may ask the model to say more (ESDM records `deliveryGuarantee` and `idempotency`). Hold the line: those are implementation properties, not model properties.
- **Cron-style schedules.** `trigger every "..."` for calendar-driven automations (nightly reconciliation) is a plausible follow-up to `after`. Deferred until a concrete model needs it, to avoid designing a schedule syntax speculatively.
- **Version header adoption.** Old files without the header stay valid forever under the "absence means 1" rule, so adoption is driven entirely by `emod fmt`. If a version 2 ever ships, unformatted files are the risk surface; a lint info rule (`version/missing-header`) can nudge without nagging.

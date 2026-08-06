# emod DSL Reference

The emod DSL is a human-readable, version-controllable text format for describing event-driven architectures using Event Modeling concepts. Files use the `.emod` extension.

- **Source**: `internal/lexer/`, `internal/parser/`
- **AST**: `internal/ast/ast.go`
- **Schema (CUE)**: `internal/cue/schema.cue`

---

## 1. General Syntax

### Comments

Single-line comments start with `#`. They are preserved in the AST and round-tripped through the formatter.

```emod
# This is a comment
model "My System"
```

### Identifiers

Unquoted PascalCase tokens used for names of commands, events, views, automations, translations, actors, invariants, and references.

```emod
command PlaceOrder       # PlaceOrder is an identifier
event  OrderPlaced       # OrderPlaced is an identifier
```

### Strings

Double-quoted values used for human-readable names (`model`, `actor`, `slice`, `trigger`, `external_system`, `source external`) and for `description` text.

```emod
model "Hotel Reservation"
```

### Block Delimiters

Curly braces `{ }` define hierarchical scopes. Square brackets `[ ]` delimit comma-separated lists (e.g., `subscribes`).

---

## 2. Version Header

A file may open with a version directive on its own line, before `model`. It pins the file to a revision of the grammar.

```
emod <n>
```

The version is a single integer.

```emod
emod 1
model "Hotel Reservation"
```

- **Absence means version 1:** a file with no header parses exactly as one declaring `emod 1`.
- **`emod fmt` inserts the header** as the first line, so formatted files are always pinned. A declared version is preserved; a file without a header is pinned to the version the tool supports.
- **An unsupported version is rejected by the parser**, so the diagnostic reports the declared version and the supported version on line 1 instead of a parse failure deep in the file:

  ```
  reservation.emod:1: unsupported version 2: this tool supports emod version 1
  ```

Version numbers follow the Kubernetes convention: additive grammar changes, such as new optional keywords, do not bump the version; breaking changes do. Adopting new optional syntax therefore never requires touching the header.

---

## 3. Top-Level Constructs

### `model`

Required root declaration. Names the system being modeled.

```
model "<name>"

model "<name>" {
  description "<text>"
}
```

Exactly one per file. The block form carries a [description](#10-descriptions) and accepts no other entry; an empty block is allowed. See [examples/all_patterns.emod](/examples/all_patterns.emod).

### `actor`

Declares a persona or external system that interacts with the model. Appears at the top level.

```
actor "<name>"

actor "<name>" {
  description "<text>"
}
```

Multiple actors are allowed. Actors are referenced by triggers (see [Slice Patterns → Command](#command-pattern)). The block form carries a [description](#10-descriptions) and accepts no other entry; an empty block is allowed.

---

## 4. Bounded Contexts

### `context`

Groups aggregates (or slices in DCB mode) under a bounded context.

```
context "<name>" {
  aggregate "<name>" { ... }
  ...
}
```

An optional `mode` clause declares the consistency boundary style:

| Mode | Description |
|------|-------------|
| `aggregate` (default) | Slices must live inside an `aggregate` block. Tags, `decides_on`, and direct slices produce lint warnings. |
| `dcb` | Slices live directly under the context (no `aggregate`). Events use `tags`, commands use `decides_on`. An `aggregate` block produces a lint warning. |
| `mixed` | Both aggregate-wrapped and direct slices are accepted. DCB constructs (`tags`, `decides_on`) are allowed without warnings. |

```
context "<name>" mode dcb {
  slice "<name>" { ... }
  ...
}
```

A model must contain at least one context. Context names are referenced by automations (see [`target context`](#automation-pattern)). See [examples/dcb_model.emod](/examples/dcb_model.emod) for a DCB-mode example.

### `aggregate`

Groups slices within a context. Represents a consistency boundary. Not used in `mode dcb` contexts.

```
aggregate "<name>" {
  slice "<name>" { ... }
  ...
}
```

Multiple aggregates per context, multiple slices per aggregate.

### `invariant`

Declares a business rule the enclosing scope keeps true. An invariant is an identifier and a quoted prose statement of the rule, written on one line.

```
invariant <Name> "<statement>"
```

In aggregate mode an invariant is declared on the `aggregate` whose consistency boundary keeps it:

```
aggregate "<name>" {
  invariant <Name> "<statement>"
  slice "<name>" { ... }
}
```

A `mode dcb` context has no aggregate, so its invariants are declared directly on the context:

```
context "<name>" mode dcb {
  invariant <Name> "<statement>"
  slice "<name>" { ... }
}
```

Any number per block, in any position among the block's other entries.

```emod
emod 1
model "Library Lending"

context "Lending" {
  aggregate "Loan" {
    invariant OneCopyPerLoan "A loan covers exactly one copy of one title"
    invariant FiveCopiesPerMember "A member holds at most five copies at one time"
    slice "Borrow a Copy" {
      command BorrowCopy {
        fields {
          memberId string required
          copyId   string required
        }
      }

      event CopyBorrowed {
        fields {
          loanId   string required
          memberId string required
          copyId   string required
        }
      }

      flow {
        command -> event: BorrowCopy -> CopyBorrowed
      }
    }
  }
}

context "Reading Room" mode dcb {
  invariant OneReaderPerDesk "A desk seats at most one reader at any moment"
  invariant OneDeskPerReader "A reader holds at most one desk for the length of a session"
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
        sessionId string required
        deskId    string required
        memberId  string required
      }
    }

    flow {
      command -> event: ClaimDesk -> DeskClaimed
    }
  }
}
```

- **A name is declared once per scope:** two invariants sharing a name in one scope is a validation error, reported on the second declaration:

  ```
  lending.emod:7: invariant "OneCopyPerLoan" is already declared in aggregate "Loan"
  ```

- **An aggregate and its context are separate scopes**, as are two sibling aggregates. The same identifier may be declared in each, and neither declaration hides the other.
- **An invariant nothing references is not an error:** no construct refers to an invariant, and neither `emod validate` nor `emod lint` reports one for going unmentioned.
- **`emod fmt` writes them at the top of the block**, after the `description` line and ahead of the block's aggregates and slices, one per line, with the statement written verbatim.
- **The glossary lists them under the scope that declares them:** `emod glossary` puts an "Invariants" group under each aggregate and each context declaring one, in declaration order, with the statement standing as the definition. `emod glossary --format json` carries the same grouping, pairing each name with a `description` key holding the statement.
- **Exports carry them:** `emod export --format json` and `emod export --format cue` emit an `invariants` list of `name` and `statement` on every aggregate and context that declares one; the key is absent where none was written. The bundled schema printed by `emod schema` declares it as an optional key on both definitions.
- **No diagram renders them:** a diagram shows elements and the arrows between them, so the drawio, SVG, mermaid and ASCII renderings of a model are identical whether or not it declares invariants.

---

## 5. Slices

A slice is an implementation unit representing a single use case in Event Modeling. All event-modeling primitives live inside slices. In aggregate mode, slices are nested inside an `aggregate`. In DCB mode, slices are direct children of the context.

```
slice "<name>" {
  trigger     ...     # 0-1 trigger (Command Pattern)
  command     ...     # 0+ commands
  event       ...     # 0+ events (may include tags block in DCB/mixed mode)
  view        ...     # 0+ views (View Pattern)
  automation  ...     # 0+ automations (Automation Pattern)
  translation ...     # 0+ translations (Translation Pattern)
  flow { ... }       # 0+ command→event wiring
  spec "<name>" { ... }  # 0+ Given-When-Then specs
}
```

Every slice must contain at least one element. See [Slice Patterns](#6-slice-patterns) for valid combinations.

---

## 6. Slice Patterns

The DSL supports four Event Modeling slice patterns. Each pattern prescribes which blocks appear together.

The two elements that start a chain belong to different chains. A `trigger` is the human entry point of the command chain: something outside the system — a person at a screen, a form, a call taken by an agent — initiates the slice. An `automation` is the processor of the automation chain: the system initiates the slice itself, with nobody at a screen. Neither stands in for the other, and a slice that needs both declares both.

### Command Pattern

A user or system action drives a command that produces an event.

```
slice "<name>" {
  trigger "<name>" {
    actor <ActorName>
    reads <ViewName>
  }

  command <CommandName> {
    fields { ... }
  }

  event <EventName> {
    fields { ... }
  }

  flow {
    command -> event: <CommandName> -> <EventName>
  }
}
```

`flow` wires the command to the event (see [Flows](#7-flows)). `trigger` is optional — a slice may define a command directly without a trigger.

### View Pattern

A read model projects from subscribed events.

```
slice "<name>" {
  view <ViewName> {
    fields { ... }
    subscribes [<EventName>, <EventName>, ...]
  }
}
```

`subscribes` references event names defined elsewhere in the model (see [Cross-References](#11-cross-references)).

### Automation Pattern

A processor woken by an event or by a schedule reads its outstanding work and issues a command, possibly in a different context.

```
slice "<name>" {
  automation <Name> {
    on <EventName>       # activation — exactly one of on and every
    every "<expr>"       # activation — exactly one of on and every
    reads <ViewName>
    command <CommandName>
    target context <ContextName>
  }
}
```

- `on`: the event whose arrival wakes the processor. The name must be an event declared in the model.
- `every`: the schedule on which the processor wakes instead. The quoted expression is either a Go duration (`"5m"`, `"1h"`) for a fixed interval or a five-field cron expression (`"0 2 * * *"`) for a wall-clock schedule. Only the shape is checked; an expression that is neither is a validation error naming both forms:

  ```
  reservation.emod:34: schedule expression "nightly" is neither a Go duration nor a five-field cron expression
  ```

- `reads`: the todo list — the view holding the work the processor has left to do. Optional. The name must resolve to a view declared **anywhere in the model** — in any slice, under any aggregate or context — and `emod validate` reports one that resolves to nothing. That resolution is what separates an automation's `reads` from a trigger's and a translation's, which are recorded and left unchecked.
- `command`: the command the automation issues. Required.
- `target context`: a context name (cross-reference, validated at `emod validate`).

An automation states **exactly one** of `on` and `every`: declaring neither is an error, and declaring both is an error. Requiring the choice makes the wake-up explicit, so the model says whether the processor is woken by the event that adds to its todo list or polls on a cadence — two designs with different failure modes.

### Translation Pattern

An external system integration: reads a view, issues a command, produces an event.

```
slice "<name>" {
  command <CommandName> {
    fields { ... }
  }

  translation <Name> {
    external_system "<name>"
    reads <ViewName>
    command <CommandName>
    event <EventName> {
      fields { ... }
    }
  }
}
```

`external_system`, `reads`, `command`, and `event` are required.

### `spec`

A `spec` states expected behaviour as Given-When-Then, next to the structure it describes. A slice may hold any number of specs, in any position among its other entries.

```
slice "Borrow a Copy" {
  command BorrowCopy { fields { copyId string required } }
  event CopyBorrowed { fields { copyId string required } }

  spec "borrows a free copy" {
    given []
    when BorrowCopy
    then [CopyBorrowed]
  }
}
```

`given` is the history the command decides against, as an ordered event list. `given []` and omitting `given` entirely mean the same empty history:

```
spec "borrows a free copy" {
  when BorrowCopy
  then [CopyBorrowed]
}
```

`when` names the command under test. `then` states the outcome, in one of two shapes. A bracketed event list is the success outcome — the events appended, in order:

```
then [CopyBorrowed, LoanOpened]
```

`then rejected <invariantName>` is the failure outcome: the command is refused and nothing is appended. The name refers to a declared [`invariant`](#invariant), so a rejection is a checkable reference to a stated rule rather than free text:

```
spec "rejects a second borrow" {
  given [CopyBorrowed]
  when BorrowCopy
  then rejected OneCopyPerLoan
}
```

**Name resolution.** Every event in `given` and in a `then` list, and the command in `when`, must be defined somewhere in the model. A `rejected` name must be declared on the enclosing aggregate or, for a slice declared directly on a context, on that context — an aggregate and its context are separate scopes, so a name declared one level up does not resolve.

Specs are carried through `emod fmt`, the JSON and CUE exports, and the embedded schema.

The spec shapes for view, automation, and translation slices — `then view <Name>` and `then command <Name>` — are not part of the language yet.

---

## 7. Flows

`flow` declares a causal relationship: a command produces an event.

```
flow {
  command -> event: <CommandName> -> <EventName>
}
```

The `->` arrow is a required operator. Multiple flow entries may appear in one `flow { }` block. Flow declaration is optional for patterns where the wiring is implicit (automation already shows the trigger→command link; translation shows the command→event link via the nested event).

---

## 8. Fields

Commands, events, and views contain structured field definitions.

```
fields {
  <name> <type> [modifier]
}
```

- **name**: Any identifier, including any word the DSL uses as a keyword. The examples in this document use lowerCamelCase.
- **type**: Any identifier (`string`, `date`, `timestamp`, `int`, domain types like `RoomID`, etc.).
- **modifier** (optional): `required` (default behavior) or `optional`.

The formatter alignment pads the name and type columns.

### Keywords as Field Names

The DSL has no reserved words. A keyword carries its meaning only in the position that expects it. Every keyword is therefore usable as a field name, as a type and as a modifier. A model whose domain has a `source`, a `description` or a `tags` attribute names the field after it:

```emod
emod 1
model "Hotel Reservation"

context "Reservations" {
  aggregate "Reservation" {
    slice "Reserve a Room" {
      command ReserveRoom {
        fields {
          roomId      string required
          source      string required
          description string required
        }
      }

      event RoomReserved {
        fields {
          reservationId string required
          tags          string required
        }
      }

      flow {
        command -> event: ReserveRoom -> RoomReserved
      }
    }
  }
}
```

- **A new keyword never invalidates an existing field name:** a new keyword is an additive grammar change (see [Version Header](#2-version-header)). A field named after a word that the DSL later takes as a keyword keeps its name and its meaning.
- **The guarantee covers fields, not construct names:** the name of a command, event, view, automation or translation must be an identifier that is not a keyword. `command source { }` is rejected with `expected identifier after command`.

### External Source Events

An event can declare it originates from an external system:

```
event <Name> {
  source external "<Provider Name>"
  fields { ... }
}
```

---

## 9. Dynamic Consistency Boundaries

DCB mode is additive — aggregate-based models continue to work without changes. DCB constructs are valid in `dcb` and `mixed` modes only.

### Event Tags

Events in a DCB slice declare tags that cross-reference event fields. Tags allow commands to express consistency boundaries over event data.

```
event <Name> {
  tags {
    <key>: <fieldRef>
    ...
  }
  fields { ... }
}
```

- **key**: An identifier naming the tag dimension (e.g., `entity`, `category`, `region`).
- **fieldRef**: A field name declared on this event.
- Tags are validated at `emod validate`: every `fieldRef` must match a declared field.

### Command decides_on

Commands declare a consistency boundary by listing the event types they depend on and a predicate over tag values.

```
command <Name> {
  decides_on {
    events [<EventName>, ...]
    where <Predicate>
  }
  fields { ... }
}
```

- **events**: One or more event names defined elsewhere in the model.
- **where**: A predicate expression. Optional — if omitted, the command matches any event.

### Predicate Expressions

Predicates filter events by comparing tag values to event fields:

| Expression | Syntax | Example |
|---|---|---|
| Tag equality | `tag(<key> = <fieldRef>)` | `tag(entity = customerId)` |
| Logical AND | `<expr> and <expr>` | `tag(entity = id) and tag(region = country)` |
| Logical OR | `<expr> or <expr>` | `tag(status = active) or tag(status = trial)` |
| Negation | `not <expr>` | `not tag(category = premium)` |
| Grouping | `( <expr> )` | `(tag(a = x) or tag(a = y)) and tag(b = z)` |

The tag key and field reference in each `tag()` term are validated at `emod validate`:
- The **tag key** must be declared on at least one event listed in `events`.
- The **field reference** must be a declared field on at least one listed event.

### Example

See [examples/dcb_model.emod](/examples/dcb_model.emod) for a complete DCB-mode model demonstrating tags, `decides_on`, compound predicates, and direct slices.

---

## 10. Descriptions

Any construct that has a block may carry a `description`: prose explaining what the construct is for.

```
description "<text>"
```

`context`, `aggregate`, `slice`, `trigger`, `command`, `event`, `view`, `automation` and `translation` accept it, including the `event` nested inside a `translation`. `model` and `actor` accept it in their block form (see [Top-Level Constructs](#3-top-level-constructs)).

```emod
emod 1
model "Hotel Reservation" {
  description "Room inventory, bookings and check-out."
}

actor "Guest" {
  description "A person who books and stays in a room."
}

context "Reservations" {
  description "Owns the booking lifecycle."
  aggregate "Reservation" {
    description "One booking, from request to check-out."
    slice "Reserve a Room" {
      description "A guest holds a room for a date range."
      trigger "Reservation Form" {
        description "The public booking form."
        actor Guest
      }

      command ReserveRoom {
        description "Holds a room for the requested dates."
        fields {
          roomId    string required
          guestName string required
        }
      }

      event RoomReserved {
        description "A room is held for a guest."
        fields {
          reservationId string required
          roomId        string required
        }
      }

      flow {
        command -> event: ReserveRoom -> RoomReserved
      }
    }
  }
}
```

- **A description is documentation, not structure:** it is optional on every construct, nothing in the model refers to it, and no validation or lint rule reads it.
- **`description` is not a reserved word:** `fields { description string required }` still declares an ordinary field named `description`. The same holds for every keyword (see [Fields](#8-fields)).
- **Position inside the block is free:** the parser accepts `description` before or after the construct's other entries.
- **`emod fmt` moves it to the first line of the block**, ahead of the pattern-specific attributes and `fields`. A `model` or `actor` that carries a description is formatted in its block form.
- **Exports carry the text:** `emod export --format json` and `emod export --format cue` emit a `description` key on the model, on each actor and on each described construct; the key is absent where no description was written. The bundled schema printed by `emod schema` declares it as an optional key on every definition that accepts one.
- **Diagrams surface it on the shape:** `emod diagram --format drawio` attaches the description to the construct's shape as a tooltip, and `emod diagram --format svg` emits it as a `<title>` element inside the shape, which browsers show on hover.
- **A construct without a shape stays off the diagrams:** `model`, `actor`, `aggregate` and `slice` own no shape in either renderer. The `mermaid` and `ascii` formats carry no descriptions at all.
- **The glossary reads it as the definition:** `emod glossary` prints the description beneath the name it defines — the model, every actor, and each context with its aggregates and the commands, events and views its slices declare. `emod glossary --format json` pairs every name with a `description` key that stays present even where no description was written.
- **A construct that defines no term stays out of the glossary:** `slice`, `trigger`, `automation` and `translation` contribute no term of their own in either format, so their descriptions do not reach it.

---

## 11. Cross-References

Names are resolved during validation (`emod validate`). All references use unqualified names.

| Declaration | Referenced By | Context |
|---|---|---|
| `context "<name>"` | `automation { target context <Name> }` | [`automation`](#automation-pattern) |
| `event <Name>` | `subscribes [<Name>]`, `flow`, `automation { trigger <Name> }`, `automation { command <Name> }`, `translation { event <Name> }`, `translation { command <Name> }`, `spec { given [<Name>] }`, `spec { then [<Name>] }` | [`view`](#view-pattern), [`automation`](#automation-pattern), [`translation`](#translation-pattern), [`spec`](#spec) |
| `command <Name>` | `flow`, `automation { command <Name> }`, `translation { command <Name> }`, `spec { when <Name> }` | [`flow`](#7-flows), [`automation`](#automation-pattern), [`translation`](#translation-pattern), [`spec`](#spec) |
| `view <Name>` | `trigger { reads <Name> }`, `translation { reads <Name> }` | [`command` pattern](#command-pattern), [`translation`](#translation-pattern) |
| `actor "<name>"` | `trigger { actor <Name> }` | [`command` pattern](#command-pattern) |
| `invariant <name>` | `spec { then rejected <name> }` | [`spec`](#spec) |

Validation detects:
- Missing target contexts, commands, events, or views.
- **Orphan commands**: defined but never referenced by any flow, automation, or translation.
- **Orphan events**: defined but never produced by any flow, external source, or translation.
- **Redeclared invariants**: one name declared twice in a single scope (see [Bounded Contexts](#4-bounded-contexts)).
- **Undefined spec references**: an event in `given` or `then`, or a command in `when`, that the model does not define — reported as `event "<Name>" does not exist` or `command "<Name>" does not exist`.
- **Unresolved rejections**: a `then rejected <name>` whose invariant is not declared in the enclosing scope — reported as `invariant "<name>" is not declared in <scope> "<Name>"`.

---

## 12. Pipeline

The CLI processes `.emod` files through a linear pipeline:

```
.emod file
  → Scanner/Lexer       (internal/lexer/tokenizer.go)
  → Token stream
  → Recursive-descent Parser  (internal/parser/parser.go)
  → AST                  (internal/ast/ast.go)
  → Validator            (internal/validator/validator.go)
  → Linter               (internal/linter/linter.go)
  → Formatter / Exporter / Diagram Generator
```

Each stage preserves source position (file, line, column) for error reporting.

## 13. Diagram Palette

All diagram renderers use the same palette for element types. The SVG, draw.io, and web viewer renderers all draw the same element type with the same fill and stroke, and the DSL reference itself is the source of truth for those values.

| Element     | Fill      | Stroke    | Notes                                   |
|-------------|-----------|-----------|-----------------------------------------|
| Trigger     | #ffffff   | #333333   | Drawn as a screen/monitor shape.        |
| Command     | #dae8fc   | #6c8ebf   | Blue sticky note.                       |
| Event       | #ffe6cc   | #d79b00   | Orange sticky note.                     |
| View        | #d5e8d4   | #82b366   | Green sticky note.                      |
| Automation  | #e1d5e7   | #9673a6   | Purple processor.                       |
| Translation | #f5f5f5   | #666666   | Grey integration.                       |



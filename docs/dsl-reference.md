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

Unquoted PascalCase tokens used for names of commands, events, views, automations, translations, actors, and references.

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
}
```

Every slice must contain at least one element. See [Slice Patterns](#6-slice-patterns) for valid combinations.

---

## 6. Slice Patterns

The DSL supports four Event Modeling slice patterns. Each pattern prescribes which blocks appear together.

### Command Pattern

A user or system action drives a command that produces an event.

```
slice "<name>" {
  trigger <Kind> "<name>" {
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

An event triggers a reactive processor that issues a command, possibly in a different context.

```
slice "<name>" {
  automation <Name> {
    trigger <EventName>
    command <CommandName>
    target context <ContextName>
  }
}
```

- `trigger`: the event that activates this automation.
- `command`: the command the automation issues.
- `target context`: a context name (cross-reference, validated at `emod validate`).

`trigger` and `command` are required.

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
      trigger UI "Reservation Form" {
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
| `event <Name>` | `subscribes [<Name>]`, `flow`, `automation { trigger <Name> }`, `automation { command <Name> }`, `translation { event <Name> }`, `translation { command <Name> }` | [`view`](#view-pattern), [`automation`](#automation-pattern), [`translation`](#translation-pattern) |
| `command <Name>` | `flow`, `automation { command <Name> }`, `translation { command <Name> }` | [`flow`](#7-flows), [`automation`](#automation-pattern), [`translation`](#translation-pattern) |
| `view <Name>` | `trigger { reads <Name> }`, `translation { reads <Name> }` | [`command` pattern](#command-pattern), [`translation`](#translation-pattern) |
| `actor "<name>"` | `trigger { actor <Name> }` | [`command` pattern](#command-pattern) |

Validation detects:
- Missing target contexts, commands, events, or views.
- **Orphan commands**: defined but never referenced by any flow, automation, or translation.
- **Orphan events**: defined but never produced by any flow, external source, or translation.

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


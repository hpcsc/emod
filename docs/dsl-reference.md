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

Double-quoted values used for human-readable names (`model`, `actor`, `slice`, `trigger`, `external_system`, `source external`).

```emod
model "Hotel Reservation"
```

### Block Delimiters

Curly braces `{ }` define hierarchical scopes. Square brackets `[ ]` delimit comma-separated lists (e.g., `subscribes`).

---

## 2. Top-Level Constructs

### `model`

Required root declaration. Names the system being modeled.

```
model "<name>"
```

Exactly one per file. No braces — it is a single-line declaration. See [examples/all_patterns.emod](/examples/all_patterns.emod).

### `actor`

Declares a persona or external system that interacts with the model. Appears at the top level.

```
actor "<name>"
```

Multiple actors are allowed. Actors are referenced by triggers (see [Slice Patterns → Command](#command-pattern)).

---

## 3. Bounded Contexts

### `context`

Groups aggregates under a bounded context.

```
context "<name>" {
  aggregate "<name>" { ... }
  ...
}
```

A model must contain at least one context. Context names are referenced by automations (see [`target context`](#automation-pattern)).

### `aggregate`

Groups slices within a context. Represents a consistency boundary.

```
aggregate "<name>" {
  slice "<name>" { ... }
  ...
}
```

Multiple aggregates per context, multiple slices per aggregate.

---

## 4. Slices

A slice is an implementation unit representing a single use case in Event Modeling. All event-modeling primitives live inside slices.

```
slice "<name>" {
  trigger   ...       # 0-1 trigger (Command Pattern)
  command   ...       # 0+ commands
  event     ...       # 0+ events
  view      ...       # 0+ views (View Pattern)
  automation ...      # 0+ automations (Automation Pattern)
  translation ...     # 0+ translations (Translation Pattern)
  flow { ... }       # 0+ command→event wiring
}
```

Every slice must contain at least one element. See [Slice Patterns](#5-slice-patterns) for valid combinations.

---

## 5. Slice Patterns

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

`flow` wires the command to the event (see [Flows](#6-flows)). `trigger` is optional — a slice may define a command directly without a trigger.

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

`subscribes` references event names defined elsewhere in the model (see [Cross-References](#7-cross-references)).

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

## 6. Flows

`flow` declares a causal relationship: a command produces an event.

```
flow {
  command -> event: <CommandName> -> <EventName>
}
```

The `->` arrow is a required operator. Multiple flow entries may appear in one `flow { }` block. Flow declaration is optional for patterns where the wiring is implicit (automation already shows the trigger→command link; translation shows the command→event link via the nested event).

---

## 7. Fields

Commands, events, and views contain structured field definitions.

```
fields {
  <name> <type> [modifier]
}
```

- **name**: PascalCase identifier.
- **type**: Any identifier (`string`, `date`, `timestamp`, `int`, domain types like `RoomID`, etc.).
- **modifier** (optional): `required` (default behavior) or `optional`.

The formatter alignment pads the name and type columns.

### External Source Events

An event can declare it originates from an external system:

```
event <Name> {
  source external "<Provider Name>"
  fields { ... }
}
```

---

## 8. Cross-References

Names are resolved during validation (`emod validate`). All references use unqualified names.

| Declaration | Referenced By | Context |
|---|---|---|
| `context "<name>"` | `automation { target context <Name> }` | [`automation`](#automation-pattern) |
| `event <Name>` | `subscribes [<Name>]`, `flow`, `automation { trigger <Name> }`, `automation { command <Name> }`, `translation { event <Name> }`, `translation { command <Name> }` | [`view`](#view-pattern), [`automation`](#automation-pattern), [`translation`](#translation-pattern) |
| `command <Name>` | `flow`, `automation { command <Name> }`, `translation { command <Name> }` | [`flow`](#6-flows), [`automation`](#automation-pattern), [`translation`](#translation-pattern) |
| `view <Name>` | `trigger { reads <Name> }`, `translation { reads <Name> }` | [`command` pattern](#command-pattern), [`translation`](#translation-pattern) |
| `actor "<name>"` | `trigger { actor <Name> }` | [`command` pattern](#command-pattern) |

Validation detects:
- Missing target contexts, commands, events, or views.
- **Orphan commands**: defined but never referenced by any flow, automation, or translation.
- **Orphan events**: defined but never produced by any flow, external source, or translation.

---

## 9. Pipeline

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


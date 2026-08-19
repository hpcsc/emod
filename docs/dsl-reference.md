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

```
command PlaceOrder       # PlaceOrder is an identifier
event  OrderPlaced       # OrderPlaced is an identifier
```

### Strings

Double-quoted values. Most carry a human-readable name (`model`, `actor`, `slice`, `trigger`, `spec`, `external_system`, `source external`) or prose (`description` text, an `invariant`'s statement). The rest carry a value read by something outside the prose: an automation's `every` schedule expression and its `after` delay, and an event's wire `type`.

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

      spec "borrows a copy no one holds" {
        when BorrowCopy
        then [CopyBorrowed]
      }

      spec "refuses a copy already on loan" {
        given [CopyBorrowed]
        when BorrowCopy
        then rejected OneCopyPerLoan
      }

      spec "refuses when member has five copies" {
        given [CopyBorrowed]
        when BorrowCopy
        then rejected FiveCopiesPerMember
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

    spec "claims a free desk" {
      when ClaimDesk
      then [DeskClaimed]
    }

    spec "refuses when reader is already seated" {
      given [DeskClaimed]
      when ClaimDesk
      then rejected OneReaderPerDesk
    }

    spec "refuses when desk is taken" {
      given [DeskClaimed]
      when ClaimDesk
      then rejected OneDeskPerReader
    }
  }
}
```

- **A name is declared once per scope:** two invariants sharing a name in one scope is a validation error, reported on the second declaration:

  ```
  lending.emod:7: invariant "OneCopyPerLoan" is already declared in aggregate "Loan"
  ```

- **An aggregate and its context are separate scopes**, as are two sibling aggregates. The same identifier may be declared in each, and neither declaration hides the other.
- **An invariant nothing references is not a validation error:** `emod validate` reports nothing, but `emod lint` warns with `spec/invariant-never-exercised` for an invariant no `then rejected` spec names in its own scope.
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
  flow { ... }       # 0+ command→event and command→rejection wiring
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
    subscribes [<EventName>, <EventName>, ...]
    fields { ... }
  }
}
```

`subscribes` references event names declared **anywhere in the model** — in any slice, under any aggregate or context, the same scope an automation's `reads` resolves in (see [Cross-References](#11-cross-references)).

- **`view/never-read` asks who acts on it:** the rule reports at warning severity when no `reads` anywhere in the model names the view — neither a trigger's, an automation's nor a translation's. A view takes one of two legitimate shapes: a trigger reads it, or a processor reads it as its todo list. The rule fires only once the model states at least one `reads`, so a model that has not adopted the concept reports nothing. It also stays silent for the whole model while some `reads` names a view no slice declares: that name is already reported where it is written (see [Cross-References](#11-cross-references)), and nothing here can tell which view it was meant to be, so one misspelling yields one diagnostic rather than a second one pointing at the view. Run `emod lint --explain view/never-read` for the full description.

### Automation Pattern

A processor woken by an event — at once, or a fixed duration after it — or by a schedule reads its outstanding work and issues a command, possibly in a different context.

```
slice "<name>" {
  automation <Name> {
    on <EventName> after "<duration>"   # activation — exactly one of on and every; after is optional
    every "<expr>"                      # activation — exactly one of on and every
    reads <ViewName>
    command <CommandName>
    target context <ContextName>
  }
}
```

- `on`: the event whose arrival wakes the processor. The name must be an event declared in the model.
- `after`: an optional delay on the `on` line, read as — the stated duration after each occurrence of the `on` event, issue the command. The quoted value is a Go duration (`"30m"`, `"24h"`, `"1h30m"`). Only the syntax is judged; a value that does not parse as one is a validation error naming the value and the form it wanted:

  ```
  reservation.emod:34: delay "1 day" is not a Go duration such as "30m", "24h" or "1h30m"
  ```

  Without `after` the automation reacts immediately. The delay qualifies an activation rather than standing on its own, so writing it on a line of its own is an error asking for it to be moved onto the activation line.
- `every`: the schedule on which the processor wakes instead. The quoted expression is either a Go duration (`"5m"`, `"1h"`) for a fixed interval or a five-field cron expression (`"0 2 * * *"`) for a wall-clock schedule. Only the shape is checked; an expression that is neither is a validation error naming both forms:

  ```
  reservation.emod:34: schedule expression "nightly" is neither a Go duration nor a five-field cron expression
  ```

- `reads`: the todo list — the view holding the work the processor has left to do. Optional. The name must resolve to a view declared **anywhere in the model** — in any slice, under any aggregate or context — and `emod validate` reports one that resolves to nothing.
- `command`: the command the automation issues. Required.
- `target context`: a context name (cross-reference, validated at `emod validate`). The command the automation issues is handled there, so the event acknowledging that work is declared there too — and a todo list only loses the row it acted on by observing that event. A `reads` view therefore `subscribes` across the context boundary, which is what keeps the loop closed when the two halves of it live in different contexts. [examples/all_patterns.emod](/examples/all_patterns.emod) shows the shape: `ConfirmationEmailReactor` sits in `Reservations`, issues `SendConfirmationEmail` into `Notifications`, and reads a view subscribing to the `ConfirmationEmailSent` that context declares.

An automation states **exactly one** of `on` and `every`: declaring neither is an error, and declaring both is an error. Requiring the choice makes the wake-up explicit, so the model says whether the processor is woken by the event that adds to its todo list or polls on a cadence — two designs with different failure modes.

`after` qualifies the `on` half of that choice only, and stating it beside `every` is an error:

```
reservation.emod:41: automation block cannot declare after with every: an every schedule is already absolute, and after measures a delay from an on event
```

The two are one rule rather than two because they answer the same question in units that do not compose. A schedule is absolute — every hour, or at two in the morning — and names no occurrence to count from, while a delay is measured from an event a schedule-driven automation never receives. `every "0 2 * * *" after "24h"` therefore has no reading, and the error is reported on the delay.

- **The drawn diagrams label the edge, not the box:** `emod diagram --format drawio` and `--format svg` write `after "<duration>"` on the arrow from the `on` event to the automation, and reserve the clock badge on the automation's own shape for an `every` schedule. A relative delay and a wall-clock cadence therefore read differently at a glance, and no automation carries both marks.
- **The text formats state it beside the activation:** `emod diagram --format ascii` prints `(<EventName>) after "<duration>" -> ⚙ <Name> -> [<CommandName>]`, and `--format mermaid`, which draws no arrows, carries the delay inside the automation's own label.

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

A `spec` states expected behaviour as Given-When-Then, next to the structure it describes. The `then` shape a spec may state depends on the pattern of the enclosing slice. A slice may hold any number of specs, in any position among its other entries.

**Command slice spec.** A slice that declares a `command` or a `translation` accepts the event-list and rejection outcomes. The command slice is the pattern the existing examples show:

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

`when` names the command under test. A bracketed event list after `then` is the success outcome — the events appended, in order:

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

**View slice spec.** A slice that declares a `view` concludes its spec with `then view <ViewName>`. The outcome names a view declared anywhere in the model. A view-slice spec omits `when` — a view has no command to exercise:

```
slice "Review Member Loans" {
  view MemberLoansView {
    subscribes [CopyBorrowed, CopyReturned]
    fields { loanId string required }
  }

  spec "shows active member loans" {
    given [CopyBorrowed, CopyReturned, LoanRenewed]
    then view MemberLoansView
  }
}
```

**Automation slice spec.** A slice that declares an `automation` concludes its spec with `then command <CommandName>`. The outcome names a command declared anywhere in the model. The `when` entry distinguishes the two activation forms: an event-driven automation's `when` names the event the automation activates on, while a schedule-driven automation omits `when` entirely:

```
# Event-driven
spec "recalls overdue copy" {
  when CopyOverdue
  then command RecallCopy
}

# Schedule-driven
spec "chases overdue copies on schedule" {
  then command ChaseOverdue
}
```

A spec that omits `when` in a view slice and a schedule-driven automation are told apart by their outcome — `then view` for the former, `then command` for the latter.

**Translation slice spec.** A translation accepts the given/when/then-events form its enclosing slice's command-and-event pair exercises:

```
slice "Claim Desk" {
  command ClaimCopy { fields { copyId string required } }
  translation DeskWatcher {
    external_system "Desk"
    reads DeskClaimsView
    command ClaimCopy
    event CopyClaimed { fields { copyId string required } }
  }

  spec "claims a copy via the desk" {
    given [CopyBorrowed]
    when ClaimCopy
    then [CopyClaimed]
  }
}
```

**Outcome–pattern rule.** A `then` shape the enclosing slice cannot state is a validation error. The rule is local to the slice: a `view` outcome requires a `view` declaration, a `command` outcome requires an `automation`, a rejection or an event list requires a `command` or a `translation`. The message names the outcome shape and the construct kind the slice would have to declare.

**Name resolution.** Every event in `given` and in a `then` list, and the command in `when`, must be defined somewhere in the model. A `rejected` name must be declared on the enclosing aggregate or, for a slice declared directly on a context, on that context — an aggregate and its context are separate scopes, so a name declared one level up does not resolve. A `then view` name must be a view declared anywhere in the model, and a `then command` name a command declared anywhere in the model; both are unqualified and model-wide.

**Example payloads.** Any event or command reference in a spec may be followed by a brace block of `field: value` pairs, stating the example values the scenario runs on. A payload may be written on every element of a `given` list, on the `when` reference, and on every element of a `then` event list:

```
spec "borrows a copy no one holds" {
  given [CopyReturned { copyId: "C-93204" }]
  when BorrowCopy { copyId: "C-93204", dueOn: "2024-07-19" }
  then [CopyBorrowed { lateFee: 12.50, renewals: 4, expedited: true }]
}
```

The opening brace sits on the line of the reference it qualifies; the entries and the closing brace may span lines, and the comma between entries is optional. `then rejected` names an invariant rather than an event or a command, so it takes no payload, and expected view-state payloads on `then view` are not part of the language.

A payload is partial by design: a field declared `required` may be omitted, and a names-only reference stays valid everywhere. Writing `{}` means the same as writing no payload at all.

Three literal forms exist, and each satisfies a set of declared [field types](#8-fields):

| Literal | Written | Satisfies |
|---|---|---|
| String | `"C-93204"` | `string`; `date` when the value reads as `YYYY-MM-DD`; `timestamp` when it reads as an RFC 3339 timestamp; `uuid` when it reads as the canonical 36-character 8-4-4-4-12 hexadecimal form, in either case |
| Number | `42`, `12.50` | `decimal`; `int` when the literal states no fractional part. Numbers are unsigned — a leading `-` is not part of the literal grammar |
| Boolean | `true`, `false` | `bool` |

Any other declared type is a domain type, opaque to the model, and accepts any literal unchecked. `true` and `false` are recognised by position rather than reserved, so both stay usable as field names, types and modifiers.

`emod validate` reports two things about a payload: a field name the referenced construct does not declare on its `fields`, and a literal whose kind or format the declared type does not admit. A payload on a reference the model does not declare is left alone — the missing reference is reported on its own.

Specs and the payloads they state are carried through `emod fmt`, the JSON and CUE exports, and the embedded schema. A number reaches both exports as a number and a string as a string, digit for digit — a leading zero is dropped, since neither format has a literal for one, and everything else is written exactly as it was read.

- **The drawn diagrams show them as a card, on request:** `emod diagram --specs` draws a slice's specs as a Given-When-Then card under that slice, in a `Specs` band below the lowest lane, for the `drawio` and `svg` formats. A card states each spec's name, its `given` events, its `when` and its `then` outcome — naming the invariant for a rejection, the view for a `then view` and the command for a `then command` — and states element names only, never a payload's values or an invariant's prose.
- **The flag is off by default and refused where nothing draws a card:** without `--specs` every format renders exactly what it rendered before, and `emod diagram --specs` with `--format mermaid`, `--format ascii` or `--serve` exits 1 naming the flag and the two formats that do draw one, rather than writing a diagram missing what was asked for.

---

## 7. Flows

`flow` declares what a command leads to. A block holds two kinds of entry: the event a command produces, and the invariant that refuses it.

```
flow {
  command -> event: <CommandName> -> <EventName>
  command -> rejected: <CommandName> -> <invariantName>
}
```

The `->` arrow is a required operator. Both entry kinds may appear in one `flow { }` block, in any number and any order; `emod fmt` writes every `command -> event:` entry first, then every `command -> rejected:` entry. Flow declaration is optional for patterns where the wiring is implicit: an automation shows the activation→command link through its `on` event or its `every` schedule, and a translation shows the command→event link through its nested event.

`command -> rejected:` states that an [`invariant`](#invariant) can refuse the command. The command fails and **nothing is appended** — a rejection entry names no event because there is none. A failure the business cares about, such as a declined payment, is an event and belongs in a `command -> event:` entry instead; the rejection entry exists for the case where the timeline would otherwise be silent about how a command can fail.

The invariant name resolves against the enclosing aggregate, or for a slice declared directly on a `mode dcb` context, that context — the same scope rule and the same message a spec's `then rejected` gets. The command name is left unchecked, exactly as a flow's is.

- **The drawn diagrams show it as a dashed edge into a rejection badge:** `emod diagram --format drawio` and `--format svg` draw a dashed arrow from the command to a badge carrying the invariant's name, placed alongside that slice's events. The badge carries the invariant's statement as a draw.io tooltip and as an SVG `<title>` element, which browsers show on hover.
- **The ASCII preview states the relation as a line:** `emod diagram --format ascii` prints `[<CommandName>] -> ✗ <invariantName>`, a marker no element type wears. The `mermaid` format draws no arrows at all, so it shows nothing for a rejection entry.
- **`emod lint` asks for the scenario:** `flow/rejection-without-spec` reports at info severity when the slice holding the entry states no spec exercising it — no spec whose `when` names that command and whose `then` is a rejection naming that invariant.

---

## 8. Fields

Commands, events, and views contain structured field definitions.

```
fields {
  <name> <type> [modifier]
}
```

- **name**: Any identifier, including any word the DSL uses as a keyword. The examples in this document use lowerCamelCase.
- **type**: Any identifier (`string`, `date`, `timestamp`, `int`, domain types like `RoomID`, etc.). Seven spellings are checked against the literals a spec's [example payloads](#spec) state — `string`, `date`, `timestamp`, `uuid`, `int`, `decimal` and `bool`; any other type is a domain type and accepts any literal unchecked.
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

### Wire Types

An event can bind the type a consumer outside the model routes by — the subject a schema registry keys on, the `type` a CloudEvents consumer switches on:

```
event <Name> {
  type "<wire type>"
  fields { ... }
}
```

The value is an opaque string. Nothing in the model resolves against it, and the language places no structure on it:

```emod
emod 1
model "Hotel Reservation"

context "Reservations" {
  aggregate "Reservation" {
    slice "Reserve a Room" {
      command ReserveRoom {
        fields {
          roomId  string required
          guestId string required
        }
      }

      event RoomReserved {
        type "com.acme.reservations.room-reserved"
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

- **The attribute is optional:** an event that binds no wire type validates, formats and exports exactly as it did before the attribute existed. A model that uses none is unaffected in every respect.
- **Two events may not bind the same wire type:** `emod validate` reports an error naming both events and the repeated value. The scope is the whole model, not a context — a wire type is by construction an identifier outside the model, so two contexts binding one produce output a consumer cannot tell apart. This is the deliberate contrast with an [invariant](#invariant) name, which never leaves the model and is scoped per aggregate or context.
- **The comparison is verbatim:** wire types are case-sensitive, so `com.acme.room-reserved` and `Com.Acme.Room-Reserved` are two different types and do not collide.
- **The exports carry it, which is the point of the attribute:** `emod export -f json` and `emod export -f cue` both emit the wire type under a `type` key on the event, and the embedded schema declares it. `emod glossary` and the diagrams deliberately do not show it: a glossary defines the terms of a ubiquitous language, and a deployment identifier is not one of them.
- **`wire/type-format` nudges without enforcing:** the rule reports at info severity when a wire type does not read as reverse-DNS kebab-case — two or more dot-separated segments, each built from lowercase letters, digits and inner hyphens. A wire type that does not follow the convention is still valid emod, and an event binding none can never trip the rule. Run `emod lint --explain wire/type-format` for the full description.

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

All references use unqualified names, and `emod validate` resolves every one of them.

| Declaration | Referenced By | Context |
|---|---|---|
| `context "<name>"` | `automation { target context <Name> }` | [`automation`](#automation-pattern) |
| `event <Name>` | `subscribes [<Name>]`, `flow`, `automation { on <Name> }`, `automation { command <Name> }`, `translation { event <Name> }`, `translation { command <Name> }`, `spec { given [<Name>] }`, `spec { then [<Name>] }` | [`view`](#view-pattern), [`automation`](#automation-pattern), [`translation`](#translation-pattern), [`spec`](#spec) |
| `command <Name>` | `flow`, `automation { command <Name> }`, `translation { command <Name> }`, `spec { when <Name> }`, `spec { then command <Name> }` | [`flow`](#7-flows), [`automation`](#automation-pattern), [`translation`](#translation-pattern), [`spec`](#spec) |
| a command's or an event's `fields` | the field name of an [example payload](#spec) on a spec reference: `spec { when <Name> { <field>: <value> } }` | [`spec`](#spec) |
| `view <Name>` | `automation { reads <Name> }`, `trigger { reads <Name> }`, `translation { reads <Name> }`, `spec { then view <Name> }` | [`automation`](#automation-pattern), [`command` pattern](#command-pattern), [`translation`](#translation-pattern), [`spec`](#spec) |
| `actor "<name>"` | `trigger { actor <Name> }` | [`command` pattern](#command-pattern) |
| `invariant <name>` | `spec { then rejected <name> }`, `flow { command -> rejected: <Cmd> -> <name> }` | [`spec`](#spec), [`flow`](#7-flows) |

All three constructs that spell `reads` resolve alike — a trigger's, an automation's and a translation's. Each is looked up against the views the whole model declares, and a name matching none is reported at the `reads` entry that spells it, in the same words for all three:

```
lending.emod:22: view "MemberLoansVeiw" does not exist
```

Validation detects:
- Missing target contexts, commands, events, or views.
- **Orphan commands**: defined but never referenced by any flow, automation, or translation.
- **Orphan events**: defined but never produced by any flow, external source, or translation.
- **Redeclared invariants**: one name declared twice in a single scope (see [Bounded Contexts](#4-bounded-contexts)).
- **Undefined spec references**: an event in `given` or `then`, or a command in `when`, that the model does not define — reported as `event "<Name>" does not exist` or `command "<Name>" does not exist`.
- **Unresolved rejections**: a `then rejected <name>` in a spec, or a `command -> rejected:` entry in a `flow` block, whose invariant is not declared in the enclosing scope — both reported as `invariant "<name>" is not declared in <scope> "<Name>"`.
- **Undefined spec outcome references**: a `then view <Name>` naming a view no slice declares, or a `then command <Name>` naming a command no slice declares — reported as `view "<Name>" does not exist` or `command "<Name>" does not exist`.
- **Outcome–pattern mismatches**: a `view` outcome in a slice declaring no view, a `command` outcome in a slice declaring no automation, a rejection in a slice declaring no command, or an event list in a slice declaring neither a command nor a translation — each reported naming the outcome shape and the construct kind the slice would have to declare.
- **Undeclared payload fields**: an example payload naming a field the referenced construct's `fields` does not declare — reported as `payload field "<field>" is not declared on <kind> "<Name>"`.
- **Payload literal mismatches**: a literal whose kind or format the field's declared type does not admit — reported as `payload value <value> for field "<field>" is not a valid <type>`.

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



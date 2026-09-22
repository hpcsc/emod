# emod DSL Reference

The emod DSL is a human-readable, version-controllable text format for describing event-driven architectures using Event Modeling concepts. A model is written in HCL and files use the `.emod` extension.

- **Source**: `internal/parser/`
- **AST**: `internal/ast/ast.go`
- **Schema (CUE)**: `internal/cue/schema.cue`

---

## 1. General Syntax

### Comments

A comment starts with `#`, `//`, or opens with `/*` and closes with `*/`. `emod fmt` writes every comment back above the construct it was written above.

```
# This is a comment
model "My System" {}
```

### Blocks and attributes

A construct is a block: a keyword, a quoted name, and a body in braces. What the construct carries is an attribute: a name, `=`, and a value.

```
command "PlaceOrder" {
  description = "Ask the shop to place an order"
}
```

A name is a quoted label, so any text names a construct. `command "source"` and `slice "Reserve a Room"` are both legal.

### References

A reference to another construct is written unquoted, and `emod validate` resolves every one of them.

```
actor = Guest
subscribes = [RoomReserved, GuestCheckedOut]
```

### Calls

A call qualifies a value. Six of them exist, and no other call is accepted:

| Call | Where | Meaning |
|---|---|---|
| `required(<type>)` | a field | the field must be present |
| `optional(<type>)` | a field | the field may be absent |
| `external("<name>")` | an event's `source` | the event originates outside the model |
| `rejected(<invariant>)` | a spec's `then` | the command is refused |
| `view(<view>)` | a spec's `then` | the view holds the outcome |
| `command(<command>)` | a spec's `then` | the automation issues the command |
| `tag(<key>, <field>)` | a `where` predicate | a tag term |

A spec reference is also written as a call when it carries an example payload: `ReserveRoom({ roomId = "R-204" })`.

Nothing is evaluated. emod reads each call as it is written, so no variables, no functions of your own and no interpolation take part.

---

## 2. Version Header

A file opens with a version attribute, before `model`. It pins the file to a revision of the grammar.

```
emod = 1
model "Hotel Reservation" {}
```

The version is a single integer.

- **Absence means version 1:** a file with no header parses exactly as one declaring `emod = 1`.
- **`emod fmt` writes the header** as the first line, so formatted files are always pinned.
- **An unsupported version is rejected**, and the diagnostic reports the declared version and the supported version on line 1:

  ```
  reservation.emod:1: unsupported version 2: this tool supports emod version 1
  ```

Version numbers follow the Kubernetes convention: additive grammar changes, such as new optional attributes, do not bump the version; breaking changes do.

---

## 3. Top-Level Constructs

### `model`

Required root declaration. Names the system being modeled.

```
model "<name>" {
  description = "<text>"
}
```

Exactly one per file. The body carries a [description](#10-descriptions) and nothing else; an empty body is allowed. See [examples/all_patterns.emod](/examples/all_patterns.emod).

### `actor`

Declares a persona or external system that interacts with the model.

```
actor "<name>" {
  description = "<text>"
}
```

Multiple actors are allowed. Actors are referenced by triggers (see [Slice Patterns → Command](#command-pattern)).

---

## 4. Bounded Contexts

### `context`

Groups aggregates (or slices in DCB mode) under a bounded context.

```
context "<name>" {
  aggregate "<name>" { ... }
}
```

An optional `mode` attribute declares the consistency boundary style. It is written as a bare word:

| Mode | Description |
|------|-------------|
| `aggregate` (default) | Slices must live inside an `aggregate` block. Tags, `decides_on`, and direct slices produce lint warnings. |
| `dcb` | Slices live directly under the context (no `aggregate`). Events use `tags`, commands use `decides_on`. An `aggregate` block produces a lint warning. |
| `mixed` | Both aggregate-wrapped and direct slices are accepted. DCB constructs are allowed without warnings. |

```
context "<name>" {
  mode = dcb

  slice "<name>" { ... }
}
```

A model must contain at least one context. Context names are referenced by automations (see [`target`](#automation-pattern)). See [examples/dcb_model.emod](/examples/dcb_model.emod).

### `aggregate`

Groups slices within a context. Represents a consistency boundary. Not used in `mode dcb` contexts.

```
aggregate "<name>" {
  slice "<name>" { ... }
}
```

### `invariants`

Declares the business rules the enclosing scope keeps true. Each entry is a name and a quoted prose statement of the rule.

```
invariants {
  <Name> = "<statement>"
}
```

In aggregate mode the block sits on the `aggregate` whose consistency boundary keeps the rules. A `mode dcb` context has no aggregate, so its block sits on the context.

```emod
emod = 1
model "Library Lending" {}

context "Lending" {
  aggregate "Loan" {
    invariants {
      OneCopyPerLoan      = "A loan covers exactly one copy of one title"
      FiveCopiesPerMember = "A member holds at most five copies at one time"
    }

    slice "Borrow a Copy" {
      command "BorrowCopy" {
        fields {
          memberId = string
          copyId   = string
        }
      }

      event "CopyBorrowed" {
        fields {
          loanId   = string
          memberId = string
          copyId   = string
        }
      }

      flow = <<-FLOW
        command -> event: BorrowCopy -> CopyBorrowed
      FLOW

      spec "borrows a copy no one holds" {
        when = BorrowCopy
        then = [CopyBorrowed]
      }

      spec "refuses a copy already on loan" {
        given = [CopyBorrowed]
        when  = BorrowCopy
        then  = rejected(OneCopyPerLoan)
      }

      spec "refuses when member has five copies" {
        given = [CopyBorrowed]
        when  = BorrowCopy
        then  = rejected(FiveCopiesPerMember)
      }
    }
  }
}

context "Reading Room" {
  mode = dcb

  invariants {
    OneReaderPerDesk = "A desk seats at most one reader at any moment"
    OneDeskPerReader = "A reader holds at most one desk for the length of a session"
  }

  slice "Claim Desk" {
    command "ClaimDesk" {
      decides_on {
        events = [DeskClaimed]
        where  = tag(desk, deskId) && tag(reader, memberId)
      }

      fields {
        memberId = string
        deskId   = string
      }
    }

    event "DeskClaimed" {
      tags {
        desk   = deskId
        reader = memberId
      }

      fields {
        sessionId = string
        deskId    = string
        memberId  = string
      }
    }

    flow = <<-FLOW
      command -> event: ClaimDesk -> DeskClaimed
    FLOW

    spec "claims a free desk" {
      when = ClaimDesk
      then = [DeskClaimed]
    }

    spec "refuses when reader is already seated" {
      given = [DeskClaimed]
      when  = ClaimDesk
      then  = rejected(OneReaderPerDesk)
    }

    spec "refuses when desk is taken" {
      given = [DeskClaimed]
      when  = ClaimDesk
      then  = rejected(OneDeskPerReader)
    }
  }
}
```

- **A name is declared once per scope:** two entries sharing a name in one scope is a validation error.
- **An aggregate and its context are separate scopes**, as are two sibling aggregates. The same name may be declared in each, and neither declaration hides the other.
- **An invariant nothing references is not a validation error:** `emod validate` reports nothing, but `emod lint` warns with `spec/invariant-never-exercised` for an invariant no `then rejected` spec names in its own scope.
- **`emod fmt` writes the block at the top of its scope**, after the `description` line and ahead of the aggregates and slices.
- **The glossary lists them under the scope that declares them:** `emod glossary` puts an "Invariants" group under each aggregate and each context declaring one, in declaration order, with the statement standing as the definition.
- **Exports carry them:** `emod export --format json` and `emod export --format cue` emit an `invariants` list of `name` and `statement` on every aggregate and context that declares one.
- **No diagram renders them:** a diagram shows elements and the arrows between them.

---

## 5. Slices

A slice is an implementation unit representing a single use case in Event Modeling. All event-modeling primitives live inside slices. In aggregate mode, slices are nested inside an `aggregate`. In DCB mode, slices are direct children of the context.

```
slice "<name>" {
  description = "<text>"       # optional
  flow        = <<-FLOW ...    # 0-1 flow heredoc

  trigger     "<name>" { ... } # 0-1 trigger (Command Pattern)
  command     "<Name>" { ... } # 0+ commands
  event       "<Name>" { ... } # 0+ events
  view        "<Name>" { ... } # 0+ views (View Pattern)
  automation  "<Name>" { ... } # 0+ automations (Automation Pattern)
  translation "<Name>" { ... } # 0+ translations (Translation Pattern)
  spec        "<name>" { ... } # 0+ Given-When-Then specs
}
```

Every slice must contain at least one element. See [Slice Patterns](#6-slice-patterns) for valid combinations.

---

## 6. Slice Patterns

The DSL supports four Event Modeling slice patterns. Each pattern prescribes which blocks appear together.

The two elements that start a chain belong to different chains. A `trigger` is the human entry point of the command chain: something outside the system initiates the slice. An `automation` is the processor of the automation chain: the system initiates the slice itself. Neither stands in for the other, and a slice that needs both declares both.

### Command Pattern

A user or system action drives a command that produces an event.

```
slice "<name>" {
  trigger "<name>" {
    actor = <ActorName>
    reads = <ViewName>
  }

  command "<CommandName>" {
    fields { ... }
  }

  event "<EventName>" {
    fields { ... }
  }

  flow = <<-FLOW
    command -> event: <CommandName> -> <EventName>
  FLOW
}
```

`flow` wires the command to the event (see [Flows](#7-flows)). `trigger` is optional — a slice may define a command directly without a trigger.

### View Pattern

A read model projects from subscribed events.

```
slice "<name>" {
  view "<ViewName>" {
    subscribes = [<EventName>, <EventName>]
    fields { ... }
  }
}
```

`subscribes` references event names declared **anywhere in the model** — in any slice, under any aggregate or context, the same scope an automation's `reads` resolves in (see [Cross-References](#11-cross-references)).

- **`view/never-read` asks who acts on it:** the rule reports at warning severity when no `reads` anywhere in the model names the view. A view takes one of two legitimate shapes: a trigger reads it, or a processor reads it as its todo list. The rule fires only once the model states at least one `reads`. Run `emod lint --explain view/never-read` for the full description.

### Automation Pattern

A processor woken by an event — at once, or a fixed duration after it — or by a schedule reads its outstanding work and issues a command, possibly in a different context.

```
slice "<name>" {
  automation "<Name>" {
    on      = <EventName>   # activation — exactly one of on and every
    after   = "<duration>"  # optional, only beside on
    every   = "<expr>"      # activation — exactly one of on and every
    reads   = <ViewName>
    command = <CommandName>

    target {
      context = <ContextName>
    }
  }
}
```

- `on`: the event whose arrival wakes the processor. The name must be an event declared in the model.
- `after`: an optional delay, read as — the stated duration after each occurrence of the `on` event, issue the command. The quoted value is a Go duration (`"30m"`, `"24h"`, `"1h30m"`). A value that does not parse as one is a validation error:

  ```
  reservation.emod:34: delay "1 day" is not a Go duration such as "30m", "24h" or "1h30m"
  ```

  Without `after` the automation reacts immediately.
- `every`: the schedule on which the processor wakes instead. The quoted expression is either a Go duration (`"5m"`, `"1h"`) for a fixed interval or a five-field cron expression (`"0 2 * * *"`) for a wall-clock schedule:

  ```
  reservation.emod:34: schedule expression "nightly" is neither a Go duration nor a five-field cron expression
  ```

- `reads`: the todo list — the view holding the work the processor has left to do. Optional. The name must resolve to a view declared **anywhere in the model**.
- `command`: the command the automation issues. Required.
- `target`: names the context that handles the command. The event acknowledging that work is declared there too, so a `reads` view `subscribes` across the context boundary, which is what keeps the loop closed.

An automation states **exactly one** of `on` and `every`: declaring neither is an error, and declaring both is an error. Requiring the choice makes the wake-up explicit.

`after` qualifies `on` only, and stating it beside `every` is an error:

```
reservation.emod:41: automation block cannot declare after with every: an every schedule is already absolute, and after measures a delay from an on event
```

A schedule is absolute and names no occurrence to count from, while a delay is measured from an event a schedule-driven automation never receives.

- **The drawn diagrams label the edge, not the box:** `emod diagram --format drawio` and `--format svg` write `after "<duration>"` on the arrow from the `on` event to the automation, and reserve the clock badge on the automation's own shape for an `every` schedule.
- **The text formats state it beside the activation:** `emod diagram --format ascii` prints `(<EventName>) after "<duration>" -> ⚙ <Name> -> [<CommandName>]`, and `--format mermaid` carries the delay inside the automation's own label.

### Translation Pattern

An external system integration: reads a view, issues a command, produces an event.

```
slice "<name>" {
  command "<CommandName>" {
    fields { ... }
  }

  translation "<Name>" {
    external_system = "<name>"
    reads           = <ViewName>
    command         = <CommandName>

    event "<EventName>" {
      fields { ... }
    }
  }
}
```

`external_system`, `reads`, `command`, and the nested `event` are required.

### `spec`

A `spec` states expected behaviour as Given-When-Then, next to the structure it describes. The `then` shape a spec may state depends on the pattern of the enclosing slice. A slice may hold any number of specs.

**Command slice spec.** A slice that declares a `command` or a `translation` accepts the event-list and rejection outcomes.

```
slice "Borrow a Copy" {
  command "BorrowCopy" {
    fields { copyId = string }
  }

  event "CopyBorrowed" {
    fields { copyId = string }
  }

  spec "borrows a free copy" {
    given = []
    when  = BorrowCopy
    then  = [CopyBorrowed]
  }
}
```

`given` is the history the command decides against, as an ordered event list. `given = []` and omitting `given` entirely mean the same empty history.

`when` names the command under test. A list after `then` is the success outcome — the events appended, in order:

```
then = [CopyBorrowed, LoanOpened]
```

`then = rejected(<invariantName>)` is the failure outcome: the command is refused and nothing is appended. The name refers to a declared [invariant](#invariants), so a rejection is a checkable reference to a stated rule rather than free text:

```
spec "rejects a second borrow" {
  given = [CopyBorrowed]
  when  = BorrowCopy
  then  = rejected(OneCopyPerLoan)
}
```

**View slice spec.** A slice that declares a `view` concludes its spec with `then = view(<ViewName>)`. A view-slice spec omits `when` — a view has no command to exercise:

```
spec "shows active member loans" {
  given = [CopyBorrowed, CopyReturned, LoanRenewed]
  then  = view(MemberLoansView)
}
```

**Automation slice spec.** A slice that declares an `automation` concludes its spec with `then = command(<CommandName>)`. The `when` entry distinguishes the two activation forms: an event-driven automation's `when` names the event it activates on, while a schedule-driven automation omits `when`:

```
# Event-driven
spec "recalls overdue copy" {
  when = CopyOverdue
  then = command(RecallCopy)
}

# Schedule-driven
spec "chases overdue copies on schedule" {
  then = command(ChaseOverdue)
}
```

A spec that omits `when` in a view slice and a schedule-driven automation are told apart by their outcome.

**Translation slice spec.** A translation accepts the given/when/then-events form its enclosing slice's command-and-event pair exercises.

**Outcome–pattern rule.** A `then` shape the enclosing slice cannot state is a validation error. The rule is local to the slice: a `view` outcome requires a `view` declaration, a `command` outcome requires an `automation`, a rejection or an event list requires a `command` or a `translation`.

**Name resolution.** Every event in `given` and in a `then` list, and the command in `when`, must be defined somewhere in the model. A `rejected` name must be declared on the enclosing aggregate or, for a slice declared directly on a context, on that context. A `view` name must be a view declared anywhere in the model, and a `command` name a command declared anywhere in the model.

**Example payloads.** Any event or command reference in a spec may be written as a call carrying the example values the scenario runs on:

```
spec "borrows a copy no one holds" {
  given = [CopyReturned({ copyId = "C-93204" })]
  when  = BorrowCopy({ copyId = "C-93204", dueOn = "2024-07-19" })
  then  = [CopyBorrowed({ lateFee = 12.50, renewals = 4, expedited = true })]
}
```

A payload may span lines, and the comma between entries is optional when each entry sits on its own line. `then = rejected(...)` names an invariant rather than an event, so it takes no payload.

A payload is partial by design: a field declared `required` may be omitted, and a names-only reference stays valid everywhere. Writing `({})` means the same as writing no payload at all.

Three literal forms exist, and each satisfies a set of declared [field types](#8-fields):

| Literal | Written | Satisfies |
|---|---|---|
| String | `"C-93204"` | `string`; `date` when the value reads as `YYYY-MM-DD`; `timestamp` when it reads as an RFC 3339 timestamp; `uuid` when it reads as the canonical 36-character 8-4-4-4-12 hexadecimal form |
| Number | `42`, `12.50` | `decimal`; `int` when the literal states no fractional part |
| Boolean | `true`, `false` | `bool` |

Any other declared type is a domain type, opaque to the model, and accepts any literal unchecked.

`emod validate` reports two things about a payload: a field name the referenced construct does not declare on its `fields`, and a literal whose kind or format the declared type does not admit.

Specs and the payloads they state are carried through `emod fmt`, the JSON and CUE exports, and the embedded schema. A number reaches both exports as a number and a string as a string, digit for digit.

- **The drawn diagrams show them as a card, on request:** `emod diagram --specs` draws a slice's specs as a Given-When-Then card under that slice, in a `Specs` band below the lowest lane, for the `drawio` and `svg` formats.
- **The flag is off by default and refused where nothing draws a card:** `emod diagram --specs` with `--format mermaid`, `--format ascii` or `--serve` exits 1 naming the flag and the two formats that do draw one.

---

## 7. Flows

`flow` declares what a command leads to. It is a heredoc, because the arrow has no operator in HCL. A block holds two kinds of entry: the event a command produces, and the invariant that refuses it.

```
flow = <<-FLOW
  command -> event:    <CommandName> -> <EventName>
  command -> rejected: <CommandName> -> <invariantName>
FLOW
```

Both entry kinds may appear in one heredoc, in any number and any order; `emod fmt` writes every `command -> event:` entry first, then every `command -> rejected:` entry, and lines the heredoc up with the attribute that opens it. `<<-` strips that indentation again when the file is read.

Flow declaration is optional for patterns where the wiring is implicit: an automation shows the activation→command link through its `on` event or its `every` schedule, and a translation shows the command→event link through its nested event.

`command -> rejected:` states that an [invariant](#invariants) can refuse the command. The command fails and **nothing is appended** — a rejection entry names no event because there is none. A failure the business cares about, such as a declined payment, is an event and belongs in a `command -> event:` entry instead.

The invariant name resolves against the enclosing aggregate, or for a slice declared directly on a `mode dcb` context, that context. The command name is left unchecked.

- **The drawn diagrams show it as a dashed edge into a rejection badge:** `emod diagram --format drawio` and `--format svg` draw a dashed arrow from the command to a badge carrying the invariant's name. The badge carries the invariant's statement as a tooltip.
- **The ASCII preview states the relation as a line:** `emod diagram --format ascii` prints `[<CommandName>] -> ✗ <invariantName>`. The `mermaid` format draws no arrows at all.
- **`emod lint` asks for the scenario:** `flow/rejection-without-spec` reports at info severity when the slice holding the entry states no spec exercising it.

---

## 8. Fields

Commands, events, and views contain structured field definitions. Each field is one attribute: a name, and the type it carries.

```
fields {
  <name> = <type>
  <name> = optional(<type>)
  <name> = required(<type>)
}
```

- **name**: any identifier, including any word the DSL uses as a keyword.
- **type**: any identifier (`string`, `date`, `timestamp`, `int`, domain types like `RoomID`). Seven spellings are checked against the literals a spec's [example payloads](#spec) state — `string`, `date`, `timestamp`, `uuid`, `int`, `decimal` and `bool`; any other type is a domain type and accepts any literal unchecked.
- **`optional()` and `required()`**: optional. A bare type and `required(<type>)` say the same thing, and `emod fmt` keeps whichever was written.

```
command "ReserveRoom" {
  fields {
    roomId     = string
    guestName  = string
    checkIn    = date
    accessible = optional(bool)
  }
}
```

### Keywords as Field Names

The DSL has no reserved words. A `fields` block has its own namespace, so every keyword is usable as a field name:

```emod
emod = 1
model "Hotel Reservation" {}

context "Reservations" {
  aggregate "Reservation" {
    slice "Reserve a Room" {
      command "ReserveRoom" {
        fields {
          roomId      = string
          source      = string
          description = string
        }
      }

      event "RoomReserved" {
        fields {
          reservationId = string
          tags          = string
        }
      }

      flow = <<-FLOW
        command -> event: ReserveRoom -> RoomReserved
      FLOW
    }
  }
}
```

A construct name is a quoted label, so the same freedom covers it: `command "source"` names a command `source`.

### External Source Events

An event can declare it originates from an external system:

```
event "<Name>" {
  source = external("<Provider Name>")
  fields { ... }
}
```

### Wire Types

An event can bind the type a consumer outside the model routes by — the subject a schema registry keys on, the `type` a CloudEvents consumer switches on:

```
event "<Name>" {
  type = "<wire type>"
  fields { ... }
}
```

The value is an opaque string. Nothing in the model resolves against it.

- **The attribute is optional:** an event that binds no wire type validates, formats and exports exactly as it did before the attribute existed.
- **Two events may not bind the same wire type:** `emod validate` reports an error naming both events and the repeated value. The scope is the whole model, because a wire type is by construction an identifier outside the model. This is the deliberate contrast with an invariant name, which never leaves the model and is scoped per aggregate or context.
- **The comparison is verbatim:** wire types are case-sensitive.
- **The exports carry it:** `emod export -f json` and `emod export -f cue` both emit the wire type under a `type` key on the event. `emod glossary` and the diagrams deliberately do not show it: a glossary defines the terms of a ubiquitous language, and a deployment identifier is not one of them.
- **`wire/type-format` nudges without enforcing:** the rule reports at info severity when a wire type does not read as reverse-DNS kebab-case. Run `emod lint --explain wire/type-format` for the full description.

---

## 9. Dynamic Consistency Boundaries

DCB mode is additive — aggregate-based models continue to work without changes. DCB constructs are valid in `dcb` and `mixed` modes only.

### Event Tags

Events in a DCB slice declare tags that cross-reference event fields. Tags allow commands to express consistency boundaries over event data.

```
event "<Name>" {
  tags {
    <key> = <fieldRef>
  }

  fields { ... }
}
```

- **key**: a name for the tag dimension (for example `entity`, `category`, `region`).
- **fieldRef**: a field name declared on this event, written unquoted.
- Tags are validated at `emod validate`: every `fieldRef` must match a declared field.

### Command decides_on

Commands declare a consistency boundary by listing the event types they depend on and a predicate over tag values.

```
command "<Name>" {
  decides_on {
    events = [<EventName>]
    where  = <predicate>
  }

  fields { ... }
}
```

- **events**: one or more event names defined elsewhere in the model.
- **where**: a predicate expression. Optional — if omitted, the command matches any event.

### Predicate Expressions

A predicate is an ordinary HCL expression, so its grouping and its precedence come from HCL.

| Expression | Syntax | Example |
|---|---|---|
| Tag equality | `tag(<key>, <fieldRef>)` | `tag(entity, customerId)` |
| Logical AND | `<expr> && <expr>` | `tag(entity, id) && tag(region, country)` |
| Logical OR | `<expr> \|\| <expr>` | `tag(status, active) \|\| tag(status, trial)` |
| Negation | `!<expr>` | `!tag(category, premium)` |
| Grouping | `(<expr>)` | `(tag(a, x) \|\| tag(a, y)) && tag(b, z)` |

The tag key and field reference in each `tag()` term are validated at `emod validate`:
- The **tag key** must be declared on at least one event listed in `events`.
- The **field reference** must be a declared field on at least one listed event.

### Example

See [examples/dcb_model.emod](/examples/dcb_model.emod) for a complete DCB-mode model demonstrating tags, `decides_on`, compound predicates, and direct slices.

---

## 10. Descriptions

Any construct that has a body may carry a `description`: prose explaining what the construct is for.

```
description = "<text>"
```

`model`, `actor`, `context`, `aggregate`, `slice`, `trigger`, `command`, `event`, `view`, `automation` and `translation` accept it, including the `event` nested inside a `translation`.

```emod
emod = 1
model "Hotel Reservation" {
  description = "Room inventory, bookings and check-out."
}

actor "Guest" {
  description = "A person who books and stays in a room."
}

context "Reservations" {
  description = "Owns the booking lifecycle."

  aggregate "Reservation" {
    description = "One booking, from request to check-out."

    slice "Reserve a Room" {
      description = "A guest holds a room for a date range."

      trigger "Reservation Form" {
        description = "The public booking form."
        actor       = Guest
      }

      command "ReserveRoom" {
        description = "Holds a room for the requested dates."

        fields {
          roomId    = string
          guestName = string
        }
      }

      event "RoomReserved" {
        description = "A room is held for a guest."

        fields {
          reservationId = string
          roomId        = string
        }
      }

      flow = <<-FLOW
        command -> event: ReserveRoom -> RoomReserved
      FLOW
    }
  }
}
```

- **A description is documentation, not structure:** it is optional on every construct, nothing in the model refers to it, and no validation or lint rule reads it.
- **`description` is not a reserved word:** `fields { description = string }` still declares an ordinary field named `description`.
- **Position inside the body is free:** the parser accepts `description` before or after the construct's other entries.
- **`emod fmt` moves it to the first line of the body**, ahead of the construct's other attributes and its nested blocks.
- **Exports carry the text:** `emod export --format json` and `emod export --format cue` emit a `description` key on the model, on each actor and on each described construct.
- **Diagrams surface it on the shape:** `emod diagram --format drawio` attaches the description to the construct's shape as a tooltip, and `--format svg` emits it as a `<title>` element.
- **A construct without a shape stays off the diagrams:** `model`, `actor`, `aggregate` and `slice` own no shape in either renderer. The `mermaid` and `ascii` formats carry no descriptions at all.
- **The glossary reads it as the definition:** `emod glossary` prints the description beneath the name it defines.
- **A construct that defines no term stays out of the glossary:** `slice`, `trigger`, `automation` and `translation` contribute no term of their own.

---

## 11. Cross-References

All references use unqualified names, written unquoted, and `emod validate` resolves every one of them.

| Declaration | Referenced By | Context |
|---|---|---|
| `context "<name>"` | `automation { target { context = <Name> } }` | [`automation`](#automation-pattern) |
| `event "<Name>"` | `subscribes`, `flow`, `automation { on }`, `translation { event }`, `spec { given }`, `spec { then }` | [`view`](#view-pattern), [`automation`](#automation-pattern), [`translation`](#translation-pattern), [`spec`](#spec) |
| `command "<Name>"` | `flow`, `automation { command }`, `translation { command }`, `spec { when }`, `spec { then = command(...) }` | [`flow`](#7-flows), [`automation`](#automation-pattern), [`translation`](#translation-pattern), [`spec`](#spec) |
| a command's or an event's `fields` | the field name of an [example payload](#spec) | [`spec`](#spec) |
| `view "<Name>"` | `automation { reads }`, `trigger { reads }`, `translation { reads }`, `spec { then = view(...) }` | [`automation`](#automation-pattern), [`command` pattern](#command-pattern), [`translation`](#translation-pattern), [`spec`](#spec) |
| `actor "<name>"` | `trigger { actor }` | [`command` pattern](#command-pattern) |
| `invariants` entry | `spec { then = rejected(...) }`, `flow { command -> rejected: }` | [`spec`](#spec), [`flow`](#7-flows) |

All three constructs that spell `reads` resolve alike — a trigger's, an automation's and a translation's. Each is looked up against the views the whole model declares, and a name matching none is reported at the `reads` entry that spells it:

```
lending.emod:22: view "MemberLoansVeiw" does not exist
```

Validation detects:
- Missing target contexts, commands, events, or views.
- **Orphan commands**: defined but never referenced by any flow, automation, or translation.
- **Orphan events**: defined but never produced by any flow, external source, or translation.
- **Redeclared invariants**: one name declared twice in a single scope.
- **Undefined spec references**: an event in `given` or `then`, or a command in `when`, that the model does not define.
- **Unresolved rejections**: a `then = rejected(<name>)` in a spec, or a `command -> rejected:` entry in a flow, whose invariant is not declared in the enclosing scope.
- **Undefined spec outcome references**: a `view(<Name>)` naming a view no slice declares, or a `command(<Name>)` naming a command no slice declares.
- **Outcome–pattern mismatches**: a `view` outcome in a slice declaring no view, a `command` outcome in a slice declaring no automation, a rejection in a slice declaring no command, or an event list in a slice declaring neither a command nor a translation.
- **Undeclared payload fields**: an example payload naming a field the referenced construct's `fields` does not declare.
- **Payload literal mismatches**: a literal whose kind or format the field's declared type does not admit.

---

## 12. Pipeline

The CLI processes `.emod` files through a linear pipeline:

```
.emod file
  → HCL parser        (hashicorp/hcl/v2/hclsyntax)
  → Decoder           (internal/parser/hcl.go)
  → AST               (internal/ast/ast.go)
  → Validator         (internal/validator/validator.go)
  → Linter            (internal/linter/linter.go)
  → Formatter / Exporter / Diagram Generator
```

HCL carries a source range on every node, so each stage reports a file, line and column.

The decoder reads what HCL parsed rather than evaluating it: `hcl.AbsTraversalForExpr` for a reference, `hcl.ExprList` for a list, and a type switch on `*hclsyntax.FunctionCallExpr` for the six calls. A field keeps the order it was written in, taken from each attribute's source range.

## 13. Diagram Palette

The renderers that draw the whole model use one palette for element types. The SVG, draw.io, and web viewer renderers all draw the same element type with the same fill and stroke, and the DSL reference itself is the source of truth for those values. `emod diagram --format event-flow` and `--format event-flow-mermaid` draw a different picture and paint it in a palette of their own, listed below.

| Element     | Fill      | Stroke    | Notes                                   |
|-------------|-----------|-----------|-----------------------------------------|
| Trigger     | #ffffff   | #333333   | Drawn as a screen/monitor shape.        |
| Command     | #dae8fc   | #6c8ebf   | Blue sticky note.                       |
| Event       | #ffe6cc   | #d79b00   | Orange sticky note.                     |
| View        | #d5e8d4   | #82b366   | Green sticky note.                      |
| Automation  | #e1d5e7   | #9673a6   | Purple processor.                       |
| Translation | #f5f5f5   | #666666   | Grey integration.                       |

### Event Flow Palette

`--format event-flow` collapses the commands away and draws events, automations
and what starts a chain from outside them. It states both themes, because it is
read on a page of either, and writes every colour out in full: a renderer
outside a browser resolves no custom property and paints what it cannot resolve
black. `--format event-flow-mermaid` paints the light column of the same table
in its `classDef` lines, Mermaid's own theme carrying the dark one.

| Element               | Light fill | Light stroke | Dark fill | Dark stroke |
|-----------------------|------------|--------------|-----------|-------------|
| Event pill            | #FFE0C0    | #B44E12      | #40260F   | #F2A868     |
| Event nothing reads   | #FFE0C0    | #A33A2A      | #40260F   | #E88C78     |
| Automation gear       | #16201B    | —            | #E2E9E5   | —           |
| Origin box            | #ECEFED    | #5A6861      | #1E2723   | #93A199     |
| Invariant refusing it | #FFFFFF    | #A33A2A      | #16201B   | #E88C78     |
| Arrow                 | —          | #16201B      | —         | #E2E9E5     |
| Arrow to a refusal    | —          | #A33A2A      | —         | #E88C78     |

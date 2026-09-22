# HCL as the `.emod` Syntax — Proposal

## Problem

emod owns its surface syntax end to end. `internal/lexer` (422 lines) and
`internal/parser` (1,832 lines) turn `.emod` text into the AST, and
`internal/formatter` (637 lines) writes it back. Beside them sit
`editors/tree-sitter-emod`, its three query files and a VS Code extension that
exists mostly to attach them.

That is 2,891 lines of front end plus a grammar in a second language, and it
buys nothing that a reader of the model can name. Every grammar addition pays
the cost four times: the lexer, the parser, the formatter and the tree-sitter
grammar. Every editor that is not VS Code shows an `.emod` file as plain text.

HCL supplies the same shapes — labelled blocks, attributes, lists, objects,
comments, heredocs — with a parser, a writer, precise source ranges, a
tree-sitter grammar and editor support that already exist.

## Goals

- Delete the lexer, the parser, the formatter and the tree-sitter grammar.
- Keep the emod schema exactly as it is, so the AST and everything downstream of
  it stay in place.
- Keep the file readable. A model must not grow much, and the `flow` block must
  keep its arrows.
- Keep diagnostics precise to the line and column.
- Keep the `.emod` extension.

## Non-Goals

- **Adopting the Event Modeling HCL specification.** That language has a
  different schema: events live in a context catalog, slices become top-level
  workflows, and references are qualified traversals. It also has no place for
  invariants, rejections, DCB, schedules or wire types. This proposal changes
  the syntax only, so an `.emod` file stays an `.emod` file and `emhcl validate`
  still rejects it.
- **Changing what a model means.** No construct gains or loses meaning. The JSON
  and CUE exports of a converted file match the exports of the file it came
  from once the positions are set aside, which is the acceptance test.
- **Multi-file models.** One file is still one model.
- **Evaluating expressions.** No `EvalContext` and no function table take part.
  Every form is read statically.

## Decisions

Three forms had more than one reasonable spelling. They are settled.

| Choice | Decision | Reason |
| --- | --- | --- |
| `flow` | A heredoc holding today's text | HCL has no `->` operator, and the arrow is the one place where order reads as time |
| A qualified value | A call: `optional(bool)` | One rule covers every qualifier, and it costs no line |
| An automation's target | A block: `target { context = X }` | A target gains attributes later; an attribute does not |

The call rule reaches further than the optional field. A call qualifies a value
wherever emod used a keyword in front of one: `optional(bool)`,
`rejected(Invariant)`, `view(View)`, `command(Command)`, `external("Provider")`
and a spec payload such as `ReserveRoom({ ... })`.

---

## DSL Surface

### 1. General syntax

| Item | Today | Proposed |
| --- | --- | --- |
| Comment | `# text` | `# text`, and also `//` and `/* */` |
| Construct name | bare identifier: `command ReserveRoom` | quoted label: `command "ReserveRoom"` |
| Human name | `slice "Reserve a Room"` | unchanged |
| Reference | bare name: `reads AvailableRoomsView` | bare name: `reads = AvailableRoomsView` |
| Attribute | by position: `actor Guest` | by `=`: `actor = Guest` |
| List | `subscribes [A, B]` | `subscribes = [A, B]` |
| Reserved words | none | none |

A reference stays unquoted because the decoder reads the expression with
`hcl.AbsTraversalForExpr` instead of evaluating it.

A field name stays free. `fields { description = string }` and
`fields { type = string }` are both legal, because the `fields` block has its
own namespace. This is stronger than the guarantee today, where a construct name
must not be a keyword: a label is a string, so `command "source"` also works.

### 2. Version header

```emod
emod 1
```
```hcl
emod = 1
```

The decoder reads the top-level attribute before anything else, so an
unsupported version is still reported on line 1 rather than as a parse failure
deep in the file. "The header is the first line" stops being a parser rule and
becomes a formatter rule.

### 3. model, actor, aggregate, slice

Labels and `=`, otherwise unchanged.

```hcl
model "Hotel Reservation" {
  description = "How the hotel takes a stay"
}
```

### 4. context and mode

```emod
context "Reading Room" mode dcb {
```
```hcl
context "Reading Room" {
  mode = dcb
```

`dcb` is a bare word, read as a traversal, so the mode keeps the spelling it has
today.

### 5. invariant

One block replaces the repeated keyword.

```emod
invariant RoomNotDoubleBooked "A room holds at most one reservation"
invariant StayClosedOnce      "A stay is closed exactly once"
```
```hcl
invariants {
  RoomNotDoubleBooked = "A room holds at most one reservation"
  StayClosedOnce      = "A stay is closed exactly once"
}
```

Declaration order comes from each attribute's `SrcRange`. The scope rules, the
duplicate-name error and the glossary grouping are unchanged.

### 6. trigger

```hcl
trigger "Reservation Form" {
  actor = Guest
  reads = AvailableRoomsView
}
```

### 7. fields

`required` is the default today, so only an optional field carries a wrapper.

| Today | Proposed |
| --- | --- |
| `roomId string required` | `roomId = string` |
| `roomId string` | `roomId = string` |
| `accessible bool optional` | `accessible = optional(bool)` |
| `total RoomPrice required` | `total = RoomPrice` |

```hcl
command "ReserveRoom" {
  fields {
    roomId     = string
    guests     = int
    accessible = optional(bool)
  }
}
```

A type is a traversal, so the seven checked spellings and every domain type keep
their look. `optional(bool)` is a `*hclsyntax.FunctionCallExpr`, read statically.

### 8. event, wire type and external source

```emod
event RoomReserved {
  type "com.hotel.reservations.room-reserved"
  source external "Partner Booking"
}
```
```hcl
event "RoomReserved" {
  type   = "com.hotel.reservations.room-reserved"
  source = external("Partner Booking")
}
```

### 9. view

```hcl
view "MemberLoansView" {
  subscribes = [CopyBorrowed, CopyReturned]
  fields { loanId = string }
}
```

### 10. automation

```emod
automation ConfirmationEmailReactor {
  on RoomReserved after "15m"
  reads PendingConfirmationsView
  command SendConfirmationEmail
  target context Notifications
}
```
```hcl
automation "ConfirmationEmailReactor" {
  on      = RoomReserved
  after   = "15m"
  reads   = PendingConfirmationsView
  command = SendConfirmationEmail

  target {
    context = Notifications
  }
}
```

The schedule form states `every` in place of `on`.

`after` becomes a sibling of `on` rather than a suffix on its line. Two parser
rules therefore move to the validator: "exactly one of `on` and `every`", and
"`after` needs `on`". The messages stay as they are.

### 11. translation

```hcl
translation "BookingComImport" {
  external_system = "Booking.com API"
  reads           = AvailableRoomsView
  command         = ImportExternalReservation

  event "ExternalReservationImported" {
    fields { externalRef = string }
  }
}
```

### 12. flow

The heredoc holds today's text, so the existing flow parser reads it unchanged.

```hcl
flow = <<-FLOW
  command -> event:    ReserveRoom -> RoomReserved
  command -> rejected: ReserveRoom -> RoomNotDoubleBooked
FLOW
```

Diagnostics stay precise. HCL gives the heredoc's `hcl.Range`, and a byte offset
inside it gives the line and column of any name, which is what the flow parser
already computes.

### 13. spec

The outcome kind becomes a call.

| Today | Proposed |
| --- | --- |
| `given []` | `given = []` |
| `given [CopyBorrowed]` | `given = [CopyBorrowed]` |
| `when BorrowCopy` | `when = BorrowCopy` |
| `then [CopyBorrowed, LoanOpened]` | `then = [CopyBorrowed, LoanOpened]` |
| `then rejected OneCopyPerLoan` | `then = rejected(OneCopyPerLoan)` |
| `then view MemberLoansView` | `then = view(MemberLoansView)` |
| `then command RecallCopy` | `then = command(RecallCopy)` |

A payload qualifies its reference, so it is a call with one object argument:

```hcl
spec "holds a room nobody has booked" {
  when = ReserveRoom({
    roomId    = "R-204"
    guestName = "Ada Lovelace"
  })
  then = [RoomReserved({
    reservationId = "RES-8814"
    guests        = 2
  })]
}
```

A short payload stays on one line:

```hcl
given = [RoomReserved({ roomId = "R-204", checkIn = "2026-04-17" })]
```

The three literal kinds, the type rules they satisfy and the two payload
diagnostics are unchanged.

### 14. DCB tags and decides_on

```hcl
event "DeskClaimed" {
  tags {
    desk   = deskId
    reader = memberId
  }
}

command "ClaimDesk" {
  decides_on {
    events = [DeskClaimed, DeskReleased]
    where  = (tag(desk, deskId) || tag(desk, altDeskId)) && !tag(reader, memberId)
  }
}
```

This is the one construct that gains. Grouping and precedence come from HCL, so
the predicate parser goes away. `and`, `or` and `not` become `&&`, `||` and `!`.
`tag` takes two positional arguments, because HCL has no named arguments.

---

## Size

The whole of `examples/all_patterns.emod`, converted and parsed:

| File | Today | Converted | Ratio |
| --- | --- | --- | --- |
| `examples/all_patterns.emod` | 373 | 387 | 1.04 |
| its first 80 lines | 80 | 84 | 1.05 |

For comparison, the literal HCL translation that gives every field its own block
takes the same 80 lines to 159, a ratio of 1.98. The decisions above are what
keep the file its current size.

## Internal Representation

The schema does not change, so the work is confined to the front end.

### Deleted

| Package | Lines |
| --- | --- |
| `internal/lexer` | 422 |
| `internal/parser` | 1,832 |
| `internal/formatter` | 637 |
| `editors/tree-sitter-emod` | the grammar and three query files |

### Replaced

- **A decoder**, reading `hclsyntax.Body` into the existing AST. It uses
  `hcl.AbsTraversalForExpr` for a reference, `hcl.ExprList` for a list, a type
  switch on `*hclsyntax.FunctionCallExpr` for the six call forms, a type switch
  on `*hclsyntax.BinaryOpExpr` and `*hclsyntax.UnaryOpExpr` for a predicate, and
  each attribute's `SrcRange` to keep declaration order.
- **A writer**, on `hclwrite`. It aligns `=` in place of aligning the name and
  type columns.

### Changed

- `internal/validator` (815) and `internal/linter` (1,127): the logic is
  unchanged. Three rules move here from the parser, and the DCB predicate is
  walked as an HCL expression rather than as an emod one.
- `internal/importer` (444) and the viewer's `emod-export.js`: they write
  through `hclwrite`. The round trip is otherwise unchanged.
- `editors/vscode`: it maps `.emod` to the HCL grammar and keeps the LSP client.
- `internal/lsp/completer.go` (454): rewritten. It decides what to offer by
  reading the characters before the cursor, not by reading the AST, and it keys
  on the old surface — a bare word at the head of a line to name the block, an
  arrow to spot a flow entry, a colon to spot a payload. Its own scanner
  (`lineTokens`, `findBlockKeyword`, `codeOutsideStringsAndComments`, about 150
  lines) gives way to a walk over `hclsyntax.LexConfig` tokens, which never
  fails and already marks strings, comments and heredocs. The tables it drives
  change with the surface.
- `internal/lsp/semantictokens.go`: a command, an event and a view become
  quoted labels, so three calls move from `addIdentifier` to `addQuoted`.

### Unchanged

`internal/ast` (425), `internal/diagnostic` (38), `internal/diagram` (3,517),
`internal/export` (1,535), `internal/glossary` (277), `internal/arrange` (274),
`internal/cue/schema.cue`, `internal/oracle` and `internal/pipeline`. In the
LSP, hover, go-to-definition and find-references read the AST and stand.

`ast.Position` stays as it is and is filled from HCL's positions, rather than
becoming an `hcl.Range`. Every reader of a position therefore keeps its code.

## Migration

### Phase 1 — read both

The decoder lands beside the existing parser. `emod` picks one by sniffing the
first token: an `=` after the leading word means HCL. Everything else is
untouched, and every existing file keeps working.

Acceptance: for every file in `examples/` and the test fixtures, the AST the
decoder produces equals the AST the parser produces.

### Phase 2 — write HCL

`emod fmt --hcl` writes the HCL form, which makes it a converter: read with the
old parser, write with `hclwrite`.

Acceptance: `emod export --format json` and `--format cue` give the same
output before and after conversion, once the positions are set aside. Line and
column numbers move with the text; nothing else may.

### Phase 3 — HCL only

Convert the 25 `.emod` files in the repository, delete the lexer, the parser,
the formatter and the tree-sitter grammar, and drop the sniffing.

### Phase 4 — the surface

Rewrite `docs/dsl-reference.md` (913 lines), point the VS Code extension at the
HCL grammar, and update the e2e suites.

## Verification

Every form in this proposal parses with `github.com/hashicorp/hcl/v2 v2.23.0`,
checked with a parse harness that reports which node each form became. The
converted `examples/all_patterns.emod` parses with no diagnostic, and its
construct counts match the original one for one: 13 commands, 5 events, 3 views,
2 automations, 1 translation, 15 specs, 1 trigger, 8 slices, 2 aggregates, 2
contexts and 6 invariants.

## Risks and Open Questions

- **Syntax errors change their wording.** HCL reports a malformed file in its
  own words, before the validator runs. emod keeps its wording for everything
  the decoder reaches, which is every rule the validator and the linter state.
- **Field order comes from source ranges.** A cty object sorts its keys, so the
  decoder reads order from each attribute's `SrcRange`. A renderer that shows
  fields in declaration order depends on this being right.
- **The heredoc is opaque to an editor.** A generic HCL editor highlights the
  `flow` block as text. The LSP still reports its diagnostics.
- **`hclwrite` decides the layout.** Comment placement and alignment follow
  `hclwrite`, so formatted output differs from today's in ways the formatter no
  longer controls.
- **Open: whether `emod migrate` stays.** A one-shot converter is enough for
  this repository. A published converter helps anyone who adopted the DSL
  already.

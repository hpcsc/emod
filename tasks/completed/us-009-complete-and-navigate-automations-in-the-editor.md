# US-009: Complete and navigate automations in the editor

## Progress
- [x] Task 1: See both homes a slice has from every LSP walker
- [x] Task 2: Describe `on`, `every` and `reads`, and name the pattern `trigger` and `automation` belong to
- [x] Task 3: Offer an automation's entries inside an automation block
- [x] Task 4: Offer event names after `on` and view names after `reads`
- [x] Task 5: Resolve an automation's `reads` to the view it names, both ways

---

## Story Reference

`user-stories/triggers-and-automations.md` → **US-009: Complete and navigate automations in the
editor** (ninth of eleven stories in "Triggers and Automations"). Design notes:
`docs/proposals/triggers-and-automations-proposal.md` — `:225-229` for the LSP surfaces, `:201` for the
entry order `emod fmt` fixes, `:105` for the two schedule forms `every` accepts.

**Dependencies, both landed on `main`:** US-001 (`automation` accepts `reads <ViewName>`) and US-003
(`automation` accepts `every "<expr>"`, exactly one of `on`/`every` required). US-002's `on` spelling
landed with them. So `ast.Automation` already carries `OnEvent`, `Schedule` and `Reads` with their
positions, and this story is the editor catching up to a language the parser already speaks.

**In scope:** the LSP server, and nothing else. Keyword hover for `on`, `every` and `reads`; rewritten
hover for `trigger` and `automation` naming the Event Modeling pattern each belongs to; the
automation-body keyword completion list; event-name completion after `on` and view-name completion
after `reads`; go-to-definition from an automation's `reads`; find-references on a view listing the
automations that read it. Carried along because the story's criteria cannot be honestly met without
them: the one-home slice walks in the LSP files, which make every feature in this story blind to a
`mode dcb` context's own slices, together with the ordinal-range keyword eligibility test that made
`on` and `every` invisible to hover whatever the description map said (both Task 1's); and the
`pendingKeyword` latch in `resolveContext` (`internal/lsp/completer.go:46`), which drops the cursor
back to the top-level list for the rest of a block after any line beginning with `command` or `event`
(verified: a cursor inside `automation … { on … / command … / ⟨here⟩ }` returns `model actor context`).

**Out of scope:** the `reads` edge (US-005); lane placement
(US-006); the palette (US-007); the `automation/missing-todo-list` rule (US-008) — this story adds no
diagnostic and no `RuleName`; the VS Code TextMate alternation
(`editors/vscode/syntaxes/emod.tmLanguage.json:63`) and `editors/tree-sitter-emod/queries/*.scm`
(US-010); `docs/dsl-reference.md`, `README.md` and `examples/` (US-011). No Go file outside
`internal/lsp` changes, and no `.emod` file, fixture, golden or corpus case in the tree is edited.

**Consequences of that boundary, decided.** Nine shapes the story does not spell out:

1. *The hover text for `trigger` describes the element's role, never its syntax.* The language has no
   kind slot — `ast.Trigger` (`internal/ast/ast.go:173-185`) carries none and the parser expects a
   quoted name straight after the keyword (`internal/parser/parser.go:553-591`) — while
   `docs/dsl-reference.md:286` still spells the header `trigger <Kind> "<name>"`. Hover text copied
   from the reference would therefore describe a shape `emod validate` rejects. `trigger` is the human
   entry point of the Command pattern, and that is what the text says. Task 2 pins the absence of the
   three kind words so the stale spelling cannot leak in.
2. *No task authors a new `trigger` declaration.* The two LSP test documents that need one already
   spell `trigger "MyTrigger"` (`internal/lsp/definition_test.go:38`,
   `internal/lsp/references_test.go:35`); Task 5 extends those documents' *automations* and leaves the
   trigger lines exactly as they are. `internal/test/fixtures.go` is likewise read, never edited.
3. *Do not write hover or completion text from `docs/dsl-reference.md`.* `tasks/learnings.md` records
   that the reference's Automation Pattern skeleton (`:322-340`) still documents the retired
   `trigger <EventName>` spelling and says "`trigger` and `command` are required" — a shape
   `emod validate` now rejects. The parser and `writeAutomation` are the authorities for what an
   automation accepts; US-011 owns fixing the reference.
4. *The automation completion list is the five entries the story names, in the order `emod fmt` writes
   them.* `writeAutomation` (`internal/formatter/formatter.go:347-356`) emits `on`, `every`, `reads`,
   `command`, `target context`; the list reads in that order so what an author picks arrives in the
   order it will be formatted into. `description` is omitted, matching every sibling list — the
   slice-level list (`completer.go:143`) omits it too, though a slice accepts one.
5. *`target context` is one completion item.* A bare `target` is never a valid entry, so completing it
   alone would offer a token the parser rejects.
6. *The value list is suppressed inside a `fields` block and nowhere else.* Keywords stay legal as field
   names, types and modifiers, so `id reads required` is a valid field line and a cursor after `reads `
   there must still see field types. Gating on "not `ctxFields`" is one rule that covers the collision
   while keeping `reads` working in all three blocks that spell it — automation, trigger and
   translation. Gating each keyword on its own block instead would need `resolveContext` to learn the
   trigger and translation bodies, which the proposal (`:227`) changes.
7. *A value list needs whitespace between the keyword and the cursor.* `on|` is a half-typed keyword and
   completes from the block's keyword list, which the client filters down; `on |` is a value position
   and completes event names. Without the rule, typing `o`, `n` would replace the keyword list with
   event names mid-word.
8. *Completion returns the whole list and lets the client filter by prefix.* This is what the existing
   lists already do and what LSP clients expect; a server-side prefix filter would also have to decide
   what a `CompletionList` with `IsIncomplete: false` means after a backspace.
9. *A command, event or view in a `mode dcb` context's own slice has a context but no aggregate, and
   hover says so.* The existing hover text is `**Command** in <Context> > <Aggregate>`; a slice with no
   aggregate drops the second segment rather than printing an empty one. This is a new output shape, so
   Task 1 asserts both on one model.

**Deliberately not fixed here.** Hover keys on the token's spelling, so a field named `every` inside a
`fields` block hovers as the `every` keyword. That is the behaviour every keyword already has — a field
named `reads` does the same today — and making it position-aware is a different change from the one
this story is asked for. Task 2 pins the behaviour so it reads as decided rather than discovered.
Likewise, `keywordDescriptions` gains entries for `on` and `every` only: `description`, `invariant`,
`spec`, `given`, `when`, `then`, `rejected`, `mode`, `tags`, `decides_on`, `where`, `and`, `or`, `not`,
`tag`, `events` and `emod` stay undescribed, and Task 2 pins one of them — `mode` — as still returning
nil, so the map is what decides and describing any of the others stays a one-line change. And a cursor
inside a `trigger`, `view` or `translation` block still returns the top-level list, as it does today —
Task 3 teaches `resolveContext` the automation body alone, because the proposal (`:227`) adds an entry
to a trigger's body that a later story owns.

**Learnings folded in** from `tasks/learnings.md`: keyword surfaces fan out past the lexer and parser,
so ask the lexer which keywords exist and never range over `Kind`; assert a short keyword with a
`\b`-bounded `require.Regexp`, since `on` hides inside `automatiOn`, `descriptiOn` and `cOntext`; a
slice has two homes; prefer a single structural assertion over a contains-loop, and delete the
narrower `Contains` leaf the new whole-list assertion subsumes rather than keeping both; an assertion
whose expected value comes from the code under test cannot fail; a second `require.Contains` on one
message is often shadowed by the first; a task's change-set assertion must name every file its own
patterns require it to change; acceptance criteria describe the working tree, and a commit-message
receipt is the commit author's obligation, never a criterion; `docs/dsl-reference.md` is the one
keyword surface no test reaches and still documents both the retired automation `trigger` spelling and
the retired trigger kind slot.

---

## Codebase Context

**Lexer.** `internal/lexer/token.go` declares `Kind` in one iota block with `KeywordOn` and
`KeywordEvery` appended last, and spells them in the `keywords` map (`:110-111`). `Keywords()` returns
the spellings sorted from that map and `Kind.IsKeyword()` is a lookup in the `keywordNames` inversion —
the eligibility test the LSP already asks.

**The shared model walk** (`internal/lsp/model.go`). `parseModel` (`:9`) scans and parses, keeping the
model and discarding the diagnostics. `scopedSlices` (`:23-36`) yields every slice as a `scopedSlice` —
the slice, its context, and its aggregate where it has one, `nil` for a slice a `mode dcb` context
declares directly — an aggregate's slices first, then the slices the context declares directly, in
declaration order. `newDeclaredNames` (`:57-81`) returns four *maps* from name to declaration position,
keyed `commandName`, `eventName`, `viewName`, `contextName`; being maps they carry no order.
`referencesIn` (`:95-130`) returns the ordered `[]nameRef` of every site that names something rather
than declaring it, and it is where each reference kind is stated once: per scoped slice, a view's
`subscribes` entries (`:104-110`), then each automation's `on`, `command` and `target context`
(`:111-115`), then each translation's `reads` and `command`, then a trigger's `reads`, then the flow
entries. An automation's `reads` is the one reference site the list omits, though `ast.Automation`
carries `Reads`/`ReadsPos` (`internal/ast/ast.go:210-211`) — verified: go-to-definition on it returns
nil, and find-references on a view an automation reads lists the trigger and the translation only.

**Hover** (`internal/lsp/hover.go`). `keywordDescriptions` (`:11-31`) maps nineteen spellings to text;
`trigger` (`:21`) reads "Defines a manual trigger for a slice", `automation` (`:23`) reads "Defines an
automation that triggers on an event and sends a command", `reads` (`:28`) reads "Defines the view a
trigger or translation reads from" — none of the three mentions a schedule, a view an automation reads,
or a pattern, and `on` and `every` have no entry at all. Eligibility is `tok.Type.IsKeyword()` (`:74`),
so a keyword the lexer knows becomes answerable the moment the map describes it, and an undescribed
keyword such as `mode` returns nil. `GetHover` (`:41`) walks `scopedSlices` for a command, event or view
definition name (`:53-69`), then falls through to the token scan (`:72-80`); `declaredIn` (`:85`) builds
the `<Context> > <Aggregate>` scope string, dropping the second segment where there is no aggregate. The
automation description is spelled in exactly two places outside `hover.go`:
`internal/lsp/hover_test.go:177` and `internal/lsp/server_test.go:994`.

**Completion** (`internal/lsp/completer.go`). `GetCompletions` (`:10`) never parses — `resolveContext`
(`:33`) is a line scanner over brace counts, `findBlockKeyword` (`:97`) recognises six spellings
(`context`, `aggregate`, `slice`, `command`, `event`, `fields`) and `completionsFor` (`:133`) maps a
`blockContext` to a fixed label slice, every item `KeywordCompletion`. Two mechanics matter: a
recognised keyword on a line with no `{` sets `pendingKeyword` (`:62`) and nothing clears it until a
later line opens a brace, so the terminal guard (`:87`) returns `ctxUnknown` — this is why a cursor
below an automation's `command <Name>` line completes top-level keywords; and an unrecognised keyword
with a brace pushes `ctxUnknown` (`:70-75`), which is why an automation body, a trigger body and a
translation body all complete top-level keywords today. Verified on the current tree: all three return
`model actor context`. `CompletionItem` (`internal/lsp/protocol.go:79`) carries `Label`, `Kind`,
`Detail` and `Documentation`; `EventCompletion` (`:73`) and `ClassCompletion` (`:57`) are the kinds
`GetSemanticTokens` already assigns events and views (`internal/lsp/semantictokens.go:57-63`). The
server advertises `" "` as a completion trigger character (`internal/lsp/server.go:99`), so a request
fires the moment an author types the space after `on`.

**Definition** (`internal/lsp/definition.go`). Thirty-one lines: `GetDefinition` (`:8`) parses, builds
`newDeclaredNames`, and returns the first `referencesIn` entry the cursor sits on (`:21-28`) whose name
the model declares. It names no construct itself — every reference kind it can navigate from is one
`referencesIn` states. `locationFor` (`position.go`) is the shared location builder and `at.onName` the
only hit test; it returns false for an empty name, so a construct that omits an entry needs no guard.

**References** (`internal/lsp/references.go`). Sixty-six lines over the same two helpers.
`GetReferences` (`:14`) resolves the cursor through `targetAt` (`:47`) — which checks the declarations
of the three `referenceTargetKinds` (`:7`) first, then the `referencesIn` entries — and then emits the
declaration followed by every `referencesIn` entry of the same kind and name, in list order (`:32-40`).
A site added to `referencesIn` therefore reaches the cursor resolution and the listing together.

**Semantic tokens** (`internal/lsp/semantictokens.go`). `GetSemanticTokens` walks `scopedSlices`;
events get `TokenTypeEvent` (`:115`) and views `TokenTypeClass` (`:118`), the two types
`tokenTypeIndex` (`:9`) maps to legend indices 1 and 2.

**Both-homes precedents.** `newModelIndex` (`internal/validator/validator.go:57`), `allSlicesIn`
(`internal/glossary/glossary.go:61-67`), `declaredSlices` (`internal/test/fixtures.go`) and
`exportedSlices` (`internal/export/export_test.go`) all visit both homes and agree with `scopedSlices`
on order — an aggregate's slices first, then the slices a `mode dcb` context declares directly — so a
hand-transcribed expectation reads the same in every package.

**LSP tests.** All `_test.go` files in `internal/lsp` are `//go:build unit`, package `lsp_test`, one
umbrella `Test<Function>` per file with a `const testDoc` and a `posIn` helper that resolves a substring
to 0-based coordinates *within the first occurrence of a container substring*. `internal/lsp/server_test.go`
(`:114`) is the over-the-wire umbrella with a `t.Run` group per LSP method — `completion` (`:450`),
`definition` (`:591`), `references` (`:737`), `hover` (`:891`). In `references_test.go`,
`requireLocation` (`:75`) is a find-in-list check that cannot see a missing or extra entry, and
`locationOf` (`:94`) builds the whole `lsp.Location` a named reference should be reported as — the
builder the whole-list `require.Equal` leaves use.

**Shared fixtures this story reads.** `test.AutomationReadsLibraryLending`
(`internal/test/fixtures.go:578`) is the model with the shape every task needs. `MemberLoansView` is
declared in the "Review Member Loans" slice of the aggregate "Loan" (`:610`) and read twice: by
`RecallOverdueCopy` (`:650`), an automation in another slice of the same aggregate, and by
`RemindReaderOfLoans` (`:729`), an automation in a slice the `mode dcb` context "Reading Room" declares
directly — a read that crosses contexts. `DeskOccupancyView` is declared directly on that DCB context
(`:714`) and read by `FreeDeskAtClosing` (`:724`) in another of its own slices. `DeskClaimed` is
declared in a DCB slice, subscribed by `DeskOccupancyView` and named by an automation's `on`; the
trigger "Lending Desk" (`:588`) reads `AvailableCopiesView`, a name the model never declares — the
unresolved-reference case. `test.AutomationScheduleLibraryLending` (`:944`) is the same shape with
schedules. `internal/test/models.go` holds the parsing accessors.

**Not touched, deliberately.** `internal/lexer`, `internal/parser`, `internal/ast`,
`internal/formatter`, `internal/validator`, `internal/linter`, `internal/export`, `internal/importer`,
`internal/cue`, `internal/diagram`, `internal/glossary`, `internal/cli`, `internal/viewer`,
`internal/test`, `editors/`, `docs/`, `examples/`, `e2e/`.

---

## Tasks

### Task 1: See both homes a slice has from every LSP walker

**Behavior:** hover, go-to-definition, find-references and semantic tokens resolve names declared in a
slice that hangs directly off a `mode dcb` context, not only names in a slice nested in an aggregate.
One walk in `internal/lsp` yields every slice with the context it belongs to and the aggregate it hangs
off where it has one, and the four files use it instead of each nesting its own loop. Hover on a
construct in an aggregate-less slice names its context and omits the aggregate segment; every name in an
aggregate's slice resolves exactly as it does today.

**Acceptance Criteria:**
- [ ] Hovering the name of a command, an event and a view declared directly on a `mode dcb` context
      returns hover text naming that context, asserted on `test.AutomationReadsLibraryLendingModel`'s
      source; the same three constructs in an aggregate's slice of the same model still return the
      `<Context> > <Aggregate>` text they return today, asserted in the same subtest so the two shapes
      are proved together
- [ ] The aggregate-less hover text contains no empty segment, no trailing separator and no placeholder
      standing in for the missing aggregate — asserted with one `require.Equal` on the whole string, not
      a `Contains` on the context name
- [ ] Go-to-definition from a `subscribes` entry naming an event declared in another slice of the same
      `mode dcb` context returns that event's declaration position
- [ ] Go-to-definition from an automation's `command` naming a command declared in a `mode dcb`
      context's own slice returns that command's declaration position
- [ ] Find-references on an event declared in a `mode dcb` context's own slice returns its declaration,
      the `subscribes` entry that names it and the automation `on` entry that names it, asserted with
      one `require.Equal` against the full expected location list rather than a find-in-list loop
- [ ] `GetSemanticTokens` over the same model emits a token for each command, event and view declared
      directly on the `mode dcb` context, in addition to those it emits today — asserted by the count of
      tokens in the delta-encoded data, together with the position of at least one token that only the
      second home can produce
- [ ] The walk visits an aggregate's slices before the slices a context declares directly, matching
      `allSlicesIn` (`internal/glossary/glossary.go:60`), `declaredSlices` and `exportedSlices`, so a
      transcribed location list reads in the same order it would in any other package — pinned by the
      full-list assertion above
- [ ] Every existing subtest in `internal/lsp/hover_test.go`, `definition_test.go`, `references_test.go`
      and `semantictokens_test.go` passes with its expectations unedited: none of their documents
      declares a `mode dcb` context, so no byte of their expected output may move
- [ ] `rg 'ctx\.Aggregates' internal/lsp` returns hits only from the extracted walk — no second copy of
      the nesting survives in `hover.go`, `definition.go`, `references.go` or `semantictokens.go`

**Affected Files/Modules:**
- `internal/lsp/hover.go` — the definition walk (`:64-84`) and the three `hoverFor*` builders
  (`:106`, `:117`, `:139`), which take the aggregate by value today
- `internal/lsp/definition.go` — the definition maps (`:32-47`) and the reference walk (`:55-129`)
- `internal/lsp/references.go` — all three walks (`:31-45`, `:84-177`, `:203-255`)
- `internal/lsp/semantictokens.go` — the entry walk (`:81-134`)
- `internal/lsp/hover_test.go`, `definition_test.go`, `references_test.go`, `semantictokens_test.go`

**Patterns to Follow:**
- `allSlicesIn` (`internal/glossary/glossary.go:60-67`) is the shape and the ordering; `newModelIndex`
  (`internal/validator/validator.go:71-76`) is the same walk where the aggregate is not needed
- `tasks/learnings.md` "A slice has two homes, and much of the repo still walks only one", which names
  `definition.go:33`, `hover.go:65`, `references.go:32`/`:85`/`:204` and `semantictokens.go:91` as the
  live instances
- `tasks/learnings.md` "De-duplicate before a fan-out edit, and land the de-duplication with proof" —
  carry the differential receipt (the same model resolving names in both homes) rather than asserting
  the extraction is equivalent
- `tasks/learnings.md` "Name an extracted helper after the contract its callers rely on" — the
  postcondition callers depend on is *every* slice, with its context, and its aggregate only where one
  exists; a name promising an aggregate per slice would mislead
- `tasks/learnings.md` "Prefer a single structural assertion" — `requireLocation`
  (`internal/lsp/references_test.go:74`) is a find-in-list check and cannot catch a missing entry; the
  new leaves assert the whole list
- `test.AutomationReadsLibraryLendingModel(t)` (`internal/test/models.go:37`) and the source constant
  behind it are read as input; `internal/test/fixtures.go` is not edited

**Testable:** Yes — through `lsp.GetHover`, `lsp.GetDefinition`, `lsp.GetReferences` and
`lsp.GetSemanticTokens`, all exported.

**Verification:** `go test -tags unit ./internal/lsp/...`; `go build ./...`.

**Depends on:** None

---

### Task 2: Describe `on`, `every` and `reads`, and name the pattern `trigger` and `automation` belong to

**Behavior:** hovering `on` or `every` inside an automation describes what each declares, and hovering
`reads` describes the view an automation, trigger or translation consults. Hovering `trigger` or
`automation` names the Event Modeling pattern the element belongs to, describing its role rather than
its current syntax.

**Acceptance Criteria:**
- [ ] Hovering `on` in `automation <Name> { on <Event> … }` returns markdown describing the event that
      activates the automation; hovering `every` in an automation that declares one returns markdown
      describing the cadence and naming both accepted forms — a duration and a five-field cron
      expression — asserted on one document carrying an event-activated and a scheduled automation side
      by side, so neither is checked in isolation
- [ ] The `automation` hover text names the Automation pattern, names the view the automation reads, and
      names both activation forms; `on` and `every` are each asserted present with a `\b`-bounded
      `require.Regexp`, since `on` occurs inside `automation`, `description` and `context`
- [ ] The `automation` hover text no longer states that an automation triggers on an event and sends a
      command as its only shape — the two sites spelling the current string
      (`internal/lsp/hover_test.go:177`, `internal/lsp/server_test.go:994`) move to the new wording, and
      `rg 'triggers on an event and sends a command' --glob '!docs/**' --glob '!tasks/**'` returns
      nothing
- [ ] The `trigger` hover text names the Command pattern and describes the element as the human entry
      point into a slice
- [ ] The `trigger` hover text contains none of `UI`, `Schedule` or `Processor`, and describes no header
      shape with a kind between the keyword and the name — the language has no such slot, though
      `docs/dsl-reference.md:286` still prints one
- [ ] The `reads` hover text names automations alongside triggers and translations
- [ ] Hovering `model` still returns its current description, unedited, and hovering `mode` — a keyword
      the lexer knows (`internal/lexer/token.go:93`) and this task leaves undescribed — still returns
      nil, asserted beside a described keyword so "answers for what it describes" and "stays silent
      otherwise" are proved together
- [ ] A `fields` block declaring a field named `every` hovers that field's name as the `every` keyword,
      returning the same markdown as the `every` entry of an automation in the same document — the
      behaviour a field named `reads` already has, asserted so the shared spelling is a stated decision
      rather than an accident
- [ ] Over the wire, `textDocument/hover` at a position on `on` and at a position on `every` in an open
      document returns the same markdown as the direct call, added to the `hover` group in
      `internal/lsp/server_test.go` (`:891`)

**Affected Files/Modules:**
- `internal/lsp/hover.go` — `keywordDescriptions` (`:11-31`) alone; the eligibility test at `:74` is
  already `tok.Type.IsKeyword()`, so a described keyword the lexer knows is answerable without touching
  `GetHover`
- `internal/lsp/hover_test.go` — the keyword leaves (`:155`, `:163`) and the expectation at `:177`
- `internal/lsp/server_test.go` — the `hover` group (`:891`) and the expectation at `:994`

**Patterns to Follow:**
- `tasks/learnings.md` "Keyword surfaces fan out past the lexer, parser and tree-sitter grammar" — this
  task is the LSP half of that fan-out; `keywordDescriptions` is the only surface left in `hover.go`
  that decides whether a keyword answers
- `tasks/learnings.md` "Assert a short keyword in a diagnostic with a `\b`-bounded `require.Regexp`" —
  the same hazard applies to asserting hover prose
- `tasks/learnings.md` "A second `require.Contains` on one message is often shadowed by the first" —
  before adding a needle to a string that already has one, check it is not inside the earlier needle
- `docs/dsl-reference.md:276-350` is the authority for the four pattern names (Command, View,
  Automation, Translation) and for `trigger` belonging to the Command pattern; its Automation Pattern
  *skeleton* (`:322-340`) and its trigger header (`:286`) are stale and must not be copied — the
  skeleton still spells the rejected `trigger <EventName>` entry and the header a kind slot the parser
  no longer accepts
- `writeAutomation` (`internal/formatter/formatter.go:347-357`) and `parseAutomationEntry`
  (`internal/parser/parser.go:1089`) are what an automation actually accepts; the two schedule forms
  `every` takes are in `docs/proposals/triggers-and-automations-proposal.md:105`

**Testable:** Yes — through `lsp.GetHover` and the server's `textDocument/hover` handler.

**Verification:** `go test -tags unit ./internal/lsp/...`.

**Depends on:** 1

---

### Task 3: Offer an automation's entries inside an automation block

**Behavior:** a cursor inside `automation <Name> { … }` completes the five entries an automation
accepts, in the order `emod fmt` writes them, instead of the top-level keywords it offers today. The
list survives a preceding reference line: an automation's `command <Name>` line no longer drops every
later cursor in the block back to the top-level list.

**Acceptance Criteria:**
- [ ] A cursor on a blank line inside an otherwise empty `automation <Name> { }` returns exactly
      `on`, `every`, `reads`, `command`, `target context`, in that order, each with
      `lsp.KeywordCompletion` kind — asserted with one `require.Equal` against the full ordered label
      slice, in the shape `extractLabels` (`internal/lsp/completer_test.go:144`) already supports
- [ ] `target context` arrives as a single item; no returned label is a bare `target`
- [ ] A cursor on a blank line below `command <CommandName>` inside the same automation body returns the
      same five labels — today it returns `model`, `actor`, `context`, because a recognised keyword on a
      braceless line latches and is never cleared
- [ ] A cursor on a blank line inside a `slice` block, below a fully closed `automation … { … }` block,
      still returns the slice-level list — the brace-close accounting is unchanged
- [ ] The slice-level list still returns `command`, `event`, `trigger`, `view`, `automation`,
      `translation`, `flow`, and `git diff` moves no existing expectation in
      `internal/lsp/completer_test.go` — every change to that file is an added subtest
- [ ] A cursor inside a `trigger`, `view` or `translation` block still returns the top-level list,
      asserted beside the automation-body leaf so the asymmetry is a recorded decision rather than an
      oversight — the proposal (`:227`) adds an entry to a trigger's body that a later story owns
- [ ] Over the wire, `textDocument/completion` at a position inside an automation body of an open
      document returns those five labels, added to the `completion` group in
      `internal/lsp/server_test.go` (`:450`)

**Affected Files/Modules:**
- `internal/lsp/completer.go` — the `blockContext` constants (`:21-29`), `resolveContext`'s
  `pendingKeyword` handling (`:44-89`), `findBlockKeyword` (`:97-131`) and `completionsFor` (`:133-158`)
- `internal/lsp/completer_test.go` — a group beside `"slice block"` (`:60`)
- `internal/lsp/server_test.go` — the `completion` group (`:450`)

**Patterns to Follow:**
- The existing groups in `internal/lsp/completer_test.go` fix the shape: one `t.Run` per block, a raw
  string document with a `// cursor here` marker, and `require.Equal` on the full ordered label slice
- `writeAutomation` (`internal/formatter/formatter.go:347-357`) fixes the entry order; `description` is
  omitted to match the sibling lists, which omit it too
- `tasks/learnings.md` "Name an extracted helper after the contract its callers rely on" — if the latch
  is replaced rather than patched, the replacement's postcondition (what a braceless keyword line means
  for the *next* line) is the thing to name
- `parseAutomationEntry` (`internal/parser/parser.go:1089`) is the authority for which entries an automation
  accepts; the block imposes no order and no arity beyond the exactly-one-of rule over `on` and `every`,
  and completion imposes none either — both appear in the list
- Value completion after a keyword is Task 4's; this task changes only which keyword list is chosen

**Testable:** Yes — through `lsp.GetCompletions` and the server's `textDocument/completion` handler.

**Verification:** `go test -tags unit ./internal/lsp/...`.

**Depends on:** None

---

### Task 4: Offer event names after `on` and view names after `reads`

**Behavior:** with the cursor in the value position of an entry line, completion offers the names the
model declares rather than keywords — the events for `on`, the views for `reads`. Names come from
everywhere in the model, both homes a slice has included, and from a document that does not yet parse
cleanly, because the line being typed is itself incomplete.

**Acceptance Criteria:**
- [ ] With the cursor after `on ` on an entry line of an automation body, the returned items are the
      event names the model declares, in declaration order across both slice homes, each with
      `lsp.EventCompletion` kind, and no keyword item appears among them — asserted with one
      `require.Equal` against the full ordered label slice
- [ ] With the cursor after `reads ` on an entry line of an automation body, the returned items are the
      view names the model declares, each with `lsp.ClassCompletion` kind — the kinds match what
      `GetSemanticTokens` already assigns events (`internal/lsp/semantictokens.go:115`) and views
      (`:118`), so one construct does not read as two things
- [ ] The same view list is returned with the cursor after `reads ` inside a `trigger` block and inside
      a `translation` block — every block that spells `reads` names a view
- [ ] For a document carrying a `mode dcb` context, the view list contains both a view declared in an
      aggregate's slice and a view declared directly on the context, and the event list likewise —
      asserted against a hand-written expected list, never against names re-derived from the model under
      test
- [ ] An automation whose `on` entry has nothing after it still completes: the document does not parse
      cleanly, and the event list is produced anyway
- [ ] With the cursor immediately after `on` and no space, the block's keyword list is returned, not the
      event list — a half-typed keyword is not a value position
- [ ] Inside a `fields` block, a field line spelled `id reads required` with the cursor after `reads `
      returns the field types and modifiers, not view names — keywords stay legal as field names, types
      and modifiers
- [ ] With the cursor after `command ` and after `target context ` in an automation body, the block's
      keyword list is returned unchanged — this story adds value completion for `on` and `reads` only
- [ ] A document declaring no view at all returns an empty item list after `reads `, not the keyword
      list and not a nil list, so an author sees "nothing to read" rather than a wrong suggestion
- [ ] Over the wire, `textDocument/completion` at a position after `on ` returns the event names of the
      open document, added to the `completion` group in `internal/lsp/server_test.go` (`:450`)

**Affected Files/Modules:**
- `internal/lsp/completer.go` — `GetCompletions` (`:10`), which parses no document today, and
  `completionsFor` (`:133`)
- `internal/lsp/completer_test.go` — a group beside the block groups
- `internal/lsp/server_test.go` — the `completion` group (`:450`)

**Patterns to Follow:**
- `parseModel` (`internal/lsp/model.go:9`) is the one parse-then-walk opening every other LSP entry
  point uses (`hover.go:46`, `definition.go:13`, `references.go:19`, `semantictokens.go`); the parser is
  error-tolerant and returns a model for a half-written document
- `scopedSlices` (`internal/lsp/model.go:23-36`) supplies the names from both homes in declaration
  order; a fresh nested loop here would re-fork the shape it already carries. `newDeclaredNames`
  (`:57-81`) cannot serve this criterion — it returns maps, and a map has no order to assert against
- `tasks/learnings.md` "An assertion whose expected value comes from the code under test is the
  recurring review finding" — write the expected name list by hand from the test document, never by
  walking the parsed model in the test
- `tasks/learnings.md` "Prefer a single structural assertion" — one `require.Equal` on the ordered label
  slice covers length, contents and order together
- `CompletionItemKind` values are in `internal/lsp/protocol.go:48-76` — `ClassCompletion` (`:57`),
  `KeywordCompletion` (`:64`), `EventCompletion` (`:73`)
- The server already advertises `" "` as a trigger character (`internal/lsp/server.go:99`), so no
  capability changes

**Testable:** Yes — through `lsp.GetCompletions` and the server's `textDocument/completion` handler.

**Verification:** `go test -tags unit ./internal/lsp/...`.

**Depends on:** 1, 3

---

### Task 5: Resolve an automation's `reads` to the view it names, both ways

**Behavior:** an automation's `reads` entry becomes a reference site like every other one the model
states, so go-to-definition on the view name navigates to that view's declaration wherever in the model
it is declared, and find-references on a view returns the automations that read it alongside the
triggers and translations that do — with the cursor on the declaration or on any one of those
references. The automation's `on` event and its `command` keep navigating and keep listing as they do
today, and a command's and an event's references are unchanged.

**Acceptance Criteria:**
- [ ] Go-to-definition with the cursor on the view name in an automation's `reads` entry returns the
      location of that view's declaration, with the range starting at the declaration's name and ending
      at name start plus name length
- [ ] The cursor on the same automation's `on` event and on its `command` return their declarations —
      asserted in the same subtest group as the `reads` jump, so the story's three jumps are proved
      together rather than two of them assumed
- [ ] The jump resolves each shape `test.AutomationReadsLibraryLending` declares, asserted on its
      source: `RecallOverdueCopy` reading `MemberLoansView`, declared in another slice of the same
      aggregate; `RemindReaderOfLoans`, in a slice the `mode dcb` context declares directly, reading
      that same view across the context boundary; and `FreeDeskAtClosing` reading `DeskOccupancyView`,
      declared directly on that context
- [ ] An automation whose `reads` names a view the model does not declare returns nil from
      go-to-definition, and an automation whose `reads` resolves returns a location — both asserted on
      one document carrying the two automations, so the nil is not satisfiable by resolving nothing
- [ ] The cursor on the `reads` keyword itself, rather than on the value, returns nil — navigation is
      from the reference, not from the entry
- [ ] Find-references with the cursor on a view's declaration name returns, in one `require.Equal`
      against the full ordered list, the declaration followed by every site naming it in the order
      `referencesIn` (`internal/lsp/model.go:95-130`) yields — within a slice, the automations' `reads`
      before the translations' `reads` before the trigger's `reads`; across slices, an aggregate's
      slices before the slices a `mode dcb` context declares directly
- [ ] Find-references with the cursor on an automation's `reads` value returns that same full list, so
      the reference resolves in both directions
- [ ] On `test.AutomationReadsLibraryLending`'s source, references to `MemberLoansView` return its
      declaration, `RecallOverdueCopy`'s `reads` and `RemindReaderOfLoans`'s `reads`; references to
      `DeskOccupancyView` return its declaration and `FreeDeskAtClosing`'s `reads`
- [ ] On one document declaring two views, one read by an automation and one read by nothing, the first
      returns its declaration plus the automation's `reads` and the second returns its declaration
      alone — asserted in the same leaf, so "lists what reads it" and "lists nothing else" are proved
      against the same walk
- [ ] The `views` group of `internal/lsp/references_test.go` calls `requireLocation` nowhere: the
      declaration-cursor leaf that asserts the full list subsumes the find-in-list leaf it replaces, and
      the replaced leaf is gone rather than kept beside it
- [ ] References on a command still return the declaration, the flow entry, the automation `command`
      and the translation `command`, and references on an event still return the declaration, the
      `subscribes` entry, the automation `on` and the flow entry — `git diff` shows no edit to the
      asserted counts or locations of the `events` and `commands` groups
- [ ] The production change is confined to `referencesIn` in `internal/lsp/model.go`: `git diff` shows
      no edit to `internal/lsp/definition.go` or `internal/lsp/references.go`, both of which read the
      list it returns
- [ ] The automations in `internal/lsp/definition_test.go`'s `testDoc` (`:28-32`) and
      `internal/lsp/references_test.go`'s `testDoc` (`:25-29`) each gain a `reads` entry naming the view
      the document already declares, so one document carries an automation, a translation and a trigger
      reading the same view; the `trigger "MyTrigger"` blocks (`:38`, `:35`) are left exactly as written
- [ ] Every other subtest in `internal/lsp/definition_test.go` and `internal/lsp/references_test.go`
      passes with its cursor still landing on the entry its name claims, the translation-`reads` leaves
      included — `posIn` resolves the *first* occurrence of its container, and an added automation
      `reads OrderView` line precedes the translation's
- [ ] The `references` group in `internal/lsp/server_test.go` (`:737`) gains a leaf returning, over the
      wire, the locations for a view an automation reads, and the `definition` group (`:591`) still
      passes

**Affected Files/Modules:**
- `internal/lsp/model.go` — the automation loop in `referencesIn` (`:111-115`), which states `on`,
  `command` and `target context` and not `reads`
- `internal/lsp/definition_test.go` — `testDoc` (`:18`), the automation leaves, and the
  translation-`reads` leaf (`:133-137`) whose container needs re-anchoring
- `internal/lsp/references_test.go` — `testDoc` (`:15`), the `views` group (`:255-284`), and
  `locationOf` (`:94`), which the new whole-list leaves use in place of `requireLocation`
- `internal/lsp/server_test.go` — the `references` group (`:737`)

**Patterns to Follow:**
- The translation entries in the same loop (`internal/lsp/model.go:116-119`) already state a `reads`
  value as a `viewName` reference and are the line-for-line shape; the trigger's (`:120-122`) is its
  single-valued sibling. `GetDefinition` iterates the returned list and `GetReferences` uses it for both
  `targetAt` and the listing, so one statement serves both features — no branch is added to either file
- The entry sits in the automation loop where `emod fmt` writes it, after `on` and before `command`
  (`writeAutomation`, `internal/formatter/formatter.go:347-357`), so the list reads in the order an
  author sees the block
- `add` carries no empty-name guard because `at.onName` returns false for an empty name and a target
  name is never empty — the same reason the sibling entries need none
- `tasks/learnings.md` "Prefer a single structural assertion" and "Strengthening a test to a
  whole-sequence `require.Equal` means deleting the subtest it subsumes" — the whole-list leaf replaces
  the `Len` + `requireLocation` leaf for the same cursor rather than joining it; fold any input flavour
  only the old leaf carried into the new document
- `tasks/learnings.md` "An assertion whose expected value comes from the code under test is the
  recurring review finding" — the expected location list is transcribed from the test document by hand
  through `locationOf`, never gathered from the parsed model
- `tasks/learnings.md` "Only an automation's `reads` resolves; a trigger's and a translation's must stay
  unchecked" is about the *validator*, not navigation — the LSP resolves all three and returns nil for
  an unresolved name, which is why the undeclared-view criterion above is a nil and not a diagnostic
- `test.AutomationReadsLibraryLending` (`internal/test/fixtures.go:578`) is read as input; no `.emod`
  file, fixture, golden or corpus case is edited, and drawing a `reads` edge in the diagram outputs is
  US-005's

**Testable:** Yes — through `lsp.GetDefinition`, `lsp.GetReferences` and the server's
`textDocument/definition` and `textDocument/references` handlers.

**Verification:** `go test -tags unit ./internal/lsp/...`; `go build ./...`.

**Depends on:** 1

---

## Summary

**Five tasks**, ordered dependency-first and, within that, by how much of the story each unblocks.

Task 1 comes first because every feature that follows reads the model, and adding those reads onto the
one-home loops that stood in each LSP file would fork that shape three more times. It is also the only
task that changes an existing output shape (hover text for a construct with no aggregate), so it lands
alone; it leaves behind `internal/lsp/model.go`, where the parse, the slice walk, the declaration maps
and the reference list are each stated once, and every task after it edits one statement in that file
or none at all.

Tasks 2 and 3 are independent of each other and of everything after them: Task 2 owns `hover.go`, Task
3 owns the keyword half of `completer.go`. Task 4 follows Task 3 because it edits the same file and
needs the automation-body determination to exist before it can decide that a value position sits inside
one, and follows Task 1 because its name lists must reach both slice homes and must come out in
declaration order, which only `scopedSlices` gives. Task 5 rests on Task 1 alone and is the smallest
production change in the story — a single reference statement, which `GetDefinition` and
`GetReferences` both read — carrying the navigation and the listing assertions the story asks for.

**Story criteria coverage:**

| Story criterion | Task |
|---|---|
| Completion inside an automation block offers `on`, `every`, `reads`, `command`, `target context` | 3 |
| Completion after `on` offers event names; after `reads`, view names | 4 |
| Hovering `on`, `every` or `reads` describes what each does | 2 |
| Hovering `trigger` or `automation` states which Event Modeling pattern the element belongs to | 2 |
| Go-to-definition from an automation's `on` event, its `reads` view, and its `command` | 5 (`reads` added; `on` and `command` pinned) |
| Find-references on a view lists the automations and triggers that read it | 5 |

Carried along, not stated by the story: the both-homes slice walk and the lexer-backed keyword
eligibility test (1), without which every criterion above is silently false inside a `mode dcb` context
and `on` and `every` have no hover whatever the description map says; and the `pendingKeyword` latch
(3), which is why a cursor below an automation's `command` line completes top-level keywords.

**One reference statement, two features.** An automation's `reads` is missing from exactly one place —
the automation loop of `referencesIn` — and both go-to-definition and find-references read that list,
so the jump and the listing arrive together and cannot be split across two tasks or two files. Task 5
therefore owns both, and its criteria assert the definition jumps and the whole reference list against
the order `referencesIn` yields rather than against any per-file collection order.

**Trigger declarations.** Task 2's hover text for `trigger` describes the element's role and is pinned
against naming `UI`, `Schedule` or `Processor` or describing a kind slot — the language has none, and
`docs/dsl-reference.md:286` still prints one. Task 5 extends the automations of the two LSP test
documents and leaves their `trigger "MyTrigger"` headers untouched. No task writes a new trigger
declaration.

**Deferred to later stories in the feature:** the `reads` edge (US-005), lane placement (US-006), the
palette (US-007), the `automation/missing-todo-list` rule (US-008), the VS Code TextMate alternation and
the tree-sitter highlight queries (US-010), and `docs/dsl-reference.md` — whose Automation Pattern
skeleton still documents the retired `trigger <EventName>` spelling and whose Command Pattern skeleton
still documents the retired trigger kind slot — along with `README.md` and `examples/` (US-011).

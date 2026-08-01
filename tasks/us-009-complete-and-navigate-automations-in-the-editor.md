# US-009: Complete and navigate automations in the editor

## Progress
- [ ] Task 1: See both homes a slice has from every LSP walker
- [ ] Task 2: Describe `on`, `every` and `reads`, and name the pattern `trigger` and `automation` belong to
- [ ] Task 3: Offer an automation's entries inside an automation block
- [ ] Task 4: Offer event names after `on` and view names after `reads`
- [ ] Task 5: Jump from an automation's `reads` to the view it names
- [ ] Task 6: List the automations that read a view among its references

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
them: the `isKeyword` ordinal range in `internal/lsp/hover.go:37`, which today makes `on` and `every`
invisible to hover no matter what the description map says (verified: `GetHover` returns nil on both);
the `pendingKeyword` latch in `resolveContext` (`internal/lsp/completer.go:46`), which drops the cursor
back to the top-level list for the rest of a block after any line beginning with `command` or `event`
(verified: a cursor inside `automation … { on … / command … / ⟨here⟩ }` returns `model actor context`);
and the one-home slice walks in all four LSP files, which make every feature in this story blind to a
`mode dcb` context's own slices.

**Out of scope:** dropping the trigger kind slot (US-004); the `reads` edge (US-005); lane placement
(US-006); the palette (US-007); the `automation/missing-todo-list` rule (US-008) — this story adds no
diagnostic and no `RuleName`; the VS Code TextMate alternation
(`editors/vscode/syntaxes/emod.tmLanguage.json:63`) and `editors/tree-sitter-emod/queries/*.scm`
(US-010); `docs/dsl-reference.md`, `README.md` and `examples/` (US-011). No Go file outside
`internal/lsp` changes, and no `.emod` file, fixture, golden or corpus case in the tree is edited.

**Consequences of that boundary, decided.** Nine shapes the story does not spell out:

1. *The hover text for `trigger` describes the element's role, never its syntax.* US-004 removes the
   kind slot, so hover text naming `UI`, `Schedule` or `Processor`, or spelling the header as
   `trigger <Kind> "<name>"`, would be wrong one story later. `trigger` is the human entry point of the
   Command pattern (`docs/dsl-reference.md:280`), and that is what the text says. Task 2 pins the
   absence of the three kind words so the collision cannot land silently.
2. *No task authors a new `trigger` declaration.* The two LSP test documents that need one already
   spell `trigger manual "MyTrigger"` (`internal/lsp/definition_test.go:37`,
   `internal/lsp/references_test.go:34`); Tasks 5 and 6 extend those documents' *automations* and leave
   the trigger lines exactly as they are, so US-004 migrates two occurrences it was already going to
   migrate rather than four. `internal/test/fixtures.go` is likewise read, never edited.
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
   trigger and translation bodies, which US-004 and the proposal both change.
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
`tag`, `events` and `emod` stay undescribed, and Task 2's fix to the eligibility test is what makes
describing any of them a one-line change later rather than a silent no-op. And a cursor inside a
`trigger`, `view` or `translation` block still returns the top-level list, as it does today — Task 3
teaches `resolveContext` the automation body alone, because US-004 rewrites the trigger header and the
proposal (`:227`) adds an entry to its body.

**Learnings folded in** from `tasks/learnings.md`: keyword surfaces fan out past the lexer and parser,
and `isKeyword` is an ordinal range that silently excludes every `Kind` appended after
`KeywordExternal` — the load-bearing one for this story; ask the lexer which keywords exist and never
range over `Kind`; assert a short keyword with a `\b`-bounded `require.Regexp`, since `on` hides inside
`automatiOn`, `descriptiOn` and `cOntext`; a slice has two homes and every LSP walker still walks one;
de-duplicate before a fan-out edit and land the de-duplication with proof; name an extracted helper
after the contract its callers rely on; prefer a single structural assertion over a contains-loop; an
assertion whose expected value comes from the code under test cannot fail; a second `require.Contains`
on one message is often shadowed by the first; acceptance criteria describe the working tree, and a
commit-message receipt is the commit author's obligation, never a criterion; `docs/dsl-reference.md` is
the one keyword surface no test reaches and still documents the retired automation `trigger` spelling.

---

## Codebase Context

**Lexer.** `internal/lexer/token.go` declares `Kind` in one iota block (`:9-63`) with `KeywordOn` and
`KeywordEvery` appended last (`:48-49`), well past `KeywordExternal` (`:30`). `Keywords()` returns the
spellings sorted from the `keywords` map and `Kind.IsKeyword()` is a lookup in the `keywordNames`
inversion — the correct eligibility test, already exported and already used by
`Instance.checkIdentifierLike`.

**Hover** (`internal/lsp/hover.go`). `keywordDescriptions` (`:13-33`) maps nineteen spellings to text;
`trigger` (`:23`) reads "Defines a manual trigger for a slice", `automation` (`:25`) reads "Defines an
automation that triggers on an event and sends a command", `reads` (`:30`) reads "Defines the view a
trigger or translation reads from" — none of the three mentions a schedule, a view an automation reads,
or a pattern. `isKeyword` (`:37`) is `k >= lexer.KeywordModel && k <= lexer.KeywordExternal`. `GetHover`
(`:48`) walks `ctx.Aggregates` → `agg.Slices` only (`:64-84`), returning contextual hover for a command,
event or view definition name, then falls through to a token scan for keywords (`:87-101`).
`hoverForCommand`/`hoverForEvent`/`hoverForView` (`:106`, `:117`, `:139`) all interpolate
`ctx.Name` and `agg.Name`. The old automation string is spelled in exactly two places outside
`hover.go`: `internal/lsp/hover_test.go:121` and `internal/lsp/server_test.go:994`.

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

**Definition** (`internal/lsp/definition.go`). `GetDefinition` (`:14`) builds four name→position maps
from `ctx.Aggregates` → `agg.Slices` (`:32-47`), then checks five reference kinds in order: view
`subscribes` (`:60`), automation `on`/`command`/`target context` (`:73-89`), translation
`reads`/`command` (`:92`), trigger `reads` (`:106`) and flow command/event (`:115`). An automation's
`reads` is absent from both halves — verified: the cursor on it returns nil. `cursorOnName` (`:136`) and
`locationFor` (`:148`) are the shared helpers, `nameRange` (`hover.go:159`) their hover counterpart.

**References** (`internal/lsp/references.go`). `GetReferences` (`:14`) has three walks, all one-home:
the definition maps (`:31-45`), the cursor-on-reference resolution (`:84-177`) and the collection
(`:203-255`). The collection covers `subscribes`, automation `on` and `command`, translation `reads` and
`command`, trigger `reads`, and flow command/event. An automation's `reads` appears in neither the
resolution nor the collection — verified: find-references on a view read by both an automation and a
trigger returns the declaration and the trigger only.

**Semantic tokens** (`internal/lsp/semantictokens.go`). `GetSemanticTokens` (`:67`) walks
`ctx.Aggregates` → `agg.Slices` (`:81-134`); commands, events, views, actors, contexts and aggregates
get token types, and a command declared directly on a `mode dcb` context gets none.

**Both-homes precedents.** `newModelIndex` (`internal/validator/validator.go:71-76`), `allSlicesIn`
(`internal/glossary/glossary.go:61-67`), `declaredSlices` (`internal/test/fixtures.go`) and
`exportedSlices` (`internal/export/export_test.go`) all visit both. The last three agree on order —
an aggregate's slices first, then the slices a `mode dcb` context declares directly — and every
hand-transcribed expectation in the repo is written in that order.

**LSP tests.** All `_test.go` files in `internal/lsp` are `//go:build unit`, package `lsp_test`, one
umbrella `Test<Function>` per file with a `const testDoc` and a `posIn` helper that resolves a substring
to 0-based coordinates. `internal/lsp/server_test.go` (`:114`) is the over-the-wire umbrella with a
`t.Run` group per LSP method — `completion` (`:450`), `definition` (`:591`), `references` (`:737`),
`hover` (`:891`). `requireLocation` (`references_test.go:74`) is a find-in-list check that cannot see a
missing or extra entry. No LSP test imports `internal/test` today.

**Shared fixtures this story reads.** `test.AutomationReadsLibraryLending`
(`internal/test/fixtures.go:578`) is the model with the shape every task needs: `MemberLoansView`
declared in an aggregate's slice (`:610`) and read by an automation in that aggregate (`:652`) *and* by
an automation in the `mode dcb` context "Reading Room" (`:731`); `DeskOccupancyView` declared directly
on that DCB context (`:714`) and read by an automation there (`:726`); `DeskClaimed` declared in a DCB
slice, subscribed by `DeskOccupancyView` and named by an automation's `on`; and a trigger reading
`AvailableCopiesView`, a name the model never declares — the unresolved-reference case.
`test.AutomationScheduleLibraryLending` (`:746`) is the same shape with schedules.
`internal/test/models.go` holds the parsing accessors.

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

**Verification:** `mise exec -- go test -tags unit ./internal/lsp/...`; `mise exec -- go build ./...`.

**Depends on:** None

---

### Task 2: Describe `on`, `every` and `reads`, and name the pattern `trigger` and `automation` belong to

**Behavior:** hovering `on` or `every` inside an automation describes what each declares, and hovering
`reads` describes the view an automation, trigger or translation consults. Hovering `trigger` or
`automation` names the Event Modeling pattern the element belongs to, describing its role rather than
its current syntax. A keyword's eligibility for hover comes from the lexer, so a keyword declared after
`KeywordExternal` is no longer invisible.

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
      (`internal/lsp/hover_test.go:121`, `internal/lsp/server_test.go:994`) move to the new wording, and
      `rg 'triggers on an event and sends a command' --glob '!docs/**'` returns nothing
- [ ] The `trigger` hover text names the Command pattern and describes the element as the human entry
      point into a slice
- [ ] The `trigger` hover text contains none of `UI`, `Schedule` or `Processor`, and describes no header
      shape with a kind between the keyword and the name — US-004 removes that slot, and hover must not
      have to move with it
- [ ] The `reads` hover text names automations alongside triggers and translations
- [ ] Hovering `model` still returns its current description, unedited, and hovering a token that is
      neither a described keyword nor a definition name still returns nil — the widened eligibility test
      does not start answering for undescribed keywords
- [ ] `rg 'KeywordModel|KeywordExternal' internal/lsp` returns nothing: the eligibility test is
      `lexer.Kind.IsKeyword()`, not an ordinal comparison, so the next keyword appended to
      `internal/lexer/token.go` is eligible on the day the lexer learns it
- [ ] A `fields` block declaring a field named `every` hovers that field's name as the `every` keyword —
      the behaviour a field named `reads` already has, asserted so the shared spelling is a stated
      decision rather than an accident
- [ ] Over the wire, `textDocument/hover` at a position on `on` and at a position on `every` in an open
      document returns the same markdown as the direct call, added to the `hover` group in
      `internal/lsp/server_test.go` (`:891`)

**Affected Files/Modules:**
- `internal/lsp/hover.go` — `keywordDescriptions` (`:13-33`) and `isKeyword` (`:37`)
- `internal/lsp/hover_test.go` — the keyword group, and the expectation at `:121`
- `internal/lsp/server_test.go` — the `hover` group (`:891`) and the expectation at `:994`

**Patterns to Follow:**
- `tasks/learnings.md` "Keyword surfaces fan out past the lexer, parser and tree-sitter grammar" — this
  task is the LSP half of that fan-out, and the `isKeyword` ordinal range it names is the reason `on`
  and `every` have no hover today
- `tasks/learnings.md` "Ask the lexer which keywords exist; never restate the set and never range over
  `Kind`" — `lexer.Kind.IsKeyword()` (`internal/lexer/token.go`) is the replacement it names
- `tasks/learnings.md` "Assert a short keyword in a diagnostic with a `\b`-bounded `require.Regexp`" —
  the same hazard applies to asserting hover prose
- `tasks/learnings.md` "A second `require.Contains` on one message is often shadowed by the first" —
  before adding a needle to a string that already has one, check it is not inside the earlier needle
- `docs/dsl-reference.md:276-350` is the authority for the four pattern names (Command, View,
  Automation, Translation) and for `trigger` belonging to the Command pattern; its Automation Pattern
  *skeleton* (`:322-340`) is stale and must not be copied — `tasks/learnings.md` records that it still
  documents the rejected `trigger <EventName>` spelling
- `writeAutomation` (`internal/formatter/formatter.go:347-357`) and `parseAutomationEntry`
  (`internal/parser/parser.go`) are what an automation actually accepts

**Testable:** Yes — through `lsp.GetHover` and the server's `textDocument/hover` handler.

**Verification:** `mise exec -- go test -tags unit ./internal/lsp/...`.

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
      asserted so the asymmetry is a recorded decision: US-004 rewrites the trigger header and the
      proposal adds an entry to its body
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
- `parseAutomationEntry` (`internal/parser/parser.go`) is the authority for which entries an automation
  accepts; the block imposes no order and no arity beyond the exactly-one-of rule over `on` and `every`,
  and completion imposes none either — both appear in the list
- Value completion after a keyword is Task 4's; this task changes only which keyword list is chosen

**Testable:** Yes — through `lsp.GetCompletions` and the server's `textDocument/completion` handler.

**Verification:** `mise exec -- go test -tags unit ./internal/lsp/...`.

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
      `GetSemanticTokens` already assigns events and views
      (`internal/lsp/semantictokens.go:57-63`), so one construct does not read as two things
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
- The parse-then-walk opening every other LSP entry point uses — `lexer.Scan` then `parser.New(...)`
  then `Parse`, keeping the model and discarding the diagnostics (`internal/lsp/hover.go:53-58`,
  `definition.go:19-24`, `references.go:19-24`, `semantictokens.go:72-77`); the parser is error-tolerant
  and returns a model for a half-written document
- Task 1's slice walk supplies the names from both homes; a fresh nested loop here would re-fork the
  shape that task removed
- `tasks/learnings.md` "An assertion whose expected value comes from the code under test is the
  recurring review finding" — write the expected name list by hand from the test document, never by
  walking the parsed model in the test
- `tasks/learnings.md` "Prefer a single structural assertion" — one `require.Equal` on the ordered label
  slice covers length, contents and order together
- `CompletionItemKind` values are in `internal/lsp/protocol.go:47-76`; `tokenTypeIndex`
  (`internal/lsp/semantictokens.go:11`) is where events and views already have a type each
- The server already advertises `" "` as a trigger character (`internal/lsp/server.go:99`), so no
  capability changes

**Testable:** Yes — through `lsp.GetCompletions` and the server's `textDocument/completion` handler.

**Verification:** `mise exec -- go test -tags unit ./internal/lsp/...`.

**Depends on:** 1, 3

---

### Task 5: Jump from an automation's `reads` to the view it names

**Behavior:** go-to-definition on the view name in an automation's `reads` entry navigates to that
view's declaration, wherever in the model it is declared. The automation's `on` event and its `command`
keep navigating as they do today, so all three jumps the story asks for work from one automation block.

**Acceptance Criteria:**
- [ ] The cursor on the view name in an automation's `reads` entry returns the location of that view's
      declaration, with the range starting at the declaration's name and ending at name start plus name
      length
- [ ] The cursor on the same automation's `on` event and on its `command` return their declarations —
      asserted in the same subtest group as the `reads` jump, so the story's three jumps are proved
      together rather than two of them assumed
- [ ] The jump resolves a view declared in a different aggregate from the automation, and a view
      declared directly on a `mode dcb` context while the automation sits in an aggregate's slice —
      both asserted on `test.AutomationReadsLibraryLendingModel`'s source, which declares exactly those
      two shapes
- [ ] An automation whose `reads` names a view the model does not declare returns nil, matching what a
      trigger reading an undeclared view already does
- [ ] The cursor on the `reads` keyword itself, rather than on the value, returns nil — navigation is
      from the reference, not the entry
- [ ] The automation in `internal/lsp/definition_test.go`'s `testDoc` (`:26-30`) gains a `reads` entry
      naming the view that document already declares; the `trigger manual "MyTrigger"` block (`:36`) is
      left exactly as written and no added assertion mentions a trigger kind, so US-004 migrates what it
      was already going to migrate
- [ ] Every existing subtest in `internal/lsp/definition_test.go` passes with its expectations unedited
      apart from coordinates shifted by the added line, and the `definition` group in
      `internal/lsp/server_test.go` (`:591`) still passes

**Affected Files/Modules:**
- `internal/lsp/definition.go` — the automation branch of the reference walk (`:73-89`), which handles
  `on`, `command` and `target context` and not `reads`
- `internal/lsp/definition_test.go` — `testDoc` (`:17`) and the automation group

**Patterns to Follow:**
- The translation branch immediately below (`internal/lsp/definition.go:92-103`) already resolves a
  `reads` value against the view map and is the line-for-line shape; the trigger branch (`:106-112`) is
  its single-valued sibling
- Task 1's slice walk is what makes the view map cover both homes; the branch itself resolves against
  that map
- `cursorOnName` (`internal/lsp/definition.go:136`) is the only hit test, and it returns false for an
  empty name, so an automation with no `reads` needs no extra guard
- `tasks/learnings.md` "Only an automation's `reads` resolves; a trigger's and a translation's must stay
  unchecked" is about the *validator*, not navigation — the LSP resolves all three and returns nil for
  an unresolved name, which is why the undeclared-view criterion above is a nil and not a diagnostic
- No `.emod` file, fixture or example is edited; `internal/test/fixtures.go` is read

**Testable:** Yes — through `lsp.GetDefinition` and the server's `textDocument/definition` handler.

**Verification:** `mise exec -- go test -tags unit ./internal/lsp/...`.

**Depends on:** 1

---

### Task 6: List the automations that read a view among its references

**Behavior:** find-references on a view returns every place the model reads it — the automations, the
triggers and the translations — together with its declaration, and the same list comes back with the
cursor on any one of those references. A command's and an event's references are unchanged.

**Acceptance Criteria:**
- [ ] Find-references with the cursor on a view's declaration name returns the declaration, every
      automation `reads` naming it, every trigger `reads` naming it and every translation `reads` naming
      it — asserted with one `require.Equal` against the full expected location list in walk order, not
      with the find-in-list `requireLocation` helper, which cannot see a missing or an extra entry
- [ ] Find-references with the cursor on an automation's `reads` value returns that same full list, so
      the reference resolves both ways
- [ ] On `test.AutomationReadsLibraryLendingModel`'s source, references to the view declared in an
      aggregate's slice and read by an automation in that aggregate *and* by an automation in the
      `mode dcb` context list both automations; references to the view declared directly on that context
      list the automation there
- [ ] A view that nothing reads returns its declaration alone
- [ ] References on a command still return the declaration, the flow entry, the automation `command` and
      the translation `command`; references on an event still return the declaration, the `subscribes`
      entry and the automation `on` — `git diff` moves no existing expectation in
      `internal/lsp/references_test.go`, and every change to that file is an added subtest or a
      coordinate shifted by an added line
- [ ] The automation in `internal/lsp/references_test.go`'s `testDoc` (`:22-26`) gains a `reads` entry
      naming the view that document already declares and that its trigger already reads, so one document
      carries an automation and a trigger reading the same view; the `trigger manual "MyTrigger"` block
      (`:31`) is left exactly as written and no added assertion mentions a trigger kind
- [ ] The `references` group in `internal/lsp/server_test.go` (`:737`) gains a leaf returning the
      locations for a view read by an automation over the wire

**Affected Files/Modules:**
- `internal/lsp/references.go` — the automation branch of the cursor-on-reference resolution
  (`:106-119`) and of the collection walk (`:218-225`), neither of which reads `Automation.Reads`
- `internal/lsp/references_test.go` — `testDoc` (`:14`), the view group, and `requireLocation` (`:70`),
  which the new leaves do not use
- `internal/lsp/server_test.go` — the `references` group (`:737`)

**Patterns to Follow:**
- The translation branch in both walks (`internal/lsp/references.go:126-139`, `:228-235`) already
  resolves and collects a `reads` value for a view target and is the shape for both halves; the trigger
  branch (`:146-153`, `:238-242`) is its single-valued sibling
- Task 1's slice walk is what makes the collection reach both homes; the branches themselves only add
  the automation's `reads`
- `tasks/learnings.md` "Prefer a single structural assertion" and "An assertion whose expected value
  comes from the code under test is the recurring review finding" — the expected location list is
  written from the test document by hand, and the whole list is compared at once
- The resolution walk's `break` structure after each reference kind (`:101`, `:121`, `:141`, `:155`)
  fixes where a new check goes; adding one inside the automation branch keeps the existing short-circuit
- Drawing a `reads` edge in the diagram outputs is US-005's; this task adds no edge and no exporter
  change

**Testable:** Yes — through `lsp.GetReferences` and the server's `textDocument/references` handler.

**Verification:** `mise exec -- go test -tags unit ./internal/lsp/...`.

**Depends on:** 1

---

## Summary

**Six tasks**, ordered dependency-first and, within that, by how much of the story each unblocks.

Task 1 comes first because three of the five features that follow add a walk over the model, and adding
them onto the one-home loops that exist today would fork that shape three more times — the exact
duplication `tasks/learnings.md` records as already live in all four LSP files. It is also the only task
that changes an existing output shape (hover text for a construct with no aggregate), so it lands alone.
Tasks 2 and 3 are independent of each other and of everything after them: Task 2 owns `hover.go`, Task 3
owns the keyword half of `completer.go`. Task 4 follows Task 3 because it edits the same file and needs
the automation-body determination to exist before it can decide that a value position sits inside one,
and follows Task 1 because its name lists must reach both slice homes. Tasks 5 and 6 are independent of
each other — one file each — and both rest on Task 1's walk.

**Story criteria coverage:**

| Story criterion | Task |
|---|---|
| Completion inside an automation block offers `on`, `every`, `reads`, `command`, `target context` | 3 |
| Completion after `on` offers event names; after `reads`, view names | 4 |
| Hovering `on`, `every` or `reads` describes what each does | 2 |
| Hovering `trigger` or `automation` states which Event Modeling pattern the element belongs to | 2 |
| Go-to-definition from an automation's `on` event, its `reads` view, and its `command` | 5 (`reads` added; `on` and `command` pinned) |
| Find-references on a view lists the automations and triggers that read it | 6 |

Carried along, not stated by the story: the both-homes slice walk (1), without which every criterion
above is silently false inside a `mode dcb` context; the `isKeyword` ordinal range (2), which is why
`on` and `every` have no hover today whatever the description map says; and the `pendingKeyword` latch
(3), which is why a cursor below an automation's `command` line completes top-level keywords.

**Flagged for US-004.** Task 2's hover text for `trigger` describes the element's role and is pinned
against naming `UI`, `Schedule` or `Processor` or describing a kind slot, so the two stories do not
collide. Tasks 5 and 6 extend the automations of the two LSP test documents and leave their
`trigger manual "MyTrigger"` headers untouched, so this story adds no new occurrence of the kind slot
for US-004 to migrate. No other task writes a trigger declaration.

**Deferred to later stories in the feature:** the `reads` edge (US-005), lane placement (US-006), the
palette (US-007), the `automation/missing-todo-list` rule (US-008), the VS Code TextMate alternation and
the tree-sitter highlight queries (US-010), and `docs/dsl-reference.md` — whose Automation Pattern
skeleton still documents the retired `trigger <EventName>` spelling — along with `README.md` and
`examples/` (US-011).

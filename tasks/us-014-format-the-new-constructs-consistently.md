# US-014: Format the new constructs consistently

## Progress
- [ ] Task 1: Order a view's `subscribes` ahead of its `fields`
- [ ] Task 2: Align a spec's `given`, `when` and `then` keywords
- [ ] Task 3: Align the `:` across a flow block's entries
- [ ] Task 4: Wrap a payload that does not fit onto one field per line
- [ ] Task 5: Round-trip a model stating every new construct through `emod fmt`

---

## Story Reference

`user-stories/specs-and-metadata.md` → **US-014: Format the new constructs consistently**
(fourteenth story of "Specs, Invariants, and Model Metadata", lines 181-191). Design notes:
`docs/proposals/specs-and-metadata-proposal.md`. The original formatter story is
`tasks/completed/us-004-format-emod-files.md`, which established the `fields` column convention this
story generalises.

**This story is not "make the formatter aware of the new syntax."** Every story in this batch already
carries its own formatter task, and each was told to emit a canonical one-line form and leave column
alignment here:

| Story | Task | What it already owns |
|---|---|---|
| US-007 | Task 5 | `then view <Name>` / `then command <Name>` outcomes survive `emod fmt` |
| US-009 | **Task 1** | the `command -> rejected:` entry — parser *and* formatter, deliberately not split |
| US-010 | Task 5 | payloads survive `emod fmt`, in a canonical single-line form |
| US-012 | Task 3 | an event's wire `type` survives `emod fmt` |
| US-013 | Task 4 | `after "<duration>"` survives `emod fmt` |

US-009 Task 1 is the one to read first: it records that a parser-only commit is a commit at which
`emod fmt` silently *deletes* a user's rejection edges, which is why the two land together. The same
reasoning is why none of that work belongs here.

**In scope** — the four canonicalisation criteria the other stories explicitly deferred, plus the
integration criterion:

1. Canonical attribute order within a block: `description` first, then the block's pattern-specific
   attributes, then `fields`, then `spec` blocks last (Task 1).
2. `given` / `when` / `then` alignment within a spec (Task 2).
3. `:` alignment across `command -> event:` and `command -> rejected:` entries within one `flow`
   block (Task 3).
4. Payloads on one line when they fit, else one field per line with values aligned, matching the
   `fields` block convention (Task 4).
5. "A file using every new construct formats round-trip without data loss" (Task 5) — the
   integration criterion, which depends on **all five** upstream formatter tasks, not on this story
   alone.

**Out of scope**, named explicitly:

- Teaching the formatter the new syntax at all. Every construct's emission is owned by the table
  above. If `emod fmt` drops a construct, that is a bug in the owning story's formatter task, and it
  is fixed there — never worked around here.
- Any parser, validator, linter, export, glossary or diagram change. This story writes bytes only.
- LSP hover, completion and navigation over the new constructs (US-015). Note that
  `internal/lsp/server.go:362` serves `textDocument/formatting` from `formatter.Format`, so the LSP
  inherits every change here for free and needs no edit.
- Rendering specs on diagrams (US-016), syntax highlighting (US-017), examples and the DSL reference
  (US-018).
- Bringing the checked-in `.emod` files to canonical form. `emod fmt --check` already fails on
  `examples/all_patterns.emod`, `examples/dcb_model.emod` and `internal/parser/testdata/all_patterns.emod`
  today (verified by running it), nothing in the repo guards them, and closing that pre-existing gap
  is a separate piece of work. No task here rewrites a `.emod` file.

**What happens if an upstream formatter task has not landed.**

- **Tasks 1 and 2 need nothing from this batch.** US-001, US-002 and US-006 are on main; a view's
  `subscribes` and a spec's `given`/`when`/`then` are all emitted today. They can start immediately.
- **Task 3 is blocked on US-009 Task 1, not skippable.** Until a `flow` block can hold a second entry
  kind, every entry in a block has the same prefix width, so the alignment has exactly one possible
  output and no test can tell it from today's formatter. Landing it early produces a vacuous task.
- **Task 4 is blocked on US-010 Task 5.** No payload is emitted at all until then, so there is
  nothing to wrap.
- **Task 5 is blocked on all five.** A fixture stating a construct whose formatter task has not
  landed makes the round-trip fail *for the upstream story's reason* — `emod fmt` deleted the
  construct. That failure is a true report and belongs upstream; do not weaken the fixture to make it
  pass.
- The dependency runs one way only. US-007's two new outcomes are `then` *values*, so Task 2's
  keyword column already covers them; US-012's wire type is one more line inside an `event` block,
  so Task 1's canonical order already covers it. Neither needs a follow-up here. US-009's and
  US-010's constructs do, which is why Tasks 3 and 4 sit *after* them rather than before.

**Measured: the canonical-order blast radius.** The risky criterion is attribute reordering, because
it changes the bytes of already-formatted files. Measured rather than assumed, with a throwaway
walker that parsed every checked-in model in the repo — the 18 `.emod` files, every ` ```emod ` fence
in every `.md`, and every backtick-quoted emod block in every `.go` file (314 model sources,
`internal/test/fixtures.go` and `internal/wasm`'s `billingModel` included) — then deleted:

- Today's formatter deviates from the stated canonical order in **exactly one** writer: `writeView`
  (`internal/formatter/formatter.go:334-345`) emits `fields` before `subscribes`. `writeCommand`,
  `writeEvent`, `writeTrigger`, `writeAutomation`, `writeTranslation`, `writeContext`,
  `writeAggregate` and `writeSlice` already read description → pattern-specific → `fields` → `spec`.
- **74 `view` blocks** exist across those sources. **32 of them declare both `fields` and
  `subscribes`**, in 12 files: `internal/test/fixtures.go` (11), `internal/arrange/arrange_test.go`
  (4), `examples/all_patterns.emod` (3), `README.md` (2), `internal/parser/testdata/all_patterns.emod`
  (2), `internal/parser/parser_test.go` (2), `internal/cli/validate_test.go` (2),
  `internal/cli/fmt_test.go` (2), `internal/parser/testdata/multi_context.emod` (1),
  `internal/cli/slices_arrange_test.go` (1), `examples/error_diagnostics_test.emod` (1),
  `docs/proposals/ai/02-model-import-reverse-engineering.md` (1).
- **Every one of the 32 writes `subscribes` after `fields`.** Not one checked-in model states the
  other order, so the reorder moves all 32 sites — there is no "most files already agree" escape.
- The distinction that shrinks the work: the emod parser is order-independent, so an *input* fixture
  needs no edit. Only transcriptions of formatter *output* must move, and there are three:
  `internal/formatter/formatter_test.go`'s "formats view with fields and subscribes" golden
  (`:311-360`, expectation at `:355`), and the canonical constants `keywordFieldFormattedEmod`
  (`internal/cli/fmt_test.go:135`, view at `:182-190`) and `specFormattedEmod` (`:244`, view at
  `:324-332`). `internal/wasm`'s `billingModel` declares no view and stays byte-stable, which matters
  because `internal/wasm/pipeline_test.go:131` asserts the format round-trip returns it unchanged.
- The two remaining alignment criteria are byte-neutral on today's tree, by construction: a `flow`
  block holding only `command -> event:` entries has one prefix width, and no checked-in model
  carries a payload. `given`/`when`/`then` alignment is *not* neutral — it moves every spec that
  states a `given`, which is `specFormattedEmod` (2 of its 3 specs) and the `"specs"` group in
  `internal/formatter/formatter_test.go:3597` (expectations at `:3652-3654` and `:3694-3696`).

**Open questions, decided.** Five shapes the story does not name, each decided so the criteria are
checkable and the churn is bounded:

1. *Alignment width is computed per block, over the entries that block actually states* — never a
   fixed language-wide column. This is the existing `fields` and `tags` convention (`fieldColumnWidths`
   at `internal/formatter/formatter.go:313`, the `maxKey` loop in `writeTags` at `:299`). It is also
   what keeps the churn bounded: a spec stating no `given` is unchanged, and a `flow` block holding
   one entry kind is unchanged.
2. *`subscribes` is a `view`'s pattern-specific attribute, so it precedes `fields`.* The alternative
   reading — that `fields` merely has to sit above `spec` — would leave the criterion with nothing to
   do in any block, since every other writer already complies. The story asked for a rule with teeth.
3. *"Fits" is 100 columns of rendered line, counting its leading indent.* The repo fixes no line
   width anywhere: there is no `.editorconfig`, no linter config, and the longest existing line in
   `internal/cli/fmt_test.go` is 116 characters. The story therefore has to choose one, and it must
   live in a single named constant in `internal/formatter` rather than as a literal at the comparison
   site. 100 leaves a payload at the usual slice nesting depth room to stay on one line.
4. *A payload that wraps forces its enclosing `given` / `then` list one element per line.* A
   multi-line brace block cannot sit inside a single-line comma list and stay readable, and the
   wrapped list has to re-parse. `when` has no enclosing list, so it wraps on its own.
5. *This story adds no keyword.* Both CI-enforced drift tests — `internal/lsp/keywords_test.go` and
   `editors/tree-sitter-emod/test/queries/keywords_test.go` — derive their expectation set from
   `lexer.Keywords()` and name no keyword individually (verified by reading both), so both stay green
   with no edit. No task here touches `internal/lexer`, `editors/vscode/syntaxes/emod.tmLanguage.json`
   or `editors/tree-sitter-emod/`. Wrapping (Task 4) introduces a source shape no checked-in file
   uses, so it owes a check that the tree-sitter grammar accepts the wrapped form — a check, not a
   grammar edit; the grammar must never be stricter than the Go parser.

**Learnings folded in** from `tasks/learnings.md`:

- *"`emod fmt` canonicalises order, so a fmt golden is never the input re-indented"* (`:141`) — every
  expected value this story adds is written as canonical output, and passed to `requireFmtSettlesOn`
  (`internal/cli/fmt_test.go:629`) rather than handing the input fixture back.
- *"A new block entry goes after `description` and ahead of nested blocks, in every writer"* (`:156`)
  and *"`ast.ThenClause` dispatches through five type switches, none of which errors"* (`:186`) — the
  formatter renders from the AST and emits only what it knows, so **the parse → format → reparse
  comparison against the original model is the only thing that notices a dropped construct.** Neither
  idempotence nor an existing golden does. Every task here carries that comparison, and Task 5 exists
  because it is the only assertion that can catch a construct this batch added being silently lost.
- *"`emod fmt` moves a spec to the end of its slice and orders its entries given/when/then"* (`:196`)
  — the entry order Task 2 aligns is already canonical; Task 2 changes the column, not the sequence.
- *"Formatter output always begins with `emod N`"* (`:31`) — every expected constant added here opens
  with the version header, while input fixtures may omit it.
- *"Never write emod source with `%q`"* (`:46`) — Task 4 renders payload string values, so it owes
  the hazard-character round-trip (backslash, tab, quote, `%`, non-ASCII) that `quoted()` exists for.
- *"Additive output changes owe a byte-identical receipt for models that do not use the feature"*
  (`:51`) and *"A differential receipt must first prove the twin actually differs"* (`:96`) — each
  task states which models must not move and proves it.
- *"De-duplicate before a fan-out edit, and land the de-duplication with proof"* (`:71`) and *"Name an
  extracted helper after the contract its callers rely on"* (`:101`) — this story adds a third,
  fourth and fifth "compute a column width, then pad" site to the two `writeFields`/`writeTags`
  already carry. Task 2 extracts the shape once, with a byte-identical receipt for the existing
  callers, and Tasks 3 and 4 reuse it instead of re-deriving it.
- *"A 'no expected constant moves' criterion is unsatisfiable when the task edits a shared fixture"*
  (`:481`) — the criteria here name the expected values that *do* move, one by one, so the guard
  still catches an unexpected one while staying satisfiable.
- *"A task criterion requiring 'committed' output cannot close"* (`:21`) and *"A commit-message
  receipt is the commit author's obligation, never an acceptance criterion"* (`:246`) — no criterion
  below references a commit, branch, tag or remote, or asks for a receipt stated in a commit message.
- *"`emod fmt <file>` writes in place, so a receipt run dirties the working tree"* (`:336`) — any
  verification that runs `emod fmt` against a checked-in file copies it to a temp path first, or uses
  `--check`.
- *"A new shared fixture owes `internal/oracle` a zero-diagnostic subtest"* (`:151`), *"Shared
  fixtures come in an unfeatured/featured pair"* (`:66`) and *"A new optional field ships a six-part
  fixture kit"* (`:216`) — Task 5 adds a shared fixture and owes the oracle leaf.
- *"A tested, defensible improvement found on the way is still a separate commit"* (`:461`) — a
  change is in scope only when the task's own criteria cannot pass without it.
- *"A task's change-set assertion must name every file its own patterns require it to change"*
  (`:326`) — the Affected Files list and the change-set criterion in each task below agree.

**Repo-drift note.** `internal/export/export.go` no longer exists; the package is now `json.go`,
`cue.go` and `diagram.go`. Several `tasks/learnings.md` entries still cite the old path. No task here
touches the export package, but a reader following a learning's file reference should expect the split.

---

## Codebase Context

**The formatter is one file.** `internal/formatter/formatter.go` is 407 lines: a `writer` over a
`strings.Builder` with `line`/`lineIfSet`/`quotedLineIfSet`/`blankLine` primitives, one `write*`
method per AST node, and `quoted()` (`:57-63`) as the single gate on emod string output.
`formatter.Format(model)` (`:10-15`) is the whole public surface, and it is the *only* writer of emod
source in the repo — `internal/cli/fmt.go:33`, `internal/cli/slices_arrange.go:46`,
`internal/wasm/pipeline.go:76`, `internal/lsp/server.go:362` and `internal/importer` all route
through it. A change here reaches `emod fmt`, `emod slices arrange`, the WASM viewer's save path and
the LSP's format-on-save in one edit.

**Alignment today** exists in two places, both computing a maximum width then padding with `%-*s`:
`fieldColumnWidths` + `writeFields` (`:285-297`, `:313-323`) aligns a field's name and type columns,
and `writeTags` (`:299-311`) aligns a tag's key before its `:`. Both compute per block. They are the
convention every criterion in this story points at, and they are the two callers Task 2's extraction
must leave byte-identical.

**The block writers and their current entry order.** `writeContext` (`:120`) and `writeAggregate`
(`:142`) emit description → invariants → nested blocks. `writeSlice` (`:156`) emits description →
trigger → commands → events → views → automations → translations → flows → **specs last**.
`writeCommand` (`:217`) emits description → `decides_on` → `fields`. `writeEvent` (`:269`) emits
description → `tags` → `source external` → `fields`. `writeTrigger` (`:208`), `writeAutomation`
(`:347`) and `writeTranslation` (`:359`) hold no `fields`. **`writeView` (`:334-345`) emits
description → `fields` → `subscribes`** — the one deviation.

**`writeSpec` (`:372-385`)** emits `given` (as `bracketed(specElementNames(...))`), then `when`, then
`then` (through `formatOutcome`, `:387-399`), each as a bare `w.line(level+1, "<keyword> %s", ...)`
with a single space. An entry the spec does not state is left out entirely, so the set of keywords
present varies per spec — which is what makes per-spec width computation the right rule.

**`writeFlows` (`:325-332`)** emits one `flow {` block per slice holding every entry as
`command -> event: %s -> %s`. The prefix is a literal in the format string; US-009 Task 1 adds a
second one, `command -> rejected:`, 3 characters wider (20 vs 17).

**Formatter tests.** `internal/formatter/formatter_test.go` is one `TestFormat` umbrella (`:21`) with
twelve top-level groups: `"version header"` (`:22`), `"element formatting"` (`:32`), **`"round-trip
through the parser"` (`:702`)** — the parse → format → reparse group, which uses
`test.RequireEqual(t, original, reparsed, ignoreFormatterNormalizations)` and is where a dropped
construct is caught — `"blank lines and ordering"` (`:1259`), `"field alignment"` (`:1719`),
`"comments"` (`:2053`), `"blank line normalisation"` (`:2308`), `"context modes"` (`:2593`),
`"event tags"` (`:2814`), `"decides_on"` (`:3033`), `"specs"` (`:3597`) and `"dcb regression"`
(`:3709`). Byte expectations are `strings.Join([]string{...}, "\n")` line slices; the round-trip
group's leaves compare ASTs and so do not move when a column does.

**CLI fmt tests.** `internal/cli/fmt_test.go` holds the canonical `*FormattedEmod` constants —
`formattedEmod` (`:47`), `describedFormattedEmod` (`:49`), `modifierlessFormattedEmod` (`:118`),
`keywordFieldFormattedEmod` (`:135`), `specFormattedEmod` (`:244`), `scheduledAutomationFormattedEmod`
(`:446`) — each a transcription of formatter output, fed to `requireFmtSettlesOn` (`:629`), which
formats once, compares against the constant, formats again and compares again. `--check` leaves are
at `:590-628`.

**Shared fixtures.** `internal/test/fixtures.go` (1404 lines) holds the model constants every package
consumes — `HotelReservation`, `DescribedHotelReservation`, `KeywordFieldSearchCatalog`,
`InvariantLibraryLending`, `SpecLibraryLending`, `AutomationReadsLibraryLending`,
`TriggerReadsLibraryLending` — with parsing accessors in `internal/test/models.go` (e.g.
`SpecLibraryLendingModel(t)` at `:37`), `Without…` twins built on `copyWithEditedSlices` (`:1257`)
and `editedCopies` (`:1275`), and `Declared…` getters (`:1299-1352`). `internal/oracle/oracle_test.go`
holds one zero-diagnostic leaf per fixture in its `"clean input"` group (`:24`).

**Keyword drift.** `internal/lsp/keywords_test.go` (`TestKeywordCoverage`) iterates `lexer.Keywords()`
for hover and completion coverage; `editors/tree-sitter-emod/test/queries/keywords_test.go`
(`//go:build grammar`, run by `task test:grammar`) does the same for the tree-sitter grammar, the
highlight query and the TextMate grammar. Neither names a keyword, and this story adds none.

---

## Tasks

### Task 1: Order a view's `subscribes` ahead of its `fields`

**Behavior:** `emod fmt` writes every block's entries in one canonical order — `description` first,
then the block's pattern-specific attributes, then its `fields` block, then `spec` blocks last. Today
one writer deviates: a `view` emits `subscribes` after `fields`. It now emits it before, so a view
reads as what it is, what it listens to, then what it holds, the shape every other block already has.

**Acceptance Criteria:**
- [x] A `view` declaring both a `fields` block and a `subscribes` list formats with the `subscribes`
      line above `fields {`, whichever order the source stated them in, and re-parsing that output
      yields a view with the same fields in declaration order and the same subscriptions in
      declaration order
- [x] A `view` declaring only `fields`, and one declaring only `subscribes`, each format to exactly
      the bytes they format to today, so `internal/formatter/formatter_test.go:1671` ("view with
      subscribes but no fields omits fields block") and `:2594` ("view with fields but no subscribes
      omits subscribes line" — filed under the `"context modes"` group at `:2593`, which is where to
      look for it) pass with no edit
- [x] A `description` on a view still precedes both
- [x] One test formats a single model that states every attribute every block accepts — `context`,
      `aggregate`, `slice`, `trigger`, `command`, `event`, `view`, `automation`, `translation` — and
      compares the whole rendered text against one expected value, so the canonical order is pinned
      in one readable place rather than inferred from nine scattered goldens
- [x] Deleting the reorder from `writeView` makes that whole-output test fail, and re-parsing its
      output compares equal to the original model under `test.RequireEqual` with
      `ignoreFormatterNormalizations`; formatting the output again is byte-identical
- [x] The only expected values that move are those transcribing a view that declares both `fields`
      and `subscribes`, and every such transcription in the repository moves. Measured on the branch
      the task ran against, that is seven, not the three the blast-radius walk above recorded: the
      walk ran before US-007, US-009 and US-010 added canonical constants of their own. The seven are
      the "formats view with fields and subscribes" golden in
      `internal/formatter/formatter_test.go`; `keywordFieldFormattedEmod`, `specFormattedEmod`,
      `payloadFormattedEmod`, `slicePatternFormattedEmod` (two views) and `rejectionFormattedEmod` in
      `internal/cli/fmt_test.go`; and the comment round-trip expectation in
      `internal/importer/importer_test.go`. `git diff` shows no other golden, canonical constant or
      transcribed name list moving
- [x] `git diff` leaves every input fixture untouched — `internal/test/fixtures.go`,
      `internal/arrange/arrange_test.go`, `internal/parser/parser_test.go`,
      `internal/cli/validate_test.go`, `internal/cli/slices_arrange_test.go`, `examples/*.emod` and
      `internal/parser/testdata/*.emod` all parse identically either way and need no edit
- [x] `internal/wasm/pipeline_test.go`'s format round-trip over `billingModel` (`:131`) still compares
      equal with no edit — that model declares no view, so no byte of it may move

**Affected Files/Modules:**
- `internal/formatter/formatter.go` — `writeView` (`:334-345`), the two emission calls swapped
- `internal/formatter/formatter_test.go` — the golden at `:311-360`, and the new whole-output
  canonical-order test (it belongs in `"blank lines and ordering"` (`:1259`), which already owns
  cross-block ordering, rather than in `"element formatting"`, which owns one construct at a time)
- `internal/cli/fmt_test.go` — `keywordFieldFormattedEmod` (`:135`) and `specFormattedEmod` (`:244`)

**Patterns to Follow:**
- The order every other writer already has: `writeCommand` (`internal/formatter/formatter.go:217-228`)
  and `writeEvent` (`:269-283`)
- Whole-output byte comparison with a `strings.Join([]string{...}, "\n")` expectation:
  `internal/formatter/formatter_test.go:1259` onwards
- `tasks/learnings.md` "A '`no expected constant moves`' criterion is unsatisfiable when the task
  edits a shared fixture" (`:481`) — why the change-set criterion above names its three sites instead
  of forbidding all movement
- `tasks/learnings.md` "`emod fmt` canonicalises order, so a fmt golden is never the input
  re-indented" (`:141`)

**Testable:** Yes

**Verification:** `mise exec -- task test` (or `mise exec -- go test -tags unit ./internal/...`) is
green; `git diff --stat` lists exactly `internal/formatter/formatter.go`,
`internal/formatter/formatter_test.go` and `internal/cli/fmt_test.go`.

**Depends on:** None

---

### Task 2: Align a spec's `given`, `when` and `then` keywords

**Behavior:** Inside a formatted `spec` block the `given`, `when` and `then` keywords are padded to a
common width, so their values start in one column the way a `fields` block aligns its types. The
width is computed over the entries that spec actually states, so a spec that omits `given` — the
widest of the three — formats to exactly the bytes it does today. The column-width-then-pad shape that
`writeFields` and `writeTags` each carry a copy of is extracted first, so this story's three new
alignment sites reuse it instead of adding three more copies.

**Acceptance Criteria:**
- [x] A spec stating `given`, `when` and `then` formats with all three values starting in the same
      column, one space past the widest keyword the spec states
- [x] A spec stating only `when` and `then` formats with no padding on either, so its bytes are
      unchanged from today; a spec stating only `given` and `then` likewise
- [x] The width is computed per spec block: two sibling specs in one slice, one stating `given` and
      one not, align independently, asserted against one whole-block expected output rather than by
      searching for a line
- [x] The aligned output re-parses to a spec equal to the original in name, `given` events and their
      order, `when` reference, and outcome — and formatting that output again is byte-identical
- [x] The column-width computation is extracted once and both `writeFields` and `writeTags` call it,
      and formatting `test.HotelReservation`, `test.DescribedHotelReservation`,
      `test.KeywordFieldSearchCatalog`, `test.InvariantLibraryLending` and `test.SpecLibraryLending`
      produces byte-identical output to before the extraction for every `fields` and `tags` block, so
      the extraction moves no byte of its existing callers. The extraction landed as its own commit,
      and the receipt was taken by capturing the five fixtures' formatted output at the parent commit
      and again after it: 14456 bytes, byte-identical, over 34 `fields` blocks and 4 `tags` blocks
      including a padded tag key
- [x] The extracted helper is named for the postcondition a second caller relies on — a padded column
      width across a set of rows — not for `fields`, and it takes the same parameters its siblings do
- [x] The only expected values that move are those transcribing a spec that states `given`. Measured
      on the branch: four canonical constants in `internal/cli/fmt_test.go` — `specFormattedEmod`,
      `payloadFormattedEmod`, `rejectionFormattedEmod` and `slicePatternFormattedEmod`, the last three
      added by US-009 and US-010 after the walk above was taken — and four expectations in
      `internal/formatter/formatter_test.go`: the three in the `"specs"` group whose spec states a
      `given`, plus the canonical-order constant task 1 added. `git diff` shows no other golden or
      canonical constant moving; every expectation whose spec states only `when` and `then` keeps its
      bytes, and no input fixture moves
- [x] `emod fmt --check` over a canonically formatted file whose specs state `given` reports no change
      needed and leaves the file on disk unchanged

**Affected Files/Modules:**
- `internal/formatter/formatter.go` — `writeSpec` (`:372-385`); `fieldColumnWidths` (`:313-323`),
  `writeFields` (`:285-297`) and `writeTags` (`:299-311`) for the extraction
- `internal/formatter/formatter_test.go` — the `"specs"` group (`:3597`) and the `"field alignment"`
  group (`:1719`), which is where the extraction's byte-identical receipt belongs
- `internal/cli/fmt_test.go` — `specFormattedEmod` (`:244`) and its `requireFmtSettlesOn` leaf (`:578`)

**Patterns to Follow:**
- The alignment convention this generalises: `fieldColumnWidths` + the `%-*s` calls in `writeFields`
  (`internal/formatter/formatter.go:285-297`, `:313-323`) and the `maxKey` loop in `writeTags`
  (`:299-311`)
- `tasks/learnings.md` "De-duplicate before a fan-out edit, and land the de-duplication with proof"
  (`:71`) — including its warning against a helper that hardcodes what its siblings parameterise
- `tasks/learnings.md` "Name an extracted helper after the contract its callers rely on" (`:101`)
- `tasks/learnings.md` "`emod fmt` moves a spec to the end of its slice and orders its entries
  given/when/then" (`:196`) — the sequence is already canonical; only the column changes here
- The round-trip assertion class: `internal/formatter/formatter_test.go:876` and `:903`

**Testable:** Yes

**Verification:** `mise exec -- go test -tags unit ./internal/formatter/ ./internal/cli/` is green;
`git diff` moves exactly the expected values named above.

**Depends on:** None (US-006 is on main). Independent of Task 1 — the two touch different writers and
different goldens.

---

### Task 3: Align the `:` across a flow block's entries

**Behavior:** Within one formatted `flow` block, the `:` of every entry lands in the same column, so a
block mixing `command -> event:` and `command -> rejected:` entries reads as two columns rather than a
ragged list. The width is computed per `flow` block over the entries that block states, so a block
holding only one entry kind is unchanged — which is every `flow` block in the repo today.

**Acceptance Criteria:**
- [x] A `flow` block holding both `command -> event:` and `command -> rejected:` entries formats with
      every entry's `->` operand starting in the same column, the narrower prefix padded to the wider
- [x] A `flow` block holding only `command -> event:` entries, and one holding only
      `command -> rejected:` entries, each format with a single space after the `:` and no padding
- [x] The width is computed per `flow` block, not per slice or per file: two slices in one model, one
      with a mixed block and one with a single-kind block, are asserted together against one whole
      expected output and align independently
- [x] Padding is inserted before the operand only; no entry gains trailing whitespace, and no line in
      the output ends in a space — proven by the whole-output expected value, which is compared byte
      for byte and holds no trailing space, rather than by a separate absence assertion the equality
      above it already subsumes
- [x] Parsing a source stating both entry kinds, formatting it and re-parsing yields a model whose
      flow entries and rejection entries match the original in kind, order and both names — the
      comparison being against the original model, never against a second format run
- [x] Formatting the output again is byte-identical
- [x] Existing expected values move only where a `flow` block states both entry kinds. When the walk
      above was taken US-009 had not landed and no such block existed anywhere, so the criterion read
      "moves no existing expected value"; on the branch this task ran against there are four — the
      mixed-block golden in `internal/formatter/formatter_test.go`'s `"element formatting"` group, the
      leading-comment leaf's expectation, task 1's canonical-order constant, and
      `rejectionFormattedEmod` in `internal/cli/fmt_test.go`. Every single-kind block in all of
      `internal/formatter/formatter_test.go`, `internal/cli/fmt_test.go`,
      `internal/cli/slices_arrange_test.go` and `internal/wasm/pipeline_test.go` keeps its bytes, and
      no input fixture moves
- [x] `emod fmt --check` over a canonically formatted file whose flow block mixes both kinds reports
      no change needed and leaves the file on disk unchanged

**Affected Files/Modules:**
- `internal/formatter/formatter.go` — `writeFlows` (`:325-332`), plus whatever US-009 Task 1 added
  beside it for the rejection entry
- `internal/formatter/formatter_test.go` — a leaf in the group US-009 Task 1 put its flow output in,
  and a leaf in `"round-trip through the parser"` (`:702`)
- `internal/cli/fmt_test.go` — a canonical constant for a model with a mixed flow block, fed to
  `requireFmtSettlesOn` (`:629`)

**Patterns to Follow:**
- The per-block width shape extracted in Task 2 — reuse it, do not re-derive a third `maxLen` loop
- `writeTags` (`internal/formatter/formatter.go:299-311`) is the closest existing shape: a padded key,
  then `:`, then a value
- `tasks/us-009-show-rejection-paths-on-the-timeline.md` Task 1, whose criterion explicitly hands this
  alignment here ("with no column alignment across the two kinds — US-014 owns the `:` alignment")
- `tasks/learnings.md` "A new block entry goes after `description` and ahead of nested blocks, in
  every writer" (`:156`) — why the round-trip comparison against the *original* model is the assertion
  that matters here, not idempotence

**Testable:** Yes

**Verification:** `mise exec -- go test -tags unit ./internal/formatter/ ./internal/cli/` is green;
`git diff` shows no pre-existing expected value moving.

**Depends on:** Task 2 (for the shared width helper), and **US-009 Task 1**, which must be on the
branch first — until a `flow` block can hold a second entry kind there is one possible prefix width
and this task cannot be told from today's formatter by any test.

---

### Task 4: Wrap a payload that does not fit onto one field per line

**Behavior:** A spec element's example payload stays on one line when the rendered line fits the
formatter's column budget. When it does not, the payload is written one `field: value` per line with
the values column-aligned, the same convention a `fields` block uses, and the enclosing `given` or
`then` list breaks to one element per line so the brace block is not buried inside a comma list. The
wrapped form re-parses to the same payload and formatting it again is byte-identical.

**Acceptance Criteria:**
- [ ] `internal/formatter` states its column budget as one named constant, and no comparison site
      repeats the number as a literal
- [ ] A `when` reference whose rendered line is exactly the budget stays on one line; one character
      wider wraps — both boundaries asserted, so the comparison cannot be off by one
- [ ] A wrapped payload writes one `field: value` per line, with every value starting in the same
      column, the field names padded to the widest name in that payload
- [ ] The width is computed per payload: two payloads in one spec align independently
- [ ] When any element of a `given` or `then` list wraps, every element of that list goes on its own
      line, including elements that would have fit and elements carrying no payload at all
- [ ] A wrapped payload re-parses to a payload equal to the original in field names, declaration
      order, values and literal kinds; formatting the re-parsed model produces byte-identical text
- [ ] A payload short enough to fit formats exactly as US-010 Task 5 leaves it, so
      `internal/formatter/formatter_test.go` and `internal/cli/fmt_test.go` pass with no edit to any
      expected value US-010 added for the single-line form
- [ ] A payload string value containing a backslash, a tab, a double quote, a `%` and a non-ASCII
      character survives parse → format → parse → format with identical bytes in the wrapped form as
      well as the single-line form, proving the wrapped writer also goes through `quoted()` and never
      through `%q`
- [ ] The tree-sitter grammar parses the wrapped form — asserted by a corpus case, since the grammar
      must never be stricter than the Go parser and no checked-in file uses this shape
- [ ] `emod fmt --check` over a canonically formatted file carrying a wrapped payload reports no
      change needed and leaves the file on disk unchanged

**Affected Files/Modules:**
- `internal/formatter/formatter.go` — the payload rendering US-010 Task 5 added inside `writeSpec`
  (`:372-385`) and around `bracketed` (`:65-67`), plus the new budget constant
- `internal/formatter/formatter_test.go` — the `"specs"` group (`:3597`), the round-trip group
  (`:702`) and its quoting-hazard table (`:953`)
- `internal/cli/fmt_test.go` — a canonical constant for a model with a wrapping payload, fed to
  `requireFmtSettlesOn` (`:629`)
- `editors/tree-sitter-emod/test/corpus/` — one corpus case for the wrapped shape, in the file US-010
  Task 7 put its payload cases in

**Patterns to Follow:**
- The multi-line alignment convention: `writeFields` and `fieldColumnWidths`
  (`internal/formatter/formatter.go:285-297`, `:313-323`); the `key: value` alignment in `writeTags`
  (`:299-311`)
- The per-block width helper extracted in Task 2
- `quoted()` (`internal/formatter/formatter.go:57-63`) and `tasks/learnings.md` "Never write emod
  source with `%q`" (`:46`), including its obligation to carry a round-trip subtest per hazard
  character
- `tasks/us-010-state-example-payloads-in-specs.md` Task 5 for the single-line canonical form this
  extends, and Task 7 for where the grammar's payload corpus cases live
- `tasks/learnings.md` "The tree-sitter grammar must never be stricter than the Go parser" (`:61`)

**Testable:** Yes

**Verification:** `mise exec -- go test -tags unit ./internal/formatter/ ./internal/cli/` and
`mise exec -- task test:grammar` are green (the grammar suite is not reached by a plain `go test ./...`).

**Depends on:** Task 2 (for the shared width helper), and **US-010 Task 5**, which must be on the
branch first — no payload is emitted at all until then, so there is nothing to wrap.

---

### Task 5: Round-trip a model stating every new construct through `emod fmt`

**Behavior:** One shared model states every construct this batch adds, and `emod fmt` writes it back
without losing any of them. This is the story's integration criterion and the assertion no individual
formatter task can make: each upstream task proves its own construct survives, while only a model
carrying all of them at once proves the writers compose — that adding one construct's line did not
displace another's, and that the canonicalisation Tasks 1-4 introduced did not drop something on the
way through.

**Acceptance Criteria:**
- [ ] `internal/test/fixtures.go` gains one model constant stating, in one file: the `emod` version
      header; a `description` on the model, an actor, a context, an aggregate, a slice, a trigger, a
      command, an event, a view, an automation and a translation; a named `invariant` on an aggregate
      and on a DCB context; a `spec` in both slice homes covering all four `then` shapes — an event
      list, `rejected <name>`, `view <Name>` and `command <Name>`; example payloads on `given`, `when`
      and `then` references, at least one short enough to stay on one line and one long enough to
      wrap; a `flow` block mixing `command -> event:` and `command -> rejected:` entries; a wire
      `type` on an event; and an automation firing `after` an elapsed duration
- [ ] The fixture omits at least one instance of each optional construct *mid-block* rather than at
      the end, so a writer that stops emitting after the first omission is caught
- [ ] `internal/test/models.go` gains the parsing accessor for it, following the existing
      `SpecLibraryLendingModel(t)` shape (`:37`)
- [ ] `internal/oracle/oracle_test.go`'s `"clean input"` group (`:24`) gains a leaf asserting
      `oracle.Check` reports nothing for it, so the fixture is pinned as a model `emod validate` and
      `emod lint` both accept — the DCB half needs tags on its events and a `decides_on` reaching them
      or `dcb/untagged-event` and `dcb/orphan-tag-key` fire
- [ ] Parsing the fixture, formatting it and re-parsing the output yields a model equal to the
      original under `test.RequireEqual` with `ignoreFormatterNormalizations` — the comparison being
      against the original model, never against a second format run
- [ ] Deleting any one of the five upstream emissions — the view/command outcome, the rejection entry,
      the payload, the wire type, the `after` suffix — makes that comparison fail rather than the
      output silently losing the construct, verified for each by removing the emission and observing
      the failure
- [ ] Formatting the formatter's own output of the fixture produces byte-identical text
- [ ] `internal/cli/fmt_test.go` gains a canonical `*FormattedEmod` constant for the fixture, opening
      with the `emod` version header, written as the canonical text `emod fmt` produces rather than
      the fixture source re-indented, and fed to `requireFmtSettlesOn` (`:629`)
- [ ] That constant shows all four canonicalisations at once: every spec's `given`/`when`/`then`
      aligned, the mixed `flow` block's `:` aligned, `subscribes` above `fields` in the view, the
      wrapping payload one field per line with its values aligned
- [ ] `emod fmt --check` over the canonical text reports no change needed and leaves the file on disk
      unchanged
- [ ] `git diff` adds the fixture and its accessor without moving any existing constant in
      `internal/test/fixtures.go`, and moves no existing golden or `*FormattedEmod` constant in
      `internal/formatter` or `internal/cli`

**Affected Files/Modules:**
- `internal/test/fixtures.go` — the new model constant
- `internal/test/models.go` — its parsing accessor, beside `SpecLibraryLendingModel` (`:37`)
- `internal/oracle/oracle_test.go` — one leaf in `"clean input"` (`:24`)
- `internal/formatter/formatter_test.go` — one leaf in `"round-trip through the parser"` (`:702`),
  folded into the existing per-fixture group rather than a parallel table
- `internal/cli/fmt_test.go` — the canonical constant and its `requireFmtSettlesOn` leaf

**Patterns to Follow:**
- The fixture kit: `tasks/learnings.md` "A new optional field ships a six-part fixture kit, not a
  bespoke model per package" (`:216`) and "Shared fixtures come in an unfeatured/featured pair,
  guarded by a walk that must be extended" (`:66`) — `test.SpecLibraryLending` and
  `test.AutomationReadsLibraryLending` in `internal/test/fixtures.go` are the two worked examples
- `tasks/learnings.md` "A new shared fixture owes `internal/oracle` a zero-diagnostic subtest"
  (`:151`), including its warning about DCB shapes tripping lint rules
- `tasks/learnings.md` "Exercise an omitted optional part mid-block, never as the last entry" (`:91`)
- The round-trip leaf shape: `internal/formatter/formatter_test.go:1037` (the whole-shared-model leaf)
- `tasks/learnings.md` "`emod fmt` canonicalises order, so a fmt golden is never the input
  re-indented" (`:141`) and "Formatter output always begins with `emod N`" (`:31`)

**Testable:** Yes

**Verification:** `mise exec -- task test` is green; `emod fmt --check` over a temp copy of the
canonical text exits 0 (copy first — `emod fmt` writes in place, `tasks/learnings.md:336`).

**Depends on:** Tasks 1-4, and **all five upstream formatter tasks** — US-007 Task 5, US-009 Task 1,
US-010 Task 5, US-012 Task 3, US-013 Task 4. This is the last task of the batch to land.

---

## Summary

**Total tasks:** 5.

**Ordering rationale — measured risk first, then blocked-on-upstream in dependency order.**

Task 1 is the only criterion that changes bytes an author already has on disk, and the walk above put
a number on it: 32 view blocks across 12 files, all of them in the order the reorder inverts. It goes
first, alone, so its churn is one reviewable commit and the three expected-value sites it moves are
named up front rather than discovered mid-run. Task 2 follows because it needs nothing from this batch
either, and because it is where the column-width shape gets extracted — the learning about
de-duplicating before a fan-out edit applies with three new alignment sites incoming, and Tasks 3 and
4 both consume the extraction. Tasks 3 and 4 then wait on the constructs they align: US-009 Task 1 for
the second flow entry kind, US-010 Task 5 for payloads. Neither can be pulled forward without becoming
vacuous, and a vacuous formatter task is precisely the failure mode `tasks/learnings.md` records for
this package. Task 5 is last because it is the integration criterion and depends on every upstream
formatter task in the batch.

**Story acceptance criteria coverage:**

| Story criterion | Task |
|---|---|
| A file using every new construct formats round-trip without data loss | Task 5 (needs all five upstream formatter tasks) |
| Attributes inside a block follow the canonical order: `description`, pattern-specific, `fields`, `spec` last | Task 1 |
| `given` / `when` / `then` keywords align within a spec | Task 2 |
| The `:` aligns across `command -> event:` and `command -> rejected:` within a `flow` block | Task 3 |
| Payloads stay on one line when they fit; otherwise one field per line with values aligned | Task 4 |

**Nothing deferred.** All five story criteria are covered. The two that are not achievable from this
story alone — the round-trip integration criterion and, within it, the emission of each new construct
— are covered by naming their upstream owners rather than by re-doing their work.

**Not touched by any task here:** `internal/lexer` (this story adds no keyword, so
`internal/lsp/keywords_test.go` and `editors/tree-sitter-emod/test/queries/keywords_test.go` both stay
green unedited), `internal/parser`, `internal/validator`, `internal/linter`, `internal/export`,
`internal/glossary`, `internal/diagram`, `internal/viewer`, `docs/`, `examples/` and
`editors/vscode/`. The one exception is a single tree-sitter corpus case in Task 4, for a wrapped
payload shape no checked-in file carries — a corpus addition, not a grammar edit, and it is named in
that task's change-set list.

# US-001: Flag a view nothing reads

## Story Reference

`user-stories/lint-unread-view.md` → **US-001: Flag a view nothing reads** (first of two stories in
"Flagging a Read Model Nothing Reads").

**In scope:** one lint rule, `view/never-read`, at warning severity, firing once per declared `view`
whose name appears in no `reads` anywhere in the model, positioned at the view's `NamePos`; a
model-wide `readViews` set filled from all three spellings of `reads` — a trigger's, an automation's
and a translation's; a model-wide guard that keeps the rule silent until the model states at least
one `reads`; the rule's entry in `internal/linter/descriptions.go` so `emod lint --explain
view/never-read` answers it; the rule reaching an author through `emod lint` in both output formats,
through `emod validate`, and through the LSP's published diagnostics by virtue of carrying
`diagnostic.Warning`; and the reference's account of the rule. Carried along because the rule is what
makes them warn: the five shared fixtures in `internal/test/fixtures.go` that declare
`MemberLoansView` and give it no reader, and the four expected values in two test files that restate
those fixtures' own text.

## Boundaries

**Out of scope** — carried from the story's Non-Goals:

- A suppression directive for a view that is legitimately unread — a datalake export, say. The tool
  has no ignore mechanism for any rule today, and introducing one for this rule alone would be the
  wrong place to start that conversation
- Making `reads` multi-valued so one screen can consult several views. Real screens do, and the
  single-valued field is a genuine constraint, but it is a grammar change with its own consequences —
  `docs/proposals/screens-as-first-class-proposal.md` takes it up
- A distinct `screen` or `ui` element separate from `trigger`. The same discussion as multi-valued
  `reads`, and it should not be settled by a lint rule; that proposal makes the case for it
- Checking the other end: whether a view's `subscribes` actually supply the fields its consumer needs
- Any automatic fix, such as deleting an unread view
- Changing `god-view`, which counts subscriptions and is unrelated

**Out of scope** — added by this decomposition:

- **US-002, "Resolve a trigger's and a translation's `reads`", is a following story and none of it is
  built here.** This rule *reads* all three spellings; it resolves none of them. `modelIndex.viewNames`
  in `internal/validator/validator.go` keeps checking an automation's `reads` alone, and the
  reference's statement that a trigger's and a translation's are "recorded and left unchecked"
  (`docs/dsl-reference.md:820`, and the same claim under **Automation Pattern** at `:381`) stays as
  written. `internal/validator` is not edited by any task here. The window US-002's own **Depends on**
  note describes is accepted: until it lands, a misspelled view name in a trigger or a translation can
  present itself as an unread view elsewhere in the file
- **Promoting the rule to an error, or a knob for choosing.** Settled: `view/never-read` is a
  `warning()` entry, matching `automation/missing-todo-list`, the rule guarding the other end of the
  same relationship. There is no promotion path in this breakdown and no configuration to add
- **Scoping the guard per context.** Settled for this story: the "only fires once the model states at
  least one `reads`" guard is model-wide, mirroring the model-wide `readViews` set the story's
  Context section specifies. See **Deferred** below
- **Sorting, ordering or grouping lint output.** `Lint` already position-sorts its whole result
  (`internal/linter/linter.go:121`, `byPosition` at `:130`), so this rule's diagnostics interleave
  with every other rule's by line and column with no work. The ordering owed here is an assertion,
  not a change
- **Treating a view read only by a trigger differently from one read by an automation.** The story's
  third open question turns on US-002; all three spellings count equally here
- **The viewer, the diagrams, the exports, the importer, the grammar and the editor surfaces.** No AST
  field is added and no serialized key moves. `internal/formatter`, `internal/export`,
  `internal/importer`, `internal/cue`, `internal/diagram`, `internal/viewer`, `internal/lsp`,
  `internal/glossary` and `editors/` hold no production edit. Task 1 does touch one *expected value*
  in `internal/formatter/formatter_test.go`, because that value transcribes a fixture it changes
- **`README.md` and `examples/`.** No `.emod` file in the tree fires the rule (measured — see
  **Codebase Context**), so none is edited, and the reference is the only document the story's
  criteria name

**Deferred:**

- Scoping the guard per context rather than per model — the story's second open question. A large
  model where one context has adopted views and another has not is reported on the strength of the
  first. Nothing here forecloses it: the guard is one model-wide fact, and narrowing it later is a
  change to where that fact is computed
- Resolving a trigger's and a translation's `reads` — US-002, in the same story file
- A rule about the other end of the relationship (whether a view's `subscribes` supply the fields its
  consumer reads) — the story lists it as a non-goal and nothing here begins it

---

## Codebase Context

**The linter.** `internal/linter/linter.go` is one `Lint(model)` walk (`:50-124`). It builds two
model-wide facts before touching any node — `flowCount` for `left-chair` (`:57-63`), and
`hasSpec`/`exercisedCommands`/`commandsWithRejection` for the spec rules (`:65-78`) — then per context
runs the mode-aware and DCB checks, then calls `checkSlice` once per slice through `ctx.SliceRefs()`
(`:109-115`), which yields both slice homes. `checkSlice` (`:140-191`) takes those model-wide facts as
its trailing parameters and dispatches per construct; its view loop is `:177-184` (`checkViewNaming`,
`checkGodView`) and its automation loop `:185-189` (`checkMissingTodoList`). Three entry constructors
sit at the top of the file — `info` (`:17`), `warning` (`:28`), `errorEntry` (`:39`) — each taking a
position, a rule name and a message. `Lint` ends with `slices.SortFunc(diags, byPosition)` (`:121`),
so every rule's diagnostics arrive in source order by line then column whatever order the walk
produced them in. The `dcb/`, `spec/`, `flow/` and `automation/` prefixes are the file's precedent for
a namespaced rule.

**The nearest rule.** `checkMissingTodoList` (`:600-612`) guards the other end of the same
relationship: an `automation` with no `reads`. It is a `warning`, reports at `NamePos`, states the
finding, then a semicolon, then the remedy. Its `ruleDescriptions` entry
(`internal/linter/descriptions.go:30`) opens by naming the shape, says what it costs, and ends with
the remedy. `spec/command-without-spec` (`:32`) is the precedent for the guard's wording: "The rule
fires only when the model already states at least one spec elsewhere, so a model that has not adopted
specs reports nothing."

**Rule descriptions.** `internal/linter/descriptions.go` is a 23-entry `ruleDescriptions` map plus
`RuleDescription(name) (string, bool)`. `RunLintExplain` (`internal/cli/lint.go:93`, wired at
`internal/cli/app.go:88`) is its only caller. Note the file is **not** `gofmt`-clean at HEAD: `gofmt
-d` on it produces a 44-line realignment of map keys the change does not touch, and no CI step runs
`gofmt` (`.github/workflows/ci.yml` runs `task build` and the five test targets only). Add the entry
without reformatting the rest of the map.

**Severity and exit codes.** `diagnostic.Entry` (`internal/diagnostic/entry.go`) carries `Severity`
and `RuleName`; `String()` prints `<file>:<line>: [<rule>] <message>`. `formatJSON`
(`internal/cli/lint.go:46`) emits `file`, `line`, `rule`, `severity`, `message`;
`exitCodeForDiagnostics` (`internal/cli/export.go:74`) returns 2 if any entry is an error and 1
otherwise, so a warning-only run exits 1. `RunValidate` (`internal/cli/validate.go:27`) is
`oracle.Check` and returns a `*LintError` for *any* diagnostic, severity ignored.

**Where the linter runs.** `oracle.Run` (`internal/oracle/oracle.go:26`) is lexer → parser →
validator → linter, and it is what `internal/cli/validate.go:27`, `internal/cli/lint.go:118`,
`internal/cli/export.go:33`, `internal/cli/diagram.go:35` and `:156`, `internal/lsp/server.go:391`
and `internal/wasm/pipeline.go:44` all call. A warning-level rule is therefore not free: it turns
every exit-0 and every zero-diagnostic assertion red for every model exhibiting the shape.

**Measured blast radius.** Every shared fixture, every `.emod` file in the tree and every ` ```emod `
block fenced in `README.md` and `docs/dsl-reference.md` was parsed with the repo's own lexer and
parser, its declared views compared against every `reads` its triggers, automations and translations
state, and the "model states at least one `reads`" guard applied. Then the rule was prototyped and
`task test:unit` and `task test:integration` run. Both are verified fact, not estimate.

Five shared fixtures fire, all on the same view:

| Fixture (`internal/test/fixtures.go`) | `reads` stated | Unread view | Reader it has today |
|---|---|---|---|
| `InvariantLibraryLending` (`:314`) | 1 | `MemberLoansView` | trigger reads `AvailableCopiesView`, which the model does not declare |
| `SpecLibraryLending` (`:423`) | 1 | `MemberLoansView` | same |
| `RejectionLibraryLending` (`:594`) | 1 | `MemberLoansView` | same |
| `SlicePatternLibraryLending` (`:759`) | 3 | `MemberLoansView` | no trigger at all; its automations and translation read its other three views |
| `WireTypeLibraryLending` (`:1765`) | 2 | `MemberLoansView` | same |

`PayloadLibraryLending` (`:1589`) declares `MemberLoansView` unread but states **zero** `reads`, so
the guard suppresses it — it is the repository's checked-in witness that a model which has not adopted
the concept reports nothing. `HotelReservation`, `DescribedHotelReservation`,
`KeywordFieldSearchCatalog`, `AutomationReadsLibraryLending`, `TriggerReadsLibraryLending` and
`AutomationScheduleLibraryLending` have every declared view read and do not fire.

No `.emod` file in the tree fires. `examples/all_patterns.emod`, `examples/dcb_model.emod`,
`internal/parser/testdata/{all_patterns,minimal,multi_context}.emod` and the editor test corpora have
every declared view read; `examples/error_diagnostics_test.emod` declares `AvailableRoomsView` unread
but states zero `reads`, so the guard suppresses it. No fenced ` ```emod ` block in `README.md` or
`docs/dsl-reference.md` fires either, which matters because `internal/oracle/oracle_test.go:83`
("documented models") runs `oracle.Check` over every one of them and requires an empty result.

With the rule prototyped and no fixture edited, exactly seven leaves fail, in three packages:

- `internal/oracle/oracle_test.go` — the "clean input" `require.Empty` leaves for `SpecLibraryLending`
  (`:55`), `RejectionLibraryLending` (`:67`), `SlicePatternLibraryLending` (`:73`) and
  `WireTypeLibraryLending` (`:79`)
- `internal/oracle/oracle_test.go:133` — the "invariants never exercised" leaf for
  `InvariantLibraryLending`, whose transcribed list gains an entry
- `internal/linter/linter_test.go:3929` — "the shared rejection fixture exercises every edge it
  states", a `require.Empty` over `RejectionLibraryLending`
- `internal/cli/export_test.go:129` — `RunExport(path, "json")` over `WireTypeLibraryLending` asserted
  `NoError` with `require.Empty(t, doc["diagnostics"])`; `oracle.Run` lints, so a warning makes it
  exit 1

**The repair, measured.** Giving each of the five fixtures a reader for `MemberLoansView` — repointing
the trigger the first three already carry, and adding the same trigger block to the two that have
none — leaves all seven of those leaves passing untouched, and breaks exactly four expected values
that restate a changed fixture's own text: `reads AvailableCopiesView` inside `specFormattedEmod`
(`internal/cli/fmt_test.go:257`) and `rejectionFormattedEmod` (`:755`), the trigger block
`slicePatternFormattedEmod` (`:922`) does not yet carry, and the `triggerReads` field of the
`WireTypeLibraryLendingModel` row in the formatter round-trip table
(`internal/formatter/formatter_test.go:1487-1493`), which expects `nil` today. With those four
updated and the rule reverted, `task test:unit` and `task test:integration` are both green — verified.
`internal/diagram`, `internal/export`, `internal/glossary`, `internal/importer`, `internal/lsp` and
`internal/validator` need no edit, although the repointed and added triggers do give three of those
fixtures a `reads` arrow the pictures did not draw before.

**Two fixtures must keep `reads AvailableCopiesView`.** `AutomationReadsLibraryLending` (`:1041`) and
`AutomationScheduleLibraryLending` (`:1407`) carry the same dangling trigger reference, and neither
fires — their views are read by automations. `internal/formatter/formatter_test.go:1439` asserts
`require.Contains(t, formattedTwin, "reads AvailableCopiesView")` by name, and the round-trip table
transcribes `triggerReads: []string{"AvailableCopiesView"}` for both (`:1462`, `:1477`). Leave them
alone.

**Linter tests.** `internal/linter/linter_test.go` is one `TestLint` umbrella of eighteen top-level
groups, one per rule or area, ordered roughly as the rules were added; the four most recent are named
for the rule itself — `"automation/missing-todo-list"` (`:3472`), `"flow/rejection-without-spec"`
(`:3743`), `"spec/command-without-spec"` (`:3933`), `"spec/given-outside-boundary"` (`:5015`). Models
are built in Go, not parsed, with positions set by hand. `"all rules together"` (`:914`) is the one
leaf that fires several rules at once.

**CLI tests.** `internal/cli/lint_test.go` holds `TestLint` and `TestLintExplain` (`:1588`). The
per-rule fixture constants at the top of the file each carry a comment naming precisely which rules
they fire — `automationWithoutViewEmod` (`:165`) is the sibling rule's, "which
automation/missing-todo-list reports and no other rule does" — and the leaves that use them assert
`require.Len(t, entries, 1)`. `"automation reading no view"` (`:1250`) is the JSON-plus-text pair to
copy. `"all rules have descriptions"` (`:1671`) is a hand-transcribed slice of all 23 rule names, each
run as its own subtest. `internal/cli/validate_test.go:703` ("returns error for model with only lint
warnings") is where a rule proves it reaches an author through `emod validate`; note its inline model
states no `reads` at all, so this rule stays silent on it.

**The reference.** `docs/dsl-reference.md` has **no** lint-rule table or listing section. Rules that
appear in it are written as a bullet in the section that owns the construct, in a fixed shape — a
bolded lead, a colon, the severity and the condition, and sometimes an `emod lint --explain` pointer:
`spec/invariant-never-exercised` under `### invariant` (`:281`), `flow/rejection-without-spec` under
`## 7. Flows` (`:560`), `wire/type-format` under `### Wire Types` (`:675`). `### View Pattern`
(`:343-356`) is the section that owns `view`, and it carries no bullet list today — one prose line
about `subscribes` after the fence. `god-view` and `view-naming` are documented nowhere in the file,
so this bullet is the section's first. Two constraints on editing the file: every heading is
`## <n>. <Title>` and four in-document links cite those numbers, so adding or reordering a numbered
section renumbers and invalidates them (a bullet inside an existing section does neither); and section
13's `Diagram Palette` table is machine-read by `dslReferencePalette`
(`internal/diagram/contract_test.go:1349`), which locates it by heading and parses its rows.

---

## Tasks

### Task 1: Give the library-lending fixtures a reader for the view they declare

**Behavior:** the five shared fixtures that declare `MemberLoansView` state who reads it. Each names
the view on a trigger — the screen a librarian works from — and each names a view its own model
declares, replacing a trigger reference to `AvailableCopiesView`, which none of them declares. The
fixtures keep validating, formatting and exporting exactly as they do today.

**Acceptance Criteria:**

- [x] The trigger in `test.InvariantLibraryLending` (`internal/test/fixtures.go:325`),
      `test.SpecLibraryLending` (`:434`) and `test.RejectionLibraryLending` (`:605`) reads
      `MemberLoansView` — the view each of those models declares in its `slice "Review Member Loans"` —
      in place of `AvailableCopiesView`, which none of them declares anywhere
- [x] `test.SlicePatternLibraryLending` and `test.WireTypeLibraryLending` each declare a trigger in
      their `slice "Borrow Copy"` reading `MemberLoansView`, written as the block the three fixtures
      above already carry
- [x] `test.DeclaredTriggerReads` over each of the five parsed models
      (`test.InvariantLibraryLendingModel` and its four siblings in `internal/test/models.go`) reads
      back exactly `[]string{"MemberLoansView"}` — a getter answering `nil` is what this criterion
      exists to rule out
- [x] `oracle.Check` returns exactly what it returns today for all five: nothing for
      `SpecLibraryLending`, `RejectionLibraryLending`, `SlicePatternLibraryLending` and
      `WireTypeLibraryLending`, and for `InvariantLibraryLending` the same five
      `spec/invariant-never-exercised` lines `internal/oracle/oracle_test.go:136-140` transcribes,
      unchanged down to their line numbers. No leaf in `internal/oracle/oracle_test.go` is edited
- [x] `git diff internal/test/fixtures.go` shows changes only inside the five fixtures named above. In
      particular `test.AutomationReadsLibraryLending` (`:1041`) and
      `test.AutomationScheduleLibraryLending` (`:1407`) keep `reads AvailableCopiesView` on their
      triggers, which `internal/formatter/formatter_test.go:1439` asserts by name, and
      `test.PayloadLibraryLending` keeps no `reads` at all, which is what makes it Task 2's witness for
      the guard
- [x] The only expected values that move are the ones restating a changed fixture's own text: the
      `reads` line in `specFormattedEmod` (`internal/cli/fmt_test.go:257`) and in
      `rejectionFormattedEmod` (`:755`), the trigger block `slicePatternFormattedEmod` (`:922`) gains
      where `emod fmt` writes it, and the `triggerReads` field of the `WireTypeLibraryLendingModel` row
      in the round-trip table (`internal/formatter/formatter_test.go:1487-1493`), which holds `nil`
      today and must name the view instead. No other golden, `*FormattedEmod` constant or transcribed
      name list moves
- [x] `git diff --stat` names exactly three files: `internal/test/fixtures.go`,
      `internal/cli/fmt_test.go`, `internal/formatter/formatter_test.go`
- [x] `mise exec -- task test:unit` and `mise exec -- task test:integration` are green

**Affected Files/Modules:**

- `internal/test/fixtures.go` — the trigger `reads` line at `:325`, `:434` and `:605`; a trigger block
  added to the `slice "Borrow Copy"` of `SlicePatternLibraryLending` (`:768`) and
  `WireTypeLibraryLending` (`:1772`)
- `internal/cli/fmt_test.go` — the canonical constants at `:245`, `:743` and `:922`
- `internal/formatter/formatter_test.go` — one row of the round-trip table at `:1487-1493`

**Patterns to Follow:**

- The trigger block to write, and where it sits in a `slice "Borrow Copy"` body, is the one already at
  `internal/test/fixtures.go:323-326`
- `test.TriggerReadsLibraryLending` (`internal/test/fixtures.go:1199`) is the fixture that exists for a
  trigger's `reads`, and its own trigger at `:1208-1211` shows the same shape naming a view another
  slice declares — the arrangement these five fixtures are being brought in line with
- `tasks/learnings.md` "A `Declared…` getter answers `nil` for a fixture that declares none of the
  construct" — the `triggerReads` row for `WireTypeLibraryLendingModel` is exactly the vacuous `nil`
  expectation that learning describes, and this task is what turns it into a real one
- `tasks/learnings.md` "A 'no expected constant moves' criterion is unsatisfiable when the task edits a
  shared fixture" — the four expected values above are named individually for that reason
- `tasks/learnings.md` "`emod fmt` canonicalises order, so a fmt golden is never the input re-indented"
  — `slicePatternFormattedEmod` is a canonical constant, so the trigger block lands where the formatter
  writes it, not where it was written in the fixture
- `tasks/learnings.md` "Only an automation's `reads` resolves; a trigger's and a translation's must
  stay unchecked" — the views named here are ones the models declare, which this rule needs and the
  validator does not check; `test.HotelReservation` remains the fixture witnessing the unchecked
  property, and is not edited
- `tasks/learnings.md` "`emod fmt <file>` writes in place, so a receipt run dirties the working tree" —
  copy to a temp path or `git checkout --` afterwards if `emod fmt` is run to derive the canonical text

**Testable:** Yes — through `oracle.Check`, `test.DeclaredTriggerReads`, `cli.RunFmt` and
`formatter.Format`, all exported, and all already exercised by leaves this task updates rather than
adds.

**Verification:** `mise exec -- task test:unit`; `mise exec -- task test:integration`;
`git diff --stat` lists the three files above and no others.

**Depends on:** None

---

### Task 2: Warn on a view nothing reads

**Behavior:** `emod lint` and `emod validate` report `view/never-read` at warning severity, once per
declared `view` whose name appears in no `reads` anywhere in the model, positioned at the view's name.
The message says who is missing and names both shapes a reader can take, so the fix is apparent from
the diagnostic. All three spellings of `reads` count as a consumer, resolution is model-wide, and the
rule stays silent on a model that states no `reads` at all. `emod lint --explain view/never-read`
describes it.

**Acceptance Criteria:**

- [x] A model declaring a view no `reads` names, in a model that states at least one `reads` elsewhere,
      produces one lint diagnostic with rule name `view/never-read`, `diagnostic.Warning` severity, and
      the filename, line and column of the view's `NamePos` — asserted with one `require.Equal` against
      the whole `*diagnostic.Entry`, not with stacked `require.Contains` calls
- [x] That diagnostic's message is exactly
      `view "MemberLoansView" is read by no trigger, automation or translation, so nothing in the model says who acts on it; give the trigger that opens on it a reads entry, or name it as a processor's todo list`
      for a view named `MemberLoansView` — pinned as one whole string and nowhere additionally asserted
      by fragment
- [x] A view read by a trigger's `reads`, one read by an automation's, and one read by a translation's
      each produce no diagnostic — asserted on a model that also declares an unread view, so the rule is
      proved to be running
- [x] A view declared in one aggregate and read by an automation in another aggregate produces no
      diagnostic, and neither does one read from another context — resolution is model-wide, matching
      `reads`
- [x] A model that states no `reads` at all produces no diagnostic however many views it declares, and
      `oracle.Check(test.PayloadLibraryLending, …)` still returns nothing, its leaf at
      `internal/oracle/oracle_test.go:61` staying in the "clean input" group unedited — the repository's
      checked-in witness for the guard
- [x] Two unread views, one in an aggregate's slice and one on a `mode dcb` context's own slice, both
      report, and the two diagnostics arrive in source order — asserted with one `require.Equal` against
      the reported lines so a misordering shows the whole list on failure
- [x] A view name declared by two slices and read once produces no diagnostic for either declaration;
      the same name declared twice and read by nothing produces one diagnostic per declaration, each at
      its own `NamePos`
- [x] A model whose every view is read produces no diagnostic from this rule
- [x] `linter.RuleDescription("view/never-read")` resolves, and `cli.RunLintExplain("view/never-read")`
      prints a description and returns no error. The description names the two shapes a reader takes and
      says the rule waits for the model's first `reads`, the way `spec/command-without-spec`'s entry
      (`internal/linter/descriptions.go:32`) states its own guard
- [x] The hand-transcribed rule list at `internal/cli/lint_test.go:1671` gains `view/never-read` as its
      24th entry, written by hand rather than derived from `ruleDescriptions`, and its subtest passes
- [x] `cli.RunLint(path, "json")` over a file whose only finding is this rule emits exactly one entry
      whose `rule` is `view/never-read`, whose `severity` is `warning`, and whose `file` and `line` name
      the view — and the returned `*cli.LintError` carries exit code 1, not 2
- [x] `cli.RunLint(path, "text")` over the same file returns a `*cli.LintError` with exit code 1 whose
      `Error()` equals the whole formatted line — path, line, `[view/never-read]` and the message above
- [x] `cli.RunValidate` over the same file returns an error whose message names the rule and the view,
      so the rule reaches an author through `emod validate` as well as `emod lint`
- [x] `internal/linter/linter_test.go`'s `"all rules together"` (`:914`) still asserts the same
      diagnostics it does today, and no expected value in that file moves for a rule other than this one
- [x] Every leaf in `internal/oracle/oracle_test.go`, `internal/cli/export_test.go` and
      `internal/linter/linter_test.go` that runs one of the five fixtures Task 1 repaired still passes
      unedited — in particular the four "clean input" `require.Empty` leaves (`:55`, `:67`, `:73`,
      `:79`), the "invariants never exercised" transcription (`:133`), "the shared rejection fixture
      exercises every edge it states" (`internal/linter/linter_test.go:3929`) and the wire-type export
      envelope (`internal/cli/export_test.go:125`)
- [x] `mise exec -- task test:unit` and `mise exec -- task test:integration` are green

**Affected Files/Modules:**

- `internal/linter/linter.go` — the model-wide fact built alongside `flowCount` (`:57-63`) and the spec
  maps (`:65-78`), and a checker beside `checkViewNaming` (`:566`) and `checkGodView` (`:614`) reached
  from `checkSlice`'s view loop (`:177-184`)
- `internal/linter/descriptions.go` — a 24th `ruleDescriptions` entry
- `internal/linter/linter_test.go` — a new top-level group named `"view/never-read"`, beside
  `"automation/missing-todo-list"` (`:3472`)
- `internal/cli/lint_test.go` — a fixture constant beside `automationWithoutViewEmod` (`:165`), a
  JSON-and-text leaf pair in the shape of `"automation reading no view"` (`:1250`), and the rule list
  at `:1671`
- `internal/cli/validate_test.go` — one leaf beside `:703`

**Patterns to Follow:**

- The whole shape of the rule — construction, positioning, the two-clause message, the
  `descriptions.go` entry, and the `checkSlice` loop it is called from — is
  `automation/missing-todo-list`: `checkMissingTodoList` (`internal/linter/linter.go:600`), its call
  site (`:185-189`), its entry (`descriptions.go:30`) and its test group
  (`internal/linter/linter_test.go:3472`). It guards the other end of the same relationship, so the two
  should read as a pair
- Where the model-wide fact is built and how it reaches the checker: `flowCount`
  (`internal/linter/linter.go:57-63`) and the `hasSpec`/`exercisedCommands`/`commandsWithRejection`
  trio (`:65-78`) are both filled by a `model.AllSlices()` pass before the per-context walk and threaded
  into `checkSlice` as trailing parameters (`:114`, `:140`). `checkCommandWithoutSpec` (`:580`) is the
  existing rule whose guard has the same shape as this one's
- Message phrasing: the existing messages state the finding, then a semicolon, then the remedy —
  `internal/linter/linter.go:568`, `:587`, `:597`, `:611`, `:616`
- `tasks/learnings.md` "`RuleName` marks a diagnostic `emod lint --explain` can describe" — a rule name
  without a `ruleDescriptions` entry ships the `orphan-command` defect that learning records, which is
  why the entry and the rule land in one task
- `tasks/learnings.md` "A lint fixture trips exactly one rule, so it is never the minimal model" — the
  `internal/cli/lint_test.go` constant needs two views (one read, one not), an automation that reads
  the first (or `automation/missing-todo-list` fires), a `flow` for every command and event (or
  `orphan-command`/`orphan-event` fire), `View`-suffixed names, fewer than five `subscribes`, events
  with more than a lone ID field, imperative command names, and no spec or invariant at all. Carry a
  comment naming exactly which rule it fires, as its siblings do
- `tasks/learnings.md` "A rule whose message branches on model state is pinned by whole formatted
  lines" — this rule has one text, so the reason to compare whole lines here is the sibling rule's
  wording, which shares the phrase "nothing in the model": a `require.Contains` on that fragment
  passes against either rule
- `tasks/learnings.md` "A second `require.Contains` on one message is often shadowed by the first" and
  "An `assert…` helper that returns the value it pinned whole makes every follow-up check dead" — pin
  the message once, whole; do not add fragment assertions beside the exact string
- `tasks/learnings.md` "A slice has two homes, and much of the repo still walks only one" — `checkSlice`
  is reached through `ctx.SliceRefs()` for both homes, and `model.AllSlices()` covers both for the
  model-wide fact; the both-homes criterion above is what proves neither was written as a one-home loop
- `tasks/learnings.md` "Diagnostics gathered from more than one AST collection must be position-sorted"
  — `Lint` already sorts (`:121`), so this applies here as the ordering *assertion* the criteria ask
  for, not as a call to add a sort
- `tasks/learnings.md` "CLI diagnostic tests must assert the distinguishing message text" — the CLI
  leaves assert the rule name and the view's name, not only a path and a count
- `tasks/learnings.md` "An assertion whose expected value comes from the code under test is the
  recurring review finding" — the transcribed rule list and the expected message are written by hand,
  never read out of `ruleDescriptions` or rebuilt from the entry constructors
- `internal/linter/descriptions.go` is not `gofmt`-clean at HEAD and no CI step checks it; add the entry
  without letting `gofmt -w` realign the 23 keys around it

**Testable:** Yes — through `linter.Lint`, `linter.RuleDescription`, `oracle.Check`, `cli.RunLint`,
`cli.RunLintExplain` and `cli.RunValidate`, all exported.

**Verification:** `go test -tags unit ./internal/linter/... ./internal/cli/... ./internal/oracle/...`;
`mise exec -- task test:unit`; `mise exec -- task test:integration`.

**Depends on:** Task 1

---

### Task 3: Describe `view/never-read` in the DSL reference

**Behavior:** an author reading the reference's account of the View Pattern learns that a view no
`reads` names is reported, at what severity, that the rule waits for the model's first `reads`, and
how to get the full description from the tool.

**Acceptance Criteria:**

- [ ] The `### View Pattern` subsection of `docs/dsl-reference.md` (`:343`) carries a bullet naming
      `view/never-read`, stating that it reports at warning severity when no trigger, automation or
      translation reads the view, that it fires only once the model states at least one `reads`, and
      pointing at `emod lint --explain view/never-read` — written in the shape of the lint bullets at
      `:281`, `:560` and `:675`
- [ ] The rule name in the bullet is byte-identical to the key `linter.RuleDescription` resolves, so
      the `emod lint --explain` invocation the bullet prints succeeds when run
- [ ] The reference's statement that a trigger's `reads` and a translation's are recorded and left
      unchecked is unchanged at `:381` and `:820` — resolving them is US-002, and this rule reads all
      three without resolving any
- [ ] No `## <n>. <Title>` heading is added, removed or renumbered: `rg -n '^## [0-9]+\.
      ' docs/dsl-reference.md` still lists thirteen headings with the same numbers and titles, so every
      number-prefixed in-document link still resolves
- [ ] No fenced ` ```emod ` block is added or edited — `rg -c '```emod' docs/dsl-reference.md` is still
      7 — so `internal/oracle/oracle_test.go`'s "documented models" leaves over the file still report
      nothing
- [ ] Section 13's `Diagram Palette` heading and its four-column table are untouched, so
      `dslReferencePalette` (`internal/diagram/contract_test.go:1349`) still parses it
- [ ] `git diff --stat` names `docs/dsl-reference.md` and no other file
- [ ] `mise exec -- task test:unit` is green

**Affected Files/Modules:**

- `docs/dsl-reference.md` — the `### View Pattern` subsection (`:343-356`)

**Patterns to Follow:**

- The three lint bullets already in the file are the form to copy, and the third is the closest fit
  because it also states a guard and points at `--explain`: `spec/invariant-never-exercised` (`:281`),
  `flow/rejection-without-spec` (`:560`), `wire/type-format` (`:675`)
- `tasks/learnings.md` "`docs/dsl-reference.md` anchors embed the section number" — a bullet inside an
  existing subsection renumbers nothing, which is why the criteria above pin the heading list rather
  than asking for a new section
- `tasks/learnings.md` "`docs/dsl-reference.md` section 13 is machine-read, so its table's spelling is
  load-bearing" — the one part of the file a Go test parses, and it is not the part being edited
- `tasks/learnings.md` "`docs/dsl-reference.md` is the one keyword surface no test reaches, and a
  retirement story forgets it" — nothing links the doc to the linter, so the rule name is transcribed by
  hand and the second criterion above is what catches a typo in it

**Testable:** No — the reference carries no assertion surface for prose. `internal/oracle`'s
"documented models" leaves parse the file's fenced ` ```emod ` blocks only, and this task adds none;
they serve here as the guard that the edit did not disturb one.

**Verification:** `mise exec -- task test:unit`; `emod lint --explain view/never-read` prints the
description the bullet points at; `rg -n '^## [0-9]+\. ' docs/dsl-reference.md` and
`rg -c '```emod' docs/dsl-reference.md` return the same thirteen headings and the same count of 7.

**Depends on:** Task 2

---

## Summary

**Three tasks**, ordered so the tree is green at every commit.

Task 1 precedes the rule because the rule cannot land first: `oracle.Run` lints, and `RunValidate`,
`RunLint` and `RunExport` all return an error for any diagnostic whatever its severity, so the moment
`view/never-read` exists five shared fixtures stop being models the pipeline accepts and seven leaves
across `internal/oracle`, `internal/linter` and `internal/cli` fail. All five can gain a reader without
losing the witness they exist to be — none of them is a fixture about unread views, and three of them
already carry a trigger pointed at a view no model declares — so none is moved out of its "clean input"
leaf, which is where this story parts company with US-008's precedent. The fixture repair is measured:
five small edits, four expected values that restate a changed fixture's own text, and a green suite
with the rule reverted.

Task 2 is the rule, and it is one task rather than three because a rule name without its
`descriptions.go` entry is the defect `tasks/learnings.md` records as live in the validator, and
because a rule that fires without its guard is a wrong answer for every half-drafted model.

Task 3 is the reference, split out because it is the one surface with no assertion behind it and a
different kind of review: a reviewer of that commit reads prose against the shipped behaviour, not a
diff against a test.

**Story criteria coverage:**

| Story criterion | Task |
|---|---|
| `view/never-read` fires for a view whose name appears in no `reads`, at the view's `NamePos` | 2 |
| All three spellings of `reads` count as a consumer | 2 |
| Resolution is model-wide | 2 |
| The message names both legitimate shapes | 2 |
| The rule fires only once the model states at least one `reads` | 2 |
| `emod lint --explain view/never-read` returns a description | 2 |
| The rule appears in the reference's lint listing | 3 |
| A model whose every view is read produces no diagnostic | 2 |

Carried along, not stated by the story: the five shared fixtures that would otherwise stop being
models the pipeline accepts, and the four expected values that transcribe them (Task 1).

**Settled, not deferred:** the rule stays a warning, and the guard stays model-wide. Nothing here
builds a path to changing either.

**Deferred to the following story:** resolving a trigger's and a translation's `reads` (US-002), and
with it the reference's account of those two being left unchecked.

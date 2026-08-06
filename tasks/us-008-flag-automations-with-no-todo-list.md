# US-008: Flag automations with no todo list

## Progress
- [x] Task 1: Name the view the shared fixtures' automations read
- [x] Task 2: Name the view the repository's `.emod` models' automations read
- [x] Task 3: Warn on an automation that declares no view

---

## Story Reference

`user-stories/triggers-and-automations.md` → **US-008: Flag automations with no todo list**
(eighth of eleven stories in "Triggers and Automations"). Design notes:
`docs/proposals/triggers-and-automations-proposal.md` — the opening (`:8-11`) for why the todo list is
the point of the pattern, section 2 (`:73-86`) for `reads`, `:216-221` for the rule and its two
message cases, `:470` for the obligation to register a `descriptions.go` entry, and `:42` for the
non-goal that forbids inferring a view.

**In scope:** one lint rule, `automation/missing-todo-list`, at warning severity, firing once per
`automation` that declares no `reads`, positioned at the automation's name; two message texts chosen by
the activation form the automation states; the rule's entry in `internal/linter/descriptions.go` so
`emod lint --explain automation/missing-todo-list` answers it; the rule reaching an author through
`emod lint` in both output formats, through `emod validate`, and through the LSP's published
diagnostics by virtue of carrying `diagnostic.Warning`. Carried along because the rule is what makes
them warn: every model in the repository that `emod validate` accepts today and whose automations name
no view — three `.emod` files and three shared fixtures in `internal/test/fixtures.go` — gains the view
its automations read, and the two shared fixtures that cannot gain one without losing the witness they
exist to be stop being members of the `oracle.Check` "clean input" group.

**Out of scope:** dropping the trigger kind slot (US-004); the view→automation `reads` edge, so naming
a view still cannot move any diagram (US-005 — `internal/diagram/contract_test.go:270` states this as
its reason); lane placement (US-006); the palette (US-007); LSP hover, completion and quickfixes over
the rule, and the `RuleName` that `ConvertDiagnostics` (`internal/lsp/diagnostics.go:28`) does not
carry into an LSP diagnostic (US-009); highlighting (US-010); `docs/dsl-reference.md`, `README.md` and
the prose that documents the rule (US-011 — and note that the reference still documents the retired
automation `trigger` spelling and an Automation Pattern skeleton with no `reads`, both of which US-011
owns). Per the feature's stated non-goal (`:42`), nothing here infers a view for an automation that
declares none. Per the story's own open question, settled: the rule stays a warning. There is no
promotion path to error in this breakdown and no configuration knob to add.

**Consequences of that boundary, decided.** Eight shapes the story does not spell out:

1. *"The existing severity configuration" is the shared `warning()` helper and the surfaces that read
   `Entry.Severity`, not a config file.* The repository has no per-rule severity configuration — no
   `.emodrc`, no rule table in `Taskfile.yml`, nothing in `internal/cli` reading one (`internal/llm/config`
   is unrelated). What the criterion names is the machinery every existing rule already flows through:
   `warning()` (`internal/linter/linter.go:25`) stamps `diagnostic.Warning`; `Entry.String()` prints the
   `[rule]` bracket; `formatJSON` (`internal/cli/lint.go:46`) emits `"severity": "warning"` and
   `exitCodeForDiagnostics` keeps the exit code at 1 rather than 2; `ConvertDiagnostics` maps it to
   `SeverityWarning`. Honouring it means declaring the severity once through the helper and asserting
   each of those surfaces, not adding a mechanism.
2. *The rule has exactly two cases, chosen on the schedule.* An automation states `on` or `every` and
   the parser rejects a block stating neither or both — but `linter.Lint` runs on models the parser has
   already reported (`oracle.Check` appends lint diagnostics unconditionally), so the two cases have to
   cover every automation the rule can be handed. The schedule wording is what an automation stating a
   schedule gets; every other automation gets the event wording. A malformed automation therefore still
   yields exactly one warning and never a third message, which is what Task 3 pins.
3. *`checkSlice` has never walked automations, and this is its first automation rule.*
   `checkSlice` (`internal/linter/linter.go:118`) visits events, translations, commands and views. It is
   called once per aggregate slice and once per context slice (`:101-110`), so adding the walk there
   covers both homes a slice has by construction — and a criterion asserting both homes is what proves
   the walk was put there rather than in a one-home loop of its own.
4. *Only an automation is subject to the rule.* `trigger` and `translation` also spell `reads`, and
   `tasks/learnings.md` records that neither resolves, because `test.HotelReservation` and
   `examples/all_patterns.emod` name views on those two that no model declares. Widening this rule to
   them would fire on the flagship example; a criterion pins that a translation naming no view produces
   no diagnostic from this rule.
5. *Lint diagnostics stay in walk order; nothing is sorted.* `linter.Lint` has never position-sorted,
   and the AST holds each kind in its own field, so a slice's automation diagnostics arrive after its
   view diagnostics whatever the source order. Imposing a sort would move the order of existing rules'
   output — including the `[]any{"info", "error"}` sequence pinned at `internal/cli/lint_test.go:459` —
   and belongs to a story about lint output, not to this one. The order this story owes is
   determinism, asserted across both slice homes.
6. *Three fixtures gain a `reads`; two deliberately keep none.* `test.HotelReservation`,
   `test.DescribedHotelReservation` and `test.KeywordFieldSearchCatalog` reach
   `internal/cli/validate_test.go`, which asserts `RunValidate` returns no error, and are the
   pipeline-wide "this is a valid model" witnesses — they must stay clean, and each already declares a
   view for its automation to read. `test.AutomationReadsLibraryLending` cannot: its
   `RemindOnDueDate` omits `reads` *mid-block on purpose* (`internal/test/fixtures.go:575-577`), which
   is the witness that an omitted optional entry does not run on into the line below it, and
   `AutomationReadsLibraryLendingViewNames` is documented as reading back shorter than the automation
   count. `test.AutomationScheduleLibraryLending` holds the only automations in the repository that
   state a schedule and no view, so it is the only model that can witness the second message text at
   all. Both keep their bytes and move out of the `oracle.Check` "clean input" group into a leaf that
   names exactly which automations warn — which is stronger coverage than `require.Empty`, and honest:
   they are models this rule is about.
7. *This story edits `examples/` and `internal/parser/testdata/` even though US-011 owns documentation
   and examples.* `emod validate` exits 0 today for five `.emod` files (verified); three of them declare
   automations with no `reads` and would exit 1 the moment the rule lands, one of those
   (`internal/parser/testdata/all_patterns.emod`) under an e2e assertion that pins `EXIT:0`
   (`e2e/tests/validate.test.ts:11`). A story that makes a file in the tree warn owns fixing that file.
   `examples/all_patterns.emod` is byte-identical to that fixture (verified with `cmp`) and takes the
   same edit — leaving the flagship example as the one model in the repository that its own linter
   complains about would be a worse outcome than the boundary is worth. The twinning is a convenience,
   not an invariant: nothing asserts the two copies match, and US-011 diverges them. What stays US-011's is the prose: the
   Automation Pattern skeleton and bullets in `docs/dsl-reference.md:322-340`, the cross-reference table
   and `README.md`.
8. *The transcribed rule list in `TestLintExplain` stays hand-written.* `internal/cli/lint_test.go:502`
   names all fourteen rules in a literal slice. Deriving it from `ruleDescriptions` would make the
   assertion read its expectation out of the map it is checking, which `tasks/learnings.md` records as
   the repository's recurring review finding. The list gains a fifteenth entry by hand.

**Learnings folded in** from `tasks/learnings.md`: `RuleName` marks a diagnostic `emod lint --explain`
can describe, and naming a rule obliges you to register its description; a new shared fixture owes
`internal/oracle` a zero-diagnostic subtest, and a fixture that trips a lint rule breaks every
downstream CLI test asserting exit 0; a slice has two homes, and much of the repo still walks only one;
exercise an omitted optional part mid-block, never as the last entry; only an automation's `reads`
resolves, and a trigger's and a translation's must stay unchecked; CLI diagnostic tests must assert the
distinguishing message text; a second `require.Contains` on one message is often shadowed by the first,
so assert the whole formatted line; assert a short keyword in a diagnostic with a `\b`-bounded
`require.Regexp` — `on` hides inside `automation` and `description`; an assertion whose expected value
comes from the code under test cannot fail; diagnostics gathered from more than one AST collection have
an order worth pinning; a `Declared…` getter answers `nil` for a fixture that declares none of the
construct; additive output changes owe a byte-identical receipt for models that do not use the feature;
acceptance criteria describe the working tree, and a commit-message receipt is the commit author's
obligation, never a criterion; run repo tooling through `mise exec --`.

---

## Codebase Context

**Linter.** `internal/linter/linter.go` is one `Lint(model)` walk (`:47-114`) that builds a
model-wide flow count, then per context runs the mode-aware and DCB checks, then calls `checkSlice`
once per aggregate slice (`:101-105`) and once per context slice (`:108-110`). `checkSlice` (`:118-151`)
dispatches to per-construct checkers over `slice.Events`, `slice.Translations`, `slice.Commands` and
`slice.Views`; `slice.Automations` is not visited by any rule today. Three constructors sit at the top
of the file — `info` (`:14`), `warning` (`:25`) and `error` (`:36`) — each taking a position, a rule
name and a message. Every checker takes the AST node and returns `*diagnostic.Entry` or nil, and
reports at the node's `NamePos`. The `dcb/` prefix is the file's only naming precedent for a namespaced
rule (`:231`, `:342`, `:394`).

**Rule descriptions.** `internal/linter/descriptions.go` is a fourteen-entry `ruleDescriptions` map and
the `RuleDescription(name) (string, bool)` lookup over it. `RunLintExplain`
(`internal/cli/lint.go:93`) is the only caller: an unknown name returns a `LintError` wrapping
`ErrUnknownRule` with exit code 1, a known one prints the description and returns nil. The command is
wired at `internal/cli/app.go:81-89`, where `--explain` short-circuits ahead of the file argument.

**Severity and exit codes.** `diagnostic.Entry` (`internal/diagnostic/entry.go`) carries
`Severity` (`Error`, `Warning`, `Info`) and `RuleName`, and `String()` prints
`<file>:<line>: [<rule>] <message>` when the rule name is set. `formatJSON`
(`internal/cli/lint.go:46`) emits `file`, `line`, `rule`, `severity`, `message` and returns exit 2 if
any entry is an error, 1 otherwise; `formatText` (`:82`) concatenates `String()` and always returns
exit 1. `exitCodeForDiagnostics` (`internal/cli/export.go:68`) is the same rule for `emod export`, and
`internal/cli/diagram.go:62-75` branches the same way. `ConvertDiagnostics`
(`internal/lsp/diagnostics.go:9`) maps `Warning` to `SeverityWarning` and everything else to
`SeverityError`, and drops `RuleName` entirely.

**Where the linter runs.** `linter.Lint` is called from `internal/cli/lint.go:138`,
`internal/cli/export.go:56`, `internal/cli/diagram.go:58` and `:187`, `internal/lsp/server.go:405`,
`internal/wasm/pipeline.go:52`, and `oracle.Check` (`internal/oracle/oracle.go:21`), which is what
`emod validate` runs (`internal/cli/validate.go:39`). `oracle.Check` appends lint diagnostics whether or
not the parser or validator reported anything, so a lint rule sees half-parsed models.

**Linter tests.** `internal/linter/linter_test.go` is one `TestLint` umbrella of twelve top-level
groups named for the area under test — "event naming" (`:21`), "command naming" (`:290`), "view naming"
(`:427`), "coupling and cohesion" (`:499`), "all rules together" (`:913`), "context mode mismatches"
(`:1048`), "untagged events" (`:1621`), "dcb/query-too-broad" (`:1895`), "dcb/single-tag-everywhere"
(`:2226`), "dcb/orphan-tag-key" (`:2807`). Models are built in Go, not parsed from source, and
positions are set by hand. "all rules together" (`:914`) is the one leaf that fires several rules at
once and asserts each rule's severity and line.

**CLI lint tests.** `internal/cli/lint_test.go` holds `TestLint` and `TestLintExplain` (`:475`).
`TestLintExplain` has a known-rule leaf, an unknown-rule leaf asserting `unknown rule` and the name, and
"all rules have descriptions" (`:502`) — a hand-transcribed slice of all fourteen rule names, each run
as its own subtest. The severity leaves at `:419-471` are the shape for asserting a rule's JSON
severity, its exit code and its text line together; `writeTemp` and `captureStdout` are the helpers.

**Fixtures.** `internal/test/fixtures.go`. `HotelReservation` (`:13`) declares `view ReservationsView`
(`:48`) and `automation AutoConfirm` (`:66`) with no `reads`; `DescribedHotelReservation` (`:100`)
mirrors it (`:146`, `:167`); `KeywordFieldSearchCatalog` (`:208`) declares `view SavedSearchesView`
(`:249`) and `automation AutoShare` (`:271`) with no `reads`. `AutomationReadsLibraryLending` (`:578`)
has four automations, of which `RemindOnDueDate` (`:646`) states no `reads`;
`AutomationScheduleLibraryLending` (`:746`) has six, of which `RecallOnSecondReminder` (`:819`),
`SweepOverdueLoans` (`:823`) and `SweepIdleDesks` (`:906`) state none — the last two on a schedule, and
one of them on a context slice. `InvariantLibraryLending` and `SpecLibraryLending` declare no
automation at all. The twins, getters and `declaredSlices` walk sit at `:989-1100`;
`internal/test/models.go` holds one parsing accessor per fixture.

**Who reads which fixture.** `HotelReservation` and `KeywordFieldSearchCatalog` reach
`internal/cli/validate_test.go` (`validEmod` at `:20`, asserted `NoError` at `:30`, `:490`, `:650`);
`AutomationReadsLibraryLending` and `AutomationScheduleLibraryLending` reach only
`internal/{diagram,export,formatter,importer,oracle,test}` — no CLI test and nothing that runs the
linter except the `oracle.Check` leaves at `internal/oracle/oracle_test.go:65` and `:71`. No diagram
test uses `HotelReservation`, no formatter or CLI golden is derived from any of these three fixtures
(`internal/cli/fmt_test.go`'s canonical constants are independent literals), and the two
`HotelReservation` export leaves (`internal/export/export_test.go:1351`, `:3255`) assert only that an
undescribed model carries no `description` key.

**`.emod` files.** `emod validate` exits 0 today for `examples/all_patterns.emod`,
`examples/dcb_model.emod`, `internal/parser/testdata/all_patterns.emod`,
`internal/parser/testdata/minimal.emod` and `internal/parser/testdata/multi_context.emod`, and exits 1
for `examples/error_diagnostics_test.emod` and `internal/parser/testdata/invalid.emod` (verified).
The two `all_patterns.emod` files are byte-identical (verified). `all_patterns.emod` declares
`view AvailableRoomsView` (`:43`) in slice 2 and `automation ConfirmationEmailReactor` (`:77`) alone in
slice 4; `multi_context.emod` declares `automation OrderNotifier` (`:26`) and no view anywhere.
`e2e/tests/validate.test.ts:11` pins `EXIT:0` for the testdata twin;
`internal/parser/integration_test.go` asserts whole `ast.Automation` values for both files (`:236`,
`:322`) and per-slice comment text for `all_patterns.emod` (`:114-144`);
`internal/importer/importer_test.go:77` and `:90` round-trip both through the diagram document.

**Not touched, deliberately.** `internal/validator` (this is a rule, not a hard error),
`internal/formatter`, `internal/export`, `internal/importer`, `internal/cue`, `internal/diagram`,
`internal/viewer`, `editors/`, `docs/` and `README.md`. No AST field is added and no serialized key
moves: the rule reads `Automation.Reads`, `Automation.Schedule` and `Automation.NamePos`, all of which
US-001 and US-003 already landed.

---

## Tasks

### Task 1: Name the view the shared fixtures' automations read

**Behavior:** the three shared fixtures that stand for "a valid model" throughout the repository, and
the inline model `internal/cli/validate_test.go` uses for the same purpose, state the canonical
todo-list shape: every automation they declare names the view it reads. They keep validating and
formatting exactly as before, and the views they name are views those models already declare.

**Acceptance Criteria:**
- [x] `AutoConfirm` in `test.HotelReservation` (`internal/test/fixtures.go:66`) names the view the same
      model declares at `:48`, and `test.DeclaredAutomationReads(test.HotelReservationModel(t))` reads
      back that one view name — a getter answering `nil` is what this criterion exists to rule out
- [x] `AutoConfirm` in `test.DescribedHotelReservation` (`:167`) names the view declared at `:146`, and
      the same getter over `test.DescribedHotelReservation` reads it back
- [x] `AutoShare` in `test.KeywordFieldSearchCatalog` (`:271`) names the view declared at `:249`, and
      the same getter reads it back
- [x] `oracle.Check` over all three fixtures still returns no diagnostics — the view an automation now
      names resolves, so no `does not exist` error appears
- [x] The model at `internal/cli/validate_test.go:322` declares a view and both its automations
      (`OrderNotifier`, `Sender`) read it, and `cli.RunValidate` over it still returns no error
- [x] `test.AutomationReadsLibraryLending` and `test.AutomationScheduleLibraryLending` are not edited:
      `git diff internal/test/fixtures.go` shows changes only inside the three fixtures named above
- [x] Every existing subtest in `internal/{parser,formatter,export,importer,diagram,cli,oracle}` passes.
      The only expected values that move are those that restate a changed fixture's own text: the
      `reads` line inside `keywordFieldFormattedEmod` (`internal/cli/fmt_test.go`) and the
      `automationReads` row for `HotelReservationModel` (`internal/formatter/formatter_test.go`). No
      other golden, `*FormattedEmod` constant or transcribed name list moves
- [x] `mise exec -- task test:unit` and `mise exec -- task test:integration` are green

**Affected Files/Modules:**
- `internal/test/fixtures.go` — the three automations at `:66`, `:167` and `:271`
- `internal/cli/validate_test.go` — the model at `:322` gains a view and two `reads` entries

**Patterns to Follow:**
- The automations of `test.AutomationReadsLibraryLending` (`internal/test/fixtures.go:650`, `:724`,
  `:729`) are the shape for an automation naming the view it reads, including where the entry sits in
  the block
- `tasks/learnings.md` "Only an automation's `reads` resolves; a trigger's and a translation's must
  stay unchecked" — the view named here must be one the model declares, unlike the names
  `test.HotelReservation` gives its trigger and its translation
- `tasks/learnings.md` "A `Declared…` getter answers `nil` for a fixture that declares none of the
  construct" — pair the getter with the name actually written, never with an empty expectation
- `tasks/learnings.md` "Additive output changes owe a byte-identical receipt for models that do not use
  the feature" — the receipt here runs the other way: the two library-lending fixtures are the models
  this task must leave alone

**Testable:** Yes — through `oracle.Check`, `cli.RunValidate` and `test.DeclaredAutomationReads`, all
exported.

**Verification:** `mise exec -- task test:unit`; `mise exec -- task test:integration`;
`git diff --stat` lists only the two files above.

**Depends on:** None

---

### Task 2: Name the view the repository's `.emod` models' automations read

**Behavior:** every `.emod` file in the repository that `emod validate` accepts states the canonical
todo-list shape — the automation reads a view of the work it has left to do — so the flagship example
demonstrates the pattern the feature exists to make default, and no shipped file is one its own linter
would complain about.

**Acceptance Criteria:**
- [ ] The automation in `internal/parser/testdata/all_patterns.emod` (`:77`) names a view of pending
      confirmations declared in its own slice, and that view subscribes to the event the automation
      activates on
- [ ] The automation in `examples/all_patterns.emod` (`:77`) gets the same edit, so both copies keep
      exiting 0. The two files are byte-identical today (verified with `cmp`) and applying one edit to
      both is the cheapest way to keep them that way, but do **not** assert the twinning: nothing in
      the suite asserts the copies match — they have separate consumers (`e2e/tests/validate.test.ts`,
      `importer_test.go` and `integration_test.go` read the fixture; only docs reference the example)
      — and US-011 deliberately diverges them by adding slices to the example alone. A `cmp` criterion
      here would fail the moment US-011 lands, whichever order the two stories run in
- [ ] The automation in `internal/parser/testdata/multi_context.emod` (`:26`) names a view its model
      declares
- [ ] `emod validate` exits 0 for `examples/all_patterns.emod`, `examples/dcb_model.emod`,
      `internal/parser/testdata/all_patterns.emod`, `internal/parser/testdata/minimal.emod` and
      `internal/parser/testdata/multi_context.emod` — the same five files that exit 0 today — and still
      exits 1 for `examples/error_diagnostics_test.emod` and `internal/parser/testdata/invalid.emod`
- [ ] The `all_patterns.emod` integration assertions still hold with the automation's expected value and
      the automation slice's contents updated in place: the model still parses with no diagnostics, the
      aggregate still has five slices, and each slice's `# Slice N: … Pattern` comment still attaches
      where `internal/parser/integration_test.go:114-144` says it does
- [ ] The `multi_context.emod` automation assertion (`internal/parser/integration_test.go:322`) names
      the view alongside the activation event, command and target context, and the fixture still
      validates with no diagnostics
- [ ] Both files still survive the diagram-document round-trip at `internal/importer/importer_test.go:77`
      and `:90`, and `all_patterns.emod` still round-trips through the formatter with its comments and
      field alignment intact (`internal/formatter/formatter_test.go:1820`, `:1938`, `:1957`)
- [ ] `e2e/tests/validate.test.ts` is not edited — the file it names exits 0 on its own merits

**Affected Files/Modules:**
- `internal/parser/testdata/all_patterns.emod` — slice 4 (`:76-81`)
- `examples/all_patterns.emod` — the same edit, keeping the twins identical
- `internal/parser/testdata/multi_context.emod` — the automation at `:26` and the model it lives in
- `internal/parser/integration_test.go` — the two expected `ast.Automation` values (`:236`, `:322`) and
  the automation slice's assertions

**Patterns to Follow:**
- The view-and-automation pairing to copy is the one in
  `internal/test/fixtures.go:610-653` — a view declared in the model, an automation naming it, and the
  event the view subscribes to being the one the automation activates on
- `docs/proposals/triggers-and-automations-proposal.md:8-11` states the canonical chain the example is
  being brought in line with; `:42` forbids inferring the view, which is why it is written out
- `tasks/learnings.md` "A new shared fixture owes `internal/oracle` a zero-diagnostic subtest" records
  the DCB tripwires to watch if any view added here lands in a `mode dcb` context — none of these three
  files declares one, so nothing extra is owed
- The five-file baseline in **Codebase Context** was produced by running `emod validate` over every
  `.emod` file in the tree; reproduce it before and after rather than reasoning about which files changed
- `tasks/learnings.md` "urfave/cli v2 discards every flag written after the file argument" — write any
  verification invocation with flags ahead of the path

**Testable:** Yes — through `cli.RunValidate` over the checked-in files and the existing integration
tests that parse them.

**Verification:** `mise exec -- task test:unit`; `mise exec -- task test:integration`;
`emod validate` over each `.emod` file in the tree reproduces the five-clean/two-failing baseline;
`cmp examples/all_patterns.emod internal/parser/testdata/all_patterns.emod` succeeds.

**Depends on:** None

---

### Task 3: Warn on an automation that declares no view

**Behavior:** `emod lint` and `emod validate` report `automation/missing-todo-list` at warning severity,
once per automation that declares no `reads`, positioned at the automation's name. An automation
activated by an event is told that nothing in the model shows what work is outstanding and to project a
view of pending work; an automation activated by a schedule is told the model does not state what the
processor acts on. The rule is describable through `emod lint --explain automation/missing-todo-list`,
and an automation that declares a view produces nothing.

**Acceptance Criteria:**
- [ ] A model whose automation states an activation event and no `reads` produces exactly one lint
      diagnostic, with rule name `automation/missing-todo-list`, `diagnostic.Warning` severity, and the
      filename, line and column of the automation's name — asserted with one `require.Equal` against the
      whole formatted diagnostic line, not with stacked `require.Contains` calls
- [ ] That diagnostic's message is exactly
      `automation "AutoConfirm" reads no view, so nothing in the model shows what work is outstanding; project a view of pending work and read it`
      for an automation named `AutoConfirm` — pinned as a whole string, since `on` occurs inside
      `automation` and a `require.Contains` on either half is satisfiable by the other message
- [ ] A model whose automation states a schedule and no `reads` produces exactly one diagnostic under
      the same rule name and severity, whose message is exactly
      `automation "SweepOverdueLoans" reads no view, so the model does not state what the processor acts on; project a view of pending work and read it`
- [ ] An automation that states a `reads` produces no diagnostic under this rule, asserted on a model
      that also carries an automation without one so the rule is proved to be running
- [ ] Two automations without `reads` — one in an aggregate's slice, one on a `mode dcb` context's own
      slice — both report, and the two diagnostics arrive in that order, asserted with one
      `require.Equal` against the reported lines so a misordering shows the whole list on failure
- [ ] A slice whose `translation` names no view, and one whose `trigger` names no view, produce no
      diagnostic under this rule — the rule reads automations only
- [ ] An automation stating neither `on` nor `every` (a block the parser has already reported) produces
      exactly one diagnostic carrying the event-activated message, and one stating both produces exactly
      one carrying the schedule message: two cases, never a third
- [ ] `linter.RuleDescription("automation/missing-todo-list")` resolves, and
      `cli.RunLintExplain("automation/missing-todo-list")` prints a description naming the todo list and
      the `reads` entry and returns no error; the hand-transcribed rule list at
      `internal/cli/lint_test.go:502` gains the name and its subtest passes
- [ ] `cli.RunLint(path, "text")` over a file whose only finding is this rule returns a `*cli.LintError`
      with exit code 1 whose message contains `[automation/missing-todo-list]`, the file path and the
      automation's line
- [ ] `cli.RunLint(path, "json")` over the same file emits one entry whose `rule` is
      `automation/missing-todo-list`, whose `severity` is `warning`, and whose `file` and `line` name the
      automation — and the returned exit code is 1, not 2: this rule never raises a run to error
- [ ] `cli.RunValidate` over the same file returns an error whose message names the rule and the
      automation, so the rule reaches an author through `emod validate` as well as `emod lint`
- [ ] `oracle.Check` over `test.AutomationReadsLibraryLending` returns exactly one diagnostic — this
      rule, at `RemindOnDueDate` — and nothing else; the fixture's other three automations, which name
      views, are silent, and the leaf moves out of the "clean input" group
      (`internal/oracle/oracle_test.go:65`) into one that states what the model warns about
- [ ] `oracle.Check` over `test.AutomationScheduleLibraryLending` returns exactly three diagnostics, all
      this rule: the event-activated message at `RecallOnSecondReminder` and the schedule message at
      `SweepOverdueLoans` in the aggregate's slice, then the schedule message at `SweepIdleDesks` on the
      context's slice — one fixture witnessing both message texts, both slice homes and the declaration
      order, and its leaf moves the same way (`:71`)
- [ ] `oracle.Check` over `test.HotelReservation`, `test.DescribedHotelReservation`,
      `test.KeywordFieldSearchCatalog`, `test.InvariantLibraryLending` and `test.SpecLibraryLending`
      still returns no diagnostics, and those five leaves stay in "clean input" unedited
- [ ] `git diff` moves no expected constant in `internal/linter/linter_test.go` for a rule other than
      this one — in particular "all rules together" (`:914`) still asserts eight diagnostics, its model
      declaring no automation

**Affected Files/Modules:**
- `internal/linter/linter.go` — `checkSlice` (`:118-151`) gains the automation walk, and a checker
  beside `checkViewNaming` (`:491`) and `checkGodView` (`:505`)
- `internal/linter/descriptions.go` — a fifteenth `ruleDescriptions` entry
- `internal/linter/linter_test.go` — a new top-level group named for the rule, beside
  `"dcb/orphan-tag-key"` (`:2807`)
- `internal/cli/lint_test.go` — the transcribed rule list (`:502`), an `--explain` leaf, and a
  severity/exit-code leaf in the shape of `:419-471`
- `internal/cli/validate_test.go` — one leaf beside `:420`
- `internal/oracle/oracle_test.go` — the two leaves at `:65` and `:71` move out of "clean input"

**Patterns to Follow:**
- Construction and positioning: `checkViewNaming` (`internal/linter/linter.go:491`) and `checkGodView`
  (`:505`) are the smallest single-node checkers in the file, and `checkSlice` (`:118`) is where they
  are called from — once for each of the two slice homes, by virtue of `Lint` (`:101-110`) calling it
  twice
- Rule naming: the `dcb/` prefix at `:231`, `:342` and `:394` is the file's namespaced-rule precedent;
  `automation/missing-todo-list` is the name the proposal fixes (`:216`)
- Message phrasing: the existing messages state the finding, then a semicolon, then the remedy —
  `internal/linter/linter.go:417`, `:488`, `:500`, `:507`. The two texts above follow it, and
  `docs/proposals/triggers-and-automations-proposal.md:218-219` is where their content comes from
- Description text: the fourteen entries in `internal/linter/descriptions.go` open by naming the shape,
  say why it costs something, and end with the remedy. This one owes the same, and should say that
  `reads` is deliberately optional so an automation can be sketched before its view exists — the
  proposal's own reason (`:42`, and the open question at `:470`) for the rule being a warning
- `tasks/learnings.md` "`RuleName` marks a diagnostic `emod lint --explain` can describe" — a rule name
  without a `ruleDescriptions` entry ships the `orphan-command` defect that learning records
- `tasks/learnings.md` "A second `require.Contains` on one message is often shadowed by the first" and
  "Assert a short keyword in a diagnostic with a `\b`-bounded `require.Regexp`" — the two messages share
  a prefix and a suffix, so only a whole-line comparison distinguishes them
- `tasks/learnings.md` "A slice has two homes, and much of the repo still walks only one" and
  "Diagnostics gathered from more than one AST collection must be position-sorted" — the second applies
  as an ordering assertion here, not as a call to sort: see the boundary note above
- `tasks/learnings.md` "CLI diagnostic tests must assert the distinguishing message text" — the CLI
  leaves assert the rule name and the automation's name, not just a path and a count
- `tasks/learnings.md` "An assertion whose expected value comes from the code under test is the
  recurring review finding" — the transcribed rule list and the expected message strings are written by
  hand, never read out of `ruleDescriptions` or rebuilt from the formatter

**Testable:** Yes — through `linter.Lint`, `linter.RuleDescription`, `oracle.Check`, `cli.RunLint`,
`cli.RunLintExplain` and `cli.RunValidate`, all exported.

**Verification:** `go test -tags unit ./internal/linter/... ./internal/cli/...
./internal/oracle/...`; `mise exec -- task test:unit`; `mise exec -- task test:integration`.

**Depends on:** 1, 2

---

## Summary

**Three tasks**, ordered so the tree is green at every commit.

Tasks 1 and 2 are independent of each other and both precede Task 3, because the rule cannot land first:
the moment it exists, three shared fixtures stop being models `emod validate` accepts, an inline CLI
model stops returning `NoError`, and an e2e assertion pinning `EXIT:0` for a checked-in `.emod` file
fails. Splitting them by artefact family keeps each commit's receipt distinct — Task 1's is the Go test
suite with no golden edited, Task 2's is `emod validate` reproducing its five-clean baseline plus `cmp`
over the twin example files. Task 3 is the rule itself, and it is one task rather than three because a
rule name without its `descriptions.go` entry is the defect `tasks/learnings.md` records as live in the
validator, and because a rule that fires with only one of its two messages is a wrong answer for every
scheduled automation in the tree.

**Story criteria coverage:**

| Story criterion | Task |
|---|---|
| `automation/missing-todo-list` (warning) fires for an automation with no `reads` | 3 |
| The event-activated message says nothing shows what work is outstanding, and suggests projecting a view of pending work | 3 |
| The schedule-activated message says the model does not state what the processor acts on | 3 |
| The rule honours the existing severity configuration and `emod lint --explain <rule>` | 3 |
| An automation with `reads` produces no diagnostic | 3 |

Carried along, not stated by the story: the three shared fixtures and one inline CLI model that would
otherwise stop validating (1), and the three checked-in `.emod` files — one of them under an e2e
`EXIT:0` assertion — that would otherwise ship as models their own linter complains about (2).

**Deferred to later stories in the feature:** dropping the trigger kind slot (US-004); the
view→automation `reads` edge, without which naming a view still cannot move a diagram (US-005); lane
placement (US-006); the palette (US-007); LSP surfacing of the rule and the `RuleName` that
`ConvertDiagnostics` drops (US-009); highlighting (US-010); and every document that describes the rule
or shows an automation, including the Automation Pattern skeleton in `docs/dsl-reference.md` (US-011).

**Settled, not deferred:** the rule stays a warning. The proposal's open question about promoting it to
an error (`:470`) is answered for this story, and nothing here builds a path to changing it.

# US-008: Lint spec coverage and boundary assumptions

## Progress
- [x] Task 1: Give every specced command in the shared spec fixture a rejection path
- [x] Task 2: Report a command no spec exercises
- [x] Task 3: Report a command whose specs never reject
- [x] Task 4: Report an invariant no rejection references
- [ ] Task 5: Report a `given` event outside the aggregate boundary
- [ ] Task 6: Report a `given` event the command's `decides_on` does not match

---

## Story Reference

`user-stories/specs-and-metadata.md` → **US-008: Lint spec coverage and boundary assumptions**
(eighth story of "Specs, Invariants, and Model Metadata", lines 105-118). Design notes:
`docs/proposals/specs-and-metadata-proposal.md` §4 "Linting" (the four-row rule table at lines
249-252 and the paragraph at line 254 naming `spec/given-outside-boundary` as the rule that pays for
the feature), line 282 (a rejection flow edge also counts as a reference for
`spec/invariant-never-exercised`), line 415 ("all new lint rules respect the existing severity and
`--explain` machinery"), and line 595 (the four lint rules belong to the same phase as the specs
themselves).

**In scope:** four lint rules, all of them Go-side additions to `internal/linter` with no change to
the language. `spec/command-without-spec` (info) for a command no spec exercises;
`spec/no-rejection-path` (info) for a command some spec exercises but none rejects;
`spec/invariant-never-exercised` (warning) for a declared invariant no `rejected` spec in its own
scope references; and `spec/given-outside-boundary` (warning) in two arms — a `given` event outside
the enclosing aggregate, and a `given` event a DCB command's `decides_on` does not name. Each rule
registers a description so `emod lint --explain <rule>` answers for it, joins the hand-maintained
rule list in `internal/cli/lint_test.go`, and carries its story-stated severity through `emod lint`
in both output formats, through `emod validate`, and into the LSP's published diagnostics. Carried
with them because the rules are what make them warn: the shared spec fixture gains a rejection spec
per command, `test.InvariantLibraryLending` leaves `oracle.Check`'s clean-input group for one named
after the rule it now trips, and the `docs/dsl-reference.md` §4 fence plus the sentence claiming
`emod lint` never reports an unreferenced invariant both move.

**Out of scope:**

- **Value-aware boundary checking (US-011).** `spec/given-outside-boundary` here is **type-level
  only**: in DCB mode it asks whether the `given` event's *type* appears in the `when` command's
  `decides_on` events list, never whether a tagged field's value matches. US-011 sharpens it once
  payloads exist (`docs/proposals/specs-and-metadata-proposal.md:254`). Nothing in this story reads
  `DecidesOnClause.Predicate`.
- **`flow/rejection-without-spec` and rejection edges (US-009).** No `flow` entry kind is added, and
  `ast.Slice` gains no rejection collection. Task 4 states the seam US-009 extends.
- **Example payload literals (US-010).** `ast.SpecElement` is read for its name and position only.
- **The formatter (US-014).** No rule changes what `emod fmt` writes; `internal/formatter` is
  untouched by every task here.
- **The LSP (US-015).** No hover, completion or navigation over specs or invariants. The rules reach
  the editor only through `ConvertDiagnostics` (`internal/lsp/diagnostics.go:9-42`), which already
  publishes whatever `oracle.Check` returns.
- **Diagrams (US-016).** No rule renders. `internal/diagram` and `internal/export/diagram.go` are
  untouched.
- **Highlighting (US-017).** This story adds no lexer keyword, so no editor grammar moves — see the
  keyword note below.
- **Examples and reference coverage (US-018).** No file under `examples/` gains a spec. The only
  `docs/dsl-reference.md` edit is Task 4's, and it lands because `spec/invariant-never-exercised`
  makes an existing sentence in §4 false and an existing fenced model stop validating — not as
  documentation of the rules themselves. No section of the reference or the README lists lint rules
  today (§12 "Pipeline" names the linter as a stage; `README.md:113-116` shows `emod lint` with no
  rule table), and the prior pure-lint story
  (`tasks/completed/us-008-flag-automations-with-no-todo-list.md`, three tasks, no documentation
  task) set that precedent.

**No lexer keyword is added, and this was checked.** `spec`, `given`, `when`, `then`, `rejected` and
`invariant` all entered `internal/lexer/token.go` with US-005 and US-006; the four rule names are Go
string constants, not language. So `lexer.Keywords()` does not grow, and neither of the two
CI-enforced drift tests has anything to gain here:
`editors/tree-sitter-emod/test/queries/keywords_test.go:47` (`TestEditorKeywordCoverage`, requiring
every spelling in all three editor grammars) and `internal/lsp/keywords_test.go:18`
(`TestKeywordCoverage`, requiring hover text per keyword) are both already satisfied for every word
these rules reason about. No task in this list touches `editors/`.

**Open questions, decided.** Six, each decided against evidence gathered from the tree, and each
chosen so that US-009 and US-011 stay additive:

1. **`spec/command-without-spec` fires only in a model that states at least one spec.** Measured over
   the tree as it stands: ungated, the rule reports **47 commands** — 28 across the eight shared
   fixtures in `internal/test/fixtures.go`, 12 across `examples/`, 7 across
   `internal/parser/testdata/`, plus 3 of the 7 ` ```emod ` fences in `README.md` and
   `docs/dsl-reference.md`, and `billingModel` in `internal/wasm/pipeline_test.go`. Since any
   diagnostic fails `emod validate` and `emod lint` regardless of severity
   (`tasks/learnings.md:466-469`), closing that would mean writing a spec for every command in every
   model the repository ships as valid — including `internal/parser/testdata/minimal.emod`, whose
   name is its purpose, and `examples/all_patterns.emod`, whose job is to show the patterns rather
   than their scenarios. The gate makes the rule opt-in per file: a model that has adopted specs is
   told which commands it has not covered; a model that has not adopted them is told nothing. The
   precedent is in the same package — `checkOrphanTagKeys` (`internal/linter/linter.go:301-304`)
   returns nil when no event declares a tag, on exactly this reasoning. With the gate, **no
   checked-in model moves for this rule**: the one model that states specs,
   `test.SpecLibraryLending`, already has a spec for each of its four commands.
2. **A spec exercises a command when its `when` names that command.** `then command <Name>` (US-007)
   states which command an automation *issues*, not a scenario for that command's own behaviour, and
   does not count. Deciding it now keeps US-007 additive: when `ThenCommand` lands, these rules are
   unchanged. The lookup is model-wide, matching how the validator already resolves a spec's `when`
   (`internal/validator/validator.go:344`), so a spec may exercise a command declared in another
   slice. `tasks/learnings.md:191-194` records the counterpart asymmetry to keep in mind: a spec is
   *not* a reference for orphan detection, so a command only a spec exercises is still
   `orphan-command`, and these rules do not change that.
3. **`spec/invariant-never-exercised` is not gated on specs.** Unlike rule 1, it is gated by its own
   subject — a model declaring no invariant cannot trip it. Gating it on specs as well would silence
   the commonest case it exists for (invariants declared and never referenced at all), and would
   break US-009: a model whose invariants are referenced only by rejection *flow edges*, with no
   specs, must not be silenced by a spec gate. It is also the one rule the story rates a warning
   while the two coverage rules are info. Ungated, it fires on exactly two surfaces, and Task 4 moves
   both: `test.InvariantLibraryLending` (5 invariants, 0 specs) and the `docs/dsl-reference.md` §4
   fence at line 175 (4 invariants, 0 specs). `test.SpecLibraryLending` already exercises both of
   its invariants and stays clean.
4. **An invariant is exercised within its own scope only, and only by a resolving reference.** A
   `then rejected <name>` in one of aggregate A's slices exercises A's invariants; the same name in a
   sibling aggregate's slice does not. This is the scope rule US-005 established and
   `invariantScope`/`unresolvedRejections` (`internal/validator/validator.go:203-264`) implement. A
   `rejected` name that resolves to no declared invariant in that scope is already a validation
   error, and does not count as a reference — otherwise a typo would silence the rule for the
   invariant it was meant to name.
5. **`spec/given-outside-boundary` picks its arm by where the slice lives, not by parsing the mode
   string twice.** A slice nested in an aggregate takes the aggregate arm (the boundary is that
   aggregate); a slice declared directly on a context takes the DCB arm (the boundary is the `when`
   command's `decides_on`). In every well-formed model this is exactly the story's wording, because
   aggregate mode puts slices under aggregates and `mode dcb` puts them on the context; it also
   gives mixed mode a definition per slice for free, and stays defined for the malformed shapes
   `dcb-in-aggregate-mode` and `aggregate-in-dcb-mode` already report. `ast.SliceRef`
   (`internal/ast/traverse.go:19-25`) carries the home, so the walk `Lint` already runs answers it.
6. **In the DCB arm, a command stating no `decides_on` puts nothing outside the boundary.** The rule
   compares a `given` event against a declared query; with no query declared there is nothing to
   compare against, and inventing "no query matches nothing" would fire on a shape the DCB rules
   already treat as unqueried — `checkQueryTooBroad` (`internal/linter/linter.go:373-376`) skips a
   command with no `decides_on` outright. This is load-bearing rather than theoretical:
   `test.SpecLibraryLending`'s `ClaimDesk` declares no `decides_on` while its spec "refuses a desk
   another reader is seated at" states `given [DeskClaimed]`, so the other reading would take the
   flagship spec fixture out of `oracle.Check`'s clean-input group for a shape the story does not ask
   about. A `given` name no slice declares is likewise skipped in both arms, since the validator
   already errors on it and a lint rule must not double-report.

**Where "the existing severity configuration" lives.** There is no configuration file, no severity
flag and no options parameter anywhere in the repo — `rg -n 'severity' --type go` over
non-test sources returns only `internal/diagnostic/entry.go`, the writers, and the three constructor
helpers. Severity is chosen statically at the call site through `info`, `warning` and `errorEntry`
(`internal/linter/linter.go:16-47`), and "respecting" it means three observable things a task can
check: the `severity` field of `emod lint -f json`, the exit code (`formatText`,
`internal/cli/lint.go:80-89`, returns 1 for any non-empty list; `formatJSON` returns 2 once an error
is present), and the LSP mapping. One live gap is worth knowing and is **not** this story's to close:
`ConvertDiagnostics` (`internal/lsp/diagnostics.go:31-36`) maps `diagnostic.Warning` to
`SeverityWarning` and everything else — including `diagnostic.Info` — to `SeverityError`, so the two
info rules here will publish as editor errors exactly as `dcb/single-tag-everywhere` already does.
Fixing that changes an existing rule's behaviour and belongs in its own commit
(`tasks/learnings.md:461-464`).

**Overarching constraint:** every existing `.emod` file stays valid with unchanged meaning. Three of
the four rules can only fire on a model that states a spec or declares an invariant, and the fourth
is gated to the same effect, so the models that move are the two Task 4 names and the one fixture
Task 1 edits — no file under `examples/` or `internal/parser/testdata/` changes in this story.

**Learnings folded in** from `tasks/learnings.md`: a lint warning fails `emod validate`, so a new
rule sweeps every checked-in model before it lands (:466-469) — the sweep evidence is stated per rule
above and re-run per task; a lint fixture trips exactly one rule, so it is never the minimal model
(:471-474) — sharpened here because the four rules are tripwires for *each other*; a rule whose
message branches on model state is pinned by whole formatted lines (:476-479) — which
`spec/given-outside-boundary` needs and the other three avoid by carrying one text each; `RuleName`
marks a diagnostic `emod lint --explain` can describe, so naming a rule obliges registering its
description (:166-169); diagnostics gathered from more than one AST collection must be
position-sorted (:181-184); a slice has two homes and much of the repo walks only one (:171-174); a
spec is not a reference, so a command only a spec exercises is still orphaned (:191-194); a spec's
`when` resolves against commands *and* events while `given`/`then` resolve against events only
(:201-204); `ast.ThenClause` dispatches through five type switches, none of which errors (:186-189);
a second `require.Contains` on one message is often shadowed by the first (:136-139); CLI diagnostic
tests must assert the distinguishing message text (:6-9); an assertion whose expected value comes
from the code under test cannot fail (:126-129); a "no expected constant moves" criterion is
unsatisfiable when the task edits a shared fixture (:481-484); a new shared fixture owes
`internal/oracle` a zero-diagnostic subtest (:151-154); an ```emod fence is a promise that the block
validates (:526-529); `docs/dsl-reference.md` sub-heading anchors are cited more often than the
numbered ones (:541-544); a `_test.go` file always carries the `Test…` umbrella for the name it wears
(:456-459); a tested improvement found on the way is still a separate commit (:461-464); and
acceptance criteria never reference commit, branch or remote state (:21-24, :246-249).

---

## Codebase Context

**The linter.** `internal/linter` is two files. `linter.go` holds `Lint(model *ast.Model)`
(`:49-103`), which builds one model-wide map at the top — `flowCount`, filled from every slice's
flows (`:57-62`) — then loops `model.Contexts`, dispatches the mode-scoped context checks, and calls
`checkSlice(ref.Slice, aggregateName, flowCount)` once per `ctx.SliceRefs()` entry (`:93-99`).
`checkSlice` (`:107-145`) is a flat sequence of per-element loops: events (twice — slice events and
the event nested in a translation), commands, views, automations. `descriptions.go` is a single
`ruleDescriptions` map (`:10-31`, seventeen entries) behind `RuleDescription(name)`. There is no rule
registry, no severity table and no options parameter: a rule is a function, a call site, one of
`info`/`warning`/`errorEntry` (`:16-47`), and a description entry.

**The four templates this story copies.** `checkLeftChair` (`:490-495`) is a per-command rule reading
a map built model-wide before the walk — the exact shape rules 1 and 2 need. `checkOrphanTagKeys`
(`:271-343`) is the rule that returns nil when the feature is unused, that collects across every
slice of a context before deciding, and that sorts its findings by declaration position with a
comment saying why Go's map order would otherwise reorder them. `checkQueryTooBroad` (`:367-391`)
shows how a `decides_on` rule skips a command that declares none. `dcb/single-tag-everywhere`
(`:238-266`) is the only info-severity rule in the tree, and its suite
(`internal/linter/linter_test.go:2226-2806`) is the template for asserting severity beside message
and position.

**Mode helpers.** `isAggregateMode` (`:149-151`) treats an empty mode string as aggregate;
`isDCBMode` (`:153-155`) and `isMixedMode` (`:157-159`) are exact matches. `Lint` already gates four
checks on DCB-or-mixed. Decision 5 above means `spec/given-outside-boundary` reads
`ref.Aggregate == nil` rather than these helpers, but they are what the existing rules read and what
a reviewer will compare against.

**The AST the rules read.** `internal/ast/ast.go:95-126` is the US-006 shape: `Spec` (name, `Given
[]*SpecElement`, `When *SpecElement`, `Then ThenClause`, positions), `SpecElement` (name and
position — US-010 hangs a payload here), and the sealed `ThenClause` with `ThenEvents` and
`ThenRejected{InvariantName, InvariantPos}`. US-007 adds `ThenView` and `ThenCommand` to that set
(`tasks/us-007-write-specs-for-view-automation-and-translation-slices.md:153-154`), and
`tasks/learnings.md:186-189` records that every site dispatching on `ThenClause` fails silently on a
variant it has not heard of. **The rules here must be written so a fifth variant is not a fifth
silent failure**: rules 1, 2 and 4 read `Spec.When` and `Spec.Given` and never dispatch on `Then` at
all, and rule 3's only interest is "is this `*ast.ThenRejected`", which stays correct as variants are
added. `Invariant` (`:68-74`) carries name, statement and positions; `Command.DecidesOn`
(`:135`) points at `DecidesOnClause` (`:246-253`), whose `Events []string` is the type-level list
this story reads and whose `Predicate` it deliberately does not.

**Traversal.** `internal/ast/traverse.go` is where the two slice homes are reconciled. `SliceRef`
(`:19-25`) pairs a slice with its `Context` and its `Aggregate` (nil for a `mode dcb` context's own
slices); `Context.SliceRefs()` (`:30-50`) returns both homes in source order, `Model.SliceRefs()`
(`:66-77`) composes them, and `AllSlices()` drops the pairing. `Lint` already consumes
`ctx.SliceRefs()`, so the home each slice has is available where the rules need it —
`tasks/learnings.md:171-174` records how many walks in this repo got that wrong.

**Invariant scoping, in the validator.** `invariantScope` and `invariantScopes`
(`internal/validator/validator.go:203-230`) build one scope per context and one per aggregate, each
holding that scope's own invariants and its own slices, with the comment at `:216-219` stating that
an aggregate and its enclosing context are separate scopes. `unresolvedRejections` (`:246-264`) is
the walk rule 4 inverts: it visits each scope's slices' specs, type-asserts `*ast.ThenRejected`, and
reports the names *not* declared; rule 4 reports the declarations *not* named. Both are unexported
and in another package, so the linter builds its own two-line-deep walk over `model.Contexts` and
`ctx.Aggregates` — the same shape `checkDCBInAggregateMode` (`internal/linter/linter.go:163-197`)
already uses. Two call sites is below this repo's de-duplication threshold
(`tasks/learnings.md:71-74`); exporting a shared scope helper from `internal/ast` would drag the
validator into the same change and is deliberately not done here.

**The pipeline.** `oracle.Run` (`internal/oracle/oracle.go:26-31`) is the one lex → parse → validate
→ lint chain, and `Check` (`:35-38`) is its diagnostics-only form. `RunValidate`
(`internal/cli/validate.go:13-...`) and `RunLint` (`internal/cli/lint.go:104-129`) both call it, which
is why `emod validate` and `emod lint` report identically and why an info-severity rule is not free.
`emod lint --explain` resolves through `RunLintExplain` (`internal/cli/lint.go:91-102`) into
`linter.RuleDescription`, and the only thing covering the description map is the hand-maintained
`rules` list in `internal/cli/lint_test.go:627-645`, so a new rule is untested there unless it is
added by hand.

**The models a new rule must not disturb, and where they are enumerated.** `oracle.Check` zero-diagnostic
leaves live in `internal/oracle/oracle_test.go`: "clean input" (`:26-110`, one leaf per shared
fixture) and "documented models" (`:112-129`, extracting every ` ```emod ` fence from `README.md` and
`docs/dsl-reference.md` — seven blocks, at `docs/dsl-reference.md:17,35,55,175,460,577` and
`README.md:25`). `internal/cli/validate_test.go` adds two more: the three-path
`internal/parser/testdata/` leaf (`:37-49`) and the `examplePaths` walk (`:52-86`), which splits
`examples/` on the `_test.emod` suffix and requires every other file to return no error.
`internal/wasm/pipeline_test.go:13-36` embeds `billingModel` and asserts an empty diagnostics
envelope. The group US-008's predecessor created for fixtures that legitimately warn —
"automations reading no view" (`internal/oracle/oracle_test.go:131-157`) — asserts whole formatted
lines through `reportedLines` (`:402`) and is the shape Task 4 copies.

**Measured blast radius, per rule.** Taken by walking every one of those models with each rule's
decided semantics:

| Rule | Fires on, before this story | Task that accounts for it |
|---|---|---|
| `spec/command-without-spec` | Nothing, given decision 1. Ungated it would fire 47 times. | Task 2 states the gate and re-runs the sweep. |
| `spec/no-rejection-path` | `test.SpecLibraryLending` twice — `ReturnCopy` and `ReleaseDesk` each carry specs with no rejection among them. | Task 1 gives both a rejection spec, ahead of the rule. |
| `spec/invariant-never-exercised` | `test.InvariantLibraryLending` 5 times (all five invariants, no specs); `docs/dsl-reference.md:175` 4 times. | Task 4 moves the fixture's oracle leaf and rewrites the fence and the §4 bullet it falsifies. |
| `spec/given-outside-boundary` | Nothing. `test.SpecLibraryLending`'s four aggregate-mode `given` events all name events of their own aggregate `Loan`; of its two DCB `given` events, `ReleaseDesk`'s is named by that command's `decides_on` and `ClaimDesk`'s is skipped under decision 6. | Tasks 5 and 6 assert the sweep stays clean. |

**Where the fixtures are transcribed.** `test.SpecLibraryLending` (`internal/test/fixtures.go:423-...`)
is mirrored by `SpecLibraryLendingSpecNames` (`:1119`), `DeclaredSpecNames` (`:1299`) and the
`WithoutSpecs` twin (`:1216`), and read back by `libraryLendingSpecs`
(`internal/export/export_test.go:4306`), `requireSpecCoverage` (`internal/parser/parser_test.go:1265`),
`internal/glossary/glossary_test.go:587-592`, `internal/diagram/contract_test.go:442-447`,
`internal/formatter/formatter_test.go:877`, and `specEmod` →
`internal/cli/fmt_test.go:579`'s canonical constant. `test.InvariantLibraryLending` (`:314-...`) is
mirrored by `libraryLendingInvariants` (`internal/export/export_test.go:4279`) and consumed by
`internal/formatter/formatter_test.go:808`, `internal/parser/parser_test.go:1000`,
`internal/diagram/contract_test.go:428`, and `invariantEmod` →
`internal/cli/fmt_test.go:560` and `internal/cli/glossary_test.go:146,295`.

---

## Tasks

### Task 1: Give every specced command in the shared spec fixture a rejection path

**Behavior:** `test.SpecLibraryLending` states a rejection scenario for each of the four commands it
specs, not just for two of them, so the fixture the repository holds up as the model of spec practice
demonstrates a failure path per command. Nothing about the tool changes; the fixture and every
expected value that transcribes it move together, and `oracle.Check` still reports nothing about it.

**Why it comes first:** `spec/no-rejection-path` (Task 3) fires twice on this fixture as it stands —
`ReturnCopy` and `ReleaseDesk` each carry specs and no rejection among them. Landing the fixture edit
first keeps the tree green at both commits: this task is a no-op for diagnostics today, and Task 3
finds the fixture already clean instead of having to move it in the same change as the rule.

**Acceptance Criteria:**
- [ ] `test.SpecLibraryLending` states at least one spec whose `then` is a rejection for each of
      `BorrowCopy`, `ReturnCopy`, `ClaimDesk` and `ReleaseDesk`, each naming an invariant already
      declared in that command's own scope, so no invariant is added
- [ ] The fixture keeps both slice homes carrying specs — the aggregate-nested slices under `Loan`
      and the slices declared directly on the `mode dcb` context "Reading Room" — and keeps its
      existing property of writing one spec's `then` above its `when`
- [ ] `oracle.Check` over the fixture returns an empty diagnostic list, so its leaf in "clean input"
      (`internal/oracle/oracle_test.go:60-64`) is unchanged and still passes
- [ ] `test.SpecLibraryLendingSpecNames` transcribes every spec name the fixture now states, in
      declaration order across both homes, and `test.DeclaredSpecNames` over the parsed fixture
      equals it
- [ ] Every expected value elsewhere that restates this fixture's own text moves with it and no
      other: `libraryLendingSpecs` (`internal/export/export_test.go:4306`), the canonical constant
      behind `specEmod` in `internal/cli/fmt_test.go`, and any transcription in
      `internal/glossary`, `internal/diagram` or `internal/formatter` that names the fixture's specs.
      `git diff --stat` names no golden, canonical constant or transcribed list belonging to a
      fixture this task does not edit
- [ ] The new specs give `spec/given-outside-boundary` (Tasks 5 and 6) nothing to report once it
      exists: a `given` written into the aggregate-mode slices names only events the `Loan` aggregate
      declares, and one written into a DCB slice names only events the `when` command's `decides_on`
      lists — or omits `given` entirely
- [ ] `task test:unit` passes, and `internal/parser`, `internal/formatter`, `internal/export`,
      `internal/glossary`, `internal/diagram`, `internal/oracle` and `internal/cli` are green with no
      test skipped or weakened
- [ ] `internal/linter` and `internal/linter/descriptions.go` are untouched — this task adds no rule

**Affected Files/Modules:**
- `internal/test/fixtures.go` — the `SpecLibraryLending` constant and the
  `SpecLibraryLendingSpecNames` transcription
- `internal/export/export_test.go` — the `libraryLendingSpecs` transcription
- `internal/cli/fmt_test.go` — the canonical formatted constant for the spec fixture
- `internal/glossary/glossary_test.go`, `internal/diagram/contract_test.go`,
  `internal/parser/parser_test.go` — the leaves that read the spec-name list back

**Patterns to Follow:**
- The fixture's own existing rejection specs are the shape to copy: `internal/test/fixtures.go:460-464`
  and `:531-535`
- `tasks/learnings.md:481-484` on how to word a change-set criterion for a task that edits a shared
  fixture, and `:196-199` on where `emod fmt` places a spec within its slice, which is what the
  canonical constant must reflect
- `tasks/learnings.md:256-259` on pairing a `Declared…` getter only with a non-empty transcription

**Testable:** Yes — the existing suites are the test, and the fixture's own leaves assert the new
scenarios are read back.

**Verification:** `task test:unit` passes; `oracle.Check` over the fixture is empty; `git diff` shows
movement only in the fixture and the values that transcribe it.

**Depends on:** None.

---

### Task 2: Report a command no spec exercises

**Behavior:** `emod lint` reports `spec/command-without-spec` at info severity, positioned on the
command's name, for every command declared in a model that states at least one spec and that no
spec's `when` names. A model stating no spec anywhere reports nothing, so no file that has not
adopted specs changes behaviour.

**Acceptance Criteria:**
- [ ] A model stating at least one spec and declaring a command no spec's `when` names produces
      exactly one diagnostic for that command, at `diagnostic.Info`, rule name
      `spec/command-without-spec`, positioned at the command's name, with one message text that names
      the command
- [ ] A model that states no spec anywhere produces no diagnostic from this rule, however many
      commands it declares — this is the gate, and a leaf asserts it over a model with several
      commands and no spec
- [ ] A command exercised by a spec declared in a *different* slice of the model produces no
      diagnostic, matching how the validator resolves a spec's `when` model-wide
- [ ] A spec whose `when` is absent, and a spec whose `when` names an event rather than a command,
      exercise no command and leave every command's judgement unchanged
- [ ] The rule reaches commands in both slice homes — nested in an aggregate and declared directly on
      a `mode dcb` context — and a model with several uncovered commands across both homes reports
      them in declaration order, asserted as one comparison over the whole list of formatted lines
- [ ] `linter.RuleDescription` answers for `spec/command-without-spec`, `emod lint --explain
      spec/command-without-spec` prints that non-empty description and returns no error, and an
      unknown rule name still returns an error
- [ ] The hand-maintained rule list in `internal/cli/lint_test.go:627-645` names
      `spec/command-without-spec`, so the "all rules have descriptions" leaf covers it
- [ ] A CLI lint fixture states one spec and leaves one command uncovered, trips this rule and no
      other — asserted with a length of exactly one entry — and its declaring comment names the rule
      it is written to fire
- [ ] `cli.RunLint` and `cli.RunValidate` both return an error for that fixture (an info diagnostic
      is still a diagnostic), the text output names the rule, the command and the line it is declared
      on, and `-f json` reports `"severity": "info"` with exit code 1
- [ ] Every shared fixture in `internal/test/fixtures.go`, every file under `examples/` the
      repository ships as valid, every fixture under `internal/parser/testdata/`, every ` ```emod `
      fence in `README.md` and `docs/dsl-reference.md`, and `billingModel` in
      `internal/wasm/pipeline_test.go` still produce zero diagnostics from `oracle.Check`
- [ ] `internal/parser`, `internal/validator`, `internal/formatter`, `internal/export` and
      `internal/diagram` are untouched: this rule reads the AST and changes no other stage

**Affected Files/Modules:**
- `internal/linter/linter.go` — the model-wide pre-pass collecting every spec's `when`, alongside
  `flowCount` (`:57-62`), and a per-command check reached from `checkSlice`'s command loop
  (`:123-130`)
- `internal/linter/descriptions.go` — the rule description (`:10-31`)
- `internal/linter/linter_test.go` — a top-level group for the rule: the gate, both slice homes,
  cross-slice coverage, severity, position and ordering
- `internal/cli/lint_test.go` — the single-rule fixture, its text and JSON leaves, and the rule-name
  list at `:627-645`
- `internal/oracle/oracle_test.go` — if a sweep shows any clean-input fixture newly reporting, its
  leaf; the measurement in Codebase Context says none should

**Patterns to Follow:**
- `checkLeftChair` (`internal/linter/linter.go:490-495`) with its `flowCount` pre-pass (`:57-62`) is
  the structural template: a per-command rule whose judgement comes from a map built model-wide
  before the slice walk
- `checkOrphanTagKeys` (`internal/linter/linter.go:301-304`) for the "the feature is unused, so say
  nothing" early return, and `:320-335` for sorting findings by declaration position
- `info(pos, rule, msg)` (`internal/linter/linter.go:16-25`) — the whole severity story
- `dcb/single-tag-everywhere` and its suite (`internal/linter/linter_test.go:2226-2806`) for
  asserting an info rule's severity beside its message and position
- `tasks/learnings.md:471-474` on why a lint fixture is never the minimal model: `left-chair`,
  `god-view`, `view-naming`, `clickbait-event`, `orphan-command`, `orphan-event`,
  `automation/missing-todo-list` and the `dcb/*` family are the tripwires, so the fixture needs full
  flows and events with real fields
- `tasks/learnings.md:466-469` on sweeping every checked-in model, and `:6-9` on asserting the
  distinguishing message text in a CLI diagnostic leaf

**Testable:** Yes — through `linter.Lint`, `cli.RunLint`, `cli.RunLintExplain` and `oracle.Check`.

**Verification:** `task test:unit` passes; `go run ./cmd/emod lint` over each model the repository
ships as valid exits 0; `go run ./cmd/emod lint --explain spec/command-without-spec` prints the
description.

**Depends on:** None.

---

### Task 3: Report a command whose specs never reject

**Behavior:** `emod lint` reports `spec/no-rejection-path` at info severity, positioned on the
command's name, for a command some spec exercises where none of those specs states a rejection
outcome — happy-path-only coverage. A command no spec exercises is Task 2's, not this rule's, so the
two never fire on the same command.

**Acceptance Criteria:**
- [ ] A command exercised by one or more specs, none of whose `then` is a rejection, produces exactly
      one diagnostic at `diagnostic.Info`, rule name `spec/no-rejection-path`, positioned at the
      command's name, with one message text naming the command
- [ ] A command with at least one rejection spec among its specs produces no diagnostic, whatever
      other outcomes its other specs state
- [ ] A command no spec exercises produces no diagnostic from this rule — it is Task 2's subject, and
      a leaf asserts the two rules do not both fire on one command
- [ ] Rejection specs count across slices: a rejection spec declared in a different slice from the
      command's declaration silences the rule, matching Task 2's model-wide resolution
- [ ] The rule reaches commands in both slice homes, and several findings across both homes come back
      in declaration order, asserted as one comparison over the whole list of formatted lines
- [ ] `linter.RuleDescription` answers for `spec/no-rejection-path`, `emod lint --explain
      spec/no-rejection-path` prints that non-empty description and returns no error, and the rule
      name joins the list at `internal/cli/lint_test.go:627-645`
- [ ] A CLI lint fixture gives every command a spec so `spec/command-without-spec` stays quiet and
      leaves one command's specs rejection-free, tripping this rule and no other — asserted with a
      length of exactly one entry — with a declaring comment naming the rule it is written to fire
- [ ] `cli.RunLint` and `cli.RunValidate` both return an error for that fixture, the text output
      names the rule, the command and the line it is declared on, and `-f json` reports
      `"severity": "info"` with exit code 1
- [ ] `oracle.Check` over `test.SpecLibraryLending` returns an empty diagnostic list — Task 1's
      rejection specs are what make this true, and the leaf at `internal/oracle/oracle_test.go:60-64`
      is unchanged
- [ ] Every other shared fixture, every file under `examples/` the repository ships as valid, every
      fixture under `internal/parser/testdata/`, every ` ```emod ` fence in `README.md` and
      `docs/dsl-reference.md`, and `billingModel` in `internal/wasm/pipeline_test.go` still produce
      zero diagnostics from `oracle.Check`
- [ ] The rule reads `Spec.Then` only to ask whether it is a rejection, so a `then` variant US-007
      adds is neither counted as a rejection nor treated as an error

**Affected Files/Modules:**
- `internal/linter/linter.go` — the rejection half of Task 2's model-wide pre-pass, and a second
  per-command check in the same command loop
- `internal/linter/descriptions.go` — the rule description
- `internal/linter/linter_test.go` — a top-level group for the rule, including the leaf proving it
  and `spec/command-without-spec` are mutually exclusive
- `internal/cli/lint_test.go` — the single-rule fixture, its leaves, and the rule-name list

**Patterns to Follow:**
- Task 2's pre-pass and command loop are the direct extension point; the two rules share one walk and
  one map rather than opening a second
- `unresolvedRejections` (`internal/validator/validator.go:246-264`) for the `*ast.ThenRejected`
  type assertion over a slice's specs — the same test, asked from the opposite direction
- `tasks/learnings.md:186-189` on `ThenClause` dispatch: an `ok`-guarded assertion for one variant
  stays correct as variants are added, which is why this rule does not switch over the interface
- `tasks/learnings.md:471-474` on a fixture that must keep the other rules — now including Task 2's —
  quiet

**Testable:** Yes.

**Verification:** `task test:unit` passes; `emod lint` over every model the repository ships as valid
exits 0; `emod lint --explain spec/no-rejection-path` prints the description.

**Depends on:** Task 1 (the shared spec fixture must already carry a rejection per command), Task 2
(the pre-pass and the mutual-exclusion leaf).

---

### Task 4: Report an invariant no rejection references

**Behavior:** `emod lint` reports `spec/invariant-never-exercised` at warning severity, positioned on
the invariant's name, for a declared invariant that no `then rejected` spec in that invariant's own
scope names. Two surfaces in the repository state invariants nothing references and move with the
rule: `test.InvariantLibraryLending`, whose oracle leaf moves from "clean input" to a group named for
what it now warns about, and the fenced model in `docs/dsl-reference.md` §4, which gains the
rejections that exercise its invariants — alongside the bullet in that section which claims `emod
lint` never reports one.

**Acceptance Criteria:**
- [ ] An invariant declared on an aggregate that no `then rejected` in that aggregate's own slices
      names produces exactly one diagnostic at `diagnostic.Warning`, rule name
      `spec/invariant-never-exercised`, positioned at the invariant's name, with one message text
      naming the invariant and the scope declaring it
- [ ] An invariant declared directly on a `mode dcb` context behaves the same way against that
      context's own slices
- [ ] Scope is not inherited in either direction: a `rejected` naming an invariant declared on the
      enclosing context does not exercise it from an aggregate's slice, nor does a sibling
      aggregate's rejection exercise it — a model declaring the same identifier in two scopes and
      rejecting it in one reports the other and only the other
- [ ] A `then rejected` whose name matches no invariant declared in that scope — already a validation
      error — does not count as a reference, so the invariant it was meant to name is still reported
- [ ] A model declaring no invariant produces nothing, and a model whose every invariant is exercised
      produces nothing; the rule is not gated on the model stating a spec
- [ ] Findings from a context's own invariants and its aggregates' come back in declaration order,
      asserted as one comparison over the whole list of formatted lines — the AST holds the two in
      separate collections, so an unsorted walk reports them in field order rather than source order
- [ ] `linter.RuleDescription` answers for `spec/invariant-never-exercised`, `emod lint --explain
      spec/invariant-never-exercised` prints that non-empty description and returns no error, and the
      rule name joins the list at `internal/cli/lint_test.go:627-645`
- [ ] A CLI lint fixture declares one unexercised invariant and gives every command a spec and a
      rejection so Tasks 2 and 3 stay quiet, tripping this rule and no other — asserted with a length
      of exactly one entry — with a declaring comment naming the rule it is written to fire
- [ ] `cli.RunLint` and `cli.RunValidate` both return an error for that fixture, the text output
      names the rule, the invariant and the line it is declared on, and `-f json` reports
      `"severity": "warning"` with exit code 1
- [ ] `test.InvariantLibraryLending` leaves `oracle.Check`'s "clean input" group for a group named
      after the shape it now demonstrates, whose leaf asserts the whole formatted line for each of
      the five invariants it declares, in declaration order across both slice homes; the fixture's
      own text is unchanged, since its purpose is to declare invariants without referencing them
- [ ] The fenced model at `docs/dsl-reference.md:175` states rejections exercising all four of its
      invariants, and the "documented models" leaf (`internal/oracle/oracle_test.go:112-129`) reports
      nothing for every one of the seven fences
- [ ] The §4 bullet reading "An invariant nothing references is not an error: no construct refers to
      an invariant, and neither `emod validate` nor `emod lint` reports one for going unmentioned"
      (`docs/dsl-reference.md:247`) states what is now true — that it is not a validation error and
      that `emod lint` reports `spec/invariant-never-exercised` for one no rejection names
- [ ] The `### invariant` and `### spec` heading texts in `docs/dsl-reference.md` are unchanged and
      no numbered section is added, moved or renumbered, so the fourteen sub-heading links and the
      four number-prefixed links elsewhere in the document still resolve
- [ ] Every other shared fixture, every file under `examples/` the repository ships as valid, every
      fixture under `internal/parser/testdata/`, and `billingModel` in
      `internal/wasm/pipeline_test.go` still produce zero diagnostics from `oracle.Check`
- [ ] The set of things that count as a reference is gathered in one place the rule then reads, so
      US-009 adds rejection flow edges to that set without touching the comparison, the message or
      the scope walk — a reviewer can point at the single collector and say where the next
      contributor goes

**US-009 seam, stated:** US-009's criterion "a rejection edge counts as a reference for
`spec/invariant-never-exercised`" is additive against this task if and only if the rule keeps
"which invariant names does this scope reference?" separate from "which of this scope's invariants
are unreferenced?". This task writes the first as a collector over the scope's slices — today reading
each spec's `then` for a rejection — and the second as the comparison against the scope's declared
invariants. US-009 then appends its `flow` rejection entries to the collector's result and changes
nothing else. Inlining the `*ast.ThenRejected` assertion at the comparison site would force US-009 to
restructure the rule instead of extending it.

**Affected Files/Modules:**
- `internal/linter/linter.go` — the scope walk over `model.Contexts` and `ctx.Aggregates`, the
  reference collector, and the comparison
- `internal/linter/descriptions.go` — the rule description
- `internal/linter/linter_test.go` — a top-level group for the rule: both scopes, the non-inheritance
  cases, the unresolved-name case, ordering, severity and position
- `internal/cli/lint_test.go` — the single-rule fixture, its leaves, and the rule-name list
- `internal/oracle/oracle_test.go` — `test.InvariantLibraryLending`'s leaf moves out of "clean input"
  into a group named for the rule
- `docs/dsl-reference.md` — §4's fenced model and the bullet at `:247`

**Patterns to Follow:**
- `invariantScope`, `invariantScopes` and `unresolvedRejections`
  (`internal/validator/validator.go:203-264`) for the scope rule and the walk this rule inverts,
  including the comment at `:216-219` explaining why an aggregate and its context are separate scopes
- `checkDCBInAggregateMode` (`internal/linter/linter.go:163-197`) for a linter check that walks a
  context's aggregates itself
- `checkOrphanTagKeys` (`internal/linter/linter.go:320-335`) for sorting findings gathered from more
  than one collection, and the comment at `:320-323` on why
- The "automations reading no view" group (`internal/oracle/oracle_test.go:131-157`) is the exact
  precedent for a fixture that legitimately warns: the leaf asserts whole formatted lines through
  `reportedLines` (`:402`)
- `tasks/learnings.md:136-139` on a message naming both a symbol and its scope, where a second
  `require.Contains` is shadowed by the first — assert the whole formatted line instead
- `tasks/learnings.md:526-529` on an ```emod fence being a promise the block validates, and
  `:541-544` on `docs/dsl-reference.md` heading anchors

**Testable:** Yes.

**Verification:** `task test:unit` passes; `emod lint` over every model the repository ships as valid
exits 0; `emod lint --explain spec/invariant-never-exercised` prints the description; the fence at
`docs/dsl-reference.md:175`, extracted and run through `oracle.Check`, reports nothing.

**Depends on:** Task 3 (its lint fixture must keep `spec/no-rejection-path` quiet, which needs that
rule to exist to be checkable).

---

### Task 5: Report a `given` event outside the aggregate boundary

**Behavior:** `emod lint` reports `spec/given-outside-boundary` at warning severity, positioned on
the offending event name inside the `given` list, when a spec in a slice nested in an aggregate names
a `given` event that aggregate does not declare — history the consistency boundary cannot see. A
spec in a slice declared directly on a context is the DCB arm and is Task 6's.

**Acceptance Criteria:**
- [ ] A spec in an aggregate-nested slice whose `given` names an event declared by another
      aggregate's slice produces exactly one diagnostic at `diagnostic.Warning`, rule name
      `spec/given-outside-boundary`, positioned at that event name inside the `given` list, with a
      message naming the event, the aggregate the spec belongs to and the aggregate that declares the
      event
- [ ] A `given` event declared by a slice in *another context* is reported the same way
- [ ] A `given` event declared by a sibling slice of the same aggregate produces no diagnostic — the
      boundary is the aggregate, not the slice, and a leaf asserts this with the event declared one
      slice away from the spec that names it
- [ ] An event declared inside a `translation` counts as declared by the slice holding that
      translation, so a spec naming it from a sibling slice of the same aggregate produces no
      diagnostic
- [ ] A `given` name no slice in the model declares produces no diagnostic from this rule — the
      validator already reports it, and a leaf asserts the rule does not double-report
- [ ] A spec with `given []`, and a spec omitting `given` entirely, produce no diagnostic
- [ ] Every offending name in one `given` list is reported, and several findings across a model come
      back in declaration order, asserted as one comparison over the whole list of formatted lines
- [ ] A spec in a slice declared directly on a context produces no diagnostic from this task,
      whatever its `given` names — Task 6 owns that arm, and a leaf asserts this task leaves it alone
- [ ] `linter.RuleDescription` answers for `spec/given-outside-boundary`, `emod lint --explain
      spec/given-outside-boundary` prints that non-empty description and returns no error, and the
      rule name joins the list at `internal/cli/lint_test.go:627-645`. The description covers both
      arms, since Task 6 shares the rule name
- [ ] A CLI lint fixture declares two aggregates, gives every command a spec and a rejection and
      exercises every invariant it declares so Tasks 2, 3 and 4 stay quiet, and states one `given`
      naming the other aggregate's event — tripping this rule and no other, asserted with a length of
      exactly one entry, with a declaring comment naming the rule it is written to fire
- [ ] `cli.RunLint` and `cli.RunValidate` both return an error for that fixture, the text output
      names the rule, the event and the line the `given` is written on, and `-f json` reports
      `"severity": "warning"` with exit code 1
- [ ] `oracle.Check` over `test.SpecLibraryLending` returns an empty diagnostic list: its four
      aggregate-mode `given` events all name events of their own aggregate
- [ ] Every other shared fixture, every file under `examples/` the repository ships as valid, every
      fixture under `internal/parser/testdata/`, every ` ```emod ` fence in `README.md` and
      `docs/dsl-reference.md`, and `billingModel` in `internal/wasm/pipeline_test.go` still produce
      zero diagnostics from `oracle.Check`
- [ ] The rule reads no `decides_on` and no predicate, and no example payload: `DecidesOnClause` is
      untouched by this task

**Affected Files/Modules:**
- `internal/linter/linter.go` — a model-wide index from event name to the slice home declaring it,
  built with the pre-passes at the top of `Lint`, and a per-spec check reached from the slice walk
  where `ast.SliceRef` still carries the aggregate
- `internal/linter/descriptions.go` — the rule description, worded to cover both arms
- `internal/linter/linter_test.go` — a top-level group for the rule's aggregate arm
- `internal/cli/lint_test.go` — the two-aggregate fixture, its leaves, and the rule-name list

**Patterns to Follow:**
- `Lint`'s existing model-wide pre-pass and its `ctx.SliceRefs()` loop
  (`internal/linter/linter.go:57-62`, `:93-99`) — the aggregate each slice belongs to is already in
  hand there; `checkSlice` receives only the aggregate *name* (`:107`), so decide deliberately
  whether the check hangs off the ref loop or takes a widened parameter
- `ast.SliceRef` and `Context.SliceRefs()` (`internal/ast/traverse.go:19-50`) for reading both slice
  homes, and `tasks/learnings.md:171-174` on why a walk that sees only one home is the recurring
  defect here
- `checkSlice`'s two event loops (`internal/linter/linter.go:109-122`) for the reminder that a
  slice's events are its own plus each translation's nested one
- `checkOrphanTagKeys` (`internal/linter/linter.go:320-335`) for position-sorted findings
- `warning(pos, rule, msg)` (`internal/linter/linter.go:27-36`)
- `tasks/learnings.md:476-479` on a rule whose message branches on model state: this rule has two
  texts once Task 6 lands, so every leaf compares complete formatted diagnostics rather than
  fragments

**Testable:** Yes.

**Verification:** `task test:unit` passes; `emod lint` over every model the repository ships as valid
exits 0; `emod lint --explain spec/given-outside-boundary` prints the description.

**Depends on:** Task 4 (its lint fixture must keep `spec/invariant-never-exercised` quiet, which
needs that rule to exist to be checkable).

---

### Task 6: Report a `given` event the command's `decides_on` does not match

**Behavior:** `emod lint` reports `spec/given-outside-boundary` at warning severity, positioned on
the offending event name, when a spec in a slice declared directly on a context names a `given` event
whose type the `when` command's `decides_on` does not list. The check is type-level: it compares
event names against the `decides_on` events list and reads no predicate and no field value.

**Acceptance Criteria:**
- [ ] A spec in a context-level slice whose `when` names a command declaring `decides_on`, and whose
      `given` names an event that list does not carry, produces exactly one diagnostic at
      `diagnostic.Warning`, rule name `spec/given-outside-boundary`, positioned at that event name,
      with a message naming the event and the command whose `decides_on` does not reach it
- [ ] The two arms of the rule carry different texts, and every leaf asserting either compares the
      whole formatted diagnostic line, so a leaf cannot pass against the other arm's wording
- [ ] A `given` event the `when` command's `decides_on` does list produces no diagnostic
- [ ] A `when` command declaring no `decides_on` puts nothing outside the boundary: every `given` in
      such a spec produces no diagnostic. A leaf states this over the shape
      `test.SpecLibraryLending` carries — a context-level command with no `decides_on` and a spec
      naming a `given`
- [ ] A spec whose `when` is absent, and a spec whose `when` names an event rather than a command,
      produce no diagnostic
- [ ] A `given` name no slice in the model declares produces no diagnostic — the validator already
      reports it
- [ ] The `where` predicate is never read: two models identical but for their `decides_on` predicates
      produce identical diagnostics from this rule, and `DecidesOnClause.Predicate` appears nowhere
      in the change
- [ ] Every offending name in one `given` list is reported, and several findings come back in
      declaration order, asserted as one comparison over the whole list of formatted lines
- [ ] A spec in an aggregate-nested slice still takes Task 5's arm and its message, whatever the
      enclosing context's mode, and a leaf asserts a model with one offending spec in each home
      reports one diagnostic per arm with the arm's own text
- [ ] The rule description registered in Task 5 covers this arm too, `emod lint --explain
      spec/given-outside-boundary` still returns no error, and no second rule name is introduced
- [ ] A CLI lint fixture declares a `mode dcb` context whose commands carry `decides_on`, gives every
      command a spec and a rejection and exercises every invariant so Tasks 2, 3 and 4 stay quiet,
      and states one `given` the `when` command's `decides_on` does not list — tripping this rule and
      no other, asserted with a length of exactly one entry, and keeping the `dcb/*` family quiet by
      tagging every event and referencing every tag key
- [ ] `cli.RunLint` and `cli.RunValidate` both return an error for that fixture, the text output
      names the rule, the event and the line the `given` is written on, and `-f json` reports
      `"severity": "warning"` with exit code 1
- [ ] `oracle.Check` over `test.SpecLibraryLending` returns an empty diagnostic list: its
      `ReleaseDesk` spec names a `given` that command's `decides_on` lists, and its `ClaimDesk` spec
      is silent under the no-`decides_on` rule
- [ ] Every other shared fixture, every file under `examples/` the repository ships as valid — including
      `examples/dcb_model.emod` — every fixture under `internal/parser/testdata/`, every ` ```emod `
      fence in `README.md` and `docs/dsl-reference.md`, and `billingModel` in
      `internal/wasm/pipeline_test.go` still produce zero diagnostics from `oracle.Check`

**Affected Files/Modules:**
- `internal/linter/linter.go` — the DCB arm of Task 5's check, plus a model-wide index from command
  name to its declaration so a `when` can be resolved to the command carrying the `decides_on`
- `internal/linter/descriptions.go` — the description text, if Task 5's wording needs widening
- `internal/linter/linter_test.go` — the DCB arm's leaves inside Task 5's group, including the
  both-arms-in-one-model leaf
- `internal/cli/lint_test.go` — the DCB fixture, its leaves

**Patterns to Follow:**
- `checkQueryTooBroad` (`internal/linter/linter.go:367-391`) for reading a command's `decides_on` and
  skipping a command that declares none
- `DecidesOnClause` (`internal/ast/ast.go:246-253`) — `Events []string` is the whole of what this
  task reads; `Predicate` is US-011's
- `test.SpecLibraryLending`'s "Release Desk" slice (`internal/test/fixtures.go:540-...`) is the
  in-boundary shape — its command's `decides_on` names the event its spec's `given` states — and its
  "Claim Desk" slice (`:507-...`) the no-`decides_on` shape
- `tasks/learnings.md:476-479` on pinning a two-text rule by whole formatted lines
- `tasks/learnings.md:151-154` on `mode dcb` shapes being the usual tripwire in a fixture — a DCB
  context needs tags on its events and a `decides_on` reaching them, or `dcb/untagged-event` and
  `dcb/orphan-tag-key` fire

**Testable:** Yes.

**Verification:** `task test:unit` passes; `emod lint` over every model the repository ships as valid
exits 0, `examples/dcb_model.emod` included; `emod lint --explain spec/given-outside-boundary` prints
the description.

**Depends on:** Task 5 (the rule name, its description entry, its group in the linter suite and the
arm-selection rule all land there).

---

## Summary

**Six tasks.**

**Ordering rationale — fixture-first, then cheapest rule to hardest, with each rule's fixture
depending on the rules before it.** Task 1 moves the one shared fixture a later rule would otherwise
push out of `oracle.Check`'s clean-input group, and is a no-op for diagnostics at the commit it lands
in — the same fixture-before-rule sequencing the predecessor US-008 used for
`automation/missing-todo-list` (`tasks/learnings.md:466-469`). Tasks 2 and 3 come next because they
share one model-wide pre-pass and are mutually exclusive per command, so writing them adjacent is
what lets a leaf assert that exclusivity. Task 4 follows because it is the only rule that forces
checked-in surfaces to move — one fixture's oracle leaf and one section of the DSL reference — and
because it carries the US-009 seam that must be settled before US-009 is decomposed. Tasks 5 and 6
close with the rule that pays for the feature, split by arm: after Task 5 the rule exists and reports
in aggregate mode, after Task 6 it reports in both, and each commit leaves the tree green. The
dependency chain is also a fixture chain: every lint fixture from Task 3 onward must keep the earlier
rules quiet, which is only checkable once those rules exist.

**Story acceptance criteria, all six covered, none deferred:**

| Story criterion | Task |
|---|---|
| `spec/command-without-spec` (info) fires for a command no spec exercises | 2 |
| `spec/no-rejection-path` (info) fires for a command with specs but no rejection spec | 1, 3 |
| `spec/invariant-never-exercised` (warning) fires for a declared invariant no `rejected` spec references | 4 |
| `spec/given-outside-boundary` (warning) fires in aggregate mode for a `given` event of another aggregate | 5 |
| `spec/given-outside-boundary` (warning) fires in DCB mode for a `given` event the `when` command's `decides_on` does not match | 6 |
| All four rules respect the existing severity configuration and `emod lint --explain <rule>` | 2, 3, 4, 5 (6 shares 5's rule name) |

**Two things a reader of this list should carry forward.** First, `spec/command-without-spec` is
gated on the model stating at least one spec, and that gate is the difference between 0 and 47
diagnostics across models the repository already ships as valid — the story's wording does not name
it, and Open question 1 is where the decision and its measurement live. Second, US-009 depends on
Task 4's seam: the set of things that reference an invariant is gathered in one collector so that
rejection flow edges join it additively, and US-009's decomposition should point at that collector
rather than re-deriving the rule.

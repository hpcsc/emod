# US-018: Learn the new constructs from examples and the reference

## Progress
- [x] Task 1: Pin the version, describe the constructs, bind wire types and delay an automation in the flagship example
- [x] Task 2: State the invariants, specs and rejection path in the flagship example
- [x] Task 3: Add `examples/specs_hotel.emod` as a model that validates
- [x] Task 4: Document the elapsed-time automation and close the reference's coverage of the batch

---

## Story Reference

`user-stories/specs-and-metadata.md` → **US-018: Learn the new constructs from examples and the
reference** (eighteenth and last story of "Specs, Invariants, and Model Metadata", lines 226-234).
Design notes: `docs/proposals/specs-and-metadata-proposal.md` — the Worked Example (`:432-583`),
which is the model `examples/specs_hotel.emod` mirrors, and Phase 5 (`:605-609`), which lists this
story's three surfaces from the proposal's side: "Update `examples/all_patterns.emod`; add
`examples/specs_hotel.emod`. Extend `docs/dsl-reference.md`."

### This story cannot start until US-001 through US-013 have landed

US-001 through US-006 are delivered on main. **US-007 through US-013 are decomposed and not
implemented**, and every one of them contributes a construct or a rule this story's first criterion
names. The story is the integration point: it is the first time every construct in the batch appears
together in one model, so it is also the first time their interactions are exercised.

Which criteria could be verified earlier, and which cannot:

| Work | Earliest verifiable |
|---|---|
| The version header, the descriptions, the block forms of `model` and `actor` in the flagship example | **Today, from main** — US-001 and US-002 are delivered |
| Wire types on events (Task 1) | US-012 |
| A delay on an automation (Task 1) | US-013 |
| Invariants and command specs with both US-006 outcomes | US-006, delivered — but see the finding below: adding them *without* US-007 to US-011 in the tree produces a file that has to be reopened, because the gate that forces the rest of the edit does not exist yet |
| The four spec shapes, payloads, the rejection flow edge, and the whole of Task 2 as a green end state | **US-011 at the earliest, US-013 for the file as a whole** |
| `examples/specs_hotel.emod` (Task 3) | US-013 |
| The `after` documentation and the reference coherence pass (Task 4) | US-013, and after every upstream documentation task has landed |

Task 1 is therefore the only task with a partial early start, and even that one wants US-012 and
US-013 before it can close. Nothing here should be attempted against a tree missing any of them.

### In scope

The three teaching surfaces the story names, and nothing else:

- `examples/all_patterns.emod` — the flagship example, the one the reference sends a reader to
  (`docs/dsl-reference.md:86`). It gains every construct this batch adds and keeps returning no error
  from `cli.RunValidate`.
- `examples/specs_hotel.emod` — a new example mirroring the proposal's Worked Example: invariants,
  specs with payloads, a rejection flow edge, a wire type and a timer, in the hotel domain the
  proposal writes it in.
- `docs/dsl-reference.md` — **only what the upstream stories do not already own.** That is the
  elapsed-time automation, which US-013 explicitly defers here, plus the coherence pass over the
  finished document: the sentences the batch falsifies that nobody claimed, and the reconciliation of
  both anchor families once five stories have edited the file.

Carried along, because the story's first criterion is otherwise unfalsifiable: a leaf that reads the
flagship example back and requires it to *state* each construct, not merely to validate. Nothing in
the tree asserts what an example demonstrates — `examplePaths` (`internal/cli/validate_test.go:773`)
asserts only that it validates, so an edit that silently dropped every spec from the file would leave
the suite green while the story's claim became false.

### Out of scope, named explicitly

- **US-014 (formatter), US-015 (LSP), US-016 (diagram spec cards), US-017 (highlighting).** None is a
  dependency and none needs example coverage here. US-014 owns canonical ordering and alignment for
  the new constructs, and the examples are hand-formatted and deliberately not run through `emod fmt`
  — verified: `emod fmt --check examples/all_patterns.emod` fails today, before any edit, because
  formatting would insert a header, re-align every field column and move every flow past its
  translation. US-015 and US-017 are editor surfaces with no `.emod` artefact. US-016's `--specs`
  flag renders what a slice already declares, so the specs Task 2 writes are its input, not its
  obligation. If any of the four is later found to need an example, it is a change to that story.
- **Every code surface.** No file under `internal/`, `cmd/`, `editors/`, `e2e/` or `e2e-viewer/`
  changes except the one test file carrying the coverage leaf.
- **`README.md`.** The story names the reference only, US-009 Task 11 already states the README is
  outside its change set, and the README's own quick-start was brought current by
  `tasks/completed/us-011-learn-the-realignment-from-examples-and-the-reference.md`. Its
  `emod diagram <file> -f <format>` block (`README.md:159-163`) documents four invocations that
  silently produce drawio, and its `emod export <file> -f <format>` block (`:168-171`) two more that
  silently produce JSON; both are the urfave/cli defect recorded in `tasks/learnings.md:111-114` and
  belong to a CLI story, not this one.
- **`internal/parser/testdata/all_patterns.emod`.** A parser fixture that already differs from the
  example (verified) and is guarded separately by `internal/cli/validate_test.go:36-48`. Nothing
  asserts the two copies match, and this story does not make them match.
- **`examples/error_diagnostics_test.emod`.** Authored to fail; `examplePaths` already asserts the
  diagnostics it demonstrates. It gains nothing.
- **`examples/dcb_model.emod`.** See decision 4.
- **Lint rules in the reference.** No section of `docs/dsl-reference.md` or `README.md` lists lint
  rules, `emod lint --explain` is where a rule describes itself, and the precedent is
  `tasks/completed/us-008-flag-automations-with-no-todo-list.md`, a pure-lint story with no
  documentation task. So `spec/command-without-spec`, `spec/no-rejection-path`,
  `spec/invariant-never-exercised`, `spec/given-outside-boundary`, `flow/rejection-without-spec` and
  the whole of US-011 are documented nowhere and nothing is owed. `wire/type-format` is the one
  exception, and US-012 Task 7 already carries it.

### What is genuinely left, stated plainly

Nearly every story in this batch carries its own documentation task, and restating "document each new
construct" would duplicate owned work and put two stories on one file. The ownership, verified by
reading each task file:

| Construct the batch adds | Where the reference teaches it | Owner |
|---|---|---|
| `emod <n>` version header | §2 Version Header | US-001, delivered |
| `description` on every construct; block forms of `model` and `actor` | §10 Descriptions, §3 | US-002, delivered |
| A keyword in field-name position | §8 `### Keywords as Field Names` | US-003, delivered |
| `invariant <Name> "<statement>"`, on an aggregate and on a `mode dcb` context | §4 `### invariant` | US-005 delivered; its fenced model at `:175` and its bullet at `:247` rewritten by **US-008 Task 4** |
| `spec` with `given`, `when`, `then [events]`, `then rejected` | §6 `### spec` | US-006, delivered |
| `then view` and `then command`; which shape each slice pattern may state | §6 `### spec`, §11 table and bullets | **US-007 Task 8** |
| `command -> rejected:` as a `flow` entry | §7 Flows, §11 table row and bullet | **US-009 Task 11** |
| Payload literals on a spec's element references | §6 `### spec`, §8 field-type bullet, §11 | **US-010 Task 8** |
| `type "<wire type>"` on an event | a new `###` sub-heading, precedent `### External Source Events` | **US-012 Task 7** |
| `on <Event> after "<duration>"` | §6 `### Automation Pattern` | **nobody — US-018 Task 4** |

The story's third criterion is therefore *not* met by verification alone, and it is not met by a
"read it again" task either. Two things are genuinely missing and one thing is genuinely
unreconciled:

1. **`after` is documented by no story.** `tasks/us-013-fire-automations-after-elapsed-time.md:57-59`
   reads: "Examples and reference coverage (US-018): no `.emod` file under `examples/` or
   `internal/parser/testdata/` gains a delay, and `docs/dsl-reference.md` keeps its current Automation
   Pattern section." Its Summary (`:1219-1221`) repeats it. US-013 has ten tasks and none of them
   touches the reference. `after` is a new construct, the story's criterion says each new construct is
   documented with at least one example, and §6 `### Automation Pattern` (`:324-351`) today lists
   `on`, `every`, `reads`, `command` and `target context` and no delay. This alone earns a task.
2. **Two sentences the batch falsifies belong to nobody.** §1 `### Strings` (`:33`) enumerates which
   quoted values carry a human-readable name, which carry prose, and which carry a machine-read value
   — a spec's name, an event's wire type and an `after` duration are three new quoted values it does
   not account for. §5's slice skeleton (`:265`) annotates `flow { ... }` as "0+ command→event
   wiring", which US-009's rejection entry falsifies; US-009 Task 11's change set is §7 and §11 only.
3. **The final anchor reconciliation has no owner.** Each upstream documentation task reconciles the
   `^## [0-9]+\.` list against `\(#[0-9]+-` and the `^### ` list against `\(#[a-z]` *against its own
   end state*, and none of them knows whether it lands first or last. US-012 Task 7 is the only one
   permitted to add a `###` heading and its text is unspecified, so the sub-heading link family cannot
   be settled until it exists. One reconciliation over the finished document is what makes the
   twelve-plus number-prefixed and fourteen-plus sub-heading links a checked claim rather than an
   inherited one.

### Findings

Five things this decomposition surfaced that the story does not say and a reviewer should know.

**F1 — Adopting specs in the flagship example is all-or-nothing, and it costs more than the story
suggests.** `spec/command-without-spec` is gated: US-008's Open question 1 decides it "fires only in a
model that states at least one spec", precisely so that models which have not adopted specs are told
nothing. A lint diagnostic of any severity fails `emod validate` (`tasks/learnings.md:466-469`). So
the moment `examples/all_patterns.emod` states its first spec, the gate opens for the whole file and
**every one of its five commands** — `ReserveRoom`, `CheckOutGuest`, `SendConfirmationEmail`,
`ImportExternalReservation`, `ExpireReservation` — needs a spec whose `when` names it, or the example
stops validating. `spec/no-rejection-path` then fires on each of those commands unless at least one of
its specs states a rejection, and each rejection must name an invariant declared in that command's own
scope, and `spec/invariant-never-exercised` requires every invariant so declared to be exercised. The
file's spec adoption is one indivisible edit, which is why Task 2 is one task and not five.

The sharp end: US-008 justified the gate partly by writing that `examples/all_patterns.emod`'s "job is
to show the patterns rather than their scenarios"
(`tasks/us-008-lint-spec-coverage-and-boundary-assumptions.md:86`), and concluded "no checked-in model
moves for this rule" (`:88`). That conclusion is true when US-008 lands and false the day US-018 does.
This is not a conflict in the gate's design — the gate is what keeps the *other* eleven checked-in
models silent — but US-008's measured blast radius does not survive this story, and the cost lands
here in full. Nothing in US-008 needs to change.

**F2 — The proposal's Worked Example does not pass `emod validate` as written.** Transcribed
verbatim into `examples/specs_hotel.emod` it fails at least three ways, all of them cheap to fix and
none of them visible from reading the proposal:
- Its view `UnreleasedHolds` (`docs/proposals/specs-and-metadata-proposal.md:538`) does not end in
  `View`, which `view-naming` reports today. The automation reading it, the reference to it, and its
  slice all move with the rename.
- It states specs, so F1's gate opens: `HoldRoom` and `ReleaseHold` have no spec whose `when` names
  them. `then command ReleaseHold` does not count — US-008's Open question 2 decides that a spec
  exercises a command only through its `when`.
- Once those two commands gain specs, `spec/no-rejection-path` requires a rejection among each one's
  specs, so the example needs more than the single `roomNotDoubleBooked` invariant it declares.

**F3 — Nothing else in the batch conflicts.** The cross-construct question the story exists to answer
— whether one model can state a wire type, a timer, a rejection edge, invariants and payloads at once
— resolves cleanly against the stories' own decisions, and the checks are stated per task below.
US-007's Open question 1 gives the decisive table: a `then rejected` outcome requires its slice to
declare a `command`, `then view` requires a `view`, `then command` requires an `automation`, and a
`then` event list requires a `command` or a `translation`. Every slice of `examples/all_patterns.emod`
that must state an outcome already declares the construct that outcome requires — slice 4 declares
both a view and an automation, slice 5 both a command and a translation, slice 6 both a command and an
automation — so no slice has to be restructured to carry its spec.

**F4 — Aggregate "Notification" declares no event, which bounds what its spec may say.**
`spec/given-outside-boundary`'s aggregate arm (US-008 Open question 5) takes the boundary from the
aggregate the slice is nested in. `SendConfirmationEmail` lives in aggregate "Notification", which
declares no event of its own, so a spec written there states no `given` — or the slice gains an event
first. This is the one place in the flagship example where the boundary rule has teeth.

**F5 — US-011 is invisible in every teaching surface, and nothing is owed.** Value-aware boundary
checking adds no construct; it sharpens a lint rule, and the reference documents no lint rules.
`docs/dsl-reference.md:175`'s fenced model — a `mode dcb` context declaring invariants, with a
`decides_on` command — is rewritten by US-008 Task 4 to state rejections exercising them, so the DCB
half of the batch does get a machine-checked demonstration, in the reference rather than in
`examples/`. That is why decision 4 leaves `examples/dcb_model.emod` alone.

### Open questions, decided

Six shapes the story does not pin down.

1. **"Every new construct" means every construct, once, in whichever scope the flagship example can
   state it.** `examples/all_patterns.emod` is an aggregate-mode model and stays one. `invariant` on a
   `mode dcb` context is the same construct in a different scope, not a second construct, and forcing
   a DCB context into the flagship example would make a file whose own header comment reads "exercises
   all four slice patterns" into a two-mode file and roughly double it. The DCB scope is shown,
   validated and guarded in `docs/dsl-reference.md:175`'s fence, which `internal/oracle`'s "documented
   models" leaf runs through the whole pipeline.
2. **The construct list is enumerated, so the criterion is checkable.** Task 1 and Task 2 each carry
   the constructs they add as named criteria and a leaf that reads them back from the parsed example.
   Presence only — at least one of each — never a count or a name, so the example stays free to grow.
3. **The flagship example's edit splits at the lint gate, not by construct family.** The version
   header, the descriptions, the wire types and the delay are each independently valid and open no
   gate: each can land, validate and be committed on its own. The invariants, specs, payloads and
   rejection edge cannot — F1 makes them one atomic change. Two tasks, split exactly there.
4. **`examples/dcb_model.emod` is not edited.** No story criterion names it; adding one spec would
   open F1's gate for its four commands, requiring four invariants on the context, four rejection
   specs and a `given` for each that its command's `decides_on` matches — a substantial edit no
   criterion asks for. US-008 Task 6 and US-011 both already carry criteria requiring it to keep
   returning zero diagnostics, which is the guarantee that matters.
5. **Both examples are `emod fmt` canonical.** This overturns the decomposition's own answer, which
   was that neither example would be formatted because `emod fmt --check examples/all_patterns.emod`
   already failed and formatting it "would insert the header, re-align every field column, hoist
   invariants, move every spec to the end of its slice and drop the file's blank lines, producing a
   diff that buries the story's change". Measured against the tree rather than estimated, that is
   wrong: the file's whole distance from canonical form was three lines, all of them `emod fmt`
   hoisting a view's `subscribes` above its `fields`. No header is inserted (one is already there),
   no field column moves and no flow is reordered. Hand-written spec, flow and payload blocks add
   roughly ninety more lines of distance, so leaving the examples unformatted would have shipped a
   flagship the project's own formatter immediately rewrites — in the one story whose purpose is that
   a reader can learn the language from these files. `emod fmt <file>` writes in place
   (`tasks/learnings.md:336-339`), so it is run deliberately rather than as a verification step.
6. **The `after` documentation and the coherence pass are one task, not two.** `after` is a genuine
   gap and a story criterion, so a reference task exists on its own merits. Split off, the coherence
   half would be a task whose only content is "re-read the document", whose criteria would duplicate
   the anchor reconciliation every upstream task already carries, and which could not be judged
   independently. Kept together, the task has one behaviour — the reference is complete and coherent
   over the batch — and the coherence criteria are the closing half of it.

### Learnings folded in

From `tasks/learnings.md`, each load-bearing here rather than decorative:

- *`examples/` is enumerated by its guard, and `_test.emod` means authored to fail* (`:531-534`).
  `examplePaths` (`internal/cli/validate_test.go:773-791`) reads the directory and splits on the
  suffix, so `examples/specs_hotel.emod` is guarded the moment it exists with no list to edit — the
  answer to "build on the test US-011 landed" is that there is nothing to build. What it does *not*
  cover is what an example demonstrates, which is the gap Tasks 1 and 2 close.
- *A lint warning fails `emod validate`, so a new rule sweeps every checked-in model before it lands*
  (`:466-469`). This is the whole of F1, and it is why "passes `emod validate`" in both story criteria
  means lint-clean under all nineteen-odd rules, not merely parseable.
- *A lint fixture trips exactly one rule, so it is never the minimal model* (`:471-474`). The tripwires
  the examples must keep quiet while gaining specs: `view-naming`, `god-view`, `clickbait-event`,
  `left-chair`, `command-past-tense`, `orphan-command`, `orphan-event`,
  `automation/missing-todo-list`, and the `dcb/*` family.
- *A spec is not a reference: a command only a spec exercises is still orphaned* (`:191-194`). Adding
  specs to an example removes no flow and creates no reference; every command still needs its flow,
  automation or translation.
- *An ```emod fence is a promise that the block validates* (`:526-529`). Any fenced block Task 4 adds
  is a whole model `oracle.Check` accepts, or it carries a plain fence.
- *`docs/dsl-reference.md` anchors embed the section number* (`:36-39`) and *sub-heading anchors are
  cited more often than the numbered ones* (`:541-544`). Task 4 adds no numbered section, renames no
  heading, and closes with the reconciliation over both families.
- *`docs/dsl-reference.md` section 13 is machine-read* (`:411-414`). `dslReferencePalette`
  (`internal/diagram/contract_test.go:1349`) locates §13 by its `Diagram Palette` heading and parses
  six rows. Task 4 does not touch it, and `TestExporterPaletteMatchesReference` is the receipt.
- *urfave/cli v2 discards every flag written after the file argument* (`:111-114`). Every invocation
  the reference writes today puts its flags first or takes no positional argument at all; Task 4 keeps
  that shape and adds no invocation in the broken one.
- *Acceptance criteria never reference commit or branch state* (`:21-24`, `:246-249`), and *a "no
  expected constant moves" criterion is unsatisfiable when the task edits a shared fixture*
  (`:481-484`). No task here edits a shared fixture, so the change-set criteria can be stated
  strictly; they are stated against the working tree throughout.
- *An assertion whose expected value comes from the code under test cannot fail* (`:126-129`). The
  coverage leaves read the example from disk and assert against what the story requires, never against
  a re-derivation.
- *Never write emod source with `%q`* (`:46-49`) — binding the moment a leaf writes an extracted block
  or an example to a temporary file.
- *A tested, defensible improvement found on the way is still a separate commit* (`:461-464`). The
  repo's known defects — `emod export <file> --format cue` silently emitting JSON, the fourteen stale
  `internal/export/export.go` references in `tasks/learnings.md` — are not fixed by any task here.

---

## Codebase Context

**The examples directory.** Three files today.

- `examples/all_patterns.emod` (178 lines) is the flagship: model "Hotel Reservation", one actor, two
  contexts, seven slices under aggregate "Reservation" in context "Reservations" and one under
  aggregate "Notification" in context "Notifications". Five commands — `ReserveRoom` (`:16`),
  `CheckOutGuest` (`:56`), `ExpireReservation` (`:132`), `ImportExternalReservation` (`:96`) and
  `SendConfirmationEmail` (`:169`). Four declared events plus the translation's nested
  `ExternalReservationImported` (`:110`). Three views — `AvailableRoomsView` (`:43`),
  `PendingConfirmationsView` (`:77`), `UnconfirmedReservationsView` (`:153`). One trigger (`:11`), two
  automations — `ConfirmationEmailReactor` (`:86`, event-activated) and
  `UnconfirmedReservationExpirer` (`:126`, `every "0 2 * * *"`) — and one translation (`:106`) reading
  the undeclared `BookingComWebhookView`, which stays undeclared because only an automation's `reads`
  resolves (`tasks/learnings.md:211-214`). No version header, no description, no invariant, no spec.
  `cli.RunValidate` returns no error for it today (verified).
- `examples/dcb_model.emod` is the DCB model: `mode dcb`, four slices declared directly on the
  context, tagged events and `decides_on` predicates. Untouched by this story.
- `examples/error_diagnostics_test.emod` is authored to fail. Untouched.

**What already guards the examples.** `internal/cli/validate_test.go` `t.Run("examples", …)`
(`:51-88`) drives `examplePaths` (`:773-791`), which reads `../../examples`, splits on the
`_test.emod` suffix, runs every other file through `cli.RunValidate` expecting no error, and requires
each `_test.emod` file's expected diagnostics to be listed in the `demonstrated` map. A new example is
guarded the day it lands. The sibling leaf at `:36-48` covers `internal/parser/testdata/*.emod`
separately.

**Two Go tests read the flagship example for its content**, and both must keep passing:
`internal/export/export_test.go:3984-4026` and `internal/cli/export_test.go:452-491` each locate the
slice named "Reserve a Room", read its trigger, and assert the trigger's name, actor and `reads` plus
the absence of a `kind` key. So that slice keeps its name and its trigger keeps its three entries;
everything else in the file is free to move. Both read the example through `lexer.Scan` +
`parser.New(...).Parse()` in a `//go:build unit` test, which is the precedent for a leaf that asserts
what an example declares.

**The DSL reference.** `docs/dsl-reference.md`, 689 lines, thirteen numbered sections and twenty
`###` sub-headings. Six ` ```emod ` fences (`:17`, `:35`, `:55`, `:175`, `:460`, `:577`), all complete
models; every skeleton with `<placeholder>` names sits behind a plain fence. The places this batch
reaches, and who reaches them:

| Line | What it holds | Reached by |
|---|---|---|
| `:33` | the Strings list, naming which quoted values carry a name, prose, or a machine-read value | **Task 4** — a spec's name, a wire type and a delay are three it does not account for |
| `:147-253` | `### invariant`, its fenced model at `:175` and the bullet at `:247` | US-008 Task 4 |
| `:267` | the slice skeleton's `flow { ... }` annotation, "0+ command→event wiring" | **Task 4** — falsified by US-009's second entry kind |
| `:324-351` | `### Automation Pattern`: skeleton, the `on`/`every`/`reads`/`command`/`target context` bullets, the exactly-one-of paragraph | **Task 4** — `after` appears nowhere |
| `:376-424` | `### spec`, ending on a sentence saying the view, automation and translation shapes "are not part of the language yet" | US-007 Task 8 (the four shapes), US-010 Task 8 (payloads) |
| `:426-438` | §7 Flows, "a command produces an event", and the skeleton behind a plain fence | US-009 Task 11 |
| `:451` | §8's field-type bullet | US-010 Task 8 |
| `:493-505` | `### External Source Events`, the precedent for an event-level attribute as a sub-heading | US-012 Task 7 adds its wire-type sibling near here |
| `:634-655` | §11 Cross-References: the table and the validation bullets | US-007 Task 8, US-009 Task 11, US-010 Task 8 |
| `:676-689` | §13 Diagram Palette, machine-read by `dslReferencePalette` | nobody; untouched |

**The documented-models guard.** `internal/oracle/oracle_test.go` `t.Run("documented models", …)`
(`:112-129`) extracts every ` ```emod ` fence from `README.md` and `docs/dsl-reference.md` via
`emodBlocksIn`, requires each document to yield at least one block, and requires `oracle.Check` — the
whole pipeline including the linter — to report nothing for each. So a fenced block Task 4 adds is
checked, and a fenced block any upstream documentation task adds is checked too.

**The rules the examples must satisfy.** `RunValidate` (`internal/cli/validate.go`) is `oracle.Check`,
which appends `linter.Lint` and returns an error for any non-empty list whatever the severity. The
rules in the tree today that constrain new example content: `view-naming`, `god-view`,
`clickbait-event`, `left-chair`, `command-past-tense`, `automation/missing-todo-list`, the `dcb/*`
family, and the validator's `orphan-command` / `orphan-event`. The rules this batch adds and that this
story is the first checked-in model to meet together: `spec/command-without-spec`,
`spec/no-rejection-path`, `spec/invariant-never-exercised`, `spec/given-outside-boundary`,
`flow/rejection-without-spec`, `wire/type-format`.

**Not touched, deliberately.** Everything under `internal/` and `cmd/` except the one test file
carrying the coverage leaf; `editors/`; `README.md`; `docs/proposals/`; `user-stories/`;
`tasks/completed/` and `tasks/learnings.md`; `e2e/` and `e2e-viewer/`;
`internal/parser/testdata/*.emod`; `examples/dcb_model.emod` and
`examples/error_diagnostics_test.emod`.

---

## Tasks

### Task 1: Pin the version, describe the constructs, bind wire types and delay an automation in the flagship example

**Behavior:** `examples/all_patterns.emod` shows the four constructs of this batch that stand alone —
the version header pinning the file, a `description` on every construct kind that accepts one, a wire
type on events, and an automation that fires a fixed duration after its activation event — and a test
reads them back off the parsed example, so the file's claim to demonstrate them is checked rather than
asserted. Each of the four is independently valid: none opens a lint gate, and the file returns no
error from `cli.RunValidate` after every one of them.

**Acceptance Criteria:**
- [x] `examples/all_patterns.emod` declares the version header on its own line ahead of the `model`
      declaration, and the file's existing leading comment is preserved above it — verified acceptable:
      a header written below a leading comment parses and validates with exit 0
- [x] The `model` and the `actor` are written in their block forms, each carrying a `description`
- [x] Every construct kind in the file that accepts a `description` carries one somewhere in the file:
      `context`, `aggregate`, `slice`, `trigger`, `command`, `event`, `view`, `automation` and
      `translation` — so `emod glossary` over the example renders a vocabulary with no empty definition
      for a kind the file could have described
- [x] At least two events state a wire type, the two values differ, and each is two or more
      dot-separated segments built from lowercase letters, digits and hyphens with no empty segment and
      no segment opening or closing with a hyphen — the shape `wire/type-format` accepts, per US-012's
      Open question 4
- [x] At least one event states no wire type, so the example shows both halves of "events without a
      wire type validate and export exactly as before"
- [x] The event-activated automation states a delay on its activation line, expressed as a value
      `time.ParseDuration` accepts
- [x] The schedule-activated automation `UnconfirmedReservationExpirer` states no delay — `after` on a
      schedule-driven automation is a validation error in US-013, and the example is where a reader
      learns the two never combine
- [x] `cli.RunValidate` returns no error for `examples/all_patterns.emod`, so no rule in the tree
      fires: the delay changes no event's producer, and the descriptions and wire types are additive
- [x] A leaf reads `examples/all_patterns.emod` from disk, parses it, and requires: the version header
      declared; a description present on the model, on the actor, and on at least one construct of each
      of the nine kinds above; at least two distinct wire types and at least one event without one; and
      at least one automation stating a delay while at least one automation states a schedule and no
      delay. Presence only — the leaf names no construct, no wire type and no duration, so the example
      stays free to grow
- [x] That leaf runs under `task test:unit`, and its failure output names which construct kind was
      missing rather than reporting a bare count
- [x] `internal/export/export_test.go` and `internal/cli/export_test.go` pass with no edit: the slice
      named "Reserve a Room" keeps its name and its trigger keeps its `actor` and `reads` entries
- [x] `git diff` touches `examples/all_patterns.emod` and one test file, and nothing else — in
      particular `internal/parser/testdata/all_patterns.emod`, `examples/dcb_model.emod`,
      `examples/error_diagnostics_test.emod` and `README.md` are unchanged
- [ ] No field column in `examples/all_patterns.emod` re-aligns and no blank line moves except where
      this task's own additions require it: `emod fmt` is not run over the file, and the file is not
      required to pass `emod fmt --check`, which it does not pass today either — **superseded by
      Open question 5's corrected answer: the file is formatted and passes `emod fmt --check`**
- [x] `task test:unit` passes with no test skipped or weakened

**Affected Files/Modules:**
- `examples/all_patterns.emod` — the header, the `model` (`:2`) and `actor` (`:4`) block forms,
  descriptions across the file, wire types on the declared events, and the activation line of
  `ConfirmationEmailReactor` (`:86-91`)
- `internal/cli/validate_test.go` — the coverage leaf, in the `t.Run("examples", …)` group (`:51-88`)
  that already owns what the repository claims of its examples

**Patterns to Follow:**
- The construct content: `docs/proposals/specs-and-metadata-proposal.md:432-583`, whose Worked Example
  writes the block forms, the descriptions, the wire types and the delay in the same hotel domain this
  example already models
- Which constructs accept a description, and the guard that a fixture describes all of them:
  `describableConstructs` and the subtest above it in `internal/parser/parser_test.go` — the same walk
  this leaf's coverage claim mirrors
- Reading a repository `.emod` file by relative path from a `//go:build unit` test and parsing it:
  `internal/export/export_test.go:3984-3996`; `internal/importer/importer_test.go:77` is the older
  precedent
- The group and leaf shape for a claim about `examples/`: `internal/cli/validate_test.go:51-88` and
  `examplePaths` (`:773-791`)
- `tasks/learnings.md:256-259` — a `Declared…`-style getter answers `nil` for a model declaring none of
  the construct, so each presence assertion needs a non-empty expectation, not an equality against a
  possibly-empty list
- `tasks/learnings.md:126-129` — the leaf's expected values come from this task's criteria, never from
  a second call into the parser
- `tasks/learnings.md:336-339` — `emod fmt <file>` writes in place, so no verification step runs it
  against the tracked example

**Testable:** Yes — through `cli.RunValidate` and through `lexer.Scan` + `parser.Parse` over the
example read from disk, all exported.

**Verification:** `mise exec -- task test:unit` passes;
`go run ./cmd/emod validate examples/all_patterns.emod` exits 0;
`go run ./cmd/emod glossary examples/all_patterns.emod` prints a definition for every term;
`git diff --stat` lists two files.

**Depends on:** None — but see the startability note: US-012 and US-013 must have landed.

---

### Task 2: State the invariants, specs and rejection path in the flagship example

**Behavior:** `examples/all_patterns.emod` shows the behaviour half of the batch — the business rules
its aggregates keep, a Given-When-Then spec for every command it declares, the spec shape belonging to
each of the four slice patterns, example payloads on the references inside them, and a rejection edge
on the timeline — and still returns no error from `cli.RunValidate`. This is one indivisible change:
the moment the file states its first spec, `spec/command-without-spec` opens on every command it
declares, so a partial adoption does not validate.

**Acceptance Criteria:**
- [x] Every aggregate that owns a slice stating a rejection declares at least one `invariant`, and
      every invariant the file declares is named by at least one `then rejected` in a slice of the
      scope that declares it — so `spec/invariant-never-exercised` reports nothing
- [x] Every command the file declares — the five under both contexts — is named by the `when` of at
      least one spec, so `spec/command-without-spec` reports nothing. A `then command` outcome does not
      count, per US-008's Open question 2
- [x] Every command that any spec exercises has at least one spec among its own whose `then` is a
      rejection, so `spec/no-rejection-path` reports nothing
- [x] All four spec shapes appear in the file, each in a slice that declares the construct US-007's
      Open question 1 requires of it: a `then` event list and a `then rejected` in slices declaring a
      command; a `then view` outcome in a slice declaring a view; a `then command` outcome in a slice
      declaring an automation
- [x] Both automation spec shapes appear: one spec whose `when` names an event-activated automation's
      activation event, and one in a schedule-activated automation's slice that omits `when` entirely
      — the two are told apart by the `then` shape, and the file is where a reader sees that
- [x] A spec in a view slice omits `when` and concludes with a `then view` outcome naming a view the
      file declares
- [x] Every event named in a `given` list belongs to the aggregate enclosing the slice that states it,
      so `spec/given-outside-boundary`'s aggregate arm reports nothing. Aggregate "Notification"
      declares no event of its own, so the spec exercising `SendConfirmationEmail` states no `given`
      unless that slice first declares an event
- [x] At least one event reference in a `given`, at least one command reference in a `when`, and at
      least one event reference in a `then` list carry an example payload, and every field a payload
      names is declared on the referenced construct's `fields`
- [x] At least one payload states a literal that is not a string, and the field it names is declared
      with a type that literal satisfies — so the example shows more than one of the three literal
      forms
- [x] At least one `flow` block states a rejection entry naming a command and an invariant declared in
      the enclosing scope, and the slice declaring that entry also declares a spec whose `when` names
      that same command and whose `then` rejects that same invariant — both halves, on that slice, per
      US-009's Open question 5, so `flow/rejection-without-spec` reports nothing
- [x] `cli.RunValidate` returns no error for `examples/all_patterns.emod`, and `cli.RunLint` likewise:
      the six rules this batch adds and the rules already in the tree are all quiet, including
      `orphan-command` and `orphan-event`, which a spec does not satisfy
      (`tasks/learnings.md:191-194`) — so every command keeps its flow, automation or translation and
      every event keeps its producer
- [x] The coverage leaf from Task 1 grows to require: at least one invariant declared; at least one
      spec stating each of the four `then` shapes; at least one spec carrying a payload on a `given`, a
      `when` and a `then` reference; and at least one rejection entry in a `flow`. Presence only, as
      before
- [x] `internal/export/export_test.go` and `internal/cli/export_test.go` pass with no edit
- [x] `git diff` touches `examples/all_patterns.emod` and the one test file and nothing else;
      `examples/dcb_model.emod`, `examples/error_diagnostics_test.emod`,
      `internal/parser/testdata/*.emod`, `internal/test/fixtures.go` and every golden and canonical
      constant in the repository are unchanged
- [ ] `emod fmt` is not run over the file and the file is not required to pass `emod fmt --check`
      — **superseded by Open question 5's corrected answer**
- [x] `mise exec -- task test:unit` passes with no test skipped or weakened

**Affected Files/Modules:**
- `examples/all_patterns.emod` — invariants on aggregates "Reservation" (`:7`) and "Notification"
  (`:167`); specs in the slices at `:10`, `:42`, `:55`, `:76`, `:95`, `:125`, `:152` and `:168`; a rejection
  entry in one of the `flow` blocks at `:36`, `:70`, `:146`
- `internal/cli/validate_test.go` — the coverage leaf Task 1 added

**Patterns to Follow:**
- The spec content and its payloads: `docs/proposals/specs-and-metadata-proposal.md:432-583`, the
  Worked Example — but read F2 first, since it does not validate as written
- The shape a spec takes in a checked-in model, in both slice homes and with both outcomes:
  `internal/test/fixtures.go`'s `SpecLibraryLending`, and the rejection specs US-008 Task 1 adds to it
- The rule each criterion answers to, and why it fires:
  `tasks/us-008-lint-spec-coverage-and-boundary-assumptions.md` Open questions 1, 2, 5 and 6;
  `tasks/us-007-write-specs-for-view-automation-and-translation-slices.md` Open question 1 for the
  outcome-to-slice table; `tasks/us-009-show-rejection-paths-on-the-timeline.md` Open question 5 for
  what "exercising" means on a rejection edge
- `tasks/learnings.md:466-469` for why a lint diagnostic of any severity fails `emod validate`, and
  `:471-474` for the rules a model gaining specs must keep quiet
- `tasks/learnings.md:191-194` — a spec is not a reference, so no flow may be removed in favour of one
- `tasks/learnings.md:46-49` — never write emod source with `%q`, which binds if any verification step
  writes the example to a temporary file

**Testable:** Yes — through `cli.RunValidate`, `cli.RunLint`, and the parsed example.

**Verification:** `mise exec -- task test:unit` passes;
`go run ./cmd/emod validate examples/all_patterns.emod` exits 0;
`go run ./cmd/emod lint examples/all_patterns.emod` exits 0;
`go run ./cmd/emod slices list examples/all_patterns.emod` prints the same table as before, since a
spec changes no slice's pattern;
`git diff --stat` lists two files.

**Depends on:** Task 1

---

### Task 3: Add `examples/specs_hotel.emod` as a model that validates

**Behavior:** A second example carries the proposal's Worked Example into the repository as working
model source: the invariants a hotel reservation keeps, specs with example payloads that state the
repeated room id and the overlapping dates, a rejection edge on the timeline, wire types on the events
the outside world consumes, and an automation that releases a hold a fixed duration after it was
taken. It returns no error from `cli.RunValidate`, and the existing `examples/` guard covers it the
moment it lands.

**Acceptance Criteria:**
- [x] `examples/specs_hotel.emod` exists, opens with the version header, and models the domain the
      proposal's Worked Example models — the reserve, hold and release-hold lifecycle with the views
      and the automation that closes its own loop
- [x] It declares at least one invariant, and every invariant it declares is named by a `then rejected`
      in a slice of the scope that declares it
- [x] Every command it declares is named by the `when` of at least one spec, and each of those commands
      has at least one spec whose `then` is a rejection — F1's arithmetic, which the Worked Example as
      written does not satisfy for `HoldRoom` and `ReleaseHold`
- [x] At least one spec states example payloads that link the scenario by repeating a value — the same
      room id in `given` and in `when` — and every field a payload names is declared on the referenced
      construct's `fields`, with each literal satisfying that field's declared type
- [x] At least one `flow` block states a rejection entry, with its exercising spec on the same slice
      naming the same command and the same invariant
- [x] Every event it declares that states a wire type states a distinct one conforming to the shape
      `wire/type-format` accepts
- [x] One automation fires a fixed duration after its activation event, reading a view of the work it
      has left to do, and the event that automation ultimately causes is one that view subscribes to —
      the closed loop the proposal states as the point of the example's shape
- [x] Every view the file declares is named so `view-naming` accepts it, subscribes to fewer than five
      events, and every event carries more than a single identifier field — the Worked Example's
      `UnreleasedHolds` does not satisfy the first of these and the name moves with every reference to
      it
- [x] Every command is named by a flow, an automation or a translation and every event is produced by
      a flow, an external source or a translation, so neither `orphan-command` nor `orphan-event`
      reports
- [x] `cli.RunValidate` and `cli.RunLint` both return no error for `examples/specs_hotel.emod`, and it
      is covered by `internal/cli/validate_test.go`'s `t.Run("examples", …)` group with no edit to
      `examplePaths` or to the `demonstrated` map — the guard enumerates the directory
- [x] `internal/cli/validate_test.go`'s examples group reports a subtest named after the new file, so a
      red run points at one example
- [x] `git diff` adds one file under `examples/` and changes nothing else: no Go file, no other
      example, no fixture, no golden
- [ ] The file is not required to pass `emod fmt --check`, and `emod fmt` is not run over it —
      **superseded by Open question 5's corrected answer**
- [x] `mise exec -- task test:unit` passes with no test skipped or weakened

**Affected Files/Modules:**
- `examples/specs_hotel.emod` — new

**Patterns to Follow:**
- The model itself: `docs/proposals/specs-and-metadata-proposal.md:432-583`, and the two paragraphs
  below it (`:581-583`) stating what the example's shape is meant to show — the todo list that closes
  its own loop, and one pattern per slice so the outcome-to-slice rule stays decidable
- The file's layout, comment header and hand alignment: `examples/all_patterns.emod` as Tasks 1 and 2
  leave it — a second example that contradicts the flagship is worse than one that repeats it
- The rules the file must keep quiet, and why: F1 and F2 above, plus `tasks/learnings.md:471-474`
- `tasks/learnings.md:531-534` — the guard enumerates `examples/` and splits on the `_test.emod`
  suffix, so this file must *not* carry that suffix and must *not* be added to any list
- `tasks/learnings.md:151-154` — the sibling convention for a new model in the repository: write it,
  run the whole pipeline over it, and let the diagnostics tell you it is not clean

**Testable:** Yes — through `cli.RunValidate` and `cli.RunLint` over the new file, which the existing
directory-enumerating guard reaches without being told about it.

**Verification:** `mise exec -- task test:unit` passes, its examples group naming the new file;
`go run ./cmd/emod validate examples/specs_hotel.emod` exits 0;
`go run ./cmd/emod lint examples/specs_hotel.emod` exits 0;
`git status --porcelain` lists one added file.

**Depends on:** Task 2 — not technically, since it shares no file, but the gate arithmetic in F1 is
settled once on the flagship example and applied here, and the two files should agree on how a hotel
reservation is modelled.

---

### Task 4: Document the elapsed-time automation and close the reference's coverage of the batch

**Behavior:** `docs/dsl-reference.md` teaches the one construct in this batch no other story documents
— an automation that fires a fixed duration after its activation event — and the document reads as one
reference rather than as five appended sections: the sentences the batch falsified say what is now
true, every construct the batch adds is taught with at least one example, and every in-document link
resolves against the finished document.

**Acceptance Criteria:**
- [x] `### Automation Pattern` (`docs/dsl-reference.md:324-351`) documents the delay: its skeleton
      writes it as the optional suffix on the `on` line, a bullet beside the existing `on`, `every`,
      `reads`, `command` and `target context` bullets describes it, states that the value is a Go
      duration and that a value which does not parse as one is a validation error with location, and
      states that without it the automation reacts immediately
- [x] That section states that the delay and the schedule never combine — a delay on a schedule-driven
      automation is a validation error — and says why, so the exactly-one-of paragraph and the delay
      read as one rule rather than two
- [x] It states what the diagrams do with the delay: the duration rides on the `event -> automation`
      edge while the clock badge on the automation box stays reserved for the schedule, so a relative
      delay and a wall-clock schedule read differently
- [x] `### Automation Pattern` heading text is byte-identical to what it is now, so the six links
      citing `#automation-pattern` still resolve
- [x] The Strings list (`docs/dsl-reference.md:33`) accounts for the quoted values this batch adds — a
      spec's name, an event's wire type and an automation's delay — either by naming each in the group
      it belongs to or by wording the sentence so it does not claim to enumerate every quoted value
- [x] The slice skeleton's `flow` annotation (`docs/dsl-reference.md:267`) describes what a `flow`
      block holds now that it accepts a second entry kind, rather than naming the command-to-event
      wiring alone
- [x] Every construct this batch adds is taught somewhere in the reference with at least one example,
      checked against the enumeration in this document's Story Reference: the version header, the
      `description` attribute and the two block forms, a keyword in field-name position, `invariant` in
      both scopes, `spec` with `given`/`when` and all four `then` shapes, payload literals, the
      rejection `flow` entry, an event's wire type, and the automation delay. A construct whose section
      an upstream story owns is checked as present, not rewritten
- [x] Every ` ```emod ` fence in `docs/dsl-reference.md` is a complete model `oracle.Check` reports
      nothing for, and any fence this task adds is either such a model or carries a plain fence
      instead; `internal/oracle/oracle_test.go`'s "documented models" leaf passes over every block in
      both documents
- [x] No `## <n>.` heading is added, removed, renamed or reordered, and no `### ` heading is renamed:
      the `^## [0-9]+\.` list is reconciled against the `\(#[0-9]+-` list and the `^### ` list against
      the `\(#[a-z]` list, both listed in full and compared against each other rather than spot-checked
      — this is the one reconciliation over the document after every upstream edit has landed
- [x] §13's `Diagram Palette` table is untouched and `TestExporterPaletteMatchesReference`
      (`internal/diagram/contract_test.go`) passes: the section is machine-read, its heading locates it
      and its six rows are parsed
- [x] Any invocation this task writes or edits puts its flags ahead of the file path or takes no
      positional argument, matching every invocation the reference writes today — urfave/cli v2
      discards a flag written after the path
- [x] The document reads as if it had always described these constructs: no "now supports", no note of
      what a section used to say, no migration paragraph anywhere in it
- [x] `git diff` touches `docs/dsl-reference.md` only. `README.md`, `examples/`, `docs/proposals/`,
      `user-stories/` and every file under `internal/` are unchanged
- [x] `mise exec -- task test:unit` passes with no test skipped or weakened

**Affected Files/Modules:**
- `docs/dsl-reference.md` — `### Strings` (`:31-37`), the slice skeleton in §5 (`:259-270`), and
  `### Automation Pattern` in §6 (`:324-351`); the rest of the document is read and reconciled, not
  rewritten

**Patterns to Follow:**
- The construct's shape and its two rules: `docs/proposals/specs-and-metadata-proposal.md:299-325` for
  the elapsed-time automation, and `user-stories/specs-and-metadata.md:169-179` for the criteria,
  including the diagram split between edge label and box badge
- The bullet-and-skeleton voice, and how a section states a validation error with its message: the
  `every` bullet directly above, which is the sibling shape for an
  entry whose quoted value is machine-read and whose malformed form has a named diagnostic — it sits at
  `docs/dsl-reference.md:341-345`, and the `on` bullet the delay attaches to at `:340`
- The consumer-bullet shape for what does and does not read a value: §10's closing bullets
  (`docs/dsl-reference.md:626-631`)
- `tasks/learnings.md:36-39` and `:541-544` — the two anchor families and the instruction to re-derive
  them by listing both sides and reconciling, never by spot-checking
- `tasks/learnings.md:411-414` — §13 is machine-read, so its heading text and its six-row table are
  load-bearing and this task leaves them alone
- `tasks/learnings.md:526-529` — an ```emod fence is a promise the block validates whole; a skeleton
  with `<placeholder>` names cannot, and takes a plain fence
- `tasks/learnings.md:111-114` — the flag-after-path defect, which this task must not propagate and
  does not fix
- `~/.claude/rules/markdown-docs.md` — the result reads as a first version; the language's state is
  content, the document's history is not

**Testable:** No — prose. Its executable halves are already carried: `oracle.Check` runs every
` ```emod ` fence through the whole pipeline, and `TestExporterPaletteMatchesReference` holds §13. The
anchor reconciliation has no runner in `Taskfile.yml` or `.github/workflows/ci.yml`, and adding a
markdown link check would have to cover ninety-odd markdown files, most of them archives — a larger
change than this story.

**Verification:** `mise exec -- task test:unit` passes, including the "documented models" leaf and
`TestExporterPaletteMatchesReference`;
`rg -n '^## [0-9]+\.' docs/dsl-reference.md` and `rg -n '\(#[0-9]+-' docs/dsl-reference.md` reconcile;
`rg -n '^### ' docs/dsl-reference.md` and `rg -n '\(#[a-z][a-z-]*\)' docs/dsl-reference.md` reconcile;
`git diff --stat` lists one file.

**Depends on:** Tasks 1, 2, 3 — and every upstream documentation task: US-007 Task 8, US-008 Task 4,
US-009 Task 11, US-010 Task 8, US-012 Task 7. The coherence criteria are claims about the finished
document, so they cannot close while an upstream story still owes it a section.

---

## Summary

**Four tasks**, ordered so the flagship example is complete before a second example repeats its
vocabulary, and so the reference's coherence pass runs over a document nothing further will edit.

Task 1 comes first because its four constructs open no lint gate: each is additive, each leaves the
example validating on its own, and the coverage leaf it establishes is the machinery Task 2 extends.
Task 2 is one task rather than five because F1 makes it indivisible — the moment the file states a
spec, every command in it needs one, so there is no smaller green step. Task 3 shares no file with
either and could run alongside them, but is ordered third so the gate arithmetic is settled once and
the two examples agree on the domain. Task 4 is last by necessity, not preference: its closing
criteria are claims about a document five other stories are still editing.

**Story criteria coverage:**

| Story criterion | Task |
|---|---|
| `examples/all_patterns.emod` exercises every new construct | 1 (version header, descriptions, wire types, delay), 2 (invariants, all four spec shapes, payloads, rejection edge) |
| ... and passes `emod validate` | 1 and 2, each closing on `cli.RunValidate` returning no error |
| A new `examples/specs_hotel.emod` mirrors the proposal's worked example — invariants, specs with payloads, a rejection flow edge, a wire type and a timer — and passes `emod validate` | 3 |
| `docs/dsl-reference.md` documents each new construct with at least one example | 4 for the automation delay, which no other story documents, and for the enumeration that makes the claim checkable; the individual sections belong to US-007 Task 8, US-008 Task 4, US-009 Task 11, US-010 Task 8 and US-012 Task 7 |

Carried along, not stated by the story: the leaf that reads the flagship example back and requires it
to *state* each construct (1 and 2), without which the first criterion is a claim nothing re-checks;
the correction of §1's Strings list and §5's slice-skeleton annotation, two sentences the batch
falsifies and no story claimed (4); and the single anchor reconciliation over the finished reference
(4), which each upstream task performs against its own end state and none against the last.

**Left to other stories:** every section of `docs/dsl-reference.md` in the ownership table above;
canonical formatting of the new constructs and the activation-keyword column the proposal's worked
example shows (US-014); LSP hover, completion and navigation over specs, invariants and payloads
(US-015); the `--specs` diagram flag (US-016); highlighting the new keywords and payload literals
(US-017); and the `emod export <file> -f <format>` and `emod diagram <file> -f <format>` flag-ordering
defect, which reaches `README.md:159-163` and `:168-171` and belongs to a CLI story.

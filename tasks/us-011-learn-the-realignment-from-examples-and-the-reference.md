# US-011: Learn the realignment from examples and the reference

## Progress
- [x] Task 1: Show the realigned automation in the examples and hold them to `emod validate`
- [x] Task 2: Document the kindless trigger and the realigned automation in the reference's pattern sections
- [ ] Task 3: Correct the reference's cross-references and remaining activation prose
- [ ] Task 4: Put the new forms in the README quick-start and validate every fenced model
- [ ] Task 5: Sweep the repository for removed spellings

---

## Story Reference

`user-stories/triggers-and-automations.md` → **US-011: Learn the realignment from examples and the
reference** (eleventh of eleven stories in "Triggers and Automations"). Design notes:
`docs/proposals/triggers-and-automations-proposal.md` — section 1 (`:52-71`) for the kindless trigger,
section 2 (`:73-98`) for `reads`, section 3 (`:100-109`) for the two activation forms, section 4
(`:111-121`) for the removal table, and the Worked Example (`:315-412`) for the before-and-after of the
exact domain the flagship example models. `:456` lists this story's surfaces from the proposal's side.

**In scope:** the prose and the narrative shape of the teaching surfaces — `examples/all_patterns.emod`
gaining an automation that names the view it reads and a schedule-activated automation beside the view
*it* reads; `docs/dsl-reference.md` documenting `on`, `every` and `reads` on an automation and the
trigger without a kind, and stating which Event Modeling chain each of the two elements belongs to; the
README quick-start; and a repository-wide sweep for the two retired spellings. Carried along because
the criteria are otherwise unfalsifiable: a test that runs `emod validate` over every file under
`examples/` (nothing does today), and a test that runs every fenced emod model in the README and the
reference through the same pipeline (`docs/dsl-reference.md` is the one keyword surface no test reaches,
which is exactly how it came to be documenting a spelling the parser rejects).

**Out of scope:** the parser, formatter, exporter and renderer changes themselves (US-004 and earlier);
the `reads` edge (US-005); lane placement and the "Wireframes" lane label (US-006); the palette
(US-007 — see the coordination note below); the `automation/missing-todo-list` rule (US-008); LSP hover,
completion and navigation (US-009); VS Code and tree-sitter highlighting (US-010); `wireframe` on a
trigger and `on <Event> after "<duration>"`, neither of which is in the language; and a DSL version
bump — the header stays at 1.

### The boundary with US-004, stated

US-004 has not been implemented. **This breakdown is written against the post-US-004 tree** and assumes
that when work starts, `trigger "<name>" { ... }` parses with the quoted name directly after the
keyword and `trigger UI|Schedule|Processor "<name>"` is rejected. If that is not true, Task 1 fails at
its first criterion and this story is not startable.

The two stories genuinely overlap on `.emod` files, so the split is by *kind of file*, not by edit:

| Owner | Files |
|---|---|
| **US-004** — the mechanical migration | `internal/parser/testdata/*.emod`, `internal/test/fixtures.go`, the inline sources in `internal/parser/parser_test.go`, `internal/formatter/formatter_test.go`, `internal/cli/fmt_test.go`, `internal/lsp/*_test.go`, `e2e-viewer/tests/helpers.js`, `editors/tree-sitter-emod/grammar.js` and its corpus under `test/corpus/`, and the one-line respelling of the trigger in `examples/all_patterns.emod` |
| **US-011** — the prose and the narrative | `docs/dsl-reference.md`, `README.md`, the *content* of `examples/all_patterns.emod` beyond that one line, the live proposals under `docs/proposals/ai/`, and the two guard tests |

The single shared file is `examples/all_patterns.emod`, and the shared line is its trigger. Every
criterion here that touches it is written as an **end state** ("every trigger in the file is written
with the quoted name directly after the keyword"), never as an edit, so a task that finds the line
already migrated closes with an empty diff on it rather than undoing or repeating US-004's work.

Three files this story deliberately does **not** touch even though they hold the same text:

- `internal/parser/testdata/all_patterns.emod` is byte-identical to the example today, and diverges
  once this story adds slices. It is a parser fixture, validated by `e2e/tests/validate.test.ts:11` and
  read by `internal/importer/importer_test.go:77`; it stays as US-004 leaves it. Nothing asserts the
  two copies match.
- `examples/dcb_model.emod` declares no trigger and no automation, so the realignment does not reach it.
- `examples/error_diagnostics_test.emod` is a deliberately broken model (verified: `emod validate`
  reports six diagnostics and exits 1). It is the reason Task 1's guard cannot be "every file under
  `examples/` validates" — see consequence 2.

### Consequences of that boundary, decided

Nine things the story does not spell out:

1. *`emod validate` runs the linter, so "passes `emod validate`" means lint-clean too.* `RunValidate`
   (`internal/cli/validate.go:39`) calls `oracle.Check`, which appends `linter.Lint`, and returns an
   error for a non-empty diagnostic list whatever the severity. New example content is therefore
   constrained by rules the story never mentions: `view-naming` (a view's name must end in `View`),
   `clickbait-event` (an event carrying one ID field and nothing else), `god-view` (five or more
   `subscribes`), `left-chair` (a command referenced by three or more flows), `command-past-tense`, and
   the orphan checks in the validator (a command no flow, automation or translation references; an
   event no flow, external source or translation produces). Verified: `emod validate` and `emod lint`
   both exit 0 on `examples/all_patterns.emod` and `examples/dcb_model.emod` today.
2. *The broken example is guarded by asserting that it fails, not by skipping it.* A skip-list entry
   that silently starts validating is invisible; a leaf requiring `error_diagnostics_test.emod` to
   report diagnostics fails the day someone repairs it, which is when the guard should speak.
3. *Every automation in `examples/` gets a `reads`, including the event-activated one.* US-008 adds
   `automation/missing-todo-list` as a lint *warning*, and by consequence 1 a lint warning fails
   `emod validate`. An automation left without `reads` therefore breaks this story's own criterion the
   day US-008 lands. Giving `ConfirmationEmailReactor` (`examples/all_patterns.emod:77`) a view now is
   the cheap version of that; discovering it from US-008's failing example is the expensive one.
4. *Only an automation's `reads` resolves; the translation's undeclared view stays undeclared.*
   `tasks/learnings.md` records that `examples/all_patterns.emod` names `BookingComWebhookView` on a
   translation with no such view, and that widening the lookup would stop the flagship example
   validating. So the automation's new `reads` must name a view declared in the file, while the
   translation's `reads` at `:98` is left exactly as it is. Task 3 makes the reference honest about the
   asymmetry rather than leaving the cross-reference table implying all three are resolved.
5. *The example is hand-formatted and `emod fmt` is not run over it.* Verified:
   `emod fmt --check examples/all_patterns.emod` fails today — formatting would insert an `emod 1`
   header, re-align every field column and drop the file's blank lines, producing a diff that buries
   this story's change. New slices match the file's existing hand alignment.
6. *The scheduled automation goes in `examples/all_patterns.emod`, not in a new example file.* It is the
   file the story's first criterion names, the file `docs/dsl-reference.md:86` and `README.md` link to,
   and the file whose domain the proposal's Worked Example is already written in. A fourth example file
   would need its own place in the reference, the README and the linking.
7. *The roles-and-chains statement goes inside an existing numbered section, and no numbered section is
   added, removed or reordered.* This is the anchor hazard below, answered by not creating it. The
   statement belongs at the head of `## 6. Slice Patterns`, which is where the four patterns are
   introduced and the one place a reader meets both elements together.
8. *The reference gets no snippet-extraction test beyond fenced models.* Of the seven ` ```emod ` blocks
   in `docs/dsl-reference.md`, six are complete models and validate today (`:17`, `:35`, `:55`, `:175`,
   `:449`, `:566`); one (`:26`, illustrating identifiers) is two bare declaration lines and does not.
   The skeletons showing `<Placeholder>` syntax sit in plain fences already. So the guard is
   "a ` ```emod ` fence holds a complete model" plus a retag of the two fragments, not a general
   markdown-snippet validator — the placeholder skeletons cannot parse by construction and are the
   larger half of the document.
9. *"No document shows a removed spelling" means shows it as current syntax.* A document that names a
   spelling as removed, and an archived record of the world before the removal, are not showing it —
   `~/.claude/rules/markdown-docs.md` puts the subject matter's past squarely in the content. Task 5
   carries the enumerated residue list this reading produces, so the negative claim is checkable rather
   than asserted.

### The anchor hazard in `docs/dsl-reference.md`, and how these tasks answer it

`tasks/learnings.md` records that every heading is `## <n>. <Title>` and its in-document links are
number-prefixed slugs, that inserting or reordering a section renumbers every heading below it and
invalidates each link citing one of those numbers, and that nothing in `Taskfile.yml` or
`.github/workflows/ci.yml` checks it. **The current state is clean** — verified by listing both sides:

- Twelve numbered headings, 1–12: General Syntax, Version Header, Top-Level Constructs, Bounded
  Contexts, Slices, Slice Patterns, Flows, Fields, Dynamic Consistency Boundaries, Descriptions,
  Cross-References, Pipeline.
- Ten number-citing links, all resolving: `#10-descriptions` ×2 (`:86`, `:100`), `#6-slice-patterns`
  (`:272`), `#7-flows` ×2 (`:305`, `:631`), `#11-cross-references` (`:320`), `#2-version-header`
  (`:479`), `#3-top-level-constructs` (`:564`), `#8-fields` (`:612`), `#4-bounded-contexts` (`:640`).

There is a **second hazard the learnings do not record**: fifteen links cite `###` sub-heading slugs —
`#automation-pattern` ×4 (`:132`, `:629`, `:630`, `:631`), `#command-pattern` ×3 (`:100`, `:632`,
`:633`), `#translation-pattern` ×3, `#spec` ×3, `#view-pattern`, `#invariant`. Tasks 2 and 3 edit the
bodies of `### Command Pattern` and `### Automation Pattern`, so renaming either heading — to
"Trigger Pattern", say — silently breaks seven links. Both tasks carry a criterion holding the heading
text fixed, and Task 3 closes with the reconciliation over both link families.

**Coordination with US-007**, which owns "the palette documented in the DSL reference matches what the
exporters emit" and is the other story editing this file: a palette section appended after
`## 12. Pipeline` renumbers nothing and is the safe shape; a section inserted anywhere above it
renumbers every heading below and needs the same reconciliation. Whichever of the two lands second
re-derives the anchors.

### Learnings folded in

From `tasks/learnings.md`: `docs/dsl-reference.md` anchors embed the section number, and the fix is to
re-derive them by listing `^## [0-9]+\.` and `\(#[0-9]+-` and reconciling the two; `docs/dsl-reference.md`
is the one keyword surface no test reaches, and it still documents the retired automation `trigger`
spelling in the Automation Pattern section (`:329`, `:336`, `:340`) and the cross-reference table
(`:630`) — verified: pasting that skeleton into a file yields two diagnostics from `emod validate`
today, and this story owns fixing it; `internal/glossary` collects once and renders twice, and section
10's closing bullet about which constructs contribute no term is the doc counterpart, which stays
correct; urfave/cli v2 discards every flag written after the file argument, so every invocation in a
criterion or a README example writes its flags before the path or takes none; acceptance criteria
describe the working tree, and a commit-message receipt is the commit author's obligation, never a
criterion; an assertion whose expected value comes from the code under test cannot fail — which is why
the two guard tests read their inputs from disk and compare against `emod validate`'s own answer rather
than against a re-derivation; and never write emod source with `%q`, which reaches the guard tests as
soon as either of them writes an extracted block to a temporary file.

---

## Codebase Context

**Examples.** Three files. `examples/all_patterns.emod` (129 lines) is the flagship: model
"Hotel Reservation", one actor, two contexts, five slices in `Reservations` (command, view, command,
automation, translation) and one in `Notifications`. The trigger sits at `:11`, the sole automation
`ConfirmationEmailReactor` at `:77-81` (it states `on RoomReserved`, `command SendConfirmationEmail`,
`target context Notifications`, and no `reads`), the sole view `AvailableRoomsView` at `:43-51`
subscribing to `[RoomReserved, GuestCheckedOut]`, and the translation at `:96-111` reading the
undeclared `BookingComWebhookView`. `examples/dcb_model.emod` is the DCB model, `mode dcb`, four slices,
no trigger and no automation. `examples/error_diagnostics_test.emod` is intentionally broken.
**Nothing in the test suite validates any file under `examples/`** — `e2e/tests/validate.test.ts:11`
validates the parser testdata copy instead.

**The DSL reference.** `docs/dsl-reference.md`, 662 lines, twelve numbered sections. The places the
realignment reaches:

| Line | What it says now |
|---|---|
| `:33` | the Strings list, naming the constructs whose values are quoted |
| `:261` | the slice skeleton's `trigger` comment — correct as it stands |
| `:286-289` | the Command Pattern skeleton, writing the trigger with a `<Kind>` slot |
| `:305` | "`trigger` is optional — a slice may define a command directly without a trigger" |
| `:322-340` | the Automation Pattern: heading, prose, skeleton writing `trigger <EventName>`, three bullets, and "`trigger` and `command` are required" |
| `:425` | the Flows note — "automation already shows the trigger→command link" |
| `:582` | the section-10 descriptions example, a complete model writing a kinded trigger |
| `:630` | the cross-reference table's `event <Name>` row, listing `automation { trigger <Name> }` |
| `:632` | the `view <Name>` row, listing `trigger { reads <Name> }` and `translation { reads <Name> }` and not the automation's |

**The README.** `README.md`, 305 lines. Two ` ```emod ` fences: the quick-start model at `:25-62` and a
DCB illustration at `:87-110` written with literal `...` placeholders. **The quick-start does not
parse** — verified: `actor Guest`, `context Reservations {` and `aggregate Reservation {` write their
names unquoted, and `emod validate` reports 71 diagnostics from line 3 onward. The trigger at `:33`
writes a kind. `:117-127` documents six invocations with the format flag after the path, which
`tasks/learnings.md` records as silently exercising the default format.

**Test surfaces the guards would join.** `internal/cli/validate_test.go` is `//go:build unit`, package
`cli_test`, and drives whole commands through `cli.RunValidate(path, format)`; it already reads files
from disk. `internal/oracle/oracle_test.go` is one umbrella `TestCheck` whose "clean input" group
(`:24`) holds one `require.Empty(t, oracle.Check(...))` leaf per shared fixture — `oracle.Check(source,
filename)` performs no I/O, the caller supplies the text and the name. The precedent for a test reading
a repository file by relative path is `internal/importer/importer_test.go:77`.

**Retired spellings elsewhere in the repository** (full inventory, taken with `rg` over everything but
`node_modules`, for the trigger keyword followed by an identifier ahead of a quoted name and for the
trigger keyword followed by a bare identifier): the teaching surfaces above; five live proposals under
`docs/proposals/ai/` (`01` at `:99`, `:288`, `:328`, `:354`; `02:327`; `06:339`; `09:319`; `10:256-257`);
the code and fixture surfaces US-004 owns; and the archival set — the feature proposal's Problem
section, removal table, Worked Example and Versioning note, `docs/proposals/completed/proposal.md`,
`docs/proposals/completed/dcb-proposal.md`, `user-stories/triggers-and-automations.md`,
`tasks/learnings.md` and `tasks/completed/`. `docs/proposals/specs-and-metadata-proposal.md` is already
aligned (it writes the kindless trigger at `:449` and `on … after` at `:306`), and
`docs/wasm-architecture.md`, `docs/proposals/emod-desktop-proposal.md` and everything under
`user-stories/ai/` hold none.

**Not touched, deliberately.** Every Go and JS file except the two test files the guards land in;
`editors/`; `internal/viewer/`; `web/`; `e2e/` and `e2e-viewer/`.

---

## Tasks

### Task 1: Show the realigned automation in the examples and hold them to `emod validate`

**Behavior:** `examples/all_patterns.emod` shows the realigned pair as working model source — a trigger
carrying no kind, every automation naming the view it reads, and one automation activated by a schedule
beside the view *it* reads — and a test runs `emod validate` over every file under `examples/`, so the
examples stay executable artefacts rather than prose that happens to look like emod.

**Acceptance Criteria:**
- [ ] Every `trigger` in `examples/all_patterns.emod` is written with its quoted name directly after the
      keyword and no kind identifier — an end state, so if US-004 already made this edit the diff on
      that line is empty
- [ ] Every automation in `examples/all_patterns.emod` states `reads`, naming a view declared somewhere
      in the same file
- [ ] At least one automation in the file states `every` with a duration or a five-field cron
      expression, states no `on`, and states `reads`; the view it names is declared in the file and
      `subscribes` only to events the file declares
- [ ] Every view the file declares has a name ending in `View` and subscribes to fewer than five events,
      and every event carries more than a single identifier field — the `view-naming`, `god-view` and
      `clickbait-event` rules, which `emod validate` enforces because it runs the linter
- [ ] Every command the file declares is named by a flow, an automation or a translation, and every
      event it declares is produced by a flow, an external source or a translation, so neither
      `orphan-command` nor `orphan-event` is reported
- [ ] `cli.RunValidate` returns no error for `examples/all_patterns.emod` and `examples/dcb_model.emod`,
      and returns an error naming a diagnostic for `examples/error_diagnostics_test.emod`, from a test
      that enumerates the `examples/` directory rather than naming files — so a fourth example is
      covered on the day it lands, and the intentionally broken one is asserted broken rather than
      skipped
- [ ] The test's failure output names the file that failed, so a red run points at one example rather
      than at the directory
- [ ] The translation's `reads` at `examples/all_patterns.emod:98` still names a view the file does not
      declare, and the file still validates — only an automation's `reads` is resolved, and this example
      is the reason
- [ ] `git diff examples/all_patterns.emod` touches only the trigger line, the automation entries and
      the added slices: no field column re-aligns, no blank line moves, and no `emod 1` header appears,
      because `emod fmt` is not run over the file
- [ ] `git diff` is empty for `internal/parser/testdata/all_patterns.emod` and for both other files
      under `examples/`
- [ ] `go test -tags unit ./...` passes, with no expected constant edited anywhere: no Go
      test reads `examples/` today except the one this task adds

**Affected Files/Modules:**
- `examples/all_patterns.emod` — the trigger (`:11`), `ConfirmationEmailReactor` (`:77-81`), and new
  slices for the scheduled automation, the command and event it drives, and the views the two
  automations read
- `internal/cli/validate_test.go` — one leaf driving `cli.RunValidate` over the directory

**Patterns to Follow:**
- The model content: `docs/proposals/triggers-and-automations-proposal.md:88-98` for the todo-list
  automation and `:360-408` for the Worked Example's after half, which models the same hotel domain the
  example already uses and closes the loop the proposal describes — the event the processor causes is
  one the view it read subscribes to
- The slice shapes to copy live in the file itself: the view slice at `:42-52` and the automation slice
  at `:76-82`
- `internal/cli/validate_test.go` for the `//go:build unit` header, the `cli_test` package and the
  `RunValidate` call shape; `internal/importer/importer_test.go:77` for reading a repository file by
  relative path from a test
- `tasks/learnings.md` "A new shared fixture owes `internal/oracle` a zero-diagnostic subtest" is the
  sibling convention for fixtures, and its warning that DCB shapes trip `dcb/untagged-event` and
  `dcb/orphan-tag-key` applies to any `mode dcb` content
- `tasks/learnings.md` "CLI diagnostic tests must assert the distinguishing message text" — the leaf
  requiring `error_diagnostics_test.emod` to fail asserts what it reports, not merely that it failed

**Testable:** Yes — through `cli.RunValidate`, exported.

**Verification:** `go test -tags unit ./internal/cli/...`;
`go run ./cmd/emod validate examples/all_patterns.emod` exits 0;
`mise exec -- git diff --stat` lists two files.

**Depends on:** None (assumes US-004 has landed — see the boundary above)

---

### Task 2: Document the kindless trigger and the realigned automation in the reference's pattern sections

**Behavior:** `docs/dsl-reference.md` describes the two realigned elements as they are: a trigger takes
a quoted name and no kind, an automation states exactly one of `on` and `every` and may name the view it
reads, and the section that introduces the four patterns says which Event Modeling chain each element
belongs to — the trigger as the human entry point of the command chain, the automation as the processor
of the automation chain.

**Acceptance Criteria:**
- [ ] The Command Pattern skeleton (`docs/dsl-reference.md:286-289`) writes the trigger's quoted name
      directly after the keyword, with no kind slot, matching what the parser accepts after US-004
- [ ] The Automation Pattern skeleton (`:326-334`) writes `on` and `every` as the two activation forms,
      `reads` as the view entry, and `command` and `target context`, and writes `trigger` nowhere
- [ ] The bullets below it (`:336-340`) describe each entry, state that exactly one of `on` and `every`
      is required and that declaring both is an error, name the two accepted `every` expressions — a
      duration and a five-field cron expression — and state that a malformed expression is a validation
      error naming both forms
- [ ] The bullets state that `reads` is optional and that the name must resolve to a view declared
      anywhere in the model, which is what separates it from the trigger's and the translation's `reads`
- [ ] `## 6. Slice Patterns` opens with a statement that a trigger is the human entry point of the
      command chain and an automation is the processor of the automation chain, so a reader meets the
      distinction where both elements are introduced rather than having to infer it from two skeletons
- [ ] The complete model in section 10 (`:566-609`) writes its trigger without a kind, and
      `emod validate` accepts the block's text unchanged apart from that
- [ ] Pasting the Automation Pattern skeleton's entry names into a model — the shape the learnings
      record as yielding two diagnostics today — no longer names an entry the parser rejects
- [ ] The `### Command Pattern` and `### Automation Pattern` heading text is byte-identical to what it
      is now, so the seven links citing `#command-pattern` and `#automation-pattern` still resolve
- [ ] The list of `^## [0-9]+\.` headings is unchanged: twelve sections, same numbers, same titles, same
      order — this task adds no numbered section and reorders none
- [ ] `git diff docs/dsl-reference.md` touches only sections 6 and 10; sections 1–5, 7–9 and 11–12 are
      untouched, and no other file changes

**Affected Files/Modules:**
- `docs/dsl-reference.md` — `## 6. Slice Patterns` (`:276-341`) and the section-10 example (`:582`)

**Patterns to Follow:**
- The entry list and the exactly-one-of wording:
  `docs/proposals/triggers-and-automations-proposal.md:73-98` (the automation block and `reads`) and
  `:100-109` (the two activation forms and why exactly one is required)
- The Translation Pattern section directly below (`docs/dsl-reference.md:342-363`) is the sibling shape
  for a skeleton followed by a required-entries sentence; `### spec` (`:365-411`) is the shape for a
  section that explains a rule and its name resolution in prose
- `tasks/learnings.md` "`docs/dsl-reference.md` is the one keyword surface no test reaches, and a
  retirement story forgets it" names the two places in this file that still document the retired
  automation `trigger`; this task owns the first of them and Task 3 the second
- `tasks/learnings.md` "`docs/dsl-reference.md` anchors embed the section number" — the answer here is
  not to create the problem; Task 3 carries the reconciliation
- `~/.claude/rules/markdown-docs.md` — the reference reads as if it always described these forms: no
  "now", "previously", "the old spelling" or migration note anywhere in it

**Testable:** No — the change is prose and skeletons. The executable check over the section-10 model
arrives in Task 4, which cannot land earlier because the README quick-start it also covers does not
parse until Task 4 fixes it.

**Verification:** extract the ` ```emod ` block starting at `docs/dsl-reference.md:566` and confirm
`go run ./cmd/emod validate` exits 0 on it;
`rg -n '^## [0-9]+\.' docs/dsl-reference.md` lists the same twelve headings as before;
`rg -n '^### (Command|Automation) Pattern' docs/dsl-reference.md` lists both.

**Depends on:** 1

---

### Task 3: Correct the reference's cross-references and remaining activation prose

**Behavior:** the rest of `docs/dsl-reference.md` agrees with section 6 — the cross-reference table
resolves an automation's activation event through `on` and lists the automation among the constructs
that read a view, the prose that describes an automation's activation as a trigger no longer does, and
every in-document link resolves.

**Acceptance Criteria:**
- [ ] The `event <Name>` row of the cross-reference table (`:630`) names the automation's activation
      entry as `on` and not as `trigger`
- [ ] The `view <Name>` row (`:632`) names the automation's `reads` alongside the trigger's and the
      translation's
- [ ] The table or the prose around it states that only an automation's `reads` is resolved during
      validation, and that a trigger's and a translation's are recorded but unchecked — the table's
      opening sentence claims all its rows are resolved, and for two of those three it is not so
- [ ] The Flows note (`:425`) describes an automation as showing the link from its activation to its
      command, without calling the activation a trigger
- [ ] The Strings list (`:33`) accounts for the schedule expression, which is a quoted value that is not
      a human-readable name — either by naming it or by wording the sentence so it does not claim to be
      exhaustive over quoted values
- [ ] No line in `docs/dsl-reference.md` writes the trigger keyword inside an automation, and no line
      writes a kind identifier between the trigger keyword and a quoted name — asserted over the whole
      file, so section 6's fix and this one are proved together
- [ ] Every `(#<n>-<slug>)` link in the file cites a number that a `## <n>. <Title>` heading carries and
      a slug matching that heading's title: the ten links and the twelve headings are listed
      separately, with `rg -n '^## [0-9]+\.'` and `rg -n '\(#[0-9]+-'`, and reconciled against each
      other rather than spot-checked
- [ ] Every `(#<slug>)` link naming a sub-heading — `#automation-pattern`, `#command-pattern`,
      `#view-pattern`, `#translation-pattern`, `#spec`, `#invariant` — matches a `###` heading in the
      file, listed the same way
- [ ] `git diff` touches `docs/dsl-reference.md` only, and only outside sections 6 and 10, which Task 2
      settled

**Affected Files/Modules:**
- `docs/dsl-reference.md` — `## 11. Cross-References` (`:623-643`), the Flows note (`:425`), the Strings
  list (`:33`)

**Patterns to Follow:**
- `tasks/learnings.md` "Only an automation's `reads` resolves; a trigger's and a translation's must stay
  unchecked" is the fact the table has to state, and records why widening the lookup is not on offer
- `tasks/learnings.md` "`docs/dsl-reference.md` anchors embed the section number" — its closing
  instruction is exactly the reconciliation these two criteria ask for, and it records a stale
  `#7-cross-references` link that survived an earlier insertion unnoticed, which is what a spot-check
  misses and a listed reconciliation does not
- The table's existing rows are the shape for a new entry: declaration, the spellings that reference it,
  and the anchor links to the sections that show each
- `~/.claude/rules/markdown-docs.md`

**Testable:** No — prose and links, with no code surface. Nothing in `Taskfile.yml` or
`.github/workflows/ci.yml` runs a markdown link check, and adding one is a larger change than this
story: it would have to cover 92 markdown files, most of them archives.

**Verification:** `rg -n '^## [0-9]+\.' docs/dsl-reference.md` and `rg -n '\(#[0-9]+-'
docs/dsl-reference.md` reconcile; `rg -n '^### ' docs/dsl-reference.md` and `rg -n '\(#[a-z][a-z-]*\)'
docs/dsl-reference.md` reconcile; `rg -n 'trigger[ \t]+(UI|Schedule|Processor)\b' docs/dsl-reference.md`
and a search for the trigger keyword followed by a bare identifier both return nothing.

**Depends on:** 2

---

### Task 4: Put the new forms in the README quick-start and validate every fenced model

**Behavior:** the README's quick-start is a model a reader can copy into a file and run `emod validate`
on, written in the realigned forms — and a test proves that, for it and for every complete model the
reference embeds, by running the same pipeline `emod validate` runs. The document that has been drifting
from the language since the language last changed acquires the check that would have caught it.

**Acceptance Criteria:**
- [ ] The README quick-start model (`README.md:25-62`) parses, validates and lints clean: its actor,
      context and aggregate names are quoted, and its trigger carries no kind
- [ ] The quick-start shows the realigned automation as well as the trigger — an automation naming its
      activation with `on` and the view it reads — so a reader meets both halves of the realignment in
      the first model they see, with every command referenced and every event produced so the model
      stays clean under the orphan checks and the naming rules
- [ ] A test runs `oracle.Check` over every ` ```emod ` fenced block in `README.md` and
      `docs/dsl-reference.md` and requires no diagnostics from any of them
- [ ] The test names the document and the block's starting line in its failure output, so a red run
      points at one block
- [ ] The two blocks that are deliberate fragments rather than models — the identifiers illustration at
      `docs/dsl-reference.md:26-29` and the DCB illustration at `README.md:87-110`, which writes literal
      `...` placeholders — carry a fence other than `emod`, so the rule the test enforces is stated by
      the documents themselves: an `emod` fence holds a complete model
- [ ] The test finds a non-zero number of blocks in each of the two documents, so a change that breaks
      the extraction reports a failure rather than passing over an empty list
- [ ] The six blocks in the reference that are complete models today — at `:17`, `:35`, `:55`, `:175`,
      `:449` and `:566` — are covered by that count, and none of them is edited by this task
- [ ] Any invocation this task writes or edits in `README.md` puts its flags before the file path, since
      urfave/cli v2 discards a flag written after it
- [ ] `git diff` touches `README.md`, the fence tag at `docs/dsl-reference.md:26`, and the one test file

**Affected Files/Modules:**
- `README.md` — the quick-start model (`:25-62`) and the DCB illustration's fence (`:87`)
- `docs/dsl-reference.md` — the identifiers illustration's fence (`:26`)
- `internal/oracle/oracle_test.go` — a group beside "clean input" (`:24`)

**Patterns to Follow:**
- `internal/oracle/oracle_test.go:24` for the group shape and the `require.Empty(t, oracle.Check(...))`
  leaf; `oracle.Check(source, filename)` takes the text and the name, so the extracted block's document
  path and starting line go in as the filename and come back on every diagnostic
- `internal/importer/importer_test.go:77` for reading a repository file by relative path from a test
- The model content: `examples/all_patterns.emod` as Task 1 leaves it — the quick-start is the same
  domain one slice at a time, and a quick-start that contradicts the flagship example is worse than one
  that repeats it
- `tasks/learnings.md` "Never write emod source with `%q` — the language has no escape sequences", which
  binds the moment an extracted block is written to a temporary file rather than passed as text
- `tasks/learnings.md` "urfave/cli v2 discards every flag written after the file argument", which
  records `README.md:117-127` as documenting six invocations in exactly that shape — this task does not
  own fixing them, but must not add a seventh
- `~/.claude/rules/markdown-docs.md`

**Testable:** Yes — through `oracle.Check`, exported, over text this test reads from the two documents.

**Verification:** `go test -tags unit ./internal/oracle/...`; extracting the quick-start to
a file and running `go run ./cmd/emod validate` on it exits 0.

**Depends on:** 3

---

### Task 5: Sweep the repository for removed spellings

**Behavior:** no document in the repository presents either retired spelling as syntax to write today —
neither the automation-level trigger entry US-002 removed nor the trigger kind slot US-004 removed —
and the files that still contain the words contain them only as a description of what was removed or as
an archived record of the world before it.

**Acceptance Criteria:**
- [ ] Searching the repository, excluding `node_modules`, for the trigger keyword followed by an
      identifier ahead of a quoted name, and for the trigger keyword followed by a bare identifier,
      returns hits in only these files: `docs/proposals/triggers-and-automations-proposal.md`,
      `docs/proposals/completed/proposal.md`, `docs/proposals/completed/dcb-proposal.md`,
      `user-stories/triggers-and-automations.md`, `tasks/learnings.md`, and files under
      `tasks/completed/`
- [ ] Every surviving hit sits in a sentence, table row or example that names the spelling as removed or
      records the world before the removal — the feature proposal's Problem section, its removal table
      (`:111-121`), the Before half of its Worked Example (`:319-354`), its Versioning note (`:432`) and
      its Phase 1 summary (`:447`); the two completed proposals; the story file that states the criteria
      retiring the spellings; and the archived run records. None presents either spelling as current
      syntax
- [ ] The five live proposals under `docs/proposals/ai/` write the realigned forms:
      `01-nl-to-model-generation.md` at `:99`, `:288`, `:328` and `:354`,
      `02-model-import-reverse-engineering.md:327`, `06-mcp-server.md:339`,
      `09-bdd-test-generation.md:319`, and `10-conversational-viewer-editing.md:256-257` — each an emod
      snippet or a sentence naming the spellings a generator must emit
- [ ] Each rewritten snippet keeps the point its surrounding prose makes: the diff-shaped snippet at
      `10-conversational-viewer-editing.md:256-257` still shows one activation event being replaced by
      another, and `01-nl-to-model-generation.md:99` still names the constructs a generator must cover
- [ ] Any hit in a `.go`, `.js` or grammar-corpus file means US-004 is incomplete; each is corrected
      with the same one-line respelling US-004 would have made, and `go test -tags unit
      ./...` and `mise exec -- task test:grammar` both pass afterwards
- [ ] `go test -tags unit ./internal/cli/... ./internal/oracle/...` passes, so the two
      guards from Tasks 1 and 4 still hold over the swept tree
- [ ] No file under `tasks/completed/`, and no line of `tasks/learnings.md`, is edited: an archived
      record of a run is not a document that shows syntax
- [ ] No document gains a note about the sweep, a migration section, or any sentence describing what it
      used to say

**Affected Files/Modules:**
- `docs/proposals/ai/01-nl-to-model-generation.md`, `02-model-import-reverse-engineering.md`,
  `06-mcp-server.md`, `09-bdd-test-generation.md`, `10-conversational-viewer-editing.md`
- Whatever the search turns up that Tasks 1–4 and US-004 have not already settled

**Patterns to Follow:**
- The replacement table at `docs/proposals/triggers-and-automations-proposal.md:111-121` gives the
  three retired trigger spellings and what each becomes; section 3 (`:100-109`) gives the automation's
  activation rename
- `docs/proposals/specs-and-metadata-proposal.md:306` and `:449` are the two places a live proposal
  already writes the realigned forms, and are the model for how a proposal's snippets should read
- `~/.claude/rules/markdown-docs.md` — "the subject matter's past and present are content", which is
  what keeps the feature proposal's Before/After and the completed proposals out of this sweep, and
  "change summaries go elsewhere", which keeps the sweep itself out of every file it touches
- `tasks/learnings.md` "Keyword surfaces fan out past the lexer, parser and tree-sitter grammar" lists
  the hand-maintained surfaces no test cross-checks; the documents are the same class of surface and
  this sweep is the only thing that reaches them

**Testable:** No — the claim is a negative over documents, checked by a search with an enumerated
expected residue. The executable half of it is already carried by Tasks 1 and 4: a retired spelling
reaching an example or a fenced model fails those guards at the parser.

**Verification:** the two searches above return only the listed files;
`go test -tags unit ./...` passes; `git status --porcelain` lists no file
under `tasks/`.

**Depends on:** 1, 2, 3, 4

---

## Summary

**Five tasks**, ordered so that each one's output is executable by the time the next depends on it, and
so that the two guards land after the documents they guard are correct rather than before.

Task 1 comes first because the examples are the only artefact in this story a machine already checks —
and because its first criterion is where a missing US-004 is discovered, before any prose is written
against a grammar that does not exist. Tasks 2 and 3 split the reference by concern rather than by file
region: 2 rewrites what an author copies (the two pattern skeletons and the model in section 10), 3
rewrites what an author looks up (the cross-reference table and the activation prose) and closes with
the anchor reconciliation over the whole file, which only makes sense once both sets of edits are in.
Task 4 must follow all three, because the harness it adds cannot be green while the README quick-start
does not parse and the reference's section-10 model writes a kinded trigger — that ordering is forced,
not chosen. Task 5 is last because it is the only task whose criterion is a claim about the whole
repository.

**Story criteria coverage:**

| Story criterion | Task |
|---|---|
| `examples/all_patterns.emod` uses the kindless trigger and an automation that reads a view, and passes `emod validate` | 1 |
| A schedule-activated automation appears in the examples, together with the view it reads | 1 |
| `docs/dsl-reference.md` documents `on`, `every` and `reads` on automations, and the trigger without a kind | 2 (the pattern sections), 3 (the cross-reference table) |
| The reference states that a trigger is the human entry point and an automation is the processor, naming the chain each belongs to | 2 |
| The README quick-start uses the new forms | 4 |
| No document or example in the repository shows a removed spelling | 5, with the executable half in 1 and 4 |

Carried along, not stated by the story: the guard that runs `emod validate` over `examples/` (1), which
nothing does today; the guard that runs every fenced emod model in the README and the reference through
the same pipeline (4), which is the durable answer to the reference having documented a rejected
spelling since US-002; the correction of the cross-reference table's claim that a trigger's and a
translation's `reads` are resolved (3); giving the event-activated automation in the example a `reads`
so US-008's warning does not break the example's own criterion (1); and making the README quick-start a
model that parses at all (4).

**Left to other stories:** the palette's row in the reference (US-007 — coordinate on this file, and see
the anchor note above); hover text describing `automation` (US-009); highlighting (US-010); and the
mechanical migration of every `.emod` file, Go and JS fixture and grammar corpus case (US-004, which
this story assumes has landed and Task 5 re-checks).

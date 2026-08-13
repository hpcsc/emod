# US-002: Resolve a trigger's and a translation's `reads`

## Story Reference

`user-stories/lint-unread-view.md` → **US-002: Resolve a trigger's and a translation's `reads`**
(second of two stories in "Flagging a Read Model Nothing Reads"; US-001 is delivered and archived at
`tasks/completed/us-001-flag-a-view-nothing-reads.md`).

**In scope:** widening `referenceDiagnostics` (`internal/validator/validator.go:397`) so a `trigger`'s
`reads` and a `translation`'s resolve against `modelIndex.viewNames` the way an `automation`'s already
does, each reported at the value's own `ReadsPos` with the message the automation's check already
produces; keeping `view/never-read` from adding a second diagnostic for the same mistake, so a
misspelled view name yields one finding at the `reads` rather than one there and one at the view;
the reference's account of the asymmetry, which stops being true; and the `tasks/learnings.md` entry
that records the asymmetry as a constraint. Carried along because the check is what makes them fail:
the five shared fixtures and the two checked-in `all_patterns.emod` files that name a view no slice
declares, plus the expected values that transcribe them and the two picture-and-document guards that
used a shared fixture as their example of an unresolvable name.

## Boundaries

**Out of scope** — carried verbatim from the story's Non-Goals:

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

- **Inventing a view anywhere.** Every model that names a view no slice declares is repaired by
  pointing its existing `reads` at a view the *same model already declares*, which is the precedent
  `tasks/learnings.md` records from US-008. No fixture, example or testdata file gains a `view` block,
  so no model gains a construct, and no diagram box, export object, glossary term or `*FormattedEmod`
  constant moves for a reason other than the changed `reads` value itself
- **Near-miss or "did you mean" suggestions.** The diagnostic names the unresolved view and stops
  there, exactly as an automation's does today. Nothing here computes an edit distance between an
  unresolved name and a declared one, and the `view/never-read` suppression deliberately does not try
  to guess which view the typo meant
- **Changing `view/never-read`'s severity, its guard, its position or its message.** US-001 shipped it
  as a `warning()` at the view's `NamePos`, fired only once the model states at least one `reads`. All
  four stay. The rule gains one further reason to stay silent and one sentence in its
  `ruleDescriptions` entry; nothing else about it moves
- **Any change to how the other three view references resolve.** An automation's `reads`, a spec's
  `then view <Name>` and the diagram's view lookups are already checked and are not touched
- **Sorting `validator.Validate`'s output.** Nothing in the repository sorts validator diagnostics into
  source order — `referenceDiagnostics` already reports an automation's `target context` ahead of its
  `on`, which is not the order they are written — and this story does not start. The order the new
  checks emit in is pinned by a criterion so it is deliberate rather than accidental, but no sort is
  added
- **The AST, the grammar, the formatter, the importer, the exports and the editor surfaces.** No field
  is added, no serialized key moves, no keyword changes. `internal/ast`, `internal/parser`,
  `internal/formatter`, `internal/export`, `internal/importer`, `internal/cue`, `internal/glossary`,
  `internal/lsp`, `internal/viewer`, `internal/wasm` and `editors/` hold no production edit — Tasks 2
  and 3 touch *test* files in `internal/export` and `internal/diagram` only, because those files
  transcribe a fixture they change
- **The `.emod` files under `editors/`.** `editors/tree-sitter-emod/test/highlight/unreserved-keywords.emod`
  and `editors/vscode/test/scopes/{activations,dcb,wire-type}.emod` each name a view no slice declares.
  Nothing parses, validates or lints them — they are grammar and scope assertions driven by the
  tree-sitter and `vscode-tmgrammar-test` runners — so they are left as written
- **`docs/proposals/`.** Seven files there fence emod snippets and nothing validates any of them;
  `tasks/learnings.md` records that surface as uncovered. The story's criterion names the reference,
  and the reference is the only document edited

**Deferred:**

- Scoping the `view/never-read` adoption guard per context rather than per model — the story's second
  open question, untouched by US-001 and untouched here
- A rule about the other end of the relationship (whether a view's `subscribes` supply the fields its
  consumer reads) — the story lists it as a non-goal and nothing here begins it
- Sweeping `docs/proposals/ai/` for emod snippets that name a view no slice declares. Those snippets
  are now wrong in a way `emod validate` would catch if anything ran it over them; giving that surface
  a harness is its own story

---

## Open Questions, Decided

The story's Open Questions are shared with US-001 and mostly settled by US-001 shipping. Two bear on
this story, and both are decided here.

### "Does a view read *only* by a trigger deserve different treatment from one read by an automation?"

**Closed by this story: no — after it, the three spellings of `reads` are equivalent.** The question
existed only because a trigger's `reads` was unresolved, so it could not reliably prove consumption:
`reads CaseWorkspacveView` counted as a consumer of a view that does not exist. Once every spelling
resolves, a trigger naming a declared view proves consumption exactly as an automation's does, and
`view/never-read` goes on treating all three alike, which is what US-001's second acceptance criterion
already requires.

### How `view/never-read` avoids a second diagnostic for one mistake (story criterion 4)

**Decided: the rule stays silent for the whole model while any `reads` names a view no slice declares
— option (a).**

The interaction is real and measured. `checkNeverRead` (`internal/linter/linter.go:634`) reports a view
whose name is absent from the model-wide `readViews` set built at `linter.go:65-80`. A trigger writing
`reads CaseWorkspacveView` puts the typo into that set and leaves `CaseWorkspaceView` out of it, so
after Task 4 the model reports both the unresolved reference at the trigger and `view/never-read` at
the view — two diagnostics, one mistake, and the second one points twelve lines away from the cause.
Note this is not new with US-002: an *automation* misspelling a view name already produces both today,
so the fix is uniform across the three spellings rather than special-cased to the two being widened.

**Does `internal/linter` have the declared-view set it would need?** No. `internal/linter/linter.go`
touches `slice.Views` in exactly one place — the per-slice loop at `:194` that runs `checkViewNaming`,
`checkGodView` and `checkNeverRead`. The set is new, and it is built the way `readViews` is: one
`model.AllSlices()` pass, which covers both slice homes (`tasks/learnings.md`, "A slice has two homes,
and much of the repo still walks only one").

**The narrower options were considered and rejected.** Scoping the suppression to the context holding
the unresolved `reads` fails on the rule's own model-wide resolution — US-001's third criterion says a
view in one aggregate read by an automation in another is read, so a typo in one context can orphan a
view in another and a context-scoped suppression would misreport it. Suppressing only views whose name
is a near-miss of the unresolved one needs an edit-distance heuristic the repository has nowhere else,
and it still double-reports the case where a `reads` names a view that was renamed or deleted outright
rather than mistyped. Keeping the report but changing its wording ("…or a `reads` elsewhere names a
view no slice declares") still emits one diagnostic per view, which is what the criterion forbids.
Excluding the unresolved name from `readViews` instead of suppressing the rule does not help at all:
the real view is still absent from the set and still reports.

**The cost, stated.** A genuinely unread view elsewhere in a large model goes unreported until the
unrelated typo is fixed. That is acceptable because the model does not ship in that state: the
unresolved reference is a hard error, so `emod validate` and `emod lint` both exit non-zero until it is
corrected, and the never-read finding is deferred to the next run rather than lost.

**Correction to the brief: `emod lint` does *not* run without the validator, so nothing goes quiet.**
`RunLint` (`internal/cli/lint.go:118`) calls `oracle.Check`, which is `oracle.Run` — lexer, parser,
**validator** and linter — the same call `RunValidate` makes (`internal/cli/validate.go:27`). Verified
by reading `internal/oracle/oracle.go:26-36`. `linter.Lint` has exactly one production caller in the
repository, `oracle.Run`, so every frontend — CLI, LSP (`internal/lsp/server.go:391`), wasm
(`internal/wasm/pipeline.go:44`) — sees the validator's diagnostics beside the linter's. There is no
surface on which the suppression can leave an author with nothing to read: wherever
`view/never-read` falls silent, the unresolved-reference error is reported in the same list, at the
`reads` the author mistyped. `emod lint` and `emod validate` differ only in output shaping and exit
codes, not in which checks run.

---

## Codebase Context

**The validator's reference checks.** `referenceDiagnostics(slice, index)`
(`internal/validator/validator.go:397-424`) is one function per slice. Its automation loop makes four
calls to `appendUndeclaredRef(diags, kind, name, pos, declared)` (`:454`), which carries the empty-name
guard, the map lookup and the `errorAt` construction, so a new reference check is one line rather than
a fifth copy of that shape. `index.viewNames` is filled in `collect(slice)` (`:102-104`) from every
slice `model.AllSlices()` yields, so it covers both slice homes and resolution is model-wide.
`errorAt` (`:177`) produces a `diagnostic.Error` with no `RuleName`. A two-line comment at `:403-404`
records that a trigger's and a translation's `reads` stay unchecked "on purpose: existing models name
views in them that no slice declares" — the statement this story falsifies.

**The AST carries what the check needs.** `ast.Trigger` has `Reads string` / `ReadsPos Position`
(`internal/ast/ast.go:228-229`) and `ast.Translation` the same (`:275-276`). `Slice.Trigger` is a
pointer and may be nil; `Slice.Translations` is a list. Nothing new is needed in the AST.

**The linter.** `Lint(model)` (`internal/linter/linter.go:50-141`) builds its model-wide facts before
the per-context walk: `flowCount` (`:57-63`), `readViews` (`:65-80`, filled from all three spellings),
then the spec trio (`:82-95`). They reach the per-slice checks as trailing parameters of `checkSlice`
(`:157`). `checkNeverRead` (`:634-648`) is two guards and a `warning()`: an empty `readViews` means the
model has not adopted the concept, and a hit means the view has a reader. `slice.Views` is read in one
place only, `checkSlice`'s view loop (`:194-204`).

**Measured blast radius of the widening.** The check was applied as a throwaway probe —
`appendUndeclaredRef` for `slice.Trigger.Reads` at `Trigger.ReadsPos` and `tr.Reads` at `tr.ReadsPos`
— then `go test ./...` was run and the probe reverted. Two packages fail, `internal/oracle` and
`internal/cli`, and every failure traces to one of **five** shared fixtures. This is one more pair than
the brief listed: `AutomationReadsLibraryLending` and `AutomationScheduleLibraryLending` fail too, and
they fail through their *exact-line* oracle leaves rather than through a clean-input `require.Empty`,
which is why a scan for zero-diagnostic assertions misses them.

| Fixture (`internal/test/fixtures.go`) | Construct | Names | Model declares | Fails |
|---|---|---|---|---|
| `HotelReservation` (`:13`) | trigger (`:23`), translation (`:83`) | `AvailableRoomsView`, `BookingWebhookView` | `ReservationsView` | oracle "clean input"; ~40 `internal/cli` subtests across `TestValidate`, `TestLint`, `TestExport`, `TestDiagram`, `TestArgs` (it is `validEmod`, `internal/cli/validate_test.go:20`) |
| `DescribedHotelReservation` (`:101`) | trigger (`:119`), translation (`:189`) | the same two | `ReservationsView` | oracle "clean input"; the `describedEmod` CLI subtests |
| `KeywordFieldSearchCatalog` (`:210`) | translation (`:290`) | `VendorSearchWebhookView` | `SavedSearchesView` (`:251`) | oracle "clean input" (`keyword-fields.emod:81`); the `keywordFieldEmod` CLI subtests |
| `AutomationReadsLibraryLending` (`:1043`) | trigger (`:1053`) | `AvailableCopiesView` | `MemberLoansView` (`:1075`), `DeskOccupancyView` (`:1179`) | oracle "automations reading no view" — its transcribed list gains `automation-reads.emod:11: view "AvailableCopiesView" does not exist` |
| `AutomationScheduleLibraryLending` (`:1409`) | trigger (`:1419`) | `AvailableCopiesView` | `MemberLoansView` (`:1441`), `DeskOccupancyView` (`:1549`) | the same, at `automation-schedule.emod:11` |

Two checked-in `.emod` files fail: `examples/all_patterns.emod:108` and
`internal/parser/testdata/all_patterns.emod:108`, both the `translation BookingComImport` reading
`BookingComWebhookView`, which neither declares. Each is guarded — the first by `examplePaths`
(`internal/cli/validate_test.go:52`), the second by the parser-fixture leaf (`:37-49`). Both files
declare `AvailableRoomsView` at `:43`. `examples/error_diagnostics_test.emod` and
`internal/parser/testdata/invalid.emod` are authored to fail and are unaffected.

Nothing else in the tree fails. Every `.emod` under `examples/` and `internal/parser/testdata/` was
scanned, and every fenced ` ```emod ` block in `README.md` and `docs/dsl-reference.md` — the seven
blocks `internal/oracle/oracle_test.go`'s "documented models" leaves run — names only views its own
block declares. `docs/dsl-reference.md:496` (`reads DeskClaims`) sits under a plain ` ``` ` fence, not
an emod one, so it is deliberately outside the harness.

**What holds the fixtures' `reads` values as expected text.** Repairing them moves exactly these:

- `internal/cli/fmt_test.go:223` — the `reads VendorSearchWebhookView` line inside
  `keywordFieldFormattedEmod` (`:136`), the canonical constant that restates
  `KeywordFieldSearchCatalog` line for line. `describedFormattedEmod` (`:50`) is a smaller model and
  states no `reads`; no other `*FormattedEmod` constant transcribes any of the five fixtures
- `internal/formatter/formatter_test.go:1440` — `require.Contains(t, formattedTwin, "reads AvailableCopiesView")`
- `internal/formatter/formatter_test.go:1462`, `:1469`, `:1477` — the `triggerReads` fields of the
  `AutomationReadsLibraryLendingModel`, `HotelReservationModel` and
  `AutomationScheduleLibraryLendingModel` rows of the round-trip table
- `internal/export/export_test.go:4595` — `readingTrigger := map[string]string{"Borrow Copy": "AvailableCopiesView"}`
- `internal/parser/integration_test.go:257` — `Reads: "BookingComWebhookView"` in the expected
  `ast.Translation` for the testdata example. This file is `//go:build integration`, so `go test ./...`
  does not reach it; `task test:integration` does

**Two guards use a shared fixture as their example of an unresolvable name, and both already have a
better subject.** `internal/diagram/contract_test.go:684` ("a trigger whose reads names a view no
slice declares is drawn no arrow beside an automation whose does not") opens with
`require.Equal(t, []string{"AvailableCopiesView"}, test.DeclaredTriggerReads(reading), "its trigger has
to name a view no slice declares, or the silence below says nothing")`. Its claim is already made in
full, sixty lines above at `:628`, over `mixedReadsModel()` (`:780`) — a model built in the test whose
`Booking Page` trigger and `ExpireReservation` automation both read `DeskWaitlistView`, which no slice
declares, beside a trigger and an automation reading the declared `DeskOccupancyView`, with
`require.Len(t, drawn, 2)` pinning that nothing else is drawn. The export side has the same pair:
`internal/export/export_test.go:3692` asserts `require.Empty(t, viewsReadBy(readingDoc, "trigger"),
"the trigger of this model reads AvailableCopiesView, which no slice declares")` inside a subtest about
automations, while the bespoke model at `:2837-2887` already states the whole claim
("`AvailableCopiesView` resolves to no node"). The behaviour stays — the picture and document exporters
render any `*ast.Model`, including one the importer builds from a viewer save, and
`tasks/learnings.md` records that a draw.io cell id must not be allocated for an edge that is then
skipped — but a *shared fixture* can no longer be its example.

**Existing `view/never-read` leaves.** `internal/linter/linter_test.go:3744-4025` is the rule's group,
eight leaves, models built in Go with positions set by hand. Seven of them name only views their model
declares and are unaffected by the suppression. The eighth is not: "one reads answers for every
declaration of that name, and no reads reports each of them" → "read by nothing, with the model stating
a reads elsewhere" (`:4016`) passes `&ast.Trigger{Reads: "OverdueLoansView"}` to a model that declares
`MemberLoansView` twice and `OverdueLoansView` never — it uses an *unresolved* `reads` purely to open
the adoption guard, which is exactly the state the suppression turns off.
`internal/cli/lint_test.go`'s `viewNeverReadEmod` (`:168`) is safe: its automation reads
`OverdueLoansView`, which the fixture declares at `:191`.

**The reference.** Three statements become false, plus one that gains a clause:

- `:811` — "All references use unqualified names. `emod validate` resolves them, except for two of the
  three spellings of `reads`."
- `:823` — the paragraph beneath the cross-reference table: "Of the three constructs that spell
  `reads`, only an automation's is resolved… A trigger's `reads` and a translation's are recorded and
  left unchecked, so either may name a view no slice declares."
- `:383` — the Automation Pattern's `reads` bullet, whose last sentence draws the same contrast
- `:358` — the `### View Pattern` bullet for `view/never-read`, which should say the rule goes quiet
  while a `reads` is unresolved

The cross-reference table row for `view <Name>` (`:819`) already lists all three spellings and needs no
edit. Two constraints on editing the file: every heading is `## <n>. <Title>` and four in-document
links cite those numbers, while fourteen more cite `###` sub-heading slugs — so heading *text* must be
held fixed and no numbered section added or reordered; and section 13's `Diagram Palette` table is
machine-read by `dslReferencePalette` (`internal/diagram/contract_test.go:1349`).

**Test targets.** `task test` runs `test:unit` (`go test -tags unit`), `test:integration`
(`go test -tags integration`), `test:viewer` (vitest), `test:grammar` and `test:vscode`. Nothing in
this story touches the viewer, the grammar or the VS Code surfaces; the first two are the targets that
must stay green at every commit.

---

## Tasks

### Task 1: Point the pipeline fixtures' trigger and translation `reads` at a view their own model declares

**Behavior:** the three fixtures the pipeline runs as models it accepts — `test.HotelReservation`,
`test.DescribedHotelReservation` and `test.KeywordFieldSearchCatalog` — name only views their own
model declares. What each model contains, how it formats and what it exports is otherwise unchanged.

**Acceptance Criteria:**

- [x] The trigger (`internal/test/fixtures.go:23`) and the translation (`:83`) of
      `test.HotelReservation` both read `ReservationsView`, the view its `slice "View Reservations"`
      declares, in place of `AvailableRoomsView` and `BookingWebhookView`, which it declares nowhere
- [x] The same two lines of `test.DescribedHotelReservation` (`:119`, `:189`) read `ReservationsView`,
      the fixture keeping every description it states — `test.DeclaredDescriptions` over the parsed
      model still equals `test.DescribedHotelReservationDescriptions` unedited
- [x] The translation of `test.KeywordFieldSearchCatalog` (`:290`) reads `SavedSearchesView`, the view
      its `slice "Browse Saved Searches"` declares (`:251`), in place of `VendorSearchWebhookView`
- [x] `rg -n 'AvailableRoomsView|BookingWebhookView|VendorSearchWebhookView' internal/test/fixtures.go`
      prints nothing
- [x] `test.DeclaredTriggerReads(test.HotelReservationModel(t))` and the same over
      `test.DescribedHotelReservationModel(t)` each read back `[]string{"ReservationsView"}` — a getter
      answering `nil` is what this criterion exists to rule out
- [x] `oracle.Check` returns nothing for all three, exactly as it does today, and no leaf in
      `internal/oracle/oracle_test.go` is edited
- [x] The only expected values that move are the ones restating a changed fixture's own text: the
      `reads` line inside `keywordFieldFormattedEmod` (`internal/cli/fmt_test.go:223`) and the
      `triggerReads` field of the `HotelReservationModel` row in the formatter round-trip table
      (`internal/formatter/formatter_test.go:1469`). No other golden, `*FormattedEmod` constant or
      transcribed name list moves
- [x] `git diff --stat` names exactly three files: `internal/test/fixtures.go`,
      `internal/cli/fmt_test.go`, `internal/formatter/formatter_test.go`
- [x] `mise exec -- task test:unit` and `mise exec -- task test:integration` are green

**Affected Files/Modules:**

- `internal/test/fixtures.go` — five `reads` lines: `:23`, `:83`, `:119`, `:189`, `:290`
- `internal/cli/fmt_test.go` — the `reads` line at `:223` inside `keywordFieldFormattedEmod` (`:136`)
- `internal/formatter/formatter_test.go` — the `triggerReads` field of one round-trip table row
  (`:1467-1471`)

**Patterns to Follow:**

- The repair shape is US-008's, recorded in `tasks/learnings.md` under "A lint warning fails `emod
  validate`, so a new rule sweeps every checked-in model before it lands": point the existing `reads`
  at a view the model already declares rather than inventing one
- `tasks/learnings.md` "A 'no expected constant moves' criterion is unsatisfiable when the task edits a
  shared fixture" — the two expected values above are named individually for that reason, and
  `keywordFieldFormattedEmod` is the canonical constant that learning cites by name
- `tasks/learnings.md` "Shared fixtures come in an unfeatured/featured pair, guarded by a walk that must
  be extended" — `HotelReservation` and `DescribedHotelReservation` are that pair, so the same two
  lines change in both and the described twin keeps every description it states
- `tasks/learnings.md` "A `Declared…` getter answers `nil` for a fixture that declares none of the
  construct" — the `DeclaredTriggerReads` criterion above is written against a non-empty expected list
  for that reason
- `tasks/learnings.md` "`emod fmt <file>` writes in place, so a receipt run dirties the working tree" —
  copy to a temp path or `git checkout --` afterwards if `emod fmt` is run to derive the canonical text

**Testable:** No — no behaviour changes. All three models answer `oracle.Check`, `cli.RunValidate`,
`cli.RunFmt` and the exporters exactly as they did before, and the leaves that assert so already exist
and stay unedited. The task exists so Task 4 can land without turning them red.

**Verification:** `mise exec -- task test:unit`; `mise exec -- task test:integration`;
`rg -n 'AvailableRoomsView|BookingWebhookView|VendorSearchWebhookView' internal/test/fixtures.go`
prints nothing; `git diff --stat` lists the three files above and no others.

**Depends on:** None

---

### Task 2: Point the library-lending fixtures' trigger `reads` at a declared view, and leave the unresolvable-`reads` guards a model of their own

**Behavior:** `test.AutomationReadsLibraryLending` and `test.AutomationScheduleLibraryLending` name
only views their own model declares. The two assertions that used those fixtures as their example of a
`reads` naming a view no slice declares keep their claim, on a model built for it.

**Acceptance Criteria:**

- [x] The trigger of `test.AutomationReadsLibraryLending` (`internal/test/fixtures.go:1053`) and of
      `test.AutomationScheduleLibraryLending` (`:1419`) reads `MemberLoansView`, the view each model's
      `slice "Review Member Loans"` declares, in place of `AvailableCopiesView`, which neither declares
- [x] `rg -n 'AvailableCopiesView' internal/test/fixtures.go` prints nothing
- [x] `test.DeclaredTriggerReads` over both parsed models reads back `[]string{"MemberLoansView"}`
- [x] `oracle.Check` over both returns exactly the lines `internal/oracle/oracle_test.go:108-110` and
      `:124-128` transcribe today, unchanged down to their line numbers, and no leaf in that file is
      edited — the repair replaces a name on a line rather than moving one
- [x] `internal/diagram` still asserts that a `reads` naming a view no slice declares is drawn no
      arrow, over `mixedReadsModel()` (`internal/diagram/contract_test.go:780`), and no leaf in that
      package rests that claim on a shared fixture. Stated as behaviour rather than as
      `rg -n 'AvailableCopiesView' internal/diagram` printing nothing, because `wiredSlice()`
      (`internal/diagram/graph_test.go:19`) keeps the name in a hand-built slice that pins what
      `SliceEdges` derives before any drawer resolves it — it rests no claim on a shared fixture, and
      renaming it would churn a test guarding a different layer
- [x] `internal/export` still asserts that such a name reaches no diagram node and draws no edge, over
      the model built at `internal/export/export_test.go:2837-2887`, and the trailing assertion of
      "the view an automation reads reaches its node and draws an edge of its own…" (`:3692`) states
      the trigger's own edge instead of its absence, its subtest name no longer claiming "while a name
      no slice declares draws none"
- [x] The only expected values that move are the ones restating a changed fixture's own text:
      `require.Contains(t, formattedTwin, "reads AvailableCopiesView")`
      (`internal/formatter/formatter_test.go:1440`), the `triggerReads` fields of the
      `AutomationReadsLibraryLendingModel` and `AutomationScheduleLibraryLendingModel` rows (`:1462`,
      `:1477`), `readingTrigger` (`internal/export/export_test.go:4595`), the cursor anchor of the
      trigger row in the `reads` completion table (`internal/lsp/completer_test.go:267`), the
      reference list of "cursor on a view declaration lists every construct reading it…"
      (`internal/lsp/references_test.go:264`), and the two assertions named in the criteria above. No
      `*FormattedEmod` constant and no transcribed `…ViewNames`, `…ActivationEvents` or `…Schedules`
      list moves
- [x] `git diff --stat` names exactly six files: `internal/test/fixtures.go`,
      `internal/formatter/formatter_test.go`, `internal/export/export_test.go`,
      `internal/diagram/contract_test.go`, `internal/lsp/completer_test.go` and
      `internal/lsp/references_test.go`. The two `internal/lsp` files were missed when this list was
      written: both consume `AutomationReadsLibraryLending`, and once its trigger reads a *declared*
      view the completer's cursor anchor no longer exists in the document and find-references
      correctly gains the trigger's site
- [x] `mise exec -- task test:unit` and `mise exec -- task test:integration` are green

**Affected Files/Modules:**

- `internal/test/fixtures.go` — two `reads` lines: `:1053`, `:1419`
- `internal/formatter/formatter_test.go` — `:1440`, and the `triggerReads` fields at `:1462` and `:1477`
- `internal/export/export_test.go` — the trailing assertion and subtest name at `:3674-3693`, and
  `readingTrigger` at `:4595`
- `internal/diagram/contract_test.go` — the leaf at `:684-699`

**Patterns to Follow:**

- `mixedReadsModel()` (`internal/diagram/contract_test.go:780`) and the model at
  `internal/export/export_test.go:2837-2887` are what a claim about an unresolvable `reads` is asserted
  over from now on — both build their model in the test file, both put an unresolved reader beside a
  resolving one, and the diagram one pins `require.Len(t, drawn, 2)` so nothing else can be drawn
- `tasks/learnings.md` "Strengthening a test to a whole-sequence `require.Equal` means deleting the
  subtest it subsumes" — read the leaf at `internal/diagram/contract_test.go:684` against the one at
  `:628` before deciding whether it has anything left to say; a parallel change-detector for one
  behaviour fails twice for one regression
- `tasks/learnings.md` "A task's change-set assertion must name every file its own patterns require it
  to change" — the change-set criterion above names all four files, including the two test files in
  packages this story otherwise leaves alone
- `tasks/learnings.md` "A tested, defensible improvement found on the way is still a separate commit" —
  the filter is whether this task's criteria can pass without the edit. The two guard sites cannot,
  because their models are the fixtures being repaired; anything else noticed in those files is a
  separate commit
- `tasks/learnings.md` "Allocate a draw.io cell id only once the cell is certain to be written" — why
  the guard being relocated is permanent behaviour rather than a leftover, and must not simply be
  deleted without its claim landing somewhere

**Testable:** No — the exporters, the formatter and the pipeline answer for both models exactly as they
did; only which view the trigger names changes. The relocated assertions are existing coverage moving,
not new behaviour.

**Verification:** `mise exec -- task test:unit`; `mise exec -- task test:integration`;
`rg -n 'AvailableCopiesView' internal/test internal/diagram` prints nothing; `git diff --stat` lists
the four files above and no others.

**Depends on:** None

---

### Task 3: Point the flagship example's translation at a view it declares

**Behavior:** `emod validate` accepts both checked-in `all_patterns.emod` files with every `reads`
resolving, so the example a reader copies from demonstrates a translation reading a view the model
declares rather than a name nothing defines.

**Acceptance Criteria:**

- [x] The `translation BookingComImport` in `examples/all_patterns.emod:108` and in
      `internal/parser/testdata/all_patterns.emod:108` reads `AvailableRoomsView`, the view each file's
      `slice "View Available Rooms"` declares at `:43`, in place of `BookingComWebhookView`, which
      neither declares
- [x] No `.emod` file the pipeline accepts still names `BookingComWebhookView`. Stated this way rather
      than as `rg -n 'BookingComWebhookView' examples internal/parser` printing nothing, because
      `internal/parser/parser_test.go:3135` keeps the name in an inline source asserting the parser
      *records* a translation's `reads` (`:3158`) — the parser resolves nothing, so that leaf is
      unaffected by the widening and renaming it would churn a test guarding a different layer.
      `docs/proposals/completed/proposal.md:97` keeps it too, which `tasks/learnings.md` requires: a
      completed proposal records the world before a change and is never swept
- [x] `diff examples/all_patterns.emod internal/parser/testdata/all_patterns.emod` still differs only by
      the example's two trailing slices (its lines 123-161, "Slice 6" and "Slice 7") — the two files
      stay in step everywhere else
- [x] `cli.RunValidate` returns no error for either path, which the `examplePaths` leaf
      (`internal/cli/validate_test.go:51-62`) and the parser-fixture leaf (`:37-49`) already assert;
      neither leaf is edited
- [x] The expected `Reads` on the translation in `internal/parser/integration_test.go:257` names the
      view the file now reads, and nothing else in that expected `ast.Translation` — its name, external
      system, command or nested event — moves
- [x] `git diff --stat` names exactly three files: `examples/all_patterns.emod`,
      `internal/parser/testdata/all_patterns.emod`, `internal/parser/integration_test.go`
- [x] `mise exec -- task test:unit` and `mise exec -- task test:integration` are green

**Accepted, not fixed — the fixture's three `reads` slots no longer carry three distinct names.**
Both corpora declare fewer views than they state `reads`, so once the translation stops naming a view
nothing declares it must share a name with the trigger or the automation. An audit round raised the
consequence: a stub that ignores a translation's own `reads` and emits the constant
`"AvailableRoomsView"` now passes the formatter round-trip and the parser integration suite over these
two files. Declaring a `BookingComWebhookView` to restore three names was tried and reverted — the only
event available to feed it is `ExternalReservationImported`, which the very translation that reads it
emits, so the flagship example gained a cycle and a read model fed by its own consumer, contradicting
the Translation shape `docs/dsl-reference.md` documents. A low-severity gap in one fixture's
discriminating power is the better trade, particularly as `internal/cli`'s `TestFmt`, the
`formats_translation_block_with_nested_event` leaf and five other round-trip fixtures still catch that
stub. Giving the example an honest inbound view needs an event carrying `source external`, which no
example declares today; that is its own change.

**Affected Files/Modules:**

- `examples/all_patterns.emod` — the `reads` line at `:108`
- `internal/parser/testdata/all_patterns.emod` — the `reads` line at `:108`
- `internal/parser/integration_test.go` — `Reads` in the expected translation at `:257`

**Patterns to Follow:**

- The file's own idiom for a resolving `reads` is its `automation ConfirmationEmailReactor` (`:86-91`)
  reading `PendingConfirmationsView` and its `UnconfirmedReservationExpirer` (`:140-144`) reading
  `UnconfirmedReservationsView`. The translation joins them by naming a view the file already declares;
  no eighth slice and no new `view` block is added, which is what keeps `require.Len(t, agg.Slices, 5)`
  (`internal/parser/integration_test.go:111`) and the slice-comment table at `:113-126` unedited
- `tasks/learnings.md` "`examples/` is enumerated by its guard, and `_test.emod` means authored to
  fail" — `examples/error_diagnostics_test.emod` keeps the suffix and stays authored to fail; it is not
  edited and its `demonstrated` entry (`internal/cli/validate_test.go:66-69`) is not touched
- `internal/parser/integration_test.go` is `//go:build integration`, so `go test ./...` does not reach
  it — `tasks/learnings.md` records that shape under "A test that shells out to a CLI runs with
  `-count=1`"; here the consequence is simply that `task test:integration` must be run to see this
  task's own expected value pass

**Testable:** No — both files validate today and validate after; the leaves that assert so already
exist and are not edited. The task exists so Task 4 can land without turning them red.

**Verification:** `mise exec -- task test:unit`; `mise exec -- task test:integration`;
`rg -n 'BookingComWebhookView' examples internal/parser` prints nothing;
`go run ./cmd/emod validate examples/all_patterns.emod` exits 0.

**Depends on:** None

---

### Task 4: Resolve a trigger's and a translation's `reads` at `emod validate`

**Behavior:** `emod validate` reports a `trigger`'s `reads` and a `translation`'s `reads` naming a view
no slice declares, each at the value's own position, with the message an `automation`'s `reads` already
produces — so a misspelled view name is reported where the author wrote it, and the three constructs
that spell `reads` read the same.

**Acceptance Criteria:**

- [ ] A model whose `trigger` reads a name no slice declares produces exactly one diagnostic: message
      `view "<Name>" does not exist`, at the trigger's `ReadsPos` filename, line **and column**,
      severity `diagnostic.Error`, and an empty `RuleName`
- [ ] The same holds for a `translation`'s `reads`, at its own `ReadsPos`
- [ ] A model whose trigger, automation and translation each read one same undeclared name reports
      three diagnostics whose messages are byte-identical, one per construct at that construct's own
      `ReadsPos` — asserted as one `require.Equal` over the reported lines, in the order the checks
      emit: the trigger's, then the automation's, then the translation's
- [ ] A trigger's `reads` and a translation's naming a view the model declares produce no diagnostic,
      whether that view is declared in the same slice, in another aggregate, in another context, or on
      a `mode dcb` context's own slice — resolution is model-wide, matching an automation's
- [ ] A slice declaring no trigger, a trigger stating no `reads`, and a translation stating no `reads`
      each produce no diagnostic
- [ ] A name declared only as a command or only as an event does not resolve as a view for either
      construct, reported with the same `view "<Name>" does not exist` wording
- [ ] `cli.RunValidate` over a file whose trigger misspells a view the model declares returns an error
      whose `Error()` equals the whole formatted line — the path, the line the `reads` is written on,
      and `view "<Name>" does not exist`
- [ ] The comment at `internal/validator/validator.go:403-404` recording that a trigger's and a
      translation's `reads` stay unchecked is gone, because it is no longer true
- [ ] No fixture, example, testdata file or golden is edited by this task: `git diff --stat` names
      exactly `internal/validator/validator.go`, `internal/validator/validator_test.go` and
      `internal/cli/validate_test.go`
- [ ] `mise exec -- task test:unit` and `mise exec -- task test:integration` are green

**Affected Files/Modules:**

- `internal/validator/validator.go` — `referenceDiagnostics` (`:397-424`): one check before the
  automation loop for `slice.Trigger`, one inside the translations loop (`:407-409`), and the comment
  at `:403-404` removed
- `internal/validator/validator_test.go` — new subtests in the existing `"view references"` group
  (`:890-1089`)
- `internal/cli/validate_test.go` — one leaf beside the existing view-reference leaf at `:502`

**Patterns to Follow:**

- `tasks/learnings.md` "Only an automation's `reads` resolves; a trigger's and a translation's must stay
  unchecked" — this story is what overturns that entry. Read it for the mechanics it records: the five
  reference checks funnel through `appendUndeclaredRef(diags, kind, name, pos, declared)`
  (`internal/validator/validator.go:454`), which carries the empty-name guard, so each new check is one
  line and not a copy of the guard/lookup/`errorAt` shape; the diagnostic sits on the *value's*
  position, not the construct's name; and it carries no `RuleName`
- The subtests to copy, wholesale, are the automation's four in the same group: "a view no slice
  declares is reported on the reads entry, not on the automation name" (`:938`, which asserts the whole
  formatted line, the column, the severity and the empty rule name in four calls), "an automation on a
  context's own slice is checked like one inside an aggregate" (`:974`), "an automation without a reads
  entry produces no diagnostic while its sibling reading an undeclared view is reported" (`:1013`) and
  "a name declared only as an event does not resolve as a view" (`:1050`)
- Placement: a slice's `trigger` is its first-written entry, so its check belongs ahead of the
  automation loop; a translation writes `reads` above `command`, so its check belongs above the
  existing `tr.Command` call. Note the automation loop is already not in source order (`target context`
  before `on`), so no sort is added and nothing else is reordered
- `tasks/learnings.md` "`RuleName` marks a diagnostic `emod lint --explain` can describe" — these are
  hard errors no configuration silences, so the field stays empty and nothing is added to
  `internal/linter/descriptions.go` by this task
- `tasks/learnings.md` "A second `require.Contains` on one message is often shadowed by the first" and
  "A rule whose message branches on model state is pinned by whole formatted lines" — assert the
  diagnostic as a whole formatted line (`reportedLines`, or `diags[0].String()`), not as stacked
  `Contains` calls on a message that already contains the view's name
- `tasks/learnings.md` "CLI diagnostic tests must assert the distinguishing message text" — the
  `internal/cli` leaf asserts the message, not only the path and a line number
- `tasks/learnings.md` "A slice has two homes, and much of the repo still walks only one" —
  `referenceDiagnostics` runs over `index.slices`, which is `model.AllSlices()`, so both homes are
  already covered; the `mode dcb` criterion above is what proves the new checks did not become a
  one-home loop

**Testable:** Yes — through `validator.Validate` and `cli.RunValidate`, both exported.

**Verification:** `go test -tags unit ./internal/validator/... ./internal/cli/... ./internal/oracle/...`;
`mise exec -- task test:unit`; `mise exec -- task test:integration`;
`rg -n 'stay unchecked' internal/validator/validator.go` prints nothing.

**Depends on:** Task 1, Task 2, Task 3

---

### Task 5: Keep `view/never-read` quiet while a `reads` names a view no slice declares

**Behavior:** a view named only by a misspelled `reads` yields one diagnostic — the unresolved
reference, at the `reads` the author mistyped — and not a second one at the view. The rule reports
exactly as before on every model whose `reads` all resolve.

**Acceptance Criteria:**

- [ ] `linter.Lint` reports no `view/never-read` diagnostic for any view of a model in which some
      `reads` — a trigger's, an automation's or a translation's — names a name no slice of the model
      declares
- [ ] The leaf proving it is written over a model that states a **resolving** `reads` on a second view,
      so `readViews` is non-empty and the rule's own adoption guard cannot be what silenced it, and
      that gives the view under test no second reader, so the unresolved `reads` is the only sufficient
      cause of the silence
- [ ] The same model with the unresolved `reads` removed — its construct stating no `reads` at all —
      reports `view/never-read` for that view, at its `NamePos`, so the leaf above cannot pass against
      a rule that never fired
- [ ] The suppression holds whichever of the three constructs spells the unresolved `reads`, asserted
      once per construct
- [ ] Through `oracle.Check`, a model declaring `CaseWorkspaceView` whose trigger reads
      `CaseWorkspacveView` reports exactly one line — `view "CaseWorkspacveView" does not exist` at the
      trigger's `reads` — asserted with one `require.Equal` over `reportedLines`, so a second
      diagnostic anywhere in the list fails it
- [ ] Every other leaf of `internal/linter/linter_test.go`'s `"view/never-read"` group (`:3744-4025`)
      passes unedited, except "read by nothing, with the model stating a reads elsewhere" (`:4016`),
      whose `reads elsewhere` names `OverdueLoansView` — a view that model declares nowhere. It names a
      view the model does declare instead, and still reports one diagnostic per `MemberLoansView`
      declaration, at `:20` and `:44`
- [ ] `ruleDescriptions["view/never-read"]` (`internal/linter/descriptions.go:31`) states the
      suppression, so `emod lint --explain view/never-read` tells an author why the rule went quiet and
      where to look instead. `internal/cli/lint_test.go`'s "all rules have descriptions" loop passes
      unedited
- [ ] `internal/cli/lint_test.go`'s `viewNeverReadEmod` (`:168`) and its leaves at `:1370` and `:1383`
      pass unedited — that fixture's automation reads a view it declares, so the rule still fires there
- [ ] `git diff --stat` names exactly `internal/linter/linter.go`,
      `internal/linter/descriptions.go`, `internal/linter/linter_test.go` and
      `internal/oracle/oracle_test.go`
- [ ] `mise exec -- task test:unit` and `mise exec -- task test:integration` are green

**Affected Files/Modules:**

- `internal/linter/linter.go` — a second model-wide set built beside `readViews` (`:65-80`) from
  `slice.Views` over `model.AllSlices()`, threaded to `checkNeverRead` (`:634`) the way `readViews`
  already is through `checkSlice` (`:157`, `:201`)
- `internal/linter/descriptions.go` — the `view/never-read` entry at `:31`
- `internal/linter/linter_test.go` — new leaves in the `"view/never-read"` group, and the repointed
  `reads` in the leaf at `:4016`
- `internal/oracle/oracle_test.go` — one whole-pipeline leaf for the one-mistake-one-diagnostic claim

**Patterns to Follow:**

- `tasks/learnings.md` "A set that doubles as a rule's adoption guard needs the emptiness check mutated,
  not the collection arm" — `readViews` is both a lookup and a has-adopted flag, and the new
  declared-view set sits beside it. Mutation-test the *suppression* independently of the guard, or a
  leaf passes for the wrong reason
- `tasks/learnings.md` "A silence assertion under a guarded rule must prove the guard is not what
  silenced it" — the second criterion above is that discipline written down: the model needs a second,
  resolving `reads` so `readViews` is non-empty whatever the suppression does
- `tasks/learnings.md` "A leaf that pairs the feature with something independently satisfying the same
  rule cannot see the feature" — the view under test must have no second reader, or the silence is
  explained by the reader rather than by the suppression
- `tasks/learnings.md` "A rule whose message branches on model state is pinned by whole formatted
  lines" — the `oracle.Check` leaf compares the complete list with `require.Equal` over
  `reportedLines` (`internal/oracle/oracle_test.go:389`), which pins count and order as well, so a
  surviving `view/never-read` line fails it
- The existing group's shape — models built in Go with hand-set positions, `viewAt`/`lendingWith`
  helpers (`internal/linter/linter_test.go:3747-3763`), `linesReportedBy(diags, rule)` for filtering by
  rule name (`:3928`) — is what the new leaves are written in
- `tasks/learnings.md` "A slice has two homes, and much of the repo still walks only one" — the new set
  is filled from `model.AllSlices()`, the same walk `readViews` uses, so a `mode dcb` context's own
  views count as declared
- `tasks/learnings.md` "A decided open question derived from a half-constraining validator check is the
  one to distrust" — the decision above was not settled by reading the validator. It is settled by what
  `checkNeverRead` and `readViews` do, and by the verified fact that `oracle.Run` is the linter's only
  production caller and always runs the validator first

**Testable:** Yes — through `linter.Lint`, `oracle.Check` and `cli.RunLintExplain`, all exported.

**Verification:** `go test -tags unit ./internal/linter/... ./internal/oracle/... ./internal/cli/...`;
`mise exec -- task test:unit`; `mise exec -- task test:integration`;
`go run ./cmd/emod lint --explain view/never-read` prints a description naming the suppression.

**Depends on:** Task 4

---

### Task 6: State in the reference that all three spellings of `reads` resolve, and correct the learning that says they do not

**Behavior:** an author reading `docs/dsl-reference.md` learns that `emod validate` resolves every
`reads` — a trigger's, an automation's and a translation's alike — and that `view/never-read` steps
aside while one of them is unresolved. The repository's own accumulated learning stops contradicting
the shipped code.

**Acceptance Criteria:**

- [ ] `docs/dsl-reference.md:811` no longer excepts "two of the three spellings of `reads`" from the
      references `emod validate` resolves
- [ ] The paragraph beneath the cross-reference table (`:823`) states that all three constructs that
      spell `reads` resolve against the views the model declares, and names the message they share,
      replacing the claim that a trigger's and a translation's are recorded and left unchecked
- [ ] The Automation Pattern's `reads` bullet (`:383`) keeps its account of model-wide resolution and
      loses its closing contrast with a trigger's and a translation's
- [ ] The `### View Pattern` bullet for `view/never-read` (`:358`) states that the rule stays silent for
      the whole model while any `reads` names a view no slice declares, so a misspelled name is
      reported once, where it is written
- [ ] `rg -n 'unchecked' docs/dsl-reference.md` returns only the three lines about something other than
      `reads` — a domain type accepting any literal (`:535`, `:577`) and a rejection's command name
      (`:558`)
- [ ] The cross-reference table row for `view <Name>` (`:819`) still lists all four referencing sites,
      and no other table row changes
- [ ] No `## <n>. <Title>` heading is added, removed, renumbered or reworded — `rg -c '^## [0-9]+\. '
      docs/dsl-reference.md` is still 13 — and no `### ` heading text changes, so every
      number-prefixed and every sub-heading in-document link still resolves
- [ ] No fenced ` ```emod ` block is added or edited — `rg -c '```emod' docs/dsl-reference.md` is still
      7 — so `internal/oracle/oracle_test.go`'s "documented models" leaves over the file still report
      nothing
- [ ] Section 13's `Diagram Palette` heading and its four-column table are untouched, so
      `dslReferencePalette` (`internal/diagram/contract_test.go:1349`) still parses it
- [ ] `tasks/learnings.md`'s entry "Only an automation's `reads` resolves; a trigger's and a
      translation's must stay unchecked" no longer states a constraint the code contradicts: it records
      what shipped — that all three resolve through `appendUndeclaredRef`, which models had to move
      first and why, and that `view/never-read` goes quiet model-wide while any `reads` is unresolved.
      No other entry in the file is edited
- [ ] `git diff --stat` names exactly two files: `docs/dsl-reference.md` and `tasks/learnings.md`
- [ ] `mise exec -- task test:unit` is green

**Affected Files/Modules:**

- `docs/dsl-reference.md` — `:358` (View Pattern lint bullet), `:383` (Automation Pattern `reads`
  bullet), `:811` and `:823` (section 11, Cross-References)
- `tasks/learnings.md` — the entry headed "Only an automation's `reads` resolves; a trigger's and a
  translation's must stay unchecked"

**Patterns to Follow:**

- `tasks/learnings.md` "A criterion may name an artefact the repository does not have" — the four
  locations above were each confirmed by `rg` before this criterion list was written; they exist and
  hold the sentences named
- `tasks/learnings.md` "`docs/dsl-reference.md` anchors embed the section number" and
  "`docs/dsl-reference.md` sub-heading anchors are cited more often than the numbered ones" — editing
  prose inside existing sections renumbers nothing, which is why the criteria pin both heading families
  rather than asking for a new section. Close by reconciling `^## [0-9]+\.` against `\(#[0-9]+-` and
  `^### ` against `\(#[a-z]`
- `tasks/learnings.md` "`docs/dsl-reference.md` section 13 is machine-read, so its table's spelling is
  load-bearing" — the one part of the file a Go test parses, and it is not the part being edited
- `tasks/learnings.md` "A decided open question derived from a half-constraining validator check is the
  one to distrust" — its closing instruction is that when the code and a recorded decision diverge, the
  correction is written down beside the decision rather than quietly dropped. The learning being
  corrected here is a `constraint` entry whose premise this story removed
- The rule bullets already in the file are the form for the View Pattern edit:
  `spec/invariant-never-exercised` under `### invariant`, `flow/rejection-without-spec` under
  `## 7. Flows`, `wire/type-format` under `### Wire Types`
- `~/.claude/rules/markdown-docs.md` — the reference must read as if it were the first version ever
  written. State what the tool does today; never narrate what the section used to say

**Testable:** No — the reference carries no assertion surface for prose, and `tasks/learnings.md` none
at all. `internal/oracle`'s "documented models" leaves parse the file's fenced ` ```emod ` blocks only,
and this task adds none; they serve here as the guard that the edit disturbed no block.

**Verification:** `mise exec -- task test:unit`; `rg -n 'unchecked' docs/dsl-reference.md` returns the
three non-`reads` lines only; `rg -c '^## [0-9]+\. ' docs/dsl-reference.md` returns 13 and
`rg -c '```emod' docs/dsl-reference.md` returns 7.

**Depends on:** Task 5

---

## Summary

**Six tasks**, ordered so the tree is green at every commit.

Tasks 1-3 move models before the check exists, which is not optional: `RunValidate` and `RunLint` are
both `oracle.Check`, and both return an error for any diagnostic whatever its severity, so the moment
a trigger's and a translation's `reads` resolve, five shared fixtures and two checked-in `.emod` files
stop being models the pipeline accepts and take `internal/oracle` and roughly forty `internal/cli`
subtests with them. This is the same sequencing `tasks/learnings.md` records US-008 using for a lint
rule, and it applies unchanged to a hard error. They are three tasks rather than one because they have
three different review surfaces — the pipeline fixtures and their canonical constants, the
library-lending fixtures and the two picture-and-document guards that used them, and the flagship
example with its integration-tagged AST transcription — and because each is independently green.

The brief's blast-radius measurement listed three models; the probe run for this decomposition found
**five**, and the two extra ones — `AutomationReadsLibraryLending` and `AutomationScheduleLibraryLending`
— are exactly the ones a scan for zero-diagnostic assertions misses, because they already carry an
expected diagnostic list and fail by gaining a line rather than by going from empty to non-empty. Task 2
exists for them.

Task 4 is the widening, and it is small by construction: `appendUndeclaredRef` already carries the
guard, the lookup and the `errorAt`, so the production diff is two lines and one deleted comment. What
it owes is coverage — the trigger's and the translation's positions, the model-wide scope, the absent
and empty cases, and the three constructs reading the same.

Task 5 comes after Task 4, not before, and the order matters: land the suppression first and a model
with a misspelled `reads` reports **nothing at all** for one commit, which is worse than the two
diagnostics the story is fixing. After Task 4 and before Task 5 the model reports both, which is the
state the repository is already in for an automation's `reads` today.

Task 6 is the prose, split out because it is the one surface with no assertion behind it and a
different kind of review: a reviewer of that commit reads the reference against the shipped behaviour.
It also carries the correction to `tasks/learnings.md`, which would otherwise ship as a recorded
constraint the code contradicts.

**Story criteria coverage:**

| Story criterion | Task |
|---|---|
| `emod validate` reports a trigger's `reads` naming a view no slice declares, at the trigger's `ReadsPos` | 4 |
| The same for a translation's `reads`, at its `ReadsPos` | 4 |
| The diagnostic matches the shape of the existing unresolved-`reads` message an automation produces | 4 |
| A view named only by a misspelled `reads` reports the unresolved reference and not also as never-read | 5 |
| The reference's statement that a trigger's and a translation's `reads` are recorded and left unchecked is updated | 6 |

Carried along, not stated by the story: the five shared fixtures and the two `all_patterns.emod` files
that would otherwise stop being models the pipeline accepts, the seven expected values that transcribe
them (Tasks 1-3), and the correction to `tasks/learnings.md` (Task 6).

**Settled, not deferred:** every repair points an existing `reads` at a view the same model already
declares — no view is invented anywhere; `view/never-read` goes silent for the whole model while any
`reads` is unresolved, with no near-miss heuristic and no per-context scoping; and the story's third
open question closes, all three spellings of `reads` being equivalent from Task 4 onward.

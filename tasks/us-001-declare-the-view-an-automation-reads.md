# US-001: Declare the view an automation reads

## Progress
- [x] Task 1: Parse `reads` inside an automation block
- [x] Task 2: Reject an automation `reads` that names no declared view
- [x] Task 3: Accept an automation's `reads` in the tree-sitter grammar
- [x] Task 4: Share a fixture whose automations read views
- [x] Task 5: Emit `reads` from `emod fmt` at a fixed position in the automation block
- [x] Task 6: Carry automation `reads` into JSON, CUE and the embedded schema
- [x] Task 7: Carry automation `reads` through the diagram document and back
- [x] Task 8: Show the view an automation reads in the viewer's details panel

---

## Story Reference

`user-stories/triggers-and-automations.md` → **US-001: Declare the view an automation reads**
(first of eleven stories in "Triggers and Automations"). Design notes:
`docs/proposals/triggers-and-automations-proposal.md`.

**In scope:** an optional `reads <ViewName>` entry inside `automation <Name> { ... }`; a validation
error, positioned at the view name, when the name resolves to no view declared anywhere in the model;
`emod fmt` emitting the entry at a fixed position so repeated formatting is stable; the value in the
JSON and CUE exports and in the embedded `schema.cue`; the value surviving export to diagram JSON,
editing in the viewer, and re-import; an automation with no `reads` behaving exactly as before at
every one of those stages. Carried along because the repo's writers would otherwise silently drop the
entry: the tree-sitter grammar (which must never reject what `emod validate` accepts) and a shared
fixture that exercises the entry everywhere it can appear.

**Out of scope:** renaming an automation's activation `trigger` to `on` (US-002); the `every "<expr>"`
schedule attribute (US-003); removing the trigger kind slot (US-004); drawing the `reads` edge in
SVG, draw.io or the viewer, and folding a viewer-drawn view→automation edge back into a `reads` entry
(US-005) — this story stops at the file and the two documents; lane placement (US-006); the palette
(US-007); the `automation/missing-todo-list` lint rule, so this story adds no rule, no `RuleName` and
no severity configuration (US-008); LSP hover, completion, go-to-definition and find-references over
an automation's `reads` (US-009); VS Code and tree-sitter *highlighting* queries (US-010);
`examples/*.emod`, `docs/dsl-reference.md` and `README.md` (US-011).

**Consequences of that boundary, decided.** Four shapes the story does not spell out:

1. *A `reads` entry is accepted in any automation, in any position within the block.* Every block body
   in this repo is an unbounded loop with no arity restriction, and the tree-sitter grammar mirrors
   that; imposing at-most-one or a fixed source order here would make US-002 and US-003 fight the
   parser rather than extend it. An entry written twice keeps the value written last, the way
   `description` already behaves.
2. *Only an automation's `reads` is resolved.* A trigger's and a translation's `reads` have never been
   checked, and `test.HotelReservation` plus `examples/all_patterns.emod` both name views in those two
   positions that the model never declares. Widening the check to them would break every fixture and
   example in the repo, which is not this story's business.
3. *No `reads` edge is added to the diagram JSON document.* The importer therefore reads the value
   from node metadata only. US-005 owns the edge in both directions, and the diagram differential
   Task 4 adds is the receipt that this story did not start it.
4. *The featured fixture's viewer round trip is measured on the automation, not in formatted bytes.*
   The diagram JSON document has no field for a context's `mode`, an event's `tags` or a command's
   `decides_on` — `jsonDiagramNode` (`internal/export/export.go:649-668`) and `convertModelToDiagram`
   (`:885`) carry none of them, and `internal/importer` decodes none of them — which is why the
   existing round-trip group runs on `all_patterns.emod`, `multi_context.emod` and small non-dcb
   sources rather than on a featured fixture. `test.AutomationReadsLibraryLending` puts its second
   slice home on a `mode dcb` context with tagged events and a `decides_on`, so its formatted round
   trip comes back without those three constructs however well `reads` survives (verified:
   `context "Reading Room" mode dcb` returns as `context "Reading Room"`, both `tags` blocks and the
   `decides_on` gone). Carrying them through the diagram document belongs to no story in this
   feature. The fixture's receipt is therefore `test.DeclaredAutomationReads` read off the imported
   model, and the byte-level receipt rides on a non-dcb source that already round-trips.

**Overarching constraint:** an automation without `reads` validates, formats and exports exactly as
before. That is load-bearing in five places — no existing golden in `internal/formatter`,
`internal/export`, `internal/diagram`, `internal/glossary` or `internal/cli` may need editing;
`oracle.Check` stays empty for all five current fixtures; the formatter's byte output for
`test.HotelReservation` is unchanged; the JSON and CUE documents for a model with no automation
`reads` are unchanged; and the diagram JSON node and edge lists for such a model are unchanged.

**Learnings folded in** from `tasks/learnings.md`: a new block entry keyword owes three things to the
parser's diagnostics (message, single-diagnostic recovery, `require.Len(t, diags, 1)`); put a new
parser subtest in the group that owns the construct; a new block entry goes after `description` and
ahead of nested blocks, in every writer — and the formatter silently deletes an entry it has not heard
of, so the parse→format→reparse comparison is the guard; a new exported field must land in JSON, CUE
and `schema.cue` in the same change, and JSON and CUE order their keys differently so the ordering
comes from a `json*` sibling; additive output changes owe a byte-identical receipt for models that do
not use the feature; a differential receipt must first prove the twin actually differs, and
`require.NotEqual` on a stripped twin is satisfiable without stripping anything; shared fixtures come
in an unfeatured/featured pair; a new shared fixture owes `internal/oracle` a zero-diagnostic subtest,
and a `mode dcb` fixture needs tagged events and a `decides_on` reaching them; a slice has two homes;
name an extracted helper after the contract its callers rely on; a second `require.Contains` on one
message is often shadowed by the first; `RuleName` marks a diagnostic `emod lint --explain` can
describe; an assertion whose expected value comes from the code under test cannot fail; the
tree-sitter grammar must never be stricter than the Go parser and its generated `src/` stays
gitignored, with repo tooling run through `mise exec --`; urfave/cli v2 discards every flag written
after the file argument, so an acceptance criterion must never be phrased `emod export <file> -f cue`;
never write emod source with `%q`; `emod fmt` canonicalises order so a fmt golden is never the input
re-indented; acceptance criteria describe the working tree, never commit or branch state.

---

## Codebase Context

**Lexer.** `reads` is already a keyword — `internal/lexer/token.go:28` (`KeywordReads`) and `:88` in
the `keywords` map. No lexer change is needed anywhere in this story, and the keyword-coverage
subtests that iterate `lexer.Keywords()` (`internal/lexer/tokenizer_test.go:14`,
`internal/parser/parser_test.go:225` and `:243`, `internal/oracle/oracle_test.go:44`) already prove
`reads` stays usable as a field name, type and modifier.

**AST.** `ast.Automation` (`internal/ast/ast.go:202-216`) carries `Comments`, `Name`/`NamePos`,
`Description`/`DescriptionPos`, `TriggerEvent`/`TriggerEventPos`, `Command`/`CommandPos`,
`TargetContext`/`TargetContextPos` and the open/close positions. `ast.Translation` (`:218-233`) is the
node that already holds a `Reads`/`ReadsPos` pair between the entry that activates it and its
`Command`, and `ast.Trigger` (`:173-187`) holds one after `Actor`.

**Parser.** `parseAutomation` (`internal/parser/parser.go:1035-1127`) is an unbounded loop over
`description`, `trigger`, `command` and `target context`, ending in a fallthrough branch that reports
`expected description, trigger, command, or target in automation, got %q` (`:1097`); no test asserts
that string today. Two existing entries are exactly the shape a `reads` entry takes: the trigger's
(`:594-603`) and the translation's (`:1164-1173`). Both recover by advancing a single token, which
`parseDescriptionInto` (`:1428-1439`) improves on by draining through `skipRestOfLineOrBlockEnd`
(`:1523-1527`) — that helper also stops at a closing brace, which is what lets a block still close.
`checkIdentifierLike` (`:1503-1509`) asks `Kind.IsKeyword()` rather than comparing ordinals. After the
loop, `parseAutomation` reports `automation block requires a trigger event` and `... requires a
command`; `reads` is optional and gets no such check.

**Parser tests.** `internal/parser/parser_test.go` is one umbrella split into thirteen top-level
groups. `"automations"` (`:1898`) owns the construct; `"error reporting"` (`:2355`) owns recovery and
message shapes; `"triggers"` (`:1493`) and `"translations"` (`:2128`) hold the existing `reads`
subtests to copy.

**Validator.** `internal/validator/validator.go` builds a `modelIndex` (`:39-50`) in `newModelIndex`
(`:52-78`), which walks both homes a slice has — `ctx.Slices` and every `agg.Slices` (`:65-70`) — then
calls `collect` (`:80-116`) per slice. `collect` indexes command and event names but **no view names**:
a view's own name is never indexed, only the events it subscribes to are checked
(`referenceDiagnostics:312-318`). The automation branch of `referenceDiagnostics` (`:296-306`) is the
shape for a new reference check — three sibling checks reading `<kind> %q does not exist`, each
positioned at the reference rather than at the automation name, built through `errorAt` (`:144-151`),
which stamps the error severity and leaves `RuleName` empty.
`diagnostic.Entry.String()` (`internal/diagnostic/entry.go:33-38`) formats as `file:line: message`,
with `Column` held on the entry but not printed.

**Validator tests.** `internal/validator/validator_test.go` groups by rule — `"target context
references"` (`:21`), `"command references"` (`:147`), `"event references"` (`:450`). The
whole-formatted-line `require.Equal(t, ..., diags[0].String())` at `:968`, `:1005` and `:1375` is the
assertion shape to copy — `reportedLines` (`:2907`) with one `require.Equal` over the whole list at
`:1436` is its multi-diagnostic form. Layering a second `require.Contains` on one message is the
finding `tasks/learnings.md` records.

**Formatter.** `writeAutomation` (`internal/formatter/formatter.go:340-354`) emits `description`, then
`trigger`, `command`, `target context`, each `lineIfSet`-style. `writeTranslation` (`:356-373`) shows
the position a `reads` line takes among its siblings, and `writeTrigger` (`:197-208`) the other
existing one. `writeSlice` (`:145-195`) emits kinds in a fixed order and `Format` always opens with
`emod <n>`. `internal/formatter/formatter_test.go:428` `"round-trip through the parser"` is the only
test class that catches the formatter dropping a declaration, and its subtests are per-fixture
(`:497`, `:505`). `internal/cli/fmt_test.go` pins canonical `*FormattedEmod` constants and feeds them
to `requireFmtSettlesOn`; no CLI change is needed here, since `RunFmt` writes whatever
`formatter.Format` produces.

**Exports.** `internal/export/export.go` keeps three document families apart.
`jsonAutomation` (`:158-171`) lists positions first and then values — `jsonTranslation` (`:173-187`)
is the sibling that already has `ReadsPosition` and `Reads` — and `convertAutomation` (`:596-614`) is
its converter. `cueWriter.writeAutomation` (`:1363-1370`) emits `name`, `description`,
`trigger_event`, `command`, `target_context` via `lineIfSet`; `writeTranslation` (`:1372-1382`) shows
where `reads` sits. `internal/cue/schema.cue` `#Automation` (`:54-61`) mirrors the JSON keys, and
`#Translation` (`:63-71`) and `#Trigger` (`:15-22`) already declare `reads?: string`. Two subtests
couple the surfaces: `"output conforms to the schema's Model definition"`
(`internal/export/export_test.go:3442-3456`, one leaf per fixture, running `cue vet -d '#Model'`) and
`"CUE and JSON exports describe the same model"` (`:3458-3477`, decoding both exports and comparing).
`internal/cue/embed_test.go` holds `fullModelJSON` (`:126`), whose comment states it exercises every
definition the schema declares, checked by `"accepts a model using every element the language offers"`
(`:66`).

**Diagram JSON and the viewer.** `jsonDiagramNode` (`internal/export/export.go:649-668`) already has a
shared `Reads` field, populated for the trigger node (`:800`) and the translation node (`:858`); the
automation node (`:825-842`) carries `TriggerEvent`, `Command` and `TargetContext` only. Automation
edges (`:1062-1091`) draw `automation_trigger` and `automation_command`; the `reads` edge type exists
but is emitted for translations alone (`:1103-1111`). `internal/importer/importer.go` reverses the
document: `diagramNode` (`:25-42`) already decodes `reads`, `buildSlice` copies it onto the trigger
(`:170`) and the translation (`:221`) but not the automation (`:206-215`), and `foldEdges` (`:251-287`)
folds a `reads` edge onto a translation only (`:281-284`). `internal/wasm/pipeline.go:60` and `:79` are
the two halves the viewer calls, so the round trip is entirely Go-side — `internal/viewer/static`
passes node objects through untouched (`model.js:19-31`, `emod-export.js`). The viewer's details panel
prints a `Reads` row for a trigger (`ui.js:337`) and a translation (`:367`) but not for an automation
(`:350-359`).

**Fixtures.** `internal/test/fixtures.go` holds `HotelReservation` (`:13`, uses no optional feature and
is the byte-identical witness), `DescribedHotelReservation` (`:100`), `KeywordFieldSearchCatalog`
(`:208`), `InvariantLibraryLending` (`:311`) and `SpecLibraryLending` (`:420`, which declares its
construct in both slice homes and is the model to copy). `WithoutSpecs` (`:604-635`) is the twin
helper that returns a copy and deliberately leaves a nil list nil; `DeclaredSpecNames` (`:640-648`)
and `SpecLibraryLendingSpecNames` (`:575-583`) are the read-back-and-transcribe pair;
`declaredSlices` (`:650-659`) is the walk that visits both homes. `internal/test/models.go:13-38`
holds one parsed-model helper per fixture. `internal/oracle/oracle_test.go:24` `"clean input"` holds
one `require.Empty` subtest per fixture, and `oracle.Check` runs lexer, parser, validator *and*
linter. `internal/diagram/contract_test.go:207` and `:221` are the two differentials to copy, with
`withoutInvariants` (`:706`) as the copy-returning stripper.

**Tree-sitter.** `editors/tree-sitter-emod/grammar.js` builds every block with
`buildDescribedBlock($, ...items)`, so entries are unordered and unbounded.
`automation_definition` (`:217-227`) lists `trigger`, `command` and `target context`;
`translation_definition` (`:229-240`) and `trigger_definition` (`:168-178`) both already carry a
`reads` item, spelled against the permissive identifier rule rather than the strict one so a keyword
spelling still matches. The corpus case `Slice with automation` lives in
`test/corpus/slice.txt:259-286`. `src/` is gitignored and `task test:grammar` regenerates before
running the corpus.

**Not touched, deliberately.** `internal/lexer` (`reads` is already a keyword);
`internal/linter` (US-008 owns the rule, and there is no unused-view rule for a `reads` reference to
satisfy); `internal/glossary` (an automation contributes no term of its own, and a `reads` value is a
reference rather than a term, so its existing goldens are the receipt); `internal/cli` (no new command,
flag or error kind — the validator diagnostic reaches `emod validate` through the existing path, and
`emod fmt` writes whatever `formatter.Format` produces); `internal/diagram/{svg,drawio,mermaid,ascii}.go`
(US-005); `internal/lsp` (US-009); `editors/vscode/syntaxes/emod.tmLanguage.json`, whose keyword
alternation (`:63`) already contains `reads`, and `editors/tree-sitter-emod/queries/highlights.scm`
(US-010); `examples/*.emod`, `docs/dsl-reference.md` — including the cross-reference table at
`:632`, which stays stale until US-011 — and `README.md`; `e2e/` and `e2e-viewer/`.

---

## Tasks

### Task 1: Parse `reads` inside an automation block

**Behavior:** `automation <Name> { ... }` accepts an optional `reads <ViewName>` entry anywhere among
its other entries, recording the view name and the source position of that name on the automation. An
automation with no `reads` parses exactly as it does today. A `reads` entry whose value is missing or
is not identifier-like reports exactly one diagnostic and does not swallow the entry written on the
following line. The message the parser reports for an unrecognised entry inside an automation names
`reads` among the entries it accepts.

**Acceptance Criteria:**
- [ ] An automation declaring `reads PendingConfirmationsView` between its activation event and its
      command parses with no diagnostics, and the automation carries that view name together with the
      filename, line and column of the name token
- [ ] The same entry written as the first entry of the block, and written after `target context`,
      both parse with no diagnostics and yield the same recorded view name — position within the
      block is free, as it is for `description`
- [ ] An automation declaring `reads` twice keeps the value written last
- [ ] An automation with no `reads` parses with no diagnostics and records an empty view name, and
      `oracle.Check` over `test.HotelReservation`, `test.DescribedHotelReservation`,
      `test.KeywordFieldSearchCatalog`, `test.InvariantLibraryLending` and `test.SpecLibraryLending`
      still returns no diagnostics
- [ ] A `reads` entry with no value, followed on the next line by a `command` entry, reports exactly
      one diagnostic (`require.Len(t, diags, 1)`) whose message names both `reads` and `automation`,
      and the `command` entry on the following line is still parsed onto the automation
- [ ] A `reads` entry with no value written as the last entry of the block still lets the automation
      block and its enclosing slice close, reporting one diagnostic
- [ ] The message reported for an unrecognised entry inside an automation names `reads` alongside
      description, trigger, command and target, and a subtest asserts that string
- [ ] No existing subtest in `internal/parser/parser_test.go` needs editing

**Affected Files/Modules:**
- `internal/ast/ast.go` — `Automation` (`:202-216`) gains the view name and its position
- `internal/parser/parser.go` — `parseAutomation` (`:1035-1127`) accepts the entry; its "expected …"
  message (`:1097`) grows a term
- `internal/parser/parser_test.go` — subtests in the `"automations"` group (`:1898`), and the message
  and recovery subtests in `"error reporting"` (`:2355`)

**Patterns to Follow:**
- The entry itself: the translation's `reads` (`internal/parser/parser.go:1164-1173`) and the
  trigger's (`:594-603`)
- Recovering from a malformed value with one diagnostic: `parseDescriptionInto`
  (`internal/parser/parser.go:1428-1439`) draining via `skipRestOfLineOrBlockEnd` (`:1523-1527`) —
  `tasks/learnings.md` "A new block entry keyword owes three things to the parser's diagnostics",
  including the `require.Len(t, diags, 1)` pin, and "Name an extracted helper after the contract its
  callers rely on" for why that helper also stops at `}`
- AST field placement: `ast.Translation` (`internal/ast/ast.go:218-233`) puts its `Reads`/`ReadsPos`
  pair between the entry that activates it and its `Command`; mirror that ordering so every writer
  added later reads the same sequence off the struct
- Identifier-like values accept keyword spellings: `checkIdentifierLike`
  (`internal/parser/parser.go:1503-1509`) — `tasks/learnings.md` "Ask the lexer which keywords exist"
- Subtest placement: `tasks/learnings.md` "Put a new parser subtest in the group that owns the
  construct"
- The block loop imposes no arity and no order — `tasks/learnings.md` "The tree-sitter grammar must
  never be stricter than the Go parser" records why every body in this repo is an unbounded loop

**Testable:** Yes — through `lexer.Scan` + `parser.Parse` and `oracle.Check`, all exported.

**Verification:** `mise exec -- go test -tags unit ./internal/ast/... ./internal/parser/...
./internal/oracle/...`; `go build ./...`.

**Depends on:** None

---

### Task 2: Reject an automation `reads` that names no declared view

**Behavior:** `emod validate` reports an error when an automation's `reads` names a view no slice in
the model declares. The diagnostic sits at the position of the view name inside the `reads` entry, and
its message quotes that name. Resolution is model-wide and covers both homes a slice has, so an
automation may read a view declared in another aggregate, in another context, or in a slice declared
directly on a `mode dcb` context. A trigger's `reads` and a translation's `reads` stay unchecked.

**Acceptance Criteria:**
- [ ] An automation whose `reads` names a view declared in a slice of another context validates with
      no diagnostics
- [ ] An automation whose `reads` names a view declared in a slice directly under a `mode dcb`
      context validates with no diagnostics
- [ ] An automation declared in a slice directly under a `mode dcb` context, whose `reads` names an
      undeclared view, produces the diagnostic — the check reaches automations in both slice homes
- [ ] An automation whose `reads` names an undeclared view produces exactly one diagnostic whose whole
      formatted line, asserted with a single `require.Equal` on `diags[0].String()`, names the file,
      the line of the `reads` entry and the quoted view name
- [ ] The diagnostic's `Line` and `Column` are those of the view name token, not of the automation
      name — demonstrated on an input where the two differ
- [ ] The diagnostic's `Severity` is `diagnostic.Error` and its `RuleName` is empty, so
      `emod lint --explain` is never asked to describe it
- [ ] An automation with no `reads` produces no diagnostic from this check
- [ ] A trigger's `reads` and a translation's `reads` naming an undeclared view still produce no
      diagnostic: `oracle.Check` over `test.HotelReservation`, which names `AvailableRoomsView` on a
      trigger and `BookingWebhookView` on a translation while declaring neither, still returns no
      diagnostics
- [ ] `internal/cli/validate_test.go` needs no editing, and `emod validate` over every file under
      `examples/` still exits 0

**Affected Files/Modules:**
- `internal/validator/validator.go` — `modelIndex` (`:39-50`) and `collect` (`:80-116`) learn view
  names; the automation branch of `referenceDiagnostics` (`:296-306`) gains the check
- `internal/validator/validator_test.go` — a new group beside `"target context references"` (`:21`)
  and `"event references"` (`:450`)

**Patterns to Follow:**
- Message and construction: the sibling checks at `internal/validator/validator.go:298-305`, built
  with `errorAt` (`:144-151`)
- Index the name where the construct is collected, so `newModelIndex`'s walk over `ctx.Slices` and
  every `agg.Slices` (`internal/validator/validator.go:65-70`) covers both homes for free —
  `tasks/learnings.md` "A slice has two homes, and much of the repo still walks only one"
- Assert the whole formatted diagnostic line rather than layering `require.Contains` calls:
  `internal/validator/validator_test.go:968`, `:1005` and `:1375` — `tasks/learnings.md` "A second
  `require.Contains` on one message is often shadowed by the first"
- Leave `RuleName` empty for a hard error — `tasks/learnings.md` "`RuleName` marks a diagnostic
  `emod lint --explain` can describe"
- Do not extend the check to a trigger or a translation; decision 2 in the Story Reference above
  records why, and `test.HotelReservation` is the fixture that fails if you do

**Testable:** Yes — through `validator.Validate` and `oracle.Check`, both exported.

**Verification:** `mise exec -- go test -tags unit ./internal/validator/... ./internal/oracle/...`;
`go run ./cmd/emod validate examples/all_patterns.emod` and the other files under `examples/` exit 0.

**Depends on:** 1

---

### Task 3: Accept an automation's `reads` in the tree-sitter grammar

**Behavior:** the tree-sitter grammar parses an automation block containing a `reads` entry without an
`ERROR` node, so a file `emod validate` accepts is not red-squiggled in an editor using the grammar.
Everything the grammar accepted before is still accepted.

**Acceptance Criteria:**
- [ ] `automation_definition` in `editors/tree-sitter-emod/grammar.js` admits a `reads` entry, spelled
      the way `translation_definition` and `trigger_definition` spell theirs
- [ ] A corpus case in `editors/tree-sitter-emod/test/corpus/slice.txt` covers an automation with
      `reads` written between its activation event and its command, and its expected tree contains no
      `ERROR` or `MISSING` node
- [ ] A second corpus case covers `reads` written as the first entry of the automation block, so the
      grammar is proved not to impose an order the Go parser does not
- [ ] The existing `Slice with automation` case (`:259-286`) still passes unedited
- [ ] `mise exec -- task test:grammar` passes when run through `mise exec --`, which resolves the
      repo-pinned tree-sitter CLI rather than whichever one is on `PATH`
- [ ] `editors/tree-sitter-emod/.gitignore` still ignores `src/`, so `git check-ignore
      editors/tree-sitter-emod/src` succeeds and the regenerated parser stays out of the tree; the
      only files this task changes under `editors/tree-sitter-emod` are `grammar.js` and
      `test/corpus/slice.txt`

**Affected Files/Modules:**
- `editors/tree-sitter-emod/grammar.js` — `automation_definition` (`:217-227`)
- `editors/tree-sitter-emod/test/corpus/slice.txt` — beside `Slice with automation` (`:259-286`)

**Patterns to Follow:**
- The entry spelling: `translation_definition` (`editors/tree-sitter-emod/grammar.js:229-240`) and
  `trigger_definition` (`:168-178`)
- Items go into `buildDescribedBlock`, never behind an `optional(...)` in a block body —
  `tasks/learnings.md` "The tree-sitter grammar must never be stricter than the Go parser"
- Run the target through `mise exec --`, and leave `src/` gitignored — `tasks/learnings.md` "Run repo
  tooling through `mise exec --`, not bare PATH" and "Generated tree-sitter `src/` stays gitignored"
- Highlighting queries are US-010's; this task changes no `.scm` file

**Testable:** Yes — the tree-sitter corpus is the test surface, run by `task test:grammar`.

**Verification:** `mise exec -- task test:grammar`; `git status --porcelain editors/tree-sitter-emod`
lists only `grammar.js` and `test/corpus/slice.txt`.

**Depends on:** 1

---

### Task 4: Share a fixture whose automations read views

**Behavior:** `internal/test` gains a pipeline-wide fixture whose automations declare the views they
read, in both homes a slice has and reaching across contexts, alongside an automation that omits
`reads` mid-block. The fixture is clean under `oracle.Check`, so every downstream package can use it
as the featured model for this feature while `test.HotelReservation` stays the witness that a model
with no automation `reads` is untouched. A twin helper returns a copy with every automation `reads`
cleared, and the four diagram renderers produce identical bytes for the fixture and that twin — the
receipt that this story stopped at the file and left the picture to US-005.

**Acceptance Criteria:**
- [ ] `internal/test/fixtures.go` declares a new `.emod` fixture constant named for its feature and
      domain in the shape of `InvariantLibraryLending` and `SpecLibraryLending`, with a doc comment
      stating which role it plays, and `internal/test/models.go` gains its parsed-model helper beside
      the existing four
- [ ] The fixture declares at least one automation with `reads` in a slice nested in an aggregate and
      at least one in a slice declared directly under a `mode dcb` context
- [ ] At least one automation reads a view declared in a different context, so the fixture witnesses
      that resolution is model-wide
- [ ] At least one automation omits `reads` and is followed by a further entry in the same block, so
      the omitted case cannot be answered by an entry that had nothing to run into
- [ ] `oracle.Check` over the fixture returns no diagnostics, in a subtest beside the existing
      per-fixture leaves in `internal/oracle/oracle_test.go:24` — lexer, parser, validator and linter
      all clean, including `dcb/untagged-event` and `dcb/orphan-tag-key`
- [ ] `internal/test` exports the fixture's automation `reads` values transcribed by hand, in
      declaration order across both slice homes, and a read-back helper over the parsed model returns
      exactly that list
- [ ] A twin helper returns a *copy* of the parsed model with every automation `reads` cleared in both
      slice homes, leaving the model it was given carrying all of its values; the helper's name states
      that it returns a copy
- [ ] `internal/diagram/contract_test.go` gains one subtest per exporter asserting that draw.io, SVG,
      Mermaid and ASCII output for the fixture is byte-identical to the output for its twin, opening
      with `require.NotEqual` on the two models plus a positive check that the twin lost the values in
      both homes and the original still carries the transcribed list
- [ ] No existing golden or expected constant in `internal/formatter`, `internal/export`,
      `internal/diagram`, `internal/glossary` or `internal/cli` needs editing

**Affected Files/Modules:**
- `internal/test/fixtures.go` — the fixture constant, the transcribed value list, the read-back helper
  and the copy-returning twin helper
- `internal/test/models.go` — the parsed-model helper (`:13-38`)
- `internal/oracle/oracle_test.go` — a leaf in `"clean input"` (`:24`)
- `internal/diagram/contract_test.go` — a subtest beside `"declaring invariants leaves the picture
  untouched"` (`:207`) and `"stating specs leaves the picture untouched"` (`:221`)

**Patterns to Follow:**
- The fixture shape, including a lint-clean `mode dcb` context with tagged events and a `decides_on`
  reaching those tags: `test.SpecLibraryLending` (`internal/test/fixtures.go:420-569`) —
  `tasks/learnings.md` "A new shared fixture owes `internal/oracle` a zero-diagnostic subtest"
- Both slice homes, and a transcribed list read back off the model:
  `SpecLibraryLendingSpecNames` (`internal/test/fixtures.go:575-583`), `DeclaredSpecNames`
  (`:640-648`) and `declaredSlices` (`:650-659`) — `tasks/learnings.md` "A slice has two homes"
- The copy-returning stripper: `WithoutSpecs` (`internal/test/fixtures.go:604-635`), including why
  `slicesWithoutSpecs` leaves a nil list nil, and `withoutInvariants`
  (`internal/diagram/contract_test.go:706`) — `tasks/learnings.md` "Name an extracted helper after the
  contract its callers rely on" (the `WithOrdinaryFieldNames` trap of renaming in place)
- The differential and its guards: `internal/diagram/contract_test.go:207-233` —
  `tasks/learnings.md` "A differential receipt must first prove the twin actually differs" and
  "`require.NotEqual` on a stripped twin is satisfiable without stripping anything"
- Keep the unfeatured/featured pair: `test.HotelReservation` stays free of the feature —
  `tasks/learnings.md` "Shared fixtures come in an unfeatured/featured pair"
- Place the omitted `reads` mid-block — `tasks/learnings.md` "Exercise an omitted optional part
  mid-block, never as the last entry"
- Write the fixture as literal emod source; never produce emod text through `%q` —
  `tasks/learnings.md` "Never write emod source with `%q`"
- The diagram differential is deliberately temporary: US-005 draws the edge and replaces it. Say so in
  the subtest's failure message so the next story reads it as a boundary rather than a regression

**Testable:** Yes — through `oracle.Check`, the exported `internal/test` helpers, and the exported
`diagram.Export*` renderers.

**Verification:** `mise exec -- go test -tags unit ./internal/oracle/... ./internal/diagram/...
./internal/test/...`; `mise exec -- go test -tags unit ./...` shows no other package needing an edit.

**Depends on:** 1, 2

---

### Task 5: Emit `reads` from `emod fmt` at a fixed position in the automation block

**Behavior:** `emod fmt` writes an automation's `reads` entry at one fixed position in the block —
after the activation event and ahead of the command, where a translation's `reads` sits — whatever
position it was written in. A parse → format → reparse cycle recovers the value rather than dropping
it, and formatting the formatter's own output produces identical bytes. An automation with no `reads`
formats to exactly the bytes it formats to today.

**Acceptance Criteria:**
- [ ] Formatting an automation that declares `reads` puts the entry on its own line at the block's
      indent, between the activation event line and the command line, asserted against a whole
      expected output in the `"element formatting"` group
- [ ] Two sources for the same model — one writing `reads` as the block's first entry, one writing it
      after `target context` — format to the same bytes, and that expected byte string is written out
      in the test rather than being either input re-indented
- [ ] Formatting `test.AutomationReadsLibraryLendingModel(t)` and reparsing the output yields a model
      whose `test.DeclaredAutomationReads` equals `test.AutomationReadsLibraryLendingViewNames`, and
      formatting that output again produces identical bytes
- [ ] The same parse → format → reparse over `test.WithoutAutomationReads` of that model reads back
      no automation `reads` at all, while the trigger's `reads AvailableCopiesView` line still
      appears in the twin's formatted output — the entry is written from the automation, not from
      whatever else in the slice happens to carry one
- [ ] The fixture's formatted output and its twin's differ (`require.NotEqual`), and the two agree
      once every `reads` line is dropped from both — the receipt that this task moved no other line
      of a two-hundred-line model
- [ ] `"formats automation block"` (`internal/formatter/formatter_test.go:258-306`), whose model
      declares no `reads`, passes unedited, and so does every canonical `*FormattedEmod` constant in
      `internal/cli/fmt_test.go`

**Affected Files/Modules:**
- `internal/formatter/formatter.go` — `writeAutomation` (`:340-354`)
- `internal/formatter/formatter_test.go` — a subtest in `"element formatting"` (`:32`, beside
  `"formats automation block"` at `:258`) and a leaf in `"round-trip through the parser"` (`:428`,
  beside `:497` and `:505`)

**Patterns to Follow:**
- The line to emit and where it sits: `writeTranslation` (`internal/formatter/formatter.go:356-373`)
  and `writeTrigger` (`:197-208`)
- A new block entry goes after `description` and ahead of nested blocks in every writer —
  `tasks/learnings.md` "A new block entry goes after `description` and ahead of nested blocks, in
  every writer", including why the formatter is the writer that hurts to forget
- The parse → format → reparse comparison against the original model is the guard, not idempotence
  and not an existing golden: `internal/formatter/formatter_test.go:428-503`, and
  `requireStableFormat` (`:3366-3376`) is the helper that pairs the reparse with the second-pass
  byte comparison
- The twin plus its two guards: `test.WithoutAutomationReads` and `test.DeclaredAutomationReads`
  (`internal/test/fixtures.go:789-805`, `:850-866`) — `tasks/learnings.md` "`require.NotEqual` on a
  stripped twin is satisfiable without stripping anything" and "A differential receipt must first
  prove the twin actually differs"
- A criterion phrased "exactly as before" has to be proved against input that exercises the new
  handling: the fixture's `RemindOnDueDate` sits mid-block with no `reads` ahead of an automation
  that has one — `tasks/learnings.md` "Exercise an omitted optional part mid-block, never as the last
  entry"
- Every expected string starts with `emod <n>` — `tasks/learnings.md` "Formatter output always begins
  with `emod N`"
- A fmt golden is never the input re-indented — `tasks/learnings.md` "`emod fmt` canonicalises order,
  so a fmt golden is never the input re-indented"
- No `internal/cli` change: `RunFmt` writes whatever `formatter.Format` returns

**Testable:** Yes — through `formatter.Format`, `parser.Parse` and the exported `internal/test`
helpers.

**Verification:** `mise exec -- go test -tags unit ./internal/formatter/... ./internal/cli/...`;
`mise exec -- go test -tags unit ./...`.

**Depends on:** 1, 4

---

### Task 6: Carry automation `reads` into JSON, CUE and the embedded schema

**Behavior:** `export.ExportJSON` and `export.ExportCUE` carry an automation's `reads` value and the
position it was read from, and `internal/cue/schema.cue` declares the key so `emod schema` describes
it and `cue vet -d '#Model'` accepts a document that has it. A model whose automations declare no
`reads` produces byte-identical output in both formats.

**Acceptance Criteria:**
- [ ] The JSON export of `test.AutomationReadsLibraryLendingModel(t)` carries every automation `reads`
      value, keyed the way the trigger and translation documents key theirs, together with the
      position of the view name; both new fields sit where `jsonTranslation`'s do — the position after
      the activation event's position, the value after the activation event — so the emitted object's
      key order matches its `json*` siblings rather than the schema's
- [ ] The CUE export of the same fixture carries the same values, emitted after the activation event
      and ahead of the command
- [ ] `#Automation` in `internal/cue/schema.cue` declares the key as optional, and
      `mise exec -- go run ./cmd/emod schema` prints it
- [ ] `internal/cue/embed_test.go`'s `fullModelJSON` (`:174-179`) gives its automation the key, so
      "accepts a model using every element the language offers" (`:66`) exercises it, and deleting
      the key from `#Automation` makes that subtest fail
- [ ] A leaf beside `"output conforms to the schema's Model definition"`
      (`internal/export/export_test.go:3441-3456`) runs `cue vet -d '#Model'` over the fixture's CUE
      export and passes
- [ ] A leaf beside `"CUE and JSON exports describe the same model"` (`:3458-3477`) requires the
      fixture's two exports equal, and a further assertion reads the `reads` values back out of the
      decoded document, keyed by the slice that owns each automation, against a map transcribed by
      hand in the test — so a value neither writer emits cannot agree trivially
- [ ] The same read-back over `test.WithoutAutomationReads` of that model finds no automation `reads`
      anywhere in either document, while the trigger's and the translation's `reads` are still
      present in both — the differential that pins this task to the automation
- [ ] No existing expected constant in `internal/export/export_test.go` is edited, so the JSON and
      CUE exports of every model with no automation `reads` are unchanged
- [ ] The diagram JSON document is not changed by this task: the automation node still carries
      `trigger_event`, `command` and `target_context` only, and every subtest in `"diagram json"`
      (`:1544`) passes unedited

**Affected Files/Modules:**
- `internal/export/export.go` — `jsonAutomation` (`:158-171`), `convertAutomation` (`:596-614`),
  `cueWriter.writeAutomation` (`:1363-1370`)
- `internal/cue/schema.cue` — `#Automation` (`:54-61`)
- `internal/cue/embed_test.go` — `fullModelJSON` (`:126`, its automation at `:174-179`)
- `internal/export/export_test.go` — leaves in the `"model json"` (`:21`), `"cue"` (`:3074`), schema
  conformance (`:3441`) and export-parity (`:3458`) groups

**Patterns to Follow:**
- Field, key name and struct ordering: `jsonTranslation` (`internal/export/export.go:173-187`) and
  `jsonTrigger` (`:132-145`) — positions first, then values, and copy the ordering from a `json*`
  sibling, never from the schema (`tasks/learnings.md` "JSON and CUE order their document keys
  differently — do not mirror one struct into the other")
- CUE emission: `cueWriter.writeTranslation` (`internal/export/export.go:1372-1382`) using
  `lineIfSet`
- All three surfaces land together — `tasks/learnings.md` "A new exported field must land in JSON, CUE
  and `schema.cue` in the same change". `#Automation` is a closed definition, so `cue vet -d '#Model'`
  rejects the key until the schema declares it
- The two coupled guards cannot see a value neither writer emits, so read the values back out of the
  decoded document against a hand-transcribed map: `listsKeyedBy`
  (`internal/export/export_test.go:3912-3927`) with `keywordFieldsByOwner` as the transcribed-constant
  precedent — `tasks/learnings.md` "The two export guards cannot see a list neither writer emits".
  Note that `requireBothFormatsAgree` (`:4023-4039`) strips positions before comparing, so the
  position key is only observable in the JSON document
- The twin and its guards: `test.WithoutAutomationReads` and `test.DeclaredAutomationReads`
  (`internal/test/fixtures.go:789-805`, `:850-866`) — `tasks/learnings.md` "`require.NotEqual` on a
  stripped twin is satisfiable without stripping anything"
- Do not merge `jsonDiagramEvent` back into `jsonEvent`, and do not touch the diagram document here —
  `tasks/learnings.md` "A new exported field must land in JSON, CUE and `schema.cue` in the same
  change" records why the diagram document is deliberately forked
- Never write an acceptance check as `emod export <file> -f cue`: the flag is discarded after the file
  argument (`tasks/learnings.md` "urfave/cli v2 discards every flag written after the file argument").
  Exercise `export.ExportJSON` / `export.ExportCUE`, or put the flag before the path
- Prove the byte-identical claim for a model with no automation `reads` rather than asserting it —
  `tasks/learnings.md` "Additive output changes owe a byte-identical receipt for models that do not
  use the feature"

**Testable:** Yes — through `export.ExportJSON`, `export.ExportCUE` and `cue.Schema`, all exported.

**Verification:** `mise exec -- go test -tags unit ./internal/export/... ./internal/cue/...`;
`mise exec -- go run ./cmd/emod schema` prints the key.

**Depends on:** 1, 4

---

### Task 7: Carry automation `reads` through the diagram document and back

**Behavior:** the diagram JSON document carries an automation's `reads` value as node metadata and
`importer.ImportDiagram` reads it back onto the automation it builds, so a model exported for the
viewer and re-imported keeps the view each automation reads. No `reads` edge is drawn for an
automation and none is folded back onto one — US-005 owns the edge in both directions.

**Acceptance Criteria:**
- [ ] The automation node in `export.ExportDiagramJSON`'s output carries the view name under the same
      key the trigger and translation nodes use, asserted beside `"automation node with
      trigger_event/command/target_context"` (`internal/export/export_test.go:1798-1843`)
- [ ] Exporting `test.AutomationReadsLibraryLendingModel(t)` to diagram JSON and importing the result
      yields a model whose `test.DeclaredAutomationReads` equals
      `test.AutomationReadsLibraryLendingViewNames` — both slice homes, including the automation whose
      view another context declares
- [ ] The same export → import over `test.WithoutAutomationReads` of that model reads back no
      automation `reads`, while the model it was copied from still carries all three, so the pair
      cannot be answered by a builder that hardcodes or drops the value
- [ ] A leaf in `"round trip"` (`internal/importer/importer_test.go:38`) over a hand-written non-dcb
      source — already in canonical formatted form, declaring a view in one slice and, in a sibling
      slice, one automation that reads it beside one that reads nothing — formats back to the
      identical bytes after the export → import path. Verified precondition: that shape already
      round-trips byte for byte, failing on the absent `reads` line alone, so the leaf fails for
      exactly one reason before this task
- [ ] The featured fixture is not used for a byte-level round trip, for the reason recorded as
      decision 4 in the Story Reference, and the five existing round-trip leaves (`:39`, `:52`,
      `:63`, `:90`, `:119`) pass unedited
- [ ] The edge list `export.ExportDiagramJSON` produces for the fixture contains no `reads` edge whose
      target is an automation node, and `foldEdges` still folds a `reads` edge onto a translation
      only; both assertions name US-005 in their failure message, so the next story reads them as a
      boundary rather than a regression
- [ ] No file under `internal/viewer` is changed by this task — `git status --porcelain
      internal/viewer` lists nothing — since node objects pass through `model.js` and
      `emod-export.js` untouched and `internal/wasm/pipeline.go:60` and `:79` are the two halves the
      viewer calls

**Affected Files/Modules:**
- `internal/export/export.go` — the automation diagram node (`:825-842`)
- `internal/importer/importer.go` — the automation branch of `buildSlice` (`:206-215`)
- `internal/importer/importer_test.go` — leaves in `"round trip"` (`:38`) and `"edges"` (`:140`)
- `internal/export/export_test.go` — leaves in `"diagram json"` (`:1544`)

**Patterns to Follow:**
- Node metadata: the trigger node (`internal/export/export.go:788-802`) and the translation node
  (`:844-882`) both set the shared `Reads` field on `jsonDiagramNode` (`:649-668`); no new struct
  field is needed, and `diagramNode` (`internal/importer/importer.go:25-42`) already decodes the key
- Importer side: the trigger (`internal/importer/importer.go:166-173`) and translation (`:217-234`)
  branches of `buildSlice`
- The byte-level leaf shape, and why a hand-written source rather than a fixture:
  `"preserves slices declared directly under a context"` (`internal/importer/importer_test.go:63-88`)
  and `"preserves a translation without duplicating its nested event"` (`:90-117`), both comparing
  `formatter.Format(importFrom(t, source))` against the source itself; `importFrom` (`:26-34`) is the
  export → import path the viewer uses
- The diagram-document differentials that prove a value reached one surface and not another:
  `"a model stating specs still produces a document free of them"`
  (`internal/export/export_test.go:2925-2939`)
- Leave `foldEdges` (`internal/importer/importer.go:251-287`) and the automation edge block
  (`internal/export/export.go:1062-1091`) alone — US-005 owns the edge in both directions
- Name an input that would make each assertion fail before writing it — `tasks/learnings.md` "An
  assertion whose expected value comes from the code under test is the recurring review finding"

**Testable:** Yes — through `export.ExportDiagramJSON` and `importer.ImportDiagram`, both exported.

**Verification:** `mise exec -- go test -tags unit ./internal/export/... ./internal/importer/...`;
`mise exec -- go test -tags unit ./...`.

**Depends on:** 1, 4, 5

---

### Task 8: Show the view an automation reads in the viewer's details panel

**Behavior:** selecting an automation in the viewer shows the view it reads in the automation section
of the details panel, beside the activation event, the command and the target context, and shows the
same placeholder the other rows use when the automation reads nothing.

**Acceptance Criteria:**
- [ ] The automation section of the details panel shows a `Reads` row carrying the node's view name,
      in the row shape the trigger (`internal/viewer/static/ui.js:337`) and translation (`:367`)
      sections use, and the row sits between the activation event and the command
- [ ] An automation node carrying no view name shows the same em-dash placeholder the section's other
      rows show, rather than an empty cell or a missing row
- [ ] The view name is escaped on the way into the panel, the way every other row's value is
- [ ] The rows already in the section — activation event, command, target context — still show what
      they showed, so the assertion covers the whole section rather than the new row alone
- [ ] The panel test drives `UI.showDetailPanel` through the exported `UI` object with a store the
      test builds, and passes under `mise exec -- task test:viewer`

**Affected Files/Modules:**
- `internal/viewer/static/ui.js` — the automation section of `showDetailPanel` (`:351-360`)
- `internal/viewer/tests/` — a new `*.test.js` file for the details panel; no test covers
  `showDetailPanel` today

**Patterns to Follow:**
- The row to add: the trigger section (`internal/viewer/static/ui.js:331-340`) and the translation
  section (`:362-376`), both of which read the same `reads` key off the node object
- The test harness: `internal/viewer/tests/visibility.test.js:1-41` — `installSVGGeometry()` from
  `tests/svg-env.js`, the dynamic `await import('../static/ui.js')`, and a `createStore` helper that
  builds the DOM elements the function under test reaches for. `showDetailPanel` reads
  `store.dom.detailPanel`, `store.dom.dpContent` and `store.dom.svg`, and writes
  `store.interaction.selectedNodeId`
- Assert what the panel shows, not how the HTML is built — the caller here is the person reading the
  panel (`~/.config/ai/guidelines/testing/caller-patterns.md`, the UI pattern: assert visible
  content, never markup structure)
- The node object is what the viewer receives from `export.ExportDiagramJSON`, so the test's node
  literal spells the same keys that document uses (`internal/export/export.go:649-668`)
- No Go change: this task edits `ui.js` and adds one test file. The exporter's key is Task 7's

**Testable:** Yes — through the exported `UI.showDetailPanel`, under vitest's jsdom environment
(`internal/viewer/vitest.config.js`).

**Verification:** `mise exec -- task test:viewer`; `git status --porcelain` lists changes under
`internal/viewer` only.

**Depends on:** 7

---

## Summary

**Eight tasks.**

**Ordering rationale — dependency-first, then blast radius.** Task 1 puts the entry into the language,
because nothing else can be written or read until the parser records it. Task 2 makes the name mean
something, so the fixture built next can only be written with a name that resolves. Task 3 follows
Task 1 immediately and stands alone: the tree-sitter grammar must never reject a file `emod validate`
accepts, so closing the gap it opens is the next thing to do. Task 4 lands the shared
fixture and its twin, which every remaining task needs as its featured model, and pays the diagram
byte-identical receipt while the diagram code is still untouched. Tasks 5 and 6 are the two writer
families, ordered formatter-first because the formatter is the one that silently *deletes* an entry it
has not heard of. Task 7 needs Task 5 in place, because its byte-level receipt is measured in
formatted bytes. Task 8 is the one JavaScript slice, and it comes after Task 7 because the value it
displays is the one the exporter starts emitting there.

**Story acceptance criteria coverage:**

| Story criterion | Task |
|---|---|
| An `automation` accepts an optional `reads <ViewName>` entry | 1 (with 3 mirroring it in the grammar) |
| The name must resolve to a view declared anywhere in the model; an unresolved name is a validation error naming the view and its location | 2 |
| `emod fmt` emits `reads` at a fixed position within the automation block, so repeated formatting is stable | 5 |
| `emod export -f json` and `-f cue` carry the value | 6 |
| A model exported, edited in the viewer, and re-imported keeps its `reads` value | 7, with 8 showing it in the details panel |
| An automation without `reads` validates, formats and exports exactly as before | 1, 2, 4, 5, 6, 7 — each task carries the criterion for its own surface, and Task 4 is the differential receipt |

**Nothing deferred from this story's criteria.** Deliberately left to later stories in the feature, and
therefore stale in the working tree until then: `docs/dsl-reference.md` — the Automation Pattern block
(`:322-340`), the entry list beneath it, and the cross-reference row for `view <Name>` (`:632`), which
lists a trigger's and a translation's `reads` but will not list an automation's — plus
`examples/all_patterns.emod`, all owned by US-011; the `reads` edge in SVG, draw.io and the viewer,
owned by US-005; `automation/missing-todo-list`, owned by US-008; LSP hover, completion and navigation,
owned by US-009.

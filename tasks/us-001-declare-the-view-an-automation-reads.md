# US-001: Declare the view an automation reads

## Progress
- [x] Task 1: Parse `reads` inside an automation block
- [x] Task 2: Reject an automation `reads` that names no declared view
- [x] Task 3: Accept an automation's `reads` in the tree-sitter grammar
- [x] Task 4: Share a fixture whose automations read views
- [ ] Task 5: Emit `reads` from `emod fmt` at a fixed position in the automation block
- [ ] Task 6: Carry automation `reads` into JSON, CUE and the embedded schema
- [ ] Task 7: Keep automation `reads` across the viewer round trip

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

**Consequences of that boundary, decided.** Three shapes the story does not spell out:

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
after the activation event and ahead of the command, matching where a translation's `reads` sits —
whatever position it was written in. Formatting the same model twice produces identical bytes, and a
parse → format → reparse cycle recovers the value rather than dropping it. An automation with no
`reads` formats to exactly the bytes it formats to today.

**Acceptance Criteria:**
- [ ] Formatting an automation that declares `reads` emits the entry on its own line, indented with
      the block's other entries, between the activation event and the command
- [ ] An automation whose source writes `reads` first, and one that writes it after `target context`,
      format to identical bytes
- [ ] Formatting the formatter's own output produces byte-identical bytes on the second pass
- [ ] Parsing the fixture from Task 4, formatting it, and reparsing yields a model whose automation
      `reads` values equal the transcribed list from Task 4 — the guard that the formatter did not
      silently delete an entry it has not heard of
- [ ] Formatting `test.HotelReservation`, `test.DescribedHotelReservation`,
      `test.KeywordFieldSearchCatalog`, `test.InvariantLibraryLending` and `test.SpecLibraryLending`
      produces the bytes it produces before this task, with no expected constant in
      `internal/formatter/formatter_test.go` or `internal/cli/fmt_test.go` edited
- [ ] The formatter's expected output in any new subtest is a canonical form written out in the test,
      not the input fixture re-indented

**Affected Files/Modules:**
- `internal/formatter/formatter.go` — `writeAutomation` (`:340-354`)
- `internal/formatter/formatter_test.go` — subtests in `"element formatting"` (`:32`) and a
  per-fixture leaf in `"round-trip through the parser"` (`:428`, beside `:497` and `:505`)

**Patterns to Follow:**
- The line to emit and where it sits: `writeTranslation` (`internal/formatter/formatter.go:356-373`)
  and `writeTrigger` (`:197-208`)
- A new block entry goes after `description` and ahead of nested blocks in every writer —
  `tasks/learnings.md` "A new block entry goes after `description` and ahead of nested blocks, in
  every writer", including why the formatter is the writer that hurts to forget
- The parse → format → reparse comparison against the original model is the guard, not idempotence
  and not an existing golden: `internal/formatter/formatter_test.go:428-503`
- Every expected string starts with `emod <n>` — `tasks/learnings.md` "Formatter output always begins
  with `emod N`"
- A fmt golden is never the input re-indented — `tasks/learnings.md` "`emod fmt` canonicalises order,
  so a fmt golden is never the input re-indented"
- No `internal/cli` change: `RunFmt` writes whatever `formatter.Format` returns

**Testable:** Yes — through `formatter.Format`, `parser.Parse` and the exported `internal/test`
helpers.

**Verification:** `mise exec -- go test -tags unit ./internal/formatter/... ./internal/cli/...`;
`go run ./cmd/emod fmt --check examples/all_patterns.emod` behaves as before.

**Depends on:** 1, 4

---

### Task 6: Carry automation `reads` into JSON, CUE and the embedded schema

**Behavior:** `export.ExportJSON` and `export.ExportCUE` carry an automation's `reads` value and the
position it was read from, and `internal/cue/schema.cue` declares the key so `emod schema` describes
it and `cue vet -d '#Model'` accepts a document that has it. A model whose automations declare no
`reads` produces byte-identical output in both formats.

**Acceptance Criteria:**
- [ ] The JSON export of the Task 4 fixture carries every automation `reads` value, keyed the way the
      trigger and translation documents key theirs, together with the position of the view name; the
      key order inside the automation object matches its `json*` siblings rather than the schema's
- [ ] The CUE export of the same fixture carries the same values, emitted after the activation event
      and ahead of the command
- [ ] `#Automation` in `internal/cue/schema.cue` declares the key as optional, and `emod schema`
      prints it
- [ ] `internal/cue/embed_test.go`'s `fullModelJSON` (`:126`) gives its automation the key, so the
      "accepts a model using every element the language offers" subtest (`:66`) exercises it, and
      removing the key from the schema makes a subtest fail
- [ ] A leaf in `"output conforms to the schema's Model definition"`
      (`internal/export/export_test.go:3442-3456`) runs `cue vet -d '#Model'` over the Task 4
      fixture's export and passes
- [ ] A leaf in `"CUE and JSON exports describe the same model"` (`:3458-3477`) requires the two
      exports of the Task 4 fixture equal, and a further assertion reads the `reads` values back out
      of the decoded document and compares them to the transcribed list from Task 4 — so a list
      neither writer emits cannot agree trivially
- [ ] The JSON and CUE exports of `test.HotelReservation` are byte-identical to what they were before
      this task, with no existing expected constant in `internal/export/export_test.go` edited
- [ ] The diagram JSON document is not changed by this task, and the existing subtest that walks it
      still passes unedited

**Affected Files/Modules:**
- `internal/export/export.go` — `jsonAutomation` (`:158-171`), `convertAutomation` (`:596-614`),
  `cueWriter.writeAutomation` (`:1363-1370`)
- `internal/cue/schema.cue` — `#Automation` (`:54-61`)
- `internal/cue/embed_test.go` — `fullModelJSON` (`:126`)
- `internal/export/export_test.go` — leaves in the `"model json"` (`:21`), `"cue"` (`:3074`), schema
  conformance (`:3442`) and export-parity (`:3458`) groups

**Patterns to Follow:**
- Field, key name and struct ordering: `jsonTranslation` (`internal/export/export.go:173-187`) and
  `jsonTrigger` (`:132-145`) — positions first, then values, and copy the ordering from a `json*`
  sibling, never from the schema (`tasks/learnings.md` "JSON and CUE order their document keys
  differently — do not mirror one struct into the other")
- CUE emission: `cueWriter.writeTranslation` (`internal/export/export.go:1372-1382`) using
  `lineIfSet`
- All three surfaces land together — `tasks/learnings.md` "A new exported field must land in JSON, CUE
  and `schema.cue` in the same change"
- The two coupled guards cannot see a value neither writer emits, so read the values back out of the
  decoded document against a hand-transcribed list: `listsKeyedBy`
  (`internal/export/export_test.go:3912`) and `tasks/learnings.md` "The two export guards cannot see a
  list neither writer emits"
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
`go run ./cmd/emod schema` prints the key.

**Depends on:** 1, 4

---

### Task 7: Keep automation `reads` across the viewer round trip

**Behavior:** the diagram JSON document carries an automation's `reads` value as node metadata, the
importer reads it back onto the automation, and the viewer's details panel shows it beside the
automation's other entries. A model exported to diagram JSON, loaded and re-exported through the
viewer's path, comes back with its `reads` values intact. No `reads` edge is drawn for an automation —
that is US-005's.

**Acceptance Criteria:**
- [ ] The automation node in `export.ExportDiagramJSON`'s output carries the `reads` value under the
      same key the trigger and translation nodes use
- [ ] `importer.ImportDiagram` sets the value back onto the automation it builds
- [ ] Parsing the Task 4 fixture, exporting to diagram JSON, re-importing and formatting reproduces
      the formatted source of the original model with comments stripped — the round trip in
      `internal/importer/importer_test.go:38-138` extended with a leaf for this fixture
- [ ] An automation with no `reads` round-trips to an automation with no `reads`, and the existing
      round-trip leaves in `internal/importer/importer_test.go` pass unedited
- [ ] The edge list `export.ExportDiagramJSON` produces for the Task 4 fixture contains no `reads`
      edge whose target is an automation, and a viewer-drawn view→automation edge is not folded into a
      `reads` entry — both stay US-005's, and the subtest says so in its failure message
- [ ] `internal/viewer/static/ui.js` prints a `Reads` row in the automation section of the details
      panel, matching the trigger (`:337`) and translation (`:367`) rows
- [ ] `mise exec -- task test:viewer` passes

**Affected Files/Modules:**
- `internal/export/export.go` — the automation diagram node (`:825-842`)
- `internal/importer/importer.go` — the automation branch of `buildSlice` (`:206-215`)
- `internal/importer/importer_test.go` — a leaf in `"round trip"` (`:38`)
- `internal/export/export_test.go` — a leaf in `"diagram json"` (`:1544`)
- `internal/viewer/static/ui.js` — the automation section of the details panel (`:350-359`)

**Patterns to Follow:**
- Node metadata: the trigger node (`internal/export/export.go:788-802`) and the translation node
  (`:844-882`) both set the shared `Reads` field on `jsonDiagramNode` (`:649-668`); no new struct field
  is needed
- Importer side: the trigger (`internal/importer/importer.go:166-173`) and translation (`:217-234`)
  branches of `buildSlice`
- The round-trip assertion shape, including why comments are stripped from the baseline:
  `importFrom` and `"reproduces the formatted source for every documented pattern"`
  (`internal/importer/importer_test.go:26-50`)
- Details-panel row: the trigger and translation sections of `internal/viewer/static/ui.js`
  (`:331-340`, `:361-375`)
- Leave `foldEdges` (`internal/importer/importer.go:251-287`) and the automation edge block
  (`internal/export/export.go:1062-1091`) alone — US-005 owns the edge in both directions
- Name an input that would make each assertion fail before writing it — `tasks/learnings.md` "An
  assertion whose expected value comes from the code under test is the recurring review finding"

**Testable:** Yes — through `export.ExportDiagramJSON` and `importer.ImportDiagram`, both exported,
plus the viewer's vitest suite for the panel row.

**Verification:** `mise exec -- go test -tags unit ./internal/export/... ./internal/importer/...`;
`mise exec -- task test:viewer`.

**Depends on:** 1, 4, 5

---

## Summary

**Seven tasks.**

**Ordering rationale — dependency-first, then blast radius.** Task 1 puts the entry into the language,
because nothing else can be written or read until the parser records it. Task 2 makes the name mean
something, so the fixture built next can only be written with a name that resolves. Task 3 follows
Task 1 immediately and stands alone: the tree-sitter grammar must never reject a file `emod validate`
accepts, so closing the gap it opens is the next thing to do. Task 4 lands the shared
fixture and its twin, which every remaining task needs as its featured model, and pays the diagram
byte-identical receipt while the diagram code is still untouched. Tasks 5 and 6 are the two writer
families, ordered formatter-first because the formatter is the one that silently *deletes* an entry it
has not heard of. Task 7 comes last because its round trip is measured in formatted bytes and so needs
Task 5 in place.

**Story acceptance criteria coverage:**

| Story criterion | Task |
|---|---|
| An `automation` accepts an optional `reads <ViewName>` entry | 1 (with 3 mirroring it in the grammar) |
| The name must resolve to a view declared anywhere in the model; an unresolved name is a validation error naming the view and its location | 2 |
| `emod fmt` emits `reads` at a fixed position within the automation block, so repeated formatting is stable | 5 |
| `emod export -f json` and `-f cue` carry the value | 6 |
| A model exported, edited in the viewer, and re-imported keeps its `reads` value | 7 |
| An automation without `reads` validates, formats and exports exactly as before | 1, 2, 4, 5, 6, 7 — each task carries the criterion for its own surface, and Task 4 is the differential receipt |

**Nothing deferred from this story's criteria.** Deliberately left to later stories in the feature, and
therefore stale in the working tree until then: `docs/dsl-reference.md` — the Automation Pattern block
(`:322-340`), the entry list beneath it, and the cross-reference row for `view <Name>` (`:632`), which
lists a trigger's and a translation's `reads` but will not list an automation's — plus
`examples/all_patterns.emod`, all owned by US-011; the `reads` edge in SVG, draw.io and the viewer,
owned by US-005; `automation/missing-todo-list`, owned by US-008; LSP hover, completion and navigation,
owned by US-009.

# US-002: Describe constructs where they are declared

## Progress
- [x] Task 1: Carry a `description` attribute on the nine block constructs
- [x] Task 2: Give `actor` and `model` an optional block form holding a description
- [x] Task 3: Preserve descriptions through `emod fmt`
- [x] Task 4: Carry descriptions through the JSON and CUE exports and the embedded schema
- [x] Task 5: Attach descriptions as draw.io tooltips
- [ ] Task 6: Attach descriptions as SVG `<title>` elements
- [ ] Task 7: Accept `description` and the new block forms in the tree-sitter grammar
- [ ] Task 8: Document `description` in the DSL reference

---

## Story Reference

`user-stories/specs-and-metadata.md` → **US-002: Describe constructs where they are declared**
(second story of the "Specs, Invariants, and Model Metadata" feature). Design notes live in
`docs/specs-and-metadata-proposal.md` §2 "Descriptions and Glossary" (lines 52–101), the grammar
notes at lines 374–392, and the phase-1 summary at line 500.

**In scope:** the `description` string attribute on `context`, `aggregate`, `slice`, `command`,
`event`, `view`, `automation`, `translation` and `trigger`; an optional block form for `actor` and
`model` carrying the same attribute; pass-through in `emod export -f json` and `-f cue`; draw.io
tooltips and SVG `<title>` elements; and no change in behaviour for files that use no descriptions.

**Out of scope (resolved for this run):** `emod glossary` (US-004); field-level descriptions (an
explicit feature non-goal); highlight scopes in `editors/vscode/syntaxes/emod.tmLanguage.json` and
`editors/tree-sitter-emod/queries/highlights.scm` (US-017); rewriting `examples/*.emod` (US-018);
LSP hover, completion and go-to-definition for descriptions (US-015); the `diagram-json` export
consumed by the web viewer (no acceptance criterion mentions it, and its node/edge shape is a
separate contract); mermaid and ascii diagram formats (the story names draw.io and SVG only);
generated tree-sitter `src/` output, which stays gitignored.

**Deliberately pulled in:** Task 3 (formatter). The story's criteria do not mention `emod fmt`, and
US-014 owns formatter work for the feature as a whole — but US-014 is twelve stories away, and until
it lands a single `emod fmt` run would silently delete every description an author writes, hollowing
out the pass-through criteria this story does own. Task 3 implements only the `description` slice of
US-014's canonical ordering rule ("`description` first, then pattern-specific attributes, then
`fields`"); spec blocks, `given`/`when`/`then` alignment, flow colon alignment and payload wrapping
stay with US-014.

**Learnings folded in** from `tasks/learnings.md`: the keyword-as-field-name pattern and its two
required test cases; the single `keywords` map (not two switch cases); the ordinal-range trap in
`internal/lsp/hover.go:37`; `mise exec --` for `task test:grammar`; generated tree-sitter `src/`
stays untracked; acceptance criteria never reference commit or branch state; formatter output always
opens with `emod N`; `docs/dsl-reference.md` anchors embed the section number.

---

## Codebase Context

**Pipeline.** `internal/oracle/oracle.go:14-24` is the canonical lex → parse → validate → lint chain.
`internal/cli/validate.go`, `internal/cli/export.go:47-57`, `internal/cli/diagram.go:49-57`,
`internal/lsp/server.go` and `internal/wasm/pipeline.go` each rebuild the same chain, so anything
that must hold everywhere belongs in the lexer or parser, never in a caller.

**Lexer.** `internal/lexer/token.go:5-57` declares `Kind` as one iota block — keywords first
(`KeywordModel` … `KeywordEmod`), then `Identifier`, `String`, `Integer`, then punctuation. The
keyword string ↔ `Kind` mapping is the single `keywords` map at `:59-89`, inverted once into
`keywordNames` at `:91-99`; `Kind.String()` (`:108-147`) consults that inversion before its switch,
so a new keyword is one map entry and no switch case. `internal/lexer/tokenizer.go:203` is the only
lookup site.

**Keyword ordering constraint.** `internal/parser/parser.go:1283-1291` (`checkIdentifierLike`) treats
any `Kind` ordered **before** `Identifier` as valid in field-name position; `parseFields`
(`:1030-1058`), `parseField` (`:1060-1086`) and `parseTagEntry` (`:1118`) all gate on it. That is why
`fields { source string }` works today and why a new keyword must be declared inside the keyword
block of the iota. Separately, `internal/lsp/hover.go:37` (`isKeyword`) is an *ordinal range*
(`KeywordModel … KeywordExternal`), so a `Kind` appended after `KeywordEmod` is invisible to hover —
which is the correct outcome here, since hover for descriptions belongs to US-015.

**Parser shape.** Every block construct follows the same loop: consume the keyword, read the name,
consume `{`, then `for !p.check(lexer.CloseBrace) && !p.isAtEnd()` dispatching on the leading keyword
of each entry, with an `else` branch that reports "expected X, Y or Z in <construct>" and advances
one token. Sites: `parseContext` (`:180-236`), `parseAggregate` (`:238-279`), `parseSlice`
(`:281-344`), `parseTrigger` (`:346-411`), `parseCommand` (`:413-454`), `parseEvent` (`:693-754`),
`parseView` (`:756-806`), `parseAutomation` (`:808-898`), `parseTranslation` (`:900-994`). The
"keyword followed by a quoted string" attribute already exists as the `source external "<name>"`
branch at `:718-737`, including its error recovery. Unclosed blocks report
`unclosed brace for "<construct>" block opened at line N`.

`parseModel` (`:155-164`) and `parseActor` (`:166-178`) are the two single-line declarations; both
are reached through the `handlers` map built in `New` (`:36-58`) and dispatched by the top-level loop
in `Parse` (`:71-87`). A `{` can never follow a top-level declaration today, so lookahead on `{` is
unambiguous for the new block form.

**AST.** `internal/ast/ast.go` pairs every value with its source position — `Context.Mode`/`ModePos`
(`:37-38`), `Event.Source`/`SourcePos` (`:84-85`), `Trigger.Reads`/`ReadsPos` (`:119-120`). Blocks
also carry `OpenPos`/`ClosePos`. `ast.Model` (`:17-25`) and `ast.Actor` (`:27-31`) currently have
neither, since they are single-line. `internal/parser` and `internal/formatter` both import `ast`;
`formatter` does not import `parser`.

**Validator and linter are name-and-reference oriented.** `internal/validator/validator.go:12` walks
names, tags and `decides_on` references only, and `internal/linter/linter.go` rules key off naming
and cardinality. Neither reads a new descriptive string, so no rule fires or stops firing because a
description is present.

**Formatter.** `internal/formatter/formatter.go` writes the whole tree from `writeModel` (`:39-54`),
which unconditionally emits `emod %d` first. Per-construct writers: `writeContext` (`:63-85`),
`writeAggregate` (`:87-97`), `writeSlice` (`:99-169`), `writeTrigger` (`:171-181`), `writeCommand`
(`:183-193`), `writeEvent` (`:234-247`), `writeView` (`:298-308`), `writeAutomation` (`:310-323`),
`writeTranslation` (`:325-341`). Existing coverage includes round-trip
(`internal/formatter/formatter_test.go:398-438`), idempotency (`:1219-1228`), canonical element
ordering inside a slice (`:595-657`) and comment preservation (`:1201-1218`). Byte-for-byte
downstream fixtures: `internal/cli/fmt_test.go`, `internal/importer/importer_test.go:86,114,133`, and
`e2e-viewer/tests/helpers.js` (the viewer export path is wasm → `formatter.Format`).

**JSON export.** `internal/export/export.go:27-166` declares one `json*` struct per AST node, all
using `omitempty` for optional values, converted by the `convert*` functions at `:244-620`.
`ExportJSON` (`:239`) is the `-f json` path; `ExportJSONDiagnostics` (`:186`) wraps it in the
`{diagnostics, model}` envelope the CLI prints. Positions are exported selectively — `jsonEvent` has
`source_position` but `jsonView` exports `subscribes` with no positions — so position export is a
judgement call per field, not a rule. `ExportDiagramJSON` (`:633`) is a completely separate
node/edge document for the web viewer and is not part of this story.

**CUE export and schema.** `ExportCUE` (`internal/export/export.go:1090-1095`) drives `cueWriter`
(`:1097-1450`), one `write*List` per construct, each emitting optional keys only when non-empty. The
target shape is `internal/cue/schema.cue`, embedded by `internal/cue/embed.go` and printed by
`emod schema -f cue` (`internal/cli/schema.go`). `internal/cue/embed_test.go` shells out to the real
`cue` binary (skipping when absent) and vets `fullModelJSON` (`:117-161`) against `#Model`; that
constant is deliberately the model that "exercises every definition the schema declares".

**Diagram renderers.** `internal/diagram/drawio.go` and `internal/diagram/svg.go` both consume
`collectSlices` (`drawio.go:678-700`), whose `sliceEntry` carries only `slice`, `ctxName` and
`fromDCB` — a context's description is not reachable from it today. Shapes are emitted by
`vertexCell` (`drawio.go:792-796`, value escaped through `escapeXML` at `:818-826`) and by
`svgRect`/`svgRoundedRect`/`svgDashedRoundedRect` (`svg.go:383-395`) followed by a separate
`svgText`/`svgMultilineText` line. Constructs that own a shape: context label
(`drawio.go:286-293`, `svg.go:81-84`), trigger, command, view, event, translation event, automation,
translation reactor, external system (`drawio.go:355-513`, `svg.go:122-245`). Aggregates and slices
own no shape in either renderer — they are layout only. draw.io has three styles (`StyleAuto`,
`StyleProjected`, `StyleDCB`, `drawio.go:15-22`) that share these emitters.

**Diagram test contract.** `internal/diagram/contract_test.go:46-114` asserts shared behaviour across
all four exporters and locates shapes textually: `drawioFillOfLabel` (`:82-93`) matches a
`<mxCell … value="…" style="…">` line, and `svgFillOfLabel` (`:95-114`) walks backwards from the
`<text>` line to the nearest `<rect … fill="…">` line. Both are sensitive to changes in the emitted
element shape, so anything wrapping or nesting a shape must leave description-free output untouched.

**Tree-sitter grammar.** `editors/tree-sitter-emod/grammar.js` is a deliberately looser mirror of the
Go grammar (it has no `mode`, `tags` or `decides_on`). `word: $ => $.identifier` where `identifier`
is `/[A-Z][a-zA-Z0-9_]*/`, and `any_identifier` (`:215`) is the permissive
`/[a-zA-Z_][a-zA-Z0-9_]*/` used in field-name, subscribes and attribute-value positions.
`version_header` (`:21-24`) shows how a keyword token is narrowed so it cannot match outside its own
position. Corpus tests live in `test/corpus/`; `version_header.txt:31-58` is the "Field named emod
inside a fields block" case. `task test:grammar` runs `tree-sitter generate` before `tree-sitter
test`, and `editors/tree-sitter-emod/.gitignore` keeps `src/` untracked.

**Shared fixture.** `internal/test/fixtures.go:7-89` (`HotelReservation`) is the "realistic model"
every stage's tests share. It has no descriptions, which makes it the regression guard for the
"exactly as before" criterion — it must not gain any.

**Test conventions** (`CLAUDE.md` "Go Test Organization"): one umbrella `Test{TypeName}` per type,
`t.Run` groups named after the operation, leaf subtests reading as sentences about the observed
outcome, `testify/require`, fresh fixtures per leaf, `//go:build unit` / `//go:build integration`
tags. AST comparisons use `test.RequireEqual` with `cmpopts.IgnoreTypes(ast.Position{})`
(`internal/parser/parser_test.go:20`).

---

## Tasks

### Task 1: Carry a `description` attribute on the nine block constructs

**Behavior:** Any of `context`, `aggregate`, `slice`, `command`, `event`, `view`, `automation`,
`translation` and `trigger` may contain `description "<text>"` inside its block, and the parsed
construct carries that text. The attribute is optional everywhere, `description` stays usable as a
field name, and a file that uses no descriptions parses to the same AST and the same diagnostics as
before.

**Acceptance Criteria:**
- [x] Each of the nine block constructs accepts `description "<text>"` among its entries, in any
      position within the block, and the parsed construct exposes that text; parsing such a model
      produces zero diagnostics
- [x] The description is recorded with its source position, matching the value-plus-position
      convention of `Context.Mode`/`ModePos` and `Event.Source`/`SourcePos`
- [x] The `event` nested inside a `translation` accepts a description on the same terms as a
      top-level event
- [x] `fields { description string required }` still parses as an ordinary field named
      `description`, asserted as a case alongside the existing "emod is usable as a field name"
      subtest in `internal/parser/parser_test.go:224-245`
- [x] `description` followed by something other than a quoted string produces exactly one diagnostic
      that names the construct and the offending token, positioned at the offending token, and the
      enclosing block still parses to completion and reports its remaining contents
- [x] `internal/test.HotelReservation` and every fixture under `internal/parser/testdata/` parse to
      the same AST and the same diagnostics as before this task, and `HotelReservation` gains no
      descriptions
- [x] `oracle.Check` on a model carrying a description on all nine constructs returns zero
      diagnostics, so `emod validate` on that model is clean
- [x] A described counterpart fixture (alongside, not replacing, `HotelReservation`) exists in
      `internal/test/fixtures.go` carrying a description on every construct that accepts one, so the
      export and diagram tasks can assert against one shared model

**Affected Files/Modules:**
- `internal/lexer/token.go` — new keyword `Kind` inside the iota keyword block plus its `keywords`
  map entry
- `internal/lexer/tokenizer_test.go` — coverage that the new word scans as a keyword rather than an
  identifier
- `internal/ast/ast.go` — description text and position on `Context`, `Aggregate`, `Slice`,
  `Command`, `Event`, `View`, `Automation`, `Translation`, `Trigger`
- `internal/parser/parser.go` — a `description` branch in each of the nine block loops
- `internal/parser/parser_test.go` — per-construct capture, the field-named-`description` case, the
  malformed-attribute diagnostic, backward compatibility
- `internal/test/fixtures.go` — the described counterpart fixture

**Patterns to Follow:**
- One `keywords` map entry plus the existing inversion — `internal/lexer/token.go:59-99`; `Kind.String()`
  needs no new switch case (`tasks/learnings.md`, "the keyword string/Kind mapping is a single map")
- Declare the new `Kind` inside the iota keyword block, appended after `KeywordEmod`: before
  `Identifier` so `checkIdentifierLike` (`internal/parser/parser.go:1283-1291`) accepts it in
  field-name position, and after `KeywordExternal` so it stays outside the ordinal range
  `internal/lsp/hover.go:37` uses — hover, completion and TextMate scopes belong to US-015/US-017
  (`tasks/learnings.md`, "New DSL keywords must stay usable as field names" and "Keyword surfaces fan
  out past the lexer, parser and tree-sitter grammar")
- Attribute parsing and error recovery for "keyword then quoted string": the `source external` branch
  at `internal/parser/parser.go:718-737`
- Block-loop structure and the "expected …, … or … in <construct>" fallback: `parseView`
  (`internal/parser/parser.go:756-806`) is the smallest complete example
- Value-plus-position AST convention: `internal/ast/ast.go:33-43` and `:80-92`
- Test layout: `internal/parser/parser_test.go:22-60`; `CLAUDE.md` "Go Test Organization"
- Caller pattern **Inbound** (`~/.config/ai/guidelines/testing/caller-patterns.md`): the source text
  is the input, the `(*ast.Model, diagnostics)` pair is the observable outcome — assert acceptance,
  rejection and the resulting AST, never parser internals. Unit of behavior per
  `~/.config/ai/guidelines/go/testing-patterns.md`
- Assert the distinguishing content of the new diagnostic (construct name and offending token), not
  just its position or count (`tasks/learnings.md`, "CLI diagnostic tests must assert the
  distinguishing message text")

**Testable:** Yes — `lexer.Scan`, `parser.Instance.Parse` and `oracle.Check` are exported and already
have suites.

**Verification:** `mise exec -- task test:unit` and `mise exec -- task test:integration` pass.

**Depends on:** None

---

### Task 2: Give `actor` and `model` an optional block form holding a description

**Behavior:** `actor "<name>" { description "<text>" }` and `model "<name>" { description "<text>" }`
parse and carry the description. The existing single-line forms remain valid and unchanged.

**Acceptance Criteria:**
- [ ] `actor "<name>" { description "<text>" }` parses with zero diagnostics into an actor carrying
      the description
- [ ] `model "<name>" { description "<text>" }` parses with zero diagnostics into a model carrying
      the description
- [ ] `actor "<name>"` and `model "<name>"` with no block still parse with zero diagnostics, produce
      an empty description, and yield the same AST as before this task
- [ ] An empty block (`{ }`) on either construct parses with zero diagnostics, matching how a trigger
      with an empty body behaves today
- [ ] A block form whose brace is never closed reports the existing
      `unclosed brace for "<construct>" block opened at line N` diagnostic naming that construct
- [ ] An entry other than `description` inside either block produces one diagnostic naming the
      construct and the offending token, and the parse recovers to the closing brace
- [ ] A top-level declaration following a single-line `actor` is still recognised — an `actor`
      followed on the next line by `context`, `model` or another `actor` produces no diagnostics
- [ ] `internal/test.HotelReservation` (single-line `model` and `actor`) parses to the same AST and
      the same diagnostics as before this task

**Affected Files/Modules:**
- `internal/ast/ast.go` — description text and position on `Model` and `Actor`, plus block positions
  consistent with the other block constructs
- `internal/parser/parser.go` — optional block after the name in `parseModel` and `parseActor`
- `internal/parser/parser_test.go` — both block forms, both single-line forms, empty block, unclosed
  block, unexpected entry, and the declaration-follows-actor case

**Patterns to Follow:**
- `parseModel` and `parseActor` as they stand today, and the top-level `handlers` dispatch that
  reaches them: `internal/parser/parser.go:36-58`, `:71-87`, `:155-178`
- Block-loop and unclosed-brace reporting: `parseTrigger` (`internal/parser/parser.go:346-411`),
  which also demonstrates a block whose body may legitimately be empty
- The description branch introduced in Task 1 — the same attribute, parsed the same way
- Caller pattern **Inbound**; test layout per `CLAUDE.md` "Go Test Organization"

**Testable:** Yes — through `parser.Instance.Parse`.

**Verification:** `mise exec -- task test:unit` and `mise exec -- task test:integration` pass.

**Depends on:** Task 1

---

### Task 3: Preserve descriptions through `emod fmt`

**Behavior:** Formatting a model that carries descriptions emits them back, `description` first
inside each block, and uses the block form for a described `actor` or `model`. A model with no
descriptions formats byte-identically to before.

**Acceptance Criteria:**
- [ ] Formatting a model carrying a description on every construct that accepts one, re-parsing the
      output, and comparing the two ASTs yields equality — no description is lost, following the
      round-trip subtest at `internal/formatter/formatter_test.go:398-438`
- [ ] Inside every block, `description` is the first line, before pattern-specific attributes and
      before `fields`
- [ ] An `actor` or `model` that carries a description formats as the block form; one that does not
      keeps the single-line form
- [ ] Formatting is idempotent for a described model, matching the existing idempotency subtest at
      `internal/formatter/formatter_test.go:1219-1228`
- [ ] Formatting `internal/test.HotelReservation` and every existing formatter golden produces
      byte-identical output to before this task
- [ ] `RunFmt` in `--check` mode reports no change needed for a canonically formatted file that uses
      descriptions

**Affected Files/Modules:**
- `internal/formatter/formatter.go` — description emission in `writeModel`, `writeContext`,
  `writeAggregate`, `writeSlice`, `writeTrigger`, `writeCommand`, `writeEvent`, `writeView`,
  `writeAutomation`, `writeTranslation`, plus the actor loop in `writeModel`
- `internal/formatter/formatter_test.go` — per-construct output, block form for actor and model,
  round-trip and idempotency over a described model
- `internal/cli/fmt_test.go` — a `--check` fixture that uses descriptions

**Patterns to Follow:**
- Attribute emission order inside a block: `writeTrigger` (`internal/formatter/formatter.go:171-181`)
  and `writeEvent` (`:234-247`)
- Canonical ordering is already asserted for slice contents at
  `internal/formatter/formatter_test.go:595-657` — extend that notion, do not invent a second one
- Every expected string in a formatter or `RunFmt` golden must open with the version header line
  (`tasks/learnings.md`, "Formatter output always begins with `emod N`"); input fixtures may omit it
- US-014 owns the rest of the canonical order (spec blocks, `given`/`when`/`then` alignment, flow
  colon alignment, payload wrapping) — implement `description` placement only
- Caller pattern **Exported API** for `formatter.Format`, **Inbound** for `RunFmt`
  (`~/.config/ai/guidelines/testing/caller-patterns.md`)

**Testable:** Yes — `formatter.Format` and `cli.RunFmt` are exported.

**Verification:** `mise exec -- task test:unit` and `mise exec -- task test:integration` pass; the
viewer export fixtures in `e2e-viewer/tests/helpers.js` still match.

**Depends on:** Task 1, Task 2

---

### Task 4: Carry descriptions through the JSON and CUE exports and the embedded schema

**Behavior:** `emod export -f json` and `emod export -f cue` both emit each construct's description,
the embedded CUE schema declares the key as optional wherever it can appear, and a model with no
descriptions exports exactly as it does today.

The two formats are one task because `internal/export/export_test.go:3238-3256` asserts that the JSON
and CUE exports of `buildFullModel()` decode to the *same* document — teaching one format about
descriptions while the other stays ignorant breaks that parity test the moment the shared fixture
gains a description.

**Acceptance Criteria:**
- [ ] `export.ExportJSON` emits the description for the model, each actor, and each of the nine block
      constructs that carries one
- [ ] `export.ExportCUE` emits the same descriptions, and the JSON/CUE parity subtest still passes
      with `buildFullModel()` extended to carry a description on every construct that accepts one
- [ ] The key is absent — not empty-valued — for every construct with no description, so `ExportJSON`
      and `ExportCUE` on `internal/test.HotelReservation` both produce byte-identical output to
      before this task
- [ ] `RunExport` with format `json` on a described file prints the descriptions inside the `model`
      object of the `{diagnostics, model}` envelope, and the envelope shape is otherwise unchanged
- [ ] A description containing quotes, `\` and non-ASCII characters round-trips its exact text
      through both formats
- [ ] `internal/cue/schema.cue` declares an optional description on `#Model`, `#Actor`, `#Context`,
      `#Aggregate`, `#Slice`, `#Command`, `#Event`, `#View`, `#Automation`, `#Translation` and
      `#Trigger`
- [ ] `cue vet -d '#Model'` accepts a model document carrying a description on every one of those
      definitions — `fullModelJSON` in `internal/cue/embed_test.go:117-161` is extended so a dropped
      or misnamed key fails the existing acceptance subtest — and rejects one whose description is
      not a string
- [ ] The CUE export of a described model still conforms to `#Model`, keeping the
      "output conforms to the schema's Model definition" subtest green
- [ ] `emod schema -f cue` prints a schema containing the new optional key on each of those
      definitions, and the schema still imports nothing
- [ ] `diagram-json` output is unchanged for both described and description-free models

**Affected Files/Modules:**
- `internal/export/export.go` — a description field on `jsonModel`, `jsonActor`, `jsonContext`,
  `jsonAggregate`, `jsonSlice`, `jsonCommand`, `jsonEvent`, `jsonView`, `jsonAutomation`,
  `jsonTranslation`, `jsonTrigger` with their `convert*` functions, and the matching emission in each
  `cueWriter` `write*` function (`:1097-1450`)
- `internal/cue/schema.cue` — the optional key on each definition
- `internal/cue/embed_test.go` — extended `fullModelJSON`, plus a rejection case for a non-string
  description
- `internal/export/export_test.go` — per-construct serialization in both formats, omission when
  unset, special characters, and `buildFullModel()` (`:3340`) extended
- `internal/cli/export_test.go` — `-f json` on a described file
- `internal/cli/schema_test.go` — the schema the command prints

**Patterns to Follow:**
- JSON struct-tag and omission convention: `internal/export/export.go:27-166`, where every optional
  scalar uses `omitempty` and every `convert*` function copies fields one-for-one (`:244-620`)
- `jsonView.Subscribes` (`:136`) is the precedent for exporting a value without its position; follow
  it and export the description text only, in both formats
- CUE optional-key emission: `writeEvent`'s handling of `source` and `external_name`
  (`internal/export/export.go:1280-1306`)
- Schema style: `internal/cue/schema.cue:29-35` (`#Event`), where every optional scalar is declared
  with `?` and a `string` type
- The `cue` binary is optional in this environment — `requireCue` (`internal/cue/embed_test.go:19-28`)
  and `lookupCue` (`internal/export/export_test.go:3285`) skip when it is missing; do not add a hard
  dependency on it
- `fullModelJSON` and `buildFullModel()` are each documented as exercising every construct the
  exports cover, so extend those rather than adding parallel fixtures
- Assertion style: `internal/export/export_test.go:19-60` unmarshals into `map[string]any` and
  asserts on the decoded document, not on the raw byte string
- Caller pattern **Exported API** for `export.*` and **UI** for the CLI leaf — assert the data a
  consumer reads, not the serializer's traversal order
  (`~/.config/ai/guidelines/testing/caller-patterns.md`)

**Testable:** Yes — `export.ExportJSON`, `export.ExportJSONDiagnostics`, `export.ExportCUE`,
`cue.Schema`, `cli.RunExport` and `cli.RunSchema` are all exported.

**Verification:** `mise exec -- task test:unit` and `mise exec -- task test:integration` pass,
including the `cue vet` conformance and JSON/CUE parity subtests when the `cue` binary is available.

**Depends on:** Task 1, Task 2

---

### Task 5: Attach descriptions as draw.io tooltips

**Behavior:** A draw.io shape drawn for a construct that carries a description shows that description
as its tooltip. Shapes for constructs with no description are emitted exactly as before.

**Acceptance Criteria:**
- [ ] Every draw.io shape whose construct carries a description carries that text as the shape's
      tooltip, in all three styles (`StyleAuto`, `StyleProjected`, `StyleDCB`)
- [ ] Coverage spans every construct that owns a shape: the context label, trigger, command, view,
      event, translation event, automation, translation reactor and external system
- [ ] Aggregates and slices, which own no shape in this renderer, produce no extra cell — the cell
      count for a described model matches the cell count for the same model without descriptions
- [ ] A described shape keeps the same label, style and geometry it had before this task
- [ ] Shapes for constructs with no description are unchanged, so `ExportDrawio` on
      `internal/test.HotelReservation` produces byte-identical output to before this task in all
      three styles
- [ ] Output remains well-formed XML for a described model, and a description containing `<`, `&`
      and `"` appears escaped rather than breaking the document
- [ ] The shared exporter contract tests in `internal/diagram/contract_test.go` still locate shapes
      by label for both described and description-free models

**Affected Files/Modules:**
- `internal/diagram/drawio.go` — shape emission (`:286-293`, `:355-513`) and the cell builder
  (`:792-796`); `sliceEntry`/`collectSlices` (`:678-700`) needs the context's description reachable
- `internal/diagram/drawio_test.go` — tooltip per construct, three styles, escaping, unchanged output
  without descriptions
- `internal/diagram/contract_test.go` — only if a shared helper needs to keep resolving shapes

**Patterns to Follow:**
- draw.io/mxGraph carries a per-shape tooltip on an `<object …>` element wrapping the `mxCell`; the
  `mxCell` keeps its style and `mxGeometry`. Emit that wrapper only for shapes that have a
  description, so untouched output stays byte-identical and `drawioFillOfLabel`
  (`internal/diagram/contract_test.go:82-93`) keeps matching plain cells
- Escaping goes through `escapeXML` (`internal/diagram/drawio.go:818-826`); note the context label at
  `:289` already escapes before `vertexCell` escapes again, so do not copy that call shape
- `svg.go` consumes `collectSlices` too — any change to `sliceEntry` must keep that caller compiling
  and its output unchanged until Task 6
- Assertion style: `internal/diagram/drawio_test.go:16-80` builds a small `ast.Model` literal per
  subtest and asserts on substrings of the emitted XML plus `requireValidXML`
- Caller pattern **UI** (`~/.config/ai/guidelines/testing/caller-patterns.md`): the reader of the
  diagram is the caller — assert that the description text is present on the right shape, not the
  precise nesting of the XML beyond what draw.io requires to read it

**Testable:** Yes — `diagram.ExportDrawio` is exported, and `cli.RunDiagram` drives it.

**Verification:** `mise exec -- task test:unit` passes, including the shared exporter contract tests.

**Depends on:** Task 1

---

### Task 6: Attach descriptions as SVG `<title>` elements

**Behavior:** An SVG shape drawn for a construct that carries a description carries that description
as a `<title>` element, which browsers surface as the shape's tooltip. Shapes for constructs with no
description are emitted exactly as before.

**Acceptance Criteria:**
- [ ] Every SVG shape whose construct carries a description carries that text in a `<title>` element
      belonging to that shape
- [ ] Coverage spans every construct that owns a shape: the context label, trigger, command, view,
      event, translation event, automation, translation reactor and external system
- [ ] Aggregates and slices produce no extra element — the shape count for a described model matches
      the shape count for the same model without descriptions
- [ ] Shapes for constructs with no description are unchanged, so `ExportSVG` on
      `internal/test.HotelReservation` produces byte-identical output to before this task
- [ ] Output remains well-formed XML for a described model, and a description containing `<`, `&`
      and `"` appears escaped
- [ ] `svgFillOfLabel` (`internal/diagram/contract_test.go:95-114`) still resolves each shape's fill
      for both described and description-free models — its backwards walk from the `<text>` line to
      the nearest `<rect … fill="…">` line must keep finding the right rect

**Affected Files/Modules:**
- `internal/diagram/svg.go` — shape emission (`:81-84`, `:122-245`) and the rect builders
  (`:383-395`)
- `internal/diagram/svg_test.go` — `<title>` per construct, escaping, unchanged output without
  descriptions
- `internal/diagram/contract_test.go` — only if the shared fill helper needs to keep resolving shapes

**Patterns to Follow:**
- SVG `<title>` is a child of the element it titles, so a titled shape can no longer be a
  self-closing tag; emit the title only for shapes that have a description so untouched output stays
  byte-identical
- The rect-then-text emission order at `internal/diagram/svg.go:130-131` is what
  `svgFillOfLabel` depends on — verify that helper still passes rather than rewriting it
- Existing SVG assertions: `internal/diagram/svg_test.go` plus `requireValidXML`
  (`internal/diagram/contract_test.go:35-38`)
- Caller pattern **UI** — assert the description reaches the right shape, not DOM nesting depth

**Testable:** Yes — `diagram.ExportSVG` is exported, and `cli.RunDiagram` drives it.

**Verification:** `mise exec -- task test:unit` passes, including the shared exporter contract tests.

**Depends on:** Task 1

---

### Task 7: Accept `description` and the new block forms in the tree-sitter grammar

**Behavior:** The tree-sitter grammar parses `description "<text>"` inside every block construct that
accepts it, parses the new `actor` and `model` block forms, and still parses a field named
`description`.

**Acceptance Criteria:**
- [ ] `grammar.js` accepts `description "<text>"` inside `context`, `aggregate`, `slice`, `command`,
      `event`, `view`, `automation`, `translation` and `trigger` blocks, in any position within the
      block, with the corpus expectations naming the description node
- [ ] `grammar.js` accepts `actor "<name>" { description "<text>" }` and
      `model "<name>" { description "<text>" }`, and the single-line forms of both still parse
- [ ] A corpus case parses `fields { description string required }` as a `field_line` of
      `any_identifier` nodes, mirroring `test/corpus/version_header.txt:31-58`
- [ ] Every existing corpus expectation still matches unchanged
- [ ] `mise exec -- task test:grammar` passes and no corpus expectation contains an `ERROR` or
      `MISSING` node
- [ ] The working tree changes only `editors/tree-sitter-emod/grammar.js` and files under
      `editors/tree-sitter-emod/test/corpus/`; `src/` stays untracked and
      `editors/tree-sitter-emod/.gitignore` is unmodified
- [ ] `editors/tree-sitter-emod/queries/highlights.scm` and
      `editors/vscode/syntaxes/emod.tmLanguage.json` are unmodified — highlight scopes are US-017

**Affected Files/Modules:**
- `editors/tree-sitter-emod/grammar.js` — a description rule plus its use in each block, and the
  optional block on `model_definition` and `actor_definition`
- `editors/tree-sitter-emod/test/corpus/` — cases for the attribute per construct, both block forms,
  and the field named `description`

**Patterns to Follow:**
- Keyword tokens must be narrow enough to match only in their own position, and `any_identifier`
  (`editors/tree-sitter-emod/grammar.js:215`) is what keeps keywords usable as field names —
  `version_header` (`:21-24`) is the worked example (`tasks/learnings.md`, "New DSL keywords must
  stay usable as field names")
- Existing block rules to extend: `context_definition` (`:45-51`), `aggregate_definition` (`:54-60`),
  `slice_definition` (`:63-69`), `command_definition` (`:84-90`), `event_definition` (`:93-102`),
  `trigger_definition` (`:122-132`), `view_definition` (`:154-163`), `automation_definition`
  (`:177-187`), `translation_definition` (`:190-201`)
- Corpus file format and node-expectation style: `test/corpus/version_header.txt` and
  `test/corpus/full_model.txt`
- Run the target through `mise exec --`, not bare PATH: the repo pins tree-sitter-cli 0.26.9 while a
  global pin of 0.25.10 may win on PATH and produce different generated output
  (`tasks/learnings.md`, "Run repo tooling through `mise exec --`")
- Generated `src/` stays gitignored — do not un-ignore it to prove regeneration
  (`tasks/learnings.md`, "Generated tree-sitter `src/` stays gitignored")

**Testable:** Yes — the grammar's parse behaviour is exercised by the corpus suite.

**Verification:** `mise exec -- task test:grammar` passes, and `git status` shows no new or modified
files under `editors/tree-sitter-emod/src/`.

**Depends on:** Task 1, Task 2

---

### Task 8: Document `description` in the DSL reference

**Behavior:** A reader of `docs/dsl-reference.md` learns that any block construct can carry a
description, and that `actor` and `model` have an optional block form for the same purpose.

**Acceptance Criteria:**
- [ ] The reference documents the `description` attribute with at least one example, and names every
      construct that accepts it
- [ ] The `model` and `actor` entries in "3. Top-Level Constructs" document the optional block form
      and no longer state that `model` has no braces (`docs/dsl-reference.md:82`)
- [ ] The reference states that descriptions are carried through the `json` and `cue` exports and
      rendered as draw.io tooltips and SVG `<title>` elements
- [ ] Every `.emod` snippet added to the reference parses and validates cleanly with `emod validate`
- [ ] If a numbered section is added, moved or renumbered, every in-document link citing a section
      number still points at the section it names — reconcile the `## <n>. Title` headings against
      the `(#<n>-…)` links
- [ ] `examples/*.emod`, `internal/parser/testdata/*.emod` and `internal/test/fixtures.go` are
      unmodified — rewriting examples is US-018

**Affected Files/Modules:**
- `docs/dsl-reference.md` — the description attribute, the two block forms, and the consumer note

**Patterns to Follow:**
- Section 2 "Version Header" (`docs/dsl-reference.md:45-70`) is the closest precedent: a short
  grammar sketch, a worked `emod` snippet, then a bulleted list of the behavioural consequences
- Heading anchors embed the section number, and nothing in CI checks the links — after editing the
  section list, list `^## [0-9]+\.` and `\(#[0-9]+-` and reconcile the two
  (`tasks/learnings.md`, "`docs/dsl-reference.md` anchors embed the section number")
- `~/.claude/rules/markdown-docs.md`: the result must read as a first version — no "new in this
  version", no narration of what the document used to say

**Testable:** No — prose documentation with no runtime behaviour of its own. The snippets are
verified by running `emod validate` against them, not by an automated test.

**Verification:** Every added snippet passes `emod validate`; the numbered-heading and link lists
reconcile; `mise exec -- task test` remains green.

**Depends on:** Task 1, Task 2

---

## Summary

**Total tasks:** 8.

**Ordering rationale:** dependency-first, then consumer breadth. Tasks 1 and 2 establish the language
surface — nothing else can be written until the AST carries a description, and Task 2's block form is
a distinct grammar change that reuses Task 1's attribute. Task 3 comes next because it is the only
task that prevents an active regression (`emod fmt` deleting descriptions) rather than adding a
feature. Tasks 4, 5 and 6 are the three independent consumer surfaces named in the story's criteria
and can run in any order or in parallel; JSON and CUE are one task because an existing parity subtest
requires both formats to describe the same document. Task 7 mirrors the same syntax into the editor
grammar, which touches no Go code but must agree with the syntax Tasks 1 and 2 settle. Task 8
documents what the previous seven built.

**Story criteria coverage:**
- "`context`, `aggregate`, `slice`, `command`, `event`, `view`, `automation`, `translation`, and
  `trigger` each accept an optional `description` string attribute" → Task 1 (Task 7 mirrors it in
  the tree-sitter grammar)
- "`actor` and `model` gain an optional block form that holds a description; the existing single-line
  form remains valid" → Task 2 (Task 7 mirrors it)
- "`emod export -f json` and `-f cue` carry descriptions through" → Task 4
- "draw.io diagrams attach descriptions as tooltips; SVG diagrams as `<title>` elements" → Tasks 5
  and 6
- "A file with no descriptions parses, validates, exports, and renders exactly as before" → carried
  as an explicit unchanged-output criterion in Tasks 1, 3, 4, 5 and 6, anchored on
  `internal/test.HotelReservation` staying description-free

**Beyond the story's criteria:** Task 3 (formatter) and Task 8 (reference documentation). Task 3
implements only the `description` placement rule that US-014 later generalises; Task 8 follows the
precedent set when US-001 documented the version header in the same story that introduced it. Neither
overlaps work already done, and both are small.

**Deferred, with the story that owns them:** `emod glossary` (US-004); field-level descriptions
(feature non-goal); LSP hover, completion and go-to-definition over descriptions (US-015); syntax
highlighting for the new keyword (US-017); rewriting `examples/*.emod` (US-018); the rest of US-014's
canonical formatting order. The `diagram-json` export, the mermaid and ascii renderers, and the web
viewer are untouched — no acceptance criterion reaches them.

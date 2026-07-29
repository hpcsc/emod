# US-003: Use reserved words as field names

## Progress
- [x] Task 1: End a field declaration at the end of its line
- [x] Task 2: Accept every keyword the lexer knows in field-name position
- [x] Task 3: Keep keyword-named fields intact through `emod validate` and `emod fmt`
- [x] Task 4: Carry keyword-named fields through the JSON, CUE and diagram-JSON exports and the renderers
- [x] Task 5: Cover every keyword as a field name in the tree-sitter corpus
- [x] Task 6: Document that any keyword may be used as a field name

---

## Story Reference

`user-stories/specs-and-metadata.md:44-53` → **US-003: Use reserved words as field names** (third
story of "Specs, Invariants, and Model Metadata"). The feature's overarching constraint —
"every existing `.emod` file stays valid with unchanged meaning" — is the one this story exists to
protect.

**In scope:** field-name position inside `fields` blocks; the mechanism that makes the guarantee
general rather than per-keyword; and the downstream behaviour of such fields in validation,
formatting, exports and diagrams.

**What exploration changed about the scope.** The Go parser and the tree-sitter grammar already
accept all thirty keywords in field-name position today — verified by running
`go run ./cmd/emod validate` and `tree-sitter parse` over one file per keyword. So the deliverable is
not "make it work"; it is:

1. **Make the guarantee general instead of per-keyword.** `checkIdentifierLike`
   (`internal/parser/parser.go:1356-1364`) decides field-name eligibility by *ordinal comparison*
   (`typ < lexer.Identifier`), and the two tests that pin the behaviour
   (`internal/lexer/tokenizer_test.go:13-46`, `internal/parser/parser_test.go:224-249`) each carry a
   hand-written list — the lexer's already omits nine of the thirty recognised keywords. A keyword
   whose `Kind` is declared in the wrong place silently loses field-name eligibility with no test
   failing. That is the "new grammar keywords never force me to rename my domain's fields" promise,
   and it is currently held together by a convention nothing checks.
2. **Fix a defect that makes the promise false today.** `parseField`
   (`internal/parser/parser.go:1118-1143`) is not line-aware, so a field written without a modifier
   swallows the next line's field name. Verified: `fields { alpha string` / `beta string required }`
   parses as `alpha string beta` plus `string required`, and `emod fmt` rewrites the file to that
   mangled shape and calls it canonical. `fields { type string` followed by any other field — the
   exact declaration the story's Context calls plausible — is silently corrupted today.
3. **Pin the downstream behaviour** the third acceptance criterion names.

**The forward-looking keywords in criterion 1** (`type`, `spec`, `given`, `when`, `then`, `rejected`,
`invariant`, `after`) are not keywords yet — they arrive with US-005, US-006 and US-013, and parse as
ordinary identifiers today. This story does not add them. It makes the *enumeration* the source of
truth, so each is covered on the day it becomes a keyword without anyone remembering to extend a
list. `description` and `emod` already exist and are covered concretely.

**Out of scope (resolved for this run):**
- **LSP hover.** `isKeyword` (`internal/lsp/hover.go:37`) is an ordinal range ending at
  `KeywordExternal`, and `keywordDescriptions` (`:13-33`) has nineteen entries, so hovering a field
  named `source` shows "Defines the source for an external system." No acceptance criterion mentions
  hover, and US-015 owns LSP work for the feature. Do not widen the range here.
- **Completion** (`internal/lsp/completer.go`) — same owner, same reasoning.
- **Syntax highlighting.** `editors/vscode/syntaxes/emod.tmLanguage.json:63` is a
  case-insensitive word alternation with no positional context, so a field named `event` is already
  painted as a keyword in VS Code. Making a regex grammar position-aware is a separate exercise and
  US-017 owns highlight scopes. The tree-sitter side is already correct: `highlights.scm:19-40`
  anchors on anonymous keyword tokens, and a keyword in field position is an `any_identifier` node.
- **Field-line grouping in the tree-sitter grammar.** `field_line` (`grammar.js:116-123`) is
  deliberately greedy (`prec.right`), so after Task 1 it groups a modifier-less field with the next
  line's name where the Go parser will not. A *looser* grammar is the sanctioned direction
  (`tasks/learnings.md`, "The tree-sitter grammar must never be stricter than the Go parser"); making
  it newline-sensitive needs an external scanner, since `extras` swallows `\s`. Leave it.
- **Keywords as construct names.** `command source { }` is rejected today (`parser.go:464` and its
  siblings gate on `lexer.Identifier`), and the story asks only about field-name position. Task 2
  pins that this stays rejected. The diagnostic cascade that rejection produces is pre-existing and
  is not this story's to fix.
- **Rewriting `examples/*.emod`** to demonstrate the feature (US-018 owns example refreshes).
- **Generated tree-sitter `src/`**, which stays untracked.

**Learnings folded in** from `tasks/learnings.md`: the keyword-as-field-name pattern and the two
corpus/parser cases it requires; the ordinal-range trap in `internal/lsp/hover.go:37`; the single
`keywords` map as the one source of truth; `mise exec --` for `task test:grammar`; generated
tree-sitter `src/` stays untracked; acceptance criteria never reference commit, branch or remote
state; formatter output always opens with `emod N`; never emit emod source through `%q`;
`docs/dsl-reference.md` anchors embed the section number; additive output changes owe a
byte-identical receipt; the unfeatured/featured fixture pair in `internal/test/fixtures.go`; CLI
diagnostic tests must assert distinguishing message text; one diagnostic per malformed block entry,
pinned with `require.Len(t, diags, 1)`.

---

## Codebase Context

**Pipeline.** `internal/oracle/oracle.go:14-24` is the canonical lex → parse → validate → lint chain;
`internal/cli/validate.go`, `internal/cli/export.go`, `internal/cli/diagram.go`,
`internal/lsp/server.go` and `internal/wasm/pipeline.go` each rebuild it. Anything that must hold
everywhere belongs in the lexer or parser.

**Lexer.** `internal/lexer/token.go:5-56` declares `Kind` as one iota block — thirty keywords first
(`KeywordModel` … `KeywordDescription`), then `Identifier`, `String`, `Integer`, then punctuation and
`Comment`/`EOF`. The string ↔ `Kind` mapping is the single `keywords` map at `:60-91`, inverted once
into `keywordNames` at `:93-101`; `Kind.String()` (`:110-148`) consults that inversion before its
switch. `keywordOrIdentifier` (`internal/lexer/tokenizer.go:202-207`) is the only lookup site.
Nothing exports the keyword set, so every consumer that wants "all keywords" writes its own literal.

**The ordinal coupling.** `checkIdentifierLike` (`internal/parser/parser.go:1356-1364`) returns true
for `Identifier` or any `Kind` ordered *before* it. It gates field names, field types and field
modifiers (`parseField`, `:1118-1143`), the `fields` block loop (`parseFields`, `:1088-1116`), tag
keys and tag field references (`parseTagEntry`, `:1174-1200`), `mode` values (`:236`) and `tag()`
predicate operands (`:690`, `:710`). `internal/lsp/hover.go:37` uses the same ordinal trick over a
narrower range and is already out of step with the keyword list.

**Field parsing.** `parseField` reads name, then type, then an optional modifier, each guarded only by
`checkIdentifierLike` — there is no line boundary anywhere in the sequence, and the enclosing loop
stops only at `}` or EOF. `parseDescriptionInto` (`:1288-1305`) is the one construct in the parser
that *does* bound itself to a line, using `checkSameLineAs` (`:1352-1354`) to drain the rest of the
line on error; the version header uses the same helper at `:92` and `:123`. The observable
consequence of the gap: `emod fmt` on

```
fields {
  alpha string
  beta string required
}
```

emits `alpha  string   beta` / `string required` and is then idempotent on the corruption. No
`.emod` file in the repo currently has a modifier-less field (checked across `examples/`,
`internal/parser/testdata/` and `internal/test/fixtures.go`), so the blast radius is test fixtures
only — but `internal/formatter/formatter_test.go:613-660` asserts the formatter *emits* that shape,
so the formatter can produce output it cannot read back.

**Formatter.** `writeFields` (`internal/formatter/formatter.go:260-272`) pads name and type columns
via `fieldColumnWidths` (`:288-298`) and omits the trailing space when there is no modifier.
`formatter.Format` always emits `emod N` first. Round-trip and idempotence leaves live in
`internal/formatter/formatter_test.go:427-511` (`parseModel` → `Format` → re-parse compared with
`test.RequireEqual` under `ignoreFormatterNormalizations`), with CLI-level equivalents in
`internal/cli/fmt_test.go:122`.

**Exports.** `convertFields` (`internal/export/export.go:448`) fills `jsonField` (`:105`) for the JSON
export; `cueWriter.writeField` (`:1273-1278`) does the same for CUE; `convertFieldsToDiagram`
(`:1123`) fills the deliberately separate `jsonDiagramField` for `ExportDiagramJSON`. Field names are
*values* in all three (`{name: "type", type: "string"}`), never keys, so no CUE identifier collision
is possible — verified by exporting a keyword-field model in both formats. Two coupled subtests guard
the surfaces: "output conforms to the schema's Model definition"
(`internal/export/export_test.go:3296`, `cue vet -d '#Model'` against `internal/cue/schema.cue`) and
"CUE and JSON exports describe the same model" (`:3317`).

**Diagrams.** None of `internal/diagram/drawio.go`, `svg.go`, `mermaid.go` or `ascii.go` renders field
names — verified by grepping every format's output for a keyword field name and finding nothing. The
third acceptance criterion is therefore satisfied for diagrams by a *differential* receipt, the shape
`withoutDescriptions` (`internal/diagram/contract_test.go:589-...`) already establishes: render the
keyword-field model and the ordinary-field model and compare bytes. `ExportDiagramJSON` is the one
diagram-adjacent surface that *does* carry field names, through its own `jsonDiagramField` fork —
which must stay forked (`tasks/learnings.md`).

**Validator and linter.** `internal/validator/validator.go:157-176` collects field names only to check
tag references; `internal/linter/linter.go:519` reads `Fields[0].Name` for the `clickbait-event` rule.
Neither has a naming convention for field names, so a keyword-named field triggers nothing new.

**Tree-sitter grammar.** `editors/tree-sitter-emod/grammar.js` is a deliberately looser mirror.
`word: $ => $.identifier` where `identifier` is `/[A-Z][a-zA-Z0-9_]*/`, so lowercase keyword tokens
are not word-extracted and the lexer only offers them in states that accept them; `any_identifier`
(`:211`) is the permissive `/[a-zA-Z_][a-zA-Z0-9_]*/` used in field, subscribes and attribute-value
positions. `version_header` (`:27-30`) shows how a keyword token is narrowed so it cannot match
outside its own position. Corpus cases live in `test/corpus/`; the two existing single-keyword cases
are `version_header.txt:31-58` and `description.txt:302-329`. `task test:grammar` runs
`tree-sitter generate` before `tree-sitter test`, and `editors/tree-sitter-emod/.gitignore` keeps
`src/` untracked.

**Shared fixtures.** `internal/test/fixtures.go` holds `HotelReservation` (`:7-89`, uses no optional
feature) and `DescribedHotelReservation` (`:94-197`, exercises descriptions everywhere), consumed by
`internal/parser`, `internal/formatter`, `internal/export`, `internal/oracle` and `internal/cli`.
`Unparseable` (`:199`) is the error fixture.

**Docs.** `docs/dsl-reference.md:274-301` is `## 8. Fields`. It currently describes a field name as a
"PascalCase identifier" (`:284`), which contradicts every example in the same document. Headings are
`## <n>. Title` and four in-document links embed the number, so sections must not be renumbered.

**Test conventions** (`CLAUDE.md` "Go Test Organization"): one umbrella `Test{TypeName}` per type,
`t.Run` groups named after the operation, leaf subtests reading as sentences about the observed
outcome, `testify/require`, fresh fixtures per leaf, `//go:build unit` / `//go:build integration`
tags. AST comparisons use `test.RequireEqual` with the ignore options declared at
`internal/parser/parser_test.go:20` and `internal/formatter/formatter_test.go`.

---

## Tasks

### Task 1: End a field declaration at the end of its line

**Behavior:** A field declaration inside a `fields` block is confined to the line it starts on. A
field written without a modifier no longer absorbs the next line's field name, so what `emod fmt`
writes is what the parser reads back. A field name left with nothing after it on its own line is
reported once, and the enclosing block and everything after it still parse.

**Acceptance Criteria:**
- [ ] A `fields` block whose first entry is `<name> <type>` with no modifier and whose second entry is
      a full `<name> <type> <modifier>` line parses to exactly two fields carrying the names, types
      and modifiers as written, with no diagnostics
- [ ] Parsing a model that contains a modifier-less field, formatting it, and re-parsing the output
      yields a model equal to the original under the comparison already used at
      `internal/formatter/formatter_test.go:427-464`; a second `formatter.Format` of the re-parsed
      model is byte-identical to the first
- [ ] Running the `fmt` CLI entry point over a file containing a modifier-less field, then running it
      again, leaves the file byte-identical after the second run, and `--check` over the
      once-formatted file reports nothing to change
- [ ] A field name with nothing following it on the same line produces exactly one diagnostic,
      positioned on the field name's own line, naming what was expected; the `fields` block still
      closes and the constructs after it still parse
- [ ] `fields { <name> <type> <modifier> }` written entirely on one line still parses as one field
- [ ] `internal/test.HotelReservation`, `internal/test.DescribedHotelReservation`, every file under
      `internal/parser/testdata/` and every file under `examples/` parse to the same AST and the same
      diagnostics as they do without this change

**Affected Files/Modules:**
- `internal/parser/parser.go` — `parseField` (`:1118-1143`) and its caller `parseFields`
  (`:1088-1116`)
- `internal/parser/parser_test.go` — the `fields in command` group (`:783-812`) and the diagnostics
  groups
- `internal/formatter/formatter_test.go` — the round-trip group (`:427-511`) and
  "fields without modifier omit trailing whitespace" (`:613-660`)
- `internal/cli/fmt_test.go` — the idempotence and `--check` leaves (`:122`)

**Patterns to Follow:**
- Line-bounded parsing and its error recovery already exist in this file: `parseDescriptionInto`
  (`internal/parser/parser.go:1288-1305`) with the `checkSameLineAs` helper (`:1352-1354`); the
  version header uses the same helper at `:92` and `:123`
- Recover by draining the rest of the line and stopping at `lexer.CloseBrace`, and pin the result
  with `require.Len(t, diags, 1)` — `tasks/learnings.md`, "A new block entry keyword owes three
  things to the parser's diagnostics"
- Formatter goldens and `RunFmt` canonical constants always open with `emod N` —
  `tasks/learnings.md`, "Formatter output always begins with `emod N`"
- CLI leaves must assert the tokens that identify *this* diagnostic, not just a line number or a
  count — `tasks/learnings.md`, "CLI diagnostic tests must assert the distinguishing message text";
  see `internal/cli/validate_test.go:253-258`
- Caller pattern **Inbound** (`~/.config/ai/guidelines/testing/caller-patterns.md`): the emod source
  is the input, the `(*ast.Model, diagnostics)` pair and the formatted bytes are what the caller
  observes — assert acceptance, rejection and resulting shape, never parser internals. Unit of
  behavior per `~/.config/ai/guidelines/go/testing-patterns.md` — "a field declaration ends at its
  line" is one behavior; the parser's token bookkeeping is not
- `CLAUDE.md` "Go Test Organization": umbrella `TestParser` / `TestFormatter`, operation-named
  groups, sentence-shaped leaf names, `testify/require`, fresh fixtures per leaf

**Testable:** Yes — `parser.Instance.Parse`, `formatter.Format` and the `fmt` CLI entry point are all
exported and already have umbrella tests.

**Verification:** `mise exec -- task test:unit` and `mise exec -- task test:integration` pass; running
the built binary's `fmt` twice over a file with a modifier-less field leaves identical bytes.

**Depends on:** None

---

### Task 2: Accept every keyword the lexer knows in field-name position

**Behavior:** `internal/lexer` exposes the keyword spellings it recognises, derived from the one
`keywords` map, and the parser decides field-name eligibility from that rather than from where a
`Kind` happens to sit in the iota block. Every keyword the lexer recognises is usable as a field
name, a field type and a field modifier, and both the lexer and parser tests enumerate the exported
set instead of hand-written subsets — so a keyword added by a later story is covered without anyone
editing a list.

**Acceptance Criteria:**
- [ ] `internal/lexer` exposes the set of keyword spellings it recognises, derived from the same
      `keywords` map the tokenizer already uses (`internal/lexer/token.go:60-91`) rather than from a
      second literal, and it reports all thirty
- [ ] Every spelling in that set scans to a token whose kind is not `lexer.Identifier`, whose `Value`
      is the spelling, and whose `Kind.String()` returns the spelling
- [ ] The lexer's "keyword tokens" group (`internal/lexer/tokenizer_test.go:13-46`) is driven by the
      exported set rather than its current twenty-one-entry literal, which omits nine recognised
      keywords (`mode`, `tags`, `decides_on`, `where`, `and`, `or`, `not`, `tag`, `events`)
- [ ] For every spelling in that set, a `fields` block containing `<spelling> string required` parses
      to a single field with that name, that type and that modifier, and produces no diagnostics
- [ ] For every spelling in that set, a field written with the spelling in all three positions parses
      to one field whose name, type and modifier are all that spelling, with no diagnostics
- [ ] The field-name coverage in `internal/parser/parser_test.go:224-249` no longer names keywords in
      a literal slice, and its subtest names still identify which spelling failed
- [ ] `internal/parser` contains no comparison of a `lexer.Kind` against `lexer.Identifier` by
      ordinal ordering — checkable by reading `internal/parser/parser.go` for `< lexer.Identifier`
- [ ] `oracle.Check` over a complete model whose command and event `fields` blocks name a field after
      every recognised keyword returns no error-severity diagnostics
- [ ] A keyword in *construct-name* position is still rejected: `command source { }` produces a
      diagnostic naming what was expected after `command`

**Affected Files/Modules:**
- `internal/lexer/token.go` — the exported keyword accessor and, if one is introduced, the keyword
  predicate on `Kind`; both derived from `keywords` / `keywordNames` (`:60-101`)
- `internal/lexer/tokenizer_test.go` — the "keyword tokens" group driven by the exported set
- `internal/parser/parser.go` — `checkIdentifierLike` (`:1356-1364`)
- `internal/parser/parser_test.go` — the keyword-as-field-name coverage (`:224-249`); consider moving
  it out of the `version header` group, whose subject it no longer is

**Patterns to Follow:**
- The single `keywords` map inverted once into `keywordNames` (`internal/lexer/token.go:60-101`) is
  the established one-source-of-truth shape; add the accessor beside the existing inversion rather
  than a parallel literal — `tasks/learnings.md`, "Keyword surfaces fan out past the lexer, parser
  and tree-sitter grammar"
- The guarantee this task generalises is recorded in `tasks/learnings.md`, "New DSL keywords must
  stay usable as field names" — keep both halves of the pattern it names (Go parser and tree-sitter
  corpus, the latter is Task 5)
- Do **not** extend `internal/lsp/hover.go:37` to use the new predicate; its narrower ordinal range is
  deliberate and US-015 owns it (`tasks/learnings.md`, same entry)
- Caller pattern **Exported API** for the lexer accessor (other packages are the caller — assert the
  contract, not the map) and **Inbound** for the parser behaviour
  (`~/.config/ai/guidelines/testing/caller-patterns.md`). A table-driven leaf per spelling with the
  spelling as the subtest name keeps failures legible — per
  `~/.config/ai/guidelines/go/testing-patterns.md`, "every recognised keyword is usable as a field
  name" is the unit of behavior, not the accessor's return type
- `CLAUDE.md` "Go Test Organization" for both umbrellas

**Testable:** Yes — the accessor, `lexer.Scan`, `parser.Instance.Parse` and `oracle.Check` are
exported.

**Verification:** `mise exec -- task test:unit` passes; `rg -n "< lexer.Identifier" internal/parser`
returns nothing.

**Depends on:** Task 1

---

### Task 3: Keep keyword-named fields intact through `emod validate` and `emod fmt`

**Behavior:** A model whose fields are named after DSL keywords validates with no diagnostics and
survives `emod fmt` unchanged — names, types, modifiers and column alignment — with re-formatting a
no-op. A shared fixture makes that model available to the export and diagram task as well.

**Acceptance Criteria:**
- [ ] A third fixture sits alongside `HotelReservation` and `DescribedHotelReservation` in
      `internal/test/fixtures.go` whose command, event and view `fields` blocks name fields after DSL
      keywords — covering at minimum `description`, `emod`, `events`, `source`, `model`, `fields`,
      `and`, `not`, `where`, `tag` — including at least one modifier-less field followed by a further
      field, and it parses with no diagnostics
- [ ] The fixture is a complete, realistic model in the shape of `HotelReservation`: it validates,
      formats, exports and renders, so `oracle.Check` over it returns no error-severity diagnostics
- [ ] Parsing the fixture, formatting it and re-parsing yields an equal model, and a second
      `formatter.Format` is byte-identical to the first
- [ ] The formatted fixture's `fields` blocks are asserted against exact expected lines, showing each
      keyword-named field on its own line with the name, type and modifier columns padded
- [ ] The `fmt` CLI entry point run with `--check` over the formatted fixture reports nothing to
      change, and run twice over the unformatted fixture leaves identical bytes after the second run
- [ ] The `validate` CLI entry point over a file holding the fixture exits reporting no diagnostics
- [ ] `HotelReservation` and `DescribedHotelReservation` are unchanged, and the guard subtest at
      `internal/parser/parser_test.go:3157` still passes

**Affected Files/Modules:**
- `internal/test/fixtures.go` — the new keyword-field fixture, alongside the existing two
- `internal/oracle/oracle_test.go` — clean-validation coverage for the new fixture (`:17-30` shows the
  existing pair)
- `internal/formatter/formatter_test.go` — round-trip, idempotence and exact-output leaves in the
  groups at `:427-511` and `:613-660`
- `internal/cli/validate_test.go` and `internal/cli/fmt_test.go` — CLI leaves (`:20-21` and `:122`
  show how the existing fixtures are wired in)

**Patterns to Follow:**
- The fixture file's established role split — an unfeatured witness plus a featured model every
  downstream package asserts against — is described in `tasks/learnings.md`, "Shared fixtures come in
  an unfeatured/featured pair, guarded by a walk that must be extended". This fixture is a third,
  independent witness; it does not replace either
- Every formatter expectation and every canonical `RunFmt` constant opens with `emod 1` —
  `tasks/learnings.md`, "Formatter output always begins with `emod N`"
- Field names are written verbatim into emod source; never route emod source text through `%q` —
  `tasks/learnings.md`, "Never write emod source with `%q`", and `quoted()` at
  `internal/formatter/formatter.go:47` is only for quoted values
- Column padding behaviour lives in `writeFields`/`fieldColumnWidths`
  (`internal/formatter/formatter.go:260-298`) — assert the emitted lines, not the widths
- CLI leaves assert distinguishing message content, not just counts — `tasks/learnings.md`, "CLI
  diagnostic tests must assert the distinguishing message text"
- Caller pattern **Inbound** for validate and **UI** for what `fmt` writes back
  (`~/.config/ai/guidelines/testing/caller-patterns.md`): the model author is the caller, the file on
  disk is what they observe
- `CLAUDE.md` "Go Test Organization"

**Testable:** Yes — `oracle.Check`, `formatter.Format` and the `validate` / `fmt` CLI entry points are
exported.

**Verification:** `mise exec -- task test:unit` and `mise exec -- task test:integration` pass.

**Depends on:** Task 2

---

### Task 4: Carry keyword-named fields through the JSON, CUE and diagram-JSON exports and the renderers

**Behavior:** Field names that happen to be DSL keywords appear verbatim in `emod export -f json`,
`-f cue` and `-f diagram-json`; the two exports still describe the same model and still conform to
the embedded schema; and the draw.io, SVG, mermaid and ASCII renderings are unaffected by the naming.

**Acceptance Criteria:**
- [ ] `ExportJSON` of the keyword-field fixture carries every field's name, type and modifier
      verbatim, including the modifier-less field
- [ ] `ExportCUE` of the same fixture carries the same names, and the "CUE and JSON exports describe
      the same model" comparison (`internal/export/export_test.go:3317`) holds when applied to it
- [ ] The CUE export of the fixture passes the `cue vet -d '#Model'` check against
      `internal/cue/schema.cue` used at `internal/export/export_test.go:3296`
- [ ] `ExportDiagramJSON` of the fixture carries the keyword field names in its node `fields`, and no
      keyword field name leaks into any other part of the diagram document
- [ ] The draw.io, SVG, mermaid and ASCII renderings of the fixture are byte-identical to the
      renderings of the same model with its keyword field names replaced by ordinary ones — the
      differential receipt, since no renderer draws field names
- [ ] The `export` CLI entry point with `--format json` and with `--format cue` over a file holding
      the fixture prints the keyword field names and reports no diagnostics
- [ ] `internal/cue/schema.cue` is unchanged — a keyword field name is a value, not a new key

**Affected Files/Modules:**
- `internal/export/export_test.go` — JSON, CUE, cross-format and schema-conformance leaves
  (`:3296`, `:3317`)
- `internal/diagram/contract_test.go` — the differential rendering receipt, plus whichever of
  `drawio_test.go`, `svg_test.go`, `mermaid_test.go`, `ascii_test.go` the contract test drives
- `internal/cli/export_test.go` (`:37`) and `internal/cli/diagram_test.go` (`:23`) — CLI leaves

**Patterns to Follow:**
- The JSON/CUE/`schema.cue` coupling and the two subtests that enforce it are described in
  `tasks/learnings.md`, "A new exported field must land in JSON, CUE and `schema.cue` in the same
  change" — the same entry records that `jsonDiagramEvent`/`jsonDiagramField` are a deliberate fork
  of the JSON structs and must not be re-merged
- The "prove the output did not move" receipt has an existing shape: `withoutDescriptions`
  (`internal/diagram/contract_test.go:589-...`) strips a feature from a model and compares the render
  against the featured one — `tasks/learnings.md`, "Additive output changes owe a byte-identical
  receipt for models that do not use the feature"
- Field conversion sites, for locating what to assert on: `convertFields`
  (`internal/export/export.go:448`), `cueWriter.writeField` (`:1273-1278`),
  `convertFieldsToDiagram` (`:1123`)
- Caller pattern **Exported API** for the export functions and **UI** for what the CLI prints
  (`~/.config/ai/guidelines/testing/caller-patterns.md`): assert the document a downstream consumer
  decodes, not the writer's internals
- `CLAUDE.md` "Go Test Organization"

**Testable:** Yes — `ExportJSON`, `ExportCUE`, `ExportDiagramJSON`, the diagram renderers and the
`export` / `diagram` CLI entry points are exported.

**Verification:** `mise exec -- task test:unit` and `mise exec -- task test:integration` pass; the
`cue`-backed leaf runs rather than skipping when the `cue` binary is present.

**Depends on:** Task 3

---

### Task 5: Cover every keyword as a field name in the tree-sitter grammar corpus

**Behavior:** The tree-sitter grammar parses a `fields` block that names a field after every keyword
the emod lexer recognises, producing ordinary `field_line` / `any_identifier` nodes and no error
node — so an editor never red-squiggles a file `emod validate` accepts.

**Acceptance Criteria:**
- [ ] A corpus case parses a `fields` block holding one field per keyword spelling in
      `internal/lexer/token.go:60-91` (all thirty), with an expected tree of `field_line` nodes made
      of `any_identifier` children and no `ERROR` or `MISSING` node anywhere
- [ ] The keyword spellings in that case match the thirty in the lexer's `keywords` map exactly —
      checkable by listing the field names in the corpus case against
      `rg -n '^\t"' internal/lexer/token.go`
- [ ] The existing single-keyword cases still pass: "Field named emod inside a fields block"
      (`editors/tree-sitter-emod/test/corpus/version_header.txt:31-58`) and "Field named description
      inside a fields block" (`editors/tree-sitter-emod/test/corpus/description.txt:302-329`)
- [ ] `mise exec -- task test:grammar` passes
- [ ] Running `mise exec -- task test:grammar` leaves every tracked file under
      `editors/tree-sitter-emod/` byte-identical — generated `src/` stays untracked and is not added

**Affected Files/Modules:**
- `editors/tree-sitter-emod/test/corpus/fields.txt` — new corpus file for field-block cases
- `editors/tree-sitter-emod/grammar.js` — only if a spelling fails; none does today, so a diff here
  needs a failing case to justify it

**Patterns to Follow:**
- Corpus case format, including the expected-tree block: `version_header.txt:31-58`
- `grammar.js:211` (`any_identifier`) is what makes field positions permissive, and `:27-30`
  (`version_header`) is the reference for a keyword token narrowed to its own position —
  `tasks/learnings.md`, "New DSL keywords must stay usable as field names"
- Run the target through `mise exec --`; the repo pin and the user's global `tree-sitter` pin produce
  different generated output — `tasks/learnings.md`, "Run repo tooling through `mise exec --`"
- Do not un-ignore `src/` — `tasks/learnings.md`, "Generated tree-sitter `src/` stays gitignored"
- The grammar is allowed to be looser than the Go parser and must never be stricter —
  `tasks/learnings.md`, "The tree-sitter grammar must never be stricter than the Go parser".
  Specifically: leave `field_line`'s greedy `prec.right` (`grammar.js:116-123`) alone even though
  Task 1 makes the Go parser line-aware; matching it needs newline sensitivity the grammar cannot
  express while `extras` swallows `\s`
- Caller pattern **Not Every Test Has a Caller / config guard**
  (`~/.config/ai/guidelines/testing/caller-patterns.md`): the corpus pins editor-side parity with the
  Go parser, so the assertion is the tree shape, nothing about editors

**Testable:** Yes — through `tree-sitter test`, driven by `task test:grammar`.

**Verification:** `mise exec -- task test:grammar` passes and `git status` shows no new tracked files
under `editors/tree-sitter-emod/src/`.

**Depends on:** None

---

### Task 6: Document that any keyword may be used as a field name

**Behavior:** The DSL reference states the guarantee, so a model author reading it knows that
`fields { type string required }` is valid and will stay valid as the language grows.

**Acceptance Criteria:**
- [ ] `## 8. Fields` (`docs/dsl-reference.md:274-301`) states that a field name may be any
      identifier, including any word the DSL uses as a keyword, and that new keywords will not
      invalidate existing field names
- [ ] The section shows a `fields` block naming at least one field after a keyword, in the same
      shape the parser tests from Task 2 cover
- [ ] The "PascalCase identifier" description of a field name (`docs/dsl-reference.md:284`) no longer
      contradicts the lowerCamelCase field names used in every example in the same document
- [ ] No numbered section is added, removed or reordered: the `## <n>. Title` headings and the
      number-prefixed in-document links still agree, reconciled by listing `^## [0-9]+\.` against
      `\(#[0-9]+-` in the file

**Affected Files/Modules:**
- `docs/dsl-reference.md` — section 8 only

**Patterns to Follow:**
- Heading numbers are baked into this document's anchors; inserting or reordering a numbered section
  silently breaks four links — `tasks/learnings.md`, "`docs/dsl-reference.md` anchors embed the
  section number". Adding prose and an example *inside* section 8 is safe; a new `## 9.` is not
- The version-header documentation added by US-001 (`docs/dsl-reference.md:45-71`, commit `3c925ee`)
  and the descriptions section added by US-002 (`:364-428`, commit `a1b8cb3`) show the house style:
  a short statement of the rule, then a fenced example
- Prose style: `~/.config/ai/guidelines/writing/`

**Testable:** No — documentation prose with no automated assertion. The anchor-consistency criterion
is checked by running the two listings and reconciling them.

**Verification:** `rg -n '^## [0-9]+\.' docs/dsl-reference.md` and `rg -n '\(#[0-9]+-'
docs/dsl-reference.md` still agree; the new example matches a case covered by the Task 2 parser tests.

**Depends on:** Task 2

---

## Summary

**Six tasks.**

**Ordering rationale — defect first, then generality, then reach.** Task 1 comes first because it is
the only task with a user-visible bug behind it: `emod fmt` currently corrupts a `fields` block whose
first entry has no modifier, which makes the story's own example (`fields { type string` followed by
anything) unsafe today, and it violates the feature's overarching "every existing `.emod` file stays
valid with unchanged meaning" constraint. It also touches the same function Task 2 edits, so
sequencing them avoids a collision. Task 2 then converts the guarantee from a per-keyword convention
into an enumeration-driven one — the actual subject of the story, and the thing that makes the
forward-looking keywords in criterion 1 free. Tasks 3 and 4 fan the guarantee out to the four
surfaces criterion 3 names, split by output kind so each stays one commit: what the author reads back
(`validate`, `fmt`) and what downstream tools consume (exports, renderers). Task 3 builds the shared
fixture Task 4 consumes, hence the dependency. Task 5 is independent of the Go chain and can run in
parallel with Tasks 1-4. Task 6 documents the settled rule and depends only on Task 2.

**Acceptance criteria coverage:**

| Story criterion | Covered by |
|---|---|
| A `fields` block accepts each new keyword (`type`, `description`, `spec`, `given`, `when`, `then`, `rejected`, `invariant`, `after`, `emod`) as a field name | Tasks 2 and 5. `description` and `emod` are covered concretely as today's keywords; the other eight are not keywords yet, and the enumeration-driven tests in Task 2 cover each automatically on the day its story adds it to `internal/lexer/token.go` |
| Words that were already reserved (e.g. `events`, `source`) are also accepted in field-name position | Tasks 2 and 5 — all thirty recognised keywords, enumerated from the lexer's own map rather than a list |
| Such fields behave like any other field in validation, formatting, exports, and diagrams | Task 3 (validation, formatting) and Task 4 (JSON, CUE, diagram-JSON exports; draw.io, SVG, mermaid, ASCII renderings) |

**Deliberately pulled in:** Task 1. No criterion mentions the line boundary, and no story owns it —
but the field-name guarantee is hollow while `emod fmt` rewrites `type string` plus the next field
into `type string <nextname>`, and the two-test matrix in Task 2 would otherwise have to avoid
modifier-less fields to stay green, quietly leaving the plausible case uncovered.

**Deferred, with the owner named in the Story Reference:** LSP hover and completion for
keyword-named fields (US-015); TextMate highlight scopes, where a field named `event` is already
painted as a keyword (US-017); refreshing `examples/*.emod` (US-018); and the tree-sitter
`field_line` grouping divergence introduced by Task 1, which is the sanctioned direction for the
grammar to differ and would need an external scanner to close.

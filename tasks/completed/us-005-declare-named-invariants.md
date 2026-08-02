# US-005: Declare named invariants

## Progress
- [x] Task 1: Declare named invariants on an aggregate
- [x] Task 2: Declare the same invariants directly on a context
- [x] Task 3: Preserve invariants through `emod fmt`
- [x] Task 4: Reject two invariants sharing a name in one scope
- [x] Task 5: List invariants under their aggregate or context in `emod glossary`
- [x] Task 6: Carry invariants through the JSON and CUE exports and the embedded schema
- [x] Task 7: Accept invariant entries in the tree-sitter grammar
- [x] Task 8: Document invariants in the DSL reference

---

## Story Reference

`user-stories/specs-and-metadata.md` → **US-005: Declare named invariants** (fifth story of "Specs,
Invariants, and Model Metadata"). Design notes: `docs/proposals/specs-and-metadata-proposal.md`
§3 "Named Invariants" (lines 103-125), with the AST shape at lines 345-367 and the keyword list at
line 374.

**In scope:** an `invariant <identifier> "<prose statement>"` entry accepted inside an `aggregate`
block and directly inside a `context` block; a validation error when one scope declares the same
invariant name twice; no error for an invariant nothing references; the invariants listed under
their aggregate or context by `emod glossary` in both the markdown and the JSON rendering. Carried
with them, because the repo's writers would otherwise silently drop the new entry: `emod fmt`
round-trip, the JSON and CUE exports plus the embedded schema, and the tree-sitter grammar (which
must never reject what `emod validate` accepts), and the DSL reference.

**Out of scope:** `spec` blocks and `rejected <name>` references (US-006, US-007), which are the
first consumers of a declared invariant; flow rejection edges (US-009); the
`spec/invariant-never-exercised` lint rule (US-008) that is what eventually flags an unreferenced
invariant — this story's fourth criterion says the validator must stay silent about it; syntax
highlighting in `editors/vscode/syntaxes/emod.tmLanguage.json` and
`editors/tree-sitter-emod/queries/highlights.scm` (US-017); LSP hover, completion, go-to-definition
and find-references over invariants (US-015); `examples/*.emod` (US-018 requires every new construct
to appear there); diagram rendering of invariants, which the proposal gives only to the US-009
rejection edge; a lint rule for an invariant declared on an aggregate-mode context, which no story
asks for.

**Open question, decided:** the source proposal is silent on duplicate invariant names in one scope.
This story specifies a validation error and Task 4 implements it. "Same scope" means one `aggregate`
block, or one `context` block's own invariants — the same resolution scope US-006 and US-009 will
use for `rejected <name>`. An aggregate and its enclosing context are different scopes, as are two
sibling aggregates.

**Overarching constraint:** every existing `.emod` file stays valid with unchanged meaning. That is
load-bearing in three places here — the new keyword must stay usable as a field name, `emod fmt`
must still produce its current bytes for a model with no invariants, and no existing golden in
`internal/formatter`, `internal/export`, `internal/diagram` or `internal/glossary` may need editing.

**Learnings folded in** from `tasks/learnings.md`: ask the lexer which keywords exist (never restate
the set); a new keyword must stay usable as a field name on both the Go and tree-sitter side; a new
block entry keyword owes three things to the parser's diagnostics; gate every optional trailing token
on the first token's line; never write emod source with `%q`; additive output changes owe a
byte-identical receipt; a differential receipt must first prove the twin differs; a new exported
field lands in JSON, CUE and `schema.cue` together; the glossary collects once and renders twice;
generated tree-sitter `src/` stays gitignored and repo tooling runs through `mise exec --`;
`docs/dsl-reference.md` anchors embed the section number; acceptance criteria never reference commit
or branch state; assertions must have an input that makes them fail.

---

## Codebase Context

**Lexer.** `internal/lexer/token.go` holds one `keywords` map (thirty spellings today, `description`
and `emod` the most recent) and derives `Keywords()` and `Kind.IsKeyword()` from it. Four
keyword-coverage tests iterate `lexer.Keywords()` and never name a keyword —
`internal/lexer/tokenizer_test.go:14`, `internal/parser/parser_test.go:224` and `:242`,
`internal/oracle/oracle_test.go:44` — so adding `invariant` to the map immediately puts it under
"usable as a field name, type and modifier". Nothing needs restating; the risk is the opposite,
a parser path that does not go through `checkIdentifierLike`.

**AST.** `internal/ast/ast.go` has no invariant node. `Aggregate` (`:55-64`) and `Context`
(`:41-53`) both carry `Comments`, `Name`/`NamePos`, `Description`/`DescriptionPos`, `Slices` and
open/close positions; `Context` additionally carries `Mode`/`ModePos` and `Aggregates`. The proposal
adds `Invariants []*Invariant` to both and an `Invariant` node of comments, name and prose with a
position for each (`docs/proposals/specs-and-metadata-proposal.md:354-367`). Note the proposal calls
the prose field `Description` — in this repo `Description` everywhere means the value of a
`description` keyword, and an invariant has no `description` entry, so the name deserves a second
look.

**Parser.** `internal/parser/parser.go` parses every block as an unbounded
`for !p.check(lexer.CloseBrace) && !p.isAtEnd()` loop with a `switch`/`if-else` over the entry
keywords and a final `else` that reports `expected <entries> in <construct>` and advances one token.
`parseContext` (`:220-278`) and `parseAggregate` (`:280-323`) are the two blocks this story extends;
their messages (`:265`, `:310`) are asserted by tests. The closest existing shape to
`invariant <name> "<prose>"` is `parseTagEntry` (`:1177-1203`), an identifier-then-value entry on one
line. `parseDescriptionInto` (`:1289-1300`) is the model for reporting a malformed value exactly once
— it drains with `p.skipRestOfLine(keywordTok)` (`:1375-1379`), which also halts at `}` so the
enclosing block still closes. `checkSameLineAs` / `checkIdentifierLikeSameLineAs` (`:1351-1365`) are
the line gates that stop a declaration reaching into the next line, the defect
`tasks/learnings.md` records for `parseField`. `takePendingComments` (`:1329-1334`) attaches comments
to the node being built.

**Parser tests.** `internal/parser/parser_test.go` is one umbrella split into thirteen top-level
groups; `"contexts, aggregates and slices"` starts at `:568` and owns both constructs this story
touches. `:3272` is the guard subtest "the shared described model describes every construct that
accepts one", driven by `describableConstructs` (`:4180`) — it walks constructs carrying a
`description`, which an invariant's positional prose is not.

**Validator.** `internal/validator/validator.go` collects names across the model, then reports
cross-reference failures and the two orphan rules. It has no duplicate-name rule of any kind today.
Two conventions matter: `orphanNames` (`:110-137`) sorts by position precisely so diagnostics never
come out in Go's map order, and only the two rule-backed diagnostics set `Severity` and `RuleName`
(`:84-105`) — the reference errors leave both at their zero value, which is `diagnostic.Error`.
`internal/oracle/oracle.go` is the lex → parse → validate → lint chain every CLI command and the LSP
run through.

**Formatter.** `internal/formatter/formatter.go` `writeContext` (`:98-117`) writes the description
first, then aggregates, then slices; `writeAggregate` (`:119-130`) writes the description then
slices. `blankLineBetweenBlocks` (`:29-37`) is the separator helper. `quoted` (`:47-49`) is the only
correct way to emit a string — the language has no escape sequences, so `%q` would grow the text on
every run. Output always opens with `emod <n>`. `internal/formatter/formatter_test.go:427`
"round-trip through the parser" is the only test class that catches a formatter that mangles a
declaration; goldens alone do not.

**Glossary.** `internal/glossary/glossary.go` `newDocument` is the single walk; `RenderMarkdown`
(`markdown.go`) and `RenderJSON` (`json.go`) both start from it, and `document` / `contextSection` /
`term` carry the JSON tags directly. `contextSection.Aggregates` is a bare `[]term` (`:15`), so an
aggregate currently owns no terms of its own — listing invariants *under their aggregate* is the
first thing that changes that, and it is the one respect in which this story does not fit the
"one more field on `contextSection` plus one more `termGroup`" shape `tasks/learnings.md` predicts.
`term.Description` deliberately has no `omitempty` (`:22-28`). Heading levels and the group headings
live in `markdown.go:11-25`; `appendGroups` (`:59`) renders a heading plus its terms one level down.

**Exports.** `internal/export/export.go` keeps document types separate from the AST: `jsonContext`
(`:44-52`) carries only name, description, positions, comments and aggregates — a DCB context's
direct slices are already missing there, a pre-existing gap this story does not fix —
and `jsonAggregate` (`:54-61`) adds slices. `convertContext` / `convertAggregate` are at `:318-355`;
the CUE writers `writeContext` / `writeAggregate` at `:1218-1229` build on `writeCUEList` and
`lineIfSet`. `internal/cue/schema.cue` mirrors them at `#Aggregate` (`:87-92`) and `#Context`
(`:94-99`) and is what `emod schema` prints. Two subtests couple the three surfaces:
`internal/export/export_test.go:3324` runs `cue vet -d '#Model'` over the output, and `:3332` decodes
both exports of one fixture and requires them equal. The diagram JSON document is deliberately
forked from the model document so a new AST field cannot leak into the node-and-edge contract;
`:2809` walks the whole diagram document asserting prose appears nowhere in it, and `:2820-2836` is
the differential that opens with `require.NotEqual` before asserting everything else is equal.

**Fixtures.** `internal/test/fixtures.go` holds `HotelReservation` (no optional feature, the witness
that plain models still parse and render byte-identically), `DescribedHotelReservation` (a
description on every construct that accepts one) and `KeywordFieldSearchCatalog` (fields named after
keywords, modifier-less ones placed mid-block on purpose). `examples/dcb_model.emod` is the
lint-clean DCB-mode model — a `mode dcb` context needs tagged events and `decides_on` commands or
`internal/linter` reports `dcb/untagged-event` and friends.

**Tree-sitter.** `editors/tree-sitter-emod/grammar.js` builds every block with
`buildDescribedBlock($, ...items)` (`:1-5`) — `'{' repeat(choice($.description, ...items)) '}'` — so
entries are unordered and unbounded, matching the Go parser. `context_definition` (`:58-62`) accepts
only aggregates today and `aggregate_definition` (`:65-69`) only slices; neither `mode` nor a
context-level `slice` is in the grammar, a pre-existing narrowness left alone here. `any_identifier`
(`:211`) is what keeps keywords usable in field position. `src/` is gitignored;
`task test:grammar` regenerates before running the corpus, and the corpus lives in
`test/corpus/*.txt`.

**Not touched, deliberately.** `internal/lsp/hover.go` (`isKeyword` at `:37` is an ordinal range that
already excludes everything after `KeywordExternal`, so a new keyword is invisible to hover whatever
`keywordDescriptions` says — US-015's problem), `internal/lsp/completer.go:133-150`,
`editors/vscode/syntaxes/emod.tmLanguage.json:63` and
`editors/tree-sitter-emod/queries/highlights.scm` (US-017), `internal/importer` (diagram JSON →
AST, and the diagram JSON carries no invariants), `internal/wasm/pipeline.go` (diagram and JSON
exports only), `e2e/tests/validate.test.ts`.

---

## Tasks

### Task 1: Declare named invariants on an aggregate

**Behavior:** `invariant` becomes a keyword the lexer knows, and an `aggregate` block accepts any
number of `invariant <identifier> "<prose statement>"` entries anywhere among its other entries. Each
is recorded on the aggregate in declaration order, carrying the identifier, the prose, a source
position for each, and any comments written above it. A malformed entry reports exactly one
diagnostic and does not consume the entry on the next line. Because the keyword joins the lexer's
map, it is simultaneously usable as a field name, a field type and a field modifier.

**Acceptance Criteria:**
- [ ] An aggregate whose body declares two invariants parses with no diagnostics, and both appear on
      the aggregate in declaration order with the identifier, the prose statement and a position for
      each
- [ ] An invariant declared between the aggregate's `description` and its first slice, and another
      declared after the last slice, both parse — position within the block is free, as it is for
      `description`
- [ ] An aggregate that declares no invariant parses exactly as before: `oracle.Check` over
      `test.HotelReservation` and `test.DescribedHotelReservation` still returns no diagnostics, and
      no existing subtest in `internal/parser/parser_test.go` needs editing
- [ ] An `invariant` entry that names an identifier but no prose statement, with a further entry on
      the following line, reports exactly one diagnostic (`require.Len(t, diags, 1)`) whose message
      names the invariant construct, and the entry on the following line is still parsed onto the
      aggregate
- [ ] An `invariant` entry followed immediately by `}` reports exactly one diagnostic and the
      aggregate block still closes — the recovery stops at the brace
- [ ] The message the parser reports for an unrecognised entry inside an aggregate names `invariant`
      among the entries it accepts
- [ ] `lexer.Keywords()` contains `invariant`, and the keyword-coverage subtests in
      `internal/lexer/tokenizer_test.go`, `internal/parser/parser_test.go` and
      `internal/oracle/oracle_test.go` cover it without any of them naming a keyword literally
- [ ] A `fields` block declaring a field named `invariant`, and one using `invariant` as name, type
      and modifier at once, both parse as ordinary fields
- [ ] An invariant whose identifier spells a DSL keyword parses, the same courtesy field names get

**Affected Files/Modules:**
- `internal/lexer/token.go` — the keyword kind and its entry in the `keywords` map
- `internal/ast/ast.go` — the invariant node and the aggregate's collection of them
- `internal/parser/parser.go` — `parseAggregate` accepts the entry; the entry parser itself; the
  aggregate's "expected …" message
- `internal/parser/parser_test.go` — subtests in the `"contexts, aggregates and slices"` group
  (`:568`)

**Patterns to Follow:**
- One-line identifier-then-value entry inside a block: `parseTagEntry`,
  `internal/parser/parser.go:1177-1203`
- Report a malformed value once and recover by draining the line: `parseDescriptionInto`
  (`internal/parser/parser.go:1289-1300`) with `skipRestOfLine` (`:1375-1379`) —
  `tasks/learnings.md` "A new block entry keyword owes three things to the parser's diagnostics",
  including the `require.Len(t, diags, 1)` pin
- Gate each part of the declaration on the first token's line: `checkSameLineAs` /
  `checkIdentifierLikeSameLineAs` (`internal/parser/parser.go:1351-1365`) —
  `tasks/learnings.md` "A line-oriented declaration must gate every optional trailing token on the
  first token's line"
- Comment attachment: `takePendingComments` (`internal/parser/parser.go:1329-1334`), as
  `parseAggregate` (`:281`) already does
- Node shape: `docs/proposals/specs-and-metadata-proposal.md:354-367`; sibling AST nodes carry
  `Name`/`NamePos` plus a value and its position (`ast.TagEntry`, `internal/ast/ast.go:192-197`).
  Name the prose field for what it is — `tasks/learnings.md` "Name an extracted helper after the
  contract its callers rely on" applies to fields too, and `Description` in this repo means the
  value of a `description` keyword
- Never restate the keyword set and never range over `Kind` ordinals —
  `tasks/learnings.md` "Ask the lexer which keywords exist"
- Subtests belong to the group that owns the construct — `tasks/learnings.md` "Put a new parser
  subtest in the group that owns the construct"

**Testable:** Yes — through `lexer.Scan` + `parser.Parse` and `oracle.Check`, all exported.

**Verification:** `go test -tags unit ./internal/lexer/... ./internal/parser/...
./internal/oracle/...`; `go build ./...`.

**Depends on:** None

---

### Task 2: Declare the same invariants directly on a context

**Behavior:** A `context` block accepts the same `invariant` entries directly on the context, which
is how a DCB-mode context — which has no aggregate — declares the rules it protects. The parser
accepts them whatever the context's mode, matching how every other mode-specific construct is parsed
and left to the linter to judge. A shared fixture carrying invariants in both homes joins
`internal/test`, so the writers and renderers downstream all assert against one model.

**Acceptance Criteria:**
- [ ] A `context "X" mode dcb { ... }` declaring two invariants ahead of its slices parses with no
      diagnostics, and both appear on the context in declaration order with identifier, prose and
      positions
- [ ] Invariants declared on an aggregate stay on that aggregate and invariants declared on the
      enclosing context stay on the context, including when both use the same identifier
- [ ] An invariant on a context whose mode is `aggregate` or unset parses with no parser diagnostic
      and no new lint diagnostic — mode mismatch stays the linter's concern and this story adds no
      rule
- [ ] The message the parser reports for an unrecognised entry inside a context names `invariant`
      among the entries it accepts
- [ ] A context-level `invariant` missing its prose statement, with a further entry on the following
      line, reports exactly one diagnostic and the following entry still parses onto the context
- [ ] `internal/test/fixtures.go` gains a shared source that declares invariants on an aggregate and,
      in a `mode dcb` context, directly on the context; `oracle.Check` over it returns no diagnostics
      at all, so every downstream package can use it as a clean input
- [ ] The invariant-carrying fixture places at least one invariant mid-block, ahead of a further
      entry, rather than all of them as the last entries in their block
- [ ] `HotelReservation`, `DescribedHotelReservation` and `KeywordFieldSearchCatalog` are unchanged,
      so every existing golden keeps witnessing a model that uses no invariants

**Affected Files/Modules:**
- `internal/ast/ast.go` — the context's collection of invariants
- `internal/parser/parser.go` — `parseContext` (`:220-278`) accepts the entry; the context's
  "expected …" message
- `internal/parser/parser_test.go` — subtests in the `"contexts, aggregates and slices"` group
- `internal/test/fixtures.go` — the shared invariant-carrying source

**Patterns to Follow:**
- The entry parser written in Task 1, reused unchanged for both homes rather than copied —
  `tasks/learnings.md` "De-duplicate before a fan-out edit"
- Both slice homes and the two-context shape: `examples/dcb_model.emod` is the lint-clean DCB model
  (tagged events, `decides_on` commands) to mirror; `internal/linter/linter.go:169-245` is what
  reports `dcb-in-aggregate-mode`, `aggregate-in-dcb-mode` and `dcb/untagged-event` if the fixture
  drifts from that shape
- Fixture roles and the guard on them: `tasks/learnings.md` "Shared fixtures come in an
  unfeatured/featured pair" — `HotelReservation` keeps the unfeatured role, the new source takes the
  featured one
- Exercise the feature mid-block, never only as the last entry — `tasks/learnings.md` "Exercise an
  omitted optional part mid-block"
- Fixture prose style and comment header: `internal/test/fixtures.go:203-303`

**Testable:** Yes — through `parser.Parse` and `oracle.Check` over the new fixture.

**Verification:** `go test -tags unit ./internal/...`;
`go run ./cmd/emod validate` over a temporary file holding the new fixture, expecting exit 0.

**Depends on:** Task 1

---

### Task 3: Preserve invariants through `emod fmt`

**Behavior:** The formatter writes every declared invariant back out, so formatting a model no longer
loses the rules it protects. Invariants are emitted inside their block after the `description` and
ahead of the aggregates or slices, one per line, with the prose written as a verbatim emod string.
A model that declares no invariant formats to exactly the bytes it formatted to before.

**Acceptance Criteria:**
- [ ] Parsing the Task 2 fixture, formatting it and re-parsing yields a model whose aggregate-scoped
      and context-scoped invariants match the original in identifier, prose and declaration order
- [ ] Formatting the formatter's own output produces byte-identical text
- [ ] In the formatted output each invariant sits after its block's `description` line and before the
      block's first aggregate or slice
- [ ] A comment written above an invariant appears above it in the formatted output
- [ ] An invariant whose prose contains a backslash, a tab, a double quote, a `%` and a non-ASCII
      character survives parse → format → parse → format with identical bytes, proving the text is
      never escaped
- [ ] `internal/formatter/formatter_test.go` and `internal/cli/fmt_test.go` pass with no edit to any
      existing expected-output constant, so a model without invariants formats exactly as before
- [ ] `emod fmt --check` over an already-formatted file carrying invariants reports no change needed

**Affected Files/Modules:**
- `internal/formatter/formatter.go` — `writeContext` (`:98-117`) and `writeAggregate` (`:119-130`)
- `internal/formatter/formatter_test.go` — a round-trip subtest and the escape-hazard table
- `internal/cli/fmt_test.go` — the command-level behaviour over the invariant fixture

**Patterns to Follow:**
- `quoted` (`internal/formatter/formatter.go:47-49`) for the prose, never `%q` —
  `tasks/learnings.md` "Never write emod source with `%q`", including its obligation to carry a
  round-trip subtest per hazard character (the table at `internal/formatter/formatter_test.go`)
- Description-then-body ordering and the block separator: `writeAggregate`
  (`internal/formatter/formatter.go:119-130`), `blankLineBetweenBlocks` (`:29-37`)
- Round-trip through the parser is the assertion that catches a mangled declaration, not a golden:
  `internal/formatter/formatter_test.go:427`
- Every expected string starts with the `emod <n>` header — `tasks/learnings.md` "Formatter output
  always begins with `emod N`"
- `tasks/learnings.md` "Additive output changes owe a byte-identical receipt for models that do not
  use the feature": the untouched goldens are that receipt, and it belongs in the commit message

**Testable:** Yes — through `formatter.Format` and `cli.RunFmt`.

**Verification:** `go test -tags unit ./internal/formatter/... ./internal/cli/...`;
`go run ./cmd/emod fmt` over a temporary file holding the Task 2 fixture, then again over the result.

**Depends on:** Task 2

---

### Task 4: Reject two invariants sharing a name in one scope

**Behavior:** `emod validate` reports an error when one scope declares the same invariant identifier
twice — one `aggregate` block, or one `context` block's own invariants. Two sibling aggregates may
each declare the same name, and so may an aggregate and its enclosing context: those are separate
resolution scopes, the same ones US-006 and US-009 will resolve `rejected <name>` against. An
invariant that nothing in the model references stays silent — no error, no warning.

**Acceptance Criteria:**
- [ ] Two invariants with the same identifier in one aggregate produce exactly one diagnostic, at
      `Error` severity, positioned on the second declaration, whose message names the repeated
      identifier and the aggregate it was declared in
- [ ] Two invariants with the same identifier declared directly on one context produce the equivalent
      diagnostic naming the identifier and the context
- [ ] Two aggregates in the same context each declaring an invariant with the same identifier produce
      no diagnostic
- [ ] An aggregate and its enclosing context each declaring an invariant with the same identifier
      produce no diagnostic
- [ ] Three declarations of one identifier in one scope produce two diagnostics, one per
      redeclaration
- [ ] An invariant that no other construct in the model mentions produces no diagnostic from
      `validator.Validate` and none from `linter.Lint`
- [ ] Over a model carrying duplicates in several scopes, the diagnostics come out in declaration
      order and are identical across repeated runs — never Go's map order
- [ ] `cli.RunValidate` over a file with a duplicate exits with `ExitCode` 1 and the reported message
      contains the duplicated identifier, not merely the path and line number
- [ ] `oracle.Check` over the Task 2 fixture still returns no diagnostics

**Affected Files/Modules:**
- `internal/validator/validator.go` — the duplicate-declaration check over both invariant homes
- `internal/validator/validator_test.go` — the scope boundaries and the ordering guarantee
- `internal/cli/validate_test.go` — the diagnostic as the user receives it

**Patterns to Follow:**
- Position-ordered emission so map iteration never reaches the output: `orphanNames`,
  `internal/validator/validator.go:110-137` and the comment above it
- Diagnostic construction, severity and the `RuleName` choice: the orphan rules at
  `internal/validator/validator.go:81-105` carry a rule name; the cross-reference errors at
  `:198-261` carry none. `emod lint --explain` covers `internal/linter` rules only, so a validator
  diagnostic that no rule configuration can silence follows the cross-reference shape
- Assert the tokens that identify *this* diagnostic at the CLI layer — `tasks/learnings.md` "CLI
  diagnostic tests must assert the distinguishing message text", with
  `internal/cli/validate_test.go:253-258` as the model
- Umbrella + `t.Run` grouping and `testify/require` throughout: `internal/validator/validator_test.go`

**Testable:** Yes — through `validator.Validate`, `oracle.Check` and `cli.RunValidate`.

**Verification:** `go test -tags unit ./internal/validator/... ./internal/cli/...
./internal/oracle/...`; `go run ./cmd/emod validate` over a temporary file declaring one identifier
twice, expecting exit 1 and the identifier in the message.

**Depends on:** Task 2

---

### Task 5: List invariants under their aggregate or context in `emod glossary`

**Behavior:** The glossary lists every declared invariant beneath the aggregate or the context that
declares it, with the prose statement standing as the term's definition, in both the markdown and the
JSON rendering. This is what makes an invariant visible to the whole team before anything references
it. A model that declares none renders exactly as it renders today.

**Acceptance Criteria:**
- [ ] In the markdown, an aggregate's invariants appear beneath that aggregate's own heading, in
      declaration order, each with its prose statement as the text under its name
- [ ] In the markdown, a context's own invariants appear beneath the context and under no aggregate
- [ ] Two aggregates in one context each declaring an invariant with the same identifier list one
      invariant each, under their own aggregate
- [ ] The JSON rendering carries the same identifiers, statements, grouping and order as the markdown
      for the same model
- [ ] An invariant declared with an empty prose statement still appears, and its JSON key is present
      and empty rather than omitted — the rule `term.Description` follows
      (`internal/glossary/glossary.go:22-28`)
- [ ] Over `test.HotelReservation` and `test.DescribedHotelReservation`, both renderings are unchanged
      — no empty heading, no new key — and every existing subtest in
      `internal/glossary/glossary_test.go` and `internal/cli/glossary_test.go` passes with no edit to
      its expected output
- [ ] `cli.RunGlossary` over the Task 2 fixture in the markdown format and in the `json` format both
      list every invariant the fixture declares, under the aggregate or context that declares it
- [ ] The document is still built by a single walk of the AST shared by both renderings

**Affected Files/Modules:**
- `internal/glossary/glossary.go` — the collected document; the aggregate entry gains terms of its
  own, so `contextSection.Aggregates` can no longer be a bare `[]term` (`:15`)
- `internal/glossary/markdown.go` — the heading for the new group and its level
- `internal/glossary/json.go` — unchanged if the document carries the tags, which is the point of the
  shape
- `internal/glossary/glossary_test.go`, `internal/cli/glossary_test.go`

**Patterns to Follow:**
- One walk, two renderings, JSON tags on the collected document: `newDocument`
  (`internal/glossary/glossary.go:30-51`) and `tasks/learnings.md` "`internal/glossary` collects once
  and renders twice" — note that entry predicts "a field on `contextSection` plus one more
  `termGroup`", which covers the context-scoped half only; the aggregate-scoped half needs the
  aggregate entry itself to carry a group
- Group heading and level handling: `appendGroups` / `appendTerm` / `headingLine`
  (`internal/glossary/markdown.go:59-82`) and the level constants (`:11-15`)
- Descriptions take no `omitempty` while collections do: `internal/glossary/glossary.go:22-28`
- Do not add a "render it twice" assertion here — `tasks/learnings.md` "An assertion whose expected
  value comes from the code under test is the recurring review finding"; this package's paths never
  range over a Go map, so such an assertion cannot fail
- CLI test layout, `captureStdout`, and asserting on the decoded JSON document rather than the
  serialized text: `internal/cli/glossary_test.go`

**Testable:** Yes — through `glossary.RenderMarkdown`, `glossary.RenderJSON` and `cli.RunGlossary`.

**Verification:** `go test -tags unit ./internal/glossary/... ./internal/cli/...`;
`go run ./cmd/emod glossary <fixture file>` and the same with `-f json` piped through `jq .`.

**Depends on:** Task 2

---

### Task 6: Carry invariants through the JSON and CUE exports and the embedded schema

**Behavior:** Both model exports emit each aggregate's and each context's invariants with their
identifier and prose in declaration order, the bundled schema declares them, and the diagram
document — which is nodes and edges — carries no trace of them. A model without invariants exports
byte-identically to before.

**Acceptance Criteria:**
- [ ] The JSON export of the Task 2 fixture carries the invariants of each aggregate and of the DCB
      context, with identifier and prose, in declaration order
- [ ] The CUE export of the same fixture carries the same, and the "CUE and JSON exports describe the
      same model" subtest (`internal/export/export_test.go:3332`) passes for it
- [ ] `internal/cue/schema.cue` declares invariants as an optional key on the aggregate and the
      context definitions, and `cue vet -d '#Model'` over the export of the fixture passes
      (`internal/export/export_test.go:3324`)
- [ ] `emod schema` prints a schema that declares invariants on both definitions
- [ ] Walking the whole diagram JSON document produced from the fixture finds neither an invariant
      identifier nor an invariant statement at any key or depth
- [ ] Every diagram rendering of the fixture — drawio, SVG, mermaid and ASCII — is byte-identical to
      the rendering of the same model with its invariants stripped, and the comparison opens by
      asserting the two models actually differ
- [ ] Existing subtests in `internal/export/export_test.go` and `internal/diagram` pass with no edit
      to any expected output, so a model without invariants exports and renders exactly as before

**Affected Files/Modules:**
- `internal/export/export.go` — `jsonContext` (`:44-52`), `jsonAggregate` (`:54-61`),
  `convertContext` / `convertAggregate` (`:318-355`), `cueWriter.writeContext` / `writeAggregate`
  (`:1218-1229`)
- `internal/cue/schema.cue` — a definition for the invariant plus the key on `#Aggregate` (`:87-92`)
  and `#Context` (`:94-99`)
- `internal/export/export_test.go` — the two exports, the schema conformance, the diagram-document
  guard and the diagram differential
- `internal/diagram/contract_test.go` — the differential across the four renderers, if it is the
  better home for it

**Patterns to Follow:**
- All three surfaces land together: `tasks/learnings.md` "A new exported field must land in JSON, CUE
  and `schema.cue` in the same change"
- List emission in CUE: `writeCUEList` and `lineIfSet` as used by `writeContext` /
  `writeAggregate` (`internal/export/export.go:1218-1229`)
- Keep the diagram document forked: `jsonDiagramEvent` exists precisely so a new AST field cannot
  leak into the node-and-edge contract; the walk asserting prose appears nowhere is
  `internal/export/export_test.go:2809-2818`
- A differential must first prove its twin differs: `internal/export/export_test.go:2820-2836` and
  `internal/diagram/contract_test.go:201-206` both open with `require.NotEqual`;
  `withoutDescriptions` (`internal/diagram/contract_test.go:616`) is the strip-helper to model on,
  and it must visit both invariant homes or the receipt is vacuous —
  `tasks/learnings.md` "A differential receipt must first prove the twin actually differs"
- Name a strip-helper for what it guarantees, and prefer one that leaves the original model intact —
  `tasks/learnings.md` "Name an extracted helper after the contract its callers rely on"

**Testable:** Yes — through `export.ExportJSON`, `export.ExportCUE`, `export.ExportDiagramJSON`,
`cli.RunSchema` and the four `diagram.Export*` renderers.

**Verification:** `go test -tags unit ./internal/export/... ./internal/diagram/...
./internal/cli/...`; `go run ./cmd/emod export --format cue <fixture file>` and
`go run ./cmd/emod schema`.

**Depends on:** Task 2

---

### Task 7: Accept invariant entries in the tree-sitter grammar

**Behavior:** The tree-sitter grammar parses the invariant entry inside both an aggregate block and a
context block, so a file `emod validate` accepts is not red-squiggled by a tree-sitter-backed editor.
The grammar stays looser than the Go parser: any number of entries, in any position, alongside the
existing ones.

**Acceptance Criteria:**
- [ ] A corpus case for an aggregate declaring two invariants parses to the expected tree
- [ ] A corpus case placing an invariant before a slice and another after it parses, showing position
      within the block is free and the count is unbounded
- [ ] A corpus case for a context declaring an invariant directly parses to the expected tree
- [ ] A corpus case for a `fields` block declaring a field named `invariant` still parses as a field
      line, not as an invariant entry
- [ ] `mise exec -- task test:grammar` passes, and running it a second time leaves every tracked file
      under `editors/tree-sitter-emod/` byte-identical
- [ ] No file under `editors/tree-sitter-emod/src/` is tracked — `.gitignore` still ignores it
- [ ] No file under `editors/tree-sitter-emod/queries/` changes, and
      `editors/vscode/syntaxes/emod.tmLanguage.json` is untouched — highlighting is US-017's story

**Affected Files/Modules:**
- `editors/tree-sitter-emod/grammar.js` — the invariant rule and its place in the aggregate
  (`:65-69`) and context (`:58-62`) block bodies
- `editors/tree-sitter-emod/test/corpus/aggregate.txt`, `test/corpus/context.txt`,
  `test/corpus/fields.txt` — the new cases

**Patterns to Follow:**
- `buildDescribedBlock($, ...items)` (`editors/tree-sitter-emod/grammar.js:1-5`) is how every block
  body admits unordered, unbounded entries; an `optional(...)` in a block body would be a bug —
  `tasks/learnings.md` "The tree-sitter grammar must never be stricter than the Go parser"
- Field names must keep matching `any_identifier` (`:211`) — `tasks/learnings.md` "New DSL keywords
  must stay usable as field names", with
  `editors/tree-sitter-emod/test/corpus/version_header.txt` as the existing example of the
  field-named-after-a-keyword case
- Corpus case layout: `editors/tree-sitter-emod/test/corpus/aggregate.txt`
- Run through `mise exec --`, and do not un-ignore `src/` — `tasks/learnings.md` "Run repo tooling
  through `mise exec --`" and "Generated tree-sitter `src/` stays gitignored"

**Testable:** Yes — the corpus cases are the tests, run by `task test:grammar`.

**Verification:** `mise exec -- task test:grammar`, run twice, the second run leaving the tracked
files untouched; `git ls-files editors/tree-sitter-emod/src` returning nothing.

**Depends on:** Task 2

---

### Task 8: Document invariants in the DSL reference

**Behavior:** The DSL reference teaches the invariant: where it may be declared, what its two parts
are, that redeclaring a name in one scope is an error, that declaring one nothing references is not,
and which tools surface it. A reader learning the language finds it next to `aggregate` and
`context`, where it is declared.

**Acceptance Criteria:**
- [ ] `docs/dsl-reference.md` §4 "Bounded Contexts" gains an `### invariant` subsection covering the
      identifier-plus-prose form, the aggregate placement and the DCB-context placement
- [ ] That subsection states that two invariants sharing a name in one scope is a validation error,
      that an aggregate and its context are separate scopes, and that an invariant nothing references
      is not an error
- [ ] It states that `emod glossary` lists invariants under their aggregate or context in both
      formats, that the JSON and CUE exports carry them, and that no diagram renders them
- [ ] No `## <n>.` heading in `docs/dsl-reference.md` is added, removed or reordered, so every
      existing `(#<n>-<slug>)` link in the document still resolves to its heading
- [ ] The emod snippet in the new subsection, written to a file on its own, passes `emod validate`
      with exit 0

**Affected Files/Modules:**
- `docs/dsl-reference.md` — an `### invariant` subsection inside §4 (`:104-146`), and the consumer
  bullets in §10 (`:456-464`) if invariants belong in that list

**Patterns to Follow:**
- Subsection voice, the fenced form-then-example shape and the bullet list of consumers:
  `docs/dsl-reference.md:104-146` (§4's `context` and `aggregate`) and `:456-464` (§10's consumer
  bullets)
- A subsection inside an existing numbered section is safe; a new numbered section renumbers
  everything below it and breaks four in-document links — `tasks/learnings.md`
  "`docs/dsl-reference.md` anchors embed the section number"
- Write the document as if it were its first version: no note of what the reference used to say

**Testable:** No — prose only; correctness is that the documented snippet validates and no anchor
breaks.

**Verification:** `go run ./cmd/emod validate <snippet written to a temp file>`; reconcile the
`^## [0-9]+\.` heading list against the `\(#[0-9]+-` link list in `docs/dsl-reference.md`.

**Depends on:** Tasks 3, 4, 5, 6

---

## Summary

**Total tasks:** 8

**Ordering rationale:** dependency-first, with the language surface settled before anything reads it.
Tasks 1 and 2 add the construct in its two homes — the aggregate first because it is the simpler
scope and carries the keyword, the lexer entry and the entry parser, then the context, which reuses
that parser and lands the shared fixture every later task asserts against. Task 3 comes next because
a formatter that does not know a construct silently deletes it, which is the most damaging gap the
story can leave open. Task 4 delivers the one criterion that produces a diagnostic. Task 5 delivers
the last criterion the story names. Tasks 6 and 7 close the two surfaces the repo's conventions
require of any new construct — the exports plus the schema, and a grammar that is never stricter than
the parser — and both depend only on Task 2, so they can run alongside 3-5. Task 8 documents the
finished surface.

**Coverage of the story's acceptance criteria:**

| Criterion | Task |
|---|---|
| An `aggregate` accepts `invariant <identifier> "<prose statement>"` entries | 1 |
| A context in DCB mode accepts the same entries directly on the context | 2 |
| Two invariants with the same name in the same scope produce a clear validation error | 4 |
| A declared invariant that nothing references is not a validation error | 4 |
| `emod glossary` lists invariants under their aggregate or context | 5 |
| Every existing `.emod` file stays valid with unchanged meaning (story-wide constraint) | 1 (keyword stays usable as a field name), 3 (formatter goldens), 5 (glossary goldens), 6 (export and diagram receipts) |

Tasks 3, 6, 7 and 8 carry no story criterion of their own — they are the fan-out this repo requires
of any new construct: `emod fmt` must not drop it, the JSON/CUE/schema trio moves together, the
tree-sitter grammar must not reject what `emod validate` accepts, and the reference must teach it.

Nothing from the story is deferred. What US-005 deliberately leaves to later stories: `rejected`
references from specs (US-006, US-007) and flow rejection edges (US-009), which are what make an
invariant reference checkable; `spec/invariant-never-exercised` (US-008), which is the lint rule that
eventually flags the unreferenced invariant Task 4 keeps silent; invariants in
`examples/*.emod` (US-018); LSP hover, completion and navigation over invariants (US-015); and
keyword highlighting in the VS Code grammar and the tree-sitter queries (US-017).

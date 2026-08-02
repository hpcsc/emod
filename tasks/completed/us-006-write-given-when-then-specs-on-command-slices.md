# US-006: Write Given-When-Then specs on command slices

## Progress
- [x] Task 1: Parse `spec` blocks with `given`, `when` and a `then` event list
- [x] Task 2: Parse `then rejected <invariantName>` and share a spec-carrying fixture
- [x] Task 3: Reject spec references to events and commands the model does not define
- [x] Task 4: Resolve `then rejected` against the enclosing aggregate or DCB context
- [x] Task 5: Preserve specs through `emod fmt`
- [x] Task 6: Carry specs through the JSON and CUE exports and the embedded schema
- [x] Task 7: Accept spec blocks in the tree-sitter grammar
- [x] Task 8: Document specs in the DSL reference

---

## Story Reference

`user-stories/specs-and-metadata.md` → **US-006: Write Given-When-Then specs on command slices**
(sixth story of "Specs, Invariants, and Model Metadata"). Design notes:
`docs/proposals/specs-and-metadata-proposal.md` §4 "Given-When-Then Specs" (lines 127-193), the
validation list at lines 223-232, the AST shape at lines 313-345, and the keyword list at line 374.

**In scope:** any number of `spec "<name>" { ... }` blocks inside a slice; `given` as an ordered
event list, with `given []` and an omitted `given` meaning the same empty history; `when` naming the
command under test; `then [EventA, EventB]` as the success outcome; `then rejected <invariantName>`
as the failure outcome, resolved against the enclosing aggregate or, for a slice declared directly
on a context, that context; a validation error with location for an unresolved invariant name and
for any event or command name in a spec that the model does not define. Carried with them, because
the repo's writers would otherwise silently drop the new construct: `emod fmt` round-trip, the JSON
and CUE exports plus the embedded schema, the tree-sitter grammar (which must never reject what
`emod validate` accepts), and the DSL reference.

**Out of scope:** the view, automation and translation spec shapes — `then view <Name>`,
`then command <Name>`, and the rule that a `then` shape must match the slice pattern (US-007);
example payload literals `{ field: value }` on element references (US-010); the four `spec/*` lint
rules (US-008), so this story adds no lint rule and no severity configuration; flow rejection edges
(US-009); `given` / `when` / `then` alignment inside a formatted spec and the wider canonical
attribute order (US-014), so the formatter here emits a canonical one-line form per entry and leaves
column alignment alone; LSP hover, completion and navigation over specs (US-015); syntax
highlighting in `editors/vscode/syntaxes/emod.tmLanguage.json` and
`editors/tree-sitter-emod/queries/highlights.scm` (US-017); specs in `examples/*.emod` (US-018).

**Open questions, decided.** Three shapes the story does not name, each decided so that US-007 and
US-010 remain additive:

1. *A spec block is accepted inside any slice, not only a command slice.* A view slice's spec omits
   `when` and an automation slice's `when` names an event — both are US-007's. Parsing a spec the
   same way in every slice means US-007 adds `then` shapes and a shape-versus-pattern rule, not a
   second parse path. This story therefore reports nothing when a spec omits `when`, omits `then`,
   or names an event where a command is expected: US-007 owns that judgement, and Task 3 checks only
   that names it *does* see are defined.
2. *An entry written twice in one spec block keeps the last one*, the way `description` already
   behaves, and the block loop imposes no arity — `tasks/learnings.md` records that every block body
   in this repo is an unbounded loop and that the tree-sitter grammar must not be stricter.
3. *`then rejected <name>` resolves in exactly one scope* — the enclosing aggregate for a slice
   nested in an aggregate, the enclosing context for a slice declared directly on a context. An
   aggregate and its context are separate scopes and so are two sibling aggregates, which is the
   rule US-005 established for redeclaration; a name declared one level up does not resolve.

**Overarching constraint:** every existing `.emod` file stays valid with unchanged meaning. That is
load-bearing in four places here — the five new keywords must stay usable as field names, `emod fmt`
must still produce its current bytes for a model with no specs, no existing golden in
`internal/formatter`, `internal/export`, `internal/diagram`, `internal/glossary` or `internal/cli`
may need editing, and a slice that declares no spec must validate exactly as before.

**Learnings folded in** from `tasks/learnings.md`: ask the lexer which keywords exist and never
restate the set; a new keyword must stay usable as a field name on both the Go and the tree-sitter
side; keyword surfaces fan out past the lexer, so decide per surface rather than assume; a new block
entry keyword owes three things to the parser's diagnostics; a line-oriented declaration must gate
every optional trailing token on the first token's line; exercise an omitted optional part
mid-block; put a new parser subtest in the group that owns the construct; never write emod source
with `%q`; a fmt golden is never the input re-indented; additive output changes owe a byte-identical
receipt; a differential receipt must first prove the twin differs; a new exported field lands in
JSON, CUE and `schema.cue` together; JSON and CUE order their keys differently; a new shared fixture
owes `internal/oracle` a zero-diagnostic subtest; shared fixtures come in an unfeatured/featured
pair; `RuleName` marks a diagnostic `emod lint --explain` can describe; a second `require.Contains`
on one message is often shadowed by the first; CLI diagnostic tests must assert the distinguishing
message text; an assertion whose expected value comes from the code under test cannot fail; generated
tree-sitter `src/` stays gitignored and repo tooling runs through `mise exec --`;
`docs/dsl-reference.md` anchors embed the section number; acceptance criteria never reference commit
or branch state.

---

## Codebase Context

**Lexer.** `internal/lexer/token.go` holds one `keywords` map (`:66-98`, thirty-one spellings today,
`invariant` the most recent) and derives `Keywords()` and `Kind.IsKeyword()` from it. Four
keyword-coverage tests iterate `lexer.Keywords()` and name no keyword —
`internal/lexer/tokenizer_test.go:14`, `internal/parser/parser_test.go:225` and `:243`,
`internal/oracle/oracle_test.go:44` — so the five spellings this story adds fall under "usable as a
field name, type and modifier" the moment the map learns them. `checkIdentifierLike`
(`internal/parser/parser.go:1390`) is what makes that true on the parser side, and it asks
`IsKeyword()` rather than comparing ordinals.

**AST.** `internal/ast/ast.go` has no spec node. `Slice` (`:76-92`) carries `Comments`,
`Name`/`NamePos`, `Description`/`DescriptionPos`, a `Trigger` and one collection per element kind,
plus open and close positions. `Invariant` (`:68-74`) is the node US-005 added — comments, name and
statement, each with a position — and is the shape a sibling one-line construct takes. The proposal
puts `Specs []*Spec` on `Slice` and gives a spec a name, an ordered `Given`, a single `When`, and a
`Then` that is one of several outcome kinds (`docs/proposals/specs-and-metadata-proposal.md:313-345`).
US-010 later hangs a payload off every event and command reference in a spec, so a reference wants to
be a node with a name and a position rather than a bare string; the outcome kinds are a closed set
US-007 extends with two more.

**Parser.** `internal/parser/parser.go` parses every block as an unbounded
`for !p.check(lexer.CloseBrace) && !p.isAtEnd()` loop with an `if`/`switch` over the entry keywords
and a final `else` reporting `expected <entries> in <construct>`. `parseSlice` (`:360-425`) is the
block this story extends, and its message (`:412`) is asserted by tests. Three existing shapes cover
everything a spec entry needs: `parseInvariant` (`:333-358`) is the one-line
keyword-identifier-value entry, gating each part on the keyword token's line and draining with
`skipRestOfLineOrBlockEnd` (`:1410-1416`); `parseDecidesOnEvents` (`:594-626`) and `parseSubscribes`
(`:1089-1121`) are the two bracketed comma-separated identifier lists, both accepting an empty list
and both recovering inside the brackets; `parseDescriptionInto` (`:1324-1335`) is the model for
reporting a malformed value exactly once. Commands and events are declared with `lexer.Identifier`
names (`parseCommand:499`, `parseEvent:781`), while invariant names are identifier-*like*
(`:337`), so references should match the thing they refer to. `takePendingComments` (`:1364-1369`)
attaches comments to the node being built.

**Parser tests.** `internal/parser/parser_test.go` is one umbrella split into thirteen top-level
groups; `"contexts, aggregates and slices"` (`:568`) owns the slice, and `"error reporting"`
(`:2110`) owns the recovery and message shapes. The keyword-as-field-name subtests sit in
`"version header"` (`:225`, `:243`) for historical reasons and iterate `lexer.Keywords()`.

**Validator.** `internal/validator/validator.go` builds a `modelIndex` (`:51-77`) that flattens every
slice in the model into one list — `index.slices` loses which aggregate or context a slice came from,
which is exactly what `then rejected` resolution needs, so that check wants its own scope-aware walk
rather than another pass over the index. Command and event names in the index are model-wide and
unqualified (`:79-115`), which is why a flow may name an event declared in another context.
`referenceDiagnostics` (`:236-268`) is the shape for "X %q does not exist" at the reference's
position; `errorAt` (`:136-144`) is the constructor, and it sets `Severity: diagnostic.Error` and no
`RuleName` — `emod lint --explain` resolves only `internal/linter` rules, so a hard error carries no
rule name. `redeclaredInvariantDiagnostics` (`:200-234`) shows the scope-aware walk over both
invariant homes and sorts by `comparePositions` (`:179-187`) so map order never reaches the output.

**Validator tests.** `internal/validator/validator_test.go` groups by rule; `"duplicate invariant
declarations"` (`:1223`) is the US-005 group and its ordering subtest (`:1408`) asserts whole
formatted diagnostic lines with `require.Equal` rather than layering `require.Contains` calls —
`tasks/learnings.md` records that a second `Contains` on one message is often shadowed by the first.

**Formatter.** `internal/formatter/formatter.go` `writeSlice` (`:141-186`) emits a slice's entries in
a fixed kind order — description, trigger, commands, events, views, automations, translations, flows
last — separated by `blankLineBetweenBlocks` (`:29-37`). `writeInvariants` (`:64-69`) is the
one-line-entry writer US-005 added; `quoted` (`:47-49`) is the only correct way to emit a string,
since the language has no escape sequences. Output always opens with `emod <n>`.
`internal/formatter/formatter_test.go:427` `"round-trip through the parser"` is the only test class
that catches a formatter mangling a declaration. `internal/cli/fmt_test.go` pins canonical
`*FormattedEmod` constants (`:118`, `:135`) and feeds them to `requireFmtSettlesOn` (`:381`); passing
an input fixture back as the expected value leaves a subtest that also passes when nothing is written.

**Exports.** `internal/export/export.go` keeps document types separate from the AST: `jsonSlice`
(`:71-86`) lists one collection per element kind, `jsonInvariant` (`:66-70`) is the node US-005 added
and `convertInvariants` (`:342`) its converter; the CUE writers `writeContext` / `writeAggregate`
(`:1247`, `:1255`) build on `writeCUEList` and `lineIfSet`, with `writeInvariant` at `:1259`.
`internal/cue/schema.cue` mirrors them — `#Slice` at `:73-85`, `#Invariant` at `:87-91` — and is what
`emod schema` prints. Three subtests couple the surfaces: `internal/export/export_test.go:3362` runs
`cue vet -d '#Model'` over the export of the invariant fixture, `:3366` decodes both exports of one
model and requires them equal, and `:3379` pins that the two formats agree on invariants. The diagram
JSON document is deliberately forked so a new AST field cannot leak into the node-and-edge contract;
`:2851` is the guard that walks the whole diagram document (`invariantTextAnywhere`, `:3641`) after
first proving the same search finds the text in the model document.

**Diagrams.** `internal/diagram/contract_test.go` runs one table across drawio, SVG, mermaid and
ASCII. `"declaring invariants leaves the picture untouched"` (`:207`) is the differential to copy:
it opens with `require.NotEqual` on the two models, checks the twin lost the feature in *both* homes
against a transcribed list (`libraryLendingInvariantNames`, `:679`), then requires the two renderings
equal. `withoutInvariants` (`:692`) returns a copy rather than stripping in place, precisely so the
caller is not comparing a model with itself.

**Glossary.** `internal/glossary/glossary.go` `newDocument` is the single walk feeding both
`RenderMarkdown` and `RenderJSON`. A slice, trigger, automation and translation contribute no term of
their own, and a spec is likewise a scenario rather than a term of the ubiquitous language, so the
glossary gains nothing here — its existing goldens are the receipt that a spec-carrying model renders
the same vocabulary as one without.

**Fixtures.** `internal/test/fixtures.go` holds `HotelReservation` (uses no optional feature),
`DescribedHotelReservation`, `KeywordFieldSearchCatalog` and `InvariantLibraryLending` (`:305-411`),
which declares invariants on an aggregate and on a `mode dcb` context and places each ahead of a
later entry on purpose. `internal/test/models.go:25` is its parsed-model helper.
`internal/oracle/oracle_test.go:24` `"clean input"` holds one `require.Empty` subtest per fixture,
because `oracle.Check` runs lexer, parser, validator *and* linter — a `mode dcb` fixture needs tagged
events and a `decides_on` reaching them or `dcb/untagged-event` and `dcb/orphan-tag-key` fire.

**Tree-sitter.** `editors/tree-sitter-emod/grammar.js` builds every block with
`buildDescribedBlock($, ...items)` (`:1-5`), so entries are unordered and unbounded.
`slice_definition` (`:78-82`) admits `_slice_item` (`:85-94`); `subscribes_block` (`:170-176`) is the
bracketed comma-separated list shape; `any_identifier` (`:217`) is what keeps keywords usable in
field position, and `field_line` (`:125-129`) uses `prec.right` for the optional modifier. `src/` is
gitignored; `task test:grammar` regenerates before running the corpus in `test/corpus/*.txt`.

**Not touched, deliberately.** `internal/linter` (US-008 owns every `spec/*` rule, and this story
must add none); `internal/glossary`; `internal/cli/slices.go`, whose `detectPattern` classifies a
slice by the elements it declares and which a spec does not change; `internal/importer` and
`internal/wasm/pipeline.go`, both of which move the diagram JSON document that Task 6 keeps free of
specs; `internal/lsp` (`isKeyword`, `hover.go:37`, is an ordinal range that already excludes
everything past `KeywordExternal`, so a new keyword is invisible to hover whatever
`keywordDescriptions` says — US-015's problem, and `completer.go` likewise);
`editors/vscode/syntaxes/emod.tmLanguage.json` and `editors/tree-sitter-emod/queries/highlights.scm`
(US-017); `examples/*.emod` (US-018); `e2e/tests`.

---

## Tasks

### Task 1: Parse `spec` blocks with `given`, `when` and a `then` event list

**Behavior:** `spec`, `given`, `when` and `then` become keywords the lexer knows, and a slice accepts
any number of `spec "<name>" { ... }` blocks anywhere among its other entries. Inside a block,
`given [EventA, EventB]` records an ordered history, `when <CommandName>` records the command under
test, and `then [EventA, EventB]` records the events appended on success. An omitted `given` and
`given []` both mean the empty history. Each part is recorded with a source position, and comments
written above a spec attach to it. A malformed entry reports exactly one diagnostic and does not
consume the entry on the following line. Because the keywords join the lexer's map, all four are
simultaneously usable as field names, field types and field modifiers.

**Acceptance Criteria:**
- [ ] A slice whose body declares two spec blocks parses with no diagnostics, and both appear on the
      slice in declaration order with their names and a position for each
- [ ] A spec declaring `given [CopyBorrowed, CopyReturned]` records both event names in that order,
      each with its own position
- [ ] A spec written with `given []` and the same spec written with no `given` entry at all both
      parse with no diagnostics and yield an empty history, and the two parsed specs are equal in
      every respect other than the positions their entries were read from
- [ ] `when BorrowCopy` records the command name and its position; `then [CopyBorrowed]` records the
      event names in declaration order, each with its position
- [ ] A spec block declared between two other slice entries, and another declared after the last
      entry, both parse — position within the slice is free, as it is for `description`
- [ ] `when` written ahead of `given` in one block parses the same as the canonical order, and an
      entry written twice in one block leaves the block with the value written last
- [ ] A comment written above a spec block appears on that spec
- [ ] A `when` entry naming no command, with a further entry on the following line, reports exactly
      one diagnostic (`require.Len(t, diags, 1)`) whose message names the spec construct, and the
      entry on the following line is still parsed onto the spec
- [ ] A `given` entry whose bracket list is never closed reports exactly one diagnostic and the
      enclosing slice block still closes
- [ ] The message the parser reports for an unrecognised entry inside a slice names `spec` among the
      entries it accepts, and the message for an unrecognised entry inside a spec block names the
      entries a spec accepts
- [ ] `lexer.Keywords()` contains `spec`, `given`, `when` and `then`, and the keyword-coverage
      subtests in `internal/lexer/tokenizer_test.go`, `internal/parser/parser_test.go` and
      `internal/oracle/oracle_test.go` cover them without any of those tests naming a keyword
- [ ] A `fields` block declaring fields named `spec`, `given`, `when` and `then`, and one using each
      as name, type and modifier at once, all parse as ordinary fields
- [ ] A slice that declares no spec parses exactly as before: `oracle.Check` over
      `test.HotelReservation`, `test.DescribedHotelReservation`, `test.KeywordFieldSearchCatalog` and
      `test.InvariantLibraryLending` still returns no diagnostics, and no existing subtest in
      `internal/parser/parser_test.go` needs editing

**Affected Files/Modules:**
- `internal/lexer/token.go` — the four keyword kinds and their entries in the `keywords` map
  (`:66-98`)
- `internal/ast/ast.go` — the spec node, the node standing for one event or command reference, the
  success outcome, and the slice's collection of specs (`Slice`, `:76-92`)
- `internal/parser/parser.go` — `parseSlice` (`:360-425`) accepts the entry; the spec block parser and
  its entry parsers; the slice's "expected …" message (`:412`)
- `internal/parser/parser_test.go` — subtests in the `"contexts, aggregates and slices"` group
  (`:568`) and, for the recovery and message shapes, `"error reporting"` (`:2110`)

**Patterns to Follow:**
- Block construct with a quoted name, an unbounded entry loop and an unclosed-brace message:
  `parseSlice` (`internal/parser/parser.go:360-425`)
- One-line entry gated on the keyword token's line, recovering with `skipRestOfLineOrBlockEnd`:
  `parseInvariant` (`internal/parser/parser.go:333-358`) and `parseDescriptionInto` (`:1324-1335`) —
  `tasks/learnings.md` "A new block entry keyword owes three things to the parser's diagnostics",
  including the `require.Len(t, diags, 1)` pin, and "A line-oriented declaration must gate every
  optional trailing token on the first token's line"
- Bracketed, comma-separated, possibly empty identifier list with in-bracket recovery:
  `parseDecidesOnEvents` (`internal/parser/parser.go:594-626`) and `parseSubscribes` (`:1089-1121`)
- A reference matches how the thing it names is declared: commands and events are declared with
  `lexer.Identifier` (`parseCommand:499`, `parseEvent:781`)
- Comment attachment: `takePendingComments` (`internal/parser/parser.go:1364-1369`)
- Node shape: `docs/proposals/specs-and-metadata-proposal.md:313-345`, and `ast.Invariant`
  (`internal/ast/ast.go:68-74`) for the name-plus-position convention. US-010 adds a payload to every
  event and command reference and US-007 adds two more outcome kinds, so give a reference its own
  node type and keep the outcome a closed set — both extensions should then add a case, not change a
  collection's element type
- Never restate the keyword set and never range over `Kind` ordinals — `tasks/learnings.md` "Ask the
  lexer which keywords exist"; note also "Keyword surfaces fan out past the lexer", whose VS Code,
  hover and completion surfaces this story deliberately leaves to US-015 and US-017
- Subtests belong to the group that owns the construct — `tasks/learnings.md` "Put a new parser
  subtest in the group that owns the construct"

**Testable:** Yes — through `lexer.Scan` + `parser.Parse` and `oracle.Check`, all exported.

**Verification:** `go test -tags unit ./internal/lexer/... ./internal/parser/...
./internal/oracle/...`; `go build ./...`.

**Depends on:** None

---

### Task 2: Parse `then rejected <invariantName>` and share a spec-carrying fixture

**Behavior:** `rejected` becomes a keyword, and `then rejected <invariantName>` records the second
outcome a command spec can state: the command fails and nothing is appended. The parser tells the two
outcomes apart by what follows `then`, records the invariant name with its position, and accepts the
entry the same way in a slice nested in an aggregate and in one declared directly on a context — the
scope the name resolves in is Task 4's judgement, not the parser's. A shared fixture carrying specs
of both shapes in both slice homes joins `internal/test`, so every writer and renderer downstream
asserts against one model.

**Acceptance Criteria:**
- [ ] A spec whose `then` reads `rejected OneCopyPerLoan` parses with no diagnostics and records the
      invariant name with its position, as an outcome distinguishable from an event-list outcome
      without inspecting the name
- [ ] A spec inside a slice declared directly on a `mode dcb` context, whose `then` names an
      invariant declared on that context, parses with no parser diagnostic
- [ ] An invariant name that spells a DSL keyword is accepted after `rejected`, the same courtesy
      `invariant <name>` itself gets
- [ ] `then rejected` naming nothing, with a further entry on the following line, reports exactly one
      diagnostic and the entry on the following line is still parsed onto the spec
- [ ] `then` followed by neither a bracket list nor `rejected` reports exactly one diagnostic whose
      message names the outcomes a `then` accepts, and the enclosing spec block still closes
- [ ] `lexer.Keywords()` contains `rejected`, a `fields` block may declare a field named `rejected`,
      and the keyword-coverage subtests cover it without naming a keyword
- [ ] `internal/test/fixtures.go` gains a shared source declaring specs in an aggregate-nested slice
      and in a slice on a `mode dcb` context, together with the invariants their rejections name, and
      `internal/test/models.go` gains its parsed-model helper alongside
      `InvariantLibraryLendingModel` (`:25`)
- [ ] That fixture exercises, across its specs: a `given` list of more than one event, `given []`, an
      omitted `given`, a `then` event list, and `then rejected`
- [ ] At least one of its specs writes `then` ahead of `when`, and at least one spec block sits ahead
      of a further slice entry rather than last — an entry that runs on into what follows it is only
      caught when something follows it
- [ ] `oracle.Check` over the fixture returns no diagnostics at all, and
      `internal/oracle/oracle_test.go` `"clean input"` (`:24`) carries that subtest
- [ ] `HotelReservation`, `DescribedHotelReservation`, `KeywordFieldSearchCatalog` and
      `InvariantLibraryLending` are unchanged, so every existing golden keeps witnessing a model that
      declares no spec

**Affected Files/Modules:**
- `internal/lexer/token.go` — the `rejected` keyword kind and map entry
- `internal/ast/ast.go` — the rejection outcome
- `internal/parser/parser.go` — the `then` entry parser gains its second shape
- `internal/parser/parser_test.go` — subtests in `"contexts, aggregates and slices"` and
  `"error reporting"`
- `internal/test/fixtures.go`, `internal/test/models.go` — the shared spec-carrying source and its
  model helper
- `internal/oracle/oracle_test.go` — the zero-diagnostic subtest for the new fixture

**Patterns to Follow:**
- The spec entry parser written in Task 1, extended rather than forked —
  `tasks/learnings.md` "De-duplicate before a fan-out edit"
- Identifier-like name after a keyword, gated on the keyword token's line: `parseInvariant`
  (`internal/parser/parser.go:333-358`), which is also what declares the names this outcome refers to
- Fixture roles: `tasks/learnings.md` "Shared fixtures come in an unfeatured/featured pair" —
  `InvariantLibraryLending` keeps its role as the model with invariants and no specs, which Task 6's
  differential needs, so the new source is a sibling rather than an edit
- Exercise an omitted optional part mid-block, never as the last entry — `tasks/learnings.md`
- A `mode dcb` fixture needs tagged events and a `decides_on` reaching them or the linter fires:
  `internal/test/fixtures.go:357-410` and `tasks/learnings.md` "A new shared fixture owes
  `internal/oracle` a zero-diagnostic subtest"
- Fixture prose style and comment header: `internal/test/fixtures.go:305-311`

**Testable:** Yes — through `parser.Parse` and `oracle.Check` over the new fixture.

**Verification:** `go test -tags unit ./internal/...`; `go run ./cmd/emod validate` over
a temporary file holding the new fixture, expecting exit 0.

**Depends on:** Task 1

---

### Task 3: Reject spec references to events and commands the model does not define

**Behavior:** `emod validate` reports an error for every event named in a `given` or a `then` list,
and every command named in a `when`, that no construct in the model declares. Each is reported at the
reference's own position and names the missing construct, so an author fixing a typo is pointed at
the word they mistyped. References are unqualified and resolved model-wide, the same rule flows and
automations already follow.

**Acceptance Criteria:**
- [ ] An event named in `given` that the model does not declare produces exactly one diagnostic, at
      `Error` severity, positioned on that reference, naming the event
- [ ] An event named in `then` that the model does not declare produces the equivalent diagnostic
- [ ] A command named in `when` that the model does not declare produces the equivalent diagnostic,
      naming the command and distinguishing it from a missing event
- [ ] An event or command declared in a different context than the spec still resolves, producing no
      diagnostic — spec references are unqualified and model-wide
- [ ] An event contributed by a translation's nested event resolves, the same way it does for a flow
- [ ] A spec naming several undefined constructs produces one diagnostic per reference, in
      declaration order, identical across repeated runs of `validator.Validate` over the same model
- [ ] These diagnostics carry no `RuleName`, so `emod lint --explain` gains nothing to answer for and
      no lint rule is added by this task
- [ ] `cli.RunValidate` over a file whose spec names an undefined event exits with `ExitCode` 1 and
      the reported message contains the undefined name and the kind of construct it was looked up as,
      not merely the path and line number
- [ ] `oracle.Check` over the Task 2 fixture, and over every fixture that declares no spec, still
      returns no diagnostics

**Affected Files/Modules:**
- `internal/validator/validator.go` — the spec reference check, alongside `referenceDiagnostics`
  (`:236-268`)
- `internal/validator/validator_test.go` — a group for spec references
- `internal/cli/validate_test.go` — the diagnostic as the user receives it

**Patterns to Follow:**
- "X %q does not exist" at the reference's position: `referenceDiagnostics`
  (`internal/validator/validator.go:236-268`) and `errorAt` (`:136-144`), which already sets
  `Error` severity and leaves `RuleName` empty
- Which names exist, and that they are model-wide: `modelIndex.collect`
  (`internal/validator/validator.go:79-115`), including the translation's nested event
- No `RuleName` on a diagnostic no configuration can silence — `tasks/learnings.md` "`RuleName` marks
  a diagnostic `emod lint --explain` can describe"
- Assert whole formatted diagnostic lines rather than layering `Contains` calls:
  `internal/validator/validator_test.go:1408` and `tasks/learnings.md` "A second `require.Contains`
  on one message is often shadowed by the first"
- Assert the tokens that identify *this* diagnostic at the CLI layer — `tasks/learnings.md` "CLI
  diagnostic tests must assert the distinguishing message text", with
  `internal/cli/validate_test.go:253-258` as the model
- Umbrella + `t.Run` grouping and `testify/require` throughout:
  `internal/validator/validator_test.go`

**Testable:** Yes — through `validator.Validate`, `oracle.Check` and `cli.RunValidate`.

**Verification:** `go test -tags unit ./internal/validator/... ./internal/cli/...
./internal/oracle/...`; `go run ./cmd/emod validate` over a temporary file whose spec names a
misspelled event, expecting exit 1 and the misspelled name in the message.

**Depends on:** Task 2

---

### Task 4: Resolve `then rejected` against the enclosing aggregate or DCB context

**Behavior:** `emod validate` resolves the name in `then rejected <invariantName>` against the
invariants of the scope that owns the slice — the enclosing aggregate for a slice nested in one, the
enclosing context for a slice declared directly on a context. A name no invariant in that scope
declares is an error positioned on the reference, naming both the invariant and the scope it was
looked up in. Scopes do not nest: a name declared on a sibling aggregate, or on the context above the
aggregate that owns the slice, does not resolve, which is the same boundary US-005 draws for
redeclaration.

**Acceptance Criteria:**
- [ ] A `rejected` naming an invariant declared on the aggregate that encloses the slice produces no
      diagnostic
- [ ] A `rejected` in a slice declared directly on a context, naming an invariant declared on that
      context, produces no diagnostic
- [ ] A `rejected` naming an invariant no scope declares produces exactly one diagnostic, at `Error`
      severity, positioned on the reference, whose whole formatted line names the invariant, the kind
      of scope and the scope's name
- [ ] A `rejected` naming an invariant declared on a sibling aggregate of the one enclosing the slice
      produces that diagnostic — sibling aggregates are separate scopes
- [ ] A `rejected` in an aggregate-nested slice naming an invariant declared on the enclosing context
      produces that diagnostic, and the mirror case — a slice directly on a context naming an
      invariant on one of that context's aggregates — does too
- [ ] Two scopes declaring the same invariant name each resolve their own slices' rejections, with no
      diagnostic from either
- [ ] A model with several unresolved rejections reports them in declaration order, identical across
      repeated runs
- [ ] `cli.RunValidate` over a file with an unresolved rejection exits with `ExitCode` 1 and the
      reported message contains the invariant name and the scope name
- [ ] `oracle.Check` over the Task 2 fixture still returns no diagnostics

**Affected Files/Modules:**
- `internal/validator/validator.go` — a scope-aware walk over contexts, their aggregates and the
  slices each owns; `modelIndex` (`:51-77`) flattens that structure away, so this check reads the
  model rather than the index
- `internal/validator/validator_test.go` — the scope boundaries and the ordering guarantee
- `internal/cli/validate_test.go` — the diagnostic as the user receives it

**Patterns to Follow:**
- The scope-aware walk over both invariant homes, and sorting by position so map iteration never
  reaches the output: `redeclaredInvariantDiagnostics` and `redeclarationsIn`
  (`internal/validator/validator.go:200-234`), `comparePositions` (`:179-187`), and the comment above
  `orphanNames` (`:157-160`) explaining why
- The scope rule itself, and the wording that names a symbol and its scope together:
  `internal/validator/validator.go:195-199` and the message at `:215-216`
- `errorAt` (`internal/validator/validator.go:136-144`) for severity and the empty `RuleName`
- Assert the whole formatted line rather than two `require.Contains` calls whose needles nest:
  `internal/validator/validator_test.go:1408` and `tasks/learnings.md` "A second `require.Contains`
  on one message is often shadowed by the first"
- CLI-layer assertion content: `tasks/learnings.md` "CLI diagnostic tests must assert the
  distinguishing message text", `internal/cli/validate_test.go:253-258`

**Testable:** Yes — through `validator.Validate`, `oracle.Check` and `cli.RunValidate`.

**Verification:** `go test -tags unit ./internal/validator/... ./internal/cli/...
./internal/oracle/...`; `go run ./cmd/emod validate` over a temporary file whose `rejected` names an
invariant declared one scope up, expecting exit 1.

**Depends on:** Task 2

---

### Task 5: Preserve specs through `emod fmt`

**Behavior:** The formatter writes every spec back out, so formatting a model no longer loses the
behaviour it describes. Specs are emitted inside their slice after every other entry, each as a block
holding at most one `given`, one `when` and one `then` line, with the spec name and any prose written
as verbatim emod strings. Because `given []` and an omitted `given` mean the same thing, they format
to the same text. A model that declares no spec formats to exactly the bytes it formatted to before.

**Acceptance Criteria:**
- [ ] Parsing the Task 2 fixture, formatting it and re-parsing yields a model whose specs match the
      original in name, declaration order, given events and their order, when command, and outcome —
      including which of the two outcomes each spec states
- [ ] Formatting the formatter's own output produces byte-identical text
- [ ] In the formatted output every spec block sits after its slice's other entries, and a slice's
      specs keep their declaration order
- [ ] A spec written with `given []` and the same spec written with no `given` entry format to
      identical text
- [ ] A spec whose entries are written out of canonical order formats to the canonical order, and
      re-parsing the result yields the same spec
- [ ] A comment written above a spec block appears above it in the formatted output
- [ ] A spec name containing a backslash, a tab, a double quote, a `%` and a non-ASCII character
      survives parse → format → parse → format with identical bytes, proving the text is never
      escaped
- [ ] `internal/formatter/formatter_test.go` and `internal/cli/fmt_test.go` pass with no edit to any
      existing expected-output constant, so a model without specs formats exactly as before
- [ ] `emod fmt --check` over an already-formatted file carrying specs reports no change needed, and
      the file on disk is unchanged

**Affected Files/Modules:**
- `internal/formatter/formatter.go` — `writeSlice` (`:141-186`) and the spec writer
- `internal/formatter/formatter_test.go` — a round-trip subtest in `"round-trip through the parser"`
  (`:427`) and the escape-hazard table
- `internal/cli/fmt_test.go` — a canonical formatted constant for the spec fixture and the
  command-level behaviour over it

**Patterns to Follow:**
- Entry ordering, the block separator and the one-line-entry writer: `writeSlice`
  (`internal/formatter/formatter.go:141-186`), `blankLineBetweenBlocks` (`:29-37`),
  `writeInvariants` (`:64-69`)
- Bracketed list rendering: `writeDecidesOn`'s events line (`internal/formatter/formatter.go:217-219`)
  and `writeView`'s subscribes line (`:325-327`)
- `quoted` (`internal/formatter/formatter.go:47-49`) for every string, never `%q` —
  `tasks/learnings.md` "Never write emod source with `%q`", including its obligation to carry a
  round-trip subtest per hazard character
- Round-trip through the parser is the assertion that catches a mangled declaration, not a golden:
  `internal/formatter/formatter_test.go:427`
- A fmt golden is a pinned canonical constant, never the input fixture handed back —
  `tasks/learnings.md` "`emod fmt` canonicalises order", `internal/cli/fmt_test.go:118`, `:135`,
  `requireFmtSettlesOn` (`:381`)
- Every expected string starts with the `emod <n>` header — `tasks/learnings.md` "Formatter output
  always begins with `emod N`"
- `tasks/learnings.md` "Additive output changes owe a byte-identical receipt for models that do not
  use the feature": the untouched goldens are that receipt, and it belongs in the commit message

**Testable:** Yes — through `formatter.Format` and `cli.RunFmt`.

**Verification:** `go test -tags unit ./internal/formatter/... ./internal/cli/...`;
`go run ./cmd/emod fmt` over a temporary file holding the Task 2 fixture, then again over the result.

**Depends on:** Task 2

---

### Task 6: Carry specs through the JSON and CUE exports and the embedded schema

**Behavior:** Both model exports emit every spec under its slice in declaration order, with its name,
its given history, the command it exercises and which of the two outcomes it states; the bundled
schema declares them; and the diagram document — which is nodes and edges — carries no trace of them,
nor does any rendered diagram or the glossary. A model without specs exports and renders
byte-identically to before.

**Acceptance Criteria:**
- [ ] The JSON export of the Task 2 fixture carries every spec under the slice that declares it, in
      declaration order, with its name, its given event names in order and the command it exercises
- [ ] The JSON export distinguishes the two outcomes: reading a rejection back tells which invariant
      it names, and reading a success outcome back gives the event names in order
- [ ] A spec written with no `given` and a spec written with `given []` export alike
- [ ] The CUE export of the fixture carries the same, and the "CUE and JSON exports describe the same
      model" subtest (`internal/export/export_test.go:3366`) passes for it
- [ ] `internal/cue/schema.cue` declares specs as an optional key on the slice definition, and
      `cue vet -d '#Model'` over the export of the fixture passes
      (`internal/export/export_test.go:3362`)
- [ ] `emod schema` prints a schema that declares specs on the slice definition
- [ ] Walking the whole diagram JSON document produced from the fixture finds no spec name, no given
      or then event reference belonging only to a spec, and no rejected invariant name at any key or
      depth — after first proving the same search finds them in the model document
- [ ] Every diagram rendering of the fixture — drawio, SVG, mermaid and ASCII — is byte-identical to
      the rendering of the same model with its specs stripped, and the comparison opens by asserting
      the two models actually differ and that the twin lost the specs of both slice homes
- [ ] The glossary markdown and JSON renderings of the fixture are identical to those of the same
      model with its specs stripped — a spec is a scenario, not a term
- [ ] Existing subtests in `internal/export/export_test.go`, `internal/diagram` and
      `internal/glossary` pass with no edit to any expected output, so a model without specs exports
      and renders exactly as before

**Affected Files/Modules:**
- `internal/export/export.go` — `jsonSlice` (`:71-86`), the spec document types alongside
  `jsonInvariant` (`:66-70`), their converters alongside `convertInvariants` (`:342`), and
  `cueWriter`'s slice writer alongside `writeInvariant` (`:1259`)
- `internal/cue/schema.cue` — a definition for the spec and its outcomes plus the key on `#Slice`
  (`:73-85`)
- `internal/export/export_test.go` — the two exports, the schema conformance, the diagram-document
  guard (`:2851`) and the helpers around it (`:3635-3650`)
- `internal/diagram/contract_test.go` — the differential across the four renderers, alongside
  `"declaring invariants leaves the picture untouched"` (`:207`)
- `internal/glossary/glossary_test.go` — the receipt that the vocabulary is unchanged

**Patterns to Follow:**
- All three surfaces land together: `tasks/learnings.md` "A new exported field must land in JSON, CUE
  and `schema.cue` in the same change"
- Key order in a `json*` document type comes from its siblings, not from `schema.cue` —
  `tasks/learnings.md` "JSON and CUE order their document keys differently", whose worked example is
  `jsonInvariant` (`internal/export/export.go:66`)
- List emission in CUE: `writeCUEList` and `lineIfSet` as used at
  `internal/export/export.go:1247-1259`
- Keep the diagram document forked: `jsonDiagramEvent` exists precisely so a new AST field cannot
  leak into the node-and-edge contract; the guard that walks the whole document and first proves the
  search works is `internal/export/export_test.go:2851-2858` with `invariantTextAnywhere` (`:3641`)
- A differential must first prove its twin differs, and the strip helper must visit every home or the
  receipt is vacuous: `internal/diagram/contract_test.go:207-227`, `withoutInvariants` (`:692`) —
  which returns a copy rather than stripping in place — and `libraryLendingInvariantNames` (`:679`)
  as the transcribed list a short walk fails against; `tasks/learnings.md` "A differential receipt
  must first prove the twin actually differs"
- Name a strip helper for what it guarantees — `tasks/learnings.md` "Name an extracted helper after
  the contract its callers rely on"
- Do not add a "render it twice" assertion — `tasks/learnings.md` "An assertion whose expected value
  comes from the code under test is the recurring review finding"

**Testable:** Yes — through `export.ExportJSON`, `export.ExportCUE`, `export.ExportDiagramJSON`,
`cli.RunSchema`, the four `diagram.Export*` renderers and `glossary.RenderMarkdown` /
`glossary.RenderJSON`.

**Verification:** `go test -tags unit ./internal/export/... ./internal/diagram/...
./internal/glossary/... ./internal/cli/...`; `go run ./cmd/emod export --format cue <fixture file>`
and `go run ./cmd/emod schema`.

**Depends on:** Task 2

---

### Task 7: Accept spec blocks in the tree-sitter grammar

**Behavior:** The tree-sitter grammar parses a spec block inside a slice, with its `given`, `when` and
both `then` shapes, so a file `emod validate` accepts is not red-squiggled by a tree-sitter-backed
editor. The grammar stays looser than the Go parser: any number of specs, in any position among the
slice's other entries, and any number of entries inside a spec in any order.

**Acceptance Criteria:**
- [ ] A corpus case for a slice declaring two spec blocks parses to the expected tree
- [ ] A corpus case placing a spec block before a `flow` and another after it parses, showing
      position within the slice is free and the count is unbounded
- [ ] A corpus case covers `given` with several events, `given []`, and a spec with no `given` at all
- [ ] A corpus case covers a `then` event list and a `then rejected <name>` outcome
- [ ] A corpus case writing a spec's entries out of canonical order parses
- [ ] A corpus case for a `fields` block declaring fields named `spec`, `given`, `when`, `then` and
      `rejected` still parses them as field lines, not as spec entries
- [ ] `mise exec -- task test:grammar` passes, and running it a second time leaves every tracked file
      under `editors/tree-sitter-emod/` byte-identical
- [ ] No file under `editors/tree-sitter-emod/src/` is tracked — `.gitignore` still ignores it
- [ ] No file under `editors/tree-sitter-emod/queries/` changes, and
      `editors/vscode/syntaxes/emod.tmLanguage.json` is untouched — highlighting is US-017's story

**Affected Files/Modules:**
- `editors/tree-sitter-emod/grammar.js` — the spec rule, its entries, and its place in `_slice_item`
  (`:85-94`)
- `editors/tree-sitter-emod/test/corpus/slice.txt` and `test/corpus/fields.txt` — the new cases

**Patterns to Follow:**
- `buildDescribedBlock($, ...items)` (`editors/tree-sitter-emod/grammar.js:1-5`) is how every block
  body admits unordered, unbounded entries; an `optional(...)` in a block body would be a bug —
  `tasks/learnings.md` "The tree-sitter grammar must never be stricter than the Go parser"
- Bracketed, comma-separated, possibly empty list: `subscribes_block`
  (`editors/tree-sitter-emod/grammar.js:170-176`)
- A one-line keyword-plus-identifier entry: `invariant`
  (`editors/tree-sitter-emod/grammar.js:57-61`)
- Field names must keep matching `any_identifier` (`:217`) — `tasks/learnings.md` "New DSL keywords
  must stay usable as field names", with `editors/tree-sitter-emod/test/corpus/version_header.txt`
  as the existing field-named-after-a-keyword case
- Corpus case layout: `editors/tree-sitter-emod/test/corpus/slice.txt`
- Run through `mise exec --`, and do not un-ignore `src/` — `tasks/learnings.md` "Run repo tooling
  through `mise exec --`" and "Generated tree-sitter `src/` stays gitignored"

**Testable:** Yes — the corpus cases are the tests, run by `task test:grammar`.

**Verification:** `mise exec -- task test:grammar`, run twice, the second run leaving the tracked
files untouched; `git ls-files editors/tree-sitter-emod/src` returning nothing.

**Depends on:** Task 2

---

### Task 8: Document specs in the DSL reference

**Behavior:** The DSL reference teaches the spec: where it may be written, what `given`, `when` and
`then` each state, the two outcomes a command spec can end in, that `given []` and an omitted `given`
are the same empty history, which names must resolve and against what, and which tools carry specs.
A reader learning the language finds it next to the command pattern it describes.

**Acceptance Criteria:**
- [ ] The `slice` skeleton in `docs/dsl-reference.md` §5 (`:255-272`) lists `spec` among the entries a
      slice accepts
- [ ] §6 "Slice Patterns" gains a `### spec` subsection covering the block form, the `given` list,
      `when`, and both `then` outcomes, with an example of each
- [ ] That subsection states that `given []` and omitting `given` are the same empty history, that a
      slice may hold any number of specs in any position, and that the shapes for view, automation
      and translation slices are not part of the language yet
- [ ] It states the resolution rules: every `given` and `then` event and every `when` command must be
      defined somewhere in the model, and a `rejected` name must be declared on the enclosing
      aggregate or, for a slice on a context, that context — with an aggregate and its context being
      separate scopes, linking to the `invariant` subsection (`:147`)
- [ ] §11 "Cross-References" (`:574-592`) lists the spec as a referencing site for events, commands
      and invariants, and its validation bullet list names the two errors this story adds
- [ ] No `## <n>.` heading in `docs/dsl-reference.md` is added, removed or reordered, so every
      existing `(#<n>-<slug>)` link in the document still resolves to its heading
- [ ] The emod snippet in the new subsection, written to a file on its own, passes `emod validate`
      with exit 0

**Affected Files/Modules:**
- `docs/dsl-reference.md` — the slice skeleton in §5 (`:255-272`), a `### spec` subsection in §6
  after "Command Pattern" (`:279-304`), and the table plus bullet list in §11 (`:574-592`)

**Patterns to Follow:**
- Subsection voice, the fenced form-then-example shape and the consumer bullets:
  `docs/dsl-reference.md:147-200` (the `invariant` subsection) and `:279-304` (the command pattern)
- A subsection inside an existing numbered section is safe; a new numbered section renumbers
  everything below it and breaks the in-document links — `tasks/learnings.md`
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
Task 1 lands four of the five keywords, the AST nodes and the whole success-path spec block, which is
the largest single piece of new grammar; Task 2 adds the one remaining outcome and the shared fixture
every later task asserts against, so the two together close the syntax half of the story. Tasks 3 and
4 deliver the story's two validation criteria and are independent of each other — Task 3 reads the
model-wide name index that already exists, Task 4 needs a scope-aware walk. Task 5 comes next among
the fan-out tasks because a formatter that does not know a construct silently deletes it, which is
the most damaging gap the story could leave open. Tasks 6 and 7 close the two surfaces this repo
requires of any new construct — the exports plus the schema, and a grammar that is never stricter
than the parser. Tasks 3-7 all depend only on Task 2 and can run alongside one another. Task 8
documents the finished surface.

**Coverage of the story's acceptance criteria:**

| Criterion | Task |
|---|---|
| A slice accepts any number of `spec "<name>" { ... }` blocks | 1 |
| `given` takes an ordered list of event names; `given []` and omitting `given` are equivalent | 1 |
| `when` names the command under test | 1 |
| `then [EventA, EventB]` states the events appended on success | 1 |
| `then rejected <invariantName>` states the command fails | 2 (syntax), 4 (name resolves to an invariant on the enclosing scope; unresolved is an error with location) |
| Every event in `given` and `then`, and the command in `when`, must be defined in the model | 3 |
| Every existing `.emod` file stays valid with unchanged meaning (story-wide constraint) | 1 (keywords stay usable as field names), 5 (formatter goldens), 6 (export, diagram and glossary receipts) |

Tasks 5, 6, 7 and 8 carry no story criterion of their own — they are the fan-out this repo requires
of any new construct: `emod fmt` must not drop it, the JSON/CUE/schema trio moves together, the
tree-sitter grammar must not reject what `emod validate` accepts, and the reference must teach it.

Nothing from the story is deferred. What US-006 deliberately leaves to later stories: the view,
automation and translation spec shapes and the rule that a `then` shape must match the slice pattern
(US-007); the four `spec/*` lint rules, including the boundary rule that is what the feature
ultimately pays for (US-008); flow rejection edges (US-009); example payload literals on element
references (US-010); `given` / `when` / `then` alignment and the wider canonical attribute order in
`emod fmt` (US-014); LSP hover, completion and navigation over specs (US-015); keyword highlighting
in the VS Code grammar and the tree-sitter queries (US-017); and specs in `examples/*.emod` (US-018).

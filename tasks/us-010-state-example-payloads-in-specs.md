# US-010: State example payloads in specs

## Progress
- [x] Task 1: Lex number literals and record example payloads on spec references
- [ ] Task 2: Share a payload-carrying fixture and its twin
- [ ] Task 3: Reject payload field names the referenced construct does not declare
- [ ] Task 4: Check payload literal kinds against the declared field type
- [ ] Task 5: Preserve payloads through `emod fmt`
- [ ] Task 6: Carry payloads through the JSON and CUE exports and the embedded schema
- [ ] Task 7: Accept payloads in the tree-sitter grammar
- [ ] Task 8: Document payloads in the DSL reference

---

## Story Reference

`user-stories/specs-and-metadata.md` → **US-010: State example payloads in specs** (tenth story of
"Specs, Invariants, and Model Metadata", lines 134-145). Design notes:
`docs/proposals/specs-and-metadata-proposal.md` §4 "Payload literals" (lines 205-233), the validation
list at lines 241-243, the AST sketch at lines 344-356, the lexer note at line 392, the parser note at
line 402, and Phase 3 at lines 597-599. This story depends only on US-006, which is delivered on
main; US-007 through US-009 are not, and nothing here waits on them.

**In scope:** an optional `{ field: value, ... }` block after any event or command reference inside a
spec — every element of a `given` list, the `when` reference, and every element of a `then` event
list. Three literal forms: quoted strings, numbers with an optional fractional part, and `true` /
`false`. Payloads are partial, so a field declared `required` may be omitted, and a names-only
reference stays valid everywhere. Two validation rules: a payload field name must be declared on the
referenced construct's `fields`, and the literal's kind must match the declared field type — strings
satisfy `string`, and `date`, `timestamp` and `uuid` when the value parses as that format; numbers
satisfy `decimal`, and `int` when the literal has no fractional part; `true` / `false` satisfy `bool`;
every other declared type is a domain type and accepts any literal unchecked. Carried with them,
because the repo's writers would otherwise silently drop the new construct: `emod fmt` round-trip, the
JSON and CUE exports plus the embedded schema, the tree-sitter grammar (which must never reject what
`emod validate` accepts), and the DSL reference.

**Out of scope:** value-aware boundary checking in DCB mode, where a `given` payload's tagged value is
compared against the `when` payload's (US-011); payload formatting and alignment rules — one field per
line with values aligned when the payload does not fit, and the wider canonical attribute order
(US-014), so the formatter here emits one canonical single-line form and leaves column alignment
alone; payload field-name completion in the LSP, and hover or navigation over a payload (US-015);
rendering payloads on diagrams, which the `--specs` flag would show (US-016); highlighting numbers and
`true` / `false` as literals in the VS Code grammar and the tree-sitter highlight queries (US-017);
payloads in `examples/*.emod` and the wider example and reference coverage sweep (US-018). Out of
scope per the story file's Non-Goals: expected view-state payloads on `then view` outcomes, and
variable binding (`let`) — payload linkage is by repetition of the same value, and nothing links two
values mechanically. Also deliberately untouched: the field-type list `internal/lsp/completer.go:247`
offers inside a `fields` block, which names four types where this story teaches the validator seven
(US-015 owns the LSP surface), and the four `spec/*` lint rules (US-008), so this story adds no lint
rule and no severity configuration.

**Open questions, decided.** Ten shapes the story does not name, each decided so the surrounding
stories stay additive:

1. *Payloads hang off `ast.SpecElement`* — the node `given`, `when` and a `then` event list are all
   made of. `then rejected <invariantName>` names an invariant rather than an event or command and
   therefore takes no payload. This settles the US-007 interaction the story leaves open: when US-007
   adds `then command <CommandName>`, building that outcome on `SpecElement` gives it payloads with no
   further work, which is the reading "any event or command reference" asks for; `then view <ViewName>`
   must *not* be built on `SpecElement`, because expected view-state payloads are an explicit Non-Goal
   of the story file. US-007 owns that choice; this story only ensures the payload rides on the
   element rather than on the clause.
2. *Number literals are unsigned.* A leading `-` is the one literal shape that collides with the `->`
   arrow token, the story's table names only `42` and `12.50`, and admitting a sign later is purely
   additive — a file that does not use one is unaffected. A payload value written `-5` is rejected.
3. *`true` and `false` do not become keywords.* They are recognized by position, the way the proposal
   asks (`docs/proposals/specs-and-metadata-proposal.md:392`), so this story adds no entry to the
   lexer's `keywords` map and `lexer.Keywords()` is unchanged. That keeps every keyword-coverage guard
   — `internal/lexer/tokenizer_test.go:14`, `internal/parser/parser_test.go:225` and `:243`,
   `internal/oracle/oracle_test.go`, `internal/lsp/keywords_test.go` and
   `editors/tree-sitter-emod/test/queries/keywords_test.go` — out of the story entirely, and it is why
   no task here carries the usual "the new keyword is usable as a field name" obligation. The seven
   field type names the validator learns to check are likewise not keywords.
4. *The comma between payload entries is optional*, the latitude every bracketed list in this parser
   already gives (`parseIdentifiersUntil`, `internal/parser/parser.go:761`, advances over a comma when
   it finds one and does not require it). US-014 may put one field per line, and a parser that
   required commas would make that a breaking change rather than a formatting choice.
5. *A payload's opening brace must sit on the line of the reference it qualifies*; its entries and its
   closing brace may span lines. Without that gate a `{` opening the next construct is read as a
   payload for the reference above it — the failure `tasks/learnings.md` records under "A line-oriented
   declaration must gate every optional trailing token on the first token's line".
6. *An empty payload `{}` is accepted and means no payload*, mirroring `given []` and an omitted
   `given`, which US-006 already made equivalent. The formatter writes no braces for it, so the two
   spellings format to identical text.
7. *A field name written twice in one payload keeps both entries, in order, and is not an error here.*
   Every block body in this repo is an unbounded loop with no arity rule, the tree-sitter grammar must
   not be stricter, and a lint rule is the natural home for the advice — US-008's territory, not this
   story's.
8. *A payload on a reference the model does not declare produces no payload diagnostic* — only the
   existing "event %q does not exist" / "command %q does not exist" from US-006. There is nothing to
   check the field names against, and reporting both turns one typo into a cascade.
9. *The lexer's existing `Integer` kind stays what the version header consumes*, and the fractional
   form is recognized alongside it rather than by widening `Integer` into one number kind. The version
   header is the only construct in the language that reads a numeric token (`internal/parser/parser.go:97`
   and `:123`), and keeping its token narrow is what lets `emod 1` behave exactly as it does today
   while `emod 1.5` reports the header's own message. It also means the three `lexer.Integer`
   assertions in `internal/lexer/tokenizer_test.go:78-107` keep their meaning, and the "no fractional
   part" half of the `int` check falls out of which token the scanner produced.
10. *Exported payload values carry their natural JSON and CUE type* — a string as a string, a number as
   a number, `true` as a boolean — rather than a source-text string beside a kind tag. That is what a
   schema-registry or code-generation consumer of `emod export` expects, and the declared field type on
   the construct already tells such a consumer whether a number is an `int` or a `decimal`. The one
   loss is trailing zeros: `12.50` reads back from the export as `12.5`. The formatter reads the AST,
   not the export, so `emod fmt` still writes `12.50` back verbatim.

**Overarching constraint:** every existing `.emod` file stays valid with unchanged meaning. That is
load-bearing in four places here — the lexer's new numeric token must leave the `emod <n>` version
header behaving as it does today, `emod fmt` must still produce its current bytes for a model whose
specs carry no payload, no existing golden in `internal/formatter`, `internal/diagram`,
`internal/glossary` or `internal/cli` may need editing, and a spec written names-only must validate
exactly as before.

**Learnings folded in** from `tasks/learnings.md`: a bracketed list's terminator set decides whether
one typo cascades — and a payload's closing brace is a `lexer.CloseBrace`, which the spec list's
current terminator set treats as the end of the list; a line-oriented declaration must gate every
optional trailing token on the first token's line; a quoted or identifier block entry is one `case` on
the `parse*EntryInto` family; parser diagnostics at a stored position go through `errorAtPosition`; a
drain that eats the `}` is caught by reading back `ClosePos.Line`, not by a diagnostic count; put a new
parser subtest in the group that owns the construct; `ast.ThenClause` dispatches through five type
switches, none of which errors, so a second sealed interface repeats that trap; a new optional field
ships a six-part fixture kit and `editedCopies` is shallow, so an edit reaching inside a slice must
nest; `require.NotEqual` on a stripped twin is satisfiable without stripping anything; a new shared
fixture owes `internal/oracle` a zero-diagnostic subtest, and a spec is not a reference so every
spec-carrying fixture still needs its flows; diagnostics gathered from more than one AST collection
must be position-sorted; `RuleName` marks a diagnostic `emod lint --explain` can describe, so a hard
error carries none; a second `require.Contains` on one message is often shadowed by the first, and a
rule whose message branches is pinned by whole formatted lines; CLI diagnostic tests must assert the
distinguishing message text; an assertion whose expected value comes from the code under test cannot
fail; never write emod source with `%q`; `emod fmt` canonicalises order so a fmt golden is never the
input re-indented, and formatter output always begins with `emod N`; `emod fmt <file>` writes in place,
so a receipt run dirties the tree; additive output changes owe a byte-identical receipt; a new exported
field lands in JSON, CUE and `schema.cue` together, JSON and CUE order their keys differently, JSON key
order is assertable with `emittedKeyOrder`, and a key rename owes a retired-key negative assertion; a
"no expected constant moves" criterion is unsatisfiable when the task changes a shape an expected
constant transcribes; the tree-sitter grammar must never be stricter than the Go parser, every
`grammar.js` rule carries a one-line example of its full shape, generated `src/` stays gitignored and
repo tooling runs through `mise exec --`; `docs/dsl-reference.md` anchors embed the section number and
its `###` sub-heading anchors are cited more often; an ```emod fence is a promise that the block
validates; acceptance criteria never reference commit, branch or remote state.

---

## Codebase Context

**Lexer.** `internal/lexer/token.go` holds one `keywords` map (`:73-112`, thirty-eight spellings) and
derives `Keywords()` and `Kind.IsKeyword()` from it; this story adds nothing to it. It already declares
an `Integer` kind (`:54`) whose only producer is `readInteger` (`internal/lexer/tokenizer.go:168-177`,
digits only, no sign and no fractional part) and whose only consumers are the version header's two
sites, `parseVersionHeader` (`internal/parser/parser.go:97`) and `reportMisplacedVersionHeader`
(`:123`). `Kind.String()` (`token.go:140-179`) spells every non-keyword kind for diagnostics. The
scanner's dispatch (`tokenizer.go:28-84`) tests `-` followed by `>` for the arrow *before* it reaches
the digit case, and reports `unrecognized character: %c` for anything it cannot classify — today
`emod 1.5` produces two diagnostics, `unrecognized character: .` then
`unrecognized keyword "5"; expected one of: actor, context, model` (verified).

**AST.** `internal/ast/ast.go` has `Spec` (`:95-104`), `SpecElement` (`:106-109`, a bare name and
position today), and the sealed `ThenClause` (`:111-126`) with `ThenEvents` and `ThenRejected`.
`Field` (`:156-163`) carries `Name`, `Type` and `Modifier`, each with a position; the type is any
identifier and nothing in the repo interprets it. The proposal's payload sketch is at
`docs/proposals/specs-and-metadata-proposal.md:344-356`.

**Parser.** `internal/parser/parser.go` parses a spec in `parseSpec` (`:431-484`), an unbounded entry
loop over `given` / `when` / `then`. Three call sites build `SpecElement`s: `parseSpecCommand`
(`:486-496`) for `when`, and `parseSpecEventList` (`:522-545`) for both `given` and a `then` list,
which delegates to the shared `parseIdentifiersUntil` (`:761-777`) with the terminator predicate
`atSpecEventListEnd` (`:547-549`) — stopping at `]`, `}` and the sibling entry keywords. That shared
helper also serves `parseIdentifierList` (`:731-759`, backing `events` and `subscribes`), whose looser
terminator set `tasks/learnings.md` records as the cascade case, so the spec list wants its own element
loop rather than a widened shared helper. `parseSpecOutcome` (`:498-520`) is the `then` disambiguation.
Two existing shapes cover what a payload entry needs: `parseTagEntry` (`:1307-1333`) is the
identifier-like key, `:` , value triple, and `parseField` (`:1248-1275`) is the line-gated declaration
with an optional trailing part. `checkIdentifierLike` (`:1506-1512`) is what keeps keywords usable in
name position; `checkSameLineAs` (`:1502`) and `skipRestOfLineOrBlockEnd` (`:1526-1530`, which also
halts at `}`) are the gate and the drain; `errorAt` (`:1549`) and `errorAtPosition` (`:1553`) are the
two diagnostic constructors, the second for a position captured earlier rather than the current token.

**Parser tests.** `internal/parser/parser_test.go` is one umbrella split into groups named for the area
under test; `"contexts, aggregates and slices"` owns the slice and the spec, `"error reporting"` owns
recovery and message shapes, and `"version header"` holds the keyword-as-field-name subtests that
iterate `lexer.Keywords()`.

**Validator.** `internal/validator/validator.go` builds a `modelIndex` (`:43-81`) whose
`collect`/`collectEventShape` (`:83-131`) record command and event *names* and, for events only, a
list of field names (`eventFields`, `:53`) and tag keys — no field *types* and no command fields at
all, so the payload checks extend the index. `declaresCommandOrEvent` (`:136-138`) backs `when`, which
resolves against commands *and* events, while `undeclaredSpecEvents` (`:359-368`) checks events only:
`tasks/learnings.md` records that the two lookups are deliberately different. `specDiagnostics`
(`:341-357`) is the spec reference check and already sorts with `sortInDeclarationOrder` (`:193-201`)
because a spec's parts live in three AST fields. `errorAt` (`:150-158`) sets `Error` severity and no
`RuleName`. `scheduleExpressionDiagnostics` (`:370-382`) with `isWellFormedSchedule` is the existing
"this string must parse as a format" check, and the package already imports `time` and `regexp`.

**Validator tests.** `internal/validator/validator_test.go` groups by rule and its newer groups assert
whole formatted diagnostic lines with `require.Equal` over `reportedLines(diags)` rather than layering
`require.Contains` calls.

**Formatter.** `internal/formatter/formatter.go` `writeSlice` (`:141-206`) emits specs last;
`writeSpec` (`:372-386`) writes `given`, `when` and `then` in that fixed order whatever the source
order was, `bracketed` (`:65-67`) joins names with `", "`, `specElementNames` (`:401`) flattens
elements to names, and `quoted` (`:57-63`) is the only correct way to emit a string because the
language has no escape sequences. `internal/formatter/formatter_test.go` holds the round-trip group
(the spec leaf at `:876`, the quoting-hazard leaf at `:953`) and the canonical-order group `"specs"`
(`:3597`). `internal/cli/fmt_test.go` pins `specFormattedEmod` (`:244`) and feeds canonical constants
to `requireFmtSettlesOn`.

**Exports.** `internal/export` is three files, not the `export.go` older learnings name: `json.go`,
`cue.go`, `diagram.go`. `jsonSpec` (`json.go:189-199`) carries `given` as `[]string`, `when` as a
`string` beside a sibling `when_position`, and `then` as `jsonSpecOutcome` (`:201-205`) whose `events`
is likewise `[]string`; `convertSpec` (`:420-435`) and the three `specElement*` helpers (`:450-473`)
build them. `cue.go` mirrors it in `writeSpec` (`:134-142`) and `writeSpecOutcome` (`:144-151`), and
emits no positions at all. `internal/cue/schema.cue` declares `#SpecOutcome` (`:74-77`) and `#Spec`
(`:79-85`). Three subtests couple the surfaces: `internal/export/export_test.go:3812` runs
`cue vet -d '#Model'` over the export of the spec fixture, `:3841` requires the two formats to agree on
specs, and `:1463` reads the JSON back against the hand-transcribed `libraryLendingSpecs`
(`:4306-4340`), which spells the element as a bare name. `internal/cue/embed_test.go`'s `fullModelJSON`
(`:232-242`) does too, and that file's negative leaves (`:112`, `:123`, `:137`) are the shape for
proving a retired key is rejected. The diagram document is deliberately forked and carries nothing from
a spec, guarded at `export_test.go:3179`; `emittedKeyOrder` (`:4760`) reads JSON key order out of the
raw bytes, and `objectsUnder` / `statedUnder` / `exportedSlices` / `listsKeyedBy` (`:4646-4740`) are
the read-back walkers.

**Diagrams and glossary.** Neither renders a spec. `internal/diagram/contract_test.go:441`
`"stating specs leaves the picture untouched"` and `internal/glossary/glossary_test.go:586` are the
differentials to copy — both open by proving the twin actually lost the feature and that the featured
model states it, against a transcribed list.

**Fixtures.** `internal/test/fixtures.go` holds eight models, `SpecLibraryLending` (`:416-579`) being
the spec-carrying one, with its transcription `SpecLibraryLendingSpecNames` (`:1119`), its twin
`WithoutSpecs` (`:1216`) built on `copyWithEditedSlices` (`:1257`) and the generic `editedCopies`
(`:1275`), and its getter `DeclaredSpecNames` (`:1299`). `internal/test/models.go` holds one parsing
accessor per fixture (`:37` for specs). `internal/oracle/oracle_test.go:60` is the zero-diagnostic leaf
that proves a fixture is something `emod validate` and `emod lint` both accept.

**Tree-sitter.** `editors/tree-sitter-emod/grammar.js` declares `spec_definition` (`:107-113`),
`spec_given` (`:115`), `spec_when` (`:120`), `spec_then` (`:125`) and `spec_event_list` (`:133-140`),
all built from `any_identifier` (`:339`). `string` (`:330`) is the only literal rule; the version
header's digits are an aliased `token.immediate` (`:29`). `test/corpus/specs.txt` holds five cases.
`queries/folds.scm` and `queries/indents.scm` list neither `spec_definition` nor any spec entry today,
so a payload's braces add nothing there. `src/` is gitignored and `task test:grammar` regenerates
before running.

**Not touched, deliberately.** `internal/linter` (US-008 owns every `spec/*` rule);
`internal/importer` and `internal/wasm/pipeline.go`, which move the diagram document that carries no
spec; `internal/viewer`; `internal/cli/slices.go`, whose `detectPattern` classifies a slice by the
elements it declares; `internal/lsp` in full, including the field-type list at `completer.go:247`
(US-015); `editors/vscode/syntaxes/emod.tmLanguage.json` and
`editors/tree-sitter-emod/queries/highlights.scm` (US-017); `examples/*.emod` (US-018); `e2e/`.

---

## Tasks

### Task 1: Lex number literals and record example payloads on spec references

**Behavior:** The lexer learns a number literal — digits with an optional fractional part — as a single
token. A payload block may follow any event or command reference inside a spec: every element of a
`given` list, the `when` reference, and every element of a `then` event list. A payload holds
field-name / literal pairs separated by an optional comma, where a literal is a quoted string, a
number, or `true` / `false` recognized by position rather than as keywords. The opening brace sits on
the reference's line; the entries and the closing brace may span lines. An empty payload means no
payload, a payload field may be named after a DSL keyword, and each field records the name, the
literal, the literal's kind and a position for both halves. `then rejected <invariantName>` takes no
payload. A malformed payload reports once and leaves the enclosing blocks closing.

**Acceptance Criteria:**
- [x] A spec whose `given` list states `[RoomReserved { roomId: "101", nights: 3, vip: true }]` parses
      with no diagnostics, and the element records three payload fields in declaration order, each with
      its name, its literal, the literal's kind, and a position for the name and for the value
- [x] A payload's closing brace does not end the list that contains it: `given [A { x: "1" }, B]`
      records two elements, the first carrying a payload and the second none
- [x] A payload on the `when` reference, and one on an element of a `then` event list, parse and are
      recorded the same way
- [x] A payload written across several lines, and the same payload written on one line, and the same
      payload written with no commas between entries, all yield the same fields in the same order —
      equal in every respect other than the positions they were read from
- [x] A payload written `{}` yields a spec equal in every respect other than positions to the same spec
      written with no payload at all
- [x] A payload field named `type`, `description` or `given` parses as a field name
- [x] A field name written twice in one payload records both entries, in order, with no diagnostic
- [x] `42` and `12.50` both parse as number literals and both keep their source text, so `12.50` is not
      reduced to `12.5`
- [x] `true` and `false` parse as boolean literals, and `lexer.Keywords()` returns exactly the
      spellings it returns today — the keyword-coverage subtests in `internal/lexer/tokenizer_test.go`,
      `internal/parser/parser_test.go`, `internal/oracle/oracle_test.go`, `internal/lsp/keywords_test.go`
      and `editors/tree-sitter-emod/test/queries/keywords_test.go` need no edit
- [x] An identifier that is neither `true` nor `false` in value position reports exactly one diagnostic
      (`require.Len(t, diags, 1)`) naming the payload field, and the spec entry on the following line is
      still parsed onto the spec
- [x] A payload entry with no `:` reports exactly one diagnostic, and the spec's later entries still
      parse onto the spec
- [x] A payload value written `-5` reports a diagnostic — signed numbers are not part of the literal
      grammar — and the enclosing spec block still closes
- [ ] A payload whose brace is never closed reports exactly one parser diagnostic, and the spec, the
      slice and the context that enclose it all report a non-zero `ClosePos.Line`
      — met where the payload is bounded by the list's `]` or by a sibling spec entry, and *not* met
      where the payload is the spec's last entry: its closing brace is the same token as the spec's,
      so the payload takes it, exactly as every other block in this parser does with an unclosed
      brace. Bounded to two diagnostics instead of one per token of the construct below.
- [x] A `{` written on the line *after* a spec element is not read as that element's payload
- [x] `emod 1` still parses as the version header with no diagnostic, and `emod 1.5` reports exactly
      one diagnostic, naming the version header and quoting `1.5`, in place of today's two —
      `unrecognized character: .` followed by `unrecognized keyword "5"`
- [x] The three subtests in `internal/lexer/tokenizer_test.go:78-107` that read a version header's
      value back as `lexer.Integer` still pass unedited, so the header's token stays as narrow as it is
      today
- [x] A spec that states no payload parses exactly as before: `oracle.Check` over every fixture in
      `internal/test/fixtures.go` still returns no diagnostics, and no existing subtest in
      `internal/parser/parser_test.go` or `internal/lexer/tokenizer_test.go` needs editing

**Affected Files/Modules:**
- `internal/lexer/token.go` — the number kind and its spelling in `Kind.String()` (`:140-179`)
- `internal/lexer/tokenizer.go` — the scanner's digit case (`:77-80`) and `readInteger` (`:168-177`)
- `internal/ast/ast.go` — the payload field node and the literal it carries, hung off `SpecElement`
  (`:106-109`)
- `internal/parser/parser.go` — the payload parser; the spec list's element loop replacing the shared
  `parseIdentifiersUntil` call in `parseSpecEventList` (`:522-545`); `parseSpecCommand` (`:486-496`).
  The version header's two `lexer.Integer` sites (`:97`, `:123`) stay as they are — "Open questions,
  decided" 9 keeps the header's token narrow, and the improved `emod 1.5` message comes from the
  scanner producing one token where it produced two, not from editing either site
- `internal/lexer/tokenizer_test.go`, `internal/parser/parser_test.go` — subtests in the group that
  owns the construct

**Patterns to Follow:**
- The list whose terminator set is the whole recovery story: `atSpecEventListEnd`
  (`internal/parser/parser.go:547-549`) and `parseIdentifiersUntil` (`:761-777`) — and
  `tasks/learnings.md` "A bracketed list's terminator set decides whether one typo cascades", noting
  that the shared helper also serves `parseIdentifierList` (`:731-759`) and that a payload's closing
  brace is a `lexer.CloseBrace`, which the predicate treats as the list's end today
- The identifier-like key, `:`, value triple: `parseTagEntry` (`internal/parser/parser.go:1307-1333`)
- The line-gated declaration with an optional trailing part: `parseField`
  (`internal/parser/parser.go:1248-1275`), `checkSameLineAs` (`:1502`),
  `checkIdentifierLikeSameLineAs` (`:1514`) — `tasks/learnings.md` "A line-oriented declaration must
  gate every optional trailing token on the first token's line"
- Report once and drain: `skipRestOfLineOrBlockEnd` (`internal/parser/parser.go:1526-1530`) and the
  `parse*EntryInto` family (`:1419`, `:1432`) — `tasks/learnings.md` "A new block entry keyword owes
  three things to the parser's diagnostics" and "A quoted or identifier block entry is one `case` on
  the `parse*EntryInto` family"
- Proving the drain stopped in the right place by reading back `ClosePos.Line` at three levels rather
  than by a diagnostic count alone — `tasks/learnings.md` "Retiring a keyword needs its own parser arm"
- A diagnostic anchored at a position captured earlier: `errorAtPosition`
  (`internal/parser/parser.go:1553`) — `tasks/learnings.md` "Parser diagnostics at a stored
  `ast.Position` go through `p.errorAtPosition`"
- Keywords stay usable in name position through `checkIdentifierLike`
  (`internal/parser/parser.go:1506-1512`), which asks `IsKeyword()` rather than comparing ordinals
- Node shape: `docs/proposals/specs-and-metadata-proposal.md:344-356`, and `ast.Invariant`
  (`internal/ast/ast.go:68-74`) for the name-plus-position convention. The literal is read by the
  formatter, both exporters and two validator checks, and `tasks/learnings.md` "`ast.ThenClause`
  dispatches through five type switches, none of which errors" records what a second sealed interface
  costs — prefer a representation whose unheard-of case is a compile error or a single dispatch point
- Payloads hang off `ast.SpecElement`, not off `ThenClause`, so US-007's `then command <Name>` inherits
  them by being built on the same node and `then view <Name>` does not — see "Open questions, decided"
- Subtests belong to the group that owns the construct — `tasks/learnings.md` "Put a new parser subtest
  in the group that owns the construct"

**Testable:** Yes — through `lexer.Scan`, `parser.Parse` and `oracle.Check`, all exported.

**Verification:** `go test -tags unit ./internal/lexer/... ./internal/parser/... ./internal/oracle/...`;
`go build ./...`; `go run ./cmd/emod validate` over a temporary file whose spec states a payload,
expecting exit 0 once Tasks 3 and 4 are absent or the payload is well formed.

**Depends on:** None

---

### Task 2: Share a payload-carrying fixture and its twin

**Behavior:** `internal/test` gains a model stating example payloads in both homes a slice has, on
`given`, `when` and `then` references, over fields of every type the story checks and one domain type,
so every writer and renderer downstream asserts against one model. It arrives with the kit the repo
expects of a new optional field: a parsed-model accessor, a hand-transcribed list of the payloads it
states, a twin that clears them, a getter that reads them back, and a zero-diagnostic `oracle.Check`
leaf.

**Acceptance Criteria:**
- [ ] `internal/test/fixtures.go` gains a source declaring payloads inside a slice nested in an
      aggregate *and* inside a slice declared directly on a `mode dcb` context, and
      `internal/test/models.go` gains its parsed-model helper alongside `SpecLibraryLendingModel`
      (`:37`)
- [ ] Across its specs the fixture exercises: a payload on a `given` element, one on `when`, one on a
      `then` event-list element, a names-only element beside a payload-carrying one *in the same list*,
      a spec that states no payload at all, and a `then rejected` spec whose `given` and `when` carry
      payloads
- [ ] Its payload values cover a field declared `string`, one `date`, one `timestamp`, one `uuid`, one
      `int`, one `decimal`, one `bool` and one of a domain type, and every value parses as the format
      its field declares
- [ ] One payload-carrying element sits ahead of a further element in its list, and one payload-carrying
      spec sits ahead of a further slice entry rather than last — an entry that runs on into what
      follows it is only caught when something follows it
- [ ] No payload value in the fixture equals a construct name, a field name or a bare small integer, so
      a whole-document text search for one cannot match a position, an id or another construct
- [ ] `oracle.Check` over the fixture returns no diagnostics at all, and `internal/oracle/oracle_test.go`
      `"clean input"` carries that subtest
- [ ] A hand-written transcription in `internal/test/fixtures.go` restates every payload the fixture
      states — filed under the reference that states it, in declaration order across both slice homes —
      and a getter reads the same back off a parsed model
- [ ] A twin helper returns a copy with every payload cleared and every spec otherwise intact: the
      getter over the twin is empty, the getter over the fixture equals the transcription, and the model
      handed to the twin still states its payloads afterwards
- [ ] `SpecLibraryLending`, `HotelReservation`, `DescribedHotelReservation`, `KeywordFieldSearchCatalog`,
      `InvariantLibraryLending`, `AutomationReadsLibraryLending`, `TriggerReadsLibraryLending` and
      `AutomationScheduleLibraryLending` are unchanged, so every existing golden keeps witnessing a
      model whose specs carry no payload

**Affected Files/Modules:**
- `internal/test/fixtures.go` — the source, its transcription, its twin and its getter, beside
  `SpecLibraryLending` (`:416-579`), `SpecLibraryLendingSpecNames` (`:1119`), `WithoutSpecs` (`:1216`)
  and `DeclaredSpecNames` (`:1299`)
- `internal/test/models.go` — the parsed-model accessor
- `internal/oracle/oracle_test.go` — the zero-diagnostic leaf for the new fixture

**Patterns to Follow:**
- The kit and its parts: `tasks/learnings.md` "A new optional field ships a six-part fixture kit, not a
  bespoke model per package", with `SpecLibraryLending` and `AutomationReadsLibraryLending` as the two
  worked examples; naming follows the `…LibraryLending` family
- The twin is a copy built on `copyWithEditedSlices` (`internal/test/fixtures.go:1257`) and
  `editedCopies` (`:1275`), which are **shallow** and leave a nil list nil on purpose — a payload lives
  inside a slice's specs' elements, so the edit has to nest or it writes through to the model the caller
  passed and every downstream differential compares a model with itself
- Prove the twin differs by content, not by `require.NotEqual` alone — `tasks/learnings.md`
  "`require.NotEqual` on a stripped twin is satisfiable without stripping anything"
- A spec is not a reference, so every slice still needs its `flow`, and a `mode dcb` context needs
  tagged events and a `decides_on` reaching them — `tasks/learnings.md` "A spec is not a reference" and
  "A new shared fixture owes `internal/oracle` a zero-diagnostic subtest"
- Exercise an omitted optional part mid-block, never as the last entry — `tasks/learnings.md`
- Fixture prose style and comment header stating what each placement is for:
  `internal/test/fixtures.go:416-422`

**Testable:** Yes — through `oracle.Check`, `parser.Parse` and the getter over the parsed fixture.

**Verification:** `go test -tags unit ./internal/...`; `go run ./cmd/emod validate` over a temporary
file holding the new fixture, expecting exit 0.

**Depends on:** Task 1

---

### Task 3: Reject payload field names the referenced construct does not declare

**Behavior:** `emod validate` reports an error for every payload field name the referenced command or
event does not declare on its `fields`. Each is reported at the field name's own position and names
both the field and the construct it was looked up on, so an author fixing a typo is pointed at the word
they mistyped. A `when` reference resolves its fields against a command *or* an event, the same
asymmetry US-006 established for the reference itself; a payload on a reference the model does not
declare produces no payload diagnostic.

**Acceptance Criteria:**
- [ ] A payload on a `given` element naming a field the referenced event does not declare produces
      exactly one diagnostic, at `Error` severity, positioned on the field name, whose whole formatted
      line names the field and the event
- [ ] A payload on `when` naming a field the referenced command does not declare produces the equivalent
      diagnostic, naming the command
- [ ] A payload on a `then` event-list element produces the equivalent diagnostic
- [ ] A `when` that names an event rather than a command resolves its payload fields against that
      event's fields
- [ ] A payload field declared on the referenced construct produces no diagnostic, including when the
      field is declared `required` and every other field is omitted — a payload is partial by design
- [ ] A payload on a reference no construct in the model declares produces only the existing
      "event %q does not exist" or "command %q does not exist" diagnostic, and no payload diagnostic
- [ ] A payload on a construct that declares no `fields` block at all reports one diagnostic per payload
      field, the same way
- [ ] A payload field named after a DSL keyword resolves against a field of that name, and a keyword-named
      field the construct does not declare is reported like any other
- [ ] A spec whose payloads name several undeclared fields produces one diagnostic per field, in
      declaration order, identical across repeated runs of `validator.Validate` over the same model
- [ ] These diagnostics carry no `RuleName`, so `emod lint --explain` gains nothing to answer for and no
      lint rule is added by this task
- [ ] `cli.RunValidate` over a file whose payload names an undeclared field exits with `ExitCode` 1, and
      the reported message names the field and the construct rather than only the path and line number
- [ ] `oracle.Check` over the Task 2 fixture, and over every fixture that states no payload, still
      returns no diagnostics

**Affected Files/Modules:**
- `internal/validator/validator.go` — the payload field check alongside `specDiagnostics` (`:341-357`),
  and `modelIndex` / `collect` / `collectEventShape` (`:43-131`), which today record event field *names*
  only and no command fields at all
- `internal/validator/validator_test.go` — a group for payload field names
- `internal/cli/validate_test.go` — the diagnostic as the user receives it

**Patterns to Follow:**
- "X %q does not exist" at the reference's own position: `specDiagnostics`
  (`internal/validator/validator.go:341-357`), `undeclaredSpecEvents` (`:359-368`),
  `appendUndeclaredRef` (`:325-331`) and `errorAt` (`:150-158`), which sets `Error` and leaves
  `RuleName` empty
- The two deliberately different lookups: `declaresCommandOrEvent` (`:136-138`) for `when` against
  `index.eventNames` for `given` and `then` — `tasks/learnings.md` "A spec's `when` resolves against
  commands *and* events; `given`/`then` against events only"
- Position-sorting a check that gathers from more than one AST collection: `sortInDeclarationOrder`
  (`internal/validator/validator.go:193-201`) — `tasks/learnings.md` "Diagnostics gathered from more
  than one AST collection must be position-sorted", and note `collectEventShape` (`:124-131`) *appends*
  per name, so two constructs sharing a name accumulate rather than replace
- No `RuleName` on a diagnostic no configuration can silence — `tasks/learnings.md` "`RuleName` marks a
  diagnostic `emod lint --explain` can describe"
- Assert whole formatted lines over `reportedLines(diags)` rather than layering `Contains` calls —
  `tasks/learnings.md` "A second `require.Contains` on one message is often shadowed by the first" and
  "A rule whose message branches on model state is pinned by whole formatted lines"
- Assert the tokens that identify *this* diagnostic at the CLI layer — `tasks/learnings.md` "CLI
  diagnostic tests must assert the distinguishing message text"

**Testable:** Yes — through `validator.Validate`, `oracle.Check` and `cli.RunValidate`.

**Verification:** `go test -tags unit ./internal/validator/... ./internal/cli/... ./internal/oracle/...`;
`go run ./cmd/emod validate` over a temporary file whose payload misspells a field name, expecting exit
1 and the misspelled name in the message.

**Depends on:** Task 2

---

### Task 4: Check payload literal kinds against the declared field type

**Behavior:** `emod validate` checks each payload literal against the type its field declares. A string
literal satisfies `string`, and `date`, `timestamp` and `uuid` when the value parses as that format —
`YYYY-MM-DD` for a date, RFC 3339 for a timestamp, the canonical 36-character 8-4-4-4-12 hexadecimal
form in either case for a uuid. A number satisfies `decimal`, and `int` when the literal states no
fractional part. `true` and `false` satisfy `bool`. Every other declared type is a domain type and
accepts any literal unchecked — domain types are opaque to the model. Each mismatch is one error at the
value's position, naming the value, the field and the type it was checked against.

**Acceptance Criteria:**
- [ ] A string literal against a `string` field produces no diagnostic, including the empty string
- [ ] A string literal that parses as a date against a `date` field, as RFC 3339 against a `timestamp`
      field, and as a canonical uuid against a `uuid` field, each produce no diagnostic; a uuid written
      in upper case is accepted
- [ ] A string literal that does not parse as the declared format produces exactly one diagnostic, at
      `Error` severity, positioned on the value, whose whole formatted line names the value, the field
      and the declared type — asserted separately for `date`, `timestamp` and `uuid`, since one message
      is chosen per type
- [ ] A date-only value against a `timestamp` field is reported, and a timestamp value against a `date`
      field is reported
- [ ] A number literal against a `decimal` field produces no diagnostic whether or not it states a
      fractional part; a number literal with no fractional part against an `int` field produces none;
      one with a fractional part against an `int` field produces exactly one diagnostic
- [ ] `true` and `false` against a `bool` field produce no diagnostic
- [ ] Each of the cross-kind mismatches produces exactly one diagnostic: a number against `string`, a
      string against `int`, a boolean against `string`, a string against `bool`
- [ ] A field declared with a domain type accepts a string, a number and a boolean with no diagnostic,
      and so does a field whose declared type spells a DSL keyword — a type this story does not name is
      a domain type
- [ ] The set of type names this check knows is declared once in production code, and a test transcribes
      the seven names by hand and requires the two to agree, so a type cannot be taught to one surface
      alone
- [ ] A spec whose payloads state several mismatched literals produces one diagnostic per literal, in
      declaration order, identical across repeated runs
- [ ] These diagnostics carry no `RuleName`
- [ ] `cli.RunValidate` over a file whose payload states a mismatched literal exits with `ExitCode` 1
      and the reported message names the value and the declared type
- [ ] `oracle.Check` over the Task 2 fixture still returns no diagnostics, and a model that states no
      payload validates exactly as before

**Affected Files/Modules:**
- `internal/validator/validator.go` — the literal-kind check beside the payload field check from Task 3,
  and the field *types* the index carries for it
- `internal/validator/validator_test.go` — the type table and the ordering guarantee
- `internal/cli/validate_test.go` — the diagnostic as the user receives it

**Patterns to Follow:**
- The existing "this value must parse as a format" check, which already reports at the value's stored
  position and whose package already imports `time`: `scheduleExpressionDiagnostics` and
  `isWellFormedSchedule` (`internal/validator/validator.go:370-382` and below)
- `errorAt` (`internal/validator/validator.go:150-158`) for severity and the empty `RuleName`, and
  `sortInDeclarationOrder` (`:193-201`) for the ordering guarantee
- The transcription must be hand-written and compared against the production set, never derived from it
  — `tasks/learnings.md` "An assertion whose expected value comes from the code under test is the
  recurring review finding", and `viewerNodePalette` (`internal/diagram/contract_test.go:1305`) for the
  refuse-a-partial-table shape a cross-checked expectation takes
- One whole formatted line per assertion, since the message varies with the declared type —
  `tasks/learnings.md` "A rule whose message branches on model state is pinned by whole formatted lines"
- CLI-layer assertion content: `tasks/learnings.md` "CLI diagnostic tests must assert the distinguishing
  message text"

**Testable:** Yes — through `validator.Validate`, `oracle.Check` and `cli.RunValidate`.

**Verification:** `go test -tags unit ./internal/validator/... ./internal/cli/... ./internal/oracle/...`;
`go run ./cmd/emod validate` over a temporary file whose payload states `12.50` for an `int` field,
expecting exit 1.

**Depends on:** Task 3

---

### Task 5: Preserve payloads through `emod fmt`

**Behavior:** The formatter writes every payload back out, so formatting a model no longer loses the
example values its specs state. A payload is emitted on the line of the reference it qualifies, as a
brace block of comma-separated `field: value` pairs in declaration order, with strings written as
verbatim emod strings, numbers written as their source text, and `true` / `false` written bare. A
payload stating nothing is written as no braces at all, so `{}` and an omitted payload format alike. A
model whose specs carry no payload formats to exactly the bytes it formatted to before.

**Acceptance Criteria:**
- [ ] Parsing the Task 2 fixture, formatting it and re-parsing yields a model whose spec elements match
      the original in name, declaration order, and payload field names, values and literal kinds, across
      `given`, `when` and `then` — the comparison being against the original model, never against a
      second format run
- [ ] Formatting the formatter's own output produces byte-identical text
- [ ] A payload written across several lines, and one written with no commas between entries, both
      format to the same canonical single-line form, and re-parsing the result yields the same payload
- [ ] A payload written `{}` formats to a reference with no braces, and the result re-parses to a spec
      equal to one written with no payload
- [ ] A payload value containing a backslash, a tab, a double quote, a `%` and a non-ASCII character
      survives parse → format → parse → format with identical bytes, proving the text is never escaped
- [ ] A `12.50` value formats as `12.50`, not as `12.5`
- [ ] A names-only element formats exactly as before: `internal/formatter/formatter_test.go` and
      `internal/cli/fmt_test.go` pass with no edit to any existing expected-output constant, including
      `specFormattedEmod` (`internal/cli/fmt_test.go:244`)
- [ ] `internal/cli/fmt_test.go` gains a canonical formatted constant for the Task 2 fixture and feeds
      it to `requireFmtSettlesOn`, rather than handing the input fixture back as the expected value
- [ ] `emod fmt --check` over an already-formatted file whose specs state payloads reports no change
      needed, and the file on disk is unchanged

**Affected Files/Modules:**
- `internal/formatter/formatter.go` — `writeSpec` (`:372-386`), the element rendering that
  `specElementNames` (`:401`) and `bracketed` (`:65-67`) flatten today
- `internal/formatter/formatter_test.go` — the round-trip group (the spec leaf at `:876`), the
  quoting-hazard table (`:953`) and the canonical-order group `"specs"` (`:3597`)
- `internal/cli/fmt_test.go` — a canonical constant for the payload fixture beside `specFormattedEmod`
  (`:244`), and the command-level behaviour over it

**Patterns to Follow:**
- Entry order, the one-line writers and the list join: `writeSpec`
  (`internal/formatter/formatter.go:372-386`), `bracketed` (`:65-67`), `lineIfSet` (`:25-30`)
- `quoted` (`internal/formatter/formatter.go:57-63`) for every string, never `%q` —
  `tasks/learnings.md` "Never write emod source with `%q`", including its obligation to carry a
  round-trip subtest per hazard character
- The round-trip through the parser is what catches a mangled declaration, not a golden:
  `internal/formatter/formatter_test.go:876` — and fold a new per-fixture assertion into the existing
  round-trip leaf rather than opening a parallel table (`tasks/learnings.md` "A `Declared…` getter
  answers `nil` for a fixture that declares none of the construct")
- A fmt golden is a pinned canonical constant, never the input fixture handed back —
  `tasks/learnings.md` "`emod fmt` canonicalises order", `internal/cli/fmt_test.go:244` and
  `requireFmtSettlesOn`
- Every expected string starts with the `emod <n>` header — `tasks/learnings.md` "Formatter output
  always begins with `emod N`"
- `emod fmt <file>` writes in place, so a receipt run copies the fixture to a temp path first —
  `tasks/learnings.md` "`emod fmt <file>` writes in place"
- `tasks/learnings.md` "Additive output changes owe a byte-identical receipt for models that do not use
  the feature": the untouched goldens are that receipt, and stating it belongs to the commit message
  rather than to a criterion
- Alignment inside a payload is US-014's; this task emits one canonical form and pads nothing

**Testable:** Yes — through `formatter.Format` and `cli.RunFmt`.

**Verification:** `go test -tags unit ./internal/formatter/... ./internal/cli/...`; `go run ./cmd/emod fmt`
over a temporary copy of the Task 2 fixture, then again over the result.

**Depends on:** Task 2

---

### Task 6: Carry payloads through the JSON and CUE exports and the embedded schema

**Behavior:** Both model exports carry every payload under the reference that states it. A spec's
`given` entries, its `when` reference and a `then` outcome's events each become an object naming the
construct and, when one is stated, listing the payload's fields in declaration order with each value
carried as its natural JSON and CUE type. The bundled schema declares them. The diagram document — which
is nodes and edges — carries no trace of a spec or a payload, and neither the rendered diagrams nor the
glossary change. A model whose specs carry no payload exports and renders as it did before.

**Acceptance Criteria:**
- [ ] The JSON export of the Task 2 fixture files every payload under the reference that states it, in
      declaration order, read back against the hand-transcribed list from Task 2 rather than against
      another export
- [ ] A string value reads back as a JSON string, a number as a JSON number and `true` as a JSON boolean
- [ ] A reference that states no payload states no payload key at all, and a spec whose references are
      all names-only exports the same content it exports today apart from the element's shape
- [ ] The CUE export carries the same, and "CUE and JSON exports agree on the specs a model states"
      (`internal/export/export_test.go:3841`) passes for the payload fixture
- [ ] `internal/cue/schema.cue` declares the spec element and its payload field, and `cue vet -d '#Model'`
      over the export of the fixture passes (`internal/export/export_test.go:3812`)
- [ ] `emod schema` prints a schema that declares a payload on a spec element
- [ ] A document stating a spec's `given` in the retired shape — a bare list of names — fails `cue vet`
      with a message naming it, the way `internal/cue/embed_test.go:112` pins a retired key
- [ ] The keys inside the spec element object are emitted in the order its `json*` siblings emit theirs,
      pinned with `emittedKeyOrder` (`internal/export/export_test.go:4760`) against a sibling's key list
      in the same subtest
- [ ] Walking the whole diagram JSON document produced from the fixture finds no payload field name and
      no payload value at any key or depth — after first proving the same search finds them in the model
      document
- [ ] Every diagram rendering of the fixture — drawio, SVG, mermaid and ASCII — is byte-identical to the
      rendering of the same model with its payloads stripped, and the comparison opens by asserting the
      two models differ, that the twin states no payload, and that the featured model states the whole
      transcribed list
- [ ] The glossary markdown and JSON renderings of the fixture are identical to those of the twin
- [ ] The only expected values that move are those restating the spec element's wire shape —
      `libraryLendingSpecs` (`internal/export/export_test.go:4306`) and `fullModelJSON`
      (`internal/cue/embed_test.go:232`); no golden, canonical constant or transcribed list in
      `internal/formatter`, `internal/diagram`, `internal/glossary` or `internal/cli` moves

**Affected Files/Modules:**
- `internal/export/json.go` — `jsonSpec` (`:189-199`), `jsonSpecOutcome` (`:201-205`), `convertSpec`
  (`:420-435`), `convertSpecOutcome` (`:437-448`) and the `specElement*` helpers (`:450-473`)
- `internal/export/cue.go` — `writeSpec` (`:134-142`) and `writeSpecOutcome` (`:144-151`)
- `internal/cue/schema.cue` — `#Spec` (`:79-85`), `#SpecOutcome` (`:74-77`) and the element definition
  they gain
- `internal/export/export_test.go` — the read-back (`:1463`), the positions leaf (`:1467`), the schema
  conformance (`:3812`), the format-parity leaf (`:3841`), the diagram-document guard (`:3179`) and the
  transcription (`:4306`)
- `internal/cue/embed_test.go` — `fullModelJSON` (`:232`) and a retired-shape negative leaf
- `internal/diagram/contract_test.go` — the differential across the four renderers, alongside
  `"stating specs leaves the picture untouched"` (`:441`)
- `internal/glossary/glossary_test.go` — the receipt that the vocabulary is unchanged (`:586`)

**Patterns to Follow:**
- All three surfaces land together — `tasks/learnings.md` "A new exported field must land in JSON, CUE
  and `schema.cue` in the same change"
- Key order in a `json*` document type comes from its siblings, not from `schema.cue`, and is assertable
  from the raw bytes — `tasks/learnings.md` "JSON and CUE order their document keys differently" and
  "JSON key order is assertable from the raw bytes — `emittedKeyOrder` already exists"
- A changed wire shape owes a retired-shape negative on the surface that reads it —
  `tasks/learnings.md` "A key rename owes a retired-key negative assertion on every surface that reads
  the key", with `internal/cue/embed_test.go:112` as the worked example
- Read the decoded document back with the generic walkers in the writer's slice order: `objectsUnder`,
  `statedUnder`, `exportedSlices`, `listsKeyedBy` (`internal/export/export_test.go:4646-4740`) —
  `tasks/learnings.md` "Read a decoded export document back with `objectsUnder`/`statedUnder`"
- List and object emission in CUE: `writeCUEList`, `listIfSet` and `writeObject` as used at
  `internal/export/cue.go:117-151`
- Keep the diagram document forked, and prove the search works before asserting it finds nothing:
  `internal/export/export_test.go:3179` — and choose payload values distinctive enough that the search
  cannot match a position, an id or a construct name, which is what Task 2's value criterion is for
- A differential must first prove its twin differs and that the stripping reached every home:
  `internal/diagram/contract_test.go:441-450` and `tasks/learnings.md` "`require.NotEqual` on a stripped
  twin is satisfiable without stripping anything"
- Do not add a "render it twice" assertion — `tasks/learnings.md` "An assertion whose expected value
  comes from the code under test is the recurring review finding"
- A criterion listing expected values that may move names each of them, because a wire-shape change
  necessarily moves the constants that transcribe it — `tasks/learnings.md` "A 'no expected constant
  moves' criterion is unsatisfiable when the task edits a shared fixture"

**Testable:** Yes — through `export.ExportJSON`, `export.ExportCUE`, `export.ExportDiagramJSON`,
`cli.RunSchema`, the four `diagram.Export*` renderers and `glossary.RenderMarkdown` /
`glossary.RenderJSON`.

**Verification:** `go test -tags unit ./internal/export/... ./internal/cue/... ./internal/diagram/...
./internal/glossary/... ./internal/cli/...`; `go run ./cmd/emod export --format cue <fixture file>` and
`go run ./cmd/emod schema`.

**Depends on:** Task 2

---

### Task 7: Accept payloads in the tree-sitter grammar

**Behavior:** The tree-sitter grammar admits a payload block after any event or command reference inside
a spec, with string, number and boolean literals, so a file `emod validate` accepts is not red-squiggled
by a tree-sitter-backed editor. The grammar stays looser than the Go parser: any number of payload
fields in any order, commas optional, and the payload free to span lines.

**Acceptance Criteria:**
- [ ] Corpus cases parse a payload on a `given` element, on the `when` reference, and on a `then`
      event-list element
- [ ] A corpus case covers a payload with several fields, one with a single field, and one written `{}`
- [ ] A corpus case covers a string value, a number with a fractional part, a number without one, and
      `true` and `false`
- [ ] A corpus case places a payload-carrying element beside a names-only element in the same list, and
      another spreads a payload across several lines with no commas between its fields
- [ ] A corpus case for a `fields` block declaring fields named `true` and `false` parses them as field
      lines, not as literals
- [ ] The expected trees of the existing cases in `editors/tree-sitter-emod/test/corpus/specs.txt` are
      unchanged — the payload is admitted as an optional addition rather than by wrapping every
      reference in a new node
- [ ] The rule comment above `spec_definition` (`editors/tree-sitter-emod/grammar.js:106`) spells the
      payload out, so the file's one description of the construct is not left under-stating it
- [ ] `mise exec -- task test:grammar` passes, and running it a second time leaves every tracked file
      under `editors/tree-sitter-emod/` byte-identical
- [ ] No file under `editors/tree-sitter-emod/src/` is tracked — `.gitignore` still ignores it
- [ ] No file under `editors/tree-sitter-emod/queries/` changes and
      `editors/vscode/syntaxes/emod.tmLanguage.json` is untouched — highlighting numbers and booleans as
      literals is US-017
- [ ] `editors/tree-sitter-emod/test/queries/keywords_test.go` needs no edit, since this story adds no
      keyword

**Affected Files/Modules:**
- `editors/tree-sitter-emod/grammar.js` — the payload rule and its literals, and its optional place in
  `spec_when` (`:120-123`) and `spec_event_list` (`:133-140`)
- `editors/tree-sitter-emod/test/corpus/specs.txt` and `test/corpus/fields.txt` — the new cases

**Patterns to Follow:**
- Block bodies admit unordered, unbounded entries, and an `optional(...)` inside one is a bug:
  `buildDescribedBlock` (`editors/tree-sitter-emod/grammar.js:1-5`) and `tasks/learnings.md` "The
  tree-sitter grammar must never be stricter than the Go parser"
- The bracketed, comma-separated, possibly empty list already in the file: `spec_event_list`
  (`editors/tree-sitter-emod/grammar.js:133-140`) and `events_list` (`:158-166`)
- Keyword-named and literal-named field lines must keep matching `any_identifier` (`:339`) —
  `tasks/learnings.md` "New DSL keywords must stay usable as field names", with
  `editors/tree-sitter-emod/test/corpus/version_header.txt` as the existing field-named-after-a-keyword
  case
- Every rule carries a one-line example of its full shape — `tasks/learnings.md` "Every `grammar.js`
  rule carries a one-line example of its full shape"
- Corpus case layout: `editors/tree-sitter-emod/test/corpus/specs.txt`
- Run through `mise exec --` and do not un-ignore `src/` — `tasks/learnings.md` "Run repo tooling
  through `mise exec --`" and "Generated tree-sitter `src/` stays gitignored"

**Testable:** Yes — the corpus cases are the tests, run by `task test:grammar`.

**Verification:** `mise exec -- task test:grammar`, run twice, the second run leaving the tracked files
untouched; `git ls-files editors/tree-sitter-emod/src` returning nothing.

**Depends on:** Task 1

---

### Task 8: Document payloads in the DSL reference

**Behavior:** The DSL reference teaches the payload: where it may be written, that it is partial by
design, which literal forms exist, which declared field types each literal satisfies and what a value
must parse as, that any other declared type is a domain type accepting anything, and which tools carry
payloads. A reader learning the language finds it inside the `spec` subsection that already describes
Given-When-Then.

**Acceptance Criteria:**
- [ ] The `### spec` subsection (`docs/dsl-reference.md:376`) covers the payload block on any event or
      command reference in `given`, `when` and a `then` event list, with an example of each, and states
      that payloads are partial — a field declared `required` may be omitted — and that a names-only
      spec stays valid
- [ ] It states the three literal forms and, for each, the declared field types it satisfies: strings
      satisfy `string`, and `date`, `timestamp` and `uuid` when the value parses as that format, naming
      the format expected for each; numbers satisfy `decimal`, and `int` when there is no fractional
      part; `true` and `false` satisfy `bool`
- [ ] It states that any other declared type is a domain type and accepts any literal unchecked, that
      `then rejected` takes no payload, and that expected view-state payloads are not part of the
      language
- [ ] The field-type bullet in §8 "Fields" (`docs/dsl-reference.md:451`) names the types a payload
      literal is checked against and links to the `spec` subsection
- [ ] §11 "Cross-References" lists the payload field name as a referencing site for a command's or an
      event's fields, and its validation bullets name the two errors this story adds
- [ ] No `## <n>.` heading and no `### ` heading in `docs/dsl-reference.md` is added, removed, renamed or
      reordered, so every existing `(#<n>-<slug>)` and `(#<sub-heading>)` link still resolves
- [ ] Any block added under an ```emod fence is a whole model that `oracle.Check` reports nothing for; a
      fragment that is an illustration rather than a model takes a plain fence
- [ ] The subsection reads as if it were its first version, with no note of what the reference used to
      say

**Affected Files/Modules:**
- `docs/dsl-reference.md` — the `### spec` subsection in §6 (`:376-422`), the field-type bullet in §8
  (`:451`), and the table plus bullet list in §11

**Patterns to Follow:**
- Subsection voice, the fenced form-then-example shape and the consumer sentence:
  `docs/dsl-reference.md:376-422` (the `spec` subsection this task extends) and `:147` (the `invariant`
  subsection)
- Extending an existing subsection is safe; adding, renaming or reordering a heading is not —
  `tasks/learnings.md` "`docs/dsl-reference.md` anchors embed the section number" and
  "`docs/dsl-reference.md` sub-heading anchors are cited more often than the numbered ones", noting
  `#spec` is cited three times
- An ```emod fence is a promise the block validates, and `internal/oracle/oracle_test.go:112`
  "documented models" extracts every fenced block from `README.md` and `docs/dsl-reference.md` and
  requires `oracle.Check` to report nothing — `tasks/learnings.md` "An ```emod fence is a promise that
  the block validates"
- The literal-to-type table in `docs/proposals/specs-and-metadata-proposal.md:223-233` is the content to
  state, not to quote
- Write the document as if it were its first version

**Testable:** No — prose only; correctness is that any fenced model validates and no anchor breaks.

**Verification:** `go test -tags unit ./internal/oracle/...` (the "documented models" leaf at `:112`
extracts and checks every ```emod fence); reconcile the `^## [0-9]+\.` heading list against the
`\(#[0-9]+-` link list, and the `^### ` list against the `\(#[a-z]` list, in `docs/dsl-reference.md`.

**Depends on:** Tasks 3, 4, 5, 6

---

## Summary

**Total tasks:** 8

**Ordering rationale:** dependency-first, with the language surface settled before anything reads it.
Task 1 is the risky piece and the only one that touches the lexer: it introduces the first literal value
grammar the DSL has, and the two traps it has to clear — a payload's closing brace looking like the end
of the list that contains it, and a `{` on the next line being swallowed as a payload — are the failures
`tasks/learnings.md` records for exactly this shape of change. Task 2 turns that syntax into the one
model every later task asserts against, and is separated from Task 1 because the fixture must be clean
under checks Tasks 3 and 4 have not added yet, which is only true if its values were chosen with those
checks in mind. Tasks 3 and 4 deliver the story's two validation criteria in that order: Task 3 builds
the reference-to-construct-fields lookup, Task 4 puts the type table on top of it, and splitting them
keeps the first notion of a checked field type this repo has ever had in a commit of its own. Tasks 5, 6
and 7 close the surfaces this repo requires of any new construct — a formatter that does not know a
construct silently deletes it, the JSON/CUE/schema trio moves together, and the tree-sitter grammar must
never reject what `emod validate` accepts. Tasks 3-6 depend only on Task 2 and Task 7 only on Task 1, so
four of the eight can run alongside one another. Task 8 documents the finished surface.

**Coverage of the story's acceptance criteria:**

| Criterion | Task |
|---|---|
| Any event or command reference in `given`, `when` or `then` accepts a `{ field: value, ... }` block | 1 |
| Payloads are partial: fields declared `required` may be omitted | 1 (parsing), 3 (no diagnostic for an omitted field) |
| Names-only specs remain valid; payloads are additive per element reference | 1, 5 (formatter goldens), 6 (export and render receipts) |
| A payload field name not declared on the referenced construct's `fields` is a validation error | 3 |
| Literal kinds are checked against the declared field type, and a value must parse as its format | 4 |
| Fields of domain types accept any literal unchecked | 4 |
| Every existing `.emod` file stays valid with unchanged meaning (story-wide constraint) | 1 (the version header and the untouched keyword set), 5 (formatter goldens), 6 (export, diagram and glossary receipts) |

Tasks 2, 5, 6, 7 and 8 carry no story criterion of their own — they are the fixture kit and the fan-out
this repo requires of any new construct: one shared model rather than a bespoke one per package,
`emod fmt` not dropping it, the JSON/CUE/schema trio moving together, a grammar that is never stricter
than the parser, and a reference that teaches it.

Nothing from the story is deferred. What US-010 deliberately leaves to later stories: value-aware
boundary checking in DCB mode, which is where payloads stop being documentation and start being
model-checking (US-011); payload alignment and the wider canonical attribute order in `emod fmt`
(US-014); payload field-name completion, hover and navigation in the LSP, including the field-type list
`internal/lsp/completer.go:247` offers (US-015); rendering payloads on diagrams behind `--specs`
(US-016); highlighting numbers and `true` / `false` as literals in the VS Code grammar and the
tree-sitter highlight queries (US-017); and payloads in `examples/*.emod` (US-018).

# US-012: Bind model events to wire-level types

## Progress
- [x] Task 1: Spell `type` on the hand-maintained editor keyword surfaces
- [ ] Task 2: Carry a wire `type` attribute on `event`
- [ ] Task 3: Preserve wire types through `emod fmt`
- [ ] Task 4: Reject two events sharing one wire type
- [ ] Task 5: Carry wire types through the JSON and CUE exports and the embedded schema
- [ ] Task 6: Nudge wire types toward reverse-DNS kebab-case with `wire/type-format`
- [ ] Task 7: Document the wire type in the DSL reference

---

## Story Reference

`user-stories/specs-and-metadata.md` → **US-012: Bind model events to wire-level types**
(twelfth story of "Specs, Invariants, and Model Metadata", lines 159-167). Design notes:
`docs/proposals/specs-and-metadata-proposal.md` §6 "Wire-Level Event Types" (lines 284-297), the AST
shape at line 384, the keyword list at line 390, and the phase-4 summary at line 603. The story
declares no dependency and builds on `main` as it stands: US-001 through US-006 are delivered, so the
version header, the `description` attribute on `event`, the keyword-as-field-name fix, `emod
glossary`, named invariants and command-slice specs are all already in the tree.

**In scope:** the optional `type "<string>"` attribute on `event`, its value an opaque string; an
`emod validate` error when two events share the same wire type; the wire type in `emod export -f
json` and `-f cue`; and a `wire/type-format` info lint rule that nudges toward reverse-DNS kebab-case
without enforcing it. Carried with them, because the repo's writers and its drift guards would
otherwise break or silently drop the attribute: the `emod fmt` round-trip, the embedded CUE schema,
the tree-sitter grammar (which must never reject what `emod validate` accepts), the two editor
highlight surfaces, the LSP's keyword hover map, and the DSL reference.

**Out of scope:**

- **`emod glossary` showing wire types alongside events.** The story file's Open Questions assume not,
  "until a consumer asks for it", and this run holds that line. `internal/glossary/glossary.go:90-103`
  builds an event's term from its name and description only; a wire type is a deployment identifier,
  not ubiquitous language, and adding it would mean a `term` field every other term kind leaves empty.
  Task 5's criteria carry an unchanged-glossary receipt so the decision is checkable rather than
  assumed.
- **Formatter canonical ordering beyond a round-trip (US-014).** Task 3 places `type` and nothing
  else; `given`/`when`/`then` alignment, flow colon alignment, payload wrapping and the wider
  attribute order stay with US-014.
- **LSP completion and go-to-definition for the new keyword (US-015).** Hover is pulled in — see
  below — but the per-block completion lists in `internal/lsp/completer.go` are not: no test forces
  them (`internal/lsp/keywords_test.go:59-91` asserts only that everything offered *is* a lexer
  keyword, never the converse), and US-015 owns the feature.
- **Syntax highlighting as a feature (US-017).** Task 1 gives `type` the minimum each highlight
  surface needs to stay green, and nothing more: no scope is added for any other new keyword, and no
  existing scope moves. The consequence is worth stating plainly rather than leaving for US-017 to
  discover: because this story is what introduces `type` to the lexer, and because
  `TestEditorKeywordCoverage` will not tolerate a keyword no editor surface spells, **this story
  necessarily satisfies the `type` half of US-017's first criterion**. US-017 keeps that criterion —
  it also names `spec`, `given`, `when`, `then`, `rejected`, `invariant`, `description`, `after` and
  `emod` — and inherits `type` already done, along with its second and third criteria (literals in
  payload position, and a keyword in field-name position reading as a field name) untouched here
  beyond the two-directional assertions Task 1 owes for `type` alone.
- **Examples and reference coverage (US-018).** No file under `examples/` gains a wire type. Task 7
  documents the construct in `docs/dsl-reference.md` only, following the precedent of US-001 and
  US-002, which each documented the construct they introduced.
- **The diagram document, the importer and the web viewer.** `jsonDiagramEvent`
  (`internal/export/diagram.go:53-63`) forks `jsonEvent` precisely so a new AST field cannot leak into
  the node-and-edge contract, and a subtest already walks the whole diagram document asserting the key
  appears nowhere (`tasks/learnings.md:56-59`). US-002 drew the same line for `description`. The
  consequence — a wire type does not survive a save from the viewer, whose export path is
  `importer.ImportDiagram` — is the same one descriptions already have, and changing it is a separate
  contract decision.
- **Diagram rendering.** No acceptance criterion reaches draw.io, SVG, mermaid or ascii; those
  renderers owe only a byte-identical receipt. The edge half needs no thought at all: `SliceEdges`
  (`internal/diagram/graph.go:51`) is now the single edge derivation the drawio, SVG and diagram-JSON
  exporters all read, and it keys entirely on construct *names* — an automation's activation event, a
  translation's command and event — never on an event's attributes. A wire type cannot reach an edge,
  so the receipt is a comparison, not an audit.

**Deliberately pulled in, and why.** Three surfaces the story's criteria do not mention are
nevertheless required for the tree to stay green, each forced by a guard that landed after US-002:

1. **`emod fmt` (Task 3).** The formatter renders from the AST and emits only what it knows, so a run
   of `emod fmt` would silently *delete* every wire type an author writes, hollowing out the export
   criteria this story does own. Neither idempotence nor an existing golden notices; only a
   parse → format → reparse comparison does (`tasks/learnings.md:156-159`).
2. **The three editor keyword surfaces (Task 1).** `TestEditorKeywordCoverage`
   (`editors/tree-sitter-emod/test/queries/keywords_test.go:47-64`) iterates `lexer.Keywords()` and
   requires `editors/tree-sitter-emod/grammar.js`, `editors/tree-sitter-emod/queries/highlights.scm`
   and `editors/vscode/syntaxes/emod.tmLanguage.json` each to spell every one of them. The moment
   `internal/lexer/token.go` learns `type`, all three go red under `task test:grammar`, which is a CI
   step.
3. **The LSP keyword hover map (Task 2).** `TestKeywordCoverage/hover`
   (`internal/lsp/keywords_test.go:19-57`) requires `lsp.GetHover` to return non-empty text for every
   lexer keyword, so `keywordDescriptions` (`internal/lsp/hover.go:10-49`) must gain a `type` entry in
   the same change as the lexer or `task test:unit` fails.

**Open questions, decided.** Six shapes the story does not settle:

1. *The uniqueness check spans the whole model, not a context.* A wire type is by construction an
   identifier outside the model — the subject a schema registry keys on, the `type` a CloudEvents
   consumer routes by — and nothing in an `.emod` file says two contexts publish to different brokers,
   so two contexts binding one wire type produce output a consumer cannot tell apart. The repo agrees:
   `modelIndex` (`internal/validator/validator.go:57-122`) keys commands and events model-wide and
   unqualified, which is why a flow may name an event declared in another context. Invariants are the
   deliberate contrast — US-005 scoped them per aggregate or context because an invariant name never
   leaves the model. A wire type always does.
2. *An empty wire type is no wire type.* `type ""` is treated exactly as `description ""` is today:
   the formatter's `quotedLineIfSet` drops the line, the JSON export omits the key via `omitempty`,
   the CUE writer's `lineIfSet` skips it, the uniqueness check ignores it and the lint rule stays
   silent. The emptiness test is the value, never a stored position. Inventing a "stated but empty"
   state would give this one attribute a concept no other optional attribute in the repo has.
3. *Wire types compare verbatim.* CloudEvents types are case-sensitive, and the story calls the value
   opaque, so two events binding `com.acme.room-reserved` and `Com.Acme.Room-Reserved` do not collide.
   The lint rule flags the second for its shape; the validator does not.
4. *`wire/type-format` flags one shape with one message.* A wire type conforms when it is two or more
   dot-separated segments, every segment lowercase and built from `a`-`z`, `0`-`9` and hyphens, no
   segment empty, and no segment opening or closing with a hyphen. Anything else draws exactly one
   info diagnostic carrying one text, which names the event, the offending value and the convention.
   The rule does not branch on *which* part failed: a message that varies with model state can only be
   pinned by whole-line comparison and multiplies the leaves needed to hold it honest
   (`tasks/learnings.md:476-479`).
5. *"Without enforcing it" is the severity label, not the exit code.* `RunValidate`
   (`internal/cli/validate.go:33-35`) and `formatText` (`internal/cli/lint.go:80-89`) return exit 1
   for *any* diagnostic, severity ignored — `dcb/single-tag-everywhere`, the only info rule in the
   tree today, already behaves this way (`tasks/learnings.md:466-469`). Changing that is a repo-wide
   exit-code decision, not this story's. What actually makes the rule non-enforcing is its narrowness:
   it can only fire on an event that *states* a wire type, so no model that does not use the feature
   can trip it, which is the same fact the story's "exactly as before" criterion asks for.
6. *The wire type exports no position key.* `jsonEvent` (`internal/export/json.go:99-111`) carries
   `source_position` and `external_name_position` but no `description_position`: position export is a
   judgement call per field. A wire type follows `description`, its closest analogue — an opaque
   optional string with no reference resolving against it — and exports the text only.

**Overarching constraint.** "Events without a wire type validate and export exactly as before" is
load-bearing in six places: `emod fmt` must still produce its current bytes, no existing golden,
canonical `*FormattedEmod` constant or transcribed name list may be edited, the JSON and CUE exports
of `test.HotelReservation` must stay byte-identical, every checked-in `.emod` must keep validating
clean, the diagram document must not gain a key, and `type` must remain usable as a field name.

**Learnings folded in** from `tasks/learnings.md`: ask the lexer which keywords exist and never
restate the set; a new keyword must stay usable as a field name on both the Go and the tree-sitter
side; keyword surfaces fan out past the lexer, so decide per surface; a keyword that is only a keyword
in one position never joins the flat TextMate alternation; every `grammar.js` rule carries a one-line
example of its full shape; the tree-sitter grammar must never be stricter than the Go parser;
generated tree-sitter `src/` stays gitignored and repo tooling runs through `mise exec --`; a test
that shells out to a CLI runs with `-count=1`; a quoted block entry is one `case` on the
`parse*EntryInto` family; a new block entry keyword owes three things to the parser's diagnostics;
put a new parser subtest in the group that owns the construct; assert a short keyword in a diagnostic
with a `\b`-bounded `require.Regexp`; a second `require.Contains` on one message is often shadowed by
the first; never write emod source with `%q`; `emod fmt` canonicalises order, so a fmt golden is
never the input re-indented; formatter output always begins with `emod N`; a new block entry goes
after `description` and ahead of nested blocks, in every writer; additive output changes owe a
byte-identical receipt; a differential receipt must first prove the twin actually differs;
`require.NotEqual` on a stripped twin is satisfiable without stripping anything; a new optional field
ships a six-part fixture kit; exercise an omitted optional part mid-block; a new shared fixture owes
`internal/oracle` a zero-diagnostic subtest; a slice has two homes; a new exported field must land in
JSON, CUE and `schema.cue` in the same change; JSON and CUE order their document keys differently;
JSON key order is assertable with `emittedKeyOrder`; a serialized key spells the DSL keyword; the two
export guards cannot see a list neither writer emits; diagnostics gathered from more than one AST
collection must be position-sorted; `RuleName` marks a diagnostic `emod lint --explain` can describe;
a lint warning fails `emod validate`; a lint fixture trips exactly one rule; a rule whose message
branches on model state is pinned by whole formatted lines; CLI diagnostic tests must assert the
distinguishing message text; urfave/cli v2 discards every flag written after the file argument; an
```emod fence is a promise that the block validates; `docs/dsl-reference.md` anchors embed the section
number and its sub-heading anchors are cited more often; a task's change-set assertion must name every
file its own patterns require it to change; a commit-message receipt is never an acceptance criterion;
and no criterion may reference commit, branch or remote state.

---

## Codebase Context

**Pipeline.** `internal/oracle/oracle.go:17-38` is the one lex → parse → validate → lint chain every
frontend runs; `Check` returns validator *and* linter diagnostics together, which is why `emod
validate` and `emod lint` report identically and why an info-severity rule is not free.

**Lexer.** `internal/lexer/token.go` holds a single `keywords` map (`:73-112`, thirty-eight spellings
today) and derives `Keywords()` (`:124-126`) and `Kind.IsKeyword()` (`:135-138`) from it.
`internal/lexer/tokenizer.go:203` is the only lookup site. Five keyword-coverage suites iterate
`Keywords()` and name no keyword — `internal/lexer/tokenizer_test.go:14`,
`internal/parser/parser_test.go:225` and `:243`, `internal/oracle/oracle_test.go:45`,
`internal/lsp/keywords_test.go:20` — so `type` falls under "usable as a field name, type and modifier"
the day the map learns it.

**US-003's fix is keyword-set-driven, so `type` inherits it with no list to extend.**
`checkIdentifierLike` (`internal/parser/parser.go:1506-1512`) accepts a token when it is an
`Identifier` *or* when `Kind.IsKeyword()` answers yes, and `IsKeyword()` is a lookup in the inversion
of the one `keywords` map — not an ordinal range and not an enumerated list. `parseFields`,
`parseField` and `parseTagEntry` all gate on that one helper. So the motivating case the story file
calls out at line 52, `fields { type string required }`, keeps working the moment the map gains the
spelling, and the two subtests that would catch a regression already iterate `Keywords()` without
naming any keyword. Task 2 states this as a criterion rather than adding a case.

Two further suites turn the same list into obligations rather than gifts:
`internal/lsp/keywords_test.go:19-57` (hover) and
`editors/tree-sitter-emod/test/queries/keywords_test.go:47-64` (the three editor surfaces).

**AST.** `ast.Event` (`internal/ast/ast.go:140-154`) pairs every value with its position —
`Description`/`DescriptionPos`, `Source`/`SourcePos`, `ExternalName`/`ExternalNamePos`. The proposal
names the new pair `WireType`/`WireTypePos` (`docs/proposals/specs-and-metadata-proposal.md:384`);
`Type` would collide in the reader's eye with `ast.Field.Type` (`internal/ast/ast.go:159`), and the
repo already lets the Go identifier name the concept while the serialized key spells the DSL keyword
(`ast.Automation.Schedule` ships as `every`).

**Parser.** `parseEvent` (`internal/parser/parser.go:929-992`) is an unbounded loop that runs to the
closing brace or end of input, dispatching on each entry's leading keyword, with a fallback at `:979`
reading `expected description, fields, source, or tags in event`. The
`description` branch (`:952-953`) is one call to `parseQuotedEntryInto` (`:1419-1430`), which advances
the keyword, requires a `lexer.String`, reports once at the offending token and drains with
`skipRestOfLineOrBlockEnd`. Ten such fallback messages exist across the block constructs (`:205`,
`:269`, `:318`, `:416`, `:621`, `:673`, `:979`, `:1024`, `:1130`, `:1193`) and the tests assert their
text. A translation's nested event goes through the same `parseEvent`.

**Parser tests.** `internal/parser/parser_test.go` is one umbrella split into thirteen top-level
groups; `"commands, events and flows"` owns the event and `"error reporting"` owns recovery and
message shapes. The keyword-as-field-name subtests (`:225`, `:243`) iterate `lexer.Keywords()`.

**Formatter.** `writeEvent` (`internal/formatter/formatter.go:269-283`) emits comments, the header
line, `description` (`:272`), `tags` (`:273-275`), `source external` (`:276-278`) and `fields`
(`:279-281`). `writeDescription` (`:75-77`) is a `quotedLineIfSet`, and `quoted()` (`:61-63`) is a
bare pair of quotes — deliberately not `%q`, since the lexer performs no unescaping.

**Validator.** `Validate` (`internal/validator/validator.go:15-41`) runs each check in turn and wires
the redeclaration check at `:33`. `redeclaredInvariantDiagnostics` (`:266-273`) is the duplicate-name
template: a seen-set flagging the second and later occurrence (`:232-244`), a
message naming both the symbol and the scope, `errorAt` (`:150-158`) as the constructor with no
`RuleName`, and `sortInDeclarationOrder` (`:193-201`) applied before returning because the AST keeps
each kind in a field of its own. `modelIndex.collect` (`:83-122`) shows exactly which collections hold
events: `slice.Events` (`:88-95`) and each translation's nested event (`:99-104`), over
`model.AllSlices()` (`:74`), which visits both slice homes.

**Linter.** There is no rule registry: a rule is a `check*` function passing a literal rule name to
`info`/`warning`/`errorEntry` (`internal/linter/linter.go:16-47`), a hand-written call site, and an
entry in `ruleDescriptions` (`internal/linter/descriptions.go:10-31`). Severity is chosen statically
at the call site — there is no config file, no `--severity` flag and no options parameter on
`linter.Lint`. `checkClickbaitEvent` (`:524-529`) is the closest template for a per-event rule: it is
called from *both* event loops in `checkSlice` (`:111-113` for slice events, `:118-120` for a
translation's nested event) and deliberately sits outside `checkEvent` (`:393-404`), whose early
returns would suppress it. `dcb/single-tag-everywhere` (`:234-266`) is the only info rule today and
its tests (`internal/linter/linter_test.go:2226-2739`) are the severity template.
`emod lint --explain` resolves through `RunLintExplain` (`internal/cli/lint.go:91-102`) into
`RuleDescription`; the "all rules have descriptions" leaf hardcodes a seventeen-element list at
`internal/cli/lint_test.go:627-645`, so a new rule is untested there unless it is added by hand. No
shipped document enumerates rule names — `README.md`, `docs/architecture.md` and
`docs/dsl-reference.md` all mention linting without listing rules.

**Exports.** `jsonEvent` (`internal/export/json.go:99-111`) opens with `Name`, `Description`, the
positions, `Comments`, then its own values `Source`, `ExternalName`, `Fields`; `convertEvent`
(`:498-515`) is reached from `convertEvents` (`:406`) and from the translation's nested event (`:639`).
The CUE side, `cueWriter.writeEvent` (`internal/export/cue.go:168-175`), emits `comments`, `name`,
`description`, `source`, `external_name`, `fields` — a different order from the JSON struct, matching
`internal/cue/schema.cue:30-37` (`#Event`). Three subtests couple the surfaces: "CUE and JSON exports
describe the same model", `cue vet -d '#Model'` over the embedded schema, and `emittedKeyOrder`
(`internal/export/export_test.go:4760`), which reads the emitted key order out of the raw bytes.
The `cue` binary is optional here and the suites skip when it is absent.

**Fixtures.** `internal/test/fixtures.go` pairs an unfeatured witness with a featured model per
optional feature — `HotelReservation` (`:13`) and `DescribedHotelReservation` (`:101`),
`SpecLibraryLending` (`:423`), `AutomationReadsLibraryLending` (`:581`),
`TriggerReadsLibraryLending` (`:749`), `AutomationScheduleLibraryLending` (`:947`). The most recent
kits ship six parts: the fixture const declaring the feature in both slice homes with one instance
omitted mid-block, a `…Model(t)` accessor in `internal/test/models.go`, a hand-transcribed expectation
var in declaration order, a `Without…` twin built on `copyWithEditedSlices` (`:1257-1269`) and
`editedCopies` (`:1275-1281`), a `Declared…` getter walking `declaredSlices` (`:1364`), and a
zero-diagnostic `oracle.Check` leaf in `internal/oracle/oracle_test.go:26-64`. `editedCopies` leaves a
nil list nil on purpose and its copies are shallow, so an edit reaching inside a slice's events must
nest a second `editedCopies` or it writes through to the caller's model.

**Editor surfaces.** `editors/tree-sitter-emod/grammar.js` folds `description` into every construct
through `buildDescribedBlock` (`:1-5`, with the `description` rule at `:52-55`), while per-construct
entries are listed as further arguments — `event_definition` (`:209-219`) passes `tags_block`,
`fields_block` and the `source external` sequence. Each rule carries a one-line comment spelling the
construct out whole, and nothing tests those comments. `any_identifier` (`:339`) is the permissive
regex that keeps keywords usable as field names, so no grammar change is needed for that.
`editors/tree-sitter-emod/queries/highlights.scm:24-63` is the hand-kept `@keyword` list of anonymous
node literals — `on` and `every` are in it, and a field named after either still captures
`variable.member`, because a field name parses as the named `any_identifier` node and never as the
anonymous literal. A query naming a node type the grammar does not define fails to compile, so the
grammar and the query must move together. `editors/vscode/syntaxes/emod.tmLanguage.json` is the
opposite shape: `#keywords` (`:95-98`) is a positionless case-insensitive alternation, so `on` and
`every` are kept out of it and live in `#activations` (`:67-84`), case-sensitive and keyed on their
operand — `every` matching only before a quoted string (`:80`). The `fields` block rule includes
`#standalone-tokens` (`:85-94`, which pulls in `#keywords`) and deliberately omits `#activations`.
Assertion files: `editors/tree-sitter-emod/test/corpus/*.txt`, `test/highlight/*.emod` (picked up
automatically by `tree-sitter test`), and `editors/vscode/test/scopes/*.emod` (run by
`task test:vscode`). A highlight marker only discriminates while another highlighted token follows it
on the same line, which is why `test/highlight/unreserved-keywords.emod` writes trailing comments.

**Documentation guard.** `internal/oracle/oracle_test.go:112-129` extracts every ` ```emod ` fence
from `README.md` and `docs/dsl-reference.md` and requires `oracle.Check` to report nothing. Because
`Check` runs the linter, a documented wire type must be a conforming one or the reference stops
building. `docs/dsl-reference.md` numbers its `##` headings and slugs its links from those numbers,
and fourteen further links cite `###` sub-heading slugs; nothing in CI checks either family.

**Test conventions** (`CLAUDE.md` "Go Test Organization"): one umbrella `Test{TypeName}` per type,
`t.Run` groups named after the operation, leaf subtests reading as sentences about the observed
outcome, `testify/require`, fresh fixtures per leaf, `//go:build unit` / `//go:build integration`
tags. AST comparisons use `test.RequireEqual` with `cmpopts.IgnoreTypes(ast.Position{})`.

---

## Tasks

### Task 1: Spell `type` on the hand-maintained editor keyword surfaces

**Behavior:** The tree-sitter grammar parses `type "<text>"` inside an `event` block, the tree-sitter
highlight query paints it as a keyword there, and the VS Code grammar paints it as a keyword only
where it opens a quoted attribute — while a field named `type` keeps its field scopes on both
surfaces. Every existing corpus, highlight and scope expectation is unchanged.

This task runs **before** the lexer learns the keyword, and that order is the point.
`TestEditorKeywordCoverage` (`editors/tree-sitter-emod/test/queries/keywords_test.go:47-64`) asserts
one direction only — every spelling `lexer.Keywords()` reports must appear in each of the three files
— so a surface may spell a word the lexer does not yet define, and all three suites stay green. The
reverse order does not exist: adding the keyword first turns `task test:grammar` red until this task
lands. The grammar accepting syntax the Go parser does not yet accept is the safe direction, and the
only one this repo permits (`tasks/learnings.md:61-64`).

**Acceptance Criteria:**
- [x] `grammar.js` parses `type "<text>"` among an `event` block's entries, in any position within the
      block and any number of times, with a corpus expectation naming the node and containing no
      `ERROR` or `MISSING` node
- [x] The nested `event` inside a `translation` accepts it on the same terms, with its own corpus case
- [x] A corpus case parses `fields { type string required }` as a `field_line` of `any_identifier`
      nodes, mirroring `test/corpus/version_header.txt:31-58`
- [x] The `// event Name { ... }` comment above `event_definition` names the new entry, so the file's
      one description of the construct still states it whole
- [x] `highlights.scm` captures the `type` keyword of an attribute as `@keyword`, asserted by a marker
      in `editors/tree-sitter-emod/test/highlight/`, and the same file asserts that a field named
      `type` still captures `variable.member` and a field *typed* `type` still captures `@type`
- [x] Each new highlight marker has a further highlighted token after it on its source line, so the
      assertion discriminates rather than scanning on into a later line
- [x] `emod.tmLanguage.json` scopes `type` as `keyword.control.emod` only where a quoted string
      follows it, case-sensitively, matching the `every` rule at `:80` — it is **not** added to the
      positionless `#keywords` alternation at `:97`, and the pattern group it joins is one the `fields`
      block rule does not include
- [x] The rule group `type` joins states in its name or `comment` what it now covers, rather than
      leaving a `type` attribute filed under a heading that says activations
- [x] `editors/vscode/test/scopes/` asserts both directions: `type "com.acme.reservations.room-reserved"`
      scopes `type` as a keyword and the value as a string, and `fields { type string required }`
      scopes `type` as a field name with the keyword scope absent
- [x] An event named `Type` and a field named `Type` keep their existing scopes on both surfaces — the
      lexer's keyword lookup is case-sensitive, so the highlight rules must be too
- [x] `TestEditorKeywordCoverage` still passes for every spelling `lexer.Keywords()` reports today,
      and its "yields distinct spellings rather than one run of text" leaf still holds for all three
      files
- [x] Every existing corpus, highlight and scope expectation matches unchanged
- [x] `mise exec -- task test:grammar` and `mise exec -- task test:vscode` pass, and no file under
      `editors/tree-sitter-emod/src/` is tracked, and `editors/tree-sitter-emod/.gitignore` is
      unmodified

**Affected Files/Modules:**
- `editors/tree-sitter-emod/grammar.js` — a wire-type rule, its use in `event_definition` (`:209-219`),
  and the rule's leading comment
- `editors/tree-sitter-emod/test/corpus/` — the attribute on a top-level event, on a translation's
  nested event, and the field named `type`
- `editors/tree-sitter-emod/queries/highlights.scm` — the spelling in the `@keyword` list (`:24-63`)
- `editors/tree-sitter-emod/test/highlight/` — markers for the keyword position and the field
  positions
- `editors/vscode/syntaxes/emod.tmLanguage.json` — a case-sensitive quoted-attribute pattern
- `editors/vscode/test/scopes/` — the two-directional scope assertions

**Patterns to Follow:**
- Block entries join the repeated-choice list `buildDescribedBlock` builds; making an entry
  at-most-once in a block body is a bug, not a constraint, because the Go parser imposes no arity
  (`tasks/learnings.md:61-64`)
- The `description` rule (`editors/tree-sitter-emod/grammar.js:52-55`) is the keyword-then-string
  entry to copy; `version_header` (`:21-24`) is the worked example of a token narrowed so it cannot
  match outside its own position
- Every rule carries a one-line example of its full shape (`tasks/learnings.md:251-254`)
- `highlights.scm` keyword entries are anonymous node literals and are inherently positional — `on`
  and `every` are already in the list while `test/highlight/unreserved-keywords.emod` proves a field
  named after either keeps `variable.member`; the field patterns select by leading `.` anchors with
  `(comment)*` steps, never by `#match?` (`tasks/learnings.md:496-499`)
- A highlight marker needs a trailing token on its line to discriminate
  (`tasks/learnings.md:491-494`)
- The TextMate split: `#keywords` (`emod.tmLanguage.json:95-98`) is positionless and case-insensitive,
  so a keyword that is only a keyword in one position lives elsewhere — `#activations` (`:67-84`) is
  the precedent, and `every`'s pattern at `:80` is the exact shape for a keyword before a quoted
  string (`tasks/learnings.md:501-504`)
- The suite that pins another tool's output owes a mutated-input negative control — the existing scope
  and query suites each carry one, and a new assertion file follows suit
  (`tasks/learnings.md:506-509`)
- Run the targets through `mise exec --`: the repo pins tree-sitter-cli 0.26.9 while a global 0.25.10
  may win on PATH (`tasks/learnings.md:11-15`); `task test:grammar` already passes `-count=1` because
  the grammar and query files reach the Go tests through a spawned CLI
  (`tasks/learnings.md:511-514`)
- Generated `src/` stays gitignored — do not un-ignore it to prove regeneration
  (`tasks/learnings.md:16-19`)

**Testable:** Yes — the corpus suite, the `test/highlight/*.emod` runner and `vscode-tmgrammar-test`
all exercise these files through the real tooling.

**Verification:** `mise exec -- task test:grammar` and `mise exec -- task test:vscode` pass;
`mise exec -- task test:unit` and `mise exec -- task test:integration` are unaffected and still pass;
`git status` shows no new or modified files under `editors/tree-sitter-emod/src/`.

**Depends on:** None

---

### Task 2: Carry a wire `type` attribute on `event`

**Behavior:** An `event` may contain `type "<text>"` inside its block, and the parsed event carries
that text with its source position. The attribute is optional, `type` stays usable as a field name, a
file that uses no wire types parses to the same AST and the same diagnostics as before, and the LSP
describes the new keyword on hover.

**Acceptance Criteria:**
- [ ] A top-level `event` accepts `type "<text>"` among its entries, in any position within the block,
      and the parsed event exposes that text; parsing such a model produces zero diagnostics
- [ ] The value is recorded with its source position, matching the value-plus-position convention of
      `Event.Description`/`DescriptionPos` and `Event.Source`/`SourcePos`
      (`internal/ast/ast.go:140-154`)
- [ ] The `event` nested inside a `translation` accepts it on the same terms as a top-level event
- [ ] The entry stated twice in one event block keeps the last value, matching how `description`
      already behaves — the block loop imposes no arity
- [ ] `fields { type string required }` parses as an ordinary field named `type`, and
      `fields { published type required }` as an ordinary field *typed* `type`; both are covered by
      the existing subtests that iterate `lexer.Keywords()` (`internal/parser/parser_test.go:225`,
      `:243`) without naming the keyword
- [ ] `type` followed by something other than a quoted string produces exactly one diagnostic — pinned
      by a length assertion, not by inspecting only the first entry — that names the construct and the
      offending token, sits at the offending token, and leaves the enclosing block parsing to
      completion with its remaining contents reported
- [ ] The event block's fallback message (`internal/parser/parser.go:979`) names the new entry, and
      the assertion on it is a `\b`-bounded `require.Regexp` rather than a `require.Contains` that a
      four-letter needle could satisfy from an unrelated word
- [ ] `lsp.GetHover` returns non-empty text for `type` in an event block, so
      `TestKeywordCoverage/hover` (`internal/lsp/keywords_test.go:19-57`) passes for every spelling
      `lexer.Keywords()` reports
- [ ] `internal/test.HotelReservation`, `internal/test.DescribedHotelReservation` and every fixture
      under `internal/parser/testdata/` parse to the same AST and the same diagnostics as before this
      task, and none of them gains a wire type
- [ ] A new shared fixture in `internal/test/fixtures.go` declares wire types on events in **both**
      slice homes — inside an aggregate and directly on a `mode dcb` context — with one event's wire
      type omitted **mid-block**, ahead of a further event, and with a translation's nested event
      carrying one; every wire type it states is distinct and conforms to reverse-DNS kebab-case, so
      the fixture stays clean once Tasks 4 and 6 land
- [ ] The fixture ships the rest of its kit: a `…Model(t)` accessor in `internal/test/models.go`, a
      hand-transcribed expectation var listing the declared wire types in declaration order across both
      homes, a `Without…` twin, and a `Declared…` getter walking `declaredSlices`
- [ ] The twin is proved to differ before anything is compared against it: the stripped model reads
      back an empty declared-wire-type list while the stated model reads back the transcribed one, so
      a twin that stripped nothing fails
- [ ] `oracle.Check` on the new fixture returns zero diagnostics, asserted as its own leaf in
      `internal/oracle/oracle_test.go`'s "clean input" group

**Affected Files/Modules:**
- `internal/lexer/token.go` — a new `Kind` in the iota keyword block plus its `keywords` map entry
  (`:73-112`)
- `internal/ast/ast.go` — the wire-type text and position on `Event` (`:140-154`); name the Go field
  for the concept so it does not read as `ast.Field.Type` (`:159`)
- `internal/parser/parser.go` — a branch in `parseEvent`'s entry loop (`:951-982`) and the fallback
  message at `:979`
- `internal/parser/parser_test.go` — capture on a top-level and a nested event, repetition, the
  malformed-value diagnostic, the fallback message, backward compatibility
- `internal/lsp/hover.go` — a `keywordDescriptions` entry (`:10-49`)
- `internal/test/fixtures.go`, `internal/test/models.go` — the six-part fixture kit
- `internal/oracle/oracle_test.go` — the zero-diagnostic leaf

**Patterns to Follow:**
- One `keywords` map entry and one `Kind`; `Kind.String()` needs no switch case, and every
  keyword-coverage suite picks the spelling up from `Keywords()`
  (`tasks/learnings.md:81-84`)
- The entry is one `case` calling `parseQuotedEntryInto` (`internal/parser/parser.go:1419-1430`),
  which already carries the advance, the `errorAt` and the `skipRestOfLineOrBlockEnd` drain that keeps
  one malformed line from cascading; the message interpolates the consumed keyword, so nothing
  existing moves (`tasks/learnings.md:316-319`)
- A new block entry keyword owes three things to the parser's diagnostics: the fallback message
  edited, single-diagnostic recovery, and a length assertion pinning the count at one
  (`tasks/learnings.md:76-79`)
- The `description` branch (`internal/parser/parser.go:952-953`) is the shape; `parseEvent`
  (`:929-992`) is the loop
- Put the subtests in the group that owns the construct — `"commands, events and flows"` for capture,
  `"error reporting"` for recovery and message shape (`tasks/learnings.md:106-109`)
- The fixture kit's six parts, `copyWithEditedSlices`/`editedCopies` (`internal/test/fixtures.go:1257`,
  `:1275`) and the two mechanics that make a twin honest — a nil list left nil, and shallow copies
  needing a nested `editedCopies` to reach inside a slice's events
  (`tasks/learnings.md:216-219`, `:206-209`)
- An optional part omitted at the end of a block witnesses nothing; put the omission mid-block
  (`tasks/learnings.md:91-94`)
- A fixture declaring the construct in one slice home cannot catch the walkers that visit only one
  (`tasks/learnings.md:171-174`)
- A `mode dcb` context needs tags on its events and a `decides_on` reaching them, or the fixture trips
  `dcb/untagged-event` and `dcb/orphan-tag-key`; write the fixture, add the oracle leaf, and let it
  tell you the model is not clean (`tasks/learnings.md:151-154`)
- Assert a short keyword in a diagnostic with a `\b`-bounded `require.Regexp`
  (`tasks/learnings.md:236-239`), and check a new `require.Contains` needle is not already inside an
  earlier one (`:136-139`)
- Caller pattern **Inbound** (`~/.config/ai/guidelines/testing/caller-patterns.md`): the source text is
  the input and the `(*ast.Model, diagnostics)` pair the observable outcome — assert acceptance,
  rejection and the resulting AST, never parser internals

**Testable:** Yes — `lexer.Scan`, `parser.Instance.Parse`, `lsp.GetHover` and `oracle.Check` are all
exported and already have suites.

**Verification:** `mise exec -- task test:unit` and `mise exec -- task test:integration` pass, and
`mise exec -- task test:grammar` stays green because Task 1 already taught the three editor surfaces
the spelling.

**Depends on:** Task 1

---

### Task 3: Preserve wire types through `emod fmt`

**Behavior:** Formatting a model that carries wire types emits them back, on the line after the
event's `description`. A model with no wire types formats byte-identically to before.

This task exists to stop a regression rather than to add a feature: until it lands, one `emod fmt`
run deletes every wire type an author writes, and neither idempotence nor any existing golden notices.

**Acceptance Criteria:**
- [ ] Formatting the wire-type fixture, re-parsing the output and comparing the two ASTs yields
      equality — no wire type is lost — following the round-trip subtest in
      `internal/formatter/formatter_test.go`
- [ ] Inside an event block the wire type is emitted directly after `description` and ahead of `tags`,
      `source external` and `fields`; the existing relative order of those three is unchanged
- [ ] A translation's nested event emits its wire type on the same terms
- [ ] An event with no wire type, or one whose wire type is the empty string, emits no such line
- [ ] Formatting `internal/test.HotelReservation` and every existing formatter golden, canonical
      `*FormattedEmod` constant and transcribed name list produces byte-identical output to before
      this task — no existing expected value moves, since this task adds no line to any existing
      fixture
- [ ] Formatting is idempotent for a model carrying wire types
- [ ] A wire type containing a backslash, a tab, a quote, a `%` and non-ASCII characters survives a
      second `Format` of the output byte-identically, so the value is written through `quoted()` and
      never through `%q`
- [ ] `RunFmt` in `--check` mode reports no change needed for a canonically formatted file that uses
      wire types, and the canonical constant it is compared against opens with the `emod 1` header and
      is written as canonical output rather than as the input fixture re-indented
- [ ] The round-trip group gains its assertion inside the existing per-fixture leaf rather than a
      parallel table, and pairs the `Declared…` getter with the non-empty transcribed list so the
      assertion cannot pass against a writer that emits nothing

**Affected Files/Modules:**
- `internal/formatter/formatter.go` — emission in `writeEvent` (`:269-283`)
- `internal/formatter/formatter_test.go` — per-event output, round-trip and idempotency over the
  wire-type fixture, the hazard-character table
- `internal/cli/fmt_test.go` — a `--check` fixture and its canonical constant

**Patterns to Follow:**
- `writeDescription` (`internal/formatter/formatter.go:75-77`) is the `quotedLineIfSet` shape, and
  `quoted()` (`:61-63`) is why emod source is never written with `%q`
  (`tasks/learnings.md:46-49`)
- A new block entry goes after `description` and ahead of the nested blocks, in every writer
  (`tasks/learnings.md:156-159`) — the formatter is the writer that hurts to forget, and a
  parse → format → reparse comparison is the only guard that notices
- A fmt golden is never the input re-indented; pin a canonical constant and pass *that* to the
  settle helper (`tasks/learnings.md:141-144`)
- Every formatter and `RunFmt` expected string opens with the version header line
  (`tasks/learnings.md:31-34`)
- Fold the new per-fixture assertion into the existing round-trip leaf, and never pair a `Declared…`
  getter with a fixture whose transcribed list is empty (`tasks/learnings.md:256-259`)
- US-014 owns the rest of the canonical order — implement the wire type's placement only
- Caller pattern **Exported API** for `formatter.Format`, **Inbound** for `RunFmt`

**Testable:** Yes — `formatter.Format` and `cli.RunFmt` are exported.

**Verification:** `mise exec -- task test:unit` and `mise exec -- task test:integration` pass; the
viewer export fixtures in `e2e-viewer/tests/helpers.js` still match.

**Depends on:** Task 2

---

### Task 4: Reject two events sharing one wire type

**Behavior:** `emod validate` reports an error when two events in a model bind the same wire type,
naming both events and the value. Events with distinct wire types, and events with none, are
unaffected.

**Acceptance Criteria:**
- [ ] Two events in one model stating the same wire type produce one error, positioned at the *second*
      event's wire-type value, naming the repeated value, the event reporting it and the event that
      stated it first
- [ ] The check spans the whole model: the pair collides when the two events sit in the same slice, in
      two aggregates of one context, in two different contexts, and when one of them is a
      translation's nested event
- [ ] Three events sharing one wire type produce two errors — the second and third occurrences — each
      naming the first declarer
- [ ] Distinct wire types produce nothing; a model in which no event states a wire type produces
      nothing
- [ ] Two wire types differing only in letter case do not collide — the value is opaque and compared
      verbatim
- [ ] An event whose wire type is the empty string is treated as stating none: it never collides, and
      two events both writing an empty wire type produce nothing
- [ ] The diagnostics come back in declaration order across both slice homes, asserted as one
      `require.Equal` over the whole list of formatted lines rather than a length plus a first-element
      check
- [ ] The message is asserted as a complete formatted line, so an assertion naming the value cannot be
      satisfied by a message that dropped one of the two event names
- [ ] The diagnostic carries no `RuleName`: it is a hard error no configuration can silence, and
      `emod lint --explain` describes nothing by that name
- [ ] `cli.RunValidate` on a file with a collision returns a non-nil error whose text names both
      events and the repeated wire type, not merely a position and a count
- [ ] `internal/test.HotelReservation`, the wire-type fixture from Task 2, every file under
      `examples/` the repository ships as valid, and every fixture under `internal/parser/testdata/`
      still return zero diagnostics from `oracle.Check`

**Affected Files/Modules:**
- `internal/validator/validator.go` — a wire-type uniqueness check and its wiring in `Validate`
  (`:15-41`, alongside `:33`)
- `internal/validator/validator_test.go` — the collision shapes, the ordering, the non-collisions
- `internal/cli/validate_test.go` — the CLI leaf asserting the distinguishing message text

**Patterns to Follow:**
- `redeclaredInvariantDiagnostics` (`internal/validator/validator.go:266-273`) with
  `invariantScope.redeclarations()` (`:232-244`) is the duplicate-name template: a `declared` map
  flagging the second and later occurrence, the position of the offending declaration, and
  `errorAt` (`:150-158`) as the constructor. The difference is scope — invariants resolve per
  aggregate or context, wire types across the whole model — so this check wants one flat walk, not
  `invariantScopes` (`:216-230`)
- `modelIndex.collect` (`internal/validator/validator.go:83-122`) shows which collections hold events:
  `slice.Events` and each translation's nested event, over `model.AllSlices()`, which visits both slice
  homes. The index itself keys by name and cannot answer this question — a duplicate would overwrite
  rather than collide (`tasks/learnings.md:171-174`)
- Findings gathered from more than one AST collection are position-sorted before emitting:
  `sortInDeclarationOrder` (`internal/validator/validator.go:193-201`), applied by both existing
  multi-collection checks at `:292` and `:354` (`tasks/learnings.md:181-184`)
- A hard error carries no `RuleName` (`tasks/learnings.md:166-169`)
- The assertion shape is one `require.Equal` against the list of formatted diagnostic lines, which
  pins order and count together — `internal/validator/validator_test.go:2474-2477` is the sibling to
  copy, and `reportedLines` is the helper (`tasks/learnings.md:476-479`)
- CLI diagnostic tests assert the tokens that identify *this* diagnostic, matching the parser-level
  test one layer down (`tasks/learnings.md:6-10`)
- Caller pattern **Exported API** for `validator.Validate` and **Inbound** for `cli.RunValidate`

**Testable:** Yes — `validator.Validate`, `oracle.Check` and `cli.RunValidate` are exported.

**Verification:** `mise exec -- task test:unit` and `mise exec -- task test:integration` pass,
including the leaves that walk `examples/` and `internal/parser/testdata/` asserting the models the
repository ships as valid still validate.

**Depends on:** Task 2

---

### Task 5: Carry wire types through the JSON and CUE exports and the embedded schema

**Behavior:** `emod export -f json` and `emod export -f cue` both emit an event's wire type, the
embedded CUE schema declares the key as optional on `#Event`, and a model with no wire types exports
exactly as it does today.

The two formats are one task because an existing subtest decodes the JSON and CUE exports of one
fixture and requires the two documents to be equal: teaching one format about wire types while the
other stays ignorant breaks that parity the moment the shared export fixture gains one.

**Acceptance Criteria:**
- [ ] `export.ExportJSON` and `export.ExportCUE` each emit the wire type for every event that states
      one, including a translation's nested event
- [ ] Both wires spell the key after the DSL keyword — the same spelling the language uses — rather
      than after the Go field name
- [ ] The key is absent, not empty-valued, for an event with no wire type, so `ExportJSON` and
      `ExportCUE` on `internal/test.HotelReservation` both produce byte-identical output to before
      this task
- [ ] The "CUE and JSON exports describe the same model" subtest still passes with the shared export
      fixture extended to carry wire types
- [ ] An `emittedKeyOrder` assertion pins the whole list of keys an event object emits, alongside the
      sibling object whose key order it should match; the universal prefix every `json*` document
      shares — name, description, positions, comments — is unchanged
- [ ] `internal/cue/schema.cue` declares the key as an optional string on `#Event`, and the CUE
      writer's emission order matches the schema's field order, which is not the JSON struct's
- [ ] `cue vet -d '#Model'` accepts a model document carrying a wire type — the full-model constant in
      `internal/cue/embed_test.go` is extended, so a dropped or misnamed key fails the existing
      acceptance subtest — and rejects one whose wire type is not a string
- [ ] A negative leaf re-keys the value under a spelling the writer never emits and requires `cue vet`
      to fail naming that key, so the wire vocabulary cannot silently split from the language
- [ ] The CUE export of a model carrying wire types still conforms to `#Model`
- [ ] `emod schema -f cue` prints a schema containing the new optional key, and the schema still
      imports nothing
- [ ] `cli.RunExport` called with the `json` format on a file using wire types prints them inside the
      `model` object of the `{diagnostics, model}` envelope, and the envelope shape is otherwise
      unchanged; the format is passed to the function directly rather than as an argument written
      after the file path
- [ ] A wire type containing quotes, a backslash and non-ASCII characters round-trips its exact text
      through both formats
- [ ] `ExportDiagramJSON` output is unchanged for both a model carrying wire types and one without,
      and the existing whole-document walk still finds the key nowhere
- [ ] `glossary.RenderMarkdown` and `glossary.RenderJSON` produce byte-identical output for a model
      carrying wire types and the same model without them — the glossary deliberately does not show
      them

**Affected Files/Modules:**
- `internal/export/json.go` — a field on `jsonEvent` (`:99-111`) and its copy in `convertEvent`
  (`:498-515`)
- `internal/export/cue.go` — the emission in `writeEvent` (`:168-175`)
- `internal/cue/schema.cue` — the optional key on `#Event` (`:30-37`)
- `internal/cue/embed_test.go` — the extended full-model constant plus the non-string and re-keyed
  rejection cases
- `internal/export/export_test.go` — per-format serialization, omission when unset, key order, special
  characters, the shared export fixture extended, and the diagram-document and glossary receipts
- `internal/cli/export_test.go`, `internal/cli/schema_test.go` — the CLI leaves

**Patterns to Follow:**
- A new exported field lands in JSON, CUE and `schema.cue` in the same change; the diagram document is
  deliberately *not* coupled and its forked event type must stay forked
  (`tasks/learnings.md:56-59`)
- JSON and CUE order their keys differently — copy the ordering from a `json*` sibling, never from the
  schema (`tasks/learnings.md:146-149`) — and `emittedKeyOrder`
  (`internal/export/export_test.go:4760`) is what turns that from a convention into an assertion
  (`tasks/learnings.md:231-234`)
- A serialized key spells the DSL keyword while the Go field may name the concept, and the repo keeps
  three negative leaves per wire key to hold that honest (`tasks/learnings.md:296-299`)
- The two export guards cannot see a value neither writer emits, so read the decoded document back
  against the names an author wrote rather than relying on parity and `cue vet` alone
  (`tasks/learnings.md:161-164`); `objectsUnder`/`statedUnder` are the read-back walkers, visited in
  the writer's slice order (`:291-294`)
- Optional-key emission on the CUE side: the `source` and `external_name` lines in
  `internal/export/cue.go:168-175`; schema style: `internal/cue/schema.cue:30-37`
- The `cue` binary is optional in this environment and the suites skip when it is missing — do not add
  a hard dependency on it
- urfave/cli v2 stops flag parsing at the first positional argument, so an acceptance criterion or
  README example written as `emod export <file> -f cue` silently exercises the default format
  (`tasks/learnings.md:111-114`)
- Assertion style: decode into `map[string]any` and assert on the document, except where key order is
  the subject, which is read from the raw bytes
- Caller pattern **Exported API** for `export.*` and `cue.Schema`, **UI** for the CLI leaves

**Testable:** Yes — `export.ExportJSON`, `export.ExportCUE`, `export.ExportDiagramJSON`,
`cue.Schema`, `glossary.RenderMarkdown`, `cli.RunExport` and `cli.RunSchema` are all exported.

**Verification:** `mise exec -- task test:unit` and `mise exec -- task test:integration` pass,
including the `cue vet` conformance and JSON/CUE parity subtests when the `cue` binary is available.

**Depends on:** Task 2

---

### Task 6: Nudge wire types toward reverse-DNS kebab-case with `wire/type-format`

**Behavior:** An event whose wire type does not read as reverse-DNS kebab-case draws one info
diagnostic pointing at the CloudEvents convention. An event whose wire type conforms, and an event
with no wire type at all, draw nothing.

**Acceptance Criteria:**
- [ ] A wire type conforms when it is two or more dot-separated segments, every segment lowercase and
      built from letters `a`-`z`, digits and hyphens, no segment empty, and no segment opening or
      closing with a hyphen; `com.acme.reservations.room-reserved` conforms
- [ ] Each of these draws exactly one diagnostic, asserted as a table: a name with no dot at all; a
      name carrying an uppercase letter in any segment; a name using an underscore as a separator; a
      name with an empty segment from a leading, trailing or doubled dot; a segment opening or closing
      with a hyphen
- [ ] The diagnostic sits at the event's wire-type value, carries severity info and rule name
      `wire/type-format`, and its single message names the event, the offending value and the
      convention — one text, with no branch on which part of the value failed
- [ ] An event with no wire type produces nothing, and an event whose wire type is the empty string
      produces nothing
- [ ] The rule fires for a translation's nested event as well as for a slice event, in both slice
      homes, and fires once per event rather than once per naming rule that also matched — it is
      reachable independently of the event-naming checks that short-circuit each other
- [ ] `linter.RuleDescription` answers for `wire/type-format`, `emod lint --explain wire/type-format`
      prints that non-empty description and returns no error, and an unknown rule name still returns
      an error
- [ ] The hand-maintained rule list in `internal/cli/lint_test.go:627-645` names `wire/type-format`,
      so the new description is covered by the "all rules have descriptions" leaf
- [ ] A CLI lint fixture states one non-conforming wire type and trips this rule and no other,
      asserted with a length of exactly one entry, and its declaring comment names the rule it is
      written to fire
- [ ] `cli.RunLint` and `cli.RunValidate` both return an error for that fixture — an info diagnostic
      is still a diagnostic — and the error text names the rule and the offending value
- [ ] Every shared fixture in `internal/test/fixtures.go`, every file under `examples/` the repository
      ships as valid, every fixture under `internal/parser/testdata/`, and every ` ```emod ` fence in
      `README.md` and `docs/dsl-reference.md` still return zero diagnostics from `oracle.Check`
- [ ] The diagnostics for several non-conforming events come back in declaration order across both
      slice homes, asserted as one comparison over the whole list of formatted lines

**Affected Files/Modules:**
- `internal/linter/linter.go` — a per-event check and its two call sites in `checkSlice` (`:111-113`
  and `:118-120`)
- `internal/linter/descriptions.go` — the rule description (`:10-31`)
- `internal/linter/linter_test.go` — the conformance table, both slice homes, the translation-nested
  event, severity and rule name
- `internal/cli/lint_test.go` — the single-rule fixture, its leaves, and the rule-name list at
  `:627-645`

**Patterns to Follow:**
- `checkClickbaitEvent` (`internal/linter/linter.go:524-529`) is the structural template for a
  per-event rule: called from both event loops in `checkSlice` and deliberately kept outside
  `checkEvent` (`:393-404`), whose early returns would suppress it
- `info(pos, rule, msg)` (`internal/linter/linter.go:16-25`) is the whole severity story — there is no
  configuration file, no severity flag and no options parameter on `linter.Lint`
- `dcb/single-tag-everywhere` (`internal/linter/linter.go:234-266`) is the only other info rule; its
  suite (`internal/linter/linter_test.go:2226-2739`) is the shape for asserting severity alongside
  message and position
- Naming a rule obliges you to register its description (`tasks/learnings.md:166-169`); the coverage
  leaf that would otherwise miss it is a hardcoded list, not a reflection over a registry
- A lint fixture trips exactly one rule, so it is never the minimal model: `left-chair`, `god-view`,
  `view-naming`, `orphan-command`, `orphan-event`, `clickbait-event` and the `dcb/*` family are the
  tripwires, which is why a fixture gives its slices a full `flow` and its events real fields
  (`tasks/learnings.md:471-474`)
- A new rule sweeps every checked-in model before it lands, because any diagnostic fails
  `emod validate` (`tasks/learnings.md:466-469`). This rule can only fire on an event that states a
  wire type, and today no checked-in model states one — so the sweep is a receipt to run, not a set of
  files to move. The one model that must be checked is the fixture Task 2 added
- A rule pinned by whole formatted lines rather than by `require.Contains` on a fragment
  (`tasks/learnings.md:476-479`), and diagnostics gathered from more than one AST collection are
  position-sorted (`:181-184`)
- Caller pattern **Exported API** for `linter.Lint` and `linter.RuleDescription`, **Inbound** for
  `cli.RunLint` and `cli.RunLintExplain`

**Testable:** Yes — `linter.Lint`, `linter.RuleDescription`, `oracle.Check`, `cli.RunLint` and
`cli.RunLintExplain` are all exported.

**Verification:** `mise exec -- task test:unit` and `mise exec -- task test:integration` pass,
including the oracle leaves over the shared fixtures, the examples, the parser testdata and the
documented models.

**Depends on:** Task 2

---

### Task 7: Document the wire type in the DSL reference

**Behavior:** A reader of `docs/dsl-reference.md` learns that an event can bind a wire type, what the
value means, that two events may not share one, that the exports carry it, and that a lint rule
nudges toward the CloudEvents convention without enforcing it.

**Acceptance Criteria:**
- [ ] The reference documents the attribute with at least one worked `emod` example, states that the
      value is an opaque string, and states that the attribute is optional and that a model using none
      behaves exactly as before
- [ ] It states the uniqueness rule and its scope — two events anywhere in the model may not bind the
      same wire type — and names `emod validate` as what reports it
- [ ] It states that `emod export -f json` and `-f cue` carry the wire type, which is the point of the
      attribute, and that `emod glossary` and the diagrams deliberately do not show it
- [ ] It names `wire/type-format` and describes the convention it nudges toward, stating that the rule
      is informational and that a wire type not following the convention is still valid
- [ ] Every ` ```emod ` fenced block added to the reference passes `oracle.Check` with zero
      diagnostics, so every wire type written in an example is unique within its block and conforms to
      the convention — an illustrative fragment that is not a whole model carries a plain fence
      instead
- [ ] No `## <n>. Title` heading is added, removed or renumbered, so every `(#<n>-…)` link still
      points at the section it names; and no existing `###` heading is renamed, so the fourteen links
      citing sub-heading slugs still resolve
- [ ] `examples/*.emod`, `internal/parser/testdata/*.emod` and `internal/test/fixtures.go` are
      unmodified — rewriting the examples is US-018

**Affected Files/Modules:**
- `docs/dsl-reference.md` — the wire-type attribute, its uniqueness rule, and the consumer note

**Patterns to Follow:**
- `### External Source Events` (`docs/dsl-reference.md:493-505`) is the nearest precedent for an
  event-level attribute documented as a sub-heading, and section 10 "Descriptions" (`:567-633`) is the
  precedent for a short grammar sketch, a worked snippet, then a bulleted list of behavioural
  consequences — including the line at `:622` that states plainly what does *not* read the value
- A new `###` sub-heading changes no anchor; adding or reordering a numbered section renumbers every
  heading below it and breaks the links citing those numbers. After editing, reconcile both families:
  `^## [0-9]+\.` against `\(#[0-9]+-`, and `^### ` against `\(#[a-z]`
  (`tasks/learnings.md:36-39`, `:541-544`)
- An ```emod fence is a promise that the block validates, and `oracle.Check` runs the linter as well
  as the validator, so a fenced block owes a whole model: every command referenced defined, every
  event produced consumed, views named `…View`, no orphan — and now a conforming, unique wire type
  (`tasks/learnings.md:526-529`)
- `~/.claude/rules/markdown-docs.md`: the result must read as a first version — no narration of what
  the document used to say

**Testable:** No — prose documentation with no runtime behaviour of its own. Its snippets are
executable, though: the "documented models" leaf in `internal/oracle/oracle_test.go:112-129` extracts
and checks every fenced block, so a malformed or non-conforming example fails the suite.

**Verification:** `mise exec -- task test:unit` passes, including the documented-models leaf; the
numbered-heading and sub-heading link lists reconcile.

**Depends on:** Task 2, Task 4, Task 5, Task 6

---

## Summary

**Total tasks:** 7.

**Ordering rationale:** guard-first, then dependency, then consumer breadth. Task 1 comes first for a
reason particular to this story: `TestEditorKeywordCoverage` turns three editor files red the instant
`internal/lexer` learns a spelling they do not carry, and the coverage assertion runs in one direction
only — so teaching the surfaces ahead of the lexer is green at every step, while the reverse order has
no green intermediate state. Task 2 then establishes the language surface: nothing downstream can be
written until the AST carries the value, and the shared fixture it builds is the one model the
formatter, validator, export and lint tasks all assert against. Task 3 follows immediately because it
is the only task that prevents an active regression — `emod fmt` deleting the attribute — rather than
adding a feature. Tasks 4, 5 and 6 are the three consumer surfaces the story's criteria name; they
depend only on Task 2 and can run in any order or in parallel. JSON and CUE are one task because an
existing parity subtest requires both formats to describe the same document. Task 7 documents what the
previous six built, and depends on all of them because its fenced examples are executed by the same
pipeline they change.

**Story criteria coverage:**
- "An `event` accepts an optional `type "<string>"` attribute; the value is an opaque string" → Task 2,
  with Task 1 mirroring the syntax in the tree-sitter grammar and Task 3 keeping `emod fmt` from
  deleting it
- "`emod validate` errors when two events share the same wire type" → Task 4, scoped to the whole
  model and justified in the Open Questions above
- "`emod export -f json` and `-f cue` emit the wire type" → Task 5, together with the embedded
  `schema.cue` that the export must keep conforming to
- "`wire/type-format` (info) nudges toward reverse-DNS kebab-case without enforcing it" → Task 6, with
  the conforming shape stated precisely and the non-enforcement resolved as a severity label rather
  than an exit code
- "Events without a wire type validate and export exactly as before" → carried as an explicit
  unchanged-output criterion in Tasks 1, 3, 4, 5 and 6, anchored on `internal/test.HotelReservation`
  and `internal/test.DescribedHotelReservation` gaining no wire type and on no existing golden,
  canonical constant or transcribed list being edited

**Beyond the story's criteria:** Task 1 (editor keyword surfaces), the LSP hover entry inside Task 2,
Task 3 (formatter) and Task 7 (reference documentation). The first three are forced — by
`TestEditorKeywordCoverage`, by `TestKeywordCoverage/hover`, and by the formatter rendering only what
it knows — and each is held to the minimum that keeps the tree green: no completion list, no scope for
any other keyword, and only the wire type's placement rather than US-014's wider ordering rule. Task 7
follows the precedent set when US-001 and US-002 each documented the construct they introduced.

**Deferred, with the story that owns them:** `emod glossary` showing wire types (deferred by the story
file's own Open Questions, until a consumer asks); the rest of US-014's canonical formatting order;
LSP completion and go-to-definition over the new keyword (US-015); syntax highlighting as a feature,
including any scope for the other keywords US-017 names (US-017); wire types in `examples/*.emod`
(US-018). Untouched, with a receipt rather than an assumption: the diagram document, the importer and
the web viewer — whose forked event type exists precisely to keep a new AST field out of the
node-and-edge contract — the draw.io, SVG, mermaid and ascii renderers, and `emod glossary`. One
pre-existing defect is noted and left alone: `internal/lsp/diagnostics.go` maps `diagnostic.Info`
through its default arm onto the LSP error severity, so an info rule renders as a red squiggle in the
editor. It already affects `dcb/single-tag-everywhere`, fixing it means declaring an LSP
informational severity and changing how an existing rule renders, and that belongs in a change with
its own receipt rather than inside this story.

# US-004: Drop the trigger kind slot

## Progress
- [x] Task 1: Parse a trigger whose quoted name follows the keyword directly
- [x] Task 2: Accept the kindless trigger in the tree-sitter grammar
- [x] Task 3: Emit the kindless form from `emod fmt` and migrate every emod source in the repository
- [x] Task 4: Drop the kind from the Mermaid timeframe and the ASCII trigger label
- [ ] Task 5: Drop `kind` and `kind_position` from the JSON, CUE and embedded schema exports
- [x] Task 6: Drop `kind` from the diagram node, the importer and the viewer's details panel
- [x] Task 7: Reject `trigger <Kind> "<name>"` with a message naming its replacement
- [x] Task 8: Remove the kind from the tree-sitter grammar and migrate the corpus
- [ ] Task 9: Delete `ast.Trigger.Kind` and `KindPos`

---

## Story Reference

`user-stories/triggers-and-automations.md` → **US-004: Drop the trigger kind slot** (fourth of eleven
stories in "Triggers and Automations"; the story file marks it as shipping together with US-003, which
has landed). Design notes: `docs/proposals/triggers-and-automations-proposal.md` — section 1 (`:52-71`)
for the block shape and why the slot is removed rather than validated down to one value, section 4
(`:111-121`) for the replacement table, `:200` for the formatter, `:233` for `schema.cue`, `:237` for
the JSON export, `:246-247` for the importer, `:256-259` for the tree-sitter grammar, `:311` for the
Mermaid timeframe collapse, `:424-432` for why the version header stays at 1.

**In scope:** `trigger "<name>" { ... }` as the only accepted spelling of a slice-level trigger; a parse
error for each retired spelling that names its replacement; `emod fmt` emitting the kindless header;
the removal of the kind from the JSON export, the CUE export, the embedded `internal/cue/schema.cue`,
the diagram document node, the importer that reads that node back, and the viewer's details panel; the
Mermaid `ui`/`pcr` branch and the ASCII `<<Kind: Name>>` label, the only two renderers that ever showed
the value; the tree-sitter grammar and its corpus; every `.emod` file and every emod source embedded in
a Go, JS or corpus fixture; and the deletion of `ast.Trigger.Kind`/`KindPos` once nothing reads them.

**Out of scope:** the `reads` edge from a view to a trigger or automation (US-005); lane placement, so a
trigger box stays exactly where it sits today (US-006); the palette, so the trigger fill stays `#ffffff`
in SVG/draw.io and `#e1d5e7` in the viewer (US-007); the `automation/missing-todo-list` lint rule
(US-008); LSP hover, completion, go-to-definition and find-references, including the wording of
`keywordDescriptions` at `internal/lsp/hover.go:23` (US-009); the VS Code TextMate grammar
(`editors/vscode/syntaxes/emod.tmLanguage.json`) and `editors/tree-sitter-emod/queries/*.scm` (US-010);
`docs/dsl-reference.md`, `README.md` and the narrative prose of `examples/` (US-011). Also out of scope:
the `wireframe` entry the proposal's section 5 adds to a trigger, any validation of a trigger's `actor`
or `reads`, and a DSL version bump — `ast.SupportedVersion` stays at 1 and a retired spelling fails as
a parse error, not a versioning message.

**The `.emod`/prose boundary, resolved.** US-011 owns documentation, but this story's own criterion
demands that every example and fixture use the new form and pass `emod validate`. The split is by file
type, not by directory: every `.emod` file and every emod source embedded in a `.go`, `.js` or
`test/corpus/*.txt` fixture migrates here, because the parser stops accepting the old form in Task 7 and
those files are compiled or executed by the suite. Markdown snippets do not migrate. The consequence is
deliberate and must be handed to US-011: after Task 7, `README.md:33`, `docs/dsl-reference.md:582` and
five files under `docs/proposals/` show a trigger spelling `emod validate` rejects. `tasks/learnings.md`
records that US-002 left exactly this gap in the reference and that nothing in CI links a doc to the
language, so the list is written out in the Summary rather than left to be rediscovered.

**Consequences of that boundary, decided.** Eight shapes the story does not spell out:

1. *The order is accept → migrate → reject, not one breaking change.* `tasks/learnings.md` records the
   sequence from US-002: teach the parser the new spelling first, let `emod fmt` and a source sweep move
   the tree, and only then reject the old spelling. Collapsed into one commit, every fixture, golden,
   canonical constant, example, corpus case and expected diagnostic has to move in the same change as
   the parser, and nothing isolates a regression. Seven `internal/test` fixtures, ten parser-test
   inputs, two `.emod` files, two `internal/cli/fmt_test.go` constants, two LSP fixtures and one
   Playwright helper depend on this ordering to stay green.
2. *The formatter change and the source migration are one task, not two.* `emod fmt` dropping the kind
   makes parse → format → reparse read back an empty `Trigger.Kind` for a source that still spells one,
   and that comparison (`internal/formatter/formatter_test.go:538`) compares the field. So the writer
   and every source it round-trips over move together, or the round-trip group goes red.
3. *A bare identifier after `trigger` is told apart from a quoted name by token kind alone.* A quoted
   name lexes as `lexer.String` and a kind as `lexer.Identifier` (`internal/lexer/tokenizer.go:202-207`
   returns `Identifier` for any word not in the `keywords` map — `UI`, `Schedule` and `Processor` are
   not keywords and never become any). No lookahead and no keyword additions are needed; the existing
   `p.check(lexer.String)` / `p.check(lexer.Identifier)` pair is the whole discrimination.
4. *Three retired spellings, two messages, and a general case.* The story names `UI`, `Schedule` and
   `Processor`, but the kind is free-form today — `trigger OnOrder "ERP" {}` parses, and the corpus and
   the LSP fixtures use `OnOrder` and `manual`. So `Schedule` and `Processor` get the message naming the
   automation-with-`every` replacement, and every other identifier, `UI` included, gets the message
   saying the word is dropped. An author who merely forgot the quotes around a name lands in the general
   case and gets a second diagnostic from the missing quoted name, which is the right pair of hints.
5. *Recovery skips the kind token and keeps parsing.* Rejecting the spelling by abandoning the trigger
   would cascade its body into the slice's entry loop. `tasks/learnings.md` records that what proves a
   retirement's drain stopped in the right place is not the diagnostic count but reading back the
   entries below the rejected token plus a non-zero `ClosePos.Line` on the trigger, its slice and its
   context — so `trigger UI "Reservation Form" { actor Guest reads AvailableRoomsView }` must report
   once and still yield a fully populated trigger.
6. *Every reader stops reading the kind before the parser stops accepting it.* Once the parser rejects
   the kinded form, `Trigger.Kind` is always empty, and the surfaces that interpolate it unconditionally
   would print a hole: `internal/diagram/ascii.go:114` renders `<<%s: %s>>` and would emit
   `<<: SubmitForm>>`, while `jsonTrigger.Kind` is tagged `json:"kind"` with no `omitempty` and would
   emit `"kind": ""`. Tasks 4, 5 and 6 therefore precede Task 7, so no commit ships a hole or an empty
   wire key.
7. *No new shared fixture, and no fixture kit.* This is a removal, so there is no optional field to
   exercise in both slice homes. All seven fixtures in `internal/test/fixtures.go` already declare a
   trigger with `actor` and `reads`, and once migrated they are the witnesses every downstream package
   needs. `triggerReadsByOwner` (`internal/export/export_test.go:4406`) and the importer's
   canonical-source round-trip already exist for the read-backs.
8. *The Mermaid `pcr` fixture becomes a plain trigger, not an automation.* `contract_test.go:938-941`
   declares `Kind: "Schedule", Name: "ShipTimer"` inside `fullModel()`, and `mermaid_test.go:275-276`
   pins `tf 09 pcr Orders.ShipTimer` for it. The proposal's replacement for a schedule trigger is an
   automation with `every`, but converting a render fixture would move node counts, edge sets, lane
   assignments and `fullModelLabels` across four exporters — scope no criterion in this story asks for.
   The trigger keeps its name and loses its kind, and its timeframe line becomes `ui`. Automations
   already emit `pcr` (`internal/diagram/mermaid_test.go:19-25`), so nothing else moves.

**Learnings folded in** from `tasks/learnings.md`: rename a DSL keyword by accepting both spellings
first and letting `emod fmt` migrate the tree; retiring a keyword needs its own parser arm, not the
fall-through, and three closing braces as the receipt; a new block entry keyword owes three things to
the parser's diagnostics, and the drain is `skipRestOfLineOrBlockEnd`; parser diagnostics at a stored
position go through `p.errorAtPosition`; assert a short keyword in a diagnostic with a `\b`-bounded
`require.Regexp` (`UI` is two characters and `on`/`every` recur in this story's messages); a second
`require.Contains` on one message is often shadowed by the first; put a new parser subtest in the group
that owns the construct; `emod fmt` canonicalises order, so a fmt golden is never the input re-indented,
and formatter output always begins with `emod N`; never write emod source with `%q`; a new exported
field must land in JSON, CUE and `schema.cue` in the same change and the two order their keys
differently — the mirror obligation on removal; `emittedKeyOrder` makes the JSON key order assertable
from the raw bytes; read a decoded export document back with `objectsUnder`/`statedUnder`; the two
export guards cannot see a list neither writer emits, so a removal needs a positive read-back and a
negative vet, not just agreement; a key rename owes a retired-key negative assertion on every surface
that reads the key; a diagram-node key has three readers and they must move in one commit; the viewer's
save path is `importer.ImportDiagram`, so a diagram-node field owes a read-back; `internal/viewer/static`
is a display surface with its own vitest harness; the tree-sitter grammar must never be stricter than
the Go parser, its generated `src/` stays gitignored, and every `grammar.js` rule carries a one-line
example of its full shape; run repo tooling through `mise exec --`; CLI diagnostic tests must assert the
distinguishing message text; an assertion whose expected value comes from the code under test cannot
fail; `docs/dsl-reference.md` is the one keyword surface no test reaches and a retirement story forgets
it; acceptance criteria describe the working tree, and a commit-message receipt is the commit author's
obligation, never a criterion.

---

## Codebase Context

**Lexer.** `internal/lexer/token.go` declares the `Kind` iota block at `:10-71` (`KeywordTrigger` at
`:21`) and the lowercase `keywords` map at `:73-112` (`"trigger"` at `:83`). `UI`, `Schedule` and
`Processor` appear nowhere in it; `keywordOrIdentifier` (`internal/lexer/tokenizer.go:202-207`) returns
`lexer.Identifier` for any word the map does not hold. This story adds no keyword and edits no lexer
file.

**AST.** `ast.Trigger` (`internal/ast/ast.go:173-187`) carries `Comments` (`:174`), `Kind`/`KindPos`
(`:175-176`), `Name`/`NamePos` (`:177-178`), `Description`/`DescriptionPos` (`:179-180`), `Actor`/
`ActorPos` (`:181-182`), `Reads`/`ReadsPos` (`:183-184`), `OpenPos` (`:185`) and `ClosePos` (`:186`).
`Slice.Trigger` (`:82`) is a single pointer, not a list. `ast.Automation` (`:202-220`) has no
`Kind` pair at all and is the shape a trigger converges on.

**Parser.** `parseTrigger` (`internal/parser/parser.go:551-618`) consumes the keyword (`:553`), requires
`lexer.Identifier` for the kind (`:554-556`, message `expected identifier after trigger`), records
`Kind`/`KindPos` (`:559-564`), then requires `lexer.String` for the name (`:566-568`, message
`expected quoted string after trigger kind`), then `{` (`:574-576`). Its entry loop (`:581-608`) is an
if/else-if chain — unlike `parseAutomationEntry` (`:1066`), which is a `switch` and holds the retired
`trigger` arm at `:1074-1077` (`trigger is not an automation entry: name the activation event with on`),
the shape this story's rejection copies. The unrecognised-entry message is at `:604-607`, the unclosed
brace at `:610-613`. `parseSlice` dispatches at `:393-396`. Helpers: `parseQuotedEntryInto` (`:1396`),
`parseIdentifierEntryInto` (`:1409`), `skipRestOfLineOrBlockEnd` (`:1503`), `error` (`:1522`), `errorAt`
(`:1526`), `errorAtPosition` (`:1530`), `position` (`:1421`).

**Parser tests.** `internal/parser/parser_test.go` is one umbrella with thirteen groups; the ones this
story touches are `"triggers"` (`:1493`), `"automations"` (`:1898`), `"error reporting"` (`:2595`),
`"descriptions"` (`:3694`) and `"comments"` (`:4409`). Ten inputs spell a kind — `:1499`, `:1525`,
`:1549`, `:1588`, `:1611`, `:2224`, `:3778`, `:4006`, `:4265`, `:4521` — and three assert its value
(`:1514`, `:1538`, `:2243`, all `require.Equal(t, "UI", …)`). Two subtest names are written around the
kind: `:1494` "trigger with kind, name, actor, and reads" and `:1520` "trigger with only kind and name
(empty body)". `internal/parser/integration_test.go:150` asserts the whole `ast.Trigger` for
`testdata/all_patterns.emod` through `test.RequireEqual`.

**Formatter.** `writeTrigger` (`internal/formatter/formatter.go:208-215`) emits the header at `:210` as
`trigger %s %s {` with the kind interpolated raw and the name through `quoted` (`:61-63`), then the
description, `actor` (`:212`) and `reads` (`:213`) through `lineIfSet` (`:25-30`). Called from
`writeSlice` (`:167`). Goldens: `internal/formatter/formatter_test.go:155` "formats trigger block" (AST
literal at `:168`, expected line at `:190`) and the slice-ordering subtest (literal `:1207`, needles
`:1289` and `:1307`); the model comparison hook is `cmpopts.IgnoreFields(ast.Trigger{}, "Comments")` at
`:3750`; the round-trip group is at `:538`. `internal/cli/fmt_test.go` holds two canonical constants
carrying a trigger — `keywordFieldFormattedEmod` (`:135`, trigger at `:144`) and `specFormattedEmod`
(`:243`, trigger at `:253`) — fed to `requireFmtSettlesOn`.

**Model exports.** `jsonTrigger` (`internal/export/export.go:132-146`) opens with `comments`, then
`kind` (`:134`, tagged `json:"kind"` with **no** `omitempty`), `kind_position` (`:135`), `name`,
`description`, `position`, `actor`/`actor_position`, `reads`/`reads_position`, `open_position`,
`close_position` — an interleaved value/position order unlike `jsonAutomation`'s positions-then-values
block. `convertTrigger` (`:556-573`) fills it (`:562-563`). `cueWriter.writeTrigger` (`:1329-1336`)
writes `kind` first at `:1331`. `internal/cue/schema.cue` `#Trigger` (`:15-22`) declares `kind: string`
at `:17` as a **required** key, referenced by `#Slice` at `:92`. Test surfaces:
`internal/export/export_test.go:129` (`tr["kind"]` at `:164`), `:989` (`kind_position` column at
`:1030`), `:3342` (whole-object CUE equality at `:3365-3370`), plus literals at `:602`, `:1673`,
`:2012`, `:2120`, `:2561`, `:3889`. The read-back walkers are `statedUnder` (`:4361`), `objectsUnder`
(`:4371`), `triggerReadsByOwner` (`:4406`) and `emittedKeyOrder` (`:4461`, whose only call site is
`:1267`, an automation — there is no key-order pin for a trigger today).
`internal/cue/embed_test.go` holds `fullModelJSON` (`:148`) whose trigger object is `:171-177` (`kind`
at `:172`) and six negative-vet leaves at `:74`, `:83`, `:92`, `:101`, `:112` and `:123`.

**Diagram document, importer and viewer.** `jsonDiagramNode` (`internal/export/export.go:657-677`)
carries `Kind` at `:665` tagged `json:"kind,omitempty"`; the trigger node is built at `:797-810`
(`Kind` at `:807`) and is the **only** writer of that key. `trigger_command` edges (`:1035-1052`) key on
`triggerIDs[s.Trigger.Name]`, never on the kind. `internal/importer/importer.go` reverses it:
`defaultTriggerKind = "UI"` at `:15-17` exists so a kindless node does not format as `trigger  "name"`,
`diagramNode.Kind` is at `:30`, and `buildSlice`'s trigger branch at `:161-174` applies the fallback at
`:163-166`. `foldEdges` (`:254`) has no trigger case. `internal/importer/importer_test.go:478` is the
subtest "a trigger with no kind falls back to a parseable one" (assertion at `:491`); `:76` is the
round-trip over `../parser/testdata/all_patterns.emod`; `automationNodeKeying` (`:59`) is the
two-keyings closure shape. `internal/viewer/static/ui.js:331-339` renders the trigger section of the
details panel with rows Kind (`:335`), Actor (`:336`) and Reads (`:337`); `renderer.js` never reads the
kind. `internal/viewer/tests/detail-panel.test.js` covers automation nodes only — **no trigger node is
tested at all today**. `web/static/` is a build-time copy of `internal/viewer/static/` produced by
`Taskfile.yml:44-52` and gitignored (`.gitignore:9`), so there is one source of truth.

**Go diagram renderers.** Only two read the kind. `internal/diagram/mermaid.go:80-85`, `:174-179` and
`:307-312` each select the timeframe element type `pcr` when the kind is `Schedule` or `Processor` and
`ui` otherwise; automations already emit `pcr` independently. `internal/diagram/ascii.go:114` builds the
label `<<%s: %s` from kind and name, with the actor appended at `:115-117`, the doc comment at `:16` and
the call site at `:52-53`. `svg.go`, `drawio.go`, `labels.go` and `flows.go` contain no `Kind`
reference: the trigger box's fill, stroke, lane and label are all kind-independent
(`svg.go:74-87`, `drawio.go:307-315`). Tests: `mermaid_test.go:54-62`, `:64-72`, `:74-82`, `:265-266`,
`:275-276`, `:397-412`; `ascii_test.go:48-71`, `:77-84`, `:152`/`:171`, `:335-336`; and the AST literals
at `contract_test.go:523`, `:713-718`, `:900-904`, `:938-941`, `svg_test.go:42`/`:74`,
`drawio_test.go:55`/`:94`/`:421`.

**Tree-sitter.** `trigger_definition` (`editors/tree-sitter-emod/grammar.js:169-178`) is
`seq('trigger', $.identifier, $.string, buildDescribedBlock(...))` — the kind is a bare positional
`$.identifier` (`:171`), not a named field — under the one-line example comment at `:168`.
`identifier` is `/[A-Z][a-zA-Z0-9_]*/` (`:253`), so the lowercase `manual` the LSP fixtures use has
never parsed in the grammar. Three corpus cases carry a kind: `test/corpus/slice.txt:97` ("Slice with
trigger", source `:103`, tree `:117-119`), `slice.txt:498` ("Slice with trigger containing actor and
reads", source `:504`, tree `:521-525`) and `description.txt:214` ("Trigger with description", source
`:220`, tree `:237-242`). `fields.txt:18` uses `trigger` as a field name and is unaffected.
`queries/highlights.scm:60` is `(trigger_definition (identifier) @function)` — US-010's to retire.
`src/` is gitignored and `task test:grammar` regenerates before running.

**Other emod sources.** `internal/test/fixtures.go` spells a kinded trigger at `:21`, `:115`, `:216`,
`:320`, `:429`, `:586` and `:754` — one per fixture, every one with an `actor` and a `reads`.
`e2e-viewer/tests/helpers.js:13` carries one inside `SAMPLE`, whose header comment (`:3-4`) states the
constant must stay byte-identical to canonical `emod fmt` output because the export spec asserts it.
`internal/lsp/definition_test.go:37` and `internal/lsp/references_test.go:34` spell
`trigger manual "MyTrigger"`, and `references_test.go:196` and `:208` embed that line verbatim inside
`posIn` needles. The two `.emod` files with a trigger are `examples/all_patterns.emod:11` and
`internal/parser/testdata/all_patterns.emod:11`; the other five `.emod` files have none. No Go test
reads anything under `examples/`.

**Not touched, deliberately.** `internal/validator` (nothing in it reads a trigger; the comment at
`validator.go:316-317` records that a trigger's `reads` stays unresolved on purpose), `internal/linter`
(no rule references `ast.Trigger`), `internal/glossary` (`triggerActorNames`,
`internal/glossary/glossary.go:123-134`, reads only the actor — its test literals carry a kind and move
with the field deletion), `internal/cli/slices.go` (`detectPattern:139` tests only for a trigger's
presence; no listing row names a trigger), `internal/lsp` non-test files (US-009),
`editors/vscode` and `editors/tree-sitter-emod/queries` (US-010), `docs/`, `README.md`, `e2e/`.

---

## Tasks

### Task 1: Parse a trigger whose quoted name follows the keyword directly

**Behavior:** `trigger "<name>" { ... }` parses, with the name token immediately after the keyword and
no kind, producing a trigger that records the name, its position, and whatever `description`, `actor`
and `reads` the block declares. The kinded spelling still parses unchanged — this task widens the
grammar and removes nothing. A trigger stating no kind also formats without a gap where the kind used
to be, so the new spelling survives `emod fmt` from the first commit.

**Acceptance Criteria:**
- [ ] `trigger "Reservation Form" { actor Guest reads AvailableRoomsView }` inside a slice parses with
      no diagnostics and yields a trigger carrying that name, actor and reads, with the filename, line
      and column of the quoted name token recorded as the trigger's name position
- [ ] The same source with a `description` entry parses with no diagnostics and carries the description
- [ ] `trigger "Reservation Form" {}` — an empty body — parses with no diagnostics
- [ ] Every existing subtest in `internal/parser/parser_test.go` that feeds a kinded trigger passes
      unedited, including the three that assert the kind value (`:1514`, `:1538`, `:2243`)
- [ ] `trigger` followed by neither a quoted name nor an identifier — an opening brace, say — reports
      exactly one diagnostic (`require.Len(t, diags, 1)`) whose message asks for a quoted name after
      the keyword, positioned at the offending token
- [ ] Formatting a trigger that states no kind emits its header with the quoted name directly after the
      keyword and a single space between them, and formatting a trigger that states a kind emits
      exactly the bytes it emits today — both asserted against one expected whole-block output
- [ ] Parsing a kindless trigger, formatting the model and reparsing the result yields a model equal to
      the original under the round-trip comparison at `internal/formatter/formatter_test.go:538`
- [ ] `git diff` moves no byte of any existing expected constant in
      `internal/formatter/formatter_test.go` or `internal/cli/fmt_test.go`
- [ ] `oracle.Check` over every fixture in `internal/test/fixtures.go` returns no diagnostics, with
      those fixtures unedited

**Affected Files/Modules:**
- `internal/parser/parser.go` — `parseTrigger` (`:551-618`), specifically the kind/name discrimination
  at `:554-568`
- `internal/parser/parser_test.go` — the `"triggers"` group (`:1493`)
- `internal/formatter/formatter.go` — `writeTrigger` (`:208-215`), the header line at `:210`
- `internal/formatter/formatter_test.go` — the trigger golden at `:155`

**Patterns to Follow:**
- The token-kind discrimination already exists in the function being edited: `p.check(lexer.String)` at
  `internal/parser/parser.go:566` and `p.check(lexer.Identifier)` at `:554`. No lookahead helper and no
  lexer change is required — `internal/lexer/tokenizer.go:202-207` is why a kind can only ever be
  `lexer.Identifier`
- `tasks/learnings.md` "Rename a DSL keyword by accepting both spellings first and letting `emod fmt`
  migrate the tree" — this task is that pattern's first step, and the kinded form must keep working
- `tasks/learnings.md` "Put a new parser subtest in the group that owns the construct" — the construct
  is the trigger, group `"triggers"` at `:1493`
- `tasks/learnings.md` "Formatter output always begins with `emod N`" and "`emod fmt` canonicalises
  order, so a fmt golden is never the input re-indented"
- `lineIfSet` (`internal/formatter/formatter.go:25-30`) is the existing shape for a part that is written
  only when the value is present; `quoted` (`:61-63`) is the only correct way to emit emod text

**Testable:** Yes — through `parser.Parse`, `formatter.Format` and `oracle.Check`.

**Verification:** `mise exec -- go test -tags unit ./internal/parser/... ./internal/formatter/...
./internal/oracle/...`; `mise exec -- go build ./...`.

**Depends on:** None

---

### Task 2: Accept the kindless trigger in the tree-sitter grammar

**Behavior:** the tree-sitter grammar parses `trigger "<name>" { ... }` without an `ERROR` node, so a
file `emod validate` accepts is not red-squiggled in an editor using the grammar. Everything the grammar
accepted before is still accepted, the kinded spelling included.

**Acceptance Criteria:**
- [ ] `trigger_definition` (`editors/tree-sitter-emod/grammar.js:169-178`) admits a trigger whose quoted
      name follows the keyword directly, while still admitting one that names a kind first
- [ ] The one-line comment above the rule (`:168`) spells the construct out whole in the form this story
      makes canonical, so the file's only description of a trigger matches what an author should write
- [ ] A corpus case in `editors/tree-sitter-emod/test/corpus/slice.txt` covers a kindless trigger with
      `actor` and `reads`, and its expected tree contains no `ERROR` or `MISSING` node
- [ ] A second corpus case covers a kindless trigger with a `description`, matching the described-trigger
      case that already exists at `test/corpus/description.txt:214`
- [ ] The three existing kinded corpus cases (`slice.txt:97`, `slice.txt:498`, `description.txt:214`)
      pass with their expected trees unedited
- [ ] `mise exec -- task test:grammar` passes, run through `mise exec --` so the repo-pinned tree-sitter
      CLI resolves rather than whichever one is on `PATH`
- [ ] `git check-ignore editors/tree-sitter-emod/src` succeeds, and the only files this task changes
      under `editors/tree-sitter-emod` are `grammar.js` and files under `test/corpus/`

**Affected Files/Modules:**
- `editors/tree-sitter-emod/grammar.js` — `trigger_definition` (`:168-178`)
- `editors/tree-sitter-emod/test/corpus/slice.txt` — beside `Slice with trigger containing actor and
  reads` (`:498`)
- `editors/tree-sitter-emod/test/corpus/description.txt` — beside `Trigger with description` (`:214`)

**Patterns to Follow:**
- `tasks/learnings.md` "The tree-sitter grammar must never be stricter than the Go parser" — this task
  exists so the window in which the grammar rejects what Task 1 accepts is one commit long. Note the
  learning's warning is about block *bodies*, which must stay `buildDescribedBlock`; the kind sits ahead
  of the block and is a positional part, so making it optional there is the faithful mirror of the Go
  parser, not a loosened block rule
- `tasks/learnings.md` "Every `grammar.js` rule carries a one-line example of its full shape" — the
  comment at `:168` is the only place the construct is written out and nothing tests it
- `tasks/learnings.md` "Run repo tooling through `mise exec --`, not bare PATH" and "Generated
  tree-sitter `src/` stays gitignored"
- Highlighting queries are US-010's; this task changes no `.scm` file, and
  `queries/highlights.scm:60` keeps matching for as long as the grammar still admits a kind

**Testable:** Yes — the tree-sitter corpus is the test surface, run by `task test:grammar`.

**Verification:** `mise exec -- task test:grammar`; `git status --porcelain editors/tree-sitter-emod`
lists only `grammar.js` and corpus files.

**Depends on:** 1

---

### Task 3: Emit the kindless form from `emod fmt` and migrate every emod source in the repository

**Behavior:** `emod fmt` writes every trigger's header as the keyword and the quoted name, dropping any
kind the source stated, so formatting is itself the migration path for a user's file. Every `.emod` file
and every emod source embedded in a Go, JS or test fixture in this repository is rewritten to that form,
and each still parses and validates. The kinded spelling is still accepted by the parser — nothing in
the tree spells it any more.

**Acceptance Criteria:**
- [ ] Formatting a model whose trigger carries a kind emits a header naming only the quoted name, with
      the description, `actor` and `reads` lines unchanged — asserted from an AST literal that still
      states a kind, so the drop is proved rather than assumed
- [ ] Formatting the same model twice produces identical bytes, and a trigger name containing a
      backslash, a tab, a quote and a `%` survives a second format run unchanged
- [ ] `emod validate` exits 0 for `examples/all_patterns.emod`, `internal/parser/testdata/
      all_patterns.emod`, `internal/parser/testdata/minimal.emod` and
      `internal/parser/testdata/multi_context.emod`
- [ ] `git diff` for each `.emod` file this task edits touches only its `trigger` line: nothing else in
      those files is re-ordered or re-indented, since `emod fmt` canonicalises entry order and a
      wholesale rewrite would move flows, specs and field columns this story has no business moving
- [ ] No `.go`, `.js` or `.emod` file outside `editors/` matches a trigger keyword followed by a bare
      word and then a quoted string — the tree-sitter grammar's example comment and its three corpus
      cases are Task 8's, and every other site moves here: the seven fixtures in
      `internal/test/fixtures.go` (`:21`, `:115`, `:216`, `:320`, `:429`,
      `:586`, `:754`), the ten inputs in `internal/parser/parser_test.go`, the two canonical constants
      in `internal/cli/fmt_test.go` (`:144`, `:253`), the goldens and needles in
      `internal/formatter/formatter_test.go` (`:190`, `:1289`, `:1307`), the two LSP fixtures
      (`internal/lsp/definition_test.go:37`, `internal/lsp/references_test.go:34`), the `SAMPLE`
      constant in `e2e-viewer/tests/helpers.js:13` and the two `.emod` files are all migrated
- [ ] The `posIn` needles at `internal/lsp/references_test.go:196` and `:208` embed the migrated trigger
      line verbatim, and the LSP find-references subtests still resolve the view named in that trigger's
      `reads` — a needle left spelling the old line would fail to locate a position at all
- [ ] `emod fmt --check` accepts the text of `SAMPLE` in `e2e-viewer/tests/helpers.js`, which its header
      comment requires to stay byte-identical to canonical formatter output
- [ ] No assertion on `Trigger.Kind` remains in `internal/parser/parser_test.go`, and the two subtests
      named after the kind (`:1494`, `:1520`) are renamed for the behaviour they now exercise
- [ ] `internal/parser/integration_test.go:150` compares the parsed trigger of
      `testdata/all_patterns.emod` against a literal that states no kind, and passes
- [ ] `oracle.Check` over every fixture in `internal/test/fixtures.go` returns no diagnostics after the
      migration, and the round-trip group at `internal/formatter/formatter_test.go:538` passes for every
      fixture — this is the assertion that would fail if the formatter change shipped without the
      sources, because the comparison reads `Trigger.Kind` back

**Affected Files/Modules:**
- `internal/formatter/formatter.go` — `writeTrigger` (`:208-215`), the header line at `:210`
- `internal/formatter/formatter_test.go` — the trigger golden (`:155`, expected line `:190`) and the
  slice-ordering needles (`:1289`, `:1307`)
- `internal/cli/fmt_test.go` — `keywordFieldFormattedEmod` (`:135`) and `specFormattedEmod` (`:243`)
- `internal/test/fixtures.go` — seven trigger lines
- `internal/parser/parser_test.go`, `internal/parser/integration_test.go`
- `internal/lsp/definition_test.go`, `internal/lsp/references_test.go`
- `examples/all_patterns.emod`, `internal/parser/testdata/all_patterns.emod`
- `e2e-viewer/tests/helpers.js`

**Patterns to Follow:**
- `tasks/learnings.md` "Rename a DSL keyword by accepting both spellings first and letting `emod fmt`
  migrate the tree" — this is that pattern's middle step, and the reason the tree is quiet when Task 7
  lands
- `tasks/learnings.md` "`emod fmt` canonicalises order, so a fmt golden is never the input re-indented"
  and "A new block entry goes after `description` and ahead of nested blocks, in every writer", whose
  warning that the formatter silently deletes what it has never heard of is here the intended behaviour
  for exactly one part
- `tasks/learnings.md` "Never write emod source with `%q` — the language has no escape sequences", with
  its counterpart obligation of a round-trip subtest per hazard character
- `tasks/learnings.md` "Formatter output always begins with `emod N`" — every canonical constant starts
  with the version header even when the input fixture omits it
- `README.md`, `docs/dsl-reference.md` and every file under `docs/` are US-011's and are not edited here

**Testable:** Yes — through `formatter.Format`, `cli.RunFmt`, `cli.RunValidate` and `oracle.Check`.

**Verification:** `mise exec -- go test -tags unit ./...`;
`mise exec -- go run ./cmd/emod validate examples/all_patterns.emod`;
`rg -n 'trigger[ \t]+[A-Za-z_][A-Za-z0-9_]*[ \t]+"' -g '*.go' -g '*.js' -g '*.emod' -g '!editors/**'`
returns nothing — the grammar's example comment at `editors/tree-sitter-emod/grammar.js:168` and the
three corpus cases are Task 8's.

**Depends on:** 1

---

### Task 4: Drop the kind from the Mermaid timeframe and the ASCII trigger label

**Behavior:** the two text renderers that showed a trigger's kind stop showing it. A Mermaid timeframe
line for a trigger names the `ui` element type whatever the model says, and the ASCII label for a
trigger names the trigger and its actor with no kind prefix and no empty prefix left behind. Automations
keep the `pcr` element type they already emit, and every other renderer is untouched.

**Acceptance Criteria:**
- [ ] The Mermaid output for a slice with a trigger names the `ui` element type for that trigger,
      asserted on one model whose triggers state `UI`, `Schedule` and `Processor` respectively, so all
      three produce the same element type and the branch is proved gone
- [ ] The Mermaid output for a model carrying both a trigger and an automation still names `pcr` for the
      automation, asserted in the same subtest as the trigger's `ui` line so the two are not confused
- [ ] The ASCII label for a trigger names the trigger and, when it states one, its actor, with no kind
      and no leading separator — asserted on a trigger that states a kind and on one that states none,
      both in the same output
- [ ] Every rune of the ASCII output for a model carrying a trigger is ASCII apart from `⚙` — the
      existing assertion at `internal/diagram/ascii_test.go:319` passes unedited
- [ ] `internal/diagram/ascii.go`'s package doc comment (`:16`) describes the label shape the renderer
      now produces
- [ ] The SVG and draw.io renderings of a model carrying a trigger are byte-identical to what they
      produce today: `git diff` moves no expected constant in `internal/diagram/svg_test.go` or
      `internal/diagram/drawio_test.go`, and the trigger box keeps its lane, position, size, fill and
      stroke — neither renderer ever read the kind
- [ ] No test in `internal/diagram` asserts a `pcr` element type for a trigger, and no test asserts a
      kind inside an ASCII label

**Affected Files/Modules:**
- `internal/diagram/mermaid.go` — the three element-type selections (`:80-85`, `:174-179`, `:307-312`)
- `internal/diagram/ascii.go` — the trigger label (`:114-118`) and the doc comment (`:16`)
- `internal/diagram/mermaid_test.go` — the three per-kind subtests (`:54-62`, `:64-72`, `:74-82`) and the
  `fullModel()` expectations (`:265-266`, `:275-276`, `:397-412`)
- `internal/diagram/ascii_test.go` — the kind subtests (`:48-71`, `:77-84`) and the `fullModel()`
  expectations (`:152`/`:171`, `:335-336`)

**Patterns to Follow:**
- `docs/proposals/triggers-and-automations-proposal.md:311` states the outcome: with the kinds gone,
  triggers always emit `ui` and the branch collapses, and automations keep `pcr` — which
  `internal/diagram/mermaid_test.go:19-25` shows they already do
- The three Mermaid sites are the same selection written three times; `tasks/learnings.md`
  "De-duplicate before a fan-out edit" applies only if they are collapsed into a helper, which a
  deletion does not require — each site loses a branch and gains nothing
- `tasks/learnings.md` "An assertion whose expected value comes from the code under test is the
  recurring review finding" — name the expected timeframe line and the expected label, do not rebuild
  either from the renderer
- `contract_test.go:938-941`'s `Kind: "Schedule"` literal stays a literal until Task 9; the decision
  recorded above is that `ShipTimer` remains a trigger and its timeframe line becomes `ui`
- Lane placement is US-006's and the palette is US-007's; this task moves no box and repaints nothing

**Testable:** Yes — through `diagram.ExportMermaid` and `diagram.ExportASCII`.

**Verification:** `mise exec -- go test -tags unit ./internal/diagram/...`.

**Depends on:** None

---

### Task 5: Drop `kind` and `kind_position` from the JSON, CUE and embedded schema exports

**Behavior:** `emod export` describes a trigger by its name, description, actor and reads in both
formats, and neither document carries a kind. The embedded `internal/cue/schema.cue` no longer declares
the key, so a document that still states one is rejected rather than quietly accepted. Every other key
a trigger emits keeps its value and its position in the object.

**Acceptance Criteria:**
- [ ] The JSON export of a model carrying a trigger contains no `kind` and no `kind_position` key under
      that trigger, while its `name`, `description`, `actor`, `reads` and every `*_position` key it
      emits today are all still present with their values — asserted by reading the trigger object back,
      not by searching the raw bytes for a string
- [ ] `emittedKeyOrder` shows the trigger object emitting its remaining keys in the same relative order
      as today, and the automation object's key list asserted in the same subtest is unchanged — the
      sibling is what makes the expectation non-arbitrary, and this story reorders nothing
- [ ] The CUE export of the same model carries no `kind` line under the trigger, and the "CUE and JSON
      exports describe the same model" subtest passes over a model carrying a trigger with an actor and
      a reads
- [ ] `internal/cue/schema.cue`'s `#Trigger` (`:15-22`) no longer declares `kind`, and the
      schema-conformance subtest passes for a model carrying a trigger
- [ ] The embedded-schema fixture `fullModelJSON` (`internal/cue/embed_test.go:148`) states no `kind`
      under its trigger and vets clean against `#Model`
- [ ] A copy of that fixture with a `kind` key restored under the trigger fails `cue vet`, and the
      failure names that key — proving `#Trigger` is closed and the retired spelling is rejected rather
      than ignored, in the shape of the negative leaf at `internal/cue/embed_test.go:112`
- [ ] `emod export` and `emod export -f cue` both still succeed for `examples/all_patterns.emod`, and
      the trigger appears in each output under its slice with its actor and its reads

**Affected Files/Modules:**
- `internal/export/export.go` — `jsonTrigger` (`:132-146`, the `kind`/`kind_position` fields at
  `:134-135`), `convertTrigger` (`:556-573`, `:562-563`), `cueWriter.writeTrigger` (`:1329-1336`,
  `:1331`)
- `internal/cue/schema.cue` — `#Trigger` (`:15-22`)
- `internal/export/export_test.go` — `:129`/`:164`, `:989`/`:1030`, `:3342`/`:3365-3370`
- `internal/cue/embed_test.go` — `fullModelJSON` (`:171-177`) and a negative-vet leaf beside `:112`

**Patterns to Follow:**
- `tasks/learnings.md` "A new exported field must land in JSON, CUE and `schema.cue` in the same change"
  — the removal carries the same obligation, and the two coupled subtests (export parity, schema
  conformance) are what fail if one surface is missed
- `tasks/learnings.md` "The two export guards cannot see a list neither writer emits" — a key neither
  writer emits makes the parity subtest agree trivially, so the proof that the removal is complete is
  the closed-definition vet failure, not the parity pass
- `tasks/learnings.md` "JSON key order is assertable from the raw bytes — `emittedKeyOrder` already
  exists"; note `jsonTrigger` interleaves values and positions where `jsonAutomation` blocks them, and
  this task preserves that difference rather than tidying it
- `tasks/learnings.md` "A key rename owes a retired-key negative assertion on every surface that reads
  the key" — for a removal the negative assertion is the vet failure above
- The diagram document is a separate surface with its own key and its own task; `jsonDiagramEvent`
  forks `jsonEvent` for exactly this reason, and the two documents stay unmerged

**Testable:** Yes — through `export.ExportJSON`, `export.ExportCUE` and `cue vet`.

**Verification:** `mise exec -- go test -tags unit ./internal/export/... ./internal/cue/...`.

**Depends on:** None

---

### Task 6: Drop `kind` from the diagram node, the importer and the viewer's details panel

**Behavior:** a trigger's node in the diagram document carries its name, actor and reads and nothing
else; the importer builds a trigger from those three and needs no fallback value to keep its output
parseable; and the viewer's details panel lists a trigger's actor and reads without a row for a value
nothing supplies. A model exported, edited in the viewer and re-imported keeps the trigger's name, actor
and reads.

**Acceptance Criteria:**
- [ ] The diagram document's trigger node states no `kind` key, while its type, parent, label, actor and
      reads are unchanged — asserted by reading the node back from the exported document
- [ ] Importing a document whose trigger node states a `kind` yields a trigger identical to the one
      imported from the same document without it, so the reader is proved to ignore the retired key
      rather than to have kept a second spelling alive
- [ ] Exporting a model that declares a trigger with an actor and a reads to a diagram document and
      importing it back yields a trigger carrying that name, actor and reads, and formatting the
      reimported model produces the same canonical bytes as formatting the original — this is the
      viewer's save path and the story's round-trip criterion
- [ ] The round-trip over `internal/parser/testdata/all_patterns.emod`
      (`internal/importer/importer_test.go:76`) reproduces the formatted source byte for byte
- [ ] Importing a trigger node produces a trigger that `emod fmt` writes as a header the parser accepts:
      no placeholder value stands in for the removed one, and the subtest at
      `internal/importer/importer_test.go:478` is replaced by one asserting the parseable output
      directly rather than the fallback that produced it
- [ ] The viewer's details panel for a trigger node shows its actor and its reads and no row for a kind,
      with the em-dash placeholder shown for an actor or reads the node does not state — asserted in a
      new trigger case in `internal/viewer/tests/detail-panel.test.js`, which covers only automation
      nodes today
- [ ] A trigger node whose data still carries a kind renders the same panel as one without it, so a
      stale document opened in the viewer shows no orphaned row
- [ ] An actor or reads value containing markup is shown as text, matching the escaping the automation
      rows already prove (`internal/viewer/tests/detail-panel.test.js:73-82`)
- [ ] `mise exec -- task test:viewer` and `mise exec -- task test:unit` both pass — the exporter, the
      importer and the panel are the three readers of this key and they move in this one task

**Affected Files/Modules:**
- `internal/export/export.go` — `jsonDiagramNode` (`:657-677`, `Kind` at `:665`) and the trigger node
  build (`:797-810`, `:807`)
- `internal/importer/importer.go` — `defaultTriggerKind` (`:15-17`), `diagramNode` (`:25-42`, `:30`) and
  the trigger branch of `buildSlice` (`:161-174`)
- `internal/viewer/static/ui.js` — the trigger section of the details panel (`:331-339`, the Kind row at
  `:335`)
- `internal/export/export_test.go` (`:1780`/`:1821`), `internal/importer/importer_test.go` (`:478-491`),
  `internal/viewer/tests/detail-panel.test.js`

**Patterns to Follow:**
- `tasks/learnings.md` "A diagram-node key has three readers, and they must move in one commit" — the
  exporter, the importer and `showDetailPanel`. US-002 split them and shipped a commit whose panel
  showed an em-dash for every automation with both suites green; a removal fails the same way, leaving
  a row that can never be filled
- `tasks/learnings.md` "The viewer's save path is `importer.ImportDiagram`, so a diagram-node field owes
  a read-back" — the guard is the canonical-source round-trip plus `importExported(t, model)`
- `automationNodeKeying` (`internal/importer/importer_test.go:59`) is the closure shape for importing
  one document under two keyings, which is how the ignored-retired-key criterion is written
- `tasks/learnings.md` "`internal/viewer/static` is a display surface with its own vitest harness" — the
  vitest tests are not part of `task test:unit`, and restructuring `showDetailPanel` beyond removing the
  row belongs in its own commit. `web/static/` is a build-time copy (`Taskfile.yml:44-52`) and is
  gitignored, so it is not edited
- The `trigger_command` edge (`internal/export/export.go:1035-1052`) keys on the trigger's name and is
  unchanged; the `reads` edge from a view to a trigger is US-005's and is not added here

**Testable:** Yes — through `export.ExportDiagramJSON`, `importer.ImportDiagram` and the vitest harness.

**Verification:** `mise exec -- go test -tags unit ./internal/export/... ./internal/importer/...`;
`mise exec -- task test:viewer`.

**Depends on:** 5

---

### Task 7: Reject `trigger <Kind> "<name>"` with a message naming its replacement

**Behavior:** a bare word between the `trigger` keyword and its quoted name is a parse error, and the
message tells the author what to write instead: for `Schedule` and `Processor`, an automation whose
activation is stated with `every`; for `UI` and any other word, that the word is simply dropped. The
parser recovers past the offending word, so the rest of the trigger, its slice and its context still
parse and one bad line yields one diagnostic.

**Acceptance Criteria:**
- [ ] `trigger UI "Reservation Form" { actor Guest reads AvailableRoomsView }` reports exactly one
      diagnostic (`require.Len(t, diags, 1)`) positioned at the line and column of the word `UI`, whose
      message names that word and says to drop it
- [ ] The same input still yields a trigger carrying the name, actor and reads it declared, and the
      trigger, its slice and its context all close (`require.NotZero` on each `ClosePos.Line`) — a
      recovery that drained the line or abandoned the block would leave those zero while the diagnostic
      count still looked right
- [ ] `trigger Schedule "Nightly Sweep" { reads PendingExpiries }` reports exactly one diagnostic naming
      `Schedule`, `automation` and `every`, with `every` asserted through a `\b`-bounded
      `require.Regexp` — it hides inside no word of the message, but `on` does and the same bounding is
      what keeps a later edit honest
- [ ] `trigger Processor "Hold Sweeper" { reads PendingExpiries }` reports the same replacement, naming
      `Processor`
- [ ] A trigger naming an arbitrary word — `trigger OnOrder "ERP" {}`, the spelling the grammar corpus
      used — reports the drop-the-word message, so the free-form kind the language actually accepted is
      covered and not only the three the story names
- [ ] Each of the three messages names its own offending word: no two of them are satisfied by the same
      assertion, and a needle asserted on one message is not a substring of a needle asserted above it
- [ ] `trigger "Reservation Form" { ... }` continues to parse with no diagnostics, asserted on a model
      that also carries a rejected trigger so the rejection is proved to be running
- [ ] A trigger whose kind is followed by no quoted name at all reports the retired-spelling diagnostic
      and then the missing-name diagnostic, and the following slice entry is still parsed — two
      diagnostics, not a cascade
- [ ] `cli.RunValidate` on a file spelling `trigger Schedule "..."` returns an error whose message names
      the offending word and the automation-with-`every` replacement — the same distinguishing content
      as the parser test one layer down, not just a path and a line number
- [ ] `mise exec -- go test -tags unit ./...` passes with no source in the repository migrated by this
      task: Task 3 already moved every one

**Affected Files/Modules:**
- `internal/parser/parser.go` — `parseTrigger` (`:551-618`), the kind arm at `:554-568`
- `internal/parser/parser_test.go` — the `"triggers"` group (`:1493`) for the accepted form and the
  `"error reporting"` group (`:2595`) for the messages and the recovery
- `internal/cli/validate_test.go` — one leaf beside the existing diagnostic leaves

**Patterns to Follow:**
- `tasks/learnings.md` "Retiring a keyword needs its own parser arm, not the fall-through, and three
  closing braces as the receipt" — the arm reports at the offending token with `p.errorAt` and the
  receipt is the entries below it plus three non-zero `ClosePos.Line` values. The precedent in this file
  is `parseAutomationEntry`'s retired `trigger` arm (`internal/parser/parser.go:1074-1077`)
- `skipRestOfLineOrBlockEnd` (`internal/parser/parser.go:1503`) is the drain for a *keyword entry* whose
  value is malformed; it is the wrong tool here, because the quoted name and the opening brace sit on
  the same line as the offending word and draining would swallow the whole block
- `tasks/learnings.md` "Assert a short keyword in a diagnostic with a `\b`-bounded `require.Regexp`" —
  `UI` is two characters and `on`/`every` recur in these messages
- `tasks/learnings.md` "A second `require.Contains` on one message is often shadowed by the first" —
  check each new needle is not inside one asserted above it
- `tasks/learnings.md` "CLI diagnostic tests must assert the distinguishing message text" and "New
  file-taking CLI commands compose `parseModelFile` and `reportExitError`"
- `tasks/learnings.md` "Put a new parser subtest in the group that owns the construct"
- The replacement wording comes from `docs/proposals/triggers-and-automations-proposal.md:111-121`

**Testable:** Yes — through `parser.Parse` and `cli.RunValidate`.

**Verification:** `mise exec -- go test -tags unit ./...`;
`mise exec -- go run ./cmd/emod validate examples/all_patterns.emod` exits 0.

**Depends on:** 3, 4, 5, 6

---

### Task 8: Remove the kind from the tree-sitter grammar and migrate the corpus

**Behavior:** the tree-sitter grammar admits exactly the trigger the Go parser admits — keyword, quoted
name, block — so an editor using the grammar builds the same tree an author's file describes and no
corpus case documents a spelling the language rejects.

**Acceptance Criteria:**
- [ ] `trigger_definition` (`editors/tree-sitter-emod/grammar.js:169-178`) admits only a quoted name
      after the keyword, and its one-line comment (`:168`) spells that shape
- [ ] The three corpus cases that spelled a kind (`test/corpus/slice.txt:97`, `slice.txt:498`,
      `description.txt:214`) are rewritten to the kindless form, expected trees included, and parse with
      no `ERROR` or `MISSING` node
- [ ] A corpus case covers a trigger written with a bare word before its quoted name and its expected
      tree contains an `ERROR` node, so the grammar is pinned as rejecting what the parser rejects
- [ ] The keyword-per-field corpus case (`test/corpus/fields.txt:1-40`), which uses `trigger` as a field
      name at `:18`, passes unedited
- [ ] `mise exec -- task test:grammar` passes, run through `mise exec --` so the repo-pinned tree-sitter
      CLI resolves
- [ ] `git check-ignore editors/tree-sitter-emod/src` succeeds, and the only files this task changes
      under `editors/tree-sitter-emod` are `grammar.js`, `queries/highlights.scm` and files under `test/corpus/`

**Affected Files/Modules:**
- `editors/tree-sitter-emod/grammar.js` — `trigger_definition` (`:168-178`)
- `editors/tree-sitter-emod/test/corpus/slice.txt` (`:97-119`, `:498-525`)
- `editors/tree-sitter-emod/test/corpus/description.txt` (`:214-242`)
- `editors/tree-sitter-emod/queries/highlights.scm` — delete the now-impossible `(trigger_definition (identifier) @function)` pattern at `:60`

**Patterns to Follow:**
- `tasks/learnings.md` "The tree-sitter grammar must never be stricter than the Go parser" — which is
  why this task follows Task 7 rather than preceding it: narrowing the grammar first would red-squiggle
  a file `emod validate` still accepts
- `tasks/learnings.md` "Every `grammar.js` rule carries a one-line example of its full shape" — nothing
  tests the comments, so only the diff catches a stale one
- `tasks/learnings.md` "Run repo tooling through `mise exec --`" and "Generated tree-sitter `src/` stays
  gitignored"
- **This task MUST delete `queries/highlights.scm:60`** — `(trigger_definition (identifier) @function)`.
  It does not "simply stop matching". Once `trigger_definition` no longer admits an `identifier`, that
  pattern is *impossible*, and tree-sitter refuses to compile the **whole query file**, taking every
  highlight in the language with it. Measured on a scratch copy with the kind removed from the rule:
  `tree-sitter query queries/highlights.scm probe.emod` exits 1 with
  `Query error at 60:21. Impossible pattern`, producing zero captures, against 24 captures and exit 0
  on the unmodified grammar. And it is not invisible to CI: injecting an impossible pattern into an
  otherwise-green tree flips `mise exec -- tree-sitter test` from exit 0 to exit 1, so **this task's own
  Verification command fails** until the line goes. Delete only `:60`; keep `:61`
  `(trigger_definition (string) @function)`, which highlights the quoted name US-010's criterion 2 wants.
- The VS Code TextMate rule at `editors/vscode/syntaxes/emod.tmLanguage.json:28-33` also assumes a kind
  identifier, but it is inert data — no build step compiles it — so it stays US-010's and this task
  changes no file under `editors/vscode`.

**Testable:** Yes — the tree-sitter corpus is the test surface, run by `task test:grammar`.

**Verification:** `mise exec -- task test:grammar`; `git status --porcelain editors/tree-sitter-emod`
lists only `grammar.js` and corpus files.

**Depends on:** 7

---

### Task 9: Delete `ast.Trigger.Kind` and `KindPos`

**Behavior:** the trigger node carries no kind, so no writer, renderer, exporter or future feature can
start reading a value the language no longer has. Nothing about the tool's output changes — the field
has had no producer since Task 7 and no reader since Task 6.

**Acceptance Criteria:**
- [ ] `ast.Trigger` (`internal/ast/ast.go:173-187`) declares neither `Kind` nor `KindPos`, and
      `mise exec -- go build ./...` succeeds
- [ ] No file in the repository names a trigger's kind field: the struct literals in
      `internal/parser/integration_test.go`, `internal/formatter/formatter_test.go`,
      `internal/glossary/glossary_test.go`, `internal/export/export_test.go`, `internal/importer/
      importer_test.go` and the five files under `internal/diagram` are all updated, and the compiler is
      what proves the list complete
- [ ] `mise exec -- go test -tags unit ./...` and `mise exec -- go test -tags integration ./...` pass
- [ ] `git diff` moves no expected constant, golden, canonical `*FormattedEmod` string or `.emod` file
      in this task: every literal edited is a field on an input, never a byte of expected output
- [ ] `emod fmt`, `emod validate`, `emod export`, `emod export -f cue`, `emod diagram` in each of its
      four formats and `emod slices` all produce for `examples/all_patterns.emod` exactly what they
      produced before this task

**Affected Files/Modules:**
- `internal/ast/ast.go` — `Trigger` (`:175-176`)
- `internal/parser/integration_test.go` (`:150`), `internal/formatter/formatter_test.go` (`:168`,
  `:1207`), `internal/glossary/glossary_test.go` (`:185`, `:186`, `:194`, `:235`, `:267`, `:293`,
  `:478`), `internal/export/export_test.go` (`:142`, `:602`, `:1002-1003`, `:1673`, `:1793`, `:2012`,
  `:2120`, `:2561`, `:3353`, `:3889`), `internal/diagram/contract_test.go` (`:523`, `:714`, `:901`,
  `:939`), `internal/diagram/svg_test.go` (`:42`, `:74`), `internal/diagram/drawio_test.go` (`:55`,
  `:94`, `:421`), `internal/diagram/ascii_test.go` (`:58`, `:77`, `:152`),
  `internal/diagram/mermaid_test.go` (`:56`, `:66`, `:76`, `:397`)

**Patterns to Follow:**
- `tasks/learnings.md` "Rename an AST field without moving a single wire name" — its receipt applies
  inverted here: no `.emod` source, golden or expected constant may be edited, because every wire
  spelling of this field was already removed in Tasks 5 and 6. A diff that moves an expected byte string
  means a surface was missed earlier
- `internal/formatter/formatter_test.go:3750`'s `cmpopts.IgnoreFields(ast.Trigger{}, "Comments")` names
  the struct and is the place a stale field name would surface as a comparison failure rather than a
  compile error
- `tasks/learnings.md` "Additive output changes owe a byte-identical receipt for models that do not use
  the feature" — the mirror obligation for a deletion, and the CLI comparison above is that receipt

**Testable:** No — the field has no producer and no reader by this point, so there is no observable
behaviour to assert; the compiler and the unchanged output of every command are the verification.

**Verification:** `mise exec -- go build ./...`; `mise exec -- go test -tags unit ./...`;
`mise exec -- go test -tags integration ./...`; `mise exec -- task test:viewer`;
`rg -n 'KindPos|Trigger\{[^}]*Kind' -g '*.go'` returns nothing.

**Depends on:** 4, 5, 6, 7

---

## Summary

**Nine tasks**, ordered as accept → migrate → stop reading → reject → delete. That order is the story's
whole risk management: the parser learns the new spelling while the old one still works (1, 2), the tree
moves to the new spelling under a formatter that writes it (3), every surface that displayed or
serialized the kind stops (4, 5, 6), only then does the old spelling become an error (7), and the
grammar and the AST field follow once nothing can produce or consume the value (8, 9).

The three intermediate groupings each answer a recorded failure. Task 3 cannot be split into "change the
formatter" and "migrate the sources", because the round-trip comparison at
`internal/formatter/formatter_test.go:538` reads `Trigger.Kind` and a source that still spells one would
read back empty. Tasks 4, 5 and 6 precede Task 7 so that no commit ships an ASCII label reading
`<<: Name>>` or a JSON trigger carrying `"kind": ""` — `jsonTrigger.Kind` has no `omitempty` and
`ascii.go:114` interpolates unconditionally. Task 6 keeps the exporter, the importer and the viewer's
panel in one commit, because `tasks/learnings.md` records US-002 splitting exactly those three and
shipping a panel that showed an em-dash forever with both suites green.

Task 4 and Task 5 depend on nothing and can run alongside Tasks 1–3. Tasks 5 and 6 are sequenced because
both write `internal/export/export.go`. Task 8 follows Task 7 rather than preceding it, because a
grammar narrower than the parser red-squiggles files `emod validate` accepts, which is the one failure
direction that matters.

**Story criteria coverage:**

| Story criterion | Task |
|---|---|
| `trigger "<name>" { ... }` parses, with the quoted name following the keyword directly | 1 (parser), 2 and 8 (grammar) |
| `trigger UI/Schedule/Processor "<name>"` are no longer accepted | 7 |
| The error for a removed spelling names its replacement | 7 |
| `emod fmt` emits the kindless form | 3 |
| JSON and CUE exports no longer carry a kind | 5 |
| A model exported, edited in the viewer and re-imported keeps name, actor and reads | 6 |
| Every example and fixture uses the new form and passes `emod validate` | 3 (sources), 7 (the rejection that makes it enforced) |

Carried along, not stated by the story: the two text renderers that displayed the kind (4), the diagram
document node and the viewer's details panel (6), the tree-sitter grammar and corpus (2, 8), and the AST
field itself (9).

**Handed to US-011, with the sites.** Markdown is out of scope here, so after Task 7 these files show a
trigger spelling `emod validate` rejects, and nothing in CI will notice: `README.md:33`,
`docs/dsl-reference.md:582`, `docs/proposals/completed/proposal.md:38`,
`docs/proposals/completed/dcb-proposal.md:193`, `docs/proposals/ai/06-mcp-server.md:339`,
`docs/proposals/ai/01-nl-to-model-generation.md:99`. `tasks/learnings.md` records that US-002 left the
same gap in `docs/dsl-reference.md` for two stories running, which is why the list is written out rather
than left to be rediscovered.

**Deferred to later stories in the feature:** the `reads` edge from a view to a trigger or automation
(US-005); lane placement, so the trigger box stays in the lane it occupies today (US-006); the palette,
so the viewer's trigger fill stays `#e1d5e7` against SVG and draw.io's `#ffffff` (US-007); the
`automation/missing-todo-list` rule (US-008); LSP hover, completion, go-to-definition and
find-references, including the "manual trigger" wording at `internal/lsp/hover.go:23` (US-009); the VS
Code TextMate rule at `editors/vscode/syntaxes/emod.tmLanguage.json:28-33` and the tree-sitter highlight
query at `editors/tree-sitter-emod/queries/highlights.scm:60`, both of which assume a kind identifier
(US-010); and the `wireframe` entry from the proposal's section 5, which has no story yet.

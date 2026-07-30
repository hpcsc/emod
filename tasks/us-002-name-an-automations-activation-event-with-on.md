# US-002: Name an automation's activation event with `on`

## Progress
- [x] Task 1: Accept `on <EventName>` as an automation's activation event
- [x] Task 2: Accept `on` inside an automation in the tree-sitter grammar
- [x] Task 3: Rename the activation-event field to `OnEvent` across the Go tree
- [ ] Task 4: Emit `on` from `emod fmt`
- [ ] Task 5: Reject `trigger <EventName>` inside an automation and move every model to `on`
- [ ] Task 6: Name the activation event `on_event` in the JSON, CUE and embedded schema exports
- [ ] Task 7: Name the activation event `on_event` in the diagram document and read it back
- [ ] Task 8: Show an automation's activation event as `On Event` in the viewer's details panel

---

## Story Reference

`user-stories/triggers-and-automations.md` → **US-002: Name an automation's activation event with `on`**
(second of eleven stories in "Triggers and Automations"). Design notes:
`docs/proposals/triggers-and-automations-proposal.md` — section 3 for the rename, `:161` for the AST
field name, `:233` and `:238` for the schema and document key names.

**In scope:** an `on <EventName>` entry inside `automation <Name> { ... }` carrying exactly the meaning
and the name resolution the previous `trigger <EventName>` spelling had; retiring that spelling with a
parse error that names `on` as its replacement; `emod fmt` writing `on`; the JSON, CUE, embedded
`schema.cue` and diagram-JSON documents naming the field consistently with the keyword; the value
surviving export to diagram JSON, editing in the viewer and re-import, and being shown in the viewer's
details panel; slice-level `trigger` keeping its current meaning and behaviour. Carried along because
the repo would otherwise be inconsistent or red: the `on` keyword in `internal/lexer/token.go`, the
tree-sitter grammar (which must never reject what `emod validate` accepts), the `ast.Automation` field
name, and every `.emod` source in the tree that spells the retired entry.

**Out of scope:** the `every "<expr>"` schedule attribute and the exactly-one-of rule over `on` and
`every` (US-003) — the post-block "requires an activation event" diagnostic is restated by this story
in terms of `on` and rewritten again by US-003; removing the trigger kind slot, so
`trigger UI "..."`, `trigger Schedule "..."` and `trigger Processor "..."` all still parse at slice
level (US-004); drawing the `reads` edge (US-005); lane placement (US-006); the palette (US-007); the
`automation/missing-todo-list` lint rule, so this story adds no rule and no `RuleName` (US-008); LSP
hover, completion, go-to-definition and find-references over `on` — including the
`keywordDescriptions` entry in `internal/lsp/hover.go:13-33` and the automation-body completion list
(US-009); the VS Code TextMate keyword alternation
(`editors/vscode/syntaxes/emod.tmLanguage.json:63`) and `editors/tree-sitter-emod/queries/*.scm`
(US-010); `docs/dsl-reference.md` — the Automation Pattern block (`:325-340`), which keeps showing
`trigger` until then — `README.md`, and `examples/*.emod` beyond the single line named below (US-011).
No DSL version bump: `emod` is pre-release, the header stays at 1, and the retired spelling fails as a
parse error rather than earning a versioning message.

**Consequences of that boundary, decided.** Six shapes the story does not spell out:

1. *Both spellings coexist for four tasks.* Task 1 adds `on` while `trigger` still parses, so every
   commit up to Task 5 leaves the tree green with no fixture, golden or example edited. Task 5 is the
   one that flips every model in the repo and removes the old branch. Ordering it any other way makes
   the first commit touch twenty-two files at once, which is what the story's "keep each commit to the
   task it is for" instruction rules out.
2. *`on` becomes a reserved word for element names, and stays legal as a field name, type and
   modifier.* Declarations and cross-references test `lexer.Identifier` strictly; field parts test
   `checkIdentifierLike` (`internal/parser/parser.go:1513-1520`), which admits any keyword. The
   keyword table is lowercase and lookup is case-sensitive
   (`internal/lexer/tokenizer.go`, `internal/lexer/token.go:67-104`), so a capitalised `On` stays an
   identifier. Nothing in the repo names an element `on` today — verified across `examples/`,
   `internal/parser/testdata/` and `internal/test/fixtures.go`.
3. *The `automation_trigger` edge type keeps its name*, in `export.ExportDiagramJSON`
   (`internal/export/export.go:1082`), in `foldEdges` (`internal/importer/importer.go:270`) and in the
   viewer's `model.js:145`, `config.js:22`/`:32` and `layout.js:267`. The proposal keeps it
   (`:239`), it is a document-and-styling contract rather than DSL surface, and renaming it would
   ripple into viewer CSS classes and the e2e suite for no gain to this story.
4. *`examples/all_patterns.emod` changes on exactly one line* — its automation's activation entry
   (`:78`) — even though `examples/` belongs to US-011. Leaving it would ship a flagship example that
   `emod validate` rejects. Nothing else under `examples/` is touched, and
   `examples/error_diagnostics_test.emod` keeps its deliberate errors.
5. *The Go field is renamed to `OnEvent`/`OnEventPos` in a commit of its own*, ahead of every wire
   name, with no output byte moving. Eighty-odd references across twenty-two Go files would otherwise
   ride inside a behaviour commit, which `tasks/learnings.md` records as the thing reviewers flag.
6. *There is no unfeatured twin and no byte-identical receipt for this story.* Unlike `reads`, an
   automation's activation event is required — `test.HotelReservation` has one, every fixture has one,
   and no model can omit it — so "an automation without the feature is untouched" has no meaning here.
   The receipts are the other direction: the values must be read back under their new names from input
   that carries them, and every renamed surface must reject or drop the old name.

**Evidence discipline for this story.** A test that asserts only the absence of a value, on input that
omits the entry, passes with the feature deleted. Every criterion below phrased "unchanged", "still",
"keeps its meaning" is written against input that exercises the changed handling: the slice-level
`trigger` criterion is asserted on a model that also contains an automation spelling `on`, the
old-key criteria are asserted on documents that actually carry the old key, and the round-trip
criteria are asserted on the fixture whose automations all declare activation events.

**Learnings folded in** from `tasks/learnings.md`: a new block entry keyword owes three things to the
parser's diagnostics (the "expected …" message, single-diagnostic recovery through
`skipRestOfLineOrBlockEnd`, and a `require.Len(t, diags, 1)` pin); ask the lexer which keywords exist
and never restate the set, so the keyword-coverage subtests cover `on` on the day the lexer learns it;
new DSL keywords must stay usable as field names, on both the Go and the tree-sitter side; put a new
parser subtest in the group that owns the construct; the formatter silently deletes an entry it has
not heard of, so parse → format → reparse against the original model is the guard, and `emod fmt`
canonicalises order so a golden is never the input re-indented; formatter output always begins with
`emod N`; a new exported field must land in JSON, CUE and `schema.cue` in the same change, and the two
order their keys differently so the ordering comes from a `json*` sibling — `emittedKeyOrder` makes
that assertable from the raw bytes; the two export guards cannot see a value neither writer emits;
the viewer's save path runs through `importer.ImportDiagram`, so a diagram-node key owes a read-back,
and `internal/viewer/static` is a display surface with its own vitest harness; the tree-sitter grammar
must never be stricter than the Go parser, its generated `src/` stays gitignored, and repo tooling
runs through `mise exec --`; CLI diagnostic tests must assert the distinguishing message text; a
second `require.Contains` on one message is often shadowed by the first; an assertion whose expected
value comes from the code under test cannot fail; never write emod source with `%q`; acceptance
criteria describe the working tree, never commit or branch state.

---

## Codebase Context

**Lexer.** `internal/lexer/token.go` declares thirty-six `Kind` constants in one iota block (`:9-47`)
and a lowercase `keywords` map (`:67-104`); `keywordNames` inverts it, `Keywords()` (`:120`) lists the
spellings sorted, and `Kind.IsKeyword()` (`:131`) is a map lookup. Four coverage subtests range over
`Keywords()` — `internal/lexer/tokenizer_test.go:13`, `internal/parser/parser_test.go:226` ("keywords
are usable as field names") and `:242` ("… as field name, type and modifier at once"), and
`internal/oracle/oracle_test.go:44` — so a keyword added to the map is covered the moment it exists.
`isKeyword` in `internal/lsp/hover.go:36` is still the ordinal range `KeywordModel..KeywordExternal`,
so a `Kind` appended after `KeywordRejected` is invisible to hover, which matches US-009 owning hover.

**AST.** `ast.Automation` (`internal/ast/ast.go:202-217`) carries `Comments`, `Name`/`NamePos`,
`Description`/`DescriptionPos`, `TriggerEvent`/`TriggerEventPos`, `Reads`/`ReadsPos`,
`Command`/`CommandPos`, `TargetContext`/`TargetContextPos` and the open/close positions.
`ast.Trigger` (`:173-187`) is the unrelated slice-level construct with `Kind`, `Name`, `Actor` and
`Reads`, reached through `Slice.Trigger` (`:82`). `TriggerEvent` has eighty-odd references across
twenty-two Go files: `internal/export/export_test.go` (14), `internal/validator/validator_test.go` (9),
`internal/export/export.go` (9), `internal/parser/parser_test.go` (6), `internal/diagram/mermaid.go`
(6), `internal/lsp/references.go` (5), `internal/importer/importer.go` (4), `internal/parser/parser.go`
(3), `internal/lsp/definition.go` (3), `internal/formatter/formatter_test.go` (3), and one or two each
in `internal/parser/integration_test.go`, `internal/formatter/formatter.go`,
`internal/diagram/{svg,drawio,ascii,contract}{,_test}.go`, `internal/ast/ast.go`,
`internal/validator/validator.go`, `internal/cli/slices.go`.

**Parser.** `parseAutomation` (`internal/parser/parser.go:1035-1136`) is an unbounded
`for !p.check(lexer.CloseBrace)` loop over `description`, `trigger`, `reads`, `command` and
`target context`. The `trigger` branch (`:1060-1069`) recovers from a missing value with a bare
`p.advance()`; the `reads` branch (`:1070-1079`), added by US-001, is the better shape — `errorAt` on
the keyword token then `skipRestOfLineOrBlockEnd`. The fallthrough reports
`expected description, trigger, reads, command, or target in automation, got %q` (`:1107`). After the
loop it reports `automation block requires a trigger event` (`:1119-1126`) and `… requires a command`.

**Parser tests.** `internal/parser/parser_test.go` is one umbrella of thirteen groups; `"automations"`
(`:1898`) owns the construct, `"error reporting"` (`:2464`) owns messages and recovery, `"triggers"`
(`:1493`) owns the slice-level construct. Three subtests are the ones this story rewrites:
`"trigger keyword inside automation is event name, not trigger block"` (`:2088`, which asserts
`slice.Trigger` stays nil), the unrecognised-entry loop over
`[]string{"description", "trigger", "reads", "command", "target"}` (`:2852`), and
`"automation missing trigger produces error"` (`:2857`, matching the message exactly). The
`reads`-position table at `:1955-1990` is the shape for a position-independence table.

**Formatter.** `writeAutomation` (`internal/formatter/formatter.go:340-357`) emits `description`, then
the activation event, then `reads`, `command`, `target context`. `"formats automation block"`
(`internal/formatter/formatter_test.go:258`) is the whole-output golden; `"round-trip through the
parser"` (`:545`) compares `parseModel(t, formatter.Format(original))` against the original with
`test.RequireEqual(..., ignoreFormatterNormalizations)`, which is what catches a dropped entry.
`internal/cli/fmt_test.go` pins canonical `*FormattedEmod` constants (its automation activation line
is `:203`) and feeds them to `requireFmtSettlesOn`.

**Exports.** `jsonAutomation` (`internal/export/export.go:158-172`) lists `name`, `description`,
`position`, then the four `*_position` keys, then `comments`, then the four values;
`convertAutomation` (`:598-618`) fills it; `cueWriter.writeAutomation` (`:1368-1376`) emits
`trigger_event`, `reads`, `command`, `target_context` via `lineIfSet`; `internal/cue/schema.cue`
`#Automation` (`:54-62`) declares the same keys with `comments?` and `name` first.
`emittedKeyOrder` pins the JSON key order at `internal/export/export_test.go:1224-1233`, beside the
translation's; `internal/cue/embed_test.go`'s `fullModelJSON` (`:126`, its automation at `:177`) is
the document `"accepts a model using every element the language offers"` (`:66`) vets; the coupled
guards are `"CUE and JSON exports describe the same model"` and `"output conforms to the schema's
Model definition"` (`:3441`, `:3458`). The CUE regex at `:3217` spells the current key.

**Diagram document and the viewer.** `jsonDiagramNode` (`internal/export/export.go:653-672`) carries
`trigger_event` among its type-specific keys; the automation node is built at `:830-846` and the
`automation_trigger`/`automation_command` edges at `:1067-1095`. `internal/importer/importer.go`
reverses it: `diagramNode` (`:25-42`) decodes `trigger_event`, `buildSlice` copies it onto the
automation (`:206-215`), and `foldEdges` (`:251-290`) folds an `automation_trigger` edge onto an
automation that has none. `internal/wasm/pipeline.go:60` and `:79` are the two halves the viewer calls.
`internal/viewer/static/ui.js:351-361` renders the automation section of the details panel — a
`Trigger Event` row reading `node.trigger_event`, then `Reads`, `Command`, `Target Context` — and
`internal/viewer/tests/detail-panel.test.js:40-75` asserts the whole section, row by row, twice.

**Fixtures and `.emod` sources spelling the retired entry.** `internal/test/fixtures.go` at `:67`
(`HotelReservation`), `:169` (`DescribedHotelReservation`), `:272` (`KeywordFieldSearchCatalog`),
`:647` and `:651` (`SpecLibraryLending`), `:725` and `:730` (`AutomationReadsLibraryLending`);
`internal/parser/testdata/all_patterns.emod:78` and `multi_context.emod:27`;
`examples/all_patterns.emod:78`. Inline test sources spell it in `internal/parser/parser_test.go`
(31 lines, including slice-level ones), `internal/formatter/formatter_test.go` (8),
`internal/cli/validate_test.go` (6), `internal/lsp/references_test.go` (5),
`internal/cli/fmt_test.go` (3, one of them the automation line at `:203`),
`internal/oracle/oracle_test.go` (3), `internal/lsp/definition_test.go` (3),
`internal/importer/importer_test.go` (2), `internal/lsp/server_test.go` (1),
`internal/cli/slices_test.go` (1). `internal/validator/validator_test.go` and the diagram tests build
`ast.Automation` literals instead, so they move with the field rename, not with the keyword.
`internal/test/fixtures.go:758` (`AutomationReadsLibraryLendingViewNames`) and `:854`
(`DeclaredAutomationReads`) are the transcribe-and-read-back pair to mirror; `declaredSlices` (`:867`)
is the walk that visits both slice homes.

**Tree-sitter.** `automation_definition` (`editors/tree-sitter-emod/grammar.js:218-228`) passes its
entries to `buildDescribedBlock`, so they are unordered and unbounded. Corpus cases live in
`test/corpus/slice.txt` — `Slice with automation` (`:259`), `Slice with automation containing reads`
(`:289`) and `Slice with automation declaring reads first` (`:322`) — plus a described automation in
`description.txt:155` and the keyword-as-field-name block in `fields.txt:15-30`. `version_header`'s
`token(seq('emod', /[ \t]+/))` is the precedent for narrowing a keyword token that would otherwise
compete with `any_identifier`. `src/` is gitignored and `task test:grammar` regenerates before running.

**Not touched, deliberately.** `internal/linter` (no rule reasons about the activation event, and
US-008 owns the new one); `internal/glossary` (an automation contributes no term of its own —
`tasks/learnings.md` records that, and its goldens are the receipt); `internal/cli` beyond fixture
sources and the canonical fmt constants (no new command, flag or error kind; `RunFmt` writes whatever
`formatter.Format` produces and `emod validate` reports whatever the parser produces);
`internal/diagram/{svg,drawio,mermaid,ascii}.go` beyond the field rename (US-005 through US-007 own
the picture); `internal/lsp` beyond the field rename (US-009); `editors/vscode` and
`editors/tree-sitter-emod/queries` (US-010); `docs/`, `README.md` (US-011); `e2e/` and `e2e-viewer/`,
whose sources use slice-level `trigger` only.

---

## Tasks

### Task 1: Accept `on <EventName>` as an automation's activation event

**Behavior:** `automation <Name> { ... }` accepts `on <EventName>` anywhere among its entries,
recording the event name and the source position of that name exactly where the `trigger <EventName>`
spelling records them. Both spellings work for now, so nothing else in the tree needs to move. An `on`
entry with a missing or non-identifier value reports exactly one diagnostic and does not swallow the
entry on the following line. The message reported for an unrecognised entry inside an automation names
`on`. `on` is a keyword, so it is reserved for element names and still legal as a field name, type and
modifier.

**Acceptance Criteria:**
- [ ] An automation declaring `on RoomReserved` parses with no diagnostics, and the automation carries
      that event name together with the filename, line and column of the name token
- [ ] `on` written as the block's first entry and `on` written after `target context` both parse with
      no diagnostics and record the same event name — position within the block is free, as it is for
      `description` and `reads`
- [ ] An automation declaring `on` twice keeps the value written last
- [ ] An automation spelling `trigger RoomReserved` still parses with no diagnostics and records the
      same event name, and every existing subtest in the `"automations"` group
      (`internal/parser/parser_test.go:1898`) passes unedited — Task 5 is what retires the spelling
- [ ] An automation writing `trigger EventA` and then `on EventB` records `EventB`, and one writing
      them the other way round records `EventA`, both with no diagnostics — the two spellings are one
      entry, not two
- [ ] An `on` entry with no value, followed on the next line by a `command` entry, reports exactly one
      diagnostic (`require.Len(t, diags, 1)`) whose message names both `on` and `automation`, and the
      `command` entry on the following line is still parsed onto the automation
- [ ] An `on` entry with no value written as the last entry of the block still lets the automation
      block and its enclosing slice close, reporting exactly one diagnostic
- [ ] The message reported for an unrecognised entry inside an automation names `on` alongside
      description, trigger, reads, command and target — the loop at
      `internal/parser/parser_test.go:2852` gains `on` and passes
- [ ] `lexer.Keywords()` contains `on`, `lexer.Scan("on", …)` yields a token whose `Kind` is not
      `Identifier` and whose `Kind.String()` is `on`, and a field named `on` with type `on` and
      modifier `on` parses with no diagnostics — all three come from the existing subtests that range
      over `Keywords()` (`internal/lexer/tokenizer_test.go:13`, `internal/parser/parser_test.go:226`,
      `:242`, `internal/oracle/oracle_test.go:44`), which must pass without being edited
- [ ] A model declaring an event named `On` and an automation activating on it parses with no
      diagnostics, pinning that keyword lookup is case-sensitive and that capitalised element names
      survive the reservation
- [ ] `oracle.Check` over `test.HotelReservation`, `test.DescribedHotelReservation`,
      `test.KeywordFieldSearchCatalog`, `test.InvariantLibraryLending`, `test.SpecLibraryLending` and
      `test.AutomationReadsLibraryLending` returns no diagnostics, with those fixtures unedited

**Affected Files/Modules:**
- `internal/lexer/token.go` — the `Kind` iota block (`:9-47`) and the `keywords` map (`:67-104`)
- `internal/parser/parser.go` — `parseAutomation` (`:1035-1136`) gains the branch; its "expected …"
  message (`:1107`) grows a term
- `internal/parser/parser_test.go` — subtests in `"automations"` (`:1898`), and the message and
  recovery subtests in `"error reporting"` (`:2464`)

**Patterns to Follow:**
- The branch to copy: the automation's `reads` entry (`internal/parser/parser.go:1070-1079`) —
  `errorAt` on the keyword token, then `skipRestOfLineOrBlockEnd`, which also stops at `}` so the
  block still closes. Do not copy the older `trigger` branch's bare `p.advance()` recovery
- `tasks/learnings.md` "A new block entry keyword owes three things to the parser's diagnostics" —
  the message list, one diagnostic per malformed entry, and the `require.Len(t, diags, 1)` pin
- `tasks/learnings.md` "Ask the lexer which keywords exist; never restate the set and never range over
  `Kind`" — add the spelling to the `keywords` map and append the `Kind` after `KeywordRejected`;
  `checkIdentifierLike` (`internal/parser/parser.go:1513-1520`) already asks `Kind.IsKeyword()`
- `tasks/learnings.md` "New DSL keywords must stay usable as field names" and "Keyword surfaces fan
  out past the lexer, parser and tree-sitter grammar" — the TextMate alternation, `keywordDescriptions`
  and the completion lists are US-009's and US-010's, and are deliberately left alone here
- Position independence, table-driven: the `reads` table at `internal/parser/parser_test.go:1955-1990`
- Subtest placement: `tasks/learnings.md` "Put a new parser subtest in the group that owns the
  construct" — the construct is the automation, not the trigger
- The block loop imposes no arity and no order; keep it that way (`tasks/learnings.md` "The tree-sitter
  grammar must never be stricter than the Go parser" records why)

**Testable:** Yes — through `lexer.Scan`, `parser.Parse` and `oracle.Check`, all exported.

**Verification:** `mise exec -- go test -tags unit ./internal/lexer/... ./internal/parser/...
./internal/oracle/...`; `mise exec -- go build ./...`.

**Depends on:** None

---

### Task 2: Accept `on` inside an automation in the tree-sitter grammar

**Behavior:** the tree-sitter grammar parses an automation whose activation event is written with `on`
without an `ERROR` node, so a file `emod validate` accepts is not red-squiggled in an editor using the
grammar. Everything the grammar accepted before is still accepted, including a field named `on`.

**Acceptance Criteria:**
- [ ] `automation_definition` (`editors/tree-sitter-emod/grammar.js:218-228`) admits an `on` entry,
      spelled the way its sibling entries are and passed to `buildDescribedBlock` rather than wrapped
      in `optional(...)`
- [ ] A corpus case in `editors/tree-sitter-emod/test/corpus/slice.txt` covers an automation whose
      activation event is written with `on` ahead of `reads` and `command`, and its expected tree
      contains no `ERROR` or `MISSING` node
- [ ] A second corpus case covers `on` written after `target context`, so the grammar is proved not to
      impose an order the Go parser does not
- [ ] A corpus case covers a field named `on` inside a `fields` block, in the shape of the existing
      keyword-as-field-name case (`test/corpus/fields.txt:15-30`), with no `ERROR` node — if the
      anonymous `on` token competes with `any_identifier`, narrow it the way `version_header` narrows
      `emod` with `token(seq(...))` rather than loosening the field rule
- [ ] The three existing automation cases (`slice.txt:259`, `:289`, `:322`) and the described
      automation in `description.txt:155` pass unedited
- [ ] `mise exec -- task test:grammar` passes, run through `mise exec --` so the repo-pinned
      tree-sitter CLI resolves rather than whichever one is on `PATH`
- [ ] `git check-ignore editors/tree-sitter-emod/src` succeeds, and the only files this task changes
      under `editors/tree-sitter-emod` are `grammar.js` and files under `test/corpus/`

**Affected Files/Modules:**
- `editors/tree-sitter-emod/grammar.js` — `automation_definition` (`:218-228`)
- `editors/tree-sitter-emod/test/corpus/slice.txt` — beside `Slice with automation` (`:259`)
- `editors/tree-sitter-emod/test/corpus/fields.txt` — the keyword-as-field-name block (`:15-30`)

**Patterns to Follow:**
- The entry spelling: the `reads` items in `automation_definition`, `translation_definition`
  (`:231-241`) and `trigger_definition` (`:168-178`)
- `tasks/learnings.md` "The tree-sitter grammar must never be stricter than the Go parser" — items go
  into `buildDescribedBlock`, never behind `optional(...)`
- `tasks/learnings.md` "New DSL keywords must stay usable as field names" — the permissive
  `any_identifier` plus a keyword token narrow enough to match only in its own position, as
  `version_header` does
- `tasks/learnings.md` "Run repo tooling through `mise exec --`, not bare PATH" and "Generated
  tree-sitter `src/` stays gitignored"
- Highlighting queries are US-010's; this task changes no `.scm` file

**Testable:** Yes — the tree-sitter corpus is the test surface, run by `task test:grammar`.

**Verification:** `mise exec -- task test:grammar`; `git status --porcelain editors/tree-sitter-emod`
lists only `grammar.js` and corpus files.

**Depends on:** 1

---

### Task 3: Rename the activation-event field to `OnEvent` across the Go tree

**Behavior:** `ast.Automation` names its activation event after the keyword that introduces it, and
every reader and writer in the Go tree follows. Nothing observable changes: the same emod source
parses to the same values, and the formatter, the four diagram renderers, the JSON, CUE and diagram
documents and the importer produce the same bytes they produced before. The wire names stay
`trigger_event` until Tasks 6 and 7.

**Acceptance Criteria:**
- [ ] `ast.Automation` (`internal/ast/ast.go:202-217`) declares `OnEvent`/`OnEventPos` in the position
      `TriggerEvent`/`TriggerEventPos` held — after the description and ahead of `Reads` — so every
      writer still reads the same sequence off the struct
- [ ] `rg -n 'TriggerEvent' -g '*.go' .` returns no match
- [ ] The `json:"trigger_event…"` struct tags in `internal/export/export.go` (`:162`, `:169`, `:669`)
      and `internal/importer/importer.go:35`, the `lineIfSet("trigger_event", …)` call
      (`internal/export/export.go:1372`), `internal/cue/schema.cue:58` and `node.trigger_event` in
      `internal/viewer/static/ui.js:355` are all unchanged by this task
- [ ] Every file this task changes is a `.go` file: no `.emod` source, no `.cue` file, no `.js` file
      and no golden or expected-output constant is edited
- [ ] `mise exec -- go test -tags unit ./...` and `mise exec -- go test -tags integration ./...` pass
      with no expected value in `internal/formatter`, `internal/export`, `internal/cue`,
      `internal/diagram`, `internal/importer`, `internal/glossary`, `internal/cli` or `internal/lsp`
      edited — only the identifier references in those packages' tests move
- [ ] `mise exec -- task test:viewer` passes with no file under `internal/viewer` changed

**Affected Files/Modules:**
- `internal/ast/ast.go`, `internal/parser/parser.go`, `internal/formatter/formatter.go`,
  `internal/validator/validator.go`, `internal/export/export.go`, `internal/importer/importer.go`,
  `internal/cli/slices.go`, `internal/lsp/{definition,references}.go`,
  `internal/diagram/{mermaid,svg,drawio,ascii}.go`
- Their tests, which reference the field in `ast.Automation` literals and assertions:
  `internal/export/export_test.go`, `internal/validator/validator_test.go`,
  `internal/parser/{parser,integration}_test.go`, `internal/formatter/formatter_test.go`,
  `internal/diagram/{contract,svg,drawio,ascii}_test.go`

**Patterns to Follow:**
- The name comes from `docs/proposals/triggers-and-automations-proposal.md:161` — `OnEvent`, matching
  the `on` keyword, so `ast.Trigger` is the only thing in the AST still called a trigger
- `tasks/learnings.md` "De-duplicate before a fan-out edit, and land the de-duplication with proof" —
  this task is the mechanical half on its own; do not restructure, extract or reorder anything while
  renaming, and do not fold the wire names in
- The story's own instruction: a task that asks for one field gets exactly that

**Testable:** No — the rename has no observable behaviour of its own; the existing suites, passing with
no expected value edited, are what prove nothing moved.

**Verification:** `mise exec -- go build ./...`; `mise exec -- go test -tags unit ./...`;
`mise exec -- task test:viewer`; `git status --porcelain` lists `.go` files only.

**Depends on:** 1

---

### Task 4: Emit `on` from `emod fmt`

**Behavior:** `emod fmt` writes an automation's activation event as `on <EventName>`, on its own line
at the block's indent, above `reads`. A file written with the retired spelling is rewritten to `on`,
which is the migration path a reader has before Task 5 removes the spelling. A parse → format →
reparse cycle recovers the activation event rather than dropping it, and formatting the formatter's
own output produces identical bytes.

**Acceptance Criteria:**
- [ ] Formatting an automation puts `on <EventName>` on its own line between the description and the
      `reads` line, asserted against a whole expected output in `"element formatting"`
      (`internal/formatter/formatter_test.go:258` `"formats automation block"`)
- [ ] Two sources for the same model — one spelling the activation entry `trigger`, one spelling it
      `on` — format to identical bytes, and that expected byte string is written out in the test
      rather than being either input re-indented
- [ ] Parsing `test.AutomationReadsLibraryLendingModel(t)`, formatting it and reparsing yields an AST
      equal to the original under `ignoreFormatterNormalizations`, so every automation's activation
      event survives; formatting that output again produces identical bytes
- [ ] The same parse → format → reparse over `test.SpecLibraryLendingModel(t)` and
      `test.HotelReservationModel(t)` holds, so the guard covers a model in both slice homes and the
      unfeatured fixture
- [ ] The canonical `*FormattedEmod` constants in `internal/cli/fmt_test.go` spell the automation's
      activation entry `on` (its automation line is `:203`), `requireFmtSettlesOn` passes for each,
      and running `RunFmt` over a file already in that canonical form leaves the bytes unchanged
- [ ] Every expected string still begins with `emod 1`
- [ ] No line other than the automation's activation line moves in any golden or canonical constant —
      state the diff in the commit message

**Affected Files/Modules:**
- `internal/formatter/formatter.go` — `writeAutomation` (`:340-357`)
- `internal/formatter/formatter_test.go` — `"formats automation block"` (`:258`) and the leaves of
  `"round-trip through the parser"` (`:545`)
- `internal/cli/fmt_test.go` — the canonical `*FormattedEmod` constants

**Patterns to Follow:**
- The line to emit and where it sits: the sibling `lineIfSet`-style lines in `writeAutomation` and
  `writeTranslation` (`internal/formatter/formatter.go:359-377`)
- `tasks/learnings.md` "A new block entry goes after `description` and ahead of nested blocks, in every
  writer" — and why the formatter is the writer that hurts to forget: it emits only what it knows
- `tasks/learnings.md` "`emod fmt` canonicalises order, so a fmt golden is never the input
  re-indented", and "Formatter output always begins with `emod N`"
- The round-trip comparison against the original model is the guard, not idempotence and not an
  existing golden: `internal/formatter/formatter_test.go:545-583` with
  `test.RequireEqual(..., ignoreFormatterNormalizations)`
- `tasks/learnings.md` "Never write emod source with `%q`" — activation event names go out through the
  same path every other identifier does
- No `internal/cli` code change: `RunFmt` writes whatever `formatter.Format` returns

**Testable:** Yes — through `formatter.Format`, `parser.Parse`, the exported `internal/test` model
helpers and `cli.RunFmt`.

**Verification:** `mise exec -- go test -tags unit ./internal/formatter/... ./internal/cli/...`;
`mise exec -- go test -tags unit ./...`.

**Depends on:** 1, 3

---

### Task 5: Reject `trigger <EventName>` inside an automation and move every model to `on`

**Behavior:** `trigger` is no longer an automation entry. Writing it inside an automation body reports
one parse error naming `on` as its replacement, and the rest of the block still parses. The
missing-activation diagnostic names `on` too. Every model in the repo — fixtures, parser testdata,
inline test sources, the tree-sitter corpus and the one flagship example — spells the entry `on`.
Slice-level `trigger` is untouched: it still introduces a wireframe block with a kind, a quoted name,
an optional actor and an optional `reads`.

**Acceptance Criteria:**
- [ ] `trigger <EventName>` inside an automation body reports exactly one diagnostic
      (`require.Len(t, diags, 1)`), positioned at the `trigger` token, whose message names both
      `trigger` and `on` as its replacement; the subtest asserts the whole formatted diagnostic line
      with one `require.Equal` on `diags[0].String()` rather than layering `require.Contains` calls
- [ ] The `reads`, `command` and `target context` entries written on the lines after a rejected
      `trigger` entry are still parsed onto the automation, and the automation block and its enclosing
      slice still close
- [ ] An automation declaring no activation event reports its post-block diagnostic naming `on`, and
      no message anywhere in `parseAutomation` still reads "trigger event"; the subtest at
      `internal/parser/parser_test.go:2857` asserts the current text
- [ ] The unrecognised-entry message no longer names `trigger`; the loop at
      `internal/parser/parser_test.go:2852` asserts the current list
- [ ] `internal/parser/parser_test.go:2088` `"trigger keyword inside automation is event name, not
      trigger block"` is replaced by a subtest asserting the rejection
- [ ] One model containing, in the same slice, a slice-level `trigger UI "Reservation Form" { actor …
      reads … }` block and an automation spelling `on <EventName>` parses with no diagnostics, and one
      subtest asserts both together — the slice's `Trigger` kind, name, actor and `reads`, and the
      automation's activation event — so a change that confused the two constructs fails it
- [ ] Every `.emod` source in the tree spells the entry `on`: `internal/test/fixtures.go` (`:67`,
      `:169`, `:272`, `:647`, `:651`, `:725`, `:730`), `internal/parser/testdata/all_patterns.emod:78`,
      `internal/parser/testdata/multi_context.emod:27`, and the inline sources in the parser, formatter,
      CLI, LSP, oracle and importer tests. No automation body anywhere in the tree still spells
      `trigger`; every remaining `trigger` line is a slice-level block carrying a kind and a quoted name
- [ ] `oracle.Check` returns no diagnostics for all six `internal/test` fixtures, and
      `mise exec -- go run ./cmd/emod validate <path>` exits 0 for every file under `examples/` except
      `examples/error_diagnostics_test.emod`, whose deliberate errors are unchanged
- [ ] `examples/all_patterns.emod` differs on exactly one line — its automation's activation entry
      (`:78`) — and no other file under `examples/` is changed
- [ ] `automation_definition` in `editors/tree-sitter-emod/grammar.js` no longer lists a `trigger`
      entry; the corpus cases that spelled it now spell `on`; `mise exec -- task test:grammar` passes
      and `src/` stays gitignored
- [ ] The model version header stays at 1: `ast.SupportedVersion` is unchanged, no file gains a new
      header, and the retired spelling produces a parse error rather than a version diagnostic

**Affected Files/Modules:**
- `internal/parser/parser.go` — the `trigger` branch of `parseAutomation` (`:1060-1069`) becomes the
  rejection; the "expected …" message (`:1107`) and the post-block diagnostic (`:1119-1126`)
- `internal/parser/parser_test.go` — `"automations"` (`:1898`), `"triggers"` (`:1493`) and
  `"error reporting"` (`:2464`)
- `internal/test/fixtures.go`, `internal/parser/testdata/{all_patterns,multi_context}.emod`,
  `examples/all_patterns.emod`
- Inline sources in `internal/formatter/formatter_test.go`, `internal/cli/{fmt,validate,slices}_test.go`,
  `internal/lsp/{references,definition,server}_test.go`, `internal/oracle/oracle_test.go`,
  `internal/importer/importer_test.go`, `internal/parser/integration_test.go`
- `editors/tree-sitter-emod/grammar.js` and `test/corpus/slice.txt`

**Patterns to Follow:**
- Reporting once and recovering: `errorAt` on the keyword token then `skipRestOfLineOrBlockEnd`
  (`internal/parser/parser.go:1070-1079`, `:1533-1537`) — a bare `p.advance()` leaves the event name
  to fail again against the entry list, which is the cascade `tasks/learnings.md` "A new block entry
  keyword owes three things to the parser's diagnostics" describes
- Assert the whole formatted diagnostic line: `internal/validator/validator_test.go:968`, `:1005`,
  `:1375` — `tasks/learnings.md` "A second `require.Contains` on one message is often shadowed by the
  first"
- The slice-level construct to leave alone: `parseTrigger` and the `"triggers"` group
  (`internal/parser/parser_test.go:1493-1629`), which US-004 will revisit for the kind slot
- If a CLI-level leaf is added for the new error, assert the tokens that identify this diagnostic, not
  just a path and a line — `tasks/learnings.md` "CLI diagnostic tests must assert the distinguishing
  message text"
- The grammar may stay looser than the Go parser but must never be stricter, so dropping the entry
  there is tidiness, not a constraint — `tasks/learnings.md` "The tree-sitter grammar must never be
  stricter than the Go parser"
- Fixture edits are one line each: do not restructure a fixture, add a construct, or reorder entries
  while migrating the spelling
- The formatter already writes `on` (Task 4), so no golden or canonical constant should need editing
  here; if one does, the two tasks have drifted

**Testable:** Yes — through `lexer.Scan`, `parser.Parse`, `oracle.Check` and the `emod validate` CLI
path.

**Verification:** `mise exec -- go test -tags unit ./...`; `mise exec -- go test -tags integration
./...`; `mise exec -- task test:grammar`; `mise exec -- go run ./cmd/emod validate` over each file
under `examples/` and `internal/parser/testdata/`.

**Depends on:** 1, 2, 4

---

### Task 6: Name the activation event `on_event` in the JSON, CUE and embedded schema exports

**Behavior:** `export.ExportJSON` and `export.ExportCUE` name an automation's activation event after
the keyword that declares it, and `internal/cue/schema.cue` declares the same key, so `emod schema`
describes it and `cue vet -d '#Model'` accepts a document that carries it and rejects one that carries
the old name. The value and its position are the ones the parser recorded.

**Acceptance Criteria:**
- [ ] The JSON export names the automation's activation event `on_event` and its position
      `on_event_position`, in the struct positions the old keys held, so `emittedKeyOrder` for an
      automation reads `name`, `position`, `on_event_position`, `reads_position`, `command_position`,
      `on_event`, `reads`, `command` — asserted in the same subtest that lists the translation's keys
      (`internal/export/export_test.go:1224-1233`), which makes the expectation non-arbitrary
- [ ] The CUE export emits `on_event` after `description` and ahead of `reads`, and the regex subtest
      at `internal/export/export_test.go:3217` asserts that order against the fixture
- [ ] `#Automation` in `internal/cue/schema.cue` declares `on_event?` and no longer declares
      `trigger_event?`, and a subtest vets a document carrying `trigger_event` against `#Model` and
      requires the failure — the schema's rejection of the old key is asserted, not assumed
- [ ] `internal/cue/embed_test.go`'s `fullModelJSON` (`:126`, its automation at `:177`) uses the new
      key, and `"accepts a model using every element the language offers"` (`:66`) passes
- [ ] `"CUE and JSON exports describe the same model"` and `"output conforms to the schema's Model
      definition"` (`internal/export/export_test.go:3441`, `:3458`) pass for every fixture
- [ ] `internal/test` exports the activation events of `test.AutomationReadsLibraryLending`
      transcribed by hand, in declaration order across both slice homes, alongside a read-back helper
      over a parsed model; a subtest decodes the fixture's JSON export and its CUE export and requires
      both to carry exactly that list, so a value neither writer emits cannot agree trivially
- [ ] `mise exec -- go run ./cmd/emod schema` prints `on_event` and does not print `trigger_event`
- [ ] The diagram JSON document is untouched by this task — its automation node still carries
      `trigger_event`, and every subtest in `"diagram json"` (`internal/export/export_test.go:1544`)
      passes unedited

**Affected Files/Modules:**
- `internal/export/export.go` — `jsonAutomation` (`:158-172`), `convertAutomation` (`:598-618`),
  `cueWriter.writeAutomation` (`:1368-1376`)
- `internal/cue/schema.cue` — `#Automation` (`:54-62`)
- `internal/cue/embed_test.go` — `fullModelJSON` (`:126`, `:177`)
- `internal/export/export_test.go` — leaves in `"model json"`, `"cue"`, the key-order subtest
  (`:1224`), schema conformance (`:3441`) and export parity (`:3458`)
- `internal/test/fixtures.go` — the transcribed activation-event list and its read-back helper

**Patterns to Follow:**
- Key naming and struct ordering: `jsonTranslation` (`internal/export/export.go:175-189`) — positions
  first, then values, and copy the ordering from a `json*` sibling, never from the schema
  (`tasks/learnings.md` "JSON and CUE order their document keys differently")
- `tasks/learnings.md` "JSON key order is assertable from the raw bytes — `emittedKeyOrder` already
  exists"
- `tasks/learnings.md` "A new exported field must land in JSON, CUE and `schema.cue` in the same
  change" — `#Automation` is closed, so `cue vet -d '#Model'` rejects an undeclared key
- `tasks/learnings.md` "The two export guards cannot see a list neither writer emits" — read the
  values back out of both decoded documents against a hand-transcribed list; `listsKeyedBy`
  (`internal/export/export_test.go`) is the read-back shape, and `requireBothFormatsAgree` strips
  positions before comparing, so the position key is observable in the JSON document only
- The transcribe-and-read-back pair to mirror: `AutomationReadsLibraryLendingViewNames`
  (`internal/test/fixtures.go:758`), `DeclaredAutomationReads` (`:854`) and `declaredSlices` (`:867`),
  which visits both slice homes — `tasks/learnings.md` "A slice has two homes"
- No twin helper is needed: an activation event is required, so there is no absent case to strip
- Never phrase a check as `emod export <file> -f cue` — the flag after the file argument is discarded
  (`tasks/learnings.md` "urfave/cli v2 discards every flag written after the file argument"); exercise
  `export.ExportJSON` / `export.ExportCUE` directly
- Do not touch the diagram document or re-merge `jsonDiagramEvent` into `jsonEvent`

**Testable:** Yes — through `export.ExportJSON`, `export.ExportCUE` and the embedded schema, all
exported.

**Verification:** `mise exec -- go test -tags unit ./internal/export/... ./internal/cue/...
./internal/test/...`; `mise exec -- go run ./cmd/emod schema`.

**Depends on:** 3

---

### Task 7: Name the activation event `on_event` in the diagram document and read it back

**Behavior:** the diagram JSON document names an automation's activation event after the keyword too,
and `importer.ImportDiagram` reads that key back onto the automation it builds, so a model exported
for the viewer, edited there and re-imported keeps its activation event. The `automation_trigger` edge
keeps its type name and still sets the activation event of an automation drawn without one.

**Acceptance Criteria:**
- [ ] The automation node in `export.ExportDiagramJSON`'s output carries the activation event under
      `on_event`, asserted in `"automation node with …"` (`internal/export/export_test.go:1845`)
- [ ] Exporting `test.AutomationReadsLibraryLendingModel(t)` to diagram JSON and importing the result
      yields a model whose activation events, read with the Task 6 helper, equal the transcribed list
      — both slice homes, in declaration order
- [ ] A diagram document whose automation node still spells `trigger_event` imports with no activation
      event on that automation, so the read-back is proved to be reading the new key rather than
      tolerating either
- [ ] A leaf in `"round trip"` (`internal/importer/importer_test.go:38`) over a hand-written non-dcb
      source already in canonical `emod fmt` form, whose slice declares an automation with `on` and a
      `command`, formats back to identical bytes after the export → import path
- [ ] `export.ExportDiagramJSON` still emits an edge of type `automation_trigger` from the event node
      to the automation node, and `foldEdges` still folds that type onto an automation carrying no
      activation event — asserted with an edge-only document, so a viewer-drawn event→automation arrow
      still sets the activation event
- [ ] The five existing `"round trip"` leaves (`:39`, `:52`, `:63`, `:90`, `:119`) pass unedited
- [ ] No file under `internal/viewer` is changed by this task — `git status --porcelain
      internal/viewer` lists nothing

**Affected Files/Modules:**
- `internal/export/export.go` — `jsonDiagramNode` (`:653-672`) and the automation node (`:830-846`)
- `internal/importer/importer.go` — `diagramNode` (`:25-42`) and the automation branch of `buildSlice`
  (`:206-215`)
- `internal/export/export_test.go` — leaves in `"diagram json"` (`:1544`)
- `internal/importer/importer_test.go` — leaves in `"round trip"` (`:38`) and `"edges"` (`:140`)

**Patterns to Follow:**
- `tasks/learnings.md` "The viewer's save path is `importer.ImportDiagram`, so a diagram-node field
  owes a read-back" — the export key and the decode tag move together or the value is silently dropped
  on save
- The byte-level round-trip shape, and why a hand-written source rather than a fixture:
  `"preserves slices declared directly under a context"` (`internal/importer/importer_test.go:63-88`)
  and `"preserves a translation without duplicating its nested event"` (`:90-117`), both comparing
  `formatter.Format(importFrom(t, source))` against the source itself
- Node metadata and edges are separate channels: a value carried as metadata survives with no edge,
  and the edge is what a viewer-drawn arrow leaves behind
- Decision 3 in the Story Reference: `automation_trigger` keeps its name in the document, in
  `foldEdges` and in `internal/viewer/static/{model,config,layout}.js`
- `tasks/learnings.md` "An assertion whose expected value comes from the code under test is the
  recurring review finding" — name the input that makes each assertion fail before writing it

**Testable:** Yes — through `export.ExportDiagramJSON` and `importer.ImportDiagram`, both exported.

**Verification:** `mise exec -- go test -tags unit ./internal/export/... ./internal/importer/...`;
`mise exec -- go test -tags unit ./...`.

**Depends on:** 3, 6

---

### Task 8: Show an automation's activation event as `On Event` in the viewer's details panel

**Behavior:** selecting an automation in the viewer shows its activation event under a label naming
the `on` entry, above the view it reads, the command it issues and its target context, and shows the
same placeholder the other rows use when the node carries no activation event.

**Acceptance Criteria:**
- [ ] The automation section of the details panel reads the node's `on_event` key and labels the row
      `On Event`, derived from that key the way every sibling label is derived from its own; the row
      stays first in the section, above `Reads`
- [ ] An automation node carrying no activation event shows the same em-dash placeholder the section's
      other rows show, rather than an empty cell or a missing row
- [ ] The value is escaped on the way into the panel, asserted on a value containing markup, the way
      the section's existing escaping case is
- [ ] Both whole-section assertions in `internal/viewer/tests/detail-panel.test.js` (`:53`, `:66`)
      list the new label and still list `Reads`, `Command` and `Target Context` with their values, so
      the row order and the other rows are covered rather than the new row alone
- [ ] A node object spelling the old key shows the placeholder, since the panel reads the key the
      exporter now writes
- [ ] `mise exec -- task test:viewer` passes, and `git status --porcelain` lists changes under
      `internal/viewer` only

**Affected Files/Modules:**
- `internal/viewer/static/ui.js` — the automation section of `showDetailPanel` (`:351-361`)
- `internal/viewer/tests/detail-panel.test.js` — the automation describe block (`:40-80`)

**Patterns to Follow:**
- The row shape and the placeholder: the section's own `Reads`, `Command` and `Target Context` rows
  (`internal/viewer/static/ui.js:355-358`), and the trigger and translation sections (`:331-340`,
  `:362-376`)
- `tasks/learnings.md` "`internal/viewer/static` is a display surface with its own vitest harness" —
  a node key no section names is invisible however faithfully the Go pipeline carries it, and
  restructuring `showDetailPanel` belongs in its own commit; this task edits one row
- The test harness: `internal/viewer/tests/detail-panel.test.js` already builds its store, its
  `automationNode` helper and its `shownRows` reader; extend them rather than adding a second file
- Assert what the panel shows, not how the HTML is built —
  `~/.config/ai/guidelines/testing/caller-patterns.md`, the UI pattern: the caller is the person
  reading the panel, so assert visible labels and values, never markup structure
- The node literal in the test spells the keys `export.ExportDiagramJSON` writes
  (`internal/export/export.go:653-672`), which Task 7 renamed
- No Go change: the exporter's key is Task 7's

**Testable:** Yes — through the exported `UI.showDetailPanel`, under vitest's jsdom environment
(`internal/viewer/vitest.config.js`).

**Verification:** `mise exec -- task test:viewer`; `git status --porcelain` lists files under
`internal/viewer` only.

**Depends on:** 7

---

## Summary

**Eight tasks.**

**Ordering rationale — dependency-first, with the widest edit isolated.** Task 1 puts `on` into the
language beside the spelling it replaces, so every later commit has somewhere to move to and nothing
in the tree has to move yet. Task 2 follows immediately because the tree-sitter grammar must never
reject a file `emod validate` accepts. Task 3 is the mechanical rename, landed alone and early so that
Tasks 4 to 8 are written against the final field name and no behaviour commit carries eighty
identifier edits. Task 4 comes before Task 5 in both senses: the formatter must already emit `on`
before the old spelling stops parsing, or its own output would be unparseable, and doing it first
means Task 5 needs no golden edits. Task 5 is the story's headline and its widest edit — the
rejection, every model in the repo, and the grammar entry — kept to exactly that. Tasks 6 and 7 are
the two document families, ordered model-document first because the diagram document's read-back
reuses the transcribed list Task 6 adds. Task 8 is the one JavaScript slice, and it follows Task 7
because the key it reads is the one the exporter starts writing there.

**Story acceptance criteria coverage:**

| Story criterion | Task |
|---|---|
| An `automation` accepts `on <EventName>`, carrying the same meaning and name resolution the previous spelling had | 1, with 2 mirroring it in the grammar; the validator's existing check resolves the same field, renamed by 3 |
| `trigger <EventName>` inside an automation is no longer accepted, and the error names `on` as its replacement | 5 |
| `emod fmt` emits `on` | 4 |
| JSON and CUE exports name the field consistently with the keyword | 6, with 7 doing the same for the diagram document |
| A model exported, edited in the viewer, and re-imported keeps its activation event | 7, with 8 showing it in the details panel |
| `trigger` at slice level keeps its current meaning and behaviour | 5, asserted on a model that also contains an automation spelling `on` |

**Nothing deferred from this story's criteria.** Deliberately left to later stories in the feature, and
therefore stale in the working tree until then: `docs/dsl-reference.md`'s Automation Pattern block
(`:325-340`), which keeps showing `trigger` and describing it as the activating entry, plus the rest
of `examples/*.emod` and `README.md`, all owned by US-011; the `every` schedule and the exactly-one-of
rule that rewrites the missing-activation diagnostic again, owned by US-003; the trigger kind slot,
owned by US-004; `keywordDescriptions`, hover and completion entries for `on` in `internal/lsp`, owned
by US-009; the VS Code TextMate keyword alternation and the tree-sitter highlighting queries, which do
not list `on`, owned by US-010.

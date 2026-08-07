# US-013: Fire automations after elapsed time

## Progress
- [ ] Task 1: Accept and colour `after` across the three editor grammars
- [ ] Task 2: Parse `after "<duration>"` on an automation's activation line
- [ ] Task 3: Reject an `after` value that is not a Go duration
- [ ] Task 4: Emit `after` from `emod fmt`
- [ ] Task 5: Share a fixture whose automations fire after an elapsed delay
- [ ] Task 6: Carry the delay into the JSON, CUE and embedded schema exports
- [ ] Task 7: Carry the delay through the diagram document and back
- [ ] Task 8: Label the `event -> automation` arrow with the delay in SVG and draw.io
- [ ] Task 9: Name the delay in Mermaid, ASCII and `emod slices list`
- [ ] Task 10: Show an automation's delay in the viewer

---

## Story Reference

`user-stories/specs-and-metadata.md` → **US-013: Fire automations after elapsed time**
(thirteenth story of "Specs, Invariants, and Model Metadata", lines 169-179). Design notes:
`docs/proposals/specs-and-metadata-proposal.md` §7 "Elapsed-Time Automations" (`:299-322`), the AST
extension at `:384`, the keyword list at `:390`, the diagram line at `:428`, the worked example at
`:564-576`, and the two risks recorded at `:618-619`.

**In scope:** an optional `after "<duration>"` suffix on an automation's `on` clause, read as "the
stated duration after each `on` event occurrence, issue the command"; the duration stated in Go
duration syntax, with a located error when the text does not parse as one; the rule that `after` on a
schedule-driven automation is an error, since `every` is already absolute; and diagrams carrying the
duration on the `event -> automation` edge while the clock badge on the automation box stays reserved
for `every`, so a relative delay and a wall-clock schedule read differently. Carried with it, because
each surface would otherwise drop the clause or go red: the `after` keyword in
`internal/lexer/token.go`; the three editor grammars, which an executable keyword-coverage suite
requires to name every lexer keyword (see the scoping note below); `emod fmt`, which silently deletes
an entry it has never heard of; the JSON and CUE exports plus the embedded `schema.cue`; the diagram
document and the viewer's save path; and a shared fixture the packages downstream read the delay out
of.

**Out of scope:**
- Timer runtime concerns. How the delay is implemented — durable scheduling, delivery guarantees,
  idempotency, clock skew, redelivery, missed timers — is a runtime property and stays out of the
  model, the same line the DCB proposal drew for append-condition checking. `after "24h"` states a
  duration and nothing else. The story's Non-Goals list "Modeling timer runtime properties" outright,
  and the proposal (`:618`) records that teams will ask the model to say more and that the answer is
  no. Nothing in this story adds a field, key, badge or diagnostic about any of them.
- Canonical attribute ordering and alignment in `emod fmt` beyond a faithful round-trip (US-014). This
  story writes `after` as an unaligned suffix on the activation line; the proposal's worked example
  aligns the activation keywords into a column, and that column is US-014's.
- LSP **completion** for `after` — `valueSlots` and `keywordsFor` (`internal/lsp/completer.go:176`,
  `:223`) — US-015. This half is genuinely deferrable, and the direction of the assertion is why:
  `TestKeywordCoverage/completion` (`internal/lsp/keywords_test.go:59-75`) iterates `keywordBlocks`
  and requires every label a block *offers* to be a lexer keyword, not every lexer keyword to be
  offered. An `after` the completer never suggests breaks nothing. LSP **hover** is a different story
  and is not deferrable — see the scoping note below. A duration is a literal rather than a declared
  name, so `after` earns no go-to-definition or find-references entry in `internal/lsp/model.go:115-120`
  either, in this story or in US-015.
- Rendering specs on diagrams (US-016), which is the other `--specs` diagram flag and unrelated.
- Examples and reference coverage (US-018): no `.emod` file under `examples/` or
  `internal/parser/testdata/` gains a delay, and `docs/dsl-reference.md` keeps its current Automation
  Pattern section. `README.md` is likewise untouched.
- `internal/linter`: this story adds no rule and no `RuleName`. `checkMissingTodoList`
  (`internal/linter/linter.go:497-509`) branches its wording on `auto.Schedule` and is not re-branched
  on a delay.
- `internal/glossary` (an automation contributes no term of its own) and `internal/arrange`
  (`arrange.go:25`, `:138` orders slices by the `on` event; a delay does not change which event causes
  which slice).
- `e2e/` and `e2e-viewer/`, neither of which declares an automation.

**Two scoping corrections, stated up front.** Two executable drift suites turn "the lexer knows a new
keyword" into obligations on hand-written surfaces this story would otherwise defer. Both were written
after US-003 was decomposed, which is why that story could defer what this one cannot.

1. **The three editor grammars cannot be deferred to US-017.** `TestEditorKeywordCoverage`
   (`editors/tree-sitter-emod/test/queries/keywords_test.go:47-64`, built under `//go:build grammar`
   and run by `task test:grammar`, which `task test` and CI both invoke) iterates `lexer.Keywords()`
   and requires every spelling to appear in `editors/tree-sitter-emod/grammar.js`, in
   `editors/tree-sitter-emod/queries/highlights.scm` and in
   `editors/vscode/syntaxes/emod.tmLanguage.json`. The moment `after` joins the lexer's `keywords`
   map, all three go red unless they name it. Task 1 therefore lands the editor surfaces *before* the
   lexer learns the word, so every commit in this story is green. US-017 keeps its criterion; this
   story necessarily satisfies the `after` half of it, and US-017's remaining work is the other nine
   keywords plus payload literals.
2. **The LSP keyword hover map cannot be deferred to US-015.** `TestKeywordCoverage/hover`
   (`internal/lsp/keywords_test.go:19-57`, build tag `unit`, so `task test:unit` and CI) iterates
   `lexer.Keywords()` and requires `lsp.GetHover` to return non-nil hover with non-empty contents for
   every one, its comment saying outright that the descriptions are hand-written and that this is the
   assertion that remembers. So `keywordDescriptions` (`internal/lsp/hover.go:10-49`) must gain an
   `after` entry in the same change as the lexer. That is Task 2, by the same reasoning Task 1 rests
   on: the change that makes the lexer aware of the word owes every hand-maintained surface its entry.
   Completion is the deferrable half — the assertion beside it runs the other direction (see Out of
   scope above) — so US-015 keeps completion and loses only hover.

**A pre-existing defect this story must not appear to introduce.** `internal/lsp/diagnostics.go:31-36`
maps severities with a two-arm switch — `diagnostic.Warning` to the LSP warning severity, everything
else through `default` to the LSP *error* severity — so an `Info` diagnostic renders as a red squiggle
in an editor. It already affects `dcb/single-tag-everywhere` and the other info-level rules. This
story adds no info-level diagnostic and touches no file in that path, so nothing here changes the
behaviour; it is recorded so a reviewer reading an LSP-adjacent diff does not read it as new.

**Open questions, decided.** Seven shapes the story does not spell out, each decided so the story
lands as one coherent surface:

1. *`after` parses as a trailing suffix on **both** activation lines, and the combination is what is
   rejected.* The proposal writes `every "0 2 * * *" after "24h"` as the text that has no reading, so
   the error has to be reachable from text the grammar admits. Admitting `after` only behind `on`
   would answer that input with `expected description, on, every, reads, command, or target in
   automation, got "after"` — a message that names no rule and explains nothing. Both activation
   entries take the suffix; the block-level check is what says the two never combine.
2. *That combination check is the parser's, and the duration's shape is the validator's.* This is the
   line US-003 drew and recorded (`tasks/completed/us-003-activate-an-automation-on-a-schedule.md`,
   "Consequences of that boundary" §1): the post-block completeness and arity diagnostics live in
   `parseAutomation` (`internal/parser/parser.go:1076-1084`), and a check on the *shape of a value*
   lives in the validator beside `scheduleExpressionDiagnostics`
   (`internal/validator/validator.go:370-383`). Both reach an author through `emod validate`, which is
   what the story's criteria ask for.
3. *The combination check fires only for an automation that states `every` and no `on`.* An automation
   stating both activation forms already reports "cannot declare both on and every"
   (`internal/parser/parser.go:1077-1078`); its `after` does qualify a real `on`, so re-reporting it
   would give one broken block two diagnostics for one mistake. Fixing the arity leaves the delay
   valid. This is the narrower reading of "schedule-driven", and it keeps `tasks/learnings.md` "An
   exactly-one-of block rule reports on top of the malformed entry's own diagnostic" from compounding.
4. *`after` written on its own line gets its own parser arm, not the fall-through.* It is a suffix, so
   it is not added to the automation's accepted-entry message; but a keyword the lexer tokenizes and
   the entry switch does not name falls to `default:` and is reported with a message naming no
   replacement — `tasks/learnings.md` "Retiring a keyword needs its own parser arm, not the
   fall-through" records exactly that failure, and the `trigger` arm
   (`internal/parser/parser.go:1097-1100`) is the shape. Breaking the activation line is the likeliest
   authoring mistake here, because `emod fmt` is what puts the two on one line.
5. *The delay rides on the automation **node** in the diagram document, not on the arrow.* The viewer
   gives every arrow endpoint repointing handles and lets a user delete an arrow, and `foldEdges`
   (`internal/importer/importer.go:244-281`) reads an `automation_trigger` edge back only when the
   node states no activation of its own. A value carried on the edge is therefore lost on the first
   save after a user removes or redraws that arrow, while node metadata survives with no edge at all —
   `tasks/learnings.md` "The viewer's save path is `importer.ImportDiagram`" records that the two
   channels are separate. The delay is a property of the automation; the arrow is where a *reader*
   looks for it, which is a rendering decision each format makes.
6. *There is a `Without…` twin here, unlike `every`.* Stripping a schedule leaves a model the parser
   rejects, which is why no `WithoutAutomationSchedules` exists. Stripping a delay leaves a valid
   event-activated automation, so the full six-part fixture kit applies and the differential receipts
   compare a delayed model against its own stripped twin rather than against a second model.
7. *Every automation in the new fixture reads a view.* `oracle.Check` runs the linter, and
   `automation/missing-todo-list` warns for an automation with no `reads`, which is why the
   `AutomationReads`, `TriggerReads` and `AutomationSchedule` kits sit in
   `internal/oracle/oracle_test.go`'s `"automations reading no view"` group (`:131`) rather than
   `"clean input"` (`:26`). Giving every automation a `reads` keeps the new fixture in `"clean input"`
   and keeps every downstream exit-0 assertion simple. The part deliberately omitted mid-block is the
   *delay*, which is what this story needs witnessed.

**Learnings folded in** from `tasks/learnings.md`: a line-oriented declaration must gate every
optional trailing token on the first token's line, and only a parse → format → reparse comparison
against the original model catches the failure; a new block entry keyword owes three things to the
parser's diagnostics; a keyword retired or repositioned needs its own parser arm, not the
fall-through; parser diagnostics at a stored `ast.Position` go through `p.errorAtPosition`; a quoted
block entry is one `case` on the `parse*EntryInto` family; an exactly-one-of block rule reports on top
of the malformed entry's own diagnostic; ask the lexer which keywords exist and never restate the set;
new DSL keywords must stay usable as field names, on the Go, tree-sitter and TextMate sides; keyword
surfaces fan out past the lexer, and every editor-highlight surface now has an executable suite run a
different way; a keyword that is only a keyword in one position never joins the flat TextMate
alternation; `highlights.scm` field patterns select by anchor; a tree-sitter highlight marker only
discriminates while another highlighted token follows it on the same line; put a new parser subtest in
the group that owns the construct; assert a short keyword in a diagnostic with a `\b`-bounded
`require.Regexp`; a second `require.Contains` on one message is often shadowed by the first; CLI
diagnostic tests must assert the distinguishing message text; `RuleName` marks a diagnostic
`emod lint --explain` can describe; diagnostics gathered from more than one AST collection must be
position-sorted; a new block entry goes after `description` and ahead of nested blocks in every
writer, and `emod fmt` silently deletes an entry it has never heard of; never write emod source with
`%q`; `emod fmt` canonicalises order, so a fmt golden is never the input re-indented, and formatter
output always begins with `emod N`; `emod fmt <file>` writes in place, so a receipt run dirties the
tree; a new optional field ships a six-part fixture kit; a new shared fixture owes `internal/oracle` a
zero-diagnostic subtest, and a lint warning fails `emod validate`; a slice has two homes; exercise an
omitted optional part mid-block, never as the last entry; a `Declared…` getter answers `nil` for a
fixture declaring none of the construct; `require.NotEqual` on a stripped twin is satisfiable without
stripping anything, and a differential receipt must first prove the twin differs; a new exported field
lands in JSON, CUE and `schema.cue` together, the two order their keys differently, `emittedKeyOrder`
makes the JSON order assertable, and a `*_position` key needs its value pinned to its own AST
position; read a decoded export document back with `objectsUnder`/`statedUnder`; the two export guards
cannot see a value neither writer emits; a serialized key spells the DSL keyword; a diagram-node key
has three readers that must move in one commit, and a key rename owes a retired-key negative
assertion; an arrow between two constructs is drawn by several surfaces; read an SVG or draw.io arrow
back as the two boxes it meets; `svgPicture` sees labelled boxes and arrows only; assert diagram box
placement relationally, by label; `ExportSVG` and `ExportASCII` ignore the `Style` they are handed;
the viewer shows a node twice, and `internal/viewer/static` is a display surface with its own vitest
harness; the tree-sitter grammar must never be stricter than the Go parser, its generated `src/` stays
gitignored, every `grammar.js` rule carries a one-line example of its full shape, and repo tooling
runs through `mise exec --`; a suite that pins another tool's output owes a mutated-input negative
control, and a test that shells out to a CLI runs with `-count=1`; an assertion whose expected value
comes from the code under test cannot fail; a task's change-set assertion must name every file its own
patterns require it to change; a "no expected constant moves" criterion is unsatisfiable when the task
edits a shared fixture; acceptance criteria describe the working tree, and a commit-message receipt is
the commit author's obligation, never a criterion.

---

## Codebase Context

**Lexer.** `internal/lexer/token.go` declares `Kind` in one iota block (`:10-71`, keywords at
`:12-49`, `KeywordOn` at `:48` and `KeywordEvery` at `:49`) and a lowercase `keywords` map (`:73-112`,
thirty-eight entries, `on` at `:110` and `every` at `:111`). `Keywords()` (`:124-126`) sorts the
spellings and `Kind.IsKeyword()` (`:135-138`) is a lookup in the `keywordNames` inversion (`:114`), so
nothing restates the set. Four subtests range over `Keywords()` —
`internal/lexer/tokenizer_test.go:13`, `internal/parser/parser_test.go:226` and `:242`,
`internal/oracle/oracle_test.go:45` — and a fifth, `TestEditorKeywordCoverage`
(`editors/tree-sitter-emod/test/queries/keywords_test.go:47`), ranges over it across the three editor
files.

**AST.** `ast.Automation` (`internal/ast/ast.go:200-218`) carries `Comments`, `Name`/`NamePos`,
`Description`/`DescriptionPos`, `OnEvent`/`OnEventPos` (`:206-207`), `Schedule`/`SchedulePos`
(`:208-209`), `Reads`/`ReadsPos`, `Command`/`CommandPos`, `TargetContext`/`TargetContextPos`,
`OpenPos`, `ClosePos` — every writer reads the struct in that sequence. The proposal puts
`After string`, `AfterPos Position` on it (`docs/proposals/specs-and-metadata-proposal.md:384`),
described there as "the delay qualifying `OnEvent`".

**Parser.** `parseAutomation` (`internal/parser/parser.go:1043-1087`) loops `parseAutomationEntry`
until `}`, then applies the post-block rules at the automation's stored `NamePos` through
`p.errorAtPosition` (`:1553`): both activation forms (`:1077-1078`), neither (`:1079-1080`), no
command (`:1082-1084`). `parseAutomationEntry` (`:1089-1133`) dispatches `on` to
`parseIdentifierEntryInto` (`:1432-1442`) and `every` to `parseQuotedEntryInto` (`:1419-1430`); both
helpers advance the keyword, require the value token, `errorAt` the offending one and drain with
`skipRestOfLineOrBlockEnd(keywordTok)` (`:1526-1530`), which also halts at `}` so the block still
closes. The `trigger` arm (`:1097-1100`) is the repo's one example of a keyword given a dedicated
`case` in order to say something better than the `default:` message at `:1130`
(`expected description, on, every, reads, command, or target in automation, got %q`). The line guards
are `checkSameLineAs` (`:1502-1504`), `checkIdentifierLike` (`:1506-1512`) and
`checkIdentifierLikeSameLineAs` (`:1514-1516`). Both `on` and `every` are last-wins.

**Parser tests.** `internal/parser/parser_test.go` is one umbrella; the `"automations"` group opens at
`:1945`, with the `on` subtests at `:1981` and `:2022`, the `every` subtests at `:2037` and `:2073`,
the side-by-side model at `:2088`, and the case-sensitivity loop over `{"On", "Every"}` at `:2124`.
`"error reporting"` owns the message and recovery shapes, including the `\b`-bounded accepted-entry
loop and its `require.NotRegexp` on `trigger`.

**Validator.** `Validate` (`internal/validator/validator.go:15-41`) calls
`scheduleExpressionDiagnostics(index.slices)` at `:32`; that check (`:370-383`) skips an empty
schedule, reports at `auto.SchedulePos` and carries no `RuleName`. `isWellFormedSchedule` (`:388-390`)
is `isGoDuration` (`:392-396`, `time.ParseDuration`) or `isCronExpression` (`:406-419`), and the
comment at `:385-387` records that the check is shape-only and deliberately does not range-check.
`errorAt` (`:150-158`) sets `Error` severity. `referenceDiagnostics` (`:297-323`) resolves an
automation's `target context`, `command`, `on` event (`:302`) and `reads` view through
`appendUndeclaredRef` (`:325-331`). Nothing in the validator reads `OnEvent` and `Schedule` together —
the mutual exclusion lives only in the parser. The validator test group `"schedule expressions"` opens
at `internal/validator/validator_test.go:1087`, with its model helper `modelSchedulingEvery` at
`:3291`.

**Formatter.** `writeAutomation` (`internal/formatter/formatter.go:347-357`) emits comments, the
header, `description`, then `on` through `lineIfSet` (`:25-30`, bare identifier, `:351`), `every`
through `quotedLineIfSet` (`:32-37`, `:352`), `reads`, `command` and `target context`. `quoted`
(`:61`) is the only correct way to emit emod text — the language has no escape sequences. The slice
writer calls it at `:185-189`. The automation goldens are `internal/formatter/formatter_test.go:366`
(the whole block), `:417` (an automation reading a view between its activation event and its command),
`:470` (entries written in canonical order wherever the source put them) and `:527` (a schedule
written under the description of the automation declaring it and invented for no sibling). The
round-trip group opens at `:702`; its shared-model table leaf is `:1037` (rows at `:1062`, assertions
at `:1082-1083`), its schedule leaf `:1130`, its quoting-hazard leaf `:1167`, and `:1090` is the twin
leaf asserting a `Without…` helper clears its construct in both slice homes and leaves the model it
was handed whole. `internal/cli/fmt_test.go` pins canonical `*FormattedEmod` constants
(`scheduledAutomationFormattedEmod` at `:446`) and feeds them to `requireFmtSettlesOn` (`:629`).

**Exports.** The package is split — there is no `internal/export/export.go`. `jsonAutomation`
(`internal/export/json.go:154-171`) opens with `name`, `description`, `position`, the five
`*_position` keys (`on_event_position` `:158`, `every_position` `:159`, `reads_position`,
`command_position`, `target_context_position`), `open_position`, `close_position`, `comments`, then
the five values (`on_event` `:166`, `every` `:167`, `reads`, `command`, `target_context`);
`convertAutomation` fills it at `:594-616`. `cueWriter.writeAutomation` (`internal/export/cue.go:197-206`)
emits `name`, `description`, `on_event`, `every`, `reads`, `target_context` through `lineIfSet`.
`internal/cue/schema.cue` `#Automation` (`:53-62`) is a closed definition declaring the same keys, so
an unknown key fails `cue vet`. The read-back walkers are `exportedAutomations`, `statedUnder`,
`objectsUnder` and `emittedKeyOrder` in `internal/export/export_test.go`, and the automation position
leaf is "includes positions for automation name/activation event/reads/command/target and braces".
`internal/cue/embed_test.go` vets `fullModelJSON` against `#Model` and carries the retired-key
negative leaves at `:112` and `:123`.

**Diagram graph.** `internal/diagram/graph.go` is new since US-003 and is the single place edges are
derived: `EdgeKind` (`:6-36`), `Edge{Kind, From, To}` (`:41-46`) and `SliceEdges(*ast.Slice)`
(`:51-115`). `EdgeAutomationTrigger` (`:17-19`) is emitted only when `a.OnEvent != ""` (`:86-88`), so
a scheduled automation contributes no arrow. Its doc comment states the point: deriving edges in one
place keeps every renderer and the diagram-JSON exporter describing the same picture. Three consumers
read it — `internal/diagram/drawio.go:411`, `internal/diagram/svg.go:175` and
`internal/export/diagram.go:376`. Mermaid and ASCII do not: they walk `s.Automations` themselves.

**Diagram labels and boxes.** `internal/diagram/labels.go` (25 lines) holds `gearMarking` and
`clockMarking` (`:5-8`), `reactorLabel` (`:10-12`), `cadenceLabel` (`:14-16`, renders `every "<s>"`)
and `automationLabel` (`:18-25`, appends the clock line only when `auto.Schedule != ""`).
`internal/diagram/layout.go` holds the shared box model — `reactorBox` (`:213-220`), `reactorBoxes`
(`:226-242`), `automationBox` (`:245-252`), `itemLayout` (`:264-274`), `laneRowY` (`:209-211`) — used
by draw.io (`drawio.go:314`) and SVG (`svg.go:126`) alike.

**Diagram renderers.** draw.io handles `EdgeAutomationTrigger` explicitly at `drawio.go:441-451`,
routing waypoints and calling `edgeCellWaypoints` (`:552-564`), which writes an `mxCell` with no
`value=`; `edgeCell` (`:546-550`) and `vertexCell` (`:513-527`) are the siblings that do carry a label
and a tooltip. SVG has no case for that kind — it falls into the `default:` branch (`svg.go:187-193`)
drawing `svgArrowBetween` (`:303-305`) / `svgArrowPath` (`:307-328`), which emits a `<path>` alone;
`svgText` (`:268-271`) and `svgMultilineText` (`:273-295`) are the text emitters. Mermaid renders an
automation as one `pcr` timeframe line whose label `mermaidAutomationLabel` (`mermaid.go:130-146`)
builds as `"%s (%s → %s)"` with the activation slot at `:137-140`. ASCII prints one chain per
automation at `ascii.go:85-88` with `activationMarking` (`:122-128`) filling the head.

**Diagram tests.** `internal/diagram/contract_test.go` runs one table across the four formats:
`exporter` (`:28-47`), `exporters()` (`:99-133`) — mermaid supplies only `export` and
`requireWellFormed`, ASCII adds `countConnections`, which counts occurrences of `" -> "` (`:128`).
`diagramConnection` (`:82-87`) names an arrow by the labels of the boxes at its ends plus the string
it is painted with, and **has no label field**. The differential receipts to copy are "declaring
invariants leaves the picture untouched" (`:427-439`) and "stating specs leaves the picture untouched"
(`:441-453`); `forEachSlice` (`:1043-1054`) visits both slice homes and every `without*` helper rides
on it. `TestExporterAutomationSchedule` (`:853-915`) is the schedule contract, reading a cadence off a
box label with `scheduleShown` (`:1142-1149`). The format readers are `drawioEdges`
(`drawio_test.go:994-1014`, whose `drawioEdge` regex at `:826` captures style, source and target only)
and `svgConnections` (`svg_test.go:462-492`, which maps path endpoints back to box centres through
`svgShapes` at `:354-399` — and `svgShapes` binds any `<text>` to the *last rect seen* at `:390-393`).
`ascii_test.go:396-412` asserts the gear is the one deliberate non-ASCII marker in that format, and
`libraryLendingAutomationChains` (`:19-26`) with `reactorChains` (`:28-37`) pins the chain lines.

**Diagram document and importer.** `jsonDiagramNode` (`internal/export/diagram.go:20-39`) carries
`on_event` (`:31`) and `every` (`:32`) among its type-specific keys; the automation node is built at
`:195-214`. `jsonDiagramEdge` (`:41-45`) is `{source, target, type}` and carries no label. The edge
conversion loop (`:375-424`) maps `EdgeAutomationTrigger` to `"automation_trigger"` at `:405-408`.
`internal/importer/importer.go` reverses it: `diagramNode` (`:21-38`, `on_event` `:30`, `every` `:31`),
`buildSlice`'s automation loop (`:197-208`), and `foldEdges` (`:244-281`) whose `automation_trigger`
case (`:262-268`) fills `OnEvent` from the edge only when the node states neither activation form.

**Viewer.** `internal/viewer/static/renderer.js` draws the clock badge at `:176-183` inside
`appendBlockLabels` (`:159-186`), inserting `svgTitle(node.every)` as the group's first child (`:179`);
`clockMarking` is at `:5`. The edge loop (`:325-376`) appends a hit path, a path and two repoint
handles and renders **no** edge label of any kind; paint comes from `edgeConfig`
(`static/config.js:39-47`, `automation_trigger` at `:42`). `static/model.js:142-151` types an
interactively drawn arrow, `"event>automation"` at `:145`. `static/ui.js:350-361` is the automation
section of the details panel — rows `On Event` (`:354`), `Every` (`:355`), `Reads`, `Command`,
`Target Context`, each falling back to an em dash. Tests are vitest on jsdom under
`internal/viewer/tests`, run by `task test:viewer`, which is not part of `task test:unit`;
`renderer.test.js:72-74` and `:238` cover the clock badge, `detail-panel.test.js` the panel rows.

**Editor grammars.** `automation_definition` (`editors/tree-sitter-emod/grammar.js:301-312`) passes
`seq('on', $.any_identifier)` (`:306`) and `seq('every', $.string)` (`:307`) to `buildDescribedBlock`,
under a one-line comment (`:300`) spelling the construct out whole. `any_identifier` is what keeps
keywords usable as field names, and `field_line` uses `optional` plus `prec.right` for its modifier —
so an optional trailing token *inside a one-line entry* is the established shape, distinct from the
forbidden `optional($.rule)` in a block body. Corpus cases live in `test/corpus/slice.txt` and
`test/corpus/fields.txt`; `src/` is gitignored. `queries/highlights.scm` keeps a hand-written
`@keyword` alternation (`:18-60`) whose header comment names `TestEditorKeywordCoverage` as its guard,
and `test/highlight/unreserved-keywords.emod` asserts each keyword stays a field name, type and
modifier, every marked token followed by another highlighted token on its line.
`editors/vscode/syntaxes/emod.tmLanguage.json` keeps `on` and `every` out of the positionless
`#keywords` alternation (`:95-97`) and colours them in a case-sensitive `#activations` rule (`:67-84`)
keyed on the operand — `on` before an identifier (`:72`), `every` before a quoted string (`:80`); the
`fields` block rule deliberately omits `#activations`. Its assertion files are
`editors/vscode/test/scopes/activations.emod` and `unreserved-keywords.emod`, run by
`task test:vscode`.

**Fixtures.** `internal/test/fixtures.go` holds `HotelReservation` (`:13`),
`DescribedHotelReservation` (`:101`), `KeywordFieldSearchCatalog` (`:210`), `InvariantLibraryLending`,
`SpecLibraryLending`, `AutomationReadsLibraryLending` (`:581`), `TriggerReadsLibraryLending` (`:749`)
and `AutomationScheduleLibraryLending` (`:947`, doc `:939-946`) — the last of which mixes a duration
and a cron expression in each of the two slice homes and places an event-activated automation
mid-block on purpose. The transcribed lists are at `:1135`, `:1146`, `:1159`, `:1169`, `:1181` and
`:1192`; the getters `DeclaredAutomationReads` (`:1313`), `DeclaredActivationEvents` (`:1337`) and
`DeclaredSchedules` (`:1345`) compose `declaredAutomationEntries` (`:1352-1362`, which skips
automations stating nothing) over `declaredSlices` (`:1364`). The twins `WithoutSpecs` (`:1216`),
`WithoutAutomationReads` (`:1228`) and `WithoutTriggerReads` (`:1243`) ride on `copyWithEditedSlices`
(`:1257`) and `editedCopies` (`:1275`). `internal/test/models.go` holds one `…Model(t)` accessor per
fixture. `internal/oracle/oracle_test.go` splits its fixtures between `"clean input"` (`:26`) and
`"automations reading no view"` (`:131`), the latter asserting the exact warning lines.

**LSP.** `internal/lsp/keywords_test.go` (build tag `unit`) holds `TestKeywordCoverage`, whose header
comment records that the hover descriptions and the completer's per-block lists are both hand-written
and derived from nothing. Its two halves guard in opposite directions: `hover` (`:19-57`) iterates
`lexer.Keywords()` and requires `lsp.GetHover` to answer for every one, while `completion` (`:59-75`)
iterates `keywordBlocks` and requires every offered label to *be* a lexer keyword. So a new keyword
owes `keywordDescriptions` (`internal/lsp/hover.go:10-49`, `on` at `:23` and `every` at `:24`) an
entry and owes `valueSlots` (`completer.go:176`, holding `on` and `reads` only) and `keywordsFor`
(`:223`) nothing. `internal/lsp/model.go` is the package's single AST traversal, and `referencesIn`
(`:115-120`) lists the sites naming a declared construct — a duration names none.

**CLI.** `automationActivation` (`internal/cli/slices_list.go:125-130`) is the single helper choosing
between `every "<s>"` and the activation event for the `KEY ELEMENTS` column, used by both the text
and JSON listings through `keyElementsForPattern` (`:132`, automation arm at `:158-163`).

---

## Tasks

### Task 1: Accept and colour `after` across the three editor grammars

**Behavior:** an editor backed by any of the repo's three grammars accepts and colours
`on <Event> after "<duration>"` and `every "<expr>" after "<duration>"` before the Go tool does. The
tree-sitter grammar parses the suffix on both activation entries with no error node, the tree-sitter
highlight query and the VS Code TextMate grammar colour `after` as a keyword where it qualifies an
activation, and a field named `after` keeps its field colouring in all three. Landing the editor
surfaces first is what keeps every commit green: `TestEditorKeywordCoverage` requires all three to
name every lexer keyword, so the commit that teaches the lexer `after` would otherwise be red.

**Acceptance Criteria:**
- [ ] `automation_definition` (`editors/tree-sitter-emod/grammar.js:301-312`) admits an `after` suffix
      taking a string on the `on` entry and on the `every` entry, written inside the entry's own `seq`
      rather than as a new item in the block body — the block stays unordered and unbounded
- [ ] The one-line comment above that rule (`:300`) spells the construct out whole including the
      suffix, so the file's only description of an automation still lists everything it admits
- [ ] A corpus case in `editors/tree-sitter-emod/test/corpus/slice.txt` covers an automation whose
      `on` entry carries `after`, and its expected tree contains no `ERROR` or `MISSING` node
- [ ] A second corpus case covers an `every` entry carrying `after`, which the Go parser will reject
      and the grammar must still parse — the grammar is never stricter than the language
- [ ] A third corpus case covers an automation whose `on` entry carries no `after`, written ahead of a
      further entry rather than last, and parses unchanged
- [ ] The keyword-per-field corpus case in `editors/tree-sitter-emod/test/corpus/fields.txt` gains a
      field named `after`, one typed `after` and one modified `after`, all parsing as field lines
      rather than as activation suffixes
- [ ] `editors/tree-sitter-emod/queries/highlights.scm` names `after` in its `@keyword` list
      (`:18-60`), and `editors/tree-sitter-emod/test/highlight/unreserved-keywords.emod` asserts
      `after` in the three field positions it already asserts `on` and `every` in, each marked token
      followed by another highlighted token on the same line
- [ ] `editors/vscode/syntaxes/emod.tmLanguage.json` colours `after` from the case-sensitive
      `#activations` rule (`:67-84`) keyed on its operand, not from the positionless `#keywords`
      alternation (`:95-97`) — an automation named `After` and a field named `after` must both keep
      their own scopes
- [ ] `editors/vscode/test/scopes/activations.emod` asserts the keyword scope on `after` in a
      delayed automation and the string scope on the duration beside it, and
      `editors/vscode/test/scopes/unreserved-keywords.emod` asserts `after` carries no keyword scope
      as a field name, a field type or a field modifier
- [ ] `mise exec -- task test:grammar` and `mise exec -- task test:vscode` both pass, and running
      `task test:grammar` a second time leaves every tracked file under `editors/tree-sitter-emod/`
      byte-identical
- [ ] `git ls-files editors/tree-sitter-emod/src` returns nothing — the generated parser stays
      untracked
- [ ] `mise exec -- task test:unit` passes with no Go file changed by this task

**Affected Files/Modules:**
- `editors/tree-sitter-emod/grammar.js` — `automation_definition` (`:301-312`) and its comment (`:300`)
- `editors/tree-sitter-emod/test/corpus/slice.txt`, `test/corpus/fields.txt` — the new cases
- `editors/tree-sitter-emod/queries/highlights.scm` — the `@keyword` list (`:18-60`)
- `editors/tree-sitter-emod/test/highlight/unreserved-keywords.emod` — the field-position assertions
- `editors/vscode/syntaxes/emod.tmLanguage.json` — the `#activations` rule (`:67-84`)
- `editors/vscode/test/scopes/activations.emod`, `test/scopes/unreserved-keywords.emod`

**Patterns to Follow:**
- The string-valued activation entry and its comment: `seq('every', $.string)`
  (`editors/tree-sitter-emod/grammar.js:307`), and `tasks/learnings.md` "Every `grammar.js` rule
  carries a one-line example of its full shape"
- An optional trailing token inside a one-line entry: `field_line`
  (`editors/tree-sitter-emod/grammar.js`), which uses `optional` with `prec.right`. This is not the
  shape `tasks/learnings.md` "The tree-sitter grammar must never be stricter than the Go parser"
  forbids — that entry rules out `optional($.rule)` as a *block body item*, and a reviewer reading
  this diff should be told which of the two it is
- If the new anonymous token competes with `any_identifier` in field position, narrow the token the
  way `version_header` narrows `emod` rather than loosening the field rule — `tasks/learnings.md`
  "New DSL keywords must stay usable as field names"
- The TextMate placement decision and its two constraints (case sensitivity, and `fields` omitting
  `#activations`): `tasks/learnings.md` "A keyword that is only a keyword in one position never joins
  the flat TextMate alternation", with `every` at
  `editors/vscode/syntaxes/emod.tmLanguage.json:78-82` as the operand-keyed sibling
- Highlight-marker placement: `tasks/learnings.md` "A tree-sitter highlight marker only discriminates
  while another highlighted token follows it on the same line", and the header comment of
  `editors/tree-sitter-emod/test/highlight/unreserved-keywords.emod` saying so
- `tasks/learnings.md` "Every editor-highlight surface now has an executable suite, and each is run a
  different way", "Run repo tooling through `mise exec --`, not bare PATH", "Generated tree-sitter
  `src/` stays gitignored" and "A test that shells out to a CLI runs with `-count=1`"

**Testable:** Yes — the corpus cases, the highlight assertion file and the VS Code scope assertion
files are the tests, run by `task test:grammar` and `task test:vscode`.

**Verification:** `mise exec -- task test:grammar` (twice, the second run leaving the tree clean);
`mise exec -- task test:vscode`; `git status --porcelain -- '*.go'` is empty.

**Depends on:** None

---

### Task 2: Parse `after "<duration>"` on an automation's activation line

**Behavior:** `after` becomes a keyword, and an automation's `on` entry accepts an optional
`after "<duration>"` suffix written on the same line, recording the text between the quotes and the
position of the string token. The same suffix parses on an `every` entry so the combination is
expressible, and the parser reports it once, at the delay, saying that a schedule is already absolute:
a schedule-driven automation cannot also state a relative delay. `after` written as an entry of its
own gets a message saying it qualifies an activation rather than the generic list of accepted entries.
An automation stating no delay parses exactly as it does today. Because `after` joins the lexer's
keyword map it is simultaneously usable as a field name, a field type and a field modifier, and it
describes itself on hover in an LSP-capable editor — the hover map is hand-written, and the assertion
that guards it iterates the lexer's keywords, so the word arrives there in the same change.

**Acceptance Criteria:**
- [ ] An automation declaring `on RoomHeld after "24h"` and a command parses with no diagnostics and
      carries `RoomHeld` as its activation event and `24h` as its delay, together with the filename,
      line and column of the delay's string token
- [ ] An automation declaring `on RoomHeld` and no delay parses with no diagnostics and carries an
      empty delay, asserted on the same model as the delayed automation so both are read back together
- [ ] An automation whose `on` entry states no delay, written *above* a further entry in the same
      block, does not absorb the entry on the next line: the following entry is parsed onto the
      automation and the delay stays empty
- [ ] A source file writing `after "24h"` on the line below an `on` entry reports exactly one
      diagnostic (`require.Len(t, diags, 1)`) whose message names `after` and points the author at the
      activation line it qualifies; the automation still closes and any entry below it is still parsed
- [ ] The unrecognised-entry message inside an automation is unchanged — `after` is a suffix, not an
      entry, so the accepted-entry loop in `internal/parser/parser_test.go` still names exactly
      `description`, `on`, `every`, `reads`, `command` and `target`, each with a `\b`-bounded
      `require.Regexp`, and a `require.NotRegexp` pins that it does not name `after`
- [ ] An automation declaring `every "0 2 * * *" after "24h"` and a command reports exactly one
      diagnostic, positioned at the delay rather than at the automation's name or the keyword, whose
      message names `after` and `every` under `\b` bounding and states that a schedule is already
      absolute
- [ ] An automation declaring `on RoomHeld after "24h"` alongside `every "5m"` reports exactly one
      diagnostic — the existing "cannot declare both on and every" — and not a second about the delay:
      the delay qualifies a stated `on`, so fixing the arity leaves it valid
- [ ] An `after` whose value is a bare identifier rather than a quoted string, with a `command` entry
      on the following line, reports exactly one diagnostic naming `after` and `automation`, and the
      `command` entry on the following line is still parsed onto the automation
- [ ] An `after` with nothing after it, written as the last entry of a block followed by a second
      automation, reports exactly one diagnostic, and `require.NotZero` holds on the automation's, the
      slice's and the context's `ClosePos.Line` with the second automation parsed
- [ ] An automation declaring `after` twice on one activation line keeps the value written last
- [ ] `lexer.Keywords()` contains `after`, `lexer.Scan` yields a token for it whose `Kind` is not
      `Identifier`, and a field named `after` with type `after` and modifier `after` parses with no
      diagnostics — all from the subtests that range over `Keywords()`
      (`internal/lexer/tokenizer_test.go:13`, `internal/parser/parser_test.go:226`, `:242`,
      `internal/oracle/oracle_test.go:45`), which pass unedited
- [ ] A model declaring an event named `After` and an automation activating on it parses with no
      diagnostics, pinning that keyword lookup stays case-sensitive
- [ ] `keywordDescriptions` (`internal/lsp/hover.go:10-49`) describes `after`, naming it as the delay
      qualifying an activation event rather than a schedule, and `TestKeywordCoverage/hover`
      (`internal/lsp/keywords_test.go:19-57`) passes for every spelling `lexer.Keywords()` reports
      with no edit to that test — hovering `after` returns non-nil hover with non-empty contents
- [ ] `TestKeywordCoverage/completion` (`internal/lsp/keywords_test.go:59-75`) passes unedited with no
      `after` entry added to `valueSlots` or `keywordsFor` (`internal/lsp/completer.go:176`, `:223`):
      it requires every label a block offers to be a lexer keyword, not the reverse, so completion
      stays US-015's
- [ ] `oracle.Check` returns the diagnostics it returns today for every fixture in
      `internal/test/fixtures.go`, with those fixtures unedited — no model in the repo states a delay
      yet
- [ ] `mise exec -- task test:grammar` passes, which is the receipt that Task 1 gave all three editor
      surfaces the spelling `TestEditorKeywordCoverage` now demands of them

**Affected Files/Modules:**
- `internal/lexer/token.go` — the `Kind` iota block (`:10-71`) and the `keywords` map (`:73-112`)
- `internal/ast/ast.go` — `Automation` (`:200-218`) gains the delay and its position, placed to match
  the field order every writer reads
- `internal/parser/parser.go` — `parseAutomationEntry` (`:1089-1133`): the `on` arm (`:1093-1094`) and
  the `every` arm (`:1095-1096`) gain the trailing suffix, a new arm answers `after` written as an
  entry, and `parseAutomation`'s post-block switch (`:1076-1084`) gains the combination rule
- `internal/parser/parser_test.go` — the `"automations"` group (`:1945`) and the message and recovery
  subtests in `"error reporting"`
- `internal/lsp/hover.go` — a `keywordDescriptions` entry (`:10-49`), which `TestKeywordCoverage/hover`
  demands of the change that teaches the lexer the word. No file under `internal/lsp` beyond this map
  is touched: `completer.go` is US-015's, and `diagnostics.go` carries a pre-existing severity mapping
  this story neither uses nor changes

**Patterns to Follow:**
- The quoted-value entry and its recovery: `parseQuotedEntryInto` (`internal/parser/parser.go:1419-1430`)
  — it interpolates the consumed keyword's own spelling into its message, so a new caller moves no
  existing diagnostic text — and `skipRestOfLineOrBlockEnd` (`:1526-1530`), which halts at `}` so the
  block still closes. `tasks/learnings.md` "A quoted or identifier block entry is one `case` on the
  `parse*EntryInto` family" and "A new block entry keyword owes three things to the parser's
  diagnostics", including the `require.Len(t, diags, 1)` pin
- The same-line guard, positioned against the *first* token of the declaration:
  `checkSameLineAs` (`internal/parser/parser.go:1502-1504`) and `checkIdentifierLikeSameLineAs`
  (`:1514-1516`). `tasks/learnings.md` "A line-oriented declaration must gate every optional trailing
  token on the first token's line" is the entry this task exists inside — it records that formatter
  idempotence does not catch the failure and that only a parse → format → reparse comparison against
  the original model does (Task 4 carries that receipt), and that the omitted optional part must be
  exercised mid-block
- A keyword given its own arm rather than the fall-through: the `trigger` arm
  (`internal/parser/parser.go:1097-1100`) and `tasks/learnings.md` "Retiring a keyword needs its own
  parser arm, not the fall-through, and three closing braces as the receipt", whose `ClosePos.Line`
  assertions are the shape for proving a drain stopped in the right place
- The post-block rules and where they report: `parseAutomation` (`internal/parser/parser.go:1076-1084`)
  through `p.errorAtPosition` (`:1553`) — `tasks/learnings.md` "Parser diagnostics at a stored
  `ast.Position` go through `p.errorAtPosition`". The delay's own position is the more useful anchor
  here than the automation's name, since the author's mistake is on that token
- `tasks/learnings.md` "An exactly-one-of block rule reports on top of the malformed entry's own
  diagnostic" — a test wanting exactly one diagnostic out of a malformed activation line must give
  the block a valid sibling activation entry
- `tasks/learnings.md` "Assert a short keyword in a diagnostic with a `\b`-bounded `require.Regexp`",
  which already names `every` as one of the remaining short keywords; and "A second `require.Contains`
  on one message is often shadowed by the first"
- `tasks/learnings.md` "Ask the lexer which keywords exist; never restate the set" — add the spelling
  to the map and append the `Kind`; `checkIdentifierLike` (`:1506-1512`) already asks `IsKeyword()`
- `tasks/learnings.md` "Keyword surfaces fan out past the lexer, and each is run a different way" —
  the surfaces are now guarded by two suites running in opposite build configurations
  (`TestEditorKeywordCoverage` under `//go:build grammar`, `TestKeywordCoverage/hover` under `unit`),
  so `go test -tags unit ./internal/lsp/...` and `mise exec -- task test:grammar` are two separate
  receipts and neither substitutes for the other. The existing hover entries for `on` and `every`
  (`internal/lsp/hover.go:23-24`) fix the voice for the new one: one sentence naming what the keyword
  states, with `every`'s spelling out its accepted forms
- `tasks/learnings.md` "Put a new parser subtest in the group that owns the construct" — the construct
  is the automation, so `"automations"` (`internal/parser/parser_test.go:1945`) owns the shape
  subtests and `"error reporting"` owns the messages and the recovery

**Testable:** Yes — through `lexer.Scan`, `parser.Parse` and `oracle.Check`, all exported.

**Verification:** `go test -tags unit ./internal/lexer/... ./internal/parser/... ./internal/oracle/...
./internal/lsp/...`; `go build ./...`; `mise exec -- task test:grammar`. The last two of those are the
two drift suites, and each answers for surfaces the other cannot see.

**Depends on:** Task 1

---

### Task 3: Reject an `after` value that is not a Go duration

**Behavior:** `emod validate` reports an automation whose delay is not Go duration syntax, positioned
at the delay itself, with a message naming the accepted form and quoting what was written. What is
checked is that the text parses as a duration; the value it denotes is not judged, so a zero or
negative duration is accepted the same way an out-of-range cron field already is — the model states a
delay that nothing here evaluates.

**Acceptance Criteria:**
- [ ] A model whose automation declares `after "30m"`, one declaring `after "24h"`, one declaring
      `after "72h"` and one declaring `after "1h30m"` validate with no diagnostics
- [ ] A model whose automation declares `after "tomorrow"` reports exactly one diagnostic whose
      message quotes `tomorrow` and names the Go duration form, positioned at the line and column of
      the delay's string token — not of the `after` keyword, the `on` keyword or the automation's name
- [ ] A model whose automation declares `after "24 hours"` and one declaring `after "24"` are each
      reported the same way, so a near-miss on the duration form is not silently accepted
- [ ] One model carrying an automation with `after "24h"` beside one with `after "24 hours"` produces
      exactly one diagnostic, naming the second — the accepted and the rejected case are pinned in the
      same assertion
- [ ] A model whose automation declares `after "0s"` and one declaring `after "-5m"` validate with no
      diagnostics, pinning that the duration's value is not judged, only its syntax
- [ ] An automation stating `on` and no delay produces no diagnostic from this check, asserted on a
      model that also carries a rejected delay so the check is proved to be running
- [ ] A model with two malformed delays reports them in declaration order across both slice homes —
      an aggregate's slices before the slices a `mode dcb` context declares directly — identical
      across repeated runs of `validator.Validate` over the same model, asserted with one
      `require.Equal` over the reported lines
- [ ] The printed diagnostic carries no `[rule]` bracket and no rule name resolvable through
      `emod lint --explain`: what no configuration can silence is a hard error, not a rule
- [ ] `cli.RunValidate` on a file with a malformed delay returns an error whose message names the
      offending text and the accepted form — the same distinguishing content the validator test one
      layer down asserts, not merely the path and a line number

**Affected Files/Modules:**
- `internal/validator/validator.go` — a check beside `scheduleExpressionDiagnostics` (`:370-383`),
  reached from `Validate` (`:15-41`, the schedule check is wired at `:32`)
- `internal/validator/validator_test.go` — a group beside `"schedule expressions"` (`:1087`), with a
  model helper in the shape of `modelSchedulingEvery` (`:3291`)
- `internal/cli/validate_test.go` — one leaf for the diagnostic as the user receives it

**Patterns to Follow:**
- The directly analogous check, its position, its severity and its empty `RuleName`:
  `scheduleExpressionDiagnostics` (`internal/validator/validator.go:370-383`), `isGoDuration`
  (`:392-396`), `errorAt` (`:150-158`), and the comment at `:385-387` recording that the check is
  shape-only by decision rather than by omission
- `tasks/learnings.md` "`RuleName` marks a diagnostic `emod lint --explain` can describe" — a hard
  error carries no rule name
- `tasks/learnings.md` "Diagnostics gathered from more than one AST collection must be
  position-sorted", and the `require.Equal`-against-reported-lines assertion it names, which shows the
  whole list on failure where `require.Len` plus one `String()` cannot catch a misordering
- `tasks/learnings.md` "A rule whose message branches on model state is pinned by whole formatted
  lines" and "A second `require.Contains` on one message is often shadowed by the first"
- `tasks/learnings.md` "CLI diagnostic tests must assert the distinguishing message text", with
  `internal/cli/validate_test.go` as the model, and "urfave/cli v2 discards every flag written after
  the file argument" for any invocation written with a flag after the path
- The accepted form comes from the story and `docs/proposals/specs-and-metadata-proposal.md:315`: a Go
  duration string, `"30m"` / `"24h"` / `"72h"`

**Testable:** Yes — through `validator.Validate` and `cli.RunValidate`.

**Verification:** `go test -tags unit ./internal/validator/... ./internal/cli/...`.

**Depends on:** Task 2

---

### Task 4: Emit `after` from `emod fmt`

**Behavior:** `emod fmt` writes an automation's delay as an `after "<duration>"` suffix on the
activation line, so a delayed automation survives formatting and formatting again is stable. The
suffix follows whichever activation form the automation states, so a file the parser rejected for
combining `every` with a delay still round-trips rather than silently losing the clause an author
wrote. An automation stating no delay produces exactly the bytes it produces today.

**Acceptance Criteria:**
- [ ] Formatting an automation that declares `on RoomHeld after "24h"` emits both on one line, single
      spaced, with the duration between plain quotes, at the same indent as the sibling entries and in
      the position `on` already occupies
- [ ] An automation carrying a delay and a schedule but no activation event — the shape the parser
      rejects — still writes its delay, as a suffix on the `every` line, so formatting never deletes
      the clause
- [ ] A model whose slice declares a delayed automation beside an automation stating `on` and no
      delay formats to canonical bytes for both, asserted against one expected whole-block output
      rather than by searching for a line
- [ ] Formatting the same model twice produces identical bytes, and a duration containing a backslash,
      a tab, a double quote, a `%` and a non-ASCII character survives parse → format → parse → format
      with identical bytes, proving the text is never escaped
- [ ] Parsing a delayed automation, formatting the model and reparsing the result yields a model equal
      to the original under the round-trip comparison at `internal/formatter/formatter_test.go:702` —
      the assertion class that catches a declaration running on into the next line, which no golden and
      no idempotence check can catch
- [ ] A source file writing `after` with the author's own spacing, and one writing the activation entry
      after `command`, are both rewritten by `RunFmt` to the canonical position and settle there —
      pinned with a canonical `*FormattedEmod` constant passed to `requireFmtSettlesOn`
      (`internal/cli/fmt_test.go:629`), never by handing the input fixture back as the expectation
- [ ] Every expected constant this task adds opens with the `emod <n>` version header
- [ ] `git diff` shows no change to any existing expected constant in
      `internal/formatter/formatter_test.go` or `internal/cli/fmt_test.go`: no automation in either
      file states a delay, so no byte of their output may move

**Affected Files/Modules:**
- `internal/formatter/formatter.go` — `writeAutomation` (`:347-357`), whose `on` line (`:351`) and
  `every` line (`:352`) each gain the suffix
- `internal/formatter/formatter_test.go` — the automation goldens (`:366`, `:417`, `:470`, `:527`),
  the round-trip group (`:702`) and its quoting-hazard leaf (`:1167`)
- `internal/cli/fmt_test.go` — a canonical constant beside the existing ones
  (`scheduledAutomationFormattedEmod` at `:446`) and its `requireFmtSettlesOn` leaf

**Patterns to Follow:**
- The two line writers the suffix attaches to, and the guard each applies: `lineIfSet`
  (`internal/formatter/formatter.go:25-30`) and `quotedLineIfSet` (`:32-37`), which tests the raw
  string it is handed — a pre-quoted value defeats its guard
- `quoted` (`internal/formatter/formatter.go:54`) for every string, never `%q` — `tasks/learnings.md`
  "Never write emod source with `%q` — the language has no escape sequences", including its
  counterpart obligation of a round-trip subtest per hazard character
- `tasks/learnings.md` "A new block entry goes after `description` and ahead of nested blocks, in
  every writer", whose warning is that the formatter renders from the AST and silently deletes what it
  has never heard of, and that the guard is a parse → format → reparse comparison against the original
  model
- `tasks/learnings.md` "`emod fmt` canonicalises order, so a fmt golden is never the input
  re-indented" and "Formatter output always begins with `emod N`"
- Alignment across the activation keywords is US-014's; this task writes single-spaced entries and
  moves no existing column
- `tasks/learnings.md` "`emod fmt <file>` writes in place, so a receipt run dirties the working tree" —
  copy to a temp path, or `git checkout --` afterwards

**Testable:** Yes — through `formatter.Format` and `cli.RunFmt`.

**Verification:** `go test -tags unit ./internal/formatter/... ./internal/cli/...`.

**Depends on:** Task 2

---

### Task 5: Share a fixture whose automations fire after an elapsed delay

**Behavior:** one shared model states delays in both homes a slice has — nested in an aggregate and
directly on a `mode dcb` context — beside automations that activate immediately and beside one that
runs on a schedule, so every package downstream reads delayed, undelayed and scheduled automations out
of one source instead of writing its own. The fixture is a model `emod validate` and `emod lint` both
accept, it survives a format round-trip with every delay intact, and it comes with a twin that strips
only the delays so later differentials can compare it against itself.

**Acceptance Criteria:**
- [ ] `internal/test/fixtures.go` gains a fixture whose automations state `after` in both slice homes,
      with at least two different durations, beside an automation stating `on` and no delay and an
      automation stating `every` and no delay — one model in which the two timing notations and the
      absence of both are all readable
- [ ] The automation stating `on` and no delay sits *mid-block* ahead of a further automation, never
      as the last entry: an entry that runs on into what follows it is only caught when something
      follows it
- [ ] Every automation in the fixture states a `reads`, so `automation/missing-todo-list` stays quiet
      and the fixture's `oracle.Check` leaf belongs in the `"clean input"` group
      (`internal/oracle/oracle_test.go:26`) rather than `"automations reading no view"` (`:131`)
- [ ] `oracle.Check` over the fixture returns no diagnostics at all — lexer, parser, validator and
      linter each accept it, and a `mode dcb` context in it carries the tags and `decides_on` its
      events need
- [ ] `internal/test/models.go` gains the parsing accessor for it, in the shape of
      `AutomationScheduleLibraryLendingModel`
- [ ] A hand-transcribed exported list names every delay the fixture states, both slice homes together
      and in declaration order, and is non-empty
- [ ] A `Declared…` getter walks `declaredSlices` (`internal/test/fixtures.go:1364`) and returns the
      delay every automation states, skipping the automations that state none, and `require.Equal`
      against the transcribed list holds — a getter reaching only one slice home reads back short
- [ ] `test.DeclaredActivationEvents` and `test.DeclaredSchedules` over the same fixture each return a
      non-empty list, so the fixture is proved to carry all three activation shapes
- [ ] A `Without…` twin clears the delays and nothing else: `require.Empty` on the twin's delays,
      `require.Equal` against the transcribed list for the fixture itself, and the twin's activation
      events and schedules still equal to the fixture's — a strip reaching further than the delays
      cannot pass
- [ ] The twin returns a copy, so the model handed to it reads back unchanged afterwards; the copy is
      shallow, so the edit reaching inside a slice's automations nests a second `editedCopies`
- [ ] The formatter round-trip group gains this fixture in its existing per-fixture leaf
      (`internal/formatter/formatter_test.go:1037`, whose table rows sit at `:1062` and whose
      assertions sit at `:1082-1083`), asserting the reparsed model reads back the transcribed delays
      alongside the activation events and schedules — one row, not a parallel table
- [ ] `git diff` leaves every existing fixture in `internal/test/fixtures.go` untouched, and moves no
      transcribed name list and no `*FormattedEmod` constant belonging to another fixture: the models
      stating no delay are this story's byte-identical witnesses

**Affected Files/Modules:**
- `internal/test/fixtures.go` — the fixture const, the transcribed list, the getter and the twin,
  beside `AutomationScheduleLibraryLending` (`:947`), `…Schedules` (`:1181`), `DeclaredSchedules`
  (`:1345`) and `WithoutAutomationReads` (`:1228`)
- `internal/test/models.go` — the accessor
- `internal/oracle/oracle_test.go` — one leaf in `"clean input"` (`:26`)
- `internal/formatter/formatter_test.go` — the shared-model round-trip table leaf (`:1037`)

**Patterns to Follow:**
- `tasks/learnings.md` "A new optional field ships a six-part fixture kit, not a bespoke model per
  package" — the fixture const, the `…Model(t)` accessor, the transcribed list, the `Without…` twin,
  the `Declared…` getter and the `oracle.Check` leaf. `AutomationScheduleLibraryLending`
  (`internal/test/fixtures.go:947`, doc `:939-946`) is the model to repeat, including its doc comment
  explaining why the omission sits mid-block; unlike a schedule, a delay *can* be stripped, so this
  kit has the twin the schedule kit could not
- The twin machinery: `WithoutAutomationReads` (`internal/test/fixtures.go:1228-1235`),
  `copyWithEditedSlices` (`:1257`), `editedCopies` (`:1275`) — which leaves a nil list nil on purpose,
  and whose copies are shallow, so an edit reaching inside a slice's automations must nest a second
  call or it writes through to the caller's model
- `tasks/learnings.md` "`require.NotEqual` on a stripped twin is satisfiable without stripping
  anything" and "A differential receipt must first prove the twin actually differs" — state the twin's
  emptiness *and* the fixture's transcribed list, and state what the twin must still carry. The
  round-trip leaf at `internal/formatter/formatter_test.go:1090` already asserts a twin clears its
  construct in both slice homes and leaves the model it was handed whole, and is the shape to copy
- `tasks/learnings.md` "A slice has two homes", "Exercise an omitted optional part mid-block, never as
  the last entry", and "A `Declared…` getter answers `nil` for a fixture that declares none of the
  construct" — pair the getter only with the non-empty transcribed list, and fold the round-trip
  assertion into the existing per-fixture leaf
- `tasks/learnings.md` "A new shared fixture owes `internal/oracle` a zero-diagnostic subtest", with
  its warning that DCB shapes are the usual tripwire, and "A lint warning fails `emod validate`", which
  is why every automation here reads a view
- `tasks/learnings.md` "A 'no expected constant moves' criterion is unsatisfiable when the task edits a
  shared fixture" — this task adds a fixture rather than editing one, which is what keeps that
  criterion satisfiable
- `declaredAutomationEntries` (`internal/test/fixtures.go:1352-1362`) already generalises the getter

**Testable:** Yes — through `oracle.Check`, `formatter.Format` and the exported getters.

**Verification:** `go test -tags unit ./internal/test/... ./internal/oracle/... ./internal/formatter/...`;
`go run ./cmd/emod validate` over a temporary copy of the fixture, expecting exit 0.

**Depends on:** Tasks 3, 4

---

### Task 6: Carry the delay into the JSON, CUE and embedded schema exports

**Behavior:** `emod export` names an automation's delay in both formats and the embedded schema
declares it, so a consumer of either document reads the delay the author wrote and knows where in the
automation it was written. An automation stating no delay exports exactly what it exported before.

**Acceptance Criteria:**
- [ ] The JSON export of the Task 5 fixture states every delay the transcribed list names, read back
      with `statedUnder(exportedAutomations(doc), …)` in the writer's slice order, and states the
      transcribed activation events and schedules under their existing keys in the same document
- [ ] The exported key spells the DSL keyword, and its position key is that spelling plus `_position`
- [ ] `emittedKeyOrder` shows the automation object filing the delay's position key among the other
      `*_position` keys and its value among the other values, matching the order a `json*` sibling
      uses — the sibling's key list asserted in the same subtest so the expectation is not arbitrary
- [ ] The automation position leaf ("includes positions for automation name/activation event/reads/
      command/target and braces", `internal/export/export_test.go`) gains the delay, asserting the line
      and column of its own AST position — a position key wired to a neighbouring field satisfies both
      the key-order and the text-search assertions without this one
- [ ] The CUE export of the same fixture carries the same delays, and the "CUE and JSON exports
      describe the same model" subtest passes over it
- [ ] `internal/cue/schema.cue` declares the key on `#Automation` (`:53-62`), and the
      schema-conformance subtest running `cue vet -d '#Model'` passes for a model carrying delays
- [ ] `emod schema` prints a schema declaring the key on the automation definition
- [ ] The embedded-schema fixture in `internal/cue/embed_test.go` carries an automation stating a delay
      beside one that does not, and vets clean against `#Model`
- [ ] A document stating the delay under a key `#Automation` does not declare fails `cue vet`, and the
      failure names that key — proving the definition is closed and the emitted spelling is the one
      the schema knows
- [ ] Exporting `test.AutomationScheduleLibraryLendingModel(t)`, whose automations state no delay,
      produces JSON and CUE containing the delay key nowhere, while the Task 5 fixture's exports
      contain it — both asserted in the same subtest, so the search is proved to work before it is
      required to find nothing

**Affected Files/Modules:**
- `internal/export/json.go` — `jsonAutomation` (`:154-171`) and `convertAutomation` (`:594-616`)
- `internal/export/cue.go` — `cueWriter.writeAutomation` (`:197-206`)
- `internal/cue/schema.cue` — `#Automation` (`:53-62`)
- `internal/export/export_test.go`, `internal/cue/embed_test.go`

**Patterns to Follow:**
- `tasks/learnings.md` "A new exported field must land in JSON, CUE and `schema.cue` in the same
  change" and "JSON and CUE order their document keys differently — do not mirror one struct into the
  other": take the Go struct's field order from a `json*` sibling, not from the schema, whose worked
  example is `jsonInvariant`
- `tasks/learnings.md` "A serialized key spells the DSL keyword; the Go field may name the concept" —
  `ast.Automation.Schedule` ships as `every` and `every_position` for exactly this reason, and the
  three negative leaves that catch a key spelled after the Go field are named there
- `tasks/learnings.md` "JSON key order is assertable from the raw bytes — `emittedKeyOrder` already
  exists", "Read a decoded export document back with `objectsUnder`/`statedUnder`, in the writer's
  slice order", and "A `*_position` key needs its value pinned to its own AST position"
- `tasks/learnings.md` "The two export guards cannot see a list neither writer emits" — the parity and
  conformance subtests agree trivially about a key neither writer emits, so the read-back against the
  transcribed list is what proves the value arrived
- The closed-definition negative vet: `internal/cue/embed_test.go:112` and `:123`
- List emission in CUE: `lineIfSet` as used at `internal/export/cue.go:197-206`

**Testable:** Yes — through `export.ExportJSON`, `export.ExportCUE`, `cli.RunSchema` and `cue vet`.

**Verification:** `go test -tags unit ./internal/export/... ./internal/cue/... ./internal/cli/...`;
`go run ./cmd/emod schema`.

**Depends on:** Task 5

---

### Task 7: Carry the delay through the diagram document and back

**Behavior:** an automation's delay rides on its node in the diagram document and is read back when
that document is imported, so a delayed automation exported from `emod`, edited in the viewer and
saved keeps its delay — including after the user deletes or repoints the arrow that drew it. The
delay travels as node metadata rather than on the edge, which is the channel that survives an arrow
the user removes.

**Acceptance Criteria:**
- [ ] The diagram document's automation node states the delay the author wrote, read back for the
      Task 5 fixture against the transcribed list, while the automations of the same document that
      state an activation event or a schedule still carry those
- [ ] The node's delay key is the same spelling the model export uses, and `jsonDiagramEdge`
      (`internal/export/diagram.go:41-45`) gains no field: the arrow stays `{source, target, type}`
- [ ] Importing a document whose automation node states a delay yields a model whose automation
      carries it; importing the same document with the value moved to a key the importer does not read
      yields an automation with no delay, so the reader is proved not to accept two spellings
- [ ] Exporting the Task 5 fixture to a diagram document and importing it back yields delays equal to
      the transcribed list, and formatting that reimported model produces the fixture's canonical
      bytes — the viewer's save path is export → import → format, so this is the round-trip that
      matters
- [ ] Importing a document whose automation node states a delay and *no* `automation_trigger` edge
      still yields that delay, so the delay is proved not to depend on the arrow existing
- [ ] `foldEdges` (`internal/importer/importer.go:244-281`) is unchanged in what it folds: an
      `automation_trigger` edge still fills the activation event only when the node states neither
      activation form, and a delayed automation keeps its delay through that fold
- [ ] Re-scanning the formatted model produced from an imported document carrying a delay reports no
      diagnostics from `oracle.Check`, so the save is proved to be text `emod validate` accepts rather
      than only field-equal to what went in

**Affected Files/Modules:**
- `internal/export/diagram.go` — `jsonDiagramNode` (`:20-39`) and the automation node loop (`:195-214`)
- `internal/importer/importer.go` — `diagramNode` (`:21-38`) and `buildSlice`'s automation loop
  (`:197-208`)
- `internal/export/export_test.go`, `internal/importer/importer_test.go`

**Patterns to Follow:**
- `tasks/learnings.md` "The viewer's save path is `importer.ImportDiagram`, so a diagram-node field
  owes a read-back" — the guard is `importExported(t, model)` plus the canonical-source round-trip in
  `internal/importer/importer_test.go`, and it records that the node-metadata and edge channels are
  separate
- `tasks/learnings.md` "A diagram-node key has three readers, and they must move in one commit" — the
  exporter and importer move here, and the viewer is the third reader, which Task 10 moves. The
  intermediate state a reviewer must not accept is an exporter and importer keyed differently
- `tasks/learnings.md` "A key rename owes a retired-key negative assertion on every surface that reads
  the key" — `documentKeying` (`internal/importer/importer_test.go`) is the closure shape for
  importing one document under two keyings
- `tasks/learnings.md` "An importer fold must re-check the endpoint node types, not only the edge
  type" and "A mutually exclusive alternative has no stripped twin, and the importer must not fold one
  in" — both describe `foldEdges`'s existing guards, which this task must leave intact
- `tasks/learnings.md` "A 'the save is text emod accepts' receipt must keep the parse diagnostics" —
  `savedModelDiagnostics` runs the validator alone, so a receipt about the whole pipeline uses
  `oracle.Check` over a document complete enough to be one
- The diagram document is deliberately forked from the model document (`jsonDiagramEvent` exists so a
  new AST field cannot leak into the node-and-edge contract); do not re-merge the two automation types

**Testable:** Yes — through `export.ExportDiagramJSON`, `importer.ImportDiagram` and `oracle.Check`.

**Verification:** `go test -tags unit ./internal/export/... ./internal/importer/... ./internal/oracle/...`.

**Depends on:** Task 6

---

### Task 8: Label the `event -> automation` arrow with the delay in SVG and draw.io

**Behavior:** the arrow from an event to a delayed automation carries the duration, so a reader sees
the delay on the relationship it qualifies rather than on the box. The clock badge stays reserved for
`every`: a scheduled automation is drawn exactly as it is today, with its cadence on the box and no
incoming arrow, so a relative delay and a wall-clock schedule are told apart by where they are drawn.
An automation with no delay draws the arrow it draws today, unlabelled, with every box in the same
place, at the same size, in the same fill.

**Acceptance Criteria:**
- [ ] The SVG of a delayed automation carries the duration as text on the arrow between the activating
      event and the automation, and the draw.io XML carries it on that arrow's cell, with the document
      still parsing as well-formed XML
- [ ] The duration does not appear on the automation's box in either format: the box label for a
      delayed automation is exactly the label an undelayed automation's box carries, read back through
      the existing box-label helpers
- [ ] A scheduled automation in the same model still carries its clock badge and its cadence on its
      box, and still draws no incoming activation arrow — asserted on one model declaring a delayed,
      an undelayed and a scheduled automation, so all three cases move together
- [ ] The arrow to an automation stating no delay carries no label text in either format
- [ ] Every delayed automation of the Task 5 fixture appears in both renderings with its own duration
      on its own arrow, matched against the transcribed list rather than against one hand-written
      string
- [ ] The label changes no box's position, size or fill and adds, removes or repaints no arrow:
      rendering the Task 5 fixture and rendering its `Without…` twin produce identical boxes and
      identical arrows, differing only in the labels on the arrows — the comparison opening by
      asserting the twin lost the delays of both slice homes while the fixture still states the
      transcribed list
- [ ] `internal/diagram/contract_test.go`'s `diagramConnection` (`:82-87`) carries the arrow's label,
      and both format readers resolve it — `drawioEdges` (`drawio_test.go:994-1014`, whose regex at
      `:826` captures style, source and target only) and `svgConnections` (`svg_test.go:462-492`)
- [ ] `svgShapes` (`svg_test.go:354-399`) attributes an arrow's label to that arrow and not to the
      last rect it saw, so `svgBoxes`, `boxLabelled` and `labelsOf` read the same box labels for a
      delayed model as for its twin
- [ ] `git diff` moves no expected constant in `internal/diagram/svg_test.go`, `drawio_test.go` or
      `contract_test.go` that describes a model without delays, and the palette and reactor-placement
      tests pass unedited

**Affected Files/Modules:**
- `internal/diagram/graph.go` — `Edge` (`:41-46`) and the automation arm of `SliceEdges` (`:82-95`);
  this is the one place edges are derived, and its doc comment says why the arrow's text belongs with
  the arrow rather than being re-derived in each format
- `internal/diagram/drawio.go` — the `EdgeAutomationTrigger` case (`:441-451`) and `edgeCellWaypoints`
  (`:552-564`), whose sibling `edgeCell` (`:546-550`) already writes a label
- `internal/diagram/svg.go` — the edge loop (`:175-193`), `svgArrowBetween` (`:303-305`),
  `svgArrowPath` (`:307-328`) and the text emitters (`:268-295`)
- `internal/diagram/contract_test.go`, `svg_test.go`, `drawio_test.go`

**Patterns to Follow:**
- The shared edge derivation and its three consumers: `SliceEdges`
  (`internal/diagram/graph.go:51-115`), read by `drawio.go:411`, `svg.go:175` and
  `internal/export/diagram.go:376`. `tasks/learnings.md` "An arrow between two constructs is drawn by
  six surfaces, and none of them reads another" predates this file and is now only partly true — the
  two picture formats share the derivation and route their own arrows, and Mermaid, ASCII and the
  viewer are still independent (Tasks 9 and 10)
- The label strings live in one place: `internal/diagram/labels.go`, which already holds
  `cadenceLabel` and `automationLabel` and is where the delay's rendered text belongs
- `tasks/learnings.md` "Read an SVG or draw.io arrow back as the two boxes it meets" — assert
  connections by box label and appearance against an arrow of the same kind in the *same* render,
  never by restating coordinates, cell ids or a style string
- `tasks/learnings.md` "Allocate a draw.io cell id only once the cell is certain to be written" — a
  label must not change how many ids are taken, or a differential fails on ids alone
- `tasks/learnings.md` "`svgPicture` sees labelled boxes and arrows only, so it is not the receipt for
  a new mark", and "Assert diagram box placement relationally, by label" — the box comparison against
  the twin is what proves the clock badge slot is untouched
- `tasks/learnings.md` "A differential receipt must first prove the twin actually differs" and
  "`require.NotEqual` on a stripped twin is satisfiable without stripping anything" — the twin's
  emptiness *and* the fixture's transcribed list, both stated, with `forEachSlice`
  (`contract_test.go:1043-1054`) as the both-homes walk
- `tasks/learnings.md` "`ExportSVG` and `ExportASCII` ignore the `Style` they are handed" — a subtest
  parameterised over `StyleDCB`/`StyleProjected` asserts nothing extra in SVG
- The existing schedule contract is `TestExporterAutomationSchedule`
  (`internal/diagram/contract_test.go:853-915`) with `scheduleShown` (`:1142-1149`); it must keep
  passing, which is the receipt that the badge stayed on the box
- `tasks/learnings.md` "A suite that pins another tool's output owes a mutated-input negative control"
  applies to the two format readers extended here: a reader that silently returns an empty label
  agrees with every assertion that expects none

**Testable:** Yes — through `diagram.ExportSVG` and `diagram.ExportDrawio`.

**Verification:** `go test -tags unit ./internal/diagram/...`.

**Depends on:** Task 5

---

### Task 9: Name the delay in Mermaid, ASCII and `emod slices list`

**Behavior:** the three surfaces that summarise an automation as a line of text read correctly for a
delayed automation: Mermaid and ASCII name the delay beside the event it is measured from, and
`emod slices list` shows it in the automation's key elements. All three keep the distinction the
picture formats draw — a delay reads as attached to the event, a schedule as the automation's own
cadence — and all three stay exactly as they are for an automation that states no delay.

**Acceptance Criteria:**
- [ ] The Mermaid output for a delayed automation names the duration alongside the event it activates
      on and the command it issues; the output for an undelayed automation and for a scheduled one in
      the same model is unchanged from what it produces today, asserted against their existing
      expected lines
- [ ] The ASCII chain for a delayed automation names the duration beside the activating event, and the
      chains for an undelayed and a scheduled automation in the same model are unchanged
- [ ] Every rune of the ASCII output for the Task 5 fixture is ASCII apart from `⚙` — the assertion at
      `internal/diagram/ascii_test.go:396-412` passes unedited, so no clock glyph reaches this format
- [ ] The ASCII chain for a delayed automation contains no additional occurrence of `" -> "`, so
      `countConnections` (`internal/diagram/contract_test.go:128`) reports the same connection count
      for the Task 5 fixture as for its `Without…` twin
- [ ] `emod slices list` prints the duration alongside the activating event for a slice whose
      automation is delayed, the activation event alone for one that is not, and the cadence for a
      scheduled one — all three asserted on one file, so no row is checked in isolation
- [ ] `emod slices list -f json` carries the same key elements as the text listing for that file
- [ ] Rendering `test.AutomationScheduleLibraryLendingModel(t)` and
      `test.AutomationReadsLibraryLendingModel(t)` in all four diagram formats, and listing the
      examples under `examples/`, produces the output they produce today: `git diff` moves no expected
      constant describing a model without delays

**Affected Files/Modules:**
- `internal/diagram/mermaid.go` — `mermaidAutomationLabel` (`:130-146`) and its activation slot
  (`:137-140`)
- `internal/diagram/ascii.go` — the automation chain (`:85-88`) and `activationMarking` (`:122-128`)
- `internal/diagram/labels.go` — the shared label strings both formats already draw from
- `internal/cli/slices_list.go` — `automationActivation` (`:125-130`)
- `internal/diagram/mermaid_test.go`, `internal/diagram/ascii_test.go`,
  `internal/diagram/contract_test.go`, `internal/cli/slices_list_test.go`

**Patterns to Follow:**
- All three surfaces already funnel their activation text through one helper each —
  `mermaidAutomationLabel`, `activationMarking` and `automationActivation` — and two of the three
  already call `cadenceLabel` (`internal/diagram/labels.go:14-16`). `tasks/learnings.md`
  "De-duplicate before a fan-out edit, and land the de-duplication with proof" applies if they are
  collapsed further: carry a differential receipt and prefer a separate preparatory commit if the
  refactor outgrows the feature edit
- ASCII's two hard constraints: the gear is the one deliberate non-ASCII marker
  (`internal/diagram/ascii_test.go:396-412`), and the contract table counts ASCII connections by
  occurrences of `" -> "` (`internal/diagram/contract_test.go:128`)
- The chain expectations are pinned by `libraryLendingAutomationChains`
  (`internal/diagram/ascii_test.go:19-26`) read back through `reactorChains` (`:28-37`)
- `tasks/learnings.md` "An assertion whose expected value comes from the code under test is the
  recurring review finding" — name the expected line, never rebuild it from the renderer
- `tasks/learnings.md` "Strengthening a test to a whole-sequence `require.Equal` means deleting the
  subtest it subsumes" — if a whole-output comparison is added over a surface an existing
  `Contains`-based leaf already checks, fold that leaf's unique input into the new fixture and delete
  it
- `tasks/learnings.md` "urfave/cli v2 discards every flag written after the file argument" — the
  `slices list` JSON invocation must put the flag before the path, or the assertion exercises the
  default format
- Mermaid's timeframe letters are settled; this task changes no timeframe assignment

**Testable:** Yes — through `diagram.ExportMermaid`, `diagram.ExportASCII` and the `slices list`
command.

**Verification:** `go test -tags unit ./internal/diagram/... ./internal/cli/...`.

**Depends on:** Task 8

---

### Task 10: Show an automation's delay in the viewer

**Behavior:** the web viewer draws the duration on the arrow from the activating event to a delayed
automation, matching where the SVG and draw.io exports put it, and lists it in the details panel
beside the activation event and the cadence. The clock badge on the box keeps meaning `every` alone,
so the two timing notations stay legible in the viewer for the same reason they do in the exports. An
automation stating no delay looks exactly as it does today.

**Acceptance Criteria:**
- [ ] The SVG the viewer produces for a document whose automation node states a delay carries the
      duration as text on the `automation_trigger` arrow between the activating event's node and the
      automation's node
- [ ] An automation node stating no delay contributes an arrow with no label, and an automation node
      stating a schedule keeps its clock badge and its tooltip and gains no arrow label — all three
      asserted in one test so the cases move together
- [ ] A delayed automation's box draws the labels it draws today: the delay appears on the arrow and
      not among the box's `<text>` elements, read back by walking the group's text nodes rather than
      its `textContent`, which folds the `<title>` in
- [ ] The delayed automation's box keeps the position, size and fill it has without the delay,
      compared across a render with and without the value through the drawn-box reader
- [ ] The details panel shows a row for the delay beside the existing `On Event`, `Every`, `Reads`,
      `Command` and `Target Context` rows (`internal/viewer/static/ui.js:350-361`), in a fixed
      position, listing the duration for a delayed automation and the em-dash placeholder for one that
      states none
- [ ] A delay containing markup is shown as text in both the panel and the drawn label, matching the
      escaping the other rows already prove
- [ ] A node stating the delay under a key the viewer does not read shows the placeholder and draws no
      arrow label, so both readers are proved to read the same key the exporter writes
- [ ] An arrow the user draws between an event and an automation still types as `automation_trigger`
      (`internal/viewer/static/model.js:145`), and no pairing is added to `EDGE_TYPE_BY_ENDS` — the
      delay travels on the node, so no new arrow direction becomes meaningful
- [ ] `mise exec -- task test:viewer` passes, and `mise exec -- task test:unit` is unaffected — this
      task changes no Go file

**Affected Files/Modules:**
- `internal/viewer/static/renderer.js` — the edge loop (`:325-376`), which renders no label of any
  kind today, and `appendBlockLabels` (`:159-186`), whose clock badge (`:176-183`) must stay as it is
- `internal/viewer/static/ui.js` — the automation section of the details panel (`:350-361`)
- `internal/viewer/tests/renderer.test.js`, `internal/viewer/tests/detail-panel.test.js`

**Patterns to Follow:**
- `tasks/learnings.md` "The viewer shows a node twice — the canvas box and the detail panel" — both
  readers are separate, and a panel-only change leaves the canvas silent; assertions on drawn text
  walk the group's `<text>` elements, because `textContent` folds the `<title>` in, and the receipt
  that nothing moved is the drawn-box reader compared across a render with and without the value
- `tasks/learnings.md` "A diagram-node key has three readers, and they must move in one commit" — the
  exporter and importer moved in Task 7 and this is the third; a placeholder shown for a
  correctly-keyed node is the regression this task exists to prevent, and the learnings record it
  shipping green once already
- `tasks/learnings.md` "`internal/viewer/static` is a display surface with its own vitest harness" —
  jsdom implements no SVG geometry, so a module touching geometry is loaded through the dynamic
  `await import` spelling after `installSVGGeometry()`; restructuring `showDetailPanel` beyond the
  added row belongs in its own commit
- `tasks/learnings.md` "A viewer leaf must be able to fail only for the paint its name blames" — do
  not restate palette or hue relations that Go already owns
- `tasks/learnings.md` "`EDGE_TYPE_BY_ENDS` carries only the direction the exporter writes" — this
  task adds no pairing
- `tasks/learnings.md` "The viewer vitest pool is capped at two threads on purpose" — leave
  `internal/viewer/vitest.config.js` alone
- The existing row order and placeholder character in `internal/viewer/static/ui.js:350-361`, and the
  clock badge's `svgTitle` insertion as the group's first child (`renderer.js:179`), are what the new
  label must sit beside without disturbing

**Testable:** Yes — through the vitest harness under `internal/viewer/tests`.

**Verification:** `mise exec -- task test:viewer`; `git status --porcelain -- '*.go'` is empty.

**Depends on:** Task 7

---

## Summary

**Ten tasks**, ordered so that every commit leaves `task test` green, then dependency-first, then by
how much of the repo each unblocks.

Task 1 comes first for a reason particular to this story: `TestEditorKeywordCoverage` requires all
three editor grammars to name every lexer keyword, so teaching the lexer `after` before the grammars
know it would make the language commit itself red. Landing the looser surface first also keeps the
tree-sitter grammar from ever being stricter than the parser. The second drift suite,
`TestKeywordCoverage/hover`, cannot be answered ahead of time the same way — it calls `lsp.GetHover`
rather than reading a file — so the LSP hover entry rides inside Task 2 instead, which is the change
that teaches the lexer the word. Task 2 is the language change every
later task rests on, and it carries the mutual-exclusion rule along with the suffix, because a
grammar that admits `every "…" after "…"` without a rule about it would silently accept a shape with
no reading. Tasks 3 and 4 fan out from it and are independent of each other; the formatter comes early
so the window in which `emod fmt` would silently delete a delay is one commit long. Task 5 turns the
language change into a shared model, and — unlike the schedule kit before it — it can ship the
`Without…` twin that Tasks 8 and 9's differentials compare against. Tasks 6 and 7 are sequenced
because both edit `internal/export/export_test.go`; Task 8 precedes Task 9 for the same reason inside
`internal/diagram`. Task 10 comes last because the viewer reads the diagram-document key Task 7
establishes, and a placeholder shown for a correctly-keyed node is the exact regression the learnings
record from an earlier story.

**Story criteria coverage:**

| Story criterion | Task |
|---|---|
| An automation's `on` clause accepts an optional `after "<duration>"` suffix | 2 (parsed and recorded), 1 (the editor grammars accept it) |
| The duration is a Go duration string; a value that does not parse is a validation error with location | 3 |
| Without `after`, behaviour is unchanged: the automation reacts immediately | 2, 4, 6, 7, 8, 9, 10 — every task carries the receipt for its own surface, and no existing golden or expected constant moves |
| `after` on a schedule-driven automation is a validation error | 2 |
| Diagrams carry the duration on the `event -> automation` edge, the clock badge staying reserved for `every` | 8 (SVG, draw.io), 9 (Mermaid, ASCII), 10 (viewer) |

Carried along, not stated by the story: the editor grammars (1), the LSP hover entry (2), `emod fmt`
(4), the shared fixture and its twin (5), the JSON/CUE/schema trio (6), the diagram document and the
viewer's save path (7), and `emod slices list` (9), which summarises an automation's activation as
text and would otherwise show a delayed and an undelayed automation identically. The first two are
carried not by convention but by two executable drift suites that go red otherwise.

Nothing from the story is deferred. What US-013 deliberately leaves to later stories: canonical
attribute alignment inside an automation block, including the activation-keyword column the proposal's
worked example shows (US-014); LSP completion over `after` (US-015 — hover lands here, because
`TestKeywordCoverage/hover` requires it of the change that adds the keyword, while the completion
assertion runs the other direction and leaves an unoffered keyword alone); rendering specs as diagram
cards (US-016); the highlighting of the other nine new
keywords and of payload literals (US-017 — the `after` half of its criterion is satisfied here because
the keyword-coverage suite leaves no choice); and every document and example in the repository,
including the delayed automation `docs/dsl-reference.md` and `examples/` will show (US-018). Timer
runtime properties — durable scheduling, delivery guarantees, idempotency, clock skew — are not
deferred but excluded: they are a runtime concern and the story's Non-Goals put them outside the model
permanently.

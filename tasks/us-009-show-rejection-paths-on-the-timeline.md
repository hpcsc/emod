# US-009: Show rejection paths on the timeline

## Progress
- [ ] Task 1: Accept a rejection entry in a `flow` block and re-emit it through `emod fmt`
- [ ] Task 2: Share a model that states rejection edges in both slice homes
- [ ] Task 3: Resolve a rejection edge's invariant against its declaring scope
- [ ] Task 4: Derive the rejection edge and state it in the ASCII preview
- [ ] Task 5: Draw the rejection edge dashed into a badge in the SVG diagram
- [ ] Task 6: Draw the same dashed edge and badge in the draw.io diagram
- [ ] Task 7: Carry rejection edges through the JSON and CUE exports and the embedded schema
- [ ] Task 8: Report a rejection edge no spec exercises
- [ ] Task 9: Count a rejection edge as a reference for `spec/invariant-never-exercised`
- [ ] Task 10: Accept the rejection entry in the tree-sitter grammar
- [ ] Task 11: Document the rejection edge in the DSL reference

---

## Story Reference

`user-stories/specs-and-metadata.md` → **US-009: Show rejection paths on the timeline** (ninth story
of "Specs, Invariants, and Model Metadata", lines 120-132). Its Context is load-bearing and is
restated here because it bounds the whole story: *the edge covers invariant rejections only — the
command fails and nothing is appended. A failure the business cares about (a payment declined) is an
event and already has a flow entry; the rejection edge exists for the case where the timeline is
otherwise silent.* So there is no "business failure" edge kind, no rejection that appends anything,
and no second outcome shape.

**Prerequisites.** US-005 (named invariants) and US-006 (`then rejected` in specs) are delivered on
main — `ast.Invariant` (`internal/ast/ast.go:68-74`), `ast.ThenRejected` (`:121-126`), and the
`invariantScope` resolution rule (`internal/validator/validator.go:203-295`). US-008 is **decomposed
but not implemented** (`tasks/us-008-lint-spec-coverage-and-boundary-assumptions.md`); only Task 9
below depends on it, and it names the dependency as hard.

**In scope:** a second entry kind inside the `flow` block — `command -> rejected: <CommandName> ->
<invariantName>` — parsed into its own AST collection; the invariant name resolved against the
enclosing aggregate or, for a slice declared directly on a context, that context, reusing the scope
rule US-005 established and reporting the message it already emits; the two picture formats drawing
the edge dashed into a rejection badge carrying the invariant's prose as a draw.io tooltip and an SVG
`<title>`; the ASCII preview stating the same relation as text; a `flow/rejection-without-spec` lint
rule at info severity; and a rejection edge counting as a reference for
`spec/invariant-never-exercised`. Carried with them, because this repo's writers otherwise silently
drop a construct they have not heard of: `emod fmt` (which *deletes* an unknown entry —
`tasks/learnings.md:156-159`), the JSON and CUE exports plus the embedded `schema.cue`, the
tree-sitter grammar, and the DSL reference. A shared fixture kit carries the entry through every
downstream package.

**Out of scope**, each named with its owner:

- **Example payloads** (US-010). A rejection edge names a command and an invariant, both by
  identifier. Nothing here reads or writes `ast.SpecElement`, and no payload braces are accepted
  anywhere in a `flow` block.
- **Value-aware boundary checking** (US-011). `DecidesOnClause.Predicate` is read nowhere in this
  story.
- **Formatter alignment** (US-014). US-014 owns aligning the `:` across `command -> event:` and
  `command -> rejected:` entries within one `flow` block. Task 1 emits a canonical one-line form per
  entry and aligns no column — the same split US-006 made for `given`/`when`/`then`
  (`tasks/completed/us-006-write-given-when-then-specs-on-command-slices.md:36-38`).
- **LSP hover, completion, navigation and find-references over rejection edges** (US-015).
  `internal/lsp` is untouched by every task here. In particular `referencesIn`
  (`internal/lsp/model.go:127-130`) keeps naming only a flow's command and event, so go-to-definition
  from a rejection edge does not work after this story and is US-015's third and fourth criteria.
- **Spec cards on diagrams** (US-016). That story draws a *card* under a slice; this one draws an
  *edge* and its target. `tasks/us-016-render-specs-on-diagrams.md:38-42` states the same boundary
  from its side. The two meet in exactly one place, recorded under "Open questions, decided" item 8.
- **Highlighting** (US-017). No lexer keyword is added — see below — so no editor grammar is
  obliged to move. Task 10 edits `editors/tree-sitter-emod/grammar.js` because the grammar must never
  be stricter than the Go parser, not for highlighting: `"rejected"` is already in
  `editors/tree-sitter-emod/queries/highlights.scm:62` and in the `#keywords` alternation at
  `editors/vscode/syntaxes/emod.tmLanguage.json:97`, and both colour it wherever it stands.
- **Examples and the reference sweep for the spec constructs** (US-018). No file under `examples/`
  and no fixture under `internal/parser/testdata/` gains a rejection edge. Task 11's edits to
  `docs/dsl-reference.md` are the two sections that become *incomplete* the moment the entry exists
  (§7 Flows, which documents the flow block's only entry kind, and §11's cross-reference row for
  `invariant <name>`), not documentation of the feature set.
- **The viewer, the diagram-JSON document and the importer.** Decided in item 6 below.
- **`emod glossary`.** An invariant is already a glossary term (US-005); a reference to one adds no
  term. `internal/glossary` is untouched.

**No lexer keyword is added, and this was measured.** `lexer.Keywords()`
(`internal/lexer/token.go:124`) reports **38** spellings today, `rejected`
(`internal/lexer/token.go:109` → `KeywordRejected`), `invariant`, `spec` and `flow` among them — all
four arrived with US-005 and US-006. This story spends `rejected` in a second position and adds no
map entry, so neither CI-enforced drift test has anything to gain:
`editors/tree-sitter-emod/test/queries/keywords_test.go:47` (`TestEditorKeywordCoverage`, which
asserts each `lexer.Keywords()` entry is *spelled somewhere* in each of the three editor grammars)
and `internal/lsp/keywords_test.go:18` (`TestKeywordCoverage`, hover text per keyword). Both were run
against the tree as it stands and both pass; `rejected describes itself on hover` and `invariant
describes itself on hover` are already among `TestKeywordCoverage`'s 52 leaves. Task 10 therefore
edits `grammar.js` and its corpus and touches no `.scm` query and no TextMate file.

**Open questions, decided.** The story fixes five criteria and leaves the shape of every one of them
open. Each decision below is taken against evidence read out of the tree, and each is chosen so that
US-014, US-015 and US-016 stay additive.

1. **A rejection is its own AST collection, `Slice.Rejections`, not a variant of `ast.Flow`.**
   Measured: `Slice.Flows` has **eleven production readers across ten files**, and every one of them
   would change meaning if a rejection entry joined that list — `writeFlows`
   (`internal/formatter/formatter.go:325`), `convertFlows` (`internal/export/json.go:535`),
   `writeCUEList(w, "flows", …)` (`internal/export/cue.go:127`), `SliceEdges`
   (`internal/diagram/graph.go:54`), `declaresFlow` (`internal/diagram/flows.go:13`), `ExportASCII`'s
   flow loop plus `standaloneCommands` and `standaloneEvents` (`internal/diagram/ascii.go:69,132,155`),
   `flowCount` behind `left-chair` (`internal/linter/linter.go:57-62`), `foldFlow`
   (`internal/importer/importer.go:323`), `arrange`'s `KindFlow` reference
   (`internal/arrange/arrange.go:143`), `referencesIn` (`internal/lsp/model.go:128`), and
   `detectPattern`/`keyElementsForPattern` (`internal/cli/slices_list.go:119,135`). A shared struct
   with an empty `EventName` would silently make `left-chair` count rejections toward its
   three-flow threshold, make a rejected command stop reading as standalone in ASCII, make a slice
   whose only wiring is a rejection read as the `command` pattern in `emod slices list`, make the LSP
   resolve an invariant name as an event name, and make the JSON export emit `"event_name": ""`. A
   separate collection makes every one of those an explicit opt-in and leaves each reader byte-exact
   until it opts in. This is the same reasoning `tasks/learnings.md:186-189` records against
   `ast.ThenClause`'s five silent type switches, applied before the fan-out rather than after it.
2. **The two entry kinds share one `flow { }` block in the source and one canonical order in the
   formatter: every `command -> event:` entry, then every `command -> rejected:` entry.** The AST
   holds them in two collections, so source interleaving is unrecoverable — the same property
   `tasks/learnings.md:141-144` records for every other kind in a slice, and the reason a fmt golden
   is never the input re-indented. Reading "what succeeds, then what fails" is the order stated. A
   slice stating only rejections still emits a `flow {` block.
3. **Only the invariant name resolves; the command name in a rejection entry is left unchecked,
   exactly as a flow's already is.** `referenceDiagnostics` (`internal/validator/validator.go:315-317`)
   checks a flow's `EventName` against `index.eventNames` and deliberately checks nothing about its
   `CommandName`. Adding a check for one entry kind and not the other would split the flow block's
   behaviour in two for a rule the story does not ask for.
4. **A rejection edge does not count as a reference for `orphan-command`.** `orphan-command`'s own
   description is "A command no flow references is never exercised: nothing in the model shows what
   event it produces" (`internal/linter/descriptions.go:14`), and a rejection edge shows no event. A
   command whose only wiring is a failure path really is a command that does nothing, which is the
   case the rule exists to report. `index.collect` (`internal/validator/validator.go:109-117`) is
   therefore not extended, and `tasks/learnings.md:191-194` records the sibling asymmetry already in
   place: a spec is not a reference either.
5. **`flow/rejection-without-spec` looks for the exercising spec on the rejection edge's own slice,
   and matches both halves of the edge.** The story's wording is "no spec **on the slice** exercising
   that rejection", and a rejection edge is declared inside one slice's `flow` block, so that slice
   is its scope. This is a deliberate departure from US-008's model-wide `when` resolution (its Open
   question 2): a spec in an unrelated slice must not silence an edge written here. "Exercising"
   means a spec on that slice whose `when` names the edge's command *and* whose `then` is a
   `*ast.ThenRejected` naming the edge's invariant — matching only the invariant would let one
   command's rejection spec silence another command's edge. Measured blast radius, below, is zero.
6. **Mermaid, the diagram-JSON document, the importer and the viewer are unchanged; ASCII states the
   relation.** Each decided separately, from what the surface actually does:
   - **Mermaid draws no arrows at all.** Verified by reading all three of its layouts
     (`exportMermaidStandard:69`, `exportMermaidProjected:152`, `exportMermaidDCB:285`): it emits
     `tf NN <kind> <name>` timeframe lines and `%%` comments, and there is no arrow vocabulary for a
     rejection edge to join. The receipt is that `ExportMermaid` returns identical bytes for a model
     with rejection edges and its `WithoutRejections` twin.
   - **ASCII does render every relation as a text arrow** (`[Cmd] -> (Evt)`, `(Evt) -> {View}`,
     `{View} -> [Ext]`, `⚙ Name -> [Cmd]`), and its own doc comment
     (`internal/diagram/ascii.go:11-20`) is the list of markers it draws. Leaving the rejection out
     would make `emod diagram -f ascii` a lossy view of the model — the "silently ignore what you
     cannot honour" trap `tasks/learnings.md:376-379` records against `ExportSVG`'s discarded
     `Style`. It gets one line per rejection edge and a marker of its own (Task 4).
   - **The diagram-JSON document gets an explicit no-representation case.** An invariant is not a
     diagram-JSON node, and giving it one would oblige a `nodePalette` entry, an
     `EDGE_TYPE_BY_ENDS` pairing, an `edgeConfig`/`arrowClassMap` pair, a `showDetailPanel` section
     and a `foldEdges` arm — six surfaces (`tasks/learnings.md:341-344`, `:346-349`, `:226-229`) for
     a construct the viewer cannot edit. The precedent is one `case` away in the same switch:
     `case diagram.EdgeTranslationExternal:` (`internal/export/diagram.go:420`) carries the comment
     "External systems are not diagram-JSON nodes; the translation node stands in for them, so this
     edge has no representation." `EdgeRejection` takes the same shape.
   - **The consequence, stated rather than hidden:** a model saved out of the web viewer loses its
     rejection edges, because the viewer rebuilds the model from nodes and edges alone
     (`internal/wasm/pipeline.go` → `importer.ImportDiagram`). That is already true of specs and of
     invariants themselves, both of which the diagram document has never carried; this story neither
     introduces nor worsens it. Closing it means giving the viewer an invariant node, which is a
     story of its own.
7. **A rejection badge is not an element type.** It gets its own fill and stroke constants beside
   the six in `internal/diagram/layout.go:71-89`, named for what they paint and never aliased onto an
   equal value (`tasks/learnings.md:421-424`), and gets **no** `nodePalette` entry in
   `internal/viewer/static/config.js`, no `.hl.<type>-block` rule in `viewer.html`, and no row in the
   `Diagram Palette` table of `docs/dsl-reference.md` §13. `TestExporterPalette`,
   `TestExporterPalettePinsViewer` and `TestExporterPaletteMatchesReference`
   (`internal/diagram/contract_test.go`) must pass with no edit and must still see exactly six
   element types — `viewerNodePalette` asserts `require.Len(t, matches, 6)` and would refuse a
   seventh (`tasks/learnings.md:406-409`, `:416-419`). The precedent is the external system box,
   which `tasks/learnings.md:421-424` records as "not an element type and holds no palette entry of
   its own".
8. **The badge takes a place in the row its slice's events occupy, alongside them, rather than in a
   band of its own.** US-016 put its spec cards in a band below the lowest lane because its criterion
   is "the flag is off by default; without it, diagram output is unchanged" — a byte-exact receipt
   that forces every existing box to keep its coordinates. This story has no such criterion: the edge
   is unconditional, so the only receipt owed is the one the repo's convention already asks for, that
   a model stating no rejection edge renders byte-identically (`tasks/learnings.md:51-54`), which any
   placement satisfies. Drawing the badge in the event row is what makes the story's point visible —
   the dashed edge runs command → badge exactly where the flow edge runs command → event, so a
   command that can be rejected reads differently from one that cannot at a glance. Two consequences
   follow and both are Task 5 and Task 6 criteria: a slice's event row reflows to hold one more box
   (its `itemLayout` already narrows boxes as their count grows), and in draw.io's projected layout a
   slice stating a rejection edge must force `hasEventsLane` true the way an untagged DCB event and a
   translation event already do (`internal/diagram/drawio.go:96-122`), or the badge has no lane to
   sit in. Because US-016 also grows the picture and is decomposed but not implemented, whichever
   lands second inherits `svgLaneLabels`' behaviour — it infers "lane" from being the widest shape
   (`internal/diagram/svg_test.go:415-437`) — and a badge in the event row is narrower than a lane, so
   this story adds nothing to that count.
9. **A badge is keyed by its edge, not by the invariant's name.** Both picture exporters key their
   box maps by name across every slice, on purpose, so an arrow can reach a box another slice drew —
   `nameToBox` (`internal/diagram/svg.go:57`) and `nameToElem` (`internal/diagram/drawio.go:351`,
   which keeps the *first* entry for a repeated name while `nameToBox` keeps the last). Two slices
   may reject the same invariant, and a badge filed under the invariant's name alone would collapse
   into one box with both dashed arrows pointing at whichever survived — in opposite directions in
   the two formats. The resolution is the shape already used for the one other edge whose endpoint is
   not the box the name would find: `reactorExternal` (`internal/diagram/svg.go:167-172`), the
   per-slice map `EdgeTranslationReads` resolves through. Tasks 5 and 6 each carry a leaf over a
   model whose two slices reject one invariant.

**Overarching constraint:** every existing `.emod` file stays valid with unchanged meaning. That is
cheap to hold here and was measured rather than assumed — see the blast radius table below. The
grammar addition is a new production reachable only after `command ->`, so no existing text changes
meaning; the validator rule fires only on an entry no checked-in model states; the lint rule's
subject does not exist in the tree; and every writer's output moves only for a model that states the
new entry.

**Learnings folded in** from `tasks/learnings.md`, each named where it bites:

- *A new block entry goes after `description` and ahead of nested blocks, in every writer* (:156-159)
  — the writer that hurts to forget is the formatter, because it renders from the AST and **deletes**
  what it has never heard of, and neither idempotence nor an existing golden notices. This is why
  Task 1 lands the parser and the formatter in one commit rather than splitting them the way US-006
  did: a parser-only commit is a commit at which `emod fmt` silently destroys a user's rejection
  edges (:281-284 records the same class of intermediate-commit regression for a wire rename).
- *A new block entry keyword owes three things to the parser's diagnostics* (:76-79) — the
  `expected command in flow` message (`internal/parser/parser.go:1352`) and the six inside
  `parseFlowEntry` (`:1375-1405`) all name the one entry kind that exists, a malformed entry must
  report once, and `require.Len(t, diags, 1)` on the malformed input is what makes a cascade a
  failure. Note the standing gap Task 1 closes: **no test in the repository asserts any of those
  seven flow messages today** (searched across `internal/parser` and `internal/cli`).
- *A quoted or identifier block entry is one `case` on the `parse*EntryInto` family* (:316-319) and
  *a line-oriented declaration must gate every optional trailing token on the first token's line*
  (:86-89) — a flow entry is line-oriented, and `skipRestOfLineOrBlockEnd`
  (`internal/parser/parser.go:1498`) is the drain that stops one malformed entry from cascading and
  still lets the `}` close the block.
- *`emod fmt` canonicalises order, so a fmt golden is never the input re-indented* (:141-144) — the
  canonical constant Task 2 pins is what `emod fmt` writes, not the fixture re-indented.
- *Never write emod source with `%q`* (:46-49) — an invariant's *statement* reaches the diagrams as
  prose in Tasks 5 and 6; the emod side goes through `quoted()`, the XML side through `escapeXML`.
- *Additive output changes owe a byte-identical receipt for models that do not use the feature*
  (:51-54) and *a differential receipt must first prove the twin actually differs* (:96-99), with
  *`require.NotEqual` on a stripped twin is satisfiable without stripping anything* (:206-209) — the
  `WithoutRejections` twin Task 2 builds is paired everywhere with `test.DeclaredRejections` empty on
  the twin and equal to the transcription on the stated model.
- *A new optional field ships a six-part fixture kit* (:216-219) and *shared fixtures come in an
  unfeatured/featured pair* (:66-69) — Task 2's shape, including `editedCopies` leaving a nil list
  nil and the copies being shallow.
- *A new shared fixture owes `internal/oracle` a zero-diagnostic subtest* (:151-154) — and, because
  `oracle.Check` runs the linter, Task 2's fixture is designed forward so Task 8's rule and US-008's
  `spec/invariant-never-exercised` both find it already clean.
- *Diagnostics gathered from more than one AST collection must be position-sorted* (:181-184) — Task
  3's finding set now comes from a scope's slices' `Specs` *and* their `Rejections`, two collections
  that lose source order between them; `scopedInvariantDiagnostics`
  (`internal/validator/validator.go:287`) already sorts and must keep doing so.
- *A slice has two homes, and much of the repo still walks only one* (:171-174) — every walk and
  every fixture here covers `agg.Slices` and a `mode dcb` context's own `ctx.Slices`.
- *A new exported field must land in JSON, CUE and `schema.cue` in the same change* (:56-59), *JSON
  and CUE order their document keys differently* (:146-149), *the two export guards cannot see a list
  neither writer emits* (:161-164), *JSON key order is assertable from the raw bytes* (:231-234) and
  *a `*_position` key needs its value pinned to its own AST position* (:311-314) — Task 7's five
  obligations.
- *An arrow between two constructs is drawn by six surfaces, and none of them reads another*
  (:341-344), *SVG and draw.io share box placement* (:381-384), *assert diagram box placement
  relationally, by label* (:386-389), *read an SVG or draw.io arrow back as the two boxes it meets*
  (:361-364) and *allocate a draw.io cell id only once the cell is certain to be written* (:356-359)
  — Tasks 4, 5 and 6.
- *`svgPicture` sees labelled boxes and arrows only, so it is not the receipt for a new mark*
  (:431-434) — re-verified here: `svgShapes` (`internal/diagram/svg_test.go:354-397`) appends a shape
  on each `<rect>` and, on each `</text>`, overwrites the label of the shape it appended most
  recently (`:392`), and it captures a `<title>` **only** when that title sits inside a rect. A badge
  drawn as anything but one rect followed by exactly one `<text>`, with its `<title>` nested in the
  rect, silently corrupts an unrelated box's label. Task 5 closes this with a whole-list receipt.
  The neighbouring hazard is `svgConnections` (`:461-490`): it resolves each arrow endpoint through
  `labelled[shape.rect.centre()]` and `require.True(t, drawn, "an arrow meets %v, where the diagram
  draws no box", point)`, so a dashed arrow whose end is not exactly a rect's centre fails the whole
  suite loudly.
- *`ExportSVG` and `ExportASCII` ignore the `Style` they are handed* (:376-379) — so the badge's
  placement under `StyleDCB` and `StyleProjected` is a draw.io claim only (Task 6); an SVG subtest
  parameterised over those styles asserts nothing its `StyleAuto` case did not.
- *`RuleName` marks a diagnostic `emod lint --explain` can describe* (:166-169) — Task 8 registers a
  description; Task 3's validator diagnostic carries no `RuleName`, because nothing configures it.
- *A lint warning fails `emod validate`, so a new rule sweeps every checked-in model before it lands*
  (:466-469) and *a lint fixture trips exactly one rule, so it is never the minimal model* (:471-474)
  — Task 8's sweep is stated below and re-run in the task; its CLI fixture needs full flows and real
  fields to keep the other seventeen rules quiet.
- *A rule whose message branches on model state is pinned by whole formatted lines* (:476-479) and
  *a second `require.Contains` on one message is often shadowed by the first* (:136-139) — Task 3's
  message names both an invariant and a scope whose names may be substrings of each other, and
  `internal/validator/validator_test.go:1408` already compares the whole formatted line for exactly
  that reason.
- *CLI diagnostic tests must assert the distinguishing message text* (:6-9) and *an assertion whose
  expected value comes from the code under test is the recurring review finding* (:126-129).
- *A "no expected constant moves" criterion is unsatisfiable when the task edits a shared fixture*
  (:481-484) — Task 2 adds a fixture rather than editing `test.SpecLibraryLending`, and says why.
- *A `Declared…` getter answers `nil` for a fixture that declares none of the construct* (:256-259)
  — `test.DeclaredRejections` is paired only with the non-empty transcription.
- *Every `grammar.js` rule carries a one-line example of its full shape* (:251-254), *the tree-sitter
  grammar must never be stricter than the Go parser* (:61-64), *generated tree-sitter `src/` stays
  gitignored* (:16-19) and *run repo tooling through `mise exec --`* (:11-14) — Task 10.
- *`docs/dsl-reference.md` anchors embed the section number* (:36-39) and *sub-heading anchors are
  cited more often than the numbered ones* (:541-544) — Task 11 adds no heading and renames none;
  `#7-flows` is cited from §11's cross-reference table and `#invariant` and `#spec` from several
  places.
- *An ```emod fence is a promise that the block validates* (:526-529) — §7's flow skeleton sits in a
  bare ``` fence (`docs/dsl-reference.md:430-434`), which is what keeps a syntax skeleton out of
  `internal/oracle`'s "documented models" harness; Task 11 keeps it bare.
- *A task criterion requiring "committed" output cannot close* (:21-24) and *a commit-message receipt
  is the commit author's obligation, never an acceptance criterion* (:246-249) — every criterion
  below is checkable in an uncommitted working tree.
- *A tested, defensible improvement found on the way is still a separate commit* (:461-464) and
  *a task's change-set assertion must name every file its own patterns require it to change*
  (:326-329).

**Pre-existing defect, recorded and deliberately not fixed.** `ConvertDiagnostics`
(`internal/lsp/diagnostics.go:31-36`) maps `diagnostic.Warning` to `SeverityWarning` and everything
else — including `diagnostic.Info` — to `SeverityError`, so Task 8's info rule publishes as a red
squiggle in an editor, exactly as `dcb/single-tag-everywhere` already does. This is named here so a
reviewer of an LSP-adjacent diff does not read it as introduced by this story. US-008's decomposition
reached the same conclusion independently
(`tasks/us-008-lint-spec-coverage-and-boundary-assumptions.md:143-148`); fixing it changes an existing
rule's behaviour and belongs in its own commit.

---

## Codebase Context

**Measured blast radius.** Every model the repository ships was walked — the nine shared constants in
`internal/test/fixtures.go`, the three files under `examples/`, the four under
`internal/parser/testdata/`, the seven ` ```emod ` fences in `README.md` and `docs/dsl-reference.md`,
and `billingModel` in `internal/wasm/pipeline_test.go`: **24 models, 78 slices, 52 flow entries, 58
commands, 11 invariants, 7 specs.** The relevant numbers:

| Question | Measured answer |
|---|---|
| Rejection flow edges in the tree today | **0.** `ast.Flow` has no rejection variant, so one is unrepresentable; a raw scan of all **55** flow-body lines across every `.emod` file, `fixtures.go`, `pipeline_test.go`, `README.md` and `docs/dsl-reference.md` found every one of them spelling `command -> event:`. |
| Every literal `rejected` in a checked-in model | **Two spec rejections** — `internal/test/fixtures.go:463` (`then rejected OneCopyPerLoan`) and `:534` (`then rejected OneReaderPerDesk`), both in `test.SpecLibraryLending` — plus `docs/dsl-reference.md:414`, which sits in a **bare** ``` fence so `emodBlocksIn` never parses it, and six prose mentions. |
| How many times `flow/rejection-without-spec` fires before any fixture is edited | **0.** The rule is vacuous until Task 2's fixture exists, which is why Task 8 depends on it. |
| Models declaring at least one invariant | **Three**: `test.InvariantLibraryLending` (5 invariants, **0 specs**), `test.SpecLibraryLending` (2 invariants, 7 specs, both exercised), the fence at `docs/dsl-reference.md:175` (4 invariants, **0 specs**). |
| Slices declaring both a `flow` block and specs | **Four**, all in `test.SpecLibraryLending`: `Borrow Copy` and `Return Copy` under aggregate `Loan`, `Claim Desk` and `Release Desk` declared directly on the `mode dcb` context "Reading Room". It is the only model in the tree where a flow and a spec share a slice. |

So this story moves no checked-in model. Task 2 adds one.

**The flow block, in the parser.** `parseFlows` (`internal/parser/parser.go:1335-1369`) consumes
`flow`, requires `{`, loops until `}` accepting only `lexer.KeywordCommand` and reporting
`expected command in flow` otherwise (`:1352`), reports the unclosed-brace message with the opening
line, and finally attaches the block's pending comments to `flows[0]` — or returns them to
`p.pending` when the block produced no entry (`:1363-1367`). `parseFlowEntry` (`:1372-1414`) is a
flat sequence of seven checks with a `p.advance()` after each failure and six distinct messages, the
third of which — `expected event after -> in flow` (`:1381`) — is the one that becomes a choice. None
of the seven is asserted by any test in the repository.

**Invariant scoping, in the validator.** `invariantScope` (`internal/validator/validator.go:203-208`)
pairs a scope's kind and name with its own invariants and its own slices; `invariantScopes` (`:220-230`)
builds one per context and one per aggregate, carrying the comment at `:215-218` stating that an
aggregate and its enclosing context are separate resolution scopes. `unresolvedRejections`
(`:246-264`) walks each scope's slices' specs, type-asserts `*ast.ThenRejected`, and reports the names
not declared; `unresolvedRejectionDiagnostics` (`:275-283`) composes them and hands
`scopedInvariantDiagnostics` (`:287`) the format string `invariant %q is not declared in %s %q`,
which sorts by position and stamps no `RuleName`. **This is the rule Task 3 reuses verbatim** — the
same scope construction, the same message, the same sort — with `slice.Rejections` appended as a
second source alongside `slice.Specs`.

**The edge derivation.** `internal/diagram/graph.go` is one file: an `EdgeKind` iota of eleven values
with a doc comment each, an `Edge{Kind, From, To}` whose endpoints are *names* rather than resolved
elements, and `SliceEdges(s *ast.Slice)` (`:51-115`), which the file's own comment describes as the
single derivation that "keeps every renderer and the diagram-JSON exporter describing the same
picture". Three consumers read it: `internal/diagram/svg.go:175`, `internal/diagram/drawio.go:411`
and `internal/export/diagram.go:376`. **`internal/diagram/ascii.go` does not** — it walks `s.Flows`
directly at `:69`, and `standaloneCommands`/`standaloneEvents` (`:130-174`) walk it twice more.
`ExportMermaid` reads neither, because it draws no arrows.

**What happens to a new `EdgeKind` before any renderer learns it.** Both picture exporters ignore it
silently and safely, which is what makes Task 4 a green commit: SVG's edge loop ends in a `default:`
arm (`svg.go:187-193`) that looks `edge.From` and `edge.To` up in `nameToBox` and draws nothing
unless both are found — an invariant name has no box — and draw.io's switch (`drawio.go:415-456`)
names its kinds explicitly and simply matches none. `internal/export/diagram.go`'s switch (`:377-423`)
likewise has no `default`, so an unnamed kind emits no edge; Task 4 adds the explicit case anyway, so
that a reader sees the omission is deliberate rather than forgotten.

**The picture exporters' shared vocabulary.** Layout constants (`marginX 40`, `marginY 60`,
`sliceWidth 280`, `boxWidth 240`, `boxHeight 55`, `laneHeight 190`, `laneGap 30`,
`laneHeaderHeight 30`, `reactorHeight` = ¾ `boxHeight`, `reactorGap 6`) and the twelve element colour
constants live in `internal/diagram/layout.go:51-89`, alongside `collectSlices` (`:100-111`, which
reaches both slice homes through `model.SliceRefs()`), `sliceXPositions` (`:115`), `contextBounds`
(`:146`), `layoutWidth` (`:166`), `laneRowY` (`:209`), `reactorBoxes` (`:226`, the shared
automation/translation row, taking the format's line break as a parameter) and `itemLayout` (`:264`,
which narrows each box as a row's item count grows). `internal/diagram/labels.go` (25 lines) turns AST
nodes into drawn text. Two `tasks/learnings.md` entries — "SVG and draw.io share box placement, and
share it through `drawio.go`" and "One fill and stroke constant per element type" — still cite
`drawio.go` as the home of these; the claims hold, the line references do not.

**SVG.** `ExportSVG(model, _ Style)` (`internal/diagram/svg.go:12`) ignores the style and always draws
four lanes — `Wireframes`, `Commands / Views`, `Events`, `External Systems` — with `diagramW` and
`diagramH` computed at `:22-23` before anything is drawn. Shapes go through `svgRect`/
`svgRoundedRect`/`svgDashedRoundedRect`/`svgRectElement` (`:216-256`, self-closing unless a
description makes it carry a nested `<title>`), text through `svgText` (`:268`) and
`svgMultilineText` (`:273`). Every box is recorded in one `nameToBox` map (`:57`) so an arrow can
cross slices, arrows are drawn last (`:174-195`), and `svgArrowBetween`/`svgArrowPath` (`:303-328`)
hardcode `stroke="#666666"`, `stroke-width="1.5"` and `marker-end="url(#arrow)"` with no dash — so a
dashed edge needs a path builder of its own, and its `paint` is what distinguishes it when read back.

**draw.io.** `ExportDrawio(model, style)` (`internal/diagram/drawio.go:27`) branches into three lane
sets at `:64-145`: standard four (`Wireframes` / `Commands / Views` / `Events` / `External Systems`),
DCB's three (`Triggers / Commands` / `Events` / `External Systems`, with `cmdViewLaneY == triggerLaneY`),
and projected's (`Triggers / Commands`, an `Events` lane **only if** `hasEventsLane`, one `Tag: <key>`
lane per key, then `External Systems`). `hasEventsLane` is computed at `:96-122` from aggregate
events, untagged DCB events and translation events. `allocID` (`:53-58`) is a running counter shared
by every vertex and edge. `nameToElem` (`:351-356`) keeps the **first** entry for a repeated name.
Four edge styles are declared at `:343-346`; `extStyle` already carries `dashed=1` and is the nearest
existing dashed arrow.

**How the pictures are tested.** `internal/diagram/contract_test.go` holds the shared `exporter`
struct (`:28-46`) with per-format readers — `fillOfLabel`, `strokeOfLabel`, `countConnections`,
`boxes`, `connections`, `export`, `requireWellFormed` — nil for the text formats so a picture-only
subtest is written once and skipped by a `nil` guard. `diagramBox{label, appearance, rect}` and
`diagramConnection{source, target, paint}` are what a picture reads back as; `boxRect` carries
`centre`, `overlaps` and `within`; the relational helpers are `boxesDrawnOver`, `gearedBoxes`,
`labelsWithin`, `labelsBelow` and `lowestEdge`. Format-specific readers are `svgShapes`/`svgBoxes`/
`svgLaneLabels`/`svgConnections`/`svgTooltipOf`/`svgPicture` (`svg_test.go:342-540`) and
`drawioShapes`/`drawioBoxes`/`drawioEdges` (`drawio_test.go`). `arrowCount` counts SVG lines
containing `marker-end` (`svg_test.go:326-340`); draw.io's counts `edge="1"`; ASCII's counts `" -> "`
(`contract_test.go:105,115,128`) — so an ASCII rejection line spelled with ` -> ` joins that count and
a leaf must expect it.

**The exports.** `jsonFlow` (`internal/export/json.go:122-128`) is `Comments`, `CommandName`,
`CommandPosition`, `EventName`, `EventPosition`; `convertFlows`/`convertFlow` (`:535-548`) build it;
`jsonSlice.Flows` sits between `Fields` and `Views` (`:82`). The CUE side is
`writeCUEList(w, "flows", s.Flows, w.writeFlow)` (`internal/export/cue.go:127`) and `writeFlow`
(`:183-187`), emitting `command_name` then `event_name`. `internal/cue/schema.cue` declares `#Flow`
(`:39-43`) and `#Slice.flows?` (`:95`). Note that `jsonFlow` opens with `Comments`, the same
deviation from the `json*` family's `Name`-first convention that `tasks/learnings.md:146-149` records
against `jsonInvariant`; a rejection is a flow's sibling inside one block, so Task 7 matches
`jsonFlow` rather than the family — filing the two entry kinds' keys differently from each other
would be the worse outcome.

**The linter.** `internal/linter` is two files. `Lint` (`linter.go:49-103`) builds `flowCount` from
every slice's flows (`:57-62`), dispatches the mode-scoped context checks, then calls
`checkSlice(ref.Slice, aggregateName, flowCount)` once per `ctx.SliceRefs()` entry (`:93-99`).
`checkSlice` (`:107-145`) is a flat sequence of per-element loops. Severity is chosen statically at
the call site through `info`/`warning`/`errorEntry` (`:16-47`) — there is no configuration file and no
options parameter anywhere in the repo. `descriptions.go` is a single `ruleDescriptions` map of
**seventeen** entries behind `RuleDescription(name)`, and the only thing covering it is the
hand-maintained `rules` list at `internal/cli/lint_test.go:627-645`. Note `internal/linter/linter_test.go:1013`
hard-codes `require.Len(t, ruleNames, 8)` for one model that trips eight heuristic rules; that model
states no rejection edge, so Task 8's rule cannot perturb it.

**The pipeline and the models a new diagnostic must not disturb.** `oracle.Run`
(`internal/oracle/oracle.go:26-31`) is the one lex → parse → validate → lint chain and `Check`
(`:35-38`) its diagnostics-only form; `RunValidate` and `RunLint` both call it, which is why a
diagnostic of any severity fails `emod validate`. The zero-diagnostic leaves live in
`internal/oracle/oracle_test.go`: "clean input" (`:26`, one leaf per shared fixture) and "documented
models" (`:112`, every ` ```emod ` fence in `README.md` and `docs/dsl-reference.md`). `examplePaths`
(`internal/cli/validate_test.go:773`) reads `../../examples`, splits on the `_test.emod` suffix, and
requires every other file to validate. `reportedLines` (`oracle_test.go:402`) is the whole-formatted-line
helper the fixtures-that-legitimately-warn group uses.

**The fixture kit, as it stands.** `internal/test/fixtures.go` holds nine model constants, their
transcriptions (`SpecLibraryLendingSpecNames:1119`, `AutomationReadsLibraryLendingViewNames:1133`, …),
the `Without…` twins built on `copyWithEditedSlices` and the generic `editedCopies[T]`/`editedCopy[T]`
(`:1253-1298`), and the `Declared…` getters that walk `declaredSlices` (`:1364`). `internal/test/models.go`
holds one `…Model(t *testing.T) *ast.Model` accessor per fixture over `parseFixture` (`:64`).
`test.SpecLibraryLending` (`:423-572`) is transcribed by six downstream expected values —
`libraryLendingSpecs` (`internal/export/export_test.go`), the canonical constant behind `specEmod`
(`internal/cli/fmt_test.go`), and leaves in `internal/glossary`, `internal/diagram`,
`internal/formatter` and `internal/parser` — which is why Task 2 adds a fixture rather than editing
it.

**The tree-sitter grammar.** `flow_definition` (`editors/tree-sitter-emod/grammar.js:264-269`) admits
a repeat of `_flow_entry` (`:272-280`), which is a **hidden** rule, so a flow's children in the parse
tree are bare `(identifier)` pairs. Four committed corpus expectations
depend on that shape: `test/corpus/slice.txt:92` and `:642`, `test/corpus/full_model.txt:36`,
`test/corpus/specs.txt:88`. `queries/folds.scm:16`, `indents.scm:35` and `textobjects.scm:21,51`
reference `flow_definition` only, and `queries/highlights.scm` already carries both `"flow"` (`:34`)
and `"rejected"` (`:62`).

---

## Tasks

### Task 1: Accept a rejection entry in a `flow` block and re-emit it through `emod fmt`

**Behavior:** `emod validate` accepts `command -> rejected: <CommandName> -> <invariantName>` written
inside a `flow` block, alongside any number of `command -> event:` entries, and `emod fmt` writes
every entry of both kinds back out. A file that states no rejection entry parses and formats to the
exact bytes it does today.

**Why parser and formatter land together:** the formatter renders from the AST and emits only what it
knows, so a commit that teaches the parser and not the formatter is a commit at which `emod fmt`
silently *deletes* a user's rejection edges — `tasks/learnings.md:156-159` records that neither
idempotence nor an existing golden notices, and `:281-284` records the same class of intermediate
regression for a wire rename split across two commits.

**Acceptance Criteria:**
- [ ] `internal/ast` carries a rejection node with the command's name and position, the invariant's
      name and position, and comments, and a slice carries a collection of them separate from
      `Slice.Flows`; `Slice.Flows`, `ast.Flow` and every one of the eleven production readers of
      `Slice.Flows` listed in "Open questions, decided" item 1 are unchanged, which the compiler and
      an unedited `internal/linter`, `internal/arrange`, `internal/importer`, `internal/lsp` and
      `internal/cli/slices_list.go` witness
- [ ] A `flow` block containing only `command -> event:` entries parses to exactly the AST it parses
      to today: a model with no rejection entry reads back an empty rejection collection on every
      slice, and every existing subtest in `internal/parser/parser_test.go`'s "commands, events and
      flows" group passes with no edit
- [ ] A `flow` block containing both entry kinds, in either written order and interleaved, parses
      each into its own collection with the positions of the command name and the invariant name
      recorded on the tokens the author wrote
- [ ] A `flow` block containing only rejection entries parses, and its slice reads back no flows and
      one rejection per entry
- [ ] The block's leading comments attach to its first entry whichever kind that entry is — today
      they attach to `flows[0]` (`internal/parser/parser.go:1363-1367`) — and a block whose only
      entry is a rejection does not silently drop them
- [ ] Each of the malformed shapes — a word other than `command` opening an entry, a missing `->`, a
      word after `->` that is neither `event` nor `rejected`, a missing `:`, a missing identifier in
      either position, a missing `->` between the two identifiers — reports **exactly one**
      diagnostic, asserted with `require.Len(t, diags, 1)`, and the block still closes: the slice's
      and the model's `ClosePos.Line` read back non-zero, so the recovery drained to the end of the
      offending line rather than eating the `}`
- [ ] The message reporting a word after `->` that is neither `event` nor `rejected` names both
      accepted spellings, asserted with `\b`-bounded `require.Regexp` for each — `event` and
      `rejected` are both substrings of longer words the surrounding messages contain
      (`tasks/learnings.md:236-239`)
- [ ] `emod fmt` writes a slice's flow entries as one `flow {` block holding every
      `command -> event:` entry in declaration order and then every `command -> rejected:` entry in
      declaration order, one entry per line, with no column alignment across the two kinds — US-014
      owns the `:` alignment
- [ ] A slice stating rejection entries and no flow entries still emits a `flow {` block; a slice
      stating neither emits none, so `internal/formatter`'s existing goldens and every
      `*FormattedEmod` constant in `internal/cli/fmt_test.go` pass with no edit
- [ ] The parse → format → reparse comparison (`internal/formatter/formatter_test.go:427`, "round-trip
      through the parser") holds for a source stating both entry kinds in both slice homes: the
      reparsed model states the same rejection entries, in the same order, as the original — this is
      the assertion that catches the formatter dropping the entry, and neither idempotence nor a
      golden does
- [ ] Formatting is idempotent over a source whose invariant name and command name are ordinary
      identifiers, and no emod source text produced here goes through `%q`
      (`tasks/learnings.md:46-49`)
- [ ] `oracle.Check` over every model the repository ships as valid returns an empty diagnostic list:
      the nine shared fixtures, `examples/all_patterns.emod`, `examples/dcb_model.emod`, the three
      valid files under `internal/parser/testdata/`, the seven ` ```emod ` fences, and `billingModel`
      in `internal/wasm/pipeline_test.go`
- [ ] `internal/validator`, `internal/linter`, `internal/export`, `internal/diagram`,
      `internal/importer`, `internal/lsp`, `internal/glossary` and `editors/` are untouched: this task
      teaches the language to read and write the entry and nothing to interpret it

**Affected Files/Modules:**
- `internal/ast/ast.go` — the rejection node beside `Flow` (`:165-171`), and its collection on
  `Slice` (`:76-93`)
- `internal/parser/parser.go` — `parseFlows` (`:1335-1369`) writing into two collections and
  attaching comments to whichever entry came first, and `parseFlowEntry` (`:1372-1414`) branching
  after the first `->`
- `internal/parser/parser_test.go` — subtests in the "commands, events and flows" group (`:1270`),
  and the malformed shapes in "error reporting" (`:1849`)
- `internal/formatter/formatter.go` — `writeSlice`'s flow guard (`:195-198`) and `writeFlows`
  (`:325-331`)
- `internal/formatter/formatter_test.go` — the round-trip leaf and a formatted-output leaf

**Patterns to Follow:**
- The entry's parse-and-recover shape and its drain: `parseInvariant` (`internal/parser/parser.go:333-358`)
  and `skipRestOfLineOrBlockEnd` (`:1498`); `tasks/learnings.md:76-79` and `:86-89`
- Where a new parser subtest belongs: `tasks/learnings.md:106-109`
- What `emod fmt` does to a slice's entry kinds and why a golden is never the input re-indented:
  `tasks/learnings.md:141-144` and `:196-199`
- The receipt that the formatter did not drop the entry: `internal/formatter/formatter_test.go:427`
  and `tasks/learnings.md:156-159`

**Testable:** Yes — through `parser.Parse` and `formatter.Format`, both exported.

**Verification:** `task test:unit` passes; `go run ./cmd/emod validate` and `go run ./cmd/emod fmt
--check` over each model the repository ships as valid exit 0; a scratch file stating both entry kinds
survives `emod fmt` unchanged in meaning.

**Depends on:** None.

---

### Task 2: Share a model that states rejection edges in both slice homes

**Behavior:** One model in `internal/test/fixtures.go` states rejection edges in an aggregate-nested
slice and in a slice declared directly on a `mode dcb` context, so every downstream package carries
the construct through its own pipeline from one source instead of writing its own. `oracle.Check`
reports nothing about it.

**Why a new fixture rather than an edit to `test.SpecLibraryLending`:** six downstream expected values
transcribe that fixture's own text — `libraryLendingSpecs` (`internal/export/export_test.go`), the
canonical constant behind `specEmod` (`internal/cli/fmt_test.go`), and leaves in `internal/glossary`,
`internal/diagram`, `internal/formatter` and `internal/parser` — and `tasks/us-016-render-specs-on-diagrams.md`
depends on it staying still as its off-by-default receipt. `tasks/learnings.md:481-484` records what a
shared-fixture edit costs a change-set criterion.

**Acceptance Criteria:**
- [ ] `internal/test/fixtures.go` gains one model constant stating rejection edges in **both** slice
      homes — at least one in a slice nested in an aggregate, at least one in a slice declared
      directly on a `mode dcb` context — with every invariant it names declared in that slice's own
      scope
- [ ] The fixture states at least one `flow` block holding both entry kinds, and at least one slice
      whose flow block holds a rejection entry **ahead of** a further `command -> event:` entry, so
      an entry running on into what follows it is caught; a rejection written only as a block's last
      entry witnesses nothing (`tasks/learnings.md:91-94`)
- [ ] The fixture declares at least one slice with a flow block and **no** rejection entry, so a walk
      that assumes every flow block has one reads back wrong
- [ ] Every rejection edge in the fixture is exercised by a spec on its own slice — a spec whose
      `when` names that edge's command and whose `then` is `rejected <that invariant>` — so the
      fixture is already clean when Task 8's rule lands and needs no second edit
- [ ] Every invariant the fixture declares is named by at least one `then rejected` spec, so the
      fixture is also clean under US-008's `spec/invariant-never-exercised` whenever that lands
- [ ] `oracle.Check` over the fixture returns an empty diagnostic list, and a leaf in
      `internal/oracle/oracle_test.go`'s "clean input" group (`:26`) asserts it — `oracle.Check` runs
      the linter too, and a `mode dcb` shape is the usual tripwire: tag every event and let a
      `decides_on` reach every tag key, or `dcb/untagged-event` and `dcb/orphan-tag-key` fire
      (`tasks/learnings.md:151-154`)
- [ ] A hand-transcribed list names every rejection edge the fixture states — both halves of each
      edge, its command and its invariant — both slice homes together and in declaration order, so a
      walk or a strip reaching only one home reads back short against it
- [ ] A `Declared…` getter walks `declaredSlices` (`internal/test/fixtures.go:1364`) and returns that
      same list for the parsed fixture, and `require.Equal` against the transcription passes
- [ ] A `Without…` twin returns a copy whose slices state no rejection edge in either home, built on
      `copyWithEditedSlices`/`editedCopies` so a nil list stays nil; the twin reads back empty from
      the getter while the original still reads back the full transcription, which is what makes a
      later differential a comparison of two different things rather than a model with itself
      (`tasks/learnings.md:96-99`, `:206-209`)
- [ ] The twin clears rejection edges **only**: the fixture's flows, specs and invariants are
      identical in both, asserted positively rather than by count
- [ ] `internal/test/models.go` gains the parsed accessor for the fixture, matching the eight
      existing `…Model(t *testing.T) *ast.Model` siblings (`:13-61`)
- [ ] `internal/cli/fmt_test.go` gains a canonical constant holding what `emod fmt` writes for this
      fixture, and `requireFmtSettlesOn` is passed *that* constant rather than the fixture source —
      the fixture is not byte-stable under `Format`, because `emod fmt` canonicalises the order of a
      slice's entry kinds (`tasks/learnings.md:141-144`)
- [ ] `internal/formatter/formatter_test.go`'s round-trip group gains one leaf for this fixture,
      folded into the group's existing per-fixture shape rather than opening a parallel table, and
      pairs the `Declared…` getter with the non-empty transcription (`tasks/learnings.md:256-259`)
- [ ] No existing fixture constant, transcription, twin, getter, golden or `*FormattedEmod` constant
      moves: `git diff` in `internal/test`, `internal/formatter`, `internal/cli`, `internal/export`,
      `internal/glossary` and `internal/diagram` shows additions only

**Affected Files/Modules:**
- `internal/test/fixtures.go` — the fixture constant, its transcription, its twin and its getter
- `internal/test/models.go` — the parsed accessor
- `internal/oracle/oracle_test.go` — the "clean input" leaf (`:26`)
- `internal/cli/fmt_test.go` — the canonical formatted constant
- `internal/formatter/formatter_test.go` — the round-trip leaf

**Patterns to Follow:**
- The six-part kit and the mechanics of `copyWithEditedSlices`/`editedCopies`/`editedCopy`
  (`internal/test/fixtures.go:1253-1298`); `tasks/learnings.md:216-219`
- The nearest sibling fixture, declaring the construct in both homes with one instance omitted
  mid-block: `test.SpecLibraryLending` (`internal/test/fixtures.go:416-572`) and its comment
- Why an omitted optional part goes mid-block, never last: `tasks/learnings.md:91-94`
- A fixture that must be clean under `oracle.Check`, and the DCB tripwires:
  `tasks/learnings.md:151-154`
- Pairing a `Declared…` getter only with a non-empty transcription: `tasks/learnings.md:256-259`

**Testable:** Yes — the fixture's own leaves in `internal/oracle`, `internal/formatter` and
`internal/cli` are the test.

**Verification:** `task test:unit` passes; `oracle.Check` over the fixture is empty; `git diff --stat`
names only the five files above.

**Depends on:** Task 1.

---

### Task 3: Resolve a rejection edge's invariant against its declaring scope

**Behavior:** `emod validate` reports an error, positioned on the invariant name, when a rejection
edge names an invariant the enclosing aggregate — or, for a slice declared directly on a context,
that context — does not declare. It is the same rule, the same scope construction and the same
message a spec's `then rejected` already gets, so a model author sees one behaviour for one concept.

**Acceptance Criteria:**
- [ ] A rejection edge in a slice nested in an aggregate whose invariant name that aggregate does not
      declare produces exactly one diagnostic at `diagnostic.Error`, positioned at the invariant
      name, whose whole formatted line equals the line a spec's `then rejected` produces for the same
      name and scope — asserted as one `require.Equal` over the formatted line, not as two
      `require.Contains` calls, because the invariant's name and its scope's name may be substrings
      of each other (`tasks/learnings.md:136-139`)
- [ ] A rejection edge in a slice declared directly on a `mode dcb` context resolves against that
      context's own invariants and reports the same way
- [ ] Scope is not inherited in either direction: an invariant declared on the enclosing context does
      not resolve a rejection edge written in one of its aggregates' slices, and a sibling
      aggregate's invariant does not resolve it either — a model declaring the same identifier in two
      scopes and rejecting it in one reports the other and only the other
- [ ] A rejection edge whose invariant resolves produces no diagnostic, and neither does a model with
      no rejection edges: every model the repository ships as valid, including Task 2's fixture,
      still returns an empty `oracle.Check`
- [ ] Findings gathered from a scope's slices' specs and from their rejection edges come back in
      declaration order, asserted as one comparison over the whole list of formatted lines — the two
      are separate AST collections, so an unsorted walk reports them in field order rather than
      source order (`tasks/learnings.md:181-184`)
- [ ] The diagnostic carries **no** `RuleName`: nothing configures it and `emod lint --explain` must
      not claim to describe it (`tasks/learnings.md:166-169`)
- [ ] The rejection edge's **command** name is not resolved, matching a flow's, which
      `referenceDiagnostics` (`internal/validator/validator.go:315-317`) checks only for its event: a
      rejection edge naming a command no slice declares produces no diagnostic from this task, and a
      leaf states that
- [ ] The scope construction is reused rather than rebuilt: `invariantScopes`
      (`internal/validator/validator.go:220-230`) is called once and serves redeclaration, spec
      rejections and rejection edges alike, so a reviewer can point at one place where "which
      invariants does this scope declare, and which of its slices may name them" is decided
- [ ] `internal/linter`, `internal/formatter`, `internal/export`, `internal/diagram` and
      `internal/lsp` are untouched

**Affected Files/Modules:**
- `internal/validator/validator.go` — `invariantScope.unresolvedRejections` (`:246-264`) gaining the
  rejection-edge source alongside the spec source, feeding the existing
  `unresolvedRejectionDiagnostics` (`:275-283`) and `scopedInvariantDiagnostics` (`:287`)
- `internal/validator/validator_test.go` — leaves beside the existing invariant-scope group

**Patterns to Follow:**
- The scope rule and its comment on why an aggregate and its context are separate scopes:
  `internal/validator/validator.go:203-230`, especially `:215-218`
- The walk this extends, and the format string it must not change:
  `unresolvedRejections` (`:246-264`) and `unresolvedRejectionDiagnostics` (`:275-283`)
- Whole-formatted-line assertions over a message naming both a symbol and its scope:
  `internal/validator/validator_test.go:1408`; `tasks/learnings.md:136-139` and `:476-479`
- Position-sorted findings from more than one collection: `scopedInvariantDiagnostics` (`:287`) and
  its comment; `tasks/learnings.md:181-184`

**Testable:** Yes — through `validator.Validate` and `oracle.Check`.

**Verification:** `task test:unit` passes; `go run ./cmd/emod validate` over each model the repository
ships as valid exits 0; a scratch file naming an undeclared invariant in a rejection edge reports the
same line a spec naming it would.

**Depends on:** Task 2.

---

### Task 4: Derive the rejection edge and state it in the ASCII preview

**Behavior:** `SliceEdges` returns a rejection edge for every rejection entry a slice declares, so
every renderer and the diagram-JSON exporter draw from one derivation, and `emod diagram -f ascii`
states the rejection as a line of its own. The two picture formats and Mermaid are unchanged at this
commit, and the diagram-JSON document says explicitly that a rejection edge has no representation in
it.

**Why the pictures stay still here:** both ignore an edge kind they have not learnt, safely. SVG's
edge loop ends in a `default:` arm (`internal/diagram/svg.go:187-193`) that draws nothing unless both
endpoint names are in `nameToBox`, and an invariant name is in no box map; draw.io's switch
(`drawio.go:415-456`) names its kinds explicitly and matches none. That makes this a green commit
whose picture receipts are byte-identity, and lets Tasks 5 and 6 be about drawing rather than about
deriving.

**Acceptance Criteria:**
- [ ] `internal/diagram/graph.go` gains one `EdgeKind` for a rejection, with a doc comment in the
      shape its eleven siblings carry, and `SliceEdges` emits one such edge per rejection entry,
      naming the entry's command and its invariant, in declaration order, positioned in the returned
      slice among the flow edges of the same slice
- [ ] A slice stating no rejection entry returns exactly the edges it returns today, asserted as a
      whole-list comparison so an edge appearing in a new position fails
- [ ] `ExportASCII` states one line per rejection edge, in a marker vocabulary distinct from the
      event marker `(Name)`, the command marker `[Name]`, the view marker `{Name}` and the automation
      marker `⚙ Name`, so a reader cannot mistake a rejected invariant for an event; the marker is
      documented in `ExportASCII`'s own doc comment (`internal/diagram/ascii.go:11-20`), which today
      lists every marker the format draws
- [ ] `standaloneCommands` and `standaloneEvents` (`internal/diagram/ascii.go:130-174`) are unchanged
      and still read `s.Flows` alone: a command whose only wiring is a rejection edge still prints as
      standalone, which is the reading "Open questions, decided" item 4 fixes for `orphan-command`
- [ ] `countConnections` for ASCII counts `" -> "` (`internal/diagram/contract_test.go:128`), so a
      leaf states what the rejection line contributes to that count rather than leaving it to be
      discovered by a failing contract subtest
- [ ] `ExportMermaid` returns byte-identical output for Task 2's fixture and its `Without…` twin,
      with the twin first proved to have lost the edges of both homes and the stated model proved to
      read back the full transcription — Mermaid draws no arrows in any of its three layouts, so it
      has nothing to add
- [ ] `ExportSVG` and `ExportDrawio` return byte-identical output for Task 2's fixture and its twin,
      under all three styles for draw.io: the new edge kind reaches both and neither draws it yet
- [ ] `internal/export/diagram.go`'s edge switch (`:377-423`) gains an explicit case for the new kind
      carrying a comment stating that an invariant is not a diagram-JSON node and the edge therefore
      has no representation — the shape `case diagram.EdgeTranslationExternal:` (`:420`) already
      uses. A leaf walks the whole diagram document for Task 2's fixture and requires no edge of the
      new type, no node carrying an invariant's name, and the same node and edge lists the twin
      produces
- [ ] `internal/importer/importer.go`, `internal/viewer/static/*` and `internal/wasm` are untouched,
      and `internal/viewer/tests` needs no run: nothing was added to a diagram node or edge for the
      viewer to read
- [ ] `internal/diagram/flows.go`'s `declaresFlow` is unchanged — a translation's derived
      command→event edge is suppressed by an explicit *flow*, and a rejection entry is not one

**Affected Files/Modules:**
- `internal/diagram/graph.go` — the `EdgeKind` and its emission in `SliceEdges` (`:51-115`)
- `internal/diagram/ascii.go` — the rejection line and the doc comment listing its marker
- `internal/diagram/ascii_test.go` — the rejection-line leaves
- `internal/export/diagram.go` — the explicit no-representation case (`:377-423`)
- `internal/export/export_test.go` — the diagram-document walk
- `internal/diagram/contract_test.go` — the byte-identity receipts for the four exporters against the
  twin

**Patterns to Follow:**
- The edge-kind doc comments and the "endpoints are names, not resolved elements" contract:
  `internal/diagram/graph.go:5-50`
- A kind the diagram document deliberately does not represent:
  `internal/export/diagram.go:420-422`
- Why each renderer has to be taught separately, and which five surfaces read nothing from each
  other: `tasks/learnings.md:341-344`
- Differential receipts that first prove the twin differs: `internal/diagram/contract_test.go:441-453`
  and `:427-439`; `tasks/learnings.md:96-99` and `:206-209`

**Testable:** Yes — through `diagram.SliceEdges`, `diagram.ExportASCII` and
`export.ExportDiagramJSON`, all exported.

**Verification:** `go test -tags unit ./internal/diagram/... ./internal/export/...`;
`go run ./cmd/emod diagram -f ascii <a scratch model stating a rejection edge>` shows the line.

**Depends on:** Task 2.

---

### Task 5: Draw the rejection edge dashed into a badge in the SVG diagram

**Behavior:** `ExportSVG` draws a dashed arrow from a rejected command to a rejection badge sitting
alongside that slice's events, and the badge carries the invariant's prose statement as a `<title>` a
browser shows on hover. A command that can be rejected no longer renders identically to one that
cannot. A model stating no rejection edge draws the bytes it draws today.

**Acceptance Criteria:**
- [ ] Rendering Task 2's fixture draws one badge per rejection edge, labelled with the invariant's
      name, read back through `svgBoxes` as naming every invariant in the fixture's transcription in
      declaration order across both slice homes
- [ ] Each badge's `<title>` carries that invariant's `Statement` verbatim, read back through
      `svgTooltipOf` (`internal/diagram/svg_test.go:513`), and a badge for an invariant whose
      statement contains XML-significant characters carries them escaped and decodes back to the
      original
- [ ] `svgConnections` reports one connection per rejection edge, its source the rejected command's
      label and its target that badge's label, with a `paint` differing from the paint of a flow
      arrow drawn in the **same** render — the comparison is against a sibling arrow in that render,
      never against a restated stroke or dash string (`tasks/learnings.md:361-364`)
- [ ] `svgShapes` over the render reports exactly one more shape per badge than the same model's
      `Without…` twin reports in total, and the label of every shape the twin's render produces is
      byte-identical in both — asserted as a whole-list comparison of the twin's shapes against the
      featured render's, so a badge emitting a stray `<text>`, or emitting its text before its rect,
      fails. `svgShapes` binds each `</text>` to the most recently appended `<rect>`
      (`internal/diagram/svg_test.go:392`), so a badge is one rect followed by exactly one `<text>`,
      with its `<title>` nested inside the rect
- [ ] `svgConnections` over the featured render resolves every arrow — it fails loudly when an
      endpoint is not exactly a rect's centre (`internal/diagram/svg_test.go:472`) — and reports
      the twin's arrows unchanged by source label, target label and paint, with the rejection
      arrows added
- [ ] `boxesDrawnOver` reports no overlap among the featured render's boxes, badges included, and
      `labelsWithin` places every badge inside the lane its slice's events are drawn in
- [ ] A model whose **two** slices reject the **same** invariant draws two badges and two dashed
      arrows, each arrow ending at the badge in its own slice's column — the box maps in both picture
      exporters are keyed by name across every slice (`nameToBox`, `internal/diagram/svg.go:57`), so
      a badge filed under the invariant's name alone collapses the two. This leaf is what fails if it
      does
- [ ] The badge is not an element type: `internal/viewer/static/config.js`,
      `internal/viewer/static/viewer.html` and `docs/dsl-reference.md` §13 are not in this task's
      change set, and `TestExporterPalette`, `TestExporterPalettePinsViewer` and
      `TestExporterPaletteMatchesReference` pass with no edit, still seeing exactly six element types
- [ ] The badge's fill and stroke are their own named constants beside the twelve in
      `internal/diagram/layout.go:71-89`, named for what they paint and not aliased onto an equal
      value (`tasks/learnings.md:421-424`)
- [ ] `ExportSVG` over Task 2's fixture's `Without…` twin returns bytes byte-identical to what it
      returns for that twin before this task, and `svgLaneLabels` reports the same four lanes for
      both the featured and the default render — a badge is narrower than a lane, and
      `svgLaneLabels` infers "lane" from being the widest shape (`internal/diagram/svg_test.go:415-437`)
- [ ] The featured render is well-formed XML and its `viewBox` is unchanged: a badge takes a place in
      a row that already exists rather than growing the canvas
- [ ] `internal/diagram/drawio.go`, `mermaid.go`, `ascii.go`, `graph.go` and `internal/export/` are
      unchanged by this task

**Affected Files/Modules:**
- `internal/diagram/svg.go` — the badge box and the dashed arrow builder beside `svgArrowPath`
  (`:303-328`), and the rejection arm in the edge loop (`:174-195`) resolving through a per-slice
  badge map rather than `nameToBox`
- `internal/diagram/layout.go` — the badge's fill and stroke constants (`:71-89`) and where a badge
  sits in its slice's event row, beside `itemLayout` (`:264-274`)
- `internal/diagram/labels.go` — the badge's label, beside `automationLabel` (`:18-25`)
- `internal/diagram/svg_test.go` — the badge and arrow leaves and the two `svgShapes` hazard receipts
- `internal/diagram/contract_test.go` — if the badge needs a relational placement leaf shared with
  Task 6, it is added there rather than duplicated

**Patterns to Follow:**
- An edge whose endpoint is resolved through a per-slice map rather than the shared name map:
  `reactorExternal` and the `EdgeTranslationReads` arm (`internal/diagram/svg.go:165-185`)
- A shape carrying a nested `<title>` only when it has prose: `svgRectElement`
  (`internal/diagram/svg.go:250-256`) and its comment
- A dashed shape already in the file: `svgDashedRoundedRect` (`:237-239`)
- Reading a picture back relationally: `svgBoxes`, `svgShapes`, `svgConnections`, `svgTooltipOf`
  (`internal/diagram/svg_test.go:342-540`) and `boxesDrawnOver`/`labelsWithin`/`labelsBelow`
  (`contract_test.go`); `tasks/learnings.md:386-389` and `:361-364`
- Why the receipt is written against `svgShapes` and not `svgPicture`: `tasks/learnings.md:431-434`
- One fill and stroke constant per thing painted, never aliased: `tasks/learnings.md:421-424`

**Testable:** Yes — through `diagram.ExportSVG` and the `internal/test` fixture kit.

**Verification:** `go test -tags unit ./internal/diagram/...`; `go run ./cmd/emod diagram -f svg -o
<temp>.svg <a copy of a rejection-stating model>` and read the badge and its `<title>` out of the
written file. Copy the model to a temp path first — `emod fmt`/`emod diagram` receipts against a
tracked file dirty the tree (`tasks/learnings.md:336-339`).

**Depends on:** Task 4.

---

### Task 6: Draw the same dashed edge and badge in the draw.io diagram

**Behavior:** `ExportDrawio` draws the same badge and the same dashed arrow, in every one of its three
lane sets, with the invariant's prose as the cell's tooltip. The two picture formats state the same
badge text for the same model.

**Acceptance Criteria:**
- [ ] Rendering Task 2's fixture draws one draw.io cell per rejection edge, `drawioBoxes` reading
      their labels back as every invariant in the fixture's transcription in declaration order, and
      each carrying that invariant's `Statement` as its tooltip — which means the cell is wrapped in
      an `<object>`, the shape `vertexCell` (`internal/diagram/drawio.go:513`) already takes for a
      cell with prose
- [ ] The badge text draw.io states and the badge text SVG states are equal once each format's line
      break is normalised, asserted once at the contract level so a change to one format's label and
      not the other fails
- [ ] `drawioEdges` reports one edge per rejection edge, source the rejected command's cell and
      target that badge's cell, with a style differing from the standard flow edge's style in the
      same render and carrying draw.io's dash
- [ ] Under `StyleAuto`, `StyleDCB` and `StyleProjected` the badge is drawn inside a lane the style
      draws and overlaps no cell of any lane. This is asserted in draw.io only: `ExportSVG` declares
      its `Style` parameter `_` and always draws the same four lanes, so an SVG subtest parameterised
      over the other two styles asserts nothing its `StyleAuto` case did not
      (`tasks/learnings.md:376-379`)
- [ ] In `StyleProjected`, a model whose only slices are DCB slices with fully tagged events and no
      translations — a shape for which `hasEventsLane` is false today
      (`internal/diagram/drawio.go:96-122`) — draws its rejection badges in a lane rather than at a
      lane's coordinates with no lane behind them: a slice stating a rejection edge counts toward
      `hasEventsLane` the way an untagged DCB event and a translation event already do. A leaf states
      this over exactly that model
- [ ] Every cell the `Without…` twin's render writes appears in the featured render with the same id,
      the same geometry and the same style string, except for the cells of the row a badge joins,
      whose reflow is the intended consequence of one more box in that row — no badge id is allocated
      before both the "this slice states a rejection edge" guard and the badge's lane have been
      resolved, so an id taken early does not renumber every later cell
      (`tasks/learnings.md:356-359`)
- [ ] A model whose two slices reject the same invariant draws two badge cells and two edges, each
      edge ending at the badge in its own slice's column — `nameToElem`
      (`internal/diagram/drawio.go:351-356`) keeps the **first** entry for a repeated name, the
      opposite of `nameToBox`, so a badge filed by invariant name alone fails differently in the two
      formats and this leaf catches both
- [ ] `ExportDrawio` over the twin returns bytes byte-identical to what it returns for that twin
      before this task, under all three styles
- [ ] The three palette tests pass with no edit and still see exactly six element types
- [ ] The featured render is well-formed XML

**Affected Files/Modules:**
- `internal/diagram/drawio.go` — the badge cell, its style constant beside `style*` (`:12-25`), the
  `hasEventsLane` contribution (`:96-122`), and the rejection arm in the edge loop (`:411-456`)
- `internal/diagram/layout.go` / `labels.go` — consumed unchanged from Task 5, with draw.io's line
  break passed in
- `internal/diagram/drawio_test.go` — the badge and edge leaves and the three style-dependent
  placement leaves
- `internal/diagram/contract_test.go` — the cross-format badge-text equality leaf

**Patterns to Follow:**
- A cell that carries prose as a tooltip: `vertexCell` (`internal/diagram/drawio.go:513`)
- Allocating a cell id only once the cell is certain to be written: the `readsEdge` closure
  (`internal/diagram/drawio.go:360-370`) and its comment; `tasks/learnings.md:356-359`
- The dashed edge style already in the file: `extStyle` (`:346`)
- Reading draw.io back by label and by the two boxes an arrow meets: `drawioBoxes` and `drawioEdges`
  (`internal/diagram/drawio_test.go`); `tasks/learnings.md:361-364`
- A subtest written once for both picture formats and skipped for the text ones by the nil-`boxes`
  guard: `TestExporterReactorPlacement` (`internal/diagram/contract_test.go`)

**Testable:** Yes — through `diagram.ExportDrawio` and the contract harness.

**Verification:** `go test -tags unit ./internal/diagram/...`; `go run ./cmd/emod diagram --style
projected -o <temp>.drawio <a copy of a DCB rejection-stating model>` and read the badge cell and its
tooltip out of the written file.

**Depends on:** Task 5.

---

### Task 7: Carry rejection edges through the JSON and CUE exports and the embedded schema

**Behavior:** `emod export -f json` and `-f cue` carry every rejection edge a model states, and
`internal/cue/schema.cue` describes it, so the three surfaces that must agree do. A model stating no
rejection edge exports the exact bytes it exports today.

**Acceptance Criteria:**
- [ ] `emod export -f json` emits one object per rejection edge under a slice, carrying the command's
      name and position and the invariant's name and position, and the CUE export emits the same
      content under the same key name
- [ ] The JSON object files its keys in the order `jsonFlow` (`internal/export/json.go:122-128`)
      files its own, asserted through `emittedKeyOrder` (`internal/export/export_test.go`) against
      the flow object's key list produced by the **same** subtest, so the expectation is a sibling
      rather than an arbitrary literal (`tasks/learnings.md:231-234`). Note this deliberately
      inherits `jsonFlow`'s `Comments`-first opening, which departs from the `json*` family's
      `Name`-first convention (`tasks/learnings.md:146-149`): the two entry kinds of one block filing
      their keys differently from each other would read worse than either filing them differently
      from the family
- [ ] Each position key's value is wired to its **own** AST position: a leaf asserts the line and
      column of the command position and of the invariant position separately, over a fixture where
      the two differ, so swapping them fails (`tasks/learnings.md:311-314`)
- [ ] `internal/cue/schema.cue` declares the rejection definition beside `#Flow` (`:39-43`) and the
      slice-level key beside `flows?` (`:95`), and the "output conforms to the schema's Model
      definition" subtest — `cue vet -d '#Model'` — passes for Task 2's fixture
- [ ] The "CUE and JSON exports describe the same model" subtest passes for Task 2's fixture, and a
      transcribed read-back leaf in the shape of `listsKeyedBy` asserts both documents carry the
      fixture's full transcription of rejection edges under **both** slice homes — the export-parity
      and schema-conformance guards agree trivially over a list neither writer emits
      (`tasks/learnings.md:161-164`)
- [ ] A retired-key style negative leaf exists on the schema side in the shape of
      `internal/cue/embed_test.go:112`: re-keying the rejection object in a valid document and
      requiring `cue vet` to fail naming the wrong key, so a schema left accepting both spellings
      cannot ship green (`tasks/learnings.md:276-279`)
- [ ] Exporting Task 2's fixture's `Without…` twin in both formats produces bytes byte-identical to
      what those formats produce for the twin before this task, with the twin proved empty of
      rejection edges and the stated model proved to read back the full transcription first
- [ ] The diagram-JSON document is unchanged and still carries no rejection key: `jsonDiagramEvent`
      and its siblings in `internal/export/diagram.go` are deliberately forked from the model
      documents so a new field cannot leak into the node-and-edge contract
      (`tasks/learnings.md:56-59`), and a subtest walks the whole diagram document asserting the key
      appears nowhere
- [ ] `internal/importer/importer.go` and `internal/viewer` are untouched: the key is a model-export
      key, not a diagram-node key, so nothing reads it back on the viewer's save path

**Affected Files/Modules:**
- `internal/export/json.go` — the document type beside `jsonFlow` (`:122-128`), the slice field beside
  `Flows` (`:82`), and the converter beside `convertFlow` (`:535-548`)
- `internal/export/cue.go` — the writer beside `writeFlow` (`:183-187`) and the list call beside
  `writeCUEList(w, "flows", …)` (`:127`)
- `internal/cue/schema.cue` — the definition beside `#Flow` (`:39-43`) and the key beside
  `#Slice.flows?` (`:95`)
- `internal/export/export_test.go` — the key-order leaf, the position leaf, the read-back leaf and
  the twin differential
- `internal/cue/embed_test.go` — the re-keyed negative leaf

**Patterns to Follow:**
- The sibling to copy on the Go side, and the one *not* to copy from: `jsonFlow`
  (`internal/export/json.go:122-128`) rather than `internal/cue/schema.cue`'s key order;
  `tasks/learnings.md:146-149`
- Read-back helpers that answer content questions, and the ones that answer order questions:
  `listsKeyedBy`, `objectsUnder`/`statedUnder`/`exportedSlices` (`internal/export/export_test.go`)
  and `emittedKeyOrder`; `tasks/learnings.md:291-294` and `:231-234`
- The two coupled guards and what they cannot see: `tasks/learnings.md:56-59` and `:161-164`
- Escaping in the exporters is the opposite obligation from the formatter's: `tasks/learnings.md:46-49`

**Testable:** Yes — through `export.ExportJSON`, `export.ExportCUE` and `cue vet`.

**Verification:** `go test -tags unit ./internal/export/... ./internal/cue/...`; `go run ./cmd/emod
export -f json <a copy of a rejection-stating model>` and read the entries back.

**Depends on:** Task 2.

---

### Task 8: Report a rejection edge no spec exercises

**Behavior:** `emod lint` reports `flow/rejection-without-spec` at info severity, positioned on the
rejection edge's invariant name, when the slice holding that edge states no spec exercising it — no
spec whose `when` names the edge's command and whose `then` rejects that same invariant. The edge says
a command can fail; the rule asks for the scenario that shows how.

**Acceptance Criteria:**
- [ ] A rejection edge on a slice whose specs include none whose `when` names that edge's command and
      whose `then` is a rejection naming that edge's invariant produces exactly one diagnostic at
      `diagnostic.Info`, rule name `flow/rejection-without-spec`, positioned at the invariant name in
      the edge, with one message text naming both the command and the invariant
- [ ] A rejection edge whose slice states a matching spec produces no diagnostic
- [ ] Matching is on both halves and a leaf states each miss separately: a spec rejecting that
      invariant but naming a different command in its `when` does not silence the edge, and a spec
      naming that command but rejecting a different invariant does not either
- [ ] The search is slice-local: a matching spec declared in a **different** slice of the same
      aggregate, and one declared in a different context, each leave the edge reported — this is a
      deliberate departure from US-008's model-wide `when` resolution, and a leaf pins it so a later
      "tidy" toward the model-wide lookup fails
- [ ] A slice stating no rejection edge produces nothing however many specs or flows it declares, and
      a model stating no rejection edge anywhere produces nothing
- [ ] The rule reaches both slice homes — nested in an aggregate and declared directly on a `mode
      dcb` context — and a model with several unexercised rejection edges across both homes reports
      them in declaration order, asserted as one comparison over the whole list of formatted lines
- [ ] `linter.RuleDescription` answers for `flow/rejection-without-spec`, `emod lint --explain
      flow/rejection-without-spec` prints that non-empty description and returns no error, an unknown
      rule name still returns an error, and the rule name joins the hand-maintained list at
      `internal/cli/lint_test.go:627-645` so the "all rules have descriptions" leaf covers it
- [ ] A CLI lint fixture states one rejection edge with no matching spec and trips this rule and no
      other — asserted with a length of exactly one entry — with a declaring comment naming the rule
      it is written to fire. Keeping the other seventeen quiet needs full flows and events with real
      fields: `orphan-command`, `orphan-event`, `left-chair`, `god-view`, `view-naming`,
      `clickbait-event` and the `dcb/*` family are the tripwires
      (`tasks/learnings.md:471-474`)
- [ ] `cli.RunLint` and `cli.RunValidate` both return an error for that fixture — an info diagnostic
      is still a diagnostic — the text output names the rule, the command, the invariant and the line
      the edge is written on, and `-f json` reports `"severity": "info"` with exit code 1
- [ ] Every model the repository ships as valid still produces zero diagnostics from `oracle.Check`:
      the nine shared fixtures **including Task 2's**, every file under `examples/` without the
      `_test.emod` suffix, every fixture under `internal/parser/testdata/`, every ` ```emod ` fence in
      `README.md` and `docs/dsl-reference.md`, and `billingModel` in `internal/wasm/pipeline_test.go`.
      Measured before the rule exists: it fires **0** times across the tree, because no checked-in
      model stated a rejection edge and Task 2's fixture exercises every one of its own
- [ ] `require.Len(t, ruleNames, 8)` at `internal/linter/linter_test.go:1013` passes unedited: the
      model that leaf builds states no rejection edge
- [ ] The rule asks of a spec's `then` only whether it is a rejection naming a given name, through an
      `ok`-guarded assertion rather than a switch over `ast.ThenClause`, so a variant US-007 adds is
      neither counted as a rejection nor treated as an error (`tasks/learnings.md:186-189`)
- [ ] `internal/parser`, `internal/validator`, `internal/formatter`, `internal/export` and
      `internal/diagram` are untouched

**Affected Files/Modules:**
- `internal/linter/linter.go` — a per-slice check reached from `checkSlice` (`:107-145`), which
  already receives the whole slice
- `internal/linter/descriptions.go` — the rule description (`:10-31`)
- `internal/linter/linter_test.go` — a top-level group for the rule: both slice homes, both halves of
  the match, the slice-local scope, severity, position and ordering
- `internal/cli/lint_test.go` — the single-rule fixture, its text and JSON leaves, and the rule-name
  list at `:627-645`

**Patterns to Follow:**
- The one existing info-severity rule and its suite, the template for asserting severity beside
  message and position: `dcb/single-tag-everywhere` (`internal/linter/linter.go:238-266`)
- `info(pos, rule, msg)` (`internal/linter/linter.go:16-25`) — the whole severity story; there is no
  configuration file anywhere in the repo
- The `*ast.ThenRejected` assertion asked from the opposite direction: `unresolvedRejections`
  (`internal/validator/validator.go:246-264`)
- Sorting findings by declaration position and the comment saying why: `checkOrphanTagKeys`
  (`internal/linter/linter.go:320-335`)
- Why a lint fixture is never the minimal model: `tasks/learnings.md:471-474`; and why a new rule
  sweeps every checked-in model before it lands: `:466-469`
- Naming a rule obliges registering its description: `tasks/learnings.md:166-169`
- CLI leaves assert the distinguishing message text: `internal/cli/validate_test.go:253-258`;
  `tasks/learnings.md:6-9`

**Testable:** Yes — through `linter.Lint`, `cli.RunLint`, `cli.RunLintExplain` and `oracle.Check`.

**Verification:** `task test:unit` passes; `go run ./cmd/emod lint` over each model the repository
ships as valid exits 0; `go run ./cmd/emod lint --explain flow/rejection-without-spec` prints the
description.

**Depends on:** Task 2.

---

### Task 9: Count a rejection edge as a reference for `spec/invariant-never-exercised`

**Behavior:** An invariant that no spec rejects, but that a rejection edge in its own scope names,
stops being reported by `spec/invariant-never-exercised`. A model that states its error handling on
the timeline rather than in specs is not told its invariants are unused.

**Hard dependency, outside this story.** This task extends US-008 Task 4
(`tasks/us-008-lint-spec-coverage-and-boundary-assumptions.md:495-597`), which is decomposed and not
implemented. That task was written to leave this seam and states it explicitly at `:554-561`: it
implements the rule as a **collector** ("which invariant names does this scope reference?") over the
scope's slices, kept separate from the **comparison** ("which of this scope's invariants are
unreferenced?"). This task appends rejection edges to the collector's result and changes nothing
else. Two of US-008's decisions are honoured rather than re-litigated: `spec/invariant-never-exercised`
is **not** gated on the model stating a spec (its Open question 3, taken precisely so a model whose
invariants are referenced only by rejection edges is not silenced), and an invariant is exercised
within its own scope only and only by a reference that resolves (its Open question 4). This task
starts only after US-008 Task 4 has landed.

**Acceptance Criteria:**
- [ ] An invariant declared on an aggregate that no spec in that aggregate's slices rejects, but that
      a rejection edge in one of those slices names, produces no `spec/invariant-never-exercised`
      diagnostic
- [ ] The same holds for an invariant declared directly on a `mode dcb` context and named by a
      rejection edge in one of that context's own slices
- [ ] Scope is not widened: a rejection edge in a sibling aggregate's slice does not exercise an
      invariant, and one in an aggregate's slice does not exercise an invariant declared on the
      enclosing context — a model declaring the same identifier in two scopes and naming it from a
      rejection edge in one reports the other and only the other
- [ ] A rejection edge naming an invariant no scope declares — already a validation error from Task 3
      — does not count as a reference, so the invariant it was meant to name is still reported;
      otherwise a typo silences the rule for the invariant it was written for
- [ ] An invariant referenced by both a spec rejection and a rejection edge is reported once, which
      is to say not at all, and an invariant referenced by neither is still reported exactly once
- [ ] The rule's message, severity, position and rule name are unchanged: a leaf compares whole
      formatted lines against US-008 Task 4's own expectations for a model where the rule still fires
- [ ] The change is confined to the collector US-008 Task 4 built: the comparison, the message and
      the scope walk are untouched, and `git diff` in `internal/linter/linter.go` shows the
      collector's body and nothing else
- [ ] A linter fixture whose **only** reference to an invariant is a rejection edge — no spec
      anywhere in the model — reports nothing for that invariant, which is the case the seam exists
      for and the one a spec gate would have broken
- [ ] `oracle.Check` over Task 2's fixture stays empty, and over every other model the repository
      ships as valid stays as US-008 Task 4 left it — `test.InvariantLibraryLending` and the fence at
      `docs/dsl-reference.md:175` are that task's to move, not this one's, and neither gains a
      rejection edge here
- [ ] `internal/validator`, `internal/formatter`, `internal/export`, `internal/diagram` and
      `docs/` are untouched

**Affected Files/Modules:**
- `internal/linter/linter.go` — the reference collector US-008 Task 4 introduces
- `internal/linter/linter_test.go` — leaves inside that rule's group: the rejection-edge reference in
  both scopes, the two non-inheritance cases, the unresolved-name case, and the specs-free model

**Patterns to Follow:**
- The seam and its rationale: `tasks/us-008-lint-spec-coverage-and-boundary-assumptions.md:495-561`,
  especially the "US-009 seam, stated" paragraph at `:554-561`
- The scope rule the collector walks: `invariantScope`/`invariantScopes`
  (`internal/validator/validator.go:203-230`) and the comment at `:215-218`
- Whole-formatted-line assertions for a rule whose message names a symbol and its scope:
  `tasks/learnings.md:476-479` and `:136-139`

**Testable:** Yes — through `linter.Lint` and `oracle.Check`.

**Verification:** `task test:unit` passes; `go run ./cmd/emod lint` over each model the repository
ships as valid exits 0; a scratch model declaring one invariant, naming it from a rejection edge and
stating no spec reports nothing.

**Depends on:** Task 8, and **US-008 Task 4**, which is outside this story and not yet implemented.

---

### Task 10: Accept the rejection entry in the tree-sitter grammar

**Behavior:** The tree-sitter grammar parses a `flow` block holding both entry kinds, so an editor
does not red-squiggle a file `emod validate` accepts.

**Acceptance Criteria:**
- [ ] `editors/tree-sitter-emod/grammar.js` accepts a `flow` block holding any mixture of
      `command -> event:` and `command -> rejected:` entries in any order, and the parse tree tells
      the two apart
- [ ] The two entry rules are **named** rather than hidden. `_flow_entry` (`:272-280`) is hidden
      today, so a `flow_definition`'s children are bare `(identifier)` pairs and a corpus case for a
      rejection entry would produce the shape an event entry already produces — an assertion that
      cannot fail for the reason it was written. Naming both moves the four committed corpus
      expectations that state `flow_definition`'s children:
      `editors/tree-sitter-emod/test/corpus/slice.txt:92` and `:642`,
      `test/corpus/full_model.txt:36`, `test/corpus/specs.txt:88`. Those four files are in this
      task's change set for that reason
- [ ] `test/corpus/` gains a case for a block holding only a rejection entry and a case for a block
      holding both kinds, each with the entry kinds distinguishable in the expected tree
- [ ] The one-line comment above `flow_definition` (`:263`) states the rule's full shape including
      the new entry — it is the only place in the file the admitted entries are listed, and the rule
      body imposes no order (`tasks/learnings.md:251-254`)
- [ ] The grammar is not stricter than the Go parser: it imposes no arity and no ordering on the two
      entry kinds, matching the unbounded loop `parseFlows` runs over the block body
      (`internal/parser/parser.go:1346`) (`tasks/learnings.md:61-64`)
- [ ] `editors/tree-sitter-emod/queries/highlights.scm`, `folds.scm`, `indents.scm` and
      `textobjects.scm` are **not** in this task's change set: `folds.scm:16`, `indents.scm:35` and
      `textobjects.scm:21,51` reference `flow_definition` and not its entries, and `highlights.scm`
      already carries `"flow"` (`:34`) and `"rejected"` (`:62`)
- [ ] `editors/vscode/syntaxes/emod.tmLanguage.json` and `editors/vscode/test/scopes/` are not in
      this task's change set: no keyword is added, and `rejected` is already in the `#keywords`
      alternation at `:97`
- [ ] `task test:grammar` passes, run through `mise exec --` so the repo-pinned tree-sitter CLI
      resolves rather than a globally pinned one (`tasks/learnings.md:11-14`), and after that run the
      only tracked files this task has changed under `editors/` are `grammar.js` and files under
      `editors/tree-sitter-emod/test/corpus/` — the target regenerates artefacts, so checking the
      working tree afterwards is what catches a generated file having become tracked
- [ ] `editors/tree-sitter-emod/src/` stays gitignored and untracked: the repo does not track
      generated artefacts, `task test:grammar` regenerates them, and un-ignoring them would put a
      ~3k-line CLI-dependent diff in this commit (`tasks/learnings.md:16-19`)
- [ ] `TestEditorKeywordCoverage` (`editors/tree-sitter-emod/test/queries/keywords_test.go:47`) and
      `TestKeywordCoverage` (`internal/lsp/keywords_test.go:18`) both pass with no edit — this story
      adds no entry to `internal/lexer/token.go`'s `keywords` map, which reports 38 spellings before
      and after

**Affected Files/Modules:**
- `editors/tree-sitter-emod/grammar.js` — `flow_definition` (`:264-269`), `_flow_entry` (`:272-280`)
  and the comment at `:263`
- `editors/tree-sitter-emod/test/corpus/slice.txt` — the two existing flow expectations (`:92`,
  `:642`) and the new cases
- `editors/tree-sitter-emod/test/corpus/full_model.txt` (`:36`) and `test/corpus/specs.txt` (`:88`) —
  the two other expectations that state `flow_definition`'s children

**Patterns to Follow:**
- A rule offering a choice of shapes after a shared prefix, one of them opening on `rejected`:
  `spec_then` (`editors/tree-sitter-emod/grammar.js:125-131`)
- The one-line example comment every rule carries: `tasks/learnings.md:251-254`
- The grammar must never be stricter than the Go parser: `tasks/learnings.md:61-64`
- Running the target through `mise exec --` and checking the tree afterwards:
  `tasks/learnings.md:11-14`; generated `src/` stays gitignored: `:16-19`
- A test that shells out to a CLI runs with `-count=1`: `tasks/learnings.md:511-514`

**Testable:** Yes — through `tree-sitter test` corpus cases, run by `task test:grammar`.

**Verification:** `mise exec -- task test:grammar` passes; `git diff --exit-code` after it shows only
`grammar.js` and `test/corpus/` files; a scratch `.emod` file stating both entry kinds parses with no
`ERROR` node.

**Depends on:** Task 1.

---

### Task 11: Document the rejection edge in the DSL reference

**Behavior:** The two places `docs/dsl-reference.md` describes what a `flow` block holds and what
references an invariant state the rejection entry, so a reader learns the syntax without reading the
grammar.

**Acceptance Criteria:**
- [ ] §7 "Flows" (`docs/dsl-reference.md:426-438`) states both entry kinds, gives the rejection
      entry's skeleton `command -> rejected: <CommandName> -> <invariantName>`, and says what it
      means — the command is refused and nothing is appended — pointing at
      [`invariant`](#invariant) for where the name is declared
- [ ] §7 states what each diagram surface does with it, in the shape §10's closing bullet (`:627`)
      uses: the draw.io and SVG outputs draw a dashed edge into a rejection badge carrying the
      invariant's statement as a tooltip and a `<title>`, the ASCII preview states the relation as a
      line, and Mermaid draws no arrows so it shows nothing
- [ ] §7's skeleton stays inside a **bare** ``` fence rather than an ` ```emod ` one, as it is today
      (`:430-434`): an ` ```emod ` fence is a promise the block validates whole, and a syntax
      skeleton with `<placeholder>` names does not (`tasks/learnings.md:526-529`). No new
      ` ```emod ` fence is added by this task, and `internal/oracle`'s "documented models" leaf passes
      unchanged over its seven blocks
- [ ] §11's cross-reference table row for `invariant <name>` (`:645`) names the flow rejection entry
      alongside `spec { then rejected <name> }`, and its Context column links to both
      [`spec`](#spec) and [`flow`](#7-flows)
- [ ] §11's "Unresolved rejections" bullet (`:655`) states that the check covers a rejection edge as
      well as a spec's `then rejected`, and that the message is the same
- [ ] §11's row for `command <Name>` (`:643`) is left as it is: a rejection entry names a command,
      but `emod validate` does not resolve it — nor does it resolve a flow's — so listing it would
      state a check that does not exist
- [ ] No heading is added, renamed, renumbered or reordered: the `^## [0-9]+\.` list reconciles
      against the `\(#[0-9]+-` list and the `^### ` list against the `\(#[a-z]` list exactly as they
      do before this task, so the four number-prefixed links and the fourteen sub-heading links —
      `#invariant` and `#spec` among them — still resolve (`tasks/learnings.md:36-39`, `:541-544`)
- [ ] §13's `Diagram Palette` table is untouched and `TestExporterPaletteMatchesReference` passes: a
      rejection badge is not an element type and holds no palette row, and the table is machine-read
      with exactly six rows (`tasks/learnings.md:411-414`)
- [ ] `docs/proposals/` and `user-stories/` are not in this task's change set, and neither is
      `README.md`: the README's `emod diagram` block describes invocations, not model syntax

**Affected Files/Modules:**
- `docs/dsl-reference.md` — §7 "Flows" (`:426-438`) and §11's cross-reference table row and
  validation bullet (`:645`, `:655`)

**Patterns to Follow:**
- The sentence shape for "what the diagram surfaces do with this construct":
  `docs/dsl-reference.md:627`
- The section that already documents a rejection and its name resolution: §6's `### spec`
  (`:376-424`), especially the "Name resolution" paragraph at `:418`
- Heading anchors and how the two link families are reconciled: `tasks/learnings.md:36-39` and
  `:541-544`
- Which fences the oracle harness extracts and which it must not: `tasks/learnings.md:526-529`

**Testable:** No — prose in one document. Correctness is that no anchor breaks, no ` ```emod ` fence
is added, and the two doc-reading Go tests still pass.

**Verification:** `go test -tags unit ./internal/oracle/... ./internal/diagram/...`; reconcile the two
heading/link lists in `docs/dsl-reference.md`; `git diff --stat` shows one document and nothing else.

**Depends on:** Tasks 3, 6 and 8 — the section states what the validator, the diagrams and the linter
do, so each must exist before it is described.

---

## Summary

**Total tasks:** 11.

**Ordering rationale — language, then meaning, then pictures, then wire, then lint, with the two
editor-and-document tasks trailing the behaviour they describe.**

Tasks 1 and 2 make the entry expressible and give every downstream package one model to carry it
through. They are one before the other because the fixture needs the syntax; they are two tasks
because the first already spans the AST, the parser and the formatter, and the kit adds four more
files. Task 1 refuses US-006's parser-then-formatter split for a reason recorded since: a
parser-only commit is a commit at which `emod fmt` silently deletes a user's rejection edges.

Task 3 gives the entry its meaning by reusing US-005's scope rule wholesale — the same
`invariantScopes` construction, the same message, the same sort — so an author sees one behaviour
whether the rejection is written in a spec or on the timeline.

Tasks 4, 5 and 6 are the picture arc, and Task 4 exists to make the other two safe. Putting the edge
kind into `SliceEdges` first is a green commit on its own — verified: SVG's `default:` arm draws
nothing for an endpoint no box carries, and draw.io's switch names no case for it — which turns the
"nothing else moves" claim into a byte-identity receipt taken *before* any drawing lands, and lets
Task 4 settle what the four surfaces that draw no badge do. Task 5 pays for the design in the format
whose harness is most fragile: `svgShapes` binds every `</text>` to the last `<rect>` it saw, and
`svgConnections` fails the whole suite when an arrow ends anywhere but a rect's exact centre — a
badge with a `<title>` at the end of a dashed arrow is both hazards at once. Task 6 is then a second
consumer of geometry that already exists, which is why the cross-format equality of the badge text is
asserted there and not in Task 5: it is the first point at which both badges exist to compare.

Task 7 closes the three coupled export surfaces in one commit, as this repo requires. Tasks 8 and 9
are the lint arc, split so that the rule this story owns is not blocked on a story that has not been
implemented: Task 8 stands alone, and Task 9 names US-008 Task 4 as a hard external dependency and
touches only the collector that task was written to leave behind. Tasks 10 and 11 make the entry
parseable in an editor and findable in the reference.

**Story acceptance criteria, all five covered, none deferred:**

| Story criterion | Task |
|---|---|
| A `flow` block accepts `command -> rejected: <CommandName> -> <invariantName>` entries alongside the existing entry kind | 1 (parse and re-emit), 2 (a model that states them), 10 (the editor grammar) |
| The invariant name resolves against the enclosing aggregate (or context in DCB mode), the same rule as `rejected` in specs; an unresolved name is a validation error | 3 |
| Diagrams render the edge dashed, ending in a rejection badge whose tooltip (draw.io) or `<title>` (SVG) carries the invariant's prose | 4 (the edge derivation, and what ASCII, Mermaid and the diagram document do), 5 (SVG), 6 (draw.io) |
| `flow/rejection-without-spec` (info) fires when a rejection edge has no spec on the slice exercising that rejection | 8 |
| A rejection edge counts as a reference for `spec/invariant-never-exercised` | 9 |

Task 7 (exports and schema) and Task 11 (the reference) carry no story criterion. They are the two
places this repo's conventions say a new construct must reach regardless: a field that lands in JSON
without CUE and `schema.cue` fails two coupled subtests, and `docs/dsl-reference.md` §7 currently
documents the flow block's *only* entry kind, which the story makes false.

**Three things a reader of this list should carry forward.**

First, the AST decision in "Open questions, decided" item 1 is the one everything else rests on. A
rejection entry sharing `ast.Flow` would have changed the behaviour of eleven production readers of
`Slice.Flows` at once, most of them silently — `left-chair` counting rejections toward its threshold,
`emod slices list` calling a rejection-only slice a command slice, the LSP resolving an invariant name
as an event. A separate collection makes each of those an explicit opt-in, and this story opts three
of them in and leaves eight alone.

Second, the story's third criterion names two formats, and this list decides for the other four. ASCII
states the relation, because it is the format that already renders every arrow as text and silently
dropping one is the trap `tasks/learnings.md` records against `ExportSVG`'s discarded `Style`. Mermaid
does not, because it draws no arrows at all — verified across all three of its layouts. The
diagram-JSON document does not, following the `EdgeTranslationExternal` precedent one case away in the
same switch, and the consequence is stated rather than hidden: a model saved out of the web viewer
loses its rejection edges, exactly as it already loses its specs and its invariants.

Third, Task 9 is the only task blocked on work outside this story, and the block is real: US-008 Task
4 must land first, with the collector/comparison split it was written to leave. The other ten tasks
depend on nothing outside this list.

**Findings surfaced during decomposition that belong to no story yet, recorded rather than absorbed.**
No test in the repository asserts any of the seven diagnostics `parseFlows` and `parseFlowEntry`
produce (`internal/parser/parser.go:1352`, `:1375-1405`) — Task 1 adds coverage for the arm it
changes and leaves the rest as it found them. And `ConvertDiagnostics`
(`internal/lsp/diagnostics.go:31-36`) maps `diagnostic.Info` onto the LSP **error** severity, so
Task 8's info rule will draw a red squiggle in an editor, as `dcb/single-tag-everywhere` already does;
fixing it changes an existing rule's behaviour and belongs in its own commit.

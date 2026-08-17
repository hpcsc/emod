# US-016: Render specs on diagrams

## Progress
- [x] Task 1: Draw a slice's specs as a card in the SVG diagram
- [x] Task 2: Draw the same card in the draw.io diagram
- [x] Task 3: Render the cards from the command line behind `--specs`
- [ ] Task 4: Document `--specs` where the diagram command is described

---

## Story Reference

`user-stories/specs-and-metadata.md` → **US-016: Render specs on diagrams** (sixteenth story of
"Specs, Invariants, and Model Metadata", lines 205-214). Design notes:
`docs/proposals/specs-and-metadata-proposal.md` §4 "Pattern variants" for the four `then` shapes the
card has to state, and the worked example at lines 537-575.

**Prerequisite, hard.** This story renders what US-006 and US-007 put in the AST. US-006 is delivered
on main (`ast.Spec`, `ast.SpecElement`, `ast.ThenEvents`, `ast.ThenRejected`,
`internal/ast/ast.go:95-126`). US-007 is decomposed but not implemented
(`tasks/us-007-write-specs-for-view-automation-and-translation-slices.md`), and this story's third
criterion — a card showing "given, when, and then" for every slice — cannot be met for view,
automation and translation slices until US-007 Task 1 lands `ast.ThenView` and `ast.ThenCommand` and
US-007 Task 2 lands the four-pattern fixture. Task 1 below is blocked on both. Nothing here is
written against a placeholder: the card's `then` dispatch covers four variants from its first commit.

**In scope:** a rendering option on the two picture exporters that draws each spec-stating slice's
specs as a Given-When-Then card under that slice, in a band below the lowest lane; the card stating
each spec's name, its `given` events, its `when`, and its `then` outcome in all four shapes —
including a rejection stating the invariant's name; the same card in draw.io; a `--specs` flag on
`emod diagram` that turns the option on for the two formats that draw cards and refuses the surfaces
that do not; and the flag documented where the diagram command is described. Off by default
throughout: a model rendered without the option produces the bytes it produces today, and a model
whose slices state no spec produces those same bytes *with* the option.

**Out of scope**, named:

- **Rejection edges on the timeline** — `command -> rejected: <Cmd> -> <inv>` in a `flow`, the dashed
  edge, the rejection badge, and the invariant prose carried as a draw.io tooltip or an SVG `<title>`
  (US-009). That story changes the *edge*; this one adds a *card*, and the two never meet: no card
  carries an invariant's prose, only its name, and this story adds no `EdgeKind` to
  `internal/diagram/graph.go` and no arrow of any sort.
- **Example payloads on element references** (US-010). `SpecElement` may or may not carry a `Payload`
  by the time this lands; either way the card states element *names* only. The story's own criteria
  name given, when, then and rejected outcomes and nothing else, and the file's Non-Goals defer
  expected view-state payloads outright.
- **`emod fmt`** (US-014), **LSP** (US-015), **syntax highlighting** (US-017) and
  **examples / DSL reference sweep for the spec constructs** (US-018). Task 4 documents one CLI flag;
  it adds no fenced `emod` block, renames no heading, and touches no `.emod` file.
- **The four `spec/*` lint rules** (US-008). No rule is added, so no model that renders a card starts
  failing `emod validate`.
- **The diagram-JSON document and the viewer.** `internal/export/diagram.go` carries no spec key today
  and gains none here; `internal/viewer/static` draws no card and `internal/importer` reads none back.
  A user who saves from the viewer loses nothing, because nothing spec-shaped was ever put on a
  diagram node. `emod diagram --serve` therefore refuses `--specs` rather than serving a picture the
  flag cannot change (Task 3).
- **Mermaid and ASCII cards.** `ExportMermaid` and `ExportASCII` keep their present two-parameter
  signature, so the type system states which formats can draw a card, and the CLI refuses `--specs`
  for them rather than accepting a flag it would ignore.
- **Recovering flags written after the file argument.** See "Open questions, decided" item 6.

**Open questions, decided.** The story fixes four criteria and leaves the shape of the thing entirely
open. Each decision below is made so that the off-by-default receipt stays byte-exact and so US-009
and US-010 stay additive:

1. **The option rides on the two picture exporters as a trailing variadic `Option`, not as a fifth
   positional parameter.** `internal/diagram` gains an exported `Option` type and a `WithSpecs`
   constructor, and `ExportSVG` and `ExportDrawio` take a variadic run of options after the `Style`
   they already take. The repo already has this shape: `Option` and `WithHTTPClient` in
   `internal/llm/bedrock/adapter.go:27-40`, consumed by `New` at `:54`. The reason
   is a measurement, not taste: the four exporters are called from 111 sites across
   `internal/diagram/*_test.go` plus four in `internal/cli/diagram.go`, and a positional `bool` or a
   widened options struct moves every one of them inside the commit that adds the feature — the exact
   "~300 mechanical lines inside a field-addition commit with nothing showing the output did not move"
   that `tasks/learnings.md` records as the thing reviewers flag. With a variadic option, no existing
   call site moves, and the diff is the feature.
2. **A card sits in a band below the lowest lane, not inside one.** Every existing box keeps the
   coordinates it has today, in both picture formats and in all three draw.io lane sets, and the
   picture grows only downwards (in SVG, by the band's height; draw.io declares no canvas size —
   `xmlProlog`, `internal/diagram/drawio.go:479-485` — so it grows for free). This is what makes the
   featured render provable: "every box other than the cards is drawn at the same rect it is drawn at
   without the option" is a whole-list comparison, not a coordinate restatement. Putting cards inside
   the Commands / Views lane instead would reflow the reactor row and move boxes, which no criterion
   of this story asks for.
3. **The band is drawn like a lane and labelled `Specs`, but takes its height from the tallest card
   rather than the fixed `laneHeight`.** A band spanning the picture is the vocabulary both formats
   already have for "a horizontal region with a name" (`svgLane`, `internal/diagram/svg.go:258`;
   `swimlaneCell`, `drawio.go:502`). The consequence to hold in mind is a test one:
   `svgLaneLabels` (`internal/diagram/svg_test.go:415-437`) identifies lanes as *the widest shapes in
   the picture*, so under `--specs` it reports `Specs` alongside the four it reports today, and any
   assertion over lane labels in a featured render expects five. `laneHeight` is 190 and a card
   stating three specs will not fit it, so the band is sized, not constant.
4. **A card states, per spec, in declaration order: the spec's name, its `given` event names, its
   `when` name, and its `then` outcome.** The name is on the card because
   `test.SpecLibraryLending`'s "Borrow Copy" states three specs and a card listing three unnamed
   given/when/then triples reads as noise. A spec that omits `given` and one that writes `given []`
   both contribute no given line, which is exactly what `emod fmt` writes for the two spellings
   (`tasks/learnings.md`, "`emod fmt` moves a spec to the end of its slice"), so the card and the
   formatted source agree. A spec with no `when` — the view and schedule-driven-automation shapes —
   contributes no when line.
5. **The card is not an element type.** It gets its own fill and stroke constants in
   `internal/diagram/layout.go:71-89` beside the six that exist, and it gets **no** `nodePalette` entry in
   `internal/viewer/static/config.js`, no `.hl.<type>-block` rule in `viewer.html`, and no row in the
   `Diagram Palette` table of `docs/dsl-reference.md` §13. `TestExporterPalette`,
   `TestExporterPalettePinsViewer` and `TestExporterPaletteMatchesReference`
   (`internal/diagram/contract_test.go:1388-1446`) must still see exactly six element types and must
   pass with no edit. The precedent is the external system box, which `tasks/learnings.md` records as
   "not an element type and holds no palette entry of its own". A seventh palette entry would break
   three Go tests, a JS surface and a machine-read markdown table for a shape no viewer draws.
6. **`--specs` is written before the file argument, and this story does not teach `diagram` to
   recover flags written after it.** `tasks/learnings.md` records that urfave/cli v2 stops parsing at
   the first positional argument; re-verified here — `emod diagram examples/all_patterns.emod
   -f mermaid -o /tmp/x.mmd` wrote `examples/all_patterns.drawio` and no `/tmp/x.mmd`. So
   `emod diagram model.emod --specs` draws no card and reports nothing, and `README.md:160-163`
   documents four invocations in that same broken shape. Fixing it for `diagram` means rescanning the
   leftovers the way `glossaryPathAndFormat` (`internal/cli/glossary.go`) does, which would
   simultaneously change what `-f`, `-o` and `--style` do for every existing invocation and every
   README example — a behaviour change for four flags this story does not own. Every criterion,
   example and doc line here therefore spells `emod diagram --specs <file>`, and the general fix
   belongs in a story of its own.
7. **`--specs` is refused, not ignored, on the surfaces that draw no card.** `-f mermaid`, `-f ascii`
   and `--serve` each exit 1 with a message naming the flag and the two formats that render cards.
   The repo's existing precedent is the opposite — `ExportSVG` and `ExportASCII` declare `_ Style`
   and silently ignore a `--style` they cannot honour, which `tasks/learnings.md` records as a trap
   ("a subtest parameterised over `StyleDCB` asserts nothing its `StyleAuto` case did not"). A refusal
   is also the additive direction: turning an error into output later breaks nobody, while turning
   silence into output changes what a working script produces.

**Overarching constraint:** the story's second criterion — "the flag is off by default; without it,
diagram output is unchanged" — is the load-bearing one, and this repo already has the receipt for it
in place. `TestExporterContract` → "stating specs leaves the picture untouched"
(`internal/diagram/contract_test.go:441-453`) renders `test.SpecLibraryLendingModel` against its
`WithoutSpecs` twin across all four exporters and requires the bytes to be equal. That subtest runs
the *default* render, so it must keep passing **with no edit** through every task here. Its sibling —
that the featured render does move, and moves only by cards — is what Task 2 adds, so the pair reads
as one claim rather than as a comparison whose twin was never proved to differ.

**Learnings folded in** from `tasks/learnings.md`:

- *"`svgPicture` sees labelled boxes and arrows only"* and its neighbour on `svgShapes`
  (`internal/diagram/svg_test.go:354-397`) — verified here: the walker appends a shape on each
  `<rect>` and, on each closing `</text>`, overwrites the label of the shape it appended most
  recently (`:392`). Two hazards follow
  and Task 1 closes both. A `<text>` emitted with **no rect before it** overwrites the label of
  whatever box was drawn last, silently corrupting an unrelated assertion — this is the US-013
  finding, and a multi-line card is its larger form. And a card drawn as *several* `<text>` elements
  keeps only the last one as its label, so every assertion about the card's contents would read one
  line. The card is therefore one rect followed by exactly one `<text>`, multi-line through
  `svgMultilineText` (`internal/diagram/svg.go:273-295`) the way a two-line event label already is
  (`:112`).
- *"An arrow between two constructs is drawn by six surfaces, and none of them reads another"* — the
  inverse applies: `internal/diagram/svg.go` and `drawio.go` each walk the AST themselves, so the
  card has to be taught to both, and `SliceEdges` (`graph.go:51`) is untouched because a card is not
  an edge. Mermaid, ASCII, the diagram document, the importer and the viewer are five further
  independent surfaces, each of which stays exactly as it is.
- *"SVG and draw.io share box placement, and share it through `drawio.go`"* — placement has since moved and lives in
  `internal/diagram/layout.go` (`laneRowY:209`, `itemLayout:264`, `reactorBoxes:226`, the last taking
  the format's line break as a parameter). The card's geometry and its lines are computed once and
  consumed by both formats, with the line break parameterised, exactly as `reactorBoxes` does.
- *"De-duplicate before a fan-out edit"* — the card's text is derived once. `internal/diagram/labels.go`
  is where `automationLabel` already turns an AST node plus a line break into a drawn label, and it is
  where the card's lines belong.
- *"`ast.ThenClause` dispatches through five type switches, none of which errors"* — the card is the
  sixth. It must state a `then` line for every variant, and a variant it has not heard of must not
  fall through to an empty line. The receipt is one leaf per outcome kind, over a fixture that
  declares all four.
- *"Allocate a draw.io cell id only once the cell is certain to be written"* (`allocID`,
  `internal/diagram/drawio.go:53`) — a card is conditional twice over (the option, and whether the
  slice states a spec), so an id taken before both guards renumbers every later cell and fails a
  differential on ids alone.
- *"Read an SVG or draw.io arrow back as the two boxes it meets"* and *"Assert diagram box placement
  relationally, by label"* — cards are asserted through `svgBoxes`/`drawioBoxes` and the relational
  helpers (`labelsWithin`, `labelsBelow`, `lowestEdge`, `boxesDrawnOver`,
  `internal/diagram/contract_test.go:259-334`), never by restating a coordinate.
- *"A differential receipt must first prove the twin actually differs"* and *"`require.NotEqual` on a
  stripped twin is satisfiable without stripping anything"* — every comparison in Tasks 1 and 2 opens
  by reading the fixture's transcribed spec names back off the stated model and requiring the twin's
  to be empty, the shape `contract_test.go:441-453` already uses.
- *"`ExportSVG` and `ExportASCII` ignore the `Style` they are handed"* — an SVG subtest parameterised
  over `StyleDCB`/`StyleProjected` asserts nothing, so the band's placement under the non-standard
  lane sets is a draw.io claim only (Task 2).
- *"urfave/cli v2 discards every flag written after the file argument"* — decided in item 6 above.
- *"CLI diagnostic tests must assert the distinguishing message text"* — Task 3's refusal leaves
  assert the flag name and the formats the message lists, not merely a non-zero exit.
- *"New file-taking CLI commands compose `parseModelFile` and `reportExitError`"* and *"`RuleName`
  marks a diagnostic `emod lint --explain` can describe"* — the refusal is a `*LintError` with the
  `ErrUnsupportedFormat` family's shape (`internal/cli/lint.go:18`), carries no `RuleName`, and needs
  no entry in `internal/linter/descriptions.go`.
- *"`docs/dsl-reference.md` sub-heading anchors are cited more often than the numbered ones"* — `#spec`
  is cited three times, so Task 4 adds a bullet under `### spec` and does not rename it.
- *"An ```emod fence is a promise that the block validates"* — Task 4 writes prose and `bash` fences
  only, so `internal/oracle`'s "documented models" leaf gains nothing to check and keeps passing.
- *"A task criterion requiring 'committed' output cannot close"* and *"A commit-message receipt is the
  commit author's obligation"* — every criterion below is checkable in an uncommitted working tree.

---

## Codebase Context

**What a spec is, in the AST.** `internal/ast/ast.go`: `Slice.Specs []*Spec` (`:90`), `Spec` (`:95-104`,
carrying `Name`, `Given []*SpecElement`, `When *SpecElement`, `Then ThenClause`), `SpecElement`
(`:106-109`, a name and a position), and the sealed `ThenClause` (`:111-126`) with `ThenEvents` and
`ThenRejected` today, plus `ThenView` and `ThenCommand` once US-007 Task 1 lands. A slice has two
homes — `agg.Slices` and, in a `mode dcb` context, `ctx.Slices` — and `collectSlices`
(`internal/diagram/layout.go:100-111`) already reaches both through `model.SliceRefs()`, so the
diagram package needs no new walk.

**Specs reach no picture today.** Confirmed by search: `internal/diagram/*.go` contains no reference
to `Spec` at all, and `internal/export/diagram.go` contains none either. The only mention anywhere in
the two packages is the contract subtest asserting the picture does *not* move
(`internal/diagram/contract_test.go:441`). This is the central fact the story changes, and it is why
the story is four tasks rather than a fan-out across every writer: nothing downstream has a spec-shaped
hole to fill.

**The layout model.** `internal/diagram/layout.go` holds everything both picture formats share:
`sliceEntry` and `collectSlices` (`:91-111`), `sliceXPositions` (`:115-132`, one x per slice, a wider
gap at a context boundary), `contextBounds` (`:146-163`), `layoutWidth` (`:166-171`), `laneRowY`
(`:209-211`), `reactorBoxes` (`:226-242`, the shared automation/translation row, taking `lineBreak`),
and `itemLayout` (`:264-274`). Layout constants at `:51-68` (`marginX 40`, `marginY 60`,
`sliceWidth 280`, `boxWidth 240`, `boxHeight 55`, `laneHeight 190`, `laneGap 30`,
`laneHeaderHeight 30`), colour constants at `:71-89`. `internal/diagram/labels.go` (25 lines) turns AST
nodes into drawn text: `reactorLabel`, `cadenceLabel`, `automationLabel(auto, lineBreak)`.

**SVG.** `ExportSVG(model *ast.Model, _ Style)` (`internal/diagram/svg.go:12`) ignores the style
entirely and always draws four lanes — `Wireframes`, `Commands / Views`, `Events`,
`External Systems` — at `triggerLaneY = marginY` and each `laneHeight + laneGap` below the last
(`:34-43`). The canvas is `diagramW = layoutWidth(sliceXs) + marginX + 120` by
`diagramH = 2*marginY + 4*laneHeight + 3*laneGap` (`:22-23`), both computed before anything is drawn.
Shapes are emitted through `svgRect`/`svgRoundedRect`/`svgRectElement` (`:216-256`, self-closing
unless a description makes it carry a `<title>`), text through `svgText` (`:268`) and
`svgMultilineText` (`:273`, one `<text>` with a `<tspan>` per line). Boxes are recorded in one
`nameToBox` map so arrows can cross slices (`:57`), and arrows are drawn last from `SliceEdges`
(`:174-195`).

**draw.io.** `ExportDrawio(model *ast.Model, style Style)` (`internal/diagram/drawio.go:27`) branches
on style into three lane sets — standard four, DCB's `Triggers / Commands` + `Events` +
`External Systems`, and projected's per-tag lanes — with `extLaneY` the lowest in all three
(`:66-140`). `allocID` (`:53-57`) is a running counter shared by every vertex and edge. Cells are
written by `swimlaneCell` (`:502`), `vertexCell` (`:513`, plain `mxCell` unless a tooltip wraps it in
an `<object>`), `triggerFramingCell` (`:532`) and the two edge builders (`:546`, `:552`). Multi-line
values use a literal `\n` inside the value (`:214`, `:240`). There is no canvas size to grow:
`xmlProlog` (`:479-485`) writes `<mxGraphModel dx="0" dy="0" grid="0" gridSize="10">`.

**How the pictures are tested.** `internal/diagram/contract_test.go` is the shared harness: an
`exporter` struct (`:27-46`) with per-format readers — `fillOfLabel`, `strokeOfLabel`,
`countConnections`, `boxes`, `connections`, `export`, `requireWellFormed` — nil for the text formats,
so a picture-only subtest is written once and skipped for mermaid and ASCII by a `nil` guard.
`diagramBox{label, appearance, rect}` (`:47-53`) and `diagramConnection{source, target, paint}`
(`:78-84`) are what a picture reads back as, and `boxRect` carries `centre`, `overlaps` and `within`
(`:56-77`). The relational helpers are `boxesDrawnOver` (`:259`), `gearedBoxes` (`:277`),
`labelsWithin` (`:299`), `labelsBelow` (`:314`) and `lowestEdge` (`:326`). `e.run(t, model, style)`
(`:218-225`) is the single call every subtest makes. The umbrellas are `TestExporterContract` (`:349`),
`TestExporterTranslationEdges` (`:462`), `TestExporterReadsEdges` (`:548`),
`TestExporterAutomationSchedule` (`:853`), `TestExporterReactorPlacement` (`:1061`),
`TestExporterTriggerScreen` (`:1237`) and the three palette tests (`:1388`, `:1414`, `:1446`).
Format-specific detail lives in `TestExportSVG` (`internal/diagram/svg_test.go:19`) and
`TestExportDrawio` (`drawio_test.go:20`); the readers are `svgShapes`/`svgBoxes`/`svgLaneLabels`/
`svgConnections` (`svg_test.go:354-500`) and `drawioBoxes`/`drawioEdges` (`drawio_test.go`).

**The CLI.** `RunDiagram(path, outputPath, format string, style diagram.Style) error`
(`internal/cli/diagram.go:29-117`): read, `oracle.Run`, print diagnostics, exit 2 on any error,
validate the format against the four names (`:58-64`, a `*LintError` with `ErrUnsupportedFormat`),
dispatch (`:68-77`), print to stdout for mermaid and ascii with no `-o`, otherwise write a file whose
default path comes from `defaultDrawioPath`/`defaultSVGPath` (`:128-142`), and finish through
`lintExit` (`:119-124`), which is exit 1 when any warning was reported. `RunDiagramServe`
(`:148-186`) is the `--serve` path and never reaches `RunDiagram`. The command is declared at
`internal/cli/app.go:113-149` with `--format`, `--style`, `-o` and `--serve`, and its `Action` is the
usual one-liner into `reportExitError`. `internal/cli/diagram_test.go` is one `TestDiagram` umbrella
(`:23`) of flat subtests, including "unsupported diagram format returns error listing supported
formats" (`:258`) and a `serve` group (`:381`).

**Fixtures.** `test.SpecLibraryLending` (`internal/test/fixtures.go:423-572`) states seven specs
across four slices in both homes, covering `then [Events]` and `then rejected` — but no `then view`
and no `then command`. Its kit is `SpecLibraryLendingSpecNames` (`:1115-1127`), `WithoutSpecs`
(`:1212`) and `DeclaredSpecNames` (`:1296-1311`). US-007 Task 2 adds the sibling four-pattern
fixture — a command slice with both outcomes, a view slice with `then view`, an event-driven and a
schedule-driven automation slice with `then command`, and a translation slice — in both homes, with
its own transcribed spec-name and outcome-kind lists and its own parsed-model helper in
`internal/test/models.go`. That fixture is the one this story renders; `test.SpecLibraryLending`
remains the model every off-by-default receipt is written against, because six downstream expected
values transcribe it and none of them may move.

**Paths `tasks/learnings.md` still cites from an earlier tree.** Two entries above are quoted for
their content, not their coordinates. `internal/export/export.go` no longer exists — the package is
`json.go`, `cue.go` and `diagram.go`, and everything the learnings file says about `jsonDiagramNode`,
`convertModelToDiagram` and `collectSliceNodes` now lives in `internal/export/diagram.go`. And the
layout constants, colour constants, `laneRowY`, `itemLayout` and `reactorBoxes` have moved out of
`internal/diagram/drawio.go` into `internal/diagram/layout.go`, which several entries — including
"SVG and draw.io share box placement, and share it through `drawio.go`" and "One fill and stroke
constant per element type" — still place in the old file. The claims hold; the line references do not.

**Docs.** `README.md:157-164` is the "Generate diagrams" block, four `bash` lines.
`docs/dsl-reference.md` has `### spec` (§6, `:376`) and a `## 12. Pipeline` section (`:659`); §10's
closing bullet (`:627`) is the precedent for "here is what the diagram surfaces do with this
construct". `docs/architecture.md:196` is a one-row-per-command CLI table.

---

## Tasks

### Task 1: Draw a slice's specs as a card in the SVG diagram

**Behavior:** `ExportSVG` accepts a rendering option that draws, under each slice stating at least one
spec, a card listing that slice's specs — each with its name, its `given` events, its `when`, and its
`then` outcome in whichever of the four shapes it takes. The cards sit in a labelled band below the
lowest lane, one per slice, in that slice's column; the picture grows downwards and nothing already
drawn moves. Without the option, `ExportSVG` returns the bytes it returns today.

**Acceptance Criteria:**
- [x] `internal/diagram` exports an `Option` type and a `WithSpecs()` constructor, and `ExportSVG`
      takes options after the `Style` it already takes; every existing call to `ExportSVG` in the
      repository compiles unedited, and `git diff` shows no line changed in
      `internal/diagram/mermaid.go`, `ascii.go`, `graph.go` or `internal/export/`
      — with one forced exception: `contract_test.go` assigns `ExportSVG` as a *value* to the
      harness's `export` field, and Go does not accept a variadic function where a non-variadic
      function type is declared, so that one line becomes `exportSVGDefault`. Calls are unaffected
- [x] Rendered with `WithSpecs()`, the four-pattern fixture US-007 Task 2 adds draws one card for each
      slice that states a spec and no card for a slice that states none; the cards' labels, read back
      through `svgBoxes`, name every spec in that fixture's transcribed spec-name list, in declaration
      order, across both slice homes
- [x] A card states, for each spec it lists: the spec's name; the names of its `given` events in the
      order written; the name its `when` states; and its `then` outcome — the event names for
      `ThenEvents`, the invariant's name for `ThenRejected`, the view's name for `ThenView`, the
      command's name for `ThenCommand`. One leaf per outcome kind, each failing if that kind's card
      line is empty or states another kind's text
- [x] A rejection's card line names the invariant and not its prose statement: rendering the
      four-pattern fixture with `WithSpecs()` produces output containing no invariant `statement`
      string declared anywhere in that fixture
- [x] A spec that omits `given` and a spec that writes `given []` produce the same card lines as each
      other, and neither states an empty given
- [x] A spec that states no `when` — the view and schedule-driven automation shapes — produces a card
      with no when line and with its given and then lines intact
- [x] `svgShapes` (`internal/diagram/svg_test.go:354-397`) over the featured render reports exactly one
      more shape per card than the default render reports in total, and the label of every shape the
      default render produces is byte-identical in both — no card text attaches to a box drawn before
      it. A leaf asserts this as a whole-list comparison of the default render's shapes against the
      featured render's leading shapes, so a card emitting a stray `<text>` fails it
- [x] `svgConnections` over the featured render reports the same arrows, by source label, target label
      and paint, as over the default render — a card is drawn into no arrow and captures no arrow's
      endpoint
- [x] Every box the default render draws is drawn at the same `boxRect` in the featured render, and
      `boxesDrawnOver` reports no overlap among the featured render's boxes — cards included
- [x] The featured render's SVG is well-formed XML and its `viewBox` height is greater than the default
      render's, while its width is unchanged
- [x] Without `WithSpecs()`, `ExportSVG` over `test.SpecLibraryLendingModel` and over its
      `test.WithoutSpecs` twin returns identical bytes, with the twin first proved to have lost the
      specs of both homes (`test.DeclaredSpecNames` empty) and the stated model proved to read back
      `test.SpecLibraryLendingSpecNames` in full — this is exactly what `contract_test.go`'s "stating
      specs leaves the picture untouched" already asserts for the SVG exporter, so it is met by that
      subtest passing unedited rather than by a second copy of it in `svg_test.go`
- [x] With `WithSpecs()`, a model no slice of which states a spec renders byte-identically to the same
      model without the option — no band, no label, no change in height
- [x] `internal/diagram/contract_test.go` "stating specs leaves the picture untouched" (`:441-453`)
      passes with no edit, and so do `TestExporterPalette`, `TestExporterPalettePinsViewer` and
      `TestExporterPaletteMatchesReference` — this task adds no element type, so
      `internal/viewer/static/config.js` and `docs/dsl-reference.md` are not in its change set
- [x] No existing expected value in `internal/diagram/svg_test.go`, `drawio_test.go`,
      `mermaid_test.go`, `ascii_test.go` or `contract_test.go` moves; the diff there is additions only
      apart from the one forced `export:` field noted above, which is not an expected value

**Affected Files/Modules:**
- `internal/diagram/svg.go` — `ExportSVG` (`:12`) takes the option; the band and its cards are drawn
  after the four lanes and their contents, and `diagramH` (`:23`) grows by the band
- `internal/diagram/layout.go` — the card's geometry beside `reactorBoxes` (`:226-242`): where the band
  starts, how tall each card is, and where a slice's card sits in its column
- `internal/diagram/labels.go` — the card's lines, beside `automationLabel` (`:18-25`), taking the
  format's line break the way `reactorBoxes` does
- `internal/diagram/svg_test.go` — the card leaves and the two hazard receipts, under `TestExportSVG`
  (`:19`)

**Patterns to Follow:**
- A shared placement helper taking the format's line break as a parameter: `reactorBoxes`
  (`internal/diagram/layout.go:226-242`) and its two consumers, `internal/diagram/svg.go:126-134` and
  `internal/diagram/drawio.go`; `tasks/learnings.md` "SVG and draw.io share box placement" and
  "De-duplicate before a fan-out edit"
- A label derived once from an AST node: `automationLabel` (`internal/diagram/labels.go:18-25`)
- A multi-line SVG label as one `<text>`: `svgMultilineText` (`internal/diagram/svg.go:273-295`) and
  its use for a two-line event label at `:112` — the shape that keeps `svgShapes` honest
- The functional-option shape: `internal/llm/bedrock/adapter.go:27-40` and `:54`
- Differential receipts that first prove the twin differs: `internal/diagram/contract_test.go:441-453`
  and `:427-439`; `tasks/learnings.md` "A differential receipt must first prove the twin actually
  differs" and "`require.NotEqual` on a stripped twin is satisfiable without stripping anything"
- Reading a picture back relationally rather than by coordinate: `svgBoxes`/`svgShapes`/`svgConnections`
  (`internal/diagram/svg_test.go:354-500`), `boxesDrawnOver`/`labelsWithin`/`labelsBelow`/`lowestEdge`
  (`contract_test.go:259-334`); `tasks/learnings.md` "Assert diagram box placement relationally, by
  label"
- The `ThenClause` dispatch this task joins as the sixth site: `formatOutcome`
  (`internal/formatter/formatter.go:390`) is the closest reader, and `tasks/learnings.md`
  "`ast.ThenClause` dispatches through five type switches, none of which errors" names the rest
- `tasks/learnings.md` "`svgPicture` sees labelled boxes and arrows only, so it is not the receipt for
  a new mark" — the reason the hazard receipt is written against `svgShapes` and not `svgPicture`

**Testable:** Yes — `diagram.ExportSVG`, `diagram.WithSpecs` and the `internal/test` fixture kit are
all exported.

**Verification:** `go test -tags unit ./internal/diagram/...`; `go build ./...`; `git diff --stat` shows
`internal/diagram/{svg.go,layout.go,labels.go,svg_test.go}` and nothing else.

**Depends on:** US-007 Task 1 (`ast.ThenView` and `ast.ThenCommand`) and US-007 Task 2 (the
four-pattern fixture) — both outside this story and both required before this task starts.

---

### Task 2: Draw the same card in the draw.io diagram

**Behavior:** `ExportDrawio` accepts the same option and draws the same cards, below the lowest lane
of whichever of its three lane sets the style selected. The two picture formats state the same card
text for the same model, differing only in how each spells a line break, and the contract harness
asserts the card once for both.

**Acceptance Criteria:**
- [x] `ExportDrawio` takes options after the `Style` it already takes; every existing call to it in the
      repository compiles unedited — as with `ExportSVG` in Task 1, the one *assignment* of it as a
      function value (the contract harness's `export` field) becomes `exportDrawioDefault`, which Go's
      assignability rule forces and which no call site is affected by
- [x] Rendered with `WithSpecs()`, the four-pattern fixture draws one draw.io card per spec-stating
      slice, and `drawioBoxes` reads their labels back naming every spec in the fixture's transcribed
      spec-name list, in declaration order
- [x] The card text draw.io states and the card text SVG states are equal once each format's line break
      is normalised — one contract-level leaf comparing the two formats' card labels for one model, so
      a line added to one card and not the other fails
- [x] Under `StyleAuto`, `StyleDCB` and `StyleProjected`, the band is drawn below the lowest lane the
      style draws, and no cell of any lane the style draws overlaps a card; this is asserted in
      draw.io only, because `ExportSVG` ignores the `Style` it is handed
      (`tasks/learnings.md`, "`ExportSVG` and `ExportASCII` ignore the `Style` they are handed")
- [x] Every cell the default render writes appears in the featured render with the same id, the same
      geometry and the same style string — no card id is allocated before both the option and the
      "this slice states a spec" guard have been passed
- [x] `drawioEdges` over the featured render reports the same arrows, by source label, target label and
      paint, as over the default render
- [x] `internal/diagram/contract_test.go` gains a subtest, run for both picture exporters and skipped
      for the text ones by the existing `nil` guard, asserting that the featured render of a
      spec-stating model differs from its default render and that the only boxes it adds are the cards
      plus the band — the positive sibling of "stating specs leaves the picture untouched", which
      still passes with no edit
- [x] Without the option, `ExportDrawio` over `test.SpecLibraryLendingModel` and over its
      `test.WithoutSpecs` twin returns identical bytes, with the twin proved empty of specs first —
      met by `contract_test.go`'s "stating specs leaves the picture untouched" passing unedited, which
      already runs for the draw.io exporter, rather than by a second copy in `drawio_test.go`
- [x] With the option, a model no slice of which states a spec renders byte-identically to the same
      model without it
- [x] The three palette tests pass with no edit and still see exactly six element types

**Affected Files/Modules:**
- `internal/diagram/drawio.go` — `ExportDrawio` (`:27`) takes the option; the band and its cards are
  written after the lanes and their cells, below `extLaneY` in each of the three lane sets; the card's
  draw.io style string joins the `style*` constants (`:12-25`)
- `internal/diagram/layout.go` — the card's fill and stroke join the colour constants (`:71-89`)
- `internal/diagram/layout.go` / `labels.go` — consumed unchanged from Task 1, with the draw.io line
  break passed in
- `internal/diagram/drawio_test.go` — the draw.io card leaves and the style-dependent placement leaves
- `internal/diagram/contract_test.go` — the cross-format card subtest

**Patterns to Follow:**
- Conditional cell allocation: the `readsEdge` closure (`internal/diagram/drawio.go:467`) resolves its
  target and returns before calling `allocID` (`:54-58`); `tasks/learnings.md` "Allocate a draw.io cell
  id only once the cell is certain to be written"
- A multi-line draw.io value: the two-line command and event labels at
  `internal/diagram/drawio.go:214` and `:240`
- Colour constants named for what they paint, never aliased onto an equal value:
  `internal/diagram/layout.go:71-89` and `tasks/learnings.md` "One fill and stroke constant per element
  type — never named after the shape, never aliased" (which still cites the constants' old home in
  `drawio.go`; they live in `layout.go` now)
- A subtest written once for both picture formats and skipped for the text ones: the nil-`boxes` guard
  opening `TestExporterReactorPlacement` (`internal/diagram/contract_test.go:1061-1066`)
- Reading draw.io back by label and by the two boxes an arrow meets: `drawioBoxes` and `drawioEdges`
  (`internal/diagram/drawio_test.go`); `tasks/learnings.md` "Read an SVG or draw.io arrow back as the
  two boxes it meets"

**Testable:** Yes — through `diagram.ExportDrawio` and the contract harness.

**Verification:** `go test -tags unit ./internal/diagram/...`; `go build ./...`.

**Depends on:** Task 1

---

### Task 3: Render the cards from the command line behind `--specs`

**Behavior:** `emod diagram --specs <file>` writes a diagram whose slices carry their spec cards, for
the two formats that draw them. Without the flag, every diagram the command has ever written is
unchanged. Asked for cards on a surface that draws none — mermaid, ASCII, or the viewer — the command
refuses with a message naming the flag and the formats that do.

**Acceptance Criteria:**
- [x] `emod diagram --specs <file>` with the default format writes a draw.io file whose content names
      each spec the model states; the same invocation with `--format svg` writes an SVG that does —
      the criterion's `-f` spelling does not exist for this command and never has: only `glossary`
      declares an `f` alias (`internal/cli/app.go`), so `emod diagram -f svg` exits 1 with
      "flag provided but not defined: -f". Verified end to end with `--format`
- [x] `emod diagram <file>` with no `--specs`, in each of the four formats, writes byte-identical
      output to what the same invocation writes before this task — asserted as a differential against
      the model's `WithoutSpecs` twin, so it is checkable from inside the working tree
- [x] `emod diagram --specs --format mermaid <file>` and `emod diagram --specs --format ascii <file>`
      each write no output file, print nothing to stdout, and return an error whose exit code is 1 and
      whose message names `--specs` and lists `drawio` and `svg`; the leaves assert those tokens, not
      merely a non-zero code. (`-f` as written above is not a flag this command declares.)
- [x] `emod diagram --specs --serve <file>` returns the same shape of error and starts no server
- [x] The error is a `*LintError` carrying a cause callers can match with `errors.Is`, in the shape
      `internal/cli/lint.go:18` and `internal/cli/diagram.go:58-64` already use, and it registers no
      rule name and no entry in `internal/linter/descriptions.go`
- [x] A model with parse or validation errors still exits 2 and writes no file whether or not `--specs`
      is given, and a model with lint warnings still writes its diagram and exits 1 either way
- [x] `--specs` appears in `emod diagram --help` with a usage string saying what it draws
- [x] Every existing subtest in `internal/cli/diagram_test.go` passes with no edit — no assertion,
      expected value or fixture of one moves. Go has no default arguments, so all 21 existing
      `cli.RunDiagram(...)` calls gain a trailing `false`; the diff in that file is exactly those 21
      call lines plus the new leaves. The alternative, a variadic `diagram.Option` the CLI cannot
      inspect, would have to ignore `--specs` for mermaid and ASCII rather than refuse it, which is
      the trap "Open questions, decided" item 7 rejects

**Affected Files/Modules:**
- `internal/cli/app.go` — the `--specs` flag on the `diagram` command (`:113-149`), alongside
  `--format`, `--style`, `-o` and `--serve`, and the `Action` that reads it
- `internal/cli/diagram.go` — `RunDiagram` (`:29`) gains the parameter, refuses the surfaces that draw
  no card, and passes the option through to the two picture exporters at `:68-77`; the `--serve`
  branch in `app.go` refuses it before reaching `RunDiagramServe`
- `internal/cli/diagram_test.go` — leaves under the existing `TestDiagram` umbrella (`:23`) and its
  `serve` group (`:381`)

**Patterns to Follow:**
- The refusal's shape: the unsupported-format `*LintError` at `internal/cli/diagram.go:58-64`, its
  siblings at `internal/cli/slices_list.go:20-23` and `validate.go:16-19`, and the `errors.Is` leaf at
  `internal/cli/slices_list_test.go:297-303`
- Every `Action` is one line into `reportExitError`: `internal/cli/app.go:12` and the `diagram` action
  at `:136-148`; `tasks/learnings.md` "New file-taking CLI commands compose `parseModelFile` and
  `reportExitError`"
- A CLI leaf asserts the tokens that identify *this* diagnostic: `internal/cli/validate_test.go:253-258`;
  `tasks/learnings.md` "CLI diagnostic tests must assert the distinguishing message text"
- Flags are written **before** the file argument in every invocation this task exercises —
  `tasks/learnings.md` "urfave/cli v2 discards every flag written after the file argument", and
  "Open questions, decided" item 6 above. A leaf that spells `emod diagram <file> --specs` would pass
  against a flag the command never received

**Testable:** Yes — through `cli.RunDiagram` and the app's flag parsing, both exercised by
`internal/cli/diagram_test.go` today.

**Verification:** `go test -tags unit ./internal/cli/...`; `go run ./cmd/emod diagram --specs -o
<temp>.svg -f svg <a copy of a spec-stating example>` and read the card out of the written file;
`go run ./cmd/emod diagram --specs -f ascii <same file>` and read the refusal.

**Depends on:** Task 2

---

### Task 4: Document `--specs` where the diagram command is described

**Behavior:** The three places that describe what `emod diagram` can do mention the flag, so a reader
learns the option exists without reading `--help` or the source.

**Acceptance Criteria:**
- [ ] `README.md`'s "Generate diagrams" block (`:157-164`) gains a line showing `--specs` with the flag
      written before the file argument, and says which two formats draw the card
- [ ] `docs/dsl-reference.md`'s `### spec` section (`:376`) gains a bullet stating that
      `emod diagram --specs` draws a slice's specs as a Given-When-Then card under that slice in the
      draw.io and SVG outputs, and that the other formats do not — written in the shape §10's closing
      bullet (`:627`) uses for descriptions
- [ ] `docs/architecture.md`'s CLI table row for `emod diagram` (`:196`) names the flag alongside
      `--serve`
- [ ] No heading in `docs/dsl-reference.md` is added, renamed, renumbered or reordered, so no
      `#<n>-` link and no `#spec`-style sub-heading link changes: the `^## [0-9]+\.` list reconciles
      against the `\(#[0-9]+-` list, and the `^### ` list against the `\(#[a-z]` list, exactly as they
      do before this task
- [ ] No new ` ```emod ` fence is added by this task, and `internal/oracle`'s "documented models"
      subtest passes unchanged
- [ ] The `Diagram Palette` table in §13 is untouched, and `TestExporterPaletteMatchesReference` passes

**Affected Files/Modules:**
- `README.md` — the "Generate diagrams" bash block
- `docs/dsl-reference.md` — one bullet under `### spec`
- `docs/architecture.md` — one table row

**Patterns to Follow:**
- The sentence shape for "what the diagrams do with this construct": `docs/dsl-reference.md:627`
- `tasks/learnings.md` "`docs/dsl-reference.md` sub-heading anchors are cited more often than the
  numbered ones" — `#spec` is cited three times, so the heading text is held fixed
- `tasks/learnings.md` "An ```emod fence is a promise that the block validates" — `bash` fences and
  prose only
- `tasks/learnings.md` "`docs/dsl-reference.md` section 13 is machine-read" — §13 is not in this task's
  change set

**Testable:** No — prose across three documents. Correctness is that no anchor breaks, no fenced
`emod` block is added, and the two doc-reading Go tests still pass.

**Verification:** `go test -tags unit ./internal/oracle/... ./internal/diagram/...`; reconcile the two
heading/link lists in `docs/dsl-reference.md`; `git diff --stat` shows the three documents and nothing
else.

**Depends on:** Task 3

---

## Summary

**Total tasks:** 4

**Ordering rationale:** dependency-first, and deliberately shallow. Unlike the other stories in this
file, US-016 fans out across almost nothing: specs reach no picture today, so there is no formatter to
teach, no export key to add, no schema to widen, no grammar to loosen and no viewer surface to keep in
step — verified by searching `internal/diagram` and `internal/export/diagram.go`, neither of which
mentions a spec. What remains is one drawing, taught to two renderers, wired to one flag, and written
down.

Task 1 carries the design — the option, the card's lines, the card's geometry, the band — and pays for
it immediately in the format whose test harness is most fragile: `svgShapes` binds a `<text>` to the
last rect it saw, so a card is the largest instance yet of the hazard US-013's decomposition found, and
Task 1 closes it with a whole-list receipt rather than a note. Task 2 is then a second consumer of
helpers that already exist, which is why the cross-format equality of the card text is asserted there
and not in Task 1: it is the first point at which both cards exist to compare. Task 3 turns the option
into the flag the story asks for and decides, in code, what the three surfaces that draw no card do
about it. Task 4 makes the flag discoverable. Tasks 2, 3 and 4 are each strictly downstream of the one
before it; only Task 1 has dependencies outside this story, and they are hard ones — `ast.ThenView` and
`ast.ThenCommand` from US-007 Task 1, and the four-pattern fixture from US-007 Task 2.

**Coverage of the story's acceptance criteria:**

| Criterion | Task |
|---|---|
| A `--specs` flag renders each slice's specs as a Given-When-Then card under the slice | 1 (the card and the band, in SVG), 2 (in draw.io), 3 (the flag) |
| The flag is off by default; without it, diagram output is unchanged | 1 and 2 (each format's differential against the `WithoutSpecs` twin, plus "with the option, a spec-less model is byte-identical"), 3 (the CLI's four formats, and every existing `diagram_test.go` leaf passing unedited), 1 and 2 again (the contract subtest at `contract_test.go:441-453` passing with no edit) |
| The card shows given, when, and then — including rejected outcomes with the invariant name | 1 (one leaf per outcome kind, and the leaf requiring no invariant *statement* text to appear), 2 (the same content in draw.io, asserted as equality between the two formats' cards) |
| The card renders in both SVG and draw.io outputs | 1 (SVG), 2 (draw.io, plus the cross-format equality leaf and the contract-level positive sibling) |

Task 4 carries no story criterion: a flag no document mentions is a flag nobody finds, and the repo's
own convention is that `docs/dsl-reference.md` states what each diagram surface does with a construct.

Nothing from the story is deferred. What US-016 deliberately leaves to later stories, each named in
"Out of scope" above with the reason: rejection edges on the timeline and the invariant prose they
carry on hover (US-009); payload values on card lines (US-010); formatting (US-014); LSP (US-015);
highlighting (US-017); examples and the reference sweep for the spec constructs themselves (US-018);
and, per the story file's Non-Goals, expected view-state payloads on a `then view` outcome. Two
findings surfaced during decomposition that belong to no story yet and are recorded here rather than
absorbed: `emod diagram <file> --specs` will silently draw no card, because urfave/cli v2 discards
every flag written after the file argument — re-verified against `-f`/`-o` — and `README.md:160-163`
documents four invocations in exactly that broken shape; and `svgLaneLabels`
(`internal/diagram/svg_test.go:415`) infers "lane" from being the widest shape in the picture, which
the specs band satisfies, so it reports five lanes under `--specs` and four without.

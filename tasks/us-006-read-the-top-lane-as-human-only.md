# US-006: Read the top lane as human-only

## Progress
- [x] Task 1: Draw automations and translation reactors in the command and view lane
- [ ] Task 2: Label the top lane "Wireframes"
- [ ] Task 3: Stack automations and translations below the trigger row in the viewer
- [ ] Task 4: Emit the UI timeframe for a trigger and the processor timeframe for an automation

---

## Story Reference

`user-stories/triggers-and-automations.md` → **US-006: Read the top lane as human-only** (sixth of
eleven stories in "Triggers and Automations"). Design notes:
`docs/proposals/triggers-and-automations-proposal.md` — `:266-280` for the lane move and the viewer's
different fix, `:278` for why processors do not get a fourth lane, `:309-311` for the Mermaid
timeframes, `:462-466` for the phase-3 touch list.

**In scope:** the y at which SVG and draw.io draw an automation box and a translation reactor box,
moving from the top lane into the command and view lane without colliding with the commands and views
already there; the label on the top lane in both formats; the position of the viewer's combined
automation-and-translation row within a slice's vertical stack; the timeframe letter Mermaid emits for
a trigger and for an automation; and, as a property of all of the above, that the gear marking and
every edge touching a moved box survive the move intact.

**Out of scope:** the palette (US-007) — this story moves boxes and repaints nothing, and note the
viewer's automation fill (`#fff2cc`, `internal/viewer/static/renderer.js:253-261`) already disagrees
with SVG and draw.io's `fillReactor` (`#e1d5e7`, `internal/diagram/drawio.go:78`), which is US-007's to
settle, not this story's; the `automation/missing-todo-list` rule (US-008); LSP (US-009); syntax
highlighting (US-010); documentation and examples (US-011). Per the feature's stated non-goals: no
fourth lane for processors, and no change to the Translation pattern's wiring — only where its reactor
is drawn. ASCII has no lanes and no timeframes and is not edited. No AST, parser, formatter, validator,
exporter or importer file is touched: layout is a rendering concern in each output, and the diagram
document carries no coordinates (verified — no `x`/`y` key exists on `jsonDiagramNode`).

---

## Premises this breakdown was written against

**US-004 has landed.** The breakdown assumes `trigger "<name>" { ... }` parses with the quoted name
directly after the keyword, `trigger UI/Schedule/Processor "<name>"` is rejected, `ast.Trigger` carries
no `Kind`/`KindPos`, and the JSON and CUE exports carry no trigger kind. Whoever implements this should
re-check that before starting. Where it matters:

- **Task 4 cannot compile against today's tree.** `internal/diagram/mermaid.go:81`, `:175` and `:308`
  each read `s.Trigger.Kind`, so removing the field forces US-004 to touch all three. Task 4 is
  therefore written as *finish and pin*: if US-004 already collapsed the branch, the task's remaining
  work is the assertions, which do not exist today for any of the three Mermaid layouts; if US-004 left
  a trigger able to reach `pcr` by some other route, Task 4 removes it.
- **Tasks 1, 2 and 3 have no compile-level dependency on US-004.** `svg.go`, `drawio.go` and
  `internal/viewer/static/layout.js` never read `Trigger.Kind`. They would build and pass against `main`
  as it stands. The dependency is a semantic one — until `trigger Schedule` and `trigger Processor` are
  gone, the top lane legitimately holds machine-driven triggers and calling it human-only is a lie — so
  the tasks should still land after US-004, but a slip in US-004 does not block them mechanically.
- **Fixtures US-004 also rewrites.** `fullModel()` (`internal/diagram/contract_test.go:890`) states
  `Kind: "UI"` and `Kind: "Schedule"`, `describedModel()` (`:697`) and the per-format fixtures in
  `svg_test.go` and `drawio_test.go` state a kind, and `e2e-viewer/tests/helpers.js:13` writes
  `trigger UI "Checkout Form"`. Every one of those is code US-004 must migrate. Tasks 1 and 2 assert
  against `fullModel()`; expect its literal to already read the kindless form.

**US-005 is assumed NOT to have landed.** Nothing on `main` draws a `reads` edge from a view to an
automation — `internal/export/export.go` builds the `reads` edge for translations only, and neither
`svg.go` nor `drawio.go` draws one for an automation. The story's final criterion names "the view it
reads", so the criteria below are written over *every edge the exporter draws touching an automation
box*, rather than naming the reads edge. If US-005 has landed by the time this is implemented, the
view→automation edge joins that set with no re-decomposition needed — add it to Task 1's and Task 3's
edge inventory and nothing else moves. Both stories change edge code in `svg.go`, `drawio.go` and
`renderer.js`, so if they are in flight together, whichever lands second rebases onto the other.

---

## Consequences of that boundary, decided

Six things the story does not spell out.

1. **The naive move produces overlapping boxes, and the story is not done until it does not.** This is
   the substance of Task 1, so the arithmetic is worth stating. A lane is 190 tall with a 30 header
   (`laneHeight`, `internal/diagram/drawio.go:60`), so its content band runs from +30 to +190 relative
   to the lane top. Commands and views are centred there at +82 and are 55 tall, occupying +82 to +137
   and leaving 52 free above and 53 free below. An automation box is 41 tall and the stack steps 46 per
   row, anchored at the *bottom* of the lane and growing upward
   (`svg.go:150-164`, `drawio.go:404-416`). Rewriting `triggerLaneY` to `cmdViewLaneY` and changing
   nothing else therefore puts the first automation 3 pixels inside the command band and the second one
   wholly inside it. Horizontally they collide too: a lone command is 240 wide at `sliceX+10` and an
   automation is 210 wide at the same `sliceX+10`, so the columns coincide and vertical separation is
   the only lever. `fullModel()`'s "Create Order" slice already declares one automation *and* one
   translation, and the reactor stack continues the automation stack's index
   (`svg.go:171`, `drawio.go:424`), so two rows is the case that ships, not a contrived one.
2. **How the collision is resolved is the implementer's call; that it is resolved is the criterion.**
   Three levers exist — raise `laneHeight` (a constant in `drawio.go:60` that `svg.go:23` already
   folds into `diagramH`, so growing it needs no height plumbing and no expected constant moves,
   because every geometry assertion in `internal/diagram` compares two renderings of the same build
   rather than pinning absolute pixels), give the reactor stack its own x column away from the command
   row, or shrink the boxes. Task 1 states the outcome and names the levers; it does not pick one.
3. **The DCB and projected draw.io layouts share one lane, and keep it.** `cmdViewLaneY = triggerLaneY`
   at `drawio.go:151` and `:162`, so in those two modes the move is a no-op by construction — the boxes
   land on the same pixels — and the lane genuinely holds commands and views, which is why its label
   stays `"Triggers / Commands"` and does not become "Wireframes". Splitting them into two lanes to make
   the human-only property observable there would shift `eventLaneY`, every `tagLaneYs[i]` and
   `extLaneY`, and add a swimlane to both branches: a larger change than this story, and one the story's
   criteria do not ask for. What those modes *do* owe is the non-overlap property, since a fix to the
   shared band applies to them as well. `ExportSVG` takes its `Style` as `_` (`svg.go:12`) and has one
   layout only, so this concerns draw.io alone.
4. **"Keeps its gear marking" is a survival criterion, and the viewer has no gear to keep.** The gear
   lives in `internal/diagram/labels.go:6` and reaches SVG, draw.io, ASCII and Mermaid through
   `reactorLabel` and `automationLabel`. The viewer draws no gear anywhere — an automation is told apart
   by fill and by its label rows (`renderer.js:117-144`, `:253-261`). This story adds no gear to the
   viewer; giving the viewer a marking is a distinguishability question US-007 owns.
5. **`web/static/` is a gitignored build artefact and is never edited.** `/web` is in `.gitignore`,
   `git ls-files web/` is empty, and `Taskfile.yml:44-52` regenerates it with a one-way
   `cp -r ./internal/viewer/static/*`. The local copy is already several commits stale and nothing
   checks it. The proposal's instruction to change "its `web/static/` copy" (`:284`) does not apply:
   `internal/viewer/static` is the only source.
6. **The reorder must not touch the viewer's logical edge directions.** `EDGE_TYPE_BY_ENDS`
   (`internal/viewer/static/model.js:137-149`) is keyed on node type with no geometry in it, and its own
   comment records that its directions must match what the exporter writes because the importer reads
   them back. Moving a box down flips only which way an arrow is *drawn*; `event>automation` is still
   `automation_trigger`. A change to that table would silently corrupt the viewer's save path.

**Learnings folded in** from `tasks/learnings.md`: additive and positional output changes owe a
differential receipt rather than an assertion, and a differential must first prove its twin actually
differs; `require.NotEqual` on a stripped twin is satisfiable without stripping anything, so pair it
with a positive check on the content that must be gone and the content that must be found;
de-duplicate before a fan-out edit and land the de-duplication with proof, giving extracted helpers the
parameters their siblings take; an assertion whose expected value comes from the code under test cannot
fail; a second `require.Contains` on one string is often shadowed by the first; `internal/viewer/static`
is a display surface with its own vitest harness that `task test:unit` does not run, jsdom implements no
SVG geometry so `installSVGGeometry()` and the dynamic `await import` spelling are required, and
assertions on a drawn label must walk the group's `<text>` elements because `textContent` folds the
`<title>` in; the viewer shows a node twice, on the canvas and in the detail panel; run repo tooling
through `mise exec --`; acceptance criteria describe the working tree, and a commit-message receipt is
the commit author's obligation, never a criterion.

---

## Codebase Context

**SVG.** `ExportSVG` (`internal/diagram/svg.go:12`) has one layout. `diagramH` is
`2*marginY + 4*laneHeight + 3*laneGap` (`:23`); the four lane origins are computed at `:34-37` and their
rects and labels written at `:41-48`, with `"UI / Triggers"` at `:42` and `"Commands / Views"` at `:44`.
The four content-centre lines are `:56-59`. Per slice: the trigger at `:73-87` (the only thing drawn in
the top lane), commands at `:89-100` and views at `:102-112` sharing `midCenterY` and `itemLayout`,
events at `:114-147`, **automations at `:149-164`** and **translation reactors at `:166-181`** (whose
comment at `:166` names the lane it is leaving), external systems at `:183-198`. Arrows are
centre-to-centre straight paths through `svgArrowPath`: trigger→command `:214-226`, command→event
`:228-237`, event→view `:239-253`, **event→automation→command `:255-270`**, and the translation chain
`:272-300`.

**draw.io.** Layout constants are `:53-63`, with `laneHeight`/`laneGap` at `:60-61` and
`waypointMargin` at `:63`. Lane origins branch three ways at `:147-218` — DCB `:147-157`, projected
`:158-209`, standard `:210-218` — and the swimlane cells are written at `:225-265`, `"UI / Triggers"` at
`:254`. Centres are `:274-284`. **Automations `:404-416`** and **translation reactors `:418-430`** repeat
the SVG arithmetic; both go through `vertexCell(id, value, tooltip, x, y, w, h, style)` (`:787-801`)
with `styleReactor` (`:95`). Edge styles are `:451-454`. **event→automation `:546-555`** is the one edge
in the file that reads box coordinates — it routes through a column `waypointMargin` right of the event
and up to the automation's mid-y, so it follows the box automatically, but it shares that column with
event→view (`:529`). **automation→command `:557-560`**, external→reactor `:579-580` and reactor→command
`:582-588` are endpoint-only and auto-routed by mxGraph.

**Diagram tests.** `internal/diagram/contract_test.go` holds what all four formats share: the `exporter`
table (`:57-87`) with its `boxes` and `countConnections` hooks, `diagramBox{label, appearance}`
(`:40-45`), `boxLabelled` (`:141-153`), and the differential shapes to copy — `appearancesOf` and
`labelsExcept` (`:443-460`) compared across two renderings at `:408-422`, and the `withoutDescriptions`
/ `withoutInvariants` twins (`:797`, `:873`). `svgBoxes` (`svg_test.go:396`) reports each box's
attributes as one string; `drawioBoxes` (`drawio_test.go:834`) does the same from `drawioShapes`
(`:765`). `arrowCount`/`svgArrows` are `svg_test.go:322-336`. `singleSliceModel`
(`contract_test.go:645`) accepts commands, events, views, a trigger and automations — **its type switch
has no `*ast.Translation` case**, so a model needing a reactor either extends the helper or is built
literally. The lane labels are asserted at `svg_test.go:26`, `:35`, `:275` and `drawio_test.go:40`,
`:252`, `:497`; the shared DCB label at `drawio_test.go:355`, `:439`, `:536`. `svg_test.go:278` pins
`fullModel()` at 11 arrows and `svg_test.go:192` pins the automation chain at 2.

**Mermaid.** Three layout functions each emit one `tf NN <letter> <id>` line per element:
`exportMermaidStandard` (`:67`, trigger `:79-86`, automation `:110-113`, translation `:115-122`),
`exportMermaidDCB` (trigger `:173-181`, automation `:221-224`, translation `:227-232`) and the projected
layout (trigger `:306-314`, automation `:339-342`, translation `:346`). All three carry the same
`Kind == "Schedule" || Kind == "Processor"` branch sending a trigger to `pcr`, at `:81`, `:175` and
`:308`. Automations and translation reactors already emit `pcr` unconditionally everywhere.

**Viewer.** `layoutSlice` (`internal/viewer/static/layout.js:50-161`) is the only place slice-internal
y is assigned. It buckets children by type at `:51-56`, builds **`topRowTypes = translations.concat(automations)`
at `:58`**, sizes the slice from the widest row at `:60-73`, then walks a `blockY` cursor from
`slY + L.sliceTopPad` with a local `gap = 75` (`:75-76`) through five hardcoded blocks in source order:
**the combined automation-and-translation row `:78-90`**, triggers `:92-99`, commands `:100-107`, events
`:108-120`, views `:121-128`. There is no ordering table. `topRowTypes` and events lay their members out
side by side and bump `blockY` once; triggers, commands and views give each member its own row. The
slice box, its `minH` and the context height follow from the final `blockY` (`:130-157`, `:191-198`), so
a pure reorder leaves every height numerically identical. `computeArrowD` (`:284-313`) takes two rects
and already branches on `downward` (`:307-311`), so an arrow whose target is now above its source
attaches to the source's top and the target's bottom — the case `layout.test.js:175` covers with
hand-built rects and which this change makes real for `event>automation` and `automation>command`.
`renderer.js:300-353` draws edges from those positions; `appendBlockLabels` (`:117-144`) offsets every
label from `pos.y`, so labels follow the box.

**Viewer tests.** `internal/viewer/tests`, vitest on jsdom, run by `task test:viewer` and not by
`task test:unit`. `renderer.test.js` holds `drawnBoxes` (`:112-124`, a map of node id to x/y/width/
height/fill), `drawnLabels`/`drawnText` (`:95-101`), `tooltipOf` (`:103`) and `render` (`:75-79`);
`:219` is the closest thing to a layout guard today, comparing whole box geometries between two renders
(`:233`). `drag-containment.test.js:101` asserts a slice sizes to `minW`/`minH` and contains its blocks,
and `:165-171` is the one absolute-y assertion in the suite (a drag clamp, over a fixture with no
automations). `svg-env.js` provides `installSVGGeometry()`. **No test asserts the vertical order of
automations against the trigger row** — the behaviour this story changes is currently uncovered.

**Not touched, deliberately.** `internal/ast`, `internal/parser`, `internal/formatter`,
`internal/validator`, `internal/export`, `internal/importer`, `internal/cue`, `internal/linter`,
`internal/glossary`, `internal/lsp`, `internal/cli`, `internal/diagram/ascii.go`, `editors/`, `docs/`,
`README.md`, `examples/`, `web/`, `e2e/` and `e2e-viewer/`.

---

## Tasks

### Task 1: Draw automations and translation reactors in the command and view lane

**Behavior:** SVG and draw.io draw every automation box and every translation reactor box inside the
command and view lane, beside the commands and views they wire to, instead of inside the top lane. Each
box lands wholly within that lane and overlaps no other box, including when a slice declares several of
them. Each keeps the gear marking it already carried, and every edge that touched one still connects the
same two boxes. Nothing but a trigger is drawn in the top lane.

**Acceptance Criteria:**
- [ ] In both formats, for a model whose slice declares a trigger, two commands, a view, an event, two
      automations and a translation, every automation box and every translation reactor box lies wholly
      within the command and view lane's rect — its top edge at or below the lane's content band and its
      bottom edge at or above the lane's bottom
- [ ] In the same rendering, no box overlaps any other box: comparing every pair of drawn boxes by their
      x, y, width and height finds no intersection. This is the criterion the naive move fails — with the
      padding as written the first reactor row sits 3 pixels inside the command band and the second sits
      wholly inside it
- [ ] In the same rendering, no box drawn inside the top lane's rect is anything other than the trigger,
      asserted by listing the labels of the boxes whose y falls in that lane and requiring the trigger's
      to be the only one
- [ ] Every automation box's label and every translation reactor box's label still contains the gear
      marking, and the automation labels of a model with a schedule still carry the clock marking and the
      expression beside it — the existing subtests at `internal/diagram/svg_test.go:141`,
      `drawio_test.go:152` and `contract_test.go:370` pass, the last one unedited
- [ ] For each automation in the rendering, an edge connects the box of the event that activates it to
      that automation's box, and another connects that automation's box to the box of the command it
      issues. In SVG each such arrow's two endpoints are distinct and coincide with the centres of the two
      boxes it joins; in draw.io each such edge's source and target resolve to those two cells' ids. The
      translation chain — external system to reactor, reactor to command — is asserted the same way
- [ ] `fullModel()` still renders 11 arrows in SVG (`internal/diagram/svg_test.go:278`) and the automation
      chain still renders 2 (`:192`), both unedited: the move adds and removes no edge
- [ ] Rendering a model with no translations, and the same model with its automations removed, produces
      identical appearances for every remaining box in both formats — with the featured rendering first
      required to draw the automation boxes and the stripped one required to draw none, so the comparison
      is not two identical pictures agreeing. The twin strips automations only: removing a translation
      also removes its event and external-system boxes and reflows the event lane, and the reactor stack's
      padding counts the automations ahead of it, so no twin proves both at once
- [ ] Under `StyleDCB` and `StyleProjected` over a DCB model, draw.io draws every automation and reactor
      box without overlapping any command or view box in the shared lane. Those two layouts set
      `cmdViewLaneY = triggerLaneY` (`internal/diagram/drawio.go:151`, `:162`), so the lane move itself is
      a no-op there and their lane count is unchanged — what must hold is the non-overlap
- [ ] Both formats' output is still well formed: `requireValidXML` passes for each model above

**Affected Files/Modules:**
- `internal/diagram/svg.go` — the automation block (`:149-164`), the reactor block (`:166-181`) and its
  stale lane comment (`:166`); `diagramH` (`:23`) and the lane origins (`:34-37`) if the vertical budget
  is solved by growing the lane
- `internal/diagram/drawio.go` — the automation block (`:404-416`), the reactor block (`:418-430`) and its
  stale comment (`:418`); `laneHeight` (`:60`) on the same condition
- `internal/diagram/contract_test.go` — the shared receipt: a box-overlap check and a lane-containment
  check reachable from the `exporter` table (`:57-87`), and the twin
- `internal/diagram/svg_test.go`, `internal/diagram/drawio_test.go` — the per-format leaves

**Patterns to Follow:**
- The differential to copy: `internal/diagram/contract_test.go:408-422`, which renders one model twice
  and compares `appearancesOf` against `labelsExcept` (`:443-460`), after first requiring the featured
  twin to show what must be there
- The twin builders to copy: `withoutDescriptions` (`:797-813`) and `withoutInvariants` (`:873`)
- `tasks/learnings.md` "A differential receipt must first prove the twin actually differs" and
  "`require.NotEqual` on a stripped twin is satisfiable without stripping anything"
- `tasks/learnings.md` "De-duplicate before a fan-out edit, and land the de-duplication with proof" — the
  automation and reactor placement is the same arithmetic four times across two files, and
  `sliceXPositions`/`contextBounds`/`layoutWidth` are the precedent for a shared layout helper in this
  package; if one is extracted, carry the differential receipt and give it the parameters its callers
  take
- Box geometry is read through `svgBoxes` (`internal/diagram/svg_test.go:396`) and `drawioBoxes`
  (`drawio_test.go:834`), both of which report appearance as one string — a numeric comparison needs the
  x/y/width/height read out of it
- The edge inventories to walk: `internal/diagram/svg.go:255-300` and
  `internal/diagram/drawio.go:546-599`. `waypointMargin` (`drawio.go:63`) is shared by the event→view and
  event→automation routes (`:529`, `:547`)
- `singleSliceModel` (`internal/diagram/contract_test.go:645`) has no `*ast.Translation` case; the
  reactor-carrying model needs one added or a literal model
- Palette and the `reads` edge are US-007's and US-005's; this task repaints nothing and adds no edge

**Testable:** Yes — through `diagram.ExportSVG` and `diagram.ExportDrawio`.

**Verification:** `mise exec -- go test -tags unit ./internal/diagram/...`; `mise exec -- go build ./...`.

**Depends on:** None

---

### Task 2: Label the top lane "Wireframes"

**Behavior:** the lane that now holds only human entry points is named for what it holds. SVG and
draw.io write "Wireframes" where they wrote "UI / Triggers"; every other lane label is untouched.

**Acceptance Criteria:**
- [ ] The SVG for a model renders a lane label reading `Wireframes`, and the string `UI / Triggers`
      appears nowhere in the output
- [ ] The draw.io XML for the same model carries a swimlane cell whose value is `Wireframes`, the string
      `UI / Triggers` appears nowhere in the output, and the document still parses as well-formed XML
- [ ] `Commands / Views`, `Events` and `External Systems` are unchanged in both formats, asserted in the
      same subtest as the rename so a sweep that renamed too much fails
- [ ] Under `StyleDCB` and `StyleProjected` over a DCB model, draw.io's shared top lane still reads
      `Triggers / Commands` — that lane holds commands and views as well as triggers
      (`internal/diagram/drawio.go:151`, `:162`), so it is not the human-only lane and is not renamed
- [ ] The empty-model leaf at `internal/diagram/svg_test.go:26` still asserts that a model with no
      contexts draws no lane at all, now naming the new label
- [ ] All six existing assertion sites move to the new label: `internal/diagram/svg_test.go:26`, `:35`,
      `:275` and `internal/diagram/drawio_test.go:40`, `:252`, `:497`; `git diff` touches no other
      expected string in either file

**Affected Files/Modules:**
- `internal/diagram/svg.go` — the lane label at `:42`
- `internal/diagram/drawio.go` — the swimlane cell at `:254`
- `internal/diagram/svg_test.go`, `internal/diagram/drawio_test.go` — the six sites

**Patterns to Follow:**
- `swimlaneCell` (`internal/diagram/drawio.go:776-782`) writes its value unescaped; `svgLaneLabel`
  (`internal/diagram/svg.go:363`) is the SVG counterpart
- The three sites asserting the DCB label — `internal/diagram/drawio_test.go:355`, `:439`, `:536` — are
  the ones that must *not* move, and the subtest name at `:414` names that lane
- `tasks/learnings.md` "A second `require.Contains` on one message is often shadowed by the first" — the
  unchanged labels asserted alongside the new one must not be substrings of it
- `docs/dsl-reference.md` and `README.md` name no lane label (verified); documentation is US-011's

**Testable:** Yes — through `diagram.ExportSVG` and `diagram.ExportDrawio`.

**Verification:** `mise exec -- go test -tags unit ./internal/diagram/...`.

**Depends on:** 1

---

### Task 3: Stack automations and translations below the trigger row in the viewer

**Behavior:** within a slice, the viewer lays the combined automation-and-translation row out below the
trigger rows instead of above them, so the processor no longer sits higher on the canvas than the person
it serves. The row keeps its side-by-side shape and its widths; the slice, aggregate and context boxes
keep the sizes they had; and the edges into and out of a moved node are still drawn, now running upward.

**Acceptance Criteria:**
- [ ] For a slice declaring a trigger, an automation, a translation, a command, an event and a view, the
      automation's and the translation's y is greater than the trigger's y and less than the command's,
      read from the positions `Layout.computeLayout` returns
- [ ] The automation and the translation remain one row: they share a y and sit side by side with the
      translation first, with the same widths and x positions the row had, so only the row's place in the
      stack changed
- [ ] Every child box of the slice lies wholly inside the slice box, and the slice's height and width
      equal its `minH` and `minW` when nothing has been dragged — the shape asserted at
      `internal/viewer/tests/drag-containment.test.js:101`
- [ ] No two boxes within the slice overlap
- [ ] The arrow from the activating event to the automation and the arrow from the automation to the
      command it issues are both drawn: each has a non-empty path, distinct endpoints, and attaches to the
      source box's top edge and the target box's bottom edge, which is the upward branch at
      `internal/viewer/static/layout.js:307-311`. The count of drawn arrows equals the number of edges in
      the model, so the reorder drops none
- [ ] A slice that declares no automation and no translation lays out with the same row sequence it had —
      trigger, then command, then events, then views — so the change is inert for models not using either
- [ ] `autoDetectEdgeType` still answers `automation_trigger` for an event-to-automation pair,
      `automation_command` for an automation-to-command pair and `reads` for a view-to-translation pair;
      `git diff` shows no change to `EDGE_TYPE_BY_ENDS` (`internal/viewer/static/model.js:137-149`), whose
      directions the importer reads back on the viewer's save path
- [ ] An automation node's drawn labels and, where it states a schedule, its clock badge and tooltip still
      sit within that node's box — the existing leaf at `internal/viewer/tests/renderer.test.js:219`
      passes, and the box-geometry comparison at `:233` still holds
- [ ] `mise exec -- task test:viewer` passes, and `git status --porcelain -- '*.go'` is empty: this task
      changes no Go file
- [ ] `git check-ignore web/static` succeeds and `git ls-files web/` lists nothing, confirming the copy
      under `web/` is a build artefact regenerated by `task build:web` from `internal/viewer/static` — so
      no file under `web/` is edited by this task, and the proposal's instruction to change "its
      `web/static/` copy" (`docs/proposals/triggers-and-automations-proposal.md:284`) does not apply

**Affected Files/Modules:**
- `internal/viewer/static/layout.js` — the row block at `:78-90` within `layoutSlice` (`:50-161`)
- `internal/viewer/tests/` — a leaf asserting the slice's row order, beside the geometry leaves in
  `renderer.test.js` and the containment leaves in `drag-containment.test.js`

**Patterns to Follow:**
- The five row blocks in `layoutSlice` are a hardcoded sequence sharing one `blockY` cursor
  (`internal/viewer/static/layout.js:78-128`), each advancing it by `L.boxHeight + gap`; the slice box and
  its `minH` derive from the final cursor (`:130-157`), so the bookkeeping stays self-consistent under a
  reorder and no height needs recomputing
- `docs/proposals/triggers-and-automations-proposal.md:274` states the intended landing place: after the
  trigger row rather than before it
- The geometry helpers to use: `drawnBoxes` (`internal/viewer/tests/renderer.test.js:112-124`) and the
  positions `Layout.computeLayout` returns; `installSVGGeometry()` from `./svg-env.js` at module scope and
  the dynamic `const { … } = await import(…)` spelling, as `renderer.test.js:4-7` does
- `tasks/learnings.md` "`internal/viewer/static` is a display surface with its own vitest harness", whose
  note that assertions on a drawn label must walk the group's `<text>` elements applies to any label
  assertion added here
- `tasks/learnings.md` "The viewer's save path is `importer.ImportDiagram`" — positions never reach that
  path, which is why the edge-direction table must stay put
- The viewer draws no gear and this task adds none; its automation fill and the trigger's framing are
  US-007's

**Testable:** Yes — through the vitest harness under `internal/viewer/tests`.

**Verification:** `mise exec -- task test:viewer`; `git status --porcelain -- '*.go'` is empty.

**Depends on:** None

---

### Task 4: Emit the UI timeframe for a trigger and the processor timeframe for an automation

**Behavior:** Mermaid has no lanes, so a timeframe letter is where an element's kind is expressed. Every
trigger emits the UI timeframe and every automation emits the processor timeframe, in all three of
Mermaid's layouts, with nothing about a trigger able to send it to the processor slot.

**Acceptance Criteria:**
- [ ] For a model rendered under the standard layout, the DCB layout and the projected layout, every
      trigger's `tf` line carries the `ui` timeframe and every automation's carries `pcr` — asserted for
      all three layouts, which no subtest covers today
- [ ] A trigger whose name reads like a schedule (a name such as `NightlySweep`) still emits `ui`, so the
      timeframe is proved to come from the element type and not from the text of its name
- [ ] `internal/diagram/mermaid.go` contains no branch that can send a trigger to `pcr`: the three sites
      that read a trigger kind (`:81`, `:175`, `:308`) are gone. US-004 removes `Trigger.Kind` from the
      AST, so those sites cannot compile against it — if US-004 already deleted them, this criterion is a
      read of the file rather than an edit
- [ ] A translation reactor still emits `pcr` in all three layouts, unchanged: this story moves where a
      reactor is drawn and does not change what Mermaid calls it
- [ ] The `tf` sequence numbers for a model are unbroken and consecutive, so no element lost or gained a
      line
- [ ] `git diff` moves no expected Mermaid line for an element other than a trigger in
      `internal/diagram/mermaid_test.go`

**Affected Files/Modules:**
- `internal/diagram/mermaid.go` — the trigger emission in `exportMermaidStandard` (`:79-86`),
  `exportMermaidDCB` (`:173-181`) and the projected layout (`:306-314`)
- `internal/diagram/mermaid_test.go` — the three-layout assertions

**Patterns to Follow:**
- The automation and translation emissions already write `pcr` unconditionally
  (`internal/diagram/mermaid.go:111`, `:117`, `:222`, `:229`, `:340`, `:348`) and are the shape a
  collapsed trigger branch takes
- `docs/proposals/triggers-and-automations-proposal.md:309-311` states the intent: with the kinds gone
  the branch collapses and triggers always emit `ui`
- `tasks/learnings.md` "An assertion whose expected value comes from the code under test is the recurring
  review finding" — name the expected `tf` line, do not rebuild it from the renderer
- `tasks/learnings.md` "Assert a short keyword in a diagnostic with a `\b`-bounded `require.Regexp`" — the
  same hazard applies to a two- and three-letter timeframe letter, since `ui` and `pcr` occur inside
  element and context names
- Mermaid renders no boxes, so `contract_test.go`'s `exporter` entry for it carries no `boxes` hook
  (`:76-79`) and the geometry receipts of Task 1 skip it

**Testable:** Yes — through `diagram.ExportMermaid`.

**Verification:** `mise exec -- go test -tags unit ./internal/diagram/...`.

**Depends on:** None

---

## Summary

**Four tasks**, ordered by where the story's weight sits rather than by a dependency chain — only Task 2
depends on another, and it depends on Task 1 both because a lane cannot honestly be called "Wireframes"
while it still holds machines and because the two write the same four files. Tasks 3 and 4 are
independent of everything: the viewer is JavaScript with its own harness and its own layout engine, and
Mermaid has no geometry at all. They can run in parallel with Tasks 1 and 2.

Task 1 is the story. It is the only task where the obvious edit — swapping one lane origin for another —
produces a worse diagram than the one it replaces, because the destination lane is already occupied at
the y the boxes would land on. Its criteria are therefore written as geometric properties of the whole
rendering (containment, non-overlap, who is in the top lane) rather than as an assertion that a box moved,
and its edge criteria resolve endpoints to the boxes they join rather than counting arrows.

**Story criteria coverage:**

| Story criterion | Task |
|---|---|
| In SVG and draw.io, automations and translation reactors render in the command and view lane | 1 |
| In the viewer, automations and translations render below the trigger row | 3 |
| The top lane is labelled "Wireframes" in SVG and draw.io | 2 |
| Automations emit the processor timeframe in Mermaid, and triggers the UI timeframe | 4 |
| Every automation and translation reactor keeps its gear marking | 1 (SVG and draw.io hold the marking; the viewer has none to keep) |
| An automation's edges remain drawn and legible after the move | 1 (SVG and draw.io), 3 (the viewer) |

Carried along, not stated by the story: the non-overlap and lane-containment properties, without which
the first criterion is satisfiable by a diagram nobody can read; the DCB and projected draw.io layouts,
whose shared lane makes the move a no-op but still owes non-overlap; and the guard that the viewer's
logical edge directions did not move with its boxes.

**Deferred to later stories in the feature:** the `reads` edge from a view to a trigger or automation
(US-005 — if it lands first, its edge joins Task 1's and Task 3's edge inventories); the palette,
including the viewer's disagreeing automation fill and the trigger's screen framing (US-007); the
`automation/missing-todo-list` rule (US-008); LSP (US-009); highlighting (US-010); and every document and
example (US-011).

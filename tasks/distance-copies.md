# Distance copies and the SVG arrow router

## Contents
1. Task 1: Copy far events by slice distance in the SVG and the diagram JSON
2. Task 2: Copy far views and commands in the SVG
3. Task 3: Draw copies of views and commands in the viewer
4. Task 4: Write view and command copies into the diagram JSON and read them back on save
5. Task 5: Start and end every SVG arrow on the side of the boxes it joins
6. Task 6: Give each SVG arrow a horizontal leg of its own between lanes
7. Task 7: Route every SVG arrow clear of the boxes it does not join

## Story Reference

Inline story from the conversation: replace long arrows in the lane diagram with copies, by distance, in the SVG and the viewer, and fix the SVG router.

1. **Copy rule by distance**, replacing the count rule in `internal/diagram/event_copy.go` (`copyThreshold = 3`). When an arrow joins two slices that are 2 or more slices apart in layout order, the slice at the far end draws a copy of the element and the arrow starts or ends at the copy. It applies to: event -> view (subscribes), event -> automation (on), view -> trigger/automation/translation (reads), automation/translation -> command (command). A copy of an event or view says "from <slice>", a copy of a command says "in <slice>". A slice draws one copy per element. Arrows inside a slice or between two slices next to each other stay arrows.
2. **One rule, every surface.** The SVG (`internal/diagram/svg.go`, `layout.go`) and the viewer (diagram JSON in `internal/export/diagram.go`, `internal/importer`, which must ignore copies on the way back, and the JS viewer in `internal/frontend/static`) all use the same rule, decided once in `internal/diagram`.
3. **SVG router fix.** `svgArrowPath` routes each arrow centre-to-centre with one horizontal leg at the midpoint of the two centres, so all arrows between the same two rows lie on one line, and arrows go across other boxes. Give each arrow its own horizontal y in the space between lanes, and start and end arrows at box edges, so that arrows no longer share a line or go across a box.

A prototype of part 1, SVG only, is in the worktree `/private/tmp/claude-501/-Users-davidnguyen-Personal-Code-emod/b7811524-de39-4605-bd6a-cce7eac6e50e/scratchpad/wt` (branch `prototype/distance-copies`, uncommitted: `internal/diagram/element_copy.go` plus edits to `layout.go` and `svg.go`). On a generated 40-slice model it took crossings from 376 to 2, shared line length from 272605 to 3990 px, arrows through a box from 105 to 10, and copies from 25 to 56, for 3% more width. The generator and the measure script are `/private/tmp/claude-501/-Users-davidnguyen-Personal-Code-emod/b7811524-de39-4605-bd6a-cce7eac6e50e/scratchpad/gen/gen_model.py` and `/private/tmp/claude-501/-Users-davidnguyen-Personal-Code-emod/b7811524-de39-4605-bd6a-cce7eac6e50e/scratchpad/gen/measure.py`.

## Boundaries

**Out of scope:**
- The viewer's own arrow routing. The viewer draws its arrows as straight lines and Bézier curves (`arrowCurve` in `internal/frontend/static/layout.js`); part 3 fixes the SVG router only.
- Mermaid and ASCII. Mermaid writes a view's subscriptions as comments, and ASCII lists each slice's relations by name, so neither draws an arrow across the picture, neither draws copies, and their output does not change. Their unchanged bytes are the receipt that the copy rule stayed in the pictures that draw arrows.
- The event-flow diagram (`--format event-flow`), which has its own layout and its own arrows.
- Arrows outside the story's list: flow arrows, trigger -> command, external system -> translation, and the dashed arrow to a rejection badge. These arrows draw from the original element. The one exception is the command -> event arrow a translation implies (`EdgeTranslationFlow`): when the translation's slice draws a copy of its command, that arrow starts at the copy, because the event sits in the translation's slice and an arrow from the original command is the long arrow this story removes.
- Copies of triggers, automations, translations and external systems.
- A setting for the distance. It is a constant, 2. Nothing configures it, and the prototype's `EMOD_COPY_DISTANCE` variable is not carried over.
- Fewer arrow crossings as a goal of its own. The router gives each arrow its own horizontal leg and keeps every arrow off the boxes it does not join; crossings that remain after that are not a criterion.

**Deferred:**
- The viewer's highlight walk in `user-stories/model-queries.md` and `docs/proposals/model-queries-proposal.md` (point 4) says that an event copy stands for its original. With copies of views and commands, that walk must treat every copy as its original. Those stories own the walk, so they own that change.

## Codebase Context

**The copy rule today.** `EventCopies` (`internal/diagram/event_copy.go:28-50`) copies every event that a view subscribes to from another slice, once that view subscribes to more than three such events. It keys the copies by slice, in `AllSlices` order, and `declaringSlices` (`:69-80`) resolves each event to the first slice that declares it, translation-nested events included. `query.Declarations` resolves the same names, but `internal/query` imports `internal/diagram` (`trace.go`), so `internal/diagram` cannot use it.

**Layout order.** `collectSlices` (`internal/diagram/layout.go:151-172`) draws slices left to right in `ast.Model.SliceRefs()` order: contexts in order, and each context's slices sorted by source position. `AllSlices` is the same order. The viewer lays each context out as its own row (`layout.js` `computeLayout`), so it does not place slices in that order on screen. It draws the copies the diagram JSON gives it, which keeps the decision in Go.

**The SVG.** `ExportSVG` (`internal/diagram/svg.go`) places every box per slice, then draws every connection from `SliceEdges` (`graph.go:58-129`). Event copies sit in the event row (`svg.go:154-162`), and `columnWidth` (`layout.go:188-205`) widens a slice for them. The subscription arrow starts at the slice's own copy when there is one (`svg.go:246-255`). Every arrow is written after every box and runs from box centre to box centre (`svgArrowPath`, `svg.go:457-472`), so today a line crosses the labels of both boxes it joins, and its arrowhead sits on the target's label.

**The prototype.** It adds `ElementCopy{Kind, Name, DeclaredIn}`, `CopyKind` and `ElementCopies`. View and command copies go in the Commands / Views row, keyed per slice by kind and name, and `boxFor` resolves an arrow end to the slice's copy or to the original. It reads `EMOD_COPY_DISTANCE`, and it keeps `EventCopies` for the diagram JSON, so in the prototype the SVG and the viewer use two different rules.

**Prototype test failures** (`go test -tags unit ./internal/diagram/` in the prototype worktree):
- `TestExporterEventCopies` pins the count rule.
- `TestExporterAutomationDelay` finds an activation arrow by its source event's name, and that arrow now starts at a copy whose label is "<event>\nfrom <slice>".
- `TestExporterReadsEdges` matches a reads arrow's source label to a view name exactly. It also compares the boxes of a model with and without reads, and clearing a far read now also removes that read's view copy.
- The unexamined subtest, `TestExportSVG/spec_cards/draws_a_card_under_each_slice…` (`svg_test.go:518`), fails because `boxLabelled` matches by substring (`contract_test.go:317`). It now finds two boxes for `DeskOccupancyView`: the view and its copy "DeskOccupancyView / from Desk Occupancy". The lookup in the test is at fault, not where the card is drawn. `boxNamed` (`contract_test.go:2069`) already matches whole lines.

**The diagram JSON.** `diagramBuilder` (`internal/export/diagram.go:144-299`) writes an `event_copy` node per copy, labelled with the event. `pointCopiesAtOriginals` then sets `copy_of` to the first node drawn for that event's name. `appendEdges` (`:457-505`) starts a subscription edge at the slice's copy, and every other edge at the originals. The format is a contract outside this repository: `emod export --format diagram-json` (`internal/cli/export.go`) and the MCP `emod_export` tool (`internal/mcp/server.go:336`) both emit it.

**The importer.** The nodes and edges carry plain strings, so the importer reads by type and by label:
- `buildSlice` (`importer.go:254-266`) reads children by type, so it never reads an `event_copy` node as a declaration.
- `foldEdges` takes `src.Label`, which for a copy is the event's name.
- The viewer's save path is `ApplyDiagram` (`diagram_edit.go`), and it knows about copies in three places. `matchDrawnNodes` (`:243-247`) skips a copy's missing position. `nameCopiesAfterOriginals` (`:76-88`) gives a copy the name its original has now, so a rename travels along a kept edge from the copy. `inResult` (`:534-548`) keeps a copy only while its original stands.
- Each of these checks `isEventCopy` (`importer.go:46-50`). A `view_copy` or `command_copy` node reaches no matching arm, so `matchDrawnNodes` returns "carries no position" for it.

**The viewer.** The viewer handles `event_copy` in these places:
- `model.js:223-235`: `isEventCopy`, `sliceDeclaring`, and the move refusal at `:156-170`.
- `layout.js`: `copyFromLine` (`:69-73`), the copy widths and the copy row (`:100-118`, `:155-161`), and `getSliceChildNodeIds` (`:314-322`).
- `renderer.js`: `buildCopyBlock` (`:537-558`), with the event's stroke hard-coded, dispatched at `:645`.
- `ui.js`: the tooltip (`:83-99`), the mark on the original (`:99-106`), the click that shows the original (`:126-131`, `:1097`), and the refused context menu (`:1195`).
- `interaction.js:368`: a connection or repoint released over a copy is refused.
- `config.js:55-57`: `eventCopyStyle`.
- `viewer.html:1292-1299`: the `.event-copy-block` and `.copy-original` rules.

The tests that pin this behaviour are `internal/frontend/tests/event-copy.test.js` and `viewer.test.js:2225-2350`. In a browser, `e2e-viewer/tests/copies.spec.js` renders a view that subscribes to four events in the four slices before it. Under the distance rule that view draws three copies, and `event_copy-4` no longer exists.

**Test readers for the SVG.**
- `svgConnections` (`svg_test.go:1278-1314`) names each arrow by the box whose centre is at each end.
- `dashedArrowEndpoints` (`:941-965`) and the rejection-badge tests (`:418-470`) compare arrow ends with badge centres.
- The comment on `svgDashedArrowBetween` (`svg.go:440-445`) states that contract.
- `svgShapes` (`:1066`) returns every box and gear in document order.
- The event-flow SVG starts an arrow on a node's bottom side, below its captions (`placedNode.bottom`, `eventflow_svg.go:101-109`), and ends it on the top side (`eventFlowArrow`, `:166-185`). It still puts its leg at the midpoint.

**Words already taken.**
- `diagram.Edge` is an arrow, so a box's edge is a *side*.
- `release.Channel` owns "channel", and a lane is a swimlane.
- `internal/query` owns "reference" (`emod references`), and its `closest.go` uses "distance" for the edit distance between names.
- The SVG source already calls an arrow's horizontal stretch its *leg* (`svgArrowLabel`'s comment).

**Docs.** `docs/dsl-reference.md:357` states the count rule, for the SVG only. `:860` describes how an event copy is painted. `README.md` and `site/content/docs` say nothing about copies.

## Tasks

### Task 1: Copy far events by slice distance in the SVG and the diagram JSON

**Behavior:** An event drawn in the slice at the far end of an arrow that joins two slices 2 or more apart in layout order, for an event's arrow into a view (subscribes) or into an automation (on). The SVG draws the copy, and the diagram JSON writes it as an `event_copy` node, from one rule in `internal/diagram`. The count rule is gone.

**Acceptance Criteria:**
- [x] An event's arrow into a view or into an automation that joins two slices 2 or more apart in layout order starts at a copy of the event. The copy is drawn in the slice of the view or the automation. This holds in the SVG and in the diagram JSON, where the `subscription` or `automation_trigger` edge starts at the `event_copy` node in that slice.
- [x] The copy shows the event's name and, under it, "from <slice>", where <slice> is the first slice that declares the event, translation-nested events included.
- [x] An event's arrow into a view or an automation in the same slice, or in the slice next to it in layout order, starts at the event itself. Two slices next to each other across a context boundary count as next to each other.
- [x] A slice draws one copy of an event, however many of its views and automations that event feeds. A slice that declares the event draws no copy of it, and an event that no slice declares draws no arrow and no copy.
- [x] The rule no longer counts subscriptions. In the hotel stays fixture (`test.SubscribedEventsHotelStaysModel`):
  - "See Stay History" copies RoomReserved, GuestCheckedIn and BreakfastOrdered, and its arrow from GuestCheckedOut (Check Out, the slice next to it) starts at the event.
  - "See Occupancy", whose view subscribes to only three events, copies all three, because each is 2 or more slices away.
  - "Plan Cleaning" draws its arrow from RoomCleaned (Clean Room, next to it) from the event.
- [x] An arrow from a copy into a delayed automation still carries that automation's delay.
- [x] The distance is a constant of 2 slices. No environment variable or flag changes it.
- [x] On the viewer's save path, a diagram that starts an activation arrow at an event copy behaves as one that starts it at the event:
  - Applied unedited, it leaves the model as written.
  - Renaming the event renames the `on` of the automation it activates through the copy.
  - Deleting the arrow from the copy clears that automation's `on`.
- [x] Mermaid and ASCII output is unchanged: their existing tests pass without edits.
- [x] `e2e-viewer/tests/copies.spec.js` states the copies the distance rule gives for its fixture.
- [x] `docs/dsl-reference.md` describes the distance rule, for subscriptions and activations, in the SVG and the viewer. It no longer describes a count.

**Affected Files/Modules:**
- `internal/diagram/event_copy.go`: replaced by the distance rule, as the `ElementCopies` type of the planned design, holding event copies (file named for the type).
- `internal/diagram/event_copy_test.go`: rewritten against the distance rule, in the test file of the new type.
- `internal/diagram/layout.go`: `collectSlices` and `sliceEntry.copies` take the new rule.
- `internal/diagram/svg.go`: event copies are filed per slice by element, and the subscription and activation arrows start at the slice's copy.
- `internal/diagram/contract_test.go`: `TestExporterEventCopies` states the distance rule. `TestExporterAutomationDelay` reads an activation arrow from a copy as one from its event.
- `internal/export/diagram.go`: the builder takes the new rule, and `automation_trigger` edges start at the slice's copy.
- `internal/export/export_test.go`: the copy and subscription-source expectations.
- `internal/importer/diagram_edit_test.go`: activation through a copy on the save path.
- `e2e-viewer/tests/copies.spec.js`: the copy count and the copy ids for its fixture.
- `docs/dsl-reference.md`: the paragraph at line 357, and the note at line 860 if its wording names the count.

**Patterns to Follow:**
- Follow the copy rule's shape in `internal/diagram/event_copy.go:24-100`, and its tests in `internal/diagram/event_copy_test.go:14-149`.
- Follow how the SVG files and draws copies in `internal/diagram/svg.go:154-162` and `internal/diagram/svg.go:246-255`.
- Follow how the JSON writes copies and starts edges at them in `internal/export/diagram.go:272-299` and `internal/export/diagram.go:476-481`.
- Follow the contract tests in `internal/diagram/contract_test.go:1942-2096`.
- Follow the JSON copy assertions in `internal/export/export_test.go:3792-3829` and `internal/export/export_test.go:5922-5948`.
- Follow the save-path copy tests in `internal/importer/diagram_edit_test.go:2090-2160`.
- Prototype: `/private/tmp/claude-501/-Users-davidnguyen-Personal-Code-emod/b7811524-de39-4605-bd6a-cce7eac6e50e/scratchpad/wt/internal/diagram/element_copy.go`. Leave out its environment variable.
- Learning: "An arrow between two constructs is drawn by five surfaces, and none of them reads another".
- Learning: "The viewer's save path is `importer.ImportDiagram`, so a diagram-node field owes a read-back".
- Learning: "An importer fold must re-check the endpoint node types, not only the edge type".
- Learning: "Read an SVG arrow back as the two boxes it meets".
- Learning: "A per-slice shape needs its column asserted, not just its label and its count".
- Learning: "Assert diagram box placement relationally, by label".
- Learning: "Look at the rendered page: it finds what a green viewer suite cannot". The viewer now draws activation arrows from copies.

**Testable:** Yes

**Certainty:** high: event copies already exist end to end (`event_copy.go:28-50`, `svg.go:154-162`, `export/diagram.go:272-299`, `export/diagram.go:476-481`). This task changes the predicate and joins the activation edge to the same mechanism, and the prototype has the SVG half.

**Blast radius:** high: the diagram-json output of `emod export` and of the MCP `emod_export` tool is consumed outside this repository, and this task changes which node a `subscription` or `automation_trigger` edge starts at. The viewer's save path also writes the user's `.emod` file through those edges.

**Verification:** `mise exec -- go test -tags unit ./internal/diagram/ ./internal/export/ ./internal/importer/` passes, then `task test:unit`. Run `task test:e2e:viewer` where the environment can run Playwright. Then build the bundle (`task build:web`), open the fixture in the browser, and look at the activation arrows that start at a copy.

**Depends on:** None

### Task 2: Copy far views and commands in the SVG

**Behavior:** The SVG applies the same distance rule to a view's arrow into a trigger, an automation or a translation (reads), and to an automation's or translation's arrow into a command (command). The reader's slice draws a copy of the far view, with "from <slice>". The automation's or translation's slice draws a copy of the far command, with "in <slice>". Both kinds of copy sit in the Commands / Views lane. The diagram JSON is not changed by this task.

**Acceptance Criteria:**
- [x] A view's arrow into a trigger, an automation or a translation that joins two slices 2 or more apart in layout order starts at a copy of the view, drawn in the reader's slice. For a translation, the arrow from the copy runs into the external system box in front of the reactor, as the arrow from the view does today.
- [x] An automation's or a translation's arrow into a command 2 or more slices away ends at a copy of the command, drawn in the slice of the automation or the translation.
- [x] When a translation's slice draws a copy of its command, the command -> event arrow that the translation implies starts at that copy and ends at the event the translation nests.
- [x] A copy of a view shows the view's name and "from <slice>". A copy of a command shows the command's name and "in <slice>". In both, <slice> is the first slice that declares the element.
- [x] Arrows inside one slice, or between two slices next to each other in layout order, stay arrows. A slice draws one copy per element, however many of its readers or reactors join it, and no copy of an element it declares itself.
- [x] Copies of views and commands sit in the Commands / Views lane, in the column of the slice that draws them, and overlap no other box in that row. The column is widened for them as it is for event copies. Assert the column relationally, by a box the slice owns.
- [x] A copy is outlined dashed in the stroke of the element type it copies and filled white. The fill constant is named for copies, not for events.
- [x] A test that finds a box by an element's name does not find that element's copy. The spec-card subtest and `TestExporterReadsEdges` state what a reader sees: an arrow from a copy counts as an arrow from the view it copies, and clearing a far read takes away that read's copy along with its arrow.
- [x] The diagram JSON does not change: `go test -tags unit ./internal/export/ ./internal/importer/` passes with no edits to those tests.
- [x] `docs/dsl-reference.md` states the rule for views and commands in the SVG, and its palette note covers copies of views and commands.

**Affected Files/Modules:**
- `internal/diagram/element_copies.go` (from Task 1): adds the view and command kinds, and the copied end of the reads and command arrows.
- `internal/diagram/element_copies_test.go`: the rule for views and commands.
- `internal/diagram/layout.go`: `columnWidth` counts the copies in the Commands / Views row. The copy fill constant is renamed.
- `internal/diagram/svg.go`: places view and command copies in the Commands / Views row. Reads arrows start at a view copy, and command arrows end at a command copy.
- `internal/diagram/contract_test.go`: subtests for view and command copies. `TestExporterReadsEdges` reads a copy as its view.
- `internal/diagram/svg_test.go`: the spec-card subtest's box lookup.
- `internal/export/diagram.go`: only as needed to keep the JSON as Task 1 left it.
- `internal/test/fixtures.go`, `internal/test/models.go`: a fixture with a far read and a far command, if no existing fixture states both.
- `docs/dsl-reference.md`: the copy paragraph and the palette note.

**Patterns to Follow:**
- Follow how the SVG places event copies in `internal/diagram/svg.go:154-162`, and how it places the middle row in `internal/diagram/svg.go:101-118`.
- Follow how a reads arrow resolves its view in `internal/diagram/svg.go:194-204`.
- Follow how a slice is widened for its copies in `internal/diagram/layout.go:188-205`.
- Follow the exact-line lookup in `internal/diagram/contract_test.go:2069-2081` (`boxNamed`) instead of the substring lookup at `internal/diagram/contract_test.go:317-329`.
- Follow the reads-arrow assertions in `internal/diagram/contract_test.go:796-1040`.
- Follow how the SVG writes the colour constants in `internal/diagram/layout.go:78-108`.
- Task 1's `ElementCopies` and its tests.
- Prototype diff of `svg.go` and `layout.go` in `/private/tmp/claude-501/-Users-davidnguyen-Personal-Code-emod/b7811524-de39-4605-bd6a-cce7eac6e50e/scratchpad/wt`.
- Learning: "A per-slice shape needs its column asserted, not just its label and its count".
- Learning: "Assert diagram box placement relationally, by label".
- Learning: "Read an SVG arrow back as the two boxes it meets".
- Learning: "One fill and stroke constant per element type — never named after the shape, never aliased".
- Learning: "The element palette is written down four times and pinned by three Go tests". A copy is not an element type, so it has no palette row.

**Testable:** Yes

**Certainty:** medium: event copies in the event row are the precedent (`svg.go:154-162`, `layout.go:188-205`). The variation is that no copy sits in the Commands / Views row yet, and that a command copy is the end an arrow ends at, where every copy today starts its arrow.

**Blast radius:** low. The task changes only the SVG picture. The diagram JSON is unchanged.

**Verification:** `mise exec -- go test -tags unit ./internal/diagram/` passes, then `task test:unit`. Render `examples/all_patterns.emod` with `emod diagram --format svg` and look at the copies in the Commands / Views lane.

**Depends on:** Task 1

### Task 3: Draw copies of views and commands in the viewer

**Behavior:** The viewer draws a `view_copy` or `command_copy` node from a diagram document in the same way it draws an `event_copy`. The copy is a dashed white block in the stroke of the type it copies, with the original's name and the slice line: "from <slice>" for a copy of an event or a view, "in <slice>" for a copy of a command. Each copy behaves as an event copy does: the pointer marks its original, a click shows the original, and the viewer never moves, connects or edits a copy. No writer emits the two new types until Task 4.

**Acceptance Criteria:**
- [x] A document node of type `view_copy` or `command_copy` is drawn inside the slice that its `parentId` names, as a dashed block filled white in the stroke of a view or of a command.
- [x] A copy shows its original's name and, under it, the slice that holds the node its `copy_of` names. That line is "from <slice>" for a copy of an event or a view, and "in <slice>" for a copy of a command. A copy shows the name alone once the document no longer holds its original.
- [x] An arrow whose source or target is a copy is drawn between the copy and the other block: a `reads` arrow from a view copy, and an `automation_command` or `translation_command` arrow into a command copy.
- [x] Every behaviour the viewer's tests state for a copy of an event holds for a copy of every kind (`internal/frontend/tests/event-copy.test.js`, and `viewer.test.js` "on a copy of an event"):
  - its layout and its paint;
  - no port;
  - a connection or a repoint released over it is refused;
  - a double click opens no rename box, and a right click opens no menu;
  - the pointer over it marks its original, and the tooltip names the slice;
  - a click highlights the original and centres it;
  - the pipeline is asked nothing;
  - dropped on another slice, it stays in its own.
- [x] Each slice is wide enough for its widest copy, every copy stays inside its slice at rest, and dragging a slice moves its copies with it.
- [x] The node type strings are `view_copy` and `command_copy`, as the planned design names them.
- [x] `task test:viewer` passes. A page built with `task build:web` and loaded with a document holding all three kinds of copy shows them legibly, with no overlap.

**Affected Files/Modules:**
- `internal/frontend/static/model.js`: one predicate for every copy kind, and the move refusal.
- `internal/frontend/static/layout.js`: where copies of each kind sit, their widths, the slice line, and `getSliceChildNodeIds`.
- `internal/frontend/static/renderer.js`: `buildCopyBlock` takes its stroke from the copied type, and the dispatch reaches every copy kind.
- `internal/frontend/static/ui.js`: the tooltip, the mark on the original, the click and the context menu, for every copy kind.
- `internal/frontend/static/interaction.js`: the connect and repoint refusal.
- `internal/frontend/static/config.js`: the copy style, named for copies, not for events.
- `internal/frontend/static/viewer.html`: the copy block rules, if the class changes.
- `internal/frontend/tests/event-copy.test.js`, `internal/frontend/tests/viewer.test.js`: the copy behaviour across kinds.
- `e2e-viewer/tests/copies.spec.js`: only if the copy block's class changes.

**Patterns to Follow:**
- Follow the predicate and the original lookup in `internal/frontend/static/model.js:223-235`, and the move refusal in `internal/frontend/static/model.js:156-170`.
- Follow the slice line in `internal/frontend/static/layout.js:69-73`, the copy widths in `internal/frontend/static/layout.js:100-118`, the slice stack in `internal/frontend/static/layout.js:155-161`, and the slice's child ids in `internal/frontend/static/layout.js:314-322`.
- Follow the copy block in `internal/frontend/static/renderer.js:537-558` and its dispatch at `internal/frontend/static/renderer.js:645`.
- Follow the copy interactions in `internal/frontend/static/ui.js:83-131`, `internal/frontend/static/ui.js:1090-1100` and `internal/frontend/static/ui.js:1190-1200`.
- Follow the connect refusal in `internal/frontend/static/interaction.js:360-370`.
- Follow the copy style in `internal/frontend/static/config.js:55-57`.
- Follow the tests in `internal/frontend/tests/event-copy.test.js:1-175` and `internal/frontend/tests/viewer.test.js:2225-2350`.
- Learning: "`computeLayout`'s `minW` must cover every child it placed, not the column band".
- Learning: "A diagram-node key has three readers, and they must move in one commit". This task is the reader half that comes first; see the Summary.
- Learning: "`svgPicture` sees labelled boxes and arrows only, so it is not the receipt for a new mark". `drawnBoxes` in `renderer.test.js` reads a group's first rect only.
- Learning: "The element palette is written down four times and pinned by three Go tests". A copy is not an element type and gets no `nodePalette` entry.
- Learning: "Look at the rendered page: it finds what a green viewer suite cannot".

**Testable:** Yes

**Certainty:** medium: the viewer draws and guards event copies throughout (`renderer.js:537-558`, `model.js:223-235`, `ui.js:83-131`). The variation is that the stroke and the slice line now follow the copied type, with "in" for commands, and that an arrow can end at a copy where every arrow today starts at one.

**Blast radius:** low. The viewer only draws node types that no writer emits yet, and a copy is never read back as a declaration.

**Verification:** `task test:viewer` passes. Then build with `task build:web` and look at the rendered page with a hand-written document holding all three copy kinds.

**Depends on:** None

### Task 4: Write view and command copies into the diagram JSON and read them back on save

**Behavior:** The diagram JSON writes a `view_copy` or `command_copy` node for every view or command copy that the rule in `internal/diagram` decides. `reads` edges start at a view copy, and `automation_command` and `translation_command` edges end at a command copy. After this task, the SVG and the viewer draw the same copies. On the viewer's save path, the importer reads all three copy kinds as stand-ins for their originals and never as declarations.

**Acceptance Criteria:**
- [x] For every copy of a view or a command that `ElementCopies` decides, the diagram JSON holds one node of type `view_copy` or `command_copy`. The node sits in the slice that draws the copy, is labelled with the element's name, and its `copy_of` names the node drawn for the first declaration of that view or command.
- [x] A `reads` edge whose view is copied starts at that slice's `view_copy` node. An `automation_command` or `translation_command` edge whose command is copied ends at that slice's `command_copy` node, and the command -> event edge that the translation implies starts at that same node. Every other edge joins the original nodes as before.
- [x] For each fixture, the diagram JSON and the SVG copy the same elements in the same slices.
- [x] On the viewer's save path (`ApplyDiagram`), every behaviour stated for a copy of an event holds for a copy of a view and a copy of a command:
  - Applied unedited, a diagram with copies leaves the model as written.
  - Renaming the original renames the reference made through the copy (a reader's `reads`, a reactor's `command`).
  - Deleting the original clears the references made through its copies.
  - Deleting the arrow at a copy clears that one reference.
- [x] `ImportDiagram` of a document holding view and command copies declares each view and each command once.
- [x] In a browser, the viewer draws view and command copies and their arrows for a model with a far read and a far command (`e2e-viewer/tests/copies.spec.js`). Look at the rendered page.
- [x] `docs/dsl-reference.md` states that the viewer draws copies of views and commands as the SVG does.

**Affected Files/Modules:**
- `internal/export/diagram.go`: the `view_copy` and `command_copy` nodes, `copy_of` for views and commands, and the reads and command edges at copies.
- `internal/export/export_test.go`: the copy nodes and the edge ends.
- `internal/importer/importer.go`: one copy predicate for every copy type.
- `internal/importer/diagram_edit.go`: `matchDrawnNodes`, `nameCopiesAfterOriginals` and `inResult` for every copy type.
- `internal/importer/diagram_edit_test.go`, `internal/importer/importer_test.go`: the save and import behaviour for view and command copies. The `node` helper must pass over every copy type.
- `e2e-viewer/tests/copies.spec.js`: view and command copies in a browser.
- `docs/dsl-reference.md`: the copy paragraph.

**Patterns to Follow:**
- Follow the copy nodes and their originals in `internal/export/diagram.go:272-299`.
- Follow the edge at a copy in `internal/export/diagram.go:476-481`.
- Follow the first-declared ids in `internal/export/diagram.go:155-163`.
- Follow the JSON copy assertions in `internal/export/export_test.go:3792-3829` and `internal/export/export_test.go:5922-5948`.
- Follow the copy predicate in `internal/importer/importer.go:46-50`.
- Follow the save path's copy handling in `internal/importer/diagram_edit.go:76-88`, `internal/importer/diagram_edit.go:243-247` and `internal/importer/diagram_edit.go:534-548`.
- Follow the save-path copy tests in `internal/importer/diagram_edit_test.go:2090-2160` and their helpers in `internal/importer/diagram_edit_test.go:327-357`.
- Follow the browser tests in `e2e-viewer/tests/copies.spec.js:1-140`.
- Task 2's `ElementCopies` kinds and Task 3's viewer readers.
- Learning: "A diagram-node key has three readers, and they must move in one commit".
- Learning: "The viewer's save path is `importer.ImportDiagram`, so a diagram-node field owes a read-back".
- Learning: "An importer fold must re-check the endpoint node types, not only the edge type". `readsDrawnFromView` takes a label only from a node of type `view`.
- Learning: "`EDGE_TYPE_BY_ENDS` carries only the direction the exporter writes". The viewer never draws an arrow from a copy, so no pairing is added.
- Learning: "An arrow between two constructs is drawn by five surfaces, and none of them reads another".
- Learning: "Look at the rendered page: it finds what a green viewer suite cannot".

**Testable:** Yes

**Certainty:** medium: `appendEventCopies` and `pointCopiesAtOriginals` (`export/diagram.go:272-299`), and the importer's event-copy handling (`diagram_edit.go:76-88`, `:534-548`), are the precedent. The variation is that `copy_of` now points at a view or a command node, and that the copied end of a command edge is its target, where every copy today is an edge's source.

**Blast radius:** high: this task adds two node types to the diagram-json output of `emod export` and of the MCP `emod_export` tool, which are consumed outside this repository. The viewer's save path reads them before it writes the user's `.emod` source.

**Verification:** `task test:unit` and `task test:viewer` pass. Run `task test:e2e:viewer` where the environment can run Playwright. Build with `task build:web` and look at a model with a far read and a far command in the browser.

**Depends on:** Task 2, Task 3

### Task 5: Start and end every SVG arrow on the side of the boxes it joins

**Behavior:** Today every lane-SVG arrow runs from box centre to box centre and is written after the boxes, so each line crosses the labels of both boxes it joins. After this task, each arrow leaves its source box on a side and ends with the tip of its arrowhead on a side of its target box. The test reader names an arrow by the boxes its ends lie on, not by box centres.

**Acceptance Criteria:**
- [x] Every arrow the lane SVG draws, whatever its kind, starts on the outline of its source box and ends with its arrowhead tip on the outline of its target box. No arrow is drawn across the inside of either box it joins. This includes the dashed arrow to a rejection badge, arrows from and to copies, and the arrow into an external system box.
- [x] An arrow into or out of an automation does not cross the name written under its gear.
- [x] The text that an arrow carries (an automation's delay) sits on that arrow's horizontal leg.
- [x] `svgConnections` names each arrow by the boxes whose outline its two ends lie on. It fails loudly when an end lies on no box's outline. The rejection-badge tests find the badge that each dashed arrow reaches by the box its end lies on, not by a centre point.
- [x] Every test that reads connections passes with its assertions unchanged, apart from the reader itself.
- [x] Mermaid and ASCII output is unchanged.

**Affected Files/Modules:**
- `internal/diagram/svg.go`: `svgArrowBetween`, `svgDashedArrowBetween` and its centre contract comment, `svgArrowPath`, and `svgArrowLabel`.
- `internal/diagram/svg_test.go`: `svgConnections`, `dashedArrowEndpoints`, and the rejection-badge subtests.
- `internal/diagram/contract_test.go`: any helper that reads an arrow end as a centre.

**Patterns to Follow:**
- Follow how the event-flow SVG starts an arrow on a node's side, below its captions, in `internal/diagram/eventflow_svg.go:101-109` and `internal/diagram/eventflow_svg.go:166-185`.
- Follow the reader to change in `internal/diagram/svg_test.go:1278-1314`.
- Follow the badge comparisons in `internal/diagram/svg_test.go:418-470` and `internal/diagram/svg_test.go:941-965`.
- Follow the shape reader in `internal/diagram/svg_test.go:1066-1100`.
- Learning: "Read an SVG arrow back as the two boxes it meets". This task changes the mechanism that learning describes; the run records the new one.
- Learning: "`svgPicture` sees labelled boxes and arrows only, so it is not the receipt for a new mark".
- Learning: "Assert diagram box placement relationally, by label".

**Testable:** Yes

**Certainty:** medium: `eventFlowArrow` (`eventflow_svg.go:166-185`) starts on a node's bottom side and ends on the top side. The variation is that lane-diagram arrows run up, down and inside one lane (automation -> command, view -> automation), that a gear's name hangs under it, and that the test reader resolves ends by centre today.

**Blast radius:** low. The task changes only the SVG picture.

**Verification:** `mise exec -- go test -tags unit ./internal/diagram/` passes. Render `examples/all_patterns.emod` and confirm that no line crosses a box's text.

**Depends on:** Task 2

### Task 6: Give each SVG arrow a horizontal leg of its own between lanes

**Behavior:** Today all arrows between the same two rows put their horizontal leg at the midpoint of the two rows, so they lie on one line. After this task, each arrow that joins two lanes runs its horizontal leg in the lane gap between them, at a y that no other arrow's leg in that gap uses, and each delay label stays on its own arrow's leg.

**Acceptance Criteria:**
- [ ] An arrow that joins boxes in two different lanes runs its horizontal leg in a lane gap between those two lanes.
- [ ] Two arrows whose horizontal legs run in the same lane gap run them at different y.
- [ ] When a lane gap must hold more legs than its height allows, the gap grows. The lanes below it move down together with their boxes and any spec card band, so no box leaves its lane.
- [ ] An arrow's delay text sits on its own arrow's horizontal leg, and on no other arrow.
- [ ] These hold for every model in `examples/` that parses, and for a generated model of 40 slices built in the test.
- [ ] The tests that assert box placement still pass with their relational assertions unchanged.

**Affected Files/Modules:**
- `internal/diagram/svg.go`: the arrow route and the place of the arrow's text.
- `internal/diagram/layout.go`: the lane gap, if it grows with the legs it holds, and the lane positions that follow from it.
- `internal/diagram/svg_test.go`: a reader for horizontal legs, and the subtests for this task.

**Patterns to Follow:**
- Follow the route to change in `internal/diagram/svg.go:457-472`, and the label placement in `internal/diagram/svg.go:417-434`.
- Follow how lane positions are derived in `internal/diagram/svg.go:22-38`, and the gap constants in `internal/diagram/layout.go:51-76`.
- Follow the arrow reader in `internal/diagram/svg_test.go:1278-1314` (as Task 5 leaves it).
- The measure script at `/private/tmp/claude-501/-Users-davidnguyen-Personal-Code-emod/b7811524-de39-4605-bd6a-cce7eac6e50e/scratchpad/gen/measure.py` counts shared line length. Its `overlap` is the check, if the scratchpad still exists.
- Learning: "Assert diagram box placement relationally, by label".
- Learning: "`svgPicture` sees labelled boxes and arrows only, so it is not the receipt for a new mark".

**Testable:** Yes

**Certainty:** low: no precedent. Both SVG writers in this repository put every leg at the midpoint (`svg.go:469`, `eventflow_svg.go:182`). How legs share out a gap, and whether a gap grows, is what this task decides.

**Blast radius:** low. The task changes only the SVG picture.

**Verification:** `mise exec -- go test -tags unit ./internal/diagram/` passes. Run the generator and the measure script, if they still exist, to see the shared line length.

**Depends on:** Task 5

### Task 7: Route every SVG arrow clear of the boxes it does not join

**Behavior:** Some arrows still go across boxes they do not join: a vertical stretch through a gear under a command, or a reads arrow down through the Events lane to an external system. After this task, the router chooses the point on each side and the x of each vertical stretch so that no arrow crosses any box it does not join, and no two arrows from different boxes run along one vertical line.

**Acceptance Criteria:**
- [ ] No arrow runs across any box the SVG draws, other than the two it joins. A box here is every shape that `svgShapes` reads back except the lanes, together with the name written under an automation's gear.
- [ ] No vertical stretch of one arrow lies on another arrow, unless both arrows start at one box or both end at one box.
- [ ] The guarantees of Tasks 5 and 6 still hold.
- [ ] These hold for every model in `examples/` that parses, and for the generated 40-slice model of Task 6.

**Affected Files/Modules:**
- `internal/diagram/svg.go`: the route of each arrow (the side point and the vertical stretches).
- `internal/diagram/svg_test.go`: a reader for the stretches of an arrow, and the subtests for this task.

**Patterns to Follow:**
- Follow the shape reader in `internal/diagram/svg_test.go:1066-1100`.
- Follow how reactors sit under commands and views in `internal/diagram/layout.go:314-337`.
- Follow how translations stack their external system boxes in `internal/diagram/svg.go:177-189`.
- The router Tasks 5 and 6 leave.
- The measure script's `through` check at `/private/tmp/claude-501/-Users-davidnguyen-Personal-Code-emod/b7811524-de39-4605-bd6a-cce7eac6e50e/scratchpad/gen/measure.py`, if the scratchpad still exists.
- Learning: "`svgPicture` sees labelled boxes and arrows only, so it is not the receipt for a new mark". A gear's name is text, not a box.
- Learning: "Assert diagram box placement relationally, by label".

**Testable:** Yes

**Certainty:** low: no router in this repository avoids boxes. Where on a side an arrow starts, and which x a vertical stretch takes past the reactor row and the Events lane, is the design this task makes.

**Blast radius:** low. The task changes only the SVG picture.

**Verification:** `task test:unit` passes. Render `examples/all_patterns.emod` and the generated 40-slice model, and look. The measure script, if it still exists, reports 0 arrows through a box.

**Depends on:** Task 6

## Summary

- **Seven tasks.** Six are mostly Go. Task 3 is JavaScript.
- **Order.** The copy rule goes first, because it removes most long arrows: on the 40-slice model the prototype cut arrows through a box from 105 to 10 and crossings from 376 to 2. The router then has fewer, shorter arrows to place.
  - Task 1 keeps the existing `event_copy` node type, so it changes no wire format. It makes the SVG and the diagram JSON agree on the distance rule for events.
  - Task 2 extends the rule to views and commands in the SVG only.
  - Task 3 teaches the viewer to read the two new node types before any writer emits them.
  - Task 4 switches the writer on, together with the importer, which reads the same nodes on the save path.
  - The router follows as three steps, each a visible fix on its own: arrows end on box sides, each arrow gets its own leg, and arrows stay clear of the boxes they do not join. The riskiest work comes last, on top of a picture the copies have already simplified.
- **Why the intermediate commits are safe.** After Task 2, the SVG copies views and commands while the viewer still draws those arrows long, as it does today. That is a difference between two pictures, not a regression, and Task 4 removes it. After Task 3, the viewer reads `view_copy` and `command_copy` before anything writes them. The learning on three readers warns about a writer that is ahead of its readers. Readers ahead of the writer leave no node unread, so no commit drops a node or an arrow that the user could see. Task 4 moves the remaining writer and reader (export and importer) together.
- **Story criteria covered:**
  - Part 1: Tasks 1 and 2 in the SVG, and Tasks 1, 3 and 4 in the diagram JSON, the importer and the viewer.
  - Part 2 (one rule, decided once in `internal/diagram`): `ElementCopies` from Task 1, which Tasks 2 and 4 extend and consume.
  - Part 3: Tasks 5, 6 and 7.
  - Nothing is deferred except the viewer's highlight walk, which the model-queries stories own.

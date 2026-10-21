# US-011: Keep the source panel and the diagram in agreement

## Contents

1. Task 1: Apply a diagram edit onto the model the diagram was drawn from
2. Task 2: Answer an applied diagram edit through the platform seam
3. Task 3: Write diagram edits into the source panel
4. Task 4: Say when the diagram and the source panel disagree, and let the user choose
5. Task 5: Write field editor edits into the source panel without losing what is typed
6. Task 6: Move a block to another slice by dragging it there

Task 1 decides what the source becomes. It is first because it is the least certain and because
everything after it depends on it. Task 2 carries that answer to both hosts. Tasks 3 to 6 are the
viewer: Task 3 writes diagram edits into the panel, Task 4 handles the case where both surfaces were
edited, Task 5 adds the field editor and Task 6 adds the drag that moves a block between slices.

| Slice | Tasks | Exit |
|---|---|---|
| **What the source becomes** | 1 | Go applies any edit the viewer can make onto the parsed model. Everything the diagram does not carry stays, and a fmt-clean file stays fmt-clean |
| **Getting it to the page** | 2 | `applyDiagram` answers the edited source with its diagnostics and its diagram, in the browser and on the desktop |
| **The panel follows the diagram** | 3 | Every canvas edit except the field editor's rewrites the panel, redraws in place and marks the window modified. Save writes it |
| **Nothing is lost when both changed** | 4 | Edits on both surfaces raise a notice that offers a choice between them. Nothing is discarded without that choice |
| **Field edits** | 5 | Field editor edits reach the panel, and an async redraw does not lose what is typed |
| **Moving between parents** | 6 | A block dropped on another slice moves there, in the diagram and in the source |

---

## Story Reference

`user-stories/emod-desktop.md` → **US-011: Keep the source panel and the diagram in agreement**
(`:165-178`), with Open Question 1 (`:263`). It depends on US-009, which is delivered
(`tasks/completed/us-009-edit-source-in-the-app-and-see-diagnostics-as-you-type.md`). It also builds on
US-003 and US-004 (save, unsaved-change marking) and on US-010, all delivered.

| Story criterion | Task |
|---|---|
| Adding, moving between parents, or deleting elements on the diagram updates the source panel to match | 1 (what the source becomes for every edit, moves included), 3 (adding, deleting and every other canvas edit reach the panel), 5 (field edits), 6 (the gesture that moves a block between parents) |
| Editing the source and revalidating updates the diagram, as in US-009 | 3 (a regression guard after a diagram edit has been applied), 4 (the one case it is held off: while a conflict stands) |
| An edit on one surface never silently discards an unapplied edit on the other; when both have diverged the app says so and offers a resolution | 3 (an answer never overwrites typing, and an open, a Render, a host failure or a second edit never drops an edit being applied), 4 (the notice and its two resolutions), 5 (text typed in the field editor) |
| Save writes exactly the source shown in the panel | 3 (after a diagram edit, with CRLF kept, and when Save comes while an edit is being applied), 4 (during a conflict) |
| A file that was `emod fmt`-clean before a diagram edit is still `emod fmt`-clean after the app rewrites its source | 1 (every edit kind, in Go), 3 (through the real pipeline in a browser) |
| Purely visual state — node offsets, pan, zoom, panel collapse — never appears in the saved source | 1 (display-only keys change nothing), 3 (no visual gesture starts an apply), 6 (a block dropped inside its own slice stays a visual offset) |

---

## Boundaries

**Out of scope** (the story set's non-goals that bear on this story, verbatim from
`user-stories/emod-desktop.md:248-259`):

- **No replacement of existing distributions.** `emod diagram --serve` and the hosted web viewer keep working unchanged; desktop is a third option, not a migration.
- **No desktop-only diagram features.** Anything added to the canvas stays available in all three distributions.
- **No full text editor.** The source panel gains validation, navigation, and highlighting — not find-and-replace, multi-cursor, or refactoring.
- **No automated tests driving the desktop window.** Manual smoke testing of the shell, backed by the existing Go and browser suites.

**Out of scope** (added by this breakdown):

- **Export .emod does not change.** It still re-serialises the whole diagram through
  `pipeline.ExportEmod` and drops what the diagram does not carry. The new path is separate.
- **Diagrams with no source behind them.** This covers the diagram `emod diagram --serve` hands the
  page and diagram JSON pasted into the panel. Diagram edits on these do not write the panel, and
  they cannot raise a conflict (Decision 1). The drag in Task 6 still moves a block on them, and
  Export writes the move.
- **Moving a slice to another aggregate or context.** Slices are reordered only among their own
  siblings, as today (user decision 2).
- **Merging the two surfaces' edits.** A conflict is resolved by choosing one side. Neither side is
  combined with the other.
- **References the diagram draws no arrow for**, when a construct is renamed or deleted: a spec's
  `given`/`when`/`then` names, rejection lines, `decides_on` events, a tag's field and an
  automation's `target` context. They keep the old name, and the validator reports them (user
  decision 1).
- **Comments that `emod fmt` does not keep.** These are comments inside a `fields` block, after the
  last entry of a block, or at the end of the file. They are lost when a file that was not in the
  `emod fmt` layout is rewritten. The bar says how many were lost (Decision 2).
- **Undo of a diagram edit in the source panel.** When the viewer replaces the panel's text, the
  browser clears the textarea's own undo history. The file on disk does not change until Save.
- **Deleting an arrow the source has no text for**: a trigger's arrow to a command in its slice, or
  the command→event arrow a translation implies. Deleting one does not change the source, so the
  redraw draws it again.

**Deferred:**

- **Save while a rename box or a field input still has focus.** Save writes what the panel holds.
  The text typed in the box reaches the panel when the box is left. If wanted, this is a later
  story.
- **A highlight on the slice that a dragged block would move to.** Task 6 moves the block on release
  and shows no target while the block is dragged.
- **A message that says why a slice refused a block** (a second trigger, or the event a translation
  nests). Task 6 returns the block to where it was and shows no message.

---

## Decisions

### Decision 1 — all three distributions, and only where the diagram has source behind it

The user decided this before decomposition. The frontend is one copy (`internal/frontend/static`),
so the change ships in the desktop app, the web viewer and `emod diagram --serve`
(`tasks/learnings.md:1358-1363`). A diagram edit is written into the panel only when the diagram on
screen was drawn from `.emod` source in the panel. That source can be opened, dropped, pasted or
typed. A diagram from pasted diagram JSON is not written back. Neither is the diagram that
`emod diagram --serve` gives as `INITIAL_DATA` (`internal/viewer/serve.go`, read at
`viewer.js:830-832`). Diagram edits on these behave as they do today.

**Goal tension, for the user:** in the web viewer and `--serve`, the panel now changes when the
diagram is edited, a conflict notice can appear, and a block dragged onto the next slice moves there
instead of growing its own slice. The last change applies even with no source. US-009 read the goal
as "nothing stops working". Under that reading, Render, drop, Export and Save keep working. One
gesture does change meaning: dragging a block past its slice's right edge onto the neighbouring
slice. If the user reads the goal more strictly, the place to say so is the plan gate, before
Task 6.

### Decision 2 — a diagram edit is applied onto the parsed source, not exported again

The user decided this before decomposition. `pipeline.ExportEmod` (`internal/pipeline/pipeline.go:76`)
is the Export path. On `examples/specs_hotel.emod` it differs from `emod fmt` by 70 diff lines,
because it drops the model comment, the invariants, the specs, the rejections and the event
`type =` lines. So it cannot be the path a diagram edit takes. Instead, Go parses the source the
diagram was drawn from and changes only what the diagram changed. It then writes the result with
`formatter.Format`. `emod slices arrange` already does this: parse, change the model, format
(`internal/cli/slices_arrange.go:13-51`). The table shows what stays because the diagram does not
carry it:

| Kept from the parsed source | Where the diagram document stands |
|---|---|
| The model's comment and descriptions | No node |
| Comments on every construct, with their order | The node carries text only, no position (`tasks/learnings.md:571-575`) |
| Invariants and their comments, specs, rejection lines | No node |
| Flow lines and their comments, other than the ones an edit adds or removes | Edges only |
| Event `type =`, `tags`, context `mode`, `decides_on`, automation `target` | Not carried |

A file that is already in the `emod fmt` layout comes back in that layout. A file that is not gets
that layout on its first diagram edit, and the bar says so (Decision 4). **This breakdown adds one
thing to the user's decision:** the notice also says how many comments the layout dropped, when it
dropped any. `emod fmt` keeps only comments written above a construct. A probe confirms that it
drops a comment inside a `fields` block, a comment after a block's last entry, and a comment at the
end of the file. Open Question 1 is about these comments. The count costs one comparison in Go. The
caller can remove this addition.

### Decision 3 — what an edit is applied onto: the drawn source, matched by the export's ids

An edit is applied onto the **drawn source**: the text the diagram on screen was drawn from. It is
never applied onto whatever the panel holds now. Diagram node ids are assigned per kind, in document
order (`internal/export/diagram.go:122-136`). So exporting the drawn source again gives each
unedited node the id it has on screen. Comparing that export with the edited diagram gives the
nodes added, removed and kept, and the edges added and removed. An edge is identified by its
source, target and type. Nodes the viewer adds have ids the export never gives
(`Model.generateNodeId`, `model.js:7-13`). After each applied edit, the panel holds the result and
the diagram is drawn from it. The result therefore becomes the drawn source for the next edit.

A diagram drawn from source that did not parse gives an edit no model to apply onto. An example is
a file opened with parse errors, which `viewer.js:254-264` draws anyway. Applying the edit fails
with a reason, and the diagram keeps the edit, marked out of date (Task 3).

### Decision 4 — where the window says each thing

The window says three things. Each goes where the user can read it with the data panel collapsed
(`tasks/learnings.md:1067-1071`):

- **The source was rewritten in the `emod fmt` layout.** This goes to the bar's `#save-status`
  (`viewer.html:1353-1360`), as a confirmation does. The bar does not wrap and the slot is 30ch, so
  the full text is in the hover title. A save confirmation written later replaces it. It never
  replaces a save confirmation that came after the edit (`tasks/learnings.md:1187-1191`).
- **The two surfaces disagree.** This goes in a notice over the canvas, where `#stale-notice` sits
  (`viewer.html:220-245`, `:1320`). The notice holds two buttons, so it cannot go in a bar slot
  that clips text.
- **A diagram edit could not be applied.** This uses the stale notice with the reason. The diagram
  then shows something the panel does not, which is what that notice exists to say.

### Decision 5 — moving between parents means dragging a block onto another slice

The user decided this before decomposition. Today `clampIntoSlice` and `commitDrag`
(`interaction.js:146-158`, `:217-231`) keep a dragged block inside its slice and record a visual
offset. This changes in two ways:

- A command, event, view, automation, translation or trigger released with its centre over
  **another** slice moves to that slice. The slice can be in any context or aggregate. The block's
  centre decides, not the pointer, so the grab point does not change where a block lands.
- A block released anywhere else keeps today's behaviour.

In the source, the declaration moves to the new slice and goes after the blocks of its kind already
there. Its comments go with it. Flow lines, rejection lines and specs that name the moved block stay
in the slice that states them. This follows the user's wording, "the source moves its declaration",
and the validator resolves those names across the whole model. The alternative is to move a flow
line with its command, and the caller can choose it. Two moves are refused:

- A slice holds one trigger, so a trigger released over a slice that has one goes back.
- The node of an event that a translation nests is declared inside its translation's block. It goes
  back when released alone, and it moves when its translation moves.

Slices are still reordered only among their siblings.

### Decision 6 — the conflict, and the holes this breakdown closes in it

| Surface | Has an unapplied edit when |
|---|---|
| Source | The panel's text differs from the drawn source. This happens while typing pauses, while a parse is in flight, when the text does not parse (the diagram is marked out of date) and when the panel is empty |
| Diagram | A diagram edit has not reached the panel: it is being applied, or it is held by a conflict, or applying it failed |

A diagram edit made while the source has an unapplied edit puts the window in a **conflict**. So
does typing while a diagram edit is being applied. The notice offers two choices:

- **Keep diagram edits** applies every diagram edit since the drawing onto the drawn source and
  replaces the panel's text with the result.
- **Keep source edits** redraws the diagram from the panel. When the panel does not parse, or is
  empty, it redraws from the drawn source and keeps the out-of-date mark.

While the conflict stands, revalidation lists the panel's diagnostics but never redraws the
diagram. A redraw would silently discard the diagram edits.

This breakdown closes five holes in the user's design:

- **The unsaved-edits guard.** It asks only when the panel differs from the file. A conflict whose
  panel matches the file would let Close or Open discard the diagram edits without asking. While a
  diagram edit is unapplied, the window counts as modified.
- **Save during a conflict.** Save writes exactly the panel, as criterion 4 says. The window stays
  marked modified, because the diagram edits are still not in the file. When the guard's Save is
  chosen during a conflict, the panel is written and the window stays open.
- **Render during a conflict.** Render means "draw the panel", so it resolves the conflict as Keep
  source edits does.
- **An open while an edit is being applied.** The guard reads the panel before the edit lands, so
  it would not ask. It must ask as if the edit had landed (Task 3).
- **A diagram drawn from source that did not parse.** Keep diagram edits has nothing to apply onto,
  so it reports the same failure Task 3 reports.

### Decision 7 — applying a diagram edit is a render nobody asked for

An apply redraws the diagram, and it is also the first render that writes the panel's text. The
learnings warn that these renders lose ordering races to opens, Render clicks, host failures and
saves, and that each audit round found one more (`tasks/learnings.md:1309-1314`, `:1351-1356`,
`:1073-1077`). Task 3 lists every claim an apply can lose to and states the outcome for each:

- A second edit.
- Typing.
- Save.
- An open from the dialog, recent files, a drop or a host delivery.
- A Render click.
- A host failure, such as a refused drop or an unreadable or empty file.

Plan two or three audit rounds for Tasks 3 and 4.

### Decision 8 — a field row with no name is not yet a field

**+ Add Field** adds a row with an empty name (`ui.js:539-547`). If that row were written to the
source it would not parse. In Go, a field with no name is left out (Task 1). In the viewer, adding a
row is not a diagram edit until the row has a name (Task 5). The field editor commits a value when
its input loses focus, and focus moves to the next input of the same editor. So the redraw after a
field edit must not lose the editor, or the text typed into it since (Task 5).

---

## Codebase Context

**The Go side.**

- **The importer.** `importer.ImportDiagram` (`internal/importer/importer.go:79-123`) builds an AST
  from diagram JSON. Its `builder` turns each node into a construct (`:174-311`). It recognises the
  standalone event node the exporter adds for a translation's nested event (`translationEventNames`,
  `:242-254`). It folds edges into the model (`foldEdges`, `foldReads`, `foldFlow`, `:320-409`).
  The fold only ever **adds**: it fills a relationship that is empty. The node keys `subscribes`,
  `on_event`, `command` and `reads` survive an edge's deletion, so a deleted subscription comes back
  on Export. The new apply has to clear what a removed edge stood for. It has to reuse the fold and
  nested-event rules rather than write them again.
- **The exporter.** It draws the nested event of a translation as its own node, with the event id
  that comes next in order (`internal/export/diagram.go:378-406`). It draws every edge from
  `diagram.SliceEdges` (`:408-455`). Two edge types come from no text of their own:
  `trigger_command` and the flow a translation implies (`internal/diagram/graph.go:84`, `:125`).
- **The formatter.** `formatter.Format` writes each slice's entries in a fixed kind order and
  hoists invariants (`tasks/learnings.md:141-145`). Its output starts with `emod N` (`:31-35`).
  A golden must be a canonical constant, never the input re-indented.
- **Fixtures.** `examples/all_patterns.emod`, `examples/dcb_model.emod` and
  `examples/specs_hotel.emod` are fmt-clean. Together they carry every kind of construct the
  diagram does not, listed under Decision 2. So are `internal/parser/testdata/all_patterns.emod`,
  `minimal.emod` and `multi_context.emod`. `examples/error_diagnostics_test.emod` parses but is not
  fmt-clean: `emod fmt` moves its leading comment and realigns its flow lines, and keeps all six
  comments.
- **The seam.** It carries one operation in five places. Go: `pipeline.ExportEmodJSON`
  (`internal/pipeline/pipeline.go:91-106`), `ModelService.ExportEmod`
  (`internal/desktop/service.go:31-37`) and `jsExportEmod` (`cmd/emod-wasm/main.go:26-37`).
  JavaScript: the contract in `platform.js:7` and its implementations at
  `platform.browser.js:132-142` and `platform.desktop.js:22-30`. Four guards scan these files:
  - `viewer.test.js:3674-3711` checks the export lists.
  - `internal/desktop/binding_names_test.go:22-36` checks the binding names.
  - `internal/pipeline/pipeline_test.go:56-67` checks the request keys each runtime sends.
  - `pipeline_test.go:162-176` checks the answer keys the viewer reads.

  `wasm-worker.js` dispatches any registered global by name, so it does not change. The WASM binary
  is not tracked (`.gitignore`), and `task build:web` builds it. Learning 1007-1011 puts each
  cross-boundary guard on the task that adds the caller. So the answer-keys guard belongs to Task 3,
  where `viewer.js` first reads the answer.

**The viewer.**

- **Render orchestration.** All of it is in `init`'s closure (`viewer.js:195-295`). It holds
  `latestRender`, `latestRenderSettled` with `rendersSettled()` (`:666-675`), and `checkedSource`,
  the last text the pipeline answered for, parsed or not. It schedules revalidation with
  `scheduleRevalidation` and `revalidate` (`:367-382`). `keepStaleDiagram` (`:317-323`) keeps a
  stale diagram. `reportHostFailure` (`:342-350`) claims a render number and draws nothing.
- **Save.** `saveQueue` (`:653-664`) orders saves. `sourceToSave` (`:32-42`) keeps a CRLF file
  CRLF. `modelIsModified` and `reportModified` (`:53-55`, `:358-360`) track unsaved edits. A
  programmatic `.value` write fires no `input` event.
- **Guards and opens.** `guarded` and `clearedToReplace` (`:396-444`) guard every open.
  `openModel` (`:410-416`) is the single entry point for opens.
- **In-place redraw.** `Model.setModelDataInPlace` (`model.js:23-33`) keeps offsets and hidden
  nodes by identity. The identity is parent path, type and label. The viewer computes it from the
  outgoing nodes, which already carry an edit's new labels and parents, so a renamed or moved node
  keeps its visual state across the redraw.
- **The drawn source.** No variable records it yet. `checkedSource` is not it, because a stale
  diagram's `checkedSource` is text that did not parse.

**Where the canvas changes the model.** Every site mutates `store.nodes` or `store.edges` in place
and then emits `data:changed` or `node:delete`:

- `CtxActions.apply` (`ctx-actions.js:63-87`) handles add slice, add command, add event, add flow,
  delete arrow, and move slice left or right. `viewer.js:790-813` dispatches it.
- Node delete, from the detail panel button (`ui.js:549-561`) and from Delete or Backspace
  (`ui.js:1046-1062`), is handled at `viewer.js:84-101`.
- Arrow delete comes from the detail panel (`ui.js:593-603`) and from the keyboard.
- Inline rename (`ui.js:699-727`) emits on Enter, and emits on blur through `setTimeout`.
- Field add and field delete (`ui.js:539-547`). The field input blur (`ui.js:516-523`) mutates the
  node and emits **nothing**.
- Connect and repoint (`interaction.js:293-343`). Repoint emits even when abandoned.
- Slice drag reorder (`interaction.js:345-388`).

Each emits `data:changed` for a re-render.

**Visual-only emitters.** These also emit `data:changed` but must never apply:

- The visibility tree (`ui.js:205-214`).
- `commitDrag` (`interaction.js:217-231`).
- Reset layout (`viewer.js:733-737`).
- Pan, zoom and fit view, which emit `viewport:changed`.
- The data panel header (`viewer.js:728-730`).

**Tests.**

- **vitest.** `viewer.test.js` drives a real `init()` against a mocked platform
  (`viewer.test.js:36-77`). It records and clears every timer a test starts (`:246-266`;
  `tasks/learnings.md:1316-1321`). Its helpers are `openBilling`, `typeIntoPanel`, `answerNext`,
  `blockFor`, `menuItemFor`, `dragBy` and `save` (`:136-245`).
- **The test this story inverts.** `'says nothing when the context menu adds a node to the diagram'`
  (`:1803-1810`) pins today's behaviour, which this story reverses. An added node is now a change
  that Save writes (`tasks/learnings.md:905-909`).
- **Drag tests.** `drag-containment.test.js` pins in-slice drag behaviour against laid-out positions
  under an identity screen mapping.
- **Playwright.** `e2e-viewer` runs the built web bundle with the real WASM pipeline. Its rules: open
  the data panel before filling the source, write comments with `#`
  (`tasks/learnings.md:1323-1328`), and check overlays with `elementFromPoint` (`landsOn`,
  `source-panel.spec.js:10-20`). `export.spec.js:53-115` already edits the diagram from the
  context menu.
- **Desktop.** No test drives the desktop window, so Tasks 3 to 6 each close with a manual check in
  the desktop build.

**Words already taken.**

- **move** already means reordering a slice among its siblings (`Model.moveSlice`,
  `move-slice-left`). That is why a block's change of slice is called a **move to another slice**.
- **edit** in the viewer is the rename box (`store.interaction.inlineEdit`). A **diagram edit** is
  any change on the canvas to what the model says, and a rename made through that box is one of
  them.
- **apply** already names `CtxActions.apply`, `applyViewport` and `applyRepoint`, all in the sense
  "make it take effect". That fits applying a diagram edit onto the source.
- **stale** is the out-of-date mark (`store.diagramStale`). **conflict** is unused in the repo.
- **view** is a construct, so the canvas-only state is called **visual state**.

---

## Tasks

### Task 1: Apply a diagram edit onto the model the diagram was drawn from

**Behavior:** `importer.ApplyDiagram` takes two inputs: a model parsed from source, and the diagram
document the viewer holds for it. That document is the one the exporter drew from the model, edited
on the canvas. The function changes the model to say what the edited diagram says, and changes
nothing the diagram did not change. Everything else in the model stays, including what the diagram
document never carries. Formatting the result with `formatter.Format` gives the source with the edit
applied, in the `emod fmt` layout.

**Acceptance Criteria:**
- [x] Given the exporter's own diagram for a model, unedited, the model does not change. This holds
  for every `.emod` file under `examples/` and `internal/parser/testdata/` that `emod fmt` leaves
  unchanged: formatting the result gives the file back byte for byte. The set is found by checking
  each file, not kept as a list.
- [x] A diagram edit elsewhere in the model keeps everything `formatter.Format` writes that the
  diagram document does not carry: the model's comment, the comments above every construct,
  invariants and their comments, specs, flow and rejection lines with their comments, event `type =`,
  `tags`, context `mode`, `decides_on` and automation `target`. This is shown on
  `examples/specs_hotel.emod` and `examples/dcb_model.emod`. Each kept item is first required of the
  fixture, so a fixture that lacks it cannot pass by default.
- [x] Adding nodes the way the context menu does declares each new construct with the name its node
  carries, after the existing constructs of its kind in its parent. This covers a slice under a
  context, a slice under an aggregate, a command or an event in a slice, and Add Flow's event (with
  a command when the slice has none) and its flow arrow. The test compares exact expected source.
- [x] Deleting a block node of any kind the viewer deletes (`ui.js:400-403`) removes its declaration
  and the comments written above it. This includes deleting the arrows the viewer deletes with it
  (`viewer.js:84-101`). It also clears each relationship one of those arrows was drawn from: a view's
  `subscribes` entry, an automation's `on` and `command`, a translation's `command`, the `reads` of a
  trigger, automation or translation, and a flow line. A reference that has no arrow in the diagram
  stays as written. These are a spec's `given`/`when`/`then`, a rejection line, `decides_on` events,
  a tag's field and an automation's `target` context.
- [x] A node whose label changed renames the construct it was exported from. This applies to every
  node kind the viewer renames: context, aggregate, slice and each block kind. The new name also
  replaces the old one in every relationship an arrow to or from that node was drawn from: flow line
  ends, `subscribes` entries, `on`, `command` and `reads`. References that have no arrow in the
  diagram keep the old name.
- [x] A command's or an event's fields follow its node's fields when a field is renamed, retyped,
  given a modifier, cleared of one, deleted or added. A field with an empty name is left out. A
  construct whose node's fields did not change keeps its fields exactly as written.
- [x] An arrow added between two nodes folds into the model as `ImportDiagram` folds it. An arrow the
  importer cannot fold changes nothing. A removed arrow clears the relationship it was drawn from.
  Moving either end of an arrow moves the relationship to the new end. Removing an arrow that no text
  of its own states changes nothing: a trigger's arrow to a command in its slice, or the
  command→event arrow a translation implies.
- [x] The slices of one context or one aggregate are declared in the order the diagram lists them.
  A slice moved left or right moves its whole block, comments included.
- [x] When a block node's parent is now another slice, its declaration and its comments move to that
  slice, after the blocks of its kind already there. A translation takes its nested event with it.
  Flow lines, rejection lines and specs that name the moved block stay in the slice that states them
  (Decision 5). A move the target slice cannot hold leaves the block where it was. This covers a
  trigger moved onto a slice that has one, and the node of an event a translation nests, moved apart
  from its translation.
- [x] The node of an event a translation nests stands for that nested event. Renaming it, or
  changing its fields, changes the event inside the translation block. Deleting it removes that
  event from the translation. It never becomes a second, top-level declaration.
- [x] A key a node carries only for display changes nothing. This covers its `position`, and any
  other key the importer does not read.
- [x] Each edit above, applied to a fmt-clean fixture, formats to text that is `emod fmt`-clean:
  formatting what it parses to gives it back. That text also parses with no parse diagnostics.
- [x] Malformed diagram JSON returns an error and leaves the model unchanged.
- [x] `ImportDiagram` and Export .emod behave as they did before, and the existing importer tests
  pass. The apply uses `ImportDiagram`'s own rules to build a construct from a node, to fold an
  arrow, and to recognise a translation's nested event. Each rule has one text in the package.
- [x] The importer paragraph in `docs/architecture.md` (`:185-187`) says that the importer also
  applies a diagram edit onto the model the diagram was drawn from.
- [x] `task test:unit` passes.

**Affected Files/Modules:**
- `internal/importer/diagram_edit.go` — new: `ApplyDiagram` and the `diagramEdit` it builds (see
  the planned design)
- `internal/importer/importer.go` — `builder`'s rules, made available to the apply without a second
  copy
- `internal/importer/diagram_edit_test.go` — new: `TestApplyDiagram`, using the test helpers in
  `importer_test.go`, which share its test package
- `docs/architecture.md` — the importer paragraph

**Patterns to Follow:**
- `internal/cli/slices_arrange.go:13-51`
- `internal/arrange/arrange.go:57-71`
- `internal/importer/importer.go:79-123`
- `internal/importer/importer.go:174-311`
- `internal/importer/importer.go:242-254`
- `internal/importer/importer.go:320-409`
- `internal/export/diagram.go:122-136`
- `internal/export/diagram.go:378-406`
- `internal/export/diagram.go:408-455`
- `internal/importer/importer_test.go:23-98`
- `internal/importer/importer_test.go:215-258`
- `internal/importer/importer_test.go:773-991`
- `internal/importer/importer_test.go:1031-1076`
- `internal/formatter/formatter_test.go:57-80`
- `tasks/learnings.md:31-35`
- `tasks/learnings.md:141-145`
- `tasks/learnings.md:221-225`
- `tasks/learnings.md:366-370`
- `tasks/learnings.md:571-575`

**Testable:** Yes

**Certainty:** low — no code in this repo compares two diagrams, or maps a diagram back onto the
model it was exported from. `ImportDiagram` only builds, and its fold only adds. Removing, renaming
across references, reordering and moving are new algorithms.

**Blast radius:** high — a destructive write. The function's output replaces the user's source
text. A construct or comment it drops is gone from the file on the next Save, and nothing in the
app brings it back.

**Verification:** `task test:unit` passes. Then check `emod fmt --check` on the output of each
edit's golden, written to a scratch file.

**Depends on:** None

---

### Task 2: Answer an applied diagram edit through the platform seam

**Behavior:** The browser and the desktop app each give the viewer an `applyDiagram(source,
filename, diagram)` operation. It applies the edited diagram onto the source and runs the full
pipeline on the result. It answers the edited source together with the diagnostics, the diagram
and the `parsed` flag that a parse of that source answers. It also says whether the rewrite changed
the layout of a source that was not in `emod fmt` layout, and how many comments that layout dropped.
Nothing in the viewer calls it yet.

**Acceptance Criteria:**
- [x] `pipeline.ApplyDiagramJSON` takes the envelope `{source, filename, diagram}`. It answers
  `{emod, diagnostics, diagram, parsed, reformatted, dropped_comments}`:
  - `emod` is the source with the edit applied (Task 1) and formatted.
  - `diagnostics`, `diagram` and `parsed` are exactly what `parseEmod` answers for `emod` under the
    same file name.
  - `reformatted` is true only when `emod fmt` would have changed the source given.
  - `dropped_comments` is the number of the source's comments that the `emod fmt` layout does not
    keep. It is 0 whenever `reformatted` is false.
- [x] Deleting a construct that has comments does not count those comments in `dropped_comments`. A
  source that is not fmt-clean but keeps all its comments, such as
  `examples/error_diagnostics_test.emod`, answers `reformatted` true and `dropped_comments` 0.
- [x] A missing file name gives `input.emod`, as a parse does.
- [x] Each of these answers `{error}` with a reason, and each reason is tested: a malformed
  envelope, a missing or empty `source`, a missing `diagram`, a diagram that is not a document, and
  a source that does not parse. The last reason says that the diagram edit cannot be applied
  because the source does not parse.
- [x] Non-ASCII text in labels, descriptions and comments comes back unchanged through the
  envelope and the answer.
- [x] The WASM module registers `applyDiagram`. `platform.browser.js`'s `applyDiagram` sends
  `{source, filename, diagram}` to the worker under that name. It resolves with the answer. It
  rejects with the error's reason, as `exportEmod` does. Before the module is ready it rejects with
  `WASM not ready yet`.
- [x] `ModelService.ApplyDiagram` answers the same document as `pipeline.ApplyDiagramJSON` for the
  same envelope. `platform.desktop.js`'s `applyDiagram` calls it and resolves or rejects as the
  browser does.
- [x] Each seam guard covers the new operation:
  - The contract in `platform.js` names `applyDiagram`, and both implementations satisfy it
    (`viewer.test.js:3674-3711`).
  - The binding-names guard finds `ApplyDiagram` among `ModelService`'s exported methods.
  - The bindings stub answers it.
  - Both runtimes send exactly the keys the apply envelope reads (the pattern at
    `pipeline_test.go:56-67`).
- [x] `parseEmod`, `exportEmod` and every existing seam test behave as before.
- [x] `docs/wasm-architecture.md` names `applyDiagram` among the functions the WASM module registers.
- [x] `task test:unit`, `task test:viewer` and `task build:web` pass.

**Affected Files/Modules:**
- `internal/pipeline/pipeline.go` — `ApplyDiagramJSON`
- `internal/pipeline/pipeline_test.go` — its contract, and the request-keys guard for `applyDiagram`
- `internal/desktop/service.go` — `ModelService.ApplyDiagram`
- `internal/desktop/service_test.go` — its envelope
- `cmd/emod-wasm/main.go` — the `applyDiagram` registration
- `internal/frontend/static/platform.js` — the contract's export list
- `internal/frontend/static/platform.browser.js` — `applyDiagram` over the worker
- `internal/frontend/desktop/platform.desktop.js` — `applyDiagram` over the binding
- `internal/frontend/tests/platform.browser.test.js`, `internal/frontend/tests/platform.desktop.test.js`
  — each implementation's behaviour
- `internal/frontend/tests/bindings-stub.js` — the `ApplyDiagram` binding
- `internal/frontend/tests/viewer.test.js` — the contract list in `'the platform seam has one contract'`
- `docs/wasm-architecture.md` — the registered functions

**Patterns to Follow:**
- `internal/pipeline/pipeline.go:73-124`
- `internal/pipeline/pipeline_test.go:56-67`
- `internal/pipeline/pipeline_test.go:243-282`
- `internal/pipeline/pipeline_test.go:448-466`
- `internal/desktop/service.go:31-37`
- `internal/desktop/service_test.go:112-173`
- `internal/desktop/binding_names_test.go:22-36`
- `cmd/emod-wasm/main.go:10-37`
- `internal/frontend/static/platform.js:1-7`
- `internal/frontend/static/platform.browser.js:125-142`
- `internal/frontend/desktop/platform.desktop.js:16-30`
- `internal/frontend/tests/platform.browser.test.js:42-55`
- `internal/frontend/tests/platform.browser.test.js:138-159`
- `internal/frontend/tests/platform.desktop.test.js:85-126`
- `internal/frontend/tests/bindings-stub.js:9-18`
- `internal/frontend/tests/viewer.test.js:3674-3711`
- `docs/wasm-architecture.md:1-12`
- `tasks/learnings.md:971-975`
- `tasks/learnings.md:989-993`
- `tasks/learnings.md:1007-1011`
- `tasks/learnings.md:1103-1107`
- `tasks/learnings.md:1199-1203`
- `task:1`

**Testable:** Yes

**Certainty:** high — `exportEmod` is already threaded through every one of these files and guards
(`pipeline.go:91-106`, `service.go:31-37`, `main.go:26-37`, `platform.browser.js:132-142`,
`platform.desktop.js:22-30`). This task repeats that shape for one more operation.

**Blast radius:** low — transport only. What the source becomes is decided in Task 1, and when the
panel takes it is decided in Task 3. The seam is used only inside this repository.

**Verification:** `task test:unit`, `task test:viewer` and `task build:web` pass. `task build:desktop`
succeeds where its C toolchain is present, which means the generated bindings export
`ApplyDiagram`.

**Depends on:** Task 1

---

### Task 3: Write diagram edits into the source panel

**Behavior:** This task applies when the diagram on screen was drawn from `.emod` source in the
panel, and the panel still holds that source. In that case every edit made on the canvas
(Decision 6) replaces the panel's text with the drawn source plus that edit. The canvas edits are
the context menu, the detail panel, the keyboard, inline rename, drawing, repointing and deleting
arrows, and dragging a slice to reorder it. The field editor is Task 5 and the drag between slices
is Task 6. The diagram is redrawn in place from the new text, its diagnostics are listed, and the
window is marked modified. Save then writes that text. Visual state never reaches the panel.

**Acceptance Criteria:**
- [x] Every edit that changes the diagram's nodes or edges reaches the panel. The sites that make one
  today were found from where the canvas changes `store.nodes` or `store.edges`. They are:
  - add slice, add command, add event, add flow, delete arrow, and move slice left or right, from the
    context menu;
  - deleting a block, from the detail panel button and from Delete or Backspace;
  - deleting an arrow, from the detail panel and from Delete or Backspace;
  - renaming a context, an aggregate, a slice or a block, committed by Enter or by leaving the box;
  - drawing an arrow from a port;
  - repointing either end of an arrow;
  - dragging a slice to a new place among its siblings.

  Each site has a test that asserts the panel's text afterwards.
- [x] An edit that changes nothing leaves the panel and the marker as they were and calls no
  `applyDiagram`. Examples are a rename to the same label, an abandoned connect or repoint, a
  refused duplicate arrow, a slice dropped back where it was, and move-left on the first slice.
- [x] After an edit, the panel shows the text the apply answered. The diagram is redrawn from that
  text in place, keeping dragged offsets, hidden nodes, pan and zoom as revalidation keeps them. The
  diagnostics panel and the gutter show that text's diagnostics. The data panel stays open or
  collapsed as it was.
- [x] The window is marked modified once a diagram edit reaches the panel. The existing test
  `'says nothing when the context menu adds a node to the diagram'` (`viewer.test.js:1803-1810`) is
  replaced by one that pins the opposite: Save now writes an added node.
- [x] Save after a diagram edit writes the panel's text. A file that arrived with CRLF line endings
  is written with CRLF, and one with LF is written with LF.
- [x] Save pressed while a diagram edit is being applied writes the text with that edit.
- [x] When the apply answers `reformatted`, the bar says the source was rewritten in the `emod fmt`
  layout. When `dropped_comments` is above zero, it also says how many comments the layout dropped.
  The full text is in the hover title. A later save confirmation replaces it. An edit on a source
  that was already fmt-clean shows no such notice.
- [x] A diagram with no `.emod` source behind it takes diagram edits as it does today: the panel does
  not change and no `applyDiagram` is called. This covers diagram JSON pasted into the panel and the
  diagram `emod diagram --serve` hands the page.
- [x] When the panel has an unapplied edit, a diagram edit leaves the panel's text alone and calls no
  `applyDiagram`, as before this story. The panel has an unapplied edit when its text differs from
  the drawn source: while typing pauses, while a parse is in flight, when the text does not parse,
  and when the panel is empty. Task 4 replaces this behaviour.
- [x] An apply whose answer lands after the panel's text changed never replaces that text. The typed
  text revalidates as in US-009.
- [x] A second diagram edit made while the first is being applied leaves the panel holding both
  edits. An older answer that lands after a newer one never replaces the newer one.
- [x] Each of these, arriving while a diagram edit is being applied, keeps that edit:
  - A Render click.
  - A host failure, such as a refused drop or an unreadable or empty file.
  - An open from the dialog, recent files, a drop or a host delivery. It asks the unsaved-edits
    question as if the edit had already reached the panel.
- [x] A diagram edit whose result does not parse (for example a name typed that is not an
  identifier) still writes the result into the panel. The diagnostics list why, and the diagram is
  marked out of date, as when the same text is typed.
- [x] When `applyDiagram` fails, the panel's text does not change, and the diagram keeps the edit
  with the out-of-date mark giving the reason. A failure includes a diagram drawn from a file that
  opened with parse errors.
- [x] After a diagram edit has reached the panel, typing into the panel and pausing redraws the
  diagram from the typed text, as in US-009.
- [x] No gesture that changes only visual state calls `applyDiagram` or changes the panel. The
  gestures are taken from the state outside `store.nodes` and `store.edges` in `createStore` and
  from the toolbar:
  - dragging a block inside its own slice;
  - pan, zoom, fit view and reset layout;
  - hiding and showing in the visibility tree;
  - collapsing and expanding the data panel;
  - opening and closing the minimap, legend and diagnostics panels;
  - selecting and highlighting.

  A diagram edit made after any of them writes the same text as the same edit made without them.
- [x] Every key `viewer.js` reads off the apply's answer is one the Go answer writes. The pattern is
  `pipeline_test.go:162-176`.
- [x] In a real browser with the real pipeline, `examples/specs_hotel.emod` is rendered and a
  command is added from a slice's context menu. The panel then holds exactly the file plus that
  command's declaration, with its comments, invariants, specs, rejections and `type =` lines intact.
  Deleting an arrow from its context menu then removes its line. The `export.spec.js` specs pass
  unchanged.
- [x] The README's source-panel paragraph says that a change made on the diagram is written into the
  source panel, and that Save writes it. It also says that a file not in the `emod fmt` layout takes
  that layout on its first diagram edit, and that the bar says so.
- [x] The README's Save-versus-Export paragraph (`README.md:464-467`) and its "What it does not do
  yet" line (`:554-556`) no longer say that the two are not reconciled. Each claim is pinned the way
  `viewer.test.js:3574-3610` pins the source-panel paragraph.
- [ ] `task test:viewer`, `task test:unit` and `task test:e2e:viewer` pass.

**Affected Files/Modules:**
- `internal/frontend/static/viewer.js` — the drawn source, applying a diagram edit, the redraw in
  place, the panel write, the marker, the bar notice, and ordering against opens, Render, host
  failures, saves and typing
- `internal/frontend/static/ctx-actions.js`, `internal/frontend/static/ui.js`,
  `internal/frontend/static/interaction.js` — each canvas edit site announces a diagram edit on the
  bus, and only when it changed something
- `internal/frontend/static/model.js` — only if the source-backed check needs `sendParse` to say
  which kind of input it read
- `internal/frontend/tests/viewer.test.js` — the platform mock's `applyDiagram`, every criterion
  above, the inverted Save test, and the README guards
- `internal/frontend/tests/ctx-actions.test.js` — only where an action's own contract changes
- `internal/pipeline/pipeline_test.go` — the answer-keys guard for `applyDiagram`
- `e2e-viewer/tests/diagram-edits.spec.js` — new: the real-pipeline checks
- `README.md` — the desktop app section

**Patterns to Follow:**
- `internal/frontend/static/viewer.js:221-295`
- `internal/frontend/static/viewer.js:342-382`
- `internal/frontend/static/viewer.js:396-444`
- `internal/frontend/static/viewer.js:560-616`
- `internal/frontend/static/viewer.js:650-675`
- `internal/frontend/static/viewer.js:84-101`
- `internal/frontend/static/viewer.js:790-813`
- `internal/frontend/static/viewer.js:32-55`
- `internal/frontend/static/model.js:23-33`
- `internal/frontend/static/model.js:135-182`
- `internal/frontend/static/ctx-actions.js:63-87`
- `internal/frontend/static/ui.js:549-561`
- `internal/frontend/static/ui.js:593-603`
- `internal/frontend/static/ui.js:699-727`
- `internal/frontend/static/ui.js:1046-1062`
- `internal/frontend/static/interaction.js:293-388`
- `internal/frontend/tests/viewer.test.js:36-77`
- `internal/frontend/tests/viewer.test.js:246-266`
- `internal/frontend/tests/viewer.test.js:1775-1828`
- `internal/frontend/tests/viewer.test.js:2856-3043`
- `internal/frontend/tests/viewer.test.js:3329-3530`
- `internal/frontend/tests/viewer.test.js:3574-3610`
- `internal/pipeline/pipeline_test.go:162-176`
- `e2e-viewer/tests/export.spec.js:53-115`
- `e2e-viewer/tests/helpers.js:102-120`
- `e2e-viewer/tests/source-panel.spec.js:95-106`
- `README.md:401-421`
- `tasks/learnings.md:905-909`
- `tasks/learnings.md:1007-1011`
- `tasks/learnings.md:1025-1035`
- `tasks/learnings.md:1067-1077`
- `tasks/learnings.md:1127-1131`
- `tasks/learnings.md:1187-1191`
- `tasks/learnings.md:1282-1287`
- `tasks/learnings.md:1309-1328`
- `tasks/learnings.md:1351-1356`
- `task:2`

**Testable:** Yes

**Certainty:** medium — the render orchestration and revalidation already show how an asynchronous
render is ordered and redrawn in place (`viewer.js:221-295`, `:367-382`, `model.js:23-33`). The
variation: no render before this one writes the panel's text from an answer, and no existing render
carries an edit that must survive being overtaken. Today an overtaken render simply stands down.

**Blast radius:** high — a destructive write. This task decides when the user's source text is
replaced. A wrong ordering overwrites typed text, or drops a diagram edit the user believes Save
will write.

**Verification:** `task test:viewer`, `task test:unit` and `task test:e2e:viewer` pass. Then make a
manual check in the desktop build (`task build:desktop`):

1. Open a fmt-clean model with comments.
2. Add a command and rename an event from the canvas.
3. Save.
4. Check that `emod fmt --check` passes on the saved file and that `git diff` shows only those two
   edits.

Repeat with a CRLF copy of the file.

**Depends on:** Task 2

---

### Task 4: Say when the diagram and the source panel disagree, and let the user choose

**Behavior:** The window enters a conflict in two cases: a diagram edit is made while the panel
holds text the diagram was not drawn from, or the panel is typed into while a diagram edit is being
applied. Neither surface is changed. A notice over the canvas says that the diagram and the source
panel disagree, and offers **Keep diagram edits** and **Keep source edits** (Decision 6). Until one
is chosen, nothing redraws the diagram from the panel, and the window counts the diagram edits as
unsaved.

**Acceptance Criteria:**
- [x] A diagram edit made while the panel has an unapplied edit puts the window in conflict. This
  holds whatever the unapplied edit is: typing in the pause, a parse in flight, text that does not
  parse, or an empty panel. Neither the panel's text nor the diagram changes. This replaces Task 3's
  interim behaviour.
- [x] Typing while a diagram edit is being applied puts the window in conflict. The apply's answer is
  not written into the panel.
- [x] The notice says that the diagram and the source panel disagree, and offers the two choices. In
  a real browser, both the notice and its buttons can be reached with the data panel collapsed
  and with it open. This is checked with `elementFromPoint`. While the notice is shown, it takes the
  place of the out-of-date notice.
- [x] While the conflict stands:
  - A pause in typing lists the panel's diagnostics and updates the gutter, but leaves the diagram
    as edited.
  - Further diagram edits are kept, and do not change the panel.
  - The window is marked modified, even when the panel's text matches the file.
- [x] **Keep diagram edits** applies every diagram edit made since the drawing onto the drawn source.
  The result replaces the panel's text, and the diagram is redrawn from it in place. The notice
  closes, and the bar notice from Task 3 appears when the apply answers `reformatted`. When the apply
  fails, the reason is shown and the conflict stays.
- [x] **Keep source edits** redraws the diagram in place from the panel's text, and does not touch
  that text. When the text does not parse, or the panel is empty, the diagram is redrawn from the
  drawn source and keeps the out-of-date mark. The notice closes.
- [x] A Render click during a conflict resolves it as **Keep source edits** does.
- [x] Save during a conflict writes exactly the panel's text, and leaves the window marked modified
  with the notice still shown.
- [x] Closing, quitting or opening another model during a conflict asks Save, Discard or Cancel:
  - **Save** writes the panel and leaves the window open, with the conflict still standing.
  - **Discard** goes ahead.
  - **Cancel** does nothing.

  Once another model is open, it has no conflict.
- [x] A resolution chosen while something else is in flight keeps to the ordering rules of Task 3:
  Save waits for it, and an open asks as if it had landed.
- [x] The README's source-panel paragraph describes the conflict and its two resolutions. It no
  longer says that typing replaces changes made on the diagram (`README.md:418-419`). The
  `viewer.test.js:3574-3610` pattern pins this.
- [x] In a real browser with the real pipeline, typing into the panel and adding a command from the
  context menu before typing pauses shows the notice. **Keep diagram edits** then leaves the panel
  holding the command and not the typed text. Doing the same and choosing **Keep source edits** leaves
  the panel holding the typed text and the diagram drawn from it.
- [ ] `task test:viewer` and `task test:e2e:viewer` pass.

**Affected Files/Modules:**
- `internal/frontend/static/viewer.js` — when a conflict starts, how it holds off revalidation, its
  two resolutions, Render during it, and its place in the unsaved-edits question
- `internal/frontend/static/ui.js` — showing and hiding the conflict notice
- `internal/frontend/static/store.js` — the conflict state and the notice's DOM reference
- `internal/frontend/static/viewer.html` — the notice and its two buttons, beside `#stale-notice`
- `internal/frontend/tests/viewer.test.js` — every criterion above, the new elements in
  `createRequiredElements`, and the README guard
- `e2e-viewer/tests/diagram-edits.spec.js` — the notice's reach and both resolutions
- `README.md` — the source-panel paragraph

**Patterns to Follow:**
- `internal/frontend/static/ui.js:370-385`
- `internal/frontend/static/viewer.html:220-245`
- `internal/frontend/static/viewer.html:1320`
- `internal/frontend/static/viewer.js:221-295`
- `internal/frontend/static/viewer.js:367-382`
- `internal/frontend/static/viewer.js:396-449`
- `internal/frontend/static/viewer.js:650-675`
- `internal/frontend/static/store.js:1-67`
- `internal/frontend/tests/viewer.test.js:79-125`
- `internal/frontend/tests/viewer.test.js:1831-1985`
- `internal/frontend/tests/viewer.test.js:2241-2320`
- `internal/frontend/tests/viewer.test.js:3044-3530`
- `internal/frontend/tests/viewer.test.js:3616-3642`
- `e2e-viewer/tests/source-panel.spec.js:10-36`
- `README.md:401-421`
- `tasks/learnings.md:977-981`
- `tasks/learnings.md:1067-1077`
- `tasks/learnings.md:1309-1314`
- `tasks/learnings.md:1351-1356`
- `task:3`

**Testable:** Yes

**Certainty:** low — nothing in the viewer holds two diverged versions and asks the user to choose
between them. This task decides how a standing conflict holds off revalidation, how it counts toward
Save and the unsaved-edits question, and what Render does during it.

**Blast radius:** high — a destructive write. Each resolution discards one surface's edits by
design. A wrong trigger, or a revalidation that slips past the conflict, discards edits with no
choice offered.

**Verification:** `task test:viewer` and `task test:e2e:viewer` pass. Then make a manual check in
the desktop build:

1. Raise a conflict.
2. Try Close, choose Save, and check that the window stays open with the notice.
3. Resolve each way, and check what Save writes.

**Depends on:** Task 3

---

### Task 5: Write field editor edits into the source panel without losing what is typed

**Behavior:** Changing a command's or an event's field in the field editor reaches the panel as
every other diagram edit does. This covers renaming a field, retyping it, setting its modifier and
deleting it. The editor commits a value when its input loses focus, usually because focus moves to
the next input of the same editor. The redraw that follows must not lose the editor, or what is
typed into it.

**Acceptance Criteria:**
- [x] Changing a field's name, type or modifier and leaving the input writes the change into the
  panel. It goes through the same path as Task 3, and through the conflict of Task 4 when the panel
  has an unapplied edit.
- [x] Deleting a field with its × button writes the change into the panel.
- [x] A row added with **+ Add Field** changes nothing in the panel until the row has a name. Leaving
  its name input with a name typed writes the field.
- [x] Leaving an input whose value did not change calls no `applyDiagram` and changes nothing.
- [x] Nothing typed into the field editor is lost to a redraw. A user can tab from one field's input
  to the next and type there. After the first edit reaches the panel, the editor is still open on
  the same element, with focus in the input it moved to and the text typed there since. When focus
  leaves the second input, both edits are in the panel.
- [x] The field editor's other behaviour is as before. The existing `'viewer field editor'` tests
  (`viewer.test.js:846-895`) and `detail-panel.test.js` pass.
- [x] In a real browser with the real pipeline, the field editor is opened on a command. One field's
  type is changed, Tab moves to its modifier, and a modifier is typed there. The panel then holds
  both changes.
- [ ] `task test:viewer` and `task test:e2e:viewer` pass.

**Affected Files/Modules:**
- `internal/frontend/static/ui.js` — the field editor's inputs, × button and **+ Add Field** announce
  a diagram edit when the field changed. The editor survives the redraw that follows
- `internal/frontend/static/viewer.js` — only if keeping the editor across the in-place redraw needs
  the orchestration
- `internal/frontend/tests/viewer.test.js` — every criterion above
- `e2e-viewer/tests/diagram-edits.spec.js` — the tab-and-type check

**Patterns to Follow:**
- `internal/frontend/static/ui.js:405-561`
- `internal/frontend/static/viewer.js:121-129`
- `internal/frontend/tests/viewer.test.js:846-895`
- `internal/frontend/tests/detail-panel.test.js:1-40`
- `tasks/learnings.md:576-580`
- `task:3`
- `task:4`

**Testable:** Yes

**Certainty:** medium — the field editor (`ui.js:515-547`) and Task 3's apply path are the
precedent. The variation: this edit is committed on blur while focus moves into another input of the
same editor. A render rebuilds that editor today (`viewer.js:121-129`), which no other site has to
survive.

**Blast radius:** low — when this task goes wrong, a field edit or the text being typed is lost
before it reaches the source. Existing source is not rewritten wrongly. What the edit does to the
source is Task 1's.

**Verification:** `task test:viewer` and `task test:e2e:viewer` pass. Then make a manual check in
the desktop build, because WebKit moves focus the way the e2e checks only in Chromium.

**Depends on:** Task 3, Task 4

---

### Task 6: Move a block to another slice by dragging it there

**Behavior:** A command, event, view, automation, translation or trigger released with its centre
over another slice moves to that slice (Decision 5). On a diagram drawn from source, the source then
declares it there, through Task 3's path. On a diagram with no source, the move changes the diagram
only, and Export writes it. A block released anywhere else keeps today's behaviour: it stays in its
slice as a visual offset. A slice refuses a second trigger, and the event a translation nests moves
only with its translation.

**Acceptance Criteria:**
- [ ] A block of each kind the viewer drags, released with its centre over another slice, moves to
  that slice. The slice can be left or right of the block's own, in the same or another aggregate or
  context. The diagram draws the block inside its new slice, at its laid-out place, with no offset
  from the old slice. On a diagram drawn from source, the panel then declares it in that slice, as
  Task 1 places it.
- [ ] A block released over its own slice, or over no slice, keeps today's behaviour, and that
  behaviour calls no `applyDiagram`: the slice grows to hold it, it is held at the slice's top and
  left edges, and its offset accumulates and survives a redraw. `drag-containment.test.js` still
  pins this, using drags that stay within the slice. A test whose release lands over another slice
  now pins the move (`tasks/learnings.md:905-909`).
- [ ] While a block is dragged, it is held inside its own slice only while its centre is not over
  another slice. So a block can be dragged across its slice's left edge onto the slice before it.
- [ ] A trigger released over a slice that already has a trigger goes back to where it was before the
  drag. The panel does not change and no `applyDiagram` is called. The same holds for the node of an
  event a translation nests, released over another slice. A translation released over another slice
  moves with its nested event.
- [ ] A one-finger touch drag behaves as the mouse drag does.
- [ ] A slice dragged over another context or aggregate is still only reordered among its own
  siblings.
- [ ] On a diagram with no source behind it, the move changes the diagram only, and Export .emod
  writes the block in its new slice.
- [ ] In a real browser with the real pipeline, a command is dragged onto the next slice. The panel
  then declares it in that slice and the block is drawn inside that slice.
- [ ] The README's source-panel paragraph says that dragging a block onto another slice moves it
  there, in the diagram and in the source. The `viewer.test.js:3574-3610` pattern pins this.
- [ ] `task test:viewer` and `task test:e2e:viewer` pass.

**Affected Files/Modules:**
- `internal/frontend/static/interaction.js` — which slice a release lands over, the hold inside the
  slice, the move, the refusals, and the touch path
- `internal/frontend/tests/drag-containment.test.js` — the move, the refusals, and the in-slice
  behaviour that still holds
- `internal/frontend/tests/viewer.test.js` — the move reaching the panel, and the refusals leaving it
  alone
- `e2e-viewer/tests/diagram-edits.spec.js` — the real-pipeline move
- `README.md` — the source-panel paragraph

**Patterns to Follow:**
- `internal/frontend/static/interaction.js:146-158`
- `internal/frontend/static/interaction.js:200-231`
- `internal/frontend/static/interaction.js:345-388`
- `internal/frontend/static/interaction.js:645-680`
- `internal/frontend/static/interaction.js:985-1005`
- `internal/frontend/static/layout.js:289-301`
- `internal/frontend/tests/drag-containment.test.js:1-200`
- `e2e-viewer/tests/helpers.js:211-216`
- `e2e-viewer/tests/touch-input.js`
- `README.md:401-421`
- `tasks/learnings.md:905-909`
- `task:1`
- `task:3`

**Testable:** Yes

**Certainty:** medium — `tryReorderSliceOnDrop` (`interaction.js:345-388`) already decides a
structural change from where a drop lands against the laid-out positions. The variation: this drop
changes a block's parent rather than its order. The hold inside the slice (`interaction.js:146-158`)
has to let go over another slice, which no drag does today.

**Blast radius:** low — the gesture only changes which slice a node names. What that does to the
source, including refusing a move a slice cannot hold, is Task 1's.

**Verification:** `task test:viewer` and `task test:e2e:viewer` pass. Then make a manual check in
the desktop build: drag blocks onto slices in another context, save, and check the saved file with
`emod fmt --check`.

**Depends on:** Task 1, Task 3

---

## Summary

- **Six tasks**, ordered dependency-first, then risk-first:
  - Task 1 is the least certain piece and the one with the highest cost when wrong, and every other
    task builds on it.
  - Task 2 is the routine transport that has to exist before the viewer can call anything.
  - Task 3 is the core viewer behaviour.
  - Task 4 is the least certain viewer piece, so it comes next.
  - Tasks 5 and 6 add one edit surface each, onto a path that is settled by then.
- **Assessments:**
  - Two `low` (Task 1, Task 4), three `medium` (Task 3, Task 5, Task 6), one `high` (Task 2).
  - Blast radius is `high` on the three tasks that decide what the user's source text becomes or
    when it is replaced (Task 1, Task 3, Task 4).
  - Plan two or three audit rounds for Tasks 3 and 4 (`tasks/learnings.md:1351-1356`).
- **Story criteria:** all six are covered (see the table under Story Reference). Nothing is
  deferred. The **Deferred** list under Boundaries contains only behaviour the story does not ask
  for.
- **Additions to the user's decisions**, which the caller can overrule:
  - The bar names the number of comments the `emod fmt` layout dropped (Decision 2).
  - Flow, rejection and spec lines stay in the slice that states them when a block moves
    (Decision 5).
  - The five holes closed in the conflict design: the guard counts unapplied diagram edits, Save
    during a conflict, Render during a conflict, an open during an apply, and a drawn source that
    did not parse (Decision 6).
  - A field row with no name is left out (Decision 8).

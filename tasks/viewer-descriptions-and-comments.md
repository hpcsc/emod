# Descriptions and Comments in the Viewer

## Progress
- [x] Task 1: Carry every construct's description on its diagram node and read it back
- [x] Task 2: Carry a construct's comments on its diagram node and read them back
- [ ] Task 3: Show a node's description in the detail panel
- [ ] Task 4: Render context and aggregate descriptions on their headers
- [ ] Task 5: Show a node's description in the hover tooltip, including nodes with no fields
- [ ] Task 6: Mark a described node and slice, and read the description off the marker
- [ ] Task 7: Mark a commented node and slice, and read the comments off the marker
- [ ] Task 8: Mark commented context and aggregate headers

## Story Reference

`user-stories/viewer-descriptions-and-comments.md` — US-001 through US-006. The story's checkboxes are the contract; every criterion below traces to one of them.

## Codebase Context

**The wire.** The viewer is fed the *diagram* document, not the model document: `export.ExportDiagramJSON` (`internal/export/diagram.go`) via `wasm.RunPipelineExportDiagram` (`internal/wasm/pipeline.go:50`). `jsonDiagramNode` (`internal/export/diagram.go:20-39`) carries `id`, `type`, `label`, `parentId`, `fields`, `position` and per-type metadata — no `description`, no `comments`. The model document (`internal/export/json.go`) already serializes both and is the reference for field naming (`jsonComment` at `:21-24`, `convertComments` at `:309`).

**The save path.** The viewer's export button hands its nodes and edges back through `wasm.ExportEmod` → `importer.ImportDiagram` → `formatter.Format`. `internal/viewer/static/emod-export.js` is a pass-through and `model.js:setModelData` stores `data.nodes` verbatim, so any key the exporter writes reaches the save unchanged — but `diagramNode` (`internal/importer/importer.go:21-38`) must read it back or it is silently dropped.

**Two carriers per event.** A translation's nested event travels twice: as the `event` object on the translation node (`convertEventToDiagram`, `internal/export/diagram.go:430`) and as a standalone event node (`:239-251`). The importer rebuilds `trans.Event` from the nested object only (`internal/importer/importer.go:217-224`) and discards the standalone node (`:173-176`). `jsonDiagramEvent` already carries `comments`; the importer's `diagramEvent` (`:40-45`) reads neither `comments` nor `description`.

**Two slice homes.** `convertModelToDiagram` builds a slice node under an aggregate (`internal/export/diagram.go:322-329`) and again directly under a context (`:338-345`). Both branches need the new fields.

**An existing test blocks US-001.** `internal/export/export_test.go:3133` — "a described model still produces a document free of prose" — asserts `descriptionsAnywhere` is empty over the diagram document. Task 1 must replace it.

**The viewer.** `showDetailPanel` (`internal/viewer/static/ui.js:284`) builds header → Fields → per-type tables → Source. `showTooltip` (`:7-22`) renders the custom tooltip; the block-hover guard at `:664` reads `if (!node || !node.fields || node.fields.length === 0) return;`. `renderer.js` draws the context name left-anchored at `cp.x + 16` (`:212`, header height `L.swimlaneHdr = 44`), the aggregate label at `row.x + 16` (`:234`, row height `L.aggLabelH = 22`, `row.w` floored at 100px, with the overlap note at `:230-232`) and the slice name **centred** at `sp.x + sp.w / 2` (`:262`, `L.sliceHdrH = 28`). The marker convention already exists: `clockMarking` (`:5`), drawn through `centeredText` for a scheduled automation (`:176-181`), with `svgTitle` at `:44`. `layout.js` exports a cached `labelWidth()` (`:16-23`) measuring through a hidden SVG at font-size 13 and adding 16px of padding.

**Test surfaces.** Go tests are behind build tags (`task test:unit` → `go test -tags unit`); a plain `go test ./...` runs nothing. Viewer tests are vitest on jsdom under `internal/viewer/tests/`, run by `task test:viewer` (not part of `task test:unit`), with `installSVGGeometry()` from `./svg-env.js` at module scope and a dynamic `await import('../static/ui.js')` where the module touches geometry. jsdom's shimmed `getComputedTextLength` is 7px per character.

## Tasks

### Task 1: Carry every construct's description on its diagram node and read it back

**Behavior:** A diagram node carries its construct's `description`, and a round trip through `ImportDiagram` and `formatter.Format` re-emits every one of them, so a visual edit no longer deletes prose the file had.

**Acceptance Criteria:**
- [ ] `jsonDiagramNode` and `jsonDiagramEvent` each carry an optional `description` key, and exporting `test.DescribedHotelReservationModel(t)` yields the fixture's own description text on the actor, context, aggregate, every slice, the trigger, every command, every event, every view, the automation, the translation and the translation's nested `event` object — read back as one map from node label to description and compared in a single `require.Equal` against a hand-transcribed expectation
- [ ] `descriptionsAnywhere` over the diagram document of `test.HotelReservationModel(t)` finds nothing while the same walk over `test.DescribedHotelReservationModel(t)`'s document returns one entry per described construct, both asserted in one subtest — the key is omitted rather than emitted empty, so an undescribed model marshals the bytes it did before
- [ ] A model declaring one described slice under an aggregate and a second described slice directly on a context exports both slice nodes carrying their own description text, compared as one map, so neither branch of `convertModelToDiagram` can be left behind
- [ ] `internal/export/export_test.go` holds no subtest asserting that a described model's diagram document is free of descriptions
- [ ] `cli.RunExport(path, "diagram-json")` over a file holding `test.DescribedHotelReservation` prints a document whose context node, a slice node and a command node each carry the fixture's description text for that construct
- [ ] One subtest runs `importExported` over both `test.DescribedHotelReservationModel(t)` and `test.HotelReservationModel(t)`: the described model's round-tripped `formatter.Format` output equals the fixture formatted with its comments stripped (descriptions intact), and the undescribed model's still equals its own comment-stripped baseline
- [ ] `internal/cue/schema.cue` and the `json*` model-document types in `internal/export/json.go` are unchanged — `git diff --name-only` names neither file

**Affected Files/Modules:**
- `internal/export/diagram.go` — `description` on `jsonDiagramNode` and `jsonDiagramEvent`; populated for the actor, context, aggregate, both slice branches, and every construct `collectSliceNodes` visits including the standalone event node built for a translation's nested event
- `internal/importer/importer.go` — `description` on `diagramNode` and `diagramEvent`; read back in the actor arm of `ImportDiagram`, in `buildContext` and in `buildSlice`
- `internal/export/export_test.go` — replaces the "free of prose" subtest; adds the described read-back, the undescribed key-absence leaf and the two-slice-homes guard
- `internal/importer/importer_test.go` — round-trip leaf over the shared described and undescribed fixtures
- `internal/cli/export_test.go` — a `diagram-json` leaf over the described fixture

**Patterns to Follow:**
- Field order: copy a `json*` sibling in `internal/export/json.go:26-52`, which opens with the name, then `description`, then the positions, then `comments` — not the ordering in `internal/cue/schema.cue`. `tasks/learnings.md` "JSON and CUE order their document keys differently — do not mirror one struct into the other".
- `tasks/learnings.md` "The viewer's save path is `importer.ImportDiagram`, so a diagram-node field owes a read-back" — the exporter tag and the importer tag move in this one change, or a save silently drops the value.
- `tasks/learnings.md` "A slice has two homes, and much of the repo still walks only one" — the two slice-node sites are `internal/export/diagram.go:322-329` and `:338-345`.
- `tasks/learnings.md` "A new exported field must land in JSON, CUE and `schema.cue` in the same change" does **not** apply here: the diagram document is deliberately uncoupled from the model export and `jsonDiagramEvent` is a deliberate fork of `jsonEvent`. Do not re-merge them and do not touch `internal/cue/schema.cue`.
- `tasks/learnings.md` "Additive output changes owe a byte-identical receipt for models that do not use the feature" — the receipt here is the key-absence leaf over `test.HotelReservationModel`, which is byte-identity given `omitempty`.
- `tasks/learnings.md` "urfave/cli v2 discards every flag written after the file argument" — any manual receipt must be spelled `emod export -f diagram-json <file>`, flag first; the test calls `cli.RunExport(path, "diagram-json")` directly and sidesteps it.
- Read the document back with the walkers already in `internal/export/export_test.go`: `diagramDocOf` (`:4862`), `descriptionsAnywhere` (`:4959`), `eachObject` (`:4741`), `nodesOfType`/`statedUnder`. `tasks/learnings.md` "Read a decoded export document back with `objectsUnder`/`statedUnder`".
- Round-trip helpers already exist: `importExported` (`internal/importer/importer_test.go:35`) and `stripComments` (`:748`).
- `tasks/learnings.md` "A differential receipt must first prove the twin actually differs" — the described half of each paired leaf is what makes the undescribed half mean something; keep both in the same subtest.
- Go test conventions: one `Test{TypeName}` umbrella per type, `t.Run` groups per operation, scenario names as full sentences, `github.com/stretchr/testify/require`, fresh fixtures per leaf. Prefer one structural `require.Equal` over a composite value to a chain of per-field assertions.
- `~/.config/ai/guidelines/comments.md` — add a comment only when you can name the wrong conclusion a reader would otherwise draw.

**Testable:** Yes — through `export.ExportDiagramJSON`, `importer.ImportDiagram`, `formatter.Format` and `cli.RunExport`, all exported.

**Verification:** `mise exec -- task test:unit` passes; `git diff --name-only` names neither `internal/cue/schema.cue` nor `internal/export/json.go`.

**Depends on:** None

---

### Task 2: Carry a construct's comments on its diagram node and read them back

**Behavior:** A diagram node carries the comments attached to its construct, and a round trip re-emits them above the construct they belong to, so a visual edit no longer deletes the notes a team left in the file.

**Acceptance Criteria:**
- [ ] `jsonDiagramNode` carries an optional `comments` key holding each comment's text and source position, built with the existing `convertComments`, populated for the actor, context, aggregate, both slice branches, the trigger, commands, events, views, automations, translations and the standalone event node built for a translation's nested event
- [ ] Exporting a model that comments a construct of every one of those kinds, in both slice homes, yields as one map from node label to comment texts exactly the comments each construct was given, while the same walk over `test.HotelReservationModel(t)`'s document finds no `comments` key on any node — both asserted in one subtest, so an uncommented model still marshals the bytes it did before
- [ ] `formatter.Format(importExported(t, model))` for a canonical `.emod` source that comments a context, an aggregate, a slice in each home, a trigger, a command, an event, a view, an automation and a translation reproduces that source byte for byte, every comment back above its own construct
- [ ] Removing a commented construct's node from a document before importing drops that construct's comments with it: the formatted result contains neither the construct nor any of its comment texts, while the sibling construct declared beside it keeps its own
- [ ] The `all_patterns.emod` and `multi_context.emod` round-trip leaves compare against a baseline that strips only the comments no diagram node can carry — the model's, an invariant's, a spec's, a flow's and a `decides_on` clause's — and each leaf first requires that baseline to still contain a construct comment (`# Slice 1: Command Pattern`), so an over-stripping helper cannot make the comparison vacuous
- [ ] The helper that produces that baseline is named for the comments it strips rather than for stripping all of them, and `ImportDiagram`'s doc comment no longer states that diagram JSON carries no comments
- [ ] `internal/cue/schema.cue` and the `json*` model-document types in `internal/export/json.go` are unchanged — `git diff --name-only` names neither file

**Affected Files/Modules:**
- `internal/export/diagram.go` — `comments` on `jsonDiagramNode`, filled through `convertComments` at every node-building site
- `internal/importer/importer.go` — a comment type on `diagramNode` and on `diagramEvent`, converted back to `[]*ast.Comment`; the package/`ImportDiagram` doc comments
- `internal/export/export_test.go` — the commented read-back, the uncommented key-absence half, and a comment walker built on `eachObject` beside `descriptionsAnywhere`
- `internal/importer/importer_test.go` — the narrowed strip helper, the canonical commented round-trip source, and the deleted-node leaf

**Patterns to Follow:**
- Reuse `jsonComment` and `convertComments` (`internal/export/json.go:21-24`, `:309-318`); `jsonDiagramEvent` already declares `Comments` (`internal/export/diagram.go:60`) — the gap is that the importer's `diagramEvent` never reads it.
- `ast.Comment.Text` keeps its leading `#` (`internal/formatter/formatter.go:69-73` writes it verbatim, and `internal/export/export_test.go:569` expects `"# Invariant comment"`). Carry it through unchanged; stripping the `#` is the viewer's job in Task 7.
- The importer discards every position it is given (`diagramNode` carries no `position` key at all), so the comment type it reads needs `text` only — matching how `convertFields` drops everything but name, type and modifier.
- `tasks/learnings.md` "Name an extracted helper after the contract its callers rely on" — the renamed strip helper must say which comments it removes.
- `tasks/learnings.md` "A `require.NotEqual` on a stripped twin is satisfiable without stripping anything" — pair the uncommented half with a positive check that the commented half really carries the comments.
- `internal/importer/importer_test.go:113-167` is the shape for a canonical round-trip source: the source is already in `emod fmt` form and is compared against itself. `tasks/learnings.md` "Formatter output always begins with `emod N`" — the source needs the version header, and no model-level comment, since no node carries one.
- `tasks/learnings.md` "The viewer's save path is `importer.ImportDiagram`, so a diagram-node field owes a read-back" and "A slice has two homes" apply here exactly as in Task 1.
- Go test conventions as in Task 1; `~/.config/ai/guidelines/comments.md`.

**Testable:** Yes — through `export.ExportDiagramJSON`, `importer.ImportDiagram` and `formatter.Format`.

**Verification:** `mise exec -- task test:unit` passes; `git diff --name-only` names neither `internal/cue/schema.cue` nor `internal/export/json.go`.

**Depends on:** Task 1

---

### Task 3: Show a node's description in the detail panel

**Behavior:** Selecting a node shows its description in a labelled block above the Fields section, so a reader gets the full text without hovering or opening the source.

**Acceptance Criteria:**
- [ ] For a command node carrying a description and fields, the panel's section titles read in document order `['Description', 'Fields', 'Source']`, and the description block's text equals the node's `description` in full, with no truncation, ellipsis or substring applied
- [ ] For the same node with its `description` removed, and again with `description` set to the empty string, the section titles read `['Fields', 'Source']` — no description block and no empty placeholder
- [ ] One `it.each` over `trigger`, `view`, `automation`, `translation`, `context`, `aggregate` and `slice` nodes, none of them carrying `fields`, shows each one's own description text in the block, so a node type with no Fields section still gets it
- [ ] A description containing `<b>&</b>` shows as text: the block's `textContent` holds the raw string and `dpContent.querySelector('b')` is null
- [ ] The stylesheet rule for the description block sets neither `white-space: nowrap` nor `text-overflow`, so the text wraps over as many lines as it needs

**Affected Files/Modules:**
- `internal/viewer/static/ui.js` — `showDetailPanel` (`:284`), a description block emitted after the header and delete button and before the Fields section
- `internal/viewer/static/viewer.html` — a `.dp-description` rule beside the existing `.dp-section` (`:759`) and `.dp-source` (`:805`) rules
- `internal/viewer/tests/detail-panel.test.js` — new `describe` block for the description section

**Patterns to Follow:**
- Follow the section shape already in `showDetailPanel`: `<div class="dp-section">` wrapping a `<div class="dp-section-title">`, exactly as the Trigger, Automation, Translation and Source sections do (`internal/viewer/static/ui.js:331-382`).
- Every value that reaches the panel goes through `Renderer.esc` — see `:293` and every row builder below it.
- `internal/viewer/tests/detail-panel.test.js` is the spec home and the shape to copy: `installSVGGeometry()` at module scope, `const { UI } = await import('../static/ui.js')`, a per-test `createStore()`, per-type node factories, and a reader helper (`shownRows`) that projects the DOM into plain data before asserting. Add a section-title reader in the same style rather than asserting on `innerHTML`.
- `tasks/learnings.md` "`internal/viewer/static` is a display surface with its own vitest harness" — a node key no section names is invisible however faithfully the Go pipeline carries it; and restructuring `showDetailPanel` belongs in its own commit, so keep this change additive.
- `~/.config/ai/guidelines/javascript/testing-patterns.md` "Unit of Behavior" and "Test Behavior, Not DOM Structure": assert the text a reader sees and the order of the sections, not the tag names or class strings that produce them.
- `~/.config/ai/guidelines/testing/caller-patterns.md` "How to Identify the Caller" and "Quick Reference" — this is the **UI** pattern (input from the user, output read by the user): assert visible content, not the HTML structure or the serialization the panel was built from.
- `~/.config/ai/guidelines/comments.md`.

**Testable:** Yes — `UI.showDetailPanel` is exported (`internal/viewer/static/ui.js:970`).

**Verification:** `mise exec -- task test:viewer` passes.

**Depends on:** Task 1

---

### Task 4: Render context and aggregate descriptions on their headers

**Behavior:** A context's and an aggregate's description reads off the diagram beside its name, subordinate to it, without either header growing and without an over-long aggregate description painting over its neighbour.

**Acceptance Criteria:**
- [ ] A context node carrying a description draws a second `<text>` inside its swimlane header holding that description, with an `x` greater than the context name's `x`, while a context in the same render carrying none draws only its name
- [ ] The description `<text>` in the context header has a smaller `font-size` and a different `fill` than the context name `<text>` in the same header, so it reads as subordinate
- [ ] An aggregate node carrying a description draws its description inside its own label row, with an `x` greater than the aggregate name's `x`, while an aggregate in the same render carrying none draws only its name
- [ ] With two aggregates side by side, an over-long description on the first is truncated to end with `…` and its right edge — `x + Layout.labelWidth(textContent) - 16` — sits at or left of the second aggregate's row `x`; in the same render a short description on the second aggregate is drawn whole, with no `…`
- [ ] Rendering one node set twice, once with descriptions on the context and aggregates and once without, produces identical `x`, `y`, `width` and `height` on every `.ctx-header`, `.agg-row`, `.agg-area`, `.slice-box` and `.slice-header` rect and identical `x`/`y` on the context and aggregate name `<text>` elements; the described render differs only by the extra description `<text>` elements
- [ ] The slice header is untouched: neither render draws a description in `.slice-header`, and the slice name `<text>` keeps `text-anchor="middle"` at the same `x` and `y` in both
- [ ] `git diff --name-only` does not name `internal/viewer/static/config.js` — no layout constant moves

**Affected Files/Modules:**
- `internal/viewer/static/renderer.js` — the context header text (`:212`) and the aggregate label pass (`:233-236`)
- `internal/viewer/tests/renderer.test.js` — new `describe` blocks under `Renderer.buildSVG`

**Patterns to Follow:**
- Reuse `Layout.labelWidth` (`internal/viewer/static/layout.js:16-23`, already exported at `:350`) both to place the description after the name and to measure truncation candidates. It measures at font-size 13 and adds 16px of padding, so used on a smaller description it over-estimates — that errs toward truncating early, which is the safe direction. The criterion is non-overlap, not maximal fit.
- Keep the two-pass ordering at `internal/viewer/static/renderer.js:226-236`: every `.agg-row` rect goes down before any label text. The description belongs in the label pass, after the name. The comment at `:230-232` explains why, and it is the hazard the truncation criterion protects.
- `internal/viewer/tests/renderer.test.js` is the spec home: `twoAggregates()` (`:40`) and `narrowAndWideContexts()` (`:54`) are the fixtures for side-by-side rows, `labelFor`/`rowFor` (`:210`, `:214`) select by `data-agg-id`, `numeric` (`:255`) reads an attribute as a number, and `drawnText`/`drawnLabels` (`:224-230`) walk `<text>` elements rather than `textContent`. The "draws every row before any label" leaf (`:284`) is the model for asserting paint order.
- `tasks/learnings.md` "A viewer leaf must be able to fail only for the paint its name blames" — the subordinate-styling leaf compares the two `<text>` elements in one header against each other; do not restate a hex value or a font size the change does not own.
- `tasks/learnings.md` "The viewer shows a node twice — the canvas box and the detail panel" — assertions on a drawn label walk the group's `<text>` elements, because `textContent` folds any `<title>` in.
- Slice descriptions stay out of the slice header on purpose (US-005 Context): the slice name is centred and the header width is derived from the slice's contents through `rowSpanWidth`, so inline text there would shift every slice name off-centre and force `L.sliceHdrH`/`L.sliceTopPad` and every computed position to move.
- `~/.config/ai/guidelines/testing/caller-patterns.md` Quick Reference — **UI** pattern: assert what the reader sees and where it sits relative to the other marks, not the coordinate arithmetic that produced it.
- `~/.config/ai/guidelines/comments.md`.

**Testable:** Yes — `Renderer.buildSVG` and `Layout.computeLayout` are exported and already driven by `render()` in `renderer.test.js:204`.

**Verification:** `mise exec -- task test:viewer` passes; `git diff --name-only` does not name `internal/viewer/static/config.js`.

**Depends on:** Task 1

---

### Task 5: Show a node's description in the hover tooltip, including nodes with no fields

**Behavior:** Hovering a block shows its description in the viewer's own tooltip, and a node that has a description but no fields gets a tooltip where today none appears at all.

**Acceptance Criteria:**
- [ ] After `UI.initDelegation(store)`, dispatching `pointerover` on the group of a node carrying a description and no `fields` displays the tooltip and its text holds that description; dispatching the same event on a sibling group whose node has neither fields nor a description leaves the tooltip hidden — both in one leaf
- [ ] Hovering a node carrying both a description and two fields shows the description text and one row per field in the same tooltip
- [ ] Hovering a node carrying fields and no description shows the field rows and no description block
- [ ] A description containing `<b>&</b>` appears in the tooltip as text: `tooltip.textContent` holds the raw string and `tooltip.querySelector('b')` is null
- [ ] Hovering the node whose id is `store.interaction.selectedNodeId` still shows no tooltip, and a node with no fields shows no empty field table

**Affected Files/Modules:**
- `internal/viewer/static/ui.js` — `showTooltip` (`:7-22`) gains a description block and emits the fields table only when the node has fields; the block-hover guard at `:664` widens from "has fields" to "has fields or a description"
- `internal/viewer/static/viewer.html` — a tooltip description rule beside the existing `#tooltip .tt-header` (`:623`) rules
- `internal/viewer/tests/tooltip.test.js` — new spec file for the tooltip family

**Patterns to Follow:**
- `showTooltip` already opens with `<div class="tt-header">` and escapes through `Renderer.esc`; keep both.
- The new spec file takes the name of the surface it covers, as `detail-panel.test.js` does for `showDetailPanel` — `tasks/learnings.md` "A `_test.go` file always carries the `Test…` umbrella for the name it wears" is the same convention on the JS side.
- Copy the harness from `internal/viewer/tests/detail-panel.test.js:1-20`: `installSVGGeometry()` at module scope, `const { UI } = await import('../static/ui.js')`, a `createStore()` per test building the DOM it needs — here `dom.tooltip` and `dom.svg` — and `document.body.innerHTML = ''` in `beforeEach`.
- Drive the hover through the real delegation (`UI.initDelegation(store)` then a dispatched `pointerover` on an element inside a `.diagram-node[data-node-id]` group) rather than calling `showTooltip` directly, so the widened guard is what the leaf exercises. `~/.config/ai/guidelines/javascript/testing-patterns.md` "Event Delegation".
- `~/.config/ai/guidelines/javascript/testing-patterns.md` "Unit of Behavior" — one behaviour per `it`, named as a sentence about what the reader observes.
- `~/.config/ai/guidelines/testing/caller-patterns.md` Quick Reference — **UI** pattern: assert the visible tooltip content and whether it is shown, not the HTML the builder assembled.
- `~/.config/ai/guidelines/comments.md`.

**Testable:** Yes — `UI.showTooltip`, `UI.hideTooltip`, `UI.positionTooltip` and `UI.initDelegation` are all on the `UI` export at `internal/viewer/static/ui.js:957-982`.

**Verification:** `mise exec -- task test:viewer` passes.

**Depends on:** Task 1

---

### Task 6: Mark a described node and slice, and read the description off the marker

**Behavior:** Every described construct wears a marker in its top-right corner — a slice wears it in its header — and hovering the marker shows that description in the viewer's own tooltip, so a reader can see at a glance which parts of the model are documented.

**Acceptance Criteria:**
- [ ] A render of one slice holding a described command and an undescribed command draws exactly one element carrying `data-marker="description"`, and its `data-node-id` is the described command's
- [ ] That marker sits in the block's top-right: its `x` is greater than the block's horizontal centre and less than the block's right edge, and its `y` is below the block's top edge and above the block's vertical centre
- [ ] In a render holding a described automation stating a cadence and a described translation stating an external system, each marker's left edge — `x - Layout.labelWidth(textContent) / 2` — sits right of the right edge of every other `<text>` in the same group, so no marker paints over the cadence row or the external-system rows
- [ ] A described slice draws a `data-marker="description"` element inside its header; in the same render an undescribed slice draws none, and both slices' name `<text>` elements keep `text-anchor="middle"` with the `x` and `y` they have when neither is described
- [ ] Rendering one node set twice, described and undescribed, leaves every `.slice-box`, `.slice-header` and node-block rect identical in `x`, `y`, `width` and `height`; the described render differs only by the marker `<text>` elements
- [ ] No element carrying `data-marker` has a `<title>` child, so the browser's native tooltip cannot fire alongside the viewer's own
- [ ] After `UI.initDelegation(store)`, dispatching `pointerover` on a description marker displays the tooltip holding that node's description and no field rows, even for a node that carries fields; dispatching `pointerout` hides it again
- [ ] `git diff --name-only` does not name `internal/viewer/static/config.js`

**Affected Files/Modules:**
- `internal/viewer/static/renderer.js` — a description marking constant beside `clockMarking` (`:5`); the marker appended in `appendBlockLabels` or beside it (`:159-186`) and in `renderSlice` (`:251-267`)
- `internal/viewer/static/ui.js` — a marker branch in the `pointerover` delegation (`:656-667`) taking precedence over the block branch, plus the matching `pointerout`/`pointermove` handling
- `internal/viewer/tests/renderer.test.js` — marker placement and marker-free leaves
- `internal/viewer/tests/tooltip.test.js` — marker-hover leaves

**Patterns to Follow:**
- Follow the marking convention already in `internal/viewer/static/renderer.js`: a module-level glyph constant at `:5` and `centeredText` (`:50`) to draw it, exactly as the automation cadence badge does at `:176-181`.
- Do **not** use `svgTitle` (`:44`) for the marker. A native `<title>` on a child element fires alongside the group's own tooltip and shows two tooltips at once; the group-level `<title>` at `:179` stays as it is for the cadence.
- Route the marker's tooltip through the ui.js tooltip family built in Task 5 (`showTooltip`/`positionTooltip`/`hideTooltip`), reading the node from `store.nodeById` by the marker's `data-node-id`. The marker branch must be checked before `evt.target.closest('.diagram-node')`, because a block marker sits inside the block group; a slice-header marker has no `.diagram-node` ancestor and only the marker branch can reach it.
- `tasks/learnings.md` "The viewer shows a node twice — the canvas box and the detail panel" — assert drawn marks by walking the group's `<text>` elements (`drawnLabels`/`drawnText`, `internal/viewer/tests/renderer.test.js:224-230`), never `group.textContent`, which folds a `<title>` in.
- `tasks/learnings.md` "`svgPicture` sees labelled boxes and arrows only, so it is not the receipt for a new mark" — `drawnBoxes` (`renderer.test.js:241`) reads a group's *first* rect by design, so it reports the block and never a mark; use it for the "nothing else moved" half and the `<text>` walk for the marker itself.
- `pairedAutomations(schedule)` (`renderer.test.js:80`) is the shape for a fixture that shows the marked case, the unmarked case and a neighbouring box in one render — build the described/undescribed pair the same way.
- `tasks/learnings.md` "A viewer leaf must be able to fail only for the paint its name blames".
- `~/.config/ai/guidelines/testing/caller-patterns.md` Quick Reference — **UI** pattern.
- `~/.config/ai/guidelines/comments.md`.

**Testable:** Yes — through `Renderer.buildSVG` and `UI.initDelegation`.

**Verification:** `mise exec -- task test:viewer` passes; `git diff --name-only` does not name `internal/viewer/static/config.js`.

**Depends on:** Task 5

---

### Task 7: Mark a commented node and slice, and read the comments off the marker

**Behavior:** Every construct carrying comments wears a second, distinguishable marker, and hovering it shows those comments in source order, one per line, with the leading `#` stripped and nothing to edit them with.

**Acceptance Criteria:**
- [ ] A render of one slice holding a commented command and an uncommented command draws exactly one element carrying `data-marker="comments"`, its `data-node-id` the commented command's, and its glyph differs from the description marker's
- [ ] A node carrying both a description and comments draws both markers, their measured x ranges — `x ± Layout.labelWidth(textContent) / 2` — are disjoint, and both stay inside the block's right half and above its vertical centre
- [ ] Hovering the description marker of that node shows the tooltip holding the description and none of the comment texts; hovering its comments marker shows the comment texts and not the description
- [ ] Hovering the comments marker of a node whose `comments` read `['# first note', '# second note']` shows two lines, `first note` then `second note`, in that order — the source order, each with its leading `#` removed
- [ ] The comments tooltip contains no `input`, `textarea`, `button` or `[contenteditable]` element, and the detail panel for the same node shows none either
- [ ] A commented slice draws a `data-marker="comments"` element inside its header, and in the same render an uncommented slice draws none while both slice name `<text>` elements keep `text-anchor="middle"` with unchanged `x` and `y`
- [ ] After `UI.initDelegation(store)`, dispatching `pointerover` on the group of a node carrying comments but no fields and no description displays the tooltip holding those comment lines, while a sibling group whose node carries none of the three leaves it hidden — the block guard now reads fields, description or comments
- [ ] A comment text containing `<b>&</b>` appears in the tooltip as text and `tooltip.querySelector('b')` is null

**Affected Files/Modules:**
- `internal/viewer/static/renderer.js` — a comment marking constant beside the description marking; the marker drawn on blocks and in the slice header, offset from the description marker
- `internal/viewer/static/ui.js` — the marker branch resolves `data-marker="comments"` to a comments-only tooltip; `showTooltip` gains a comments block; the block-hover guard widens to fields, description or comments
- `internal/viewer/static/viewer.html` — a tooltip comments rule beside the existing `#tooltip` rules
- `internal/viewer/tests/renderer.test.js` — comment marker placement and coexistence leaves
- `internal/viewer/tests/tooltip.test.js` — comment tooltip content and read-only leaves

**Patterns to Follow:**
- The `#` arrives on the wire: `ast.Comment.Text` keeps its leading `#` and Task 2 carries it through unchanged, so the stripping is the viewer's job and belongs in one place in `ui.js`. Strip the `#` and at most one following space; leave the rest of the text alone.
- Comments are read-only by design (US-006 Context): a comment's meaning is tied to a source position that viewer editing can invalidate. No editing affordance in the tooltip or the panel.
- Reuse the marker drawing and delegation built in Task 6 — one marker element shape parameterised by kind, not a second copy. `tasks/learnings.md` "De-duplicate before a fan-out edit, and land the de-duplication with proof".
- `tasks/learnings.md` "One fill and stroke constant per element type — never named after the shape, never aliased" is the sibling convention for the two glyph constants: each marker names its own, even if the two are drawn by one helper.
- `tasks/learnings.md` "A viewer leaf must be able to fail only for the paint its name blames" — the "each shows only its own text" leaf asserts both tooltips in one render, so a tooltip that concatenates the two fails on the half it should not hold.
- `~/.config/ai/guidelines/javascript/testing-patterns.md` "Unit of Behavior" and "Test Behavior, Not DOM Structure".
- `~/.config/ai/guidelines/testing/caller-patterns.md` Quick Reference — **UI** pattern.
- `~/.config/ai/guidelines/comments.md`.

**Testable:** Yes — through `Renderer.buildSVG`, `UI.initDelegation` and `UI.showDetailPanel`.

**Verification:** `mise exec -- task test:viewer` passes.

**Depends on:** Task 2, Task 6

---

### Task 8: Mark commented context and aggregate headers

**Behavior:** A context and an aggregate carrying comments wear the same comment marker on their headers, beside the name and any inline description, and hovering it reads the comments off the diagram.

**Acceptance Criteria:**
- [ ] A render holding one commented context and one uncommented context draws a `data-marker="comments"` element inside the commented one's swimlane header and none inside the other's
- [ ] A render holding one commented aggregate and one uncommented aggregate beside it draws the marker inside the commented one's label row, with its measured x range disjoint from that aggregate's name and description `<text>` ranges and its right edge at or left of `row.x + row.w`
- [ ] An aggregate carrying both comments and a description too wide for its row truncates the description with `…` so the description's right edge sits at or left of the marker's left edge; in the same render an aggregate with a short description and no comments draws its description whole
- [ ] Rendering one node set twice, with and without comments on the context and the aggregates, leaves every `.ctx-header`, `.agg-row`, `.agg-area`, `.slice-box` and `.slice-header` rect identical in `x`, `y`, `width` and `height` and every name and description `<text>` identical in `x` and `y`; the commented render differs only by the marker `<text>` elements
- [ ] After `UI.initDelegation(store)`, dispatching `pointerover` on the context header marker and on the aggregate row marker each displays the tooltip holding that construct's comments, one per line with the leading `#` removed
- [ ] `git diff --name-only` does not name `internal/viewer/static/config.js`

**Affected Files/Modules:**
- `internal/viewer/static/renderer.js` — the swimlane header (`:205-213`) and the aggregate label pass (`:233-236`), which must reserve the marker's width before truncating the description added in Task 4
- `internal/viewer/tests/renderer.test.js` — header marker placement and the description/marker interaction
- `internal/viewer/tests/tooltip.test.js` — header marker hover

**Patterns to Follow:**
- Reuse the marker element and the delegation from Tasks 6 and 7 unchanged; the header markers are new *placements*, not a new kind. The delegation already matches on `[data-marker]` anywhere in the SVG, so a marker outside a `.diagram-node` group is reachable without a second handler.
- Keep the marker's class off `ctx-label`, `ctx-header`, `agg-label` and `agg-row`: those selectors drive the highlight-on-click behaviour in `internal/viewer/static/ui.js:720-742`, and a marker wearing one of them would highlight the whole context when clicked.
- The aggregate row is the crowded one — 22px tall, `row.w` floored at 100px, with the overlap note at `internal/viewer/static/renderer.js:230-232`. Subtract the marker's measured width from the space the description may occupy before truncating, using `Layout.labelWidth` as in Task 4.
- Neither header may grow in height: `L.swimlaneHdr` and `L.aggLabelH` stay where they are and `config.js` is not edited.
- `tasks/learnings.md` "A viewer leaf must be able to fail only for the paint its name blames" and "The viewer shows a node twice — the canvas box and the detail panel".
- `~/.config/ai/guidelines/testing/caller-patterns.md` Quick Reference — **UI** pattern.
- `~/.config/ai/guidelines/comments.md`.

**Testable:** Yes — through `Renderer.buildSVG` and `UI.initDelegation`.

**Verification:** `mise exec -- task test:viewer` passes; `git diff --name-only` does not name `internal/viewer/static/config.js`.

**Depends on:** Task 4, Task 7

---

## Summary

**Total tasks:** 8

**Ordering rationale:** Dependency-first, then risk. The two Go tasks come first because nothing in the viewer can show prose the wire does not carry, and because both keys owe an importer read-back in the same change or a viewer save silently deletes them. Descriptions precede comments: they touch the same two structs, and the comments task inherits the narrowed round-trip baseline the descriptions task establishes. The four display tasks then run from the least constrained surface to the most: the detail panel has room to spare, the context and aggregate headers have a documented overlap hazard, the tooltip has to start firing for nodes it ignores today, and the markers have to fit around the cadence, external-system and header text already drawn. Comment markers come last because they must sit beside the description markers and the inline aggregate description that the earlier tasks put there.

**Coverage:**

| Story | Criteria | Tasks |
|---|---|---|
| US-001 | all four | Task 1 |
| US-002 | all four | Task 2 |
| US-003 | all four | Task 3 |
| US-004 | all four | Task 4 |
| US-005 | marker placement, slice header marker, viewer's own tooltip, no marker without a description | Task 6 |
| US-005 | tooltip for a node with a description and no fields | Task 5 |
| US-006 | node and slice markers, distinguishable glyph, both markers each showing only its own text, source order with `#` stripped, read-only | Task 7 |
| US-006 | context and aggregate headers wearing the marker | Task 8 |

**Deferred, per the story's Non-Goals:** editing descriptions or comments in the viewer; repositioning comments; description text inside a node's box; field-level descriptions and comments; comments on invariants, specs, `decides_on` clauses and the model declaration, none of which has a diagram node; preserving blank lines and comment indentation exactly as authored.

**Out of scope by instruction:** `user-stories/progress.md` checkboxes are the human's post-run review step and stay untouched.

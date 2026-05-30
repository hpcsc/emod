# Full DSL Support in Diagram JSON Export and Web Viewer

## Progress
- [x] Task 1: Emit diagram nodes for trigger, view, automation, and translation constructs
- [x] Task 2: Emit diagram edges for new cross-slice relationships
- [x] Task 3: Position and render new block types in the web viewer
- [ ] Task 4: Add interaction (tooltip, drag, click-detail) for new block types
- [ ] Task 5: Render and interact with new edge types in the web viewer

---

## Story Reference
`user-stories/full-dsl-diagram-support.md` — US-001 and US-002.

---

## Codebase Context

### Affected Modules

| Module | Key Files | Role |
|--------|-----------|------|
| `internal/export/` | `export.go`, `export_test.go` | Builds the diagram-oriented JSON. `convertModelToDiagram()` (line 658) currently creates nodes only for actors, contexts, aggregates, slices, commands, events and edges only for flows. |
| `internal/viewer/` | `viewer.html` | 2178-line embedded HTML/JS frontend. Single-page SVG-based diagram viewer with pan/zoom, block drag, tooltip, detail panel, minimap. |
| `internal/ast/` | `ast.go` | AST types — `Trigger`, `View`, `Automation`, `Translation` already exist with all fields. No changes needed. |
| `internal/diagram/` | `drawio.go` | draw.io renderer already supports all new node and edge types. Useful reference for the export patterns and color scheme. |

### Existing Patterns (Export)

- `convertModelToDiagram()` (line 658): Iterates model hierarchy (actors → contexts → aggregates → slices) and creates `jsonDiagramNode` entries and `jsonDiagramEdge` entries. Uses `diagramIDGenerator` for deterministic sequential IDs (`command-1`, `event-2`).
- Flow edges (lines 763-776): Within a slice, name→ID maps (`cmdIDs`, `evtIDs`) are built in a first pass, then flow references are resolved in a second pass. Unresolved names are silently skipped.
- `jsonDiagramNode` (lines 593-600): Generic node struct with `ID`, `Type`, `Label`, `ParentID`, `Fields`, `Position`. Currently lacks fields for type-specific metadata (kind, actor, subscribes, trigger_event, etc.).
- `jsonDiagramEdge` (lines 601-606): Simple struct with `Source`, `Target`, `Type` (string discriminator).
- All JSON intermediate types for the new constructs already exist (`jsonTrigger`, `jsonView`, `jsonAutomation`, `jsonTranslation` at lines 115-166) but are used only by the human-readable `ExportJSON` path, not the diagram path.

### Existing Patterns (Viewer)

- **Layout** (lines 746-767): `computeLayout()` positions commands then events within each slice column. Blocks placed vertically with `L.boxHeight` and gap. Only `command` and `event` types are positioned.
- **Node rendering** (lines 886-903): SVG `<g>` groups with `<rect>` and `<text>` for command (blue) and event (orange) blocks. New types need distinct CSS classes and colors: trigger (`#e1d5e7`/`#9673a6`), view (`#d5e8d4`/`#82b366`), automation (`#f8cecc`/`#b85450`), translation (`#f5f5f5`/`#666666`).
- **Edge rendering** (lines 905-918): Only `flow` type edges are rendered. Single `computeArrowD()` function (line 1059) draws vertical-then-orthogonal paths. Manhattan routing not implemented.
- **Tooltip** (lines 950-1018): `attachBlockHandlers()` registers mouseenter/mousemove/mouseleave on `.cmd-block, .evt-block` selectors. Reads `node.fields` for table display.
- **Drag** (lines 1421-1595): `mousedown` checks for `.cmd-block, .evt-block` (single) or `.slice-header` (group). `mousemove` applies transform and updates arrows. `mouseup` commits.
- **Touch drag** (lines 1623-1695): Same selectors.
- **Click → detail panel** (lines 2093-2161): Event delegation checks `.cmd-block, .evt-block`, finds node by ID, calls `showDetailPanel()`. Panel shows label, type, fields, source position.
- **Edge click** (lines 2128-2139): `.flow-arrow` click highlights source and target.
- **Highlight** (lines 2072-2091): `addHighlight()` adds `.hl` class to elements with matching `data-node-id`. Context/aggregate label click (lines 2101-2124) highlights descendants.

---

## Tasks

### Task 1: Emit diagram nodes for trigger, view, automation, and translation constructs

**Behavior:** The diagram JSON export (`ExportDiagramJSON`) produces nodes of type `trigger`, `view`, `automation`, and `translation` when the model contains those constructs. Each new node has a `parentId` referencing its containing slice. Trigger nodes carry their `kind`, `actor`, and `reads` metadata. View nodes carry a `fields` array and `subscribes` array. Automation nodes carry `trigger_event`, `command`, and `target_context`. Translation nodes carry `external_system`, `reads`, `command`, and a nested event representation.

**Acceptance Criteria:**
- [ ] A model containing trigger, view, automation, and translation patterns produces diagram JSON nodes with types `trigger`, `view`, `automation`, and `translation`
- [ ] Each new node has a `parentId` set to its containing slice node's ID
- [ ] Trigger nodes include `kind`, `actor`, and `reads` as type-specific properties
- [ ] View nodes include `fields` array and `subscribes` array
- [ ] Automation nodes include `trigger_event`, `command`, and `target_context`
- [ ] Translation nodes include `external_system`, `reads`, `command`, and a nested event representation
- [ ] The `jsonDiagramNode` struct carries optional type-specific properties (not just generic `fields`)
- [ ] Existing nodes (actor, context, aggregate, slice, command, event) are unchanged
- [ ] Node IDs follow the `trigger-1`, `view-1`, `auto-1`, `trans-1` convention

**Affected Files/Modules:**
- `internal/export/export.go` — Extend `jsonDiagramNode` to carry type-specific metadata; add node creation for triggers, views, automations, translations in `convertModelToDiagram()`
- `internal/export/export_test.go` — Add tests for all new node types, parentId chains, type-specific properties

**Patterns to Follow:**
- Command node creation in `internal/export/export.go:725-741` — pattern for building a diagram node with parentId, fields, and position
- `convertFieldsToDiagram()` in `internal/export/export.go:784-799` — pattern for converting AST fields to diagram fields
- Existing JSON types `jsonTrigger`, `jsonView`, `jsonAutomation`, `jsonTranslation` in `internal/export/export.go:115-166` — reference for field names and structure
- `buildFullModel()` in `internal/export/export_test.go:2614-2658` — pattern for constructing test models with all node types

**Testable:** Yes — `TestExportDiagramJSON` provides the test harness pattern with `export.ExportDiagramJSON()`.

**Verification:** `go test ./internal/export/ -run TestExportDiagramJSON` passes, confirming new nodes appear with correct structure.

**Depends on:** None

---

### Task 2: Emit diagram edges for new cross-slice relationships

**Behavior:** The diagram JSON export produces five new edge types: `trigger_command` (trigger→command within same slice), `subscription` (event→view, may cross boundaries), `automation_trigger` (event→automation), `automation_command` (automation→command, may cross boundaries), and `translation_event` (translation→nested event). Name resolution uses a two-pass strategy: first pass builds a global name-to-ID map across all slices, second pass resolves references and emits edges. Unresolved name references are silently skipped.

**Acceptance Criteria:**
- [ ] `trigger_command` edges emitted from each trigger to its command within the same slice
- [ ] `subscription` edges emitted from each event to each subscribing view (cross-boundary)
- [ ] `automation_trigger` edges emitted from matching event node to automation node
- [ ] `automation_command` edges emitted from automation node to referenced command (cross-boundary)
- [ ] `translation_event` edges emitted from translation node to its nested event node
- [ ] Name resolution uses two-pass strategy: first pass collects all name→ID mappings globally, second pass emits edges
- [ ] Unresolved name references silently skipped (no panic, no broken output)
- [ ] Existing flow edges are unchanged

**Affected Files/Modules:**
- `internal/export/export.go` — Refactor `convertModelToDiagram()` from single-pass to two-pass name resolution; add edge creation for all new edge types
- `internal/export/export_test.go` — Add tests for each new edge type, cross-boundary resolution, and silent skip on unresolved names

**Patterns to Follow:**
- Flow edge resolution in `internal/export/export.go:763-776` — pattern for name-based lookup and edge creation (skip on missing name)
- Build `cmdIDs`/`evtIDs` maps (lines 722-723) — pattern for name→ID collection; extend to global maps across all slices
- Two-pass approach in the drawio renderer `internal/diagram/drawio.go:297-406` — reference for cross-slice edge patterns (subscription, automation wiring, translation wiring)

**Testable:** Yes — existing `TestExportDiagramJSON` test structure provides the harness.

**Verification:** `go test ./internal/export/ -run TestExportDiagramJSON` passes, confirming all new edge types are emitted correctly.

**Depends on:** Task 1 (edge targets must exist as nodes)

---

### Task 3: Position and render new block types in the web viewer

**Behavior:** The web viewer positions all block types (triggers, commands, events, views, automations, translations) within each slice column in a deterministic vertical order. Each type renders as an SVG block with a distinct CSS class and color scheme. Tooltips appear on hover showing the node's fields.

**Acceptance Criteria:**
- [ ] Layout computation positions triggers at the top of each slice column
- [ ] Layout positions commands below triggers, then events, then views, then automations, then translations
- [ ] Trigger blocks rendered with purple fill (`#e1d5e7`) and stroke (`#9673a6`), CSS class `trg-block`
- [ ] View blocks rendered with green fill (`#d5e8d4`) and stroke (`#82b366`), CSS class `view-block`
- [ ] Automation blocks rendered with red fill (`#f8cecc`) and stroke (`#b85450`), CSS class `auto-block`
- [ ] Translation blocks rendered with gray fill (`#f5f5f5`) and stroke (`#666666`), CSS class `trans-block`
- [ ] Each new block type has distinct CSS hover/highlight styles (existing pattern: `.cmd-block:hover rect`)
- [ ] Hovering any new block type shows a tooltip with its fields (following existing tooltip pattern)
- [ ] Existing command/event blocks are unchanged in position, color, and behavior

**Affected Files/Modules:**
- `internal/viewer/viewer.html` — Update `computeLayout()` to position all six block types (trigger, command, event, view, automation, translation) in order; add SVG rendering branches for each new type with colors and CSS classes; add CSS hover rules; extend `attachBlockHandlers()` selectors for new CSS classes

**Patterns to Follow:**
- Layout positioning for commands and events in `internal/viewer/viewer.html:746-767` — pattern for block positioning within slices; extend with additional type blocks in order
- SVG block rendering for command/event in `internal/viewer/viewer.html:886-903` — pattern for `<g>` group with `<rect>` and `<text>`, CSS class, and `data-node-id`
- CSS hover styles in `internal/viewer/viewer.html:317-331` — pattern for `.cmd-block:hover rect` / `.evt-block:hover rect`
- Tooltip handlers in `internal/viewer/viewer.html:996-1018` — extend CSS class selectors in `querySelectorAll` to include all block types
- Tooltip HTML rendering in `internal/viewer/viewer.html:953-964` — reads `node.fields`, shows them in a table

**Testable:** No — rendering logic is embedded in HTML/JS; verified by building `go build ./cmd/emod` and visually checking the viewer.

**Verification:** `go build ./cmd/emod` succeeds. Open viewer with a model containing all new types — blocks appear at correct positions with correct colors.

**Depends on:** Task 1 (needs trigger/view/automation/translation nodes in the JSON)

---

### Task 4: Add interaction (tooltip, drag, click-detail) for new block types

**Behavior:** All new block types support the full set of viewer interactions: individual drag, slice-group drag, click to open detail panel with type-specific metadata, and highlight on context/aggregate label click. The detail panel shows type-specific information (e.g., kind/actor/reads for triggers, fields/subscribes for views, trigger_event/command/target_context for automations).

**Acceptance Criteria:**
- [ ] Each new block type can be dragged individually (mousedown on block, threshold-activated drag)
- [ ] Slice-group drag moves all children including new block types
- [ ] Clicking a new block type opens the detail panel showing its label, type, fields, and type-specific metadata
- [ ] The detail panel shows correct metadata per type: trigger shows kind/actor/reads, view shows fields/subscribes, automation shows trigger_event/command/target_context, translation shows external_system/reads/command/nested event
- [ ] New block types are highlighted when clicking their parent context label
- [ ] New block types are highlighted when clicking their parent aggregate label
- [ ] New block types participate in touch drag
- [ ] Existing command/event interactions (drag, click, highlight) are unchanged

**Affected Files/Modules:**
- `internal/viewer/viewer.html` — Update `getSliceChildNodeIds()` to include new types; update `showDetailPanel()` to render type-specific metadata sections; update all `querySelectorAll`/`closest` references that currently filter by `.cmd-block, .evt-block` to include new CSS classes; update `mousedown` drag detection, `mousemove` drag handling, `mouseup` commit, touch equivalents, `detail panel`, `click handler`, `updateBlockTransform`, `commitDrag`, `getConnectedEdges`, `updateArrowsForNode`

**Patterns to Follow:**
- Single block drag initiation in `internal/viewer/viewer.html:1425-1443` — check `evt.target.closest('.cmd-block, .evt-block')`; extend selectors
- Slice group drag in `internal/viewer/viewer.html:1447-1479` — `getSliceChildNodeIds()` currently only returns command/event children; extend to return all block type children
- Detail panel `showDetailPanel()` in `internal/viewer/viewer.html:2027-2063` — currently shows label, type, fields, source position; extend with conditional sections per type
- Click handling in `internal/viewer/viewer.html:2141-2152` — check `target.closest(".cmd-block, .evt-block")`; extend selectors
- `getSliceChildNodeIds()` in `internal/viewer/viewer.html:1114-1121` — currently filters `n.type === "command" || n.type === "event"`; extend to include all block types
- `getConnectedEdges()` in `internal/viewer/viewer.html:1049-1057` — currently filters `e.type === "flow"`; may need to include new edge types if called during drag
- `updateBlockTransform()` in `internal/viewer/viewer.html:1083-1091` — query selector needs extension
- `commitDrag()` in `internal/viewer/viewer.html:1095-1111` — query selector needs extension
- All QSA references (touch, mouse, etc.) — uniformly extend selectors

**Testable:** No — interaction logic is embedded in HTML/JS; verified by manual testing.

**Verification:** `go build ./cmd/emod` succeeds. Open viewer — new blocks are draggable, clickable, and show correct detail panel data.

**Depends on:** Task 3 (blocks must be rendered before they can be interacted with)

---

### Task 5: Render and interact with new edge types in the web viewer

**Behavior:** The web viewer renders four new edge types with distinct visual styles: `subscription` (dashed green), `automation_trigger` (dashed red), `automation_command` (solid red), `trigger_command` (solid purple). Edges crossing slice/aggregate/context boundaries use Manhattan routing (horizontal segment then vertical segment). Clicking an edge highlights both its source and target nodes. Edges update correctly when connected nodes are dragged.

**Acceptance Criteria:**
- [ ] `subscription` edges rendered as dashed green (`#82b366`) arrows
- [ ] `automation_trigger` edges rendered as dashed red (`#b85450`) arrows
- [ ] `automation_command` edges rendered as solid red (`#b85450`) arrows
- [ ] `trigger_command` edges rendered as solid purple (`#9673a6`) arrows
- [ ] `flow` edges (existing) remain solid gray (`#666666`)
- [ ] Cross-boundary edges use Manhattan routing (exit source at source Y, horizontal to target column X, vertical to target)
- [ ] Edges within the same slice use the existing orthogonal routing
- [ ] Clicking an edge highlights both its source and target nodes
- [ ] Edges update position when connected nodes are dragged
- [ ] Distinct CSS classes for each edge type to support hover/highlight

**Affected Files/Modules:**
- `internal/viewer/viewer.html` — Extend edge rendering to handle four new edge types with per-type SVG stroke styles and colors; implement Manhattan routing for cross-boundary edges; add arrow markers for new edge types; extend `getConnectedEdges()` to include new edge types; extend `computeArrowD()` or create a Manhattan variant; update edge click handler to recognize new edge class names; update `updateArrowsForNode()` and `updateArrowsForSlice()` to include new edge types

**Patterns to Follow:**
- Flow edge rendering in `internal/viewer/viewer.html:905-918` — pattern for SVG `<path>` with `marker-end`, stroke, and data attributes
- Arrow marker `defs` in `internal/viewer/viewer.html` (near top of file, likely in initial SVG setup) — pattern for creating arrowhead markers; add per-color markers
- Edge click highlighting in `internal/viewer/viewer.html:2128-2139` — pattern for `.flow-arrow` click to highlight source and target
- `getConnectedEdges()` in `internal/viewer/viewer.html:1049-1057` — currently filters `edge.type === "flow"`; extend to include all edge types
- Manhattan routing approach: edges exit source node bottom-center horizontally to the target's column X, then vertically to target node top-center (see `internal/diagram/drawio.go:336-343` for the draw.io reference of subscription edge routing with waypoints)

**Testable:** No — rendering logic is embedded in HTML/JS; verified by manual testing.

**Verification:** `go build ./cmd/emod` succeeds. Open viewer with a model containing cross-slice references — edges appear with correct colors, styles, and Manhattan routing. Clicking edges highlights source and target. Dragging nodes updates connected edges.

**Depends on:** Task 2 (needs edges in the JSON), Task 3 (needs blocks rendered for edge routing positions), Task 4 (needs drag implemented for edge update-on-drag)

---

## Summary

**Total tasks:** 5

**Ordering rationale:** Dependency-first, then build-vs-interact separation.

1. **Task 1** (export nodes) and **Task 2** (export edges) form the backend foundation — they must come first because the viewer consumes the diagram JSON. Task 2 depends on Task 1 because edges reference node IDs.

2. **Task 3** (viewer block rendering) and **Task 4** (viewer block interaction) are split because rendering is independently verifiable (see blocks render correctly) before adding drag/click behavior. Task 4 depends on Task 3 because interaction requires rendered elements.

3. **Task 5** (viewer edges) comes last because it depends on all prior tasks: edges need blocks rendered (Task 3), drag to update on (Task 4), and data from the export (Task 2).

**Acceptance criteria coverage:**

| Task | US-001 ACs | US-002 ACs |
|------|-----------|-----------|
| 1 | Export: nodes for all types, parentId, type-specific properties | — |
| 2 | — | Export: all five edge types, two-pass resolution, silent skip |
| 3 | Viewer: block colors, positioning, ordering, CSS classes, tooltip | — |
| 4 | Viewer: tooltip, drag, click→detail panel, context/aggregate highlight | — |
| 5 | — | Viewer: edge colors/styles, Manhattan routing, click highlight, drag update |

All acceptance criteria from US-001 and US-002 are covered. No criteria are deferred.

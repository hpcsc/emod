## Progress
- [x] Task 1: Add source position to diagram JSON nodes
- [x] Task 2: Implement detail panel, element highlighting, and click dismissal

---

## Story Reference

**US-005** from `user-stories/web-diagram-viewer.md` — Inspect node details.

**Description:** As a model author, I want to click on a diagram element to see its full details so that I can review field definitions, types, and modifiers without reading the source file.

---

## Codebase Context

### Exploration Findings

**Go Backend (`internal/export/export.go`):**
- `jsonDiagramNode` struct (lines 593-599) currently has no position field. The full AST JSON export already defines `jsonPosition` (lines 16-21) and `convertPosition()` (lines 227-236), which are used throughout the full export types (e.g., `jsonCommand` at line 78, `jsonEvent` at line 87).
- `convertModelToDiagram()` (lines 657-775) builds all diagram nodes but never populates position data. Every AST node type (`Command`, `Event`, `Context`, `Aggregate`, `Slice`, `Actor`) has a `NamePos Position` field in `internal/ast/ast.go`.
- Existing tests in `internal/export/export_test.go` use `testify/require` and verify full AST JSON position fields (e.g., lines 776-816 for command positions). Diagram JSON tests start at line 1404 and test node fields, edges, parentId chains.

**Frontend (`internal/viewer/viewer.html`):**
- Command/event blocks are rendered as SVG `<g>` elements with `data-node-id` attribute and `cursor: pointer` CSS (lines 681-698). Context and aggregate nodes are rendered as SVG rectangles/text groups (swimlanes at line 638, aggregate headers at line 651) with no click handlers or data attributes.
- Flow arrows are SVG `<path>` elements with `pointer-events="none"` (line 717), making them unclickable.
- A hover tooltip already shows field info on mouseenter/mouseleave (lines 750-817). The tooltip is positioned via `position: fixed` and uses `pointer-events: none` (line 354).
- The existing `#data-panel` (line 438) is a toggle sidebar for the source textarea — not the detail panel this story requires.
- Panning is skipped when clicking on `.cmd-block, .evt-block` elements (line 1099). No other click handling exists for diagram elements.

**Server (`internal/viewer/serve.go`):**
- The `/parse` endpoint returns diagram JSON via `export.ExportDiagramJSONDiagnostics()`. No changes needed here — the position data flows through automatically once added to `jsonDiagramNode`.

---

## Tasks

### Task 1: Add source position to diagram JSON nodes

**Behavior:** The diagram JSON produced by `ExportDiagramJSON` includes a `position` object on each node (commands, events, contexts, aggregates, slices, actors) containing `filename`, `line`, and `column` from the AST source position.

**Acceptance Criteria:**
- [ ] `jsonDiagramNode` struct gains a `Position *jsonPosition` field
- [ ] `convertModelToDiagram()` populates the position field for every node type from the corresponding AST node's `NamePos`
- [ ] Command and event nodes include their source position
- [ ] Context, aggregate, slice, and actor nodes include their source position
- [ ] Zero-value positions produce `null`/omitted position (consistent with `convertPosition` behavior)
- [ ] Existing diagram JSON tests continue to pass (backward compatible — existing JSON consumers are unaffected by the new field)

**Affected Files/Modules:**
- `internal/export/export.go` — Add `Position *jsonPosition` to `jsonDiagramNode` struct (line 593); populate in `convertModelToDiagram()` for each node type (lines 673-753)
- `internal/export/export_test.go` — Add test scenarios verifying position is populated for command, event, context, aggregate, slice, and actor nodes

**Patterns to Follow:**
- Existing usage of `convertPosition()` and `jsonPosition` in the full AST JSON types — see `internal/export/export.go:78` for `jsonCommand`, `export.go:87` for `jsonEvent`, and `export.go:225-236` for the `convertPosition` function itself
- Existing diagram JSON test structure in `internal/export/export_test.go:1588-1660` (the "command and event nodes include fields array" test) shows how to drill into diagram JSON nodes and verify nested properties

**Testable:** Yes — `ExportDiagramJSON` is an exported function returning `[]byte`. Tests construct AST models with `NamePos` set, call `ExportDiagramJSON`, unmarshal, and verify the `position` field contents. Follow the existing `TestExportDiagramJSON` umbrella pattern.

**Verification:** `go test ./internal/export/...` passes; new test assertions verify position fields.

**Depends on:** None

---

### Task 2: Implement detail panel, element highlighting, and click dismissal

**Behavior:** Clicking diagram elements triggers context-appropriate feedback:
- Clicking a command or event opens a persistent detail panel showing the element name, all fields with types and modifiers, and the source file location (file name, line number).
- Clicking a context or aggregate swimlane header highlights all elements within it.
- Clicking a flow arrow highlights both the source command and target event.
- Pressing Escape or clicking on the diagram background dismisses the detail panel.

**Acceptance Criteria:**
- [ ] Clicking a command or event block opens a fixed-position detail panel displaying the element name, all fields (name, type, modifier), and source position
- [ ] Clicking a context swimlane header (or its label) applies a visual highlight to all commands, events, slices, and aggregates within that context
- [ ] Clicking an aggregate label applies a visual highlight to all elements within that aggregate
- [ ] Clicking a flow arrow highlights the source command and target event blocks
- [ ] The detail panel shows source file name and line number from the node's `position` data (or a graceful fallback if absent)
- [ ] Pressing the Escape key closes the detail panel and removes all highlights
- [ ] Clicking on the diagram background (not on any interactive element) closes the detail panel and removes all highlights
- [ ] The hover tooltip continues to work while no detail panel is open — opening the detail panel suppresses the hover tooltip for the selected element
- [ ] Flow arrows are made clickable (`pointer-events` enabled, data attributes for source/target added)

**Affected Files/Modules:**
- `internal/viewer/viewer.html` — All frontend changes:
  - New CSS for the detail panel and highlight styles (e.g., `.highlighted` class with stroke/opacity change)
  - New HTML for the detail panel container (separate from `#tooltip` and `#data-panel`)
  - JavaScript: click handlers for blocks, swimlanes, aggregate labels, and flow arrows
  - JavaScript: detail panel rendering logic (name, fields table, source position line)
  - JavaScript: highlight/unhighlight logic (toggle CSS class on SVG elements by node ID)
  - JavaScript: dismiss handlers (Escape keydown, diagram background click, close button)
  - Modified SVG rendering for flow arrows: add `data-edge-source` and `data-edge-target` attributes, change `pointer-events="none"` to allow interaction
  - Modified context/aggregate rendering: add `data-node-id` attribute and `cursor: pointer` to swimlane groups and aggregate header text for click targeting

**Patterns to Follow:**
- Existing tooltip implementation at `viewer.html:750-794` shows the pattern for creating positioned overlays with element data — follow for detail panel presentation style
- Block rendering at `viewer.html:681-698` shows how `data-node-id` is attached to SVG elements — follow for adding data attributes to context/aggregate groups and flow arrows
- The panning guard at `viewer.html:1098-1099` shows how interactive SVG elements are excluded from pan — extend to include the new clickable elements

**Testable:** No — The frontend is vanilla JS in a single HTML file with no test framework. Changes are verified through manual interaction in a browser.

**Verification:** Manual testing: serve the viewer, load an `.emod` file, click each element type and verify the described behavior, press Escape to dismiss, click background to dismiss.

**Depends on:** Task 1 — The detail panel needs `position` data on diagram JSON nodes to display source file location

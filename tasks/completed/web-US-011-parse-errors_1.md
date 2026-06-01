# US-011: Display parse errors alongside partial diagram

## Progress
- [x] Task 1: Add diagnostics storage to store and wire from parse response
- [x] Task 2: Render diagnostics panel and error badge in the UI
- [x] Task 3: Wire click-to-highlight from diagnostic items to diagram nodes

---

## Story Reference

User story US-011 from `user-stories/web-diagram-viewer.md` (lines 162–172).  
Depends on US-003 (already completed).

---

## Codebase Context

### Parse response already includes diagnostics
The WASM entry point at `internal/wasm/pipeline.go` and the direct JSON path in `web/model.js:sendParse()` both return a `{ diagnostics, diagram }` envelope. The diagnostics array is always present — empty for valid files, populated for files with parse errors. However, the frontend currently ignores `diagnostics` everywhere.

### Where diagnostics are discarded
- `web/model.js:93-97` — Direct JSON input paths hardcode `diagnostics: []` instead of preserving whatever was in the input
- `web/viewer.js:131` — `Model.setModelData(store, data.diagram)` only passes the `diagram` field; `data.diagnostics` is never stored

### Store structure
`web/store.js:createStore()` returns a flat object with arrays for `nodes`, `edges`, `layoutPositions`, `arrowData`; maps for `nodeById`; and objects for `viewport`, `interaction`, `dom`. There is no `diagnostics` field yet.

### UI pattern for panels
The viewer has multiple toggleable panels — minimap (`web/ui.js:154-167`), context panel (`web/ui.js:201-214`), detail panel (`web/ui.js:229-380`). Each follows the pattern of: DOM elements in `index.html`, CSS classes (`hidden`, `collapsed`), and a UI module function that toggles visibility.

### Bus event pattern
The event bus (`web/bus.js`) uses `bus.emit(event, data)` and `bus.on(event, handler)`. Existing events: `model:updated`, `data:changed`, `viewport:changed`, `diagram:before-render`, `diagram:rendered`, `node:delete`.

### Highlighting mechanism
`web/ui.js:390-409` provides `clearHighlights(store)`, `addHighlight(store, nodeId)`, and `highlightElements(store, ids)`. Highlighting works by adding an `hl` CSS class to SVG elements with `data-node-id` attributes matching the requested IDs.

### Diagnostic entry fields
Each diagnostic from the WASM/JSON response has: `file` (string), `line` (int), `column` (int), `message` (string), `severity` (string), `rule_name` (string, optional). Diagram nodes may have a `position` object with `filename` and `line` fields — matching diagnostics to nodes uses file name and line number.

### Test structure
Tests live in `internal/viewer/tests/` and import from `internal/viewer/static/` (a copy of `web/`). The `viewer.test.js` file uses `vi.mock` for all modules and `createRequiredElements()` to set up DOM fixtures. The `model.test.js` file tests `sendParse` with mocked WASM. Both files must be updated to match any changes to their tested modules.

---

## Tasks

### Task 1: Add diagnostics storage to store and wire from parse response

**Behavior:** The parse response's `diagnostics` array is preserved in the store and available via the event bus for other modules to consume. The existing diagram rendering continues unchanged — diagnostics do not affect the diagram display yet.

**Acceptance Criteria:**
- [ ] `createStore()` returns a store object with a `diagnostics: []` field
- [ ] When `sendParse` resolves with non-empty diagnostics (from WASM or direct JSON input), the viewer stores them in `store.diagnostics` and emits a `diagnostics:changed` bus event with the diagnostics data
- [ ] When `sendParse` resolves with empty diagnostics, the viewer stores `[]` and emits `diagnostics:changed` with an empty array
- [ ] When `sendParse` resolves with direct JSON input (diagram JSON or AST JSON), the viewer stores the actual diagnostics from the input (not hardcoded `[]`)
- [ ] All existing tests in `model.test.js` and `viewer.test.js` continue to pass without modification

**Affected Files/Modules:**
- `web/store.js` — Add `diagnostics: []` to the store object returned by `createStore()`
- `web/model.js` — In `sendParse`, return the diagnostics from the input data rather than hardcoding empty arrays on the direct-JSON paths (lines 93 and 97)
- `web/viewer.js` — In the render button click handler (around line 131), store `data.diagnostics` in the store and emit `diagnostics:changed`

**Patterns to Follow:**
- Store field pattern: `web/store.js:1-48` — flat field initialization for all store properties
- Bus emit pattern: `web/viewer.js:12-14` — `bus.emit('data:changed', { store: s })` for signaling state changes
- `web/model.js:19-31` — `setModelData` pattern for bulk store updates

**Testable:** Yes — test that store has expected diagnostics after sendParse resolves, by calling `Model.sendParse` and inspecting store state or listening for bus events.

**Verification:** Tests pass, `go build` succeeds, manual test by pasting invalid `.emod` content into the viewer and confirming the bus event fires with diagnostics payload.

**Depends on:** None

---

### Task 2: Render diagnostics panel and error badge in the UI

**Behavior:** When diagnostics are present, a panel listing all parse errors is shown, and a persistent error badge in the header displays the error count. When all diagnostics are cleared (valid file), both the panel and badge disappear. The panel is dismissible via a close button but remains accessible by clicking the error badge.

**Acceptance Criteria:**
- [ ] A diagnostics panel appears when `diagnostics:changed` fires with a non-empty array; it lists each diagnostic showing file name, line number, severity, and error description
- [ ] A persistent error badge appears in the header showing the count of errors when diagnostics are present; clicking the badge opens the panel if closed
- [ ] The diagnostics panel has a close button that dismisses it (still accessible via the badge)
- [ ] When `diagnostics:changed` fires with an empty array, the diagnostics panel is hidden and the error badge is removed
- [ ] A fully valid file (zero diagnostics) shows no diagnostics panel or error badge at any point during or after rendering
- [ ] The diagram renders normally alongside the diagnostics panel — they coexist without layout conflicts

**Affected Files/Modules:**
- `web/index.html` — Add diagnostics panel DOM (container, close button, list area), error badge DOM (in the header alongside existing buttons), and CSS styles for both
- `web/ui.js` — Add exported functions: `updateDiagnosticsPanel(store, diagnostics)`, `toggleDiagnosticsPanel(store)`, `hideDiagnosticsPanel(store)`, and error badge update logic
- `web/viewer.js` — Subscribe to `diagnostics:changed` bus event to call the new UI functions; store DOM references for the new panel and badge elements

**Patterns to Follow:**
- Panel toggle pattern: `web/ui.js:154-167` — `toggleMinimap` show/hide behavior with CSS class switching
- DOM reference pattern: `web/viewer.js:96-120` — storing element IDs from `document.getElementById` in `store.dom`
- Header button pattern: `web/index.html:797-801` — existing header buttons (minimap, contexts, export) as model for error badge placement
- Bus subscriber pattern: `web/viewer.js:12-29` — `bus.on('data:changed', ...)` for reacting to state changes

**Testable:** Yes — test by creating DOM fixtures (similar to `viewer.test.js:43-78`), setting diagnostics on store, calling the new UI functions, and asserting on visible panel content and badge presence.

**Verification:** Tests pass, manual test by pasting `.emod` content with errors and confirming the panel appears with correct diagnostic info, badge shows count, close/badge toggle works, and valid file shows nothing.

**Depends on:** Task 1

---

### Task 3: Wire click-to-highlight from diagnostic items to diagram nodes

**Behavior:** Clicking a diagnostic item in the diagnostics panel finds matching diagram nodes by file name and line number (from the node's `position` data), highlights them in the diagram, and scrolls the viewport to bring them into view. If no matching node exists (element could not be parsed), the diagnostic item shows a "not rendered" indicator instead.

**Acceptance Criteria:**
- [ ] Clicking a diagnostic item highlights all diagram nodes whose `position.filename` and `position.line` match the diagnostic's `file` and `line`
- [ ] Highlighted nodes use the existing highlight mechanism (`.hl` class, dimmed unhighlighted nodes)
- [ ] If no diagram node matches the diagnostic location, the clicked diagnostic item displays a "could not be rendered" visual indicator (e.g., text or icon) without attempting to highlight
- [ ] Clicking a diagnostic that matches a node also optionally re-centers the viewport to make the node visible (using existing `fitToView` or viewport adjustment)
- [ ] The highlight is cleared when the diagnostics panel is closed or when the user clicks elsewhere on the diagram
- [ ] Multiple diagnostic clicks work sequentially — each click replaces the previous highlight

**Affected Files/Modules:**
- `web/ui.js` — Extend existing highlight functions or add new logic to match diagnostics to nodes by file/line; add click delegation on the diagnostics panel items; add "not rendered" state management for unmatched diagnostics
- `web/viewer.js` — Store DOM reference for diagnostics panel list container; wire click delegation for diagnostic items
- `web/index.html` — May require minor CSS additions for the "could not be rendered" indicator styling and diagnostic item hover states

**Patterns to Follow:**
- Highlight mechanism: `web/ui.js:390-409` — `clearHighlights`, `addHighlight`, `highlightElements` functions
- SVG click delegation: `web/ui.js:571-637` — click event delegation via `.closest()` on SVG elements as pattern for panel item click handling
- Viewport centering: `web/interaction.js:30-55` — `fitToView` as reference for viewport adjustment to reveal highlighted nodes
- Panel item click pattern: `web/ui.js:188-198` — checkbox change handler on context panel items as reference for per-item event binding in a panel

**Testable:** Yes — test by creating a store with nodes that have positions matching diagnostics, rendering the diagnostics panel, dispatching a click on a diagnostic item, and asserting that the correct node IDs are highlighted (via `store.interaction.highlighted`). Also test the unmatched case.

**Verification:** Tests pass, manual test by loading a partially valid `.emod` file, clicking on a diagnostic entry that maps to a successfully parsed element (highlighting works) and one that maps to nothing ("could not be rendered" shown).

**Depends on:** Task 2

---

## Summary

- **Total tasks:** 3
- **Task ordering rationale:** Data plumbing first (Task 1), then display (Task 2), then interaction (Task 3). Each builds on the previous without circular dependencies.
- **Acceptance criteria coverage:**
  - AC "partial diagram renders": Already satisfied by the existing parser and renderer — no code changes needed.
  - AC "diagnostics panel displays errors": Covered by Task 2.
  - AC "clicking a diagnostic highlights in diagram": Covered by Task 3.
  - AC "panel is dismissible but accessible via error badge": Covered by Task 2.
  - AC "valid file shows no panel or badge": Covered by Tasks 1 and 2 together (empty diagnostics from Task 1 trigger correct display in Task 2).
- **No test-only tasks:** Each task writes its own tests alongside the implementation.

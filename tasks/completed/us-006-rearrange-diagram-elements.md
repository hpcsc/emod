# US-006: Rearrange diagram elements by dragging

## Progress
- [x] Task 1: Drag command and event nodes with real-time arrow updates
- [x] Task 2: Drag slice group header to move all slice contents together
- [x] Task 3: Add reset layout button

## Story Reference
`user-stories/web-diagram-viewer.md` — **US-006: Rearrange diagram elements by dragging**

## Codebase Context

**Current viewer state:**
`internal/viewer/viewer.html` (~1692 lines, single self-contained HTML+CSS+JS file). The viewer already has pan/zoom (US-004), minimap, detail panel with highlighting (US-005), and click handlers for context/aggregate labels, flow arrows, and command/event blocks.

**Key state variables:**
- `nodes[]` — flat array of all diagram nodes
- `edges[]` — flat array of all diagram edges
- `layoutPositions{}` — map of nodeId -> `{x, y, w, h}` populated by `computeLayout()`
- `viewport{}` — `{offsetX, offsetY, zoomScale}` for current pan/zoom
- `arrowData[]` — computed arrow paths for flow edges

**Rendering flow:**
`renderDiagram()` calls `computeLayout()` (which resets `layoutPositions = {}`), generates SVG HTML, and sets it via `svg.innerHTML`. After render, it calls `renderActorAnnotations()`, `attachBlockHandlers()`, and `applyViewport()`.

**Important constraint:**
`computeLayout()` (lines 676-759) always resets `layoutPositions` to a fresh auto-layout. Any user-modified positions must be stored separately and applied as overrides after `computeLayout()` runs, otherwise re-rendering the diagram (e.g., clicking the Render button) would discard manual positions.

**Existing drag-related infrastructure:**
- `screenToDiagram()` (lines 980-992) converts screen coordinates to diagram coordinates accounting for viewport transform
- Pan uses mousedown/mousemove/mouseup on the SVG; mousedown on interactive elements (`.cmd-block`, `.evt-block`, `.flow-arrow`, `.ctx-label`, `.agg-label`) is skipped at line 1251
- TouchState supports one-finger pan and two-finger pinch via touchstart/touchmove/touchend
- Command/event blocks have `data-node-id` attributes and classes `cmd-block` / `evt-block`
- Slice groups have classes `slice-{sl.id}` (rendered at lines 811-825)
- Arrow `<path>` elements have class `flow-arrow` and `data-source` / `data-target` attributes

**Non-goal from the story:**
"Persisting diagram layout positions to disk or across sessions" — positions are session-only, no localStorage or server-side persistence required.

---

## Tasks

### Task 1: Drag command and event nodes with real-time arrow updates

**Behavior:** Individual command and event blocks can be dragged to new positions. Arrows (edges) connected to dragged nodes update in real-time during the drag. Clicking a block (without dragging) still opens the detail panel as before — click is distinguished from drag by movement threshold.

**Acceptance Criteria:**
- [ ] Individual command and event nodes can be dragged to new positions (mouse and touch)
- [ ] Edges (arrows) follow their connected nodes in real-time as the node is dragged
- [ ] Clicking a block without dragging still opens the detail panel (click vs. drag is distinguished)
- [ ] Dragged positions survive re-renders (e.g., clicking Render preserves manual positions) and pan/zoom operations
- [ ] The cursor changes to `move` when hovering over a draggable block and to `grabbing` during active drag
- [ ] Dragging works correctly regardless of the current zoom level and pan offset
- [ ] Pan is not triggered when starting a drag on a command/event block

**Affected Files/Modules:**
- `internal/viewer/viewer.html` — Add a `userPositions` map (nodeId -> `{x, y}`) alongside `layoutPositions`; modify `renderDiagram()` to apply `userPositions` overrides after `computeLayout()`; add mousedown/mousemove/mouseup handlers for `.cmd-block` and `.evt-block` elements; add corresponding touchstart/touchmove/touchend handlers; implement arrow path recomputation for edges connected to the dragged node; implement click-vs-drag detection

**Patterns to Follow:**
- Follow the pan event handling pattern at lines 1244-1277 for mousedown/mousemove/mouseup structure and global document listener approach
- Follow the touch event handling pattern at lines 1280-1393 for touchstart/touchmove/touchend structure and mode management
- Follow the `screenToDiagram()` function at lines 980-992 for coordinate conversion during drag
- Follow the arrow path computation logic at lines 851-868 to recompute paths for edges connected to the dragged node
- Follow the existing event delegation pattern at lines 1610-1675 for the SVG click handler — extend the mousedown guard at line 1251 to handle the new drag case

**Testable:** No — interactive browser behavior, verified by manual testing

**Verification:** Load a diagram with at least one command, one event, and a flow edge between them. Drag a command block — it moves freely and the arrow updates in real-time. Drag an event block — same behavior. Click (without drag) on a block — the detail panel opens. Pan and zoom — dragged positions remain. Click Render again — manual positions are preserved.

**Depends on:** None

---

### Task 2: Drag slice group header to move all slice contents together

**Behavior:** Dragging the header area of a slice column moves all command and event blocks within that slice as a group, maintaining their relative positions. This lets users reorganize entire columns at once.

**Acceptance Criteria:**
- [ ] Dragging a slice group header (the header `<rect>` or `<text>` at the top of the slice column) moves all command/event blocks within that slice together
- [ ] Empty slices without child nodes still respond to header drag (the header itself moves)
- [ ] Edges connected to moved nodes update in real-time during the drag
- [ ] Works with both mouse and touch input
- [ ] Clicking the slice header (without drag) does not cause any unwanted behavior
- [ ] The cursor changes to `move` when hovering over the slice header area and to `grabbing` during active drag

**Affected Files/Modules:**
- `internal/viewer/viewer.html` — Add event handlers for slice header elements (which are rendered inside `<g class="slice-{sl.id}">`); identify all child command/event nodes of the dragged slice by traversing the `nodes[]` array or using `layoutPositions`; update positions for all child nodes in `userPositions`; recompute arrow paths for all edges connected to any moved child

**Patterns to Follow:**
- Follow the drag event handling pattern established in Task 1 (mousedown/mousemove/mouseup, touchstart/touchmove/touchend)
- Follow the arrow recomputation pattern from Task 1, applied to multiple nodes
- The slice structure is rendered at lines 815-824 with class `slice-{sl.id}` — use this class or a new data attribute to identify the slice group in the DOM
- Follow the `getDescendantIds()` function pattern at lines 1519-1540 to identify child nodes of a slice (filter by parentId matching the slice id)

**Testable:** No — interactive browser behavior, verified by manual testing

**Verification:** Load a diagram with a slice containing multiple commands and events. Drag the slice header (the top bar/label area of the slice column) — all blocks within that slice move together as a unit. Edges connected to those blocks update in real-time. Switch between individual node drag (Task 1) and slice header drag — both work without interference.

**Depends on:** Task 1

---

### Task 3: Add reset layout button

**Behavior:** A "Reset layout" button in the header toolbar clears all user-modified positions and restores the auto-computed layout. The diagram re-renders with the original auto-layout positions.

**Acceptance Criteria:**
- [ ] A "Reset layout" button is visible in the header toolbar alongside the existing "Fit view" and "Minimap" controls
- [ ] Clicking the button clears all user-modified positions and restores the auto-computed layout for every node
- [ ] Arrows are recomputed based on the restored auto-layout positions
- [ ] The button is disabled or visually subdued when there are no user-modified positions (no manual drags performed)
- [ ] The viewport position and zoom level are preserved (only the node positions reset, not the camera)

**Affected Files/Modules:**
- `internal/viewer/viewer.html` — Add button HTML in the header toolbar (at lines 546-551); add CSS styling consistent with existing header buttons (`#fit-view` styling at lines 142-156); add `resetLayout()` JavaScript function that clears `userPositions` and calls `renderDiagram()` followed by `applyViewport()`; wire the button click event

**Patterns to Follow:**
- Follow the existing `#fit-view` button pattern at lines 142-156 for styling (background, color, border, padding, hover behavior)
- Follow the `#fit-view` event listener wiring at line 1145 for the click handler registration
- Follow the header button layout at lines 546-551 for placement in the header toolbar

**Testable:** No — interactive browser behavior, verified by manual testing

**Verification:** Load a diagram, drag several nodes to new positions (Tasks 1 and 2). Click "Reset layout" — all nodes snap back to their auto-computed positions. The viewport position and zoom level remain unchanged. After reset, the button becomes disabled or subdued.

**Depends on:** Task 1

---

## Summary

**Total tasks:** 3

**Ordering rationale:** Dependency-first. Task 1 establishes the core infrastructure (`userPositions` map, drag event handling for blocks, arrow recomputation during drag, click-vs-drag detection) — both Tasks 2 and 3 depend on this foundation. Task 2 adds a distinct drag behavior (slice group header drag) that reuses the patterns from Task 1. Task 3 adds the reset-layout utility that depends on the `userPositions` concept from Task 1.

**Acceptance criteria coverage:**

| AC | Covered In |
|---|---|
| Individual command and event nodes can be dragged to new positions | Task 1 |
| Edges (arrows) follow their connected nodes when nodes are moved | Task 1 |
| Dragging a slice group header moves all elements within that slice together | Task 2 |
| A "reset layout" button restores the auto-calculated layout | Task 3 |
| Rearranged positions are preserved within the current session (not lost on pan/zoom) | Tasks 1, 2 (via userPositions override mechanism) |

All five acceptance criteria are fully covered with no deferred scope.

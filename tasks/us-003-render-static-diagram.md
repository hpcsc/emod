# US-003: Render a static diagram in the browser

## Progress
- [x] Task 1: Create the HTML viewer skeleton with data loading and SVG canvas
- [x] Task 2: Implement auto-layout and render context swimlanes with slice columns
- [x] Task 3: Render command and event blocks with color coding
- [x] Task 4: Render flow connections as directed arrows
- [ ] Task 5: Display actor annotations and show field details on hover

## Story Reference
`user-stories/web-diagram-viewer.md` — **US-003: Render a static diagram in the browser**

## Codebase Context

**Diagram JSON format (from US-002):**
The viewer consumes the diagram-oriented JSON defined in `internal/export/export.go:587-611`. The format has `model_name`, `nodes` (each with `id`, `type`, `label`, `parentId`, `fields`), and `edges` (each with `source`, `target`, `type`). Node types: `actor`, `context`, `aggregate`, `slice`, `command`, `event`. Edges of type `"flow"` connect command nodes to event nodes.

**Existing layout pattern (Go SVG renderer as layout reference):**
`internal/diagram/drawio.go:12-43` defines layout constants (`marginX`, `marginY`, `sliceWidth`, `boxWidth`, `boxHeight`, `sliceGap`, `contextGap`, `laneHeight`) and color constants (`fillCommand` = `#dae8fc` / blue, `fillEvent` = `#ffe6cc` / orange). `internal/diagram/drawio.go:420-443` implements `collectSlices` (flattens the context → aggregate → slice hierarchy into an ordered list) and `itemLayout` (distributes items horizontally within a slice column). The SVG renderer at `internal/diagram/svg.go:12-364` implements the full layout algorithm: context bounding box tracking, lane assignment, horizontal positioning, and orthogonal arrow pathing.

**Example data:**
`examples/all_patterns.emod` contains a Hotel Reservation model with 2 contexts (Reservations, Notifications), 6 slices across 2 aggregates, multiple commands/events/flows, and 1 actor (Guest). This is the primary test data.

**No existing frontend assets:**
No `.html`, `.js`, or `.css` files exist in the project. This is greenfield.

**Dependency on US-002:**
The diagram JSON format from `export.ExportDiagramJSON` (internal/export/export.go:620-623) is the input to the viewer. The viewer must consume nodes/edges JSON, not the AST-oriented JSON.

---

## Tasks

### Task 1: Create the HTML viewer skeleton with data loading and SVG canvas

**Behavior:** A standalone HTML file that loads diagram-oriented JSON and renders an empty SVG canvas with the model name as a title. The page is self-contained (no external dependencies) and opens in a browser by double-clicking the file.

**Acceptance Criteria:**
- [ ] Single `.html` file exists at `internal/viewer/viewer.html` (or project root `viewer.html`) that opens in a browser via file:// protocol
- [ ] Page displays the model name from the JSON as a page title / header
- [ ] Page contains an `<svg>` element sized to fill the content area (with CSS)
- [ ] Diagram JSON is loaded from a hardcoded JavaScript variable containing the `examples/all_patterns.emod` export output (so the page renders immediately without extra tooling)
- [ ] A `<textarea>` or input mechanism allows pasting alternative diagram JSON for re-rendering
- [ ] JavaScript parses the JSON and stores `nodes` and `edges` arrays in a global-ish data model (or module-scoped variables)
- [ ] Page is visually clean: a reset CSS, sans-serif font, header band, and diagram canvas area

**Affected Files/Modules:**
- `internal/viewer/viewer.html` — New file: complete HTML page with embedded CSS and JavaScript

**Patterns to Follow:**
- SVG element sizing and viewport setup follows the pattern in `internal/diagram/svg.go:368-369` (`svgHeader` — width/height viewBox approach)
- The hardcoded test data should be the JSON output of running `emod export examples/all_patterns.emod --format diagram-json` (or the bare diagram JSON, unwrapped from diagnostics)

**Testable:** No — the HTML file can only be verified by manual inspection in a browser; no automated testing framework is set up for frontend code.

**Verification:** Open the `.html` file in a browser; confirm the model name appears and the SVG canvas is visible (empty).

**Depends on:** None

---

### Task 2: Implement auto-layout and render context swimlanes with slice columns

**Behavior:** The diagram auto-layouts all elements into a coordinate grid. Bounded contexts are rendered as horizontal swimlane bands. Slices within each context/aggregate are rendered as vertical columns, ordered left-to-right as they appear in the model. The layout algorithm computes x/y positions for every node without manual positioning.

**Acceptance Criteria:**
- [ ] Each bounded context is rendered as a horizontal swimlane (a bordered band spanning the diagram width, with the context name as a header)
- [ ] Swimlanes are stacked vertically
- [ ] Slices within each context are rendered as vertical columns, ordered left-to-right matching their order in the model
- [ ] Different contexts have a visual gap separating their swimlanes
- [ ] Each slice column has a header showing the slice name
- [ ] The aggregate name (parent of the slice) is displayed as a sub-header or annotation within the swimlane
- [ ] Layout recalculates when new JSON is pasted into the input mechanism
- [ ] No manual positioning or drag-to-place is required — positions are purely algorithm-driven

**Affected Files/Modules:**
- `internal/viewer/viewer.html` — Add JavaScript functions for: flattening the node hierarchy (context → aggregate → slice), computing x/y/width/height for swimlanes and columns, rendering SVG `<rect>` and `<text>` elements for each swimlane and column

**Patterns to Follow:**
- Flattening logic mirrors `collectSlices` in `internal/diagram/drawio.go:420-430`
- Layout constants (`marginX`, `sliceWidth`, `sliceGap`, `contextGap`, `laneHeight`) adapt the values from `internal/diagram/drawio.go:12-24`
- Context swimlane bounds tracking follows the pattern in `internal/diagram/svg.go:22-56`
- Horizontal item distribution within a slice follows `itemLayout` in `internal/diagram/drawio.go:433-443`

**Testable:** No — visual rendering, verified by opening the HTML file in a browser with the sample data

**Verification:** Open the HTML file; confirm that "Reservations" and "Notifications" appear as horizontal swimlane bands, and slices like "Reserve a Room", "View Available Rooms", etc., appear as labeled vertical columns within their swimlanes.

**Depends on:** Task 1

---

### Task 3: Render command and event blocks with color coding

**Behavior:** Nodes of type `command` are rendered as blue rounded-rectangle blocks and nodes of type `event` as orange rounded-rectangle blocks, positioned within their parent slice column. Each block displays its label and fits within the slice column width.

**Acceptance Criteria:**
- [ ] Command nodes are rendered as blue blocks (`#dae8fc` fill, `#6c8ebf` stroke), positioned within their parent slice's column
- [ ] Event nodes are rendered as orange blocks (`#ffe6cc` fill, `#d79b00` stroke), positioned within their parent slice's column
- [ ] Each block displays its `label` (the command/event name) centered within the block
- [ ] Blocks are vertically distributed within the column so they don't overlap
- [ ] Multiple commands/events within the same slice are evenly distributed horizontally within the column width
- [ ] The parent hierarchy maps correctly: command's `parentId` references a slice node, which is in a context swimlane
- [ ] Node color scheme matches the existing diagram tools (same colors as SVG/drawio output)

**Affected Files/Modules:**
- `internal/viewer/viewer.html` — Add JavaScript rendering functions for command/event blocks; look up parent slice positions from the layout computed in Task 2 to position blocks; compute individual block dimensions

**Patterns to Follow:**
- Color constants (`fillCommand = "#dae8fc"`, `fillEvent = "#ffe6cc"`) match `internal/diagram/drawio.go:28-30`
- Block positioning within a slice follows the pattern in `internal/diagram/svg.go:138-196`
- `itemLayout` in `internal/diagram/drawio.go:433-443` calculates per-item width and x-position when multiple items share a slice

**Testable:** No — visual rendering, verified by opening the HTML file

**Verification:** Open the HTML file; confirm ReserveRoom, CheckOutGuest, ImportExternalReservation, and SendConfirmationEmail appear as blue blocks, and RoomReserved, GuestCheckedOut, ExternalReservationImported appear as orange blocks, correctly positioned within their respective slice columns.

**Depends on:** Task 2

---

### Task 4: Render flow connections as directed arrows

**Behavior:** Edges of type `flow` from the diagram JSON are rendered as directed arrows connecting source command blocks to target event blocks. Arrows use orthogonal routing (right-angle turns) and end with arrowhead markers.

**Acceptance Criteria:**
- [ ] Each edge with `type: "flow"` is rendered as a directed arrow from its source node to its target node
- [ ] Source and target are resolved by matching the edge's `source`/`target` IDs against the rendered nodes' `id` fields
- [ ] Arrows are drawn using orthogonal (right-angle) paths between the center of the source block and the center of the target block
- [ ] Arrow paths have arrowhead markers at the target end
- [ ] Arrow color is neutral (gray/dark) consistent with the existing diagram tools
- [ ] Arrows connect the correct blocks (cross-check: ReserveRoom → RoomReserved, CheckOutGuest → GuestCheckedOut)
- [ ] If a source or target node ID is not found, the edge is silently skipped (graceful degradation)

**Affected Files/Modules:**
- `internal/viewer/viewer.html` — Add JavaScript functions for: resolving edge source/target IDs to rendered node positions, computing orthogonal arrow paths, creating SVG `<path>` elements with arrowhead `<marker>` definitions

**Patterns to Follow:**
- Arrow marker defs (`<marker>` with `<path>`) follow the pattern in `internal/diagram/svg.go:372-379`
- Orthogonal arrow path calculation (vertical-first, midpoint) follows `svgArrowPath` in `internal/diagram/svg.go:429-449`
- Edge resolution logic (lookup by ID from rendered elements) follows the pattern in `internal/diagram/svg.go:252-286`

**Testable:** No — visual rendering, verified by opening the HTML file

**Verification:** Open the HTML file; confirm arrows connect from the ReserveRoom command block to the RoomReserved event block, and from CheckOutGuest to GuestCheckedOut.

**Depends on:** Task 3

---

### Task 5: Display actor annotations and show field details on hover

**Behavior:** Actor nodes from the diagram JSON are displayed as annotations on the diagram (e.g., as a label or badge near the model header or associated swimlane). Hovering or clicking a command or event block reveals its field details (name, type, modifier) as a tooltip or inline popup.

**Acceptance Criteria:**
- [ ] Nodes with `type: "actor"` are displayed as annotations visible on the diagram (e.g., as a badge/banner near the model name or top of the diagram)
- [ ] The actor name is clearly readable (e.g., "Guest")
- [ ] Hovering over a rendered command block displays a tooltip/popup listing its fields: field name, type, and modifier
- [ ] Hovering over a rendered event block displays a tooltip/popup listing its fields: field name, type, and modifier
- [ ] Blocks without fields show no tooltip or a minimal tooltip
- [ ] The tooltip is positioned near the hovered block and does not overflow the viewport
- [ ] Moving the mouse away dismisses the tooltip

**Affected Files/Modules:**
- `internal/viewer/viewer.html` — Add JavaScript for: actor node rendering/annotation, mouse event handlers (mouseenter/mouseleave or click/toggle) for command/event blocks, tooltip HTML element creation and positioning logic

**Patterns to Follow:**
- The existing diagram tools do not handle actor display directly in the same way (actors appear as separate nodes in the JSON but are not embedded in the SVG swimlane layout) — the web viewer has freedom here to choose the right visual treatment (e.g., a "Model Actors" section header or annotation badges)

**Testable:** No — interactive behavior, verified by opening the HTML file and hovering over blocks

**Verification:** Open the HTML file; confirm "Guest" appears as an actor annotation. Hover over ReserveRoom to see a tooltip with roomId (string, required), guestName (string, required), etc. Hover over RoomReserved to see its fields. The tooltip disappears when moving the mouse away.

**Depends on:** Task 3

---

## Summary

**Total tasks:** 5

**Ordering rationale:** Dependency-first, then visual layering. Task 1 creates the file and data pipeline. Task 2 adds the structural layout (swimlanes and columns) — everything else depends on positions. Task 3 renders the colored blocks inside those positions. Task 4 draws arrows between blocks. Task 5 adds annotations and interactivity, which are visual polish on top of the rendered blocks.

**Acceptance criteria coverage:**

| AC | Covered In |
|---|---|
| Bounded contexts as horizontal swimlanes | Task 2 |
| Slices as vertical columns, left-to-right | Task 2 |
| Commands as blue blocks, events as orange | Task 3 |
| Flow connections as directed arrows | Task 4 |
| Actor names as annotations | Task 5 |
| Auto-layout without manual positioning | Task 2 (layout algorithm) |
| Field details visible on hover/click | Task 5 |

All seven acceptance criteria are fully covered with no deferred scope.

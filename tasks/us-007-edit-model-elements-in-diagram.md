# US-007: Edit model elements in the diagram

## Progress
- [x] Task 1: Add inline rename via double-click on nodes
- [x] Task 2: Add context menu on contexts and aggregates for adding slices
- [x] Task 3: Add context menu on slices for adding commands, events, and flows
- [x] Task 4: Add field editing (add, edit, remove) to the detail panel
- [ ] Task 5: Add node deletion

## Story Reference
`user-stories/web-diagram-viewer.md` — **US-007: Edit model elements in the diagram**

## Codebase Context

**Current viewer state:**
`internal/viewer/viewer.html` (~2597 lines, single self-contained HTML+CSS+JS file). The viewer already has pan/zoom (US-004), minimap, detail panel with highlighting, click handlers, tooltips (US-005), and drag support for individual nodes and slice groups (US-006).

**Key state variables:**
- `nodes[]` — flat array of all diagram nodes. Each node has: `id`, `type`, `label`, `parentId`, `fields[]` (for commands/events/views), and type-specific metadata (kind, actor, reads, subscribes, etc.)
- `edges[]` — flat array of all diagram edges. Each edge has: `source`, `target`, `type`.
- `layoutPositions{}` — map of nodeId -> `{x, y, w, h}` populated by `computeLayout()`
- `nodeOffsets{}` — map of nodeId -> `{dx, dy}` for persistent drag offsets
- `selectedNodeId` — node id currently shown in the detail panel

**Detail panel (existing, from US-005):**
- Fixed-position overlay at `#detail-panel` displaying node label, type, fields table (name/type/modifier columns), type-specific metadata sections, and source position
- Dismissed via Escape key, close button, or clicking elsewhere on the diagram
- Fields table is currently read-only (lines 2400-2482)
- The detail panel is rebuilt entirely each time `showDetailPanel()` is called

**Click handling:**
- SVG event delegation at lines 2513-2580 handles left-click on context labels, aggregate labels, flow arrows, and command/event/trigger/view/automation/translation blocks
- `contextmenu` event (right-click) is not currently handled — right-click shows the browser's default context menu
- All interactive SVG elements have `data-node-id`, `data-ctx-id`, `data-agg-id`, `data-slice-id`, or `data-source`/`data-target` attributes

**Node structures:**
- `jsonDiagramField` (Go backend, lines 618-622): `{name, type, modifier}` — fields are identified by their index in the array (no unique field ID)
- Fields are rendered in the detail panel table by index
- Node labels are stored in `node.label` and rendered as SVG text in `renderDiagram()`

**Rendering flow:**
- `renderDiagram()` calls `computeLayout()`, generates SVG HTML, sets it via `svg.innerHTML`, then post-processes (annotations, handlers, viewport)
- Re-rendering the full diagram is the primary mechanism for reflecting edits visually

**Key constraint:**
No test infrastructure exists for the viewer — it's pure browser JS. All verification is manual in-browser testing.

---

## Tasks

### Task 1: Add inline rename via double-click on nodes

**Behavior:** Double-clicking any diagram node (command, event, trigger, view, automation, translation, context, aggregate, or slice) opens an inline text editor directly over the node's label. Pressing Enter or blurring the editor commits the new label to the node's data and re-renders the diagram immediately. Pressing Escape cancels the edit and restores the original label.

**Acceptance Criteria:**
- [ ] Double-clicking a node replaces its label text with an editable input field positioned over the label
- [ ] Pressing Enter or clicking outside the input commits the new label and re-renders the diagram
- [ ] Pressing Escape cancels the edit and restores the original label
- [ ] The node's `label` property in the `nodes[]` array is updated after commit
- [ ] The diagram reflects the new label immediately after the edit (no page reload)
- [ ] Works for all node types: contexts, aggregates, slices, commands, events, triggers, views, automations, translations
- [ ] Double-click on a node does not interfere with single-click behaviors (detail panel, highlighting)
- [ ] Detail panel and highlights are dismissed when entering inline edit mode

**Affected Files/Modules:**
- `internal/viewer/viewer.html` — Add `dblclick` event handler on SVG node elements via event delegation; create inline edit mechanism (positioned `<input>` or `contenteditable` using an SVG `<foreignObject>` or a positioned HTML element); handle commit (Enter/blur) and cancel (Escape); update `node.label` in `nodes[]` array; call `renderDiagram()` after commit

**Patterns to Follow:**
- Follow the existing SVG click delegation at lines 2513-2580 for adding the `dblclick` handler — detect the target node via `data-node-id`, `data-ctx-id`, `data-agg-id`, or `data-slice-id` attributes, then find the corresponding node in the `nodes[]` array using `findNodeById()` at line 2369
- Follow the tooltip positioning pattern at lines 1248-1263 for placing the inline editor input element relative to the SVG node's bounding box
- Follow the Escape key dismissal pattern at lines 2583-2588 for canceling the edit
- Follow the detail panel dismissal at line 2484 (`hideDetailPanel()`) for clearing selection state before editing

**Testable:** No — interactive browser behavior, verified by manual testing

**Verification:** Serve the viewer with a diagram. Double-click on a command block — its label becomes editable. Type a new name, press Enter — the label updates. Double-click again, type, press Escape — the original label is restored. Repeat for each node type.

**Depends on:** None

---

### Task 2: Add context menu on contexts and aggregates for adding slices

**Behavior:** Right-clicking on a context swimlane header or an aggregate label shows a custom context menu with an "Add Slice" option. Clicking "Add Slice" creates a new slice node inside the target aggregate with a default name, places it in the diagram, and re-renders immediately. The browser's default context menu is suppressed on these targets.

**Acceptance Criteria:**
- [ ] Right-clicking on a context swimlane header (`.ctx-label`) or an aggregate label (`.agg-label`) shows a custom context menu with "Add Slice"
- [ ] The browser's default context menu is suppressed on these target elements
- [ ] Clicking "Add Slice" creates a new slice node with `type: "slice"`, a generated label (e.g., "new-slice"), and the correct `parentId` set to the aggregate's ID
- [ ] The new slice appears in the diagram immediately (no page reload)
- [ ] Clicking outside the context menu dismisses it
- [ ] The context menu is positioned at the right-click location
- [ ] Right-clicking elsewhere on the diagram (not on a context/aggregate) still shows the browser's default context menu

**Affected Files/Modules:**
- `internal/viewer/viewer.html` — Add `contextmenu` event listener on the SVG; detect if the target is a `.ctx-label` or `.agg-label` element (via `closest()`); prevent default browser menu; create a positioned floating context menu `<div>` with "Add Slice" option; handle the click to create a new slice node in `nodes[]`; dismiss menu on outside click; call `renderDiagram()` after slice creation

**Patterns to Follow:**
- Follow the context label click detection at lines 2521-2530 for identifying which context/aggregate was right-clicked — use `ctxLabel.getAttribute("data-ctx-id")` and `aggLabel.getAttribute("data-agg-id")` to determine the parent ID
- Follow the tooltip/detail panel positioning at lines 1248-1263 for the context menu overlay positioning (position fixed at right-click coordinates)
- Follow the detail panel overlay styling at lines 531-581 for the context menu CSS (fixed position, z-index, border, shadow, rounded corners)
- Follow the `buildTree()` function at lines 872-884 for understanding the parent-child relationships needed to determine which aggregate a context belongs to (context → aggregate → slice)
- Follow the existing node structure: a slice node needs `id` (generated), `type: "slice"`, `label`, `parentId` set to the target aggregate ID, and no `fields`

**Testable:** No — interactive browser behavior, verified by manual testing

**Verification:** Serve the viewer with a diagram containing at least one context with an aggregate. Right-click on the context header — "Add Slice" appears. Click it — a new slice appears inside the aggregate. Right-click on the aggregate label — same behavior.

**Depends on:** None

---

### Task 3: Add context menu on slices for adding commands, events, and flows

**Behavior:** Right-clicking on a slice's background area (not on a specific block) shows a custom context menu with options to "Add Command", "Add Event", and "Add Flow". Selecting an option creates the corresponding node (and edge for flows) inside the slice and re-renders immediately. The browser's default context menu is suppressed on slice targets.

**Acceptance Criteria:**
- [ ] Right-clicking on a slice background area shows a custom context menu with "Add Command", "Add Event", and "Add Flow" options
- [ ] The browser's default context menu is suppressed on slice targets
- [ ] "Add Command" creates a new command node with `type: "command"`, a generated label, and `parentId` set to the slice's ID
- [ ] "Add Event" creates a new event node with `type: "event"`, a generated label, and `parentId` set to the slice's ID
- [ ] "Add Flow" creates a new event node AND a new flow edge (from the most recent command to the new event, or a standalone event) — follow the pattern that a flow connects a command to an event
- [ ] All new nodes appear in the diagram immediately (no page reload)
- [ ] Right-clicking on a block element (command/event/etc.) inside the slice does NOT trigger the slice context menu
- [ ] Clicking outside the context menu dismisses it
- [ ] The context menu is positioned at the right-click location

**Affected Files/Modules:**
- `internal/viewer/viewer.html` — Extend the `contextmenu` handler from Task 2 to detect `.slice-header` target elements (and potentially the slice background area); show a different menu with "Add Command", "Add Event", "Add Flow" options; handle creation of new command/event nodes in `nodes[]` and flow edges in `edges[]`; call `renderDiagram()` after creation

**Patterns to Follow:**
- Follow the context menu infrastructure from Task 2 (positioning, styling, dismissal)
- Follow the slice header detection at lines 1816-1848 for determining which slice was targeted — detect via `data-slice-id` attribute
- The slice background (the dashed-border rectangle rendered at lines 1107-1108) may need a data attribute added for right-click detection, or use the existing `<g class="slice-{id}">` container
- Follow the existing `edges[]` entry structure: a flow edge has `{source: cmdId, target: evtId, type: "flow"}`
- For "Add Flow", determine the source: either use the most recently added command in the slice, or the last command in the slice's order — follow how drag state `getSliceChildNodeIds()` at lines 1432-1440 identifies child nodes within a slice

**Testable:** No — interactive browser behavior, verified by manual testing

**Verification:** Serve the viewer with a diagram containing at least one slice. Right-click on the slice background — the context menu shows "Add Command", "Add Event", "Add Flow". Click "Add Command" — a new command block appears in the slice. Click "Add Event" — a new event block appears. Right-click again and select "Add Flow" — a new event and flow arrow appear.

**Depends on:** Task 2

---

### Task 4: Add field editing (add, edit, remove) to the detail panel

**Behavior:** The detail panel for commands and events gains editable field rows. Each row has inline text inputs for the field name, type, and an optional modifier. An "Add Field" button appends a new row. A delete button per row removes that field. All changes update the in-memory `nodes[]` array and re-render the field table immediately. The diagram is re-rendered when fields change so the tooltip and any field-dependent display stays in sync.

**Acceptance Criteria:**
- [ ] The detail panel's fields table for commands and events shows editable inputs instead of read-only text for name, type, and modifier columns
- [ ] An "Add Field" button is present at the bottom of the fields section
- [ ] Clicking "Add Field" appends a new empty row with default values
- [ ] Each field row has a delete button that removes that field
- [ ] Changes to field name, type, or modifier update the corresponding field in `nodes[]` on blur (or on input with debounce)
- [ ] New fields are added to the `fields[]` array on the corresponding node in `nodes[]`
- [ ] Deleting a field removes it from the `fields[]` array
- [ ] The diagram re-renders after field changes so tooltips reflect updated fields
- [ ] Fields on both command and event nodes are editable
- [ ] Non-field node types (triggers, views, automations, translations) still show read-only metadata sections
- [ ] The detail panel remains open and visible during editing (not dismissed by incidental clicks)

**Affected Files/Modules:**
- `internal/viewer/viewer.html` — Modify `showDetailPanel()` at lines 2400-2482 to render editable field inputs instead of read-only text; add "Add Field" button and per-row delete buttons; attach event handlers for field input changes (blur or input events); implement functions to add/update/remove fields from the node's `fields[]` array; call `renderDiagram()` after field mutations to sync tooltips

**Patterns to Follow:**
- Follow the existing fields table rendering at lines 2411-2421 for the layout structure (table with three columns: Field, Type, Modifier)
- Follow the existing detail panel HTML structure at lines 549-581 for consistent styling of the editing UI
- Follow the existing node data access pattern at line 2567 (`findNodeById(nodeId)`) to locate the node being edited
- Follow the existing `showDetailPanel()` pattern at lines 2400-2482 — the panel content is rebuilt each time from scratch, so incorporate editable inputs into the HTML string construction

**Testable:** No — interactive browser behavior, verified by manual testing

**Verification:** Serve the viewer with a diagram containing a command with defined fields. Click the command to open the detail panel. Edit a field name in the input — the detail panel stays open. Add a new field via "Add Field" — a new row appears. Delete a field — the row is removed. Close and re-open the detail panel — the changes are reflected. Hover over the command — the tooltip shows the updated fields.

**Depends on:** None (depends on the existing detail panel from US-005, which is already completed)

---

### Task 5: Add node deletion

**Behavior:** A selected node can be deleted via the Delete/Backspace key or a delete button in the detail panel. Deleting a node removes it from the `nodes[]` array and removes all edges connected to it from the `edges[]` array. The diagram re-renders immediately without the deleted element.

**Acceptance Criteria:**
- [ ] Pressing the Delete or Backspace key while a node is selected (detail panel is open) deletes that node
- [ ] A delete button is present in the detail panel header area, visible when a deletable node is selected
- [ ] Deleting a node removes it from the `nodes[]` array
- [ ] All edges where `source` or `target` matches the deleted node's ID are removed from `edges[]`
- [ ] The diagram re-renders immediately without the deleted element (no page reload)
- [ ] The detail panel closes after deletion
- [ ] Only block-level nodes with edges are deletable (commands, events, triggers, views, automations, translations)
- [ ] Deleting a translation's inline event node (which has no edges) removes the event node cleanly
- [ ] The reset layout button state is updated after deletion (nodeOffsets for the deleted node are cleaned up)

**Affected Files/Modules:**
- `internal/viewer/viewer.html` — Add Delete/Backspace key handler (extend the existing `keydown` listener at lines 2583-2588); add delete button to the detail panel HTML (similar to the close button at lines 565-579); implement `deleteNode(nodeId)` function that removes the node from `nodes[]`, removes connected edges from `edges[]`, cleans up `nodeOffsets` and `layoutPositions`, calls `renderDiagram()`, and updates reset layout button state

**Patterns to Follow:**
- Follow the Escape key dismissal at lines 2583-2588 for the Delete key handler addition — extend the same `keydown` listener with a new `evt.key === "Delete" || evt.key === "Backspace"` branch
- Follow the detail panel close button at lines 565-579 and 2591-2594 for the delete button placement and event wiring
- Follow `getConnectedEdges()` at lines 1319-1327 for finding all edges connected to a node (though deletion needs to filter for ALL edge types, not just specific ones)
- Follow `updateResetLayoutButton()` at lines 1674-1679 for resetting button state after deletion
- Follow the node ID generation pattern (e.g., `cmd-1`, `evt-3`) to ensure consistency — deletion is purely in-memory so no new IDs are generated, but understanding the pattern helps with any edge cases

**Testable:** No — interactive browser behavior, verified by manual testing

**Verification:** Serve the viewer with a diagram containing a command connected to an event via a flow edge. Click the command to open the detail panel. Press Delete — the command and its flow edge disappear. The detail panel closes. Verify the remaining diagram still renders correctly.

**Depends on:** None (depends on existing detail panel from US-005 and the existing selectedNodeId mechanism, both already completed)

---

## Summary

**Total tasks:** 5

**Ordering rationale:** Dependency-first where possible, risk-first otherwise.
- Task 1 (inline rename) and Task 2 (context menu for adding slices) have no dependencies and can be implemented in parallel.
- Task 3 (context menu for adding commands/events/flows) depends on Task 2 because both share the context menu infrastructure — Task 2 builds the context menu foundation, Task 3 extends it with more options.
- Task 4 (field editing in detail panel) is independent of the other editing tasks — it only modifies the detail panel internals.
- Task 5 (node deletion) is also independent but conceptually simpler and can be done after the other editing foundations are in place.

**Acceptance criteria coverage:**

| AC | Covered In |
|---|---|
| Double-clicking a node opens an inline editor to rename it | Task 1 |
| A context menu (right-click) on a slice offers options to add a new command, event, or flow | Task 3 |
| A context menu on a context or aggregate offers an option to add a new slice | Task 2 |
| Fields on commands and events can be added, edited (name, type, modifier), and removed via the detail panel | Task 4 |
| Deleting a node removes it and all its connected edges from the diagram | Task 5 |
| All edits are reflected immediately in the diagram without a page reload | Tasks 1–5 (each task ensures immediate re-render) |

All six acceptance criteria are fully covered with no deferred scope.

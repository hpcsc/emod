# US-004: Pan, zoom, and navigate the diagram

## Progress
- [x] Task 1: Add viewport state management and SVG transform mechanism
- [x] Task 2: Add mouse wheel zoom centered on cursor position
- [x] Task 3: Add click-and-drag pan on diagram background
- [ ] Task 4: Add fit-to-view control
- [ ] Task 5: Add pinch-to-zoom for touch devices and trackpads
- [ ] Task 6: Add minimap showing current viewport position within the full diagram

## Story Reference
`user-stories/web-diagram-viewer.md` — **US-004: Pan, zoom, and navigate the diagram**

## Codebase Context

**Current viewer state:**
`internal/viewer/viewer.html` is a single self-contained HTML file (~779 lines) with embedded CSS and JS. The SVG uses a viewBox set dynamically by `renderDiagram()` after `computeLayout()` returns `{width, height}`. Currently the viewer has no zoom, pan, minimap, or navigation features.

**Key existing functions:**
- `computeLayout()` (lines 426-509) — computes x/y/width/height for all nodes, returns total dimensions
- `renderDiagram()` (lines 512-621) — clears the SVG via `svg.innerHTML = html`, sets the viewBox, renders all visual elements, then re-attaches event handlers
- `clearSVG()` (lines 737-741) — clears SVG content while preserving `<defs>`
- `attachBlockHandlers()` (lines 685-706) — attaches mouseenter/mousemove/mouseleave to command/event blocks for tooltips

**DOM structure:**
- `<div id="canvas-container">` (flex:1, overflow:hidden) wraps the `<svg id="diagram-canvas">`
- The SVG has `viewBox` set dynamically; CSS sets `width: 100%; height: 100%`
- All diagram content (swimlanes, columns, blocks, arrows) is rendered directly as SVG children

**Rendering approach:**
`renderDiagram()` writes to `svg.innerHTML` each time, which destroys and recreates all child elements. Any viewport group element must either be re-created on each render (with the transform re-applied) or managed by preserving it across innerHTML writes.

**No existing test infrastructure:**
The viewer is a browser-only HTML file with no test framework. All verification is manual in-browser testing.

**Relevant patterns from the codebase:**
- The existing `mouseenter`/`mousemove`/`mouseleave` pattern in `attachBlockHandlers()` shows how interactive handlers are re-attached after each render
- The `computeLayout()` → `renderDiagram()` flow is the hook point where viewport state should be applied after rendering

---

## Tasks

### Task 1: Add viewport state management and SVG transform mechanism

**Behavior:** Introduce viewport state variables (offsetX, offsetY, zoomScale) and wrap all rendered diagram content in a `<g>` element with a `transform` attribute derived from that state. This provides the foundation for all subsequent pan/zoom operations to work without re-rendering the diagram.

**Acceptance Criteria:**
- [ ] Viewport state is held in module-scoped variables: `viewport = { offsetX: 0, offsetY: 0, zoomScale: 1 }`
- [ ] `renderDiagram()` wraps all rendered SVG content inside a `<g id="viewport-group">` element (re-created on each render)
- [ ] An `applyViewport()` function computes `transform="translate(offsetX, offsetY) scale(zoomScale)"` and sets it on the viewport group
- [ ] `applyViewport()` is called at the end of `renderDiagram()` so the transform is applied after every render
- [ ] A helper `screenToDiagram(screenX, screenY)` function converts screen coordinates to diagram coordinates accounting for the current transform
- [ ] A helper `clampZoom(scale)` function constrains zoomScale to a defined range (e.g., 0.1 to 5.0)
- [ ] Changing viewport state and calling `applyViewport()` updates the visible diagram without calling `renderDiagram()`

**Affected Files/Modules:**
- `internal/viewer/viewer.html` — Add viewport state variables after line 378; modify `renderDiagram()` (around line 512) to wrap content in `<g id="viewport-group">`; add `applyViewport()`, `screenToDiagram()`, and `clampZoom()` functions

**Patterns to Follow:**
- The post-render handler re-attachment pattern in `renderDiagram()` (lines 619-620) — `applyViewport()` should be called alongside `renderActorAnnotations()` and `attachBlockHandlers()` after the SVG content is built

**Testable:** No — visual behavior, verified by opening the HTML file in a browser and inspecting the SVG transform attribute in developer tools

**Verification:** Open the HTML file; inspect the SVG in dev tools to confirm viewport-group exists with `transform="translate(0, 0) scale(1)"`. Call `applyViewport()` from the console with modified state and confirm the transform attribute updates.

**Depends on:** None

---

### Task 2: Add mouse wheel zoom centered on cursor position

**Behavior:** Scrolling the mouse wheel over the diagram canvas zooms in and out, keeping the point under the cursor stationary. This makes it feel like zooming toward or away from whatever the user is pointing at.

**Acceptance Criteria:**
- [ ] A `wheel` event listener is attached to the SVG or canvas container
- [ ] Scrolling up (negative deltaY) increases zoomScale; scrolling down (positive deltaY) decreases zoomScale
- [ ] Zoom is centered on the cursor position — the diagram point under the cursor stays fixed as the zoom level changes
- [ ] zoomScale is clamped to the range [0.1, 5.0]
- [ ] Browser default scroll behavior is prevented (page does not scroll when zooming the diagram)
- [ ] The zoom feels responsive and updates the transform immediately via `applyViewport()`
- [ ] The event listener is attached once (not re-attached on every render) — use event delegation or attach to a persistent parent

**Affected Files/Modules:**
- `internal/viewer/viewer.html` — Add `wheel` event handler on `#canvas-container` or `#diagram-canvas`; implement cursor-centric zoom math using `screenToDiagram()` from Task 1

**Patterns to Follow:**
- Event delegation on a persistent container element follows the same approach as the persistent listeners already in the HTML (e.g., `renderBtn.addEventListener("click", render)` at line 764)
- Post-render handler re-attachment in `renderDiagram()` (lines 619-620) — the wheel handler should be attached once externally, not inside `attachBlockHandlers()`

**Testable:** No — interactive behavior, verified by manual testing in browser

**Verification:** Open the HTML file with a rendered diagram. Scroll up — the diagram zooms in centered on the cursor. Scroll down — it zooms out. The cursor position stays fixed relative to the diagram content. Zoom is clamped at the limits.

**Depends on:** Task 1

---

### Task 3: Add click-and-drag pan on diagram background

**Behavior:** Clicking and dragging on the empty diagram background (not on a command/event block) pans the viewport, letting users explore different areas of large diagrams.

**Acceptance Criteria:**
- [ ] `mousedown` on the SVG or canvas container initiates pan mode (only when the target is the viewport group or SVG background, not a command/event block)
- [ ] `mousemove` during pan mode updates `viewport.offsetX` and `viewport.offsetY` and calls `applyViewport()` for real-time visual feedback
- [ ] `mouseup` ends pan mode
- [ ] Cursor changes to `grab` when hovering over the background and `grabbing` during an active pan drag
- [ ] Dragging on interactive elements (`.cmd-block`, `.evt-block`) does NOT trigger pan — those elements show the pointer cursor and handle their own click events
- [ ] Pan works smoothly without jitter or lag

**Affected Files/Modules:**
- `internal/viewer/viewer.html` — Add `mousedown`, `mousemove`, `mouseup` event handlers; track pan state (isPanning, lastX, lastY) in a module-scoped variable; update cursor style via CSS or inline style

**Patterns to Follow:**
- The `mouseenter`/`mouseleave` event pattern in `attachBlockHandlers()` (lines 685-706) demonstrates how to distinguish interactive children from the background
- Global `mousemove`/`mouseup` on `document` (not just the SVG) follows standard drag patterns to prevent losing the drag when the mouse leaves the element

**Testable:** No — interactive behavior, verified by manual testing in browser

**Verification:** Open the HTML file with a diagram larger than the viewport. Click and drag on empty space — the diagram pans following the mouse. Cursor changes to `grabbing` during drag. Clicking a command block selects it instead of panning.

**Depends on:** Task 1

---

### Task 4: Add fit-to-view control

**Behavior:** A visible control button resets the viewport to show the entire diagram, computing the optimal zoom and offset so all content fits within the available canvas area.

**Acceptance Criteria:**
- [ ] A "Fit to view" button is rendered as a UI control (e.g., in the header bar or as an overlay button in the canvas corner)
- [ ] Clicking the button calculates the zoom scale needed to fit the full diagram width/height within the container dimensions
- [ ] The viewport is centered after fit-to-view (offsetX/offsetY adjusted to center the diagram)
- [ ] All diagram content is visible after fit-to-view
- [ ] If the diagram is already fully visible, fit-to-view is still a no-op (does not break)
- [ ] The control is always accessible regardless of zoom/pan state

**Affected Files/Modules:**
- `internal/viewer/viewer.html` — Add "Fit to view" button HTML in the header or as an overlay in `#canvas-container`; add `fitToView()` function that computes optimal zoom/offset and calls `applyViewport()`

**Patterns to Follow:**
- The existing `#panel-toggle` button (lines 72-88) shows how to style an overlay control in the canvas
- The existing header buttons/layout (lines 29-55) is an alternative placement location

**Testable:** No — interactive behavior, verified by manual testing in browser

**Verification:** Open the HTML file with a large diagram. Zoom in and pan to a corner. Click "Fit to view" — the entire diagram is visible and centered in the canvas.

**Depends on:** Task 1

---

### Task 5: Add pinch-to-zoom for touch devices and trackpads

**Behavior:** Two-finger pinch gestures on touch devices and trackpads zoom the diagram in and out, centered on the pinch midpoint. Single-finger drag on the background pans the viewport via touch.

**Acceptance Criteria:**
- [ ] `touchstart` listener records initial touch points and detects multi-touch
- [ ] Two-finger `touchmove` calculates the change in distance between fingers and adjusts zoomScale proportionally
- [ ] Zoom is centered on the midpoint of the two touch points
- [ ] A single finger dragged on the background pans the viewport (same behavior as Task 3's mouse pan)
- [ ] `touchend` ends the gesture cleanly
- [ ] Default touch behavior (scroll, zoom) is prevented on the canvas container
- [ ] Works on trackpads that emit TouchEvents (macOS Safari/Chrome) and on mobile devices
- [ ] Transition between one-finger pan and two-finger zoom is smooth without jarring jumps

**Affected Files/Modules:**
- `internal/viewer/viewer.html` — Add `touchstart`, `touchmove`, `touchend` event handlers; track touch state (active touches, initial distance, midpoint); implement pinch-zoom math; prevent default touch actions via CSS `touch-action: none` on the canvas container

**Patterns to Follow:**
- The zoom-centering math follows the same `screenToDiagram()` approach used in Task 2 for cursor-centric zoom
- The pan-on-drag logic mirrors the mouse pan from Task 3 but adapted for `TouchEvent` coordinates

**Testable:** No — interactive behavior, verified by manual testing on a touch device or trackpad

**Verification:** Open the HTML file on a touch-capable device. Pinch with two fingers — the diagram zooms in/out centered on the pinch point. Drag with one finger — the diagram pans. On a trackpad, the same gestures work.

**Depends on:** Task 1

---

### Task 6: Add minimap showing current viewport position within the full diagram

**Behavior:** A small minimap in the corner of the canvas shows an overview of the full diagram with a rectangle overlay indicating the current visible viewport area. The minimap updates in real-time as the user pans and zooms.

**Acceptance Criteria:**
- [ ] A minimap element is rendered as a fixed overlay in one corner of `#canvas-container` (e.g., bottom-left)
- [ ] The minimap displays a scaled-down representation of the full diagram bounds (a simple filled rectangle representing the total diagram area, or a miniature rendering of the swimlanes)
- [ ] A rectangle overlay on the minimap indicates the current viewport position and visible area relative to the full diagram
- [ ] The viewport rectangle updates in real-time as the user pans and zooms
- [ ] Clicking or dragging on the minimap repositions the main viewport to the clicked location
- [ ] The minimap has a subtle border and semi-transparent background so it doesn't obstruct the diagram
- [ ] The minimap can be toggled on/off via a control button

**Affected Files/Modules:**
- `internal/viewer/viewer.html` — Add minimap HTML element (an SVG or `<div>` with positioned elements); add CSS styling for fixed positioning and sizing; add JavaScript functions to: create/update the minimap, calculate viewport rectangle position/size from viewport state and diagram dimensions, handle click/drag on minimap to reposition the main viewport

**Patterns to Follow:**
- The overlay positioning pattern follows `#panel-toggle` (lines 72-88) and `#tooltip` (lines 254-268) for fixed/absolute positioning within `#canvas-container`
- The z-index layering follows the existing overlay stack (tooltip at z-index 1000, panel-toggle at 20)

**Testable:** No — visual/interactive behavior, verified by manual testing in browser

**Verification:** Open the HTML file with a large diagram. Confirm the minimap appears in the corner showing a scaled representation of the full diagram with a viewport rectangle. Pan and zoom — the rectangle updates position/size. Click on a different area of the minimap — the main viewport jumps to that area.

**Depends on:** Task 1

---

## Summary

**Total tasks:** 6

**Ordering rationale:** Dependency-first, then by interaction modality. Task 1 establishes the foundational viewport state and transform mechanism — every subsequent task depends on it. Tasks 2 and 3 add the two primary mouse interactions (wheel zoom and click-drag pan) as separate, independently verifiable behaviors. Task 4 adds a utility control (fit-to-view) that's simple but requires viewport state from Task 1. Task 5 adds touch interactions which share zoom math with Task 2 but require separate event handling and don't depend on the mouse handlers. Task 6 (minimap) is last because it's a standalone feature that depends only on viewport state and diagram bounds.

**Acceptance criteria coverage:**

| AC | Covered In |
|---|---|
| Mouse wheel zooms in/out, centered on cursor position | Task 2 |
| Click-and-drag on background pans the viewport | Task 3 |
| A "fit to view" control resets the viewport | Task 4 |
| Pinch-to-zoom on trackpads and touch devices | Task 5 |
| Minimap shows current viewport position | Task 6 |

All five acceptance criteria are fully covered with no deferred scope.

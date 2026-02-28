# Interactive Web Diagram Viewer

## Overview

Provide an interactive, browser-based viewer that renders `.emod` files as event modeling diagrams. The viewer displays bounded contexts as horizontal swimlanes, slices as vertical columns, and color-coded blocks for commands, events, and other elements — following the standard event modeling visual convention. Users can pan, zoom, inspect details, rearrange elements, and edit the model directly in the diagram with changes exported back to `.emod` format. The viewer works both as a standalone web application (upload/paste files) and when served locally by the emod CLI.

## Goals

- Render parsed `.emod` models as interactive event modeling diagrams in the browser
- Support the full inspect-rearrange-edit cycle: view details, move things around, modify the model, and export changes
- Provide two JSON export formats: a raw AST serialization and a diagram-oriented representation with nodes and edges
- Work standalone (static site with file upload) and via the CLI (`emod diagram --serve`)
- Gracefully handle `.emod` files with parse errors by rendering the valid portion alongside diagnostics

## User Stories

### US-001: Export model as raw AST JSON
**Description:** As a tool author or AI agent, I want to export the parsed AST as a JSON document that mirrors the internal model structure so that I can consume the full model programmatically without loss of information.

**Acceptance Criteria:**
- [ ] `emod export <file> --format json` outputs a JSON document to stdout representing the complete AST
- [ ] The JSON structure preserves the full hierarchy: model name, actors, contexts, aggregates, slices, commands, events, fields, and flows
- [ ] Field types and modifiers (e.g. `required`, `optional`) are included
- [ ] Source positions (file, line, column) are included for each named element
- [ ] If the file has parse errors, the command still outputs JSON for the successfully parsed portion and includes a `diagnostics` array with file, line, and message for each error
- [ ] `emod export <file> --format json` with a fully valid file produces an empty `diagnostics` array

---

### US-002: Export model as diagram-oriented JSON
**Description:** As a diagram renderer, I want a JSON format specifically structured as nodes and edges so that I can feed it directly into a graph layout engine without transforming the AST myself.

**Acceptance Criteria:**
- [ ] `emod export <file> --format diagram-json` outputs a JSON document with top-level `nodes` and `edges` arrays
- [ ] Each node has an `id`, `type` (one of: `actor`, `context`, `aggregate`, `slice`, `command`, `event`), `label`, and `parentId` (referencing its containing node, or null for top-level)
- [ ] Each edge has a `source` node ID, `target` node ID, and `type` (e.g. `flow` for command→event connections)
- [ ] Field details for commands and events are included as a `fields` property on the respective nodes
- [ ] The model name is included as metadata at the top level
- [ ] Parse errors are included in a `diagnostics` array, and successfully parsed elements are still emitted as nodes/edges

**Depends on:** US-001

---

### US-003: Render a static diagram in the browser
**Description:** As a model author, I want to see my `.emod` model rendered as a visual event modeling diagram in the browser so that I can understand the system's structure at a glance.

**Acceptance Criteria:**
- [ ] Given diagram-oriented JSON, the web viewer renders bounded contexts as horizontal swimlanes
- [ ] Slices within each context/aggregate are laid out as vertical columns, ordered left-to-right as they appear in the model
- [ ] Commands are rendered as blue blocks, events as orange blocks
- [ ] Flow connections (command → event) are rendered as directed arrows between blocks
- [ ] Actor names are displayed as annotations associated with their model
- [ ] The diagram auto-layouts without manual positioning using a graph layout algorithm
- [ ] Field details are visible on each command and event block (either inline or on hover/click)

**Depends on:** US-002

---

### US-004: Pan, zoom, and navigate the diagram
**Description:** As a model author, I want to pan and zoom the diagram so that I can explore large models that don't fit on a single screen.

**Acceptance Criteria:**
- [ ] Mouse wheel zooms in and out, centered on the cursor position
- [ ] Click-and-drag on the background pans the viewport
- [ ] A "fit to view" control resets the viewport to show the entire diagram
- [ ] Pinch-to-zoom works on trackpads and touch devices
- [ ] A minimap shows the current viewport position within the full diagram

**Depends on:** US-003

---

### US-005: Inspect node details
**Description:** As a model author, I want to click on a diagram element to see its full details so that I can review field definitions, types, and modifiers without reading the source file.

**Acceptance Criteria:**
- [ ] Clicking a command or event node opens a detail panel showing its name and all fields with types and modifiers
- [ ] Clicking a context or aggregate node highlights all elements within it
- [ ] Clicking a flow arrow highlights both the source command and target event
- [ ] The detail panel shows the source file location (file name and line number) for the selected element
- [ ] Pressing Escape or clicking elsewhere dismisses the detail panel

**Depends on:** US-004

---

### US-006: Rearrange diagram elements by dragging
**Description:** As a model author, I want to drag nodes to rearrange the diagram layout so that I can organize the visual presentation to match my mental model.

**Acceptance Criteria:**
- [ ] Individual command and event nodes can be dragged to new positions
- [ ] Edges (arrows) follow their connected nodes when nodes are moved
- [ ] Dragging a slice group header moves all elements within that slice together
- [ ] A "reset layout" button restores the auto-calculated layout
- [ ] Rearranged positions are preserved within the current session (not lost on pan/zoom)

**Depends on:** US-004

---

### US-007: Edit model elements in the diagram
**Description:** As a model author, I want to modify my model directly in the diagram — adding, renaming, and removing elements — so that I can iterate on the design visually without switching to a text editor.

**Acceptance Criteria:**
- [ ] Double-clicking a node opens an inline editor to rename it
- [ ] A context menu (right-click) on a slice offers options to add a new command, event, or flow within that slice
- [ ] A context menu on a context or aggregate offers an option to add a new slice
- [ ] Fields on commands and events can be added, edited (name, type, modifier), and removed via the detail panel
- [ ] Deleting a node removes it and all its connected edges from the diagram
- [ ] All edits are reflected immediately in the diagram without a page reload

**Depends on:** US-005, US-006

---

### US-008: Export edited model back to .emod format
**Description:** As a model author, I want to export the diagram back to `.emod` syntax so that changes I make visually are persisted in the source file.

**Acceptance Criteria:**
- [ ] An "Export .emod" action generates valid `.emod` syntax reflecting the current state of the diagram including any edits
- [ ] The exported file uses consistent formatting (2-space indent, one blank line between slices)
- [ ] Elements are output in the order they appear in the diagram (left-to-right slices, top-to-bottom within slices)
- [ ] The exported `.emod` file can be parsed by `emod validate` without errors
- [ ] The export is offered as a file download in the browser

**Depends on:** US-007

---

### US-009: Standalone web viewer with file upload
**Description:** As a model author without the emod CLI installed, I want to open the web viewer in my browser and upload or paste a `.emod` file so that I can view diagrams without any local tooling.

**Acceptance Criteria:**
- [ ] The web viewer is a static site that works without a backend server
- [ ] A landing page offers two options: upload a `.emod` file or paste `.emod` content into a text area
- [ ] After uploading or pasting, the file is parsed client-side and the diagram renders
- [ ] The viewer also accepts diagram-oriented JSON as input (for users who have pre-exported)
- [ ] No file data leaves the browser — all parsing and rendering happens client-side

**Context:** For client-side parsing, the Go parser would need to be compiled to WebAssembly, or a compatible parser implemented in TypeScript. This story focuses on the user-facing behavior; the parsing mechanism is an implementation decision.

**Depends on:** US-003

---

### US-010: Serve the viewer locally via the CLI
**Description:** As a model author with the emod CLI installed, I want to run a single command that opens the diagram viewer in my browser with my `.emod` file already loaded so that I get a zero-friction path from source file to visual diagram.

**Acceptance Criteria:**
- [ ] `emod diagram --serve` starts a local HTTP server and opens the default browser to the viewer
- [ ] When run without a file argument, the viewer opens with the file upload/paste interface
- [ ] When run with a file argument (`emod diagram --serve myfile.emod`), the viewer opens with that file already rendered
- [ ] The server shuts down cleanly when the user presses Ctrl+C in the terminal
- [ ] The terminal displays the local URL where the viewer is accessible

**Depends on:** US-009

---

### US-011: Display parse errors alongside partial diagram
**Description:** As a model author, I want to see a partial diagram for `.emod` files that have parse errors so that I can understand what the parser was able to interpret and where the problems are.

**Acceptance Criteria:**
- [ ] When a `.emod` file has parse errors, the successfully parsed portion renders as a diagram
- [ ] A diagnostics panel displays all parse errors with file name, line number, and error description
- [ ] Clicking a diagnostic in the panel highlights the corresponding location in the diagram (if the element was partially parsed) or indicates it could not be rendered
- [ ] The diagnostics panel is dismissible but accessible via a persistent error badge showing the count of errors
- [ ] A fully valid file shows no diagnostics panel or error badge

**Depends on:** US-003

## Non-Goals

- Real-time collaboration (multiple users editing the same diagram simultaneously)
- Version history or undo beyond the current browser session
- Generating static image exports (PNG, PDF) from the web viewer — the CLI's static diagram commands cover this
- Embedding the viewer as a component in third-party applications
- Offline PWA support (service workers, local caching)
- Persisting diagram layout positions to disk or across sessions

## Open Questions

- Should the standalone web viewer include a WASM-compiled Go parser, or should a lightweight TypeScript parser be written for client-side use?
- When exporting edited models back to `.emod`, should comments from the original file be preserved, or is it acceptable to lose them?
- Should the `--serve` command support a `--watch` flag that auto-reloads the diagram when the source file changes on disk, or is that a separate story?
- What is the right behavior when the model is too large for the viewport at default zoom — should it start zoomed-to-fit or at a fixed scale?
- Should rearranged node positions be exportable as a separate layout file so they can be shared or version-controlled independently of the `.emod` source?

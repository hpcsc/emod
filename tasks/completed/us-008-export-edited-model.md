# US-008: Export edited model back to .emod format

## Progress
- [x] Task 1: Create `emod-export.js` module with full .emod serialization and tests
- [x] Task 2: Add "Export .emod" button to toolbar with file download wiring

---

## Story Reference

**File:** n/a (inline user story US-008)

**Description:** As a model author, I want to export the diagram back to `.emod` syntax so that changes I make visually are persisted in the source file.

**Depends on:** US-007 (already completed)

---

## Codebase Context

### Project structure
- `internal/viewer/viewer.html` — Main HTML with all CSS and toolbar buttons. Existing toolbar buttons: Reset layout (`#reset-layout`), Fit view (`#fit-view`), Minimap (`#minimap-toggle`), Contexts (`#context-toggle`). Each follows a consistent styling pattern: `background: rgba(255,255,255,0.15)`, `border: 1px solid rgba(255,255,255,0.25)`, `border-radius: 4px`, `padding: 5px 12px`.
- `internal/viewer/viewer.js` — Orchestrator that initializes all modules and wires DOM event listeners. Creates `store` via `createStore()`, binds DOM refs, subscribes to bus events. Button click handlers follow a consistent pattern: `store.dom.[ref].addEventListener("click", function() { ... })`.
- `internal/viewer/model.js` — Data operations on `store.nodes[]` and `store.edges[]`. `nodes` are objects with `id`, `type`, `label`, `parentId`, and type-specific metadata (`fields`, `kind`, `actor`, `reads`, `subscribes`, `trigger_event`, `command`, `target_context`, `external_system`, `event`).
- `internal/viewer/store.js` — Central store with `nodes[]`, `edges[]`, `modelName`, `nodeById` map.

### Node structure
Nodes in `store.nodes[]` have these relevant fields:
- **context:** `id`, `type: "context"`, `label`, `parentId: null`
- **aggregate:** `id`, `type: "aggregate"`, `label`, `parentId: <contextId>`
- **slice:** `id`, `type: "slice"`, `label`, `parentId: <aggregateId>`
- **trigger:** `id`, `type: "trigger"`, `label`, `parentId: <sliceId>`, `kind`, `actor`, `reads`
- **command:** `id`, `type: "command"`, `label`, `parentId: <sliceId>`, `fields[]`
- **event:** `id`, `type: "event"`, `label`, `parentId: <sliceId>`, `fields[]`, `source`, `external_name`
- **view:** `id`, `type: "view"`, `label`, `parentId: <sliceId>`, `fields[]`, `subscribes[]`
- **automation:** `id`, `type: "automation"`, `label`, `parentId: <sliceId>`, `trigger_event`, `command`, `target_context`
- **translation:** `id`, `type: "translation"`, `label`, `parentId: <sliceId>`, `external_system`, `reads`, `command`, `event`

Edges in `store.edges[]`:
- `{ source, target, type }` where type is one of: `"flow"`, `"subscription"`, `"automation_trigger"`, `"automation_command"`, `"trigger_command"`, `"reads"`, `"translation_command"`

### Canonical formatting reference
- `internal/formatter/formatter.go` — Go formatter producing the canonical `.emod` output. Shows 2-space indent (`strings.Repeat("  ", level)`), blank lines between slices, `flow { command -> event: ... }` syntax, column-aligned fields, `trigger <Kind> "..." { ... }`, `event <Name> { ... source external "..." }`, `automation <Name> { ... }`, `translation <Name> { ... }}`.

### Traversal order
The `layout.js:computeLayout()` iterates: contexts → aggregates (left-to-right) → slices (left-to-right) → triggers → commands → events → views → automations → translations. This is the same order used by the Go formatter and the required output order.

### File download pattern
Currently no file download exists in the viewer. The download should use a Blob + `URL.createObjectURL` + temporary `<a>` element click, a standard browser pattern.

---

## Tasks

### Task 1: Create `emod-export.js` module with full .emod serialization and tests

**Behavior:** A new `exportToEmodString(store)` function serializes the current store state (nodes + edges) into a `.emod` string that matches the canonical formatting conventions.

**Acceptance Criteria:**
- [ ] An "Export .emod" action generates valid `.emod` syntax reflecting the current state of the diagram including any edits
- [ ] The exported file uses consistent formatting (2-space indent, one blank line between slices)
- [ ] Elements are output in the order they appear in the diagram (left-to-right slices, top-to-bottom within slices)

**Affected Files/Modules:**
- `internal/viewer/emod-export.js` — **New file.** Exports `exportToEmodString(store)` function. Serializes all node types, handles field column alignment, flow edges, and type-specific metadata.
- `internal/viewer/emod-export.test.js` — **New file.** Tests for the serialization function.
- `internal/viewer/embed.go` — Add `emod-export.js` to the `//go:embed` directive.

**Patterns to Follow:**
- Follow the Go formatting conventions in `internal/formatter/formatter.go:10-262` for indent, field alignment, blank lines, flow syntax, trigger syntax, event external source, automation body, translation body.
- Follow the traversal order in `internal/viewer/layout.js:39-181` (`computeLayout`) for node iteration order: contexts → aggregates → slices → trigger/commands/events/views/automations/translations → flows.
- Follow the existing module export pattern in `internal/viewer/model.js:105-112` — export as a named object with functions.

**Testable:** Yes

**Verification:** `npm test` passes; serialized output matches expected `.emod` format strings.

**Depends on:** None

---

### Task 2: Add "Export .emod" button to toolbar with file download wiring

**Behavior:** A new "Export .emod" button appears in the viewer toolbar. Clicking it generates the `.emod` content from the current store state and triggers a browser file download named `<model-name>.emod` (or `diagram.emod` as fallback).

**Acceptance Criteria:**
- [ ] The export is offered as a file download in the browser
- [ ] The exported `.emod` file can be parsed by `emod validate` without errors (manual verification)

**Affected Files/Modules:**
- `internal/viewer/viewer.html` — Add "Export .emod" button to the `<header>` toolbar div, following the pattern of existing buttons.
- `internal/viewer/viewer.js` — Import the new export module, add click handler for the export button that calls `exportToEmodString(store)`, creates a Blob, and triggers a file download via temporary anchor element.

**Patterns to Follow:**
- Follow existing toolbar button styling in `internal/viewer/viewer.html:205-241` (`#reset-layout`, `#fit-view`) for consistent appearance.
- Follow existing button click handler wiring in `internal/viewer/viewer.js:176-185` (`resetLayoutBtn`, `fitViewBtn` event listeners) for the click handler pattern.

**Testable:** No (browser-only download UI; tests for serialization correctness are in Task 1)

**Verification:** Manual — click "Export .emod" and verify a `.emod` file is downloaded, then run `emod validate <file>` on it.

**Depends on:** Task 1

---

## Summary

- **Total tasks:** 2
- **Ordering rationale:** Dependency-first. Task 1 delivers the core serialization logic (testable in isolation), Task 2 adds the UI button and download wiring on top.
- **Language:** Both tasks are JavaScript/TypeScript (browser-side only, no Go changes beyond `embed.go`).
- **Acceptance criteria coverage:** All 5 criteria are covered. Criteria 1 (valid syntax), 2 (formatting), and 3 (element order) are covered by Task 1's tests. Criteria 4 (parseable by `emod validate`) and 5 (file download) are covered by Task 2's manual verification.

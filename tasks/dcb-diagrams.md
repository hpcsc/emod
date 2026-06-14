# US-003: Visualize DCB models in diagrams — Implementation Tasks

## Progress
- [x] Task 1: Extend collectSlices for DCB and add --style CLI plumbing
- [x] Task 2: Implement projected swim-lane layout for draw.io
- [x] Task 3: Implement projected swim-lane layout for Mermaid
- [ ] Task 4: Implement query-lens view for draw.io and Mermaid

## Story Reference

**US-003** from `user-stories/dynamic-consistency-boundary.md`:
As a model author, I want to generate diagrams from a DCB model that show tags and decision queries, so that I can communicate the cross-cutting consistency boundaries to my team.

**Depends on:** US-001 (authoring DCB models — parser, validator, linter)

## Codebase Context

**Affected packages:**
- `internal/diagram/` — all four export functions (`ExportDrawio`, `ExportMermaid`, `ExportSVG`, `ExportASCII`) currently share `collectSlices()` as their entry point. That function only walks `ctx.Aggregates[].Slices`, ignoring `ctx.Slices` (direct context slices — the DCB pattern). Each function also has its own layout logic based on aggregate-grouped swim lanes.
- `internal/cli/diagram.go` — `RunDiagram()` dispatches to export functions by format. No `--style` flag exists yet. The `app.go` wiring defines diagram command flags.
- `internal/ast/ast.go` — `Context` already has `Mode`, `Slices []*Slice`, `Aggregates []*Aggregate`. Events have `Tags []TagEntry`. Commands have `DecidesOn *DecidesOnClause`.

**Existing patterns:**
- `collectSlices()` in `drawio.go:420-430` — flattens all slices from aggregates into `[]sliceEntry`. All four exporters call it. This is the single point where DCB slice collection needs extending.
- `itemLayout()` in `drawio.go:433-443` — shared utility for computing element position within a slice usable by all formats.
- Swim lane rendering in `drawio.go:111-123` — fixed lanes based on element type (trigger, command/view, event, external). Tag-projected lanes replace this for DCB contexts.
- Test helpers: `singleSliceModel()`, `minimalModel()`, `fullModel()` in `drawio_test.go:384-518` — all construct aggregate-based models. New DCB model helpers will be needed.

**Key insight:** DCB contexts have no aggregates, so `collectSlices()` returns nothing for them today → empty diagrams. The fix must be upstream of all four exporters.

---

## Tasks

### Task 1: Extend collectSlices for DCB contexts and add --style CLI plumbing

**Behavior:** Slices belonging to a DCB context (stored directly on `Context.Slices`) are collected and rendered by all four diagram formats. The CLI accepts a `--style` flag that controls layout selection, and each export function selects the appropriate layout strategy based on context mode and the requested style. Aggregate-mode contexts remain fully backward-compatible.

**Acceptance Criteria:**
- [ ] `collectSlices()` collects `ctx.Slices` in addition to `ctx.Aggregates[].Slices`, deduplicating or ordering predictably
- [ ] All four export functions (`ExportDrawio`, `ExportMermaid`, `ExportSVG`, `ExportASCII`) produce output for a DCB-only model (context with direct slices, no aggregates) instead of returning empty
- [ ] A `Style` type is defined in the diagram package with values for default/auto, projected, and query-lens
- [ ] The `diagram` CLI command accepts a new `--style` flag with values `projected` and `dcb`; the default is auto-detection based on context mode
- [ ] `RunDiagram()` threads the style value through to export functions
- [ ] Export functions select layout strategy: aggregate mode → current behavior; DCB mode with no flag → projected; DCB mode with `--style=dcb` → query-lens
- [ ] Aggregate-based models produce identical output before and after this change (backward compatibility)
- [ ] Tests cover DCB context rendering in all four formats, style flag parsing, and style auto-detection

**Affected Files/Modules:**
- `internal/diagram/drawio.go` — `collectSlices()` at line 420, `ExportDrawio()` at line 46
- `internal/diagram/mermaid.go` — `ExportMermaid()` at line 15
- `internal/diagram/svg.go` — `ExportSVG()` at line 12
- `internal/diagram/ascii.go` — `ExportASCII()` at line 20
- `internal/cli/diagram.go` — `RunDiagram()` at line 30
- `internal/cli/app.go` — diagram command flag definition at line 143

**Patterns to Follow:**
- Style detection/dispatch: model the style enum and its selection logic following the style of `RunDiagram()` format dispatch in `internal/cli/diagram.go:88-97`
- CLI flag definition: follow the existing `--format`, `-o`, and `--serve` flag definitions in `internal/cli/app.go:143-157`
- Slice collection: extend `collectSlices()` in `internal/diagram/drawio.go:420-430` — the existing function already produces a flat list of `sliceEntry` structs with a `ctxName` field
- Test patterns: follow `internal/diagram/drawio_test.go:35-43` and `internal/diagram/mermaid_test.go:20-28` for empty/nil model tests

**Language:** Go

**Testable:** Yes — all four export functions are tested with DCB models through their exported API. CLI tests verify `--style` flag parsing. Backward compatibility verified by running existing aggregate-based tests unchanged.

**Verification:** `go test -tags unit ./internal/diagram/... ./internal/cli/...` passes

**Depends on:** None (US-001 provides the parsed AST types, which are already in the codebase)

---

### Task 2: Implement projected swim-lane (tag-based) layout for draw.io

**Behavior:** When rendering a DCB (or mixed-mode) context in projected style, draw.io output shows one swim lane per unique tag key found across events in that context. Each event appears in the lane(s) corresponding to its declared tag keys. An event with multiple tags appears in multiple lanes with a visible connector linking its representations. Commands and triggers render in a neutral lane or are distributed to all relevant tag lanes. Aggregate-mode contexts continue to use the existing aggregate-grouped swim lane layout without change.

**Acceptance Criteria:**
- [ ] A DCB context renders one swim lane per unique tag key (ordered alphabetically or by first appearance)
- [ ] An event with a single tag appears only in that tag's lane
- [ ] An event with multiple tags appears in each tag's lane, with a visible connector (e.g., a dashed line or shared border indicator) linking the representations
- [ ] Commands and triggers are rendered once and visible across all lanes (not duplicated per lane)
- [ ] The `projects` header labels each lane by tag key
- [ ] Mixed-mode contexts show both aggregate-grouped and tag-grouped slices in the same diagram
- [ ] Aggregate-mode contexts produce identical output to the pre-change baseline
- [ ] Tag-projected output remains valid draw.io XML

**Affected Files/Modules:**
- `internal/diagram/drawio.go` — new rendering path for tag-projected swim lanes; existing aggregate path unchanged
- `internal/diagram/drawio_test.go` — new test cases for projected swim-lane output

**Patterns to Follow:**
- Swim lane cell creation: follow `swimlaneCell()` in `internal/diagram/drawio.go:480-486` for creating per-tag lanes
- Element placement within lanes: follow the trigger/command/event placement pattern in `drawio.go:165-287` — each lane needs its own y-center calculation
- Multi-tag connectors: follow the edge/waypoint pattern in `edgeCell()` and `edgeCellWaypoints()` at `drawio.go:494-512`
- XML cell builders: reuse `vertexCell()`, `swimlaneCell()`, and related helpers in `drawio.go:456-521`

**Language:** Go

**Testable:** Yes — `ExportDrawio` is an exported function tested through `drawio_test.go`. Tests construct DCB models with tagged events and verify lane labels, event placement, and connector presence in the output XML.

**Verification:** `go test -tags unit ./internal/diagram/...` passes; visual inspection of draw.io XML output confirms correct lane structure

**Depends on:** Task 1 (collectSlices fix and style plumbing)

---

### Task 3: Implement projected swim-lane (tag-based) layout for Mermaid

**Behavior:** When rendering a DCB context in projected style, Mermaid output uses the `eventmodeling` diagram type with tag-key-based grouping instead of aggregate-based grouping. One logical swim lane per tag key, events tagged with multiple keys appear in multiple lanes with a visible connector. Aggregate-mode Mermaid output is unchanged.

**Acceptance Criteria:**
- [ ] A DCB context renders with tag-based grouping — one group per tag key
- [ ] An event with a single tag appears in that tag's group
- [ ] An event with multiple tags appears in multiple groups with a connector reference
- [ ] Commands and triggers render in an ungrouped or cross-cutting section
- [ ] Aggregate-mode contexts produce identical Mermaid output to the pre-change baseline
- [ ] Mermaid output for projected style is syntactically valid eventmodeling markup

**Affected Files/Modules:**
- `internal/diagram/mermaid.go` — new rendering path for tag-projected swim lanes; existing aggregate path unchanged
- `internal/diagram/mermaid_test.go` — new test cases for projected swim-lane Mermaid output

**Patterns to Follow:**
- Timeframe rendering: follow the `tf NN type Name` pattern in `internal/diagram/mermaid.go:42-121` for all element types
- Namespace grouping: follow the dot-notation pattern at `mermaid.go:48-49` for context-qualified names
- Comment annotations: follow the `% Slice:` comment pattern at `mermaid.go:39` for tagging metadata

**Language:** Go

**Testable:** Yes — `ExportMermaid` is an exported function tested through `mermaid_test.go`. Tests construct DCB models and verify tag-based grouping in the output text.

**Verification:** `go test -tags unit ./internal/diagram/...` passes

**Depends on:** Task 1 (collectSlices fix and style plumbing); shares the tag-key extraction logic with Task 2

---

### Task 4: Implement query-lens view for draw.io and Mermaid

**Behavior:** When `--style=dcb` is specified, both draw.io and Mermaid output show a query-lens view: a single flat timeline with events in chronological order, tags rendered as colored badges on each event, and commands rendered as labelled brackets that list the event types and predicate from their `decides_on` clause. Aggregate-mode contexts with `--style=dcb` behave reasonably (or fall through to default layout).

**Acceptance Criteria:**
- [ ] A single flat swim lane displays all events in order (no tag-based grouping)
- [ ] Each event shows its tags as colored badge indicators (short label showing tag key:value or tag key)
- [ ] Each command is rendered as a labelled bracket: `[CommandName]` with an annotation showing `decides_on: EventType1, EventType2 where tag(key = fieldRef, ...)`
- [ ] Command-to-event flow arrows remain visible
- [ ] The output is valid for the target format (valid XML for draw.io, valid eventmodeling markup for Mermaid)
- [ ] Non-DCB contexts with `--style=dcb` render without error

**Affected Files/Modules:**
- `internal/diagram/drawio.go` — new query-lens rendering path
- `internal/diagram/mermaid.go` — new query-lens rendering path
- `internal/diagram/drawio_test.go` — new test cases for query-lens output
- `internal/diagram/mermaid_test.go` — new test cases for query-lens output

**Patterns to Follow:**
- Command rendering with annotation: follow the escape-label pattern in `drawio.go:188-192` for commands, extending to include `decides_on` information
- Tag badge rendering: follow the color-fill pattern in `drawio.go:223-224` for event cells, adapting to show tag data as inline labels
- Flat timeline layout: follow the existing slice-based horizontal layout in `drawio.go:147-162` but without tag-key grouping lanes

**Language:** Go

**Testable:** Yes — both `ExportDrawio` and `ExportMermaid` are exported functions. Tests construct DCB models with tags and `decides_on` clauses and verify the query-lens output structure, tag badges, and command annotations.

**Verification:** `go test -tags unit ./internal/diagram/...` passes

**Depends on:** Task 1 (collectSlices fix and style plumbing)

---

## Summary

- **Total tasks:** 4
- **Language:** All tasks are Go
- **Task ordering rationale:** Task 1 is the prerequisite (foundation: slice collection + style plumbing). Tasks 2 and 3 are independent and can be parallelized (projected swim-lane for draw.io and Mermaid respectively). Task 4 (query-lens) depends only on Task 1 and can be worked on in parallel with Tasks 2–3.
- **Acceptance criteria coverage:**
  - AC1 (tag-projected swim lanes by default): Tasks 1 (auto-detect), 2 (draw.io), 3 (Mermaid)
  - AC2 (multi-tag events in multiple lanes with connector): Tasks 2, 3
  - AC3 (query-lens with `--style=dcb`): Task 4
  - AC4 (Mermaid and draw.io support projected swim-lane): Tasks 2, 3
  - AC5 (aggregate models unchanged): Task 1 (backward compat verified), reaffirmed in Tasks 2, 3, 4
- **Formats not explicitly required for projected/query-lens:** SVG and ASCII benefit from the `collectSlices` fix in Task 1 (they now render DCB content) but are not required to implement the new layout styles per the acceptance criteria. They continue to use the default aggregate-grouped layout, which works correctly for aggregate-mode contexts.

# Task Breakdown: Keep each trigger in its own slice when two triggers share a name

## Contents

1. Task 1: Join each trigger's arrows to the box of its own slice in the SVG
2. Task 2: Key each trigger node and its edges by its slice in the diagram document

## Story Reference

`user-stories/screens-as-first-class.md`, US-002 "Keep each trigger in its own slice when two triggers share a name"
(story ID `screens-as-first-class/US-002`). Goal G1 of `docs/proposals/screens-as-first-class-proposal.md`.

## Boundaries

**Out of scope:**

- The Mermaid and ASCII diagrams. The story names the SVG, the viewer and the save. US-004 names the other
  formats for the screen appearances.
- Events, views and commands. They have one box and one node for each slice that declares them already
  (`6c00a67`, `diagram.DeclaredElement`).

## Codebase Context

- `internal/diagram/svg.go`: the trigger box goes into `nameToBox` under the trigger name, one map for all
  slices, so the last slice drawn wins. `EdgeTriggerReads` reads `nameToBox[edge.To]`, and
  `EdgeTriggerCommand` goes through the `default` case, which reads `nameToBox` for both ends.
- `internal/export/diagram.go`: `triggerIDs` is keyed by the trigger name, and `appendEdges` links
  `EdgeTriggerReads` and `EdgeTriggerCommand` through it.
- The viewer draws each edge of the diagram document by node id (`internal/frontend/static`), and no viewer
  code looks a trigger up by its label. So the arrows of the viewer follow the document.
- Both save paths read node ids: `importer.ImportDiagram` and `importer.ApplyDiagram`
  (`diagramEdit.drawnFrom` maps a node id to the construct). With one id for each trigger, an edge lands on the
  trigger of its own slice.

## Tasks

### Task 1: Join each trigger's arrows to the box of its own slice in the SVG

**Behavior:** In a model where the slices "Apply Discount" and "Place Order" each have `trigger "Checkout Page"`,
the SVG draws two trigger boxes, and each box gets the `reads` arrows and the command arrows of its own slice
only.

**Acceptance Criteria:**

- [x] With two slices that each have `trigger "Checkout Page"`, each reading its own view and each with its own
      command, the SVG draws two trigger boxes, and each box gets exactly the arrows of its own slice: one from
      its view and one to its command.
- [x] No arrow joins a trigger box to the command or the view of the other slice.
- [x] A model where each trigger has a unique name renders the same as before: the existing SVG tests pass with
      no change to their expected values.
- [x] `task test:unit` passes.

**Affected Files/Modules:**

- `internal/diagram/svg.go`
- `internal/diagram/contract_test.go` or `internal/diagram/svg_test.go`

**Patterns to Follow:**

- `internal/diagram/svg.go:128-133` (badges filed per slice, for the same reason)
- `internal/diagram/svg.go:142-160` (`declaredBoxes` and `joinedBox`)
- `internal/diagram/contract_test.go:2246-2271` (arrows read back by box)
- `tasks/learnings.md` "A lane-diagram name is not an identity: key per-element lookups by the declaring slice"

**Testable:** Yes, through the SVG export that the contract harness runs.

**Certainty:** high: the badges and the copies already file a box per slice in the same function
(`svg.go:128-160`).

**Blast radius:** low: a picture; no tool parses it.

**Verification:** `task test:unit`.

**Depends on:** None

---

### Task 2: Key each trigger node and its edges by its slice in the diagram document

**Behavior:** The diagram document gives each trigger node its own id and joins the `reads` and
`trigger_command` edges of a slice to the trigger node of that slice. So the viewer draws each trigger's arrows in its own
slice, and a save from the viewer keeps both triggers, each in its own slice, with what each reads.

**Acceptance Criteria:**

- [ ] With two slices that each have `trigger "Checkout Page"`, the diagram document has two trigger nodes, each
      a child of its own slice, and each node is the target of the `reads` edge from its own slice's view and
      the source of the `trigger_command` edge to its own slice's command. No edge joins a trigger node to a node
      of the other slice.
- [ ] `ImportDiagram` on that document, then `formatter.Format`, gives back the source, with both triggers, each
      in its own slice and each reading its own view.
- [ ] `ApplyDiagram` with no edit leaves that source as it was, and a new `reads` arrow onto the trigger of one
      slice changes the reads of that trigger only.
- [ ] A model where each trigger has a unique name exports the same document as before: the existing diagram
      document tests pass with no change to their expected values.
- [ ] `task test:unit` and `task test:integration` pass.

**Affected Files/Modules:**

- `internal/export/diagram.go`: `diagramBuilder.triggerIDs`, `appendTrigger`, `appendEdges`
- `internal/export/export_test.go`, group "diagram json"
- `internal/importer/importer_test.go`, `internal/importer/diagram_edit_test.go`

**Patterns to Follow:**

- `internal/export/diagram.go:336-345` (`declaredIDs`, keyed by `DeclaredElement`)
- `internal/export/diagram.go:552-562` (`copiedEndIDs` hands `link` a one-entry map)
- `internal/importer/diagram_edit_test.go` group "reads of a trigger"
- `tasks/learnings.md` "Two triggers with one name share one diagram node, so an ApplyDiagram edit can name a
  view the drawn trigger does not read"

**Testable:** Yes, through `export.ExportDiagramJSON`, `importer.ImportDiagram` and `importer.ApplyDiagram`.

**Certainty:** high: `declaredIDs` and `copiedEndIDs` key the other elements by slice in the same builder
(`diagram.go:336-345`, `:552-562`).

**Blast radius:** medium: the diagram document is an output of `emod export --format diagram-json` and of the
MCP `emod_export` tool, and both viewer save paths read it. Node ids are not stable across exports, so only the
edges of models with repeated trigger names change.

**Verification:** `task test:unit && task test:integration`.

**Depends on:** None

## Summary

- **Total:** 2 tasks, independent of each other.
- **The command end:** a trigger's arrow ends at the command of the trigger's own slice in both writers. The
  language accepts one command name in two slices, and the story says that no arrow joins a trigger to a
  command of another slice.
- **Story criteria:** the SVG boxes and arrows: Task 1. The viewer arrows and the save: Task 2. Unique names render
  the same: both tasks.

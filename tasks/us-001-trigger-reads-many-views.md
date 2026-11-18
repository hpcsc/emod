# Task Breakdown: Let a trigger read more than one view

## Contents

1. Task 1: Accept a list of views in the reads of a trigger, and resolve each entry
2. Task 2: Carry each entry through the diagram document, both viewer saves and the detail panel
3. Task 3: Give the reads of a trigger as a list in the JSON and CUE exports and the schema
4. Task 4: Print each entry in the ASCII diagram, and pin the SVG arrows and copies
5. Task 5: Document the list form in the DSL reference

## Story Reference

`user-stories/screens-as-first-class.md`, US-001 "Let a trigger read more than one view" (story ID
`screens-as-first-class/US-001`). Goal G3 of `docs/proposals/screens-as-first-class-proposal.md`, phase 1
"`reads` takes a list".

## Boundaries

**Out of scope:**

- From the story: "An automation and a translation keep one `reads` name: a processor reads one view as its
  todo list."
- From the proposal's non-goals: "Changes to how `view`, `automation` or `translation` connect. Only the reader
  at the human end of `reads` changes."
- The Mermaid diagram and the event flow diagram. They draw no arrow for the `reads` of a trigger today, and
  the story names only the SVG, the viewer and the ASCII diagram.
- A check that reports one view named two times in one list.
- Completion of view names inside a `reads` list in the editor. The completer offers views after `reads =`
  only.
- A diagram document that gives an automation node or a translation node more than one `reads` entry. The
  exporter never writes one.

**Deferred:**

- The hover text of `reads` and `trigger` in the editor (`internal/lsp/hover.go:21`, `:31`) and the message
  and description of `view/never-read` (`internal/linter/linter.go:666`, `descriptions.go:53`), which say "the
  view" and "a trigger". US-004 rewrites them when the screen replaces the trigger (proposal, sections "LSP"
  and "Linter").
- One box for each slice when two triggers share a name. That is US-002. The diagram document still keys a
  trigger node by its name.
- The list on the `shows` block. That is US-004. The fields `Reads` and `ReadsPos` carry over to `Appearance`
  with the same names.

## Codebase Context

**Every reader of a trigger's `reads`.** gopls finds these readers of `ast.Trigger.Reads` and `ReadsPos`
(`internal/ast/ast.go:220-232`):

- Source: `internal/parser/parser.go:408-435` (`r.trigger` reads the value with `r.name`, `:114-123`; `r.names`,
  `:125-140`, reads a list only). `internal/formatter/formatter.go:87-94` writes it with `writeRef` (`:270`).
  `subscribes` uses `refListTokens` (`:164`, `:310`).
- Name lookups: `internal/validator/validator.go:399-401` (`appendUndeclaredRef`, `:457-463`),
  `internal/linter/linter.go:65-84`, `internal/query/names.go:134-136` (`NameSites`, which feeds
  go-to-definition and find-references through `internal/lsp/definition.go:22` and
  `internal/lsp/references.go:31`), `internal/query/references.go:309-311`, and
  `internal/arrange/arrange.go:132-134`. The prompt did not list `internal/arrange`; the proposal's phase 1
  does.
- Arrows: `internal/diagram/graph.go:74-84` (`SliceEdges`). SliceEdges is the one source of arrows for the
  SVG (`svg.go:288-295`), the copies of far elements (`element_copies.go:84-102`, `:138-151`), the edges of the
  diagram document (`internal/export/diagram.go:473-490`) and `emod trace` (`internal/query/trace.go:203`,
  `:250-253`). One edge for each entry gives all four an arrow for each entry. This replaces part of the
  learning "An arrow between two constructs is drawn by five surfaces": the SVG and the diagram document now
  read the same edges. The ASCII diagram (`ascii.go:52-55`) and Mermaid still walk the AST themselves, and
  neither draws the `reads` of a trigger today.
- Serialized forms: `internal/export/json.go:145-156`, `:652-667`; `internal/export/cue.go:184-190`;
  `internal/cue/schema.cue:15-21`; the diagram node `internal/export/diagram.go:21-44`, `:373-390`; the importer
  `internal/importer/importer.go:22-44`, `:89-97`, `:426-449`; the edit of a parsed source
  `internal/importer/diagram_edit.go:550-576`, `:602-664`.
- Viewer: only the detail panel reads the `reads` of a trigger node (`internal/frontend/static/ui.js:535-542`).
  The viewer already types an arrow from a view to a trigger as `reads` (`model.js:271`), and the canvas draws
  each edge of the document.
- Test support: `internal/test/fixtures.go:3600-3613` (`WithoutTriggerReads`) and `:3833-3846`
  (`DeclaredTriggerReads`), plus about fifty AST literals in tests that set `Reads:` to a string.

**The type change reaches every reader in one commit.** When `Reads string` becomes `Reads []string`, each site
above stops compiling. So the task that changes the type must also rewrite each reader. For a name lookup and
for SliceEdges, the rewrite is a loop over the entries, and it is the behaviour of the story, so Task 1 tests
each of them. The serialized forms need a design of their own (Tasks 2 and 3). Until those tasks land, Task 1
keeps each serialized form on the one name that it writes today.

**Two save paths.** The viewer saves an edit through `importer.ApplyDiagram` (`internal/pipeline/pipeline.go:118-160`,
called from `viewer.js:512`). It edits the parsed source arrow by arrow. The export button and a viewer opened
on a `.json` document use `importer.ImportDiagram` (`pipeline.go:80-93`), which builds the model from the
document. `ApplyDiagram` handles a `subscription` entry by entry (`diagram_edit.go:121-165`, `:558-560`,
`:621-636`), and that is the precedent for a list of `reads`. `ImportDiagram` takes the `reads` that a node
records, and takes an edge only when the node records none (`importer.go:426-449`).

**One `reads` field serves three node types.** `jsonDiagramNode.Reads` and `diagramNode.Reads` serve the
trigger, the automation and the translation nodes. The automation and the translation keep one name. A second
Go field with the same `reads` key is not a way out: encoding/json drops both fields of a struct when two of
them carry the same key, and gives no error. The diagram document is an output of
`emod export --format diagram-json` (`internal/cli/export.go:20`) and of the MCP `emod_export` tool, and the
viewer opens a saved `.json` document, so documents that give a trigger's `reads` as one name exist.

**Exports.** `subscribes` is the precedent list on each export surface (`json.go:166`, `cue.go:232`,
`schema.cue:57`). No position key of the JSON export holds a list today. `ReadsPos []Position` follows
`View.SubscribesPos` (`ast.go:241-242`), so the JSON key `reads_position` keeps its name and holds one position
for each entry. `internal/cue/embed_test.go:181-203` rejects the retired bare-name shape of a spec's `given`
and `when`, which is the precedent for the retired one-name `reads`. `internal/cli/export_test.go:543` asserts
the one-name shape.

**The text-format contract.** `TestExporterReadsEdges` (`internal/diagram/contract_test.go:827-850`) requires
that Mermaid and ASCII draw a model with trigger `reads` exactly as the same model without them. Task 4 makes
ASCII print the arrow of a trigger, so that leaf keeps the rule for Mermaid, and for the `reads` of an
automation in ASCII.

**The parser is HCL.** The learning "A bracketed list's terminator set decides whether one typo cascades" is
about the hand-written parser that HCL replaced. `parseIdentifiersUntil` does not exist now. `hcl.ExprList`
reads a list, and `r.name` reports an entry that is not a name at that entry (`parser.go:114-140`). `r.name`
does not call `Value(nil)`, so the IsKnown learning does not apply to it.

**Docs.** `docs/dsl-reference.md` documents the `reads` of a trigger at `:318-341` (the skeleton and the
far-copy bullet), `:811` (the cross-reference table) and `:815-819` (each `reads` entry is reported where it
is written). The MCP server serves the file (`internal/mcp/server.go:134`). `site/content/docs/dsl-reference.md`
is a separate, shorter file; `:106` says a trigger names "the view they read". `web/` is ignored build output.

**Words.** The story and the code say "entry" for one name in a `reads`, and "bare name" for
`reads = CartView`. `element` belongs to `diagram.DeclaredElement` and `query.element`, so it is not a word for
an entry.

## Tasks

### Task 1: Accept a list of views in the reads of a trigger, and resolve each entry

**Behavior:** A trigger accepts `reads = [CartView, DiscountView]` and still accepts `reads = CartView`. The
model holds the views in the order that the author wrote them, each with its own position. `emod fmt` writes
one entry as a bare name and two or more entries as a list. Each name lookup reads each entry: `emod validate`
reports each undeclared entry at that entry, `view/never-read` counts each entry as a reader, `emod references`
and the editor list the trigger under each view, `emod arrange` treats each entry as a reference, and
SliceEdges gives one arrow from each view, so `emod trace` follows each entry. The JSON and CUE exports, the
trigger node of the diagram document and the importer go on with the one name that they give a trigger today,
which is the first entry. Tasks 2 and 3 give them the list.

**Acceptance Criteria:**

- [x] A slice whose trigger states `reads = [CartView, DiscountView]` parses with no diagnostics. The trigger
      reads `CartView` and then `DiscountView`, each with the line and column of its own entry.
- [x] A trigger with `reads = CartView` parses to a list of one entry, at the position of the name. Each `.emod`
      file under `examples/` and `internal/parser/testdata/` parses and validates with the diagnostics that it
      gives today, and no file changes.
- [x] An entry of the list that is not a name, for example a quoted string, gives one diagnostic, at that
      entry. The other entries of the list are still read.
- [x] `reads = []` reads no view, as a trigger with no `reads` does.
- [x] `emod fmt` writes two or more entries as `reads = [CartView, DiscountView]`, in the order written. It
      writes one entry as `reads = CartView`, also when the source states `reads = [CartView]`. A second run of
      `emod fmt` on its output changes nothing.
- [x] `emod validate` reports each entry that names no declared view, at the line and column of that entry,
      with the message `view "<Name>" does not exist` that it gives for the `reads` of an automation. It
      reports no entry that names a declared view.
- [x] `view/never-read` does not report a view that only the second entry of a trigger's list names.
- [x] `emod references` on each view of a trigger's list lists the trigger in the group of readers, with the
      keyword `reads` and the line of that entry.
- [x] In the editor, go-to-definition on the second entry of a trigger's list goes to the declaration of that
      view, and find-references on that view includes the entry.
- [x] In a model where only the second entry of a trigger's list reads a view, and a slice after the trigger's
      slice declares that view, `emod arrange` moves the view's slice ahead of the trigger's slice, as it does
      for a view that a single `reads` names.
- [x] `diagram.SliceEdges` gives one `reads` edge from a view to the trigger for each entry, in list order.
- [x] `emod trace` follows the `reads` edge of each entry: a trace from an event that only the view of the
      second entry subscribes to reaches the trigger.
- [x] For a model whose triggers each read one view or none, the output of `emod fmt`, of `emod export` in
      each format, of `emod diagram` in each format, of `emod references` and of `emod trace` does not change:
      the expected outputs of the existing tests stay as they are. Only the AST literals that build a trigger,
      and the assertions that read `Trigger.Reads` or `ReadsPos`, change to the list type, with the same
      values.
- [x] `task test:unit` and `task test:integration` pass.

**Affected Files/Modules:**

- `internal/ast/ast.go`: `Trigger.Reads` becomes `[]string` and `ReadsPos` becomes `[]Position`.
- `internal/parser/parser.go`: `r.trigger` reads a bare name or a list; `parser_test.go`,
  `integration_test.go`.
- `internal/formatter/formatter.go`: the trigger in `writeSlice`; `formatter_test.go`, `internal/cli/fmt_test.go`.
- `internal/validator/validator.go`: `referenceDiagnostics`; `validator_test.go`.
- `internal/linter/linter.go`: the set of read views in `Lint`; `linter_test.go`.
- `internal/query/names.go`, `internal/query/references.go`: `references_test.go`, `trace_test.go`.
- `internal/lsp/definition_test.go`, `internal/lsp/references_test.go`: tests only. The LSP reads
  `query.NameSites`.
- `internal/arrange/arrange.go`: `arrange_test.go`.
- `internal/diagram/graph.go`: `SliceEdges`; `graph_test.go`.
- `internal/export/json.go`, `internal/export/cue.go`, `internal/export/diagram.go`,
  `internal/importer/importer.go`, `internal/importer/diagram_edit.go`: compile against the list and keep
  the one-name output.
- `internal/test/fixtures.go`: `WithoutTriggerReads`, `DeclaredTriggerReads`.
- Each test that builds an `ast.Trigger` with `Reads:`, in `internal/diagram`, `internal/export`,
  `internal/glossary`, `internal/importer`, `internal/linter`, `internal/parser` and `internal/validator`.

**Patterns to Follow:**

- `internal/parser/parser.go:114-140`
- `internal/formatter/formatter.go:161-167`, `:310-325`
- `internal/validator/validator.go:413-417`; `internal/validator/validator_test.go:1088-1122`, `:1162-1200`
- `internal/linter/linter.go:65-84`; `internal/linter/linter_test.go:3792-3842`
- `internal/query/names.go:117-122`; `internal/query/references.go:299-308`;
  `internal/query/references_test.go:170-191`; `internal/query/trace_test.go:223-235`
- `internal/lsp/definition_test.go:122-126`; `internal/lsp/references_test.go:255-278`
- `internal/arrange/arrange.go:127-131`; `internal/arrange/arrange_test.go:38-112`
- `internal/diagram/graph.go:87-94`; `internal/diagram/graph_test.go:14-60`
- `internal/cli/fmt_test.go:1882-1886`, `:2000`; `internal/cli/validate_test.go:37-48`
- `tasks/learnings.md:191-194` "All three spellings of `reads` resolve, and widening a reference check moves the
  models first"
- `tasks/learnings.md:121-124` "`emod fmt` canonicalises order, so a fmt golden is never the input re-indented"
- `tasks/learnings.md:41-44` "Additive output changes owe a byte-identical receipt for models that do not use the
  feature"
- `tasks/learnings.md:6-9` "CLI diagnostic tests must assert the distinguishing message text"
- `tasks/learnings.md:406-409` "`internal/lsp` walks the model once now"
- `tasks/learnings.md:1010-1015` "fold by mechanism not by file"

**Testable:** Yes. Through `parser.Parse`, `formatter.Format`, `cli.RunFmt`, `cli.RunValidate`,
`validator.Validate`, `linter.Lint`, `query.References`, `query.Trace`, the LSP definition and references
handlers, `arrange` and `diagram.SliceEdges`.

**Certainty:** medium: `subscribes` is a list of names on each of these surfaces (`parser.go:125-140`,
`formatter.go:164`, `validator.go:413-417`, `names.go:117-122`, `arrange.go:127-131`, `graph.go:87-94`), but no
attribute accepts both a bare name and a list.

**Blast radius:** high: the type change forces edits into the code of both viewer save paths, which rewrite
the author's `.emod` file, and into the code of the JSON, CUE and diagram-json exports.

**Verification:** `task test:unit && task test:integration`; `go build ./...`; `go vet ./...`.

**Depends on:** None

---

### Task 2: Carry each entry through the diagram document, both viewer saves and the detail panel

**Behavior:** The trigger node of the diagram document gives `reads` as a list of each entry, also for one
entry. An automation node and a translation node keep one name. The document has one `reads` edge from each
view in the list, or from the copy of the view in the trigger's slice when a slice two or more slices away
declares the view. Both save paths keep the list. `ImportDiagram` builds a trigger that reads each view that its
node lists and each further view that a `reads` arrow joins to it. `ApplyDiagram` changes one entry for each
`reads` arrow that the viewer adds, removes, renames or moves, and keeps the other entries. A document that
gives a trigger's `reads` as one name, as a document written before this change does, imports as a list of one.
The detail panel of the viewer lists each view that the trigger reads.

**Acceptance Criteria:**

- [x] The diagram document of a model whose trigger reads `[CartView, DiscountView]` gives the trigger node
      `"reads": ["CartView", "DiscountView"]`. The node of a trigger that reads one view gives
      `"reads": ["CartView"]`. No trigger node gives `reads` as a string. The `reads` of an automation node and
      of a translation node stays one string.
- [x] The diagram document has one `reads` edge into the trigger node from each view in the list. For a view
      that a slice two or more slices away declares, the edge starts at the `view_copy` node in the trigger's
      slice.
- [x] `ImportDiagram` on the diagram document of a model whose trigger reads two views, then `formatter.Format`,
      gives back the source, with `reads = [CartView, DiscountView]`.
- [x] In `ImportDiagram`, a `reads` edge from a view, or from a copy of a view, onto a trigger adds that view
      after the views that the node lists, unless the trigger reads it already. A `reads` edge from a command or
      an event adds nothing. An automation and a translation keep the rule that they have today.
- [x] A diagram document whose trigger node gives `"reads": "CartView"` imports a trigger that reads
      `[CartView]`, and the detail panel shows `CartView` for that node.
- [x] In `ApplyDiagram`, for a trigger that reads `[CartView, DiscountView]`: removing the `reads` arrow from
      `CartView`, or deleting the view `CartView`, gives `reads = DiscountView`; renaming `CartView` to
      `BasketView` gives `reads = [BasketView, DiscountView]`; a new `reads` arrow from a third view adds that
      view to the end of the list; moving the arrow from `CartView` to another view replaces `CartView` and keeps
      `DiscountView`. Each edit leaves the other entries and the rest of the source as they were.
- [x] Removing the `reads` arrow from the copy of a far view removes that entry only.
- [x] The detail panel of a trigger lists each view that it reads, in list order, and shows the placeholder
      for a missing value when it reads none. The rows of the automation and translation sections do not
      change.
- [x] `task test:unit`, `task test:integration` and `task test:viewer` pass.

**Affected Files/Modules:**

- `internal/export/diagram.go`: `jsonDiagramNode`, `appendTrigger`; `internal/export/export_test.go`, group
  "diagram json".
- `internal/importer/importer.go`: `diagramNode`, `asTrigger`, `foldReads`; `importer_test.go`.
- `internal/importer/diagram_edit.go`: `clearRelationship`, `pendingRename`, `setReads`, and the pair of
  arrows that a move gives; `diagram_edit_test.go`.
- `internal/frontend/static/ui.js`: the trigger section of `showDetailPanel`;
  `internal/frontend/tests/detail-panel.test.js`.

**Patterns to Follow:**

- `internal/importer/diagram_edit.go:121-165`, `:558-560`, `:621-636`
- `internal/importer/importer.go:401-404`, `:475-482`
- `internal/importer/diagram_edit_test.go:1000-1030`, `:1426-1450`, `:2247-2280`
- `internal/importer/importer_test.go:944-1030`
- `internal/frontend/static/ui.js:545-551`; `internal/frontend/tests/detail-panel.test.js:175-196`
- `internal/export/export_test.go:2065-2070`
- `tasks/learnings.md:256-259` "A diagram-node key has three readers, and they must move in one commit"
- `tasks/learnings.md:201-204` "The viewer's save path is `importer.ImportDiagram`, so a diagram-node field owes
  a read-back"
- `tasks/learnings.md:251-254` "A key rename owes a retired-key negative assertion on every surface that reads the
  key"
- `tasks/learnings.md:296-299` "The viewer shows a node twice"
- `tasks/learnings.md:206-209` "`internal/viewer/static` is a display surface with its own vitest harness" (the
  viewer lives in `internal/frontend` now)
- `tasks/learnings.md:1620-1625` "A lane-diagram name is not an identity"

**Testable:** Yes. Through `export.ExportDiagramJSON`, `importer.ImportDiagram`, `importer.ApplyDiagram` and
`UI.showDetailPanel`.

**Certainty:** low: no key of the diagram node has a shape that depends on the node type, and the importer
has no reader that accepts one name or a list.

**Blast radius:** high: both save paths rewrite the author's `.emod` file, so an error deletes a view from
the source; the diagram document is an output of `emod export --format diagram-json` and of the MCP
`emod_export` tool.

**Verification:** `task test:unit && task test:integration && task test:viewer`.

**Depends on:** Task 1

---

### Task 3: Give the reads of a trigger as a list in the JSON and CUE exports and the schema

**Behavior:** `emod export --format json` and `emod export --format cue` give the `reads` of a trigger as a
list of strings, also for one entry. The JSON key `reads_position` gives one position for each entry, in the
same order. `#Trigger` in `internal/cue/schema.cue` declares `reads` as a list of strings. The `reads` of an
automation and of a translation stays one string.

**Acceptance Criteria:**

- [ ] The JSON export of a trigger that reads `[CartView, DiscountView]` gives
      `"reads": ["CartView", "DiscountView"]` and a `reads_position` list with the line and column of each
      entry, in the same order. A trigger that reads one view gives a list of one in both keys. A trigger that
      reads none gives neither key.
- [ ] The CUE export of the same trigger gives `reads: ["CartView", "DiscountView"]`, and a list of one for one
      entry.
- [ ] Both exports carry the list for a trigger in the slice of an aggregate and for a trigger in a slice that
      a context declares directly.
- [ ] The JSON and CUE exports of a model whose trigger reads two views describe the same model, and the JSON
      export conforms to `#Model` under `cue vet`.
- [ ] `cue vet -d '#Model'` rejects a document whose trigger gives `reads` as one string, and its output names
      `reads`.
- [ ] Both exports give the `reads` of an automation and of a translation as one string, as they do today.
- [ ] `task test:unit` and `task test:integration` pass.

**Affected Files/Modules:**

- `internal/export/json.go`: `jsonTrigger`, `convertTrigger`.
- `internal/export/cue.go`: `writeTrigger`.
- `internal/cue/schema.cue`: `#Trigger`.
- `internal/export/export_test.go`, `internal/cue/embed_test.go`, `internal/cli/export_test.go` (`:543`).

**Patterns to Follow:**

- `internal/export/json.go:158-167`, `:674-688`
- `internal/export/cue.go:36-41`, `:232`
- `internal/cue/schema.cue:52-58`
- `internal/export/export_test.go:1000-1043`, `:4671-4720`, `:6505-6520`
- `internal/cue/embed_test.go:181-203`
- `tasks/learnings.md:46-49` "A new exported field must land in JSON, CUE and `schema.cue` in the same change"
- `tasks/learnings.md:141-144` "The two export guards cannot see a list neither writer emits"
- `tasks/learnings.md:251-254` "A key rename owes a retired-key negative assertion on every surface that reads the
  key"

**Testable:** Yes. Through `export.ExportJSON`, `export.ExportCUE`, `cli.RunExport` and `cue vet` against the
embedded schema.

**Certainty:** medium: `subscribes` is a list on all three surfaces (`json.go:166`, `cue.go:232`,
`schema.cue:57`), but no position key of the JSON export holds a list.

**Blast radius:** high: the output of `emod export --format json` and `--format cue`, which the MCP
`emod_export` tool also returns, and the published schema change for each model with a trigger that reads a
view.

**Verification:** `task test:unit && task test:integration`.

**Depends on:** Task 1

---

### Task 4: Print each entry in the ASCII diagram, and pin the SVG arrows and copies

**Behavior:** The ASCII diagram prints one `{CartView} -> <<Checkout Page>>` line for each entry of a trigger's
`reads`, also for one entry. These lines take the place of the trigger's own `<<Checkout Page>>` line, as the
`(Event) -> {View}` lines of a view take the place of its `{View}` line. A trigger that reads no view prints its
own line as today. The SVG draws one arrow from each view in the list to the trigger, and a copy in the
trigger's slice of each view that a slice two or more slices away declares. The SVG gets this from
SliceEdges (Task 1); this task pins it in the picture.

**Acceptance Criteria:**

- [ ] For a trigger `Checkout Page` that reads `[CartView, DiscountView]`, the ASCII diagram prints
      `{CartView} -> <<Checkout Page>>` and `{DiscountView} -> <<Checkout Page>>`, in list order, at the place of
      the trigger's own line, and prints no separate `<<Checkout Page>>` line.
- [ ] A trigger that reads one view prints one such line. A trigger with an actor appears as the ASCII diagram
      labels it, for example `{CartView} -> <<Checkout Page (Guest)>>`.
- [ ] A trigger that reads no view prints `<<Checkout Page>>` as today, and the ASCII diagram of a model whose
      triggers read no view does not change.
- [ ] The ASCII diagram still prints no line for the `reads` of an automation. The Mermaid diagram of a model
      whose triggers read views is the same as the Mermaid diagram of that model with no trigger `reads`.
- [ ] The SVG of a model whose trigger reads two views of its own slice draws one arrow from each view box to
      the trigger box, with the paint of the arrow from a view to a reader in the same picture, and draws no
      other new arrow.
- [ ] The SVG of a model whose trigger reads a view that a slice two or more slices away declares draws a copy
      of that view in the trigger's slice, and the arrow starts at the copy. A view of the next slice gets no
      copy.
- [ ] `task test:unit` passes.

**Affected Files/Modules:**

- `internal/diagram/ascii.go`: the trigger lines of `ExportASCII`; `ascii_test.go`.
- `internal/diagram/contract_test.go`: the text-format leaf of `TestExporterReadsEdges`, and the SVG leaves.
- `internal/diagram/svg_test.go`: if a leaf fits better there than in the contract harness.

**Patterns to Follow:**

- `internal/diagram/ascii.go:79-88`, `:119-126`
- `internal/diagram/ascii_test.go:62-100`
- `internal/diagram/contract_test.go:827-930`, `:2246-2271`
- `internal/diagram/contract_test.go:113-120`; `internal/diagram/svg_test.go:2110-2140`
- `internal/diagram/element_copies_test.go:238-265`
- `tasks/learnings.md:331-334` "Read an SVG arrow back as the two boxes it meets"
- `tasks/learnings.md:316-319` "An arrow between two constructs is drawn by five surfaces, and none of them reads
  another"
- `tasks/learnings.md:41-44` "Additive output changes owe a byte-identical receipt for models that do not use the
  feature"
- `tasks/learnings.md:1010-1015` "fold by mechanism not by file"

**Testable:** Yes. Through `diagram.ExportASCII`, `diagram.ExportMermaid` and the SVG export that the contract
harness runs.

**Certainty:** medium: the ASCII line repeats the `(Event) -> {View}` lines of a view (`ascii.go:79-88`), and
the SVG leaves repeat `contract_test.go:854-876` and `:2246-2271`, but ASCII now draws the `reads` of a trigger
and still draws none for an automation, and the text-format leaf of `TestExporterReadsEdges`
(`contract_test.go:832-850`) treats both as one rule.

**Blast radius:** low: the ASCII diagram is a preview for a person in a terminal and the SVG is a picture; no
tool parses either.

**Verification:** `task test:unit`.

**Depends on:** Task 1

---

### Task 5: Document the list form in the DSL reference

**Behavior:** `docs/dsl-reference.md` documents that the `reads` of a trigger takes a bare name or a list, how
`emod fmt` writes each, that each entry resolves and is reported at its own position, that each entry draws its
own arrow and its own copy, and that the JSON and CUE exports give the list, also for one entry. The site copy
says that a trigger names the views that it reads.

**Acceptance Criteria:**

- [ ] The Command Pattern section of `docs/dsl-reference.md` shows the `reads` of a trigger as a bare name or a
      list, and says that an automation and a translation take one name.
- [ ] It says that `emod fmt` writes one entry as a bare name and two or more entries as a list.
- [ ] It says that the SVG and the viewer draw one arrow from each view in the list, with a copy for each far
      view, and that the ASCII diagram prints one line for each entry.
- [ ] It says that `emod export --format json` and `emod export --format cue` give the `reads` of a trigger as a
      list, also for one entry.
- [ ] The Cross-References section says that each entry of a trigger's list resolves alone, and that an
      undeclared entry is reported at that entry.
- [ ] `site/content/docs/dsl-reference.md:106` says that a trigger names the views that the actor reads.
- [ ] Each ```` ```emod ```` fence in `docs/dsl-reference.md` validates: the "documented models" leaf of
      `internal/oracle/oracle_test.go` passes.

**Affected Files/Modules:**

- `docs/dsl-reference.md`: Command Pattern (`:318-341`), the cross-reference table (`:811`), the paragraph on
  `reads` (`:815-819`).
- `site/content/docs/dsl-reference.md`: `:106`.

**Patterns to Follow:**

- `docs/dsl-reference.md:341`, `:278-281`, `:811-819`
- `internal/oracle/oracle_test.go:114-140`
- `tasks/learnings.md:241-244` "`docs/dsl-reference.md` is the one keyword surface no test reaches, and a
  retirement story forgets it"
- `tasks/learnings.md:471-474` "An ```emod fence is a promise that the block validates"

**Testable:** No: prose. The "documented models" leaf checks each ```` ```emod ```` fence that the task adds.

**Certainty:** high: the far-copy bullet of a trigger (`docs/dsl-reference.md:341`) and the export bullet of
invariants (`:280`) are the precedents.

**Blast radius:** low: prose; the MCP server serves the file to a reader, and no tool parses it.

**Verification:** `go test -tags unit ./internal/oracle/...`; read both files.

**Depends on:** Task 1, Task 2, Task 3, Task 4

## Summary

- **Total:** 5 tasks.
- **Order:** dependency first, then risk.
  - Task 1 is the type change. When `Reads` becomes a list, each reader stops compiling, so the commit that
    changes the type rewrites each name lookup and SliceEdges, with a test for each. For this reason the
    suggested folds "AST plus parser plus formatter" and "validator plus linter plus query" are one task. The
    task keeps each serialized form on one name until Tasks 2 and 3.
  - Task 2 comes next. After Task 1, a viewer save of a trigger with two entries can drop a view, because both
    save paths still hold one name. Task 2 closes that first.
  - Task 3 changes the export contract and the schema. Task 4 pins the pictures. Task 5 documents the final
    state, so it comes last.
- **Story criteria:**
  - List accepted, bare name stays valid, each `.emod` file parses: Task 1.
  - `emod validate` reports each entry at its position: Task 1.
  - One arrow for each view, copies of far views, the detail panel: SliceEdges in Task 1; the viewer's arrows,
    copies and panel in Task 2; the SVG in Task 4.
  - One ASCII line for each entry: Task 4.
  - `emod fmt` writes a bare name or a list and settles: Task 1.
  - `view/never-read`, `emod references`, `emod trace`: Task 1.
  - JSON and CUE exports give a list: Task 3. `docs/dsl-reference.md`: Task 5.
- **Deferred:** no criterion of the story. The Boundaries section lists what this story does not do.
- **Decisions made without the user** (each one can be overruled before Task 1 starts):
  - The JSON key `reads_position` keeps its name and holds a list, as `ReadsPos []Position` does.
  - The diagram node gives a list for a trigger and one name for an automation and a translation, and the
    importer reads one name as a list of one, so an older `.json` document still saves.
  - In `ImportDiagram`, a `reads` edge from a further view adds an entry to a trigger, where today the
    recorded view wins. The existing leaf `internal/importer/importer_test.go:970` changes for the trigger half.
  - In the ASCII diagram, the arrow lines of a trigger take the place of its bare line, as a view's lines do.

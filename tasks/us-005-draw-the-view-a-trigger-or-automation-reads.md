# US-005: Draw the view a trigger or automation reads

## Progress
- [x] Task 1: Share a fixture whose triggers read the views they open on
- [x] Task 2: Draw a `reads` edge from a view to a trigger and to an automation in the diagram document
- [ ] Task 3: Fold a viewer-drawn `reads` edge onto a trigger and an automation
- [ ] Task 4: Draw the `reads` edge to a trigger and an automation in SVG and draw.io
- [ ] Task 5: Type a view-to-trigger and view-to-automation arrow as `reads` in the viewer

---

## Story Reference

`user-stories/triggers-and-automations.md` → **US-005: Draw the view a trigger or automation reads**
(fifth of eleven stories in "Triggers and Automations"). Design notes:
`docs/proposals/triggers-and-automations-proposal.md` — `:26` for the wire that has never been drawn,
`:239` for the two new export edges, `:247` for the importer's counterpart, `:297-305` for the edge
list and the viewer's `EDGE_TYPE_BY_ENDS`, `:445` for drawing `Trigger.Reads` in the same pass.

**In scope:** a `reads` edge from a view node to a trigger node and from a view node to an automation
node, in the diagram JSON document, in SVG and in draw.io; the same edge appearing in the web viewer,
which draws whatever the document carries; the viewer minting a `reads` edge when a user drags an
arrow from a view to a trigger or an automation, and `importer.ImportDiagram` folding such an edge
back onto that element as its `reads`; and a trigger or automation that names no view — or names one
no slice declares — drawing exactly what it draws today. Carried along because the story cannot be
measured without it: a shared fixture whose triggers read views the model actually declares, since no
fixture in the tree has one, together with its twin and its read-back getter.

**Out of scope:** dropping the trigger kind slot, so `trigger UI "…"`, `trigger Schedule "…"` and
`trigger Processor "…"` all still parse and every trigger still carries a kind (US-004); lane
placement, so an automation stays exactly where it sits today and both new arrows run between boxes
where they already are (US-006); the palette (US-007); the `automation/missing-todo-list` lint rule,
so this story adds no rule and no `RuleName` (US-008); LSP (US-009); highlighting queries, VS Code
and tree-sitter (US-010); `docs/dsl-reference.md`, `README.md` and `examples/*.emod`, none of which
this story edits (US-011). Also out of scope and deliberately untouched: the language, which already
parses, formats, validates and exports `reads` on all three constructs — this story adds no keyword,
no AST field, no formatter line, no JSON or CUE key and no `schema.cue` change; `internal/glossary`,
where a `reads` value is a reference rather than a term; and `e2e/`, `e2e-viewer/` and
`internal/wasm`, which pass the document through without per-construct knowledge.

**Consequences of that boundary, decided.** Eight shapes the story does not spell out:

1. *The new edges end at the trigger box and at the automation box, and the translation edge is left
   where it points.* The three surfaces disagree today about what a translation's `reads` connects
   to: SVG (`internal/diagram/svg.go:282-289`) and draw.io (`internal/diagram/drawio.go:569-575`)
   draw the arrow from the view to the **external system** box, while the diagram document
   (`internal/export/export.go:1114-1122`) and the viewer's `EDGE_TYPE_BY_ENDS` route it to the
   **translation** node. A trigger and an automation have no external system, so there is no fork to
   resolve for them — both new arrows end at the element itself, matching the document. Repointing
   the translation's arrow to agree would repaint a picture this story does not own and would move
   output for every model with a translation; it stays as written.
2. *Mermaid and ASCII get nothing, and their unchanged output is a receipt rather than an omission.*
   Neither format draws a `reads` edge today: Mermaid writes a `%   reads <View>` comment line
   (`mermaid.go:123-125`, `:235-237`, `:351-353`) and ASCII writes `{View} -> [ExternalSystem]` into
   the translation's text chain (`ascii.go:91-94`). The story's criteria name three outputs twice, and
   its third criterion asks for "the visual treatment the translation `reads` edge already uses" —
   which is an arrow, and neither of those two formats has one. Unlike US-003's cadence, an element
   with no edge prints no hole and no leading comma in either format, so nothing there misreads the
   shape. The contract differential therefore keeps asserting byte-identity for Mermaid and ASCII,
   and that is what shows the change reached exactly the two renderers that draw boxes.
3. *The viewer draws the edge with no change to its renderer.* `internal/viewer/static/renderer.js:300-315`
   styles an edge purely from `edgeConfig[edge.type]`, and `reads` is already in that table
   (`internal/viewer/static/config.js:25`), in `arrowClassMap` (`:35`) and in the focus filter
   (`internal/viewer/static/layout.js:267`). So the viewer's half of criteria 1 and 2 lands the moment
   Task 2 emits the edge, and criterion 3 is satisfied by the type string rather than by any new
   styling. What the viewer does need is the auto-detect table, which has no `view>trigger` and no
   `view>automation` entry, so an arrow a user drags between those ends is minted `flow`
   (`internal/viewer/static/model.js:156`) and then dropped by `foldFlow`.
4. *Only a `reads` naming a declared view draws an edge, and exactly two models in the tree have a
   trigger that does.* The translation edge is already guarded on the view name resolving
   (`viewIDs[t.Reads]` in the exporter, `nameToPos[tr.Reads]` in SVG, `nameToElem[tr.Reads]` in
   draw.io) and the two new edges take the same guard — they have to, because the validator resolves
   an automation's `reads` but leaves a trigger's and a translation's unchecked, on purpose
   (`internal/validator/validator.go:316-318`). Verified across all seven shared fixtures and every
   `.emod` under `examples/`: only `test.KeywordFieldSearchCatalog` (trigger `"Search Builder"` reads
   `SavedSearchesView`, declared) and `examples/all_patterns.emod` (trigger `"Reservation Form"` reads
   `AvailableRoomsView`, declared at `:43`) have a trigger whose `reads` resolves. Every other trigger
   in the repo names `AvailableRoomsView`, `AvailableCopiesView` or a webhook view no slice declares,
   draws nothing, and is untouched. The two differentials that render `KeywordFieldSearchCatalog`
   (`internal/diagram/contract_test.go:224-225`, `internal/export/export_test.go:2993-2994`) compare
   it against a twin that renames fields and keeps the view, so both sides gain the same edge and both
   still hold.
5. *The fold requires the source node to be a view.* `foldEdges`' `reads` arm
   (`internal/importer/importer.go:287-290`) type-asserts the target and then takes `src.Label`
   without checking what the source is, so an edge whose source end was dragged off the view still
   writes that node's label. That is harmless for a translation, whose `reads` nothing resolves; it is
   not harmless for an automation, whose `reads` the validator checks, because the fold would produce
   a file `emod validate` rejects. The shape is reachable: `applyRepoint`
   (`internal/viewer/static/interaction.js:284-299`) moves an edge's endpoints and never recomputes
   its type. Both new arms therefore require the source node's type to be `view`, following `foldFlow`
   (`internal/importer/importer.go:295-297`), which already type-checks both ends. The translation arm
   keeps the behaviour it has; changing it is not this story's business.
6. *Node metadata still wins over the edge, and deleting the edge does not clear the value.* A
   trigger's and an automation's `reads` already ride on the node (`internal/export/export.go:809`,
   `:849`) and are read back in `buildSlice` (`internal/importer/importer.go:190`, `:232`), so the fold
   exists only for an edge a user drew in the viewer. Both new arms keep the `== ""` guard the
   existing arms use, which is what stops an exported-then-reimported model from having its value
   rewritten by the edge the exporter itself emitted. The consequence, identical to the translation's
   today, is that removing the arrow in the viewer does not remove the `reads` — the metadata
   survives. Making a deletion stick would need the viewer to clear the node field too, which is a
   different feature.
7. *A new shared fixture rather than an edit to an existing one.* No shared fixture has a trigger
   whose `reads` resolves except `KeywordFieldSearchCatalog`, whose role is keyword-named fields and
   whose doc comment says so. `AutomationReadsLibraryLending` covers the automation half and already
   ships the twin and the getter, and its trigger reads the undeclared `AvailableCopiesView` — which
   makes it this story's witness that a `reads` naming no declared view draws no edge. Declaring that
   view inside it instead would move its output in four packages at once. So the trigger half gets its
   own fixture, and `git diff` leaves every existing fixture alone.
8. *Both arrows cross a lane or a row boundary, and that is fine here.* In SVG and draw.io a view
   sits in the Commands/Views lane (`svg.go:102-112`, `drawio.go:335-342`) while a trigger and an
   automation sit in the UI/Triggers lane (`svg.go:73-87`, `:149-164`; `drawio.go:306-316`, `:404-416`),
   so both new arrows run upward across a lane boundary. In the viewer a view is the bottom row of a
   slice and automations the top row (`internal/viewer/static/layout.js:58`, `:121-129`), so the arrow
   spans the slice. US-006 moves the boxes; this story draws the arrow between them where they are and
   moves nothing.

**Learnings folded in** from `tasks/learnings.md`: the viewer's save path is `importer.ImportDiagram`,
so a diagram-node field owes a read-back, and `reads` edges in an imported document fold onto
translations only — this story is the one that changes that; a diagram-node key has three readers that
must move in one commit, which is why the exporter, the importer and the viewer's edge table each get
their own task and each carries its own read-back rather than trusting the tags; `internal/viewer/static`
is a display surface with its own vitest harness, run by `task test:viewer`, which is not part of
`task test:unit`; additive output changes owe a byte-identical receipt for models that do not use the
feature, and the diagram packages pay it with a differential rather than an assertion; a differential
receipt must first prove the twin actually differs, and `require.NotEqual` on a stripped twin is
satisfiable without stripping anything, so pair it with a positive check on what must be gone and what
must remain; a new optional field ships a fixture kit whose twins are built on `copyWithEditedSlices`
plus `editedCopies`, and the copies are shallow — an edit reaching inside a slice must copy the thing
it edits or it writes through to the caller's model; a `Declared…` getter answers `nil` for a fixture
declaring none of the construct, so pair it only with a non-empty transcribed list and fold the
assertion into the existing round-trip leaf rather than opening a parallel table; a slice has two
homes and a fixture declaring the construct in only one cannot catch a one-home walk; a new shared
fixture owes `internal/oracle` a zero-diagnostic subtest, with DCB shapes the usual tripwire; only an
automation's `reads` resolves and a trigger's must stay unchecked; an assertion whose expected value
comes from the code under test is the recurring review finding; a second `require.Contains` on one
message is often shadowed by the first; de-duplicate before a fan-out edit and land the
de-duplication with proof; never write emod source with `%q`; run repo tooling through `mise exec --`;
acceptance criteria describe the working tree, and a commit-message receipt is the commit author's
obligation, never a criterion.

---

## Codebase Context

**What already exists in the file and in the AST.** `ast.Trigger` (`internal/ast/ast.go:173-187`)
carries `Reads`/`ReadsPos` after `Actor`; `ast.Automation` (`:202-220`) carries them between
`Schedule` and `Command`; `ast.Translation` (`:222-237`) between `ExternalSystem` and `Command`. All
three parse, format, and export to JSON, CUE and `schema.cue` today. `internal/validator/validator.go:318`
resolves an automation's `reads` against `index.viewNames`; `:316-317` records in a comment why a
trigger's and a translation's stay unchecked. **This story adds no language surface at all** — no
lexer keyword, no parser branch, no formatter line, no tree-sitter rule, no schema key.

**The diagram document.** `jsonDiagramNode` (`internal/export/export.go:657-677`) has one flat
`reads` key shared by the trigger, automation and translation nodes, written at `:809`, `:849` and
`:869` by `collectSliceNodes` (`:748-894`). `jsonDiagramEdge` (`:679-683`) is a bare
source/target/type triple with no id, label or style. `convertModelToDiagram` (`:896-1147`) builds
nodes in a first pass and edges from `:990`, over the name-keyed ID maps built at `:911-914` and
filled for every slice in both homes at `:969` and `:985` — so the maps are **model-wide** and a
cross-context reference resolves. Seven edge type strings are emitted: `flow` (`:1025-1029` and
`:1134-1138`), `trigger_command` (`:1035-1048`), `subscription` (`:1064-1068`),
`automation_trigger` (`:1085-1089`), `automation_command` (`:1095-1099`), `reads` (`:1114-1122`,
translations only) and `translation_command` (`:1126-1130`).

**The importer.** `diagramNode` (`internal/importer/importer.go:25-43`) decodes the same keys;
`buildSlice` (`:167-243`) copies `reads` onto the trigger (`:190`), the automation (`:232`) and the
translation. `foldEdges` (`:250-293`) writes edge-carried relationships back, with five cases; its
`reads` arm (`:287-290`) type-asserts the target to `*ast.Translation` and guards on `Reads == ""`,
so an edge onto a trigger or an automation is silently discarded. `foldFlow` (`:295-297`) is the one
arm that checks both endpoint *types*.

**SVG.** `ExportSVG` (`internal/diagram/svg.go:12`) draws every box first, then a `// --- Connections ---`
block from `:201`, so arrows paint over boxes. Boxes are collected into a flat `elems` list and
flattened into a **global, name-keyed** `nameToPos` at `:204-208`; an edge is two map lookups plus the
two box centres. The single `reads` arrow is `:282-289`, inside the translation loop `:273-311`, whose
opening guard (`:276-279`) skips the whole translation when the external system or the reactor is
missing. Every arrow in the file is `svgArrowPath` (`:397-418`) — stroke `#666666`, width `1.5`,
`marker-end="url(#arrow)"` (the marker is defined once in `svgDefs`, `:325-332`), no dash, no label.
There is no per-type styling in SVG.

**draw.io.** `ExportDrawio` (`internal/diagram/drawio.go:100`) holds the layout constants both Go
renderers share (`:53-61`), the vertex styles (`:74-91`), and four edge styles (`:451-454`). Its
`nameToElem` (`:459-463`) is the same global name-keyed lookup, first-write-wins. The single `reads`
edge is `:569-575`, drawn with `standardStyle` (`:451`) — the same style the trigger→command
(`:498-507`), automation→command (`:559-562`) and translation→command (`:586`) edges use, and the one
`edgeCell` (`:803-807`) emits without a `value=`, so no draw.io edge is ever labelled. `extStyle`
(`:454`) is the only dashed one.

**Diagram contract tests.** `internal/diagram/contract_test.go:25-38` defines the `exporter` table
entry: `fillOfLabel`, `countConnections` and `boxes` are nil for the formats that have no such
concept, and `exporters()` (`:57-87`) fills them — `boxes` and `fillOfLabel` for drawio and svg only,
`countConnections` for drawio, svg and ascii. `:261-273` is the differential this story inverts, whose
failure message already names US-005 at `:272`. `TestExporterTranslationEdges` (`:282`) is the nearest
edge test, and its model `translationModel` (`:322-335`) sets no `Reads` at all — so **no test in
`internal/diagram` exercises the existing SVG or draw.io `reads` edge**, and the trigger and
automation edges are greenfield there.

**Export and importer tests.** `internal/export/export_test.go:2296` is the translation `reads` edge
test (asserting source `view-1`, target `trans-1`); `:3033-3047` is the boundary subtest that
currently requires the diagram edge lists of `AutomationReadsLibraryLending` and its twin to be
equal, naming US-005 in the failure message at `:3046-3047`. `:1780` and `:1889` are the trigger and
automation node-metadata subtests, whose models declare no views, so no edge appears in them. Nothing
in the `"diagram json"` group has an absolute `require.Len(t, edges, N)` over a model that declares
the view its trigger or automation reads — the inline models pair `Reads: "V"` with a declared
`MyView` on purpose. `internal/importer/importer_test.go:392-413` is the boundary subtest that
currently requires a `reads` edge onto an automation to be dropped, again naming US-005 at
`:412-413`; `:156` and `:251` are the metadata read-backs, and the round-trip group is at `:38`.

**The viewer.** `internal/viewer/static/config.js:19-27` is `edgeConfig`, whose `reads` entry (`:25`)
gives class `reads-arrow`, stroke `#666666`, marker `url(#arrowhead)` and no dash; `arrowClassMap`
(`:29-37`) mirrors it at `:35`. `renderer.js:300-315` draws an edge from `edgeConfig[edge.type]` and
**skips any edge whose type is not in that table** (`:301-302`). `model.js:140-148` is
`EDGE_TYPE_BY_ENDS`, keyed `srcType + ">" + tgtType`, with a comment at `:136-139` stating the
directions must match what the exporter writes because the importer reads them back;
`autoDetectEdgeType` (`:150-157`) falls back to `flow`. A user draws an edge by dragging from a port
(`renderer.js:270-298`) through `settleConnect` (`interaction.js:314-335`) into `addEdge` (`:301-312`),
which rejects only self-loops and exact duplicates. `layout.js:264-272` lists `reads` among the types
that follow a dragged node. `ui.js:337` and `:358` already print a `Reads` row for a trigger and an
automation in the details panel, so nothing is owed there. Tests: `internal/viewer/tests/model.test.js:240`
holds the `autoDetectEdgeType` table (`:248-256`, with `['view', 'translation', 'reads']` at `:254`)
and `renderer.test.js` the SVG harness; `task test:viewer` runs them and is not part of
`task test:unit`. `web/static/` is a **generated, gitignored** copy of `internal/viewer/static`
(`Taskfile.yml:49-52`) — never edit it.

**Fixtures.** `internal/test/fixtures.go` holds `AutomationReadsLibraryLending` (`:578-735`, doc
`:571-577`) with `AutomationReadsLibraryLendingViewNames` (`:934-938`),
`…ActivationEvents` (`:945-950`), `WithoutAutomationReads` (`:1005-1012`) and
`DeclaredAutomationReads` (`:1064-1066`). The twin machinery is `copyWithEditedSlices` (`:1019-1035`)
and `editedCopies` (`:1037-1046`), whose doc comments record the two traps — a copied slice still
points at the original's children, and a nil list is left nil on purpose.
`declaredAutomationEntries` (`:1087-1097`) generalises the getters over `declaredSlices` (`:1099-1108`),
the walk that visits an aggregate's slices and then a context's own. `internal/test/models.go:36-40`
is the accessor shape, `parseFixture` at `:50-61`. `internal/oracle/oracle_test.go:24` keeps one
zero-diagnostic leaf per fixture; `internal/formatter/formatter_test.go:594` is the round-trip group,
with a per-feature leaf at `:906` and a shared per-fixture table at `:928-960`.

**Not touched, deliberately.** `internal/lexer`, `internal/ast`, `internal/parser`,
`internal/formatter` (beyond one round-trip case for the new fixture), `internal/validator`,
`internal/cue`, the model JSON and CUE documents, `internal/glossary`, `internal/linter`,
`internal/lsp`, `internal/cli`, `editors/`, `docs/`, `README.md`, `examples/`,
`internal/parser/testdata/`, `internal/wasm`, `e2e/`, `e2e-viewer/` and the generated `web/`.

---

## Tasks

### Task 1: Share a fixture whose triggers read the views they open on

**Behavior:** `internal/test` gains a model whose triggers name views the model actually declares —
the shape no fixture in the repo has today — alongside automations that read views and, mid-model, a
trigger and an automation that read nothing. The fixture declares the construct in both homes a slice
has and reaches across contexts, so every package downstream measures both new edges and their absent
counterparts on one model. A twin returns a copy with every trigger's `reads` cleared and everything
else intact, and a getter reads the values back off a parsed model.

**Acceptance Criteria:**
- [ ] `internal/test/fixtures.go` gains a fixture constant named for its feature and domain in the
      shape of `AutomationReadsLibraryLending` (`:578`), with a doc comment stating the role it plays:
      the model whose triggers read views the model declares
- [ ] The fixture declares a trigger whose `reads` names a view declared in **another slice** and a
      trigger whose `reads` names a view declared in **another context**, so a lookup scoped to the
      slice or the context reads back short against the transcribed list
- [ ] The fixture puts a trigger in a slice nested in an aggregate and a trigger in a slice declared
      directly on a `mode dcb` context, and does the same for its automations, so a walk reaching only
      one home reads back short
- [ ] The fixture carries a trigger that reads nothing and an automation that reads nothing, each
      followed by further declarations rather than sitting last, so the absent case cannot be answered
      by an entry with nothing after it
- [ ] `internal/test/models.go` gains the parsing accessor for it, in the shape of
      `AutomationReadsLibraryLendingModel` (`:36-40`)
- [ ] A hand-transcribed exported list names every view the fixture's triggers read, both slice homes
      together and in declaration order, and is non-empty; a second transcribed list does the same for
      its automations, or the existing `DeclaredAutomationReads` is paired with a transcribed list of
      this fixture's own
- [ ] A `Declared…` getter for a trigger's `reads` walks `declaredSlices` (`internal/test/fixtures.go:1099`),
      skips the triggers that read nothing and the slices that have no trigger at all, and
      `require.Equal` against the transcribed list holds
- [ ] A `Without…` twin returns a **copy** whose every trigger reads nothing while the model it was
      given still reads every view it was written reading, and whose automations still read theirs —
      asserted with the getter on both, so a twin that strips one home, strips both constructs, or
      writes through to the caller's model fails
- [ ] `test.WithoutAutomationReads` over the new fixture clears its automations' `reads` and leaves
      its triggers' alone, so the two twins are proved to be independent
- [ ] `oracle.Check` over the fixture returns no diagnostics, added as a leaf in the "clean input"
      group (`internal/oracle/oracle_test.go:24`) — lexer, parser, validator and linter all accept it,
      the `mode dcb` context carries the tags and the `decides_on` its events need, and every view the
      fixture declares subscribes to a declared event
- [ ] The fixture joins the shared round-trip table at `internal/formatter/formatter_test.go:928-960`
      as one case, and the reparsed model reads back both transcribed lists — one leaf, not a parallel
      table
- [ ] `git diff` leaves every existing fixture in `internal/test/fixtures.go` untouched and changes no
      expected constant anywhere: the fixture is additive, and the models whose triggers read no
      declared view are this story's untouched witnesses

**Affected Files/Modules:**
- `internal/test/fixtures.go` — the fixture constant beside `AutomationReadsLibraryLending` (`:578`),
  the transcribed lists beside `…ViewNames` (`:934`), the twin beside `WithoutAutomationReads`
  (`:1005`) and the getter beside `DeclaredAutomationReads` (`:1064`)
- `internal/test/models.go` — the accessor (`:36-40` is the sibling)
- `internal/oracle/oracle_test.go` — one leaf in "clean input" (`:24`)
- `internal/formatter/formatter_test.go` — one case in the round-trip table (`:928-960`)

**Patterns to Follow:**
- `tasks/learnings.md` "A new optional field ships a six-part fixture kit, not a bespoke model per
  package" — `AutomationReadsLibraryLending` (`internal/test/fixtures.go:578-735`) is the model to
  repeat, including its doc comment's account of why each shape is there
- The twin machinery and its two traps: `copyWithEditedSlices` (`internal/test/fixtures.go:1019-1035`)
  and `editedCopies` (`:1037-1046`), whose comments record that a copied slice still points at the
  original's children and that a nil list is left nil deliberately. A slice's trigger is a single
  pointer field rather than a list, so `editedCopies` does not reach it — copy the trigger itself or
  the twin writes through to the model the caller passed
- `tasks/learnings.md` "`require.NotEqual` on a stripped twin is satisfiable without stripping
  anything" and "A differential receipt must first prove the twin actually differs" — pair the
  inequality with a positive check on what is gone and what remains
- `declaredAutomationEntries` (`internal/test/fixtures.go:1087-1097`) generalises the getter shape;
  `declaredSlices` (`:1099-1108`) is the both-homes walk — `tasks/learnings.md` "A slice has two
  homes, and much of the repo still walks only one"
- `tasks/learnings.md` "A new shared fixture owes `internal/oracle` a zero-diagnostic subtest", with
  its warning that DCB shapes are the usual tripwire, and "A spec is not a reference: a command only a
  spec exercises is still orphaned" for why every command and event still needs its flow
- `tasks/learnings.md` "A `Declared…` getter answers `nil` for a fixture that declares none of the
  construct" — pair each getter only with its non-empty transcribed list, and fold the assertion into
  the existing round-trip table rather than opening a new one
- `tasks/learnings.md` "Exercise an omitted optional part mid-block, never as the last entry" — the
  reading and non-reading triggers interleave
- `tasks/learnings.md` "Never write emod source with `%q`" — the fixture is literal emod text
- Leave `test.AutomationReadsLibraryLending` exactly as it is: its trigger reads the undeclared
  `AvailableCopiesView`, and that is this story's witness that an unresolvable name draws nothing

**Testable:** Yes — through `oracle.Check`, `formatter.Format` and the exported `internal/test`
getters.

**Verification:** `mise exec -- go test -tags unit ./internal/test/... ./internal/oracle/...
./internal/formatter/...`; `mise exec -- go test -tags unit ./...` shows no other package needing an
edit.

**Depends on:** None

---

### Task 2: Draw a `reads` edge from a view to a trigger and to an automation in the diagram document

**Behavior:** `export.ExportDiagramJSON` emits a `reads` edge from a view node to a trigger node and
from a view node to an automation node, the same type string and the same direction the translation
edge already uses, whenever the named view resolves to a node the document carries. Because the web
viewer draws whatever edges the document holds and already knows how to paint a `reads` edge, this is
also what makes both wires appear in the viewer. A trigger or automation that reads nothing, or reads
a name no slice declares, contributes no edge and its node is unchanged.

**Acceptance Criteria:**
- [ ] The diagram document for a model whose trigger reads a declared view carries one more edge than
      before, typed `reads`, whose source is that view's node id and whose target is the trigger's
      node id — matched by id rather than by position in the edge list
- [ ] The same holds for an automation whose `reads` names a declared view, with the automation's node
      id as the target
- [ ] A trigger and an automation whose `reads` names a view no slice in the model declares contribute
      no edge, asserted on one model that also carries a resolving `reads`, so the guard is proved to
      be a guard rather than the feature being absent
- [ ] A trigger and an automation that read nothing contribute no edge, asserted on the same model
- [ ] Exporting the shared fixture from Task 1 produces one `reads` edge per view its triggers read
      and one per view its automations read, across both slice homes, read back against the
      transcribed lists rather than against a count
- [ ] The view an automation reads in another context resolves to that view's node, so the edge is
      proved to cross a context boundary rather than being scoped to the slice
- [ ] The document for the fixture's trigger twin loses exactly the trigger edges and keeps every
      automation edge, and the document for its automation twin loses exactly the automation edges and
      keeps every trigger edge — the two differentials that pin each edge to its own construct
- [ ] `internal/export/export_test.go:3033-3047` no longer requires the edge lists of
      `test.AutomationReadsLibraryLendingModel(t)` and its twin to be equal: the reading model now
      carries a `reads` edge per view its automations read and the twin carries none, asserted against
      `test.AutomationReadsLibraryLendingViewNames` rather than against a bare count
- [ ] The trigger of `test.AutomationReadsLibraryLendingModel(t)`, whose `reads` names the undeclared
      `AvailableCopiesView`, still contributes no edge to that document
- [ ] The existing translation `reads` edge is unchanged in type, direction and endpoints —
      `internal/export/export_test.go:2296` passes unedited
- [ ] Every `require.Len(t, edges, N)` already in the `"diagram json"` group holds with its current
      number, and no expected constant in `internal/export/export_test.go` describing a model whose
      trigger or automation reads no declared view is edited
- [ ] The node payload is untouched: `jsonDiagramNode`'s `reads` key still carries the value for all
      three constructs, and the trigger and automation node subtests (`:1780`, `:1889`) pass unedited

**Affected Files/Modules:**
- `internal/export/export.go` — the trigger edge block (`:1035-1048`) and the automation edge block
  (`:1075-1102`) of `convertModelToDiagram`, beside the translation `reads` edge (`:1114-1122`)
- `internal/export/export_test.go` — leaves in the `"diagram json"` group, and the boundary subtest at
  `:3033`

**Patterns to Follow:**
- The edge to copy, wholesale: the translation `reads` edge (`internal/export/export.go:1114-1122`) —
  the same `"reads"` string, the same view-to-element direction, the same "skip unless the view id
  resolves" guard. `jsonDiagramEdge` (`:679-683`) needs no new field
- The ID maps are model-wide (`internal/export/export.go:911-914`, filled at `:969` and `:985`), which
  is what lets an automation read a view another context declares — `tasks/learnings.md` "A slice has
  two homes, and much of the repo still walks only one"
- `tasks/learnings.md` "Read a decoded export document back with `objectsUnder`/`statedUnder`, in the
  writer's slice order" — the read-back helpers live at `internal/export/export_test.go:4177-4220`, and
  `diagramAutomationReads` is the existing per-construct read-back to sit beside
- `tasks/learnings.md` "An assertion whose expected value comes from the code under test is the
  recurring review finding" — name the source and target node ids the assertion expects; do not
  rebuild them from the exporter
- `tasks/learnings.md` "De-duplicate before a fan-out edit, and land the de-duplication with proof" —
  the guard-and-append shape is about to appear three times in one function; if it is extracted, carry
  a differential showing the edge list for an existing fixture did not move
- The three surfaces that read a diagram-node key must move together (`tasks/learnings.md` "A
  diagram-node key has three readers"); an *edge* has two — this exporter and the importer — and Task
  3 is the other half. The viewer is the third reader of an edge's type and needs no change to paint
  it, because `reads` is already in `internal/viewer/static/config.js:25`
- Do not repoint the translation edge at the reactor; decision 1 in the Story Reference records why

**Testable:** Yes — through `export.ExportDiagramJSON`.

**Verification:** `mise exec -- go test -tags unit ./internal/export/...`; `mise exec -- go test -tags
unit ./...`.

**Depends on:** 1

---

### Task 3: Fold a viewer-drawn `reads` edge onto a trigger and an automation

**Behavior:** `importer.ImportDiagram` reads a `reads` edge whose target is a trigger or an automation
back as that element's `reads`, so an arrow a user drags in the viewer from a view to either element
survives the save. The value already recorded on the node still wins, and an edge whose source is not
a view is ignored rather than written through as a view name.

**Acceptance Criteria:**
- [ ] Importing a document whose `reads` edge runs from a view node to a trigger node yields a model
      whose trigger reads that view's label, with the trigger's node carrying no `reads` of its own
- [ ] The same holds for a `reads` edge whose target is an automation node
- [ ] The existing behaviour for a translation is unchanged, asserted on one document that carries all
      three edges, so the three arms are proved to coexist —
      `internal/importer/importer_test.go:392-413` becomes that subtest and its `require.Empty` on the
      automation is replaced by the view's name
- [ ] A `reads` edge onto a trigger or an automation whose node already states a `reads` leaves the
      node's value in place, so the exporter's own edge cannot overwrite what the node recorded
- [ ] A `reads` edge whose source node is a command or an event, and whose target is an automation or
      a trigger, is ignored and leaves that element reading nothing — the guard that stops an arrow
      repointed off the view from producing a model `emod validate` rejects
- [ ] Exporting the Task 1 fixture to a diagram document and importing it back yields triggers and
      automations reading exactly the transcribed lists, in both slice homes and across contexts, and
      formatting that reimported model against formatting the original leaves the `reads` lines in
      place — the viewer's save path is export → import → format
- [ ] The same export → import over each of the two twins reads back no trigger `reads` and no
      automation `reads` respectively, while the model each was copied from still carries all of
      them, so neither direction can be answered by a builder that hardcodes or drops the value
- [ ] A round-trip leaf over a hand-written non-dcb source already in canonical `emod fmt` form —
      declaring a view in one slice and, in a sibling slice, a trigger and an automation that read it
      beside a trigger and an automation that read nothing — formats back to identical bytes
- [ ] `examples/all_patterns.emod`, whose trigger reads the declared `AvailableRoomsView` and which
      therefore gains an edge in the document from Task 2, still round-trips to identical bytes: the
      node metadata already carried the value and the edge may not duplicate or change it
- [ ] The five existing round-trip leaves in `internal/importer/importer_test.go` (group at `:38`) and
      the metadata read-backs at `:156` and `:251` pass unedited

**Affected Files/Modules:**
- `internal/importer/importer.go` — the `reads` arm of `foldEdges` (`:287-290`)
- `internal/importer/importer_test.go` — the boundary subtest at `:392-413`, a leaf in the `"edges"`
  group, and a leaf in `"round trip"` (`:38`)

**Patterns to Follow:**
- The arm to extend: the translation case (`internal/importer/importer.go:287-290`), including its
  `== ""` guard, which is what keeps node metadata authoritative —
  `tasks/learnings.md` "The viewer's save path is `importer.ImportDiagram`, so a diagram-node field
  owes a read-back"
- Endpoint type-checking: `foldFlow` (`internal/importer/importer.go:295-297`) already rejects an edge
  whose ends are the wrong types, and is the precedent for requiring the source node to be a view.
  Decision 5 in the Story Reference records why the new arms need it and the translation arm keeps its
  behaviour
- `tasks/learnings.md` "A mutually exclusive alternative has no stripped twin, and the importer must
  not fold one in" — the `automation_trigger` arm (`:273-280`) is the precedent for an arm whose guard
  exists to stop the fold producing text the parser or validator rejects
- The byte-level round-trip shape and why a hand-written source rather than a fixture:
  the leaves comparing `formatter.Format(importFrom(t, source))` against the source itself in
  `internal/importer/importer_test.go`, with `importFrom` as the export → import path the viewer uses
- `documentKeying`-style closures (`internal/importer/importer_test.go`) are the shape for importing
  one document under two spellings — `tasks/learnings.md` "A key rename owes a retired-key negative
  assertion on every surface that reads the key"
- Leave the missing `trigger_command` arm alone: the exporter emits that edge and the importer has
  never read it, which is harmless because co-slice membership already implies it

**Testable:** Yes — through `importer.ImportDiagram`, `export.ExportDiagramJSON` and
`formatter.Format`.

**Verification:** `mise exec -- go test -tags unit ./internal/importer/... ./internal/export/...`;
`mise exec -- go test -tags unit ./...`.

**Depends on:** 2

---

### Task 4: Draw the `reads` edge to a trigger and an automation in SVG and draw.io

**Behavior:** the two renderers that draw boxes draw an arrow from the view a trigger reads to that
trigger, and from the view an automation reads to that automation, with the same paint the translation
`reads` arrow already carries in each format. A trigger or automation that reads nothing, or reads a
name no slice declares, draws exactly what it draws today — same boxes, same coordinates, same fills,
one fewer arrow. Mermaid and ASCII are unchanged.

**Acceptance Criteria:**
- [ ] The SVG for a model whose trigger reads a declared view draws one more arrow than the same model
      without that `reads`, running between the view's box and the trigger's box, with the stroke,
      width, arrowhead marker and absence of a dash that `svgArrowPath` (`internal/diagram/svg.go:397-418`)
      gives the translation `reads` arrow
- [ ] The same holds for an automation whose `reads` names a declared view
- [ ] The draw.io XML for both cases carries one more `edge="1"` cell each, referencing the view's cell
      id as source and the trigger's or automation's cell id as target, drawn with the style the
      translation `reads` edge uses (`internal/diagram/drawio.go:569-575`) rather than the dashed
      external-system style, and the document still parses as well-formed XML
- [ ] In both formats, a trigger and an automation whose `reads` names a view no slice declares draw
      no arrow — asserted on one model that also carries a resolving `reads`, so the marked and
      unmarked cases are proved together
- [ ] In both formats, a trigger and an automation that read nothing draw no arrow, asserted on the
      same model
- [ ] Rendering the Task 1 fixture in both formats draws one arrow per view its triggers read and one
      per view its automations read, in both slice homes, including the automation reading a view
      another context declares
- [ ] For drawio and svg, the fixture and each of its two twins draw **the same boxes** — every box's
      label, position, size and fill identical, compared through the `boxes` helpers
      (`internal/diagram/contract_test.go:33-35`) — while the connection count differs by exactly the
      number of resolving `reads` the twin lost; the box comparison is what makes "renders exactly as
      before" a receipt rather than an assertion
- [ ] For mermaid and ascii, the fixture and both twins still render byte-identically, so the change
      is proved to have reached exactly the two formats that draw boxes and nothing else
- [ ] `internal/diagram/contract_test.go:261-273` no longer requires the drawio and svg renderings of
      `test.AutomationReadsLibraryLendingModel(t)` and its twin to be equal; its trigger, which reads
      the undeclared `AvailableCopiesView`, still draws no arrow in either format
- [ ] `TestExporterTranslationEdges` (`internal/diagram/contract_test.go:282`) passes unedited, and no
      expected constant in `internal/diagram/svg_test.go`, `drawio_test.go`, `mermaid_test.go` or
      `ascii_test.go` is changed

**Affected Files/Modules:**
- `internal/diagram/svg.go` — the connections block (`:201`), beside the trigger→command arrows
  (`:213-226`) and the automation arrows (`:255-270`)
- `internal/diagram/drawio.go` — the connections block (`:449`), beside the trigger→command edges
  (`:498-507`) and the automation edges (`:539-562`)
- `internal/diagram/contract_test.go` — the differential at `:261-273` and the exporter table's
  capability fields (`:25-38`, `:57-87`)
- `internal/diagram/svg_test.go`, `internal/diagram/drawio_test.go`

**Patterns to Follow:**
- The arrow to copy in each format: `internal/diagram/svg.go:282-289` and
  `internal/diagram/drawio.go:569-575`, both of which look the view up in the global name-keyed map
  (`svg.go:204-208`, `drawio.go:459-463`) and skip when it is absent. Neither takes a label, and no
  edge in either format is ever labelled
- Placement within the file: every arrow is written after every box, so both renderers paint
  connections over shapes (`svg.go:201`, `drawio.go:449`); the new arrows go with the construct that
  owns them, not with the translation block
- `tasks/learnings.md` "Additive output changes owe a byte-identical receipt for models that do not
  use the feature" and "A differential receipt must first prove the twin actually differs" — the
  fixture must be shown to read the views before the twin comparison means anything
- `tasks/learnings.md` "`require.NotEqual` on a stripped twin is satisfiable without stripping
  anything" — pair the box comparison with the getters over both models
- The exporter table's nil-capability convention (`internal/diagram/contract_test.go:25-38`): `boxes`
  and `fillOfLabel` are nil for the text formats, which is the existing way to say "this assertion
  applies to the formats that draw boxes"
- `tasks/learnings.md` "De-duplicate before a fan-out edit, and land the de-duplication with proof" —
  the same guard-and-draw shape now appears three times per renderer; an extraction needs a
  differential and its parameters kept in line with the callers
- Lane placement, the palette and Mermaid's timeframe letters belong to US-006, US-007 and US-006
  respectively; this task moves no box and repaints nothing

**Testable:** Yes — through `diagram.ExportSVG` and `diagram.ExportDrawio`.

**Verification:** `mise exec -- go test -tags unit ./internal/diagram/...`.

**Depends on:** 1

---

### Task 5: Type a view-to-trigger and view-to-automation arrow as `reads` in the viewer

**Behavior:** dragging an arrow in the web viewer from a view to a trigger, or from a view to an
automation, creates a `reads` edge rather than falling through to a generic flow, so the wire the user
drew is the wire the exported `.emod` states. An edge already typed `reads` between those ends is
drawn with the same appearance the view-to-translation arrow has.

**Acceptance Criteria:**
- [ ] An arrow drawn from a view to a trigger is typed `reads`, and one drawn from a view to an
      automation is typed `reads` — added to the table-driven `autoDetectEdgeType` cases at
      `internal/viewer/tests/model.test.js:248-256`, beside the existing view-to-translation case
- [ ] The pairings that already have a type keep it, and a pairing with no entry still falls back to
      `flow` — the existing cases at `:272` and `:277` pass unedited
- [ ] The reverse directions — trigger to view and automation to view — are **not** typed `reads`,
      since the exporter writes the edge in one direction only and the importer reads it back that way
- [ ] Rendering a document containing a `reads` edge from a view node to an automation node draws a
      path in the produced SVG carrying the class, stroke and arrowhead marker
      `internal/viewer/static/config.js:25` defines, and no `stroke-dasharray`; the same holds for a
      trigger target
- [ ] That drawn path's appearance is identical to the one drawn for a view-to-translation `reads`
      edge in the same render, so criterion 3's "the treatment the translation edge already uses" is
      compared rather than restated
- [ ] A document whose automation and trigger nodes carry a `reads` value but no `reads` edge draws no
      such path, so the arrow is proved to come from the edge list rather than from node metadata
- [ ] `internal/viewer/static/config.js`, `renderer.js`, `layout.js` and `ui.js` need no edit: the
      `reads` entry already exists in `edgeConfig` (`:25`) and `arrowClassMap` (`:35`), the focus
      filter already lists it (`layout.js:267`), and the details panel already prints a `Reads` row for
      a trigger (`ui.js:337`) and an automation (`:358`) — `git status --porcelain internal/viewer/static`
      lists `model.js` alone
- [ ] `mise exec -- task test:viewer` passes, and this task changes no Go file
      (`git status --porcelain -- '*.go'` is empty)
- [ ] Nothing under `web/` is edited: it is a generated, gitignored copy of `internal/viewer/static`
      (`Taskfile.yml:49-52`), and `git check-ignore web` succeeds

**Affected Files/Modules:**
- `internal/viewer/static/model.js` — `EDGE_TYPE_BY_ENDS` (`:140-148`)
- `internal/viewer/tests/model.test.js` — the `autoDetectEdgeType` table (`:240-263`)
- `internal/viewer/tests/renderer.test.js` — a leaf for the drawn arrow

**Patterns to Follow:**
- The entry to copy: `"view>translation": "reads"` (`internal/viewer/static/model.js:147`), and the
  comment above the table (`:136-139`) stating that its directions must match what the exporter writes
  because the importer reads them back — Task 2 is the exporter half and Task 3 the importer half
- `tasks/learnings.md` "`internal/viewer/static` is a display surface with its own vitest harness" —
  `installSVGGeometry()` from `tests/svg-env.js` and the dynamic `await import` spelling are required
  for any module touching geometry, and `task test:viewer` is not part of `task test:unit`
- `tasks/learnings.md` "The viewer shows a node twice — the canvas box and the detail panel" — assert
  on what the SVG draws by walking the produced elements, not on `textContent`, and note the
  counterpart here is an edge rather than a node, so the assertion is on the path's attributes
- `tasks/learnings.md` "A diagram-node key has three readers, and they must move in one commit" — an
  edge type has the same three readers, and this task is the last of them; an edge type the renderer
  does not know is silently not drawn (`internal/viewer/static/renderer.js:301-302`), which is the
  regression this task's render leaf exists to prevent
- `tasks/learnings.md` "An assertion whose expected value comes from the code under test is the
  recurring review finding" — comparing the new arrow's appearance against the translation arrow's in
  the same render is the non-arbitrary form; re-reading `edgeConfig` in the test is not
- `applyRepoint` (`internal/viewer/static/interaction.js:284-299`) never recomputes an edge's type and
  is left alone; Task 3's source-type guard is what makes that safe

**Testable:** Yes — through the vitest harness under `internal/viewer/tests`
(`internal/viewer/vitest.config.js`).

**Verification:** `mise exec -- task test:viewer`; `git status --porcelain` lists changes under
`internal/viewer` only.

**Depends on:** 3

---

## Summary

**Five tasks**, ordered dependency-first and, within that, by how much of the story each unblocks.

This story adds nothing to the language — `reads` has parsed, formatted, validated and exported on
both constructs since before US-001 — so there is no parser, formatter, grammar or schema task. What
it adds is one edge, in the two places an edge is written (the diagram document and the two box-drawing
renderers) and the two places one is read (the importer and the viewer's edge table).

Task 1 comes first because the story cannot be measured without it: no fixture in the tree has a
trigger whose `reads` names a view the model declares, so until one exists every criterion about the
trigger edge would be asserted on a model that draws none. Task 2 follows because the diagram document
is the widest surface — it is simultaneously the exporter's output, the importer's input and what the
viewer paints, so emitting the edge there is what makes the wire appear in the viewer at all. Task 3
closes the loop the story's fourth criterion describes and must follow Task 2, since its receipt is
measured on a document Task 2 produces. Task 4 depends only on Task 1 and can be picked up alongside
Tasks 2 and 3; it is placed after them because the viewer half of the story is the part the proposal
calls out as easy to leave a commit behind. Task 5 comes last because it is the third reader of the
edge type Task 2 establishes, and a user-drawn arrow that the importer would drop is exactly the
regression the learnings record from US-002.

**Story criteria coverage:**

| Story criterion | Task |
|---|---|
| A trigger with `reads` renders an edge from the named view to the trigger, in SVG, draw.io and the viewer | 4 (SVG, draw.io), 2 (the viewer, which paints what the document carries), with 1 supplying the only model in the tree that can show it |
| An automation with `reads` renders an edge from the named view to the automation, in the same three outputs | 4, 2 |
| Both edges carry the visual treatment the translation `reads` edge already uses | 4 (`svgArrowPath` and the translation edge's draw.io style), 5 (the `reads` entry in the viewer's edge table, compared against the translation arrow in the same render) |
| An edge drawn in the viewer from a view to a trigger or automation is re-imported as a `reads` entry on that element | 5 (the viewer types the edge), 3 (the importer folds it) |
| A trigger or automation without `reads` renders exactly as before | 1 (the twins and the getters), 2, 3, 4 — each writer carries the criterion for its own surface, and Task 4's same-boxes/one-more-arrow comparison is the receipt |

**Deferred to later stories in the feature:** dropping the trigger kind slot (US-004); lane placement,
so both new arrows cross a lane in SVG and draw.io and span a slice in the viewer until then (US-006);
the palette (US-007); the `automation/missing-todo-list` rule (US-008); LSP (US-009); highlighting
(US-010); and `docs/dsl-reference.md`, `README.md` and `examples/` (US-011).

**Left as it was found, deliberately, and recorded here so a later story can pick it up:** the
translation `reads` arrow in SVG and draw.io ends at the external-system box while the diagram
document and the viewer end it at the translation reactor — three surfaces that disagree about one
edge, which this story neither widens nor repairs; and a `reads` edge deleted in the viewer does not
clear the value, because node metadata carries it independently of the edge.

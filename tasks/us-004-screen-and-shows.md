# Task Breakdown: Declare a screen one time and show it in many slices

## Contents

1. Task 1: Accept `screen` and `shows` beside `trigger`, format them, and check their names
2. Task 2: Draw each appearance with its own arrows in the SVG, ASCII, Mermaid and event flow diagrams
3. Task 3: Answer references and traces about screens, and count their reads in the slice commands
4. Task 4: Offer, describe and paint `screen` and `shows` in the editor
5. Task 5: Give the screens and the appearances in the JSON and CUE exports and in the schema
6. Task 6: Carry screens and appearances through the diagram document, both viewer saves and the viewer
7. Task 7: Move every checked-in model, fixture, sample and document from `trigger` to `screen` and `shows`
8. Task 8: Remove the trigger from the exports, the schema, the diagram document, the importer and the viewer
9. Task 9: Reject `trigger` in the parser, and remove `ast.Trigger` with every reader

## Story Reference

`user-stories/screens-as-first-class.md`, US-004 "Declare a screen one time and show it in many slices" (story
ID `screens-as-first-class/US-004`). Goals G1, G2 and G3 of `docs/proposals/screens-as-first-class-proposal.md`,
phase 2 "Identity". US-001 (a trigger reads a list), US-002 (each trigger keyed by its slice) and US-003 (`actor`
retired) are on main.

Decisions for this run:

- **No released state has both forms.** At the end of the branch a `trigger` block is an error at the keyword
  that names `screen` and `shows`. Inside the branch each commit compiles and passes the suite, so the work goes
  in three phases. Tasks 1 to 6 add `screen` and `shows` beside `trigger`: the AST holds both, and each surface
  reads both. Task 7 moves every checked-in model and document to `screen` and `shows`. Tasks 8 and 9 remove the
  trigger: first from the wire formats, then from the language and the AST. The Summary gives the reason for the
  order.
- **The `command` of a `shows` block is explicit.** A `shows` block draws an arrow only to the commands that its
  `command` names. The migration adds a `command` entry for each command that its trigger issued: each command
  of its slice that no automation and no translation of the slice issues.
- **A screen label is a bare name.** The migration renames each trigger label that holds a space
  (`"Reservation Form"` becomes `"ReservationForm"`). Two triggers with one label in two slices become one
  declaration and two `shows` blocks.
- **One box for each `shows` block, keyed by its slice.** Two `shows` blocks of one screen in two slices draw two
  boxes, each with the arrows of its own slice. A slice holds any number of `shows` blocks. The drawings find
  the box of an appearance by its slice and its screen, so `emod validate` reports a slice that shows one screen
  two times (the second `shows` block, at its `screen` entry). The viewer lets a user move an appearance into a
  slice that already holds one.
- **Serialized names spell the DSL keyword** (`tasks/learnings.md:271-274`). The JSON and CUE exports use the keys
  `screens` (on the model) and `shows` (on a slice), and an appearance uses `screen`, `reads` and `command`, with
  `screen_position`, `reads_position` and `command_position` in the JSON export. The schema definitions are
  `#Screen` and `#Appearance`: a definition is a type name, so it names the concept. The diagram document gives
  each declaration a node of type `screen` with no parent, and each appearance a node of type `shows` in its
  slice. The label of a `shows` node is the name of its screen, as the label of every node is the name drawn in
  its box, so the node has no separate `screen` key. The arrows are `reads` (view to `shows`) and `shows_command`
  (`shows` to command). The viewer palette key is `shows`, and the palette table of `docs/dsl-reference.md`
  section 13 has the row `Shows`.
- **Edge and query kinds.** `diagram.EdgeScreenReads` and `diagram.EdgeScreenCommand` come in beside
  `EdgeTriggerReads` and `EdgeTriggerCommand`, and the trigger kinds go in Task 9. At the end of the branch only
  the screen kinds exist, which is the rename that the proposal asks for. `query.KindScreen` (`screen`) replaces
  `query.KindTrigger` in Task 3. Until Task 9, a trigger reads as kind `screen`.
- **The ASCII diagram prints the command arrows of a screen** (`<<Screen>> -> [Command]`), as it prints the command
  of an automation and of a translation, so that the explicit `command` shows in each drawn format.
- **A viewer save declares a missing screen.** When an appearance names a screen that no declaration holds (a
  renamed `shows` node, or a document with no `screen` nodes), the save adds one declaration for each such name,
  after the declarations that the model already has, so that the result validates.
- **`view/never-read` names a screen** in its message and its description, and the LSP hover texts of `reads`,
  `screen` and `shows` describe the new shape.

## Boundaries

**Out of scope:**

- From the stories file, verbatim: "Layout, style, or the structure of components on a screen. A screen names its
  mockup by its path only."
- From the stories file, verbatim: "Navigation between screens. A screen graph is a different model."
- From the stories file, verbatim: "A mockup for each appearance. A second mockup means a second screen."
- From the stories file, verbatim: "Images inside `.emod` files."
- From the stories file, verbatim: "Changes to how `view`, `automation` or `translation` connect."
- From the stories file, verbatim: "A period where `trigger` and `screen` are both valid, and a new language
  version. The proposal retires `trigger` directly (Versioning section)." `ast.SupportedVersion` stays at 1.
- From the stories file, verbatim: "A `title` attribute that restores a human label with spaces."
- From the stories file, verbatim: "A tool that migrates a model. The migration is by hand, with the steps in the
  proposal." The migration of Task 7 is by hand or by a script that the run discards.
- The `image` attribute of a screen. The proposal's DSL Surface shows `image` in the screen block, but its phase 3
  and US-007 deliver it. In this story a `screen` block accepts `description` only, and `image` is an unexpected
  attribute.
- A check that a screen label is a bare name. As for a command, a label with a space parses, and no `shows` block
  can name it. The viewer does not check that a label a user types is a bare name.
- `orphan-command` does not change: a `command` entry of a `shows` block does not count as a flow that names the
  command, as a trigger did not.
- `emod slices arrange` counts the `reads` of an appearance as a reference and does not count its `command`, as
  for an automation.
- The old diagram document. A saved `.json` diagram document that still has a `trigger` node or a
  `trigger_command` edge opens and imports with no error, and the importer takes no screen and no appearance
  from it. Nothing converts it.
- The kinded trigger forms (`trigger "Schedule" "Nightly Sweep"`). After Task 9 they get the one message that
  every `trigger` block gets, which names `screen` and `shows`, at the keyword.
- A reserved word. `trigger` stays a legal name for a field or a construct. Only the block in a slice is an
  error.
- The words that do not name the construct: the edge type `automation_trigger`, `TriggerCharacters` in the LSP
  protocol, and English prose such as "triggering event" or "retriggering".
- The historical records keep the word: `user-stories/`, `docs/proposals/completed/`, `docs/proposals/ai/completed/`,
  `docs/proposals/screens-as-first-class-proposal.md`, `docs/proposals/hcl-syntax-proposal.md` (implemented, it
  records the syntax change), `tasks/completed/`, `tasks/learnings.md` and the `tasks/*.design.yaml` files of
  other stories.

**Deferred:**

- US-005: `emod references <file> CheckoutPage` for a screen, go-to-definition and find-references on a screen
  name, and completion after `screen =` in a `shows` block. `query.ReferenceKinds` and the LSP's
  `referenceTargetKinds` gain `screen` there. This story gives no value completion after `screen =` or
  `command =`.
- US-006: the viewer highlights the other appearances of a selected screen, and the detail panel lists them.
- US-007: `image`, its exports, its save and `screen/missing-image-asset`.
- US-008: `bed`. US-009: `screen/never-shown`. The shared screen fixture of Task 1 shows each screen that it
  declares, so that US-009 does not have to move it.
- US-010: screens in the glossary. Until then `docs/dsl-reference.md` says that `screen` and `shows` contribute no
  glossary term.

## Codebase Context

**Every reader of the trigger.** gopls finds these production readers of `ast.Slice.Trigger` and `ast.Trigger`
(`internal/ast/ast.go:152-170`, `:296-306`):

- Language: `internal/parser/parser.go:364-365` (the slice case), `:386-423` (`retiredTriggerKind` and
  `r.trigger`, with the retired `actor` attribute at `:414-415`); `internal/formatter/formatter.go:76-85`;
  `internal/validator/validator.go:398-403`; `internal/linter/linter.go:65-90` (the read views of
  `view/never-read`), with its message at `:671` and its description at `internal/linter/descriptions.go:53`.
- Pictures: `internal/diagram/graph.go:73-85` (`SliceEdges`, which gives a trigger an arrow to each command of its
  slice), `svg.go:168-176` and `:285-303`, `ascii.go:51-61` and `:124-126`, `mermaid.go:78-82`, `:173-177`,
  `:302-306`, `eventflow.go:255-263` and `:311-345`. `element_copies.go:138-153` decides which end of each edge
  kind a copy can stand in for.
- Queries: `internal/query/references.go:13`, `:257-264`, `:309-315`; `trace.go:14`, `:164-166`, `:250-253`;
  `names.go:134-138`. `internal/arrange/arrange.go:132-136`. `internal/ast/slice_record.go:44-49`.
- LSP: `internal/lsp/model.go:85-88` (the hover card "Trigger"), `completer.go:483-500` (`keywordsFor`),
  `hover.go:11-45`, `semantictokens.go:91-123`. `internal/test/keywords.go:17` is the list that
  `TestKeywordCoverage` (`internal/lsp/keywords_test.go:18-75`) pairs with hover and completion. The token type
  `parameter` stays in the legend for screens (US-003 left it there for this story).
- Exports: `internal/export/json.go:71`, `:137-146`, `:412`, `:629-643`; `cue.go:113-115`, `:177-182`;
  `internal/cue/schema.cue:15-20`, `:110`.
- Diagram document and saves: `internal/export/diagram.go:182`, `:283`, `:384-400`, `:493-504`, `:533-535`;
  `internal/importer/importer.go:114-121`, `:328-335`, `:448-462`; `diagram_edit.go:182-191`, `:221-235`,
  `:291-354`, `:402-411`, `:456-462`.
- Viewer: `internal/frontend/static/model.js:161` (refuses a second trigger in a slice) and `:264-275`
  (`EDGE_TYPE_BY_ENDS`), `layout.js:93-156`, `:307`, `:317`, `renderer.js:385-389` and `:520-526`, `ui.js:458`,
  `:519-526`, `:799`, `:830`, `config.js:43-50` and `:62-79`, `legend.js`, `viewer.html:942` and `:1282`.
- Test support: `internal/test/fixtures.go:1441-1600` (`TriggerReadsLibraryLending`, the trigger kit, in both
  slice homes), `:3352-3372`, `:3537-3549` (`WithoutTriggerReads`), `:3681-3725` (`DeclaredDescriptions`),
  `:3766-3778` (`DeclaredTriggerReads`); `internal/cli/examples_test.go:293-294`.

**Why the trigger stays until the end.** The checked-in models can move to `screen` and `shows` only after each
guard that runs over them can read the new shape: `TestApplyDiagram` runs a viewer save over each example and
parser testdata file and requires the same text back (`internal/importer/diagram_edit_test.go:542-580`),
`requireSaveKeepsAllButItsLosses` does the same for the shared fixtures through `importer.ImportDiagram`
(`internal/importer/importer_test.go:1327-1334`), the e2e-viewer export tests reproduce `SAMPLE` byte for byte
(`e2e-viewer/tests/helpers.js:3-35`), and the picture and export tests read the shared fixtures. And the trigger
can go only after no checked-in model has one. So each surface learns `shows` first while it still reads
`trigger` (Tasks 1 to 6), the sweep follows (Task 7), and the trigger goes last (Tasks 8 and 9). This is the
order of US-003 turned around: there each surface stopped reading the actor first, because the change removed
something. Here the change adds something before it removes something. `emod fmt` cannot migrate a file, because
the shape changes (one trigger becomes a declaration and an appearance), so the learning "Rename a DSL keyword by
accepting both spellings first" (`tasks/learnings.md:236-239`) gives the order but not the migration.

**Two save paths.** `importer.ApplyDiagram` (`internal/importer/diagram_edit.go:14-22`) reuses the parsed slice of
each unchanged slice node and resets only the fields that it knows (`sliceFor`, `:402-454`), and it replaces
only `model.Contexts`. So it keeps a new AST field that it does not know, which hides a missing read-back.
`importer.ImportDiagram` (`importer.go:229-247`) builds the whole model from the document, so a field that the
document does not carry is lost there. Each change to the document needs a test on both paths
(`tasks/learnings.md:201-204`). Today `appendTrigger` (`importer.go:328-335`) takes the first trigger child of a
slice and drops the rest, and `canHold` (`diagram_edit.go:348-354`) and `moveToSlice` (`model.js:156-174`)
refuse a second trigger in a slice. `trigger_command` edges are never folded: the arrow is inferred, so the
importer has no arm for it.

**Edges are named by their ends.** `diagram.SliceEdges` gives each edge the names of its two ends. The SVG, the
copies, the diagram document and `emod trace` draw from it, and the ASCII, Mermaid and event flow diagrams walk
the AST themselves (`tasks/learnings.md:316-321`). The SVG keeps one trigger box for each slice
(`svg.go:128`, `:175`), centred in the Wireframes lane, and the viewer stacks the trigger children of a slice
in a column (`layout.js:156`). No precedent draws two boxes in the top lane of one slice.

**A command end that can be far away.** A trigger's command is always in its own slice. A `command` entry can
name a command that any slice declares, as an automation's `command` can. `ElementCopies` already copies the
command end of an automation's and a translation's arrow when the declaring slice is two or more slices away
(`element_copies.go:138-153`), and the diagram document finds that end through `copiedEndIDs`
(`diagram.go:545-556`).

**One list on the far end of an arrow.** `diagramEdit.entriesOf` (`diagram_edit.go:182-191`) returns the list on
the edge's target that holds the name of the edge's source (a view's `subscribes`, a trigger's `reads`). An
appearance's `command` is a list on the edge's source that holds the names of the edge's targets. The
automation's `command` has that direction but holds one name (`clearRelationship`, `:584-587`;
`pendingRename`, `:646-649`). No instance clears, renames or repoints an entry of a list held by the source.

**The palette is pinned in four places.** `nodePalette` in `config.js:43-50`, the hover and selection rules in
`viewer.html`, the colour constants in `internal/diagram/layout.go:86-87`, and the table in
`docs/dsl-reference.md` section 13 (`:899-912`). `TestExporterPalettePinsViewer` and
`TestExporterPaletteMatchesReference` (`internal/diagram/contract_test.go:1893-2000`, `:2420-2450`) require exactly six element
types in the viewer table and the reference table, and map the viewer type `trigger` to the box labelled `Form`
(`:1927`) (`tasks/learnings.md:373-381`).

**Messages that quote the top-level blocks.** The parser's message for an unknown top-level block lists
`model, context` (`parser.go:211-213`). `internal/parser/parser_test.go` and `internal/mcp/server_test.go` (five
times) compare the parser's output with it, and `internal/lsp/diagnostics_test.go` and
`internal/diagnostic/entry_test.go` hold a hand-written copy.

**Checked-in models with a trigger.** `examples/all_patterns.emod` (`:26`, `:109`) and `specs_hotel.emod` (`:18`);
`internal/parser/testdata/all_patterns.emod` (`:11`), which `internal/parser/integration_test.go:129-133` reads;
sixteen fixture sources in `internal/test/fixtures.go`; inline sources in the tests of `internal/cli`,
`internal/arrange`, `internal/ast`, `internal/query`, `internal/lsp`, `internal/oracle`, `internal/mcp` and
`internal/importer`; the e2e-viewer `SAMPLE` (`e2e-viewer/tests/helpers.js:13`). The CLI e2e suite validates
`internal/parser/testdata/all_patterns.emod` (`e2e/tests/validate.test.ts:11`). Some tests assert the line of a
diagnostic or a reference inside a fixture, so a sweep moves those lines (`tasks/learnings.md:1678-1683`).
About a hundred Go test literals build an `ast.Trigger` directly; the sweep does not touch them.

**Documents.** `docs/dsl-reference.md` names the trigger at `:282`, `:300-332` (the Command Pattern), `:371`,
`:410`, `:789`, `:806`, `:844`, `:858-861` (the cross-reference table and its note) and `:905` (the palette row).
`internal/oracle/oracle_test.go` validates each fenced emod block of `README.md` and `docs/dsl-reference.md`, and
`docs/dsl_reference.go` embeds the reference for the MCP server. `site/content/docs/dsl-reference.md` (`:85`,
`:95`), `site/content/docs/_index.md` (`:59`) and `site/content/docs/mcp.md` (`:327`, an `emod slices arrange`
label) are written by hand. `README.md` names the trigger at `:63` (a fence), `:229` (the event flow), `:286`
(`emod references` output) and `:531` (the viewer). `docs/architecture.md:167` names the trigger arrow. The live
proposals name the construct at `docs/proposals/ai/README.md:88`, `02-semantic-model-review.md:137`,
`04-model-from-conversation.md:119`, `06-rename-and-fix-suggestions.md:16` and `:102`, `07-model-import.md:171`,
and `docs/proposals/model-queries-proposal.md:66-67` and `:91-92` (its stories in `user-stories/model-queries.md`
are open).

**Words.** The story and the proposal call one occurrence of a screen in a slice an appearance, and the DSL
spells it `shows`. In `internal/diagram/contract_test.go` the test field `diagramBox.appearance` (`:61-63`) means
where a box sits and how it is painted, so Task 2 renames that field before the package gets screen
appearances. The code already says "screen" for the shape of a trigger box (`svgTriggerRect`, `screenBar` in
`renderer.js`), which is the same concept. The references keyword `issues` (`references.go:58`, `:261`) exists
because a trigger issued the commands of its slice with no DSL word. An appearance has the word `command`.

## Tasks

### Task 1: Accept `screen` and `shows` beside `trigger`, format them, and check their names

**Behavior:** A model can declare screens at the top level and show them from slices. The parser reads
`screen "Name" { description = "..." }` into `Model.Screens`, and
`shows { screen = Name, reads = ..., command = ... }` into `Slice.Appearances`. `reads` and `command` each take
a bare name or a list, and both are optional. `emod fmt` writes the screens after the `model` block and the
`shows` blocks first in each slice. `emod validate` reports five name faults, each at its own position.
`view/never-read` counts each `reads` entry of each `shows` block, and its message and its description name a
screen. `trigger` still parses and validates as before, so no checked-in model changes. The task also adds the
shared screen fixture that the later tasks test against.

**Acceptance Criteria:**

- [x] `screen "CheckoutPage" { description = "Where a guest reviews the cart" }` parses into one entry of
      `Model.Screens`, with its name, the position of its name, its description, its leading comments and its
      brace positions. `screen "CheckoutPage" {}` parses with no diagnostic and no description.
- [x] A slice with `shows { screen = CheckoutPage, reads = [CartView, DiscountView], command = ApplyDiscount }`
      parses into one entry of `Slice.Appearances`, with each `reads` entry and each `command` entry at its own
      position, in source order. `reads = CartView` parses as a list of one, and
      `command = [ApplyDiscount, RemoveDiscount]` as a list of two. A `shows` block with no `reads` and no
      `command` parses. A slice with three `shows` blocks holds three appearances in source order. A slice with a
      `trigger` still parses as before.
- [x] A `shows` block with no `screen` gives one diagnostic, at the `shows` keyword. An unknown attribute or a
      nested block in a `screen` or a `shows` block gives one diagnostic that names the block. A `shows` block
      with a label, and a `screen` block with no label, each give one diagnostic.
- [x] A top-level block of an unknown type gives the list of the allowed blocks, and the list names `model`,
      `screen` and `context`. The tests that compare the parser's message, and the hand-written copies of it,
      name `screen`.
- [x] A field named `screen` or `shows` in a `fields` block, and a command named `screen` or `shows`, parse with
      no diagnostic.
- [x] `emod fmt` writes each screen after the `model` block and before the first context, in declaration order,
      with its description and its leading comments. It writes the `shows` blocks of a slice first, after the
      slice's description, each with its leading comments. A `reads` or a `command` with one entry is a bare
      name, and with two or more entries it is a list. A second run of `emod fmt` changes nothing, and a parse,
      format, parse round trip gives the same screens and appearances.
- [x] `emod validate` reports each of these at its own position: the second declaration of a screen name (at its
      name); a `shows` block whose `screen` names no declared screen (at the `screen` entry); a `reads` entry that
      names no declared view (at that entry, with the message that an automation's `reads` gives); a `command`
      entry that names no declared command (at that entry, with the message that an automation's `command`
      gives); and a second `shows` block in one slice that names the same screen (at its `screen` entry). A
      `shows` block that names a view or a command that another slice or another context declares is valid.
- [x] `view/never-read` reports nothing for a view that only a `shows` block reads, in either slice home. Its
      message and its description name a screen and no trigger, and `emod lint --explain view/never-read` gives
      the new description.
- [x] A model with no screen formats, validates and lints as before: no golden and no expected constant moves,
      except the strings that quote the list of top-level blocks and the text of `view/never-read`.
- [x] `internal/test` has a shared screen fixture that declares screens and `shows` blocks in both slice homes.
      It holds a screen shown in two slices with different reads and commands, a slice with two `shows` blocks of
      two screens, a `shows` block with no `command` written between two blocks that have one, a `shows` block
      with no `reads`, a `command` list of two, a described screen, and a slice where an automation issues a
      command beside a `shows` block that issues another. It shows each screen that it declares. It has a parsing
      accessor, transcribed lists of the reads and of the commands of its appearances in declaration order across
      both homes, getters that walk both homes, and two independent twins that strip only the reads or only the
      commands. `oracle.Check` reports nothing for it.
- [x] `task test:unit` passes.

**Affected Files/Modules:**

- `internal/ast/ast.go`: `Screen`, `Appearance`, `Model.Screens`, `Slice.Appearances`.
- `internal/parser/parser.go`: the top-level case and the `default` message, the slice case, a reader for each
  block.
- `internal/parser/parser_test.go`: a group for `screen` and a group for `shows`, and the field-name leaves.
- `internal/formatter/formatter.go` and `formatter_test.go`.
- `internal/validator/validator.go` and `validator_test.go`.
- `internal/linter/linter.go`, `descriptions.go` and `linter_test.go`.
- `internal/cli/fmt_test.go`, `validate_test.go`, `lint_test.go`.
- `internal/mcp/server_test.go`, `internal/lsp/diagnostics_test.go`, `internal/diagnostic/entry_test.go`: the
  quoted list of top-level blocks.
- `internal/test/fixtures.go`, `internal/test/models.go`, `internal/oracle/oracle_test.go`: the screen fixture and
  its clean-input leaf.

**Patterns to Follow:**

- `internal/parser/parser.go:396-423`
- `internal/parser/parser.go:580-592`
- `internal/parser/parser.go:142-148`
- `internal/parser/parser.go:191-217`
- `internal/formatter/formatter.go:76-85`
- `internal/formatter/formatter.go:269-277`
- `internal/validator/validator.go:299-306`
- `internal/validator/validator.go:398-425`
- `internal/linter/linter.go:65-90`
- `internal/test/fixtures.go:1441-1600`
- `tasks/learnings.md:21-24` ("New DSL keywords must stay usable as field names")
- `tasks/learnings.md:136-139` ("A new block entry goes after `description` and ahead of nested blocks, in every
  writer")
- `tasks/learnings.md:196-199` ("A new optional field ships a six-part fixture kit")
- `tasks/learnings.md:343-346` ("A second construct gaining the same field extends the existing fixture, with
  independent twins")
- `tasks/learnings.md:131-134` ("A new shared fixture owes `internal/oracle` a zero-diagnostic subtest")
- `tasks/learnings.md:151-154` ("A slice has two homes")
- `tasks/learnings.md:41-44` ("Additive output changes owe a byte-identical receipt")
- `tasks/learnings.md:216-219` ("Assert a short keyword in a diagnostic with a `\b`-bounded `require.Regexp`")

**Testable:** Yes

**Certainty:** medium. Each part has a precedent: the trigger reader for a name or a list (`parser.go:396-423`),
the label-less `target` block (`:580-592`), the redeclared invariant (`validator.go:299-306`) and the trigger
writer (`formatter.go:76-85`). But no instance declares a construct at the top level that slices then name by a
bare reference, and no redeclaration check runs over the whole model.

**Blast radius:** high. `emod fmt` writes the author's file in place, so a writer that drops or misplaces a
screen or a `shows` block destroys the author's text, and the new diagnostics change what `emod validate`
accepts.

**Verification:** `task test:unit` passes.

**Depends on:** None

---

### Task 2: Draw each appearance with its own arrows in the SVG, ASCII, Mermaid and event flow diagrams

**Behavior:** `diagram.SliceEdges` gives an `EdgeScreenReads` edge for each `reads` entry and an
`EdgeScreenCommand` edge for each `command` entry of each appearance. A trigger keeps its two edge kinds until
Task 9. The SVG draws each appearance as a screen box in the Wireframes lane of its slice, side by side when a
slice has more than one, and joins each arrow to the box of its own slice. The command end of an appearance's
arrow joins the command as the command end of an automation's arrow does: the command that its own slice
declares, else the command of the first slice that declares it, through a copy when that slice is two or more
slices away. The ASCII diagram prints `{View} -> <<Screen>>` for each `reads` entry, `<<Screen>>` alone for an
appearance with no `reads`, and `<<Screen>> -> [Command]` for each `command` entry. Mermaid writes a `ui` line for
each appearance in its slice. The event flow takes the issuers of a command from the `command` entries, with the
caption `screen`. The trigger pictures do not change.

**Acceptance Criteria:**

- [x] In the SVG of the shared screen fixture, each `shows` block draws one box, labelled with the name of its
      screen, in the Wireframes lane, inside the column of its slice, with the header bar of a screen. A screen
      shown in two slices draws two boxes.
- [x] Each `reads` entry draws one arrow into the box of its own appearance, from the view, or from the copy of
      a view that a slice two or more slices away declares. Each `command` entry draws one arrow from the box to
      the command that it names. No arrow joins an appearance box to a command that its `command` does not name:
      a command of the same slice that no entry names, a command that an automation of the same slice issues, and
      a command of the same name in another slice all get no arrow from it, checked by the boxes at both ends of
      each arrow. An appearance with no `command` draws no arrow to a command.
- [x] A `command` entry that names a command that only a slice two or more slices away declares draws a copy of
      that command in the appearance's slice, and the arrow ends at the copy.
- [x] Two `shows` blocks in one slice draw two boxes in the Wireframes lane, with no overlap, each inside the
      slice column. The commands, views and events of the slice keep their rows.
- [x] The ASCII diagram prints, in the section of the slice that shows the screen: one `{View} -> <<Screen>>`
      line for each `reads` entry in list order, `<<Screen>>` alone for an appearance with no `reads`, and one
      `<<Screen>> -> [Command]` line for each `command` entry. A screen shown in two slices prints in both
      sections, each with the lines of its own appearance.
- [x] The Mermaid output writes one `ui` line for each appearance, in the slice that shows it, in the aggregate
      layout and in the DCB layout. A screen shown in two slices gets two lines.
- [x] The event flow gives a command one issuer for each screen whose appearance names it in `command`, captioned
      `screen`, labelled with the screen name and described with the description of the screen declaration. An
      appearance with no `command` adds no node. A command that an automation of the same slice issues has the
      automation as its only issuer.
- [x] A model with no appearance renders byte for byte as before in each format.
- [x] In `internal/diagram`, "appearance" names only an appearance of a screen: the test field
      `diagramBox.appearance` has a name of its own.
- [x] The paragraph on the event flow diagram in `README.md` (`:225-236`) names a screen among the origins, and
      `docs/architecture.md` (`:167`) names the screen arrows.
- [x] `task test:unit` passes.

**Affected Files/Modules:**

- `internal/diagram/graph.go`: the two edge kinds and the appearance loop of `SliceEdges`.
- `internal/diagram/element_copies.go`: the copied end of `EdgeScreenReads` and `EdgeScreenCommand`.
- `internal/diagram/svg.go` and `layout.go`: the appearance boxes, their row in the Wireframes lane, the column
  width, and the two arrow cases.
- `internal/diagram/ascii.go`, `mermaid.go`, `eventflow.go`.
- `internal/diagram/graph_test.go`, `svg_test.go`, `ascii_test.go`, `mermaid_test.go`, `eventflow_test.go`,
  `element_copies_test.go`, `contract_test.go`.
- `README.md`, `docs/architecture.md`.

**Patterns to Follow:**

- `internal/diagram/graph.go:73-85`
- `internal/diagram/svg.go:168-176`
- `internal/diagram/svg.go:285-303`
- `internal/diagram/element_copies.go:138-153`
- `internal/diagram/ascii.go:51-61`
- `internal/diagram/mermaid.go:78-82`
- `internal/diagram/eventflow.go:311-345`
- `internal/diagram/svg_test.go:101-130`
- `internal/diagram/svg_test.go:2187-2210`
- `internal/diagram/contract_test.go:828-870`
- `internal/diagram/contract_test.go:984-1000`
- `tasks/learnings.md:316-321` ("An arrow between two constructs is drawn by five surfaces")
- `tasks/learnings.md:1615-1620` ("A lane-diagram name is not an identity")
- `tasks/learnings.md:1664-1669` ("A criterion that says no arrow joins X to a Y of another slice covers
  same-named Ys")
- `tasks/learnings.md:333-336` ("Read an SVG arrow back as the two boxes it meets")
- `tasks/learnings.md:393-396` ("A trigger's header bar is a second rect")
- `tasks/learnings.md:1012-1017` ("fold by mechanism not by file")
- `tasks/learnings.md:1657-1662` ("A text edit anchored on a func line")

**Testable:** Yes

**Certainty:** medium. The trigger box keyed by its slice (`svg.go:168-176`, `:285-303`) and the copied command
end of an automation (`element_copies.go:138-153`) are the precedents. But no instance draws more than one box in
the Wireframes lane of one slice, and the column width counts only the middle row (`layout.go:189-191`).

**Blast radius:** low. Pictures that a person reads.

**Verification:** `task test:unit` passes. Look at the SVG of the shared screen fixture.

**Depends on:** Task 1

---

### Task 3: Answer references and traces about screens, and count their reads in the slice commands

**Behavior:** `query.KindScreen` (`screen`) replaces `query.KindTrigger`. `emod references` lists a `shows`
block under "issued by" of each command that its `command` names, with the keyword `command`, and under
"read by" of each view that its `reads` names. `emod trace` takes the kind `screen`, follows the arrows into
and out of each appearance, and refuses the kind `trigger`. `query.NameSites` gives each `reads` entry and each
`command` entry of a `shows` block, so go-to-definition and find-references in the editor work on those names.
`emod slices arrange` counts each `reads` entry as a reference, and `emod slices list` classifies a slice with a
`shows` block, a command and a flow as the command pattern. Until Task 9 a trigger reads as kind `screen`,
with the keyword `issues`.

**Acceptance Criteria:**

- [x] `emod references <file> ApplyDiscount`, in text and in JSON, lists under "issued by" one line for each
      `shows` block whose `command` names `ApplyDiscount`: the keyword `command`, the line of that entry, the
      element `screen CheckoutPage` and the slice of the `shows` block. A command that a slice declares and no
      `shows` block of the slice names lists no screen.
- [x] `emod references <file> CartView` lists under "read by" one line for each `shows` block whose `reads`
      names `CartView`, with the keyword `reads` and the line of that entry.
- [x] `emod trace <file> CheckoutPage --kind screen` reaches each command that an appearance of the screen
      names, with the label `command`, and with `--upstream` reaches each view that an appearance reads, with the
      label `reads`. Its root has the kind `screen`. `--kind trigger` returns the error that lists the supported
      kinds, and the list names `screen` and no `trigger`.
- [x] In the editor, go-to-definition on a view name in the `reads` of a `shows` block, and on a command name in
      its `command`, goes to the declaration. Find-references on that view or command lists those entries.
- [x] `emod slices arrange` counts each `reads` entry of each `shows` block as a reference from the slice that
      declares the view, labelled `<View> -> screen <Screen>`.
- [x] `emod slices list` gives the pattern `command` to a slice with a `shows` block, a command and a flow.
- [x] `query.KindTrigger` does not exist, and `query.TraceKinds` lists `screen`.
- [x] `task test:unit` passes.

**Affected Files/Modules:**

- `internal/query/references.go`, `trace.go`, `names.go` and their tests.
- `internal/cli/references_test.go`, `trace_test.go`, `slices_list_test.go`, `slices_arrange_test.go`.
- `internal/arrange/arrange.go` and `arrange_test.go`.
- `internal/ast/slice_record.go` and `slice_record_test.go`.
- `internal/lsp/definition_test.go`, `references_test.go`.

**Patterns to Follow:**

- `internal/query/references.go:257-264`
- `internal/query/references.go:309-315`
- `internal/query/trace.go:164-166`
- `internal/query/trace.go:250-270`
- `internal/query/names.go:134-138`
- `internal/arrange/arrange.go:132-136`
- `internal/ast/slice_record.go:44-49`
- `tasks/learnings.md:408-411` ("`internal/lsp` walks the model once now")
- `tasks/learnings.md:151-154` ("A slice has two homes")

**Testable:** Yes

**Certainty:** high. The `reads` of a trigger and the `command` of an automation are already references and
trace edges in the same functions (`references.go:257-315`, `trace.go:250-270`), and the appearance is one more
case of each.

**Blast radius:** high. The JSON of `emod references` and `emod trace` and the values of `--kind` are CLI output
formats that tools outside the repository read.

**Verification:** `task test:unit` passes.

**Depends on:** Task 1, Task 2

---

### Task 4: Offer, describe and paint `screen` and `shows` in the editor

**Behavior:** Completion offers `screen` at the top level and `shows` in a slice, and no `trigger`. Inside a
`shows` block it offers `screen`, `reads` and `command`, and inside a `screen` block `description`. Hover
describes `screen` and `shows`, the `reads` text names an appearance of a screen, and `trigger` has no hover. The
name of a screen declaration gets the `parameter` token, and its hover card has the kind Screen.
`internal/test/keywords.go` lists `screen` and `shows` and no `trigger`.

**Acceptance Criteria:**

- [x] Completion in an empty document and at the top level offers `model`, `screen` and `context`. Completion in
      a slice offers `shows` and no `trigger`. Completion in a `shows` block offers `screen`, `reads` and
      `command`, and in a `screen` block `description`. Completion after `reads =` in a `shows` block offers the
      declared views, as after `reads =` in an automation. Completion after `screen =` and `command =` offers no
      value.
- [x] Hover on the word `screen` and on the word `shows` gives a description of each. The hover text of `reads`
      names a `shows` block. Hover on the word `trigger` gives no hover.
- [x] Hover on the name of a screen declaration gives a card of the kind Screen with its description.
- [x] `lsp.GetSemanticTokens` gives the name of each screen declaration the `parameter` token, and
      `lsp.GetTokens` gives it the category `lsp.type.parameter`. No other name gets that token.
- [x] `internal/test/keywords.go` lists `screen` and `shows` and no `trigger`, `TestKeywordCoverage` passes, and
      the parser leaf that reads the keyword list as field names covers `screen` and `shows`.
- [x] `README.md` (`:415`, `:515-520`) and `docs/architecture.md` (`:376-378`) name screens among the constructs that the
      editor and the source panel paint.
- [x] `task test:unit` passes.

**Affected Files/Modules:**

- `internal/lsp/completer.go`: the two block arms for `screen` and `shows`, the top-level and slice lists.
- `internal/lsp/hover.go`, `semantictokens.go`, `model.go`.
- `internal/test/keywords.go`.
- `internal/lsp/completer_test.go`, `hover_test.go`, `semantictokens_test.go`, `token_test.go`,
  `server_test.go`.
- `README.md`, `docs/architecture.md`.

**Patterns to Follow:**

- `internal/lsp/keywords_test.go:18-75`
- `internal/lsp/completer.go:483-500`
- `internal/lsp/completer.go:339-345`
- `internal/lsp/hover.go:11-45`
- `internal/lsp/semantictokens.go:91-123`
- `internal/lsp/model.go:61-135`
- `tasks/learnings.md:408-411` ("`internal/lsp` walks the model once now")
- `tasks/learnings.md:413-416` ("The completer reads raw lines, not the AST, so a new block owes two arms and a
  value-slot decision")
- `tasks/learnings.md:21-24` ("New DSL keywords must stay usable as field names")

**Testable:** Yes

**Certainty:** high. US-003 changed the same lists with the same guard (`keywords_test.go:18-75`), and the
completer arms for a label-less block follow the arm for `target` (`completer.go:483-500`).

**Blast radius:** low. Editor features and the colours of the source panel.

**Verification:** `task test:unit` passes. The e2e-viewer row for the colour of a screen name comes in Task 7,
when the largest example declares a screen.

**Depends on:** Task 1

---

### Task 5: Give the screens and the appearances in the JSON and CUE exports and in the schema

**Behavior:** `emod export --format json` gives a `screens` list on the model and a `shows` list on each slice.
`emod export --format cue` gives the same lists, and `internal/cue/schema.cue` declares `#Screen` and
`#Appearance`, so `emod schema` and the MCP resource `emod://schema` give them. The `reads` and the `command` of an
appearance are lists, also for one entry. The `trigger` key stays until Task 8.

**Acceptance Criteria:**

- [x] The JSON export of the shared screen fixture has a `screens` list on the model, after `comments` and before
      `contexts`, with each screen's `name`, `description`, positions and comments, in declaration order. A
      screen that no slice shows is in the list.
- [x] Each slice has a `shows` list in the place of `trigger`, with one object for each `shows` block, in source
      order, carrying `screen`, `reads`, `command`, their position keys, its brace positions and its comments.
      `reads` and `command` are lists, also for one entry. `reads_position` and `command_position` have one
      position for each entry. Each position key holds the position of its own entry, and the keys of each new
      object are in the order of a sibling object, read from the raw bytes.
- [x] Read back from the decoded JSON across both slice homes, the reads and the commands of the appearances equal
      the transcribed lists of the fixture.
- [x] The CUE export gives `screens` and `shows` with the same content, and it passes `cue vet` against
      `#Model`. The JSON and CUE exports of the fixture describe the same model.
- [x] `internal/cue/schema.cue` declares `#Screen` and `#Appearance`, `screens` on `#Model` and `shows` on
      `#Slice`. `cue vet` rejects a `shows` entry with no `screen`, and one whose `reads` or `command` is one
      string, and each failure names the key in the cue error text. `emod schema` prints `#Screen` and
      `#Appearance`.
- [x] A model with no screen exports byte for byte as before, and the `trigger` key does not change.
- [x] `task test:unit` passes, with `cue` on the path.

**Affected Files/Modules:**

- `internal/export/json.go`: `jsonModel`, `jsonSlice`, a screen and an appearance document type and their
  converters.
- `internal/export/cue.go`: the two lists and their writers.
- `internal/cue/schema.cue`.
- `internal/export/export_test.go`, `internal/cue/embed_test.go`, `internal/cli/export_test.go`,
  `internal/cli/schema_test.go`.

**Patterns to Follow:**

- `internal/export/json.go:137-146`
- `internal/export/json.go:629-643`
- `internal/export/cue.go:113-115`
- `internal/export/cue.go:177-182`
- `internal/cue/schema.cue:15-20`
- `internal/export/export_test.go:6612-6640`
- `internal/export/export_test.go:6499-6520`
- `internal/cue/embed_test.go:180-210`
- `tasks/learnings.md:46-49` ("A new exported field must land in JSON, CUE and `schema.cue` in the same change")
- `tasks/learnings.md:126-129` ("JSON and CUE order their document keys differently")
- `tasks/learnings.md:141-144` ("The two export guards cannot see a list neither writer emits")
- `tasks/learnings.md:211-214` ("JSON key order is assertable from the raw bytes")
- `tasks/learnings.md:266-269` ("Read a decoded export document back with `objectsUnder`/`statedUnder`")
- `tasks/learnings.md:271-274` ("A serialized key spells the DSL keyword")
- `tasks/learnings.md:286-289` ("A `*_position` key needs its value pinned to its own AST position")
- `tasks/learnings.md:1685-1690` ("cue vet prints the temp folder")

**Testable:** Yes

**Certainty:** high. The trigger's document type, converter, CUE writer and closed definition
(`json.go:137-146`, `:629-643`, `cue.go:177-182`, `schema.cue:15-20`) are the shape to copy, and the model
already carries a list (`contexts`).

**Blast radius:** high. The JSON export, the CUE export and `schema.cue` are contracts that consumers outside the
repository read, through `emod export`, `emod schema` and `emod://schema`.

**Verification:** `task test:unit` passes, with `cue` on the path.

**Depends on:** Task 1

---

### Task 6: Carry screens and appearances through the diagram document, both viewer saves and the viewer

**Behavior:** The diagram document gives each screen declaration a node of type `screen` with no parent, and each
`shows` block a node of type `shows` in its slice, with `reads` and `command` lists and arrows of the types
`reads` and `shows_command`. `importer.ImportDiagram` builds the screens and the appearances from those nodes and
edges, and `importer.ApplyDiagram` writes each viewer edit of them back into the source: a moved, renamed or
deleted `shows` node, and an added, removed, repointed or renamed arrow. A save keeps each screen declaration and
declares a screen that an appearance names and no declaration holds. The viewer draws each `shows` node as a
screen, side by side in the top band of its slice, lets a user move one into a slice that already holds one, types
the arrows that a user draws, and lists the views and the commands of an appearance in the detail panel. The
exporter, the importer and the viewer change in one commit, because each reads the node type and its keys. Trigger
nodes and `trigger_command` edges stay until Task 8.

**Acceptance Criteria:**

- [x] `emod export --format diagram-json` of the shared screen fixture has one node of type `screen` with no
      parent for each declaration, in declaration order, before the first context node, with its label,
      description, position and comments. It has one node of type `shows` in its slice for each `shows` block,
      labelled with the screen name, with `reads` and `command` as lists (also for one entry), its position and
      its comments, and with no `screen` key. Two appearances of one screen in two slices are two nodes.
- [x] Each `reads` entry gives a `reads` edge from the view, or from its copy, to the `shows` node of its own
      slice. Each `command` entry gives a `shows_command` edge to the command it names, or to its copy. No
      `shows_command` edge ends at a command that no entry names, at a command that an automation of the slice
      issues, or at a command of the same name in another slice.
- [x] `importer.ImportDiagram` of that document gives each screen, also one that no slice shows, and each `shows`
      block with its reads and its commands in order. The export and import round trip of the fixture formats to
      the fixture's text, minus what a save still loses. An appearance whose screen has no `screen` node gets a
      declaration, after the declarations that the document has.
- [x] `importer.ApplyDiagram` of an unchanged document keeps each screen declaration, also one that no slice
      shows, and each `shows` block as written. Each of these edits saves as stated:
      - a `shows` node moved into a slice that already holds one is written into that slice and out of the old
        one, and its declaration does not change;
      - a `shows_command` arrow drawn from a `shows` node to a command adds that command to its `command`
        entries; a deleted one removes the entry; a repointed one replaces the entry in its place; a renamed
        command renames the entry; the other entries stay;
      - a `reads` arrow drawn onto a `shows` node adds the view, and a deleted, repointed or renamed one changes
        that entry only;
      - a renamed `shows` node names the new screen, and the save declares it when no declaration holds it;
      - a deleted `shows` node removes the block and keeps the declaration.
- [x] A `reads` edge onto a `shows` node from a node that is not a view, and a `shows_command` edge to a node that
      is not a command, add no entry.
- [x] The viewer draws each `shows` node as a screen box with a bar across its top, in the colours that a trigger
      has, in the top band of its slice. Two `shows` nodes in one slice sit side by side with no overlap, and the
      rows under them stay level with the rows of the slice beside it. The viewer draws no box for a `screen`
      node.
- [x] A drag of a `shows` node onto a slice that already holds one moves it there, and the save writes it there.
- [x] An arrow that a user draws from a `shows` box to a command gets the type `shows_command`, and one from a view
      to a `shows` box gets the type `reads`. An arrow drawn in the other direction gets neither type.
- [x] The detail panel of a `shows` node lists the views and the commands of the appearance in list order, with
      the em-dash placeholder for an empty list. The legend names the `shows` box and the `shows_command` arrow
      after a screen.
- [x] `nodePalette`, the hover and selection rules of `viewer.html`, the palette table of `docs/dsl-reference.md`
      section 13 and the Go palette pins carry `shows` with the colours of `trigger`.
- [x] The paragraph on the viewer in `README.md` (`:525-535`) says that a slice takes any number of screens.
- [x] `task test:unit`, `task test:viewer` and `task test:e2e:viewer` pass.

**Affected Files/Modules:**

- `internal/export/diagram.go`: the two node types, the `command` key as one name or a list, the two edge cases.
- `internal/export/export_test.go`: the diagram document leaves.
- `internal/importer/importer.go`: the node reader, the `screen` pass of `ImportDiagram`, the appearances of a
  slice, the folds of `reads` and `shows_command`.
- `internal/importer/diagram_edit.go`: the match of drawn nodes, `sliceFor`, `canHold`, the list on the source of
  a `shows_command` edge in the clear, rename and repoint paths, the missing declarations.
- `internal/importer/importer_test.go`, `diagram_edit_test.go`.
- `internal/frontend/static/model.js`, `layout.js`, `renderer.js`, `ui.js`, `config.js`, `legend.js`,
  `viewer.html`.
- `internal/frontend/tests/model.test.js`, `layout.test.js`, `slice-stack.test.js`, `renderer.test.js`,
  `detail-panel.test.js`, `legend.test.js`, `drag-containment.test.js`, `viewer.test.js`.
- `internal/diagram/contract_test.go` and `internal/diagram/layout.go`: the palette pins.
- `docs/dsl-reference.md` (section 13), `README.md`.

**Patterns to Follow:**

- `internal/export/diagram.go:384-400`
- `internal/export/diagram.go:487-556`
- `internal/importer/importer.go:22-46`
- `internal/importer/importer.go:409-470`
- `internal/importer/diagram_edit.go:182-191`
- `internal/importer/diagram_edit.go:568-667`
- `internal/importer/importer_test.go:844-880`
- `internal/importer/diagram_edit_test.go:542-580`
- `internal/importer/diagram_edit_test.go:2038-2080`
- `internal/export/export_test.go:2292-2340`
- `internal/frontend/static/model.js:156-174`
- `internal/frontend/static/model.js:260-275`
- `internal/frontend/static/layout.js:93-160`
- `internal/frontend/static/renderer.js:520-526`
- `internal/frontend/tests/detail-panel.test.js:174-215`
- `internal/frontend/tests/drag-containment.test.js:342-389`
- `tasks/learnings.md:256-259` ("A diagram-node key has three readers, and they must move in one commit")
- `tasks/learnings.md:201-204` ("The viewer's save path is `importer.ImportDiagram`, so a diagram-node field owes a
  read-back")
- `tasks/learnings.md:323-326` ("`EDGE_TYPE_BY_ENDS` carries only the direction the exporter writes")
- `tasks/learnings.md:328-331` ("An importer fold must re-check the endpoint node types")
- `tasks/learnings.md:296-299` ("The viewer shows a node twice")
- `tasks/learnings.md:373-381` (the palette in four places, and section 13 is machine-read)
- `tasks/learnings.md:520-523` ("A comment rides a diagram node without its position")
- `tasks/learnings.md:1650-1655` ("Two views with one name share one diagram node")
- `tasks/learnings.md:1692-1697` ("The audit cannot build the web bundle")

**Testable:** Yes

**Certainty:** low. No instance clears, renames or repoints an entry of a list that the edge's source holds:
`entriesOf` covers a list on the target (`diagram_edit.go:182-191`), and the automation's `command` on the source
holds one name, so the repoint pairing and the reserve-and-fill of a `shows_command` arrow are new design. The
viewer also has no precedent for more than one box in the top band of a slice.

**Blast radius:** high. A save from the viewer writes the author's `.emod` file from this document, and the
diagram document is the output of `emod export --format diagram-json` and of the MCP tool `emod_export`.

**Verification:** `task test:unit`, `task test:viewer` and `task test:e2e:viewer` pass. Look at the shared screen
fixture in the viewer.

**Depends on:** Task 1, Task 2

---

### Task 7: Move every checked-in model, fixture, sample and document from `trigger` to `screen` and `shows`

**Behavior:** No example, fixture, test source, sample or document presents `trigger` as syntax to write. Each
trigger becomes a screen declaration at the top level and a `shows` block in its slice, by the steps of the
proposal's Versioning section: the label loses its spaces, the description moves to the declaration, the `reads`
stays, and a `command` entry names each command that the trigger issued. The parser still accepts `trigger` in
this task, and Tasks 1 to 6 taught each surface the new shape, so each change keeps the suite green. The
expected values that restate a moved fixture move with it: labels, arrows and line numbers.

**Acceptance Criteria:**

- [x] No example, parser testdata file, fixture source in `internal/test/fixtures.go`, inline test source,
      e2e-viewer sample or text that a test types into the source panel states a `trigger` block. Only these
      still do: the trigger kit `TriggerReadsLibraryLending` and the tests that read it, the parser and CLI tests
      of the trigger form, and the tests that feed a trigger to a surface to show that it still reads one, which
      Tasks 8 and 9 replace. The Go test literals that build an `ast.Trigger` are not text and stay for Task 9.
- [x] Each migrated trigger is one screen declaration, whose label is a bare name and whose description is the
      trigger's, and one `shows` block with the trigger's `reads` and a `command` entry for each command of its
      slice that no automation and no translation of the slice issues. Two triggers with one label in two slices
      become one declaration and two `shows` blocks. Each changed `.emod` file and each e2e-viewer sample is in
      the `emod fmt` layout, and `emod fmt --check` passes on it.
- [x] The only expected values that move are those that restate the text of a changed fixture or example: its
      labels, its arrows and the line numbers below a changed line.
- [x] `examples/`, `README.md`, `docs/dsl-reference.md`, `site/content/docs/`, `e2e/` and `e2e-viewer/` contain no
      word `trigger` found with a `\b`-bounded search, which does not match `automation_trigger` or `triggers`.
- [x] `docs/dsl-reference.md` has a `screen` section under section 3. The slice skeleton, the Command Pattern
      skeleton and its bullets show `shows` with `screen`, `reads` and `command`, and say that a `shows` block with
      no `command` issues no command. The View Pattern example, the `view/never-read` bullet, the list of
      constructs that take a description (`screen`, not `shows`), its fenced example and the note on constructs
      that define no glossary term name `screen` and `shows`. The cross-reference table has a `screen "<Name>"` row
      whose reference is `shows { screen }`, the view row names `shows { reads }`, the command row names
      `shows { command }`, and the note on the spellings of `reads` names a `shows` block. Each heading text that
      a link cites stays, and the in-document links resolve.
- [x] The quick starts in `README.md` and `site/content/docs/_index.md` and the site copy of the reference use
      `screen` and `shows`. The `emod references` output quoted in `README.md` (`:280-292`) and the
      `emod slices arrange` label in `site/content/docs/mcp.md` (`:327`) are what the commands print for the
      migrated example.
- [x] The live proposals under `docs/proposals/ai/` and `docs/proposals/model-queries-proposal.md` present no
      `trigger` construct. A use of the word in English that names no construct stays.
- [x] The historical records that the Boundaries section names do not change.
- [x] The examples guard, the "documented models" leaf of `internal/oracle`, `TestApplyDiagram` over examples and
      parser testdata, and the save round trips of the shared fixtures still run over each changed file. The
      described fixture describes a screen, the guard over it and `DeclaredDescriptions` read screens, and the
      examples leaf on described kinds asks for a screen.
- [x] The e2e-viewer test of the colour of each declared name has a row for the name of a screen in the largest
      example, in the colour of `lsp.type.parameter`.
- [x] `task test`, `task test:viewer` and `task test:e2e:viewer` pass.

**Affected Files/Modules:**

- `examples/all_patterns.emod`, `examples/specs_hotel.emod`.
- `internal/parser/testdata/all_patterns.emod` and the assertions of `internal/parser/integration_test.go` that
  read it.
- `internal/test/fixtures.go`: the fixture sources, the transcribed values that restate them, `DeclaredDescriptions`
  and `DescribedHotelReservationDescriptions`.
- `internal/cli/examples_test.go`: the described kinds.
- Inline sources in `internal/cli` (`fmt_test.go`, `validate_test.go`, `references_test.go`, `trace_test.go`,
  `slices_list_test.go`, `slices_arrange_test.go`, `lint_test.go`), `internal/arrange/arrange_test.go`,
  `internal/ast/slice_record_test.go`, `internal/query` tests, `internal/lsp` tests, `internal/oracle/oracle_test.go`,
  `internal/mcp/server_test.go`, `internal/importer` tests, `internal/parser/parser_test.go` (the described-model
  guard).
- `e2e-viewer/tests/helpers.js`, `e2e-viewer/tests/source-panel.spec.js`.
- `README.md`, `docs/dsl-reference.md`, `site/content/docs/dsl-reference.md`, `site/content/docs/_index.md`,
  `site/content/docs/mcp.md`.
- `docs/proposals/ai/README.md`, `02-semantic-model-review.md`, `04-model-from-conversation.md`,
  `06-rename-and-fix-suggestions.md`, `07-model-import.md`, `docs/proposals/model-queries-proposal.md`.

**Patterns to Follow:**

- `tasks/learnings.md:1678-1683` ("A sweep that removes lines from shared fixtures moves the line numbers that tests
  assert below them")
- `tasks/learnings.md:448-451` ("A 'no expected constant moves' criterion is unsatisfiable when the task edits a
  shared fixture")
- `tasks/learnings.md:495-498` ("A retirement sweep's 'hits only in these files' criterion must whitelist the
  rejection tests")
- `tasks/learnings.md:241-244` ("`docs/dsl-reference.md` is the one keyword surface no test reaches")
- `tasks/learnings.md:485-488` ("The live proposals under `docs/proposals/ai/` are now the emod surface no test
  reaches")
- `tasks/learnings.md:490-493` ("`docs/dsl-reference.md` sub-heading anchors are cited more often than the
  numbered ones")
- `tasks/learnings.md:473-476` ("An ```emod fence is a promise that the block validates")
- `tasks/learnings.md:478-483` ("`examples/` is enumerated by its guard")
- `tasks/learnings.md:664-668` ("Repairing a model to satisfy a new check is a modelling decision")
- `tasks/learnings.md:1573-1578` ("The two all_patterns.emod fixtures start the same and arrange differently")
- `tasks/learnings.md:1692-1697` ("The audit cannot build the web bundle")
- `internal/oracle/oracle_test.go:114-130`
- `internal/importer/diagram_edit_test.go:542-580`
- `e2e-viewer/tests/source-panel.spec.js:704-720`

**Testable:** No. The task changes content and adds no behaviour. The guards that the criteria name check it.

**Certainty:** medium. The US-003 sweep (`700f70f`) over the same guards is the precedent. But that sweep removed
lines, and this one rewrites one block into two and renames labels, so the transcribed names, the drawn labels
and the arrows move as well as the line numbers.

**Blast radius:** low. The content of examples, fixtures and documents, and each guard runs over it.

**Verification:** `task test`, `task test:viewer` and `task test:e2e:viewer` pass. The `\b`-bounded search gives
only the allowed hits. `emod fmt --check` passes on each changed `.emod` file and each e2e-viewer sample.

**Depends on:** Task 1, Task 2, Task 3, Task 4, Task 5, Task 6

---

### Task 8: Remove the trigger from the exports, the schema, the diagram document, the importer and the viewer

**Behavior:** No wire format carries a trigger. The JSON and CUE exports drop the `trigger` key of a slice, and
`internal/cue/schema.cue` drops `#Trigger` and the `trigger` of `#Slice`, so `cue vet` now refuses a slice that
carries one. The diagram document drops the `trigger` node type and the `trigger_command` edge type. The importer
takes nothing from a `trigger` node or a `trigger_command` edge in an old document and gives no error. The viewer
drops its trigger handling and draws nothing for a `trigger` node. The AST still has `Slice.Trigger` in this
task, so a model whose AST holds a trigger is the input that shows each surface leaves it out. After Task 7 no
checked-in model has a trigger.

**Acceptance Criteria:**

- [x] `emod export --format json` of a model whose AST holds a trigger has no `trigger` key on its slice, and
      `emod export --format cue` has no `trigger` field.
- [x] `internal/cue/schema.cue` declares no `#Trigger` and no `trigger` on `#Slice`. `cue vet` rejects a slice that
      carries `trigger`, and the cue error text says that the field is not allowed. `emod schema` prints no
      `#Trigger`.
- [x] `emod export --format diagram-json` of a model whose AST holds a trigger has no node of the type `trigger`
      and no edge of the type `trigger_command`.
- [x] `importer.ImportDiagram` and `importer.ApplyDiagram` of a document with a `trigger` node, a `reads` edge onto
      it and a `trigger_command` edge give no error, and the formatted model has no screen and no `shows` block from
      them. Each other node keeps what it carries.
- [x] The viewer draws no box for a `trigger` node in an old document and still draws the rest. `nodePalette`,
      `edgeConfig`, `arrowClassMap`, `EDGE_TYPE_BY_ENDS`, the legend, the layout, the renderer, the detail panel
      and `viewer.html` have no trigger type and no `trigger_command`. The palette table of
      `docs/dsl-reference.md` section 13 has no Trigger row, and the Go palette pins count six element types.
- [x] No Go code in `internal/export` or `internal/importer` reads `Slice.Trigger` or builds an `ast.Trigger`, and
      no code in `internal/frontend/static` reads a `trigger` node or a `trigger_command` edge. The viewer tests
      that fed a `trigger` node feed a `shows` node, or go when a `shows` test covers the same behaviour.
- [x] `task test:unit`, `task test:viewer` and `task test:e2e:viewer` pass.

**Affected Files/Modules:**

- `internal/export/json.go`, `cue.go`, `diagram.go`, `internal/cue/schema.cue`.
- `internal/export/export_test.go`, `internal/cue/embed_test.go`, `internal/cli/export_test.go`,
  `internal/cli/schema_test.go`.
- `internal/importer/importer.go`, `diagram_edit.go`, `importer_test.go`, `diagram_edit_test.go`.
- `internal/frontend/static/model.js`, `layout.js`, `renderer.js`, `ui.js`, `config.js`, `legend.js`,
  `viewer.html`.
- `internal/frontend/tests/`: the leaves that feed a `trigger` node or a `trigger_command` edge.
- `internal/diagram/contract_test.go` and `internal/diagram/layout.go`: the palette pins.
- `docs/dsl-reference.md` (section 13).

**Patterns to Follow:**

- `internal/importer/importer_test.go:1187-1215`
- `internal/frontend/tests/viewer.test.js:1111-1130`
- `internal/cue/embed_test.go:180-203`
- `internal/cli/export_test.go:523-560`
- `tasks/learnings.md:251-254` ("A key rename owes a retired-key negative assertion on every surface that reads the
  key")
- `tasks/learnings.md:256-259` ("A diagram-node key has three readers, and they must move in one commit")
- `tasks/learnings.md:1685-1690` ("cue vet prints the temp folder")
- `tasks/learnings.md:373-381` (the palette in four places, and section 13 is machine-read)
- `tasks/learnings.md:1692-1697` ("The audit cannot build the web bundle")

**Testable:** Yes

**Certainty:** high. US-003 removed the actor node and key in the same way (`fe1d4fc`) and the actor keys from
the exports and the schema (`26dbd83`): the importer ignores the old node (`importer_test.go:1187-1215`), and the
viewer draws the rest of an old document (`viewer.test.js:1111-1130`).

**Blast radius:** high. The JSON export, the CUE export, `schema.cue` and the diagram document are contracts that
consumers outside the repository read, and a save from the viewer writes the author's file from the document.

**Verification:** `task test:unit`, `task test:viewer` and `task test:e2e:viewer` pass.

**Depends on:** Task 7

---

### Task 9: Reject `trigger` in the parser, and remove `ast.Trigger` with every reader

**Behavior:** A `trigger` block in a slice is an error at the keyword. The message says that the language retired
`trigger` and names `screen` and `shows` as the replacement. The rest of the file still parses, so the author sees
one error for each trigger. `trigger` stays a legal name. `ast.Trigger` and `Slice.Trigger` go, with the parser
function that filled them, the formatter code that wrote them, and each reader: the validator, the linter, the
diagram package (with `EdgeTriggerReads`, `EdgeTriggerCommand` and the caption `trigger`), the queries, the LSP,
`internal/arrange` and the slice records. The trigger kit and the tests that only showed the trigger path go, and
the tests that also cover a behaviour no screen test covers move to appearances. The editor, `emod validate`, the
MCP tools and the viewer show the new error through the parser.

**Acceptance Criteria:**

- [ ] A `trigger` block in a slice gives one diagnostic, at the line and column of the keyword `trigger`. The
      message names `trigger`, `screen` and `shows`, each matched with a `\b`-bounded pattern, and says that the
      language retired `trigger`. A trigger with a kind label gives the same diagnostic.
- [ ] The text after a rejected block still parses: the commands written after it are in the slice, and the
      slice, the aggregate and the context have their close positions.
- [ ] `emod validate` on a file with a `trigger` block returns an error that names the file, the line of the
      keyword and the replacement.
- [ ] A field named `trigger` and a command named `trigger` still parse with no diagnostic.
- [ ] `ast.Trigger`, `Slice.Trigger`, `EdgeTriggerReads`, `EdgeTriggerCommand`, the event flow caption `trigger`
      and the advice for kinded triggers do not exist, and each package and each test package compiles.
- [ ] The trigger kit (`TriggerReadsLibraryLending`, its accessor, its transcriptions, its getter and its twin) and
      the tests that only showed that a surface reads a trigger do not exist. A test that covered a behaviour that
      no appearance test covers is an appearance test.
- [ ] A `\b`-bounded search of the repository for `trigger` finds it only in the historical records that the
      Boundaries section names, in the files of this breakdown, in the rejection tests of this task, and in the
      tests of Task 8 that feed an old document a `trigger` node or a slice a `trigger` key.
- [ ] `task test` passes.

**Affected Files/Modules:**

- `internal/ast/ast.go`: `Trigger`, `Slice.Trigger`.
- `internal/parser/parser.go`: the slice case, `retiredTriggerKind`, `r.trigger`.
- `internal/parser/parser_test.go`: a "retired trigger" group, and the actor leaf that used a trigger.
- `internal/formatter/formatter.go`, `internal/validator/validator.go`, `internal/linter/linter.go`.
- `internal/diagram/graph.go`, `svg.go`, `ascii.go`, `mermaid.go`, `eventflow.go`, `element_copies.go`.
- `internal/query/references.go`, `trace.go`, `names.go`; `internal/lsp/model.go`; `internal/arrange/arrange.go`;
  `internal/ast/slice_record.go`.
- `internal/test/fixtures.go`, `internal/test/models.go`.
- `internal/cli/validate_test.go`: the leaf beside the retired actor, and the kinded-trigger leaf.
- The test files whose literals or sources still feed a trigger: `internal/diagram/*_test.go`,
  `internal/export/export_test.go`, `internal/linter/linter_test.go`, `internal/validator/validator_test.go`,
  `internal/formatter/formatter_test.go`, `internal/parser/parser_test.go`.

**Patterns to Follow:**

- `internal/parser/parser.go:191-217`
- `internal/parser/parser_test.go:90-122`
- `internal/cli/validate_test.go:100-140`
- `tasks/learnings.md:246-249` ("Retiring a keyword needs its own parser arm, not the fall-through, and three
  closing braces as the receipt")
- `tasks/learnings.md:1671-1676` ("A keyword retired surface by surface leaves tests that feed the AST the field,
  and the last task deletes them")
- `tasks/learnings.md:216-219` ("Assert a short keyword in a diagnostic with a `\b`-bounded `require.Regexp`")
- `tasks/learnings.md:495-498` ("A retirement sweep's 'hits only in these files' criterion must whitelist the
  rejection tests")
- `tasks/learnings.md:1657-1662` ("A text edit anchored on a func line")

**Testable:** Yes

**Certainty:** high. US-003 rejected `actor` and removed its fields in the same way (`c99d893`): a retired arm in
the dispatch that reports at the keyword (`parser.go:207-208`), its parser group (`parser_test.go:90-122`) and its
CLI leaf (`validate_test.go:100-118`). The compiler finds each reader of the removed fields.

**Blast radius:** high. This is a breaking change to the language that each author writes in: a file with
`trigger` stops parsing.

**Verification:** `task test` passes. `gopls` finds no reference to the removed type and field.

**Depends on:** Task 8

## Summary

- **Total tasks:** 9.
- **Order.** The order adds before it removes, because the change replaces one shape with another and every
  guard over the checked-in models must read the new shape before the models move. Task 1 teaches the language
  the new shape (parse, format, validate, lint) and gives the shared screen fixture that the surface tasks test
  against. Tasks 2 to 6 teach each surface the appearance while it still reads the trigger: the pictures, the
  queries, the editor, the exports, and the diagram document with both saves and the viewer. Task 6 comes after
  Task 2, because the diagram document draws its arrows from `diagram.SliceEdges`. Task 3 comes after Task 2 for
  the same reason (`emod trace`). Tasks 2, 4 and 5 have no order among themselves. Task 7 moves every model and
  document when each guard can read the result. Task 8 takes the trigger off the wire formats, which are the
  contracts that need negative assertions, while the AST still holds the trigger as test input. Task 9 rejects
  `trigger` and removes the AST type last, when the compiler finds each remaining reader.
- **Coverage of the story's criteria:**
  - A top-level `screen` with an optional `description`: Task 1, carried by Tasks 5 and 6.
  - A `shows` block with `screen`, `reads` and `command`, each a name or a list, any number per slice: Task 1.
  - `emod validate` reports a redeclared screen, an unknown screen, an unknown view and an unknown command, each at
    its own position: Task 1.
  - A `shows` block draws an arrow only to the commands that its `command` names: Task 2 (SVG, ASCII), Task 6
    (diagram document, viewer), Task 7 (each migrated model states its commands).
  - Two `shows` blocks of one screen in two slices draw two boxes with their own arrows in the SVG, the ASCII
    diagram and the Mermaid output (Task 2) and the viewer (Task 6). The detail panel lists the views and the
    commands: Task 6.
  - A `trigger` block fails at the keyword and names `screen` and `shows`: Task 9.
  - `emod fmt` (Task 1), and the JSON and CUE exports (Task 5).
  - A viewer save keeps each declaration, and a user can move an appearance into a slice that holds one: Task 6.
  - `emod references` and `emod trace` (Task 3), and the issuer in the event flow diagram (Task 2).
  - `view/never-read` counts the `reads` of each `shows` block and names a screen: Task 1.
  - The LSP offers, describes and paints: Task 4.
  - The examples, the reference and its site copy, the quick starts, the e2e suites and the cross-reference table:
    Task 7, with the prose that describes a changed surface moved in Tasks 2, 4 and 6. The last repository search
    is in Task 9.
  - No criterion is deferred.
- **Decisions beyond the story.** `emod validate` also reports a slice that shows one screen two times, because the
  drawings find the box of an appearance by its slice and its screen. The ASCII diagram prints the command arrows
  of a screen. A viewer save declares a screen that an appearance names and no declaration holds. The caller can
  overrule each of these before Task 1, Task 2 and Task 6 start.
- **Assessments.** Task 6 is `low` certainty: the diagram edit has no precedent for a list on the source end of an
  arrow. Tasks 1, 2 and 7 are `medium`, each for the variation that it names. Tasks 3, 4, 5, 8 and 9 are `high`,
  each with a precedent in the trigger, the automation or the US-003 retirement. Tasks 1, 3, 5, 6, 8 and 9 have a
  `high` blast radius: the author's file through `emod fmt`, the CLI output of the queries, the export contracts,
  the viewer save, and the language itself.

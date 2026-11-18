# Task Breakdown: Retire `actor`

## Contents

1. Task 1: Label a trigger with its name only in the SVG and ASCII diagrams, and caption it `trigger` in the event flow
2. Task 2: Remove the Actors group from `emod glossary`
3. Task 3: Stop the LSP from offering, describing and painting `actor`
4. Task 4: Remove the actor from the JSON and CUE exports and from the CUE schema
5. Task 5: Remove `actor` from every checked-in model and document
6. Task 6: Remove the actor from the diagram document, the importer and the viewer
7. Task 7: Reject `actor` in the parser, and remove it from the AST and the formatter

## Story Reference

`user-stories/screens-as-first-class.md`, US-003 "Retire `actor`" (story ID `screens-as-first-class/US-003`).
Goal G6 of `docs/proposals/screens-as-first-class-proposal.md`. The proposal deletes `actor` in phase 2
("`actor` deleted from end to end"). The stories ship it first as a separate story, so that US-004 does not
carry the actor.

Decisions for this run:

- This is a breaking change. A file with `actor` stops parsing, with an error that names the fix. No period
  exists where both forms are valid. The proposal's sections "Versioning" and "Phased Implementation" give the
  intent.
- `ast.Actor`, `Model.Actors`, `Trigger.Actor` and `Trigger.ActorPos` go from the AST, with each reader. No
  dead field stays.
- The JSON and CUE exports and `internal/cue/schema.cue` drop `actors`, `#Actor` and the `actor` of a trigger.
  The diagram document drops the actor node and the `actor` key of a trigger node, and the importer stops
  reading them. Each surface that read a key gets an assertion that the key is gone.

## Boundaries

**Out of scope:**

- From the stories file, verbatim: "A period where `trigger` and `screen` are both valid, and a new language
  version. The proposal retires `trigger` directly (Versioning section)." This story applies the same rule to
  `actor`: no period exists where `actor` is valid with a warning, and `ast.SupportedVersion` stays at 1.
- From the stories file, verbatim: "A tool that migrates a model. The migration is by hand, with the steps in
  the proposal." Step 4 of the proposal's Versioning section ("Delete each `actor` block, and each `actor =`
  line") is the migration for this story.
- The other non-goals of the stories file (layout, navigation, a mockup for each appearance, images, how
  `view`, `automation` and `translation` connect, a `title` attribute) are about screens. This story does not
  touch them.
- A check that resolves an actor, and any attribute that names the person who uses a screen. The proposal's
  "Risks and Open Questions" section keeps that for a later attribute on the screen declaration.
- The rejection of an old diagram document. A saved `.json` diagram document that still has an `actor` node or
  a trigger `actor` key opens and imports with no error. The importer ignores both.
- A reserved word. `actor` stays a legal name: a field or a construct named `actor` parses as before. Only the
  top-level block and the trigger attribute are errors.
- A key for compatibility. The JSON export does not keep an empty `actors` key.
- The historical records keep the word: `user-stories/`, `docs/proposals/completed/`,
  `docs/proposals/ai/completed/`, `docs/proposals/screens-as-first-class-proposal.md`,
  `docs/proposals/hcl-syntax-proposal.md` (implemented, it records the syntax change), `tasks/completed/` and
  `tasks/learnings.md`.
- "actors" as English for the people of a domain, where it does not name the construct: the interview topic
  `actors` in `docs/proposals/ai/04-model-from-conversation.md:36`, `:76` and `:195`.

**Deferred:**

- Screens take the places that actors leave. The top-level case in the parser and the declared constructs of
  the LSP go to US-004. The semantic token type `parameter` stays in the legend, because the proposal's LSP
  section gives it to screens in US-004. The slot in each glossary context goes to US-010.
- The hover text of `trigger` loses its actor clause only. US-004 writes the hover text of `screen` and
  `shows`.

## Codebase Context

**Every reader of the fields.** gopls finds these readers of `ast.Model.Actors`, `ast.Actor`,
`ast.Trigger.Actor` and `ast.Trigger.ActorPos` (`internal/ast/ast.go:41`, `:73-80`, `:313-314`):

- Source: `internal/parser/parser.go:207-208` (the top-level case), `:255-273` (`r.actor`), `:434-435` (the
  trigger attribute). `internal/formatter/formatter.go:26-31` (the actor loop) and `:92` (the `actor` line of a
  trigger).
- Pictures: `internal/diagram/svg.go:173-175`, `ascii.go:126-129` (and its doc comment, `:16`),
  `eventflow.go:258-262` (and the comment over the captions, `:49-51`). Mermaid reads no actor.
- Glossary: `internal/glossary/glossary.go:9`, `:20`, `:39-40`, `:55`, `:115-158`; `markdown.go:22`, `:45`,
  `:63`; the doc comments at `markdown.go:30-35` and `json.go:9-12`.
- LSP: `internal/lsp/model.go:76-77` (`declaredConstructs`, which gives the hover card **Actor**),
  `semantictokens.go:59` and `:123-125` (an actor name is painted `parameter`), `completer.go:486`
  (`keywordsFor` at the top level), `hover.go:13` and `:21`. `internal/test/keywords.go:11` is the list that
  `TestKeywordCoverage` pairs with hover and completion.
- Exports: `internal/export/json.go:32`, `:36-41`, `:150-151`, `:338`, `:355-366`, `:665-666`; `cue.go:84`,
  `:88-92`, `:188`; `internal/cue/schema.cue:19`, `:146-150`, `:156`.
- Diagram document: `internal/export/diagram.go:58` (the node's `actor` key), `:224-237` (`appendActor`),
  `:415`, `:597-599`; `internal/importer/importer.go:57`, `:92-98` (`asActor`), `:129`, `:253-254`;
  `diagram_edit.go:225-227`.
- Test support: `internal/test/fixtures.go:3759` (`DeclaredDescriptions`), `internal/cli/examples_test.go:35`
  and `:287-289`, `internal/diagram/contract_test.go:2865-2867`.

No code in the validator, the linter, `internal/query` or `internal/arrange` reads an actor. No check resolves
one.

**When the fields go, each reader stops compiling.** So the order is this. Tasks 1 to 4 and Task 6 stop each
surface from reading the actor while the AST still has it, so that a test can still give a surface an actor
and show that the surface leaves it out. Task 5 removes the actor from each checked-in model while the parser
still accepts it. Task 7 rejects `actor` and deletes the fields. At that point only the parser, the formatter
and some test literals still name them, and the compiler finds each one.

**The sweep comes before the importer.** `TestApplyDiagram` "leaves every model in the emod fmt layout under
examples and parser testdata as it was written" (`internal/importer/diagram_edit_test.go:554-575`) runs a
viewer save over each example and testdata file, and requires the same source back.
`requireSaveKeepsAllButItsLosses` (`internal/importer/importer_test.go:1305-1312`) does the same for the shared
fixtures. A save builds each trigger again from its node (`asTrigger`, `importer.go:125-133`). When the
document carries no actor, a save drops each `actor =` line. So these guards stay green only after Task 5.
The tests of Tasks 1 to 4 compare output only, so those tasks go first.

**Two save paths.** The viewer saves an edit through `importer.ApplyDiagram`
(`internal/pipeline/pipeline.go:139-169`). It replaces the contexts of the parsed model and keeps
`model.Actors`. A viewer that opens a `.json` document uses `importer.ImportDiagram`, which builds the whole
model from the document. Between Task 6 and Task 7, a viewer save of a file that still states an actor writes
its triggers with no `actor =` line and keeps its top-level `actor` blocks. After Task 7 such a file does not
parse, so the viewer cannot save it.

**Retired forms in the HCL reader.** The parser reads HCL through `hclsyntax`. `retiredTriggerKind`
(`parser.go:406-414`) is the one retired form. It reports a label that a trigger carried before, at that
label, with a message that names the replacement (`:417-419`). Its test is
`internal/cli/validate_test.go:100-120`, and `parser_test.go` has no test of it. The learning "Retiring a
keyword needs its own parser arm" is about the hand-written parser that HCL replaced, but two of its points
still apply. First, the retired word needs its own case, not the `default` arm. The `default` arm lists
"model, actor, context" (`parser.go:211-213`), and `parser_test.go:448-455` asserts that list. Second, a
receipt must show that the text after the rejected form still parses. HCL parses the whole file first, so the
receipt is that the constructs after the rejected block or attribute are in the model.

**Messages that quote the list of top-level blocks.** `internal/parser/parser_test.go:453`,
`internal/cli/glossary_test.go:389` and `internal/mcp/server_test.go:940`, `:1134`, `:1367`, `:1514`, `:1707`
quote the message of the parser for `test.Unparseable`. `internal/lsp/diagnostics_test.go:128-135` and
`internal/diagnostic/entry_test.go:27-31` hold a hand-written message with the same list.

**Triggers of one name in the diagram tests.** Two US-002 tests tell two triggers of one name apart by their
actor. `contract_test.go:907-930` labels them "Checkout Page (Guest)" and "Checkout Page (Clerk)", and the
connection helpers (`:1157-1197`) find a box by a `strings.Contains` on its label. `svg_test.go:102-128` uses
the actor labels too, but it does the real check by position with `svgArrowEnds` (`:2137-2155`). Without the
actor, both boxes have the label "Checkout Page". The diagram document test of the same case
(`internal/export/export_test.go:2378`) uses no actor.

**The viewer.** `internal/frontend/static/ui.js:175-189` (`renderActorAnnotations`) draws the bar. It is
exported at `:1269` and called from `viewer.js:140`. `viewer.js:182` and `store.js:61` keep its element, and
`viewer.html:981-1000` and `:1393` declare the element and its style. `ui.js:539` draws the Actor row of the
detail panel. No test covers the bar: `viewer.test.js:142` only has the element in its page fixture.
`detail-panel.test.js:174-245` covers the Actor row. `web/` is ignored build output.

**Documents.** `docs/dsl-reference.md` names the actor at `:39`, `:100-110`, `:321`, `:802`, `:810`, `:825`,
`:858`, `:860` and `:877` (the row of the cross-reference table). The MCP server serves this file
(`internal/mcp/server.go:134`), so an AI client reads it. `internal/oracle/oracle_test.go:114-130` checks each
fenced emod block of `README.md` and `docs/dsl-reference.md`, so their fences must lose `actor` before the
parser rejects it. The cross-reference table (section 11) is not machine-read. Section 13 is, and this story
does not touch it. `site/content/docs/dsl-reference.md` (`:39-48`, `:106`) and `site/content/docs/_index.md`
(`:56-63`) are separate files written by hand. `README.md` names the actor at `:60` and `:67` (a fence),
`:234` (the event flow caption), `:419` and `:524` (the painted constructs). `docs/architecture.md:378` lists
the painted constructs. The live proposals name the construct at
`docs/proposals/ai/02-semantic-model-review.md:136`, `04-model-from-conversation.md:97` and `:100`,
`06-rename-and-fix-suggestions.md:399-400` and `07-model-import.md:171`.

**Checked-in models.** `examples/all_patterns.emod`, `dcb_model.emod`, `specs_hotel.emod` and
`error_diagnostics_test.emod`; `internal/parser/testdata/all_patterns.emod` and `minimal.emod`, which
`internal/parser/integration_test.go:24-26`, `:102-104` and `:136` read; about twenty fixture sources in
`internal/test/fixtures.go`, with the transcribed map at `:3491-3492`; inline sources in the tests of
`internal/cli`, `internal/arrange`, `internal/query`, `internal/lsp`, `internal/oracle` and
`internal/importer`; the e2e-viewer samples (`e2e-viewer/tests/helpers.js:10-17`, `:66`), which must stay
canonical under `emod fmt --check`, and `source-panel.spec.js:811-817`; text typed into the panel in
`internal/frontend/tests/source-panel.test.js:159-162`, `:265` and `viewer.test.js:5804-6724`. The CLI e2e
suite (`e2e/`) has no actor. Some tests assert the line of a diagnostic inside a fixture (for example
`internal/oracle/oracle_test.go:133-160`), so a removed `actor` block moves those lines.

**Editors.** The repository has no grammar of its own, and `editors/vscode` has no list of keywords. The LSP
gives completion, hover and semantic tokens. The viewer colours its source panel with `lsp.GetTokens`, which
reads the same semantic tokens. `e2e-viewer/tests/source-panel.spec.js:704-712` checks that an actor name has
the colour of `lsp.type.parameter`. Nothing else paints `parameter`.

**Words.** The story and the code say "retire" for a form that the language accepted before and now rejects
with an error that names the fix (`retiredTriggerKind`, the learnings). The event flow calls the text under
the label of an origin its "caption" (`diagram.Node.Caption`, `triggerCaption`). The story's "bar of actors"
is `renderActorAnnotations` in the code. It goes, so no word for it stays.

## Tasks

### Task 1: Label a trigger with its name only in the SVG and ASCII diagrams, and caption it `trigger` in the event flow

**Behavior:** The pictures stop showing the actor. The SVG writes the name of a trigger in its box with no
actor in brackets. The ASCII diagram writes a trigger as `<<Name>>`. The event flow diagram gives each trigger
the caption `trigger`. The AST still has `Trigger.Actor` in this task, so a trigger that names an actor is the
input that shows each picture leaves it out. The two US-002 tests that tell two triggers of one name apart stop
using the actor to do it.

**Acceptance Criteria:**

- [x] `diagram.ExportSVG` labels the box of a trigger that names an actor with the name of the trigger only.
      The output has no actor in brackets.
- [x] `diagram.ExportASCII` writes a trigger that names an actor as `<<Name>>`, on its own line and after the
      arrow from each view that it reads.
- [x] `diagram.CollapseEventFlow` gives the caption `trigger` to a trigger that names an actor, and to each
      trigger of the whole-model fixture `test.EveryConstructLibraryLendingModel`.
- [x] The test "two triggers that share a name each get the arrows of their own slice" still tells the two
      boxes apart when both have the label `Checkout Page`. An arrow that goes into the box of the other slice
      makes it fail.
- [x] The SVG test that connects each trigger to the command of its own slice holds when both boxes have one
      label.
- [x] No code in `internal/diagram` reads `Trigger.Actor`. The doc comment of `ExportASCII` and the comment over
      the event flow captions name no actor. The paragraph of `README.md` on the event flow diagram (`:229-236`)
      names no actor.
- [x] `task test:unit` passes.

**Affected Files/Modules:**

- `internal/diagram/svg.go`: the label of a trigger box (`:168-178`).
- `internal/diagram/ascii.go`: `formatTrigger` (`:125-132`) and the doc comment of `ExportASCII` (`:16`).
- `internal/diagram/eventflow.go`: `triggerNode` (`:258-271`) and the comment over the captions (`:49-51`).
- `internal/diagram/svg_test.go`: the actor label test (`:53-62`) and the test of one command name in two
  slices (`:102-128`).
- `internal/diagram/ascii_test.go`: the actor label tests (`:62-86`, `:114-122`) and the full model
  expectation (`:505-510`).
- `internal/diagram/eventflow_test.go`: the caption tests (`:94-123`) and the whole-model expectation
  (`:310-325`).
- `internal/diagram/contract_test.go`: the test of two triggers of one name (`:907-930`), the labels of
  `reactiveModel` (`:1584-1588`, `:1735-1743`), `describedModel` and `withoutDescriptions` (`:2755-2790`,
  `:2860-2870`).
- `README.md`: the paragraph on the event flow diagram.

**Patterns to Follow:**

- `internal/diagram/svg_test.go:102-128`
- `internal/diagram/svg_test.go:2137-2155`
- `internal/diagram/contract_test.go:1157-1197`
- `internal/diagram/eventflow_test.go:111-123`
- `internal/diagram/ascii_test.go:88-100`
- `tasks/learnings.md:1657-1662` ("A text edit anchored on a func line puts the new function under the doc
  comment of the next one")

**Testable:** Yes

**Certainty:** medium. `svgArrowEnds` (`svg_test.go:2137-2155`) tells two boxes of one label apart by
position. But the contract test names each box by its label (`contract_test.go:1157-1197`), and no test at the
contract level tells two boxes of one label apart yet.

**Blast radius:** low. Labels and captions in pictures that a person reads. No machine reads them.

**Verification:** `task test:unit` passes. `gopls` finds no reference to `Trigger.Actor` in `internal/diagram`.

**Depends on:** None

---

### Task 2: Remove the Actors group from `emod glossary`

**Behavior:** `emod glossary` stops listing actors. The markdown has no "Actors" heading under the model or
under a context. The JSON has no `actors` key at either level. Each other group keeps its place and its terms.
The AST still has the actor in this task, so a model that declares actors is the input that shows the glossary
leaves them out.

**Acceptance Criteria:**

- [x] For a model that declares an actor that no trigger names, and an actor that a trigger names,
      `emod glossary` writes no "Actors" heading at the top level or in any context.
- [x] `emod glossary --format json` of that model has no `actors` key at the top level or in any context.
- [x] The other groups keep their order and their terms in the CLI expectations.
- [x] No code in `internal/glossary` reads `Model.Actors` or `Trigger.Actor`. The doc comments of
      `RenderMarkdown` and `RenderJSON` name no actor.
- [x] `task test:unit` passes.

**Affected Files/Modules:**

- `internal/glossary/glossary.go`: the `Actors` field of `document` and of `contextSection`, and the four
  actor functions (`:115-158`).
- `internal/glossary/markdown.go`: `actorsHeading`, the two actor groups and the doc comment of
  `RenderMarkdown`.
- `internal/glossary/json.go`: the doc comment of `RenderJSON`.
- `internal/glossary/glossary_test.go`: the actor leaves (`:175-300`) and the JSON leaf (`:467-500`).
- `internal/cli/glossary_test.go`: the markdown expectations (`:24-60`, `:113`, `:181`) and the JSON
  expectations (`:250`, `:288`, `:329`).

**Patterns to Follow:**

- `internal/glossary/markdown.go:55-66`
- `internal/glossary/glossary.go:13-21`
- `internal/glossary/glossary_test.go:467-500`
- `internal/cli/glossary_test.go:24-60`
- `tasks/learnings.md:111-114` ("`internal/glossary` collects once and renders twice; the collected document
  is the JSON shape")

**Testable:** Yes

**Certainty:** high. A term group is one field on `contextSection` and one `termGroup`
(`markdown.go:55-66`, `glossary.go:13-21`). This task removes one at each level.

**Blast radius:** high. It removes the `actors` key from the JSON that `emod glossary --format json` prints,
which is a CLI output format.

**Verification:** `task test:unit` passes.

**Depends on:** None

---

### Task 3: Stop the LSP from offering, describing and painting `actor`

**Behavior:** Completion at the top level offers `model` and `context`. Hover gives nothing for the word
`actor`, and the hover text of `trigger` names no actor. The name of an `actor` block gets no semantic token
and no hover card, so the editor and the source panel of the viewer colour it as plain HCL.
`internal/test/keywords.go` stops listing `actor`, so `TestKeywordCoverage` stops asking for a hover text and a
completion for it. The token type `parameter` stays in the legend for US-004. The parser still accepts `actor`
in this task, so a document with an actor block is the input that shows each feature leaves it out.

**Acceptance Criteria:**

- [x] Completion in an empty document, and at the top level below other blocks, offers `model` and `context`
      and no `actor`. The completion labels that the server gives hold no `actor`.
- [x] Hover on the word `actor` gives no hover. The hover text of `trigger` names no actor.
- [x] Hover on the name of an `actor` block gives no card of the kind Actor.
- [x] `lsp.GetSemanticTokens` and `lsp.GetTokens` give the name of an `actor` block no token and no
      `lsp.type.parameter` category.
- [x] `internal/test/keywords.go` lists no `actor`, and `TestKeywordCoverage` passes.
- [x] No code in `internal/lsp` reads `Model.Actors`. `README.md` (`:419`, `:524`) and `docs/architecture.md`
      (`:378`) name no actor among the constructs that the editor and the source panel paint. The e2e-viewer
      test of the colour of each declared name (`e2e-viewer/tests/source-panel.spec.js:704-712`) has no
      `actor` row.
- [x] `task test:unit` passes.

**Affected Files/Modules:**

- `internal/lsp/completer.go`: the top-level list of `keywordsFor` (`:486`).
- `internal/lsp/hover.go`: the `actor` entry and the `trigger` text of `keywordDescriptions` (`:13`, `:21`).
- `internal/lsp/semantictokens.go`: the actor loop (`:123-125`) and the doc comment of `GetSemanticTokens`
  (`:59`).
- `internal/lsp/model.go`: the actor loop of `declaredConstructs` (`:76-77`).
- `internal/test/keywords.go`: the `actor` entry (`:11`).
- `internal/lsp/completer_test.go` (`:19-33`, `:153`, `:936`, `:1305`, `:1338`), `hover_test.go` (`:189-195`,
  `:499`), `semantictokens_test.go` (`:148-157`, `:196`, `:237-276`), `token_test.go` (`:255-310`),
  `server_test.go` (`:353`, `:557`, `:1071`).
- `README.md` and `docs/architecture.md`: the lists of painted constructs.
- `e2e-viewer/tests/source-panel.spec.js`: the `lsp.type.parameter` row.

**Patterns to Follow:**

- `internal/lsp/keywords_test.go:18-75`
- `internal/lsp/completer.go:483-500`
- `internal/lsp/hover.go:11-35`
- `internal/lsp/semantictokens_test.go:148-157`
- `tasks/learnings.md:408-411` ("`internal/lsp` walks the model once now")
- `tasks/learnings.md:413-416` ("The completer reads raw lines, not the AST")
- `tasks/learnings.md:1357-1362` ("An editor colours emod with its own HCL grammar plus emod lsp's semantic
  tokens")

**Testable:** Yes

**Certainty:** high. `TestKeywordCoverage` (`keywords_test.go:18-75`) pairs the keyword list with the hover and
completion lists that are kept by hand. Each change removes one entry from a list that it already guards.

**Blast radius:** low. Editor features and the colours of the source panel. No file and no contract changes.

**Verification:** `task test:unit` passes. Where Playwright can run, `task test:e2e:viewer` passes the source
panel suite.

**Depends on:** None

---

### Task 4: Remove the actor from the JSON and CUE exports and from the CUE schema

**Behavior:** `emod export --format json` and `emod export --format cue` stop writing actors: no `actors` list
on the model, and no `actor` or `actor_position` on a trigger. `internal/cue/schema.cue` drops `#Actor`, the
`actors` of `#Model` and the `actor` of `#Trigger`. The definitions are closed, so the schema now rejects a
document that carries one of those keys. `emod schema` and the MCP resource `emod://schema` give the new
schema. The AST still has the actor in this task, so a model that declares one is the input that shows each
export leaves it out.

**Acceptance Criteria:**

- [x] `emod export --format json` of a model that declares an actor, with a trigger that names it, has no
      `actors` key, and the trigger has no `actor` and no `actor_position` key. The same holds for
      `examples/all_patterns.emod`.
- [x] `emod export --format cue` of that model has no `actors` field and no `actor` field on the trigger, and
      it passes `cue vet` against the schema.
- [x] `internal/cue/schema.cue` declares no `#Actor`, no `actors` on `#Model` and no `actor` on `#Trigger`.
      `cue vet` rejects a model document that carries `actors`, and one whose trigger carries `actor`. Each
      failure names the key.
- [x] `emod schema` prints no `#Actor`.
- [x] The schema leaf "rejects a model with no name" fails on the missing name only: its document carries no
      key that the schema rejects for another reason.
- [x] No code in `internal/export/json.go` or `internal/export/cue.go` reads `Model.Actors` or `Trigger.Actor`.
      The exports bullet of `docs/dsl-reference.md` (`:858`) names no actor.
- [x] `task test:unit` passes.

**Affected Files/Modules:**

- `internal/export/json.go`: `jsonModel.Actors`, `jsonActor`, the actor keys of `jsonTrigger`,
  `convertActors`, `convertActor` and `convertTrigger`.
- `internal/export/cue.go`: the actor list of the model, `writeActor` and the `actor` line of `writeTrigger`.
- `internal/cue/schema.cue`: `#Actor`, `#Model.actors` and `#Trigger.actor`.
- `internal/export/export_test.go`: the JSON and CUE leaves that read actors (`:36-65`, `:165-175`,
  `:500-510`, `:640-700`, `:1030-1040`, `:1125-1132`, `:1425-1445`, `:2050-2058`, `:4450-4460`, `:4536-4610`,
  `:5278-5316`, `:6935-6946`).
- `internal/cue/embed_test.go`: the nameless model (`:72-79`), `fullModelJSON` (`:240-270`) and the leaves for
  the retired keys.
- `internal/cli/export_test.go` (`:65-120`, `:529-563`) and `internal/cli/schema_test.go` (`:22`).
- `docs/dsl-reference.md`: the exports bullet (`:858`).

**Patterns to Follow:**

- `internal/cue/embed_test.go:112-121`
- `internal/export/export_test.go:5296-5303`
- `internal/cli/export_test.go:540-548`
- `tasks/learnings.md:251-254` ("A key rename owes a retired-key negative assertion on every surface that
  reads the key")

**Testable:** Yes

**Certainty:** high. The retired trigger `kind` (`export_test.go:5296-5303`, `cli/export_test.go:540-548`) and
the retired key `trigger_event` (`embed_test.go:112-121`) are the same change on the same writers and on the
same closed schema.

**Blast radius:** high. The JSON export, the CUE export and `schema.cue` are contracts that consumers outside
the repository read, through `emod export`, `emod schema` and the MCP resource `emod://schema`.

**Verification:** `task test:unit` passes, with `cue` on the path so that the schema leaves run.

**Depends on:** None

---

### Task 5: Remove `actor` from every checked-in model and document

**Behavior:** No example, fixture, test source, sample or document presents `actor` as syntax to write. The
parser still accepts `actor` in this task, so each change keeps the suite green, and when Task 7 rejects
`actor`, no checked-in model fails to parse. Tasks 1 to 4 already removed the actor from each output, so the
removal of an actor from a fixture changes no output that a test compares, except the line numbers of the lines
below it.

**Acceptance Criteria:**

- [ ] No example, parser testdata file, fixture in `internal/test/fixtures.go`, test source or e2e sample
      states an `actor` block or an `actor` attribute. Only these tests still spell one: a test that gives
      `actor` to a surface to show that the surface rejects it or shows nothing of it; the parser tests that
      accept `actor`, which Task 7 replaces; and the importer and viewer tests that carry an actor through the
      diagram document, which Task 6 replaces.
- [ ] `examples/`, `docs/dsl-reference.md`, `site/content/docs/dsl-reference.md`, `site/content/docs/_index.md`,
      `README.md`, `e2e/` and `e2e-viewer/` contain no word `actor`, found with a `\b`-bounded search, which
      does not match `reactor`.
- [ ] `docs/dsl-reference.md` has no `actor` section. The references example, the skeleton of the command
      pattern, the list of constructs that take a description, the fenced example of descriptions and the note
      on constructs that own no shape name no actor. The cross-reference table has no actor row.
- [ ] The live proposals under `docs/proposals/ai/` present no `actor` block, attribute or construct. A use of
      "actors" that means the people of a domain stays.
- [ ] The historical records that the Boundaries section names keep the word, unchanged.
- [ ] The examples guard, the "documented models" leaf of `internal/oracle` and the `TestApplyDiagram` guard
      over examples and parser testdata still run over each changed file. The examples leaf "describes at least
      one construct of every kind that accepts a description" does not ask for an actor.
- [ ] Each e2e-viewer sample stays canonical: `emod fmt --check` passes on it.
- [ ] `task test` passes.

**Affected Files/Modules:**

- `examples/all_patterns.emod`, `dcb_model.emod`, `specs_hotel.emod`, `error_diagnostics_test.emod`.
- `internal/parser/testdata/all_patterns.emod`, `minimal.emod`, and the actor assertions of
  `internal/parser/integration_test.go` that read them.
- `internal/test/fixtures.go`: the fixture sources, the transcribed map (`:3491-3492`) and the actor loop of
  `DeclaredDescriptions` (`:3759`).
- `internal/cli/examples_test.go`: the list of kinds (`:35`) and the actor loop of `describedKinds`
  (`:287-289`).
- Inline sources in `internal/cli/fmt_test.go`, `validate_test.go` (`:640-662`), `references_test.go`,
  `trace_test.go`, `slices_list_test.go`, `slices_arrange_test.go`; `internal/arrange/arrange_test.go`;
  `internal/query/references_test.go`, `trace_test.go`; `internal/lsp/definition_test.go`, `references_test.go`,
  `hover_test.go` (`:486`); `internal/oracle/oracle_test.go` (`:185-192`); the `internal/importer` tests whose
  sources state an actor without being about it.
- Line numbers that tests assert inside changed fixtures, for example in `internal/oracle/oracle_test.go`.
- `internal/frontend/tests/source-panel.test.js` and `viewer.test.js`: the text that tests type into the panel.
- `e2e-viewer/tests/helpers.js` (the samples) and `source-panel.spec.js` (`:811-817`).
- `README.md`, `docs/dsl-reference.md`, `site/content/docs/dsl-reference.md`, `site/content/docs/_index.md`.
- `docs/proposals/ai/02-semantic-model-review.md`, `04-model-from-conversation.md`,
  `06-rename-and-fix-suggestions.md`, `07-model-import.md`.

**Patterns to Follow:**

- `tasks/learnings.md:485-488` ("The live proposals under `docs/proposals/ai/` are now the emod surface no test
  reaches")
- `tasks/learnings.md:495-498` ("A retirement sweep's \"hits only in these files\" criterion must whitelist the
  rejection tests")
- `tasks/learnings.md:241-244` ("`docs/dsl-reference.md` is the one keyword surface no test reaches")
- `tasks/learnings.md:473-476` ("An ```emod fence is a promise that the block validates")
- `tasks/learnings.md:478-483` ("`examples/` is enumerated by its guard")
- `tasks/learnings.md:151-154` ("A slice has two homes")
- `internal/oracle/oracle_test.go:114-130`
- `internal/cli/validate_test.go:51-87`

**Testable:** No. The task changes content and adds no behaviour. The guards named in the criteria check it.

**Certainty:** high. The US-011 sweep of retired spellings is the precedent (`tasks/learnings.md:485-488`,
`:495-498`), over the same guards.

**Blast radius:** low. The content of examples, fixtures and documents, and each guard runs over it.

**Verification:** `task test` passes. The `\b`-bounded search gives only the allowed hits. `emod fmt --check`
passes on each e2e-viewer sample. Where Playwright can run, `task test:e2e:viewer` passes.

**Depends on:** Task 1, Task 2, Task 3, Task 4

---

### Task 6: Remove the actor from the diagram document, the importer and the viewer

**Behavior:** The diagram document stops carrying the actor: no node of the type `actor`, and no `actor` key on
a trigger node. The importer ignores both in a document that still has them, so an old saved `.json` document
opens and imports with no error, and a save from the viewer writes no actor. The viewer drops the bar of actors
above the diagram and the Actor row of the detail panel. The panel of a trigger shows its Reads only. The
exporter, the importer and the viewer change in one commit, because each reads the key and an exporter and an
importer keyed differently drop a value on the save path with no error. The AST still has the actor in this
task.

Between this task and Task 7, a save from the viewer of a file that still states an actor writes its triggers
with no `actor =` line and keeps its top-level `actor` blocks. After Task 5 no checked-in model states an
actor, and after Task 7 such a file does not parse.

**Acceptance Criteria:**

- [ ] `emod export --format diagram-json` of a model that declares an actor, with a trigger that names it, has
      no node of the type `actor`, and the trigger node has no `actor` key.
- [ ] `importer.ImportDiagram` of a document with a top-level `actor` node and a trigger node that carries an
      `actor` key gives no error, and the formatted model states no `actor`. Each other node keeps what it
      carries.
- [ ] `importer.ApplyDiagram` of an edited document whose new trigger node carries an `actor` key writes that
      trigger with its name and its reads and with no `actor` line.
- [ ] The viewer shows no bar of actors above the diagram for a document with an `actor` node, and
      `viewer.html` has no element for the bar.
- [ ] The detail panel of a trigger node shows a Reads row and no Actor row, also for a node that carries an
      `actor` key.
- [ ] No code in `internal/export/diagram.go` or `internal/importer` reads `Model.Actors` or `Trigger.Actor`,
      and no code in `internal/frontend/static` reads an `actor` node or an `actor` key.
- [ ] `task test:unit` and `task test:viewer` pass.

**Affected Files/Modules:**

- `internal/export/diagram.go`: the `actor` key of `jsonDiagramNode`, `appendActor`, the trigger node and
  `convertModelToDiagram`.
- `internal/export/export_test.go`: the diagram document leaves that read actors (`:2160-2216`, `:2275-2285`,
  `:3420-3432`).
- `internal/importer/importer.go`: the `actor` key of `diagramNode`, `asActor`, `asTrigger` and the top-level
  pass of `ImportDiagram` (`:248-256`).
- `internal/importer/diagram_edit.go`: the actor loop of `matchDrawnNodes` (`:221-230`).
- `internal/importer/importer_test.go` (`:661-705`) and `diagram_edit_test.go` (`:841-845` and the other leaves
  about the actor of a trigger).
- `internal/frontend/static/ui.js` (`renderActorAnnotations`, the Actor row, the export at `:1269`),
  `viewer.js` (`:140`, `:182`), `store.js` (`:61`), `viewer.html` (`:981-1000`, `:1393`).
- `internal/frontend/tests/detail-panel.test.js` (`:174-245`), `viewer.test.js` (`:142`), `model.test.js`
  (`:232`).

**Patterns to Follow:**

- `internal/importer/importer_test.go:681-705`
- `internal/importer/importer_test.go:744-775`
- `internal/frontend/tests/detail-panel.test.js:222-233`
- `internal/export/export_test.go:5296-5303`
- `internal/importer/diagram_edit_test.go:554-575`
- `tasks/learnings.md:256-259` ("A diagram-node key has three readers, and they must move in one commit")
- `tasks/learnings.md:251-254` ("A key rename owes a retired-key negative assertion on every surface that
  reads the key")

**Testable:** Yes

**Certainty:** medium. The retired trigger `kind` (`importer_test.go:681-705`, `detail-panel.test.js:222-233`)
is the precedent for a key that a node stops carrying. But no change removed a whole node type, which the
top-level pass of the importer (`importer.go:248-256`), the match of drawn nodes (`diagram_edit.go:221-230`)
and the bar of the viewer each handle.

**Blast radius:** high. A save from the viewer writes the author's `.emod` file from this document, and the
diagram document is the output of `emod export --format diagram-json` and of the MCP tool `emod_export`.

**Verification:** `task test:unit` and `task test:viewer` pass.

**Depends on:** Task 5

---

### Task 7: Reject `actor` in the parser, and remove it from the AST and the formatter

**Behavior:** A top-level `actor` block is an error at the keyword, and an `actor` attribute in a trigger is an
error at the attribute. Each message says that the language retired `actor` and that the author must delete
it. The rest of the file still parses, so the author sees one error for each actor and no other error from it.
The message for an unknown top-level block lists `model` and `context` only. `actor` stays a legal name.
`ast.Actor`, `Model.Actors`, `Trigger.Actor` and `Trigger.ActorPos` go, with the parser function that filled
them and the formatter code that wrote them. The test literals that still set the fields lose them, and their
assertions stay. The editor, `emod validate`, the MCP tools and the viewer show the new error through the
parser.

**Acceptance Criteria:**

- [ ] A top-level `actor` block gives one diagnostic, at the line and column of the keyword `actor`. The
      message names `actor` (matched with a `\b`-bounded pattern), says that the language retired it, and says
      to delete the block.
- [ ] An `actor` attribute in a trigger gives one diagnostic, at the line and column of the attribute name,
      with the same advice for the attribute.
- [ ] The text after a rejected block or attribute still parses: the context after a rejected `actor` block is
      in the model with its close position, and a trigger keeps its name and the `reads` written after a
      rejected attribute.
- [ ] A top-level block of an unknown type still gives the list of the allowed blocks. The list names `model`
      and `context` and does not name `actor`.
- [ ] A field named `actor` and a construct named `actor` still parse with no diagnostic.
- [ ] `emod validate` on a file with a top-level `actor` block returns an error that names the file, the line of
      the keyword and the advice.
- [ ] `ast.Actor`, `Model.Actors`, `Trigger.Actor` and `Trigger.ActorPos` do not exist, and each package and
      each test package compiles.
- [ ] The test strings that quote the list of top-level blocks name no `actor`.
- [ ] A `\b`-bounded search of the repository for `actor` finds it only in the historical records that the
      Boundaries section names, in the files of this breakdown, and in tests that give `actor` to a surface to
      show that the surface rejects it or shows nothing of it.
- [ ] `task test` passes.

**Affected Files/Modules:**

- `internal/ast/ast.go`: `Model.Actors`, `Actor`, `Trigger.Actor`, `Trigger.ActorPos`.
- `internal/parser/parser.go`: the top-level case and the list in the `default` message (`:207-213`),
  `r.actor` (`:255-273`) and the `actor` case of `r.trigger` (`:434-435`).
- `internal/parser/parser_test.go`: the "model and actor" group (`:61-96`), the comments leaf (`:433-445`), the
  faults leaf (`:448-455`) and the leaves for the retired forms.
- `internal/formatter/formatter.go`: the actor loop (`:26-31`) and the `actor` line of a trigger (`:92`).
- `internal/cli/validate_test.go`: a leaf beside the retired trigger kind (`:100-120`).
- `internal/cli/glossary_test.go` (`:389`), `internal/mcp/server_test.go` (`:940`, `:1134`, `:1367`, `:1514`,
  `:1707`), `internal/lsp/diagnostics_test.go` (`:128-135`), `internal/diagnostic/entry_test.go` (`:27-31`):
  the quoted list of top-level blocks.
- The test files whose literals still set the removed fields: `internal/diagram/svg_test.go`, `ascii_test.go`,
  `eventflow_test.go`, `mermaid_test.go`, `contract_test.go`; `internal/export/export_test.go`;
  `internal/glossary/glossary_test.go`; `internal/linter/linter_test.go` (`:3613`).

**Patterns to Follow:**

- `internal/parser/parser.go:406-419`
- `internal/cli/validate_test.go:100-120`
- `internal/parser/parser_test.go:448-455`
- `tasks/learnings.md:246-249` ("Retiring a keyword needs its own parser arm, not the fall-through, and three
  closing braces as the receipt")
- `tasks/learnings.md:216-219` ("Assert a short keyword in a diagnostic with a `\b`-bounded `require.Regexp`")
- `tasks/learnings.md:1657-1662` ("A text edit anchored on a func line puts the new function under the doc
  comment of the next one")

**Testable:** Yes

**Certainty:** medium. `retiredTriggerKind` (`parser.go:406-419`) reports a retired label with a message that
names the fix. But no instance retires a top-level block type or an attribute, and `parser_test.go` has no
test of a retired form.

**Blast radius:** high. This is a breaking change to the language that each author writes in: a file with
`actor` stops parsing.

**Verification:** `task test` passes. `gopls` finds no reference to the removed fields.

**Depends on:** Task 1, Task 2, Task 3, Task 4, Task 5, Task 6

## Summary

- **Total tasks:** 7.
- **Order.** Tasks 1 to 4 stop the pictures, the glossary, the LSP and the exports from reading the actor while
  the AST still has it. Each one can then show with an actor in its input that it leaves the actor out, and
  each commit compiles. Task 5 removes `actor` from each checked-in model and document while the parser still
  accepts it, so that no commit leaves a model that fails to parse. Task 6 changes the diagram document, the
  importer and the viewer after the sweep, because the round-trip guards of the importer require each example
  and fixture to come back unchanged from a save. Task 7 rejects `actor` and deletes the fields last, when only
  the parser, the formatter and some test literals still name them. Tasks 1 to 4 have no order among
  themselves.
- **Coverage of the story's criteria:**
  - A top-level `actor` block fails at the keyword with the advice to delete it: Task 7.
  - An `actor` attribute in a trigger fails at the attribute with the same advice: Task 7.
  - The SVG and ASCII labels and the event flow caption: Task 1.
  - No bar of actors and no actor row in the viewer: Task 6.
  - No "Actors" group in `emod glossary`: Task 2.
  - No actor in the JSON and CUE exports (Task 4), the diagram document and a save from the viewer (Task 6),
    and the completion, hover and highlights of the LSP (Task 3).
  - No `actor` in the examples, the reference and its site copy, the README and the e2e suites, and no actor
    row in the cross-reference table: Task 5, with the prose that describes a changed surface moved in Tasks 1,
    3 and 4. The last search of the repository is in Task 7.
  - No criterion is deferred.
- **Assessments.** No task is `low` certainty. The story removes code and content, and each task has a
  precedent in a retirement that the repository did before: the trigger `kind`, the key `trigger_event` and the
  US-011 sweep. Tasks 1, 6 and 7 are `medium`, each for the variation that it names. Tasks 2, 4, 6 and 7 have a
  `high` blast radius, because each changes a contract that is read outside the repository: the glossary JSON,
  the exports and the schema, the diagram document and the save path, and the language itself.

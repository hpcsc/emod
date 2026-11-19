# US-007: Give a screen the path of its mockup

## Contents
1. Task 1: Accept `image` in a screen declaration and keep it through `emod fmt`
2. Task 2: Report `screen/missing-image-asset` from `emod lint` and `emod validate`
3. Task 3: Give `image` in the JSON and CUE exports and the schema, and draw every picture as before
4. Task 4: Carry `image` through the diagram document and both viewer saves
5. Task 5: Show the image path of its screen in the detail panel of an appearance

## Story Reference

US-007 in `user-stories/screens-as-first-class.md` (story ID `screens-as-first-class/US-007`), goal G4. The design
is in `docs/proposals/screens-as-first-class-proposal.md`: DSL Surface section 1 (`image` is a path relative to the
`.emod` file, the parser does not resolve it, and a path that leads to no file is a lint warning), section 4 (the
rule `screen/missing-image-asset`), Internal Representation (`Image` and `ImagePos` on `ast.Screen`, the linter, and
the risk "`screen/missing-image-asset` needs the location of the file"), and Web viewer (the detail panel of an
appearance shows "the image of the screen"). US-001 to US-006 are on main.

**Description:** As a model author, I want to give a screen the path of its mockup one time so that each appearance
of the page refers to the same design.

**Acceptance Criteria (story):**
1. A screen declaration accepts `image = "./wireframes/checkout.png"`, a path relative to the `.emod` file.
   `emod fmt`, the JSON and CUE exports, and a save from the viewer keep it.
2. A model with an `image` path that leads to no file still parses, and its exports and diagrams work.
3. `emod lint` reports `screen/missing-image-asset` at the `image` attribute when the path leads to no file. It
   reports nothing when the file exists.
4. Where no file system is present, as in the web viewer, the rule does not run and reports nothing.
5. `emod lint --explain screen/missing-image-asset` gives a description, and the reference lists the rule.
6. The detail panel of an appearance in the viewer shows the `image` path of its screen.
7. `docs/dsl-reference.md` documents `image`.

Decisions for this run:

- **The severity is warning**, as the proposal states. Every lint diagnostic makes `emod validate` exit 1
  (`tasks/learnings.md:428-432`), so **a model whose mockup is missing fails `emod validate`**. The same model
  still parses, `emod export` and `emod diagram` exit 0 for it, and the viewer and the desktop app report nothing
  for it. No checked-in model, example, parser testdata file or fenced ` ```emod ` block of the documents gains
  `image`, and the repository adds no mockup file, so the repository's own models stay valid.
- **Only the two commands that check a file run the rule.** `linter.Lint` keeps its contract: it reads the model
  and nothing else. A new `linter.LintFile` takes the model and the path of the `.emod` file, and gives every
  diagnostic of `Lint` and the diagnostics of this rule, in position order. A new `oracle.CheckFile` is
  `oracle.Check` with `LintFile` in place of `Lint`. `emod lint` and `emod validate` call it. Every other caller
  keeps `oracle.Check`, `Run`, `RunParsed` or `Parse`, so the rule does not run there. No flag and no build tag
  decide this: a caller runs the rule only when it gives the path. The table in Codebase Context gives each
  caller and the reason.
- **How a path resolves.** A relative path resolves against the directory of the path that the command was given,
  not against the working directory. An absolute path is checked as it stands. A path that names a directory is
  reported, because a directory is not a mockup. A symbolic link is followed: a link to a file reports nothing,
  and a link to nothing is reported. `image = ""` is the same as no `image`: the rule reports nothing, and
  `emod fmt` drops it, as it drops an empty `description`. No command reads a model from stdin (`readSourceFile`,
  `internal/cli/parse.go:13-31`, needs a path, and `emod lint` and `emod validate` require exactly one file
  argument), so the case does not occur. The rule only asks whether a file is there. It never opens the file.
- **`os.Root` does not apply.** The learning "Read a caller's path under a root through os.Root"
  (`tasks/learnings.md:1491-1497`) governs a tool that reads or writes a path for a caller under a root that the
  caller must not see past, such as the MCP server under `--root`. `emod lint` runs with the author's own rights,
  over the author's own model, and asks only whether a file exists. A path with `..` that leaves the model's
  directory is a normal layout (`../wireframes/checkout.png`). So the rule has no root to contain it, and it uses
  no `os.Root`. The containment question goes with the MCP, which does not run the rule (Deferred).
- **The image reaches the appearance by a copy, as the description does.** The viewer draws each `shows` node and
  never the `screen` node (`tasks/learnings.md:1701-1707`). So the diagram document gives the `screen` node its
  `image`, and copies the `image` of the declaration onto each `shows` node of that screen. A save from the viewer
  takes `image` from the `screen` node only. The copy on a `shows` node never reaches the source: `ast.Appearance`
  has no image, and a screen that a save declares for a renamed appearance gets no image.
- **The serialized keys spell the keyword** (`tasks/learnings.md:271-275`): `image` in the JSON export, the CUE
  export, `#Screen` and both node types of the diagram document, and `image_position` in the JSON export, as an
  automation has `every` and `every_position`.

## Boundaries

**Out of scope:**

The non-goals of the feature, as `user-stories/screens-as-first-class.md` states them:

- Layout, style, or the structure of components on a screen. A screen names its mockup by its path only.
- Navigation between screens. A screen graph is a different model.
- A mockup for each appearance. A second mockup means a second screen.
- Images inside `.emod` files.
- Changes to how `view`, `automation` or `translation` connect.
- A period where `trigger` and `screen` are both valid, and a new language version. The proposal retires `trigger`
  directly (Versioning section).
- A `title` attribute that restores a human label with spaces.
- A tool that migrates a model. The migration is by hand, with the steps in the proposal.

The boundaries that this breakdown adds:

- The mockup itself in the viewer or the desktop app. The detail panel shows the path as text: no picture, no
  link, no preview. In a browser, the viewer cannot read a file next to the model (the story's open question).
- The image anywhere but the detail panel of an appearance: the box on the canvas, its tooltip, the SVG, ASCII,
  Mermaid and event flow pictures, the glossary, and the LSP hover card of a screen name show no image.
- A check on the form of the path. The parser accepts any quoted string. No rule says that the path must be
  relative, must stay under the model's directory, or must name a picture. A URL is a path that leads to no file,
  and the rule reports it.
- A check on the file itself. The rule does not open the file, and does not look at its extension or its
  contents.
- An `image` in any checked-in model, and a mockup file in the repository. `examples/all_patterns.emod` does not
  show `image`, because a missing mockup fails `emod validate` for the example, and the leaf that compares the
  viewer's diagnostics with those of `emod validate` for each example (`internal/pipeline/pipeline_test.go:205-222`)
  holds only while no example has a missing mockup.
- `site/content/docs/dsl-reference.md`. It is a short overview that lists no attribute of a screen, and the story
  names only `docs/dsl-reference.md`.
- A configurable severity, or a way to turn the rule off. The rule reports a warning, as every rule of the linter
  does without configuration.

**Deferred:**

- The LSP does not run the rule. It lints the editor's buffer by its URI (`internal/lsp/server.go:388-410`),
  `internal/lsp` has no conversion from a URI to a path, and a buffer can be untitled or unsaved. A later story can
  run the rule for a `file:` document. Until then an editor shows no `screen/missing-image-asset` that
  `emod lint` reports.
- The MCP tools do not run the rule. They read a path under `--root` through `os.Root` (`internal/mcp/root.go`), and
  a check of an image path outside the root tells the caller which files exist beyond it. A later story can run
  the rule under the root, with the cases that `tasks/learnings.md:1491-1497` lists. Until then, for a model whose
  mockup is missing, `emod_check` gives one diagnostic fewer than `emod validate --format json`, which its
  description calls "the same records" (`internal/mcp/server.go:82-92`).
- A lower severity or a separate exit status for `emod validate`, if authors need `emod validate` to pass before
  their mockups exist. This breakdown records the consequence and does not change it.

## Codebase Context

**The screen today.** `ast.Screen` (`internal/ast/ast.go:173-181`) has a name, a description, comments and the
positions of its braces. `hclReader.screen` (`internal/parser/parser.go:390-410`) reads `description` with
`r.text` (`:104-112`), which records the position of the value, and rejects any other attribute as
"unexpected … in screen". The leaf "an unknown attribute and a nested block each give one diagnostic that names the
screen" (`internal/parser/parser_test.go:127-131`) uses `image` as its unknown attribute, so it must pick another
word. `hclReader.appearance` (`:412-441`) rejects an unknown attribute of a `shows` block in the same way. The
formatter writes a screen as one block with its description (`internal/formatter/formatter.go:26-30`), and
`writeText` (`:262-267`) writes nothing for an empty value.

**The precedent for an optional quoted value on a construct** is the wire type of an event (`type`) and the cadence
of an automation (`every`), end to end: the reader (`parser.go:489-492`, `:578`), the JSON key order (`jsonAutomation`,
`internal/export/json.go:170-189`: each `…_position` after `position`, each value after `comments`), the CUE writer's
`lineIfSet` (`internal/export/cue.go:237`), the schema (`internal/cue/schema.cue:70`), the six-part fixture kit
(`WireTypeLibraryLending`, `internal/test/fixtures.go:2378`, `:3712-3719`, `:3847-3863`, `:4110-4131`;
`internal/test/models.go:96-100`; `internal/oracle/oracle_test.go:111-114`), and the export tests that read the
kit back (`internal/export/export_test.go:4941-4974`). Screens are declared at the top level only, so the twin
copies `model.Screens` with `editedCopies` (`fixtures.go:3888-3905`) and needs no `copyWithEditedSlices`, and the
`Declared…` getter walks `model.Screens`, not `declaredSlices`.

**The linter and `--explain`.** `linter.Lint` (`internal/linter/linter.go:50-158`) takes only the model, and sorts
its diagnostics with `byPosition` (`:160-169`). Each rule is a free function that calls `info`, `warning` or
`errorEntry` with a literal rule name; `checkWireTypeFormat` (`:694-707`) reports one text whatever part of the
value failed. `descriptions_test.go:47-54` parses every non-test `.go` file of the package, so a rule in a new file
is read too (`tasks/learnings.md`, "A guard that parses Go source reads the whole package"), and it requires that
the set of reported rule names and the set of described rule names are the same. `emod lint --explain` is
`cli.RunLintExplain` (`internal/cli/lint.go:80-91`), which reads `linter.RuleDescription` over `ruleDescriptions`
(`internal/linter/descriptions.go:32-61`). `RuleName` is what prints the `[rule]` bracket (`tasks/learnings.md:146-150`).
The MCP tool `emod_explain_rule` and the `rules` resource read the same map (`internal/mcp/server.go:497`), so they
list the rule with no change. `internal/cli/lint_test.go:1932-1958` is a hand-written list of every rule name.

**The reference.** `docs/dsl-reference.md` has no table of rules. A rule is a bullet in the section that owns its
construct, in one shape: a bold lead, the severity, the condition, and a pointer to `emod lint --explain`
(`wire/type-format` at `:735`, `view/never-read` at `:430`). The `### screen` section is `:100-112`; its syntax block
is an untagged fence, and its prose says that the body carries a description and nothing else. The "documented
models" leaves (`internal/oracle/oracle_test.go`) run `oracle.Check` over each ` ```emod ` fence of the file, and
the headings embed their numbers (`tasks/learnings.md:31-35`).

**Who calls the linter, and who runs the rule.** `oracle.Run`, `RunParsed` and `Check`
(`internal/oracle/oracle.go:19-42`) run lexer, parser, validator and `linter.Lint`, and promise no I/O.

| Caller | File system and the path of the file | Runs the rule | Why |
|---|---|---|---|
| `emod lint` (`internal/cli/lint.go:107`) | yes | yes | Story criterion 3. |
| `emod validate` (`internal/cli/validate.go:27`) | yes | yes | It is the same check as `emod lint` (`tasks/learnings.md:428-432`); the decision above records the consequence. |
| `emod export`, `emod diagram`, `emod diagram --serve`, `emod fmt`, `emod slices arrange` (`internal/cli/export.go:33`, `diagram.go:46`, `:171`, `fmt.go:23`, `slices_arrange.go:24`) | yes | no | They produce output. `emod export` exits 1 for any warning (`exitCodeForDiagnostics`, `export.go:45-57`), and story criterion 2 says that the exports and the diagrams of the model work. |
| LSP diagnostics (`internal/lsp/server.go:388-410`) | the editor's buffer, by URI | no | Deferred: no URI-to-path conversion, and the buffer can be untitled. |
| MCP tools (`internal/mcp/server.go:290`, `:334`, `:376`) | under `--root`, through `os.Root` | no | Deferred: a check outside the root tells the caller which files exist there. |
| `internal/llm/repair` (`repair.go:91`) | no; a generated draft | no | There is no file. |
| Web viewer: WASM (`cmd/emod-wasm/main.go`) → `internal/pipeline` (`pipeline.go:51-66`) | no | no | Story criterion 4. |
| Desktop app: `internal/desktop/service.go` → `internal/pipeline` | yes, but the frontend gives the pipeline the name of the file (`store.currentFile.name`, `internal/frontend/static/viewer.js:506`), not its path | no | Both runtimes bind one pipeline and answer one contract (`service.go:10-14`), so the desktop app reports what the web viewer reports for the same source. |

**Exit codes.** `emod lint` and `emod validate` in text format exit 1 for any diagnostic; in JSON format they exit 1
for warnings and 2 when an error is present (`internal/cli/lint.go:41-78`). The leaves "returns no error for each
parser fixture the repository ships as valid" and the examples group (`internal/cli/validate_test.go:37-60`) run
`emod validate` over each checked-in model, so they are the receipt that no checked-in model gains a missing
mockup. `writeTemp` (`internal/cli/validate_test.go:1316-1323`) writes a model into its own temporary directory.

**Keywords.** `test.Keywords()` (`internal/test/keywords.go`) lists every block type, attribute name and call.
`TestKeywordCoverage` (`internal/lsp/keywords_test.go:18-100`) requires a hover text for each
(`internal/lsp/hover.go:11-50`), and requires each block's completion to offer only listed words
(`keywordsFor`, `internal/lsp/completer.go:515-516` for a screen; `completer_test.go:113`).

**The diagram document and the two saves.** `diagramBuilder` copies each screen's description onto each of its
`shows` nodes through `screenDescriptions` (`internal/export/diagram.go:187-189`, `:228-238`, `:403-422`), and the
leaf `internal/export/export_test.go:2190-2219` holds it. The viewer saves in two ways. Its export to `.emod`
(`pipeline.ExportEmod`, `internal/pipeline/pipeline.go:77-94`) runs `importer.ImportDiagram`
(`internal/importer/importer.go:249-273`), which builds each declaration from a `screen` node through `asScreen`
(`:127-133`) and declares each screen that a `shows` node names and no declaration holds, with a name only
(`declareShownScreens`, `:276-291`). `asAppearance` (`:135-142`) reads nothing that belongs to the screen. Its edit
in place (`pipeline.ApplyDiagramJSON`, `:118-174`) runs `importer.ApplyDiagram`, whose `applyTo`
(`internal/importer/diagram_edit.go:113-142`) replaces only `model.Contexts`, so each parsed declaration, and its
image, stays as it was; a renamed appearance takes the new name in `appearanceFor` (`:456-463`), and
`declareShownScreens` declares the new screen. The read-back receipts are the round trip of a canonical source
(`importFrom`, `internal/importer/importer_test.go:30-36`), `importExported` (`:37`) over a shared fixture with its
`Declared…` getter and its twin (`:577-592`), and `TestApplyDiagram` (`internal/importer/diagram_edit_test.go:589`).
A diagram-node key has three readers: the exporter, the importer and `ui.js` (`tasks/learnings.md`, "A diagram-node
key has three readers").

**The detail panel.** For a `shows` node, `showDetailPanel` (`internal/frontend/static/ui.js:520-526`) draws a
section "Screen" with the rows Reads and Commands, each the joined list or an em-dash. The leaves in
`internal/frontend/tests/detail-panel.test.js:168-217` compare the whole list of rows, and `:155-165` is the
precedent for a key that the exporter never writes. The desktop app serves the same `ui.js`.

**Words.** `image` is the word of the DSL and of every key. The proposal rejects `wireframe` (the label of the lane,
and a word for the screen itself), `mockup` and `sketch` as attribute names. No Go identifier in the repository
uses `image` today (the word appears only in the desktop packaging tests, about icons). The rule name
`screen/missing-image-asset` is fixed by the proposal and the story. The `.emod` file that a command read is the
"file" (`LintFile`, `CheckFile`); `filename` stays the name that diagnostics report, `source` the text of the model,
and `document` the LSP's buffer.

---

## Tasks

### Task 1: Accept `image` in a screen declaration and keep it through `emod fmt`

**Behavior:** A screen declaration states the path of its mockup with `image`. The parser keeps the path as the
author wrote it, with its position, and does not resolve it, so a model whose mockup does not exist yet parses.
`emod fmt` writes `image` after `description`. The editor knows the word, and the reference documents it.

**Acceptance Criteria:**
- [x] A screen with `description` and `image`, and a screen with `image` only, parse with no diagnostic. Each
      screen holds its path exactly as written and the position of the value. A screen with no `image` holds none.
- [x] A model whose `image` paths lead to no file parses with no diagnostic, and `oracle.Check` of it reports
      nothing (story criterion 2).
- [x] An `image` whose value is not a quoted string is one error at the value, with the message that `description`
      gives for the same mistake.
- [x] An `image` in a `shows` block is one error at the attribute, as each unknown attribute of a `shows` block is:
      an appearance has no image of its own.
- [x] The leaf at `internal/parser/parser_test.go:127-131` names another unknown attribute in place of `image`, and
      still gets one diagnostic for the attribute and one for the nested block.
- [x] `formatter.Format` writes `image` in the screen block after `description`, and a second format changes
      nothing. Formatting the twin of the featured fixture gives the text of formatting the featured fixture with
      each `image` line removed, so a model with no image formats as before. Each existing formatter expectation
      passes unchanged.
- [x] The fixture kit exists, in the six parts of `tasks/learnings.md:196-200`: the featured fixture
      `ScreenImageLibraryLending`, which declares a screen with no `image` between two screens that have one, a
      screen with `image` and no `description`, a path with `./` and a path with `..`, and shows each screen it
      declares; `ScreenImageLibraryLendingModel(t)`; the transcribed `ScreenImageLibraryLendingImages` in
      declaration order; the twin `WithoutScreenImages`, which leaves the model it is given unchanged;
      `DeclaredScreenImages`; and a leaf in `internal/oracle` where `oracle.Check` of the fixture reports nothing.
      No path of the fixture names a file in the repository.
- [x] The round-trip group of `internal/formatter/formatter_test.go` reads the featured fixture back unchanged, and
      `DeclaredScreenImages` of the parsed fixture equals the transcribed list.
- [x] `test.Keywords()` lists `image`. A hover on `image` describes it. The completion in a screen body offers
      `description` and `image`, in the order that `emod fmt` writes them; the completion in a `shows` body does
      not offer `image`.
- [x] The `### screen` section of `docs/dsl-reference.md` documents `image` (story criterion 7): its syntax block
      shows the optional `image`, and its prose says that the path is relative to the `.emod` file, that the parser
      does not resolve it so a model parses before its mockup exists, that a screen has one image and an appearance
      has none, and that `emod fmt` writes it after `description`. The statement that the body carries a
      description and nothing else is gone. No `## <n>.` heading is added or renumbered, the count of ` ```emod `
      fences does not change, and no fence states `image`.

**Affected Files/Modules:**
- `internal/ast/ast.go` — `Screen` gains the image and its position
- `internal/parser/parser.go` — `hclReader.screen` reads `image`
- `internal/parser/parser_test.go` — the screen group, and the leaf at `:127-131`
- `internal/formatter/formatter.go`, `internal/formatter/formatter_test.go` — the screen block, the round-trip group
- `internal/test/fixtures.go`, `internal/test/models.go`, `internal/test/keywords.go` — the kit and the keyword
- `internal/oracle/oracle_test.go` — the clean-input leaf for the featured fixture
- `internal/lsp/hover.go`, `internal/lsp/completer.go`, `internal/lsp/completer_test.go` — the hover text and the
  screen body's completion
- `docs/dsl-reference.md` — the `### screen` section

**Patterns to Follow:**
- `internal/parser/parser.go:390-410`, `:104-112`, `:489-492`
- `internal/parser/parser_test.go:92-131`
- `internal/formatter/formatter.go:26-30`, `:262-267`
- `internal/formatter/formatter_test.go:20-60`
- `internal/test/fixtures.go:2378`, `:3712-3719`, `:3847-3863`, `:3888-3905`, `:4110-4131`
- `internal/test/models.go:96-100`
- `internal/oracle/oracle_test.go:111-114`
- `internal/lsp/hover.go:11-50`, `internal/lsp/completer.go:515-518`, `internal/lsp/completer_test.go:87-123`
- `docs/dsl-reference.md:100-112`
- `tasks/learnings.md:136-140` (a new entry goes after `description`), `:196-200` (the fixture kit), `:41-45` (the
  byte-identical receipt), `:21-30` (keywords stay usable as field names), `:31-35` (headings embed their numbers),
  and the learnings "Exercise an omitted optional part mid-block", "A `Declared…` getter answers `nil`", "A new
  shared fixture owes `internal/oracle` a zero-diagnostic subtest"

**Testable:** Yes

**Certainty:** high — the screen's own `description` reader and writer, and an event's optional `type`, are the
precedent for each part, and the wire-type kit is the precedent for the fixtures.

**Blast radius:** high — `emod fmt` writes the author's file in place, so a writer that drops or misplaces `image`
destroys the author's text.

**Verification:** `mise exec -- task test:unit` and `mise exec -- task test:integration` pass.

**Depends on:** None

---

### Task 2: Report `screen/missing-image-asset` from `emod lint` and `emod validate`

**Behavior:** `emod lint` and `emod validate` warn at the `image` of each screen whose path, resolved against the
directory of the `.emod` file, leads to no file. Every other surface, among them the web viewer and the desktop
app, does not run the rule. `emod lint --explain` describes the rule, and the reference lists it.

**Acceptance Criteria:**
- [x] `emod lint` of a model whose `image` leads to no file reports one `screen/missing-image-asset` diagnostic at
      warning severity, at the line of the `image` attribute, with a message that names the screen and the path as
      written. The text format prints it with the `[screen/missing-image-asset]` bracket and exits 1; the JSON
      format gives the rule, the severity, the file and the line, and exits 1.
- [x] The same model, with a file at that path, reports nothing; the file is the only difference between the two
      runs.
- [x] A relative path resolves against the directory of the `.emod` file, not the working directory: a model whose
      path leads to a file only from the working directory is reported, and a model whose path leads to a file
      from its own directory reports nothing while the working directory is elsewhere. A path that goes up with
      `..` to an existing file reports nothing.
- [x] An absolute path is checked as it stands: one that names an existing file reports nothing, and one that names
      nothing is reported.
- [x] A path that names a directory is reported. A symbolic link to a file reports nothing, and a link to nothing
      is reported.
- [x] Each screen is checked on its own: a model with a screen whose file is missing, a screen whose file exists
      and a screen with no `image` reports exactly one diagnostic. `image = ""` reports nothing.
- [x] `linter.LintFile` gives each diagnostic that `linter.Lint` gives and the diagnostics of this rule, in position
      order; when every image exists, it gives what `linter.Lint` gives.
- [x] `emod validate` reports the same warning for the same model and exits 1. `emod validate` still exits 0 for
      each model that the repository ships as valid (`internal/cli/validate_test.go:37-60`), and no checked-in
      `.emod` file or ` ```emod ` fence states `image`.
- [x] No other surface runs the rule (story criterion 4): for a model whose `image` leads to no file, `oracle.Check`,
      `oracle.Run`, `oracle.RunParsed` and `linter.Lint` report nothing, `pipeline.RunPipelineExportDiagram` (the
      entry point of the web viewer and of the desktop app) answers no diagnostic, and `emod export` in each
      format and `emod diagram` exit 0 and print no diagnostic (story criterion 2).
- [x] `emod lint --explain screen/missing-image-asset` prints a description that says what the rule checks, that
      the path is relative to the `.emod` file, that only `emod lint` and `emod validate` run the rule and the
      viewer, the desktop app, the editor and the MCP tools do not, that the warning makes `emod validate` exit 1,
      and that the model still parses, exports and draws. `linter.Rules()` lists the rule, the guards in
      `internal/linter/descriptions_test.go` pass, and the list at `internal/cli/lint_test.go:1932-1958` names it.
- [x] The model fixture of the CLI leaves trips this rule and no other, says so in its comment, and its leaves
      assert exactly one entry.
- [x] The `### screen` section of `docs/dsl-reference.md` carries a bullet for the rule, in the shape of the bullet
      at `:735`: warning severity, the condition, that only `emod lint` and `emod validate` run it, that the warning
      makes `emod validate` exit 1, and the `emod lint --explain` invocation, which succeeds as printed. No
      `## <n>.` heading is added or renumbered, and no ` ```emod ` fence is added or edited.

**Affected Files/Modules:**
- `internal/linter/linter.go` (or a new file of the package) — `LintFile` and the rule
- `internal/linter/descriptions.go` — the description
- `internal/linter/linter_test.go` — the rule's leaves, over temporary directories
- `internal/oracle/oracle.go`, `internal/oracle/oracle_test.go` — `CheckFile`, and the leaves that the in-memory
  chain does not run the rule
- `internal/cli/lint.go`, `internal/cli/validate.go` — call `oracle.CheckFile`
- `internal/cli/lint_test.go`, `internal/cli/validate_test.go`, `internal/cli/export_test.go`,
  `internal/cli/diagram_test.go` — the CLI leaves and the rule list
- `internal/pipeline/pipeline_test.go` — the viewer's entry point reports nothing
- `docs/dsl-reference.md` — the bullet in `### screen`

**Patterns to Follow:**
- `internal/linter/linter.go:50-169`, `:694-707`
- `internal/linter/descriptions.go:32-61`, `internal/linter/descriptions_test.go:47-54`
- `internal/oracle/oracle.go:13-42`
- `internal/cli/lint.go:80-118`, `internal/cli/validate.go:13-38`
- `internal/cli/lint_test.go:1466-1499`, `:1932-1968`
- `internal/cli/validate_test.go:37-60`, `:963`, `:1316-1323`
- `internal/pipeline/pipeline_test.go:103-160`, `:205-222`
- `internal/cli/export.go:28-57`, `internal/cli/diagram.go:40-70`
- `docs/dsl-reference.md:430`, `:735`
- `tasks/learnings.md:146-150` (`RuleName` and `--explain`), `:428-432` (a warning fails `emod validate`),
  `:433-437` (a lint fixture trips exactly one rule), `:1491-1497` (why `os.Root` does not apply here), and the
  learnings "A guard that parses Go source reads the whole package", "A leaf that pairs the feature with something
  independently satisfying the same rule cannot see the feature"
- `task:1`

**Testable:** Yes

**Certainty:** low — no rule of the linter reads the disk, and `Lint` and the oracle promise no I/O, so the entry
point that gives the rule the location of the file is new design, and so is the resolution of a relative, an
absolute and a directory path.

**Blast radius:** high — it changes the exit status of `emod validate`, a CLI contract that authors' CI pipelines
read, and a wrong resolution warns on every valid model.

**Verification:** `mise exec -- task test:unit` and `mise exec -- task test:integration` pass.
`emod lint --explain screen/missing-image-asset` prints the description. `emod lint` of a model in a temporary
directory reports the rule while its mockup is absent and reports nothing once the file is there.

**Depends on:** Task 1

---

### Task 3: Give `image` in the JSON and CUE exports and the schema, and draw every picture as before

**Behavior:** The JSON and CUE exports give the image of each screen, and the schema accepts it. A model with an
image draws each picture exactly as the same model with no image does.

**Acceptance Criteria:**
- [x] The JSON export gives a screen with an image the key `image`, the path as written, and `image_position`, the
      position of the value. `image_position` follows `position`, and `image` follows `comments`, as an
      automation's `every_position` and `every` do. A screen with no image has neither key.
- [x] The CUE export gives a screen with an image the key `image` after `description`, and a screen with no image
      none.
- [x] `#Screen` in `internal/cue/schema.cue` declares `image` as an optional string. The exports of the featured
      fixture conform to `#Model`, and the CUE and the JSON export of it describe the same model.
- [x] Both exports of the featured fixture carry the images of `ScreenImageLibraryLendingImages`, in declaration
      order. Both exports of its twin carry no `image` key and none of those paths.
- [x] For a model with no image, each export is byte-identical to before: each existing export expectation passes
      unchanged.
- [x] For each format of `diagram.Formats()`, the drawing of the featured fixture is byte-identical to the drawing
      of its twin: an image draws nothing (story criterion 2).

**Affected Files/Modules:**
- `internal/export/json.go` — `jsonScreen` and `convertScreen`
- `internal/export/cue.go` — `cueWriter.writeScreen`
- `internal/cue/schema.cue` — `#Screen`
- `internal/export/export_test.go` — the exports of the kit, the key order, schema conformance and parity
- `internal/diagram/contract_test.go` — the drawings of the fixture and its twin

**Patterns to Follow:**
- `internal/export/json.go:36-43`, `:170-189`, `:345-354`
- `internal/export/cue.go:88-92`, `:237`
- `internal/cue/schema.cue:15-19`, `:68-72`
- `internal/export/export_test.go:4653-4690`, `:4920-4974`
- `internal/diagram/contract_test.go:1332-1360`
- `tasks/learnings.md:46-50` (JSON, CUE and the schema move together), `:271-275` (a key spells the keyword),
  `:41-45` (the byte-identical receipt), `:141-145` (the two export guards cannot see a key neither writer emits)
- `task:1`

**Testable:** Yes

**Certainty:** high — an automation's `every` and `every_position` are the precedent in each of the three files,
and the wire-type tests are the precedent for the read-back over the kit.

**Blast radius:** high — the JSON export, the CUE export and `schema.cue` are contracts that consumers outside the
repository read.

**Verification:** `mise exec -- task test:unit` and `mise exec -- task test:integration` pass (the schema leaves
need the `cue` binary).

**Depends on:** Task 1

---

### Task 4: Carry `image` through the diagram document and both viewer saves

**Behavior:** The diagram document gives each `screen` node its image, and copies the image of the declaration onto
each `shows` node of that screen, as it copies the description. Each save from the viewer keeps the image on the
declaration, and never takes it from a `shows` node.

**Acceptance Criteria:**
- [x] The `screen` node of a screen with an image carries the key `image`, the path as written. The `screen` node
      of a screen with no image has no `image` key.
- [x] Each `shows` node carries the `image` of the declaration of its screen: two appearances of one screen carry
      the same path, and an appearance of a screen with no image has no `image` key.
- [x] `importer.ImportDiagram` of the document of the featured fixture gives back each image on its declaration
      (`DeclaredScreenImages` equals `ScreenImageLibraryLendingImages`), and the import of its twin gives none.
      For a source in canonical `emod fmt` form with images, formatting the import of its document gives back the
      source byte for byte.
- [x] A save takes `image` from `screen` nodes only. When a `shows` node carries an `image` that no `screen` node
      of that name carries (a renamed appearance, or a document with no `screen` nodes), the declaration that the
      save adds has no image, and the saved text parses with no diagnostic, so no `shows` block holds `image`.
- [x] `importer.ApplyDiagram` keeps the image of each screen when the viewer edits the document. When the viewer
      renames an appearance to a screen that no declaration holds, the saved text declares that screen with no
      image, and the screen that the appearance left keeps its image.
- [x] The document of the twin carries no `image` key, and the document of a model with no image is byte-identical
      to before: each existing expectation of the diagram document passes unchanged.

**Affected Files/Modules:**
- `internal/export/diagram.go` — `jsonDiagramNode`, `appendScreen`, `appendShows`
- `internal/export/export_test.go` — the keys of both node types
- `internal/importer/importer.go` — `diagramNode` and `asScreen`
- `internal/importer/importer_test.go`, `internal/importer/diagram_edit_test.go` — both save paths

**Patterns to Follow:**
- `internal/export/diagram.go:187-189`, `:228-238`, `:403-422`
- `internal/export/export_test.go:2190-2219`
- `internal/importer/importer.go:50-71`, `:127-142`, `:249-291`
- `internal/importer/diagram_edit.go:113-142`, `:456-463`
- `internal/importer/importer_test.go:30-40`, `:577-592`
- `internal/importer/diagram_edit_test.go:589`
- `tasks/learnings.md:201-205` (the viewer's save owes a read-back), `:1701-1707` (a split construct loses its
  description in the viewer), and the learnings "A diagram-node key has three readers", "A comment rides a diagram
  node without its position, and only where a node exists"
- `task:1`

**Testable:** Yes

**Certainty:** medium — the screen's description is the precedent for the copy onto each `shows` node and for the
read back from the `screen` node, but no leaf holds that a value copied onto an appearance never returns to the
source, and a renamed appearance must declare its new screen without the copy it carries.

**Blast radius:** high — a save from the viewer writes the author's `.emod` file from this document, and the
document is the output of `emod export --format diagram-json` and of the MCP tool `emod_export`.

**Verification:** `mise exec -- task test:unit` and `mise exec -- task test:integration` pass.

**Depends on:** Task 1

---

### Task 5: Show the image path of its screen in the detail panel of an appearance

**Behavior:** The detail panel of a `shows` node gives the image path of its screen in the Screen section, after the
views and the commands of the appearance. The desktop app shows the same panel.

**Acceptance Criteria:**
- [ ] The Screen section of the panel of a `shows` node has the rows Reads, Commands and Image, in that order. The
      Image row gives the node's `image` as text (story criterion 6).
- [ ] A `shows` node with no `image`, or with an empty one, shows the em-dash in the Image row.
- [ ] A path that contains markup shows as text, and the panel holds no element that the path spells.
- [ ] A node that states its path under a key that the exporter never writes (for example `wireframe`) shows the
      em-dash in the Image row.
- [ ] The panel of a node of any other type has no Image row.
- [ ] The leaves that compare the whole list of rows of a `shows` node
      (`internal/frontend/tests/detail-panel.test.js:173-217`) include the Image row.

**Affected Files/Modules:**
- `internal/frontend/static/ui.js` — `showDetailPanel`
- `internal/frontend/tests/detail-panel.test.js` — the shows group

**Patterns to Follow:**
- `internal/frontend/static/ui.js:520-526`
- `internal/frontend/tests/detail-panel.test.js:141-165`, `:168-217`
- `tasks/learnings.md:271-275` (the viewer reads the key that the exporter writes), and the learnings "A
  diagram-node key has three readers", "The viewer shows a node twice — the canvas box and the detail panel" (only
  the panel is in scope here)
- `task:4`

**Testable:** Yes

**Certainty:** high — the Reads and Commands rows of the same section are the precedent, and the automation's rows
are the precedent for the em-dash of a key that the exporter never writes.

**Blast radius:** low — text that a person reads in the viewer.

**Verification:** `mise exec -- task test:viewer` passes. Open a model with an image in the viewer (`emod diagram
--serve`) and open the detail panel of each appearance of that screen.

**Depends on:** Task 4

---

## Summary

- **Total tasks:** 5.
- **Order.** Task 1 comes first, because each other task reads the field and the fixture kit that it adds. Task 2 is
  the riskiest (low certainty, and it changes what `emod validate` accepts), so it comes next, where a wrong seam
  costs the least. Tasks 3 and 4 depend only on Task 1. Task 5 reads the key that Task 4 writes.
- **Assessments.** Certainty is high for Tasks 1, 3 and 5, medium for Task 4, and low for Task 2. Blast radius is
  high for each Go task, because each touches a file that the author keeps (`emod fmt`, a viewer save) or a
  contract that someone outside the repository reads (the exports, the exit status of `emod validate`), and low
  for the viewer panel.
- **Coverage of the story's criteria:**
  1. `image` accepted and kept by `emod fmt` — Task 1; by the JSON and CUE exports — Task 3; by a save from the
     viewer — Task 4.
  2. A model whose path leads to no file parses (Task 1), exports and draws (Task 3), and `emod export` and
     `emod diagram` of it exit 0 (Task 2).
  3. `emod lint` reports the rule at the `image` attribute, and nothing when the file exists — Task 2.
  4. No file system, no rule: the web viewer, the desktop app and every caller of the in-memory chain — Task 2.
  5. `--explain` and the reference — Task 2.
  6. The detail panel of an appearance — Task 5 (the key reaches the appearance in Task 4).
  7. `docs/dsl-reference.md` documents `image` — Task 1.
- **Recorded consequence of the severity:** a model whose mockup is missing fails `emod validate` (exit 1), and
  still parses, exports and draws. The repository's own models do not use `image`.
- **Deferred:** the rule in the LSP and in the MCP tools (see Boundaries).

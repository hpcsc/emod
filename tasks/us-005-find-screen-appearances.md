# Task Breakdown: Find each appearance of a screen

## Contents

1. Task 1: Answer for a screen in `emod references`, go-to-definition and find-references
2. Task 2: Offer the declared screens after `screen =` in a `shows` block

## Story Reference

`user-stories/screens-as-first-class.md`, US-005 "Find each appearance of a screen" (story ID
`screens-as-first-class/US-005`). Goal G1 of `docs/proposals/screens-as-first-class-proposal.md`, section
"Name resolution (`internal/query`)" and section "LSP (`internal/lsp`)". US-001 to US-004 are on main: a screen
is declared one time at the top level (`model.Screens`), and each slice shows it with a `shows` block
(`slice.Appearances`).

Decisions for this run:

- **A screen is a declared name of the model, as a command, an event and a view are.** `query.Declarations`,
  `query.NewDeclaredNames` and `query.NameSites` carry it, and each surface reads it from there: `emod
  references` through `query.References`, go-to-definition and find-references through the declared names and
  the name sites, and completion through `declaredNamesInOrder`. No surface reads `model.Screens` for itself.
- **The references of a screen are one group, `shown in`.** Each `shows` block that names the screen is one
  reference, with the keyword `screen` (the entry that holds the name, as `reads` and `command` are the keywords
  of the references that a `shows` block makes to a view and a command), the line of that entry, and the name of
  its slice. A `shows` block has no name, so the reference has no element. Specs never name a screen, so a screen
  has no `named in specs` group.
- **A screen has no place.** The model declares it at the top level, so its block in `emod references` has no
  context, no aggregate and no slice: the text header gives the kind, the name and the line only, and the JSON has
  no `context`, `aggregate` or `slice` key, as the screen node of `emod trace` has none.
- **`screen` is a reference kind after `view`.** `query.ReferenceKinds` lists command, event, view, screen and
  invariant, the order in which `query.TraceKinds` gives the first four. Each message that lists the kinds follows
  that order, and `emod references` builds its unsupported-kind message from the list, as `emod trace` does.
- **A screen completes as a variable.** `GetSemanticTokens` paints a screen as a parameter, and LSP has no
  parameter item kind, so the item kind is `VariableCompletion`, the item kind of a parameter in most editors.

## Boundaries

**Out of scope:**

- From the stories file, verbatim: "Layout, style, or the structure of components on a screen. A screen names its
  mockup by its path only."
- From the stories file, verbatim: "Navigation between screens. A screen graph is a different model."
- From the stories file, verbatim: "A mockup for each appearance. A second mockup means a second screen."
- From the stories file, verbatim: "Images inside `.emod` files."
- From the stories file, verbatim: "Changes to how `view`, `automation` or `translation` connect."
- From the stories file, verbatim: "A period where `trigger` and `screen` are both valid, and a new language
  version. The proposal retires `trigger` directly (Versioning section)."
- From the stories file, verbatim: "A `title` attribute that restores a human label with spaces."
- From the stories file, verbatim: "A tool that migrates a model. The migration is by hand, with the steps in the
  proposal."
- `emod trace`. It already takes the kind `screen`, and its screen node has the line of the declaration and no
  context or slice (`internal/query/trace.go:155-160`). This story does not change it.
- Hover on the screen name in a `shows` block. Hover describes a construct at its declaration only, for each kind,
  and the declaration of a screen already has a hover. A reference site of any kind gets none.
- Completion after `command =` in a `shows` block. No `command` value slot exists in any block, and the test
  "after command in a shows block returns no command name" (`internal/lsp/completer_test.go:116`) stays as it is.
- Semantic highlighting of the screen name in a `shows` block. Semantic tokens paint declarations only, for each
  kind.
- The editor feature lists in `README.md:419` and `site/content/docs/editor-setup.md:14` also leave out an
  invariant. Task 1 adds a screen to those lists and does not add the invariant (`tasks/learnings.md:428-431`).

**Deferred:**

- The MCP tools of `user-stories/model-queries.md`. They do not exist yet. When they ship, they read
  `query.References`, so they give the answer of Task 1 for a screen with no further work.
- Highlighting the other appearances of a screen in the viewer and the desktop app: US-006.

## Codebase Context

- **One package answers.** `internal/query` resolves names for `emod references` (`internal/cli/references.go`),
  `emod trace` (`internal/cli/trace.go`) and the LSP (`internal/lsp/definition.go`, `references.go`,
  `completer.go`, `model.go`). No MCP tool imports it.
- **Screens are top-level; appearances are in slices.** `ast.Model.Screens` holds each `screen` declaration;
  `ast.Screen.NamePos` is at the opening quote, so a declaration position moves one column onto the name, as
  `afterQuote` in `internal/query/names.go:77-80` does for the other kinds. `ast.Slice.Appearances` holds each
  `shows` block; `ast.Appearance.ScreenPos` is at the first character of the bare name
  (`internal/parser/parser_test.go:150-153`). `model.AllSlices()` gives the slices of both homes, an aggregate's
  and a `mode dcb` context's own (`tasks/learnings.md:151-154`).
- **Today a screen is not a declared name.** `query.Declarations` (`names.go:32-66`) gives contexts and the
  commands, events and views of each slice. `query.NewDeclaredNames` (`names.go:85-98`) starts one map for each of
  four kinds, and a declaration of a kind with no map would write into a nil map. `query.NameSites`
  (`names.go:111-169`) gives the `reads` and `command` entries of each `shows` block (`:135-142`) but not its
  `screen` entry, and `TestNameSites` (`names_test.go:13-46`) pins that list whole.
- **`query.References` answers for `ReferenceKinds` only** (`references.go:18`, `:108-140`). `KindScreen` exists
  (`references.go:13`); `GroupsOf` (`:33-45`) and `declarationReferences` (`:189-211`) have no screen arm.
  `NameReferences.Context` has no `omitempty` tag (`:51`), and `Reference.Element` has one (`:62`): a spec
  reference already has no element. `unknownName` (`:357-376`) names the kinds of `ReferenceKinds` in its message,
  and `namesOf` (`:382-398`) draws the closest names from `Declarations`.
- **The CLI text assumes a place and an element.** `formatNameReferences` (`internal/cli/references.go:73-114`)
  writes the header with `query.ScopeName(context, aggregate)`, which is empty for a screen, and
  `referenceElement` (`:116-125`) reads `ref.Element` for each reference outside `named in specs`, which is nil for
  a reference with no element. The unsupported-kind message is a fixed string (`:27-33`), while `emod trace`
  builds its own from its kind list with `kindList` (`trace.go:68-74`). The `--kind` flag usage in
  `internal/cli/app.go:194-197` lists the kinds in words.
- **The editor gets most of it from `query`.** `GetDefinition` (`internal/lsp/definition.go:10-43`) is a loop over
  `query.NameSites` against `query.NewDeclaredNames`, so a screen site resolves with no change there.
  `GetReferences` (`internal/lsp/references.go:21-52`) answers only for `referenceTargetKinds` (`:10-13`).
- **The agreement test joins the CLI query and the editor.** `TestReferences` "agreement with the editor"
  (`internal/query/references_test.go:450-494`) takes each non-context entry of `query.Declarations` in each
  example, asks `query.References` and `lsp.GetReferences` at the declaration, and requires both to answer with the
  same lines. Each example declares screens, so the commit that makes a screen a declaration must also make
  `query.References` and find-references answer for it. Task 1 is therefore one task.
- **The completer reads raw lines** (`tasks/learnings.md:413-416`). `valueSlots` (`completer.go:345-352`) maps an
  entry keyword to the names after it; `slotFor` (`:364-375`) makes `context` a value slot in a `target` block only,
  because the same word opens a block elsewhere. `screen` is the same case: `blockKeyword` (`:290-321`) makes it
  open a `screen` block at the top level. The test "after screen in a shows block returns no screen name"
  (`completer_test.go:115`) pins the behaviour that Task 2 changes. `namesOfKind` (`:379-384`) reads
  `declaredNamesInOrder` (`model.go:224-232`), which filters `query.Declarations`.
- **Fixture.** `test.ScreenLibraryLending` (`internal/test/fixtures.go:3425-3628`) shows `DeskKiosk` from a slice of
  the aggregate "Loan" and from a slice of the `mode dcb` context "Reading Room", and `LendingDesk` from two slices.
  Each of its screens is shown, so a screen that no slice shows needs an inline model.
- **Documents.** `README.md:273-301` and `site/content/docs/diagrams.md:55-66` say which kinds `emod references`
  answers for; `README.md:419` and `site/content/docs/editor-setup.md:14` say which constructs find-references
  finds. `docs/dsl-reference.md` does not document `emod references`, and its cross-reference table already lists
  `screen` with `shows { screen }` as its reference.

## Tasks

### Task 1: Answer for a screen in `emod references`, go-to-definition and find-references

**Behavior:** `emod references <file> CheckoutPage` prints the declaration of the screen and each `shows` block
that names it, with the slice and the line of each, in text and in JSON. In an editor, go-to-definition on the
screen name of a `shows` block goes to the `screen` declaration, and find-references on the screen lists the
declaration and each `shows` block that names it. Each surface takes the answer from `internal/query`.

**Acceptance Criteria:**

- [x] `emod references <file> CheckoutPage` prints one block for the screen. Its header gives the kind `screen`,
  the name and the line of the declaration, with no context and no slice. Its one group, `shown in`, gives each
  `shows` block that names the screen, in source order: the line of its `screen` entry, the keyword `screen`, and
  the quoted name of its slice.
- [x] `emod references <file> CheckoutPage --format json` gives the same answer: the kind `screen`, the name and
  the line, no `context`, `aggregate` or `slice` key, and one reference for each `shows` block with the group
  `shown in`, the keyword `screen`, the line of the entry and the slice, and no `element` key.
- [x] A screen that no slice shows has a references list with its declaration only: `shown in` says `none` in the
  text, and `references` is an empty list in the JSON.
- [x] A screen that a slice of an aggregate and a slice of a `mode dcb` context both show lists both `shows`
  blocks.
- [x] A name that a screen and a command, an event or a view share gives one block for each kind, in source
  order, and the block of the screen lists only `shows` blocks.
- [x] `--kind screen` gives the block of the screen only. A kind that `emod references` does not answer for is
  refused with a message that lists exactly the kinds that `--kind` accepts, `screen` among them, and the usage of
  the `--kind` flag names `screen`. A name that the model does not declare names `screen` among the kinds it
  looked in, and the declared screens are among the closest names it can list.
- [x] The text and the JSON of `emod references` for a command, an event, a view and an invariant do not change,
  apart from the messages that list the kinds.
- [x] In an editor, go-to-definition on the screen name in a `shows` block goes to the name in the `screen`
  declaration, in a slice of either home. On a `shows` block that names no declared screen it goes nowhere, and
  on the declaration itself it goes nowhere, as for each other kind.
- [x] In an editor, find-references with the cursor on the name of a `screen` declaration, or on the screen name
  of a `shows` block, lists the declaration and the `screen` entry of each `shows` block that names the screen,
  and nothing else. For a screen that no slice shows, it lists the declaration only.
- [x] The test "agreement with the editor" in `internal/query/references_test.go` covers the screens of each
  example and passes.
- [x] `README.md` and `site/content/docs/diagrams.md` say that `emod references` answers for a screen, with the
  slices that show it. `README.md` and `site/content/docs/editor-setup.md` name a screen among the constructs that
  find-references finds.

**Affected Files/Modules:**

- `internal/query/names.go` — a screen is a declaration, a declared name and, at the `screen` entry of each
  `shows` block, a name site
- `internal/query/references.go` — `screen` in `ReferenceKinds`, the group `shown in`, the references of a
  screen, and no `context` key for a declaration without one
- `internal/query/names_test.go`, `internal/query/references_test.go` — the screen site, the references of a
  screen, the lookup and message cases
- `internal/cli/references.go` — the header and the reference line of a screen, and the unsupported-kind message
  from the kind list
- `internal/cli/app.go` — the usage of `--kind`
- `internal/cli/references_test.go` — the text, JSON and error cases of a screen
- `internal/lsp/references.go` — find-references answers for a screen
- `internal/lsp/definition_test.go`, `internal/lsp/references_test.go` — go-to-definition and find-references on
  a screen
- `README.md`, `site/content/docs/diagrams.md`, `site/content/docs/editor-setup.md` — a screen among the kinds

**Patterns to Follow:**

- `internal/query/names.go:32-66` and `internal/query/names.go:85-98` for a declaration kind and its map;
  `internal/query/names.go:135-142` for the sites of a `shows` block
- `internal/query/references.go:189-211` and `internal/query/references.go:297-330` for the references of one
  declaration kind; `internal/query/references.go:33-45` for its groups
- `internal/query/trace.go:155-160` for a screen with the line of its declaration and no place
- `internal/cli/references.go:73-125` for the text block; `internal/cli/trace.go:68-74` for a message built from a
  kind list; `internal/cli/trace.go:166-171` for a place that a node does not have
- `internal/lsp/references.go:10-13` for the kinds that find-references answers for
- Tests: `internal/query/references_test.go:197-285` (the `shows` group), `internal/query/references_test.go:364-447`
  (the `lookup` group), `internal/query/names_test.go:13-46`, `internal/lsp/definition_test.go:131-167`,
  `internal/lsp/references_test.go:264-310`, `internal/cli/references_test.go:66-100`,
  `internal/cli/references_test.go:173-192` and `internal/cli/references_test.go:214-238`
- `internal/test/fixtures.go:3425-3628` for a screen shown in both slice homes
- `tasks/learnings.md:151-154` (both slice homes), `tasks/learnings.md:408-411` (the LSP reads `query`, not a walk
  of its own), `tasks/learnings.md:595-598` (two kinds that share a name), `tasks/learnings.md:106-109` (an expected
  value from the code under test), `tasks/learnings.md:6-9` (assert the text that tells a message apart),
  `tasks/learnings.md:41-44` (the output of models that do not use the feature stays the same),
  `tasks/learnings.md:423-426` (the umbrella of a test file), `tasks/learnings.md:575-578` (audit a fix to a
  resolution path across every lookup)

**Testable:** Yes — through `query.References`, `query.NameSites`, `cli.RunReferences`, `lsp.GetDefinition` and
`lsp.GetReferences`.

**Certainty:** medium — the view and command arms of `internal/query` and `referenceTargetKinds` are the
precedent, but no reference kind so far is declared at the top level, so no block of `emod references` has had no
place and no reference has had no element outside `named in specs`.

**Blast radius:** high — a CLI's output format: it fixes the text and the JSON of `emod references` for a screen
(the kind, the group `shown in`, the keyword, the keys it leaves out), which scripts and the MCP tools of
model-queries read, and it changes the `context` tag that each block of the JSON uses.

**Verification:** `mise exec -- task test:unit` passes, with the agreement test over each example. `mise exec -- go
run ./cmd/emod references examples/all_patterns.emod ReservationForm`, and the same with `--format json`, print
the block of the screen.

**Depends on:** None

### Task 2: Offer the declared screens after `screen =` in a `shows` block

**Behavior:** In an editor, completion after `screen =` in a `shows` block offers each screen that the model
declares. Everywhere else, `screen` keeps its meaning: at the top level it opens a declaration.

**Acceptance Criteria:**

- [x] In a `shows` block, completion after `screen =` offers each declared screen, in declaration order, as items
  of kind `VariableCompletion`. This holds for a `shows` block in a slice of either home, for a `shows` block
  written on one line, and while the `screen` entry has no value yet and the document does not parse.
- [x] With the cursor inside a screen name that the author has begun to type, completion still offers the declared
  screens.
- [x] At the top level, completion after `screen` offers the top-level keywords and no screen name. Inside a
  `screen` block, completion offers `description` only. After `command =` in a `shows` block, completion offers
  the entries of the block (`screen`, `reads`, `command`) and no name, as it does now.
- [x] The test that pins no screen name after `screen =` in a `shows` block pins the declared screens.

**Affected Files/Modules:**

- `internal/lsp/completer.go` — `screen` is a value slot inside a `shows` block only
- `internal/lsp/completer_test.go` — the screen slot and the places where `screen` is no slot

**Patterns to Follow:**

- `internal/lsp/completer.go:364-375` for a word that is a value slot in one block only (`context` in a `target`
  block); `internal/lsp/completer.go:345-352` and `internal/lsp/completer.go:379-384` for a slot that offers the
  declared names of one kind
- Tests: `internal/lsp/completer_test.go:87-124` (the `screen and shows blocks` group) and
  `internal/lsp/completer_test.go:257-400` (the `value position` group, with the half-written document and the
  name begun)
- `task:1` for the screens in `query.Declarations`, which `declaredNamesInOrder` reads
- `tasks/learnings.md:413-416` (the completer reads raw lines, and the item kind follows the semantic token),
  `tasks/learnings.md:151-154` (both slice homes)

**Testable:** Yes — through `lsp.GetCompletions`.

**Certainty:** high — `slotFor` already makes `context` a value slot in one block only, and `namesOfKind` already
offers the declared names of one kind.

**Blast radius:** low — it adds completion items in one editor position.

**Verification:** `mise exec -- task test:unit` passes.

**Depends on:** Task 1

## Summary

- **Total tasks:** 2.
- **Order:** dependency first. Task 1 makes a screen a declared name of `internal/query`, and Task 2 offers the
  declared screens from that list. Task 1 cannot be split by surface: the agreement test requires `emod references`
  and find-references to answer for each declaration of each example in the same commit, the CLI text cannot
  print a screen block until it handles a block with no place and a reference with no element, and
  go-to-definition follows from the same name sites with no code of its own.
- **Assessments:** this story is routine. It extends one package that each surface already reads, so no task is
  `low` certainty. Task 1 is `medium` because a screen is the first reference kind with no place, and `high` blast
  radius because it fixes the CLI output of a screen.
- **Story criteria:**
  - "`emod references <file> CheckoutPage` lists the declaration of the screen and each `shows` block that names
    it, with the slice and the line of each." — Task 1.
  - "In an editor, go-to-definition on `CheckoutPage` in a `shows` block goes to the `screen` declaration." —
    Task 1.
  - "In an editor, find-references on a `screen` declaration lists each `shows` block that names the screen." —
    Task 1.
  - "In an editor, completion after `screen =` in a `shows` block offers each declared screen." — Task 2.
  - "A screen that no slice shows has a references list with its declaration only." — Task 1, for `emod
    references` and for find-references.
  - Context: "`emod references`, `emod trace` and the LSP take their answers from one package" — Task 1 adds no
    resolution outside `internal/query`; `emod trace` already answers for a screen.
- **Deferred:** none of the story's criteria.

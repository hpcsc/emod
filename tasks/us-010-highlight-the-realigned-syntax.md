# US-010: Highlight the realigned syntax

## Progress
- [x] Task 1: Pin tree-sitter highlighting with an executable highlight suite, and drop the trigger's kind capture
- [x] Task 2: Highlight `on` and `every` as keywords, and a field named after either as a field name
- [x] Task 3: Pin folding, indentation and text-object captures over the realigned blocks
- [ ] Task 4: Give the VS Code grammar an executable scope test
- [ ] Task 5: Highlight the kindless trigger's quoted name in VS Code
- [ ] Task 6: Highlight `on` and `every` in VS Code only where their own operand follows

---

## Story Reference

`user-stories/triggers-and-automations.md` → **US-010: Highlight the realigned syntax** (tenth of
eleven stories in "Triggers and Automations"). Design notes:
`docs/proposals/triggers-and-automations-proposal.md` — section 1 (`:52-70`) for the kindless trigger,
section 2 (`:71-97`) and section 3 (`:98-112`) for the two activation forms, and `:254-261` for the
editor surfaces this story owns.

**This story assumes US-004 has landed.** US-004 drops the trigger kind slot: `trigger "<name>" { … }`
parses with the quoted name directly after the keyword, and `trigger UI "<name>"`,
`trigger Schedule "<name>"` and `trigger Processor "<name>"` are rejected. Its grammar work —
`trigger_definition` in `editors/tree-sitter-emod/grammar.js:169-178` losing `$.identifier`, and the
three trigger fixtures in `test/corpus/slice.txt:96`, `:497` and `test/corpus/description.txt:220`
migrating to the kindless form — belongs to US-004, following the precedent US-003 set by carrying its
own grammar rule in Task 3 of `tasks/completed/us-003-activate-an-automation-on-a-schedule.md`. This
story changes no rule in `grammar.js` and migrates no corpus fixture. Its second criterion — a
trigger's quoted name highlighting as a name, now that no kind identifier precedes it — is meaningless
before US-004 lands, and Tasks 1, 3 and 5 all read the post-US-004 shape.

**In scope:** the tree-sitter highlight, fold, indent and text-object queries under
`editors/tree-sitter-emod/queries/`, and the TextMate grammar at
`editors/vscode/syntaxes/emod.tmLanguage.json` — two independent implementations of the same
requirement, neither of which has ever been verified by anything executable. Carried along because the
criteria are otherwise unfalsifiable: a highlight suite under `editors/tree-sitter-emod/test/highlight/`
that `tree-sitter test` already knows how to run, a query-compilation check for the three query files
`tree-sitter test` does not load, and a TextMate scope test for the VS Code grammar with its Taskfile
target and CI step.

**Out of scope, owned elsewhere:** dropping the trigger kind slot itself, including the `grammar.js`
rule and the corpus fixtures (US-004); the LSP's hover, completion and navigation over `on` and `every`,
including `keywordDescriptions` (`internal/lsp/hover.go:13`), the ordinal-range `isKeyword` beside it
(`:37`) and the per-context lists in `internal/lsp/completer.go` (US-009); `docs/dsl-reference.md`,
`README.md` and `examples/*.emod` (US-011); a `wireframe` entry on a trigger, which the proposal's
Non-Goals defer (`:207`) and which therefore reaches no grammar or highlighter here.

**Consequences of that boundary, decided.** Seven things the story does not spell out:

1. *`highlights.scm` is broken the moment US-004 lands, and nothing in the repo notices.* Verified:
   with `trigger_definition` no longer admitting an `identifier`, the capture at
   `editors/tree-sitter-emod/queries/highlights.scm:60` becomes an impossible pattern, and tree-sitter
   refuses to compile the **whole file** — `tree-sitter query queries/highlights.scm <file>` exits 1
   with `Query error at 60:21. Impossible pattern`. Every highlight in the language goes with it, not
   just the trigger's. **Correction to an earlier reading of this: CI is not blind to it.**
   `tree-sitter test` compiles the query file, so `task test:grammar` catches it — measured by injecting
   `(string (identifier) @x)` into an otherwise-green tree, which flips `mise exec -- tree-sitter test`
   from exit 0 to exit 1 with `Query error ... Impossible pattern`. US-004 therefore *cannot* land this
   green: its own grammar task fails its own verification until `:60` goes, and US-004's breakdown now
   carries deleting that line. What remains true is the ordering caveat in finding 3 — the highlight
   section runs only once the corpus is green, so a task that leaves the corpus red still hides query
   failures behind corpus ones. Task 1 stays first: if US-004 has already deleted `:60`, Task 1 has
   nothing to repair and its value is the harness it establishes, not the repair.
2. *The tree-sitter half is testable, and this story is where that stops being a claim.* Verified
   against the repo-pinned CLI (`mise.toml` pins `cargo:tree-sitter-cli = "0.26.9"`): `tree-sitter test`
   runs `test/highlight/*.emod` assertion files in addition to the corpus, needs no extra configuration
   beyond the `highlights` entry already in `tree-sitter.json`, prints a `syntax highlighting:` section,
   and **exits 1** both when an assertion fails and when the query file fails to compile. Because
   `task test:grammar` already runs `tree-sitter test` and already runs in CI, the suite becomes a real
   gate with no Taskfile or workflow edit. Every prior task touching these files was written
   `Testable: No` (`tasks/completed/tree-sitter-grammar.md:106`,
   `tasks/completed/us-ide-013-neovim-integration.md:50`, `:81`); that was true of the tooling then
   available and is not true now.
3. *The highlight suite runs only when the corpus is green.* Verified: with three corpus cases failing,
   `tree-sitter test` never reaches the highlighting section. So a highlight assertion cannot substitute
   for a corpus case, and a task that leaves the corpus red hides its own highlight failures.
4. *"Highlights as a field name" is read positively on the tree-sitter side and negatively on the
   TextMate side.* `highlights.scm` captures a field's type and its modifier (`:64-74`) but not its
   name, so there is no capture for an assertion to name. Giving `field_line`'s name position its own
   capture makes the criterion checkable and costs nothing — verified that it coexists with the type and
   modifier captures, all three asserting correctly on one field line. TextMate has no field-name scope
   for any field and no context in which to acquire one cheaply, so there the criterion is met by `on`
   and `every` receiving no keyword scope in field position — exactly the plain-text treatment
   `roomType` gets today.
5. *The TextMate grammar's other keywords stay as they are.* The alternation at
   `editors/vscode/syntaxes/emod.tmLanguage.json:63` is case-insensitive and positionless, so
   `command string required` inside a `fields` block already paints `command` as a keyword —
   `tasks/completed/us-003-use-reserved-words-as-field-names.md:57-61` records that and declines to fix
   it. The story's criterion names `on` and `every`; widening the repair to the other twenty-eight
   keywords is a restructuring of the whole grammar file and is not carried here. Tasks 5 and 6 leave
   every existing pattern's behaviour where it stands, and Task 4's test pins that so the boundary is
   visible rather than assumed.
6. *The LSP is not a third highlighting surface for this story.* `GetSemanticTokens`
   (`internal/lsp/semantictokens.go`) emits tokens for contexts, aggregates, commands, events, views and
   actors only — it names no trigger and no keyword, so neither the kindless trigger's name nor `on` and
   `every` reach it. Confirmed by reading the file: it contains no reference to a trigger. Nothing in
   this story edits `internal/lsp`.
7. *No `.emod` file outside `editors/` is touched.* `examples/`, `internal/parser/testdata/` and the Go
   fixtures belong to US-004 (migration) and US-011 (documentation). The sample sources this story adds
   live under `editors/tree-sitter-emod/test/highlight/` and beside the VS Code grammar, and exist to be
   read by a highlighter rather than by `emod validate`.

**Learnings folded in** from `tasks/learnings.md`: new DSL keywords must stay usable as field names, and
the tree-sitter grammar mirrors the Go parser on this with a permissive `any_identifier` plus keyword
tokens narrow enough to match only in their own position — which is why the tree-sitter half of the
field-name criterion is structural rather than regex work; generated tree-sitter `src/` stays
gitignored, so nothing this story does may un-ignore it or commit a regenerated artefact; run repo
tooling through `mise exec --`, because a bare PATH `tree-sitter` is a different pinned version;
keyword surfaces fan out past the lexer, parser and grammar into the TextMate alternation, and each
surface is a per-keyword decision rather than automatic propagation; every `grammar.js` rule carries a
one-line example of its full shape, and the same courtesy is owed to a query file's section comments;
the tree-sitter grammar must never be stricter than the Go parser; an assertion whose expected value
comes from the code under test is the recurring review finding, so a highlight assertion must be able to
fail — the trigger-name case is the witness, because a name the grammar cannot parse falls back to the
generic string capture; acceptance criteria describe the working tree, never the repository's history,
and a commit-message receipt is the commit author's obligation, never a criterion.

---

## Codebase Context

**The tree-sitter highlight query.** `editors/tree-sitter-emod/queries/highlights.scm` is 80 lines in
seven commented sections: comments (`:12`), a generic string capture (`:16`), a bracketed list of
nineteen anonymous keyword tokens (`:20-40`), quoted entity names for `model`/`actor`/`context`/
`aggregate`/`slice` (`:46-50`), identifier entity names for `command`/`event`/`view`/`automation`/
`translation` (`:53-57`), the trigger's kind identifier and quoted name (`:60-61`), field types
(`:64-67`) and field modifiers (`:70-74`) selected by a regex predicate over `field_line`'s second and
third `any_identifier`, operators (`:77`) and punctuation (`:80`). The keyword list omits `on` and
`every`, and also `spec`, `given`, `when`, `then`, `rejected` and `invariant`, which the grammar admits
today. Its header comment states that it mirrors the TextMate grammar's scope assignments.

**Why the field-name criterion is already half-won here.** `grammar.js` sets `word: $ => $.identifier`
(`:15`), and `identifier` (`:253`) is PascalCase-only, so lowercase keyword spellings are not extracted
through the word token and compete with `any_identifier` (`:256`) by parse state alone. Inside
`fields_block` (`:152-157`) no keyword token is valid, so a field named `on` parses as `any_identifier`
— pinned by the keyword-per-field corpus case at `test/corpus/fields.txt:1-47`, whose list already
carries `on` (`:45`) and `every` (`:46`). A highlight query that captures anonymous tokens therefore
cannot reach a field name. What is missing is not correctness but a capture to assert against.

**The other three query files.** `folds.scm:9-22` lists twelve node types as `@fold`; `indents.scm`
pairs `"{"` `@indent` with `"}"` `@dedent` per construct (automation at `:15-17`, trigger at `:47-49`)
plus the bracketed `subscribes_block` (`:56-58`); `textobjects.scm` lists eleven node types as
`@block.outer` (`:14-26`) and the same eleven brace-delimited as `@block.inner` (`:32-66`, automation at
`:36-38`, trigger at `:60-62`). All three name `trigger_definition` and `automation_definition` as whole
nodes or by their braces, never by a child, so US-004's rule change does not invalidate them — but
nothing in the repo has ever compiled them against the grammar. `tree-sitter.json` declares only
`queries/highlights.scm`, so `tree-sitter test` loads that one file and no other.

**The TextMate grammar.** `editors/vscode/syntaxes/emod.tmLanguage.json` is 83 lines. Its top-level
`patterns` list (`:6-15`) orders comments, then `keyword-entity`, then strings, then the flat keyword
alternation, then field modifiers, field types, operators and punctuation. `keyword-entity` (`:25-61`)
holds the four context-sensitive patterns: `trigger` with a kind identifier and a quoted name (`:27-35`,
which US-004 makes dead), keywords taking a quoted entity name (`:36-43`), keywords taking an identifier
entity name (`:44-51`, which still lists `trigger`), and `target` with its trailing context name
(`:52-59`). The flat alternation at `:63` carries nineteen spellings and neither `on` nor `every`.
Everything in the file is case-insensitive, while the lexer's keyword lookup is case-sensitive —
`tasks/completed/us-003-activate-an-automation-on-a-schedule.md:268` records a subtest pinning that an
event may be named `Every`. Field names carry no scope; only types (`:70-73`) and modifiers (`:66-69`)
do, both as positionless word matches.

**What runs today.** `Taskfile.yml` has `test` (`:54`) fanning out to `test:unit`, `test:integration`,
`test:viewer` and `test:grammar`; `test:grammar` (`:79`) changes into `editors/tree-sitter-emod` and
runs `tree-sitter generate` then `tree-sitter test`; `test:viewer` (`:72`) is the precedent for a
Node-based target that runs `npm ci` in a subdirectory before its tests. `build:vscode` (`:17`) compiles
`src/extension.ts` only and never reads the syntax file; `setup:vscode` (`:24`) symlinks the extension
into `~/.vscode/extensions`. Neither runs in CI. `.github/workflows/ci.yml` runs `task test:grammar` as
its only step touching `editors/`, plus `task test:viewer` behind a `Setup Node.js` step keyed to
`internal/viewer/package.json`. `editors/vscode/package.json` declares the grammar contribution, has
`compile` and `watch` scripts (`:34-37`) and one devDependency group (`:38`) holding TypeScript and the
VS Code and Node types.

**Not touched, deliberately.** `editors/tree-sitter-emod/grammar.js` and everything under
`test/corpus/` (US-004); `internal/lsp` (US-009); `docs/`, `README.md` and `examples/` (US-011);
`internal/lexer/token.go`, which already knows both keywords; every Go package, `internal/viewer`,
`e2e/` and `e2e-viewer/`.

---

## Tasks

### Task 1: Pin tree-sitter highlighting with an executable highlight suite, and drop the trigger's kind capture

**Behavior:** `editors/tree-sitter-emod/queries/highlights.scm` compiles against the post-US-004
grammar and gives a trigger's quoted name the entity-name treatment it gives every other construct's
name, with no pattern left reaching for a kind identifier that no longer exists. The claim is proved
rather than asserted: a highlight suite under `editors/tree-sitter-emod/test/highlight/` states, at
named positions in real emod source, which capture each token receives, and `task test:grammar` fails
when any of them is wrong or when the query file stops compiling.

**Acceptance Criteria:**
- [ ] `mise exec -- task test:grammar` passes and its output carries a `syntax highlighting:` section
      reporting the new file with a non-zero assertion count — the section is absent when no highlight
      file exists, so its presence is what shows the suite is running
- [ ] The suite asserts a kindless trigger's quoted name receives the entity-name capture and not the
      generic string capture — this assertion fails today for two independent reasons, and both must be
      gone: the query file does not compile, and a trigger the grammar cannot parse yields a string
- [ ] No pattern in `highlights.scm` names a child identifier of `trigger_definition`; running
      `tree-sitter query queries/highlights.scm` over a source file declaring a kindless trigger exits 0
      instead of reporting an impossible pattern
- [ ] The suite also pins the captures the file already assigns, so a later edit that moves one is
      caught: a comment, a quoted string that is not an entity name, a keyword, a quoted entity name
      following `model`/`actor`/`context`/`aggregate`/`slice`, an identifier entity name following
      `command`/`event`/`view`/`automation`/`translation`, a field's type and a field's modifier
- [ ] Every assertion in the suite is one that can fail: for each, changing the capture named in the
      assertion makes `mise exec -- task test:grammar` fail, and the run reports the offending row,
      column and the capture actually produced
- [ ] The suite's source is valid emod in the post-US-004 shape — `emod validate` is not run over it and
      it declares no model header, but no construct in it uses a spelling the parser rejects
- [ ] `git check-ignore editors/tree-sitter-emod/src` succeeds and
      `git status --porcelain editors/tree-sitter-emod` lists only `queries/highlights.scm` and files
      under `test/highlight/`
- [ ] `git diff` shows no change to `editors/tree-sitter-emod/grammar.js`, to any file under
      `test/corpus/`, to `Taskfile.yml` or to `.github/workflows/ci.yml` — the suite needs none of them

**Affected Files/Modules:**
- `editors/tree-sitter-emod/queries/highlights.scm` — the trigger section (`:59-61`)
- `editors/tree-sitter-emod/test/highlight/` — a new directory holding the assertion sources
- `editors/tree-sitter-emod/tree-sitter.json` — read only; its `highlights` entry is what lets
  `tree-sitter test` find the query file, and it already names it

**Patterns to Follow:**
- The assertion file format is tree-sitter's own: a source file whose lines are annotated by comment
  lines carrying a column marker and the expected capture name. emod's comment token is `#`, which the
  runner recognises from the grammar. Verified working with the pinned CLI; the runner reports the row,
  column, expected capture and actual captures on failure
- The captures to assert are the ones `highlights.scm` already names in its section comments
  (`:11-80`); the quoted entity names at `:46-50` are the sibling treatment a trigger's name joins
- `tasks/learnings.md` "Run repo tooling through `mise exec --`, not bare PATH" — the repo pins
  0.26.9 and a PATH CLI is a different version
- `tasks/learnings.md` "Generated tree-sitter `src/` stays gitignored" — `task test:grammar`
  regenerates before running, so nothing generated is added to the tree
- `tasks/learnings.md` "An assertion whose expected value comes from the code under test is the
  recurring review finding" — name each expected capture, and check by hand that changing it fails
- `tasks/learnings.md` "Every `grammar.js` rule carries a one-line example of its full shape" — the
  query file's section comments serve the same role and must still describe what the section captures

**Testable:** Yes — through `tree-sitter test`, driven by `task test:grammar`, which already runs in CI.

**Verification:** `mise exec -- task test:grammar`; `mise exec -- sh -c 'cd editors/tree-sitter-emod &&
tree-sitter query queries/highlights.scm <sample>'` exits 0.

**Depends on:** None (US-004)

---

### Task 2: Highlight `on` and `every` as keywords, and a field named after either as a field name

**Behavior:** an automation's activation reads as structure: `on` and `every` receive the keyword
treatment every other DSL keyword receives, wherever they introduce an automation entry. A field named
`on` or `every` inside a `fields` block receives the field-name treatment instead, alongside the type
and modifier treatments its line already gets — because emod keywords are not reserved words and the
grammar already parses those names as ordinary identifiers.

**Acceptance Criteria:**
- [ ] The highlight suite asserts that `on` introducing an automation's activation event and `every`
      introducing its schedule each receive the keyword capture, in the same automation block
- [ ] The highlight suite asserts that a field named `on` and a field named `every` inside a `fields`
      block each receive a field-name capture and not the keyword capture
- [ ] The same assertions hold for `on` and `every` written as a field's *type* and as a field's
      *modifier*, so the criterion covers every position a keyword may legally occupy on a field line
- [ ] A field line whose name is an ordinary word still reports the field-name capture on its name, the
      type capture on its type and the modifier capture on its modifier — all three asserted on one
      line, so a new name capture is proved not to have displaced the two that were already there
- [ ] Removing `on` or `every` from the keyword capture list makes `mise exec -- task test:grammar`
      fail, and removing the field-name capture makes it fail too — neither assertion passes vacuously
- [ ] `editors/tree-sitter-emod/grammar.js` is unchanged by this task: the grammar already admits both
      keywords inside an automation and both names inside a `fields` block, pinned by the corpus cases
      at `test/corpus/slice.txt:388`, `:421` and `test/corpus/fields.txt:45-46`, which pass unedited
- [ ] The keyword capture list stays a single bracketed list in the order the file already uses, and its
      section comment still describes what it captures
- [ ] `git status --porcelain editors/tree-sitter-emod` lists only `queries/highlights.scm` and files
      under `test/highlight/`

**Affected Files/Modules:**
- `editors/tree-sitter-emod/queries/highlights.scm` — the keyword list (`:20-40`) and the field sections
  (`:63-74`)
- `editors/tree-sitter-emod/test/highlight/` — the assertions added in Task 1

**Patterns to Follow:**
- The field type and modifier captures at `queries/highlights.scm:64-74` are the siblings a field-name
  capture files beside, and they select their position within `field_line` — the grammar rule is at
  `editors/tree-sitter-emod/grammar.js:162-166`
- `tasks/learnings.md` "New DSL keywords must stay usable as field names" — the tree-sitter side is
  already correct structurally, because a keyword token is only valid in its own parse state and a
  field name is an `any_identifier`; this task makes that assertable rather than changing it
- `tasks/learnings.md` "Keyword surfaces fan out past the lexer, parser and tree-sitter grammar" — the
  keyword list here is one of the hand-maintained surfaces; `spec`, `given`, `when`, `then`, `rejected`
  and `invariant` are also missing from it, and closing that gap belongs to the story that owns those
  keywords, not to this one
- `tree-sitter test` reports the capture actually produced when an assertion fails, which is how a
  capture that lands on the wrong token is distinguished from one that lands nowhere

**Testable:** Yes — through `tree-sitter test`, driven by `task test:grammar`.

**Verification:** `mise exec -- task test:grammar`.

**Depends on:** 1

---

### Task 3: Pin folding, indentation and text-object captures over the realigned blocks

**Behavior:** folding, auto-indentation and structural selection work on a kindless trigger block and on
an automation block stating either activation form, and a query file that stops matching the grammar
fails the build instead of silently degrading in an editor. `tree-sitter test` loads only the highlight
query, so the other three files gain their own check.

**Acceptance Criteria:**
- [ ] `mise exec -- task test:grammar` compiles each of `folds.scm`, `indents.scm` and
      `textobjects.scm` against the generated parser over a sample source declaring a kindless trigger,
      an automation stating `on`, an automation stating `every`, and a `fields` block — and fails if any
      of them reports a query error
- [ ] Deleting a node type from `grammar.js` that one of the three files names, or renaming one in the
      query file, makes that target fail — the check is proved to be reading both sides rather than only
      confirming a file exists
- [ ] The fold capture for the kindless trigger spans from its keyword through its closing brace, and
      the same holds for each of the two automations — so folding a trigger no longer depends on a kind
      identifier being present
- [ ] The outer text-object capture for each of those three blocks spans the same range as its fold, and
      the inner capture's bounds sit strictly inside the block's braces
- [ ] The indent and dedent captures land on the opening and closing braces of the trigger block and of
      each automation block
- [ ] The check is reachable from `task test` and runs in CI: `.github/workflows/ci.yml` executes it,
      either as its own step or through the `task test:grammar` step it already runs
- [ ] The sample source is the one the highlight suite already uses, or a sibling beside it — no second
      copy of the same shapes drifts from the first
- [ ] The check compares no generated bytes against a checked-in golden: the pinned CLI's output format
      is a version-dependent artefact, and the same trap that keeps `src/` gitignored applies here

**Affected Files/Modules:**
- `editors/tree-sitter-emod/queries/folds.scm`, `indents.scm`, `textobjects.scm` — read, and edited only
  if the check finds them wrong
- `Taskfile.yml` — `test:grammar` (`:79`)
- `.github/workflows/ci.yml` — only if the check needs a step of its own
- `editors/tree-sitter-emod/test/` — the sample source

**Patterns to Follow:**
- `tree-sitter query <query-file> <source>` compiles the query against the grammar and prints each
  capture with its name and its start and end position, exiting non-zero on a query error. Verified:
  all four query files exit 0 against the current grammar once the parser is regenerated, and
  `highlights.scm` exits 1 under the post-US-004 grammar until Task 1 repairs it
- `Taskfile.yml:79` already changes into `editors/tree-sitter-emod` and regenerates before testing; the
  parser must be regenerated first, because a stale `src/` compiles queries against the wrong grammar —
  observed while preparing this breakdown
- `tasks/completed/us-ide-013-neovim-integration.md:50-52`, `:81-83` describe what these three files
  were verified by until now — "file exists", "queries reference only valid node types", eyeballed in
  Neovim. This task replaces the manual cross-read with an executed one
- `tasks/learnings.md` "Run repo tooling through `mise exec --`, not bare PATH" and "Generated
  tree-sitter `src/` stays gitignored"
- `README.md:213-248` documents the Neovim text-object and highlighting setup this criterion serves; it
  describes installation rather than scopes and is not edited here

**Testable:** Yes — through `tree-sitter query`, driven by `task test:grammar`.

**Verification:** `mise exec -- task test:grammar`; the same target fails when a query file names a node
type the grammar does not have.

**Depends on:** 1

---

### Task 4: Give the VS Code grammar an executable scope test

**Behavior:** `editors/vscode/syntaxes/emod.tmLanguage.json` gets what the tree-sitter query got in
Task 1 — a test that loads the grammar with a TextMate engine and asserts, at named positions in real
emod source, which scope each token receives. The test pins the scopes the grammar assigns today, so the
two edits that follow it land with a receipt rather than a claim about what an editor would show.

**Acceptance Criteria:**
- [ ] A Taskfile target runs the scope test against `editors/vscode/syntaxes/emod.tmLanguage.json` and
      exits non-zero when an asserted scope does not match, naming the position and the scope actually
      produced
- [ ] The target is reachable from `task test` (`Taskfile.yml:54`) and `.github/workflows/ci.yml` runs
      it, so a scope regression fails CI — today nothing in CI reads the file
- [ ] The test asserts the scopes the grammar assigns today, and each assertion can fail: a comment, a
      quoted string, a keyword from the flat alternation, a quoted entity name following `model`,
      `context` and `slice`, an identifier entity name following `command`, `event`, `view`,
      `automation` and `translation`, the context name following `target context`, a field's type and a
      field's modifier
- [ ] Changing any asserted scope name in the test makes the target fail, and changing the grammar's
      scope for that token makes it fail too — checked for at least one assertion in each direction
- [ ] The test records the grammar's current treatment of a field named after a keyword: a `fields`
      block line whose name is `command` receives the keyword scope today, and the assertion states that
      as the observed behaviour rather than as the desired one, so Task 6's boundary is visible in the
      test file instead of only in this document
- [ ] The target installs its dependencies the way `test:viewer` (`Taskfile.yml:72`) does — a clean
      install inside the extension directory, not a global one — and adds no dependency to any existing
      `package.json` outside `editors/vscode/`
- [ ] `git diff` shows no change to `editors/vscode/syntaxes/emod.tmLanguage.json`: this task changes no
      scope, it only starts measuring them
- [ ] `mise exec -- task test:unit`, `test:viewer` and `test:grammar` are unaffected — this task edits
      no Go file, no viewer file and nothing under `editors/tree-sitter-emod`

**Affected Files/Modules:**
- `editors/vscode/package.json` — the devDependency group (`:38`) and a test script beside `compile`
  (`:34-37`)
- `editors/vscode/` — the test sources and their emod sample
- `Taskfile.yml` — a target beside `test:viewer` (`:72`) and `test:grammar` (`:79`), added to `test`
  (`:54`)
- `.github/workflows/ci.yml` — a step beside the existing grammar and viewer steps

**Patterns to Follow:**
- `vscode-tmgrammar-test` is the standard harness for this; version 0.1.3 is published and resolvable
  from npm. It reads the grammar through the `contributes.grammars` entry
  `editors/vscode/package.json` already declares, and its assertion files use the same
  column-marker-plus-expected-name shape as tree-sitter's highlight tests, so both halves of this story
  read alike
- `Taskfile.yml:72-77` is the shape for a Node-based test target: `dir:` into the package, clean install,
  run the script. `.github/workflows/ci.yml` already has a `Setup Node.js` step before `task test:viewer`
- `tasks/completed/us-ide-001-syntax-highlighting-textmate.md:65-67` records what this file has been
  verified by since it was written — valid JSON and a manual look in VS Code. That is the precedent this
  task departs from, and the reason to depart is that three of this story's four criteria are claims
  about this file
- `tasks/learnings.md` "An assertion whose expected value comes from the code under test is the
  recurring review finding" — name each expected scope, and confirm by mutation that it fails
- The scope names to assert are the ones the file already uses: `comment.line.number-sign.emod`,
  `string.quoted.double.emod`, `keyword.control.emod`, `entity.name.function.emod`,
  `storage.type.emod`, `storage.modifier.emod`

**Testable:** Yes — the new target is itself the test surface, and it fails on a mutated scope.

**Verification:** `mise exec -- task test` runs the new target and passes; mutating one scope in
`emod.tmLanguage.json` makes it fail.

**Depends on:** None (US-004)

---

### Task 5: Highlight the kindless trigger's quoted name in VS Code

**Behavior:** a trigger written in its new shape reads like every other named construct: the keyword
takes the keyword scope and the quoted name takes the entity-name scope, with nothing in the grammar
still expecting a kind identifier between them.

**Acceptance Criteria:**
- [ ] The scope test asserts that in `trigger "<name>" { … }` the keyword takes the keyword scope and
      the quoted name takes the entity-name scope — not the generic quoted-string scope every other
      string gets
- [ ] The pattern that matched a trigger's kind identifier followed by a quoted name
      (`editors/vscode/syntaxes/emod.tmLanguage.json:27-35`) no longer appears, and `trigger` no longer
      appears in the alternation of keywords taking an identifier entity name (`:44-51`), which US-004
      made unreachable for it
- [ ] `trigger` still takes the keyword scope where it appears with no name after it at all, so a
      half-written line in an editor does not lose its keyword colour
- [ ] The scopes assigned to `model`, `actor`, `context`, `aggregate` and `slice` and to their quoted
      names are unchanged, asserted in the same test file as the trigger — a trigger joining their
      treatment must not alter theirs
- [ ] The scopes assigned to `command`, `event`, `view`, `automation` and `translation` and to their
      identifier names are unchanged, asserted in the same test file — removing `trigger` from that
      alternation must leave the other five where they were
- [ ] The file is valid JSON, and the top-level `patterns` order (`:6-15`) is unchanged, so the
      precedence between comments, entity patterns, strings and the flat alternation still holds
- [ ] `git diff` touches `editors/vscode/syntaxes/emod.tmLanguage.json` and the scope test only

**Affected Files/Modules:**
- `editors/vscode/syntaxes/emod.tmLanguage.json` — `keyword-entity` (`:25-61`)
- The scope test added in Task 4

**Patterns to Follow:**
- The pattern for keywords taking a quoted entity name (`emod.tmLanguage.json:36-43`) is the treatment a
  kindless trigger joins; it is also what `highlights.scm:46-50` does on the tree-sitter side, and the
  two files are meant to agree — `queries/highlights.scm:7-8` says so
- `docs/proposals/triggers-and-automations-proposal.md:52-70` and `:113-121` state the new trigger shape
  and the three spellings US-004 removes
- The order of the `keyword-entity` sub-patterns decides which wins where two could match; the existing
  four are ordered most-specific first

**Testable:** Yes — through the scope test from Task 4.

**Verification:** `mise exec -- task test` passes; the trigger assertion fails when the entity-name
pattern is reverted.

**Depends on:** 4

---

### Task 6: Highlight `on` and `every` in VS Code only where their own operand follows

**Behavior:** `on` and `every` take the keyword scope where they introduce an automation's activation —
`on` before an event name, `every` before a quoted expression — and the event name takes the entity-name
scope the way a `target context` name does. In a `fields` block, a field named `on` or `every` takes no
keyword scope and reads as the plain text every other field name reads as.

**Acceptance Criteria:**
- [ ] The scope test asserts that inside an automation block, `on` before an event name and `every`
      before a quoted expression each take the keyword scope
- [ ] The scope test asserts that the event name following `on` takes the entity-name scope, matching
      how the context name following `target context` (`emod.tmLanguage.json:52-59`) is treated
- [ ] The quoted expression following `every` keeps the quoted-string scope
- [ ] The scope test asserts that a `fields` block line whose name is `on`, and one whose name is
      `every`, take no keyword scope on that name — and the same for `on` and `every` written as a
      field's type and as a field's modifier
- [ ] Adding either spelling to the flat keyword alternation (`emod.tmLanguage.json:63`) makes the
      field-name assertions fail: that alternation carries no positional context, and satisfying the
      first criterion by that route breaks the third
- [ ] An event named `Every` and an event named `On` keep the entity-name scope where they are declared,
      pinning that the treatment does not paint a capitalised identifier as a keyword — the lexer's
      keyword lookup is case-sensitive and both names are legal
- [ ] The scopes every other keyword receives are unchanged, including the observed treatment of a field
      named `command` that Task 4 pinned: this task narrows nothing and widens nothing for the
      twenty-eight keywords the story does not name
- [ ] The file is valid JSON and `git diff` touches `editors/vscode/syntaxes/emod.tmLanguage.json` and
      the scope test only

**Affected Files/Modules:**
- `editors/vscode/syntaxes/emod.tmLanguage.json` — `keyword-entity` (`:25-61`) and, if the treatment
  needs it, the top-level `patterns` order (`:6-15`)
- The scope test added in Task 4

**Patterns to Follow:**
- The four existing `keyword-entity` sub-patterns (`emod.tmLanguage.json:27-59`) are the precedent for a
  keyword scoped by what follows it rather than by the word alone; `target` (`:52-59`) is the closest,
  because it scopes a keyword and a trailing name while skipping the word between them
- The tree-sitter side reaches the same outcome structurally, because `fields_block`
  (`editors/tree-sitter-emod/grammar.js:152-166`) admits no keyword token — a regex grammar has no
  equivalent, so the context has to come from what the pattern requires around the keyword
- `tasks/learnings.md` "Keyword surfaces fan out past the lexer, parser and tree-sitter grammar" and
  "New DSL keywords must stay usable as field names" — this file is the surface both point at, and the
  second is the rule the third criterion enforces
- `tasks/completed/us-003-use-reserved-words-as-field-names.md:57-61` records the pre-existing gap for
  the other keywords and why it was left; leave it exactly where it stands
- `docs/proposals/triggers-and-automations-proposal.md:98-112` states the two activation forms and their
  operands

**Testable:** Yes — through the scope test from Task 4.

**Verification:** `mise exec -- task test` passes; the field-name assertions fail when either spelling is
added to the flat alternation.

**Depends on:** 5

---

## Summary

**Six tasks**, ordered by which surface is broken and which harness already exists.

Tasks 1 through 3 are the tree-sitter half and come first because `highlights.scm` does not compile once
US-004 lands — an impossible pattern takes the whole file with it. CI does notice (measured: an
impossible pattern flips `tree-sitter test` from exit 0 to exit 1), which is why deleting `:60` now sits
in US-004's own grammar task rather than waiting here. Task 1 therefore turns `tree-sitter test` into a
real gate over the query file — which it already knows how to be, and which `task test:grammar` already
runs in CI — and repairs `:60` only if US-004 has not already. Task 2 rests on
that harness for both halves of the keyword criterion. Task 3 covers the three query files
`tree-sitter test` does not load, and reuses Task 1's sample rather than growing a second copy of the
same shapes.

Tasks 4 through 6 are the VS Code half. Task 4 comes before either grammar edit because the harness is
larger than the edits it protects, and landing infrastructure inside a behaviour change is what
`tasks/learnings.md` records as reading like unprotected churn. Tasks 5 and 6 both write
`emod.tmLanguage.json`, so they are sequenced rather than parallel.

**Story criteria coverage:**

| Story criterion | Task |
|---|---|
| `on` and `every` highlight as keywords in the VS Code extension and the tree-sitter grammar | 2 (tree-sitter), 6 (VS Code) |
| A trigger's quoted name highlights as a name, now that no kind identifier precedes it | 1 (tree-sitter), 5 (VS Code) |
| `on` and `every` in field-name position highlight as field names, not keywords | 2 (tree-sitter), 6 (VS Code) |
| Folding, indentation, and text-object selection work on trigger and automation blocks in their new shapes | 3 |

Carried along, not stated by the story: the highlight suite and the query-compilation check that make
the tree-sitter criteria falsifiable (1, 3), and the scope test that does the same for VS Code (4). All
three exist because every prior task touching these files was written `Testable: No`, and this story
consists of nothing but claims about them.

**Deferred to other stories:** the trigger kind slot itself, its grammar rule and its corpus fixtures
(US-004); LSP hover, completion and navigation over `on` and `every` (US-009); the reference, the README
and the examples (US-011). Left where it stands: the flat TextMate alternation's positionless treatment
of the other twenty-eight keywords, and the six grammar keywords — `spec`, `given`, `when`, `then`,
`rejected`, `invariant` — missing from the tree-sitter keyword capture list.

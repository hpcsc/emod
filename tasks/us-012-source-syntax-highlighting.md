# US-012: Read source with syntax highlighting

## Contents

1. Task 1: Classify a document's text into the categories an editor colours it with
2. Task 2: Answer a source's tokens through the platform seam in every distribution
3. Task 3: Draw the source panel's text under a transparent textarea, level with its lines
4. Task 4: Colour the source panel's text by the tokens the Go core answers
5. Task 5: Keep the colours on the text while it is typed

Tasks 1 and 2 are Go. They give the frontend one operation, `tokenizeEmod`, which answers the
tokens of a text and the category of each token. Tasks 3 to 5 are the source panel. Task 3 does
not need Tasks 1 and 2, and it is the risk for criterion 5, so it can go first if the Go work
waits.

| Slice | Tasks | Exit |
|---|---|---|
| **One rule for what is what** | 1 | `lsp.GetTokens` gives every stretch of a document the category an HCL editor with `emod lsp` colours it with, and it agrees with `GetSemanticTokens` on every bundled example |
| **Reachable everywhere** | 2 | The browser runtime and the desktop runtime answer `tokenizeEmod` with those tokens |
| **A surface that can be coloured** | 3 | The panel draws its text under a transparent textarea. It looks and edits as it does today |
| **Colours** | 4 | The panel shows the text in the colours of its tokens, also for source that does not parse |
| **Typing** | 5 | The colours stay on the text while it is typed into the largest example |

---

## Story Reference

`user-stories/emod-desktop.md` → **US-012: Read source with syntax highlighting** (`:180-190`).
It depends on US-009 (delivered). US-010 and US-011 are delivered too
(`tasks/completed/us-010-jump-between-a-diagnostic-and-the-source-that-caused-it.md`,
`tasks/completed/us-011-keep-the-source-panel-and-the-diagram-in-agreement.md`). US-010
Decision 2 turned soft wrap off and placed the gutter by line number, so that this story could put
its colours behind the text without touching the gutter.

| Story criterion | Task |
|---|---|
| Keywords, identifiers, strings, numbers, and comments are visually distinguished in the source panel | 4 (from the categories Task 1 gives) |
| The same construct is highlighted as the same category the editor integrations already use | 1 holds the rule. 4 paints what the answer names and holds no rule of its own |
| Highlighting keeps up with typing on the largest bundled example model | 5 (2 gives the request, 4 keeps the long-task check green) |
| Source that fails to parse degrades to plain or partially highlighted text rather than breaking the panel | 1 answers tokens for it. 4 shows them, and shows plain text when the request fails |
| Selection, copy, paste, and undo in the source panel behave as in a plain text field | 3 |

---

## Boundaries

**Out of scope** (the story set's non-goals that bear on this story, verbatim from
`user-stories/emod-desktop.md:248-259`):

- **No replacement of existing distributions.** `emod diagram --serve` and the hosted web viewer keep working unchanged; desktop is a third option, not a migration.
- **No desktop-only diagram features.** Anything added to the canvas stays available in all three distributions.
- **No full text editor.** The source panel gains validation, navigation, and highlighting — not find-and-replace, multi-cursor, or refactoring.
- **No automated tests driving the desktop window.** Manual smoke testing of the shell, backed by the existing Go and browser suites.

**Out of scope** (added by this breakdown):

- What editors receive does not change. `emod lsp` sends the same semantic tokens. It does not
  start to send lexical tokens, and no editor grammar comes back into the repository (commit
  79a28f8 removed them).
- A name where it is used, not where it is declared (a `reads`, `subscribes`, `given`, `when`,
  `then`, `actor` or flow reference). `emod lsp` colours declarations only
  (`internal/lsp/semantictokens.go:77-89`), and the panel does the same. A reference gets the
  category HCL gives it: `variable`, or `function` for a call.
- A theme. The palette is the app's own and light only. It does not follow the user's editor
  theme. There is no dark palette and no setting.
- Editor conveniences: line numbers, bracket matching, auto-closing quotes or brackets, folding,
  indentation help. The story's non-goal already excludes find-and-replace, multi-cursor and
  refactoring.
- A grammar for diagram JSON. JSON pasted into the panel gets what HCL's lexer reads in it (its
  strings, numbers and punctuation), and no construct names.
- The same colours as tree-sitter in broken text. Where HCL's parser cannot recover, an identifier
  it did not reach takes the query's fallback, `variable`. Tree-sitter has its own error recovery
  and can colour the same broken text differently.
- Soft wrap. It stays off (US-010 Decision 2).
- The model that `emod diagram --serve <file>` serves. The initial state carries only the diagram
  (`internal/frontend/static/viewer.js:1097-1100`), so the panel holds no source to colour. US-010
  recorded the same limit.
- Performance past the largest bundled example. Criterion 3 names that model, and Task 5
  measures that model.

**Deferred:**

- The length `emod lsp` gives a construct name that holds non-ASCII text. `GetSemanticTokens`
  measures the name with `len(name)`, which counts bytes (`internal/lsp/semantictokens.go:65-77`).
  LSP counts UTF-16 units, so an editor colours such a name past its end. `GetTokens` measures
  names from the lexer's range, so the app is correct. To fix the editor's answer changes a
  contract that editors outside the repository read, so that fix needs its own story.

---

## Decisions

### Decision 1 — all three distributions get the change

The frontend is one copy (`internal/frontend/static`). `task build:web` and `task build:desktop`
copy all of it, and `internal/frontend/embed.go` embeds all of it. At US-009 the user chose all
three distributions for a source-panel change, and cited the non-goal that canvas features stay
available everywhere (`tasks/learnings.md:1348-1353`). This story does the same. The browser
runtime answers `tokenizeEmod` from the WebAssembly worker, which serves the web viewer and
`emod diagram --serve`. The desktop runtime answers it from `ModelService`.

**Goal tension, for the user:** the story set's goal says that the web viewer and `--serve`
"keep working exactly as they do today". The visible change is that their source panel shows
colours. Under US-009's reading ("nothing stops working"), this change is allowed: typing,
selection, copy, paste, undo, Render, drop, Open, Save and Export all work as before.

### Decision 2 — the category rule lives once, in Go, and the frontend only paints

An editor colours an `.emod` file in two layers. Its HCL grammar colours the lexical and the
structural parts (README "Editor Setup", `README.md:296-356`). `emod lsp` adds semantic tokens for
the names of declared constructs (`internal/lsp/semantictokens.go:90-156`). The README's Neovim
setup names the grammar exactly: `:TSInstall hcl`, the tree-sitter HCL parser, whose colours come
from nvim-treesitter's `queries/hcl/highlights.scm`. Helix and Zed use the same parser. Neovim
names a semantic token's highlight group `lsp.type.<type>`, and gives it priority over the
grammar's capture.

| Option | Where the rule is | Problem |
|---|---|---|
| A JavaScript HCL tokenizer, and construct names from the parse answer | Lexical rules in JS, names in Go | It writes the grammar a second time. `docs/architecture.md:325-331` says that "nothing in this repository spells the grammar a second time". The rule is in two places |
| Construct names only, attached to the debounced parse answer | Go, but only for names | Strings, comments and numbers stay uncoloured. All colour waits 300 ms after typing stops |
| **`lsp.GetTokens` classifies the whole text with HCL's own lexer and parser, and puts the semantic tokens on top. The frontend asks for it on each change** | **One function in Go** | A new seam operation (US-011 added `applyDiagram` the same way, commit ec9a239), and one request for each change of the text (Decision 3) |

**Chosen: the third option.** Each HCL element gets the capture that the nvim-treesitter HCL query
gives it. A declared construct's name gets `lsp.type.` followed by the type that `emod lsp`'s
legend gives it (`internal/lsp/semantictokens.go:31-43`). `GetTokens` and `GetSemanticTokens` read
names through one walk of the model, so the panel and an editor cannot disagree about a name. The
frontend has no table from construct to category: it paints the category that the answer names.
`internal/lsp` already lexes HCL (`internal/lsp/model.go:18-37`) and owns the semantic walk, so
the function goes there, and no other package needs to export the walk.

### Decision 3 — tokens are asked for on each change of the text

Criterion 3 asks that the colours keep up with typing. An editor colours lexically on each
keystroke, and sends semantic tokens after a short pause. Colours that waited for the 300 ms
revalidation pause (`internal/frontend/static/config.js:40`) would stay behind the typing for as
long as the typing continues. So the viewer asks `tokenizeEmod` on each change of the panel's text,
and does not wait for the pause.

**Measured cost:** the whole pipeline (parse, validate, lint, diagram export) over
`examples/all_patterns.emod` takes 5.6–7.4 ms (median 6.7 ms) in WebAssembly under V8. The probe
called the built module's `parseEmod` ten times under Node. Tokenizing does less work than that. The
desktop runtime runs Go natively. One request for each change therefore fits easily in the time
between keys. Task 5 holds at most one request open, so a backlog cannot grow.

**Between a change and its answer**, the panel shows the newest answer it has, moved across the
edits made since that answer (Task 5). Only the stretch that is edited waits for the next answer.
This is different from the gutter, which keeps its line numbers between revalidations (US-010
Decision 4). A gutter entry marks a line. A colour sits on characters, so it must move with them or
be removed.

### Decision 4 — the colours are a copy of the text, drawn under a transparent textarea

The textarea stays the only surface that edits the text. That keeps selection, copy, paste, undo
and redo native (criterion 5). The textarea draws no text of its own, but it keeps its caret and its
selection. Under it, the source colours draw the same text with the same font, padding, border,
line height and tab size. They read the shared CSS variables on `#source-area`, follow the
textarea's scroll on both axes, and sync again from the `ResizeObserver` that the gutter already
uses (`internal/frontend/static/source-panel.js:5-25`, `tasks/learnings.md:1362-1367`). US-010
Decision 2 expected this design.

### Decision 5 — offsets count UTF-16 units of the text that was sent

A token's start and end count UTF-16 code units of the source that the viewer sent. That source is
the textarea's value, with its line endings already changed to LF (`tasks/learnings.md:1015-1020`).
The frontend can therefore cut the panel's text at the offsets with no conversion. Go converts
from HCL's byte ranges. The diagnostic columns that HCL counts in grapheme clusters
(`tasks/learnings.md:1376-1381`) and the byte length of the LSP's names do not cross this boundary.

---

## Codebase Context

**The source panel.** `#source-input` is a `wrap="off"` textarea with `white-space: pre` inside
`#source-area` (`internal/frontend/static/viewer.html:683-735`, `:1403-1406`). The gutter places
each entry from `--source-line-height`, `--source-padding` and `--source-border`, minus
`--source-scroll`, which `followScroll` copies from the textarea's `scrollTop`. A `ResizeObserver`
syncs it again after the collapsed data panel opens (`internal/frontend/static/source-panel.js:5-25`).
`e2e-viewer/tests/source-panel.spec.js:174-185` (`gutterOffsetFrom`) and `:272-310`
(`caretInView`) measure a place on screen from the panel's own computed style. The source colours
need the same alignment checks.

**What changes the panel's text.** Typing, paste, cut, undo, redo and dropped text fire `input`
(`internal/frontend/static/viewer.js:436-444`). The viewer itself writes the value in three places,
and these fire no event (`tasks/learnings.md:1021-1026`): `renderPanelSource` for an opened,
dropped or reloaded model (`viewer.js:257-260`), the same function when it puts the previous text
back after a refused open (`viewer.js:329-333`), and `writeAppliedSource` for a diagram edit that
rewrites the source (`viewer.js:580-584`). `ui.js:688` and `:820` write the canvas's field editors,
not the panel.

**The platform seam.** `platform.js:7` re-exports the contract. `platform.browser.js:49-89`
answers it through `callWasm` to the worker, and rejects calls before the module is ready.
`internal/frontend/desktop/platform.desktop.js:16-40` answers it through `ModelService`. Commit
ec9a239 ("Answer an applied diagram edit through the seam") threaded `applyDiagram` through every
layer: `pipeline.ApplyDiagramJSON`, `cmd/emod-wasm/main.go`, `ModelService.ApplyDiagram`, the
three platform files, `tests/bindings-stub.js`, and `docs/wasm-architecture.md`. `tokenizeEmod`
repeats that change. The guards already in place cover a new operation:
`internal/pipeline/pipeline_test.go:56-66` (both runtimes send the `Request` keys),
`:160-176` (each key the viewer reads off an answer is one the pipeline writes), `:455-468`
(the WASM page names each registered function), and `internal/desktop/binding_names_test.go:22-63`
(each `ModelService` method the frontend calls is exported).

**The editor's rule today.** `GetSemanticTokens` walks contexts, aggregates, the slices' commands,
events and views, and actors, and puts each name at the column after its opening quote
(`internal/lsp/semantictokens.go:90-156`). `oracle.Parse` is best-effort (`internal/oracle/oracle.go:13-19`).
A probe through the built WebAssembly module confirms that `examples/all_patterns.emod` with an
unclosed context appended still yields 36 diagram nodes beside "unclosed configuration block". So
names above an error survive. Text that is not HCL at all yields none
(`internal/lsp/semantictokens_test.go:267-271`). The parser walks `hclsyntax` blocks, attributes and
expressions (`internal/parser/parser.go:17-35`, `:706-752`). `internal/lsp` builds for `js/wasm`
(checked with `GOOS=js GOARCH=wasm go build ./internal/lsp ./internal/pipeline`).

**Async ordering.** The viewer numbers renders and deliveries, because answers can land out of
order (`viewer.js:203-247`). The desktop platform queues `SetModified` calls, because each binding
call is served by its own goroutine (`platform.desktop.js:187-211`).

**Words.** In this repository *highlight* means the diagram's emphasis of nodes
(`ui.js:769-790`, `store.interaction.highlighted`, `highlightFill` in `config.js:47-52`), so the
colouring code does not use the word. *Copy* is taken by event copies (`model.js:223-226`), and
*kind* by prose kinds (`config.js:31-35`) and by `constructDecl.kind` in `internal/lsp/model.go`.
*Token* and *category* are free in the frontend. In Go, `hclsyntax.Token` is HCL's lexer token, and
`tokenEntry` in `internal/lsp/semantictokens.go` is a semantic token before encoding. See the
`words` in the design file.

---

## Tasks

### Task 1: Classify a document's text into the categories an editor colours it with

**Behavior:** `lsp.GetTokens` answers, for an `.emod` document, the stretches of its text and the
category of each one. The category is what an editor colours that stretch with when it runs the
tree-sitter HCL grammar and `emod lsp`. Construct names come from the same walk that
`GetSemanticTokens` uses.

**Acceptance Criteria:**
- [x] Each HCL element takes the capture that nvim-treesitter's `queries/hcl/highlights.scm` gives it. When two patterns match, the later pattern wins, as in the query. The suite exercises every pattern of that query that HCL syntax can produce. A comment beside the suite names the query's revision that the tests were derived from
- [x] An element that the query gives no capture (the query's `@none` and whitespace) has no token
- [x] The name of each declared construct takes `lsp.type.` followed by the type that `GetSemanticTokens` gives it. The token covers exactly the characters between the label's quotes, and the quotes keep their string category
- [x] For every bundled example (`examples/*.emod`, the `_test.emod` model too), the construct-name tokens are exactly the names `GetSemanticTokens` reports, on the same lines, at the same starting characters, and with the same types. Neither function reports a name that the other does not
- [x] `GetTokens` and `GetSemanticTokens` read construct names through one walk of the model. No new walk of the model is added
- [x] `GetSemanticTokens` keeps its contract: the existing semantic-token suite passes, and its expectations are not changed
- [x] Tokens come in document order and never overlap. Start and end count UTF-16 code units of the document. On a line with a combining mark, a ZWJ emoji and a character outside the Basic Multilingual Plane before a token, the token's offsets select exactly that token's text
- [x] A document that does not parse still answers the tokens of everything HCL's lexer reads, and the names of the constructs the parser recovered before the error. Text that is not HCL at all answers tokens without a panic, and they do not overlap. Examples: plain words, an unterminated string, heredoc or block comment, and invalid UTF-8
- [x] The empty document answers no tokens
- [x] `docs/architecture.md`'s LSP paragraph (`:206-211`) names `GetTokens` with the package's other pure functions, and says that the viewer is its caller

**Affected Files/Modules:**
- `internal/lsp/token.go` — new: the `Token` type and `GetTokens`
- `internal/lsp/token_test.go` — new: `TestGetTokens`
- `internal/lsp/semantictokens.go` — the construct-name walk, shared by both functions
- `internal/lsp/model.go` — only if the shared walk moves there, beside the package's other walks
- `docs/architecture.md` — the LSP paragraph

**Patterns to Follow:**
- `internal/lsp/semantictokens.go:77-156` for the construct-name walk to share
- `internal/lsp/model.go:18-46` for lexing with `hclsyntax.LexConfig` in this package
- `internal/parser/parser.go:17-35` and `:706-752` for parsing with `hclsyntax.ParseConfig` and switching over its expression types
- `internal/lsp/semantictokens_test.go:39-100` and `internal/lsp/cursor_test.go:14` for the test style and for finding a position by its text
- `internal/pipeline/pipeline_test.go:636-650` for walking every bundled example
- `tasks/learnings.md:431-435` (one model walk in `internal/lsp`), `:1376-1381` (columns count grapheme clusters), `:496-500` (a suite that pins another tool's output needs a negative control that changes the input)

**Testable:** Yes — through `lsp.GetTokens` and `lsp.GetSemanticTokens`.

**Certainty:** low — nothing in this repository maps HCL syntax to an editor's highlight captures, and the mapping comes from nvim-treesitter's HCL query, which is not in the repository.

**Blast radius:** high — it shares `GetSemanticTokens`'s walk, and `emod lsp`'s semanticTokens answer is a contract that editors outside this repository read.

**Verification:** `go test -tags unit ./internal/lsp/...` and `task test:unit` pass. `GOOS=js GOARCH=wasm go build ./internal/lsp` succeeds.

**Depends on:** None

---

### Task 2: Answer a source's tokens through the platform seam in every distribution

**Behavior:** The platform contract gains `tokenizeEmod(source)`. It answers the tokens that
`lsp.GetTokens` gives the source. The browser runtime answers it from the WebAssembly worker
(the web viewer and `emod diagram --serve`). The desktop runtime answers it from `ModelService`.

**Acceptance Criteria:**
- [x] `tokenizeEmod(source)` is part of the contract in `internal/frontend/static/platform.js`. Both runtimes resolve with the same decoded `{tokens}` document, and each token carries `start`, `end` and `category`
- [x] Both runtimes send the source in the `{source, filename}` envelope that `pipeline.Request` reads. The keys guard at `internal/pipeline/pipeline_test.go:56-66` covers `tokenizeEmod`
- [x] An error envelope rejects with its reason in both runtimes. The browser runtime rejects with "WASM not ready yet" before the module is ready, as its other operations do
- [x] The Go entry point answers the error envelope for a malformed envelope and for an empty source, and never panics
- [x] `ModelService.TokenizeEmod` answers what the pipeline answers for the same envelope
- [x] The Build step of `docs/wasm-architecture.md` names `tokenizeEmod`, so the documentation guard passes. Every `ModelService` method that `platform.desktop.js` calls is exported by Go (the existing binding-name guard passes)

**Affected Files/Modules:**
- `internal/pipeline/pipeline.go` — `TokenizeEmodJSON`
- `internal/pipeline/pipeline_test.go` — the answer, and the envelope-keys guard for `tokenizeEmod`
- `cmd/emod-wasm/main.go` — registers `tokenizeEmod`
- `internal/desktop/service.go`, `internal/desktop/service_test.go` — `ModelService.TokenizeEmod`
- `internal/frontend/static/platform.js` — the contract's export list
- `internal/frontend/static/platform.browser.js`, `internal/frontend/desktop/platform.desktop.js` — `tokenizeEmod`
- `internal/frontend/tests/platform.browser.test.js`, `internal/frontend/tests/platform.desktop.test.js`, `internal/frontend/tests/bindings-stub.js`
- `docs/wasm-architecture.md` — the Build step and the browser flow

**Patterns to Follow:**
- `internal/pipeline/pipeline.go:94-123` and `:193-215` for a JSON entry point and its error envelope
- `cmd/emod-wasm/main.go:11-46` and `internal/desktop/service.go:32-43` for registering and binding an entry point
- `internal/frontend/static/platform.browser.js:78-89` and `internal/frontend/desktop/platform.desktop.js:32-40` for one seam operation in each runtime
- `internal/frontend/tests/platform.browser.test.js:164-187`, `internal/frontend/tests/platform.desktop.test.js:127-153` and `internal/desktop/service_test.go:112-133` for their tests
- `internal/pipeline/pipeline_test.go:446-468` for the envelope-keys and documentation guards
- `tasks/learnings.md:1093-1098` (a seam operation touches the same files in every task, so fold by mechanism) and `:961-966` (a Go symbol that only generated bindings call reads as dead code)

**Testable:** Yes — through `pipeline.TokenizeEmodJSON`, `ModelService.TokenizeEmod`, and each runtime's exported `tokenizeEmod`.

**Certainty:** high — it repeats commit ec9a239, which threaded `applyDiagram` through the same files.

**Blast radius:** low — the seam is internal to the repository's own distributions, and this adds an operation without changing any existing one.

**Verification:** `task test:unit` and `task test:viewer` pass. `task build:wasm` succeeds.

**Depends on:** Task 1

---

### Task 3: Draw the source panel's text under a transparent textarea, level with its lines

**Behavior:** The source panel draws its text in the source colours, under the textarea that edits
it. The textarea draws no text of its own, but keeps its caret and its selection. The panel looks,
scrolls and edits as it does today. No colours yet: every character is drawn in the plain text
colour.

**Acceptance Criteria:**
- [x] The text the user sees is drawn by the source colours. The textarea's own text is transparent, and its caret is visible where the textarea puts it
- [x] The source colours read exactly the textarea's value after every change to that value. The rule covers typing, paste, cut, undo, redo, text dropped onto the panel, and every assignment to the source panel's value in `internal/frontend/static`. Today these assignments are an opened, dropped or reloaded model, a refused open that puts the previous text back, and a diagram edit that rewrites the source
- [x] Each character sits within 1 px, on both axes, of where the textarea draws it. This holds on the largest bundled example: at the top, after a scroll to its last line, after a scroll sideways to the end of its longest line, after the panel's height is resized, and after the data panel is collapsed and opened again with the panel scrolled. It also holds on a line with a tab, a combining mark, a ZWJ emoji and a CJK character
- [x] The placeholder is readable when the panel is empty. Selected text stays readable through the selection: the selection's computed colour is translucent, in Chromium and in WebKit
- [x] Selection, copy, paste, undo and redo work as in a plain textarea. In a real browser, text selected with the keyboard or the mouse copies as exactly the selected source, with no markup. Pasted text lands at the caret. Undo after typing and after a paste restores the previous text one step at a time, and redo applies it again. The colouring never writes the textarea's value
- [x] Assistive technology reads the source once: the source colours are hidden from it, and the textarea stays the input
- [x] The gutter's entries stay level with their lines (the existing gutter specs pass). The caret line's message still neither moves nor resizes the text. Every main-thread task stays under 50 ms while the largest example is typed into (the existing check)

**Affected Files/Modules:**
- `internal/frontend/static/viewer.html` — the source colours' element in `#source-area`, and the CSS that stacks it under a transparent textarea
- `internal/frontend/static/source-panel.js` — `showText`, and scroll following on both axes
- `internal/frontend/static/viewer.js` — the DOM reference, and `showText` after each change to the panel's value
- `internal/frontend/static/store.js` — `dom.sourceColours`
- `internal/frontend/tests/source-panel.test.js`, `internal/frontend/tests/viewer.test.js`
- `e2e-viewer/tests/source-panel.spec.js` — alignment and editing in a real browser

**Patterns to Follow:**
- `internal/frontend/static/source-panel.js:5-25` and `internal/frontend/static/viewer.html:683-735` for a layer level with the panel's lines, built from the shared CSS variables
- `e2e-viewer/tests/source-panel.spec.js:174-253` for the alignment check and the collapsed-panel case, `:255-268` for sideways scroll, and `:272-310` for measuring a column on screen
- `internal/frontend/static/viewer.js:252-262`, `:325-333` and `:580-590` for the writes to the panel's value that fire no event
- `internal/frontend/tests/source-panel.test.js:5-18` for setting up the panel in vitest
- `internal/frontend/tests/viewer.test.js:5616-5640` for the guard that every id `init` reads exists in `viewer.html`
- `tasks/learnings.md:1362-1367` (a layer level with the lines), `:1355-1360` (run once under Playwright's WebKit), `:1313-1318` (open the data panel before filling), `:757-762` (look at the rendered page), `:1129-1134` (read CSS back out of the running webview)

**Testable:** Yes — through `SourcePanel.showText`, the viewer's `init`, and the page in a real browser.

**Certainty:** medium — the gutter is the precedent for a layer level with the lines (`source-panel.js:5-25`), but no instance draws the whole text, follows the horizontal scroll, or must keep a transparent textarea's caret and selection readable.

**Blast radius:** low — a display change in the shared frontend. The textarea's value, which Save reads through `sourceToSave`, is not touched.

**Verification:** `task test:viewer` and `task test:e2e:viewer` pass. Run the source-panel spec once under Playwright's WebKit, with a temporary config that adds a `devices['Desktop Safari']` project to `playwright.config.js`. Build with `task build:web`, serve it, then look at the page and select, copy, paste and undo in it. Build with `task build:desktop` and do the same in the desktop window.

**Depends on:** None

---

### Task 4: Colour the source panel's text by the tokens the Go core answers

**Behavior:** After each change to the panel's text, the viewer asks `tokenizeEmod` for the new
text. The source colours then paint each token in the colour of the category that its answer
names. Keywords, identifiers, strings, numbers, comments and construct names show different
colours. Source that does not parse is coloured as far as the answer goes. When there is no answer,
the text is plain.

**Acceptance Criteria:**
- [ ] Each change to the panel's text, of every kind that Task 3 keeps the source colours in step with, asks `tokenizeEmod` for the new text once the Go core is ready. Text that arrives before the core is ready is coloured when the core is ready
- [ ] Keywords, identifiers, strings, numbers and comments are visually distinguished. In a real browser on the largest example, a block type, a bare identifier, a string, a number and a comment each have a computed colour that differs from the other four. Each token's computed colour is the colour that its category's CSS rule names
- [ ] Each category's colour contrasts at least 4.5:1 with the panel's background
- [ ] In the web bundle, with the real pipeline, a declared construct's name shows the colour of its `lsp.type` category, not the string colour. Its quotes, and the label of a block that `emod lsp` does not colour (a `slice` or `spec` label), show the string colour
- [ ] The frontend holds no category rule. A token gets the colour of the category its answer names, also where the text suggests a different category. A category that has no colour of its own paints as plain text
- [ ] An answer paints only the text it was asked for. An answer for text that has changed since never colours the new text
- [ ] Source that fails to parse degrades. With an unclosed context at the end of the largest example, its comments, strings and numbers keep their colours, the construct names above the error keep theirs, and the panel still edits. When `tokenizeEmod` refuses or fails (an error envelope, or a WebAssembly module that did not load), the panel shows its text plain, still edits, and raises no uncaught error
- [ ] An empty panel sends no request and shows its placeholder
- [ ] Each key that the frontend reads off a tokenize answer and off a token is a key the Go side writes. A guard in `internal/pipeline/pipeline_test.go` checks this
- [ ] `docs/architecture.md`'s viewer section says where the source panel's colours come from
- [ ] Task 3's alignment checks pass with the colours painted, and every main-thread task stays under 50 ms while the largest example is typed into

**Affected Files/Modules:**
- `internal/frontend/static/viewer.js` — asks `tokenizeEmod` after each change to the panel's text, and keeps the answer
- `internal/frontend/static/source-panel.js` — paints tokens in `showText`
- `internal/frontend/static/store.js` — the `tokens` field
- `internal/frontend/static/viewer.html` — one colour rule for each category
- `internal/frontend/tests/viewer.test.js` — the platform mock gains `tokenizeEmod`, and tests for the request and the answer
- `internal/frontend/tests/source-panel.test.js`
- `e2e-viewer/tests/source-panel.spec.js` — computed colours, construct names, and the broken source
- `internal/pipeline/pipeline_test.go` — the wire-key guard
- `docs/architecture.md` — the viewer section

**Patterns to Follow:**
- `internal/frontend/static/viewer.js:252-345` for a request and an answer about the panel's text, and `:487-505` for a seam call that waits for `ready`
- `e2e-viewer/tests/source-panel.spec.js:204-214` for comparing computed colours in a real browser, and `:7` for the unparseable source
- `internal/frontend/tests/renderer.test.js:16-25` and `:1510-1523` for reading a colour out of the stylesheet, so that a failure names the rule that changed
- `internal/pipeline/pipeline_test.go:160-176` for the guard that the keys the frontend reads are keys Go writes
- `internal/frontend/tests/viewer.test.js:37-81` for the platform mock
- `task:1` for the categories, and `task:3` for the source colours to paint
- `tasks/learnings.md:1369-1374` (a colour criterion needs a computed-style assertion), `:426-430` (a leaf must fail only for the paint it names), `:997-1002` (a parity guard belongs to the task that adds the caller), `:943-948` (concurrent dynamic imports and the mock)

**Testable:** Yes — through the viewer's `init` with the platform mocked, and through the page against the real pipeline.

**Certainty:** medium — `renderPanelSource` is the precedent for a request and an answer about the panel's text (`viewer.js:252-345`), but here the request follows each change of the text, also the writes that fire no `input` event, where revalidation follows only typing.

**Blast radius:** low — a display change in the shared frontend, and a test-only guard in Go.

**Verification:** `task test:viewer`, `task test:unit` and `task test:e2e:viewer` pass. Run the colour spec once under Playwright's WebKit. Build with `task build:web`, serve it, and look at `examples/all_patterns.emod` in the panel. Build with `task build:desktop` and look at the same model in the desktop window.

**Depends on:** Task 2, Task 3

---

### Task 5: Keep the colours on the text while it is typed

**Behavior:** While the answer for the current text is still to come, the panel shows the newest
answer it has, moved across the edits made since that answer. The colours stay on their own
characters, and typed text takes the colour of the token it is typed into. At most one request is
open, and the panel's newest text is always the next text asked for. On the largest example the
colours follow the typing and do not slow it.

**Acceptance Criteria:**
- [ ] While the answer for the current text is still to come, the panel shows the newest answer it holds, moved across the edits made since. A token completely outside the edited stretch keeps its colour on its own characters. Text typed inside a token takes that token's colour. No colour lands on a character that its answer did not classify
- [ ] At most one `tokenizeEmod` request is open at a time. When that request settles, the panel's text at that moment is asked for next, if it is not the text that was answered. An answer for older text never replaces the colours of an answer for newer text
- [ ] In the web bundle, typing into the largest bundled example at 20 ms per key: every main-thread task stays under 50 ms. Within 100 ms of the last key, the colours describe the final text exactly: each token on screen is a token that `tokenizeEmod` gives that text. The colours do not wait for the revalidation pause
- [ ] A new text that arrives while a request is open (a model opened, or a diagram edit that rewrites the source) is coloured from its own answer as soon as that answer arrives

**Affected Files/Modules:**
- `internal/frontend/static/source-panel.js` — moves the tokens of an answer across the edits made since it was asked for
- `internal/frontend/static/viewer.js` — one request open at a time, with the newest text asked for next
- `internal/frontend/tests/source-panel.test.js`, `internal/frontend/tests/viewer.test.js`
- `e2e-viewer/tests/source-panel.spec.js` — the timing on the largest example

**Patterns to Follow:**
- `internal/frontend/static/viewer.js:203-247` for numbering answers that can land out of order
- `internal/frontend/desktop/platform.desktop.js:187-211` for keeping one call open at a time
- `e2e-viewer/tests/source-panel.spec.js:95-124` for typing into the largest example and measuring long tasks
- `internal/frontend/tests/viewer.test.js:290-312` for clearing the timers a test starts
- `task:4` for the request and the painting this task changes
- `tasks/learnings.md:1306-1311` (a timer outlives the vitest test that started it, and so does an animation frame), `:1341-1346` (plan audit rounds for work that no gesture starts, and list each claim it competes with: open, Render, host failure, save, diagram edit)

**Testable:** Yes — through the viewer's `init` with a platform mock that holds its answers, and through the page in a real browser.

**Certainty:** low — nothing here moves positioned data across an edit. The gutter keeps its lines on purpose (US-010 Decision 4), so this task decides how a token follows the characters it classified.

**Blast radius:** low — a display change in the shared frontend.

**Verification:** `task test:viewer` and `task test:e2e:viewer` pass. Build with `task build:desktop` and type into `examples/all_patterns.emod` in the desktop window.

**Depends on:** Task 4

---

## Summary

- **Five tasks.** Two are Go (the rule, and the seam that reaches it). Three are JavaScript (a
  surface that can be coloured, the colours, and typing).
- **Ordering.** Dependencies first, then risk. Task 1 is the least certain task and holds the rule
  that every other task relies on, so it goes first. Task 2 exposes the rule, and repeats a known
  pattern. Task 3 needs neither Go task and holds the risk for criterion 5, so it can come first if
  the Go work waits. Task 4 joins the two halves. Task 5 refines the typing behaviour after
  correctness is in place.
- **Coverage.** All five story criteria are covered (see the table under Story Reference). Nothing
  is deferred from the story. The one deferral (the byte length the LSP gives non-ASCII names) is in
  editors, which this story does not change.
- **Assessments.** Low certainty: Task 1 (the mapping comes from an upstream query that is not in
  the repository) and Task 5 (no precedent for moving tokens across an edit). High blast radius:
  Task 1 (`emod lsp`'s semantic tokens are an external contract). Task 2 is high certainty, because
  commit ec9a239 is an exact precedent. Tasks 4 and 5 add work that typing starts, not a click, on
  top of answers that can arrive out of order. Learning `tasks/learnings.md:1341-1346` says to
  expect two or three audit rounds on such work.
- **Numbers the caller can overrule:** the 100 ms bound in Task 5 and the 4.5:1 contrast in Task 4
  are this breakdown's thresholds. The story does not state them. The measured cost behind
  Decision 3 is about 7 ms for the whole pipeline on the largest example.

# US-010: Jump between a diagnostic and the source that caused it

## Contents

1. Task 1: Show each diagnostic's line and column, or that it applies to the whole file
2. Task 2: Mark the lines that carry diagnostics in a gutter beside the source panel
3. Task 3: Move the source panel's caret to a diagnostic when it is clicked
4. Task 4: Show the diagnostics of the caret's line under the source panel

Task 1 changes what the diagnostics panel says. Tasks 2 to 4 change the source panel. Task 2 is first of
those because it is the least certain, and because it stops the panel soft-wrapping. That change sets
the geometry that Task 3 scrolls and Task 4 sits under.

| Slice | Tasks | Exit |
|---|---|---|
| **Where a diagnostic is** | 1 | The diagnostics panel shows each diagnostic's line and column, or says it applies to the whole file |
| **Which lines carry one** | 2 | The source panel marks each such line with its severity |
| **Getting there** | 3 | A click on a diagnostic puts the caret on its location and keeps it in view. The diagram highlight still works |
| **Reading it there** | 4 | The caret's line shows its diagnostics under the source panel |

---

## Story Reference

`user-stories/emod-desktop.md` → **US-010: Jump between a diagnostic and the source that caused it**
(`:152-163`). It depends on US-009 (`:137-150`), which is delivered
(`tasks/completed/us-009-edit-source-in-the-app-and-see-diagnostics-as-you-type.md`).

US-011 (`:165-178`) reconciles diagram edits with source edits. US-012 (`:180-190`) highlights the
source. This breakdown stops short of both.

| Story criterion | Task |
|---|---|
| Each diagnostic in the panel shows its line, and its column where one is known | 1 |
| Clicking a diagnostic moves the source panel's caret to that position and scrolls it into view | 3 |
| Clicking a diagnostic still highlights the matching diagram element when one exists, as it does today | 3 (a regression guard: the highlight exists today, at `ui.js:305-351`) |
| Lines carrying a diagnostic are marked in the source panel, visually distinguishing error, warning, and info severities | 2 |
| Putting the caret on a marked line surfaces that diagnostic's message | 4 |
| A diagnostic with no position reports that it applies to the whole file rather than jumping to line 1 | 1 reports it. 3 does not jump |

---

## Boundaries

**Out of scope** (the story set's non-goals that bear on this story, verbatim from
`user-stories/emod-desktop.md:248-259`):

- **No replacement of existing distributions.** `emod diagram --serve` and the hosted web viewer keep working unchanged; desktop is a third option, not a migration.
- **No desktop-only diagram features.** Anything added to the canvas stays available in all three distributions.
- **No full text editor.** The source panel gains validation, navigation, and highlighting — not find-and-replace, multi-cursor, or refactoring.
- **No automated tests driving the desktop window.** Manual smoke testing of the shell, backed by the existing Go and browser suites.

**Out of scope** (added by this breakdown):

- Syntax highlighting is US-012. This story does not copy the panel's text into a second layer, does
  not tint text and does not colour tokens. See Decision 2.
- Line numbers in the gutter. The gutter shows only the lines that carry a diagnostic.
- Moving from the gutter to the diagnostics panel. Clicking a gutter entry and selecting a diagnostic
  from the source are not part of this story. The story asks for diagnostic → source, and for the
  caret → the message.
- Keyboard focus on the diagnostics panel's items. Today the items are reached with the mouse, and
  that does not change.
- Moving the gutter entries when lines are inserted or deleted between two revalidations. See
  Decision 4.
- Diagnostics inside pasted diagram JSON. They refer to the `.emod` file that produced the JSON, not
  to the JSON in the panel. In practice none arrive: `model.js:148-163` reads diagnostics only from a
  JSON document that has `nodes` or `model` at the top. The CLI writes its diagnostics in a
  `{diagnostics, diagram}` envelope (`internal/export/diagram.go:88-97`), which that check does not
  accept. No special handling is added.
- The model that `emod diagram --serve <file>` serves. The initial state carries only the diagram
  (`viewer.js:821`), so the panel never gets that model's source and the panel lists no diagnostics
  for it. This story applies to source that is typed, pasted, dropped or opened into the panel.
  US-009 recorded the same limit.

**Deferred:**

- Correct columns for flow entries that come after non-ASCII text on the same line.
  `internal/parser/parser.go:915-916` gets a flow entry's name column from a byte index. Every other
  diagnostic column is HCL's count of user-perceived characters. On a flow line with non-ASCII text
  before the name, the caret lands to the right of the name, but on the correct line. The fix belongs
  in `internal/parser`, and it also changes what the LSP reports. The caret placement in this story
  uses HCL's unit (Decision 5), so the viewer needs no change when the parser is fixed.

---

## Decisions

### Decision 1 — all three distributions get the change

The change ships in the desktop app, `emod diagram --serve` and the web viewer together. The frontend
is one copy (`internal/frontend/static`). `task build:web` and `task build:desktop` both copy it whole,
and `internal/frontend/embed.go` embeds it whole. A new module file reaches all three distributions
with no extra step. US-009 set the precedent (`tasks/learnings.md:1364-1369`): at match-request the
user chose all three distributions and cited the non-goal that canvas features stay available
everywhere. This story is mostly about the source panel, not the canvas. Criterion 3 does touch the
canvas, and a desktop-only switch would need a platform branch in shared code.

**Goal tension, for the user:** the story set's goal says the web viewer and `--serve` "keep working
exactly as they do today". The visible changes are: diagnostic locations gain a column, a gutter
appears beside the source panel, a message line appears under it, and the source panel stops
soft-wrapping long lines. US-009's reading of the goal was "nothing stops working". Under that
reading this is the same as US-009: Render, drop, Open, Save and Export all keep working. If the user
reads the goal more strictly, the place to say so is the plan gate, before Task 2.

### Decision 2 — the lines are marked in a gutter beside a source panel that does not wrap

A textarea cannot style its own lines. There are three ways to mark a line:

| Option | What it builds | Effect on US-012 |
|---|---|---|
| Tint the line, with a layer behind a transparent textarea that repeats its text | A copy of the panel's text, aligned under it | This is the highlighting overlay US-012 builds. It would be built early, or built twice |
| A gutter, with soft wrap kept, where a hidden copy of the text measures where each wrapped line falls | A second copy of the text, used only to measure | US-012 then has two copies of the text to keep aligned |
| **A gutter, with soft wrap turned off. Entries are placed by line number and moved with the panel's vertical scroll** | No copy of the text. An entry depends only on its line number, the row height and the scroll offset | US-012 puts its overlay behind the text and does not touch the gutter |

**Chosen: the gutter with soft wrap turned off.** It is the only option that does not copy the text,
so it is not part of a highlighting engine and US-012 has nothing to remove. Turning wrap off is what
lets a line number give an entry's position without any measuring. It also means US-012's overlay
does not have to reproduce the textarea's wrapping. The cost: long lines (`examples/all_patterns.emod`
has 12 over 100 characters, mostly descriptions) scroll horizontally instead of wrapping. The caller
can overrule this choice.

### Decision 3 — the caret line's message appears directly under the source panel's text

Learning `tasks/learnings.md:1073-1078` says to name the surface. Criterion 5 applies only when the
caret is in the source panel, and the caret can be there only when the data panel is open. A line in
`#data-panel-body`, directly under the text and above the Render row, is therefore always readable
when the criterion applies. The other surfaces do not work:

- `#render-status` belongs to render outcomes, and each revalidation replaces it
  (`viewer.js:316-325`).
- `#stats` is the bottom bar. It does not wrap, and a lint message is one sentence that can be long.

Space for the message line is reserved, so the text does not move when the message appears or clears.

### Decision 4 — between a keystroke and the next revalidation, the gutter and the message keep their lines

The diagnostics describe the text that was last checked. The panel's text is committed before the
next parse (`tasks/learnings.md:1037-1042`). The gutter and the caret line's message keep the
last-reported diagnostics at their line numbers until the next revalidation replaces them. That
happens `REVALIDATE_PAUSE_MS` (300 ms, `config.js:34`) after typing pauses. They do not move with
inserted lines, and they are not cleared on each keystroke, because clearing them would make them
flicker on every key.

If a click names a line past the end of the current text, the caret goes to the end of the text. When
a different model's text arrives, the old gutter stays for the short time until the parse resolves.
If that parse is refused, `viewer.js:270-276` restores the previous text, which those entries
describe.

### Decision 5 — what a diagnostic's line and column count

- **Lines** are the panel's own LF lines. `renderPanelSource` sends `parseEmod` the textarea's
  `.value` (`viewer.js:221`, `:233`), and the textarea has already converted CRLF to LF
  (`tasks/learnings.md:1031-1036`). The pipeline therefore counts the same lines the panel shows, and
  a CRLF file does not shift them.
- **Columns** are HCL's: user-perceived characters (grapheme clusters), counted from 1. HCL's
  `hcl.Pos` documents this, and the parser reports through it (`internal/parser/parser.go:47-66`). A
  probe through the real pipeline (`emod export --format json`) confirms it. On the line
  `actor "Cáfe👩‍💻" { 7 }`, where the `á` is a combining sequence and the emoji is a ZWJ sequence, the
  diagnostic at `7` has column **17**. A byte count would give 33, a UTF-16 count 22, and a
  code-point count 20. A JavaScript caret offset counts UTF-16 units, so the conversion must count
  grapheme clusters, not add the column to the line start.
- Learning `tasks/learnings.md:821-825` describes a lexer that no longer exists. `internal/lexer` was
  removed when the emod grammar was retired for HCL (`git log`: `ef27bf9`, `f404d2e`). Its per-byte
  advance does not apply now. Only the flow-entry columns are byte-based (see Deferred).

### Decision 6 — Delete and Backspace in the source panel edit the text

`initKeyboard` (`ui.js:1039-1073`) deletes the selected diagram element or connection on Delete or
Backspace, wherever focus is, and stops the key's default action. When a diagnostic is clicked, focus
lands in the source panel, and the diagram selection can remain. `handleDiagnosticClick` hides the
detail panel only when a node matches (`ui.js:345-350`). Without a guard, the first Backspace a user
types to fix the error deletes a diagram element and leaves the text unchanged. The hazard already
exists for a plain click into the panel. This story makes it the normal path to fixing an error, so
Task 3 adds the guard.

### Decision 7 — criterion 6 guards a contract value; no stage produces it today

No stage of today's pipeline emits a diagnostic without a line. The parser reports only HCL
diagnostics that have a subject (`internal/parser/parser.go:47-53`). Every validator and linter entry
takes the position of a parsed element (`internal/validator/validator.go:178-186`,
`internal/linter/linter.go:17-48`). Probes of a file with no `model` block, an empty file, a duplicate
`model` and an unsupported version all report a line, or report nothing. Line 0 can still arrive:
`jsonDiagnosticEntry.Line` is a plain `int` (`internal/export/json.go:244-252`), and pasted diagram
JSON brings its own diagnostics. The current panel shows `?` for line 0 (`ui.js:261`), and an
existing test fixture uses it (`diagnostics-highlight.test.js:77-87`). Criterion 6 is tested by
supplying line 0 through the platform seam.

---

## Codebase Context

**Where diagnostics come from.** `oracle.RunParsed` (`internal/oracle/oracle.go`) runs HCL parse →
validate → lint. `pipeline.RunPipelineExportDiagram` wraps the result as
`{diagnostics, diagram, parsed}` (`internal/export/diagram.go:99-120`). Each diagnostic reaches the
frontend as `{file, line, column, message, severity, rule_name}`
(`internal/export/json.go:244-292`). `line` and `column` are always present, and 0 means unknown.
`file` is the name the panel's text was parsed under: the open file's name, or `input.emod` for
pasted source (`internal/pipeline/pipeline.go:27-43`). **No Go change is needed.** Every field the
story uses already reaches the frontend. The browser's WASM worker and the desktop's
`ModelService.ParseEmod` both use this one route.

**Where they are shown.** `viewer.js:301-304` stores them on `store.diagnostics` and emits
`diagnostics:changed`. `viewer.js:72-74` sends that event to `UI.updateDiagnosticsPanel`
(`ui.js:233-278`), which renders each item as severity pill + `file:line` + `[rule] message`. The
`.diag-location` span shows `?` for a missing line. The ranking that `SEVERITY_NOUNS`, `severityOf`
and `mostSevere` apply (`ui.js:280-303`) has two rules: an unknown severity counts as an error, and
error outranks warning, which outranks info. The badge uses this ranking.

**Where the click goes.** `UI.initDiagnosticsDelegation` (`ui.js:353-359`) attaches one listener to
`#diagnostics-list`. `handleDiagnosticClick` (`ui.js:305-351`) does the following:

- It clears the not-rendered markers.
- When the diagnostic has no file or no line, it marks the item not-rendered and returns.
- When the diagram is stale, it clears highlights, marks the item and returns.
- Otherwise it highlights every node whose `position.filename` and `position.line` match.

The early returns are where Task 3's caret move must not be cut off. A parse error leaves the diagram
stale, and a parse error is exactly the diagnostic whose line the user most needs to reach.

**The source panel.** `#source-input` is a textarea inside `#data-panel-body`, inside `#data-panel`
(`viewer.html:1267-1281`). Its styles are at `viewer.html:629-642`: 12px monospace, line-height 1.5,
padding 10px, resizable vertically. A successful non-revalidation render collapses the data panel
(`viewer.js:255-257`), which hides its body. Several places open it again with the `collapsed` class
(`viewer.js:338`, `:544`). Store DOM references are created in `store.js` and set in `init`
(`viewer.js:135-169`). Typing calls `reportModified` and `scheduleRevalidation`
(`viewer.js:351-371`).

**Words already taken.** In the renderer, "mark" and "marking" are the construct icons drawn on
diagram blocks (`renderer.js:16-18`, `:298-335`). "marker" is both the SVG arrow marker and the
window's modified marker (`viewer.js:415`). "highlight" is the diagram highlight (`.hl`,
`highlightElements`). "annotation" means actor annotations. "cursor" is only the CSS pointer. That is
why the planned design uses **gutter** for the strip, **caret** for the insertion point and
**location** for a diagnostic's `file:line:column`, which matches the existing `.diag-location`.

**Tests.**

- **Unit tests:** vitest on jsdom, in `internal/frontend` (`vitest.config.js`, limited to two threads
  on purpose). Run them with `task test:viewer`.
- **Driving the whole viewer:** `viewer.test.js` stubs only the platform seam
  (`viewer.test.js:36-77`). It builds its DOM by hand in `createRequiredElements` (`:79-125`), so any
  element `init` looks up must be added there. The test `viewer.html satisfies what init reads from
  it` (`:3346-3372`) fails if `viewer.html` lacks an id that `viewer.js` looks up.
- **Timers:** the suite records and clears every timer a test starts (`:248-266`). Fake timers are not
  an option (`tasks/learnings.md:1322-1328`).
- **The diagram highlight on click:** `diagnostics-highlight.test.js` drives `UI` directly, with a
  store that has no `sourceInput`. The new click behaviour must tolerate that store.
- **Real browser:** Playwright in `e2e-viewer` runs against the built web bundle and the real WASM
  pipeline (`task test:e2e:viewer`). `source-panel.spec.js` already holds the source-panel specs:
  `landsOn` at `:13-20`, the largest example and `MANY_DIAGNOSTICS` at `:62-93`, the long-task budget
  at `:107-136`, and the bottom-dock reach checks at `:139-174`.
- **What jsdom cannot check:** jsdom has no layout. Scroll position, gutter alignment and how the
  severities look can only be checked in the e2e suite and by looking at the page
  (`tasks/learnings.md:767-771`). The desktop app runs WebKit, not Chromium, so look at it in the
  desktop build too. The story set does not allow automated tests that drive the desktop window.

**Documentation.** The README's **The source panel** paragraph (`README.md:401-410`) describes what
the panel does. US-009 extended it, and `viewer.test.js:3323-3340` pins some of its words.

---

## Tasks

### Task 1: Show each diagnostic's line and column, or that it applies to the whole file

**Behavior:** Each item in the diagnostics panel shows where its diagnostic is: the file, the line,
and the column when one is known. A diagnostic with no line says that it applies to the whole file,
and shows no line number and no placeholder.

**Acceptance Criteria:**
- [x] A diagnostic with a line and a column shows them in the conventional order, e.g.
  `orders.emod:12:5`
- [x] A diagnostic whose column is 0 or absent shows only the file and line, e.g. `orders.emod:12`.
  It does not show `:0`
- [x] A diagnostic whose line is 0 or absent says it applies to the whole file. Its location shows no
  line number and no `?`
- [x] The other content is the same as today: the severity pill, the `[rule]` prefix before the
  message, the badge counts and the not-rendered marker on a click that matches no diagram element
- [x] `task test:viewer` passes

**Affected Files/Modules:**
- `internal/frontend/static/ui.js` — the location text in `updateDiagnosticsPanel`
- `internal/frontend/tests/viewer.test.js` — the `what the diagnostics panel reports` block
- `internal/frontend/tests/diagnostics-highlight.test.js` — only if its whole-file case reads the
  location text

**Patterns to Follow:**
- `internal/frontend/static/ui.js:259-271`
- `internal/frontend/tests/viewer.test.js:2442-2475`
- `internal/frontend/tests/diagnostics-highlight.test.js:77-87`
- `internal/export/json.go:244-252`

**Testable:** Yes

**Certainty:** high — the location is already rendered at `ui.js:259-271` and tested at
`viewer.test.js:2442-2475`. This task changes only its text.

**Blast radius:** low — display text in the viewer. The diagnostics JSON contract does not change.

**Verification:** `task test:viewer` passes.

**Depends on:** None

---

### Task 2: Mark the lines that carry diagnostics in a gutter beside the source panel

**Behavior:** Beside the source panel's text, a gutter marks every line that carries at least one
diagnostic, with that line's most severe severity. Error, warning and info are told apart at a
glance, in the colours the diagnostics panel uses for them. The gutter scrolls with the text. The
source panel no longer soft-wraps, so each source line is one row and lines up with its gutter entry
(Decision 2). The gutter follows every render and revalidation, including one whose source does not
parse.

**Acceptance Criteria:**
- [x] After a render or revalidation that reports diagnostics on several lines, the gutter marks
  exactly those lines, and no others
- [x] Each gutter entry names its line's severity in text that a hover or a screen reader can read.
  This is in addition to its colour
- [x] Error, warning and info entries are drawn in three different colours: the colours the
  diagnostics panel's severity pills use (`viewer.html:325-336`)
- [x] A line that carries diagnostics of more than one severity shows the most severe. A severity the
  viewer does not know counts as an error. The gutter uses the same ranking the badge uses
  (`ui.js:280-303`), not a second copy of it
- [x] A diagnostic with no line marks no line
- [x] A revalidation that reports nothing on a line removes that line's entry. A clean revalidation
  and an empty panel clear the gutter. A revalidation whose source does not parse still marks the
  lines it reports
- [x] Typing does not move or clear the gutter entries. The revalidation after the pause replaces
  them (Decision 4)
- [x] The source panel does not soft-wrap. A line longer than the panel scrolls horizontally, and the
  text that Save writes is unchanged
- [x] In a real browser with the real pipeline, the largest bundled example with diagnostics far down
  the file is scrolled in the source panel. After the scroll, each gutter entry is level with its
  line's text
- [x] The existing e2e specs `source-panel.spec.js` and `smoke.spec.js` pass unchanged. These cover
  the source, Render and status staying in reach while diagnostics are listed, and no main-thread
  task over 50 ms while the largest example is typed into
- [x] The README's source-panel paragraph says that lines carrying diagnostics are marked by severity
- [x] `task test:viewer` and `task test:e2e:viewer` pass

**Affected Files/Modules:**
- `internal/frontend/static/source-panel.js` — new: the `SourcePanel` module (see the planned design)
- `internal/frontend/static/viewer.html` — the gutter beside `#source-input`, the non-wrapping text,
  and the severity colours
- `internal/frontend/static/viewer.js` — DOM references, and calling the gutter on
  `diagnostics:changed`
- `internal/frontend/static/store.js` — the gutter's DOM reference
- `internal/frontend/static/config.js` — the severity ranking, so the badge and the gutter read one
  ranking
- `internal/frontend/static/ui.js` — reads the ranking from its new home
- `internal/frontend/tests/source-panel.test.js` and/or `internal/frontend/tests/viewer.test.js` — the
  gutter's behaviour, and the new element in `createRequiredElements`
- `e2e-viewer/tests/source-panel.spec.js` — the gutter alignment after a scroll
- `README.md` — the source-panel paragraph

**Patterns to Follow:**
- `internal/frontend/static/viewer.js:72-74`
- `internal/frontend/static/minimap.js:124-160`
- `internal/frontend/static/ui.js:280-303`
- `internal/frontend/static/ui.js:233-278`
- `internal/frontend/static/viewer.html:259-360`
- `internal/frontend/static/viewer.html:573-582`
- `internal/frontend/static/viewer.html:629-642`
- `internal/frontend/static/viewer.html:1267-1281`
- `internal/frontend/static/store.js:1-65`
- `internal/frontend/tests/viewer.test.js:79-125`
- `internal/frontend/tests/viewer.test.js:2825-2860`
- `internal/frontend/tests/viewer.test.js:3346-3372`
- `e2e-viewer/tests/source-panel.spec.js:13-20`
- `e2e-viewer/tests/source-panel.spec.js:62-93`
- `e2e-viewer/tests/source-panel.spec.js:107-136`
- `README.md:401-410`
- `tasks/learnings.md:767-771`
- `tasks/learnings.md:1329-1333`
- `tasks/learnings.md:1322-1328`

**Testable:** Yes

**Certainty:** low — nothing in this repo is positioned against a textarea's lines. The no-wrap gutter
is designed in this task, and only a real browser can check that it lines up.

**Blast radius:** low — presentation of the source panel. The panel's value, and so what Save writes,
does not change.

**Verification:** `task test:viewer` and `task test:e2e:viewer` pass. Then build the web bundle
(`task build:web`), serve it, open `examples/all_patterns.emod` with an error added near the end,
scroll, and check that the gutter lines up. Do the same in the desktop build (`task build:desktop`),
because WebKit lays out a textarea differently from Chromium.

**Depends on:** None

---

### Task 3: Move the source panel's caret to a diagnostic when it is clicked

**Behavior:** Clicking a diagnostic that has a line does the following:

- The data panel opens if it was collapsed.
- The source panel gets focus.
- The caret goes to the diagnostic's location: before the character its column names, or to the
  start of the line when the column is unknown.
- The source panel scrolls so the caret is visible.

The click still highlights the matching diagram element, as it does today. The caret moves also when
the diagram is out of date, which is the case for a parse error. A diagnostic that applies to the
whole file leaves the caret and the scroll where they were. Delete and Backspace typed in the source
panel then edit the text. They do not delete a diagram element that happens to be selected
(Decision 6).

**Acceptance Criteria:**
- [ ] Clicking a diagnostic at line L, column C places the caret, with nothing selected, before the
  C-th character of line L of the panel's text. A column of 0, or no column, places it at the start of
  line L
- [ ] A column counts user-perceived characters, as HCL's columns do (Decision 5). On the line
  `actor "Cáfe👩‍💻" { 7 }`, a diagnostic at column 17 places the caret immediately before the `7`
- [ ] The click opens a collapsed data panel and gives the source panel focus, so the caret can be
  seen
- [ ] In a real browser with the real pipeline, the largest bundled example has a diagnostic near
  its end. Clicking that diagnostic scrolls the source panel so the caret's line is visible. Clicking
  a diagnostic whose column is past the panel's width scrolls the panel horizontally to it
- [ ] Clicking a diagnostic still highlights the diagram element declared at its line, and still
  marks it not-rendered when no element matches. The existing tests in
  `diagnostics-highlight.test.js` pass
- [ ] While the diagram is out of date, a click still moves the caret to the diagnostic. It
  highlights nothing, as today
- [ ] Clicking a diagnostic that applies to the whole file leaves the caret position and the source
  panel's scroll unchanged. It does not move to line 1
- [ ] A diagnostic whose line is past the end of the panel's current text places the caret at the end
  of the text (Decision 4)
- [ ] With a diagram element or connection selected, Delete and Backspace typed in the source panel
  edit its text, and the selection stays on the diagram. With focus outside any text field, the two
  keys still delete the selected element or connection, as before
- [ ] The README's source-panel paragraph says that clicking a diagnostic moves the caret to the
  source that caused it
- [ ] `task test:viewer` and `task test:e2e:viewer` pass

**Affected Files/Modules:**
- `internal/frontend/static/source-panel.js` — moving the caret to a location
- `internal/frontend/static/ui.js` — `handleDiagnosticClick` sends the click to the source panel
  before its early returns. `initKeyboard` stops taking Delete and Backspace away from a text field
- `internal/frontend/static/viewer.js` — only if the panel-opening step needs wiring there
- `internal/frontend/tests/diagnostics-highlight.test.js` — the highlight regression guard, which
  stays green
- `internal/frontend/tests/viewer.test.js` and/or `internal/frontend/tests/source-panel.test.js` —
  the caret, the panel opening, the whole-file case, the stale-diagram case and the keyboard guard
- `e2e-viewer/tests/source-panel.spec.js` — scrolling the caret into view on the largest example
- `README.md` — the source-panel paragraph

**Patterns to Follow:**
- `internal/frontend/static/ui.js:305-359`
- `internal/frontend/static/ui.js:1039-1073`
- `internal/frontend/static/viewer.js:334-339`
- `internal/frontend/static/viewer.js:206-233`
- `internal/frontend/tests/diagnostics-highlight.test.js:1-136`
- `internal/frontend/tests/viewer.test.js:2477-2540`
- `internal/frontend/tests/viewer.test.js:153-157`
- `internal/parser/parser.go:47-66`
- `e2e-viewer/tests/source-panel.spec.js:62-69`
- `e2e-viewer/tests/helpers.js:102-121`
- `tasks/learnings.md:1031-1036`
- `tasks/learnings.md:1329-1333`
- `task:1`
- `task:2`

**Testable:** Yes

**Certainty:** medium — the click delegation (`ui.js:305-359`) and opening the data panel
(`viewer.js:338`) both have precedent. Nothing here yet moves a textarea's caret or scrolls it to a
line, and nothing converts a diagnostic's column, counted in grapheme clusters, to a text offset.

**Blast radius:** low — caret placement and a keyboard guard. A diagram deletion changes only the
in-memory diagram, and Save writes the panel's text (`viewer.js:31-41`).

**Verification:** `task test:viewer` and `task test:e2e:viewer` pass. In the web bundle and the desktop
build, click diagnostics near the end of a long model and check that the caret is visible. WebKit and
Chromium scroll a focused textarea differently.

**Depends on:** Task 1, Task 2

---

### Task 4: Show the diagnostics of the caret's line under the source panel

**Behavior:** When the caret is on a line that carries diagnostics, a line directly under the source
panel's text shows each of them, with severity, `[rule]` prefix and message, most severe first. This
happens however the caret got there: arrow keys, a click in the text, or a click on a diagnostic. When
the caret moves to a line with no diagnostics, the line under the text clears. A revalidation that
changes what the caret's line carries updates the message without moving the caret. Space for the
message line is reserved, so the text does not move when the message appears or clears
(Decision 3).

**Acceptance Criteria:**
- [ ] Moving the caret onto a line that carries a diagnostic shows that diagnostic's severity and
  message under the source panel. The `[rule]` prefix appears as the diagnostics panel shows it. This
  holds when the caret is moved by the keyboard, by a click in the text, and by a click on the
  diagnostic in the diagnostics panel
- [ ] A line that carries several diagnostics shows all of them, most severe first, ranked as the
  badge ranks them
- [ ] Moving the caret to a line that carries no diagnostics clears the message
- [ ] A revalidation that adds or removes a diagnostic on the caret's line updates the message, and
  the caret does not move
- [ ] A diagnostic that applies to the whole file is never shown as the message of a line
- [ ] In a real browser, the source panel's text keeps the same position and size when the message
  appears and when it clears
- [ ] The README's source-panel paragraph says that the caret's line shows its diagnostics under the
  panel
- [ ] `task test:viewer` and `task test:e2e:viewer` pass

**Affected Files/Modules:**
- `internal/frontend/static/source-panel.js` — showing the caret line's diagnostics, and following the
  caret
- `internal/frontend/static/viewer.html` — the message line under `#source-input`, with reserved space
- `internal/frontend/static/viewer.js` — DOM reference, and calling the message on
  `diagnostics:changed`
- `internal/frontend/static/store.js` — the message line's DOM reference
- `internal/frontend/tests/viewer.test.js` and/or `internal/frontend/tests/source-panel.test.js` —
  the message, and the new element in `createRequiredElements`
- `e2e-viewer/tests/source-panel.spec.js` — the text keeps its geometry when the message appears and
  clears
- `README.md` — the source-panel paragraph

**Patterns to Follow:**
- `internal/frontend/static/viewer.js:351-354`
- `internal/frontend/static/ui.js:259-271`
- `internal/frontend/static/ui.js:280-303`
- `internal/frontend/static/viewer.html:644-680`
- `internal/frontend/tests/viewer.test.js:2825-2860`
- `e2e-viewer/tests/source-panel.spec.js:139-174`
- `tasks/learnings.md:1073-1078`
- `tasks/learnings.md:1037-1042`
- `task:2`
- `task:3`

**Testable:** Yes

**Certainty:** medium — listening to the source panel (`viewer.js:351-354`) and rendering a message
with its rule (`ui.js:259-271`) both have precedent. Nothing here yet follows the caret of a textarea,
and the message line is a new surface.

**Blast radius:** low — a read-only display under the source panel.

**Verification:** `task test:viewer` and `task test:e2e:viewer` pass. In the web bundle, move the caret
through a model with diagnostics on neighbouring lines, and check that the text does not move.

**Depends on:** Task 2, Task 3

---

## Summary

- **Total tasks:** 4, all in JavaScript in `internal/frontend/static`. No Go change: every field the
  story needs (`line`, `column`, `severity`, `message`, `rule_name`) already reaches the frontend.
- **Ordering:**
  - Task 1 comes first because it is certain and small, and it defines the "whole file" case that
    Task 3 relies on.
  - Task 2 comes before the other source-panel tasks because it is the least certain, and because it
    sets the panel's geometry by turning off soft wrap. Task 3 scrolls that geometry, and Task 4's
    message line sits under it.
  - Task 3 depends on Tasks 1 and 2.
  - Task 4 depends on Tasks 2 and 3, so that a diagnostic click shows the message too.
- **Coverage:** all six story criteria are covered (see the table under Story Reference). None is
  deferred. Criterion 3 is a regression guard on behaviour that exists today.
- **Assessments:**
  - Certainty is spread across the tasks: one `high`, one `low`, two `medium`.
  - Blast radius is `low` throughout. This is a genuine finding, not a default. No task touches auth,
    money, a migration, secrets, personal data, or a contract consumed outside the repository. The
    diagnostics JSON is unchanged, and Save writes the panel's text, which no task alters.
- **Planned design:** `tasks/us-010-jump-between-a-diagnostic-and-the-source-that-caused-it.design.yaml`.
  It plans one new module, `SourcePanel` (`static/source-panel.js`), and changes to `UI`. Clerk's
  design reader reads only Go and Elixir. It cannot see `SourcePanel`, so at the last task
  `clerk finish` reports `design-absent-type` for it. Answer that with a `clerk design note` that
  names the JavaScript module. The finding does not mean the design is wrong.
- **For the caller to confirm or overrule before Task 2:**
  - Decision 2: turning off soft wrap.
  - Decision 6: guarding Delete and Backspace in text fields. This fixes a hazard that already exists,
    which this story makes the normal path.
  - Decision 1: the distribution and its goal tension.

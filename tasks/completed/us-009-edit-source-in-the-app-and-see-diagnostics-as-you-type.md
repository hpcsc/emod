# US-009: Edit source in the app and see diagnostics as you type

## Contents

1. Task 1: Report the diagnostics `emod validate` reports for the panel's source
2. Task 2: Keep the last diagram on screen, marked stale, when the panel's source does not parse
3. Task 3: Re-render the panel's source in place, keeping pan, zoom, node layout and visibility
4. Task 4: Revalidate the source panel automatically after a pause in typing

Tasks 1–3 change what a render of the panel's own source does, and are reached through the
Render control while they land. Task 4 is the switch that starts those renders without a click.
That order means no commit on the way re-renders on every pause while a half-typed construct can
still empty the canvas or undo the user's layout.

| Slice | Tasks | Exit |
|---|---|---|
| **What a render reports** | 1 | The badge and panel say what `emod validate` says for the same text |
| **What a render keeps** | 2, 3 | Source that does not parse leaves the diagram, marked stale; source that does redraws without moving anything the user arranged |
| **Rendering without a click** | 4 | Editing the panel revalidates it after a pause, in all three distributions, without overtaking an open, a save or a reported failure |

---

## Story Reference

`user-stories/emod-desktop.md` → **US-009: Edit source in the app and see diagnostics as you type**
(`:137-150`). Depends on US-001; US-002 through US-008 are delivered as well (`tasks/completed/`).

Its siblings draw the boundary, and each is named below where the work stops short of it: US-010
(`:152-163`) moves between a diagnostic and its source, US-011 (`:165-178`) reconciles diagram edits
with source edits, US-012 (`:180-190`) highlights the source.

| Story criterion | Task |
|---|---|
| Typing in the source panel revalidates automatically after a short pause, with no Render click | 4 |
| The diagnostics badge and panel update on each revalidation with the same rule messages, severities, and locations the CLI reports for the same source | 1 decides what they say; 4 says it on each revalidation |
| Source that validates cleanly re-renders the diagram in place, preserving the current pan, zoom, and node layout | 3 |
| Source that fails to parse keeps the last successfully rendered diagram on screen and marks it as stale, rather than blanking the canvas | 2 |
| Typing stays responsive while validation runs, on the largest bundled example model | 4 |
| An explicit render control remains available | 4; 2 and 3 settle what it does |

---

## Theory and decided questions

### Decision 1 — continuous validation ships in all three distributions

The browser viewer, `emod diagram --serve` and the desktop app all get it:

- The viewer is one copy of `internal/frontend/static`, and US-001 pinned that no shared file has a
  second. A desktop-only switch would be either a branch on the platform in shared code or a seam
  operation whose only job is to turn a feature off.
- The stale mark and the in-place redraw are canvas behaviour, and the story's non-goal is explicit:
  "Anything added to the canvas stays available in all three distributions".
- CLI parity holds everywhere because every route reaches one chain. The browser's WASM `parseEmod`
  (`cmd/emod-wasm/main.go:19-21`) and the desktop's `ModelService.ParseEmod`
  (`internal/desktop/service.go:20-22`) both run `pipeline.RunPipelineExportDiagram` →
  `oracle.Run`, the chain `emod validate` runs through `oracle.Check` (`internal/cli/validate.go:27`).
- The goal that the web viewer "keep[s] working exactly as [it does] today" is read as: nothing it
  does today stops working. Render still renders, a drop still opens, Export still downloads.

The one runtime difference that matters is where the pipeline runs: synchronously on the browser's
main thread (`internal/frontend/static/platform.browser.js:44-54`), over an asynchronous binding on
the desktop. Responsiveness is therefore verified in the browser build, the worst case.

`--serve` has one limit of its own: the server injects the rendered diagram, not the source
(`internal/cli/diagram.go:172-193`), and `init` reads only `state.diagram`
(`internal/frontend/static/viewer.js:714-716`). The served model's source never reaches the panel,
so continuous validation there applies to what is typed or pasted into it. See Boundaries.

### Decision 2 — "fails to parse" means the lexer or the parser reported something, and the pipeline has to say so

The pipeline recovers from a syntax error and still answers a diagram, so "does not parse" cannot be
read off whether a diagram came back. Measured on `examples/all_patterns.emod` (30 nodes, 22 edges)
with one half-typed line inserted at line 60:

| Inserted line | Nodes | Edges |
|---|---|---|
| `command HalfTyped {` | 13 | 4 |
| `description "unterminated` | 7 | 2 |
| `comm` | 30 | 22 |

Rendering what was recovered empties most of the canvas mid-keystroke — the blanking the criterion
forbids. And the viewer cannot tell a parse error from a validation error by what reaches it today:
lexer and parser diagnostics carry no rule name and are always error severity
(`internal/lexer/tokenizer.go:95-97`, `internal/parser/parser.go:1824-1831`), and hard validator
errors carry no rule name either (`tasks/learnings.md:166-170`). The oracle already runs the stages
apart (`internal/oracle/oracle.go:14-31`), so the fact exists in Go and is simply not reported.

It is reported on the viewer's route only. `emod export --format diagram-json` prints the same
`{diagnostics, diagram}` wrapper from `export.ExportDiagramJSONDiagnostics`
(`internal/cli/export.go:81-101`) — a CLI output format. The pipeline's answer is a separate route
that only the two viewer runtimes read.

Source that parses but carries validation errors or lint findings re-renders. That is what US-002
already asks of a file with errors ("the diagram renders what it can"), and many models — the
examples among them — carry lint findings indefinitely; reading "validates cleanly" as "no
diagnostics at all" would freeze their diagrams for good.

### Decision 3 — opening a model is not a re-render

The keep-the-last-diagram rule and the in-place rule apply when the panel's own source is rendered
again — by the Render control, or by a revalidation once Task 4 lands. A model arriving through
`openModel` (`internal/frontend/static/viewer.js:295-301`), the one function every open goes through
(pinned by `internal/frontend/tests/viewer.test.js:2288-2332`), always draws what the pipeline
recovered and starts with fresh view state, as it does today. Keeping the departing model's diagram
for an arriving file would put one model's diagram under another's name, and would leave
`store.currentFile` uncommitted while the panel holds the arriving text — the gap
`tasks/learnings.md:1037-1042` records, through which the next Save overwrites the wrong file.

One consequence, stated plainly: US-002 says a file with errors opens "exactly as pasted source with
the same errors does". For errors that stop a parse that stays true of pasted source landing on an
empty canvas; pasted over a diagram already on screen, such source now keeps that diagram, marked
stale — which is what criterion 4 asks.

### Decision 4 — "stale" is a property of the canvas, and is marked on it

A diagram is stale when it was not drawn from the source the panel now holds. The mark goes on the
canvas because every other surface fails in a known way: `#render-status` sits in `#data-panel`,
which every successful render collapses to its header (`internal/frontend/static/viewer.js:224`,
`tasks/learnings.md:983-988`, `:1073-1078`); `#save-status` is the bar's one slot and already has
two writers (`tasks/learnings.md:1193-1198`); the diagnostics panel can be closed. The canvas is on
screen whenever the diagram is, and the overlays `#canvas-container` holds sit at its edges — the
minimap top-right (`internal/frontend/static/viewer.html:69-73`), the visibility panel top-left, the
legend bottom-left, the data and diagnostics panels along the bottom (`:210-225`, `:483-500`). A
jsdom text assertion cannot tell whether the mark can be seen, so Task 2 settles it in a real
browser.

### Decision 5 — what "pan, zoom, and node layout" covers, and what a node is across two parses

Derived from what a render can reset rather than from the criterion's three words:

| State | Where it lives | What a render does to it today |
|---|---|---|
| Pan and zoom | `store.viewport` | Nothing — `setModelData` (`internal/frontend/static/model.js:19-31`) never assigns it, so both already survive |
| Dragged node positions, which Reset layout restores | `store.nodeOffsets`, read at `layout.js:155-162` | `setModelData` empties it |
| Contexts, aggregates and slices hidden from the visibility tree | `store.hiddenNodes`, read at `layout.js:78` and `:179-186` | `setModelData` empties it |
| Reset layout's enabled state | the button | Disabled by every render (`viewer.js:64-65`, `:227-228`) |
| The data panel's collapse | `#data-panel` | Collapsed by every successful render (`viewer.js:224`) — Task 4's concern, since the user is typing in it |
| The diagnostics panel's open state | `#diagnostics-panel` | Re-opened by every render that reports (`ui.js:265`) — Task 4's concern |
| Selection, highlights, the detail panel | `store.interaction`, `#detail-panel` | Cleared before every render (`viewer.js:112-120`) — left so; see Boundaries |

Offsets and hidden state are keyed by node id, and node ids are per-kind sequence numbers in document
order (`internal/export/diagram.go:99-113`): adding a command above another renumbers every command
after it, so carrying the maps across by id would move the user's arrangement onto other nodes.
**A node is the same node across two renders when the source names it the same way — the same kind
of construct, the same name, inside the same named parents.** Renaming a node in the source makes it
a different node.

### Decision 6 — what "the same … the CLI reports" covers

Derived from what `emod validate` prints: the text form is `file:line: [rule] message`
(`internal/diagnostic/entry.go:33-38`), the JSON form carries `file`, `line`, `rule`, `severity`,
`message` (`internal/cli/lint.go:41-47`), and neither prints a column — showing one is US-010's
first criterion. The viewer's answer already carries lint results, because it runs the whole chain.
Four things still differ:

| Part | `emod validate` | The panel today |
|---|---|---|
| Message | the message, prefixed `[rule]` when it has one | the message only — the rule name is carried (`internal/export/json.go:251`) and never shown (`ui.js:253-262`) |
| Severity | `error`, `warning`, `info` | each item names its own, but the badge says "N errors" whatever they are (`ui.js:251`) |
| File | the path it was given | `input.emod`, fixed in the pipeline (`internal/pipeline/pipeline.go:45`) |
| Line | counted from the file's first byte | counted after `.trim()` (`viewer.js:212`), so leading blank lines shift every line — measured: three leading blank lines give CLI line 10, the trimmed text line 7 |

CRLF needs nothing: the lexer counts `\n` and treats `\r` as whitespace
(`internal/lexer/tokenizer.go:99-115`), so the LF text the panel normalises to reports the same lines.

### Decision 7 — responsiveness needs a pause, not a worker

Measured in Node 23 against the repository's own WASM build, the pipeline over
`examples/all_patterns.emod` takes 2.7 ms at the median and 29 ms on its first, cold call. No work
per keystroke plus a revalidation well inside one long task is enough, and moving the pipeline off
the main thread is not needed. The check is the browser's own Long Tasks threshold, 50 ms, in the
browser build.

**Reversed on 2026-09-23.** The HCL parser made the pipeline about 4.5 times slower in WASM: over
the same example, measured in Node 23, the median went from 1.9 ms to 8.5 ms. On the GitHub runner,
two revalidations took 53 ms and 66 ms, and the long-task check failed. The browser build now runs
`emod.wasm` in a Web Worker (`internal/frontend/static/wasm-worker.js`), and the main thread does
only the redraw.

### Decision 8 — the Render control stays, and follows the same rules

It stays where it is and renders at once. It follows the keep-last and in-place rules a revalidation
does, so the canvas never loses content to a half-typed construct however the render was asked for.
It still collapses the panel on success as it does today — a click on it asks to see the diagram —
while a revalidation leaves the panel open, because the user is typing in it.

---

## Boundaries

**Out of scope** — the story's Non-Goals (`user-stories/emod-desktop.md:248-259`), carried verbatim:

- **No code signing or notarization** on any platform — no Apple Developer account, no App Store, no Windows signing certificate. Users take a documented one-time step on first launch.
- **No Linux distributions below the declared floor** — Ubuntu 22.04, Debian 12, RHEL 9 and other pre-GTK4 stacks are out of scope, with no legacy build variant.
- **No auto-update.** New versions are downloaded manually.
- **No replacement of existing distributions.** `emod diagram --serve` and the hosted web viewer keep working unchanged; desktop is a third option, not a migration.
- **No desktop-only diagram features.** Anything added to the canvas stays available in all three distributions.
- **No multi-window, tabs, or project workspaces.** One model per window.
- **No full text editor.** The source panel gains validation, navigation, and highlighting — not find-and-replace, multi-cursor, or refactoring.
- **No telemetry or crash reporting.**
- **No automated tests driving the desktop window.** Manual smoke testing of the shell, backed by the existing Go and browser suites.
- **No published package of the shared frontend.** It stays an internal asset assembled at build time.

Also out of scope, from this decomposition:

- **Reconciling diagram edits with source edits.** A revalidation renders the panel's source exactly
  as a Render click does, which replaces structural edits made on the canvas — added, deleted or
  renamed nodes, drawn arrows, reordered slices. Typing now triggers what only a click did before.
- **Any change to a CLI output.** The file name and the parse outcome travel only in the pipeline's
  viewer-facing request and answer; `emod export --format diagram-json`, `emod validate` and
  `emod lint` print what they print. The story's goal keeps the CLI working exactly as it does.
- **The served model's source in the panel, or its startup diagnostics, in `--serve`.** Decision 1.
- **Running the pipeline in a Web Worker.** Decision 7.
- **Keeping a node's offset across a rename in the source.** Decision 5.
- **Keeping the selection, a diagnostic's highlight or the detail panel across a re-render.** Every
  render clears them before it draws (`viewer.js:112-120`), and still does.
- **Collapsing repeated parser diagnostics.** The CLI prints them repeated too —
  `examples/error_diagnostics_test.emod:39` is reported three times — so listing them the same way is
  parity.
- **Any change to what Save or Export writes.** Save still writes `sourceToSave(store)`; Export still
  re-serialises the diagram on screen, stale or not.

**Deferred** — wanted, but not here:

- Columns in the panel, moving the caret to a diagnostic, marking the lines that carry one, and the
  whole-file diagnostic → US-010.
- A diagram edit reaching the source panel, and saying so when both surfaces have diverged → US-011.
- Syntax highlighting → US-012.
- **The LSP reports info-severity diagnostics as errors** — `Info` falls to the default branch of
  `ConvertDiagnostics` (`internal/lsp/diagnostics.go:31-36`). That is two surfaces disagreeing on a
  rule's severity, which the story's Context says must not happen; this story compares the panel with
  the CLI, and no story owns the LSP mapping. Raise it on its own.
- **`--serve` drops the diagnostics it computed at startup** (`viewer.js:714-716`), so a served model
  with findings shows no badge until something is rendered from the panel. No story owns it.

---

## Codebase Context

Verified against the working tree on 2026-09-10, branch `main`, with US-001 to US-008 landed.

**The render path.** `renderPanelSource(text, file)` (`internal/frontend/static/viewer.js:206-245`)
clears the save bar, writes the panel's text, trims it, claims a render number and hands it to
`Model.sendParse` (`internal/frontend/static/model.js:77-120`). The resolved branch commits
`store.currentFile` only when `file` is given, emits `diagnostics:changed`, calls
`Model.setModelData`, collapses the data panel and disables Reset layout; the rejected branch leaves
the canvas as it was and restores the panel's text for an open. The Render button calls it with no
arguments (`:331-333`); the only call with a file is `openModel` (`:295-301`). Three counters order
everything (`:176-194`): `latestDelivery` numbers opens, `latestRender` numbers renders, and
`latestRenderSettled` is what `rendersSettled()` chases so a save waits out every render in flight
(`:534-559`). The panel's own `input` event is already wired, to the unsaved marker (`:263-267`).
Every render today starts from a user gesture; the only timer in the file clears a "Ready" status
(`:734-739`).

**What reaches Go.** Browser and `--serve`: `platform.browser.js:44-54` calls the WASM global
synchronously on the main thread → `cmd/emod-wasm/main.go:19-21` → `pipeline.RunOnSource`
(`internal/pipeline/pipeline.go:97-113`). Desktop: `platform.desktop.js:16-20` calls
`ModelService.ParseEmod`, an asynchronous binding → the same `RunOnSource`. Both send
`{"source": …}`; `ExtractSource` (`pipeline.go:22-34`) reads only that key, and `runPipeline` runs
`oracle.Run(source, "input.emod")` (`:45`). Keeping `ParseEmod` a string-in, string-out method means
no binding is regenerated and `TestBindingNames` is untouched
(`internal/desktop/binding_names_test.go:22-36`). The CLI's `validate` and `lint` run
`oracle.Check(source, path)` (`internal/cli/validate.go:27`, `lint.go:123`); the LSP runs
`oracle.Run(text, uri)` (`internal/lsp/server.go:391`).

**What the answer carries and the panel shows.** Each diagnostic arrives as `file`, `line`, `column`,
`message`, `severity`, `rule_name` (`internal/export/json.go:244-293`). `UI.updateDiagnosticsPanel`
(`internal/frontend/static/ui.js:233-266`) writes the badge as "N error(s)", lists severity,
`file:line` and message, and removes `hidden` from the panel on every call that has diagnostics.
`handleDiagnosticClick` (`:268-308`) highlights nodes whose `position.filename` and `position.line`
equal the diagnostic's, and the detail panel prints a node's `position.filename:line` (`:464`) — so
the file name the pipeline is given reaches three places, all from one answer. The stylesheet styles
`.diag-severity.error` and `.warning` and has no rule for `.info` (`viewer.html:268-286`).

**The panels over the canvas.** `#data-panel` sits at the bottom of `#canvas-container`, at most half
its height, z-index 15 (`viewer.html:483-500`); `#diagnostics-panel` sits at the bottom too, at most
200 px, z-index 30 (`:210-225`). When both show, the diagnostics panel lies over the data panel's
lowest rows — where the Render button and `#render-status` are (`:1216-1219`) — by the stylesheet's
geometry. Today that is reached only by reopening the panel after a render; a revalidation that
re-opens the diagnostics panel on every pause makes it the normal state while typing.

**Everything that writes the status area.** Besides the render path: `reportHostFailure` (a guard
turn that threw, a drop refused by name, a file the host could not read, an empty file —
`viewer.js:250-255`, `:283`, `:391`, `:423`, `:431`), `reportSaveFailure` (`:510-522`), Export's
failure (`:638`) and the initial load (`:726-753`). The bar's `#save-status` is cleared at the start
of every render (`:207`, `:445-452`).

**Node identity.** `diagramIDGenerator` (`internal/export/diagram.go:99-113`) numbers nodes per kind
in document order — `command-1`, `command-2` — and edges name nodes by those ids. Offsets are
relative to the computed position and applied inside the slice (`layout.js:155-162`), recorded by a
drag (`interaction.js:217-231`), cleared by Reset layout (`viewer.js:617-621`).

**The harnesses.** `internal/frontend/tests/viewer.test.js` mocks only `../static/platform.js`
(`:36-77`) and drives everything through `init()`: `typeIntoPanel` fires the panel's `input` event
(`:152-156`), `dragBy`, `blockFor` and `hideFromVisibilityTree` drive the canvas (`:145-169`), and
`flush()` is a real `setTimeout(0)` (`:233-235`). No test in the suite fakes timers, and doing so
would stall every `flush()`. `sendParse` reaches the seam through a dynamic `import()`, so two renders
started in the same turn race the mock (`tasks/learnings.md:959-964`). `e2e-viewer/` is a Playwright
suite that drives the real `task build:web` bundle, WASM included, in Chromium; CI runs it through
`task test:e2e:viewer` (`.github/workflows/ci.yml:41`), and its helpers open the viewer, render
source and read the viewport transform back (`e2e-viewer/tests/helpers.js:94-126`). Go tests that
need every example derive the set from the directory (`internal/cli/validate_test.go:1094-1117`).

**The measurements behind the decisions.** CLI on three leading blank lines: `line 10` against `7`
for the same model without them. Parser recovery on `all_patterns.emod`: 30 nodes down to 13 or 7.
WASM pipeline on the same file in Node 23: 2.7 ms median, 29 ms cold. `emod validate` on an empty or
whitespace-only file reports nothing; the viewer refuses such a panel before parsing it
(`model.js:78-82`).

---

## Tasks

### Task 1: Report the diagnostics `emod validate` reports for the panel's source

**Behavior:** Whatever renders the panel's source — the Render control, an open, and after Task 4 a
revalidation — the badge and the diagnostics panel say what `emod validate` says for the same text:
the same messages under the same rule names, the same severities, and the same file and line.

**Acceptance Criteria:**
- [x] For every `.emod` under `examples/` — the set read from the directory, not listed — the diagnostics the viewer's parse answer carries when asked under that file's name equal what `emod validate --format json` reports for the file: message, severity, rule and line, entry for entry and in order
- [x] Each diagnostic that carries a rule name shows it beside its message, as `emod validate` prints it; one that carries none shows no rule
- [x] The badge counts each severity it reports separately, and a model reporting only warnings and infos is never announced as having errors
- [x] With a file open, each diagnostic's location names that file by its name rather than `input.emod`; source with no file behind it is reported under `input.emod`
- [x] Clicking a diagnostic still highlights the node declared at its line, and a node's detail panel names the same file its diagnostics name
- [x] Source beginning with blank lines or indentation reports each diagnostic at the line `emod validate` reports for the same bytes — three leading blank lines ahead of an orphaned command put it at line 10, not 7
- [x] A panel holding only whitespace is still refused as empty rather than rendered
- [x] Both platform implementations send the file name with the source, and the key they send it under is the key the pipeline reads, pinned by a guard that reads both sides

**Affected Files/Modules:**
- `internal/pipeline/pipeline.go` — the parse request carries the file name the diagnostics and node positions report
- `internal/pipeline/pipeline_test.go` — the request's leaves, the examples parity guard, and the key guard
- `internal/frontend/static/model.js` — `sendParse` hands on the panel's text with its leading lines, and the file name
- `internal/frontend/static/viewer.js` — names the file a render belongs to: the arriving file for an open, the open file otherwise
- `internal/frontend/static/platform.browser.js`, `internal/frontend/desktop/platform.desktop.js` — both build the request with the name
- `internal/frontend/static/ui.js` — the rule name in each item, the badge's count by severity
- `internal/frontend/static/viewer.html` — the rule name's style beside `.diag-severity`
- `internal/frontend/tests/viewer.test.js`, `internal/frontend/tests/diagnostics-highlight.test.js`, `internal/frontend/tests/model.test.js`, `internal/frontend/tests/platform.desktop.test.js`, `internal/frontend/tests/platform.browser.init-success.test.js`

**Patterns to Follow:**
- `internal/pipeline/pipeline.go:22-34`, `:97-113`
- `internal/pipeline/pipeline_test.go:16-73`
- `internal/cli/validate_test.go:1094-1117`
- `internal/cli/lint.go:41-94`
- `internal/frontend/static/ui.js:233-266`
- `internal/frontend/tests/diagnostics-highlight.test.js:1-123`
- `internal/frontend/tests/viewer.test.js:2334-2389`
- `internal/frontend/tests/platform.desktop.test.js:85-100`
- `internal/frontend/tests/platform.browser.init-success.test.js:36-59`
- `internal/desktop/binding_names_test.go:170-217`
- `tasks/learnings.md:1013-1018`, `:166-170`, `:226-230`, `:1109-1114`

**Testable:** Yes — the pipeline through its exported functions, each platform's request through its own suite, and the panel through `init()` against the mocked seam.

**Certainty:** high — each change repeats a pattern already here: the request envelope both platforms build and `RunOnSource` unwraps (`pipeline.go:97-113`), the panel's item and badge (`ui.js:233-266`), and the example set derived from the directory (`validate_test.go:1094-1117`).

**Blast radius:** low — it changes what the badge and panel say; the request is internal to this repository, built into both viewer runtimes from the same tree, and no CLI output changes.

**Verification:** `task test:unit` and `task test:viewer` green. `task build:web`, serve `web/`, paste `examples/error_diagnostics_test.emod` with three blank lines added above it, press Render, and read the panel against `emod validate --format json` on the same bytes.

**Depends on:** None

---

### Task 2: Keep the last diagram on screen, marked stale, when the panel's source does not parse

**Behavior:** Rendering the panel's own source again when that source does not parse leaves the
diagram that was on screen where it is and marks it as not reflecting the source, while the badge
and panel list what is wrong. Source that parses redraws and clears the mark. Opening a model always
draws that model.

**Acceptance Criteria:**
- [x] The viewer's parse answer states whether the source parsed — whether the lexer or the parser reported anything — identically in the browser and desktop builds
- [x] `emod export --format diagram-json` prints a document with exactly two top-level keys, `diagnostics` and `diagram`
- [x] Rendering the panel's source when it does not parse, with a diagram on screen, leaves that diagram node for node, lists the new source's diagnostics in the badge and panel, and marks the diagram stale — for every way such a render ends without a parsed model: the pipeline reporting a lexer or parser error, and each rejection `Model.sendParse` returns
- [x] Source that parses, including source with validation errors or lint findings, redraws the diagram and clears the mark
- [x] With no diagram on screen yet, source that does not parse draws what the pipeline recovered, unmarked
- [x] A model opened by any route — each reaches the screen through `openModel` — draws what the pipeline recovered even when it does not parse, clears the mark, names the window and becomes the save target; the departing model's diagram is never kept for it
- [x] While the diagram is marked stale, clicking a diagnostic highlights no node
- [x] In the browser build, driven by the e2e-viewer suite, `elementFromPoint` at the mark lands on it with the data panel collapsed and with it expanded, and with every panel the header's toggles open showing
- [x] The mark lives in the shared `viewer.html` and `viewer.js`, so the page `task build:desktop` derives from `viewer.html` carries it too
- [x] The key the viewer reads the parse outcome from is pinned against the key the pipeline writes, by the guard Task 1 introduced

**Affected Files/Modules:**
- `internal/pipeline/pipeline.go` — the viewer's answer says whether the source parsed, leaving the export package's wrapper the CLI prints as it is
- `internal/oracle/oracle.go` — where the chain already runs parsing apart from validation and linting
- `internal/pipeline/pipeline_test.go` — the answer's leaves, and the key guard's second pair
- `internal/frontend/static/viewer.js` — a render of the panel's own source keeps the diagram and marks it; an open and parsed source draw and clear it
- `internal/frontend/static/store.js` — whether the diagram on screen is stale
- `internal/frontend/static/ui.js` — no highlight while stale
- `internal/frontend/static/viewer.html` — the mark and its style
- `internal/frontend/tests/viewer.test.js`, `internal/frontend/tests/diagnostics-highlight.test.js`
- `e2e-viewer/tests/` — the mark on screen, with real WASM

**Patterns to Follow:**
- `internal/frontend/static/viewer.js:206-245`
- `internal/oracle/oracle.go:14-31`
- `internal/pipeline/pipeline.go:50-54`
- `internal/frontend/static/viewer.js:295-301`
- `internal/frontend/tests/viewer.test.js:2288-2332`
- `internal/frontend/tests/viewer.test.js:364-422`
- `internal/frontend/tests/viewer.test.js:1059-1118`
- `internal/frontend/static/viewer.html:69-73`
- `e2e-viewer/tests/smoke.spec.js:19-25`
- `e2e-viewer/tests/helpers.js:94-126`
- `tasks/learnings.md:983-988`, `:1073-1078`, `:767-772`, `:1037-1042`, `:1145-1150`
- `task:1`

**Testable:** Yes — the pipeline's answer in Go, the keep, draw and clear rules through `init()` against the mocked seam, and the mark's place on screen in the e2e-viewer suite.

**Certainty:** medium — a rejected parse already leaves the canvas as it was (`viewer.js:231-242`) and the oracle already runs parsing apart from validation and linting (`oracle.go:14-31`), but no branch accepts an answer's diagnostics while refusing its diagram, and the answer does not yet say which stage reported.

**Blast radius:** high — it splits the resolved branch of `renderPanelSource`, which is where `store.currentFile` is committed; applying the keep rule to an open leaves the arriving model's text in the panel under the departing file's identity, and the next Save overwrites that file with it.

**Verification:** `task test:unit`, `task test:viewer` and `task test:e2e:viewer` green. `task build:desktop && ./bin/emod-desktop`: open `examples/all_patterns.emod`, delete a closing brace, press Render, and see every node stay with the mark over them; open another model and see the mark go.

**Depends on:** Task 1

---

### Task 3: Re-render the panel's source in place, keeping pan, zoom, node layout and visibility

**Behavior:** Rendering the panel's own source again redraws the diagram where it is: the viewport
stays put, a node the user dragged keeps its offset, and what was hidden stays hidden — each followed
by what the source names it, so an edit that adds or removes constructs elsewhere never moves the
user's arrangement onto other nodes. Opening a model still starts it fresh.

**Acceptance Criteria:**
- [x] Rendering the panel's source again leaves pan and zoom as they were
- [x] A node dragged before the render keeps its offset after it, including when the edit adds, removes or reorders constructs of the same kind ahead of it
- [x] A node the edit removed takes its offset and its hidden state with it; neither passes to the node that takes its place in document order
- [x] A context, aggregate or slice hidden from the visibility tree stays hidden after the render, under the same edits, and the visibility tree shows it hidden
- [x] Reset layout stays enabled while a kept offset exists, and still restores the computed layout
- [x] Offsets and hidden state kept through a stale render apply when the source next parses
- [x] Opening a model by any route starts it with no offsets and nothing hidden

**Affected Files/Modules:**
- `internal/frontend/static/viewer.js` — a render of the panel's own source keeps the view state; an open resets it; Reset layout's enabled state follows the kept offsets
- `internal/frontend/static/model.js` — `setModelData` is where a new model's view state is reset
- `internal/frontend/tests/viewer.test.js`

**Patterns to Follow:**
- `internal/frontend/static/model.js:19-31`
- `internal/frontend/static/interaction.js:217-231`
- `internal/frontend/static/layout.js:155-162`
- `internal/frontend/static/viewer.js:617-621`
- `internal/export/diagram.go:99-113`
- `internal/frontend/tests/viewer.test.js:1742-1793`
- `internal/frontend/tests/viewer.test.js:145-169`
- `task:2`

**Testable:** Yes — a drag, a pan and the visibility tree are all driven through `init()` in the suite already, and each outcome reads off the rendered canvas.

**Certainty:** medium — offsets and hidden state are keyed by node id, read by layout and cleared by Reset layout (`layout.js:155-162`, `viewer.js:617-621`), but no render carries them from one parse's nodes to the next, and node ids are per-kind sequence numbers that shift with any edit ahead of a node (`export/diagram.go:99-113`).

**Blast radius:** low — view state only; the source, the save target and the file are untouched.

**Verification:** `task test:viewer` green. In the browser build, drag a node, pan and zoom, add a command above it in the panel, press Render, and see nothing move.

**Depends on:** Task 2

---

### Task 4: Revalidate the source panel automatically after a pause in typing

**Behavior:** Editing the source panel revalidates it once typing pauses — the badge and panel
update, source that parses redraws in place, source that does not keeps the stale diagram — with no
Render click, in all three distributions. The Render control stays and can be reached. A render
nobody clicked never overtakes an open, a save or a reported failure, and typing stays responsive
on the largest example.

**Acceptance Criteria:**
- [x] Every edit the user makes in the source panel — each change that fires its `input` event — revalidates the panel after a short pause, with no Render click; edits arriving within the pause produce one revalidation, and none starts before the pause ends
- [x] Each revalidation updates the badge and the diagnostics list as Task 1 describes, and does not re-open a diagnostics panel the user has closed
- [x] A revalidation leaves the data panel expanded, and leaves the source panel's text, caret and focus as the user left them
- [x] The Render control stays in the panel and renders at once when clicked; in the browser build, with the source panel expanded and diagnostics listed, `elementFromPoint` at the Render control and at the status beside it lands on them, not on the diagnostics panel
- [x] At any point in an open — queued behind another, waiting on the unsaved-edits question, or with its parse in flight — a revalidation waiting, parsing or starting never wins: once the open settles the canvas shows the opened model, the window names it, and the next Save writes to its path
- [x] A Save requested while a revalidation is waiting writes the panel's current text to the open file; a revalidation leaves the bar along the bottom as it is, so the save's confirmation survives the revalidation after it
- [x] A revalidation already waiting when the status area reports a failure — any reason a writer of `store.dom.statusEl` other than the render path puts there — does not replace that reason
- [x] A revalidation neither raises nor clears the unsaved-changes marker, and records nothing in the recent-files list
- [x] In the browser build, driven by the e2e-viewer suite, typing into the panel holding the largest model under `examples/`, once that model is on screen, records no main-thread task over 50 ms — the Long Tasks API threshold — across the keystrokes and the revalidations they start
- [x] The README's desktop section and `docs/wasm-architecture.md`'s flow summary describe revalidation as the panel is edited, the stale mark, and that the panel lists what `emod validate` reports; neither says the diagram waits for a Render click

**Affected Files/Modules:**
- `internal/frontend/static/viewer.js` — the pause, the revalidation it starts, and its order against opens, saves and reported failures
- `internal/frontend/static/config.js` — the pause's length, named
- `internal/frontend/static/store.js` — a revalidation waiting on its pause
- `internal/frontend/static/ui.js` — the diagnostics panel keeps the user's closing across revalidations
- `internal/frontend/static/viewer.html` — the diagnostics panel and the expanded data panel stop covering each other's controls
- `internal/frontend/tests/viewer.test.js`
- `e2e-viewer/tests/` — the Render control's reach, and the long-task measurement
- `README.md`, `docs/wasm-architecture.md`

**Patterns to Follow:**
- `internal/frontend/static/viewer.js:176-194`
- `internal/frontend/static/viewer.js:206-245`
- `internal/frontend/static/viewer.js:275-301`
- `internal/frontend/static/viewer.js:534-559`
- `internal/frontend/static/viewer.js:263-267`
- `internal/frontend/static/viewer.js:445-452`
- `internal/frontend/static/config.js:32`
- `internal/frontend/tests/viewer.test.js:1385-1546`
- `internal/frontend/tests/viewer.test.js:1953-2030`
- `internal/frontend/tests/viewer.test.js:152-156`
- `internal/frontend/tests/viewer.test.js:233-235`
- `e2e-viewer/tests/helpers.js:94-126`
- `e2e-viewer/tests/interaction.spec.js:41-70`
- `README.md:385-391`
- `docs/wasm-architecture.md:57-65`
- `tasks/learnings.md:1037-1042`, `:1079-1084`, `:1133-1138`, `:1043-1048`, `:959-964`, `:983-988`, `:1193-1198`
- `task:1`, `task:2`, `task:3`

**Testable:** Yes — the pause, the ordering and the status and bar rules through `init()` against the mocked seam; the Render control's reach and the long-task measurement in the e2e-viewer suite. The desktop window itself is smoke-tested by hand, as the story's non-goals require.

**Certainty:** low — no render in the viewer starts without a gesture; one started by a timer can begin inside an open's two-moment commit (`viewer.js:206-245`), a window no caller reaches at human speed today, and US-003 and US-005 each took several audit rounds to close windows of this shape (`tasks/learnings.md:1079-1084`, `:1133-1138`).

**Blast radius:** high — it decides whether an opened file becomes the save target when a revalidation overlaps the open; a wrong order makes the next Save overwrite one model's file with another model's source.

**Verification:** `task test:viewer` and `task test:e2e:viewer` green. `task build:desktop && ./bin/emod-desktop`: open `examples/all_patterns.emod`, type, and watch the badge and diagram follow without a click; press ⌘S mid-pause and check the file on disk and the confirmation; drop another model mid-pause and check the window's name and what the next ⌘S writes.

**Depends on:** Task 1, Task 2, Task 3

---

## Summary

**Four tasks**, ordered by dependency, and within that so every commit on the way is one a user
could live with: Tasks 1–3 change what a render of the panel's source reports, keeps and redraws,
reached through the Render control while they land; Task 4 starts those renders without a click. The
riskiest task is last rather than first because running it earlier would re-render on every pause
while a half-typed construct could still empty the canvas and every pause reset the user's layout.

- Task 1 stands alone and fixes what every render already reports, whether or not anything else lands.
- Task 2 builds on Task 1's parse request and its key guard.
- Task 3 needs Task 2 only for the stale render its sixth criterion carries view state through.
- Task 4 needs all three: it is what makes them happen on every pause.

**Story coverage.** All six criteria are covered, none deferred. Three carry a reading stated in full
above: "fails to parse" as the lexer or parser reporting anything, with source that parses but fails
validation re-rendering (Decision 2); "pan, zoom, and node layout" as every piece of view state a
render can reset, visibility included, with a node followed by what the source names it (Decision 5);
and "the same … the CLI reports" as the message with its rule name, the severity in the badge as well
as the list, the file's own name, and lines counted from the first byte (Decision 6).

**Assessments to overrule before they are built on.**

- **Task 4 — low certainty, high blast radius.** No render here has ever started without a gesture,
  and a timer-driven one can land inside an open's two-moment commit; getting the order wrong makes
  the next Save overwrite the wrong file. Expect the audit to spend its rounds here.
- **Task 2 — high blast radius.** It splits the branch that commits the save target, so the keep rule
  must not reach an open.
- **Decision 3's consequence** — pasted source that does not parse, landing over a diagram already on
  screen, now keeps that diagram (stale) instead of drawing the recovered fragment. It is what
  criterion 4 asks, and it narrows how US-002's "exactly as pasted source with the same errors does"
  reads; overrule it here if pasted source should always draw.
- **Decision 1** — continuous validation in all three distributions, not the desktop alone.
- **Two findings no story owns**: the LSP reports info as error (`internal/lsp/diagnostics.go:31-36`),
  and `--serve` drops its startup diagnostics (`viewer.js:714-716`).

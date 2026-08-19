# US-001: Render a model in a native desktop window

## Contents

1. Task 1: Rename `internal/wasm` to `internal/pipeline`
2. Task 2: Split `internal/viewer` into `internal/frontend` and `internal/viewer`
3. Task 3: Put the Go bridge behind a `platform.js` contract
4. Task 4: Read a dropped file through the platform contract
5. Task 5: Write an exported model through the platform contract
6. Task 6: Read the injected initial state through the platform contract
7. Task 7: Add the desktop model service over `internal/pipeline`
8. Task 8: Pin Wails v3 exactly and open a desktop window
9. Task 9: Render the shared frontend in the desktop window through `platform.desktop.js`
10. Task 10: Document the desktop distribution

The tasks fall into three independently mergeable slices, matching the proposal's phases
(`docs/proposals/emod-desktop-proposal.md:669-691`). Each slice leaves `main` shippable:

| Slice | Tasks | Exit |
|---|---|---|
| **Phase 0 — Restructure** (~0.5d) | 1, 2 | All existing tests green; zero behaviour change |
| **Phase 1 — Platform adapter** (~1d) | 3, 4, 5, 6 | CLI viewer and web bundle behave identically to today; no desktop code yet |
| **Phase 2 — Wails v3 shell** (~1.5d) | 7, 8, 9, 10 | A desktop window renders a diagram from pasted source |

---

## Story Reference

`user-stories/emod-desktop.md` → **US-001: Render a model in a native desktop window**
(`:23-36`, first story of "emod Desktop App"). Design detail:
`docs/proposals/emod-desktop-proposal.md`, of which this story spans three of the five phases
in §10 (`:669-691`).

The story's eight criteria and where they land:

| Story criterion | Task |
|---|---|
| Window opens with the same viewer interface — source panel, canvas, minimap, visibility toggles, diagnostics badge | 9 |
| Pasted `.emod` renders a diagram identical to the browser viewer's | 9, resting on 7 for the envelope equality |
| Pan, zoom, fit-to-view, selection, detail panel, layout reset, context actions behave as in the browser | 9 |
| Invalid source fills the badge and panel with the same messages, severities and locations | 9, resting on 7 |
| No network access, no local HTTP server, no listening port | 9 |
| `emod diagram --serve` and the published web viewer behave exactly as before | 1, 2, 3, 4, 5, 6 — each closes on the existing Go, vitest and Playwright suites |
| A shared viewer UI file change appears in all three distributions without a second copy | 2, 3, 9 |
| The framework version is pinned exactly in `go.mod` and the tool manifest | 8 |

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

Also out of scope, from the proposal and this decomposition:

- **`.goreleaser.yaml` and the existing cross-compile.** Untouched (§7.1, `:426-444`). The CLI and
  WASM builds stay `CGO_ENABLED=0`; only `cmd/emod-desktop` links CGO.
- **The string/envelope contract.** The desktop service hands JSON strings across the boundary and
  wraps errors the way the WASM shims do (§4.6, `:314-333`), so `model.js` and `emod-export.js`
  keep their bodies and only their import specifier changes.
- **A desktop window driver.** §8.2 (`:603-615`) and the story's Non-Goals both rule it out.

**Deferred** — wanted, but not here:

- **Native open and save dialogs, and save-in-place** → US-002, US-003. Task 9 lands the contract's
  file-open and file-save entry points on the desktop as an explicit "not available in this build"
  result, so the window never throws an unhandled rejection at a user who presses Export. Those two
  stories replace them.
- **Drag-drop on the desktop with a real path** → US-005. Task 4 puts a place for the path in the
  contract's result; the browser leaves it empty because a browser drop has none.
- **Packaging (`wails3 package`, `.app` bundle)** → US-007, proposal Phase 4 (`:698-702`).
- **CI desktop artifacts, the Linux runtime-dependency README section, the Ubuntu 24.04 floor** →
  US-015, US-014, proposal Phase 5 (`:703-711`).
- **Menu bar, recent files, file association, file watching** → US-004, US-006, US-008, US-013.
- **Live validation, diagnostics navigation, syntax highlighting in the source panel** → US-009,
  US-010, US-012. The desktop app renders on a Render click, exactly as the browser viewer does.
- **Embedding the LSP in the desktop app.** Proposal open question 6 (`:665-667`) assumes
  diagram-only; nothing here changes that.
- **`internal/frontend` as a published npm package.** Proposal open question 7 (`:668`) assumes no;
  copy-assembly is what Tasks 3 and 9 use.

---

## Codebase Context

Verified against the working tree on 2026-08-19. The proposal was written 2026-07-26 against branch
`us-002-construct-descriptions`; its inventory still holds, with line numbers shifted and the
frontend grown.

**Nothing has been started.** No `cmd/emod-desktop`, no `internal/frontend`, no `internal/pipeline`,
no `platform*.js`. `internal/wasm` and `internal/viewer` are in their pre-split shape, and
`go.mod` has no Wails requirement.

**`internal/wasm`** is two files — `pipeline.go` (100 lines) and `pipeline_test.go`. Its doc comment
(`:1-4`) already states it exists to be independent of `syscall/js`. Six exported functions:
`ExtractSource`, `RunPipelineExportDiagram`, `RunPipelineExportJSON`, `ExportEmod`,
`ExportEmodJSON`, `ErrorJSON`. One non-test importer: `cmd/emod-wasm/main.go:8`, which is 54 lines
of `js.Value` marshalling over them and nothing else.

**`internal/viewer`** conflates two concerns, as §5.2 says (`:351-368`):

- shared frontend — `static/` (15 files, 4,302 lines of JS plus `viewer.html`), `tests/` (24 vitest
  files), `embed.go`, `package.json`, `package-lock.json`, `vitest.config.js`, and the gitignored
  `generated/` the WASM build writes into
- CLI delivery — `serve.go` (101 lines) and `serve_test.go`

`embed.go:7` is `//go:embed static/* generated/*`, so `generated/` has to sit beside `static/` in
whichever package holds `embed.go`. §5.2's file list does not mention it. `serve.go` routes
`/static/` and `/generated/` off that FS (`:31-40`) and injects `window.INITIAL_DATA` into the
`<!--INITIAL_DATA-->` marker at `static/viewer.html:1162` through `buildHTML` (`:81-101`), escaping
`</` so a model spelling `</script>` cannot end the block. `internal/cli/diagram.go:195` is the only
caller of `ServeViewer`.

**The seam, at today's lines.** Five touch points across three files, as §4.1 claims (`:142-154`):

| # | What | Where |
|---|---|---|
| 1 | Go bridge | `static/wasm.js`, still exactly 71 lines: `fetch('generated/emod.wasm')` at `:15`, `instantiateStreaming` with an `arrayBuffer` fallback at `:22-32`, exports `parseEmod, exportEmod, ready, isReady` at `:71` |
| 2 | Drag-drop open | `viewer.js:166-188` — `evt.dataTransfer.files[0]` (`:170`), the `.emod`/`.json` check and its message (`:172-177`), `new FileReader()` (`:178`), `readAsText` (`:187`) |
| 3 | Export/save | `viewer.js:208-222` — `new Blob` (`:210`), `createObjectURL` (`:211`), `a.download` (`:214`), `revokeObjectURL` (`:218`) |
| 4 | Initial state | `viewer.js:291-292` reads the `INITIAL_DATA` global that `serve.go:95-96` injects |
| 5 | Readiness gate | `viewer.js:302` (`isReady`) and `:307` (`ready.then`), plus the deferred dynamic import at `model.js:104` |

**Exactly three importers of `wasm.js`**, as §4.2 claims (`:155-170`): `viewer.js:12` and
`emod-export.js:1` statically, `model.js:104` dynamically ("dynamic import to defer init side
effects").

**Seven test files reach the bridge**, as §8.1 claims (`:592-602`):

- `wasm.test.js`, `wasm.fallback.test.js`, `wasm.init-success.test.js`, `wasm.init-http-error.test.js`
  import `../static/wasm.js` directly and drive it through `vi.hoisted` globals — these four test
  WASM fetch, instantiate and fallback specifically and are browser-only behaviour
- `viewer.test.js:13` and `model.test.js:5` `vi.mock('../static/wasm.js', …)`
- `emod-export.test.js:11` `vi.doMock`s it inside a `vi.resetModules()` helper so each test controls
  when `ready` settles

`viewer.test.js` also owns the drop path: `fireDrop` (`:65-72`) stubs `dataTransfer`, `MockFileReader`
(`:93-102`) stands in for the browser read and is installed as `globalThis.FileReader` (`:132`), and
three leaves at `:191-233` cover the rejected extension, a dropped `.emod` and a dropped `.json`.
There is no vitest leaf for the export button.

**What guards the browser end to end.** `e2e-viewer/` is Playwright over the real `task build:web`
bundle with the real WASM module. `open()` (`helpers.js:96-100`) waits for
`typeof globalThis.parseEmod === 'function'`, so it will still gate on the WASM global in the browser
build. `exportEmod()` (`:129-137`) clicks Export and reads the download stream, and
`export.spec.js:8-23` asserts both the exported bytes equal `SAMPLE` and the download is named
`Billing.emod`. Nothing there drops a file. `task test:e2e:viewer` depends on `build:web`, so a
broken web bundle fails CI before the Pages deploy job runs.

**The build.** `Taskfile.yml`: `build` (`:7-15`, `CGO_ENABLED=0`, deps `build:wasm`), `build:wasm`
(`:33-42`, writes `emod.wasm` and copies `wasm_exec.js` into `internal/viewer/generated/`),
`build:web` (`:44-52`, copies `static/*` and `generated/*` into `web/` then
`mv web/static/viewer.html web/index.html`), `test:unit` and `test:integration`
(`:63-71`, `go test -tags <tag> $(go list ./... | grep -v /cmd/emod-wasm)` — the existing precedent
for excluding a package that cannot be compiled in the default configuration), `test:viewer` and
`test:viewer:deps` (`:73-91`, both `dir: internal/viewer`).

`mise.toml` pins four tools exactly and nothing floats. `.gitignore` carries `/web` and
`internal/viewer/generated/` — the generated-vs-source discipline §2.4 relies on (`:84-91`).
`.github/workflows/ci.yml:31` reads `node-version-file: 'internal/viewer/package.json'`.
`.goreleaser.yaml` builds `./cmd/emod` only, `CGO_ENABLED=0`, linux+darwin cross-compiled from one
runner.

**Documentation that names the moving paths.** `docs/architecture.md:213-241` (the browser-viewer
section, its mermaid diagram and the `internal/wasm` paragraph) and `docs/wasm-architecture.md:9,19,50,59`.
Both go stale the moment Tasks 1 and 2 land.

---

## Findings

**F1 — The CLI's viewer flag is `--serve`.** It is spelled `--serve`
(`internal/cli/app.go:132-135`, `Usage: "Start viewer server with diagram data"`) and has been
since it was introduced; `emod diagram --viewer` names nothing the CLI accepts. Task criteria and
verification commands here say `--serve`. Nothing in this story changes the flag, and renaming it
is not in scope.

**F2 — `generated/` has to move with `embed.go`, and §5.2 does not say so.** `embed.go:7` names
`static/*` *and* `generated/*`, so splitting the package moves the WASM build's output directory
too: `Taskfile.yml:36-38`'s three paths, `.gitignore`'s `internal/viewer/generated/` line, and
`build:web`'s copy source (`:51`) all move with it. A split that moves only the files §5.2 lists
leaves a package whose embed directive cannot resolve, and `//go:embed` fails at compile time when a
pattern matches nothing — the same coupling `docs/architecture.md:238-240` already warns about
("run `task build` (not bare `go build`)").

**F3 — `cmd/emod-desktop` must be excluded from `test:unit` and `test:integration`, or CI breaks.**
Both tasks run `go test` over `$(go list ./... | grep -v /cmd/emod-wasm)`. A new package that
imports Wails needs CGO and the GTK4/WebKitGTK development headers to compile, which
`ubuntu-latest` does not have and which §7.1 (`:426-444`) explicitly keeps out of the shared build
paths. The package also embeds an assembled, gitignored frontend directory, so it cannot compile at
all before its build task has run. The exclusion is the same shape `cmd/emod-wasm` already has, and
it belongs in the task that creates the package.

**F4 — `platform.js` must exist on disk for the vitest suite to resolve.** Once `viewer.js` does
`import … from './platform.js'`, Vite resolves that specifier when the suite loads the module under
test, and `vi.mock('../static/platform.js', factory)` does not create a file. So a `platform.js`
that exists only after a build task has run makes `task test:viewer` depend on that build task —
which today it does not (`Taskfile.yml:73-79` deps only on `test:viewer:deps`). See "Open questions,
decided" Q1.

**F5 — the desktop adapter must not ship to Pages, and `static/*` is copied wholesale.**
`build:web` copies `./internal/viewer/static/*` (`Taskfile.yml:50`) and `embed.go` embeds `static/*`.
If `platform.desktop.js` lives in that directory, the Pages deploy serves it and the CLI binary
embeds it — the opposite of §4.5's stated result, "no desktop code in the Pages deploy" (`:250`).
Task 3 closes on the outcome (`web/static/` carries exactly one platform implementation) rather than
on a placement, because either excluding the file from the copy or keeping it outside `static/`
achieves it.

**F6 — §4.3's `openFile()` takes no argument, but the browser's only file source is a drop event.**
The contract (`:171-188`) lists `openFile()` → `Promise<{path, content} | null>`, which fits a native
dialog and US-002. In the browser today there is no Open control at all: the only way a file enters
the viewer is `viewer.js:166-188`, which already holds the file object by the time it needs reading.
Task 4 therefore decides the shape the drop handler calls through, and its criteria are written
against the observable drop behaviour rather than against a signature.

**F7 — a globally installed `wails3` would shadow the repo's pin.** `tasks/learnings.md:11-14`
records this exact failure with `tree-sitter-cli`: a tool pinned in `mise.toml` loses to a global
pin on `PATH` in a non-activated shell, silently, producing different output while the tree looks
clean. A new pinned CLI whose output feeds a build is the same hazard. Every verification command
in Phase 2 goes through `mise exec --`.

**F8 — the desktop page must be derived from `viewer.html`, not written.** The story's seventh
criterion is the one that keeps the three distributions from forking. `viewer.html` carries two
browser-only lines — `<script src="generated/wasm_exec.js">` (`:1161`) and the
`<!--INITIAL_DATA-->` marker (`:1162`) — so the desktop assembly has to produce its page from that
file rather than beside it. `build:web` already does the same class of thing (`mv viewer.html
index.html`, `Taskfile.yml:52`).

---

## Open questions, decided

**Q1 — is `platform.js` a committed file or a build output?** *Decided: either, as long as
`task test:viewer` passes from a checkout where no build task has run (F4), `task build` yields a
CLI viewer that resolves it, and `web/static/` ends up with the browser implementation and nothing
desktop-specific (F5).* Two shapes satisfy all three: a committed `platform.js` that re-exports the
browser implementation, overwritten only inside the desktop app's own assembled directory; or a
gitignored `platform.js` produced by an assembly step that `build`, `build:web` and `test:viewer`
all depend on. §4.5 (`:231-251`) recommends assembly-time copy and rejects runtime detection; both
shapes are assembly-time, and the choice between them is a Taskfile question, not a design one.
Whichever is chosen, the desktop target's copy is assembled — Task 9's criteria require it.

**Q2 — where does `ModelService` live?** *Decided: in `internal/`, not `cmd/emod-desktop/`.*
§5.3 (`:369-392`) sketches it as `cmd/emod-desktop/service.go`. But a `package main` there cannot be
built or tested without the framework, CGO and the assembled frontend, and F3 removes that package
from `task test:unit` for exactly those reasons — which would take the service's tests with it. The
service as §4.6 writes it (`:314-333`) imports nothing from the framework: it is `internal/pipeline`
calls and string envelopes. Keeping it in `internal/` is what lets Task 7 land tested, ahead of and
independent of the alpha dependency, and it does not weaken §9.1's containment claim
(`:660-663`) — no v3-specific code moves out of `cmd/emod-desktop/`.

**Q3 — what do the desktop's file-open and file-save entry points do in this story?** *Decided:
exist, and report that they are not available in this build.* US-002 and US-003 own them.
`platform.desktop.js` must export the same names as `platform.browser.js` or the shared modules
break on import, and a silent no-op or an unhandled rejection when a user presses Export is worse
than a status-area message. Task 9 states this and Boundaries records it as deferred.

**Q4 — which Wails version?** *Decided: whatever is current at implementation time, pinned exactly
in both places.* The proposal verified `v3.0.0-alpha2.117` on 8 Jul 2026 (`:629-631`); this
breakdown is written 2026-08-19 and the alpha tags move nightly. Task 8's criterion is that the two
pins name the same exact version and that neither floats — not that they name a particular one.
Read the pinned version's own package docs for the Manager API surface; §9.1 (`:641-650`) records
that earlier alphas' flat top-level functions are gone and that third-party tutorials are already
stale.

---

## Theory

The whole story is one structural move performed three times, and the reason it is cheap is that the
repo already put a seam where the seam needs to go. `wasm.js` is 71 lines with three importers and
seven test files mocking it; every test that stubs it is already stubbing *the boundary between the
UI and the Go core*, not the browser. So Phase 1 is not "introduce an abstraction" — it is "rename
the abstraction that exists to what it is, and move three more browser-shaped things behind it".
Read Tasks 4, 5 and 6 as three small subtractions from `viewer.js`: after them that file names no
browser API that a native shell cannot provide, and it is the only file that had any.

Phase 0 is the part that looks like busywork and is not. `internal/viewer` currently means both "the
UI every distribution shares" and "the localhost server one distribution uses"; a desktop app that
imports it would be importing an HTTP server it must never start. Splitting it makes the dependency
the desktop app is allowed to have expressible. The trap is `generated/` (F2): it is not in §5.2's
list, it is gitignored, and `embed.go` names it — so the split is not "move four things", it is
"move the assets *and* redirect the build output that lands among them", and a half-done version
fails at `//go:embed` rather than at a test.

The two places to read hardest are the ones nothing in the tree can catch. First, F5: `static/*` is
copied wholesale by `build:web` and embedded wholesale by `embed.go`, so a desktop adapter parked in
that directory is shipped to GitHub Pages and baked into the CLI binary, and no test anywhere would
notice. Second, F3: `task test:unit` enumerates packages with `go list ./...`, so the moment
`cmd/emod-desktop` exists it is in the set, and CI — which installs no WebKit headers — starts
failing for reasons that read as unrelated to the commit that caused them. Both are one-line
mistakes with no test surface, which is why both are acceptance criteria rather than notes.

The alpha is the only genuine unknown, and it is deliberately isolated into Task 8: a pinned
dependency and a window, with no adapter, no bindings and no shared assets in the commit. If the
framework's API has moved since the proposal was written, that is where it surfaces — as a build
failure in a task whose whole diff is a `go.mod` line, a `mise.toml` line, a Taskfile target and a
`main.go`, rather than tangled into the frontend assembly. Task 9's blank-page risk is the mirror
image: a webview that fails to import one module renders nothing and says nothing, so verify it by
what the window *draws*, not by what the build exits with.

---

## Tasks

---

**Phase 0 — Restructure (~0.5d). Tasks 1 and 2. Exit: all existing tests green, zero behaviour
change. Mergeable on its own.**

### Task 1: Rename `internal/wasm` to `internal/pipeline`

**Behavior:** The package that orchestrates lex → parse → validate → lint → export is named for what
it does rather than for the first caller it happened to have, so a native caller can import it
without importing a lie. Nothing observable changes: the same functions, the same JSON envelopes, the
same three WASM globals.

**Acceptance Criteria:**
- [x] `internal/pipeline/` holds what `internal/wasm/` holds today, declaring `package pipeline`, and
      `internal/wasm/` no longer exists
- [x] The package doc comment names `pipeline` and describes what the package orchestrates rather
      than which caller it was extracted for (current text at `internal/wasm/pipeline.go:1-4`)
- [x] `cmd/emod-wasm/main.go` imports the renamed package and every `wasm.` qualifier in it reads
      `pipeline.` (`:8`, `:20`, `:24`, `:32`, `:35`, `:43`, `:45`, `:50`)
- [x] No exported identifier changes spelling — `ExtractSource`, `RunPipelineExportDiagram`,
      `RunPipelineExportJSON`, `ExportEmod`, `ExportEmodJSON`, `ErrorJSON` — so no JSON envelope key
      and no `js.Global()` name moves
- [x] `internal/pipeline/pipeline_test.go` passes with no assertion weakened and no subtest skipped
- [x] No Go file, no `Taskfile.yml` entry, no workflow and no file under `docs/` other than
      `docs/proposals/` names `internal/wasm`
- [x] `docs/architecture.md:222` and `:237` name the renamed package
- [x] `mise exec -- task build` still produces `internal/viewer/generated/emod.wasm`

**Affected Files/Modules:**
- `internal/wasm/pipeline.go`, `internal/wasm/pipeline_test.go` → `internal/pipeline/` — package
  clause, doc comment, test import and every qualifier
- `cmd/emod-wasm/main.go` — the single non-test importer
- `docs/architecture.md` — `:222` and `:237`

**Patterns to Follow:**
- `tasks/learnings.md:241-244` — the rename discipline this repo already applies: the Go identifier
  moves, the wire name does not
- `internal/wasm/pipeline.go:1-4` — the doc comment being replaced, and the reason it now misleads
  (`docs/proposals/emod-desktop-proposal.md:345-350`)
- `cmd/emod-wasm/main.go:19-54` — every call site, all in one file

**Testable:** Yes — `internal/pipeline`'s six exported functions are exercised by the moved
`pipeline_test.go`, which runs under `task test:unit`.

**Certainty:** high — a compiler-proven rename with one non-test importer (`cmd/emod-wasm/main.go:8`),
following the rename discipline `tasks/learnings.md:241-244` already records.

**Blast radius:** low — an `internal/` package rename with every exported name and every JSON
envelope unchanged; nothing outside the repository can observe it.

**Verification:** `mise exec -- task build`; `mise exec -- task test:unit`;
`mise exec -- task test:integration`;
`rg -n 'internal/wasm' -g '!docs/proposals' -g '!tasks'` prints nothing.

**Depends on:** None.

---

**Phase 1 — Platform adapter** (~1d) | 3, 4, 5, 6 | CLI viewer and web bundle behave identically to today; no desktop code yet |
| **Phase 2 — Wails v3 shell** (~1.5d) | 7, 8, 9, 10 | A desktop window renders a diagram from pasted source |

---

## Story Reference

`user-stories/emod-desktop.md` → **US-001: Render a model in a native desktop window**
(`:23-36`, first story of "emod Desktop App"). Design detail:
`docs/proposals/emod-desktop-proposal.md`, of which this story spans three of the five phases
in §10 (`:669-691`).

The story's eight criteria and where they land:

| Story criterion | Task |
|---|---|
| Window opens with the same viewer interface — source panel, canvas, minimap, visibility toggles, diagnostics badge | 9 |
| Pasted `.emod` renders a diagram identical to the browser viewer's | 9, resting on 7 for the envelope equality |
| Pan, zoom, fit-to-view, selection, detail panel, layout reset, context actions behave as in the browser | 9 |
| Invalid source fills the badge and panel with the same messages, severities and locations | 9, resting on 7 |
| No network access, no local HTTP server, no listening port | 9 |
| `emod diagram --serve` and the published web viewer behave exactly as before | 1, 2, 3, 4, 5, 6 — each closes on the existing Go, vitest and Playwright suites |
| A shared viewer UI file change appears in all three distributions without a second copy | 2, 3, 9 |
| The framework version is pinned exactly in `go.mod` and the tool manifest | 8 |

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

Also out of scope, from the proposal and this decomposition:

- **`.goreleaser.yaml` and the existing cross-compile.** Untouched (§7.1, `:426-444`). The CLI and
  WASM builds stay `CGO_ENABLED=0`; only `cmd/emod-desktop` links CGO.
- **The string/envelope contract.** The desktop service hands JSON strings across the boundary and
  wraps errors the way the WASM shims do (§4.6, `:314-333`), so `model.js` and `emod-export.js`
  keep their bodies and only their import specifier changes.
- **A desktop window driver.** §8.2 (`:603-615`) and the story's Non-Goals both rule it out.

**Deferred** — wanted, but not here:

- **Native open and save dialogs, and save-in-place** → US-002, US-003. Task 9 lands the contract's
  file-open and file-save entry points on the desktop as an explicit "not available in this build"
  result, so the window never throws an unhandled rejection at a user who presses Export. Those two
  stories replace them.
- **Drag-drop on the desktop with a real path** → US-005. Task 4 puts a place for the path in the
  contract's result; the browser leaves it empty because a browser drop has none.
- **Packaging (`wails3 package`, `.app` bundle)** → US-007, proposal Phase 4 (`:698-702`).
- **CI desktop artifacts, the Linux runtime-dependency README section, the Ubuntu 24.04 floor** →
  US-015, US-014, proposal Phase 5 (`:703-711`).
- **Menu bar, recent files, file association, file watching** → US-004, US-006, US-008, US-013.
- **Live validation, diagnostics navigation, syntax highlighting in the source panel** → US-009,
  US-010, US-012. The desktop app renders on a Render click, exactly as the browser viewer does.
- **Embedding the LSP in the desktop app.** Proposal open question 6 (`:665-667`) assumes
  diagram-only; nothing here changes that.
- **`internal/frontend` as a published npm package.** Proposal open question 7 (`:668`) assumes no;
  copy-assembly is what Tasks 3 and 9 use.

---

## Codebase Context

Verified against the working tree on 2026-08-19. The proposal was written 2026-07-26 against branch
`us-002-construct-descriptions`; its inventory still holds, with line numbers shifted and the
frontend grown.

**Nothing has been started.** No `cmd/emod-desktop`, no `internal/frontend`, no `internal/pipeline`,
no `platform*.js`. `internal/wasm` and `internal/viewer` are in their pre-split shape, and
`go.mod` has no Wails requirement.

**`internal/wasm`** is two files — `pipeline.go` (100 lines) and `pipeline_test.go`. Its doc comment
(`:1-4`) already states it exists to be independent of `syscall/js`. Six exported functions:
`ExtractSource`, `RunPipelineExportDiagram`, `RunPipelineExportJSON`, `ExportEmod`,
`ExportEmodJSON`, `ErrorJSON`. One non-test importer: `cmd/emod-wasm/main.go:8`, which is 54 lines
of `js.Value` marshalling over them and nothing else.

**`internal/viewer`** conflates two concerns, as §5.2 says (`:351-368`):

- shared frontend — `static/` (15 files, 4,302 lines of JS plus `viewer.html`), `tests/` (24 vitest
  files), `embed.go`, `package.json`, `package-lock.json`, `vitest.config.js`, and the gitignored
  `generated/` the WASM build writes into
- CLI delivery — `serve.go` (101 lines) and `serve_test.go`

`embed.go:7` is `//go:embed static/* generated/*`, so `generated/` has to sit beside `static/` in
whichever package holds `embed.go`. §5.2's file list does not mention it. `serve.go` routes
`/static/` and `/generated/` off that FS (`:31-40`) and injects `window.INITIAL_DATA` into the
`<!--INITIAL_DATA-->` marker at `static/viewer.html:1162` through `buildHTML` (`:81-101`), escaping
`</` so a model spelling `</script>` cannot end the block. `internal/cli/diagram.go:195` is the only
caller of `ServeViewer`.

**The seam, at today's lines.** Five touch points across three files, as §4.1 claims (`:142-154`):

| # | What | Where |
|---|---|---|
| 1 | Go bridge | `static/wasm.js`, still exactly 71 lines: `fetch('generated/emod.wasm')` at `:15`, `instantiateStreaming` with an `arrayBuffer` fallback at `:22-32`, exports `parseEmod, exportEmod, ready, isReady` at `:71` |
| 2 | Drag-drop open | `viewer.js:166-188` — `evt.dataTransfer.files[0]` (`:170`), the `.emod`/`.json` check and its message (`:172-177`), `new FileReader()` (`:178`), `readAsText` (`:187`) |
| 3 | Export/save | `viewer.js:208-222` — `new Blob` (`:210`), `createObjectURL` (`:211`), `a.download` (`:214`), `revokeObjectURL` (`:218`) |
| 4 | Initial state | `viewer.js:291-292` reads the `INITIAL_DATA` global that `serve.go:95-96` injects |
| 5 | Readiness gate | `viewer.js:302` (`isReady`) and `:307` (`ready.then`), plus the deferred dynamic import at `model.js:104` |

**Exactly three importers of `wasm.js`**, as §4.2 claims (`:155-170`): `viewer.js:12` and
`emod-export.js:1` statically, `model.js:104` dynamically ("dynamic import to defer init side
effects").

**Seven test files reach the bridge**, as §8.1 claims (`:592-602`):

- `wasm.test.js`, `wasm.fallback.test.js`, `wasm.init-success.test.js`, `wasm.init-http-error.test.js`
  import `../static/wasm.js` directly and drive it through `vi.hoisted` globals — these four test
  WASM fetch, instantiate and fallback specifically and are browser-only behaviour
- `viewer.test.js:13` and `model.test.js:5` `vi.mock('../static/wasm.js', …)`
- `emod-export.test.js:11` `vi.doMock`s it inside a `vi.resetModules()` helper so each test controls
  when `ready` settles

`viewer.test.js` also owns the drop path: `fireDrop` (`:65-72`) stubs `dataTransfer`, `MockFileReader`
(`:93-102`) stands in for the browser read and is installed as `globalThis.FileReader` (`:132`), and
three leaves at `:191-233` cover the rejected extension, a dropped `.emod` and a dropped `.json`.
There is no vitest leaf for the export button.

**What guards the browser end to end.** `e2e-viewer/` is Playwright over the real `task build:web`
bundle with the real WASM module. `open()` (`helpers.js:96-100`) waits for
`typeof globalThis.parseEmod === 'function'`, so it will still gate on the WASM global in the browser
build. `exportEmod()` (`:129-137`) clicks Export and reads the download stream, and
`export.spec.js:8-23` asserts both the exported bytes equal `SAMPLE` and the download is named
`Billing.emod`. Nothing there drops a file. `task test:e2e:viewer` depends on `build:web`, so a
broken web bundle fails CI before the Pages deploy job runs.

**The build.** `Taskfile.yml`: `build` (`:7-15`, `CGO_ENABLED=0`, deps `build:wasm`), `build:wasm`
(`:33-42`, writes `emod.wasm` and copies `wasm_exec.js` into `internal/viewer/generated/`),
`build:web` (`:44-52`, copies `static/*` and `generated/*` into `web/` then
`mv web/static/viewer.html web/index.html`), `test:unit` and `test:integration`
(`:63-71`, `go test -tags <tag> $(go list ./... | grep -v /cmd/emod-wasm)` — the existing precedent
for excluding a package that cannot be compiled in the default configuration), `test:viewer` and
`test:viewer:deps` (`:73-91`, both `dir: internal/viewer`).

`mise.toml` pins four tools exactly and nothing floats. `.gitignore` carries `/web` and
`internal/viewer/generated/` — the generated-vs-source discipline §2.4 relies on (`:84-91`).
`.github/workflows/ci.yml:31` reads `node-version-file: 'internal/viewer/package.json'`.
`.goreleaser.yaml` builds `./cmd/emod` only, `CGO_ENABLED=0`, linux+darwin cross-compiled from one
runner.

**Documentation that names the moving paths.** `docs/architecture.md:213-241` (the browser-viewer
section, its mermaid diagram and the `internal/wasm` paragraph) and `docs/wasm-architecture.md:9,19,50,59`.
Both go stale the moment Tasks 1 and 2 land.

---

## Findings

**F1 — The CLI's viewer flag is `--serve`.** It is spelled `--serve`
(`internal/cli/app.go:132-135`, `Usage: "Start viewer server with diagram data"`) and has been
since it was introduced; `emod diagram --viewer` names nothing the CLI accepts. Task criteria and
verification commands here say `--serve`. Nothing in this story changes the flag, and renaming it
is not in scope.

**F2 — `generated/` has to move with `embed.go`, and §5.2 does not say so.** `embed.go:7` names
`static/*` *and* `generated/*`, so splitting the package moves the WASM build's output directory
too: `Taskfile.yml:36-38`'s three paths, `.gitignore`'s `internal/viewer/generated/` line, and
`build:web`'s copy source (`:51`) all move with it. A split that moves only the files §5.2 lists
leaves a package whose embed directive cannot resolve, and `//go:embed` fails at compile time when a
pattern matches nothing — the same coupling `docs/architecture.md:238-240` already warns about
("run `task build` (not bare `go build`)").

**F3 — `cmd/emod-desktop` must be excluded from `test:unit` and `test:integration`, or CI breaks.**
Both tasks run `go test` over `$(go list ./... | grep -v /cmd/emod-wasm)`. A new package that
imports Wails needs CGO and the GTK4/WebKitGTK development headers to compile, which
`ubuntu-latest` does not have and which §7.1 (`:426-444`) explicitly keeps out of the shared build
paths. The package also embeds an assembled, gitignored frontend directory, so it cannot compile at
all before its build task has run. The exclusion is the same shape `cmd/emod-wasm` already has, and
it belongs in the task that creates the package.

**F4 — `platform.js` must exist on disk for the vitest suite to resolve.** Once `viewer.js` does
`import … from './platform.js'`, Vite resolves that specifier when the suite loads the module under
test, and `vi.mock('../static/platform.js', factory)` does not create a file. So a `platform.js`
that exists only after a build task has run makes `task test:viewer` depend on that build task —
which today it does not (`Taskfile.yml:73-79` deps only on `test:viewer:deps`). See "Open questions,
decided" Q1.

**F5 — the desktop adapter must not ship to Pages, and `static/*` is copied wholesale.**
`build:web` copies `./internal/viewer/static/*` (`Taskfile.yml:50`) and `embed.go` embeds `static/*`.
If `platform.desktop.js` lives in that directory, the Pages deploy serves it and the CLI binary
embeds it — the opposite of §4.5's stated result, "no desktop code in the Pages deploy" (`:250`).
Task 3 closes on the outcome (`web/static/` carries exactly one platform implementation) rather than
on a placement, because either excluding the file from the copy or keeping it outside `static/`
achieves it.

**F6 — §4.3's `openFile()` takes no argument, but the browser's only file source is a drop event.**
The contract (`:171-188`) lists `openFile()` → `Promise<{path, content} | null>`, which fits a native
dialog and US-002. In the browser today there is no Open control at all: the only way a file enters
the viewer is `viewer.js:166-188`, which already holds the file object by the time it needs reading.
Task 4 therefore decides the shape the drop handler calls through, and its criteria are written
against the observable drop behaviour rather than against a signature.

**F7 — a globally installed `wails3` would shadow the repo's pin.** `tasks/learnings.md:11-14`
records this exact failure with `tree-sitter-cli`: a tool pinned in `mise.toml` loses to a global
pin on `PATH` in a non-activated shell, silently, producing different output while the tree looks
clean. A new pinned CLI whose output feeds a build is the same hazard. Every verification command
in Phase 2 goes through `mise exec --`.

**F8 — the desktop page must be derived from `viewer.html`, not written.** The story's seventh
criterion is the one that keeps the three distributions from forking. `viewer.html` carries two
browser-only lines — `<script src="generated/wasm_exec.js">` (`:1161`) and the
`<!--INITIAL_DATA-->` marker (`:1162`) — so the desktop assembly has to produce its page from that
file rather than beside it. `build:web` already does the same class of thing (`mv viewer.html
index.html`, `Taskfile.yml:52`).

---

## Open questions, decided

**Q1 — is `platform.js` a committed file or a build output?** *Decided: either, as long as
`task test:viewer` passes from a checkout where no build task has run (F4), `task build` yields a
CLI viewer that resolves it, and `web/static/` ends up with the browser implementation and nothing
desktop-specific (F5).* Two shapes satisfy all three: a committed `platform.js` that re-exports the
browser implementation, overwritten only inside the desktop app's own assembled directory; or a
gitignored `platform.js` produced by an assembly step that `build`, `build:web` and `test:viewer`
all depend on. §4.5 (`:231-251`) recommends assembly-time copy and rejects runtime detection; both
shapes are assembly-time, and the choice between them is a Taskfile question, not a design one.
Whichever is chosen, the desktop target's copy is assembled — Task 9's criteria require it.

**Q2 — where does `ModelService` live?** *Decided: in `internal/`, not `cmd/emod-desktop/`.*
§5.3 (`:369-392`) sketches it as `cmd/emod-desktop/service.go`. But a `package main` there cannot be
built or tested without the framework, CGO and the assembled frontend, and F3 removes that package
from `task test:unit` for exactly those reasons — which would take the service's tests with it. The
service as §4.6 writes it (`:314-333`) imports nothing from the framework: it is `internal/pipeline`
calls and string envelopes. Keeping it in `internal/` is what lets Task 7 land tested, ahead of and
independent of the alpha dependency, and it does not weaken §9.1's containment claim
(`:660-663`) — no v3-specific code moves out of `cmd/emod-desktop/`.

**Q3 — what do the desktop's file-open and file-save entry points do in this story?** *Decided:
exist, and report that they are not available in this build.* US-002 and US-003 own them.
`platform.desktop.js` must export the same names as `platform.browser.js` or the shared modules
break on import, and a silent no-op or an unhandled rejection when a user presses Export is worse
than a status-area message. Task 9 states this and Boundaries records it as deferred.

**Q4 — which Wails version?** *Decided: whatever is current at implementation time, pinned exactly
in both places.* The proposal verified `v3.0.0-alpha2.117` on 8 Jul 2026 (`:629-631`); this
breakdown is written 2026-08-19 and the alpha tags move nightly. Task 8's criterion is that the two
pins name the same exact version and that neither floats — not that they name a particular one.
Read the pinned version's own package docs for the Manager API surface; §9.1 (`:641-650`) records
that earlier alphas' flat top-level functions are gone and that third-party tutorials are already
stale.

---

## Theory

The whole story is one structural move performed three times, and the reason it is cheap is that the
repo already put a seam where the seam needs to go. `wasm.js` is 71 lines with three importers and
seven test files mocking it; every test that stubs it is already stubbing *the boundary between the
UI and the Go core*, not the browser. So Phase 1 is not "introduce an abstraction" — it is "rename
the abstraction that exists to what it is, and move three more browser-shaped things behind it".
Read Tasks 4, 5 and 6 as three small subtractions from `viewer.js`: after them that file names no
browser API that a native shell cannot provide, and it is the only file that had any.

Phase 0 is the part that looks like busywork and is not. `internal/viewer` currently means both "the
UI every distribution shares" and "the localhost server one distribution uses"; a desktop app that
imports it would be importing an HTTP server it must never start. Splitting it makes the dependency
the desktop app is allowed to have expressible. The trap is `generated/` (F2): it is not in §5.2's
list, it is gitignored, and `embed.go` names it — so the split is not "move four things", it is
"move the assets *and* redirect the build output that lands among them", and a half-done version
fails at `//go:embed` rather than at a test.

The two places to read hardest are the ones nothing in the tree can catch. First, F5: `static/*` is
copied wholesale by `build:web` and embedded wholesale by `embed.go`, so a desktop adapter parked in
that directory is shipped to GitHub Pages and baked into the CLI binary, and no test anywhere would
notice. Second, F3: `task test:unit` enumerates packages with `go list ./...`, so the moment
`cmd/emod-desktop` exists it is in the set, and CI — which installs no WebKit headers — starts
failing for reasons that read as unrelated to the commit that caused them. Both are one-line
mistakes with no test surface, which is why both are acceptance criteria rather than notes.

The alpha is the only genuine unknown, and it is deliberately isolated into Task 8: a pinned
dependency and a window, with no adapter, no bindings and no shared assets in the commit. If the
framework's API has moved since the proposal was written, that is where it surfaces — as a build
failure in a task whose whole diff is a `go.mod` line, a `mise.toml` line, a Taskfile target and a
`main.go`, rather than tangled into the frontend assembly. Task 9's blank-page risk is the mirror
image: a webview that fails to import one module renders nothing and says nothing, so verify it by
what the window *draws*, not by what the build exits with.

---

## Tasks

---

**Phase 0 — Restructure (~0.5d). Tasks 1 and 2. Exit: all existing tests green, zero behaviour
change. Mergeable on its own.**

### Task 1: Rename `internal/wasm` to `internal/pipeline`

**Behavior:** The package that orchestrates lex → parse → validate → lint → export is named for what
it does rather than for the first caller it happened to have, so a native caller can import it
without importing a lie. Nothing observable changes: the same functions, the same JSON envelopes, the
same three WASM globals.

**Acceptance Criteria:**
- [x] `internal/pipeline/` holds what `internal/wasm/` holds today, declaring `package pipeline`, and
      `internal/wasm/` no longer exists
- [x] The package doc comment names `pipeline` and describes what the package orchestrates rather
      than which caller it was extracted for (current text at `internal/wasm/pipeline.go:1-4`)
- [x] `cmd/emod-wasm/main.go` imports the renamed package and every `wasm.` qualifier in it reads
      `pipeline.` (`:8`, `:20`, `:24`, `:32`, `:35`, `:43`, `:45`, `:50`)
- [x] No exported identifier changes spelling — `ExtractSource`, `RunPipelineExportDiagram`,
      `RunPipelineExportJSON`, `ExportEmod`, `ExportEmodJSON`, `ErrorJSON` — so no JSON envelope key
      and no `js.Global()` name moves
- [x] `internal/pipeline/pipeline_test.go` passes with no assertion weakened and no subtest skipped
- [x] No Go file, no `Taskfile.yml` entry, no workflow and no file under `docs/` other than
      `docs/proposals/` names `internal/wasm`
- [x] `docs/architecture.md:222` and `:237` name the renamed package
- [x] `mise exec -- task build` still produces `internal/viewer/generated/emod.wasm`

**Affected Files/Modules:**
- `internal/wasm/pipeline.go`, `internal/wasm/pipeline_test.go` → `internal/pipeline/` — package
  clause, doc comment, test import and every qualifier
- `cmd/emod-wasm/main.go` — the single non-test importer
- `docs/architecture.md` — `:222` and `:237`

**Patterns to Follow:**
- `tasks/learnings.md:241-244` — the rename discipline this repo already applies: the Go identifier
  moves, the wire name does not
- `internal/wasm/pipeline.go:1-4` — the doc comment being replaced, and the reason it now misleads
  (`docs/proposals/emod-desktop-proposal.md:345-350`)
- `cmd/emod-wasm/main.go:19-54` — every call site, all in one file

**Testable:** Yes — `internal/pipeline`'s six exported functions are exercised by the moved
`pipeline_test.go`, which runs under `task test:unit`.

**Certainty:** high — a compiler-proven rename with one non-test importer (`cmd/emod-wasm/main.go:8`),
following the rename discipline `tasks/learnings.md:241-244` already records.

**Blast radius:** low — an `internal/` package rename with every exported name and every JSON
envelope unchanged; nothing outside the repository can observe it.

**Verification:** `mise exec -- task build`; `mise exec -- task test:unit`;
`mise exec -- task test:integration`;
`rg -n 'internal/wasm' -g '!docs/proposals' -g '!tasks'` prints nothing.

**Depends on:** None.

---

### Task 2: Split `internal/viewer` into `internal/frontend` and `internal/viewer`

**Behavior:** The frontend assets every distribution shares stop sharing a package with the localhost
HTTP delivery only the CLI uses, so a caller that needs the assets can depend on them without
depending on a server. Zero behaviour change: the CLI viewer serves the same bytes on the same routes,
the web bundle assembles identically, and the vitest suite collects the same 24 files.

**Acceptance Criteria:**
- [ ] `internal/frontend/` holds `static/`, `tests/`, `embed.go`, `package.json`,
      `package-lock.json` and `vitest.config.js`; `internal/viewer/` holds `serve.go` and
      `serve_test.go` and nothing else
- [ ] `internal/frontend/embed.go` declares `package frontend` and its `//go:embed` directive reaches
      the same two directories it reaches today (`internal/frontend/embed.go:7`)
- [ ] The WASM build output lands beside the assets the same package embeds: `Taskfile.yml:36-38`'s
      three paths, `.gitignore`'s `internal/viewer/generated/` entry and `build:web`'s copy source
      (`:51`) all name the new location, and `mise exec -- task build` succeeds from a tree with no
      generated directory present
- [ ] `internal/viewer/serve.go` reads the embedded assets from the frontend package;
      `ServeViewer`'s signature, both its routes (`/static/`, `/generated/`) and `buildHTML`'s
      `<!--INITIAL_DATA-->` substitution and `</` escaping (`:87-99`) are unchanged
- [ ] `internal/viewer/serve_test.go` passes with no assertion weakened
- [ ] `mise exec -- task build:web` produces `web/index.html`, `web/static/` and `web/generated/`
      with the same file list as before this task
- [ ] `test:viewer` and `test:viewer:deps` run in the directory that now holds `package.json`, and
      `mise exec -- task test:viewer` collects all 24 test files and passes
- [ ] `.github/workflows/ci.yml:31`'s `node-version-file` names the moved `package.json`
- [ ] `docs/architecture.md:213-241` and `docs/wasm-architecture.md:9,19,50,59` name paths that exist
      in the tree after this task
- [ ] No file outside `docs/proposals/` and `tasks/` names `internal/frontend/static`,
      `internal/frontend/tests`, `internal/viewer/generated` or `internal/viewer/package.json`
- [ ] `mise exec -- task test:e2e:viewer` passes — the web bundle still loads, instantiates the WASM
      module and renders

**Affected Files/Modules:**
- `internal/viewer/{static,tests,embed.go,package.json,package-lock.json,vitest.config.js}` →
  `internal/frontend/` — the move, plus `embed.go`'s package clause
- `internal/viewer/serve.go` — the embedded-FS reference at `:31`, `:35`, `:82`
- `Taskfile.yml` — `build:wasm` (`:36-38`), `build:web` (`:50-51`), `test:viewer` (`:77`),
  `test:viewer:deps` (`:85`)
- `.gitignore` — the `internal/viewer/generated/` entry
- `.github/workflows/ci.yml:31`
- `docs/architecture.md`, `docs/wasm-architecture.md`

**Patterns to Follow:**
- `internal/frontend/embed.go:1-8` and `internal/viewer/serve.go:31-40` — what the embed directive
  names and how `fs.Sub` reaches each half of it
- `Taskfile.yml:44-52` — the copy-and-rename assembly idiom, which does not change here but which
  Task 3 and Task 9 both extend
- `docs/proposals/emod-desktop-proposal.md:351-392` — §5.2's rationale and §5.3's resulting layout
- `tasks/learnings.md:226-229` — where the vitest config lives, that `task test:viewer` runs `npm ci`
  in the package directory, and that it is not part of `task test:unit`

**Testable:** Yes — `internal/viewer/serve_test.go` drives `ServeViewer` over the moved FS through
its exported entry point, and `task test:viewer` collects the moved vitest suite.

**Certainty:** medium — the embed-and-serve arrangement (`internal/frontend/embed.go:1-8`,
`serve.go:31-40`) and the `build:web` copy (`Taskfile.yml:44-52`) are both in front of you, but no
package in this repo has been split before and §5.2's file list omits `generated/`, which `embed.go`
also names — so where the WASM build output lands, and the `.gitignore`, `build:wasm` and `build:web`
edits that follow from it, are decided here (F2).

**Blast radius:** low — a package move under `internal/` with no exported behaviour change; the CLI's
routes, the CLI's output and the web bundle's layout are byte-identical afterwards, and
`task test:e2e:viewer` runs against the real bundle before the Pages deploy job.

**Verification:** `mise exec -- task build`; `test:unit`; `test:integration`; `test:viewer`;
`test:e2e:viewer`; `fd . internal/viewer` lists only `serve.go` and `serve_test.go`;
`diff <(ls web/static) <(ls internal/frontend/static)` after `task build:web` shows only
`viewer.html`.

**Depends on:** None — shares no file with Task 1. Ordered after it so the two renames land as
separate reviewable commits.

---

**Phase 1 — Platform adapter (~1d). Tasks 3 to 6. Exit: the CLI viewer and the web bundle behave
identically to today; no desktop code anywhere. Mergeable on its own.**

### Task 3: Put the Go bridge behind a `platform.js` contract

**Behavior:** Every shared UI module reaches the Go core through one module named for the seam rather
than for the browser's implementation of it. The browser implementation is today's `wasm.js` moved
whole. Which implementation a distribution gets is decided when that distribution is assembled, not
by sniffing the environment at runtime. Nothing a user can see changes.

**Acceptance Criteria:**
- [x] `internal/frontend/static/platform.browser.js` holds what `wasm.js` holds today — the same
      `fetch` of the WASM module, the same `instantiateStreaming` path and `arrayBuffer` fallback,
      the same `WASM initialization failed` and `WASM not ready yet` messages — and exports
      `parseEmod`, `exportEmod`, `ready` and `isReady` under those names
- [x] A `platform.js` module carries the contract every implementation satisfies, named in
      `docs/proposals/emod-desktop-proposal.md:171-188`
- [x] No shared UI module branches on which platform it is running on: outside the `platform.*.js`
      files, nothing under `internal/frontend/static/` names the desktop framework, inspects
      `navigator`, or tests for a runtime global to choose a code path
- [x] No file under `internal/frontend/static/` names `wasm.js`; the modules that reached the bridge
      reach `./platform.js` instead (today `viewer.js:12`, `emod-export.js:1`, `model.js:104`)
- [x] `model.js` keeps its deferred dynamic import — the platform module is still not loaded until a
      raw `.emod` source needs parsing
- [x] No file under `internal/frontend/tests/` imports or mocks `../static/wasm.js`. The four files
      that drive WASM fetch, instantiate and fallback directly test `platform.browser.js` and are
      named for it; the three that stub the bridge while another module is under test stub
      `../static/platform.js`
- [x] `mise exec -- task test:viewer` passes from a checkout where no build task has run, with no
      test file skipped and no assertion removed (F4)
- [x] `mise exec -- task build` produces a CLI whose served viewer resolves `platform.js`, and
      `emod diagram --serve` renders a model end to end
- [x] After `mise exec -- task build:web`, `web/static/` carries exactly one platform
      implementation — the browser one — and no module that imports desktop bindings (F5)
- [x] `mise exec -- task test:e2e:viewer` passes: the web bundle instantiates the WASM module,
      renders `SAMPLE` and exports it byte-identically

**Affected Files/Modules:**
- `internal/frontend/static/wasm.js` → `platform.browser.js`
- `internal/frontend/static/platform.js` — the contract (see Q1 for whether it is committed or
  assembled)
- `internal/frontend/static/viewer.js:12`, `emod-export.js:1`, `model.js:104` — the import specifier
  only; no body changes
- `internal/frontend/tests/wasm.test.js`, `wasm.fallback.test.js`, `wasm.init-success.test.js`,
  `wasm.init-http-error.test.js` — renamed to name the browser implementation they test
- `internal/frontend/tests/viewer.test.js:13`, `model.test.js:3-9`, `emod-export.test.js:11` — the
  mock target
- `Taskfile.yml` — `build`, `build:web` and possibly `test:viewer`, per Q1
- `.gitignore` — only if `platform.js` becomes a build output

**Patterns to Follow:**
- `docs/proposals/emod-desktop-proposal.md:171-230` — §4.3's contract and §4.4's before/after for
  each importer
- `docs/proposals/emod-desktop-proposal.md:231-251` — §4.5 on assembly-time selection and why
  runtime detection was rejected
- `docs/proposals/emod-desktop-proposal.md:592-602` — §8.1 on redirecting the mocks and which four
  test files are browser-only
- `Taskfile.yml:44-52` — the copy-and-rename idiom already in use
- `internal/frontend/tests/viewer.test.js:1-18` — the comment stating that only the bridge is stubbed
  because it is the module boundary; it names the boundary and must stay true of the new target
- `tasks/learnings.md:226-229` — the vitest harness's shape and how it is run

**Testable:** Yes — through the public module surface the seven redirected test files already drive.

**Certainty:** medium — `wasm.js` moves verbatim and the suite already mocks at exactly this boundary
(`internal/frontend/tests/viewer.test.js:13`, `model.test.js:5`, `emod-export.test.js:11`), but nothing
in this repo assembles a *single module* per target: `build:web` copies whole trees
(`Taskfile.yml:50`) and never substitutes one file for another, and the vitest suite has to resolve
`./platform.js` in a tree where no build task has run (F4).

**Blast radius:** low — it changes which module the shared UI imports, not what any of them does; the
Go API, the CLI's output and the JSON envelopes are untouched, and both the vitest and the Playwright
suites run against the result in CI.

**Verification:** `mise exec -- task test:viewer` on a clean checkout; `mise exec -- task build` then
`go run ./cmd/emod diagram --serve examples/all_patterns.emod` renders;
`mise exec -- task build:web && ls web/static`; `mise exec -- task test:e2e:viewer`;
`rg -n "wasm\.js" internal/frontend/` prints nothing.

**Depends on:** Task 2.

---

### Task 4: Read a dropped file through the platform contract

**Behavior:** Dropping a file on the source panel goes through the seam rather than through the
browser's file reader directly, so a native implementation can later hand back a real path from the
same gesture. Unchanged from a user's seat: the same accepted extensions, the same rejection message,
the same drag-over highlight, the same render.

**Acceptance Criteria:**
- [x] `viewer.js` names neither `FileReader` nor `readAsText`; the drop handler obtains the dropped
      file's contents from the platform module (today `viewer.js:166-188`)
- [x] What the platform hands back carries a place for the file's path as well as its contents — the
      shape `docs/proposals/emod-desktop-proposal.md:171-188` names — and the browser implementation
      leaves the path empty, because a browser drop has none (F6)
- [x] The extension check and its wording stay in the shared module: dropping a file with any other
      extension still shows `✗ Only .emod and .json files are supported` and leaves the current
      diagram on screen
- [x] A read that fails still shows `✗ Failed to read file` and leaves the current diagram on screen
- [x] The `drag-over` class is still added on `dragover` and removed on both `dragleave` and `drop`
- [x] The three drag-and-drop leaves in `internal/frontend/tests/viewer.test.js:191-233` pass, and
      what they install to stand in for the read is the platform module rather than
      `globalThis.FileReader` (`:93-102`, `:132`)
- [x] `mise exec -- task test:viewer` and `mise exec -- task test:e2e:viewer` pass

**Affected Files/Modules:**
- `internal/frontend/static/viewer.js:166-188` — the drop handler
- `internal/frontend/static/platform.browser.js` — the file read
- `internal/frontend/static/platform.js` — the contract entry
- `internal/frontend/tests/viewer.test.js:65-72`, `:93-102`, `:129-136`, `:191-233`

**Patterns to Follow:**
- `internal/frontend/static/viewer.js:166-188` — the handler being changed, including which parts are
  shared UI (the extension check, the status messages, the highlight) and which are the browser's
- `internal/frontend/tests/viewer.test.js:65-72` and `:93-102` — `fireDrop` and `MockFileReader`, the
  two stubs the leaves rest on
- `docs/proposals/emod-desktop-proposal.md:142-154` — §4.1 row 2, the touch point and what the
  desktop side of it will be
- `docs/proposals/emod-desktop-proposal.md:679-684` — Phase 1's exit condition

**Testable:** Yes — through the drop handler the three existing vitest leaves already drive.

**Certainty:** medium — the path is pinned by three leaves and a `MockFileReader` that already stands
in for the browser read (`internal/frontend/tests/viewer.test.js:93-102`, `:191-233`), but §4.3's
`openFile()` takes no argument while the browser's only file source is the drop event, so the shape
the handler calls through is decided here (F6).

**Blast radius:** low — one handler in the shared UI, with its three existing leaves as the guard;
no Go, no build, no output format.

**Verification:** `mise exec -- task test:viewer`; `mise exec -- task test:e2e:viewer`;
`rg -n "FileReader|readAsText" internal/frontend/static/viewer.js` prints nothing;
`mise exec -- task build` then drop a `.emod`, a `.json` and a `.txt` onto the served viewer.

**Depends on:** Task 3.

---

### Task 5: Write an exported model through the platform contract

**Behavior:** The Export .emod button hands the platform a suggested filename and the content instead
of building a download itself. The browser implementation keeps the blob-and-anchor dance verbatim,
so the same file with the same name and the same bytes still arrives in Downloads.

**Acceptance Criteria:**
- [ ] `viewer.js` names none of `Blob`, `URL.createObjectURL`, `URL.revokeObjectURL` or the anchor's
      `download` attribute; the export handler calls the platform's save with a suggested name and
      the content (today `viewer.js:208-222`)
- [ ] The suggested name is still the model name, or `diagram` when the model has none, with the
      `.emod` suffix
- [ ] An export that fails still puts its message in the status area, as it does today
      (`viewer.js:219-221`)
- [ ] `e2e-viewer/tests/export.spec.js` passes with no edit: the downloaded bytes still equal
      `SAMPLE` (`:8-13`) and the download is still named `Billing.emod` (`:15-23`)
- [ ] `mise exec -- task test:viewer` and `mise exec -- task test:e2e:viewer` pass

**Affected Files/Modules:**
- `internal/frontend/static/viewer.js:208-222` — the export handler
- `internal/frontend/static/platform.browser.js` — the save
- `internal/frontend/static/platform.js` — the contract entry

**Patterns to Follow:**
- `docs/proposals/emod-desktop-proposal.md:189-230` — §4.4's before/after for this exact block
- `internal/frontend/static/viewer.js:208-222` — the block being moved, verbatim
- `e2e-viewer/tests/helpers.js:129-137` and `e2e-viewer/tests/export.spec.js:8-23` — the guard that
  already pins both the bytes and the download name

**Testable:** Yes — through the Export button, which `e2e-viewer/tests/export.spec.js` drives against
the real bundle.

**Certainty:** high — the block moves whole from `internal/frontend/static/viewer.js:209-219`, and
`e2e-viewer/tests/export.spec.js:8-23` already pins both the exported bytes and the download name
through `exportEmod` (`e2e-viewer/tests/helpers.js:129-137`).

**Blast radius:** low — one handler in the shared UI, guarded end to end by the Playwright export
suite; the exported bytes come from Go and do not change.

**Verification:** `mise exec -- task test:e2e:viewer`; `mise exec -- task test:viewer`;
`rg -n "Blob|createObjectURL|revokeObjectURL|\.download" internal/frontend/static/viewer.js` prints
nothing.

**Depends on:** Task 4 — both rewrite the same seam module and adjacent regions of `viewer.js`.

---

### Task 6: Read the injected initial state through the platform contract

**Behavior:** The viewer stops reading a global that the CLI's HTTP layer injects and asks the
platform for its initial state instead. In the browser that is still `window.INITIAL_DATA`, injected
by `serve.go` exactly as today; a native shell can answer the same question with no `<script>` tag.
After this task `viewer.js` names no browser API a native shell cannot provide.

**Acceptance Criteria:**
- [ ] `viewer.js` does not name `INITIAL_DATA`; it asks the platform module for the initial state and
      renders it when there is one (today `viewer.js:291-292`)
- [ ] `internal/viewer/serve.go` is unchanged: the same `<!--INITIAL_DATA-->` marker at
      `internal/frontend/static/viewer.html:1162`, the same injection and `</` escaping
      (`serve.go:87-99`), and `serve_test.go` passes with no assertion weakened
- [ ] With no initial state the viewer still opens the data panel, shows the landing instructions,
      sets the name display to `(no model)` and sets the source placeholder —
      `internal/frontend/tests/viewer.test.js:139-150`
- [ ] With initial state the viewer still renders the supplied diagram and hides the landing page —
      `:151-162`
- [ ] The readiness gate still shows `⏳ Loading parser...` while the platform is not ready and
      clears it once it is — `:163-190`
- [ ] `emod diagram --serve <file>` still renders the model on first paint with no Render click
- [ ] `mise exec -- task test:viewer`, `test:unit`, `test:integration` and `test:e2e:viewer` pass

**Affected Files/Modules:**
- `internal/frontend/static/viewer.js:291-292` and the readiness block at `:302-315`
- `internal/frontend/static/platform.browser.js` — reading the injected global
- `internal/frontend/static/platform.js` — the contract entry
- `internal/frontend/tests/viewer.test.js:138-190`

**Patterns to Follow:**
- `internal/viewer/serve.go:81-101` — `buildHTML`, which produces the global and does not change
- `internal/frontend/static/viewer.html:1161-1163` — the marker and the two browser-only lines around
  it, which Task 9 has to derive a desktop page from
- `internal/frontend/tests/viewer.test.js:138-190` — the leaves covering both outcomes and the
  readiness gate
- `docs/proposals/emod-desktop-proposal.md:142-154` — §4.1 rows 4 and 5

**Testable:** Yes — through the viewer's initial load, covered by the existing leaves for both the
present and the absent case.

**Certainty:** high — the read is one branch at `internal/frontend/static/viewer.js:291-292`, both its
outcomes are already covered at `internal/frontend/tests/viewer.test.js:139-162`, and the injection
that feeds it (`internal/viewer/serve.go:87-99`) does not move.

**Blast radius:** low — one branch in the shared UI; the server-side injection, its escaping and the
CLI's behaviour are untouched.

**Verification:** `mise exec -- task test:viewer`; `mise exec -- task test:unit`;
`mise exec -- task test:e2e:viewer`; `mise exec -- task build` then
`go run ./cmd/emod diagram --serve examples/all_patterns.emod` paints the model immediately;
`rg -n "INITIAL_DATA" internal/frontend/static/` names only `viewer.html` and `platform.browser.js`.

**Depends on:** Task 5.

---

**Phase 2 — Wails v3 shell (~1.5d). Tasks 7 to 10. Exit: a desktop window renders a diagram from
pasted source, with feature parity to the browser and nothing more.**

### Task 7: Add the desktop model service over `internal/pipeline`

**Behavior:** A plain Go type exposes the three pipeline entry points with the same JSON-string
arguments and the same envelopes the WASM shims hand the frontend, so one adapter interface serves
both runtimes and `model.js` and `emod-export.js` need no per-runtime knowledge. It links no CGO and
imports no desktop framework, so it is testable wherever the rest of the repo is.

**Acceptance Criteria:**
- [ ] A `ModelService` type exposes `ParseEmod`, `ExportJSON` and `ExportEmod`, each taking and
      returning strings, mapping 1:1 onto `pipeline.RunPipelineExportDiagram`,
      `pipeline.RunPipelineExportJSON` and `pipeline.ExportEmodJSON`
      (`docs/proposals/emod-desktop-proposal.md:63-72`)
- [ ] `ParseEmod` and `ExportJSON` take the `{"source": "..."}` envelope; `ExportEmod` takes the bare
      diagram document, keeping the asymmetry `cmd/emod-wasm/main.go:27-29` documents and the viewer
      already relies on
- [ ] For source the pipeline accepts, `ParseEmod` returns exactly the bytes
      `pipeline.RunPipelineExportDiagram` returns — the `{diagnostics, diagram}` envelope, unwrapped
      and unreshaped
- [ ] For input the pipeline rejects, each method returns exactly `pipeline.ErrorJSON`'s string for
      that error, so a caller cannot tell the two runtimes apart from the payload
- [ ] `ExportEmod` returns the `{"emod": "..."}` envelope for a diagram document it can import, and
      `ErrorJSON`'s shape for one it cannot
- [ ] Source with diagnostics still returns a diagram alongside them rather than an error, as it does
      through WASM today
- [ ] The package imports nothing from the desktop framework, and `go build` of it succeeds with
      `CGO_ENABLED=0`
- [ ] `TestModelService` is a single umbrella test grouped by operation with `t.Run`, its scenario
      subtests named as sentences about observed behaviour, `testify/require` assertions, and a fresh
      service per leaf — and it runs under `mise exec -- task test:unit`

**Affected Files/Modules:**
- `internal/desktop/service.go` — the service (see Q2 for why it is not under `cmd/emod-desktop/`)
- `internal/desktop/service_test.go` — `TestModelService`

**Patterns to Follow:**
- `cmd/emod-wasm/main.go:19-54` — the shim being transliterated: which envelope each entry point
  takes, where `ErrorJSON` is applied, and the comment at `:27-29` explaining the asymmetry
- `docs/proposals/emod-desktop-proposal.md:252-340` — §4.6's sketch, and its instruction to keep the
  string/envelope contract in v1 even though v3 could marshal real types (`:328-333`)
- `internal/pipeline/pipeline_test.go` — the umbrella shape, the `decodeEmodEnvelope` helper and the
  fixtures, post-Task-1
- `internal/pipeline/pipeline.go:79-100` — `ExportEmodJSON` and `ErrorJSON`, the two envelope writers the
  criteria compare against

**Testable:** Yes — `ModelService`'s three methods are exported and take and return strings; nothing
about them needs a window.

**Certainty:** high — a direct transliteration of `cmd/emod-wasm/main.go:19-54` with `js.Value`
removed, over the same `internal/pipeline` functions, with `internal/wasm/pipeline_test.go` as the
test shape.

**Blast radius:** low — a new package with no existing caller; it links no CGO and cannot reach the
CLI, the WASM build or the viewer.

**Verification:** `mise exec -- task test:unit`; `CGO_ENABLED=0 go build ./internal/desktop`;
`rg -n "wails" internal/desktop/` prints nothing.

**Depends on:** Task 1.

---

### Task 8: Pin Wails v3 exactly and open a desktop window

**Behavior:** `task build:desktop` produces a native binary that opens a window, built from a
framework version pinned exactly in both `go.mod` and the tool manifest. The CLI and WASM builds keep
linking no CGO, and every existing task keeps passing on a machine with no desktop toolchain
installed. The window's contents are a throwaway the build task writes; Task 9 replaces it with the
shared frontend.

**Acceptance Criteria:**
- [ ] `go.mod` requires the Wails v3 module at an exact `v3.0.0-alphaN.NNN` version and `mise.toml`
      pins the `wails3` CLI at the matching version — neither floats, neither says `latest`, and the
      two name the same version
- [ ] The pinned version is the one current at implementation time, confirmed against the framework's
      own releases rather than copied from `docs/proposals/emod-desktop-proposal.md:629-631`
      (`v3.0.0-alpha2.117`, verified 8 Jul 2026; this breakdown is written 2026-08-19)
- [ ] `cmd/emod-desktop/main.go` builds the application through the Manager API — window creation
      goes through the application value, not through a top-level package function that earlier
      alphas exposed
- [ ] `mise exec -- task build:desktop` produces a binary that opens a window carrying the emod name
- [ ] The page the window loads is written by the build task into a gitignored directory; the tree
      holds no hand-written HTML file for the desktop app, and this task copies or duplicates no file
      from `internal/frontend/static/`
- [ ] `mise exec -- task build`, `task test:unit` and `task test:integration` pass on a machine with
      no GTK4 or WebKit development packages and no `wails3` on `PATH`: the CLI build stays
      `CGO_ENABLED=0` and `cmd/emod-desktop` is excluded from both `go list ./...` package sets the
      way `cmd/emod-wasm` already is (`Taskfile.yml:66`, `:71`) (F3)
- [ ] `build:desktop` does not depend on `build:wasm`, and produces no `.wasm` payload and no
      `wasm_exec.js`
- [ ] `.goreleaser.yaml` is unchanged and `mise exec -- task release:local` still produces the five
      CLI archives
- [ ] Everything the desktop build assembles or generates is gitignored: after
      `mise exec -- task build:desktop`, `git status --porcelain -- cmd/emod-desktop` reports nothing
      beyond this task's own source files
- [ ] The desktop toolchain is invoked through the repo's pin rather than whatever is on `PATH`
      (`tasks/learnings.md:11-14`, F7)

**Affected Files/Modules:**
- `go.mod`, `go.sum` — the pinned requirement
- `mise.toml` — the `wails3` pin, alongside the four tools already pinned exactly
- `cmd/emod-desktop/main.go` — the application, the window and the asset handler
- `Taskfile.yml` — a new `build:desktop`, and the package exclusions in `test:unit` (`:66`) and
  `test:integration` (`:71`)
- `.gitignore` — the assembled frontend directory and the generated bindings directory

**Patterns to Follow:**
- `docs/proposals/emod-desktop-proposal.md:252-312` — §4.6's `main.go` sketch, including the embed
  directive and `application.Options`
- `docs/proposals/emod-desktop-proposal.md:627-668` — §9.1: pin exactly in both places, use the
  Manager API, why older tutorials mislead, and why the blast radius stays small
- `docs/proposals/emod-desktop-proposal.md:426-444` — §7.1's CGO quarantine and the table of which
  distribution links what
- `Taskfile.yml:33-42` — a build task carrying its own `env:` block
- `Taskfile.yml:63-71` — the existing `grep -v /cmd/emod-wasm` exclusion, the precedent F3 extends
- `mise.toml:1-7` — every tool pinned exactly, no ranges
- `tasks/learnings.md:11-14` — a repo-pinned CLI loses to a global pin on `PATH`, silently

**Testable:** No — the deliverable is a build target and a window. The framework's own behaviour is
not this repo's to test, §8.2 (`:603-615`) rules out a desktop window driver, and the Go this task
adds is `application.New` wiring that cannot run headless. The service it will host is tested in
Task 7.

**Certainty:** low — this repo has never linked CGO, never pinned a pre-release dependency and never
called this framework; §9.1 records that the alpha's earlier flat API is gone and that third-party
references are already stale, so the shape has to be read from the pinned version's own package docs
rather than from the proposal or a tutorial.

**Blast radius:** low — the new binary is quarantined in its own `cmd/` package and its own build
task; `.goreleaser.yaml`, the CLI build and the WASM build are untouched, and nothing it adds is
reachable from `internal/` or from either shipped distribution.

**Verification:** `mise exec -- task build:desktop` then launch the binary and see a window;
`mise exec -- task build`; `mise exec -- task test:unit`; `mise exec -- task test:integration`;
`mise exec -- task release:local`; `git status --porcelain -- cmd/emod-desktop` after the build lists
only this task's own source files; `grep -n wails go.mod mise.toml` shows one exact version in each.

**Depends on:** None technically — it shares no file with Tasks 1 to 6. Numbered after them so the
Phase 1 slice can be merged without it.

---

### Task 9: Render the shared frontend in the desktop window through `platform.desktop.js`

**Behavior:** The desktop window loads the same frontend files the CLI viewer and the web bundle
load, with the desktop implementation of the platform contract in place of the browser one. Pasting
`.emod` source and rendering produces the same diagram, the same diagnostics and the same
interactions as the browser viewer — with no WASM, no HTTP server and no listening port.

**Acceptance Criteria:**
- [ ] `build:desktop` assembles the desktop app's frontend from `internal/frontend/static/`: no file
      under that directory is edited by this task, and no file under it gains a desktop twin anywhere
      in the tree outside the gitignored assembly directory
- [ ] The page the desktop app loads is derived from `internal/frontend/static/viewer.html` by the
      build task rather than hand-written, so editing `viewer.html` changes what all three
      distributions load (F8)
- [ ] `platform.desktop.js` is the only file that imports the generated bindings, and it exports the
      same names `platform.browser.js` exports
- [ ] The assembled frontend and the generated bindings are gitignored: after
      `mise exec -- task build:desktop`, `git status --porcelain -- cmd/emod-desktop` reports nothing
      beyond this task's own source files
- [ ] The desktop app loads no `.wasm` payload and no `wasm_exec.js`, and its window makes no request
      for either
- [ ] The window opens showing the viewer interface: source panel, diagram canvas, minimap,
      visibility toggles and diagnostics badge, as `emod diagram --serve` shows them
- [ ] Pasting the model at `e2e-viewer/tests/helpers.js:5-35` and rendering draws the same nodes and
      edges the browser viewer draws for the same source
- [ ] Source the pipeline reports on fills the diagnostics badge and panel with the same messages,
      severities and locations the browser viewer shows for the same source —
      `examples/error_diagnostics_test.emod` is a source that produces some
- [ ] Pan, zoom, fit-to-view, node selection, the detail panel, layout reset and the diagram context
      actions behave as they do in the browser viewer
- [ ] The running app opens no listening socket and issues no network request, checked against the
      running process
- [ ] The contract's file-open and file-save entry points exist in `platform.desktop.js` and report
      in the status area that they are not available in this build, so pressing Export produces a
      message rather than silence or an unhandled rejection (Q3)
- [ ] The desktop's initial state resolves to nothing, so the app opens on the same empty state the
      browser viewer shows with no injected data
- [ ] `emod diagram --serve` and the web bundle behave exactly as before this task:
      `mise exec -- task test:viewer`, `test:unit`, `test:integration`, `test:e2e:viewer` and
      `test:e2e` all pass

**Affected Files/Modules:**
- `internal/frontend/static/platform.desktop.js` — or wherever it can live without being copied into
  `web/` or embedded in the CLI binary (F5)
- `cmd/emod-desktop/main.go` — registering the service from Task 7 and pointing the window at the
  assembled frontend
- `Taskfile.yml` — `build:desktop` gains the frontend assembly, the page derivation and the bindings
  generation
- `.gitignore` — if the assembly's target directories changed since Task 8

**Patterns to Follow:**
- `docs/proposals/emod-desktop-proposal.md:252-340` — §4.6, in particular the generated-bindings
  import at `:335-340` and the instruction that only `platform.desktop.js` imports them
- `docs/proposals/emod-desktop-proposal.md:231-251` — §4.5's three assembly lines, one per target
- `Taskfile.yml:44-52` — `build:web`, which already copies `static/*` and renames `viewer.html`; the
  desktop assembly is the same idiom with a different destination
- `internal/frontend/static/platform.browser.js` — the contract's other implementation, whose exports
  this one must match exactly
- `internal/frontend/static/viewer.html:1161-1163` — the two browser-only lines the derived page must
  not carry
- `docs/proposals/emod-desktop-proposal.md:603-615` — §8.2, which says manual smoke testing is the
  answer here and why

**Testable:** No — §8.2 and the story's Non-Goals both rule out a desktop window driver, and the
desktop-specific code is the adapter plus the shell. Both halves are covered from the sides:
`ModelService` by Task 7's Go tests, and every shared frontend module by the vitest and Playwright
suites that keep running against the browser implementation. This task's criteria close on a manual
smoke pass against the running window.

**Certainty:** low — nothing in this repo has driven the viewer from anything but WASM globals; the
import path and call shape `wails3 generate bindings` emits are not knowable until the pinned alpha
is installed, and deriving the desktop page from `viewer.html` without a second copy in the tree (F8)
is decided here.

**Blast radius:** low — everything it adds lives in `cmd/emod-desktop/` and one platform
implementation; the CLI, the web bundle and every shared module are unchanged, and the existing
suites prove it.

**Verification:** `mise exec -- task build:desktop` then launch and smoke-test the window against the
criteria above; `lsof -p <pid>` (or the platform equivalent) shows no listening socket;
`mise exec -- task test:viewer`, `test:unit`, `test:integration`, `test:e2e:viewer`, `test:e2e`;
`git status --porcelain -- cmd/emod-desktop` after the build lists only this task's own source files;
`rg -n "bindings" internal/frontend/static/` names only `platform.desktop.js`.

**Depends on:** Tasks 6, 7 and 8.

---

### Task 10: Document the desktop distribution

**Behavior:** The repo's own architecture documents describe three distributions over two runtimes
rather than two distributions over one, and the README tells a reader how to build and run the
desktop app and what it deliberately does not do yet.

**Acceptance Criteria:**
- [ ] `docs/architecture.md`'s viewer section names the platform seam and all three distributions,
      and its diagram shows the desktop shell alongside the CLI and web paths (`:213-241` today)
- [ ] `docs/wasm-architecture.md` states that the desktop app reaches Go natively and uses no WASM,
      so a reader does not conclude the WASM path is the only way the frontend reaches Go
- [ ] The README documents `task build:desktop`, names where the framework version is pinned
      (`go.mod` and `mise.toml`, which must agree), and states what the desktop build does not do
      yet — no file dialogs, no save, no packaged app, no prebuilt download
- [ ] Every path, task name and file name the three documents state exists in the tree
- [ ] This task changes documentation only: `git diff --stat` lists no file outside `docs/` and
      `README.md`

**Affected Files/Modules:**
- `docs/architecture.md:211-241`
- `docs/wasm-architecture.md`
- `README.md`

**Patterns to Follow:**
- `docs/wasm-architecture.md:1-70` — the existing per-subsystem document: a mermaid diagram followed
  by a numbered flow summary
- `docs/architecture.md:211-241` — the section that names the two distributions today, and the build
  coupling note at `:238-240` that the desktop target now also has
- `docs/proposals/emod-desktop-proposal.md:92-139` — §3's framing, three distributions over two
  runtimes, and its diagram
- `tasks/learnings.md:36-39` — a document's own cross-references go stale silently; nothing in CI
  checks a markdown link

**Testable:** No — documentation. Its claims are checked by reading them against the tree, which the
fourth criterion states.

**Certainty:** high — `docs/wasm-architecture.md:1-70` is the existing per-distribution document and
`docs/architecture.md:211-241` is the section that names the two distributions today; both are
edited in place.

**Blast radius:** low — prose only; no build, no test, no shipped artefact.

**Verification:** `mise exec -- task test:unit` (the documented-models guard reads fenced `emod`
blocks from `README.md`); every path named in the three files resolves under `fd`;
`git diff --stat` lists documentation only.

**Depends on:** Task 9.

---

## Summary

**Ten tasks**, in three independently mergeable slices, ordered so that each slice leaves `main`
shippable and each task leaves the tree green.

The ordering is dependency-first within each slice and risk-last across them. Phase 0 (1, 2) is pure
motion: the two renames that make the desktop app's dependencies expressible, both guarded entirely
by tests that already exist. Phase 1 (3–6) is four subtractions from `viewer.js`, each one moving a
browser-shaped thing behind the seam and each closing on the vitest and Playwright suites that
already cover it; Task 3 carries the structural weight because it decides how a per-target module is
assembled, and 4, 5 and 6 are small once it has. Phase 2 (7–10) puts the tested, framework-free half
first: `ModelService` lands green under `task test:unit` before the alpha is added to `go.mod` at
all, so the two low-certainty tasks that follow are about the framework and nothing else. Task 8
isolates the alpha to a commit whose whole diff is a pin, a build target and a `main.go`; Task 9 is
the first commit in which anything can render.

**The two assessments.** Certainty comes out high on five tasks (1, 5, 6, 7, 10), medium on three
(2, 3, 4) and low on two (8, 9). The two lows are both the Wails v3 alpha — a dependency this repo
has never had, whose API §9.1 records as moving and whose tutorials it records as stale — and they
are adjacent on purpose, so the unknown is met twice in a row rather than spread through the plan.
Blast radius is `low` on every task, and that is a genuine reading rather than a skipped step: this
story touches no authentication, no money, no schema, no credential, no personal data and no
irreversible write, and its one externally-consumed contract — the CLI's behaviour and output — is
explicitly held constant by the story's sixth criterion and by a criterion in every task in Phases 0
and 1. The riskiest thing in the story is a broken web bundle, and `task test:e2e:viewer` runs
against the real bundle in CI before the Pages deploy job is allowed to start.

**Story criteria coverage.** All eight are covered; see the table under Story Reference. Two are
carried as categories rather than lists, deliberately: "the same viewer interface" and "the diagram
context actions behave as they do in the browser viewer" stay as the story states them in Task 9's
criteria, because the set of shared UI modules is 15 files today and was 10 when the proposal was
written — an enumeration would silently freeze whichever ones this document happened to name.

**Left to other stories:** everything in Boundaries → Deferred. In particular the desktop's
open and save entry points exist after Task 9 but report that they are unavailable — US-002 and
US-003 replace them, and until then the desktop app is a viewer for pasted source, which is exactly
what proposal Phase 2's exit says it should be (`:685-691`).

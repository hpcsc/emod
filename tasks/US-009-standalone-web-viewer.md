# US-009: Standalone web viewer with file upload

## Progress
- [x] Task 1: Create WASM entry point with exported parse functions
- [x] Task 2: Add WASM build target to Taskfile
- [x] Task 3: Add WASM runtime assets and embed them in the viewer
- [x] Task 4: Create WASM loader module for the frontend
- [x] Task 5: Refactor frontend to use WASM for client-side parsing
- [ ] Task 6: Polish landing page and standalone operation UX

---

## Story Reference

User story US-009 from `user-stories/web-diagram-viewer.md` (lines 132–143).  
Depends on US-003 (already completed).

---

## Codebase Context

### Project structure
- Module: `github.com/hpcsc/emod`
- CLI entry: `cmd/emod/main.go` — standalone Go binary
- Viewer frontend: `internal/viewer/` — all HTML, JS, CSS, and Go server code

### Viewer frontend architecture (all ES modules, vitest for testing)
The viewer is a modular single-page app backed by an event bus:

| File | Purpose |
|---|---|
| `viewer.html` | Single HTML file with all CSS + SVG template + script module loader |
| `viewer.js` | Entry point: creates store, wires DOM refs, subscribes to bus events, starts init |
| `model.js` | Data layer: `sendParse(store, source, statusEl)` calls `POST /parse` on server, `setModelData` loads nodes/edges into store |
| `store.js` | Central store: `nodes[]`, `edges[]`, `modelName`, `viewport`, `interaction` state, `dom` references |
| `bus.js` | Simple pub/sub event bus: `on`, `off`, `emit` |
| `renderer.js` | SVG DOM construction: `buildSVG(store)`, `clearSVG`, `inject` |
| `layout.js` | Auto-layout algorithm: `computeLayout(store)` → positions dict with x/y/w/h per node |
| `interaction.js` | Pan, zoom, drag nodes, inline editing, context menus |
| `ui.js` | Tooltip, detail panel, minimap, context panel, stats |
| `emod-export.js` | Serialize store back to `.emod` string |
| `config.js` | Layout constants, edge type definitions, color config |

### Current parse flow (server-dependent)
In `internal/viewer/model.js:77-103`, `sendParse()` sends an HTTP POST to `/parse` with `{ source: "..." }`. The server handler at `internal/viewer/serve.go:118-163` runs the full lex → parse → validate → lint → export pipeline and returns `{ diagnostics, diagram }` JSON.

### Initial data loading
At `internal/viewer/viewer.js:447-454`, the viewer checks for `window.INITIAL_DATA` (injected by the Go server at `internal/viewer/serve.go:87-99`). If present, it renders immediately. If absent, it opens the data panel with a placeholder textarea.

### Data panel (existing upload/paste UI)
The viewer already has a textarea (`#source-input`) and a Render button (`#render-btn`), plus drag-and-drop file upload on the panel body (`#data-panel-body`). These currently feed into the `/parse` endpoint.

### Export pipeline (used by WASM entry point)
- `internal/export/export.go:ExportDiagramJSONDiagnostics(model, diagnostics)` — wraps diagram JSON + diagnostics into `{ diagnostics: [...], diagram: { model_name, nodes, edges } }`
- `internal/export/export.go:ExportDiagramJSON(model)` — produces `{ model_name, nodes, edges }`
- `internal/export/export.go:ExportJSONDiagnostics(model, diagnostics)` — wraps raw AST JSON + diagnostics
- These functions call into `internal/lexer` → `internal/parser` → `internal/validator` → `internal/linter` → `internal/export`

### Embedding infrastructure
- `internal/viewer/embed.go:7-8` — `//go:embed` directive listing all frontend files. Current list: `viewer.html viewer.js store.js config.js bus.js layout.js renderer.js interaction.js ui.js model.js emod-export.js`
- `internal/viewer/serve.go` — HTTP server that serves embedded files and handles `/parse`
- Static assets served under `/static/` path prefix

### WASM considerations
- Parser (`internal/parser`) is pure Go with no CGO or platform-specific imports
- Lexer (`internal/lexer`), AST (`internal/ast`), diagnostic (`internal/diagnostic`), export (`internal/export`) are all pure Go — excellent WASM candidates
- `syscall/js` is the Go standard library package for JS interop
- Go WASM requires `wasm_exec.js` from GOROOT (`$(go env GOROOT)/misc/wasm/wasm_exec.js`)
- WASM binary needs to be served over HTTP (cannot load from `file://` protocol due to MIME type and CORS restrictions)
- Build command: `GOOS=js GOARCH=wasm go build -o <output> ./cmd/emod-wasm`

### Taskfile conventions
- `Taskfile.yml` already has `build`, `test`, `test:unit`, `test:integration`, `test:viewer` tasks
- `test:viewer` runs `npm ci && npm test` in `internal/viewer/` directory

---

## Tasks

### Task 1: Create WASM entry point with exported parse functions

**Behavior:** A new `cmd/emod-wasm/main.go` compiles to WebAssembly and exports two JS-callable functions: `parseEmod(inputJSON string) -> string` and `exportJSON(inputJSON string) -> string`. Both wrap the existing Go pipeline (lexer → parser → validator → linter → export) and communicate with JavaScript via `syscall/js`. The `parseEmod` function accepts `.emod` source text as a JSON object `{ source: "..." }` and returns the diagram JSON diagnostics wrapper (`{ diagnostics, diagram }`). The `exportJSON` function accepts `.emod` source text and returns the raw AST JSON diagnostics wrapper (`{ diagnostics, model }`).

**Acceptance Criteria:**
- [ ] `parseEmod(sourceJSON string) string` — accepts `JSON.stringify({ source: ".emod content" })`, runs lex → parse → validate → lint → `ExportDiagramJSONDiagnostics`, returns `JSON.stringify({ diagnostics, diagram })`
- [ ] `exportJSON(sourceJSON string) string` — accepts `JSON.stringify({ source: ".emod content" })`, runs same pipeline but returns `JSON.stringify({ diagnostics, model })` (raw AST JSON)
- [ ] Both functions register via `js.Global().Set("parseEmod", js.FuncOf(...))` for synchronous calling (or use a promise-based pattern if synchronous is impractical — the functions should feel like native JS calls)
- [ ] `main()` is a no-op that blocks with `select {}` (standard WASM pattern to keep the module alive)
- [ ] Error handling: if input is malformed JSON, return `JSON.stringify({ error: "..." })`. If parsing succeeds with diagnostics, return both the diagram data and diagnostics. If the pipeline panics, catch and return an error JSON.
- [ ] The file compiles (`go vet`, `GOOS=js GOARCH=wasm go build`) without errors

**Affected Files/Modules:**
- `cmd/emod-wasm/main.go` — New file: WASM entry point with `syscall/js` exports
- (No changes to existing internal packages — they are reused as-is)

**Patterns to Follow:**
- The export pipeline in `internal/viewer/serve.go:144-163` (`handleParse`) is the exact sequence needed: lex → parse → validate → lint → export. Reuse it by calling the same functions.
- WASM entry patterns: `main()` with `select {}` to block, `js.FuncOf` for exports, `js.Global().Set` for registration
- The JSON envelope structure follows `jsonDiagramDiagnosticsWrapper` in `internal/export/export.go:625-628` for `parseEmod` output
- The JSON envelope structure follows `jsonDiagnosticsWrapper` in `internal/export/export.go:169-172` for `exportJSON` output

**Testable:** Yes — Build the WASM binary, then load it in a Node.js or jsdom test with `WebAssembly.instantiate` and the Go WASM glue, call the exported functions, and verify the JSON output structure and content. Alternatively, write a Go unit test that builds and invokes the WASM module via `os/exec`.

**Verification:** `GOOS=js GOARCH=wasm go build -o /tmp/emod.wasm ./cmd/emod-wasm` succeeds. A simple Node.js script loading the WASM module and calling `parseEmod` returns valid diagram JSON.

**Depends on:** None

---

### Task 2: Add WASM build target to Taskfile

**Behavior:** A new Taskfile target `build:wasm` compiles the WASM entry point into `emod.wasm` and places it in `internal/viewer/` (for embedding). An optional `build:web` target copies all static viewer assets (HTML, JS files, `emod.wasm`, `wasm_exec.js`) into a `web/` directory at the project root for standalone deployment.

**Acceptance Criteria:**
- [ ] Running `task build:wasm` produces `internal/viewer/emod.wasm` from `./cmd/emod-wasm` with `GOOS=js GOARCH=wasm`
- [ ] The build sets `CGO_ENABLED=0` explicitly
- [ ] The existing `task build` (CLI binary) is unaffected
- [ ] Running `task test:viewer` after `task build:wasm` passes (to verify the WASM binary is buildable and consistent with the frontend)

**Affected Files/Modules:**
- `Taskfile.yml` — Add `build:wasm` task (and optionally `build:web`)

**Patterns to Follow:**
- Existing `build` task in `Taskfile.yml:9-13` sets `CGO_ENABLED: '0'` and uses `go build -o <output> <package>`
- Environment variable override pattern: `env: { GOOS: 'js', GOARCH: 'wasm' }` (or inline env vars)

**Testable:** Yes — `task build:wasm` exits 0 and `file internal/viewer/emod.wasm` reports a WebAssembly binary

**Verification:** `task build:wasm` runs cleanly; `ls -la internal/viewer/emod.wasm` shows a non-trivial WASM file.

**Depends on:** Task 1 (WASM entry point must exist to build)

---

### Task 3: Add WASM runtime assets and embed them in the viewer

**Behavior:** The Go WASM glue script (`wasm_exec.js`) is copied from GOROOT into `internal/viewer/` and added to the `//go:embed` directive alongside the existing viewer assets. After this task, `internal/viewer/` contains `wasm_exec.js` (the Go WASM runtime) and `emod.wasm` (the compiled parser), both served as static assets.

**Acceptance Criteria:**
- [ ] `wasm_exec.js` exists in `internal/viewer/` (either symlinked or copied from `$(go env GOROOT)/misc/wasm/wasm_exec.js`)
- [ ] `embed.go` includes `wasm_exec.js` and `emod.wasm` in its `//go:embed` directive
- [ ] `viewer.html` gains a `<script src="static/wasm_exec.js"></script>` tag (loaded before any module scripts that depend on WASM)

**Affected Files/Modules:**
- `internal/viewer/wasm_exec.js` — New file: copied from Go GOROOT `misc/wasm/wasm_exec.js`
- `internal/viewer/embed.go` — Add `wasm_exec.js` and `emod.wasm` to the `//go:embed` directive
- `internal/viewer/viewer.html` — Add `<script src="static/wasm_exec.js"></script>` in the `<head>` before module scripts

**Patterns to Follow:**
- Existing embed pattern in `internal/viewer/embed.go:7-8` — space-separated list of filenames in the `//go:embed` comment
- Existing static file serving pattern in `internal/viewer/serve.go:45` — `/static/` handler serves embedded files via `http.FileServer`

**Testable:** No (file copy, embed directive, and HTML script tag additions; verified by downstream tasks that load and use the WASM)

**Verification:** After `task build:wasm` and copying `wasm_exec.js`, `go build ./...` succeeds (embed will include all files). Viewing the served HTML confirms `wasm_exec.js` is loaded without 404.

**Depends on:** Task 2 (provides `emod.wasm`)

---

### Task 4: Create WASM loader module for the frontend

**Behavior:** A new `wasm.js` module encapsulates WASM loading, initialization, and provides a promise-based `parseEmod(source)` function. It manages loading state (exposes `isReady` and a `ready` promise) so other modules can await WASM readiness before calling parse. The module handles error reporting if WASM fails to load or initialize.

**Acceptance Criteria:**
- [ ] `wasm.js` exports `{ parseEmod, ready, isReady }`:
  - `ready` — a Promise that resolves when the Go WASM runtime is fully initialized and functions are registered
  - `isReady` — a boolean property, true after the `ready` promise resolves
  - `parseEmod(source)` — wraps `globalThis.parseEmod(source)` after verifying `ready`, returning the parsed JSON object `{ diagnostics, diagram }` or throwing on error
- [ ] The module creates a `Go` class instance (from `wasm_exec.js`) and calls `WebAssembly.instantiateStreaming` (with a fallback to `instantiate` for environments that don't support streaming)
- [ ] Download errors (WASM file not found, HTTP 404, network failure) are surfaced as a rejected `ready` promise with a descriptive message
- [ ] The module is importable from `viewer.js` via `import { parseEmod, ready } from './wasm.js'`

**Affected Files/Modules:**
- `internal/viewer/wasm.js` — New file: WASM loader module
- `internal/viewer/embed.go` — Add `wasm.js` to the `//go:embed` directive

**Patterns to Follow:**
- Module export pattern in `internal/viewer/model.js:105-112` — named exports via `export const Model = { ... }`
- The Go WASM execution pattern: `const go = new Go(); const result = await WebAssembly.instantiateStreaming(fetch("emod.wasm"), go.importObject); go.run(result.instance);`
- Promise pattern for async initialization follows the spirit of `model.js:sendParse()` returning a Promise

**Testable:** Yes — vitest tests with a mocked `WebAssembly` and `Go` class can verify:
- `ready` promise resolves when WASM instantiation succeeds
- `ready` promise rejects on network failure or instantiation error
- `parseEmod` delegates to `globalThis.parseEmod` and returns parsed JSON
- `parseEmod` throws if called before `ready` resolves (or if WASM failed)
- `isReady` reflects correct state

**Verification:** `npm test` passes in `internal/viewer/`. Manual: open the viewer in a browser, confirm no console errors about WASM loading, and `parseEmod` is available on the global scope.

**Depends on:** Task 3 (WASM assets must be served for the loader to fetch)

---

### Task 5: Refactor frontend to use WASM for client-side parsing

**Behavior:** The `Model.sendParse` function is refactored to call the WASM `parseEmod` function instead of `fetch("/parse", ...)`. The function signature and return value remain identical (`(store, source, statusEl) → Promise<{ diagnostics, diagram }>`), so all callers are unaffected. The refactored function handles three input formats: raw `.emod` source (parsed via WASM), diagram-oriented JSON (used directly without parsing), and raw AST JSON (used directly). Input type is auto-detected by attempting JSON parse.

**Acceptance Criteria:**
- [ ] `Model.sendParse` no longer makes a `fetch("/parse", ...)` call when WASM is available
- [ ] If input is `.emod` source (not valid JSON), it calls `parseEmod(source)` (WASM) and returns the result
- [ ] If input is diagram-oriented JSON (has `nodes` array), it wraps it as `{ diagnostics: [], diagram: <input> }` and returns it directly (no parsing needed)
- [ ] If input is raw AST JSON (has `model` key), wraps it similarly and returns directly
- [ ] If WASM is not ready when `sendParse` is called, it awaits the `ready` promise (with a timeout/error fallback)
- [ ] Error messages from WASM are propagated to the status display element (same UX as before)
- [ ] The `Model.sendParse` function signature and return type are unchanged — all imports and callers in `viewer.js` continue to work

**Affected Files/Modules:**
- `internal/viewer/model.js` — Rewrite `sendParse` to call WASM `parseEmod` instead of `fetch("/parse", ...)`, add input format detection logic, import `ready` and `parseEmod` from the new `wasm.js` module
- `internal/viewer/model.test.js` — Update `sendParse` tests to mock WASM instead of fetch; add tests for input format detection (raw emod, diagram JSON, AST JSON)

**Patterns to Follow:**
- Existing `sendParse` signature and Promise-based error handling in `internal/viewer/model.js:77-103`
- The JSON-detection pattern: try `JSON.parse(input)`, check for `nodes` or `model` keys to determine format
- The WASM call pattern from `wasm.js` (Task 4): `import { parseEmod, ready } from './wasm.js'`

**Testable:** Yes — vitest tests with mocked WASM functions can verify:
- Calling `sendParse` with `.emod` content calls `parseEmod` and returns the result
- Calling `sendParse` with diagram JSON bypasses WASM and returns it wrapped in the expected envelope
- Calling `sendParse` with AST JSON bypasses WASM and wraps it correctly
- Error propagation: WASM parse error → Promise rejected with the error message
- Status element is updated during loading and after completion

**Verification:** `npm test` passes. Manual: open viewer, paste `.emod` content, click Render, confirm diagram renders without any server call (check Network tab in devtools — no `/parse` requests).

**Depends on:** Task 4 (WASM loader module must exist)

---

### Task 6: Polish landing page and standalone operation UX

**Behavior:** The viewer's initial state (when no `INITIAL_DATA` is present) is polished into a clear landing page that guides users to upload a `.emod` file, paste `.emod` content, or paste pre-exported JSON. A loading indicator shows while WASM is initializing. The data panel is open by default in standalone mode, and the textarea placeholder clearly describes the accepted formats. The drag-and-drop file upload supports both `.emod` files and `.json` files.

**Acceptance Criteria:**
- [ ] When the page loads without `INITIAL_DATA`, the data panel is open with prominent instructions: "Upload a .emod file or paste content below"
- [ ] A loading indicator (spinner or status text) is displayed while WASM initializes — "Loading parser..." or similar
- [ ] Once WASM is ready, the loading indicator is removed (or transitions to "Ready")
- [ ] The textarea placeholder reads: "Paste .emod source or diagram JSON here"
- [ ] Drag-and-drop file upload handles `.emod` and `.json` files:
  - `.emod` files: content is parsed via WASM after drop
  - `.json` files: content is used directly (as diagram or AST JSON)
- [ ] If a `.json` file is dropped, auto-detect whether it's diagram JSON or AST JSON and handle accordingly
- [ ] No data leaves the browser — all parsing and rendering is client-side (this is an invariant, not a UI change)
- [ ] The existing behavior when `INITIAL_DATA` is present (CLI serve mode) is unchanged

**Affected Files/Modules:**
- `internal/viewer/viewer.js` — Add WASM loading state display logic (show/hide loading indicator based on `wasm.ready` promise), update initial empty state to show the landing page guidance
- `internal/viewer/viewer.html` — Add a loading spinner/indicator element for WASM initialization (hidden by default, shown by JS during WASM load)
- `internal/viewer/viewer.js` — Update drag-and-drop handler to accept `.json` files and auto-detect format
- `internal/viewer/viewer.html` — Update textarea placeholder text
- `internal/viewer/viewer.test.js` — New file (or extend existing tests) for the landing page UX behavior

**Patterns to Follow:**
- Existing initial state logic in `internal/viewer/viewer.js:447-454` — the `INITIAL_DATA` check pattern
- Existing drag-and-drop file handling in `internal/viewer/viewer.js:143-170` — extend to check file extension and handle `.json` differently
- The existing interaction for the data panel (collapsed/uncollapsed) follows the pattern in `internal/viewer/viewer.js:172-175`
- WASM loading state follows the pattern established in `wasm.js` (Task 4) — `ready` promise for readiness, `isReady` boolean

**Testable:** Yes — vitest tests with mocked WASM loading can verify:
- Initial state shows the data panel open and loading indicator when WASM not ready
- Loading indicator is removed when the `ready` promise resolves
- Textarea has the correct placeholder text
- `.emod` file drop triggers WASM parse
- `.json` file drop triggers direct data load
- When `INITIAL_DATA` is present, the landing page is skipped (viewer goes straight to rendering)

**Verification:** `npm test` passes. Manual: serve the viewer statically (e.g., `python3 -m http.server` from a directory with the built files), open in browser, confirm:
- A loading state appears briefly during WASM init, then disappears
- Two options are presented: file upload and text paste
- Pasting `.emod` content renders the diagram
- Dragging a `.emod` file renders the diagram
- Dragging a `.json` file (diagram JSON) renders the diagram
- Network tab shows no calls to any backend

**Depends on:** Task 5 (WASM-based parsing must work before the landing page can offer it)

---

### Task 7: Set up GitHub Pages deployment for standalone web viewer

**Behavior:** A GitHub Actions workflow builds the WASM binary and deploys the standalone viewer to GitHub Pages. On pushes to `main`, the workflow compiles `emod.wasm`, copies `wasm_exec.js` from GOROOT, assembles the static site in `web/`, and deploys it. A `build:web` Taskfile target is added to assemble the deployable directory locally.

**Acceptance Criteria:**
- [ ] `.github/workflows/pages.yml` exists with `build` and `deploy` jobs using `actions/upload-pages-artifact` and `actions/deploy-pages`
- [ ] The build job:
  - Sets up Go and checks out the repo
  - Runs `task build:web` (or equivalent steps) to produce `web/emod.wasm` and copy `wasm_exec.js`
  - Uploads the `web/` directory as a Pages artifact
- [ ] The deploy job runs after build, deploys to Pages, requires `id-token: write` and `pages: write` permissions
- [ ] `task build:web` exists in `Taskfile.yml` and produces:
  - `web/index.html` — copied from `internal/viewer/viewer.html`
  - `web/*.js` — all JS modules copied from `internal/viewer/`
  - `web/emod.wasm` — compiled WASM binary
  - `web/wasm_exec.js` — Go WASM glue from GOROOT
- [ ] The resulting static site loads in a browser without any backend server (all parsing via WASM)
- [ ] The workflow uses `concurrency` to cancel in-progress deployments on new pushes

**Affected Files/Modules:**
- `.github/workflows/pages.yml` — New file: GitHub Actions workflow for Pages deployment
- `Taskfile.yml` — Add `build:web` target that assembles the `web/` directory

**Patterns to Follow:**
- Existing `.github/workflows/ci.yml` for Go setup and checkout patterns
- Existing `.github/workflows/release.yml` for Task and Go setup
- Standard GitHub Pages deployment pattern: `actions/upload-pages-artifact@v3` + `actions/deploy-pages@v4`
- `wasm_exec.js` must be copied in CI from GOROOT (never committed) — `cp "$(go env GOROOT)/misc/wasm/wasm_exec.js" web/`

**Testable:** Partially — `task build:web` exits 0 and `ls web/` shows all expected files. Full verification requires a GitHub Pages deploy (or previewing the built `web/` directory locally with a static file server).

**Verification:** `task build:web` produces a complete `web/` directory. After merging to main, the GitHub Pages environment URL shows the viewer loading and rendering diagrams client-side (check Network tab for no backend calls).

**Depends on:** Task 1 (WASM entry point must exist), Task 5 (WASM-based frontend refactoring must be complete so the standalone site works without a backend)

---

## Summary

### Task Count: 7

### Ordering Rationale
**Dependency-first, then value-first:** The WASM Go entry point (Task 1) and the build target (Task 2) are the foundational prerequisites — without a WASM binary, nothing else is possible. The runtime assets (Task 3: `wasm_exec.js`, embedding) and loader module (Task 4: `wasm.js`) establish the bridge between the browser and the Go parser. Task 5 refactors the core `sendParse()` function to use WASM instead of the server endpoint, which is the critical behavioral change. Task 6 adds UX polish on top — the landing page guidance, loading states, and input format handling that make the standalone experience complete and intuitive.

### Acceptance Criteria Coverage

| Criterion | Covered In |
|---|---|
| The web viewer is a static site that works without a backend server | Tasks 3–6 (all client-side; no server-side `/parse` dependency after Task 5) |
| A landing page offers two options: upload a `.emod` file or paste `.emod` content into a text area | Task 6 |
| After uploading or pasting, the file is parsed client-side and the diagram renders | Tasks 4–5 (WASM loader provides client-side parsing; Task 5 refactors sendParse to use it) |
| The viewer also accepts diagram-oriented JSON as input | Task 5 (input format auto-detection: JSON bypasses WASM, used directly) |
| No file data leaves the browser — all parsing and rendering happens client-side | Tasks 4–5 (WASM runs entirely in the browser; no server calls for parsing after Task 5) |

All five acceptance criteria are fully covered with no deferred scope.

### Language Distribution
- **Tasks 1–2:** Go (WASM entry point, build target)
- **Tasks 3–4:** Go + JavaScript (embedding, WASM loader module)
- **Tasks 5–6:** JavaScript (frontend refactoring, UX polish)

# US-010: Serve the viewer locally via the CLI

## Progress
- [x] Task 1: Modify viewer HTML to remove hardcoded sample data and accept server-injected initial data
- [x] Task 2: Embed viewer HTML and serve it via local HTTP server with graceful shutdown
- [x] Task 3: Wire `--serve` flag into the diagram CLI command

---

## Story Reference

User story US-010 from `user-stories/web-diagram-viewer.md` (lines 148–158).  
Depends on US-009 (Standalone web viewer with file upload) — not yet implemented.

---

## Codebase Context

### Viewer HTML
- `internal/viewer/viewer.html` — Self-contained single-file web viewer (801 lines, HTML+CSS+JS)
- Currently hardcodes `SAMPLE_DATA` (Hotel Reservation model) and renders it on page load
- Has a collapsible "Data" panel with a JSON textarea and "Render" button for paste-and-render
- No file upload capability yet (this is US-009, not done)
- No mechanism to receive data from a server (no API calls, no injected globals)

### Embedding pattern
- `internal/cue/embed.go` embeds `schema.cue` using `//go:embed` — the same pattern should be used for the viewer HTML

### CLI structure
- CLI commands defined in `internal/cli/app.go` using `github.com/urfave/cli/v2`
- The `diagram` command is registered at `internal/cli/app.go:120-152` with `--format` and `-o` flags
- `internal/cli/diagram.go` contains `RunDiagram(path, outputPath, format string) error` — reads a file, lexes/parses/validates/lints, generates diagram output, writes to file or stdout
- Each handler follows the pattern: read file → lex → parse → validate → lint → produce output → return `*LintError` with exit code
- `LintError` type in `internal/cli/lint.go:15-18` for structured error/exit-code reporting

### Export pipeline
- `internal/export/export.go` has `ExportDiagramJSON(model *ast.Model) ([]byte, error)` — produces the diagram-oriented `{nodes, edges}` JSON
- `ExportDiagramJSONDiagnostics(model, diagnostics)` wraps the diagram JSON with a diagnostics array — useful for serving error info alongside partial data
- The diagram JSON format matches what the viewer HTML expects (same structure as `SAMPLE_DATA`)

### Test infrastructure
- Tests in `internal/cli/*_test.go` use `//go:build unit` build tag, `package cli_test`
- Helpers: `writeTemp`, `captureStdout`, `captureStderr` in `internal/cli/lint_test.go` and `internal/cli/validate_test.go`
- Test fixtures: `validEmod` / `invalidEmod` constants in `internal/cli/validate_test.go`
- Assertions use `github.com/stretchr/testify/require`

### Test command
- `go test -tags unit ./...`

---

## Tasks

### Task 1: Modify viewer HTML to remove hardcoded sample data and accept server-injected initial data

**Behavior:** The viewer HTML no longer hardcodes sample data. On page load, it checks for a global `INITIAL_DATA` variable injected by the server. If present, that data is rendered immediately. If absent, the viewer shows an empty state with the data panel open, a clear JSON textarea, and the upload/paste interface ready for user interaction.

**Acceptance Criteria:**
- [ ] Remove the `SAMPLE_DATA` constant and its loading on page load
- [ ] On page load, check for `window.INITIAL_DATA` — if present, load and render it; if absent, show empty state with data panel open
- [ ] The default empty state shows a placeholder in the JSON textarea (e.g., "Paste diagram JSON here or upload a .emod file") instead of sample data
- [ ] The "Render" button and data panel continue to work for manual paste-and-render
- [ ] The model name display shows "(no model)" or similar placeholder when no data is loaded
- [ ] The diagram canvas renders an empty/blank state when no nodes exist

**Affected Files/Modules:**
- `internal/viewer/viewer.html` — Remove `SAMPLE_DATA`, add `INITIAL_DATA` check, update initial load logic

**Patterns to Follow:**
- Keep the existing JS code structure (functions like `loadData`, `render`, `renderDiagram`)
- The `INITIAL_DATA` check should go near the bottom of the `<script>` block where the current `jsonInput.value = JSON.stringify(...)` and `loadData`/`renderDiagram` calls happen (lines 796–798)

**Testable:** No — The HTML change is verified through the HTTP server tests in Task 2, which fetch the HTML and validate its behavior programmatically.

**Verification:** Open the modified `viewer.html` directly in a browser — should show an empty diagram with the data panel open and blank textarea, no sample data pre-loaded.

**Depends on:** None

---

### Task 2: Embed viewer HTML and serve it via local HTTP server with graceful shutdown

**Behavior:** A new internal package embeds the viewer HTML using `//go:embed`. An exported function starts a local HTTP server that serves the viewer HTML at the root URL, accepting an optional diagram JSON payload to inject as `INITIAL_DATA` into the HTML. On startup, the server prints the local URL to stdout. On SIGINT/Ctrl+C, the server shuts down cleanly with no dangling resources.

**Acceptance Criteria:**
- [ ] Viewer HTML is embedded via `//go:embed` (following the pattern in `internal/cue/embed.go`)
- [ ] A `ServeViewer(port int, diagramJSON []byte)` function (or equivalent) starts an HTTP server on localhost
- [ ] If `diagramJSON` is provided, it is injected into the HTML as `window.INITIAL_DATA = <json>;` before the closing `</script>` tag or equivalent
- [ ] If `diagramJSON` is nil/empty, the HTML is served without injection (viewer shows its empty state)
- [ ] The server prints the URL to stdout (e.g., "Viewer available at http://localhost:XXXX")
- [ ] The server listens only on 127.0.0.1 (localhost) for security
- [ ] Ctrl+C (SIGINT) triggers graceful shutdown — the server stops accepting new connections and active requests complete
- [ ] The server can be started programmatically (not just via CLI) — the function returns an error or a shutdown function

**Affected Files/Modules:**
- `internal/viewer/embed.go` — New file; embeds `viewer.html` via `//go:embed`
- `internal/viewer/serve.go` — New file; HTTP server setup, handler with optional data injection, graceful shutdown via signal handling
- `internal/viewer/serve_test.go` — New file; tests for the server

**Patterns to Follow:**
- Embedding pattern in `internal/cue/embed.go:7-8` — `//go:embed viewer.html` with a `var ViewerHTML string`
- Use standard library `net/http` for the server (no external dependency needed)
- Use `context.Context` with cancellation for graceful shutdown (signal.NotifyContext)
- Test pattern: start the server on a random port, make HTTP GET requests, verify response body contains expected content

**Testable:** Yes — Tests can start the server on `127.0.0.1:0` (random port), fetch the HTML via HTTP, and verify:
- The response contains the viewer HTML structure
- With injected data, the response contains `window.INITIAL_DATA =` followed by the JSON
- Without injected data, there is no `INITIAL_DATA` assignment
- The server shuts down cleanly via context cancellation

**Verification:** `go test -tags unit ./internal/viewer/...` passes; manual test with a small Go program calling `ServeViewer` confirms the HTML is served and browser can connect.

**Depends on:** Task 1 (relies on modified viewer HTML that accepts `INITIAL_DATA`)

---

### Task 3: Wire `--serve` flag into the diagram CLI command

**Behavior:** The `emod diagram` command gains a `--serve` flag. When `--serve` is used, instead of writing diagram output to a file, the CLI starts the viewer server locally and opens the default browser. If a file argument is provided alongside `--serve`, the file is parsed and the diagram JSON is served with the viewer (pre-rendered). If no file argument is given, the viewer opens without pre-loaded data, showing the upload/paste interface. The existing file-output behavior is completely unchanged when `--serve` is absent.

**Acceptance Criteria:**
- [ ] `emod diagram --serve` starts the HTTP server, prints the URL to stdout, and opens the default browser to that URL
- [ ] `emod diagram --serve myfile.emod` parses the file through the existing lex/parse/validate/lint pipeline, generates diagram JSON via `ExportDiagramJSON`, starts the server with that data injected, prints the URL, and opens the browser
- [ ] `emod diagram myfile.emod` (without `--serve`) continues to work exactly as before — no behavioral change
- [ ] When a file has parse/validation errors, diagnostics are printed to stderr, but the server still starts with whatever diagram data was produced (partial data)
- [ ] Ctrl+C shuts down the server cleanly
- [ ] The terminal displays the URL (e.g., "Viewer available at http://localhost:XXXX")
- [ ] The `--serve` flag is a boolean flag (no value required)

**Affected Files/Modules:**
- `internal/cli/app.go` — Add `--serve` boolean flag to the `diagram` command definition
- `internal/cli/diagram.go` — Add a `RunServe(path string)` function (or modify the command action to branch on `--serve`) that reuses the existing pipeline to produce diagram JSON and delegates to the server from Task 2
- `internal/cli/diagram_test.go` — Add tests for `--serve` behavior

**Patterns to Follow:**
- CLI flag pattern in `internal/cli/app.go:125-133` — existing `-o` and `--format` flags on the diagram command; add `--serve` alongside them
- Handler pattern in `internal/cli/export.go:103-123` — specifically `handleDiagramJSONExport` which calls `ExportDiagramJSONDiagnostics` and handles diagnostics — this is the same pipeline the serve path needs
- The action closure in `internal/cli/app.go:135-151` should check `c.Bool("serve")` and branch to either `RunDiagram` or the new serve function
- Test pattern in `internal/cli/diagram_test.go` — use `writeTemp`, `captureStdout`, `captureStderr`

**Testable:** Yes — Tests can:
- Run the CLI with `--serve` and a file arg, capture stdout to find the URL, make an HTTP request to that URL, and verify the response contains the injected diagram data
- Run with `--serve` and no file arg, verify the URL is printed and the viewer shows the empty state (no `INITIAL_DATA` in response)
- Run without `--serve` and verify existing file-output behavior is unchanged
- Verify that Ctrl+C or context cancellation triggers server shutdown

**Verification:** `go test -tags unit ./internal/cli/...` passes; manual smoke test with `emod diagram --serve examples/all_patterns.emod` opens a browser with the diagram rendered.

**Depends on:** Task 2 (the server infrastructure must exist before the CLI can start it)

---

## Summary

### Task Count: 3

### Ordering Rationale
- **Dependency-first:** Task 1 (HTML modification) enables Task 2 (server embedding and serving), which enables Task 3 (CLI wiring). This ordering ensures each task builds on stable prerequisites.
- **Risk-first:** The HTML/JS changes (Task 1) carry moderate risk (breaking the viewer's rendering logic), so they are tackled first with manual verification. The server infrastructure (Task 2) is the core new functionality and should be tested independently before wiring into the CLI. The CLI wiring (Task 3) is the lowest-risk change as it primarily delegates to existing components.

### Acceptance Criteria Coverage

| Acceptance Criterion | Covered In |
|---|---|
| `emod diagram --serve` starts a local HTTP server and opens the browser | Task 3 |
| Without a file argument, viewer opens with upload/paste interface | Task 1 (empty state) + Task 3 (no injection) |
| With a file argument, viewer opens with that file already rendered | Task 2 (injection mechanism) + Task 3 (pipeline + injection) |
| Server shuts down cleanly on Ctrl+C | Task 2 (graceful shutdown) + Task 3 (signal propagation) |
| Terminal displays the local URL | Task 2 (URL printing) + Task 3 (stdout output) |

No acceptance criteria are deferred — all are covered by the three tasks. Note that file upload capability (part of "upload/paste interface") depends on US-009 which is a prerequisite for this story.

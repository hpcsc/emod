# US-002: Export model as diagram-oriented JSON

## Progress
- [x] Task 1: Add ExportDiagramJSON and ExportDiagramJSONDiagnostics with tests
- [x] Task 2: Wire diagram-json format into CLI export command

## Story Reference
`user-stories/rate-limiting.md` — **US-002: Export model as diagram-oriented JSON** (inline in decompose request)

## Codebase Context

**Exists today:**
- `internal/export/export.go` — `ExportJSON(model)`, `ExportJSONDiagnostics(model, diags)`, and intermediate JSON types (`jsonModel`, `jsonActor`, `jsonContext`, etc.)
- `internal/export/export_test.go` — Comprehensive tests in `package export_test` using `//go:build unit` tag; round-trips to `map[string]any` with `testify/require`
- `internal/cli/export.go` — `RunExport(path, format string) error` dispatches to `handleJSONExport` / `handleCUEExport` based on format; currently validates `format != "json" && format != "cue"`
- `internal/cli/app.go` — export command registered with `--format` flag, usage: `"Output format (json|cue)"`
- `internal/cli/export_test.go` — CLI integration tests with `writeTemp`, `captureStdout`, `captureStderr` helpers; uses `validEmod` / `invalidEmod` constants from `validate_test.go`
- `internal/ast/ast.go` — AST types: `Model`, `Actor`, `Context`, `Aggregate`, `Slice`, `Command`, `Event`, `Field`, `Flow`, `Trigger`, `View`, `Automation`, `Translation`
- `internal/diagnostic/entry.go` — `Entry` with Filename, Line, Column, Message, Severity, RuleName

**Pattern for the new export function:**
- Define intermediate JSON types (not AST-coupled) in `internal/export/export.go`
- `ExportDiagramJSON` returns `([]byte, error)` — mirrors `ExportJSON`
- `ExportDiagramJSONDiagnostics` wraps it with diagnostics envelope — mirrors `ExportJSONDiagnostics`
- Node IDs: deterministic sequential numbering prefixed by type (e.g., `actor-1`, `context-1`, `aggregate-1`, `slice-1`, `command-1`, `event-1`)
- `parentId` references the containing node by ID, or `null` for top-level nodes (actors, contexts)
- Fields on command/event nodes use simplified `{name, type, modifier}` (omit position details)

---

## Tasks

### Task 1: Add ExportDiagramJSON and ExportDiagramJSONDiagnostics with tests

**Behavior:** The `ExportDiagramJSON` function converts an `*ast.Model` into a JSON document with `nodes` and `edges` arrays. The `ExportDiagramJSONDiagnostics` function wraps that document with a `diagnostics` array. Both are exported and independently testable.

**Acceptance Criteria:**
- [ ] `ExportDiagramJSON(model)` returns valid JSON with top-level keys: `model_name`, `nodes`, `edges`
- [ ] Nodes include `id`, `type` (one of: `actor`, `context`, `aggregate`, `slice`, `command`, `event`), `label`, and `parentId` (string or `null`)
- [ ] Edges have `source`, `target`, and `type` (e.g. `"flow"`)
- [ ] Command and event nodes include a `fields` array with `{name, type, modifier}` entries; modifier omitted when empty
- [ ] Actors and contexts have `parentId: null`; aggregates reference their parent context; slices reference their parent aggregate; commands/events reference their parent slice
- [ ] `ExportDiagramJSONDiagnostics(model, diags)` wraps the document with a `diagnostics` array at the top level (same format as `ExportJSONDiagnostics`)
- [ ] Nil model or empty model handled gracefully
- [ ] All existing tests in `internal/export/export_test.go` continue to pass

**Affected Files/Modules:**
- `internal/export/export.go` — Add intermediate types: `jsonDiagramDocument`, `jsonDiagramNode`, `jsonDiagramEdge`, `jsonDiagramField`; add `ExportDiagramJSON()` and `ExportDiagramJSONDiagnostics()` functions; add internal conversion helpers (similar to `convertModel`/`convertActor`/etc.)
- `internal/export/export_test.go` — Add `TestExportDiagramJSON` umbrella function with subtests covering: minimal model, full model with all node types, nodes with parentId chains, edges from flows, fields on command/event nodes, nil model, model with empty slices; add `TestExportDiagramJSONDiagnostics` with subtests for empty/nil diagnostics and mixed-severity diagnostics

**Patterns to Follow:**
- Existing `ExportJSON`/`ExportJSONDiagnostics` in `internal/export/export.go:238-242` and `internal/export/export.go:186-198` — function signatures, return patterns, and error handling
- Existing intermediate type definitions (`jsonModel`, `jsonActor`, etc.) at `internal/export/export.go:28-34` — define diagram-specific types in the same package without coupling to AST types
- Test structure in `TestExportJSON` at `internal/export/export_test.go:18-1211` and `TestExportJSONDiagnostics` at `internal/export/export_test.go:1213-1402` — umbrella `TestExportDiagramJSON` with `t.Run` subtests, round-trip via `json.Unmarshal` to `map[string]any`, `testify/require` assertions

**Testable:** Yes

**Verification:** `go test -tags=unit ./internal/export/` passes

**Depends on:** None

---

### Task 2: Wire diagram-json format into CLI export command

**Behavior:** Running `emod export <file> --format diagram-json` produces the diagram-oriented JSON document on stdout, and the `--format` flag usage text acknowledges the new format. The same diagnostic-wrapping behavior as JSON format applies — diagnostics appear in the output, and the exit code reflects severity.

**Acceptance Criteria:**
- [ ] `emod export <file> --format diagram-json` outputs a JSON document with `model_name`, `nodes`, `edges`, and `diagnostics` at the top level
- [ ] A valid file produces exit code 0 with empty diagnostics array
- [ ] A file with parse/validation errors still emits the diagram document with diagnostics included (exit code reflects severity)
- [ ] The `--format` flag usage text includes `diagram-json` in the list of supported formats
- [ ] Invalid format still returns the appropriate error with all supported formats listed
- [ ] All existing CLI tests continue to pass

**Affected Files/Modules:**
- `internal/cli/export.go` — Update format validation in `RunExport` (line 23) to allow `"diagram-json"`; add a `"diagram-json"` case to the switch statement with a new handler function that calls `export.ExportDiagramJSONDiagnostics` and writes to stdout (mirroring `handleJSONExport`)
- `internal/cli/app.go` — Update `--format` flag usage string on line 99 from `"Output format (json|cue)"` to include `diagram-json`
- `internal/cli/export_test.go` — Add subtests under `TestExport` covering: valid file with `--format diagram-json` produces correct structure; invalid file with `--format diagram-json` produces diagnostics wrapper with nodes/edges; unsupported format error message includes diagram-json

**Patterns to Follow:**
- Format validation at `internal/cli/export.go:23-28` — add `"diagram-json"` to the allowed formats
- `handleJSONExport` at `internal/cli/export.go:81-99` — mirror this pattern for the diagram-json handler (call export function, write to stdout, return `LintError` with exit code)
- CLI tests at `internal/cli/export_test.go:39-58` — `writeTemp` + `captureStdout` + JSON unmarshal pattern for diagram-json format
- Invalid file test pattern at `internal/cli/export_test.go:84-116` — test that diagnostics are embedded and non-empty while nodes/edges are still present
- Unsupported format test at `internal/cli/export_test.go:219-232` — update to verify the new format name appears in the error message

**Testable:** Yes

**Verification:** `go test -tags=unit ./internal/cli/` passes

**Depends on:** Task 1

---

## Summary

**Total tasks:** 2

**Ordering rationale:** Dependency-first. Task 1 (export function) must exist before Task 2 (CLI wiring) can reference and test it. Within each task, happy-path and diagnostics/error handling are grouped because they belong to the same behavior (export function with diagnostics wrapper, CLI command with error handling).

**Acceptance criteria coverage:**

| AC | Covered In |
|---|---|
| `emod export <file> --format diagram-json` outputs JSON with `nodes` and `edges` arrays | Task 1 (export function) + Task 2 (CLI wiring) |
| Each node has `id`, `type`, `label`, `parentId` | Task 1 |
| Each edge has `source`, `target`, `type` | Task 1 |
| Field details on command and event nodes | Task 1 |
| Model name as top-level metadata | Task 1 |
| Diagnostics array with successfully parsed elements still emitted | Task 1 (function) + Task 2 (CLI integration) |

All six acceptance criteria are fully covered with no deferred scope.

## Progress
- [x] Task 1: Add source positions to JSON export types
- [x] Task 2: Add diagnostics wrapper for JSON output
- [x] Task 3: Update CLI to always output JSON with diagnostics

---

## Story Reference

US-001: Export model as raw AST JSON — inline description.

**Acceptance Criteria:**
- [ ] `emod export <file> --format json` outputs a JSON document to stdout representing the complete AST
- [ ] The JSON structure preserves the full hierarchy: model name, actors, contexts, aggregates, slices, commands, events, fields, and flows
- [ ] Field types and modifiers (e.g. `required`, `optional`) are included
- [ ] Source positions (file, line, column) are included for each named element
- [ ] If the file has parse errors, the command still outputs JSON for the successfully parsed portion and includes a `diagnostics` array with file, line, and message for each error
- [ ] `emod export <file> --format json` with a fully valid file produces an empty `diagnostics` array

---

## Codebase Context

### AST types (`internal/ast/ast.go`)
- Every named element has a `Position` struct (Filename, Line, Column) via `NamePos` or named position fields (e.g. `SourcePos`, `TypePos`, `ModPos`)
- `Comment` embeds `Position` directly
- `Context`, `Aggregate`, `Slice`, `Command`, `Event`, `Trigger`, `View`, `Automation`, `Translation` have `OpenPos`/`ClosePos` in addition to `NamePos`

### Current JSON export (`internal/export/export.go`)
- `ExportJSON(model *ast.Model) ([]byte, error)` converts AST to intermediate JSON types, then marshals
- JSON intermediate types (`jsonModel`, `jsonActor`, `jsonContext`, `jsonAggregate`, `jsonSlice`, `jsonCommand`, `jsonEvent`, `jsonField`, `jsonFlow`, `jsonTrigger`, `jsonView`, `jsonAutomation`, `jsonTranslation`, `jsonComment`) do NOT include any position fields
- No diagnostics wrapper — just raw model JSON
- No import of `internal/diagnostic` package

### Current CLI (`internal/cli/export.go`)
- `RunExport(path, format string) error` reads, lexes, parses, validates, lints
- When diagnostics exist: writes text diagnostics to stderr, returns non-zero exit code via `LintError`, does NOT output JSON
- When clean: outputs bare JSON to stdout, returns nil
- Only imports `internal/diagnostic` for severity checks

### Diagnostic types (`internal/diagnostic/entry.go`)
- `Entry` has Filename, Line, Column, Message, Severity (Error/Warning), RuleName
- `String()` formats as `file:line: message` or `file:line: [rule] message`

### Test infrastructure
- `internal/export/export_test.go` — `//go:build unit`, uses `testify/require`, `encoding/json`, builds AST models inline, unmarshals output to `map[string]any` to verify structure
- `internal/cli/export_test.go` — `//go:build unit`, uses `writeTemp` (from `validate_test.go`) and `captureStdout` (from `lint_test.go`), `captureStderr` helper
- Test fixtures `validEmod` and `invalidEmod` defined in `internal/cli/validate_test.go`

### LintError type
- Defined somewhere in `internal/cli/` — used for exit code signaling

---

## Tasks

### Task 1: Add source positions to JSON export types

**Behavior:** The JSON output from `ExportJSON` includes `filename`, `line`, and `column` for every AST element that has source position information. This enriches the existing output without changing its structure (other than adding fields).

**Acceptance Criteria:**
- [ ] Source positions (file, line, column) are included for each named element in JSON output
- [ ] Position fields use `omitempty` so zero-value positions don't clutter output
- [ ] Existing model fields (name, type, modifier, comments, etc.) are unchanged
- [ ] `ExportJSON` function signature is unchanged

**Affected Files/Modules:**
- `internal/export/export.go` — Add position fields to all JSON intermediate types (`jsonModel`, `jsonActor`, `jsonContext`, `jsonAggregate`, `jsonSlice`, `jsonCommand`, `jsonEvent`, `jsonField`, `jsonFlow`, `jsonTrigger`, `jsonView`, `jsonAutomation`, `jsonTranslation`, `jsonComment`). Update all `convert*` functions to copy position data from AST nodes to JSON types. Add a `jsonPosition` type with `filename`, `line`, `column` fields.
- `internal/export/export_test.go` — Add test cases verifying position fields appear in JSON output for at least model name, actor name, context name, aggregate name, slice name, command name, event name, field name/type/modifier, flow command/event names, trigger kind/name, view name, automation name, translation name, and nested event.

**Patterns to Follow:**
- Follow the existing JSON intermediate type pattern in `internal/export/export.go:15-113` — add new `jsonPosition` struct and embed/reference it similarly to how comments are handled.
- Follow the existing `convert*` function pattern in `internal/export/export.go:120-411` for copying position data.
- Follow the existing test pattern in `internal/export/export_test.go:17-627` — build AST models inline, call `ExportJSON`, unmarshal to `map[string]any`, assert field presence.

**Testable:** Yes

**Verification:** `go test ./internal/export/ -tags=unit -run TestExportJSON` passes.

**Depends on:** None

---

### Task 2: Add diagnostics wrapper for JSON output

**Behavior:** A new function `ExportJSONDiagnostics` (or similar) wraps the model JSON and a diagnostics slice into a structured envelope: `{"diagnostics": [...], "model": ...}`. Each diagnostic entry includes `file`, `line`, `column`, `message`, `severity`, and `rule_name`. For a clean model, `diagnostics` is an empty array.

**Acceptance Criteria:**
- [ ] The wrapper JSON structure has top-level `diagnostics` array and `model` object
- [ ] Each diagnostic entry includes `file`, `line`, `column`, `message`, `severity`, and `rule_name` (with `omitempty` for `rule_name`)
- [ ] Passing a nil/empty diagnostics slice produces `"diagnostics": []` (not null)
- [ ] Passing a nil model produces `"model": null`
- [ ] `ExportJSON` function remains unchanged and still produces bare model JSON

**Affected Files/Modules:**
- `internal/export/export.go` — Add a new exported function `ExportJSONDiagnostics` that takes `*ast.Model` and `[]diagnostic.Entry` and returns wrapped `[]byte`. Add intermediate type(s) for the wrapper envelope and diagnostic entries. Import `internal/diagnostic`.
- `internal/export/export_test.go` — Add test cases for the new function: empty diagnostics, multiple diagnostics with/without rule names, nil model, valid model with no diagnostics.

**Patterns to Follow:**
- Follow the struct tag pattern from `internal/export/export.go:15-113` for the wrapper envelope and diagnostic JSON types.
- The `diagnostic.Entry` type is defined in `internal/diagnostic/entry.go:21-28`. Use its fields directly; do not reproduce field semantics in the description.
- Follow the test pattern used throughout `internal/export/export_test.go` — inline model construction, JSON round-trip via `map[string]any`, testify assertions.

**Testable:** Yes

**Verification:** `go test ./internal/export/ -tags=unit -run TestExportJSONDiagnostics` passes.

**Depends on:** None (the new function is additive; does not require Task 1 but naturally benefits from it)

---

### Task 3: Update CLI to always output JSON with diagnostics included

**Behavior:** `RunExport` no longer writes diagnostics to stderr. Instead, it always outputs a complete JSON document to stdout via the diagnostics wrapper function. The exit code still distinguishes clean (0), warnings-only (1), and errors (2) — but the JSON is always produced on stdout.

**Acceptance Criteria:**
- [ ] A fully valid file produces JSON on stdout with `"diagnostics": []` and exits 0
- [ ] A file with only lint warnings produces JSON on stdout with populated `diagnostics` array and exits 1
- [ ] A file with parse/validation errors produces JSON on stdout with populated `diagnostics` array (including partially parsed model) and exits 2
- [ ] No text diagnostics are written to stderr for any export invocation
- [ ] The CUE export format behavior (`--format cue`) remains unchanged (text output to stdout, diagnostics to stderr on error)

**Affected Files/Modules:**
- `internal/cli/export.go` — Modify `RunExport` to collect all diagnostics, call the new wrapper function, always write JSON to stdout, and return the appropriate `LintError` exit code. The JSON path (default/`"json"` format) uses the new wrapper; the CUE path (`"cue"` format) stays unchanged.
- `internal/cli/export_test.go` — Update existing tests:
  - "valid file outputs model JSON to stdout" — update JSON path to dig into `model` wrapper
  - "valid file output includes actors and contexts" — update JSON path to dig into `model` wrapper
  - "file with validation errors writes diagnostics to stderr" — change to verify JSON body on stdout with diagnostics array and still has non-zero exit
  - "file with only lint warnings writes diagnostics to stderr with non-zero exit" — change to verify JSON body on stdout with diagnostics array and still has non-zero exit
  - "unparseable file writes diagnostics to stderr with non-zero exit" — change to verify JSON body on stdout with diagnostics array and still has non-zero exit
  - "default format is json and produces valid output" — update to check wrapper structure

**Patterns to Follow:**
- Follow the existing `RunExport` flow in `internal/cli/export.go:19-88` — keep the lex/parse/validate/lint pipeline intact; only change the output and error-return logic for the JSON case.
- Follow the existing `captureStdout` test helper pattern from `internal/cli/lint_test.go:17-34` for capturing stdout in tests.
- Follow the existing `captureStderr` test helper pattern from `internal/cli/export_test.go:17-34` for testing stderr output in CUE-related tests.

**Testable:** Yes

**Verification:** `go test ./internal/cli/ -tags=unit -run TestExport` passes.

**Depends on:** Task 2 (the wrapper function must exist before the CLI can use it)

---

## Summary

**Total tasks:** 3

**Ordering rationale:** Dependency-first. Task 1 adds positions to the JSON output (purely additive, no behavioral changes). Task 2 adds the diagnostics wrapper function (additive, no existing code broken). Task 3 changes the CLI behavior to use the wrapper (only task that modifies observable CLI behavior). Each task is independently committable with a green codebase.

**Acceptance criteria coverage:**
| Criterion | Task(s) |
|---|---|
| Source positions in JSON | Task 1 |
| Full hierarchy preserved | Task 1 (adds positions to existing hierarchy) |
| Field types and modifiers | Already implemented; verified by existing tests |
| Parse errors produce JSON with diagnostics array | Tasks 2 + 3 |
| Valid file produces empty diagnostics array | Tasks 2 + 3 |

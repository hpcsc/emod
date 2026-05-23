## Progress
- [x] Task 1: Create and bundle the emod CUE schema definition
- [x] Task 2: Add ExportCUE function to serialize AST model to CUE text
- [x] Task 3: Wire `-f cue` format support in the CLI export command
- [x] Task 4: Add `emod schema` CLI command to print the bundled CUE schema

---

## Story Reference

**US-009** — Convert .emod to CUE for external validation (`user-stories/emod-dsl-and-diagrams.md`)

As a model author, I want to export my model as CUE so that I can use CUE's constraint system for additional custom validation beyond the built-in rules.

**Acceptance Criteria:**
- [ ] `emod export reservation.emod -f cue` outputs a CUE file that conforms to the emod CUE schema
- [ ] The exported CUE file can be validated with `cue vet` against the emod schema without errors
- [ ] Round-trip fidelity: exporting to CUE and re-importing produces an equivalent model
- [ ] The emod CUE schema definition is bundled with the tool and can be printed via `emod schema --format cue`

**Depends on:** US-008

---

## Codebase Context

**Exploration findings:**

- **AST types** (`internal/ast/ast.go`): Full model hierarchy — `Model`, `Actor`, `Context`, `Aggregate`, `Slice`, `Command`, `Event`, `Field`, `Flow`, `Trigger`, `View`, `Automation`, `Translation`. Each type has `Name`, `Comments`, and positional metadata. Fields carry `Name`, `Type`, and `Modifier`. Events carry `Source`/`ExternalName`. Automations carry `TargetContext`. Translations carry `ExternalSystem`, `Reads`, `Command`, and a nested `Event`.

- **Export package** (`internal/export/export.go`): Already has `ExportJSON(model *ast.Model) ([]byte, error)` and intermediate JSON types (`jsonModel`, `jsonActor`, etc.) with `convertModel` and per-type `convert*` functions. Follows a "convert to intermediate types → marshal" pattern.

- **Formatter package** (`internal/formatter/formatter.go:10-15`): Uses a `writer` struct with a `strings.Builder` and recursive `writeModel`/`writeContext`/`writeAggregate`/`writeSlice`/etc. methods to produce formatted text output from `*ast.Model`. This is the closest analogue to how the CUE export should work (text generation from AST).

- **CLI export** (`internal/cli/export.go:19-25`): `RunExport(path, format string) error` currently validates `format == "json"` and rejects all other formats. Error handling uses `LintError` with `Message` and `ExitCode`.

- **CLI app** (`internal/cli/app.go:44-91`): Commands registered as `*urfave.Command`. Error handling pattern uses `errors.As` + `LintError` + `urfave.Exit`. Usage strings describe supported formats inline.

- **No CUE files exist in the project**: No `*.cue` files, no `//go:embed` usage, no `internal/cue/` directory.

- **Test conventions** (`internal/cli/*_test.go`, `internal/export/export_test.go`): `//go:build unit` build tag, `package cli_test`, umbrella `Test{Name}` wrapping `t.Run` groups. `writeTemp(t, name, content)` from `validate_test.go:555-562`. `captureStdout(t, fn)` from `lint_test.go:17-34`. `captureStderr(t, fn)` from `export_test.go:17-34`. Uses `github.com/stretchr/testify/require`.

- **Internal test helpers** (`internal/test/assertions.go`): `RequireEqual` using `google/go-cmp`.

- **CUE tool available** at `/Users/davidnguyen/.local/share/mise/installs/cue/0.14.2/cue`. CUE 0.14.2 is not a Go dependency — it is an external CLI tool used for manual verification and round-trip testing.

---

## Tasks

### Task 1: Create and bundle the emod CUE schema definition

**Behavior:** A CUE schema file defines the valid structure of an emod model (model name, actors, contexts with nested aggregates, slices, commands, events, flows, triggers, views, automations, translations, and fields with types and modifiers). The schema is embedded into the Go binary using `//go:embed` so it can be printed at runtime and referenced as the canonical shape that exports must conform to.

**Acceptance Criteria:**
- [ ] A CUE schema file exists at `internal/cue/schema.cue` defining the emod model structure
- [ ] The schema is embedded into the Go binary via `//go:embed` and accessible through a Go API
- [ ] The schema covers all model elements: model name, actors, contexts, aggregates, slices, triggers, commands, events, fields (with type and modifier), flows, views, automations, translations
- [ ] The schema does not reference external CUE packages so it remains self-contained and portable

**Affected Files/Modules:**
- `internal/cue/schema.cue` — New file: the CUE schema definition
- `internal/cue/embed.go` — New file: Go package that embeds `schema.cue` and exports it (e.g., `Schema string` variable or `Schema() string` function). Uses `//go:embed` directive.

**Patterns to Follow:**
- The CUE schema structure should mirror the JSON output shape from `internal/export/export.go:17-110` (the `jsonModel`, `jsonActor`, etc. types), since the CUE export will produce the same logical structure.
- The `//go:embed` directive follows Go standard library conventions — no existing usage in the project to reference, but the pattern is well-documented.

**Testable:** Yes — Tests verify the embedded schema is non-empty, valid CUE (parseable by the CUE tool), and contains expected field names matching the JSON intermediate types.

**Verification:** `go build` succeeds, the embedded schema string is non-empty, and running `cue vet` on the schema with a minimal instance does not error.

**Depends on:** None

---

### Task 2: Add ExportCUE function to serialize AST model to CUE text

**Behavior:** A new `ExportCUE(model *ast.Model) ([]byte, error)` function in the `internal/export` package converts a parsed AST model into CUE text output. The output uses the CUE list-of-structs format (as described in the user story) representing all model elements. The exported CUE conforms to the schema from Task 1 so that `cue vet schema.cue exported.cue` passes.

**Acceptance Criteria:**
- [ ] `ExportCUE` produces valid CUE text output for a model with actors, contexts, aggregates, slices, triggers, commands, events, fields, flows, views, automations, and translations
- [ ] Field types and modifiers (`required`/`optional`) are preserved in the CUE output
- [ ] Cross-context references (event source/external name, automation target context, view subscribes, translation details) are included
- [ ] The exported CUE conforms to the schema from Task 1 — verified by running `cue vet schema.cue exported.cue`
- [ ] Round-trip fidelity: exporting to CUE, converting CUE to JSON via `cue export`, and comparing with direct `ExportJSON` output produces equivalent structures
- [ ] A nil or empty model is handled without panicking (produces empty/minimal output)
- [ ] The function handles all node types including triggers, events with external sources, views with subscribes, automations with target contexts, translations with nested events, and flows

**Affected Files/Modules:**
- `internal/export/export.go` — Add `ExportCUE(model *ast.Model) ([]byte, error)` function and supporting conversion logic
- `internal/export/export_test.go` — Add CUE export tests under a new `TestExportCUE` umbrella, including round-trip fidelity tests

**Patterns to Follow:**
- `internal/formatter/formatter.go:10-15` and the `writer` struct pattern (lines 17-31) — Pattern for a package that traverses `*ast.Model` and produces text output. The CUE export should use a similar recursive traversal approach, generating CUE syntax instead of `.emod` syntax.
- `internal/export/export.go:113-116` — Pattern for the exported function signature (`ExportJSON(model *ast.Model) ([]byte, error)`).
- `internal/export/export.go:118-129` — Pattern for the top-level `convertModel` function that dispatches to per-type converters. The CUE export should follow a similar recursive dispatch through model → contexts → aggregates → slices → commands/events/etc.
- `internal/export/export_test.go:14-30` — Pattern for export unit tests using programmatically constructed `*ast.Model` values and verifying output structure.

**Testable:** Yes — Tests construct models programmatically, verify the CUE output is parseable by `cue export`, and compare round-tripped JSON with direct JSON export.

**Verification:** Unit tests pass with programmatic model construction. Manual verification: `cue vet schema.cue exported.cue` on sample output.

**Depends on:** Task 1

---

### Task 3: Wire `-f cue` format support in the CLI export command

**Behavior:** `emod export reservation.emod -f cue` reads an `.emod` file, runs the lex/parse/validate/lint pipeline, and if clean, outputs the CUE representation (produced by ExportCUE from Task 2) to stdout. The `-f` flag usage string and error messages reflect that both "json" and "cue" are supported. If the model has any diagnostics, they are printed to stderr with a non-zero exit code (same behavior as JSON export).

**Acceptance Criteria:**
- [ ] `emod export reservation.emod -f cue` outputs CUE text to stdout for a valid `.emod` file
- [ ] The `-f` flag usage string lists both "json" and "cue" as supported formats
- [ ] An unsupported format still returns an appropriate error listing supported formats
- [ ] If the file has validation errors, diagnostics are written to stderr and exit code is non-zero (same behavior as the JSON path)
- [ ] If the file has only lint warnings, diagnostics are written to stderr and exit code is non-zero
- [ ] The default format remains "json" (backward compatible)

**Affected Files/Modules:**
- `internal/cli/export.go` — Update `RunExport` to accept format "cue", update error message listing supported formats, call `export.ExportCUE` for the "cue" format
- `internal/cli/app.go` — Update the `export` command usage string to mention "cue" as a supported format
- `internal/cli/export_test.go` — Add tests for the `-f cue` path using `writeTemp` and `captureStdout`

**Patterns to Follow:**
- `internal/cli/export.go:73-81` — Pattern for calling the export function and writing output to stdout with `fmt.Println`.
- `internal/cli/export.go:54-71` — Pattern for diagnostic handling (write to stderr, return `LintError` with appropriate exit code).
- `internal/cli/export_test.go:37-49` — Pattern for testing export output with `captureStdout`.
- `internal/cli/export_test.go:72-88` — Pattern for testing diagnostic output with `captureStderr`.

**Testable:** Yes — Tests use `writeTemp` with valid and invalid `.emod` fixtures, verify CUE output on stdout for valid files, and diagnostic output on stderr for invalid files.

**Verification:** Tests pass. Manual: `emod export examples/hotel.emod -f cue` produces valid CUE text.

**Depends on:** Task 2

---

### Task 4: Add `emod schema` CLI command to print the bundled CUE schema

**Behavior:** Running `emod schema --format cue` prints the bundled CUE schema definition (from Task 1) to stdout. Currently only the "cue" format is supported, but the `--format` flag is designed to allow future schema formats (e.g., JSON Schema). This gives users a way to see the exact schema their exported models must conform to.

**Acceptance Criteria:**
- [ ] `emod schema --format cue` prints the embedded CUE schema to stdout
- [ ] An unsupported format returns an appropriate error (e.g., "unsupported format")
- [ ] The schema output is valid CUE (same content as `internal/cue/schema.cue`)
- [ ] The command is registered in the `emod` CLI and appears in help output

**Affected Files/Modules:**
- `internal/cli/schema.go` — New file: `RunSchema(format string) error` function that reads the embedded schema and prints it for the "cue" format
- `internal/cli/schema_test.go` — New file: tests using `captureStdout`
- `internal/cli/app.go` — Register the `schema` command with `--format` flag

**Patterns to Follow:**
- `internal/cli/export.go:19-25` — Pattern for format validation and `LintError` for unsupported formats.
- `internal/cli/export.go:73-81` — Pattern for printing output to stdout.
- `internal/cli/app.go:44-91` — Pattern for registering a command with flags and `Action` closure with `LintError` handling.

**Testable:** Yes — Tests use `captureStdout` to verify the schema is printed to stdout, and verify error handling for unsupported formats.

**Verification:** Tests pass. Manual: `emod schema --format cue` prints the CUE schema.

**Depends on:** Task 1

---

## Summary

- **Total tasks:** 4
- **Ordering rationale:** Dependency-first. Task 1 (schema) is foundational since the schema is referenced by both the export function and the schema command. Task 2 (ExportCUE) depends on having the schema to align output structure. Task 3 (CLI -f cue) wires Task 2's function into the user-facing command. Task 4 (schema command) depends on the schema from Task 1 but is independent of the export function.
- **Acceptance criteria coverage:**
  - AC 1 (`emod export -f cue` outputs CUE) — Task 3
  - AC 2 (exported CUE validates with `cue vet` against schema) — Task 2 (verified in tests)
  - AC 3 (round-trip fidelity) — Task 2 (verified via CUE-to-JSON comparison test)
  - AC 4 (`emod schema --format cue`) — Task 4
- **Coverage is complete:** All four ACs are covered across the four tasks. No AC is deferred.

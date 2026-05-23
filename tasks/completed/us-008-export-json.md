## Progress
- [x] Task 1: Create `internal/export` package to serialize AST model to JSON
- [x] Task 2: Wire the `export` CLI command with read/parse/validate/export pipeline

---

## Story Reference

**US-008** — Export model as JSON (`user-stories/emod-dsl-and-diagrams.md`)

As an AI agent, I want to export a parsed and validated model as JSON so that I can consume and manipulate event models programmatically.

**Acceptance Criteria:**
- [ ] `emod export reservation.emod -f json` outputs the full model as a JSON document to stdout
- [ ] The JSON structure includes `model.name`, `model.actors`, `model.contexts` with nested aggregates, slices, commands, events, views, automations, and translations
- [ ] Field types and modifiers (required, optional) are preserved in the JSON output
- [ ] Cross-context references and external source metadata are included
- [ ] If the model has validation errors, the command exits with a non-zero code and prints errors to stderr instead of emitting JSON

**Depends on:** US-007

---

## Codebase Context

**Exploration findings:**

- **AST types** (`internal/ast/ast.go`): The full model hierarchy is defined here — `Model`, `Actor`, `Context`, `Aggregate`, `Slice`, `Command`, `Event`, `Field`, `Flow`, `Trigger`, `View`, `Automation`, `Translation`. Each type has positional metadata (`Position`) and `Comments`. Fields carry `Name`, `Type`, and `Modifier` (e.g. `required`/`optional`). Events optionally carry `Source`/`ExternalName` for cross-context references. Automations carry `TargetContext`. Translations carry `ExternalSystem`, `Reads`, `Command`, and a nested `Event`.

- **CLI command pattern** (`internal/cli/app.go`): Commands are registered as `*urfave.Command` entries with `Action` closures that call `Run{Command}` functions. Error handling uses `LintError` (defined in `lint.go`) with `Message` and `ExitCode`. Errors are printed to stderr and return `urfave.Exit("", code)`.

- **Read/Parse/Validate pipeline** (`internal/cli/validate.go`): The standard flow is `os.ReadFile` → `lexer.Scan` → `parser.New(...).Parse()` → `validator.Validate(model)` → `linter.Lint(model)`. Diagnostics accumulate from each stage.

- **JSON output pattern** (`internal/cli/lint.go:24-66`): A `jsonEntry` struct with `json:"..."` tags is used with `json.Marshal`. The `formatJSON` function marshals diagnostics and writes to stdout via `fmt.Println`. Exit codes are 1 (warnings only) or 2 (errors present).

- **Test conventions** (`internal/cli/*_test.go`): `//go:build unit` build tag, `package cli_test`, umbrella `Test{Name}` wrapping `t.Run` groups, `writeTemp(t, name, content)` and `captureStdout(t, fn)` helpers from lint_test.go.

- **No export package exists yet**: No `internal/export/` directory or export-related code found.

- **Formatter pattern** (`internal/formatter/formatter.go`): A dedicated package that takes `*ast.Model` and produces an output representation. This is the closest analogue to the export serialization task.

---

## Tasks

### Task 1: Create `internal/export` package to serialize AST model to JSON

**Behavior:** A new `internal/export` package provides a function that takes a parsed `*ast.Model` and produces a JSON byte slice. The JSON output represents the complete model structure including all nested types, field types and modifiers, cross-context references (external source metadata on events, target context on automations), and flow connections.

**Acceptance Criteria:**
- [ ] The JSON output includes `model.name`, `model.actors`, `model.contexts` with nested aggregates, slices, commands, events, views, automations, and translations
- [ ] Field types and modifiers (`required`/`optional`) are preserved in the JSON output
- [ ] Cross-context references (`automation.target_context`, `event.source`, `translation.external_system`, `view.subscribes`) are included
- [ ] Flow entries are included as command-to-event mappings
- [ ] Trigger details (kind, name, actor, reads) are included
- [ ] The function handles a nil or empty model without panicking

**Affected Files/Modules:**
- `internal/export/export.go` — New file: exported function `MarshalModel(model *ast.Model) ([]byte, error)` (or similar)
- `internal/export/export_test.go` — New file: unit tests with programmatically constructed AST models

**Patterns to Follow:**
- `internal/formatter/formatter.go` (entire file) — Pattern for a package that transforms `*ast.Model` into an output representation. Study the exported function signature, package structure, and recursive traversal of the AST hierarchy.
- `internal/cli/lint.go:24-30` — Pattern for defining structs with `json:"..."` struct tags for controlled JSON serialization.

**Testable:** Yes

**Verification:** Unit tests pass with programmatically constructed `*ast.Model` inputs verifying JSON output structure, field types/modifiers, and cross-context references.

**Depends on:** None

---

### Task 2: Wire the `export` CLI command with read/parse/validate/export pipeline

**Behavior:** Running `emod export <file> -f json` reads an `.emod` file, lexes, parses, validates, lints, and if the model is clean, outputs the JSON representation (from Task 1) to stdout. If the model has any diagnostics (parser errors, validation failures, lint warnings), the command prints them to stderr and exits with a non-zero exit code instead of emitting JSON.

**Acceptance Criteria:**
- [ ] `emod export reservation.emod -f json` outputs the full model JSON to stdout for a valid file
- [ ] If the file has validation errors, diagnostics are written to stderr and exit code is non-zero
- [ ] If the file has only lint warnings, diagnostics are written to stderr and exit code is non-zero
- [ ] If the file is unparseable, diagnostics are written to stderr and exit code is non-zero
- [ ] A missing file argument or nonexistent file returns an appropriate error
- [ ] The `-f` flag defaults to `json` (the only supported format for export)

**Affected Files/Modules:**
- `internal/cli/export.go` — New file: `RunExport(path string) error` function following the validate pattern
- `internal/cli/export_test.go` — New file: tests using `writeTemp` and `captureStdout`
- `internal/cli/app.go` — Register the `export` command with `-f` flag

**Patterns to Follow:**
- `internal/cli/validate.go:13-61` — Pattern for the CLI handler function: read file, lex, parse, validate, lint, then decide output based on diagnostics.
- `internal/cli/app.go:64-91` — Pattern for registering a command (`lint`) with flags and error handling using `LintError` and `urfave.Exit`.
- `internal/cli/lint.go:15-22` — Pattern for `LintError` used as the command error type.
- `internal/cli/lint_test.go:17-34` — Pattern for `captureStdout` test helper.
- `internal/cli/validate_test.go:555-562` — Pattern for `writeTemp` test helper.
- `internal/cli/validate_test.go:373-402` — Pattern for testing JSON output CLI commands with `captureStdout`.

**Testable:** Yes

**Verification:** Tests pass using `writeTemp` to create `.emod` fixtures and `captureStdout`/capture stderr to verify JSON output and error output. Different fixture types exercise the valid model path, the validation-error path, and the parse-error path.

**Depends on:** Task 1

---

## Summary

- **Total tasks:** 2
- **Ordering rationale:** Dependency-first. Task 1 creates the core serialization logic as a standalone package. Task 2 wires it into the CLI. This ordering ensures Task 2 can import the export package and focus on CLI concerns.
- **Acceptance criteria coverage:**
  - AC 1 (CLI command output) — Task 2
  - AC 2 (JSON structure with all nested types) — Task 1
  - AC 3 (field types and modifiers preserved) — Task 1
  - AC 4 (cross-context references and external source metadata) — Task 1
  - AC 5 (validation errors → stderr, non-zero exit) — Task 2
- **Coverage is complete:** All five ACs are covered across the two tasks. No AC is deferred.

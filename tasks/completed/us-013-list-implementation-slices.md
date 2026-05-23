## Progress
- [x] Task 1: Register `slices` command and implement text table output
- [x] Task 2: Add `--format json` support to `slices` command

## Story Reference
**US-013 — List implementation slices**

> As a model author, I want to list all slices in my model with their pattern type so that I can plan implementation work and see the scope of the model at a glance.

**Acceptance Criteria:**
1. `emod slices reservation.emod` prints a table with columns: slice name, pattern type (command/view/automation/translation), context, and key elements (command name, event name, or view name)
2. Slices are listed in the order they appear in the model, grouped by context
3. `emod slices --format json` outputs the same information as a JSON array
4. If the model has parse errors, the command fails with a descriptive message

**Depends on:** US-001

## Codebase Context

- CLI commands are registered in `internal/cli/app.go` as `*urfave.Command` entries with `Action` closures calling `Run{Command}` functions defined in `internal/cli/*.go` files.
- Error handling uses `LintError` (defined in `internal/cli/lint.go:15-23`) with `Message` and `ExitCode`.
- The AST model is in `internal/ast/ast.go`. A `Slice` contains `Commands`, `Events`, `Views`, `Automations`, `Translations`, `Flows`, `Trigger`, and `Fields`.
- Existing test fixtures: `validEmod` in `internal/cli/validate_test.go:16-98` contains a comprehensive model with all 4 pattern types:
  - Command: "Make Reservation" (trigger, command, event, flow)
  - View: "View Reservations" (view)
  - Automation: "Auto Confirm Reservation" (command, flow, automation)
  - Translation: "Import External Booking" (command, flow, translation)
- `invalidEmod` in `internal/cli/validate_test.go:100-102` is used for parse-error tests.
- Tests use `//go:build unit`, `package cli_test`, `require` from testify, `writeTemp(t, name, content)` helper in `validate_test.go:555-562`, and `captureStdout(t, fn)` helper in `lint_test.go:17-34`.
- `captureStderr(t, fn)` helper is in `export_test.go:17-34`.
- The standard file-processing pipeline across all commands is: `os.ReadFile` → `lexer.Scan` → `parser.New(...).Parse()`.
- `collectSlices` in `internal/diagram/drawio.go:282-292` demonstrates flattening slices from a model while preserving context name.

## Tasks

### Task 1: Register `slices` command and implement text table output

**Behavior:** Add a new `slices` subcommand to the CLI. When invoked with a valid `.emod` file, it prints a text table listing every slice in the model with its name, inferred pattern type, enclosing context, and key elements. If the file has parse, validation, or lint errors, the command fails with a descriptive message and non-zero exit code.

**Acceptance Criteria:**
- [ ] `emod slices <file>` prints a table with columns: slice name, pattern type, context, and key elements
- [ ] Slices are listed in the order they appear in the model, grouped by context
- [ ] If the model has parse errors, the command fails with a descriptive message

**Affected Files/Modules:**
- `internal/cli/app.go` — register the new `slices` command
- `internal/cli/slices.go` — new file containing `RunSlices` function with text table formatting, pattern detection, and standard error handling
- `internal/cli/slices_test.go` — new test file covering happy path (text output, ordering, key elements), missing file, nonexistent file, and parse-error behaviour

**Patterns to Follow:**
- Follow the command registration pattern in `internal/cli/app.go:15-43` for adding the new `slices` command.
- Follow the `RunValidate` pattern in `internal/cli/validate.go:17-61` for the file-processing pipeline and error handling.
- Follow the diagram error-handling pattern in `internal/cli/diagram.go:52-70` for writing diagnostics to stderr and returning the appropriate `LintError` exit code.
- Follow the `collectSlices` pattern in `internal/diagram/drawio.go:282-292` for traversing contexts, aggregates, and slices.
- Follow the test organization in `internal/cli/validate_test.go` and `internal/cli/lint_test.go` for subtest naming, fixture construction, and assertion style.

**Testable:** Yes

**Verification:** `go test -tags=unit ./internal/cli/...` passes

**Depends on:** None

### Task 2: Add `--format json` support to `slices` command

**Behavior:** Extend the `slices` command with a `--format` flag that supports `json`. When `--format json` is passed, the command outputs a JSON array containing the same slice information (name, pattern type, context, key elements) instead of the text table. Parse/validation errors still go to stderr and return a non-zero exit code; the JSON output only occurs for clean models.

**Acceptance Criteria:**
- [ ] `emod slices --format json <file>` outputs a JSON array with slice name, pattern type, context, and key elements
- [ ] Unsupported formats return a clear error

**Affected Files/Modules:**
- `internal/cli/app.go` — add the `--format` flag to the `slices` command registration
- `internal/cli/slices.go` — extend `RunSlices` to accept a format parameter and branch between text table and JSON output
- `internal/cli/slices_test.go` — add subtests for JSON format, unsupported format, and JSON output structure validation

**Patterns to Follow:**
- Follow the flag declaration pattern in `internal/cli/app.go:20-26` for adding the `--format` string flag to the new command.
- Follow the JSON formatting pattern in `internal/cli/lint.go:32-66` for encoding and printing JSON output.
- Follow the existing test pattern in `internal/cli/lint_test.go:131-140` and `internal/cli/export_test.go:36-49` for verifying JSON output and unsupported format errors.

**Testable:** Yes

**Verification:** `go test -tags=unit ./internal/cli/...` passes

**Depends on:** Task 1

## Summary

- **Total tasks:** 2
- **Ordering rationale:** Task 1 delivers the primary user-facing behavior (text table output, error handling, command registration). Task 2 builds on it by adding the JSON format option. Both are independently committable vertical slices.
- **Acceptance criteria coverage:**
  - AC 1 (text table) → Task 1
  - AC 2 (ordering, grouped by context) → Task 1
  - AC 3 (JSON output) → Task 2
  - AC 4 (parse errors) → Task 1
- **Nothing deferred:** All acceptance criteria are covered.

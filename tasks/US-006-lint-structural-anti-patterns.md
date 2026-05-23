# US-006: Lint models for structural anti-patterns

## Progress
- [x] Task 1: Add structural anti-pattern detection rules (left chair, right chair, clickbait event)
- [ ] Task 2: Add `--format json` flag and severity-aware exit codes

## Story Reference

**Source:** Inline user story — US-006: Lint models for structural anti-patterns

**Summary:** As a model author, I want the linter to detect structural problems like the left chair, right chair, and clickbait event patterns so that my model avoids known architectural pitfalls.

**Acceptance Criteria (from story):**
- A command that produces 3 or more events triggers a "left chair" warning suggesting the command may contain multiple decisions
- A view that subscribes to 5 or more events triggers a "right chair" / "god view" warning
- An event whose fields contain only a single ID field triggers a "clickbait event" warning suggesting the payload should include relevant business data
- Warnings are categorized by severity: `error` for structural violations, `warning` for naming conventions
- `emod lint --format json` outputs a JSON array of diagnostics with `file`, `line`, `rule`, `severity`, and `message` fields
- Exit code is 2 for errors, 1 for warnings only, 0 for clean

**Depends on:** US-005 (completed — naming convention linting)

## Codebase Context

**Project structure:** Go CLI tool using urfave/cli v2. Entry point at `cmd/emod/main.go`, CLI commands wired in `internal/cli/app.go`. Existing commands: `validate`, `fmt`, `lint`.

**AST types (`internal/ast/ast.go`):**
- `Flow` (line 93-99) maps `CommandName` to `EventName` within a slice — used to detect how many events a single command produces (left chair)
- `View` (line 115-123) has `Subscribes []string` — used to detect views subscribing to 5+ events (right chair)
- `Event` (line 71-82) has `Fields []*Field` — used to detect events with only a single ID field (clickbait)
- `Field` (line 84-91) has `Name`, `Type`, `Modifier` — the `Name` determines if a field is an ID reference
- Model hierarchy: `Model` -> `Contexts` -> `Aggregates` -> `Slices` -> `Commands`, `Events`, `Views`, `Flows`

**Existing linter (`internal/linter/linter.go`):** The `Lint` function (line 25-58) walks the AST and returns naming convention diagnostics. It currently uses a `warning()` helper (line 14-23) that produces `diagnostic.Warning` severity entries. The `checkEvent` function (line 60-71) dispatches to individual rule checks.

**Diagnostic type (`internal/diagnostic/entry.go`):** `Entry` has `Filename`, `Line`, `Column`, `Message`, `Severity` (int — `Error=0`, `Warning=1`), and `RuleName` fields. The `Severity` type and constants are defined at lines 5-10. The `String()` method (line 21-26) formats with rule name in brackets when present.

**CLI lint command (`internal/cli/lint.go`):** `RunLint` (line 14-43) reads a file, lexes, parses, runs the linter, and returns a single error with all diagnostics concatenated. Currently there is no `--format` flag support.

**CLI app (`internal/cli/app.go`):** The lint command is registered at line 48-60. The action always returns `urfave.Exit("", 1)` for any error (line 56). There is no routing by severity to different exit codes.

**Test patterns:**
- `internal/linter/linter_test.go`: Single `TestLint` function with `t.Run` groups per rule. Tests construct `*ast.Model` in-memory and assert on returned `[]*diagnostic.Entry` using `github.com/stretchr/testify/require`.
- `internal/cli/lint_test.go`: Single `TestLint` function. Tests use a `writeTemp` helper (defined in `internal/cli/validate_test.go:285-292`) to create temp `.emod` files, call `RunLint`, and assert on the returned error.

## Tasks

### Task 1: Add structural anti-pattern detection rules (left chair, right chair, clickbait event)

**Behavior:** The linter detects three structural anti-patterns in the model. Left chair: a command that produces 3 or more events triggers a warning. Right chair (god view): a view that subscribes to 5 or more events triggers a warning. Clickbait event: an event whose fields contain only a single ID field triggers a warning. All three rules produce diagnostics with `Error` severity, while the existing naming convention rules continue to produce `Warning` severity.

**Acceptance Criteria:**
- [ ] A command with 3 or more flows referencing it triggers a "left-chair" diagnostic with `Error` severity
- [ ] A command with 1 or 2 flows referencing it produces no left-chair diagnostic
- [ ] A view subscribing to 5 or more events triggers a "god-view" diagnostic with `Error` severity
- [ ] A view subscribing to 4 or fewer events produces no god-view diagnostic
- [ ] An event with exactly one field whose name indicates it is an ID reference triggers a "clickbait-event" diagnostic with `Error` severity
- [ ] An event with multiple fields (including an ID field) produces no clickbait-event diagnostic
- [ ] An event with one non-ID field produces no clickbait-event diagnostic
- [ ] Each structural diagnostic carries the file path and line number from the relevant AST node's position
- [ ] Existing naming convention diagnostics (state-obsession, property-sourcing, command-in-disguise, command-past-tense, view-naming) continue to produce `Warning` severity
- [ ] All five existing naming rules continue to fire alongside the three new structural rules in a single `Lint` invocation

**Affected Files/Modules:**
- `internal/linter/linter.go` — add an `error()` helper (similar to the existing `warning()` at line 14-23), add `checkLeftChair`, `checkRightChair`, `checkClickbaitEvent` functions, wire them into the `Lint` function
- `internal/linter/linter_test.go` — add test groups for the three new rules

**Patterns to Follow:**
- Follow the existing checker function signature pattern in `internal/linter/linter.go:60-71` for consistent function naming and return types
- Follow the existing `warning()` helper at `internal/linter/linter.go:14-23` when creating the `error()` variant
- Follow the AST traversal pattern in `internal/linter/linter.go:32-55` for iterating contexts, aggregates, slices, and elements
- Follow the test structure in `internal/linter/linter_test.go:21-56` for constructing in-memory AST models and asserting diagnostics

**Testable:** Yes

**Verification:** `go test -tags unit ./internal/linter/...` passes. Tests verify each structural rule fires correctly for violating inputs and stays silent for compliant inputs. Tests verify severity is `Error` for structural rules and remains `Warning` for existing naming rules.

**Depends on:** None

---

### Task 2: Add `--format json` flag and severity-aware exit codes

**Behavior:** The `emod lint` command gains a `--format` flag that accepts `json` (outputs a JSON array of diagnostics) in addition to the default text format. Diagnostics in JSON format include `file`, `line`, `rule`, `severity` (as string "error" or "warning"), and `message` fields. The command exit code changes from always-1 to three levels: 0 for clean (no diagnostics), 1 for warnings only, and 2 when any error-severity diagnostic is present. Infrastructure failures (missing file, parse error) still exit 1.

**Acceptance Criteria:**
- [ ] `emod lint --format json clean.emod` outputs `[]` (empty JSON array) and exits 0
- [ ] `emod lint --format json problematic.emod` (with naming warnings only) outputs a JSON array of diagnostics each with `file`, `line`, `rule`, `severity` (string `"warning"`), and `message` fields, and exits 1
- [ ] `emod lint --format json structural.emod` (with structural errors) outputs a JSON array with at least one diagnostic having `severity: "error"`, and exits 2
- [ ] `emod lint --format json mixed.emod` (with both naming warnings and structural errors) outputs diagnostics of both severities and exits 2
- [ ] `emod lint` (default text format) produces the same text output as before for all diagnostics
- [ ] `emod lint --format unknown` produces an error message and exits 1
- [ ] `emod lint` (no file argument) produces an error message and exits 1
- [ ] `emod lint nonexistent.emod` produces an error message and exits 1
- [ ] Existing `emod lint` behavior (text output, exit code 1 for any diagnostic) is preserved when `--format` is not specified and diagnostics are only warnings

**Affected Files/Modules:**
- `internal/cli/lint.go` — add `--format` flag parsing, implement JSON output path, change return type or add a mechanism to communicate max severity to the caller
- `internal/cli/lint_test.go` — add tests for JSON output format, test each exit code scenario (0, 1, 2), test unknown format
- `internal/cli/app.go:48-60` — update the lint command action to route exit codes based on max severity (2 for errors, 1 for warnings, 0 for clean)
- `internal/diagnostic/entry.go` — add JSON serialization support for the `Entry` type so that `file`, `line`, `rule`, `severity` (as string), and `message` are serialized correctly

**Patterns to Follow:**
- Follow the existing flag pattern in `internal/cli/app.go:32-36` for adding the `--format` flag to the lint command definition
- Follow the existing action pattern in `internal/cli/app.go:48-60` for reading flag values and calling `RunLint`
- Follow the existing `RunLint` pattern in `internal/cli/lint.go:14-43` for the overall file processing flow
- Follow the test pattern in `internal/cli/lint_test.go:13-19` for testing with temp `.emod` files and asserting on error values
- Follow the existing test helper `writeTemp` in `internal/cli/validate_test.go:285-292` for creating temp files

**Testable:** Yes

**Verification:** `go test -tags unit ./internal/cli/...` and `go test -tags unit ./internal/diagnostic/...` pass. Tests verify JSON output structure, text output preservation, and each exit code scenario (0 for clean, 1 for warnings only, 2 for errors, 1 for infrastructure failures).

**Depends on:** Task 1

## Summary

**Total tasks:** 2

**Ordering rationale:** Dependency-first, business logic before wiring:
1. Task 1 (structural lint rules) builds the core detection logic and introduces Error-severity diagnostics alongside the existing Warning-severity naming rules.
2. Task 2 (JSON format and exit codes) wires the diagnostics into the CLI with the correct output format and exit code routing. It depends on Task 1 because the exit-code-2 path requires error-severity diagnostics to exist.

**Acceptance criteria coverage:**

| Story Acceptance Criterion | Covered By |
|---|---|
| Command with 3+ events triggers left chair warning | Task 1 |
| View with 5+ subscriptions triggers god view warning | Task 1 |
| Event with single ID field triggers clickbait event warning | Task 1 |
| Warnings categorized: `error` for structural, `warning` for naming | Task 1, Task 2 |
| `emod lint --format json` outputs JSON array with file/line/rule/severity/message | Task 2 |
| Exit code 2 for errors, 1 for warnings, 0 for clean | Task 2 |

**Deferred:** None. All six acceptance criteria from the story are covered.

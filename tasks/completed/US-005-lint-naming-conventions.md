# US-005: Lint Models for Naming Convention Violations

## Progress
- [x] Task 1: Add a diagnostic severity level to distinguish warnings from errors
- [x] Task 2: Implement event naming lint rules (state obsession, property sourcing, command-in-disguise)
- [x] Task 3: Implement command and view naming lint rules
- [x] Task 4: Wire up the `emod lint` CLI command

## Story Reference

**Source:** Inline user story -- US-005: Lint models for naming convention violations

**Summary:** As a model author, I want the linter to catch naming problems in my events, commands, and views so that my model follows event modeling conventions.

**Acceptance Criteria (from story):**
- Events with names ending in `Updated`, `Changed`, or `Modified` produce a "state obsession" warning with the event name and a suggestion to use a specific business fact name
- Events with names matching `[Entity][Field]Changed` produce a "property sourcing" warning
- Events with names ending in `Initiated` produce a warning that these are likely commands in disguise
- Commands that are not in imperative form (e.g. past tense) produce a warning
- Views whose names do not end in `View` produce a warning
- Each warning includes the file path, line number, rule name, and a human-readable explanation
- `emod lint` exits with code 0 when no warnings, code 1 when warnings are found

**Depends on:** US-001

## Codebase Context

**Project structure:** Go CLI tool using urfave/cli v2. Entry point at `cmd/emod/main.go`, CLI commands wired in `internal/cli/app.go`. Existing commands: `validate` (`internal/cli/validate.go`) and `fmt` (`internal/cli/fmt.go`).

**AST types (`internal/ast/ast.go`):** The AST carries all the node types needed for linting. `Event` has `Name` and `NamePos Position` (line 71-82). `Command` has `Name` and `NamePos Position` (line 62-69). `View` has `Name` and `NamePos Position` (line 115-123). The model tree is: `Model` -> `Context[]` -> `Aggregate[]` -> `Slice[]` -> `Command[]`, `Event[]`, `View[]`. Translation nodes can also contain inline `Event` nodes (line 149-152).

**Diagnostic type (`internal/diagnostic/entry.go`):** `Entry` has `Filename`, `Line`, `Column`, and `Message` fields. Its `String()` method formats as `filename:line: message`. The current type has no severity level or rule name field -- warnings from the linter need a way to include the rule name in the output, distinguishable from validation errors.

**Existing validator (`internal/validator/validator.go`):** The `Validate` function takes `*ast.Model` and returns `[]*diagnostic.Entry`. It walks the AST tree via `Model` -> `Contexts` -> `Aggregates` -> `Slices` and checks cross-references (automations, translations, views, flows). The linter is a distinct concern from the validator: the validator checks structural correctness (do referenced names exist?), while the linter checks naming conventions (do names follow best practices?).

**CLI command wiring pattern (`internal/cli/app.go:14-48`):** Commands are registered in the `Commands` slice of the `urfave.App`. Each command has `Name`, `Usage`, `ArgsUsage`, and an `Action` function that delegates to a `RunX` function.

**CLI command implementation pattern (`internal/cli/validate.go`):** `RunValidate` reads the file, lexes, parses, validates, collects diagnostics, and returns an error containing all diagnostic messages formatted as strings. The `fmt` command in `internal/cli/fmt.go` follows the same pattern.

**Test patterns:** Unit tests use `//go:build unit` build tag. Tests use `github.com/stretchr/testify/require`. The validator tests in `internal/validator/validator_test.go` construct `*ast.Model` in-memory and assert on the returned diagnostics. The CLI tests in `internal/cli/validate_test.go` use a `writeTemp` helper to create temp `.emod` files, call the `RunX` function, and assert on the returned error.

**Caller pattern analysis:** The linter package is an **Exported API** -- the CLI is the caller, depending on the return value (list of warnings). Tests should assert on the returned diagnostics for given AST inputs: warning count, rule names, messages, and positions. The CLI `lint` command is **Inbound** -- the user runs a command and observes exit code and stderr output.

## Tasks

### Task 1: Add a diagnostic severity level to distinguish warnings from errors

**Behavior:** The `diagnostic.Entry` type gains a severity level and a rule name so that lint warnings can be distinguished from parse/validation errors in the output. The `String()` method includes the rule name when present. All existing code continues to work unchanged because the new fields default to zero values.

**Acceptance Criteria:**
- [ ] `diagnostic.Entry` carries a severity field that can represent at least "error" and "warning"
- [ ] `diagnostic.Entry` carries a rule name field for lint rules
- [ ] `String()` includes the rule name in the formatted output when the rule name is non-empty (e.g., `file.emod:5: [state-obsession] message`)
- [ ] `String()` produces unchanged output when the rule name is empty (backward compatible with existing parse/validation errors)
- [ ] All existing tests pass without modification

**Affected Files/Modules:**
- `internal/diagnostic/entry.go` -- add severity and rule name fields, update `String()` formatting
- `internal/diagnostic/entry_test.go` -- add tests for the new formatting with rule name

**Patterns to Follow:**
- Follow the existing `Entry` struct definition in `internal/diagnostic/entry.go:5-10` for field naming conventions.
- Follow the existing `String()` method in `internal/diagnostic/entry.go:12-14` for the formatting approach.
- Follow the test pattern in `internal/diagnostic/entry_test.go:12-34` for testing the `String()` output.

**Testable:** Yes

**Verification:** `go test -tags unit ./...` passes. New tests verify `String()` output with and without the rule name field.

**Depends on:** None

---

### Task 2: Implement event naming lint rules (state obsession, property sourcing, command-in-disguise)

**Behavior:** A new `internal/linter` package provides a function that accepts an `*ast.Model` and returns lint warnings for event naming violations. Three rules are checked: (1) event names ending in `Updated`, `Changed`, or `Modified` produce a "state-obsession" warning; (2) event names matching the pattern of an entity name followed by a field name followed by `Changed` produce a "property-sourcing" warning; (3) event names ending in `Initiated` produce a "command-in-disguise" warning. Each warning carries the file path, line number, rule name, and a human-readable explanation. The function walks the full AST including inline events within translations.

**Acceptance Criteria:**
- [ ] An event named `OrderUpdated` produces a warning with rule name `state-obsession` and a message suggesting a specific business fact name
- [ ] An event named `AccountModified` produces a `state-obsession` warning
- [ ] An event named `CustomerChanged` produces a `state-obsession` warning
- [ ] An event named `OrderStatusChanged` produces a `property-sourcing` warning (entity `Order`, field `Status`)
- [ ] An event named `CustomerAddressChanged` produces a `property-sourcing` warning
- [ ] An event named `PaymentInitiated` produces a `command-in-disguise` warning suggesting it should be a command
- [ ] Each warning includes the file path and line number from the event's `NamePos`
- [ ] Events with compliant names (e.g., `OrderPlaced`, `RoomReserved`) produce no warnings
- [ ] Events defined inline within translations are also checked
- [ ] A nil model produces no warnings

**Affected Files/Modules:**
- `internal/linter/` (new package) -- linter implementation with event naming rules
- `internal/linter/linter_test.go` (new) -- unit tests constructing `*ast.Model` values and asserting on returned diagnostics

**Patterns to Follow:**
- Follow the AST traversal pattern in `internal/validator/validator.go:18-35` for walking `Model` -> `Contexts` -> `Aggregates` -> `Slices` -> `Events` and `Translations`.
- Follow the diagnostic construction pattern in `internal/validator/validator.go:44-49` for creating `diagnostic.Entry` values with position information.
- Follow the test structure in `internal/validator/validator_test.go:13-18` for constructing in-memory AST models and asserting on diagnostic output.

**Testable:** Yes

**Verification:** `go test -tags unit ./...` passes. Tests verify each rule produces the expected warning with correct rule name, message content, file path, and line number.

**Depends on:** Task 1

---

### Task 3: Implement command and view naming lint rules

**Behavior:** The linter gains two additional rules: (1) commands whose names appear to be past tense (not imperative form) produce a warning; (2) views whose names do not end in `View` produce a warning. These rules are added to the same linter function introduced in Task 2, so a single call checks all five naming rules.

**Acceptance Criteria:**
- [ ] A command named `OrderPlaced` (past tense) produces a warning with a rule name indicating it should be imperative
- [ ] A command named `ReservationCancelled` (past tense) produces a warning
- [ ] A command named `PlaceOrder` (imperative) produces no warning
- [ ] A command named `CancelReservation` (imperative) produces no warning
- [ ] A view named `OrderList` (not ending in `View`) produces a warning suggesting it should end in `View`
- [ ] A view named `OrderListView` (ending in `View`) produces no warning
- [ ] Each warning includes the file path and line number from the element's `NamePos`
- [ ] All five rules (three event rules from Task 2 plus two new rules) are checked in a single linter invocation

**Affected Files/Modules:**
- `internal/linter/` -- add command and view naming rules to the existing linter function
- `internal/linter/linter_test.go` -- add tests for command past-tense detection and view naming validation

**Patterns to Follow:**
- Follow the AST traversal pattern in `internal/validator/validator.go:18-35` for walking `Model` -> `Contexts` -> `Aggregates` -> `Slices` -> `Commands` and `Views`.
- Follow the diagnostic construction pattern established in Task 2 for creating `diagnostic.Entry` values.
- Follow the test pattern established in Task 2 for constructing in-memory AST models.

**Testable:** Yes

**Verification:** `go test -tags unit ./...` passes. Tests verify past-tense command detection, imperative-form commands pass, view name suffix checking, and position information.

**Depends on:** Task 2

---

### Task 4: Wire up the `emod lint` CLI command

**Behavior:** The `emod lint` command is registered in the CLI application. Running `emod lint <file>` reads the file, parses it, runs the linter, and prints any warnings to stderr. The command exits with code 0 when no warnings are found and code 1 when warnings are present. Parse errors are reported as errors and also cause a non-zero exit.

**Acceptance Criteria:**
- [ ] `emod lint clean.emod` (a file with no naming violations) exits with code 0 and produces no output
- [ ] `emod lint problematic.emod` (a file with naming violations) exits with code 1 and prints each warning to stderr with file path, line number, rule name, and explanation
- [ ] `emod lint` with no file argument produces an error message and exits with code 1
- [ ] `emod lint nonexistent.emod` produces an error message and exits with code 1
- [ ] `emod lint invalid.emod` (unparseable file) reports parse errors and exits with code 1
- [ ] The `lint` command appears in `emod --help` output

**Affected Files/Modules:**
- `internal/cli/app.go` -- register the `lint` command in the `Commands` slice
- `internal/cli/lint.go` (new) -- implement the `RunLint` function
- `internal/cli/lint_test.go` (new) -- unit tests for the lint command

**Patterns to Follow:**
- Follow the command registration pattern in `internal/cli/app.go:14-27` for adding the `lint` command alongside `validate` and `fmt`.
- Follow the command implementation pattern in `internal/cli/validate.go:16-44` for file reading, lexing, parsing, diagnostic collection, and error formatting.
- Follow the test pattern in `internal/cli/validate_test.go:94-125` (including the `writeTemp` helper) for testing with temporary `.emod` files and asserting on error/nil returns.

**Testable:** Yes

**Verification:** `go test -tags unit ./...` passes. Tests verify exit behavior for clean files, files with warnings, missing arguments, nonexistent files, and unparseable files.

**Depends on:** Task 3

## Summary

**Total tasks:** 4

**Ordering rationale:** Dependency-first, bottom-up layering:
1. Task 1 (diagnostic severity) is a leaf change to the shared diagnostic type, required before lint warnings can carry rule names
2. Task 2 (event naming rules) builds the core linter package with the three event-specific rules
3. Task 3 (command and view rules) extends the linter with two more rules, completing all five naming checks
4. Task 4 (CLI wiring) connects the complete linter to the `emod lint` command

Tasks 2 and 3 are split because they target different AST node types (events vs. commands/views) and represent distinct naming convention concerns. They could be merged into one task, but separating them keeps each task focused on a single category of lint rules.

**Acceptance criteria coverage:**

| Story Acceptance Criterion | Covered By |
|---|---|
| Events ending in `Updated`/`Changed`/`Modified` produce "state obsession" warning | Task 2 |
| Events matching `[Entity][Field]Changed` produce "property sourcing" warning | Task 2 |
| Events ending in `Initiated` produce command-in-disguise warning | Task 2 |
| Commands not in imperative form produce a warning | Task 3 |
| Views not ending in `View` produce a warning | Task 3 |
| Each warning includes file path, line number, rule name, and human-readable explanation | Task 1, Task 2, Task 3 |
| `emod lint` exits 0 with no warnings, 1 with warnings | Task 4 |

**Deferred:** None. All seven acceptance criteria from the story are covered.

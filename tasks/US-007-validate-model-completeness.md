# US-007: Validate Model Completeness

## Progress
- [x] Task 1: Add orphan command completeness check to validator
- [x] Task 2: Add orphan event completeness check to validator
- [ ] Task 3: Integrate lint rules into `emod validate`
- [ ] Task 4: Add `--format json` flag to `emod validate`

---

## Story Reference
File: user-stories/rate-limiting.md — as interpreted from inline description:

> **US-007: Validate model completeness** — As a model author, I want validation to catch incomplete or inconsistent models so that I know my model is structurally sound before generating diagrams.

**Acceptance Criteria:**
- [ ] A command that is not connected to any event produces an error
- [ ] An event that is not produced by any command or external source produces an error
- [ ] A view whose `subscribes` list references a non-existent event produces an error (ALREADY IMPLEMENTED — skip)
- [ ] An automation whose trigger references a non-existent event produces an error (ALREADY IMPLEMENTED — skip)
- [ ] `emod validate` runs all lint rules plus completeness checks
- [ ] `emod validate --format json` outputs structured diagnostics matching the lint JSON format

**Depends on:** US-003, US-006

---

## Codebase Context

### Affected Modules
- **`internal/validator/`** — `Validate()` currently checks cross-references (nonexistent contexts, commands, events referenced by automations, translations, views, flows). It does NOT check for orphan commands (defined but never referenced by a flow) or orphan events (defined but never produced). Diagnostics are `*diagnostic.Entry` without `RuleName` or `Severity` set currently.
- **`internal/linter/`** — `Lint()` checks naming conventions and structural anti-patterns. Produces `*diagnostic.Entry` with `Severity` and `RuleName` populated. Already wired into `RunLint` but NOT yet into `RunValidate`.
- **`internal/cli/validate.go`** — `RunValidate(path string) error` reads, lexes, parses, runs `validator.Validate()`, and formats diagnostics as text.
- **`internal/cli/lint.go`** — `RunLint(path, format string) error` has the pattern for `--format json` support: `jsonEntry` struct, `formatJSON()` function, `LintError` type with exit codes.
- **`internal/cli/app.go`** — Validate command currently has no flags. Lint command shows the pattern for a `--format` flag.
- **`internal/ast/ast.go`** — `Flow` has `CommandName` and `EventName` fields. `Event` has `Source` (external source indicator). `Translation` has `Event *Event` for inline event definitions.

### Key Patterns
- **Validator cross-reference checks** — `validator.go:15-35` builds name sets (`commandNames`, `eventNames`); `validator.go:39-102` iterates all slices to check membership and emit `*diagnostic.Entry`.
- **Validator unit tests** — `validator_test.go:13-872` uses `TestValidate` umbrella function, `t.Run` for each scenario, fresh `*ast.Model` fixtures in each subtest, `testify/require` assertions.
- **CLI JSON output** — `lint.go:24-66` defines `jsonEntry` struct and `formatJSON()` function; `lint_test.go:131-297` tests JSON output via `captureStdout`, JSON unmarshal, field verification.
- **CLI flag wiring** — `app.go:49-76` shows `--format` flag on the lint command with `urfave.StringFlag`.
- **Existing test scaffolding** — `writeTemp(t, name, content)` helper in `validate_test.go:287-294`.

### Connection Rules
- **Command to event connection**: A command is "connected" when its name appears in a `Flow.CommandName` within any slice. A command with zero flow references is an orphan.
- **Event production**: An event is "produced" if (a) its name appears in a `Flow.EventName`, or (b) `Event.Source != ""` (external source), or (c) it is defined inside a `Translation.Event`. An event with none of these is an orphan.

---

## Tasks

### Task 1: Add orphan command completeness check to validator

**Behavior:** The validator reports an error when a command is defined but has no `flow` that connects it to an event.

**Acceptance Criteria:**
- [ ] A command with zero flows referencing it produces an orphan-command error
- [ ] A command referenced by at least one flow produces no diagnostic
- [ ] The diagnostic includes the command's definition position (`Command.NamePos`)

**Affected Files/Modules:**
- `internal/validator/validator.go` — Add orphan command detection logic
- `internal/validator/validator_test.go` — Add test cases under `TestValidate`

**Patterns to Follow:**
- Follow the name-set-building pattern in `validator.go:15-35` to collect command names and flow command references
- Follow the diagnostic creation pattern in `validator.go:44-49` for position-aware entries
- Follow the test structure in `validator_test.go:13-872` for umbrella `TestValidate` with `t.Run` subtests

**Testable:** Yes

**Verification:** Unit tests pass with `go test -tags=unit ./internal/validator/`

**Depends on:** None

---

### Task 2: Add orphan event completeness check to validator

**Behavior:** The validator reports an error when an event is defined but is not produced by any flow, external source, or translation.

**Acceptance Criteria:**
- [ ] An event with no producer (no flow reference, no external source, not inside a translation) produces an orphan-event error
- [ ] An event referenced by at least one flow produces no diagnostic
- [ ] An event with `Event.Source != ""` (external source) produces no diagnostic
- [ ] An event defined inside a `Translation.Event` produces no diagnostic
- [ ] The diagnostic includes the event's definition position (`Event.NamePos`)

**Affected Files/Modules:**
- `internal/validator/validator.go` — Add orphan event detection logic
- `internal/validator/validator_test.go` — Add test cases under `TestValidate`

**Patterns to Follow:**
- Follow the name-set-building pattern in `validator.go:15-35` to collect event names and flow event references
- Follow the same file as Task 1 but this task is independent (can be done in any order)
- Follow the test structure in `validator_test.go:13-872` for umbrella `TestValidate` with `t.Run` subtests

**Testable:** Yes

**Verification:** Unit tests pass with `go test -tags=unit ./internal/validator/`

**Depends on:** None (independent of Task 1)

---

### Task 3: Integrate lint rules into `emod validate`

**Behavior:** Running `emod validate` runs both lint rules (from `linter.Lint()`) and completeness/cross-reference checks (from `validator.Validate()`), reporting all diagnostics together.

**Acceptance Criteria:**
- [ ] `emod validate` runs all lint rules plus completeness checks
- [ ] A model with lint violations gets reported by `validate` in addition to validation errors
- [ ] A model with only lint warnings (no errors) still exits with an error
- [ ] The existing `RunValidate` text output format is preserved

**Affected Files/Modules:**
- `internal/cli/validate.go` — Update `RunValidate` to also call `linter.Lint(model)` and merge its diagnostics
- `internal/cli/validate_test.go` — Add test cases verifying lint violations surface through `RunValidate`

**Patterns to Follow:**
- Follow the pipeline pattern in `lint.go:79-121` (lex → parse → lint → format)
- Follow the existing `validate_test.go:96-285` test pattern for `TestValidate` with CLI-level subtests

**Testable:** Yes

**Verification:** Tests pass with `go test -tags=unit ./internal/cli/`

**Depends on:** Task 1, Task 2

---

### Task 4: Add `--format json` flag to `emod validate`

**Behavior:** `emod validate --format json` outputs structured diagnostics in the same JSON format as `emod lint --format json`. Text format remains the default.

**Acceptance Criteria:**
- [ ] `emod validate --format json` outputs a JSON array of diagnostic entries
- [ ] Each JSON entry contains `file`, `line`, `rule`, `severity`, and `message` fields (matching the lint JSON format)
- [ ] A clean file with no diagnostics outputs `[]`
- [ ] `emod validate` without `--format` still outputs text and has unchanged behavior
- [ ] The existing `RunValidate` text-only tests continue to pass

**Affected Files/Modules:**
- `internal/cli/validate.go` — Add `--format json` handling (reuse or share `formatJSON` from lint.go)
- `internal/cli/validate_test.go` — Add JSON output tests following the lint JSON test pattern
- `internal/cli/app.go` — Add `--format` flag to the validate command definition

**Patterns to Follow:**
- Follow the JSON format implementation in `lint.go:24-66` and the `--format` flag wiring in `app.go:49-76`
- Follow the JSON testing pattern in `lint_test.go:131-297` (captureStdout, JSON unmarshal, field verification)

**Testable:** Yes

**Verification:** Tests pass with `go test -tags=unit ./internal/cli/`

**Depends on:** Task 3

---

## Summary

- **Total tasks:** 4
- **Task ordering rationale:** Dependency-first. Tasks 1 and 2 are independent orphan-detection checks that add completeness rules to the validator. Task 3 integrates lint rules into `emod validate` and depends on the completeness checks existing. Task 4 adds JSON output and depends on the full validate pipeline (with lint + completeness) being in place.
- **AC coverage:**
  - AC 1 (orphan command error) → Task 1
  - AC 2 (orphan event error) → Task 2
  - AC 3 (view subscribes non-existent event) → ALREADY IMPLEMENTED — skipped
  - AC 4 (automation trigger non-existent event) → ALREADY IMPLEMENTED — skipped
  - AC 5 (`emod validate` runs lint + completeness) → Task 1 + Task 2 + Task 3
  - AC 6 (`emod validate --format json` matches lint JSON format) → Task 4

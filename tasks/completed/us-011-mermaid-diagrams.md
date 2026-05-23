## Progress
- [x] Task 1: Write unit tests for ExportMermaid in mermaid_test.go
- [x] Task 2: Update CLI diagram tests for the new RunDiagram signature and mermaid format
- [x] Task 3: Update US-011 acceptance criteria in the story document
- [ ] Task 4: Run full test suite to verify everything passes

## Story Reference

**US-011** from `user-stories/emod-dsl-and-diagrams.md` — Generate Mermaid diagrams using native event modeling syntax.

The `ExportMermaid` function in `internal/diagram/mermaid.go` has already been implemented. The CLI wiring in `internal/cli/diagram.go` and `internal/cli/app.go` has also been updated. The remaining work is writing tests, updating story documentation, and final verification.

## Codebase Context

### Affected Packages
- **`internal/diagram/`** — Contains `ExportMermaid(model *ast.Model) ([]byte, error)` at `internal/diagram/mermaid.go:15`. The existing `collectSlices()` in `drawio.go:283` is reused. Tests live in `drawio_test.go` under `package diagram_test` with `//go:build unit` tag.
- **`internal/cli/`** — `RunDiagram(path, outputPath, format string)` at `diagram.go:24` dispatches to `ExportMermaid` when `format == "mermaid"`. `app.go:124-129` registers the `--format` flag (default `"drawio"`). Tests in `diagram_test.go` under `package cli_test` call `RunDiagram` with 2 args — these need updating to 3 args.

### Existing Patterns
- **Diagram export function signature** — `ExportDrawio(model *ast.Model) ([]byte, error)` in `drawio.go:38`. `ExportMermaid` follows the same pattern.
- **Test structure for diagram generators** — `drawio_test.go` uses `package diagram_test`, a `func TestExportDrawio(t *testing.T)` umbrella, `t.Run` for scenarios, `require.NoError`/`require.Contains` from testify, and builder helpers (`minimalModel`, `singleSliceModel`, `fullModel`, `command`, `event`, `view`, `eventWithSource`).
- **CLI test helpers** — `writeTemp(t, name, content)` in `validate_test.go:555` and `captureStderr(t, fn)` in `export_test.go:17` are shared across the `cli_test` package.
- **Test tag** — All unit tests use `//go:build unit` build tag. Run with `go test -tags unit ./...` or `task test:unit`.

### Relevant Types
- **`ast.Model`** — Top-level model with `Name`, `Contexts` (each containing `Aggregates` → `Slices`).
- **`ast.Slice`** — Contains `Trigger`, `Commands`, `Events`, `Views`, `Automations`, `Flows`, `Translations`.
- **`ast.Trigger`** — Has `Kind` field (`"UI"`, `"Schedule"`, `"Processor"`) and `Name`.
- **`ast.Automation`** — Has `Name`, `TriggerEvent`, `Command`, `TargetContext`.
- **`ast.View`** — Has `Name`, `Subscribes` (list of event names).

## Tasks

### Task 1: Write unit tests for ExportMermaid in mermaid_test.go

**Behavior:** Unit tests verify that `ExportMermaid` correctly generates Mermaid `eventmodeling` diagram markup from various AST model configurations.

**Acceptance Criteria:**
- [ ] Nil model returns empty bytes with no error
- [ ] Empty model (no contexts) returns output starting with `eventmodeling` and no timeframe entries
- [ ] Single slice with trigger (`UI` kind) renders as `tf NN ui TriggerName`
- [ ] Single slice with schedule trigger renders as `tf NN pcr TriggerName`
- [ ] Single slice with processor trigger renders as `tf NN pcr TriggerName`
- [ ] Commands render as `tf NN cmd CommandName`
- [ ] Events render as `tf NN evt EventName`
- [ ] Views render as `tf NN rmo ViewName`
- [ ] Automations render as `tf NN pcr AutomationName`
- [ ] Context names are prepended with dot notation (e.g., `Orders.CreateOrder`)
- [ ] Automation with `TargetContext` uses the target context as the namespace prefix
- [ ] Sequential numbering increments across all slices (no restart per slice)
- [ ] Model name appears as a comment (`% ModelName`)
- [ ] Slice names appear as comments (`% Slice: SliceName`)
- [ ] Complete model with all element types produces well-formed output with correct ordering of tf entries

**Affected Files/Modules:**
- `internal/diagram/mermaid_test.go` — New file; unit tests for `ExportMermaid`

**Patterns to Follow:**
- Follow the test structure in `internal/diagram/drawio_test.go:14-347` — same `package diagram_test`, same `//go:build unit` build tag, same testify assertions
- Reuse the builder helpers from `internal/diagram/drawio_test.go:384-451` (`minimalModel`, `singleSliceModel`, `fullModel`, `command`, `event`, `view`, `eventWithSource`)
- Follow the umbrella test function pattern: `func TestExportMermaid(t *testing.T)`
- Follow the `t.Run` naming conventions from `internal/diagram/drawio_test.go` (scenario descriptions as full sentences)

**Testable:** Yes — `ExportMermaid` is an exported function.

**Verification:** `go test -tags unit ./internal/diagram/...` passes.

**Depends on:** None

---

### Task 2: Update CLI diagram tests for the new RunDiagram signature and mermaid format

**Behavior:** CLI diagram tests are updated for the 3-argument `RunDiagram(path, outputPath, format string)` signature, and a new test verifies mermaid output is printed to stdout.

**Acceptance Criteria:**
- [ ] All existing `RunDiagram(path, "")` calls in `diagram_test.go` are updated to `RunDiagram(path, "", "drawio")`
- [ ] All existing `RunDiagram(path, outputPath)` calls are updated to `RunDiagram(path, outputPath, "drawio")`
- [ ] A new test verifies that mermaid format output is printed to stdout (no file created) when no `-o` path is given
- [ ] A new test verifies that mermaid output starts with `eventmodeling`
- [ ] A new test verifies that mermaid output can be written to a specific file with `-o`
- [ ] An unsupported format produces an error message listing supported formats
- [ ] All existing tests continue to pass (drawio output behavior unchanged)

**Affected Files/Modules:**
- `internal/cli/diagram_test.go` — Update existing `RunDiagram` calls; add mermaid-related test scenarios

**Patterns to Follow:**
- Follow the existing test structure in `internal/cli/diagram_test.go` — same `package cli_test`, same `//go:build unit` tag
- Use `writeTemp` from `internal/cli/validate_test.go:555` to create temp .emod files
- Use `captureStdout` from `internal/cli/lint_test.go:17` to capture mermaid output printed to stdout
- Use `captureStderr` from `internal/cli/export_test.go:17` to capture error messages
- Follow the pattern for custom output paths from `internal/cli/diagram_test.go:29-38` (using `filepath.Join(t.TempDir(), ...)`)

**Testable:** Yes — `RunDiagram` is an exported function.

**Verification:** `go test -tags unit ./internal/cli/...` passes.

**Depends on:** Task 1 (core `ExportMermaid` function should be tested first)

---

### Task 3: Update US-011 acceptance criteria in the story document

**Behavior:** The US-011 acceptance criteria in `user-stories/emod-dsl-and-diagrams.md` are updated to reflect the actual implementation using Mermaid's native `eventmodeling` diagram syntax instead of the originally planned flowchart syntax.

**Acceptance Criteria:**
- [ ] Acceptance criteria reflect that the diagram uses `eventmodeling` diagram type with timeframe definitions (`tf`)
- [ ] Acceptance criteria specify that triggers are rendered as `ui` type (or `pcr` for schedule/processor triggers)
- [ ] Acceptance criteria specify that commands are rendered as `cmd` type
- [ ] Acceptance criteria specify that events are rendered as `evt` type
- [ ] Acceptance criteria specify that views/read models are rendered as `rmo` type
- [ ] Acceptance criteria specify that automations are rendered as `pcr` type
- [ ] Acceptance criteria specify that contexts use dot notation (e.g., `Orders.CreateOrder`)
- [ ] Acceptance criteria specify that timeframes use unique sequential numbering across all slices
- [ ] Removed or replaced the old flowchart-based criteria (subgraphs, styled nodes for colors)
- [ ] The `--format mermaid` flag and output path (`-o`) behavior is documented
- [ ] Updated criteria mention Mermaid v11.15.0+ requirement

**Affected Files/Modules:**
- `user-stories/emod-dsl-and-diagrams.md` — Update US-011 (lines 163-174) acceptance criteria

**Patterns to Follow:**
- Match the existing formatting style for other user stories in the same file

**Testable:** No — documentation only.

**Verification:** Visual inspection confirms the criteria match the implemented behavior.

**Depends on:** None (but should reflect the final state after Tasks 1 and 2)

---

### Task 4: Run full test suite to verify everything passes

**Behavior:** Run the complete unit test suite to confirm all existing and new tests pass, and the codebase is green.

**Acceptance Criteria:**
- [ ] `go test -tags unit ./...` exits with code 0
- [ ] All tests in the `diagram` package pass (including new `mermaid_test.go`)
- [ ] All tests in the `cli` package pass (including updated diagram tests)
- [ ] No regressions in other packages

**Affected Files/Modules:**
- No file changes — this is a verification step

**Testable:** No — this is a verification task. All testable behavior has been verified in Tasks 1 and 2.

**Verification:** `go test -tags unit ./...` passes with exit code 0.

**Depends on:** Task 1, Task 2

## Summary

**Total tasks:** 4

**Ordering rationale:** Dependency-first. Task 1 (unit tests for the core `ExportMermaid` function) has no dependencies and is the foundation. Task 2 (CLI tests) depends on the core function being tested. Task 3 (story documentation) is independent but should reflect the final state. Task 4 (full test suite) is the final verification.

**Acceptance criteria coverage:**
- All US-011 acceptance criteria are covered by Tasks 1 and 2 (testing) and Task 3 (documentation)
- Task 1 covers: eventmodeling syntax, timeframe definitions, entity type mapping (ui/cmd/evt/rmo/pcr), dot notation context names, sequential numbering across slices, automation/schedule trigger mapping
- Task 2 covers: `-f mermaid` flag, stdout output, `-o` file output, default format backward compatibility
- Task 3 keeps the story document in sync with what was implemented
- Task 4 ensures no regressions

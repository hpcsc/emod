# US-010: Generate draw.io Diagrams

## Progress
- [x] Task 1: Implement draw.io XML rendering engine in `internal/diagram/`
- [x] Task 2: Wire the `diagram` CLI command with `-o` flag support

---

## Story Reference

User story US-010 from `user-stories/emod-dsl-and-diagrams.md` (lines 146–159).  
Depends on US-007 (Validate model completeness) — already completed.

---

## Codebase Context

### Existing CLI structure
- CLI commands registered in `internal/cli/app.go` (lines 11–148). Each command follows the pattern: define a `urfave.Command` with `Flags`, `ArgsUsage`, and an `Action` closure that delegates to a `RunXxx` function.
- Existing commands: `validate`, `fmt`, `lint`, `export`, `schema`.
- The `export` command (`internal/cli/export.go`) is the closest analog — it reads a file, lexes/parses/validates/lints, then calls an `internal/export/` function and writes output. This pattern should be followed for `diagram`.
- A `LintError` type (`internal/cli/lint.go:15-18`) is used across CLI handlers for structured error/exit-code reporting.

### AST model types
- Defined in `internal/ast/ast.go`. Key types: `Model`, `Context`, `Aggregate`, `Slice`, `Trigger`, `Command`, `Event`, `View`, `Automation`, `Translation`, `Flow`, `Field`.
- Each `Slice` has optional `Trigger`, lists of `Commands`, `Events`, `Flows`, `Views`, `Automations`, `Translations`.

### Existing export pattern (closest analog)
- `internal/export/export.go` traverses the AST with converter functions (`convertModel`, `convertSlice`, `convertCommand`, etc.) to produce a serializable representation.
- Tests in `internal/export/export_test.go` construct `ast.Model` values directly and verify output.
- Tests use `//go:build unit` build tag and `package export_test` (external test package).

### CLI test infrastructure
- Tests in `internal/cli/*_test.go` use `//go:build unit` build tag, `package cli_test`.
- Helpers: `writeTemp` (creates temp file from string content), `captureStdout` (captures stdout via pipe), `captureStderr` (captures stderr via pipe).
- Test fixtures: `validEmod` / `invalidEmod` constants in `internal/cli/validate_test.go`.
- Assertions use `github.com/stretchr/testify/require`.

### draw.io XML format
- draw.io uses the mxGraph XML format: an `<mxfile>` wrapper containing `<diagram>` with an `<mxGraphModel>` containing a `<root>` of `<mxCell>` elements.
- Cells can be vertices (elements) or edges (connections).
- Styling is done through a `style` attribute string (e.g., `rounded=1;fillColor=#FFA500;strokeColor=#FF8C00`).
- Colors for the diagram: orange (#FFA500 / #FF8C00) for events, blue (#4A90D9 / #357ABD) for commands, green (#5CB85C / #4CAF50) for views, white for triggers.
- A gear icon for automation reactors can be rendered using `shape=image;image=data:image/svg+xml,...` with a gear SVG.
- External systems use `dashed=1;strokeColor=#888888;fillColor=none`.

### Example model file
- `examples/all_patterns.emod` exercises all four slice patterns (command, view, automation, translation) across two bounded contexts.

### Test command
- `go test -tags unit ./...`

---

## Tasks

### Task 1: Implement draw.io XML rendering engine in `internal/diagram/`

**Behavior:** A new `internal/diagram/` package exports a function that takes a parsed and validated `*ast.Model` and returns valid draw.io (mxGraph) XML bytes. The XML produces a diagram with three horizontal swimlanes ("UI / Triggers" top, "Commands / Views" middle, "Events" bottom), colored boxes (orange events, blue commands, green views, white triggers), left-to-right slice ordering, and connections matching event modeling flows. Automation reactors render with a gear icon. External system translations render as gray dashed boxes.

**Acceptance Criteria:**
- [ ] `ExportDrawio(model *ast.Model) ([]byte, error)` produces valid mxGraph XML with `<mxfile>`, `<diagram>`, `<mxGraphModel>`, and `<root>` elements
- [ ] Three swimlane rows with labels "UI / Triggers", "Commands / Views", and "Events"
- [ ] Events are filled orange (#FFA500) with orange stroke (#FF8C00)
- [ ] Commands are filled blue (#4A90D9) with blue stroke (#357ABD)
- [ ] Views are filled green (#5CB85C) with green stroke (#4CAF50)
- [ ] Triggers are filled white with gray border
- [ ] Slices are laid out left-to-right in model order
- [ ] Connections: trigger to first command, command to event via flow declarations
- [ ] Event to view: subscriptions create edges from event nodes to view nodes
- [ ] Automation reactors render as a node with a gear icon style, connected: event -> automation -> command
- [ ] External system translations render as a gray dashed box surrounding the translation's command and event nodes
- [ ] A model with no slices produces an empty diagram (no error)

**Affected Files/Modules:**
- `internal/diagram/diagram.go` — New file; main `ExportDrawio` function, model-to-mxGraph conversion
- `internal/diagram/diagram_test.go` — New file; unit tests for diagram generation
- `internal/diagram/layout.go` — New file; positioning/layout algorithm for swimlanes and element placement

**Patterns to Follow:**
- AST traversal pattern in `internal/export/export.go:120-411` — convert functions that walk the model tree
- Test pattern in `internal/export/export_test.go:17-33` — construct AST models directly and assert on output bytes
- Test organization: use `//go:build unit` build tag and external test package (`package diagram_test`)

**Testable:** Yes — `ExportDrawio` is an exported function tested via package-external tests that construct AST models and verify XML output for content, structure, colors, connections, and special element rendering.

**Verification:** `go test -tags unit ./internal/diagram/...` passes; XML output renders correctly when opened in draw.io

**Depends on:** None (no code outside the `diagram` package is needed; the AST types from `internal/ast` are already stable)

---

### Task 2: Wire the `diagram` CLI command with `-o` flag support

**Behavior:** `emod diagram <file>` reads an `.emod` file, validates it, generates draw.io XML from the model, and writes the result to a `.drawio` file in the same directory as the input. The output path can be overridden with `-o path/to/output.drawio`. If validation fails, diagnostics are written to stderr and a non-zero exit code is returned.

**Acceptance Criteria:**
- [ ] `emod diagram reservation.emod` writes `reservation.drawio` to the same directory as the input file
- [ ] The `.drawio` file contains valid mxGraph XML matching the diagram output from `ExportDrawio`
- [ ] `emod diagram reservation.emod -o /custom/path/output.drawio` writes to the specified path
- [ ] If the `.emod` file has parse/validation errors, diagnostics are written to stderr and exit code is non-zero (no `.drawio` file is written)
- [ ] Missing file argument returns an error message
- [ ] Nonexistent file returns an error message
- [ ] A valid model with no diagnositics produces a clean exit (code 0) and a well-formed `.drawio` file

**Affected Files/Modules:**
- `internal/cli/app.go` — Register the new `diagram` command with `ArgsUsage: "<file>"` and `-o` flag
- `internal/cli/diagram.go` — New file; `RunDiagram(path string, outputPath string) error` function implementing the handler
- `internal/cli/diagram_test.go` — New file; tests for the CLI handler

**Patterns to Follow:**
- CLI command registration in `internal/cli/app.go:92-119` (the `export` command) — flags, action closure, error handling with `LintError`
- CLI handler pattern in `internal/cli/export.go:19-88` — file reading, lexing, parsing, validation, linter, then core logic call, output writing
- Test pattern in `internal/cli/export_test.go:37-49` — use `writeTemp` to create `.emod` file, call `RunXxx`, verify output file existence and content
- Use `captureStdout`/`captureStderr` from `internal/cli/lint_test.go:17-34` and `internal/cli/export_test.go:17-34` for stderr assertions
- Use `validEmod` / `invalidEmod` constants from `internal/cli/validate_test.go:16-102` for test fixtures

**Testable:** Yes — tests create temp `.emod` files, call `RunDiagram`, and verify that:
- The correct `.drawio` file was created with expected content
- Output path respects `-o` flag
- Invalid input produces stderr diagnostics and non-zero exit
- Error cases (missing file, nonexistent file) produce appropriate error messages

**Verification:** `go test -tags unit ./internal/cli/...` passes; `go build ./cmd/emod` succeeds; manual smoke test with `emod diagram examples/all_patterns.emod` produces a valid `.drawio` file

**Depends on:** Task 1 (the `diagram` package must exist with `ExportDrawio` before the CLI handler can call it)

---

## Summary

### Task Count: 2

### Ordering Rationale
- **Dependency-first:** Task 1 (rendering engine) is an independent concern with no dependencies; Task 2 (CLI wiring) depends on Task 1. This ordering allows each task to be independently developed, tested, and committed.
- **Risk-first:** The rendering engine is the riskier component (mxGraph XML format correctness, layout algorithm, special elements like gear icons and dashed boxes). Getting it right first with thorough unit tests reduces the risk when wiring the CLI.

### Acceptance Criteria Coverage

| Acceptance Criterion | Covered In |
|---|---|
| `emod diagram reservation.emod` writes `.drawio` to same directory | Task 2 |
| Swimlanes: UI/Triggers, Commands/Views, Events | Task 1 |
| Event colors (orange), Command colors (blue), View colors (green), Trigger colors (white) | Task 1 |
| Slices left-to-right in model order | Task 1 |
| Connections: trigger->command->event, event->view, event->reactor->command | Task 1 |
| Automation reactors with gear icon | Task 1 |
| External systems as gray dashed boxes | Task 1 |
| Output path override with `-o` | Task 2 |

No acceptance criteria are deferred — all are covered by the two tasks.

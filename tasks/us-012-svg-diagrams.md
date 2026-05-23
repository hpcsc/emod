## Progress
- [x] Task 1: Implement SVG rendering engine in `internal/diagram/svg.go`
- [ ] Task 2: Wire SVG format into CLI and add CLI tests

## Story Reference

**US-012** from `user-stories/emod-dsl-and-diagrams.md` — Generate SVG diagrams.

Depends on US-010 (draw.io diagrams — already completed).

---

## Codebase Context

### Existing layout and rendering infrastructure
All layout constants, color constants, and helper functions live in `internal/diagram/drawio.go` at package level in `package diagram`:
- Layout constants: `marginX`, `marginY`, `sliceWidth`, `boxWidth`, `boxHeight`, `sliceGap`, `laneHeight`, `laneGap` (lines 12–21)
- Color constants: `fillEvent`/`strokeEvent`, `fillCommand`/`strokeCommand`, `fillView`/`strokeView`, `fillTrigger`/`strokeTrigger`, `fillExternal`/`strokeExternal` (lines 24–35)
- `collectSlices(model *ast.Model) []sliceEntry` (lines 283–293) — flatten all slices from the AST
- `itemLayout(usableW, numItems, index, sliceX int) (int, int)` (lines 296–306) — computes width and x-position for elements within a slice column
- `sliceEntry` struct (lines 277–280)

Any new file in `package diagram` (e.g., `svg.go`) can directly reuse all of the above.

### Rendering logic needed for SVG
The draw.io rendering (lines 37–275) performs these steps that SVG must replicate:
1. Three swimlane rows with labels at fixed y-positions (`triggerLaneY`, `cmdViewLaneY`, `eventLaneY`)
2. Per-slice columns positioned horizontally via `marginX + i*(sliceWidth+sliceGap)`
3. Element boxes positioned within each lane using `itemLayout`
4. Styled connection edges: trigger→command, command→event (via `Flows`), event→view (via `Subscribes`), event→automation→command, command→external→event

The SVG file must recalculate all positions (there is no shared position-computation layer) but can use the same constants and helpers.

### CLI wiring pattern
- `internal/cli/diagram.go` — `RunDiagram(path, outputPath, format string) error` dispatches to diagram functions. Formats validated at line 73; dispatch via `switch` at lines 82–87; default output path uses `defaultDrawioPath()` at lines 100–102.
- `internal/cli/app.go:121-152` — `diagram` command with `--format` flag (current usage: `"Output format (drawio|mermaid)"`) and `-o` flag.
- The `defaultDrawioPath` function (lines 131–138) hardcodes `.drawio` extension. SVG will need a default `.svg` extension.

### CLI test infrastructure
- `writeTemp(t, name, content)` in `internal/cli/validate_test.go:555`
- `captureStdout(t, fn)` in `internal/cli/lint_test.go:17`
- `captureStderr(t, fn)` in `internal/cli/export_test.go:17`
- Test constants `validEmod`/`invalidEmod` in `internal/cli/validate_test.go`
- All CLI unit tests use `//go:build unit` tag and `package cli_test`

---

## Tasks

### Task 1: Implement SVG rendering engine in `internal/diagram/svg.go`

**Behavior:** A new `ExportSVG` function in the `diagram` package takes a parsed and validated `*ast.Model` and returns self-contained SVG bytes. The SVG produces a diagram with three horizontal swimlanes ("UI / Triggers", "Commands / Views", "Events"), colored boxes matching the draw.io output color scheme, left-to-right slice ordering, and styled arrow connections that match the event modeling flow (trigger→command→event, event→view, event→automation→command, command→external→event). The SVG is fully self-contained (no external CSS, fonts, or images).

**Acceptance Criteria:**
- [ ] `ExportSVG(model *ast.Model) ([]byte, error)` produces valid SVG XML with `<svg>` root element containing `xmlns="http://www.w3.org/2000/svg"`
- [ ] Nil model returns empty bytes with no error
- [ ] Empty model (no slices) returns a valid SVG with no diagram content
- [ ] Three swimlane rectangles with labels "UI / Triggers", "Commands / Views", and "Events"
- [ ] Events are filled orange (#f8cecc) with stroke #b85450
- [ ] Commands are filled blue (#dae8fc) with stroke #6c8ebf
- [ ] Views are filled green (#d5e8d4) with stroke #82b366
- [ ] Triggers are filled white (#ffffff) with stroke #000000
- [ ] Slices are laid out left-to-right in model order
- [ ] Arrows connect trigger→command (when both exist in a slice)
- [ ] Arrows connect command→event via flow declarations
- [ ] Arrows connect event→view via subscription declarations
- [ ] Arrows connect event→automation→command for automation reactors
- [ ] Arrows connect command→external system→event for translations
- [ ] The SVG is self-contained: no external references, renders in any modern browser
- [ ] Swimlane labels are present as SVG text elements

**Affected Files/Modules:**
- `internal/diagram/svg.go` — New file; `ExportSVG` function that renders model to SVG XML
- `internal/diagram/svg_test.go` — New file; unit tests for SVG generation

**Patterns to Follow:**
- Use the same `//go:build unit` build tag and `package diagram_test` external test package as `internal/diagram/drawio_test.go:1-3`
- Reuse builder helpers from `internal/diagram/drawio_test.go:384-451` (`minimalModel`, `singleSliceModel`, `fullModel`, `command`, `event`, `view`, `eventWithSource`)
- Follow the umbrella test function structure in `internal/diagram/drawio_test.go:14` (`func TestExportDrawio(t *testing.T)`)
- Reuse package-level constants and helpers from `internal/diagram/drawio.go` directly (colors at lines 24–35, `collectSlices` at line 283, `itemLayout` at line 296)
- Assert SVG output with `require.Contains` on expected SVG elements and `require.True(t, validSVG(output))` for well-formedness
- Connection rendering should follow the same iteration logic as `internal/diagram/drawio.go:193-269` (iterate entries, build connections by type)

**Testable:** Yes — `ExportSVG` is an exported function; tests construct AST models and verify SVG output for structure, colors, labels, arrows, and self-contained-ness.

**Verification:** `go test -tags unit ./internal/diagram/...` passes.

**Depends on:** None (reuses existing package-level constants and helpers from `package diagram`)

---

### Task 2: Wire SVG format into CLI and add CLI tests

**Behavior:** `emod diagram reservation.emod -f svg` reads an `.emod` file, validates it, generates SVG from the model, and writes the result to a `.svg` file. The output path can be overridden with `-o path/to/output.svg`. Validation errors produce diagnostics on stderr with no output file. The `--format` flag usage text lists `svg` alongside `drawio` and `mermaid`.

**Acceptance Criteria:**
- [ ] `emod diagram test.emod -f svg` writes `test.svg` to the same directory as the input file
- [ ] `emod diagram test.emod -f svg -o /custom/path/output.svg` writes to the specified path
- [ ] Custom `-o` path with nested directories creates the directory structure
- [ ] Validation errors produce diagnostics on stderr and exit code 2 (no `.svg` file is written)
- [ ] Lint warnings still produce the `.svg` file but with exit code 1
- [ ] Unsupported format still returns an error message listing `drawio`, `mermaid`, and `svg`
- [ ] Missing file argument returns error
- [ ] Nonexistent file returns error
- [ ] `--format` flag usage text lists all three formats

**Affected Files/Modules:**
- `internal/cli/diagram.go` — Add `"svg"` to format validation (line 73); add `case "svg":` in the dispatch switch (line 82); add default output path logic for SVG format (lines 100–102)
- `internal/cli/app.go` — Update `--format` flag usage string (line 127) to include `svg`
- `internal/cli/diagram_test.go` — Add tests for SVG format output; add `svg` to unsupported-format error test

**Patterns to Follow:**
- Follow the draw.io format handler pattern in `internal/cli/diagram.go:86` (`output, err = diagram.ExportDrawio(model)`) — SVG will use the same pattern with `diagram.ExportSVG`
- Follow the existing CLI test structure in `internal/cli/diagram_test.go` — same `package cli_test`, same `//go:build unit` tag
- Use `writeTemp` from `internal/cli/validate_test.go:555` for temp `.emod` files
- Use `filepath.Join(t.TempDir(), ...)` for custom output paths (pattern from `internal/cli/diagram_test.go:29-38`)
- Follow the `validEmod` test fixture pattern from `internal/cli/validate_test.go` for SVG tests
- Validate SVG output by reading the file and checking for `<svg` prefix (pattern from the draw.io XML validation test at `internal/cli/diagram_test.go:118-133`, which reads the file and checks `strings.HasPrefix`)

**Testable:** Yes — `RunDiagram` is an exported function; tests create temp `.emod` files, call `RunDiagram` with `format="svg"`, and verify output file existence, content prefix, and error conditions.

**Verification:** `go test -tags unit ./internal/cli/...` passes.

**Depends on:** Task 1 (the `diagram.ExportSVG` function must exist before the CLI handler can call it)

---

## Summary

**Total tasks:** 2

**Ordering rationale:** Dependency-first. Task 1 (SVG rendering engine) has no code dependencies — it reuses existing package-level constants but introduces no changes to existing files. Task 2 (CLI wiring) depends on `ExportSVG` existing and modifies `diagram.go` and `app.go`. This ordering allows each task to be independently developed, tested, and committed with a green codebase.

**Acceptance criteria coverage:**

| Acceptance Criterion | Covered In |
|---|---|
| `emod diagram reservation.emod -f svg` writes an SVG file | Task 2 |
| Same color scheme as draw.io (orange events, blue commands, green views, white triggers) | Task 1 |
| Swimlane labels present in the SVG | Task 1 |
| Connections between elements rendered as styled arrows | Task 1 |
| SVG is self-contained (no external dependencies), renders in any browser | Task 1 |
| Output path override with `-o path/to/output.svg` | Task 2 |

All six acceptance criteria from US-012 are covered. None are deferred.

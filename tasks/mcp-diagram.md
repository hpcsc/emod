# US-007: Draw a diagram from an agent

## Contents
1. Task 1: Pick the renderer of a diagram format in `internal/diagram`
2. Task 2: Draw a diagram with `emod_diagram`
3. Task 3: Document `emod_diagram`

---

## Story Reference

`user-stories/mcp-server.md` → **US-007: Draw a diagram from an agent**. The design is in
`docs/proposals/ai/01-mcp-server.md`, section "Tool Surface", with the note "Formats of `emod_diagram`". The
proposal sets the direction, and this story sets the scope.

The story's acceptance criteria, numbered for the Summary:

- **AC1.** `emod_diagram` takes `path` or `content` and a `format`: `svg`, `mermaid`, `ascii`, `event-flow`,
  `event-flow-mermaid` or `event-flow-mermaid-no-icons`. The default is `svg`, as in the CLI.
- **AC2.** For the same source and options, the result is the same text that `emod diagram` gives for that
  format.
- **AC3.** `style` accepts `auto`, `projected` and `dcb`, with `auto` as the default, as
  `emod diagram --style` does.
- **AC4.** `specs: true` draws the Given-When-Then cards in `svg`. With another format, the tool returns an
  error result that says only `svg` draws spec cards, as the CLI does.
- **AC5.** A source with an error returns the diagnostics and no diagram. A source with only warnings returns
  the diagram and the warnings.
- **AC6.** The tool writes no file and starts no server.

## Boundaries

**Out of scope:**

- `emod diagram --serve` and any viewer server.
- A change to a renderer or to the output of `emod diagram`.
- Added by this breakdown:
  - **The list of formats and the choice of the renderer move into `internal/diagram`**, next to the
    renderers, so that the CLI and the tool share them. Only the file output and the stdout rule stay in the
    CLI.
  - **The result.** A source with an error gives `{"diagnostics":[...]}` with the records of `emod_check`,
    and no diagram, as a normal result. Any other source gives `{"diagram":"...","diagnostics":[...]}`, with
    the list empty for a clean source. The first text block holds the diagram text, so that a host shows
    `mermaid` and `ascii` in the chat. A second text block holds the warnings, one line each, as the CLI
    prints them, when the source has a warning.
  - **The input schema** gives `format` and `style` as enums with their defaults, and `specs` as a boolean.
    The SDK refuses an unknown format or style with an error result that lists the values.
  - **The order of the checks is the order of the CLI**: an error in the source first, then `specs` with a
    format other than `svg`.

## Codebase Context

- **The CLI.** `RunDiagram` (`internal/cli/diagram.go`) runs `oracle.Run`, prints each diagnostic to stderr,
  and returns exit code 2 with no diagram when one is an error. Then it checks the format against
  `diagramFormats`, refuses `--specs` for a format not in `specCardFormats` through `unsupportedSpecsSurface`
  ("--specs is not supported for format %q; only svg draws spec cards"), and picks the renderer in a switch
  over the six formats. `mermaid`, `ascii` and the two `event-flow-mermaid` formats go to stdout with
  `fmt.Println`; `svg` and `event-flow` go to a file, `-o` or a path beside the model.
- **The renderers.** `internal/diagram` has `ExportSVG(model, style, options...)`, `ExportMermaid`,
  `ExportASCII`, `ExportEventFlow`, `ExportEventFlowMermaid` and `ExportEventFlowMermaidNoIcons`,
  `ParseStyle`, `Style` and `WithSpecs`. The learning "`ExportSVG` and `ExportASCII` ignore the `Style` they
  are handed" means that a test of the style uses a format that reads it.
- **The lint.** The learning "A change to internal/ast/traverse.go is refused by type-methods-split" applies to
  any file with methods of a type declared elsewhere, so the new code goes in a new file of free functions.
- **The server.** `internal/mcp/server.go` has `formatSchema`, `formatInput`, `checkResult`,
  `structuredResult` and `oneOf`. `cmd/emod/main_test.go` has `modelPaths`, `callTool` and `textOf`.

## Tasks

### Task 1: Pick the renderer of a diagram format in `internal/diagram`

**Behavior:** `diagram.Formats()` gives the six formats in the order of the CLI, `diagram.DrawsSpecCards`
says whether a format draws spec cards, and `diagram.Export(model, format, style, options...)` gives the
output of the renderer of the format, or an error that lists the formats for an unknown one. `RunDiagram`
uses them and keeps the file output, the stdout rule and its messages.

**Acceptance Criteria:**
- [ ] `diagram.Formats()` gives `svg`, `mermaid`, `ascii`, `event-flow`, `event-flow-mermaid` and
  `event-flow-mermaid-no-icons`.
- [ ] `diagram.Export` gives, for each format, the output of its renderer, with the style and the options
  that the call gives.
- [ ] `diagram.Export` gives an error that names the unknown format and each format.
- [ ] `diagram.DrawsSpecCards` is true for `svg` only.
- [ ] `internal/cli` has no `diagramFormats`, `specCardFormats` or switch over the renderers, and each
  existing leaf of `internal/cli/diagram_test.go` passes with no change to an existing leaf.

**Patterns to follow:**
- `internal/cli/diagram.go`
- `internal/diagram/option.go`

**Affected files:**
- `internal/diagram/format.go` (new)
- `internal/diagram/format_test.go` (new)
- `internal/cli/diagram.go`

**Certainty:** medium. The move is small, but the learning on moved code says that its branches owe tests.
**Blast radius:** low. The output of `emod diagram` does not change.

**Depends on:** none

### Task 2: Draw a diagram with `emod_diagram`

**Behavior:** The server registers `emod_diagram` with the input of `emod_check`, a `format` enum with the
default `svg`, a `style` enum with the default `auto`, and a boolean `specs`. It gives the result that the
boundaries above name.

**Acceptance Criteria:**
- [ ] `tools/list` lists `emod_diagram`, with the two enums, their defaults and `specs`, and an output schema
  of one of the diagram result and the diagnostics.
- [ ] A call with no format gives the SVG of the model.
- [ ] A source with an error gives `{"diagnostics":[...]}` and no diagram.
- [ ] A source with only a warning gives the diagram and the warning, with the warning in a second text block.
- [ ] `specs: true` with `svg` draws the cards, and with `mermaid` gives an error result that says only svg
  draws spec cards.
- [ ] `style` reaches the renderer: the `mermaid` of a dcb model differs between `projected` and `dcb`.
- [ ] A call with a path writes no file under the root, also for `svg` and `event-flow`.
- [ ] Through the built binary, for each file in `examples/` and `internal/parser/testdata/` and each format,
  the tool gives the text that `emod diagram` gives, or the records of `emod validate --format json` for a
  source with an error.

**Patterns to follow:**
- `internal/mcp/server.go` (`exportTool`, `formatSchema`, `oneOf`)
- `internal/mcp/server_test.go`
- `cmd/emod/main_test.go` (the export parity group)
- task:1

**Affected files:**
- `internal/mcp/server.go`
- `internal/mcp/server_test.go`
- `cmd/emod/main_test.go`

**Certainty:** high. `emod_export` is the precedent. **Blast radius:** low. The tool is read-only.

**Depends on:** 1

### Task 3: Document `emod_diagram`

**Behavior:** The MCP page of the docs describes the tool, and the README names it.

**Acceptance Criteria:**
- [ ] `site/content/docs/mcp.md` has a section on `emod_diagram`: its formats, `style`, `specs`, the result
  for errors and warnings, and that the host saves an SVG with its own file tool.
- [ ] The README section "Check models from an agent (MCP)" names the tool.

**Patterns to follow:**
- `site/content/docs/mcp.md`

**Affected files:**
- `site/content/docs/mcp.md`
- `README.md`

**Certainty:** high. **Blast radius:** low.

**Depends on:** 2

## Summary

| AC | Task |
| --- | --- |
| AC1 | 1, 2 |
| AC2 | 1, 2 |
| AC3 | 2 |
| AC4 | 1, 2 |
| AC5 | 2 |
| AC6 | 2 |

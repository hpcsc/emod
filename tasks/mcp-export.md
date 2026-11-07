# US-006: Export a model and its glossary

## Contents
1. Task 1: Export a model with `emod_export`
2. Task 2: Give the glossary with `emod_glossary`
3. Task 3: Document `emod_export` and `emod_glossary`

---

## Story Reference

`user-stories/mcp-server.md` → **US-006: Export a model and its glossary**. The design is in
`docs/proposals/ai/01-mcp-server.md`, section "Tool Surface". The proposal sets the direction, and this story
sets the scope.

The story's acceptance criteria, numbered for the Summary:

- **AC1.** `emod_export` takes `path` or `content` and a `format`: `json`, `diagram-json` or `cue`. The
  default is `json`, as in the CLI.
- **AC2.** For `json` and `diagram-json`, the result is the same document as `emod export --format <format>`,
  with the diagnostics in it.
- **AC3.** For `cue`, a clean source returns the same text as `emod export --format cue`. A source with any
  diagnostic, also a warning, returns the diagnostics and no CUE, as the CLI does.
- **AC4.** `emod_glossary` takes `path` or `content` and a `format`: `markdown` or `json`, with `markdown` as
  the default. The result is the same text as `emod glossary --format <format>`. A source that does not
  parse returns the parse diagnostics and no glossary, as the CLI does.
- **AC5.** An unknown format returns an error result that lists the formats of that tool.
- **AC6.** Neither tool writes a file.

## Boundaries

**Out of scope:**

- `emod_diagram` (US-007), and any change to the export or the glossary itself.
- An output schema for the export document. The document has a schema of its own, `emod://schema`.
- Added by this breakdown:
  - **`format` is an enum in the input schema**, with its default. The SDK then refuses an unknown format
    with an error result that lists the values of the enum, and a host can show the choices.
  - **The result for each format:**
    - `json` and `diagram-json`: the structured content is the export document, and the text block holds
      the same JSON.
    - `cue` for a clean source, and `markdown`: one text block with the text, and no structured content.
    - `cue` for a source with any diagnostic: `{"diagnostics":[...]}` with the records of `emod_check`, as
      a normal result.
    - `emod_glossary` with `json`: the structured content is the glossary document, and the text block
      holds the same JSON.
    - `emod_glossary` for a source that does not parse: `{"diagnostics":[...]}` with the parse diagnostics,
      as `emod_slices_list` gives them.
  - **Both tools take the input of `emod_check`** (`content`, `path`, `name`) and add `format`.

## Codebase Context

- **The CLI.** `RunExport` (`internal/cli/export.go`) runs `oracle.Run` and calls
  `export.ExportJSONDiagnostics`, `export.ExportDiagramJSONDiagnostics` or `export.ExportCUE`. For `json` and
  `diagram-json` it prints the document even when the source has errors, and the exit code tells the
  severity. For `cue` it prints the diagnostics to stderr and no CUE when the source has any diagnostic, and
  it prints the CUE with `fmt.Println`. `RunGlossary` (`internal/cli/glossary.go`) parses with
  `parseModelFile`, so a parse error stops it, and prints `glossary.RenderMarkdown` or `glossary.RenderJSON`
  with `fmt.Print`.
- **The server.** `internal/mcp/server.go` has `sourceInput`, `sourceSchema`, `Server.source`,
  `checkResult`, `parseDiagnostics`, `structuredResult` and `jsonResult`. `emod_slices_list` is the
  precedent for a tool with `Out any` that stops on a parse error. The learning "An MCP tool over the go-sdk
  typed AddTool is routine, not low certainty" says how the SDK writes the text block and the output schema.
- **Parse diagnostics.** The learning "oracle.Check lints the model that the parser recovers, so parse
  diagnostics are not what emod validate prints" applies to the glossary: its parity test compares with the
  stderr of `emod glossary`, and a fixture for a parse error has a validator finding in its recovered part.
- **The tests.** `internal/mcp/server_test.go` drives the server through an SDK client. `cmd/emod/main_test.go`
  has the parity leaves of `emod_check` and `emod_slices_list` over each file in `examples/` and
  `internal/parser/testdata/`.

## Tasks

### Task 1: Export a model with `emod_export`

**Behavior:** The server registers `emod_export` with the input of `emod_check` and a `format` enum of
`json`, `diagram-json` and `cue`, with the default `json`. Each format gives the result that the boundaries
above name.

**Acceptance Criteria:**
- [x] `tools/list` lists `emod_export`, and its input schema has `content`, `path`, `name` and `format`, with
  the enum `json`, `diagram-json`, `cue` and the default `json`.
- [x] A call with no `format` gives the `json` document.
- [x] `json` and `diagram-json` give the export document with its diagnostics, also for a source with errors.
- [x] `cue` gives the CUE text for a clean source, and `{"diagnostics":[...]}` and no CUE for a source with
  only a warning.
- [x] An unknown format gives an error result that names `json`, `diagram-json` and `cue`.
- [x] A call with a `path` writes no file under the root.
- [x] Through the built binary, for each file in `examples/` and `internal/parser/testdata/`, each format
  gives what `emod export --format <format>` prints for that file, or for `cue` with diagnostics, the
  records of `emod validate --format json`.

**Patterns to follow:**
- `internal/mcp/server.go` (`slicesListTool`, `structuredResult`)
- `internal/mcp/server_test.go`
- `cmd/emod/main_test.go` (the parity leaves)
- `internal/cli/export.go`

**Affected files:**
- `internal/mcp/server.go`
- `internal/mcp/server_test.go`
- `cmd/emod/main_test.go`

**Certainty:** high. `emod_slices_list` is the precedent, and the export functions need no change. **Blast
radius:** low. The tool is read-only.

**Depends on:** none

### Task 2: Give the glossary with `emod_glossary`

**Behavior:** The server registers `emod_glossary` with the input of `emod_check` and a `format` enum of
`markdown` and `json`, with the default `markdown`. A source that does not parse gives its parse
diagnostics.

**Acceptance Criteria:**
- [x] `tools/list` lists `emod_glossary`, and its input schema has `content`, `path`, `name` and `format`,
  with the enum `markdown`, `json` and the default `markdown`.
- [x] A call with no `format` gives the Markdown glossary as one text block.
- [x] `json` gives the glossary document as the structured content and the text block.
- [x] A source that does not parse gives only its parse diagnostics, not the findings of the validator on
  what the parser recovered, and no glossary.
- [x] An unknown format gives an error result that names `markdown` and `json`.
- [x] A call with a `path` writes no file under the root.
- [x] Through the built binary, for each file in `examples/` and `internal/parser/testdata/`, each format
  gives what `emod glossary --format <format>` prints, or the parse diagnostics that it prints.

**Patterns to follow:**
- `internal/mcp/server.go` (`slicesListTool`, `parseDiagnostics`)
- `internal/mcp/server_test.go`
- `cmd/emod/main_test.go`
- `internal/cli/glossary.go`
- task:1

**Affected files:**
- `internal/mcp/server.go`
- `internal/mcp/server_test.go`
- `cmd/emod/main_test.go`

**Certainty:** high. **Blast radius:** low.

**Depends on:** 1

### Task 3: Document `emod_export` and `emod_glossary`

**Behavior:** The MCP page of the docs describes both tools, and the README names them.

**Acceptance Criteria:**
- [x] `site/content/docs/mcp.md` has a section on each tool: its formats, its default, and what each format
  gives, with the rule for diagnostics.
- [x] The README section "Check models from an agent (MCP)" names both tools.

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
| AC1 | 1 |
| AC2 | 1 |
| AC3 | 1 |
| AC4 | 2 |
| AC5 | 1, 2 |
| AC6 | 1, 2 |

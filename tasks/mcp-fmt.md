# US-008: Format a model from an agent

## Contents
1. Task 1: Format a model with `emod_fmt`
2. Task 2: Document `emod_fmt`

---

## Story Reference

`user-stories/mcp-server.md` → **US-008: Format a model from an agent**. The design is in
`docs/proposals/ai/01-mcp-server.md`, section "Tool Surface", with the note "A parse error stops some tools".
The proposal sets the direction, and this story sets the scope.

The story's acceptance criteria, numbered for the Summary:

- **AC1.** `emod_fmt` takes `path` or `content` and returns `formatted` and `changed`.
- **AC2.** `formatted` is the same text that `emod fmt` writes for that source.
- **AC3.** `changed` is false when the source is already in the canonical format, and true when it is not. It
  gives the same answer as the exit code of `emod fmt --check`.
- **AC4.** A source that does not parse returns the parse diagnostics and no text.
- **AC5.** The tool does not write to the file.

## Boundaries

**Out of scope:**

- `write: true` and `--allow-write` (US-010). The tool has no `write` input.
- `emod_slices_arrange` (US-009).
- A change to the formatter or to the output of `emod fmt`.
- Added by this breakdown:
  - **The result.** A source that parses gives `{"formatted":"...","changed":true|false}`. A source that
    does not parse gives `{"diagnostics":[...]}` with the parse diagnostics as the records of `emod_check`,
    and no `formatted` key. Both are normal results, as for `emod_slices_list`. The text block holds the
    same JSON.
  - **The tool takes the same input as `emod_check`**: `content`, `path` and `name`, with the same rules.
  - **Only parse diagnostics stop the tool**, as in the CLI. A source with validation errors or warnings
    still gives its formatted text.

## Codebase Context

- **The CLI.** `RunFmt` (`internal/cli/fmt.go`) reads the file, runs `oracle.Parse`, and returns the parse
  diagnostics as one error, one line each. Else it runs `formatter.Format(model)`. With `--check` it
  returns `"<path> is not formatted"` when the text differs from the source. Else it writes the file only
  when the text differs. `internal/cli/app.go` prints the error and exits with code 1 for both a parse
  error and a file that is not formatted.
- **The learnings.** "`emod fmt` canonicalises order, so a fmt golden is never the input re-indented",
  "Formatter output always begins with `emod N`", and "`emod fmt <file>` writes in place, so a receipt run
  dirties the working tree". A parity test therefore runs `emod fmt` on a copy of each file.
- **The server.** `internal/mcp/server.go` has `sourceInput`, `sourceSchema`, `Server.source`,
  `parseDiagnostics`, `structuredResult` and `oneOf`. `emod_slices_list` is the precedent for a tool that
  stops on a parse error. `cmd/emod/main_test.go` has `modelPaths`, `callTool`, `diagnosticLines` and the
  parity groups of the other tools.

## Tasks

### Task 1: Format a model with `emod_fmt`

**Behavior:** The server registers `emod_fmt` with the input of `emod_check` and an output schema of one of
the formatted result and the parse diagnostics. A source that parses gives `formatted`, the text of
`formatter.Format`, and `changed`, which is true when that text differs from the source. A source that does
not parse gives only its parse diagnostics. The tool writes no file.

**Acceptance Criteria:**
- [ ] `tools/list` lists `emod_fmt`, with the input of `emod_check` and an output schema of one of the
  formatted result and the parse diagnostics.
- [ ] A source in another layout gives the canonical text and `changed: true`.
- [ ] A source in the canonical format gives the same text and `changed: false`.
- [ ] A source that does not parse gives its parse diagnostics and no `formatted`, as a normal result.
- [ ] A source with a validation error still gives its formatted text.
- [ ] A call with a path does not change the file, and writes no other file under the root.
- [ ] Through the built binary, for each file in `examples/` and `internal/parser/testdata/`, `formatted` is
  the text that `emod fmt` writes to a copy of the file, and `changed` matches the exit code of
  `emod fmt --check`. A file that does not parse gives the lines that `emod fmt` prints.

**Patterns to follow:**
- `internal/mcp/server.go` (`slicesListTool`, `listSlices`)
- `internal/mcp/server_test.go` (the group "list slices")
- `cmd/emod/main_test.go` (the group "list slices", `listSlicesCLI`)

**Affected files:**
- `internal/mcp/server.go`
- `internal/mcp/server_test.go`
- `cmd/emod/main_test.go`

**Certainty:** high. `emod_slices_list` is the precedent. **Blast radius:** low. The tool is read-only.

**Depends on:** none

### Task 2: Document `emod_fmt`

**Behavior:** The MCP page of the docs describes the tool, and the README names it.

**Acceptance Criteria:**
- [ ] `site/content/docs/mcp.md` has a section on `emod_fmt`: its result, the result of a parse error, and
  that the host writes `formatted` with its own file tool.
- [ ] The README section "Check models from an agent (MCP)" names the tool.

**Patterns to follow:**
- `site/content/docs/mcp.md`

**Affected files:**
- `site/content/docs/mcp.md`
- `README.md`

**Certainty:** high. **Blast radius:** low.

**Depends on:** 1

## Summary

| AC | Task |
| --- | --- |
| AC1 | 1 |
| AC2 | 1 |
| AC3 | 1 |
| AC4 | 1 |
| AC5 | 1 |

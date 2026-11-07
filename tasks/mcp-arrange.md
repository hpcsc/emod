# US-009: Arrange the slices of a model from an agent

## Contents
1. Task 1: Arrange the slices with `emod_slices_arrange`
2. Task 2: Document `emod_slices_arrange`

---

## Story Reference

`user-stories/mcp-server.md` → **US-009: Arrange the slices of a model from an agent**. The design is in
`docs/proposals/ai/01-mcp-server.md`, section "Tool Surface", with the notes "A parse error stops some
tools" and "The arrange report is the useful half". The proposal sets the direction, and this story sets the
scope.

The story's acceptance criteria, numbered for the Summary:

- **AC1.** `emod_slices_arrange` takes `path` or `content` and returns `arranged`, `changed`, `moved`,
  `backwardBefore`, `backwardAfter` and `backward`. Each item in `backward` has `kind`, `label`, `from` and
  `to`.
- **AC2.** `arranged` is the same text that `emod slices arrange` writes for that source.
- **AC3.** When no slice moves, `changed` is false and `arranged` is the same as the source. `changed` gives
  the same answer as the exit code of `emod slices arrange --check`.
- **AC4.** `moved`, `backwardBefore`, `backwardAfter` and `backward` match the report that
  `emod slices arrange` prints for that source.
- **AC5.** A source that does not parse returns the parse diagnostics and no report.
- **AC6.** The tool does not write to the file.

## Boundaries

**Out of scope:**

- `write: true` and `--allow-write` (US-010). The tool has no `write` input.
- A change to how `internal/arrange` orders the slices, or to the output of `emod slices arrange`.
- Added by this breakdown:
  - **The result.** A source that parses gives
    `{"arranged":"...","changed":...,"moved":...,"backwardBefore":...,"backwardAfter":...,"backward":[...]}`.
    `backward` is an empty list, not null, when no reference points backward. `from` and `to` are the names
    of the two slices, as the CLI prints them. A source that does not parse gives `{"diagnostics":[...]}`
    with the parse diagnostics as the records of `emod_check`, and no report. Both are normal results, as for
    `emod_slices_list` and `emod_fmt`. The text block holds the same JSON.
  - **The tool takes the same input as `emod_check`**: `content`, `path` and `name`, with the same rules.
  - **Only parse diagnostics stop the tool**, as in the CLI. A source with validation errors or warnings
    still gives its report.

## Codebase Context

- **The CLI.** `RunSlicesArrange` (`internal/cli/slices_arrange.go`) reads the file, runs `oracle.Parse`, and
  returns the parse diagnostics as one error, one line each. Else it runs `arrange.Model(model)`, which
  orders the slices of the model in place and gives an `arrange.Report`. With `--check` it returns
  `"<path> is not arranged"` when `report.Changed()`, else it prints the report. Without `--check` it writes
  `formatter.Format(model)` only when `report.Changed()`, because a file in which nothing moves must not be
  reformatted, and then prints the report. `internal/cli/app.go` exits with code 1 for both errors.
- **The report text.** `printArrangeReport` prints `moved <n> slice(s); backward references <before> -> <after>`
  when a slice moves, else `already arranged; <after> backward reference(s)`, and then one line
  `  [<kind>] <label> (<from> -> <to>)` for each item of `report.Backward`. When nothing moves, the order
  does not change, so `backwardBefore` is the same as `backwardAfter`.
- **The report type.** `arrange.Report` has `Moved`, `BackwardBefore`, `BackwardAfter` and `Backward`, a list
  of `arrange.Reference` with `From` and `To` as `*ast.Slice`, `Kind` (`subscribes`, `reads`, `on` or
  `flow`) and `Label`.
- **The learnings.** "`emod fmt` canonicalises order, so a fmt golden is never the input re-indented" and the
  learning on the precedent budget of the audit: name the server files only, because the test files of the
  server and the binary spend the budget.
- **The server.** `emod_fmt` (`internal/mcp/server.go`) is the precedent: the same input, `parseDiagnostics`
  and `oneOf`. `cmd/emod/main_test.go` has `fmtCLI` and the group "fmt", which run the CLI on a copy of each
  file.

## Tasks

### Task 1: Arrange the slices with `emod_slices_arrange`

**Behavior:** The server registers `emod_slices_arrange` with the input of `emod_check` and an output schema
of one of the arrange result and the parse diagnostics. A source that parses gives the arranged text, which
is the formatted text when a slice moves and the source when none moves, and the report of
`arrange.Model`. A source that does not parse gives only its parse diagnostics. The tool writes no file.

**Acceptance Criteria:**
- [x] `tools/list` lists `emod_slices_arrange`, with the input of `emod_check` and an output schema of one of
  the arrange result and the parse diagnostics.
- [x] A source with a view after its reader gives the text with the view first, `changed: true`, and the
  counts of the move.
- [x] A source in which no slice moves gives the source as it is, also when it is not in the canonical
  format, and `changed: false`.
- [x] Each reference that stays backward is one item with its kind, its label, and the names of the two
  slices, and `backward` is an empty list when none stays.
- [x] A source that does not parse gives its parse diagnostics and no report, as a normal result.
- [x] A call with a path does not change the file, and writes no other file under the root.
- [x] Through the built binary, for each file in `examples/` and `internal/parser/testdata/`, `arranged` is
  the text that `emod slices arrange` writes to a copy of the file, the report matches the report that it
  prints, and `changed` matches the exit code of `emod slices arrange --check`. A file that does not parse
  gives the lines that the CLI prints.

**Patterns to follow:**
- `internal/mcp/server.go` (`fmtTool`, `formatModel`)
- `internal/cli/slices_arrange.go`

**Affected files:**
- `internal/mcp/server.go`
- `internal/mcp/server_test.go`
- `cmd/emod/main_test.go`

**Certainty:** high. `emod_fmt` is the precedent. **Blast radius:** low. The tool is read-only.

**Depends on:** none

### Task 2: Document `emod_slices_arrange`

**Behavior:** The MCP page of the docs describes the tool, and the README names it.

**Acceptance Criteria:**
- [ ] `site/content/docs/mcp.md` has a section on `emod_slices_arrange`: its result, the meaning of a
  reference that stays backward, the result of a parse error, and that the host writes `arranged` with its
  own file tool.
- [ ] The README section "Check models from an agent (MCP)" names the tool.

**Patterns to follow:**
- `README.md`

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
| AC6 | 1 |

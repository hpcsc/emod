# US-005: List the slices of a model

## Contents
1. Task 1: Give each slice its pattern and its key elements in `internal/ast`
2. Task 2: List the slices with `emod_slices_list`
3. Task 3: Document `emod_slices_list`

---

## Story Reference

`user-stories/mcp-server.md` → **US-005: List the slices of a model**. The design is in
`docs/proposals/ai/01-mcp-server.md`, section "Tool Surface", with the note "Lifted helpers". The proposal
sets the direction, and this story sets the scope.

The story's acceptance criteria, numbered for the Summary:

- **AC1.** `emod_slices_list` takes `path` or `content` and returns one record for each slice, with `name`,
  `pattern`, `context` and `keyElements`.
- **AC2.** For the same source, the records are the same as the output of `emod slices list --format json`.
- **AC3.** A model with no slice returns an empty list.
- **AC4.** A source that does not parse returns the parse diagnostics and no list, as `emod slices list`
  does.

## Boundaries

**Out of scope:**

- `emod slices arrange` and `emod_slices_arrange` (US-009).
- A change to how a slice gets its pattern or its key elements, or to the text output of `emod slices list`.
- Added by this breakdown:
  - **The result is an object.** A source that parses gives `{"slices":[...]}`, also when the list is empty.
    A source that does not parse gives `{"diagnostics":[...]}` with the parse diagnostics as the records of
    `emod_check`, and no `slices` key. Both are normal results, not error results, because diagnostics are
    data. An error result is only for a call that the tool cannot do, such as a path outside the root.
  - **The tool takes the same input as `emod_check`**: `content`, `path` and `name`, with the same rules.
  - **Only parse diagnostics stop the list**, as in the CLI. A source with validation errors or warnings
    still gives its slices.

## Codebase Context

- **The CLI.** `RunSlicesList` (`internal/cli/slices_list.go`) parses with `parseModelFile`, walks
  `model.SliceRefs()`, and calls the private `detectPattern`, `keyElementsForPattern` and
  `automationActivation` for each slice, once for the text rows and once for the JSON entries
  (`sliceJSONEntry`). The JSON output is a bare array, `[]` for a model with no slice.
- **The proposal moves the helpers.** "`detectPattern` and `keyElementsForPattern` ... classify a slice, so
  they move next to `ast.Model.SliceRefs`. Only the column layout of the text output stays in the CLI."
- **The AST.** `internal/ast/ast.go` declares `Slice` and `Automation` with no behaviour.
  `internal/ast/traverse.go` declares `SliceRef` and the methods `Context.SliceRefs` and `Model.SliceRefs`,
  which give every slice of a context, direct or under an aggregate, in source order. The learning "A slice
  has two homes, and much of the repo still walks only one" is why the listing walks `SliceRefs`.
- **The record precedent.** US-001 moved the JSON record of a diagnostic into `diagnostic.Record`, so that
  the CLI and the server write one shape. The slice record follows it.
- **The tests.** `internal/cli/slices_list_test.go` covers each pattern, the order, a DCB context, the three
  timing shapes of an automation, the JSON keys, an empty model and a parse error.
- **The server.** `internal/mcp/server.go` has `sourceInput`, `sourceSchema` and `Server.source`, which
  `emod_check` uses for `content`, `path` and `name`. `cmd/emod/main_test.go` checks the parity of
  `emod_check` with `emod validate --format json` for each file in `examples/` and
  `internal/parser/testdata/`.

## Tasks

### Task 1: Give each slice its pattern and its key elements in `internal/ast`

**Behavior:** `ast.SliceRecords(model)` gives one `ast.SliceRecord` for each slice, with the pattern and
the key elements that `emod slices list` prints, in the order of `SliceRefs`, with the JSON keys `name`, `pattern`, `context` and `keyElements`. `RunSlicesList`
uses them for both formats and keeps only the column layout. The output of `emod slices list` does not
change.

**Acceptance Criteria:**
- [x] `internal/cli` has no `detectPattern`, `keyElementsForPattern`, `automationActivation` or
  `sliceJSONEntry`.
- [ ] Each existing leaf of `internal/cli/slices_list_test.go` passes with no change to the test file.
  Not met as written: no existing leaf changed, but the file has two new leaves. They pin the column
  widths and the text for a model with no slice, which no leaf covered before.
- [x] `ast.SliceRecords(model)` gives an empty list, not nil, for a model with no slice, so that the JSON is
  `[]`.

**Patterns to follow:**
- `internal/ast/traverse.go` (`Model.SliceRefs`)
- `internal/diagnostic/record.go` (`Records`, never nil)
- `internal/cli/slices_list.go`

**Affected files:**
- `internal/ast/slice_record.go` (new)
- `internal/ast/slice_record_test.go` (new)
- `internal/cli/slices_list.go`
- `internal/cli/slices_list_test.go`

**Certainty:** high. The logic moves with no change, and the CLI tests pin its output. **Blast radius:**
low. Only `emod slices list` uses the logic.

**Depends on:** none

### Task 2: List the slices with `emod_slices_list`

**Behavior:** The server registers `emod_slices_list` with the input of `emod_check`. A source that parses
gives `{"slices": ast.SliceRecords(model)}`. A source that does not parse gives `{"diagnostics": ...}` with
the parse diagnostics.

**Acceptance Criteria:**
- [x] `tools/list` lists `emod_slices_list` with `content`, `path` and `name` as optional strings, and a
  description that names `emod slices list --format json`.
- [x] A source with slices gives one record for each slice, with `name`, `pattern`, `context` and
  `keyElements`, in source order, for slices in a DCB context and under an aggregate.
- [x] A source with no slice gives `{"slices":[]}`.
- [x] A source that does not parse gives `{"diagnostics":[...]}` with the parse diagnostics and no `slices`
  key, and the result is not an error result.
- [x] A path outside the root gives the error result of `emod_check`.
- [x] Through the built binary, for each file in `examples/` and `internal/parser/testdata/` that parses, the
  `slices` of a `path` call are the output of `emod slices list --format json` for that file.

**Patterns to follow:**
- `internal/mcp/server.go` (`checkTool`, `Server.source`)
- `internal/mcp/server_test.go`
- `cmd/emod/main_test.go` (the parity leaves of `emod_check`)
- task:1

**Affected files:**
- `internal/mcp/server.go`
- `internal/mcp/server_test.go`
- `cmd/emod/main_test.go`

**Certainty:** high. `emod_check` is the precedent for the input and the parity test. **Blast radius:**
low. The tool is read-only.

**Depends on:** 1

### Task 3: Document `emod_slices_list`

**Behavior:** The MCP page of the docs describes the tool, and the README names it.

**Acceptance Criteria:**
- [x] `site/content/docs/mcp.md` has a section on `emod_slices_list`: its input, the record, the patterns,
  and the result of a source that does not parse.
- [x] The README section "Check models from an agent (MCP)" names the tool.

**Patterns to follow:**
- `site/content/docs/mcp.md`

**Affected files:**
- `site/content/docs/mcp.md`
- `README.md`
- `docs/architecture.md`

**Certainty:** high. **Blast radius:** low.

**Depends on:** 2

## Summary

| AC | Task |
| --- | --- |
| AC1 | 2 |
| AC2 | 1, 2 |
| AC3 | 1, 2 |
| AC4 | 2 |

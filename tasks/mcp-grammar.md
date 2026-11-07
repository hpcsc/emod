# US-004: Give the agent the grammar of emod

## Contents
1. Task 1: Put the examples and the DSL reference into the binary
2. Task 2: Give the schema with `emod_schema` and `emod://schema`
3. Task 3: Serve the examples and the DSL reference as resources
4. Task 4: Serve each resource from any working directory, and document the resources

---

## Story Reference

`user-stories/mcp-server.md` → **US-004: Give the agent the grammar of emod**. The design is in
`docs/proposals/ai/01-mcp-server.md`, section "Tool Surface". The proposal sets the direction, and this story
sets the scope.

The story's acceptance criteria, numbered for the Summary:

- **AC1.** The host lists these resources: `emod://schema`, `emod://examples/all_patterns`,
  `emod://examples/dcb_model`, `emod://examples/specs_hotel` and `emod://docs/dsl-reference`.
- **AC2.** `emod://schema` and the tool `emod_schema` return the same text that `emod schema` prints.
- **AC3.** Each example resource returns the same text as its file in `examples/`, and each example passes
  `emod_check` with no error.
- **AC4.** `emod://docs/dsl-reference` returns the same text as `docs/dsl-reference.md`.
- **AC5.** The server does not serve `examples/error_diagnostics_test.emod`.
- **AC6.** An installed binary serves each resource from any working directory, with no copy of the emod
  repository on disk.

## Boundaries

**Out of scope:**

- The lint rules (US-003 serves them) and any prompt.
- A change to the text of an example, of the schema or of the DSL reference.
- A resource for each section of the DSL reference.
- Added by this breakdown:
  - **The three examples are named in one list.** The list is in the package that embeds them, with a
    description of each one for the host. A new file in `examples/` is not served until it gets a
    description, and a guard in that package fails until then. The binary holds no
    `_test.emod` file.
  - **MIME types.** `emod://schema` and each example are `text/plain`. `emod://docs/dsl-reference` is
    `text/markdown`.
  - **`emod_schema` takes no input** and returns the schema as one text block, with no structured content.

## Codebase Context

- **The schema.** `internal/cue/embed.go` embeds `schema.cue` as `cue.Schema`, and `emod schema`
  (`RunSchema` in `internal/cli/schema.go`) prints it with `fmt.Print`, so the text has no added newline.
- **No embed reaches `examples/` or `docs/`.** `//go:embed` takes only paths under the folder of the package,
  so `internal/mcp` cannot embed `../../examples`. A Go package in each folder embeds its own files, as
  `internal/cue/embed.go` does for `schema.cue`. Each reader of `examples/` in the tests takes only `.emod`
  files: `examplePaths` in `internal/cli/validate_test.go` and `internal/pipeline/pipeline_test.go`, and the
  globs in `internal/lsp/token_test.go`, `internal/importer/diagram_edit_test.go` and
  `cmd/emod/main_test.go`. So a `.go` file in `examples/` does not count as a model.
- **The examples.** `examples/` holds `all_patterns.emod` (the four slice patterns), `dcb_model.emod` (a DCB
  context with `tags` and `decides_on`), `specs_hotel.emod` (Given-When-Then specs) and
  `error_diagnostics_test.emod`, which has errors on purpose. The learning "`examples/` is enumerated by its
  guard, and `_test.emod` means authored to fail" names the convention, and the learning "A
  directory-enumerating guard asserts nothing about a file being there" says why a named file needs its own
  assertion.
- **The server.** `internal/mcp/server.go` registers `emod_check`, `emod_explain_rule` and `emod://rules`.
  The learning "An MCP tool over the go-sdk typed AddTool is routine, not low certainty" describes the SDK:
  it fills an empty URI and MIME type of resource contents, and it escapes `<`, `>` and `&` in the text
  block that it writes for a tool.
- **The tests.** `internal/mcp/server_test.go` asserts one resource in "lists emod://rules as JSON, with a
  description". `cmd/emod/main_test.go` starts `emod mcp` with its working directory at the repository root.

## Tasks

### Task 1: Put the examples and the DSL reference into the binary

**Behavior:** The package `examples` (in `examples/`) embeds the three examples that validate, and
`examples.All()` returns each one with its name, a description and its source, in the order of the names.
The package `docs` (in `docs/`) embeds `dsl-reference.md` as `docs.DSLReference`.

**Acceptance Criteria:**
- [x] `examples.All()` returns `all_patterns`, `dcb_model` and `specs_hotel`, in that order.
- [x] The source of each example is the text of `examples/<name>.emod`.
- [x] The names are the `.emod` files of `examples/` that do not end in `_test.emod`, so a new example fails
  the guard until it gets a description.
- [x] Each example has a description that is not empty.
- [x] `docs.DSLReference` is the text of `docs/dsl-reference.md`.

**Patterns to follow:**
- `internal/cue/embed.go`
- `internal/cli/validate_test.go` (`examplePaths`)

**Affected files:**
- `examples/example.go` (new)
- `examples/example_test.go` (new)
- `docs/dsl_reference.go` (new)
- `docs/dsl_reference_test.go` (new)

**Certainty:** high. `internal/cue/embed.go` is the precedent for an embedded file. **Blast radius:** low.
The binary gets about 57 KB, and nothing else changes.

**Depends on:** none

### Task 2: Give the schema with `emod_schema` and `emod://schema`

**Behavior:** The server registers the tool `emod_schema`, with no input, and the resource
`emod://schema`. Both return `cue.Schema`, the text that `emod schema` prints.

**Acceptance Criteria:**
- [x] `tools/list` lists `emod_schema` with no input property, and with a description that names
  `emod://schema`.
- [x] `emod_schema` returns one text block with the text of `cue.Schema`, with `<`, `>` and `&` as they are.
- [x] `resources/read` of `emod://schema` returns the text of `cue.Schema`, with the MIME type `text/plain`.
- [x] Through the built binary, the text of `emod_schema` and of `emod://schema` is the text that
  `emod schema` prints.

**Patterns to follow:**
- `internal/mcp/server.go` (`explainRuleTool`, `rulesResource`)
- `internal/mcp/server_test.go`
- `cmd/emod/main_test.go`

**Affected files:**
- `internal/mcp/server.go`
- `internal/mcp/server_test.go`
- `cmd/emod/main_test.go`

**Certainty:** high. The learning on the typed `AddTool` and `emod://rules` are the precedent. **Blast
radius:** low.

**Depends on:** none

### Task 3: Serve the examples and the DSL reference as resources

**Behavior:** The server registers `emod://examples/<name>` for each example of `examples.All()`, with its
description, and `emod://docs/dsl-reference` with `docs.DSLReference`.

**Acceptance Criteria:**
- [x] `resources/list` lists `emod://rules`, `emod://schema`, `emod://examples/all_patterns`,
  `emod://examples/dcb_model`, `emod://examples/specs_hotel` and `emod://docs/dsl-reference`, and no other
  resource.
- [x] Each example resource returns the text of its file in `examples/`, with the MIME type `text/plain`, and
  its description is the description of the example.
- [x] `emod_check` with the text of each example resource returns an empty list.
- [x] `emod://docs/dsl-reference` returns the text of `docs/dsl-reference.md`, with the MIME type
  `text/markdown`.
- [x] `resources/read` of `emod://examples/error_diagnostics_test` returns an error.

**Patterns to follow:**
- `internal/mcp/server.go`
- `internal/mcp/server_test.go`
- task:1

**Affected files:**
- `internal/mcp/server.go`
- `internal/mcp/server_test.go`

**Certainty:** high. **Blast radius:** low.

**Depends on:** 1, 2

### Task 4: Serve each resource from any working directory, and document the resources

**Behavior:** The integration test starts the built binary in an empty temporary folder, and reads each
resource of the story. The MCP page of the docs and the README describe the resources and `emod_schema`.

**Acceptance Criteria:**
- [ ] With its working directory in an empty temporary folder, the binary returns for each example resource
  the text of its file, for `emod://docs/dsl-reference` the text of `docs/dsl-reference.md`, and for
  `emod://schema` the text of `emod schema`.
- [ ] `site/content/docs/mcp.md` lists each resource and `emod_schema`, with what each one returns, and says
  that the text comes from the binary, so it matches the version of emod that the agent calls.
- [ ] The README section "Check models from an agent (MCP)" names the grammar resources and `emod_schema`.

**Patterns to follow:**
- `cmd/emod/main_test.go` (`startServer`)
- `site/content/docs/mcp.md`

**Affected files:**
- `cmd/emod/main_test.go`
- `site/content/docs/mcp.md`
- `README.md`

**Certainty:** high. **Blast radius:** low.

**Depends on:** 3

## Summary

| AC | Task |
| --- | --- |
| AC1 | 3 |
| AC2 | 2 |
| AC3 | 1, 3 |
| AC4 | 1, 3 |
| AC5 | 1, 3 |
| AC6 | 1, 4 |

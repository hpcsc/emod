# US-002: Check a model file in the project

## Contents
1. Task 1: Read a model file only from under a root
2. Task 2: Let `emod_check` take a `path`
3. Task 3: Set the root with `emod mcp --root`, and check a path through the built binary
4. Task 4: Document the `path` input and the root

---

## Story Reference

`user-stories/mcp-server.md` → **US-002: Check a model file in the project**. The design is in
`docs/proposals/ai/01-mcp-server.md`, sections "Tools call the code that the CLI calls", "Tool Surface" and
"Risks and Mitigations". The proposal sets the direction, and this story sets the scope.

The story's acceptance criteria, numbered for the Summary:

- **AC1.** `emod_check` accepts `path` in place of `content`. For a regular file under the root of at most
  1 MiB, it returns the same records as `emod validate --format json` for that file. A larger file, a
  directory or a named pipe returns an error result.
- **AC2.** When a call gives both `path` and `content`, the tool checks `content`.
- **AC3.** The root is the working directory of the server. `emod mcp --root <dir>` sets a different root.
- **AC4.** The tool refuses a path outside the root, for example `../../etc/passwd`, an absolute path to
  another folder, or a symbolic link to a file outside the root. Under the root, the tool follows only
  relative symbolic links, so it also refuses an absolute symbolic link. The error result names the root
  and holds no text from the file.
- **AC5.** A path that does not exist, or that the server cannot read, returns an error result that names
  the path and the cause.
- **AC6.** The tool never writes to the file.

## Boundaries

**Out of scope:**

- Any tool other than `emod_check`, any resource, and any prompt. US-003 to US-009 add them, and each one
  takes the same `path` or `content` input that this story gives `emod_check`.
- `--allow-write` and any write to disk. US-010 adds the write mode.
- A `.mcp.json` in the repository, and the setup for hosts other than Claude Code. US-011 adds them.
- The non-goals of the story set in `user-stories/mcp-server.md`.
- Added by this breakdown:
  - **When a call gives `content`, the tool does not read `path`.** The path then only names the records,
    so it needs no root check, and it can name a file that does not exist yet: an agent checks the draft
    of a new file before it writes the file.
  - **The records of `content` carry `name`, else `path`, else `inline.emod`.** The records of a `path`
    with no `content` carry the path as the call gives it, as `emod validate --format json <path>` does.
    `name` applies only to `content`.
  - A call with neither `content` nor `path` is an error result that names both.

## Codebase Context

- **The tool.** `internal/mcp/server.go` registers `emod_check` with `sdk.AddTool` over `sourceInput`
  (`Content`, `Name`) and `checkResult`. The handler `(*Server).check` gives the name `inline.emod` to a call
  with no name. `internal/mcp/server_test.go` drives the server through an SDK client over `io.Pipe`
  (`connect`). The learning "An MCP tool over the go-sdk typed AddTool is routine, not low certainty"
  describes the SDK behaviour.
- **The schema.** The SDK infers the input schema from the struct with `jsonschema.For`. A field with no
  `omitempty` is required. A pointer field gets the type `["null","string"]` (checked with a probe). To tell
  an absent `content` from an empty one, `Content` must be a pointer, so the tool sets the type of
  `content` to `string` in the schema, and the US-001 leaf on the input schema keeps holding.
  `github.com/google/jsonschema-go` is already a direct requirement in `go.mod`.
- **The CLI reads files with `readSourceFile`** (`internal/cli/parse.go:13-31`): the message is
  `reading <path>: <cause>`.
- **Symbolic links.** `filepath.EvalSymlinks` refuses a path that does not exist.
  `internal/desktop/file_service.go:188-213` follows links by hand for a save; a read needs no such loop. On
  macOS, `t.TempDir()` is under `/var`, a link to `/private/var`, so the root must be resolved before any
  comparison.
- **The command.** `internal/cli/app.go` has the `mcp` command with no flags, and `RunMCP(ctx)` in
  `internal/cli/mcp.go`. `internal/cli/mcp_test.go` drives the command over swapped `os.Stdin` and
  `os.Stdout`. The integration test `cmd/emod/main_test.go` builds the binary once and starts `emod mcp`
  with its working directory at the repository root.
- **The docs.** `site/content/docs/mcp.md` documents `emod_check` with `content` and `name`.

## Tasks

### Task 1: Read a model file only from under a root

**Behavior:** `internal/mcp` gets a root: the one directory under which the server reads files. The root
resolves its directory once, with symbolic links followed. It reads the text of a path that is under it,
relative to the root or absolute, and refuses every other path with an error that names the root.

**Acceptance Criteria:**
- [ ] A root made from a directory reads a file under it by a relative path and by an absolute path.
- [ ] A root made from a symbolic link to a directory reads the files of that directory.
- [ ] A root made from a path that does not exist, or from a file, gives an error that names the path.
- [ ] A read of `../outside.emod`, of an absolute path in another folder, and of a symbolic link under the
  root to a file outside it gives an error that names the root. The error holds no text from the file.
- [ ] A read of a path under the root that does not exist gives an error that names the path and says that
  the file does not exist. A read of a file with no read permission names the path and the cause.
- [ ] A read changes neither the bytes nor the modification time of the file.

**Affected Files/Modules:**
- `internal/mcp/root.go` (new): the root
- `internal/mcp/root_test.go` (new, `//go:build unit`): the tests of the root

**Patterns to Follow:**
- `internal/cli/parse.go:13-31` for the text `reading <path>: <cause>`
- `internal/desktop/file_service.go:188-213` for `filepath.EvalSymlinks`

**Testable:** Yes. The root is an exported API that the server and the command call.

**Certainty:** high. `filepath.EvalSymlinks` and `filepath.Rel` decide the containment, as
`internal/desktop/file_service.go:188-213` uses `EvalSymlinks` for a save.

**Blast radius:** high. The root is the security boundary of a server that a person registers for all
projects. A path that escapes it reads any file that the user can read.

**Verification:** `mise exec -- task test:unit`.

**Depends on:** None

---

### Task 2: Let `emod_check` take a `path`

**Behavior:** `emod_check` takes `path` as an alternative to `content`. With no `content`, the tool reads
the file through the root of the server and checks it, and the records carry the path. With `content`, the
tool checks `content` and does not read `path`. A call with neither is an error result.

**Acceptance Criteria:**
- [ ] `tools/list` shows `content`, `path` and `name` as strings, and none of them is required.
- [ ] A call with `path` and no `content` returns the records of that file, each with the path as its
  `file`, with no `isError`.
- [ ] A call with `path` and `content` checks `content`. With no `name`, its records carry the path.
- [ ] A call with a path outside the root returns an error result that names the root.
- [ ] A call with neither `content` nor `path` returns an error result that names both.
- [ ] A call with a path that does not exist returns an error result that names the path.
- [ ] The description of `emod_check` says what `path` and `content` do, and that `content` wins.

**Affected Files/Modules:**
- `internal/mcp/server.go`: the input, the schema, the handler and the root of the server
- `internal/mcp/server_test.go`: the leaves above, and `connect` with a root

**Patterns to Follow:**
- `internal/mcp/server.go`
- `internal/mcp/server_test.go`
- `task:1`

**Testable:** Yes. The caller is an inbound MCP client.

**Certainty:** high. The tool, its schema and its tests follow `internal/mcp/server.go` and
`internal/mcp/server_test.go` from US-001.

**Blast radius:** high. The input of `emod_check` is a public contract, and US-003 to US-009 take the same
input.

**Verification:** `mise exec -- task test:unit`.

**Depends on:** Task 1

---

### Task 3: Set the root with `emod mcp --root`, and check a path through the built binary

**Behavior:** `emod mcp` makes its root from the working directory, or from `--root <dir>`. A root that it
cannot use stops the command with an error that names the directory. The integration test checks each
fixture file by its path through the built binary and compares the records with `emod validate`.

**Acceptance Criteria:**
- [ ] `emod mcp --help` lists `--root`, and says that the default is the working directory.
- [ ] `emod mcp --root <dir>` reads a path relative to that directory.
- [ ] `emod mcp --root <a path that does not exist>` ends with exit code 1 and a message that names the
  path.
- [ ] The integration test calls `emod_check` with the path of each `.emod` file under `examples/` and
  `internal/parser/testdata/`, relative to the repository root, and gets the records that the same binary
  prints for `emod validate --format json <path>`.

**Affected Files/Modules:**
- `internal/cli/app.go`: the `--root` flag of `mcp`
- `internal/cli/mcp.go`: `RunMCP` makes the root
- `internal/cli/mcp_test.go`: the leaves of the flag
- `cmd/emod/main_test.go`: the path leaves

**Patterns to Follow:**
- `internal/cli/app.go`
- `internal/cli/mcp_test.go`
- `cmd/emod/main_test.go`
- `task:2`

**Testable:** Yes.

**Certainty:** high. The flag follows the other string flags of `internal/cli/app.go`, and the tests extend
`internal/cli/mcp_test.go` and `cmd/emod/main_test.go`.

**Blast radius:** low. The command gets one flag, and its default keeps the behaviour of US-001.

**Verification:** `mise exec -- task test:unit && mise exec -- task test:integration`.

**Depends on:** Task 2

---

### Task 4: Document the `path` input and the root

**Behavior:** The docs page of the MCP server tells what `path` does, that `content` wins, which name the
records carry, and how the root and `--root` limit the files that the server reads.

**Acceptance Criteria:**
- [ ] `site/content/docs/mcp.md` lists `path` in the input table, and says that `content` wins when a call
  gives both.
- [ ] The page says that the server reads only files under its root, that the root is the working
  directory, and that `emod mcp --root <dir>` sets another root.
- [ ] The page documents no `--allow-write`, no `.mcp.json`, and no host other than Claude Code.

**Affected Files/Modules:**
- `site/content/docs/mcp.md`

**Patterns to Follow:**
- `site/content/docs/mcp.md`

**Testable:** No. It is documentation only.

**Certainty:** high. The page exists, and the task adds to it.

**Blast radius:** low. The task changes documentation only.

**Verification:** `mise exec -- hugo --source site --destination <scratch>` builds the page.

**Depends on:** Task 3

---

## Summary

- **Total tasks:** 4. Three are code tasks, and the last task is documentation.
- **Ordering rationale:** the root is the security boundary, so it comes first, with its own tests. The tool
  uses it next. The flag and the end-to-end check come last among the code tasks, so the binary reads no
  path until the server can limit it.
- **Acceptance criteria coverage:**
  - AC1: Task 2 (the tool reads a path) and Task 3 (parity with `emod validate` through the binary).
  - AC2: Task 2.
  - AC3: Task 3.
  - AC4: Task 1 (each kind of path outside the root) and Task 2 (the error result).
  - AC5: Task 1 and Task 2.
  - AC6: Task 1.

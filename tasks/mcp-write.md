# US-010: Let a trusted agent write the formatted and arranged model

## Contents
1. Task 1: Replace the text of a file under the root
2. Task 2: Write the result of `emod_fmt` and `emod_slices_arrange` with `write: true`
3. Task 3: Allow the writes with `emod mcp --allow-write`
4. Task 4: Document the write mode

---

## Story Reference

`user-stories/mcp-server.md` → **US-010: Let a trusted agent write the formatted and arranged model**. The
design is in `docs/proposals/ai/01-mcp-server.md`, sections "Tool Surface", "Interface" and "Risks and
Mitigations" ("A tool that writes is dangerous by default"). The proposal sets the direction, and this story
sets the scope.

The story's acceptance criteria, numbered for the Summary:

- **AC1.** When the server starts with `emod mcp --allow-write`, `emod_fmt` and `emod_slices_arrange` write
  their result to the file in `path` for a call that gives `write: true`.
- **AC2.** When the server starts without `--allow-write`, a call with `write: true` returns an error result
  that names the `--allow-write` flag. The file on disk does not change.
- **AC3.** A call with `write: true` must give `path` and no `content`. Any other call with `write: true`
  returns an error result and writes nothing.
- **AC4.** A write follows the root of US-002. The tool refuses a path outside the root and writes nothing.
- **AC5.** When the result is the same as the file, the tool does not write the file, and the modification
  time of the file does not change.
- **AC6.** A source that does not parse is never written.
- **AC7.** No other tool writes to disk, with or without `--allow-write`.

## Boundaries

**Out of scope:**

- A write of text that the call gives. The tool writes only a new layout of the file on disk.
- A new file. The tool writes only a file that exists.
- US-011 (the registration for teams and other hosts).
- Added by this breakdown:
  - **The write replaces only the text that the tool read.** The tool reads the file, and then writes it. If
    the file changes between the two steps, for example because a person saves it in the desktop app, the
    root refuses the write with an error that says that the file changed, and the file keeps the new text.
    The story says that the tool writes "only a new layout of the file that is already on disk", and a write
    over a newer text breaks that.
  - **The write is in place**, as `emod fmt` and `emod slices arrange` write: the same file, its mode kept,
    and a relative link under the root followed as for a read. The root does not create a file.
  - **The order of the checks** for a call with `write: true`: first `--allow-write`, then `path` and no
    `content`, then the root, then the parse. Each refusal is an error result, except a parse error, which
    gives the parse diagnostics as a normal result, as without `write`.
  - **The result does not change.** A call with `write: true` gives the same result as one without it. The
    tool writes the file when `changed` is true.

## Codebase Context

- **The root.** `internal/mcp/root.go` reads a path through `os.Root`: `relativeOf` maps an absolute path to
  the root by `os.SameFile`, `escapeMessage` tells an escape apart, `O_NONBLOCK` keeps a named pipe from a
  wait, and the file must be regular. The learning "Read a caller's path under a root through os.Root, and
  match the root by file identity" lists the cases that failed three audit rounds, and says to plan such a
  task as low certainty, with each case in its criteria. Go 1.25 is the module version.
- **The server.** `internal/mcp/server.go` has `sourceInput`, `sourceSchema` and `Server.source`, and the
  handlers `formatModel` and `arrangeSlices`. `NewServer(in, out, root)` is called from
  `internal/cli/mcp.go` and from `connectAt` in `internal/mcp/server_test.go`.
- **The CLI.** `internal/cli/app.go` declares `emod mcp` with `--root` and calls `RunMCP(ctx, root)`.
  `internal/cli/mcp_test.go` checks the help text of the flags and serves through stdin and stdout.
  `cmd/emod/main_test.go` starts the built binary with `startServerIn`.
- **The CLI writes.** `RunFmt` and `RunSlicesArrange` write with `os.WriteFile`, only when the text differs.

## Tasks

### Task 1: Replace the text of a file under the root

**Behavior:** `Root.Replace(path, old, new)` writes `new` to the file at `path` under the root, in place,
when the file still holds `old`. It follows the rules of `Root.Read` for the path, and it never creates a
file.

**Acceptance Criteria:**
- [x] Replaces the text of a file under the root by a relative path and by an absolute path, and keeps the
  mode of the file.
- [x] Refuses a path that climbs out of the root, an absolute path in another directory, and a symbolic link
  to a file outside the root, as outside the root, and changes no file.
- [x] Refuses a missing file and does not create it.
- [x] Refuses a directory and a named pipe as not a regular file, and does not wait for a reader.
- [x] Refuses a file that no longer holds `old`, with an error that says that the file changed, and the file
  keeps its text.
- [x] Writes the file that a relative symbolic link under the root points to, as `Root.Read` reads it.
- [x] A file with no write permission gives an error that names the path and the cause.

**Patterns to follow:**
- `internal/mcp/root.go`

**Affected files:**
- `internal/mcp/root.go`
- `internal/mcp/root_test.go`

**Certainty:** low. The learning on the root records three audit rounds of path cases, and a write adds the
check of the old text. **Blast radius:** high. A defect writes over a model file of the user, or outside the
root.

**Depends on:** none

### Task 2: Write the result of `emod_fmt` and `emod_slices_arrange` with `write: true`

**Behavior:** `emod_fmt` and `emod_slices_arrange` take a boolean `write`. The server takes whether it allows
writes. With `write: true` and writes allowed, a call with `path` and no `content` writes the formatted or
arranged text with `Root.Replace` when `changed` is true. Every other call with `write: true` gives an error
result and writes nothing.

**Acceptance Criteria:**
- [x] `tools/list` gives `emod_fmt` and `emod_slices_arrange` a boolean `write`, with a description that names
  `--allow-write` and `path`, and the other tools no `write`.
- [x] On a server that allows writes, `write: true` with a path writes the text of `formatted` or `arranged`
  to the file when `changed` is true, and gives the same result as a call without `write`.
- [x] On a server that does not allow writes, `write: true` gives an error result that names `--allow-write`,
  and the file does not change.
- [x] `write: true` with `content`, or with no `path`, gives an error result that says that a write needs
  `path` and no `content`, and writes nothing.
- [x] `write: true` with a path outside the root gives the error result of the root, and writes nothing.
- [x] `write: true` for a file in the canonical layout, or one in which no slice moves, writes nothing, and the
  modification time of the file does not change.
- [x] `write: true` for a file that does not parse gives the parse diagnostics, and the file does not change.
- [x] On a server that allows writes, a call of each other tool with a path, and a call of `emod_fmt` and
  `emod_slices_arrange` with no `write`, change no file under the root.

**Patterns to follow:**
- `internal/mcp/server.go` (`formatModel`, `arrangeSlices`, `formatSchema`)

**Affected files:**
- `internal/mcp/server.go`
- `internal/mcp/server_test.go`
- `internal/cli/mcp.go`

**Certainty:** medium. The tools and their tests are the precedent; the gate is new. **Blast radius:** high.
The gate is what keeps the default server read-only.

**Depends on:** 1

### Task 3: Allow the writes with `emod mcp --allow-write`

**Behavior:** `emod mcp` has a boolean flag `--allow-write`, and passes it to the server. Without the flag,
the server allows no write.

**Acceptance Criteria:**
- [ ] `emod mcp --help` lists `--allow-write` with a usage that names `emod_fmt` and `emod_slices_arrange`.
- [ ] Through the built binary, `emod mcp --allow-write` writes the text that `emod fmt` writes to an
  unformatted copy of a model, and the text that `emod slices arrange` writes to an unarranged copy, for a
  call with `write: true`.
- [ ] Through the built binary, `emod mcp` with no flag gives an error result that names `--allow-write` for
  a call with `write: true`, and the file does not change.

**Patterns to follow:**
- `internal/cli/app.go` (the `--root` flag of `emod mcp`)
- `internal/cli/mcp_test.go`

**Affected files:**
- `internal/cli/app.go`
- `internal/cli/mcp.go`
- `internal/cli/mcp_test.go`
- `cmd/emod/main_test.go`

**Certainty:** high. `--root` is the precedent. **Blast radius:** medium. The default of the flag decides
whether a server for all projects can write.

**Depends on:** 2

### Task 4: Document the write mode

**Behavior:** The MCP page of the docs describes `--allow-write` and `write: true`, and the README names
them.

**Acceptance Criteria:**
- [ ] `site/content/docs/mcp.md` describes `--allow-write`, `write: true`, the rules for a call with `write`,
  the refusal of a file that changed after the read, and that no other tool writes.
- [ ] The section "The root" of the MCP page no longer says that the server never writes to a file, and says
  when it writes.
- [ ] The README section "Check models from an agent (MCP)" names `--allow-write`.

**Patterns to follow:**
- `site/content/docs/mcp.md`

**Affected files:**
- `site/content/docs/mcp.md`
- `README.md`

**Certainty:** high. **Blast radius:** low.

**Depends on:** 3

## Summary

| AC | Task |
| --- | --- |
| AC1 | 1, 2, 3 |
| AC2 | 2, 3 |
| AC3 | 2 |
| AC4 | 1, 2 |
| AC5 | 2 |
| AC6 | 2 |
| AC7 | 2 |

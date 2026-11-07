# US-011: Register the server for a team and for other MCP hosts

## Contents
1. Task 1: Register `emod mcp` in the `.mcp.json` of the repository
2. Task 2: Check the server with the MCP Inspector
3. Task 3: Document the registration in each host

---

## Story Reference

`user-stories/mcp-server.md` → **US-011: Register the server for a team and for other MCP hosts**. The design
is in `docs/proposals/ai/01-mcp-server.md`, section "Register the server with a host". The proposal sets the
direction, and this story sets the scope.

The story's acceptance criteria, numbered for the Summary:

- **AC1.** The user documentation shows how to register `emod mcp` in Claude Code, Claude Desktop, Cursor and
  Zed.
- **AC2.** The documentation shows a project file, `.mcp.json`, that starts the server with `--root .`, so
  that each person who opens the project gets the server.
- **AC3.** The emod repository has this `.mcp.json`, and Claude Code starts the server from it in the
  repository.
- **AC4.** The server passes the checks of the MCP Inspector: the handshake, the list of tools and a call to
  each tool, and the list and read of each resource.
- **AC5.** In each of the four hosts, a person who follows the documented steps gets a result from
  `emod_check`.
- **AC6.** The documentation says that the server needs no API key and no AI settings.

## Boundaries

**Out of scope:**

- A change to the server, its tools or its resources.
- An installer or a package for a host, for example a Claude Desktop extension.
- Added by this breakdown:
  - **The run checks Claude Code and the Inspector, not the other three hosts.** AC5 needs a person in
    Claude Desktop, Cursor and Zed. The run checks the configuration of each host against the docs of that
    host, and it says which checks a person must still do.
  - **The Inspector check is a script**, `scripts/mcp-inspector.sh`, behind `task check:mcp`, so that a
    person can run it again. It is not part of `task test`, because it downloads the Inspector with `npx`.
    It pins the version of the Inspector. It calls each tool that `tools/list` gives, with arguments that
    the script names for each tool, and it fails for a tool that the script does not know. It reads each
    resource that `resources/list` gives.
  - **The paths in each host.** Claude Code and Cursor start the server for a project: the project file
    gives `--root .` for Claude Code and `--root ${workspaceFolder}` for Cursor. Claude Desktop and Zed have
    no project file and no variable for the project in their docs, so the docs give an absolute `--root`.
    Claude Desktop starts the server with a short `PATH`, so its docs give the absolute path of `emod`.

## Codebase Context

- **The repository file.** `.mcp.json` registers `gopls` only (commit "Register gopls and its build tags
  with the repo"). The `emod` entry goes next to it.
- **The docs.** `site/content/docs/mcp.md` has "Register the server with Claude Code" with
  `claude mcp add emod -- emod mcp`, and says in its first section that the server needs no API key and no
  `EMOD_AI_*` setting. The README section "Check models from an agent (MCP)" shows the same command.
- **The binary tests.** `cmd/emod/main_test.go` builds the binary once, and `startServerIn` starts
  `emod mcp` in a directory with flags. The learning "cmd/emod/main_test.go builds emod once and drives it as
  an MCP host" applies.
- **The hosts.** Claude Desktop reads `claude_desktop_config.json` (macOS:
  `~/Library/Application Support/Claude/`, Windows: `%APPDATA%\Claude\`) with `mcpServers`. Cursor reads
  `.cursor/mcp.json` in the project or `~/.cursor/mcp.json`, with `mcpServers`, and expands
  `${workspaceFolder}`. Zed reads `context_servers` in its settings file, with `command`, `args` and `env`,
  and shows a green dot in Settings → AI → MCP Servers for a server that works.
- **The Inspector.** `npx @modelcontextprotocol/inspector --cli <command> --method tools/list`, with
  `tools/call` (`--tool-name`, `--tool-args-json`), `resources/list` and `resources/read` (`--uri`). The
  version on npm is 2.14.0.
- **The scripts.** `e2e/entrypoint.sh` is the one shell script of the repository.

## Tasks

### Task 1: Register `emod mcp` in the `.mcp.json` of the repository

**Behavior:** `.mcp.json` registers `emod` with the command `emod` and the arguments `mcp`, `--root` and
`.`, next to `gopls`. A test starts the built binary with the arguments of that entry in the root of the
repository and gets a result from `emod_check` for a path in the repository.

**Acceptance Criteria:**
- [x] `.mcp.json` has an `emod` entry with the command `emod` and the arguments `["mcp", "--root", "."]`, and
  keeps the `gopls` entry.
- [x] Through the built binary, the arguments of that entry, started in the root of the repository, give a
  server that answers `emod_check` for `examples/all_patterns.emod` with no diagnostic.
- [ ] `claude mcp list` in the repository, with the built binary first on `PATH`, shows `emod` as connected.
  Not met as written: `claude mcp list` shows `emod: emod mcp --root . - ⏸ Pending approval`, because Claude
  Code asks a person to approve the servers of a project file once. With `claude -p --mcp-config .mcp.json`,
  Claude Code started the server from the file and `emod_check` gave `{"diagnostics":[]}`.

**Patterns to follow:**
- `cmd/emod/main_test.go` (`startServerIn`, the group "read grammar")

**Affected files:**
- `.mcp.json`
- `cmd/emod/main_test.go`

**Certainty:** medium. `startServerIn` is the precedent; how Claude Code starts a project server is not in
the repository. **Blast radius:** low. The file only adds a server for this repository.

**Depends on:** none

### Task 2: Check the server with the MCP Inspector

**Behavior:** `task check:mcp` builds the binary and runs `scripts/mcp-inspector.sh`, which runs the
Inspector CLI against `emod mcp --root .`: the list of tools, a call to each tool, the list of resources and
a read of each resource. The script fails on an error of the Inspector, on an `isError` result, on a tool
that it has no arguments for, and on an empty read.

**Acceptance Criteria:**
- [x] `task check:mcp` passes on the branch, and its output names each tool and each resource that it
  checked.
- [x] The script fails for a tool that `tools/list` gives and the script does not know.
- [x] The script fails when a call gives an `isError` result.
- [x] `task test` does not run the script.

**Patterns to follow:**
- `Taskfile.yml` (`build`)
- `e2e/entrypoint.sh`

**Affected files:**
- `scripts/mcp-inspector.sh` (new)
- `Taskfile.yml`

**Certainty:** medium. The CLI flags of the Inspector come from its docs, and the repository has no
precedent. **Blast radius:** low. The check runs only on request.

**Depends on:** 1

### Task 3: Document the registration in each host

**Behavior:** The MCP page of the docs shows how to register the server in Claude Code, Claude Desktop,
Cursor and Zed, shows the project file `.mcp.json` with `--root .`, and says that the server needs no API
key and no AI settings. The README points to it.

**Acceptance Criteria:**
- [ ] `site/content/docs/mcp.md` shows the registration in Claude Code (`claude mcp add`, and the project
  scope that writes `.mcp.json`), Claude Desktop, Cursor and Zed, each with the file and the JSON, and how
  to see that the server works.
- [ ] The page shows the `.mcp.json` of a project with `--root .`, the same entry as the repository has.
- [ ] The page says that the server needs no API key and no AI settings, in the section on registration.
- [ ] The README section "Check models from an agent (MCP)" shows the project file and names the other
  hosts.

**Patterns to follow:**
- `site/content/docs/mcp.md`

**Affected files:**
- `site/content/docs/mcp.md`
- `README.md`

**Certainty:** medium. The formats come from the docs of each host, and only Claude Code is checked here.
**Blast radius:** low.

**Depends on:** 1

## Summary

| AC | Task |
| --- | --- |
| AC1 | 3 |
| AC2 | 1, 3 |
| AC3 | 1 |
| AC4 | 2 |
| AC5 | 1, 3 (Claude Code only; the other hosts need a person) |
| AC6 | 3 |

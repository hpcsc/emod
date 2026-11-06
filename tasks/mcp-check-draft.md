# US-001: Check a draft model from an agent

## Contents
1. Task 1: Give the diagnostic record of `--format json` one home in `internal/diagnostic`
2. Task 2: Serve `emod_check` over MCP from `internal/mcp`
3. Task 3: Start the server with `emod mcp` and check the handshake with the built binary
4. Task 4: Document the registration of `emod mcp` with Claude Code

---

## Story Reference

`user-stories/mcp-server.md` → **US-001: Check a draft model from an agent** (lines 69-94). The design
is in `docs/proposals/ai/01-mcp-server.md`, sections "How It Works", "Tool Surface" and "Interface". The
proposal sets the direction, and this story sets the scope.

The story's acceptance criteria, numbered for the Summary:

- **AC1.** `emod mcp` starts a server that speaks MCP over stdin and stdout. An MCP host that starts the
  server lists the tool `emod_check`.
- **AC2.** `emod_check` takes `.emod` text in `content` and returns a list of diagnostics. Each record has
  `file`, `line`, `severity` and `message`, and `rule` when a rule name goes with the diagnostic.
- **AC3.** For the same source, `emod_check` returns the same records as `emod validate --format json`. A
  clean draft returns an empty list.
- **AC4.** The records carry the file name from the `name` input, or `inline.emod` when the call gives no
  `name`.
- **AC5.** Errors and warnings are normal results. The tool returns an error result (`isError`) only when
  it cannot do the check, for example when the call gives no `content`.
- **AC6.** The server works with no API key, no `EMOD_AI_*` setting and no network access.
- **AC7.** The user documentation shows the command that registers the server with Claude Code:
  `claude mcp add emod -- emod mcp`.

## Boundaries

**Out of scope:**

- The `path` input, `emod mcp --root`, and any file read by the server. US-002 adds them. `emod_check`
  in this story reads only `content`.
- Any tool other than `emod_check`, any resource, and any prompt. US-003 to US-009 and the prompt
  proposals add them.
- `--allow-write` and any tool that writes to disk. US-010 adds the write mode.
- Registration for Claude Desktop, Cursor and Zed, a `.mcp.json` in the repository, and the MCP
  Inspector checks. US-011 adds them.
- The non-goals of the story set, verbatim from `user-stories/mcp-server.md`:
  - **The workflow prompts.** Each prompt goes to the server with the proposal that writes its text:
    - `model-domain` with [04 model from a conversation](../docs/proposals/ai/04-model-from-conversation.md)
    - `review-model` with [02 semantic model review](../docs/proposals/ai/02-semantic-model-review.md)
    - `import-from-code` with [07 model import](../docs/proposals/ai/07-model-import.md)
    - `suggest-specs` with [05 spec suggestions](../docs/proposals/ai/05-spec-suggestions.md)
  - **The query tools.** `emod_references` and `emod_trace` go to the server with
    [exact model queries](../docs/proposals/model-queries-proposal.md), which builds `emod references` and
    `emod trace`.
  - **Spec export.** `emod_specs_export` goes to the server with
    [05 spec suggestions](../docs/proposals/ai/05-spec-suggestions.md), which builds `emod specs export`.
  - No call to a model from emod. emod adds no provider, no API key and no network access.
  - No tool that writes `.emod` text from a call to disk. The host writes files with its own file tools.
  - No HTTP or SSE transport, and no hosted server for many users.
  - No viewer server like `emod diagram --serve`.
- Added by this breakdown:
  - **The `rule` key stays optional.** A diagnostic with no rule name, such as each validator hard error,
    has no `rule` key. This is true in `emod validate --format json` today, and `emod_check` does the
    same, because AC3 asks for the same records. If every record carried `rule`, the output of
    `emod validate --format json` would change, and that output is a contract outside this repository.
  - The rule names of `orphan-command` and `orphan-event`, which `emod lint --explain` cannot describe,
    stay as they are. They are a known bug and not a precedent.
  - The diagnostics shape of `emod export` (`column`, `rule_name`, in `internal/export/json.go`) is a
    different contract and does not change. This story gives one shape only to the records of
    `--format json` and the records of `emod_check`.
  - No limit on the size of `content`, no MCP logging, no tool annotations, and no progress
    notifications.

**Deferred:**

- A check of the handshake against a client that is not the Go SDK goes to US-011. US-011 runs the MCP
  Inspector, which uses the TypeScript SDK, and the four hosts. The end-to-end check in Task 3 uses a Go
  client, so it cannot find a problem that the Go SDK has on both sides of the connection.
- An MCP case in the Docker-based `e2e/` suite is not planned. That suite runs against a release build
  through `task test:e2e`, which is not part of the Go test commands of this run.

## Codebase Context

- **The check.** `oracle.Check(source, filename)` (`internal/oracle/oracle.go:38-43`) runs the parser,
  the validator and the linter, and returns `[]*diagnostic.Entry`. It does no I/O. `RunValidate`
  (`internal/cli/validate.go`) and `RunLint` (`internal/cli/lint.go:109-134`) both call it, then
  `formatJSON` (`internal/cli/lint.go:41-83`). `formatJSON` turns each entry into the unexported
  `jsonEntry` (`file`, `line`, `rule` with `omitempty`, `severity`, `message`), and marshals a non-nil
  slice, so a clean file prints `[]`. The exit code is 0 when there are no diagnostics, 1 for warnings
  only, and 2 when there is an error. `diagnostic.Entry` (`internal/diagnostic/entry.go`) has no JSON
  tags, and no code marshals it directly.
- **Another diagnostics shape exists.** `internal/export/json.go:239-292` has its own entry with `column`
  and `rule_name` for `emod export`. Its `convertDiagnostics` is the precedent for "no entries gives `[]`,
  not `null`".
- **The current records of the shipped fixtures**, from `emod validate --format json`:
  - `examples/error_diagnostics_test.emod` gives two errors. Line 43 has no `rule` and the message
    `event "GuestCheckedOut" does not exist`. Line 64 has the rule `orphan-command` and the message
    `command "SendEmail" is orphaned (no flow references it)`.
  - `internal/parser/testdata/invalid.emod` gives two parse errors with no rule, on lines 6 and 3.
  - The other files under `examples/` and `internal/parser/testdata/` are clean.
  - An empty file is clean: it prints `[]` and exits 0.
  - The warning-only source in `internal/cli/validate_test.go:1112-1159` gives one `state-obsession`
    warning.
- **The `lsp` precedent.** The `lsp` entry is at `internal/cli/app.go:321-331`, and `RunLSP` is in
  `internal/cli/lsp.go`. It builds `lsp.NewServer(os.Stdin, os.Stdout)`. `lsp.Server`
  (`internal/lsp/server.go:15-55`) takes an `io.Reader` and an `io.Writer`, so `server_test.go:16-55`
  drives it over pipes. The `lsp` action does not compose `reportExitError`, and learning 1 says that the
  new command must.
- **The MCP SDK.** `github.com/modelcontextprotocol/go-sdk` is already in `go.mod` as an indirect
  requirement at v1.6.1, through the `wails3` tool (`go mod why -m`). v1.8.0 needs `go 1.25.0`, so the
  `go 1.25.5` directive stays. If go.mod requires the SDK directly, the `wails3` tool moves to the same
  version. The SDK package is also named `mcp`, so `internal/mcp` imports it under an alias. Behaviour of
  the typed `AddTool` that the design depends on, checked in the v1.6.1 and v1.8.0 sources:
  - It checks the arguments against the schema that it infers from the input struct. A failed check comes
    back as an `isError` result, not as a protocol error. So a missing required `content` is an error
    result.
  - A struct schema does not allow properties that the struct does not name. So an argument that the
    tool does not know, such as `path` before US-002, is an error result.
  - A field with `omitempty` is optional.
  - The output schema must be of type `object`, so a list goes under a key.
  - When a handler sets no content, the SDK writes the JSON of the structured output into a text block.
  - `IOTransport` takes an `io.ReadCloser` and an `io.WriteCloser`. `StdioTransport` sends
    newline-delimited JSON over `os.Stdin` and `os.Stdout`.
- **The import boundaries today.** `go list -deps` shows that neither `cmd/emod-wasm` nor
  `cmd/emod-desktop` reaches `internal/cli`, and the desktop build does not reach the MCP SDK.
  `internal/llm/isolation_test.go:19-78` walks module packages with `go/build`, and records external
  packages without walking into them.
- **The AI settings.** `emod ai` reads them, and no other command does. They come from the `EMOD_AI_*`
  variables and `$HOME/.config/emod/config` (`internal/llm/config/config.go:146`). `cmd/emod` links
  `anthropic-sdk-go` through `internal/cli` → `internal/llm`. So the AC6 guard is on what `internal/mcp`
  reaches, not on what the binary links.
- **The tests.** `task test:integration` runs `go test -tags integration` over `go list -e ./...`
  without `cmd/emod-wasm` and `cmd/emod-desktop`, and with no `-count=1`. The only integration test today
  is `internal/parser/integration_test.go`. No Go test spawns `emod` itself. The precedent for a spawned
  process is the `exec.Command` of the `cue` binary in `internal/export/export_test.go:1728-1735`.
- **Text encoding.** MCP arguments come as JSON strings. When Go decodes them, it changes invalid bytes
  to U+FFFD before the handler sees them, so `content` and `name` are always valid UTF-8 at the handler.
  The server has nothing to refuse on encoding grounds, and a name or message comes back unchanged.
- **The user docs.** The README has the section "Use AI (optional)" (`README.md:296-308`). The docs
  site keeps one page for each topic under `site/content/docs/`, with a `weight` in the front matter.
  `ai.md` has weight 6, the highest.
- **Words.** The story calls each JSON object a "record", and the proposal calls the CLI type the
  "`jsonEntry` records". `diagnostic.Entry` keeps the name for the diagnostic in memory. No type in the
  repository is named `Record`. `desktop.RecentFiles.Record` is a verb on another type.

## Tasks

### Task 1: Give the diagnostic record of `--format json` one home in `internal/diagnostic`

**Behavior:** `emod validate --format json` and `emod lint --format json` build their output from a
record type that `internal/diagnostic` exports, and not from a private type in `internal/cli`. Task 2
returns the same type from `emod_check`, so the CLI and the MCP tool give the same records by
construction. The output of the CLI does not change.

**Acceptance Criteria:**
- [ ] `internal/diagnostic` exports the record of one diagnostic, with the JSON keys `file`, `line`,
  `rule`, `severity` and `message` in that order. It also turns a list of entries into a list of records.
- [ ] No entries give an empty list of records that encodes as `[]`, never as `null`.
- [ ] An entry with no rule name gives a record with no `rule` key. An entry with a rule name gives a
  record whose `rule` is that name.
- [ ] `internal/cli` declares no record type of its own. `formatJSON` builds its output from the
  `internal/diagnostic` record.
- [ ] `emod validate --format json examples/error_diagnostics_test.emod` prints two records with their
  keys in the order `file`, `line`, `rule`, `severity`, `message`, and exits 2. The first record is
  line 43, with no `rule` key and the message `event "GuestCheckedOut" does not exist`. The second
  record is line 64, with the rule `orphan-command` and the message
  `command "SendEmail" is orphaned (no flow references it)`.
- [ ] `emod validate --format json` on a clean file prints `[]` and exits 0. On the warning-only source
  of `internal/cli/validate_test.go:1112-1159`, it prints one `state-obsession` warning and exits 1.
  `emod lint --format json` prints the same records as `emod validate --format json` for each of these
  sources.

**Affected Files/Modules:**
- `internal/diagnostic/record.go` (new): the exported record type and the conversion from entries
- `internal/diagnostic/record_test.go` (new): the tests of the record
- `internal/cli/lint.go`: `formatJSON` uses the record. `jsonEntry` goes.
- `internal/cli/validate_test.go` and `internal/cli/lint_test.go`: the existing JSON leaves must pass.
  Add a leaf only where a criterion above has none.

**Patterns to Follow:**
- `internal/cli/lint.go:41-83`
- `internal/export/json.go:270-292`
- `internal/diagnostic/entry_test.go`
- `internal/cli/validate_test.go:1112-1159` for the assertions on rule names and message tokens
  (learning 2)

**Testable:** Yes. The record is an exported API that `internal/cli` calls, and the CLI's JSON leaves
show its output.

**Certainty:** high. The conversion of `formatJSON` (`internal/cli/lint.go:41-83`) moves as it is, and
`internal/export/json.go:270-292` is the precedent for "no entries gives `[]`".

**Blast radius:** high. The task moves the code that writes the output of `emod validate --format json`
and `emod lint --format json`, a CLI output format that scripts and CI read outside this repository.

**Verification:** `mise exec -- task test:unit`. Also run
`mise exec -- go run ./cmd/emod validate --format json examples/error_diagnostics_test.emod` and compare
it with the records in the criteria.

**Depends on:** None

---

### Task 2: Serve `emod_check` over MCP from `internal/mcp`

**Behavior:** A new package `internal/mcp` runs an MCP server over a reader and a writer that the caller
gives it. A client that connects completes the handshake and lists one tool, `emod_check`. When the
client calls `emod_check` with `.emod` text in `content`, and optionally a file name in `name`, it gets
the records of `oracle.Check` for that text as a normal result. The records are in the
`internal/diagnostic` shape from Task 1. The tool returns an error result only when it cannot do the
check. The MCP SDK enters `go.mod` here, and only `internal/mcp` imports it.

The tool returns the records as a list under `diagnostics` in its structured result, because MCP
structured content must be an object. The text block of the result holds the same JSON, so a host that
reads only text sees the same records. The tool's description tells the model what the list means, so
that an agent does not stop at the first warning.

**Acceptance Criteria:**
- [ ] `go.mod` requires `github.com/modelcontextprotocol/go-sdk` directly, at one pinned version, and
  the `go` directive does not change. `mise exec -- go mod tidy` leaves `go.mod` and `go.sum` unchanged.
  The `wails3` tool still builds (`mise exec -- go build -o /dev/null github.com/wailsapp/wails/v3/cmd/wails3`),
  because it shares the version of the SDK.
- [ ] A client that connects to the server over a pair of pipes completes `initialize`. `tools/list`
  then lists exactly one tool, `emod_check`. Its input schema requires `content`, a string, and accepts
  an optional `name`, a string.
- [ ] The description of `emod_check` tells the model that an empty list means the source passes the
  check. It also tells the model that a warning is a finding to fix, not a failure of the tool.
- [ ] A call with a clean source in `content` returns a result with no `isError`, and its `diagnostics`
  is an empty list, not `null`.
- [ ] A call with the text of `examples/error_diagnostics_test.emod` returns a result with no `isError`.
  It holds two error records: line 43, with no `rule` key, with a message that names
  `GuestCheckedOut`; and line 64, with the rule `orphan-command` and a message that names `SendEmail`.
- [ ] A call with a warning-only source returns a result with no `isError`, whose records are
  warnings. For example, the source of `internal/cli/validate_test.go:1112-1159` gives one
  `state-obsession` record.
- [ ] Each record carries `name` as its `file`. With no `name`, or an empty `name`, each record carries
  `inline.emod`. A `name` with non-ASCII characters, such as `café.emod`, comes back unchanged. The tool
  refuses no input on encoding grounds.
- [ ] A call with no `content` returns an `isError` result whose text names `content`, and does no
  check. An empty string in `content` is a source, the same as an empty file: it returns an empty list
  with no `isError`.
- [ ] The text block of each successful result holds the same records as its structured result.
- [ ] `internal/llm/isolation_test.go` has these new leaves, and each leaf fails when its forbidden
  import is added:
  - The walk from `internal/mcp` reaches `internal/oracle`, and reaches no `internal/llm` package and no
    provider SDK.
  - The walk from `cmd/emod-wasm` (js/wasm) and the walk from `cmd/emod-desktop` reach neither
    `internal/mcp` nor the MCP SDK.
  - No non-test package of the module except `internal/mcp` imports the MCP SDK.

**Affected Files/Modules:**
- `go.mod`, `go.sum`: the SDK becomes a direct requirement
- `internal/mcp/server.go` (new): the server, the registration of `emod_check`, the input of the tool
  and its result
- `internal/mcp/server_test.go` (new, `//go:build unit`): drives the server through an MCP client over
  pipes
- `internal/llm/isolation_test.go`: the new walks and leaves

**Patterns to Follow:**
- `internal/lsp/server.go:15-55` for the shape of a server built over an injected reader and writer
- `internal/lsp/server_test.go:16-55` for a server under test over `io.Pipe`, with cleanup
- `internal/llm/isolation_test.go:19-78` (learning 5)
- `internal/oracle/oracle.go:38-43`
- `internal/cli/validate_test.go:1112-1159` for a warning-only source and the assertion on the rule name
  (learning 2)
- `task:1` for the record type

**Testable:** Yes. The caller is an inbound MCP client. Assert on what the client sees: the tool list,
the records, and `isError`. Do not assert on how the SDK routes a call.

**Certainty:** low. The repository has never used the MCP SDK or any MCP code, and this task adds that
dependency. This task also sets the result shape of the tool: the list under `diagnostics`, and the
text block.

**Blast radius:** high. The tool name, its input schema and the shape of its result are a public API
that MCP hosts and agent prompts outside this repository read. US-002 to US-010 build on the same input
and result.

**Verification:** `mise exec -- task test:unit`. Also check each new isolation leaf once by hand: add
the forbidden import to the package that the leaf walks, see the leaf fail, then remove the import.

**Depends on:** Task 1

---

### Task 3: Start the server with `emod mcp` and check the handshake with the built binary

**Behavior:** `emod mcp` runs the server of Task 2 on stdin and stdout until the host closes the
connection. It reads no AI setting. An integration test does what a host does: it spawns the `emod`
binary that it builds from the working tree, completes the handshake, lists the tools, and calls
`emod_check`. It then compares the records with what the same binary prints for
`emod validate --format json`. This is the last code task, so one revert of this task removes `emod mcp`
from the binary.

**Acceptance Criteria:**
- [ ] `internal/cli/app.go` has an `mcp` command. Its action is one line that passes the result of
  `RunMCP` to `reportExitError`, as the other commands do (learning 1). `emod --help` lists `mcp`, with a
  usage that says that it starts the MCP server on stdin and stdout.
- [ ] An integration test (`//go:build integration`) spawns the `emod` binary, built from the working
  tree, with the argument `mcp`. Over the stdin and stdout of the process, the test completes
  `initialize`, sends `notifications/initialized`, finds `emod_check` in `tools/list`, and gets records
  from `tools/call`.
- [ ] For each `.emod` file under `examples/` and under `internal/parser/testdata/`, the test calls
  `emod_check` with the text of the file as `content` and its path as `name`. Each call returns the same
  records, field for field, that the same binary prints for `emod validate --format json <path>`. For
  each clean file, both are empty lists.
- [ ] The spawned processes have no `EMOD_AI_*` variable, no `ANTHROPIC_API_KEY`, no
  `OPENAI_API_KEY`, and a `HOME` with no emod config file, and the calls above still succeed.
- [ ] When the test closes the stdin of the server, the process ends.
- [ ] `task test:integration` runs the Go tests with `-count=1`, so a change to the binary cannot leave
  a cached pass (learning 4). The package list of the target stays on `go list -e`.

**Affected Files/Modules:**
- `internal/cli/app.go`: the `mcp` command, next to `lsp`
- `internal/cli/mcp.go` (new): `RunMCP`, which runs the `internal/mcp` server on `os.Stdin` and
  `os.Stdout`
- `cmd/emod/main_test.go` (new, `//go:build integration`): the end-to-end check of the binary
- `Taskfile.yml`: `-count=1` on `test:integration`

**Patterns to Follow:**
- `internal/cli/app.go:28-43` and `internal/cli/app.go:74-78` for an action that composes
  `reportExitError` (learning 1)
- `internal/cli/app.go:321-331` and `internal/cli/lsp.go` for a stdio command
- `internal/export/export_test.go:1728-1735` for a test that spawns a process
- `internal/parser/integration_test.go` for an integration-tagged test
- `Taskfile.yml:132` for `-count=1` on a test target
- `task:2`

**Testable:** Yes. The test is an inbound check through the binary, which is the surface that a host
starts.

**Certainty:** medium. The command follows `lsp` (`internal/cli/app.go:321-331`), and the spawned
process follows `internal/export/export_test.go:1728-1735`. The difference: no test in this repository
spawns `emod` itself, so the test must build the binary from the working tree and run uncached.

**Blast radius:** low. The task adds a command and does not change the output of any existing command.
The change to the Taskfile only stops the cache.

**Verification:** `mise exec -- task test:unit && mise exec -- task test:integration`. Also run
`printf '' | mise exec -- go run ./cmd/emod mcp` and check that it ends with nothing on stdout.

**Depends on:** Task 2

---

### Task 4: Document the registration of `emod mcp` with Claude Code

**Behavior:** A person who reads the README or the docs site finds the command that registers the
server with Claude Code. The docs also tell the person what `emod_check` takes and returns, and that
the server needs no AI setup.

**Acceptance Criteria:**
- [ ] The README has a section, next to "Use AI (optional)", that shows
  `claude mcp add emod -- emod mcp`. The section says that the server needs no API key and no
  `EMOD_AI_*` setting, and links to the docs page.
- [ ] A new page under `site/content/docs/` shows the same command. It names the tool `emod_check` and
  its inputs, `content` and `name`, with `inline.emod` as the default name. It says that the records are
  those of `emod validate --format json`, and that errors and warnings are normal results. It says that
  an empty list means that the draft passes, and that the server needs no API key, no `EMOD_AI_*`
  setting and no network access.
- [ ] Neither the README nor the page documents `path`, `--root`, `--allow-write`, `.mcp.json`, or a
  host other than Claude Code. Later stories document them.

**Affected Files/Modules:**
- `README.md`: a short section with a link
- `site/content/docs/mcp.md` (new): the page, with a `weight` after `ai.md`

**Patterns to Follow:**
- `README.md:296-308`
- `site/content/docs/ai.md`
- `site/content/docs/editor-setup.md`

**Testable:** No. It is documentation only.

**Certainty:** high. The commit "Document the AI setup for users" added the same pair: a README section
with a link (`README.md:296-308`) and a site page (`site/content/docs/ai.md`).

**Blast radius:** low. The task changes documentation only.

**Verification:** Read both texts against the criteria. Optional checks:
- `mise exec -- hugo --source site` builds the site with the new page. This needs the Hextra module.
- With the built binary on `PATH`, `claude mcp add emod -- emod mcp`, then `claude mcp list`, shows
  `emod` as connected. Run `claude mcp remove emod` after the check. This is the one check of the
  handshake with a real host in this story. US-011 owns the full host matrix.

**Depends on:** Task 3

---

## Summary

- **Total tasks:** 4. Three are code tasks, and the last task is documentation.
- **Ordering rationale:** Dependencies come first, and the revert of Task 3 stays clean:
  1. Task 1 gives the record one shape before anything else uses it. It is a refactor, and the existing
     CLI tests keep it green.
  2. Task 2 has the most risk: the new dependency and the public contract of the tool. It comes next
     and runs only in its own tests, so a problem with the SDK shows before the binary is affected.
  3. Task 3 is the last code task, so one revert takes `emod mcp` out of the binary. It holds the
     end-to-end check of the main unknown of the story: the handshake over stdio with a spawned process.
  4. Task 4 documents a command that exists.
- **Acceptance criteria coverage:**
  - AC1: Task 2 (the server lists `emod_check` in process) and Task 3 (`emod mcp` and the spawned
    handshake).
  - AC2: Task 1 (the record shape) and Task 2 (the tool returns the records). The `rule` key stays
    optional. See Boundaries.
  - AC3: Task 1 (one shape for the CLI and the tool), Task 2 (the known records and the empty list),
    and Task 3 (field-for-field parity with `emod validate --format json` for each shipped fixture).
  - AC4: Task 2.
  - AC5: Task 2.
  - AC6: Task 2 (`internal/mcp` reaches no AI package and no provider SDK) and Task 3 (the binary runs
    with no key, no `EMOD_AI_*` variable and no config file).
  - AC7: Task 4.
  - No criterion is deferred. The check against the MCP Inspector and the four hosts belongs to US-011
    (see Boundaries).

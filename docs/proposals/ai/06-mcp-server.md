# emod MCP Server

> Part of the [emod AI proposals](./README.md). Unlike the others, this proposal adds **no LLM dependency** to emod — the MCP host brings the model. See the [LLM foundation](./00-llm-foundation.md) for where that fits.

## Problem

emod already has everything an AI assistant needs to author correct event models:
a parser, a `validator`, a `linter` with explained rules, a CUE schema, and JSON /
CUE / diagram exporters. But today an assistant can only reach those capabilities
by shelling out to the `emod` binary, parsing human-formatted stdout, and guessing
exit codes — brittle glue that every user has to re-invent. Worse, an assistant
generating a `.emod` model has no in-loop way to *check its own work*: it produces
text, the user runs `emod validate`, and the error is relayed back by hand.

The other proposals (`01`–`10`) close that gap by building an LLM *into* emod —
adding the `llm.Model` port, a Bedrock-backed Claude adapter, and the
generate → validate → lint → repair loop in emod core. That is the right long-term
investment, but it adds a provider dependency, credentials, and cost to a tool
whose `go.mod` is currently tiny.

There is a cheaper, higher-leverage first move. The
[foundation doc](./00-llm-foundation.md) records the key fact:

> **MCP server (`06`) needs zero provider code.** It exposes emod's
> parse/validate/lint/export as MCP tools; the *host* (Claude Code, etc.) brings
> the model. For that whole class of use, "which provider" is not our problem.

This proposal specifies that server. It makes emod **AI-consumable** rather than
AI-powered: any MCP-capable assistant — Claude Code, Claude Desktop, Cursor,
Zed — gets emod's deterministic oracle as a set of callable tools, and the
repair loop happens *host-side*, driven by the model the host already runs.

## Goals

- A new `emod mcp` subcommand that speaks the Model Context Protocol over stdio,
  wired into `internal/cli/app.go` exactly the way `lsp` already is.
- A Go MCP server (`internal/mcp`) that maps each tool one-to-one onto an existing
  `internal/cli` `Run*` function or `internal/{export,diagram,linter,cue}` API — no
  new modeling logic, no behavioral fork from the CLI.
- Tools accept either a **file path** or **inline `.emod` text**, so an assistant
  drafting a model in chat can validate it before anything touches disk.
- Structured tool output (JSON diagnostics with `rule`, `severity`, `line`) so the
  model gets machine-readable feedback, not pretty-printed columns.
- Expose the CUE schema, the bundled examples, and the DSL reference as MCP
  **resources**, and ship a "generate an emod model" **prompt** so a fresh host
  knows the grammar without the user pasting it.
- **Zero** LLM/provider code, credentials, or network dependency added to emod.
  A read-only default so the server is safe to register globally.

## Non-Goals

- Building, importing, or calling `llm.Model` — that is proposals `01`/`02`/`03`+.
  This server never talks to a model; it only answers a model's tool calls.
- Implementing the repair loop *inside* emod. The loop lives in the host's agent
  turn (see [How It Works](#how-it-works)); emod just provides the oracle each
  iteration calls.
- HTTP/SSE transport or a hosted multi-tenant server. Stdio, launched per session
  by the host, mirrors `emod lsp` and is enough for local authoring.
- Re-implementing CLI flag parsing. The MCP tool schemas are the contract; they
  delegate to the same `Run*`/export functions the CLI uses.

---

## How It Works

### MCP server over stdio

MCP is JSON-RPC 2.0 (the same wire protocol the LSP server in `internal/lsp`
already frames in `transport.go`). An MCP host launches `emod mcp` as a subprocess
and exchanges three kinds of capability:

- **Tools** — functions the model can call (`emod_validate`, `emod_lint`, …).
- **Resources** — read-only content the host can pull into context (the CUE
  schema, example models, the DSL reference).
- **Prompts** — parameterized templates the user can invoke (e.g. "scaffold an
  emod model for <domain>").

Rather than hand-roll the JSON-RPC plumbing (as `internal/lsp` does for the LSP
spec), this server uses the **official Go MCP SDK**,
`github.com/modelcontextprotocol/go-sdk`. It owns the protocol envelope, the
`initialize` handshake, tool/resource/prompt registration, and the stdio
transport, leaving us to register handlers that call existing emod internals. This
is the only new dependency, and it is a protocol library — not a provider SDK.

```go
package mcp

import (
	"context"
	"os"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Serve runs the emod MCP server over stdio until the client disconnects.
// It is the analogue of lsp.Server.Run — protocol on stdin/stdout, emod
// capabilities behind it.
func Serve(ctx context.Context) error {
	server := mcpsdk.NewServer(&mcpsdk.Implementation{
		Name:    "emod",
		Version: buildVersion,
	}, nil)

	registerTools(server)     // emod_validate, emod_lint, emod_fmt, ...
	registerResources(server) // emod://schema, emod://examples/*, emod://docs/*
	registerPrompts(server)   // generate-model, review-model

	return server.Run(ctx, mcpsdk.NewStdioTransport())
}
```

### Mapping tools to existing internals

Every tool is a thin adapter. The pattern matches what `internal/cli/validate.go`,
`lint.go`, `export.go`, etc. already do: read source (from a path *or* inline
text), run `lexer.Scan` → `parser.New(...).Parse()` → `validator.Validate` →
`linter.Lint`, then format. The difference is the result is returned as a
structured MCP tool result instead of being printed and exit-coded.

To avoid duplicating the read-and-pipeline logic across both the CLI and the MCP
server, a small shared helper is the natural refactor — the diagnostics shape is
already standardized as `[]*diagnostic.Entry` (`internal/diagnostic/entry.go`,
fields `Filename`, `Line`, `Column`, `Message`, `Severity`, `RuleName`):

```go
// resolveSource returns .emod text from either inline content or a file path,
// honoring the configured root for path scoping (see Security).
func resolveSource(in toolInput) (source string, name string, err error)

// runPipeline parses, validates, and lints — the same sequence RunValidate uses —
// and returns the model plus diagnostics for any tool to format.
func runPipeline(source, name string) (*ast.Model, []*diagnostic.Entry)
```

A representative handler — `emod_validate` — reuses that helper and serializes the
existing `jsonEntry` shape from `internal/cli/lint.go`:

```go
func handleValidate(ctx context.Context, in validateInput) (*mcpsdk.CallToolResult, error) {
	source, name, err := resolveSource(in.source)
	if err != nil {
		return toolError(err), nil
	}
	_, diags := runPipeline(source, name)
	return jsonResult(toEntries(diags)), nil // {file,line,rule,severity,message}[]
}
```

`emod_export`, `emod_diagram`, `emod_slices`, and `emod_schema` follow the same
shape, delegating to `export.ExportJSONDiagnostics` / `export.ExportCUE`,
`diagram.ExportMermaid` / `ExportASCII` / `ExportSVG`, `cli.collectSliceEntries`,
and the embedded `cue.Schema` respectively. `emod_explain_rule` is a direct
passthrough to `linter.RuleDescription` (the same call behind `emod lint
--explain`). Nothing here is new modeling code; it is the CLI's body re-presented
as MCP.

### Resources and prompts

**Resources** let the host pull authoritative context without the user pasting it:

- `emod://schema` → the bundled `cue.Schema` (`internal/cue/embed.go`). The model
  reads the grammar instead of guessing it.
- `emod://examples/{all_patterns,dcb_model,...}` → the files under `examples/`
  (`all_patterns.emod`, `dcb_model.emod`, `inbound_customer_comms_agentic_reply.emod`),
  served as few-shot exemplars.
- `emod://docs/dsl-reference` and `emod://docs/dcb` → `docs/dsl-reference.md` and
  `docs/dcb-proposal.md`, so the assistant has the prose spec for DCB tags,
  `decides_on`, and context `mode`.

Resources are read-only by construction, which keeps the default server safe.

**Prompts** are reusable, host-surfaced templates. A `generate-model` prompt
assembles a system message that (a) references `emod://schema` and a chosen
example resource, and (b) instructs the model to call `emod_validate` and
`emod_lint` until clean. A `review-model` prompt points the model at an existing
file and asks it to run `emod_lint` and explain each finding via
`emod_explain_rule`. The prompt ships the workflow so users do not have to know it.

### The host-side repair loop

This is the crux of the contrast with proposal `01`. The
[foundation's repair loop](./00-llm-foundation.md#the-repair-loop-provider-agnostic-core)
runs *inside* emod, calling `llm.Model.Generate` and `pipeline.Check` in a Go
`for` loop. The MCP server inverts that: emod provides only `pipeline.Check`
(as `emod_validate`/`emod_lint` tools), and the *loop* is the host's agent turn.

```
proposal 01 (loop in emod):           proposal 06 (loop in host):

  emod ── Generate ──▶ llm.Model        host model ── tool_call ──▶ emod_validate
   ▲                       │             (in host's                      │
   └── pipeline.Check ◀────┘              agent loop)  ◀── diagnostics ──┘
   (Go for-loop in emod core)            (the model re-drafts and calls again)
```

The model emits a `.emod` draft, calls `emod_validate` with the inline text, reads
the JSON diagnostics, fixes them, and calls again — exactly the generate → validate
→ lint → repair loop, but with emod as a *tool* and the host's model as the
*driver*. emod stays deterministic and provider-free; the host supplies whatever
model it runs.

This composes for free with the user's existing agents and skills: any skill that
can call MCP tools (a "model this domain" skill, a domain-design reviewer, an
implement loop) gains emod's self-check the moment `emod mcp` is registered — no
per-skill integration, no emod-side prompt engineering.

## Tool Surface

All tools accept `path` **or** `content` (inline `.emod` text); `content` wins if
both are set, so an assistant can validate an unsaved draft. Output is JSON unless
noted. "Side effect" marks whether a tool writes to disk (see
[Security](#risks--mitigations)).

| MCP tool | Maps to | Input | Output | Side effect |
|----------|---------|-------|--------|-------------|
| `emod_validate` | `lexer.Scan` → `parser` → `validator.Validate` → `linter.Lint` (body of `RunValidate`) | `{path? , content?}` | `[]{file,line,rule,severity,message}` | none (read-only) |
| `emod_lint` | `linter.Lint` (body of `RunLint`) | `{path?, content?}` | `[]{file,line,rule,severity,message}` | none |
| `emod_explain_rule` | `linter.RuleDescription` (`descriptions.go`; behind `lint --explain`) | `{rule}` e.g. `"state-obsession"` | `{rule, description}` | none |
| `emod_fmt` | `formatter.Format` (body of `RunFmt`) | `{path?, content?, write?}` | `{formatted, changed}`; writes file only if `write:true` **and** path given | optional write |
| `emod_export` | `export.ExportJSONDiagnostics` / `export.ExportCUE` (body of `RunExport`) | `{path?, content?, format:"json"\|"cue"}` | JSON object or CUE text + diagnostics | none |
| `emod_diagram` | `diagram.ExportMermaid` / `ExportASCII` / `ExportSVG` (body of `RunDiagram`) | `{path?, content?, format:"mermaid"\|"svg"\|"ascii", style?}` | diagram text (mermaid/svg/ascii string) | none |
| `emod_slices` | `cli.collectSliceEntries` + `detectPattern` (body of `RunSlices`) | `{path?, content?}` | `[]{name,pattern,context,keyElements}` | none |
| `emod_schema` | embedded `cue.Schema` (`RunSchema`) | `{}` | CUE schema text | none |

Notes on fidelity to existing behavior:

- `emod_validate` runs the **full** pipeline (validator + linter), matching
  `RunValidate`; `emod_lint` runs only `linter.Lint` after a clean parse, matching
  `RunLint`. Both return the same `{file,line,rule,severity,message}` records the
  CLI's `--format json` already emits.
- `emod_diagram` deliberately omits `drawio` file output and the `--serve` viewer —
  those write files / start servers. It returns the inline-renderable formats
  (`mermaid`, `ascii`, `svg`) the model can embed or reason over. `style` accepts
  the same `auto|projected|dcb` values as `diagram.ParseStyle`.
- Exit codes do not cross the MCP boundary. Severity is conveyed *in* the result
  (`severity: "error"|"warning"|"info"`), so the model branches on data, not a
  process exit. A tool reports an `isError` MCP result only for genuine failures
  (unreadable path, malformed input), never for model-level diagnostics — those
  are expected output the model is meant to act on.

## Interface

### The `emod mcp` command

Wired into the command tree in `internal/cli/app.go` immediately after `lsp`,
which it mirrors:

```go
{
	Name:  "mcp",
	Usage: "Start the MCP server (stdin/stdout transport)",
	Flags: []urfave.Flag{
		&urfave.StringFlag{
			Name:  "root",
			Usage: "Restrict file-path tools to this directory (default: cwd)",
		},
		&urfave.BoolFlag{
			Name:  "allow-write",
			Usage: "Enable tools that write to disk (emod_fmt --write)",
		},
	},
	Action: func(c *urfave.Context) error {
		if err := RunMCP(c.String("root"), c.Bool("allow-write")); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return urfave.Exit("", 1)
		}
		return nil
	},
},
```

With the thin `cli` entry point alongside `RunLSP`:

```go
// internal/cli/mcp.go
func RunMCP(root string, allowWrite bool) error {
	return mcp.Serve(context.Background(), mcp.Options{Root: root, AllowWrite: allowWrite})
}
```

### Registering the server with a host

**Claude Code** (CLI) — one command registers the local binary:

```sh
claude mcp add emod -- emod mcp
```

or, equivalently, in the project's `.mcp.json` (checked in, shared with the team):

```json
{
  "mcpServers": {
    "emod": {
      "command": "emod",
      "args": ["mcp", "--root", "."]
    }
  }
}
```

**Claude Desktop / Cursor / Zed** — the same shape in the host's MCP config
(`claude_desktop_config.json` and equivalents):

```json
{
  "mcpServers": {
    "emod": {
      "command": "/usr/local/bin/emod",
      "args": ["mcp"]
    }
  }
}
```

No API key, no `EMOD_AI_*` environment variable, no provider configuration — the
server has no model to authenticate to. Registration is just "run this binary."

## Worked Example

A user, in Claude Code with the `emod` MCP server registered, asks:

> "Model a slice where a customer places an order and the system records it."

The assistant first reads the grammar from a resource, drafts a model, and
self-corrects against the oracle — all within one agent turn, no human relay:

```
assistant → reads resource  emod://schema
assistant → reads resource  emod://examples/all_patterns.emod

assistant (draft 1) → calls emod_validate
  content: |
    model "Ordering"
    actor Customer
    context Orders {
      aggregate Order { stream "Order-{orderId}" }
      slice "Place Order" {
        trigger UI "Order Form" { actor Customer }
        command PlaceOrder { aggregate Order
          fields { orderId OrderID required } }
        event OrderUpdated { aggregate Order
          fields { orderId OrderID } }
        command -> event: PlaceOrder -> OrderUpdated
      }
    }

emod_validate → [
  { "line": 9, "rule": "state-obsession", "severity": "warning",
    "message": "event \"OrderUpdated\" uses a generic state-change name" },
  { "line": 9, "rule": "clickbait-event", "severity": "warning",
    "message": "event \"OrderUpdated\" carries only an identifier" }
]

assistant → calls emod_explain_rule { "rule": "state-obsession" }
emod_explain_rule → "Events named with generic state-change suffixes like
  \"Updated\"... Prefer names that describe a specific business fact
  (e.g. \"OrderShipped\" instead of \"OrderUpdated\")."

assistant (draft 2) → calls emod_validate
  # renames OrderUpdated → OrderPlaced and adds domain fields
  event OrderPlaced { aggregate Order
    fields { orderId OrderID  customerId CustomerID  total Money  placedAt Timestamp } }
  command -> event: PlaceOrder -> OrderPlaced

emod_validate → []      # clean

assistant → calls emod_diagram { "content": "...", "format": "mermaid" }
emod_diagram → "flowchart LR ..."   # renders the slice for the user
```

The model converged in two iterations because the deterministic findings
(`state-obsession`, `clickbait-event`) told it *exactly* what to fix, and
`emod_explain_rule` gave it the rationale and the fix shape. emod ran no model;
Claude Code's model drove the loop and called emod as a tool. The user only sees
the final, validated `.emod` and its diagram.

## Implementation Plan

**Phase 1 — Core read-only server (S).**
Add `github.com/modelcontextprotocol/go-sdk`. Create `internal/mcp` with `Serve`
over stdio and the `mcp` subcommand in `app.go` + `internal/cli/mcp.go`. Implement
the read-only tools that already have JSON-shaped output: `emod_validate`,
`emod_lint`, `emod_explain_rule`, `emod_schema`, `emod_slices`. Refactor the
shared read+pipeline path so the CLI and MCP server call one helper rather than
diverging. Ships the highest-value capability (the self-check loop) on its own.

**Phase 2 — Export, diagram, format, resources (M).**
Add `emod_export` (json/cue), `emod_diagram` (mermaid/svg/ascii), and `emod_fmt`
(read-only `formatted`/`changed`; `write` gated behind `--allow-write`). Register
resources: `emod://schema`, `emod://examples/*`, `emod://docs/*`. Add path scoping
(`--root`) and the read-only/write tool split.

**Phase 3 — Prompts and packaging (M).**
Add the `generate-model` and `review-model` prompts. Document registration for
Claude Code, Claude Desktop, Cursor, and Zed. Provide a checked-in `.mcp.json`
for the repo so contributors get the tools automatically. Consider an
`mcp --list` self-describe for debugging.

**Phase 4 — Hardening and conformance (L, optional).**
Run against the MCP Inspector and the reference clients; add transport tests
mirroring `internal/lsp/transport_test.go`; consider an opt-in HTTP/SSE transport
only if a non-local host needs it.

## Risks & Mitigations

- **Write tools are dangerous by default.** Only `emod_fmt --write` mutates disk.
  Mitigation: writes are off unless `--allow-write` is passed *and* a `path`
  (not inline `content`) is given; everything else is read-only. The server is
  safe to register globally in its default mode.
- **Path traversal via the `path` input.** A model could pass `../../etc/...`.
  Mitigation: `--root` (default cwd) scopes every path tool; paths are resolved
  and rejected if they escape the root. Inline `content` needs no filesystem
  access at all and is the preferred input for drafts.
- **Behavioral drift between CLI and MCP.** Two code paths could diverge.
  Mitigation: both call the same `runPipeline` helper and the same
  `export`/`diagram`/`linter`/`cue` APIs; the MCP layer only re-formats results.
  Tests assert MCP tool output matches the CLI's `--format json` for the same
  input.
- **New dependency surface.** `go-sdk` is a real addition to a deliberately small
  `go.mod`. Mitigation: it is a protocol library (no network clients, no provider
  SDKs), isolated to `internal/mcp`; the WASM build (`cmd/emod-wasm`) excludes the
  package, so the browser bundle is unaffected.
- **SDK churn.** The MCP Go SDK and spec are young and evolving. Mitigation: keep
  all SDK contact inside `internal/mcp` behind our handlers; pin the version; the
  tool/resource/prompt *contract* is stable even if the SDK surface shifts.
- **Diagnostics misread as failures.** A host might treat model-level warnings as
  tool errors and abort the loop. Mitigation: diagnostics are normal successful
  results carrying `severity`; `isError` is reserved for I/O and parse-input
  failures. The prompt templates spell out that warnings are signal to repair, not
  to stop.

## Open Questions

- **Which SDK, exactly?** This proposes `github.com/modelcontextprotocol/go-sdk`
  (the official Go SDK). If its API or maintenance status is unsuitable at
  implementation time, the fallback is to frame MCP JSON-RPC directly on top of
  the `internal/lsp/transport.go` machinery, which already does Content-Length
  framing — at the cost of owning the protocol envelope ourselves.
- **Should `emod_diagram` ever return `drawio`?** drawio is XML the model can't
  render inline but the user might want saved. Return it as text, or keep diagram
  file output strictly CLI-only? Current proposal: text formats only over MCP.
- **One `emod_check` super-tool vs. separate `validate`/`lint`?** A single tool is
  fewer round-trips for the model; separate tools match the CLI and let a host
  call just the linter. Current proposal keeps them separate for CLI parity.
- **Resource granularity for examples.** Expose each example as its own resource,
  or one `emod://examples` index the model lists then fetches? Indexing scales as
  `examples/` grows.
- **Versioning the schema resource.** Should `emod://schema` carry the emod
  version so a model caching it across sessions knows when the grammar changed?
- **Prompt ownership.** Prompts encode a workflow opinion. Do they belong in emod
  (shipped, consistent) or in the user's host config (customizable per team)?
  Shipping a default that hosts can override is the likely answer.

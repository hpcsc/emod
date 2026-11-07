# emod MCP Server

> Part of the [emod AI proposals](./README.md). Phase 1. This proposal adds **no LLM dependency** to emod:
> the MCP host brings the model. See the [AI foundation](./00-llm-foundation.md) for where this fits.

## Problem

emod has the parts that an AI assistant needs to write a correct event model: a parser, a validator, a
linter with an explanation for each rule, a CUE schema, exporters, and diagrams. But an assistant reaches
them only through the `emod` binary. It must parse output that is formatted for people, and it must guess
the meaning of exit codes. Each user writes this glue again.

An assistant that drafts a `.emod` model also has no way to check its own draft before the draft is on
disk. The assistant writes text, the user runs `emod validate`, and the user copies the errors back by
hand.

Many facilitators and developers already work with an agent such as Claude Code or Cursor. For them, the
fastest path to AI help is not a new AI command in emod. It is to give their agent the check of emod as a
tool. The agent then drafts, checks and repairs a model with the model that the host already runs. emod
needs no provider, no credentials and no cost for this.

## Goals

- **G1:** A new `emod mcp` command that speaks the Model Context Protocol over stdio, in the same way as `emod lsp`.
- **G2:** An MCP server in `internal/mcp`. Each tool calls the same code as a CLI command. The server adds no
  logic about models and no behaviour that the CLI does not have.
- **G3:** Each tool accepts a **file path** or **inline `.emod` text**. An assistant can check a draft before it
  writes the draft to disk.
- **G4:** Structured results: diagnostics as JSON records with `rule`, `severity` and `line`, not columns of text.
- **G5:** Exact query tools ([exact model queries](../model-queries-proposal.md)), so that the agent answers
  questions from facts, not from a guess (moved to `docs/proposals/model-queries-proposal.md`)
- **G6:** Resources for the schema, the examples, the rules and the DSL reference, so that a new host knows the
  grammar with no text pasted by the user.
- **G7:** The `model-domain` prompt for the capture step of the workflow in the [README](./README.md)
  (moved to `docs/proposals/ai/04-model-from-conversation.md`)
- **G8:** **No** LLM code, provider code, credentials or network access in emod. Read-only by default, so that a
  user can register the server for all projects.
- **G9:** Spec export ([05](./05-spec-suggestions.md)), so that the agent answers questions from facts, not from a
  guess (moved to `docs/proposals/ai/05-spec-suggestions.md`)
- **G10:** The `review-model` prompt for the challenge step (moved to `docs/proposals/ai/02-semantic-model-review.md`)
- **G11:** The `import-from-code` prompt for the import step (moved to `docs/proposals/ai/07-model-import.md`)
- **G12:** The `suggest-specs` prompt for the specify step (moved to `docs/proposals/ai/05-spec-suggestions.md`)

## Non-Goals

- A call to `llm.Model`. This server never calls a model. It only answers the tool calls of a model.
- The repair loop of the [foundation](./00-llm-foundation.md) inside emod. The loop runs in the agent turn
  of the host. emod gives the check that each step of the loop calls.
- A tool that writes `.emod` content to disk. The host writes files with its own file tools, under the
  permissions that the user gives the host.
- An HTTP or SSE transport, or a hosted server for many users. Stdio is enough for local work, as it is for
  `emod lsp`.
- The flag parser of the CLI. The MCP tool schemas are the contract.

## How It Works

### MCP server over stdio

MCP is JSON-RPC 2.0. An MCP host starts `emod mcp` as a subprocess and gets three kinds of capability:

- **Tools:** functions that the model can call, such as `emod_check` and `emod_trace`.
- **Resources:** read-only content that the host can put into its context, such as the CUE schema and the
  examples.
- **Prompts:** templates that the user starts by name, such as `model-domain`.

The server uses the official Go MCP SDK, `github.com/modelcontextprotocol/go-sdk`. The SDK owns the
protocol envelope, the `initialize` handshake, the registration of tools, resources and prompts, and the
stdio transport. emod only registers handlers. The SDK is the only new dependency, and it is a protocol
library, not a provider SDK.

```go
package mcp

// Serve runs the emod MCP server over stdio until the client disconnects.
func Serve(ctx context.Context, options Options) error {
	server := mcpsdk.NewServer(&mcpsdk.Implementation{Name: "emod", Version: version.Current()}, nil)

	registerTools(server, options)
	registerResources(server)
	registerPrompts(server)

	return server.Run(ctx, mcpsdk.NewStdioTransport())
}
```

### Tools call the code that the CLI calls

Each tool is a thin adapter. It reads the source from a path or from inline text, calls the same function
as the CLI command, and returns the result as a structured MCP result. It does not print, and it does not
set an exit code.

The CLI already routes each command through `internal/oracle`:

- `RunValidate` and `RunLint` call `oracle.Check`, which runs the parser, the validator and the linter.
  Both commands report the same diagnostics.
- `RunExport` and `RunDiagram` call `oracle.Run`, which also returns the model.
- `RunFmt`, `RunSlicesArrange`, `RunSlicesList` and `RunGlossary` call `oracle.Parse`, and stop on a parse
  error.

The MCP handlers call the same three functions. One helper resolves the source:

```go
// sourceOf returns the .emod text and its file name, from inline content or
// from a path under the root (see Risks and Mitigations).
func sourceOf(input sourceInput, root string) (source string, filename string, err error)
```

The diagnostics have one shape everywhere: `[]*diagnostic.Entry`. The tools return the `jsonEntry` records
that `emod validate --format json` prints today: `file`, `line`, `rule`, `severity` and `message`.

### Resources

Resources let the host read authoritative context with no text pasted by the user:

| Resource | Content | Source |
| --- | --- | --- |
| `emod://schema` | The bundled CUE schema | `cue.Schema` |
| `emod://rules` | Each lint rule with its description | `linter.RuleDescription` for each rule name |
| `emod://examples/all_patterns` | All four slice patterns, invariants and specs | `examples/all_patterns.emod` |
| `emod://examples/dcb_model` | A DCB context with tags and `decides_on` | `examples/dcb_model.emod` |
| `emod://examples/specs_hotel` | Given-When-Then specs with example payloads | `examples/specs_hotel.emod` |
| `emod://docs/dsl-reference` | The prose reference of the DSL | `docs/dsl-reference.md` |

`go:embed` cannot read a file outside the directory of its package. Therefore `examples/embed.go` (package
`examples`) embeds the three example files, and `docs/embed.go` (package `docs`) embeds the DSL reference.
`internal/linter` gets an exported `Rules()` function that lists the rule names, so that `emod://rules`
does not copy the map. `examples/error_diagnostics_test.emod` is a test fixture, so no resource serves it.

### Prompts

A prompt gives the host a workflow, so that the user does not have to know it. Each prompt matches one
step of the workflow in the [README](./README.md):

| Prompt | Arguments | Workflow step | What it tells the host model to do |
| --- | --- | --- | --- |
| `model-domain` | `description`, `output` | Capture ([04](./04-model-from-conversation.md)) | Read `emod://schema`, `emod://rules` and one example. Ask the user questions about actors, outcomes, boundaries, order, external systems and failures, with lettered options, before any draft. List the assumptions. Then draft, call `emod_check` until it reports nothing, and write the file. |
| `review-model` | `path` | Challenge ([02](./02-semantic-model-review.md)) | Call `emod_check` and `emod_explain_rule` for each finding. Then read the model and give semantic findings in the taxonomy of 02, each with a location, a direction and a confidence. Do not edit the file until the user accepts a finding. |
| `import-from-code` | `path`, `output` | Capture from code ([07](./07-model-import.md)) | Find the event, command, handler and projection definitions with the search tools of the host. Make an inventory with `file:line` for each item. Draft the model with a `DRAFT` header, call `emod_check` until clean, and list each item that has no source. |
| `suggest-specs` | `path`, `slice` | Specify ([05](./05-spec-suggestions.md)) | Read the slice, its invariants and the result of `emod_references` for its events. Suggest the specs that the slice does not have: one rejection spec for each invariant, and one spec for each branch. Call `emod_check` on the result, show a diff, and write only after the user accepts. |

The prompts state that a `warning` is a signal to repair, not a reason to stop. A host can override a
prompt. emod ships a default so that each team starts from the same workflow.

### The repair loop runs in the host

The [foundation](./00-llm-foundation.md) runs the repair loop inside emod: `repair.Generate` calls
`llm.Model.Generate` and `oracle.Check` in a Go loop. The MCP server turns this around. emod gives only the
check, as the `emod_check` tool. The loop is the agent turn of the host.

```
repair.Generate (loop in emod):          MCP server (loop in the host):

  emod ── Generate ──▶ llm.Model           host model ── tool call ──▶ emod_check
   ▲                       │                (agent turn                    │
   └──── oracle.Check ◀────┘                 of the host) ◀── diagnostics ──┘
   (a Go loop in emod)                      (the model drafts again and calls again)
```

The model writes a draft, calls `emod_check` with the inline text, reads the JSON diagnostics, fixes them,
and calls again. emod stays deterministic and has no provider. Any skill or agent that can call MCP tools
gets the check of emod when the user registers `emod mcp`. The skill needs no change.

### The workflow with the desktop app

The MCP server and the desktop app together give the capture and shape steps of the workflow, with no AI
code in the desktop app:

1. The facilitator opens the `.emod` file in the desktop app and shares the screen with the group.
2. The facilitator asks the agent for a change, for example "add a refund path for a cancelled stay".
3. The agent drafts the change, calls `emod_check` until it is clean, and writes the file with its own file
   tool.
4. The desktop app sees the change. `desktop.OpenFileWatch` checks the open file on disk and tells the page
   about a change. The page compares the bytes with the bytes that it holds.
5. If the window has no unsaved edits, the page puts the new text in the source panel and redraws the
   diagram in place. Pan, zoom and the position of each node stay. The status bar shows
   `↻ <file> reloaded: it changed on disk`.
6. If the window has unsaved edits, the app asks the user which version to keep. The agent does not
   overwrite the work of a person without a question.

The group sees each change on the diagram a moment after the agent writes it. If a write leaves a file that
does not parse, the page keeps the last diagram on screen and marks it out of date. The page does not
reload an empty file. The web viewer has no file watch, so the desktop app is the surface for this workflow.

## Tool Surface

All tools accept `path` **or** `content`. If both are set, `content` wins. A tool checks inline content under
the name in `name`, or under `inline.emod` when the call gives no `name`. Output is JSON unless the table says other.

| MCP tool | Calls | Input | Output | Writes to disk |
| --- | --- | --- | --- | --- |
| `emod_check` | `oracle.Check`, the body of `RunValidate` and `RunLint` | `{path?, content?, name?}` | `[]{file, line, rule, severity, message}` | No |
| `emod_explain_rule` | `linter.RuleDescription`, the body of `RunLintExplain` | `{rule}`, for example `"state-obsession"` | `{rule, description}` | No |
| `emod_fmt` | `oracle.Parse`, then `formatter.Format`, the body of `RunFmt` | `{path?, content?, write?}` | `{formatted, changed}` | Only with `write: true`, a `path`, and `--allow-write` |
| `emod_slices_arrange` | `oracle.Parse`, `arrange.Model`, then `formatter.Format`, the body of `RunSlicesArrange` | `{path?, content?, write?}` | `{arranged, changed, moved, backwardBefore, backwardAfter, backward: []{kind, label, from, to}}` | Only with `write: true`, a `path`, and `--allow-write` |
| `emod_slices_list` | `oracle.Parse`, `ast.Model.SliceRefs`, `detectPattern` and `keyElementsForPattern`, the body of `RunSlicesList` | `{path?, content?}` | `[]{name, pattern, context, keyElements}` | No |
| `emod_export` | `oracle.Run`, then `export.ExportJSONDiagnostics`, `export.ExportDiagramJSONDiagnostics` or `export.ExportCUE`, the body of `RunExport` | `{path?, content?, format: "json" \| "diagram-json" \| "cue"}` | The export and its diagnostics | No |
| `emod_diagram` | `oracle.Run`, then the `diagram.Export*` function for the format, the body of `RunDiagram` | `{path?, content?, format, style?, specs?}` | The diagram as text | No |
| `emod_glossary` | `oracle.Parse`, then `glossary.RenderMarkdown` or `glossary.RenderJSON`, the body of `RunGlossary` | `{path?, content?, format: "markdown" \| "json"}` | The glossary | No |
| `emod_references` | `query.References` ([exact model queries](../model-queries-proposal.md)) | `{path?, content?, name}` | The declaration and each reference, grouped by role | No |
| `emod_trace` | `query.Trace` ([exact model queries](../model-queries-proposal.md)) | `{path?, content?, from, to?, upstream?, withViews?}` | The paths, each as a list of steps | No |
| `emod_specs_export` | The function behind `emod specs export` ([05](./05-spec-suggestions.md)) | `{path?, content?, format?, slice?}` | The exported specs | No |
| `emod_schema` | `cue.Schema`, the body of `RunSchema` | `{}` | The CUE schema as text | No |

Notes on the match with the CLI:

- **One check tool.** `emod validate` and `emod lint` both call `oracle.Check` and report the same
  diagnostics. Therefore one tool, `emod_check`, serves both. Two tools with the same result make the model
  call twice for nothing.
- **Diagnostics are data, not errors.** Exit codes do not cross the MCP boundary. Each record carries its
  `severity`. A tool returns an `isError` result only for a real failure, such as a path that it cannot
  read or a path outside the root.
- **Formats of `emod_diagram`.** `format` takes the values of `diagramFormats` in `internal/cli/diagram.go`:
  `svg`, `mermaid`, `ascii`, `event-flow`, `event-flow-mermaid` and `event-flow-mermaid-no-icons`. `style`
  takes the values of `diagram.ParseStyle`: `auto`, `projected` and `dcb`. `specs` draws the
  Given-When-Then cards, and only `svg` accepts it, as in the CLI. The tool returns the SVG text and writes
  no file. It has no `--serve` mode, because that mode starts a server.
- **A parse error stops some tools.** `emod_diagram` returns the diagnostics and no diagram when the check
  reports an error, as `RunDiagram` does. `emod_fmt`, `emod_slices_arrange`, `emod_slices_list` and
  `emod_glossary` return the parse diagnostics and stop, as their commands do.
- **Lifted helpers.** `detectPattern` and `keyElementsForPattern` are unexported functions in
  `internal/cli`. They classify a slice, so they move next to `ast.Model.SliceRefs`. Only the column
  layout of the text output stays in the CLI.
- **The arrange report is the useful half.** `emod_slices_arrange` returns `arrange.Report` with the
  rewritten text. The references that still point backward are the ones that no order can fix. A model
  that sees them stops a useless attempt. The read-only default is the `--check` mode of the CLI, and
  `changed` carries the answer that `--check` gives through its exit code.

## Interface

### The `emod mcp` command

`internal/cli/app.go` gets an `mcp` command next to `lsp`:

```go
{
	Name:  "mcp",
	Usage: "Start the MCP server (stdin/stdout transport)",
	Flags: []urfave.Flag{
		&urfave.StringFlag{Name: "root", Usage: "Restrict path inputs to this directory (default: the working directory)"},
		&urfave.BoolFlag{Name: "allow-write", Usage: "Let emod_fmt and emod_slices_arrange write to disk"},
	},
	Action: func(ctx context.Context, cmd *urfave.Command) error {
		return reportExitError(RunMCP(ctx, cmd.String("root"), cmd.Bool("allow-write")))
	},
},
```

```go
// internal/cli/mcp.go
func RunMCP(ctx context.Context, root string, allowWrite bool) error {
	return mcp.Serve(ctx, mcp.Options{Root: root, AllowWrite: allowWrite})
}
```

### Register the server with a host

**Claude Code:** one command registers the local binary:

```sh
claude mcp add emod -- emod mcp
```

A project can also check in a `.mcp.json`, so that each person on the team gets the server:

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

**Claude Desktop, Cursor and Zed:** the same shape in the MCP configuration of the host.

The server needs no API key, no `EMOD_AI_*` setting and no provider. It has no model to connect to. The
registration only tells the host which binary to start.

## Worked Example

A facilitator works in Claude Code with the `emod` server registered, and the desktop app has
`ordering.emod` open. The facilitator asks:

> "Model a slice where a customer places an order and the system records it."

The host model reads the grammar, drafts, and corrects the draft against the check in one agent turn:

```
assistant → reads resource  emod://schema
assistant → reads resource  emod://examples/all_patterns

assistant (draft 1) → calls emod_check { "name": "ordering.emod", "content": <draft 1> }
```

```emod
emod = 1

model "Ordering" {
}

actor "Customer" {
}

context "Orders" {
  aggregate "Order" {
    slice "Place Order" {
      trigger "Order Form" {
        actor = Customer
      }

      command "PlaceOrder" {
        fields {
          orderId = required(string)
        }
      }

      event "OrderUpdated" {
        fields {
          orderId = required(string)
        }
      }

      flow = <<-FLOW
        command -> event:    PlaceOrder -> OrderUpdated
      FLOW
    }
  }
}
```

```
emod_check → [
  { "file": "ordering.emod", "line": 22, "rule": "state-obsession", "severity": "warning",
    "message": "event \"OrderUpdated\" uses a generic state-change suffix; prefer a name that describes a specific business fact" },
  { "file": "ordering.emod", "line": 22, "rule": "clickbait-event", "severity": "warning",
    "message": "event \"OrderUpdated\" has a single ID field \"orderId\"; consider adding domain-relevant fields or inlining the identifier" }
]

assistant → calls emod_explain_rule { "rule": "state-obsession" }
emod_explain_rule → { "rule": "state-obsession", "description": "Events named with generic state-change
  suffixes like \"Updated\", \"Changed\", or \"Modified\" hide the business meaning ... (e.g. \"OrderShipped\"
  instead of \"OrderUpdated\")." }

assistant (draft 2) → calls emod_check { "name": "ordering.emod", "content": <draft 2> }
```

```emod
emod = 1

model "Ordering" {
}

actor "Customer" {
}

context "Orders" {
  aggregate "Order" {
    slice "Place Order" {
      trigger "Order Form" {
        actor = Customer
      }

      command "PlaceOrder" {
        fields {
          orderId    = required(string)
          customerId = required(string)
          total      = required(decimal)
        }
      }

      event "OrderPlaced" {
        fields {
          orderId    = required(string)
          customerId = required(string)
          total      = required(decimal)
          placedAt   = required(timestamp)
        }
      }

      flow = <<-FLOW
        command -> event:    PlaceOrder -> OrderPlaced
      FLOW
    }
  }
}
```

```
emod_check → []

assistant → writes ordering.emod with the file tool of the host
desktop app → ↻ ordering.emod reloaded: it changed on disk
```

The draft converged in two steps, because the findings `state-obsession` and `clickbait-event` told the
model what to fix, and `emod_explain_rule` gave the reason and the shape of the fix. emod ran no model.
The host model ran the loop and called emod as a tool. The group saw only the checked model, on the diagram
in the desktop app.

## Implementation Plan

**Phase 1: the read-only core (S).** Add `github.com/modelcontextprotocol/go-sdk`. Create `internal/mcp`
with `Serve` over stdio, and the `mcp` command in `internal/cli/app.go` and `internal/cli/mcp.go`. Add the
tools that have JSON output already: `emod_check`, `emod_explain_rule`, `emod_schema` and
`emod_slices_list`. Move `detectPattern` and `keyElementsForPattern` out of `internal/cli`. This phase
gives the self-check loop on its own.

**Phase 2: export, diagram, glossary, format and resources (M).** Add `emod_export`, `emod_diagram`,
`emod_glossary`, and the two tools that rewrite a model, `emod_fmt` and `emod_slices_arrange`. Add the
`--allow-write` gate and the `--root` scope. Add the `examples` and `docs` embed packages, `linter.Rules()`,
and the resources.

**Phase 3: prompts and registration (M).** Add the four prompts. Document the registration for Claude Code,
Claude Desktop, Cursor and Zed. Check in a `.mcp.json` for this repository.

**Phase 4: query and spec tools (S each).** Add `emod_references` and `emod_trace` when
The [model queries proposal](../model-queries-proposal.md) ships `internal/query`. Add `emod_specs_export` when
[05](./05-spec-suggestions.md) ships `emod specs export`. Each tool is one handler over a function that
the CLI already calls.

**Phase 5: conformance (M, optional).** Run the server against the MCP Inspector and the reference
clients. Add transport tests in the style of `internal/lsp/transport_test.go`.

Tests compare the result of each tool with the `--format json` output of the CLI command for the
same input, so that the two paths cannot give different answers.

## Risks and Mitigations

- **A tool that writes is dangerous by default.** Only `emod_fmt` and `emod_slices_arrange` can write to
  disk. *Mitigation:* they write only when the user starts the server with `--allow-write` and the call
  gives a `path` and `write: true`. All other tools are read-only. In its default mode, the server is safe
  to register for all projects.
- **A path outside the project.** A model can pass `../../etc/passwd` as `path`. *Mitigation:* `--root`
  (default: the working directory) limits each path. The server resolves each path and rejects a path
  outside the root. Inline `content` needs no file access, and the prompts prefer it for drafts.
- **The CLI and the MCP server give different answers.** *Mitigation:* both call the same `oracle`,
  `export`, `diagram`, `glossary`, `arrange` and `query` functions. The MCP layer only formats the result.
  Tests compare the two outputs.
- **A new dependency.** `go-sdk` adds to a small `go.mod`. *Mitigation:* it is a protocol library with no
  network clients and no provider SDK. Only `internal/mcp` imports it. The WASM build and the desktop app
  do not import `internal/mcp`.
- **The SDK and the protocol change often.** *Mitigation:* only `internal/mcp` touches the SDK. emod pins
  the version. The contract of the tools, the resources and the prompts stays stable when the SDK changes.
- **A host reads a warning as a failure.** A host can stop the loop on the first warning. *Mitigation:*
  diagnostics are normal results with a `severity`. `isError` is only for input and file failures. The
  prompts state that a warning is a signal to repair.
- **A prompt hides a decision from the group.** The `model-domain` prompt can lead to a full draft with no
  discussion. *Mitigation:* the prompt tells the model to ask questions before any draft and to list the
  assumptions. The person decides when the draft is ready.

## Open Questions

- **The SDK.** If `github.com/modelcontextprotocol/go-sdk` is not suitable at implementation time, the
  server can frame JSON-RPC on the machinery in `internal/lsp/transport.go`. Then emod owns the protocol
  envelope.
- **Resource granularity for examples.** One resource for each example works for three files. An index
  resource that the model lists, then reads, scales better when `examples/` grows.
- **The version of the schema resource.** `emod://schema` can carry the emod version, so that a host that
  keeps the schema across sessions knows when the grammar changes.
- **Prompt ownership.** A prompt states an opinion about the workflow. emod ships defaults, and a host can
  override them. A team can want to keep its own prompts in its repository.
- **A file watch for the served viewer.** `emod diagram --serve` can also watch the file and send a change
  to the page. Then a user without the desktop app gets the same workflow.

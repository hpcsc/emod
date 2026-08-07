# AI: emod MCP Server

## Overview

emod already owns everything an AI assistant needs to author correct event models: a parser, a validator, a linter with explained rules, a CUE schema, and JSON / CUE / diagram exporters. But today an assistant can only reach those capabilities by shelling out to the `emod` binary, parsing human-formatted stdout, and guessing exit codes — brittle glue every user re-invents. Worse, an assistant drafting a `.emod` model has no in-loop way to check its own work: it emits text, a human runs `emod validate`, and the error is relayed back by hand.

This feature exposes emod's existing deterministic capabilities to any MCP-capable assistant (Claude Code, Claude Desktop, Cursor, Zed) through a new `emod mcp` command that speaks the Model Context Protocol over a stdio transport. Each emod capability — validate, lint, explain-rule, fmt, export, diagram, slices, schema — becomes a callable MCP **tool**; the CUE schema, bundled examples, and DSL reference become MCP **resources**; and reusable workflows ship as MCP **prompts**. The assistant generates a model, calls the validate/lint tools, reads the structured diagnostics, and converges — the generate → validate → lint → repair loop runs entirely **host-side**, driven by the model the host already runs.

This makes emod **AI-consumable** rather than **AI-powered**. It adds **no LLM provider code, no credentials, and no network dependency** to emod: the MCP host brings the model; emod only answers the model's tool calls. The server defaults to read-only so it is safe to register globally.

## Goals

- Provide an `emod mcp` command that serves emod's capabilities over the Model Context Protocol (stdio transport), launched per session by an MCP host.
- Expose each existing emod capability as a one-to-one MCP tool whose behaviour matches the equivalent CLI command (validate, lint, explain-rule, fmt, export json/cue, diagram mermaid/svg/ascii, slices, schema).
- Let every model-input tool accept either a file path or inline `.emod` text, so an assistant can check an unsaved draft before anything touches disk.
- Return structured, machine-readable tool output (JSON diagnostics carrying `rule`, `severity`, `line`) rather than pretty-printed columns, so the model branches on data.
- Expose the CUE schema, bundled example models, and DSL reference as MCP resources, and ship prompts that encode the authoring and review workflows.
- Keep the server safe by default: read-only unless a write flag is set, with file-path access scoped to a configured root directory.
- Add no LLM/provider dependency, credential, or network requirement to emod.

## User Stories

### US-MCP-001: Start the MCP server over stdio

**Description:** As an MCP-host user, I want to launch the emod MCP server with a single command so that an MCP host can connect to emod's capabilities over a stdio transport.

**Acceptance Criteria:**
- [ ] Running `emod mcp` starts a server that speaks the Model Context Protocol over stdin/stdout and runs until the client disconnects
- [ ] The server completes the MCP initialize handshake and reports its name (`emod`) and version to the connecting client
- [ ] The server advertises its supported capabilities (tools, resources, prompts) during initialization
- [ ] An MCP client that lists tools receives the set of registered emod tools with their names, descriptions, and input schemas
- [ ] Starting the server requires no API key, provider configuration, or `EMOD_AI_*` environment variable
- [ ] Protocol messages are exchanged over stdin/stdout only; no network port is opened

**Context:** This mirrors the existing `emod lsp` subcommand, which also speaks a JSON-RPC protocol over stdin/stdout and is launched as a subprocess by its host. MCP exchanges three kinds of capability: tools (functions the model calls), resources (read-only content the host pulls into context), and prompts (parameterized templates the user invokes).

---

### US-MCP-002: Validate a model via an MCP tool

**Description:** As an AI assistant, I want to call a validate tool with a model so that I receive structured diagnostics I can act on without parsing human-formatted terminal output.

**Acceptance Criteria:**
- [ ] A tool (e.g. `emod_validate`) accepts either a file `path` or inline `content` (`.emod` text); when both are supplied, `content` is used
- [ ] The tool runs the full pipeline (parse, then validate, then lint), matching what `emod validate` produces for the same input
- [ ] The result is a JSON array of diagnostic records, each carrying `file`, `line`, `rule`, `severity`, and `message`
- [ ] A model with no problems returns an empty array
- [ ] `severity` is conveyed in the result (`error`, `warning`, or `info`); the model branches on this data rather than on a process exit code
- [ ] Model-level diagnostics are returned as a normal successful result, not as a tool error
- [ ] A genuine failure (unreadable path, input that cannot be lexed/parsed at all) is reported as an MCP tool error distinct from model-level diagnostics

**Context:** The CLI's `--format json` output already emits the same `{file,line,rule,severity,message}` records. Exit codes do not cross the MCP boundary, so severity must travel inside the result. Diagnostics are expected output the model is meant to repair, not a signal to abort.

**Depends on:** US-MCP-001

---

### US-MCP-003: Lint a model via an MCP tool

**Description:** As an AI assistant, I want to call a lint-only tool so that I can get just the linter's findings on a model that already parses cleanly.

**Acceptance Criteria:**
- [ ] A tool (e.g. `emod_lint`) accepts either a file `path` or inline `content`; `content` wins when both are present
- [ ] The tool runs only the linter after a clean parse, matching what `emod lint` produces for the same input
- [ ] The result is a JSON array of diagnostic records carrying `file`, `line`, `rule`, `severity`, and `message`
- [ ] A model with no lint findings returns an empty array
- [ ] The same model and input produce the same records as the CLI's `emod lint --format json`

**Context:** Validate and lint are kept as separate tools to mirror the CLI and let a host call just the linter. `emod_validate` runs validator plus linter; `emod_lint` runs only the linter.

**Depends on:** US-MCP-001

---

### US-MCP-004: Explain a lint rule via an MCP tool

**Description:** As an AI assistant, I want to ask for the explanation of a named lint rule so that I learn the rationale and the fix shape for a finding before I re-draft.

**Acceptance Criteria:**
- [ ] A tool (e.g. `emod_explain_rule`) accepts a `rule` name (e.g. `state-obsession`)
- [ ] The tool returns the rule's description text, matching what `emod lint --explain` produces for that rule
- [ ] The result includes the rule name and its description
- [ ] An unknown rule name returns a clear error indicating the rule is not recognized
- [ ] The tool needs no model input and performs no file access

**Context:** This is a direct passthrough to the same rule-description lookup behind `emod lint --explain`. Pairing it with the validate/lint tools lets an assistant turn a finding code into actionable guidance within a single agent turn.

**Depends on:** US-MCP-001

---

### US-MCP-005: Retrieve the slices of a model via an MCP tool

**Description:** As an AI assistant, I want to retrieve the slices detected in a model so that I can reason about its structure and patterns without re-deriving them.

**Acceptance Criteria:**
- [ ] A tool (e.g. `emod_slices_list`) accepts either a file `path` or inline `content`; `content` wins when both are present
- [ ] The result is a JSON array of slice records, each carrying the slice `name`, detected `pattern`, `context`, and key elements
- [ ] The output matches what `emod slices list` produces for the same input
- [ ] A model with no slices returns an empty array
- [ ] Input that cannot be parsed is reported as a tool error, not as an empty slice list

**Depends on:** US-MCP-001

---

### US-MCP-006: Retrieve the CUE schema via an MCP tool

**Description:** As an AI assistant, I want to fetch the emod CUE schema as a tool result so that I can ground a draft in the grammar without the user pasting it.

**Acceptance Criteria:**
- [ ] A tool (e.g. `emod_schema`) takes no model input and returns the bundled CUE schema as text
- [ ] The returned schema matches what `emod schema` produces
- [ ] The tool performs no file access and never writes to disk

**Context:** The schema is the same embedded CUE definition the CLI serves. Exposing it as a tool (in addition to a resource, see US-MCP-011) lets a model that cannot list resources still pull the grammar on demand.

**Depends on:** US-MCP-001

---

### US-MCP-007: Self-correct a generated model host-side using the validate and lint tools

**Description:** As an AI assistant generating a model, I want to validate my own draft against emod and repair it iteratively so that the user only ever sees a clean, validated model.

**Acceptance Criteria:**
- [ ] The assistant can submit a `.emod` draft as inline `content` to the validate tool and receive structured diagnostics without writing any file
- [ ] When diagnostics are returned, the assistant can resolve a finding's rationale via the explain-rule tool and submit a revised draft for re-validation
- [ ] Repeated validate calls reflect the changes in each successive draft (fewer or different diagnostics as the draft improves)
- [ ] A draft that has been repaired to clean returns an empty diagnostics array, signalling convergence
- [ ] The entire loop runs within the host's agent turn using only emod tool calls — emod itself invokes no model and holds no loop state
- [ ] Warnings are treated as signal to repair, not as a reason to abort the loop

**Context:** This is the crux of the design. The repair loop that proposal `01` runs *inside* emod is inverted here: emod provides only the oracle (the validate/lint tools), and the loop is the host's agent turn. Any host skill that can call MCP tools gains emod's self-check the moment the server is registered, with no per-skill integration.

**Depends on:** US-MCP-002, US-MCP-004

---

### US-MCP-008: Export a model via an MCP tool

**Description:** As an AI assistant, I want to export a model to JSON or CUE so that I can hand structured model data to the user or to a downstream tool.

**Acceptance Criteria:**
- [ ] A tool (e.g. `emod_export`) accepts either a file `path` or inline `content`, plus a `format` of `json` or `cue`
- [ ] With `format: json`, the result is a JSON object representing the model, matching what `emod export --format json` produces
- [ ] With `format: cue`, the result is CUE text matching what `emod export --format cue` produces
- [ ] The result also surfaces any diagnostics produced while reading the model
- [ ] The tool never writes to disk
- [ ] An unsupported `format` value returns a clear error listing the accepted values

**Depends on:** US-MCP-001

---

### US-MCP-009: Render a diagram via an MCP tool

**Description:** As an AI assistant, I want to render a model as a Mermaid, SVG, or ASCII diagram so that I can present or reason over the model visually.

**Acceptance Criteria:**
- [ ] A tool (e.g. `emod_diagram`) accepts either a file `path` or inline `content`, plus a `format` of `mermaid`, `svg`, or `ascii`
- [ ] The result is the diagram text for the requested format, matching what `emod diagram` produces for that format
- [ ] An optional `style` accepts the same values as the CLI (`auto`, `projected`, `dcb`)
- [ ] The tool does not produce drawio file output and does not start the diagram viewer/server
- [ ] The tool never writes to disk
- [ ] An unsupported `format` or `style` value returns a clear error listing the accepted values

**Context:** Only the inline-renderable formats (`mermaid`, `ascii`, `svg`) are exposed — formats a model can embed or reason over. The drawio output and the `--serve` viewer are intentionally excluded because they write files / start servers and are not useful to an in-context model.

**Depends on:** US-MCP-001

---

### US-MCP-010: Format a model via a read-only MCP tool

**Description:** As an AI assistant, I want to format a model and see the formatted text without modifying any file so that I can show the user a clean version of their draft safely.

**Acceptance Criteria:**
- [ ] A tool (e.g. `emod_fmt`) accepts either a file `path` or inline `content`
- [ ] By default the tool returns the formatted text and a flag indicating whether formatting changed the input, and writes nothing to disk
- [ ] The formatted output matches what `emod fmt` produces for the same input
- [ ] Comments are preserved in the formatted output
- [ ] When the server is in its default (read-only) mode, the tool never writes a file regardless of any write request

**Context:** Formatting is one of the two capabilities that can mutate disk, the other being arranging (US-MCP-018). This story covers only the read-only behaviour; the gated write behaviour is US-MCP-014. Keeping the default read-only is what makes the server safe to register globally.

**Depends on:** US-MCP-001

---

### US-MCP-011: Expose the schema, examples, and docs as MCP resources

**Description:** As an AI assistant, I want to pull the CUE schema, example models, and DSL reference as MCP resources so that I have authoritative context for authoring without the user pasting it.

**Acceptance Criteria:**
- [ ] The server registers a resource for the CUE schema (e.g. `emod://schema`) whose content matches the bundled schema
- [ ] The server registers a resource per bundled example model (e.g. `emod://examples/all_patterns`, `emod://examples/dcb_model`) whose content matches the corresponding file under `examples/`
- [ ] The server registers resources for the DSL reference and the DCB reference (e.g. `emod://docs/dsl-reference`, `emod://docs/dcb`) whose content matches the corresponding docs files
- [ ] An MCP client that lists resources receives all registered resources with their URIs and descriptions
- [ ] Reading any resource returns its current content
- [ ] All resources are read-only; no resource operation writes to disk

**Context:** Resources let the host pull emod's grammar, few-shot exemplars, and prose spec (covering DCB tags, `decides_on`, and context `mode`) into the model's context. Being read-only by construction keeps the default server safe.

**Depends on:** US-MCP-001

---

### US-MCP-012: Ship authoring and review prompts

**Description:** As an MCP-host user, I want ready-made prompts for generating and reviewing emod models so that I and the assistant follow the right workflow without me having to describe it.

**Acceptance Criteria:**
- [ ] The server registers a generate-model prompt that the host can list and invoke with a domain argument
- [ ] The generate-model prompt directs the model to reference the schema and example resources and to call the validate and lint tools until the model is clean
- [ ] The server registers a review-model prompt that the host can invoke against an existing model
- [ ] The review-model prompt directs the model to call the lint tool and explain each finding via the explain-rule tool
- [ ] An MCP client that lists prompts receives both prompts with their names, descriptions, and accepted arguments
- [ ] The prompts state that warnings are signal to repair, not a reason to stop

**Context:** Prompts ship the workflow so users do not have to know it, and so any host gets consistent behaviour. They reference the resources from US-MCP-011 and the tools from US-MCP-002 through US-MCP-004.

**Depends on:** US-MCP-007, US-MCP-011

---

### US-MCP-013: Scope file-path tools to a root directory

**Description:** As an MCP-host user, I want the server's file-path access confined to a directory I choose so that a model cannot read files outside the project.

**Acceptance Criteria:**
- [ ] `emod mcp` accepts a `--root` option that restricts every file-path tool to that directory; the default root is the current working directory
- [ ] A tool given a `path` inside the root resolves and reads that file as expected
- [ ] A tool given a `path` that resolves outside the root (e.g. via `../`) is rejected with a clear error and no file is read
- [ ] Tools called with inline `content` instead of a `path` require no filesystem access and are unaffected by the root setting
- [ ] The root restriction applies to read tools and to any write tool equally

**Context:** A model could pass a traversal path such as `../../etc/...`. Scoping to a root (default cwd) and resolving/rejecting paths that escape it contains this. Inline `content` is the preferred input for drafts because it needs no filesystem access at all.

**Depends on:** US-MCP-002

---

### US-MCP-014: Gate disk writes behind an explicit opt-in

**Description:** As an MCP-host user, I want disk-writing behaviour disabled by default and enabled only by an explicit flag so that registering the server globally cannot let a model modify my files unexpectedly.

**Acceptance Criteria:**
- [ ] In the default server mode, no tool writes to disk under any input
- [ ] `emod mcp` accepts an `--allow-write` option that enables write behaviour for the format and arrange tools
- [ ] With `--allow-write` set, a writing tool writes its result to a file only when given a file `path` (not inline `content`) and an explicit write request
- [ ] With `--allow-write` set but called with inline `content`, a writing tool returns its result text and writes nothing
- [ ] A write attempt is still subject to the root scoping from US-MCP-013
- [ ] All tools other than the format and arrange tools remain read-only regardless of the `--allow-write` setting

**Context:** Formatting and arranging are the only capabilities exposed over MCP that can mutate disk. Defaulting writes off — and requiring both the flag and a path — is what lets the server be registered globally in its safe default mode.

**Depends on:** US-MCP-010, US-MCP-013, US-MCP-018

---

### US-MCP-015: Register the emod server with an MCP host

**Description:** As an MCP-host user, I want clear, copy-pasteable instructions to register the emod server so that my assistant can use emod's tools without bespoke setup.

**Acceptance Criteria:**
- [ ] Documentation shows registering the server with Claude Code via a single add command pointing at `emod mcp`
- [ ] Documentation shows the equivalent project-level MCP configuration entry (command `emod`, args including `mcp`) that can be checked in and shared with a team
- [ ] Documentation shows the equivalent configuration for Claude Desktop, Cursor, and Zed
- [ ] After registration, the assistant can list and call the emod tools and read the emod resources in a session
- [ ] The documentation states that registration requires no API key, environment variable, or provider configuration
- [ ] A checked-in project MCP configuration file is provided so contributors get the tools automatically

**Context:** Registration is just "run this binary" — the server has no model to authenticate to. A checked-in project config gives every contributor the tools without per-person setup.

**Depends on:** US-MCP-001

---

### US-MCP-016: Tool output matches the CLI for the same input

**Description:** As an MCP-host user, I want the MCP tools to behave identically to the equivalent CLI commands so that I can trust the assistant's results as much as my own terminal runs.

**Acceptance Criteria:**
- [ ] For the same model, `emod_validate` returns the same diagnostic records as `emod validate --format json`
- [ ] For the same model, `emod_lint` returns the same diagnostic records as `emod lint --format json`
- [ ] For the same model and format, `emod_export` and `emod_diagram` return the same content as the corresponding CLI commands
- [ ] For the same model, `emod_slices_list` returns the same slice records as `emod slices list`
- [ ] For the same rule, `emod_explain_rule` returns the same description as `emod lint --explain`

**Context:** Both surfaces share the same read-and-pipeline path and the same export/diagram/linter/schema capabilities; the MCP layer only re-presents the result. This story is the guard against the two surfaces drifting apart.

**Depends on:** US-MCP-002, US-MCP-003, US-MCP-005, US-MCP-008, US-MCP-009

---

### US-MCP-017: Self-describe the server for debugging

**Description:** As an MCP-host user, I want to inspect the server's tools, resources, and prompts from the command line so that I can debug a registration or capability problem without an MCP client.

**Acceptance Criteria:**
- [ ] A command (e.g. `emod mcp --list`) prints the registered tools, resources, and prompts without starting the stdio server loop
- [ ] Each listed tool shows its name and a one-line description
- [ ] Each listed resource shows its URI; each listed prompt shows its name and accepted arguments
- [ ] The command exits without waiting for client input

**Depends on:** US-MCP-001

---

### US-MCP-018: Arrange a model's slices via a read-only MCP tool

**Description:** As an AI assistant, I want to reorder a model's slices so its references read forward, and to see the ones still pointing backward, so that I can hand back a model that reads in order without guessing which of its arrows were fixable.

**Acceptance Criteria:**
- [ ] A tool (e.g. `emod_slices_arrange`) accepts either a file `path` or inline `content`; `content` wins when both are present
- [ ] By default the tool returns the arranged text and a flag indicating whether any slice moved, and writes nothing to disk
- [ ] The arranged output matches what `emod slices arrange` produces for the same input
- [ ] The result reports the references still pointing backward, each naming its kind (`subscribes`, `reads`, `on` or `flow`), the reference itself, and the two slices it runs between
- [ ] A model whose slices are already arranged reports nothing moved and returns its input unchanged
- [ ] A comment written above a slice stays with that slice wherever it lands
- [ ] When the server is in its default (read-only) mode, the tool never writes a file regardless of any write request

**Context:** Arranging moves view slices only — the process slices keep their authored order — so the tool reorders a model without rewriting the story it tells. The backward references it reports are the ones no ordering can remove: two slices producing one event leaves the second pointing back at the declaration whichever way round they go. Reporting them is what stops a model arranging the same file repeatedly trying to reach zero. As with formatting (US-MCP-010), this story covers the read-only behaviour only; the gated write is US-MCP-014.

**Depends on:** US-MCP-001

---

## Non-Goals (Out of Scope)

- **No LLM provider code in emod.** The server never builds, imports, or calls a language model; it never authenticates to a model, holds credentials, or makes a network call to a provider. The host brings the model. (This is the defining contrast with proposals `01`/`02`/`03`+.)
- **No repair loop inside emod.** The generate → validate → lint → repair loop lives in the host's agent turn; emod only provides the oracle each iteration calls.
- **No HTTP/SSE or multi-tenant hosted transport.** Stdio, launched per session by the host, is the only transport in scope (an opt-in HTTP/SSE transport is deferred until a non-local host needs it).
- **No drawio file output or diagram viewer over MCP.** Only the inline-renderable diagram formats (`mermaid`, `ascii`, `svg`) are exposed; file-writing and server-starting diagram modes stay CLI-only.
- **No new modeling logic.** Every tool delegates to an existing emod capability; the MCP layer re-presents results and adds no new validator rules, lint rules, or export formats.
- **No re-implementation of CLI flag parsing.** The MCP tool input schemas are the contract; they delegate to the same capabilities the CLI uses.
- **No effect on the WASM/browser build.** The MCP capability is excluded from the browser bundle.

## Open Questions

Clarifying questions were not raised; the following assumptions were made from the source proposal and are open to revision.

- **Tool naming.** Stories use the proposal's `emod_*` tool names (`emod_validate`, `emod_lint`, etc.) as illustrative; final names are negotiable as long as each maps one-to-one to its CLI capability.
- **Combined vs. separate validate/lint.** Assumed separate `validate` and `lint` tools for CLI parity (US-MCP-002, US-MCP-003), rather than a single `emod_check` super-tool. A combined tool would mean fewer round-trips for the model — revisit if round-trip cost proves significant.
- **Example resource granularity.** Assumed one resource per example model (US-MCP-011). An alternative is a single index resource the model lists then fetches, which scales better as the example set grows.
- **Schema versioning.** Open whether the schema resource should carry the emod version so a model caching it across sessions knows when the grammar changed.
- **Prompt ownership.** Assumed prompts ship with emod for consistency (US-MCP-012), with hosts free to override them. Whether the canonical home is emod or per-team host config is unresolved.
- **drawio over MCP.** Assumed drawio is not exposed (US-MCP-009); revisit if users want the XML returned as text for saving even though a model cannot render it inline.
- **Workspace-level operations.** Whether any tool should operate over all `.emod` files in the root, rather than a single path/content per call, is out of scope for now but may be requested.

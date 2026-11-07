# AI Foundation: the Model Port, the Providers and the Repair Loop

> Part of the [emod AI proposals](../README.md). Each feature proposal uses the structure in this document
> and does not specify it again.

## Problem

The AI features of emod all call a language model, turn its answer into something emod can use, and check
that answer against the rules of emod. If each feature does this in its own way, each feature gets its own
client, its own retry logic and its own tie to one vendor. No test of that code can run without a network.

The users of emod also use different providers. One team has an Anthropic API key. Another team can only
use Azure OpenAI or Gemini. A team with confidential domains runs a local model with Ollama, so that the
model of their business does not leave the machine. A foundation that speaks only to one vendor excludes
most of these users, and most of the people who use the desktop app.

This document records the decisions: one small port that emod owns, two protocol adapters behind it, one
configuration, and one repair loop. The deterministic check of emod, not the model, is the engine.

## Goals

- **G1:** One interface, `llm.Model`, that every AI feature depends on.
- **G2:** Access to Claude through the Anthropic API, with an API key.
- **G3:** Access to any provider that speaks the OpenAI-compatible Chat Completions API: OpenAI, Azure OpenAI,
  Gemini, OpenRouter, LiteLLM, Ollama, LM Studio and vLLM.
- **G4:** No framework and no new heavy dependency.
- **G5:** The repair loop lives in emod core and depends only on the interface and on `oracle.Check`.
- **G6:** Each AI feature has unit tests against a fake model, with no network.
- **G7:** AI stays opt-in. With no provider configured, every command works as it does with no AI.

## Non-Goals

- A general agent framework such as `langchaingo`.
- One adapter for each vendor. emod speaks two protocols, and each protocol reaches many vendors.
- Provider code or credentials in the WASM build.
- AI in the static web viewer.
- A promise of equal quality on every model. See [Support tiers](#support-tiers).
- A provider for each cloud platform of Claude, such as AWS Bedrock or Google Vertex AI. Each platform adds
  its own credentials and settings. A user of such a platform sets `EMOD_AI_ENDPOINT` to a gateway, such as
  LiteLLM, that speaks the Messages API or the OpenAI-compatible API.

## The key point: emod has its own check

Most AI applications have no way to know if an answer is correct. emod has one: the parser, the validator,
the linter and the bundled CUE schema. `internal/oracle` puts them behind one function:

```go
// internal/oracle/oracle.go
func Check(source string, filename string) []*diagnostic.Entry
```

When the answer of a model is `.emod` source, emod checks it and sends the diagnostics back to the model
until the source is clean. The model does not have to be correct in one step. It has to converge against a
check that emod owns. This loop does not depend on the provider. The provider is a leaf of the design, not
its core.

The check proves that a model is valid and idiomatic. It does not prove that the model is true for the
domain. A person decides that.

## Where AI runs

| Surface | What calls the model | Whose credentials |
| --- | --- | --- |
| MCP server (`emod mcp`) | The MCP host, for example Claude Code or Cursor. emod only answers tool calls. | The credentials of the host |
| CLI (`emod ai ...`) | The `emod` process | The emod configuration |
| Desktop app | The Go process of the desktop app, in the same process as the pipeline | The emod configuration, which the settings dialog of the app writes |
| LSP (`emod lsp`) | The `emod lsp` process, only after the user picks an AI action | The emod configuration |
| Static web viewer | Nothing. The viewer has no AI. | None |
| Viewer from `emod diagram --serve` | Nothing. See [Open questions](#open-questions). | None |

The desktop app does not need a backend for AI. Its Go process holds `llm.Model` directly, as the CLI does.
The static web viewer runs in a browser, which cannot hold a key safely, so it has no AI.

## Decision 1: a thin port that emod owns

emod core depends on an interface that it owns, never on an SDK:

```go
// internal/llm/model.go
type Effort int // EffortUnset, EffortLow, EffortMedium, EffortHigh, EffortXHigh

type Message struct {
	Role    string // "user" or "assistant"
	Content string
}

type GenerateRequest struct {
	System   string
	Messages []Message
	Schema   json.RawMessage // optional JSON Schema; when set, the answer must conform
	Effort   Effort
}

type Usage struct {
	InputTokens  int
	OutputTokens int
}

type Response struct {
	Text  string
	Usage Usage
}

type Model interface {
	Generate(ctx context.Context, request GenerateRequest) (Response, error)
}
```

The port gives two things. Each feature has tests against a fake `Model`, and the adapter can change with
no change to the features. `internal/llm` also holds two test doubles: `Fake` returns canned responses and
records the requests, and `Broken` returns an error.

## Decision 2: two protocols, not one adapter for each vendor

Most providers speak one of two HTTP protocols. emod has one adapter for each protocol:

| Protocol | Package | Providers it reaches |
| --- | --- | --- |
| Anthropic Messages API | `internal/llm/anthropic` | The Anthropic API with an API key; a proxy that speaks the Messages API |
| OpenAI-compatible Chat Completions API | `internal/llm/openai` | OpenAI, Azure OpenAI, Gemini through its OpenAI-compatible endpoint, OpenRouter, LiteLLM, Ollama, LM Studio, vLLM |

The `anthropic` adapter uses `anthropic-sdk-go`. It passes `option.WithoutEnvironmentDefaults()` to the
client, so the SDK reads no credential and no base URL from the environment. Without it, the SDK reads
`ANTHROPIC_AUTH_TOKEN`, a profile of `ant auth login` and `ANTHROPIC_BASE_URL`, and a call can then use a
credential or a host that the settings of emod do not name. emod resolves the API key and the endpoint
itself, as [Configuration](#configuration) describes, and gives both to the client.

The `openai` adapter uses `net/http` and no SDK. emod needs one call, `POST /chat/completions`, with
messages, an optional `response_format` and an optional `reasoning_effort`. An SDK adds many dependencies
and ties the adapter to the features of one vendor, but the adapter must work with many servers.

- The adapter sends the key as a bearer token. With no key, it sends no `Authorization` header, because a
  local server needs none.
- It sends no limit on the output tokens. Servers name that field in different ways, such as `max_tokens`
  and `max_completion_tokens`, and a wrong name gives an HTTP 400.
- A fallback runs only after an HTTP 400 whose message names the field. A 400 for another reason, such as
  a prompt that is too long, ends the call with that error.

### Why two protocols

- **One protocol is not enough.** The Anthropic protocol reaches only Claude. Users of Azure OpenAI, Gemini
  and local models cannot use emod AI without the second protocol.
- **One adapter for each vendor is too much.** The OpenAI-compatible protocol already reaches most vendors
  and all popular local servers. A vendor adapter adds code that the compatible endpoint makes
  unnecessary.
- **The second protocol is not a gateway requirement.** A user can put LiteLLM in front of any vendor and
  point emod at it, but a user of Ollama or OpenAI does not need a gateway at all.
- **Claude keeps its own adapter.** The Messages API gives adaptive thinking, effort, and structured output
  with a schema. emod uses these features on its tested path.

### Structured output

Some features need data that conforms to a schema: a list of findings, a list of suggestions. The
`Schema` field of `GenerateRequest` carries the JSON Schema.

| Adapter | How it asks for the schema | If the server does not support it |
| --- | --- | --- |
| `anthropic` | `output_config.format` with the schema | Not applicable. The Anthropic API supports it. |
| `openai` | `response_format` of type `json_schema`, with `strict: true` | After an HTTP 400 that names `response_format` or `json_schema`, the adapter puts the schema in the system prompt and tries again one time. It takes the JSON out of a fenced block before the check. |

Both adapters check the answer against the schema locally before they return it, with the shared check in
`internal/llm/schema`. An answer that does not conform is an error, and the feature decides to retry or to
stop. Therefore a weak server cannot give a feature data of the wrong shape.

### Effort

| Adapter | How it sends the effort |
| --- | --- |
| `anthropic` | Adaptive thinking and `output_config.effort`. emod does not set `budget_tokens` or sampling parameters. |
| `openai` | `reasoning_effort`: `low`, `medium` or `high`. `xhigh` becomes `high`. After an HTTP 400 that names the field, the adapter removes it and tries again one time. |

### Support tiers

The check of emod catches structural errors from any model. It does not catch a weak judgement. A semantic
review from a small local model can be noisy, and a suggested name can be poor. emod states this:

| Tier | Models | What emod promises |
| --- | --- | --- |
| Tested | Claude models through the `anthropic` provider | The evaluation corpus of [02](../02-semantic-model-review.md) and [04](../04-model-from-conversation.md) runs on them. emod sets their defaults. |
| Compatible | Any model behind the `openai` provider that supports JSON Schema output | The adapter works, and the check applies to all `.emod` output. emod does not measure the quality of the judgement. |

When the configured model is not in the tested tier, the CLI prints one notice for each run, and the
desktop panel shows the notice in its header.

## Decision 3: no framework

`tmc/langchaingo` and similar frameworks promise to swap any provider. emod does not use them. They lag the
features of the providers, their abstractions leak, and they are large dependencies for a CLI. The port in
Decision 1 is about 40 lines, and the two adapters are small. A third protocol gets a third adapter behind
the same port, when a user needs it.

## The repair loop

The repair loop lives in `internal/llm/repair` and depends only on `llm.Model` and `oracle.Check`:

```go
// internal/llm/repair/repair.go
type Outcome struct {
	Source      string
	Converged   bool
	Attempts    int
	Diagnostics []*diagnostic.Entry
	Usage       llm.Usage
}

func Generate(ctx context.Context, model llm.Model, request llm.GenerateRequest, maxAttempts int) (Outcome, error)
```

1. The loop asks the model for `.emod` source.
2. It takes the source from the answer. If the answer has a fenced block, the loop uses the block.
3. It runs `oracle.Check` on the source.
4. If the check reports nothing, the loop returns the source with `Converged` set.
5. If the check reports diagnostics, the loop adds the answer and the diagnostics to the conversation and
   asks again.
6. After `maxAttempts`, the loop returns the last source and its diagnostics with `Converged` not set.

The outcome carries the usage of every call, also when a call fails. A feature never shows a source that
did not converge as a success. It shows the diagnostics that the last attempt left.

`repair.WithProgress(func(repair.Attempt))` reports each attempt when its check ends: the number of the
attempt, `maxAttempts`, the diagnostics, and the usage of that attempt. It reports nothing for a model call
that fails. `emod ai` prints one line for each attempt, and the desktop panel of
[03](../03-desktop-ai-panel.md) sends the same data as the `ai:progress` event.

### Options for edits

The loop above suits a new model, which must have no diagnostic. An edit starts from a model that can
already have findings, for example an `info` finding of a `spec/*` rule. With the loop above, such an edit
never converges. The features that edit a model add three things to `internal/llm/repair`:

| Addition | What it does | Added by |
| --- | --- | --- |
| `repair.WithBaseline(diagnostics)` | The loop converges when the result adds no diagnostic to the baseline. It sends back only the diagnostics that the result adds. | [03](../03-desktop-ai-panel.md) |
| `Outcome.Answer` | The text of the last answer, so that a feature can read the notes of the model outside the fenced source | [03](../03-desktop-ai-panel.md) |
| `repair.WithAssemble(func(answer string) (string, error))` | The loop turns the answer into the whole file before the check. A feature that asks for structured data renders the file from it. | [05](../05-spec-suggestions.md) |

Proposals [03](../03-desktop-ai-panel.md), [04](../04-model-from-conversation.md) and
[05](../05-spec-suggestions.md) use this loop. The MCP server does not: there, the host model
runs the loop, and emod gives the check as a tool.

## Configuration

One configuration serves the CLI, the LSP and the desktop app:

| Key | Flag | Used by | Meaning |
| --- | --- | --- | --- |
| `EMOD_AI_PROVIDER` | `--ai-provider` | All | `anthropic` or `openai`. When it is not set, the provider is `anthropic`. |
| `EMOD_AI_API_KEY` | none | `anthropic`, `openai` | The API key. When it is not set, emod reads `ANTHROPIC_API_KEY` for `anthropic` or `OPENAI_API_KEY` for `openai`. The `anthropic` provider needs a key. A local server such as Ollama needs none. |
| `EMOD_AI_ENDPOINT` | `--ai-endpoint` | All | The base URL. For `openai`, it names the server, for example `http://localhost:11434/v1` for Ollama, and it is the OpenAI API, `https://api.openai.com/v1`, when it is not set. For `anthropic`, it names a proxy that speaks the Messages API. |
| `EMOD_AI_MODEL` | `--ai-model` | All | The main model. The `anthropic` provider has a default Claude model. The `openai` provider has no default, so the user must set it. |
| `EMOD_AI_MODEL_CHEAP` | `--ai-model-cheap` | All | The model for small passes, such as a name suggestion. When it is not set for `openai`, emod uses `EMOD_AI_MODEL`. |
| `EMOD_AI_EFFORT` | `--ai-effort` | All | The default effort: `low`, `medium`, `high` or `xhigh`. |

emod reads the keys from the flags, then the environment, then `~/.config/emod/config`, then the built-in
defaults. The first source that sets a key wins. The file holds one `KEY=VALUE` line for each key. emod
removes the spaces around the key and the value, and a pair of quotes of the same kind, `"` or `'`,
around the value.

The API key has its own rules:

- It has no flag, because a flag shows in the process list and in the shell history.
- It never comes from `~/.config/emod/config`. When the file holds `EMOD_AI_API_KEY`, emod stops and tells
  the user to remove the line, so that the key does not stay in a plain file.
- [03](../03-desktop-ai-panel.md#key-storage) adds the OS credential store as the third source, after the
  two environment variables.

**Opt-in.** AI is on when one `EMOD_AI_*` key is set. `ANTHROPIC_API_KEY` or `OPENAI_API_KEY` alone does
not turn AI on, because a user can hold it for another tool. That user sets `EMOD_AI_PROVIDER`, and emod
then uses the key of that provider. emod does not send `ANTHROPIC_API_KEY` to an `openai` server, or
`OPENAI_API_KEY` to the Anthropic API.

The desktop app has a settings dialog. The dialog writes the settings to the same file with mode `0600`,
and the API key to the key store of [03](../03-desktop-ai-panel.md#key-storage). It shows the provider, the
endpoint and the model in the header of the AI panel. Therefore a user of the desktop app does not edit a
file or set an environment variable.

`emod ai` with no subcommand checks the configuration and runs one round of the repair loop. `emod ai status`
shows the settings and the source of the API key, but not the key, and calls no model. Each AI feature adds
a subcommand, for example `emod ai review`. With no provider configured, `emod ai` prints the setup steps
and exits with code 0. The LSP shows no AI actions, and the desktop app hides the AI panel. No other command
changes.

## Data that leaves the machine

A prompt holds the `.emod` source or its JSON export, and the instruction of the user. With a remote
provider, this data goes to the provider. With a local server, such as Ollama or LM Studio, no data leaves
the machine. The documentation of each AI command states what it sends, and the desktop panel shows the
host of the endpoint before the first call.

## Tokens and latency

emod shows the input tokens and the output tokens of each run, from `Response.Usage`. It does not show a
price.

A run at high effort can take many seconds. The CLI and the desktop panel show the progress of each attempt
of the repair loop, and `maxAttempts` limits each loop. A small pass, such as a name suggestion, uses
`EMOD_AI_MODEL_CHEAP`.

## Testing

- Each feature has unit tests against `llm.Fake`, with canned responses, and against `llm.Broken`. No unit
  test calls a live model.
- The `anthropic` adapter has tests against an HTTP transport that records each request. The tests cover the
  request shape, the key in the `x-api-key` header, and that the SDK reads no setting from the environment.
- The `openai` adapter has tests against an `httptest` server. The tests cover the fallback for
  `response_format`, the fallback for `reasoning_effort`, and the local schema check.
- `internal/llm/isolation_test.go` makes sure that the WASM build and the core packages do not import a
  provider SDK. The test also covers `internal/llm/openai`.
- Tests follow the Go test rules of the repository: one umbrella `Test{Type}` for each type, `t.Run` groups
  for each operation, scenario names that describe behaviour, `testify/require`, and fresh fixtures in each
  leaf subtest.

## Which proposals use this foundation

| Proposal | Uses `llm.Model` | Uses the repair loop | Notes |
| --- | :---: | :---: | --- |
| [01 MCP server](../01-mcp-server.md) | No | No | The host brings the model and runs the loop |
| [02 Semantic model review](../02-semantic-model-review.md) | Yes | No | Findings with structured output. Its DCB knowledge also serves 03, 04 and 06. |
| [03 AI panel in the desktop app](../03-desktop-ai-panel.md) | Yes | Yes, with a baseline | In the process of the desktop app |
| [04 Model from a conversation](../04-model-from-conversation.md) | Yes | Yes | Questions, then a draft |
| [05 Spec suggestions](../05-spec-suggestions.md) | Suggestions only | Yes, with a baseline and assembly | Spec export has no AI |
| [06 Rename and fix suggestions](../06-rename-and-fix-suggestions.md) | AI part only | No | Rename has no AI. The check runs on each suggested edit. |
| [07 Model import](../07-model-import.md) | No | No | Converters with no LLM. Code import runs in an MCP host. |

## Open questions

- **Cloud platforms.** `anthropic-sdk-go` also reaches Claude on AWS Bedrock and Google Vertex AI. Add a
  provider for one of them when a user asks for it and a gateway does not serve that user. Each one needs
  its own settings: a region for Bedrock, and a project ID and a region for Vertex AI.
- **AI in `emod diagram --serve`.** The served viewer runs from a local Go process, which can hold the
  credentials. Decide after the desktop panel ships if the served viewer gets the same panel.

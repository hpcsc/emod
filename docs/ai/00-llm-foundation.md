# AI Foundation for emod — LLM Integration Architecture

> This is the shared foundation for the emod AI proposals (`docs/ai/01`–`10`). Each
> feature proposal assumes the structure described here and does not re-specify it.

## Problem

The AI proposals all need to call an LLM, turn its output into something emod can
act on, and check that output against emod's own rules. Done ad hoc, every feature
would grow its own client, its own retry logic, and its own coupling to a specific
vendor — and none of it would be testable without a network. We want one small,
boring seam that every feature shares.

The temptation is to reach for a "LangChain for Go" framework so providers can be
swapped freely. That is the wrong default for a tool whose `go.mod` is currently
tiny. This document records the decision: **a thin internal port, not a
multi-provider framework**, and explains why the deterministic part of emod — not
the model — is the real engine.

## The key insight: emod is a validatable target

Unlike most LLM applications, emod already has a **deterministic correctness
oracle**: `emod validate`, `emod lint`, and the bundled CUE schema. Generated or
edited models can be machine-checked, and the errors fed back to the model until
the output is clean.

That makes the highest-value pattern a **generate → validate → lint → repair
loop**, and that loop is *provider-agnostic by construction*: it depends only on
emod's existing pipeline plus a generic "ask the model" call. The provider is a
leaf, not the core. Keep it that way.

## Goals

- One internal interface (`llm.Model`) that all AI features depend on.
- Concrete LLM access with **zero new heavy dependencies** beyond an official SDK.
- The validate/lint repair loop lives in emod core, depends only on the interface.
- Every AI feature is unit-testable with a mock model (no network in tests).
- A clear, documented answer to "can we swap providers?" — yes, at a seam, without
  a framework.

## Non-Goals

- Adopting a general agent/LLM framework (`langchaingo` or similar).
- Speculative multi-vendor adapters before a real second provider exists.
- Embedding provider credentials or SDKs into the WASM build.
- Inventing a new prompt-orchestration DSL. Plain Go control flow is enough.

---

## Decision 1 — A thin port, defined by emod

emod core depends on an interface it owns, never on a concrete SDK:

```go
package llm

type Message struct {
	Role    string // "user" | "assistant"
	Content string
}

type GenerateRequest struct {
	System   string
	Messages []Message
	Schema   []byte // optional JSON Schema; when set, the response must conform
	Effort   string // "low" | "medium" | "high" | "xhigh" — maps to provider effort
}

type Response struct {
	Text  string
	Usage Usage // input/output tokens, for cost reporting
}

// Model is the only LLM surface emod core knows about.
type Model interface {
	Generate(ctx context.Context, req GenerateRequest) (Response, error)
}
```

This is ~20 lines. It buys the two things that matter: features can be tested
against a mock `Model`, and the concrete client can change without touching
feature logic.

## Decision 2 — Back it with the official Anthropic Go SDK (Bedrock)

The concrete adapter wraps `anthropic-sdk-go`. That SDK is itself the
*backend* abstraction for Claude — the same `Messages.New` surface works across
the Anthropic API, AWS Bedrock, and Google Vertex. indebted already runs LLMs
through Bedrock, so the default adapter uses the **Bedrock Mantle client**:

```go
client := bedrock.NewMantleClient(ctx, bedrock.MantleClientConfig{AWSRegion: region})
// model IDs carry the anthropic. prefix on Bedrock, e.g. "anthropic.claude-opus-4-8"
```

This covers "swap the account / region / hosting backend" with no abstraction code
of our own.

**Model selection** (Claude, latest generation):

| Use | Model | Why |
|-----|-------|-----|
| Generation, reverse-engineering, semantic review | `anthropic.claude-opus-4-8` | Hardest reasoning; correctness matters |
| Cheap mechanical passes (rename suggestions, single-finding fixes) | `anthropic.claude-haiku-4-5` | Fast and cheap; low-stakes |

Use adaptive thinking (`thinking: {type: "adaptive"}`) and set `effort` per call
(`high`/`xhigh` for generation and review, `low` for mechanical passes). Do **not**
set `budget_tokens` or sampling params — they 400 on this generation.

**Structured output.** Most features need the model to return data conforming to a
schema (an AST fragment, a list of findings, a verdict). On the Go SDK this is the
strict-tool-use / structured-output path — emod passes the JSON Schema and gets a
validated object back, so there is no brittle string parsing. The `Schema` field on
`GenerateRequest` carries it.

## Decision 3 — Don't adopt a framework

`tmc/langchaingo` is the main "swap any provider" option in Go. We are **not**
using it: it lags provider features, its abstractions leak, and it is a large
dependency for a parser/CLI tool. Our 20-line interface we control beats a
framework we don't.

If a genuine **non-Claude** vendor is ever required (the Anthropic SDK only spans
Claude across backends), there are two clean paths — pick when the need is real,
not before:

1. **Front everything with an OpenAI-compatible gateway** (LiteLLM proxy,
   OpenRouter, or Bedrock's own Converse API, which normalizes across
   Claude/Llama/Mistral/Titan). One adapter speaks one protocol; vendor swap is
   config, not code.
2. **Add a second adapter** behind `llm.Model`. YAGNI until the provider exists.

---

## The repair loop (provider-agnostic core)

The engine that makes emod special lives in core and depends only on `llm.Model`
plus the existing pipeline:

```go
// GenerateAndRepair asks the model for a model, then loops:
// parse → validate → lint, feeding diagnostics back until clean or attempts run out.
func GenerateAndRepair(ctx context.Context, m llm.Model, prompt string, maxAttempts int) (string, error) {
	req := initialRequest(prompt)
	for attempt := 0; attempt < maxAttempts; attempt++ {
		resp, err := m.Generate(ctx, req)
		if err != nil {
			return "", err
		}
		diags := pipeline.Check(resp.Text) // parser + validator + linter, already in emod
		if len(diags) == 0 {
			return resp.Text, nil
		}
		req = repairRequest(req, resp.Text, diags) // append the .emod + the errors
	}
	return "", ErrNotConverged
}
```

`pipeline.Check` is the deterministic oracle. The model never has to be "right" in
one shot — it has to converge. Features 01, 02, 05, and 10 reuse this loop.

## Where provider code does and does not go

- **MCP server (`06`) needs zero provider code.** It exposes emod's
  parse/validate/lint/export as MCP tools; the *host* (Claude Code, etc.) brings
  the model. For that whole class of use, "which provider" is not our problem.
- **WASM / browser viewer (`10`) needs an HTTP seam, not an embedded SDK.** A
  browser can't safely hold a key and the Go SDK is awkward in WASM, so the viewer
  calls a small backend (ours or the user's proxy) that holds the `llm.Model`. The
  `llm.Model` interface sits behind that network boundary as well.

## Configuration

A single config block (env or `~/.config/emod` / flags), read once and passed to
the adapter:

- `EMOD_AI_REGION` — Bedrock region.
- `EMOD_AI_MODEL` / `EMOD_AI_MODEL_CHEAP` — override the defaults above.
- `EMOD_AI_EFFORT` — default effort.
- `EMOD_AI_ENDPOINT` — optional gateway base URL (enables the OpenAI-compatible
  path of Decision 3 without code changes).

AI features are opt-in: a build/run without credentials configured simply doesn't
offer them; nothing in the existing `validate`/`lint`/`diagram`/`export`/`lsp`
paths gains a hard dependency on an LLM.

## Testing

Every feature is tested against a mock `llm.Model` that returns canned responses,
so behavior (prompt assembly, repair-loop convergence, diagnostic handling) is
covered without a network. Follow the repo's Go test conventions: one umbrella
`Test{Type}` per type, `t.Run` groups by operation, behavior-named scenarios,
`testify/require`, fresh fixtures per leaf.

## Cost and latency

- Token cost is reported back to the user from `Response.Usage` (the existing
  bedrock-cost tooling already models this).
- Generation/review runs can take many seconds at high effort — stream where the
  surface allows (CLI progress, LSP progress notifications) and keep the repair
  loop bounded by `maxAttempts`.
- Prefer Haiku for the high-volume, low-stakes passes to keep cost down.

---

## Which proposals depend on this foundation

| Proposal | Uses `llm.Model` | Uses repair loop | Notes |
|----------|:----------------:|:----------------:|-------|
| 01 NL → model generation | ✅ | ✅ | The canonical loop user |
| 02 Model import / reverse-engineering | ✅ | ✅ | Loop over inferred model |
| 03 Semantic model reviewer | ✅ | — | Emits diagnostics |
| 04 Lint quick-fixes (LSP) | ✅ | — | Anchored to deterministic findings |
| 05 DCB modeling assistant | ✅ | ✅ | DCB-specific suggestions |
| 06 MCP server | ❌ | ❌ | Host brings the model |
| 07 Talk to your model (Q&A) | ✅ | — | Grounded on JSON export |
| 08 Docs / onboarding generation | ✅ | — | Prose output |
| 09 BDD / test generation | ✅ | — | Schema-conformant fixtures |
| 10 Conversational viewer editing | ✅ | ✅ | Behind an HTTP seam |

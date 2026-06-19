# AI Foundation — LLM Integration Seam

## Overview

The emod AI proposals (`01`–`10`) all need to call an LLM, turn its output into
something emod can act on, and check that output against emod's own rules. Built ad
hoc, every feature would grow its own client, its own retry logic, and its own
coupling to a specific vendor — none of it testable without a network.

This feature delivers the one small, shared seam every AI feature depends on: a thin
interface that emod core owns (`llm.Model`), a concrete adapter behind it, the
deterministic generate → validate → lint → repair loop that makes emod special, a
mock for network-free testing, configuration, cost reporting, and a guarantee that
AI stays opt-in. It closes with a thin `emod ai` smoke command that proves the seam
works end to end.

The features themselves (`01`–`10`) are out of scope here and have their own stories;
this set builds only the shared plumbing plus the smoke check.

## Goals

- One internal interface (`llm.Model`) that every AI feature depends on, owned by
  emod core and free of any provider SDK.
- Real LLM access through a single concrete adapter, with no new heavy dependency
  beyond the official SDK.
- A reusable generate → validate → lint → repair loop in core that depends only on
  the interface plus emod's existing deterministic pipeline.
- Every AI feature unit-testable against a mock model — no network in tests.
- AI entirely opt-in: existing commands and the WASM build keep working with no
  credentials and no new dependency.
- A working end-to-end proof that the seam delivers value to a real user.

## User Stories

### US-001: Define the `llm.Model` port
**Description:** As an emod feature developer, I want a small interface that emod core owns for talking to a model, so that I can build AI features without coupling them to any vendor SDK.

**Acceptance Criteria:**
- [ ] A dedicated `llm` package exposes the request/response contract: a `Message` (role + content), a `GenerateRequest` (system prompt, messages, optional schema, effort), a `Response` (text + usage), and a `Model` interface with a single `Generate(ctx, request)` method.
- [ ] `Effort` accepts the documented levels (`low`, `medium`, `high`, `xhigh`); an unset value behaves as a sensible default.
- [ ] The `llm` package depends on no provider SDK and no network, and adds no new heavy dependency to emod core.
- [ ] Feature code can be written and compiled against `llm.Model` alone, with no reference to a concrete client.
- [ ] Each field's meaning is documented, especially the optional schema and the effort levels.

**Context:** emod core must depend on an interface it owns, never on a concrete SDK, so the client can change without touching feature logic. The interface is intentionally tiny (~20 lines) — it exists to make features testable and the provider swappable, not to model the whole provider surface.

### US-002: Mock model for network-free tests
**Description:** As an emod feature developer, I want a mock implementation of `llm.Model` that returns canned responses, so that I can unit-test AI features and the repair loop without any network call.

**Acceptance Criteria:**
- [ ] A test double implements `llm.Model` and returns pre-configured responses in sequence.
- [ ] The mock records the requests it received (system, messages, schema, effort) so a test can assert on how the prompt was assembled.
- [ ] The mock can be configured to return an error, so failure handling can be exercised.
- [ ] The mock can return a different response per call, so repair-loop convergence and non-convergence can both be driven.
- [ ] Tests using the mock run with no network access and no credentials.

**Context:** The foundation requires every feature to be unit-testable with a mock model. This is the reusable harness all later stories test against. Follow the repo's Go test conventions: one umbrella `Test{Type}` per type, `t.Run` groups by operation, behavior-named scenarios, `testify/require`, fresh fixtures per leaf.

**Depends on:** US-001

### US-003: Concrete Bedrock adapter
**Description:** As an emod maintainer, I want a concrete `llm.Model` backed by the official Anthropic Go SDK over Bedrock, so that AI features can call a real Claude model through the existing Bedrock setup.

**Acceptance Criteria:**
- [ ] An adapter implements `llm.Model` by wrapping the Anthropic Go SDK's messages surface via the Bedrock client.
- [ ] A `Generate` call returns the model's text and token usage in the `Response`.
- [ ] The adapter uses adaptive thinking and maps the request `Effort` to the provider's effort setting.
- [ ] Generation requests succeed against the model generation in use — no parameters are sent that this generation rejects.
- [ ] The default model is the documented Opus generation, with a cheaper model selectable for low-stakes passes.
- [ ] The only new heavy dependency introduced is the official Anthropic SDK.

**Context:** indebted already runs LLMs through Bedrock; the SDK's `Messages.New` surface spans the Anthropic API, Bedrock, and Vertex, so the adapter targets the Bedrock Mantle client. Model IDs carry the `anthropic.` prefix on Bedrock (e.g. `anthropic.claude-opus-4-8`). This story carries the highest external risk (SDK behavior, model IDs, effort mapping), so it is sequenced early to retire that risk.

**Depends on:** US-001

### US-004: Single AI configuration block
**Description:** As an operator, I want to configure the AI backend in one place, so that I can point emod at the right account and models without code changes.

**Acceptance Criteria:**
- [ ] AI configuration is read once from a single source combining environment variables, an `~/.config/emod` file, and/or flags.
- [ ] Supported settings include Bedrock region, default model, cheap model, default effort, and an optional gateway endpoint URL.
- [ ] The resolved configuration is passed to the adapter, and precedence between flags, file, and env is defined and documented.
- [ ] Missing or invalid configuration produces a clear message naming exactly what is missing.
- [ ] Setting the optional gateway endpoint routes calls through it with no code change.

**Context:** The foundation names these keys: `EMOD_AI_REGION`, `EMOD_AI_MODEL`, `EMOD_AI_MODEL_CHEAP`, `EMOD_AI_EFFORT`, `EMOD_AI_ENDPOINT`. No configuration or environment handling exists in the codebase today, so this introduces the first such block. The endpoint override is what later enables an OpenAI-compatible gateway without new code.

**Depends on:** US-003

### US-005: Single deterministic correctness oracle
**Description:** As an emod feature developer, I want one call that runs parse → validate → lint over an emod source and returns all diagnostics, so that the repair loop and other consumers share one deterministic correctness oracle.

**Acceptance Criteria:**
- [ ] A single function accepts emod source text and returns the combined diagnostics from the lexer/parser, validator, and linter.
- [ ] Clean input returns an empty diagnostic list; broken input returns entries carrying position, severity, and message.
- [ ] The existing `validate` and `lint` commands produce the same diagnostics and the same exit codes after adopting the shared oracle.
- [ ] The oracle has no dependency on any LLM or network.

**Context:** Today the pipeline is assembled inline in each CLI command (for example, `RunValidate` chains the lexer, parser, `validator.Validate`, and `linter.Lint`, all producing `[]*diagnostic.Entry`). Consolidating that into one reusable call gives the repair loop the "deterministic oracle" the foundation describes, and lets the existing commands reuse it unchanged.

### US-006: Generate → validate → lint → repair loop
**Description:** As an emod feature developer, I want a reusable loop that asks the model for an emod source and repairs it against the oracle until it is clean or attempts run out, so that features get correct output without the model having to be right on the first try.

**Acceptance Criteria:**
- [ ] Given a prompt and a maximum number of attempts, the loop calls the model, runs the output through the oracle, and returns the source as soon as there are no diagnostics.
- [ ] When diagnostics remain, the loop feeds the offending source and the diagnostics back to the model for another attempt.
- [ ] The loop stops after the maximum attempts and returns a clear "did not converge" outcome.
- [ ] The loop depends only on `llm.Model` and the oracle — no provider SDK.
- [ ] Both convergence and non-convergence are demonstrable with the mock model, with no network.

**Context:** The foundation names this `GenerateAndRepair`; features 01, 02, 05, and 10 reuse it. The principle is that the model never has to be right in one shot — it only has to converge — and the deterministic oracle is what makes that safe.

**Depends on:** US-001, US-002, US-005

### US-007: Schema-conformant structured output
**Description:** As an emod feature developer, I want to ask the model for output that conforms to a JSON Schema and get a validated object back, so that features can consume structured results without brittle string parsing.

**Acceptance Criteria:**
- [ ] When a `GenerateRequest` carries a schema, the adapter requests structured output and returns content conforming to that schema.
- [ ] Output that does not conform to the schema is surfaced as an error rather than silently returned.
- [ ] A feature can supply a schema and receive a validated object it consumes directly.
- [ ] Requests without a schema continue to return plain text.

**Context:** Most features need structured results — an AST fragment, a list of findings, a verdict — rather than prose. On the Go SDK this is the strict-tool-use / structured-output path; the optional `Schema` field on `GenerateRequest` carries the JSON Schema, and validation happens before the result reaches the feature.

**Depends on:** US-001, US-003

### US-008: Cost and token usage reporting
**Description:** As an emod CLI user, I want to see the token usage and cost of an AI run, so that I understand what each generation or review costs me.

**Acceptance Criteria:**
- [ ] Every `Response` carries input and output token counts.
- [ ] After an AI command completes, the user sees the tokens used and an estimated cost.
- [ ] The reported cost reflects the model actually used for the run.
- [ ] Usage from every attempt of a repair loop is included, not just the final attempt.

**Context:** `Response.Usage` carries the token counts. The existing bedrock-cost tooling already models per-model pricing and can be reused to turn token counts into a cost estimate. Generation and review runs can be many seconds at high effort, so showing what they cost matters to the user.

**Depends on:** US-001, US-003

### US-009: AI stays opt-in; existing paths and WASM stay provider-free
**Description:** As an emod user, I want AI to be entirely optional, so that the existing commands and the browser/WASM build keep working with no credentials and no extra dependencies.

**Acceptance Criteria:**
- [ ] With no AI configuration present, `validate`, `lint`, `diagram`, `export`, and `lsp` run exactly as they do today.
- [ ] With no AI configuration present, AI commands are absent or clearly report that AI is not configured — they never break the other commands.
- [ ] The WASM build excludes the provider SDK and credentials and builds successfully.
- [ ] No existing non-AI code path gains a hard dependency on an LLM or on the network.

**Context:** A real WASM target exists (`cmd/emod-wasm/main.go`). A browser can't safely hold a key and the Go SDK is awkward in WASM, so provider code must stay out of that build; any feature that needs a model in the browser goes through an HTTP seam instead of an embedded SDK.

**Depends on:** US-003, US-004

### US-010: End-to-end smoke command proving the seam
**Description:** As an emod CLI user, I want a small `emod ai` command that exercises the whole seam end to end, so that I can confirm AI is configured and the repair loop works before relying on any real feature.

**Acceptance Criteria:**
- [ ] Running the command with AI configured makes one real round-trip and reports success, the model used, and the token usage and cost.
- [ ] The command runs one generate → validate → repair round on a built-in trivial prompt and shows whether the result passed the oracle or did not converge.
- [ ] Running the command with no AI configuration prints a clear setup message and a non-error "not configured" outcome, without affecting other commands.
- [ ] The command is a health/smoke check only and does not implement a real authoring feature.

**Context:** This is the thin proving consumer requested for the foundation — it ties together the adapter, config, repair loop, cost reporting, and opt-in gating into one user-observable result. It should stay a diagnostic/health command and not grow into feature 01 (NL → model generation), which has its own story set.

**Depends on:** US-004, US-006, US-008, US-009

## Non-Goals

- Adopting a general agent/LLM framework (`langchaingo` or similar).
- Speculative multi-vendor adapters before a real non-Claude provider exists.
- Embedding provider credentials or the provider SDK into the WASM build.
- Inventing a new prompt-orchestration DSL — plain Go control flow is enough.
- Building any of features `01`–`10` themselves; only the shared seam plus the smoke command are in scope.
- Streaming as a hard requirement — it is a nice-to-have where the surface allows (CLI progress, LSP notifications), not part of the seam contract.

## Open Questions

- **Smoke command shape:** pure connectivity/health check, or always run one generate → validate → repair round? (Assumed: a minimal round-trip *plus* one repair round on a trivial built-in prompt.)
- **Cost visibility:** should token usage and cost print on every AI command by default, or only behind a `--cost`/verbose flag?
- **Config precedence:** exact order of flags vs file vs env. (Assumed: flags > env > file — to confirm.)
- **Cost pricing source:** reuse the bedrock-cost tooling's pricing table, or embed a small table in emod? (Assumed: reuse.)
- **Repair-loop default:** what default maximum number of attempts should the smoke command use? (Assumed: a small bound such as 3.)
- **Mock placement:** should the mock model live in the `llm` package as an exported test helper, or in a separate testing subpackage?

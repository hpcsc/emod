# Talk to Your Model — Grounded Q&A over an `.emod`

> Part of the [emod AI proposals](./README.md). Assumes the shared [LLM foundation](./00-llm-foundation.md) (the `llm.Model` port and Bedrock-backed Claude) and does not re-specify it.

## Problem

A non-trivial event model is a graph, not a document. `examples/inbound_customer_comms_agentic_reply.emod` has five contexts, ten-plus slices, and a web of `command -> event -> automation -> command` edges that hop across context boundaries — an `EmailConversationReplyInitiated` event in `InboundEmail` fans out into three automations targeting three *other* contexts (`OutboundDelivery`, `CollectionHolds`, `ConversationHoldScheduling`). Answering an ordinary question about that graph — *"trace the path from inbound email to hold release"*, *"which contexts touch `CustomerCollectionHold`?"*, *"what breaks if I rename `EmailConversationClassified`?"* — means tracing edges by eye across hundreds of lines and several context blocks. That is exactly the work people stop doing, so models drift out of people's heads.

Some of these questions emod can already answer deterministically: `emod export -f json` materializes the whole graph, the LSP does find-references and go-to-definition, and `emod slices list` lists slices with their pattern types. But those answers are *structured*, not *conversational*. They require you to know the exact name to look up, to read JSON, and to do the cross-referencing synthesis yourself. The gap is the fuzzy, natural-language layer: ask in English, get a prose answer that names the real slices, contexts, and events involved, and that you can trust because it was derived from the actual model — not hallucinated.

The naive version of this fails in a known way: paste the `.emod` into a chat window and ask. The model invents plausible-but-wrong edges (an automation that doesn't exist, a subscription on the wrong view), answers about event-modeling in general rather than *this* model, and gives you no way to tell a grounded claim from a confident guess. The fix is to make the answer *derive from* a machine-extracted representation of the model and to forbid the model from going beyond it.

## Goals

- A CLI command to ask a natural-language question about a specific `.emod` file and get a grounded, prose answer that cites real names from the model: `emod ai ask file.emod "<question>"`.
- An interactive REPL mode for follow-up questions over the same model, with the model context loaded once and reused across the session.
- Answers grounded on `emod export -f json` (and, when useful, the raw `.emod`) as the *authoritative* source, so claims are derived from the real graph, not invented.
- A faithfulness contract: the model answers **only** from the provided model and explicitly says *"that is not in the model"* when the answer is not derivable, rather than guessing.
- The query types this shines at: reachability / trace queries across the `command -> event -> automation -> command` graph, impact / rename analysis, "who subscribes to / who emits X", and cross-context dependency questions.
- A `--json` answer form for tooling (the answer text plus the structured list of model elements it cited), so the LSP or viewer can consume it later.
- Zero impact on existing commands; opt-in and absent without configured credentials (per the foundation's configuration section).

## Non-Goals

- **Editing or extending** the model. This is read-only Q&A; conversational editing is [proposal 10](./10-conversational-viewer-editing.md).
- **Generating prose documentation / onboarding narratives.** That is the deliberate, structured output of [proposal 08](./08-docs-generation.md); this is ad-hoc question answering.
- **Emitting review findings / modeling smells.** Opinions about whether the model is *good* belong to [proposal 03](./03-semantic-model-reviewer.md). Q&A answers what the model *says*, not what it *should* say.
- **Replacing the deterministic queries.** Where emod can compute an exact answer (find-references, the dependency graph, slice listing), the deterministic path is authoritative; the AI layer is for fuzzy / NL questions and synthesis, and it *consults* deterministic results rather than competing with them.
- Re-specifying the `llm.Model` port, the Bedrock adapter, or model selection — all defined in the foundation. (This feature uses `llm.Model` but **not** the repair loop — there is no `.emod` being produced to validate.)

## How It Works

### Grounding on the JSON export

The whole model fits comfortably in context, so there is no retrieval problem to solve — the right move is to put the *authoritative, machine-extracted* representation in front of the model and instruct it to answer only from that. emod already produces exactly this:

```
emod export -f json file.emod
```

This is the structured JSON from `internal/export` (`ExportJSON` / `ExportJSONDiagnostics`): the full model tree — `actors`, `contexts`, `aggregates`, `slices`, and within each slice the `commands`, `events` (with `fields` and `source`), `flows` (`command_name -> event_name`), `views` (with `subscribes`), `automations` (`trigger_event` / `command` / `target_context`), and `translations` (`external_system`). Crucially it carries **source positions** (`filename`/`line`/`column`) on every element, which lets the model cite a line and lets a tooling consumer jump to it.

The grounding context handed to the model is, in order:

1. **The JSON export** — the authoritative graph. This is what the model reasons over.
2. **A compact derived edge list** (optional, see below) — pre-computed adjacency so the model doesn't have to reconstruct the graph from nested JSON for trace questions.
3. **The raw `.emod` text** (optional, `--with-source`) — useful for questions about comments and intent that the JSON also carries (`comments[]`), but the JSON is primary.

The package owning the model-facing logic lives under `internal/ai` (e.g. `internal/ai/ask.go`), depending only on `llm.Model` plus the existing `internal/export` and `internal/parser`/`internal/validator` (to load and serialize the model). The CLI command in `internal/cli` is a thin wrapper, matching the structure of the existing `RunExport` / `RunSlicesList` actions in `internal/cli/app.go`.

### The derived edge list (deterministic pre-computation)

The diagram-oriented export (`emod export -f diagram-json`, `ExportDiagramJSON`) already resolves the graph into typed edges — `flow`, `trigger_command`, `subscription`, `automation_trigger`, `automation_command`, `reads`, `translation_command`. That resolution logic in `convertModelToDiagram` is exactly the cross-referencing a human does by hand for a trace question. Feeding the model a flattened version of those edges (source name, target name, edge type, both contexts) turns "trace the path" from a multi-hop reasoning puzzle into a graph walk the model can verify against an explicit list. This is the single most effective grounding aid for reachability questions, and it costs little: the edge list is far smaller than the full JSON.

### Query types it shines at

| Question kind | Example (against the example model) | What grounds it |
|---|---|---|
| **Reachability / trace** | "Trace the path from inbound email to hold release." | The derived edge list — a path walk over `flow`/`automation_*` edges. |
| **Impact / rename** | "What breaks if I rename `EmailConversationClassified`?" | Every edge and `subscribes`/`trigger_event` referencing the name — the same set the LSP find-references computes. |
| **Producers / consumers** | "Who subscribes to `EmailConversationReplyInitiated`?" | `views[].subscribes` and `automations[].trigger_event` across all contexts. |
| **Cross-context dependency** | "Which contexts touch `CustomerCollectionHold`?" | `automations[].target_context` plus the aggregate's home context. |
| **Conditional / branch** | "What happens if customer identification is deferred?" | The `EmailConversationCustomerIdentificationDeferred` event and the automations triggered by it. |
| **Inventory + synthesis** | "List every automation that targets the `CollectionHolds` context, and say why." | `automations[]` filtered by `target_context`, narrated. |

The first five are *fuzzy framings of questions with deterministic answers*; the model's job is to phrase the question into the graph, walk it, and narrate the result — not to invent the result. The last is pure synthesis. All of them benefit from the model naming the real `slice`, `context`, and `event` it used.

### Faithfulness controls

Hallucination is the central risk for a read-only Q&A feature, so faithfulness is engineered, not hoped for:

1. **Closed-world system prompt.** The system prompt (embedded via `go:embed`) states plainly: the JSON model is the *only* source of truth; answer strictly from it; if the answer is not derivable from the provided model, say *"That is not in the model"* and stop; never invent a slice, event, automation, context, or field name that does not appear in the JSON; when stating a relationship, cite the element names (and line, from the `position`) it comes from.
2. **Cite-or-abstain.** Every factual claim about the model must name the model element(s) it rests on. An answer that cannot cite is an answer the model should not make. This makes hallucinations *visible* — a cited name that isn't in the export is a checkable error.
3. **Post-hoc citation check (deterministic).** After the model answers, emod scans the answer for model-element names and verifies each against the names actually present in the export. Any cited name absent from the model is flagged in the output (and, in `--json`, listed under `unverified_citations`). This is a cheap, deterministic backstop that turns a silent hallucination into a labeled one. It is a *guardrail*, not a gate — it never rewrites the answer, it annotates it.
4. **Low temperature, no embellishment.** Per the foundation, sampling params are not set on this generation, but the prompt instructs terse, factual answers with no speculation beyond the model. Effort is `medium` for ordinary lookups and `high` only for multi-hop trace/impact questions.

### Relation to deterministic queries — prefer exact when exact exists

The AI layer does not replace the things emod already computes; it routes to them and narrates them. Two postures:

- **AI-first, deterministically backstopped (default).** The model reasons over the JSON + edge list and answers; the post-hoc citation check verifies names. Good for fuzzy phrasing and synthesis.
- **Deterministic-first, AI-narrated (for exact-answerable questions).** For a question that maps cleanly onto a find-references or graph query — *"who subscribes to X"*, *"what references Y"* — emod can compute the exact set first (the same code paths behind the LSP find-references and the diagram-json edge resolver) and hand that *result set* to the model as additional grounding, asking it only to phrase the answer. The deterministic set is then the authority; the model cannot under- or over-count. This is the most trustworthy mode and is the natural home for impact/rename analysis. A `--exact` flag (or automatic detection of an exact-answerable question) selects it.

The guiding rule: **if emod can compute it, compute it, and let the model narrate; only let the model reason freely when the question is genuinely fuzzy.**

### Cost, latency, and caching across a REPL session

The grounding context (JSON export + edge list, optionally raw source) is the same for every question in a session, and it is the bulk of the input tokens. In the REPL it is assembled once and reused: it is loaded into the conversation as a leading system/context block and kept across turns, so follow-up questions pay for the prompt-cacheable model context once rather than re-sending it. Per the foundation, `Response.Usage` is reported back so the user sees token cost; ordinary lookups run on `medium` effort to keep latency and cost down, reserving `high` for multi-hop questions. Single-shot `ask` invocations pay the full context each time by nature; the REPL is the cost-efficient surface for exploration.

## Interface

A new `ask` subcommand under the `ai` command group in `internal/cli/app.go` (the same group that hosts `generate` from [proposal 01](./01-nl-to-model-generation.md)). It mirrors the existing command structure (urfave/cli v2).

```
emod ai ask <file.emod> "<question>"   [flags]
emod ai ask <file.emod>                 # no question → interactive REPL

Flags:
      --repl              force interactive REPL even with a question argument
      --exact             prefer the deterministic-first path for exact-answerable questions
      --with-source       include raw .emod text in the grounding context (default: JSON only)
      --effort <level>    low|medium|high|xhigh (default: EMOD_AI_EFFORT or medium)
      --show-cost         print token usage and cost summary (default on for a TTY)
      --json              machine-readable result (answer, citations, unverified, usage)
```

Single-shot example:

```
$ emod ai ask examples/inbound_customer_comms_agentic_reply.emod \
    "Who subscribes to EmailConversationReplyInitiated?"

Three readers subscribe to EmailConversationReplyInitiated (InboundEmail context):

  - EmailDecisionsTopicView      (InboundEmail,    slice "Publish decision")
  - OutboundReplyQueueView       (OutboundDelivery, slice "Outbound reply queue")

…and three automations are triggered by it:

  - SendReplyAfterReplyInitiated     → command EmailConversationSendReply (OutboundDelivery)
  - EmailConversationApplyHold       → command CustomerCollectionHoldApply (CollectionHolds)
  - EmailConversationScheduleHoldExpiry → command ScheduleConversationHoldExpiry (ConversationHoldScheduling)

tokens: 9,210 in / 380 out  (~$0.05)
```

REPL example (context loaded once, reused):

```
$ emod ai ask examples/inbound_customer_comms_agentic_reply.emod
loaded model "Inbound Customer Comms Agentic Reply" — 5 contexts, 14 slices (cached for this session)
ask> which contexts touch CustomerCollectionHold?
CollectionHolds owns the CustomerCollectionHold aggregate. Two other contexts drive
it via cross-context automations: InboundEmail (EmailConversationApplyHold,
EmailConversationReleaseHold) and CustomerCases (CustomerCaseApplyHold), all
targeting context CollectionHolds.
ask> what would break if I renamed EmailConversationClassified?
ask> /exit
```

The `--json` form returns the answer plus its citations so it can be scripted or consumed by the LSP / viewer later:

```json
{
  "answer": "Three readers subscribe to EmailConversationReplyInitiated …",
  "citations": [
    { "name": "EmailDecisionsTopicView", "kind": "view", "context": "InboundEmail", "line": 250 },
    { "name": "SendReplyAfterReplyInitiated", "kind": "automation", "context": "InboundEmail", "line": 199 }
  ],
  "unverified_citations": [],
  "usage": { "input_tokens": 9210, "output_tokens": 380, "cost_usd": 0.05 }
}
```

## Worked Example

Using `examples/inbound_customer_comms_agentic_reply.emod`.

**Question:** *"Trace the path from inbound email to hold release."*

emod loads the file, runs `ExportJSON` for the authoritative tree and the diagram-json edge resolver for the adjacency list, and hands both to `anthropic.claude-opus-4-8` at `high` effort (a multi-hop trace) with the closed-world prompt.

**Grounded answer:**

> The path crosses four contexts. Each step cites the slice it comes from.
>
> 1. **InboundEmail / "Inbound ingest"** — `EmailConversationReceiveInboundMessage` → `EmailConversationInboundReceived` (`flow`).
> 2. **InboundEmail / "Identify customer"** — automation `EmailConversationIdentifyCustomer` reacts to `EmailConversationInboundReceived`, runs `EmailConversationRecordCustomerIdentification` → `EmailConversationCustomerIdentified`.
> 3. **InboundEmail / "Classify message"** — automation `EmailConversationClassify` reacts to `EmailConversationCustomerIdentified`, runs `EmailConversationRecordClassification` → `EmailConversationClassified`.
> 4. **InboundEmail / "Compose auto-reply"** — automation `EmailConversationComposeReply` reacts to `EmailConversationClassified`, runs `EmailConversationInitiateReply` → `EmailConversationReplyInitiated`.
> 5. **InboundEmail / "Schedule hold expiry"** — automation `EmailConversationScheduleHoldExpiry` reacts to `EmailConversationReplyInitiated` and runs `ScheduleConversationHoldExpiry`, **targeting context ConversationHoldScheduling**.
> 6. **ConversationHoldScheduling / "Schedule expiry"** — `ScheduleConversationHoldExpiry` → `ConversationHoldExpiryScheduled`.
> 7. **ConversationHoldScheduling / "Fire expiry at wakeAt"** — the `ConversationHoldExpiryClock` trigger (actor `Scheduler`) fires `FireConversationHoldExpiry` → `ConversationHoldExpiryDue`.
> 8. **InboundEmail / "Release hold at wakeAt"** — automation `EmailConversationReleaseHold` reacts to `ConversationHoldExpiryDue` and runs `CustomerCollectionHoldRelease`, **targeting context CollectionHolds**.
> 9. **CollectionHolds / "Release collection hold"** — `CustomerCollectionHoldRelease` → **`CustomerCollectionHoldReleased`**. The hold is released.
>
> Note: the hold itself is *applied* on a parallel branch — automation `EmailConversationApplyHold` also reacts to `EmailConversationReplyInitiated` (step 4's event) and runs `CustomerCollectionHoldApply` in CollectionHolds. Release closes the loop opened there.

Every name in that answer — `EmailConversationReplyInitiated`, `ConversationHoldExpiryDue`, `EmailConversationReleaseHold`, `CustomerCollectionHoldReleased`, the slice titles, the `target context` hops — is present in the JSON export. The post-hoc citation check confirms all cited names exist in the model; `unverified_citations` is empty. The model's contribution is *ordering the hops and the parallel-branch observation*, which is synthesis the deterministic export does not give you directly; the *facts* are all grounded.

Contrast a question with no grounded answer — *"How long is the quiet window before a hold expires?"*:

> That is not in the model. The model schedules an expiry via `ScheduleConversationHoldExpiry` with a `wakeAt timestamp` field (ConversationHoldScheduling / "Schedule expiry"), but the duration that computes `wakeAt` is not represented in the `.emod` — it lives in the scheduling implementation, not the event model.

The model names the relevant element (`wakeAt`) and then declines, rather than inventing a number.

## Implementation Plan

**Phase 1 — Core ask over JSON grounding (M).** `internal/ai/ask.go`: load the file via the existing parser, build the grounding context from `export.ExportJSON`, assemble the closed-world system prompt (`go:embed`), and a single `Ask(ctx, m, question, grounding)` over `llm.Model` returning answer text + usage. No edge list, no REPL yet. Unit-tested against a mock `llm.Model` (canned answers) per the repo's Go test conventions — assert prompt assembly includes the export and the closed-world instruction, and that usage is surfaced.

**Phase 2 — CLI surface (S).** `emod ai ask <file> "<question>"` in `internal/cli`, mirroring `RunExport`: file + question args, `--effort`, `--with-source`, `--show-cost`, and the `--json` result shape. Wire exit-code handling consistently with the other actions in `internal/cli/app.go`.

**Phase 3 — Edge-list grounding + citation check (M).** Feed the flattened `ExportDiagramJSON` edges as additional grounding for trace/reachability questions. Add the deterministic post-hoc citation check that verifies cited names against the export and populates `unverified_citations`. This is where trace answers become reliable.

**Phase 4 — REPL with cached context (M).** Interactive `ask>` loop when no question is given (or `--repl`): assemble grounding once, keep it across turns as a leading context block, report cumulative usage, support `/exit`, `/cost`, `/reload`. The reuse of the cached model context across turns is the cost story for exploration.

**Phase 5 — Deterministic-first path (M/L).** Detect exact-answerable questions (producer/consumer, references, cross-context dependency) and compute the answer set deterministically — reusing the find-references logic behind the LSP and the edge resolver in `convertModelToDiagram` — then hand that set to the model to *narrate only*. `--exact` forces it. This is the highest-trust mode and the right default for rename/impact questions; sequence last because it depends on factoring the reference logic into something `internal/ai` can call.

## Risks & Mitigations

- **Hallucination — invented edges or names.** The whole faithfulness stack addresses this: closed-world prompt, cite-or-abstain, the deterministic post-hoc citation check that labels any name absent from the export, and (Phase 5) computing exact-answerable questions deterministically so the model only narrates. The model is never the authority on the graph; the export is.
- **Staleness — the export drifts from the file.** The export is regenerated from the file on every `ask` invocation (and on `/reload` in the REPL), so grounding is always current with the saved file. The grounding cites `position` lines, so a user can verify against the actual source. The REPL warns if the file's mtime changes mid-session.
- **Confident-but-wrong synthesis.** Passing the citation check proves every *named* element is real; it does not prove the *relationship* the model asserted between them is correct. Mitigate by preferring the deterministic-first path for relationship questions (Phase 5), keeping answers terse and cited so a reader can check each hop, and pointing users at `emod diagram --serve` to eyeball a traced path. Q&A answers what the model contains, not whether the model is right — that is [proposal 03](./03-semantic-model-reviewer.md).
- **Cost of large models / repeated questions.** The model is small enough to fit in context, but the export is the bulk of input tokens. Mitigate with REPL context reuse (load once, reuse across turns), `medium` effort by default, JSON-only grounding (raw source is opt-in), and surfacing `Response.Usage` every run.
- **Question routed to AI when a deterministic query was exact.** Wasted tokens and lower trust. Mitigate by detecting exact-answerable questions and routing them deterministic-first (Phase 5), and documenting that `emod` already answers structural lookups via the LSP and `emod slices list` / `emod export`.
- **Non-determinism in tests.** Never call a live model in tests; the mock `llm.Model` returns canned answers, so prompt assembly, grounding content, citation-check behavior, and REPL turn handling are asserted deterministically with `testify/require`.

## Open Questions

- **Default effort.** Is `medium` the right default, escalating to `high` only on detected multi-hop questions, or should effort be uniform? Needs measurement on a question corpus — trace questions clearly need more, simple lookups clearly need less.
- **How aggressively to auto-route to the deterministic-first path.** A classifier that decides "this is exact-answerable" can itself be wrong. Start with `--exact` opt-in, and only auto-route the highest-confidence patterns (literal "who subscribes to X", "what references X").
- **Citation-check strictness.** Should an answer with unverified citations be downgraded (warned, exit non-zero) or merely annotated? Current lean: annotate and warn, never suppress — the user should see both the answer and the flag.
- **Multi-file / multi-model sessions.** The grammar is one model per file today. If a workspace spans several `.emod` files, should the REPL load and ground on all of them? Defer until multi-file models exist.
- **Prompt-cache reliance.** REPL cost savings assume the leading context block is cache-friendly across turns. How much that actually saves depends on backend caching behavior; measure before promising it as the headline cost story.
- **Overlap with the MCP server.** [Proposal 06](./06-mcp-server.md) exposes parse/validate/lint/export as MCP tools, letting a host agent do its own grounded Q&A. Is the standalone `ai ask` worth maintaining alongside it, or is it the credential-free, no-host convenience layer over the same exports? Current lean: keep both — `ai ask` is the zero-setup path, MCP is the bring-your-own-host path.

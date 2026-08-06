# Conversational Viewer Editing

> Part of the [emod AI proposals](./README.md). Assumes the shared [LLM foundation](./00-llm-foundation.md) (the `llm.Model` port behind an HTTP seam, Bedrock-backed Claude, and the repair loop) and does not re-specify it. See [`../wasm-architecture.md`](../wasm-architecture.md) for the viewer/WASM split.

## Problem

The viewer (`emod diagram --serve`) already renders a model live and lets you nudge
it: drag nodes, rename inline, add a slice from the context menu, delete a node,
export back to `.emod` (`internal/viewer/static/interaction.js`, `ui.js`,
`emod-export.js`). But every edit is *mechanical and local* — one node, one field,
one arrow at a time. The interesting changes are *structural and intentful*:

- "add a refund flow to the `OutboundDelivery` context"
- "split `EmailDecisionsTopicView` — it subscribes to too many events" (the linter's
  god-view rule already flags this: it subscribes to four events)
- "rename `EmailConversationCustomerIdentificationDeferred` to something an analyst
  would understand"
- "what happens if identification is deferred?" (read-only — a natural subset)

None of these map to a single click. Each requires understanding the whole model,
producing a coherent multi-node edit, and — crucially — staying *valid*: emod has a
deterministic correctness oracle (parser + validator + linter), and an edit that
breaks it is worse than no edit.

We want a chat panel in the viewer where the user describes an intent in natural
language, the assistant proposes a concrete `.emod` change, that change is run
through the repair loop until it parses/validates/lints clean, and the viewer
re-renders it — all without the viewer ever holding a model credential, and with the
viewer remaining fully usable when no AI backend is present.

## Goals

- A chat panel wired into the existing viewer (`bus.js` / `store.js` / `ui.js`),
  driving live re-renders through the existing WASM pipeline.
- The LLM call is **server-side** (key custody); parse/validate/lint/layout stay
  **client-side** in WASM (fast, offline-capable). Be explicit about the seam.
- Every applied edit is a *valid* model: the foundation's generate → validate → lint
  → repair loop runs before the user is shown a proposal.
- Edits are reviewable: diff preview, accept/reject, and undo.
- Read-only questions ("what happens if X?") work through the same panel without
  mutating the model.
- **Degraded mode is first-class**: the viewer works exactly as today with no
  backend; the chat panel is simply hidden.

## Non-Goals

- CLI / batch generation of a model from scratch — that is [01](./01-nl-to-model-generation.md).
- Pure grounded Q&A as a standalone feature — that is [07](./07-talk-to-your-model-qa.md)
  (we reuse its grounding for the read-only subset only).
- An MCP transport — that is [06](./06-mcp-server.md).
- Replacing the existing direct-manipulation editing. Chat is additive; drag/rename/
  delete stay.
- Persisting models server-side or multi-user collaboration. The model lives in the
  browser; the backend is stateless.
- Embedding a provider SDK or credential into the WASM build (the foundation forbids
  it).

## How It Works

The split follows one rule: **the deterministic pipeline runs where it is cheapest
and safest (the browser, in WASM); the non-deterministic model call runs where the
credential can live (the backend).**

| Step | Where | Why |
|------|-------|-----|
| Hold the current `.emod` source + selection context | Browser (`store.js`) | It's the user's working copy; never leaves except as prompt grounding |
| Ground the prompt (current `.emod` + JSON export of the model) | Browser builds it, backend receives it | Export already exists in WASM (`exportJSON`); avoids a server round-trip to read the model |
| Call the LLM for an edit | **Backend** (`llm.Model`) | A browser can't safely hold a Bedrock key; the Go SDK is awkward in WASM |
| Repair loop: parse → validate → lint, feed diagnostics back | **Backend** (native Go pipeline) | The loop must re-invoke the model on each failed attempt; co-locating it with the model call avoids N browser↔server round-trips |
| Final parse → layout → render | Browser (WASM via `model.js` → `wasm.js`) | Already the viewer's hot path; instant, offline once loaded |
| Show diff / diagnostics / accept-reject / undo | Browser (`ui.js`, `bus.js`) | Pure UX state |

### The edit flow

1. **User types an instruction** into the chat panel. The panel reads the current
   working source from `store.js` and the model's JSON export (the viewer already has
   `exportJSON` available through the WASM module — see `cmd/emod-wasm/main.go`,
   `internal/wasm/RunPipelineExportJSON`).

2. **Browser POSTs to the backend** (`/ai/edit`) with: the instruction, the current
   `.emod` source, the JSON export (cheap structured grounding so the model doesn't
   re-parse prose), and optional *focus* (the selected node / context the user
   right-clicked, from `store.interaction.selectedNodeId`). The body carries a `mode`:
   `edit` or `ask`.

3. **Backend runs the repair loop.** For `mode: edit`, it asks `llm.Model` (effort
   `high`, `anthropic.claude-opus-4-8`) for an edited `.emod`, then runs the same
   `pipeline.Check` the foundation describes (`lexer` → `parser` → `validator` →
   `linter`). On non-empty diagnostics it appends the proposed `.emod` + the
   diagnostics to the conversation and retries, bounded by `maxAttempts`. It returns
   only a model that already passes — or an error if it never converged.

4. **Backend returns** the new `.emod` (and a server-computed unified diff against
   the source it was given, so the browser doesn't depend on a diff library), plus
   the residual diagnostics (warnings the user may *choose* to accept, e.g. a lint
   warning the instruction explicitly asked for) and token usage for cost display.

5. **Browser previews the diff** in the chat panel — *it does not apply it yet*. The
   user accepts or rejects.

6. **On accept**, the browser sets the new source as the working copy and runs it
   through the *existing* client path — `Model.sendParse(store, newSource, statusEl)`
   in `model.js`, which calls `wasm.parseEmod` → the in-browser pipeline → diagram
   JSON — then `Model.setModelData`, exactly as the Render button does today
   (`viewer.js`). The diagram re-renders via `bus.emit('data:changed')`; diagnostics
   flow into the existing diagnostics panel via `bus.emit('diagnostics:changed')`.

7. **Undo** pops the previous source off a small client-side stack and re-parses it
   the same way. No server involvement.

For `mode: ask`, steps 3–7 collapse: the backend asks `llm.Model` (no repair loop,
no schema) for a prose answer grounded on the export, and the browser renders it as a
chat message. Nothing mutates.

### Why the repair loop is the engine, not the model

The model never has to be right in one shot. The foundation's
`GenerateAndRepair` re-checks against emod's own rules and feeds the errors back.
That is what makes "split this god view" safe: a naive split might leave a `view`
subscribing to an event that no longer exists, or produce an invalid `subscribes`
list — the validator catches it and the loop fixes it before the user ever sees the
proposal. The same WASM-side `pipeline.Check` that powers the diagnostics panel is
the backend-side oracle; client and server run the *same* pipeline code
(`internal/lexer`, `internal/parser`, `internal/validator`, `internal/linter`), just
compiled for different targets.

## Architecture

```
┌──────────────────────────── Browser (viewer) ────────────────────────────┐
│                                                                            │
│  chat.js (new)            store.js / bus.js            model.js + wasm.js  │
│  ┌──────────┐  intent     ┌──────────────┐  source     ┌────────────────┐ │
│  │chat panel│────────────▶│ working copy │────────────▶│ wasm.parseEmod │ │
│  │ + diff   │◀────────────│  + undo stack│◀────────────│ (emod.wasm)    │ │
│  └────┬─────┘  new .emod  └──────────────┘  diagram     └───────┬────────┘ │
│       │                                                          │ render   │
│       │ POST /ai/edit {instruction, source, export, focus, mode}│          │
│       │                                                  ┌───────▼───────┐  │
│       │                                                  │ renderer.js   │  │
│       │                                                  │ SVG diagram   │  │
└───────┼──────────────────────────────────────────────────────────────────┘
        │  (only the LLM call crosses this seam)
        ▼
┌──────────────────────── AI backend (Go) ─────────────────────────────────┐
│  internal/viewer/serve.go : /ai/edit  (or `emod ai serve`)                 │
│                                                                            │
│   handler ── repair loop ──┬─ llm.Model.Generate (Bedrock Claude)          │
│                            └─ pipeline.Check (lexer→parser→validator→      │
│                                               linter)  [native Go]         │
│   holds: EMOD_AI_* config, the Bedrock credential.  Stateless per request. │
└────────────────────────────────────────────────────────────────────────────┘
```

Two deployment shapes, same handler:

- **Integrated**: `emod diagram --serve` gains an `/ai/edit` route in
  `internal/viewer/serve.go`, *only registered when AI config is present*. This is
  the natural home — `serve.go` already serves the static viewer, the WASM binary,
  and a native `/parse` route, so it already mixes "serve the app" with "run the Go
  pipeline server-side." Adding one more route that additionally holds `llm.Model` is
  a small extension of an existing pattern.
- **Separate**: `emod ai serve` runs only the `/ai/edit` (+ `/ai/health`) endpoints,
  for users who serve the static viewer elsewhere or want the AI process isolated
  (its own credentials, scaling, network policy). The browser points at it via a
  configured base URL.

The WASM build (`cmd/emod-wasm`, `internal/wasm`) is **unchanged** — it gains no
LLM code. It keeps doing exactly what `wasm-architecture.md` describes: parse →
validate → lint → export, in-browser. The AI seam is entirely additive and lives on
the server.

### Capability discovery

On load, `chat.js` probes `GET /ai/health`. If it 200s, the chat panel is shown;
otherwise it stays hidden and the viewer is byte-for-byte the current experience.
This is the degraded-mode contract: **AI is opt-in and absent-by-default.**

## Interface

### Backend route

`POST /ai/edit`

```jsonc
// request
{
  "mode": "edit",                 // or "ask"
  "instruction": "split EmailDecisionsTopicView — it subscribes to too many events",
  "source": "model \"Inbound Customer Comms Agentic Reply\"\n...",  // current .emod
  "export": { "model": { /* JSON from exportJSON, optional but recommended */ } },
  "focus": { "nodeId": "...", "context": "InboundEmail" }            // optional
}
```

```jsonc
// response (mode: edit)
{
  "emod": "model \"Inbound Customer Comms Agentic Reply\"\n...",  // full, repaired
  "diff": "@@ -248,7 +248,18 @@ ...",                              // unified diff
  "diagnostics": [ /* residual warnings only; errors never returned */ ],
  "summary": "Split the topic view into a decisions view and an escalation view.",
  "usage": { "input_tokens": 8123, "output_tokens": 642 }
}
```

```jsonc
// response (mode: ask)
{ "answer": "If identification is deferred, ...", "usage": { ... } }
```

`/ai/health` → `200 {"enabled": true, "model": "anthropic.claude-opus-4-8"}` when
configured; the route is not registered at all otherwise (so the probe 404s and the
panel stays hidden).

The handler reuses the existing request hygiene in `serve.go`: `io.LimitReader`
(raise the cap above `maxBodyBytes` since a full model + export is the payload),
JSON-only, POST-only, `jsonError` for failures. The repair loop is the foundation's
`GenerateAndRepair`, parameterised to take the *current* source as the edit base
rather than generating from empty.

### Chat-panel UX (`internal/viewer/static/chat.js`, new)

A collapsible side panel, mounted next to the existing data/context panels and
toggled like them. New `store.dom` handles (`chatPanel`, `chatLog`, `chatInput`,
`chatSend`) follow the pattern in `store.js`; it speaks only through `bus.js`:

- `bus.emit('chat:submit', { instruction, mode })` — panel → controller.
- The controller gathers `source` + `export` from the store, POSTs `/ai/edit`,
  streams the assistant reply into `chatLog`, and on a completed `edit` shows a diff
  block with **Accept** / **Reject**.
- **Accept** → `bus.emit('chat:apply', { source })`. The handler pushes the current
  source onto an undo stack in `store` and runs `Model.sendParse` → `setModelData`,
  identical to the Render button path in `viewer.js`. Diagnostics route through the
  existing `diagnostics:changed` → `UI.updateDiagnosticsPanel`.
- **Reject** → discard; nothing changes.
- **Undo** (button + `Ctrl/Cmd-Z` when focus isn't in an input) →
  `bus.emit('chat:undo')`, pops the stack, re-parses.

Read-only answers render inline; the model is untouched.

### Concrete exchange

```
You:  rename EmailConversationCustomerIdentificationDeferred to something
      an analyst would understand

emod: I'll rename it to EmailConversationCustomerUnidentified and update the
      three references (the flow, the compose-identification-ask automation's
      activation event, and the EmailDecisionsTopicView subscription).

      [diff]
      - event EmailConversationCustomerIdentificationDeferred {
      + event EmailConversationCustomerUnidentified {
      ...
      - on EmailConversationCustomerIdentificationDeferred
      + on EmailConversationCustomerUnidentified
      ...
      subscribes [..., EmailConversationCustomerUnidentified]

      ✓ parses, validates, lints clean        [ Accept ]  [ Reject ]
```

Because the rename touches an event referenced by an `automation`'s `on` clause
(`EmailConversationComposeIdentificationAsk`), a flow edge, and a `view`'s
`subscribes` list, a single-node inline rename in the current viewer would *not*
catch all three — the validator would then flag dangling references. The repair loop
guarantees the proposal shown already resolves them.

## Worked Example

Model: `examples/inbound_customer_comms_agentic_reply.emod`. The `Publish decision`
slice in the `InboundEmail` context defines a view that subscribes to **four**
events — the linter's "no god views" rule (5+ is the hard cap, 4 is borderline and
the kind of thing a reviewer flags):

```emod
view EmailDecisionsTopicView {
  fields { ... }
  subscribes [EmailConversationReplyInitiated, EmailConversationIdentificationAskInitiated,
              CustomerCaseSubmitted, EmailConversationCustomerIdentificationDeferred]
}
```

**Instruction:** "split `EmailDecisionsTopicView` — it subscribes to too many
events. Separate the escalation/case decisions from the reply decisions."

**Backend repair loop** asks `anthropic.claude-opus-4-8` for the edit grounded on the
full source + export. A first attempt might leave the new view in a slice that
collides with an existing name (validator error) or drop a field the diff still
referenced; `pipeline.Check` returns the diagnostic, the loop feeds it back, and the
second attempt converges to:

```emod
# Reply-path decisions consumed by inbox-router for label/folder ops.
slice "Publish reply decision" {
  view EmailReplyDecisionsTopicView {
    fields {
      topic             string required
      provider          string required
      mailbox           string required
      providerMessageId string required
      decision          string required
      env               string required
      insertMime        string optional
    }
    subscribes [EmailConversationReplyInitiated, EmailConversationIdentificationAskInitiated,
                EmailConversationCustomerIdentificationDeferred]
  }
}

# Escalation/case decisions kept separate so neither view becomes a god view.
slice "Publish escalation decision" {
  view EmailEscalationDecisionsTopicView {
    fields {
      topic             string required
      provider          string required
      mailbox           string required
      providerMessageId string required
      decision          string required
      env               string required
      caseId            UUID   optional
      escalationType    string optional
    }
    subscribes [CustomerCaseSubmitted]
  }
}
```

The original `EmailDecisionsTopicView` is removed; the two replacements each
subscribe to a coherent subset. The backend returns this as a unified diff against
the source plus a one-line summary; the chat panel shows it with Accept / Reject.

**On Accept:** the new `.emod` becomes the working copy; `Model.sendParse` runs it
through `wasm.parseEmod` in the browser; `setModelData` fires `data:changed`; the
`InboundEmail` swim-lane re-renders with two view boxes where there was one, and the
diagnostics badge drops the god-view warning. The whole apply step is the existing
client render path — no server involvement after the proposal is accepted.

## Implementation Plan

**Phase 1 — Backend seam (S).** Add `/ai/edit` and `/ai/health` to
`internal/viewer/serve.go`, registered only when `EMOD_AI_*` config is present
(reuse the foundation's adapter + `GenerateAndRepair`, parameterised on a base
source). Add the `mode: ask` branch (no loop). Unit-test the handler against a mock
`llm.Model` per the foundation's testing note: convergence on a dirty first attempt,
clean pass-through, `ask` non-mutation, error when the loop never converges.

**Phase 2 — Chat panel + apply (M).** New `internal/viewer/static/chat.js`; new
`store.dom` handles and a `chat:*` event family on `bus.js`. Health probe →
show/hide. Submit → POST → render reply. Accept → reuse `Model.sendParse` /
`setModelData`; Reject → discard. This is the smallest end-to-end slice that edits a
model from chat.

**Phase 3 — Diff preview + undo (M).** Server-computed unified diff rendered in the
panel; client-side undo stack in `store` with a button and `Ctrl/Cmd-Z`. Surface
residual warnings inline so the user can knowingly accept a lint-warned edit.

**Phase 4 — Streaming + cost (S).** Stream the assistant reply token-by-token (SSE
or chunked) for `ask` and for the model's pre-repair narration; show
`usage` token counts per turn (the foundation already reports `Response.Usage`, and
the existing `bedrock-cost` tooling models the pricing).

**Phase 5 — Focus & targeted edits (L).** Use `store.interaction.selectedNodeId` /
right-click context as `focus`, and explore *targeted* edits (a structured patch over
the JSON model) as an alternative to full-file replacement — see Open Questions.
Larger because it touches the prompt contract and the apply path.

## Risks & Mitigations

- **Key custody.** A browser must never hold a Bedrock credential. *Mitigation:* the
  LLM call is the only thing that crosses the seam; the credential lives only in the
  Go backend (`llm.Model` config from `EMOD_AI_*`). The WASM build gains no provider
  code. This is the foundation's stated WASM stance.
- **Bad / destructive edits.** A model could propose a structurally wrong or
  data-losing change. *Mitigation:* (1) the repair loop guarantees the *proposal* is
  already valid; (2) nothing applies without explicit **Accept**; (3) a diff is shown
  before apply; (4) undo restores the prior source instantly. The deterministic
  oracle is the safety net, not the model's good intentions.
- **Latency.** `opus-4-8` at high effort plus a multi-attempt repair loop can take
  many seconds. *Mitigation:* stream the reply; show a "thinking / repairing
  (attempt n)" status; bound `maxAttempts`; keep the apply/render path fully client-
  side so accepting feels instant. Consider `anthropic.claude-haiku-4-5` for trivial
  mechanical edits (pure renames) detectable from the instruction.
- **Offline / no backend.** *Mitigation:* the health probe hides the panel; the
  viewer is unchanged. Parse/validate/lint/layout/render all run in WASM and never
  needed the network — only chat does.
- **Cost.** Grounding sends the full source + export each turn. *Mitigation:* send the
  JSON export (compact) rather than re-deriving from prose; report `usage` per turn;
  prefer the cheap model for mechanical passes; bound `maxAttempts`.
- **Prompt-grounding drift.** The browser's working copy could diverge from what the
  backend last saw (e.g. the user dragged/renamed between turns). *Mitigation:* every
  request sends the *current* `source` from the store, and the backend's returned diff
  is computed against exactly that source — the backend is stateless, so there is no
  stale server-side copy to drift from.

## Open Questions

- **Full-file replacement vs targeted patch.** Phases 1–4 use full-file replacement:
  simplest, reuses `Model.sendParse` verbatim, and the diff makes the change legible.
  But for a one-field tweak it sends and regenerates the whole model. A targeted patch
  (structured edit over the JSON model, or a constrained line-range edit) would be
  cheaper and lower-risk, at the cost of a more complex contract and a new apply path.
  Start with full-file; revisit in Phase 5.
- **Where does the diff get computed?** Proposed: server-side (Go has a unified-diff
  generator readily; avoids shipping a JS diff lib into the static viewer). Acceptable
  alternative: client-side if we already pull in a diff utility.
- **Conflict with direct manipulation.** If the user drags nodes (layout-only) or
  inline-renames (model-changing) *after* a proposal is shown but before accepting,
  do we re-validate against the now-current source, or invalidate the stale proposal?
  Leaning toward invalidating any pending proposal on any `data:changed`.
- **How much history?** Multi-turn chat needs the conversation in the request (the
  backend is stateless). Cap turns / token budget, and decide whether to resend full
  source each turn or only deltas.
- **Read-only reuse of 07.** The `ask` branch overlaps [07](./07-talk-to-your-model-qa.md).
  Should `/ai/edit?mode=ask` simply delegate to 07's grounded-Q&A code so there's one
  implementation, or keep a thin in-viewer variant? Prefer delegation once 07 exists.
- **Integrated vs separate backend as the default.** `emod diagram --serve` with an
  `/ai/edit` route is the lowest-friction default; `emod ai serve` is cleaner for
  isolation. Ship the integrated route first; add the standalone command if isolation
  demand is real (YAGNI otherwise, per the foundation).

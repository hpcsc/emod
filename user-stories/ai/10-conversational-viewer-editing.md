# AI: Conversational Viewer Editing

## Overview

Add a chat panel to the emod web viewer (started via `emod diagram --serve`) so that a model author can describe an edit in plain language — "add a refund flow to the `OutboundDelivery` context", "split `EmailDecisionsTopicView`, it subscribes to too many events", "rename `EmailConversationCustomerIdentificationDeferred` to something an analyst would understand" — and receive a concrete `.emod` change to review and apply.

The existing viewer already edits models mechanically: drag a node, rename inline, add a slice from the context menu, delete an element, export back to `.emod`. But those edits are local — one node, one field, one arrow at a time. The valuable changes are structural and span the whole model, and a clumsy edit can break it: emod has a deterministic correctness oracle (parse, validate, lint), and an edit that breaks it is worse than no edit.

Each proposed edit is checked against that oracle before the author ever sees it, shown as a diff to accept or reject, and — once accepted — re-rendered live through the viewer's existing pipeline. The author can also ask read-only questions about the model ("what happens if identification is deferred?") without mutating anything, and undo an applied edit instantly. Crucially, the viewer keeps working exactly as it does today when no AI backend is available: the chat panel is simply hidden, and nothing in the existing view/inspect/drag/export experience depends on it.

## Goals

- Let authors instruct structural edits to a model in natural language from within the web viewer.
- Show every proposed edit as a reviewable diff with explicit accept/reject, never applying anything automatically.
- Guarantee that any edit shown to the author already parses, validates, and lints (only valid models reach the author and only valid models render).
- Re-render the diagram live, in-browser, the instant an edit is accepted.
- Support read-only questions about the model through the same panel without changing it.
- Let authors undo an applied edit and return to the prior model immediately.
- Keep the viewer fully functional with no AI backend present — the chat panel is hidden and every existing feature is unchanged.
- Never require the author to paste a model credential or API key into the browser.

## User Stories

### US-001: Show or hide the chat panel based on backend availability
**Description:** As a model author, I want the chat panel to appear only when an AI backend is reachable so that the viewer is byte-for-byte its current self when no backend is configured.

**Acceptance Criteria:**
- [ ] When the viewer loads and an AI backend is available, a collapsible chat panel is shown alongside the existing data and context panels
- [ ] When no AI backend is available, the chat panel is absent and every existing viewer feature (render, pan, zoom, inspect, drag, inline rename, context-menu add, delete, export `.emod`) works exactly as before
- [ ] The viewer never prompts the author to enter or paste a model credential or API key into the browser
- [ ] Whether the chat panel appears is determined automatically at load, without the author flipping a setting
- [ ] When the chat panel is shown, it can be collapsed and re-expanded like the other side panels

**Context:** Today's viewer is served by `emod diagram --serve` and parses, validates, lints, lays out, and renders `.emod` entirely in the browser, with no network needed after load. AI is opt-in and absent-by-default: only the language-model call requires a backend; everything else stays client-side. The viewer must remain usable for authors who never configure AI.

---

### US-002: Instruct an edit in natural language and see the proposed result
**Description:** As a model author, I want to type an edit instruction into the chat panel and get back a proposed `.emod` change so that I can make structural edits by describing intent instead of clicking element by element.

**Acceptance Criteria:**
- [ ] The author can type a free-text instruction (e.g. "add a refund flow to the `OutboundDelivery` context") and submit it
- [ ] The proposal is generated against the model currently loaded in the viewer, including any edits the author has already made in this session
- [ ] The assistant's reply appears in the chat panel as a message, including a short summary of what the edit does
- [ ] While the proposal is being prepared, the panel shows a clear in-progress indicator so the author knows the request is being worked on
- [ ] If a proposal cannot be produced, the panel shows a clear message and the loaded model is left unchanged
- [ ] Submitting an instruction does not modify the rendered diagram at this stage

**Context:** Instructions of interest are structural and span the whole model, e.g. adding a flow, splitting an over-subscribed view, or a rename that ripples across references. A single proposal may take several seconds; a visible in-progress state matters. The author's working model never leaves the browser except as grounding for the request.

**Depends on:** US-001

---

### US-003: Only valid edits are ever proposed
**Description:** As a model author, I want every proposed edit to already parse, validate, and lint cleanly so that I never have to accept a change that breaks my model.

**Acceptance Criteria:**
- [ ] Any edit shown to the author is a complete `.emod` model that parses without error
- [ ] Any edit shown to the author passes validation with no errors (e.g. no dangling references after a rename)
- [ ] An edit shown to the author carries no blocking lint errors
- [ ] If a valid edit cannot be reached for an instruction, no proposal is shown and the author is told it could not be produced, with the loaded model left unchanged
- [ ] A rename instruction that touches an event referenced from a flow, an `automation` trigger, and a `view`'s `subscribes` list updates all of those references so the proposed model has no dangling references

**Context:** emod has a deterministic correctness oracle: parser, validator, and linter. A naive split might leave a `view` subscribing to an event that no longer exists; a naive rename misses references and produces dangling pointers. The proposal pipeline re-checks each candidate against this oracle and only surfaces one that passes, so the author is never shown a broken model. Lint warnings the author explicitly asked for are handled separately (see US-007).

**Depends on:** US-002

---

### US-004: Review a proposed edit as a diff before applying
**Description:** As a model author, I want to see the proposed change as a diff against my current model so that I can understand exactly what would change before anything is applied.

**Acceptance Criteria:**
- [ ] A proposed edit is shown as a diff highlighting added and removed lines against the model currently loaded in the viewer
- [ ] The diff is shown alongside the assistant's one-line summary of the change
- [ ] The diff is presented for review only — the diagram is not modified and nothing is applied at this stage
- [ ] The diff reflects exactly the model that would become the working copy if the author accepts

**Context:** The diff is computed against the exact model the proposal was based on, so what the author reviews is what they will get. Example: a rename diff shows the `event` declaration changing, the `automation` trigger changing, and the `subscribes` entry changing together.

**Depends on:** US-003

---

### US-005: Accept a proposed edit and see the diagram re-render live
**Description:** As a model author, I want to accept a proposed edit and have the diagram update immediately so that I can see the result of my instruction without leaving the viewer.

**Acceptance Criteria:**
- [ ] An accept control is available on each proposed diff
- [ ] On accept, the proposed model becomes the viewer's working copy and the diagram re-renders to reflect it
- [ ] The re-render uses the viewer's existing in-browser pipeline (the same path as the Render button) and requires no further backend round-trip after acceptance
- [ ] After acceptance, the diagnostics panel reflects the accepted model (e.g. a previously flagged smell is gone)
- [ ] Accepting a "split `EmailDecisionsTopicView`" proposal shows two view boxes where there was one in the relevant context's swim lane, and the god-view warning is dropped from the diagnostics badge

**Context:** The worked example splits an over-subscribed `EmailDecisionsTopicView` (it subscribes to four events; the "no god views" lint rule flags this) into two coherent views. Acceptance reuses the existing client render path so it feels instant; no server is involved once a proposal is accepted.

**Depends on:** US-004

---

### US-006: Reject a proposed edit
**Description:** As a model author, I want to reject a proposed edit so that I can dismiss a change I don't want without affecting my model.

**Acceptance Criteria:**
- [ ] A reject control is available on each proposed diff
- [ ] On reject, the proposal is discarded and the loaded model and rendered diagram are unchanged
- [ ] A rejected proposal can no longer be accepted afterward
- [ ] After rejecting, the author can submit a new instruction in the same chat session

**Depends on:** US-004

---

### US-007: Knowingly accept an edit that carries a residual lint warning
**Description:** As a model author, I want to see any lint warnings that remain on a proposed edit so that I can decide whether to accept a change I deliberately asked for even though it trips a soft rule.

**Acceptance Criteria:**
- [ ] When a proposed edit passes parse and validation but still carries one or more lint warnings (not errors), those warnings are shown inline with the diff
- [ ] Each shown warning identifies the rule it relates to so the author understands what is being flagged
- [ ] The author can still accept the edit despite the residual warnings, and the diagram re-renders with those warnings reflected in the diagnostics panel
- [ ] An edit with blocking errors is never offered for acceptance (those are repaired before the author sees anything, per US-003)

**Context:** Some instructions intentionally produce a model that trips a soft lint rule the author is choosing to live with. Warnings are advisory; errors are not. The author should be able to proceed knowingly on warnings while never being able to apply an error-laden model.

**Depends on:** US-005

---

### US-008: Undo an applied edit
**Description:** As a model author, I want to undo the last accepted edit so that I can revert to my previous model instantly if the result isn't what I wanted.

**Acceptance Criteria:**
- [ ] An undo control is available after an edit has been accepted
- [ ] Undo is also triggerable by the standard keyboard shortcut when focus is not in a text input
- [ ] On undo, the working copy returns to the model that was loaded before the last accepted edit and the diagram re-renders to match
- [ ] Undo requires no backend round-trip and updates the diagram immediately
- [ ] After successive accepted edits, repeated undo steps back through them one at a time
- [ ] After undo, the diagnostics panel reflects the restored model

**Context:** Undo is a client-side restore of the prior working copy and re-parse through the existing in-browser pipeline. It pairs with accept (US-005) as the safety net: a destructive or unwanted edit can always be reversed instantly.

**Depends on:** US-005

---

### US-009: Ask read-only questions about the model
**Description:** As a model author, I want to ask questions about my model in the chat panel and get answers without changing anything so that I can understand the model without reading the source.

**Acceptance Criteria:**
- [ ] The author can ask a read-only question (e.g. "what happens if identification is deferred?") and receive a prose answer in the chat panel
- [ ] Answering a question never modifies the loaded model, the working copy, or the rendered diagram
- [ ] The answer is grounded in the model currently loaded in the viewer
- [ ] A read-only question shows no diff and no accept/reject/undo controls
- [ ] Edit instructions and read-only questions can be intermixed in the same chat session

**Context:** Read-only Q&A is a natural subset of the same panel: it grounds an answer on the current model and returns prose, with no proposal, no diff, and no mutation. This overlaps with the dedicated grounded-Q&A feature; the intent here is the in-viewer read-only subset.

**Depends on:** US-002

---

### US-010: See progress and cost for each turn
**Description:** As a model author, I want each chat turn to stream its reply and report how much it cost so that long requests feel responsive and I can see what I'm spending.

**Acceptance Criteria:**
- [ ] The assistant's reply appears progressively as it is produced rather than only when complete
- [ ] While an edit is being prepared, the panel shows a status that distinguishes preparing the edit from re-checking a failed attempt
- [ ] Each completed turn reports a token-usage figure for that turn
- [ ] Streaming applies to both read-only answers and the narration that precedes a proposed edit
- [ ] If a turn is still in progress, the in-progress state is visible until the reply completes or fails

**Context:** Structural edits at high reasoning effort, combined with the re-check loop behind US-003, can take many seconds; streaming and a clear status keep the panel responsive. Token usage is surfaced per turn so cost is visible.

**Depends on:** US-005, US-009

---

### US-011: Target an edit at the selected element or context
**Description:** As a model author, I want an instruction to apply to the element or context I currently have selected so that I can say "rename this" or "add a slice here" without naming it explicitly.

**Acceptance Criteria:**
- [ ] When the author has a node or context selected (or has right-clicked one) and submits an instruction, the proposal is scoped to that selection
- [ ] An instruction that refers to the selection (e.g. "split this view") produces a proposal that edits the selected element
- [ ] With nothing selected, instructions still work against the whole model as in US-002
- [ ] The proposal still passes parse/validate/lint (US-003) and is shown as a diff with accept/reject (US-004)

**Context:** The viewer already tracks the selected node and the element a context menu was opened on. Passing that as focus lets the author use deictic instructions ("this", "here") instead of repeating element names, and keeps a large model's edits anchored where the author is looking.

**Depends on:** US-005

---

### US-012: Invalidate a pending proposal when the model changes underneath it
**Description:** As a model author, I want a pending proposal to be invalidated if I edit the model directly before accepting it so that I can never apply a change computed against a stale model.

**Acceptance Criteria:**
- [ ] If the author changes the model (e.g. inline rename, add, or delete) while a proposal is pending, the pending proposal is marked stale and can no longer be accepted
- [ ] A stale proposal is clearly indicated as no longer applicable in the chat panel
- [ ] After a direct edit invalidates a proposal, the author can submit a new instruction that is grounded on the now-current model
- [ ] Layout-only changes that do not alter the model (e.g. dragging a node's position) do not invalidate a pending proposal

**Context:** Chat is additive; direct manipulation (drag, inline rename, delete) stays available. A proposal is computed against the model at submit time; if the underlying model changes before acceptance, the diff no longer matches reality, so the proposal must be retired rather than applied against a model it wasn't built for. Pure layout moves don't change the model and so are exempt.

**Depends on:** US-005

---

## Non-Goals (Out of Scope)

- Generating a model from scratch in the CLI or in batch (covered by the NL-to-model generation feature).
- A standalone grounded-Q&A feature; only the in-viewer read-only subset is in scope here.
- An MCP transport or tool-host integration.
- Replacing direct manipulation; drag, inline rename, context-menu add, and delete remain and are additive to chat, not replaced by it.
- Persisting models or chat history on a server, or any multi-user collaboration; the model lives in the browser and the backend is stateless.
- Requiring or accepting a model credential or API key in the browser.
- Undo or history that survives beyond the current browser session.

## Open Questions

- **Full-file vs targeted edits.** The initial approach replaces the whole model with the repaired version and shows a diff for legibility. A one-field tweak still regenerates the whole model. A targeted, structured patch would be cheaper but adds contract and apply-path complexity. Assumption: start with full-model replacement; revisit targeted patches later (US-011 is the natural place).
- **Where the diff is computed.** The diff can be produced server-side (avoids shipping a diff utility into the static viewer) or client-side if one is already present. Assumption: server-computed diff against the exact submitted model, so the review reflects what will be applied.
- **Conflict policy for direct edits during a pending proposal.** US-012 assumes any model-changing direct edit invalidates a pending proposal, while pure layout drags do not. Open whether some edits should instead trigger a re-check against the now-current model rather than invalidation.
- **Conversation history and re-grounding.** Multi-turn chat needs prior turns carried in each request because the backend is stateless. Assumption: cap the number of turns and the token budget; open whether to resend the full model each turn or only changes.
- **Reuse of the grounded-Q&A feature.** The read-only branch (US-009) overlaps the dedicated grounded-Q&A feature. Assumption: keep a thin in-viewer variant now and prefer delegating to the shared grounded-Q&A implementation once it exists.
- **Cheaper model for trivial mechanical edits.** Pure renames could use a faster, cheaper model than full structural edits. Assumption: not differentiated initially; revisit if cost or latency on mechanical passes proves material.
- **Integrated vs separate backend.** The AI route can live inside `emod diagram --serve` or in a standalone serve command for isolation. Assumption: ship the integrated route first; add the standalone option only if isolation demand is real.

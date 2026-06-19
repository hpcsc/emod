# AI: DCB Modeling Assistant

## Overview

Dynamic Consistency Boundary (DCB) modeling is where the conceptual load in emod is highest. Aggregate-style modeling hands the author a default boundary; DCB takes it away. The author must decide, per event, which entities and relationships it keys on (its `tags`), and, per command, exactly which event types and tag predicate define its consistency boundary (its `decides_on`). emod's linter already detects the DCB failure modes after the fact (`dcb/untagged-event`, `dcb/query-too-broad`, `dcb/single-tag-everywhere`, `dcb/orphan-tag-key`), but it is a judge, not a teacher: it reports that an event is untagged or a query is too broad without proposing which tag key to add, which field it should reference, or which narrower predicate still protects the invariant.

This feature turns each `dcb/*` finding into a concrete, applicable, advisory suggestion, and assists whole-context aggregate-to-DCB conversion. Every suggestion is grounded in DCB semantics and the actual model, and is only offered after `emod validate` plus `emod lint` confirm it parses, validates, and does not reintroduce or worsen a `dcb/*` finding. The deterministic pipeline is the authority; the model proposes, and the author reviews, edits, or rejects. Nothing is applied silently.

This feature assumes the shared AI foundation (the generate → validate → lint → repair loop) and consumes the DCB DSL and lint rules exactly as defined; it does not change DCB semantics, the grammar, or the rules.

## Goals

- Turn each `dcb/*` finding into a concrete, applicable suggestion: a `tags` block for an untagged event, a narrower `decides_on` for a too-broad query, a routing-or-removal fix for an orphan tag key, and an additional tag dimension for single-tag-everywhere.
- Assist whole-context aggregate-to-DCB conversion: take an `aggregate`-mode context and produce a `mode dcb` rewrite with tagged events and tag-scoped decisions, kept valid by the repair loop.
- Ground every suggestion in DCB semantics and the real model, so suggestions use real declared field names and respect declared tag keys.
- Use `emod validate` and `emod lint` (especially the `dcb/*` rules) as the oracle: a suggestion is surfaced only if applying it parses, validates, and does not reintroduce or worsen a DCB finding.
- Keep all suggestions advisory: the author reviews and accepts, with nothing applied without explicit opt-in.

## User Stories

### US-001: Suggest tags for an untagged event
**Description:** As a model author, I want emod to suggest a `tags` block for an event flagged by `dcb/untagged-event` so that I can give the event a routing key without inventing it from scratch.

**Acceptance Criteria:**
- [ ] Running the suggest command on a file with a `dcb/untagged-event` finding returns a tag suggestion for the named event, anchored to that finding's location
- [ ] Each suggested tag is a `key: fieldRef` pair where `fieldRef` is a field actually declared on the target event; a suggestion referencing a non-declared field is never shown
- [ ] Each suggestion includes a one-or-two-sentence rationale explaining why the key and field were chosen
- [ ] A suggestion is surfaced only after applying it parses, validates, and clears `dcb/untagged-event` on that event without introducing a new `dcb/*` finding on it
- [ ] When the enclosing context already uses tag keys (for example `entity`, `category`), the suggestion reuses an existing key where it fits rather than inventing a new one
- [ ] By default the suggestion is printed (text or machine-readable form) and the file is left unchanged

**Context:** In a `mode dcb` (or `mixed`) context every event must declare at least one tag, expressed as `tags { key: fieldRef }`. The rationale exists because tag choice carries domain judgment; the author must be able to accept, edit, or reject it.

### US-002: Suggest a narrower decision query for a too-broad command
**Description:** As a model author, I want emod to suggest a narrower `decides_on` for a command flagged by `dcb/query-too-broad` so that the command's consistency boundary reads only the events it truly needs.

**Acceptance Criteria:**
- [ ] Running the suggest command on a file with a `dcb/query-too-broad` finding returns a narrow-query suggestion for the named command, anchored to that finding's location
- [ ] The suggestion states the event types to keep and a `where` predicate of the form `tag(key = fieldRef)` (optionally combined with `and`, `or`, `not`)
- [ ] Every `tag(key = ...)` in the suggested predicate resolves to a tag key declared on at least one event referenced in the kept `events [...]` list
- [ ] A suggestion is surfaced only after applying it parses, validates, and clears `dcb/query-too-broad` on that command without introducing a new `dcb/*` finding
- [ ] When the original `decides_on` has no `where` clause, the suggestion adds one rather than only trimming the event list
- [ ] Each suggestion includes a rationale describing what the narrower boundary now protects

**Context:** A command's `decides_on` query *is* its consistency boundary. `dcb/query-too-broad` fires when a query references more than five event types or has a missing or always-true predicate. The boundary should be the narrowest set of event types and tag predicate that still protects the invariant the command guards.

**Depends on:** US-001

### US-003: Preview suggestions as a diff or apply them on request
**Description:** As a model author, I want to see suggestions as a preview and choose to write them back only when I decide to, so that I stay in control of every change to my model.

**Acceptance Criteria:**
- [ ] By default the suggest command prints the suggestions and leaves the source file untouched
- [ ] A machine-readable output mode emits the full list of suggestions (kind, target, rule, proposed tags or query, rationale) for tooling to consume
- [ ] An explicit apply option writes the accepted suggestions back into the file in correctly formatted DSL
- [ ] After applying, re-running the linter shows the addressed `dcb/*` findings are gone and no new findings were introduced
- [ ] When more than one finding exists, the output reports a count and reminds the author how to apply or get machine-readable output
- [ ] The AI feature is opt-in: with no AI credentials configured the suggest command is simply unavailable, and the existing validate, lint, export, diagram, and lsp paths gain no AI dependency

**Context:** This is the cross-cutting interaction contract shared by all suggestion kinds: advisory by default, opt-in, reviewed, and never silently applied. Rendering produces syntactically correct DSL so the author never sees malformed output.

**Depends on:** US-001, US-002

### US-004: Filter suggestions by rule or target
**Description:** As a model author working through a large model, I want to limit suggestions to a single rule or a single event or command so that I can focus on one concern at a time.

**Acceptance Criteria:**
- [ ] A rule filter restricts suggestions to findings of one named `dcb/*` rule (for example only `dcb/untagged-event`)
- [ ] A target filter restricts suggestions to a single named event or command
- [ ] Rule and target filters can be combined, returning only findings that match both
- [ ] When a filter matches no findings, the command reports that clearly rather than producing an empty or confusing result
- [ ] Filtered runs still apply the same verification: only suggestions that parse, validate, and clear their finding are shown

**Depends on:** US-001, US-002

### US-005: Resolve an orphan tag key by routing or removal
**Description:** As a model author, I want emod to suggest a fix for a tag key flagged by `dcb/orphan-tag-key` so that every declared tag key either routes a decision or is removed.

**Acceptance Criteria:**
- [ ] Running the suggest command on a file with a `dcb/orphan-tag-key` finding returns a suggestion for the orphaned key, anchored to that finding's location
- [ ] The suggestion offers either a command whose `decides_on` would route on the orphan key, or a recommendation to remove the key from the events that declare it
- [ ] When a routing fix is proposed, the proposed `decides_on` predicate references the orphan key via `tag(key = fieldRef)` and resolves against a referenced event
- [ ] A suggestion is surfaced only after applying it parses, validates, and clears `dcb/orphan-tag-key` for that key without introducing a new `dcb/*` finding
- [ ] Each suggestion includes a rationale, and where both a routing fix and a removal are defensible, both options are presented for the author to choose

**Context:** `dcb/orphan-tag-key` fires when a tag key is declared on events but never referenced by any command's `decides_on`, so it routes nothing. It has two legitimate fixes (route it, or delete it), and which is right depends on intent the author holds.

**Depends on:** US-001, US-002

### US-006: Introduce an additional tag dimension for single-tag-everywhere
**Description:** As a model author, I want emod to suggest an additional tag key when a context trips `dcb/single-tag-everywhere` so that my DCB model expresses more than one routing concern instead of behaving like aggregates with extra ceremony.

**Acceptance Criteria:**
- [ ] Running the suggest command on a context with a `dcb/single-tag-everywhere` finding returns an extra-tag-key suggestion proposing a second routing dimension
- [ ] The proposed tag key maps to a field actually declared on the events it would be added to; a field that is not declared is never proposed
- [ ] The suggestion identifies which events should carry the new tag key and which command(s) could decide against it
- [ ] A suggestion is surfaced only after applying it parses, validates, and clears `dcb/single-tag-everywhere` without introducing a `dcb/orphan-tag-key` or other new `dcb/*` finding
- [ ] Each suggestion includes a rationale explaining the distinct concern the new tag dimension captures

**Context:** `dcb/single-tag-everywhere` fires when every command in a context keys on the same single tag key. The fix is a genuinely distinct second routing concern, not decoration; an invented key that nothing decides against would itself trip `dcb/orphan-tag-key`, which the verification step must reject.

**Depends on:** US-001, US-002

### US-007: Convert an aggregate-mode context to DCB
**Description:** As a model author, I want emod to rewrite a chosen `aggregate`-mode context as `mode dcb` so that I can adopt DCB on an existing context without hand-translating every event and command.

**Acceptance Criteria:**
- [ ] Running the convert command on a named context that is currently `mode aggregate` produces a `mode dcb` rewrite of that context with tagged events and tag-scoped `decides_on` decisions
- [ ] The conversion is only presented as complete when the rewrite parses, validates, and is free of `dcb/*` warnings — not merely when it parses
- [ ] The command reports its progress across repair attempts, including which `dcb/*` diagnostics were outstanding on each attempt
- [ ] The result is shown as a unified diff of the context, and the source file is left unchanged unless apply is explicitly requested
- [ ] If the rewrite does not reach a clean state within the bounded number of attempts, the command returns the best attempt plus the remaining diagnostics for the author to finish by hand, rather than failing silently or writing a non-converged result
- [ ] Token usage for the conversion is reported back to the author
- [ ] Only the named context is converted; other contexts in the file are left unchanged

**Context:** Conversion uses the foundation's repair loop with the `dcb/*` rules as the convergence oracle, so the rewrite is self-correcting against exactly the anti-patterns a hand conversion would fall into (untagged events, too-broad queries, single-tag-everywhere, orphan tag keys). Conversion carries more domain ambiguity than per-finding suggestions, so it runs at higher reasoning effort and always shows a full diff for review.

**Depends on:** US-001, US-002, US-005, US-006

### US-008: Surface DCB suggestions as editor quick-fixes
**Description:** As a model author using an editor, I want DCB suggestions to appear as quick-fixes on the corresponding `dcb/*` diagnostics so that I can apply a fix without leaving my editor.

**Acceptance Criteria:**
- [ ] A `dcb/*` diagnostic in the editor offers a code action that proposes the corresponding suggestion (for example "fix untagged event") anchored to that diagnostic's position
- [ ] Accepting the code action applies the same correctly formatted change the apply path would write
- [ ] The code action's title and rationale match the suggestion shown by the command-line suggest path
- [ ] Code actions are offered only for `dcb/*` diagnostics that have a verified suggestion; a finding with no surfaced suggestion offers no action

**Context:** This reuses the same verified suggestions produced for the command-line path, surfaced through the editor's diagnostic-anchored code-action channel. The editor wiring itself is owned by the lint quick-fixes feature; this story covers exposing the DCB-specific suggestions through it.

**Depends on:** US-003

## Non-Goals (Out of Scope)

- General modeling-smell review (past-tense events, god-views, state obsession). That belongs to the semantic model reviewer feature.
- Building the generic editor code-action plumbing that surfaces any deterministic finding as a quick-fix; this feature reuses that channel for DCB suggestions but does not build it.
- Changing DCB semantics, the grammar, or the lint rules. This consumes the DSL and the `dcb/*` rules exactly as defined.
- Generating runtime DCB infrastructure (event store, append-condition checking). emod describes the model; runtime is out of scope.
- Auto-applying suggestions or conversions without explicit author opt-in.
- Reasoning across multiple contexts at once: suggestion and conversion stay within a single context for now (see Open Questions).

## Open Questions

- **Multi-field tag inference.** When several fields could each key a tag (for example `customerId`, `orderId`, `orderType`), should the assistant propose all defensible keys ranked, or commit to one? Assumption: present the most defensible reading with a rationale and let the author add more, favoring signal over noise.
- **Predicate richness.** The predicate grammar today is `tag(key = fieldRef)` with `and`/`or`/`not`. If it later grows set membership (for example `tag(course in courseIds)`), the narrow-query suggestion must follow. Assumption: cover only the current grammar for now.
- **Cross-context boundaries.** A `decides_on` referencing events from another context is a real DCB pattern but stresses tag-key resolution. Assumption: both suggest and convert stay within one context initially; cross-context reasoning is deferred.
- **Conversion fidelity vs. snapshot semantics.** Aggregate desugaring can lose snapshot or projection meaning a team attached to an aggregate. Assumption: convert proceeds with a tag-based rewrite; whether it should warn when it detects reliance on snapshot semantics is left open.
- **Removal vs. routing for an orphan key.** Both are valid fixes and the right choice needs author intent. Assumption: present both options when each is defensible rather than picking one silently.

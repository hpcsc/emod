# AI: Model Import / Reverse-Engineering

## Overview

Most teams that would benefit from event modeling already run the system being modeled — its events, commands, projections, and external integrations are scattered across source code, an eventcatalog site, an AsyncAPI spec, or the platform's `*.emlang.yaml` files. Writing a `.emod` by hand for that system means re-deriving, by reading artifacts, what those artifacts already state.

This feature adds `emod ai import`, a command that points emod at an existing system and produces a *draft* `.emod` that already parses, validates, and has been run past the linter. It supports two input families — source code (event/command classes, handlers, topic definitions, projections) and structured artifacts (eventcatalog, AsyncAPI, `*.emlang.yaml`, JSON catalogs) — detects the family automatically, and maps detected concepts onto emod constructs (`context`, `aggregate` vs DCB, `command`/`event`/`flow`, `view`, `automation`, `translation`). The output is explicitly draft-assist: a faithful starting point for human review, not a verified model. Inherited naming smells from the source are surfaced by the linter rather than silently rewritten.

This is an opt-in AI feature. Without Bedrock credentials configured, `emod ai` is simply unavailable, and nothing in `validate`/`lint`/`fmt`/`diagram`/`export`/`lsp` gains a hard LLM dependency.

## Goals

- Turn an existing artifact (or a directory/glob of source) into a draft `.emod` with one command: `emod ai import <path-or-glob> -o model.emod`.
- Support both input families teams actually have: structured artifacts (eventcatalog, AsyncAPI, `*.emlang.yaml`, JSON catalogs) and source code.
- Auto-detect the input family, with an explicit `--from code|eventcatalog|asyncapi|emlang|catalog|auto` override.
- Map detected events, commands, projections, handlers, and external integrations onto the correct emod constructs, choosing aggregate vs DCB style appropriately.
- Always emit a draft that parses, validates, and is lint-clean of structural errors before it reaches the user.
- Keep inherited naming smells visible: the linter reports them on the draft instead of the import silently renaming source concepts.
- Set clear expectations that the output is an inferred draft for human review, not a verified model.

## User Stories

### US-001: Import a structured artifact into a draft model
**Description:** As a model author with an existing eventcatalog, AsyncAPI spec, `*.emlang.yaml`, or JSON event catalog, I want to run `emod ai import` against it and get back a draft `.emod`, so that I do not have to translate a catalog I already maintain into emod by hand.

**Acceptance Criteria:**
- [ ] Running `emod ai import <path>` against a recognized structured artifact writes a `.emod` draft to stdout, or to the path given by `-o, --output <file>`
- [ ] The draft contains a `model` and at least one `context`, with the events and commands present in the source artifact represented as `event` and `command` constructs
- [ ] The written draft parses and passes `emod validate` with no errors
- [ ] Importing the source `*.emlang.yaml` for `examples/inbound_customer_comms_agentic_reply.emod` produces a draft whose contexts, events, and commands match the bundled example
- [ ] Running the command without Bedrock credentials configured reports that the AI feature is unavailable and exits without attempting an import
- [ ] The draft begins with a provenance header naming the command and source, and a line marking it as a DRAFT inferred from the source

**Context:** Structured families already carry explicit event/command/channel definitions, so they need no heuristics — the highest signal-to-effort starting point. The bundled `examples/inbound_customer_comms_agentic_reply.emod` was hand-translated from a platform `*.emlang.yaml` and is a known-good round-trip target to validate against.

### US-002: Choose the input family explicitly
**Description:** As a model author whose inputs are ambiguous or whose files do not follow the expected naming, I want to tell `emod ai import` which family to treat the input as, so that detection mistakes do not derail my import.

**Acceptance Criteria:**
- [ ] `emod ai import <path>` with no `--from` flag detects the family automatically: `*.emlang.yaml` → emlang, an `asyncapi` document → asyncapi, an eventcatalog tree → eventcatalog, a JSON event/command catalog → catalog, and source files → code
- [ ] `--from code|eventcatalog|asyncapi|emlang|catalog|auto` overrides the detected family
- [ ] The command reports which family it is treating the input as before doing work
- [ ] Passing `--from` with a value that does not match the actual input produces a clear error rather than a malformed draft
- [ ] When auto-detection cannot confidently classify the input, the command reports this and tells the user to pass `--from` explicitly

**Depends on:** US-001

### US-003: Map detected concepts onto emod constructs
**Description:** As a model author, I want detected events, commands, handlers, projections, and integrations rendered as the idiomatic emod construct for each, so that the draft reads like a model someone wrote on purpose rather than a flat dump.

**Acceptance Criteria:**
- [ ] A cohesive module/service/package becomes a `context`
- [ ] A command (or command-handler input) becomes a `command` and a domain event becomes an `event`, each grouped under a `slice`
- [ ] A handler that reacts to an event and dispatches a command becomes an `automation` with a `trigger` and `command`; when it dispatches into another context it sets `target context`
- [ ] A projection / read model becomes a `view` with a `subscribes [...]` list recovered from the events it handles
- [ ] An event-to-command edge within a context produces a `flow { command -> event }`
- [ ] Source `Event`/`Command` type suffixes are dropped while the underlying name is otherwise preserved (e.g. `IssueInvoiceCommand` → `IssueInvoice`, `InvoiceIssuedEvent` → `InvoiceIssued`)
- [ ] The written draft passes `emod validate` with no errors

**Context:** The proposal's mapping table is the source of truth for these rules. This story covers the default aggregate-style mapping; DCB-style mapping and external-system translation are separate stories (US-004, US-005).

**Depends on:** US-001

### US-004: Map external integrations to translations
**Description:** As a model author whose system talks to external systems, I want outbound integrations and inbound external events represented as `translation` constructs, so that the draft shows where my model meets systems it does not own.

**Acceptance Criteria:**
- [ ] An outbound integration or webhook detected in the source becomes a `translation` with an `external_system "..."`
- [ ] An event that originates from outside the model is rendered with a `source external "..."`
- [ ] Cross-context delivery wiring (e.g. a topic produced in one context and consumed in another) is rendered either as an `automation` with `target context` or as a `translation`, consistent with the mapping rules
- [ ] The translations in the written draft reference only contexts and events that exist in the draft, and the draft passes `emod validate`

**Depends on:** US-003

### US-005: Default to aggregate style, infer DCB only on strong signal
**Description:** As a model author, I want the draft to use the readable aggregate style by default and reach for DCB (`mode dcb`) only when the source clearly calls for it, so that the draft is easy to review and does not invent cross-cutting boundaries that are not there.

**Acceptance Criteria:**
- [ ] An event with a single clear owner and id key is mapped to an `aggregate` (`context → aggregate → slice`) by default
- [ ] An event read by several decisions and keyed by more than one identifier is mapped using `mode dcb` with `tags` and `decides_on`
- [ ] When the draft contains DCB constructs, it is clean of `dcb/untagged-event`, `dcb/query-too-broad`, `dcb/single-tag-everywhere`, and `dcb/orphan-tag-key` findings, or those findings appear in the final lint report
- [ ] An import where no DCB signal is present produces a draft that contains no `mode dcb` contexts
- [ ] The written draft passes `emod validate`

**Context:** Aggregate style is the idiomatic, most readable form and matches most code (one handler class, one stream key). DCB suits an event multiple decisions read across more than one identifier. The DCB lint rules keep any DCB constructs honest during repair.

**Depends on:** US-003

### US-006: Force a single bounded context
**Description:** As a model author whose real domain boundaries do not follow the source's directory/package structure, I want to force the whole import into one named context, so that I do not get a draft fragmented along the wrong lines.

**Acceptance Criteria:**
- [ ] `emod ai import <path> --context <name>` places all detected events, commands, views, and automations under a single context with the given name
- [ ] Auto-bucketing into multiple contexts is skipped when `--context` is set
- [ ] The forced context name appears as the `context` name (and informs the `model` name) in the written draft
- [ ] The written draft passes `emod validate`

**Context:** Context grouping is otherwise inferred from module/package and directory proximity, which is a heuristic that real domains do not always follow.

**Depends on:** US-003

### US-007: Always emit a clean draft via validate and repair
**Description:** As a model author, I want the import to fix its own structural mistakes before writing the file, so that I never receive a draft that fails to parse or validate.

**Acceptance Criteria:**
- [ ] When the initial mapping produces validation errors (e.g. a `subscribes` listing an unknown event, an `automation` targeting a missing context, a malformed `decides_on`), the import attempts to correct them before writing
- [ ] `--max-attempts <n>` controls how many correction attempts are made (default 4)
- [ ] The command reports its repair progress, including the number of validation errors found and resolved per attempt
- [ ] On success, the written draft passes `emod validate` with no errors
- [ ] If the draft cannot be made to validate within the attempt limit, the command reports that it did not converge and exits with a non-zero status rather than writing a broken draft

**Context:** Reuses the validate → repair loop established for 01 (NL → model generation). Structural invention (a flow referencing a non-existent event) is caught here; plausible-but-wrong inference is not, which is why the output stays a reviewable draft.

**Depends on:** US-003

### US-008: Surface inherited naming smells in lint
**Description:** As a model author, I want the import to keep the source's naming smells visible by reporting them, not by silently renaming concepts, so that I can see what the source system's naming actually says and decide what to clean up.

**Acceptance Criteria:**
- [ ] After writing the draft, the command runs the standard `emod lint` over it and prints the findings
- [ ] A source event like `OrderUpdatedEvent` is rendered as `OrderUpdated` and reported as `state-obsession`, not renamed away
- [ ] Property-sourced names (e.g. `Order` + `Status` + `Changed`) trip `property-sourcing`, a `*Initiated` event trips `command-in-disguise`, a command in past tense trips `command-past-tense`, a projection missing the `View` suffix trips `view-naming`, a fan-out projection over many events trips `god-view`, and a single-id event trips `clickbait-event` — each reported, not rewritten
- [ ] The lint report identifies each finding by rule name, location in the draft, and the construct it concerns
- [ ] The command does not modify construct names to make lint findings disappear

**Context:** Inherited smells are facts about the source system worth showing a human. Guided cleanup of these findings is a separate feature (04 — lint quick-fixes), on the user's terms.

**Depends on:** US-007

### US-009: Inspect the condensed brief before spending a model call
**Description:** As a model author, I want to preview what the import detected and how it grouped it before any model call runs, so that I can catch wrong boundaries or missed signals early and cheaply.

**Acceptance Criteria:**
- [ ] `emod ai import <path> --dry-run` prints the condensed model brief — detected events, commands, handlers, projections, and the contexts they were grouped into — and exits without writing a draft
- [ ] `--dry-run` makes no mapping or repair model call
- [ ] The command reports counts of detected events, commands, handlers, and projections, and the number and names of the bounded contexts they were condensed into
- [ ] The brief retains source provenance (file:line) for detected items

**Context:** A cheap inspection point lets a user correct boundaries (e.g. via `--context`) before incurring the cost of the expensive mapping pass.

**Depends on:** US-003

### US-010: Extract signals from source code
**Description:** As a model author whose system has no catalog, only source code, I want `emod ai import --from code` to detect events, commands, handlers, projections, topics, and external calls directly from the code, so that I can get a draft model without first authoring an artifact.

**Acceptance Criteria:**
- [ ] `emod ai import <path-or-glob> --from code` resolves the glob, detects candidate events, commands, handlers, projections, topics, and external integrations, and produces a draft
- [ ] Detection is loose: types named `*Event` / past-tense suffixes / domain-event annotations are treated as events; `*Command` / command-handler inputs as commands; event-subscribed dispatching methods as automations; `*Projection` / `*ReadModel` / `*View` folds as views
- [ ] Topic/channel and outbound HTTP/SDK/webhook signals are detected and surface as cross-context delivery or external translations
- [ ] Framework, test, and config code is filtered out of the detected signals rather than mapped into the draft
- [ ] The draft is grounded only in detected signals — events, commands, fields, and flows that do not appear in the source are not introduced — and the written draft passes `emod validate`
- [ ] At least one source language (Go) is supported end-to-end against the worked example in the proposal (a billing service event, command handler, Kafka listener, and projection)

**Context:** This is the largest and least certain extraction path, deliberately shipped behind the already-working structured-artifact path. Extraction pulls compact signals (name, fields where cheaply recoverable, provenance, hints, module) rather than feeding raw files to the model.

**Depends on:** US-003, US-009

### US-011: Condense large inputs without losing the inventory
**Description:** As a model author with a large codebase, I want the import to stay within budget by condensing per bounded context, so that a big system still produces a draft instead of failing or dropping events.

**Acceptance Criteria:**
- [ ] An input whose detected signals exceed the budget is grouped into per-context buckets, each bucket summarized into a compact inventory rather than truncated
- [ ] The condensed brief preserves the event and command inventory — every detected event and command is represented across the assembled per-context summaries
- [ ] A very large input degrades to per-context drafts rather than failing outright
- [ ] The command reports when condensing/summarization was applied and to which contexts
- [ ] The written draft passes `emod validate`

**Context:** A real codebase is far larger than any context window. Condensing buckets signals by module/package and directory proximity, emits a per-bucket inventory, and assembles a single bounded brief for the mapping pass; oversized buckets are summarized rather than cut.

**Depends on:** US-010

## Non-Goals (Out of Scope)

- Greenfield generation from prose — that is feature 01 (NL → model generation). This feature starts from artifacts describing an existing system.
- A guaranteed-correct extraction. The output is draft-assist for human review, not an authoritative model; reverse-engineering infers intent (which events belong together, where a boundary sits) that the source never states.
- Round-tripping `.emod` → code, or keeping a model continuously in sync with a codebase. Import is one-shot; re-running overwrites the draft.
- A language-agnostic semantic parser. Source extraction pulls signals with cheap heuristics; it does not build a full AST analyzer per language.
- Auto-fixing inherited naming smells. The linter surfaces them; guided cleanup is feature 04 (lint quick-fixes).
- Three-way merge of an edited draft against changed source — overwrite only.
- Re-specifying the LLM port, adapter, or repair loop (covered by the shared foundation and feature 01).

## Open Questions

- **DCB granularity for `--from code` (assumption made):** This document assumes aggregate-by-default for source-code imports, with DCB emitted only on a strong multi-tag signal (US-005), rather than an opt-in flag. If reviewers prefer DCB be strictly opt-in for code imports, US-005 splits into a default-off path plus a flag.
- **Multi-language repos:** When one import spans, e.g., Go and TypeScript services, should it run as one combined model with shared contexts, or one draft per service stitched together? Assumed out of scope for the initial source-code path; US-010 targets a single language end-to-end first.
- **Provenance in the output (assumption made):** This document assumes the draft carries a provenance header (US-001) but does not assume per-construct `file:line` comments in the `.emod`; whether those appear inline, behind a `--with-provenance` flag, or only in the dry-run brief is left open.
- **Confidence signalling:** Should low-confidence mappings (e.g. an automation inferred from a loosely-matched topic name) be flagged in the draft so reviewers know where to look first? Not assumed in scope; could extend US-008's reporting.
- **Effort override surface:** The proposal lists an `--effort` flag to override model effort for the mapping pass. It is assumed available wherever model calls run but is not given its own story, as it is a cross-cutting cost/quality control rather than user-facing behavior.

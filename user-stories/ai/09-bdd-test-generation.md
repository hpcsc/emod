# AI: BDD Scenario & Test-Fixture Generation

## Overview

An `.emod` model already encodes a behavioural specification, but it is trapped in a form no test harness can run. Every command slice already says: GIVEN some prior events exist, WHEN a command arrives carrying typed `fields`, THEN one or more events result — a Given/When/Then triple in everything but name. Teams re-author those triples by hand in `.feature` files, table-driven tests, and JSON fixtures, and from then on the spec and the model drift apart: the model says `Classify message` can emit either `EmailConversationClassified` or `EmailConversationEscalationDetected`, but the suite, written weeks later, covers only the happy path.

This feature adds an `emod ai testgen` command that turns a model's slices into executable-style specs and fixtures. The scenario skeleton — which events are the precondition, which command is the trigger, which events are the possible outcomes — is fully determined by the model's `flow`, `decides_on`, and upstream automation chain, so emod assembles it deterministically; the model is asked only to fill realistic values, name scenarios in business language, and surface the implied edge and branch cases. Two hard checks keep the output trustworthy: every generated payload is validated against the field-type schema derived from the model, and every scenario's events must correspond to a `flow` (or `decides_on`/automation chain) that exists in the model. The output deliverable is the product here: Given/When/Then scenarios, Gherkin `.feature` files, and JSON sample payloads that conform to the typed `fields`. The feature is opt-in and absent without configured credentials; it adds no dependency to the existing `validate`/`lint`/`diagram`/`export`/`lsp` paths.

## Goals

- Turn a model's slices into Given/When/Then scenarios and conforming fixtures via `emod ai testgen <file.emod> -o <dir>`.
- Emit one happy-path scenario per `flow` edge and one branch scenario per additional `flow` edge, so a command with two outcome events yields two scenarios, not one.
- Generate schema-conformant sample payloads for every command and event, honouring the typed `fields` exactly (`UUID` fields get UUIDs, `timestamp` fields get RFC 3339 timestamps, `number` fields get numbers, `required` fields are always present, `optional` fields are sometimes omitted).
- Emit output in the formats real harnesses consume — Gherkin `.feature` files and/or JSON fixtures — selectable via `--format gherkin|json|both`.
- Surface the negative and implied cases the model gestures at (missing-required-field rejections, the classification→escalation branch, the identification-deferred branch) rather than only the happy path.
- Keep every generated scenario faithful to the model: the model names and fills, it does not invent structure.
- Scope generation to a single slice during iteration via `--slice <name>`, and report token cost every run.

## User Stories

### US-001: Generate Given/When/Then scenarios from a model's slices
**Description:** As a model author, I want to run `emod ai testgen <file.emod>` and get a Given/When/Then scenario for each slice's flow, so that the behaviour my model encodes becomes runnable spec material instead of staying trapped in the DSL.

**Acceptance Criteria:**
- [ ] `emod ai testgen <file.emod>` produces, for each command slice, at least one scenario whose WHEN is the slice's `command`, whose THEN is an `event` the command points to via a `flow` edge, and whose GIVEN is the slice's precondition events
- [ ] A command with two `flow` edges (e.g. `EmailConversationRecordClassification -> EmailConversationClassified` and `-> EmailConversationEscalationDetected`) produces two scenarios, one per outcome event
- [ ] The GIVEN, WHEN, and THEN of every scenario name only commands and events that exist in the model
- [ ] By default output is written to stdout; `-o <dir>` writes the generated artifacts into the named directory instead
- [ ] On success the command exits 0 and reports a summary (e.g. slice count and total scenario count)
- [ ] When AI credentials are not configured, `emod ai testgen` reports that the feature is unavailable and exits non-zero, while `emod validate`, `emod lint`, `emod diagram`, and `emod export` continue to work unchanged

**Context:** This is the foundational slice — get a faithful scenario skeleton (GIVEN/WHEN/THEN built from `flow`, `decides_on`, and the automation chain) produced for each slice, with the opt-in/no-credentials boundary respected. The branch case is structural, not invented: it falls directly out of the number of `flow` edges on the command. Realistic payloads, output formats, and edge-case enrichment build on top of this. testgen reads the grammar exactly as it is and invents no DSL surface.

---

### US-002: Fill scenario payloads that conform to the typed field schema
**Description:** As a model author, I want every command and event payload in a generated scenario to match the typed `fields` declared in my model, so that the fixtures are usable as-is and never carry a value of the wrong type.

**Acceptance Criteria:**
- [ ] Each field in a generated payload carries a value of the type its `fields` declaration specifies: `UUID` fields a syntactically valid UUID, `timestamp` fields an RFC 3339 date-time, `date` fields an RFC 3339 date, `number` fields a number, `bool` fields a boolean, `string` fields a string, `list` fields an array
- [ ] Every `required` field on each command and event in the scenario is present in the payload
- [ ] `optional` fields are sometimes present and sometimes omitted across the generated scenarios rather than always one or the other
- [ ] Field values reflect the domain meaning the field name carries (e.g. a `confidence` number is a plausible value rather than `0`, a `mailbox` string is an email address) rather than placeholder values like `"string"`
- [ ] A generated payload that does not conform to the field schema is never written; the command reports any field it could not produce a conforming value for
- [ ] No payload contains a field that is not declared on the command or event it belongs to

**Context:** The typed `fields` are a hard schema and the contract every payload must satisfy. The field *name* carries domain meaning the model can read; the field *type* and `required`/`optional` modifier are the conformance contract. Conformance is what makes the difference between a fixture a developer can run and one they must rewrite. Express the type-correctness guarantee as criteria on the produced payloads — do not prescribe how emod checks them.

**Depends on:** US-001

---

### US-003: Emit Gherkin feature files and JSON fixtures
**Description:** As a model author, I want the generated scenarios written as Gherkin `.feature` files and/or JSON fixtures, so that I can drop them straight into a BDD runner or a table-driven test harness.

**Acceptance Criteria:**
- [ ] `--format gherkin` writes `.feature` files containing `Feature`, `Scenario`, `Given`/`When`/`Then` steps, with each command and event payload rendered as a data table of its fields
- [ ] `--format json` writes JSON fixtures where each scenario carries its `slice`, `title`, a `given` list of event payloads, a `when` command payload, and a `then` event payload
- [ ] `--format both` writes both the Gherkin and JSON artifacts for the same scenarios
- [ ] `--format` defaults to `gherkin` when not specified
- [ ] Gherkin scenarios are grouped into features (one feature per slice or per aggregate) so the output reads as a navigable spec suite
- [ ] Re-running with the same model, format, and `--seed` produces identical output, so generated files are stable across runs

**Context:** Gherkin and JSON cover the common BDD and table-driven harness consumers. The rendered artifacts are the user-facing product of this feature, so the structure of the `.feature` files and JSON fixtures is in scope to specify; how emod renders them internally is not. Stable output under a fixed seed matters because these files live in source control and authors hand-edit them.

**Depends on:** US-002

---

### US-004: Scope generation to a single slice
**Description:** As a model author iterating on one slice, I want `--slice <name>` to generate scenarios for just that slice, so that I get fast, focused output instead of regenerating the whole model's suite each time.

**Acceptance Criteria:**
- [ ] `--slice "<name>"` generates scenarios only for the named slice and for no other slice in the model
- [ ] `--slice` is repeatable, so passing it more than once generates for exactly the named set of slices
- [ ] A `--slice` value that matches no slice in the model produces a clear error naming the unknown slice and exits non-zero
- [ ] The scenarios produced for a scoped run are identical to those the same slice receives in a whole-model run (same skeleton, same conformance, same format options)
- [ ] The run summary and reported cost reflect only the scoped slice(s), not the whole model

**Context:** Whole-model runs can produce a large suite across many slices; during authoring an author usually changes one slice at a time. This is a thin scoping surface on top of the core generator and must not change the per-slice generation behaviour. The example proposal shows `--slice "Classify message"` producing that slice's two flow branches plus its negatives.

**Depends on:** US-001

---

### US-005: Generate negative scenarios for missing required fields
**Description:** As a model author, I want a scenario for each required field showing the command rejected when that field is absent, so that my suite covers the schema's mandatory-field contract and not only the happy path.

**Acceptance Criteria:**
- [ ] For each `required` field on a command, a negative scenario is produced whose WHEN omits that field and whose THEN states the command is rejected because that required field is absent
- [ ] The negative scenario names the specific absent field (e.g. rejected because the required field `confidence` is absent)
- [ ] The other required fields in the negative scenario's WHEN payload are present and conform to their field types
- [ ] Negative scenarios are included by default and `--negatives=false` omits them
- [ ] Negative scenarios render in the same selected format(s) and grouping as the happy-path and branch scenarios
- [ ] The run summary distinguishes happy/branch scenario counts from negative scenario counts (e.g. "2 flow branches + 1 negative")

**Context:** Missing-required-field rejections follow purely from the schema — they need no model to invent them, only to phrase the scenario title. This is the deterministic core of negative coverage; richer implied negatives (low-confidence-should-escalate, sender-mismatch) come later under enrichment. Keep these criteria about the produced negative scenarios, not about how rejection is detected.

**Depends on:** US-002

---

### US-006: Keep generated scenarios faithful to the model's flows
**Description:** As a model author, I want any scenario whose events fall outside the model's flows dropped before it is written, so that the generated suite encodes only behaviour my model actually declares and never a fabricated outcome.

**Acceptance Criteria:**
- [ ] A scenario whose THEN event is not one the command points to via a `flow` edge is dropped and never written
- [ ] A scenario whose GIVEN references an event outside the slice's `decides_on` clause or upstream automation chain is dropped and never written
- [ ] For a `mode dcb` slice, the GIVEN events are taken from the command's `decides_on { events [...] }` clause, and a `decides_on` with a `where tag(...)` predicate produces GIVEN event payloads whose tag-referenced fields carry matching values so the predicate is satisfied
- [ ] For an aggregate-mode slice, the GIVEN is the precondition events reached by walking the upstream automation chain (e.g. `Classify message`'s GIVEN includes `EmailConversationCustomerIdentified`, the THEN of the upstream `Identify customer` slice)
- [ ] The command reports how many scenarios were dropped by the faithfulness check rather than silently discarding them
- [ ] No generated artifact contains a scenario referencing a command or event name absent from the model

**Context:** Faithfulness is the trust property of the whole feature — the value is living tests that track the model, so a scenario that strays from the model is worse than no scenario. THEN is whitelisted to the command's `flow` edges; GIVEN to the `decides_on`/automation chain. This story carries the faithfulness guarantee as observable acceptance criteria across both aggregate and DCB models; do not specify how the chain is walked, only that the produced scenarios stay within it. The example uses `examples/dcb_model.emod`'s `AuthorizePayment` (`decides_on { events [OrderPlaced] where tag(entity = customerId) and tag(category = orderType) }`) and `examples/inbound_customer_comms_agentic_reply.emod`'s `Classify message`.

**Depends on:** US-002

---

### US-007: Regenerate fixtures that fail the conformance check
**Description:** As a model author, I want a payload that comes back non-conforming to be regenerated within a bounded number of attempts, so that an occasional bad value (a string in a `number` field, a missing required field) is repaired automatically instead of landing in my fixtures.

**Acceptance Criteria:**
- [ ] When a returned payload fails field-type conformance or the flow-membership check, the command regenerates that scenario's fixtures rather than writing the non-conforming result
- [ ] Regeneration is bounded by `--attempts <n>` with a documented default of 3
- [ ] When a scenario's fixtures still do not conform after the attempt budget is exhausted, that scenario is dropped and reported rather than written non-conforming
- [ ] The conforming scenarios from the same run are still emitted even when one scenario fails to converge
- [ ] The run reports the number of repair attempts and the field-conformance result (e.g. "field conformance: ok (12/12 fields, 0 repairs)")
- [ ] Regeneration repairs only the offending values; the scenario's structure (GIVEN/WHEN/THEN events) is unchanged across attempts

**Context:** This is a mini repair loop scoped to fixture values only — materially smaller than the foundation's generate→validate→lint loop, because the structure is already fixed by the skeleton; only values are repaired against a known schema. Most slices converge in one call; the loop exists for the occasional type slip. On non-convergence the design favours emitting the conforming scenarios and reporting the dropped ones over writing a bad fixture. Describe only the observable repair/drop/report behaviour.

**Depends on:** US-002, US-006

---

### US-008: Surface implied edge and branch cases the model gestures at
**Description:** As a model author, I want testgen to propose the implied edge cases my model points at — a low-confidence classification that should escalate, a deferred-identification branch, an optional-field-present variant — so that my suite covers the interesting behaviour, not only the obvious path.

**Acceptance Criteria:**
- [ ] Generated scenarios include implied edge cases beyond the structural happy/branch/negative set (e.g. a low-confidence classification escalating, a `senderMismatch == true` identification, a deferred-identification branch with multiple candidates)
- [ ] Every proposed edge case is passed through the faithfulness check, so an edge case whose THEN event is not in the model's flows is dropped before rendering
- [ ] Edge-case scenarios carry payloads that conform to the field schema, identical to the conformance applied to happy-path scenarios
- [ ] `--effort high` produces richer edge-case coverage than the default effort, and the chosen effort is reported in the run output
- [ ] Edge-case scenarios receive business-language titles describing the case (e.g. "A hardship complaint is detected and flagged for escalation") rather than mechanical names
- [ ] Edge-case scenarios render in the selected format(s) and grouping alongside the structural scenarios

**Context:** This is where the model earns its place beyond filling values: surfacing the negative and branch cases the model implies but does not spell out. The faithfulness filter is what keeps proposed cases honest — the model may propose, but only cases whose events exist in the model survive. Default effort is a low-stakes mechanical pass; `--effort high` is for when richer edge-case reasoning is wanted. Sequence this after the core skeleton, conformance, faithfulness, and repair behaviours are solid.

**Depends on:** US-006, US-007

---

### US-009: Report token cost and machine-readable results per run
**Description:** As a model author, I want each run to report its token usage and cost, and to offer a machine-readable result, so that I understand what generation cost me and can call testgen from scripts or CI.

**Acceptance Criteria:**
- [ ] On completion the command reports total input and output token usage and a derived dollar cost for the run
- [ ] The cost summary is shown by default on an interactive terminal and can be forced or suppressed via `--show-cost`
- [ ] `--json` emits a single machine-readable object containing the generated scenarios, the per-slice attempt counts, the dropped-scenario count, and token usage with derived cost
- [ ] In `--json` mode human-oriented progress text is not interleaved into the machine-readable object on the same stream
- [ ] A scoped run (`--slice`) and a `--negatives=false` run each report cost reflecting only the work actually done
- [ ] The reported usage and cost account for every repair attempt, so a run that repaired fixtures reports visibly higher cost than a zero-repair run

**Context:** Generation across many slices multiplies token spend, so cost must be surfaced every run; the repair loop and edge-case enrichment both add attempts that should show up in the total. The machine-readable result lets testgen run in CI or feed other tooling. Keep cost wording consistent with how the project already reports Bedrock spend, and keep the `--json` field set aligned with what the human-readable run reports.

**Depends on:** US-003, US-007

## Non-Goals (Out of Scope)

- Generating narrative documentation or onboarding prose — this feature emits specs and fixtures, not explanatory text; documentation is a separate proposal.
- Answering free-form questions about a model — that is the separate talk-to-your-model proposal.
- Generating production code (aggregate handlers, projectors). This is test/spec material only.
- Asserting that the generated scenarios are complete or correct against the real implementation. They are faithful to the model; if the model is wrong, the fixtures faithfully encode a wrong model. Catching that is the semantic-reviewer proposal's job.
- Asserting business truth of "realistic" values. A conformant `confidence` of `0.94` is type-correct but cannot know an author's real thresholds; fixtures are human-editable starting points, not assertions of business truth.
- Inventing new DSL surface. testgen reads the grammar exactly as documented.
- Re-specifying the shared LLM port, the Bedrock adapter, or model selection — all defined in the AI foundation.
- Output harnesses beyond Gherkin and JSON (Go table-tests, Jest, RSpec) — deferred until a concrete second harness is requested.

## Open Questions

- **GIVEN depth for aggregate models.** Should the GIVEN be the full transitive upstream chain or only the immediate trigger event? Assumption: immediate trigger by default, with a `--given-depth full` option to expand to the full chain; the full chain is faithful but verbose, the immediate trigger is what most BDD harnesses seed.
- **Feature-file grouping granularity.** One feature file per slice, per aggregate, or per context? Assumption: per-aggregate, with one scenario per branch/negative, since that reads more like a real spec suite while staying diff-friendly.
- **Negative-case taxonomy.** Beyond missing-required and wrong-type, should testgen synthesize predicate-violation negatives for DCB (a `decides_on` where the required prior event is absent)? Assumption: high-value and purely structural, so likely yes, sequenced after the enrichment story rather than in the core negatives story.
- **Round-tripping hand edits.** If an author hand-edits a generated `.feature` and later regenerates, can their edits be preserved (merge on scenario title/seed) rather than overwritten? Assumption: defer — valuable for living tests but non-trivial; `--seed` reproducibility is the interim mitigation.
- **Pluggable harness templates.** A `--template <dir>` over the validated scenario object would generalize the renderer to Go table-tests, Jest, or RSpec without touching generation. Assumption: defer until a concrete second harness is asked for.
- **Assumption — no network in automated verification.** Acceptance criteria describe observable command behaviour and do not require a live model; deterministic verification of skeleton, conformance, faithfulness, and the repair loop is expected to use a canned/recorded model, consistent with the foundation's testing stance.
- **Assumption — opt-in and zero impact.** Acceptance criteria assume testgen is absent without configured credentials and adds no dependency to the existing non-AI command paths, consistent with the AI foundation.

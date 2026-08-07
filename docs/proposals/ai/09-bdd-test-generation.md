# BDD Scenario & Test-Fixture Generation

> Part of the [emod AI proposals](./README.md). Assumes the shared [LLM foundation](./00-llm-foundation.md) (the `llm.Model` port and Bedrock-backed Claude) and does not re-specify it.

## Problem

An `.emod` model already encodes a behavioural specification, but it is trapped in
a form no test harness can run. Each command slice says: GIVEN some prior events
exist, WHEN this command arrives (carrying these typed fields), THEN these
event(s) result. That is a Given/When/Then triple in everything but name. Teams
re-author the same triples by hand in their BDD tooling — `.feature` files,
table-driven Go tests, JSON fixtures — and from then on the spec and the model
drift apart. The model says the `Classify message` slice can emit either
`EmailConversationClassified` or `EmailConversationEscalationDetected`; the test
suite, written six weeks later, only covers the happy path because whoever wrote
it forgot the escalation branch.

The two hard, tedious parts of writing these specs are exactly the two parts the
model and an LLM can split between them:

1. **The scenario skeleton** — which events are the precondition, which command is
   the trigger, which events are the possible outcomes — is *fully determined* by
   the model's `flow`, `decides_on`, and the upstream `automation` chain. No
   guessing required; emod already parses this.
2. **The realistic content** — a UUID that looks like a real `customerId`, a
   `confidence` of `0.94` rather than `0`, a `category` of `"billing_dispute"`
   rather than `"string"`, a business-readable scenario title, and the *implied*
   negative and branch cases the model gestures at but does not spell out — is
   where an LLM earns its place.

A naive "ask an LLM to write tests from this model" approach fails the same way
generation does (proposal [01](./01-nl-to-model-generation.md)): the model invents
a payload with a `confidence` of `"high"` (a string where the field is `number`),
omits a `required` field, references an event that does not appear in any flow, or
writes a THEN that the model never says is possible. None of that is visible by
reading the output. All of it is checkable: the typed `fields` are a hard schema,
and the set of flows is a hard whitelist. So this feature reuses emod's core move —
let determinism do the structural work and *check* the model's output against the
model — but the oracle here is the **field-type schema and the flow graph**, not the
validator/linter.

## Goals

- A CLI command that turns a model's slices into executable-style specs and
  fixtures: `emod ai testgen file.emod -o test/`.
- Given/When/Then scenarios, one happy-path per `flow` edge, plus the negative and
  branch scenarios the model implies (the classification → escalation branch, the
  identification-deferred branch, a missing-`required`-field rejection).
- Schema-conformant sample payloads for every command and event, honouring the
  typed `fields` exactly — `UUID` fields get UUIDs, `timestamp` fields get RFC 3339
  timestamps, `number` fields get numbers, `required` fields are always present,
  `optional` fields are sometimes omitted.
- Output formats for real harnesses: Gherkin `.feature` files and/or JSON fixtures
  (`--format gherkin|json|both`).
- A hard conformance gate: every generated payload is validated against the field
  schema derived from the AST; mismatches trigger a bounded regenerate — a mini
  repair loop scoped to fixtures only.
- Faithfulness by construction: a generated scenario must correspond to a `flow`
  (or `decides_on`/automation chain) that exists in the model; the LLM names and
  fills, it does not invent structure.
- Scoping: `--slice <name>` to generate for one slice during iteration.
- Zero impact on existing commands; opt-in, absent without configured credentials.

## Non-Goals

- Generating narrative documentation or onboarding prose — that is
  [proposal 08](./08-docs-generation.md). This feature emits specs and fixtures,
  not explanatory text.
- Answering questions about a model — that is [proposal 07](./07-talk-to-your-model-qa.md).
- Generating *production* code (aggregate handlers, projectors). This is test/spec
  material only; the `docs/proposal.md` "Code Gen (future)" box is a separate
  concern.
- Asserting that the generated scenarios are *complete* against the real
  implementation. They are complete against the *model*; if the model is wrong, the
  tests faithfully encode a wrong model. Catching that is
  [proposal 03](./03-semantic-model-reviewer.md)'s job.
- Re-specifying the `llm.Model` port, the Bedrock adapter, or model selection — all
  in the foundation.
- A new DSL surface. testgen reads the grammar exactly as it is.

## How It Works

The deterministic skeleton comes first and does most of the work; the LLM is
called only to fill values, name scenarios, and surface implied edge cases; then
every value it produced is checked against the field schema before anything is
written.

### 1. Build the scenario skeleton deterministically

For each command slice, emod already has everything needed to assemble the
Given/When/Then *shape* from the AST (`internal/ast`), no model required:

- **WHEN** — the slice's `command` and its `fields` (`[]*ast.Field`, each with
  `Name`, `Type`, `Modifier`).
- **THEN** — every `event` that the command points to via a `flow`
  (`command -> event: X -> Y`). A command with two flow edges (e.g.
  `EmailConversationRecordClassification -> EmailConversationClassified` and
  `-> EmailConversationEscalationDetected`) yields **two** outcome branches, one
  scenario each. This is why the branch case is not "implied" but *structural*.
- **GIVEN** — the precondition events, found two ways:
  - In a `mode dcb` context, directly from the command's `decides_on { events [...] }`
    clause (`ast.DecidesOnClause.Events`) and its `where tag(...)` predicate.
  - In an aggregate-mode model, by walking the **upstream automation chain**: the
    `Classify message` command is triggered by `automation EmailConversationClassify`
    whose `on` event is `EmailConversationCustomerIdentified`, which is the THEN of
    the `Identify customer` slice, which itself chains back to
    `EmailConversationInboundReceived`. The GIVEN is that transitive set of prior
    events. emod can compute this from the automation/flow graph it already builds.

The skeleton is therefore a list of `(given[], when, then)` triples with **typed
field lists attached but no values**. It is faithful by construction: every triple
is anchored to a real `flow` edge and a real chain of events in the file.

```go
type ScenarioSkeleton struct {
    Slice   string
    Given   []*ast.Event   // precondition events (decides_on or upstream chain)
    When    *ast.Command   // the trigger command
    Then    *ast.Event     // one outcome branch
    Kind    string         // "happy" | "branch" | "negative"
}
```

A command with N flow edges produces N `happy`/`branch` skeletons. testgen also
synthesizes `negative` skeletons deterministically (one per `required` field:
"WHEN the command arrives missing `customerId`, THEN it is rejected"), because
those follow purely from the schema — the LLM does not need to invent them, only
to phrase them.

### 2. Field-type → value schema (the hard check)

From each `ast.Field` testgen derives a JSON Schema fragment keyed by the emod
type. This is the contract the LLM must satisfy and the gate its output is checked
against:

| emod type   | JSON Schema | Example generated value |
|-------------|-------------|-------------------------|
| `string`    | `{"type":"string"}` (enum hint for category-like names) | `"billing_dispute"` |
| `UUID`      | `{"type":"string","format":"uuid"}` | `"3f8c1e0a-5b2d-4c7e-9f10-6a2b3c4d5e6f"` |
| `bool`      | `{"type":"boolean"}` | `false` |
| `number`    | `{"type":"number"}` | `0.94` |
| `timestamp` | `{"type":"string","format":"date-time"}` | `"2026-06-19T14:32:05Z"` |
| `date`      | `{"type":"string","format":"date"}` | `"2026-06-19"` |
| `list`      | `{"type":"array"}` | `["acct_7781"]` |

`required` fields are added to the schema's `required` array; `optional` fields are
not. This produced schema is passed two ways: as the `GenerateRequest.Schema` so
the model's structured output is *already* constrained, and — independently — as a
local validator that re-checks the returned payload (the SDK's structured-output
guarantee is good, but a `UUID` that is a syntactically valid string but not a
real UUID, or a `confidence` outside a sane range, still needs catching). The
local check is deterministic and runs without a network.

### 3. The LLM fills values, names, and edge cases

A single structured call per slice (batched where the prompt budget allows) asks
the model to do only what determinism cannot:

- **Realistic values** for each field, conforming to the per-field schema. The
  field *name* carries domain meaning the LLM can read — `confidence` should be a
  plausible 0–1 number, `category` a plausible enum-like string, `mailbox` an email
  address, `modelVersion` something like `"clf-2026.05"`.
- **Business-language scenario titles** — `Classify message` + the
  `EmailConversationClassified` branch becomes
  `"An identified billing question is classified as a routine inquiry"`, not
  `"RecordClassification produces Classified"`.
- **Implied edge/negative cases beyond the structural ones** — e.g. a low-confidence
  classification that *should* escalate, a `senderMismatch == true` identification,
  an identification-deferred branch whose `candidates` list has two entries. The
  model proposes these; testgen keeps only the ones whose THEN event exists in the
  model (faithfulness filter).

The output schema the model fills is a list of scenario instances:

```json
{
  "type": "object",
  "required": ["scenarios"],
  "properties": {
    "scenarios": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["title", "given", "when", "then"],
        "properties": {
          "title":  { "type": "string" },
          "given":  { "type": "array", "items": { "$ref": "#/$defs/eventPayload" } },
          "when":   { "$ref": "#/$defs/commandPayload" },
          "then":   { "$ref": "#/$defs/eventPayload" }
        }
      }
    }
  }
}
```

where `commandPayload`/`eventPayload` are the per-command/per-event field schemas
from step 2, so the structural contract is enforced at generation time.

### 4. The fixture repair loop (fixtures only)

After the call, every payload is re-validated against the field schema locally. On
any mismatch — wrong type, missing `required`, an outcome event not in the flow set
— testgen feeds the specific failures back and regenerates, bounded by
`--attempts` (default 3). This is a *mini* repair loop, materially smaller than the
foundation's `GenerateAndRepair`: it does not re-parse `.emod` or re-run the
validator/linter, because the structure is already fixed by emod; it only repairs
*values* against a known schema.

```
skeleton + field schema ──▶ llm.Model.Generate (Schema-constrained)
                                      │
                                      ▼
                       local field-conformance check
                       + flow-membership check
                                      │
              ┌───────────────────────┴───────────────────────┐
            conforms                                       mismatches
              │                                                │
        render fixtures                          repairRequest(failures) ─┐
                                                                          │
                                            (loop, bounded by --attempts)
```

Most slices converge in one call; the loop exists for the occasional `"high"`-in-a-
`number`-field slip. Because the check is deterministic, the loop is fully testable
against a mock `llm.Model`.

### 5. Render

Conforming scenarios render to the requested format(s). Gherkin `.feature` files
group scenarios by slice (one feature per aggregate or per slice). JSON fixtures
emit a payload-per-command/event map plus the scenario list, ready for a
table-driven harness. testgen never asks the model to write Gherkin or JSON text
directly — it renders from the validated structured object, so the output is
deterministic given the same scenarios.

### Where it plugs in

A new `internal/ai/testgen.go` depending only on `llm.Model`, the parsed
`internal/ast`, and a small field-schema helper (which may live beside
`internal/export`, since export already walks the same field structures). The CLI
command in `internal/cli` is a thin wrapper matching the existing
`RunExport`/`RunSlicesList` actions in `internal/cli/app.go`. Model selection follows
the foundation: routine payload-filling is a low-stakes mechanical pass, so it
defaults to **`anthropic.claude-haiku-4-5`** at `low` effort; `--effort high`
escalates to `anthropic.claude-opus-4-8` when richer edge-case reasoning is wanted.

## Interface

A `testgen` subcommand under the `ai` command group introduced in proposal 01.

```
emod ai testgen <file.emod> [flags]

Flags:
  -o, --output <dir>      directory to write specs/fixtures into (default: stdout)
      --format <fmt>      gherkin | json | both        (default: gherkin)
      --slice <name>      generate for one slice only (repeatable)
      --negatives         include missing-required-field / invalid-type cases (default on)
      --attempts <n>      max fixture-repair attempts (default 3)
      --effort <level>    low|medium|high|xhigh (default: low → Haiku)
      --seed <n>          deterministic value generation for reproducible fixtures
      --show-cost         print token usage and cost summary (default on for a TTY)
      --json              machine-readable result (scenarios, attempts, usage)
```

Example:

```
$ emod ai testgen examples/inbound_customer_comms_agentic_reply.emod \
    --format both --slice "Classify message" -o test/

generating fixtures with anthropic.claude-haiku-4-5 (effort=low)...
  slice "Classify message": 2 flow branches + 1 negative → 3 scenarios
  field conformance: ok (12/12 fields, 0 repairs)
wrote test/classify_message.feature
wrote test/fixtures/classify_message.json
tokens: 2,140 in / 760 out  (~$0.004)
```

Whole-model run:

```
$ emod ai testgen examples/inbound_customer_comms_agentic_reply.emod --format gherkin -o test/
  18 command slices → 24 happy/branch + 14 negative scenarios across 5 features
```

## Worked Example

Take the `Classify message` slice from
`examples/inbound_customer_comms_agentic_reply.emod` (lines 88–125):

```emod
slice "Classify message" {
  command EmailConversationRecordClassification {
    fields {
      category     string required
      confidence   number required
      modelVersion string required
    }
  }

  event EmailConversationClassified {
    fields { category string required; confidence number required; modelVersion string required }
  }

  event EmailConversationEscalationDetected {
    fields {
      suggestedType  string required
      mappedCaseType string required
      confidence     number required
      reasoning      string required
      modelVersion   string required
    }
  }

  automation EmailConversationClassify {
    on EmailConversationCustomerIdentified
    command EmailConversationRecordClassification
    target context InboundEmail
  }

  flow {
    command -> event: EmailConversationRecordClassification -> EmailConversationClassified
    command -> event: EmailConversationRecordClassification -> EmailConversationEscalationDetected
  }
}
```

**Deterministic skeleton.** Two `flow` edges → two outcome branches. The GIVEN is
computed from the automation chain: `EmailConversationClassify` is triggered by
`EmailConversationCustomerIdentified` (the THEN of the upstream `Identify customer`
slice), which itself chains back to `EmailConversationInboundReceived`. So the
GIVEN for both branches is `[EmailConversationInboundReceived,
EmailConversationCustomerIdentified]`. The WHEN is
`EmailConversationRecordClassification`. THEN is one of the two events. testgen
also derives three `negative` skeletons (one per required field on the command).

**Generated happy-path scenario (Gherkin):**

```gherkin
Feature: Classify message

  Scenario: An identified billing question is classified as a routine inquiry
    Given an EmailConversationInboundReceived event has occurred
      | provider          | mailbox              | providerThreadId | rfc822MessageId        |
      | "gmail"           | "support@indebted.com" | "thr_8841"     | "<a1b2@mail.gmail.com>" |
    And an EmailConversationCustomerIdentified event has occurred
      | customerId                             | strategy        | senderMismatch |
      | "3f8c1e0a-5b2d-4c7e-9f10-6a2b3c4d5e6f" | "email_match"   | false          |
    When the command EmailConversationRecordClassification is received
      | category          | confidence | modelVersion  |
      | "billing_dispute" | 0.94       | "clf-2026.05" |
    Then the event EmailConversationClassified is recorded
      | category          | confidence | modelVersion  |
      | "billing_dispute" | 0.94       | "clf-2026.05" |
```

**Matching JSON fixture (`--format json`):**

```json
{
  "slice": "Classify message",
  "title": "An identified billing question is classified as a routine inquiry",
  "given": [
    { "event": "EmailConversationInboundReceived",
      "payload": { "provider": "gmail", "mailbox": "support@indebted.com",
                   "providerThreadId": "thr_8841", "providerMessageId": "msg_2290",
                   "rfc822MessageId": "<a1b2@mail.gmail.com>", "bodyRef": "s3://inbound/raw/8841" } },
    { "event": "EmailConversationCustomerIdentified",
      "payload": { "customerId": "3f8c1e0a-5b2d-4c7e-9f10-6a2b3c4d5e6f",
                   "strategy": "email_match", "senderMismatch": false } }
  ],
  "when": {
    "command": "EmailConversationRecordClassification",
    "payload": { "category": "billing_dispute", "confidence": 0.94, "modelVersion": "clf-2026.05" }
  },
  "then": {
    "event": "EmailConversationClassified",
    "payload": { "category": "billing_dispute", "confidence": 0.94, "modelVersion": "clf-2026.05" }
  }
}
```

Note the conformance the gate enforces: `customerId` is a real UUID (field type
`UUID`), `confidence` is a `number` not a string, `senderMismatch` is a `bool`, and
every `required` field on each event is present.

**Generated branch scenario** (the second `flow` edge — structural, not invented):

```gherkin
  Scenario: A hardship complaint is detected and flagged for escalation
    Given an EmailConversationInboundReceived event has occurred
    And an EmailConversationCustomerIdentified event has occurred
    When the command EmailConversationRecordClassification is received
      | category    | confidence | modelVersion  |
      | "complaint" | 0.88       | "clf-2026.05" |
    Then the event EmailConversationEscalationDetected is recorded
      | suggestedType | mappedCaseType   | confidence | reasoning                                  | modelVersion  |
      | "hardship"    | "HARDSHIP_REVIEW" | 0.88      | "Customer reports inability to pay due to job loss" | "clf-2026.05" |
```

The LLM filled `suggestedType`/`mappedCaseType`/`reasoning` (it cannot know these
from structure alone), but it could not point THEN at any event other than the two
in the flow — that whitelist is the faithfulness gate.

**Negative scenario** (deterministic skeleton, LLM-phrased title):

```gherkin
  Scenario: Classification is rejected when confidence is missing
    Given an EmailConversationCustomerIdentified event has occurred
    When the command EmailConversationRecordClassification is received
      | category          | modelVersion  |
      | "billing_dispute" | "clf-2026.05" |
    Then the command is rejected because the required field "confidence" is absent
```

For a model with a `decides_on` clause (e.g. `examples/dcb_model.emod`'s
`AuthorizePayment`, whose `decides_on { events [OrderPlaced] where tag(entity =
customerId) and tag(category = orderType) }`), the GIVEN is taken *directly* from
that clause, and the generated `OrderPlaced` fixture in the GIVEN is made to carry
matching `customerId`/`orderType` tag values so the `where` predicate is satisfied —
another place determinism constrains the LLM's freedom.

## Implementation Plan

**Phase 1 — Skeleton + field schema (M).** In `internal/ai/testgen.go`: walk
`internal/ast` to build `ScenarioSkeleton`s per command slice — flow-edge → THEN
branches, automation/flow-graph walk for the GIVEN chain, `decides_on` for DCB
GIVENs, and per-required-field negatives. Build the per-field JSON Schema from
`ast.Field.Type`/`Modifier`. Pure, no LLM; unit-tested against the two `examples/`
models with `testify/require` (assert branch count, GIVEN sets, schema shape).

**Phase 2 — LLM fill + conformance gate (M).** The structured `llm.Model` call
that fills values/titles, the local field-conformance and flow-membership checks,
and the bounded fixture-repair loop. Haiku default per the foundation; tested
against a mock model driving the 0-repair, N-repair-converge, and never-converge
paths.

**Phase 3 — Renderers (S).** Gherkin `.feature` writer (feature-per-slice/aggregate,
data tables for payloads) and JSON fixture writer. Deterministic given a scenario
set. `--format gherkin|json|both`.

**Phase 4 — CLI surface (S).** `emod ai testgen` in `internal/cli`: `-o` dir vs.
stdout, `--slice`, `--negatives`, `--attempts`, `--effort`, `--seed`, `--show-cost`,
`--json`. Match `RunExport`/`RunSlicesList` structure and `LintError` exit handling in
`internal/cli/app.go`.

**Phase 5 — Edge-case enrichment & polish (M/L).** Prompt tuning so the model
proposes valuable implied cases (low-confidence-should-escalate, deferred-
identification, sender-mismatch, optional-field-present-vs-absent variants), each
passed through the faithfulness filter. A regression harness asserting generated
fixtures conform to the field schema for the bundled examples using a recorded/mock
model (no network).

## Risks & Mitigations

- **Generated payload violates the field schema.** The whole design's hard check:
  Schema-constrained generation plus a local re-validation and a bounded repair
  loop. On non-convergence, emit the conforming scenarios and report the dropped
  ones rather than writing a bad fixture.
- **Scenario references structure not in the model (hallucinated THEN/event).**
  THEN is whitelisted to the command's `flow` edges; GIVEN to the `decides_on`/
  automation chain. Any scenario whose events fall outside those sets is dropped by
  the faithfulness filter before rendering.
- **"Realistic" values are still semantically wrong.** A conformant `confidence`
  of `0.94` is type-correct but the model can't know your real thresholds. Fixtures
  are starting points, not assertions of business truth; they are human-editable
  `.feature`/JSON, and `--seed` makes them reproducible so edits aren't clobbered
  on re-run. Semantic correctness of the *model itself* is proposal 03.
- **Tests faithfully encode a wrong model.** testgen is faithful to the `.emod`, not
  to reality — by design (see Non-Goals). Document this clearly; the value is
  *living tests that track the model*, surfacing drift when the model changes and
  the regenerated specs differ.
- **Token cost across many slices.** Default to `anthropic.claude-haiku-4-5` at
  `low` effort (payload-filling is mechanical), batch slices per call where the
  prompt budget allows, surface cost every run, and bound repair attempts.
- **Branch explosion.** A command with many flow edges × many required-field
  negatives can produce a large suite. `--slice` scopes it; `--negatives=false`
  drops the negatives; group by feature keeps output navigable.
- **Non-determinism in tests.** Never call a live model; the mock `llm.Model`
  returns canned scenario payloads, and `--seed` plus deterministic rendering make
  the pipeline reproducible.

## Open Questions

- **GIVEN depth for aggregate models.** Should the GIVEN be the *full* transitive
  upstream chain or only the immediate trigger event? The full chain is faithful
  but verbose; the immediate trigger is what most BDD harnesses actually seed.
  Lean: immediate trigger by default, `--given-depth full` to expand.
- **One feature file per slice, per aggregate, or per context?** Per-slice is
  granular and diff-friendly; per-aggregate reads more like a real spec suite.
  Lean: per-aggregate, with one `Scenario` per branch/negative.
- **Negative-case taxonomy.** Beyond missing-required and wrong-type, should
  testgen synthesize predicate-violation negatives for DCB (`decides_on` where the
  required prior event is *absent*)? That is high-value and purely structural —
  probably yes, sequenced after Phase 5.
- **Pluggable harness templates.** Gherkin and JSON cover most consumers, but Go
  table-tests, Jest, and RSpec are tempting targets. A template directory
  (`--template <dir>`) over the validated scenario object would generalize the
  renderer without touching generation. Defer until a concrete second harness is
  asked for.
- **Round-tripping edits.** If a user hand-edits a generated `.feature` and later
  regenerates, can testgen preserve their edits (merge on scenario title/seed)
  rather than overwrite? Valuable for "living tests" but non-trivial; defer.
- **Coupling to proposal 05.** DCB-aware GIVEN generation (tag-matching fixtures,
  absent-prior-event negatives) overlaps the DCB assistant. Share the
  `decides_on`-walking helper rather than duplicate it.

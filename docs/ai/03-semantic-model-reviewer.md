# Semantic Model Reviewer

> Part of the [emod AI proposals](./README.md). Assumes the shared [LLM foundation](./00-llm-foundation.md) (the `llm.Model` port and Bedrock-backed Claude) and does not re-specify it.

## Problem

`emod lint` is fast, deterministic, and shallow. Every rule in
`internal/linter/descriptions.go` is some combination of a suffix/prefix string
match (`state-obsession`, `property-sourcing`, `command-in-disguise`,
`command-past-tense`, `view-naming`), a count threshold (`left-chair` ≥3 flows,
`god-view` ≥5 events, `clickbait-event` 1 ID field, `dcb/query-too-broad` >5
events), or a structural presence check (`dcb-in-aggregate-mode`,
`aggregate-in-dcb-mode`, `dcb/untagged-event`, `dcb/single-tag-everywhere`,
`dcb/orphan-tag-key`). That is exactly the right design for those rules: they are
cheap, repeatable, and never wrong about what they measure.

But the linter measures *form*, not *meaning*. It cannot tell whether an event
that passes every regex actually captures a business fact. Consider the real
example in `examples/inbound_customer_comms_agentic_reply.emod`:

- `EmailConversationReplyInitiated` ends in `...Initiated`, so
  `command-in-disguise` *does* fire — good.
- But `EmailConversationInboundReceived`, `EmailConversationClassified`,
  `EmailConversationCustomerIdentified` are all well-formed past-tense events with
  rich payloads. They pass every rule. Whether `Classified` and
  `EscalationDetected` should be *one* event or *two*, whether
  `RecordClassification` is really one command or a classify-then-route pair, and
  whether `EmailConversationInitiateReply` / `EmailConversationReplyInitiated` is
  a command masquerading as a decision — none of that is reachable by a suffix
  check. It needs a reader who understands the domain.

The deterministic linter cannot detect:

- **Hollow events that pass the regex.** An event named `OrderProcessed` or
  `RequestHandled` is past-tense and has fields, so `state-obsession` and
  `command-in-disguise` stay silent — but it still names a *step in code* rather
  than a *fact in the business*. The name is fine; the meaning is empty.
- **Wrong aggregate / context boundaries.** Whether `CustomerCase` belongs in its
  own context, or whether the four hold/scheduling contexts in the inbound-email
  model are really one consistency concern, is a judgement about cohesion the
  linter has no notion of.
- **Missing slices or events in a process.** A command that emits a success event
  but no failure/rejection event, or a flow that jumps from "classified" to
  "reply sent" with no "reply composed/approved" step, is a *gap*. Nothing counts
  what *should* be there.
- **A command doing two jobs.** `RecordClassification` that both classifies and
  decides escalation, or a `Submit` that validates *and* schedules, should be
  split — but it has the right name and a normal field list.
- **Weak or inconsistent ubiquitous language.** `Submit` vs `Initiate` vs
  `Record` vs `Receive` as command verbs, `customerId` vs `accountIds` vs
  `referenceId` for the same notion across contexts — drift the regex can't see.

These are the modeling smells that make a model technically valid but a poor map
of the domain. They are exactly what a competent reviewer catches in a design
review, and they are out of reach for any name-based rule.

## Goals

- An `emod ai review` command that reads an existing `.emod` file and emits
  **semantic** modeling findings the regex linter cannot produce.
- Findings flow through the **same** `internal/diagnostic` pipeline as `emod lint`
  — same `Entry` shape, same severities, same text/JSON formatting, same LSP
  surface — so consumers don't learn a second format.
- High precision over high recall: a noisy reviewer is worse than none. Findings
  carry a **confidence** and are filtered before they reach the user.
- Findings are **advisory and located**: each names a likely problem, points at a
  span, and suggests a *direction*, but never rewrites the file.
- Usable both interactively (rich, exploratory) and in CI (stable, gated,
  cacheable).
- Fully testable against a mock `llm.Model`; no network in unit tests.

## Non-Goals

- **Applying fixes.** This proposal *reviews*. Turning a finding into an edit is
  proposal [04 (lint quick-fixes)](./04-lint-quickfixes-lsp.md), which anchors AI
  code actions to findings — including these.
- **DCB-specific tag/`decides_on` suggestions.** Recommending tags, narrowing a
  `decides_on`, or fixing `dcb/*` smells is proposal
  [05 (DCB modeling assistant)](./05-dcb-modeling-assistant.md). The reviewer may
  *observe* a DCB smell semantically, but it does not author tag schemes.
- **Replacing the regex linter.** The deterministic rules stay. AI findings
  *complement* them (see below). `emod lint` must keep working with no
  credentials configured.
- **The generate → validate → lint repair loop.** This feature reads a model and
  reports; it does not produce a model, so it does not use the loop (consistent
  with the foundation doc's dependency table, which marks proposal 03 as “Emits
  diagnostics”, no repair loop).

## How It Works

### Deterministic linter vs AI reviewer — a division of labour

The two are complementary by construction. The regex linter owns everything
*decidable from the AST shape*; the reviewer owns everything *requiring domain
judgement*. The reviewer is told what the linter already covers so it doesn't
re-report it.

| Concern | Owner | Why |
|---|---|---|
| Name ends in `Updated/Changed/Modified` | `state-obsession` (regex) | Exact, free, never wrong |
| `<Aggregate><Field>Changed` | `property-sourcing` (regex) | Structural |
| Name ends in `Initiated` | `command-in-disguise` (regex) | Structural |
| Command ends in `ed` | `command-past-tense` (regex) | Structural |
| View not ending in `View` | `view-naming` (regex) | Structural |
| Command in ≥3 flows / view ≥5 events | `left-chair` / `god-view` (regex) | Counting |
| Single-ID event | `clickbait-event` (regex) | Counting |
| DCB construct/mode mismatches, broad query, orphan/single tag | `dcb/*` (regex) | Structural / counting |
| Event name is past-tense and has fields but **names a code step, not a business fact** | **AI reviewer** | Needs meaning |
| Aggregate / context boundary is incohesive | **AI reviewer** | Needs cohesion judgement |
| A process is **missing** a failure event or an intermediate slice | **AI reviewer** | Needs a model of what *should* exist |
| One command is **doing two jobs** | **AI reviewer** | Needs intent inference |
| Ubiquitous language **drifts** across contexts | **AI reviewer** | Needs cross-model reading |

The rule of thumb: **if a regex or a count can decide it, the linter must own it.**
The reviewer is for the residue that only a reader of the domain can judge. The
reviewer's prompt is seeded with the list of regex rule names so it is explicitly
instructed *not* to emit findings that duplicate `state-obsession`,
`command-in-disguise`, etc. — those are already covered upstream and noisy
duplication erodes trust.

### Feeding the model the whole model

emod models are small — the largest example here is under 500 lines — so the
reviewer sends the **entire model**, not a retrieval slice. Two representations
go into the prompt, each playing a role:

1. **The raw `.emod` text**, verbatim, with line numbers. This is what the model
   reasons over for *naming, intent, and readability*, and it is how the model
   refers back to locations (“line 89, `EmailConversationRecordClassification`”).
2. **The JSON export** (`emod export --format json`, from
   `internal/export/export.go`). This gives the model a structurally unambiguous
   view — every element already carries `{filename, line, column}` positions (see
   `jsonPosition` in `export.go`) plus open/close spans for contexts and
   aggregates. The reviewer uses these to *ground every finding to a real
   position* rather than trusting the model to count lines.

Sending both is cheap (a few thousand tokens) and removes two failure modes at
once: the text keeps naming nuance the JSON flattens, and the JSON keeps the
positions the text makes the model hallucinate.

System prompt content (assembled in Go, not a DSL):

- The role: a senior event-modeling reviewer.
- The emod concept glossary (`model`, `actor`, `context`/`mode`, `aggregate`,
  `slice`, `command`, `event`, `view`, `automation`, `translation`, `flow`,
  `trigger`, `decides_on`) so the model reasons in emod's vocabulary.
- The **exclusion list**: the regex rule names from
  `internal/linter/descriptions.go`, with “these are already checked; do not
  duplicate them”.
- The finding schema (below), passed as JSON Schema via `GenerateRequest.Schema`
  so the response is a validated array, not parsed prose.
- Effort `high` on `anthropic.claude-opus-4-8` (the foundation doc's hard model —
  “Generation, reverse-engineering, semantic review”), with adaptive thinking.

### The finding schema

A finding is deliberately shaped to drop into `diagnostic.Entry` with one mapping
step. The model returns objects conforming to:

```go
package review

// Finding is the model's structured output. It maps onto diagnostic.Entry
// after confidence filtering; see ToEntry.
type Finding struct {
    RuleID     string  `json:"rule_id"`     // e.g. "ai/hollow-event"
    Message    string  `json:"message"`     // one sentence, what's wrong and why
    Filename   string  `json:"filename"`
    Line       int     `json:"line"`        // 1-based, matched to a JSON-export position
    Column     int     `json:"column"`      // 1-based
    EndLine    int     `json:"end_line"`    // optional span end
    EndColumn  int     `json:"end_column"`
    Severity   string  `json:"severity"`    // "info" | "warning" — never "error"
    Confidence float64 `json:"confidence"`  // 0.0–1.0, the model's self-rated certainty
    Direction  string  `json:"direction"`   // suggested direction, NOT a rewrite
    Evidence   string  `json:"evidence"`    // the model names what in the model led here
}
```

The `rule_id` is *rule-ish*, not from a closed set: the reviewer works from a
small **taxonomy** of semantic smells it is told to classify into, so output
clusters into stable, greppable ids rather than free-text categories:

| `rule_id` | Smell |
|---|---|
| `ai/hollow-event` | Past-tense, well-formed name that names a code step, not a business fact |
| `ai/boundary-smell` | Aggregate or context grouping lacks cohesion / splits one concern |
| `ai/missing-event` | A process is missing a failure/rejection or intermediate event |
| `ai/missing-slice` | A gap between flows where an intermediate slice should exist |
| `ai/command-overloaded` | One command is doing two distinct jobs; should be split |
| `ai/weak-naming` | Technically valid but domain-weak naming (beyond the regex rules) |
| `ai/language-drift` | Inconsistent ubiquitous language across the model |

All `ai/*` ids are namespaced so they never collide with the linter's rule names
and are trivially filterable (`emod lint` rules have no `ai/` prefix).

`Severity` is capped at `warning`: a *probabilistic* finding must never block like
a deterministic `error` (`left-chair`, `god-view`, `dcb/untagged-event` are the
only things allowed to be `error`). This keeps the trust boundary clear — exit
codes are driven by deterministic findings, AI findings advise.

### Emitting diagnostics — same pipeline

`ToEntry` collapses a `Finding` into the existing `diagnostic.Entry` so everything
downstream (`Entry.String()`, the CLI `jsonEntry`, `lsp.ConvertDiagnostics`) works
unchanged:

```go
func (f Finding) ToEntry() *diagnostic.Entry {
    return &diagnostic.Entry{
        Filename: f.Filename,
        Line:     f.Line,
        Column:   f.Column,
        Message:  f.Message, // Direction appended in text/LSP rendering
        Severity: parseSeverity(f.Severity), // capped at Warning
        RuleName: f.RuleID,
    }
}
```

Because the reviewer reuses `diagnostic.Entry`, `emod ai review --format json`
produces the *same* `jsonEntry` shape as `emod lint --format json` (the
`{file, line, rule, severity, message}` struct in `internal/cli/lint.go`), and an
LSP client sees AI findings as `emod`-sourced diagnostics alongside lint ones.
The only addition the AI path needs is surfacing `confidence` and `direction`,
which ride along in the JSON output and in hover text but don't change the core
shape.

### Precision controls

Probabilistic findings are only useful if they're rarely wrong. Four controls:

1. **Confidence threshold.** Every finding is filtered against a threshold
   (default `0.7`, `--min-confidence` to override). Below it, dropped. This is the
   primary recall/precision dial.
2. **Capped severity.** As above — AI findings cannot reach `error`, so a false
   positive never breaks a build by itself.
3. **Position grounding.** A finding whose `line` does not correspond to a real
   element position in the JSON export is discarded as a hallucination — the model
   was told to cite positions from the export, and a citation that doesn't
   resolve is a tell.
4. **Optional adversarial self-check (`--verify`).** A cheap second pass: the
   surviving findings are sent back (to `anthropic.claude-haiku-4-5`, the cheap
   model) with the model and a single instruction — *“for each finding, is this a
   real domain problem or a false positive? return keep/drop with a reason.”*
   Findings the critic drops are removed. This trades latency and a little cost
   for materially fewer false positives, so it's opt-in interactively and the
   default in CI.

## Interface

### CLI

A new `ai` command group (so future AI features — 01, 05, 07 — share a namespace
distinct from the always-available `validate`/`lint`/`diagram`/`export`):

```
emod ai review <file>                      # semantic review, text output
emod ai review <file> --format json        # same jsonEntry shape as `emod lint --format json`
emod ai review <file> --min-confidence 0.8 # raise the precision bar
emod ai review <file> --severity warning   # only show findings at/above a severity
emod ai review <file> --verify             # run the adversarial self-check pass
emod ai review <file> --cache              # write/read a content-addressed cache (see below)
```

Flags mirror `emod lint` where they overlap (`--format text|json`) so the surface
is familiar. Like all AI features (foundation doc, “Configuration”), `emod ai
review` is opt-in: with no Bedrock credentials configured the command exits with a
clear “AI features require EMOD_AI_* configuration” message and a non-zero code,
and nothing about the existing `lint` path changes.

Exit codes follow `lint`'s convention but, because AI findings cap at `warning`,
`review` returns `1` when any finding survives the threshold and `0` when clean —
it never returns `2` (reserved for deterministic errors).

Example text output:

```
$ emod ai review examples/inbound_customer_comms_agentic_reply.emod
inbound_customer_comms_agentic_reply.emod:89: [ai/command-overloaded] command "EmailConversationRecordClassification" appears to do two jobs: classify content AND detect escalation (confidence 0.82)
    direction: split into RecordClassification (-> Classified) and a separate decision that emits EscalationDetected, so escalation is its own slice
inbound_customer_comms_agentic_reply.emod:97: [ai/missing-event] slice "Classify message" has no event for the "classification failed / low-confidence" outcome that the confidence field implies (confidence 0.74)
    direction: add an event such as ClassificationInconclusive so the downstream auto-reply vs human-handoff branch is explicit in the model
2 findings (0 below confidence threshold 0.70 hidden)
```

### LSP

The LSP server (`internal/lsp`) already renders `diagnostic.Entry` via
`ConvertDiagnostics`. The reviewer plugs in two ways:

- **On demand**, as an LSP command / code-lens (“Run AI review”) rather than on
  every keystroke — review is slow and costs tokens, so it must not run on the
  document-change debounce that drives the regex linter. Results merge into the
  same `textDocument/publishDiagnostics` stream, tagged `source: "emod"`.
- **Hover** shows the finding's `direction`, `evidence`, and `confidence` (the
  fields `diagnostic.Entry` doesn't carry are kept in a side table keyed by
  position so hover can enrich them).

A `confidence` slider / setting maps to `--min-confidence` so an editor user can
dial noise without re-running.

## Worked Example

A model that is *clean under every regex rule* but has a real semantic smell:

```emod
model "Subscription Billing"

actor "Customer"

context "Billing" {
  aggregate "Subscription" {
    slice "Start subscription" {
      command StartSubscription {
        fields {
          customerId string required
          planId     string required
        }
      }

      event SubscriptionStarted {
        fields {
          subscriptionId string    required
          customerId     string    required
          planId         string    required
          startedAt      timestamp required
        }
      }

      flow {
        command -> event: StartSubscription -> SubscriptionStarted
      }
    }

    slice "Charge for period" {
      command ChargeSubscription {
        fields {
          subscriptionId string required
          amount         int    required
        }
      }

      event SubscriptionCharged {
        fields {
          subscriptionId string    required
          amount         int       required
          chargedAt      timestamp required
        }
      }

      flow {
        command -> event: ChargeSubscription -> SubscriptionCharged
      }
    }
  }
}
```

`emod lint` is silent: every event is past-tense with a rich payload (no
`state-obsession`, no `clickbait-event`), commands are imperative (no
`command-past-tense`), nothing ends in `Initiated`, no view, no flow fan-out, no
DCB constructs. The regex linter has nothing to say.

The reviewer, reading the *meaning*, would emit:

```json
[
  {
    "rule_id": "ai/missing-event",
    "message": "command \"ChargeSubscription\" emits only the success event \"SubscriptionCharged\"; a charge can be declined, but the model has no event for a failed charge",
    "filename": "subscription_billing.emod",
    "line": 30,
    "column": 7,
    "severity": "warning",
    "confidence": 0.86,
    "direction": "add an event such as SubscriptionChargeDeclined and a slice that reacts to it (dunning / retry), so the unhappy path is part of the model rather than implied",
    "evidence": "billing charges routinely fail (insufficient funds, expired card) yet only SubscriptionCharged exists; no dunning or retry slice references this aggregate"
  }
]
```

This is the class of finding that justifies the feature: structurally the model is
perfect, but a domain reviewer immediately sees the missing failure path. No suffix
rule or count threshold in `descriptions.go` can reach it.

## Implementation Plan

### Phase 1 — Core reviewer (M)

- `internal/review` package: `Finding`, the JSON Schema, `ToEntry`, the prompt
  assembler (system prompt + glossary + regex-rule exclusion list + `.emod` text +
  JSON export), and `Review(ctx, m llm.Model, src, jsonExport) ([]Finding, error)`.
- Confidence filtering, severity capping, and position grounding against the JSON
  export.
- Unit tests against a mock `llm.Model` returning canned findings: assert filtering
  (below-threshold dropped), severity capping, position grounding (unresolvable
  position dropped), and `ToEntry` mapping. Follow the repo Go test conventions
  (one umbrella `TestReview`, `t.Run` groups, behavior-named scenarios,
  `testify/require`, fresh fixtures per leaf).

### Phase 2 — CLI surface (S)

- `emod ai` command group + `review` subcommand in `internal/cli`, reusing
  `formatText`/`formatJSON` from `lint.go` (extended to carry `confidence` /
  `direction`).
- Flags: `--format`, `--min-confidence`, `--severity`, `--verify`, `--cache`.
- Graceful “no credentials” degradation.

### Phase 3 — Precision hardening (M)

- `--verify` adversarial self-check pass on `anthropic.claude-haiku-4-5`.
- Determinism/caching: content-addressed cache keyed on
  `sha256(model text + reviewer version + model id + effort + min-confidence)`,
  stored under `~/.config/emod`. CI runs with `--cache` so a re-run on an
  unchanged model returns the stored findings (zero tokens, stable output);
  interactive runs may skip the cache for freshness.

### Phase 4 — LSP integration (L)

- On-demand “Run AI review” command / code-lens (never on the change debounce).
- Side table for `confidence`/`direction`/`evidence`; hover enrichment.
- Merge into the existing `publishDiagnostics` stream via `ConvertDiagnostics`.

## Risks & Mitigations

- **False positives erode trust.** A reviewer that cries wolf gets ignored.
  Mitigations are layered: confidence threshold (default `0.7`), capped severity
  (never `error`, never breaks a build alone), position grounding (drop
  hallucinated locations), the regex-rule exclusion list (no duplicate noise), and
  the optional `--verify` critic pass. Default to *fewer, surer* findings.
- **Nondeterminism breaks CI.** LLM output varies run to run, which is poison for
  a CI gate. The content-addressed cache makes a review reproducible for an
  unchanged model: same input → same stored findings, zero tokens. For *new*
  input, CI treats AI findings as advisory (warning-only, never the exit-2 path),
  so a flaky finding can't fail a build; only deterministic `lint` errors do.
- **Cost and latency.** A high-effort Opus pass over a whole model plus an
  optional Haiku critic costs real tokens and seconds. Mitigations: send the model
  once (it's small), gate the LSP path behind an explicit action (not the
  keystroke debounce), use the cheap model for `--verify`, report
  `Response.Usage` cost back to the user (foundation doc, “Cost and latency”), and
  cache aggressively.
- **Overlap with the regex linter.** Without care the model re-reports
  `command-in-disguise` in prose. The exclusion list in the prompt plus the `ai/`
  id namespace and a post-filter that drops any finding whose span coincides with a
  freshly-computed deterministic finding keep the two from stepping on each other.
- **Scope creep into fixes / DCB.** The schema forbids rewrites (`direction`, not
  a patch) and the prompt forbids authoring tags/`decides_on`. Those live in
  proposals 04 and 05; this one only reviews.

## Open Questions

- **Fixed taxonomy vs open `rule_id`?** A closed `ai/*` set is greppable and
  filterable but may miss a smell that doesn't fit. Start with the seven-id
  taxonomy above and allow an `ai/other` escape hatch; promote frequent
  `ai/other` clusters to named ids over time.
- **Default `--verify` on or off?** It clearly improves precision but doubles
  latency and adds cost. Proposal: off interactively, on in CI (where stability
  matters more than speed). Revisit with real false-positive rates.
- **Should `--min-confidence` be per-rule?** `ai/language-drift` may warrant a
  higher bar than `ai/missing-event`. A flat threshold ships first; per-id
  thresholds are a config-file follow-on once we have calibration data.
- **Whole-model context limit.** Sending the entire model is fine at current
  example sizes. At what model size (token budget) does this break, and what's the
  fallback — context-by-context review with a final cross-context language-drift
  pass?
- **Calibrating confidence.** The model's self-rated `confidence` is not
  guaranteed to be well-calibrated. Do we trust it, or derive confidence
  externally (e.g. agreement between the main pass and the `--verify` critic, or
  repeated sampling)? Needs measurement before the default threshold is locked.

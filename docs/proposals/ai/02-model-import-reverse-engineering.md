# Model Import / Reverse-Engineering

> Part of the [emod AI proposals](./README.md). Assumes the shared [LLM foundation](./00-llm-foundation.md) (the `llm.Model` port, Bedrock-backed Claude, and the generate → validate → lint → repair loop) and does not re-specify it. Reuses the repair loop from [01 — NL → model generation](./01-nl-to-model-generation.md).

## Problem

Most teams that would benefit from event modeling already have the system. The
events, commands, projections, and external integrations exist — scattered across
event classes, command handlers, Kafka topic definitions, an eventcatalog site, an
AsyncAPI spec, or the platform's `*.emlang.yaml` files. Writing a `.emod` for that
system by hand means re-deriving, by reading code, what the code already states.

The bundled example proves the value of the destination, not the journey:
`examples/inbound_customer_comms_agentic_reply.emod` carries the header
*"Translated from docs/proposals/...event-model.emlang.yaml in the platform repo"*.
A human did that translation. It is mechanical, tedious, and exactly the kind of
"read a large pile of artifacts, map it onto a fixed vocabulary" work an LLM is
good at — provided the output is checked by something deterministic.

emod has that deterministic check. So the goal here is narrow and well-shaped: point
emod at an existing system and get back a *draft* `.emod` that already parses,
validates, and has been run past the linter — a starting point a human refines, not
a finished model.

## Goals

- A CLI command that turns an existing artifact (or a directory/glob of source) into
  a draft `.emod`: `emod ai import <path-or-glob> -o model.emod`.
- Support the two input families teams actually have:
  - **Source code** — event/command classes, command handlers, message/Kafka topic
    definitions, projections/read models.
  - **Structured artifacts** — eventcatalog YAML, AsyncAPI, the platform's
    `*.emlang.yaml`, JSON event catalogs.
- Map detected concepts onto emod constructs: `context`, `aggregate` vs DCB
  (`mode dcb`), `command`/`event`/`flow`, `view` for projections, `automation` for
  event-triggered handlers, `translation` for external systems.
- Condense inputs that exceed the context window without losing the event/command
  inventory (chunk → summarize per bounded context → assemble).
- Reuse 01's `GenerateAndRepair` loop so the emitted file is parse/validate/lint
  clean before it ever reaches the user.
- Let the linter do what it already does: surface smells the *source* naming carries
  into the model (e.g. an `OrderUpdatedEvent` class becomes `state-obsession`).

## Non-Goals

- Greenfield generation from prose — that is [01](./01-nl-to-model-generation.md).
  This proposal starts from artifacts that describe an existing system.
- Re-specifying the LLM port, adapter, or repair loop — see the
  [foundation](./00-llm-foundation.md) and [01](./01-nl-to-model-generation.md).
- A guaranteed-correct extraction. This is **draft-assist**: the output is a starting
  point for human review, not an authoritative model. Reverse-engineering infers
  intent that the source code never states (which events truly belong together, where
  a context boundary sits), and that inference can be wrong.
- Round-tripping `.emod` → code, or keeping a model continuously in sync with a
  codebase. One-shot import only; re-running overwrites the draft.
- A language-agnostic semantic parser. We extract *signals* with cheap heuristics and
  let the model do the mapping; we do not build a Java/Go/Kotlin AST analyzer.

## How It Works

Three stages: **gather** (collect and condense raw signal), **map** (LLM turns
signal into an AST-shaped draft via structured output), **repair** (01's loop makes
it clean). Only the map and repair stages call the model.

```
inputs ──▶ detect family ──▶ extract signals ──▶ condense ──▶ map (LLM) ──▶ repair loop ──▶ model.emod
          (--from auto)      (per family)        (chunk +     (structured    (01's
                                                  summarize)   output)        GenerateAndRepair)
```

### Stage 1 — Gather and condense

A source codebase is far larger than any context window and mostly irrelevant
(framework code, tests, config). We do **not** feed raw files to the model. Instead a
per-family extractor pulls a compact, structured *signal set*, and a condense step
keeps it within budget.

**Family detection (`--from auto`).** Sniff the inputs before doing any work:

| Signal | Inferred family |
|--------|-----------------|
| `*.emlang.yaml` | `emlang` (platform's own event-model YAML) |
| `asyncapi.yaml` / `asyncapi: 2.x\|3.x` key | `asyncapi` |
| eventcatalog `events/`, `services/`, `*.mdx` front-matter | `eventcatalog` |
| `*.json` catalog with an event/command array | `catalog` |
| source files (`.go`, `.kt`, `.java`, `.ts`, …) | `code` |

`--from code|eventcatalog|asyncapi|emlang|catalog|auto` overrides the sniff.
Structured families (`emlang`, `asyncapi`, `eventcatalog`, `catalog`) already carry
explicit event/command/channel definitions, so their extractor is a deterministic
parser, not an LLM call — they go almost straight to mapping. The `code` family is
the hard one and gets the heuristic extractor below.

**Code signal extraction.** Cheap, syntax-aware matching (e.g. `ast-grep` patterns
per language, or regex fallback) finds the candidates; the model never sees full
files, only extracted descriptors:

```go
type Signal struct {
	Kind    SignalKind // Event | Command | Handler | Projection | Topic | External
	Name    string     // class / type / topic name
	Fields  []Field    // name + type, when cheaply recoverable
	Source  string     // file:line, for provenance
	Hints   []string   // e.g. "@KafkaListener", "implements Projection", "calls HttpClient"
	Module  string     // package / directory — the bounded-context hint
}
```

Detectors are intentionally loose:

- **Events** — types named `*Event`, `*ed`/`*en` past-tense suffixes, classes
  annotated `@DomainEvent`, or published to an event bus.
- **Commands** — types named `*Command`, the input DTO of a command handler,
  `@CommandHandler`-annotated methods.
- **Handlers / automations** — methods subscribed to an event and dispatching a
  command (`@EventHandler`, `@KafkaListener`, saga steps) → candidate `automation`.
- **Projections / views** — classes that fold a stream of events into a read model
  (`*Projection`, `*ReadModel`, `*View`, repository writers keyed on events) →
  candidate `view` with a `subscribes [...]` list recovered from the handled types.
- **Topics / channels** — Kafka/SNS/SQS topic constants and producer/consumer wiring
  → cross-context delivery, i.e. `automation ... target context X` or `translation`.
- **External systems** — outbound HTTP/SDK calls, webhook controllers → candidate
  `translation` with `external_system "..."`.

**Condensing for context budget.** The signal set for a real system can still be
large. The condense step is deterministic plus, where needed, a cheap Haiku
summarization pass (`anthropic.claude-haiku-4-5`, `effort: low`):

1. **Bucket by bounded context.** Group signals by `Module`/package, then by
   directory proximity. Each bucket becomes a candidate `context`.
2. **Per-bucket summary.** For each bucket, emit a compact inventory — event names +
   fields, command names + fields, handler edges (event → command), projection
   subscriptions. Drop framework noise. If a bucket overflows, summarize it with
   Haiku into a short structured digest rather than truncating.
3. **Assemble a model brief.** Concatenate the per-bucket summaries into a single
   structured brief (JSON), well under the hard-model context window, and hand *that*
   to the mapping call. Provenance (`file:line`) is retained per item so the draft
   can carry source comments.

This per-context chunking matters: it keeps each summarization focused, makes the
context boundaries explicit before the model ever runs, and means the expensive
mapping call reasons over a clean inventory instead of raw code.

### Stage 2 — Map signals onto emod constructs

A single call to the hard model (`anthropic.claude-opus-4-8`, `effort: high`,
adaptive thinking) turns the condensed brief into an AST-shaped draft. It uses
**structured output** (the `Schema` field on `llm.GenerateRequest`) so the model
returns a JSON object conforming to emod's AST shape — no string-parsing of `.emod`
text. emod renders that AST to `.emod` with the existing formatter, exactly as 01
does. The JSON Schema mirrors `internal/ast/ast.go` (`Model` → `Context` →
`Aggregate`/`Slice` → `Command`/`Event`/`View`/`Automation`/`Translation`/`Flow`).

The system prompt encodes the mapping rules and emod house style (drawn from
`README.md`, `docs/proposal.md`, `docs/dsl-reference.md`):

| Detected in source | emod construct | Rule of thumb |
|--------------------|----------------|----------------|
| package / module / service | `context` | One bounded context per cohesive module. |
| event with a clear single owner + id key | `aggregate` (default) | `Context → Aggregate → Slice`. |
| event shared across handlers, tagged by multiple keys | `mode dcb` + `tags` + `decides_on` | When no single owner fits. |
| command class / handler input | `command` (imperative) | Keep the source name if already imperative. |
| domain event | `event` (past tense) | Keep the source name; the linter will flag bad ones. |
| handler: on event → dispatch command | `automation` (`trigger`/`command`/`target context`) | Cross-context → set `target context`. |
| projection / read model | `view` with `subscribes [...]` | Recover subscriptions from handled event types. |
| outbound integration / webhook | `translation` with `external_system` | External event gets `source external "..."`. |
| event → command edge within a context | `flow { command -> event }` | One flow per produced event. |

**Aggregate vs DCB.** Default to the aggregate style — it is the idiomatic, most
readable form and matches most code (one handler class, one stream key). Choose
`mode dcb` only when the signals show an event that several decisions read and that
is keyed by more than one identifier (the multi-tag, per-decision-boundary shape
described in `docs/dcb-proposal.md`). When the model emits DCB constructs, the
`dcb/*` linter rules (`dcb/untagged-event`, `dcb/query-too-broad`,
`dcb/single-tag-everywhere`, `dcb/orphan-tag-key` in
`internal/linter/descriptions.go`) keep the draft honest in the repair loop.

The model is instructed to **preserve source names verbatim** rather than "cleaning
them up." This is deliberate: a faithful draft is more useful for review, and it lets
emod's own linter — not the model's taste — be the thing that flags smells.

### Stage 3 — Repair (reused from 01)

The mapping output is fed straight into 01's `GenerateAndRepair` loop unchanged. The
deterministic oracle — parser + `internal/validator` + `internal/linter`, surfaced as
`internal/diagnostic` entries — runs over the rendered `.emod`. Validation errors
(unknown event referenced by a `subscribes`, an `automation` targeting a missing
context, a malformed `decides_on`) are fed back to the model as a repair turn until
the draft is clean or `maxAttempts` is exhausted.

Crucially, **lint findings are surfaced, not auto-fixed.** When the source had an
`OrderUpdatedEvent` class, the draft has an `OrderUpdated` event, and `emod lint`
reports `state-obsession` against it. Property-sourcing names (`Order` + `Status` +
`Changed` → `OrderStatusChanged`) trip `property-sourcing`; a `*Initiated` event
trips `command-in-disguise`; a command accidentally named in past tense trips
`command-past-tense`; a projection missing the `View` suffix trips `view-naming`; a
fan-out projection over many events trips `god-view`; a single-id event trips
`clickbait-event`. These are *inherited smells* — they are facts about the source
system worth showing a human, so import keeps them visible in the draft (as a final
`emod lint` report printed after writing the file) rather than silently rewriting
names the model can't be sure about. (Fixing them is [04](./04-lint-quickfixes-lsp.md)'s
job, on the user's terms.)

## Interface

A new `ai` command group, with `import` as its first subcommand, slotting into the
existing `urfave/cli` app in `internal/cli/app.go`:

```
emod ai import <path-or-glob> [flags]

  -o, --output <file>     Write the draft .emod here (default: stdout)
      --from <family>     code | eventcatalog | asyncapi | emlang | catalog | auto
                          (default: auto)
      --context <name>    Force a single bounded context name (skip auto-bucketing)
      --max-attempts <n>  Repair-loop attempts (default: 4)
      --effort <level>    Override model effort for the mapping pass
      --dry-run           Print the condensed model brief and exit (no model call)
```

Behavior:

- Resolves the path-or-glob, detects the family (unless `--from` is set), extracts and
  condenses, maps, repairs, and writes the draft.
- Prints a provenance header into the draft, mirroring the existing example's
  convention:

  ```emod
  # Reverse-engineered by `emod ai import` from ./services/billing (--from code).
  # DRAFT — inferred from source; review context boundaries, names, and flows.
  ```

- After writing, runs the standard `emod lint` over the result and prints the
  findings, so inherited smells are visible immediately.
- AI-gated exactly as the foundation describes: without Bedrock credentials
  configured, `emod ai` is simply unavailable; nothing in `validate`/`lint`/`fmt`/
  `diagram`/`export`/`lsp` gains a hard LLM dependency.

Example:

```console
$ emod ai import './services/inbound-email/**/*.go' --from code -o inbound.emod
detected 23 events, 11 commands, 6 handlers, 4 projections across 5 modules
condensed to 5 bounded contexts (InboundEmail, OutboundDelivery, CustomerCases,
  CollectionHolds, ConversationHoldScheduling)
mapping… repair attempt 1/4: 2 validation errors → fixed
wrote inbound.emod (validate: clean)

lint: 3 findings (inherited from source naming)
  inbound.emod:34  state-obsession   event "EmailConversationStatusUpdated"
  inbound.emod:61  clickbait-event   event "EmailConversationOpened" has only an id field
  inbound.emod:88  view-naming       view "EmailDecisionsTopic" should end in "View"

review the draft before committing — this is an inferred model, not a verified one.
```

## Worked Example

A small slice of a billing service in Go — an event, a command handler, a Kafka
listener that reacts, and a projection:

```go
// billing/events.go
type InvoiceIssuedEvent struct {
	InvoiceID  string
	CustomerID string
	AmountCents int64
	IssuedAt   time.Time
}

// billing/handlers.go
type IssueInvoiceCommand struct {
	CustomerID  string
	AmountCents int64
}

func (h *InvoiceHandler) Handle(cmd IssueInvoiceCommand) (InvoiceIssuedEvent, error) { … }

// billing/reactions.go
// @KafkaListener(topic = "billing.invoice-issued")
func (r *DunningReactor) OnInvoiceIssued(e InvoiceIssuedEvent) {
	r.payments.Send(ChargeCustomerCommand{CustomerID: e.CustomerID, AmountCents: e.AmountCents})
}

// billing/projections.go
// folds InvoiceIssuedEvent, PaymentCapturedEvent into a balance read model
type CustomerBalanceProjection struct { … }
```

Extracted signals: one event (`InvoiceIssuedEvent`, module `billing`), one command
(`IssueInvoiceCommand`), a handler edge (`IssueInvoiceCommand → InvoiceIssuedEvent`),
a reaction (`InvoiceIssuedEvent → ChargeCustomerCommand`, crossing into a `payments`
topic), and a projection subscribing to two events. The mapping pass produces, and
the repair loop confirms clean:

```emod
# Reverse-engineered by `emod ai import` from ./billing (--from code).
# DRAFT — inferred from source; review context boundaries, names, and flows.
model "Billing"

context "Billing" {
  aggregate "Invoice" {
    slice "Issue invoice" {
      command IssueInvoice {
        fields {
          customerId  string required
          amountCents int    required
        }
      }

      event InvoiceIssued {
        fields {
          invoiceId   string    required
          customerId  string    required
          amountCents int       required
          issuedAt    timestamp required
        }
      }

      flow {
        command -> event: IssueInvoice -> InvoiceIssued
      }
    }

    slice "Charge on invoice issued" {
      automation DunningReactor {
        trigger InvoiceIssued
        command ChargeCustomer
        target context Payments
      }
    }

    slice "Customer balance" {
      view CustomerBalanceView {
        fields {
          customerId   string required
          balanceCents int    required
        }
        subscribes [InvoiceIssued, PaymentCaptured]
      }
    }
  }
}
```

Note the faithful, useful transformations: the `Event`/`Command` suffixes are
dropped, the imperative `IssueInvoice` and past-tense `InvoiceIssued` are preserved
from the source, the `@KafkaListener` reaction became an `automation` with
`target context Payments`, and `CustomerBalanceProjection` became
`CustomerBalanceView`. If the source had instead named the event
`InvoiceStatusUpdatedEvent`, the draft would carry `InvoiceStatusUpdated` and the
post-write `emod lint` would flag it as `state-obsession` — the smell surfaced, not
silently invented away.

## Implementation Plan

**Phase 1 (S) — structured-artifact import.** Highest signal-to-effort, no heuristics
needed. Deterministic parsers for the `emlang`, `asyncapi`, `eventcatalog`, and
`catalog` families produce the condensed brief; mapping + 01's repair loop produce
the draft. Validate against the real
`examples/inbound_customer_comms_agentic_reply.emod` by importing the source
`*.emlang.yaml` and diffing — this is a known-good round-trip target. Wire the
`emod ai import` command and `--from` for these families.

**Phase 2 (M) — the mapping + condense core.** The AST-shaped JSON Schema, the
mapping system prompt and house-style rules, the per-context bucketing, and the Haiku
summarization fallback for oversized buckets. Add `--dry-run` (print the brief),
`--context`, and the post-write lint report. Unit-test the whole flow against a mock
`llm.Model` per the foundation's testing section (umbrella `TestImport`, `t.Run`
groups by stage, behavior-named scenarios, fresh fixtures, `testify/require`).

**Phase 3 (L) — source-code extraction.** The heuristic `Signal` extractors per
language (`ast-grep` patterns + regex fallback), framework-noise filtering, topic /
external-call detection, and DCB-vs-aggregate inference. Start with one language
(Go) end-to-end, then add Kotlin/Java/TypeScript detector packs. This is the largest
and least certain stage; ship it behind the already-working structured-artifact path.

## Risks & Mitigations

- **Inferred boundaries are wrong.** Context grouping from packages/modules is a
  heuristic; real domains don't always follow directory structure. *Mitigation:*
  `--context` to force a single context, `--dry-run` to inspect and correct the brief
  before spending a model call, and the explicit DRAFT header that sets expectations.
- **The model hallucinates fields, flows, or events not in the source.** *Mitigation:*
  ground the mapping in the extracted brief only; retain provenance (`file:line`) and
  instruct the model to mark items it could not source. The repair loop catches
  *structural* invention (a flow referencing a non-existent event), though not
  *plausible-but-wrong* invention — hence human review.
- **Large codebases blow the budget.** *Mitigation:* never feed raw files; the
  per-context bucket + Haiku summarize step keeps the brief bounded, and very large
  inputs degrade to per-context drafts rather than failing.
- **Smell-laden source produces a smell-laden model.** This is *expected and useful* —
  the linter reports it. *Mitigation:* surface, don't auto-fix; point users at
  [04](./04-lint-quickfixes-lsp.md) for guided cleanup.
- **Heuristic extractors are language- and framework-specific and will miss things.**
  *Mitigation:* treat extraction as best-effort signal, ship per-language detector
  packs incrementally, and keep the structured-artifact path (Phase 1) as the
  high-fidelity fallback for teams that have a catalog.
- **Users mistake the draft for ground truth.** *Mitigation:* the DRAFT provenance
  header, the post-write lint report, and docs that frame this as draft-assist.

## Open Questions

- **Granularity of `aggregate` vs DCB.** Should `--from code` default to never
  emitting DCB (simpler, always reviewable) and require an opt-in flag, or should it
  infer DCB when the multi-tag signal is strong? Leaning aggregate-by-default.
- **Multi-language repos.** When one import spans Go + TypeScript services, do we run
  one combined model with shared contexts, or one draft per service stitched after?
- **Provenance in the output.** Should source `file:line` be emitted as `.emod`
  comments per construct (useful for review, noisy in the file), behind a
  `--with-provenance` flag, or only in a sidecar map?
- **Re-import / merge.** Out of scope here, but if a user edits a draft and the source
  changes, is there a future path to a three-way merge rather than overwrite?
- **Confidence signalling.** Should low-confidence mappings (an `automation` inferred
  from a loosely-matched topic name) be marked in the draft so reviewers know where to
  look first?

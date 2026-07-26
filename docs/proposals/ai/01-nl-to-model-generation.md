# Natural-Language → `.emod` Generation

> Part of the [emod AI proposals](./README.md). Assumes the shared [LLM foundation](./00-llm-foundation.md) (the `llm.Model` port, Bedrock-backed Claude, and the generate → validate → lint → repair loop) and does not re-specify it.

## Problem

Writing a first `.emod` file from scratch is the steepest part of the learning
curve. A user knows their business process — "a customer emails us, we figure out
who they are, classify the message, and either auto-reply or open a case" — but
turning that into idiomatic event-modeling structure (contexts, slices, the
command → event → view/automation patterns, past-tense events, imperative
commands) takes fluency the user may not have yet. The blank page is where most
prospective users bounce off.

A naive "ask an LLM to write `.emod`" approach fails in a specific way: the model
produces plausible-looking output that is subtly wrong — events named in the
imperative, commands fanning out to five events, views subscribing to everything,
fields missing on a flow's referenced event, a context in `mode aggregate` that
sprinkles `tags` everywhere. None of this is caught by reading the text; all of it
is caught by `emod validate` and `emod lint`.

This is exactly the case emod is built to win. Unlike a typical generation task,
emod ships a **deterministic correctness oracle** — the parser, `internal/validator`,
`internal/linter`, and the bundled CUE schema (`emod schema`). The model does not
have to be right in one shot; it has to *converge* against a checker we already
own. NL → `.emod` generation is therefore the canonical user of the foundation's
`GenerateAndRepair` loop, and the feature that best demonstrates why emod's
determinism is the real engine, not the model.

## Goals

- A CLI command that turns a plain-English description into a valid, idiomatic
  `.emod` file: `emod ai generate "<description>" -o file.emod`.
- A stdin / interactive form for longer descriptions and for piping prose in.
- Output that is not merely *parseable* but *idiomatic* — clean against
  `internal/linter` rules (`state-obsession`, `command-past-tense`,
  `clickbait-event`, `god-view`, `left-chair`, `view-naming`, the `dcb/*` family),
  not just `internal/validator`.
- The validate → lint repair loop is bounded and transparent: the user sees how
  many attempts ran, what was fixed, and the token cost.
- Graceful, honest degradation when the loop cannot converge — emit the best
  candidate with the remaining diagnostics, never a silent or fabricated "success".
- Zero impact on existing commands; the feature is opt-in and absent without
  configured credentials (per the foundation's configuration section).

## Non-Goals

- Generating from existing code or structured artifacts — that is
  [proposal 02](./02-model-import-reverse-engineering.md).
- Editing or extending an *existing* `.emod` file — that is
  [proposal 10](./10-conversational-viewer-editing.md). This proposal is greenfield
  generation from prose only.
- Inventing new DSL surface. Generation targets the grammar exactly as documented
  in `README.md`, `docs/proposal.md`, and `docs/dcb-proposal.md`.
- Re-specifying the `llm.Model` port, the Bedrock adapter, model selection, or the
  `GenerateAndRepair` loop — all defined in the foundation.
- A general chat interface. This is a single-shot (internally multi-attempt)
  command, not a conversation.

## How It Works

### Where it plugs into the pipeline

Generation runs the existing pipeline *in reverse as a gate*. The normal flow is
`.emod` → lexer → parser → AST → validator + linter → diagnostics. Here the model
produces candidate `.emod` text and we feed it through that same pipeline; the
diagnostics it produces are the repair signal, not a user-facing error.

```
prose ──▶ llm.Model.Generate ──▶ candidate .emod text
                                        │
                                        ▼
                          parser → validator → linter
                            (internal/diagnostic.Entry[])
                                        │
                    ┌───────────────────┴───────────────────┐
                  clean                                   diagnostics
                    │                                         │
              write -o file                          repairRequest(...) ─┐
                                                                         │
                                              (loop, bounded by maxAttempts)
```

The package owning the model-facing logic lives under `internal/ai` (e.g.
`internal/ai/generate.go`), depending only on `llm.Model` plus the existing
`internal/validator`, `internal/linter`, and `internal/diagnostic`. The CLI command
in `internal/cli` is a thin wrapper, matching the structure of the existing
`RunValidate` / `RunLint` / `RunDiagram` actions in `internal/cli/app.go`.

### The AI flow specifics

**1. System prompt — teach the grammar and the house style.** A static system
prompt (embedded via `go:embed`, kept beside the generator) carries:

- The concrete grammar surface — `model`, `actor`, `context` (`mode aggregate|dcb|mixed`),
  `aggregate` + `stream`, `slice`, `command`, `event` (`fields`, `tags`),
  `view` (`subscribes [...]`), `automation` (`trigger`/`command`/`target context`),
  `translation` (`external_system`), `flow` (`command -> event: X -> Y`),
  `trigger UI`/`trigger Automation`, and the DCB `decides_on { events [...] where tag(...) }`.
- The idioms encoded as **rules to obey, phrased as the linter sees them**: events
  are past-tense business facts (not `...Updated`/`...Changed`/`...Initiated`),
  commands are imperative, views end in `View` and subscribe to fewer than five
  events, a command should not fan out to many events, events carry domain fields
  not just an ID. These map one-to-one onto the descriptions in
  `internal/linter/descriptions.go`, so the model is pre-conditioned to pass the
  same checker it will be graded against.

**2. Few-shot grounding from `examples/`.** The prompt includes one or two real,
lint-clean models from `examples/` as worked exemplars —
`examples/inbound_customer_comms_agentic_reply.emod` for a rich aggregate-style,
multi-context model, and `examples/dcb_model.emod` for the DCB style. These are the
canonical demonstration of structure (slices grouping a command, its events, and a
`flow`; automations crossing contexts via `target context`; translations wrapping
an `external_system`). Few-shot exemplars do more for idiomaticity than prose rules.

**3. Optional schema grounding.** For a stricter contract, the prompt can include
the output of `emod schema` (the bundled CUE definition) so the model sees the
field-level shape and naming constraints (e.g. the command/event/view name
patterns) the validator enforces. This is opt-in (`--ground-schema`) because it
costs input tokens; the few-shot examples already cover most of it.

**4. Structured output.** Per the foundation, the model returns data via the
strict-tool-use / JSON Schema path, not a free-form string. The
`GenerateRequest.Schema` field carries a small wrapper schema:

```json
{
  "type": "object",
  "required": ["emod"],
  "properties": {
    "emod":  { "type": "string", "description": "the full .emod source" },
    "notes": { "type": "string", "description": "assumptions made about the prose" }
  }
}
```

We deliberately do **not** ask the model to emit the AST as JSON and then render
it — `.emod` is the human-authored surface, `emod fmt` is the canonical
formatter, and emitting text keeps the model aligned with the examples it was
shown. The generator runs `emod fmt` on the accepted output so spacing and
ordering are normalized regardless of what the model produced.

**5. The repair loop (validate *and* lint).** This is the foundation's
`GenerateAndRepair`, with one emphasis specific to this feature: the oracle that
gates convergence is **validator + linter**, not validator alone. Producing valid
output is table stakes; producing *idiomatic* output is the differentiator, and
lint findings are where idiomaticity lives. The loop feeds both back:

```go
type genResult struct {
    Source  string
    Attempts int
    Usage   llm.Usage
    Remaining []diagnostic.Entry // empty on success
}

func Generate(ctx context.Context, m llm.Model, prose string, opts Options) (genResult, error) {
    req := initialRequest(prose, opts) // system prompt + few-shot + (optional) schema + the wrapper Schema
    var last string
    var total llm.Usage
    for attempt := 1; attempt <= opts.MaxAttempts; attempt++ {
        resp, err := m.Generate(ctx, req)
        if err != nil {
            return genResult{}, err
        }
        total = total.Add(resp.Usage)
        src := extractEmod(resp.Text) // the structured "emod" field
        last = src

        diags := check(src) // parser + internal/validator + internal/linter
        if len(diags) == 0 {
            formatted, _ := format(src) // emod fmt
            return genResult{Source: formatted, Attempts: attempt, Usage: total}, nil
        }
        // Errors first, then warnings, then info — the model fixes the blocking
        // issues before the stylistic ones, which keeps later attempts stable.
        req = repairRequest(req, src, sortBySeverity(diags))
    }
    return genResult{Source: last, Attempts: opts.MaxAttempts, Usage: total,
        Remaining: check(last)}, ErrNotConverged
}
```

`check` returns `[]diagnostic.Entry` (the type already used across the codebase,
carrying `RuleName`, `Severity`, `Line`, `Message`). The repair message renders
each entry the way the CLI already does — `file:line: [rule-name] message` — so the
model sees the exact rule it tripped (e.g. `[command-past-tense] ...` or
`[god-view] ...`) and can fix the *category*, not just the instance. Feeding the
**rule name** (not just the message) lets the model generalize the fix across the
whole model in one round, which is what keeps the attempt count low.

**6. Model and effort.** Generation is a hard reasoning task, so it uses
`anthropic.claude-opus-4-8` with `high` (or `xhigh`) effort and adaptive thinking,
per the foundation's selection table. Repair attempts reuse the same model — a
half-fixed model still needs full structural reasoning. (A future optimization
could route single-finding mechanical repairs to `anthropic.claude-haiku-4-5`, but
that is not assumed here.)

**7. Cost surfacing.** `Response.Usage` is accumulated across attempts and reported
at the end (input/output tokens, and a derived dollar figure consistent with the
existing bedrock-cost tooling). Cost is surfaced per run because the repair loop
can multiply token spend, and the user should see when a model needed four
attempts versus one.

**8. Degradation.** When the loop exhausts `MaxAttempts` without converging, the
command does **not** fail silently and does **not** pretend success. It writes the
best candidate to the output path with a clear banner stating it did not fully
converge, prints the remaining diagnostics (the same `emod validate` / `emod lint`
would print), and exits non-zero. The user gets a head start they can fix by hand,
plus an exact list of what is wrong — which is strictly better than a blank file.
`--strict` makes non-convergence write nothing and exit non-zero, for CI use.

## Interface

A new top-level `ai` command group in `internal/cli/app.go`, with `generate` as its
first subcommand. It mirrors the existing command structure (urfave/cli v2,
`LintError` handling for exit codes).

```
emod ai generate "<description>" [flags]

Flags:
  -o, --output <file>     write the .emod to a file (default: stdout)
      --stdin             read the description from stdin instead of an argument
      --attempts <n>      max repair attempts (default 4)
      --effort <level>    low|medium|high|xhigh (default: EMOD_AI_EFFORT or high)
      --ground-schema     include `emod schema` (CUE) output in the prompt
      --strict            exit non-zero and write nothing if it does not converge
      --show-cost         print token usage and cost summary (default on for a TTY)
      --json              machine-readable result (source, attempts, usage, remaining)
```

Example invocation:

```
$ emod ai generate \
    "Customers email us. We identify who they are, classify the message, then \
     either auto-reply or open a support case. While we're replying, pause dunning." \
    -o inbound.emod

generating with anthropic.claude-opus-4-8 (effort=high)...
  attempt 1: 3 diagnostics (1 error, 2 warnings) — repairing
  attempt 2: 1 diagnostic (1 warning: god-view) — repairing
  attempt 3: clean
wrote inbound.emod  (3 contexts, 7 slices)
tokens: 11,240 in / 4,980 out across 3 attempts  (~$0.21)
```

Interactive / piped form:

```
$ emod ai generate --stdin -o inbound.emod < process-notes.txt
$ pbpaste | emod ai generate --stdin
```

The `--json` form returns the structured result so the command can be scripted or
called from the LSP / viewer later:

```json
{
  "attempts": 3,
  "converged": true,
  "usage": { "input_tokens": 11240, "output_tokens": 4980, "cost_usd": 0.21 },
  "remaining": [],
  "source": "model \"Inbound Customer Comms\" { ... }"
}
```

## Worked Example

Prose in:

> When a customer emails us, we receive the message, then identify which customer
> it is. Once identified, we classify it. If it's a normal question we draft an
> auto-reply; if it looks like a complaint we open a support case instead.

A reasonable converged output (abbreviated — the real file would be `fmt`-clean):

```emod
model "Inbound Customer Comms"

actor "Customer"
actor "Agent"

context "InboundEmail" {
  aggregate "EmailConversation" {
    slice "Receive inbound message" {
      trigger Automation "InboxRoute" {
        actor Customer
      }

      command ReceiveInboundMessage {
        fields {
          mailbox    string    required
          bodyRef    string    required
          receivedAt timestamp required
        }
      }

      event InboundMessageReceived {
        fields {
          mailbox  string required
          bodyRef  string required
        }
      }

      flow {
        command -> event: ReceiveInboundMessage -> InboundMessageReceived
      }
    }

    slice "Identify customer" {
      command IdentifyCustomer {
        fields {
          bodyRef    string required
          strategy   string required
        }
      }

      event CustomerIdentified {
        fields {
          customerId UUID   required
          strategy   string required
        }
      }

      automation Identify {
        trigger InboundMessageReceived
        command IdentifyCustomer
        target context InboundEmail
      }

      flow {
        command -> event: IdentifyCustomer -> CustomerIdentified
      }
    }

    slice "Classify message" {
      command ClassifyMessage {
        fields {
          customerId UUID   required
          bodyRef    string required
        }
      }

      event MessageClassified {
        fields {
          category   string required
          confidence number required
        }
      }

      automation Classify {
        trigger CustomerIdentified
        command ClassifyMessage
        target context InboundEmail
      }

      flow {
        command -> event: ClassifyMessage -> MessageClassified
      }
    }
  }
}
```

What the repair loop earns here, beyond mere validity:

- A first draft might name the event `MessageClassificationUpdated` — tripped by
  `state-obsession`; fed back, the model renames it to the business fact
  `MessageClassified`.
- It might name the command `MessageReceived` (past tense) — `command-past-tense`
  forces `ReceiveInboundMessage`.
- An early `CustomerIdentified` carrying only `customerId` trips `clickbait-event`;
  the model adds `strategy`.
- If the model collapses "draft reply" and "open case" into one slice whose command
  fans out to several events, `left-chair` nudges it toward separate slices/commands
  (the auto-reply vs. escalation split, as in
  `examples/inbound_customer_comms_agentic_reply.emod`).

None of these are visible by reading the first draft; all are caught by the
deterministic oracle and fixed before the file is written.

## Implementation Plan

**Phase 1 — Core generator (M).** `internal/ai/generate.go`: the `Generate`
function over `llm.Model`, the wrapper output schema, the static system prompt
(`go:embed`), few-shot loading from `examples/`, and the `check` adapter that runs
parser + `internal/validator` + `internal/linter` and returns `[]diagnostic.Entry`.
Reuse the foundation's `GenerateAndRepair` shape. Unit-tested against a mock
`llm.Model` (canned responses driving 0-attempt, N-attempt-converge, and
never-converge paths), per the repo's Go test conventions.

**Phase 2 — CLI surface (S).** `emod ai generate` in `internal/cli`: argument and
`--stdin` input, `-o`/stdout output, `--attempts`/`--effort`/`--strict` flags,
progress output, and `--json`. Run `emod fmt` on accepted output. Wire `LintError`
exit-code handling consistently with the other actions in `internal/cli/app.go`.

**Phase 3 — Idiomaticity and cost (S).** Severity-ordered diagnostic feedback
(errors → warnings → info), rule-name-in-feedback so the model fixes categories,
cost accumulation and `--show-cost` summary, and the non-convergence banner +
remaining-diagnostics output.

**Phase 4 — Schema grounding and polish (M).** `--ground-schema` to inject
`emod schema` CUE output; prompt tuning against a corpus of prose → expected-shape
cases; a small regression harness that asserts generated output is lint-clean for a
fixed set of prompts (using a recorded/mock model so it runs without a network).

**Phase 5 — DCB awareness (M/L).** Teach the generator to detect cross-cutting,
per-decision consistency from the prose ("a decision that spans both a course and a
student") and emit `mode dcb` with `tags` and `decides_on`, graded against the
`dcb/*` lint rules. Overlaps with [proposal 05](./05-dcb-modeling-assistant.md);
sequence after it lands.

## Risks & Mitigations

- **Loop never converges on hard prose.** Bound by `--attempts`; degrade by writing
  the best candidate plus remaining diagnostics and exiting non-zero (or writing
  nothing under `--strict`). Honest partial output beats a fake green.
- **Lint-clean but semantically wrong.** Passing validate + lint proves the model
  is *valid and idiomatic*, not that it captures the user's intent. The model is
  not the authority on the domain. Mitigate by emitting the `notes` field
  (assumptions made), keeping output human-editable `.emod`, and pointing the user
  at `emod diagram --serve` to eyeball the flow. Semantic review is the separate
  job of [proposal 03](./03-semantic-model-reviewer.md).
- **Token cost from multi-attempt runs.** Surface cost every run; bound attempts;
  prefer few-shot over `--ground-schema` by default; consider routing trivial
  single-finding repairs to `anthropic.claude-haiku-4-5` later.
- **Prompt rot as the grammar evolves.** The system prompt and few-shot examples
  duplicate knowledge that lives in the grammar and `internal/linter/descriptions.go`.
  Mitigate by deriving the rule list in the prompt from `RuleDescription` where
  feasible, sourcing few-shot from real `examples/` files, and the Phase 4
  regression harness catching drift.
- **Non-determinism in tests.** Never call a live model in tests; the mock
  `llm.Model` returns canned candidates, so repair-loop behavior is fully
  deterministic and asserted with `testify/require`.
- **Over-trust by users.** A confident-looking model can read as authoritative.
  The progress output explicitly states attempt count and that the result is
  validated-and-linted, *not* verified against intent.

## Open Questions

- **Default attempt budget.** Is 4 the right ceiling for opus-4-8 at `high`
  effort? Needs measurement against a prompt corpus (Phase 4) — too low wastes a
  near-miss, too high burns tokens on hopeless prose.
- **Schema grounding by default?** `--ground-schema` is opt-in to save tokens, but
  if few-shot alone leaves a class of field-shape errors, it may be worth making
  default-on and trimming the few-shot instead.
- **One-shot vs. clarifying questions.** Should `generate` ever ask the user a
  clarifying question (which contexts? which actors?) before generating, or stay a
  pure one-shot command and leave conversation to
  [proposal 10](./10-conversational-viewer-editing.md)? Leaning one-shot for now.
- **Aggregate vs. DCB inference.** When prose is ambiguous about consistency
  boundaries, should the generator default to aggregate style (the conservative,
  widely-understood form) and only emit DCB when the prose clearly signals
  per-decision boundaries? Current lean: yes, default aggregate.
- **Reusing accepted output as few-shot.** Could converged, user-approved outputs
  feed back into the example corpus over time? Tempting, but risks compounding
  whatever stylistic bias the model already has — defer.
- **Partial-file generation.** Should there be a mode that generates a single
  context or slice to paste into an existing file? That edges into proposal 10's
  territory; keep this proposal greenfield-only.

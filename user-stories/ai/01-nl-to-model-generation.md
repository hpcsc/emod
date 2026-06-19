# AI: Natural-Language → .emod Generation

## Overview

Writing a first `.emod` file from scratch is the steepest part of the learning curve. An author knows their business process — "a customer emails us, we identify who they are, classify the message, then either auto-reply or open a case" — but turning that into idiomatic event-modeling structure (contexts, slices, the command → event → view/automation patterns, past-tense events, imperative commands) takes fluency they may not have yet. A naive "ask a model to write `.emod`" approach fails in a specific way: it produces plausible-looking output that is subtly wrong — events named in the imperative, commands fanning out to many events, views subscribing to everything, fields missing on a flow's referenced event — none of which is caught by reading the text, all of which is caught by `emod validate` and `emod lint`.

This feature adds an `emod ai generate` command that turns a plain-English description into a `.emod` file that is not merely parseable but idiomatic. It is the canonical user of emod's generate → validate → lint → repair loop: the tool feeds candidate output through its own deterministic oracle (parser, validator, linter, bundled CUE schema) and feeds the diagnostics back until the output converges or the attempt budget runs out. The deterministic checker — not the model — is the engine that guarantees quality. The feature is opt-in: it is absent without configured credentials, and it adds no dependency to the existing `validate`/`lint`/`diagram`/`export`/`lsp` paths.

## Goals

- Turn a plain-English description into a valid, idiomatic `.emod` file via `emod ai generate "<description>" -o file.emod`.
- Accept descriptions from a command argument, from `--stdin`, and from piped prose.
- Produce output that is lint-clean (against `state-obsession`, `command-past-tense`, `clickbait-event`, `god-view`, `left-chair`, `view-naming`, and the `dcb/*` family) — not merely validator-clean.
- Bound the repair loop and make it transparent: the user sees attempt count, what was fixed, and the token cost.
- Degrade honestly when the loop cannot converge — emit the best candidate with the remaining diagnostics, never a silent or fabricated success.
- Keep generated output normalized through `emod fmt` so spacing and ordering are consistent regardless of what the model produced.

## User Stories

### US-001: Generate a valid .emod file from a prose description
**Description:** As a model author, I want to run `emod ai generate "<description>"` and get a single `.emod` file that passes `emod validate`, so that I have a structured starting point instead of a blank page.

**Acceptance Criteria:**
- [ ] `emod ai generate "<description>"` prints generated `.emod` source to stdout by default, and `-o file.emod` writes it to the named file instead
- [ ] The emitted source contains a `model` declaration and at least one `context` with at least one `slice` reflecting the described process
- [ ] Running `emod validate` on the generated output exits 0 with no errors
- [ ] On success the command exits 0; the generated file path and a one-line summary (e.g. context and slice counts) are reported
- [ ] When AI credentials are not configured, `emod ai generate` reports that the feature is unavailable and exits non-zero, while `emod validate`, `emod lint`, `emod diagram`, and `emod export` continue to work unchanged
- [ ] The model selection and effort follow the configured defaults; an `--effort low|medium|high|xhigh` flag overrides the per-run effort

**Context:** This is the foundational slice — get a single valid `.emod` produced and written, with the opt-in/no-credentials boundary respected. Idiomaticity, the repair loop, and cost reporting build on top of this. Generation targets the grammar exactly as documented; it does not invent DSL surface.

---

### US-002: Self-correct generated output until it validates and lints clean
**Description:** As a model author, I want the generator to feed validator and linter diagnostics back to the model and retry, so that the final output is idiomatic — not just parseable.

**Acceptance Criteria:**
- [ ] When a candidate produces validator or linter diagnostics, the generator retries with those diagnostics as input rather than returning the flawed candidate
- [ ] Both `emod validate` and `emod lint` gate convergence: the accepted output is clean under `emod lint` (no `state-obsession`, `command-past-tense`, `clickbait-event`, `god-view`, `left-chair`, or `view-naming` findings), not merely valid
- [ ] The loop is bounded by a maximum number of attempts, configurable via `--attempts <n>` with a documented default
- [ ] Each retry is driven by the specific rule names and locations that were tripped (e.g. an event named in the imperative is renamed to a past-tense business fact; a command in past tense is renamed to imperative)
- [ ] Blocking validator errors are resolved before stylistic lint warnings across the retry sequence
- [ ] The accepted output is normalized through `emod fmt` before it is written or printed

**Context:** This is the heart of the feature — the validate → lint repair loop. Producing valid output is table stakes; producing idiomatic output is the differentiator, and lint findings are where idiomaticity lives. Feeding the rule name (not just the message) lets the model fix a whole category in one round, which keeps the attempt count low. Do not prescribe how the loop is structured internally; describe only what the user can observe (clean final output, bounded attempts).

**Depends on:** US-001

---

### US-003: Report attempt progress and token cost per run
**Description:** As a model author, I want to see how many attempts ran, what was fixed each round, and the token cost, so that I understand what a generation cost me and trust the result is checked.

**Acceptance Criteria:**
- [ ] During a run the command reports per-attempt progress, including the diagnostic count (with severity breakdown) and which rules were repaired
- [ ] On completion the command reports total input and output token usage and a derived dollar cost, accumulated across all attempts
- [ ] The cost summary is shown by default on an interactive terminal and can be forced or suppressed via `--show-cost`
- [ ] The progress output states that the result is validated-and-linted, not verified against the author's intent, so the result is not read as authoritative
- [ ] Token usage and cost reflect every attempt, so a four-attempt run reports visibly higher cost than a one-attempt run

**Context:** The repair loop can multiply token spend, so cost must be surfaced every run and attributed across attempts. The honesty note in progress output guards against over-trust of confident-looking output. Keep wording about cost consistent with how the project already reports Bedrock spend.

**Depends on:** US-002

---

### US-004: Degrade honestly when the loop cannot converge
**Description:** As a model author, I want a non-converging run to give me the best partial `.emod` plus an exact list of what is still wrong, so that I get a head start I can finish by hand instead of a fake success or a blank file.

**Acceptance Criteria:**
- [ ] When the attempt budget is exhausted without a clean candidate, the command writes the best candidate to the output path with a banner clearly stating it did not fully converge
- [ ] The remaining diagnostics are printed in the same `file:line: [rule-name] message` form that `emod validate` and `emod lint` produce
- [ ] A non-converging run exits non-zero
- [ ] `--strict` makes a non-converging run write nothing and exit non-zero, for CI use
- [ ] The command never reports success, and never writes a silently-modified or fabricated "clean" result, when diagnostics remain
- [ ] Accumulated token cost is still reported for a non-converging run

**Context:** Honest partial output beats a fake green. This is the key error scenario for the repair loop and warrants its own slice. The banner and the remaining-diagnostics list mirror the existing CLI diagnostic rendering so the author can act on them with familiar tooling.

**Depends on:** US-002

---

### US-005: Accept descriptions from stdin and piped prose
**Description:** As a model author, I want to feed a longer description through `--stdin` or a pipe, so that I can generate from notes in a file or from the clipboard without quoting a long argument on the command line.

**Acceptance Criteria:**
- [ ] `emod ai generate --stdin -o file.emod < notes.txt` reads the description from standard input and writes the result to the named file
- [ ] Prose piped in (e.g. `pbpaste | emod ai generate --stdin`) is generated from with the same behavior as an argument description
- [ ] When `--stdin` is set, any positional description argument is rejected with a clear error rather than silently ignored
- [ ] When neither a description argument nor `--stdin` input is provided, the command reports the missing input and exits non-zero
- [ ] Progress, cost reporting, and convergence handling behave identically whether the description came from an argument or from stdin

**Context:** Longer processes are easier to describe in a file or paragraph than in a shell-quoted argument. This is a thin input-surface slice on top of the core generator; it must not change the generation, repair, or reporting behavior.

**Depends on:** US-001

---

### US-006: Emit machine-readable results for scripting
**Description:** As a tooling author, I want `--json` output containing the source, attempt count, convergence status, usage, and remaining diagnostics, so that I can script generation or call it from other tools.

**Acceptance Criteria:**
- [ ] `--json` emits a single machine-readable object containing the generated source, attempt count, a convergence boolean, token usage with derived cost, and the remaining diagnostics
- [ ] On a converged run the convergence flag is true and the remaining-diagnostics list is empty
- [ ] On a non-converged run the convergence flag is false and the remaining diagnostics are listed with rule name, severity, and location
- [ ] In `--json` mode human-oriented progress text is not interleaved into the machine-readable object on the same stream
- [ ] Exit codes in `--json` mode match the non-JSON behavior (0 on convergence, non-zero on non-convergence or unavailability)

**Context:** A structured result lets the command be called from CI, scripts, or later editor/viewer surfaces. Keep the field set aligned with what the human-readable run already reports (attempts, usage, remaining), so the two surfaces never disagree.

**Depends on:** US-003, US-004

---

### US-007: Ground generation in idiomatic exemplars and the bundled schema
**Description:** As a model author, I want generation grounded in real lint-clean example models, and optionally in the bundled CUE schema, so that output matches the house style and the validator's field-shape constraints with fewer repair attempts.

**Acceptance Criteria:**
- [ ] Generation is grounded in real, lint-clean models drawn from the project's `examples/`, so output reflects idiomatic structure (slices grouping a command with its events and `flow`, automations crossing contexts via `target context`, translations wrapping an `external_system`)
- [ ] `--ground-schema` additionally grounds generation in the output of `emod schema` (the bundled CUE definition) so field-level shapes and name patterns are respected
- [ ] `--ground-schema` is off by default, and its effect on token cost is reflected in the per-run cost summary
- [ ] A run with `--ground-schema` and a run without it both produce output that passes `emod validate` and `emod lint`
- [ ] The grounding examples are sourced from the actual `examples/` files rather than being separately maintained, so they stay current as those files change

**Context:** Few-shot exemplars do more for idiomaticity than prose rules, and the schema is opt-in because it costs input tokens that the examples already largely cover. This slice improves convergence quality and is where prompt-drift mitigation lives (sourcing exemplars from real files). Do not specify prompt internals — only the observable grounding behavior and the `--ground-schema` flag.

**Depends on:** US-002

---

### US-008: Generate DCB-style models when prose signals per-decision boundaries
**Description:** As a model author describing a cross-cutting decision (e.g. one that spans both a course and a student), I want the generator to emit a `mode dcb` context with `tags` and `decides_on`, so that per-decision consistency boundaries are modeled idiomatically instead of being forced into artificial aggregates.

**Acceptance Criteria:**
- [ ] When the prose clearly signals a per-decision consistency boundary, the generated output uses a `mode dcb` context with tagged events and a `decides_on` query
- [ ] When the prose does not clearly signal per-decision boundaries, the generator defaults to aggregate style
- [ ] DCB output is clean against the `dcb/*` lint rules (e.g. no `dcb/untagged-event`, no `dcb/query-too-broad`)
- [ ] DCB output passes `emod validate` and `emod lint`, and is normalized by `emod fmt`
- [ ] The convergence, cost-reporting, and non-convergence behaviors apply unchanged to DCB generation

**Context:** This extends the generator to the DCB style, graded against the `dcb/*` rules just as aggregate output is graded against the core rules. The conservative default (aggregate unless the prose clearly signals DCB) avoids surprising authors with the less familiar form. This slice overlaps with the separate DCB modeling-assistant work and should be sequenced after the core generate/repair loop is solid.

**Depends on:** US-002, US-007

## Non-Goals (Out of Scope)

- Generating from existing code or structured artifacts (reverse-engineering) — covered by a separate proposal.
- Editing or extending an existing `.emod` file — this proposal is greenfield generation from prose only; conversational editing is a separate proposal.
- A partial-file mode that emits a single context or slice to paste into an existing file — edges into the conversational-editing proposal; keep this greenfield-only.
- A general chat or multi-turn conversation interface; `generate` is a single command (internally multi-attempt), not a dialogue.
- Asking the user clarifying questions before generating (which contexts? which actors?) — leaning one-shot for now.
- Inventing new DSL surface; generation targets the documented grammar exactly.
- Semantic review of whether the model captures the author's true intent — passing validate + lint proves valid and idiomatic, not correct against intent; semantic review is a separate proposal.
- Re-specifying the shared LLM port, the Bedrock adapter, model selection, or the repair-loop primitive — all defined in the AI foundation.

## Open Questions

- **Default attempt budget.** What is the right ceiling for `--attempts`? Too low wastes a near-miss; too high burns tokens on hopeless prose. Assumption: a small default (around 4) is reasonable, to be tuned against a prompt corpus.
- **Schema grounding by default.** `--ground-schema` is opt-in to save tokens. If example grounding alone leaves a class of field-shape errors, schema grounding may be worth making default-on. Assumption: keep it opt-in initially.
- **Clarifying questions.** Should `generate` ever ask a clarifying question before generating? Assumption: stay a pure one-shot command and leave conversation to the separate conversational-editing proposal.
- **Aggregate vs. DCB inference.** When prose is ambiguous about consistency boundaries, the generator defaults to aggregate style and emits DCB only when the prose clearly signals per-decision boundaries. Assumption: this conservative default is correct.
- **Reusing accepted output as few-shot grounding.** Could converged, user-approved outputs feed back into the example corpus over time? Assumption: defer — it risks compounding stylistic bias.
- **Assumption — repair feedback channel.** Acceptance criteria assume the user observes per-attempt diagnostics and final cleanliness, without prescribing how diagnostics are fed back internally; that mechanism is the implementer's choice within the shared repair-loop primitive.
- **Assumption — no network in automated verification.** Acceptance criteria describe observable command behavior and do not require a live model; deterministic verification of the repair loop is expected to use a canned/recorded model, consistent with the foundation's testing stance.

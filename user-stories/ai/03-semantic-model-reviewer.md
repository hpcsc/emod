# AI: Semantic Model Reviewer

## Overview

`emod lint` is fast, deterministic, and shallow: every rule (`state-obsession`, `command-in-disguise`, `god-view`, `clickbait-event`, `dcb/untagged-event`, and the rest) is a name match, a count threshold, or a structural presence check. Those rules measure the *form* of a model and are never wrong about what they measure — but they cannot judge its *meaning*. An event named `OrderProcessed` is past-tense with a rich payload, so every regex stays silent, yet it names a step in code rather than a fact in the business.

This feature adds an `emod ai review` command that reads an existing `.emod` file and emits *semantic* modeling findings the regex linter cannot produce: hollow events, incohesive aggregate/context boundaries, missing failure events or intermediate slices, commands doing two jobs, and ubiquitous-language drift. Findings flow through the same diagnostics pipeline as `emod lint` — same severities, same text/JSON formats, same editor surface — so consumers do not learn a second format. The reviewer complements the deterministic linter; it never replaces it.

Because the reviewer is probabilistic, the whole design favours precision over recall: a noisy reviewer is worse than none. Findings carry a confidence score, are capped below the severity that breaks a build, are grounded to real positions, and can be re-checked by an adversarial self-check pass. The feature is opt-in (requires AI configuration), usable both interactively and in CI, and never rewrites the file — it advises with a direction, not an edit.

## Goals

- Provide an `emod ai review <file>` command that emits semantic modeling findings the regex linter cannot produce.
- Emit findings through the same diagnostics surfaces as `emod lint`: CLI text, CLI JSON, and the editor (LSP).
- Classify findings into a stable, greppable, `ai/`-namespaced taxonomy of semantic smells.
- Attach a confidence and a capped severity to every finding so probabilistic results never block a build by themselves.
- Keep precision high: filter low-confidence findings, drop hallucinated locations, and never duplicate deterministic lint rules.
- Make the reviewer usable both interactively (exploratory) and in CI (stable, gated, reproducible).
- Keep all existing `validate`/`lint`/`diagram`/`export` paths working with no AI configuration present.

## User Stories

### US-001: Run a semantic review on an existing model
**Description:** As a model author, I want to run `emod ai review <file>` on an existing `.emod` file and get semantic modeling findings, so that I can catch domain-meaning problems a name-based linter cannot see.

**Acceptance Criteria:**
- [ ] `emod ai review <file>` reads an existing `.emod` file and prints zero or more findings as text output
- [ ] Each finding shows the file and line it refers to, a rule id, a one-sentence message describing what is wrong and why, and a confidence value
- [ ] The command runs on the whole model (not a fragment) and findings can reference any element in the file
- [ ] A model with no detectable semantic smells produces no findings and a clear "clean" result
- [ ] A summary line reports how many findings were emitted
- [ ] Running on a file that does not parse produces a clear error pointing at the parse problem, and no AI review is attempted

**Context:** The reviewer reads a model and reports; it does not generate or repair a model. The example `examples/inbound_customer_comms_agentic_reply.emod` passes most regex rules yet contains real semantic smells (e.g. a command that both classifies and detects escalation), and is a good manual sanity check.

### US-002: Classify findings into a stable semantic-smell taxonomy
**Description:** As a model author, I want each finding tagged with a rule id from a small, consistent taxonomy of semantic smells, so that I can scan, filter, and grep findings the same way I do for lint rules.

**Acceptance Criteria:**
- [ ] Every finding carries a rule id drawn from a defined taxonomy covering at least: hollow event (past-tense, well-formed name that names a code step, not a business fact), boundary smell (incohesive aggregate or context grouping), missing event (a process missing a failure/rejection event), missing slice (a gap where an intermediate slice should exist), command overloaded (one command doing two distinct jobs), weak naming (domain-weak naming beyond the regex rules), and language drift (inconsistent ubiquitous language across the model)
- [ ] Every finding rule id is namespaced with an `ai/` prefix so it never collides with a deterministic lint rule name and is trivially filterable
- [ ] A smell that does not fit the named taxonomy is emitted under a single documented catch-all id rather than a free-text category
- [ ] Identical smells across runs cluster under the same rule id rather than varying wording

**Context:** Deterministic lint rules (`state-obsession`, `command-in-disguise`, etc.) have no `ai/` prefix, so the prefix alone separates the two families. The catch-all is an escape hatch; frequently recurring catch-all clusters are candidates to promote into named ids later (see Open Questions).

**Depends on:** US-001

### US-003: Detect semantic smells the regex linter cannot
**Description:** As a model author, I want the reviewer to surface domain-judgement smells — hollow events, wrong boundaries, missing events/slices, overloaded commands, weak/inconsistent naming, and flow gaps — so that my model is a good map of the domain, not just structurally valid.

**Acceptance Criteria:**
- [ ] A model that is clean under every regex rule but has a command doing two distinct jobs produces a `command-overloaded` finding naming both jobs
- [ ] A command that emits only a success event when a failure/rejection outcome is plausible produces a `missing-event` finding naming the absent outcome
- [ ] A flow that jumps between states with an implied intermediate step missing produces a `missing-slice` finding describing the gap
- [ ] An event that is well-formed and past-tense but names a code step rather than a business fact produces a `hollow-event` finding
- [ ] An aggregate or context grouping that lacks cohesion or merges separate concerns produces a `boundary-smell` finding
- [ ] Inconsistent naming for the same notion across contexts (e.g. differing identifier names or command verbs) produces a `language-drift` or `weak-naming` finding identifying the inconsistent terms
- [ ] No finding duplicates a deterministic lint rule: a smell already covered by `state-obsession`, `command-in-disguise`, `property-sourcing`, `command-past-tense`, `view-naming`, `left-chair`, `god-view`, `clickbait-event`, or any `dcb/*` rule is not re-reported by the reviewer

**Context:** The division of labour is deliberate: if a regex or a count can decide it, the deterministic linter owns it; the reviewer is for the residue that only a reader of the domain can judge. This story is the heart of the feature.

**Depends on:** US-002

### US-004: Each finding includes a direction and located evidence
**Description:** As a model author, I want each finding to point at a precise location, cite the evidence that led to it, and suggest a direction, so that I can act on it without the tool rewriting my file.

**Acceptance Criteria:**
- [ ] Each finding resolves to a real position in the model (file, line, and where available a column and span) corresponding to an actual element
- [ ] Each finding includes a suggested direction phrased as guidance (e.g. "split into two slices", "add a failure event") and never as a literal replacement for the file
- [ ] Each finding cites evidence naming what in the model led to it
- [ ] The reviewer never modifies, formats, or rewrites the input file
- [ ] A finding whose cited location does not resolve to a real element position in the model is dropped rather than shown

**Context:** Findings are advisory and located, not patches. Turning a finding into an edit is a separate proposal (lint quick-fixes); the reviewer only reviews. Position grounding doubles as a hallucination filter — a citation that does not resolve is a tell.

**Depends on:** US-001

### US-005: Confidence and capped severity on every finding
**Description:** As a model author, I want every finding to carry a confidence score and a severity that can never reach the build-breaking level, so that a probabilistic finding never blocks a build the way a deterministic error does.

**Acceptance Criteria:**
- [ ] Every finding carries a confidence value between 0.0 and 1.0
- [ ] Every finding has a severity of at most `warning`; the reviewer never emits an `error`-level finding
- [ ] Confidence is shown in text output and present in JSON output for every finding
- [ ] Only deterministic lint errors (e.g. `left-chair`, `god-view`, `dcb/untagged-event`) can reach `error` severity; AI findings remain advisory regardless of confidence

**Context:** Capping severity keeps the trust boundary clear: exit codes are driven by deterministic findings; AI findings advise. A false positive must never break a build on its own.

**Depends on:** US-001

### US-006: Filter findings by confidence threshold
**Description:** As a model author, I want low-confidence findings filtered out by default and a flag to tune the threshold, so that I see fewer, surer findings and can dial precision against recall.

**Acceptance Criteria:**
- [ ] Findings below a default confidence threshold of 0.7 are hidden by default
- [ ] `--min-confidence <value>` overrides the threshold for a run
- [ ] The summary line reports how many findings were hidden below the active threshold
- [ ] Raising the threshold can only reduce or hold the set of shown findings; lowering it can only increase or hold it
- [ ] Findings at or above the active threshold are always shown

**Context:** The confidence threshold is the primary recall/precision dial. The default leans toward precision because a reviewer that cries wolf gets ignored.

**Depends on:** US-005

### US-007: Filter findings by severity
**Description:** As a model author, I want to limit output to findings at or above a chosen severity, so that I can focus on the more serious advisories.

**Acceptance Criteria:**
- [ ] `--severity <level>` shows only findings at or above the given severity level
- [ ] With no `--severity` flag, all findings that pass the confidence threshold are shown
- [ ] The flag accepts the same severity vocabulary used by `emod lint`
- [ ] The summary count reflects only the findings shown after severity filtering

**Depends on:** US-005

### US-008: JSON output matching the lint format
**Description:** As a CI author, I want `emod ai review --format json` to emit findings in the same JSON shape as `emod lint --format json`, so that my existing tooling parses AI findings without changes.

**Acceptance Criteria:**
- [ ] `emod ai review <file> --format json` emits each finding as the same core JSON entry shape produced by `emod lint --format json` (file, line, rule, severity, message)
- [ ] `--format text` produces the human-readable text output and is the default
- [ ] Confidence and direction are present in the JSON output as additional fields without changing the shape of the shared core fields
- [ ] JSON output is valid and parseable when there are zero findings
- [ ] The `--format` flag accepts the same values as the corresponding `emod lint` flag

**Depends on:** US-004, US-005

### US-009: Opt-in with graceful degradation when AI is not configured
**Description:** As an emod user without AI credentials, I want `emod ai review` to fail clearly while every existing command keeps working, so that adopting emod never forces AI configuration on me.

**Acceptance Criteria:**
- [ ] Running `emod ai review` with no AI configuration present exits with a non-zero code and a clear message stating that AI features require configuration
- [ ] The message names what configuration is missing rather than printing a raw error or stack trace
- [ ] `emod validate`, `emod lint`, `emod diagram`, and `emod export` continue to work with no AI configuration present and gain no dependency on it
- [ ] No AI calls are made for any command other than the `ai` command group

**Context:** All AI features are opt-in per the foundation. The `ai` command group is a new namespace; the always-available commands must be unaffected.

**Depends on:** US-001

### US-010: CI-friendly exit codes
**Description:** As a CI author, I want `emod ai review` to use predictable exit codes that never break a build solely on a probabilistic finding, so that I can run it as an advisory gate.

**Acceptance Criteria:**
- [ ] `emod ai review` returns exit code 0 when no findings survive the active confidence threshold
- [ ] It returns exit code 1 when one or more findings survive the threshold
- [ ] It never returns the exit code reserved for deterministic errors (the code `emod lint` returns for `error`-level findings)
- [ ] Exit-code behaviour is consistent between text and JSON output modes

**Context:** Exit codes follow `lint`'s convention but, because AI findings cap at `warning`, `review` never reaches the deterministic-error exit code. This keeps a flaky finding from failing a build that only deterministic errors should fail.

**Depends on:** US-005, US-009

### US-011: Reproducible reviews for CI via caching
**Description:** As a CI author, I want a review of an unchanged model to return the same findings on re-run without re-querying the model, so that AI review is stable and cheap enough to run in a pipeline.

**Acceptance Criteria:**
- [ ] `--cache` writes review results to a content-addressed cache and reads from it on a subsequent run
- [ ] A re-run with `--cache` on an unchanged model and unchanged review settings returns the identical findings with no model query
- [ ] Changing the model content, or a setting that affects results (such as the confidence threshold), invalidates the cache and triggers a fresh review
- [ ] Without `--cache`, each run performs a fresh review
- [ ] The cache location is documented so CI can persist and clear it

**Context:** LLM output varies run to run, which is poison for a CI gate. The cache makes a review reproducible for an unchanged model: same input produces the same stored findings with zero tokens. Interactive runs may skip the cache for freshness.

**Depends on:** US-001

### US-012: Adversarial self-check to reduce false positives
**Description:** As a model author, I want an optional second pass that re-checks surviving findings and drops the ones it judges false positives, so that I can trade a little time and cost for materially higher precision.

**Acceptance Criteria:**
- [ ] `--verify` runs a second pass over the findings that survived the confidence threshold
- [ ] The self-check returns a keep-or-drop verdict per finding, and dropped findings do not appear in the output
- [ ] Without `--verify`, no self-check pass runs and all threshold-surviving findings are shown
- [ ] The summary indicates how many findings the self-check removed
- [ ] The self-check can only remove findings; it never adds new ones

**Context:** A cheap second pass asks, for each finding, whether it is a real domain problem or a false positive. It is opt-in interactively (where speed matters) and is intended as the default in CI (where stability matters more) — see Open Questions on the default.

**Depends on:** US-006

### US-013: Report token cost and latency of a review
**Description:** As a model author, I want to see how much a review cost in tokens and how long it took, so that I can decide when to run it and budget for it.

**Acceptance Criteria:**
- [ ] After a review, the command reports the token usage consumed by the review
- [ ] When `--verify` is used, its additional cost is included in the reported usage
- [ ] A review served entirely from the `--cache` reports zero token usage
- [ ] Cost reporting appears in text output and does not corrupt the JSON output shape consumed by tooling

**Context:** A high-effort pass over a whole model, plus an optional self-check, costs real tokens and seconds. Surfacing cost lets users weigh interactive vs cached runs. The existing cost tooling already models usage.

**Depends on:** US-001

### US-014: On-demand AI review in the editor
**Description:** As a model author using an editor, I want to trigger an AI review on demand and see findings alongside lint diagnostics, so that I get semantic feedback in my editor without it running on every keystroke.

**Acceptance Criteria:**
- [ ] The editor offers an explicit "Run AI review" action rather than running the review automatically on document change
- [ ] AI findings appear in the editor's diagnostics for the file, tagged as `emod`-sourced, alongside deterministic lint diagnostics
- [ ] AI review does not run on the document-change debounce that drives the regex linter
- [ ] Hovering a finding shows its direction, evidence, and confidence
- [ ] An editor setting maps to the confidence threshold so the user can dial noise without re-running from the command line

**Context:** The editor already renders deterministic diagnostics. Review is slow and costs tokens, so it must be an explicit action, not keystroke-driven. The extra fields (direction, evidence, confidence) enrich hover even though the core diagnostic entry does not carry them.

**Depends on:** US-004, US-005, US-006

### US-015: Suppress findings that overlap deterministic lint
**Description:** As a model author, I want AI findings that land on the same problem a deterministic lint rule already reports to be suppressed, so that I never see duplicate noise from the two checkers.

**Acceptance Criteria:**
- [ ] A finding whose location coincides with a freshly-computed deterministic lint finding for the same problem is dropped before output
- [ ] The reviewer is told which deterministic rules already exist so it is steered away from re-reporting them
- [ ] When a model has both AI and deterministic findings at different locations, both are shown
- [ ] Suppression applies in text, JSON, and editor output consistently

**Context:** Without care the reviewer can re-report a deterministic smell in prose. Steering it away up front plus a post-filter on overlapping locations keeps the two checkers from stepping on each other. Duplicate noise erodes trust faster than a missed finding.

**Depends on:** US-003, US-004

## Non-Goals (Out of Scope)

- **Applying fixes.** This feature reviews and advises with a direction; it never rewrites the file. Turning a finding into an edit belongs to the lint quick-fixes proposal.
- **Authoring DCB tag schemes or `decides_on` queries.** The reviewer may observe a DCB smell semantically, but it does not recommend tags, narrow a `decides_on`, or fix `dcb/*` smells; that belongs to the DCB modeling assistant proposal.
- **Replacing the regex linter.** The deterministic rules stay and remain authoritative for the things they decide. `emod lint` must keep working with no AI configuration.
- **Generating or repairing a model.** This feature reads a model and reports; it does not produce a model and does not use the generate → validate → lint repair loop.
- **Per-rule confidence thresholds.** A single flat threshold ships first; per-id thresholds are a later follow-on.
- **Automatic banning, scoring, or grading of models.** The reviewer surfaces findings; it does not assign a quality score.

## Open Questions

- **Fixed taxonomy vs open rule id.** The seven-id taxonomy plus a catch-all ships first. Assumption: frequent catch-all clusters are promoted to named ids over time once there is data. Proceeding with the catch-all escape hatch.
- **Default for `--verify`.** It improves precision but doubles latency and adds cost. Assumption (proceeding): off interactively, on in CI. Revisit once real false-positive rates are measured.
- **Per-id confidence thresholds.** `language-drift` may warrant a higher bar than `missing-event`. Assumption: a flat threshold first; per-id thresholds deferred until calibration data exists.
- **Whole-model context limit.** Sending the entire model is fine at current example sizes. Open: at what model size does this break, and what is the fallback (e.g. context-by-context review with a final cross-context language-drift pass)?
- **Confidence calibration.** The model's self-rated confidence may not be well-calibrated. Open: do we trust it, or derive confidence externally (e.g. agreement between the main pass and the self-check, or repeated sampling)? Needs measurement before the default threshold is locked.

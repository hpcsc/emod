# AI: Documentation & Onboarding Generation

## Overview

Turn an `.emod` file into human-readable Markdown documentation with a single command: `emod ai docs file.emod -o docs/`. An `.emod` model is a precise, machine-checkable description of an event-driven system, but it is not how a new engineer wants to *learn* one. Faced with five contexts, sixteen slices, and automations crossing context boundaries, a newcomer needs the narrative that is latent in the model: an inbound email arrives, the customer is identified, the message is classified, and the system either auto-replies or opens a case — and while a conversation is live, dunning is paused.

The usual answer is a hand-written README or wiki page. It is correct the day it is written and wrong a month later, because the model is the source of truth and the prose is a copy. emod can do better: the model is already structured and exportable (`emod export -f json` gives every name, field, and inline comment; `emod diagram -f mermaid` renders an embeddable diagram), and the example models already carry authorial intent inline — every slice has a `#` comment stating what it is for. Docs generation here is not hallucinating a description of unknown code; it is expanding the intent the author already wrote, grounded on a structured model whose names can be cross-checked. Generated docs are strictly read-only over an existing, valid model.

## Goals

- Provide an `emod ai docs` command that produces human-readable Markdown documentation from a valid `.emod` file.
- Offer three output styles for three audiences: `--style reference` (catalogue), `--style narrative` (prose walkthrough), and `--style onboarding` (guided cross-context tour plus executive summary).
- Make every described flow, command, event, view, and automation faithful by construction — only entities that exist in the model may be named, and no flow may be invented.
- Honour the per-slice `#` comments as authorial intent: expand and connect them, never contradict them.
- Embed diagrams that emod itself generates (`--with-diagrams`), so the picture is always faithful to the source.
- Keep documentation in sync with the model over time by regenerating in CI and failing the build on drift (`--check`).
- Stay opt-in and zero-impact: existing `validate`/`lint`/`diagram`/`export` commands are unaffected, and the command is absent without configured credentials.

## User Stories

### US-001: Refuse to document an invalid model
**Description:** As a model author, I want `emod ai docs` to validate the model before generating anything so that I never get confident documentation describing flows that do not actually parse.

**Acceptance Criteria:**
- [ ] Running `emod ai docs file.emod` on a model with validation errors prints the errors and exits non-zero without writing any documentation files
- [ ] A model that passes validation but trips linter warnings is still documented, and the lint warnings are surfaced in the command output
- [ ] A clean model proceeds to generation and the output reports the model summary (e.g. number of contexts and slices) before producing docs
- [ ] No documentation is written to the output directory when validation fails

**Context:** Documenting a broken model would produce confident prose about flows that do not exist. The validator runs first as a gate; lint warnings are non-fatal because an un-idiomatic model is still documentable.

### US-002: Generate a faithful per-context reference catalogue
**Description:** As a model author, I want a `--style reference` run that catalogues every context, aggregate, slice, command, event, view, automation, and translation so that I have an accurate, zero-cost map of the whole model.

**Acceptance Criteria:**
- [ ] Running `emod ai docs file.emod --style reference -o docs/` writes one reference page per context plus a top-level index page
- [ ] Each context page lists its aggregates, slices, commands, events (with their fields), views (with their subscriptions), automations, and translations
- [ ] Each slice is labelled with its pattern (command, view, automation, or translation) using emod's own vocabulary
- [ ] Every named entity in the output exists in the model; no entity is invented
- [ ] The reference style produces identical output on repeated runs of the same model (it is deterministic)
- [ ] A model with no AI credentials configured can still produce the reference style

**Context:** The reference catalogue is rendered straight from the exported model with no language-model involvement, so it is faithful by construction and free to run. It also gives the later prose styles a verified skeleton to build on.

**Depends on:** US-001

### US-003: Compute and narrate cross-context edges
**Description:** As a model author, I want each context page to spell out which events it emits that other contexts react to, and which commands arrive from elsewhere, so that I can see the integration surface I would otherwise have to reconstruct from scattered slices.

**Acceptance Criteria:**
- [ ] Each context page includes an inbound/outbound edges section listing the events this context emits that drive automations in other contexts, and the commands or subscriptions arriving from other contexts
- [ ] Each listed edge identifies the source context, the triggering event, and the target context
- [ ] The edges are derived from the model's automations (their target context) and views' subscriptions, not guessed
- [ ] An event that fans out to multiple reacting contexts shows all of its downstream targets
- [ ] A context with no cross-context edges states that explicitly rather than omitting the section

**Context:** Cross-context reactions are the part humans most need and least get from reading slices in isolation. For example, one reply-initiated event can fan out to outbound delivery, a collection hold, and a hold-expiry schedule simultaneously. These edges are computed deterministically from the model.

**Depends on:** US-002

### US-004: Generate narrative prose per slice grounded on its comment
**Description:** As a new engineer, I want a `--style narrative` walkthrough that reads as connected paragraphs per context, aggregate, slice, and flow so that I can learn the system by reading rather than decoding the DSL.

**Acceptance Criteria:**
- [ ] Running `emod ai docs file.emod --style narrative -o docs/` produces per-context chapters whose slices are described in source order as flowing prose
- [ ] Each slice's prose names its triggering condition, the command it issues, and the event(s) it records, all matching the model
- [ ] The prose for a slice reflects and expands that slice's `#` comment as the author's stated intent
- [ ] When a slice's `#` comment and its structure disagree, the prose surfaces the discrepancy rather than inventing a reconciliation
- [ ] The narrative connects each slice to its downstream reactions in other contexts (e.g. that an event also pauses dunning elsewhere)
- [ ] The command reports a token-usage and estimated-cost summary at the end of the run

**Context:** The per-slice `#` comment is the single most valuable grounding signal — a human's one-line summary of each unit of behaviour. The model's job is to expand and connect those summaries, not to guess them. The added value over reading the DSL is connecting a slice to downstream reactions that the source spreads across several other slices.

**Depends on:** US-003

### US-005: Enforce faithfulness so no flow is invented
**Description:** As a model author, I want every entity and flow named in the generated prose to be checked against the model so that I can trust the documentation never describes something that does not exist.

**Acceptance Criteria:**
- [ ] Every context, aggregate, slice, command, event, view, automation, translation, and flow named in a generated section is verified to exist in the model
- [ ] A claimed flow is verified to exist as a real command-to-event edge in the model, not merely that both names exist
- [ ] When a section names an entity that does not exist, that section is regenerated once with the offending names and the correct alternatives fed back
- [ ] If a section still names a non-existent entity after regeneration, the unverifiable references are flagged inline in the output
- [ ] Running with `--strict` makes the command exit non-zero when any section fails the faithfulness check after regeneration
- [ ] The command reports how many referenced entities were resolved against the model (e.g. "22/22 referenced entities resolved")
- [ ] A context page that references none of some of its slices is reported as incomplete

**Context:** The headline risk of doc generation is a plausible-but-wrong flow. The model is never the authority on what exists — the model export is. This is the documentation analogue of the generate-side repair loop: the prose does not have to be perfectly faithful in one shot, it has to converge against the names emod owns.

**Depends on:** US-004

### US-006: Embed emod-generated diagrams with explanatory prose
**Description:** As a reader, I want each context page to embed a diagram that emod generated from the same model, with prose explaining it, so that the picture is always faithful and the explanation interprets a true source.

**Acceptance Criteria:**
- [ ] Running with `--with-diagrams` (on by default) embeds a `mermaid` event-modeling diagram for each context, produced by emod's own diagram output (`emod diagram -f mermaid`)
- [ ] The embedded diagram block is emod's output verbatim, inside a fenced `mermaid` code block, not drawn by the language model
- [ ] Each embedded diagram is followed by prose that explains the swimlane, identifies the trigger, and describes where automations hand off
- [ ] `--diagram-style projected|dcb|auto` is passed through to the diagram output so DCB models render with the appropriate lens
- [ ] Running with diagrams disabled omits the diagram blocks but still produces the surrounding prose
- [ ] `--with-diagrams` adds no language-model cost for the diagrams themselves (emod draws them)

**Context:** The model does not draw. Because the diagram is generated by emod from the same source, it cannot drift from the model, and the model is never in a position to depict a flow that does not exist. The prose only interprets the picture.

**Depends on:** US-004

### US-007: Synthesize an executive summary and onboarding walkthrough
**Description:** As a new engineer, I want a `--style onboarding` run that gives me a system-wide executive summary plus a step-by-step walkthrough following the actual automation chain across contexts so that I understand how the whole thing fits together.

**Acceptance Criteria:**
- [ ] Running `emod ai docs file.emod --style onboarding -o docs/` produces an index page with an executive summary and a system map, plus a dedicated onboarding walkthrough page
- [ ] The onboarding walkthrough follows the real automation chain across contexts as an ordered sequence of steps (e.g. inbound received → identified → classified → reply initiated → delivered / hold applied / hold scheduled → released)
- [ ] The walkthrough's cross-context chain is derived from the model's automations, not inferred
- [ ] Every entity named in the executive summary and walkthrough passes the same faithfulness check as other sections
- [ ] For onboarding, a full-model diagram (in the projected or DCB lens where applicable) is embedded on the walkthrough page
- [ ] `--effort` controls the depth of the synthesis pass, defaulting to high
- [ ] The token-usage and cost summary covers the whole run including the synthesis pass

**Context:** The executive summary and the cross-context walkthrough are the only sections that reason over the whole model at once, and they are the highest-value, hardest-to-get-right parts a new engineer actually needs.

**Depends on:** US-005, US-006

### US-008: Keep documentation in sync with a CI drift gate
**Description:** As a maintainer, I want a `--check` mode that fails the build when committed docs no longer match what the model would produce so that documentation cannot silently drift from the source.

**Acceptance Criteria:**
- [ ] Running `emod ai docs file.emod --check -o docs/` exits non-zero when the committed docs no longer match a fresh generation of the model
- [ ] The drift output names which file(s) are out of date and prints the command to regenerate them
- [ ] Running `--check` against docs that are up to date exits zero and reports no drift
- [ ] The drift signal is stable against language-model non-determinism (the deterministic parts — reference catalogue, embedded diagrams, the set of referenced entities, and/or a recorded export hash — drive the check, not a byte-for-byte prose diff)
- [ ] The documented CI recipe shows how to wire `--check` into a build

**Context:** Drift is fundamentally "did the model change?" Naive byte-diffing of language-model prose would always report drift, so the check is anchored to deterministic parts of the output.

**Depends on:** US-002

### US-009: Choose between a docs tree and a single file
**Description:** As a model author, I want to choose whether the output is a directory tree or a single Markdown file so that I can produce either a browsable docs site or one embeddable page for a PR description.

**Acceptance Criteria:**
- [ ] Running with the default `-o docs/` writes a directory tree: a top-level index page and one page per context under a contexts directory
- [ ] Running with `--single-file` writes one Markdown file containing the same content collapsed into a single document
- [ ] The single-file output preserves the section ordering and embedded diagrams of the tree output
- [ ] Internal references between sections remain navigable in both layouts
- [ ] The chosen layout applies to every style (reference, narrative, onboarding)

**Depends on:** US-007

## Non-Goals (Out of Scope)

- Interactive question answering over a model — that is a separate proposal; this produces static docs, not a chat session.
- Generating BDD scenarios or sample payloads from flows — a separate proposal.
- Generating or editing the model itself; docs generation is strictly read-only over an existing, valid model.
- Inventing a new diagram renderer — diagrams come from emod's existing `emod diagram -f mermaid` path; the prose only wraps them.
- Re-specifying the LLM access layer, provider selection, or cost reporting — those are defined in the shared AI foundation.
- Documenting linter findings inside the docs as a "modeling smells" appendix; lint stays the responsibility of `emod lint`.

## Open Questions

- **What exactly does `--check` diff?** Assumed: anchor drift detection to the deterministic parts (reference catalogue, embedded diagrams, the referenced-entity sets) and/or a recorded hash of the model export, treating prose as advisory rather than byte-comparable. Proceeding on this assumption.
- **Should the reference style ever use the language model?** Assumed: no — the LLM-free catalogue is a genuinely useful zero-cost default, with the model reserved for narrative and onboarding.
- **Prompt granularity: per-slice or per-context?** Assumed: per-slice for reference and narrative (cheaper, concurrent), per-context reasoning for the onboarding synthesis. Proceeding on this assumption.
- **Diagram-per-context vs one big diagram?** Assumed: per-context diagrams on context pages, a full-model diagram on the onboarding page.
- **Field-level prose.** Assumed: the reference style lists every field; narrative prose mentions only the fields relevant to a flow rather than naming all of them.

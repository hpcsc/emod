# AI: Talk to Your Model (Grounded Q&A)

## Overview

A non-trivial event model is a graph, not a document: contexts, slices, and a web of `command → event → automation → command` edges that hop across context boundaries. Answering an ordinary question about that graph — "trace the path from inbound email to hold release", "which contexts touch `CustomerCollectionHold`?", "what breaks if I rename `EmailConversationClassified`?" — means tracing edges by eye across hundreds of lines and several context blocks.

This feature adds a natural-language Q&A layer over a single `.emod` file via the emod CLI. A user asks a question in English and gets a prose answer that names the real contexts, slices, events, and automations involved — grounded on emod's own machine-extracted representation of the model (`emod export -f json` plus a derived edge list) so the answer derives from the actual graph rather than being invented. The answer must cite real model element names, and the model must say "that is not in the model" when an answer is not derivable from it. The feature is read-only, opt-in, and absent without configured AI credentials; it never changes existing commands.

This is the fuzzy, natural-language complement to emod's deterministic queries. Where emod can already compute an exact answer (find-references, the dependency graph, slice listing), that deterministic path stays authoritative; the AI layer handles fuzzy phrasing and multi-element synthesis, and consults deterministic results rather than competing with them.

## Goals

- Let a user ask a one-shot natural-language question about a specific `.emod` file from the CLI and get a grounded prose answer that cites real model element names.
- Let a user hold an interactive REPL session for follow-up questions over the same model, with the model context loaded once and reused across the session.
- Ground every answer on emod's own export of the model so claims derive from the real graph, with answers that name the contexts, slices, events, and automations they rest on.
- Enforce a faithfulness contract: the model answers only from the provided model and explicitly says "that is not in the model" when an answer is not derivable, rather than guessing.
- Cover the query types this shines at: reachability/trace, impact/rename analysis, producer/consumer ("who subscribes to / who triggers X"), cross-context dependency, conditional/branch, and inventory-plus-synthesis questions.
- Offer a machine-readable answer form for tooling, carrying the answer text plus the structured list of model elements it cited.
- Have zero impact on existing commands and stay opt-in: absent and harmless when no AI credentials are configured.

## User Stories

### US-001: Ask a one-shot question grounded on the model export
**Description:** As a model author, I want to ask a natural-language question about a specific `.emod` file from the command line and get a prose answer grounded on the model, so that I can understand my model without reading JSON or tracing edges by hand.

**Acceptance Criteria:**
- [ ] `emod ai ask <file.emod> "<question>"` prints a prose answer derived from the model's exported representation
- [ ] The answer names the real model elements it relies on (contexts, slices, events, commands, automations) using their exact names as they appear in the model
- [ ] Answering a question about an element that does not exist in the model returns "that is not in the model" rather than an invented element
- [ ] Running the command against a file that fails to parse reports the parse error and does not attempt to answer
- [ ] When no AI credentials are configured, the command reports that AI features are unavailable and exits without error to existing workflows
- [ ] The exported representation handed to the model is regenerated from the file on each invocation, so the answer reflects the file's current saved contents

**Context:** emod already produces a machine-extracted representation of the whole model (`emod export -f json`) carrying every context, slice, command, event, view subscription, automation, translation, and the source position of each element. This is the authoritative grounding the answer must derive from. The AI command lives alongside the existing CLI actions and must not alter their behavior.

---

### US-002: Get token cost and usage feedback for an answer
**Description:** As a model author, I want to see how many tokens an answer consumed and its approximate cost, so that I can judge the expense of asking and exploring.

**Acceptance Criteria:**
- [ ] After answering, the command reports input and output token counts and an approximate cost for that question
- [ ] The cost summary is shown by default when running in an interactive terminal
- [ ] A flag controls whether the cost summary is shown or suppressed
- [ ] The reported usage corresponds to the single question just answered, not a cumulative total across unrelated runs

**Context:** The grounding context is the bulk of the input tokens, so each one-shot question pays for it in full. Surfacing usage lets the user understand that cost and prefer the REPL for sustained exploration.

**Depends on:** US-001

---

### US-003: Faithful trace and reachability answers across the graph
**Description:** As a model author, I want to ask the tool to trace a path across the `command → event → automation → command` graph and get the ordered sequence of hops, so that I can follow a flow that crosses several contexts without doing the cross-referencing by eye.

**Acceptance Criteria:**
- [ ] Asking to trace a path between two named points returns an ordered list of hops naming the slice and context at each step
- [ ] Each hop names the real edge it follows (a command-to-event flow, an automation reacting to an event, or an automation targeting another context)
- [ ] A trace that crosses context boundaries names each context it passes through and where the boundary is crossed
- [ ] When the model contains a parallel or branching path relevant to the question, the answer notes the branch rather than presenting a single line as the whole story
- [ ] Asking to trace between points that are not connected in the model returns that no path exists rather than fabricating intermediate hops
- [ ] Every model element named in a trace answer exists in the model

**Context:** Trace questions are multi-hop graph walks. emod's diagram-oriented export already resolves the model into typed edges (flow, automation trigger, automation command, subscription, reads, translation). A flattened form of those edges — source name, target name, edge type, and both contexts — is the most effective grounding for reachability questions, turning a multi-hop reasoning puzzle into a walk over an explicit edge list the answer can be checked against.

**Depends on:** US-001

---

### US-004: Flag invented model element names in an answer
**Description:** As a model author, I want any model element name in an answer that does not actually exist in the model to be flagged, so that I can tell a grounded claim from a confident guess.

**Acceptance Criteria:**
- [ ] After an answer is produced, every model element name it cites is checked against the names actually present in the model export
- [ ] Any cited name absent from the model is surfaced to the user as an unverified citation
- [ ] An answer whose cited names all exist in the model shows no unverified citations
- [ ] The check annotates the answer and never rewrites or suppresses it — the user sees both the answer and the flags
- [ ] An answer containing unverified citations is still shown, accompanied by a visible warning

**Context:** This is a deterministic backstop against hallucination: it turns a silent invented edge or name into a labeled, checkable one. It is a guardrail, not a gate — it verifies that named elements are real, not that the relationships asserted between them are correct.

**Depends on:** US-001

---

### US-005: Answer producer/consumer questions about an element
**Description:** As a model author, I want to ask who subscribes to or who triggers a specific event (or command), and get the complete set of readers and reactions, so that I can understand an element's consumers without searching the file.

**Acceptance Criteria:**
- [ ] Asking "who subscribes to `<event>`" returns the views that read it, naming each view, its slice, and its context
- [ ] Asking "who is triggered by `<event>`" returns the automations that react to it, naming each automation, the command it runs, and the context it targets
- [ ] The answer covers consumers across all contexts, not only the context the element is declared in
- [ ] Asking about an element with no consumers states that nothing subscribes to or is triggered by it
- [ ] Asking about an element name not present in the model returns "that is not in the model"
- [ ] Every consumer named in the answer corresponds to a real subscription or automation in the model

**Context:** These are fuzzy framings of questions with deterministic answers. emod already computes find-references and resolves view subscriptions and automation triggers across contexts; the AI layer phrases the question into the graph and narrates the resulting set rather than inventing it.

**Depends on:** US-003

---

### US-006: Answer cross-context dependency questions
**Description:** As a model author, I want to ask which contexts touch a given aggregate or concept, and get the owning context plus every context that drives it, so that I can see cross-context coupling at a glance.

**Acceptance Criteria:**
- [ ] Asking "which contexts touch `<aggregate>`" names the context that owns the aggregate and every other context that drives it via cross-context automations
- [ ] For each driving context, the answer names the automation(s) and the command they run against the target context
- [ ] The answer distinguishes the owning context from the driving contexts
- [ ] Asking about an aggregate confined to a single context reports that only its owning context touches it
- [ ] Every context and automation named in the answer exists in the model

**Context:** Cross-context dependency is read off automation target contexts plus the aggregate's home context. This is the kind of synthesis that is laborious by hand across several context blocks but mechanical against the resolved edge list.

**Depends on:** US-003

---

### US-007: Continue asking in an interactive REPL session
**Description:** As a model author, I want to start an interactive session over a model and ask a series of follow-up questions, so that I can explore the model conversationally without re-running a command and re-paying for the grounding each time.

**Acceptance Criteria:**
- [ ] Running `emod ai ask <file.emod>` with no question argument starts an interactive prompt; a flag forces the REPL even when a question is supplied
- [ ] On start, the session reports the loaded model's name and a summary count of its contexts and slices
- [ ] The session accepts successive questions and answers each one, with the grounding context loaded once and reused across turns
- [ ] A command within the session exits it cleanly, returning the user to the shell
- [ ] A command within the session reloads the model from the file so a saved edit is picked up without restarting
- [ ] The session reports cumulative token usage across the questions asked
- [ ] All faithfulness behavior from one-shot questions (citations, "not in the model", flagged invented names) applies to every answer in the session

**Context:** The grounding context is the same for every question in a session and is the bulk of the input tokens. Loading it once and reusing it across turns is the cost-efficient surface for exploration, in contrast to one-shot invocations that pay the full context each time.

**Depends on:** US-001, US-004

---

### US-008: Warn when the file changes during a REPL session
**Description:** As a model author, I want the session to tell me when the underlying file has changed since it was loaded, so that I do not get answers from a stale version of the model.

**Acceptance Criteria:**
- [ ] When the file on disk changes after the session loaded it, the session surfaces a warning that its grounding may be stale
- [ ] The user can refresh the session's grounding to the current file contents from within the session
- [ ] After a refresh, subsequent answers reflect the updated file
- [ ] An unchanged file produces no staleness warning

**Context:** Grounding is captured when the session starts; without a staleness signal a user could keep asking against an out-of-date model after editing the file in another window.

**Depends on:** US-007

---

### US-009: Machine-readable answer output for tooling
**Description:** As a tool author or AI agent, I want a machine-readable form of an answer that includes its citations and usage, so that I can script the Q&A or feed it into another tool.

**Acceptance Criteria:**
- [ ] A flag makes `emod ai ask` emit a structured result instead of prose
- [ ] The structured result contains the answer text, the list of cited model elements, the list of unverified citations, and token usage
- [ ] Each cited element carries its name, kind (such as view, automation, event), context, and source line
- [ ] When all cited names are verified, the unverified-citations list is empty
- [ ] The structured output is the only thing written to standard output in this mode, so it can be parsed directly

**Context:** The structured form lets later consumers (such as the editor integration or the diagram viewer) jump from a cited element to its source location. Source positions are carried on every element in the model export, which makes the cited line available.

**Depends on:** US-004

---

### US-010: Impact and rename analysis for a model element
**Description:** As a model author, I want to ask what would break if I renamed or removed a specific element, and get the exact set of places that reference it, so that I can assess the blast radius of a change before making it.

**Acceptance Criteria:**
- [ ] Asking "what breaks if I rename `<element>`" returns every place in the model that references the element, naming each referencing element, its slice, and its context
- [ ] The reference set is computed exactly so the answer neither omits a real reference nor includes one that does not exist
- [ ] References that cross context boundaries are included and their contexts named
- [ ] Asking about an element with no references reports that nothing references it
- [ ] Asking about an element name not present in the model returns "that is not in the model"
- [ ] The answer narrates the computed reference set without adding references that are not in it

**Context:** Rename/impact is the highest-trust case for a deterministic-first posture: the exact reference set is computed first (the same resolution behind find-references and edge resolution), and the model is asked only to phrase that set, so it cannot under- or over-count. A flag can force this exact path, and it is the natural default for impact questions. This deliberately strengthens the producer/consumer and dependency stories by making their underlying sets exact rather than reasoned.

**Depends on:** US-005, US-006

---

### US-011: Decline gracefully when the answer is not in the model
**Description:** As a model author, I want the tool to clearly decline questions whose answer is not represented in the model — even when it can name a related element — so that I never receive an invented fact dressed up as a grounded one.

**Acceptance Criteria:**
- [ ] A question whose answer is not derivable from the model returns a statement that it is not in the model
- [ ] When a related element exists, the decline names that element and explains what the model does and does not represent, rather than guessing the missing detail
- [ ] The tool does not invent a value, name, duration, or relationship to satisfy a question the model cannot answer
- [ ] A question about event-modeling in general (rather than this specific model) is answered only insofar as this model represents it, otherwise declined
- [ ] Declines are consistent between one-shot and REPL modes

**Context:** The naive approach — pasting a model into a generic chat — fails by inventing plausible-but-wrong edges and answering about modeling in general rather than this model. The faithfulness contract is the core value of the feature: answers report what the model contains, not what it might contain or should contain.

**Depends on:** US-001

---

### US-012: Tune answer effort for simple versus multi-hop questions
**Description:** As a model author, I want simple lookups to answer quickly and cheaply while multi-hop trace and impact questions get more reasoning, so that I balance latency and cost against answer difficulty.

**Acceptance Criteria:**
- [ ] An effort level can be set per invocation via a flag, with a sensible default
- [ ] When unset, the effort default can be supplied by configuration
- [ ] Simple lookups complete at the lower default effort without requiring the higher level
- [ ] Multi-hop trace and impact questions can be answered at a higher effort level when requested
- [ ] The chosen effort level does not weaken any faithfulness behavior

**Context:** Ordinary lookups run cheaply by default; trace and impact questions benefit from more reasoning. Effort is observable as a flag and a configurable default; the appropriate default per question type is an open question to be measured rather than fixed here.

**Depends on:** US-003, US-010

## Non-Goals (Out of Scope)

- Editing or extending the model — this is read-only Q&A; conversational editing is a separate proposal.
- Generating prose documentation or onboarding narratives — that is the deliberate output of a separate docs-generation proposal; this is ad-hoc question answering.
- Emitting review findings or modeling smells — opinions about whether the model is good belong to the semantic-reviewer proposal. Q&A answers what the model says, not what it should say.
- Replacing emod's deterministic queries — where emod computes an exact answer (find-references, dependency graph, slice listing), the deterministic path is authoritative; the AI layer handles fuzzy/NL questions and consults deterministic results.
- Multi-file or multi-model sessions — grounding is one model per file; spanning several files is deferred until multi-file models exist.
- Re-specifying the LLM integration, model selection, or credential configuration, which are defined by the shared AI foundation.

## Open Questions

- **Default effort.** Is the lower default with escalation only on detected multi-hop questions right, or should effort be uniform? Assumption: default to the cheaper level and let the user request more; the precise per-question default needs measurement on a question corpus.
- **Auto-routing to the exact path.** A classifier that decides "this is exact-answerable" can itself be wrong. Assumption: start with the exact path as opt-in via a flag, and only auto-route the highest-confidence patterns (literal "who subscribes to X", "what references X").
- **Strictness of the citation check.** Should an answer with unverified citations be downgraded (warned, non-zero exit) or merely annotated? Assumption: annotate and warn, never suppress — the user should see both the answer and the flag.
- **Prompt-cache reliance for REPL cost savings.** The session's reuse of a leading context block assumes it is cache-friendly across turns; how much that saves depends on backend behavior. Assumption: treat reuse as the cost story but measure before promising specific savings.
- **Including the raw source as grounding.** A flag can add the raw `.emod` text (useful for comment/intent questions) on top of the structured export. Assumption: structured export is the default and authoritative grounding; raw source is opt-in.
- **Overlap with the MCP server.** A separate proposal exposes parse/validate/export as tools for a host agent to do its own grounded Q&A. Assumption: keep both — this command is the zero-setup, credential-free path; the MCP server is the bring-your-own-host path.

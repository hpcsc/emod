# AI: Lint Quick-Fixes (LSP Code Actions)

## Overview

emod's linter is good at *spotting* modeling smells but stops at *fixing* them. When it flags `state-obsession` on an event named `EmailConversationUpdated`, it tells the author to "prefer a name that describes a specific business fact" and leaves the rest to them — invent the better name, type it in, then hunt down every place that referenced the old name (`subscribes` lists, `automation`/`translation` references, `flow` lines) before the model validates again.

This feature closes that gap inside the editor. Each lint finding the linter already stands behind gets an AI-backed quick-fix, offered as an ordinary LSP code action on the squiggle the author is already looking at. The model never decides *whether* there is a problem — the linter already did — it only proposes a concrete fix, and emod re-checks the fix before it lands. Rename-style fixes are safe by construction: the set of edit sites comes from emod's own find-references behavior, not the model, so a rename updates the definition and every reference in a single edit. The author stays in their editor; with no AI configured, the editor behaves exactly as it does today and the quick-fixes are simply absent.

This builds on the deterministic findings the existing linter and the semantic model reviewer produce, and assumes the shared AI foundation. It consumes findings; it does not invent new lint rules.

## Goals

- Offer AI quick-fixes through the standard editor code-action flow, so every supported editor (VS Code, Neovim, Zed, Helix, JetBrains) gets them with no per-editor work.
- Anchor every quick-fix to a specific lint finding — never a free-floating suggestion that the linter did not flag.
- Make rename-style fixes safe by construction: a rename rewrites the definition and every reference together, using emod's own find-references behavior to locate edit sites.
- Re-validate and re-lint the proposed result, so a fix that would break the model or fails to clear the original finding is not offered (or is clearly labelled).
- Keep the editor responsive: the squiggle and the lightbulb never wait on the model; the AI call happens only when the author engages a specific fix.
- Degrade cleanly: with no AI configured, every existing editor feature (diagnostics, completion, definition, references, hover, formatting) is unchanged and the AI quick-fixes are absent.

## User Stories

### US-001: Lint findings carry their rule identity to the editor

**Description:** As a model author, I want each lint squiggle in my editor to be tied to the exact rule that flagged it so that a quick-fix can be offered against that specific finding rather than guessed from the surrounding text.

**Acceptance Criteria:**
- [ ] Every lint diagnostic shown in the editor displays the rule name that produced it (e.g. `state-obsession`, `god-view`) as part of the diagnostic.
- [ ] Hovering or inspecting a lint diagnostic surfaces the same rule name that `emod lint --explain <rule>` describes.
- [ ] When the editor asks for quick-fixes at a squiggle, the request is matched to the precise finding at that location, including which named construct (event, command, or view) it points at.
- [ ] Parser and validator diagnostics (non-lint) are unaffected and continue to appear exactly as before.
- [ ] No quick-fixes are offered for a diagnostic whose rule is not in the fixable set.

**Context:** Today the linter emits a rule name internally but it is dropped before reaching the editor, so a fix has nothing to anchor to. This is the enabling prerequisite for every story below: a fix can only be offered on a finding once that finding's rule identity and target construct round-trip to the editor and back. It carries no AI work on its own.

---

### US-002: "Explain this rule" quick-fix appears on every lint finding

**Description:** As a model author, I want an "Explain this rule" action on any lint squiggle so that I can understand why it fired without leaving my editor or running a terminal command.

**Acceptance Criteria:**
- [ ] Opening the quick-fix menu on any lint squiggle shows an "Explain this rule" entry.
- [ ] Selecting it surfaces the same explanatory text that `emod lint --explain <rule>` prints for that finding's rule.
- [ ] The action makes no model call and appears instantly when the lightbulb opens.
- [ ] The action is offered even when AI is not configured.
- [ ] The "Explain this rule" entry is visibly distinct from AI-backed fixes (which are suffixed `(AI)`).

**Context:** This is a non-AI action that proves the code-action round-trip end to end and gives immediate value before any model is wired in. It reuses the rule descriptions emod already ships.

**Depends on:** US-001

---

### US-003: Lightbulb appears instantly on fixable findings without a model call

**Description:** As a model author, I want the quick-fix lightbulb to appear immediately on a fixable lint squiggle so that the editor never feels like it is waiting on AI just to show me that a fix exists.

**Acceptance Criteria:**
- [ ] A lightbulb appears on a lint squiggle whose rule has an AI fix, without any model call having been made.
- [ ] The quick-fix menu lists a rule-specific AI action title (e.g. "Rename event to a business fact (AI)", "Split this god-view (AI)") before any model runs.
- [ ] The model is invoked only when the author engages a specific AI fix (hovers to preview or selects it), not when the lightbulb or menu first appears.
- [ ] When AI is not configured, fixable findings show only the non-AI actions (e.g. "Explain this rule") and no AI entries.
- [ ] The squiggle itself appears at the same speed as it does today, regardless of AI configuration.

**Context:** The cheap "a fix exists" step and the expensive "produce the edit" step are deliberately separated so the editor stays responsive. The menu entry is a lightweight stub; the actual suggested edit is computed later, on engagement.

**Depends on:** US-001

---

### US-004: Rename a state-change event to a business fact

**Description:** As a model author, I want a quick-fix that renames a generically-named event (flagged by `state-obsession`) to a specific business-fact name so that I do not have to invent the name and chase down every reference myself.

**Acceptance Criteria:**
- [ ] On a `state-obsession` finding (an event ending `Updated`/`Changed`/`Modified`), the quick-fix menu offers one or more "Rename to `<BusinessFactName>`" actions suffixed `(AI)`.
- [ ] Each suggested name is drawn from the surrounding slice context (e.g. `EmailConversationUpdated` → `EmailConversationReplyInitiated`).
- [ ] Accepting a rename rewrites the event definition and every reference to it (`subscribes` lists, `automation`/`translation` references, `flow` lines) in a single edit.
- [ ] After applying, the `state-obsession` squiggle is gone and no new parser, validator, or lint problem appears in the file.
- [ ] A suggested name that would collide with an existing name, or would introduce any new problem, is not offered as an action.

**Context:** The new name comes from the model; the set of edit sites comes from emod's find-references behavior, so the rename is safe by construction rather than a text search. The proposal calls these the highest-confidence fixes and ships them first.

**Depends on:** US-003

---

### US-005: Rename property-sourcing and command-in-disguise events

**Description:** As a model author, I want quick-fixes that rename events flagged by `property-sourcing` and `command-in-disguise` to proper business-fact names so that these closely-related naming smells are fixable the same way as `state-obsession`.

**Acceptance Criteria:**
- [ ] On a `property-sourcing` finding (a `<Aggregate><Field>Changed`-style event), the menu offers "Rename to `<BusinessReason>` (AI)" suggestions that describe the business reason for the change.
- [ ] On a `command-in-disguise` finding (an event ending `Initiated`), the menu offers "Rename to `<PastTenseFact>` (AI)" suggestions in happened-form (e.g. `...Initiated` → `...Requested`/`...Started`).
- [ ] Accepting any of these renames rewrites the definition and every reference together in a single edit.
- [ ] After applying, the original finding is gone and no new parser, validator, or lint problem appears.
- [ ] Names that would collide with an existing name or introduce a new problem are not offered.

**Depends on:** US-004

---

### US-006: Rename a past-tense command to imperative form

**Description:** As a model author, I want a quick-fix that renames a command flagged by `command-past-tense` to an imperative name so that commands read as instructions rather than facts.

**Acceptance Criteria:**
- [ ] On a `command-past-tense` finding (a command ending in `ed`), the menu offers "Rename to `<Imperative>` (AI)" suggestions (e.g. `OrderPlaced` → `PlaceOrder`).
- [ ] Accepting a rename rewrites the command definition and every reference to it (`automation`/`translation` references, `flow` lines) in a single edit.
- [ ] After applying, the `command-past-tense` squiggle is gone and no new parser, validator, or lint problem appears.
- [ ] Names that would collide with an existing name or introduce a new problem are not offered.

**Depends on:** US-004

---

### US-007: Rename a view to end in `View`

**Description:** As a model author, I want a quick-fix that renames a view flagged by `view-naming` to a proper `...View` name so that view naming stays consistent and all `reads`/`subscribes` references stay valid.

**Acceptance Criteria:**
- [ ] On a `view-naming` finding (a view not ending in `View`), the menu offers a "Rename to `...View` (AI)" action.
- [ ] Accepting the rename rewrites the view definition and every reference to it (including `reads` references in triggers/translations and any `subscribes` usage) in a single edit.
- [ ] After applying, the `view-naming` squiggle is gone and no new parser, validator, or lint problem appears.
- [ ] A suggested name that would collide with an existing view name is not offered.

**Depends on:** US-004

---

### US-008: Offer multiple ranked rename candidates, cleanest first

**Description:** As a model author, I want a rename quick-fix to offer more than one candidate name when several are reasonable so that I can pick the one that best fits my domain instead of accepting whatever the model returned first.

**Acceptance Criteria:**
- [ ] When a rename finding has multiple reasonable names, the quick-fix menu lists each as its own action (e.g. "Rename to `EmailConversationReplyInitiated` (AI)", "Rename to `EmailConversationReplySent` (AI)").
- [ ] Candidates that re-validate and re-lint clean are ranked above candidates that trade the finding for a different problem.
- [ ] Any candidate that would introduce a parser or validator error is not shown at all.
- [ ] A candidate that resolves the original finding but introduces a different lint warning is ranked lower and labelled so the author knows it is not fully clean.
- [ ] While the same file stays open, the offered candidates for a given finding stay stable rather than changing each time the menu is reopened.

**Context:** A good fix often has more than one defensible form, and the same finding can produce different names on different runs. Ranking by re-check cleanliness, plus a stable menu within a session, keeps the experience predictable. Applies to all rename rules (US-004 through US-007).

**Depends on:** US-004

---

### US-009: Fix a clickbait event by adding fields or inlining its identifier

**Description:** As a model author, I want a quick-fix for an event flagged by `clickbait-event` so that an event carrying only an ID either gains meaningful domain fields or is folded into its parent event.

**Acceptance Criteria:**
- [ ] On a `clickbait-event` finding (an event with a single ID field), the menu offers an "Add domain fields (AI)" action, an "Inline the identifier into the parent event (AI)" action, or both.
- [ ] The "add fields" action inserts the suggested field lines into the event's `fields` block.
- [ ] The "inline" action removes the standalone event and folds its identifier into the relevant parent event, updating references accordingly.
- [ ] After applying either action, the `clickbait-event` squiggle is gone and no new parser, validator, or lint problem appears.
- [ ] An action that would introduce a parser or validator error is not offered.

**Context:** Proposing concrete field names risks inventing data that may not exist, so the safer "inline the identifier" form is offered alongside; see Open Questions on how aggressive field invention should be.

**Depends on:** US-004

---

### US-010: Add tags to an untagged DCB event

**Description:** As a model author working in DCB or mixed mode, I want a quick-fix for an event flagged by `dcb/untagged-event` so that an event with no tags gains tag keys derived from its domain.

**Acceptance Criteria:**
- [ ] On a `dcb/untagged-event` finding (an event with no tags in dcb/mixed mode), the menu offers an "Add tags (AI)" action with suggested tag keys.
- [ ] Accepting the action inserts a `tags` block with the suggested keys into the event.
- [ ] After applying, the `dcb/untagged-event` squiggle is gone.
- [ ] The result does not introduce a `dcb/orphan-tag-key` finding (a tag key never routed on); a candidate that would is not offered or is ranked lower and labelled.
- [ ] An action that would introduce a parser or validator error is not offered.

**Depends on:** US-009

---

### US-011: Narrow an overly broad DCB query

**Description:** As a model author, I want a quick-fix for a decision query flagged by `dcb/query-too-broad` so that a `decides_on` that watches too many events or lacks a predicate gets a narrower, more specific scope.

**Acceptance Criteria:**
- [ ] On a `dcb/query-too-broad` finding (`decides_on` over more than five events, or no `where` clause), the menu offers a "Narrow this query (AI)" action.
- [ ] Accepting the action inserts or replaces the predicate / `where` clause with a narrower one.
- [ ] After applying, the `dcb/query-too-broad` squiggle is gone and no new parser, validator, or lint problem appears.
- [ ] An action that would introduce a parser or validator error is not offered.

**Depends on:** US-010

---

### US-012: Resolve an orphan DCB tag key

**Description:** As a model author, I want a quick-fix for a tag key flagged by `dcb/orphan-tag-key` so that a tag that is never routed on is either put to use in a decision query or removed.

**Acceptance Criteria:**
- [ ] On a `dcb/orphan-tag-key` finding (a tag key never used in any `decides_on`), the menu offers a "Route on this tag (AI)" action and a "Remove this tag" action.
- [ ] The "route on" action adds or edits a `decides_on` predicate that uses the orphan key.
- [ ] The "remove" action deletes the unused tag.
- [ ] After applying either action, the `dcb/orphan-tag-key` squiggle is gone and no new parser, validator, or lint problem appears.
- [ ] An action that would introduce a parser or validator error is not offered.

**Depends on:** US-010

---

### US-013: Split a god-view into focused views

**Description:** As a model author, I want a quick-fix for a view flagged by `god-view` so that a view subscribing to too many events is split into smaller, focused views partitioned by read concern.

**Acceptance Criteria:**
- [ ] On a `god-view` finding (a view subscribing to five or more events), the menu offers a "Split this god-view into focused views (AI)" action.
- [ ] Selecting the action shows a preview of the resulting views before anything is applied, including each new view's name and which events it subscribes to.
- [ ] Accepting the action replaces the single god-view block with the proposed focused-view blocks, each subscribing to fewer than five events, and all subscribed events still resolve.
- [ ] After applying, the `god-view` finding is gone and no new parser, validator, or lint problem appears.
- [ ] A proposed split where any resulting view still has five or more subscriptions, or that introduces any new problem, is ranked last or not offered, so the author is never handed a "fix" that does not fix anything.

**Context:** This is a genuine refactor, not a tweak: the partition can be semantically poor even when syntactically valid. The proposal ships it late, with a clear preview, ranked by re-check cleanliness, and always easy to decline. Never auto-applied.

**Depends on:** US-008

---

### US-014: Specialize a command used across too many flows (left-chair)

**Description:** As a model author, I want a quick-fix for a command flagged by `left-chair` so that a single command reused across many flows can be split into specialized commands wired into each flow.

**Acceptance Criteria:**
- [ ] On a `left-chair` finding (a command appearing in three or more flows), the menu offers a "Specialize this command per use case (AI)" action.
- [ ] Selecting the action shows a preview of the proposed specialized commands and how each flow would be rewired, before anything is applied.
- [ ] Accepting the action creates the specialized commands and updates each affected flow to reference the appropriate one in a single edit.
- [ ] After applying, the `left-chair` finding is gone and no new parser, validator, or lint problem appears.
- [ ] The action is presented conservatively with rationale and is trivial to decline; it is never auto-applied.

**Context:** Like `god-view`, this is a larger structural refactor whose correctness validation cannot fully judge, so it ships last and most cautiously.

**Depends on:** US-013

---

### US-015: Show progress and report cost during a quick-fix

**Description:** As a model author, I want unobtrusive feedback while an AI quick-fix is being computed and a sense of what it cost so that I know the editor is working and stay aware of usage.

**Acceptance Criteria:**
- [ ] While an AI quick-fix is being produced, the editor shows a progress indication rather than appearing frozen.
- [ ] The token usage for a produced fix is surfaced somewhere unobtrusive in the editor.
- [ ] Re-opening the same finding's menu while the file is open does not recompute candidates already produced for it.
- [ ] If the model call fails or is cancelled, the editor returns to its normal state with no edit applied and no spurious error squiggle.
- [ ] The README "Editor Setup" section documents how to enable AI quick-fixes and what behavior to expect with and without AI configured.

**Depends on:** US-008

## Non-Goals (Out of Scope)

- Whole-model semantic review — reviewing a model for smells the linter cannot see is a separate proposal; this feature consumes those findings, it does not produce them.
- Generating a model from prose — a separate proposal.
- Inventing new lint rules — this feature adds *fixes* for the rules emod already has; new rules are out of scope.
- A batch "fix all lint findings" CLI command (e.g. `emod lint --fix`) — the surface here is editor-integrated, per-finding quick-fixes. A CLI batch mode is a possible follow-on (see Open Questions).
- Auto-applying fixes without the author choosing them — every edit is a quick-fix the human explicitly accepts.
- Cross-file rename edit sets — find-references operates on a single document today; spanning multiple `.emod` files is out of scope until multi-file models exist.
- Rewriting occurrences of a name that the model's structure does not link (e.g. a name appearing only inside a comment) — only resolvable command/event/view names are rewritten.

## Open Questions

- **Multiple candidates: many menu entries or one entry that expands?** Listing one entry per ranked candidate gives the cleanest menu but front-loads the AI call into the first quick-fix request; deferring keeps the lightbulb instant but shows a single "AI fix" entry that expands on engagement. US-008 assumes the model call is deferred until the author engages a fix and the candidates are then shown as separate ranked entries — which trade-off editors handle best in practice is open.
- **Single-shot re-check or a bounded repair loop?** These stories assume a one-shot re-validate/re-lint filter on each candidate. Whether a bounded repair loop would meaningfully improve the larger `god-view`/`left-chair` splits, or is over-engineering for a quick-fix, is open.
- **A CLI batch mode?** `emod lint --fix` (apply the top-ranked AI fix for every finding non-interactively) is a natural sibling. Assumed out of scope as a follow-on to preserve the human-in-the-loop; revisit if demand appears.
- **How aggressive should `clickbait-event` field suggestions be?** Proposing concrete domain fields risks inventing data that does not exist. US-009 offers both "add fields" and "inline the identifier"; whether the field-invention form should be dropped in favor of inline-only is open.
- **What counts as "unobtrusive" cost reporting in each editor?** US-015 assumes a lightweight surface; the exact placement (status bar, notification, hover) may differ per editor and is left to refinement.

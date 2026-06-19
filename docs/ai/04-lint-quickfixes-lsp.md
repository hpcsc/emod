# AI-Powered Lint Quick-Fixes (LSP Code Actions)

> Part of the [emod AI proposals](./README.md). Assumes the shared [LLM foundation](./00-llm-foundation.md) (the `llm.Model` port and Bedrock-backed Claude) and does not re-specify it. Builds on the deterministic findings that [03 — semantic model reviewer](./03-semantic-model-reviewer.md) and the existing linter produce.

## Problem

emod's linter is good at *spotting* modeling smells and bad at *fixing* them. Today
`internal/linter/linter.go` flags `state-obsession` on an event named
`EmailConversationUpdated`, tells the author to "prefer a name that describes a
specific business fact", and stops there. The author still has to invent the better
name, type it in, and then hunt down every place that referenced the old name —
`subscribes [...]` lists, `automation`/`translation` references, `flow` lines — or
the model stops validating.

The linter cannot do better on its own because every rule is regex/name/count based.
It knows `OrderUpdated` ends in `Updated`; it has no idea the slice is about shipping,
so it cannot suggest `OrderShipped`. It knows `god-view` because a view subscribes to
five events; it cannot propose *which* three focused views to split it into. The
mechanical knowledge ("rename this, update references") and the domain knowledge
("call it `EmailConversationReplyInitiated`") live in two different places, and only a
human currently bridges them.

That bridge is exactly what an LLM is good at, and this is the **highest-trust** place
to put one in emod: every suggestion is anchored to a deterministic finding the linter
already stands behind. The model never decides *whether* there is a problem — the
linter already did. The model only proposes a concrete fix for a problem emod is
certain exists, and emod re-checks the fix before and after it lands. The author stays
in the editor; the fix arrives as an ordinary LSP quick-fix on the squiggle they were
already looking at.

## Goals

- Offer AI quick-fixes through the standard LSP `textDocument/codeAction` flow, so
  every supported editor (VS Code, Neovim, Zed, Helix, JetBrains — see the README
  "Editor Setup") gets them for free.
- Anchor every code action to a specific `diagnostic.Entry` emitted by
  `internal/linter` (or the 03 reviewer) — never a free-floating suggestion.
- Make rename-style fixes **safe by construction**: reuse the existing
  `GetReferences` machinery (`internal/lsp/references.go`) so a rename updates the
  definition *and* every reference in one `WorkspaceEdit`.
- Re-run the deterministic pipeline (`parser → validator → linter`) on the proposed
  result and only offer fixes that are clean, or clearly label ones that aren't.
- Keep the editor responsive: actions are produced lazily and asynchronously, with
  progress, never blocking the squiggle from appearing.
- Use the cheap model (`anthropic.claude-haiku-4-5`) and cache the per-document
  context, because these are small, frequent, low-stakes calls.
- Degrade cleanly: with no AI configured, the LSP behaves exactly as it does today —
  the quick-fixes are simply absent. Diagnostics, completion, definition, references,
  hover, and formatting are untouched.

## Non-Goals

- Whole-model semantic review. Reviewing a model for smells the linter *can't* see is
  [03](./03-semantic-model-reviewer.md); this proposal consumes those findings, it
  does not produce them.
- Model generation from prose. That is [01](./01-nl-to-model-generation.md).
- Inventing new lint rules. We add *fixes* for the rules already in
  `internal/linter/descriptions.go`; new rules are out of scope.
- A batch "fix all lint findings" CLI command. The surface here is editor-integrated,
  per-finding code actions. (A CLI batch mode is a possible follow-on, noted in Open
  Questions.)
- Auto-applying fixes without the author choosing them. Every edit is a quick-fix the
  human explicitly accepts.

## How It Works

### The codeAction flow

The current server (`internal/lsp/server.go`) advertises capabilities in
`handleInitialize` and dispatches methods in `dispatch`. Two pieces are added:

1. **Advertise the capability.** `ServerCapabilities` (in
   `internal/lsp/protocol.go`) gains a `CodeActionProvider`. To support lazy
   resolution it is advertised as `CodeActionOptions{ ResolveProvider: true }`
   rather than a bare `true`, so the heavy LLM work is deferred (see *Latency*
   below).

2. **Carry the rule name on diagnostics.** This is the load-bearing prerequisite.
   Today `ConvertDiagnostics` (`internal/lsp/diagnostics.go`) drops
   `diagnostic.Entry.RuleName` on the floor — the LSP `Diagnostic` struct has no
   `Code`/`Data` field. To anchor a code action to a finding, the rule name and the
   target identifier must round-trip to the client and back. `Diagnostic` gains:

   ```go
   type Diagnostic struct {
       Range    Range              `json:"range"`
       Severity DiagnosticSeverity `json:"severity"`
       Message  string             `json:"message"`
       Source   string             `json:"source"`
       Code     string             `json:"code,omitempty"` // diagnostic.Entry.RuleName
       Data     json.RawMessage    `json:"data,omitempty"` // anchor: target name/kind, positions
   }
   ```

   `Code` is the rule name (`state-obsession`, `god-view`, …); `Data` is an opaque
   anchor emod populates and the client echoes back unchanged on
   `textDocument/codeAction`.

The request/response cycle:

```
textDocument/publishDiagnostics   server → client   squiggle + Code=state-obsession + Data
textDocument/codeAction           client → server   "what fixes apply at this range?"
  → server returns lightweight CodeAction stubs (title + kind, no edit yet)   [no LLM call]
codeAction/resolve                client → server   "user is hovering / picked this one"
  → server calls llm.Model (Haiku), builds the WorkspaceEdit                  [LLM call here]
workspace/applyEdit               (client applies the returned edit)
```

The split matters: the *stub* (step `codeAction`) is cheap and synchronous — it only
needs to know the rule fired and an AI fix exists, so the lightbulb appears instantly.
The *expensive* LLM call happens in `codeAction/resolve`, only for the action the user
actually engages with.

A new handler `handleCodeAction` is added to `dispatch` for `textDocument/codeAction`,
and `handleCodeActionResolve` for `codeAction/resolve`, mirroring the existing
handlers' shape (unmarshal params → look up document via `s.documents.GetContent` →
marshal result via `writeMessage`).

### Producing the stubs (cheap, no LLM)

`handleCodeAction` receives a range and the diagnostics the client saw there. For each
diagnostic whose `Source == "emod"` and whose `Code` matches a fixable rule (the table
below), it emits a `CodeAction` stub:

```go
type CodeAction struct {
    Title       string         `json:"title"`
    Kind        string         `json:"kind"`        // "quickfix"
    Diagnostics []Diagnostic   `json:"diagnostics"` // the finding this fixes
    Data        json.RawMessage `json:"data,omitempty"` // echoed to resolve
    Edit        *WorkspaceEdit `json:"edit,omitempty"`  // nil until resolved
}
```

Titles are human-readable and rule-specific, e.g. *"Rename event to a business fact
(AI)"*, *"Split this god-view (AI)"*. No model is called here. If AI is not configured,
`handleCodeAction` returns the empty list and the flow ends — identical to having no
code-action provider.

### Producing the edit (the LLM call, in resolve)

`codeAction/resolve` is where `llm.Model` is invoked. The server:

1. Re-derives the finding from `Data` (rule name + the AST node it points at) against
   the current document text.
2. Assembles a prompt: the rule's own description from `linter.RuleDescription` (the
   same text `emod lint --explain <rule>` prints — free, authoritative grounding), the
   offending construct, and tightly scoped surrounding context (its slice, sibling
   event/command/view names, the aggregate name). The whole model is *not* sent — the
   anchor lets us send only what the fix needs, which keeps tokens (and cost) low.
3. Calls `m.Generate` with `Effort: "low"` and a **JSON Schema** (`GenerateRequest.Schema`)
   so the response is structured, not prose — typically a ranked list of candidate
   fixes rather than free text.
4. Turns the chosen candidate into a `WorkspaceEdit` and validates it (below).

Because a good fix often has more than one reasonable form (`OrderShipped` vs
`OrderDispatched`), the schema returns up to N ranked candidates. The resolve handler
can either return the top-ranked one as the resolved edit, or — preferred — the stub
phase emits *one stub per candidate* so the editor's quick-fix menu shows all of them
("Rename to `EmailConversationReplyInitiated`", "Rename to `EmailConversationReplySent`"…).
The candidate generation still happens once, in resolve, and is memoized per finding.

### Safety: validate/lint + find-references

Three layers keep a fix from breaking the model:

1. **References come from emod, not the LLM.** For any rename, the model proposes only
   the *new name*. The set of edit sites is computed deterministically by reusing
   `GetReferences` (`internal/lsp/references.go`), which already resolves a
   command/event/view name to its definition plus every reference across
   `subscribes`, `automation`, `translation`, `trigger reads`, and `flow`. The
   `WorkspaceEdit` is the union of `{definition} ∪ references`, each rewritten to the
   new name. The model is never trusted to find call sites — emod's own name
   resolution does.

2. **Re-check before offering / after applying.** The proposed document text is run
   through the existing pipeline used by `pushDiagnostics` in `server.go`
   (`lexer.Scan → parser.New().Parse → validator.Validate → linter.Lint`). A fix that
   introduces a parser/validator error is dropped, not offered. A fix that resolves
   the original finding *and* introduces no new lint warning is ranked highest; one
   that trades the finding for a different lint warning is ranked lower and labelled.
   This is the same "deterministic oracle" idea as the foundation's repair loop, used
   here as a single-shot filter rather than a multi-attempt loop.

3. **Name collisions are caught by the oracle.** If the model suggests a new name that
   already exists, the re-parse/validate surfaces the duplicate and the candidate is
   discarded before it reaches the user.

### Per-rule fix shapes

| Rule (`descriptions.go`) | Finding anchor | Fix shape | Safety |
|---|---|---|---|
| `state-obsession` | event ending `Updated`/`Changed`/`Modified` | rename to a business-fact name from slice context | rename via `GetReferences`; re-lint |
| `property-sourcing` | `<Aggregate><Field>Changed` event | rename to the business reason for the change | rename via `GetReferences`; re-lint |
| `command-in-disguise` | event ending `Initiated` | rename to a past-tense fact (often the `...Initiated` → `...Requested`/`...Started` happened-form) | rename via `GetReferences` |
| `command-past-tense` | command ending `ed` | rename to imperative (`OrderPlaced` → `PlaceOrder`) | rename via `GetReferences` |
| `view-naming` | view not ending `View` | append/rewrite to `...View` | rename via `GetReferences` (updates `reads`, `subscribes`) |
| `clickbait-event` | event with one ID field | add domain fields (suggested `fields {}` lines) **or** inline the ID into the parent event | insert edit in the `fields` block; re-validate |
| `god-view` | view subscribing to ≥5 events | split into focused views, partitioning the `subscribes` list by read concern | multi-block `WorkspaceEdit`; re-validate the new views |
| `left-chair` | command in ≥3 flows | propose specialized commands per use case and rewire each flow | larger refactor (see Risks); offered cautiously |
| `dcb/untagged-event` | event with no tags in dcb/mixed | propose `tags {}` keys from the event's domain | insert `tags` block; re-lint for `dcb/orphan-tag-key` |
| `dcb/query-too-broad` | `decides_on` >5 events or no `where` | propose a narrower predicate / `where` clause | insert/replace predicate; re-validate |
| `dcb/orphan-tag-key` | tag key never routed on | propose a `decides_on` predicate that uses the key, or drop the tag | edit predicate or remove tag |

The first five are simple, high-confidence renames and ship first. `clickbait-event`,
the DCB inserts, and `god-view`/`left-chair` are progressively larger and land in later
phases.

## Interface

The author edits an `.emod` file. The linter has already drawn a warning squiggle on
the event name (this exists today via `publishDiagnostics`). Now a lightbulb appears on
the squiggle. Opening it shows AI-backed quick-fixes alongside any non-AI ones.

The actions are titled per rule and end with `(AI)` so it is always clear which fixes
came from the model versus deterministic edits:

```
EmailConversationUpdated
  ⚠ [state-obsession] event "EmailConversationUpdated" uses a generic
    state-change suffix; prefer a name that describes a specific business fact
  💡 Quick Fixes:
     • Rename to "EmailConversationReplyInitiated"   (AI)
     • Rename to "EmailConversationReplySent"        (AI)
     • Rename to "EmailConversationClosed"           (AI)
     • Explain this rule                              (opens lint --explain text)
```

Before — what the linter flags:

```emod
slice "Compose auto-reply" {
  command EmailConversationInitiateReply { ... }

  event EmailConversationUpdated {        # ⚠ state-obsession
    fields { customerId UUID required }
  }

  flow {
    command -> event: EmailConversationInitiateReply -> EmailConversationUpdated
  }
}

view EmailDecisionsTopicView {
  subscribes [EmailConversationUpdated, ...]   # reference elsewhere in the file
}
```

After — accepting *"Rename to `EmailConversationReplyInitiated`"*. The single
`WorkspaceEdit` rewrites the definition **and** both references (the `flow` line and the
`subscribes` list), because the edit sites came from `GetReferences`, not from a
text search:

```emod
event EmailConversationReplyInitiated { ... }   # definition
...
flow {
  command -> event: EmailConversationInitiateReply -> EmailConversationReplyInitiated
}
...
subscribes [EmailConversationReplyInitiated, ...]
```

The result re-validates and re-lints clean before the action was ever offered, so the
squiggle disappears on apply with no follow-on errors.

## Worked Example

Using `examples/inbound_customer_comms_agentic_reply.emod`, the
`EmailDecisionsTopicView` (lines 250–263) subscribes to four events. Suppose a fifth
subscription is added — the linter's `checkGodView` (`linter.go`) fires `god-view`
(`len(view.Subscribes) >= 5`) as an **error** on the view name.

1. **Finding.** `pushDiagnostics` emits
   `Diagnostic{ Code: "god-view", Range: <view name>, Data: {kind:"view", name:"EmailDecisionsTopicView", subscribes:[...]} }`.

2. **codeAction (stub, no LLM).** Editor requests actions at the view name. Server
   matches `Code == "god-view"`, returns one stub: *"Split this god-view into focused
   views (AI)"*.

3. **resolve (LLM call).** User selects it. The resolve handler builds a prompt from:
   - `RuleDescription("god-view")` — *"A view that subscribes to five or more events is
     likely trying to do too much. Split into smaller, more focused views…"* (the same
     text `emod lint --explain god-view` prints);
   - the view's `fields` and `subscribes` list;
   - the names/fields of the subscribed events for read-concern grouping.

   It calls Haiku (`Effort: "low"`) with a JSON Schema describing
   `{ views: [{ name, subscribes[], rationale }] }`. The model partitions the five
   events into, say, an `EmailDecisionPublicationView` (the publication-decision events)
   and an `EmailEscalationDecisionView` (the case/escalation events).

4. **Build + validate.** The server renders the two new `view` blocks, produces a
   `WorkspaceEdit` replacing the one god-view block, and runs the full pipeline on the
   resulting document. Each new view has <5 subscriptions, all subscribed events still
   resolve, no parser/validator error appears — so `god-view` clears with no new
   findings. The action is offered (or, since resolve already validated, applied).

5. **Apply.** The editor applies the edit. Re-lint shows the `god-view` error gone and
   no new warnings.

If instead the model had returned a split where one view still had five subscriptions,
step 4's re-lint would still see `god-view`; that candidate is ranked last or dropped,
and the user is not handed a "fix" that doesn't fix anything.

## Implementation Plan

**Phase 1 — Plumbing the anchor (S).** Extend the LSP `Diagnostic` with `Code` and
`Data`; populate `Code` from `diagnostic.Entry.RuleName` and `Data` with the target
name/kind in `ConvertDiagnostics` (`internal/lsp/diagnostics.go`). Add
`CodeActionProvider` to `ServerCapabilities` and the `textDocument/codeAction` /
`codeAction/resolve` cases to `dispatch` (`server.go`). Return stubs for fixable rules
with **no LLM** yet (and a non-AI "Explain this rule" action wired to
`RuleDescription`). This is shippable on its own and proves the round-trip.

**Phase 2 — Rename fixes (M).** Wire `llm.Model` (cheap model, behind the foundation's
opt-in config) into `codeAction/resolve` for the five rename rules: `state-obsession`,
`property-sourcing`, `command-in-disguise`, `command-past-tense`, `view-naming`.
Compute edit sites with `GetReferences`; build the multi-site `WorkspaceEdit`;
re-validate/re-lint before offering. JSON-schema'd, ranked candidates → multiple stubs.

**Phase 3 — Insert/augment fixes (M).** `clickbait-event` (add `fields` / inline) and
the DCB inserts (`dcb/untagged-event`, `dcb/query-too-broad`, `dcb/orphan-tag-key`).
These edit a single block, so safety is mostly re-validate.

**Phase 4 — Structural splits (L).** `god-view` and `left-chair`. These create/rename
multiple blocks and rewire flows; they need the most validation care and the most
conservative UX (clear preview, ranked, easy to decline).

**Phase 5 — Polish (S).** Progress notifications during resolve; per-finding memoization
of candidates; surface `Response.Usage` cost (the foundation reports it) somewhere
unobtrusive; documentation in the README "Editor Setup" section.

Testing follows the repo conventions throughout: each handler is unit-tested against a
mock `llm.Model` returning canned candidates (one umbrella `Test{Type}`, `t.Run`
groups per operation, behavior-named scenarios, `testify/require`, fresh fixtures), so
the codeAction/resolve round-trip, the `GetReferences`-driven rename, and the
re-validate filter are all covered without a network.

## Risks & Mitigations

- **A "fix" that breaks the model.** Mitigated by the re-parse/validate/lint filter:
  no candidate is offered (or, with resolve, applied) unless the resulting document is
  clean. Renames never let the model find call sites — `GetReferences` does.
- **Rename misses a reference emod doesn't track.** `GetReferences` covers the
  reference kinds the parser models. If a name appears somewhere the AST doesn't link
  (e.g. inside a comment), it won't be rewritten. Mitigation: scope rename fixes to
  the resolvable name kinds (command/event/view) the references engine already
  supports, and rely on the post-apply re-validate to catch any genuine dangling
  reference the model surfaces as an `orphan-*` finding.
- **Latency makes the editor feel slow.** Mitigated by the stub/resolve split (no LLM
  until the user engages a fix), `ResolveProvider: true`, the cheap model at
  `Effort: "low"`, and `$/progress` during resolve. The squiggle and the lightbulb
  never wait on the model.
- **Cost from frequent calls.** Haiku, low effort, scoped context (anchor → send only
  the construct + its slice, not the whole model), and per-finding memoization keep
  per-fix cost tiny. Usage is reported via `Response.Usage`.
- **`left-chair` / `god-view` are genuine refactors, not tweaks.** A split can be
  wrong in ways validation won't catch (the *partition* may be semantically poor even
  if syntactically valid). Mitigation: ship these last, present them as suggestions
  with rationale, rank by re-lint cleanliness, and make declining trivial. Never
  auto-apply.
- **Non-determinism across invocations.** The same finding may yield different names on
  different days. Acceptable for a suggestion surface; mitigated by ranking and by the
  user choosing. Memoize within a session so the menu is stable while the file is open.
- **Hard dependency creep.** AI must stay opt-in. The Phase-1 plumbing (Code/Data,
  stubs, "Explain this rule") works with **zero** LLM dependency; the model is only
  touched in resolve, only when configured. With no credentials, the server offers no
  AI actions and every existing feature is byte-for-byte unchanged.

## Open Questions

- **Multiple candidates: many stubs or one stub that expands?** Emitting one stub per
  ranked candidate gives the cleanest editor menu but front-loads the LLM call into the
  first `codeAction`; deferring to resolve keeps the lightbulb instant but shows a
  single "AI fix" entry. Which trade-off do editors handle best in practice?
- **Should the re-check use the full repair loop or a single shot?** This proposal uses
  a one-shot validate/lint filter. Would a bounded repair loop (foundation's
  `GenerateAndRepair`) meaningfully improve `god-view`/`left-chair` splits, or is that
  over-engineering a quick-fix?
- **A CLI batch mode?** `emod lint --fix` (apply the top-ranked AI fix for every
  finding, non-interactively) is a natural sibling that reuses all of this. In scope as
  a follow-on, or kept strictly editor-only to preserve the human-in-the-loop?
- **Workspace-wide references.** `GetReferences` operates on a single document. If a
  model ever spans multiple `.emod` files, a rename's edit set must cross files. Out of
  scope until multi-file models exist, but the `WorkspaceEdit` shape already allows it.
- **How aggressive should `clickbait-event` field suggestions be?** Proposing concrete
  domain fields risks inventing data that doesn't exist. Should that fix instead only
  ever offer the "inline the identifier" form, leaving field invention to the human?

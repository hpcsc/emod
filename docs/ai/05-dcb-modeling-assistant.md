# DCB Modeling Assistant

> Part of the [emod AI proposals](./README.md). Assumes the shared [LLM foundation](./00-llm-foundation.md) (the `llm.Model` port, Bedrock-backed Claude, and the repair loop) and does not re-specify it. Builds on the DCB DSL defined in [`../dcb-proposal.md`](../dcb-proposal.md).

## Problem

Dynamic Consistency Boundary modeling is the part of emod where the conceptual
load is highest. Aggregate-style modeling hands the author a default boundary:
the aggregate owns its events, and `stream "X-{id}"` implies the query. DCB takes
that scaffolding away. The author must now decide, per event, *which entities and
relationships it keys on* (its `tags`), and, per command, *exactly which event
types and tag predicate define its consistency boundary* (its `decides_on`). Get
the tags too coarse and every command reads the world; get them wrong and the
boundary protects the wrong invariant.

emod already detects the failure modes after the fact. The linter
(`internal/linter/descriptions.go`) ships DCB-specific rules:

- `dcb/untagged-event` — every event in a `dcb`/`mixed` context must declare at
  least one tag.
- `dcb/query-too-broad` — a `decides_on` that references more than 5 event types,
  or has no `where` clause, reads too much and over-couples the command.
- `dcb/single-tag-everywhere` — every command keys on the same single tag key, so
  DCB is just aggregates with extra ceremony.
- `dcb/orphan-tag-key` — a tag key is declared on events but never referenced by
  any command's `decides_on`, so it routes nothing.

Plus the mode guards `dcb-in-aggregate-mode` and `aggregate-in-dcb-mode`.

The linter is a good judge and a poor teacher. It tells an author that
`OrderPlaced` is untagged or that `SendNotification` reads too broadly, but it
cannot propose *which* tag key to add, *which* field it should reference, or
*which* narrower predicate would still protect the invariant. Those answers
require reading the event's fields, the command's intent, and the surrounding
domain — exactly the reasoning an LLM is suited to, and exactly the reasoning a
regex rule cannot do. The gap is widest at the moment of authoring a new DCB
context or converting an existing aggregate context to DCB, where there is no
prior structure to copy from.

## Goals

- Turn each `dcb/*` finding into a concrete, applicable suggestion: a `tags` block
  for an untagged event, a narrower `decides_on` for a too-broad query, a routing
  command (or a removal) for an orphan tag key, an additional tag dimension for
  single-tag-everywhere.
- Assist whole-context **aggregate → DCB conversion**: take an `aggregate`-mode
  context and emit a `dcb`-mode rewrite with tagged events and tag-scoped
  decisions, kept valid by the repair loop.
- Ground every suggestion in DCB semantics (the essence of
  [`../dcb-proposal.md`](../dcb-proposal.md)) plus the actual model, so suggestions
  use real field names and respect declared tag keys.
- Use `emod validate` + `emod lint` (especially the `dcb/*` rules) as the oracle:
  a suggestion is only offered if applying it parses, validates, and does not
  reintroduce or worsen DCB findings.
- Keep suggestions **advisory**. DCB carries genuine domain ambiguity; the author
  reviews and accepts, never silent auto-apply.

## Non-Goals

- General modeling-smell review (past-tense events, god-views, state obsession).
  That is proposal [03](./03-semantic-model-reviewer.md).
- The generic LSP code-action plumbing that surfaces any deterministic finding as
  a quick-fix. That is proposal [04](./04-lint-quickfixes-lsp.md). The DCB fixes
  here *can* be exposed through that channel (see [Interface](#interface)), but the
  reasoning, not the editor wiring, is the subject of this proposal.
- Changing DCB semantics, the grammar, or the lint rules. This consumes the DSL
  and rules exactly as [`../dcb-proposal.md`](../dcb-proposal.md) and
  `internal/linter` define them.
- Generating runtime DCB infrastructure (event store, append-condition checks).
  emod describes the model; runtime is out of scope, as in the DCB proposal.

## How It Works

### Grounding the model on DCB semantics

A general LLM knows "consistency boundary" loosely; it does not know emod's exact
DCB surface. Every call is grounded on three things assembled into the `System`
field of `llm.GenerateRequest`:

1. **DCB essence.** A distilled ~1-page brief extracted from
   [`../dcb-proposal.md`](../dcb-proposal.md): events carry one or more tags
   (`key: fieldRef`); a command's `decides_on` query *is* its consistency
   boundary; the boundary should be the narrowest set of event types and tag
   predicate that still protects the invariant the command guards; a tag key
   exists to express a distinct routing concern, not to decorate events. This
   brief is checked into the repo (e.g. `internal/ai/dcb/brief.md`, embedded with
   `go:embed`) so it is versioned alongside the rules it explains.
2. **The relevant lint-rule descriptions**, pulled verbatim from
   `linter.RuleDescription` for whichever `dcb/*` rules are in play. The model is
   told the exact rule it is fixing in the rule's own words, so its output targets
   the oracle's actual checks.
3. **The model context.** The JSON export of the model (`emod export --format
   json`), or just the enclosing context, so the model sees real event field
   names, declared tag keys, command fields, and the `flow` edges.

The grammar facts the model must respect — `tags { key: fieldRef }`, `decides_on
{ events [...] where tag(key = value) ... }`, `and`/`or`/`not`, `mode dcb` — are
stated once in the brief with a worked snippet copied from
`examples/dcb_model.emod`, the canonical reference.

### Structured output

Suggestions are returned as structured data via the `Schema` field on
`GenerateRequest` (the strict structured-output path from the foundation), never
as free text to be regex-parsed. The schema is a discriminated union over the
suggestion kinds:

```go
package dcb

type SuggestionKind string

const (
	KindEventTags     SuggestionKind = "event_tags"     // fixes dcb/untagged-event
	KindNarrowQuery   SuggestionKind = "narrow_query"   // fixes dcb/query-too-broad
	KindOrphanTagKey  SuggestionKind = "orphan_tag_key" // fixes dcb/orphan-tag-key
	KindExtraTagKey   SuggestionKind = "extra_tag_key"  // fixes dcb/single-tag-everywhere
)

type TagSuggestion struct {
	Key      string `json:"key"`       // e.g. "entity"
	FieldRef string `json:"field_ref"` // a field declared on the target, e.g. "customerId"
}

type Suggestion struct {
	Kind       SuggestionKind  `json:"kind"`
	Target     string          `json:"target"`     // event or command name
	Rule       string          `json:"rule"`       // the dcb/* rule this addresses
	Tags       []TagSuggestion `json:"tags,omitempty"`        // event_tags / extra_tag_key
	Events     []string        `json:"events,omitempty"`      // narrow_query: event types to keep
	Predicate  string          `json:"predicate,omitempty"`   // narrow_query: where clause, e.g. "tag(entity = customerId)"
	Removal    bool            `json:"removal,omitempty"`     // orphan_tag_key: true => recommend removing the key
	RoutedFrom string          `json:"routed_from,omitempty"` // orphan_tag_key: command proposed to route on the key
	Rationale  string          `json:"rationale"`             // one or two sentences, shown to the author
}
```

The model returns *intent* (key + field, kept events + predicate), not raw
`.emod` text. emod renders the intent into syntactically correct DSL with its own
formatter, so the model cannot produce malformed braces or stray tokens — the
same reason structured output is preferred over string generation throughout the
AI proposals. `field_ref` is constrained: the renderer rejects any `field_ref`
that is not a field actually declared on the target, before the suggestion is ever
shown.

### The repair loop with the `dcb/*` rules as oracle

For per-finding suggestions (tags, narrow queries) the model usually answers in
one shot, and emod just verifies: render the suggestion into the file, run
`pipeline.Check`, and discard the suggestion if it fails to parse/validate or
reintroduces a `dcb/*` finding on the same target. A narrowed `decides_on`, for
instance, is only surfaced if it clears `dcb/query-too-broad` *and* still
validates (every `tag(k = …)` resolves to a declared key on a referenced event,
per the DCB proposal's validator rules).

For whole-context conversion the full `GenerateAndRepair` loop from the foundation
does the work. The prompt is "rewrite this `aggregate`-mode context as `mode dcb`",
and `pipeline.Check` is the oracle on every attempt:

```
generate dcb rewrite
   │
   ▼
parse → validate → lint  ──clean──►  present diff to author
   │ (diagnostics: parse errors, unresolved tag keys,
   │  dcb/untagged-event, dcb/query-too-broad, …)
   ▼
append .emod + diagnostics to the request, retry (≤ maxAttempts)
```

The conversion is *done* when the rewrite parses, validates, and is free of
`dcb/*` warnings — not merely when it parses. Because the linter already encodes
"untagged event", "query too broad", "single tag everywhere", and "orphan tag
key", the loop is self-correcting against precisely the anti-patterns a
hand-written conversion would fall into. The loop is bounded by `maxAttempts`; if
it does not converge it returns the best attempt plus the residual diagnostics so
the author can finish by hand, rather than failing silently.

### Why suggestions stay advisory

DCB boundaries encode invariants only the domain expert truly knows. `OrderPlaced`
could legitimately key on `entity: customerId`, on `category: orderType`, or both,
depending on what the business needs to keep consistent. The assistant proposes
the most defensible reading from the fields and flow, attaches a one-line
rationale, and lets the author accept, edit, or reject. Acceptance writes the
change; nothing is applied silently. This mirrors the foundation's stance that the
deterministic oracle — not the model — is the authority: the model proposes, the
pipeline disposes, and the human decides.

## Interface

A new `ai` command group, with a `dcb` subgroup. AI features are opt-in and inert
without Bedrock credentials configured, exactly as the foundation specifies; the
existing `validate`/`lint`/`export`/`diagram`/`lsp` paths gain no LLM dependency.

```
emod ai dcb suggest <file>            # suggest fixes for all dcb/* findings in the file
emod ai dcb suggest <file> --rule dcb/untagged-event   # only one rule's findings
emod ai dcb suggest <file> --target OrderPlaced        # only one event/command
emod ai dcb convert <context> <file>  # rewrite an aggregate-mode context as mode dcb
```

Flags consistent with the rest of the CLI (`internal/cli/app.go`):

- `--format text|json` — `json` emits the `[]Suggestion` for editors and the LSP
  code-action provider of proposal [04](./04-lint-quickfixes-lsp.md) to consume.
- `--apply` — write accepted suggestions back to the file (default is to print a
  diff and leave the file untouched).
- `--effort low|medium|high` — passed through to the model; suggestion passes
  default to `medium`, conversion to `high`.

`emod ai dcb suggest` runs the existing linter first, keeps only the `dcb/*`
findings, and asks the model for one suggestion per finding. Output, text form:

```
$ emod ai dcb suggest order.emod
order.emod:18:3  dcb/untagged-event  event OrderPlaced
  suggested tags:
    entity   : customerId      # routes per customer; the order's owning party
    category : orderType       # lets order-type policies form their own boundary
  rationale: OrderPlaced carries customerId and orderType; both are entities other
             commands will decide against, so both warrant a tag key.

order.emod:96:3  dcb/query-too-broad  command SendNotification
  current : events [PaymentAuthorized, OrderShipped]  (no where clause)
  suggested:
    events [PaymentAuthorized, OrderShipped]
    where tag(entity = customerId)
  rationale: notifications are per-customer; scoping to entity bounds the read to
             one customer's events instead of every authorization and shipment.

2 suggestions. Re-run with --apply to write them, or --format json for tooling.
```

Conversion prints a unified diff of the context, the residual diagnostics (ideally
none), and the token usage from `Response.Usage`:

```
$ emod ai dcb convert Fulfillment legacy.emod
rewriting context "Fulfillment" from mode aggregate to mode dcb …
  attempt 1: 3 diagnostics (dcb/untagged-event ×2, dcb/query-too-broad ×1)
  attempt 2: clean
--- legacy.emod (aggregate)
+++ legacy.emod (dcb)
…
converged in 2 attempts. Review the diff above; --apply to write.
```

Overlap with proposal [04](./04-lint-quickfixes-lsp.md): the same `[]Suggestion`,
anchored to the `dcb/*` diagnostic positions, can be served as LSP code actions so
a "fix untagged event" lightbulb appears in the editor. The wiring lives in 04;
the DCB-specific reasoning that produces the suggestion lives here.

## Worked Example

A first-pass DCB context that compiles but trips two `dcb/*` rules: `OrderPlaced`
has no tags, and `SendNotification` reads two event types with no predicate.

```emod
model "Order Management"

actor "Customer"

context "Fulfillment" mode dcb {
  slice "Place Order" {
    command PlaceOrder {
      fields {
        customerId string required
        orderType  string required
        total      int    required
      }
    }

    event OrderPlaced {
      fields {
        orderId    string    required
        customerId string    required
        orderType  string    required
        total      int       required
        placedAt   timestamp required
      }
    }

    flow {
      command -> event: PlaceOrder -> OrderPlaced
    }
  }

  slice "Notify Customer" {
    command SendNotification {
      decides_on {
        events [PaymentAuthorized, OrderShipped]
      }
      fields {
        message string required
      }
    }

    event NotificationSent {
      tags {
        entity: customerId
      }
      fields {
        notificationId string required
        orderId        string required
        customerId     string required
        message        string required
      }
    }

    flow {
      command -> event: SendNotification -> NotificationSent
    }
  }
}
```

`emod lint` reports:

```
Fulfillment > Place Order > OrderPlaced       dcb/untagged-event   error
Fulfillment > Notify Customer > SendNotification  dcb/query-too-broad  warning
```

`emod ai dcb suggest` grounds the model on the DCB brief, the two rule
descriptions, and the JSON export, then returns (json form, trimmed):

```json
[
  {
    "kind": "event_tags",
    "target": "OrderPlaced",
    "rule": "dcb/untagged-event",
    "tags": [
      { "key": "entity",   "field_ref": "customerId" },
      { "key": "category", "field_ref": "orderType"  }
    ],
    "rationale": "OrderPlaced is owned by a customer and shaped by an order type; both are entities other commands decide against, matching the entity/category tags already used by OrderShipped and NotificationSent."
  },
  {
    "kind": "narrow_query",
    "target": "SendNotification",
    "rule": "dcb/query-too-broad",
    "events": ["PaymentAuthorized", "OrderShipped"],
    "predicate": "tag(entity = customerId)",
    "rationale": "A notification is per-customer; scoping the read to entity bounds it to one customer's events and removes the missing-where-clause finding."
  }
]
```

Rendered into the file (the form `--apply` writes), the two slices become:

```emod
    event OrderPlaced {
      tags {
        entity  : customerId
        category: orderType
      }
      fields {
        orderId    string    required
        customerId string    required
        orderType  string    required
        total      int       required
        placedAt   timestamp required
      }
    }
```

```emod
    command SendNotification {
      decides_on {
        events [PaymentAuthorized, OrderShipped]
        where tag(entity = customerId)
      }
      fields {
        message string required
      }
    }
```

emod renders these, runs `pipeline.Check`, confirms both `dcb/*` findings are
gone and nothing new fired, and only then presents them. The author keeps
`entity` for `OrderPlaced` but may decide `category` is over-modeling and drop it —
the kind of judgment the rationale exists to support and the reason the change is
not applied automatically. (Had every command keyed only on `entity`, the context
would then trip `dcb/single-tag-everywhere`, and a follow-up `extra_tag_key`
suggestion would propose `category` as the missing routing dimension; and a
`category` declared on events but used by no `decides_on` would surface as
`dcb/orphan-tag-key`, answered by an `orphan_tag_key` suggestion that either routes
a command on it or recommends removal.)

## Implementation Plan

**Phase 1 — Tag and query suggestions (M).**
- `internal/ai/dcb`: the `Suggestion` schema, the JSON Schema for structured
  output, and the renderer that turns a `Suggestion` into formatted DSL (reusing
  the existing `fmt` writer) with `field_ref` validation against declared fields.
- The DCB brief (`internal/ai/dcb/brief.md`, `go:embed`), distilled from
  [`../dcb-proposal.md`](../dcb-proposal.md), with a snippet from
  `examples/dcb_model.emod`.
- Prompt assembly pulling `linter.RuleDescription` for the in-play rules and the
  JSON export for context.
- `emod ai dcb suggest` for `dcb/untagged-event` and `dcb/query-too-broad`, with
  the verify step (`pipeline.Check` after render). `--format text|json`, `--apply`.
- Unit tests against a mock `llm.Model` (canned suggestions): renderer correctness,
  `field_ref` rejection, and that a suggestion reintroducing a `dcb/*` finding is
  dropped.

**Phase 2 — Orphan and single-tag suggestions (S).**
- `orphan_tag_key` (route-or-remove) and `extra_tag_key` suggestion kinds and
  their renderers.
- `--rule` and `--target` filters.

**Phase 3 — Aggregate → DCB conversion (L).**
- `emod ai dcb convert <context> <file>` driving `GenerateAndRepair` over the
  selected context, with the `dcb/*` rules as the convergence oracle.
- Diff output, residual-diagnostics reporting, bounded `maxAttempts`, best-effort
  return on non-convergence.
- Tests: mock model returns a clean rewrite (converges attempt 1), and a sequence
  that needs one repair pass (converges attempt 2), asserting the diagnostics fed
  back match the linter output.

**Phase 4 — Editor surface (S, depends on 04).**
- Emit the `[]Suggestion` as LSP code actions anchored to `dcb/*` diagnostics,
  through the proposal [04](./04-lint-quickfixes-lsp.md) provider.

## Risks & Mitigations

- **Plausible-but-wrong boundaries.** A narrowed `decides_on` can parse, validate,
  and clear `dcb/query-too-broad` while still failing to protect the real
  invariant — the linter cannot know the domain rule. *Mitigation:* suggestions
  are advisory with a rationale; conversion shows a full diff; nothing applies
  without `--apply` and author review.
- **Tag inflation.** The model may invent tag keys to look thorough, then trip
  `dcb/single-tag-everywhere`'s opposite or `dcb/orphan-tag-key`. *Mitigation:*
  the verify step rejects a suggestion that introduces an orphan key, and the
  brief instructs the model to reuse tag keys already present in the context before
  inventing new ones.
- **Conversion non-convergence.** A large aggregate context may not reach a clean
  `dcb` rewrite within `maxAttempts`. *Mitigation:* return the best attempt plus
  residual diagnostics for manual finishing; never write a non-converged result.
- **Brief drift.** The embedded DCB brief can fall behind
  [`../dcb-proposal.md`](../dcb-proposal.md) or the rules. *Mitigation:* the brief
  pulls rule text live from `linter.RuleDescription` rather than restating it, and
  a test asserts every `dcb/*` rule name referenced in the brief still exists in
  `ruleDescriptions`.
- **Cost on whole-file suggest runs.** One model call per finding adds up.
  *Mitigation:* batch findings of the same kind into one structured request;
  default suggestion passes to `medium` effort and reserve `high` for conversion,
  per the foundation's effort guidance.

## Open Questions

- **Multi-field tag inference.** When several fields could each key a tag (e.g.
  `customerId`, `orderId`, `orderType`), should the assistant propose all
  defensible keys ranked, or commit to one? Ranked-with-rationale is safer but
  noisier in the editor.
- **Predicate richness.** The DCB proposal's predicate grammar is `tag(k =
  fieldRef)` with `and`/`or`/`not`. If it later grows set membership
  (`tag(course in courseIds)`), the `narrow_query` schema and renderer must follow;
  worth keeping `Predicate` a structured tree rather than a string sooner if that
  lands.
- **Cross-context boundaries.** `decides_on` referencing events from another
  context is a real DCB pattern but stresses tag-key resolution. Should `convert`
  and `suggest` reason across contexts, or stay within one context for now?
- **Conversion fidelity vs. snapshot semantics.** Aggregate desugaring loses any
  snapshot/projection meaning a team attached to an aggregate (a risk the DCB
  proposal itself flags). Should `convert` warn when it detects an aggregate likely
  relying on those, rather than silently flattening it to tags?
- **When to prefer removal over routing for an orphan key.** `dcb/orphan-tag-key`
  has two valid fixes; choosing between "add a routing command" and "delete the
  key" needs intent the model can only guess. Should the assistant always present
  both and let the author pick?

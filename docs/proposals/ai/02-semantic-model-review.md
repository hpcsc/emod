# Semantic Model Review

> Part of the [emod AI proposals](./README.md). Uses the [AI foundation](completed/00-llm-foundation.md): the
> `llm.Model` port, the providers, the support tiers, the configuration and the repair loop. This document
> does not specify them again. It also holds the [DCB knowledge](#dcb-knowledge) that the AI panel of the
> desktop app ([03](./03-desktop-ai-panel.md)) and the rename and fix suggestions
> ([06](./06-rename-and-fix-suggestions.md)) use.

## Problem

`emod lint` is fast, deterministic and shallow. Each rule in `internal/linter/descriptions.go` is one of
three kinds:

- **A match on a name:** `state-obsession`, `property-sourcing`, `command-in-disguise`,
  `command-past-tense`, `view-naming`.
- **A count:** `left-chair`, `god-view`, `clickbait-event`, `dcb/query-too-broad`.
- **A check on the structure:** the DCB mode rules and the other `dcb/*` rules,
  `automation/missing-todo-list`, `view/never-read`, `flow/rejection-without-spec`, and the `spec/*` rules.

That is the correct design for these rules. They cost nothing, they give the same answer each time, and they
are never wrong about what they measure. But they measure form, not meaning. A model can pass every rule
and still have these problems:

- **An event names a step in code, not a fact in the business.** `OrderProcessed` and `RequestHandled` are
  in the past tense and have fields, so no rule reports them.
- **A boundary is in the wrong place.** If `CustomerCase` belongs in its own context is a judgement about
  cohesion. Lint has no concept of cohesion.
- **A failure has no event.** A command emits only a success event, but the business can refuse or fail
  the action. No rule counts what must be in the model.
- **One command does two jobs.** The name is correct and the fields are normal.
- **The model uses two words for one concept.** `customerId` and `accountId`, or `Submit` and `Record`,
  for the same thing in two contexts.
- **A rule is in the prose but not in an invariant.** A slice description says "at most twice a year", but
  no invariant states the rule. The `spec/*` rules look only at declared invariants, so they cannot see it.
- **A spec does not hold.** A success spec has a `given` without the history that its outcome needs, for
  example a charge for a subscription that never started.
- **A DCB boundary uses the wrong tag.** A command that protects a rule for one order decides on all the
  events of the customer.

A reviewer in a workshop finds these problems. No rule on names, counts or structure can find them.

DCB contexts need more than a finding. The `dcb/*` lint rules say that an event has no tag or that a query
is too broad. They do not say which tag key to add, which field the key reads, or which predicate protects
the rule of the command. The review, the code actions of [06](./06-rename-and-fix-suggestions.md) and the
conversion of a context in the AI panel of [03](./03-desktop-ai-panel.md) all need these answers. One
package, `internal/ai/dcb`, gives them the same answers. See [DCB knowledge](#dcb-knowledge).

## Goals

- **G1:** `emod ai review <file>` reads a model and reports semantic findings that lint cannot report.
- **G2:** The same review is available in each surface where people work on the model: the CLI, CI, the MCP prompt
  `review-model` of [01](./01-mcp-server.md), the Review button in the AI panel of the desktop app
  ([03](./03-desktop-ai-panel.md)), and an LSP command.
- **G3:** Findings go through `internal/diagnostic` as lint findings do: the same `diagnostic.Entry`, the same
  severities and the same JSON shape, with more fields.
- **G4:** Precision is more important than recall. Each finding has a confidence, and emod removes findings below a
  threshold. A noisy review is worse than no review.
- **G5:** Each finding is advisory and anchored to an element of the model. It names the problem, the element and
  a direction. It never edits the file.
- **G6:** In CI, the review is stable and advisory. A cache gives the same findings for the same source, and an AI
  finding never fails a build alone.
- **G7:** `internal/ai/dcb` holds the DCB knowledge: a brief, a suggestion schema for `tags` and `decides_on`, and
  a check. The review, the code actions of [06](./06-rename-and-fix-suggestions.md) and the conversion in
  [03](./03-desktop-ai-panel.md) use it, so they give the same answers.
- **G8:** Each `dcb/*` finding gets concrete suggestions with the real field names of the events, and with the tag
  keys that the context declares before new keys. Each suggestion names the invariants that it protects.
- **G9:** `oracle.Check` decides if a DCB suggestion is valid. Each suggestion is advisory: nothing changes the file
  until a person accepts it.
- **G10:** Unit tests use `llm.Fake` and need no network. An evaluation corpus measures the precision of the review
  on the tested tier.

## Non-Goals

- **Fixes in a finding.** The review reports. A new name for a lint finding is
  [06](./06-rename-and-fix-suggestions.md). A structural edit, such as "add a declined event", goes to the
  chat of the AI panel ([03](./03-desktop-ai-panel.md)) or to an agent. A suggested spec is
  [05](./05-spec-suggestions.md). A DCB finding of the review gives a direction only. The concrete tags and
  predicates reach the person as a code action of [06](./06-rename-and-fix-suggestions.md) or as a
  conversion in [03](./03-desktop-ai-panel.md).
- **A DCB command.** No `emod ai dcb` command exists. An agent user asks the MCP host for a DCB change, and
  the MCP server of [01](./01-mcp-server.md) checks the result.
- **Changes to DCB.** The DCB knowledge uses the DSL and the rules as `docs/dsl-reference.md` and
  `internal/linter` define them. An event store and its append conditions are out of scope.
- **A replacement for lint.** The lint rules stay. `emod lint` works with no provider configured.
- **The repair loop.** The review produces no `.emod` source, so it does not use `repair.Generate`.
- **Checks that a count or the structure can decide.** These go to the linter, not to the review. For
  example, an outcome event that no spec names is the lint rule `spec/outcome-without-spec` of
  [05](./05-spec-suggestions.md).

## How It Works

### Lint and review: a division of labour

The rule: if a match on a name, a count or the structure of the model can decide a question, lint owns it.
The review owns what needs knowledge of the domain. The prompt gives the model every lint rule with its
description, and tells the model not to report these concerns.

| Concern | Owner | Why |
| --- | --- | --- |
| A generic suffix, `<Aggregate><Field>Changed`, `Initiated`, a command that ends in `ed`, a view name without `View` | Lint: `state-obsession`, `property-sourcing`, `command-in-disguise`, `command-past-tense`, `view-naming` | A match on a name |
| A command in three or more flows, a view on five or more events, an event with one ID field | Lint: `left-chair`, `god-view`, `clickbait-event` | A count |
| A DCB construct in the wrong mode, an untagged event, a broad query, one tag key everywhere, an orphan tag key | Lint: `dcb-in-aggregate-mode`, `aggregate-in-dcb-mode`, `dcb/*` | Structure and counts |
| An automation with no todo list, a view that nothing reads | Lint: `automation/missing-todo-list`, `view/never-read` | Structure |
| A command with no spec, no rejection spec, an invariant that no spec rejects, a rejection edge with no spec, a `given` outside the boundary | Lint: `spec/command-without-spec`, `spec/no-rejection-path`, `spec/invariant-never-exercised`, `flow/rejection-without-spec`, `spec/given-outside-boundary` | Structure |
| An event that names a step in code, not a business fact | Review: `ai/hollow-event` | Meaning |
| An aggregate or a context with no cohesion, or one concern in two places | Review: `ai/boundary-smell` | Cohesion |
| A failure or rejection outcome that has no event | Review: `ai/missing-event` | What must exist |
| A gap between two slices where an intermediate slice must be | Review: `ai/missing-slice` | What must exist |
| One command that does two jobs | Review: `ai/command-overloaded` | Intent |
| A name that passes the name rules but is weak for the domain | Review: `ai/weak-naming` | Meaning |
| Two words for one concept across the model | Review: `ai/language-drift` | A read of the whole model |
| A business rule in a description or a field that no invariant states | Review: `ai/unstated-invariant` | Meaning of the prose |
| A spec whose `given` does not have the history that its outcome needs | Review: `ai/spec-history-gap` | Meaning of the events |
| A spec whose name, or the description of its slice, contradicts its `then` | Review: `ai/spec-contradicts-intent` | Meaning of the prose |
| A `decides_on` keyed on a tag that does not match the rule it protects | Review: `ai/dcb-boundary-mismatch` | The [DCB knowledge](#dcb-knowledge) |
| A tag key that names a value, not a concern that a decision routes on | Review: `ai/dcb-tag-without-concern` | The [DCB knowledge](#dcb-knowledge) |
| A semantic problem outside the set above | Review: `ai/other` | Escape for new kinds |

Each review rule has the `ai/` prefix, so it never has the name of a lint rule, and a filter on the prefix
separates the two.

### What the model reads

emod models are small: the largest example is less than 500 lines. Therefore the review sends the whole
model, not a part of it. `internal/ai/review` puts the prompt together in Go:

1. The `.emod` source with line numbers. The model reads names, descriptions and intent from it.
2. The JSON export from `export.ExportJSON`. It gives an unambiguous structure, with a position on each
   element.
3. The lint findings for the current source, from `oracle.Run`. The model sees what lint already reports.
4. The description of each lint rule, from `linter.RuleDescription`, as the list of concerns not to report.
5. The taxonomy above and the finding schema below. The schema goes in `GenerateRequest.Schema`, so the
   answer is a list of findings that conforms to the schema, not prose.
6. The DCB prompt from `dcb.SystemPrompt`, when the model has a context in `dcb` or `mixed` mode. See
   [DCB knowledge](#dcb-knowledge).
7. A short glossary of the emod concepts: `model`, `actor`, `context` and `mode`, `aggregate`,
   `invariants`, `slice`, `description`, `trigger`, `command`, `event`, `view`, `automation`,
   `translation`, `flow`, `spec`, `tags` and `decides_on`.

The review uses `EMOD_AI_MODEL` at effort `high`. The MCP prompt `review-model` of
[01](./01-mcp-server.md) serves the same system prompt and taxonomy from this package, so a host model
reviews with the same rules.

### The finding schema

The model names an element of the model, not a line:

```go
package review

// Element names the construct a finding is about.
// Scope names the slice, aggregate or context that holds it when the name alone is ambiguous,
// as for a spec.
type Element struct {
	Kind  string `json:"kind"` // context, aggregate, slice, command, event, view, automation, translation, invariant, spec
	Name  string `json:"name"`
	Scope string `json:"scope,omitempty"`
}

type Finding struct {
	RuleID     string  `json:"rule_id"`    // one of the ai/* ids above
	Message    string  `json:"message"`    // one sentence: what is wrong, and why
	Element    Element `json:"element"`
	Severity   string  `json:"severity"`   // "info" or "warning", never "error"
	Confidence float64 `json:"confidence"` // 0.0 to 1.0
	Direction  string  `json:"direction"`  // the direction of a fix, not a rewrite
	Evidence   string  `json:"evidence"`   // what in the model led to the finding
}
```

emod finds the element in the parsed model and takes its position from the AST. emod drops a finding that
names an element that is not in the model. Therefore a finding never points at a line that the model
guessed.

`ToEntry` turns a `Finding` into a `diagnostic.Entry` with the position of the element and
`RuleName: f.RuleID`. `Entry.String()`, the JSON formatter of `internal/cli/lint.go` and
`lsp.ConvertDiagnostics` work with no change. The fields that `diagnostic.Entry` does not have
(`confidence`, `direction`, `evidence` and `element`) go in a side table that the JSON output, the LSP hover
and the desktop app read.

The severity of an AI finding is never `error`. Only deterministic checks report errors: the validator, and
the lint rules `left-chair` and `dcb/untagged-event`. A finding with a probability must not block like a
fact.

### Precision controls

1. **Confidence threshold.** emod removes each finding below the threshold. The default is `0.7`, and
   `--min-confidence` changes it.
2. **Capped severity.** An AI finding is `info` or `warning`, so a false positive never fails a build
   alone.
3. **Element check.** A finding on an element that the model does not have is a hallucination. emod
   drops it.
4. **Duplicate filter.** emod drops an AI finding on the same element as a lint finding of a related rule.
   The relations are fixed in code: `ai/weak-naming` and `ai/hollow-event` relate to the five name rules,
   `ai/command-overloaded` relates to `left-chair`, and `ai/dcb-tag-without-concern` relates to
   `dcb/orphan-tag-key`.
5. **Critic pass (`--verify`).** A second call sends the findings that survive, with the model, to
   `EMOD_AI_MODEL_CHEAP`. The instruction: for each finding, decide if it is a real problem in the domain,
   and return keep or drop with a reason. emod removes the dropped findings. The pass costs more time and a
   little money, so it is off by default in a terminal and on in CI.

### Cache

The cache key is a SHA-256 hash of the source, the version of the prompt, the provider, the model, the
effort, the threshold and the `--verify` flag. With `--cache`, the review reads and writes
`<user cache dir>/emod/review/<key>.json`, where the user cache dir comes from `os.UserCacheDir`. The same
input gives the same findings and costs no tokens. A CI job keeps this directory between runs with the
cache step of the CI system.

### Evaluation corpus

`internal/ai/review/testdata/corpus/` holds models with known semantic problems and models with none. Each
model has a file with the expected findings, as an element and a rule id. A test with the build tag `eval`
runs the review against a live model and reports the precision and the recall for each rule id. Unit tests
never run it. The corpus runs on the tested tier before each release, and its results set the default
threshold. A user can run it on any model of the compatible tier to measure that model.

## DCB knowledge

Dynamic Consistency Boundary (DCB) modeling has the highest conceptual load in emod. In aggregate mode, the
aggregate gives the author a boundary. In DCB mode, there is no aggregate. For each event, the author
decides which entities and relationships it carries as `tags`. For each command, the author decides which
event types and which tag predicate form its boundary, as `decides_on`. If the tags are too coarse, each
command reads too much. If the tags are wrong, the boundary protects the wrong rule.

The linter is a good judge and a poor teacher. `dcb/untagged-event`, `dcb/query-too-broad`,
`dcb/single-tag-everywhere` and `dcb/orphan-tag-key` find the problems, and `spec/given-outside-boundary`
checks a spec against the `decides_on` of its command. The answers need the fields of the events, the
invariants of the context, the specs and the intent of each command. `internal/ai/dcb` gives these answers
to three surfaces:

| Surface | Entry point | What the person sees |
| --- | --- | --- |
| This review | The review prompt holds `dcb.SystemPrompt` when the model has a `dcb` or `mixed` context | The findings `ai/dcb-boundary-mismatch` and `ai/dcb-tag-without-concern`, each with a direction and no edit |
| The LSP of [06](./06-rename-and-fix-suggestions.md) | The command `emod.suggestDCB` on a `dcb/*` finding calls `dcb.Suggest` and `dcb.Render` | A code action, a choice of suggestions with their rationale, then the edit |
| The AI panel of [03](./03-desktop-ai-panel.md) | "Convert to DCB" on a context, or an edit of a DCB context, adds `dcb.SystemPrompt` to the edit prompt | The progress of each attempt, then the diff, with Accept and Reject |

### The brief

`internal/ai/dcb/brief.md` is embedded with `go:embed`, so it has the same version as the rules that it
explains. It comes from section 9 of `docs/dsl-reference.md` and from
[`../completed/dcb-proposal.md`](../completed/dcb-proposal.md). It states:

- An event declares `tags { <key> = <field> }`. The field must be a field of the event.
- A command declares `decides_on { events = [...] where = <predicate> }`. A predicate is an HCL expression of
  `tag(<key>, <field>)` terms with `&&`, `||`, `!` and parentheses.
- In `dcb` mode, slices sit directly under the context, and the `invariants` block sits on the context.
- The `decides_on` of a command is its boundary. The correct boundary is the smallest set of event types and
  the narrowest predicate that still lets the command check each invariant that it protects.
- `&&` reads the events that carry both values. `||` reads the events that carry either value. Two
  invariants about two entities need `||`.
- A tag key names a concern that a decision routes on, such as an entity or a relationship. A tag key is not
  a label for a value.
- The steps of a conversion: rewrite the context as `mode = dcb` with each slice directly under it, move
  the invariants of each aggregate to the context, tag each event with the entities that its invariants
  concern, give each command a `decides_on` that reads the events that its invariants need, and keep each
  spec.

The brief also holds a snippet from `examples/dcb_model.emod`. A unit test checks that each rule name in the
brief is a rule of `linter.RuleDescription`, and that the snippet passes `oracle.Check`.

### The system prompt

`SystemPrompt(model *ast.Model, contextName string) string` puts the DCB prompt together:

1. The brief.
2. The description of each `dcb/*` rule in play, from `linter.RuleDescription`.
3. The context from `export.ExportJSON`: its events with their fields and tags, its commands with their
   fields and `decides_on`, and its flows.
4. The invariants of the context and the specs of its slices. A rejection spec says which rule a command
   protects. A new boundary must keep the `given` of each spec inside it.

### The suggestion schema

The model returns intent, not `.emod` text. The schema goes in `GenerateRequest.Schema`:

```go
package dcb

type Kind string

const (
	KindEventTags    Kind = "event_tags"     // answers dcb/untagged-event
	KindNarrowQuery  Kind = "narrow_query"   // answers dcb/query-too-broad
	KindExtraTagKey  Kind = "extra_tag_key"  // answers dcb/single-tag-everywhere
	KindOrphanTagKey Kind = "orphan_tag_key" // answers dcb/orphan-tag-key
)

type Tag struct {
	Key   string `json:"key"`   // for example "order"
	Field string `json:"field"` // a field that the event declares, for example "orderId"
}

type EventTags struct {
	Event string `json:"event"`
	Tags  []Tag  `json:"tags"` // tags to add to the event
}

type Query struct {
	Command string   `json:"command"`
	Events  []string `json:"events"`
	Where   string   `json:"where"` // an HCL predicate, for example "tag(order, orderId)"
}

type Suggestion struct {
	Kind      Kind        `json:"kind"`
	Rule      string      `json:"rule"`                 // the dcb/* rule that the suggestion answers
	AddTags   []EventTags `json:"add_tags,omitempty"`   // tags to add
	Queries   []Query     `json:"queries,omitempty"`    // a decides_on to write for each command
	RemoveKey string      `json:"remove_key,omitempty"` // orphan_tag_key: the key to remove from every event
	Protects  []string    `json:"protects,omitempty"`   // the invariants that the boundary protects
	Rationale string      `json:"rationale"`            // one or two sentences for the person
}
```

`Render(source string, suggestion Suggestion) (string, error)` applies the intent to the AST and prints the
result with `formatter.Format`. Therefore the model cannot write broken braces or tokens. `Render` refuses a
suggestion that names a field that the event does not declare, or an event or a command that the context
does not have. It reads `Where` with `parser.ParsePredicate`, an exported wrapper around the reader that the
parser uses for a `where`. It refuses a predicate that does not parse.

### The check

`Suggest(ctx context.Context, model llm.Model, source string, finding *diagnostic.Entry) ([]Suggestion, llm.Usage, error)`
asks for up to three suggestions for one finding, with `EMOD_AI_MODEL` at effort `medium`. It renders each
suggestion and runs `oracle.Check` on the result. A suggestion passes when all of these are true:

- The check reports no error that the original source does not have.
- The finding that the suggestion answers is gone.
- The check reports no `dcb/*` finding that the original source does not have.
- The check reports no `spec/given-outside-boundary` finding that the original source does not have.

`Suggest` returns only the suggestions that pass. It checks each suggestion alone against the current
source. A suggestion that needs another suggestion first does not pass now, but it can pass after the person
applies the other one. The example below shows this.

### The rules for advisory suggestions

- Each suggestion has a rationale. It names the invariants that it protects when the context declares them.
- A suggestion reuses the tag keys that the context declares before it adds a new key.
- When more than one field can carry a tag, `Suggest` returns up to three ranked suggestions.
- For `dcb/orphan-tag-key`, `Suggest` returns a route and a removal when both pass. The person chooses,
  because the choice needs intent that the model can only guess.
- No surface applies a suggestion without the acceptance of a person.

### The conversion of a context

"Convert to DCB" in the AI panel of [03](./03-desktop-ai-panel.md) sends an edit with the instruction
"convert context `<name>` to mode dcb". The edit prompt holds `dcb.SystemPrompt`, so the model follows the
steps of a conversion in the brief. The panel runs `repair.Generate` with `repair.WithBaseline` of
[05](./05-spec-suggestions.md). Therefore a lint finding outside the context, which the original source
already has, does not stop the loop.

The conversion is complete when the result parses, validates, and adds no `dcb/*`,
`aggregate-in-dcb-mode` or `spec/given-outside-boundary` finding. The group then accepts or rejects the
diff. If the loop does not converge, the panel shows the last result and its diagnostics, and writes
nothing.

### Example: three suggestions and one finding

The validator accepts this first DCB context, but lint reports three findings, so `emod validate` exits 1:

```emod
emod = 1

model "Order Management" {
}

context "Fulfillment" {
  mode = dcb

  slice "Place Order" {
    command "PlaceOrder" {
      fields {
        customerId = required(string)
        orderType  = required(string)
      }
    }

    event "OrderPlaced" {
      fields {
        orderId    = required(string)
        customerId = required(string)
        orderType  = required(string)
        placedAt   = required(timestamp)
      }
    }

    flow = <<-FLOW
      command -> event:    PlaceOrder -> OrderPlaced
    FLOW
  }

  slice "Ship Order" {
    command "ShipOrder" {
      fields {
        orderId    = required(string)
        customerId = required(string)
      }

      decides_on {
        events = [OrderShipped]
        where  = tag(entity, customerId)
      }
    }

    event "OrderShipped" {
      tags {
        entity = customerId
      }

      fields {
        orderId    = required(string)
        customerId = required(string)
        shippedAt  = required(timestamp)
      }
    }

    flow = <<-FLOW
      command -> event:    ShipOrder -> OrderShipped
    FLOW
  }

  slice "Notify Customer" {
    command "SendNotification" {
      fields {
        customerId = required(string)
        message    = required(string)
      }

      decides_on {
        events = [OrderShipped]
      }
    }

    event "NotificationSent" {
      tags {
        entity = customerId
      }

      fields {
        customerId = required(string)
        message    = required(string)
      }
    }

    flow = <<-FLOW
      command -> event:    SendNotification -> NotificationSent
    FLOW
  }
}
```

```
orders.emod:6: [dcb/single-tag-everywhere] context "Fulfillment" uses only tag key "entity" across all decides_on predicates in dcb mode
orders.emod:17: [dcb/untagged-event] event "OrderPlaced" is missing tags in dcb-mode context "Fulfillment"
orders.emod:62: [dcb/query-too-broad] command "SendNotification" has a broad query: has no where clause
```

The developer opens the code actions of the three findings in the editor. `Suggest` returns one suggestion
for each:

| Finding | Suggestion | Rationale | Check against the source at that time |
| --- | --- | --- | --- |
| `dcb/query-too-broad` on `SendNotification` | `where = tag(entity, customerId)` | A notification concerns one customer, so the command reads the shipments of that customer only. | Passes. `dcb/single-tag-everywhere` stays, but the original source has it. |
| `dcb/untagged-event` on `OrderPlaced` | Add `entity = customerId` | The context already routes on `entity`, and the event carries `customerId`. | Passes. |
| `dcb/single-tag-everywhere` on `Fulfillment` | Add `order = orderId` to `OrderShipped`, and set `where = tag(order, orderId)` on `ShipOrder` | The rule that an order ships once concerns one order, not all the orders of a customer. | Fails before the first suggestion: no predicate then uses `entity`, so the check reports `dcb/orphan-tag-key`. Passes after the first suggestion. |

The JSON form of the third suggestion:

```json
{
  "kind": "extra_tag_key",
  "rule": "dcb/single-tag-everywhere",
  "add_tags": [{ "event": "OrderShipped", "tags": [{ "key": "order", "field": "orderId" }] }],
  "queries": [{ "command": "ShipOrder", "events": ["OrderShipped"], "where": "tag(order, orderId)" }],
  "rationale": "The rule that an order ships once concerns one order, not all the orders of a customer."
}
```

The developer accepts the three suggestions in the order of the table. Each one passes the check when the
developer accepts it. The result has no finding:

```diff
     event "OrderPlaced" {
+      tags {
+        entity = customerId
+      }
+
       fields {
...
       decides_on {
         events = [OrderShipped]
-        where  = tag(entity, customerId)
+        where  = tag(order, orderId)
       }
...
     event "OrderShipped" {
       tags {
         entity = customerId
+        order  = orderId
       }
...
       decides_on {
         events = [OrderShipped]
+        where  = tag(entity, customerId)
       }
```

The review reads the first model with the DCB prompt. It reports `ai/dcb-boundary-mismatch` on `ShipOrder`:
the command decides on the shipments of one customer, but the rule that it protects concerns one order. The
direction is "key the decision on the order". The review does not edit the model. The developer gets the
edit from the third suggestion, or the facilitator gets it from the AI panel.

## Interface

### CLI

`emod ai review` is a subcommand of the `ai` command group:

```
emod ai review <file> [flags]

Flags:
      --format text|json       output format, as for emod lint (default: text)
      --min-confidence <n>     drop findings below this confidence (default: 0.7)
      --severity info|warning  show findings at or above this severity (default: info)
      --verify                 run the critic pass with EMOD_AI_MODEL_CHEAP
      --cache                  read and write the review cache
      --effort <level>         low|medium|high|xhigh (default: high)
```

Exit codes follow the lint convention, with one difference. `emod ai review` returns `1` when at least one
finding survives the filters and `0` when none does. It never returns `2`, which `emod lint` keeps for
deterministic errors. With no provider configured, the command prints the setup steps and exits with code
`0`, as `emod ai` does. Therefore a CI job with no credentials does not fail.

The `--format json` output has the fields of `emod lint --format json` (`file`, `line`, `rule`, `severity`,
`message`) and also `column`, `element`, `confidence`, `direction` and `evidence`.

### CI

The review in CI is advisory. Lint gates the build, and the review adds notes:

```yaml
- run: emod lint model.emod
- run: emod ai review model.emod --verify --cache --format json > review.json
  continue-on-error: true
```

### MCP prompt `review-model`

The MCP server of [01](./01-mcp-server.md) serves the prompt `review-model`. The prompt gives the host model
three steps:

1. Call `emod_lint`.
2. Review the model with the taxonomy and the schema of this package.
3. Report each finding with its element and its direction.

The host model uses the credentials of the host. It answers in the chat, and nothing writes the file.

### Desktop app

The AI panel of the desktop app ([03](./03-desktop-ai-panel.md)) has a Review button. The app
runs `internal/ai/review` in its Go process.

- The findings show in the diagnostics panel next to the lint findings, with an `AI` label and the
  confidence.
- The diagram marks the node of each element that has a finding.
- The facilitator can reject a finding. A rejected finding stays hidden for the session.
- The facilitator can send a finding to the chat of the AI panel as an instruction. The chat then proposes
  an edit through the repair loop, and the group accepts or rejects the diff.

### LSP

- The command `emod.review`, through `workspace/executeCommand`, runs the review. A code lens on the
  `model` line, "Run AI review", calls the command. The review never runs on the debounce after a change,
  because it is slow and costs tokens.
- The findings go into the same `textDocument/publishDiagnostics` stream as the lint findings, with
  `source: "emod ai"`. The rule id goes in the `code` field that [06](./06-rename-and-fix-suggestions.md) adds to
  the LSP `Diagnostic`.
- After each change, the server finds the element of each finding in the new parse and moves the finding
  to the new position of the element. It drops a finding whose element is gone. Because each finding is
  anchored to an element, an edit in another part of the file does not remove it.
- The hover on a finding shows its direction, evidence and confidence.

## Worked Example

This model passes `emod validate` and `emod lint` with no finding:

```emod
emod = 1

model "Subscription Billing" {
}

context "Billing" {
  aggregate "Subscription" {
    invariants {
      OneActivePerCustomer = "A customer holds at most one active subscription"
      PauseOnlyWhenActive  = "Only an active subscription can be paused"
      NoChargeWhilePaused  = "A paused subscription is not charged"
    }

    slice "Start Subscription" {
      command "StartSubscription" {
        fields {
          customerId = required(string)
          planId     = required(string)
        }
      }

      event "SubscriptionStarted" {
        fields {
          subscriptionId = required(string)
          customerId     = required(string)
          planId         = required(string)
          startedAt      = required(timestamp)
        }
      }

      flow = <<-FLOW
        command -> event:    StartSubscription -> SubscriptionStarted
      FLOW

      spec "starts a first subscription" {
        when = StartSubscription
        then = [SubscriptionStarted]
      }

      spec "refuses a second active subscription" {
        given = [SubscriptionStarted]
        when  = StartSubscription
        then  = rejected(OneActivePerCustomer)
      }
    }

    slice "Pause Subscription" {
      description = "A customer pauses for up to three months, at most twice a year."

      command "PauseSubscription" {
        fields {
          subscriptionId = required(string)
          months         = required(int)
        }
      }

      event "SubscriptionPaused" {
        fields {
          subscriptionId = required(string)
          months         = required(int)
          pausedAt       = required(timestamp)
        }
      }

      flow = <<-FLOW
        command -> event:    PauseSubscription -> SubscriptionPaused
      FLOW

      spec "pauses an active subscription" {
        given = [SubscriptionStarted]
        when  = PauseSubscription
        then  = [SubscriptionPaused]
      }

      spec "refuses to pause a paused subscription" {
        given = [SubscriptionStarted, SubscriptionPaused]
        when  = PauseSubscription
        then  = rejected(PauseOnlyWhenActive)
      }
    }

    slice "Charge for Period" {
      command "ChargeSubscription" {
        fields {
          subscriptionId = required(string)
          amount         = required(int)
        }
      }

      event "SubscriptionCharged" {
        fields {
          subscriptionId = required(string)
          amount         = required(int)
          chargedAt      = required(timestamp)
        }
      }

      flow = <<-FLOW
        command -> event:    ChargeSubscription -> SubscriptionCharged
      FLOW

      spec "charges the plan price" {
        when = ChargeSubscription
        then = [SubscriptionCharged]
      }

      spec "refuses to charge a paused subscription" {
        given = [SubscriptionStarted, SubscriptionPaused]
        when  = ChargeSubscription
        then  = rejected(NoChargeWhilePaused)
      }
    }
  }
}
```

Lint has nothing to report. The events are in the past tense and carry data. The commands are imperative.
Each command has a success spec and a rejection spec, and a rejection spec names each invariant. No view or
automation is in the model.

The review reads the meaning and reports three findings:

```
$ emod ai review billing.emod
billing.emod:47: [ai/unstated-invariant] slice "Pause Subscription" states two rules, "up to three months" and "at most twice a year", that no invariant declares (confidence 0.88)
    direction: declare invariants such as PauseAtMostThreeMonths and TwoPausesPerYear on the aggregate "Subscription", and add a rejection spec for each
billing.emod:83: [ai/missing-event] command "ChargeSubscription" has only the success event "SubscriptionCharged", but a payment provider can decline a charge (confidence 0.86)
    direction: add an event such as SubscriptionChargeDeclined, and a slice that reacts to it for a retry or for dunning
billing.emod:102: [ai/spec-history-gap] spec "charges the plan price" has an empty given, so it charges a subscription that never started (confidence 0.81)
    direction: add SubscriptionStarted to the given of the spec
3 findings (0 below the confidence threshold 0.70)
```

The JSON form of the second finding:

```json
{
  "file": "billing.emod",
  "line": 83,
  "column": 7,
  "rule": "ai/missing-event",
  "severity": "warning",
  "message": "command \"ChargeSubscription\" has only the success event \"SubscriptionCharged\", but a payment provider can decline a charge",
  "element": { "kind": "command", "name": "ChargeSubscription" },
  "confidence": 0.86,
  "direction": "add an event such as SubscriptionChargeDeclined, and a slice that reacts to it for a retry or for dunning",
  "evidence": "a charge against a card can fail, but SubscriptionCharged is the only event of the command, and no slice reacts to a failed charge"
}
```

Each finding is a gap that a domain expert sees at once. No rule on names, counts or structure finds them.
The first finding also explains why `spec/invariant-never-exercised` is silent: the rules are not
invariants yet, so lint has nothing to check.

For a DCB example, see the example in [DCB knowledge](#dcb-knowledge). There, the command `ShipOrder`
decides on the events of one customer, but the rule that it protects is about one order. The review reports
`ai/dcb-boundary-mismatch` on `ShipOrder`, with the direction "key the decision on the order".

## Implementation Plan

### Phase 1: core review (M)

- `internal/ai/review`: `Element`, `Finding`, the JSON Schema, `ToEntry`, the element check, the prompt
  assembly, the threshold, the severity cap and the duplicate filter.
- `Review(ctx context.Context, model llm.Model, source string, filename string) ([]Finding, llm.Usage, error)`.
- Unit tests against `llm.Fake` with canned findings. They cover the threshold, the cap, a finding on an
  unknown element, a duplicate of a related lint finding, and `ToEntry`. The tests follow the Go test rules
  of the repository: one umbrella `TestReview`, `t.Run` groups for each operation, scenario names that
  describe behaviour, `testify/require`, and fresh fixtures in each leaf subtest.

### Phase 2: DCB knowledge (M)

- `internal/ai/dcb`: the brief with its test, `SystemPrompt`, the suggestion schema, `Render`, and `Suggest`
  with the check.
- `parser.ParsePredicate`, exported from `internal/parser`.
- The rules `ai/dcb-boundary-mismatch` and `ai/dcb-tag-without-concern` in the review prompt, with
  `dcb.SystemPrompt` when the model has a `dcb` or `mixed` context.
- Tests against `llm.Fake` with canned suggestions: a field that the event does not declare, a predicate
  that does not parse, a suggestion that adds a `dcb/*` finding, and a suggestion that passes only after
  another one.
- The code actions of [06](./06-rename-and-fix-suggestions.md) and the conversion in
  [03](./03-desktop-ai-panel.md) start after this phase.

### Phase 3: CLI and CI (S)

- The `review` subcommand of `emod ai` in `internal/cli`, with the flags above. The text and JSON
  formatters reuse `formatText` and `formatJSON` of `internal/cli/lint.go`, with the extra fields.
- The behaviour with no provider configured, and the cache.

### Phase 4: precision (M)

- The `--verify` critic pass.
- The evaluation corpus and the `eval` test. The default threshold comes from its results.

### Phase 5: MCP prompt (S)

- The prompt `review-model` in [01](./01-mcp-server.md) takes its text from `internal/ai/review`.

### Phase 6: desktop app (M)

- The Review button of the AI panel in [03](./03-desktop-ai-panel.md), the `AI` label in the
  diagnostics panel, the marks on the diagram, and the action that sends a finding to the chat.

### Phase 7: LSP (M)

- The command `emod.review`, the code lens, the move of findings to their elements after each change, and
  the hover. This phase needs the `code` field of the LSP `Diagnostic` from
  [06](./06-rename-and-fix-suggestions.md).

## Risks and Mitigations

- **False positives make users ignore the review.** The controls are in layers: the threshold, the severity
  cap, the element check, the duplicate filter, the critic pass and the corpus. The default gives fewer
  findings with more confidence.
- **Different answers on each run break CI.** The cache gives the same findings for an unchanged model. For
  a changed model, the findings stay advisory: they are never `error`, and the CI step continues on
  failure. Only deterministic lint errors fail a build.
- **Cost and latency.** The review sends the model once, because the model is small. The critic pass uses
  the cheap model. The LSP and the desktop app run the review only when a person asks for it. The CLI shows
  the tokens from `Response.Usage`, and the cache stops repeated calls.
- **Overlap with lint.** The prompt lists every lint rule as a concern not to report. The `ai/` prefix and
  the duplicate filter keep the two sets of findings apart.
- **A weak model on the compatible tier.** The check of emod does not apply to a judgement, so a small model
  can give noisy findings and poor boundaries. The CLI and the desktop panel show the notice for a model
  outside the tested tier. The threshold removes findings with low confidence, the check removes invalid
  DCB suggestions, and the corpus measures any model.
- **A DCB boundary that passes the check but protects the wrong rule.** A narrower `decides_on` can parse,
  validate and clear `dcb/query-too-broad`, and still miss the real rule. The linter cannot know the rule.
  Each suggestion names the invariants that it protects and has a rationale, and a person accepts each one.
- **Too many tag keys.** The model can add keys to look thorough. The check refuses a suggestion that adds
  `dcb/orphan-tag-key`, and the brief tells the model to reuse declared keys first.
- **A conversion that does not converge.** A large aggregate context can need more attempts than the limit.
  The panel shows the last result and its diagnostics, and writes nothing.
- **The DCB brief falls behind the DSL.** The brief takes the rule text from `linter.RuleDescription`, and
  its test checks the rule names and the snippet.
- **The model of the business leaves the machine.** With a remote provider, the prompt holds the whole
  model. The documentation of `emod ai review` states this. A team with a confidential domain uses a local
  server on the `openai` provider.
- **The review turns into a fixer.** The schema has a `direction`, not a patch. Fixes go to
  [06](./06-rename-and-fix-suggestions.md), [05](./05-spec-suggestions.md) and
  [03](./03-desktop-ai-panel.md).

## Open Questions

- **Storage of a rejected finding.** In the desktop app, a rejected finding stays hidden for the session.
  To hide it across sessions, emod needs a store: a comment on the element, such as
  `# emod:review-ignore ai/missing-event`, or a file next to the model. A comment travels with the model in
  git. A file keeps the model clean.
- **A threshold for each rule.** `ai/language-drift` can need a higher threshold than `ai/missing-event`.
  One threshold ships first. Thresholds for each rule need data from the corpus.
- **Calibration of the confidence.** The confidence that a model gives itself is not always calibrated. The
  agreement between the review and the critic pass, or repeated samples, can give a better value. This needs
  measurement before the default threshold is final.
- **Very large models.** At some size, the whole model does not fit the context of the model. A review of
  each context, with a final pass across contexts for `ai/language-drift`, can solve this. The size at which
  this is necessary is not known.
- **New rule ids.** If many findings use `ai/other` for one kind of problem, that kind gets its own rule id.
- **The DCB predicate as a tree.** `Where` is a string that `parser.ParsePredicate` reads. A JSON tree of
  `tag`, `&&`, `||` and `!` nodes removes the parse step, but a recursive schema is harder for some servers
  on the `openai` provider.
- **DCB boundaries across contexts.** A `decides_on` can name events of another context. Decide if
  `Suggest` and the conversion read across contexts, or stay in one context.
- **The meaning of an aggregate.** A team can attach a snapshot or a projection to an aggregate. The
  conversion removes the aggregate. Decide if the panel warns when an aggregate has such a meaning.
- **The DCB brief for MCP hosts.** The MCP server of [01](./01-mcp-server.md) can serve the brief as a
  resource, so an agent converts a context with the same knowledge.

# Spec Suggestions

> Part of the [emod AI proposals](./README.md). Uses the [AI foundation](./00-llm-foundation.md): the
> `llm.Model` port, the providers, the configuration and the repair loop. It does not specify them again.

## Problem

Event modeling ends each slice with Given-When-Then specs. emod holds them in the model, in `spec` blocks
with typed example payloads, next to the structure they describe. `emod validate` checks each spec: the
names it uses, the outcome that the pattern of its slice allows, the field names and literal kinds of its
payloads, and the history that its boundary can see. The lint rules find the specs that a model does not
have:

| Rule | What it reports |
| --- | --- |
| `flow/rejection-without-spec` | A `command -> rejected:` entry that no spec of the slice exercises |
| `spec/invariant-never-exercised` | An invariant that no `then = rejected(...)` spec in its scope names |
| `spec/no-rejection-path` | A command whose specs state no rejection |
| `spec/command-without-spec` | A command that no spec exercises, in a model that has specs |
| `spec/given-outside-boundary` | A `given` event that the boundary of the command cannot see |

The rules say where a spec is absent. They do not write it, and a person needs time to write it. For a
rejection spec, the author must find the history that breaks the rule: five loans for "a member holds at
most five copies". For each spec, the author must fill payload values that agree across `given`, `when` and
`then`: the same `copyId` in the history and in the command. In a workshop, the group writes the happy path
and the time ends. The rules stay without a scenario, and the developers build them from memory.

A second problem starts after the workshop. The specs live in the model, but a test harness runs Gherkin
features or JSON fixtures. Teams copy the specs into their harness by hand, and from then on the two
differ.

The work has two parts of a different kind:

1. **Structure.** Which specs are absent, which command each one exercises, and which outcome it states.
   emod computes this from the model. No AI is necessary.
2. **Judgement.** Which history makes the outcome true, which values make the payloads agree, and which
   name tells a domain expert what the spec is about. This is where a language model helps.

This proposal gives the judgement to the model and keeps the structure in emod. The check of emod then
decides if each suggestion is valid. A spec export with no AI closes the second problem.

## Goals

- **G1:** `emod ai specs <file>` suggests the specs that the gap list names, as `spec` blocks in their slices.
- **G2:** Each suggestion closes the gap it targets and adds no diagnostic that the file did not have before.
- **G3:** Each suggestion uses only names that the model declares. emod writes the `spec` block from structured
  output. The model never writes `.emod` text into the file.
- **G4:** Each payload conforms to the declared field types, and the values agree across one spec.
- **G5:** A person sees a diff. Nothing changes the file until the person gives `--apply` or accepts the suggestion
  in the desktop app.
- **G6:** The same suggestions reach the facilitator in the desktop app ([03](./03-desktop-ai-panel.md))
  and an agent through the MCP server ([01](./01-mcp-server.md)).
- **G7:** `emod specs export <file> --format gherkin|json` writes the specs of the model as Gherkin features or JSON
  fixtures, with no AI. The output is deterministic, so CI can check it for drift.

## Non-Goals

- **New rules or new outcomes.** A suggestion exercises only invariants and outcome events that the model
  states. A rule that the model does not state is a modeling decision. The semantic review finds it as a
  finding ([02](./02-semantic-model-review.md)), and the group decides.
- **Complete payloads.** A payload in a spec is partial by design. The export gives the declared field
  types, so a harness can complete the values.
- **Step definitions or production code.** The export writes features and fixtures. The step code stays
  with the team.
- **Proof that the model is correct.** Specs that agree with a wrong model are wrong specs. The review of
  [02](./02-semantic-model-review.md) and the people in the workshop find a wrong model.
- **Questions about the model.** See the [model queries proposal](../model-queries-proposal.md).

## How It Works

```
model ──▶ gap list ──▶ one structured call for each slice ──▶ ast.Spec values
 (AST)    (no AI)       (llm.Model, schema with enums)            │
                                                                  ▼
            diff ◀── converged ◀── oracle.Check against ◀── formatter.Format
                                    the baseline                  ▲
                                         │ new diagnostics        │
                                         └──── repair loop ───────┘
```

### 1. The gap list (no AI)

A new package, `internal/specs`, computes the gap list from the AST: `specs.Gaps(model *ast.Model) []Gap`.
Each gap names a slice, the element under test, and the shape of the spec that closes it:

| Gap | How emod finds it | Shape of the suggested spec |
| --- | --- | --- |
| Rejection edge with no spec | Same predicate as `flow/rejection-without-spec` | `when` is the command; `then = rejected(<invariant>)` |
| Invariant never exercised | Same predicate as `spec/invariant-never-exercised` | `when` is a command in the same scope; `then = rejected(<invariant>)` |
| Command with no rejection spec | Same predicate as `spec/no-rejection-path`, only when the scope declares an invariant | `when` is the command; `then = rejected(<one of the invariants>)` |
| Command with no spec | Same predicate as `spec/command-without-spec`. In a model with no specs, every command. | `when` is the command; `then` is a list of its outcome events |
| Outcome event with no spec | A new lint rule, `spec/outcome-without-spec`: an event in a `command -> event:` entry that no `then` list of the slice names | `when` is the command; `then` is a list that holds the event |
| View with no spec | A view slice with no spec | `then = view(<view>)`; `given` from the events the view subscribes to |
| Automation with no spec | An automation slice with no spec | `when` is the `on` event, or absent for an `every` schedule; `then = command(<command>)` |

The lint rules and the gap list use the same predicates, so they cannot disagree. The predicates move into
a function that `internal/linter` and `internal/specs` both call. A gap that a lint rule reports also
carries the rule name, so a suggestion can say which finding it closes.

For each slice, the gap list also states the **boundary**: the events that a `given` can name. In an
aggregate, the boundary is the events that the slices of the aggregate declare. In a DCB context, the
boundary is the `events` list of the `decides_on` clause of the command, and its `where` predicate. These
are the same boundaries that `spec/given-outside-boundary` checks.

### 2. The structured call

The AI part lives in `internal/ai/specs`. It depends on `llm.Model`, `repair.Generate`, `oracle.Check` and
`internal/specs`. It sends one call for each slice that has gaps. The prompt holds:

- the source of the slice and its `description`
- the gaps of the slice
- each invariant in scope, with its statement
- the declared fields of each event and command that the slice can name
- the boundary, and the `tags` of each event in a DCB context
- the specs that the slice already has, as examples of the style of the model

The answer conforms to a JSON Schema that emod builds for the slice. The schema holds enums, so the model
can only name what the model declares:

```json
{
  "type": "object",
  "required": ["specs"],
  "properties": {
    "specs": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["gap", "name", "given", "then"],
        "properties": {
          "gap":   { "type": "integer", "description": "index into the gap list of the slice" },
          "name":  { "type": "string" },
          "given": { "type": "array", "items": { "$ref": "#/$defs/boundaryEvent" } },
          "when":  { "$ref": "#/$defs/whenElement" },
          "then":  { "$ref": "#/$defs/outcome" }
        }
      }
    }
  }
}
```

`boundaryEvent` is an object with a `name` from the enum of boundary events and a `payload` object. The
`payload` lists the declared fields of that event, each with a string, number or boolean value. `outcome`
is one of four objects: `events` with names from the outcome events of the slice, `rejected` with a name
from the invariants in scope, `view`, or `command`.

The model decides three things:

1. **The history.** A rejection spec needs a `given` that breaks the rule. For `FiveLoansPerMember`, that is
   five `CopyBorrowed` events for one member.
2. **The values.** The values agree across the spec: the `copyId` of the history is the `copyId` of the
   command. In a DCB context, the tagged fields agree with the `where` predicate.
3. **The name.** The name is a short sentence about the behaviour, in the words of the domain, in the style
   of the specs that the model already has: `refuses a copy already on loan`.

The call uses `EMOD_AI_MODEL` at `medium` effort. The choice of a history that breaks a rule needs
reasoning, and a wrong spec misleads the domain experts who read it.

### 3. Render, insert and check

emod turns each answer into `*ast.Spec` values, adds them to the `Specs` of their slice, and prints the
whole file with `formatter.Format`. This is the same path as an edit from the diagram: the AST changes, and
the formatter writes the source. Therefore a suggestion cannot change any other part of the file.

Then emod runs `oracle.Check` on the result. The file can have diagnostics before the call, so the check
compares against a **baseline**: the diagnostics of the file before the call. A suggestion converges when
two conditions are true:

- The result has no diagnostic that the baseline does not have. emod compares diagnostics by rule name and
  message, because the line numbers move.
- The lint finding that each gap carries is not in the result.

The check covers the payloads. The validator reports a field name that the element does not declare, and a
literal that the declared type does not allow:

| Declared type | A literal that the validator accepts |
| --- | --- |
| `string` | Any string |
| `date` | A string in the form `YYYY-MM-DD` |
| `timestamp` | A string in RFC 3339 form |
| `uuid` | A string in the canonical 8-4-4-4-12 hexadecimal form |
| `int` | A number with no fractional part |
| `decimal` | A number |
| `bool` | `true` or `false` |
| A domain type | Any literal. The validator does not check it. |

The loop is the repair loop of the [foundation](./00-llm-foundation.md#the-repair-loop), `repair.Generate`,
with two of its options:

- `repair.WithAssemble` turns the answer into the whole file before the check. Here, the function renders
  the structured answer and inserts the specs. This proposal adds the option.
- `repair.WithBaseline` makes the loop converge when the result adds no diagnostic to the baseline. The
  desktop panel of [03](./03-desktop-ai-panel.md) adds this option, for the same reason.

`--attempts` limits the loop, and its default is 3. A slice that does not converge gets no
suggestion. emod shows the diagnostics that the last attempt left, and the other slices still get their
suggestions.

### 4. Review by a person

The check proves that a suggestion is valid, closes its gap and agrees with the declared types. It does not
prove that the history is the right one, or that the values are realistic for the business. A person
decides that. The CLI shows a diff. The desktop app shows each suggestion as a Given-When-Then card and as a
diff ([03](./03-desktop-ai-panel.md)), so a domain expert can read it in the workshop.

### 5. Spec export (no AI)

`internal/specs` also renders the specs of a model for a test harness. The export reads only the `spec`
blocks, so it is deterministic.

**Gherkin.** emod writes one feature for each slice. Each spec becomes one scenario. The steps use a fixed
vocabulary, so one set of step definitions serves every feature:

| Spec part | Gherkin step |
| --- | --- |
| Each event in `given` | `Given the event <Event> happened` |
| `when` that names a command | `When the command <Command> arrives` |
| `when` that names an event, in an automation slice | `When the event <Event> happens` |
| No `when`, in a schedule-driven automation slice | `When the schedule runs` |
| `then = [...]` | `Then the events <Event>, <Event> are appended` |
| `then = rejected(<invariant>)` | `Then the command is rejected by <invariant>`, with the statement of the invariant as a doc string |
| `then = view(<view>)` | `Then the view <view> holds the outcome` |
| `then = command(<command>)` | `Then the automation issues the command <command>` |

A payload becomes a data table under its step, with one column for each field that the payload states. The
path of each file is `<context>/<aggregate>/<slice>.feature`, with each part in kebab case. A slice of a DCB
context has no aggregate part.

**JSON fixtures.** emod writes one document. It holds each spec with its context, aggregate, slice, name
and source line, its `given`, `when` and `then` with their payloads, and a `fields` map. The map gives the
declared type and modifier of each field of each event and command, so a harness can complete a partial
payload. A number stays a number and a string stays a string, as in `emod export --format json`.

**Drift.** `emod specs export --check` writes nothing. It renders the export and compares it with the files
on disk, and it exits with code 1 when they differ. The output is deterministic, so this check is reliable
in CI.

## Interface

### `emod ai specs`

```
emod ai specs <file> [flags]

Flags:
      --slice <name>      suggest specs for this slice only (repeatable)
      --apply             write the suggestions into the file
      --attempts <n>      repair attempts for each slice (default 3)
      --effort <level>    low|medium|high|xhigh (default medium)
      --format text|json  output format (default text)
```

- With no `--slice`, emod suggests specs for each slice that has a gap.
- With no `--apply`, emod prints the diff and the check result, and does not change the file.
- `--apply` needs a formatted file. If `emod fmt --check` fails, emod stops and asks the user to run
  `emod fmt` first. Therefore the diff holds only the new `spec` blocks.
- The exit code is 0 when each slice converged or had no gap, and 1 when a slice did not converge.
- With no AI configured, the command prints the setup steps of the foundation and exits with code 0.

`--format json` returns the same result for scripts and for the desktop app: for each slice, the gaps, the
suggested specs as structured data and as `.emod` text, the check result and the attempts; then the diff
and the token usage.

### `emod specs export`

```
emod specs export <file> [flags]

Flags:
      --format gherkin|json   output format (default gherkin)
  -o, --output <path>         a directory for gherkin, a file for json (default: stdout)
      --check                 compare with the files at --output; exit 1 when they differ
```

### Other surfaces

- **Desktop app.** "Suggest specs" on the context menu of a slice opens the AI panel with the gaps and the
  suggestions of that slice ([03](./03-desktop-ai-panel.md)).
- **MCP server.** The `suggest-specs` prompt of [01](./01-mcp-server.md) tells the host model to read the
  gaps from `emod_lint`, to write the `spec` blocks, and to check the result with `emod_validate` until it
  adds no diagnostic. The host model does the work, and emod gives the check.

## Worked Example

The slice "Borrow a Copy" of a library model has one happy-path spec. Its flow states a second outcome and
two rules:

```emod
slice "Borrow a Copy" {
  # command "BorrowCopy", event "CopyBorrowed" and event "ReservationCollected" not shown

  flow = <<-FLOW
    command -> event:    BorrowCopy -> CopyBorrowed
    command -> event:    BorrowCopy -> ReservationCollected
    command -> rejected: BorrowCopy -> OneLoanPerCopy
    command -> rejected: BorrowCopy -> FiveLoansPerMember
  FLOW

  spec "borrows a copy no one holds" {
    when = BorrowCopy({ memberId = "M-1042", copyId = "C-93204", dueOn = "2026-12-04" })
    then = [CopyBorrowed({ memberId = "M-1042", copyId = "C-93204", dueOn = "2026-12-04" })]
  }
}
```

The lint of the whole file reports three findings for the slice:

```
library.emod:51: [spec/no-rejection-path] command "BorrowCopy" is exercised by specs but none states a rejection
library.emod:80: [flow/rejection-without-spec] command "BorrowCopy" can be rejected by invariant "OneLoanPerCopy", but no spec on this slice exercises that rejection
library.emod:81: [flow/rejection-without-spec] command "BorrowCopy" can be rejected by invariant "FiveLoansPerMember", but no spec on this slice exercises that rejection
```

The gap list adds one gap that no rule reports now: no spec of the slice names `ReservationCollected`. The
facilitator asks for suggestions:

```
$ emod ai specs library.emod --slice "Borrow a Copy"
slice "Borrow a Copy": 3 gaps
  attempt 1: closes 3 gaps, adds no diagnostic

@@ -87,2 +87,20 @@ context "Lending" {
       }
+
+      spec "collects a copy the member reserved" {
+        given = [CopyReserved({ reservationId = "6f1c2a8e-4b7d-4e0a-9c3f-2d5b8a1e7f40", memberId = "M-1042", copyId = "C-93204" })]
+        when  = BorrowCopy({ memberId = "M-1042", copyId = "C-93204", dueOn = "2026-12-04" })
+        then  = [CopyBorrowed({ memberId = "M-1042", copyId = "C-93204", dueOn = "2026-12-04" }), ReservationCollected({ reservationId = "6f1c2a8e-4b7d-4e0a-9c3f-2d5b8a1e7f40", memberId = "M-1042", copyId = "C-93204" })]
+      }
+
+      spec "refuses a copy already on loan" {
+        given = [CopyBorrowed({ memberId = "M-2207", copyId = "C-93204" })]
+        when  = BorrowCopy({ memberId = "M-1042", copyId = "C-93204", dueOn = "2026-12-04" })
+        then  = rejected(OneLoanPerCopy)
+      }
+
+      spec "refuses a sixth copy to one member" {
+        given = [CopyBorrowed({ memberId = "M-1042" }), CopyBorrowed({ memberId = "M-1042" }), CopyBorrowed({ memberId = "M-1042" }), CopyBorrowed({ memberId = "M-1042" }), CopyBorrowed({ memberId = "M-1042" })]
+        when  = BorrowCopy({ memberId = "M-1042", copyId = "C-10006", dueOn = "2026-12-04" })
+        then  = rejected(FiveLoansPerMember)
+      }
     }

3 suggestions. Run again with --apply to write them.
tokens: 3,120 in / 840 out
```

These points show the parts of the design:

- The `reservationId` is a UUID in canonical form, because the event declares it as `uuid`. The validator
  rejects any other string.
- The `given` of the first spec names `CopyReserved`, an event of another slice of the same aggregate. It is
  in the boundary, so `spec/given-outside-boundary` reports nothing.
- The member and the copy agree across `given`, `when` and `then`. The check does not prove this, so the
  person reads it in the diff.
- `formatter.Format` wrote each list on one line, because that is the form of `emod fmt`.

After `--apply`, `emod validate library.emod` reports nothing and exits with code 0. The developer then
exports the specs:

```
$ emod specs export library.emod --format gherkin -o features/
wrote features/lending/loan/borrow-a-copy.feature
wrote features/lending/loan/reserve-a-copy.feature
```

One scenario of `borrow-a-copy.feature`:

```gherkin
Feature: Borrow a Copy

  Scenario: refuses a copy already on loan
    Given the event CopyBorrowed happened
      | memberId | copyId  |
      | M-2207   | C-93204 |
    When the command BorrowCopy arrives
      | memberId | copyId  | dueOn      |
      | M-1042   | C-93204 | 2026-12-04 |
    Then the command is rejected by OneLoanPerCopy
      """
      A copy is on loan to at most one member at a time
      """
```

The same spec in the JSON fixture:

```json
{
  "context": "Lending",
  "aggregate": "Loan",
  "slice": "Borrow a Copy",
  "name": "refuses a copy already on loan",
  "line": 95,
  "given": [ { "event": "CopyBorrowed", "payload": { "memberId": "M-2207", "copyId": "C-93204" } } ],
  "when": { "command": "BorrowCopy", "payload": { "memberId": "M-1042", "copyId": "C-93204", "dueOn": "2026-12-04" } },
  "then": { "rejected": { "invariant": "OneLoanPerCopy", "statement": "A copy is on loan to at most one member at a time" } }
}
```

## Implementation Plan

**Phase 1: spec export (S/M, no AI).** Add `internal/specs` with the Gherkin and JSON renderers, and the CLI
command `emod specs export` with `--check`. Test against `examples/specs_hotel.emod` and
`examples/all_patterns.emod`. This phase does not wait for the AI work.

**Phase 2: gap list (S, no AI).** Move the predicates of the spec rules into one function that the linter
and `internal/specs` call. Add `specs.Gaps` and the lint rule `spec/outcome-without-spec`, with its
description in `internal/linter/descriptions.go`.

**Phase 3: repair option (S).** Add `repair.WithAssemble` to `internal/llm/repair`.
`repair.WithBaseline` comes from the desktop panel of [03](./03-desktop-ai-panel.md). Test with
`llm.Fake`: a result that converges, a result that adds a diagnostic and then converges, and a result that
never converges.

**Phase 4: suggestions (M).** Add `internal/ai/specs` with the schema builder for each slice, the prompt
and the render step. Add the CLI command `emod ai specs` with `--apply`. Test with `llm.Fake` and canned
answers: a name outside the enum, a payload with a wrong literal, a spec that does not close its gap.

**Phase 5: other surfaces (S).** Add the "Suggest specs" action of the desktop panel with
[03](./03-desktop-ai-panel.md), and the `suggest-specs` prompt with
[01](./01-mcp-server.md).

**Phase 6: evaluation (M).** Build a corpus from the example models with their specs removed. Measure the
rate of convergence, and the share of suggestions that a person keeps with no change. Run it on the tested
tier of models before each release.

## Risks and Mitigations

- **A payload does not conform to its declared type.** The validator is part of the check, and the loop
  sends the diagnostic back. A slice that does not converge gets no suggestion.
- **A suggestion names an element that the model does not declare.** The schema restricts each name to an
  enum, the adapter checks the schema, and emod writes the block from the AST.
- **A suggestion changes other parts of the model.** It cannot. emod adds `ast.Spec` values and prints the
  file. `--apply` needs a formatted file, so the diff shows only the new blocks.
- **The history is valid but wrong.** Four loans for a five-loan rule is valid and does not break the rule.
  The check cannot see this. The person reviews the diff or the card, and the evaluation corpus measures how
  often this happens.
- **The specs agree with a wrong model.** This is by design. The semantic review of
  [02](./02-semantic-model-review.md) and the workshop find a wrong model.
- **A model in the compatible tier writes weak specs.** The check still applies to each suggestion, and the
  CLI prints the notice of the foundation for a model that is not tested.
- **The fixed Gherkin steps do not fit every harness.** The JSON fixtures carry the same data with the field
  types, for a harness that does not use Gherkin.
- **Tests depend on a live model.** No unit test calls a live model. The tests use `llm.Fake`, and the
  renderers and the gap list need no model at all.

## Open Questions

- **The cheap model.** Spec suggestions use the main model at `medium` effort. If the evaluation shows that
  `EMOD_AI_MODEL_CHEAP` keeps the same share of accepted suggestions, the cheap model becomes the default.
- **A gap tool for MCP hosts.** An `emod_spec_gaps` tool can give the host model the full gap list,
  with the gaps that no lint rule reports. Decide when the `suggest-specs` prompt ships.
- **Lint rules for views and automations.** The gap list reports a view slice or an automation slice with
  no spec, but no lint rule does. Decide if these gaps become lint rules, as `spec/outcome-without-spec`
  does.
- **Complete payloads in the export.** A `--fill` flag can add a typed placeholder for each required field
  that a payload does not state. It helps a harness that needs complete payloads, but the placeholders are
  not values from the domain.
- **One feature for each slice or for each aggregate.** One for each slice keeps the diffs small. One for
  each aggregate reads more like a test suite.
- **Templates for other harnesses.** Go table tests, Jest and RSpec are possible targets. A template
  directory over the JSON fixtures can serve them with no change to the export. Wait for a team that asks.

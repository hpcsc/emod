# Rename and Fix Suggestions in the LSP

> Part of the [emod AI proposals](./README.md). Part 1 needs no AI. Part 2 uses the
> [AI foundation](completed/00-llm-foundation.md): the `llm.Model` port, the providers and the configuration. This
> document does not specify them again.

## Problem

The linter finds a weak name and stops there. On an event called `ReservationUpdated`, `state-obsession`
tells the developer to prefer "a name that describes a specific business fact". The developer must then
find a better name. Then the developer must change the definition and every reference to it:

- the `flow` lines
- the `subscribes` lists
- the `on`, `reads` and `command` entries of automations and translations
- the `reads` of a trigger
- the `given`, `when` and `then` of specs
- the `events` of a `decides_on`

A missed reference makes the model invalid.

Two different gaps make this slow:

1. **The LSP has no rename.** `GetReferences` in `internal/lsp/references.go` finds the definition and the
   references of a command, an event, a view or an invariant, but the server does not offer
   `textDocument/rename`. The developer edits each site by hand or with a text search. This gap needs no
   AI: the set of sites is exact.
2. **The new name needs knowledge of the domain.** The linter knows that `ReservationUpdated` ends in
   `Updated`. It does not know that the slice is about new dates for a stay, so it cannot suggest
   `ReservationRescheduled`. An LLM can suggest the name. The rename of gap 1 then applies it, and the
   check of emod tests it before the developer sees it.

The rename also has a gap of its own: `GetReferences` does not collect the `events` list of a
`decides_on`. A rename of an event that a `decides_on` names leaves a reference to an event that does not
exist.

## Goals

### Part 1: rename, with no AI

- **G1:** `textDocument/prepareRename` and `textDocument/rename` for commands, events, views and invariants.
- **G2:** One `WorkspaceEdit` that changes the definition and every reference. The sites come from the same
  resolution as `GetReferences`, never from a text search.
- **G3:** The server runs `oracle.Check` on the renamed text. It refuses a rename that adds an error, for example a
  name that the model already declares.
- **G4:** The `events` of a `decides_on` become references, for rename and for find references.

### Part 2: AI suggestions on lint findings

- **G5:** Code actions on the findings of the name rules suggest up to three names. The cheap model
  (`EMOD_AI_MODEL_CHEAP`) proposes them. The rename of part 1 and `oracle.Check` test each name. The
  developer picks one.
- **G6:** Code actions on the `dcb/*` findings offer the suggestions of the DCB knowledge in
  [02](./02-semantic-model-review.md#dcb-knowledge).
- **G7:** Each action is anchored to one lint finding. The LSP `Diagnostic` carries the rule name in `code` and the
  element in `data`.
- **G8:** The editor stays responsive. The light bulb shows at once, and the server calls the model only after the
  developer picks an action.
- **G9:** With no provider configured, the AI actions are absent. Rename, the action that explains a rule, and the
  fixes with no AI stay.

## Non-Goals

- **Structural fixes.** A split of a `god-view`, a split of a `left-chair` command, and the fields of a
  `clickbait-event` add new elements to the model. They need a discussion, so they go to the chat of the AI
  panel in the desktop app ([03](./03-desktop-ai-panel.md)) or to an agent.
- **Spec fixes.** A suggested spec for `spec/*` and `flow/rejection-without-spec` findings is
  [05](./05-spec-suggestions.md).
- **A batch command in the CLI.** Each fix is the choice of a person, so no `emod lint --fix` applies AI
  names without a choice.
- **Semantic review.** The findings that lint cannot give are [02](./02-semantic-model-review.md).
- **An edit with no acceptance.** The server applies an edit only after the developer picks it.

## How It Works

### Part 1: rename

The server (`internal/lsp/server.go`) advertises its capabilities in `handleInitialize` and sends each
method to its handler in `dispatch`. Part 1 adds:

1. **The capability.** `ServerCapabilities` in `internal/lsp/protocol.go` gets
   `RenameProvider *RenameOptions` with `PrepareProvider: true`.
2. **`textDocument/prepareRename`.** The server resolves the name under the cursor with the resolution of
   `GetReferences` (`targetAt`, and `invariantLocations` for an invariant). If the name is a command, an
   event, a view or an invariant, the server returns the range of the name. If not, it returns `null`, and
   the editor tells the developer that the name cannot be renamed.
3. **`textDocument/rename`.** The server receives the new name and does these steps:
   1. It checks that the new name is an identifier. A reference is written unquoted, so a name that is not
      an identifier cannot be referenced. The server refuses such a name with an error response.
   2. It gets the definition and the references from `GetReferences`.
   3. It makes one `TextEdit` for each site, and puts them in a `WorkspaceEdit`.
   4. It applies the edits to the text in memory and runs `oracle.Check`. If the result has an error that
      the original text does not have, the server refuses the rename and returns the message of that error.
   5. It returns the `WorkspaceEdit`.

The sites for each kind of name:

| Kind | Definition | References |
| --- | --- | --- |
| Command | `command "<Name>"` | `flow` lines, the `command` of an automation or a translation, the `when` of a spec, the `then = command(...)` of a spec, rejection edges |
| Event | `event "<Name>"` | `flow` lines, `subscribes`, the `on` of an automation, the `given`, `when` and `then` of a spec, the `events` of a `decides_on` |
| View | `view "<Name>"` | The `reads` of a trigger, an automation or a translation, the `then = view(...)` of a spec |
| Invariant | An entry of `invariants` | `rejected(...)` in a spec, rejection edges, in the scope that declares the invariant |

Comments and descriptions are not references, so the rename does not change text in them.

`referencesIn` in `internal/lsp/model.go` gets the `events` of each `DecidesOnClause`, with the positions
in `EventsPos`. This also corrects find references.

The rename logic is one function that part 2 calls:

```go
// RenameEdit returns the edit that renames the name at the position, or an error that says why the
// rename is refused.
func RenameEdit(text string, line, character int, uri string, newName string) (WorkspaceEdit, error)
```

### Part 2: the anchor on the diagnostic

`ConvertDiagnostics` (`internal/lsp/diagnostics.go`) drops `diagnostic.Entry.RuleName`, because the LSP
`Diagnostic` has no field for it. A code action must know the rule and the element of a finding, so
`Diagnostic` gets two fields:

```go
type Diagnostic struct {
	Range    Range              `json:"range"`
	Severity DiagnosticSeverity `json:"severity"`
	Message  string             `json:"message"`
	Source   string             `json:"source"`
	Code     string             `json:"code,omitempty"` // diagnostic.Entry.RuleName
	Data     json.RawMessage    `json:"data,omitempty"` // the element: {"kind": "event", "name": "ReservationUpdated"}
}
```

A lint finding sits on the declaration of its element. The server resolves the name at the position of the
finding and puts its kind and name in `Data`. The client sends the diagnostic back unchanged in
`textDocument/codeAction`.

### Part 2: the flow of an AI action

```
textDocument/publishDiagnostics  server → client  finding: code = state-obsession, data = {event, ReservationUpdated}
textDocument/codeAction          client → server  the actions for this range                  [no model call]
  ← "Suggest a business name (AI)"  with the command emod.suggestName
  ← "Explain state-obsession"       with the command emod.explainRule                         [no AI]
workspace/executeCommand         client → server  emod.suggestName {uri, version, rule, element}
  → $/progress begin
  → the cheap model returns up to three names                                                  [model call]
  → for each name: RenameEdit, then oracle.Check
  → $/progress end
window/showMessageRequest        server → client  one action for each name that passed the check
  ← the name that the developer picked
workspace/applyEdit              server → client  the WorkspaceEdit of RenameEdit for that name
```

The action uses a command and a message request, not `codeAction/resolve`. A resolve returns one edit for
one action, but the developer must see more than one name to choose. A model call in
`textDocument/codeAction` makes the light bulb wait. With the command, the server calls the model only after
the developer picks the action, and still shows each name that passed.

If the client does not declare `window.showMessage.messageActionItem` in its capabilities, the server sends
the names in a `window/showMessage`. The developer then renames with part 1. If the version of the document
changes before the model answers, the server drops the result and tells the developer to run the action
again.

### Part 2: the prompt and the check

`internal/ai/rename` holds the AI part:

```go
type Candidate struct {
	Name      string `json:"name"`
	Rationale string `json:"rationale"`
}

func Suggest(ctx context.Context, model llm.Model, source string, finding *diagnostic.Entry) ([]Candidate, llm.Usage, error)
```

The prompt has:

- The description of the rule from `linter.RuleDescription`, the same text that `emod lint --explain`
  prints.
- The element, its fields, and its slice: the slice description and the names of the other elements in it.
- The names of the context and the aggregate that hold the element.
- The names that the model already declares, so the model does not suggest a name in use.

The prompt does not hold the whole model. The anchor gives the scope, which keeps the call small. The call
uses `EMOD_AI_MODEL_CHEAP` at effort `low`, and a JSON Schema in `GenerateRequest.Schema` for a list of up
to three candidates.

The server tests each candidate with `RenameEdit` and `oracle.Check`, and puts it in one of three classes:

| Class | Condition | What the developer sees |
| --- | --- | --- |
| Clean | The finding is gone, and the check reports nothing new | The name, first in the list |
| Trade | The finding is gone, but another lint finding appears | The name, after the clean names, with the new rule: "(reports property-sourcing)" |
| Broken | The check reports an error, or the finding stays | Nothing. The server drops the name. |

If no candidate passes, the server shows "No suggested name passed the check" and the developer renames by
hand.

### Part 2: DCB actions

The four `dcb/*` findings that the DCB knowledge can answer get an AI action. The command
`emod.suggestDCB` calls `dcb.Suggest` of [02](./02-semantic-model-review.md#dcb-knowledge) with `EMOD_AI_MODEL` at
effort `medium`, because a boundary is harder to choose than a name. The DCB knowledge renders and checks
each suggestion. The server shows the rationale of each suggestion in a `window/showMessageRequest`, with one
action for each suggestion and one to cancel. After the developer picks one, the server sends the line
difference between the current text and the rendered text in a `workspace/applyEdit`.

### The actions for each rule

| Rule | Action | Edit | Model |
| --- | --- | --- | --- |
| `state-obsession` | Suggest a business name (AI) | Rename | Cheap |
| `property-sourcing` | Suggest a business name (AI) | Rename | Cheap |
| `command-in-disguise` | Suggest a name for the fact (AI) | Rename | Cheap |
| `command-past-tense` | Suggest an imperative name (AI) | Rename | Cheap |
| `view-naming` | Rename to `<Name>View` | Rename | None. The fix is exact. |
| `dcb/untagged-event` | Suggest tags (AI) | Add a `tags` block | Main, through the DCB knowledge of 02 |
| `dcb/query-too-broad` | Suggest a narrower query (AI) | Change the `decides_on` | Main, through the DCB knowledge of 02 |
| `dcb/single-tag-everywhere` | Suggest a second tag key (AI) | Add a tag key and change a predicate | Main, through the DCB knowledge of 02 |
| `dcb/orphan-tag-key` | Suggest a route or a removal (AI) | Change a predicate, or remove the tag key | Main, through the DCB knowledge of 02 |
| Each rule | Explain `<rule>` | None. The server shows `linter.RuleDescription` in a `window/showMessage`. | None |

## Interface

The developer edits an `.emod` file. The linter draws a warning under the event name, and a light bulb
shows next to it:

```
event "ReservationUpdated"
  ⚠ [state-obsession] event "ReservationUpdated" uses a generic state-change suffix;
    prefer a name that describes a specific business fact
  💡 Suggest a business name (AI)
     Explain state-obsession
```

Each AI action ends in `(AI)`, so the developer always knows which actions come from a model. After the
developer picks the action, the editor shows the progress, then the names that passed the check:

```
Rename "ReservationUpdated" to:
  [ReservationRescheduled]  [StayRescheduled]  [ReservationDatesChanged (reports property-sourcing)]  [Cancel]
```

The rename of a name with no AI is the usual rename of the editor, for example `F2` in VS Code.

## Worked Example

This model has one lint finding on purpose:

```emod
emod = 1

model "Hotel Reservation" {
}

context "Reservations" {
  aggregate "Reservation" {
    slice "Change Dates" {
      command "ChangeDates" {
        fields {
          reservationId = required(string)
          checkIn       = required(date)
          checkOut      = required(date)
        }
      }

      event "ReservationUpdated" {
        fields {
          reservationId = required(string)
          checkIn       = required(date)
          checkOut      = required(date)
        }
      }

      flow = <<-FLOW
        command -> event:    ChangeDates -> ReservationUpdated
      FLOW
    }

    slice "View Stays" {
      view "UpcomingStaysView" {
        subscribes = [ReservationUpdated]

        fields {
          reservationId = required(string)
          checkIn       = required(date)
        }
      }
    }
  }
}
```

1. **Finding.** `emod lint` reports
   `hotel.emod:17: [state-obsession] event "ReservationUpdated" uses a generic state-change suffix`. The
   server publishes it with `code: "state-obsession"` and `data: {"kind": "event", "name":
   "ReservationUpdated"}`.
2. **Code action, with no model call.** The developer opens the light bulb. The server returns "Suggest a
   business name (AI)" and "Explain state-obsession".
3. **Model call.** The developer picks the AI action. The server sends the rule description, the event, its
   fields, the slice "Change Dates" with the command `ChangeDates`, and the declared names. The cheap model
   returns:

   ```json
   {
     "candidates": [
       { "name": "ReservationRescheduled", "rationale": "the command changes the dates of a stay" },
       { "name": "StayRescheduled", "rationale": "the guest moves the stay to new dates" },
       { "name": "ReservationDatesChanged", "rationale": "the dates of the reservation change" }
     ]
   }
   ```

4. **Check.** For each name, the server makes the rename edit and runs `oracle.Check`:
   - `ReservationRescheduled`: the finding is gone, and nothing new appears. Clean.
   - `StayRescheduled`: the finding is gone, and nothing new appears. Clean.
   - `ReservationDatesChanged`: the finding is gone, but `property-sourcing` reports the name as
     `<Aggregate><Field>Changed`. Trade.
5. **Choice.** The editor shows the three names, with the trade last and labelled. The developer picks
   `ReservationRescheduled`.
6. **Edit.** The server sends one `WorkspaceEdit` that changes the definition, the `flow` line and the
   `subscribes` list. The sites came from `GetReferences`, not from the model:

   ```diff
   -      event "ReservationUpdated" {
   +      event "ReservationRescheduled" {
   ...
   -        command -> event:    ChangeDates -> ReservationUpdated
   +        command -> event:    ChangeDates -> ReservationRescheduled
   ...
   -        subscribes = [ReservationUpdated]
   +        subscribes = [ReservationRescheduled]
   ```

The lint finding is gone, and the check reports nothing. For a DCB finding, see the worked example of
[02](./02-semantic-model-review.md#dcb-knowledge), which shows three DCB actions in sequence.

## Implementation Plan

**Phase 1: rename, with no AI (M).**

- `RenameProvider` with `PrepareProvider`, and the `textDocument/prepareRename` and `textDocument/rename`
  cases in `dispatch`.
- `RenameEdit`: the identifier check, the sites from `GetReferences`, the `WorkspaceEdit`, and the check
  with `oracle.Check`.
- The `events` of a `decides_on` in `referencesIn`.
- Tests for each kind of name, for a name in use, for a name that is not an identifier, and for an event in a
  `decides_on`.

**Phase 2: the anchor, with no AI (S).**

- `Code` and `Data` on `Diagnostic`, filled in `ConvertDiagnostics`.
- `CodeActionProvider` in `ServerCapabilities`, and the `textDocument/codeAction` case in `dispatch`.
- The "Explain `<rule>`" action and the exact `view-naming` action.

This phase ships with no provider and proves the round trip of the anchor.

**Phase 3: name suggestions (M).**

- `internal/ai/rename`: the prompt, the schema and `Suggest`.
- The command `emod.suggestName`, the progress, the check of each candidate, the message request, the
  fallback, the version check and `workspace/applyEdit`.
- Tests against `llm.Fake` with canned candidates: clean, trade and broken names, a list with no name that passes, and a
  document that changes before the answer.

**Phase 4: DCB actions (S, after the DCB knowledge of 02 is built).**

- The command `emod.suggestDCB` and the four actions, through `dcb.Suggest` and `dcb.Render` of
  [02](./02-semantic-model-review.md#dcb-knowledge).

All tests follow the Go test rules of the repository: one umbrella `Test{Type}` for each type, `t.Run`
groups for each operation, scenario names that describe behaviour, `testify/require`, and fresh fixtures in
each leaf subtest.

## Risks and Mitigations

- **A rename misses a reference.** The sites come from the AST, not from text. A site that the AST does not
  link is not a reference, and after the rename the validator reports it as an unknown name. The check in
  step 4 of the rename then refuses the rename, so the developer never gets a broken model.
- **The editor waits on the model.** The light bulb needs no model call. The model call starts only after
  the developer picks an action, with progress, the cheap model and effort `low`.
- **Cost from frequent calls.** The prompt holds only the scope of the element. The server keeps the
  candidates for each finding and document version, so a second request costs nothing. The cost comes from
  `Response.Usage`.
- **A different name on each run.** A suggestion is a choice for the developer, so this is acceptable. The
  stored candidates keep the list stable while the document version stays the same.
- **An editor with no message request.** The fallback shows the names in a message, and the developer
  renames with part 1.
- **A weak model on the compatible tier.** A small model can suggest poor names. The check removes broken
  names, and the developer picks or cancels.
- **AI becomes a hard dependency.** Phases 1 and 2 need no provider. With no provider configured, the server
  offers no AI action, and every other feature works as before.

## Open Questions

- **Rename of other kinds.** Contexts, actors, aggregates, slices, automations and translations are not in
  the rename. A context is referenced by the `target` of an automation, and an actor by a trigger. They can
  join when find references covers them.
- **Models in more than one file.** `GetReferences` reads one document. A `WorkspaceEdit` can change many
  files, so the rename can grow when a model spans files.
- **One rename for every surface.** The inline rename of the web viewer and the desktop app changes one
  node. If the reference resolution moves out of `internal/lsp` into its own package, the desktop app and
  the MCP server can rename with the same code.
- **Names for review findings.** When the findings of [02](./02-semantic-model-review.md) reach the LSP,
  `ai/weak-naming` and `ai/hollow-event` can get the same "Suggest a business name (AI)" action.
- **DCB edits on a file that is not formatted.** The DCB knowledge renders a suggestion with `formatter.Format`, so the
  edit also formats the rest of the file. The action can refuse an unformatted file, or the title can say
  that it also formats the file.

# AI Panel in the Desktop App

> Part of the [emod AI proposals](./README.md). Uses the [AI foundation](completed/00-llm-foundation.md): the
> `llm.Model` port, the providers, the configuration and the repair loop. The panel is the desktop surface
> of [02](./02-semantic-model-review.md), [04](./04-model-from-conversation.md),
> [05](./05-spec-suggestions.md) and [02](./02-semantic-model-review.md#dcb-knowledge), and it uses the exact queries
> of the [model queries proposal](../model-queries-proposal.md).

## Problem

The desktop app is the workspace of a workshop. The facilitator opens the `.emod` file and shares the
screen, and the group reads and changes the diagram together. The app has a source panel that shows the
diagnostics as the user types, direct edits on the diagram (drag, rename, add a slice, a command or an
event, connect, delete), native Open and Save, and a file watch that reloads the file when it changes on
disk.

These edits are local: one node, one field or one arrow at a time. The changes that a workshop asks for are
structural:

- "A guest can extend a hold once, by one more day."
- "Split this view: it reads too many events."
- "Rename this event to the word that the analysts use."
- "Convert this context to DCB."

Each of these changes touches many elements in many slices, and the result must stay valid. A person who
uses an agent such as Claude Code gets this through the MCP server ([01](./01-mcp-server.md)): the agent
edits the file, and the desktop app reloads it. But domain experts, and many facilitators, do not use an
agent. They have no AI in the place where they work.

The web viewer runs in a browser, which cannot hold a key safely. The desktop app is a Go process: it links
the emod pipeline, and it can hold `llm.Model` directly, as the CLI does. Therefore the AI panel is a part of
the desktop app, and the web viewer stays free of AI.

## Goals

- **G1:** One AI panel in the desktop app, with these parts, in this order of delivery:
  1. The settings dialog for the provider ([00](completed/00-llm-foundation.md)).
  2. The Review button ([02](./02-semantic-model-review.md)).
  3. The chat, with an edit mode (repair loop, diff, accept, reject, undo) and an ask mode (answers from the
     JSON export and the exact queries of the [model queries proposal](../model-queries-proposal.md)).
  4. "New model from description" ([04](./04-model-from-conversation.md)).
  5. "Suggest specs" on a slice ([05](./05-spec-suggestions.md)).
  6. "Convert to DCB" on a context ([02](./02-semantic-model-review.md#dcb-knowledge)).
- **G2:** The model call, the repair loop and the check run in the Go process of the desktop app. There is no HTTP
  backend, and no key goes to the page.
- **G3:** No AI code ships in the web viewer or in the viewer that the CLI embeds.
- **G4:** An accepted change goes into the source panel as an unsaved edit. Save, the marker for unsaved work, the
  questions on close and on quit, and the file watch work as they do for an edit that the user typed. The
  AI never writes the file.
- **G5:** Each proposed edit passes the check against the baseline before a person can accept it.
- **G6:** With no AI configured, the panel is hidden. Only the menu item AI ▸ Settings… stays.
- **G7:** The header of the panel shows the provider, the model and the host of the endpoint.
- **G8:** The user can cancel a call that runs.

## Non-Goals

- AI in the static web viewer. AI in the viewer of `emod diagram --serve` is an open question of
  [00](completed/00-llm-foundation.md).
- An agent that edits in many steps with no accept between them. Each change is one proposal and one
  decision.
- A general chat. The chat is about the open model only.
- Collaboration between users, or state on a server. The model lives in the window and in the file.
- A replacement for the direct edits on the diagram. The panel adds to them.
- New logic for the review, the generation or the specs. That logic lives in
  [02](./02-semantic-model-review.md), [04](./04-model-from-conversation.md) and
  [05](./05-spec-suggestions.md). This proposal gives it a surface in the desktop app.

## How It Works

### Where each step runs

| Step | Where | Why |
| --- | --- | --- |
| Hold the text of the model and the selection | The page: `store.dom.sourceInput` and `store.interaction.selectedNodeId` | The panel text is the working copy of the user |
| Call the model, run the repair loop and the check | Go: `desktop.AIService` | The credentials stay in the Go process |
| Compute the diff and the list of changes | Go | The diff and the AST are in Go |
| Show the diff; accept, reject, undo | The page: the AI panel | This is state of the screen only |
| Apply an accepted change | The page: the same path as a Render | Save, the marker and the file watch then behave as for a typed edit |
| Write the file | File ▸ Save, through `saveFile` and `FileService.Write` | The user decides when the file changes |

### The Go side

A new service, `desktop.AIService`, lives in `internal/desktop/ai_service.go`. `cmd/emod-desktop/main.go`
registers it with `application.NewService`, as it registers `ModelService` and `FileService`.

- **Contract.** Each method takes a JSON string and returns a JSON string, with the `{"error": "..."}`
  envelope that `ModelService` uses. Therefore the page reads each answer in the same way.
- **Cancel.** Each method that calls a model takes `context.Context` as its first parameter. Wails v3 binds
  such a method with a context, and its runtime returns a `CancellablePromise`. When the user clicks
  Cancel, the page cancels the promise, Wails cancels the context, and the adapter stops the HTTP request.
- **No GUI framework.** `internal/desktop` imports no GUI framework, so its tests run everywhere. Progress
  goes through an `AIProgressListener` interface. `cmd/emod-desktop` implements it with
  `window.EmitEvent("ai:progress", ...)`, as `openFileEvents` implements `OpenFileListener`.
- **Configuration.** Each call runs `config.Load(config.Flags{})`. A change in the settings dialog applies at
  the next call, and the CLI and the app read the same file.

| Method | What it calls | Proposal |
| --- | --- | --- |
| `Status()` | `config.Load` and the source of the credential, as `emod ai status` does. It returns the provider, the model, the host of the endpoint, the support tier and the source of the credential, or "not configured". It makes no model call. | [00](completed/00-llm-foundation.md) |
| `Settings()`, `SaveSettings(request)` | Read and write the settings in `~/.config/emod/config`, and the API key in the key store of [Key storage](#key-storage) | [00](completed/00-llm-foundation.md) |
| `CheckSettings(ctx, request)` | One small call with the settings in the dialog, as `emod ai` does | [00](completed/00-llm-foundation.md) |
| `Review(ctx, request)` | `internal/ai/review` | [02](./02-semantic-model-review.md) |
| `Edit(ctx, request)` | `internal/ai/edit`, which calls `repair.Generate` | This proposal |
| `Ask(ctx, request)` | `query.References`, `query.Trace` and one model call | [Exact model queries](../model-queries-proposal.md) |
| `GenerateQuestions(ctx, request)`, `GenerateDraft(ctx, request)` | `internal/ai/generate` | [04](./04-model-from-conversation.md) |
| `SuggestSpecs(ctx, request)`, `ApplySpecs(request)` | `internal/ai/specs`. `ApplySpecs` makes no model call. | [05](./05-spec-suggestions.md) |

### The page side

- **The platform seam.** The platform contract in `internal/frontend/static/platform.js` gets one more
  function, `mountAiPanel(store)`. `platform.browser.js` exports a function that does nothing.
  `platform.desktop.js` imports the panel module and mounts the panel. `viewer.js` calls `mountAiPanel`
  once, at start.
- **The panel module.** `ai-panel.js` lives in `internal/frontend/desktop/`, next to `platform.desktop.js`.
  `build:web` copies `internal/frontend/static/` and `embed.go` embeds it, so a module in that directory
  ships to the web viewer and into the CLI binary. The `build:desktop` task copies `ai-panel.js` into the
  frontend of the desktop app, with `platform.desktop.js`.
- **The guards.** Each call to `AIService` goes through a function in `platform.desktop.js`, and
  `platform.desktop.js` subscribes to `ai:progress` and to the AI menu events. `binding_names_test.go` and
  `event_names_test.go` read that file, so they guard the new service and the new events with no change.
- **The context menu.** `ctx-actions.js` gets a function that adds an entry to the context menu of a kind
  of node. The AI panel adds "Suggest Specs" to a slice and "Convert to DCB" to a context in aggregate
  mode. The web viewer adds no entry.

### The edit flow

1. The user types an instruction. The user can select a node first. The panel then sends the kind and the
   name of the selected element as the focus.
2. The panel sends the instruction, the panel text, the file name and the focus to `AIService.Edit`. It
   records the text that it sent.
3. `internal/ai/edit` computes the **baseline**: `oracle.Check` on the text that it got. It builds the
   prompt from the grounding of [04](./04-model-from-conversation.md) (the CUE schema, the rule descriptions
   in `internal/linter/descriptions.go`, the example models), the JSON export of the model from
   `pipeline.RunPipelineExportJSON`, the source, the focus and the instruction. For a model with a DCB
   context, the prompt also holds the DCB knowledge of [02](./02-semantic-model-review.md#dcb-knowledge).
4. It runs `repair.Generate` with `repair.WithBaseline`, an option of the
   [repair loop](completed/00-llm-foundation.md#the-repair-loop) that this proposal adds. The loop converges when the result adds no diagnostic to the
   baseline. Each attempt sends one `ai:progress` event.
5. `AIService.Edit` returns the new source, a unified diff, the list of changes, the diagnostics that the
   change adds and the diagnostics that it removes, the notes of the model, and the usage.
6. The panel shows the notes, the list of changes, the diff and the check result. It does not apply the
   change yet.
7. **Accept.** The panel emits `bus.emit('ai:apply', { source })`. `viewer.js` handles the event with
   `renderPanelSource(source)`, the function that a Render uses. The function writes the text into the
   source panel, parses it, redraws the diagram, shows the diagnostics and calls `reportModified`. The window
   now shows unsaved work. File ▸ Save writes the file.
8. **Reject.** The panel discards the result. Nothing changes.
9. **Undo.** The panel keeps a stack of the texts before each accepted change. Undo writes the text before
   the last change through the same path. Undo is available while the source panel holds the text that the
   last accepted change wrote.

**The list of changes.** emod computes it from the AST of the old and the new source, with no AI: the
contexts, the slices, the commands, the events, the views, the automations, the invariants and the specs
that the change adds, removes or changes. The list shows a person what changed before the person reads the
diff. A change outside the focus is marked, because it is the change that a person does not expect.

**The notes.** The prompt asks the model for its assumptions and its open questions, in text outside the
fenced source. `repair.Outcome` gets an `Answer` field with the text of the last answer, so that
`internal/ai/edit` can read the notes. The notes are the questions for the workshop.

**The diff.** A new package, `internal/textdiff`, gives a unified line diff. It is a small Myers diff with no
new dependency. `emod ai specs` in [05](./05-spec-suggestions.md) uses it too.

**Accept and new findings.** A user can ask for a change that a lint rule does not like, for example a view
that reads five events. The loop first tries to remove each new diagnostic. After the last attempt:

| Result | What the panel shows |
| --- | --- |
| No new diagnostic | "Passes the check", and Accept |
| New warnings or info findings, no new error | The new findings, and "Accept with 1 new finding" |
| A new error | The new errors, Try Again, and no Accept |

**Text that changed.** The result applies to the text that the panel sent. If the panel text is different
when the result arrives, because the user typed, the file watch reloaded the file, or the user opened
another file, the panel marks the result as out of date. Accept is then off, and "Run again on the current
text" sends the instruction again.

**Diagram edits.** The AI works on the panel text. While `store.conflict` is set, or while the diagram holds
an edit that the panel text does not hold, Send is off. The panel tells the user to resolve the edit first.
The viewer puts the state of an unapplied diagram edit on the store, so that the panel can read it.

### The ask mode

A question uses `AIService.Ask`. The Go side exports the model as JSON, and it runs `query.References` and
`query.Trace` from the [model queries proposal](../model-queries-proposal.md) for each name of the model that the question holds.
The prompt holds these exact answers as the facts. The model only orders and phrases them.

The answer names model elements. The panel shows each name as a link: a click selects the node and starts
the highlight of the [model queries proposal](../model-queries-proposal.md). Each name that the model does not declare gets the
mark "not in the model". Nothing in ask mode changes the text.

### The Review button

The Review button calls `AIService.Review`, which runs `internal/ai/review`
([02](./02-semantic-model-review.md)). The findings show in the diagnostics panel next to the lint
findings, with an `AI` label and the confidence. The diagram marks the node of each element that has a
finding. The facilitator can reject a finding, or send it to the chat as an instruction. After a change of
the text, the panel finds the element of each finding in the new parse, moves the finding to the element,
and drops a finding whose element is gone.

### New model from description

File ▸ New Model from Description… opens the panel in its generation mode. If the window holds unsaved
work, the app first asks the question that it asks before an Open, through `resolveUnsavedEdits`. Then:

1. The user types the description. `AIService.GenerateQuestions` returns the questions of
   [04](./04-model-from-conversation.md). The panel shows them as a form with option buttons.
2. `AIService.GenerateDraft` returns the draft and its assumptions, after the repair loop.
3. The panel shows the draft as a diagram, with the assumptions beside it.
4. Accept opens the draft as a new model with no file: `renderPanelSource(source, null)`. File ▸ Save then
   asks where to write it.

### Suggest specs

"Suggest Specs" on a slice calls `AIService.SuggestSpecs` for that slice
([05](./05-spec-suggestions.md)). The panel shows the gaps of the slice, and each suggestion as a
Given-When-Then card, so a domain expert can read it in the workshop. Each card has a check box. Accept
calls `AIService.ApplySpecs` with the selected suggestions. `ApplySpecs` makes no model call: it inserts
the suggestions, prints the file with `formatter.Format` and runs the check. The panel then applies the
source through `ai:apply`.

### Convert to DCB

"Convert to DCB" on a context in aggregate mode sends an edit with the instruction "convert context
`<name>` to mode dcb" and the context as the focus. The prompt holds the DCB knowledge of
[02](./02-semantic-model-review.md#dcb-knowledge). The check holds the `dcb/*` rules and the mode rules
`dcb-in-aggregate-mode` and `aggregate-in-dcb-mode`, so the loop keeps the tags and the `decides_on` clauses
in shape. The result is a normal edit: a list of changes, a diff, Accept and Reject.

### The settings dialog and the header

AI ▸ Settings… opens the dialog. It holds the keys of the configuration in [00](completed/00-llm-foundation.md):
the provider (`anthropic` or `openai`), the API key, the endpoint, the model, the cheap model and the
effort. A field shows only for the providers that use it. The Check button calls
`AIService.CheckSettings`, which sends one small request and shows the result and its tokens. Save writes
the settings to `~/.config/emod/config` with mode `0600`, and the API key to the key store of
[Key storage](#key-storage). The file never holds the key.

An app that starts from the Finder or the Dock has no environment of a shell, so the file is the usual
source of the settings, and the key store is the usual source of the API key. The app does not get
`ANTHROPIC_API_KEY` from the shell. When an environment variable sets a key, it wins over the file. The
dialog then shows that field as read-only, with the name of the variable.

Below the fields, the dialog shows the source of the API key from `AIService.Status`, but never the key.
Examples are `the OS credential store`, `EMOD_AI_API_KEY` and `ANTHROPIC_API_KEY`. The source tells the
user which key a call uses, also when an environment variable wins over the dialog.

The header of the panel shows the provider, the model and the host of the endpoint, for example
`openai · llama3.1 · localhost:11434`. Before the first call of a session, the panel states that the model
source goes to that host. For a model outside the tested tier, the header shows the notice of
[00](completed/00-llm-foundation.md). The header also shows the tokens of the session.

### Key storage

The API key of the `anthropic` and `openai` providers goes to the OS credential store, through
`github.com/zalando/go-keyring`:

| Platform | Store |
| --- | --- |
| macOS | The Keychain, through `/usr/bin/security`, so emod needs no cgo for it |
| Windows | The Credential Manager |
| Linux | The Secret Service, through D-Bus |

`go-keyring` is already in `go.mod`, through the `wails3` command of Wails v3, so the key store adds no new
module.

A Linux machine with no desktop often has no Secret Service. When the OS credential store fails, the key
goes to `~/.config/emod/credentials.json` with mode `0600`, and the dialog names that file after Save.
When the OS credential store works again, the next Save removes the copy from the file.

**One key for each provider and host.** The store keeps each key under the provider and the host of the
endpoint, for example `anthropic/api.anthropic.com` or `openai/openrouter.ai`. A user who changes the
endpoint to a proxy or to a local server does not send the key of the old host to the new host. For a new
host, the dialog asks for a key. A local server, such as Ollama, needs no key.

**The order of the sources.** emod takes the API key from the first of these that gives one:

1. `EMOD_AI_API_KEY`
2. `ANTHROPIC_API_KEY` or `OPENAI_API_KEY`, for the chosen provider
3. The key store, for the provider and the host of the endpoint

One configuration serves the CLI, the LSP and the desktop app, so the CLI and the LSP read the same key
store. A key that a user saves in the dialog therefore also works in `emod ai`.

**Clear.** The Clear button removes the key of the provider and the host from the OS credential store and
from the file.

**Tests.** The Go tests replace the OS credential store with `keyring.MockInit` and
`keyring.MockInitWithError`, so no test changes the keychain of the developer. A test that runs the binary
sets `EMOD_AI_KEY_STORE=file`, which makes emod use only the file.

## Interface

### `AIService.Edit`

```jsonc
// request
{
  "instruction": "A guest can extend a hold once, by one more day",
  "source": "emod = 1\n...",
  "filename": "hotel.emod",
  "focus": { "kind": "event", "name": "RoomHeld" }   // optional
}
```

```jsonc
// response
{
  "source": "emod = 1\n...",
  "diff": "@@ -18,2 +18,3 @@ ...",
  "changes": [
    { "change": "added", "kind": "invariant", "name": "HoldExtendedOnce", "inFocus": true },
    { "change": "added", "kind": "slice", "name": "Extend a Hold", "inFocus": true }
  ],
  "added": [],        // diagnostics that the change adds
  "removed": [],      // diagnostics that the change removes
  "converged": true,
  "attempts": 2,
  "notes": "UnreleasedHoldsView does not subscribe to HoldExtended ...",
  "usage": { "input_tokens": 14210, "output_tokens": 5120 }
}
```

### Events and page functions

| Name | Kind | Meaning |
| --- | --- | --- |
| `ai:progress` | Event from Go | `{ call, attempt, maxAttempts, added }` for each attempt of a loop |
| `ai:panel-requested`, `ai:review-requested`, `ai:settings-requested`, `ai:new-model-requested`, `ai:undo-requested` | Events from the menu | The menu items of the AI menu and of File |
| `ai:apply` | Event on `bus.js` | `{ source }`. `viewer.js` applies it with `renderPanelSource`. |
| `mountAiPanel(store)` | Platform function | Mounts the panel in the desktop app; does nothing in the web viewer |

### Menus and keys

| Where | Item |
| --- | --- |
| AI menu | Show AI Panel, Review Model, Undo AI Change, Settings… |
| File menu | New Model from Description… |
| Context menu of a slice | Suggest Specs |
| Context menu of a context in aggregate mode | Convert to DCB |
| In the panel | Cmd-Enter or Ctrl-Enter sends; Escape cancels the call that runs |

## Worked Example

A facilitator has `hotel.emod` open in the desktop app. It is `examples/specs_hotel.emod`, and the check
reports nothing for it. The domain expert says: "A guest can extend a hold once, by one more day." The
facilitator selects the event `RoomHeld` and types the sentence into the panel.

```
anthropic · claude-opus-4-8 · api.anthropic.com
attempt 1 of 4: adds 2 findings
  [spec/no-rejection-path] command "ExtendHold" is exercised by specs but none states a rejection
  [flow/rejection-without-spec] command "ExtendHold" can be rejected by invariant "HoldExtendedOnce", but no spec on this slice exercises that rejection
attempt 2 of 4: passes the check
```

The first attempt stated the rule in the flow but wrote no spec for it. The loop sent the two new findings
back, and the second attempt added the spec. The panel then shows the result:

```
Changes (all in focus)
  + invariant HoldExtendedOnce
  + slice "Extend a Hold": command ExtendHold, event HoldExtended, 2 specs

Notes
  UnreleasedHoldsView does not subscribe to HoldExtended, so ExpiredHoldReleaser
  still releases a hold 24 hours after RoomHeld. Decide if the expiry starts
  again after an extension.

Passes the check.                         [ Accept ]  [ Reject ]
```

The diff:

```diff
@@ -18,2 +18,3 @@ context "Reservations" {
       HoldReleasedOnce          = "A hold is released exactly once"
+      HoldExtendedOnce          = "A hold is extended at most once"
     }
@@ -102,2 +103,39 @@ context "Reservations" {

+    slice "Extend a Hold" {
+      command "ExtendHold" {
+        description = "Guest keeps a held room for one more day"
+
+        fields {
+          holdId = required(string)
+        }
+      }
+
+      event "HoldExtended" {
+        type = "com.hotel.reservations.hold-extended"
+
+        fields {
+          holdId     = required(string)
+          roomId     = required(string)
+          extendedAt = required(timestamp)
+        }
+      }
+
+      flow = <<-FLOW
+        command -> event:    ExtendHold -> HoldExtended
+        command -> rejected: ExtendHold -> HoldExtendedOnce
+      FLOW
+
+      spec "extends an outstanding hold" {
+        given = [RoomHeld({ holdId = "H-3391" })]
+        when  = ExtendHold({ holdId = "H-3391" })
+        then  = [HoldExtended({ holdId = "H-3391", roomId = "204", extendedAt = "2026-07-31T09:00:00Z" })]
+      }
+
+      spec "refuses a second extension" {
+        given = [RoomHeld({ holdId = "H-3391" }), HoldExtended({ holdId = "H-3391" })]
+        when  = ExtendHold({ holdId = "H-3391" })
+        then  = rejected(HoldExtendedOnce)
+      }
+    }
+
     slice "Release Hold" {
```

The group reads the new slice on the diagram and accepts. The title bar now shows unsaved work. The note is
a real question for the domain expert, who says that the expiry starts again. The facilitator sends that
as the next instruction, and the panel proposes a change to `UnreleasedHoldsView` and to the automation.
After the group accepts it, the facilitator clicks Review Model, then saves with File ▸ Save.

## Implementation Plan

**Phase 1: settings and status (S).** `desktop.AIService` with `Status`, `Settings`, `SaveSettings` and
`CheckSettings`. The key store of [Key storage](#key-storage). The settings dialog, the AI menu, the
platform function `mountAiPanel`, the panel header, and the `build:desktop` step that copies
`ai-panel.js`. `Status` reuses the work of `emod ai status`. The phase depends on the provider work of
[00](completed/00-llm-foundation.md) for the `openai` provider.

**Phase 2: review (S).** `AIService.Review` and the Review button, on top of `internal/ai/review` from
[02](./02-semantic-model-review.md). Findings in the diagnostics panel and marks on the diagram.

**Phase 3: edit and ask (L).** `internal/ai/edit`, `repair.WithBaseline`, `repair.Outcome.Answer`,
`internal/textdiff`, the list of changes from the AST, `AIService.Edit` and `AIService.Ask`. In the page:
the chat, the diff view, Accept, Reject, Undo, Cancel, `ai:apply` in `viewer.js`, and the out-of-date
check. Tests in Go use `llm.Fake` for the paths that converge, that add a warning and that never
converge. Tests in vitest cover the apply path, the undo stack and the out-of-date check, with a fake
platform.

**Phase 4: new model from description (S).** `AIService.GenerateQuestions` and `GenerateDraft` on top of
`internal/ai/generate` from [04](./04-model-from-conversation.md). The question form and the draft view.

**Phase 5: suggest specs (S).** The context menu function in `ctx-actions.js`, `AIService.SuggestSpecs` and
`AIService.ApplySpecs` on top of `internal/ai/specs` from [05](./05-spec-suggestions.md), and the
Given-When-Then cards.

**Phase 6: convert to DCB (S).** The context menu entry, and the DCB knowledge of
[02](./02-semantic-model-review.md#dcb-knowledge) in the edit prompt.

## Risks and Mitigations

- **The model changes parts that the user did not ask about.** The list of changes marks each change
  outside the focus, and the diff shows each line. The prompt asks for the smallest change. Nothing applies
  until a person accepts.
- **A valid change is wrong for the domain.** The check cannot see this. The notes state the assumptions of
  the model, the group reads the diagram before Accept, and the Review button gives a second opinion.
- **A file with findings before the call never converges.** The baseline makes the loop compare against the
  file as it was, so the findings that the file already had do not count.
- **The result applies to text that changed.** The panel records the text that it sent and turns off Accept
  when the panel text differs.
- **The AI and a diagram edit change the model at the same time.** Send is off while the diagram holds an
  edit that the panel text does not hold.
- **A long call blocks the workshop.** Each call shows the progress of each attempt, and Cancel stops the
  HTTP request through the context. The binding of Wails v3 passes the cancel of the promise to the
  context. The implementation must confirm this on each upgrade of Wails, because Wails v3 is a beta.
- **The key leaks.** The key stays in the Go process and in the OS credential store, or in a file with mode
  `0600` on a machine with no credential store. The page never gets it: `Settings()` returns only whether
  a key is set, and its source. The key belongs to one provider and one host, so a change of the endpoint
  does not send it to another server.
- **The model source goes to a remote provider.** The header names the host, and the panel states it
  before the first call of a session. A team with confidential models uses a local server through the
  `openai` provider.
- **AI code reaches the web viewer.** The panel module lives outside `internal/frontend/static/`, and the
  browser implementation of `mountAiPanel` does nothing. A unit test can check that `build:web` copies no
  file from `internal/frontend/desktop/`.

## Open Questions

- **A command that prints the key.** A setting such as `EMOD_AI_API_KEY_COMMAND` can run a command, for
  example `op read op://Private/emod/api-key`, and use its output as the key. Then the key stays in a
  password manager. The key store covers most users, so add the command when a user asks for it. The
  command must come only from `~/.config/emod/config`, never from a file in a repository.
- **A CLI for edits.** `internal/ai/edit` has no surface in the CLI. An `emod ai edit <file> "<instruction>"`
  command can reuse it for scripts. Agent users already edit through the MCP server, so wait for a request.
- **Chat history.** The chat of a session lives in the page and goes away with the window. A team can want
  the notes and the decisions of a workshop in a file next to the model.
- **The served viewer.** If `emod diagram --serve` gets AI, the same `internal/ai/edit` serves it through a
  local route. Decide after the panel ships in the desktop app.
- **A large model.** The edit sends the whole source and gets the whole source back. For a model of many
  hundred lines, an edit of one context can send only that context and merge it back. Measure the cost on
  real models first.

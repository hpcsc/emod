# Exact Model Queries

> This proposal adds **no LLM**. It gives exact answers to the questions that a group asks about a model.
> The [AI proposals](./ai/README.md) use these answers. Free questions go to an MCP host
> ([01](./ai/01-mcp-server.md)) or to the ask mode of the AI panel in the desktop app
> ([03](./ai/03-desktop-ai-panel.md)). Both use these exact answers as their facts.

## Problem

An event model is a graph. In a workshop, the group asks questions about the graph all the time:

- "Who reads `RoomReserved`?"
- "What happens after a guest reserves a room?"
- "How does a booking lead to a confirmation email?"
- "What breaks if we rename `SendConfirmationEmail`?"

Each of these questions has an exact answer, and emod already holds the data for it. But the answer is
hard to get:

- The LSP finds the references of a name, but only a developer in an editor can use it.
- `emod export --format diagram-json` gives every edge, but a person must read the JSON and follow the
  edges by hand.
- The viewer dims the arrows that do not touch a selected view. This works only for a view, and only for
  one step.

A language model can answer these questions in words, but it answers with a guess about a fact that emod
can compute. A guess can add an edge that the model does not have, or miss an edge that it has. The
[principles](./ai/README.md#principles) of these proposals say: if emod can compute an answer, emod computes
it.

## Goals

- `emod references <file> <name>`: the declaration of a name and each reference to it, grouped by role.
- `emod trace <file> <from> [<to>]`: the paths through the model from one element, or between two
  elements.
- One Go package, `internal/query`, that the CLI, the LSP, the MCP server and the AI features all call.
  Therefore each surface gives the same answer.
- In the web viewer and the desktop app: a selection highlights the producers and the consumers of any
  element. Two actions highlight the chain that follows from an element and the chain that leads to it.
- The MCP tools `emod_references` and `emod_trace` ([01](./ai/01-mcp-server.md)).
- `--format json` on each command, for scripts and for the facts of AI answers.

## Non-Goals

- An LLM in any part of this proposal.
- An `emod ai ask` command. An MCP host with these tools answers free questions, and the ask mode of the
  AI panel in the desktop app answers them for people with no agent. A third surface for the same job adds
  cost and no value.
- Questions about the quality of a model. "Is this event a real business fact?" is a semantic review
  question ([02](./ai/02-semantic-model-review.md)).
- Queries across more than one `.emod` file. A model is one file.

## How It Works

### One graph, derived in one place

`diagram.SliceEdges` (`internal/diagram/graph.go`) is the one function that derives the arrows of a slice.
The SVG, Mermaid and ASCII renderers call it. The diagram-json export calls it in `appendEdges`
(`internal/export/diagram.go`). `internal/query` calls it too, so the queries walk the same graph that the
diagram shows.

| Edge kind | From | To | DSL that creates it | Label in query output |
| --- | --- | --- | --- | --- |
| `EdgeFlow`, `EdgeTranslationFlow` | command | event | `command -> event` in `flow`; a translation event | `flow` |
| `EdgeRejection` | command | invariant | `command -> rejected` in `flow` | `rejected` |
| `EdgeTriggerCommand` | trigger | command | A trigger and a command in one slice | `issues` |
| `EdgeTriggerReads` | view | trigger | `reads` of a trigger | `reads` |
| `EdgeSubscription` | event | view | `subscribes` of a view | `subscribes` |
| `EdgeAutomationTrigger` | event | automation | `on` of an automation, with `after` | `on` |
| `EdgeAutomationReads`, `EdgeTranslationReads` | view | automation or translation | `reads` | `reads` |
| `EdgeAutomationCommand`, `EdgeTranslationCommand` | automation or translation | command | `command` | `command` |
| `EdgeTranslationExternal` | external system | translation | `external_system` | `from` |

`query.Graph(model)` walks `model.SliceRefs()`, calls `diagram.SliceEdges` for each slice, and keys each
node by its kind and its name. Each node carries its declaration position, its slice and its context. An
invariant is a node only inside its scope, a context or an aggregate, because two scopes can declare the
same invariant name. The LSP already resolves invariants this way, in `invariantScopes`.

### References

`emod references` uses the name resolution of LSP find-references: `newDeclaredNames`, `referencesIn` and
`addSpecReferences` in `internal/lsp/model.go`. These functions move to `internal/query`, and the LSP calls
them from there. Therefore `emod references`, find-references in the editor and `emod_references` cannot
give different answers.

The command groups each reference by its role:

| Kind of name | Groups |
| --- | --- |
| Event | Produced by (`flow`, translation); read by (`subscribes`, `on`); named in specs (`given`, `when`, `then`) |
| Command | Issued by (trigger, automation, translation); produces (`flow`); refused by (`rejected`); named in specs (`when`, `then = command(...)`) |
| View | Built from (`subscribes`); read by (trigger, automation, translation); named in specs (`then = view(...)`) |
| Invariant | Declared in a scope; refused commands (`rejected` in `flow`); named in specs (`then = rejected(...)`) |

### Trace

`emod trace` walks the graph from one element:

- **With `<to>`:** it lists the simple paths from `<from>` to `<to>`, shortest first, up to `--max-paths`
  (default 5). A simple path visits each node one time, so a cycle cannot make the list endless.
- **Without `<to>`:** it lists each element that follows from `<from>`, as a tree in breadth-first order.
  Each element shows one time, under the first element that reaches it.
- **`--upstream`:** it walks the edges backward, so it lists what leads to `<from>`.
- **`--no-views`:** it leaves out the `subscribes` and `reads` edges. What is left is the chain of things
  that the system and the people do, as in the event-flow diagram.

By default, trace walks the views too. A scheduled automation, such as one with `every = "0 2 * * *"`, has
no `on` event. Its only link to the rest of the model is the view that it reads. A trace with no views
does not show what feeds such an automation.

Each step of the output names the edge label, the element, its kind, its context, its slice and its line.
When a path goes from one context to another, the output says so.

### Highlights in the web viewer and the desktop app

The web viewer and the desktop app use the same frontend in `internal/frontend/static`. Today,
`showDetailPanel` in `ui.js` calls `dimArrowsNotTouching(store, node.id)` when the user selects a node. For
a view, the function marks the arrows into and out of the view with the `touches-view` class, and adds
`dims-arrows` to the SVG, so that the other arrows dim. For other kinds of node, it does nothing.
`highlightElements(store, ids)` adds the `hl` class to nodes; the diagnostics panel and the event copies
use it.

This proposal builds on these two functions:

1. **Select any node.** The dim works for each kind of node, not only a view. The arrows that touch the
   selected node stay. The nodes at the other end of those arrows get `hl`. The class `touches-view`
   becomes `touches-selection`.
2. **Show what follows** and **Show what leads here.** The detail panel and the context menu
   (`ctx-actions.js`) get two actions. Each action highlights the nodes and the arrows of the chain, and
   dims the rest.
3. **Show the path between two nodes.** The user selects one node, then Shift-clicks a second node. The
   viewer highlights the paths between them, shortest first, as `emod trace` does.
4. **Event copies.** A copy (`event_copy`, with `copy_of`) stands for its original. The walk treats the
   copy as the original and highlights both.
5. **Escape** clears the highlight.

The walk runs in JavaScript over `store.edges`, and not in WASM. The edges are the diagram-json edges, so
`diagram.SliceEdges` already derived them. They also carry the node IDs and the event copies that the
highlight needs. The answer must show at once on a click, and the WASM pipeline runs in a worker
(`wasm-worker.js`) behind an asynchronous message. The walk itself is a short breadth-first search. A
vitest contract test runs the walk on the diagram-json export of each file in `examples/`, and compares
the result with `emod trace --format json`.

The diagram-json export has no node for an invariant or for an external system. Therefore the viewer does
not highlight `rejected` and `from` edges. The CLI shows them.

### Facts for AI answers

A question such as "how does a booking lead to an email?" needs words, not only a path. Two surfaces give
the words, and both use the exact answer as the facts:

- **An MCP host** calls `emod_trace` and `emod_references`, then writes the answer. The `review-model` and
  `suggest-specs` prompts of [01](./ai/01-mcp-server.md) tell the host to call them.
- **The ask mode of the AI panel** ([03](./ai/03-desktop-ai-panel.md)) calls `query.References`
  and `query.Trace` in the process of the desktop app for each model name in the question. It puts the
  results in the prompt. The model then only orders and phrases facts that emod computed. The panel flags
  each name in the answer that the model does not declare.

## Interface

```
emod references <file> <name> [flags]

      --kind <kind>      command | event | view | invariant (when a name has two kinds)
      --format <format>  text | json (default: text)

emod trace <file> <from> [<to>] [flags]

      --upstream         walk the edges backward
      --no-views         leave out the subscribes and reads edges
      --max-paths <n>    the most paths to list when <to> is given (default 5)
      --format <format>  text | json (default: text)
```

Both commands parse the file with `oracle.Parse` and stop on a parse error, as `emod slices list` does.
A name that the model does not declare exits with code 1 and lists the declared names that are closest to
it.

## Worked Example

The examples use `examples/all_patterns.emod`.

```
$ emod references examples/all_patterns.emod RoomReserved
event RoomReserved   Reservations > Reservation, slice "Reserve a Room"   :49

produced by
  :66   flow        ReserveRoom -> RoomReserved          "Reserve a Room"
read by
  :93   subscribes  view AvailableRoomsView              "View Available Rooms"
  :161  subscribes  view PendingConfirmationsView        "Send Confirmation Email"
  :172  on          automation ConfirmationEmailReactor  "Send Confirmation Email"
  :295  subscribes  view UnconfirmedReservationsView     "View Unconfirmed Reservations"
named in specs
  :72   then   "holds a room nobody has booked"
  :76   given  "refuses a room already held over the same nights"
  :105  given  "lists the rooms a guest can still book"
  :143  given  "closes a stay that is under way"
  :149  given  "refuses to close a stay that already ended"
  :183  when   "asks for a confirmation once a booking has settled"
  :272  given  "asks for a release on every overnight sweep"
  :277  given  "releases a room nobody confirmed"
  :283  given  "refuses to release a stay that already ended"
  :305  given  "lists the bookings the sweep has left to do"
```

```
$ emod trace examples/all_patterns.emod RoomReserved ConfirmationEmailSent
2 paths from RoomReserved to ConfirmationEmailSent

path 1: 3 steps, Reservations -> Notifications
  RoomReserved              event       Reservations   "Reserve a Room"           :49
  -[on, after 15m]->
  ConfirmationEmailReactor  automation  Reservations   "Send Confirmation Email"  :170
  -[command]->
  SendConfirmationEmail     command     Notifications  "Send Confirmation"        :325
  -[flow]->
  ConfirmationEmailSent     event       Notifications  "Send Confirmation"        :335

path 2: 4 steps, Reservations -> Notifications
  RoomReserved              event       Reservations   "Reserve a Room"           :49
  -[subscribes]->
  PendingConfirmationsView  view        Reservations   "Send Confirmation Email"  :159
  -[reads]->
  ConfirmationEmailReactor  automation  Reservations   "Send Confirmation Email"  :170
  -[command]->
  SendConfirmationEmail     command     Notifications  "Send Confirmation"        :325
  -[flow]->
  ConfirmationEmailSent     event       Notifications  "Send Confirmation"        :335
```

In the desktop app, the facilitator selects `RoomReserved`. Its flow arrow from `ReserveRoom` and its
arrows to the three views and the automation stay, and the other arrows dim. The facilitator then picks
**Show what follows**. The diagram highlights the confirmation chain into the `Notifications` context and
the views that the overnight sweep reads. The group sees the answer to "what happens after a guest
reserves a room?" on the diagram, with no AI call.

## Implementation Plan

**Phase 1: `internal/query` and references (M).** Move `newDeclaredNames`, `referencesIn`,
`addSpecReferences` and `invariantScopes` from `internal/lsp` to `internal/query`. The LSP calls them from
there, and its find-references tests stay unchanged and pass. Add `query.References` and the
`emod references` command.

**Phase 2: trace (M).** Add `query.Graph` over `diagram.SliceEdges`, `query.Trace`, and the `emod trace`
command with `--upstream`, `--no-views` and `--max-paths`.

**Phase 3: highlights (M).** Make the dim work for each kind of node. Add the two actions to the detail
panel and the context menu, and the Shift-click path. Add the vitest contract test against
`emod trace --format json`.

**Phase 4: MCP tools (S).** Add `emod_references` and `emod_trace` to the MCP server
([01](./ai/01-mcp-server.md)).

Tests follow the Go test rules of the repository: one umbrella `TestReferences` and one `TestTrace`,
`t.Run` groups for each operation, scenario names that describe behaviour, and `testify/require`.

## Risks and Mitigations

- **Too many paths in a large model.** *Mitigation:* simple paths only, shortest first, and `--max-paths`.
  Without `<to>`, the tree lists each element one time.
- **The Go walk and the JavaScript walk give different answers.** *Mitigation:* both walk edges from
  `diagram.SliceEdges`, and the contract test compares them on each example.
- **One name, two kinds.** A spec `when` can name a command or an event, and a model can declare a command
  and an event with the same name. *Mitigation:* the output lists each kind that matches, and `--kind`
  picks one.
- **One invariant name in two scopes.** *Mitigation:* the output groups an invariant by its scope, as the
  LSP does.
- **A highlight hides the rest of the model.** *Mitigation:* the viewer dims the rest and does not remove
  it. Escape clears the highlight.

## Open Questions

- **Command names.** `emod references` and `emod trace` are top-level commands. An `emod query` group with
  two subcommands keeps the top level small.
- **External systems as nodes.** If the diagram-json export gets a node for an external system, the
  viewer can highlight `from` edges too.
- **A path through a schedule.** A scheduled automation starts from the clock, not from an event. A trace
  can show the schedule, `every "0 2 * * *"`, as the first step of the chain.

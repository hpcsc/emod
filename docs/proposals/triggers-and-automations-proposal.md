# Triggers and Automations — Proposal

## Problem

Event Modeling's Automation pattern is a four-hop chain:

```
events → todo-list view → processor → command → event
```

The processor is wired to a **view**, not to an event. That is the point of the pattern. The "todo list" makes pending work explicit and queryable, which is what makes retries, idempotency, and backlog visibility fall out of the model instead of being invented at runtime. Wiring a processor straight to an event is the shortcut the todo list exists to prevent.

`emod` has two elements in this space, and they are swapped relative to that pattern.

| Element | Activated by | Has `reads` | Canonical shape |
|---|---|---|---|
| `automation` | an **event** — `TriggerEvent`, required (`internal/ast/ast.go:202-216`, `internal/parser/parser.go:1109-1116`), validated against declared event names (`internal/validator/validator.go:303`) | no | no — direct event→command coupling |
| `trigger Processor` / `trigger Schedule` | nothing declared | yes — `Reads` (`internal/ast/ast.go:183`) | yes — but it sits in the wireframe lane |

So `trigger Processor "Hold Sweeper" { reads PendingExpiries }` expresses the canonical automation shape more faithfully than `automation` does. The element named after the pattern cannot express the pattern; the element in the wrong lane can.

Three further consequences follow from the split.

**The exporters have already collapsed the two concepts, inconsistently.** Automations and translation reactors are drawn *inside* the UI/Triggers lane (`internal/diagram/svg.go:149-181`, `internal/diagram/drawio.go:412-432`). Mermaid folds `trigger Schedule` and `trigger Processor` into the `pcr` (processor) timeframe — the same slot a processor would occupy (`internal/diagram/mermaid.go:71-82`). And the palette disagrees across exporters: `#e1d5e7` is the **automation** fill in SVG and draw.io (`fillReactor`, `internal/diagram/drawio.go:78`) but the **trigger** fill in the web viewer (`internal/viewer/static/renderer.js:216`). One hex, two meanings, depending on which command you ran.

**The one canonical wire `trigger` does express is never drawn.** The `reads` edge is built only for translations (`internal/export/export.go:1182-1190`). `Trigger.Reads` survives into JSON export (`internal/export/export.go:879`) and round-trips through `emod fmt`, but no exporter emits an edge for it.

**`trigger` is an overloaded keyword.** At slice level it introduces a wireframe: `trigger UI "Reservation Form" { ... }`. Inside `automation` it names an activation event: `trigger RoomReserved`. Same token (`lexer.KeywordTrigger`), two unrelated meanings, dispatched by context (`internal/parser/parser.go:393-396` vs `internal/parser/parser.go:1060-1069`). `Kind` is also free-form text — nothing validates it, so `trigger Anything "x" {}` parses.

## Goals

- One DSL element per canonical Event Modeling pattern: `trigger` is the human entry point, `automation` is the processor.
- Make `automation` able to express the todo-list shape, and make that shape the path of least resistance.
- Give schedule-driven work a home in `automation`, where it belongs, and remove it from `trigger`.
- Retire the `trigger` keyword overload so one token has one meaning.
- Make the top lane exclusively human-facing across every exporter, and settle one palette.
- Draw the `view → processor` and `view → trigger` edges the model already carries.

## Non-Goals

- Runtime scheduling semantics. How a timer fires, with what durability and delivery guarantee, stays out of the model — the same line drawn for DCB append-condition checking and for spec execution.
- Inferring a todo-list view for existing event-wired automations. Authors declare it.
- Embedding wireframe images in `.emod` files. Section 5 references an asset by path; it does not inline it.
- Changing the Translation pattern's wiring. Only its lane placement moves.

---

## DSL Surface

Four changes and one addition.

### 1. `trigger` loses its kind slot

```
trigger "<name>" {
  description "<text>"      # optional
  actor <ActorName>         # optional
  reads <ViewName>          # optional
}
```

```emod
trigger "Reservation Form" {
  actor Guest
  reads AvailableRoomsView
}
```

The `Kind` identifier is removed rather than validated down to a one-element set. With `Schedule` and `Processor` gone (section 3), every trigger is a human entry point, so the slot carries no information — it would be a mandatory token whose only legal value is `UI`.

The keyword stays `trigger` rather than becoming `wireframe`. Event Modeling calls this the wireframe lane, but the element also legitimately covers non-graphical human entry points — an IVR menu, a CSV drop, a phone call taken by a support agent. `trigger` names the role (something outside the system initiates this slice); `wireframe` in section 5 names the artifact.

### 2. `automation` reads a todo list

```
automation <Name> {
  description "<text>"          # optional
  on <EventName>                # activation — see 3
  every "<expr>"                # activation — see 3
  reads <ViewName>              # optional, lint-encouraged
  command <CommandName>         # required
  target context <ContextName>  # optional
}
```

`reads` names a view declared elsewhere in the model, validated the same way `translation.reads` and `view.subscribes` are. This is the todo list: the projection that holds the work the processor has left to do.

```emod
slice "Expire Stale Holds" {
  automation StaleHoldExpirer {
    every   "0 * * * *"
    reads   PendingExpiries
    command ExpireHold
  }
}
```

Reading: every hour, consult `PendingExpiries`, issue `ExpireHold` for what it contains. The event `ExpireHold` produces is what removes the row — which the model already shows, because `PendingExpiries` subscribes to it.

### 3. Activation is explicit, and exactly one of two forms

`automation` requires exactly one of:

- **`on <EventName>`** — event-driven. The event must be declared in the model. This replaces the `trigger <EventName>` field; the field is renamed, its semantics are unchanged.
- **`every "<expr>"`** — schedule-driven. `<expr>` is either a Go duration (`"5m"`, `"1h"`) for a fixed interval or a five-field cron expression (`"0 2 * * *"`) for a wall-clock schedule. Validation checks the shape, not the semantics.

Declaring both is an error, and declaring neither is an error. A processor whose activation is unstated is the gap that pushed schedules into `trigger` in the first place — an author with a nightly job and no `every` had nowhere to put it, so `trigger Schedule` absorbed the case.

Requiring the choice also forces a decision authors usually leave implicit for todo-list processors: is this woken by the event that adds to the list, or does it poll? Both are real designs with different failure modes, and the model should say which.

### 4. `trigger Schedule` and `trigger Processor` are removed

The kind slot is gone, so these spellings no longer parse. Their replacements:

| Removed | Replacement |
|---|---|
| `trigger Schedule "Nightly Sweep" { reads PendingExpiries }` | `automation NightlySweep { every "0 2 * * *" reads PendingExpiries command … }` |
| `trigger Processor "Hold Sweeper" { reads PendingExpiries }` | `automation HoldSweeper { every "5m" reads PendingExpiries command … }` |
| `trigger UI "Reservation Form" { … }` | `trigger "Reservation Form" { … }` |

The removed spellings could not name the command they issued. The slice's trigger-to-command edge is inferred rather than declared: the exporter fans one edge out to *every* command in the slice (`internal/diagram/drawio.go:497-506`), so a slice with a schedule trigger and two commands draws the schedule as driving both. The replacements name their command.

### 5. A trigger may point at its wireframe

```emod
trigger "Reservation Form" {
  actor     Guest
  reads     AvailableRoomsView
  wireframe "./wireframes/reservation-form.png"
}
```

The canonical top-lane element is a picture of a screen, not a labelled box. `wireframe` takes a path relative to the `.emod` file. Exporters that can carry an image use it in place of the box; the rest keep the box and use the trigger name. The path is not resolved or validated at parse time — a missing file is a lint warning, not a parse error, so a model stays usable before its mockups exist.

---

## Internal Representation

### AST (`internal/ast/ast.go`)

`Trigger` drops `Kind`/`KindPos` and gains `Wireframe`/`WireframePos`:

```go
type Trigger struct {
	Comments       []*Comment
	Name           string
	NamePos        Position
	Description    string
	DescriptionPos Position
	Actor          string
	ActorPos       Position
	Reads          string
	ReadsPos       Position
	Wireframe      string
	WireframePos   Position
	OpenPos        Position
	ClosePos       Position
}
```

`Automation` renames `TriggerEvent` to `OnEvent` and gains `Every` and `Reads`:

```go
type Automation struct {
	Comments         []*Comment
	Name             string
	NamePos          Position
	Description      string
	DescriptionPos   Position
	OnEvent          string
	OnEventPos       Position
	Every            string
	EveryPos         Position
	Reads            string
	ReadsPos         Position
	Command          string
	CommandPos       Position
	TargetContext    string
	TargetContextPos Position
	OpenPos          Position
	ClosePos         Position
}
```

### Lexer (`internal/lexer`)

Two keywords added — `on`, `every`, plus `wireframe` for section 5. `KeywordTrigger` stays; only its automation-level use goes away.

Adding `on` and `every` as keywords makes them reserved, so an event or view named `on` or `every` would stop parsing as an identifier. Keyword lookup is an exact map hit on the raw word (`internal/lexer/tokenizer.go:202-207`) against a lowercase table (`internal/lexer/token.go:72-94`), so it is case-sensitive; element names are conventionally capitalised, and `On`/`Every` stay identifiers.

### Parser (`internal/parser/parser.go`)

- `parseTrigger` (`:551-618`): drop the `Identifier` kind read at `:559-564`; the quoted name now follows the keyword directly. Add a `wireframe` branch alongside `actor` and `reads`. Update the body error at `:604-607` to list the new entry set.
- `parseAutomation` (`:1035-1116`): replace the `KeywordTrigger` branch (`:1060-1069`) with `KeywordOn`; add `KeywordEvery` and `KeywordReads` branches. Update the body error at `:1097`.
- Post-block diagnostics (`:1109-1116`): replace "automation block requires a trigger event" with the exactly-one-of check — "automation block requires either an `on` event or an `every` schedule" when neither is present, "automation block cannot declare both `on` and `every`" when both are.
- Slice dispatch (`:393-396`, `:403-406`) is unchanged.

### Formatter (`internal/formatter/formatter.go`)

- `writeTrigger` (`:197-208`): drop `trigger.Kind` from the header line; emit `wireframe` after `reads`.
- `writeAutomation` (`:341-354`): emit `on` or `every` in place of `trigger`, then `reads`, then `command`, then `target context` — attribute order fixed so `emod fmt` is deterministic.

### Validator (`internal/validator/validator.go`)

Alongside the existing automation checks (`:300-305`):

- `automation.OnEvent` must name a declared event (the current `TriggerEvent` check, renamed).
- `automation.Reads` must name a declared view — the check `translation.Reads` should also have and currently lacks.
- `automation.Every` must parse as a Go duration or a five-field cron expression.
- Exactly one of `OnEvent` / `Every` is set. The parser reports the arity; the validator is where the cross-reference lands.

### Linter (`internal/linter/linter.go`, `internal/linter/descriptions.go`)

One rule, following the `dcb/`-prefixed convention:

**`automation/missing-todo-list`** (warning) — an `automation` with no `reads`.

- With `on <Event>`: the processor is wired straight from an event to a command. Nothing in the model shows what work is outstanding, so retry and idempotency have no representation. Fix: project a view of pending work and read from it.
- With `every "<expr>"`: the processor wakes on a schedule with no declared input, so the model does not say what it acts on.

This rule is what makes the canonical shape the default rather than merely available. Section 2 alone adds an option; without the rule, `on <Event>` + `command` stays the shortest thing to type and nothing points at the cost.

A second rule covers section 5: **`trigger/missing-wireframe-asset`** (info) — a `wireframe` path that does not resolve on disk.

### LSP (`internal/lsp`)

- `completer.go:143` — the slice-level entry list is unchanged. Add `on`, `every`, `reads` to automation-body completions and `wireframe` to trigger-body completions.
- `hover.go:23-30` — `trigger` reads "Defines a manual trigger for a slice"; `automation` reads "Defines an automation that triggers on an event and sends a command". Both need rewriting to state the pattern each element belongs to, and `reads` (`:30`) needs to mention automations.
- New hover entries for `on`, `every`, `wireframe`.

### CUE schema (`internal/cue/schema.cue`)

`#Trigger` (`:15-22`) drops `kind` (currently required) and gains `wireframe?`. `#Automation` (`:54-61`) renames `trigger_event?` to `on_event?` and gains `every?` and `reads?`.

### JSON export (`internal/export/export.go`)

- Trigger: drop `kind`, add `wireframe` (`:879`, and the trigger struct at `:738`).
- Automation: rename `trigger_event` to `on_event`, add `every` and `reads`.
- Edges: keep `automation_trigger` for `on` (`:1151-1159`) and `automation_command` (`:1161-1169`). Add a `reads` edge for `automation.Reads`, mirroring the translation edge at `:1182-1190`. Add a `reads` edge for `Trigger.Reads` — the wire the model has always carried and no exporter has drawn.

### Importer (`internal/importer/importer.go`)

The importer reads diagram JSON back into an AST, so every export change has a counterpart here.

- `defaultTriggerKind` (`:15-17`) and its use at `:160-172` are deleted with the kind slot.
- The `automation_trigger` edge case (`:269-271`) sets `OnEvent`.
- A `reads` edge whose target is an automation sets `Automation.Reads`; whose target is a trigger sets `Trigger.Reads`.
- `Every` and `Wireframe` have no edge representation — they ride on the node's fields and must be carried through the node decode.

### Glossary (`internal/glossary/glossary.go`)

`triggerActorNames` (`:123-131`) reads `Trigger.Actor` and is unaffected by the kind removal.

### Tree-sitter and editors

- `editors/tree-sitter-emod/grammar.js:132-141` — `trigger_definition` drops the `$.identifier` kind and adds a `wireframe` entry.
- `editors/tree-sitter-emod/grammar.js:181-190` — `automation_definition` replaces `seq('trigger', …)` with `seq('on', …)` and adds `every` and `reads`.
- New keywords go in `highlights.scm`; `folds.scm`, `indents.scm`, and `textobjects.scm` need checking for `trigger_definition` field assumptions.
- Corpus files `test/corpus/slice.txt` and `test/corpus/description.txt` contain trigger fixtures.
- `editors/vscode/syntaxes/emod.tmLanguage.json` needs the new keywords.

---

## Diagrams

### The top lane becomes exclusively human

Automations and translation reactors move out of the UI/Triggers lane into the command/view lane.

- `internal/diagram/svg.go:149-164` (automations) and `:166-181` (translation reactors) both position against `triggerLaneY`. Both move to `cmdViewLaneY`.
- `internal/diagram/drawio.go:412` and `:426` do the same and move the same way.
- The lane label "UI / Triggers" (`internal/diagram/svg.go:42`, `internal/diagram/drawio.go:254`) becomes "Wireframes".

The viewer needs a different fix, because it has no Event Modeling lanes to move anything between. Its swimlanes are per *context* (`internal/viewer/static/renderer.js:122-129`), and within a slice the elements are a vertical stack laid out in source order by type. `topRowTypes` is `translations.concat(automations)` and is positioned first, before the trigger row (`internal/viewer/static/layout.js:58`, `:78-90`). So the viewer does not merely file automations under a human-facing label — it stacks them *above* the trigger, putting the machine higher than the person it serves. The fix is to position `topRowTypes` after the trigger row rather than before it, which lands automations next to the commands and views they wire to, matching where the other two exporters will draw them.

Growing real Event Modeling lanes in the viewer would align all three outputs on one layout model, but that is a much larger change than this proposal needs and is not required for the top row to stop being wrong.

Processors go in the command/view lane rather than getting a fourth lane of their own. Both of a processor's canonical neighbours — the view it reads and the command it issues — already live there, so the edges stay short. Giving machines their own lane would also repeat the mistake being fixed: elevating an implementation actor to lane status alongside the three element types Event Modeling actually defines.

The resulting lane structure matches the canonical board: wireframes on top, commands and views in the middle (processors among them, gear-marked), events on the timeline below.

### One palette

`#e1d5e7` currently means automation in SVG and draw.io and trigger in the web viewer. Fixing that means picking one assignment and applying it in all four places — `internal/diagram/drawio.go:74-79` (shared with `svg.go`), `internal/viewer/static/renderer.js:213-219`, and its `web/static/` copy.

| Element | Fill | Stroke |
|---|---|---|
| trigger | `#ffffff` | `#333333` |
| command | `#dae8fc` | `#6c8ebf` |
| event | `#ffe6cc` | `#d79b00` |
| view | `#d5e8d4` | `#82b366` |
| automation | `#e1d5e7` | `#9673a6` |
| translation | `#f5f5f5` | `#666666` |

This keeps the draw.io and SVG assignment, which already matches Event Modeling's three sticky colours for command/event/view, and changes the web viewer's trigger fill from `#e1d5e7` to white. The trigger box should also stop being a plain rounded rect at the same radius as every other element (`renderer.js:224-225`) — a screen outline or title-bar affordance is what stops it reading as a fourth element type.

### New and changed edges

- `view → trigger`, type `reads` — new, and the fix for the currently-dropped `Trigger.Reads`.
- `view → automation`, type `reads` — new, the todo-list wire.
- `event → automation`, type `automation_trigger` — unchanged, now driven by `on`.
- `automation → command`, type `automation_command` — unchanged.
- An `every` automation has no source node. Render a clock badge on the box with the expression as a tooltip.

`EDGE_TYPE_BY_ENDS` (`internal/viewer/static/model.js:141-149`) needs `"view>trigger"` and `"view>automation"` mapped to `reads`, and `"event>automation"` stays. The comment above that table notes the map's directions must match what the exporter writes, because the importer reads it back — so these entries and the export changes have to land together.

`internal/viewer/static/layout.js:267` filters edge types for the focus view and needs no new entry, since both additions reuse `reads`. `internal/viewer/static/ui.js:331-357` renders the detail panel for triggers and automations and needs the new fields.

### Mermaid

`internal/diagram/mermaid.go:71-82` maps `trigger Schedule`/`trigger Processor` to the `pcr` timeframe. With those kinds gone, triggers always emit `ui` and the branch collapses. Automations should emit `pcr` — the slot they were already sharing.

---

## Worked Example

The same domain, before and after.

Today, the nightly expiry of unconfirmed holds has to be spelled as a trigger, because `automation` cannot express a schedule and `trigger` is the only element with `reads`:

```emod
context Reservations {
  aggregate Reservation {
    slice "Expire Unconfirmed Holds" {
      trigger Schedule "Nightly Expiry Sweep" {
        reads PendingExpiries
      }

      command ExpireHold {
        fields { holdId string required }
      }

      event HoldExpired {
        fields {
          holdId    string    required
          expiredAt timestamp required
        }
      }

      flow {
        command -> event: ExpireHold -> HoldExpired
      }
    }

    slice "Notify Guest of Expiry" {
      automation ExpiryNotifier {
        trigger HoldExpired
        command SendExpiryNotice
        target context Notifications
      }
    }
  }
}
```

Two problems are visible. The sweep is drawn in the wireframe lane, next to human-facing screens, and nothing says when it runs — `Schedule` is a bare word. And `ExpiryNotifier` goes straight from `HoldExpired` to `SendExpiryNotice`, so a reader cannot tell whether a notice is outstanding, nor what happens if one send fails.

Under this proposal:

```emod
context Reservations {
  aggregate Reservation {
    slice "Expire Unconfirmed Holds" {
      automation StaleHoldExpirer {
        every   "0 2 * * *"
        reads   PendingExpiries
        command ExpireHold
      }

      command ExpireHold {
        fields { holdId string required }
      }

      event HoldExpired {
        fields {
          holdId    string    required
          expiredAt timestamp required
        }
      }

      flow {
        command -> event: ExpireHold -> HoldExpired
      }
    }

    slice "Pending Expiries" {
      view PendingExpiries {
        fields {
          holdId    string    required
          heldSince timestamp required
        }
        subscribes [RoomHeld, HoldConfirmed, HoldExpired]
      }
    }

    slice "Notify Guest of Expiry" {
      automation ExpiryNotifier {
        on      HoldExpired
        reads   UnsentExpiryNotices
        command SendExpiryNotice
        target context Notifications
      }
    }
  }
}
```

`StaleHoldExpirer` sits in the command/view lane with a clock badge reading `0 2 * * *`, an incoming `reads` edge from `PendingExpiries`, and an outgoing edge to `ExpireHold`. The loop closes visibly: `HoldExpired` is one of `PendingExpiries`' subscriptions, so the event the processor causes is what removes the row it acted on.

`ExpiryNotifier` keeps event activation — a notice should go out promptly, not on a schedule — but now reads `UnsentExpiryNotices`, so an unsent notice is a row someone can see and a retry is a re-read rather than a replay.

The wireframe lane holds only the guest-facing reservation form.

---

## Interaction With the Specs Proposal

`docs/proposals/specs-and-metadata-proposal.md:287-306` proposes timer triggers for automations — `trigger RoomHeld after "24h"` — and explicitly defers cron-style standalone schedules to its open questions. The two proposals meet at exactly that deferral.

- That proposal's `trigger <Event> after "<duration>"` becomes `on <Event> after "<duration>"` under the rename in section 3. The `after` clause is orthogonal to this proposal and composes with `on` unchanged.
- Its deferred cron-style schedule is section 3's `every`. Whichever proposal lands second should treat `every` as already specified rather than re-deriving it.
- `on <Event> after "<duration>"` and `every "<expr>"` stay mutually exclusive under the exactly-one-of rule: the first is "relative to an occurrence", the second is "absolute wall clock".

If the specs work lands first, the only edit needed there is the field rename.

---

## Versioning

`ast.SupportedVersion` (`internal/ast/ast.go:4`) stays at 1. This is a breaking grammar change and the version header exists to absorb exactly that kind of change, but there are no files outside this repository to protect — `emod` is pre-release, and the models that use the retired spellings are the examples and fixtures this proposal rewrites anyway.

The consequence is that a file using `trigger UI "…"` or `automation { trigger … }` fails with a parse error at the retired spelling rather than a versioning message. That is the right trade while the only affected files are ones migrated in the same change, and it keeps a version-2 grammar available for the first change that does have external users to carry across.

Two properties of the header machinery are worth knowing for that later change, since neither is obvious from the surface. The unsupported-version guard is a mismatch test rather than a floor — `declaresUnsupportedVersion` is `h.declared && h.version != ast.SupportedVersion` (`internal/parser/parser.go:80-82`) — so bumping the constant rejects older declared versions as well as newer ones, which is what makes a clear migration message possible. But it fires only on a *declared* header: an absent header implies version 1 while leaving `declared` false (`internal/parser/parser.go:85-87`), so header-less files skip the diagnostic and hit the raw parse error regardless. `emod fmt` writes a header on every file it touches (`internal/formatter/formatter.go:86`, `:106`), so formatted files are covered and unformatted ones are not.

## Phased Implementation

### Phase 1: Automation gains the canonical shape

Additive; nothing breaks.

- `Reads` and `Every` on `ast.Automation`; parser, formatter, CUE, export, importer.
- Validator: `reads` resolves to a declared view; `every` parses.
- `view → automation` `reads` edge in all four exporters.
- Draw the long-dropped `Trigger.Reads` edge in the same pass, since it is the same edge type and the same code paths.

At the end of this phase, the canonical todo-list automation is expressible and `trigger Schedule` has a replacement.

### Phase 2: Retire the overload

Breaking. `emod` is pre-release, so these go directly with no deprecation window.

- Rename `Automation.TriggerEvent` to `OnEvent`; add the `on` keyword; remove the `trigger` branch from `parseAutomation`.
- Enforce exactly-one-of `on` / `every`.
- Remove `Trigger.Kind` from the AST, parser, formatter, CUE, export, and importer, along with `defaultTriggerKind`.
- Update `examples/all_patterns.emod`, `docs/dsl-reference.md:259-339`, the README quick-start, tree-sitter grammar and corpus, and the VS Code grammar.

### Phase 3: Fix the lanes and the palette

No grammar change.

- Move automations and translation reactors to the command/view lane in `svg.go` and `drawio.go`, and reposition the viewer's `topRowTypes` stack below the trigger row (`internal/viewer/static/layout.js:58`, `:78-90`).
- Rename the lane label to "Wireframes".
- Settle the palette; change the web viewer's trigger fill to white.
- Collapse the Mermaid `pcr` branch onto automations.
- The lane label is asserted by string match in six places (`internal/diagram/svg_test.go:26,35,275`, `internal/diagram/drawio_test.go:40,252,497`) and each needs the rename.

### Phase 4: Lint and wireframe assets

- `automation/missing-todo-list`, with its `descriptions.go` entry.
- `wireframe` attribute end to end, plus `trigger/missing-wireframe-asset`.
- Exporters that can embed an image use it in place of the trigger box.

Phases 1 and 3 are independently shippable and carry most of the value: phase 1 closes the semantic gap, phase 3 stops the diagrams from teaching the wrong model. Phase 2 is the one that requires touching every `.emod` file.

---

## Risks and Open Questions

**`on` and `every` become reserved words.** The exposure is narrower than it looks. Field names, field types, modifiers, and tag keys already accept keywords — those positions test `checkIdentifierLike`, which admits `lexer.Identifier` or any keyword (`internal/parser/parser.go:1503-1509`) — so `fields { on timestamp required }` keeps working, and every future keyword inherits the same courtesy. What breaks is an *element* named `on` or `every`, since declarations and cross-references test `lexer.Identifier` strictly. The keyword table is lowercase and lookup is case-sensitive, so conventionally-capitalised names stay identifiers; the residual risk is a lowercase element name, which no current example uses. It is still a one-way door on the identifier space.

**Requiring explicit activation may be too strict.** Canonical Event Modeling boards often leave a processor's wake-up unstated; the gear implies "somehow, continuously". Forcing `on` or `every` asks authors for a decision the whiteboard lets them defer. The argument for requiring it is that the ambiguity is what pushed schedules into `trigger`, and a model that cannot say when a processor runs cannot be checked against an implementation. If this proves onerous, the relaxation is a third mode — a bare `reads` meaning "polls, cadence unspecified" — which can be added later without breaking anything.

**`every` expression syntax.** Accepting both Go durations and five-field cron in one string field means two grammars behind one attribute and shape-sniffing in the validator. The alternatives are two attributes (`every` for durations, `cron` for schedules) or cron only. One field is proposed because authors think of both as "how often", but the validator error messages need care to say which grammar failed.

**Should `reads` on an automation be required rather than lint-warned?** Making it required would guarantee the canonical shape, and the counter-case is thin: an automation that genuinely needs no input is close to a scheduled fire-and-forget, which is rare and arguably worth the friction. It is proposed as a warning because a required `reads` forces a view to exist before the automation can be sketched, which fights the way models get drafted. The lint rule can be promoted to an error later if warnings prove ignorable.

**Wireframe images in exporters.** SVG can embed a data URI; draw.io supports image shapes; Mermaid cannot. The `wireframe` attribute is therefore rendered inconsistently by construction, which risks diagrams that look complete in one format and stubbed in another.

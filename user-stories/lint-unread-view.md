# Flagging a Read Model Nothing Reads

## Overview

A view exists to be read. Either a person is looking at it — a trigger's `reads`, the closest thing the DSL has to a screen — or a processor is working from it, an automation's or translation's `reads`. A view that neither names is a read model with no reader: it draws on the diagram, it carries fields, and nothing in the model says who acts on it.

The linter already guards the other direction. `automation/missing-todo-list` fires when an automation declares no view, on the reasoning that direct event-to-command coupling should be visible rather than silent. Nothing points back the other way, so a view nobody reads passes clean. `god-view` is the only view rule and it counts `subscribes`, which says nothing about consumers.

The gap has a characteristic cause. An author models a processor whose work has no domain command — an outbound push to another system, say — finds there is no automation to write, and declares the todo list anyway so the model has something to show for that behaviour. The result reads as a read model but functions as a comment. Two of them reached a production model this way before anyone noticed.

This set adds the rule, and closes the hole in reference resolution that would otherwise make its message point at the wrong problem.

## Goals

- A view no trigger, automation or translation reads is reported, at the view's own position
- The message names the two shapes a view can legitimately take, so the fix is apparent from the diagnostic
- A misspelled view name in a trigger or translation reports as an unresolved reference rather than surfacing as an orphan view somewhere else in the file
- A model part-way through being drafted is not punished for views whose consumers are not written yet

## User Stories

### US-001: Flag a view nothing reads
**Description:** As a model author, I want a lint warning when no trigger, automation or translation reads a view so that a read model with no reader is visible rather than silent.

**Acceptance Criteria:**
- [ ] `view/never-read` fires for a view whose name appears in no `reads` anywhere in the model, reported at the view's `NamePos`
- [ ] All three spellings of `reads` count as a consumer — a trigger's, an automation's, and a translation's — and a view read by any one of them produces no diagnostic
- [ ] Resolution is model-wide, matching `reads`: a view in one aggregate read by an automation in another is read
- [ ] The message names both legitimate shapes — a trigger reading it, or a processor's todo list — rather than only stating the view is unused
- [ ] The rule fires only when the model already states at least one `reads`, so a model that has not adopted the concept reports nothing, mirroring how `spec/command-without-spec` waits for the first spec
- [ ] `emod lint --explain view/never-read` returns a description, and the rule appears in the reference's lint listing
- [ ] A model whose every view is read produces no diagnostic

**Context:** `Lint()` already builds model-wide maps before the per-node walk — `flowCount` for `left-chair`, `exercisedCommands` and `commandsWithRejection` for the spec rules. A `readViews` set built the same way from `slice.Trigger.Reads`, `slice.Automations[].Reads` and `slice.Translations[].Reads` is the shape this rule wants; `ast.Trigger`, `ast.Automation` and `ast.Translation` each already carry `Reads string`, so nothing new is needed in the AST.

---

### US-002: Resolve a trigger's and a translation's `reads`
**Description:** As a model author, I want a misspelled view name in a trigger or translation reported where I wrote it so that a typo does not present itself as an unread view elsewhere in the file.

**Acceptance Criteria:**
- [ ] `emod validate` reports a trigger's `reads` naming a view no slice declares, at the trigger's `ReadsPos`
- [ ] The same holds for a translation's `reads`, at its `ReadsPos`
- [ ] The diagnostic matches the shape of the existing unresolved-`reads` message an automation already produces, so the three read the same
- [ ] A view named only by a misspelled `reads` reports the unresolved reference and does **not** additionally report as never-read, so one mistake yields one diagnostic
- [ ] The reference's statement that a trigger's and a translation's `reads` are recorded and left unchecked is updated

**Context:** Only an automation's `reads` is resolved today; the other two are recorded and never checked. That asymmetry is harmless on its own, but it becomes actively misleading once US-001 lands: a trigger reading `CaseWorkspacveView` leaves `CaseWorkspaceView` with no consumer, and the author is told the view is never read rather than that they mistyped a name twelve lines earlier. The two changes belong together for that reason, not because either needs the other to compile.

**Depends on:** none — but shipping it after US-001 leaves a window where the rule misreports typos.

## Non-Goals (Out of Scope)

- A suppression directive for a view that is legitimately unread — a datalake export, say. The tool has no ignore mechanism for any rule today, and introducing one for this rule alone would be the wrong place to start that conversation
- Making `reads` multi-valued so one screen can consult several views. Real screens do, and the single-valued field is a genuine constraint, but it is a grammar change with its own consequences — `docs/proposals/screens-as-first-class-proposal.md` takes it up
- A distinct `screen` or `ui` element separate from `trigger`. The same discussion as multi-valued `reads`, and it should not be settled by a lint rule; that proposal makes the case for it
- Checking the other end: whether a view's `subscribes` actually supply the fields its consumer needs
- Any automatic fix, such as deleting an unread view
- Changing `god-view`, which counts subscriptions and is unrelated

## Open Questions

- Warning or error? The two nearest rules disagree: `god-view` returns an error entry, `automation/missing-todo-list` a warning. Since `emod validate` exits non-zero on lint findings either way, the practical difference is how the finding reads rather than whether it blocks — but the pair should probably be consistent with each other, given they guard the two ends of the same relationship.
- Should the "only once the model uses `reads`" guard be scoped per context rather than per model? A large model where one context has adopted views and another has not would report the second on the strength of the first.
- Does a view read *only* by a trigger deserve different treatment from one read by an automation? A trigger's `reads` is unresolved today, so before US-002 a trigger cannot reliably prove consumption; after it, the two are equivalent and this question closes.

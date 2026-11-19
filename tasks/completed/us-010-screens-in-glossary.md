# Task Breakdown: List the screens in the glossary

## Contents

1. Task 1: List each screen in a Screens group under each context where it appears, and an unshown screen at the top level

## Story Reference

`user-stories/screens-as-first-class.md`, US-010 "List the screens in the glossary" (story ID
`screens-as-first-class/US-010`). Goal G1 of `docs/proposals/screens-as-first-class-proposal.md`. US-001 to US-009 are
on main. US-003 removed the actors from the glossary (commit `d2ac177`); screens take their place, with the same
shape.

Decisions for this run:

- **The groups take the shape of the removed actors groups.** The document carries `screens` for the screens that
  no slice shows, and each context section carries `screens` for the screens that its slices show. The markdown
  writes the top-level group right after the model, at the level of a context heading, and the group of a context
  after its views.
- **A context lists its screens in declaration order,** the order of the `screen` declarations of the model, each
  one time however many slices of the context show it. Each term has the description of the declaration.
- **Only declared screens are listed.** A `shows` block that names an undeclared screen is the validator's error.
- **A screen with no label is not listed,** and an empty screen name of a `shows` block matches nothing
  (`tasks/learnings.md`, "The parser keeps an empty name for each unlabelled block").

## Boundaries

**Out of scope:**

- From the stories file, verbatim: "Layout, style, or the structure of components on a screen. A screen names its
  mockup by its path only."
- From the stories file, verbatim: "Navigation between screens. A screen graph is a different model."
- From the stories file, verbatim: "A mockup for each appearance. A second mockup means a second screen."
- From the stories file, verbatim: "Images inside `.emod` files."
- From the stories file, verbatim: "Changes to how `view`, `automation` or `translation` connect."
- From the stories file, verbatim: "A period where `trigger` and `screen` are both valid, and a new language
  version. The proposal retires `trigger` directly (Versioning section)."
- From the stories file, verbatim: "A `title` attribute that restores a human label with spaces."
- From the stories file, verbatim: "A tool that migrates a model. The migration is by hand, with the steps in the
  proposal."
- The `image` of a screen in the glossary. A glossary defines words, and a path is not one.

## Codebase Context

- `internal/glossary/glossary.go` builds one `document` that both writers read: `RenderMarkdown`
  (`markdown.go`) and `RenderJSON` (`json.go`). `contextSections` reads `ctx.AllSlices()`, so the slices of both
  homes of a context.
- `git show d2ac177` shows the removed actors code: `document.Actors` for the unreferenced ones, `contextSection.Actors`
  after `Views`, and the markdown groups at `contextLevel` after the model term and at `aggregateLevel` after the
  views of a context.
- `test.ScreenLibraryLending` shows `LendingDesk` (described) and `OverdueReport` in context "Lending", and `DeskKiosk`
  in both "Lending" and the `mode dcb` context "Reading Room". All its screens are shown.
- The MCP tool `emod_glossary` gives the same text as `emod glossary`, so it needs no change.

## Tasks

### Task 1: List each screen in a Screens group under each context where it appears, and an unshown screen at the top level

**Behavior:** `emod glossary` lists each screen, with its description, in a "Screens" group under each context
where one of its appearances is, and lists a screen that no slice shows in a "Screens" group at the top level. The
JSON format carries the same groups.

**Acceptance Criteria:**

- [x] The markdown lists, under each context, a "Screens" group after the views, with each screen that a slice of
      the context shows, in declaration order, one time, with its description; a screen shown in slices of both
      homes of a context is listed there.
- [x] A screen that appears in two contexts is listed under both.
- [x] A screen that no slice shows is listed in a "Screens" group at the top level, right after the model, and
      under no context. A model whose screens are all shown has no top-level group, and a context that shows no
      screen has no group.
- [x] The JSON carries the same groups: `screens` on the document and on each context section, each term with a
      `description` key.
- [x] A screen with no label is not listed, and a `shows` block that names an undeclared screen adds nothing.
- [x] `docs/dsl-reference.md` says in the `### screen` section that a screen defines a glossary term, and
      `docs/dsl_reference_test.go` names `emod glossary` there.

**Affected Files/Modules:**

- `internal/glossary/glossary.go`, `internal/glossary/markdown.go` — the groups
- `internal/glossary/glossary_test.go` — the screens group of both renderings
- `docs/dsl-reference.md`, `docs/dsl_reference_test.go` — the sentence and its test

**Patterns to Follow:**

- `internal/glossary/glossary.go:40-54`
- `internal/glossary/markdown.go:51-62`
- `internal/glossary/glossary_test.go:66-175`
- `internal/glossary/glossary_test.go:327-410`

**Testable:** Yes

**Certainty:** high — the removed actors groups are the precedent for each part, and `git show d2ac177` holds them.

**Blast radius:** medium — the JSON glossary gains a key that a consumer outside the repository can read, and the
markdown gains a group.

**Verification:** `mise exec -- task test:unit` and `mise exec -- task test:integration` pass.

**Depends on:** None

## Summary

- **Total tasks:** 1.
- **Story criteria:** each criterion of US-010 is in Task 1.
- **Assessments:** routine. The removed actors groups are the shape to restore.

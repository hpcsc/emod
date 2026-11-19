# Task Breakdown: Flag a screen that no slice shows

## Contents

1. Task 1: Report `screen/never-shown` at a screen declaration that no `shows` block names

## Story Reference

`user-stories/screens-as-first-class.md`, US-009 "Flag a screen that no slice shows" (story ID
`screens-as-first-class/US-009`). Goal G1 of `docs/proposals/screens-as-first-class-proposal.md`, section "4. The
Bed rule" ("`screen/never-shown`: the model declares a screen that no slice shows"). US-001 to US-008 are on main.

Decisions for this run:

- **`screen/never-shown` is a warning,** as `view/never-read` is: an unused declaration is a finding to clean up,
  not a broken model. Every lint diagnostic makes `emod validate` exit 1, so no model that the repository ships as
  valid may trip the rule.
- **The rule reports at the screen declaration,** at the name.
- **The rule stays silent while a `shows` block names a screen that the model does not declare.** The validator
  reports that name where it is written, and nothing tells which declared screen it meant, as `view/never-read`
  argues for an unresolved `reads` (`internal/linter/linter.go:710-716`).
- **A screen with no label is not reported:** the parser keeps it with an empty name in a source that does not
  parse (`tasks/learnings.md`, "The parser keeps an empty name for each unlabelled block").

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
- From the story context: an appearance that must issue a command and does not. A `shows` block with no `command`
  is a valid screen that only shows data.

## Codebase Context

- `checkNeverRead` (`internal/linter/linter.go:702-724`) is the precedent: a model-wide set of names that a block
  references, a silence while one of them is unresolved, and a warning at the declaration.
- The guard in `internal/linter/descriptions_test.go` requires a description in `ruleDescriptions`, which is not
  gofmt-clean: add the entry by hand. `cmd/emod/main_test.go` pins the rule count (26 → 27), and
  `internal/cli/lint_test.go` lists each rule by hand. `docs/dsl_reference_test.go` names the explain command of
  each rule about a screen.
- `missingImageEmod` in `internal/cli/lint_test.go` declares a screen that no slice shows, and its leaves expect
  one diagnostic, so the fixture must show its screen.

## Tasks

### Task 1: Report `screen/never-shown` at a screen declaration that no `shows` block names

**Behavior:** `emod lint` warns at the declaration of each screen that no `shows` block of any slice names.
`emod lint --explain screen/never-shown` describes the rule.

**Acceptance Criteria:**

- [x] `linter.Lint` reports one `screen/never-shown` warning at the name of each screen declaration that no `shows`
      block names, and none for a screen that a `shows` block names in a slice of either home or of any context.
- [x] The rule reports nothing while a `shows` block names a screen that the model does not declare.
- [x] The rule does not report a screen with no label.
- [x] `emod lint --explain screen/never-shown` prints a description, `linter.Rules()` lists the rule, the list in
      `internal/cli/lint_test.go` names it, and the rule count of `cmd/emod/main_test.go` is 27.
- [x] The `### screen` section of `docs/dsl-reference.md` lists the rule in a bullet with its
      `emod lint --explain screen/never-shown` invocation, and `docs/dsl_reference_test.go` names it.
- [x] Each model that the repository ships as valid still validates, and each CLI fixture still trips only the rule
      that it names.

**Affected Files/Modules:**

- `internal/linter/linter.go` — the rule
- `internal/linter/linter_test.go` — the `screen/never-shown` group
- `internal/linter/descriptions.go` — the description
- `internal/cli/lint_test.go` — the rule list and `missingImageEmod`
- `cmd/emod/main_test.go` — the rule count
- `docs/dsl-reference.md`, `docs/dsl_reference_test.go` — the bullet and its test

**Patterns to Follow:**

- `internal/linter/linter.go:702-724`
- `internal/linter/descriptions.go:32-61`
- `internal/cli/lint_test.go:1932-1968`
- `tasks/learnings.md:428-432`

**Testable:** Yes

**Certainty:** high — `view/never-read` is the precedent for each part, and `bed` added a model-wide screen walk in
the same file.

**Blast radius:** medium — a new warning makes `emod validate` exit 1 for each model that trips it, which an
author's CI reads.

**Verification:** `mise exec -- task test:unit` and `mise exec -- task test:integration` pass.

**Depends on:** None

## Summary

- **Total tasks:** 1.
- **Story criteria:** each criterion of US-009 is in Task 1.
- **Assessments:** routine. One rule over one model-wide set, with the precedent in the same file.

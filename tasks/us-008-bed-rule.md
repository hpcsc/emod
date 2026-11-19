# Task Breakdown: Flag a screen that issues too many commands

## Contents

1. Task 1: Report `bed` at a screen whose appearances issue three or more distinct commands

## Story Reference

`user-stories/screens-as-first-class.md`, US-008 "Flag a screen that issues too many commands" (story ID
`screens-as-first-class/US-008`). Goal G5 of `docs/proposals/screens-as-first-class-proposal.md`, section "4. The
Bed rule". US-001 to US-007 are on main: each screen is declared one time at the top level (`model.Screens`), and
each `shows` block of a slice names its screen and the commands it issues (`slice.Appearances`).

Decisions for this run:

- **`bed` is a warning.** The story leaves the severity open. The message asks the author a question ("consider
  whether it is several screens, or whether the commands express one intent"), and a screen whose commands express
  one intent is a valid model, so the rule is a warning, as `god-view` is, and not an error as `left-chair` is.
  Every lint diagnostic makes `emod validate` exit 1 (`tasks/learnings.md`, "A lint warning fails `emod
  validate`"), so no model that the repository ships as valid may trip the rule.
- **The rule reports at the screen declaration,** as the story assumes, one time for each screen, at the name.
- **"Across N slices" counts the slices whose appearances issue at least one of the commands,** because the
  sentence says where the commands are issued. A slice that shows the screen and issues no command does not count.
  One slice reads "across 1 slice".
- **The count comes from the `command` entries of the `shows` blocks only,** not from the commands that a slice
  declares, nor from the commands that an automation or a translation issues.

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
- A threshold that a model or a flag can change. The threshold is 3, as for `left-chair`.
- A report at each appearance. The rule reports once, at the declaration.

## Codebase Context

- `linter.Lint` (`internal/linter/linter.go:50-155`) builds model-wide maps over `model.AllSlices()`, the slices of
  both homes and of every context, before it walks each context. `left-chair` (`:606-611`) is the threshold of 3.
- `warning` (`:27-36`) builds a warning entry. The guard in `internal/linter/descriptions_test.go` finds each rule
  from the calls to `warning`, `info` and `errorEntry` and requires a description in `ruleDescriptions`
  (`internal/linter/descriptions.go`), which is not gofmt-clean: add the entry by hand.
- `cmd/emod/main_test.go` 'read rules' pins the count of rules (25), and only `task test:integration` runs it.
  `internal/cli/lint_test.go` 'all rules have descriptions' lists each rule name by hand.
- The models the repository ships as valid are checked by `internal/cli/validate_test.go` and by the clean-input
  leaves of `internal/oracle/oracle_test.go`. A model in which one screen issues three distinct commands must not be
  among them.
- `test.ScreenLibraryLending` shows `LendingDesk` in two slices that issue `BorrowCopy`, `ReturnCopy` and
  `RenewLoan`: three distinct commands, so the fixture trips the rule.

## Tasks

### Task 1: Report `bed` at a screen whose appearances issue three or more distinct commands

**Behavior:** `emod lint` warns at the declaration of each screen whose `shows` blocks issue three or more distinct
commands across the model, with a message that names the screen, the count of commands and the count of slices.
`emod lint --explain bed` describes the rule.

**Acceptance Criteria:**

- [x] `linter.Lint` reports one `bed` warning at the name of a screen declaration when the `shows` blocks of that
      screen issue 3 or more distinct commands, also when the appearances are in slices of both homes and of two
      contexts.
- [x] A command that two appearances of one screen both issue counts one time.
- [x] A screen whose appearances issue 2 distinct commands reports nothing.
- [x] The message is `screen "CheckoutPage" issues 3 commands across 3 slices; consider whether it is several
      screens, or whether the commands express one intent`, with the counts of the model; three commands of one
      `shows` block read "across 1 slice".
- [x] A command that a slice declares, or that an automation or a translation issues, does not count.
- [x] `emod lint --explain bed` prints a description, `linter.Rules()` lists the rule, the list in
      `internal/cli/lint_test.go` names it, and the rule count of `cmd/emod/main_test.go` is 26.
- [x] The `### screen` section of `docs/dsl-reference.md` lists the rule in a bullet, with its severity, its
      threshold and the `emod lint --explain bed` invocation.
- [x] Each model that the repository ships as valid still validates.

**Affected Files/Modules:**

- `internal/linter/linter.go` — the rule
- `internal/linter/linter_test.go` — the `bed` group
- `internal/linter/descriptions.go` — the description
- `internal/cli/lint_test.go` — the rule list and an explain leaf
- `cmd/emod/main_test.go` — the rule count
- `docs/dsl-reference.md` — the bullet in `### screen`

**Patterns to Follow:**

- `internal/linter/linter.go:57-63` and `internal/linter/linter.go:606-611`
- `internal/linter/descriptions.go:32-61`
- `internal/cli/lint_test.go:1932-1968`
- `tasks/learnings.md:428-432`

**Testable:** Yes

**Certainty:** high — `left-chair` is the precedent for a threshold of 3 over a model-wide map, and `view/never-read`
for a map over the `shows` blocks of every slice.

**Blast radius:** medium — a new warning makes `emod validate` exit 1 for each model that trips it, which an
author's CI reads.

**Verification:** `mise exec -- task test:unit` and `mise exec -- task test:integration` pass.

**Depends on:** None

## Summary

- **Total tasks:** 1.
- **Story criteria:** each criterion of US-008 is in Task 1.
- **Assessments:** routine. One rule over one model-wide map, with the precedents in the same file.

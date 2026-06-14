# US-002: Surface DCB anti-patterns via linting

## Progress
- [x] Task 1: Add `Info` severity to the diagnostic package
- [x] Task 2: Implement `dcb/untagged-event` lint rule
- [x] Task 3: Implement `dcb/query-too-broad` lint rule
- [x] Task 4: Implement `dcb/single-tag-everywhere` lint rule
- [ ] Task 5: Implement `dcb/orphan-tag-key` lint rule
- [ ] Task 6: Add `--explain` flag to the lint CLI command

## Story Reference
**US-002: Surface DCB anti-patterns via linting** from `user-stories/dynamic-consistency-boundary.md` (lines 31–43).

**Depends on:** US-001 (DCB parsing and AST support for tags, `decides_on`, and mode are already complete).

## Codebase Context

### Affected Modules
- **`internal/linter/`** — All DCB lint rules are new check functions added here. Existing rules are standalone functions (`checkClickbaitEvent`, `checkGodView`, etc.) called from `checkSlice()` in `linter.go:87-120`. The `Lint()` entry point at `linter.go:36-83` dispatches mode-aware checks. Mode helpers `isDCBMode()`, `isMixedMode()`, `isAggregateMode()` at `linter.go:124-134`. Tests live in `linter_test.go` under a single `TestLint` umbrella with `t.Run()` subtests and inline AST construction.
- **`internal/diagnostic/`** — `entry.go` defines `Severity` type with `Error` and `Warning` constants; no `Info` severity exists yet. `entry_test.go` tests the `String()` output format.
- **`internal/cli/`** — `app.go` wires the `lint` command with a `--format` flag (line 64–91). `lint.go` implements `RunLint(path, format string)` which parses the file, runs the linter, and formats output. `--explain` flag does not exist yet. The `LintError` type and JSON formatting already handle `Error` and `Warning` severities.
- **`internal/ast/`** — `ast.go` defines `Event.Tags []TagEntry`, `Command.DecidesOn *DecidesOnClause` (with `Events []string`, `Predicate PredicateExpr`), `TagEntry` (Key, FieldRef), and predicate types (`TagPredicate`, `LogicalExpr`, `NotExpr`). No `TruePredicate` type exists.

### Existing Patterns
- Lint rules are pure functions returning `*diagnostic.Entry` or `[]*diagnostic.Entry`.
- Rules are called from `checkSlice()` or directly from `Lint()`.
- Mode-aware dispatching: `checkDCBInAggregateMode()` is called only when `isAggregateMode()` is true; `checkAggregateInDCBMode()` only when `isDCBMode()` is true.
- Lint tests use inline `ast.Model` construction in subtests, asserting on `RuleName`, `Severity`, `Filename`, `Line`, and `Message`.
- CLI tests write `.emod` files with `writeTemp` and call `cli.RunLint()`.

## Tasks

### Task 1: Add `Info` severity to the diagnostic package

**Behavior:** The `diagnostic` package exposes an `Info` severity level (below `Warning`) alongside the existing `Error` and `Warning` constants. The `Severity.String()` method returns `"info"` for the new level. The JSON formatter in the CLI recognizes the new severity in its output. The linter package gains an `info()` helper function matching the pattern of `warning()` and `error()`.

**Acceptance Criteria:**
- [ ] `diagnostic.Info` is a valid `Severity` value with `String()` returning `"info"`
- [ ] `info()` helper in the linter returns `*diagnostic.Entry` with `Severity` set to `Info`
- [ ] JSON format output includes `"severity": "info"` for info-level entries
- [ ] Text format output for info-level entries is consistent with other severities

**Affected Files/Modules:**
- `internal/diagnostic/entry.go` — Add `Info` constant to `Severity` enum, update `String()` method
- `internal/diagnostic/entry_test.go` — Add tests for `Info` severity formatting
- `internal/linter/linter.go` — Add `info()` helper function (follow the `warning()` pattern at lines 14–23)
- `internal/cli/lint.go` — Update JSON `hasErrors` logic and severity mapping to handle `Info`
- `internal/cli/lint_test.go` — Add test that JSON output emits correct severity for info-level entries

**Patterns to Follow:**
- The `warning()` helper at `linter.go:14-23` for the `info()` helper pattern
- The `Severity.String()` method at `diagnostic/entry.go:12-19` — extend the switch
- The JSON `formatJSON` function at `lint.go:32-66` — the `Severity` field mapping is at line 41

**Testable:** Yes — `diagnostic.Entry.String()` output, JSON formatting, and `linter.Lint()` all expose this through public APIs.

**Verification:** Tests pass, `go build` succeeds, a model that triggers `dcb/single-tag-everywhere` (added in Task 4) produces `"severity": "info"` in JSON output.

**Depends on:** None

---

### Task 2: Implement `dcb/untagged-event` lint rule

**Behavior:** When running `emod lint` on a model with a context in `dcb` or `mixed` mode, any event (including events in translations) defined in a slice that lacks a `tags` clause produces an `dcb/untagged-event` error. This rule does not fire in `aggregate` mode.

**Acceptance Criteria:**
- [ ] An event without a `tags` clause in a DCB-mode context triggers `dcb/untagged-event` error
- [ ] An event without a `tags` clause in a mixed-mode context triggers `dcb/untagged-event` error
- [ ] An event with a `tags` clause (even a single tag) does not trigger the rule
- [ ] Events in translations are checked the same way as regular events
- [ ] The rule does not fire for events in aggregate-mode contexts (no false positives)

**Affected Files/Modules:**
- `internal/linter/linter.go` — Add `checkUntaggedEvent()` function; wire it into a new DCB checks dispatch (following the `checkDCBInAggregateMode()` dispatch pattern at lines 62–66)
- `internal/linter/linter_test.go` — Add subtests under `TestLint` for all scenarios

**Patterns to Follow:**
- The `checkDCBInAggregateMode()` dispatch at `linter.go:138-175` — mode-gated check structure
- The `checkClickbaitEvent()` function at `linter.go:304-309` — single event check returning `*diagnostic.Entry`
- The DCB event tag iteration in `checkDCBInAggregateMode()` at lines 149-153 for traversing events and translation events
- Mode gating: combine `isDCBMode()` and `isMixedMode()` — see `linter.go:124-134`

**Testable:** Yes — tests construct `ast.Model` instances with DCB/mixed/aggregate mode contexts and assert on diagnostics from `linter.Lint()`.

**Verification:** Tests pass, `go build` succeeds, rule fires correctly in DCB and mixed modes, does not fire in aggregate mode.

**Depends on:** Task 1 (for `Info` severity if needed downstream, though this task only uses Error severity)

---

### Task 3: Implement `dcb/query-too-broad` lint rule

**Behavior:** When running `emod lint` on a model with a context in `dcb` or `mixed` mode, a command with a `decides_on` block that references more than 5 event types, has a missing predicate, or has a predicate that is always `true` produces a `dcb/query-too-broad` warning. This rule does not fire in `aggregate` mode.

**Acceptance Criteria:**
- [ ] A `decides_on` with 6+ event types triggers `dcb/query-too-broad` warning
- [ ] A `decides_on` with exactly 5 event types does not trigger the rule
- [ ] A `decides_on` with a missing predicate triggers the warning
- [ ] A `decides_on` with a predicate that is always `true` triggers the warning
- [ ] A `decides_on` with a normal tag predicate (`tag(key) = value`) does not trigger the rule
- [ ] The rule does not fire in aggregate-mode contexts
- [ ] The rule fires in both DCB and mixed mode

**Affected Files/Modules:**
- `internal/linter/linter.go` — Add `checkQueryTooBroad()` function; wire into the DCB checks dispatch
- `internal/linter/linter_test.go` — Add subtests for all query-too-broad scenarios

**Patterns to Follow:**
- The mode-gated dispatch pattern at `linter.go:62-66`
- The `checkSlice()` command iteration at `linter.go:103-110` for iterating over commands and their `DecidesOn` field
- The `DecidesOnClause` struct in `ast.go:169-176` — `Events` is a `[]string`, `Predicate` is a `PredicateExpr`
- Existing predicate types in `ast.go:183-219` for determining what constitutes an "always true" predicate

**Testable:** Yes — tests construct models with commands that have `DecidesOn` with various event counts and predicate configurations.

**Verification:** Tests pass, `go build` succeeds, boundary cases (exactly 5 events vs 6+) are correct, mode gating works.

**Depends on:** None

---

### Task 4: Implement `dcb/single-tag-everywhere` lint rule

**Behavior:** When running `emod lint` on a model with a context in `dcb` or `mixed` mode, if every command in the context uses only one distinct tag key across all their `decides_on` predicates, an `dcb/single-tag-everywhere` info message is emitted. This rule does not fire in `aggregate` mode.

**Acceptance Criteria:**
- [ ] A context where all commands reference the same single tag key triggers `dcb/single-tag-everywhere` info
- [ ] A context where commands reference multiple distinct tag keys does not trigger the rule
- [ ] A context with no commands does not trigger the rule
- [ ] The rule does not fire in aggregate-mode contexts
- [ ] The rule fires in both DCB and mixed mode

**Affected Files/Modules:**
- `internal/linter/linter.go` — Add `checkSingleTagEverywhere()` function; wire into the DCB checks dispatch. Uses the `info()` helper from Task 1.
- `internal/linter/linter_test.go` — Add subtests for single-tag-everywhere scenarios

**Patterns to Follow:**
- Mode-gated dispatch at `linter.go:62-66`
- Tag key extraction from predicates: the `TagPredicate.Field` field at `ast.go:184` holds the tag key name
- The `checkDCBInAggregateMode()` pattern at `linter.go:138-175` for collecting data across all slices in a context

**Testable:** Yes — tests construct models with commands referencing specific tag keys in their `DecidesOn.Predicate` and assert on diagnostics.

**Verification:** Tests pass, `go build` succeeds, the info-level diagnostic is emitted with correct `RuleName` and `Severity`.

**Depends on:** Task 1 (requires `Info` severity and `info()` helper)

---

### Task 5: Implement `dcb/orphan-tag-key` lint rule

**Behavior:** When running `emod lint` on a model with a context in `dcb` or `mixed` mode, a tag key declared on events but never referenced in any command's `decides_on` predicate produces a `dcb/orphan-tag-key` warning. This rule does not fire in `aggregate` mode.

**Acceptance Criteria:**
- [ ] A tag key declared on events but not used in any command's `decides_on` predicate triggers `dcb/orphan-tag-key` warning
- [ ] A tag key declared on events and used in at least one command's predicate does not trigger the rule
- [ ] Multiple orphan tag keys each produce a separate diagnostic
- [ ] The rule does not fire in aggregate-mode contexts
- [ ] The rule fires in both DCB and mixed mode

**Affected Files/Modules:**
- `internal/linter/linter.go` — Add `checkOrphanTagKey()` function; wire into the DCB checks dispatch
- `internal/linter/linter_test.go` — Add subtests for orphan-tag-key scenarios

**Patterns to Follow:**
- Mode-gated dispatch at `linter.go:62-66`
- Tag collection from events: `Event.Tags` at `ast.go:83` is `[]TagEntry`; `TagEntry.Key` at `ast.go:162` holds the key
- Tag key reference from predicates: `TagPredicate.Field` at `ast.go:184`
- The `checkDCBInAggregateMode()` pattern at `linter.go:138-175` for collecting data across all slices in a context

**Testable:** Yes — tests construct models with specific tag keys on events and commands referencing or not referencing those keys.

**Verification:** Tests pass, `go build` succeeds, orphan keys produce diagnostics, used keys do not.

**Depends on:** Task 3 (the concept of extracting tag keys from `decides_on` predicates is shared)

---

### Task 6: Add `--explain` flag to the lint CLI command

**Behavior:** Running `emod lint --explain <rule-name>` prints a human-readable description of the specified lint rule to stdout and exits successfully without linting any file. Running `--explain` with an unknown rule name prints an error. Running normal `emod lint <file>` continues to work unchanged.

**Acceptance Criteria:**
- [ ] `emod lint --explain dcb/query-too-broad` prints a description of the `dcb/query-too-broad` rule
- [ ] `emod lint --explink dcb/nonexistent` (unknown rule) prints an error message
- [ ] `emod lint <file>` continues to work as before (no regression)
- [ ] All DCB rules added in Tasks 2–5 have descriptions accessible via `--explain`
- [ ] Descriptions for all existing rules (state-obsession, property-sourcing, etc.) are also accessible

**Affected Files/Modules:**
- `internal/linter/linter.go` — Add a rule descriptions map or `DescribeRule()` function
- `internal/cli/app.go` — Add `--explain` flag to the lint command definition (follow the `--format` flag pattern at lines 68–73)
- `internal/cli/lint.go` — Add `RunLintExplain()` or extend `RunLint()` to handle the explain case; lookup descriptions from the linter package
- `internal/cli/lint_test.go` — Add subtests for explain flag behavior

**Patterns to Follow:**
- The `--format` flag wiring at `app.go:68-73` for adding a new flag to the lint command
- The action function at `app.go:75-90` — extend to check for the explain flag before running normal lint
- Existing helper function pattern in `lint.go` (e.g., `formatJSON`, `formatText`)

**Testable:** Yes — tests call `cli.RunLint()` or the new explain function with the explain flag and assert on stdout output.

**Verification:** Tests pass, `go build` succeeds, `--explain` works for all rules, `--explain` with unknown rule returns error, normal lint still works.

**Depends on:** Tasks 2–5 (rule descriptions should match the rules implemented in those tasks)

## Summary

- **Total tasks:** 6
- **Language:** All tasks are Go (all detected languages are available but this story only concerns Go code)
- **Task ordering rationale:** Dependency-first. Task 1 is a prerequisite for Task 4 (Info severity). Tasks 2–5 are independent of each other in terms of behavior but Task 3 (query-too-broad) shares concepts with Task 5 (orphan-tag-key) regarding tag key extraction from predicates — ordering them 3 before 5 is recommended. Task 6 is last as it depends on descriptions for all rules.
- **AC coverage:** All 7 acceptance criteria are covered. AC 1 (untagged event) → Task 2. AC 2 (decides_on >5 events) and AC 3 (missing/always-true predicate) → Task 3. AC 4 (single-tag-everywhere) → Task 4. AC 5 (orphan-tag-key) → Task 5. AC 6 (--explain) → Task 6. AC 7 (mode gating) → embedded in Tasks 2–5.

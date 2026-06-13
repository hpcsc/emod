# US-001: Author a DCB context with tagged events and decision queries

## Progress
- [x] Task 1: Add DCB lexer tokens and AST types
- [ ] Task 2: Parse context mode and accept slices directly under context
- [ ] Task 3: Parse event tags clause
- [ ] Task 4: Parse command decides_on clause with predicate expressions
- [ ] Task 5: Format DCB constructs in formatter output
- [ ] Task 6: Validate DCB cross-references at parse time
- [ ] Task 7: Add mode-aware linter warnings for DCB vs aggregate constructs

## Story Reference

US-001: Author a DCB context with tagged events and decision queries — file `user-stories/dynamic-consistency-boundary-us-001.md` (or inline user story provided in the task prompt).

## Codebase Context

Exploration of `/Users/davidnguyen/Personal/Code/emod` revealed the following relevant structure:

- **Lexer** (`internal/lexer/token.go`, `tokenizer.go`): 26 keyword tokens currently defined. Token scanning uses `getKeywordKind()` switch to map identifier strings to keyword token kinds. Identifiers are scanned as contiguous alphanumeric/underscore sequences. No tokens exist yet for DCB constructs (mode, tags, decides_on, etc.).

- **AST** (`internal/ast/ast.go`): `Context` has only `Aggregates []*Aggregate` — no `Slices` or `Mode`. `Event` has no `Tags`. `Command` has no `DecidesOn`. `Slice` is exclusively nested inside `Aggregate`. Position info is attached via `ast.Position` on every node.

- **Parser** (`internal/parser/parser.go`): Recursive-descent parser. `parseContext()` currently only accepts `aggregate` blocks inside a context body. `parseEvent()` handles `fields` and `source` clauses but not `tags`. `parseCommand()` handles `fields` only. Error reporting uses `p.error(msg)` which appends to `p.diagnostics`. Comments are collected via `takePendingComments()` and attached to the appropriate node. Post-parse validation (e.g., required sub-blocks) is done after the close brace inside each `parse*` function using direct diagnostic appends.

- **Formatter** (`internal/formatter/formatter.go`): Recursive visitor that writes `model -> contexts -> aggregates -> slices -> ...`. `writeContext()` writes aggregates only. `writeEvent()` writes fields and source only. `writeCommand()` writes fields only. Column-width alignment is used for fields.

- **Validator** (`internal/validator/validator.go`): Walks `model.Contexts[].Aggregates[].Slices[]` to collect command/event names and cross-reference automations, translations, views, and flows. Currently uses `diagnostic.Entry` with no severity/rule name for most errors (orphan-command and orphan-event use `Severity` and `RuleName`).

- **Linter** (`internal/linter/linter.go`): Walks the same aggregate->slice path. Checks event naming (state-obsession, property-sourcing, command-in-disguise, clickbait), command naming (past-tense, left-chair), and view naming/structure (view-naming, god-view). Uses `diagnostic.Warning` and `diagnostic.Error` severity levels with `RuleName`. The linter has no awareness of context mode.

- **CLI** (`internal/cli/validate.go`, `lint.go`): `RunValidate()` and `RunLint()` both follow the same pipeline: lex -> parse -> (validate) -> (lint). Diagnostics from each step are appended and reported to the user.

- **Test patterns**: Build tags `unit`/`integration`, `testify/require` assertions, umbrella `TestXxx` functions with `t.Run` groups. Parser tests scan input strings, parse, and assert on AST structure and diagnostics. Linter/validator tests construct AST nodes directly. The test helper in `internal/test/assertions.go` provides `RequireEqual` with `cmpopts.IgnoreTypes` for position fields.

## Tasks

### Task 1: Add DCB lexer tokens and AST types

**Language:** Go

**Behavior:** The lexer recognises new DCB keyword tokens (`mode`, `tags`, `decides_on`, `where`, `and`, `or`, `not`, `tag`, `events`, and `=` punctuation). The AST gains new fields and types to represent mode on contexts, tags on events, decides_on on commands, and predicate expressions.

**Acceptance Criteria:**
- [ ] New keyword tokens are added to the lexer token kinds and their `String()` representations
- [ ] `getKeywordKind()` maps each new keyword string to its corresponding token kind
- [ ] The `Context` type gains a `Mode string` field and a `Slices []*Slice` field
- [ ] The `Event` type gains a `Tags []TagEntry` field (or equivalent new type)
- [ ] The `Command` type gains a `DecidesOn *DecidesOnClause` field (or equivalent new type)
- [ ] New AST types support the decides_on predicate grammar (tag predicates, logical operators, field references)

**Affected Files/Modules:**
- `internal/lexer/token.go` — add new Kind constants: `KeywordMode`, `KeywordTags`, `KeywordDecidesOn`, `KeywordWhere`, `KeywordAnd`, `KeywordOr`, `KeywordNot`, `KeywordTag`, `KeywordEvents`, and `Equals`; add `String()` cases for each
- `internal/lexer/tokenizer.go` — add cases in `getKeywordKind()` for each new keyword
- `internal/ast/ast.go` — add `Mode` and `Slices` to `Context`; add `Tags` to `Event`; add `DecidesOn` to `Command`; define new types as needed for tags and decides_on predicate trees

**Patterns to Follow:**
- Existing keyword/kind pattern in `internal/lexer/token.go:5-43`
- Existing `getKeywordKind` switch in `internal/lexer/tokenizer.go:178-221`
- Existing AST node structure in `internal/ast/ast.go:28-69` (position fields on every node, Comment slices, OpenPos/ClosePos)
- Existing field/slice collection pattern used for `Flows`, `Views`, `Automations`, `Translations` on `Slice`

**Testable:** No — pure type/constant additions with no testable behavior. The next tasks (2-4) exercise these types through parsing.

**Verification:** `go build ./...` succeeds

**Depends on:** None

---

### Task 2: Parse context mode and accept slices directly under context

**Language:** Go

**Behavior:** The parser accepts a `mode` clause on a context declaration and allows `slice` blocks directly inside a context body (alongside `aggregate` blocks). The mode value and any DCB-level slices are stored in the AST. Existing aggregate-based contexts continue to parse without changes.

**Acceptance Criteria:**
- [ ] A context with `mode dcb` (or `mode aggregate`, `mode mixed`) parses without error and the AST stores the mode value
- [ ] A context without a `mode` clause leaves `Mode` as empty string (backward compatible)
- [ ] A context with `mode dcb` accepts a `slice` block directly under it (no aggregate parent) without error
- [ ] A context can contain both `aggregate` and `slice` blocks at the top level
- [ ] An existing aggregate-based `.emod` file parses successfully with no new errors

**Affected Files/Modules:**
- `internal/parser/parser.go` — update `parseContext()` to read optional `mode <identifier>` after the context name; accept `KeywordSlice` blocks alongside `KeywordAggregate` blocks inside the context body; parse a `mode` value without a mode keyword should produce an error diagnostic
- `internal/parser/parser_test.go` — add test cases: context with mode dcb and direct slice; context with mode aggregate and aggregate block; context with no mode (backward compat); context with mode + mixed aggregate and slice; error cases for malformed mode syntax

**Patterns to Follow:**
- Context parsing entry point in `internal/parser/parser.go:109-148` — the loop inside `parseContext()` currently dispatches on `KeywordAggregate`; extend to also dispatch on `KeywordSlice`
- Error diagnostic pattern in `internal/parser/parser.go:915-923` — use `p.error(msg)` for parse errors
- Test pattern in `internal/parser/parser_test.go:59-93` — scan input, parse, assert on AST fields and error count

**Testable:** Yes — parser unit tests verify that DCB syntax produces correct AST and aggregate-only syntax continues to work.

**Verification:** All tests pass (`go test -tags=unit ./internal/parser/...`)

**Depends on:** Task 1

---

### Task 3: Parse event tags clause

**Language:** Go

**Behavior:** The parser accepts a `tags` block inside an event declaration. Each tag entry maps a tag key (identifier) to a field reference (identifier), stored in the AST.

**Acceptance Criteria:**
- [ ] An event with a `tags { key: fieldRef }` block parses without error and stores the tag entries
- [ ] A tag entry uses the syntax `tagKey: fieldRef` (identifier colon identifier)
- [ ] Multiple tag entries in the same `tags` block are accepted
- [ ] An event without a `tags` block remains valid (backward compatible)
- [ ] A `tags` block with invalid syntax (missing colon, missing field ref) produces a parse error

**Affected Files/Modules:**
- `internal/parser/parser.go` — update `parseEvent()` to recognise a `KeywordTags` token and delegate to a new `parseTags()` method; add `parseTags()` that reads a `{ ... }` block of `identifier : identifier` pairs
- `internal/parser/parser_test.go` — add test cases: event with tags; multiple tags; event without tags; malformed tags syntax

**Patterns to Follow:**
- Event parsing flow in `internal/parser/parser.go:366-425` — the inner loop dispatches on `KeywordFields` and `KeywordSource`; add a third dispatch for `KeywordTags`
- Field parsing pattern in `internal/parser/parser.go:731-757` — reads name, type, optional modifier; tag parsing follows a similar key-value pattern
- Test patterns in `internal/parser/parser_test.go:118-139` — existing event tests providing a template

**Testable:** Yes — parser unit tests verify tag parsing and AST population.

**Verification:** All tests pass (`go test -tags=unit ./internal/parser/...`)

**Depends on:** Task 1

---

### Task 4: Parse command decides_on clause with predicate expressions

**Language:** Go

**Behavior:** The parser accepts a `decides_on` block inside a command declaration. The block lists event types and a `where` predicate composed of `tag()` calls with logical operators (`and`, `or`, `not`). The parsed structure is stored in the AST.

**Acceptance Criteria:**
- [ ] A command with a `decides_on { events [...], where ... }` block parses without error
- [ ] The `events` clause lists one or more event names in a bracket-delimited list (following the same pattern as `subscribes`)
- [ ] The `where` predicate supports `tag(key = fieldRef)` terms and `and`, `or`, `not` logical operators
- [ ] Parenthesised sub-expressions in the predicate are supported
- [ ] A command without `decides_on` remains valid (backward compatible)
- [ ] Malformed decides_on syntax (missing events, bad predicate tokens) produces parse errors with location

**Affected Files/Modules:**
- `internal/parser/parser.go` — update `parseCommand()` to recognise `KeywordDecidesOn` and delegate to a new `parseDecidesOn()` method; add `parseDecidesOn()`, `parseDecidesOnEvents()` (list of identifiers in brackets), and `parsePredicate()` (recursive descent for tag/and/or/not/group expressions)
- `internal/parser/parser_test.go` — add test cases: command with decides_on (events + where); various predicate forms; command without decides_on; error cases

**Patterns to Follow:**
- Command parsing flow in `internal/parser/parser.go:325-363` — inner loop currently dispatches on `KeywordFields`; add dispatch for `KeywordDecidesOn`
- Subscribes list parsing in `internal/parser/parser.go:667-699` — bracket-delimited identifier list pattern reused for `events` list in decides_on
- Token consumption pattern via `p.consume()`, `p.check()`, `p.advance()` used throughout the parser
- Existing test patterns for commands in `internal/parser/parser_test.go:95-116` and for subscribes lists in `internal/parser/parser_test.go:483-553`

**Testable:** Yes — parser unit tests verify decides_on parsing and AST population.

**Verification:** All tests pass (`go test -tags=unit ./internal/parser/...`)

**Depends on:** Task 1

---

### Task 5: Format DCB constructs in formatter output

**Language:** Go

**Behavior:** The formatter writes DCB constructs (mode, direct slices, tags, decides_on) in its output, producing valid `.emod` DSL that round-trips through the parser.

**Acceptance Criteria:**
- [ ] A context with `mode dcb` is formatted with the mode clause before the opening brace
- [ ] Slices directly under a context (not inside an aggregate) are formatted at the context body level
- [ ] An event's `tags` block is formatted after the event declaration and before fields/source
- [ ] A command's `decides_on` block is formatted after the command declaration and before fields
- [ ] Formatted output round-trips through the parser to produce an equivalent AST
- [ ] Existing aggregate-based contexts format identically (no regressions)

**Affected Files/Modules:**
- `internal/formatter/formatter.go` — update `writeContext()` to emit mode clause and iterate `ctx.Slices` alongside `ctx.Aggregates`; update `writeEvent()` to emit tags block; update `writeCommand()` to emit decides_on block; add helper methods for writing tag entries and decides_on/predicate trees
- `internal/formatter/formatter_test.go` — add round-trip test cases for DCB constructs

**Patterns to Follow:**
- Formatter visitor pattern in `internal/formatter/formatter.go:55-65` — existing `writeContext` iterates aggregates; extend to also write slices
- Event formatting in `internal/formatter/formatter.go:172-182` — extend to write tags block before fields
- Command formatting in `internal/formatter/formatter.go:163-170` — extend to write decides_on block before fields
- View subscribes list formatting in `internal/formatter/formatter.go:226` — similar bracket-delimited list pattern for decides_on events

**Testable:** Yes — formatter tests verify output correctness; round-trip tests verify parser → formatter → parser equivalence.

**Verification:** All tests pass (`go test -tags=unit ./internal/formatter/...`)

**Depends on:** Tasks 2, 3, 4

---

### Task 6: Validate DCB cross-references at parse time

**Language:** Go

**Behavior:** The validator checks that tag keys reference declared event fields, decides_on event names exist in the model, and decides_on tag keys are declared on the referenced events. Invalid references produce clear error diagnostics with source location.

**Acceptance Criteria:**
- [ ] A tag key in `tags { key: fieldRef }` that does not match a declared field name on that event produces an error diagnostic with location
- [ ] An event name listed in `decides_on.events` that does not exist in the model produces an error diagnostic with location
- [ ] A field reference `tag(key = fieldRef)` in a decides_on where clause that references a field not declared on any of the listed events produces an error
- [ ] A tag key in `tag(key = ...)` that is not declared on any of the listed events produces an error diagnostic
- [ ] Valid references produce no diagnostics
- [ ] Existing aggregate models produce no new validation errors

**Affected Files/Modules:**
- `internal/validator/validator.go` — update `Validate()` to collect event names, their fields, and their tags from both `ctx.Aggregates[].Slices[]` and `ctx.Slices[]`; add validation functions for tag reference resolution, decides_on event name resolution, and decides_on predicate tag/field resolution
- `internal/validator/validator_test.go` — add test cases: valid DCB references; invalid tag key; invalid event name in decides_on; invalid field ref in predicate; valid cross-references across slices

**Patterns to Follow:**
- Existing cross-reference validation in `internal/validator/validator.go:21-63` — collects command/event names from all slices, then iterates again to check references
- Diagnostic format in `internal/validator/validator.go:71-78` — uses `diagnostic.Entry` with `Filename`, `Line`, `Column`, `Message`
- Test pattern in `internal/validator/validator_test.go` — constructs AST directly and calls `Validate()`

**Testable:** Yes — validator tests construct ASTs with valid/invalid references and verify diagnostics.

**Verification:** All tests pass (`go test -tags=unit ./internal/validator/...`)

**Depends on:** Tasks 2, 3, 4

---

### Task 7: Add mode-aware linter warnings for DCB vs aggregate constructs

**Language:** Go

**Behavior:** The linter checks context mode and warns when DCB constructs appear in an aggregate-mode context, or when aggregate blocks appear in a DCB-mode context. Mixed mode accepts both without warnings. Existing lint checks also apply to DCB slices.

**Acceptance Criteria:**
- [ ] A context with `mode aggregate` (or no mode, meaning default) produces warnings for DCB-only constructs (tags on events, decides_on on commands, slices directly under context)
- [ ] A context with `mode dcb` produces warnings when it contains an `aggregate` block
- [ ] A context with `mode mixed` produces no warnings for either DCB or aggregate constructs
- [ ] Existing lint checks (state-obsession, property-sourcing, command-in-disguise, clickbait, command-past-tense, left-chair, view-naming, god-view) apply to slices in both aggregate and DCB contexts
- [ ] Existing aggregate-based models produce no new warnings

**Affected Files/Modules:**
- `internal/linter/linter.go` — update `Lint()` to read `context.Mode` and emit mode-mismatch warnings; extend the walk to cover both `ctx.Aggregates[].Slices[]` and `ctx.Slices[]`; add mode-checking helper functions
- `internal/linter/linter_test.go` — add test cases: aggregate mode + DCB constructs warns; dcb mode + aggregate warns; mixed mode + both is clean; backward compat with no-mode contexts; existing lint rules fire on DCB slices

**Patterns to Follow:**
- Existing linter warning pattern in `internal/linter/linter.go:14-23` — `warning()` helper creates a `diagnostic.Entry` with `Severity: Warning`
- Existing linter walk in `internal/linter/linter.go:55-90` — nested loops over contexts → aggregates → slices; extend to also walk `context.Slices`
- Rule naming convention uses kebab-case rule names (e.g., `"state-obsession"`, `"command-past-tense"`)
- Existing linter test pattern in `internal/linter/linter_test.go` — constructs AST with positions and calls `Lint()`

**Testable:** Yes — linter tests construct ASTs in various mode configurations and verify warning diagnostics.

**Verification:** All tests pass (`go test -tags=unit ./internal/linter/...`; `go test -tags=integration ./...` for full pipeline)

**Depends on:** Tasks 2, 3, 4

## Summary

**Total tasks:** 7

**Task ordering rationale:** Dependency-first order. Tasks 1 provides the infrastructure (tokens + types) that all subsequent tasks require. Tasks 2-4 add parsing capabilities for the three new grammatical constructs in order of structural scope (context level → event level → command level). Task 5 updates the formatter to emit the new constructs. Task 6 adds cross-reference validation that the parser AST enables. Task 7 adds mode-aware linter warnings, which closes the loop by surfacing the only checks gated by the mode flag.

**Acceptance criteria coverage:**

| AC | Description | Covered By |
|---|---|---|
| AC1 | DCB context accepts direct slices | Task 2 |
| AC2 | Event tags clause | Task 3 |
| AC3 | Command decides_on with predicate | Task 4 |
| AC4 | Aggregate mode warns on DCB constructs | Task 7 |
| AC5 | DCB mode warns on aggregate blocks | Task 7 |
| AC6 | Mixed mode accepts both | Task 7 |
| AC7 | Invalid references produce clear errors | Task 6 |
| AC8 | Existing models parse successfully | Implicitly verified by all tasks (backward-compat tests in Tasks 2, 5, 6, 7) |

**Key dependencies between tasks:**
- Tasks 2, 3, 4 depend on Task 1 (need tokens and AST types)
- Task 5 depends on Tasks 2, 3, 4 (need parsed structures to format)
- Task 6 depends on Tasks 2, 3, 4 (needs parsed DCB data for cross-referencing)
- Task 7 depends on Tasks 2, 3, 4 (needs mode field and DCB slice walking)
- Tasks 2, 3, 4 are independent of each other and could be parallelised, though sequential execution is recommended for clean commit history
- Tasks 5, 6, 7 are independent of each other and could be parallelised after their dependencies are met

# US-004: Format .emod Files Consistently

## Progress
- [x] Task 1: Add a Comment token kind to the lexer and emit comment tokens
- [x] Task 2: Define a comment-preserving AST representation
- [x] Task 3: Attach comments to AST nodes during parsing
- [x] Task 4: Implement the core AST pretty-printer (formatter) without comment support
- [x] Task 5: Add field column-alignment logic to the formatter
- [x] Task 6: Add comment rendering to the formatter output
- [x] Task 7: Add blank-line normalization between slices
- [x] Task 8: Wire up the `emod fmt` CLI command with write and --check modes

## Story Reference

**Source:** Inline user story -- US-004: Format .emod files consistently

**Summary:** As a model author, I want to auto-format my `.emod` files so that models have consistent indentation and structure regardless of who wrote them.

**Acceptance Criteria (from story):**
- `emod fmt reservation.emod` rewrites the file with consistent indentation (2-space indent per nesting level)
- Keyword alignment within `fields` blocks is normalized (field names, types, and modifiers column-aligned)
- Blank lines between slices are normalized to exactly one
- Comments are preserved in their original positions
- Running `emod fmt` on an already-formatted file produces no changes
- `emod fmt --check reservation.emod` exits with code 1 if formatting changes are needed, 0 if already formatted (for CI use)

**Depends on:** US-001

## Codebase Context

**Project structure:** Go CLI tool using urfave/cli v2. Entry point at `cmd/emod/main.go`, CLI wiring in `internal/cli/app.go`. Currently has one command (`validate`) defined in `internal/cli/validate.go`.

**Lexer (`internal/lexer/tokenizer.go`):** The `Scan` function produces a flat `[]*Token` slice. Comments are currently consumed and discarded in `skipWhitespaceAndComments` (lines 82-106) -- the function reads `#`-prefixed text until newline and advances the cursor without emitting any token. This means the current token stream is comment-free.

**Token types (`internal/lexer/token.go`):** The `Kind` enum has no `Comment` variant. Tokens carry `Type`, `Value`, `Line`, and `Column`.

**AST (`internal/ast/ast.go`):** All node types are defined (Model, Actor, Context, Aggregate, Slice, Command, Event, Field, Flow, Trigger, View, Automation, Translation). No node type carries associated comments. Every node has `Position` fields but no comment attachment points.

**Parser (`internal/parser/parser.go`):** Recursive-descent parser consuming `[]*lexer.Token`. It skips unrecognized tokens with error recovery. It does not handle comment tokens since none are emitted.

**CLI command pattern (`internal/cli/app.go:10-30`):** Commands are registered as entries in the `Commands` slice of the `urfave.App`. Each command has `Name`, `Usage`, `ArgsUsage`, and an `Action` function. The `validate` command in `validate.go` shows the extraction pattern: read file, lex, parse, collect diagnostics, format errors.

**Test patterns:** Unit tests use `//go:build unit`, integration tests use `//go:build integration`. Tests use `github.com/stretchr/testify/require`. The `validate_test.go` file uses a `writeTemp` helper to create temporary `.emod` files for testing.

**Existing test fixtures:** `internal/parser/testdata/` contains `minimal.emod`, `all_patterns.emod`, `multi_context.emod`, and `invalid.emod`. These fixtures demonstrate the canonical formatting style (2-space indent, column-aligned fields, single blank lines between slices, comments before slices with `# Slice N:` prefix).

**Key challenge:** The formatter must preserve comments, but the entire pipeline (lexer -> parser -> AST) currently discards them. This requires changes at multiple layers before the formatter can be built.

## Tasks

### Task 1: Add a Comment token kind to the lexer and emit comment tokens

**Behavior:** The lexer emits `Comment` tokens for `#`-prefixed lines instead of silently discarding them. Each comment token carries the full comment text (including the `#` prefix) and its position. All existing lexer behavior remains unchanged -- keywords, identifiers, strings, punctuation, and whitespace handling are unaffected. The parser continues to work because it simply skips tokens it does not recognize.

**Acceptance Criteria:**
- [ ] A `Comment` kind exists in the `Kind` enum in `token.go`
- [ ] `Scan` emits a `Comment` token for each `#`-prefixed line, with the value containing the comment text (including the `#` character, trimming trailing newline)
- [ ] The `Comment` token carries the correct line and column position
- [ ] All existing lexer tests continue to pass (comment-related tests now see `Comment` tokens in the stream rather than their absence)
- [ ] The parser continues to produce correct ASTs (existing parser unit and integration tests pass unchanged, because the parser skips unrecognized token types)

**Affected Files/Modules:**
- `internal/lexer/token.go` -- add `Comment` to the `Kind` enum and its `String()` case
- `internal/lexer/tokenizer.go` -- modify `skipWhitespaceAndComments` to emit `Comment` tokens instead of discarding comment text
- `internal/lexer/tokenizer_test.go` -- update existing comment tests to assert `Comment` tokens are present; add new tests for comment token value and position

**Patterns to Follow:**
- Follow the existing token kind declaration pattern in `internal/lexer/token.go:6-42` for adding the new `Comment` variant.
- Follow the existing `skipWhitespaceAndComments` function in `internal/lexer/tokenizer.go:82-106` for where comment handling currently lives.

**Testable:** Yes

**Verification:** `go test ./... -tags unit,integration -count=1` passes. Existing parser tests remain green. New lexer tests verify comment tokens are emitted with correct values and positions.

**Depends on:** None

---

### Task 2: Define a comment-preserving AST representation

**Behavior:** AST node types gain the ability to carry associated comments. A comment attachment model is added so that comments appearing before a node can be associated with that node. This is a data-structure-only change -- no parsing logic changes yet.

**Acceptance Criteria:**
- [ ] A `Comment` type exists in the `ast` package with `Text` and `Position` fields
- [ ] AST container nodes that can have preceding comments gain a `Comments` field (or equivalent) -- specifically: `Model`, `Actor`, `Context`, `Aggregate`, `Slice`, `Command`, `Event`, `View`, `Automation`, `Translation`, `Trigger`, `Flow`
- [ ] `go build ./...` succeeds
- [ ] All existing tests pass unchanged (the new fields are zero-valued and unused)

**Affected Files/Modules:**
- `internal/ast/ast.go` -- add `Comment` struct; add comment fields to node types

**Patterns to Follow:**
- Follow the existing AST node pattern in `internal/ast/ast.go:3-7` (Position struct with Filename/Line/Column) for the Comment type structure.
- Follow the existing field naming conventions in `internal/ast/ast.go` (e.g., `NamePos Position`, `OpenPos Position`) for the comment attachment fields.

**Testable:** No -- data types only. Tested indirectly through parsing and formatting in later tasks.

**Verification:** `go build ./...` succeeds. `go test ./... -tags unit,integration -count=1` passes.

**Depends on:** None

---

### Task 3: Attach comments to AST nodes during parsing

**Behavior:** The parser consumes `Comment` tokens from the token stream and attaches them to the AST node that follows them. When one or more comment lines appear before a declaration (e.g., before a `slice` block), those comments are stored on the subsequent AST node. Comments at the top of the file (before the `model` declaration) are attached to the `Model` node. This ensures comments survive the parse round-trip.

**Acceptance Criteria:**
- [ ] Comments appearing before a `model` declaration are attached to the `Model` node
- [ ] Comments appearing before an `actor` declaration are attached to the `Actor` node
- [ ] Comments appearing before a `context`, `aggregate`, `slice`, `command`, `event`, `view`, `automation`, `translation`, `trigger`, or `flow` block are attached to the respective AST node
- [ ] Multiple consecutive comment lines before a node are all attached to that node
- [ ] Parsing the `all_patterns.emod` test fixture produces AST nodes with the expected comments (e.g., `# Hotel Reservation System` on the Model, `# Slice 1: Command Pattern` on the first slice)
- [ ] All existing parser tests continue to pass

**Affected Files/Modules:**
- `internal/parser/parser.go` -- add logic to collect pending `Comment` tokens and attach them to the next parsed node
- `internal/parser/parser_test.go` -- add tests verifying comment attachment for various node types
- `internal/parser/integration_test.go` -- add assertions on the `all_patterns.emod` fixture verifying comments are attached to the correct nodes

**Patterns to Follow:**
- Follow the parser's token consumption pattern in `internal/parser/parser.go:52-63` (the main parse loop that dispatches on token type) for where to add comment collection.
- Follow the existing `check`/`advance` pattern in `internal/parser/parser.go:833-838` for consuming comment tokens.

**Testable:** Yes

**Verification:** `go test ./... -tags unit,integration -count=1` passes. New tests confirm comments from `all_patterns.emod` are attached to the correct AST nodes.

**Depends on:** Task 1, Task 2

---

### Task 4: Implement the core AST pretty-printer (formatter) without comment support

**Behavior:** A new `internal/formatter` package provides a function that takes an `*ast.Model` and produces a formatted string representation of the `.emod` file. The formatter emits the canonical formatting: 2-space indentation per nesting level, correct keyword/name/brace placement for all node types (model, actor, context, aggregate, slice, command, event, fields, flow, trigger, view, automation, translation). Comments and field alignment are deferred to subsequent tasks. The formatter handles all AST node types present in the existing test fixtures.

**Acceptance Criteria:**
- [ ] A `Format` function (or equivalent) exists in `internal/formatter/` that accepts an `*ast.Model` and returns a `string`
- [ ] Top-level `model "Name"` and `actor "Name"` declarations are emitted on separate lines with no indentation
- [ ] `context`, `aggregate`, and `slice` blocks are emitted with correct nesting (2-space indent per level)
- [ ] `command`, `event`, `view`, `trigger`, `automation`, `translation`, and `flow` blocks within slices are emitted at the correct indent level
- [ ] `fields` blocks are emitted with field lines at the correct indent level (field alignment is not yet required -- one space between columns is acceptable)
- [ ] `subscribes [...]` lists are emitted correctly within view blocks
- [ ] `flow { command -> event: Name -> Name }` entries are emitted correctly
- [ ] `source external "Provider"` syntax in events is emitted correctly
- [ ] `target context Name` syntax in automations is emitted correctly
- [ ] `external_system "Name"` syntax in translations is emitted correctly
- [ ] Nested events within translations are emitted correctly
- [ ] Formatting the AST parsed from `all_patterns.emod` (ignoring comments and field alignment) produces valid output that re-parses to an equivalent AST

**Affected Files/Modules:**
- `internal/formatter/` (new package) -- formatter implementation
- `internal/formatter/formatter_test.go` (new) -- unit tests for formatting individual node types and round-trip tests

**Patterns to Follow:**
- Follow the AST traversal pattern visible in `internal/validator/validator.go:10-36` for iterating over the nested model structure (Model -> Contexts -> Aggregates -> Slices -> nested elements).
- Follow the test fixture pattern in `internal/cli/validate_test.go:14-88` for inline `.emod` content used in tests.

**Testable:** Yes

**Verification:** `go test ./... -tags unit,integration -count=1` passes. Round-trip test: parse a fixture, format it, re-parse, and assert the two ASTs are structurally equivalent.

**Depends on:** Task 2

---

### Task 5: Add field column-alignment logic to the formatter

**Behavior:** The formatter aligns field names, types, and modifiers within each `fields` block into columns. Within a single `fields` block, all field names are left-aligned to the same width, all types are left-aligned to the same width, and all modifiers start at the same column. Different `fields` blocks are aligned independently of each other.

**Acceptance Criteria:**
- [ ] Within a `fields` block, field names are padded to the width of the longest name in that block
- [ ] Within a `fields` block, field types are padded to the width of the longest type in that block
- [ ] Modifiers (e.g., `required`, `optional`) follow the type column with consistent spacing
- [ ] Fields without a modifier do not produce trailing whitespace
- [ ] Formatting the AST parsed from `all_patterns.emod` produces field blocks matching the column alignment in that fixture file
- [ ] Formatting an already-aligned file produces identical output

**Affected Files/Modules:**
- `internal/formatter/` -- enhance the field-rendering logic to compute column widths and pad accordingly

**Patterns to Follow:**
- Follow the field iteration pattern in the formatter from Task 4 for where field rendering occurs.
- Reference the expected alignment in `internal/parser/testdata/all_patterns.emod:17-22` and `internal/parser/testdata/all_patterns.emod:27-33` for the canonical column-aligned output.

**Testable:** Yes

**Verification:** `go test ./... -tags unit,integration -count=1` passes. Tests assert that formatted field blocks match expected column-aligned output character-for-character.

**Depends on:** Task 4

---

### Task 6: Add comment rendering to the formatter output

**Behavior:** The formatter emits comments that are attached to AST nodes in their correct positions -- before the node they are associated with, at the appropriate indentation level. This completes the comment-preservation requirement: comments survive a parse-format round-trip.

**Acceptance Criteria:**
- [ ] Comments attached to a node are emitted on lines immediately before that node
- [ ] Comment indentation matches the indentation level of the node they precede
- [ ] Multiple consecutive comments before a node are all emitted in order
- [ ] Formatting the AST parsed from `all_patterns.emod` preserves all comments from the original file
- [ ] Running the formatter on the formatted output of `all_patterns.emod` produces no changes (idempotency with comments)

**Affected Files/Modules:**
- `internal/formatter/` -- add comment rendering before each node type's output

**Patterns to Follow:**
- Follow the comment attachment structure defined in Task 2 (the `Comments` field on AST nodes) for reading comments during formatting.
- Reference `internal/parser/testdata/all_patterns.emod:1` and `internal/parser/testdata/all_patterns.emod:9,42,54,64` for the expected comment positions in formatted output.

**Testable:** Yes

**Verification:** `go test ./... -tags unit,integration -count=1` passes. Round-trip test on `all_patterns.emod` produces byte-identical output after formatting.

**Depends on:** Task 3, Task 5

---

### Task 7: Add blank-line normalization between slices

**Behavior:** The formatter normalizes blank lines between `slice` blocks within an `aggregate` to exactly one blank line. Consecutive slices are separated by one blank line regardless of how many blank lines (or zero) existed in the original input. Blank lines between other block types (e.g., between a `command` and an `event` within a slice) are also normalized to a single blank line for consistency.

**Acceptance Criteria:**
- [ ] Exactly one blank line appears between consecutive `slice` blocks in the formatter output
- [ ] No leading blank line before the first slice in an aggregate
- [ ] No trailing blank line after the last slice in an aggregate (before the closing brace)
- [ ] Blank lines between top-level declarations (`model`, `actor`, `context`) are normalized to exactly one
- [ ] A file with zero blank lines between slices gains the correct blank lines after formatting
- [ ] A file with excessive blank lines between slices has them reduced to exactly one after formatting
- [ ] Idempotency: formatting an already-normalized file produces no changes

**Affected Files/Modules:**
- `internal/formatter/` -- add blank-line emission logic between sibling nodes at each nesting level

**Patterns to Follow:**
- Reference `internal/parser/testdata/all_patterns.emod` for the expected blank-line pattern between slices (lines 39-42 show a blank line between slice blocks).
- Reference `internal/parser/testdata/minimal.emod` for blank-line pattern between top-level declarations.

**Testable:** Yes

**Verification:** `go test ./... -tags unit,integration -count=1` passes. Tests verify blank-line normalization for both adding and removing blank lines.

**Depends on:** Task 6

---

### Task 8: Wire up the `emod fmt` CLI command with write and --check modes

**Behavior:** The `emod fmt` command is registered in the CLI application. Running `emod fmt <file>` reads the file, parses it, formats the AST, and writes the formatted output back to the file. Running `emod fmt --check <file>` compares the formatted output to the original file content and exits with code 1 if they differ (for CI use) or code 0 if they match. Parse errors during formatting are reported to stderr and cause a non-zero exit.

**Acceptance Criteria:**
- [ ] `emod fmt reservation.emod` rewrites the file in-place with formatted content
- [ ] `emod fmt` on an already-formatted file produces no changes to the file
- [ ] `emod fmt --check reservation.emod` exits 0 when the file is already formatted
- [ ] `emod fmt --check reservation.emod` exits 1 when the file needs formatting changes
- [ ] `emod fmt` with no file argument produces an error message
- [ ] `emod fmt` on a file with parse errors reports the errors to stderr and exits non-zero without modifying the file
- [ ] The `fmt` command appears in `emod --help` output

**Affected Files/Modules:**
- `internal/cli/app.go` -- register the `fmt` command in the `Commands` slice
- `internal/cli/fmt.go` (new) -- implement the `fmt` command action function
- `internal/cli/fmt_test.go` (new) -- unit tests for the fmt command

**Patterns to Follow:**
- Follow the command registration pattern in `internal/cli/app.go:14-27` for adding the `fmt` command alongside `validate`.
- Follow the command implementation pattern in `internal/cli/validate.go` for file reading, lexing, parsing, and error reporting.
- Follow the test pattern in `internal/cli/validate_test.go` (including the `writeTemp` helper) for testing the command with temporary files.

**Testable:** Yes

**Verification:** `go test ./... -tags unit,integration -count=1` passes. Tests verify in-place write mode, --check mode exit codes, error handling for missing arguments and parse errors.

**Depends on:** Task 7

## Summary

**Total tasks:** 8

**Ordering rationale:** Dependency-first, bottom-up layering. The formatter depends on comment-aware AST nodes, which depend on comment-aware lexing and parsing. The decomposition layers:
1. Tasks 1-2 are independent leaf changes (lexer comment tokens, AST comment fields) that can be done in parallel
2. Task 3 (parser comment attachment) depends on both Tasks 1 and 2
3. Task 4 (core formatter) depends only on Task 2 (AST types) and can be worked on before comment parsing is complete
4. Task 5 (field alignment) builds on Task 4
5. Task 6 (comment rendering) needs both comment-attached ASTs (Task 3) and the formatter (Task 5)
6. Task 7 (blank-line normalization) refines the formatter output (Task 6)
7. Task 8 (CLI wiring) depends on the complete formatter (Task 7)

**Acceptance criteria coverage:**

| Story Acceptance Criterion | Covered By |
|---|---|
| `emod fmt reservation.emod` rewrites with 2-space indent | Task 4, Task 8 |
| Keyword alignment within `fields` blocks | Task 5 |
| Blank lines between slices normalized to exactly one | Task 7 |
| Comments preserved in their original positions | Task 1, Task 2, Task 3, Task 6 |
| Running on already-formatted file produces no changes | Task 4 (round-trip), Task 7 (idempotency), Task 8 (no-op write) |
| `emod fmt --check` exits 1 if changes needed, 0 if formatted | Task 8 |

**Deferred:** None. All six acceptance criteria from the story are covered.

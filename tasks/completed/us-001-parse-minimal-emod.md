# US-001: Parse a Minimal .emod File

## Progress
- [x] Task 1: Initialize Go module and project directory structure
- [x] Task 2: Define the AST types for the emod model
- [x] Task 3: Implement a lexer for .emod tokens
- [x] Task 4: Implement a recursive-descent parser for the minimal .emod grammar
- [x] Task 5: Wire up the urfave/cli CLI with a root command and validate subcommand
- [x] Task 6: Produce structured parse errors with file name, line number, and description
- [x] Task 7: Add a sample .emod fixture and end-to-end validation test

## Story Reference

**Source:** `user-stories/emod-dsl-and-diagrams.md` -- US-001

**Summary:** As a model author, I want to write a `.emod` file with a model name, one actor, one context, one aggregate, and one command-pattern slice so that the tool can parse it into a structured representation.

**Acceptance Criteria (from story):**
- `emod validate minimal.emod` exits with code 0 and prints no errors for a syntactically correct file
- The parser recognizes `model`, `actor`, `context`, `aggregate`, `slice`, `command`, `event`, and `fields` blocks
- Comments (lines starting with `#`) are ignored
- Quoted strings (e.g. model name, slice name, stream pattern) are parsed correctly
- Unrecognized keywords or unclosed braces produce an error message with file name, line number, and a description of what was expected

## Codebase Context

**Project state:** Greenfield. The repository contains only `docs/proposal.md` and `user-stories/emod-dsl-and-diagrams.md`. No Go code, no `go.mod`, no existing directory structure.

**Language and tooling:** Go 1.25, cobra for CLI. Standard Go project layout conventions (`cmd/`, `internal/`).

**DSL syntax (derived from proposal and sample file):** The `.emod` syntax is a custom brace-delimited, keyword-identified language. Key syntactic elements:
- Top-level declarations: `model "Name"`, `actor "Name"` (no braces)
- Nested block declarations: `context "Name" { ... }`, `aggregate "Name" { ... }`, `slice "Name" { ... }`
- Inner blocks: `command Identifier { ... }`, `event Identifier { ... }`, `fields { ... }`, `flow { ... }`
- Field lines inside `fields` blocks: `fieldName type modifier`
- Flow declarations inside `flow` blocks: `command -> event: Identifier -> Identifier`
- Comments: lines starting with `#`
- Quoted strings for names with spaces; bare identifiers for PascalCase names

**Relevant modules to create:**
- `cmd/emod/` -- cobra CLI entry point
- `internal/ast/` -- AST node types representing the parsed model
- `internal/lexer/` -- tokenizer for .emod input
- `internal/parser/` -- recursive-descent parser producing AST nodes
- `internal/diagnostic/` -- error types with file/line/description

**Patterns:** Since this is greenfield, no existing patterns to follow. The proposal references a hand-written recursive-descent parser approach (Phase 1 in `docs/proposal.md` line 326). The CUE-based intermediate representation is deferred to a later story (US-009).

## Tasks

### Task 1: Initialize Go module and project directory structure

**Behavior:** The repository has a valid `go.mod`, a buildable `main.go` entry point, and the conventional directory skeleton so that subsequent tasks have a place to land.

**Acceptance Criteria:**
- [ ] `go.mod` exists at the project root with module path `github.com/hpcsc/emod`
- [ ] `cmd/emod/main.go` exists and compiles with `go build ./...`
- [ ] `internal/ast/`, `internal/lexer/`, `internal/parser/`, and `internal/diagnostic/` directories exist (may contain only placeholder files or empty package declarations)
- [ ] `go test ./...` passes (no tests yet, but no compilation errors)

**Affected Files/Modules:**
- `go.mod` -- module declaration and Go version
- `cmd/emod/main.go` -- minimal main function
- `internal/ast/` -- package placeholder
- `internal/lexer/` -- package placeholder
- `internal/parser/` -- package placeholder
- `internal/diagnostic/` -- package placeholder

**Patterns to Follow:**
- Standard Go project layout: `cmd/<binary>/main.go` for entry points, `internal/` for private packages
- `docs/proposal.md` lines 261-280 for the intended architecture layers (AST/Internal Model, Validator, etc.)

**Testable:** No -- this is scaffolding only.

**Verification:** `go build ./...` succeeds; `go test ./...` exits 0.

**Depends on:** None

---

### Task 2: Define the AST types for the emod model

**Behavior:** The `internal/ast` package contains Go types representing every syntactic element in the minimal `.emod` grammar: model, actor, context, aggregate, slice, command, event, fields, field, and flow. These types are the structured representation that the parser will produce and downstream tools will consume.

**Acceptance Criteria:**
- [ ] A `Model` type exists that holds the model name, a list of actors, and a list of contexts
- [ ] An `Actor` type exists with a name field
- [ ] A `Context` type exists with a name and a list of aggregates
- [ ] An `Aggregate` type exists with a name and a list of slices
- [ ] A `Slice` type exists with a name, and lists of commands, events, and flows
- [ ] A `Command` type exists with a name and a list of fields
- [ ] An `Event` type exists with a name and a list of fields
- [ ] A `Field` type exists with name, type, and a required/optional modifier
- [ ] A `Flow` type exists representing a `command -> event` connection with source and target identifiers
- [ ] Each AST node carries position information (file name, line number, column number) for error reporting
- [ ] `go build ./...` succeeds; `go test ./...` passes

**Affected Files/Modules:**
- `internal/ast/` -- all AST node type definitions and position metadata types

**Patterns to Follow:**
- `docs/proposal.md` lines 18-148 for the full set of DSL elements (model, actor, context, aggregate, slice, command, event, fields, flow)
- The sample `.emod` file in the user story for the minimal subset of syntax this story covers

**Testable:** No -- these are data types only. Tested indirectly through parser tests in later tasks.

**Verification:** `go build ./...` succeeds.

**Depends on:** Task 1

---

### Task 3: Implement a lexer for .emod tokens

**Behavior:** The `internal/lexer` package tokenizes raw `.emod` input text into a stream of typed tokens. The lexer handles keywords, identifiers, quoted strings, braces, the arrow operator (`->`), the colon (`:`), comments, and whitespace/newlines. Each token carries its position (line, column) for downstream error reporting.

**Acceptance Criteria:**
- [ ] The lexer produces keyword tokens for: `model`, `actor`, `context`, `aggregate`, `slice`, `command`, `event`, `fields`, `flow`
- [ ] Bare identifiers (e.g. `MakeReservation`, `string`, `required`, `date`) are tokenized as identifier tokens
- [ ] Quoted strings (e.g. `"Hotel Reservation"`, `"Make Reservation"`) are tokenized as string literal tokens with the quotes stripped from the value
- [ ] Braces `{` and `}` are tokenized as open-brace and close-brace tokens
- [ ] The arrow `->` is tokenized as a single arrow token
- [ ] The colon `:` is tokenized as a colon token
- [ ] Lines starting with `#` (with optional leading whitespace) are skipped entirely
- [ ] Each token includes its line number and column number
- [ ] An EOF token is produced at the end of input
- [ ] Unrecognized characters produce an error token or a diagnostic with position information
- [ ] Unit tests cover all token types including edge cases (empty input, comment-only input, unterminated strings)

**Affected Files/Modules:**
- `internal/lexer/` -- token type definitions, lexer implementation, and lexer tests

**Patterns to Follow:**
- The sample `.emod` file provided in the user story defines the full token vocabulary for this story's scope

**Testable:** Yes

**Verification:** `go test ./internal/lexer/...` passes with tests covering each token type, comments, quoted strings, arrow operators, and error cases.

**Depends on:** Task 1

---

### Task 4: Implement a recursive-descent parser for the minimal .emod grammar

**Behavior:** The `internal/parser` package consumes the token stream from the lexer and produces an `ast.Model` value representing the parsed `.emod` file. The parser handles the minimal grammar: top-level `model` and `actor` declarations, nested `context > aggregate > slice > command/event/fields/flow` blocks.

**Acceptance Criteria:**
- [ ] Parsing the sample `.emod` file from the user story produces an `ast.Model` with the correct model name, one actor, one context, one aggregate, one slice, one command with fields, one event with fields, and one flow
- [ ] Top-level `model "Name"` declarations set the model name
- [ ] Top-level `actor "Name"` declarations add to the actors list
- [ ] `context "Name" { ... }` blocks nest aggregates
- [ ] `aggregate "Name" { ... }` blocks nest slices
- [ ] `slice "Name" { ... }` blocks nest commands, events, and flows
- [ ] `command Identifier { fields { ... } }` blocks parse the command name and its field list
- [ ] `event Identifier { fields { ... } }` blocks parse the event name and its field list
- [ ] `fields { ... }` blocks parse lines of `fieldName fieldType modifier` triples
- [ ] `flow { command -> event: Identifier -> Identifier }` blocks parse flow declarations
- [ ] The parser returns a list of parse errors (not just the first one) when multiple problems exist
- [ ] Unit tests validate successful parsing of the sample file and extraction of all elements

**Affected Files/Modules:**
- `internal/parser/` -- parser implementation and parser tests
- `internal/ast/` -- consumed (no changes expected, but may need minor adjustments discovered during implementation)
- `internal/lexer/` -- consumed (no changes expected)
- `internal/diagnostic/` -- error collection types used by the parser

**Patterns to Follow:**
- `docs/proposal.md` line 326 references a hand-written recursive descent parser approach
- The sample `.emod` file in the user story defines the grammar subset for this task

**Testable:** Yes

**Verification:** `go test ./internal/parser/...` passes with tests parsing the sample `.emod` content and asserting the resulting AST structure matches expectations.

**Depends on:** Task 2, Task 3

---

### Task 5: Wire up the cobra CLI with a root command and validate subcommand

**Behavior:** Running `emod validate <file>` reads the specified `.emod` file, runs it through the lexer and parser, and exits with code 0 (printing nothing) on success or exits with a non-zero code (printing errors to stderr) on failure. The cobra dependency is added to `go.mod`.

**Acceptance Criteria:**
- [ ] `go.mod` includes the `github.com/spf13/cobra` dependency
- [ ] Running `emod` with no arguments prints a help message listing available commands
- [ ] Running `emod validate minimal.emod` with a syntactically correct file exits with code 0 and produces no output on stdout or stderr
- [ ] Running `emod validate broken.emod` with a file containing syntax errors exits with a non-zero code and prints error messages to stderr
- [ ] Running `emod validate nonexistent.emod` exits with a non-zero code and prints a file-not-found error
- [ ] The `validate` subcommand requires exactly one positional argument (the file path)

**Affected Files/Modules:**
- `cmd/emod/main.go` -- wire cobra root command and execute
- `internal/cli/` (new) -- cobra command definitions for `root` and `validate`
- `go.mod` / `go.sum` -- cobra dependency added

**Patterns to Follow:**
- `docs/proposal.md` lines 283-294 for the intended CLI interface (`emod validate reservation.emod`)
- Standard cobra project structure: root command in one file, subcommands in separate files

**Testable:** Yes

**Verification:** `go build ./cmd/emod/` produces a binary; running the binary with a valid `.emod` fixture file exits 0; running it with an invalid file exits non-zero with error output. `go test ./...` passes.

**Depends on:** Task 4

---

### Task 6: Produce structured parse errors with file name, line number, and description

**Behavior:** When the parser encounters unrecognized keywords, unclosed braces, or other syntax errors, it produces error messages that include the file name, the line number, and a description of what was expected. The `validate` command formats and prints these errors to stderr.

**Acceptance Criteria:**
- [ ] An unrecognized keyword (e.g. `foobar { }`) produces an error like `minimal.emod:3: unrecognized keyword "foobar"; expected one of: model, actor, context`
- [ ] An unclosed brace (e.g. `context "Foo" {` with no matching `}`) produces an error like `minimal.emod:5: unclosed brace for "context" block opened at line 3`
- [ ] An unexpected token (e.g. `model 123`) produces an error like `minimal.emod:1: expected quoted string after "model", got "123"`
- [ ] Multiple errors in the same file are all reported (parser does not stop at the first error)
- [ ] Error messages are written to stderr, not stdout
- [ ] The `diagnostic` package provides a structured error type that the CLI formats for human-readable output

**Affected Files/Modules:**
- `internal/diagnostic/` -- structured error types with file, line, column, message, and expected-token fields
- `internal/parser/` -- error recovery and collection of multiple diagnostics
- `internal/cli/` -- formatting of diagnostics for stderr output

**Patterns to Follow:**
- Standard Go compiler error format: `file:line: message`
- The acceptance criteria from the user story specify "file name, line number, and a description of what was expected"

**Testable:** Yes

**Verification:** `go test ./...` passes with tests that parse malformed input and assert the resulting diagnostics contain the correct file name, line number, and descriptive messages. Running the built binary against malformed `.emod` files produces the expected stderr output.

**Depends on:** Task 4, Task 5

---

### Task 7: Add a sample .emod fixture and end-to-end validation test

**Behavior:** A complete end-to-end test exercises the full pipeline: reading a `.emod` fixture file from disk, parsing it, and verifying success. A second fixture with intentional errors validates the error path. This serves as the integration-level confidence check that the `emod validate` command works as specified.

**Acceptance Criteria:**
- [ ] A `testdata/minimal.emod` fixture file exists containing the sample from the user story (model, actor, context, aggregate, slice with command, event, fields, and flow)
- [ ] A `testdata/invalid.emod` fixture file exists containing intentional syntax errors (unrecognized keyword, unclosed brace)
- [ ] An integration test parses `testdata/minimal.emod` and asserts zero errors and a fully populated AST (model name, actor name, context name, aggregate name, slice name, command with fields, event with fields, flow connection)
- [ ] An integration test parses `testdata/invalid.emod` and asserts the returned errors contain correct file names and line numbers
- [ ] `go test ./...` passes including all integration tests

**Affected Files/Modules:**
- `testdata/minimal.emod` (new) -- valid sample fixture
- `testdata/invalid.emod` (new) -- invalid sample fixture
- `internal/parser/` or a new `internal/parser/integration_test.go` -- end-to-end parse tests using fixture files

**Patterns to Follow:**
- Go convention of using `testdata/` directories for test fixtures
- The sample `.emod` file provided in the user story for the valid fixture content

**Testable:** Yes

**Verification:** `go test ./...` passes including the integration tests that read fixture files from disk.

**Depends on:** Task 6

## Summary

**Total tasks:** 7

**Ordering rationale:** Dependency-first. The tasks build bottom-up:
1. Project scaffolding (Task 1) is the foundation everything else lands on
2. AST types (Task 2) and lexer (Task 3) are independent leaf dependencies that the parser needs
3. Parser (Task 4) depends on both AST types and lexer
4. CLI wiring (Task 5) depends on the parser to have something to call
5. Error reporting refinement (Task 6) depends on the parser and CLI being in place
6. End-to-end integration tests (Task 7) depend on everything being wired together

**Acceptance criteria coverage:**
| Story Acceptance Criterion | Covered By |
|---|---|
| `emod validate minimal.emod` exits 0, no errors | Task 5, Task 7 |
| Parser recognizes all block types (model, actor, context, aggregate, slice, command, event, fields) | Task 4, Task 7 |
| Comments (lines starting with `#`) are ignored | Task 3 |
| Quoted strings parsed correctly | Task 3, Task 4 |
| Unrecognized keywords / unclosed braces produce errors with file, line, description | Task 6, Task 7 |

**Deferred:** None. All five acceptance criteria from the story are covered.

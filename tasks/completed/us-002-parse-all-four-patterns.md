# US-002: Parse All Four Event Modeling Patterns

## Progress
- [x] Task 1: Add lexer keywords for view, automation, and translation patterns
- [x] Task 2: Add AST types for trigger, view, automation, and translation
- [x] Task 3: Parse the trigger block inside command-pattern slices
- [x] Task 4: Parse the view block inside view-pattern slices
- [x] Task 5: Parse the automation block inside automation-pattern slices
- [x] Task 6: Parse the translation block inside translation-pattern slices
- [x] Task 7: Produce specific errors for missing required sub-blocks within patterns
- [x] Task 8: Add an integration test with a fixture containing all four patterns

## Story Reference

**Source:** `user-stories/emod-dsl-and-diagrams.md` -- US-002

**Summary:** As a model author, I want to express Command, View, Automation, and Translation patterns in slices so that the full range of event modeling scenarios is captured.

**Acceptance Criteria (from story):**
- Command pattern slices accept `trigger`, `command`, `event`, and flow declarations (`command -> event: X -> Y`)
- View pattern slices accept `view` blocks with `fields` and `subscribes` lists
- Automation pattern slices accept `automation` blocks with `trigger` (event name), `command`, and optional `target context`
- Translation pattern slices accept `translation` blocks with `external_system`, `reads` (view name), `command`, and inline `event` definitions
- A file containing all four patterns parses and validates without errors
- Missing required sub-blocks within a pattern produce a specific error (e.g. "automation block requires a trigger event")

**Depends on:** US-001 (completed)

## Codebase Context

**Existing AST types (`internal/ast/ast.go`):** The AST currently models `Model`, `Actor`, `Context`, `Aggregate`, `Slice`, `Command`, `Event`, `Field`, and `Flow`. A `Slice` holds lists of `Commands`, `Events`, `Fields`, and `Flows`. There are no types for `Trigger`, `View`, `Automation`, or `Translation` yet. Each AST node carries `Position` metadata (filename, line, column).

**Existing lexer (`internal/lexer/`):** The lexer recognizes nine keywords (`model`, `actor`, `context`, `aggregate`, `slice`, `command`, `event`, `fields`, `flow`) plus identifiers, quoted strings, braces, arrow (`->`), colon, and EOF. It uses a stateless value-type `cursor` and returns tokens plus diagnostics. New keywords are needed for: `trigger`, `view`, `automation`, `translation`, `subscribes`, `target`, `external_system`, `reads`. The `[` and `]` bracket tokens are also needed for `subscribes` lists and `,` for list separators.

**Existing parser (`internal/parser/parser.go`):** A recursive-descent parser backed by a `topLevelHandler` map for dispatching on keyword type. Inside slices, parsing dispatches on `KeywordCommand`, `KeywordEvent`, and `KeywordFlow`. The parser collects multiple diagnostics rather than halting on the first error. New pattern-specific parse methods need to be registered in the slice-level dispatch loop.

**Existing test patterns:** Unit tests use `//go:build unit` tags, integration tests use `//go:build integration`. Tests follow the pattern of scanning input through the lexer, constructing a parser, calling `Parse()`, and asserting on the resulting AST and diagnostics. Integration tests read `.emod` fixtures from `internal/parser/testdata/`.

**DSL syntax for new patterns (from `docs/proposal.md` lines 37-111):**
- **Trigger:** `trigger UI "Name" { actor Identifier; reads Identifier }` -- appears inside command-pattern slices
- **View:** `view Identifier { fields { ... }; subscribes [EventA, EventB] }` -- appears inside view-pattern slices
- **Automation:** `automation Identifier { trigger EventName; command CommandName; target context ContextName }` -- appears inside automation-pattern slices
- **Translation:** `translation Identifier { external_system "Name"; reads ViewName; command CommandName; event EventName { fields { ... } } }` -- appears inside translation-pattern slices

**CLI wiring (`internal/cli/`):** Uses urfave/cli. `RunValidate` reads a file, lexes, parses, and reports diagnostics. No changes needed to CLI for this story -- the validate command already exercises the full lex-parse pipeline.

## Tasks

### Task 1: Add lexer keywords for view, automation, and translation patterns

**Behavior:** The lexer recognizes the new keywords and punctuation tokens required by the four patterns, so that the parser can dispatch on them.

**Acceptance Criteria:**
- [ ] New keyword token types exist for: `trigger`, `view`, `automation`, `translation`, `subscribes`, `target`, `external_system`, `reads`
- [ ] New punctuation token types exist for: `[` (open bracket), `]` (close bracket), `,` (comma)
- [ ] The `Kind.String()` method returns the correct string for each new token type
- [ ] The `getKeywordKind` function maps each new keyword string to its token type
- [ ] The lexer `Scan` function produces bracket and comma tokens from `[`, `]`, `,` characters in input
- [ ] Existing tests continue to pass; new unit tests cover each new token type

**Affected Files/Modules:**
- `internal/lexer/token.go` -- add new `Kind` constants and update `String()` switch
- `internal/lexer/tokenizer.go` -- add cases for `[`, `]`, `,` in the `Scan` switch, add entries in `getKeywordKind`
- `internal/lexer/tokenizer_test.go` -- add tests for the new keyword and punctuation tokens

**Patterns to Follow:**
- Follow the existing keyword constant declarations in `internal/lexer/token.go:6-16` for naming and ordering
- Follow the existing punctuation handling in `internal/lexer/tokenizer.go:39-50` for the bracket and comma cases
- Follow the existing keyword test structure in `internal/lexer/tokenizer_test.go:13-35`

**Testable:** Yes

**Verification:** `go test -tags unit ./internal/lexer/...` passes with tests covering all new token types.

**Depends on:** None

---

### Task 2: Add AST types for trigger, view, automation, and translation

**Behavior:** The `internal/ast` package contains Go types representing trigger blocks, view blocks (with fields and subscribes), automation blocks (with trigger event, command, and optional target context), and translation blocks (with external system, reads, command, and inline event). The `Slice` type is extended to hold these new constructs.

**Acceptance Criteria:**
- [ ] A `Trigger` type exists with fields for the trigger kind (e.g. "UI"), name, actor reference, reads reference, and position metadata
- [ ] A `View` type exists with a name, a list of fields, a list of subscribed event names, and position metadata
- [ ] An `Automation` type exists with a name, trigger event name, command name, optional target context name, and position metadata
- [ ] A `Translation` type exists with a name, external system name, reads view name, command name, an optional inline event, and position metadata
- [ ] The `Slice` type gains optional fields for `Trigger`, `Views`, `Automations`, and `Translations`
- [ ] `go build ./...` succeeds

**Affected Files/Modules:**
- `internal/ast/ast.go` -- add new types (`Trigger`, `View`, `Automation`, `Translation`) and extend `Slice`

**Patterns to Follow:**
- Follow the existing AST node structure in `internal/ast/ast.go:48-78` for field naming, position metadata, and pointer semantics (e.g. `*Command`, `*Event`)
- Follow the DSL syntax from `docs/proposal.md:37-111` for determining which fields each type needs

**Testable:** No -- these are data types only. Tested indirectly through parser tests in later tasks.

**Verification:** `go build ./...` succeeds.

**Depends on:** None

---

### Task 3: Parse the trigger block inside command-pattern slices

**Behavior:** The parser recognizes `trigger` blocks within slices and produces the corresponding `ast.Trigger` node. This satisfies the command-pattern acceptance criterion that slices accept `trigger` declarations alongside the existing `command`, `event`, and `flow` blocks.

**Acceptance Criteria:**
- [ ] A slice containing a `trigger` block (e.g. `trigger UI "Reservation Form" { actor Guest; reads AvailableRoomsView }`) parses into an `ast.Trigger` with the correct kind, name, actor, and reads values
- [ ] A trigger block with only the kind and name (no nested actor/reads) parses without error
- [ ] The trigger is stored in the slice AST node
- [ ] Existing command-pattern slices (command, event, flow) continue to parse correctly alongside triggers

**Affected Files/Modules:**
- `internal/parser/parser.go` -- add `parseTrigger` method and register `KeywordTrigger` in the slice-level dispatch loop (around line 204-218)
- `internal/parser/parser_test.go` -- add unit tests for trigger parsing

**Patterns to Follow:**
- Follow the existing slice-inner-block dispatch pattern in `internal/parser/parser.go:204-218` for adding the new keyword case
- Follow the existing `parseCommand` method structure in `internal/parser/parser.go:231-268` for parsing a named block with nested content

**Testable:** Yes

**Verification:** `go test -tags unit ./internal/parser/...` passes with tests covering trigger blocks with and without nested actor/reads content.

**Depends on:** Task 1, Task 2

---

### Task 4: Parse the view block inside view-pattern slices

**Behavior:** The parser recognizes `view` blocks within slices and produces the corresponding `ast.View` node, including its fields list and subscribes list. This satisfies the view-pattern acceptance criterion.

**Acceptance Criteria:**
- [ ] A slice containing a `view` block with `fields` and `subscribes` (e.g. `view AvailableRoomsView { fields { roomId RoomID } subscribes [RoomReserved, GuestCheckedOut] }`) parses into an `ast.View` with the correct name, fields, and subscribed event names
- [ ] A view block with only `fields` (no `subscribes`) parses without error
- [ ] A view block with only `subscribes` (no `fields`) parses without error
- [ ] The subscribes list correctly handles multiple comma-separated identifiers within brackets
- [ ] The view is stored in the slice AST node

**Affected Files/Modules:**
- `internal/parser/parser.go` -- add `parseView` and `parseSubscribes` methods, register `KeywordView` in the slice-level dispatch loop
- `internal/parser/parser_test.go` -- add unit tests for view parsing with fields, subscribes, and combinations

**Patterns to Follow:**
- Follow the existing `parseCommand` structure in `internal/parser/parser.go:231-268` for parsing a named block with nested content
- Follow the `parseFields` method in `internal/parser/parser.go:309-337` for parsing the fields sub-block
- The subscribes list uses bracket tokens `[` and `]` with comma-separated identifiers -- this is a new parse pattern not yet in the codebase

**Testable:** Yes

**Verification:** `go test -tags unit ./internal/parser/...` passes with tests covering view blocks with various combinations of fields and subscribes.

**Depends on:** Task 1, Task 2

---

### Task 5: Parse the automation block inside automation-pattern slices

**Behavior:** The parser recognizes `automation` blocks within slices and produces the corresponding `ast.Automation` node, including its trigger event, command, and optional target context. This satisfies the automation-pattern acceptance criterion.

**Acceptance Criteria:**
- [ ] A slice containing an `automation` block (e.g. `automation ConfirmationEmailReactor { trigger RoomReserved; command SendConfirmationEmail; target context Notifications }`) parses into an `ast.Automation` with the correct name, trigger event, command, and target context
- [ ] An automation block without a `target context` parses without error, leaving the target context empty
- [ ] The automation is stored in the slice AST node
- [ ] The `trigger` keyword inside an automation block is interpreted as the trigger event name (an identifier), not as the trigger block type used in command-pattern slices

**Affected Files/Modules:**
- `internal/parser/parser.go` -- add `parseAutomation` method, register `KeywordAutomation` in the slice-level dispatch loop
- `internal/parser/parser_test.go` -- add unit tests for automation parsing with and without target context

**Patterns to Follow:**
- Follow the existing `parseCommand` structure in `internal/parser/parser.go:231-268` for parsing a named block
- Follow the DSL syntax in `docs/proposal.md:85-91` for the expected automation block structure
- The `target context` sub-block uses two consecutive keywords (`target` then `context`) followed by an identifier -- note the parser already has `KeywordContext` which needs disambiguation in this nested scope

**Testable:** Yes

**Verification:** `go test -tags unit ./internal/parser/...` passes with tests covering automation blocks with and without target context.

**Depends on:** Task 1, Task 2

---

### Task 6: Parse the translation block inside translation-pattern slices

**Behavior:** The parser recognizes `translation` blocks within slices and produces the corresponding `ast.Translation` node, including its external system, reads view, command, and optional inline event definition. This satisfies the translation-pattern acceptance criterion.

**Acceptance Criteria:**
- [ ] A slice containing a `translation` block (e.g. `translation BookingComImport { external_system "Booking.com API"; reads BookingComWebhookView; command ImportExternalReservation; event ExternalReservationImported { fields { ... } } }`) parses into an `ast.Translation` with the correct name, external system, reads view, command, and inline event with fields
- [ ] A translation block without an inline event parses without error
- [ ] The inline event inside a translation reuses the existing `parseEvent` method
- [ ] The translation is stored in the slice AST node

**Affected Files/Modules:**
- `internal/parser/parser.go` -- add `parseTranslation` method, register `KeywordTranslation` in the slice-level dispatch loop
- `internal/parser/parser_test.go` -- add unit tests for translation parsing with and without inline events

**Patterns to Follow:**
- Follow the existing `parseCommand` structure in `internal/parser/parser.go:231-268` for parsing a named block
- Reuse the existing `parseEvent` method in `internal/parser/parser.go:270-307` for the inline event definition
- Follow the DSL syntax in `docs/proposal.md:94-111` for the expected translation block structure

**Testable:** Yes

**Verification:** `go test -tags unit ./internal/parser/...` passes with tests covering translation blocks with and without inline events.

**Depends on:** Task 1, Task 2

---

### Task 7: Produce specific errors for missing required sub-blocks within patterns

**Behavior:** When a pattern block is missing a required sub-block, the parser produces a descriptive error message naming the missing element. This satisfies the acceptance criterion about specific error messages for incomplete patterns.

**Acceptance Criteria:**
- [ ] An `automation` block missing a `trigger` entry produces an error containing "automation block requires a trigger event"
- [ ] An `automation` block missing a `command` entry produces an error containing "automation block requires a command"
- [ ] A `translation` block missing an `external_system` entry produces an error containing "translation block requires an external_system"
- [ ] A `translation` block missing a `reads` entry produces an error containing "translation block requires a reads view"
- [ ] A `translation` block missing a `command` entry produces an error containing "translation block requires a command"
- [ ] A `view` block missing both `fields` and `subscribes` produces an error containing "view block requires fields or subscribes"
- [ ] Error messages include the filename and line number of the block that is missing the required sub-block

**Affected Files/Modules:**
- `internal/parser/parser.go` -- add post-parse validation within `parseAutomation`, `parseTranslation`, and `parseView` methods to check for required sub-blocks and emit diagnostics
- `internal/parser/parser_test.go` -- add unit tests that parse incomplete blocks and assert on the specific error messages

**Patterns to Follow:**
- Follow the existing error reporting pattern in `internal/parser/parser.go:486-494` for constructing and appending diagnostics
- Follow the error assertion pattern in `internal/parser/parser_test.go:273-304` for testing specific error messages

**Testable:** Yes

**Verification:** `go test -tags unit ./internal/parser/...` passes with tests that parse incomplete pattern blocks and assert the diagnostics contain the expected specific messages with filenames and line numbers.

**Depends on:** Task 4, Task 5, Task 6

---

### Task 8: Add an integration test with a fixture containing all four patterns

**Behavior:** A comprehensive `.emod` fixture file exercises all four patterns (command, view, automation, translation) in a single model. An integration test parses this file and verifies the full AST is populated correctly. This satisfies the acceptance criterion that a file containing all four patterns parses and validates without errors.

**Acceptance Criteria:**
- [ ] A `testdata/all_patterns.emod` fixture file exists containing a model with at least one slice of each pattern type: command (with trigger, command, event, flow), view (with fields and subscribes), automation (with trigger, command, and target context), and translation (with external_system, reads, command, and inline event)
- [ ] An integration test parses `testdata/all_patterns.emod` and asserts zero lexer diagnostics and zero parser diagnostics
- [ ] The integration test asserts the correct number and type of elements in each slice: trigger present in the command-pattern slice, view with fields and subscribes in the view-pattern slice, automation with all three sub-blocks in the automation-pattern slice, and translation with external system, reads, command, and inline event in the translation-pattern slice
- [ ] The existing `minimal.emod` and `invalid.emod` integration tests continue to pass

**Affected Files/Modules:**
- `internal/parser/testdata/all_patterns.emod` (new) -- fixture file with all four pattern types
- `internal/parser/integration_test.go` -- add a new test case for the all-patterns fixture

**Patterns to Follow:**
- Follow the existing integration test structure in `internal/parser/integration_test.go:14-81` for reading fixtures and asserting on AST content
- Follow the existing fixture file format in `internal/parser/testdata/minimal.emod` for syntax style
- Base the fixture content on the sample DSL in `docs/proposal.md:20-148`

**Testable:** Yes

**Verification:** `go test -tags integration ./internal/parser/...` passes including the new all-patterns test case alongside the existing minimal and invalid fixture tests.

**Depends on:** Task 3, Task 4, Task 5, Task 6, Task 7

## Summary

**Total tasks:** 8

**Ordering rationale:** Dependency-first with parallelism where possible. Tasks 1 (lexer) and 2 (AST types) are independent leaves that can proceed in parallel. Tasks 3-6 each implement one pattern and depend on both Tasks 1 and 2 being complete; they are independent of each other and can proceed in parallel. Task 7 (validation errors for missing sub-blocks) depends on the pattern-parsing tasks 4-6 being in place. Task 8 (integration test) depends on all pattern tasks and validation.

**Acceptance criteria coverage:**

| Story Acceptance Criterion | Covered By |
|---|---|
| Command pattern slices accept `trigger`, `command`, `event`, and flow declarations | Task 3 (trigger is the new part; command/event/flow already work from US-001) |
| View pattern slices accept `view` blocks with `fields` and `subscribes` lists | Task 4 |
| Automation pattern slices accept `automation` blocks with `trigger`, `command`, and optional `target context` | Task 5 |
| Translation pattern slices accept `translation` blocks with `external_system`, `reads`, `command`, and inline `event` | Task 6 |
| A file containing all four patterns parses and validates without errors | Task 8 |
| Missing required sub-blocks within a pattern produce a specific error | Task 7 |

**Deferred:** None. All six acceptance criteria from the story are covered.

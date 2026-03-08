# US-003: Parse Cross-Context References and External Sources

## Progress
- [x] Task 1: Add `source` and `external` keywords to the lexer
- [x] Task 2: Add external source metadata to the Event AST node
- [x] Task 3: Parse `source external "Provider Name"` inside event blocks
- [x] Task 4: Create a semantic validation pass and validate automation target context references
- [x] Task 5: Validate that referenced command names exist in the model
- [x] Task 6: Validate that referenced event names exist in the model
- [x] Task 7: Add a multi-context integration test fixture and end-to-end validation tests

## Story Reference

**Source:** `user-stories/emod-dsl-and-diagrams.md` -- US-003

**Summary:** As a model author, I want to reference aggregates and commands across bounded contexts, and mark events as originating from external systems, so that multi-context models and external integrations are expressible.

**Acceptance Criteria (from story):**
- An automation's `target context` can reference a different context defined in the same model
- An event with `source external "Provider Name"` is parsed and stored with its external source metadata
- Referencing a context name that does not exist in the model produces a validation error
- Referencing a command or event name that does not exist produces a validation error

**Depends on:** US-002 (completed)

## Codebase Context

**Current automation parsing (`internal/parser/parser.go:440-528`):** The parser already recognizes `target context <Identifier>` inside automation blocks and stores the value in `ast.Automation.TargetContext` and `ast.Automation.TargetContextPos`. The lexer already has `KeywordTarget` and `KeywordContext` tokens. However, there is no post-parse validation that the referenced context name actually exists in the model.

**Current event parsing (`internal/parser/parser.go:351-388`):** Events are parsed with a name and a `fields` block. There is no support for a `source external "Provider Name"` clause. The lexer does not have `source` or `external` keywords. The `ast.Event` type has no fields for external source metadata.

**Current AST (`internal/ast/ast.go`):** The `Event` type has `Name`, `NamePos`, `Fields`, `OpenPos`, `ClosePos`. It needs additional fields for external source metadata. The `Model` type holds `Contexts` as a slice, which can be iterated to build a name-lookup index.

**Current lexer (`internal/lexer/token.go`, `internal/lexer/tokenizer.go`):** Supports 17 keyword types. New keywords `source` and `external` are needed. The `getKeywordKind` function maps keyword strings to token types.

**Current validation (`internal/cli/validate.go`):** `RunValidate` runs lexer + parser and reports diagnostics. There is no semantic validation phase that checks cross-references after parsing. A new validation module is needed that accepts an `*ast.Model` and returns `[]*diagnostic.Entry`.

**DSL syntax reference (`docs/proposal.md:137-145`):** Shows `source external "SendGrid Webhook"` as a clause within event blocks, alongside `fields`.

**Existing test patterns:** Unit tests use `//go:build unit`, integration tests use `//go:build integration`. Parser tests follow a consistent pattern of scanning, parsing, and asserting on AST nodes and diagnostics. The existing `all_patterns.emod` fixture has a single context; a multi-context fixture is needed.

## Tasks

### Task 1: Add `source` and `external` keywords to the lexer

**Behavior:** The lexer recognizes `source` and `external` as keyword tokens so that the parser can dispatch on them when parsing event blocks.

**Acceptance Criteria:**
- [ ] New keyword token types `KeywordSource` and `KeywordExternal` exist in `lexer.Kind`
- [ ] The `Kind.String()` method returns `"source"` and `"external"` respectively
- [ ] The `getKeywordKind` function maps `"source"` and `"external"` to their token types
- [ ] Input containing `source external "Provider"` produces three tokens: `KeywordSource`, `KeywordExternal`, `String("Provider")`
- [ ] Existing tests continue to pass

**Affected Files/Modules:**
- `internal/lexer/token.go` -- add `KeywordSource` and `KeywordExternal` constants, update `String()` switch
- `internal/lexer/tokenizer.go` -- add entries in `getKeywordKind`
- `internal/lexer/tokenizer_test.go` -- add tests for the new keyword tokens

**Patterns to Follow:**
- Follow the existing keyword constant declarations in `internal/lexer/token.go:6-17` for naming and ordering
- Follow the existing keyword mapping in `internal/lexer/tokenizer.go:169-207` for `getKeywordKind` entries
- Follow the existing lexer test structure in `internal/lexer/tokenizer_test.go` for new keyword tests

**Testable:** Yes

**Verification:** `go test -tags unit ./internal/lexer/...` passes with tests covering both new keyword tokens.

**Depends on:** None

---

### Task 2: Add external source metadata to the Event AST node

**Behavior:** The `ast.Event` type gains fields to represent an optional external source, so that events originating from external systems can carry their provider name and position metadata.

**Acceptance Criteria:**
- [ ] The `ast.Event` type has a `Source` field for the source kind (e.g. "external")
- [ ] The `ast.Event` type has a `SourcePos` field for position metadata of the source clause
- [ ] The `ast.Event` type has an `ExternalName` field for the external provider name (e.g. "SendGrid Webhook")
- [ ] The `ast.Event` type has an `ExternalNamePos` field for position metadata of the external name
- [ ] `go build ./...` succeeds

**Affected Files/Modules:**
- `internal/ast/ast.go` -- add `Source`, `SourcePos`, `ExternalName`, `ExternalNamePos` fields to the `Event` struct

**Patterns to Follow:**
- Follow the existing field naming pattern in `internal/ast/ast.go:60-66` where each semantic field has a corresponding `Pos` field
- Follow the naming convention of `ExternalSystem`/`ExternalPos` in `ast.Translation` at `internal/ast/ast.go:119-123`

**Testable:** No -- data types only. Tested indirectly through parser tests in Task 3.

**Verification:** `go build ./...` succeeds.

**Depends on:** None

---

### Task 3: Parse `source external "Provider Name"` inside event blocks

**Behavior:** The parser recognizes the `source external "Provider Name"` clause within event blocks and populates the corresponding AST fields. Events without a `source` clause continue to parse as before.

**Acceptance Criteria:**
- [ ] An event containing `source external "SendGrid Webhook"` is parsed with `Source` set to `"external"` and `ExternalName` set to `"SendGrid Webhook"`
- [ ] An event without a `source` clause parses with empty `Source` and `ExternalName` fields
- [ ] The `source` clause can appear alongside `fields` in any order within the event block
- [ ] An event with `source` but without `external` keyword produces a parse error
- [ ] An event with `source external` but without a quoted string produces a parse error

**Affected Files/Modules:**
- `internal/parser/parser.go` -- extend `parseEvent` to handle `KeywordSource` followed by `KeywordExternal` followed by a `String` token
- `internal/parser/parser_test.go` -- add unit tests for events with and without `source external` clauses

**Patterns to Follow:**
- Follow the existing inner-block dispatch pattern in `parseEvent` at `internal/parser/parser.go:371-377` for adding a new keyword case
- Follow the `external_system` parsing in `parseTranslation` at `internal/parser/parser.go:551-559` for parsing a keyword followed by a quoted string token

**Testable:** Yes

**Verification:** `go test -tags unit ./internal/parser/...` passes with tests covering event blocks with `source external`, without source, and with malformed source clauses.

**Depends on:** Task 1, Task 2

---

### Task 4: Create a semantic validation pass and validate automation target context references

**Behavior:** A new validation module accepts a parsed `*ast.Model` and checks that automation `target context` references point to context names that exist in the model. This introduces the semantic validation infrastructure that subsequent tasks build on. The `RunValidate` function is updated to invoke this validation after parsing.

**Acceptance Criteria:**
- [ ] An automation with `target context Notifications` where a context named `"Notifications"` exists in the model produces no validation error
- [ ] An automation with `target context NonExistent` where no context with that name exists produces a validation error with the message containing the referenced name and indicating it does not exist
- [ ] An automation without a `target context` produces no validation error for this check
- [ ] The validation error includes the filename, line, and column of the `target context` reference
- [ ] `RunValidate` invokes the semantic validation after parsing and reports any validation diagnostics alongside parse diagnostics

**Affected Files/Modules:**
- `internal/validator/` (new package) -- create a validation module with a function that accepts `*ast.Model` and returns `[]*diagnostic.Entry`
- `internal/cli/validate.go` -- invoke the new validator after parsing and append its diagnostics

**Patterns to Follow:**
- Follow the diagnostic reporting pattern used in `internal/parser/parser.go:832-839` for constructing `diagnostic.Entry` values
- Follow the module organization pattern of the existing `internal/parser/` and `internal/lexer/` packages
- Follow the `RunValidate` pipeline pattern in `internal/cli/validate.go:15-40` for integrating the new validation step

**Testable:** Yes

**Verification:** `go test -tags unit ./internal/validator/...` passes with tests for both valid and invalid context references. `go test -tags unit ./internal/cli/...` continues to pass.

**Depends on:** None (operates on existing AST types; does not require Tasks 1-3)

---

### Task 5: Validate that referenced command names exist in the model

**Behavior:** The semantic validator checks that command names referenced in automations and translations exist as defined commands somewhere in the model, producing a validation error when they do not.

**Acceptance Criteria:**
- [ ] An automation referencing a command that is defined in any context/aggregate/slice of the model produces no error
- [ ] An automation referencing a command that does not exist anywhere in the model produces a validation error with a message containing the command name
- [ ] A translation referencing a command that does not exist produces a validation error
- [ ] The validation error includes the filename, line, and column of the command reference
- [ ] Commands defined in the same context and in different contexts are both found by the lookup

**Affected Files/Modules:**
- `internal/validator/` -- extend the validation function to collect all defined command names from the model and check automation and translation command references against them

**Patterns to Follow:**
- Follow the validation pattern established in Task 4 for iterating over the model and emitting `diagnostic.Entry` values
- Follow the AST traversal structure: `Model.Contexts[].Aggregates[].Slices[].Commands[]` for collecting defined command names

**Testable:** Yes

**Verification:** `go test -tags unit ./internal/validator/...` passes with tests covering valid references, missing command references in automations, and missing command references in translations.

**Depends on:** Task 4

---

### Task 6: Validate that referenced event names exist in the model

**Behavior:** The semantic validator checks that event names referenced in automation triggers, view subscribes lists, and flow entries exist as defined events somewhere in the model, producing a validation error when they do not.

**Acceptance Criteria:**
- [ ] An automation trigger referencing an event that exists in the model produces no error
- [ ] An automation trigger referencing a non-existent event produces a validation error with a message containing the event name
- [ ] A view `subscribes` entry referencing a non-existent event produces a validation error
- [ ] A flow entry referencing a non-existent event produces a validation error
- [ ] The validation error includes the filename, line, and column of the event reference
- [ ] Events defined in any context are found by the lookup

**Affected Files/Modules:**
- `internal/validator/` -- extend the validation function to collect all defined event names from the model and check automation triggers, view subscribes, and flow event references against them

**Patterns to Follow:**
- Follow the validation pattern established in Tasks 4 and 5 for model traversal and diagnostic emission
- Follow the AST traversal structure: `Model.Contexts[].Aggregates[].Slices[].Events[]` for collecting defined event names, also checking inline events in translations (`Translation.Event`)

**Testable:** Yes

**Verification:** `go test -tags unit ./internal/validator/...` passes with tests covering valid references and missing event references in automations, views, and flows.

**Depends on:** Task 4

---

### Task 7: Add a multi-context integration test fixture and end-to-end validation tests

**Behavior:** A multi-context `.emod` fixture file exercises cross-context references (automation targeting another context) and external source events. Integration tests verify that valid multi-context models parse and validate without errors, and that invalid references produce the expected validation errors. The CLI validate command is exercised end-to-end.

**Acceptance Criteria:**
- [ ] A `testdata/multi_context.emod` fixture file exists with at least two contexts, where one context's automation targets the other context by name, and one event has a `source external` clause
- [ ] An integration test parses and validates the multi-context fixture with zero diagnostics
- [ ] An integration test or unit test verifies that an automation referencing a non-existent context produces a validation error
- [ ] An integration test or unit test verifies that referencing a non-existent command or event produces a validation error
- [ ] The existing `all_patterns.emod` and `minimal.emod` integration tests continue to pass
- [ ] The CLI `validate_test.go` tests continue to pass

**Affected Files/Modules:**
- `internal/parser/testdata/multi_context.emod` (new) -- multi-context fixture with cross-references and external source events
- `internal/parser/integration_test.go` -- add integration test for the multi-context fixture
- `internal/cli/validate_test.go` -- add or update tests to cover validation errors for invalid references

**Patterns to Follow:**
- Follow the existing integration test structure in `internal/parser/integration_test.go:14-81` for reading fixtures and asserting on AST content
- Follow the existing fixture format in `internal/parser/testdata/all_patterns.emod`
- Follow the existing CLI test pattern in `internal/cli/validate_test.go:84-115` for testing `RunValidate`

**Testable:** Yes

**Verification:** `go test -tags integration ./internal/parser/...` and `go test -tags unit ./internal/cli/...` both pass with the new and existing test cases.

**Depends on:** Task 3, Task 4, Task 5, Task 6

## Summary

**Total tasks:** 7

**Ordering rationale:** Dependency-first with parallelism where possible. Tasks 1 (lexer keywords) and 2 (AST fields) are independent leaf tasks that can proceed in parallel. Task 3 (parse `source external`) depends on Tasks 1 and 2. Task 4 (semantic validator infrastructure + context validation) is independent of Tasks 1-3 and can proceed in parallel with them since it operates on the existing AST. Tasks 5 and 6 (command and event reference validation) both depend on Task 4 but are independent of each other. Task 7 (integration tests) depends on all preceding tasks.

**Acceptance criteria coverage:**

| Story Acceptance Criterion | Covered By |
|---|---|
| An automation's `target context` can reference a different context defined in the same model | Task 4 (validates context exists), Task 7 (multi-context fixture proves it works end-to-end) |
| An event with `source external "Provider Name"` is parsed and stored with its external source metadata | Tasks 1, 2, 3 (lexer + AST + parser), Task 7 (fixture includes external source event) |
| Referencing a context name that does not exist in the model produces a validation error | Task 4 |
| Referencing a command or event name that does not exist produces a validation error | Task 5 (commands), Task 6 (events) |

**Deferred:** None. All four acceptance criteria from the story are covered.

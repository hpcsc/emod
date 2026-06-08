# US-IDE-006: LSP Keyword Completion

## Progress
- [x] Task 1: Add LSP completion protocol types
- [ ] Task 2: Add completion logic
- [ ] Task 3: Wire completion handler into LSP server

---

## Story Reference

**US-IDE-006:** As a model author, I want context-aware keyword suggestions as I type so that I can discover valid keywords for each block without consulting documentation.

File: `user-stories/rate-limiting.md` (story was provided inline)

---

## Codebase Context

The LSP server lives in `internal/lsp/` with the following structure:
- `server.go` — Main server loop; `dispatch()` routes messages by method via a `switch`. Currently handles: `initialize`, `initialized`, `textDocument/didOpen`, `textDocument/didChange`, `shutdown`. Unknown methods return `-32601`.
- `protocol.go` — LSP protocol types (`Position`, `Range`, `Diagnostic`, `InitializeResult`, `ServerCapabilities`, etc.). `ServerCapabilities` currently has only `TextDocumentSync`. No completion types exist.
- `document.go` — `DocumentManager` tracks open documents by URI with in-memory content. Has `GetContent(uri)` method.
- `transport.go` — JSON-RPC content-length framing. `ReadMessage`/`WriteMessage`.
- `server_test.go` — Tests use `io.Pipe` I/O plumbing (`startServer` helper), follow behavior-driven patterns with `testify/require`.
- `diagnostics.go` — `ConvertDiagnostics()` — pattern for standalone exported function in this package.

The AST lives in `internal/ast/ast.go` with types: `Model`, `Actor`, `Context`, `Aggregate`, `Slice`, `Command`, `Event`, `Field`, `Flow`, `Trigger`, `View`, `Automation`, `Translation`. Container types (`Context`, `Aggregate`, `Slice`, `Command`, `Event`) have `OpenPos`/`ClosePos` fields that record brace positions, enabling cursor-context resolution.

The lexer in `internal/lexer/token.go` defines all block keywords (`KeywordModel`, `KeywordActor`, `KeywordContext`, `KeywordAggregate`, `KeywordSlice`, `KeywordCommand`, `KeywordEvent`, `KeywordFields`, `KeywordFlow`, `KeywordTrigger`, `KeywordView`, `KeywordAutomation`, `KeywordTranslation`). Field type values (`string`, `date`, `timestamp`, `int`) and modifier values (`required`, `optional`) are **not** lexer keywords — they are parsed as identifiers and must be provided by the completion system as special values.

The parser in `internal/parser/parser.go` produces an AST from tokens. The AST can be used to determine which block the cursor falls into by comparing cursor line/character against each node's `OpenPos`/`ClosePos`.

Testing pattern: the `diagnostics_test.go` and `protocol_test.go` files demonstrate how to test exported functions (call with inputs, assert on outputs) and protocol type serialization (marshal/unmarshal with `require.JSONEq`).

---

## Tasks

### Task 1: Add LSP completion protocol types

**Language:** Go

**Behavior:** The LSP protocol layer supports the completion request/response types needed by the `textDocument/completion` method.

**Acceptance Criteria:**
- [ ] `CompletionItemKind` enum is defined with at least `KeywordCompletion` as a kind value
- [ ] `CompletionItem` struct is defined with `Label`, `Kind`, and optional `Detail`/`Documentation` fields, matching LSP spec JSON shape
- [ ] `CompletionList` struct is defined with `IsIncomplete` and `Items` fields
- [ ] `CompletionParams` struct is defined with `TextDocument` and `Position` fields
- [ ] `CompletionOptions` struct is defined with `TriggerCharacters`
- [ ] `ServerCapabilities` gains an optional `CompletionProvider` field of type `CompletionOptions`

**Affected Files/Modules:**
- `internal/lsp/protocol.go` — Add `CompletionItemKind`, `CompletionItem`, `CompletionList`, `CompletionParams`, `CompletionOptions`, and update `ServerCapabilities` with `CompletionProvider`
- `internal/lsp/protocol_test.go` — Add tests for JSON serialization/deserialization of new types

**Patterns to Follow:**
- Existing type definitions in `internal/lsp/protocol.go` (e.g., `Diagnostic`, `InitializeResult`, `ServerCapabilities`) — same struct tag style and field naming

**Testable:** Yes — types are testable through JSON marshal/unmarshal, following the same pattern as the existing `TestProtocolTypes` tests in `protocol_test.go`

**Verification:** `go test -tags unit <package>` passes for the `lsp` package

**Depends on:** None

---

### Task 2: Add completion logic

**Language:** Go

**Behavior:** Given document text and a cursor position, return context-appropriate keyword completions. The logic determines which block the cursor is in (top-level, `context`, `aggregate`, `slice`, `command`/`event`, or `fields`) and returns the valid keywords for that context.

**Acceptance Criteria:**
- [ ] At the top level, completions include `model`, `actor`, `context`
- [ ] Inside a `context {}` block, completions include `aggregate`
- [ ] Inside an `aggregate {}` block, completions include `slice`
- [ ] Inside a `slice {}` block, completions include `command`, `event`, `trigger`, `view`, `automation`, `translation`, `flow`
- [ ] Inside a `command {}` or `event {}` block, completions include `fields`
- [ ] Inside a `fields {}` block, field type completions include `string`, `date`, `timestamp`, `int`; modifier completions include `required`, `optional`
- [ ] An empty or unparseable document returns top-level completions as a reasonable fallback

**Affected Files/Modules:**
- `internal/lsp/completer.go` — New file with exported function for completion logic
- `internal/lsp/completer_test.go` — New file with tests covering all context contexts
- `internal/ast/ast.go` — Read reference only (types already defined)
- `internal/parser/parser.go` — Read reference only (used to produce AST from text)

**Patterns to Follow:**
- Standalone exported function pattern in `internal/lsp/diagnostics.go:9` (`ConvertDiagnostics`) — a pure function that takes inputs and returns outputs, tested through exported API
- AST node `OpenPos`/`ClosePos` fields on `Context`, `Aggregate`, `Slice`, `Command`, `Event` (in `internal/ast/ast.go`) enable cursor-context resolution by checking if cursor line/character falls within a node's brace range

**Testable:** Yes — the completion function is a pure function (text + cursor position → completion items), fully testable through exported API

**Verification:** `go test -tags unit <package>` passes for the `lsp` package

**Depends on:** Task 1 (uses `CompletionItem` and related types)

---

### Task 3: Wire completion handler into LSP server

**Language:** Go

**Behavior:** The LSP server responds to `textDocument/completion` requests by returning context-aware keyword completions. The server advertises the completion capability during initialization.

**Acceptance Criteria:**
- [ ] The LSP server responds to `textDocument/completion` requests (the AC from the story)
- [ ] Server advertises `CompletionProvider` capability in the initialize response
- [ ] Requesting completion returns a `CompletionList` with appropriate items
- [ ] Unknown document URI returns an error
- [ ] An incomplete/partial document at the cursor position still returns reasonable completions (graceful fallback)

**Affected Files/Modules:**
- `internal/lsp/server.go` — Add `textDocument/completion` case to `dispatch()`; add `handleCompletion()` method; update `handleInitialize()` to include `CompletionProvider` in capabilities
- `internal/lsp/server_test.go` — Add tests for completion requests through the server I/O interface

**Patterns to Follow:**
- Existing handler pattern in `internal/lsp/server.go` — e.g., `handleDidOpen` (line 103) shows the pattern: unmarshal params, interact with documents, write result message
- Test pattern in `internal/lsp/server_test.go` — `startServer` helper (line 26), write request via `writeMsg`, read response via `readMsg`, assert on response fields

**Testable:** Yes — test through the server's public I/O interface (io.Pipe), following the existing server_test.go patterns

**Verification:** `go test -tags unit <package>` passes for the `lsp` package

**Depends on:** Task 1 (uses `CompletionParams`, `CompletionList`, `CompletionOptions`), Task 2 (uses completion logic)

---

## Summary

- **Total tasks:** 3
- **Language:** All tasks use Go
- **Task ordering rationale:** Dependency-first. Task 1 adds the protocol types that both subsequent tasks need. Task 2 adds the core business logic (context detection, keyword mapping) which can be thoroughly tested in isolation. Task 3 wires it into the server, which is the final integrating slice. This ordering minimizes risk: the types are simple, the logic is the highest-value testing target, and wiring is the last step.
- **Acceptance criteria coverage:** All 8 AC items from the story are covered:
  - AC 1 (server responds to `textDocument/completion`): Task 3
  - AC 2 (top level: model, actor, context): Task 2
  - AC 3 (context block: aggregate): Task 2
  - AC 4 (aggregate block: slice): Task 2
  - AC 5 (slice block: command, event, trigger, view, automation, translation, flow): Task 2
  - AC 6 (command/event block: fields): Task 2
  - AC 7 (fields block: field types and modifiers): Task 2
- **No acceptance criteria deferred:** All criteria are covered.
- **Test command:** `go test -tags unit $(go list ./... | grep -v /cmd/emod-wasm)`

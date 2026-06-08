## Progress
- [x] Task 1: Add `textDocument/formatting` handler to LSP server

## Story Reference
US-IDE-008: LSP format on save — *As a model author, I want my `.emod` file to be auto-formatted when I save so that models stay consistently formatted without running `emod fmt` manually.*

## Codebase Context

**Exploration findings:**

The LSP server at `internal/lsp/server.go` uses a method-dispatch pattern (`dispatch` switch on `msg.Method`) and has existing handlers for `didOpen`, `didChange`, `completion`, `definition`, and `shutdown`. Handlers follow a consistent pattern: unmarshal params → look up document via `DocumentManager` → perform logic → marshal result → write response via `writeMessage`.

Capabilities are advertised in `handleInitialize` via the `ServerCapabilities` struct in `protocol.go`. The diagnostics pipeline (`pushDiagnostics`) already runs the full `lex → parse` chain, which is the same pipeline needed for formatting.

The formatter at `internal/formatter/formatter.go` provides `func Format(model *ast.Model) string` and is already tested for idempotency, comment preservation, blank-line normalization, and field alignment. It expects an `*ast.Model`, which requires lexing + parsing the document text first.

LSP protocol types live in `internal/lsp/protocol.go`. There is currently no `TextEdit` type — it needs to be added. Server tests live in `internal/lsp/server_test.go` and use a `serverPair` fixture with `io.Pipe` to send/receive JSON-RPC messages.

No editor-side (TypeScript) changes are needed — VS Code automatically enables "format on save" for any language server that advertises `DocumentFormattingProvider`.

## Tasks

### Task 1: Add `textDocument/formatting` handler to LSP server

**Behavior:** The LSP server responds to `textDocument/formatting` requests by formatting the document content using the existing `emod fmt` formatter and returning a full-document `TextEdit` with the formatted output.

**Acceptance Criteria:**
- [ ] The server advertises `documentFormattingProvider` capability in the `initialize` response
- [ ] The server responds to `textDocument/formatting` requests with a `TextEdit[]` result
- [ ] The formatted output matches what `formatter.Format` produces for the same document
- [ ] Comments are preserved in their original positions (inherited from the formatter)
- [ ] Unknown document URI returns an error with code `-32602`
- [ ] A syntactically invalid document returns an empty `TextEdit[]` (no formatting on parse errors)

**Affected Files/Modules:**
- `internal/lsp/protocol.go` — Add `TextEdit` struct (single edit replacing full-document range), `DocumentFormattingParams` struct, and `DocumentFormattingOptions` if needed
- `internal/lsp/server.go` — Add `DocumentFormattingProvider` to `ServerCapabilities` in `handleInitialize`; add `textDocument/formatting` case to `dispatch`; add `handleFormatting` method that retrieves the document, lexes/parses it, runs `formatter.Format`, and returns a full-document `TextEdit`
- `internal/lsp/server_test.go` — Add tests for `textDocument/formatting`: valid document returns formatted result, invalid document returns empty edits, unknown URI returns error, formatting preserves comments, formatting is idempotent via round-trip test

**Patterns to Follow:**
- Follow the handler pattern in `internal/lsp/server.go:192-222` (`handleDefinition`) for unmarshalling params, looking up the document, and returning an error for unknown URIs
- Follow the lex→parse pattern in `internal/lsp/server.go:227-231` (`pushDiagnostics`) to produce the AST needed for `formatter.Format`
- Follow the capability-addition pattern in `internal/lsp/server.go:88-110` (`handleInitialize`) — add `DocumentFormattingProvider` alongside the existing `DefinitionProvider` and `CompletionProvider`

**Testable:** Yes — Tests follow the existing `serverPair` pipe-based integration test pattern in `server_test.go`. Send a `textDocument/formatting` request, verify the response `TextEdit` content matches `formatter.Format` output.

**Verification:** `go test ./internal/lsp/...` passes; `go build ./...` succeeds.

**Depends on:** None

## Summary

- **Total tasks:** 1
- **Task ordering rationale:** Single task — the change is a self-contained addition of a new LSP method handler. No dependency ordering needed.
- **Acceptance criteria coverage:** All four acceptance criteria from the story are satisfied by this single task. The comment-preservation criterion is inherited from the existing `formatter.Format` function (already tested via `TestFormat` in `formatter_test.go`). The "editor applies on save" criterion is satisfied by advertising `DocumentFormattingProvider` — VS Code handles the rest automatically.

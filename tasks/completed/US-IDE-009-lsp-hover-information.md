## Progress
- [x] Task 1: Add LSP hover protocol types and server dispatch handler
- [x] Task 2: Implement hover content for named identifiers
- [x] Task 3: Implement hover content for keywords and finalize non-resolvable behavior

## Story Reference
**US-IDE-009: LSP hover information** — As a model author, I want to hover over an identifier and see contextual information so that I can understand where an element is defined and how it fits in the model.

## Codebase Context

The EMOD LSP server lives in `internal/lsp/` (Go). Existing handlers follow a clear pattern:
- `server.go:59-90` — `dispatch` routes JSON-RPC methods to `handleXxx` helpers.
- `server.go:92-115` — `handleInitialize` advertises capabilities in `ServerCapabilities`.
- `definition.go:14-132` — `GetDefinition` parses the document, locates the token under the cursor using `cursorOnName`, and returns a `*Location`.
- `protocol.go` — LSP protocol structs (e.g., `Position`, `Range`, `Location`, `CompletionItem`). No hover types exist yet.
- `definition_test.go` — Tests for `GetDefinition` use the `lsp_test` package, `posIn` helper, and `require` assertions.
- `server_test.go` — Integration tests spin up a real `Server` over `io.Pipe`, send JSON-RPC messages, and assert on responses.
- `ast/ast.go` — AST nodes include `Command`, `Event`, `View`, `Context`, `Aggregate`, `Slice`, etc., with `Name`, `NamePos`, `Fields`, `Subscribes`.
- The VS Code extension (`editors/vscode/`) uses `vscode-languageclient`, which automatically enables hover when the server advertises `hoverProvider: true`. No TypeScript changes are required.

## Tasks

### Task 1: Add LSP hover protocol types and server dispatch handler

**Behavior:** The LSP server advertises hover support and responds to `textDocument/hover` requests with a null result (no hover content yet).

**Acceptance Criteria:**
- [ ] The LSP server responds to `textDocument/hover` requests
- [ ] Hovering over a non-resolvable token returns no hover content (no error)

**Affected Files/Modules:**
- `internal/lsp/protocol.go` — add `Hover`, `HoverParams`, `MarkupContent`, `MarkupKind` structs
- `internal/lsp/server.go` — add `hoverProvider` capability, `handleHover` dispatch, and `handleHover` method
- `internal/lsp/server_test.go` — add integration tests for `textDocument/hover` (null result, capability advertised, error for unknown document)

**Patterns to Follow:**
- Follow the pattern in `server.go:95-102` for adding a new capability to `ServerCapabilities`
- Follow the pattern in `server.go:59-90` for dispatching a new method in `dispatch`
- Follow the pattern in `server.go:197-227` for a `handleXxx` that reads params, fetches document, calls a helper, and marshals the result
- Follow the pattern in `server_test.go:622-766` for integration tests that open a document, send a request, and assert the response shape

**Testable:** Yes

**Verification:** `go test ./internal/lsp/... -tags=unit` passes

**Depends on:** None

---

### Task 2: Implement hover content for named identifiers

**Behavior:** Given a cursor position on a command, event, or view name, `GetHover` returns a `*Hover` with `MarkupContent` describing the element's parent context and aggregate, plus relevant details (event fields, view subscriptions).

**Acceptance Criteria:**
- [ ] Hovering over a command name shows its parent context and aggregate (e.g., "Command in Reservations > Reservation")
- [ ] Hovering over an event name shows its parent context and aggregate, plus its field list
- [ ] Hovering over a view name shows its subscribed events

**Affected Files/Modules:**
- `internal/lsp/hover.go` — new file containing `GetHover` function with AST traversal logic
- `internal/lsp/server.go` — update `handleHover` to call `GetHover` instead of returning nil
- `internal/lsp/hover_test.go` — new file with unit tests for `GetHover` on commands, events, and views

**Patterns to Follow:**
- Follow the pattern in `definition.go:14-132` for `GetDefinition` — parse document, convert cursor coordinates, iterate AST to find matching token, and build a result
- Follow the pattern in `definition.go:136-143` for `cursorOnName` to detect when the cursor is on a token
- Follow the pattern in `definition_test.go:13-250` for test structure (shared test document, `posIn` helper, `assertXxx` helpers, `t.Run` groups)

**Testable:** Yes

**Verification:** `go test ./internal/lsp/... -tags=unit` passes

**Depends on:** Task 1

---

### Task 3: Implement hover content for keywords and finalize non-resolvable behavior

**Behavior:** Given a cursor position on an EMOD keyword, `GetHover` returns a `*Hover` with a brief description of the block. For any unrecognized or non-resolvable token, `GetHover` returns `nil` (no hover content, no error).

**Acceptance Criteria:**
- [ ] Hovering over a keyword (e.g., `automation`) shows a brief description of what the block does
- [ ] Hovering over a non-resolvable token returns no hover content (no error)

**Affected Files/Modules:**
- `internal/lsp/hover.go` — extend `GetHover` with keyword recognition and a description map
- `internal/lsp/hover_test.go` — add unit tests for keyword hover and non-resolvable token hover
- `internal/lsp/server_test.go` — add integration test for keyword hover through the server

**Patterns to Follow:**
- Follow the pattern in `definition.go:136-143` for `cursorOnName` to detect when the cursor is on a keyword token
- Follow the pattern in `definition_test.go:178-188` for tests that assert `nil` on keywords/non-resolvable tokens
- Follow the pattern in `server_test.go:693-737` for integration tests that assert a null result when no hover content is found

**Testable:** Yes

**Verification:** `go test ./internal/lsp/... -tags=unit` passes

**Depends on:** Task 2

## Summary

- **Total tasks:** 3
- **Estimated ordering:** Infrastructure first (protocol + dispatch), then domain identifiers (commands/events/views), then keywords and edge cases. This ordering ensures each task builds on a working server and keeps the codebase green.
- **All acceptance criteria are covered:**
  - AC1 (server responds) — Task 1
  - AC2 (command hover) — Task 2
  - AC3 (event hover) — Task 2
  - AC4 (view hover) — Task 2
  - AC5 (keyword hover) — Task 3
  - AC6 (non-resolvable token) — Task 1 (null response) and Task 3 (ensures no error)
- **No deferred acceptance criteria.**
- **Languages:** All tasks are Go. The VS Code extension (`editors/vscode/`) does not require changes because `vscode-languageclient` automatically enables hover when the server advertises `hoverProvider: true`.

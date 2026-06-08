## Progress
- [x] Task 1: Add position tracking for subscribes list entries in the parser
- [x] Task 2: Implement go-to-definition resolution logic
- [ ] Task 3: Wire go-to-definition handler into the LSP server

---

## Story Reference

Derived from **US-IDE-007** in `user-stories/ide-support.md:113-125`.

---

## Codebase Context

**Affected modules:**
- `internal/ast/` — AST type definitions (commands, events, views, triggers, automations, translations, flows)
- `internal/parser/` — Recursive-descent parser that builds the AST from tokens
- `internal/lsp/` — LSP server with JSON-RPC transport, document manager, completion handler, diagnostics

**Existing patterns:**
- All named AST elements have `Name string` + `NamePos ast.Position` fields
- Reference fields (names pointing to other definitions) have companion `*Pos` fields (e.g., `Flow.CommandPos`, `Automation.TriggerEventPos`, `Translation.CommandPos`, `Trigger.ReadsPos`)
- **Notable gap:** `View.Subscribes []string` has NO position tracking for individual entries — the parser's `parseSubscribes()` only records the identifier token's value (`p.advance().Value`), discarding the token's line/column
- The validator already builds `commandPositions` and `eventPositions` name→position maps during validation — this pattern can be reused in a dedicated resolver
- Server handler pattern uses a switch dispatch in `server.go:58-84` matching on `msg.Method`, with handler methods like `handleCompletion` that call exported pure functions (e.g., `GetCompletions`) and write JSON-RPC responses
- Server testing uses a `serverPair` helper (`server_test.go:16-54`) with `io.Pipe` I/O, sending JSON-RPC messages and reading responses
- Pure function testing (like `completer_test.go`) directly calls exported functions with text input and asserts on the result

**Caller analysis (per `caller-patterns.md`):**
- The LSP go-to-definition is an **Inbound** pattern — the editor sends a `textDocument/definition` request with a position, the server responds with a `Location` or `null`. The caller (editor) observes the navigation result. Tests should assert on the response content (Location fields) and error absence.

---

## Tasks

### Task 1: Add position tracking for subscribes list entries in the parser

**Language:** Go

**Behavior:** Each entry in a view's `subscribes [...]` list records its source position, enabling the LSP to determine which subscribe entry the cursor is on.

**Acceptance Criteria:**
- [ ] `ast.View` has a `SubscribesPos []ast.Position` field alongside `Subscribes []string`
- [ ] `parseSubscribes()` records the position of each identifier token
- [ ] Existing parser behavior is unchanged (names still parsed correctly)
- [ ] Existing tests pass without modification

**Affected Files/Modules:**
- `internal/ast/ast.go` — Add `SubscribesPos []Position` to `View` struct
- `internal/parser/parser.go` — Record position of each identifier in `parseSubscribes()`

**Patterns to Follow:**
- Position field naming convention in `internal/ast/ast.go:62-69` (e.g., `NamePos`, `CommandPos`, `EventPos`)
- Position recording pattern in `internal/parser/parser.go:834-836` (e.g., `CommandPos: p.position(cmdTok)`)
- The existing `parseSubscribes` function at `internal/parser/parser.go:667-696`

**Testable:** Yes — tests verify that each entry in the subscribes list has a position equal to the identifier's source position. Tests use the exported parser API with `//go:build unit` tag.

**Verification:** `go test ./internal/parser/...` passes. `go build ./...` succeeds.

**Depends on:** None

---

### Task 2: Implement go-to-definition resolution logic

**Language:** Go

**Behavior:** Given document text and a cursor position, determines whether the cursor is on a reference to a named element (event, command, view, or context) and resolves it to the definition's location. Handles all reference types: subscribes entries, automation trigger/command/target-context, translation reads/command, trigger reads, and flow command/event. Returns nil when no definition is found or the cursor is not on a known reference.

**Acceptance Criteria:**
- [ ] An exported `GetDefinition(text string, line, character int) *Location` function exists in the `lsp` package
- [ ] Cursor on an event name in `subscribes [...]` resolves to the `event` block's definition position
- [ ] Cursor on a command name in an `automation` block resolves to the `command` block's definition position
- [ ] Cursor on a command name in a `translation` block resolves to the `command` block's definition position
- [ ] Cursor on a view name in a `trigger`'s `reads` resolves to the `view` block's definition position
- [ ] Cursor on a view name in a `translation`'s `reads` resolves to the `view` block's definition position
- [ ] Cursor on a context name in `automation`'s `target context` resolves to the `context` block's definition position
- [ ] Cursor on an event name in a `flow`'s event resolves to the `event` block's definition position
- [ ] Cursor on a command name in a `flow`'s command resolves to the `command` block's definition position
- [ ] If the cursor is not on a known reference, or the referenced name has no definition, returns nil (no navigation)

**Affected Files/Modules:**
- `internal/lsp/protocol.go` — Add `Location` type with `URI` and `Range` fields
- `internal/lsp/definition.go` — New file containing `GetDefinition` and helper functions

**Patterns to Follow:**
- Pure function pattern in `internal/lsp/completer.go:10-17` (`GetCompletions`) — exported function takes primitive inputs, returns result
- Direct test pattern in `internal/lsp/completer_test.go` — call the exported function with crafted document text and cursor positions, assert on the result
- LSP position conversion in `internal/lsp/diagnostics.go:19-26` — AST 1-based to LSP 0-based (subtract 1 from line and column)
- Name→position map building pattern in `internal/validator/validator.go:18-32` (the `commandPositions`/`eventPositions` loop across contexts → aggregates → slices)

**Testable:** Yes — tests construct emod documents with various reference types, call `GetDefinition` at the cursor position, and verify the returned `Location` points to the correct definition position. Tests also verify that positions not on references return nil.

**Verification:** `go test ./internal/lsp/...` passes. `go build ./...` succeeds.

**Depends on:** Task 1 (for subscribes position tracking)

---

### Task 3: Wire go-to-definition handler into the LSP server

**Language:** Go

**Behavior:** The server advertises `DefinitionProvider` capability during initialization, dispatches incoming `textDocument/definition` requests to a handler, and returns the resolved location (or null if unresolved). The handler re-uses the existing document content from the `DocumentManager` and calls `GetDefinition`.

**Acceptance Criteria:**
- [ ] `ServerCapabilities` includes `DefinitionProvider: true` in the initialize response
- [ ] Server dispatches `"textDocument/definition"` to a `handleDefinition` method
- [ ] Handler reads the document from `DocumentManager`, calls `GetDefinition`, and returns the `Location` or null
- [ ] Unknown document URI returns an error response (matching existing pattern for completion)
- [ ] All existing tests continue to pass

**Affected Files/Modules:**
- `internal/lsp/protocol.go` — Add `DefinitionProvider bool` to `ServerCapabilities`
- `internal/lsp/server.go` — Add dispatch case for `"textDocument/definition"` and `handleDefinition` method
- `internal/lsp/server_test.go` — Add integration tests for go-to-definition via the server

**Patterns to Follow:**
- Dispatch case pattern at `internal/lsp/server.go:67-68` (`case "textDocument/completion": return s.handleCompletion(msg)`)
- Handler pattern at `internal/lsp/server.go:157-187` (`handleCompletion`) — unmarshal params, lookup document, call logic function, marshal result, write response
- Server test pattern at `internal/lsp/server_test.go:451-503` ("completion" test block) — initialize server, open document, send definition request, read response, assert on Location content
- Document-not-found error pattern at `internal/lsp/server.go:165-173` — return `-32602` error with descriptive message

**Testable:** Yes — server integration tests verify the full round-trip: initialize → didOpen → definition request → response with correct location. Tests also verify null response for unresolvable positions and error for unknown document URIs.

**Verification:** `go test ./internal/lsp/...` passes. `go build ./...` succeeds.

**Depends on:** Task 2

---

## Summary

- **Total tasks:** 3
- **Ordering rationale:** Dependency-first. Task 1 fixes the parser gap (subscribes position tracking) that the resolution logic needs. Task 2 implements the core go-to-definition business logic as a testable pure function. Task 3 wires it into the server with minimal dispatch/handler plumbing.
- **Coverage of acceptance criteria:**
  - "Server responds to `textDocument/definition` requests" → Task 3
  - "Clicking an event name in subscribes jumps to event block" → Task 1 + Task 2
  - "Clicking a command name in automation/translation jumps to command block" → Task 2
  - "Clicking a view name in trigger's reads or translation's reads jumps to view block" → Task 2
  - "Clicking a context name in target context jumps to context block" → Task 2
  - "If definition not found, no navigation occurs" → Task 2 (returns nil) + Task 3 (returns null result)
- **All acceptance criteria are covered** across the 3 tasks. Nothing is deferred.

# US-IDE-010: LSP find references

## Progress
- [x] Task 1: Add protocol types and implement GetReferences function with unit tests
- [x] Task 2: Wire References into the LSP server with integration tests

---

## Story Reference

Derived from **US-IDE-010** in `user-stories/ide-support.md` — "LSP find references."

---

## Codebase Context

**Affected modules:**
- `internal/lsp/protocol.go` — LSP protocol types (Position, Range, Location, ServerCapabilities, DefinitionParams, etc.)
- `internal/lsp/definition.go` — Existing go-to-definition logic that builds name→position maps and walks the AST checking references; the pattern to mirror for references
- `internal/lsp/definition_test.go` — Umbrella test pattern with `posIn` helper for locating text substrings in a document and asserting on locations
- `internal/lsp/server.go` — JSON-RPC dispatch loop, handler methods like `handleDefinition`, initialize capability advertisement
- `internal/lsp/server_test.go` — Integration tests with `serverPair` helper using piped I/O
- `internal/ast/ast.go` — All relevant AST types with name + position fields: `View.Subscribes`/`SubscribesPos`, `Automation.TriggerEvent`/`TriggerEventPos`/`Command`/`CommandPos`/`TargetContext`/`TargetContextPos`, `Translation.Reads`/`ReadsPos`/`Command`/`CommandPos`, `Trigger.Reads`/`ReadsPos`, `Flow.CommandName`/`CommandPos`/`EventName`/`EventPos`, and definition positions: `Command.NamePos`, `Event.NamePos`, `View.NamePos`

**Existing patterns:**
- `GetDefinition` at `definition.go:14-132` builds name→position maps (commandDefs, eventDefs, viewDefs, contextDefs) by walking the AST, then checks if the cursor sits on a reference. For references, the same maps are built, but the logic collects ALL reference locations for the resolved definition name instead of returning a single definition location.
- `cursorOnName` at `definition.go:136-143` checks whether a cursor (1-based AST coordinates) falls within the text span of a named token.
- `locationFor` at `definition.go:148-165` builds an LSP `Location` from an AST position, converting 1-based to 0-based coordinates.
- `nameRange` at `hover.go:159-172` is an equivalent helper that returns `*Range` instead of `*Location` — both follow the same position conversion pattern.
- Server handler pattern at `server.go:200-229` (`handleDefinition`): unmarshal params, lookup document, call exported function, marshal result, write response. Unknown document URI returns error with code `-32602` and message "document not found:".
- Definition work already established that `ServerCapabilities` has a `DefinitionProvider bool` field set in `handleInitialize`. The same pattern applies for `ReferencesProvider`.
- Definition tests in `definition_test.go` use a `posIn` helper to locate cursor positions within a shared document, and a `locationFor`-equivalent assertion pattern on the returned `Location`. References tests will need to assert on a `[]Location` slice instead of a single `*Location`.
- Server integration tests in `server_test.go` use `startServer` → write init/didOpen → write request → read response → unmarshal result pattern.
- All position tracking needed for references (including `SubscribesPos`) is already present in the AST and parser.

---

## Tasks

### Task 1: Add protocol types and implement GetReferences function with unit tests

**Language:** Go

**Behavior:** An exported `GetReferences` function finds all references to a command, event, or view name from any occurrence of that name (definition or reference) in the document. Given document text and a cursor position, it determines what name the cursor is on, resolves it to the canonical definition name, and collects every location where that name appears — its definition and all references across subscribes lists, automation blocks, translation blocks, trigger reads, and flow entries. Returns a slice of all reference locations, or nil if the cursor is not on a resolvable name.

**Acceptance Criteria:**
- [ ] `ReferenceParams` type is defined in `protocol.go` (same fields as `DefinitionParams`: `TextDocument` and `Position`)
- [ ] `ReferencesProvider` field (bool) is added to `ServerCapabilities`
- [ ] `GetReferences(text string, line int, character int, uri string) []Location` is exported in the `lsp` package
- [ ] Cursor on an event name (definition or reference) returns all locations: the event definition, all subscribes list entries referencing that event, all automation trigger references, and all flow event entries
- [ ] Cursor on a command name (definition or reference) returns all locations: the command definition, all automation command references, all translation command references, and all flow command entries
- [ ] Cursor on a view name (definition or reference) returns all locations: the view definition, all trigger reads references, and all translation reads references
- [ ] Each returned location includes correct `URI`, `Range.Start` (line+character), and `Range.End` (line+character) covering the full name text
- [ ] Cursor not on a reference or definition name returns nil
- [ ] Multiple occurrences of the same name (e.g., same event referenced in multiple subscribes lists) are all included
- [ ] Empty document returns nil

**Affected Files/Modules:**
- `internal/lsp/protocol.go` — Add `ReferenceParams` type and `ReferencesProvider` field to `ServerCapabilities`
- `internal/lsp/references.go` — New file containing `GetReferences` and helper functions
- `internal/lsp/references_test.go` — New file with umbrella test covering all reference types and edge cases

**Patterns to Follow:**
- Pure function export pattern at `internal/lsp/definition.go:14` (`func GetDefinition(...)`) — exported function takes text, line, character, uri; returns `[]Location`
- Name→position map building at `internal/lsp/definition.go:27-47` — walk contexts → aggregates → slices, build `commandDefs`, `eventDefs`, `viewDefs` maps
- `cursorOnName` at `internal/lsp/definition.go:136-143` — reuse the same helper for checking cursor-on-name
- `locationFor` at `internal/lsp/definition.go:148-165` — reuse for building individual Location values
- AST reference walking pattern at `internal/lsp/definition.go:55-129` — iterate through all references: subscribes, automations, translations, triggers, flows
- Unit test umbrella pattern at `internal/lsp/definition_test.go:13-250` — single `TestGetReferences` with `t.Run` groups per name type (event, command, view), nested subtests per reference type, and `posIn` helper for cursor placement
- Build tag convention: `//go:build unit` at test file top, `package lsp_test` (external test package)
- Test assertions use `testify/require`

**Testable:** Yes — tests construct emod documents with definitions and various reference types, call `GetReferences` at different cursor positions, and verify the returned `[]Location` slice contains all expected reference locations with correct URI and Range. Tests also cover cursor-not-on-name, empty document, and names with no references.

**Verification:** `go test ./internal/lsp/ -tags unit` passes. `go build ./...` succeeds.

**Depends on:** None

---

### Task 2: Wire References into the LSP server with integration tests

**Language:** Go

**Behavior:** The server advertises `ReferencesProvider` capability during initialization, dispatches incoming `textDocument/references` requests to a handler, and returns the list of reference locations (or null if unresolved). The handler reads document content from the `DocumentManager` and calls `GetReferences`.

**Acceptance Criteria:**
- [ ] `ServerCapabilities.ReferencesProvider` is set to `true` in the `handleInitialize` response
- [ ] Server dispatches `"textDocument/references"` to a `handleReferences` method
- [ ] Handler unmarshals `ReferenceParams`, looks up the document from `DocumentManager`, calls `GetReferences`, and marshals the result as `[]Location` (or null)
- [ ] Unknown document URI returns an error response with code `-32602` and descriptive message (matching existing pattern)
- [ ] All existing tests continue to pass

**Affected Files/Modules:**
- `internal/lsp/server.go` — Add dispatch case for `"textDocument/references"` and `handleReferences` method; set `ReferencesProvider: true` in `handleInitialize`
- `internal/lsp/server_test.go` — Add integration test block for references with subtests covering: successful reference find, null result when cursor not on a name, error for unknown document URI

**Patterns to Follow:**
- Dispatch case pattern at `internal/lsp/server.go:71-72` (`case "textDocument/definition": return s.handleDefinition(msg)`)
- Handler pattern at `internal/lsp/server.go:200-229` (`handleDefinition`) — unmarshal `ReferenceParams`, lookup document, call `GetReferences`, marshal result, write response
- Document-not-found error pattern at `internal/lsp/server.go:209-217` — return `-32602` error with `"document not found: " + uri`
- Capability advertisement pattern at `internal/lsp/server.go:102` (`DefinitionProvider: true`) — set `ReferencesProvider: true` alongside other capabilities
- Server integration test pattern at `internal/lsp/server_test.go:637-781` ("definition" test block) — `startServer`, initialize, didOpen document, send references request, read response, unmarshal and assert on locations
- Null response assertion pattern at `internal/lsp/server_test.go:750` — `require.Equal(t, "null", string(resp.Result))`

**Testable:** Yes — server integration tests verify the full round-trip: initialize (capability advertisement) → didOpen → references request → response with correct `[]Location` slice. Tests also verify null response for unresolvable positions and error for unknown URIs.

**Verification:** `go test ./internal/lsp/ -tags unit` passes. `go build ./...` succeeds.

**Depends on:** Task 1 (for `GetReferences` and `ReferenceParams`)

---

## Summary

- **Total tasks:** 2
- **Ordering rationale:** Dependency-first. Task 1 delivers the core references resolution logic (exported function + unit tests). Task 2 wires it into the server dispatch and advertises the capability (integration tests). Task 1 can be built and tested independently; Task 2 depends on Task 1's types and function.
- **Coverage of acceptance criteria:**
  - "The LSP server responds to `textDocument/references` requests" → Task 2 (dispatch + handler)
  - "Finding references on an event name returns all locations: its definition, subscribes lists, automation trigger references, and flow entries" → Task 1 (core logic) + Task 2 (wire protocol)
  - "Finding references on a command name returns all locations: its definition, automation command references, translation command references, and flow entries" → Task 1
  - "Finding references on a view name returns all locations: its definition and reads references in triggers and translations" → Task 1
  - "Results include file path, line, and column for each reference" → Task 1 (`Location` includes `URI` + `Range` with `Start`/`End` `Position`)
- **All acceptance criteria are covered** across the 2 tasks. Nothing is deferred.

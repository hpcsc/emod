# US-IDE-011: LSP Semantic Tokens

## Progress
- [x] Task 1: Add LSP protocol types for semantic tokens
- [x] Task 2: Implement semantic token computation from AST
- [x] Task 3: Wire semantic tokens handler into LSP server

## Story Reference
**File:** `user-stories/us-ide-011-semantic-tokens.md` (derived from inline description)

**Depends on:** US-IDE-004

## Codebase Context

The project has a working LSP server at `internal/lsp/` with the following structure:

- **`server.go`** — Main `Server` struct with message loop, `dispatch()` switch, and handler methods (`handleCompletion`, `handleDefinition`, `handleReferences`, `handleFormatting`, `handleHover`, `handleDidOpen`, `handleDidChange`, `handleShutdown`). Handlers follow a consistent pattern: unmarshal params, get doc from `DocumentManager`, call package-level function, marshal result, write response.
- **`protocol.go`** — LSP protocol types (`Position`, `Range`, `Diagnostic`, `CompletionItem`, `ServerCapabilities`, `InitializeResult`, etc.). All types are exported with JSON tags.
- **`diagnostics.go`** — `ConvertDiagnostics()` function.
- **`document.go`** — `DocumentManager` (thread-safe in-memory doc store).
- **`completer.go`** — `GetCompletions(text string, line, character int) CompletionList`.
- **`hover.go`** — `GetHover(text string, line, character int) *Hover`.
- **`definition.go`** — `GetDefinition(text string, line, character int, uri string) Location`.
- **`references.go`** — `GetReferences(text string, line, character int, uri string) []Location`.

The AST types are in `internal/ast/ast.go` with types: `Model`, `Actor`, `Context`, `Aggregate`, `Slice`, `Command`, `Event`, `View`, `Field`, `Flow`, `Trigger`, `Automation`, `Translation`. Each named AST node has a `Name` (string) and `NamePos` (`ast.Position` with `Filename`, `Line`, `Column` — 1-based).

The lexer (`internal/lexer/`) produces `Token` structs with `Type`, `Value`, `Line`, `Column` (0-based for Line, 1-based for Column).

The parser (`internal/parser/`) pattern: `tokens, diags := lexer.Scan(doc, uri)` then `p := parser.New(tokens, uri)` then `model, parseErrs := p.Parse()`.

Testing patterns:
- Tests use `//go:build unit` tags, `package lsp_test` (external test package), `github.com/stretchr/testify/require`.
- Unit tests for logic functions use inline document strings (see `hover_test.go`).
- Integration tests use `io.Pipe`-based server plumbing (see `server_test.go`, `startServer` helper).
- Test command: `go test -tags unit $(go list ./... | grep -v /cmd/emod-wasm)`

No existing semantic token types, handlers, or any semantic token references exist in the codebase.

The VS Code extension at `editors/vscode/` has a TextMate grammar (`syntaxes/emod.tmLanguage.json`) and uses `vscode-languageclient` to connect to the LSP server. No extension changes are required for this feature — VS Code's language client automatically handles `textDocument/semanticTokens/full` requests when the server advertises the capability, and semantic tokens automatically override TextMate scopes.

## Tasks

### Task 1: Add LSP protocol types for semantic tokens

**Behavior:** The `protocol.go` file gains the type definitions and constants needed for the LSP semantic tokens protocol, and `ServerCapabilities` gains a field to advertise the capability.

**Acceptance Criteria:**
- [ ] `SemanticTokenTypes` constant type is defined with standard LSP token types (or the subset needed for emod)
- [ ] `SemanticTokensLegend` struct is defined with `TokenTypes` and `TokenModifiers` string slices
- [ ] `SemanticTokensParams` struct is defined with `TextDocument` identifier
- [ ] `SemanticTokens` struct is defined with a `Data` uint slice field
- [ ] `SemanticTokensProviderOptions` or equivalent struct is defined for the legend
- [ ] `ServerCapabilities` gains a `SemanticTokensProvider` field for capability advertisement
- [ ] All new types have correct JSON tags for LSP wire format

**Affected Files/Modules:**
- `internal/lsp/protocol.go` — Add `SemanticTokenTypes`, `SemanticTokensLegend`, `SemanticTokensParams`, `SemanticTokens`, legend options struct, `SemanticTokensProvider` field on `ServerCapabilities`

**Patterns to Follow:**
- Follow the type definition style in `internal/lsp/protocol.go:1-208` — exported types with `json:"..."` struct tags, `const` blocks for enum values, consistent naming

**Testable:** No — pure type and constant definitions with no behavioral logic. The types are exercised when Task 2 and Task 3 use them in behavior tests.

**Verification:** `go build ./...` succeeds

**Depends on:** None

---

### Task 2: Implement semantic token computation from AST

**Behavior:** A new `GetSemanticTokens` function lexes and parses an emod document, walks the AST to identify named identifiers, and returns an LSP `SemanticTokens` struct with encoded token data (delta-encoded positions and token types). Command names, event names, view names, actor names, context names, and aggregate names each receive distinct token types.

**Acceptance Criteria:**
- [ ] `GetSemanticTokens(doc string) *SemanticTokens` is exported from the `lsp` package
- [ ] Command names receive a distinct `SemanticTokenTypes` value (e.g., `function`)
- [ ] Event names receive a distinct `SemanticTokenTypes` value (e.g., `event`)
- [ ] View names receive a distinct `SemanticTokenTypes` value (e.g., `class`)
- [ ] Actor names receive a distinct `SemanticTokenTypes` value (e.g., `parameter`)
- [ ] Context names receive a distinct `SemanticTokenTypes` value (e.g., `namespace`)
- [ ] Aggregate names receive a distinct `SemanticTokenTypes` value (e.g., `struct`)
- [ ] Token data is delta-encoded per the LSP spec: `deltaLine, deltaChar, length, tokenType, tokenModifiers`
- [ ] Documents with parse errors return an empty `SemanticTokens` (zero tokens) rather than failing
- [ ] Documents with no named identifiers (e.g., empty model) return an empty `SemanticTokens`
- [ ] Tests cover each identifier type producing the correct token type
- [ ] Tests cover the delta encoding format with multiple identifiers on the same line and across lines
- [ ] Tests cover documents with parse errors returning zero tokens

**Affected Files/Modules:**
- `internal/lsp/semantictokens.go` — New file with `GetSemanticTokens` function
- `internal/lsp/semantictokens_test.go` — New test file with unit tests

**Patterns to Follow:**
- Follow the exported function pattern in `internal/lsp/hover.go:1-100` — package-level exported function taking document text, using lexer and parser internally, returning LSP types
- Follow the test structure in `internal/lsp/hover_test.go:1-150` — external test package, inline document constants, `posIn` helper for position lookup, `t.Run` nested subtests
- Use `lexer.Scan` then `parser.New(...).Parse()` as done in `internal/lsp/server.go:319-321`
- Walk the AST model's `Actors`, `Contexts` (and their `Aggregates`, and their `Slices` with `Commands`, `Events`, `Views`) using the `NamePos` fields from `internal/ast/ast.go:1-153`
- Follow caller-pattern: **Exported API** (`~/.config/ai/guidelines/testing/caller-patterns.md:378-460`) — tests assert on return values for given inputs; use strict assertions for token type values (changing a token type changes user-visible coloring)
- Follow assertion strictness guidance from `~/.config/ai/guidelines/go/testing-patterns.md:1065-1116` — use strict assertions (`require.Equal`) for token types/legends since they are API contracts

**Testable:** Yes — tests call `lsp.GetSemanticTokens(doc)` and assert on the returned `SemanticTokens.Data` array contents

**Verification:** `go test -tags unit ./internal/lsp/...` passes

**Depends on:** Task 1

---

### Task 3: Wire semantic tokens handler into LSP server

**Behavior:** The LSP server advertises semantic tokens capability during initialization, dispatches `textDocument/semanticTokens/full` requests, calls `GetSemanticTokens` on the document content, and returns the encoded result to the client.

**Acceptance Criteria:**
- [ ] `handleInitialize` advertises `SemanticTokensProvider` with the legend in `ServerCapabilities`
- [ ] `dispatch` routes `"textDocument/semanticTokens/full"` to `handleSemanticTokensFull`
- [ ] `handleSemanticTokensFull` unmarshals params, retrieves document from `DocumentManager`, calls `GetSemanticTokens`, marshals result, and writes response
- [ ] Happy path: known document URI returns `SemanticTokens` with data in the response
- [ ] Error path: unknown document URI returns a method-not-found or invalid-params error (consistent with other handlers)
- [ ] Error path: unparseable document returns empty `SemanticTokens` data
- [ ] Integration tests verify the full request/response cycle through the server's pipe-based I/O

**Affected Files/Modules:**
- `internal/lsp/server.go` — Add `handleSemanticTokensFull` method, add dispatch case, update `handleInitialize` capabilities
- `internal/lsp/server_test.go` — Add integration test subtests under `t.Run("semanticTokens/full", ...)`

**Patterns to Follow:**
- Follow the handler pattern in `internal/lsp/server.go:267-297` (`handleHover`) — unmarshal params, get doc from `s.documents.GetContent`, handle missing doc error, call package function, marshal result, write response
- Follow the dispatch pattern in `internal/lsp/server.go:60-93` — add a new `case "textDocument/semanticTokens/full":` that calls the new handler
- Follow the capability advertisement pattern in `internal/lsp/server.go:96-121` — add `SemanticTokensProvider` field to the `ServerCapabilities` literal
- Follow the integration test pattern in `internal/lsp/server_test.go:511-650` (`t.Run("completion", ...)`) — start server, initialize, open doc, send semantic tokens request, read and validate response
- Follow caller-pattern: **UI** (`~/.config/ai/guidelines/testing/caller-patterns.md:44-103`) — tests verify the JSON response content (what the editor sees); assert on the `Result` field content

**Testable:** Yes — integration tests via `startServer` + `writeMsg`/`readMsg` pipe pattern verify end-to-end request/response

**Verification:** `go test -tags unit ./internal/lsp/...` passes

**Depends on:** Task 2

---

## Summary

**Total tasks:** 3

**Rationale for ordering:** Dependency-first. Task 1 provides the protocol types that Task 2 needs for its return type. Task 2 provides the core semantic token computation logic that Task 3 wires into the server. This ordering allows each task to be independently committed and tested.

**Coverage of acceptance criteria:**

| Criterion | Covered In |
|---|---|
| The LSP server responds to `textDocument/semanticTokens/full` requests | Task 3 |
| Command names are assigned a distinct token type | Task 2 |
| Event names are assigned a distinct token type | Task 2 |
| View names are assigned a distinct token type | Task 2 |
| Actor names, context names, and aggregate names each have distinct token types | Task 2 |
| Semantic tokens override TextMate highlighting when both are available | Not a code change — automatic VS Code behavior when the LSP server advertises the `semanticTokensProvider` capability (achieved in Task 3) |

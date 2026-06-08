# US-IDE-004: LSP Server with Real-Time Diagnostics

## Progress
- [x] Task 1: Add LSP protocol types and JSON-RPC message framing
- [x] Task 2: Add document manager and diagnostic conversion
- [x] Task 3: Build LSP server with diagnostics on file open/change
- [x] Task 4: Register `emod lsp` CLI subcommand

---

## Story Reference

`user-stories/ide-support.md` — US-IDE-004: LSP server with real-time diagnostics

---

## Codebase Context

The CLI is structured around `internal/cli/app.go` which uses `urfave/cli/v2`. Each subcommand (validate, fmt, lint, export, diagram, slices, schema) has a `Run*` function in its own file (e.g., `internal/cli/validate.go`, `internal/cli/lint.go`). The validation pipeline lives in `internal/cli/validate.go` and follows: `lexer.Scan` → `parser.New(tokens, path).Parse()` → `validator.Validate(model)` → `linter.Lint(model)`. Diagnostics are typed as `*diagnostic.Entry` with `Filename`, `Line` (1-based), `Column` (1-based), `Message`, `Severity` (`Error` or `Warning`), and `RuleName`.

No existing LSP or JSON-RPC code exists in the codebase. No LSP client libraries are in `go.mod`. The `internal/lsp/` package will be new.

Tests use `//go:build unit` tags, `package cli_test` (external test package), `testify/require` for assertions, and the `writeTemp` helper for creating temp files. The `captureStdout` helper in `internal/cli/lint_test.go` redirects `os.Stdout` for testing output.

---

## Tasks

### Task 1: Add LSP protocol types and JSON-RPC message framing

**Language:** Go

**Behavior:** JSON-RPC messages (with `Content-Length` headers) can be read from and written to any `io.Reader`/`io.Writer`. LSP protocol types for the initial handshake (`InitializeParams`, `InitializeResult`, `ServerCapabilities`) and diagnostic publishing (`Position`, `Range`, `Diagnostic`, `PublishDiagnosticsParams`) are defined.

**Acceptance Criteria:**
- [ ] JSON-RPC request/response/notification envelope structs are defined with correct JSON field tags (`jsonrpc`, `id`, `method`, `params`, `result`, `error`)
- [ ] `ReadMessage(r io.Reader) (*Message, error)` reads `Content-Length: N\r\n\r\n` header followed by N bytes of JSON body
- [ ] `WriteMessage(w io.Writer, msg *Message) error` writes the `Content-Length` header and JSON body
- [ ] LSP `Position` type has `Line` and `Character` fields (zero-based per LSP spec)
- [ ] LSP `Range` has `Start` and `End` `Position` fields
- [ ] LSP `Diagnostic` has `Range`, `Severity` (1=Error, 2=Warning), `Message`, `Source` fields
- [ ] `InitializeParams` and `InitializeResult` (with `ServerCapabilities` with `TextDocumentSyncKind` set to `Incremental` or `Full`) types are defined
- [ ] `PublishDiagnosticsParams` has `URI` and `Diagnostics` array

**Affected Files/Modules:**
- `internal/lsp/transport.go` — `ReadMessage`, `WriteMessage`, `Message` envelope types, JSON-RPC error type
- `internal/lsp/protocol.go` — `Position`, `Range`, `Diagnostic`, `Severity`, `InitializeParams`, `InitializeResult`, `ServerCapabilities`, `PublishDiagnosticsParams`, `TextDocumentItem`, `VersionedTextDocumentIdentifier`
- `internal/lsp/transport_test.go` — transport unit tests
- `internal/lsp/protocol_test.go` — protocol type unit tests

**Patterns to Follow:**
- All types use exported struct fields with `json:"..."` tags, following the convention in `internal/diagnostic/entry.go`
- Test structure follows the pattern in `internal/diagnostic/entry_test.go`: one umbrella `Test*` function with `t.Run` subtests

**Testable:** Yes — exported `ReadMessage`/`WriteMessage` functions and protocol types are testable through exported API.

**Verification:** `go test ./internal/lsp/...` passes, JSON-RPC messages can be round-tripped through a `bytes.Buffer`

**Depends on:** None

---

### Task 2: Add document manager and diagnostic conversion

**Language:** Go

**Behavior:** Open documents are tracked with their current content in memory (not on disk). Internal `diagnostic.Entry` values are converted to LSP `Diagnostic` types with correct position mapping (1-based internal → 0-based LSP) and severity mapping.

**Acceptance Criteria:**
- [ ] `DocumentManager` has `Open(uri, content string)`, `Update(uri, content string)`, `Close(uri)` methods
- [ ] `GetContent(uri string) (string, bool)` returns the current in-memory content
- [ ] `Open` replaces any existing content for the URI
- [ ] `Close` removes the document from the store
- [ ] `ConvertDiagnostics(uri string, entries []*diagnostic.Entry) []Diagnostic` converts each entry to an LSP `Diagnostic` with:
  - `Range` with `Line` = `entry.Line - 1` (LSP is 0-based), `Character` = `entry.Column - 1`
  - `Range` end position is a single-character range (start line/col, end line/col+1) since internal diagnostics point to a start location
  - `Severity` mapping: `diagnostic.Error` → 1, `diagnostic.Warning` → 2
  - `Message` from `entry.Message`
  - `Source` set to `"emod"`
- [ ] Files that have never been `Open`-ed return empty content from `GetContent`

**Affected Files/Modules:**
- `internal/lsp/document.go` — `DocumentManager` struct, `Open`/`Update`/`Close`/`GetContent` methods
- `internal/lsp/diagnostics.go` — `ConvertDiagnostics` function
- `internal/lsp/document_test.go` — document manager tests
- `internal/lsp/diagnostics_test.go` — diagnostic conversion tests

**Patterns to Follow:**
- Follow the simple data-holder pattern used in `internal/diagnostic/entry.go` — exported struct with fields, no internal framework
- Test pattern follows `internal/diagnostic/entry_test.go` — umbrella `Test*` with operation groups via `t.Run`

**Testable:** Yes — `DocumentManager` methods and `ConvertDiagnostics` are exported and testable.

**Verification:** `go test ./internal/lsp/...` passes, document content is retrievable after open/update, conversion produces correct LSP positions

**Depends on:** Task 1

---

### Task 3: Build LSP server with diagnostics on file open/change

**Language:** Go

**Behavior:** The LSP server runs an event loop reading JSON-RPC messages from stdin, dispatching to handlers, and writing responses/notifications to stdout. It responds to `initialize`, `initialized`, `textDocument/didOpen`, `textDocument/didChange`, and `shutdown` requests. On `didOpen` and `didChange`, it re-parses the document and pushes diagnostics via `textDocument/publishDiagnostics`.

**Acceptance Criteria:**
- [ ] `Server` struct is created with `NewServer(in io.Reader, out io.Writer) *Server` for dependency-injectable I/O
- [ ] `server.Run(ctx)` starts the message read-dispatch-write loop, blocking until shutdown
- [ ] Receiving `initialize` request sends back `InitializeResult` with `textDocumentSync` capability set to `Full` (document content sent on every change)
- [ ] Receiving `initialized` notification is accepted silently (no response sent)
- [ ] Receiving `textDocument/didOpen` with `TextDocumentItem` stores the content in `DocumentManager` and pushes diagnostics
- [ ] Receiving `textDocument/didChange` with content changes updates the stored document content and re-pushes diagnostics
- [ ] Diagnostics are sent as a `textDocument/publishDiagnostics` notification with the document URI and array of LSP `Diagnostic` objects
- [ ] Parser errors produce error-severity diagnostics (severity=1)
- [ ] Validator warnings produce warning-severity diagnostics (severity=2)
- [ ] Linter diagnostics also appear (following the existing validate pipeline)
- [ ] The diagnostics pipeline uses the same `lexer.Scan` → `parser.Parse` → `validator.Validate` → `linter.Lint` chain as `RunValidate`
- [ ] Receiving `shutdown` request sends an empty response and causes the run loop to exit
- [ ] The server reads from the injected `io.Reader` (not hardcoded to stdin) and writes to the injected `io.Writer`

**Affected Files/Modules:**
- `internal/lsp/server.go` — `Server` struct, `NewServer`, `Run`, internal dispatch/handler methods
- `internal/lsp/server_test.go` — server tests using piped I/O (`io.Pipe` or `bytes.Buffer` pairs)

**Patterns to Follow:**
- The server loop pattern is similar to how `emod-wasm/main.go` sets up a read-loop — but the LSP server will be its own package
- Diagnostics pipeline follows the exact sequence in `internal/cli/validate.go:40-50` (lex → parse → validate → lint)
- Tests follow the `internal/cli/validate_test.go` pattern: one umbrella `TestServer` with `t.Run` groups for each operation (initialize, didOpen, didChange, shutdown, diagnostics)

**Testable:** Yes — server can be tested by piping JSON-RPC messages through `io.Pipe` and asserting on the output.

**Verification:** `go test ./internal/lsp/...` passes, server responds correctly to LSP handshake and pushes diagnostics on file open/change

**Depends on:** Task 2

---

### Task 4: Register `emod lsp` CLI subcommand

**Language:** Go

**Behavior:** Running `emod lsp` starts the LSP server, reading from stdin and writing to stdout.

**Acceptance Criteria:**
- [ ] `emod lsp` is registered as a new subcommand in the urfave/cli app
- [ ] The command creates an LSP `Server` with `os.Stdin` and `os.Stdout` and calls `Run`
- [ ] The command accepts no arguments or flags (LSP transport is always stdin/stdout)
- [ ] The `RunLSP` function is exported at the `cli` package level, consistent with `RunValidate`, `RunLint`, etc.

**Affected Files/Modules:**
- `internal/cli/app.go` — add `lsp` command entry in `Commands` slice (follow pattern of existing commands like `validate`, `lint`)
- `internal/cli/lsp.go` — `RunLSP` function that creates and runs the server

**Patterns to Follow:**
- Follow the existing CLI command pattern in `internal/cli/app.go:16-43` (validate command): define `Name`, `Usage`, `ArgsUsage`, `Action` closure that calls a `Run*` function
- The `lsp.go` file structure follows `internal/cli/schema.go`: thin `Run*` function with minimal logic

**Testable:** No — the `RunLSP` function blocks on I/O; its meaningful behavior is verified through Task 3's server tests.

**Verification:** `go build ./...` succeeds, `emod lsp --help` shows the command, `internal/cli/app.go` compiles

**Depends on:** Task 3

---

## Summary

- **Total tasks:** 4
- **Ordering rationale:** Dependency-first. Task 1 (protocol/transport) is foundational infrastructure. Task 2 (document manager + diagnostic conversion) builds on the types from Task 1. Task 3 (server with diagnostics) wires everything together and is the main behavior delivery. Task 4 (CLI command) is thin wiring at the end.
- **Coverage of acceptance criteria:**
  - All 7 ACs from the story are covered across Tasks 1-4
  - AC "emod lsp starts an LSP server that communicates over stdin/stdout using JSON-RPC" → Tasks 1 + 4
  - AC "responds to initialize, initialized, didOpen, didChange, shutdown" → Task 3
  - AC "diagnostics pushed on file open/change" → Task 3
  - AC "diagnostics include correct line, column, message" → Task 2 (conversion) + Task 3 (pipeline)
  - AC "parser errors as error-severity" → Task 3 (pipeline uses existing diagnostics)
  - AC "validator warnings as warning-severity" → Task 2 (severity mapping) + Task 3 (pipeline)
  - AC "handles unsaved documents (in-memory buffer)" → Task 2 (DocumentManager)
- **Deferred:** None from US-IDE-004. LSP completion, go-to-definition, formatting, hover, references, and semantic tokens are separate stories (US-IDE-006 through US-IDE-011).

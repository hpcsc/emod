# US-IDE-005: VS Code extension launches the LSP server

## Progress
- [x] Task 1: Set up VS Code extension TypeScript toolchain and activation manifest
- [x] Task 2: Implement LSP client with lifecycle management and error handling

---

## Story Reference

`user-stories/ide-support.md` — US-IDE-005: VS Code extension launches the LSP server

---

## Codebase Context

**Exploration findings:**

- **Extension layout:** `editors/vscode/` contains `package.json`, `language-configuration.json`, `syntaxes/emod.tmLanguage.json`, and `.vscodeignore`. No `src/` directory exists yet. No TypeScript toolchain is configured.

- **`package.json`:** Already registers the `emod` language with `.emod` extension (US-IDE-001), has `publisher` field (US-IDE-003), but lacks `activationEvents` and `main` entry point needed for LSP client code execution.

- **LSP server (Go):** Already implemented (US-IDE-004, completed). `internal/cli/lsp.go` exports `RunLSP()` which creates an `lsp.Server` with `os.Stdin`/`os.Stdout` and calls `Run()`. The server speaks JSON-RPC over Content-Length-framed stdin/stdout transport. Protocol types (`Diagnostic`, `PublishDiagnosticsParams`, `InitializeParams`, etc.) are defined in `internal/lsp/protocol.go` and `internal/lsp/transport.go`.

- **No existing extension source code:** All current extension behavior (syntax highlighting, bracket matching) is declarative via `package.json` contributes. LSP client requires imperative TypeScript code.

- **No npm dependencies yet:** `package.json` has no `dependencies` or `devDependencies` section. The standard VS Code LSP client library is `vscode-languageclient` (npm).

- **Dotfiles note:** Testing guidelines are referenced from their canonical paths at `~/.config/ai/guidelines/` (symlinked from dotfiles).

---

## Tasks

### Task 1: Set up VS Code extension TypeScript toolchain and activation manifest

**Language:** JavaScript/TypeScript

**Behavior:** The VS Code extension has a TypeScript build pipeline that compiles `src/extension.ts` into `out/extension.js`. The extension registers `onLanguage:emod` as its activation event and points VS Code to the compiled output as its main entry point.

**Acceptance Criteria:**
- [ ] `editors/vscode/package.json` gains `"activationEvents"` with `onLanguage:emod`
- [ ] `editors/vscode/package.json` gains `"main"` pointing to `"./out/extension.js"`
- [ ] `editors/vscode/package.json` gains a `"scripts"` section with `"compile": "tsc -p ./tsconfig.json"` and `"watch": "tsc -watch -p ./tsconfig.json"`
- [ ] `editors/vscode/package.json` gains `"devDependencies"` with `typescript`, `@types/vscode`, `@types/node`, and `vscode-languageclient`
- [ ] `editors/vscode/tsconfig.json` exists with compiler options targeting ES2020 or later, strict mode enabled, output directory set to `out/`, and root directory set to `src/`
- [ ] `editors/vscode/src/extension.ts` exists with exported `activate` and `deactivate` functions (skeleton, minimal logic)
- [ ] Running `npm run compile` in `editors/vscode/` produces `editors/vscode/out/extension.js`
- [ ] `out/` is already in `.vscodeignore` (verify or add)

**Affected Files/Modules:**
- `editors/vscode/package.json` — add `activationEvents`, `main`, `scripts`, `devDependencies`
- `editors/vscode/tsconfig.json` — new file; TypeScript compiler configuration
- `editors/vscode/src/extension.ts` — new file; activate/deactivate skeleton
- `editors/vscode/.vscodeignore` — add `out/` if not already present

**Patterns to Follow:**
- Follow the existing `package.json:1-30` structure for formatting (top-level fields order, indentation style) when adding new fields
- `.vscodeignore` entries follow the style already in place

**Testable:** No — build toolchain and configuration; verified by `npm run compile` succeeding and `out/extension.js` existing

**Verification:** `npm run compile` in `editors/vscode/` succeeds and produces `out/extension.js`

**Depends on:** None

---

### Task 2: Implement LSP client with lifecycle management and error handling

**Language:** JavaScript/TypeScript

**Behavior:** When a `.emod` file is opened in VS Code, the extension starts `emod lsp` as a child process, connects to it over stdin/stdout, and forwards LSP diagnostics from the server to VS Code (shown as squiggly underlines). If the `emod` binary is not on `$PATH`, the extension displays an error message with installation instructions. When the extension deactivates (e.g., last `.emod` file closes), the LSP server process is stopped.

**Acceptance Criteria:**
- [ ] `activate()` creates a `LanguageClient` from `vscode-languageclient` configured with `command: "emod"` and `args: ["lsp"]` using `StdioClientTransport`
- [ ] The client is started during activation and stopped during deactivation
- [ ] When the client fails to start because `emod` is not found on `$PATH`, a VS Code error notification is shown with a message like "emod binary not found. Please install emod from [URL]" or similar installation guidance
- [ ] Diagnostics pushed by the LSP server via `textDocument/publishDiagnostics` appear as squiggly underlines in the editor (handled automatically by `LanguageClient`)
- [ ] The LSP server process terminates when the extension deactivates (handled by `LanguageClient.stop()`)
- [ ] The extension builds and compiles without errors (`npm run compile`)

**Affected Files/Modules:**
- `editors/vscode/src/extension.ts` — implement `activate()` with LanguageClient creation, startup with error handling, and `deactivate()` with client stop
- `editors/vscode/package.json` — no changes expected (setup done in Task 1)

**Patterns to Follow:**
- The LanguageClient `serverOptions` follows the `command`/`args` pattern documented in the `vscode-languageclient` npm package README; the `clientOptions` sets `documentSelector` to `[{ language: "emod" }]`
- Error handling for binary-not-found follows the VS Code extension pattern of catching the `StartFailed` error from the LanguageClient and showing an error message via `vscode.window.showErrorMessage`
- Import pattern uses ES module-style imports consistent with TypeScript conventions

**Testable:** No — meaningful testing requires the VS Code runtime (`@vscode/test-electron`) and an installed `emod` binary. The `vscode-languageclient` library (a trusted Microsoft-maintained dependency) handles the LSP protocol, process lifecycle, and diagnostic forwarding — the extension's logic is thin wiring around it.

**Verification:** Load the extension in VS Code (via symlink or `.vsix`), open an `.emod` file, and verify diagnostics appear as squiggly underlines. Remove `emod` from PATH and verify error notification appears.

**Depends on:** Task 1

---

## Summary

| | |
|---|---|
| **Total tasks** | 2 |
| **Ordering rationale** | Dependency-first. Task 1 establishes the TypeScript build toolchain and extension manifest so that compiled extension code can be loaded by VS Code. Task 2 delivers all user-facing behavior (LSP client startup, error handling, diagnostics forwarding, lifecycle) in a single vertical slice, since `vscode-languageclient` handles the protocol details and these behaviors are tightly coupled in the activate/deactivate flow. |
| **Coverage of acceptance criteria** | |
| AC: Extension activates when `.emod` file opened | Task 1 (activationEvents) + Task 2 (client start on activate) |
| AC: Launches `emod lsp` as child process, stdin/stdout | Task 2 (LanguageClient with command/args) |
| AC: If binary not on PATH, show error with install instructions | Task 2 (StartFailed error handling) |
| AC: Diagnostics appear as squiggly underlines | Task 2 (LanguageClient forwards publishDiagnostics) |
| AC: LSP server stopped when last `.emod` file closed | Task 2 (client stopped on deactivation) |

# IDE Support for `.emod` Files — Proposal

## Problem

`.emod` files are plain text with no IDE recognition. Editors treat them as generic files with no syntax highlighting, error reporting, or navigation. This makes authoring and reviewing `.emod` models harder than it needs to be.

## Overview

IDE support is built in layers, each adding more capability:

| Layer | Feature | Effort | Editors Covered |
|-------|---------|--------|-----------------|
| 1. TextMate Grammar | Syntax highlighting | Small | VS Code, JetBrains, Sublime |
| 2. LSP Server | Diagnostics, autocomplete, go-to-definition, hover, formatting | Medium | Any LSP-capable editor |
| 3. Tree-sitter Grammar | Incremental parsing, syntax highlighting | Medium | Neovim, Zed, Helix, GitHub |

Each layer is independent but complementary. Layer 2 (LSP) provides the highest value since it reuses the existing Go parser infrastructure.

---

## Layer 1: TextMate Grammar

### What It Provides

A TextMate grammar maps regex patterns to scopes that editors use for color-coding:

- **Keywords:** `model`, `actor`, `context`, `aggregate`, `slice`, `command`, `event`, `fields`, `flow`, `trigger`, `view`, `automation`, `translation`, `subscribes`, `target`, `external_system`, `reads`, `source`, `external`
- **Strings:** `"..."`
- **Comments:** `# ...`
- **Identifiers:** PascalCase names like `ReserveRoom`, `RoomReserved`
- **Field modifiers:** `required`, `optional`
- **Field types:** `string`, `date`, `timestamp`, `int`
- **Operators:** `->`, `:`
- **Punctuation:** `{ }`, `[ ]`, `,`

Plus a `language-configuration.json` for bracket matching, auto-closing pairs, and comment toggling (`Cmd+/`).

### Artifact

```
editors/vscode/
  package.json                   # Extension manifest
  syntaxes/emod.tmLanguage.json  # TextMate grammar
  language-configuration.json    # Bracket matching, comments
```

### Editor Coverage

| Editor | How It Works |
|--------|-------------|
| VS Code | Install as extension (`.vsix` or marketplace) |
| JetBrains (GoLand, IntelliJ, etc.) | Settings > Editor > TextMate Bundles > add the directory |
| Sublime Text | Copy to `Packages/` directory |

### Distribution

**During development:** Live in the repo at `editors/vscode/`. Users symlink or point their editor at it.

```sh
# VS Code
ln -s /path/to/emod/editors/vscode ~/.vscode/extensions/emod

# JetBrains
# Settings > Editor > TextMate Bundles > point to editors/vscode/
```

**For broader distribution:**

| Channel | How | Pros | Cons |
|---------|-----|------|------|
| GitHub Releases (`.vsix`) | `npx @vscode/vsce package`, attach to release | No accounts needed, versioned | Manual install, no auto-updates |
| VS Code Marketplace | `npx @vscode/vsce publish` | Discoverable, auto-updates | Requires free Azure DevOps publisher account |
| Open VSX Registry | `npx ovsx publish` | Covers VS Codium, Gitpod, Cursor | Second account to maintain |

### Limitations

TextMate grammars are regex-based — they can highlight keywords and strings but cannot:

- Validate that referenced names exist
- Provide autocomplete based on context
- Report parser errors
- Navigate to definitions

These require Layer 2.

---

## Layer 2: Language Server Protocol (LSP)

### What It Provides

An LSP server is a background process that editors communicate with over JSON-RPC. It enables rich editing features across any editor that supports the protocol.

| LSP Feature | What It Does | Maps To |
|-------------|-------------|---------|
| `textDocument/publishDiagnostics` | Error squiggles in real-time | `internal/parser` errors + `internal/validator` diagnostics |
| `textDocument/completion` | Context-aware keyword suggestions | Inside `slice {}` suggest `command`, `event`, `view`, etc. |
| `textDocument/hover` | Info on hover (e.g., which context an event belongs to) | AST traversal |
| `textDocument/definition` | Jump from `subscribes [RoomReserved]` to the `event` block | AST name resolution |
| `textDocument/references` | Find all usages of a command or event name | AST name resolution |
| `textDocument/formatting` | Format on save | `emod fmt` |
| `textDocument/semanticTokens` | Richer highlighting (distinguish commands vs events vs views) | Lexer token types |

### Why Go

The existing codebase already provides the core machinery:

- `internal/lexer` — tokenization with position tracking
- `internal/parser` — recursive descent parser producing a full AST with positions
- `internal/validator` — semantic validation (e.g., referenced names exist)
- `internal/diagnostic` — error entries with filename, line, column

The LSP server wraps this infrastructure. On each file change, it re-lexes, re-parses, validates, and pushes diagnostics to the editor.

### Go LSP Libraries

| Library | Notes |
|---------|-------|
| `go.lsp.dev/protocol` + `go.lsp.dev/jsonrpc2` | Well-maintained, used by `gopls`. Full protocol types. |
| `github.com/tliron/glsp` | Simpler API, less boilerplate. Good for smaller LSPs. |

### Artifact

```
cmd/emod-lsp/main.go    # LSP server binary (or subcommand: emod lsp)
internal/lsp/            # LSP handler implementations
```

The server can be either a standalone binary (`emod-lsp`) or a subcommand of the existing CLI (`emod lsp`). The subcommand approach keeps distribution simple — one binary.

### Editor Coverage

| Editor | Integration |
|--------|-------------|
| VS Code | Extension launches `emod lsp` as a child process |
| Neovim | `nvim-lspconfig` entry pointing to `emod lsp` |
| JetBrains | LSP support (built-in since 2023.2+) or via LSP plugin |
| Helix | `languages.toml` entry |
| Zed | Extension or `settings.json` LSP config |
| Sublime Text | LSP package configuration |
| Emacs | `lsp-mode` or `eglot` configuration |

### Distribution

The LSP server is distributed as part of the `emod` binary:

- **Homebrew/Go install:** `go install github.com/hpcsc/emod/cmd/emod@latest` — users already have the LSP
- **GitHub Releases:** Pre-built binaries per platform
- **VS Code extension:** Can bundle the binary or download it on first launch

### Incremental Approach

Start with the highest-value features:

1. **Diagnostics** — parser errors + validator warnings as squiggles
2. **Completion** — keyword suggestions based on current scope
3. **Go-to-definition** — jump to event/command/view definitions
4. **Formatting** — `emod fmt` on save
5. **Hover, references, semantic tokens** — polish

---

## Layer 3: Tree-sitter Grammar

### What It Provides

Tree-sitter is an incremental parsing framework. A tree-sitter grammar for `.emod` enables:

- **Syntax highlighting** in Neovim, Zed, Helix (these use tree-sitter instead of TextMate)
- **GitHub code rendering** — `.emod` files in PRs and repos get syntax highlighting on github.com
- **Structural editing** — select/move/delete by AST node (e.g., select an entire `slice` block)
- **Incremental re-parsing** — only re-parses changed regions, so highlighting stays fast on large files

### How It Works

A `grammar.js` file defines the formal grammar using tree-sitter's JavaScript DSL:

```js
// Simplified example
module.exports = grammar({
  name: 'emod',
  rules: {
    source_file: $ => repeat($._definition),
    _definition: $ => choice($.model, $.actor, $.context),
    model: $ => seq('model', $.string),
    context: $ => seq('context', $.string, '{', repeat($.aggregate), '}'),
    // ...
  }
});
```

Tree-sitter generates a C parser from this, which editors load as a shared library.

### Artifact

```
editors/tree-sitter-emod/
  grammar.js              # Grammar definition
  src/                    # Generated C parser (auto-generated)
  queries/
    highlights.scm        # Highlight queries (maps nodes to colors)
    locals.scm            # Scope/reference queries
```

### Editor Coverage

| Editor | How It Works |
|--------|-------------|
| Neovim | `nvim-treesitter` plugin loads the grammar |
| Zed | Built-in tree-sitter support |
| Helix | Built-in tree-sitter support |
| GitHub | Submit grammar to `github-linguist` for `.emod` recognition |
| Emacs | `tree-sitter-langs` package |

VS Code has experimental tree-sitter support but primarily uses TextMate grammars.

### Distribution

| Channel | How |
|---------|-----|
| npm | `npm publish` as `tree-sitter-emod` — standard for tree-sitter grammars |
| Neovim `nvim-treesitter` | Submit PR to add `emod` as a supported language |
| GitHub Linguist | Submit PR to `github-linguist` repo for GitHub syntax highlighting |

### When to Build

Tree-sitter is worth building when:

- Neovim/Zed/Helix users need `.emod` support
- You want syntax highlighting on GitHub
- You want structural editing (select entire blocks by AST node)

It overlaps with Layer 1 (both provide highlighting) but targets different editors. If most users are on VS Code/JetBrains, Layer 1 + Layer 2 covers them fully.

---

## Recommendation

| Phase | What | Why |
|-------|------|-----|
| **Now** | Layer 1: TextMate grammar in `editors/vscode/` | Immediate highlighting, tiny effort, ship in the repo |
| **Next** | Layer 2: LSP server as `emod lsp` subcommand | Highest value — diagnostics, completion, go-to-def. Reuses existing parser. |
| **Later** | Layer 3: Tree-sitter grammar | When Neovim/Zed users need support or for GitHub rendering |

### Proposed Directory Structure

```
editors/
  vscode/
    package.json
    syntaxes/emod.tmLanguage.json
    language-configuration.json
  tree-sitter-emod/          # Layer 3 (future)
    grammar.js
    queries/highlights.scm

cmd/emod/main.go              # Existing CLI
internal/lsp/                  # Layer 2: LSP handlers
```

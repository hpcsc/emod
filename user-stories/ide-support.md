# IDE Support for `.emod` Files

## Overview

`.emod` files have no IDE recognition — editors treat them as plain text with no highlighting, error reporting, or navigation. This feature adds progressively richer editor support: syntax highlighting via TextMate and Tree-sitter grammars, and intelligent editing via a Language Server Protocol (LSP) server. The goal is broad editor coverage (VS Code, JetBrains, Neovim, Zed, Helix) and syntax highlighting on GitHub.

## Goals

- Provide syntax highlighting for `.emod` files in VS Code, JetBrains, Sublime, Neovim, Zed, and Helix
- Surface parser and validator diagnostics as real-time error squiggles in editors
- Enable GitHub to render `.emod` files with syntax highlighting in PRs and code browsing
- Distribute editor support with minimal friction (repo symlink for dev, marketplace/packages for broader use)

## User Stories

### US-IDE-001: Syntax highlighting via TextMate grammar

**Description:** As a model author using VS Code, JetBrains, or Sublime, I want `.emod` files to be syntax-highlighted so that I can visually distinguish keywords, strings, comments, identifiers, and operators while editing.

**Acceptance Criteria:**
- [ ] A TextMate grammar file exists at `editors/vscode/syntaxes/emod.tmLanguage.json`
- [ ] DSL keywords are highlighted as keywords: `model`, `actor`, `context`, `aggregate`, `slice`, `command`, `event`, `fields`, `flow`, `trigger`, `view`, `automation`, `translation`, `subscribes`, `target`, `external_system`, `reads`, `source`, `external`
- [ ] Quoted strings (`"..."`) are highlighted as strings
- [ ] Comments (`# ...`) are highlighted as comments
- [ ] Identifiers after keywords (e.g., `ReserveRoom` after `command`) are highlighted as entity names
- [ ] Field modifiers (`required`, `optional`) and field types (`string`, `date`, `timestamp`, `int`) are highlighted distinctly from keywords
- [ ] Operators (`->`, `:`) and punctuation (`{ }`, `[ ]`, `,`) are highlighted
- [ ] A VS Code extension manifest (`editors/vscode/package.json`) registers the grammar for `.emod` files

**Context:** TextMate grammars are regex-based and supported natively by VS Code, JetBrains (via TextMate Bundles), and Sublime Text. This is the fastest path to visual feedback.

---

### US-IDE-002: Bracket matching and comment toggling

**Description:** As a model author, I want my editor to auto-close braces and brackets and toggle line comments with a keyboard shortcut so that editing `.emod` files feels as smooth as editing any supported language.

**Acceptance Criteria:**
- [ ] A `language-configuration.json` file exists at `editors/vscode/language-configuration.json`
- [ ] Typing `{` auto-inserts `}`; typing `[` auto-inserts `]`; typing `"` auto-inserts `"`
- [ ] The comment toggle shortcut (`Cmd+/` on macOS, `Ctrl+/` on Linux/Windows) inserts or removes `# ` at the start of the line
- [ ] Bracket pairs `{}` and `[]` are matched and highlighted when the cursor is adjacent

**Depends on:** US-IDE-001

---

### US-IDE-003: Distribute TextMate extension via symlink and `.vsix`

**Description:** As a model author, I want to install the `.emod` syntax highlighting extension so that I can use it in my editor without building it from source.

**Acceptance Criteria:**
- [ ] Symlinking `editors/vscode/` into `~/.vscode/extensions/emod` activates the extension in VS Code
- [ ] Pointing JetBrains' TextMate Bundles setting at `editors/vscode/` activates highlighting in GoLand/IntelliJ
- [ ] Running `npx @vscode/vsce package` in `editors/vscode/` produces a `.vsix` file that installs via `code --install-extension`
- [ ] The repo README documents all three installation methods (symlink, `.vsix`, JetBrains bundle)

**Depends on:** US-IDE-002

---

### US-IDE-004: LSP server with real-time diagnostics

**Description:** As a model author, I want to see parser errors and validator warnings as squiggly underlines in my editor in real-time so that I can fix problems without switching to the terminal to run `emod validate`.

**Acceptance Criteria:**
- [ ] `emod lsp` starts an LSP server that communicates over stdin/stdout using JSON-RPC
- [ ] The server responds to `initialize`, `initialized`, `textDocument/didOpen`, `textDocument/didChange`, and `shutdown` requests
- [ ] On file open or change, the server re-parses the document and pushes diagnostics via `textDocument/publishDiagnostics`
- [ ] Each diagnostic includes the correct line, column, and an error message matching what `emod validate` would report
- [ ] Parser errors (e.g., unclosed braces, unexpected tokens) appear as error-severity diagnostics
- [ ] Validator warnings (e.g., referenced event name does not exist) appear as warning-severity diagnostics
- [ ] The server handles documents that are open but unsaved (uses in-memory buffer, not the file on disk)

**Context:** The existing `internal/lexer`, `internal/parser`, `internal/validator`, and `internal/diagnostic` packages already produce positioned diagnostics. The LSP server wraps this pipeline. The `emod lsp` subcommand keeps distribution simple — one binary.

**Depends on:** US-IDE-001

---

### US-IDE-005: VS Code extension launches the LSP server

**Description:** As a VS Code user, I want the emod extension to automatically start the LSP server when I open an `.emod` file so that I get diagnostics without manual setup.

**Acceptance Criteria:**
- [ ] The VS Code extension activates when a file with the `.emod` extension is opened
- [ ] The extension launches `emod lsp` as a child process and connects to it over stdin/stdout
- [ ] If the `emod` binary is not found on `$PATH`, the extension shows an error message with installation instructions
- [ ] Diagnostics from the LSP server appear as squiggly underlines in the editor
- [ ] When the last `.emod` file is closed, the LSP server process is stopped

**Depends on:** US-IDE-004

---

### US-IDE-006: LSP keyword completion

**Description:** As a model author, I want context-aware keyword suggestions as I type so that I can discover valid keywords for each block without consulting documentation.

**Acceptance Criteria:**
- [ ] The LSP server responds to `textDocument/completion` requests
- [ ] At the top level, completions include `model`, `actor`, `context`
- [ ] Inside a `context {}` block, completions include `aggregate`
- [ ] Inside an `aggregate {}` block, completions include `slice`
- [ ] Inside a `slice {}` block, completions include `command`, `event`, `trigger`, `view`, `automation`, `translation`, `flow`
- [ ] Inside a `command {}` or `event {}` block, completions include `fields`
- [ ] Inside a `fields {}` block, field type completions include `string`, `date`, `timestamp`, `int`; modifier completions include `required`, `optional`

**Depends on:** US-IDE-004

---

### US-IDE-007: LSP go-to-definition for names

**Description:** As a model author, I want to jump from a reference (e.g., an event name in a `subscribes` list) to its definition so that I can navigate large models quickly.

**Acceptance Criteria:**
- [ ] The LSP server responds to `textDocument/definition` requests
- [ ] Clicking an event name in a `subscribes [...]` list jumps to the `event` block where it is defined
- [ ] Clicking a command name in an `automation` or `translation` block jumps to the `command` block where it is defined
- [ ] Clicking a view name in a `trigger`'s `reads` or a `translation`'s `reads` jumps to the `view` block where it is defined
- [ ] Clicking a context name in `target context` jumps to the `context` block where it is defined
- [ ] If the definition is not found, no navigation occurs (no error shown to the user)

**Depends on:** US-IDE-004

---

### US-IDE-008: LSP format on save

**Description:** As a model author, I want my `.emod` file to be auto-formatted when I save so that models stay consistently formatted without running `emod fmt` manually.

**Acceptance Criteria:**
- [ ] The LSP server responds to `textDocument/formatting` requests
- [ ] The formatting result matches the output of `emod fmt`
- [ ] The editor applies the formatting edits on save (when the user has "format on save" enabled)
- [ ] Formatting preserves comments in their original positions

**Context:** This story depends on the `emod fmt` formatter being implemented (see US-004 in the main user stories).

**Depends on:** US-IDE-004, US-004

---

### US-IDE-009: LSP hover information

**Description:** As a model author, I want to hover over an identifier and see contextual information so that I can understand where an element is defined and how it fits in the model.

**Acceptance Criteria:**
- [ ] The LSP server responds to `textDocument/hover` requests
- [ ] Hovering over a command name shows its parent context and aggregate (e.g., "Command in Reservations > Reservation")
- [ ] Hovering over an event name shows its parent context and aggregate, plus its field list
- [ ] Hovering over a view name shows its subscribed events
- [ ] Hovering over a keyword (e.g., `automation`) shows a brief description of what the block does
- [ ] Hovering over a non-resolvable token returns no hover content (no error)

**Depends on:** US-IDE-007

---

### US-IDE-010: LSP find references

**Description:** As a model author, I want to find all places where a command, event, or view is referenced so that I can understand the impact of renaming or removing it.

**Acceptance Criteria:**
- [ ] The LSP server responds to `textDocument/references` requests
- [ ] Finding references on an event name returns all locations: its definition, `subscribes` lists, automation `trigger` references, and `flow` entries
- [ ] Finding references on a command name returns all locations: its definition, automation `command` references, translation `command` references, and `flow` entries
- [ ] Finding references on a view name returns all locations: its definition and `reads` references in triggers and translations
- [ ] Results include file path, line, and column for each reference

**Depends on:** US-IDE-007

---

### US-IDE-011: LSP semantic tokens

**Description:** As a model author, I want richer syntax highlighting that distinguishes between commands, events, views, and other identifiers by color so that I can visually scan the model structure at a glance.

**Acceptance Criteria:**
- [ ] The LSP server responds to `textDocument/semanticTokens/full` requests
- [ ] Command names are assigned a distinct token type (e.g., `function`)
- [ ] Event names are assigned a distinct token type (e.g., `event` or `type`)
- [ ] View names are assigned a distinct token type (e.g., `class`)
- [ ] Actor names, context names, and aggregate names each have distinct token types
- [ ] Semantic tokens override TextMate highlighting when both are available, providing more accurate coloring

**Context:** Semantic tokens require the LSP to understand the AST — they go beyond regex-based TextMate scopes. For example, the identifier `RoomReserved` after `event` gets a different color than `ReserveRoom` after `command`.

**Depends on:** US-IDE-004

---

### US-IDE-012: Tree-sitter grammar for `.emod`

**Description:** As a model author using Neovim, Zed, or Helix, I want `.emod` files to be syntax-highlighted so that I have the same visual feedback as VS Code users.

**Acceptance Criteria:**
- [ ] A tree-sitter grammar exists at `editors/tree-sitter-emod/grammar.js` that defines the full `.emod` syntax
- [ ] `tree-sitter generate` produces a working C parser from the grammar
- [ ] `tree-sitter test` passes with test cases covering all block types (model, actor, context, aggregate, slice, command, event, view, automation, translation, flow, trigger, fields)
- [ ] Highlight queries exist at `editors/tree-sitter-emod/queries/highlights.scm` mapping nodes to standard highlight groups (keyword, string, comment, function, type, operator, punctuation)
- [ ] The grammar parses the `examples/all_patterns.emod` file without errors

---

### US-IDE-013: Tree-sitter Neovim integration

**Description:** As a Neovim user, I want to install the emod tree-sitter grammar so that `.emod` files are highlighted and support structural selection.

**Acceptance Criteria:**
- [ ] The tree-sitter grammar can be installed via `nvim-treesitter` by adding a custom parser config
- [ ] Opening an `.emod` file in Neovim shows syntax highlighting matching the highlight queries
- [ ] Tree-sitter text objects work (e.g., selecting an entire `slice` block or `fields` block)
- [ ] Installation instructions are documented in the repo README

**Depends on:** US-IDE-012

---

### US-IDE-014: GitHub syntax highlighting for `.emod` files

**Description:** As a model author, I want `.emod` files to be syntax-highlighted on GitHub so that event models are readable in pull requests and code browsing.

**Acceptance Criteria:**
- [ ] The tree-sitter grammar is published to npm as `tree-sitter-emod`
- [ ] A PR is submitted to the `github-linguist` repository to register `.emod` as a recognized language with the tree-sitter grammar
- [ ] Once accepted, `.emod` files on GitHub render with syntax highlighting matching the highlight queries

**Context:** GitHub uses tree-sitter grammars via the `github-linguist` project. Recognition requires the grammar to be published on npm and a PR to linguist with a sample file and grammar reference. This is a long-lead item — the linguist PR review process can take weeks to months.

**Depends on:** US-IDE-012

---

### US-IDE-015: Publish VS Code extension to marketplace

**Description:** As a model author, I want to install the emod extension from the VS Code Marketplace so that I get syntax highlighting and LSP features with one click and receive automatic updates.

**Acceptance Criteria:**
- [ ] The extension is published to the VS Code Marketplace under a publisher account
- [ ] Searching "emod" in the VS Code Extensions panel finds and installs the extension
- [ ] The extension is also published to the Open VSX Registry for VS Codium, Gitpod, and Cursor users
- [ ] The extension version follows semver and is updated when grammar or LSP features change

**Depends on:** US-IDE-005

## Non-Goals

- Code generation from the LSP (e.g., generating boilerplate slices) — this is a CLI concern, not an editor concern
- Rename refactoring via LSP — useful but lower priority than navigation and diagnostics
- Multi-file model support in the LSP (e.g., cross-file go-to-definition) — deferred until multi-file models are supported by the parser
- JetBrains plugin marketplace distribution — TextMate bundles and built-in LSP support cover JetBrains users without a dedicated plugin
- Emacs-specific configuration — Emacs users can configure `eglot`/`lsp-mode` manually using the LSP server

## Open Questions

- Should the VS Code extension bundle the `emod` binary, or require it to be installed separately on `$PATH`?
- For the tree-sitter grammar, should it live in the emod repo (`editors/tree-sitter-emod/`) or in a separate `tree-sitter-emod` repo (standard convention for tree-sitter grammars)?
- Should the LSP support workspace-level diagnostics (validate all `.emod` files in a directory) or only open files?
- What publisher name should be used for the VS Code Marketplace?

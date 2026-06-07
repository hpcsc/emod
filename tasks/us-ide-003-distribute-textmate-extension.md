# US-IDE-003: Distribute TextMate extension via symlink and `.vsix`

## Progress
- [x] Task 1: Add `publisher` and `.vscodeignore` to enable `.vsix` packaging
- [ ] Task 2: Document all three editor installation methods in README

---

## Story Reference

**US-IDE-003:** Distribute TextMate extension via symlink and `.vsix`

**Description:** As a model author, I want to install the `.emod` syntax highlighting extension so that I can use it in my editor without building it from source.

**Acceptance Criteria:**
- Symlinking `editors/vscode/` into `~/.vscode/extensions/emod` activates the extension in VS Code
- Pointing JetBrains' TextMate Bundles setting at `editors/vscode/` activates highlighting in GoLand/IntelliJ
- Running `npx @vscode/vsce package` in `editors/vscode/` produces a `.vsix` file that installs via `code --install-extension`
- The repo README documents all three installation methods (symlink, `.vsix`, JetBrains bundle)

---

## Codebase Context

**Exploration findings:**

- **Extension layout (already exists):** `editors/vscode/` contains `package.json`, `language-configuration.json`, and `syntaxes/emod.tmLanguage.json`. This structure is functionally complete for both VS Code and JetBrains — the grammar activates when symlinked into `~/.vscode/extensions/` or pointed at via JetBrains' TextMate Bundles setting.

- **Missing `publisher` field:** The `package.json` manifest lacks a `publisher` field, which is required by `vsce package`. Without it, `npx @vscode/vsce package` exits with an error before producing a `.vsix`.

- **No `.vscodeignore`:** There is no `.vscodeignore` file. Without it, `vsce package` includes all files in `editors/vscode/` (including hidden files, tests, etc.), producing a needlessly large package. Adding one is best practice.

- **README gap:** The root `README.md` documents only the Go CLI tool (`emod validate`, `emod lint`, `emod fmt`, `emod diagram`, `emod export`, `emod slices`). There is no section about editor installation or the VS Code extension.

---

## Tasks

### Task 1: Add `publisher` and `.vscodeignore` to enable `.vsix` packaging

**Language:** JavaScript/TypeScript

**Behavior:** Running `npx @vscode/vsce package` inside `editors/vscode/` produces a valid `emod-<version>.vsix` file in that directory.

The `.vsix` file can be installed by running `code --install-extension emod-<version>.vsix`, which activates syntax highlighting for `.emod` files.

**Acceptance Criteria:**
- [ ] `npx @vscode/vsce package` in `editors/vscode/` exits successfully and produces a `emod-<version>.vsix` file
- [ ] The produced `.vsix` file does not include unnecessary files (e.g., `.gitignore`, `language-configuration.json` if not needed at runtime)

**Affected Files/Modules:**
- `editors/vscode/package.json` — add `"publisher"` field with an appropriate value (e.g., `"hpcsc"`)
- `editors/vscode/.vscodeignore` — new file; standard exclude patterns for packaging

**Patterns to Follow:**
- See `editors/vscode/package.json:1-29` for the manifest structure — add `publisher` alongside the existing top-level fields (`name`, `displayName`, `version`, etc.)

**Testable:** Yes — `test -f editors/vscode/emod-*.vsix` after running the packaging command; `grep '"publisher"' editors/vscode/package.json`; `test -f editors/vscode/.vscodeignore`

**Verification:** `ls editors/vscode/emod-*.vsix` shows a non-empty package file

**Depends on:** None

---

### Task 2: Document all three editor installation methods in README

**Language:** JavaScript/TypeScript

**Behavior:** The repo `README.md` contains a new section titled "Editor Setup" (or similar) that documents three ways to install the `.emod` syntax highlighting extension:
1. **Symlink method** for VS Code
2. **`.vsix` method** for VS Code
3. **JetBrains TextMate bundle** for GoLand/IntelliJ

Each method includes the exact command(s) the user needs to run.

**Acceptance Criteria:**
- [ ] README documents the symlink command for VS Code: `ln -sf "$(pwd)/editors/vscode" ~/.vscode/extensions/emod`
- [ ] README documents building and installing the `.vsix` with `npx @vscode/vsce package` and `code --install-extension`
- [ ] README documents pointing JetBrains' TextMate Bundles setting at `editors/vscode/`

**Affected Files/Modules:**
- `README.md` — add a new "Editor Setup" section (placed after "Quick Start" or before "Development")

**Patterns to Follow:**
- See `README.md:6-31` for existing "Install" section formatting — match the heading style, code block style, and command presentation

**Testable:** No — documentation; requires human review

**Verification:** Visual inspection of `README.md` shows all three methods clearly documented with correct commands

**Depends on:** Task 1

---

## Summary

| | |
|---|---|
| **Total tasks** | 2 |
| **Ordering rationale** | Task 1 first (infrastructure prerequisite for Task 2). The publisher field is needed for `vsce package`; the README docs reference all three methods including `.vsix`, so the packaging must work before the docs are accurate. |
| **Coverage** | AC1 (symlink) → Task 2; AC2 (JetBrains) → Task 2; AC3 (.vsix) → Tasks 1 & 2; AC4 (README) → Task 2 |

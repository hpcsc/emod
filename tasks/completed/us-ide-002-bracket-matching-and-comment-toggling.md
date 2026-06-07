# US-IDE-002 — Bracket Matching and Comment Toggling

## Progress
- [x] Task 1: Create language-configuration.json and wire it into package.json

---

## Story Reference

**User Story:** US-IDE-002 — Bracket matching and comment toggling

**Original source:** Inline (provided by user)

---

## Codebase Context

**Exploration findings:**

- The VS Code extension lives in `editors/vscode/`.
- `editors/vscode/package.json` already registers the `emod` language (id: `"emod"`, extensions: `[".emod"]`) and its TextMate grammar (`./syntaxes/emod.tmLanguage.json`).
- A `language-configuration.json` file does not yet exist — it must be created at `editors/vscode/language-configuration.json`.
- The existing `package.json` does not have a `configuration` field under the language contribution block; it needs to be updated to add the `configuration` property pointing to the new file.
- The TextMate grammar at `syntaxes/emod.tmLanguage.json` already defines `#` as a line comment (`comment.line.number-sign.emod`), which is consistent with the comment toggle behavior.

**Relevant types/modules:**
- `editors/vscode/package.json` — language registration entry point
- `editors/vscode/syntaxes/emod.tmLanguage.json` — existing grammar (no changes needed)
- `editors/vscode/language-configuration.json` — file to be created (VS Code language configuration)

---

## Tasks

### Task 1: Create language-configuration.json and wire it into package.json

**Language:** Go

**Behavior:** VS Code auto-closes `{}`, `[]`, and `""` pairs in `.emod` files, highlights matching brackets when the cursor is adjacent, and toggles `# ` line comments with the comment shortcut (Cmd+/ on macOS, Ctrl+/ on Linux/Windows).

**Acceptance Criteria:**
- [ ] A `language-configuration.json` file exists at `editors/vscode/language-configuration.json`
- [ ] Typing `{` auto-inserts `}`; typing `[` auto-inserts `]`; typing `"` auto-inserts `"`
- [ ] The comment toggle shortcut (`Cmd+/` on macOS, `Ctrl+/` on Linux/Windows) inserts or removes `# ` at the start of the line
- [ ] Bracket pairs `{}` and `[]` are matched and highlighted when the cursor is adjacent

**Affected Files/Modules:**
- `editors/vscode/language-configuration.json` — [create] VS Code language configuration with `comments`, `brackets`, `autoClosingPairs`, and `surroundingPairs` sections
- `editors/vscode/package.json` — [edit] add `"configuration": "./language-configuration.json"` to the `emod` language entry under `contributes.languages[0]`

**Patterns to Follow:**
- Extend the existing language registration pattern in `editors/vscode/package.json:13-18` by adding the `configuration` field to the existing language object (analogous to how `grammars` entries reference their `path`).
- VS Code language-configuration.json format: `comments.lineComment` for comment toggling, `brackets` for matching pairs, `autoClosingPairs` for auto-insertion, `surroundingPairs` for surrounding selected text. Refer to VS Code's official Language Configuration guide for the schema.

**Testable:** No — This is VS Code extension configuration (JSON metadata). There is no Go public API, HTTP handler, or CLI command to test against. Verification is done by loading the extension in VS Code and exercising the behaviors manually, or by inspecting the file for correct JSON.

**Verification:** The extension loads without errors in VS Code's Extension Host. Opening an `.emod` file and typing `{`, `[`, or `"` produces the corresponding closing character. Selecting text and pressing `Cmd+/` (macOS) / `Ctrl+/` (other) toggles `# ` line comments. Placing the cursor on a bracket highlights its matching pair.

**Depends on:** None (prerequisite US-IDE-001 is already completed)

---

## Summary

- **Total tasks:** 1
- **Ordering rationale:** Single task — all acceptance criteria are delivered by creating one configuration file and making one trivial update to `package.json`. These changes are tightly coupled (the `package.json` reference is required for VS Code to discover the language configuration) and cannot be independently verified.
- **Coverage:** All four acceptance criteria from US-IDE-002 are covered by this single task.
- **Deferred:** None.

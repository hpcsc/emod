# US-IDE-001: Syntax highlighting via TextMate grammar

## Progress
- [x] Task 1: Create TextMate grammar file with all syntax highlighting rules for `.emod` files
- [x] Task 2: Create VS Code extension manifest registering the grammar for `.emod` files

## Story Reference

**Source:** `user-stories/ide-support.md` — US-IDE-001

**Summary:** As a model author using VS Code, JetBrains, or Sublime, I want `.emod` files to be syntax-highlighted so that I can visually distinguish keywords, strings, comments, identifiers, and operators while editing.

**Acceptance Criteria (from story):**
- [ ] A TextMate grammar file exists at `editors/vscode/syntaxes/emod.tmLanguage.json`
- [ ] DSL keywords are highlighted as keywords: `model`, `actor`, `context`, `aggregate`, `slice`, `command`, `event`, `fields`, `flow`, `trigger`, `view`, `automation`, `translation`, `subscribes`, `target`, `external_system`, `reads`, `source`, `external`
- [ ] Quoted strings (`"..."`) are highlighted as strings
- [ ] Comments (`# ...`) are highlighted as comments
- [ ] Identifiers after keywords (e.g., `ReserveRoom` after `command`) are highlighted as entity names
- [ ] Field modifiers (`required`, `optional`) and field types (`string`, `date`, `timestamp`, `int`) are highlighted distinctly from keywords
- [ ] Operators (`->`, `:`) and punctuation (`{ }`, `[ ]`, `,`) are highlighted
- [ ] A VS Code extension manifest (`editors/vscode/package.json`) registers the grammar for `.emod` files

**Depends on:** None

## Codebase Context

**No existing editor infrastructure:** The `editors/` directory does not exist yet — the entire tree (`editors/vscode/syntaxes/`) needs to be created from scratch. There are no existing TextMate grammars or VS Code extension manifests in this repository.

**Existing lexer (`internal/lexer/token.go` and `tokenizer.go`):** The lexer already recognizes all DSL keywords as distinct token types (`KeywordModel`, `KeywordActor`, `KeywordContext`, `KeywordAggregate`, `KeywordSlice`, `KeywordCommand`, `KeywordEvent`, `KeywordFields`, `KeywordFlow`, `KeywordTrigger`, `KeywordView`, `KeywordAutomation`, `KeywordTranslation`, `KeywordSubscribes`, `KeywordTarget`, `KeywordExternalSystem`, `KeywordReads`, `KeywordSource`, `KeywordExternal`). Field modifiers (`required`, `optional`) and field types (`string`, `date`, `timestamp`, `int`) are **not** separate token types — the lexer treats them as plain `Identifier` tokens (confirmed in `internal/lexer/tokenizer_test.go:48`). This means the TextMate grammar must highlight field modifiers/types separately from block keywords, using regex patterns in the grammar rather than relying on the lexer's classification.

**Example file:** `examples/all_patterns.emod` shows the full syntax with all pattern types (command, view, automation, translation) and serves as a validation target for checking that all highlight rules match correctly.

**Token types and punctuation already present in the lexer (for reference):**
- Keywords: model, actor, context, aggregate, slice, command, event, fields, flow, trigger, view, automation, translation, subscribes, target, external_system, reads, source, external
- String literals: `"..."` → `String` token
- Comments: `# ...` → `Comment` token
- Operators/punctuation: `->` (Arrow), `:` (Colon), `{`/`}` (Braces), `[`/`]` (Brackets), `,` (Comma), identifiers

**TextMate grammar structure:** A `.tmLanguage.json` file uses regex-based match patterns organized in a `repository` for reusability. Each pattern assigns TextMate scope names (e.g., `keyword.control.emod`, `string.quoted.emod`, `comment.line.number-sign.emod`) which VS Code maps to its default color theme. The `scopeName` should follow the convention `source.emod`.

## Tasks

### Task 1: Create TextMate grammar file with all syntax highlighting rules for `.emod` files

**Behavior:** A TextMate grammar file at `editors/vscode/syntaxes/emod.tmLanguage.json` defines regex-based highlighting patterns for all `.emod` DSL constructs: block keywords, quoted strings, line comments, entity names (identifiers after keywords), field modifiers/types, and operators/punctuation. Loading this grammar in an editor that supports TextMate (VS Code, JetBrains, Sublime) produces syntax-colored `.emod` files.

**Acceptance Criteria:**
- [ ] The file `editors/vscode/syntaxes/emod.tmLanguage.json` exists with valid JSON
- [ ] The grammar defines `scopeName: "source.emod"` and `fileTypes: ["emod"]`
- [ ] DSL block keywords (`model`, `actor`, `context`, `aggregate`, `slice`, `command`, `event`, `fields`, `flow`, `trigger`, `view`, `automation`, `translation`, `subscribes`, `target`, `external_system`, `reads`, `source`, `external`) are matched and assigned to a keyword TextMate scope (e.g., `keyword.control.emod`)
- [ ] Quoted strings (`"..."`) are matched and assigned to a string TextMate scope (e.g., `string.quoted.double.emod`)
- [ ] Line comments (`# ...` to end of line) are matched and assigned to a comment TextMate scope (e.g., `comment.line.number-sign.emod`)
- [ ] Identifiers that appear immediately after a block keyword (e.g., `ReserveRoom` after `command`, `Guest` after `actor`, `Reservations` after `context`) are matched and assigned to an entity name scope (e.g., `entity.name.function.emod`)
- [ ] Field modifier keywords (`required`, `optional`) and field type keywords (`string`, `date`, `timestamp`, `int`) are matched and assigned to a scope distinct from block keywords (e.g., `storage.type.emod` for field types, `storage.modifier.emod` for modifiers)
- [ ] Operators (`->`, `:`) and punctuation (`{`, `}`, `[`, `]`, `,`) are matched and assigned to appropriate scopes (e.g., `keyword.operator.emod`)
- [ ] The grammar successfully highlights all constructs in the `examples/all_patterns.emod` file without errors

**Affected Files/Modules:**
- `editors/vscode/syntaxes/emod.tmLanguage.json` (new) — the complete TextMate grammar file

**Patterns to Follow:**
- Follow the TextMate grammar conventions used by VS Code built-in language extensions (available in the VS Code source at `extensions/`). The grammar uses regex match patterns with a `repository` structure for reusable capture groups, a `scopeName` of `source.emod`, and `fileTypes: ["emod"]`.
- Use the token definitions in `internal/lexer/token.go:52-117` and `internal/lexer/tokenizer.go:178-221` as the authoritative source for which strings are keywords vs identifiers. Note that field modifiers/types (`required`, `optional`, `string`, `date`, `timestamp`, `int`) are treated as plain identifiers by the lexer but must be highlighted distinctively in the grammar.

**Testable:** No — TextMate grammars are JSON configuration files with no automated test harness in this project. Verification is done by loading the grammar in an editor and visually inspecting `examples/all_patterns.emod`, or by using a TextMate grammar validator.

**Verification:** `python3 -m json.tool editors/vscode/syntaxes/emod.tmLanguage.json` confirms valid JSON. Manual verification: install the extension in VS Code (via symlink or direct load) and open `examples/all_patterns.emod` to confirm all construct types are highlighted correctly.

**Depends on:** None

---

### Task 2: Create VS Code extension manifest registering the grammar for `.emod` files

**Behavior:** A `package.json` VS Code extension manifest at `editors/vscode/package.json` declares the extension metadata and registers the TextMate grammar for `.emod` files, making the syntax highlighting available when the extension is installed in VS Code. The `emod.tmLanguage.json` grammar from Task 1 is referenced as the grammar contribution.

**Acceptance Criteria:**
- [ ] The file `editors/vscode/package.json` exists with valid JSON
- [ ] The manifest includes `name: "emod"`, `displayName: "EMOD"` (or similar descriptive name), and a `description`
- [ ] The manifest sets `activationEvents: ["onLanguage:emod"]` or uses the modern `languages` contribution that implicitly activates
- [ ] The manifest includes a `contributes.languages` entry registering `.emod` as a file extension with language identifier `emod` and the grammar from Task 1
- [ ] The manifest includes a `contributes.grammars` entry mapping language `emod` to `./syntaxes/emod.tmLanguage.json`
- [ ] The manifest specifies a `version` (starting at `0.1.0`) and follows VS Code extension structure conventions (e.g., `engines.vscode`, `categories: ["Programming Languages"]`)
- [ ] The extension activates and applies syntax highlighting when an `.emod` file is opened in VS Code (verified by symlinking `editors/vscode/` into `~/.vscode/extensions/emod/` or using a local VS Code Extension Development Host)

**Affected Files/Modules:**
- `editors/vscode/package.json` (new) — the VS Code extension manifest

**Patterns to Follow:**
- Follow the VS Code extension manifest structure documented at `https://code.visualstudio.com/api/references/extension-manifest`, specifically the `contributes.languages` and `contributes.grammars` contribution points.
- Follow the existing `internal/viewer/package.json` for basic JSON formatting conventions used in this project (no trailing commas, consistent indentation).
- The grammar path in the manifest is `./syntaxes/emod.tmLanguage.json` (relative to the extension root).

**Testable:** No — VS Code extension manifests are JSON configuration files. Verification is done by installing the extension and visually confirming highlighting works.

**Verification:** `python3 -m json.tool editors/vscode/package.json` confirms valid JSON. Manual verification: symlink `editors/vscode/` to `~/.vscode/extensions/emod/`, reload VS Code, and open `examples/all_patterns.emod` to confirm syntax highlighting appears.

**Depends on:** Task 1

## Summary

**Total tasks:** 2

**Ordering rationale:** Dependency-first. Task 1 (grammar) must exist before Task 2 (manifest) can reference it. Task 1 is the core deliverable — the grammar file is independently useful for JetBrains and Sublime users even without the VS Code manifest. Task 2 wraps it for VS Code distribution.

**Acceptance criteria coverage:**

| Story Acceptance Criterion | Covered By |
|---|---|
| TextMate grammar file exists at `editors/vscode/syntaxes/emod.tmLanguage.json` | Task 1 |
| DSL keywords are highlighted as keywords | Task 1 |
| Quoted strings are highlighted as strings | Task 1 |
| Comments are highlighted as comments | Task 1 |
| Identifiers after keywords are highlighted as entity names | Task 1 |
| Field modifiers (`required`, `optional`) and field types (`string`, `date`, `timestamp`, `int`) are highlighted distinctly from keywords | Task 1 |
| Operators and punctuation are highlighted | Task 1 |
| VS Code extension manifest registers the grammar for `.emod` files | Task 2 |

**Deferred:** None. All eight acceptance criteria from the story are covered.

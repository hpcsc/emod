# US-IDE-013: Tree-sitter Neovim integration

## Progress
- [x] Task 1: Create textobjects query file for structural selection
- [x] Task 2: Create folds and indents query files
- [ ] Task 3: Update README with verified Neovim installation instructions

## Story Reference
**US-IDE-013** from conversation — Tree-sitter Neovim integration for `.emod` files.

## Codebase Context

The tree-sitter grammar at `editors/tree-sitter-emod/` (US-IDE-012) is complete:
- `grammar.js` defines all DSL constructs (model, actor, context, aggregate, slice, command, event, fields, flow, trigger, view, automation, translation)
- `queries/highlights.scm` maps nodes to standard highlight groups
- `package.json` declares `tree-sitter-emod` with proper npm metadata
- Corpus tests under `test/corpus/` cover all block types

What does NOT yet exist:
- `queries/textobjects.scm` — needed for structural selection (the main AC)
- `queries/folds.scm` — for code folding
- `queries/indents.scm` — for auto-indentation

The README already has a Neovim section (lines 155-185) with a basic `nvim-treesitter` parser config and `nvim-lspconfig` LSP integration. It does not include `nvim-treesitter-textobjects` setup or installation verification steps.

---

## Tasks

### Task 1: Create textobjects query file for structural selection

**Behavior:** Neovim users can visually select entire structural blocks (e.g., an entire `slice`, `fields`, `command`, or `event` block) using tree-sitter text objects via `nvim-treesitter-textobjects`.

**Acceptance Criteria:**
- [ ] `editors/tree-sitter-emod/queries/textobjects.scm` exists with text object queries for all major block types
- [ ] Captures cover at minimum: `slice_definition`, `fields_block`, `command_definition`, `event_definition`, `view_definition`, `flow_definition`, `trigger_definition`, `automation_definition`, `translation_definition`, and `aggregate_definition` — each as both `@block.inner` and `@block.outer`
- [ ] `@block.inner` captures the block contents (inside `{...}`) and `@block.outer` captures the full block (keyword + name + `{...}`)
- [ ] Query syntax is valid tree-sitter SCM format (no parse errors when loaded by `nvim-treesitter-textobjects`)
- [ ] Queries reference only node type names that exist in the grammar (defined in `grammar.js`)

**Affected Files/Modules:**
- `editors/tree-sitter-emod/queries/textobjects.scm` — new text object query file

**Patterns to Follow:**
- The highlight queries in `queries/highlights.scm` demonstrate how node types from the grammar are referenced in `.scm` query files — follow the same node type naming conventions
- The grammar rule names in `grammar.js` define the available node types (all `_definition` rules and `fields_block` are relevant)

**Language:** JavaScript/TypeScript

**Testable:** No — text object queries produce no independently observable output outside of Neovim's `nvim-treesitter-textobjects` plugin; their correctness is verified by structural validation (syntax matches grammar node types) and by downstream editor behavior.

**Verification:** File exists at the expected path; queries reference only valid node types from `grammar.js`; `tree-sitter parse` on a sample `.emod` file confirms the node types exist in the parse tree.

**Depends on:** None (the grammar and all node types are already defined by US-IDE-012)

---

### Task 2: Create folds and indents query files

**Behavior:** Neovim can fold structural blocks (e.g., collapse a `slice` or `context` block) and auto-indent content inside blocks based on tree-sitter fold and indent queries.

**Acceptance Criteria:**
- [ ] `editors/tree-sitter-emod/queries/folds.scm` exists with fold queries capturing all `{...}` delimited blocks
- [ ] Fold query captures use `@fold` marker for each block-delimited node type (at minimum: `context_definition`, `aggregate_definition`, `slice_definition`, `command_definition`, `event_definition`, `view_definition`, `automation_definition`, `translation_definition`, `flow_definition`, `trigger_definition`, `fields_block`, `subscribes_block`)
- [ ] `editors/tree-sitter-emod/queries/indents.scm` exists with indent queries for block nodes
- [ ] Indent query uses `@indent` and `@dedent` (or `@branch` as appropriate) on block-opening and block-closing nodes for all `{...}` delimited constructs
- [ ] Query syntax is valid tree-sitter SCM format

**Affected Files/Modules:**
- `editors/tree-sitter-emod/queries/folds.scm` — new fold query file
- `editors/tree-sitter-emod/queries/indents.scm` — new indent query file

**Patterns to Follow:**
- The highlight queries in `queries/highlights.scm` demonstrate SCM query syntax conventions for this project
- Fold queries follow the standard `@fold` capture convention expected by Neovim's tree-sitter fold provider
- Indent queries follow standard `@indent`/`@dedent` convention expected by Neovim's tree-sitter indent provider
- All captured node types must match rule names in `grammar.js`

**Language:** JavaScript/TypeScript

**Testable:** No — fold and indent queries produce no independently observable output outside of Neovim; their correctness is verified by structural validation and downstream editor behavior.

**Verification:** Files exist at the expected paths; queries reference only valid node types from `grammar.js`; syntax conforms to SCM format (no tree-sitter query parse errors).

**Depends on:** None (independent of Task 1; both are query files referencing the same grammar node types)

---

### Task 3: Update README with verified Neovim installation instructions

**Behavior:** A user following the README can successfully install the tree-sitter grammar, see syntax highlighting in `.emod` files, and configure text objects for structural selection in Neovim.

**Acceptance Criteria:**
- [ ] The existing Neovim `nvim-treesitter` parser config (lines 159-171) is validated and corrected if needed
- [ ] The `url` path in the parser config example uses a repo-relative instruction (e.g., `path/to/emod` or a git-based URL) that works for both local clones and plugin managers
- [ ] Instructions include the `nvim-treesitter-textobjects` configuration snippet required for text objects from Task 1 (pointing to the `textobjects.scm` queries)
- [ ] A verification step is documented (e.g., `:Inspect` in Neovim to check highlighting, `vaB` or text object key to test structural selection)
- [ ] The existing LSP integration section (lines 173-185) is preserved or updated if relevant
- [ ] Grammar build step (`tree-sitter generate`) is referenced as a prerequisite

**Affected Files/Modules:**
- `README.md` — modified; update Neovim section (lines 155-185)

**Patterns to Follow:**
- The existing editor setup sections in `README.md` (lines 144-185) establish the documentation pattern: a header per editor, code blocks for config, and a brief explanation of what each config enables
- The Zed section (lines 187-199) and JetBrains section (lines 131-143) follow the same structure and tone

**Language:** Generic

**Testable:** No — documentation correctness is verified by manual review and by following the steps in Neovim.

**Verification:** README renders correctly on GitHub; all code blocks contain valid Lua; instructions are self-contained and actionable without referring to external sources.

**Depends on:** Task 1 (should include `nvim-treesitter-textobjects` configuration that references `textobjects.scm`)

---

## Summary

**Total tasks:** 3

**Task ordering rationale:** Dependency-first. Task 1 creates the primary new artifact required by the acceptance criteria (textobjects.scm). Task 2 creates additional quality-of-life query files (folds, indents) that are independent of Task 1 but grouped as editor-enhancement companion work. Task 3 updates documentation and depends on Task 1 so it can reference the textobjects configuration.

**Acceptance criteria coverage:**

| Acceptance criterion | Covered by |
|---|---|
| Grammar installable via `nvim-treesitter` custom parser config | Task 3 (verify + update existing config) |
| Opening `.emod` shows syntax highlighting | Task 3 (verify existing highlights.scm works via Neovim config) |
| Tree-sitter text objects work | Task 1 (textobjects.scm) |
| Installation instructions documented in README | Task 3 (README update) |

All four story acceptance criteria are covered. No criteria are deferred.

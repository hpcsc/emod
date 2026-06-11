# US-IDE-012: Tree-sitter grammar for `.emod`

## Progress
- [x] Task 1: Initialize tree-sitter project and implement grammar for top-level blocks with lexical tokens
- [x] Task 2: Implement grammar rules for all slice-content blocks and add corpus tests
- [x] Task 3: Create highlight queries for syntax highlighting

## Story Reference
**US-IDE-012** from `user-stories/ide-support.md` — Tree-sitter grammar for `.emod`

## Codebase Context

The existing Go parser (`internal/parser/parser.go`, `internal/ast/ast.go`, `internal/lexer/token.go`) defines the complete `.emod` DSL syntax, which a tree-sitter grammar must replicate in JavaScript DSL. The TextMate grammar at `editors/vscode/syntaxes/emod.tmLanguage.json` provides a reference for which tokens map to which highlight groups. Three example `.emod` files exist at `examples/` — the most comprehensive being `examples/all_patterns.emod`, which exercises every block type. No tree-sitter directory exists yet; the target location is `editors/tree-sitter-emod/`.

---

## Tasks

### Task 1: Initialize tree-sitter project and implement grammar for top-level blocks with lexical tokens

**Behavior:** A tree-sitter grammar project exists at `editors/tree-sitter-emod/` that defines lexical tokens (comments, strings, identifiers, all keywords, punctuation) and grammar rules for the top-level block hierarchy (`model`, `actor`, `context`, `aggregate`, `slice`). `tree-sitter generate` produces a working C parser.

**Acceptance Criteria:**
- [ ] `editors/tree-sitter-emod/grammar.js` defines word rules for all DSL keywords (model, actor, context, aggregate, slice, command, event, fields, flow, trigger, view, automation, translation, subscribes, target, external_system, reads, source, external)
- [ ] Lexical rules exist for: quoted strings (`"..."`), line comments (`#`), identifiers (PascalCase), operators (`->`, `:`), punctuation (`{`, `}`, `[`, `]`, `,`)
- [ ] Grammar rules define the top-level structure: `model "<name>"`, `actor "<name>"`, `context "<name>" { aggregate "<name>" { slice "<name>" { ... } } }`
- [ ] `tree-sitter generate` in `editors/tree-sitter-emod/` produces a working C parser in `src/`
- [ ] Corpus test files cover top-level constructs (model, actor, context, aggregate, slice)

**Affected Files/Modules:**
- `editors/tree-sitter-emod/package.json` — new; npm package manifest for tree-sitter
- `editors/tree-sitter-emod/grammar.js` — new; tree-sitter grammar definition
- `editors/tree-sitter-emod/binding.gyp` — new; node-gyp build config (generated or boilerplate)
- `editors/tree-sitter-emod/` — new project directory

**Patterns to Follow:**
- The complete keyword set is defined in `internal/lexer/token.go:7-25` — match every keyword kind
- The top-level parsing flow (model→actor→context→aggregate→slice hierarchy) follows `internal/parser/parser.go:56-73` which dispatches on keyword type
- The TextMate grammar at `editors/vscode/syntaxes/emod.tmLanguage.json` demonstrates the expected token breakdown for reference

**Language:** JavaScript/TypeScript

**Testable:** Yes — `tree-sitter generate` must succeed, and `tree-sitter test` validates the corpus for implemented constructs.

**Verification:** `tree-sitter generate` exits successfully; `tree-sitter test` reports passed tests for top-level constructs; generated C parser compiles.

**Depends on:** None

---

### Task 2: Implement grammar rules for all slice-content blocks and add corpus tests

**Behavior:** The tree-sitter grammar is completed with rules for every block type that can appear inside a slice: command, event, view, automation, translation, flow, trigger, and fields. Corpus tests exercise every block type. The grammar successfully parses `examples/all_patterns.emod` without errors.

**Acceptance Criteria:**
- [ ] Grammar rules for: `command <Name> { fields { ... } }`, `event <Name> { fields { ... } source external "..." }`, `view <Name> { fields { ... } subscribes [...] }`, `automation <Name> { trigger <Event> command <Cmd> target context <Ctx> }`, `translation <Name> { external_system "..." reads <View> command <Cmd> event <Name> { ... } }`, `trigger <Kind> "Name" { actor <Name> reads <View> }`, `flow { command -> event: <Cmd> -> <Event> }`, `fields { <name> <type> [modifier] }`
- [ ] Field type identifiers and optional modifiers (`required`, `optional`) are parsed
- [ ] `subscribes [...]` lists with comma-separated identifiers are parsed
- [ ] `source external "..."` in events is parsed
- [ ] `tree-sitter test` passes with test cases covering all block types (model, actor, context, aggregate, slice, command, event, view, automation, translation, flow, trigger, fields)
- [ ] The grammar parses `examples/all_patterns.emod` file without tree-sitter parse errors

**Affected Files/Modules:**
- `editors/tree-sitter-emod/grammar.js` — modified; add slice-content block rules
- `editors/tree-sitter-emod/test/` — new; corpus test directory with `.txt` test files for each block type
- `examples/all_patterns.emod` — referenced; test input for integration parsing check (no changes)

**Patterns to Follow:**
- The parser's handling of each block type is defined in `internal/parser/parser.go` — command (lines 325-364), event (lines 366-425), view (lines 427-477), automation (lines 479-569), translation (lines 571-665), trigger (lines 258-323), flow (lines 759-841), fields (lines 701-757)
- Field definitions with type and modifier follow the pattern in `internal/parser/parser.go:731-757`
- The `subscribes` list syntax follows `internal/parser/parser.go:667-699`
- The full set of recognized field type keywords (`string`, `date`, `timestamp`, `int`) is in the TextMate grammar at `editors/vscode/syntaxes/emod.tmLanguage.json:70-73`

**Language:** JavaScript/TypeScript

**Testable:** Yes — `tree-sitter test` verifies all corpus tests pass; `tree-sitter parse examples/all_patterns.emod` verifies the full file parses.

**Verification:** `tree-sitter test` reports all tests passing; `tree-sitter parse examples/all_patterns.emod` reports no errors for the entire file.

**Depends on:** Task 1

---

### Task 3: Create highlight queries for syntax highlighting

**Behavior:** Highlight queries at `editors/tree-sitter-emod/queries/highlights.scm` map tree-sitter node types to standard highlight groups (keyword, string, comment, function, type, operator, punctuation), enabling syntax coloring in tree-sitter-capable editors.

**Acceptance Criteria:**
- [ ] `editors/tree-sitter-emod/queries/highlights.scm` exists with queries for all standard highlight groups
- [ ] Keywords (all DSL keywords) mapped to `@keyword`
- [ ] Quoted strings mapped to `@string`
- [ ] Comments (`# ...`) mapped to `@comment`
- [ ] Entity names after keywords (e.g., command/event/view identifiers) mapped to `@function`
- [ ] Field type identifiers (e.g., `string`, `date`, `timestamp`, `int`) mapped to `@type`
- [ ] Operators (`->`, `:`) and punctuation (`{`, `}`, `[`, `]`, `,`) mapped to `@operator` and `@punctuation` respectively

**Affected Files/Modules:**
- `editors/tree-sitter-emod/queries/highlights.scm` — new; tree-sitter highlight query file

**Patterns to Follow:**
- The TextMate grammar's scope assignments at `editors/vscode/syntaxes/emod.tmLanguage.json` define which tokens get which highlight groups — replicate the same groupings using tree-sitter node names instead of regex captures
- The tree-sitter node names are the names assigned in `grammar.js` via `field()`, `token()`, and grammar rule names

**Language:** JavaScript/TypeScript (SCM query language)

**Testable:** No — highlight queries produce no observable outcome until consumed by an editor integration (US-IDE-013); their correctness is verified structurally by inspection and through downstream editor behavior.

**Verification:** File exists at the expected path; query syntax is valid per tree-sitter's query format; `tree-sitter highlight` (if configured) produces colored output without errors.

**Depends on:** Task 2

---

## Summary

**Total tasks:** 3

**Task ordering rationale:** Dependency-first. Task 1 establishes the project structure and foundational grammar (lexical tokens + top-level block hierarchy). Task 2 completes the grammar with all slice-content blocks and adds the test corpus. Task 3 adds highlight queries that depend on the grammar's node types being defined.

**Acceptance criteria coverage:**
- AC1 (grammar.js exists) → Task 1 + Task 2 (cumulative)
- AC2 (`tree-sitter generate` works) → Task 1
- AC3 (`tree-sitter test` passes covering all block types) → Task 1 (partial: top-level) + Task 2 (all types)
- AC4 (highlight queries exist) → Task 3
- AC5 (parses `examples/all_patterns.emod` without errors) → Task 2

All five story acceptance criteria are covered. No criteria are deferred.

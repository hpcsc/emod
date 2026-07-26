# US-001: Pin files to a DSL version

## Progress
- [x] Task 1: Parse the optional `emod <n>` version header
- [x] Task 2: Reject an unsupported version with a single diagnostic
- [x] Task 3: Insert `emod 1` when formatting
- [x] Task 4: Accept the version header in the tree-sitter grammar
- [x] Task 5: Document the version header in the DSL reference

---

## Story Reference

`user-stories/specs-and-metadata.md` → **US-001: Pin files to a DSL version** (first story of the
"Specs, Invariants, and Model Metadata" feature). Supporting design notes live in
`docs/specs-and-metadata-proposal.md` §1 "Version Header" (lines 37–51) and the Formatter section
(line 391).

**In scope:** the `emod <n>` header, absence-means-1, the unsupported-version diagnostic, and
`emod fmt` inserting the header.

**Explicitly out of scope (resolved for this run):** the `version/missing-header` lint rule (info) is
deferred — nothing is added to `internal/linter`. Later stories in the same feature (descriptions,
glossary, invariants, specs, payloads, wire types, timers) are out of scope. Highlight scopes for the
new keyword in `editors/vscode/syntaxes/emod.tmLanguage.json` and
`editors/tree-sitter-emod/queries/highlights.scm` belong to US-017. Rewriting `examples/*.emod`
belongs to US-018 (and `examples/all_patterns.emod` is not currently `emod fmt --check` clean, so
touching it here would drag in unrelated reformatting).

`tasks/learnings.md` does not exist in this repo, so no prior-run conventions were folded in beyond
the repo's own `CLAUDE.md` and the testing guidelines.

---

## Codebase Context

**Pipeline.** `internal/oracle/oracle.go:14-24` is the canonical lex → parse → validate → lint chain
used by `emod validate`. Two other places open-code the same chain and must inherit any new
behaviour automatically: `internal/lsp/server.go:395-407` (LSP diagnostics) and
`internal/wasm/pipeline.go:39-54` (viewer). `internal/cli/fmt.go:24-38` runs lex → parse → format.
Anything that must hold "once, everywhere" therefore belongs in the lexer or parser, not in a caller.

**Lexer.** `internal/lexer/token.go` declares `Kind` as an iota block: keywords first, then
`Identifier`, `String`, then punctuation. There is currently **no numeric literal token** — a bare
`1` hits the `default:` branch in `internal/lexer/tokenizer.go:77-80` and yields
"unrecognized character". `isIdentifierStart` (tokenizer.go:250-252) excludes digits.
`getKeywordKind` (tokenizer.go:187-248) is the keyword table.

**Ordering constraint.** `internal/parser/parser.go:1208-1215` (`checkIdentifierLike`) treats any
kind ordered **before** `Identifier` as usable in field-name position. That is how `fields { source
string }` already works. A new keyword must therefore be declared inside the keyword block, and a new
literal kind must be declared after `Identifier`.

**Parser.** `internal/parser/parser.go:56-73` is the top-level loop, dispatching through a
`handlers` map keyed by `lexer.Kind` (currently `model`, `actor`, `context`). `p.peek()` /
`p.isAtEnd()` transparently skip comment tokens into `p.pending`
(`internal/parser/parser.go:1174-1189`), so comments may appear anywhere. Diagnostics are appended
via `p.error(msg)` at the current token position (parser.go:1245-1252).

**AST.** `internal/ast/ast.go:14-20` — `ast.Model` holds `Comments`, `Name`, `NamePos`, `Actors`,
`Contexts`. Both `internal/parser` and `internal/formatter` import `internal/ast`, and `formatter`
does **not** import `parser`, so `ast` is the only package both can share a constant through.

**Validator / linter tolerate an empty model.** `internal/validator/validator.go:12-15` returns nil
for a nil model and has no "model name is required" rule; `internal/linter/linter.go:46-49` likewise.
An aborted parse that returns an empty model therefore contributes no extra diagnostics — which is
what makes the "single diagnostic" acceptance criterion achievable.

**Formatter.** `internal/formatter/formatter.go:39-53` (`writeModel`) is the single entry point for
top-level output. Existing tests cover round-trip, idempotency and comment preservation
(`internal/formatter/formatter_test.go:1081-1117`, `:1227-1281`).

**Downstream byte-for-byte fixtures that a formatter change breaks:**
- `internal/cli/fmt_test.go:14-45` (`formattedEmod`) and `:47-74` (`unformattedEmod`)
- `internal/importer/importer_test.go:86,114,133` — compare `formatter.Format(...)` against inline
  source constants
- `e2e-viewer/tests/helpers.js` — `SAMPLE`, `SAMPLE_WITH_VIEW` and the wide sample are documented as
  canonical `emod fmt --check` output; `e2e-viewer/tests/export.spec.js:12,35` assert the viewer
  reproduces them byte for byte (the viewer export path is wasm → `formatter.Format`)

**Header-less fixtures that must keep their exact meaning (the AC-5 regression surface):**
`internal/parser/testdata/{minimal,all_patterns,multi_context,invalid}.emod`,
`internal/test/fixtures.go` (`HotelReservation`), `examples/*.emod`, and the e2e test at
`e2e/tests/validate.test.ts:11` which validates `internal/parser/testdata/all_patterns.emod`.

**Editor grammar.** `editors/tree-sitter-emod/grammar.js:12-28` defines `source_file` as a repeat of
`model_definition | actor_definition | context_definition`. Corpus tests live in
`editors/tree-sitter-emod/test/corpus/` and `task test:grammar` runs `tree-sitter generate` before
`tree-sitter test`.

**Test conventions in this repo** (`CLAUDE.md` "Go Test Organization"): one umbrella `Test{TypeName}`
per type, `t.Run` groups by operation, leaf subtest names that read as sentences about the observed
outcome, `testify/require`, fresh fixtures per leaf, `//go:build unit` or `//go:build integration`
tags. AST comparisons use `test.RequireEqual` with `cmpopts.IgnoreTypes(ast.Position{})`
(`internal/parser/integration_test.go:18`).

---

## Tasks

### Task 1: Parse the optional `emod <n>` version header

**Behavior:** A `.emod` file may open with `emod <integer>` on its own line before the `model`
declaration. The parsed model records the declared version; a file without the header parses exactly
as it does today and reports version 1.

**Acceptance Criteria:**
- [x] A file opening with `emod 1` followed by `model "..."` parses with zero diagnostics, and
      produces the same AST as the identical file without the header apart from the recorded version
- [x] A file with no header parses with zero diagnostics and reports version 1
- [x] The AST distinguishes "declared version 1" from "no header at all", so the formatter and future
      rules can tell the two apart
- [x] A malformed header (`emod` with nothing after it, `emod x`, `emod "1"`) produces a parse error
      on line 1 that names the version header, with filename/line/column populated
- [x] `emod <n>` appearing after the `model` declaration is a parse error identifying a misplaced
      version header rather than an unrecognized keyword
- [x] Comments preceding the header do not stop it being recognized, and stay attached to the model
      exactly as they do today
- [x] A `fields` block containing a field named `emod` still parses as an ordinary field (guards the
      `checkIdentifierLike` contract)
- [x] An integer literal anywhere other than the header position still produces a diagnostic rather
      than being silently consumed
- [x] Every fixture under `internal/parser/testdata/` and `internal/test.HotelReservation` parses to
      the same AST and the same diagnostics as before this task

**Affected Files/Modules:**
- `internal/lexer/token.go` — new keyword kind and new integer-literal kind, plus their `String()` cases
- `internal/lexer/tokenizer.go` — keyword table entry and digit handling in the scan dispatch
- `internal/lexer/tokenizer_test.go` — coverage for the new token kinds
- `internal/ast/ast.go` — version fields on `ast.Model`
- `internal/parser/parser.go` — header recognition ahead of the top-level loop
- `internal/parser/parser_test.go` — header parsing, malformed header, misplaced header, backward compatibility

**Patterns to Follow:**
- Kind declaration order and the `checkIdentifierLike` contract: `internal/parser/parser.go:1206-1215`
  read together with the iota block in `internal/lexer/token.go:5-55`
- Every `Kind` needs a case in `internal/lexer/token.go:64-153`
- Keyword table and scan dispatch: `internal/lexer/tokenizer.go:187-248` and `:28-80`
- Top-level dispatch, `expectedKeywords()` and error reporting: `internal/parser/parser.go:56-93`
- Position and diagnostic helpers: `internal/parser/parser.go:1158-1163` and `:1245-1252`
- The existing "unrecognized keyword lists the alternatives" test must keep passing:
  `internal/parser/parser_test.go:1307-1321`
- Test layout: `internal/lexer/tokenizer_test.go:13-58` and `internal/parser/parser_test.go:20-60`;
  `CLAUDE.md` "Go Test Organization"
- Assert on the exported contract — the tokens from `lexer.Scan` and the `(*ast.Model, diagnostics)`
  pair from `parser.Instance.Parse` — not on parser internals. Caller pattern: **Exported API**
  (`~/.config/ai/guidelines/testing/caller-patterns.md`); unit of behavior per
  `~/.config/ai/guidelines/go/testing-patterns.md`
- AST comparison helper: `internal/parser/integration_test.go:18` (`test.RequireEqual` with
  `cmpopts.IgnoreTypes(ast.Position{})`)

**Testable:** Yes — `lexer.Scan` and `parser.Instance.Parse` are exported and already have test suites.

**Verification:** `task test:unit` and `task test:integration` pass.

**Depends on:** None

---

### Task 2: Reject an unsupported version with a single diagnostic

**Behavior:** A file declaring a version the tool does not support is rejected with exactly one
"unsupported version" diagnostic naming both the declared and the supported version, and nothing
downstream in the file is reported.

**Acceptance Criteria:**
- [ ] The supported version is a single exported constant (value `1`) with one definition site, so a
      future bump is a one-line change
- [ ] An otherwise-valid file declaring a version above the supported one produces exactly one
      diagnostic, positioned on line 1, whose message names both the declared version and the
      supported version
- [ ] The same file with deliberately broken grammar after the header still produces exactly that one
      diagnostic — no parse errors from later in the file
- [ ] `oracle.Check` on such a file returns exactly one entry: the validator and linter contribute
      nothing
- [ ] `emod validate` on such a file exits non-zero and reports only that diagnostic, in both `text`
      and `json` output
- [ ] `emod fmt` on such a file fails with that diagnostic and leaves the file byte-identical
- [ ] A version below 1 (`emod 0`) is rejected with the same diagnostic shape rather than silently accepted
- [ ] A file declaring a supported version, and a header-less file, are unaffected — same diagnostics
      as after Task 1
- [ ] The diagnostic is an error with no rule name — it is not a lint rule, and `internal/linter` is
      untouched

**Affected Files/Modules:**
- `internal/ast/ast.go` — home for the supported-version constant, the only package both `parser` and
  `formatter` already import
- `internal/parser/parser.go` — the version gate that stops the parse
- `internal/parser/parser_test.go` — unsupported version, unsupported version plus broken grammar, boundary values
- `internal/oracle/oracle_test.go` — the single-diagnostic property across the whole chain
- `internal/cli/validate_test.go` — text and json output for an unsupported version
- `internal/cli/fmt_test.go` — fmt refuses and does not rewrite
- `internal/test/fixtures.go` — a versioned fixture may be added here; `HotelReservation` must stay
  header-less as the backward-compatibility guard

**Patterns to Follow:**
- The gate must live in the parser: `internal/oracle/oracle.go:14-24`, `internal/lsp/server.go:395-407`
  and `internal/wasm/pipeline.go:39-54` each rebuild the same chain, so a check placed in any one
  caller would not hold for the others
- Diagnostic shape, severity and `String()` formatting: `internal/diagnostic/entry.go:24-38`
- Validator and linter already no-op on an empty model: `internal/validator/validator.go:12-15`,
  `internal/linter/linter.go:46-49`
- CLI surfaces and their error types: `internal/cli/validate.go:14-49`, `internal/cli/fmt.go:14-52`,
  and the `LintError` exit-code convention
- CLI test layout and the shared `writeTemp` helper: `internal/cli/validate_test.go:17-40`,
  `internal/cli/fmt_test.go:80-164`
- For the CLI tests the caller is the model author submitting a file, so assert on
  acceptance/rejection, exit code and the message they read — not on internal routing. Caller
  pattern: **Inbound** (`~/.config/ai/guidelines/testing/caller-patterns.md`)
- `internal/oracle/oracle_test.go` for the existing whole-chain test style

**Testable:** Yes — through `parser.Instance.Parse`, `oracle.Check`, `cli.RunValidate` and `cli.RunFmt`.

**Verification:** `task test:unit` and `task test:integration` pass; the unsupported-version case
returns exactly one diagnostic from `oracle.Check`.

**Depends on:** Task 1

---

### Task 3: Insert `emod 1` when formatting

**Behavior:** `emod fmt` writes the version header as the first line of every file it formats,
inserting `emod 1` when the header is absent and preserving an explicitly declared version, so
formatted files are always pinned.

**Acceptance Criteria:**
- [ ] `formatter.Format` emits the version header as the first line, directly followed by the model's
      leading comments and the `model` declaration, with no blank line between the header and what
      follows it
- [ ] A model with no declared version formats with the supported-version constant introduced in
      Task 2, so `emod fmt` can never pin a file to a version the parser would reject
- [ ] A model that declared a version formats with that same version
- [ ] Formatting is idempotent, including for input that already carries the header:
      `format(format(x)) == format(x)`
- [ ] parse → format → parse yields an equivalent AST, and the reparsed model reports the same version
- [ ] Formatting a header-less file adds only the header line: every other line of canonical output is
      unchanged from before this task, so existing files keep their exact meaning
- [ ] `emod fmt --check` reports a header-less but otherwise canonical file as unformatted, and
      reports a header-carrying canonical file as formatted
- [ ] The viewer's emod export still matches its canonical fixtures byte for byte, with those fixtures
      updated to carry the header

**Affected Files/Modules:**
- `internal/formatter/formatter.go` — header emission in `writeModel`
- `internal/formatter/formatter_test.go` — header emission, idempotency and round-trip; existing
  expected-output constants gain the header
- `internal/cli/fmt_test.go` — `formattedEmod` / `unformattedEmod` constants and the insert-on-format case
- `internal/importer/importer_test.go` — the three byte-for-byte comparisons against inline sources
- `e2e-viewer/tests/helpers.js` — `SAMPLE`, `SAMPLE_WITH_VIEW` and the wide sample constants

**Patterns to Follow:**
- `writeModel` is the single top-level writer: `internal/formatter/formatter.go:39-53`
- Extend the existing round-trip, idempotency and comment-preservation tests rather than adding
  parallel ones: `internal/formatter/formatter_test.go:1081-1117` and `:1227-1281`
- Byte-for-byte comparisons that will fail until updated:
  `internal/importer/importer_test.go:86,114,133`
- The viewer fixtures document themselves as canonical `emod fmt --check` output and are asserted
  byte for byte: `e2e-viewer/tests/helpers.js:3-5` with `e2e-viewer/tests/export.spec.js:12,35`
- `formatter.Format` is also reached from `internal/lsp/server.go:367` (format-on-save) and
  `internal/wasm/pipeline.go:84` (viewer export); neither needs a change, but their tests may carry
  expected text
- Caller patterns: **Exported API** for `formatter.Format` (assert on the emitted text), **Inbound**
  for `cli.RunFmt` (assert on the file's contents and on whether it was rewritten)

**Testable:** Yes — through `formatter.Format` and `cli.RunFmt`.

**Verification:** `task test:unit`, `task test:integration` and `task test:e2e:viewer` pass.

**Depends on:** Task 1, Task 2

---

### Task 4: Accept the version header in the tree-sitter grammar

**Behavior:** The tree-sitter grammar parses a file that opens with `emod 1` without error nodes, so
editors that rely on it keep working now that `emod fmt` pins every file it touches.

**Acceptance Criteria:**
- [x] A file opening with `emod 1` before `model` parses into a distinct version-header node under
      `source_file`, with no `ERROR` or `MISSING` nodes
- [x] Header-less files parse exactly as before and every existing corpus test passes unchanged
- [x] The header is only accepted before the first declaration
- [x] A corpus test covers both the header and the header-less form
- [x] `tree-sitter generate && tree-sitter test` passes — 29/29 under the pinned `tree-sitter-cli`
      0.26.9. The regenerated `src/` output stays gitignored: this criterion originally called for
      committing it, which would reverse the repo's standing decision to ignore generated artifacts
      and make every grammar change carry a ~3k-line diff sensitive to the generating CLI version.
      `task test:grammar` regenerates before running, so nothing depends on it being in the tree.

**Affected Files/Modules:**
- `editors/tree-sitter-emod/grammar.js` — `source_file` and a version-header rule
- `editors/tree-sitter-emod/test/corpus/model.txt` — or a new corpus file for the header
- `editors/tree-sitter-emod/src/grammar.json`, `src/node-types.json`, `src/parser.c` — regenerated output

**Patterns to Follow:**
- `source_file` and the top-level choices: `editors/tree-sitter-emod/grammar.js:12-28`
- Corpus test format (`===` header, source, `---`, expected s-expression):
  `editors/tree-sitter-emod/test/corpus/model.txt`
- `task test:grammar` regenerates before running: `Taskfile.yml` → `test:grammar`
- Highlight scopes for the new keyword — `editors/tree-sitter-emod/queries/highlights.scm` and
  `editors/vscode/syntaxes/emod.tmLanguage.json` — belong to US-017; do not add them here

**Testable:** Yes — the corpus tests are the grammar's public contract.

**Verification:** `task test:grammar` passes.

**Depends on:** Task 1

---

### Task 5: Document the version header in the DSL reference

**Behavior:** The DSL reference explains the version header, so a model author can learn it without
reading the grammar or the proposal.

**Acceptance Criteria:**
- [x] `docs/dsl-reference.md` gains a version-header section with a syntax block and a worked example
      showing `emod 1` above `model "..."`
- [x] The section states that absence means version 1, that `emod fmt` inserts the header, and what the
      unsupported-version diagnostic reports (declared version and supported version)
- [x] The Kubernetes-style rule is stated: additive grammar changes (new optional keywords) do not bump
      the version; breaking changes do
- [x] Surrounding section numbering and cross-references stay coherent
- [x] No lint rule is documented — `version/missing-header` remains deferred

**Affected Files/Modules:**
- `docs/dsl-reference.md` — new section alongside "General Syntax" / "Top-Level Constructs"

**Patterns to Follow:**
- Section style, numbered headings and fenced `emod` examples: `docs/dsl-reference.md:11-67`
- Source wording to draw on: `docs/specs-and-metadata-proposal.md:37-51`
- Em-dash and prose conventions: `~/.config/ai/guidelines/writing/em-dash.md`

**Testable:** No — documentation only; no exported behaviour to assert on.

**Verification:** The example in the new section, saved to a scratch file, passes `emod validate` and
`emod fmt --check` unchanged.

**Depends on:** Task 2, Task 3

---

## Summary

**Total tasks:** 5

**Ordering rationale:** Dependency-first, then risk. Task 1 opens the grammar (lexer → AST → parser)
because everything else needs the header to exist and be recorded. Task 2 is the story's actual
payoff and its riskiest property — "exactly one diagnostic, not a parse error deep in the file" — and
it is separated from Task 1 because it is not simple error handling on the same behaviour: it
introduces the supported-version constant, aborts the parse, and has to hold across `oracle.Check`,
the LSP and the wasm pipeline. Task 3 is deliberately last of the Go tasks because it is the widest
blast radius: it changes the canonical bytes of every formatted file and therefore touches
`internal/cli`, `internal/importer` and the viewer's e2e fixtures in the same commit — anything less
would leave CI red. Tasks 4 and 5 are the tooling and documentation tail; Task 4 depends only on
Task 1 conceptually and can run in parallel with Tasks 2–3.

**Acceptance criteria coverage:**

| Story criterion | Covered by |
|---|---|
| `emod 1` before `model`, parses and validates as before | Task 1 |
| No header is treated as version 1, no diagnostics | Task 1 |
| Version above the supported one → one "unsupported version" diagnostic naming both versions | Task 2 |
| `emod fmt` inserts `emod 1` when absent | Task 3 |
| Every existing header-less `.emod` file keeps its exact meaning | Task 1 (parse identity for all existing fixtures) and Task 3 (formatting adds only the header line) |

**Deferred, by explicit decision:**
- The `version/missing-header` lint rule (info) — resolved as deferred for this run; `emod fmt`
  drives adoption.
- Syntax highlighting of the `emod` keyword in the VS Code TextMate grammar and the tree-sitter
  highlight queries — US-017. Task 4 covers only *parsing* the header, so editors do not show an
  error on every formatted file.
- Rewriting `examples/*.emod` to carry the header — US-018 owns the examples, and
  `examples/all_patterns.emod` is not currently `emod fmt --check` clean, so pinning it here would
  pull in unrelated reformatting.
- Carrying the version through `emod export -f json` / `-f cue` and `internal/cue/schema.cue` — not
  in this story's criteria.
- `internal/parser/testdata/*.emod` and `internal/test.HotelReservation` stay header-less on purpose:
  they are the regression fixtures for "every existing header-less file keeps its exact meaning".

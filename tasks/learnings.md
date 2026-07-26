# Learnings

Durable facts distilled from completed implementation runs. Read before starting work
in this repo; append only learnings that generalise beyond the task that surfaced them.

## CLI diagnostic tests must assert the distinguishing message text
- Type: recurring-finding
- Learning: `internal/cli/*_test.go` tests that assert only `err.Error()` contains the path, `":1:"`, a line number, or a count are vacuous — any unrelated line-1 diagnostic satisfies them. The established repo convention is to assert the tokens that identify *this* diagnostic (see `internal/cli/validate_test.go:253-258` asserting rule names plus symbol names, and `internal/parser/parser_test.go:53-80` asserting the message names both the declared and the supported version). New CLI leaves for a new error kind must assert the same distinguishing content as the parser-level test one layer down, not just position and count.
- Apply when: adding or reviewing a test in `internal/cli` (validate, fmt, export, diagram, schema) that exercises a new error or diagnostic path.

## Run repo tooling through `mise exec --`, not bare PATH
- Type: constraint
- Learning: `/Users/davidnguyen/Personal/Code/emod/mise.toml` pins `cargo:tree-sitter-cli = "0.26.9"`, but the user's global `~/.config/mise/config.toml` pins a separate `tree-sitter = "0.25.10"` that wins on PATH in a non-activated shell. The two CLIs emit different generated output (0.26.x drops the version suffix from the `parser.c` banner and rewrites `src/tree_sitter/array.h`; `grammar.json` and `node-types.json` are identical), so a bare `task test:grammar` can leave the tree looking clean while `mise exec -- task test:grammar` produces a ~111/72-line diff. CI (`jdx/mise-action@v4`) sees only the repo pin, so the repo-pinned resolution is the one that matters. CI has no dirty-tree gate, so this drift stays green and is only caught by checking `git diff --exit-code` yourself.
- Apply when: running any Taskfile target whose output is committed (notably `task test:grammar`), or verifying that generated files in a diff are reproducible.

## Generated tree-sitter `src/` stays gitignored
- Type: constraint
- Learning: `editors/tree-sitter-emod/.gitignore` ignores `src/` alongside `bindings/node/` and `node_modules/` — the repo does not track generated artefacts. A grammar change commits `grammar.js` and its corpus tests only. Do not un-ignore `src/` to satisfy a "regenerated output is committed" criterion: it would make every grammar change carry a ~3k-line diff whose bytes depend on which CLI generated it, and `task test:grammar` runs `tree-sitter generate` first so nothing depends on those files being tracked.
- Apply when: touching `editors/tree-sitter-emod/grammar.js`, or reviewing a diff that changes the grammar.

## A task criterion requiring "committed" output cannot close
- Type: constraint
- Learning: implement-flow commits a task only *after* it closes, so an acceptance criterion phrased as "... and the output is committed" is unsatisfiable from inside the task — the audit re-checks `git log`/`git ls-tree`, finds nothing, and reports the criterion unmet on every attempt until `maxResolve` is exhausted. This burned all 5 attempts on US-001 Task 4. Phrase the criterion against the working tree instead — "regenerating leaves the tracked files byte-identical", "the file exists and matches X". Note that "`git status` is clean" is *also* unusable: a task under review always has uncommitted work.
- Apply when: writing or reviewing acceptance criteria during decomposition, especially for tasks that produce generated or checked-in artefacts.

## New DSL keywords must stay usable as field names
- Type: pattern
- Learning: emod keywords are not reserved words — they remain valid identifiers inside `fields` blocks. The Go parser handles this with `Instance.checkIdentifierLike()` (`internal/parser/parser.go`), and the tree-sitter grammar mirrors it with a permissive `any_identifier: $ => /[a-zA-Z_][a-zA-Z0-9_]*/` plus a keyword token narrow enough to match only in its own position (e.g. `version_header` bakes the separator into the token via `token(seq('emod', /[ \t]+/))` with `token.immediate` digits, so a bare `emod` cannot pair with a number on the next line). Adding a keyword to `internal/lexer/token.go` therefore needs a corresponding "field named X" case on both sides — see `internal/parser/parser_test.go` "emod is usable as a field name" and `editors/tree-sitter-emod/test/corpus/version_header.txt::Field named emod inside a fields block`.
- Apply when: introducing a new keyword into the emod language, in the Go lexer/parser or the tree-sitter grammar.

## Formatter output always begins with `emod N`
- Type: convention
- Learning: `formatter.Format` unconditionally emits a version header as the first line (`emod %d` from `pinnedVersion`, preserving `Model.Version` when `VersionDeclared` is set, otherwise `ast.SupportedVersion`). Every expected string in `internal/formatter/formatter_test.go` and every canonical-output constant in `internal/cli/fmt_test.go` must start with that line, while *input* fixtures may omit the header (the parser implies version 1 and leaves `VersionDeclared` false). `fmt --check` fails a file whose only deviation is the missing header.
- Apply when: adding a formatter golden, a `RunFmt` fixture, or an `.emod` round-trip test.

## `docs/dsl-reference.md` anchors embed the section number
- Type: convention
- Learning: every heading in `docs/dsl-reference.md` is `## <n>. <Title>`, so its in-document links are number-prefixed slugs (`[Slice Patterns](#6-slice-patterns)`, `[Flows](#7-flows)`, `[Cross-References](#10-cross-references)`). Inserting, removing or reordering a section renumbers every heading below it *and* invalidates each link that cites one of those numbers — four such links exist today. Nothing catches the drift: neither `Taskfile.yml` nor `.github/workflows/ci.yml` runs a markdown link check, and a stale `#7-cross-references` link (pointing at "7. Fields") survived an earlier insertion unnoticed. After editing the section list, re-derive the anchors by listing `^## [0-9]+\.` and `\(#[0-9]+-` and reconciling the two.
- Apply when: adding, removing or reordering a numbered section in `docs/dsl-reference.md` (or any doc using `## <n>. Title` headings).

## Keyword surfaces fan out past the lexer, parser and tree-sitter grammar
- Type: pattern
- Learning: a new emod keyword touches several hand-maintained lists that no test cross-checks against `internal/lexer/token.go`: the TextMate alternation at `editors/vscode/syntaxes/emod.tmLanguage.json:63`, `keywordDescriptions` in `internal/lsp/hover.go:13`, and the per-context lists in `internal/lsp/completer.go`. `isKeyword` (`internal/lsp/hover.go:37`) is an *ordinal range* — `k >= lexer.KeywordModel && k <= lexer.KeywordExternal` — so every `Kind` appended after `KeywordExternal` (`mode`, `tags`, `decides_on`, `where`, `and`, `or`, `not`, `tag`, `events`, `emod`) is silently invisible to hover no matter what the description map contains. Decide per keyword whether each surface should cover it; do not assume adding the token propagates.
- Apply when: adding a keyword to `internal/lexer/token.go`, or auditing why a valid keyword has no VS Code highlighting, hover text or completion.

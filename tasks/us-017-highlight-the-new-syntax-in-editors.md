# US-017: Highlight the new syntax in editors

## Progress
- [ ] Task 1: Assert the field-name and keyword captures for the spec and metadata keywords in tree-sitter
- [ ] Task 2: Paint a keyword-named field as a field name in the VS Code grammar
- [ ] Task 3: Capture payload numbers and booleans as literals in the tree-sitter highlight query
- [ ] Task 4: Scope payload numbers and booleans as literals in the VS Code grammar

---

## Story Reference

`user-stories/specs-and-metadata.md` → **US-017: Highlight the new syntax in editors** (seventeenth
story of "Specs, Invariants, and Model Metadata", lines 216-224). Design notes:
`docs/proposals/specs-and-metadata-proposal.md` — the keyword list at `:390`, the `Number` token and
the positional recognition of `true` / `false` at `:392`, the field-name-position fix at `:394`, and
Phase 5 at `:607`, which names "Tree-sitter grammar and VS Code highlighting for new keywords" as this
story's whole remit.

**What the story asks for is already two-thirds delivered, and by other stories.** Eight of the ten
keywords the first criterion names — `spec`, `given`, `when`, `then`, `rejected`, `invariant`,
`description`, `emod` — are in `internal/lexer/token.go`'s `keywords` map on main (`:73-111`, thirty-seven
spellings), and therefore in all three editor surfaces already, because `TestEditorKeywordCoverage`
(`editors/tree-sitter-emod/test/queries/keywords_test.go:47-79`) ranges over `lexer.Keywords()` and
fails when `editors/tree-sitter-emod/grammar.js`,
`editors/tree-sitter-emod/queries/highlights.scm` or
`editors/vscode/syntaxes/emod.tmLanguage.json` does not spell one. The remaining two are owned
elsewhere and for the same reason: `type` belongs to US-012, whose Task 1 is titled "Spell `type` on
the hand-maintained editor keyword surfaces", and `after` belongs to US-013, whose Task 1 is titled
"Accept and colour `after` across the three editor grammars". Both land their spelling on all three
surfaces in the change that teaches the lexer, because they cannot go green otherwise.

**The drift test proves presence, not scoping — so both were measured separately, and they give
different answers.** `TestEditorKeywordCoverage` extracts every double-quoted token from
`highlights.scm` and every word from a `match`/`begin`/`end` pattern in the TextMate grammar, so it
cannot tell a `@keyword` list entry from a word inside a comment, and it says nothing at all about what
an editor paints at a given column. That gap is real, but it is narrower than it looks. Measured
against the shipped grammars, keyword *position* is correctly scoped on both surfaces for all eight
spellings:

- **tree-sitter**, via `mise exec -- tree-sitter query queries/highlights.scm` over a model carrying a
  version header, a `description`, an `invariant`, a full three-clause `spec` and a `then rejected`:
  every one of the eight receives `@keyword`, and a field named after one receives `@variable.member`
  on its name and `@type` on its type.
- **TextMate**, via `vscode-tmgrammar-snap` over the same shapes: `emod`, `description`, `invariant`,
  `spec`, `given`, `when`, `then` and `rejected` each receive `keyword.control.emod` in their own
  position, the version header's `emod` included.

`mise exec -- task test:grammar` is green on main: 70 corpus parses, the three highlight files
reporting 48 assertions between them (`constructs.emod` 29, `dcb.emod` 11,
`unreserved-keywords.emod` 8), and the `//go:build grammar` query tests passing. So the first criterion
needs no grammar change on either surface. Three things do fall through, and they are what the tasks
below cover:

1. **The scoping failure is confined to field-name position, on TextMate only** — the story's third
   criterion, failing outright. Measured against the shipped grammar: `description string required`
   inside a `fields` block yields `keyword.control.emod` on `description`; so does `spec int`; so does
   `given RoomID`. `editors/vscode/test/scopes/fields.emod:10-17` already records this for
   `command string required` and says in a comment that the assertion states "what the grammar produces
   for such a line today ... not what it ought to produce". tree-sitter has no such failure: its
   field-name capture is structural.
2. **Neither surface has any notion of a numeric or boolean literal** — the story's second criterion,
   and the one the drift test could never reach, because `true` and `false` are not keywords at all.
   `highlights.scm` captures `(string)` and nothing else that is a value; `emod.tmLanguage.json` has
   `strings` (`:18-21`) and no `constant.*` rule. Measured: in
   `given [RoomReserved { roomId: "101", nights: 3, rate: 12.50, vip: true }]`, `"101"` takes
   `string.quoted.double.emod` and `3`, `12.50` and `true` take no scope beyond `source.emod`.
3. **The tree-sitter highlight suite asserts none of it.** Its 48 assertions cover the structural
   constructs, DCB, and `on` / `every` as unreserved words. No marker in the repository names `spec`,
   `given`, `when`, `then`, `rejected`, `invariant` or the version header's `emod` — in either
   position. The captures are right; nothing says so, and nothing would notice if they stopped being
   right. This is carried rather than stated by the story, for the reason
   `tasks/completed/us-010-highlight-the-realigned-syntax.md:32-39` gives: the criteria are otherwise
   unfalsifiable.

**In scope:** the two editor surfaces named in the criteria —
`editors/tree-sitter-emod/queries/highlights.scm` with its assertion files under
`editors/tree-sitter-emod/test/highlight/`, and `editors/vscode/syntaxes/emod.tmLanguage.json` with its
assertion files under `editors/vscode/test/scopes/`. Both already have executable suites wired into
`task test` and CI, so every task here is testable and none needs new infrastructure.

**Out of scope, owned elsewhere.** LSP hover, completion, navigation and semantic tokens over the new
constructs (US-015) — a different surface from highlighting, reached by `internal/lsp/hover.go`,
`completer.go` and `keywords_test.go`, none of which this story opens. Adding `type` or `after` to
`internal/lexer/token.go` and to the three keyword surfaces (US-012 Task 1 and US-013 Task 1, each of
which also owns its own keyword-position and field-name-position assertions for its spelling). The
payload grammar itself — the lexer's number form, the parser, the validator, the formatter, the
exports and `editors/tree-sitter-emod/grammar.js` (US-010, whose own scope note at
`tasks/us-010-state-example-payloads-in-specs.md:42-43` defers "highlighting numbers and `true` /
`false` as literals in the VS Code grammar and the tree-sitter highlight queries" to this story).
`examples/*.emod` and `docs/dsl-reference.md` (US-018). Rendering specs on diagrams (US-016). Every Go
package, `internal/viewer`, `e2e/` and `e2e-viewer/`.

**Sequencing.** This story ships after US-010 and is independent of US-012 and US-013.

- *US-010 gates half of it.* Tasks 3 and 4 cannot be verified before US-010 Task 7 lands the payload
  rule in `editors/tree-sitter-emod/grammar.js`: with no payload node there is no node type for a
  capture to name, and a highlight fixture containing a payload does not parse. Tasks 1 and 2 need
  nothing from US-010 and can land before it.
- *US-012 and US-013 gate nothing here, because the drift test forces them to carry their own keyword.*
  `TestEditorKeywordCoverage` goes red the moment `type` or `after` joins the lexer's map unless all
  three surfaces already spell it, which is why both stories put their editor task first in their own
  breakdown. Waiting for them would mean this story restating criteria they already own.

**Open questions, decided.** Nine shapes the story does not spell out:

1. *The field-name repair is general, not per-keyword.* The third criterion names "a keyword", and the
   only honest TextMate fix is a rule that claims a field line's name inside the `fields` block
   regardless of its spelling — the same shape the tree-sitter side already has, where
   `highlights.scm:89-90` captures `field_line`'s first `any_identifier` as `@variable.member` no matter
   what it says. Repairing the eight the story enumerates and leaving the other twenty-nine painted as
   keywords would be a per-keyword allowlist inside a rule that has no per-keyword axis. So Task 2
   retires the recorded gap for every keyword at once, including the `command string required` line
   `fields.emod:14-17` pins today. `tasks/completed/us-003-use-reserved-words-as-field-names.md:57-61`
   declined this repair and `tasks/completed/us-010-highlight-the-realigned-syntax.md:85-92` declined it
   again, both because the file had no block context to hang it on. It has one now: `fields-block`
   (`emod.tmLanguage.json:22-38`) is a `begin`/`end` rule, added by that same story, and it is the reason
   the repair is a rule rather than a restructuring.
2. *`on` and `every` are already correct and stay untouched.* They are the two keywords the flat
   alternation deliberately omits (`emod.tmLanguage.json:101` says so), coloured instead by
   `positional-keywords` (`:67-89`) keyed on their operand. A general field-name rule must leave their
   existing assertions in `editors/vscode/test/scopes/unreserved-keywords.emod` passing unedited — those
   assertions are stated as negative scope lists, so a name that gains a positive field-name scope
   still satisfies them.
3. *The single-line `fields { <name> <type> <modifier> }` form is recorded, not required.* It is legal
   input — the parser is line-oriented only in gating a field's optional trailing tokens — but it is not
   canonical: `formatter.writeFields` (`internal/formatter/formatter.go:285-311`) writes `fields {` and
   then one field per line, and no `.emod` file tracked in the repository writes the one-line form.
   TextMate has no anchor for "the first token after the block's opening brace on the brace's own line"
   short of folding it into the `begin` pattern's captures. Task 2 therefore requires the form to be
   *covered by an assertion stating what it actually produces*, following the precedent
   `fields.emod:10-13` set, rather than requiring it to match the multi-line treatment.
4. *`true` and `false` must be position-keyed on the TextMate side, exactly as `on` and `every` are.*
   They are not keywords — US-010's decision 3 (`tasks/us-010-state-example-payloads-in-specs.md:66-73`)
   keeps them out of the lexer's map deliberately — and US-010 Task 7 adds a corpus case for a `fields`
   block declaring fields named `true` and `false`. A word alternation as positionless as the one
   serving the other thirty-five spellings would paint those field names, which is the failure
   `tasks/learnings.md` records under "A keyword that is only a
   keyword in one position never joins the flat TextMate alternation". The tree-sitter side has no such
   hazard: a field line's `true` parses as `any_identifier`, not as a literal node.
5. *A number rule may be positionless where a boolean rule may not.* A bare number can appear in only
   two places in the language — the version header's digits and a payload value — and no identifier can
   begin with one, so a digit run is unambiguous in a way `true` is not. Whether the version header's
   `1` gains a numeric scope alongside the payload's `3` is left to Task 4, which must state the choice
   in an assertion either way. The one hazard is ordering: `every "0 9 * * *"` and a wire type like
   `"com.acme.v2"` carry digits inside strings, and `standalone-tokens` (`emod.tmLanguage.json:85-94`)
   lists `#strings` first precisely so a later rule never reaches inside one.
6. *Payload field names get no capture and no scope.* The criterion names numbers and booleans. A
   payload's `roomId:` is the same shape as a DCB `tags` key (`entity: customerId`,
   `editors/tree-sitter-emod/test/highlight/dcb.emod:29`), which carries no capture today either, and
   giving one of them a treatment the other lacks would be a new inconsistency rather than a repair.
7. *An invariant's name and a spec element's reference stay uncaptured.* Verified against the shipped
   query: in `invariant OneCopyPerLoan "..."` the identifier receives no capture, and neither does
   `PlaceOrder` after `when` nor `OrderPlaced` inside a `given` list. Painting them as entity names
   would be a fourth criterion the story does not state, and it is a reference rather than a
   declaration — the two identifier-name sections of `highlights.scm` (`:67-81`) capture declaration
   sites only.
8. *The version header's keyword capture includes its separator.* Measured against the shipped query
   over `emod 1`: the `@keyword` capture spans `(0,0)-(0,5)` with text `emod ` — one character wider
   than the word — because `grammar.js:27-28` bakes the separator into the token and aliases the result
   back to the bare spelling, so a bare `emod` cannot pair with a number on the next line. An assertion
   marker must sit inside that span, and a TextMate assertion underlining the word must not assume the
   two surfaces agree on the extent.
9. *`test/highlight/constructs.emod` is load-bearing for a second suite and must not grow a duplicate
   block header.* `sample_test.go:12` names it as `samplePath`, and `headerRow` (`:38-49`) requires each
   header in `sampleBlocks` (`queries_test.go:25-30` — `trigger "Review orders"`,
   `automation NotifyCustomer`, `automation ArchiveOrders`, `fields {`) to head **exactly one** line of
   that file, failing with "heads more than one line" otherwise. A second `fields {` added to it breaks
   `TestEditorQueries`, which is unrelated to this story. New shapes belong in a sibling file.

**Learnings folded in** from `tasks/learnings.md`: every editor-highlight surface now has an executable
suite and each is run a different way — `test/highlight/*.emod` is picked up automatically by
`tree-sitter test` and needs no Taskfile or CI change, while `editors/vscode/test/scopes/*.emod` runs
through `task test:vscode`, which is already both in the `test` aggregate and in
`.github/workflows/ci.yml:35`; a tree-sitter highlight marker only discriminates while another
highlighted token follows it on the same line, which is why the existing fixtures carry deliberate
trailing comments; `highlights.scm` field patterns select by anchor, never by `#match?` on the token's
text, and each anchored step needs a `(comment)*` before it; a keyword that is only a keyword in one
position never joins the flat TextMate alternation; a suite that pins another tool's output owes a
mutated-input negative control, and the mismatch helper must assert the position, the required scope and
the scope actually produced; a test that shells out to a CLI runs with `-count=1`; keyword surfaces fan
out past the lexer and each surface is a per-keyword decision; new DSL keywords must stay usable as
field names; the tree-sitter grammar must never be stricter than the Go parser; generated tree-sitter
`src/` stays gitignored; run repo tooling through `mise exec --`, not bare PATH; an assertion whose
expected value comes from the code under test is the recurring review finding; acceptance criteria
describe the working tree, never the repository's history, and a commit-message receipt is the commit
author's obligation, never a criterion.

---

## Codebase Context

**The tree-sitter highlight query.** `editors/tree-sitter-emod/queries/highlights.scm` is 108 lines in
seven commented sections: comments (`:12`), a generic string capture (`:16`), a bracketed list of
thirty-seven anonymous keyword tokens (`:18-63`) whose header comment names `TestEditorKeywordCoverage`
as its guard and states that nothing derives the list from the grammar, quoted entity names for
`model` / `actor` / `context` / `aggregate` / `slice` / `trigger` (`:67-74`), identifier entity names for
`command` / `event` / `view` / `automation` / `translation` (`:76-81`), the three anchored `field_line`
patterns for name, type and modifier (`:83-102`), operators (`:104-105`) and punctuation (`:107-108`).
The keyword list already carries all eight of this story's landed spellings. There is no `@number`,
`@boolean` or `@constant` capture anywhere in the file.

**What that query actually produces today**, measured with the repo-pinned CLI over a model carrying a
version header, a `description`, an `invariant`, a `spec` with all three clauses, and a `fields` block
declaring fields named `description` and `spec`: every one of the eight keywords receives `@keyword`
in its own position, and a field named after one receives `@variable.member` on its name and `@type` on
its type. The tree-sitter half of criteria one and three is therefore already *correct*; what is missing
is any assertion that says so.

**The tree-sitter highlight suite.** `editors/tree-sitter-emod/test/highlight/` holds three assertion
files — `constructs.emod` (83 lines, the structural constructs and four field lines), `dcb.emod` (34
lines, `mode dcb`, `decides_on`, `events`, `where`, `tag`, the boolean operators and a `tags` key) and
`unreserved-keywords.emod` (56 lines, `on` and `every` as activations and then as a field's name, type
and modifier). Each opens with a five-line header explaining that a marker asserts against the nearest
source line above and only discriminates while another highlighted token follows it on that line.
`tree-sitter.json` declares `queries/highlights.scm` under `highlights`, which is what makes
`tree-sitter test` run these files; `Taskfile.yml:80-88` runs `tree-sitter generate`, `tree-sitter test`
and then `go test -count=1 -tags grammar ./test/queries/...`, and `.github/workflows/ci.yml:33` runs
that target.

**The TextMate grammar.** `editors/vscode/syntaxes/emod.tmLanguage.json` is 122 lines. Its top-level
`patterns` list (`:6-12`) orders `#comments`, `#fields-block`, `#keyword-entity`, `#positional-keywords`,
`#standalone-tokens`. `fields-block` (`:22-38`) is a `begin`/`end` rule keyed on `fields` and its brace,
whose inner `patterns` (`:33-37`) are `#comments`, `#keyword-entity` and `#standalone-tokens` —
`#positional-keywords` is deliberately absent, and its comment says why. `keyword-entity` (`:39-66`)
holds three patterns: keywords taking a quoted entity name, keywords taking an identifier entity name,
and `target` with its trailing context name. `positional-keywords` (`:67-89`) is the case-sensitive rule
for `on`, `every` and an event's wire `type`.
`standalone-tokens` (`:90-99`) chains `#strings`, `#keywords`, `#field-modifiers`, `#field-types`,
`#operators`, `#punctuation`. The flat `keywords` alternation (`:100-104`) is case-insensitive and
positionless and carries all thirty-five spellings other than `on`, `every` and `type`, including `emod`,
`description`, `invariant`, `spec`, `given`, `when`, `then` and `rejected`. Nothing in the file assigns a
scope to a field's *name*, and nothing assigns a numeric or boolean scope.

**The VS Code scope suite.** `editors/vscode/test/scopes/` holds nine assertion files —
`declarations.emod`, `activations.emod`, `fields.emod`, `specs.emod`, `dcb.emod`, `comments.emod`,
`strings.emod`, `unreserved-keywords.emod`, `wire-type.emod` — driven by
`editors/vscode/test/scope-assertions.test.js` through `vscode-tmgrammar-test`. That file runs five
tests from four call sites (`:109`, `:115`, `:134` twice over a table of keyword sets, `:149`): the
shipped grammar satisfying every assertion, and four negative controls built from
`grammarWithScopeRenamed` (`:48`), `grammarWithFlatKeywordsExtended` (`:54`) and
`assertionsWithScopeRenamed` (`:66`), each checked through `assertReportsMismatch` (`:94`), which
requires the failure report to name the file, the position, the required or prohibited scope, and the
scope actually produced. `specs.emod` already asserts `keyword.control.emod` on `invariant`,
`description`, `spec`, `given`, `when`, `then` and `rejected` in their own positions; no file asserts
anything about a version header. `Taskfile.yml:90-95` runs `npm ci` then `npm test` in
`editors/vscode`, `Taskfile.yml:54-61` includes it in `test`, and `.github/workflows/ci.yml:35` runs it.

**What US-010 will add and this story reads.** US-010 Task 1 teaches the lexer a number literal with an
optional fractional part beside the existing `Integer` kind, and the parser an optional
`{ field: value, ... }` block after any event or command reference in a spec, with three literal forms.
US-010 Task 7 adds the matching rule to `editors/tree-sitter-emod/grammar.js` and corpus cases in
`test/corpus/specs.txt` and `test/corpus/fields.txt`, one of which declares fields named `true` and
`false`. Its criteria state that no file under `editors/tree-sitter-emod/queries/` changes and that
`emod.tmLanguage.json` is untouched. The node names that Task 3 below captures are the ones US-010
Task 7 introduces; this story reads them rather than choosing them.

**Not touched, deliberately.** `internal/lexer`, `internal/parser`, `internal/formatter`,
`internal/validator`, `internal/linter`, `internal/export`, `internal/lsp`, `internal/viewer` and every
other Go package; `editors/tree-sitter-emod/grammar.js` and everything under `test/corpus/`;
`queries/folds.scm`, `indents.scm` and `textobjects.scm`, which name no keyword and no literal;
`editors/tree-sitter-emod/test/queries/*.go`; `Taskfile.yml` and `.github/workflows/ci.yml`, both of
which already run every suite this story writes into; `docs/`, `README.md`, `examples/`.

---

## Tasks

### Task 1: Assert the field-name and keyword captures for the spec and metadata keywords in tree-sitter

**Behavior:** a `fields` block line named after any of the eight keywords the lexer already defines —
`emod`, `description`, `invariant`, `spec`, `given`, `when`, `then`, `rejected` — receives the
field-name capture and not the keyword capture, and the same eight receive the keyword capture at their
own positions, both proved at named row and column by the highlight suite `tree-sitter test` already
runs. Measured against the shipped query, the captures are already right in both positions and nothing
in the repository says so, so this task is the story's third criterion becoming falsifiable on the
tree-sitter side and its first criterion acquiring the receipt it has never had.

**Acceptance Criteria:**
- [ ] `mise exec -- task test:grammar` passes, and its `syntax highlighting:` section reports a
      non-zero assertion count for each file under `editors/tree-sitter-emod/test/highlight/`, the new
      one included
- [ ] The suite asserts the keyword capture on each of `emod` in a version header, `description` on a
      construct, `invariant` on a context and on an aggregate, and `spec`, `given`, `when`, `then` and
      `rejected` inside a slice's spec — every one at its own row and column
- [ ] The version-header assertion's marker column falls inside the capture the query actually
      produces, which is one character wider than the word: `grammar.js:27-28` folds the separator into
      the token, so the capture spans `emod ` and not `emod`
- [ ] The suite asserts the field-name capture, and not the keyword capture, on a `fields` block line
      named after each of the same eight spellings
- [ ] The same eight spellings are also asserted in a field's *type* position and in a field's
      *modifier* position, so the criterion covers every position a keyword may legally occupy on a
      field line, matching the coverage `unreserved-keywords.emod:7-46` gives `on` and `every`
- [ ] Every marked token is followed on its own line by a further highlighted token — a brace, an
      operator, another keyword or a trailing comment — so no assertion is satisfied by a capture the
      runner found on a later line, and the file's header says so the way the three existing files' do
- [ ] Every assertion can fail: removing a spelling from the `@keyword` list at
      `queries/highlights.scm:18-63` makes `mise exec -- task test:grammar` fail on that keyword's
      assertion, and removing the `@variable.member` capture at `:89-90` makes it fail on the
      field-name assertions, each run naming the row, the column and the capture actually produced
- [ ] The assertion source is emod the parser accepts in its shape on main — no payload, no `type`
      attribute and no `after` clause, none of which have landed
- [ ] `editors/tree-sitter-emod/test/highlight/constructs.emod` gains no second line beginning
      `fields {`, `trigger "Review orders"`, `automation NotifyCustomer` or `automation ArchiveOrders`:
      `sample_test.go:12` reads that file as `samplePath` and `headerRow` (`:38-49`) fails when a
      `sampleBlocks` header heads more than one line of it
- [ ] `git diff` shows no change to `editors/tree-sitter-emod/grammar.js`, to any file under
      `test/corpus/`, to `queries/folds.scm`, `indents.scm` or `textobjects.scm`, to `Taskfile.yml` or
      to `.github/workflows/ci.yml`
- [ ] `git ls-files editors/tree-sitter-emod/src` returns nothing, and running
      `mise exec -- task test:grammar` a second time leaves every tracked file under
      `editors/tree-sitter-emod/` byte-identical

**Affected Files/Modules:**
- `editors/tree-sitter-emod/test/highlight/` — a new assertion file for the spec and metadata
  constructs, and the keyword-named-field cases, extending or filing beside
  `unreserved-keywords.emod`
- `editors/tree-sitter-emod/queries/highlights.scm` — read; edited only if an assertion proves a
  capture wrong

**Patterns to Follow:**
- The three existing assertion files are the format and the house style, headers included:
  `editors/tree-sitter-emod/test/highlight/constructs.emod`, `dcb.emod` and `unreserved-keywords.emod`
- `unreserved-keywords.emod:7-46` is the shape for covering one spelling across a field line's three
  positions, and its comments at `:31-32` and `:39-40` explain why the modifier-less and domain-typed
  variants are there
- The captures to assert are the ones `queries/highlights.scm` already names in its section comments
  (`:11-108`)
- `tasks/learnings.md` "A tree-sitter highlight marker only discriminates while another highlighted
  token follows it on the same line"
- `tasks/learnings.md` "`highlights.scm` field patterns select by anchor, never by `#match?` on the
  token's text" — the three field patterns and why each anchored step carries `(comment)*`
- `tasks/learnings.md` "New DSL keywords must stay usable as field names" — the tree-sitter side is
  already structurally correct because a keyword token is valid only in its own parse state; this task
  makes that assertable rather than changing it
- `tasks/learnings.md` "An assertion whose expected value comes from the code under test is the
  recurring review finding" — name each expected capture, and check by mutation that it fails
- `tasks/learnings.md` "Run repo tooling through `mise exec --`, not bare PATH" and "Generated
  tree-sitter `src/` stays gitignored"
- `editors/tree-sitter-emod/test/corpus/specs.txt` and `version_header.txt` for the emod shapes these
  constructs take

**Testable:** Yes — through `tree-sitter test`, driven by `task test:grammar`, which runs in CI.

**Verification:** `mise exec -- task test:grammar`, run twice, the second run leaving tracked files
untouched.

**Depends on:** None

---

### Task 2: Paint a keyword-named field as a field name in the VS Code grammar

**Behavior:** a field named after a DSL keyword reads as a field name in VS Code, the way it already
does under tree-sitter. `fields { description string required }` no longer paints `description` as
though it opened a declaration or `string` as though it were a declared name, and the same holds for
every keyword the language defines, because the treatment keys on the field line's position inside the
`fields` block rather than on the name's spelling. The version header's `emod` keeps the keyword scope
where it opens a file.

**Acceptance Criteria:**
- [ ] `mise exec -- task test:vscode` passes
- [ ] Inside a `fields` block, a line whose name is any of `emod`, `description`, `invariant`, `spec`,
      `given`, `when`, `then` or `rejected` carries no `keyword.control.emod` on that name, asserted for
      each of the eight
- [ ] Such a name carries the same scope a non-keyword field name carries, so the treatment is
      positional rather than an allowlist — asserted on one keyword-named line and one ordinary line in
      the same file
- [ ] `editors/vscode/test/scopes/fields.emod:10-17` no longer records a keyword scope as the observed
      treatment: `command string required` asserts a field name on `command` and `storage.type.emod` on
      `string`, and the comment explaining that the assertions state what the grammar produces rather
      than what it ought to is gone with the behaviour it described
- [ ] The repair reaches every keyword and not only the eight: a `fields` line named `flow`, one named
      `reads` and one named `decides_on` are asserted the same way, and no rule in
      `emod.tmLanguage.json` enumerates which spellings are exempt in field position
- [ ] A field's *type* and *modifier* keep the scopes they carry today on every line the suite already
      asserts — `storage.type.emod` on a built-in type, `storage.modifier.emod` on a modifier, and
      neither on a domain type — with `command string required` the one line that changes, because its
      type stops being painted as a declared entity name
- [ ] Every assertion in `editors/vscode/test/scopes/unreserved-keywords.emod` passes unedited: `on`
      and `every` are already correct in all three field positions and this task neither narrows nor
      widens their treatment
- [ ] A version header's `emod` carries `keyword.control.emod`, asserted in an assertion file — the one
      spelling of the eight that no scope file names today
- [ ] The single-line form `fields { <keyword> string required }` is covered by an assertion stating
      the scope its name actually receives, and a comment beside it says whether that matches the
      multi-line treatment; the form is legal input but never formatter output, since
      `internal/formatter/formatter.go:285-311` writes one field per line
- [ ] `scope-assertions.test.js` gains a negative control for the repair: a copy of the grammar with the
      field-name treatment removed makes the keyword-named-field assertions fail, and the report names
      the position, the prohibited scope and the scope actually produced
- [ ] The four negative controls already in `scope-assertions.test.js` (`:115`, `:134` twice, `:149`)
      still pass, the `on` / `every` one unedited — extending the flat alternation with those two
      spellings must still break the field-name assertions
- [ ] `emod.tmLanguage.json` is valid JSON, `positional-keywords` (`:67-89`) is unchanged, and the flat
      `keywords` alternation (`:100-104`) still spells every lexer keyword other than `on`, `every` and
      `type`, so `mise exec -- task test:grammar` still passes `TestEditorKeywordCoverage`
- [ ] `git diff` touches only `editors/vscode/syntaxes/emod.tmLanguage.json` and files under
      `editors/vscode/test/`

**Affected Files/Modules:**
- `editors/vscode/syntaxes/emod.tmLanguage.json` — `fields-block` (`:22-38`) and its inner `patterns`
  list (`:33-37`), and a repository entry for the field-name treatment
- `editors/vscode/test/scopes/fields.emod` — the recorded gap at `:10-17`
- `editors/vscode/test/scopes/` — assertions for the eight keywords in field position, for the version
  header, and for the single-line form
- `editors/vscode/test/scope-assertions.test.js` — the new negative control beside the three at
  `:115-160`

**Patterns to Follow:**
- `positional-keywords` (`emod.tmLanguage.json:67-89`) is the precedent for a treatment keyed on position
  rather than on the word alone, and `fields-block`'s comment (`:23`) states the reason the block
  boundary is the only thing that can tell a field line from an activation
- `highlights.scm:83-102` is the tree-sitter counterpart this brings VS Code level with, and its
  header comment (`:7-8`) says the two files are meant to agree on scope assignment; the capture it
  gives a field's name is `@variable.member`
- `tasks/learnings.md` "A keyword that is only a keyword in one position never joins the flat TextMate
  alternation" — including the note that `fields` is a `begin`/`end` block whose inner patterns
  deliberately omit `#positional-keywords`, and that `standalone-tokens` exists so the file-level and
  block-level tails cannot drift apart
- `tasks/learnings.md` "A suite that pins another tool's output owes a mutated-input negative control"
  — the mismatch helper `assertReportsMismatch` (`scope-assertions.test.js:94`) is what a new control
  reports through
- `tasks/completed/us-003-use-reserved-words-as-field-names.md:57-61` and
  `tasks/completed/us-010-highlight-the-realigned-syntax.md:85-92` record the two earlier decisions to
  leave this gap, and what has changed since
- The scope names in play are the ones the file already uses: `keyword.control.emod`,
  `entity.name.function.emod`, `storage.type.emod`, `storage.modifier.emod`,
  `string.quoted.double.emod`, `keyword.operator.emod`, `comment.line.number-sign.emod`
- `tasks/learnings.md` "Every editor-highlight surface now has an executable suite, and each is run a
  different way" — `task test:vscode` is already in the `test` aggregate and in CI, so no Taskfile or
  workflow edit is owed here

**Testable:** Yes — through `vscode-tmgrammar-test`, driven by `task test:vscode`, which runs in CI.

**Verification:** `mise exec -- task test:vscode`; `mise exec -- task test:grammar` still green for
`TestEditorKeywordCoverage`.

**Depends on:** None

---

### Task 3: Capture payload numbers and booleans as literals in the tree-sitter highlight query

**Behavior:** a number and a `true` / `false` written as a spec payload value read as literals rather
than as unhighlighted text, distinct from the quoted strings beside them, while a field named `true` or
`false` keeps its field-name treatment.

**Acceptance Criteria:**
- [ ] `mise exec -- task test:grammar` passes, and the highlight suite asserts a numeric capture on a
      payload value written without a fractional part and on one written with one, and a boolean
      capture on `true` and on `false`
- [ ] The literal captures are asserted in all three payload positions the payload grammar admits — on
      a `given` list element, on the `when` reference, and on a `then` event-list element
- [ ] A quoted payload value keeps the string capture the query already gives every string, asserted on
      the same payload line as a number, so the two treatments are proved distinct rather than merged
- [ ] A `fields` block declaring a field named `true` and one named `false` asserts the field-name
      capture on both, not the boolean capture — the field lines US-010 Task 7's corpus case parses as
      `any_identifier`
- [ ] The capture names follow the convention the rest of `queries/highlights.scm` uses, and the new
      section carries a header comment describing what it captures the way the file's other seven
      sections do
- [ ] Every new assertion can fail: deleting the numeric or boolean pattern from `highlights.scm` makes
      `mise exec -- task test:grammar` fail on the corresponding assertions, naming the row, the column
      and the capture actually produced
- [ ] Each marked literal is followed on its own line by a further highlighted token, so no assertion is
      satisfied by a capture found on a later line
- [ ] `editors/tree-sitter-emod/grammar.js` and every file under `test/corpus/` are unchanged: the
      payload rule and its corpus cases are US-010 Task 7's, and this task names the node types that
      task introduced rather than adding any
- [ ] `git ls-files editors/tree-sitter-emod/src` returns nothing, and running
      `mise exec -- task test:grammar` a second time leaves every tracked file under
      `editors/tree-sitter-emod/` byte-identical

**Affected Files/Modules:**
- `editors/tree-sitter-emod/queries/highlights.scm` — a literals section, filed beside the string
  capture (`:14-16`) and the keyword list (`:18-63`)
- `editors/tree-sitter-emod/test/highlight/` — payload assertions, in the file Task 1 added or beside it

**Patterns to Follow:**
- The payload rule and the node names it introduces: `editors/tree-sitter-emod/grammar.js` after
  US-010 Task 7, and the corpus cases that task adds to `test/corpus/specs.txt` and
  `test/corpus/fields.txt` — including the field-named-`true` case this task's fourth criterion reads
- `tasks/us-010-state-example-payloads-in-specs.md:66-73` — `true` and `false` are recognized by
  position and are not keywords, so nothing here belongs in the `@keyword` list
- `queries/highlights.scm:14-16` is the sibling treatment a literal joins, and the section comments
  throughout the file are the house style for the new one
- `tasks/learnings.md` "A tree-sitter highlight marker only discriminates while another highlighted
  token follows it on the same line"
- `tasks/learnings.md` "Run repo tooling through `mise exec --`, not bare PATH" and "Generated
  tree-sitter `src/` stays gitignored"

**Testable:** Yes — through `tree-sitter test`, driven by `task test:grammar`.

**Verification:** `mise exec -- task test:grammar`, run twice, the second run leaving tracked files
untouched.

**Depends on:** 1, and US-010 Task 7 — there is no payload node to capture before it lands, and a
fixture containing a payload does not parse

---

### Task 4: Scope payload numbers and booleans as literals in the VS Code grammar

**Behavior:** VS Code paints a spec payload's numbers and its `true` / `false` as literals, matching
what tree-sitter now shows, without painting a field named `true` or `false` and without reaching
inside a quoted string that happens to carry digits.

**Acceptance Criteria:**
- [ ] `mise exec -- task test:vscode` passes, and an assertion file asserts a numeric scope on a payload
      value written without a fractional part and on one written with one, and a boolean scope on `true`
      and on `false`
- [ ] The literal scopes are asserted on a `given` list element, on the `when` reference and on a `then`
      event-list element
- [ ] A quoted payload value keeps `string.quoted.double.emod`, asserted on the same line as a number,
      so the two treatments are proved distinct
- [ ] Inside a `fields` block, a field named `true` and a field named `false` carry no boolean scope —
      the boolean treatment keys on payload position the way `positional-keywords` (`:67-89`) keys
      `on`, `every` and `type` on their operand, so it cannot be a positionless word alternation
- [ ] `editors/vscode/test/scopes/strings.emod:14-17` passes unedited: a cron schedule keeps
      `string.quoted.double.emod` across its whole span, with no numeric scope on the digits inside it
      — `standalone-tokens` (`:85-94`) lists `#strings` ahead of the rules that follow it, and that
      assertion is what holds the ordering
- [ ] The treatment given to a version header's `1` is asserted either way, so the choice between a
      positionless numeric rule and a payload-keyed one is recorded in the suite rather than left to be
      re-derived from the grammar
- [ ] An identifier carrying digits — an event named `Version2Placed`, a field named `line1` — carries
      no numeric scope
- [ ] `scope-assertions.test.js` gains a negative control for the literal treatment: a copy of the
      grammar with the boolean rule turned into a positionless word alternation makes the
      field-named-`true` assertion fail, and the report names the position, the prohibited scope and the
      scope actually produced
- [ ] The four negative controls already in `scope-assertions.test.js` (`:115`, `:134` twice, `:149`)
      and the one Task 2 added still pass, unedited
- [ ] `emod.tmLanguage.json` is valid JSON, its top-level `patterns` order (`:6-12`) still puts
      `#comments` and `#fields-block` ahead of the rest, and `mise exec -- task test:grammar` still
      passes `TestEditorKeywordCoverage`
- [ ] `git diff` touches only `editors/vscode/syntaxes/emod.tmLanguage.json` and files under
      `editors/vscode/test/`

**Affected Files/Modules:**
- `editors/vscode/syntaxes/emod.tmLanguage.json` — a literals entry in `repository`, and its place in
  `standalone-tokens` (`:85-94`) or in a payload-scoped rule
- `editors/vscode/test/scopes/` — payload assertions, and the field-named-`true` case
- `editors/vscode/test/scope-assertions.test.js` — the new negative control

**Patterns to Follow:**
- `positional-keywords` (`emod.tmLanguage.json:67-89`) is the precedent for a token that is only itself
  in one position, and its comment records why it is case-sensitive where the surrounding rules are not
- `tasks/learnings.md` "A keyword that is only a keyword in one position never joins the flat TextMate
  alternation" — `true` and `false` are not keywords, but they carry exactly the same hazard, and the
  `fields` block is the same boundary that resolves it
- `tasks/us-010-state-example-payloads-in-specs.md` — the payload's shape, the three literal forms, the
  optional comma, and the decision that numbers are unsigned so a leading `-` never appears in a payload
- `queries/highlights.scm` after Task 3 — the two files are meant to agree on scope assignment, which
  its header comment (`:7-8`) states, so the TextMate scope names should be the recognised counterparts
  of the captures Task 3 chose
- `tasks/learnings.md` "A suite that pins another tool's output owes a mutated-input negative control"
  and the `assertReportsMismatch` helper (`scope-assertions.test.js:94`)
- `editors/vscode/test/scopes/strings.emod:14-17` for how the suite already pins a digit-carrying
  string's full span, which is the assertion a numeric rule must not disturb

**Testable:** Yes — through `vscode-tmgrammar-test`, driven by `task test:vscode`.

**Verification:** `mise exec -- task test:vscode`; `mise exec -- task test:grammar` still green.

**Depends on:** 2 and 3, and US-010 — the payload's spelling must be settled before an assertion file
can state it

---

## Summary

**Four tasks**, two per editor surface, ordered so the half that needs nothing from another story lands
first. The weight sits on the second and third criteria, because the first needs no grammar change on
either surface: `TestEditorKeywordCoverage` already forces every lexer spelling onto all three editor
files, and separate measurement confirms all eight are correctly *scoped* in keyword position and not
merely present. Three of the four tasks are therefore literal work or field-name work.

Tasks 1 and 2 close the field-name criterion and are available today. Task 1 is pure assertion: the
tree-sitter query is measurably correct for all eight spellings in both keyword position and field-name
position, and nothing in the repository says so or would notice if it stopped being true. Task 2 is the
one real grammar repair in the story — the TextMate grammar paints a keyword-named field as a keyword,
measured against the shipped grammar, and the `fields-block` rule added by the previous highlighting
story is what makes the fix a rule rather than a restructuring. The repair is general because the rule
has no per-keyword axis, so it retires the recorded gap for all thirty-seven spellings at once.

Tasks 3 and 4 close the literal criterion and wait on US-010 Task 7, which introduces the payload node
types Task 3 captures. They are ordered tree-sitter first so the TextMate scope names have a settled
counterpart to mirror, which is what `highlights.scm:7-8` asks of the pair.

**Story criteria coverage:**

| Story criterion | Task |
|---|---|
| `spec`, `given`, `when`, `then`, `rejected`, `invariant`, `description`, `emod` highlighted in the tree-sitter grammar | already correct on main (measured); 1 adds the assertions, since nothing pinned it |
| The same eight highlighted in the VS Code extension | already correct on main (measured), asserted for seven by `editors/vscode/test/scopes/specs.emod`; 2 adds the version header's `emod` |
| `type` highlighted in both | US-012 Task 1 |
| `after` highlighted in both | US-013 Task 1 |
| Numbers and `true` / `false` in payload position highlighted as literals | 3 (tree-sitter), 4 (VS Code) |
| A keyword in field-name position highlighted as a field name, not a keyword | 1 (tree-sitter, already correct — made assertable), 2 (VS Code, repaired) |

**What each surface can express, stated rather than promised.** Tree-sitter scopes by syntax node, so
both the field-name criterion and the literal criterion are structural there: a field line's name is
`field_line`'s first `any_identifier` whatever it spells, and a payload literal is a node the payload
rule names. Neither needs to know a list of words, and both are already right or will be the moment
US-010's rule exists — which is why the tree-sitter tasks are assertion work. TextMate matches regular
expressions against a line with only the block context a `begin`/`end` rule supplies, so both criteria
there are positional rules that must be written and can be got wrong: the field-name treatment lives
inside `fields-block` and the boolean treatment must key on payload position, because the positionless
alternation that serves thirty-five keywords would paint a field named after one. Two things TextMate
cannot reach and the tasks say so instead of promising parity: a field name on the block's own opening
line, which Task 2 records rather than requires; and any distinction that depends on the parse rather
than on the text, which is why the boolean rule is keyed on its operand and not on the word.

**Deferred to other stories:** `type` and `after` on every editor surface, each carried by the story
that teaches the lexer the spelling, because `TestEditorKeywordCoverage` will not let either land
otherwise (US-012, US-013); the payload grammar in `grammar.js`, the lexer, the parser and everything
downstream of them (US-010); LSP hover, completion, navigation and semantic tokens (US-015); diagram
rendering of specs (US-016); examples and the DSL reference (US-018). Left where it stands: an
invariant's declared name and a spec element's event or command reference, neither of which any surface
captures today, and a payload field name, which matches the uncaptured DCB `tags` key beside it.

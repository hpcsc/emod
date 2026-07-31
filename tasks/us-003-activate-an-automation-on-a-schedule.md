# US-003: Activate an automation on a schedule

## Progress
- [x] Task 1: Parse `every "<expr>"` and require exactly one activation form
- [x] Task 2: Emit `every` from `emod fmt`
- [x] Task 3: Accept `every` inside an automation in the tree-sitter grammar
- [x] Task 4: Reject an `every` expression that is neither a duration nor a five-field cron
- [x] Task 5: Share a fixture whose automations activate on a schedule
- [x] Task 6: Carry the schedule into the JSON, CUE and embedded schema exports
- [x] Task 7: Carry the schedule through the diagram document and back
- [ ] Task 8: Draw a clock badge on a scheduled automation in SVG and draw.io
- [ ] Task 9: Name the cadence in Mermaid, ASCII and `emod slices`
- [ ] Task 10: Show a scheduled automation's cadence in the viewer

---

## Story Reference

`user-stories/triggers-and-automations.md` → **US-003: Activate an automation on a schedule**
(third of eleven stories in "Triggers and Automations"). Design notes:
`docs/proposals/triggers-and-automations-proposal.md` — section 2 for the block shape, section 3 for
the two activation forms, `:195` for the post-block arity diagnostics, `:201` for the formatter,
`:207-211` for the validator, `:233`/`:238` for the schema and document keys, `:303` for the clock
badge.

**In scope:** an `every "<expr>"` entry inside `automation <Name> { ... }` whose expression is either a
Go duration or a five-field cron expression; the exactly-one-of rule over `on` and `every`, replacing
the "requires an activation event declared with on" check US-002 left in `parseAutomation`; a
validation error, positioned at the expression, when the text matches neither form; `emod fmt` writing
`every` at a fixed position in the block; the value in the JSON and CUE exports and in the embedded
`schema.cue`; the value surviving export to diagram JSON, editing in the viewer and re-import; a clock
badge carrying the expression wherever an automation is drawn, and the cadence named wherever an
automation is summarised as text. Carried along because the repo would otherwise be inconsistent or
wrong: the `every` keyword in `internal/lexer/token.go`, the tree-sitter grammar (which must never
reject what `emod validate` accepts), a shared fixture exercising the entry in both slice homes, and
the three text surfaces that interpolate an automation's activation event unconditionally
(`internal/diagram/ascii.go:86`, `internal/diagram/mermaid.go:124`/`:247`/`:390`,
`internal/cli/slices.go:174`), each of which prints a hole or a leading comma for an automation that
has no activation event.

**Out of scope:** dropping the trigger kind slot, so `trigger UI "…"`, `trigger Schedule "…"` and
`trigger Processor "…"` all still parse at slice level and no `.emod` source in the tree is migrated
(US-004); the `reads` edge from a view to an automation or trigger (US-005); lane placement, so a
scheduled automation stays exactly where an event-activated one sits today (US-006); the palette
(US-007); the `automation/missing-todo-list` lint rule, so this story adds no rule and no `RuleName`
(US-008); LSP hover, completion, go-to-definition and find-references over `every` — including
`keywordDescriptions` (`internal/lsp/hover.go:13`) and the automation-body completion list (US-009);
the VS Code TextMate keyword alternation (`editors/vscode/syntaxes/emod.tmLanguage.json:63`) and
`editors/tree-sitter-emod/queries/*.scm` (US-010); `docs/dsl-reference.md`, `README.md` and
`examples/*.emod`, none of which this story edits (US-011 — and note `tasks/learnings.md` records that
the reference already documents the retired automation `trigger` spelling, which US-011 owns fixing);
relative delays on activation (`on <Event> after "<duration>"`), runtime scheduling semantics, and a
DSL version bump — the header stays at 1.

**Consequences of that boundary, decided.** Seven shapes the story does not spell out:

1. *The arity check is the parser's, the expression check is the validator's.* The post-block
   completeness diagnostics already live in `parseAutomation` (`internal/parser/parser.go:1053`), and
   the proposal (`:195`, `:207-211`) keeps them there while giving the validator the shape check. So
   "declares both" and "declares neither" are parse diagnostics reported at the automation's name, and
   "the expression matches neither form" is a validator diagnostic reported at the expression. Both
   reach an author through `emod validate`, which is what the story's criteria say.
2. *Validation checks the shape and not the semantics, in both grammars.* A five-field expression whose
   numbers are out of range (`"99 * * * *"`) is accepted, and so is a negative or zero duration
   (`"-5m"`, `"0s"`): the model states a cadence that nothing here evaluates, and per-field range
   tables would be a second grammar to maintain for a tool that never schedules anything. This is a
   decision, not an omission — Task 4 pins it so a later change to it is deliberate.
3. *An automation may carry both values in the AST.* When a block declares `on` and `every`, the parser
   reports the arity error and still records both, so every downstream writer sees a fully parsed block
   and nothing has to reason about a half-built node. Rejecting the second entry instead would make
   the recorded value depend on writing order.
4. *There is no `Without…` twin for this feature, and the unfeatured witnesses already exist.* Every
   automation in `test.HotelReservation`, `test.DescribedHotelReservation`,
   `test.KeywordFieldSearchCatalog`, `test.SpecLibraryLending` and `test.AutomationReadsLibraryLending`
   states `on` and no `every`, so those fixtures are the byte-identical receipt that a model not using
   the feature is untouched. Stripping `every` from a scheduled automation is not available as a twin:
   it would leave a model the parser rejects.
5. *Every criterion phrased "still", "unchanged" or "as before" is asserted on input that also
   exercises the change.* The new fixture declares scheduled and event-activated automations side by
   side in both slice homes, so the with- and without- cases are read back in the same assertion rather
   than from a model that omits the feature entirely.
6. *The clock badge lands in the three renderers that draw boxes* — SVG, draw.io and the viewer, the
   same three US-005 and US-006 name. Mermaid and ASCII render an automation as a line of text and get
   the cadence in words instead; ASCII in particular must stay all-ASCII apart from `⚙`
   (`internal/diagram/ascii_test.go:319`), so no clock glyph goes there.
7. *`emod slices` is fixed even though the story does not mention it.*
   `keyElementsForPattern` (`internal/cli/slices.go:174`) joins an automation's activation event and
   command with a comma, so a scheduled automation prints a listing row beginning with one. A story
   that makes the shape expressible owns the surfaces that then misread it.

**Learnings folded in** from `tasks/learnings.md`: a new block entry keyword owes three things to the
parser's diagnostics (the "expected …" message, one diagnostic per malformed entry recovered through
`skipRestOfLineOrBlockEnd`, and a `require.Len(t, diags, 1)` pin); assert a short keyword in a
diagnostic with a `\b`-bounded `require.Regexp`, since `on` hides inside `automation` and
`description`; parser diagnostics at a stored position go through `p.errorAtPosition`; ask the lexer
which keywords exist and never restate the set; new DSL keywords must stay usable as field names, on
both the Go and the tree-sitter side; put a new parser subtest in the group that owns the construct; a
new block entry goes after `description` and ahead of nested blocks in every writer, and the formatter
silently deletes an entry it has never heard of, so parse → format → reparse against the original
model is the guard; `emod fmt` canonicalises order, so a fmt golden is never the input re-indented,
and formatter output always begins with `emod N`; never write emod source with `%q`; a new exported
field must land in JSON, CUE and `schema.cue` in the same change, the two order their keys
differently, and `emittedKeyOrder` makes the JSON order assertable from the raw bytes; read a decoded
export document back with `objectsUnder`/`statedUnder` in the writer's slice order; the two export
guards cannot see a value neither writer emits; a new optional field ships a six-part fixture kit; a
new shared fixture owes `internal/oracle` a zero-diagnostic subtest; a slice has two homes and a
fixture that declares the construct in only one cannot catch a one-home walk; exercise an omitted
optional part mid-block, never as the last entry; a `Declared…` getter answers `nil` for a fixture
declaring none of the construct, so pair it only with a non-empty transcribed list; the viewer's save
path is `importer.ImportDiagram`, so a diagram-node field owes a read-back, a diagram-node key has
three readers that must move together, and `internal/viewer/static` is a display surface with its own
vitest harness; the tree-sitter grammar must never be stricter than the Go parser, its generated
`src/` stays gitignored, and every `grammar.js` rule carries a one-line example of its full shape; run
repo tooling through `mise exec --`; CLI diagnostic tests must assert the distinguishing message text;
an assertion whose expected value comes from the code under test cannot fail; acceptance criteria
describe the working tree, and a commit-message receipt is the commit author's obligation, never a
criterion.

---

## Codebase Context

**Lexer.** `internal/lexer/token.go` declares the `Kind` constants in one iota block (`:9-56`, `on`
appended last at `:48`) and a lowercase `keywords` map (`:67-110`); `Keywords()` lists the spellings
sorted and `Kind.IsKeyword()` is a map lookup. Four subtests range over `Keywords()` —
`internal/lexer/tokenizer_test.go:13`, `internal/parser/parser_test.go:226` and `:242`,
`internal/oracle/oracle_test.go:44` — so a keyword added to the map is covered the day it exists.
`isKeyword` in `internal/lsp/hover.go:37` is still an ordinal range and will not see a `Kind` appended
after `KeywordOn`, which is US-009's to fix.

**AST.** `ast.Automation` (`internal/ast/ast.go:202-219`) carries `Comments`, `Name`/`NamePos`,
`Description`/`DescriptionPos`, `OnEvent`/`OnEventPos`, `Reads`/`ReadsPos`, `Command`/`CommandPos`,
`TargetContext`/`TargetContextPos`, `OpenPos` and `ClosePos`, in that order — every writer reads the
struct in that sequence.

**Parser.** `parseAutomation` (`internal/parser/parser.go:1020-1061`) loops `parseAutomationEntry`
(`:1063-1105`) until `}`, then reports `automation block requires an activation event declared with on`
(`:1054`) and `automation block requires a command` (`:1057`) through `p.errorAtPosition` at the
automation's name. The entry switch dispatches `description` to `parseDescriptionInto` (`:1391`, the
quoted-string shape: error at the offending token, drain with `skipRestOfLineOrBlockEnd` at `:1498`),
`on` and `reads` to `parseIdentifierEntryInto` (`:1404`), and reports
`expected description, on, reads, command, or target in automation, got %q` at `:1102`. The
`external_system` branch of `parseTranslation` (`:1136`) is the other quoted-string entry in the file.

**Parser tests.** `internal/parser/parser_test.go` is one umbrella of thirteen groups; `"automations"`
(`:1898`) owns the construct and `"error reporting"` (`:2576`) owns messages and recovery. The
subtests this story rewrites or extends: the malformed-entry loop asserting `\b`-bounded keywords
(`:2914`), the last-entry-drain loop over `{"reads", "on"}` (`:2928`), the accepted-entry loop
(`:3064`) with its `require.NotRegexp` on `trigger` (`:3066`), and the seven sites asserting the
`requires an activation event declared with on` text (`:3029`, `:3087`, `:3094`, `:3143`, `:3150`,
`:3319`, `:3325`) — all seven are in this file and nowhere else in the repo.

**Formatter.** `writeAutomation` (`internal/formatter/formatter.go:343-352`) emits `description`, `on`,
`reads`, `command`, `target context`, each through `lineIfSet`, which guards on the raw value —
`writeTranslation`'s `external_system` line (`:359`) is the shape for a value that must be written
quoted, and `quoted` (`:54`) is the only correct way to emit emod text. `"formats automation block"`
(`internal/formatter/formatter_test.go:258`) is the whole-output golden, `:309` is the entry-position
golden, and the round-trip group (`:538`, per-fixture leaves at `:851` and `:873`) is what catches a
dropped entry. `internal/cli/fmt_test.go` pins canonical `*FormattedEmod` constants (its automation is
at `:202`) and feeds them to `requireFmtSettlesOn` (`:557`).

**Validator.** `Validate` (`internal/validator/validator.go:12-36`) walks `index.slices` three times
and then appends the model-wide checks; `referenceDiagnostics` (`:298`) is where an automation's
`target context`, `command`, `on` event and `reads` view are resolved through `appendUndeclaredRef`
(`:325`). A check that reasons about a value's shape rather than its resolution has no home yet;
`scopedInvariantDiagnostics` (`:283`) and `specDiagnostics` (`:339`) are the two precedents for a
check that sorts its findings by position. Hard errors carry no `RuleName`.

**Fixtures.** `internal/test/fixtures.go` holds `AutomationReadsLibraryLending` (`:578`) with its
transcribed `…ViewNames` (`:758`) and `…ActivationEvents` (`:769`), the `Without…` twins built on
`copyWithEditedSlices` (`:821`) and `editedCopies` (`:839`), the `Declared…` getters (`:852-891`)
composed from `declaredAutomationEntries` (`:881`), and `declaredSlices` (`:893`), the walk that
visits an aggregate's slices and then the slices a `mode dcb` context declares directly.
`internal/test/models.go` holds one `…Model(t)` accessor per fixture (`:13-44`).
`internal/oracle/oracle_test.go:24` keeps one zero-diagnostic leaf per fixture.

**Exports.** `jsonAutomation` (`internal/export/export.go:158-173`) opens with `name`, `description`,
`position`, the four `*_position` keys, `comments`, then the four values; `convertAutomation` (`:598`)
fills it; `cueWriter.writeAutomation` (`:1368`) emits `on_event`, `reads`, `command`,
`target_context`; `internal/cue/schema.cue` `#Automation` (`:53-61`) declares the same keys with
`comments?` and `name` first. The read-back walkers are `exportedAutomations` (`export_test.go:4184`),
`statedUnder` (`:4210`), `objectsUnder` (`:4220`) and `emittedKeyOrder` (`:4310`); the key-order pin
for an automation is at `:1226`. `internal/cue/embed_test.go` vets `fullModelJSON` (`:126`, its
automation at `:177`) against `#Model`, and `:112` is the retired-key negative leaf.

**Diagram document and the viewer.** `jsonDiagramNode` (`internal/export/export.go:653-673`) carries
`on_event` among its type-specific keys; the automation node is built at `:829-846` and the
`automation_trigger` edge at `:1077` is guarded on a non-empty activation event, so a scheduled
automation already draws no incoming edge. `internal/importer/importer.go` reverses it: `diagramNode`
(`:25-42`) decodes the keys, `buildSlice` (`:206-215`) copies them onto the automation, and `foldEdges`
(`:271`) folds an `automation_trigger` edge onto an automation with no activation event.
`internal/viewer/static/renderer.js:206-272` draws each node — the translation branch (`:258-267`) is
the only one that draws more than a centred label, and it is the shape for a second line of text on a
box. `internal/viewer/static/ui.js:351-361` renders the automation section of the details panel.
`internal/viewer/tests/{renderer,detail-panel}.test.js` are the vitest harnesses, run by
`task test:viewer`, which is not part of `task test:unit`.

**Go diagram renderers.** SVG draws the automation box and its `⚙ Name` label at
`internal/diagram/svg.go:149-164` (tooltips ride on `svgRoundedRect`'s last argument) and its
event→automation arrow at `:256`; draw.io does the same at `drawio.go:404-416` (`vertexCell` takes a
tooltip) and `:540`; ASCII prints `(<event>) -> ⚙ <name> -> [<command>]` at `ascii.go:84-87` and
collects `flowEvts` at `:154`; Mermaid builds `<id> (<event> → <command>)` in three places
(`mermaid.go:114-127`, `:236-250`, `:379-393`). `internal/diagram/contract_test.go` asserts the
behaviour all four share and holds the differential receipts (`:236` for automation reads).

**Tree-sitter.** `automation_definition` (`editors/tree-sitter-emod/grammar.js:217-228`) passes its
entries to `buildDescribedBlock`, above a one-line comment spelling the construct out whole (`:217`).
Corpus cases live in `test/corpus/slice.txt` (`:259`, `:290`, `:323`, `:356`), `description.txt:155`
and the keyword-per-field block in `fields.txt:1-40`. `src/` is gitignored and `task test:grammar`
regenerates before running.

**Not touched, deliberately.** `internal/linter` (US-008 owns the only rule that reads an automation's
activation), `internal/glossary` (an automation contributes no term of its own), `internal/lsp`
(US-009), `editors/vscode` and `editors/tree-sitter-emod/queries` (US-010), `docs/`, `README.md`,
`examples/` and `internal/parser/testdata/` (US-011 — no `.emod` file in the tree gains a schedule),
`e2e/` and `e2e-viewer/`.

---

## Tasks

### Task 1: Parse `every "<expr>"` and require exactly one activation form

**Behavior:** `automation <Name> { ... }` accepts `every "<expr>"` anywhere among its entries,
recording the expression text and the source position of the string token. The block's completeness
check becomes an exactly-one-of rule over `on` and `every`: an automation declaring neither is
reported once, naming both as the options; an automation declaring both is reported once, naming both.
The message for an unrecognised entry names `every`. `every` is a keyword, so it is reserved for
element names and still legal as a field name, type and modifier. Nothing yet writes the value out —
the expression is parsed and held, and Task 2 is what stops `emod fmt` from dropping it.

**Acceptance Criteria:**
- [ ] An automation declaring `every "0 2 * * *"` and a command parses with no diagnostics and carries
      the text between the quotes exactly as written, together with the filename, line and column of
      the string token
- [ ] `every` written as the block's first entry and `every` written after `target context` both parse
      with no diagnostics and record the same expression — position within the block is free, as it is
      for `description`, `on` and `reads`
- [ ] An automation declaring `every` twice keeps the value written last
- [ ] One model declaring an automation with `on RoomReserved` and no `every` beside an automation with
      `every "5m"` and no `on` parses with no diagnostics, and each automation carries exactly the
      entry it declared and an empty string for the other
- [ ] An automation declaring both `on RoomReserved` and `every "5m"` reports exactly one diagnostic
      (`require.Len(t, diags, 1)`) positioned at the automation's name, whose message names `on` and
      `every` — each asserted with a `\b`-bounded `require.Regexp`, since `on` occurs inside
      `automation` — and the automation still carries both values
- [ ] An automation declaring neither reports exactly one diagnostic positioned at the automation's
      name, whose message names `on` and `every` as the two options under the same `\b` bounding; the
      seven sites in `internal/parser/parser_test.go` asserting the old
      `automation block requires an activation event declared with on` text move to the new wording,
      and no other file in the repo spells it
- [ ] An automation declaring `every` with a bare identifier in place of the quoted expression,
      followed on the next line by a `command` entry, reports exactly one diagnostic whose message
      names `every` and `automation`, and the `command` entry on the following line is still parsed
      onto the automation
- [ ] An `every` entry with nothing after it, written as the last entry of the block and followed by a
      second automation, reports exactly one diagnostic, and the automation, its slice and its context
      all close (`require.NotZero` on each `ClosePos.Line`) with the second automation parsed — the
      `{"reads", "on"}` loop at `internal/parser/parser_test.go:2928` gains the entry
- [ ] The unrecognised-entry message inside an automation names `every` alongside `description`, `on`,
      `reads`, `command` and `target` — the loop at `internal/parser/parser_test.go:3064` gains the
      entry and keeps passing, and its `require.NotRegexp` on `trigger` (`:3066`) still holds
- [ ] `lexer.Keywords()` contains `every`, `lexer.Scan` yields a token for it whose `Kind` is not
      `Identifier`, and a field named `every` with type `every` and modifier `every` parses with no
      diagnostics — all from the subtests that range over `Keywords()`
      (`internal/lexer/tokenizer_test.go:13`, `internal/parser/parser_test.go:226`, `:242`,
      `internal/oracle/oracle_test.go:44`), which pass unedited
- [ ] A model declaring an event named `Every` and an automation activating on it parses with no
      diagnostics, pinning that keyword lookup is case-sensitive
- [ ] `oracle.Check` over `test.HotelReservation`, `test.DescribedHotelReservation`,
      `test.KeywordFieldSearchCatalog`, `test.InvariantLibraryLending`, `test.SpecLibraryLending` and
      `test.AutomationReadsLibraryLending` returns no diagnostics, with those fixtures unedited

**Affected Files/Modules:**
- `internal/lexer/token.go` — the `Kind` iota block (`:9-56`) and the `keywords` map (`:67-110`)
- `internal/ast/ast.go` — `Automation` (`:202-219`) gains the expression and its position
- `internal/parser/parser.go` — `parseAutomationEntry` (`:1063`) gains the branch, its message (`:1102`)
  grows a term, and the post-block check (`:1053-1058`) becomes the exactly-one-of rule
- `internal/parser/parser_test.go` — the `"automations"` group (`:1898`) and the message and recovery
  subtests in `"error reporting"` (`:2576`)

**Patterns to Follow:**
- The entry to copy for a quoted value: `parseDescriptionInto` (`internal/parser/parser.go:1391`) —
  error at the offending token, then `skipRestOfLineOrBlockEnd` (`:1498`), which also stops at `}` so
  the block still closes. `parseIdentifierEntryInto` (`:1404`) is its identifier-valued sibling
- The post-block diagnostics: `p.errorAtPosition` with the automation's stored `NamePos`, as `:1053`
  and `:1057` already do — `tasks/learnings.md` "Parser diagnostics at a stored `ast.Position` go
  through `p.errorAtPosition`"
- `tasks/learnings.md` "A new block entry keyword owes three things to the parser's diagnostics" and
  "Assert a short keyword in a diagnostic with a `\b`-bounded `require.Regexp`", which names this
  story's `every` as one of the remaining short keywords
- `tasks/learnings.md` "Ask the lexer which keywords exist; never restate the set" — add the spelling
  to the `keywords` map and append the `Kind`; `checkIdentifierLike` already asks `Kind.IsKeyword()`
- `tasks/learnings.md` "Put a new parser subtest in the group that owns the construct" — the construct
  is the automation
- The block loop imposes no arity and no order on entries; keep it that way

**Testable:** Yes — through `lexer.Scan`, `parser.Parse` and `oracle.Check`, all exported.

**Verification:** `mise exec -- go test -tags unit ./internal/lexer/... ./internal/parser/...
./internal/oracle/...`; `mise exec -- go build ./...`.

**Depends on:** None

---

### Task 2: Emit `every` from `emod fmt`

**Behavior:** `emod fmt` writes an automation's schedule as `every "<expr>"` on its own line at the
block's indent, in a fixed position among the entries, so a scheduled automation survives formatting
and repeated formatting is stable. An automation stating `on` keeps the output it has today.

**Acceptance Criteria:**
- [ ] Formatting an automation that declares `every "0 2 * * *"` emits the entry on one line, directly
      below the automation's `description` and above `reads`, at the same indent as the sibling
      entries, with the expression written between plain quotes
- [ ] A model whose slice declares a scheduled automation and, beside it, an automation stating `on`
      formats to canonical bytes for both — the scheduled one carrying its `every` line and the
      event-activated one carrying its `on` line and no `every` line — asserted against one expected
      whole-block output, not by searching for a line
- [ ] Formatting the same model twice produces identical bytes, and an expression containing a
      backslash, a tab, a quote and a `%` survives a second format run unchanged (the language has no
      escape sequences)
- [ ] Parsing a scheduled automation, formatting the model and reparsing the result yields a model
      equal to the original under the round-trip comparison at
      `internal/formatter/formatter_test.go:538`, so the entry is proved not to be dropped
- [ ] A source file written with `every` on a line the author indented differently, and with the entry
      written after `command`, is rewritten by `RunFmt` to the canonical position and settles there —
      pinned with a canonical `*FormattedEmod` constant passed to `requireFmtSettlesOn`
      (`internal/cli/fmt_test.go:557`), never by handing the input fixture back as the expectation
- [ ] `git diff` shows no change to any existing expected constant in
      `internal/formatter/formatter_test.go` or `internal/cli/fmt_test.go`: no automation in those
      files declares a schedule, so no byte of their output may move

**Affected Files/Modules:**
- `internal/formatter/formatter.go` — `writeAutomation` (`:343-352`)
- `internal/formatter/formatter_test.go` — the automation goldens (`:258`, `:309`) and the round-trip
  group (`:538`)
- `internal/cli/fmt_test.go` — a canonical constant beside the existing ones (`:49`, `:118`, `:135`,
  `:243`) and its `requireFmtSettlesOn` leaf

**Patterns to Follow:**
- The line to copy for a quoted value: `writeTranslation`'s `external_system`
  (`internal/formatter/formatter.go:359`), which guards on the raw value and passes it through
  `quoted` (`:54`) — `lineIfSet` (`:25`) tests the string it is handed, so a pre-quoted value defeats
  its guard
- `tasks/learnings.md` "Never write emod source with `%q` — the language has no escape sequences", and
  its counterpart obligation of a round-trip subtest per hazard character
- `tasks/learnings.md` "A new block entry goes after `description` and ahead of nested blocks, in every
  writer" and "`emod fmt` canonicalises order, so a fmt golden is never the input re-indented"
- `tasks/learnings.md` "Formatter output always begins with `emod N`" — every expected constant starts
  with the version header even when the input fixture omits it

**Testable:** Yes — through `formatter.Format` and `cli.RunFmt`.

**Verification:** `mise exec -- go test -tags unit ./internal/formatter/... ./internal/cli/...`.

**Depends on:** 1

---

### Task 3: Accept `every` inside an automation in the tree-sitter grammar

**Behavior:** the tree-sitter grammar parses an automation declaring a schedule without an `ERROR`
node, so a file `emod validate` accepts is not red-squiggled in an editor using the grammar. Everything
the grammar accepted before is still accepted, including a field named `every`.

**Acceptance Criteria:**
- [ ] `automation_definition` (`editors/tree-sitter-emod/grammar.js:218-228`) admits an `every` entry
      taking a string, spelled the way its sibling entries are and passed to `buildDescribedBlock`
      rather than wrapped in `optional(...)`
- [ ] The one-line comment above the rule (`:217`) spells the construct out whole including the new
      entry, so the file's only description of an automation still lists every item it admits
- [ ] A corpus case in `editors/tree-sitter-emod/test/corpus/slice.txt` covers an automation whose
      activation is written with `every` ahead of `reads` and `command`, and its expected tree contains
      no `ERROR` or `MISSING` node
- [ ] A second corpus case covers `every` written after `target context`, proving the grammar imposes
      no order the Go parser does not
- [ ] The keyword-per-field corpus case (`test/corpus/fields.txt:1-40`) gains a field named `every` and
      parses with no `ERROR` node — if the anonymous token competes with `any_identifier`, narrow the
      token the way `version_header` narrows `emod`, rather than loosening the field rule
- [ ] The four existing automation cases (`slice.txt:259`, `:290`, `:323`, `:356`) and the described
      automation (`description.txt:155`) pass unedited
- [ ] `mise exec -- task test:grammar` passes, run through `mise exec --` so the repo-pinned
      tree-sitter CLI resolves rather than whichever one is on `PATH`
- [ ] `git check-ignore editors/tree-sitter-emod/src` succeeds, and the only files this task changes
      under `editors/tree-sitter-emod` are `grammar.js` and files under `test/corpus/`

**Affected Files/Modules:**
- `editors/tree-sitter-emod/grammar.js` — `automation_definition` (`:217-228`)
- `editors/tree-sitter-emod/test/corpus/slice.txt` — beside `Slice with automation` (`:259`)
- `editors/tree-sitter-emod/test/corpus/fields.txt` — the keyword-per-field block (`:1-40`)

**Patterns to Follow:**
- The string-valued entry spelling: `external_system` in `translation_definition` and the `description`
  item `buildDescribedBlock` already contributes
- `tasks/learnings.md` "The tree-sitter grammar must never be stricter than the Go parser" — items go
  into `buildDescribedBlock`, never behind `optional(...)`, and the exactly-one-of rule is the Go
  parser's alone
- `tasks/learnings.md` "New DSL keywords must stay usable as field names" and "Every `grammar.js` rule
  carries a one-line example of its full shape"
- `tasks/learnings.md` "Run repo tooling through `mise exec --`, not bare PATH" and "Generated
  tree-sitter `src/` stays gitignored"
- Highlighting queries are US-010's; this task changes no `.scm` file

**Testable:** Yes — the tree-sitter corpus is the test surface, run by `task test:grammar`.

**Verification:** `mise exec -- task test:grammar`; `git status --porcelain editors/tree-sitter-emod`
lists only `grammar.js` and corpus files.

**Depends on:** 1

---

### Task 4: Reject an `every` expression that is neither a duration nor a five-field cron

**Behavior:** `emod validate` reports an automation whose schedule expression is neither a Go duration
nor a five-field cron expression, positioned at the expression itself, with a message naming both
accepted forms and quoting what was written. Shape is what is checked: the numbers inside a well-formed
expression are not range-checked, because the model states a cadence that nothing here evaluates.

**Acceptance Criteria:**
- [ ] A model whose automation declares `every "5m"`, one declaring `every "1h"` and one declaring
      `every "1h30m"` validate with no diagnostics
- [ ] A model whose automation declares `every "0 2 * * *"`, one declaring `every "*/15 * * * *"` and
      one declaring `every "0 0 1,15 * 1-5"` validate with no diagnostics
- [ ] A model whose automation declares `every "nightly"` reports exactly one diagnostic whose message
      quotes `nightly` and names both accepted forms — the duration and the five-field cron
      expression — positioned at the line and column of the expression, not of the keyword or the
      automation's name
- [ ] An expression with four fields (`"0 2 * *"`) and one with six (`"0 2 * * * *"`) are each reported
      the same way, so a near-miss on the cron form is not silently accepted
- [ ] An expression that is neither (`"5 minutes"`) is reported, while `"5m"` in the same model is not —
      asserted on one model carrying both automations, so the diagnostic count and the offending
      expression are both pinned
- [ ] `"99 * * * *"` validates with no diagnostics, pinning that field values are not range-checked
- [ ] An automation stating `on` and no schedule produces no diagnostic from this check, asserted on a
      model that also carries a rejected expression so the check is proved to be running
- [ ] The printed diagnostic carries no `[rule]` bracket, and no rule name for it is resolvable through
      `emod lint --explain`: what no configuration can silence is a hard error, not a rule
- [ ] `cli.RunValidate` on a file with a malformed expression returns an error whose message names the
      offending expression and both accepted forms — the same distinguishing content as the validator
      test one layer down, not just a path and a line number
- [ ] Diagnostics from a model with two malformed expressions arrive in declaration order across both
      slice homes — an aggregate's slices before the slices a `mode dcb` context declares directly

**Affected Files/Modules:**
- `internal/validator/validator.go` — a check beside `referenceDiagnostics` (`:298`), reached from
  `Validate` (`:12-36`)
- `internal/validator/validator_test.go` — the automation group
- `internal/cli/validate_test.go` — one leaf beside `:85`

**Patterns to Follow:**
- Positioning and construction: `appendUndeclaredRef` (`internal/validator/validator.go:325`) reports at
  the value's stored position and carries no `RuleName`; `tasks/learnings.md` "`RuleName` marks a
  diagnostic `emod lint --explain` can describe"
- Ordering: `scopedInvariantDiagnostics` (`:283`) and `specDiagnostics` (`:339`) sort by
  `comparePositions`, and `tasks/learnings.md` "Diagnostics gathered from more than one AST collection
  must be position-sorted" — assert with one `require.Equal` against the reported lines
- `tasks/learnings.md` "CLI diagnostic tests must assert the distinguishing message text" and "A second
  `require.Contains` on one message is often shadowed by the first" — check a new needle is not already
  inside one asserted above it
- The accepted forms come from `docs/proposals/triggers-and-automations-proposal.md:105`: a Go duration
  for a fixed interval, a five-field cron expression for a wall-clock schedule

**Testable:** Yes — through `validator.Validate` and `cli.RunValidate`.

**Verification:** `mise exec -- go test -tags unit ./internal/validator/... ./internal/cli/...`.

**Depends on:** 1

---

### Task 5: Share a fixture whose automations activate on a schedule

**Behavior:** one shared model declares scheduled automations in both homes a slice has — nested in an
aggregate and directly on a `mode dcb` context — beside automations that state an activation event
instead, so every package downstream reads schedules and events out of the same source rather than
writing its own. The fixture is a model `emod validate` and `emod lint` both accept, and it survives a
format round-trip with every schedule intact.

**Acceptance Criteria:**
- [ ] `internal/test/fixtures.go` gains a fixture whose automations declare `every` in both slice homes,
      with at least one duration expression and one cron expression, and with an automation stating `on`
      and no schedule written *mid-block* ahead of a further automation — never as the last entry, so an
      entry running on into what follows it is caught
- [ ] `internal/test/models.go` gains the parsing accessor for it, in the shape of
      `AutomationReadsLibraryLendingModel` (`:37`)
- [ ] A hand-transcribed exported list names every schedule the fixture declares, both slice homes
      together and in declaration order, and is non-empty
- [ ] A `Declared…` getter walks `declaredSlices` (`internal/test/fixtures.go:893`) and returns the
      schedule every automation states, skipping the automations that state none, and
      `require.Equal` against the transcribed list holds — a getter reaching only one slice home reads
      back short
- [ ] `test.DeclaredActivationEvents` over the same fixture returns the events its event-activated
      automations declare, non-empty, so the fixture is proved to carry both activation forms
- [ ] `oracle.Check` over the fixture returns no diagnostics, added as a leaf in the "clean input"
      group (`internal/oracle/oracle_test.go:24`) — lexer, parser, validator and linter all accept it,
      and a `mode dcb` context in it carries the tags and `decides_on` its events need
- [ ] The formatter round-trip group gains this fixture in the existing per-fixture leaf
      (`internal/formatter/formatter_test.go:873`), asserting the reparsed model reads back the
      transcribed schedules and the transcribed activation events — one leaf, not a parallel table
- [ ] `git diff` leaves every existing fixture in `internal/test/fixtures.go` untouched: the models that
      state no schedule are this story's byte-identical witnesses and may not move

**Affected Files/Modules:**
- `internal/test/fixtures.go` — the fixture const, the transcribed list and the getter, beside
  `AutomationReadsLibraryLending` (`:578`), `…ViewNames` (`:758`) and `DeclaredAutomationReads` (`:866`)
- `internal/test/models.go` — the accessor (`:37` is the sibling)
- `internal/oracle/oracle_test.go` — one leaf in "clean input"
- `internal/formatter/formatter_test.go` — the per-fixture round-trip leaf (`:873`)

**Patterns to Follow:**
- `tasks/learnings.md` "A new optional field ships a six-part fixture kit, not a bespoke model per
  package" — `AutomationReadsLibraryLending` is the model to repeat, minus the `Without…` twin, which
  this feature cannot have (stripping a schedule leaves a model the parser rejects; the unfeatured
  witnesses are the existing fixtures)
- `tasks/learnings.md` "A slice has two homes, and much of the repo still walks only one" and
  "Exercise an omitted optional part mid-block, never as the last entry"
- `tasks/learnings.md` "A new shared fixture owes `internal/oracle` a zero-diagnostic subtest", with its
  warning that DCB shapes are the usual tripwire
- `tasks/learnings.md` "A `Declared…` getter answers `nil` for a fixture that declares none of the
  construct" — pair the getter only with the non-empty transcribed list, and fold the assertion into the
  existing round-trip leaf
- `declaredAutomationEntries` (`internal/test/fixtures.go:881`) already generalises the getter shape

**Testable:** Yes — through `oracle.Check`, `formatter.Format` and the exported getters.

**Verification:** `mise exec -- go test -tags unit ./internal/test/... ./internal/oracle/...
./internal/formatter/...`.

**Depends on:** 2, 4

---

### Task 6: Carry the schedule into the JSON, CUE and embedded schema exports

**Behavior:** `emod export` names an automation's schedule in both formats, and the embedded
`schema.cue` declares it, so a consumer of either document reads the cadence the author wrote. An
automation that states an activation event instead exports exactly what it exported before.

**Acceptance Criteria:**
- [ ] The JSON export of the scheduled fixture states every schedule the transcribed list names, read
      back with `statedUnder(exportedAutomations(doc), …)` in the writer's slice order, and states the
      transcribed activation events under their existing key in the same document
- [ ] The CUE export of the same fixture carries the same schedules, and the "CUE and JSON exports
      describe the same model" subtest (`internal/export/export_test.go:3441`) passes over it
- [ ] `emittedKeyOrder` shows the automation object emitting the schedule's position key among the
      other `*_position` keys and its value among the other values, matching the order a `json*` sibling
      uses — the sibling's key list asserted in the same subtest so the expectation is not arbitrary
- [ ] `internal/cue/schema.cue` declares the key on `#Automation`, and the schema-conformance subtest
      (`internal/export/export_test.go:3458`) passes for a model carrying schedules
- [ ] The embedded-schema fixture (`internal/cue/embed_test.go:126`) carries an automation stating a
      schedule and no activation event beside the automation that states one, and vets clean against
      `#Model`
- [ ] A document stating the schedule under a key `#Automation` does not declare fails `cue vet`, and
      the failure names that key — proving the definition is closed and the emitted spelling is the one
      the schema knows
- [ ] Exporting `test.AutomationReadsLibraryLendingModel(t)`, whose automations state no schedule,
      produces JSON and CUE containing the schedule key nowhere, while the scheduled fixture's exports
      contain it — both asserted in the same subtest

**Affected Files/Modules:**
- `internal/export/export.go` — `jsonAutomation` (`:158-173`), `convertAutomation` (`:598`),
  `cueWriter.writeAutomation` (`:1368`)
- `internal/cue/schema.cue` — `#Automation` (`:53-61`)
- `internal/export/export_test.go`, `internal/cue/embed_test.go`

**Patterns to Follow:**
- `tasks/learnings.md` "A new exported field must land in JSON, CUE and `schema.cue` in the same change"
  and "JSON and CUE order their document keys differently — do not mirror one struct into the other":
  take the Go struct's field order from a `json*` sibling, not from the schema
- `tasks/learnings.md` "JSON key order is assertable from the raw bytes — `emittedKeyOrder` already
  exists" and "Read a decoded export document back with `objectsUnder`/`statedUnder`, in the writer's
  slice order"
- `tasks/learnings.md` "The two export guards cannot see a list neither writer emits" — the parity and
  conformance subtests agree trivially about a key neither writer emits, so the read-back against the
  transcribed list is what proves the value arrived
- The retired-key negative leaf at `internal/cue/embed_test.go:112` is the shape for the unknown-key vet

**Testable:** Yes — through `export.ExportJSON`, `export.ExportCUE` and `cue vet`.

**Verification:** `mise exec -- go test -tags unit ./internal/export/... ./internal/cue/...`.

**Depends on:** 5

---

### Task 7: Carry the schedule through the diagram document and back

**Behavior:** an automation's schedule rides on its node in the diagram document and is read back when
that document is imported, so a scheduled automation exported from `emod`, edited in the viewer and
saved keeps its cadence. A scheduled automation draws no incoming activation edge, because it has no
source node.

**Acceptance Criteria:**
- [ ] The diagram document's automation node states the schedule the author wrote, read back for the
      scheduled fixture against the transcribed list, while the automations of the same document that
      state an activation event still carry it
- [ ] Importing a document whose automation node states a schedule yields a model whose automation
      carries it; importing the same document with the value moved to a key the importer does not read
      yields an automation with no schedule, so the reader is proved not to accept two spellings
- [ ] Exporting the scheduled fixture to a diagram document and importing it back yields schedules equal
      to the transcribed list, and formatting that reimported model produces the fixture's canonical
      bytes — the viewer's save path is export → import → format, so this is the round-trip that matters
- [ ] A scheduled automation contributes no `automation_trigger` edge to the document, while an
      event-activated automation in the same model still contributes one — both asserted on one export
- [ ] The exporter and the importer name the key identically, checked by the round-trip above rather
      than by reading the two struct tags: a mismatch drops the value silently on save

**Affected Files/Modules:**
- `internal/export/export.go` — `jsonDiagramNode` (`:653-673`) and the automation node build (`:829-846`)
- `internal/importer/importer.go` — `diagramNode` (`:25-42`) and `buildSlice` (`:206-215`)
- `internal/export/export_test.go`, `internal/importer/importer_test.go`

**Patterns to Follow:**
- `tasks/learnings.md` "The viewer's save path is `importer.ImportDiagram`, so a diagram-node field owes
  a read-back" — the guard is `importExported(t, model)` plus the canonical-source round-trip in
  `internal/importer/importer_test.go`
- `tasks/learnings.md` "A key rename owes a retired-key negative assertion on every surface that reads
  the key" — `documentKeying` (`internal/importer/importer_test.go:209-228`) is the closure shape for
  importing one document under two keyings
- `tasks/learnings.md` "A new exported field must land in JSON, CUE and `schema.cue` in the same change"
  records why `jsonDiagramEvent` forks `jsonEvent`; do not re-merge the diagram node with the model
  document's automation type
- The `automation_trigger` edge keeps its name and its guard (`internal/export/export.go:1077`)

**Testable:** Yes — through `export.ExportDiagramJSON` and `importer.ImportDiagram`.

**Verification:** `mise exec -- go test -tags unit ./internal/export/... ./internal/importer/...`.

**Depends on:** 6

---

### Task 8: Draw a clock badge on a scheduled automation in SVG and draw.io

**Behavior:** an automation activated by a schedule is drawn with a clock badge carrying its expression,
so a reader sees the cadence on the box rather than having to open the model. An automation activated
by an event is drawn exactly as it is today, in the same place, at the same size, with the same fill.

**Acceptance Criteria:**
- [ ] The SVG for a scheduled automation carries a clock marking and the expression text inside that
      automation's box, alongside the gear marking and the name it already carries
- [ ] The draw.io XML for the same automation carries the clock marking and the expression on that
      automation's cell, and the document still parses as well-formed XML
- [ ] In both formats, an automation stating an activation event and no schedule carries neither the
      clock marking nor any expression — asserted on one model declaring both automations, so the
      marked and unmarked cases are proved together
- [ ] Rendering `test.AutomationReadsLibraryLendingModel(t)`, whose automations state no schedule,
      produces output containing no clock marking, and `git diff` moves no expected constant in
      `internal/diagram/svg_test.go`, `drawio_test.go` or `contract_test.go` that describes an
      unscheduled model
- [ ] Every scheduled automation of the shared scheduled fixture appears in both renderings with its own
      expression, matched against the transcribed list rather than against a single hand-written string
- [ ] The badge changes no box's position, size or fill: the automation boxes of a model rendered before
      and after adding a schedule to one of them occupy the same coordinates, asserted through the
      geometry helpers `internal/diagram/contract_test.go` already uses

**Affected Files/Modules:**
- `internal/diagram/svg.go` — the automation box and label (`:149-164`)
- `internal/diagram/drawio.go` — the automation cell and label (`:404-416`)
- `internal/diagram/svg_test.go`, `internal/diagram/drawio_test.go`,
  `internal/diagram/contract_test.go`

**Patterns to Follow:**
- The existing markings are the precedent for a glyph in a label: `⚙` in both renderers
  (`svg.go:157`, `drawio.go:413`) and `formatEventTagBadges` (`drawio.go:866`) for a value rendered as a
  badge beside a name
- Tooltips already have a channel: `svgRoundedRect`'s last argument (`internal/diagram/svg.go`) and
  `vertexCell`'s `tooltip` (`internal/diagram/drawio.go`), both currently carrying the description
- `tasks/learnings.md` "Additive output changes owe a byte-identical receipt for models that do not use
  the feature", and "A differential receipt must first prove the twin actually differs" — assert the
  scheduled model renders a badge before asserting the unscheduled one renders none
- Lane placement, the palette and the `reads` edge belong to US-005 through US-007; this task moves no
  box and repaints nothing

**Testable:** Yes — through `diagram.ExportSVG` and `diagram.ExportDrawio`.

**Verification:** `mise exec -- go test -tags unit ./internal/diagram/...`.

**Depends on:** 5

---

### Task 9: Name the cadence in Mermaid, ASCII and `emod slices`

**Behavior:** the three surfaces that summarise an automation as a line of text read correctly for a
scheduled automation: Mermaid names the cadence where it names an activation event, the ASCII chain
starts from the cadence rather than from an empty pair of parentheses, and `emod slices` lists the
cadence and the command rather than a row starting with a comma.

**Acceptance Criteria:**
- [ ] The Mermaid output for a scheduled automation names the expression and the command it issues; the
      output for an event-activated automation in the same model is unchanged from what it produces
      today, asserted against its existing expected line
- [ ] The ASCII output for a scheduled automation names the expression, the automation and its command
      in one chain, with no empty source, and the chain for an event-activated automation in the same
      model is unchanged
- [ ] Every rune of the ASCII output for the scheduled fixture is ASCII apart from `⚙` — the existing
      assertion at `internal/diagram/ascii_test.go:319` passes unedited, so no clock glyph reaches this
      format
- [ ] An event declared in a slice whose automation is scheduled is still listed among that slice's
      events in ASCII output — the standalone-event collection (`internal/diagram/ascii.go:154`) keys on
      the activation event, and an automation with none must not suppress anything
- [ ] `emod slices` prints the cadence and the command for a slice whose automation is scheduled, and
      the activation event and the command for one whose automation states an event — both asserted on
      one file, so neither row is checked in isolation
- [ ] The four diagram formats and the slice listing all still produce their current output for
      `test.AutomationReadsLibraryLendingModel(t)`: `git diff` moves no expected constant describing a
      model without schedules

**Affected Files/Modules:**
- `internal/diagram/mermaid.go` — the three automation label builders (`:114-127`, `:236-250`,
  `:379-393`)
- `internal/diagram/ascii.go` — the automation chain (`:84-87`) and `standaloneEvents` (`:143-163`)
- `internal/cli/slices.go` — `keyElementsForPattern` (`:170-176`)
- `internal/diagram/mermaid_test.go`, `internal/diagram/ascii_test.go`, `internal/cli/slices_test.go`

**Patterns to Follow:**
- The three Mermaid label builders are the same six lines three times; `tasks/learnings.md`
  "De-duplicate before a fan-out edit, and land the de-duplication with proof" applies if they are
  collapsed — carry a differential receipt and keep the extraction's parameters in line with the
  callers
- `tasks/learnings.md` "An assertion whose expected value comes from the code under test is the
  recurring review finding" — name the expected line, do not rebuild it from the renderer
- Mermaid's timeframe letters (`ui` versus `pcr`) are US-006's; this task changes no timeframe
- `internal/cli/slices_test.go` asserts printed rows; `tasks/learnings.md` "urfave/cli v2 discards every
  flag written after the file argument" governs any invocation written with a flag after the path

**Testable:** Yes — through `diagram.ExportMermaid`, `diagram.ExportASCII` and `cli.RunSlices`.

**Verification:** `mise exec -- go test -tags unit ./internal/diagram/... ./internal/cli/...`.

**Depends on:** 8

---

### Task 10: Show a scheduled automation's cadence in the viewer

**Behavior:** the web viewer draws a clock badge carrying the expression on a scheduled automation's
box, with the expression also reachable as a tooltip, and lists the cadence in the details panel beside
the activation event. An automation activated by an event looks exactly as it does today.

**Acceptance Criteria:**
- [ ] A rendered automation node whose data states a schedule carries a clock marking and the expression
      text within that node's group in the produced SVG
- [ ] The same node carries the expression as a tooltip, so a long cron expression is readable when the
      box is too small for it
- [ ] An automation node stating no schedule renders with neither the marking nor the tooltip, and its
      box keeps the position, size and fill it has today — asserted in the same test as the scheduled
      node so both cases move together
- [ ] The details panel shows a row for the schedule beside the existing `On Event`, `Reads`, `Command`
      and `Target Context` rows, in a fixed position, listing the expression for a scheduled automation
      and the em-dash placeholder for one that states none
- [ ] A schedule value containing markup is shown as text, matching the escaping the other rows already
      prove (`internal/viewer/tests/detail-panel.test.js:73-82`)
- [ ] A node stating the schedule under a key the panel does not read shows the placeholder, so the
      panel is proved to read the same key the exporter writes
- [ ] `mise exec -- task test:viewer` passes, and `mise exec -- task test:unit` is unaffected — this
      task changes no Go file

**Affected Files/Modules:**
- `internal/viewer/static/renderer.js` — the node drawing loop (`:206-272`)
- `internal/viewer/static/ui.js` — the automation section of the details panel (`:351-361`)
- `internal/viewer/tests/renderer.test.js`, `internal/viewer/tests/detail-panel.test.js`

**Patterns to Follow:**
- The translation branch (`internal/viewer/static/renderer.js:258-267`) is the only node that draws more
  than a centred label and is the shape for a second line of text on a box
- `tasks/learnings.md` "`internal/viewer/static` is a display surface with its own vitest harness" — the
  jsdom SVG shim (`installSVGGeometry`) and the dynamic `await import` spelling are required for any
  module touching geometry, and restructuring `showDetailPanel` beyond the added row belongs in its own
  commit
- `tasks/learnings.md` "A diagram-node key has three readers, and they must move in one commit" — the
  exporter and importer moved in Task 7; this is the third reader, and a placeholder shown for a
  correctly-keyed node is the regression it exists to prevent
- The existing automation rows in `internal/viewer/static/ui.js:351-361` fix the row order and the
  placeholder character

**Testable:** Yes — through the vitest harness under `internal/viewer/tests`.

**Verification:** `mise exec -- task test:viewer`; `git status --porcelain -- '*.go'` is empty.

**Depends on:** 7

---

## Summary

**Ten tasks**, ordered dependency-first and, within that, by how much of the repo each unblocks.

Task 1 is the language change and the only one every other task rests on; it also carries the exactly-
one-of rule, because adding `every` without rewriting the "requires an activation event" check would
make the first scheduled automation ever written report an error. Tasks 2, 3 and 4 fan out from it and
are independent of each other: the formatter comes first of the three so the window in which `emod fmt`
would silently delete a schedule is one commit long. Task 5 turns the language change into a shared
model and is what Tasks 6 through 9 read; Task 6 must precede Task 7 because both write
`internal/export/export.go`, and Task 8 must precede Task 9 for the same reason inside
`internal/diagram`. Task 10 comes last because the viewer reads the diagram-document key Task 7
establishes, and showing a placeholder for a correctly-keyed node is exactly the regression the
learnings record from US-002.

**Story criteria coverage:**

| Story criterion | Task |
|---|---|
| An `automation` accepts `every "<expr>"`, a duration or a five-field cron expression | 1 (accepted), 4 (the two forms) |
| An expression matching neither form is a validation error naming both accepted forms | 4 |
| An automation declaring both `on` and `every` is a validation error | 1 |
| An automation declaring neither is a validation error naming both as options | 1 |
| `emod fmt` emits `every` at a fixed position, and JSON and CUE exports carry it | 2, 6 |
| A scheduled automation renders with a clock badge carrying its expression | 8 (SVG, draw.io), 10 (viewer) |

Carried along, not stated by the story: the tree-sitter grammar (3), the shared fixture (5), the diagram
document and the viewer's save path (7), and the three text surfaces that misread an automation with no
activation event (9).

**Deferred to later stories in the feature:** the `reads` edge (US-005), lane placement (US-006), the
palette (US-007), the `automation/missing-todo-list` rule (US-008), LSP hover and completion over
`every` (US-009), VS Code and tree-sitter highlighting queries (US-010), and every document and example
in the repository, including the scheduled automation `docs/dsl-reference.md` and `examples/` will show
(US-011).

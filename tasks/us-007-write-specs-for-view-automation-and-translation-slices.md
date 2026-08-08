# US-007: Write specs for view, automation, and translation slices

## Progress
- [x] Task 1: Parse `then view <ViewName>` and `then command <CommandName>`
- [x] Task 2: Share a fixture stating a spec for every slice pattern
- [x] Task 3: Resolve a spec's view and command outcomes against the model
- [x] Task 4: Reject a `then` shape the enclosing slice cannot state
- [ ] Task 5: Preserve the view and command outcomes through `emod fmt`
- [ ] Task 6: Carry the two outcomes through the JSON and CUE exports and the embedded schema
- [ ] Task 7: Accept the two outcomes in the tree-sitter grammar
- [ ] Task 8: Document the four spec shapes in the DSL reference

---

## Story Reference

`user-stories/specs-and-metadata.md` → **US-007: Write specs for view, automation, and translation
slices** (seventh story of "Specs, Invariants, and Model Metadata", lines 92-103). Design notes:
`docs/proposals/specs-and-metadata-proposal.md` §4 "Pattern variants" (lines 159-203), the validation
list at lines 235-243, the AST note at line 358 (`ThenClause is one of: ThenEvents, ThenRejected,
ThenView, ThenCommand`), the parser note at line 401 (`then` disambiguates on its first token), and
the worked example at lines 537-575.

**In scope:** two further shapes for a spec's `then` entry — `then view <ViewName>` and
`then command <CommandName>` — so that all four slice patterns can state their expected behaviour;
acceptance of a spec that omits `when` in a view slice and in a schedule-driven automation slice, and
of a `when` naming an event in an event-driven automation slice; a translation slice's spec in the
given/when/then-events form; a validation error when a `view` or `command` outcome names a construct
the model does not declare; a validation error when a `then` shape does not match the pattern of the
slice that states it. Carried with them, because the repo's writers would otherwise silently drop the
new shapes: `emod fmt` round-trip, the JSON and CUE exports plus the embedded schema, the tree-sitter
grammar (which must never reject what `emod validate` accepts), and the DSL reference.

**Out of scope:** the four `spec/*` lint rules — `spec/command-without-spec`,
`spec/no-rejection-path`, `spec/invariant-never-exercised`, `spec/given-outside-boundary` — so this
story adds no rule to `internal/linter` and no entry to `ruleDescriptions`
(`internal/linter/descriptions.go`) (US-008); flow rejection edges
`command -> rejected: <Cmd> -> <inv>` (US-009); example payload literals `{ field: value }` on
element references, which is what gives `SpecElement` a `Payload` and gives `then view` its
long-deferred view-state question (US-010); `given` / `when` / `then` alignment inside a formatted
spec and the wider canonical attribute order, so the formatter here writes one canonical line per
entry and leaves column alignment alone (US-014); LSP hover on a `view`/`command` outcome, completion
of view and command names after `then`, and go-to-definition from an outcome (US-015) — Task 8 moves
one existing hover *description string* that this story makes inaccurate and nothing else in
`internal/lsp`; rendering specs on diagrams behind `--specs` (US-016); syntax highlighting in
`editors/vscode/syntaxes/emod.tmLanguage.json` and `editors/tree-sitter-emod/queries/highlights.scm`
(US-017) — no work falls out there anyway, see "Codebase Context"; specs in `examples/*.emod` and the
wider reference sweep (US-018).

**Open questions, decided.** Four shapes the story does not pin down, each decided so that US-008 and
US-010 stay additive:

1. **Slice pattern is read off the constructs the slice declares, one requirement per outcome kind —
   there is no shared classifier with `emod slices list`.** The rule is:

   | outcome | the enclosing slice must declare |
   |---|---|
   | `then [Events]` (including `then []`) | at least one `command` **or** at least one `translation` |
   | `then rejected <invariantName>` | at least one `command` |
   | `then view <ViewName>` | at least one `view` |
   | `then command <CommandName>` | at least one `automation` |

   `detectPattern` (`internal/cli/slices_list.go:109-123`) is the repo's existing classifier and
   cannot serve here for three reasons. It lives in `internal/cli`, which reaches the validator
   through `internal/oracle`, so the validator cannot import it back. Its `command` arm requires a
   `trigger`, which `docs/dsl-reference.md:307` documents as optional — verified,
   `emod slices list` over `test.SpecLibraryLending` reports `unknown` for "Return Copy", "Claim Desk"
   and "Release Desk", three of the four spec-carrying command slices, so classifying with it would
   report every one of their existing `then` outcomes as mismatched. And its priority order
   (translation, then automation, then view, then command) gives a slice exactly one pattern, so a
   slice legitimately declaring both a command and an automation — `test.AutomationScheduleLibraryLending`'s
   "Chase Overdue Copy" declares two commands and three automations — could state only one of the two
   outcome shapes. Widening `detectPattern` would change what `emod slices list` prints, which is a
   separate behaviour change this story does not own. The cost of the element rule is that a slice
   genuinely declaring two patterns' constructs accepts both patterns' outcomes; the story's own
   example of the error, a `view` outcome inside a command slice, is still caught.
2. **This story adds no rule about `when` being present or absent.** A spec may omit `when` in any
   slice and may state one in any slice; where it states one, US-006's existing rule stands — the name
   must resolve against the model's commands *or* its events (`declaresCommandOrEvent`,
   `internal/validator/validator.go:136-138`). "No `when`" is genuinely ambiguous between a view spec
   and a schedule-driven automation spec, and requiring `when` for the event-driven automation shape
   would mean deciding from the AST which of an automation's two activation forms a spec was written
   against — a slice may declare several automations, some on `on` and some on `every`. The `then`
   shape carries the pattern, and the `then` shape is what this story judges. The story's first four
   criteria are therefore satisfied by the two new outcomes plus acceptance the parser already gives.
3. **A `then` shape diagnostic is positioned on the outcome's own name** — the view name, the command
   name, the invariant name, or the first event of a success list — and on the spec's name when the
   outcome is `then []`, which names nothing. No position field is added to `ast.Spec` or to any
   `ThenClause` variant, so no new `*_position` key reaches the exports.
4. **A `view` and a `command` outcome resolve model-wide and unqualified**, the same rule US-006 gave
   `given`, `when` and `then` event lists: a view declared in another aggregate or another context
   resolves. `tasks/learnings.md` records that a *trigger's* and a *translation's* `reads` must stay
   unchecked because shipped models name views nothing declares — that exemption does not reach here,
   because a `then view` outcome is new syntax no existing model can already contain.

**Overarching constraint:** every existing `.emod` file stays valid with unchanged meaning. That is
load-bearing in three places here — a spec that states an event list or a rejection must validate
exactly as before, including in the three trigger-less command slices of `test.SpecLibraryLending`;
`emod fmt`, the exports and the diagrams must produce their current bytes for every model that states
no `view` or `command` outcome; and no expected value that transcribes `test.SpecLibraryLending` may
move, since this story adds a sibling fixture rather than editing it.

**Learnings folded in** from `tasks/learnings.md`: `ast.ThenClause` dispatches through type switches
that fail silently on a variant they have not heard of, and the parse → format → reparse comparison
is the only thing that notices the formatter dropping one; a spec's `when` resolves against commands
*and* events while `given`/`then` resolve against events only, and the two lookups must stay
different; a bracketed list's terminator set decides whether one typo cascades; a new block entry
keyword owes three things to the parser's diagnostics, including the `require.Len(t, diags, 1)` pin;
a line-oriented declaration must gate every optional trailing token on the first token's line; put a
new parser subtest in the group that owns the construct; assert a short keyword in a diagnostic with
a `\b`-bounded `require.Regexp`; a second `require.Contains` on one message is often shadowed by the
first, so validator leaves compare whole formatted lines; diagnostics gathered from more than one AST
collection must be position-sorted; `RuleName` marks a diagnostic `emod lint --explain` can describe,
so a hard error carries none; CLI diagnostic tests must assert the distinguishing message text; a
second construct gaining the same field extends the fixture kit sideways with independent
transcriptions; a new shared fixture owes `internal/oracle` a zero-diagnostic subtest, and a lint
warning fails `emod validate` so the fixture has to keep all fifteen rules quiet; a slice has two
homes and a fixture declaring the construct in only one cannot catch a short walk; exercise an
omitted optional part mid-block, never as the last entry; a `Declared…` getter answers `nil` for a
fixture declaring none of the construct, so pair it with a non-empty transcription; a differential
receipt must first prove the twin actually differs, and `require.NotEqual` on a stripped twin is
satisfiable without stripping anything; `emod fmt` canonicalises order so a fmt golden is never the
input re-indented, and formatter output always begins with `emod N`; never write emod source with
`%q`; a new exported field must land in JSON, CUE and `schema.cue` in the same change; JSON and CUE
order their document keys differently; a serialized key spells the DSL keyword; JSON key order is
assertable with `emittedKeyOrder`; the two export guards cannot see a list neither writer emits; the
tree-sitter grammar must never be stricter than the Go parser; every `grammar.js` rule carries a
one-line example of its full shape, and generated `src/` stays gitignored while tooling runs through
`mise exec --`; `docs/dsl-reference.md` numbered *and* sub-heading anchors are cited across the
document; an ```emod fence is a promise that the block validates; a task's change-set assertion must
name every file its own patterns require it to change; a tested improvement found on the way is still
a separate commit; and acceptance criteria never reference commit, branch or remote state.

---

## Codebase Context

**No new keywords.** This is the single largest difference from US-006. `view` and `command` are
already `lexer.KeywordView` and `lexer.KeywordCommand` (`internal/lexer/token.go:17`, `:22`), so
`lexer.Keywords()` does not grow, the four keyword-coverage tests that iterate it
(`internal/lexer/tokenizer_test.go:14`, `internal/parser/parser_test.go:224` and `:243`,
`internal/oracle/oracle_test.go:46`) are already satisfied, `TestKeywordCoverage`
(`internal/lsp/keywords_test.go:18`) already has hover text and a completion entry for both, both
words are already in the tree-sitter `highlights.scm` keyword alternation and in the VS Code
TextMate alternation, and both are already exercised as field names by
`editors/tree-sitter-emod/test/corpus/fields.txt:14` and `:19`. Nothing in the keyword fan-out
`tasks/learnings.md` describes applies to this story.

**AST.** `internal/ast/ast.go:95-126` holds the US-006 shape: `Spec` (comments, name, `Given
[]*SpecElement`, `When *SpecElement`, `Then ThenClause`, open and close positions), `SpecElement`
(name plus position — US-010 hangs a payload here), and the sealed `ThenClause` interface with marker
method `thenNode()` and two variants, `ThenEvents{Events []*SpecElement}` and
`ThenRejected{InvariantName, InvariantPos}`. The proposal names the two this story adds: `ThenView`
and `ThenCommand` (`docs/proposals/specs-and-metadata-proposal.md:358`).

**Six sites dispatch on `ThenClause`, and every one of them fails silently on a variant it has not
heard of.** `parseSpecOutcome` (`internal/parser/parser.go:498-520`, `default:` reports "expected an
event list or rejected after then in spec"); `formatOutcome` (`internal/formatter/formatter.go:387-399`,
`default: return ""` — `writeSpec` at `:381` then omits the whole `then` line, so `emod fmt` *deletes*
the outcome); `convertSpecOutcome` (`internal/export/json.go:437-448`, returns nil — the JSON omits
`then`); `cueWriter.writeSpecOutcome` (`internal/export/cue.go:144-150`); and two bare type assertions
in the validator, `spec.Then.(*ast.ThenRejected)` (`internal/validator/validator.go:255`) and
`spec.Then.(*ast.ThenEvents)` (`:346`). There is no `.golangci.yml`, so no exhaustiveness check backs
any of it, and only the parse → format → reparse comparison notices the formatter.

**Parser.** `parseSpec` (`internal/parser/parser.go:431-484`) is an unbounded
`for !p.check(lexer.CloseBrace)` loop over a three-arm switch with a `default:` reporting "expected
given, when or then in spec". `parseSpecOutcome` (`:498-520`) is the arm this story extends: it
switches on the token after `then`, taking `[` for an event list and `lexer.KeywordRejected` for a
rejection, each error path draining with `p.skipRestOfLineOrBlockEnd`. `parseSpecCommand`
(`:486-496`) is the closest shape for a keyword-then-identifier outcome, and
`checkIdentifierLikeSameLineAs` (`:1514`, used at `:508`) is the line gate. Views and commands are *declared*
with `lexer.Identifier` (`parseView:994-1000`, `parseCommand:643-649`), unlike invariants, which are
declared identifier-*like* (`parseInvariant:333-341`) — a reference should match how the thing it names is declared.
`parseSpecEventList` (`:522-545`) and `atSpecEventListEnd` (`:547-549`) show the terminator set that
keeps one unclosed bracket from cascading.

**Parser tests.** `internal/parser/parser_test.go` is one umbrella of thirteen top-level groups;
`"contexts, aggregates and slices"` (`:568`) owns the slice and the spec, `"error reporting"`
(`:2671`) owns recovery and message shapes. `requireSpecCoverage` (`:5699-5728`) is US-006's
fixture-coverage guard: it reads each spec back off the fixture's *source* as well as off the parsed
model (because `given []` and an omitted `given` leave the same parsed spec) and asserts six shapes
are present — a multi-event `given`, an empty `given []`, an omitted `given`, an event-list outcome, a
rejection outcome, and a `then` written above its `when`. `outcomeKind` and `specSourceLines`
(`:5730`) sit beside it, `requireASpecAheadOfALaterEntry` (`:5738`) is the mid-block guard, and both
are called from the fixture leaf at `:1249-1266`. The receipts that a parser drain stopped in the
right place are the non-zero `ClosePos.Line` reads at `:2889-2891` and `:2948-2950`.

**Validator.** `referenceDiagnostics(slice, index)` (`internal/validator/validator.go:297-323`) is the
one walk that already has the *slice* in hand, which is what the shape rule needs; it funnels
existing checks through `appendUndeclaredRef(diags, kind, name, pos, declared)` (`:325-331`), a
one-liner carrying the empty-name guard, and ends by calling `specDiagnostics(spec, index)` (`:341-357`)
per spec. `modelIndex` (`internal/validator/validator.go:43-81`) carries `commandNames`, `eventNames`,
`viewNames` and `contextNames`, all model-wide and unqualified; `viewNames` is filled in `collect`
(`:96-98`) from both slice homes via `model.AllSlices()`. `errorAt` (`:150-158`) sets
`Severity: diagnostic.Error` and leaves `RuleName` empty. `sortInDeclarationOrder` (`:193-201`) is
what keeps a multi-diagnostic result stable. `unresolvedRejections` (`:246-264`) is the scope-aware
walk US-006 Task 4 added for `then rejected`.

**Validator tests.** `internal/validator/validator_test.go` groups by rule — `"spec references"` is US-006's group
(`:1267`) and `"duplicate invariant declarations"` US-005's (`:2293`) — and asserts a multi-diagnostic
result with one `require.Equal` against `reportedLines(diags)`, whole formatted lines and all
(`:2473-2482`, `:1145-1147`).

**Formatter.** `writeSlice` (`internal/formatter/formatter.go:156-206`) emits a slice's entries in a
fixed kind order with specs last; `writeSpec` (`:372-385`) writes `given`, then `when`, then the
outcome, each as one canonical line, and leaves the line out entirely when the part is absent;
`formatOutcome` (`:387-399`) renders the outcome; `bracketed`/`specElementNames` (`:401`) build the
list; `quoted` (`:47-49`) is the only correct way to emit a string. `internal/formatter/formatter_test.go`'s
`"round-trip through the parser"` group (`:702`) keeps one leaf per fixture, `:876` being
`test.SpecLibraryLending`'s. `internal/cli/fmt_test.go` pins canonical constants —
`specFormattedEmod` (`:244`), `keywordFieldFormattedEmod` (`:135`) — and feeds them to
`requireFmtSettlesOn` (`:629`), used at `:578-581`.

**Exports.** `jsonSpec` and `jsonSpecOutcome` (`internal/export/json.go:189-205`) are the document
types; `convertSpec`/`convertSpecOutcome` and the `specElementNames`/`specElementName`/
`specElementPosition` helpers sit at `:416-472`. The CUE side is `cueWriter.writeSpec` and
`writeSpecOutcome` (`internal/export/cue.go:134-150`), built on `writeCUEList`, `listIfSet`,
`lineIfSet` and `writeObject`. `internal/cue/schema.cue` mirrors them — `#SpecOutcome` at `:74-77`
(`events?`, `rejected?`), `#Spec` at `:79-85`, `#Slice`'s `specs?` at `:99` — and is what
`emod schema` prints. The coupling subtests: `emittedKeyOrder(t, raw)`
(`internal/export/export_test.go:4755`, used at `:1092` and `:1335`) reads emitted key order straight
off the bytes; the whole-spec position leaf at `:1496-1510` `require.Equal`s one spec's complete
exported map including `when_position` and the outcome's `rejected_position`; `specsByOwner`
(`:4423`) reads specs back per slice against the transcribed `libraryLendingSpecs` (`:4314-4363`),
built on the generic walkers `exportedSlices`, `statedUnder` and `objectsUnder` (`:4634-4676`);
`requireConformsToSchema` runs `cue vet -d '#Model'` (`:3813`) and `requireBothFormatsAgree` decodes
both exports and compares (`:3842`). The diagram JSON document is deliberately forked and carries no
spec at all (`:3180-3190`).

**Diagrams and glossary.** `internal/diagram/contract_test.go:441-453` `"stating specs leaves the
picture untouched"` is the four-renderer differential to copy: `require.NotEqual` on the two models,
then `require.Equal` of the transcribed spec names against the stated model *and* `require.Empty` on
the twin, then equality of the two renderings. `internal/glossary/glossary_test.go:587-594` is the
same receipt for the vocabulary. Neither package needs production work here — a spec is a scenario,
not an element and not a term.

**Fixtures.** `internal/test/fixtures.go` holds `HotelReservation`, `DescribedHotelReservation`,
`KeywordFieldSearchCatalog`, `InvariantLibraryLending` (`:314`), `SpecLibraryLending` (`:423-572`),
`AutomationReadsLibraryLending` (`:581`), `TriggerReadsLibraryLending` (`:749`) and
`AutomationScheduleLibraryLending` (`:947`). `SpecLibraryLending` states seven specs across four
command slices in both slice homes, all of them event-list or rejection outcomes; its one view slice,
"Review Member Loans", states none. `SpecLibraryLendingSpecNames` (`:1115-1127`) transcribes the
names, `WithoutSpecs` (`:1216-1226`) is the generic twin built on `copyWithEditedSlices` +
`editedCopies`, and `DeclaredSpecNames` (`:1299-1311`) the getter that walks `declaredSlices`.
`TriggerReadsLibraryLending` is the precedent for a *second* fixture covering an old construct and a
new one side by side, with its own transcriptions rather than an edit to the earlier model. Six
expected values downstream restate `SpecLibraryLending` line for line —
`internal/cli/fmt_test.go:244`, `internal/export/export_test.go:4314`,
`internal/test/fixtures.go:1119`, plus the leaves at `internal/diagram/contract_test.go:441`,
`internal/glossary/glossary_test.go:587` and `internal/formatter/formatter_test.go:877` — which is
why this story adds a sibling fixture instead of editing it.

**Tree-sitter.** `editors/tree-sitter-emod/grammar.js` sets `word: $ => $.identifier` (`:15`), so
lowercase keywords are plain anonymous tokens. `spec_definition` (`:107-113`) is a bare
`'{' repeat(choice($.spec_given, $.spec_when, $.spec_then)) '}'`, and `spec_then` (`:125-131`) is
`seq('then', choice($.spec_event_list, seq('rejected', $.any_identifier)))`. The precedent for the two
new arms is in the same file: `automation_definition` (`:301-312`) and `translation_definition`
(`:315-326`) both spell `seq('command', $.any_identifier)` inside a block body. `test/corpus/specs.txt`
holds five cases today ("Two spec blocks in one slice", "A spec before and after a flow", "Given with
several events, empty, and omitted", "Spec entries written out of canonical order", "Spec block in a
DCB context slice"); `test/corpus/fields.txt` already declares fields named `command` (`:14`), `view`
(`:19`), `spec`, `given`, `when`, `then` and `rejected` (`:40-44`). `src/` is gitignored;
`task test:grammar` regenerates before running the corpus.

**Reference.** `docs/dsl-reference.md` §5's slice skeleton (`:255-272`) already lists
`spec "<name>" { ... }`; §6 "Slice Patterns" holds the four pattern sections and, after them, the
`### spec` subsection (`:376-422`). Its closing line (`:422`) states outright that "The spec shapes
for view, automation, and translation slices — `then view <Name>` and `then command <Name>` — are not
part of the language yet", which this story makes false. §11 "Cross-References" (`:634-655`) has the
referencing-sites table (`:638-645`) and the validation bullet list (`:649-655`). The fenced blocks
in the `### spec` subsection are plain ``` fences, not ```emod, so `internal/oracle`'s "documented
models" leaf does not extract them — a fence upgraded to ```emod becomes a promise the block is a
whole valid model.

**Not touched, deliberately.** `internal/linter` and `internal/linter/descriptions.go` (US-008 owns
every `spec/*` rule; a warning-level rule fails `emod validate` for every model exhibiting the shape,
which is a sweep this story must not start); `internal/glossary` production code; `internal/cli/slices_list.go`,
whose `detectPattern` is discussed above and whose output must not move; `internal/importer` and
`internal/wasm/pipeline.go`, both of which move the diagram JSON document that carries no spec;
`internal/viewer`; `internal/lsp` apart from the one description string Task 8 corrects;
`editors/vscode/syntaxes/emod.tmLanguage.json` and `editors/tree-sitter-emod/queries/highlights.scm`,
which already know both keywords; `examples/*.emod` and `internal/parser/testdata/*.emod`, neither of
which declares a spec today; `e2e/tests`.

---

## Tasks

### Task 1: Parse `then view <ViewName>` and `then command <CommandName>`

**Behavior:** A spec's `then` entry accepts two further shapes beside an event list and a rejection.
`then view <ViewName>` records the view a view slice's spec concludes with; `then command <CommandName>`
records the command an automation slice's spec issues. Each records the name with its own position,
as an outcome distinguishable from the other three without inspecting the name. The parser
disambiguates on the single token after `then`, so no lookahead beyond it is needed and no new keyword
is introduced — `view` and `command` are keywords already. A malformed outcome reports exactly one
diagnostic and does not consume the entry on the following line.

**Acceptance Criteria:**
- [ ] A spec whose `then` reads `view MemberLoansView` parses with no diagnostics and records the view
      name with its position, as an outcome distinguishable from the three others without inspecting
      the name
- [ ] A spec whose `then` reads `command RecallCopy` parses with no diagnostics and records the
      command name with its position, likewise distinguishable
- [ ] Both outcomes parse identically in a slice nested in an aggregate and in a slice declared
      directly on a `mode dcb` context
- [ ] A spec stating a `view` outcome and no `when` at all parses with no diagnostics, and so does one
      stating a `command` outcome and no `when` — the two shapes a view slice and a schedule-driven
      automation slice take
- [ ] A spec whose `when` names an event and whose `then` states a `command` outcome parses with no
      diagnostics — the event-driven automation shape
- [ ] `then view` naming nothing, with a further entry on the following line, reports exactly one
      diagnostic (`require.Len(t, diags, 1)`) and the entry on the following line is still parsed onto
      the spec; the same holds for `then command` naming nothing
- [ ] A name written on the line *below* `then view` is not taken as the outcome — the outcome's name
      is gated on the line of the token that introduces it
- [ ] `then` followed by none of `[`, `rejected`, `view` or `command` reports exactly one diagnostic
      whose message names all four outcomes a `then` accepts, and the enclosing spec block still
      closes: the spec's `ClosePos.Line`, its slice's and its context's are all non-zero
- [ ] The message reported for an unrecognised `then` outcome names each of `view` and `command` under
      a word-boundary match — a `\b`-bounded `require.Regexp`, not `require.Contains` — so the
      assertion cannot be satisfied by words the sentence already contains
- [ ] Every spec shape US-006 accepted parses byte-for-byte identically: no existing subtest in
      `internal/parser/parser_test.go` needs editing, and `oracle.Check` over `test.HotelReservation`,
      `test.DescribedHotelReservation`, `test.KeywordFieldSearchCatalog`, `test.InvariantLibraryLending`
      and `test.SpecLibraryLending` still returns no diagnostics
- [ ] `lexer.Keywords()` is unchanged in length and content, and a `fields` block declaring fields
      named `view` and `command` still parses them as ordinary fields
- [ ] `go build ./...` succeeds with the two existing type assertions in `internal/validator` and the
      three type switches in the formatter and the two exporters left exactly as they are — this task
      adds the two variants and the two parse arms and nothing else

**Affected Files/Modules:**
- `internal/ast/ast.go` — the two outcome variants beside `ThenEvents` and `ThenRejected` (`:111-126`)
- `internal/parser/parser.go` — `parseSpecOutcome` (`:498-520`) gains two arms and a wider `default:`
  message
- `internal/parser/parser_test.go` — subtests in `"contexts, aggregates and slices"` for the shapes,
  and in `"error reporting"` for the recovery and message shapes

**Patterns to Follow:**
- The outcome switch this task extends rather than forks: `parseSpecOutcome`
  (`internal/parser/parser.go:498-520`), whose `rejected` arm is the exact shape both new arms take
- A keyword followed by an identifier on the same line, with a single-diagnostic drain:
  `parseSpecCommand` (`internal/parser/parser.go:486-496`) and `checkIdentifierLikeSameLineAs` (`:1514`) +
  `skipRestOfLineOrBlockEnd` (`:1526`) as used together at `:506-512` — `tasks/learnings.md` "A line-oriented declaration
  must gate every optional trailing token on the first token's line" and "A new block entry keyword
  owes three things to the parser's diagnostics"
- A reference matches how the thing it names is declared: views and commands are declared with
  `lexer.Identifier` (`parseView`, `internal/parser/parser.go:997`; `parseCommand`, `:646`), unlike an
  invariant, which is declared identifier-like (`parseInvariant`, `:337`)
- Node shape and naming: `docs/proposals/specs-and-metadata-proposal.md:358` names the two variants;
  `ast.ThenRejected` (`internal/ast/ast.go:121-126`) is the name-plus-position convention, and
  US-010 adds a payload to every `SpecElement`, so an outcome that names a *construct* rather than an
  event keeps its own field rather than reusing `SpecElement`
- Reading the block's close positions back is what proves a drain stopped in the right place —
  `tasks/learnings.md` "Retiring a keyword needs its own parser arm", with
  `internal/parser/parser_test.go:2889-2891` and `:2948-2950` as the existing spec-level receipts
- Short-keyword assertions use `\b`-bounded regexps — `tasks/learnings.md` "Assert a short keyword in
  a diagnostic with a `\b`-bounded `require.Regexp`"; `view` and `command` both sit inside other words
  of the message under test
- Subtests belong to the group that owns the construct — `tasks/learnings.md` "Put a new parser
  subtest in the group that owns the construct"

**Testable:** Yes — through `lexer.Scan` + `parser.Parse` and `oracle.Check`, all exported.

**Verification:** `go test -tags unit ./internal/lexer/... ./internal/parser/... ./internal/oracle/...`;
`go build ./...`.

**Depends on:** None

---

### Task 2: Share a fixture stating a spec for every slice pattern

**Behavior:** `internal/test` gains a model that states a spec for each of the four slice patterns —
a command slice with both a success and a rejection outcome, a view slice concluding in
`then view`, an event-driven automation slice whose `when` names its activation event and which
concludes in `then command`, a schedule-driven automation slice with no `when` at all and the same
outcome, and a translation slice in the given/when/then-events form — with specs in both homes a slice
has. Every writer and renderer downstream asserts against this one model, and the fixture's spec
names and outcome kinds are transcribed by hand so a walk or a strip that reaches only one home, or
collapses one outcome kind into another, reads back short.

**Acceptance Criteria:**
- [ ] `internal/test/fixtures.go` gains a source constant declaring, across slices in an aggregate
      **and** slices on a `mode dcb` context: a command slice spec with a `then` event list, a command
      slice spec with `then rejected`, a view slice spec with `then view`, an event-driven automation
      slice spec whose `when` names the automation's `on` event and whose `then` names a command, a
      schedule-driven automation slice spec with no `when` and a `then` naming a command, and a
      translation slice spec in the given/when/then-events form
- [ ] Its specs together exercise a `given` of more than one event, `given []`, an omitted `given`, and
      a `then` written above its `when`
- [ ] At least one spec sits ahead of a further slice entry rather than last, and at least one spec
      omits its `given` mid-block rather than as its final entry
- [ ] `internal/test/models.go` gains the parsed-model helper alongside `SpecLibraryLendingModel`
      (`:37`)
- [ ] A hand-written transcription lists the fixture's spec names in declaration order across both
      slice homes, and a second lists each spec's outcome kind in the same order, both non-empty; a
      getter reads each back off a parsed model by walking `declaredSlices`, the way `DeclaredSpecNames`
      (`internal/test/fixtures.go:1299-1311`) does
- [ ] `test.WithoutSpecs` over the new fixture's model returns a copy whose specs are gone from both
      homes and which is not equal to the original, while the original still reads back both
      transcriptions in full
- [ ] `oracle.Check` over the fixture returns no diagnostics at all — no parse error, no validation
      error and no lint diagnostic of any severity — and `internal/oracle/oracle_test.go` `"clean
      input"` (`:26`) carries that subtest beside the one for `test.SpecLibraryLending` (`:60-64`)
- [ ] `requireSpecCoverage` (`internal/parser/parser_test.go:5699-5728`) runs over the new fixture
      **unchanged** — the fixture exercises all six shapes it already names — and a sibling guard beside
      it requires the three shapes only the new fixture has: a view outcome, a command outcome, and a
      spec written with no `when` at all. `requireSpecCoverage` is not given new requirements, because
      it also guards `test.SpecLibraryLending`, which states none of those three
- [ ] Deleting any one of the three new shapes from the fixture makes the sibling guard fail, and
      `test.SpecLibraryLending`'s own leaf (`:1249-1266`) still passes with no edit
- [ ] `HotelReservation`, `DescribedHotelReservation`, `KeywordFieldSearchCatalog`,
      `InvariantLibraryLending`, `SpecLibraryLending` and `SpecLibraryLendingSpecNames` are unchanged,
      so `git diff` moves no expected value in `internal/cli/fmt_test.go`,
      `internal/export/export_test.go`, `internal/diagram/contract_test.go`,
      `internal/glossary/glossary_test.go` or `internal/formatter/formatter_test.go`

**Affected Files/Modules:**
- `internal/test/fixtures.go` — the fixture source, its two transcriptions and its getter
- `internal/test/models.go` — the parsed-model helper
- `internal/oracle/oracle_test.go` — the zero-diagnostic subtest
- `internal/parser/parser_test.go` — a leaf for the new fixture beside the one at `:1249-1266`, and a
  sibling of `requireSpecCoverage` (`:5699-5728`) guarding the three shapes only the new fixture has;
  `requireSpecCoverage` itself and the existing leaf do not move

**Patterns to Follow:**
- A second fixture covering the earlier construct and the new one side by side, with its own
  transcriptions rather than an edit to the earlier model: `TriggerReadsLibraryLending`
  (`internal/test/fixtures.go:749`) — `tasks/learnings.md` "A second construct gaining the same field
  extends the existing fixture, with independent twins" and "A new optional field ships a six-part
  fixture kit"
- Fixture prose style, the comment header stating what the fixture witnesses, and the two-home layout:
  `SpecLibraryLending` (`internal/test/fixtures.go:423-572`), whose DCB half (`:505-571`) shows the
  two tag keys and the `decides_on` that keep `dcb/untagged-event`, `dcb/orphan-tag-key` and
  `dcb/single-tag-everywhere` quiet
- The rules the fixture has to keep quiet, because a lint diagnostic of any severity fails
  `emod validate`: every automation needs a `reads` naming a declared view
  (`automation/missing-todo-list`), every view's name ends in `View` and subscribes to fewer than five
  events (`view-naming`, `god-view`), every command is named by a flow and every event produced by a
  flow, a `source` or a translation (`orphan-command`, `orphan-event` — and note a *spec* is not a
  reference, `tasks/learnings.md` "A spec is not a reference"), and no command is referenced by three
  or more pathways (`left-chair`) — the descriptions are in `internal/linter/descriptions.go`
- Exercise an omitted optional part mid-block, never as the last entry — `tasks/learnings.md`
- Pair a `Declared…` getter only with a non-empty transcription — `tasks/learnings.md` "A `Declared…`
  getter answers `nil` for a fixture that declares none of the construct"
- Twin mechanics: `copyWithEditedSlices` and `editedCopies` (`internal/test/fixtures.go:1257-1298`),
  whose nil-stays-nil behaviour is deliberate — `tasks/learnings.md` "`require.NotEqual` on a stripped
  twin is satisfiable without stripping anything"
- Verify the coverage guard by deleting the shape it claims to guard and confirming it goes red —
  `tasks/learnings.md` "Shared fixtures come in an unfeatured/featured pair, guarded by a walk that
  must be extended"

**Testable:** Yes — through `parser.Parse`, `oracle.Check` and the `internal/test` getters, all
exported.

**Verification:** `go test -tags unit ./internal/...`; `go run ./cmd/emod validate` over a temporary
file holding the new fixture, expecting exit 0 and no output; `go run ./cmd/emod lint` over the same
file, expecting exit 0.

**Depends on:** Task 1

---

### Task 3: Resolve a spec's view and command outcomes against the model

**Behavior:** `emod validate` reports an error when a `then view <ViewName>` names a view no slice in
the model declares, or a `then command <CommandName>` names a command no slice declares. Each is
reported at the reference's own position and names the missing construct and the kind it was looked up
as, so an author fixing a typo is pointed at the word they mistyped. Resolution is unqualified and
model-wide, the same rule US-006 gave a spec's `given`, `when` and `then` event references: a view or
command declared in another aggregate or another context resolves.

**Acceptance Criteria:**
- [ ] A `then view` naming a view no slice declares produces exactly one diagnostic, at `Error`
      severity, positioned on the view name, whose whole formatted line names the view and identifies
      it as a view
- [ ] A `then command` naming a command no slice declares produces the equivalent diagnostic, naming
      the command and distinguishing it from a missing view
- [ ] A `then view` naming a view declared in a different aggregate, and one naming a view declared
      directly on a `mode dcb` context, both produce no diagnostic — outcome references are unqualified
      and model-wide
- [ ] A `then command` naming a command declared in a different context produces no diagnostic
- [ ] A `then view` naming something the model declares as a *command*, and a `then command` naming
      something declared as a *view*, each produce their diagnostic — the two lookups stay separate,
      as `given`/`then` and `when` do (`internal/validator/validator.go:136-138`, `:359-368`)
- [ ] A spec naming several undefined constructs reports one diagnostic per reference in declaration
      order, and repeated runs of `validator.Validate` over the same model produce the identical list
- [ ] These diagnostics carry no `RuleName`, so `emod lint --explain` gains nothing to answer for and
      `internal/linter/descriptions.go` is untouched
- [ ] `cli.RunValidate` over a file whose `then view` names an undeclared view exits with `ExitCode`
      1 and the reported message contains the undeclared name and the kind it was looked up as, not
      merely the path and line number
- [ ] `oracle.Check` over the Task 2 fixture and over every fixture that declares no spec still returns
      no diagnostics, and no existing subtest in `internal/validator/validator_test.go` needs editing

**Affected Files/Modules:**
- `internal/validator/validator.go` — `specDiagnostics` (`:341-357`) alongside its existing
  `when`/`given`/`then`-event checks, using `index.viewNames` and `index.commandNames` (`:43-55`,
  filled at `:83-98`)
- `internal/validator/validator_test.go` — a group for the two outcome references, beside `"spec
  references"` (`:1267`)
- `internal/cli/validate_test.go` — the diagnostic as the user receives it

**Patterns to Follow:**
- "X %q does not exist" at the reference's position, with the empty-name guard:
  `appendUndeclaredRef` (`internal/validator/validator.go:325-331`) and `errorAt` (`:150-158`), which
  already sets `Error` severity and leaves `RuleName` empty
- Which names exist and that they are model-wide, both slice homes together: `modelIndex.collect`
  (`internal/validator/validator.go:83-122`) and `model.AllSlices()` (`:74`)
- Keep the outcome lookups narrow and separate — `tasks/learnings.md` "A spec's `when` resolves against
  commands *and* events; `given`/`then` against events only" records that widening one of these
  lookups leaves every test green while widening the language, so each new lookup needs its own
  negative subtest
- The reason a `then view` outcome resolves while a trigger's and a translation's `reads` do not:
  `tasks/learnings.md` "Only an automation's `reads` resolves" — shipped models already name undeclared
  views in the two unchecked places; no model can already contain a `then view`
- Multi-diagnostic ordering asserted with one `require.Equal` against `reportedLines(diags)`:
  `internal/validator/validator_test.go:2473-2482` and `tasks/learnings.md` "Diagnostics gathered from
  more than one AST collection must be position-sorted"
- Whole formatted lines rather than layered `require.Contains` calls:
  `internal/validator/validator_test.go:2473-2482` and `tasks/learnings.md` "A second
  `require.Contains` on one message is often shadowed by the first"
- CLI-layer assertion content: `tasks/learnings.md` "CLI diagnostic tests must assert the
  distinguishing message text", with `internal/cli/validate_test.go:253-258` as the model

**Testable:** Yes — through `validator.Validate`, `oracle.Check` and `cli.RunValidate`.

**Verification:** `go test -tags unit ./internal/validator/... ./internal/cli/... ./internal/oracle/...`;
`go run ./cmd/emod validate` over a temporary file whose `then view` names a misspelled view,
expecting exit 1 and the misspelled name in the message.

**Depends on:** Task 2

---

### Task 4: Reject a `then` shape the enclosing slice cannot state

**Behavior:** `emod validate` reports an error when a spec's outcome does not match the pattern of the
slice that states it — a `view` outcome in a slice declaring no view, a `command` outcome in a slice
declaring no automation, a rejection in a slice declaring no command, or a success event list in a
slice declaring neither a command nor a translation. The pattern is read off the constructs the slice
declares, so the judgement is local to the slice and the same in both slice homes. The message names
the outcome shape and the construct the slice would have to declare, so an author is told which of the
two ends to change.

**Acceptance Criteria:**
- [ ] A `then view <Name>` in a slice declaring no view — the name being a view the model declares in
      another slice, so Task 3's rule stays silent — produces exactly one diagnostic, at `Error`
      severity, positioned on the view name, whose whole formatted line names the outcome shape and the
      construct kind the slice would have to declare
- [ ] A `then command <Name>` in a slice declaring no automation, the name being a command the model
      declares elsewhere, produces the equivalent diagnostic positioned on the command name
- [ ] A `then rejected <name>` in a slice declaring no command, the name being an invariant the
      enclosing scope declares so the US-006 resolution rule stays silent, produces the equivalent
      diagnostic positioned on the invariant name
- [ ] A `then [EventA]` in a slice declaring neither a command nor a translation produces the
      equivalent diagnostic, positioned on the first event of the list; `then []` in the same slice
      produces it positioned on the spec's own name, since an empty list names nothing
- [ ] A `then view` in a slice declaring a view, a `then command` in a slice declaring an automation, a
      `then rejected` in a slice declaring a command, and a `then [EventA]` in a slice declaring either
      a command or a translation each produce no diagnostic
- [ ] A slice declaring both a command and an automation accepts a spec stating a `command` outcome and
      a sibling spec stating an event list, with no diagnostic from either
- [ ] The rule reaches slices in both homes: a slice nested in an aggregate and a slice declared
      directly on a `mode dcb` context each report their own mismatches
- [ ] The rule judges the outcome alone: a spec that omits `when`, and a spec whose `when` names an
      event rather than a command, are each accepted in every slice — this task adds no rule about
      `when`
- [ ] A model with several mismatched outcomes reports them in declaration order, identical across
      repeated runs of `validator.Validate`
- [ ] These diagnostics carry no `RuleName`, so `emod lint --explain` gains nothing to answer for and
      `internal/linter/descriptions.go` is untouched
- [ ] `cli.RunValidate` over a file with a `view` outcome inside a command slice exits with `ExitCode`
      1 and the reported message names the outcome shape and the construct kind, not merely the path
      and line number
- [ ] `oracle.Check` over the Task 2 fixture, over `test.SpecLibraryLending` — whose three trigger-less
      command slices state event-list and rejection outcomes — and over every fixture that declares no
      spec still returns no diagnostics
- [ ] `emod slices list` prints exactly what it printed before over every model: `internal/cli/slices_list.go`
      and `internal/cli/slices_list_test.go` are untouched, and `detectPattern` is neither moved nor
      widened

**Affected Files/Modules:**
- `internal/validator/validator.go` — the shape check, reached from `referenceDiagnostics(slice, index)`
  (`:297-323`), which is the one walk that already holds the slice
- `internal/validator/validator_test.go` — a group for the shape rule, covering each outcome kind
  against a matching and a mismatching slice
- `internal/cli/validate_test.go` — the diagnostic as the user receives it

**Patterns to Follow:**
- The walk that already carries the slice, and the one-line helper shape its siblings use:
  `referenceDiagnostics` and `appendUndeclaredRef` (`internal/validator/validator.go:297-331`)
- `errorAt` (`internal/validator/validator.go:150-158`) for `Error` severity and the empty `RuleName`
   — `tasks/learnings.md` "`RuleName` marks a diagnostic `emod lint --explain` can describe"
- Ordering across a spec's several fields: `sortInDeclarationOrder` (`internal/validator/validator.go:193-201`)
  and the comment at `:338-340` explaining why a spec's parts need it — `tasks/learnings.md`
  "Diagnostics gathered from more than one AST collection must be position-sorted"
- The wording convention that names a symbol together with the scope it was judged in:
  `scopedInvariantDiagnostics` (`internal/validator/validator.go:287-295`) and its two message formats
  (`:272`, `:281`)
- Whole formatted lines, and `reportedLines(diags)` for the multi-diagnostic ordering leaf:
  `internal/validator/validator_test.go:2473-2482` and `:1145-1147`
- Do not extract or share `detectPattern` (`internal/cli/slices_list.go:109-123`): the "Open questions,
  decided" note above records why, and `tasks/learnings.md` "A tested, defensible improvement found on
  the way is still a separate commit" is why widening it does not belong in this task
- CLI-layer assertion content: `tasks/learnings.md` "CLI diagnostic tests must assert the
  distinguishing message text", `internal/cli/validate_test.go:253-258`

**Testable:** Yes — through `validator.Validate`, `oracle.Check` and `cli.RunValidate`.

**Verification:** `go test -tags unit ./internal/validator/... ./internal/cli/... ./internal/oracle/...`;
`go run ./cmd/emod validate` over a temporary file whose command slice states a `view` outcome,
expecting exit 1; `go run ./cmd/emod slices list examples/all_patterns.emod` producing the same table
as before the change.

**Depends on:** Task 2

---

### Task 5: Preserve the view and command outcomes through `emod fmt`

**Behavior:** The formatter writes the two new outcomes back out, so formatting a model no longer
deletes the behaviour a view, automation or translation slice describes. Each is emitted as one
canonical `then` line inside its spec block, in the same position among the spec's entries the other
two outcomes take. A model that states neither outcome formats to exactly the bytes it formatted to
before.

**Acceptance Criteria:**
- [ ] Parsing the Task 2 fixture, formatting it and re-parsing yields a model whose specs match the
      original in name, declaration order, given events and their order, when reference, and outcome —
      including which of the four outcomes each spec states and the name it carries
- [ ] Formatting the formatter's own output of that fixture produces byte-identical text
- [ ] A spec stating a `view` outcome and a spec stating a `command` outcome each round-trip through
      parse → format → parse with the outcome intact; deleting either new case from `formatOutcome`
      makes the round-trip fail rather than silently dropping the line
- [ ] A spec that states no `when` formats with no `when` line, and re-parsing yields a spec with no
      `when`
- [ ] A spec whose entries are written out of canonical order formats to the canonical order, and
      re-parsing the result yields the same spec
- [ ] `internal/cli/fmt_test.go` gains a canonical formatted constant for the Task 2 fixture, pinned
      through `requireFmtSettlesOn` (`:629`), that opens with the `emod 1` header and shows every spec
      at the end of its slice — it is the canonical text `emod fmt` writes, not the fixture source
      re-indented
- [ ] `emod fmt --check` over an already-formatted file carrying all four outcomes reports no change
      needed and leaves the file on disk unchanged
- [ ] `internal/formatter/formatter_test.go` and `internal/cli/fmt_test.go` pass with no edit to any
      existing expected-output constant — `specFormattedEmod` (`internal/cli/fmt_test.go:244`) in
      particular does not move — so a model stating neither new outcome formats exactly as before
- [ ] The round-trip group in `internal/formatter/formatter_test.go` gains one leaf for the new
      fixture rather than a parallel table, and that leaf asserts against the fixture's non-empty
      transcribed spec names and outcome kinds

**Affected Files/Modules:**
- `internal/formatter/formatter.go` — `formatOutcome` (`:387-399`) gains two cases
- `internal/formatter/formatter_test.go` — a leaf in `"round-trip through the parser"` (`:702`, the
  per-fixture group whose `test.SpecLibraryLending` leaf is at `:876`)
- `internal/cli/fmt_test.go` — the canonical constant for the new fixture and the command-level
  behaviour over it (`:244`, `:629`, `:578-581`)

**Patterns to Follow:**
- The outcome writer this task extends: `formatOutcome` (`internal/formatter/formatter.go:387-399`),
  whose `default: return ""` is what makes an unknown variant a silent deletion —
  `tasks/learnings.md` "`ast.ThenClause` dispatches through five type switches, none of which errors"
- The one-line-per-entry spec writer and its omit-when-absent rule: `writeSpec`
  (`internal/formatter/formatter.go:372-385`), and `writeSlice` (`:156-206`) for where specs sit —
  `tasks/learnings.md` "`emod fmt` moves a spec to the end of its slice and orders its entries
  given/when/then"
- Round-trip through the parser is the assertion that catches a mangled or dropped declaration, not a
  golden: `internal/formatter/formatter_test.go:702`
- A fmt golden is a pinned canonical constant, never the input fixture handed back —
  `tasks/learnings.md` "`emod fmt` canonicalises order, so a fmt golden is never the input
  re-indented", `internal/cli/fmt_test.go:118`, `:135`, `requireFmtSettlesOn` (`:629`)
- Every expected string starts with the `emod <n>` header — `tasks/learnings.md` "Formatter output
  always begins with `emod N`"
- Fold the new per-fixture assertion into one leaf rather than opening a parallel table, and pair a
  `Declared…` getter only with a non-empty transcription — `tasks/learnings.md` "A `Declared…` getter
  answers `nil` for a fixture that declares none of the construct"
- `emod fmt <file>` writes in place, so a receipt run over a real fixture dirties the tree — copy to a
  temp path first (`tasks/learnings.md` "`emod fmt <file>` writes in place")
- Emit every string through `quoted` (`internal/formatter/formatter.go:47-49`), never `%q` —
  `tasks/learnings.md` "Never write emod source with `%q`"; the two new outcomes carry identifiers
  rather than strings, so nothing new goes through it here

**Testable:** Yes — through `formatter.Format` and `cli.RunFmt`.

**Verification:** `go test -tags unit ./internal/formatter/... ./internal/cli/...`;
`go run ./cmd/emod fmt` over a *copy* of a temporary file holding the Task 2 fixture, then again over
the result, expecting identical bytes; `git diff --exit-code -- '*.emod'` afterwards, since
`emod fmt <file>` writes in place and a receipt run over a tracked file would reformat it.

**Depends on:** Task 2

---

### Task 6: Carry the two outcomes through the JSON and CUE exports and the embedded schema

**Behavior:** Both model exports emit a spec's `view` and `command` outcomes under the same `then`
object that already carries the event list and the rejection, each with the name and the name's
position; the bundled schema declares them; and the diagram document — which is nodes and edges —
carries no trace of any spec, nor does any rendered diagram or the glossary. A model stating neither
outcome exports and renders byte-identically to before.

**Acceptance Criteria:**
- [ ] The JSON export of the Task 2 fixture carries every spec under the slice that declares it, in
      declaration order, and reading a `then` back tells which of the four outcomes it states: the
      event names in order, the rejected invariant name, the view name, or the command name
- [ ] The serialized keys spell the DSL keywords — the value keys are `view` and `command` and each
      position key is that spelling plus `_position` — and a document re-keyed with a Go-field spelling
      instead fails `cue vet` naming the key
- [ ] `fullModelJSON` (`internal/cue/embed_test.go:162`), which exists to exercise every definition the
      schema declares, states both new outcomes, so a key the schema forgets is caught by the vet leaf
      at `:63-70` rather than passing over a document that never carries it
- [ ] Each new position key's value is the position of the name it belongs to: the whole-spec position
      leaf (`internal/export/export_test.go:1496-1510`) gains one spec per new outcome, `require.Equal`
      against the complete exported map including line and column
- [ ] `emittedKeyOrder` (`internal/export/export_test.go:4755`) shows the outcome object's keys filed
      the way its existing siblings are, compared in the same subtest against the key list of an
      outcome that already exists, so the expectation is not arbitrary
- [ ] The CUE export of the fixture carries the same, and "CUE and JSON exports describe the same
      model" (`internal/export/export_test.go:3842`) passes for it
- [ ] `internal/cue/schema.cue` declares the two on `#SpecOutcome` (`:74-77`) as optional keys, and
      `cue vet -d '#Model'` over the export of the fixture passes (`internal/export/export_test.go:3813`)
- [ ] `emod schema` prints a schema whose spec-outcome definition declares all four outcomes
- [ ] A read-back of the exported document against the fixture's hand-transcribed specs, filed under
      the slice that states each, matches for both slice homes — the aggregate-nested slices and the
      slices declared directly on the `mode dcb` context — in the order the writer emits them
- [ ] The diagram JSON document produced from the fixture carries no spec at all: no spec name appears
      at any key or depth, and the document is byte-identical to the one produced from
      `test.WithoutSpecs` of the same model — a text search for an outcome's *view* name proves nothing
      here, because a declared view is a node the document legitimately carries
- [ ] Every diagram rendering of the fixture — drawio, SVG, mermaid and ASCII — is byte-identical to
      the rendering of `test.WithoutSpecs` of the same model, and the comparison opens by asserting the
      two models differ, that the stated model reads back the fixture's full transcribed spec names,
      and that the twin reads back none
- [ ] The glossary markdown and JSON renderings of the fixture are identical to those of
      `test.WithoutSpecs` of the same model — a spec is a scenario, not a term
- [ ] Existing subtests in `internal/export/export_test.go`, `internal/cli/schema_test.go`,
      `internal/diagram` and `internal/glossary` pass with no edit to any expected output, so a model
      stating neither new outcome exports and renders exactly as before

**Affected Files/Modules:**
- `internal/export/json.go` — `jsonSpecOutcome` (`:201-205`) and `convertSpecOutcome` (`:437-448`)
- `internal/export/cue.go` — `cueWriter.writeSpecOutcome` (`:144-150`)
- `internal/cue/schema.cue` — `#SpecOutcome` (`:74-77`)
- `internal/export/export_test.go` — the two exports, the key order, the positions leaf, the schema
  conformance (`:3813`), the both-formats-agree leaf (`:3842`), the per-slice read-back
  (`specsByOwner`, `:4423`; `libraryLendingSpecs`, `:4314`) and the diagram-document guard (`:3180-3190`)
- `internal/diagram/contract_test.go` — the four-renderer differential alongside `"stating specs leaves
  the picture untouched"` (`:441-453`)
- `internal/glossary/glossary_test.go` — the receipt that the vocabulary is unchanged (`:587-594`)

**Patterns to Follow:**
- All three surfaces land together: `tasks/learnings.md` "A new exported field must land in JSON, CUE
  and `schema.cue` in the same change"
- A serialized key spells the DSL keyword and a position key takes that spelling plus `_position` —
  `tasks/learnings.md` "A serialized key spells the DSL keyword; the Go field may name the concept",
  with the negative leaves it names as the shape to copy — `internal/cue/embed_test.go:111-121` and
  `:139-146` each re-key one value in a copy of `fullModelJSON` and require `cue vet` to fail naming
  the wrong spelling
- Key order comes from the `json*` siblings, not from `schema.cue` — `tasks/learnings.md` "JSON and
  CUE order their document keys differently", and `emittedKeyOrder` is what makes it assertable
  (`tasks/learnings.md` "JSON key order is assertable from the raw bytes")
- A `*_position` key needs its value pinned to its own AST position, in the positions leaf and not only
  in the key-order expectation — `tasks/learnings.md` "A `*_position` key needs its value pinned to its
  own AST position"
- Reading a decoded document back: `exportedAutomations`, `exportedSlices`, `statedUnder` and
  `objectsUnder` (`internal/export/export_test.go:4634-4676`) visit both slice homes in the writer's
  order —
  `tasks/learnings.md` "Read a decoded export document back with `objectsUnder`/`statedUnder`"
- The two export guards cannot see a key neither writer emits, so the read-back against a transcribed
  map is what catches a gap — `tasks/learnings.md` "The two export guards cannot see a list neither
  writer emits"
- Keep the diagram document forked: `internal/export/diagram.go` is a separate document by design, and
  the existing absence guard is `internal/export/export_test.go:3180-3190`
- A differential must first prove its twin differs and that the strip visited every home:
  `internal/diagram/contract_test.go:441-453` — `tasks/learnings.md` "A differential receipt must first
  prove the twin actually differs" and "`require.NotEqual` on a stripped twin is satisfiable without
  stripping anything"
- Do not add a "render it twice" or "export it twice" assertion — `tasks/learnings.md` "An assertion
  whose expected value comes from the code under test is the recurring review finding"

**Testable:** Yes — through `export.ExportJSON`, `export.ExportCUE`, `export.ExportDiagramJSON`,
`cli.RunSchema`, the four `diagram.Export*` renderers and `glossary.RenderMarkdown` /
`glossary.RenderJSON`.

**Verification:** `go test -tags unit ./internal/export/... ./internal/cue/... ./internal/diagram/...
./internal/glossary/... ./internal/cli/...`; `go run ./cmd/emod schema`; `go run ./cmd/emod export -f
cue <temp file holding the fixture>` — noting `tasks/learnings.md` "urfave/cli v2 discards every flag
written after the file argument", so the flag goes *before* the path.

**Depends on:** Task 2

---

### Task 7: Accept the two outcomes in the tree-sitter grammar

**Behavior:** The tree-sitter grammar parses `then view <Name>` and `then command <Name>` inside a
spec block, so a file `emod validate` accepts is not red-squiggled in a tree-sitter-backed editor. The
grammar stays looser than the Go parser: a spec's entries remain unordered and unbounded, and a spec
remains admissible anywhere among a slice's other entries.

**Acceptance Criteria:**
- [ ] A corpus case for a view slice whose spec omits `when` and concludes with `then view <Name>`
      parses to the expected tree
- [ ] A corpus case for an automation slice whose spec states `when <Event>` and `then command <Name>`
      parses, and another for a schedule-driven automation slice whose spec omits `when` and states the
      same outcome
- [ ] A corpus case for a translation slice's spec in the given/when/then-events form parses
- [ ] A corpus case places a spec stating a `view` outcome inside a slice on a `mode dcb` context
- [ ] A corpus case writing a spec's entries out of canonical order with one of the new outcomes parses
- [ ] The existing `fields` corpus cases still parse fields named `view`, `command`, `spec`, `given`,
      `when`, `then` and `rejected` as field lines, not as spec entries
      (`editors/tree-sitter-emod/test/corpus/fields.txt:14`, `:19`, `:40-44`)
- [ ] The one-line comment above `spec_then` in `editors/tree-sitter-emod/grammar.js` spells the rule's
      full shape, naming all four outcomes
- [ ] `mise exec -- task test:grammar` passes, and running it a second time leaves every tracked file
      under `editors/tree-sitter-emod/` byte-identical
- [ ] No file under `editors/tree-sitter-emod/src/` is tracked — `git ls-files
      editors/tree-sitter-emod/src` returns nothing
- [ ] No file under `editors/tree-sitter-emod/queries/` changes and
      `editors/vscode/syntaxes/emod.tmLanguage.json` is untouched: `view` and `command` are already in
      both keyword lists, so this task adds no highlighting

**Affected Files/Modules:**
- `editors/tree-sitter-emod/grammar.js` — `spec_then` (`:125-131`) gains two arms, and its comment
  (`:106`) is restated
- `editors/tree-sitter-emod/test/corpus/specs.txt` — the new cases

**Patterns to Follow:**
- The exact shape both new arms take, already written twice in the same file:
  `seq('command', $.any_identifier)` inside `automation_definition`
  (`editors/tree-sitter-emod/grammar.js:301-312`) and `translation_definition` (`:315-326`); the
  existing `seq('rejected', $.any_identifier)` in `spec_then` (`:129`) is the sibling arm
- The grammar must never be stricter than the Go parser: entries stay `repeat(choice(...))`, and an
  `optional(...)` in a block body is a bug — `tasks/learnings.md` "The tree-sitter grammar must never
  be stricter than the Go parser"
- Every rule carries a one-line example of its full shape, and nothing tests the comments —
  `tasks/learnings.md` "Every `grammar.js` rule carries a one-line example of its full shape"
- Corpus case layout and headings: `editors/tree-sitter-emod/test/corpus/specs.txt`, whose five
  existing cases are the format to match
- Run through `mise exec --` and do not un-ignore `src/` — `tasks/learnings.md` "Run repo tooling
  through `mise exec --`, not bare PATH" and "Generated tree-sitter `src/` stays gitignored"

**Testable:** Yes — the corpus cases are the tests, run by `task test:grammar`.

**Verification:** `mise exec -- task test:grammar`, run twice, the second run leaving the tracked files
untouched (`git diff --exit-code editors/tree-sitter-emod`); `git ls-files
editors/tree-sitter-emod/src` returning nothing.

**Depends on:** Task 1

---

### Task 8: Document the four spec shapes in the DSL reference

**Behavior:** The DSL reference teaches the spec shape each slice pattern takes: which of the four
`then` forms belongs to which pattern, that a view slice's spec and a schedule-driven automation
slice's spec both omit `when` and are told apart by their outcome, that an event-driven automation's
`when` names its activation event, which names must resolve and against what, and that an outcome the
slice cannot state is a validation error. A reader learning the language finds it in the `### spec`
subsection, beside the four pattern sections it refers to.

**Acceptance Criteria:**
- [ ] The `### spec` subsection of `docs/dsl-reference.md` documents all four `then` shapes with an
      example of each, and states which slice pattern each belongs to
- [ ] It states that a view slice's spec and a schedule-driven automation slice's spec both omit `when`
      and that the `then` shape is what tells them apart, and that an event-driven automation's `when`
      names the event the automation activates on
- [ ] It states the resolution rules the story adds: a `view` outcome's name must be a view declared
      anywhere in the model and a `command` outcome's name a command declared anywhere in the model,
      linking to [`view`](#view-pattern) and [`automation`](#automation-pattern)
- [ ] It states that an outcome the enclosing slice cannot state is a validation error, and says what
      each outcome requires the slice to declare
- [ ] The sentence at `docs/dsl-reference.md:422` saying the view, automation and translation spec
      shapes "are not part of the language yet" is gone, and the document reads as if it had always
      described all four — no note of what the reference used to say
- [ ] §11 "Cross-References" lists the spec's `then view` and `then command` outcomes as referencing
      sites in the `view <Name>` and `command <Name>` rows of its table (`:638-645`), and its
      validation bullet list (`:649-655`) names the two errors this story adds
- [ ] No `## <n>.` heading and no `### ` heading is added, removed, renamed or reordered: the
      `^## [0-9]+\.` list still reconciles against every `\(#[0-9]+-` link, and the `^### ` list against
      every `\(#[a-z]` link — including the three that cite `#spec` and the six that cite
      `#automation-pattern`
- [ ] Every fenced block the subsection gains keeps a plain ``` fence unless it is a whole model that
      `oracle.Check` accepts; any block written with an ```emod fence is a complete model reporting no
      diagnostic from `oracle.Check`, which `internal/oracle/oracle_test.go`'s "documented models" leaf
      enforces
- [ ] `keywordDescriptions["then"]` (`internal/lsp/hover.go:38`) names all four outcomes rather than
      only the events and the rejection; that one string is the only change under `internal/lsp`, and
      no entry is added to the map, to `completer.go`'s lists or to `valueSlots`
- [ ] `internal/lsp/keywords_test.go`'s hover and completion coverage subtests pass unchanged, since no
      keyword is added

**Affected Files/Modules:**
- `docs/dsl-reference.md` — the `### spec` subsection (`:376-422`) and §11's table (`:638-645`) and
  bullets (`:649-655`)
- `internal/lsp/hover.go` — the `then` entry of `keywordDescriptions` (`:38`)

**Patterns to Follow:**
- Subsection voice, the fenced form-then-example shape and the closing consumer sentence: the existing
  `### spec` subsection (`docs/dsl-reference.md:376-422`) and the `### invariant` subsection (`:147`)
- The pattern sections this subsection cross-references, whose heading text must be held fixed:
  `### View Pattern` (`:309`), `### Automation Pattern` (`:324`), `### Translation Pattern` (`:353`) —
  `tasks/learnings.md` "`docs/dsl-reference.md` sub-heading anchors are cited more often than the
  numbered ones"
- Editing inside an existing numbered section is safe; adding or reordering one renumbers every heading
  below it and invalidates each number-prefixed link — `tasks/learnings.md` "`docs/dsl-reference.md`
  anchors embed the section number"
- An ```emod fence is a promise that the block validates — `tasks/learnings.md` "An ```emod fence is a
  promise that the block validates"; the subsection's existing blocks are fragments behind plain
  fences, deliberately
- Write the document as if it were its first version: no note of what the reference used to say, and no
  "now supports" framing — the world's state is content, the document's history is not
- The `then` hover description is a one-line correction this story owes because it makes the existing
  wording false; hover, completion and navigation *over* the new outcomes belong to US-015, and
  `tasks/learnings.md` "A tested, defensible improvement found on the way is still a separate commit"
  is why nothing else in `internal/lsp` moves here

**Testable:** No — prose plus one description string; correctness is that any ```emod block validates,
no anchor breaks, and the hover coverage subtests still pass.

**Verification:** `go test -tags unit ./internal/oracle/... ./internal/lsp/...`; reconcile the
`^## [0-9]+\.` heading list against the `\(#[0-9]+-` link list, and the `^### ` heading list against
the `\(#[a-z]` link list, in `docs/dsl-reference.md`; `go run ./cmd/emod validate` over any ```emod
block written to a temp file.

**Depends on:** Tasks 3, 4, 5, 6

---

## Summary

**Total tasks:** 8

**Ordering rationale:** dependency-first, with the language surface settled before anything reads it.
Task 1 lands the two `ThenClause` variants and the parse arms — the whole of the new grammar, and
notably no new keyword, since `view` and `command` are keywords already. Task 2 lands the shared
four-pattern fixture every later task asserts against; it is a *sibling* of `test.SpecLibraryLending`
rather than an edit to it, which is what keeps six downstream transcriptions of that model from moving
and what lets the writer tasks run in any order after it. Tasks 3 and 4 deliver the story's two
validation criteria and are independent of each other — Task 3 reads the model-wide name index that
already exists, Task 4 reads the slice's own element collections. Task 5 comes next among the fan-out
tasks because a formatter that does not know an outcome silently deletes the `then` line, which is the
most damaging gap the story could leave open. Tasks 6 and 7 close the two surfaces this repo requires
of any new construct — the exports plus the schema, and a grammar that is never stricter than the
parser. Tasks 3-6 depend only on Task 2 and Task 7 only on Task 1, so all five can run alongside
one another. Task 8 documents the finished surface and corrects the one sentence in the reference, and the
one hover string, that this story makes false.

**Coverage of the story's acceptance criteria:**

| Criterion | Task |
|---|---|
| In a view slice, a spec omits `when` and concludes with `then view <ViewName>` | 1 (parse and accept the omitted `when`), 4 (the outcome belongs to a slice declaring a view) |
| In an event-driven automation slice, `when` names the automation's `on` event and the spec concludes with `then command <CommandName>` | 1 (parse; a `when` naming an event already resolves via US-006's `declaresCommandOrEvent`), 4 (the outcome belongs to a slice declaring an automation) |
| In a schedule-driven automation slice, a spec omits `when` and concludes with `then command <CommandName>` | 1, 4 |
| In a translation slice, a spec takes the given/when/then-events form | 2 (the fixture witnesses it), 4 (an event-list outcome belongs to a slice declaring a command or a translation) |
| The named view or command outcome must resolve to a construct defined in the model | 3 |
| A `then` shape that does not match the slice pattern is a validation error | 4 |
| Every existing `.emod` file stays valid with unchanged meaning (story-wide constraint) | 1 (no keyword added, every US-006 shape parses unchanged), 4 (`test.SpecLibraryLending`'s trigger-less command slices still validate), 5 (formatter goldens untouched), 6 (export, diagram and glossary receipts) |

Tasks 2, 5, 6, 7 and 8 carry no story criterion of their own — they are the fixture kit and the fan-out
this repo requires of any new construct: one shared model every writer asserts against, `emod fmt` must
not drop the shape, the JSON/CUE/schema trio moves together, the tree-sitter grammar must not reject
what `emod validate` accepts, and the reference must teach it.

Nothing from the story is deferred. What US-007 deliberately leaves to later stories: the four `spec/*`
lint rules, including `spec/given-outside-boundary`, which is what the feature ultimately pays for
(US-008); flow rejection edges (US-009); example payload literals on element references, and with them
the question of expected view state on a `then view` outcome (US-010); `given` / `when` / `then`
alignment and the wider canonical attribute order in `emod fmt` (US-014); LSP hover, completion and
go-to-definition over the two new outcomes (US-015); rendering specs on diagrams behind `--specs`
(US-016); syntax highlighting, which needs no work because both keywords are already coloured (US-017);
and specs in `examples/*.emod` (US-018).

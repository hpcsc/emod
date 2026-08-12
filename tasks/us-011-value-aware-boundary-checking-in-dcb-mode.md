# US-011: Value-aware boundary checking in DCB mode

## Progress
- [x] Task 1: State a tagged field in both payloads of the shared payload fixture
- [x] Task 2: Report a `given` payload value the command's tag predicate excludes
- [x] Task 3: Check every tag predicate a conjunction states, and no other

---

## Story Reference

`user-stories/specs-and-metadata.md` → **US-011: Value-aware boundary checking in DCB mode**
(eleventh story of "Specs, Invariants, and Model Metadata", lines 147-157). Design notes:
`docs/proposals/specs-and-metadata-proposal.md:254` (the paragraph under the four-row rule table —
"for a command with `where tag(entity = customerId)`, the rule can check not only that a `given`
event's type is matched by `decides_on`, but that its tagged field value equals the `when` command's
`customerId`") and `:599` (Phase 3 lists "the DCB value-check upgrade to
`spec/given-outside-boundary`" beside the payload grammar, "purely additive on Phase 2"). The story
file's Non-Goals list at `:236` carries the sentence that decides the whole shape of this check:
"Variable binding in specs (`let`) — payload linkage is by repetition of the same value". Repetition
of a literal *is* the linkage; this rule is the only thing that reads it.

**In scope:** one arm added to one existing lint rule. `spec/given-outside-boundary` already reports
in DCB mode when a `given` event's *type* is not named by the `when` command's `decides_on`
(US-008 Task 6). This story adds the *value* comparison on top: for a `when` command whose
`decides_on` states a `where tag(key = field)` predicate, a `given` event whose payload states a
different value for that field than the `when` payload does is reported under the same rule name, at
the same warning severity, with a third message text. No new rule name, no new severity, no new
description entry — the entry US-008 Task 5 registers is widened to cover the arm. The whole change
is Go-side, inside `internal/linter`, plus the shared payload fixture Task 1 prepares.

**Out of scope:**

- **The rule itself and its two type-level arms (US-008).** `spec/given-outside-boundary`, its name,
  its warning severity, its `emod lint --explain` description entry, its place in the
  hand-maintained rule list at `internal/cli/lint_test.go:627-645`, its aggregate arm and its DCB
  type-level arm all land in `tasks/us-008-lint-spec-coverage-and-boundary-assumptions.md` Tasks 5
  and 6. This story adds nothing that would exist without them. Its three sibling rules —
  `spec/command-without-spec`, `spec/no-rejection-path`, `spec/invariant-never-exercised` — are not
  touched, only kept quiet by the fixtures here.
- **The payload grammar (US-010).** The `{ field: value }` block, the number literal token, the
  payload field-name check and the literal-kind-against-declared-type check are all
  `tasks/us-010-state-example-payloads-in-specs.md`. This story reads the payload node US-010 hangs
  off `ast.SpecElement` (its open question 1) and adds no parsing, no validation and no literal
  form.
- **The formatter (US-014).** No rule changes what `emod fmt` writes; `internal/formatter` is
  untouched by every task here.
- **The LSP (US-015).** No hover, completion or navigation. The arm reaches the editor only through
  `ConvertDiagnostics` (`internal/lsp/diagnostics.go:9-42`), which publishes whatever `oracle.Check`
  returns.
- **Diagrams (US-016).** Nothing renders. `internal/diagram` and `internal/export/diagram.go` are
  untouched.
- **Highlighting (US-017).** No editor grammar moves — see the keyword note below.
- **Examples and reference coverage (US-018).** No file under `examples/` gains a payload or a spec.
  `docs/dsl-reference.md` is not edited: no section of it and no section of `README.md` lists lint
  rules today (§12 "Pipeline" names the linter as a stage; `README.md:113-116` shows `emod lint`
  with no rule table), the payload documentation belongs to US-010 Task 8, and the two prior
  pure-lint stories — `tasks/completed/us-008-flag-automations-with-no-todo-list.md` (three tasks)
  and `tasks/completed/US-006-lint-structural-anti-patterns.md` — set the precedent of shipping a
  rule with no documentation task.
- **Rejection flow edges (US-009).** No `flow` entry kind is read or added.

**No lexer keyword is added, and this was checked.** Every word this arm reasons about —
`decides_on`, `where`, `tag`, `and`, `or`, `not`, `spec`, `given`, `when`, `then` — is already in
`internal/lexer/token.go`; `true` and `false` are recognized by position and deliberately not
keywords (US-010 open question 3). So `lexer.Keywords()` does not grow, and neither CI-enforced
drift test has anything to gain: `editors/tree-sitter-emod/test/queries/keywords_test.go:47`
(`TestEditorKeywordCoverage`, requiring every lexer spelling in all three editor grammars) and
`internal/lsp/keywords_test.go:18` (`TestKeywordCoverage`, requiring hover text per keyword) are
both already satisfied for every word here. No task in this list touches `editors/` or
`internal/lsp`.

**The pre-existing `ConvertDiagnostics` defect does not bite this story, and was checked.**
`internal/lsp/diagnostics.go:31-36` maps `diagnostic.Warning` to `SeverityWarning` and everything
else — including `diagnostic.Info` — to `SeverityError`, so the two info-severity rules US-008 adds
publish as editor errors exactly as `dcb/single-tag-everywhere` already does. This arm is a
**warning**, so it takes the one branch the mapping gets right and publishes as a warning. The
defect is recorded here and deliberately not fixed: fixing it changes an existing rule's behaviour
and belongs in its own commit (`tasks/learnings.md:461-464`).

**Open questions, decided.** Eight shapes the story's three criteria do not name, each decided
against evidence in the tree, and each chosen so the arm stays a strict addition to US-008's.

1. **The tagged field is `TagPredicate.Value`, not `TagPredicate.Field`.** In
   `where tag(desk = deskId)` the AST records `Field: "desk"` (the tag key) and `Value: "deskId"`
   (the field reference). `predicateDiagnostics` (`internal/validator/validator.go:476-482`) is what
   settles it: it checks `Field` against `index.eventTagKeys` and `Value` against
   `index.eventFields`. So "the tagged field" the story's criteria compare is `Value`, and the field
   name a payload must state to be compared is `Value`. Two consequences worth naming: the parser
   accepts a quoted string in that slot (`parseTagPredicate`, `internal/parser/parser.go:896-903`),
   but the validator then reports `field reference %q is not declared on any event in decides_on`,
   so in any model that validates the slot always holds a field name; and `Operator` is always `=`,
   because `lexer.Equals` is the only token `parseTagPredicate` accepts there — there is no `!=` and
   therefore no operator variance for this arm to reason about.
2. **Nothing requires the tagged field to be declared on the `when` command, and today nothing in
   the repository declares it there.** The validator requires a predicate's field reference to be
   declared on at least one `decides_on` *event* only. But US-010 Task 3 makes a payload field name
   the referenced construct does not declare a validation error, so the `when` payload can state the
   tagged field only if the *command* declares it too. Measured over every checked-in model (method
   and numbers below): 16 tag predicates across 9 commands, and **2** of the 16 name a field that
   command also declares — both of them in the single ` ```emod ` fence at
   `docs/dsl-reference.md:175`, whose slice states no spec. `test.SpecLibraryLending`'s `ReleaseDesk`
   is the shape that matters and does not have it: its predicate names `deskId` and `memberId` while
   the command declares only `sessionId`. This is why Task 1 exists — without it the repository's one
   payload-stating model is structurally incapable of exercising the rule, and every "matching values
   produce no warning" criterion would close against a source string the same task authored.
3. **The value arm runs only on `given` elements the type arm accepted.** A `given` event whose name
   the `decides_on` events list does not carry is already reported by US-008 Task 6; comparing its
   payload as well would put two diagnostics on one element for one mistake. This is the same
   don't-double-report rule US-008 applies to a `given` name no slice declares (its decision 6), and
   the same rule `checkQueryTooBroad` (`internal/linter/linter.go:373-376`) applies by skipping a
   command with no `decides_on`.
4. **Silence is the answer whenever the comparison has no basis.** Six shapes, all producing no
   value-level diagnostic: either payload omits the tagged field (the story's third criterion — a
   payload is partial by design, US-010's "a spec is an example, not an instance"); the `when`
   command states no `decides_on` (US-008 decision 6, inherited unchanged); it states `decides_on`
   with no `where` predicate (there is nothing to compare against, and `dcb/query-too-broad` already
   reports that command); the spec has no `when`, or its `when` names an event rather than a command
   (US-007's automation and view shapes — an event carries no `decides_on`); either payload states
   the tagged field more than once (US-010 open question 7 keeps both entries, so the payload states
   two values and the rule cannot say which one the author meant); and the slice is nested in an
   aggregate rather than declared on a context (decision 5).
5. **Arm selection is US-008's, by slice home, and is not re-derived.** A slice nested in an
   aggregate takes the aggregate arm; a slice declared directly on a context takes the DCB arm
   (`tasks/us-008-lint-spec-coverage-and-boundary-assumptions.md` decision 5). This story extends
   the DCB arm only, so a spec in an aggregate-nested slice takes no value check whatever the
   enclosing context's mode string. `ast.SliceRef` (`internal/ast/traverse.go:19-25`) already
   carries the home where the rule needs it, and the mode helpers `isDCBMode`/`isMixedMode`
   (`internal/linter/linter.go:153-159`) are deliberately not consulted.
6. **Two literals are equal when their kinds match and their values match; a cross-kind pair is not
   compared at all.** `"101"` against `101` is reachable, not a type error and not unreachable: the
   two payloads sit on different constructs, and nothing in the language requires a command's field
   and an event's field of the same name to declare the same type, so US-010 Task 4 accepts a
   `string`-typed `"101"` on the command and an `int`-typed `101` on the event without complaint.
   The rule stays silent on that pair, for the same reason it stays silent on an omitted field: it
   cannot tell "a different value" from "a different spelling of one value", and a pair whose kinds
   differ is a field typed two ways — a modelling problem this rule has no vocabulary for and whose
   message would name a boundary. The positive half is that every warning the arm emits names two
   literals of one kind. Within a kind: strings compare by their exact text, since the language has
   no escape sequences and the source text between the quotes *is* the value
   (`tasks/learnings.md:46-49`); booleans compare by their spelling, `true` and `false` being the
   only two; **numbers compare as numbers, not as source text**, so `12.50` and `12.5` are one
   value and `007` and `7` are one value — US-010 open question 10 records that the AST keeps a
   number's source text verbatim so `emod fmt` can write it back, which makes a text comparison
   report a formatting difference as a boundary violation.
7. **One diagnostic per disagreeing tagged field, positioned on the value in the `given` payload.**
   The type arm positions at the event name inside the `given` list, because there the whole
   reference is wrong; here the event name is right and one literal is wrong, so the diagnostic sits
   on the word an author would edit. US-010 Task 1 records a position for both halves of a payload
   field, so the position exists. A `given` element disagreeing on two tagged fields reports twice,
   because each is separately fixable — and the two land at two positions, which a whole-line
   comparison then tells apart.
8. **The arm carries a third message text, distinct from both of US-008's, under the one rule
   name.** US-008 Task 6 already requires the rule's two arms to carry different texts and every
   leaf asserting either to compare the whole formatted diagnostic line
   (`tasks/learnings.md:476-479`). This arm makes it three, and takes the same obligation. Sharing
   the name is deliberate: an author configuring the rule is configuring one idea — "this spec
   assumes history the boundary cannot see" — and the type-level and value-level halves of that idea
   are not separately silenceable. What distinguishes them is the message, which names the tagged
   field, the tag key, both stated values and the command, and the position, which is a payload
   value rather than an event name. So `emod lint --explain spec/given-outside-boundary` keeps
   answering with one widened description, and `internal/cli/lint_test.go`'s rule list does not
   grow.

**Measured blast radius: the arm fires zero times, and the interesting number is why.** A throwaway
walker was written over every model the repository ships as valid — the eight shared fixtures in
`internal/test/fixtures.go`, `examples/*.emod` less the `_test.emod` suffix, `internal/parser/testdata/*.emod`
less `invalid.emod`, and every ` ```emod ` fence in `README.md` and `docs/dsl-reference.md`
(`internal/wasm/pipeline_test.go`'s `billingModel` declares no DCB context and was read by
inspection) — 20 models in all, then deleted. It counted:

| Measured over 20 models | Count |
|---|---|
| Slices declared directly on a context (the DCB arm's home) | 21 |
| Specs in those slices | 3 |
| `given` elements in those specs | 2 |
| Those specs whose `when` resolves to a command stating `decides_on` | 1 |
| …of which state a `where` predicate | 1 |
| Tag predicates reachable from that predicate | 2 |
| …in conjunctive position | 2 |
| …whose field the `when` command also declares | **0** |
| Commands stating `decides_on` anywhere in any model | 9 |
| …stating a `where` predicate | 9 |
| Tag predicates across all of them | 16 |
| …whose field that same command also declares | **2** |

Zero is the honest answer twice over. No checked-in model states a payload — payloads are US-010 —
so the rule has nothing to read. But the structural precondition fails too: the one spec in the tree
positioned to exercise the arm is `test.SpecLibraryLending`'s "frees the desk its reader is seated
at", whose `when ReleaseDesk` carries `where tag(desk = deskId) and tag(reader = memberId)` while
`ReleaseDesk` declares only `sessionId`, so neither tagged field can legally appear in its `when`
payload. The only two tag predicates in the repository whose field is declared on their own command
are both in the `docs/dsl-reference.md:175` fence, on `ClaimDesk` — and that fence states no spec
today, though US-008 Task 4 gives it rejection specs, which is why Tasks 2 and 3 sweep it by name.
The interesting work in this story is therefore not moving models out of the way; it is constructing
the one fixture that exercises the arm at all, which is Task 1.

**Overarching constraint:** every existing `.emod` file stays valid with unchanged meaning. That is
nearly free here — the arm can only fire on a spec that states payloads in a context-level slice
whose `when` command declares both a `decides_on` predicate and the field that predicate tags, and
no checked-in model has any of the three. The one model that moves is the payload fixture Task 1
edits, and it moves so the arm has something clean to be silent about.

**Learnings folded in** from `tasks/learnings.md`: a lint warning fails `emod validate`, so a new
rule sweeps every checked-in model before it lands (:466-469), which is what the measurement above
is; a lint fixture trips exactly one rule, so it is never the minimal model (:471-474) — sharpened
here because a DCB fixture must simultaneously keep the four `dcb/*` rules and all four `spec/*`
rules quiet; a rule whose message branches on model state is pinned by whole formatted lines
(:476-479), which this arm needs because it is the rule's third text; `RuleName` marks a diagnostic
`emod lint --explain` can describe (:166-169), so widening an arm obliges widening the description
rather than adding an entry; a slice has two homes and much of the repo walks only one (:171-174); a
spec is not a reference, so a spec-carrying fixture still needs its flows (:191-194); a spec's
`when` resolves against commands *and* events while `given`/`then` resolve against events only
(:201-204), which is decision 4's "`when` names an event" case; a new shared fixture owes
`internal/oracle` a zero-diagnostic subtest and a `mode dcb` model is the usual tripwire
(:151-154); a "no expected constant moves" criterion is unsatisfiable when the task edits a shared
fixture (:481-484), which Task 1 obeys; `require.NotEqual` on a stripped twin is satisfiable without
stripping anything (:206-209) and `editedCopies` is shallow, so an edit reaching inside a slice must
nest (:216-219); a second `require.Contains` on one message is often shadowed by the first
(:136-139); CLI diagnostic tests
must assert the distinguishing message text (:6-9); an assertion whose expected value comes from
the code under test cannot fail (:126-129); never write emod source with `%q` (:46-49), which is
also why decision 6 compares string literals by exact text; an ` ```emod ` fence is a promise that
the block validates (:526-529); a `_test.go` file always carries the `Test…` umbrella for the name
it wears (:456-459); a tested improvement found on the way is still a separate commit (:461-464),
which is where the `ConvertDiagnostics` defect goes; and acceptance criteria never reference commit,
branch or remote state (:21-24, :246-249).

**Repo drift, noted:** `internal/export/export.go` no longer exists — the package is `json.go`,
`cue.go` and `diagram.go`. Several `tasks/learnings.md` entries still cite the old path. No task
here touches the package.

---

## Codebase Context

**The predicate AST, and which half is the field.** `ast.DecidesOnClause`
(`internal/ast/ast.go:246-253`) carries `Events []string`, `EventsPos []Position` and
`Predicate PredicateExpr`. `PredicateExpr` (`:255-258`) is a sealed interface with three variants:
`TagPredicate` (`:261-268` — `Field`/`FieldPos` the tag key, `Operator`/`OpPos` always `=`,
`Value`/`ValuePos` the field reference), `LogicalExpr` (`:273-278` — `Left`, `Operator`, `Right`)
and `NotExpr` (`:283-286`). `tasks/learnings.md:186-189` records that this interface fans out the
same way `ThenClause` does, through type switches that fail silently on a variant they have not
heard of; the recursion this story writes must be one dispatch point, not a second copy.

**How the parser builds it, and what that means for parentheses.** `parsePredicate`
(`internal/parser/parser.go:779-782`) consumes `where` and descends `parseOrExpr` → `parseAndExpr` →
`parseNotExpr` → `parsePrimary`, so `and` binds tighter than `or` and both are left-associative.
`parsePrimary` (`:844-866`) returns a parenthesised sub-expression **unwrapped** — there is no
grouping node — so `(tag(a = x) and tag(b = y))` and the same text without parentheses produce
identical trees, while `tag(a = x) and (tag(b = y) or tag(c = z))` keeps the `or` as the `and`'s
right operand. Conjunctive position is therefore exactly "reachable from the root through
`LogicalExpr` nodes whose operator is `and`", and it is decidable from the tree alone.

**The existing predicate walk to copy.** `collectPredicateTagKeys`
(`internal/linter/linter.go:346-364`) is the repository's one recursion over `PredicateExpr`: a type
switch returning the tag key from a `TagPredicate`, concatenating both sides of a `LogicalExpr`
regardless of operator, and descending through a `NotExpr`. It is the structural template, and the
difference this story's walk must make deliberate and visible is that it does **not** cross an `or`
or a `not`.

**The linter.** `internal/linter` is two files. `Lint(model *ast.Model)` (`linter.go:49-103`) builds
one model-wide map at the top — `flowCount` (`:57-62`) — then loops `model.Contexts`, gates four
DCB checks on `isDCBMode(ctx.Mode) || isMixedMode(ctx.Mode)` (`:71-89`), and calls
`checkSlice(ref.Slice, aggregateName, flowCount)` once per `ctx.SliceRefs()` entry (`:93-99`).
`descriptions.go` is a single `ruleDescriptions` map (`:10-31`, seventeen entries) behind
`RuleDescription(name)`. There is no rule registry, no severity table and no options parameter: a
rule is a function, a call site, one of `info`/`warning`/`errorEntry` (`:16-47`), and a description
entry. This story adds no call site of its own — it extends the DCB arm US-008 Task 6 writes.

**The two rules whose shape this arm inherits.** `checkQueryTooBroad` (`:367-391`) is the precedent
for skipping a command that declares no `decides_on`, and it is also why every `decides_on` in the
tree carries a `where`: it warns when `Predicate` is nil, so a predicate-less `decides_on` is
already reported and the measurement above found 9 of 9 commands carrying one. `checkOrphanTagKeys`
(`:271-343`) is the rule that returns nil when the feature is unused, collects across a context's
slices before deciding, and sorts findings by declaration position with a comment
(`:320-323`) saying why Go's map order would otherwise reorder them.

**The spec AST the arm reads.** `Spec` (`internal/ast/ast.go:95-104`) carries `Given []*SpecElement`,
`When *SpecElement` and `Then ThenClause`. `SpecElement` (`:106-109`) is a name and a position today
and is where US-010 open question 1 hangs the payload — on the element, not on the clause, which is
what gives `given` and `when` payloads the same shape. **This arm reads `Given` and `When` and never
`Then`**, so it dispatches on `ThenClause` nowhere and the five-type-switch trap
(`tasks/learnings.md:186-189`) does not reach it: US-007's `ThenView` and `ThenCommand` are additive
against every task here.

**Traversal.** `internal/ast/traverse.go` reconciles the two slice homes. `SliceRef` (`:19-25`)
pairs a slice with its `Context` and its `Aggregate` (nil for a `mode dcb` context's own slices),
`Context.SliceRefs()` (`:30-50`) returns both homes in source order, `Model.SliceRefs()` (`:66-77`)
composes them, and `AllSlices()` drops the pairing. `Lint` already consumes `ctx.SliceRefs()`.

**Where the validator has already been.** `decidesOnDiagnostics`
(`internal/validator/validator.go:451-471`) checks that each `decides_on` event exists and hands the
predicate to `predicateDiagnostics` (`:473-493`), which recurses the same three variants and
requires each `TagPredicate`'s `Field` to be a tag key on some listed event and its `Value` to be a
field on some listed event, through `anyEventDeclares` (`:140-148`). Two things this arm may lean
on as a result: in any model that validates, a tag predicate's `Value` names a real field of a real
`decides_on` event, and its `Field` names a real tag key. Nothing there says anything about the
command's own fields — decision 2.

**The pipeline.** `oracle.Run` (`internal/oracle/oracle.go:26-31`) is the one lex → parse → validate
→ lint chain and `Check` (`:35-38`) its diagnostics-only form; `RunValidate` and `RunLint`
(`internal/cli/lint.go:104-129`) both call it, which is why a warning is not free
(`tasks/learnings.md:466-469`). `emod lint --explain` resolves through `RunLintExplain`
(`internal/cli/lint.go:91-102`) into `linter.RuleDescription`, and the only thing covering the
description map is the hand-maintained `rules` list at `internal/cli/lint_test.go:627-645` — which
US-008 Task 5 already extends with `spec/given-outside-boundary`, so this story adds no entry to it.
`diagnostic.Entry.String()` (`internal/diagnostic/entry.go:32-37`) is the formatted line every
whole-line assertion compares: `<file>:<line>: [<rule>] <message>`.

**The models a new arm must not disturb, and where they are enumerated.** `oracle.Check`
zero-diagnostic leaves live in `internal/oracle/oracle_test.go`: "clean input" (`:26-110`, one leaf
per shared fixture, including the spec fixture at `:60-64`) and "documented models" (`:112-129`,
extracting every ` ```emod ` fence from `README.md` and `docs/dsl-reference.md` — seven blocks).
`internal/cli/validate_test.go` adds the `internal/parser/testdata/` leaf (`:37-49`) and the
`examplePaths` walk (`:52-86`), which splits `examples/` on the `_test.emod` suffix.
`internal/wasm/pipeline_test.go:13-36` embeds `billingModel` and asserts an empty envelope. The
group US-008 Task 4 creates for a fixture that legitimately warns is the shape to copy if any
sweep here comes back non-empty; the measurement says none will.

**The linter suite's shape.** `internal/linter/linter_test.go` is one `TestLint` umbrella split into
twelve top-level groups, four of them named for a single rule — `"dcb/query-too-broad"` (`:1895`),
`"dcb/single-tag-everywhere"` (`:2226`), `"dcb/orphan-tag-key"` (`:2807`),
`"automation/missing-todo-list"` (`:3345`) — with `reportedLines(diags)` (`:3617-3624`) the helper
whole-list comparisons go through. US-008 Task 5 opens the group for
`spec/given-outside-boundary`; this story's leaves belong inside it.

**The CLI lint fixtures.** Each `const …Emod` in `internal/cli/lint_test.go` carries a comment
naming precisely which rules it fires (`singleTagDCBEmod` at `:18-21`, `automationWithoutViewEmod`
at `:135`) and its leaves assert `require.Len(t, entries, 1)`. A DCB fixture for this arm has an
unusually long list to keep quiet: `dcb/untagged-event` (tag every event), `dcb/orphan-tag-key`
(reference every declared tag key from some predicate), `dcb/query-too-broad` (every `decides_on`
states a `where` and at most five events), `dcb/single-tag-everywhere` (at least two distinct tag
keys across the context's predicates), `aggregate-in-dcb-mode` (declare no aggregate),
`orphan-command`/`orphan-event` (a full `flow` per slice), `clickbait-event` (more than a lone id
field per event), `command-past-tense`, `left-chair` (no command in three flows), plus the four
`spec/*` rules US-008 adds — every command specced, every specced command carrying a rejection,
every declared invariant exercised, and every `given` inside the type-level boundary.

**Where the payload fixture and its transcriptions live.** `tasks/us-010-state-example-payloads-in-specs.md`
Task 2 adds the payload-carrying model to `internal/test/fixtures.go` beside `SpecLibraryLending`
(`:416-572`), with its parsed-model accessor in `internal/test/models.go`, a hand-written
transcription of every payload it states, a twin that clears them and a getter that reads them back
— the six-part kit `tasks/learnings.md:216-219` describes. The task list does not fix the constant's
name, so Task 1 below names it by role. `copyWithEditedSlices` (`internal/test/fixtures.go:1257`)
and `editedCopies` (`:1275`) are shallow and leave a nil list nil on purpose.

---

## Tasks

### Task 1: State a tagged field in both payloads of the shared payload fixture

**Behavior:** The payload-carrying shared fixture US-010 Task 2 adds gains the shape this story's
rule reads: a `mode dcb` context whose commands declare the fields their `decides_on` predicates
tag, and whose specs state one of those fields with the *same* value in the `when` payload and in a
`given` payload. Two predicate shapes are covered — one command whose `where` states a single
`tag(key = field)`, one whose `where` states two joined by `and` — because Tasks 2 and 3 each need a
checked-in model to be silent about. Nothing about the tool changes; `oracle.Check` still reports
nothing for the fixture, and the transcription, twin and getter that mirror it move with it.

**Why it comes first:** measured over the 20 models the repository ships as valid, **0 of 16** tag
predicates name a field their own command also declares, so no checked-in model can legally state
the tagged field in a `when` payload at all. Without this task the arm's "matching values produce no
warning" criterion could only close against a source string the rule's own task wrote, and the
sweep in Tasks 2 and 3 would prove the arm inert without ever proving it inert for the right reason.
Landing the fixture ahead of the rule also keeps the tree green at both commits — this task is a
no-op for diagnostics today, and the rule then finds the fixture already clean, the same
fixture-before-rule sequencing US-008 Task 1 uses.

**Acceptance Criteria:**
- [x] The payload-carrying fixture in `internal/test/fixtures.go` declares a `mode dcb` context with
      two command slices, one whose command's `decides_on` states a `where` with a single
      `tag(key = field)` predicate and one whose command's states two joined by `and`, and each of
      those commands declares in its own `fields` block the field its predicate tags
- [x] Each of those two commands is exercised by a spec whose `when` payload states the tagged field
      and whose `given` payload states the same field with a value equal in both source text and
      literal kind, so the two payloads agree; the `given` event is one the command's `decides_on`
      events list names
- [x] One spec in that context leaves the tagged field out of its `given` payload while stating it
      in its `when` payload, and one leaves it out of its `when` payload while stating it in a
      `given` payload — the two shapes the story's third criterion requires the rule to stay silent
      on, present in the model rather than only in a test source
- [x] The tagged field's declared type is the same on the command and on the event in every pair the
      fixture states, so no pair depends on the cross-kind decision, and no stated value equals a
      construct name, a field name or a bare small integer
- [x] The context declares at least two distinct tag keys across its predicates, every tag key
      declared on any of its events is referenced by some predicate, every event carries at least one
      tag and more than one field, and every slice states a full `flow` — so `dcb/single-tag-everywhere`,
      `dcb/orphan-tag-key`, `dcb/untagged-event`, `dcb/query-too-broad`, `clickbait-event`,
      `orphan-command` and `orphan-event` all stay quiet
- [x] `oracle.Check` over the fixture returns an empty diagnostic list, so its leaf in
      `internal/oracle/oracle_test.go` "clean input" is unchanged and still passes
- [x] The hand-written payload transcription US-010 Task 2 ships restates every payload the fixture
      now states, in declaration order across both slice homes, and the getter over the parsed
      fixture equals it; the twin still clears every payload and the getter over the twin is empty
- [x] The only expected values that move are those restating this fixture's own text — its
      transcription in `internal/test/fixtures.go`, its canonical formatted constant in
      `internal/cli/fmt_test.go`, and any read-back list in `internal/export`, `internal/formatter`,
      `internal/diagram` or `internal/glossary` that transcribes it. `git diff --stat` names no
      golden, canonical constant or transcribed list belonging to a fixture this task does not edit,
      and `test.SpecLibraryLending` in particular is unchanged
- [x] `task test:unit` passes, and `internal/parser`, `internal/validator`, `internal/formatter`,
      `internal/export`, `internal/glossary`, `internal/diagram`, `internal/oracle` and
      `internal/cli` are green with no test skipped or weakened
- [x] `internal/linter` and `internal/linter/descriptions.go` are untouched — this task adds no rule
      and no arm

**Affected Files/Modules:**
- `internal/test/fixtures.go` — the payload-carrying fixture US-010 Task 2 adds, and its payload
  transcription
- `internal/cli/fmt_test.go` — the canonical formatted constant US-010 Task 5 pins for that fixture
- `internal/export/export_test.go`, `internal/glossary/glossary_test.go`,
  `internal/diagram/contract_test.go`, `internal/formatter/formatter_test.go` — any leaf that reads
  the fixture's payloads back against a transcribed list

**Patterns to Follow:**
- The DCB shapes to copy for a clean `mode dcb` context: `test.SpecLibraryLending`'s "Reading Room"
  context (`internal/test/fixtures.go:505-571`) for the tags, predicate and flow arrangement, and
  `examples/dcb_model.emod:38-42` for a command whose `decides_on` states two tag predicates joined
  by `and`. Note what neither of them does — declare the tagged field on the command — which is
  precisely what this task adds
- `tasks/learnings.md:481-484` on wording a change-set criterion for a task that edits a shared
  fixture, and `:151-154` on a `mode dcb` fixture being the usual `oracle.Check` tripwire
- `tasks/learnings.md:191-194` on a spec not being a reference, so every slice still needs its flow
- `tasks/learnings.md:91-94` on exercising an omitted optional part mid-block rather than as the
  last entry, which is what the third criterion's two omission shapes are for
- `tasks/learnings.md:206-209` on proving a twin actually differs, and `:216-219` on `editedCopies`
  being shallow, so the twin's edit has to reach inside a slice's specs' elements
- The fixture's own comment header states what each placement is for:
  `internal/test/fixtures.go:416-422` is the model

**Testable:** Yes — the existing suites are the test, and the fixture's own transcription and
`oracle.Check` leaf assert the new shape is read back and stays clean.

**Verification:** `task test:unit` passes; `oracle.Check` over the fixture is empty; `git diff` shows
movement only in the fixture and the values that transcribe it.

**Depends on:** None within this story. Requires US-010 Tasks 2 and 5 (the payload fixture and its
canonical formatted constant) to be delivered.

---

### Task 2: Report a `given` payload value the command's tag predicate excludes

**Behavior:** `emod lint` reports `spec/given-outside-boundary` at warning severity when a spec in a
context-level slice names a `given` event whose payload states a different value for a tagged field
than the `when` command's payload does. The `when` command's `decides_on` states a `where` whose
whole predicate is one `tag(key = field)`; the `given` event's type is one the `decides_on` events
list names, so the type-level arm is satisfied and only the value disagrees. The diagnostic sits on
the disagreeing value inside the `given` payload and carries the rule's third message text, naming
the field, the tag key, both stated values and the command. Matching values report nothing, and so
does every shape in which the comparison has no basis.

**Acceptance Criteria:**
- [x] A spec in a context-level slice, whose `when` names a command whose `decides_on` states one
      `tag(key = field)` predicate, and whose `given` names an event that `decides_on` lists, where
      both payloads state that field with the same literal kind and different values, produces
      exactly one diagnostic at `diagnostic.Warning`, rule name `spec/given-outside-boundary`,
      positioned at the value inside the `given` payload, whose whole formatted line names the
      field, the tag key, both values and the command
- [x] The same shape with both payloads stating that field with the same kind and the same value
      produces no diagnostic
- [x] The arm's message text differs from both texts US-008 Tasks 5 and 6 give the rule, and every
      leaf asserting any of the three compares the whole formatted diagnostic line, so a leaf cannot
      pass against another arm's wording
- [x] A `given` payload omitting the tagged field produces no diagnostic, and a `when` payload
      omitting it produces none either, however many other fields either payload states — the story's
      third criterion, asserted separately for each side
- [x] A `given` element stating no payload at all, and a `when` element stating none, each produce no
      diagnostic
- [x] Two literals of different kinds are not compared: a `when` payload stating the tagged field as
      a string and a `given` payload stating it as a number produce no diagnostic, asserted for the
      string/number pair and for a boolean against each of the other two
- [x] Two number literals equal in value but not in source text are one value: `12.50` against
      `12.5`, and `007` against `7`, each produce no diagnostic, while two numbers of different value
      produce one
- [x] Two string literals differing only in case, and two differing by surrounding whitespace, are
      each different values and produce a diagnostic — a string compares by its exact text
- [x] A payload stating the tagged field more than once produces no diagnostic, asserted for a
      repeat on the `given` side and for one on the `when` side
- [x] A `when` command stating no `decides_on` produces no diagnostic however its payloads and the
      `given` payloads disagree, and a command stating `decides_on` with no `where` predicate
      produces none either
- [x] A `given` event whose name the `when` command's `decides_on` events list does not carry
      produces exactly one diagnostic — US-008 Task 6's type-level one — and no value-level
      diagnostic, however far its payload value is from the `when` payload's
- [x] A spec whose `when` is absent, and a spec whose `when` names an event rather than a command,
      produce no diagnostic
- [x] A spec in an aggregate-nested slice produces no value-level diagnostic whatever its payloads
      state and whatever the enclosing context's mode string is, and a leaf asserts a model carrying
      the disagreeing shape in both slice homes reports once, for the context-level home only
- [x] A `where` predicate stating two tag predicates joined by `and`, one of them disagreeing,
      produces no diagnostic from this task — Task 3 owns conjunctions, and a leaf asserts this task
      leaves them alone
- [x] Several disagreeing `given` elements in one model come back in declaration order, asserted as
      one comparison over the whole list of formatted lines
- [x] `linter.RuleDescription` still answers for `spec/given-outside-boundary` with a single
      description that now covers the value-level arm alongside the two type-level ones,
      `emod lint --explain spec/given-outside-boundary` prints it and returns no error, no second
      rule name is introduced, and the hand-maintained rule list at
      `internal/cli/lint_test.go:627-645` is unchanged
- [x] A CLI lint fixture declares a `mode dcb` context whose commands each carry a `decides_on` with
      one tag predicate over the field the command declares, gives every command a spec and a
      rejection, exercises every invariant it declares, tags every event and references every tag
      key from some predicate across at least two distinct keys, and states one `given` payload
      value the `when` payload contradicts — tripping this rule and no other, asserted with a length
      of exactly one entry, with a declaring comment naming the rule and the arm it is written to
      fire
- [x] `cli.RunLint` and `cli.RunValidate` both return an error for that fixture, the text output
      names the rule, the field and the line the `given` payload is written on, and `-f json`
      reports `"severity": "warning"` with exit code 1
- [x] The arm reads `Spec.Given` and `Spec.When` only and never `Spec.Then`, so a `then` variant
      US-007 adds is neither read nor a silent failure, and no type switch over `ast.ThenClause`
      appears in the change
- [x] Every shared fixture in `internal/test/fixtures.go` — the Task 1 fixture included — every file
      under `examples/` the repository ships as valid, every fixture under
      `internal/parser/testdata/`, every ` ```emod ` fence in `README.md` and `docs/dsl-reference.md`
      including the `mode dcb` fence at `:175`, and `billingModel` in
      `internal/wasm/pipeline_test.go` still produce zero diagnostics from `oracle.Check`
- [x] `internal/parser`, `internal/validator`, `internal/formatter`, `internal/export`,
      `internal/diagram` and `internal/lsp` are untouched: this arm reads the AST and changes no
      other stage

**Affected Files/Modules:**
- `internal/linter/linter.go` — the value comparison inside the DCB arm US-008 Task 6 writes, reusing
  the model-wide command index that arm already builds to resolve a spec's `when`
- `internal/linter/descriptions.go` — the `spec/given-outside-boundary` entry, widened to state the
  value-level arm
- `internal/linter/linter_test.go` — leaves inside the group US-008 Task 5 opens for the rule: the
  disagreement, the agreement, every silence rule, both slice homes, ordering, severity and position
- `internal/cli/lint_test.go` — the DCB fixture and its text and JSON leaves

**Patterns to Follow:**
- The arm this extends and the index it already has: `tasks/us-008-lint-spec-coverage-and-boundary-assumptions.md`
  Task 6, whose production change is the DCB arm of the same rule plus a model-wide command index
- `checkQueryTooBroad` (`internal/linter/linter.go:367-391`) for reading a command's `decides_on` and
  skipping a command that declares none, and `checkOrphanTagKeys` (`:301-304`) for the
  "the feature is unused, so say nothing" early return
- `predicateDiagnostics` (`internal/validator/validator.go:473-492`) for which half of a
  `TagPredicate` is the tag key and which the field reference — it is the only place in the tree
  that states the distinction, by checking `Field` against tag keys and `Value` against event fields
- `warning(pos, rule, msg)` (`internal/linter/linter.go:27-36`) and `checkOrphanTagKeys` (`:320-335`)
  for position-sorted findings and the comment at `:320-323` saying why
- `test.SpecLibraryLending`'s "Release Desk" slice (`internal/test/fixtures.go:540-570`) for the
  no-value-comparison-possible shape this arm must stay silent on, and its "Claim Desk" slice
  (`:507-539`) for the no-`decides_on` shape
- `tasks/learnings.md:476-479` on pinning a rule with more than one text by whole formatted lines —
  this arm makes it three
- `tasks/learnings.md:471-474` on why a lint fixture is never the minimal model, and the fuller
  tripwire list for a DCB fixture in this document's Codebase Context
- `tasks/learnings.md:166-169` on `RuleName` obliging a description — here the obligation is to
  widen the existing entry rather than add one
- `tasks/learnings.md:6-9` on asserting the distinguishing message text in a CLI diagnostic leaf,
  and `:136-139` on a second `require.Contains` being shadowed by the first when one message names
  several things — which this message does, naming a field, a key, two values and a command
- `tasks/learnings.md:466-469` on sweeping every checked-in model before a warning-level rule lands

**Testable:** Yes — through `linter.Lint`, `cli.RunLint`, `cli.RunLintExplain` and `oracle.Check`.

**Verification:** `task test:unit` passes; `go run ./cmd/emod lint` over each model the repository
ships as valid exits 0; `go run ./cmd/emod lint --explain spec/given-outside-boundary` prints the
widened description.

**Depends on:** Task 1 (the shared payload fixture must already carry the matching-value shape).
Requires US-008 Tasks 5 and 6 (the rule, its name, its severity, its description entry, its group in
the linter suite and its arm-selection rule) and US-010 Tasks 1, 3 and 4 (the payload node, and the
two validation rules that make a payload field name and its literal kind meaningful).

---

### Task 3: Check every tag predicate a conjunction states, and no other

**Behavior:** A `when` command whose `where` states several `tag(key = field)` predicates joined by
`and` is checked once per predicate: every tagged field whose two payloads disagree produces its own
diagnostic, at that field's value inside the `given` payload. A tag predicate reached through an
`or` or through a `not` is not checked at all — under `or` a single disagreement does not put the
event outside the query, and under `not` it argues the opposite — so the rule stays silent there
rather than guessing. The arm's claim stays exact: every warning it emits names a predicate the
command's query requires to hold.

**Acceptance Criteria:**
- [x] A `where` stating two tag predicates joined by `and`, where the two payloads disagree on the
      field one of them tags and agree on the other's, produces exactly one diagnostic, naming the
      disagreeing field
- [x] Where the two payloads disagree on both fields, two diagnostics are produced, one per field,
      each positioned at that field's value inside the `given` payload, asserted as one comparison
      over the whole list of formatted lines so both position and order are pinned
- [x] Three tag predicates joined by two `and`s are all checked, so the walk is a recursion rather
      than a single unwrapping of the root
- [x] A `where` stating two tag predicates joined by `or` produces no diagnostic from this rule,
      however the payloads disagree — a leaf states this over a model identical to the two-`and` one
      but for the operator
- [x] A tag predicate under `not` produces no diagnostic, asserted both for `not` at the root and for
      an `and` one of whose operands is a `not`
- [x] An `and` one of whose operands is an `or` checks the `and`'s other operand and neither of the
      `or`'s, so a model whose disagreement sits inside the `or` reports nothing and one whose
      disagreement sits in the conjunctive operand reports once
- [x] Parentheses change nothing the parser does not already record: a model writing its conjunction
      with redundant parentheses produces the same diagnostics as the same model without them
- [x] Two tag predicates naming the *same* field with different tag keys count once per predicate, so
      a disagreement on that field reports once per predicate that tags it, and a leaf states which
      it chose
- [x] Every silence rule Task 2 established still holds under a conjunction: a tagged field either
      payload omits, states twice, or states at a different literal kind contributes no diagnostic
      while its sibling predicate's field still does
- [x] The single-predicate behaviour Task 2 delivers is unchanged, and the three story criteria still
      close: a leaf over a root-level lone `tag(key = field)` reports exactly as it did
- [x] The message text is the one Task 2 introduced, unchanged, and the rule name and description are
      unchanged — this task adds no fourth text, no rule name and no description entry
- [x] A CLI lint fixture states a `mode dcb` context whose offending command's `decides_on` joins two
      tag predicates with `and` and whose spec disagrees on exactly one of the two tagged fields,
      tripping this rule once and no other rule at all — asserted with a length of exactly one entry
      — with a declaring comment naming the rule and why the conjunction matters
- [x] `cli.RunLint` and `cli.RunValidate` both return an error for that fixture, the text output
      names the rule and the disagreeing field, and `-f json` reports `"severity": "warning"` with
      exit code 1
- [x] `oracle.Check` over the Task 1 fixture returns an empty diagnostic list: its conjunctive
      command's spec states the tagged field with the same value in both payloads
- [x] Every other shared fixture, every file under `examples/` the repository ships as valid
      including `examples/dcb_model.emod`, every fixture under `internal/parser/testdata/`, every
      ` ```emod ` fence in `README.md` and `docs/dsl-reference.md` including the `mode dcb` fence at
      `:175`, and `billingModel` in `internal/wasm/pipeline_test.go` still produce zero diagnostics
      from `oracle.Check`
- [x] The predicate walk is one recursion over `ast.PredicateExpr` with a single dispatch point, so a
      fourth variant added to the interface is a change at one site rather than a silent omission

**Affected Files/Modules:**
- `internal/linter/linter.go` — the recursion that selects tag predicates in conjunctive position,
  feeding the comparison Task 2 wrote
- `internal/linter/linter_test.go` — the conjunction leaves inside the rule's group, including the
  `or` and `not` silences and the both-fields-disagree ordering leaf
- `internal/cli/lint_test.go` — the conjunctive fixture and its leaves

**Patterns to Follow:**
- `collectPredicateTagKeys` (`internal/linter/linter.go:346-364`) is the structural template — the
  repository's one recursion over `ast.PredicateExpr`, a type switch over the three variants. The
  deliberate difference here is that it descends `LogicalExpr` regardless of operator and descends
  `NotExpr`, and this walk must do neither; make that visible rather than inferable
- `parseOrExpr` / `parseAndExpr` / `parseNotExpr` / `parsePrimary`
  (`internal/parser/parser.go:779-866`) for what the tree can hold: `and` binds tighter than `or`,
  both are left-associative, and `parsePrimary` returns a parenthesised sub-expression unwrapped, so
  parentheses leave no node and conjunctive position is decidable from the tree alone
- `examples/dcb_model.emod:38-42` for a real command joining two tag predicates with `and`, and
  `test.SpecLibraryLending`'s `ReleaseDesk` (`internal/test/fixtures.go:541-549`) for the same shape
  in a fixture
- `checkOrphanTagKeys` (`internal/linter/linter.go:320-335`) for sorting findings gathered from more
  than one place, and `tasks/learnings.md:181-184` on why an unsorted walk reports in field order
  rather than source order
- `tasks/learnings.md:476-479` on comparing whole formatted lines, which is what pins two diagnostics
  from one `given` element to two distinct positions
- `tasks/learnings.md:471-474` on the fixture keeping every other rule quiet

**Testable:** Yes — through `linter.Lint`, `cli.RunLint` and `oracle.Check`.

**Verification:** `task test:unit` passes; `go run ./cmd/emod lint` over every model the repository
ships as valid exits 0, `examples/dcb_model.emod` included.

**Depends on:** Task 2 (the comparison, the message text, the widened description and the rule's
group in the linter suite all land there).

---

## Summary

**Three tasks.**

**Ordering rationale — fixture first, then the arm, then the predicate shapes it reaches.** Task 1
is fixture-only and a no-op for diagnostics at the commit it lands in, following US-008 Task 1's
sequencing; it exists because the measurement says the repository has no model in which the rule
could fire even after payloads land — 0 of 16 tag predicates name a field their own command
declares — so the "matching values produce no warning" criterion has nothing checked-in to close
against without it. Task 2 delivers all three of the story's criteria for the simplest predicate the
language has, a lone `tag(key = field)`, and carries the whole of the arm's silence contract: the
two omission cases, the cross-kind case, the duplicate-field case, the no-`decides_on` and
no-predicate cases, the type-arm-owns-it case, the `when`-names-an-event case, and the
aggregate-home case. Task 3 widens the reach from that one predicate to every predicate a
conjunction states, and states in the same commit which predicates are deliberately *not* reached —
the `or` and `not` arms, where a single disagreement is inconclusive. Splitting Tasks 2 and 3 keeps
the subtle half in a commit of its own: at Task 2's commit a conjunction simply yields no
value-level diagnostic, which is a silence rather than a wrong answer, so the tree is green at both.

**Story acceptance criteria, all three covered, none deferred:**

| Story criterion | Task |
|---|---|
| A `given` event whose payload states a different value for the tagged field than the `when` payload triggers the `spec/given-outside-boundary` warning | 2 (single predicate), 3 (every predicate a conjunction states) |
| Matching values produce no warning | 1 (the checked-in witness), 2 |
| When either payload omits the tagged field, only the type-level check from US-008 applies — no value-based warning | 2, 3 (under a conjunction) |

**Four things a reader of this list should carry forward.**

First, **the tagged field is `TagPredicate.Value`, and nothing requires the command to declare it.**
The validator requires a predicate's field reference to be declared on a `decides_on` *event* only
(`internal/validator/validator.go:480`), while US-010 requires a `when` payload's field to be
declared on the *command*. The two requirements meet nowhere in the repository today, which is
Task 1's entire reason for existing and the single fact most likely to surprise an implementer.

Second, **the arm fires zero times across every checked-in model, and the measurement says why
twice over** — no model states a payload, and no model's DCB command declares the field its
predicate tags. The interesting work is constructing the fixture, not moving models out of the way.
The `docs/dsl-reference.md:175` fence is the nearest thing to a live surface: its `ClaimDesk` is the
only command in the tree declaring its own tagged fields, and US-008 Task 4 gives that fence
rejection specs, so both Tasks 2 and 3 sweep it by name.

Third, **silence is the answer wherever the comparison has no basis**, and the story's third
criterion is one instance of a rule with eight. The cross-kind decision is the one worth reading
twice: `"101"` against `101` is reachable — the two payloads sit on different constructs and nothing
requires a command's field and an event's field of one name to share a type — and it is decided as
silence, because the rule cannot tell a different value from a different spelling and a
kinds-differ pair is a modelling problem whose message would name a boundary. The counterpart is
that numbers compare numerically, so `12.50` and `12.5` are one value rather than a false positive.

Fourth, **this story adds no rule name, no severity and no description entry.** It widens the one
`spec/given-outside-boundary` entry US-008 Task 5 registers, gives the rule its third message text,
and leaves the hand-maintained rule list at `internal/cli/lint_test.go:627-645` untouched. An
implementer reaching for a new rule name has misread the story.

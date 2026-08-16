# US-015: Navigate and complete the new constructs in the editor

## Progress
- [x] Task 1: Answer hover on every declaration, naming its kind and where it was declared
- [x] Task 2: Show a construct's description on hover
- [x] Task 3: Show an invariant's prose where it is declared and where a spec rejects by it
- [x] Task 4: Jump to and from the events and commands a spec names
- [x] Task 5: Jump to and from the invariant a spec's `then rejected` names
- [x] Task 6: Offer the invariants in scope after `then rejected`
- [x] Task 7: Offer a spec's entries and the names its `given`, `when` and `then` accept
- [x] Task 8: Offer a payload's field names, scoped to the construct the reference names
- [ ] Task 9: See a flow rejection edge from hover, go-to-definition and find-references

---

## Story Reference

`user-stories/specs-and-metadata.md` → **US-015: Navigate and complete the new constructs in the
editor** (fifteenth story of "Specs, Invariants, and Model Metadata", lines 193-203). Design notes:
`docs/proposals/specs-and-metadata-proposal.md` — the tooling phase. The story's own Goals frame it:
"Model authors can attach Given-When-Then specs to slices", "Business rules live in the model as named
invariants; spec rejections and flow rejection edges reference them by name", and "Every construct can
carry a description" — this story is the editor catching up to a language three earlier stories
already teach the parser.

**Dependencies, and their state.** The story declares US-002, US-006, US-009 and US-010.

- **US-002** (`description` on every construct) and **US-006** (`spec`, `given`, `when`, `then`,
  `rejected`, and US-005's `invariant`) are **delivered on main**. `ast.Spec`, `ast.SpecElement`,
  `ast.ThenEvents`, `ast.ThenRejected` (`internal/ast/ast.go:95-126`), `ast.Invariant` (`:68-74`) and
  a `Description`/`DescriptionPos` pair on eleven constructs are all in the tree, and
  `test.SpecLibraryLending` (`internal/test/fixtures.go:423`) and `test.DescribedHotelReservation`
  (`:101`) are the shared models that state them.
- **US-009** (flow rejection edges) is **decomposed but not implemented**
  (`tasks/us-009-show-rejection-paths-on-the-timeline.md`). Only **Task 9** below depends on it, and it
  names the dependency as hard: `ast` carries no rejection node today, so nothing here can read one.
- **US-010** (example payloads) is **decomposed but not implemented**
  (`tasks/us-010-state-example-payloads-in-specs.md`). Only **Task 8** below depends on it, and it
  names the dependency as hard: `ast.SpecElement` carries `Name`/`NamePos` and nothing else today.

The other seven tasks depend on nothing beyond `main`, which is why the two gated tasks sit last.

**In scope:** `internal/lsp`, and nothing else. Hover over every declaration the language has, carrying
each construct's `description`; hover over an invariant, at its declaration and at the `then rejected`
that names it, carrying its statement; go-to-definition and find-references across a spec's `given`,
`when` and `then` references and across its invariant reference; keyword completion inside a `spec`
body and value completion in the four slots a spec has (`given`, `when`, `then`, `then rejected`);
field-name completion inside a payload's braces; and the same four surfaces over a flow rejection
edge. Carried along because the story's criteria cannot be met without them: the quoted-name column
offset (`internal/lsp/semantictokens.go:57-63` calls it out; `cursor.onName` and `nameRange` have never
had to deal with it, because every name hover answers for today is an unquoted identifier), and the
scope rule that decides which `invariant` an identifier resolves to, which the LSP has never had to
model.

**Out of scope**, each named with its owner:

- **The `spec`, payload and rejection-edge grammar itself** — US-006 (delivered), US-010 and US-009.
  No task here edits `internal/lexer`, `internal/parser`, `internal/ast`, `internal/validator` or
  `internal/formatter`. Tasks 8 and 9 *read* AST shapes those stories add and add none of their own.
- **Keyword hover entries for `type` and `after`** — US-012 and US-013.
  `internal/lsp/keywords_test.go:18` `TestKeywordCoverage/hover` iterates `lexer.Keywords()` and
  requires `lsp.GetHover` to answer for every one, so the story that adds the keyword must add its
  hover text in the same change or CI fails. Verified on this tree: `lexer.Keywords()` reports **38**
  spellings, `type` and `after` are not among them, and `keywordDescriptions`
  (`internal/lsp/hover.go:10-49`) already describes all 38 — including `invariant`, `spec`, `given`,
  `when`, `then`, `rejected` and `description`, which US-005 and US-006 added. **No task here adds or
  edits a single `keywordDescriptions` entry**, and criterion 1 is not about keywords at all.
- **Keyword completion for `type` and `after`** — also US-012 and US-013, argued below under "Open
  questions, decided" item 2.
- **The formatter** (US-014), **diagrams** (US-016), **syntax highlighting** (US-017) and **examples
  and the DSL reference** (US-018). No file under `editors/`, `docs/`, `examples/` or
  `internal/formatter` is touched, and `internal/lsp/protocol.go`'s `textDocument/formatting` handler
  delegates to `internal/formatter` unchanged.
- **Semantic tokens.** `GetSemanticTokens` (`internal/lsp/semantictokens.go:90`) paints declaration
  names only — contexts, aggregates, commands, events, views, actors — and paints no reference site at
  all, so a spec's `given` element is unpainted today and stays unpainted. No criterion of this story
  mentions highlighting, and the story that does (US-017) names the VS Code extension and the
  tree-sitter grammar, not the LSP's token legend. `internal/lsp/semantictokens.go` is untouched.
- **`internal/test/fixtures.go`.** Every model these tasks need already exists on main
  (`SpecLibraryLending`, `DescribedHotelReservation`, `HotelReservation`,
  `AutomationReadsLibraryLending`, `InvariantLibraryLending`) or is shipped by the story the gated task
  waits on (US-009 Task 2, US-010 Task 2). The two documents no shared fixture provides — a model
  declaring the *same* invariant name in two scopes, and one whose specs name a construct the model
  does not declare — are authored inside the `internal/lsp` test files that need them, the way
  `hover_test.go`'s `automationDoc` and `definition_test.go`'s `testDoc` already are. Editing a shared
  fixture would move six downstream transcriptions for no gain (`tasks/learnings.md:481-484`).

**Pre-existing defect, recorded and deliberately not fixed here.** `ConvertDiagnostics`
(`internal/lsp/diagnostics.go:31-36`) switches on `diagnostic.Severity` with a `case Warning` arm and a
`default` arm, and `diagnostic.Info` (`internal/diagnostic/entry.go:10`) falls through the `default`
onto `SeverityError`. So every info-severity rule renders in the editor as a red error squiggle. This
is already true of `dcb/single-tag-everywhere` (`internal/linter/linter.go:21` sets
`Severity: diagnostic.Info`), and US-008's and US-009's new info rules will inherit it the moment they
land. **It is named here so that an `internal/lsp` diff from this story is not misread as introducing
it.** It is out of scope, and the case for that is not "the criteria do not mention it": `protocol.go`
declares only `SeverityError = 1` and `SeverityWarning = 2`, so fixing it means declaring an
informational severity in the LSP protocol layer *and* changing how an existing, shipped rule renders
for every user — a visible behaviour change to a surface no task here otherwise touches, with its own
regression risk in `internal/lsp/diagnostics_test.go`. That belongs in its own change, alongside
whichever story cares that its info rules read as advice.

**Open questions, decided.** The five criteria fix what must work and leave the shape of nearly all of
it open. Each decision below is taken against behaviour measured on this tree, and each is chosen so
that US-009, US-010 and US-012/US-013 stay additive.

1. **"Any construct" in criterion 1 means a construct's *declaration*, not every site that names it.**
   Measured: `GetHover` answers today for a command, event or view *declaration name* only
   (`internal/lsp/hover.go:67-84`), and returns nil on a `subscribes` entry, on an automation's `on`,
   on a spec's `given` element and on a spec's `when` — every reference site in the language. Widening
   hover to reference sites is a different feature from the one criterion 1 asks for, it would double
   the surface of Tasks 1 and 2, and go-to-definition is what a reference site already offers. The one
   exception is deliberate and comes from the story itself: criterion 2 names `rejected <name>` — a
   reference site — explicitly, so Task 3 answers there and Task 9 answers on the flow edge. Nothing
   else does.
2. **`type` and `after` are not added to any completion list here.** `TestKeywordCoverage/completion`
   (`internal/lsp/keywords_test.go:59-90`) iterates `keywordBlocks` and requires every offered label to
   be a spelling `lexer.Keywords()` reports. `type` and `after` are not keywords on this tree — US-012
   and US-013 add them — so offering either before its story lands **fails CI**, and offering it after
   would make this story's tasks depend on two stories it does not declare as dependencies. The
   converse rule does not bind: the drift suite does not require every keyword to be *offered*, so an
   unoffered keyword breaks nothing. The gap is real and is recorded here so it is not lost: after
   US-012 and US-013 land, `keywordsFor(ctxEvent)` and `keywordsFor(ctxAutomation)`
   (`internal/lsp/completer.go:231-237`) will each be one entry short, and no test in the repo will say
   so. It belongs to whichever of those two stories lands second, or to a one-line follow-up.
3. **A quoted name's stored position points at the opening quote, and the LSP has never had to know.**
   `entries.addQuoted` (`internal/lsp/semantictokens.go:57-63`) exists precisely because
   `Context.NamePos`, `Aggregate.NamePos`, `Slice.NamePos`, `Trigger.NamePos`, `Model.NamePos` and
   `Actor.NamePos` point at the `"` rather than at the name's first character, while `Command.NamePos`,
   `Event.NamePos` and `View.NamePos` point at the identifier. `cursor.onName` and `nameRange`
   (`internal/lsp/position.go:20-54`) assume the second shape, because the three constructs hover
   answers for today are all identifiers. Task 1 is the first hover work to cross that line, so it owns
   the distinction; the criterion is that the reported range covers the name and neither quote.
4. **An invariant reference resolves in one scope, and that scope is the validator's.**
   `invariantScopes` (`internal/validator/validator.go:220-230`) makes a context's own invariants and
   each of its aggregates' *separate* resolution scopes — an identifier declared in one neither hides
   nor resolves against the same identifier in another, not even between an aggregate and the context
   enclosing it. The LSP's existing name resolution is `declaredNames`
   (`internal/lsp/model.go:72-92`), a flat `map[nameKind]map[string]ast.Position`, and a flat map
   cannot represent two scopes declaring `OneCopyPerLoan`: one silently wins, and the editor would jump
   to a declaration `emod validate` reports as unresolved. So invariants get their own scope-aware
   resolution in `internal/lsp/model.go` and **do not join `nameKind`, `declaredNames`,
   `declaredNamesInOrder` or `referenceTargetKinds`**. Task 3 builds it; Tasks 5, 6 and 9 consume it.
5. **A spec's `given`, `when` and `then` references *do* join `referencesIn`, and that is what makes
   find-references list them.** `referencesIn` (`internal/lsp/model.go:101-134`) is the single list both
   `GetDefinition` and `GetReferences` read, so one `add(...)` per spec site delivers jump and listing
   together — the same one-line shape US-009 (triggers-and-automations) Task 5 used for an automation's
   `reads` (`tasks/learnings.md:441-444`). This is *not* in tension with "A spec is not a reference"
   (`tasks/learnings.md:191-194`): that learning is about `index.referencedCommands` in
   `internal/validator`, which answers "is this construct exercised" and must keep ignoring specs so
   `orphan-command` keeps firing. The LSP answers "what sites name this identifier", and a spec plainly
   does. Task 4 states the non-interference as a criterion.
6. **A spec's `when` resolves against commands *and* events; `given` and `then` against events only.**
   `declaresCommandOrEvent` (`internal/validator/validator.go:136`) backs `when` because a command
   slice's `when` names a command while an automation slice's names the triggering event, and
   `undeclaredSpecEvents` (`:359`) checks `index.eventNames` alone
   (`tasks/learnings.md:201-204`). Tasks 4 and 7 mirror that asymmetry exactly rather than tidying it:
   go-to-definition from a `when` lands on either kind, and `when` completion offers both, while
   `given` and `then` offer and resolve events only.
7. **Completion after `rejected` covers the spec's `then rejected` and not the flow rejection edge.**
   The story's own wording is the boundary: criterion 2 says "in a spec **or** on a flow rejection
   edge", criterion 4 says "including flow rejection edges" and criterion 5 says "the specs **and** flow
   edges" — criterion 3 says none of that, it says "invariant names after `rejected`". The mechanism
   agrees: US-009's entry is `command -> rejected: <CommandName> -> <invariantName>`, so the identifier
   immediately after `rejected` is a **command** name and the invariant sits after a second `->`.
   `valueSlotBefore` (`internal/lsp/completer.go:178-198`) scans the typed words backwards for a slot
   keyword and cannot tell the two positions apart; making it able to would mean teaching the line
   scanner the entry's shape, which is a feature the story does not ask for. Recorded as a known gap:
   a cursor anywhere on a rejection-edge line offers the enclosing block's keyword list.
8. **A `flow` body gets no completion arm.** It has none today (verified: a cursor inside `flow { }`
   offers `model actor context`), no criterion asks for one, and the entry it would complete is an
   arrow-and-colon shape rather than a keyword list. Item 7's boundary would be pointless if a flow arm
   landed anyway.
9. **A translation's own event hovers; it still does not navigate.** `Translation.Event`
   (`internal/ast/ast.go:232`) is an `*ast.Event` declared inside the translation, and it carries a
   `description` in `test.DescribedHotelReservation:191`. Measured: hovering `BookingImported` at that
   declaration returns nil, and go-to-definition from `subscribes [BookingImported]` would return nil
   too, because `declarationsIn` (`internal/lsp/model.go:38-60`) walks `slice.Events` only. Criterion 1
   says "any construct", so Tasks 1 and 2 answer hover there. `declarationsIn` is **not** extended,
   because that would change go-to-definition and find-references results for a construct no criterion
   of this story mentions — a separate, defensible change that lands on its own
   (`tasks/learnings.md:461-464`). Task 1 pins the asymmetry so it reads as decided rather than
   discovered.
10. **Completion returns the whole list and lets the client filter**, including a payload field the
    author has already written. This is what every existing list does and what LSP clients expect; a
    server-side filter would also have to decide what `IsIncomplete: false` means after a backspace.
    Carried over from the same decision in
    `tasks/completed/us-009-complete-and-navigate-automations-in-the-editor.md`.
11. **A value slot needs whitespace between the keyword and the cursor.** `given|` is a half-typed
    keyword and completes from the block's keyword list; `given |` and `given [|` are value positions.
    `valueSlotBefore` already implements this (`internal/lsp/completer.go:188-193`); Tasks 6-8 inherit
    it rather than restating it, and Task 7 pins it once.

**Overarching constraint:** every existing `.emod` file stays valid with unchanged meaning. In this
story that reads as: **no file outside `internal/lsp` changes at all.** Every task's change set is
`internal/lsp/*.go` plus its `_test.go` siblings, and the receipt is that `go build ./...` and the full
`go test -tags unit ./...` pass with every expected value outside `internal/lsp` unedited.

**Learnings folded in** from `tasks/learnings.md`:

- *`internal/lsp` walks the model once now — extend `model.go`, never open a sixth walk* (`:441-444`).
  Every task that needs a new traversal or a new reference site puts it in `internal/lsp/model.go`.
  This is the single most load-bearing entry for this story: seven of the nine tasks touch that file.
- *The completer reads raw lines, not the AST, so a new block owes two arms and a value-slot decision*
  (`:446-449`). It names `spec` explicitly as a body with no arm that still answers the top-level list —
  verified. Task 7 owes both arms; Tasks 6 and 8 owe value-slot decisions.
- *A slice has two homes, and much of the repo still walks only one* (`:171-174`). Every task's
  assertions run over both homes, which is why `test.SpecLibraryLending` and
  `test.AutomationReadsLibraryLending` are the fixtures named throughout.
- *An `assert…` helper that returns the value it pinned whole makes every follow-up check dead*
  (`:451-454`) — and its closing sentence: *a criterion that names both an exact expected string and
  substrings it must contain is asking for a dead assertion*. No criterion below names both. Where a
  criterion pins wording it says "asserted with one `require.Equal` on the whole value"; where it pins
  a property it says so and leaves the wording free.
- *Prefer a single structural assertion* (`:396-404`) and *`requireLocation` is a find-in-list check*
  (`internal/lsp/references_test.go:51-68`, which cannot see a missing or extra entry). Every new
  find-references leaf asserts the whole location list with one `require.Equal`, using `locationOf`
  (`:66-78`); where a new whole-list assertion subsumes an existing `requireLocation` loop, the narrower
  leaf is deleted rather than kept.
- *An assertion whose expected value comes from the code under test cannot fail* (`:126-129`).
- *A spec is not a reference* (`:191-194`) — reconciled in "Open questions, decided" item 5.
- *A spec's `when` resolves against commands *and* events* (`:201-204`) — item 6.
- *`emod fmt` moves a spec to the end of its slice and orders its entries given/when/then* (`:196-199`)
  — the order Task 7's keyword list reads in, matching the automation list's precedent.
- *Ask the lexer which keywords exist; never restate the set* (`:81-84`) and *Assert a short keyword
  with a `\b`-bounded `require.Regexp`* (`:236-239`) — `on`, `and`, `or` and `not` hide inside longer
  words in this package's own descriptions.
- *A tested, defensible improvement found on the way is still a separate commit* (`:461-464`) — the
  filter for every task below is "this task's own acceptance criteria cannot pass without it". The
  `ConvertDiagnostics` severity defect and `declarationsIn`'s blindness to a translation's event are
  both on the wrong side of that filter and are recorded above rather than absorbed.
- *A `_test.go` file always carries the `Test…` umbrella for the name it wears* (`:456-459`) — no task
  below adds a `_test.go` file, so the existing umbrellas absorb every new leaf.
- *A task's change-set assertion must name every file its own patterns require it to change*
  (`:326-329`).
- *A task criterion requiring "committed" output cannot close* (`:21-24`) and *A commit-message receipt
  is the commit author's obligation, never an acceptance criterion* (`:246-249`).

**Repo drift noted while reading.** Several `tasks/learnings.md` entries cite
`internal/export/export.go` (for example `:186-189` on `convertSpecOutcome`); that file no longer
exists — the package is `json.go`, `cue.go` and `diagram.go`. No task here touches `internal/export`,
so the drift affects nothing below, but it will mislead a reader who follows the citation. Likewise
`tasks/completed/us-009-complete-and-navigate-automations-in-the-editor.md` describes `scopedSlices`,
`resolveContext`, `pendingKeyword` and `completionsFor`; those are now `ast.Model.SliceRefs`,
`enclosingBlock`, `keywordAwaitingBrace` and `keywordsFor`. The shapes it describes are still the
shapes; the names moved.

---

## Codebase Context

Everything below was read on this worktree, and every "measured"/"verified" claim was produced by
running the code, not by reading it.

**The shared model walk** (`internal/lsp/model.go`, 135 lines) is the package's only AST traversal, as
`tasks/learnings.md:441-444` requires.

- `parseModel(text, uri)` (`:9`) scans and parses, keeping the model and discarding diagnostics.
- Slice traversal is `ast.Model.SliceRefs()` (`internal/ast/traverse.go:66`), which yields every slice
  paired with its context and, where it has one, its aggregate — `nil` for the slices a `mode dcb`
  context declares directly — sorted into source order by `Slice.NamePos`. `declaredAggregates`
  (`internal/lsp/model.go:15`) is the flat aggregate list.
- `nameKind` (`:23-30`) has four members: `commandName`, `eventName`, `viewName`, `contextName`.
  `declarationsIn` (`:38-60`) yields every context name plus every command, event and view declared on
  a slice — **not** the event a translation declares inside itself. `declaredNamesInOrder` (`:62`)
  feeds completion; `newDeclaredNames`/`positionOf` (`:72-92`) feed go-to-definition and
  find-references.
- `referencesIn` (`:101-134`) is the ordered list of every site that *names* something rather than
  declaring it: a view's `subscribes` entries, an automation's `on`/`reads`/`command`/`target context`,
  a translation's `reads`/`command`, a trigger's `reads`, and a flow's command and event. **No spec
  site and no invariant site appears in it**, which is exactly why go-to-definition from every spec
  reference returns nil today.

**Hover** (`internal/lsp/hover.go`, 157 lines). `keywordDescriptions` (`:11-49`) maps all 38 lexer
keywords to text — including `invariant`, `spec`, `given`, `when`, `then` and `rejected`. `GetHover`
(`:59`) walks `model.SliceRefs()` for a command, event or view **declaration** name (`:71-88`), then
falls through to a token scan gated on `tok.Type.IsKeyword()` (`:89-98`). `declaredIn` (`:103`) renders
the `<Context> > <Aggregate>` scope string and drops the second segment where there is no aggregate.
`hoverAt` (`:124`), `bulletList` (`:146`) and `fieldDescriptions` (`:134`) are the content builders;
`nameRange` (`internal/lsp/position.go:38`) builds the range.

Measured on `test.SpecLibraryLending`, every one of these returns **nil** today: a context name, an
aggregate name, a slice name, a trigger name, a model name, an actor name, an invariant name at its
declaration, an invariant name in `then rejected`, an event name inside `given [...]`, and a command
name after `when`. Hovering the `rejected` and `spec` *keywords* returns their descriptions, so
criterion 2 is entirely about the identifier beside the keyword, never the keyword.

**Completion** (`internal/lsp/completer.go`, 250 lines). `GetCompletions` never parses for the block
decision: `enclosingBlock` (`:44`) runs `blockScanner` (`:56-101`) over the lines up to the cursor,
counting braces over `codeOutsideStringsAndComments` (`:104`) so a `#`, `//` or brace inside a quoted
description opens and closes nothing. A block needs **both** a `findBlockKeyword` arm (`:126`,
spelling → `blockContext`) and a `keywordsFor` arm (`:223`, `blockContext` → labels); one without the
other falls through to the parent's list. `keywordAwaitingBrace` holds a braceless keyword only until
the next line carrying code. Value completion is `valueSlots` (`:176-179`), an entry-keyword →
`{nameKind, CompletionItemKind}` map holding `on` and `reads` only, skipped inside `ctxFields`
(`:183-186`) because keywords stay legal as field names, types and modifiers; `valueCompletions`
(`:203`) parses the model and returns `declaredNamesInOrder(model, slot.nameKind)`, a **flat,
model-wide** list with no notion of scope. Item kinds deliberately match what `GetSemanticTokens`
paints the same names with: command → `TokenTypeFunction`, event → `TokenTypeEvent`, view →
`TokenTypeClass` (`internal/lsp/semantictokens.go:81-86`), so `FunctionCompletion`, `EventCompletion`
and `ClassCompletion` are the three kinds this story can need.

Measured: a cursor on a blank line inside `spec { }` returns `model actor context`; so does a cursor
after `given [`, after `then rejected `, on a `command -> rejected: X -> ` line, and inside a `flow { }`
body.

**Definition** (`internal/lsp/definition.go`, 31 lines). `GetDefinition` parses, builds
`newDeclaredNames`, and returns the first `referencesIn` entry the cursor sits on whose name the model
declares. It names no construct itself. Measured: nil from a `given` element, from `when`, from a
`then` element and from a `then rejected` name.

**References** (`internal/lsp/references.go`, 66 lines). `GetReferences` resolves the cursor through
`targetAt` (`:47`) — declarations of the three `referenceTargetKinds` (`:7`: command, event, view;
`contextName` is deliberately omitted) first, then the `referencesIn` entries — and emits the
declaration followed by every same-kind, same-name `referencesIn` entry, in list order. A site added to
`referencesIn` therefore reaches cursor resolution and the listing together. Measured on
`test.SpecLibraryLending`: find-references on `CopyBorrowed` returns **3** locations (declaration,
`subscribes`, flow entry) and lists none of the three specs that name it; find-references on
`OneCopyPerLoan` returns **0**.

**The invariant scope rule** (`internal/validator/validator.go:203-295`). `invariantScopes` (`:220`)
builds one scope per context (its own invariants, its own slices) and one per aggregate (its
invariants, its slices) — deliberately not nested. `unresolvedRejections` (`:246`) reports
`invariant %q is not declared in %s %q` naming the scope kind and name. `scopedInvariantDiagnostics`
(`:287`) position-sorts, because a context holds its own invariants and its aggregates' in separate
collections. This is the rule Tasks 3, 5, 6 and 9 mirror.

**Diagnostics** (`internal/lsp/diagnostics.go`, 42 lines). `ConvertDiagnostics` maps `diagnostic.Warning`
to `SeverityWarning` and everything else — including `diagnostic.Info` — to `SeverityError`. See the
pre-existing-defect note above. Untouched by every task here.

**Shared fixtures these tasks read** (all on `main`, none edited):

- `test.SpecLibraryLending` (`internal/test/fixtures.go:423`) — specs in **both** slice homes and with
  **both** `then` outcomes. Aggregate "Loan" of context "Lending" declares
  `invariant OneCopyPerLoan`; the `mode dcb` context "Reading Room" declares `invariant OneReaderPerDesk`
  directly. `spec "refuses a copy already on loan"` states `then rejected OneCopyPerLoan`;
  `spec "refuses a desk another reader is seated at"` states `then rejected OneReaderPerDesk`. Specs
  cover `given []`, an omitted `given`, a two-element `given`, and a spec written mid-block.
  `test.SpecLibraryLendingModel` (`internal/test/models.go:37`) is the parsed accessor and
  `test.DeclaredSpecNames` (`fixtures.go:1299`) the read-back.
- `test.DescribedHotelReservation` (`:101`) — a description on **every** construct that accepts one:
  model, actor, context, aggregate, slice, trigger, command, event, view, automation, translation, and
  the event the translation declares inside itself (`:190`). Its undescribed twin is
  `test.HotelReservation`, which is the differential pair Task 2 needs. It declares no `mode dcb`
  context, so the aggregate-less shapes come from the next fixture.
- `test.AutomationReadsLibraryLending` (`:581`) — slices in both homes, already the fixture
  `hover_test.go` uses for the `<Context>` vs `<Context> > <Aggregate>` pair.
- `test.InvariantLibraryLending` (`:314`) — invariants in both homes, five of them across three scopes,
  and no two scopes declaring the same name.

**LSP tests.** Every `_test.go` in `internal/lsp` is `//go:build unit`, package `lsp_test`, one
`Test<Function>` umbrella per file. `posIn` (`internal/lsp/cursor_test.go:14`) resolves a substring to
0-based coordinates *within the first occurrence of a container substring* and is the shared helper.
`hover_test.go` has `assertHover`/`assertNil` (`:60`, `:69`) — note `assertHover` both pins the whole
value and returns it, which is `tasks/learnings.md:451-454`'s recorded trap. `references_test.go` has
`requireLocation` (`:51`, find-in-list, blind to a missing entry) and `locationOf` (`:70`, the whole-
`Location` builder that whole-list assertions use). `internal/lsp/server_test.go` (1335 lines) is the
over-the-wire umbrella with a `t.Run` group per LSP method. `internal/lsp/keywords_test.go` holds the
two CI-enforced drift suites and the `keywordBlocks` table (`:102-115`) a new block joins.

**Not touched, deliberately.** `internal/lexer`, `internal/parser`, `internal/ast`, `internal/formatter`,
`internal/validator`, `internal/linter`, `internal/export`, `internal/cue`, `internal/importer`,
`internal/diagram`, `internal/glossary`, `internal/arrange`, `internal/cli`, `internal/viewer`,
`internal/wasm`, `internal/oracle`, `internal/test`, `editors/`, `docs/`, `examples/`, `e2e/`.

---

## Tasks

### Task 1: Answer hover on every declaration, naming its kind and where it was declared

**Behavior:** hovering the name of a model, an actor, a context, an aggregate, a slice, a trigger, an
automation or a translation returns markdown naming what kind of construct it is and, where the
construct sits inside another, the context and aggregate that hold it. The event a translation declares
inside itself hovers like an event. A name written in quotes resolves from a cursor anywhere on the
name, and the reported range covers the name without its quotes. Command, event and view hovers return
exactly what they return today.

**Acceptance Criteria:**
- [x] Hovering a model name, an actor name, a context name, an aggregate name, a slice name, a trigger
      name, an automation name and a translation name each returns non-nil markdown naming that
      construct's kind, asserted on `test.DescribedHotelReservation`, which declares all eight
- [x] The aggregate hover names the context that holds it; the slice, trigger, automation and
      translation hovers name the context and the aggregate that hold them, in the same
      `<Context> > <Aggregate>` shape `declaredIn` (`internal/lsp/hover.go:103`) already renders for a
      command
- [x] Hovering the name of a slice, and of a command, event and view inside it, where the slice hangs
      directly off a `mode dcb` context names the context alone — no empty segment, no trailing
      separator, no placeholder standing in for the missing aggregate — asserted with one
      `require.Equal` on the whole value, on `test.AutomationReadsLibraryLending`, in the same subtest
      as an aggregate-nested slice of the same model so both shapes are proved together
- [x] Hovering the name of the event a translation declares inside itself
      (`test.DescribedHotelReservation`, `event BookingImported` inside `translation BookingImport`)
      returns the same kind of hover an event declared on a slice returns, including its fields
- [x] Go-to-definition from `subscribes [BookingImported]` in that same model still returns nil after
      this task, and find-references on that event still returns nothing — pinned so that hover seeing
      a translation's event while navigation does not reads as decided ("Open questions, decided" item
      9) rather than as a half-finished change
- [x] For a construct whose name is written in quotes, a cursor on the name's **first** character and a
      cursor on its **last** character both resolve, and a cursor on either quote character does not;
      the returned range starts at the name's first character and ends one past its last, asserted with
      one `require.Equal` on the whole `lsp.Range` — the offset `entries.addQuoted`
      (`internal/lsp/semantictokens.go:57-63`) already documents
- [x] Hovering a construct name in a document the parser could not fully parse does not panic: the
      constructs parse recovery did recover still hover, and a token that resolves to none of them
      returns nil — the existing empty-document and non-resolvable-token leaves in `hover_test.go`
      still pass unedited
- [x] `keywordDescriptions` (`internal/lsp/hover.go:10-49`) gains no entry and loses none: hovering the
      `context`, `aggregate` and `slice` keywords still returns the text they return today, asserted
      beside a construct-name leaf on the same document so "the keyword answers as a keyword and the
      name beside it answers as a declaration" is proved in one place
- [x] The five existing command/event/view leaves in `internal/lsp/hover_test.go`
      (`command name shows parent context and aggregate` through
      `view without subscriptions omits subscribes section`) pass with their expected strings unedited.
      `internal/lsp/server_test.go`'s `hover` group passes with its expected values unedited, but its
      null-result leaf re-anchors its cursor from the name to the closing quote: `model "test"` hovers
      once criterion 1 holds, so the position that stood for "resolves to nothing" had to move to one
      that still does
- [x] `declaredNamesInOrder`, `newDeclaredNames` and `referencesIn` carry exactly the names they carry
      today, so `GetCompletions`, `GetDefinition` and `GetReferences` return byte-identical results —
      asserted by every existing subtest in `completer_test.go`, `definition_test.go` and
      `references_test.go` passing unedited
- [x] The declaration walk this task needs lives in `internal/lsp/model.go`, not in `hover.go`
      (`tasks/learnings.md:441-444`); `rg 'model\.Contexts|ctx\.Aggregates|SliceRefs' internal/lsp`
      returns hits from `model.go` and `semantictokens.go` only
- [x] `internal/test/fixtures.go` is read, never edited, and the change set is exactly
      `internal/lsp/model.go`, `internal/lsp/hover.go`, `internal/lsp/hover_test.go` and
      `internal/lsp/server_test.go`. `internal/lsp/position.go` needs no edit after all: normalising a
      quoted name's stored position onto the name's first character in the declaration walk is the
      same correction `entries.addQuoted` already applies, and it leaves `cursor.onName` and
      `nameRange` valid on the unquoted-name assumption they were written with

**Affected Files/Modules:**
- `internal/lsp/model.go` — the declaration walk (`declarationsIn`, `:38-60`) and what a declaration
  carries
- `internal/lsp/hover.go` — `GetHover`'s definition walk (`:71-88`), `declaredIn` (`:103`) and the
  `hoverFor*` builders (`:110-122`)
- `internal/lsp/position.go` — `cursor.onName` (`:20`) and `nameRange` (`:38`), which assume an
  unquoted name
- `internal/lsp/hover_test.go`

**Patterns to Follow:**
- `entries.addQuoted` and `entries.addIdentifier` (`internal/lsp/semantictokens.go:53-63`) — the
  existing statement of which constructs' positions point at a quote and which at an identifier
- `GetSemanticTokens`'s construct walk (`internal/lsp/semantictokens.go:102-124`) — the same set of
  declarations, already enumerated once in this package
- `ast.Model.SliceRefs` and `ast.Context.SliceRefs` (`internal/ast/traverse.go:30-77`) for both slice
  homes; `tasks/learnings.md:171-174`
- `tasks/learnings.md:101-104` "Name an extracted helper after the contract its callers rely on"
- `tasks/learnings.md:451-454` — do not add a `require.Contains` on a value a helper already pinned
  whole

**Testable:** Yes — through `lsp.GetHover`, which is exported.

**Verification:** `go test -tags unit ./internal/lsp/...`; `go build ./...`.

**Depends on:** None

---

### Task 2: Show a construct's description on hover

**Behavior:** every construct hover answers for carries its `description` in the returned markdown. A
construct with no description returns exactly the markdown it returned before this task.

**Acceptance Criteria:**
- [x] Hovering each of the twelve constructs `test.DescribedHotelReservation` describes — model, actor,
      context, aggregate, slice, trigger, command, event, view, automation, translation, and the event
      the translation declares — returns markdown carrying that construct's description text
- [x] Hovering each of the same constructs in `test.HotelReservation`, the undescribed twin of the same
      model, returns markdown with no description section, no stray blank line and no placeholder —
      asserted with one `require.Equal` on the whole value per construct, and run as the same table as
      the leaf above so the only difference between the two models' hovers is the description
      (`tasks/learnings.md:96-99`: the differential's twin must be proved to differ)
- [x] An event carrying a description **and** fields returns one value holding both, and a view carrying
      a description **and** subscriptions returns one value holding both; the order the sections appear
      in is the same for both, asserted by the whole-value equality above rather than by a separate
      ordering check
- [x] A description containing a `#`, a `//` or a brace is carried through unchanged
- [x] Hovering the `description` keyword still returns its `keywordDescriptions` text, unedited
- [x] `internal/lsp/server_test.go`'s `hover` group passes with its expected values unedited — its
      documents declare no descriptions
- [x] `test.HotelReservation` and `test.DescribedHotelReservation` are read as input;
      `internal/test/fixtures.go` is not edited
- [x] The change set is exactly `internal/lsp/hover.go` and `internal/lsp/hover_test.go`

**Affected Files/Modules:**
- `internal/lsp/hover.go` — the content builders (`hoverForCommand`/`hoverForEvent`/`hoverForView`,
  `:110-122`, and whatever Task 1 adds beside them), `bulletList` (`:146`)
- `internal/lsp/hover_test.go`

**Patterns to Follow:**
- `bulletList` (`internal/lsp/hover.go:146`) — the existing "omit the section entirely when the content
  is empty" shape, which is the behaviour an absent description needs
- `test.HotelReservation` / `test.DescribedHotelReservation` (`internal/test/fixtures.go:13`, `:101`) —
  the twin pair, and `tasks/learnings.md:96-99` on what a differential owes
- `tasks/learnings.md:451-454` — one whole-value `require.Equal` per leaf, no follow-up `Contains`

**Testable:** Yes — through `lsp.GetHover`.

**Verification:** `go test -tags unit ./internal/lsp/...`; `go build ./...`.

**Depends on:** Task 1

---

### Task 3: Show an invariant's prose where it is declared and where a spec rejects by it

**Behavior:** hovering an invariant's name at its declaration returns markdown naming the invariant
kind, the scope that declares it, and its statement. Hovering the invariant name in a spec's
`then rejected <name>` returns the same markdown. The name in `then rejected` resolves in the scope
`emod validate` resolves it in — the aggregate holding the spec's slice, or the context when the slice
hangs directly off one — so a name declared only in a different scope answers nothing.

**Acceptance Criteria:**
- [x] Hovering `OneCopyPerLoan` where `test.SpecLibraryLending` declares it on aggregate "Loan" returns
      non-nil markdown naming the invariant kind, the scope that declares it, and the statement text
- [x] Hovering `OneCopyPerLoan` in that model's `then rejected OneCopyPerLoan` returns a value equal to
      the declaration's, asserted with one `require.Equal` between the two hovers' contents so the two
      sites are proved to agree without transcribing the wording twice
- [x] The same declaration/reference pair holds for `OneReaderPerDesk`, which the `mode dcb` context
      "Reading Room" declares directly, so both scopes the language gives an invariant are proved in
      one subtest
- [x] On a document where two aggregates of one context each declare an invariant with the **same
      name** and different statements, and each has a slice whose spec rejects by it, each
      `then rejected` hovers with **its own** aggregate's statement — the case a flat model-wide lookup
      answers wrongly and silently ("Open questions, decided" item 4). The document is authored in
      `internal/lsp/hover_test.go`; no shared fixture is edited
- [x] A `then rejected` naming an invariant declared only in another scope returns nil — the same input
      `emod validate` reports as `invariant %q is not declared in %s %q`
      (`internal/validator/validator.go:281`)
- [x] Hovering the `rejected` keyword on that same line still returns its `keywordDescriptions` text,
      asserted in the same subtest as the invariant-name leaf so the two tokens are proved to answer
      differently
- [x] Hovering an identifier that is not an invariant name and not a declaration still returns nil, and
      every existing `hover_test.go` leaf passes unedited
- [x] The scope resolution lives in `internal/lsp/model.go` and yields, for each invariant reference,
      the reference's position and the invariants of the one scope it resolves in; it does **not** add
      a `nameKind` member and does not appear in `declaredNames`, `declaredNamesInOrder` or
      `referenceTargetKinds`, so `GetCompletions`, `GetDefinition` and `GetReferences` results are
      unchanged by this task — asserted by every existing `completer_test.go`, `definition_test.go` and
      `references_test.go` subtest passing unedited
- [x] The change set is exactly `internal/lsp/model.go`, `internal/lsp/hover.go` and
      `internal/lsp/hover_test.go`

**Affected Files/Modules:**
- `internal/lsp/model.go` — the invariant scope walk and the invariant reference list
- `internal/lsp/hover.go` — `GetHover`'s resolution order, ahead of the keyword token scan (`:89-98`)
- `internal/lsp/hover_test.go`

**Patterns to Follow:**
- `invariantScopes`, `invariantScope.unresolvedRejections` and `scopedInvariantDiagnostics`
  (`internal/validator/validator.go:220-295`) — the scope rule this must agree with, including that a
  context's own invariants and its aggregates' are separate and neither hides the other
- `ast.Invariant` (`internal/ast/ast.go:68-74`) and `ast.ThenRejected` (`:121-126`) — the two nodes
- `tasks/learnings.md:186-189` — `Spec.Then` is a sealed interface whose type switches fail silently on
  a kind they have not heard of; this task reads one variant and must leave the other alone
- `tasks/learnings.md:441-444` — the walk belongs in `model.go`
- `test.SpecLibraryLending` (`internal/test/fixtures.go:423`) and `hover_test.go`'s existing
  `automationDoc` (`:37`) as the precedent for a locally authored document

**Testable:** Yes — through `lsp.GetHover`.

**Verification:** `go test -tags unit ./internal/lsp/...`; `go build ./...`.

**Depends on:** Task 1

---

### Task 4: Jump to and from the events and commands a spec names

**Behavior:** go-to-definition from an element of a spec's `given` list, from its `when`, or from an
element of its `then` event list lands on that construct's declaration. Find-references on a command or
an event lists the spec sites that name it alongside the sites it lists today, whichever of the two the
cursor sits on.

**Acceptance Criteria:**
- [x] Go-to-definition from an element of a `given` list returns the event's declaration position, on
      `test.SpecLibraryLending`, in both slice homes
- [x] Go-to-definition from a `when` naming a command returns the command's declaration, and from a
      `when` naming an event returns the event's declaration — asserted together, because a spec's
      `when` resolves against commands **and** events while `given`/`then` resolve against events only
      (`tasks/learnings.md:201-204`)
- [x] Go-to-definition from an element of a `then` event list returns the event's declaration
- [x] Go-to-definition from a `then rejected` invariant name still returns nil after this task — Task 5
      owns it, and pinning it here keeps the two increments legible
- [x] Find-references with the cursor on an event declaration in `test.SpecLibraryLending` returns its
      declaration, every site listed before this task, and every spec `given`/`then` element naming it,
      in source order — asserted with one `require.Equal` against the whole expected list built with
      `locationOf` (`internal/lsp/references_test.go:70`), not a `requireLocation` loop
- [x] Find-references with the cursor on one of those spec elements returns that same whole list
- [x] Find-references on a command returns its declaration, its flow entry and every spec `when`
      naming it, asserted the same way
- [x] A spec naming a construct the model does not declare yields no jump and contributes no location —
      asserted on a document authored in the test file, since every shared fixture validates cleanly
- [x] `internal/validator` is not edited and its notion of a reference is unchanged: a command whose
      only mention is a spec is still reported `[orphan-command]`, so `internal/linter`'s tests pass
      unedited ("Open questions, decided" item 5; `tasks/learnings.md:191-194`)
- [x] `internal/lsp/server_test.go`'s `definition` and `references` groups pass with their expected
      values unedited — their `testDoc` declares no spec
- [x] The change set is exactly `internal/lsp/model.go`, `internal/lsp/definition_test.go` and
      `internal/lsp/references_test.go`; `definition.go` and `references.go` need no edit, because
      `referencesIn` is the one list both read (`tasks/learnings.md:441-444`)

**Affected Files/Modules:**
- `internal/lsp/model.go` — `referencesIn` (`:101-134`), which states every naming site
- `internal/lsp/definition_test.go`, `internal/lsp/references_test.go`

**Patterns to Follow:**
- `referencesIn`'s existing per-construct blocks (`internal/lsp/model.go:106-131`) — one `add(kind,
  name, pos)` per naming site, which is the whole production diff a new reference site costs
  (`tasks/learnings.md:441-444`)
- `ast.Spec` and `ast.SpecElement` (`internal/ast/ast.go:95-119`), and `ast.ThenEvents` (`:115-119`)
- `internal/validator/validator.go:341-365` (`specDiagnostics`, `undeclaredSpecEvents`) — the existing
  statement of which spec part resolves against which kind
- `locationOf` (`internal/lsp/references_test.go:70`) and `tasks/learnings.md:396-404` — whole-list
  assertions, and delete the narrower `requireLocation` leaf any new whole-list assertion subsumes
- `tasks/learnings.md:171-174` — both slice homes

**Testable:** Yes — through `lsp.GetDefinition` and `lsp.GetReferences`.

**Verification:** `go test -tags unit ./internal/lsp/... ./internal/linter/...`; `go build ./...`.

**Depends on:** None

---

### Task 5: Jump to and from the invariant a spec's `then rejected` names

**Behavior:** go-to-definition from a spec's `then rejected <name>` lands on that invariant's
declaration in the scope the name resolves in. Find-references on an invariant — with the cursor on its
declaration or on any `then rejected` naming it — lists the declaration and every spec in that scope
that rejects by it.

**Acceptance Criteria:**
- [x] Go-to-definition from `then rejected OneCopyPerLoan` in `test.SpecLibraryLending` returns the
      declaration position of the invariant on aggregate "Loan"; the same holds for
      `then rejected OneReaderPerDesk` and the invariant the `mode dcb` context declares directly, so
      both scopes are proved together
- [x] On the two-aggregates-one-name document, each `then rejected` jumps to **its own** aggregate's
      declaration and never to the other's
- [x] Go-to-definition from a `then rejected` whose name resolves in no enclosing scope returns nil
- [x] Find-references with the cursor on an invariant declaration returns the declaration followed by
      every `then rejected` in that scope naming it, in source order, asserted with one `require.Equal`
      against the whole expected list
- [x] Find-references with the cursor on a `then rejected` name returns that same whole list
- [x] On the two-aggregates-one-name document, find-references from one aggregate's declaration lists
      only that aggregate's spec sites — the other's identically named invariant and its spec appear
      nowhere in the list
- [x] Find-references on a command, an event and a view returns exactly what Task 4 made it return:
      invariants do not join `referenceTargetKinds` (`internal/lsp/references.go:7`), so no command,
      event or view listing changes — asserted by Task 4's leaves passing unedited
- [x] `internal/lsp/server_test.go`'s `definition` and `references` groups pass unedited
- [x] The change set is exactly `internal/lsp/definition.go`, `internal/lsp/references.go`,
      `internal/lsp/definition_test.go` and `internal/lsp/references_test.go` — the scope resolution
      Task 3 put in `model.go` is consumed, not rebuilt

**Affected Files/Modules:**
- `internal/lsp/definition.go` — `GetDefinition`'s reference loop (`:21-28`)
- `internal/lsp/references.go` — `GetReferences` (`:14`) and `targetAt` (`:47`)
- `internal/lsp/definition_test.go`, `internal/lsp/references_test.go`

**Patterns to Follow:**
- Task 3's invariant scope walk in `internal/lsp/model.go`
- `GetReferences`'s existing shape (`internal/lsp/references.go:29-41`): resolve the cursor, emit the
  declaration, then every matching site in list order
- `locationOf` (`internal/lsp/references_test.go:70`) and `tasks/learnings.md:396-404`
- `internal/validator/validator.go:246-264` — `unresolvedRejections`, the same walk from the
  validator's side

**Testable:** Yes — through `lsp.GetDefinition` and `lsp.GetReferences`.

**Verification:** `go test -tags unit ./internal/lsp/...`; `go build ./...`.

**Depends on:** Task 3

---

### Task 6: Offer the invariants in scope after `then rejected`

**Behavior:** a cursor after `then rejected ` inside a spec offers the invariant names declared by the
aggregate holding that spec's slice, or by the context where the slice hangs directly off one — the
same scope `emod validate` resolves the name in, so every name offered validates and no name from
another scope appears.

**Acceptance Criteria:**
- [x] With the cursor after `then rejected ` in a spec inside an aggregate's slice, the labels offered
      are exactly the invariant names that aggregate declares, in declaration order, asserted with one
      `require.Equal` on the label list
- [x] With the cursor after `then rejected ` in a spec inside a slice a `mode dcb` context declares
      directly, the labels are exactly that context's own invariant names
- [x] An invariant declared by a different aggregate of the same context, and one declared by a
      different context, appear in neither list — asserted on one document declaring invariants in
      three scopes, so a model-wide list is visibly wrong rather than coincidentally right
- [x] The offered items carry a completion kind distinct from `KeywordCompletion`, and the same kind is
      used wherever this story offers an invariant name
- [x] A cursor after `then ` with no `rejected` on the line does not offer invariant names
- [x] A cursor still touching a half-typed `rejected` offers no invariant names — the whitespace rule
      `valueSlotBefore` (`internal/lsp/completer.go:188-193`) already applies. Once Task 7 makes `then`
      a value slot the half-typed word is indistinguishable from a half-typed event name, so that slot
      answers and the client filters `rejected` out of it; the invariants stay out either way
- [x] A `fields` block declaring a field named `rejected` still offers field types and modifiers after
      it, not invariant names — the `ctxFields` suppression (`internal/lsp/completer.go:183-186`) still
      covers the new slot
- [x] A cursor on a `command -> rejected: <Command> -> <invariant>` line offers the enclosing block's
      keyword list and no invariant names — pinned because the identifier immediately after `rejected`
      on that line is a **command** name ("Open questions, decided" item 7). The line is written into a
      test document as text; nothing here reads US-009's AST, so this task does not depend on it
- [x] `TestKeywordCoverage` (`internal/lsp/keywords_test.go`) passes with `keywordBlocks` unedited: this
      task offers no keyword
- [x] Every existing `completer_test.go` subtest passes unedited
- [x] The change set is exactly `internal/lsp/completer.go`, `internal/lsp/model.go` and
      `internal/lsp/completer_test.go`

**Affected Files/Modules:**
- `internal/lsp/completer.go` — `completionsAt` (`:20`), `valueSlots` (`:176-179`), `valueSlotBefore`
  (`:181`) and `valueCompletions` (`:203`), which today resolves a flat model-wide list with no cursor
  position and no scope
- `internal/lsp/model.go` — the scope lookup that answers which invariants are in scope at a line
- `internal/lsp/completer_test.go`

**Patterns to Follow:**
- `valueSlots` and `valueCompletions` (`internal/lsp/completer.go:176-209`) — the existing entry-keyword
  → names path, and `tasks/learnings.md:446-449` on what a new value slot owes
- `invariantScopes` (`internal/validator/validator.go:220-230`) — the scope rule; Task 3's walk in
  `internal/lsp/model.go` is the LSP-side statement of it
- `ast.Slice.OpenPos`/`ClosePos`, `ast.Aggregate.OpenPos`/`ClosePos` and `ast.Context.OpenPos`/`ClosePos`
  (`internal/ast/ast.go`) — the positions that relate a cursor line to the declaration enclosing it
- `extractLabels` and `requireItemKinds` (`internal/lsp/completer_test.go`) — the existing assertion
  helpers
- `tasks/learnings.md:171-174` — both slice homes

**Testable:** Yes — through `lsp.GetCompletions`.

**Verification:** `go test -tags unit ./internal/lsp/...`; `go build ./...`.

**Depends on:** None

---

### Task 7: Offer a spec's entries and the names its `given`, `when` and `then` accept

**Behavior:** a cursor inside a `spec` body offers the three entries a spec accepts, in the order
`emod fmt` writes them. A cursor inside a `given [...]` or `then [...]` list offers the event names the
model declares; a cursor after `when ` offers the commands and the events it declares. A cursor after
`then rejected ` keeps offering the invariant names Task 6 gave it.

**Acceptance Criteria:**
- [x] A cursor on a blank line inside a `spec { }` block offers exactly `given`, `when`, `then` — the
      order `writeSpec` emits them in (`internal/formatter/formatter.go`,
      `tasks/learnings.md:196-199`) — every item a `KeywordCompletion`
- [x] The spec body joins `keywordBlocks` in `internal/lsp/keywords_test.go:102-115`, so
      `TestKeywordCoverage/completion` asserts every offered label is a spelling `lexer.Keywords()`
      reports
- [x] `description` is not offered inside a spec body: a spec accepts no description
      (US-002's criterion names nine constructs and `spec` is not among them)
- [x] A cursor after `given [` offers the event names the model declares, in declaration order, with
      the completion kind `GetSemanticTokens` paints an event with; a cursor after an element and its
      comma in the same list offers the same names
- [x] A cursor after `then [` offers the same event names
- [x] A cursor after `when ` offers the commands the model declares **and** the events it declares —
      both, because a spec's `when` resolves against both (`tasks/learnings.md:201-204`) — each with the
      completion kind `GetSemanticTokens` paints that kind of name with, asserted with one
      `require.Equal` on the whole item list so the kinds and the order are pinned together
- [x] A cursor after `then rejected ` still offers the invariant names Task 6 gave it and offers no
      event names, asserted in this task's file too — `then` becoming a value slot here must not
      outrank `rejected`
- [x] A cursor still touching a half-typed `given`, `when` or `then` offers the spec body's keyword
      list, not names
- [x] A `fields` block declaring a field named `given`, `when` or `then` still offers field types and
      modifiers after it, not event names
- [x] After a closed `spec { }` block the cursor offers the enclosing slice's keyword list, and inside
      a `spec` block written with its brace on the following line the spec entries are still offered —
      the two `blockScanner` behaviours (`internal/lsp/completer.go:56-101`) an existing context-block
      subtest already pins for another block
- [x] Every existing `completer_test.go` subtest passes unedited, and `internal/lsp/server_test.go`'s
      `completion` group passes unedited
- [x] The change set is exactly `internal/lsp/completer.go`, `internal/lsp/completer_test.go` and
      `internal/lsp/keywords_test.go`

**Affected Files/Modules:**
- `internal/lsp/completer.go` — `blockContext` (`:29-40`), `findBlockKeyword` (`:124`), `keywordsFor`
  (`:223`) and `valueSlots` (`:176-179`)
- `internal/lsp/completer_test.go`, `internal/lsp/keywords_test.go`

**Patterns to Follow:**
- `tasks/learnings.md:446-449` — a new block owes **both** a `findBlockKeyword` arm and a `keywordsFor`
  arm; one without the other falls through to the parent's list, which is how an automation body once
  offered `model actor context`
- The automation block's arms (`internal/lsp/completer.go:143`, `:237`) and its
  `completer_test.go` group (`inside automation block returns the entries an automation accepts`) —
  the closest precedent, list order taken from the formatter
- `valueSlots`' comment (`internal/lsp/completer.go:174-175`) on matching `GetSemanticTokens`' kinds,
  and `internal/lsp/semantictokens.go:80-86` for which kind each name gets
- `keywordBlocks` (`internal/lsp/keywords_test.go:102-115`) — a block joins the drift table by parking a
  cursor on a blank line inside it
- `tasks/learnings.md:196-199` for the entry order

**Testable:** Yes — through `lsp.GetCompletions`.

**Verification:** `go test -tags unit ./internal/lsp/...`; `go build ./...`.

**Depends on:** Task 6

---

### Task 8: Offer a payload's field names, scoped to the construct the reference names

**Behavior:** a cursor inside a payload's braces offers the field names declared by the construct the
enclosing spec element names — the event's fields for a `given` or `then` element, the command's for a
`when` — and nothing else.

**Hard dependency:** US-010 Tasks 1 and 2 must be delivered first. `ast.SpecElement` carries `Name` and
`NamePos` and nothing else on this tree, so there is no payload to complete inside and no fixture
stating one. US-010's "Open questions, decided" item 1 is what this task rests on: payloads hang off
`ast.SpecElement`, so `given`, `when` and `then` elements all carry them; and its item 5 — a payload's
opening brace must sit on the line of the reference it qualifies — is what makes the construct's name
recoverable from the lines up to the cursor by a scanner that never parses.

**Acceptance Criteria:**
- [x] With the cursor between the braces of a payload on a `given` element, the labels offered are
      exactly the field names the referenced **event** declares, in declaration order, asserted with one
      `require.Equal` on the label list
- [x] With the cursor inside a payload on a `when` reference, the labels are exactly the referenced
      **command**'s field names; inside a payload on a `then` event-list element, the referenced event's
- [x] Two constructs declaring different field sets in one model each offer only their own field names,
      asserted in one subtest so a list that ignores the enclosing element is visibly wrong
- [x] A payload spanning several lines offers the same field names with the cursor on a continuation
      line, not only on the line carrying the opening brace
- [x] A payload whose element names a construct the model does not declare offers nothing at all — not
      the enclosing block's keyword list, and not another construct's fields
- [x] A field name already written in the payload is still offered: the server returns the whole list
      and the client filters ("Open questions, decided" item 10)
- [x] A payload on a construct declaring a field named after a DSL keyword offers that field name — a
      payload's labels are field names, not keywords, which is why the payload block is deliberately
      **not** added to `keywordBlocks` in `internal/lsp/keywords_test.go`; `TestKeywordCoverage` passes
      with that table unedited
- [x] Outside a payload nothing changes: a `fields` block still offers types and modifiers, a `tags`
      block still offers nothing, a `spec` body still offers Task 7's three entries, and
      `then rejected ` still offers Task 6's invariant names
- [x] A `{` on the line **after** a spec element opens no payload for completion, matching the parser
      (US-010 "Open questions, decided" item 5)
- [x] The fixture US-010 Task 2 adds to `internal/test/fixtures.go` is read as input and is not edited
- [x] Every existing `completer_test.go` subtest passes unedited
- [x] The change set is exactly `internal/lsp/completer.go`, `internal/lsp/model.go` and
      `internal/lsp/completer_test.go`

**Affected Files/Modules:**
- `internal/lsp/completer.go` — `blockScanner` (`:56-101`), which must carry enough about a brace it
  opened to answer which construct the payload qualifies, and `valueCompletions` (`:203`), whose one
  path today is `nameKind` → model-wide names
- `internal/lsp/model.go` — the construct-name → declared-fields lookup, which the package has no
  equivalent of (`declarationsIn` carries names and positions only)
- `internal/lsp/completer_test.go`

**Patterns to Follow:**
- `tasks/learnings.md:446-449` — the completer reads raw lines, not the AST; a new block owes two arms
  and a value-slot decision
- `blockScanner.consume` and `keywordAwaitingBrace` (`internal/lsp/completer.go:61-79`) — the existing
  "this line's keyword claims the next brace" mechanic, and the reason the claim expires at the next
  line carrying code
- `fieldDescriptions` (`internal/lsp/hover.go:134`) — the existing read of a construct's
  `[]*ast.Field`
- `ast.Command.Fields` and `ast.Event.Fields` (`internal/ast/ast.go:128-163`)
- `FieldCompletion` (`internal/lsp/protocol.go:55`)
- `tasks/us-010-state-example-payloads-in-specs.md` Tasks 1 and 2 — the AST shape and the fixture

**Testable:** Yes — through `lsp.GetCompletions`.

**Verification:** `go test -tags unit ./internal/lsp/...`; `go build ./...`.

**Depends on:** Task 6, Task 7, and US-010 Tasks 1-2 (delivered)

---

### Task 9: See a flow rejection edge from hover, go-to-definition and find-references

**Behavior:** the invariant named on a `command -> rejected:` entry hovers with the invariant's prose,
jumps to its declaration, and appears in find-references on that invariant. The command named on the
same entry jumps and lists like a flow entry's command does.

**Hard dependency:** US-009 Tasks 1, 2 and 3 must be delivered first. `ast` carries no rejection node on this
tree, and US-009 "Open questions, decided" item 1 puts rejections in their own `Slice` collection
precisely so that `referencesIn` (`internal/lsp/model.go:128-131`) keeps naming only a flow's command
and event until a story opts in. This task is that opt-in, and US-009's own out-of-scope list names it
(`tasks/us-009-show-rejection-paths-on-the-timeline.md:44-47`).

**Acceptance Criteria:**
- [ ] Hovering the invariant name on a `command -> rejected: <Command> -> <invariant>` entry returns a
      value equal to the hover Task 3 returns for that invariant's declaration, asserted with one
      `require.Equal` between the two so the wording is not transcribed twice
- [ ] That resolution uses the same scope rule: an edge in an aggregate's slice resolves against that
      aggregate's invariants, one in a `mode dcb` context's own slice against that context's, and an
      edge naming an invariant declared only elsewhere hovers nothing
- [ ] Go-to-definition from the invariant name on a rejection edge returns the declaration position in
      the resolving scope
- [ ] Go-to-definition from the **command** name on a rejection edge returns the command's declaration
- [ ] Find-references with the cursor on an invariant declaration returns the declaration, every
      `then rejected` in scope naming it, and every rejection edge in scope naming it, in source order,
      asserted with one `require.Equal` on the whole list — this replaces Task 5's list assertion for
      the model that carries both, rather than sitting beside it (`tasks/learnings.md:401-404`)
- [ ] Find-references with the cursor on a rejection edge's invariant name returns that same whole list
- [ ] Find-references on a command returns its declaration, its flow entries, its spec `when` sites and
      every rejection edge naming it
- [ ] Find-references on an **event** returns exactly what it returned before this task, on a model
      carrying rejection edges: a rejection edge names no event, and an edge folded into the flow list
      would make an invariant name resolve as one — the trap US-009 measured
      (`tasks/us-009-show-rejection-paths-on-the-timeline.md:96-110`)
- [ ] Both slice homes are covered, using the fixture US-009 Task 2 adds to `internal/test/fixtures.go`,
      which states rejection edges in an aggregate's slice and in a `mode dcb` context's own slice; that
      fixture is read, not edited
- [ ] Completion is unchanged by this task: a cursor on a rejection-edge line still offers the enclosing
      block's keyword list, as Task 6 pinned ("Open questions, decided" item 7)
- [ ] `internal/lsp/server_test.go`'s `hover`, `definition` and `references` groups pass with their
      expected values unedited
- [ ] The change set is exactly `internal/lsp/model.go`, `internal/lsp/hover_test.go`,
      `internal/lsp/definition_test.go` and `internal/lsp/references_test.go` — `hover.go`,
      `definition.go` and `references.go` consume the walks Tasks 3 and 5 built and need no edit

**Affected Files/Modules:**
- `internal/lsp/model.go` — `referencesIn` (`:101-134`) for the edge's command name, and Task 3's
  invariant reference walk for the edge's invariant name
- `internal/lsp/hover_test.go`, `internal/lsp/definition_test.go`, `internal/lsp/references_test.go`

**Patterns to Follow:**
- `referencesIn`'s flow block (`internal/lsp/model.go:128-131`) — the sibling entry kind, and the one
  `add(...)` per site shape
- `tasks/us-009-show-rejection-paths-on-the-timeline.md` "Open questions, decided" items 1 and 3 — the
  separate AST collection, and that a rejection entry's command name is deliberately left unchecked by
  the validator exactly as a flow's is
- Task 3's invariant scope walk and Task 5's consumption of it
- `locationOf` (`internal/lsp/references_test.go:70`) and `tasks/learnings.md:396-404`

**Testable:** Yes — through `lsp.GetHover`, `lsp.GetDefinition` and `lsp.GetReferences`.

**Verification:** `go test -tags unit ./internal/lsp/...`; `go build ./...`.

**Depends on:** Task 3, Task 5, and US-009 Tasks 1-3 (delivered)

---

## Summary

**Nine tasks.**

**Ordering rationale — dependency-first, then the two hard external gates last.**

1. Tasks 1-3 build hover outward from what `GetHover` answers for today: the construct set first
   (Task 1, which also owns the quoted-name offset every later hover leaf depends on), then the
   description on top of it (Task 2), then the invariant, which is the only construct whose prose comes
   from a `Statement` rather than a `description` and the only one needing scope resolution (Task 3).
   Task 3's scope walk is the asset Tasks 5, 6 and 9 reuse, which is why it lands early despite being
   the third hover increment.
2. Tasks 4-5 are navigation, and split on the same seam: Task 4's spec element references drop into the
   existing flat `referencesIn`/`declaredNames` machinery and cost one `add(...)` per site, while
   Task 5's invariant references cannot (a flat name map cannot hold two scopes declaring one name) and
   so consume Task 3's walk instead.
3. Tasks 6-7 are completion, deliberately **invariant-first**. `valueSlotBefore` scans the typed words
   backwards, so with `rejected` claimed first a `then rejected ` cursor resolves correctly the moment
   `then` becomes an event slot. Reversed, Task 7 would ship an interval during which `then rejected `
   offered event names — green, committable, and wrong in the editor.
4. Task 8 (US-010) and Task 9 (US-009) sit last because each is blocked on a story that is decomposed
   and not implemented. Both are leaves: nothing below them depends on them, so the other seven tasks
   deliver the story's spec-side value with no wait, and each gated task can start the moment its
   dependency is delivered.

**Acceptance-criteria coverage.**

| Story criterion | Tasks |
|---|---|
| Hovering any construct shows its kind and description | 1 (kind and scope, every construct), 2 (description), 3 (the invariant, whose prose is a statement) |
| Hovering `rejected <name>` — in a spec or on a flow rejection edge — shows the invariant's prose | 3 (in a spec), 9 (on a flow rejection edge) |
| Completion offers invariant names after `rejected`, event names inside `given [...]`, and field names inside payload braces scoped to the referenced construct's `fields` | 6 (invariant names, scope-aware), 7 (event names in `given`, and the `when`/`then` slots and spec body keywords carried with them), 8 (payload field names) |
| Go-to-definition works from spec event/command references and from invariant references, including flow rejection edges | 4 (spec event and command references), 5 (invariant references in specs), 9 (flow rejection edges) |
| Find-references on an invariant lists the specs and flow edges that reference it | 5 (the specs), 9 (the flow edges) |

**Nothing is deferred out of the story**, but three boundaries inside the criteria are drawn narrower
than a maximal reading and are argued above rather than assumed:

- Criterion 1's "any construct" means a construct's **declaration**; hover on a reference site is not
  added, except at the two sites criterion 2 names explicitly ("Open questions, decided" item 1).
- Criterion 3's completion is **not** extended to the flow rejection edge, because the identifier after
  `rejected:` there is a command name and the story qualifies only criteria 2, 4 and 5 with the flow
  edge (item 7).
- `when` completion offers commands **and** events, and `then [...]` completion is carried alongside
  `given [...]`, because the same slot mechanism serves all three and a spec body that completed two of
  its four value positions would send the author back to the documentation the story is trying to
  replace (items 6 and 11).

**Three things are recorded as known and deliberately not fixed here**, so that an `internal/lsp` diff
from this story is not misread as introducing or absorbing them: `ConvertDiagnostics` rendering
`diagnostic.Info` as an LSP error (`internal/lsp/diagnostics.go:31-36`); `declarationsIn` being blind to
the event a translation declares inside itself, which leaves that event hoverable but not navigable
after Task 1 (item 9); and `keywordsFor(ctxEvent)`/`keywordsFor(ctxAutomation)` each being one entry
short once US-012's `type` and US-013's `after` land, which no test in the repo will report (item 2).

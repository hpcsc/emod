# Screens as a First-Class Element — Proposal

## Problem

Event Modeling has five building blocks. Four of them are nouns the model can name and refer to: an event, a command, a read model, and — through `automation` — a processor. The fifth is the screen, and in `emod` it is the one element with a name that nothing can say.

`trigger` is the screen. It carries an actor and a `reads` view, it occupies the lane both raster exporters label "Wireframes" (`internal/diagram/svg.go:40`, `internal/diagram/drawio.go:179`), Mermaid emits it as `ui`, and the web viewer draws it with a title bar across the top of its box so it reads as a monitor (`web/static/renderer.js:386`). The concept is present and correctly drawn. What is absent is identity.

**A trigger's name is decoration.** The cross-reference table (`docs/dsl-reference.md:766-772`) lists every declaration the language resolves — context, event, command, view, actor, invariant. `trigger` is not among them, and nothing anywhere in the model refers to a trigger by name. It is also excluded from the glossary by the same reasoning: "`slice`, `trigger`, `automation` and `translation` contribute no term of their own" (`docs/dsl-reference.md:754`).

The consequence is that a recurring screen silently splits in two. The same page named in two slices of one aggregate produces two unrelated nodes:

```
% Slice: One                  % Slice: Two
tf 01 ui C.Checkout Page      tf 04 ui C.Checkout Page
```

Both are labelled `Checkout Page`, neither knows about the other, and no diagnostic fires. Drawing the wireframe twice is *correct* — time flows left to right, so a screen recurs along the timeline — but the model cannot tell that it recurred. There is no object to hang the second appearance on.

Three further consequences follow.

**The Bed anti-pattern is not expressible.** Of the four anti-patterns on the canonical cheat sheet, `emod` implements two and approximates a third: `left-chair` counts flows per command (`internal/linter/linter.go:56-62`, `:558`), `god-view` counts subscriptions, and `spec/command-without-spec` covers the ground Shelf describes. Bed — one screen firing several commands — is missing, and it is missing *because* of the current shape. The rule is a question about a screen across slices, and there is no screen across slices to ask it of. This is not a rule someone forgot to write; it is one the AST cannot support.

**`reads` is single-valued and unresolved.** `Trigger.Reads` is one `string` (`internal/ast/ast.go:226`), so a page showing a cart and a discount panel cannot say so. It is also never resolved: `emod validate` accepts `reads DoesNotExistView` and exits 0. Only an automation's `reads` resolves (`docs/dsl-reference.md:772`).

**`actor` is a concept the model captures and never uses.** The cross-reference table claims `actor "<name>"` is referenced by `trigger { actor <Name> }`, but the validator package contains no actor check at all, so `actor NobodyDeclared` passes. Nothing else consumes it: every other use across the codebase is declaration plumbing — parse, format, export, import — and the single semantic consumer is `internal/glossary/glossary.go:115-123`, which walks triggers to group actors by context. Every `.emod` file in the repository declares exactly one actor, so in practice the attribute distinguishes nothing. A screen already says a human is present; `automation` says one is not. Naming *which* human is the only information `actor` adds, and no rule, diagram or lane consumes it.

**The mockup asset has nowhere to live.** `docs/proposals/triggers-and-automations-proposal.md:123-135` specified a `wireframe "<path>"` attribute on the trigger and it was never implemented — no keyword in the lexer, no field in the AST. As an attribute of a slice-local trigger it would have to be restated at every appearance of the page, which is the wrong place for a fact about the page.

## Goals

- Give a screen an identity: one declaration, referred to by name from every slice it appears in.
- Separate what belongs to the screen (who uses it, what it looks like) from what belongs to an appearance (which read models feed it here, which commands it fires here).
- Let one screen consult several read models, and resolve every name it states.
- Give the mockup asset a home where it is stated once.
- Make the Bed anti-pattern computable, closing the last gap against the canonical four.
- Retire `actor`, whose only reference site this change would otherwise have to carry forward.

## Non-Goals

- Layout, styling, or component structure. A screen names its mockup by path; the model does not describe what is on it. The line drawn for runtime scheduling semantics is drawn here for presentation.
- Navigation between screens. A screen graph is a different model with a different shape, and putting one inside the event model would be the read-side equivalent of drawing the call stack.
- Per-appearance mockups. A screen shows different data at different points on the timeline, but it is one design; a second asset means a second screen.
- Inlining images into `.emod` files. As before, a path is referenced, not embedded.
- Changing `view`, `automation` or `translation` wiring. Only the reader at the human end of `reads` changes.

---

## DSL Surface

### 1. `screen` is declared once, at the top level

```emod
screen CheckoutPage {
  image       "./wireframes/checkout.png"
  description "Where a guest reviews the cart and commits to the order."
}
```

`screen` replaces `trigger` outright: the keyword is retired, and nothing named `trigger` survives. This is a rename plus a split, not a new element beside the old one — a trigger already carries everything a screen is, and what it lacks is a name anything can refer to.

At model scope, alongside `context`. Context scope was the alternative and would keep a page used once in one aggregate out of a shared namespace, but a user-facing surface is not owned by a bounded context — the same page can be the entry point to two of them — and model scope is the reversible direction, since context scoping can be added later and ungrouping cannot.

Both entries are optional, so a screen with neither is `screen CheckoutPage {}` — a bare identity anchor. That is the honest cost of the declaration, and it buys three things: `image` is stated once rather than at every appearance, a mistyped appearance fails to resolve instead of silently becoming a second screen, and the declarations read as an inventory of the system's surfaces.

The name is a bare identifier rather than a quoted string, because in this language identifiers are what gets referred to: every element the cross-reference table resolves — `command PlaceOrder`, `event OrderPlaced`, `view CartView` — is declared with one, and with `actor` retired every quoted-name construct (`model`, `context`, `aggregate`, `slice`) is one nothing refers to. The rule becomes exceptionless: identifiers are referenced, quoted names are not.

The human label the current `trigger "Reservation Form"` carries goes with that choice: `screen ReservationForm` renders as `ReservationForm`, the way commands, events and views already render. `description` remains the slot for prose. A separate `title` attribute would restore the label and is not proposed, since a second prose slot earns its place only if the rendered diagrams read badly, which one model will settle.

The asset attribute is `image` rather than `wireframe` because Event Modeling uses "screen" and "wireframe" for the same building block, so `screen X { wireframe … }` reads as a thing containing itself; `mockup` and `sketch` share the defect less obviously. `image` names what the value is — a path to a picture file. The diagram lane keeps its "Wireframes" label, which is prose rather than grammar.

`image` takes a path relative to the `.emod` file, unresolved at parse time so a model stays usable before its mockups exist — a missing file is a lint warning.

### 2. A slice states a screen's appearance

```emod
slice "Apply Discount" {
  screen CheckoutPage {
    reads   [CartView, DiscountView]
    command ApplyDiscount
  }

  command ApplyDiscount { fields { ... } }
  event   DiscountApplied { fields { ... } }

  flow { command -> event: ApplyDiscount -> DiscountApplied }
}
```

The split follows the timeline. A screen's identity and its design are properties of the page and hold wherever it appears. What it displays and what it fires are properties of *this point in time*: the checkout page in "Apply Discount" reads the discount panel and fires `ApplyDiscount`; the same page in "Place Order" reads the cart and fires `PlaceOrder`. Neither half can hold both — which is why this is a top-level declaration with slice-local appearances rather than either one alone.

`command` is optional, and **absent means the screen fires nothing** — a display-only surface. It is not defaulted to every command the slice declares.

That default is what the graph builder does today (`internal/diagram/graph.go:76-81`): a trigger emits an edge to every command in its slice, so the edge is inferred and never stated. Carrying it forward breaks three ways. Two screens in one slice would each claim every command. A slice can hold commands no screen fires — `examples/all_patterns.emod` declares `ExpireReservation` in a slice where an *automation* issues it, so a screen added there would claim it, a defect that is latent today only because a slice holds at most one trigger. And `bed` would become a function of slice composition rather than of the screen: adding a third command to the slice trips the rule without anyone touching the screen.

Requiring `command` on every appearance fixes those and breaks the State View pattern, whose screen fires nothing by definition. Optional-with-an-empty-default fixes all four. It also follows the reasoning `on`/`every` already settled for automations — this language states the fact rather than inferring it, because an unstated fact cannot be checked against an implementation.

The cost is one line per migrated screen, and a rendering change for existing models: edges that were inferred are now written down.

A slice may hold any number of appearances, replacing the current `Trigger *Trigger` single slot.

### 3. `reads` becomes a list, and resolves

```emod
reads [CartView, DiscountView]
```

Every entry resolves to a view declared anywhere in the model, reported at its own position — the rule an automation's `reads` already follows. The bracketed form matches `subscribes [...]`, which is the other place the language lists view-or-event names. A single unbracketed name stays accepted as sugar for a one-element list.

### 4. The Bed rule

```
screen "CheckoutPage" fires 3 commands across 3 slices; consider whether it
is several screens, or whether the commands express one intent
```

`bed` walks appearances grouped by screen name and counts distinct commands, which is a model-wide map of exactly the shape `Lint()` already builds for `left-chair` and the spec rules (`internal/linter/linter.go:56-77`). Threshold 3, matching `left-chair`.

Two companion rules fall out of identity at no extra cost:

- `screen/never-shown` — a screen declared at top level that no slice ever shows.
- `screen/missing-image-asset` — an `image` path that resolves to no file.

---

## Internal Representation

### AST (`internal/ast/ast.go`)

`Trigger` is replaced by two types, and `Actor` is deleted along with `Model.Actors` (`:25`, `:31-39`). The declaration hangs off `Model` in its place:

```go
type Screen struct {
	Comments       []*Comment
	Name           string
	NamePos        Position
	Description    string
	DescriptionPos Position
	Image          string
	ImagePos       Position
	OpenPos        Position
	ClosePos       Position
}
```

The appearance hangs off `Slice`, replacing `Trigger *Trigger` (`:82`):

```go
type ScreenRef struct {
	Comments    []*Comment
	Name        string
	NamePos     Position
	Reads       []string
	ReadsPos    []Position
	Commands    []string
	CommandsPos []Position
	OpenPos     Position
	ClosePos    Position
}
```

`Reads`/`ReadsPos` as parallel slices follow `View.Subscribes`/`SubscribesPos` (`:239-240`). `Model.Screens []*Screen` and `Slice.Screens []*ScreenRef` are the two new fields.

### Lexer (`internal/lexer`)

`KeywordTrigger` (`token.go:21`, `:84`) and `KeywordActor` are retired; `KeywordScreen` and `KeywordImage` are added. `reads` and `command` already exist. Section "Risks" covers the identifier-space cost.

### Parser (`internal/parser/parser.go`)

`parseTrigger` (`:717-800`) splits into `parseScreen` at model scope and `parseScreenRef` at slice scope, dispatched by position rather than by lookahead — the top-level branch takes the slot the retired `actor` branch (`:156`) vacates, the slice branch replaces `:393`. `retiredTriggerKindMessage` (`:802`) gains a sibling that reports `trigger` itself as retired, so a stale file gets a migration message rather than a bare syntax error.

### Formatter (`internal/formatter/formatter.go`)

`writeTrigger` (`:208-213`) becomes `writeScreen` and `writeScreenRef`. The actor loop (`:101`) is deleted and top-level screens take its position, written through an identifier-named variant of `writeDeclaration` (`:86`), which today quotes the name it is given. Appearances format first inside the slice, where the trigger sits today (`:165-167`). A `reads` list wraps by the same rule as `subscribes`.

### Validator (`internal/validator/validator.go`)

Three checks, all new to this area:

- A screen name declared twice is an error, following `redeclaredInvariantDiagnostics` (`:298-305`) — the existing model for scoped redeclaration.
- An appearance naming an undeclared screen is an unresolved reference.
- Each `reads` entry resolves to a declared view, matching the automation message so all readers report alike.

The actor row leaves the cross-reference table rather than gaining the check it never had.

### Linter (`internal/linter/linter.go`, `descriptions.go`)

`bed`, `screen/never-shown` and `screen/missing-image-asset`, each with a `descriptions.go` entry so `lint explain` covers them. The screen-to-command map builds alongside `flowCount` (`:56-62`).

### LSP (`internal/lsp`)

`model.go:125-126` adds view references from `slice.Trigger.Reads` and becomes a loop over appearances. Screens gain go-to-definition and find-references for the first time — an appearance jumps to the declaration, and the declaration lists its appearances, which is the feature that makes a recurring screen navigable. Completion offers declared screen names inside a slice and view names inside `reads`.

### CUE schema (`internal/cue/schema.cue`)

`#Trigger` (`:15`) becomes `#Screen` and `#ScreenRef`; `trigger?: #Trigger` on the slice (`:109`) becomes `screens?: [...#ScreenRef]`, and the model definition trades its `actors` key for `screens?: [...#Screen]`.

### JSON export (`internal/export/json.go`)

`jsonTrigger` (`:144`) splits the same way; `Trigger *jsonTrigger` on the slice (`:79`) becomes a list, and the model's `Actors` key (`:32`, `:331`, `:348-352`) is replaced by `screens`. `convertTrigger` (`:648`) splits into two converters. The CUE writer drops `writeActor` (`internal/export/cue.go:84-88`, `:188`), and `internal/export/diagram.go` loses `appendActor` (`:153`, `:455`) while `:393` maps the reads edge and needs the loop.

### Importer (`internal/importer/importer.go`)

`appendTrigger` (`:191-204`) takes the first `trigger` child and discards the rest (`:196`) — the single-slot assumption made concrete. It becomes `appendScreens`, iterating, plus a model-scope pass building declarations; the actor pass (`:109`) is deleted. Where a round-tripped diagram has appearances with no declaration, the importer synthesises one per distinct name so the result parses.

### Glossary (`internal/glossary/glossary.go`)

Screens become glossary terms, which reverses the reference manual's statement that a trigger defines none. The actor machinery goes with the construct: `triggerActorNames` (`:115-123`), `actorTerms`, `actorDescriptions` (`:136`) and `unreferencedActorTerms` (`:144`) are deleted, and the `Actors` field drops off both the top-level and per-context sections (`:9`, `:20`, `:39-40`, `:55`). Screens take the per-context slot, grouped by the contexts their appearances fall in — the derivation `triggerActorNames` performed for actors, now performed for the construct that actually recurs.

### Tree-sitter and editors

`trigger_definition` (`editors/tree-sitter-emod/grammar.js:274-276`) and its use in the slice rule (`:98`) split into `screen_definition` at model scope and `screen_reference` in the slice, and the actor rule is deleted. Highlight queries, the corpus, and the VS Code TextMate grammar follow; `highlights.scm:67-74` loses both `actor` and `trigger` from its quoted-entity group, and screens join the identifier-named group at `:76-81`.

### Web viewer (`internal/viewer/static`, `web/static`)

`layout.js:86` collects triggers by node type and `:132` places them in the top column; both read `"screen"` instead. The edge map (`web/static/model.js:143`, `:148`) renames `trigger>command` and `view>trigger`. `legend.js:9`, `:22-23` relabel. The renderer's title-bar screen shape (`web/static/renderer.js:386`) is unchanged — it was already drawing a screen.

---

## Diagrams

**One box per appearance, one identity behind them.** The timeline still draws a screen once per slice it appears in; nothing about the visual changes, because drawing the recurrence is right. What changes is that both boxes now carry the same declared identity, so the viewer can highlight every appearance of a screen when one is selected, and the detail panel can list them.

**Edges.** `EdgeTriggerReads` and `EdgeTriggerCommand` (`internal/diagram/graph.go:11-14`) keep their meaning and are renamed. `EdgeTriggerReads` becomes one edge per `reads` entry rather than one per trigger; `EdgeTriggerCommand` is emitted from the appearance's `command` list, replacing the loop over every slice command at `:76-81`, so an appearance that states none draws none.

**ASCII.** `internal/diagram/ascii.go:52-54` prints the trigger and drops its `reads` edge, so a display-only screen shows as an unconnected `<<…>>` next to an unrelated `(Event) -> {View}` line. SVG and draw.io draw the edge. With `reads` resolved the omission becomes conspicuous, and ASCII should print `{CartView} -> <<CheckoutPage>>` like the others.

---

## Worked Example

```emod
emod 1

model "Storefront"

screen CheckoutPage {
  image "./wireframes/checkout.png"
}

screen OrderHistoryPage {}

context "Ordering" {
  aggregate "Order" {
    slice "Apply Discount" {
      screen CheckoutPage {
        reads   [CartView, DiscountView]
        command ApplyDiscount
      }

      command ApplyDiscount { fields { orderId string required; code string required } }
      event   DiscountApplied { fields { orderId string required; code string required } }

      flow { command -> event: ApplyDiscount -> DiscountApplied }
    }

    slice "Place Order" {
      screen CheckoutPage {
        reads   [CartView]
        command PlaceOrder
      }

      command PlaceOrder { fields { orderId string required } }
      event   OrderPlaced { fields { orderId string required; placedAt timestamp required } }

      flow { command -> event: PlaceOrder -> OrderPlaced }
    }

    slice "View Cart" {
      view CartView {
        fields { orderId string required; total decimal required }
        subscribes [DiscountApplied, OrderPlaced]
      }
    }

    slice "View Order History" {
      view OrderHistoryView {
        fields { orderId string required; placedAt timestamp required }
        subscribes [OrderPlaced]
      }

      screen OrderHistoryPage {
        reads [OrderHistoryView]
      }
    }
  }
}
```

Two facts this model states that the current grammar cannot. `CheckoutPage` is one page appearing at two points on the timeline, so `bed` can see it fires two commands and the viewer can link the appearances. And "View Order History" is a State View slice that ends at a screen with no command — the read-side pattern terminating in a wireframe, which the cheat sheet draws and which today can only be spelled as a trigger borrowed from the command chain.

---

## Interaction With `user-stories/lint-unread-view.md`

That story set stays separate and should ship first. It is non-breaking, independently valuable, and its Non-Goals already set this boundary deliberately: multi-valued `reads` is "a grammar change with its own consequences", and a distinct `screen` element "should not be settled by a lint rule". Both judgements hold. Folding a ready lint rule into a breaking restructure would block the first on a decision the second has not yet earned.

They meet at three points.

- **US-002 is subsumed for the human reader.** Resolving a trigger's `reads` is a precondition of this proposal, which needs it resolved *and* listed. If US-002 ships first, this proposal inherits the resolution and widens it to a list. If this proposal ships first, US-002 narrows to translations alone.
- **`view/never-read` must count appearances.** US-001 builds its `readViews` set from `slice.Trigger.Reads`, `slice.Automations[].Reads` and `slice.Translations[].Reads`. After this change the first term is a loop over `slice.Screens[].Reads`. One line, and the rule silently under-reports if it is missed.
- **Its third open question closes.** "Does a view read only by a trigger deserve different treatment?" is asked because a trigger's `reads` cannot prove consumption while unresolved. Under either change it can, and the answer is no.

---

## Versioning

`ast.SupportedVersion` (`internal/ast/ast.go:4`) stays at 1. This is a breaking grammar change and the version header exists to absorb exactly that kind of change, but the tool is still under development and the models using the retired spelling are the examples, fixtures and documentation this proposal rewrites anyway. Spending the bump here would buy a migration diagnostic for files that do not need one.

A file still using `trigger` therefore fails at the retired spelling rather than with a versioning message. The retired-spelling error described under Parser is what makes that failure legible — it names the replacement at the position of the old keyword, which is the whole of what a migration message would have said.

The transformation is mechanical enough to do by hand: hoist each `trigger "Some Name"` to a top-level `screen SomeName`, and leave a reference in the slice.

---

## Phased Implementation

### Phase 1: `reads` resolves and lists

Additive; no rename. `Trigger.Reads` becomes `[]string`, resolves against declared views, and draws one edge per entry. `emod fmt` learns the bracketed form; ASCII learns the edge.

Ships the largest correctness win on its own, and is the half `lint-unread-view` US-002 depends on.

### Phase 2: Identity

The breaking phase, and since the tool is under development it goes directly with no deprecation window. Top-level `screen`, slice-level appearances, `trigger` retired with an error naming its replacement, and `actor` deleted end to end — keyword, AST, parser, formatter, both exporters, importer, glossary and grammars. Examples, `docs/dsl-reference.md:289-341` and its cross-reference table, the README quick-start, tree-sitter, TextMate and the viewer all move together.

Retiring `actor` roughly doubles the deletion surface of this phase without touching its design. It rides along because it shares every file the rename already opens, and because leaving it would mean carrying a construct whose sole reference site this change removes.

### Phase 3: What identity unlocks

`image` end to end; `bed`, `screen/never-shown`, `screen/missing-image-asset`; screens in the glossary; LSP go-to-definition and find-references across appearances; appearance highlighting in the viewer.

Phase 1 stands alone and should ship regardless. Phase 3 is the payoff and is why phase 2 is worth its cost — none of it is reachable while a screen has no name to refer to.

---

## Risks and Open Questions

**`screen` reads as two constructs.** The top-level form declares and the slice form refers, dispatched by position. The previous proposal retired `trigger` partly for being "one token, two unrelated meanings" — the difference is that those meanings were unrelated (a wireframe and an activation event) where these are a declaration and its instantiation, the relationship `command` already has between its slice declaration and `automation { command X }`. A distinct keyword for the appearance — `shows CheckoutPage { … }` — would remove the objection and cost another reserved word to avoid an explanation one sentence long. It is still one more thing to explain.

**`screen` and `image` become reserved words.** The exposure is the same as for `on` and `every`: field names, types, modifiers and tag keys accept keywords via `checkIdentifierLike`, so `fields { screen string required }` keeps working. What breaks is a lowercase *element* named `screen` or `image`. The keyword table is lowercase and lookup is case-sensitive, so conventionally-capitalised names are unaffected. `image` is the more exposed of the two, being a plausible domain noun.

**Screen-to-command edges must be written down.** Making `command` explicit removes three defects, but it converts an inferred edge into an authored one: a migrated model that omits it silently loses arrows it used to draw. The migration is mechanical and the arrows reappear the moment the entry is added, but nothing fails loudly when it is missing — a screen with no `command` is a legitimate display-only surface, so the linter cannot tell the two apart. `screen/never-shown` catches an unused screen; it does not catch a screen that should fire something and does not.

**Is Bed worth a grammar change on its own?** No — and it is not offered as one. It is the clearest evidence that the missing identity has a cost, because it is a canonical rule the AST cannot express, but the case rests on the mockup's home, the multi-view `reads` and the navigable recurring screen together.

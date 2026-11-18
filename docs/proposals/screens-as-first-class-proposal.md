# Screens as a First-Class Element — Proposal

## Problem

Event Modeling has five building blocks. emod gives four of them a name that the model can refer to: an event,
a command, a view, and a processor through `automation`. The fifth building block is the screen. In emod, it
is the one element with a name that nothing can refer to.

`trigger` is the screen. It carries an actor and a `reads` view. The SVG draws it in the lane labelled
"Wireframes" (`internal/diagram/lane_positions.go:16`), Mermaid writes it as `ui`
(`internal/diagram/mermaid.go:79-81`), and the web viewer draws it with a bar across the top of its box, so
that it looks like a monitor (`internal/frontend/static/renderer.js:520`). The concept is present, and the
diagrams draw it correctly. What is absent is identity.

**The name of a trigger refers to nothing.** The cross-reference table (`docs/dsl-reference.md:801-813`)
lists each declaration that the language resolves: context, event, command, view, actor and invariant.
`trigger` is not in the table, and no element of the model refers to a trigger by its name. The glossary
leaves it out for the same reason: "`slice`, `trigger`, `automation` and `translation` contribute no term of
their own" (`docs/dsl-reference.md:797`).

As a result, a screen that recurs splits into two unrelated elements. Here, one page appears in two slices
of one aggregate:

```hcl
slice "Apply Discount" {
  trigger "Checkout Page" {
    reads = CartView
  }
}

slice "Place Order" {
  trigger "Checkout Page" {
    reads = CartView
  }
}
```

Both triggers have the label `Checkout Page`. Neither knows about the other, and `emod validate` reports
nothing. It is correct to draw the wireframe two times: time goes from left to right, so a screen recurs
along the timeline. But the model cannot tell that the screen recurred. No object exists that the second
appearance can refer to.

The drawn diagrams then join the two triggers incorrectly. The SVG keeps one box for each name across all
slices (`internal/diagram/svg.go:125-127`), and the diagram JSON keeps one ID for each trigger name
(`internal/export/diagram.go:156`). When each slice above also declares its command, the diagram JSON gives
all four arrows of the two triggers to the box in "Place Order". The box in "Apply Discount" gets no arrow,
and the box in "Place Order" issues `ApplyDiscount`, the command of the other slice.

Four more consequences follow.

**The Bed anti-pattern cannot be stated.** The canonical cheat sheet has four anti-patterns. emod implements
two and covers a third approximately:

- `left-chair` counts the flows of each command (`internal/linter/linter.go:57-63`, `:605-608`).
- `god-view` counts the subscriptions of each view (`:670-673`).
- `spec/command-without-spec` covers the ground that Shelf describes.

Bed is one screen that issues many commands, and emod does not have it. The cause is the current shape: the
rule asks a question about one screen across slices, and the model has no screen across slices. Nobody forgot
to write this rule. The AST cannot support it.

**`reads` holds one view.** `Trigger.Reads` is one `string` (`internal/ast/ast.go:228`), so a page that shows
a cart and a discount panel cannot say so. The validator resolves the name
(`internal/validator/validator.go:400-401`), but one name is all that it can resolve.

**The model records `actor` and never checks it.** The cross-reference table says that `trigger { actor }`
refers to `actor "<name>"`. But the validator package has no check for an actor, so `actor = NobodyDeclared`
passes. Each `.emod` file in the repository that declares an actor declares exactly one, so in practice the
attribute tells no trigger apart from another. The model only shows the name:

- The SVG and the ASCII diagram add it to the label of the trigger (`internal/diagram/svg.go:173-175`,
  `internal/diagram/ascii.go:121-123`).
- The event flow diagram uses it as a caption (`internal/diagram/eventflow.go:259-262`).
- The viewer lists it in a bar above the diagram and in the detail panel (`internal/frontend/static/ui.js:176-186`,
  `:539`).
- The glossary groups actors by context (`internal/glossary/glossary.go:115-126`).

No rule, lane or check reads it. A screen already says that a person is present, and `automation` says that
no person is present. The only information that `actor` adds is which person, and nothing uses that
information except a label.

**The mockup has no place.** `docs/proposals/completed/triggers-and-automations-proposal.md:124` specified a
`wireframe` attribute on the trigger. Nobody implemented it: the parser has no such attribute and the AST has
no field. A trigger belongs to one slice, so as an attribute of a trigger the path must be stated again at
each appearance of the page. That is the wrong place for a fact about the page.

## Goals

- **G1:** Give a screen an identity: one declaration, which each slice that shows the screen refers to by
  name.
- **G2:** Separate what belongs to the screen from what belongs to one appearance. The screen has its name,
  its description and its mockup. An appearance has the views that feed the screen at that point, and the
  commands that the screen issues there.
- **G3:** Let one appearance read more than one view, and resolve each name that an appearance states.
- **G4:** Give the mockup a place where the model states it one time.
- **G5:** Make the Bed anti-pattern computable. This closes the last gap against the canonical four.
- **G6:** Retire `actor`. Its only reference site is the trigger, which this change replaces.

## Non-Goals

- Layout, style, or the structure of components. A screen names its mockup by its path. The model does not
  describe what is on the mockup. The triggers-and-automations proposal kept the runtime semantics of
  schedules out of the model, and this proposal keeps presentation out in the same way.
- Navigation between screens. A screen graph is a different model with a different shape. It belongs in an
  event model no more than a call stack does.
- A mockup for each appearance. A screen shows different data at different points on the timeline, but it
  has one design. A second mockup means a second screen.
- Images inside `.emod` files. A model refers to an image by its path and does not embed it.
- Changes to how `view`, `automation` or `translation` connect. Only the reader at the human end of `reads`
  changes.

---

## DSL Surface

### 1. `screen` is declared one time, at the top level

```hcl
screen "CheckoutPage" {
  description = "Where a guest reviews the cart and commits to the order"
  image       = "./wireframes/checkout.png"
}
```

`screen` replaces `trigger` completely. The parser retires the `trigger` keyword, and no construct named
`trigger` stays. This is a rename and a split, not a new element next to the old one. A trigger already
carries all that a screen is. What it does not have is a name that another element can refer to.

The declaration is at the top level, next to `context`. Context scope is the alternative. It keeps a page that
one aggregate uses one time out of a namespace that all contexts share. But a bounded context does not own a
surface that users see: one page can be the entry point to two contexts. Also, a later change from model
scope to context scope is possible, but a change in the other direction is not.

Both attributes are optional, so a screen with neither is `screen "CheckoutPage" {}`: an anchor for identity
and nothing more. That is the real cost of the declaration, and it buys three things:

- The model states `image` one time, not at each appearance.
- An appearance with a typo does not resolve, so it does not silently become a second screen.
- The declarations give a list of the surfaces of the system.

The label of a screen is a name, like the label of a command. In HCL, a label is a quoted string and a
reference is a bare name ([HCL syntax proposal](hcl-syntax-proposal.md)). A bare name cannot hold a space, so
a screen has a label such as `CheckoutPage`, not `Checkout Page`. Each element in the cross-reference table
follows this rule: the model declares `command "PlaceOrder"` with a label and refers to it as `PlaceOrder`.

The human label that `trigger "Reservation Form"` carries goes with this rule. `screen "ReservationForm"`
renders as `ReservationForm`, as commands, events and views render now. `description` stays the place for
prose. A separate `title` attribute can restore the human label. This proposal does not add it: a second
place for prose earns its place only if the diagrams read badly, and one model will show that.

The attribute for the mockup is `image`, not `wireframe`. Event Modeling uses "screen" and "wireframe" for
the same building block, so `screen "X" { wireframe = … }` reads as a thing that contains itself. `mockup`
and `sketch` have the same defect, but less clearly. `image` names what the value is: the path to a picture
file. The diagram lane keeps its label "Wireframes", because the label is prose and not grammar.

`image` takes a path relative to the `.emod` file. The parser does not resolve it, so a model stays usable
before its mockups exist. A path that leads to no file is a lint warning.

### 2. A `shows` block states an appearance

```hcl
slice "Apply Discount" {
  shows {
    screen  = CheckoutPage
    reads   = [CartView, DiscountView]
    command = ApplyDiscount
  }

  command "ApplyDiscount" { ... }
  event "DiscountApplied" { ... }

  flow = <<-FLOW
    command -> event: ApplyDiscount -> DiscountApplied
  FLOW
}
```

A `shows` block has no label. Its `screen` attribute names the declared screen with a bare name, as
`target { context = Notifications }` names a context. Therefore each reference in the language is a bare
name, and each label declares something. A slice can hold any number of `shows` blocks. They replace the one
`trigger` slot of the slice.

The alternative is a `screen "CheckoutPage"` block in the slice as well, where the position of the block
tells a declaration from an appearance. That form makes the label in the slice a reference, the one exception
to the rule above, and it gives one keyword two meanings. HCL has no reserved words, so the separate block
type takes no name away from a field or a label.

The split follows the timeline. The identity and the design of a screen are properties of the page, and they
hold wherever the page appears. What the page shows and what it issues are properties of one point in time.
The checkout page in "Apply Discount" reads the discount panel and issues `ApplyDiscount`. The same page in
"Place Order" reads the cart and issues `PlaceOrder`. Neither half can hold both facts. That is why the
screen is a top-level declaration with appearances in slices, and not only one of the two.

`command` is optional, and **when `command` is absent, the screen issues no command.** The screen is then a
surface that only shows data. `command` does not default to each command that the slice declares.

That default is what the graph builder does now (`internal/diagram/graph.go:75-85`): a trigger gets an arrow
to each command of its slice, so the model infers the arrow and never states it. If the default stays, three
problems follow:

1. Two appearances in one slice each claim each command of the slice.
2. A slice can hold a command that no screen issues. In `examples/all_patterns.emod`, the slice "Expire
   Unconfirmed Reservations" declares `ExpireReservation`, and an automation issues it. A screen in that slice
   then claims it too. The defect is hidden now only because a slice holds at most one trigger.
3. `bed` becomes a function of what the slice holds, not of the screen. A third command in the slice trips
   the rule, and nobody changes the screen.

A `command` that each appearance must state fixes those three problems, but it breaks the State View
pattern, where the screen issues nothing. An optional `command` with an empty default fixes all four. It also
follows the reason that settled `on` and `every` for automations: the language states the fact and does not
infer it, because nobody can check an inferred fact against an implementation.

`command` takes one name or a list, as `reads` does: `command = [ApplyDiscount, RemoveDiscount]`.

The cost is one line for each migrated screen. The models that exist now also render differently: the arrows
that the graph builder infers become arrows that the author writes.

### 3. `reads` takes a list

```hcl
reads = [CartView, DiscountView]
```

Each entry resolves to a view that the model declares anywhere, and the validator reports each entry at its
own position. The list form matches `subscribes = [...]`, the other place where the language lists names. A
single bare name, `reads = CartView`, stays valid as a list of one, so each model that exists now still
parses. `emod fmt` writes one entry as a bare name and two or more entries as a list.

An automation and a translation keep one `reads` name. A processor reads one view as its todo list.

### 4. The Bed rule

```
screen "CheckoutPage" issues 3 commands across 3 slices; consider whether it
is several screens, or whether the commands express one intent
```

`bed` groups the appearances by screen and counts the distinct commands. `Lint()` already builds maps of this
shape across the whole model, for `left-chair` (`internal/linter/linter.go:57-63`) and for `view/never-read`
(`:65-84`). The threshold is 3, the same as for `left-chair` (`:606`).

Two more rules come from identity at no extra cost:

- `screen/never-shown`: the model declares a screen that no slice shows.
- `screen/missing-image-asset`: an `image` path leads to no file.

---

## Internal Representation

### AST (`internal/ast/ast.go`)

Two types replace `Trigger` (`:220-232`). `Actor` (`:31-39`) and `Model.Actors` (`:25`) go. The declaration
belongs to `Model`:

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

The appearance belongs to `Slice`, and replaces `Trigger *Trigger` (`:82`):

```go
type Appearance struct {
	Comments    []*Comment
	Screen      string
	ScreenPos   Position
	Reads       []string
	ReadsPos    []Position
	Commands    []string
	CommandsPos []Position
	OpenPos     Position
	ClosePos    Position
}
```

`Reads` and `ReadsPos` are parallel slices, as `View.Subscribes` and `View.SubscribesPos` are (`:241-242`).
The two new fields are `Model.Screens []*Screen` and `Slice.Appearances []*Appearance`.

### Parser (`internal/parser/parser.go`)

The HCL reader dispatches each block by its type and its position, so it needs no lookahead.

- At the top level, `screen` takes the case that the retired `actor` case leaves (`:199-200`).
- In a slice, `shows` replaces the `trigger` case (`:377-378`).
- `r.trigger` (`:408-435`) becomes `r.screen` and `r.appearance`. `r.actor` (`:247-265`) goes.
- `reads` and `command` in a `shows` block accept a bare name or a list. `r.names` (`:126-140`) accepts only a
  list, so the reader needs a form that accepts both.

`retiredTriggerKind` (`:398-406`) already reports the trigger kinds that the language retired before. A
`trigger` block and an `actor` block get the same treatment: an error at the keyword that names the
replacement, not the general message for an unexpected block (`:203-205`, `:394`).

### Formatter (`internal/formatter/formatter.go`)

The actor loop (`:26-31`) goes, and the top-level screens take its position. The `trigger` block in
`writeSlice` (`:87-94`) becomes a `shows` block for each appearance, first in the slice. A `reads` or
`command` with two or more entries uses `refListTokens`, the writer that `subscribes` uses (`:164`).

### Validator (`internal/validator/validator.go`)

Four checks:

- A screen name that the model declares two times is an error. `redeclaredInvariantDiagnostics` (`:300`) is
  the model for a redeclared name.
- An appearance that names a screen the model does not declare is an unresolved reference.
- Each `reads` entry resolves to a declared view, with the message that the automation and translation
  readers give now. The check at `:400-401` becomes a loop.
- Each `command` entry resolves to a declared command, with the rule for the `command` of an automation.

The actor row leaves the cross-reference table. It does not get the check that it never had.

### Linter (`internal/linter/linter.go`, `descriptions.go`)

- `bed`, `screen/never-shown` and `screen/missing-image-asset` are new, each with an entry in
  `descriptions.go`, so that `emod lint --explain` covers them. The map from each screen to its commands is
  built next to `flowCount` (`:57-63`).
- `view/never-read` builds its set of read views from `slice.Trigger.Reads` (`:68-70`). That term becomes a
  loop over each `reads` entry of each appearance.
- The message of `view/never-read` (`:667`) and its description (`descriptions.go:53`) name a trigger. They
  name a screen after this change.

### Name resolution (`internal/query`)

`internal/query` resolves the names of a model for `emod references`, `emod trace` and the LSP.

- `KindTrigger` (`references.go:13`) becomes `KindScreen`, and `TraceKinds` (`trace.go:14`) lists it.
- `names.go:135-137` adds one view reference for the `reads` of a trigger. It becomes a loop over the
  appearances, and it also adds a screen reference for the `screen` of each appearance and a command
  reference for each `command` entry.
- A trigger issues the commands of its slice "with no DSL word" (`references.go:58`). After this change, the
  word is `command` in a `shows` block. `addCommandReferences` (`:257-264`) reads that entry and not each
  command of the slice. The `reads` references (`:310-311`) become a loop.
- The trace of a screen follows the renamed arrows (`trace.go:165-166`, `:250-253`).

### LSP (`internal/lsp`)

- Go-to-definition and find-references work for a screen for the first time. An appearance goes to the
  declaration, and the declaration lists its appearances. This is the feature that lets a reader follow a
  screen that recurs. `referenceTargetKinds` (`references.go:13`) adds `KindScreen`.
- `declaredConstructs` declares actors (`model.go:76-77`) and triggers (`:89-90`). Screens replace both.
- `keywordsFor` offers `actor` at the top level (`completer.go:486`) and `trigger` in a slice (`:492`). They
  become `screen` and `shows`. Completion offers the declared screens after `screen =` in a `shows` block.
- `hover.go` describes `actor`, `trigger` and `reads` (`:13`, `:21`, `:31`). It describes `screen`, `shows`,
  `image` and the new `reads` after this change.
- `semantictokens.go` paints actors as parameters (`:59`, `:123`). Screens take their place.

### CUE schema (`internal/cue/schema.cue`)

`#Trigger` (`:15-21`) becomes `#Screen` and `#Appearance`. `trigger?: #Trigger` on the slice (`:111`) becomes
`shows?: [...#Appearance]`. The model replaces `actors?: [...#Actor]` (`:156`) with `screens?: [...#Screen]`,
and `#Actor` (`:146-150`) goes.

### JSON and CUE exports (`internal/export`)

- `json.go`: `jsonTrigger` (`:145-156`) splits in the same way. `Trigger *jsonTrigger` on the slice (`:79`)
  becomes a list under the key `shows`. The `actors` key of the model (`:32`, `:334`, `:351-362`) becomes
  `screens`. `convertTrigger` (`:652`) splits into two converters.
- `cue.go`: `writeActor` (`:84-88`) goes, and `writeTrigger` (`:121-122`, `:184-188`) splits.

### Diagram JSON and the importer

The diagram JSON (`internal/export/diagram.go`) feeds the web viewer. The importer (`internal/importer`)
reads it back when the viewer writes an edit to the source.

- The diagram JSON keys each trigger node by its name alone (`triggerIDs`, `:156`, `:378`), which causes the
  defect in the Problem section. An appearance gets a key for its slice, as an event, a view and a command
  do through `DeclaredElement` (`:161-165`).
- Each declaration travels as a node with no parent, of type `screen`, in the place that actor nodes have
  now (`appendActor`, `:197`, `:560-561`). Each appearance is a node of type `appearance` in its slice.
  `appendTrigger` (`:373-390`) becomes `appendAppearances`, and the arrows at `:482-492` follow the renamed
  arrows. The `Actor` field of a node (`:31`) goes.
- `appendTrigger` in `importer.go` (`:306-312`) takes the first `trigger` child of a slice and drops the
  rest. That is the rule of one trigger for each slice, in code. It becomes `appendAppearances` and takes
  each one. The actor pass (`:218-219`) becomes the pass that builds the screen declarations. When a diagram
  has appearances with no declaration, the importer makes one declaration for each distinct name, so that
  the result parses.
- `diagram_edit.go` refuses a second trigger in a slice (`holdsTrigger` and `canHold`, `:275-335`), and so
  does the viewer (`internal/frontend/static/model.js:161`). Both checks go.

### Glossary (`internal/glossary`)

Screens become glossary terms. This reverses the statement in the reference manual that a trigger defines no
term. The actor code goes with the construct: `triggerActorNames` (`glossary.go:115-126`), `actorTerms`
(`:128`), `actorDescriptions` (`:136`) and `unreferencedActorTerms` (`:144`). The `Actors` field goes from the
top-level section and from each context section (`:9`, `:20`, `:39-40`, `:55`), and `markdown.go` loses its
"Actors" heading (`:22`, `:45`, `:63`).

Screens take the slot of the actors in each context section, grouped by the contexts where their appearances
are. `triggerActorNames` did this for actors. After this change, the glossary does it for the construct that
actually recurs.

### Web viewer (`internal/frontend/static`)

- `layout.js` collects triggers by node type (`:93`) and places them in the top column (`:156`). Both read
  appearance nodes.
- The edge map in `model.js` (`:266`, `:271`) renames `trigger>command` and `view>trigger`. `config.js`
  (`:45`, `:67`, `:77`) and `legend.js` (`:9`, `:22-23`) follow.
- The detail panel shows the actor of a trigger and its one view (`ui.js:535-540`). For an appearance, it
  shows the views and the commands of the appearance, the image of the screen, and the other appearances of
  the screen.
- The bar that lists the actors (`ui.js:176-186`, `viewer.js:140`) goes.
- The renderer draws a screen with a bar across the top of its box (`renderer.js:520-521`). That does not
  change: the renderer already draws a screen.

### Editors

The VS Code extension maps `.emod` to the HCL grammar, so it has no list of keywords to change. The LSP gives
the semantic tokens and the completion.

---

## Diagrams

**One box for each appearance, with one identity behind them.** The timeline still draws a screen one time in
each slice where it appears. The look of the diagram does not change, because it is correct to draw the
recurrence. What changes is that each box of one screen carries the same declared identity. The viewer can
highlight each appearance of a screen when the user selects one, and the detail panel can list them.

**Boxes keyed by slice.** The SVG and the diagram JSON key an appearance by its slice, not by its name alone.
Otherwise, the arrows of one appearance end at the box of another (see the Problem section).

**Arrows.** `EdgeTriggerReads` and `EdgeTriggerCommand` (`internal/diagram/graph.go:11-14`) keep their
meaning and become `EdgeScreenReads` and `EdgeScreenCommand`. `EdgeScreenReads` becomes one arrow for each
`reads` entry, not one for each trigger. `EdgeScreenCommand` comes from the `command` list of the
appearance. It replaces the loop over each command of the slice (`:79-84`), so an appearance that states no
command draws no arrow to a command.

**Labels.** The SVG and the ASCII diagram add the actor to the label of a trigger
(`internal/diagram/svg.go:173-175`, `internal/diagram/ascii.go:119-125`). The label of an appearance is the
name of its screen.

**ASCII.** `internal/diagram/ascii.go:52-54` prints the trigger and leaves out its `reads` arrow. A screen that
only shows data therefore appears as an unconnected `<<…>>` next to an unrelated `(Event) -> {View}` line. The
SVG and the viewer draw the arrow. ASCII prints `{CartView} -> <<CheckoutPage>>`, as it already prints
`{View} -> [System]` for a translation (`:98-99`).

**Event flow.** `commandIssuers` (`internal/diagram/eventflow.go:319-341`) takes the trigger of a slice as the
issuer of each command of the slice, with the actor as the caption (`:258-270`). After this change, the
issuers come from the `command` lists of the appearances, and the caption is `screen`.

---

## Worked Example

```hcl
emod = 1

model "Storefront" {}

screen "CheckoutPage" {
  image = "./wireframes/checkout.png"
}

screen "OrderHistoryPage" {}

context "Ordering" {
  aggregate "Order" {
    slice "Apply Discount" {
      shows {
        screen  = CheckoutPage
        reads   = [CartView, DiscountView]
        command = ApplyDiscount
      }

      command "ApplyDiscount" {
        fields {
          orderId = required(string)
          code    = required(string)
        }
      }

      event "DiscountApplied" {
        fields {
          orderId = required(string)
          code    = required(string)
        }
      }

      flow = <<-FLOW
        command -> event: ApplyDiscount -> DiscountApplied
      FLOW
    }

    slice "Place Order" {
      shows {
        screen  = CheckoutPage
        reads   = CartView
        command = PlaceOrder
      }

      command "PlaceOrder" {
        fields {
          orderId = required(string)
        }
      }

      event "OrderPlaced" {
        fields {
          orderId  = required(string)
          placedAt = required(timestamp)
        }
      }

      flow = <<-FLOW
        command -> event: PlaceOrder -> OrderPlaced
      FLOW
    }

    slice "View Cart" {
      view "CartView" {
        subscribes = [DiscountApplied, OrderPlaced]

        fields {
          orderId = required(string)
          total   = required(decimal)
        }
      }
    }

    slice "View Discount" {
      view "DiscountView" {
        subscribes = [DiscountApplied]

        fields {
          orderId = required(string)
          code    = required(string)
        }
      }
    }

    slice "View Order History" {
      view "OrderHistoryView" {
        subscribes = [OrderPlaced]

        fields {
          orderId  = required(string)
          placedAt = required(timestamp)
        }
      }

      shows {
        screen = OrderHistoryPage
        reads  = OrderHistoryView
      }
    }
  }
}
```

This model states two facts that the current grammar cannot state:

- `CheckoutPage` is one page that appears at two points on the timeline. Therefore `bed` can see that it
  issues two commands, and the viewer can link the appearances.
- "View Order History" is a State View slice that ends at a screen with no command: the read side ends in a
  wireframe, as the cheat sheet draws it. Today the only way to write it is a trigger, taken from the command
  chain.

---

## Interaction With Other Work

### `user-stories/completed/lint-unread-view.md`

That feature is complete, and it gives this proposal part of what it needs. It meets this proposal at three
points:

- **US-002 made the `reads` of a trigger resolve** (`internal/validator/validator.go:400-401`). This proposal
  keeps the resolution and widens it to a list.
- **`view/never-read` must count each appearance.** US-001 builds its set of read views from
  `slice.Trigger.Reads` (`internal/linter/linter.go:68-70`). After this change, that term is a loop over each
  `reads` entry of each appearance. If the change drops the term and does not replace it, the rule reports
  each view that only a screen reads as a view that nothing reads.
- **The rule names a trigger.** Its message and its description say "trigger" (`linter.go:667`,
  `descriptions.go:53`). Its description calls a trigger "the closest the language has to a screen". After
  this change, the language has a screen, and the text names it.

### `user-stories/model-queries.md`

`emod references` and `emod trace` read the model through `internal/query`, and the other stories of that
feature use the same package. The kind `trigger` appears in each of their outputs: the `--kind` option of
`emod trace`, the JSON output, and the MCP tools. A story that ships before this proposal ships with
`trigger`, and this proposal renames it to `screen` in each surface. The "issued by" group of a command gets
a DSL word, `command` in a `shows` block, in place of the rule that a trigger issues each command of its
slice.

---

## Versioning

`ast.SupportedVersion` (`internal/ast/ast.go:4`) stays at 1. This is a breaking grammar change, and the
version header exists to absorb that kind of change. But the tool is still in development, and the models that
use the retired keyword are the examples, the test fixtures and the documentation that this proposal
rewrites anyway. A new version here buys a migration diagnostic for files that do not need one.

A file that still uses `trigger` therefore fails at the retired keyword, not with a version message. The error
for the retired keyword (see Parser) makes the failure clear: it names the replacement at the position of the
old keyword. A migration message can say no more than that.

The change is mechanical enough to do by hand:

1. For each `trigger "Some Name"`, declare `screen "SomeName"` at the top level, with the `description` of
   the trigger.
2. Replace the trigger with a `shows` block whose `screen` is `SomeName`, with the same `reads`.
3. Add a `command` entry for each command that the screen issues.
4. Delete each `actor` block, and each `actor =` line.

---

## Phased Implementation

### Phase 1: `reads` takes a list

This phase adds and renames nothing. `Trigger.Reads` becomes `[]string`, each entry resolves, and the
diagrams draw one arrow for each entry. A single bare name stays valid. `emod fmt` writes two or more entries
as a list. The ASCII diagram prints the `reads` arrow. The exports, `internal/query`, `internal/arrange`
(`arrange.go:133-134`) and `view/never-read` read the list.

`lint-unread-view` US-002 delivered the resolution, so this phase adds only the list. It stands alone, and it
ships whatever happens to phase 2.

### Phase 2: Identity

This is the breaking phase. The tool is in development, so the phase goes in directly, with no period of
deprecation. It contains:

- The top-level `screen` and the `shows` block, with `command` stated on each appearance.
- Boxes keyed by slice, in the SVG and the diagram JSON.
- `trigger` retired, with an error that names its replacement.
- `actor` deleted from end to end: the parser, the AST, the formatter, the cross-reference table, both
  exports, the diagram JSON, the importer, the glossary, the LSP and the viewer.
- The examples, `docs/dsl-reference.md` and its copy in `site/content/docs/`, the quick start in the README and
  in `site/content/docs/_index.md`, and the e2e suites, all changed together.

The retirement of `actor` makes the deletion in this phase about two times larger, but it does not change the
design. It goes with this phase for two reasons. The rename already opens each file that it touches. And to
keep it means to keep a construct whose only reference site this change removes.

### Phase 3: What identity makes possible

- `image` from end to end.
- `bed`, `screen/never-shown` and `screen/missing-image-asset`.
- Screens in the glossary.
- Go-to-definition and find-references across appearances.
- A highlight of each appearance of a screen in the viewer.

Phase 1 stands alone, and ships first. Phase 3 is the return on the change, and it is why phase 2 is worth its
cost: no part of phase 3 is possible while a screen has no name.

---

## Risks and Open Questions

**The arrows from a screen to a command must be written.** An explicit `command` removes three defects, but
it turns an inferred arrow into an arrow that the author writes. A migrated model that leaves out `command`
silently loses arrows that it drew before. The migration is mechanical, and the arrows come back when the
author adds the entry. But nothing fails while the entry is absent. A screen with no `command` is a valid
surface that only shows data, so the linter cannot tell the two cases apart. `screen/never-shown` finds a
screen that no slice shows. It does not find a screen that must issue a command and does not.

**The diagrams lose the actor.** The SVG and ASCII labels, the event flow caption and the actor bar in the
viewer show the actor now. After phase 2, they show the name of the screen only. A screen name can say who
uses the page (`GuestCheckoutPage`), and `description` holds the rest. If a model needs the person on the
diagram again, an attribute on the screen declaration can carry it, with a check that resolves it.

**`screen/missing-image-asset` needs the location of the file.** `Lint` takes only the model
(`internal/linter/linter.go:50`), and an `image` path is relative to the `.emod` file. The rule needs the
directory of the file, or the CLI checks the path before it calls the linter. Where no file system is present,
as in the WASM build, the rule does not run.

**Bed alone does not justify a grammar change.** This proposal does not offer it as the reason. It is the
clearest evidence that the absence of identity has a cost, because it is a canonical rule that the AST cannot
state. The case for the change rests on four facts together: the place for the mockup, the `reads` list, the
screen that a reader can follow across slices, and the diagrams that join two triggers of one name
incorrectly.

# US-007: One palette for element types

## Progress
- [x] Task 1: Draw a trigger as a screen in SVG and draw.io
- [x] Task 2: Draw a trigger as a screen in the viewer
- [x] Task 3: Give an automation and a translation their own fills in SVG and draw.io
- [x] Task 4: Paint the viewer's nodes from one palette table
- [x] Task 5: Pin SVG, draw.io and the viewer to one palette
- [x] Task 6: Document the palette in the DSL reference and pin it to the exporters

---

## Story Reference

`user-stories/triggers-and-automations.md` → **US-007: One palette for element types**
(seventh of eleven stories in "Triggers and Automations"; it declares no dependency and is decomposed
against `main` as it stands, with US-001, US-002 and US-003 landed). Design notes:
`docs/proposals/triggers-and-automations-proposal.md` — `:24` for the diagnosis, `:282-295` for the
settled palette table and the sentence that a trigger must stop being a plain rounded rect,
`:458-464` for the phase this story realises.

**In scope:** one fill and one stroke per element type, agreed by SVG, draw.io and the web viewer;
splitting the single `fillReactor` that currently paints both automations and translation reactors, so
six element types have six fills; repainting the viewer's trigger and automation, including the
`:hover` and `.hl` shades that `internal/viewer/static/viewer.html` declares per block class and that
would otherwise keep flashing the old assignment; a test that enumerates the palette across all three
renderers rather than spot-checking it; a screen framing that tells a trigger from a sticky note in all
three renderers without reference to colour; and a palette section in `docs/dsl-reference.md` held
equal to what the exporters emit by a test.

**Out of scope, owned by other stories:** the trigger kind slot, so `trigger UI "…"` still parses and
this story's models keep writing a kind (US-004); the `reads` edge from a view to a trigger or an
automation (US-005); lane placement — nothing moves here, no `y` coordinate changes, and the
"UI / Triggers" lane label stays as it is (US-006); the `automation/missing-todo-list` rule (US-008);
LSP (US-009); the VS Code TextMate grammar and `editors/tree-sitter-emod/queries/*.scm`, which is what
US-010's "highlighting" means — the viewer's own `.hl` node-selection shades are *not* that, and are in
scope here; `examples/`, `README.md`, and every part of `docs/dsl-reference.md` except the new palette
section (US-011). Out of scope per the feature's non-goals: wireframe assets — pointing a trigger at a
mockup image is deferred precisely until the box-versus-screen distinction this story adds proves
insufficient.

**The settled palette**, taken verbatim from the proposal (`:288-293`):

| Element | Fill | Stroke |
|---|---|---|
| trigger | `#ffffff` | `#333333` |
| command | `#dae8fc` | `#6c8ebf` |
| event | `#ffe6cc` | `#d79b00` |
| view | `#d5e8d4` | `#82b366` |
| automation | `#e1d5e7` | `#9673a6` |
| translation | `#f5f5f5` | `#666666` |

Against the tree as it stands, the edits this implies are narrower than the table suggests: command,
event and view already agree everywhere; SVG and draw.io already paint a trigger white and an
automation `#e1d5e7`; the viewer already paints a translation `#f5f5f5`. What actually moves is the
translation reactor in SVG and draw.io (off `fillReactor`, which it shares with the automation), and
the trigger and automation in the viewer (which swap).

**Consequences of that boundary, decided.** Eight things the story does not spell out:

1. *One palette means one table the three renderers are pinned to, not one file all three read.*
   `internal/viewer/generated/` is gitignored (`.gitignore:10`) and `/web` is a build product of
   `task build:web`, so a palette generated into either is invisible to `task test:viewer`, whose
   vitest suite imports modules straight out of `internal/viewer/static/`. Making the Go renderers
   parse a JS file at package init would let a text edit break `emod diagram` at run time —
   `internal/cue/embed.go` embeds a file the CUE runtime is built to parse, which is not the same
   thing. So the palette is declared twice, once per language, and Task 5 is the test that forbids
   them from disagreeing. Its expected value comes from the *other* language's file, which is what
   stops it being an assertion that cannot fail.
2. *The pin is against emitted output, not against the Go constants.* Every test file in
   `internal/diagram` is `package diagram_test`, so reading the constants would mean exporting them
   for a test's benefit. The story's criteria are about what the exporters emit anyway, so Task 5
   renders a model and reads each element type's fill and stroke back out of the SVG and the draw.io
   XML through the `fillOfLabel` readers that already exist.
3. *"No two element types share a fill" is checked over the six the story names, and over colour
   families as well as hex.* `colorFamily` (`internal/diagram/contract_test.go:540`) already
   classifies a hex by hue; the six settled fills land in six distinct families — white, blue, orange,
   green, purple, grey — so requiring six families as well as six hexes rules out a future pair of
   near-identical purples that would pass a hex comparison and fail a reader.
4. *The external-system box is not an element type and keeps `fillExternal`.* It is a property of a
   translation (`translation { external_system "…" }`), not a declared element, and the story's list
   names six. The consequence of the proposal's table is that a translation reactor and its
   external-system box become the same grey in SVG and draw.io; they stay apart by the external box's
   dashed outline, which `"draws an external system with a dashed outline"`
   (`internal/diagram/contract_test.go:509`) already pins, and by the reactor's gear marking.
5. *A white trigger beside a `#f5f5f5` translation is exactly why the framing criterion exists.* The
   two are one hex apart from confusable, and the trigger's fill is also the fill of the SVG lane
   (`internal/diagram/svg.go:41-47`) and of the viewer's slice box (`renderer.js:214`) it sits on. So
   the framing tasks come *first*, ahead of any repaint: after Task 1 and Task 2 there is no commit in
   which a trigger is a plain white rectangle on a white background.
6. *Edge colours are not touched, and one of them is left knowingly wrong.* The story's criteria name
   element types' fills and strokes; `edgeConfig` (`internal/viewer/static/config.js:19-27`) and the
   `stroke*` constants (`internal/diagram/drawio.go:80-82`) colour arrows, not elements. The
   consequence to record rather than fix here: `trigger_command` draws its arrow in `#9673a6`, which
   after this story is the automation's stroke. Nothing in the six criteria reaches it and no story
   owns it; it is named in the Summary as a follow-up rather than folded in silently.
7. *The screen framing is a rendering change in three outputs and is decomposed as two tasks, not
   three.* SVG and draw.io are one Go package and one test run; the viewer is a separate language and
   a separate harness (`task test:viewer` is not part of `task test:unit`). This is the split US-003's
   clock badge used — its Task 8 for the two Go renderers, its Task 10 for the viewer — and the same
   reason applies.
8. *The reference gains a section at the end, numbered 13.* `docs/dsl-reference.md` documents no
   colours today, so this is an addition rather than a correction, and appending after `## 12.
   Pipeline` (`:646`) renumbers no heading and invalidates none of the number-prefixed in-document
   links. Inserting it anywhere else would.

**Learnings folded in** from `tasks/learnings.md`: additive output changes owe a byte-identical
receipt for models that do not use the feature, and the diagram packages carry it with
`withoutDescriptions` and the geometry helpers in `contract_test.go`; a differential receipt must
first prove the twin actually differs; an assertion whose expected value comes from the code under
test is the recurring review finding — which is the whole design of Task 5 and Task 6; de-duplicate
before a fan-out edit and land the de-duplication with proof, and give an extracted helper the
parameters its siblings take; name an extracted helper after the contract its callers rely on;
`internal/viewer/static` is a display surface with its own vitest harness, and restructuring
`showDetailPanel`-scale code beyond the change belongs in its own commit; the viewer shows a node
twice — the canvas box and the detail panel — and assertions on a drawn label must walk the group's
`<text>` elements because `group.textContent` folds the `<title>` in; `docs/dsl-reference.md` anchors
embed the section number, and the reference is the one surface no test reaches — Task 6 gives it its
first, for the palette alone; run repo tooling through `mise exec --`; acceptance criteria describe the
working tree, and a commit-message receipt is the commit author's obligation, never a criterion.

---

## Codebase Context

**The Go palette is one block of constants, shared by both Go renderers.**
`internal/diagram/drawio.go:66-83` declares fifteen colours; `internal/diagram/svg.go` reads the same
identifiers because both files are `package diagram`. Six of them name element types today, and one
name covers two types:

| Constant | Value | Drawn for |
|---|---|---|
| `fillTrigger`/`strokeTrigger` | `#ffffff` / `#333333` | trigger (`svg.go:81-82`, `styleTrigger` at `drawio.go:91`, cell at `:314`) |
| `fillCommand`/`strokeCommand` | `#dae8fc` / `#6c8ebf` | command (`svg.go:94-95`) |
| `fillEvent`/`strokeEvent` | `#ffe6cc` / `#d79b00` | event, and a translation's nested event (`svg.go:129-130`, `:140-141`) |
| `fillView`/`strokeView` | `#d5e8d4` / `#82b366` | view (`svg.go:106-107`) |
| `fillReactor`/`strokeReactor` | `#e1d5e7` / `#9673a6` | **automation** (`svg.go:158-159`) **and translation reactor** (`svg.go:175-176`), both via `styleReactor` in draw.io (`:414`, `:429`) |
| `fillExternal`/`strokeExternal` | `#f5f5f5` / `#666666` | context label band and the external-system box (`svg.go:51`, `:192`) |

`styleReactor` (`drawio.go:95`) is the mxGraph spelling of the same collision. `boxBase`
(`drawio.go:87`) is what every element style is built on, and `styleExternalSystem` (`:96`) is the one
style that adds a token of its own (`dashed=1;`) — the precedent for expressing a shape variant as an
extra token rather than a new base.

**The SVG rect builders are already de-duplicated.** `svgRectAttributes` (`svg.go:346`) and
`svgRectElement` (`:355`) sit under `svgRect` (`:334`), `svgRoundedRect` (`:338`) and
`svgDashedRoundedRect` (`:342`), the last of which is the precedent for a variant expressed as one
extra attribute. The comment at `:351-354` is load-bearing: a description renders as a browser tooltip
only when its `<title>` is nested inside the shape, and a shape with nothing to say stays
self-closing.

**Mermaid and ASCII have no colours.** `internal/diagram/mermaid.go` emits no `classDef` and no hex,
and `exporters()` (`contract_test.go:56`) leaves `fillOfLabel` nil for both text formats, so the
palette subtests skip them. `README.md` documents no colours either.

**The palette already has a contract test, and it spot-checks.** `TestExporterPalette`
(`internal/diagram/contract_test.go:462-515`) runs per exporter with a non-nil `fillOfLabel` and holds
three leaves: `"follows the sticky-note colour convention"` (`:472`), which asserts orange / blue /
green / white / grey for `Evt`, `Cmd`, `Rmo`, `Form` and `Stripe` against both the described and the
`withoutDescriptions` twin; `"gives each element type a distinguishable fill"` (`:496`), which puts
four fills — `Evt`, `Cmd`, `Rmo`, `Stripe` — through `unique` (`:615`); and `"draws an external system
with a dashed outline"` (`:509`). `paletteModel` (`:521`) holds a trigger, a command, an event, a view
and a translation with an external system, each with its own label — and **no automation**, which is
why the collision between `fillReactor`'s two meanings has never been visible to a test.
`singleSliceModel` (`:645`) already accepts an `*ast.Automation`. `withoutDescriptions` (`:797`) walks
automations (`:828-830`), so extending `paletteModel` is covered by the undescribed twin without
editing the stripper. `colorFamily` (`:540`) classifies by hue over `hsv` (`:562`) and returns
`unclassified(hue …)` for anything outside its bands.

**The per-format readers.** `drawioFillOfLabel` (`contract_test.go:94`) finds the shape whose label
contains the name and pulls `fillColor=` out of its style; `svgFillOfLabel` (`:111`) walks back from
the `<text>` carrying the label to the nearest preceding `<rect>`. There is no `strokeOfLabel`. The
shape walkers behind them are `svgShapes` (`svg_test.go:349`), which attaches a `<text>`'s content to
the **last rect it has seen**, and `drawioShapes` (`drawio_test.go:765`). `svgBoxes` (`svg_test.go:396`)
and `drawioBoxes` (`drawio_test.go:834`) turn those into `diagramBox{label, appearance}`, the geometry
and paint receipt `appearancesOf` (`contract_test.go:443`) compares.

**The viewer declares its palette three times.** `internal/viewer/static/renderer.js:253-261` is a
`switch` assigning a fill, a stroke and a CSS class per node type; `internal/viewer/static/viewer.html`
declares a `:hover` fill per block class at `:539-556` and an identical set of `.hl` fills at
`:876-891`, each a darkened shade of that type's base fill. Its assignment for trigger (`#e1d5e7`) and
automation (`#fff2cc`) is the disagreement the story is about, and the two CSS blocks carry it too —
`.trg-block:hover` is `#d0bdd8`, a darker version of the purple that after this story belongs to
automations. `config.js` holds `L`, `edgeConfig` (`:19-27`) and `arrowClassMap`; `renderer.js` already
imports from it, so it is where a node palette belongs beside the edge one. The node loop
(`renderer.js:246-296`) then builds the group, one `rect`, two port circles and calls
`appendBlockLabels` (`:117-144`), which special-cases a translation and a scheduled automation and
otherwise centres a single label.

**The viewer's test harness.** `internal/viewer/tests/*.test.js`, vitest on jsdom
(`internal/viewer/vitest.config.js`), run by `task test:viewer` with `dir: internal/viewer` — it
`npm ci`s and is not part of `task test:unit`. `renderer.test.js` holds `automationFill = '#fff2cc'`
(`:46`), the `drawnBoxes(svg)` helper (`:112-124`) that reads `group.querySelector('rect')` — the
group's **first** rect — for each `.diagram-node`, `tooltipOf` (`:103`) reading the group's first
`<title>`, and the clock-badge leaf at `:219-234` that asserts the automation's fill and compares whole
`drawnBoxes` maps. `pairedAutomations` (`:51`) is the fixture shape for a render carrying a marked and
an unmarked node side by side. jsdom implements no SVG geometry, so a module touching it needs
`installSVGGeometry()` from `./svg-env.js` and the dynamic `await import` spelling.

**Cross-directory reads from tests are established.** `internal/export/export_test.go:4611` reads
`../cue/schema.cue`; `internal/importer/importer_test.go:77` and `:90` read `../parser/testdata/*.emod`;
`internal/formatter/formatter_test.go:3648` does the same. A test in `internal/diagram` reading
`../viewer/static/config.js` or `../../docs/dsl-reference.md` is that shape.

**The reference has no palette.** `docs/dsl-reference.md` has twelve numbered sections, the last being
`## 12. Pipeline` (`:646`), and the file ends there. Nine in-document links cite a number-prefixed
slug (`:86`, `:100`, `:272`, `:305`, `:320`, `:479`, `:564`, `:612`, `:631`, `:640`). No CI step checks
markdown links.

**Not touched, deliberately.** `internal/lexer`, `internal/parser`, `internal/ast`,
`internal/formatter`, `internal/validator`, `internal/linter`, `internal/export`, `internal/importer`,
`internal/cue`, `internal/glossary`, `internal/lsp`, `internal/test`, `editors/`, `examples/`,
`README.md`, `e2e/` and `e2e-viewer/` — none of them names an element colour, and the two e2e suites
assert none. No `.emod` file, fixture, golden or wire key moves in this story.

---

## Tasks

### Task 1: Draw a trigger as a screen in SVG and draw.io

**Behavior:** in SVG and draw.io a trigger's box carries a screen framing that no other element type's
box carries, so a reader tells a human entry point from a sticky note without reading its colour. The
trigger keeps the fill, stroke, position and size it has today, and every other element type renders
exactly as it does now.

**Acceptance Criteria:**
- [ ] Rendering `paletteModel()` to SVG draws framing on the trigger's box that the command, event,
      view and translation boxes and the external-system box do not carry — asserted by comparing the
      trigger's drawn markup against each of the others in the same render, not by searching the whole
      output for a substring
- [ ] Rendering the same model to draw.io gives the trigger's cell a shape or framing token that none
      of the other element types' styles carries, and the output still satisfies the existing
      well-formed-XML leaf (`requireWellFormed`, `internal/diagram/contract_test.go:56`)
- [ ] The distinction does not rest on colour: with every fill value in the output normalised to a
      single colour, the trigger is still distinguishable from each of the other element types in both
      formats
- [ ] The trigger's box occupies the same `x`, `y`, `width` and `height` it does without the framing,
      and every framing element the trigger draws lies within those four edges — so nothing the framing
      adds can push a neighbouring box or move an arrow endpoint
- [ ] A described trigger still renders its description as a tooltip in both formats, and an
      undescribed one still renders none — `svgTooltipOf` (`internal/diagram/svg_test.go:415`) reports
      exactly one shape for the trigger's label, so the framing must not become a second shape carrying
      that label
- [ ] `svgFillOfLabel` and `drawioFillOfLabel` still return the trigger's own `#ffffff` for the label
      `Form`: `svgShapes` (`internal/diagram/svg_test.go:349`) attaches a `<text>` to the **last rect it
      has seen**, so a framing rect emitted between the trigger's box and its label would silently
      redirect every fill assertion about a trigger
- [ ] The three existing leaves of `TestExporterPalette` (`internal/diagram/contract_test.go:472`,
      `:496`, `:509`) pass unedited
- [ ] `git diff` edits no expected constant in `internal/diagram/svg_test.go`,
      `internal/diagram/drawio_test.go` or `internal/diagram/contract_test.go` that describes a command,
      event, view, automation, translation, external system, lane, context band or arrow — the framing
      is additive to the trigger alone

**Affected Files/Modules:**
- `internal/diagram/svg.go` — the trigger box and label (`:81-82`), and the rect builders (`:334-361`)
  if the framing is expressed as a variant of them
- `internal/diagram/drawio.go` — `styleTrigger` (`:91`) and the trigger cell (`:314`)
- `internal/diagram/svg_test.go`, `internal/diagram/drawio_test.go`,
  `internal/diagram/contract_test.go`

**Patterns to Follow:**
- `svgDashedRoundedRect` (`internal/diagram/svg.go:342`) is the precedent for a shape variant built as
  one extra attribute on the shared `svgRectAttributes`/`svgRectElement` pair (`:346`, `:355`), and
  `styleExternalSystem` (`internal/diagram/drawio.go:96`) is its draw.io counterpart — one extra
  mxGraph token on `boxBase` (`:87`)
- The comment at `internal/diagram/svg.go:351-354` states why a shape carrying a description opens
  rather than self-closes; read it before adding a second element to a trigger
- `internal/viewer/static/renderer.js:214-221` draws a slice as a box plus a header band plus a
  centred label — the repo's existing "titled container" drawing, for reference on what a framing
  reads as
- `tasks/learnings.md` "De-duplicate before a fan-out edit, and land the de-duplication with proof" and
  "Name an extracted helper after the contract its callers rely on" — if the framing becomes a shared
  builder, give it the parameters its siblings take
- `tasks/learnings.md` "Additive output changes owe a byte-identical receipt for models that do not use
  the feature" — the receipt here is every non-trigger element of `paletteModel()`
- Lane placement and the lane label are US-006's; the palette is Task 3's. This task moves no box and
  repaints nothing

**Testable:** Yes — through `diagram.ExportSVG` and `diagram.ExportDrawio`.

**Verification:** `go test -tags unit ./internal/diagram/...`; `go build ./...`.

**Depends on:** None

---

### Task 2: Draw a trigger as a screen in the viewer

**Behavior:** the viewer draws a trigger node with the same screen framing the two Go renderers draw,
and no other node type gains it. The trigger keeps the fill, stroke, box geometry, ports and label it
has today, so dragging, connecting and selecting a trigger are unaffected.

**Acceptance Criteria:**
- [ ] A render containing a trigger, a command, an event, a view, an automation and a translation draws
      framing within the trigger's group that none of the other five groups contains — asserted on one
      render so the framed and unframed cases are proved together
- [ ] Every framing element the trigger draws lies within its own box's edges, and the trigger's entry
      in `drawnBoxes(svg)` (`internal/viewer/tests/renderer.test.js:112`) reports the same `x`, `y`,
      `width` and `height` as it does without the framing, with the same fill
- [ ] `drawnBoxes` still reports the trigger's own box: it reads `group.querySelector('rect')`, the
      group's **first** rect, so a framing rect prepended to the group would silently redirect that
      helper for every existing assertion — the leaf that proves it is a `drawnBoxes` comparison over a
      render containing a trigger
- [ ] The trigger's group keeps its `trg-block diagram-node` classes, its `data-node-id`, and its two
      `node-port` circles, and the existing `interaction`, `connect` and `visibility` suites pass
      unedited
- [ ] The trigger's label is still the group's only drawn `<text>` — asserted by walking the group's
      `<text>` elements as `drawnLabels`/`drawnText` do, not through `group.textContent`, which folds a
      `<title>` in
- [ ] Adding a `<title>` for the framing does not displace an existing tooltip: `tooltipOf` reads a
      group's first `<title>`, which is where `appendBlockLabels`
      (`internal/viewer/static/renderer.js:134-141`) puts a scheduled automation's cadence — a trigger's
      framing must leave that leaf (`internal/viewer/tests/renderer.test.js:219-234`) passing unedited
- [ ] `mise exec -- task test:viewer` passes, and `git status --porcelain -- '*.go'` is empty — this
      task changes no Go file

**Affected Files/Modules:**
- `internal/viewer/static/renderer.js` — the node loop (`:246-296`) and `appendBlockLabels` (`:117-144`)
- `internal/viewer/static/viewer.html` — only if the framing needs a CSS rule of its own
- `internal/viewer/tests/renderer.test.js`

**Patterns to Follow:**
- The slice box, its header band and its centred header text (`internal/viewer/static/renderer.js:214-221`)
  are the viewer's existing titled-container drawing
- The translation branch of `appendBlockLabels` (`:121-132`) is the only node that draws more than a
  centred label, and is the shape for a node type that needs extra geometry
- `tasks/learnings.md` "The viewer shows a node twice — the canvas box and the detail panel" — the
  detail panel needs nothing here, but its warning that a `<title>` is read as the group's first child
  is what the tooltip criterion above guards
- `tasks/learnings.md` "`internal/viewer/static` is a display surface with its own vitest harness" — the
  jsdom SVG shim (`installSVGGeometry` from `internal/viewer/tests/svg-env.js`) and the dynamic-import
  spelling `internal/viewer/tests/renderer.test.js` already uses are required for a module touching
  geometry, and restructuring the node loop beyond the trigger branch belongs in its own commit
- The Go framing landed in Task 1; matching its visual idea keeps the three outputs teaching one thing,
  but the two are separate code paths and neither shares a helper with the other

**Testable:** Yes — through the vitest harness under `internal/viewer/tests`.

**Verification:** `mise exec -- task test:viewer`; `git status --porcelain -- '*.go'` is empty.

**Depends on:** None

---

### Task 3: Give an automation and a translation their own fills in SVG and draw.io

**Behavior:** SVG and draw.io paint an automation and a translation reactor differently, so the six
element types the language declares have six fills and six strokes. The assignment is the settled
palette, and the test that guards it enumerates all six rather than sampling four.

**Acceptance Criteria:**
- [ ] `paletteModel()` (`internal/diagram/contract_test.go:521`) declares an automation with a label of
      its own alongside the trigger, command, event, view and translation it already declares
- [ ] `"gives each element type a distinguishable fill"` (`:496`) reads the fill of all six element
      types — trigger, command, event, view, automation and translation reactor — and requires six
      distinct values in both SVG and draw.io, so the criterion is an enumeration and not a spot-check
- [ ] The same six fills fall in six distinct families under `colorFamily`
      (`internal/diagram/contract_test.go:540`), so two near-identical shades of one hue cannot satisfy
      the enumeration; if a settled fill classifies as `unclassified(hue …)`, the classifier gains the
      band rather than the palette gaining a colour it does not want
- [ ] An automation's fill and a translation reactor's fill differ in both formats — named as its own
      leaf, since one constant painting both is the defect this task exists to fix
- [ ] A `strokeOfLabel` reader sits beside `fillOfLabel` (`internal/diagram/contract_test.go:94`,
      `:111`) for both SVG and draw.io, and the six element types' strokes are likewise six distinct
      values
- [ ] `"follows the sticky-note colour convention"` (`:472`) covers the automation's family alongside
      the five it already names, and still runs over both the described model and the
      `withoutDescriptions` twin
- [ ] The external-system box is not one of the six: it keeps `fillExternal`, `"draws an external system
      with a dashed outline"` (`:509`) passes unedited, and a leaf asserts the external box is drawn
      dashed while the translation reactor beside it is not — the two share a grey and are told apart by
      the outline
- [ ] The constant that painted both element types is replaced by one named for each concept, and no
      element style in `internal/diagram/drawio.go:85-97` is used for two element types
- [ ] `git diff` edits no expected constant in `internal/diagram/svg_test.go`,
      `internal/diagram/drawio_test.go` or `internal/diagram/contract_test.go` that describes a command,
      event, view or trigger, and moves no box's position or size: only the translation reactor's paint
      changes

**Affected Files/Modules:**
- `internal/diagram/drawio.go` — the colour constants (`:66-83`), the element styles (`:85-97`), the
  automation cell (`:404-416`) and the translation reactor cell (`:418-431`)
- `internal/diagram/svg.go` — the automation box (`:149-164`) and the translation reactor box
  (`:166-181`)
- `internal/diagram/contract_test.go` — `paletteModel` (`:521`), the three palette leaves (`:472`,
  `:496`, `:509`), `colorFamily` (`:540`), and the new stroke reader beside `fillOfLabel`
- `internal/diagram/svg_test.go`, `internal/diagram/drawio_test.go`

**Patterns to Follow:**
- The constants block (`internal/diagram/drawio.go:66-83`) is the one place both Go renderers read;
  `styleExternalSystem` (`:96`) shows how a style differs from its siblings by one token
- `tasks/learnings.md` "Name an extracted helper after the contract its callers rely on" — a constant
  called `fillReactor` describes the shape it happens to draw, not the element type its callers rely on
  it identifying, which is how one name came to cover two types. `reactorLabel`
  (`internal/diagram/labels.go:10`) keeps its name: the gear marking really is shared, and US-006 keeps
  it on both
- `singleSliceModel` (`internal/diagram/contract_test.go:645`) already routes an `*ast.Automation` onto
  the slice, and `withoutDescriptions` (`:797`) already strips an automation's description (`:828-830`),
  so the undescribed twin covers the new element without editing the stripper
- `tasks/learnings.md` "Additive output changes owe a byte-identical receipt for models that do not use
  the feature" — the receipt is the four element types whose paint does not change
- `TestExporterPalette`'s own doc comment (`:462-464`) states that it pins the convention and not the
  values; keep that split — the exact hexes are pinned against the viewer in Task 5 and against the
  reference in Task 6, not here
- The lane an automation or a translation reactor is drawn in is US-006's; this task changes no
  coordinate

**Testable:** Yes — through `diagram.ExportSVG` and `diagram.ExportDrawio`.

**Verification:** `go test -tags unit ./internal/diagram/...`.

**Depends on:** None

---

### Task 4: Paint the viewer's nodes from one palette table

**Behavior:** the viewer paints each node type the fill and stroke the settled palette names — a
trigger white with a dark outline, an automation purple — reading them from a single table rather than
from a `switch`, and its hover and highlight shades follow each type's own colour instead of the one it
used to have.

**Acceptance Criteria:**
- [ ] `internal/viewer/static/config.js` gains a table naming a fill and a stroke for each of the six
      node types, beside `edgeConfig` (`:19-27`), and the `switch` at
      `internal/viewer/static/renderer.js:253-261` paints from it
- [ ] No element-type fill or stroke literal remains anywhere in `internal/viewer/static` outside that
      table and the derived hover and highlight shades — `renderer.js` names none
- [ ] A render containing all six node types paints a trigger white with the dark outline, an
      automation purple and a translation grey, read back through `drawnBoxes(svg)`
      (`internal/viewer/tests/renderer.test.js:112`), and the six fills are pairwise distinct
- [ ] `automationFill` (`internal/viewer/tests/renderer.test.js:46`) names the settled automation fill
      and the clock-badge leaf (`:219-234`) passes with it, so the badge is still proved to sit inside a
      box whose paint the cadence does not change
- [ ] Each `:hover` fill `internal/viewer/static/viewer.html` declares (`:539-556`) and each `.hl` fill
      it declares (`:876-891`) is a darker shade of that same node type's fill in the table — asserted
      by a test that reads `viewer.html`, extracts the per-class fills and compares each against the
      table entry for the same type, so a trigger can no longer hover into the automation's purple
- [ ] The six `:hover` fills are pairwise distinct, and so are the six `.hl` fills
- [ ] That test fails when one `:hover` or `.hl` declaration is repointed at another node type's hue,
      and the failure names the node type — an assertion whose expected value comes from the file under
      test would not
- [ ] `mise exec -- task test:viewer` passes, and `git status --porcelain -- '*.go'` is empty — this
      task changes no Go file

**Affected Files/Modules:**
- `internal/viewer/static/config.js` — the node palette table, beside `edgeConfig` (`:19-27`)
- `internal/viewer/static/renderer.js` — the node loop's `switch` (`:253-261`)
- `internal/viewer/static/viewer.html` — the `:hover` fills (`:539-556`) and the `.hl` fills (`:876-891`)
- `internal/viewer/tests/renderer.test.js`, and a leaf for the stylesheet

**Patterns to Follow:**
- `edgeConfig` (`internal/viewer/static/config.js:19-27`) is the existing per-type appearance table and
  the shape to match; `renderer.js` already imports from `config.js`
- The vitest run has `dir: internal/viewer` (`Taskfile.yml`, `test:viewer`) and node builtins are
  available under jsdom, so a leaf may read `viewer.html` from disk; `internal/viewer/tests/svg-env.js`
  is the precedent for a helper module sitting beside the tests
- `colorFamily`/`hsv` (`internal/diagram/contract_test.go:540`, `:562`) is the classification the Go
  side uses to say two colours are the same hue; the JS leaf wants the same idea rather than a second
  vocabulary
- `tasks/learnings.md` "An assertion whose expected value comes from the code under test is the
  recurring review finding" — name an edit to `viewer.html` that makes the stylesheet leaf fail before
  writing it
- `tasks/learnings.md` "`internal/viewer/static` is a display surface with its own vitest harness" — the
  node loop is otherwise untouched; restructuring it beyond replacing the `switch` belongs in its own
  commit
- The arrow colours in `edgeConfig` and `arrowClassMap` are not element-type colours and are not
  touched here

**Testable:** Yes — through the vitest harness under `internal/viewer/tests`.

**Verification:** `mise exec -- task test:viewer`; `git status --porcelain -- '*.go'` is empty.

**Depends on:** 2, 3

---

### Task 5: Pin SVG, draw.io and the viewer to one palette

**Behavior:** a test fails if any one of the three renderers paints an element type differently from
the other two, so the palette cannot drift apart again the way it did. The comparison is made against
what the exporters emit and against the viewer's own file, not against a value restated in the test.

**Acceptance Criteria:**
- [ ] A test reads the viewer's node palette table out of `internal/viewer/static/config.js` and, for
      each of the six element types, requires the fill and the stroke SVG emits, the fill and the stroke
      draw.io emits, and the pair the table names to be equal — twelve values per format, asserted per
      element type rather than as a single opaque comparison, so a failure names which element type
      disagrees on which of fill or stroke
- [ ] The test fails if one value is changed on any one of the three surfaces, and the message names the
      element type — verified by naming the edit that breaks it
- [ ] Reading the table fails loudly when it is absent, renamed, or lists fewer than six element types,
      rather than comparing an empty set and passing
- [ ] The six fills the table names are pairwise distinct and fall in six distinct families under
      `colorFamily`, and so do the six strokes
- [ ] The test lives in `internal/diagram` and runs under `go test -tags unit
      ./internal/diagram/...`, so `task test:unit` — which CI runs — covers it; it does not depend on
      `task test:viewer`, which is a separate target
- [ ] `git status --porcelain -- internal/viewer` is empty: this task reads the viewer's table and
      changes nothing under it

**Affected Files/Modules:**
- `internal/diagram/contract_test.go` — a leaf beside `TestExporterPalette` (`:462-515`), using
  `fillOfLabel`, the stroke reader Task 3 adds, and `paletteModel` (`:521`)
- `internal/viewer/static/config.js` — read only

**Patterns to Follow:**
- `internal/export/export_test.go:4611` reads `../cue/schema.cue` from a sibling package's directory,
  and `internal/importer/importer_test.go:77` reads `../parser/testdata/all_patterns.emod`; a Go test
  reading `../viewer/static/config.js` is that shape
- `exporters()` (`internal/diagram/contract_test.go:56`) is the table to loop over, and its `fillOfLabel`
  is nil for Mermaid and ASCII, which have no colours — the existing skip at `:467` is why the loop
  covers exactly the two formats that paint
- `tasks/learnings.md` "An assertion whose expected value comes from the code under test is the
  recurring review finding" — this test's expected value comes from a different language's file on
  disk, which is the entire reason it has teeth; deriving it from the Go constants instead would leave
  the viewer unguarded and would need those constants exported for a test's benefit
- `tasks/learnings.md` "The two export guards cannot see a list neither writer emits" — the same trap
  applies here: a comparison over an element type the test forgot to name agrees trivially, which is
  what the fails-loudly-on-a-short-table criterion above guards
- The story's criterion is about what the exporters emit; read the values back out of rendered output
  through `fillOfLabel`, not out of `internal/diagram/drawio.go`

**Testable:** Yes — through `diagram.ExportSVG`, `diagram.ExportDrawio` and the viewer's table on disk.

**Verification:** `go test -tags unit ./internal/diagram/...`.

**Depends on:** 3, 4

---

### Task 6: Document the palette in the DSL reference and pin it to the exporters

**Behavior:** `docs/dsl-reference.md` states the fill and stroke each element type is drawn with and
that a trigger is drawn as a screen rather than as a sticky note, and a test fails if the documented
table and what the exporters emit ever disagree.

**Acceptance Criteria:**
- [ ] `docs/dsl-reference.md` gains a section listing all six element types — trigger, command, event,
      view, automation, translation — each with the fill and the stroke it is drawn with
- [ ] The section is appended after `## 12. Pipeline` (`:646`) as `## 13. …`, so no existing heading is
      renumbered — verified by listing `^## [0-9]+\.` and `\(#[0-9]+-` across the file and reconciling
      the two, which leaves all nine number-prefixed in-document links resolving to the sections they
      name
- [ ] The section states that a trigger is drawn as a screen and that the distinction from the other
      element types does not rest on colour
- [ ] A test parses the table out of `docs/dsl-reference.md` and requires each documented fill and
      stroke to equal what SVG and draw.io emit for that element type, failing when the document names
      an element type the exporters do not draw, omits one they do, or states a value they do not emit
- [ ] That test fails when one documented value is edited, and the failure names the element type
- [ ] `git diff docs/dsl-reference.md` shows an added section and no other change: the retired
      automation `trigger` spelling the reference still documents, the `examples/` and `README.md`
      updates, and every other correction to the reference are US-011's

**Affected Files/Modules:**
- `docs/dsl-reference.md` — a new section after `## 12. Pipeline` (`:646`)
- `internal/diagram/contract_test.go` — a leaf beside the one Task 5 adds

**Patterns to Follow:**
- `tasks/learnings.md` "`docs/dsl-reference.md` anchors embed the section number" — appending is what
  keeps every anchor valid; inserting anywhere above section 12 renumbers the headings below it and
  silently breaks the links that cite them
- `tasks/learnings.md` "`docs/dsl-reference.md` is the one keyword surface no test reaches, and a
  retirement story forgets it" — this task gives the reference its first test, for the palette alone;
  it does not attempt to cover the rest of the document
- `internal/export/export_test.go:4611` is the cross-directory read; from `internal/diagram` the
  reference is `../../docs/dsl-reference.md`
- `~/.claude/rules/markdown-docs.md` — the section must read as if it were always there: no "now",
  "updated", or reference to what the palette used to be
- The reader that turns rendered output into a fill or stroke per element type is the one Task 5
  already uses; the documented table is compared against the same values, so the reference, the two Go
  renderers and the viewer are one chain of equalities

**Testable:** Yes — through `diagram.ExportSVG`, `diagram.ExportDrawio` and the reference on disk.

**Verification:** `go test -tags unit ./internal/diagram/...`.

**Depends on:** 1, 2, 5

---

## Summary

**Six tasks**, ordered so that no commit leaves an element type harder to read than it is today.

The two framing tasks come first and depend on nothing. A trigger is already white in SVG and draw.io,
so the screen framing is independent of every repaint — and landing it first means the viewer's trigger
is never a plain white rectangle on the white slice box it sits on, which is what Task 4 would
otherwise produce for the three commits before Task 5. Task 3 splits the one Go constant that paints
two element types and turns the existing four-fill spot-check into a six-fill enumeration; Task 4 does
the viewer's half, including the two stylesheet blocks that carry the old assignment into hover and
selection. Task 5 is the guard the story is really asking for — the three renderers pinned to one
palette by a test whose expected values come from a different language's file. Task 6 gives the
reference its first test, and appends rather than inserts so no anchor moves.

Tasks 1, 2 and 3 are mutually independent and could land in any order; 4 follows 2 (both edit
`renderer.js`) and 3 (whose assignment it adopts), 5 follows 3 and 4, and 6 follows 5 plus the two
framing tasks whose result it describes.

**Story criteria coverage:**

| Story criterion | Task |
|---|---|
| Each of the six element types has one fill and stroke shared by SVG, draw.io and the viewer | 3 (Go), 4 (viewer), 5 (pinned equal) |
| No two element types share a fill | 3 (enumerated over all six, by hex and by colour family) |
| A trigger is distinguishable by shape or framing, not by colour alone | 1 (SVG, draw.io), 2 (viewer) |
| The palette documented in the DSL reference matches what the exporters emit | 6 |

**Carried along, not stated by the story:** the `:hover` and `.hl` fills in
`internal/viewer/static/viewer.html`, which restate the viewer's palette twice more and would keep
flashing the old assignment on hover and on selection (4); and the `strokeOfLabel` reader the contract
tests lack, without which "one fill *and stroke*" cannot be asserted at all (3).

**Left knowingly wrong, owned by nobody.** `trigger_command` draws its arrow in `#9673a6`
(`internal/viewer/static/config.js:24`), which after this story is the automation's stroke — a second
"one swatch, two meanings", in the edge vocabulary rather than the element one. None of the story's six
criteria reaches it, so it is out of scope here and worth a story of its own alongside the other edge
colours (`strokePurpleUp`, `strokeGreenUp` and the red `automation_trigger`).

**Deferred to other stories in the feature:** the trigger kind slot (US-004), the `reads` edge
(US-005), lane placement and the "Wireframes" label (US-006), the `automation/missing-todo-list` rule
(US-008), LSP (US-009), editor syntax highlighting (US-010), and the examples plus the rest of the
reference (US-011). Wireframe assets stay a non-goal of the feature: this story's framing is the
cheaper alternative being tried first.

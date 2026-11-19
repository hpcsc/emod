# US-006: Highlight each appearance of a screen in the viewer

## Contents
1. Task 1: Highlight each appearance of a screen while one of them is selected
2. Task 2: List the slices where the screen appears in the detail panel of an appearance, and select an appearance from the list

## Story Reference

US-006 in `user-stories/screens-as-first-class.md`. The design is in `docs/proposals/screens-as-first-class-proposal.md`, section "Web viewer" ("For an appearance, it shows ... the other appearances of the screen") and section "Diagrams" ("The viewer can highlight each appearance of a screen when the user selects one, and the detail panel can list them").

**Description:** As a member of a workshop, I want to select one appearance of a screen and see the others so that the group sees where the page recurs on the timeline.

**Acceptance Criteria (story):**
1. When the user selects an appearance, the viewer highlights each other appearance of the same screen.
2. The detail panel of an appearance lists each slice where the screen appears. A click on a slice in the list selects the appearance in that slice.
3. For a screen that appears in one slice only, the viewer highlights no other box, and the detail panel lists one slice.
4. When the user clears the selection, the highlight goes.
5. The desktop app behaves in the same way.

## Boundaries

**Out of scope:**

The feature's non-goals, as `user-stories/screens-as-first-class.md` states them:

- Layout, style, or the structure of components on a screen. A screen names its mockup by its path only.
- Navigation between screens. A screen graph is a different model.
- A mockup for each appearance. A second mockup means a second screen.
- Images inside `.emod` files.
- Changes to how `view`, `automation` or `translation` connect.
- A period where `trigger` and `screen` are both valid, and a new language version. The proposal retires `trigger` directly (Versioning section).
- A `title` attribute that restores a human label with spaces.
- A tool that migrates a model. The migration is by hand, with the steps in the proposal.

The boundaries this breakdown adds:

- The highlight of the producers and the consumers of a selected node (`user-stories/model-queries.md` US-004). It is not built, and this story does not build it. When it is built, it must decide how its highlight combines with the highlight of the appearances of a screen, because both use the one highlight state of the viewer.
- A new way to select a node. A selection is the open detail panel, which the block's context menu opens ("Open field editor"). A left click on a block still does nothing.
- A highlight on hover, and a highlight from the `screen` declaration node. The viewer does not draw the declaration.
- Names that tell apart two slices with the same name in the list (for example, the name of the context). Each entry still selects the appearance in its own slice.
- Bringing back an appearance that the visibility panel hides. Its slice stays in the list, and a click selects it and leaves the canvas where it is.
- Putting the highlight back while an appearance stays selected after the close button of the diagnostics panel clears every highlight (`hideDiagnosticsPanel`, `ui.js:395-410`).
- A mark that tells which entry of the list is the slice of the selected appearance.
- Any change to the Go side. The diagram document already gives each appearance its own ID and the name of its screen as its label.
- An example model in `examples/` that shows one screen in more than one slice.

**Deferred:**

- The `image` path of the screen in the detail panel of an appearance goes to US-007, which adds `image`.

## Codebase Context

**What the diagram document gives the viewer.** `internal/export/diagram.go:403-422` (`appendShows`) writes each `shows` block as its own node: type `shows`, ID `shows-N` from a counter (so two appearances of one screen have different IDs), `parentId` its slice, and `label` the name of the screen. A shows node has no separate screen key (US-004 design, `tasks/us-004-screen-and-shows.design.yaml`). The declaration is a node of type `screen` with no parent, which the viewer does not draw. So the viewer finds the appearances of a screen by label, and the story needs no Go change. The viewer already treats two shows nodes with one label as one screen: `moveToSlice` refuses to move a shows node into a slice that holds one with the same label (`internal/frontend/static/model.js:156-174`). The validator rejects a second `shows` block of one screen in one slice, so each appearance is in a different slice, and an entry for each slice is an entry for each appearance. Slice names are not unique (no rule rejects two slices with one name), so the list cannot find an appearance by the name of its slice.

**What a selection is.** `showDetailPanel` (`ui.js:464-640`) sets `store.interaction.selectedNodeId` and draws the panel. The panel opens from the block's context menu item "Open field editor" (`ui.js:896-902`, `viewer.js:1264-1269`). A left click on a block does nothing (`ui.js:1099-1102`). `hideDetailPanel` (`ui.js:756-763`) clears the selection. These reach it: Escape (`ui.js:1213-1231`), the close button of the panel (`viewer.js:1290-1293`), a click on the empty canvas (`ui.js:1104-1112`), the deletion of a node (`viewer.js:103`), and a click on a copy, an arrow, a context label or an aggregate label before each highlights something of its own. Each redraw also closes the panel and clears the highlight (`viewer.js:127-135`).

**How a highlight works.** `highlightElements`, `addHighlight` and `clearHighlights` (`ui.js:765-786`) put `.hl` on a block and `has-highlights` on the canvas. The stylesheet then dims each block that has no `.hl` to opacity 0.3 and paints a highlighted block with a blue stroke and a fill for its type, `.hl.shows-block` included (`viewer.html:1264-1294`). Today no selection keeps a highlight: each highlight first closes the panel, and `showDetailPanel` clears all highlights (`ui.js:470`). Therefore the selected appearance must be in the highlighted set, or the stylesheet dims the box that the user selected.

**How a click selects something that is somewhere else.** A click on a copy highlights its original and emits `node:show` (`ui.js:138-143`). `viewer.js:142-144` answers with `Interaction.centreOn` (`interaction.js:92-101`), which moves the canvas so that the node sits in its middle, and does nothing for a node with no layout position (a hidden one). The diagnostics list is the precedent for a list whose entries lead to nodes (`ui.js:275-376`).

**model-queries US-004 is not built.** A selection today dims arrows for a view only (`ui.js:110-128`) and highlights no node. That story highlights a different set (the ends of the arrows of a selected node), with the same highlight state.

**Every distribution gets one copy.** `task build:desktop` copies `internal/frontend/static/*` into `cmd/emod-desktop/frontend/static` and replaces only `platform.js` (`Taskfile.yml:61-70`). `emod diagram --serve` embeds the same directory (`internal/frontend/embed.go`, `internal/viewer/serve.go`), and `task build:web` copies it (`Taskfile.yml:107-117`). A change to `ui.js` therefore reaches the web viewer, `emod diagram --serve` and the desktop app. The story asks for all three, so the tension in the learning about desktop stories (`tasks/learnings.md:1261-1267`) does not apply. Two guards in `internal/frontend/tests/viewer.test.js` keep the shared modules free of host code: "only the desktop implementation reaches the host" (`:7069-7092`) and "the platform seam has one contract" (`:7098-7130`).

**Tests.** The vitest suites on jsdom are in `internal/frontend/tests` (`task test:viewer`). `detail-panel.test.js` drives `UI.showDetailPanel` with a store made by hand, and its `shows` tests are at `:167-217`. `viewer.test.js` drives the whole viewer with a mocked platform: `startViewer`, `blockFor`, `menuItemFor` and `fireMouse`. It opens the panel from the context menu (`:929-958`), checks the highlight and the centring after a click on a copy (`:2410-2433`), and checks that a gesture which only changes the view sends no edit (`:3877-3912`). The Playwright suite `e2e-viewer` (`task test:e2e:viewer`, Chromium, built from `task build:web`) reads computed paint (`e2e-viewer/tests/copies.spec.js:141-175`) and opens the panel from the context menu (`e2e-viewer/tests/interaction.spec.js:194-212`). No model in `examples/` shows one screen in two slices, so the tests bring their own model.

**Words.** "Appearance" is the word for one shows block in names and prose (US-004 and US-005 designs). `internal/frontend/tests/renderer.test.js:401` already uses `appearanceOf(arrow)` for the paint of an arrow, which is a second meaning in `internal/frontend`. "Shown in" is the name of the list of references of a screen in `emod references` (`query.GroupShownIn`, US-005 design). "Mark" names the outline that hovering a copy puts on its original (`markOriginalOf`, `.copy-original`). "Highlight" names the `.hl` state.

## Tasks

### Task 1: Highlight each appearance of a screen while one of them is selected

**Behavior:** When the user opens the detail panel of a shows box, each box that shows the same screen is highlighted. The selected box is one of them. The rest of the diagram dims, as it does for each highlight in the viewer. When the selection is cleared, nothing stays highlighted.

**Acceptance Criteria:**
- [x] When the user selects an appearance of a screen that two or more slices show, each other appearance of that screen is highlighted. (story 1)
- [x] The selected appearance is highlighted with the others and is not dimmed. No box that shows a different screen, and no node of another type, is highlighted.
- [x] For a screen that one slice shows, a selection of its appearance highlights no other box. (story 3, first half)
- [x] When the user clears the selection, by any path that reaches `hideDetailPanel`, the highlight of the appearances goes. After the clear, the only highlighted nodes are the ones that the clearing action highlights itself, for example the two ends of a clicked arrow. (story 4)
- [x] When the user selects another node, the highlight follows the new selection: a node that is not an appearance leaves no appearance highlighted, and an appearance of a different screen highlights the appearances of that screen only.
- [x] In a real browser (`e2e-viewer`), after a selection of one appearance, the other appearance of the screen has the stroke of a highlighted block, the selected appearance keeps full opacity, and a block of another type is dimmed.
- [x] The behaviour is in the shared modules under `internal/frontend/static` and calls no platform operation. No file under `internal/frontend/desktop/` changes. The guards "only the desktop implementation reaches the host" and "the platform seam has one contract" in `viewer.test.js` pass with no change. (story 5)
- [x] "Appearance" has one meaning in `internal/frontend`: the helper in `renderer.test.js` that reads the paint of an arrow has a name without that word.

**Affected Files/Modules:**
- `internal/frontend/static/model.js` — the appearances of a screen, found by label, beside the other node lookups
- `internal/frontend/static/ui.js` — a selection of a shows node highlights the appearances of its screen
- `internal/frontend/tests/viewer.test.js` — the highlight after a selection from the context menu, its removal on each clearing path, and a change of selection
- `internal/frontend/tests/renderer.test.js` — rename of the arrow-paint helper at `:401` and its callers at `:1388-1389`
- `e2e-viewer/tests/interaction.spec.js` (or a new spec beside it) — the computed paint of the highlight in Chromium, with a model that shows one screen in two slices

**Patterns to Follow:**
- `internal/frontend/static/ui.js:138-143`
- `internal/frontend/static/ui.js:765-786`
- `internal/frontend/static/ui.js:1045-1053`
- `internal/frontend/static/model.js:156-174`
- `internal/frontend/static/model.js:233-240`
- `internal/frontend/tests/viewer.test.js:929-958`
- `internal/frontend/tests/viewer.test.js:2410-2417`
- `internal/frontend/tests/diagnostics-highlight.test.js:123-140`
- `e2e-viewer/tests/copies.spec.js:141-175`
- `e2e-viewer/tests/interaction.spec.js:194-212`
- `tasks/learnings.md:206-210`
- `tasks/learnings.md:403-407`
- `tasks/learnings.md:695-700`
- `tasks/learnings.md:1227-1233`
- `tasks/learnings.md:1687-1693`

**Testable:** Yes

**Certainty:** medium — `highlightElements` over a computed set of IDs is the precedent (`ui.js:138-143`, `ui.js:1045-1053`), and `model.js:161` already groups shows nodes by label. But each existing highlight first closes the detail panel, and `showDetailPanel` clears all highlights, so no highlight yet stays on during an open selection. The selected box must also be in the set, or the `has-highlights` rule dims it.

**Blast radius:** low — it changes only what the viewer paints during a selection. It writes nothing to the model and changes no contract that a consumer outside the repository reads.

**Verification:** `task test:viewer` passes. `task test:e2e:viewer` passes. Run it yourself, because the audit cannot build the web bundle. Build the bundle with `task build:web`, serve it, render a model that shows one screen in two slices and another screen in one slice, and look: select each appearance from its context menu, then press Escape.

**Depends on:** None

---

### Task 2: List the slices where the screen appears in the detail panel of an appearance, and select an appearance from the list

**Behavior:** The detail panel of an appearance lists, under "Shown in", the slice of each appearance of its screen, its own slice included. A click on an entry selects the appearance in that slice. The panel then describes that appearance, the appearances of the screen stay highlighted, and the canvas moves until the selected appearance is in its middle. A click on a copy does the same for its original.

**Acceptance Criteria:**
- [x] The detail panel of an appearance lists each slice where its screen appears, its own slice included, by the name of the slice. The list follows the order of the slices in the diagram, so the slices of one aggregate read left to right, as on the canvas. (story 2, first half)
- [x] For a screen that one slice shows, the list has one entry: the slice of the appearance. (story 3, second half)
- [x] A click on a slice in the list selects the appearance in that slice. The panel then shows the views, the commands and the source line of that appearance, and the appearances of the screen stay highlighted. (story 2, second half)
- [x] The click moves the canvas until the selected appearance is in the middle of it.
- [x] When two slices that show the screen have the same name, the list has two entries, and each entry selects the appearance in its own slice.
- [x] A slice name that contains markup shows as text.
- [x] A click on a slice in the list sends no edit, and the source panel stays as it is.
- [x] The detail panel of a node that is not an appearance has no such list.
- [x] No file under `internal/frontend/desktop/` changes, and the two guards in `viewer.test.js` that Task 1 names pass with no change. After `task build:desktop`, the copies of `ui.js` and `model.js` under `cmd/emod-desktop/frontend/static` are byte-identical to the files in `internal/frontend/static`. (story 5)

**Affected Files/Modules:**
- `internal/frontend/static/ui.js` — the "Shown in" list in the panel of a shows node, and the click on an entry
- `internal/frontend/static/viewer.html` — the style of an entry that a click can select, if the panel needs one
- `internal/frontend/tests/detail-panel.test.js` — what the list holds, for one and for many slices, for a repeated slice name, and for markup
- `internal/frontend/tests/viewer.test.js` — a click on an entry selects that appearance, centres the canvas, keeps the highlight, and sends no edit

**Patterns to Follow:**
- `internal/frontend/static/ui.js:519-527`
- `internal/frontend/static/ui.js:529-536`
- `internal/frontend/static/ui.js:598-622`
- `internal/frontend/static/ui.js:138-143`
- `internal/frontend/static/viewer.js:142-144`
- `internal/frontend/static/ui.js:89-99`
- `internal/frontend/tests/detail-panel.test.js:167-217`
- `internal/frontend/tests/viewer.test.js:2419-2433`
- `internal/frontend/tests/viewer.test.js:3877-3912`
- `tasks/learnings.md:881-886`
- `tasks/learnings.md:1043-1048`
- `task:1`

**Testable:** Yes

**Certainty:** high — each part repeats a precedent in `ui.js`. A list of names with escaped text is at `:529-536`. A control in the panel that draws the panel again for a node is at `:598-622`. A click that selects a node somewhere else and centres the canvas on it through `node:show` is at `:138-143`. Task 1 supplies the appearances of the screen and the highlight.

**Blast radius:** low — it adds a list to the detail panel and changes the selection and the viewport. It writes nothing to the model and changes no contract that a consumer outside the repository reads.

**Verification:** `task test:viewer` passes. `task test:e2e:viewer` passes. Build the web bundle with `task build:web`, serve it, render a model that shows one screen in three slices, open the panel of one appearance, click each entry of the list, and look. `task build:desktop` succeeds, and its copies of the changed modules match `internal/frontend/static` byte for byte.

**Depends on:** Task 1

## Summary

- **Total tasks:** 2, both JavaScript in the viewer. The story needs no Go change, because the diagram document already gives each appearance its own ID and the name of its screen as its label.
- **Ordering:** Task 1 comes first because it is the riskier change. It makes a selection keep a highlight for the first time, and it adds the lookup of the appearances of a screen. Task 2 reads that lookup for the list, and its click reaches the highlight of Task 1 through the selection.
- **Story criteria:** criterion 1 → Task 1. Criterion 2 → Task 2. Criterion 3 → Task 1 (no other box highlighted) and Task 2 (one slice listed). Criterion 4 → Task 1. Criterion 5 → both tasks: the behaviour lives in the one copy of the frontend that each distribution serves, the host guards pass with no change, and Task 2 checks the copy that the desktop build makes. None is deferred.
- **Decisions to confirm:** the selected appearance is highlighted with the others, so a selection of a screen that one slice shows highlights that box and dims the rest of the diagram. If it were not highlighted, the stylesheet would dim the box that the user selected. A click on an entry of the list also centres the canvas on the appearance, as a click on a copy does. Without that, a selection off screen changes nothing that the group sees on the canvas.

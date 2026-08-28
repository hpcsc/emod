# US-004: Know when there are unsaved changes

## Contents

1. Task 1: Mark the window when the panel stops matching the file it came from
2. Task 2: Show the marker as a document-edited window on macOS
3. Task 3: Prompt before another model replaces unsaved edits
4. Task 4: Refuse to close or quit with unsaved edits
5. Task 5: Document unsaved-change marking

The five fall into four slices. Each leaves `main` shippable, and the browser and CLI
distributions behave exactly as they do today at every one of them.

| Slice | Tasks | Exit |
|---|---|---|
| **The marker** | 1, 2 | The shared viewer knows whether the panel still matches the file and tells the host; the desktop window says so in the platform's own way |
| **The prompt on opening** | 3 | One function is the only way a different model reaches the screen, and it asks first |
| **The prompt on leaving** | 4 | The shell refuses a close or a quit that would discard edits, and asks the frontend what to do |
| **Documentation** | 5 | README and architecture describe what the app now does |

---

## Story Reference

`user-stories/emod-desktop.md` → **US-004: Know when there are unsaved changes** (`:68-79`).
Depends on US-003, delivered — `tasks/completed/us-003-save-a-model-back-to-the-file-it-came-from.md`,
whose Boundaries section deferred exactly this work: "The modified marker, the prompt before
losing edits, and Discard → US-004."

US-005 (drop-to-open) and US-006 (recent files) are **not** delivered, and the story's fourth
criterion names both. How that is handled is Decision 2 below: the guard sits on the one
function that puts a different file's model on screen, so the two paths that exist today go
through it and the two that arrive later cannot avoid it.

The story's six criteria and where they land:

| Story criterion | Task |
|---|---|
| Any change that would alter the saved file marks the window as modified using the platform's convention | 1 decides and reports it, 1 writes `*` into the title, 2 replaces that with the dot on macOS |
| A successful save clears the modified marker | 1 |
| Closing the window or quitting with unsaved changes prompts with Save, Discard, and Cancel; Cancel aborts the close | 4 vetoes and asks, 3 owns the prompt |
| Opening another model — dialog, recent files, or drop — with unsaved changes goes through the same prompt | 3 |
| Discarding leaves the file on disk unchanged | 3 on the open path, 4 on the close path |
| Moving a node for layout alone does not mark the file as modified | 1 |

---

## Theory and decided questions

### Decision 1 — "a change that would alter the saved file" is a change to the source panel's text

Save writes `sourceToSave(store)` (`internal/frontend/static/viewer.js:30-40`), which is the
source panel's text put back into the file's own line-ending convention. It is never the model
re-serialised from the diagram — US-003 chose that deliberately, because the exporter
canonicalises formatting and drops comments, and `README.md:404-408` states it.

Nothing outside `viewer.js` writes to the panel. Verified: `sourceInput` is read or assigned at
nine places, all of them in `viewer.js` and `store.js`, and none of them in `ctx-actions.js`,
`ui.js`, `interaction.js` or `model.js`. So the edit surfaces divide cleanly:

| Surface | Reaches the saved file? |
|---|---|
| Typing, pasting or cutting in the source panel | **Yes** — it is what Save writes |
| Context-menu edits to the diagram (add slice/command/event/flow, delete arrow, reorder slices) | No |
| Deleting a node, renaming a label inline, editing a node's fields, drawing an edge | No |
| Dragging a node (`store.nodeOffsets`) | No — layout only |
| Pan, zoom, fit-to-view (`store.viewport`) | No |
| Hiding a node from the visibility tree, collapsing the data panel | No |

So the marker tracks the panel's text against the bytes the open file arrived with, and a
diagram edit does not raise it.

**This narrows the story's category, and it is the one place where it does.** The criterion says
"any change that would alter the saved file"; the list above is what that category contains
*today*, derived from what `writeModel` actually hands the host rather than from the story's
examples. Two things make the narrowing the right reading rather than a convenient one:

- Marking a diagram edit as modified would make Save a lie. Save would write the panel's text,
  clear the marker, and silently discard the very edit that raised it — worse than not marking.
- The story set that already knows this. Open Question 1 (`user-stories/emod-desktop.md:263`) and
  **US-011: Keep the source panel and the diagram in agreement** (`:165-178`) are where a diagram
  edit starts reaching the source. On the day US-011 lands, a canvas edit changes the panel's
  text and this marker picks it up with no change here — because it watches the panel, not a
  list of actions.

The consequence to be honest about: a user who edits only the diagram and closes the window
still loses that work with no prompt. That is today's behaviour, not something this story
introduces, and it is recorded under Boundaries rather than fixed here.

### Decision 2 — the guard sits on the one function that replaces the open model

US-005 and US-006 both add a way to put a different file on screen, and the story asks that all
of them prompt. Rather than wiring the prompt into the File ▸ Open handler, `viewer.js` gains a
single function that runs the guard and then calls `renderPanelSource` with a file — and it is
the **only** caller that passes a file argument. `renderPanelSource(undefined)`, which is what
the Render button does, stays unguarded because re-rendering the panel replaces no model.

Today two callers route through it: the host-opened file (`viewer.js:273-295`) and the panel's
drop (`:252-270`). Later:

- **US-005 (drop)** either keeps the panel's drop listener or replaces it with a native one; both
  end at the same function, and a native drop that carries a real path changes only what it
  passes as the file, not whether it is guarded.
- **US-006 (recent files)** is specified as opening "exactly as the file dialog would"
  (`user-stories/emod-desktop.md:101`), i.e. through the shell's delivery into `onFileOpened` —
  the guarded path, with no new code in `viewer.js` at all.

What is deliverable today is the dialog path and the drop path. What guarantees the other two
arrive guarded is that there is nowhere else to arrive: a new entry point that does not call
this function renders nothing.

### Decision 3 — the modified state lives in Go, and the shell's veto reads it there

Wails cancels a close from a **hook**, synchronously, before the listener that destroys the
window runs (`webview_window.go:977-1006`), and `WebviewWindow.Close()` emits the same event — so
a frontend that closed the window itself would be vetoed by the same hook. The hook therefore
cannot ask the frontend and wait; it has to already know.

So the frontend pushes its modified state across on every change, Go holds it, and the veto is a
field read. Discard clears the flag and then closes, which passes the hook by construction. This
also means a clean window closes with no round trip to the frontend at all — a wedged webview
cannot make the app unclosable.

### Decision 4 — Wails v3 beta.9 has no document-edited API, and the dot needs cgo

Read out of `$(go env GOMODCACHE)/github.com/wailsapp/wails/v3@v3.0.0-beta.9`: there is no
`documentEdited`, `setDocumentEdited` or `representedFilename` anywhere in the module — not in
the `Window` interface (`pkg/application/window.go:9-140`), not in the darwin shims, not in the
JS runtime. The framework's own answer to a modified window is to change the title.

`WebviewWindow.NativeWindow() unsafe.Pointer` (`pkg/application/webview_window.go:1649`) returns
the `NSWindow*` on darwin (`webview_window_darwin.go:1652-1654`), so `setDocumentEdited:` is
reachable from a darwin-tagged cgo file in `cmd/emod-desktop` — the one package in this
repository that already links CGO. Task 2 does that; Task 1 ships the `*` title everywhere first,
so the story's "elsewhere" convention is delivered and working before the macOS-specific route is
attempted.

### Decision 5 — the browser build is inert on both new operations

The seam gains two operations. `platform.browser.js` implements them the way it already
implements `onFileOpened` and `onSaveRequested` (`:118-124`): accepted, and doing nothing.

- **The marker.** A page has no window to mark, and marking a tab title would announce work the
  browser viewer offers no way to save — it has no save-in-place, only the Export download.
- **The confirmation.** A browser has no shell dialog with a Save button that writes anywhere, so
  it answers "discard" and the web viewer's drop behaves exactly as it does today. A page's own
  unload prompt is the browser's convention for this and adding one is not in this story.

### Decision 6 — the three-button prompt does not work on Windows, and that is US-016's problem

`Dialogs.Question` reaches the same Go dialog implementation from either language, and on Windows
that is `MessageBox` with `MB_OK`/`MB_YESNO` — custom button labels are ignored and the callback
fires only for a fixed label table (`pkg/application/dialogs_windows.go`). macOS supports up to
four custom buttons and Linux supports arbitrary ones, so Save/Discard/Cancel works on both
platforms this repository targets. **US-016** (`user-stories/emod-desktop.md:232-244`) already
carries the criterion that unsaved-change marking behaves on Windows "using Windows conventions";
that is where the degraded dialog is answered. Nothing here should pretend otherwise, which is
why Task 5's criteria forbid the README claiming it.

---

## Boundaries

**Out of scope** — the story's Non-Goals (`user-stories/emod-desktop.md:248-259`), carried
verbatim:

- **No code signing or notarization** on any platform — no Apple Developer account, no App Store, no Windows signing certificate. Users take a documented one-time step on first launch.
- **No Linux distributions below the declared floor** — Ubuntu 22.04, Debian 12, RHEL 9 and other pre-GTK4 stacks are out of scope, with no legacy build variant.
- **No auto-update.** New versions are downloaded manually.
- **No replacement of existing distributions.** `emod diagram --serve` and the hosted web viewer keep working unchanged; desktop is a third option, not a migration.
- **No desktop-only diagram features.** Anything added to the canvas stays available in all three distributions.
- **No multi-window, tabs, or project workspaces.** One model per window.
- **No full text editor.** The source panel gains validation, navigation, and highlighting — not find-and-replace, multi-cursor, or refactoring.
- **No telemetry or crash reporting.**
- **No automated tests driving the desktop window.** Manual smoke testing of the shell, backed by the existing Go and browser suites.
- **No published package of the shared frontend.** It stays an internal asset assembled at build time.

Also out of scope, from this decomposition:

- **Marking a diagram-only edit as unsaved.** Decision 1. A canvas edit does not reach the source
  panel and so is not what Save writes; marking it would produce a Save that clears the marker
  while discarding the work.
- **Any change to what Save or Export writes.** Save still writes `sourceToSave(store)`, Export
  still re-serialises the diagram through `Export.exportToEmodString`
  (`internal/frontend/static/emod-export.js:3-11`), and neither gains or loses a byte here.
- **An undo history, or a marker that survives relaunch.** The marker is derived from what is on
  screen, and the window opens with nothing on screen.
- **A prompt in the browser viewer.** Decision 5.
- **Implementing drop-to-open or recent files.** Decision 2 makes the shared entry point they will
  join; it does not build either of them. The panel's drop still yields contents and no path
  (`internal/frontend/static/platform.browser.js:75-92`), so a dropped model still cannot be
  saved back to where it came from.
- **Making the three-button prompt work on Windows.** Decision 6.
- **Noticing that the file changed on disk while it was open.** That is US-013; the marker
  compares the panel against the bytes the file arrived with, not against the file now.
- **A "Revert to saved" command.** Discard abandons the model being replaced; it does not reload
  the open file over the panel.

**Deferred** — wanted, but not here:

- **Prompting before losing a diagram-only edit** → US-011, once a diagram edit reaches the source
  panel and therefore the marker. Open Question 1 (`user-stories/emod-desktop.md:263`) is the part
  of it nobody has decided yet.
- **Dropping a file onto the window, guarded by this prompt** → US-005, whose own criterion says so
  (`user-stories/emod-desktop.md:88`).
- **Opening a recent file, guarded by this prompt** → US-006.
- **The Windows behaviour of the prompt** → US-016.
- **Prompting when the app is asked to open a file it was launched with** → US-008
  (double-click-to-open); nothing hands this window a model at startup today
  (`internal/frontend/desktop/platform.desktop.js:163-166`).

---

## Codebase Context

Verified against the working tree on 2026-08-28, branch `us-004-know-when-there-are-unsaved-changes`,
with US-001 to US-003 landed.

**The platform seam.** `internal/frontend/static/platform.js` re-exports ten names from
`./platform.browser.js`; `internal/frontend/desktop/platform.desktop.js:169` exports the same ten
over the Wails runtime and the generated bindings. `task build:desktop` copies the desktop file
over `platform.js` in its own assembled tree (`Taskfile.yml:48-65`). Three source-scanning guards
sit in `internal/frontend/tests/viewer.test.js`: the seam contract, whose list of ten names is
hardcoded (`:1392-1421`), the DOM-id scan (`:1334-1355`), and the check that no shared module
imports the Wails runtime or the generated bindings (`:1362-1385`). **A name added to the seam has
to land in all three copies in one change**, or the contract test fails.

**What Save writes.** `sourceToSave(s)` (`viewer.js:30-40`) hands back `s.currentFile.content`
untouched when the panel matches its LF normalisation, and otherwise re-applies the file's own
convention. A dirty check comparing `sourceInput.value` against the file's content would report
every CRLF model as modified the moment it opened; `sourceToSave` is the thing to compare.

**The gap between the panel's text and the open file's identity.** `renderPanelSource`
(`viewer.js:185-221`) sets the panel's text immediately and commits `store.currentFile` only once
the parse resolves — deliberately, so a render the parse rejects leaves the window naming the
model still on screen. Anything reading both is reading across that gap. US-003's answer is the
`saveQueue` plus `rendersSettled()` (`:351-373`), which re-checks `latestRenderSettled` after each
wait because a file arriving during the wait starts another render. The recorded learning names
unsaved-change marking as the next operation to hit this, and it is the highest-risk part of
Task 1: the marker must be recomputed only where the panel and the file are known to agree —
after the parse resolves and `currentFile` is committed, after a rejected parse restores the
panel's previous text, after a save adopts its target, and on the user's own input, which is the
only one of the four that is not a programmatic assignment (a programmatic `.value =` fires no
`input` event).

**The window's name.** `applyWindowTitle(s)` (`viewer.js:42-45`) composes
`"<file or model name> — Emod Diagram Viewer"` and hands it to `setWindowTitle`, which is
`document.title` in the browser (`platform.browser.js:111-113`) and `Window.SetTitle` on the
desktop (`platform.desktop.js:88-92`). It is called from the `model:updated` subscriber and again
after a save adopts a new target. **There is no title getter in Wails** (verified: `SetTitle`
exists at `pkg/application/webview_window.go:422`, nothing reads it back), so whichever module
applies a `*` has to remember the unmarked title itself.

**The two entry points that replace the open model.** The panel's drop listener
(`viewer.js:252-270`) reads the file through `droppedFile`, checks the extension and calls
`renderPanelSource(content, null)`. The host's delivery (`:273-295`) reports an error or an empty
file and otherwise calls `renderPanelSource(opened.content, {name, path, content})`. Those two
calls are the only ones in the file that pass a second argument.

**`internal/desktop`.** `ModelService` (`service.go:15-35`) and `FileService`
(`file_service.go`) are the two bound services; the package "imports no GUI framework and links
no CGO, so it is testable and buildable everywhere the rest of the repository is"
(`service.go:1-6`). Four guards cross the language boundary from here, and each constrains this
story:

- `TestBindingNames` (`binding_names_test.go:22-36`) requires every `<Service>.<Method>(` call in
  `platform.desktop.js` to be an exported method of that receiver in this package.
- `TestServiceRegistrations` (`:61-75`) requires every service the frontend imports to be
  registered in `main.go`, and finds registrations with the regex
  `application\.NewService\(&desktop\.(\w+)\{` — so a service registered through a constructor
  call rather than a struct literal is invisible to it, and the check then fails.
- `TestServiceWireKeys` (`:133-157`) pairs JSON struct tags with the fields the frontend reads.
- `TestShellEventNames` (`event_names_test.go:20-30`) compares the events `main.go` emits with the
  events `platform.desktop.js` subscribes to and requires them **equal in both directions** — so a
  new shell event and its frontend subscription must land in the same change. Its emit-side regex
  is `EmitEvent\("([^"]+)"`, which sees `window.EmitEvent(...)` and would not see
  `app.Event.Emit(...)`.

**The shell.** `cmd/emod-desktop/main.go:62-90` builds the app with three options and a services
list, then creates the window, then installs the application menu. It is excluded from
`task test:unit` and `task test:integration` (`Taskfile.yml:86-94`) because its `//go:embed
all:frontend` cannot resolve until `task build:desktop` has assembled the directory — so nothing
but that build task compiles it, and the story's own non-goal sanctions manual smoke testing here.

**The Wails v3 beta.9 surface this story uses**, read out of the module cache:

- `WebviewWindow.RegisterHook(events.Common.WindowClosing, func(*WindowEvent))`
  (`pkg/application/webview_window.go:957`) plus `WindowEvent.Cancel()` (`:162`). Hooks run first
  and synchronously in `HandleWindowEvent` (`:977-990`); a cancelled event returns before the
  listener registered in `NewWindow` (`:361-366`) — the one that actually destroys the window —
  ever runs. All three platforms already suppress the native close and route it through this
  event, so the veto is portable. The framework's own `examples/hide-window/main.go:35-38` is the
  canonical use.
- `application.Options.ShouldQuit func() bool` (`pkg/application/application_options.go:94`),
  consulted by `App.shouldQuit()` (`application.go:1029`). Cmd+Q on macOS goes through
  `applicationShouldTerminate` and **bypasses the per-window close hook entirely**, so both are
  needed. `Options.OnShutdown` cannot veto.
- JS `Dialogs.Question({Title, Message, Buttons: [{Label, IsDefault, IsCancel}]})`
  (`internal/runtime/desktop/@wailsio/runtime/src/dialogs.ts:168`) resolves with the **label of the
  button that was clicked** — `processDialogMethod` (`pkg/application/messageprocessor_dialog.go:58-76`)
  wires each button's callback to send its own label back. The dialog is attached to the window
  unless `Detached` is set, which on macOS makes it a sheet rather than a blocking modal.
- JS `Window.Close()` (`window.ts:248`) and `Application.Quit()` (`application.ts:35`).
- JS `System.IsMac()` (`system.ts:120`) is synchronous and reads
  `window._wails.environment.OS`, which the host populates at page setup — so it is safe to call
  when the marker is applied and not at module scope.

**The vitest harness.** `internal/frontend/vitest.config.js` aliases `/wails/runtime.js` to
`tests/wails-runtime-stub.js` and the generated bindings to `tests/bindings-stub.js`; both grow a
member per host operation used. `viewer.test.js` mocks only `../static/platform.js` and asserts
what a user would see, with the window's name asserted through the seam
(`:205-221`, "the window is named through the host, not by assigning document.title") because
there is no native window to look at. `task test:viewer` runs the suite and is not part of
`task test:unit`.

---

## Tasks

### Task 1: Mark the window when the panel stops matching the file it came from

**Behavior:** The viewer knows whether what Save would write still equals what the open file
arrived with, tells the host whenever that answer changes, and the desktop window says so by
carrying `*` ahead of its name. A successful save clears it; a diagram edit never raises it.

**Acceptance Criteria:**
- [x] With a file open and nothing typed, the host is not told the model is modified — including a file whose line endings are CRLF, which the panel rewrote the moment it arrived
- [x] Typing into the source panel tells the host the model is modified, without waiting for a Render
- [x] Editing the panel back to exactly the text the file arrived with tells the host it is no longer modified
- [x] A save that succeeds tells the host the model is no longer modified; a save the host refuses, and a save whose location dialog was cancelled, leave it modified
- [x] Saving pasted source with no file open, and adopting the location the host chose, tells the host the model is no longer modified
- [x] A delivered file the pipeline will not render leaves the marker describing the model still on screen, not the one that never arrived
- [x] A change confined to the diagram or the viewport tells the host nothing — witnessed by a node drag, a pan, hiding a node from the visibility tree, collapsing the data panel, and a context-menu edit that adds a node — and the same leaf shows a panel edit that does tell the host, so the leaf can fail
- [x] The desktop window's title carries `*` ahead of the name while the model is modified, and loses it when it is not
- [x] Re-naming the desktop window while it is marked keeps the `*`, and marking a window whose name has never been set still marks it
- [x] The browser implementation changes nothing a page shows or a page's title says
- [x] The seam contract test names the new operation, and both implementations satisfy it

**Affected Files/Modules:**
- `internal/frontend/static/viewer.js` — derives the modified answer beside `sourceToSave`, recomputes it at the four moments the panel and the file are known to agree, listens for the panel's own input, and pushes it through the seam
- `internal/frontend/static/platform.js` — the contract gains the operation
- `internal/frontend/static/platform.browser.js` — an inert implementation with the reason, beside the two that already are
- `internal/frontend/desktop/platform.desktop.js` — remembers the unmarked title and re-applies both it and the marker
- `internal/frontend/tests/viewer.test.js` — the shared viewer's leaves, and the seam contract's name list
- `internal/frontend/tests/platform.desktop.test.js` — the title composition
- `internal/frontend/tests/wails-runtime-stub.js` — records what the title was set to across a sequence, not only the last value

**Patterns to Follow:**
- `internal/frontend/static/viewer.js:30-45` for where a save-shaped derivation and the window's name already live
- `internal/frontend/static/viewer.js:185-221` and `:351-373` for the panel-text/file-identity gap and how US-003 answered it
- `internal/frontend/static/platform.browser.js:111-124` for an inert browser implementation carrying its reason
- `internal/frontend/desktop/platform.desktop.js:88-92` for the desktop title
- `internal/frontend/tests/viewer.test.js:205-221` for asserting a window-level effect through the seam, and `:1392-1421` for the contract the three copies share
- `internal/frontend/tests/platform.desktop.test.js:91-97` for a desktop-side leaf against the runtime stub

**Testable:** Yes — the shared viewer through `init()` and the mocked seam, the desktop module through the runtime stub.

**Certainty:** medium — `setWindowTitle` is the precedent for a window-level fact reaching the host through the seam and being asserted through it, but no existing seam operation is derived from store state that moves on every keystroke, and none has to be recomputed against `latestRenderSettled`.

**Blast radius:** low — it reports and re-titles; it writes nothing and discards nothing.

**Verification:** `task test:viewer` green. `task build:desktop && ./bin/emod-desktop`: open a model, type a character, watch `*` appear in the title, press ⌘S, watch it go.

**Depends on:** None

---

### Task 2: Show the marker as a document-edited window on macOS

**Behavior:** The shell holds the frontend's modified state, and on macOS shows it the way macOS
shows it — the dot in the close button — instead of a `*` in the title.

**Acceptance Criteria:**
- [ ] `internal/desktop` exposes a bound service the frontend tells that the model is modified, and it reports back the value it was last given
- [ ] Every value the service is told is handed to the injected thing that shows a window as edited, so a test observes the sequence with no window in existence
- [ ] Setting and reading the state from several goroutines is clean under `go test -race`
- [ ] The desktop platform module tells the service on every change of the modified state, and asks for nothing when the state has not moved
- [ ] On macOS the title carries no `*` and the window itself is marked as edited; on every other platform the title keeps the `*` Task 1 gave it
- [ ] A non-darwin build compiles and its marker does nothing
- [ ] `TestBindingNames` and `TestServiceRegistrations` both cover the new service, and `task test:unit` is green
- [ ] `task build:desktop` succeeds

**Affected Files/Modules:**
- `internal/desktop/` — a new file for the new type, holding the modified state and the role it hands each change to
- `cmd/emod-desktop/main.go` — registers the service through the struct-literal shape `TestServiceRegistrations` matches, and supplies the thing that marks the window
- `cmd/emod-desktop/` — a darwin-tagged cgo file reaching `NSWindow setDocumentEdited:` through `WebviewWindow.NativeWindow()`, and a build-tagged counterpart for everywhere else
- `internal/frontend/desktop/platform.desktop.js` — calls the service, and stops adding `*` where the shell marks the window itself
- `internal/desktop/` tests — the service's own umbrella
- `internal/frontend/tests/platform.desktop.test.js` — the two platforms' behaviour, and `internal/frontend/tests/wails-runtime-stub.js` — a settable OS
- `internal/frontend/tests/bindings-stub.js` — the new service

**Patterns to Follow:**
- `internal/desktop/service.go:1-22` for a bound service's shape and the package's own constraint
- `internal/desktop/binding_names_test.go:22-36` and `:61-75` for the two guards the registration has to satisfy, including the regex that only sees a struct literal
- `cmd/emod-desktop/main.go:62-90` for where services are registered and the window is created
- `internal/frontend/desktop/platform.desktop.js:88-92` for the module the title marker lands in

**Testable:** Yes — the Go service through its exported methods, and the platform module's branch through the runtime stub. The cgo marker itself is not: `cmd/emod-desktop` is in no test target, which the story's non-goals cover.

**Certainty:** low — nothing in this repository has ever written cgo or reached a native window handle, and the framework offers no document-edited API at all, so the route through `NativeWindow()` plus a main-thread dispatch is the task's own design rather than a variation on something here.

**Blast radius:** low — a marker on a window; it neither writes files nor decides whether work is kept.

**Verification:** `task test:unit` green, including under `-race`. `task build:desktop && ./bin/emod-desktop` on macOS: open a model, type, and look at the close button for the dot and at the title for the absence of `*`; save and watch the dot clear. Cross-compile or build on Linux to confirm the non-darwin path.

**Depends on:** Task 1

---

### Task 3: Prompt before another model replaces unsaved edits

**Behavior:** A model arriving while the panel holds unsaved edits asks first, with Save, Discard
and Cancel. Cancel leaves everything as it was; Discard replaces the model and writes nothing;
Save writes the open file and then replaces it. Every path that puts a different file's model on
screen goes through one function, so the paths US-005 and US-006 add cannot skip it.

**Acceptance Criteria:**
- [ ] A model delivered by the host while the panel holds unsaved edits asks the host to confirm before the panel, the canvas, the window's name or the path in the bar changes
- [ ] A model dropped on the panel while the panel holds unsaved edits asks the same way
- [ ] Cancel leaves the panel's text, the diagram, the window's name, the path in the bar and the modified marker exactly as they were, and asks the host to save nothing
- [ ] Discard replaces the model and asks the host to save nothing, so the file the edits belonged to is never written
- [ ] Save writes the open file and then replaces the model; a save the host refuses, and one whose location dialog was cancelled, leave the model on screen and do not open the arriving one
- [ ] With no unsaved edits, a delivered file and a dropped file each open exactly as they do today, asking the host to confirm nothing
- [ ] A delivered file the host could not read, and one that is empty, still report their reason without asking anything
- [ ] `renderPanelSource` is called with a file argument from exactly one function, which is the one that runs the guard — pinned by a scan of `viewer.js`'s own source in the style of the scans already in this suite
- [ ] The desktop confirmation offers Save, Discard and Cancel, with Cancel designated the dialog's cancel button, and answers which of the three was chosen
- [ ] The browser implementation asks nothing and answers discard, and a dropped file in the browser viewer opens exactly as it does today
- [ ] The seam contract test names the new operation, and both implementations satisfy it

**Affected Files/Modules:**
- `internal/frontend/static/viewer.js` — one function that guards and then replaces the open model, with the drop listener and the host delivery routed through it
- `internal/frontend/static/platform.js`, `internal/frontend/static/platform.browser.js` — the contract and its inert browser implementation
- `internal/frontend/desktop/platform.desktop.js` — the three-button question, and the mapping from the label the runtime answers to the outcome the viewer acts on
- `internal/frontend/tests/viewer.test.js` — the guard's leaves and the source scan
- `internal/frontend/tests/platform.desktop.test.js` — the dialog's shape and its answers
- `internal/frontend/tests/wails-runtime-stub.js` — a `Dialogs.Question` beside the two pickers

**Patterns to Follow:**
- `internal/frontend/static/viewer.js:252-295` for the two entry points being routed
- `internal/frontend/static/viewer.js:356-408` for how a save's outcome is already learned and reported
- `internal/frontend/desktop/platform.desktop.js:130-160` for driving a Wails dialog and turning what it answers into something the viewer understands
- `internal/frontend/tests/viewer.test.js:430-487` and `:540-600` for the drop and host-delivery leaves this extends
- `internal/frontend/tests/viewer.test.js:1362-1385` for the established shape of a guard that scans a module's own source
- `internal/frontend/tests/wails-runtime-stub.js:22-35` for adding a dialog to the stub

**Testable:** Yes — both entry points drive through `init()` against the mocked seam, and the dialog through the runtime stub.

**Certainty:** medium — the drop and host-delivery handlers and the `Dialogs.OpenFile` call are the precedents for both halves, but no existing seam operation asks the user a question and gates a later action on the answer, and none of them has to distinguish "the host answered nothing because it was cancelled" from "the host answered a choice".

**Blast radius:** high — a wrong mapping from a button to an outcome either writes the user's file when they asked to cancel, or discards unsaved edits when they asked to keep them; both are irreversible from inside the app.

**Verification:** `task test:viewer` green. `task build:desktop && ./bin/emod-desktop`: open a model, type, then File ▸ Open another — check each of the three buttons, and confirm with `shasum` that Discard left the first file untouched.

**Depends on:** Task 1

---

### Task 4: Refuse to close or quit with unsaved edits

**Behavior:** Closing the window or quitting the app while edits are unsaved does neither. The
shell asks the frontend instead, which raises the same Save / Discard / Cancel prompt: Cancel
leaves the window open, Discard closes and leaves the file untouched, Save writes first and then
closes. With nothing unsaved, both go straight through.

**Acceptance Criteria:**
- [ ] Closing the window while the frontend has reported unsaved edits leaves the window open and asks the frontend instead
- [ ] Quitting while the frontend has reported unsaved edits leaves the app running and asks the frontend instead
- [ ] Closing and quitting with nothing unsaved each proceed with no prompt, and with no round trip to the frontend at all
- [ ] The prompt raised is the same Save / Discard / Cancel confirmation the open path raises, from the same place
- [ ] Cancel leaves the window open with the model, the panel's text and the marker exactly as they were, and writes nothing
- [ ] Discard closes, or quits, and leaves the file on disk byte-for-byte as it was
- [ ] Save writes the open file and then closes, or quits; a save that fails and one whose location dialog was cancelled each leave the window open
- [ ] The shell's new events are subscribed by the frontend, so `TestShellEventNames`' both-directions comparison holds, and `task test:unit` is green
- [ ] `task build:desktop` succeeds

**Affected Files/Modules:**
- `internal/desktop/` — the document-state service gains the read the shell's veto uses
- `cmd/emod-desktop/main.go` — a close hook that cancels and emits, and a quit veto that does the same, both reading that state
- `internal/frontend/desktop/platform.desktop.js` — subscribes to both, runs the confirmation, and proceeds by clearing the state and closing or quitting
- `internal/desktop/` tests — the read's own leaves
- `internal/frontend/tests/platform.desktop.test.js` — what each answer does
- `internal/frontend/tests/wails-runtime-stub.js` — a `Window.Close` and an `Application.Quit` to record

**Patterns to Follow:**
- `cmd/emod-desktop/main.go:32-60` for a shell control reaching the frontend by event, and `:62-90` for where app options and the window are built
- `internal/desktop/event_names_test.go:20-42` for the guard that requires both sides to name the same events
- `internal/frontend/desktop/platform.desktop.js:105-120` for subscribing to a shell event and handing the frontend's promise back rather than dropping it
- `internal/desktop/service.go:10-22` for the service the state lives on

**Testable:** Yes on the frontend half, through the runtime stub driving each subscription and each answer. The hook and the quit veto themselves are not: `cmd/emod-desktop` is in no test target, which the story's non-goals cover.

**Certainty:** low — this repository has never registered a window hook or an application-level quit veto, and correctness rests on a hook cancelling before the listener that destroys the window plus a programmatic close passing the same hook once the flag is cleared. Both are stated by the framework's source and neither has been run here, on either platform.

**Blast radius:** high — this is the code that decides whether unsaved work survives, and a wrong branch either loses it silently or makes the window impossible to close.

**Verification:** `task test:unit` and `task test:viewer` green. `task build:desktop && ./bin/emod-desktop`: with edits pending, try ⌘W and ⌘Q separately and exercise all three buttons on each; `shasum` the file before and after a Discard; then confirm a clean window closes and quits with no prompt.

**Depends on:** Task 2, Task 3

---

### Task 5: Document unsaved-change marking

**Behavior:** The README's desktop section and the architecture document describe what the app
now does — what marks a model as modified, what clears it, and what happens on close, quit and
open — and stop claiming it does not.

**Acceptance Criteria:**
- [ ] The README's desktop section says what raises the marker, what clears it, and how it is shown on each platform
- [ ] It says what closing, quitting and opening another model do when edits are unsaved, naming all three buttons
- [ ] The README's "What it does not do yet" paragraph no longer says nothing marks unsaved changes, and still says a diagram edit does not reach the source panel and so is not what Save writes
- [ ] `docs/architecture.md`'s sentence listing what the platform seam provides names both operations added here
- [ ] Neither document claims the three-button prompt works on Windows
- [ ] Neither document describes drop-to-open or recent files as guarded, since neither exists yet
- [ ] `git diff` touches no file outside `README.md` and `docs/architecture.md`

**Affected Files/Modules:**
- `README.md` — the desktop section
- `docs/architecture.md` — the platform-seam paragraph

**Patterns to Follow:**
- `README.md:371-417` for the section's voice and the paragraph that has to change
- `docs/architecture.md:225-238` for the seam's prose list of what a host provides

**Testable:** No — prose, with no snippet either document runs.

**Certainty:** high — US-003 wrote both of these sections for the previous story (`README.md:395-416`, `docs/architecture.md:225-238`), and this extends them in place.

**Blast radius:** low — documentation.

**Verification:** Read both sections against the branch: every sentence names behaviour the code has, and every claim the code no longer makes is gone.

**Depends on:** Task 4

---

## Summary

**Five tasks.** Ordered dependency-first, and within that by how much later work rests on the
answer: the marker is the fact everything else reads, so it lands first, and the two prompts
follow in the order that lets the second reuse the first's dialog.

- Task 1 is the only one the other four need, and it is the one that decides what "modified"
  means for this app.
- Task 2 and Task 3 are independent of one another and could land in either order; Task 3 is
  written second because Task 4 needs both.
- Task 4 is last of the code tasks because its prompt is Task 3's and its state is Task 2's.
- Task 5 follows the repository's own convention of documenting a desktop story as its closing
  task.

**Story coverage.** All six criteria are covered, none deferred. Two carry a qualification stated
in full above:

- "Any change that would alter the saved file" is read as a change to the source panel's text,
  because that is what Save writes; Decision 1 says why, and names US-011 as where a diagram edit
  joins it.
- "Opening another model — dialog, recent files, or drop" is delivered for the dialog and the
  panel's drop, which are the paths that exist. Decision 2 explains the shape that makes US-005
  and US-006 arrive already guarded rather than needing to be wired in again.

**Assessments to overrule before they are built on.**

- **Task 2 — low certainty.** No cgo has ever been written here and Wails offers no
  document-edited API, so the macOS dot goes through `NativeWindow()` and Objective-C. If that
  proves unreachable, the fallback is the `*` title Task 1 already ships on every platform — but
  that misses a story criterion, so it is an escalation and not a quiet substitution.
- **Task 4 — low certainty.** No window hook and no quit veto exists in this repository; the
  design rests on framework source that has been read but not run.
- **Task 3 — high blast radius.** The mapping from a dialog button to an outcome is what decides
  whether the user's file is written or their edits are dropped.
- **Task 4 — high blast radius.** Same, plus the risk of a window that cannot be closed at all.

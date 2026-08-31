# US-005: Open a model by dropping it on the window

## Contents
1. Task 1: Route every dropped file through the host-opened-file delivery
2. Task 2: Carry a native drop's real paths from the shell to the viewer
3. Task 3: Paint the native drop target with the viewer's own drop overlay
4. Task 4: Document opening a model by dropping it

## Story Reference

`user-stories/emod-desktop.md` — **US-005: Open a model by dropping it on the window**.
Six criteria, referred to below as C1–C6 in the order the story lists them:

- **C1** Dropping a `.emod` or `.json` file renders it and makes it the current file, exactly as opening it through the dialog would
- **C2** Save then writes back to the dropped file's real location with no dialog
- **C3** Dropping a file with any other extension reports `Only .emod and .json files are supported` and leaves the current model on screen
- **C4** Dropping while unsaved changes exist goes through the unsaved-changes prompt
- **C5** Dropping several files at once opens the first supported one and names the file it opened
- **C6** The drop-target highlight behaves as it does in the browser viewer

Depends on US-004, delivered and on `main`.

## Boundaries

**Out of scope:**
- Automated tests driving the desktop window. The shell is verified by manual smoke test, backed by the existing Go and browser suites (story non-goal, carried forward from US-004).
- Anything that would make the CLI (`emod diagram --serve`) or the hosted web viewer behave differently, or that would fork the frontend: all three distributions keep loading one copy of `internal/frontend/static/` (story non-goal).
- Widening the **browser** viewer's own drop region. It keeps accepting drops only on the data panel body, which is where its highlight has always been; the desktop window accepts them anywhere, because the panel is off screen whenever it is collapsed (see Codebase Context). The two therefore differ in *region* while matching in appearance and lifecycle — the reading of C6 this breakdown takes.
- Windows and Linux. The Wails runtime resolves dropped paths on all three platforms and nothing here is macOS-specific, but only macOS is smoke-tested, per the repo's convention and the learning that a desktop feature verified on a Mac may not exist elsewhere.
- Opening more than one of several dropped files — no tabs, no second window.
- Launching the app with a file argument, and a recent-files list. `initialState()` still answers `null` on desktop.

**Deferred:** nothing from this story. C1–C6 are all closed here.

## Codebase Context

**The platform seam.** `internal/frontend/static/platform.js` re-exports thirteen names from
`platform.browser.js`; `task build:desktop` (Taskfile.yml:48-65) copies `static/*` into
`cmd/emod-desktop/frontend/static/` and overwrites `platform.js` with
`internal/frontend/desktop/platform.desktop.js`. The seam is written three times and pinned by a
literal list of the thirteen names in `internal/frontend/tests/viewer.test.js:2016-2033`, so any
change to it moves four files at once.

**Which way each operation travels.** The viewer *calls out* for anything it initiates; anything the
*host* initiates arrives at a handler the viewer registered — `onFileOpened`, `onSaveRequested`,
`onLeaveRequested`. In the browser those three register a handler that is never called, deliberately.
A native drop is host-initiated, so it belongs on that half of the seam; `droppedFile(dataTransfer)`
is on the other half and cannot serve it.

**Why the DOM drop stops working on desktop.** With `EnableFileDrop` on, macOS installs a
`WebviewDrag` NSView over the webview registered for `NSFilenamesPboardType`
(`webview_window_darwin.go:155-166`, `webview_window_darwin_drag.m`). That view is the
`NSDraggingDestination`, so the WKWebView never sees a DOM `drop` carrying `dataTransfer.files` —
today's FileReader path goes dead the moment the flag is on. Its `hitTest:` returns `nil` with an
explicit comment, so ordinary mouse events still reach the canvas and node dragging and panning are
unaffected. Linux takes the same native route. **Windows** keeps DOM drop events and resolves paths
through `chrome.webview.postMessageWithAdditionalObjects`, which means a desktop build that both
reads `dataTransfer.files` *and* subscribes to the shell's drop would open the file twice there.

**The drop path through the framework.** A native drop reaches the runtime's platform-file-drop
handler, which finds the element under the drop point and walks up to the nearest ancestor carrying the
attribute `data-file-drop-target`; with no such ancestor the drop is **silently ignored**
(`window.ts:638-673`). With one, the payload goes back to Go, which dispatches
`events.Common.WindowFilesDropped` to the listener map `OnWindowEvent` appends to — **not** the hook map
`RegisterHook` fills, which is what main.go's close veto uses. The window event's context answers the
dropped names (`application.WindowEventContext.DroppedFiles`). Hover feedback comes from three drag
callbacks the runtime exposes to Go, which toggle the class `file-drop-target-active` on that same
element.

**Where the drop target has to be.** The data panel starts collapsed in the markup, and the collapsed
rule (viewer.html:498-500) translates it down far enough that only its 40px header stays on screen;
`renderPanelSource` re-collapses it after every successful render
(viewer.js:215). So `#data-panel-body` — where the browser's drop listeners and its
`.drag-over::after` overlay live — is off screen for most of the app's life, and
`elementFromPoint` would never find the attribute there. A window-filling element has to carry it, or
C1, C2 and C4 all fail in the state a user is normally in.

**The single guarded entry point.** `openModel(text, file)` (viewer.js:286-293) is the only function
that hands a different file's model to `renderPanelSource`, pinned by a source scan in
`viewer.test.js:1834-1880`. It runs `guarded()` → `clearedToReplace()`, which is the US-004
unsaved-changes prompt. C4 is closed by arriving through it, and by nothing else. The
`onFileOpened` handler (viewer.js:360-385) wraps it with the failure envelope, the
`<name> is empty` check and the reveal of the collapsed panel — the behaviour C1 says a drop must
match "exactly".

**Cross-boundary guards that constrain the commits.** `internal/desktop/event_names_test.go` requires
the set of `EmitEvent("…")` names in `cmd/emod-desktop/main.go` to **equal** the set of
`Events.On('…')` names in `platform.desktop.js` — so a new event's two halves cannot be split across
commits. `binding_names_test.go` holds method names, service registrations and the JSON keys of
`openedFile`/`savedFile`; `FileService.Read` already answers `{name, path, content}` or
`{error: …}`, so nothing new is needed on the Go service.

**Two intended consequences of keeping one copy of the drop policy** (called out so they are not read
as scope breaches): once the browser and the desktop feed one routine, the browser viewer also gains
"open the first supported of several dropped files" and the `<name> is empty` message that the dialog
route already gives. Both follow from C1 and C5 being satisfied without a second copy of the policy
inside the desktop adapter.

## Tasks

### Task 1: Route every dropped file through the host-opened-file delivery

**Behavior:** A drop is decided in one place, whatever brought it in. The viewer registers for files a
host pushes at it, the same way it registers for a file the host's Open dialog produced, and the
browser's own DOM drop feeds the same routine. That routine looks over the dropped files, opens the
first whose name ends `.emod` or `.json`, says `Only .emod and .json files are supported` when none
does, reads the chosen one, and hands it to the same delivery `onFileOpened` uses — so a dropped file
carrying a path becomes the current file and Save's target exactly as a file chosen in the dialog
does. In the browser nothing pushes, and the path a drop carries there is still empty.

**Acceptance Criteria:**
- [x] The seam names a registration through which a host delivers dropped files, and its contract, its
      browser implementation and its desktop implementation all carry it — the contract guard in
      `viewer.test.js` names it and both implementations satisfy the contract.
- [x] The seam's DOM-drop reader answers every file a drop carried rather than only the first, and
      both implementations answer the same shape for the same drop.
- [x] A file delivered through the registration with a name ending `.emod` or `.json` renders, and the
      window title and the file shown in the bottom bar name it — for a delivery carrying a real path,
      the bottom bar shows that path and a following Save writes to it with no location dialog. (C1, C2)
- [x] A drop whose files include none ending `.emod` or `.json` reports exactly
      `Only .emod and .json files are supported`, leaves the diagram and the source panel as they were,
      and does not leave that message in a collapsed panel. (C3)
- [x] A drop carrying several files opens the first one ending `.emod` or `.json` — including when
      files before it in the drop are unsupported — and the window title and bottom bar name that file
      and no other. (C5)
- [x] A drop arriving while the model on screen has unsaved changes puts the unsaved-changes question
      once, and answering Cancel leaves the model, the panel text and the open file exactly as they
      were. (C4)
- [x] A dropped file the host cannot read reports the host's own reason, and leaves the diagram on
      screen; a dropped file with nothing in it reports that it is empty, as the dialog route does. (C1)
- [x] `renderPanelSource` is still handed a model to open from exactly one call — the existing source
      scan still passes unedited.
- [x] The browser viewer's DOM drop still opens a `.emod` and a `.json` file and still reports a path
      of nothing for them.

**Affected Files/Modules:**
- `internal/frontend/static/platform.js` — the contract's export block gains the registration and the renamed reader
- `internal/frontend/static/platform.browser.js` — the reader answers a list; the registration is accepted and never called, as `onFileOpened` is
- `internal/frontend/desktop/platform.desktop.js` — same two changes, so the contract guard passes; the desktop reader still reads the DOM drop at this point
- `internal/frontend/static/viewer.js` — one routine behind both entry points; the `onFileOpened` handler's body becomes the delivery both use
- `internal/frontend/tests/viewer.test.js` — the seam mock, the contract's literal name list, and the drop leaves
- `internal/frontend/tests/dropped-file.test.js` — the reader's list shape, on both implementations

**Patterns to Follow:**
- `internal/frontend/static/viewer.js:360-385` — the delivery a host-opened file goes through, which a drop must now share
- `internal/frontend/static/viewer.js:286-293` — the single guarded entry point
- `internal/frontend/static/viewer.js:326-358` — the DOM drop listeners and the message being preserved
- `internal/frontend/static/platform.browser.js` — the "registers a handler that stays uncalled rather than being unfinished" comment and shape
- `internal/frontend/tests/viewer.test.js:1998-2033` — the seam-contract guard, whose literal list must move in this same change
- `internal/frontend/tests/viewer.test.js:1834-1880` — the scan that pins one entry point
- `internal/frontend/tests/viewer.test.js:481-540` — the drop leaves to extend
- `tasks/learnings.md` — "A criterion naming an entry point a later story delivers is closed by the seam, not by the branch"; "In this viewer, 'the status area' is the bottom bar, not the element named for it"; "A jsdom assertion on text content cannot see that the text is off screen"; "The panel's text and the open file's identity are committed at different moments"; "Two concurrent dynamic imports race the mock; one flush between them does not"; "A resolves assertion must await the promise the code produced"

**Testable:** Yes — every criterion is reachable through the page: the DOM drop through a `drop` event, the pushed delivery through the handler the mocked seam captured, exactly as `deliverFile` already drives `onFileOpened`.

**Certainty:** medium — the registration half of the seam has three precedents (`onFileOpened`, `onSaveRequested`, `onLeaveRequested`, all in `platform.browser.js` and `platform.desktop.js`), but none of them hands the viewer a *list* to choose from, and none feeds the same delivery a second entry point already uses.

**Blast radius:** low — nothing this task can produce reaches a user's file: every path a drop carries is still empty until a host supplies one in Task 2.

**Verification:** `task test:viewer`

**Depends on:** None

---

### Task 2: Carry a native drop's real paths from the shell to the viewer

**Behavior:** Dragging files from the file manager onto the desktop window opens the first supported
one from its real location on disk. The shell accepts native file drops, marks the window as the drop
target so the framework does not discard them, and tells the frontend which paths arrived; the desktop
platform module turns those paths into files the viewer can name and read through
`desktop.FileService`, and hands them to the registration Task 1 added. Because the file carries its
own path, Save writes back there with no dialog. The desktop build stops reading dropped files out of
the DOM, since a native drop no longer arrives that way and, on Windows, doing both would open the
same file twice.

**Acceptance Criteria:**
- [ ] The window is built with native file drop enabled and a window-filling element carries the
      attribute the framework's drop handler looks for — so a drop lands wherever it is released in the
      window, including while the data panel is collapsed. The attribute is inert in the CLI and web
      builds, which load the same markup.
- [ ] The shell listens for the framework's files-dropped window event and emits one event to the
      frontend carrying the dropped paths; the frontend subscribes to that name. `task test:unit`
      passes, which is what proves the two names are one set.
- [ ] Delivering that event with a path answers the viewer a file whose name is the path's base name,
      whose path is the path, and whose contents come from `FileService.Read` — driven through the
      subscription the shell would fire, not by calling an exported helper.
- [ ] A delivered drop whose read the service refuses reports the service's own reason rather than a
      generic one.
- [ ] A DOM drop event on the desktop build yields no file, so a build that receives both a native drop
      and a DOM drop opens the file once.
- [ ] With `task build:desktop` built and `./bin/emod-desktop` running: dragging a `.emod` file from
      the file manager onto the window renders it, the window title and the bottom bar name it and show
      its real path, and File ▸ Save then writes back to that file with no dialog and confirms in the
      bottom bar. (C1, C2)
- [ ] In the same running app: dropping a file whose name ends in neither extension reports
      `Only .emod and .json files are supported` and leaves the model on screen (C3); dropping several
      files at once opens the first supported one and names it (C5); dropping onto a window holding
      unsaved changes raises the unsaved-changes dialog and honours all three answers (C4).
- [ ] In the same running app: dragging a node and panning the canvas still work with file drop enabled.

**Affected Files/Modules:**
- `cmd/emod-desktop/main.go` — the window option, and the listener that turns the framework's window event into the frontend event
- `internal/frontend/static/viewer.html` — the drop-target attribute on a window-filling element
- `internal/frontend/desktop/platform.desktop.js` — the subscription, the files it answers the viewer, and the DOM reader that now answers nothing
- `internal/frontend/tests/platform.desktop.test.js` — the drop subscription's leaves
- `internal/frontend/tests/wails-runtime-stub.js` — if a host event carrying data needs a shape the stub does not yet produce
- `internal/frontend/tests/dropped-file.test.js` — the two implementations no longer answer the same thing, so the shared cases split

**Patterns to Follow:**
- `cmd/emod-desktop/main.go:100-125` — the window options and where a listener on the window is registered; note that the files-dropped event is dispatched to `OnWindowEvent` listeners, not to the `RegisterHook` hooks the close veto uses
- `internal/frontend/desktop/platform.desktop.js` — the Open-requested subscription, the read it performs and the delivery it makes; the closest existing shape for this whole path, and the module the new subscription joins
- `internal/frontend/tests/platform.desktop.test.js:296-405` — the "opening a file" leaves, driven through `runtime.listeners[...]`
- `internal/desktop/event_names_test.go` — the set-equality the two halves must satisfy in this one change
- `internal/desktop/file_service.go` — the `openedFile` envelope and its `{"error": …}` failure shape
- `internal/frontend/static/viewer.html:1183-1195` — the panel markup, and the reason the attribute cannot live on the panel body
- `tasks/learnings.md` — "A parity guard is vacuous until the far side of the boundary calls it, so it belongs to that task"; "A low-certainty Wails shell task is routine once the framework API is read first"; "Wails adopts an application menu into a window only on UseApplicationMenu, and only Windows needs it"; "A vitest gate on a binding must still be in the answer bag when the deferred dispatch runs"; "Look at the rendered page: it finds what a green viewer suite cannot"; "A go list pipeline is emptied by the very package it filters"
- `task:1`

**Testable:** Yes — the frontend half through `internal/frontend/tests/platform.desktop.test.js`, driven at the subscription the shell fires; the shell half by the event-name guard in `task test:unit` and by manual smoke test of the built app, which is the story's stated approach for the window.

**Certainty:** medium — the shell-emits-an-event-the-desktop-module-subscribes-to path exists four times over for the File menu, but `file:dropped` would be the first event in this repo carrying a payload rather than being a bare signal, and the first listener registered with `OnWindowEvent` rather than `RegisterHook`.

**Blast radius:** high — destructive write: this is the change that gives the model on screen a real path obtained from a drag gesture, and Save then overwrites that path with no dialog and no confirmation. Pairing the wrong dropped file's path with the shown model's text would silently destroy a file the user never opened.

**Verification:** `task test:viewer`; `task test:unit`; then `task build:desktop && ./bin/emod-desktop` and work through the drop scenarios above by hand.

**Depends on:** Task 1

---

### Task 3: Paint the native drop target with the viewer's own drop overlay

**Behavior:** Dragging a file over the desktop window shows the same drop affordance the browser viewer
shows over its data panel — the dashed border, the tint and the words `Drop .emod or .json file here` —
and it goes away when the drag leaves the window or the file is released. The framework marks the drop
target with its own class while a file is over it; the stylesheet answers that class with the overlay
the viewer already has.

**Acceptance Criteria:**
- [ ] The shared stylesheet paints the framework's active-drop-target class on the drop-target element
      with the same overlay text and the same border, tint and colour the existing drag-over rule uses;
      a leaf reads both rules out of `viewer.html` and requires them to agree, and fails if either
      drifts. (C6)
- [ ] The overlay sits above the panels and the canvas and takes no pointer events, so it cannot
      intercept the drop it is announcing.
- [ ] The browser viewer's own drag-over overlay is unchanged — its rule, its wording and the element it
      paints on are the same as before this task.
- [ ] With `./bin/emod-desktop` running: dragging a file over the window shows the overlay, moving the
      drag out of the window clears it, and releasing the file clears it and opens the model. (C6)

**Affected Files/Modules:**
- `internal/frontend/static/viewer.html` — the rule for the framework's active class, beside the existing drag-over rule
- `internal/frontend/tests/viewer.test.js` — the leaf pinning the two rules against each other

**Patterns to Follow:**
- `internal/frontend/static/viewer.html:522-544` — the `#data-panel-body.drag-over::after` overlay this reproduces
- `internal/frontend/tests/legend.test.js:8` — reading `viewer.html` as text in a leaf and asserting on what it declares
- `tasks/learnings.md` — "A viewer leaf must be able to fail only for the paint its name blames"; "Look at the rendered page: it finds what a green viewer suite cannot"; "text-overflow is inert on a flex container, and #stats span outranks a bare id selector"

**Testable:** Yes — the rules are read out of the stylesheet by a vitest leaf; what the overlay actually looks like is confirmed by looking at the running app, which is what the repo's own learning says a green viewer suite cannot do.

**Certainty:** high — the overlay already exists at `internal/frontend/static/viewer.html:528-543`; this is that rule applied to a second selector, and the class the framework sets is fixed in the runtime source.

**Blast radius:** low — a stylesheet rule for a class no other build sets; nothing it can be wrong about reaches a file, a permission or a contract outside this repository.

**Verification:** `task test:viewer`; then `task build:desktop && ./bin/emod-desktop` and drag a file over the window, out of it, and onto it.

**Depends on:** Task 2

---

### Task 4: Document opening a model by dropping it

**Behavior:** The README's desktop section describes dropping a model on the window the way it already
describes File ▸ Open and File ▸ Save, and no longer lists as a known gap the very thing this work
delivers. The architecture document's account of the platform seam names the drop as a host-initiated
delivery rather than as something the viewer reads out of a DOM event, and its count of the guards
holding the language boundary together still matches what `internal/desktop` contains.

**Acceptance Criteria:**
- [ ] The README's desktop section says a `.emod` or `.json` file dropped on the window opens from where
      it lives and that Save then writes back there with no dialog, and states what happens to a drop of
      an unsupported file, a drop of several files, and a drop over unsaved changes.
- [ ] The README no longer states that a dropped file cannot be saved back to where it came from; every
      remaining entry in its "what it does not do yet" list is still true of the working tree.
- [ ] The architecture document's platform-seam section describes the drop as arriving at a handler the
      viewer registered, alongside the file the host opened, rather than as a dropped file the viewer
      reads; and its statement of how many guards hold the language boundary together agrees with the
      number of guard functions in `internal/desktop`.
- [ ] Every ```emod fence added, if any, is a model `oracle.Check` reports nothing about — `task test:unit`
      passes.

**Affected Files/Modules:**
- `README.md` — the desktop section and its list of what the app does not do yet
- `docs/architecture.md` — the platform-seam section and the guards paragraph

**Patterns to Follow:**
- `README.md` — the File ▸ Open and File ▸ Save paragraphs in the desktop section, and the "What it does not do yet" list that carries the sentence this work retires
- `docs/architecture.md` — the "The viewer, and its three distributions" section, its list of what the seam provides, and the paragraph counting the guards in `internal/desktop`
- `tasks/learnings.md` — "An ```emod fence is a promise that the block validates"; "A criterion may name an artefact the repository does not have"; "`docs/dsl-reference.md` is the one keyword surface no test reaches, and a retirement story forgets it"

**Testable:** No — prose. The one executable consequence, that any added `emod` fence validates, is covered by the existing oracle leaf in `task test:unit`.

**Certainty:** high — the precedent is the desktop section of `README.md` and the seam sections of `docs/architecture.md`, both written for US-004 and both naming the sentence this task edits.

**Blast radius:** low — documentation; nothing it changes is executed.

**Verification:** `task test:unit`; read the edited sections against the built app's behaviour.

**Depends on:** Task 3

## Summary

**Four tasks.** The ordering is dependency-first, and the dependencies are real rather than tidy: the
viewer cannot be given a real path until something supplies one (Task 1 before Task 2), the framework
will not put its active-drop-target class anywhere until an element carries the attribute (Task 2 before
Task 3), and the documentation should describe what the built app does (Task 3 before Task 4).

Within that, the folding is by mechanism rather than by file, because a story threading one operation
through a shared seam makes every task touch the same four or five files. Task 1 is *the policy* — which
file of several is opened, what is said when none can be, and the single guarded entry point it all
arrives through. Task 2 is *the delivery* — the native drop, its paths, and the read that turns one into
a file. Task 3 is *the affordance*. Two cross-boundary guards force parts of Task 2 to be one commit:
the event-name guard requires the shell's emitted names and the frontend's subscribed names to be one
set, and the seam-contract guard requires the contract and both implementations to move together, which
is why the seam edit sits wholly inside Task 1.

**Criteria coverage.** C3, C4 and C5 are closed by Task 1 and confirmed end to end in Task 2; C1 and C2
need both, since the viewer half is inert until the shell supplies a path; C6 is closed by Task 3.
Nothing is deferred.

**The assessments are not uniform, and two are worth arguing with.** Task 2 is the only `high` blast
radius — it is what makes a drag gesture produce a path that Save overwrites without asking. Tasks 3
and 4 are genuinely routine `high` certainty: an existing CSS rule applied to a second selector, and
two document sections that already carry the sentence being retired. Nothing here is `low` certainty,
and that is a claim rather than an omission: the framework's file-drop source was read before this
breakdown was written, which is what the repo's own learning says converts a Wails shell task from
undecided to routine — the option name, the listener map, the attribute, the class, the payload
accessor and the mouse-event pass-through were all confirmed in the module cache, not assumed.

**The decision most worth overruling** is where the drop target lives. C6 says the highlight should
behave as it does in the browser viewer, and in the browser the drop target is the data panel body. But
that element is translated off screen whenever the panel is collapsed, which it is after every
successful render — so copying the browser exactly would make C1, C2 and C4 fail in the state a user is
normally in, and the framework would discard those drops silently. This breakdown therefore reads C6 as
*the same overlay, appearing and clearing at the same moments*, and makes the whole desktop window the
drop region while leaving the browser's own region alone. The alternative — widening the browser's drop
region to match — was rejected as a change to the hosted web viewer the story's non-goals ask to leave
alone.

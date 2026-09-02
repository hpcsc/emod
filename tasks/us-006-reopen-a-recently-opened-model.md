# US-006: Reopen a recently opened model

## Contents
1. Task 1: Remember which models have been opened, across runs
2. Task 2: Record every model that becomes the open one
3. Task 3: Reopen a model from the File ▸ Open Recent menu
4. Task 4: Document reopening a recently opened model

## Story Reference

`user-stories/emod-desktop.md` — **US-006: Reopen a recently opened model**.
Six criteria, referred to below as C1–C6 in the order the story lists them:

- **C1** The app lists recently opened models in a menu, most recent first
- **C2** Selecting an entry opens that model exactly as the file dialog would, including
  becoming the save target
- **C3** The list survives quitting and relaunching the app
- **C4** The list holds at most 10 entries; reopening a listed file moves it to the top rather
  than duplicating it
- **C5** Selecting an entry whose file no longer exists reports that and removes the entry from
  the list
- **C6** The list can be cleared

Depends on US-002, delivered and on `main`, along with US-003, US-004 and US-005.

## Boundaries

**Out of scope:**
- Automated tests driving the desktop window. The shell is verified by manual smoke test on
  macOS, backed by the Go and vitest suites (story non-goal, carried forward from US-004 and
  US-005).
- Anything that would make the CLI (`emod diagram --serve`) or the hosted web viewer behave
  differently, or that would fork the frontend: all three distributions keep loading one copy of
  `internal/frontend/static/` (story non-goal). The browser viewer gains an inert seam
  operation and nothing else — it has no list, and a file it is handed by a drop carries no
  location to remember.
- Windows and Linux. Nothing here is macOS-specific — the framework rebuilds a menu the same way
  on each platform and `os.UserConfigDir` answers each platform's own directory — but only macOS
  is smoke-tested, per the repo's convention and the learning that a desktop feature verified on
  a Mac may not exist elsewhere. In particular **the menu refresh is verified on macOS only.**
- The other half of the story's Open Question 3. This run decides where *recent files* live;
  **window geometry and preferences** are not decided, not stored, and not designed for here.
- Pruning the list at startup, or watching listed files. C5 says an entry is removed when it is
  *selected* and found missing, so a file deleted behind the app's back stays on the menu until
  someone chooses it. Checking ten paths on every launch, and again whenever the menu opens, is
  the alternative and is deliberately not taken.
- Removing one entry from the menu without clearing the whole list; a "Reopen Last Closed"; a
  per-window list; the list appearing on the empty window's landing instructions; and the
  platform's own recent-documents integration (macOS `NSDocumentController`, the Dock's
  Recents). C6 is "the list can be cleared", not "entries can be managed".
- Launching the app with a file argument. `initialState()` still answers `null` on desktop, and
  a relaunch opens an empty window with a populated menu rather than reopening the last model.

**Deferred:** nothing from this story. C1–C6 are all closed here.

## Codebase Context

**Where the list lives — the decision this run makes.** The story's Open Question 3 offers "a
config location shared with the CLI, or a desktop-only one". The answer taken is neither
exactly: the platform's own per-user configuration directory, `os.UserConfigDir()` plus
`emod/recent-files.json` — `~/Library/Application Support/emod` on macOS, `~/.config/emod` on
Linux, `%AppData%\emod` on Windows. The repository's only existing per-user path is
`internal/llm/config/config.go:93-100`, which hard-codes `~/.config/emod` off `os.UserHomeDir`;
the two therefore **agree on Linux and differ on macOS**, which is the trade being made — a Mac
user's application state belongs where macOS keeps application state, not in a dotfile
directory the CLI invented. This is the single most overrulable decision in the breakdown.

**Where testable Go lives, and where it cannot.** `internal/desktop` imports no GUI framework
and links no CGO (`internal/desktop/service.go`), which is what keeps it in `task test:unit`,
`task test:race` and `task test:integration`. `cmd/emod-desktop` is in **none** of those package
sets — `Taskfile.yml:90` and `:104` both filter it out — so anything written there is verified
only by manual smoke test, or by a guard that reads it as source. Two such guards already exist:
`TestWindowMarkerImplementation` (`internal/desktop/window_service_test.go:155-182`) reads the
shell's `MarkEdited` body and requires `application.InvokeAsync` and forbids `InvokeSync`,
because blocking there deadlocks every later answer behind a write lock — a hang rather than a
failure, which reaches a user before it reaches a test. `RecentMenu.Show` carries the identical
requirement and earns the identical guard.

**The collaborator-behind-an-interface shape.** `desktop.WindowService`
(`internal/desktop/window_service.go`) is the precedent this story's service copies whole: an
interface the shell implements (`WindowMarker`), taken at construction rather than exposed, a
nil implementation accepted and doing nothing, the collaborator told *while the write lock is
held* so two answers cannot reach the window in the opposite order to the state they wrote, and
a documented must-not-block contract on the interface. Its tests carry the three shapes worth
copying: a recording double (`window_service_test.go:16-34`), a blocking double that proves the
lock is held across the call without `sync.Once` (`:125-153`), and a race-detector run
(`task test:race`, which exists for `./internal/desktop/...` alone).

**The file the service reads.** `desktop.FileService.Read`
(`internal/desktop/file_service.go:26-58`) resolves the path, reads the bytes, refuses anything
that is not valid UTF-8 — because `json.Marshal` would silently substitute U+FFFD and the next
Save would write that back — and answers `{name, path, content}` from the `openedFile` struct,
or `pipeline.ErrorJSON`'s `{"error": …}`. `TestServiceWireKeys`
(`internal/desktop/binding_names_test.go:170-190`) reads those `json:` tags **out of
`file_service.go` by name**, so `openedFile` has to stay declared there; a second method
answering the same envelope must share that struct and that read rather than restate either.

**The seam, and the guards that force parts of it into one commit.**
`internal/frontend/static/platform.js` re-exports fourteen names from `platform.browser.js`;
`task build:desktop` (`Taskfile.yml:47-64`) copies `static/*` into
`cmd/emod-desktop/frontend/static/` and overwrites `platform.js` with
`internal/frontend/desktop/platform.desktop.js`. The seam is written three times and pinned by a
literal name list in `internal/frontend/tests/viewer.test.js:2264-2295`, so an operation added to
it moves four files at once. Three further guards in `internal/desktop` hold the language
boundary: `TestShellEventNames` requires the set of `EmitEvent("…")` names in
`cmd/emod-desktop/main.go` to **equal** the set of `Events.On('…')` names in
`platform.desktop.js`; `TestBindingNames` requires, for **every** service registered in
`main.go`, a non-empty set of methods called on it from `platform.desktop.js`; and
`TestServiceRegistrations` requires every service the frontend imports to be registered. The
`TestBindingNames` floor is what fixes this breakdown's order: registering `RecentFiles` in
`main.go` with no JavaScript caller **fails `task test:unit`**, so the registration and its first
caller are one commit.

**Which way each operation travels.** The viewer calls out for anything it initiates; anything
the *host* initiates arrives at a handler the viewer registered — `onFileOpened`,
`onFilesDropped`, `onSaveRequested`, `onLeaveRequested`. Saying "this file is now the model on
screen" is viewer-initiated, so it is a call out, in the shape of `setWindowTitle` and
`setWindowModified`. Choosing a menu entry is host-initiated, so it arrives the way
`file:open-requested` does.

**The two moments a file becomes the open one.** `renderPanelSource`
(`internal/frontend/static/viewer.js:206-244`) writes the panel's text immediately and commits
`store.currentFile` only once the parse resolves — deliberately, so a render the parse rejects
leaves the window naming the model still on screen. That resolved branch is the single place
every entry point's file is adopted, and it already calls a seam operation from there
(`reportModified` at `:228`). The second moment is `writeModel` (`:521-557`), where a Save As
adopts the location its dialog chose and calls `applyWindowTitle` — another seam operation — at
exactly that point. Both moments are guarded by `store.currentFile === openFile` and by the
render number, so the identity being adopted is known to describe the text on screen.

**The single guarded entry point, and US-004's unfinished criterion.** `openModel(text, file)`
(`viewer.js:294-300`) is the only function that hands a different file's model to
`renderPanelSource`, pinned by a source scan in `viewer.test.js`. US-004's fourth criterion —
"Opening another model — dialog, recent files, or drop — with unsaved changes goes through the
same prompt" — named this story's entry point before it existed, and was closed by putting the
guard on that one function so a later entry point would have nowhere unguarded to arrive. A
recent entry arriving through `deliverFile` → `onFileOpened` → `openDeliveredFile` → `openModel`
is what makes that criterion true rather than merely unfalsified.

**One counter per gesture, claimed where the gesture is seen.** `latestGesture`
(`platform.desktop.js:41-50`) numbers an Open and a drop together, because an Open resolves a
picker and a read before it can deliver while a drop delivers the moment the shell names the
paths. A recent entry is a third gesture of the same family — it resolves a read before it can
deliver — and it claims the same counter, in the same module, or an Open requested first lands
on top of a menu choice made after it.

**The Wails menu API, read before this breakdown was written.**
`application.NewSubmenu(label, items)` (`pkg/application/menu.go:244`) makes a submenu item from
a `*Menu`; `NewMenuFromItems` (`:236`) and `Menu.Prepend` (`:228`) are how `main.go:56-58`
already puts Open/Save/Save As into File. `Menu.Update()` (`:68-79`) returns immediately unless
the application is running — so a refresh made while building the app populates the Go-side
items and nothing else — and on macOS it reaches `macosMenu.update()`, which wraps the whole
rebuild in `InvokeSync`. `dispatchOnMainThread` runs the work inline when the caller is already
on the main thread, so an `Update()` issued from inside an `application.InvokeAsync` callback
does not deadlock. The rebuild clears and re-walks the **root** menu's item tree — a submenu has
no impl of its own, so telling the submenu is not enough. Menu click callbacks run on their own
goroutine (`menuitem.go:243-270`, `go func()`), never on the main thread, which is what lets a
click call into a locking Go service.

## Theory

Two facts have to be true at once, and each is what makes the other trustworthy: the list has to
be one thing, and the menu has to show what the list is. So the list is a Go service that owns
both — every change moves the list, writes it to disk and tells the menu, all under the same
lock, so no order can be displayed that the service is not in and no order can be written that
the display disagrees with. The shell implements `RecentMenu` the way it already implements
`WindowMarker`, and the same must-not-block contract applies for the same reason: the shell's
half is told while the lock is held, so waiting on the UI thread there would wedge every later
change behind it. That is a hang, not a failure, so it earns the same source-scan guard the
window marker has.

Recording and reopening are separate mechanisms and are separate tasks. Recording is
viewer-initiated: the viewer alone knows the moment a file *becomes* the model on screen, and
that moment is a single branch in `renderPanelSource` that every entry point already passes
through, plus the Save As that adopts a new location. So it is a seam operation shaped exactly
like `setWindowModified` — inert in the browser, serialised through a promise queue on the
desktop — and it lands before the menu exists, because a list that fills itself is what makes
the menu worth building and its manual verification honest.

Reopening is host-initiated and arrives as an event, because the menu is native. The whole of
that task's care is that a menu entry must open **exactly** as the picker does: the same read,
the same envelope, the same delivery, the same gesture counter, and therefore the same
unsaved-changes question and the same save target. Nothing in the frontend may learn that a file
came from the menu rather than the dialog — a second copy of the open policy is the failure to
avoid, and the source scan pinning one entry point is what refuses it.

Check the ordering and the lock hardest. A menu that lags the list is the visible defect; a
`Show` that blocks is the invisible one, because it shows up as a frozen window rather than a red
test.

## Tasks

### Task 1: Remember which models have been opened, across runs

**Behavior:** A Go service holds the models that have been opened, newest first, never more than
ten and never one path twice, and puts the list back on disk on every change so a later run
finds it as it was left. It answers what a listed file holds, exactly as the file dialog's read
does; it forgets an entry whose file has gone and says so; and it empties itself on request.
Whatever is showing the list is told on every change, while the service holds its own lock, so
nothing can display an order the service is not in.

**Acceptance Criteria:**
- [ ] A service built over a path where no file exists starts holding nothing, and tells its
      display so before it answers anything else — a display built this way is populated before
      the app is running rather than after the first change. (C1)
- [ ] Recording a path leaves it at the top of the list, and a service built afterwards over the
      same path answers the same list in the same order. (C1, C3)
- [ ] Recording a path the list already holds moves it to the top instead of adding a second
      entry; the list never holds more than ten, and recording an eleventh drops the one recorded
      longest ago. (C4)
- [ ] Entries are absolute, so the same file recorded under a relative and an absolute spelling
      is one entry rather than two. (C4)
- [ ] Every change — a recording, an open that forgot a missing entry, a clear — reaches the
      display carrying the list as it now stands; and a display that blocks holds the next change
      behind it rather than being overtaken by it, so the order displayed and the order held can
      never disagree. (C1)
- [ ] A service built with no display at all works unchanged, so a shell with no menu yet is a
      supported state rather than a crash. (C1)
- [ ] Opening a listed path answers the same document `FileService.Read` answers for the same
      file — the same base name, the same absolute path, the same bytes, and the same refusal for
      a file that is not valid UTF-8 — because one read serves both. (C2)
- [ ] Opening a listed path whose file is no longer there answers a reason saying the file is not
      there any more and has been taken off the list, leaves the list without that entry, tells
      the display, and leaves the shortened list on disk. (C5)
- [ ] Opening a listed path the filesystem refuses for any other reason answers that reason and
      leaves the entry where it was. (C5)
- [ ] Clearing leaves the list empty, tells the display, and a service built afterwards over the
      same path finds nothing. (C6)
- [ ] A file at the path that is not the document this writes — truncated, not JSON at all, or
      JSON of another shape — is started from as an empty list rather than refused, and the next
      change replaces it. (C3)
- [ ] A change persisted into a directory that is not there brings the directory into being
      rather than failing. (C3)
- [ ] A change whose write the filesystem refuses answers that failure, while the list and the
      display already hold the new order. (C1)
- [ ] `task test:race` passes: the frontend writes this over a binding while the shell reads it
      from the thread that builds the menu, so the two reach it from different goroutines by
      construction.

**Affected Files/Modules:**
- `internal/desktop/recent_files.go` — new: the service, the display interface it takes, and its constructor
- `internal/desktop/recent_files_test.go` — new: its umbrella, its doubles, and its race case
- `internal/desktop/file_service.go` — the read the two answers share, and the `openedFile` envelope that stays declared here

**Patterns to Follow:**
- `internal/desktop/window_service.go` — the whole shape: the collaborator interface taken at construction, the nil implementation accepted, the collaborator told under the write lock, and the must-not-block contract stated on the interface
- `internal/desktop/window_service_test.go:16-34` — the recording double
- `internal/desktop/window_service_test.go:125-153` — the blocking double that proves the lock spans the call, and why it must not use `sync.Once`
- `internal/desktop/window_service_test.go:71-122` — the concurrency umbrella and the overtaking case
- `internal/desktop/file_service.go:26-58` — `Read`, the UTF-8 refusal, and the `openedFile` envelope
- `internal/desktop/file_service.go:90-126` — the complete-then-replace write, available if the list file wants it
- `internal/desktop/file_service.go:228-242` — `failureReason`, which is how a reason names its path once
- `internal/desktop/file_service_test.go:103-229` — the read umbrella's grouping and its subtest wording
- `internal/desktop/binding_names_test.go:170-202` — the wire-key guard that reads `openedFile`'s tags out of `file_service.go`
- `internal/desktop/service.go` — the package's no-GUI-framework constraint, which is why the display is an interface
- `tasks/learnings.md` — "sync.Once cannot hold a lock open for a test: Do makes every caller wait for the first"; "json.Marshal rewrites invalid UTF-8 as U+FFFD, so a Go service cannot promise bytes verbatim over JSON"; "A Go symbol consumed only by generated JS bindings reads as dead code"; "clerk verify reports every Test function as dead code"; "A cleanup or ordering guard that asserts the request, not the landing, cannot fail"; "A `_test.go` file always carries the `Test…` umbrella for the name it wears"; "A second `require.Contains` on one message is often shadowed by the first"; "An assertion whose expected value comes from the code under test is the recurring review finding"; "Certainty tracks whether the mechanism is decided, not how unfamiliar the API is"

**Testable:** Yes — every criterion is reachable through the exported constructor and the three
exported methods, with a temporary directory for the path and a double for the display. This is
the exported-API caller pattern: the shell is the caller, and what it depends on is the ordering,
the cap, the uniqueness and the envelope.

**Certainty:** medium — `WindowService` is the precedent for all of it (collaborator interface
taken at construction, nil accepted, told under the lock, race-tested) and `FileService` for the
envelope and the reason wording, but no service in this package has ever persisted state to disk
and read it back, and none rewrites a file from inside the lock it is mutating state under.

**Blast radius:** low — the only file it writes is the app's own list, at a path its caller
supplies, and its tests supply a temporary one; the models it reads it does not change, and a
lost list is rebuilt by opening files again.

**Verification:** `task test:unit`; `task test:race`. Expect `clerk verify`'s dead-code check to
report the exported methods until Tasks 2 and 3 supply their callers across the language
boundary — the recorded judgement for a symbol whose only consumer is a generated binding.

**Depends on:** None

---

### Task 2: Record every model that becomes the open one

**Behavior:** Whatever puts a model on screen from a real location — the host's Open dialog, a
file dropped on the window, or a Save As that adopts a new one — the app remembers where that
file lives and writes the list back to disk, so the next run starts holding it. A model with no
location behind it is not remembered, and neither is a Save that wrote back to the file already
open. The browser viewer answers the same registration with nothing, exactly as it does for the
window marker, because it has no list and its drops carry no location.

**Acceptance Criteria:**
- [ ] The seam names an operation through which the viewer says which file it has adopted, and
      the contract, the browser implementation and the desktop implementation all carry it — the
      contract guard's literal name list moves in this same change and still passes. (C1)
- [ ] **Every** way a file carrying a real path becomes the model on screen remembers it, exactly
      once, and the recording sits where they converge rather than being repeated per entry point
      — so a way of opening a model added later has nowhere to arrive that does not remember it.
      Derive the entry points from the source rather than from this list: today they are every
      call reaching the branch of `renderPanelSource` that adopts a file, plus the point in
      `writeModel` where a save adopts a location the model did not already have. (C1)
- [ ] A model with no location behind it is not remembered, and neither is a save that wrote back
      to the location already open: what makes a file worth remembering is a non-empty path that
      the model on screen did not already carry. (C1)
- [ ] A file whose render the parse rejects is not remembered, because it never became the model
      on screen. (C1)
- [ ] The desktop implementation sends each remembered file to the service's recording method,
      and two files remembered close together **land** in the order they were remembered rather
      than the order their calls happen to finish. (C4)
- [ ] A recording the shell refuses is reported where the user can read it without expanding
      anything, does not reveal the source panel, and does not stop the model opening or replace
      what the render said. (C1)
- [ ] The browser implementation accepts the registration and does nothing, the way it does for
      the window marker and the file the host opened; the browser viewer's behaviour is otherwise
      unchanged.
- [ ] Nothing on the page keeps a record of what the shell has already been told. A model opened,
      then reopened after the window is reloaded, is recorded both times — the list belongs to the
      shell for the life of the process while the page's memory is reloaded with it, so a
      de-duplication of the kind the unsaved-work answer carries would skip a recording the shell
      never heard.
- [ ] `task test:unit` passes with the service registered in the shell and imported by the
      frontend — which is what proves every method the frontend calls is one Go exports, and that
      the registration has a caller at all.
- [ ] With `task build:desktop` built and `./bin/emod-desktop` running: opening three models
      through File ▸ Open and quitting leaves a file under the platform's own per-user
      configuration directory holding those three paths, newest first; relaunching and opening
      the first of them again leaves three entries with that one at the top; opening eleven
      distinct models leaves ten. (C3, C4)

**Affected Files/Modules:**
- `internal/frontend/static/platform.js` — the contract's export block gains the operation
- `internal/frontend/static/platform.browser.js` — the inert implementation
- `internal/frontend/desktop/platform.desktop.js` — the implementation, and the queue that serialises it
- `internal/frontend/static/viewer.js` — the two moments a file becomes the open one, and where a refused recording is reported
- `internal/frontend/tests/viewer.test.js` — the seam mock, the contract's literal name list, and the leaves for what is and is not remembered
- `internal/frontend/tests/platform.desktop.test.js` — the desktop implementation's leaves, including the landing order
- `internal/frontend/tests/bindings-stub.js` — the new service the desktop module reaches
- `cmd/emod-desktop/main.go` — the path the list is kept at, and the service registered with the app

**Patterns to Follow:**
- `internal/frontend/desktop/platform.desktop.js:143-202` — `setWindowModified`, `tellShell`, `sendToShell` and `shellQueue`: the operation shape this one copies, and the queue that makes the last answer the last written
- `internal/frontend/static/platform.browser.js:116-150` — the inert implementations and the comment shape that says "accepted and never called rather than unfinished"
- `internal/frontend/static/viewer.js:206-244` — `renderPanelSource`, and the resolved branch where the file is adopted and a seam operation is already called
- `internal/frontend/static/viewer.js:521-557` — `writeModel`, and the point a save adopts its target and calls a seam operation
- `internal/frontend/static/viewer.js:443-460` — `clearSaveConfirmation` and `reportSaveOutcome`, the bottom-bar surface a failure goes to
- `internal/frontend/tests/viewer.test.js:31-66` — the seam mock every leaf drives
- `internal/frontend/tests/viewer.test.js:2259-2295` — the seam-contract guard, whose literal list must move in this same change
- `internal/frontend/tests/bindings-stub.js:25-45` — the answer bag, and `landed` beside `calls`, which is the only order a test can hold the frontend to
- `internal/frontend/tests/platform.desktop.test.js:1-60` — the module's fixture: the answer bag reset, the handler reset, and `flush`
- `cmd/emod-desktop/main.go:63-70` — the service variable declared ahead of what reads it
- `cmd/emod-desktop/main.go:114-118` — a service registered after the window rather than in `Options.Services`
- `internal/llm/config/config.go:93-100` — the repository's existing per-user path, which this deliberately differs from on macOS
- `tasks/learnings.md` — "A vitest gate on a binding must still be in the answer bag when the deferred dispatch runs"; "A cleanup or ordering guard that asserts the request, not the landing, cannot fail"; "A refutation built on a model of the mechanism is weaker than a reproduction on the real one"; "The shell's copy of a fact outlives the page that reports it, so a decision the shell makes never trusts the page's record"; "In this viewer, 'the status area' is the bottom bar, not the element named for it"; "A jsdom assertion on text content cannot see that the text is off screen"; "The panel's text and the open file's identity are committed at different moments"; "A story that threads one operation through a shared seam makes every task touch the same files, so fold by mechanism not by file"; "A parity guard is vacuous until the far side of the boundary calls it, so it belongs to that task"; "Two concurrent dynamic imports race the mock; one flush between them does not"; "A resolves assertion must await the promise the code produced, not the helper's tail"
- `task:1`

**Testable:** Yes — the viewer half through the mocked seam in `viewer.test.js`, driven at the
handlers the mock captured exactly as the existing open and drop leaves are; the desktop half
through `platform.desktop.test.js` against the bindings stub, with the landing order read off
`landed`; the shell half by the binding-name and registration guards in `task test:unit`.

**Certainty:** high — `setWindowModified` is this operation end to end: declared in the contract,
inert in `platform.browser.js:123`, implemented in `platform.desktop.js:149-202` behind a
promise queue for exactly the same reason, and called from both of the moments this one needs —
`reportModified` inside `renderPanelSource`'s resolved parse (`viewer.js:228`) and
`applyWindowTitle` where a save adopts its target (`viewer.js:547`).

**Blast radius:** low — nothing here decides what a Save overwrites: the only thing it writes is
the app's own list, and nothing reads that list back until Task 3.

**Verification:** `task test:viewer`; `task test:unit`; then `task build:desktop &&
./bin/emod-desktop` and work through the recording scenarios above by hand, reading the file
under the configuration directory between runs.

**Depends on:** Task 1

---

### Task 3: Reopen a model from the File ▸ Open Recent menu

**Behavior:** File ▸ Open Recent lists the models the app remembers, newest first, and choosing
one opens it exactly as the file dialog does — through the same read, the same delivery and the
same unsaved-changes question, so the window takes its name, the bar shows its real path, and
Save writes back there with no dialog. An entry whose file has gone reports that and leaves the
menu. Clear Menu empties the list and is greyed out when there is nothing to clear. The menu is
rebuilt every time the list moves, so what it shows is never behind what the app remembers.

**Acceptance Criteria:**
- [ ] The File menu carries an Open Recent submenu between Open… and Save, holding one item per
      remembered model, newest first, then a separator, then a Clear Menu item that is greyed out
      while the list is empty. (C1, C6)
- [ ] Two remembered models whose files have the same name are told apart in the menu. (C1)
- [ ] Choosing an entry makes the shell emit an event naming that path, and the frontend
      subscribes to that name — `task test:unit` passes, which is what proves the two names are
      one set. (C2)
- [ ] The frontend reads the chosen path through the service and hands the answer to the same
      delivery a file chosen in the Open dialog goes through, so the model becomes the current
      file and the target of the next Save with no location dialog. (C2)
- [ ] A menu choice and an Open still resolving its own read are numbered together, so whichever
      the user made last is the one that lands, whichever read finishes last. (C2)
- [ ] Choosing an entry while the model on screen holds unsaved changes puts the unsaved-changes
      question once and honours all three answers — and `renderPanelSource` is still handed a
      model to open from exactly one call, so the existing source scan passes unedited. (C2)
- [ ] Choosing an entry whose file is no longer there reports the reason the service gave —
      naming the file as gone and saying it has been taken off the list — leaves the model on
      screen, and leaves the submenu without that entry. (C5)
- [ ] Choosing Clear Menu empties the list, and the submenu afterwards holds nothing but a greyed
      out Clear Menu. (C6)
- [ ] The shell's display of the list hands its rebuild off to the main thread rather than
      waiting on it, so a rebuild issued while the service holds its lock cannot wedge every later
      change behind it — asserted by reading the shell's source, the way the window marker's is.
      (C1)
- [ ] A request arriving before the viewer has registered is discarded rather than throwing, the
      way the other host requests are.
- [ ] With `task build:desktop` built and `./bin/emod-desktop` running: after opening three
      models, File ▸ Open Recent lists all three newest first; choosing the second renders it,
      names the window, shows its real path in the bar, and File ▸ Save then writes back to that
      file with no dialog and confirms in the bar; choosing it again leaves three entries with
      that one at the top; quitting and relaunching shows the same three in the same order.
      (C1, C2, C3, C4)
- [ ] In the same running app: deleting a listed file and then choosing it reports that it is no
      longer there and takes it off the menu, leaving the model on screen (C5); Clear Menu empties
      the menu, and it is still empty after a relaunch (C6); the rest of the File menu, and Edit,
      View and Window, still work after several rebuilds.

**Affected Files/Modules:**
- `cmd/emod-desktop/main.go` — the submenu built into File, the service constructed over it and registered, and the Clear Menu item
- `cmd/emod-desktop/recent_menu.go` — new: the shell's implementation of the display interface, in its own file the way the window marker is
- `internal/desktop/recent_files_test.go` — the guard that reads the shell's rebuild for the main-thread hand-off
- `internal/frontend/desktop/platform.desktop.js` — the subscription, the gesture number it claims, the read and the delivery
- `internal/frontend/tests/platform.desktop.test.js` — the subscription's leaves, driven through the listener the shell would fire
- `internal/frontend/tests/bindings-stub.js` — the service's reading method, if Task 2 did not already need it

**Patterns to Follow:**
- `cmd/emod-desktop/main.go:34-61` — `applicationMenu`: extending the framework's default menu, `NewMenuFromItems`, `Prepend`, `FindByLabel`, and the framework's own items for standard accelerators
- `cmd/emod-desktop/main.go:63-70` — the service variable declared ahead of the callbacks that read it, which is what resolves a Clear Menu item needing the service that needs the menu
- `cmd/emod-desktop/main.go:120-130` — a shell listener turning a framework event into a frontend one carrying a payload
- `cmd/emod-desktop/window_marker_darwin.go` — the shell collaborator in its own file, and `MarkEdited`'s `InvokeAsync` hand-off, which is the shape the rebuild takes
- `internal/desktop/window_service_test.go:155-182` — `TestWindowMarkerImplementation`, the source-scan guard this one copies, and the reason a hang needs one
- `internal/frontend/desktop/platform.desktop.js:329-365` — `promptForFile`: the gesture number claimed at the gesture, the read, the guarded delivery, and `deliverFile`
- `internal/frontend/desktop/platform.desktop.js:41-75` — `latestGesture` and why an Open and a second entry point share one counter
- `internal/frontend/tests/platform.desktop.test.js:297-411` — the "opening a file" leaves, driven through `runtime.listeners[…]`
- `internal/frontend/tests/platform.desktop.test.js:413-545` — the drop leaves, including the one that pins two gestures against each other and the one asserting the listener's own answer rather than the helper's tail
- `internal/desktop/event_names_test.go` — the set equality the two halves must satisfy in this one change
- `internal/desktop/binding_names_test.go:22-75` — the method and registration guards, which need no edit and must still pass
- `internal/frontend/tests/viewer.test.js:2097-2140` — the source scan pinning one entry point, which must pass unedited
- `tasks/learnings.md` — "Two entry points opening one model need one counter, claimed where the gesture is seen"; "A criterion naming an entry point a later story delivers is closed by the seam, not by the branch"; "A parity guard is vacuous until the far side of the boundary calls it, so it belongs to that task"; "A Wails Menu can adopt pre-built items through NewMenuFromItems"; "Wails adopts an application menu into a window only on UseApplicationMenu, and only Windows needs it"; "A low-certainty Wails shell task is routine once the framework API is read first"; "Certainty tracks whether the mechanism is decided, not how unfamiliar the API is"; "sync.Once cannot hold a lock open for a test"; "A resolves assertion must await the promise the code produced, not the helper's tail"; "An audit that cannot run the code grades every runtime finding plausible"; "Look at the rendered page: it finds what a green viewer suite cannot"; "clerk verify reports every Test function as dead code"; "A `_test.go` file always carries the `Test…` umbrella for the name it wears"
- `task:2`

**Testable:** Yes — the frontend half through `platform.desktop.test.js`, driven at the listener
the shell fires rather than by calling an exported function; the shell half by the event-name and
binding guards in `task test:unit` and by the source-scan guard on the rebuild; the menu itself
by manual smoke test of the built app, which is the story's stated approach for the window.

**Certainty:** medium — the shell-emits-an-event-carrying-a-payload-the-desktop-module-subscribes
to path exists whole for the File menu and for the native drop (`main.go:128-130`,
`platform.desktop.js:62`), and `promptForFile` is the read-and-deliver shape line for line; but
nothing in this repo has rebuilt a menu after the application started running, and no shell
callback here has run while a Go service held its lock.

**Blast radius:** high — this is the change that makes a menu click produce a model carrying a
real path, which the next Save overwrites with no dialog and no confirmation. Pairing the wrong
entry's path with the text on screen, or letting an older read land on top of a newer choice,
silently destroys a file the user did not choose to open.

**Verification:** `task test:viewer`; `task test:unit`; `task test:race`; then `task
build:desktop && ./bin/emod-desktop` and work through the menu scenarios above by hand.

**Depends on:** Task 2

---

### Task 4: Document reopening a recently opened model

**Behavior:** The README's desktop section describes Open Recent the way it already describes
File ▸ Open, File ▸ Save and dropping a file — what it lists, in what order, what choosing an
entry does, what happens to an entry whose file has gone, and where the list is kept between
runs. The architecture document names the new seam operation among the things the seam provides
and says which way it travels, and its account of the guards holding the language boundary
together still matches what `internal/desktop` contains.

**Acceptance Criteria:**
- [ ] The README's desktop section says that File ▸ Open Recent lists the models opened most
      recently, newest first; that choosing one opens it exactly as the picker does and makes it
      what Save writes to; that the list holds at most ten and moves a reopened model to the top
      rather than listing it twice; that it survives quitting and relaunching; that an entry whose
      file has gone says so and leaves the list; and that Clear Menu empties it.
- [ ] The README says where the list is kept, naming the platform's own per-user configuration
      directory rather than one platform's path presented as the answer.
- [ ] Every remaining entry in the README's "What it does not do yet" list is still true of the
      working tree.
- [ ] The architecture document's platform-seam section names the operation through which the
      viewer says which file it has adopted, alongside the others the seam provides, and says it
      travels out from the viewer rather than arriving at it; and its statement of how many guards
      hold the language boundary together agrees with the number of guard functions in
      `internal/desktop`.
- [ ] The architecture document says the shell keeps the remembered list rather than the frontend,
      and why — the same reason the unsaved-work answer lives there.
- [ ] Every ```emod fence added, if any, is a model `oracle.Check` reports nothing about — `task
      test:unit` passes.

**Affected Files/Modules:**
- `README.md` — the desktop section, and its list of what the app does not do yet
- `docs/architecture.md` — the platform-seam section and the guards paragraph

**Patterns to Follow:**
- `README.md` — the File ▸ Open, dropping-a-file and File ▸ Save paragraphs in the desktop section, and the "What it does not do yet" list
- `docs/architecture.md` — the "The viewer, and its three distributions" section, its list of what the seam provides, the paragraph on which way each operation travels, and the paragraph counting the guards in `internal/desktop`
- `tasks/learnings.md` — "An ```emod fence is a promise that the block validates"; "A criterion may name an artefact the repository does not have"; "A breakdown's blast-radius count goes stale when sibling stories land after it", which is why the guard count is phrased as a rule rather than a number. No recorded learning mentions `docs/architecture.md` at all, so its conventions have to be read out of the document itself.

**Testable:** No — prose. The one executable consequence, that any added `emod` fence validates,
is covered by the existing oracle leaf in `task test:unit`.

**Certainty:** high — the precedent is the desktop section of `README.md` and the seam sections
of `docs/architecture.md`, both written for US-004 and extended for US-005, and both already
carrying the sentences this task sits beside.

**Blast radius:** low — documentation; nothing it changes is executed.

**Verification:** `task test:unit`; read the edited sections against the built app's behaviour.

**Depends on:** Task 3

## Summary

**Four tasks**, ordered dependency-first, and the dependencies are mechanical rather than tidy.
Task 1 is the list itself, with no caller. Task 2 registers it with the shell and gives it its
first caller, because `TestBindingNames` fails on a registered service no JavaScript calls — so
the registration and its first call are one commit by construction. Task 3 adds the menu, the
event and the subscription together, because `TestShellEventNames` compares the two halves as one
set and a menu item with no destination is half an item. Task 4 describes what the built app
does.

**The fold is by mechanism, not by file**, as the repo's own learning about seam stories
requires: recording an open and reopening from a menu are two mechanisms, and every task in a
seam story touches the same four or five files anyway. Task 1 is *the list* — order, cap,
uniqueness, persistence, and the display told under the lock. Task 2 is *the recording* — the
seam operation, its two call sites, and the queue. Task 3 is *the menu* — the submenu, the event,
the read and the delivery. Recording lands before the menu on purpose: a list that fills itself
is what makes the menu's manual verification honest, rather than a list hand-written into a JSON
file in a schema the test then depends on.

**Criteria coverage.** C3 and C4 are closed by Tasks 1 and 2 and re-observed through the menu in
Task 3. C1, C2, C5 and C6 have their Go half in Task 1 and are closed in Task 3, which is also
where US-004's third entry point — "opening another model … recent files … goes through the same
prompt" — stops being a criterion closed only by the seam. Nothing is deferred.

**The assessments are not uniform, and two are worth arguing with.** Task 3 is the only `high`
blast radius: it is what makes a menu click produce a path that Save overwrites without asking.
Task 2 is `high` certainty and that is a claim, not an omission — `setWindowModified` is the same
operation end to end, declared in the contract, inert in the browser, queued on the desktop, and
called from the very two moments in `viewer.js` this one needs. Nothing here is `low` certainty,
for the reason the repo's own learning gives: the framework's menu source was read before this
breakdown was written — `Menu.Update`'s `InvokeSync` path, its no-op-before-`Run` guard, the fact
that the *root* menu is what has to be rebuilt because a submenu has no impl of its own, and the
goroutine a click callback runs on — so the shell task is decided rather than merely familiar.

**Three decisions worth overruling.**

*Where the list lives.* `os.UserConfigDir()` + `emod/recent-files.json` answers Open Question 3
with the platform's own location, which coincides with the CLI's `~/.config/emod` on Linux and
deliberately differs from it on macOS. The alternative — sharing the CLI's dotfile directory
everywhere — keeps one location per user across both binaries at the cost of putting application
state where macOS does not keep it. Nothing else in this story depends on which is chosen.

*Where the menu's labels are decided.* Telling two entries with the same file name apart is a
rule about strings, and the mechanism puts it in the shell's display implementation — which lives
in `cmd/emod-desktop`, a package no test target builds. Moving that rule into `internal/desktop`
would make it testable at the cost of giving the service a presentation concern it otherwise has
none of. The criterion is written against what the user can tell apart, so either answer
satisfies it; the trade is a manual-only check versus a widened service.

*When a missing file leaves the list.* C5 removes an entry when it is chosen and found gone, so
this breakdown checks nothing at startup and nothing when the menu opens. A list can therefore
name a file that is not there, and the user finds out by choosing it. Pruning on launch would
make the menu always true at the cost of ten filesystem calls per start and a list that silently
shrinks while a network volume is unmounted.

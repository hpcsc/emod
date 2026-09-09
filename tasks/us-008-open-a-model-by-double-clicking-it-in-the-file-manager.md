# US-008: Open a model by double-clicking it in the file manager

## Contents
1. Task 1: Declare the `.emod` type and its association in the bundle plist
2. Task 2: Open a model the file manager hands the app, running or not
3. Task 3: Give `.emod` files a document icon in Finder
4. Task 4: Document the file association in the README

## Story Reference

`user-stories/emod-desktop.md` — **US-008: Open a model by double-clicking it in the
file manager**. Depends on US-007 (macOS packaging) and US-004 (unsaved changes), both
delivered on `main`. Design detail in `docs/proposals/emod-desktop-proposal.md` §10
Phase 4.

## Boundaries

**Out of scope** (the story file's Non-Goals, carried verbatim, keeping the ones this
story can actually breach):
- **No code signing or notarization** on any platform — no Apple Developer account, no App Store, no Windows signing certificate. Users take a documented one-time step on first launch.
- **No auto-update.** New versions are downloaded manually.
- **No replacement of existing distributions.** `emod diagram --serve` and the hosted web viewer keep working unchanged; desktop is a third option, not a migration.
- **No multi-window, tabs, or project workspaces.** One model per window.
- **No automated tests driving the desktop window.** Manual smoke testing of the shell, backed by the existing Go and browser suites.
- **No telemetry or crash reporting.**

**Out of scope** (added by this decomposition):
- **No Linux `.desktop` file and no Windows registry step.** US-014 and US-016 own those, and neither platform is packaged by this repo today. The Go and JS sides of Task 2 are written platform-neutrally because that costs nothing — `events.Common.ApplicationOpenedWithFile` is mapped on darwin, linux-gtk3 and windows in the pinned module — but nothing here declares an association off macOS.
- **No second window and no second process.** One model per window is a story non-goal, so a file the OS hands a running app replaces the model in the window it already has.
- **No new module dependency for reading the plist.** The guard extends what `internal/desktop` already does — it reads tracked files, starts no subprocess, and runs on the ubuntu CI that `task test:unit` runs on. A plist library would be a dependency added for a test.
- **No `Assets.car` / Icon Composer icon set, and no `.dmg` or installer.** Task 3 adds one more `.icns` in `Contents/Resources`, the way US-007 added the first.
- **No change to the picker or drop filters.** `.emod` and `.json` are already what the Open dialog and the drop path accept; this story does not widen or narrow them.
- **No claim on the `.json` default.** emod offers itself for `.json` and never takes it — if the alternate rank is measured to move the default on the test machine, the `public.json` document-type entry is dropped rather than kept. The criterion wins over the convenience.

**Deferred:**
- Linux association → US-014. Windows association → US-016. Both bind to the same
  `events.Common.ApplicationOpenedWithFile` the shell listens for in Task 2, so they
  inherit the whole runtime half and add only their platform's declaration.
- Opening several selected files at once as several models → nothing; one model per
  window is a non-goal, and the newest request wins.

## Codebase Context

**Everything below was read or run during exploration, not inferred.**

**The shell.** `cmd/emod-desktop/main.go` is framework calls and adapters only. Its
`window.OnWindowEvent(events.Common.WindowFilesDropped, …)` listener (`main.go:159-161`)
is the exact shape Task 2 needs on the application side: hear a native event, hand what
it carries onward, decide nothing. `recent_menu.go` is the other precedent — a shell
adapter over `desktop.RecentSlots`, which holds the slot-to-path decision.

**The testable side.** `internal/desktop` links no GUI framework and is the only desktop
package in the `test:unit` set. `WindowService` + `WindowMarker`
(`internal/desktop/window_service.go`) is the established shape for state the shell reads
and the page writes: a mutex, a collaborator interface the shell implements, a fake in
the test.

**The event pump, read out of `github.com/wailsapp/wails/v3@v3.0.0-beta.9`:**
- `application:openFile:` (`pkg/application/application_darwin_delegate.m:14`) calls
  `HandleOpenFile` unconditionally — there is no `hasListeners` gate on this path, unlike
  the theme and sleep notifications beside it.
- `HandleOpenFile` (`application_darwin.go:805-815`) builds an event context with the
  filename and pushes onto `applicationEvents`, which is `make(chan *ApplicationEvent, 5)`
  (`pkg/application/events.go:40`).
- The reader goroutine for that channel starts inside `Run`'s startup
  (`application.go:691-695`). So a cold-launch open is **buffered, not lost and not a
  deadlock** — but the listener runs after `Run`, by which point the webview page has not
  loaded and holds no `Events.On` subscriptions. A `window.EmitEvent` for it reaches
  nobody.
- The path is read off the event as `event.Context().Filename()`
  (`pkg/application/context_application_event.go`, key `CONTEXT_FILENAME`).
- Subscribe with `app.Event.OnApplicationEvent(events.Common.ApplicationOpenedWithFile, …)`
  before `app.Run()`. Registering before `Run` leaves `app.impl` nil, so the native
  `registerListener` call is skipped — which does not matter here, because the delegate
  method does not consult it.

**The frontend seam.** `platform.desktop.js` already has the whole delivery chain a file
from the OS needs: `openNamedBy` claims `latestGesture` and hands what it read to
`deliverFile`, `deliverFile` calls the viewer's `onFileOpened` handler, and viewer.js's
`openDeliveredFile` → `openModel` → `guarded(clearedToReplace)` **is** the unsaved-changes
prompt. `viewer.test.js:2286-2330` pins that `openModel` is the only function handing a
different file's model to `renderPanelSource`, so a delivery routed through `onFileOpened`
gets criterion 3's prompt by construction and cannot lose it later.

`initialState()` is deliberately **not** the hook for this. Its result is consumed by
viewer.js's "Initial load" block (`viewer.js:713-745`) in the browser's injected-state
shape: a non-null answer goes to `Model.setModelData`, never touches `store.currentFile`,
and so never becomes a save target — which criterion 2 requires. The empty-state branch it
shares with `initialState()` returning null lives inside `#data-panel`, which a successful
render collapses (`#landing-instructions` is at `viewer.html:1212`, inside
`#data-panel-body`), so a model arriving over the landing state hides it by collapsing the
panel rather than needing its own cleanup.

**Measured LaunchServices baselines** (run on this machine, before any change):
- `mdls -name kMDItemContentType /tmp/x.emod` → `dyn.ah62d4rv4ge80n5ptqu`. Nothing
  declares the type today, so this string moving to a declared identifier is the
  discriminating read that the exported declaration took effect.
- `mdls -name kMDItemContentType /tmp/x.json` → `public.json`, a system type with 16
  claimants in `lsregister -dump`. This is the type criterion 5 protects.
- `lsregister` is present at
  `/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister`.
  `osascript` is refused from this harness; nothing here needs it.

**Guards that constrain the design, each read:**
- `internal/desktop/event_names_test.go` requires the set of `EmitEvent("…")` names across
  every non-test file in `cmd/emod-desktop` to equal the set of `Events.On('…')` names in
  `platform.desktop.js`. A new shell event lands with its subscription or not at all.
- `internal/desktop/binding_names_test.go` requires every service `platform.desktop.js`
  imports to be registered in `main.go`, and every method it calls to be exported by Go.
  Its `servicesRegisteredBy` scanner recognises both `application.NewService(&desktop.X{})`
  and a variable assigned from `desktop.NewX(`.
- `internal/desktop/bundle_plist_test.go`'s `plistString` helper (lines 68-79) pairs a key
  with the element **after** it, positionally, and reads top-level string values only. It
  cannot see anything inside `CFBundleDocumentTypes` or `UTExportedTypeDeclarations`, and
  a nested key it happened to match would be paired with whatever followed it.
- `internal/desktop/packaging_task_test.go` pins `codesign … --sign - ./bin/emod.app` as
  the last line of `package:desktop`, so any new packaging step lands above it. Its "the
  icon is derived from an image the repository tracks" subtest (lines 60-66) uses
  `FindStringSubmatch`, which reads **only the first** `generate icons` line — a second one
  added beside it is silently unguarded.
- `internal/frontend/tests/viewer.test.js:2453-2483` pins the platform seam's export list
  in all three implementations. Nothing in this story needs a new export, and adding one
  would mean editing that list.

**Test doubles that must grow with Task 2:** `internal/frontend/tests/bindings-stub.js`
(a new service and its answers) and `internal/frontend/tests/wails-runtime-stub.js` (its
`listeners` map is how a test fires a shell event).

## Tasks

### Task 1: Declare the `.emod` type and its association in the bundle plist

**Behavior:** `bin/emod.app` tells macOS what an `.emod` file is and that emod opens one.
After the bundle is registered, Finder reports `.emod` files as emod's own type rather
than as an anonymous `dyn.…`, offers emod as an opener for them, and offers emod for
`.json` files as one choice among the applications that already handle them — without
becoming their default.

Two declarations do this and they are different things. `UTExportedTypeDeclarations`
declares the type itself: an identifier under the bundle identifier, a description, what
it conforms to (`public.plain-text` — a model is text), and the filename extension that
tags it. `CFBundleDocumentTypes` declares what this app opens: one entry naming the
exported type at `LSHandlerRank` `Owner`, and one entry naming `public.json` at
`LSHandlerRank` `Alternate`. `Alternate` is the whole mechanism behind criterion 5 — it
says "offer me" rather than "give me".

The guard is a chain, not two independent facts: `CFBundleIdentifier` prefixes the
exported type's identifier, the exported type carries the `emod` extension tag, and the
document-type entry names that same identifier. Pinning each end to a third value does not
compose into a chain.

**Acceptance Criteria:**
- [x] `build/darwin/Info.plist` declares one exported type: an identifier under the value of `CFBundleIdentifier`, a human description, conformance to `public.plain-text`, and a filename-extension tag of `emod`
- [x] `build/darwin/Info.plist` declares a document type naming that exported type with `LSHandlerRank` `Owner`, and a second naming `public.json` with `LSHandlerRank` `Alternate`
- [x] `plutil -lint build/darwin/Info.plist` reports OK — a malformed plist makes the whole bundle refuse to launch, which is the failure mode this key set risks
- [x] `task package:desktop` succeeds and `codesign --verify --deep --strict bin/emod.app` still exits 0
- [x] After registering the built bundle with LaunchServices, `mdls -name kMDItemContentType` on an `.emod` file reports the declared identifier rather than the `dyn.ah62d4rv4ge80n5ptqu` recorded as this machine's baseline
- [x] After the same registration, Finder's **Open With** for an `.emod` file lists emod
- [x] The default application for `.json` is the same before and after registration, read the same way both times, and Finder's **Open With** for a `.json` file lists emod below that default
- [x] A guard in `internal/desktop` requires the three-hop chain above to agree — bundle identifier prefixes the exported identifier, the exported type tags the `emod` extension, and the document-type entry names that identifier — and requires the `public.json` entry to rank below `Owner` and `Default`
- [x] Deleting the `Alternate` value alone makes that guard fail; so does deleting `emod` from the extension tag alone; so does changing the exported identifier without changing the document-type entry (restore each after checking)
- [x] The guard reads tracked files only: it starts no subprocess, needs no macOS, and adds no module dependency — `task test:unit` passes on the ubuntu CI configuration
- [x] `task build`, `task build:wasm` and `task build:web` still succeed and produce what they did before

**Affected Files/Modules:**
- `build/darwin/Info.plist` — the exported type declaration and the two document types
- `internal/desktop/bundle_plist_test.go` — new subtests for the nested declarations, and the reader they need; the existing `plistString` helper reads top-level string keys only and must keep doing exactly that

**Patterns to Follow:**
- The guard's shape, umbrella and subtest sentences: `internal/desktop/bundle_plist_test.go:23-58`.
- What the existing reader can and cannot see, and why the pairing is positional: `internal/desktop/bundle_plist_test.go:60-79`.
- Pinning a value in one tracked file against a value in another: `internal/desktop/packaging_task_test.go:54-66`.
- The current key set the bundle already declares: `build/darwin/Info.plist`.

**Testable:** Yes — the guard witnesses that the plist's own declarations agree with each
other and with the bundle identifier. It cannot witness what LaunchServices does with them,
which the measured criteria above close.

**Certainty:** low — no declaration in this repository has ever reached LaunchServices, no
plist key here is nested, and whether `Alternate` genuinely leaves the `.json` default
alone is an OS behaviour nothing here has exercised.

**Blast radius:** high — an exported UTI and a document-type claim are contracts consumed
outside this repository: LaunchServices records them per machine, and a rank that is wrong
takes `.json` away from whatever the user already opens it with, which the story forbids
and which is not undone by editing the plist back.

**Verification:** `plutil -lint`; `task package:desktop`; register with
`lsregister -f ./bin/emod.app`, then `mdls -name kMDItemContentType` on a scratch `.emod`
and a scratch `.json`, and read the openers back from `lsregister -dump`. Record the
`.json` default before registering and compare after — the comparison is the criterion, not
the guard. Where `lsregister -dump` is too coarse to name the default, a scratchpad CGO
probe calling `LSCopyDefaultApplicationURLForContentType` and `LSCopyApplicationURLsForURL`
answers it exactly; build it in the scratchpad with its own `go.mod` and split the
Objective-C into a `.h` and a `.m` beside the Go file, since a cgo preamble is compiled as
C. Unregister the build-directory bundle afterwards so repeated builds do not leave a pile
of stale claimants. `task test:unit`; `task build && task build:web`.

**Depends on:** None

---

### Task 2: Open a model the file manager hands the app, running or not

**Behavior:** Asking the OS to open a model with emod renders that model in the window and
makes it the save target — whether the app was already running or is starting because of
this request. A request that arrives while the app is running goes through the same
unsaved-changes question every other way of replacing the model goes through, and lands in
the window that is already open rather than starting a second copy. Launching with no file
still shows the empty state.

The one thing this needs that no existing delivery needs is survival: on a cold launch the
OS hands the path over before the webview page exists, so an event emitted then reaches
nobody. So the shell parks the request and tells the page a request is waiting; the page
takes it when it can receive one — at startup, and again whenever the shell says another
has arrived. The park-and-hand-over decision is the task's only real decision and belongs
in `internal/desktop` behind its own type, with its own unit test: which request survives
when a second arrives before the first is taken, and that a request is handed over once and
once only. The shell keeps framework calls alone — read the path off the event, hand it to
the service, emit.

Nothing here keeps a record of whether the page is ready. `Cmd+R` is in the default Wails
menu, so the page is reloaded routinely while the shell's state lives for the whole
process; a shell that remembered "the page can receive now" would emit into the gap between
one page unloading and the next registering, and the file would be gone. Parking every
request and letting the page take it removes that state entirely.

On the frontend, the request joins the gestures that already name a model to open, and
takes its number from the same counter they do — but only once the shell has actually
handed a path over. A take that answers nothing must not claim a number, or it supersedes
the take that got the file and the model is silently dropped. Enumerate the windows a take
can wait in before calling this done: startup racing an arriving request, two takes in
flight at once, and a delivery parked on the unsaved-edits question while another arrives.

What reaches the viewer must reach it through the handler `onFileOpened` registers.
That is what gives criterion 3 its prompt, gives the model its save target and its window
title, and puts it in Open Recent — all without a second copy of any of that policy.

**Acceptance Criteria:**
- [ ] With the app not running, asking the OS to open an `.emod` file launches emod, renders that model, names the window after it, shows its path in the bar along the bottom, and makes it what Save writes to with no dialog
- [ ] With the app already running and nothing unsaved, asking the OS to open a different `.emod` file renders it in the window that is already open; the process count for the app does not change and no second window appears
- [ ] With the app running and the source panel edited, the same request raises the three-button unsaved-changes dialog: Cancel leaves the edited model on screen, Discard opens the new one, Save writes the edited file and then opens the new one
- [ ] A model opened this way appears at the top of **File ▸ Open Recent**, as one opened through the picker does
- [ ] A request naming a file that cannot be read reports the reason where a failed Open reports it and leaves the model on screen
- [ ] Launching the app with no file shows the same empty state as before: the panel open, the landing instructions visible, `(no model)` as the name
- [ ] A unit test in `internal/desktop` for the new type covers a request taken after being parked, a request that arrives with nothing waiting, a second request superseding a first that was never taken, and a second take answering nothing — with fresh fixtures per leaf and `require` assertions
- [ ] Deleting the line that clears the parked request once it is handed over makes that unit test fail (restore after checking)
- [ ] A vitest case in `internal/frontend/tests/platform.desktop.test.js` fires the new shell event through the runtime stub's `listeners` map and requires the model to reach the viewer through the handler `onFileOpened` registered; routing it anywhere else makes that case fail
- [ ] A vitest case covers a take that answers no path: it delivers nothing and does not stop a later request from being delivered
- [ ] `TestShellEventNames` passes with the new event, and fails if the name is changed on the shell side alone or the frontend side alone (restore after checking)
- [ ] `TestBindingNames` and `TestServiceRegistrations` pass with the new service registered in `main.go` and imported in `platform.desktop.js`
- [ ] `viewer.js` still hands `renderPanelSource` a model to open from exactly one call, from one function, and that function still consults the unsaved-edits guard — `viewer.test.js`'s existing scan says so
- [ ] `task test:unit`, `task test:race` and `task test:viewer` pass; `task build`, `task build:wasm` and `task build:web` still succeed

**Affected Files/Modules:**
- `internal/desktop/open_requests.go` — new; the type that parks a request and hands it over once. A new type means a new file; it has no collaborator to inject, so it needs no interface and no fake. `desktop.OpenRequests` is the proposal — a plural role noun in the shape `RecentFiles` and `RecentSlots` already take, reading as English after the package name — and any name meeting that rule is as good
- `internal/desktop/open_requests_test.go` — new; its umbrella test, grouped by operation
- `cmd/emod-desktop/main.go` — the application-event subscription, registered before `app.Run()`, and the new service registered alongside the others. The event it emits joins the `file:…-requested` family and says who asked, the way `file:open-recent-requested` does; `file:open-from-os-requested` is the proposal
- `internal/frontend/desktop/platform.desktop.js` — subscribe to the new event, import the new binding, take a waiting request at startup and on the event, and deliver what it read the way `openRecent` does
- `internal/frontend/tests/bindings-stub.js` — the new service and its answer bag
- `internal/frontend/tests/platform.desktop.test.js` — the new cases

**Patterns to Follow:**
- The service shape — mutex, constructor, methods on one type in one file: `internal/desktop/window_service.go`.
- Its test's organisation — umbrella, operation groups, sentence subtests, fresh fixtures: `internal/desktop/window_service_test.go`.
- The shell listener that hears a native event and hands what it carries onward, deciding nothing: `cmd/emod-desktop/main.go:151-161`.
- Registering a service after the window and before `Run`: `cmd/emod-desktop/main.go:133-149`.
- A frontend subscription that takes what the shell resolved and delivers it: `internal/frontend/desktop/platform.desktop.js:58-96`.
- A gesture that names a model, reads it, and delivers it under the shared counter: `internal/frontend/desktop/platform.desktop.js:376-407`.
- How a test fires a shell event rather than calling an export directly: `internal/frontend/tests/platform.desktop.test.js:44-76`.
- The ordering cases the counter already has to survive: `internal/frontend/tests/platform.desktop.test.js:369-425`.
- The guard that keeps the unsaved prompt unskippable: `internal/frontend/tests/viewer.test.js:2286-2330`.
- Which artefacts the two cross-boundary guards compare: `internal/desktop/event_names_test.go`, `internal/desktop/binding_names_test.go`.

**Testable:** Yes — the parking type is a public API with its own unit test, and the
frontend path is driven end to end through the runtime and bindings stubs, which is the
same path the app takes. What no suite can witness is the OS delivering the request at
all; the launch criteria above close that.

**Certainty:** medium — `window_service.go` is the precedent for a service the shell and
the page share, and `main.go:151-161` for a shell listener that forwards a native event.
Neither covers the variation: no delivery here has to survive arriving before the page
exists, so nothing in this repo holds work across a page's lifetime, and no binding here is
called by the frontend to pull work rather than to push an answer. It also changes what the
app does after launch, which reading the framework's source settles only for the framework's
half.

**Blast radius:** high — a delivery that misses `openModel`'s guard replaces an edited
model without asking, and the author's unsaved edits are gone irreversibly. Criterion 3
makes that guard part of the story rather than a side effect.

**Verification:** `task test:unit`, `task test:race`, `task test:viewer` for the two suites.
Then the real gesture, which Task 1 makes available: `task package:desktop`, register the
bundle, and use the OS's own open on a scratch `.emod` — once with the app not running,
once with it running and clean, once with it running over an edited panel, and once on a
file that has been made unreadable. Confirm one process and one window throughout, and
launch the bundle with no file to see the empty state. Budget this launch-and-read cycle
into the task rather than into a later audit: reading Wails' source settles what its calls
do, not what AppKit does around them. If the shell-to-page hop needs isolating, a temporary
`probe.js` dropped into the assembled `cmd/emod-desktop/frontend/static/` and loaded from
`index.html` can emit the event with real paths on a timer — inject it after
`task build:desktop` (which wipes that directory) and compile with
`go build -tags production`, and it touches no tracked file.

**Depends on:** Task 1

---

### Task 3: Give `.emod` files a document icon in Finder

**Behavior:** `.emod` files carry emod's document icon in Finder rather than the blank
generic page macOS draws for a type with no icon. The icon is derived at packaging time
from a second image under version control, the way the app icon already is, and lands in
`Contents/Resources` beside it under its own name — the document icon and the app icon are
different pictures, and a folder of models should not look like a folder of applications.

The chain has four hops and each must be pinned, because two assertions that each tie one
end to a third value do not compose: the tracked image exists, the packaging step names it
as its input, that step writes the icon file into `Contents/Resources`, and the document
type declared in Task 1 names that file. The existing icon guard reads only the first
`generate icons` line in the task body, so a second one added beside it inherits no guard
at all unless the guard is widened to read every one.

**Acceptance Criteria:**
- [ ] A document image is tracked under `build/`, distinct from `build/appicon.png`
- [ ] `package:desktop` derives an `.icns` from it into `bin/emod.app/Contents/Resources` under a name that is not the app icon's, and does so above the `codesign` line
- [ ] The document type declared in Task 1 names that icon file, and the app icon key still names the app icon
- [ ] `task package:desktop` succeeds and `codesign --verify --deep --strict bin/emod.app` still exits 0
- [ ] After registering the bundle, the icon macOS draws for an `.emod` file is emod's document icon: rendered to an image file and looked at, it differs from the icon rendered the same way for a `.txt` file
- [ ] The guard reads every `generate icons` line in `package:desktop`, not the first, and requires each named input to be a tracked file; deleting the tracked document image alone makes it fail, and so does changing the icon name on the packaging side alone (restore each after checking)
- [ ] `task test:unit` passes and the guard still starts no subprocess and needs no macOS
- [ ] `task build`, `task build:wasm` and `task build:web` still succeed

**Affected Files/Modules:**
- `build/docicon.png` — new, tracked; named beside `build/appicon.png` so the pair says which is which
- `Taskfile.yml` — a second icon derivation in `package:desktop`, above the signature
- `build/darwin/Info.plist` — the document type's icon file name
- `internal/desktop/packaging_task_test.go` — widen the icon subtest to every derivation
- `internal/desktop/bundle_plist_test.go` — the icon name pinned against what the task writes

**Patterns to Follow:**
- The app icon's own chain, which this repeats for a second image: `Taskfile.yml` `package:desktop`, and `internal/desktop/packaging_task_test.go:43-66`.
- The plist side of that chain: `internal/desktop/bundle_plist_test.go:44-51`.
- The tracked source image the first derivation reads: `build/appicon.png`.
- The document type this icon attaches to: `task:1`.

**Testable:** Yes — the guard witnesses the whole chain from the tracked image to the plist
key. It cannot witness what Finder draws, which the rendered-image criterion closes.

**Certainty:** medium — `package:desktop`'s existing `generate icons` line and
`packaging_task_test.go:60-66` are the precedent for deriving an icon from a tracked image
and pinning it. The variation: this is a second derivation the existing single-match guard
cannot see, its name is read from inside a nested plist array rather than a top-level key,
and whether macOS honours the document-type entry's icon key or wants it on the exported
type declaration is not settled by anything here — expect to set it in both places and let
the probe say which took effect.

**Blast radius:** low — a wrong or missing document icon is cosmetic and is undone by
editing two lines. The one thing to keep is the signature staying last, which the existing
packaging guard already refuses to let move.

**Verification:** `task package:desktop`; `codesign --verify --deep --strict`; register the
bundle and look at a `.emod` file in Finder. The discriminating check is the image itself,
and nothing cheaper works: macOS reports the same 32 representations, a non-nil icon and a
full size list whether or not a bundle supplies one, because it synthesises a standard set
around whatever it has. So build a scratchpad CGO probe — its own `go.mod`, declarations in
a `.h` and bodies in a `.m` beside the Go file — that calls `[NSWorkspace iconForFile:]`
for a scratch `.emod` and a scratch `.txt`, renders each with `CGImageForProposedRect` into
an `NSBitmapImageRep`, writes both as PNGs, and then look at the two files. They must
differ, and the `.emod` one must be the document icon. `task test:unit`.

**Depends on:** Task 1

---

### Task 4: Document the file association in the README

**Behavior:** A reader of the README learns that `.emod` files open in emod on
double-click, what has to be true for macOS to know that (the bundle registered — moved to
`/Applications`, or opened once from where it was built), that emod offers itself for
`.json` without taking it over and how to change that if they want to, and that the
association is macOS-only today. The `### Desktop app` section already carries Open, drop,
Open Recent, Save and packaging in this voice; this joins them.

Prose is not exempt from guarding. The Gatekeeper section of US-007 was delivered in full
and guarded by nothing — deleting it left every suite green. Prose carries tokens that are
not prose: a Finder menu label, an extension, a path. Asserting those keeps the guard from
pinning sentences a later edit should be free to improve while still failing when the
instruction itself goes.

**Acceptance Criteria:**
- [ ] The `### Desktop app` section describes double-clicking a `.emod` file: it opens in the app, in the window already open if one is, through the same unsaved-changes question every other way of replacing the model uses
- [ ] It says what makes macOS aware of the association — the bundle registered by being moved to `/Applications` or opened once — since a bundle sitting unopened in `bin/` is not
- [ ] It says emod appears under Finder's **Open With** for `.json` files without becoming their default, and names the Finder route a reader takes if they want it to be
- [ ] It says the association is macOS-only today, with Linux and Windows still to come
- [ ] A guard in `internal/desktop` asserts the Finder labels this section names — the tokens, not the sentences — and those labels are absent from `README.md` before this task, so the guard is empty until the prose lands
- [ ] Deleting the double-click paragraph makes the guard fail; deleting the `.json` paragraph makes it fail separately (restore each after checking)
- [ ] The existing README subtests in `internal/desktop/packaging_task_test.go` still pass unchanged
- [ ] `task test:unit` passes

**Affected Files/Modules:**
- `README.md` — the `### Desktop app` section, alongside the packaging and Gatekeeper prose
- `internal/desktop/packaging_task_test.go` — the token assertions, beside the two README subtests already there

**Patterns to Follow:**
- README subtests that assert a token and say what breaks without it: `internal/desktop/packaging_task_test.go:75-89`.
- The voice, depth and paragraph shape of the section this joins: `README.md:452-490`.
- What the association actually does, which this describes: `task:1`, `task:2`, `task:3`.

**Testable:** Yes — the tokens are read out of the tracked README by the same guard that
already reads it for the bundle name and the quarantine command.

**Certainty:** high — `internal/desktop/packaging_task_test.go:75-89` is this exact guard
for this exact file, and `README.md:452-490` is the section and the voice.

**Blast radius:** low — documentation of behaviour the previous three tasks already
delivered; nothing reads it but a person.

**Verification:** `task test:unit`, then delete each new paragraph in turn and confirm the
guard reddens before restoring it — a cleanup or ordering guard that asserts a proxy for
the thing rather than the thing cannot fail, and the only way to know which this is, is to
delete the line and run it.

**Depends on:** Task 1, Task 2, Task 3

## Summary

**Four tasks**, ordered dependency-first. Task 1 comes before Task 2 not because the
runtime needs the declaration to compile — it does not — but because the declaration is
what makes Task 2 verifiable by the real gesture rather than by a forced open, and because
Task 2's whole value is unreachable from Finder without it. Tasks 1 and 2 are each green on
their own: after Task 1 a double-clicked model launches emod and shows the empty state,
which is unfinished but not broken. Task 3 hangs off Task 1's document-type entry and is
last of the three because it is the least valuable. Task 4 describes all three.

The story is folded by mechanism rather than by file, because every task here touches
`build/darwin/Info.plist` and `internal/desktop`, and Tasks 1 and 3 both touch
`Taskfile.yml` — splitting by file would produce tasks that cannot be committed apart.

**Story criteria coverage:**

| Criterion | Task |
|---|---|
| `.emod` files show emod as an opener in the OS | 1 |
| Double-click launches the app with the model rendered and set as the save target | 2 |
| Double-click on a running app opens it there, subject to the unsaved-changes prompt | 2 |
| `.emod` files display the app's document icon | 3 |
| The association does not take `.json` away from other applications | 1 |
| Launching with no file shows the same empty state as before | 2 |

Nothing is deferred out of the story. What is deferred is the Linux and Windows halves of
the association, which the story's scope note already assigns to US-014 and US-016 and
which inherit Task 2's runtime unchanged.

**Assessments.** One `low` certainty (Task 1 — nothing here has ever declared a type to
LaunchServices), two `medium` (Tasks 2 and 3, each with the precedent named and the
variation it does not cover stated), one `high` (Task 4, whose guard and voice both already
exist for this file). Two tasks carry a `high` blast radius for different reasons: Task 1
writes a claim into a per-machine OS database that editing the plist back does not undo,
and Task 2 sits on the path where a missed guard discards an author's unsaved edits.

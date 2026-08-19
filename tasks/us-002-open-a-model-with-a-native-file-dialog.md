# US-002: Open a model with a native file dialog

## Contents

1. Task 1: Read a chosen file through a framework-free desktop service
2. Task 2: Name the window through the platform seam
3. Task 3: Render a file the host opens, and report one it cannot read
4. Task 4: Show the native file picker and hand over what it chose
5. Task 5: Add File ▸ Open to the desktop menu bar
6. Task 6: Document the native open workflow

The six fall into three slices. Each leaves `main` shippable, and the first two slices
change nothing a desktop user can see — they land the parts the Go and browser suites
can actually hold.

| Slice | Tasks | Exit |
|---|---|---|
| **Reading** | 1 | `internal/desktop` answers a path with the file, or with the reason it could not; `task test:unit` green |
| **Seam and viewer** | 2, 3 | The shared viewer opens a file any host hands it, names the window after it and shows its path; the browser and CLI distributions behave exactly as before |
| **Desktop shell** | 4, 5, 6 | Cmd/Ctrl+O shows the OS picker and the chosen model renders |

---

## Story Reference

`user-stories/emod-desktop.md` → **US-002: Open a model with a native file dialog** (`:38-50`).
Depends on US-001, delivered — `tasks/completed/us-001-render-a-model-in-a-native-desktop-window.md`.
Design detail: `docs/proposals/emod-desktop-proposal.md` §10 Phase 3 (`:692-697`), whose scope this
story takes the first half of (open; save and recent files are US-003 and US-006).

The story's seven criteria and where they land:

| Story criterion | Task |
|---|---|
| Open in the menu bar, on the platform's standard open shortcut | 5 |
| The picker is filtered to `.emod` and `.json` files | 4 |
| Selecting a file renders it immediately — no intermediate paste or render step | 3 renders it, 4 delivers it, 1 reads it |
| The window title shows the opened file's name, and the full path is discoverable in the window | 2 puts the title on the native window, 3 makes the name and the path follow the open file |
| Cancelling the dialog leaves the currently displayed model untouched | 4 |
| A file that cannot be read reports the reason in the status area and leaves the current model on screen | 1 names the reason, 4 delivers it, 3 puts it on screen |
| A file with validation errors still opens: diagram renders what it can, diagnostics panel lists the errors, exactly as pasted source does | 3 |

---

## Boundaries

**Out of scope** — the story's Non-Goals (`user-stories/emod-desktop.md:248-259`), carried verbatim:

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

- **A Go-side dialog.** The proposal sketches `application.Get().Dialog.OpenFile()` inside the
  service (`docs/proposals/emod-desktop-proposal.md:306-322`). That import would put the GUI
  framework and CGO inside `internal/desktop`, which `task test:unit` compiles on every checkout —
  see F5. The picker stays in `platform.desktop.js`.
- **An Open control in the browser viewer.** No button, no `<input type="file">`. The seam grows a
  way for a *host* to deliver a file; a browser has no host that does, so its implementation
  registers the handler and nothing ever calls it. The web viewer's only file entry stays the drop.
- **A second window, or reusing a running app's window.** One model per window, per the non-goals.

**Deferred** — wanted, but not here:

- **Save, Save As, save-in-place** → US-003. `saveFile` on the desktop keeps rejecting with the
  message US-001 gave it (`internal/frontend/desktop/platform.desktop.js:59-62`); nothing in this
  story touches it. The path this story records on the store is what US-003 saves back to.
- **Unsaved-change tracking, and the prompt before an open replaces the model on screen** → US-004.
  Opening here replaces what is displayed with no prompt.
- **A recent-files menu** → US-006, which reuses this story's File menu and its open path.
- **A drop that carries a real path** → US-005. `droppedFile().path` stays empty in both
  implementations, and the drop handler keeps reading contents through the browser file reader.
- **Opening a file the app was launched with, or double-clicked onto** → US-008. A file a host
  delivers before the viewer has registered its handler is discarded rather than buffered; holding
  a launch argument is that story's problem.
- **Noticing that the open file changed on disk** → US-013.
- **The Windows accelerator convention beyond `CmdOrCtrl+O`** → US-016, which is where a Windows
  machine verifies it.

---

## Codebase Context

Verified against the working tree on 2026-08-20, branch `us-002-open-a-model-with-a-native-file-dialog`,
with US-001 landed.

**The platform seam.** `internal/frontend/static/platform.js` (7 lines) re-exports
`{ parseEmod, exportEmod, droppedFile, saveFile, initialState, ready, isReady }` from
`./platform.browser.js`. `internal/frontend/desktop/platform.desktop.js` (69 lines) exports the same
seven over the generated Wails bindings, and `task build:desktop` copies it over `platform.js` inside
its own gitignored assembled tree (`Taskfile.yml:48-64`). Every shared module imports `./platform.js`;
none branches on host.

**Three parity guards already stand where this story adds contract.** They are the established shape
here for pinning something the compiler cannot see:

- `internal/frontend/tests/viewer.test.js:511-546` — reads the single `export { … }` block out of all
  three seam files and requires the two implementations to equal the contract, against a **hardcoded
  list** of the seven names at `:533-535`.
- `internal/frontend/tests/viewer.test.js:487-509` — scans `viewer.js` and `minimap.js` for every
  `getElementById('…')` and requires `viewer.html` to declare that id. Adding a DOM reference to the
  viewer therefore fails the suite until the shared page declares it.
- `internal/desktop/binding_names_test.go` — regexes the exported `ModelService` methods out of
  `service.go` and the `ModelService.X(` calls out of `platform.desktop.js`, requiring the JS calls
  to be a subset. Its two helpers take a path and a receiver name, so a second service is a second
  call, not a second mechanism.

**`internal/desktop`** is one file, `service.go` (34 lines): a package doc stating it imports no GUI
framework and links no CGO, and `ModelService` with three methods over `internal/pipeline` that take
and answer JSON strings, failures wrapped as `{"error": "…"}` by `pipeline.ErrorJSON`
(`internal/pipeline/pipeline.go:116-119`). `internal/desktop/service_test.go` is `//go:build unit`,
`package desktop_test`, one umbrella `TestModelService` grouped by operation.

**The CLI already words a read failure for a user**: `internal/cli/parse.go:22-28` reports
`reading <path>: <err>`, leaning on the OS error text ("no such file or directory",
"permission denied") rather than classifying by hand.

**The viewer's render path.** `viewer.js:137-160` is the Render click: read `#source-input`, number
the render, `Model.sendParse`, set `store.diagnostics`, emit `diagnostics:changed`,
`Model.setModelData`, collapse `#data-panel`, write `✓ Rendered` into `#render-status`. The drop
handler (`:174-193`) writes the file's contents into the textarea and presses Render for the
user — the established way here to render text *exactly as pasted source*.
`Model.sendParse` (`model.js:77-101`) is also what makes `.json` free: diagram-shaped and AST-shaped
JSON are recognised before anything reaches the parser.

**Where the user can see things.** `#render-status` sits inside `#data-panel`
(`viewer.html:1137-1152`), which a successful render collapses — which is why the export handler
un-collapses the panel before writing a failure into it (`viewer.js:216-223`). `#stats`
(`viewer.html:1154-1159`) is the bottom bar and is always visible; it holds Nodes / Edges / Canvas /
Build. The header's `Model: <span id="model-name-display">` (`:1076`) is always visible too.
`bus.on('model:updated')` (`viewer.js:19-30`) writes both `#model-name-display` and `document.title`.

**The vitest harness.** `internal/frontend/vitest.config.js:5-13` aliases the generated bindings path
to `tests/bindings-stub.js`, because `platform.desktop.js` cannot even load from the source tree
otherwise. `tests/platform.desktop.test.js` drives the real desktop module against that stub;
`tests/dropped-file.test.js` runs both implementations under `describe.each` and installs the
`Go` / `WebAssembly` / `fetch` globals with `vi.hoisted` (`:13-20`) so the browser module's
load-time `init()` does not explode. `tests/viewer.test.js` mocks **only** `../static/platform.js`
(`:18-40`) and builds the DOM by hand in `createRequiredElements()` (`:42-84`).

**The Wails v3 beta.9 surface this story uses**, confirmed by reading
`$(go env GOMODCACHE)/github.com/wailsapp/wails/v3@v3.0.0-beta.9`:

- The JS runtime the bundled asset server serves at `/wails/runtime.js` exports `Dialogs`, `Events`,
  `Window`, `Call`, `Create` among others.
- `Dialogs.OpenFile(options)` shows the native picker from JS; options carry `Title`,
  `CanChooseFiles` and `Filters: [{DisplayName, Pattern}]`, where a pattern is a semicolon-separated
  glob list. It resolves to a path string, and to `""` when the user cancels.
- `Events.On(name, cb)` subscribes to what `(*application.WebviewWindow).EmitEvent(name, data…)`
  emits (`pkg/application/webview_window.go:242`). There is no `App.EmitEvent`.
- `Window.SetTitle(title)` retitles the native window from JS.
- `application.DefaultApplicationMenu()` builds App/File/Edit/View/Window/Help;
  `Menu.FindByLabel`, `MenuItem.GetSubmenu`, `Menu.Add`, `MenuItem.SetAccelerator`,
  `MenuItem.OnClick` and `MenuManager.SetApplicationMenu` are all in `pkg/application/menu*.go`.

---

## Findings

**F1 — Growing the seam is a four-place edit, and one of the places is a literal list.** The contract
guard at `viewer.test.js:533-535` enumerates the seven names. A new seam function means
`platform.js`, `platform.browser.js`, `platform.desktop.js` and that list, or the suite goes red —
which is the guard doing its job, not an obstacle.

**F2 — A new DOM reference in `viewer.js` is already pinned to `viewer.html`.** `viewer.test.js:487-509`
fails the moment `viewer.js` looks up an id the shared page does not declare, so the path indicator
lands in `viewer.html` in the same commit and reaches all three distributions from that one copy.

**F3 — `/wails/runtime.js` needs its own alias and stub.** `platform.desktop.js` currently imports
only the generated bindings, which `vitest.config.js:5-13` aliases by their relative specifier. The
runtime is a root-absolute specifier the bundled asset server answers, so it needs a second alias and
a second stub; without one, `platform.desktop.test.js` and `dropped-file.test.js` stop loading
entirely rather than failing a leaf.

**F4 — Wails does not sync `document.title` to the native window title.** Grepping the whole framework
for `document.title` finds nothing. The desktop title criterion is therefore only closed by calling
`Window.SetTitle` explicitly — setting `document.title`, which the viewer already does, changes
nothing a desktop user sees.

**F5 — The dialog cannot live in `internal/desktop`.** The proposal's sketch (`:306-322`) calls
`application.Get().Dialog.OpenFile()` from the service. That import pulls the GUI framework and CGO
into a package `task test:unit` compiles on every checkout, including CI — the containment US-001
established (`docs/architecture.md:236-241`, and the package doc at `internal/desktop/service.go:1-4`)
exists to prevent exactly that. Keeping the picker in `platform.desktop.js` costs one extra crossing
of the boundary (dialog answers a path, service answers the file) and buys a dialog that vitest can
drive against a stub — which, given the story forbids driving the window, is the only place any of
this can be tested at all.

**F6 — Installing an application menu replaces the framework's default one.** The app installs no
menu today, so it gets the framework's default, which on macOS is what carries Quit, Copy/Paste and
the window roles. Building a fresh `Menu` and setting it would silently cost the app those standard
shortcuts, so the Open item is added to `DefaultApplicationMenu()`'s File submenu.

**F7 — "Reports the reason" is a visibility criterion, and jsdom cannot see visibility.**
`tasks/learnings.md:989-993` records a message written into the collapsed data panel passing a
`textContent` assertion while sitting off screen. The read-failure path needs the same un-collapse
the export handler does (`viewer.js:216-223`), and the full-path indicator belongs somewhere always
visible — the `#stats` bar — rather than inside the panel a successful render collapses.

**F8 — A Go method whose only caller is generated JS reads as dead code to `clerk verify`.**
`tasks/learnings.md:983-987` records this for `ModelService.ParseEmod`. The new file-reading method
has exactly that shape, so expect the finding, confirm the JS caller with a grep, and pass
`--audit-accepted` rather than deleting anything.

---

## Open questions, decided

**Q1 — where does the full path show?** *Decided: in the always-visible `#stats` bar
(`viewer.html:1154-1159`), not the header and not the data panel.* The panel collapses on a
successful render (F7) and the header row is already crowded with seven controls plus the model
name. `#stats` is full-width and holds exactly this kind of standing fact. The criterion is that the
path is readable without opening anything; the element's placement inside that bar is free.

**Q2 — does the browser get an Open control?** *Decided: no.* The story is a desktop story, and a
browser file picker yields contents without a path, which is the thing US-003 needs. The browser
implementation of the new registration function accepts a handler and never calls it, which is the
same "already settled" shape `ready` has on the desktop (`platform.desktop.js:10-13`).

**Q3 — one Go service or two?** *Decided: two.* `ModelService` is the pipeline surface; reading a
file is not pipeline work, and the repo's Go architecture rule is that a new file means a new type
rather than more methods on the old one. A second `application.NewService(…)` registration in
`main.go` is what `wails3 generate bindings` needs to emit the second binding module.

---

## Tasks

### Task 1: Read a chosen file through a framework-free desktop service

**Behavior:** `internal/desktop` answers a filesystem path with the file's name, its absolute path
and its contents — or with the reason it could not be read. It imports no GUI framework, so it
compiles and is tested everywhere the rest of the repository is.

**Acceptance Criteria:**
- [x] Given a path to a readable file, the service answers a JSON envelope carrying the file's base name, its absolute path, and its contents verbatim, including files whose contents are not valid `.emod`
- [x] Given a relative path, the answered path is absolute, so a later caller can act on it without knowing the process's working directory
- [x] A path that does not exist, a path the process may not read, and a path naming a directory each answer the `{"error": "…"}` envelope, and the message names which of those it was rather than a generic failure
- [x] No envelope ever carries both contents and an error
- [x] The new type lives in its own file in `internal/desktop`, and the package still imports no GUI framework and links no CGO — `go build ./internal/desktop` succeeds with `CGO_ENABLED=0`
- [x] The binding-name parity guard covers both services: renaming an exported method on either one, on either side of the boundary, fails a Go test
- [x] `task test:unit` and `task test:integration` are green from a checkout where `task build:desktop` has never run

**Affected Files/Modules:**
- `internal/desktop/` — a new file declaring the new file-reading service type and its method
- `internal/desktop/` — its `//go:build unit` test, one umbrella per type, grouped by operation
- `internal/desktop/binding_names_test.go` — extended to read the second receiver's methods

**Patterns to Follow:**
- `internal/desktop/service.go:1-13` for the package doc and the service type's doc, and `:19-33` for the envelope-answering method shape
- `internal/desktop/service_test.go:24-40` for the umbrella test and its `t.Run` grouping
- `internal/desktop/binding_names_test.go:24-33` — `exportedMethodsOn` and `methodsCalledBy` already take a path and a receiver
- `internal/pipeline/pipeline.go:116-119` for the error envelope
- `internal/cli/parse.go:22-28` for how this repository already words a read failure to a user
- `tasks/learnings.md:983-987` — expect `clerk verify` to call the new method dead code

**Testable:** Yes — the service is a public Go API, driven directly from `internal/desktop`'s test package with real temporary files.

**Certainty:** high — `ModelService` is the same shape in the same package (`internal/desktop/service.go:19-33`), and `internal/cli/parse.go:22-28` is the precedent for the read failure.

**Blast radius:** low — no authorization, no write, nothing leaves the machine; a wrong answer costs a message on screen.

**Verification:** `mise exec -- task test:unit`; `CGO_ENABLED=0 go build ./internal/desktop`.

**Depends on:** None

---

### Task 2: Name the window through the platform seam

**Behavior:** The viewer stops naming the window itself and asks the host to. The browser host keeps
setting `document.title` exactly as today; the desktop host additionally retitles the native window,
which no amount of `document.title` can do.

**Acceptance Criteria:**
- [x] Rendering a named model still names the browser tab as it does today, and an unnamed model still falls back to the plain viewer name — the visible behaviour of the CLI and web distributions is unchanged
- [x] The desktop implementation asks the native window to take the same name it writes into `document.title`
- [x] The seam contract guard names the new function, and both implementations satisfy it
- [x] `platform.desktop.test.js` and `dropped-file.test.js` still load and pass with the desktop module importing the Wails JS runtime — a stand-in for that runtime exists and vitest resolves it
- [x] `task test:viewer` is green, and `task build:web` still produces a `web/static/` carrying exactly one platform implementation and no desktop module

**Affected Files/Modules:**
- `internal/frontend/static/platform.js` — the contract gains one name
- `internal/frontend/static/platform.browser.js` — the browser implementation
- `internal/frontend/desktop/platform.desktop.js` — the desktop implementation, over the Wails runtime's `Window`
- `internal/frontend/static/viewer.js:19-30` — the `model:updated` subscriber calls the seam instead of assigning `document.title`
- `internal/frontend/tests/` — a new stub standing in for `/wails/runtime.js`
- `internal/frontend/vitest.config.js` — the alias that resolves it
- `internal/frontend/tests/platform.desktop.test.js` — the desktop implementation's leaves
- `internal/frontend/tests/viewer.test.js:533-535` — the enumerated contract

**Patterns to Follow:**
- `internal/frontend/tests/bindings-stub.js` and `internal/frontend/vitest.config.js:5-13` for a stub plus alias standing in for a module that only resolves in the assembled tree
- `internal/frontend/static/platform.browser.js:112-117` and `internal/frontend/desktop/platform.desktop.js:63-67` for one seam function written twice
- `internal/frontend/tests/platform.desktop.test.js:1-25` for driving the real desktop module against stubs
- `internal/frontend/tests/viewer.test.js:511-546` for the contract guard being extended
- `~/.config/ai/guidelines/javascript/naming-patterns.md` — verb-noun for an operation

**Testable:** Yes — both implementations are public seam functions; the desktop one is asserted against the runtime stub, the browser one through the viewer's visible title.

**Certainty:** medium — the seam's existing pairs and the bindings stub are the precedent, but no existing stub stands in for the Wails *runtime* rather than the generated bindings, and its specifier is root-absolute where the current alias key is relative, which the existing alias shape does not cover.

**Blast radius:** low — a window name; the worst case is a wrongly labelled title bar.

**Verification:** `mise exec -- task test:viewer`; `mise exec -- task build:desktop && ./bin/emod-desktop` and look at the title bar; `mise exec -- task build:web` then confirm `web/static/` has no desktop module.

**Depends on:** None

---

### Task 3: Render a file the host opens, and report one it cannot read

**Behavior:** The shared viewer offers its host a way to hand it a file. A file that arrives renders
straight away, names the window and shows its full path; one that arrives as a reason instead is
reported where the user can see it, with the model already on screen left alone.

**Acceptance Criteria:**
- [x] A file delivered through the seam renders with no further click, and produces the same diagram, model name and diagnostics the same text produces when pasted into the panel and rendered
- [x] A delivered file whose source has validation errors renders what it can and fills the diagnostics panel with the same messages, severities and locations the same pasted source produces
- [x] After a file opens, the window is named after that file rather than the model, and the file's full path is readable in the window without opening or expanding anything
- [x] A delivery that names a reason instead of contents writes that reason into the status area with the data panel open, so it is on screen and not behind a collapsed panel
- [x] After such a failure the diagram, the model name and the path indicator still show the model that was already open
- [x] The path of the open file is recorded on the store, where a later story can save back to it
- [x] Nothing about the browser distribution changes: no host delivers a file to it, the path indicator stays empty, the drop and paste paths behave as before, and `task test:e2e:viewer` is green
- [x] The seam contract guard names the new function, both implementations satisfy it, and the shared page declares every id the viewer now reads

**Affected Files/Modules:**
- `internal/frontend/static/platform.js`, `platform.browser.js`, `internal/frontend/desktop/platform.desktop.js` — the contract gains the registration function; the browser accepts a handler nothing calls
- `internal/frontend/static/viewer.js` — registers the handler in `init`, and the `model:updated` subscriber stops being the only thing that names the window
- `internal/frontend/static/store.js` — the open file's name and path
- `internal/frontend/static/viewer.html:1154-1159` — the path indicator in the always-visible bar
- `internal/frontend/tests/viewer.test.js:18-40, :42-84, :533-535` — the seam mock captures the handler, the DOM gains the id, the contract list gains the name

**Patterns to Follow:**
- `internal/frontend/static/viewer.js:174-193` — the drop handler is how this codebase renders text "exactly as pasted source would"
- `internal/frontend/static/viewer.js:213-224` — writing a failure into `#render-status` *and* un-collapsing the panel that holds it
- `internal/frontend/static/viewer.js:19-30` — the subscriber that names the window and the model
- `internal/frontend/tests/viewer.test.js:265-315` — leaves that drive a seam operation and assert what the user ends up seeing
- `tasks/learnings.md:989-993` — a `textContent` assertion cannot tell you the text is off screen
- `tasks/learnings.md:965-969` — one `flush()` **between** two operations that reach the seam by dynamic import
- `task:2`

**Testable:** Yes — the viewer's mocked seam captures the registered handler and the leaf invokes it, which is how a host would.

**Certainty:** medium — the drop handler and the export handler are exact precedents for rendering-as-pasted and for reporting where the user can see it; what varies is that every existing entry point into the viewer starts from a DOM event, so nothing yet registers a callback for a host to invoke, and no existing leaf drives the viewer that way.

**Blast radius:** low — it touches the shared page and shared viewer, so a mistake reaches all three distributions, but nothing it touches is a permission, a write or a published contract.

**Verification:** `mise exec -- task test:viewer`; `mise exec -- task test:e2e:viewer`; `mise exec -- task build && ./bin/emod diagram --serve <model>` and confirm the browser viewer is unchanged.

**Depends on:** Task 2

---

### Task 4: Show the native file picker and hand over what it chose

**Behavior:** On the desktop, an open request shows the OS file picker filtered to `.emod` and
`.json`. The chosen file is read through the Go service and handed to the viewer; a cancelled picker
hands over nothing at all.

**Acceptance Criteria:**
- [x] An open request shows the native picker, with a filter that names `.emod` and `.json` and a title naming what is being opened
- [x] Choosing a file reads it through the file service and delivers its name, path and contents to the handler the viewer registered
- [x] A cancelled picker — the host answering with no path — delivers nothing: the handler is not called, and nothing about the displayed model, its name, its path or the status area changes
- [x] A file the service reports it could not read delivers that reason instead of contents, so the viewer can report it
- [x] An open request arriving before any handler has been registered is discarded without throwing
- [x] The desktop implementation is the only module that imports the Wails runtime or the generated bindings
- [x] `task test:viewer` is green, and the binding-name parity guard passes with the JS now calling the file service

**Affected Files/Modules:**
- `internal/frontend/desktop/platform.desktop.js` — subscribes to the host's open event, owns the dialog, reads through the service, delivers to the handler
- `internal/frontend/tests/bindings-stub.js` — gains the file service
- `internal/frontend/tests/` — the Wails runtime stub gains a scriptable `Dialogs.OpenFile` and `Events.On`
- `internal/frontend/tests/platform.desktop.test.js` — the leaves

**Patterns to Follow:**
- `internal/frontend/desktop/platform.desktop.js:15-33` for calling a binding and decoding its envelope, and `:21-33` for raising the `{"error": …}` envelope rather than resolving with it
- `internal/frontend/tests/bindings-stub.js` for a stub whose answers a test assigns and whose calls it inspects
- `internal/frontend/tests/platform.desktop.test.js:26-70` for the leaf shape
- `internal/frontend/tests/dropped-file.test.js:13-20` for `vi.hoisted` globals a module needs at load time
- `task:1`, `task:2`, `task:3`

**Testable:** Yes — `platform.desktop.js` is the module `platform.desktop.test.js` already drives, and every host call goes through a stub the test scripts.

**Certainty:** medium — Task 2 lands the runtime stub and `platform.desktop.js:15-33` is the precedent for calling across the boundary, but nothing here has yet subscribed to a Go-emitted event or driven a native dialog, and the cancel-as-empty-string contract comes from reading the framework rather than from any instance in this repository.

**Blast radius:** low — desktop-only, no write, no permission; a wrong filter or a mishandled cancel costs a re-try in a dialog.

**Verification:** `mise exec -- task test:viewer`; `mise exec -- task test:unit` for the parity guard; `mise exec -- task build:desktop` to confirm bindings still generate.

**Depends on:** Task 1, Task 3

---

### Task 5: Add File ▸ Open to the desktop menu bar

**Behavior:** The desktop app carries a File menu with an Open item on the platform's standard open
shortcut, and choosing it reaches the frontend's open path. This is the story's one criterion no
suite can close: the menu and its accelerator are only observable by running the app.

**Acceptance Criteria:**
- [x] The app installs an application menu built from the framework's default one, so the standard application, edit and window items it already had are still there
- [x] Its File menu carries an Open item bound to the platform's standard open accelerator
- [x] Choosing Open, or pressing the accelerator, emits the event on the window that the desktop platform module subscribes to
- [x] The file service is registered with the app, so its bindings are generated beside the model service's
- [x] A Go test fails if the event's name is changed on the Go side or the JS side alone
- [x] `task build:desktop` succeeds and the binary is still built with `-tags production`
- [x] `task test:unit` and `task test:integration` stay green from a checkout where `task build:desktop` has never run

**Affected Files/Modules:**
- `cmd/emod-desktop/main.go:24-45` — the menu, the accelerator, the click handler that emits, and the second service registration
- `internal/desktop/` — the event-name parity guard, in the shape of the binding-name one

**Patterns to Follow:**
- `cmd/emod-desktop/main.go:24-45` for how the app, its services and its window are built today
- `internal/desktop/binding_names_test.go:14-33` — the guard shape for a contract that crosses a language boundary with no compiler between the sides, reading files it does not compile
- `Taskfile.yml:48-64` for the build that assembles the frontend and generates bindings
- `tasks/learnings.md:971-975` — why `-tags production` stays on the build
- `tasks/learnings.md:959-963` — why `go list -e` guards the package exclusion
- `task:4`

**Testable:** Yes for the parity guard, which runs in `task test:unit`. The menu item, the accelerator and the picker they raise are **manual only** — the story forbids automated tests driving the window.

**Certainty:** low — no precedent: `main.go` has never built a menu or emitted an event, the app installs no application menu at all today, and what `DefaultApplicationMenu()` yields on this platform and whether the accelerator survives are settled by running it.

**Blast radius:** low — the desktop shell alone; nothing here is reachable from the CLI, the web bundle or any published contract.

**Verification:** `mise exec -- task test:unit`; then the manual smoke pass, which is where the story's first criterion is closed — `mise exec -- task build:desktop && ./bin/emod-desktop`, then: File ▸ Open is present and shows the shortcut; the shortcut raises the picker; the picker offers `.emod` and `.json`; choosing a model renders it with no further click; the title bar shows the file's name and the bottom bar its full path; Cancel leaves the diagram, name and path as they were; a model with lint errors renders with its diagnostics listed; a file made unreadable (`chmod 000`) reports the reason on screen with the diagram still up.

**Depends on:** Task 4

---

### Task 6: Document the native open workflow

**Behavior:** The two places that describe the desktop app describe what it now does. The README's
desktop section currently states the app has no Open dialog, and the architecture's platform-seam
paragraph enumerates the four things the seam covers.

**Acceptance Criteria:**
- [x] The README's desktop section names Open, its shortcut, and the file types the picker offers, and no longer claims the app has no Open dialog
- [x] It still names what the desktop app cannot do yet — saving, packaging, prebuilt downloads — with the list narrowed to what is actually still missing
- [x] The architecture document's platform-seam paragraph names every capability the seam now carries, and its diagram is still accurate
- [x] A grep for the superseded claims finds nothing, and no added text describes the documents' own history
- [x] No behaviour changes; every suite stays green

**Affected Files/Modules:**
- `README.md:371-388` — the "Desktop app" section, including the "What it does not do yet" paragraph
- `docs/architecture.md:215-234` — the three-distributions section and its seam paragraph

**Patterns to Follow:**
- `README.md:371-388` and `docs/architecture.md:225-234` — the paragraphs that go stale, written by US-001
- `~/.claude/rules/markdown-docs.md` — the result reads as the first version ever written
- `tasks/learnings.md:719-723` — confirm a named location exists before writing a criterion against it

**Testable:** No — prose. It closes on a read and on `rg` finding no surviving stale claim.

**Certainty:** high — both target sections exist at the named lines and were written by US-001 task 10 (`tasks/completed/us-001-render-a-model-in-a-native-desktop-window.md`), naming exactly the claims this story falsifies.

**Blast radius:** low — documentation.

**Verification:** `rg -n 'no Open or Save dialogs' README.md` finds nothing; read both sections; `mise exec -- task test` green.

**Depends on:** Task 5

---

## Summary

**Six tasks.**

**Ordering.** Dependency-first, and within that, testable-first. Tasks 1–4 put every part of the
story that a suite can hold — the read and its failures, the seam, the viewer's behaviour, the
dialog and its cancel — behind the Go and vitest suites, so that Task 5, the one part the story
forbids testing, is as thin as it can be: a menu item and an event name, with a parity guard under
the name. Task 1 and Task 2 have no dependencies and can be built in either order or at once; Task 4
is the first task that needs both halves.

**Criteria only a running window can close**, and what stands in for each in the suites:

| Criterion | Manual because | Guard in the suites |
|---|---|---|
| Open is in the menu bar and answers the standard shortcut | The menu bar is the OS's, and nothing drives the window | Task 5's parity guard fails if the event name diverges between `main.go` and `platform.desktop.js`; everything downstream of the event is covered by Task 4's leaves |
| The picker is filtered to `.emod` and `.json` | Only the OS shows the picker | Task 4 asserts the filter handed to `Dialogs.OpenFile` names both extensions |
| The **native** window title shows the file's name | Wails does not sync `document.title`, so only the real window shows it (F4) | Task 2 asserts the desktop implementation asks the window to retitle, with the same string; Task 3 asserts the string is the file's name |
| Cancelling leaves the model untouched | The user cancels a native dialog | Task 4 drives the host answering with no path and asserts no delivery and no change |
| The reason a file could not be read is **on screen** | jsdom has no layout (F7, `tasks/learnings.md:989-993`) | Task 1 asserts the reason is named; Task 3 asserts it is written into the status area with the panel un-collapsed — and the manual pass is what confirms it is visible |
| The full path is **discoverable in the window** | Same — a `textContent` assertion cannot see the bar | Task 3 puts it in the always-visible `#stats` bar and asserts its content; `viewer.test.js:487-509` pins the id to the shared page |

**Story criteria coverage.** All seven are closed: 1 by Task 5, 2 by Task 4, 3 by Tasks 1+3+4,
4 by Tasks 2+3, 5 by Task 4, 6 by Tasks 1+3+4, 7 by Task 3. None is deferred.

**Assessments.** One `high` (Task 1 — the service is `ModelService`'s shape in `ModelService`'s
package), three `medium` each with its variation named, one `low` (Task 5 — the menu has no
precedent here at all), one `high` for the documentation. No task is `high` blast radius: this story
adds a read path, a menu item and a window title, and touches no permission, no write, no money and
no contract consumed outside this repository.

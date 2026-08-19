# US-003: Save a model back to the file it came from

## Contents

1. Task 1: Write a model back to disk through the desktop file service
2. Task 2: Save the panel's source back to the file it came from
3. Task 3: Raise Save and Save As from the desktop shell
4. Task 4: Document save-in-place

The four fall into three slices. Each leaves `main` shippable, and the first two change
nothing a desktop user can see — they land the two halves the Go and browser suites can
actually hold, which meet at one string.

| Slice | Tasks | Exit |
|---|---|---|
| **Writing** | 1 | `internal/desktop` puts contents at a path, or refuses before anything on disk changes; `task test:unit` green |
| **The viewer's save** | 2 | The shared viewer hands a host the panel's source and the file's path, adopts a location it is given, and reports what came back; the browser and CLI distributions behave exactly as before |
| **Desktop shell** | 3, 4 | Cmd/Ctrl+S writes the open file in place, Cmd/Ctrl+Shift+S retargets it |

---

## Story Reference

`user-stories/emod-desktop.md` → **US-003: Save a model back to the file it came from** (`:52-66`).
Depends on US-002, delivered — `tasks/completed/us-002-open-a-model-with-a-native-file-dialog.md`,
which recorded the open file's path on the store for exactly this story.
The byte-for-byte criterion is deliberately an early, partial answer to what Save writes;
**US-011: Keep the source panel and the diagram in agreement** (`user-stories/emod-desktop.md:165-178`)
is where that question is worked through in full, and Open Question 1 (`:263`) is the part
of it nobody has decided yet.

The story's seven criteria and where they land:

| Story criterion | Task |
|---|---|
| Save is in the menu bar and on the platform's standard save shortcut | 3 |
| With a file open, Save writes to that exact path with no dialog and confirms in the status area | 2 hands over the path and confirms, 3 writes it, 1 puts the bytes down |
| With no file open, Save prompts for a location, writes there, and that path becomes the save target | 2 asks and adopts, 3 raises the dialog |
| Save As is separately available and retargets subsequent saves | 3 puts it in the menu, 2 retargets |
| Opening a file and saving it with no edits leaves the file byte-for-byte unchanged | 2 hands back the bytes it was delivered, 1 puts those bytes down unaltered |
| A failed write reports the reason and leaves the file on disk unchanged | 1 refuses and leaves the file alone, 3 raises the reason, 2 puts it on screen |
| The browser viewer's existing download-based `.emod` export is unchanged | 2 |

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

- **A Save control in the browser viewer.** No button, no keyboard binding, no `showSaveFilePicker`.
  The seam grows a way for a *host* to ask the viewer to save; a browser has no host that
  does, so its implementation registers a handler nothing ever calls — the same shape
  `onFileOpened` already has there (`internal/frontend/static/platform.browser.js:113-116`).
  The web viewer's only way of getting a model out stays the Export download.
- **A Go-side save dialog.** Same reason the open picker is not one (US-002 F5): the dialog
  would drag the GUI framework and CGO into `internal/desktop`, which `task test:unit`
  compiles on every checkout, and `cmd/emod-desktop` is in no test target so a dialog in Go
  is a dialog no suite can drive. The picker stays in `platform.desktop.js`.
- **Any change to what Export writes.** Export still re-serialises the diagram model through
  `Export.exportToEmodString` (`internal/frontend/static/emod-export.js:3-11`). Save writes the
  panel's source. They produce different text from the same model on purpose, and this story
  does not merge them — see Q4.
- **Writing through a symlinked target.** A save replaces the file at the path it was given, so
  a path that is a symlink ends up a regular file. Nobody has asked for the other behaviour and
  deciding it here would be guessing.
- **Refusing to overwrite a file that changed on disk since it was opened.** Save writes what it
  was asked to write. Noticing an external change is US-013.

**Deferred** — wanted, but not here:

- **The modified marker, the prompt before losing edits, and Discard** → US-004. Nothing here
  tracks whether the panel differs from the file on disk; Save writes whatever the panel holds
  whenever it is asked, and asking twice writes twice.
- **Save writing what a diagram edit produced** → US-011. A canvas edit does not reach the
  source panel today, so a model edited on the diagram and then saved writes the panel's source
  — the text the file arrived with — and not the edit. The story's own Context says US-011 is
  where this is worked through.
- **What happens to hand-written comments and formatting when the source is rewritten** → US-011
  and Open Question 1 (`user-stories/emod-desktop.md:263`). This story sidesteps it entirely by
  never regenerating the source.
- **Saving a dropped file back to where it was dropped from** → US-005. A drop still yields
  contents and no path (`internal/frontend/static/platform.browser.js:75-92`), so a dropped model
  saves like pasted source: it asks for a location.
- **Native save dialogs for the CLI's other outputs** — `export -f json`, `-f cue`, draw.io, SVG →
  Open Question 2 (`user-stories/emod-desktop.md:264`), not storied.
- **A recent-files list that a save updates** → US-006.

---

## Codebase Context

Verified against the working tree on 2026-08-20, branch `main`, with US-002 landed.

**The platform seam.** `internal/frontend/static/platform.js` re-exports nine names —
`{ parseEmod, exportEmod, droppedFile, saveFile, setWindowTitle, onFileOpened, initialState, ready, isReady }`
— from `./platform.browser.js`. `internal/frontend/desktop/platform.desktop.js` exports the same
nine over the Wails runtime and the generated bindings, and `task build:desktop` copies it over
`platform.js` in its own assembled tree (`Taskfile.yml:48-65`). Three guards stand where this
story adds contract: the seam guard (`internal/frontend/tests/viewer.test.js:890-920`, whose list
of nine names is hardcoded at `:908-909`), the DOM-id guard (`:832-854`), and the
shared-modules-reach-no-host guard (`:860-883`).

**`saveFile` today.** The browser builds a Blob, clicks a synthetic `<a download>` and resolves
undefined (`platform.browser.js:94-107`). The desktop rejects with
`Saving is not available in this build yet` (`platform.desktop.js:55-62`), which
`internal/frontend/tests/platform.desktop.test.js:199-204` pins and `README.md:399` states in
prose. Its only caller is the Export button (`internal/frontend/static/viewer.js:282-293`).

**The open file on the store.** `store.currentFile` is `{name, path}` or null
(`internal/frontend/static/store.js:7`). `renderPanelSource`
(`internal/frontend/static/viewer.js:155-188`) is the only place it commits, and it commits only
once the parse resolves. `applyWindowTitle` (`:19-22`) names the window after the file when
there is one, and `UI.updateStats` (`internal/frontend/static/ui.js:342-352`) writes the path into
`#stat-file` and un-hides it. Both are driven by `model:updated`, i.e. by a render.

**`internal/desktop`.** `FileService.Read` (`internal/desktop/file_service.go:32-57`) resolves the
path, reads it, refuses anything that is not valid UTF-8, and answers `{name, path, content}` or
`pipeline.ErrorJSON` (`internal/pipeline/pipeline.go:115-119`). Three guards cross the language
boundary from `internal/desktop`: `TestBindingNames` and `TestServiceRegistrations`
(`binding_names_test.go:22-75`), `TestOpenedFileWireKeys` (`:133-143`), and `TestShellEventNames`
(`event_names_test.go:20-30`).

**The Wails v3 beta.9 surface this story uses**, read out of
`$(go env GOMODCACHE)/github.com/wailsapp/wails/v3@v3.0.0-beta.9`:

- `Dialogs.SaveFile(options)` in the JS runtime (`internal/runtime/desktop/@wailsio/runtime/src/dialogs.ts:187`)
  answers `Promise<string>`, and a cancelled dialog answers `""` exactly as `Dialogs.OpenFile` does.
  Its options (`:65-73`) carry `Filename`, `Filters: [{DisplayName, Pattern}]`, `Title`,
  `ButtonText`, `Directory` and `CanCreateDirectories`.
- `application.NewSaveMenuItem()` (`pkg/application/menuitem_roles.go:310-313`) is a plain
  `MenuItem` labelled `Save` with accelerator `CmdOrCtrl+s`, and `NewSaveAsMenuItem()` (`:315-318`)
  one labelled `Save As...` with `CmdOrCtrl+Shift+s`. Neither sets a Role, so both take `.OnClick`.

---

## Findings

**F1 — Export cannot be what Save writes.** `Export.exportToEmodString`
(`internal/frontend/static/emod-export.js:3-11`) hands the diagram model to the Go exporter, which
re-serialises it: `e2e-viewer/tests/export.spec.js:106-114` pins that a re-rendered export
round-trips to *the same text*, which is precisely the canonicalisation that makes it wrong here.
Comments and hand-written layout do not survive it. Save therefore writes the source the panel
shows, and never goes near the exporter.

**F2 — but the panel's text is not the file's bytes either.** A `<textarea>`'s `value` normalises
every CRLF to a bare LF, so a CRLF file opened and saved unedited would come back with every line
ending rewritten. This is already asserted, not theorised:
`internal/frontend/tests/viewer.test.js:626-645` opens a CRLF file and requires the panel's value
to differ from what was delivered. "Save writes exactly the source shown in the panel" is
therefore not literally satisfiable together with the byte-for-byte criterion — see Q1.

**F3 — the read-only refusal and the mid-write guarantee need different mechanisms, and each
breaks the other's criterion.** A `rename` over an existing file needs write permission on the
*directory*, not on the target — so a write that lands via a temporary file and a rename will
happily replace a read-only file, failing "a read-only file reports the reason and leaves the file
unchanged". A plain `os.WriteFile` opens the target with `O_TRUNC` before it writes a byte — so it
refuses the read-only file correctly and, on a failure partway through, leaves a truncated model
where the user's working copy was. Both criteria are in the same story sentence; one mechanism
answers neither. Q2 decides.

**F4 — replacing a file drops its mode.** A temporary file created at `0600` and renamed over a
`0644` model leaves that model owner-only, silently. The repository's existing writes have the
mirror-image bug: `internal/cli/fmt.go:46` writes `0o644` unconditionally, so `emod fmt` on a
`0600` file widens it. Neither is acceptable for a save that claims to leave a file unchanged
apart from its contents.

**F5 — `TestShellEventNames` compares sets, so Save's two halves cannot be two tasks.**
`internal/desktop/event_names_test.go:20-30` requires `require.Equal` between the events
`cmd/emod-desktop/main.go` emits and the events `platform.desktop.js` subscribes to. Adding a
subscription without its emission, or an emission without its subscription, turns that guard red.
US-002 could split them only because the guard did not exist yet — its task 5 created it. Task 3
therefore carries both sides.

**F6 — a save confirmation written into `#render-status` is invisible.** That element sits inside
`#data-panel` (`internal/frontend/static/viewer.html:1170-1173`), which a successful render
collapses to a 40px header — the exact trap recorded at `tasks/learnings.md:989-993`. US-002
answered it for *failures* by revealing the panel (`viewer.js:190-198`), which is right for a rare
event with a reason worth reading and wrong for the application's most frequent keystroke. Q3
decides.

**F7 — implementing the desktop's `saveFile` changes the Export button there.** Export's handler
(`viewer.js:282-293`) is `saveFile`'s only caller, and the desktop implementation currently
rejects (`platform.desktop.js:55-62`). Once it writes, Export on the desktop stops reporting that
saving is unavailable and starts producing a file. That is a behaviour change this story causes
without the story asking for it; Task 3 owns stating it, superseding
`platform.desktop.test.js:199-204` and the README claim at `:399`.

**F8 — the binding-name guard tolerates a Go method with no caller, unlike the event guard.**
`TestBindingNames` asserts `require.Subset(methods, called)` (`binding_names_test.go:32-33`), so a
new exported method with no JS caller keeps it green. That asymmetry is what lets the Go write
land a task before the frontend calls it, while the event names cannot.

**F9 — `clerk verify` will call the new Go method dead code.** `tasks/learnings.md:983-987` records
this for `ModelService.ParseEmod`; a method reached only through generated bindings has exactly
that shape. Confirm the JS caller with a grep and pass `--audit-accepted`.

---

## Open questions, decided

**Q1 — what exactly does Save write?** *Decided: the source in the panel, in the line-ending
convention the opened file arrived with, recorded on the open file when it is delivered.* Not the
exporter's output (F1), and not the panel's raw value (F2). This is the narrowest answer that
closes the byte-for-byte criterion, and it is the answer a text editor gives: a file keeps the
convention it was written in, whatever the editing widget does internally. A file that was never
opened — pasted source saved to a new location — has no convention to keep and gets LF. US-011
inherits the remaining half of the question: what Save writes once the *diagram* has been edited.

**Q2 — how are a read-only target and a mid-write failure both answered?** *Decided: the target's
writability is established before anything is written, and the new contents reach the target only
once they are complete and on disk; whatever ends up at the path carries the permission bits the
target had.* The pre-flight check is what refuses the read-only file (F3), because the completion
mechanism will not; carrying the mode across is what keeps a save from narrowing a file's
permissions (F4). "No space" is not producible in a test — it is answered by the same completion
mechanism as any other failure partway through, and the leaf that stands for it is the one where
the target is writable but its directory is not, which a truncate-first write would fail.

**Q3 — where does a completed save confirm?** *Decided: the always-visible bottom bar
(`viewer.html:1178-1184`), where the open file's path already lives — not `#render-status`.*
A confirmation the user cannot see is not a confirmation (F6), and revealing the source panel on
every save would make the most frequent keystroke in the app rearrange the window. Failures keep
the shape US-002 established: the panel is revealed and the reason goes to `#render-status`,
because a reason is worth interrupting for. The story's phrase "the status area" is read as the
place the user can read status, which in this window is the bar along the bottom.

**Q4 — does the Export button change on the desktop?** *Decided: yes, and deliberately.* It stops
reporting that saving is unavailable and raises the native save dialog, every time — and the path
it chooses never becomes the save target, because what Export writes is the re-serialised model
and what Save writes is the panel's source (F1). Leaving `saveFile` rejecting on the desktop while
Save works would ship a build whose Export button says the app cannot save while Cmd+S saves.

**Q5 — one seam registration for Save and Save As, or two?** *Decided: one.* They are one
behaviour — write the model somewhere — differing only in whether the existing target is used, and
one registration keeps the viewer's save path single. The delivery says whether a location must be
chosen.

**Q6 — does the browser viewer get Save?** *Decided: no.* Its `saveFile` still downloads a copy,
its `onSaveRequested` accepts a handler nothing calls, and its Export button is untouched. That
last one is a story criterion, not an omission.

---

## Tasks

### Task 1: Write a model back to disk through the desktop file service

**Behavior:** `internal/desktop` puts contents at a filesystem path, or answers the reason it will
not — refusing a target it cannot write before anything on disk changes, and never leaving a
target holding a model that is neither the old one nor the new one. It imports no GUI framework,
so it is tested everywhere the rest of the repository is.

**Acceptance Criteria:**
- [x] Writing to a path that does not exist creates it holding exactly the bytes it was given
- [x] Writing over a file that exists leaves it holding exactly the bytes it was given, and holding the permission bits it held before
- [x] Contents this service reads and then writes back unedited leave the file byte-identical — for any file it will read, line endings, trailing blank lines and non-ASCII UTF-8 included
- [x] A target that exists and this process may not write is refused, with the reason naming the permission, and the file still holds what it held
- [x] A target whose directory this process may not write into is refused, and the target — writable in itself — still holds what it held, which is the shape a write that truncated its target first would fail
- [x] A path naming a directory, and a path whose parent does not exist, are each refused with a reason naming which it was
- [x] Every refusal answers the `{"error": "…"}` envelope and leaves the filesystem exactly as it found it, contents and mode alike
- [x] A completed write answers an envelope carrying no error, so a caller can tell the two apart without going to the filesystem
- [x] A reason names the path once, not twice the way the wrapped syscall error does
- [x] The new method lives beside `FileService`'s existing one, and the package still imports no GUI framework and links no CGO — `CGO_ENABLED=0 go build ./internal/desktop` succeeds
- [x] `task test:unit` and `task test:integration` are green from a checkout where `task build:desktop` has never run

**Affected Files/Modules:**
- `internal/desktop/file_service.go` — the write method, beside the read it is the counterpart of
- `internal/desktop/file_service_test.go` — a `write` group under the existing umbrella

**Patterns to Follow:**
- `internal/desktop/file_service.go:25-57` for the method shape, the absolute-path resolution and the error envelope
- `internal/desktop/file_service.go:59-68` for reporting the OS's own reason without repeating the path
- `internal/desktop/file_service_test.go:43-168` for the umbrella and its `t.Run` grouping; `:66-71` is the existing CRLF fixture, `:108-122` the mode-000 leaf with its root skip, `:150-158` the positive partner that keeps a refusal from passing with the feature deleted
- `internal/cli/fmt.go:42-47` — the repository's existing file write, and the shape this one deliberately does not take
- `internal/pipeline/pipeline.go:115-119` for the error envelope
- `tasks/learnings.md:1001-1005` — why the read refuses non-UTF-8, which is this method's premise
- `tasks/learnings.md:983-987` — expect `clerk verify` to call the new method dead code

**Testable:** Yes — a public Go API, driven from `internal/desktop`'s test package against real
temporary files, whose permissions and contents the test reads back directly.

**Certainty:** low — no instance in this repository writes a file without truncating it first
(`internal/cli/fmt.go:46` is a one-line `os.WriteFile` at a fixed mode), and the two failure
requirements pull opposite ways: the mechanism that survives a failure partway through will
overwrite a read-only target, so refusing one needs a check the mechanism does not give for free.
The service shape, the envelope and the test grouping are settled by `FileService.Read` in the
same file; the write itself is not.

**Blast radius:** high — it writes to the user's filesystem and replaces files that already exist.
Getting it wrong destroys a working copy rather than costing a message on screen.

**Verification:** `mise exec -- task test:unit`; `CGO_ENABLED=0 go build ./internal/desktop`;
`mise exec -- task test:integration`.

**Depends on:** None

---

### Task 2: Save the panel's source back to the file it came from

**Behavior:** The shared viewer gains a save path. With a file open it hands its host that file's
path and the panel's source and asks for no dialog; with none it asks the host where to put it and
adopts what it is given, so the next save asks nothing. What comes back is reported where the user
can read it.

**Acceptance Criteria:**
- [ ] A save request with a file open hands the host that file's exact path together with the panel's source, and asks for no location
- [ ] What is handed over is the panel's source in the line-ending convention the open file arrived with, so a file delivered with CRLF endings and saved with no edits hands back exactly the text that was delivered
- [ ] A save request with no file open asks the host for a location, and what the host answers becomes the open file: the window takes its name, the bottom bar shows its path, and a second save writes there without asking again
- [ ] A save-to-a-new-location request asks for a location even when a file is already open, and what the host answers replaces the open file for every later save
- [ ] A host that answers no location writes nothing, and the open file, the window's name and the path on screen stay as they were
- [ ] A completed save is reported where it is readable without opening or expanding anything, and the source panel is left collapsed or open exactly as it was
- [ ] A save the host refuses reports the reason with the source panel revealed, and leaves the model, its name, the panel's source and the save target unchanged
- [ ] The browser distribution is unchanged: the Export button hands the browser the same suggested name and the same content it does today, still reports a failed export in the revealed panel, and the browser suite's export leaves pass untouched
- [ ] The seam contract guard names the new registration and both implementations satisfy it; the shared page declares every id the viewer now reads

**Affected Files/Modules:**
- `internal/frontend/static/platform.js` — the contract gains the save registration
- `internal/frontend/static/platform.browser.js:94-116` — the browser's registration, which accepts a handler nothing calls, and its `saveFile`, whose download behaviour does not change
- `internal/frontend/desktop/platform.desktop.js` — the desktop's registration, which stores a handler nothing yet invokes, so the contract guard is satisfied
- `internal/frontend/static/viewer.js` — the save handler, registered in `init` beside the open one
- `internal/frontend/static/store.js:7` — what the open file has to carry for Save to reproduce its bytes
- `internal/frontend/static/ui.js:342-352` — the bottom bar, which already follows the open file
- `internal/frontend/static/viewer.html:1178-1184` — where a completed save is reported
- `internal/frontend/tests/viewer.test.js` — the seam mock captures the save registration, the hand-built DOM gains the id, the contract list gains the name

**Patterns to Follow:**
- `internal/frontend/static/viewer.js:240-262` — the open handler is the shape a host-driven operation takes in this viewer
- `internal/frontend/static/viewer.js:282-293` — a seam call whose failure is written into the status area with the panel revealed
- `internal/frontend/static/viewer.js:190-198` — `reportOpenFailure`, and why a report claims a render number against a parse still in flight
- `internal/frontend/static/ui.js:342-352` — how the bottom bar follows `store.currentFile`, including the hidden class and the title attribute
- `internal/frontend/tests/viewer.test.js:20-44` for the seam mock and how a host handler is captured, `:479-486` for driving one, `:626-645` for the CRLF file whose line endings the panel already rewrites
- `internal/frontend/tests/viewer.test.js:318-367` — the export leaves that must keep passing with no edit
- `e2e-viewer/tests/export.spec.js:15-22` — the real browser download this story must not disturb
- `tasks/learnings.md:989-993` — a `textContent` assertion cannot tell you the text is off screen
- `tasks/learnings.md:1007-1011` — what an added element inside `#stats` costs in CSS, and why jsdom will not tell you
- `tasks/learnings.md:965-969` — one `flush()` **between** two operations that reach the seam by dynamic import
- `tasks/learnings.md:1013-1017` — hand the promise back rather than asserting on the helper's tail

**Testable:** Yes — `viewer.test.js` mocks the seam, so the leaf registers as a host would, invokes
the save, and reads what the mock was handed and what the window then shows.

**Certainty:** medium — `onFileOpened` and the export handler are exact precedents for a
host-driven seam operation and for reporting where the user can see it, and the CRLF trap is
already pinned by an existing leaf. What varies is that every existing way the window's name and
the path bar change runs through `model:updated`, i.e. through a render; a save that retargets
changes both with no render behind it, which nothing here does yet.

**Blast radius:** high — this task decides the exact bytes an in-place save puts over the user's
working copy. Getting the line-ending answer wrong rewrites every line of a file the user did not
edit, and the diff would look like a whole-file change with no author behind it.

**Verification:** `mise exec -- task test:viewer`; `mise exec -- task test:e2e:viewer`;
`mise exec -- task build && ./bin/emod diagram --serve <model>` and confirm Export still downloads
as before.

**Depends on:** None

---

### Task 3: Raise Save and Save As from the desktop shell

**Behavior:** The desktop app carries Save and Save As on the platform's standard shortcuts, and
the desktop platform module answers them: writing to a known path through the Go service, raising
the native save dialog when there is none, and raising what the write refused so the viewer can
report it. Export stops saying saving is unavailable.

**Acceptance Criteria:**
- [ ] The app's File menu carries Save and Save As, each on the platform's standard accelerator for it, alongside the Open item already there and with the framework's default items still present
- [ ] Choosing either, or pressing its accelerator, emits an event the desktop platform module subscribes to, and a Go test fails if either name is changed on one side alone
- [ ] A save carrying a path writes through the file service to that exact path with no dialog, and answers that path
- [ ] A save carrying no path shows the native save dialog, offering the suggested name it was given and filtered to the extensions a model comes in, and writes to what was chosen
- [ ] A cancelled save dialog writes nothing and answers no path, so nothing is retargeted
- [ ] A write the service refuses is raised carrying the reason the service gave, rather than a message of the frontend's own wording
- [ ] A save request arriving before any handler has been registered is discarded without throwing
- [ ] The Export button on the desktop build shows the save dialog and writes the exported model rather than reporting that saving is unavailable, and the path it chooses does not become the save target
- [ ] The desktop implementation is still the only module that imports the Wails runtime or the generated bindings, and the binding-name guard passes with the frontend now calling the service's write
- [ ] `task build:desktop` succeeds and the binary is still built with `-tags production`
- [ ] `task test:unit`, `task test:integration` and `task test:viewer` are green from a checkout where `task build:desktop` has never run

**Affected Files/Modules:**
- `cmd/emod-desktop/main.go:33-46` — the two menu items, their accelerators and what they emit
- `internal/frontend/desktop/platform.desktop.js` — the two subscriptions, the dialog, the call to the service's write, and `saveFile` becoming real
- `internal/frontend/tests/wails-runtime-stub.js` — a scriptable save dialog
- `internal/frontend/tests/bindings-stub.js` — the file service's write
- `internal/frontend/tests/platform.desktop.test.js` — the leaves, including the one that pins saving as unavailable

**Patterns to Follow:**
- `internal/frontend/desktop/platform.desktop.js:76-118` — a shell event raising a native dialog, crossing to the Go service, and treating an empty answer as a cancel that delivers nothing
- `internal/frontend/desktop/platform.desktop.js:55-62` — the refusal this task replaces, and why it was written to reject audibly
- `cmd/emod-desktop/main.go:33-46` — a menu item added to the framework's default File submenu, with an accelerator and a click that emits
- `internal/desktop/event_names_test.go:20-30` — the guard that requires both sides in one change
- `internal/frontend/tests/wails-runtime-stub.js:22-29` and `internal/frontend/tests/bindings-stub.js:17-19` — a stub whose answer a test assigns and whose calls it inspects
- `internal/frontend/tests/platform.desktop.test.js:88-197` — the leaf shape for a host-driven operation, including the request-numbering leaf and the one that proves the viewer's own throw is not dressed up as a host failure
- `Taskfile.yml:48-65` — the build that assembles the frontend and generates bindings
- `tasks/learnings.md:995-999` — `UseApplicationMenu` is why the menu exists on Windows at all
- `tasks/learnings.md:1031-1035` — a Wails shell task is routine once the framework API has been read, and it has been: `NewSaveMenuItem` / `NewSaveAsMenuItem` at `pkg/application/menuitem_roles.go:310-318` and `Dialogs.SaveFile` with its options at `internal/runtime/desktop/@wailsio/runtime/src/dialogs.ts:65-73,187`, both in the pinned module cache
- `tasks/learnings.md:1013-1017` — hand the promise back, or the leaf asserting the subscription does not throw is vacuous
- `task:1`, `task:2`

**Testable:** Yes for the platform module and both parity guards, which `task test:viewer` and
`task test:unit` run. The menu items, their accelerators and the native dialog they raise are
**manual only** — the story forbids automated tests driving the window.

**Certainty:** high — this is Open run backwards through the same three layers, and each layer has
its precedent in the file it lands in: `platform.desktop.js:76-118` for the event-to-dialog-to-service
chain, `main.go:33-46` for the menu item and its emission, `platform.desktop.test.js:88-197` for the
leaves, and both stubs already carry a scriptable dialog and a scriptable binding. The framework
API the task needs was read before planning, which is the condition `tasks/learnings.md:1031-1035`
names for calling a shell task routine.

**Blast radius:** high — this is the task at which a keystroke starts overwriting the user's files,
and the one where a mis-wired path writes the model somewhere the user did not choose.

**Verification:** `mise exec -- task test:viewer`; `mise exec -- task test:unit`;
`mise exec -- task build:desktop`. Then the manual smoke pass, which is where the story's first
criterion is closed — `./bin/emod-desktop`, then: File ▸ Save and File ▸ Save As are present and
show their shortcuts; with a file open the shortcut writes it with no dialog and the bottom bar
says so; `diff` against a copy taken before the save shows nothing, including on a file saved with
CRLF endings; with pasted source the shortcut raises the save dialog, and the title and path bar
take the chosen file; a second save writes silently; Save As raises the dialog with a file open and
retargets; Cancel changes nothing; saving over a file made read-only (`chmod 444`) reports the
reason with the panel open and leaves the file untouched; Export raises the save dialog and writes.

**Depends on:** Task 1, Task 2

---

### Task 4: Document save-in-place

**Behavior:** The two places that describe the desktop app describe what it now does. The README's
desktop section states that the app cannot save; the architecture document's seam paragraph
enumerates what the seam carries and describes only opening as running from the host inward.

**Acceptance Criteria:**
- [ ] The README's desktop section names Save and Save As, their shortcuts, what Save writes and where it writes it, and no longer says the app cannot save
- [ ] It still names what the desktop app cannot do yet, narrowed to what is actually still missing
- [ ] The architecture document's seam paragraph names every capability the seam now carries, and the paragraph describing the host-driven direction covers saving as well as opening; its diagram is still accurate
- [ ] A grep for the superseded claims finds nothing, and no added text describes the documents' own history
- [ ] No behaviour changes; every suite stays green

**Affected Files/Modules:**
- `README.md:371-402` — the "Desktop app" section, including the "What it does not do yet" paragraph
- `docs/architecture.md:225-236` — the seam paragraph and its enumeration
- `docs/architecture.md:265-277` — the paragraph on the host-driven direction and the two guards holding the language boundary, which is now three

**Patterns to Follow:**
- `README.md:385-402` and `docs/architecture.md:265-277` — the paragraphs US-002 wrote, which this story falsifies
- `tasks/completed/us-002-open-a-model-with-a-native-file-dialog.md` — the shape the previous documentation task took
- `tasks/learnings.md:719-723` — confirm a named location exists before writing a criterion against it

**Testable:** No — prose. It closes on a read and on `rg` finding no surviving stale claim.

**Certainty:** high — `README.md:399` states the claim this story falsifies verbatim ("no saving —
Export reports that it is unavailable"), and both target paragraphs were written by US-002's
documentation task at the lines named.

**Blast radius:** low — documentation.

**Verification:** `rg -n 'no saving|Saving is not available' README.md docs/` finds nothing that is
still true; read both sections; `mise exec -- task test` green.

**Depends on:** Task 3

---

## Summary

**Four tasks.**

**Ordering.** Dependency-first, and within that, testable-first. Tasks 1 and 2 have no
dependencies on each other and can be built in either order or at once: one puts bytes on disk and
the other decides which bytes, and they meet at a single string that neither alters. Task 3 is the
first task that needs both, and it is deliberately the last one that can go wrong — by the time it
lands, the two halves of the byte-for-byte criterion and every failure mode the suites can produce
are already pinned, so what is left in it is a menu, two events and a dialog.

**The byte-for-byte criterion is closed by two halves that meet at one string.** Task 2 asserts
that a file delivered with CRLF endings and saved unedited hands the seam exactly the text it was
delivered — the panel's normalisation undone. Task 1 asserts that contents this service reads and
writes back leave the file byte-identical. Neither half can drift without failing its own leaf,
and their composition — Go read, JSON, textarea, restoration, Go write — is the manual `diff` in
Task 3's smoke pass.

**The failed-write criterion is split by mechanism, not by convenience.** A read-only target is
refused by Task 1 before anything is written, because the completion mechanism would overwrite it
(F3). A failure partway through leaves the previous contents because the new ones only reach the
target once they are whole; the leaf that stands for the unproducible "no space" is the writable
target inside an unwritable directory, which a truncate-first write fails. Task 3 raises the reason
across the boundary, and Task 2 puts it on screen with the panel revealed.

**The negative criterion — the browser export is unchanged — is owned by Task 2**, which is the
only task that touches the browser implementation or the seam it satisfies. It is proved three
ways: `internal/frontend/tests/viewer.test.js:318-367` keeps passing with no edit to those four
leaves; `e2e-viewer/tests/export.spec.js:15-22` still downloads a file named `Billing.emod` from a
real browser under `task test:e2e:viewer`; and the shared-modules guard
(`viewer.test.js:860-883`) keeps any host reach out of `static/`. Task 3 touches `saveFile` again,
on the desktop side only, and re-runs the same e2e suite for the same reason.

**Export's desktop behaviour changes, and Task 3 says so.** Once `saveFile` is real, the Export
button on the desktop raises the native save dialog and writes instead of reporting that saving is
unavailable (F7, Q4). It never adopts the chosen path as the save target. Two artifacts, two
buttons: Export writes the re-serialised model, Save writes the panel's source.

**Criteria only a running window can close**, and what stands in for each in the suites:

| Criterion | Manual because | Guard in the suites |
|---|---|---|
| Save and Save As are in the menu bar, on the standard shortcuts | The menu bar is the OS's, and nothing drives the window | Task 3's event-name guard fails if either name diverges between `main.go` and `platform.desktop.js`; everything downstream of the event is covered by Task 3's leaves |
| The save dialog is filtered and carries the suggested name | Only the OS shows the dialog | Task 3 asserts the options handed to `Dialogs.SaveFile` |
| A save **confirms** where the user can read it | jsdom has no layout (F6, `tasks/learnings.md:989-993`) | Task 2 asserts the confirmation lands in the always-visible bar and that the source panel's collapsed state is untouched; the manual pass confirms it is on screen |
| Opening and saving leaves the file byte-for-byte unchanged | The full chain crosses two languages and a webview | Task 2 pins the JS half against a CRLF fixture, Task 1 the Go half; Task 3's smoke pass `diff`s a real file |
| A read-only file reports the reason and is left alone | The dialog and the window are the OS's | Task 1 asserts the refusal and the untouched file; Task 3 asserts the reason crosses; Task 2 asserts it reaches the revealed panel |

**Story criteria coverage.** All seven are closed: 1 by Task 3, 2 by Tasks 2+3+1, 3 by Tasks 2+3,
4 by Tasks 3+2, 5 by Tasks 2+1, 6 by Tasks 1+3+2, 7 by Task 2. None is deferred.

**Assessments.** One `low` (Task 1 — nothing in this repository writes a file without truncating
it, and the story's two failure requirements pull against each other), two `high` (Task 3, which
repeats US-002's open chain layer for layer with the framework API already read; Task 4, whose
target paragraphs state the claims this story falsifies verbatim), one `medium` (Task 2, whose
variation is retargeting the window's name without a render behind it). Three of the four are
`high` blast radius, which is not dilution: this is the story where a keystroke starts overwriting
the user's working copy, and the only task that does not touch that is the documentation.

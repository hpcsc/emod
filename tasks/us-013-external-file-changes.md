# US-013: Notice when the open file changes on disk

## Contents

1. Task 1: Report what happens on disk to a watched file
2. Task 2: Tell the shell which file the window has open
3. Task 3: Reload the open file when it changes on disk with nothing unsaved
4. Task 4: Ask whether to keep the app's version or reload when the file changes under unsaved edits
5. Task 5: Report a missing open file and keep its model saveable elsewhere
6. Task 6: Document how the app follows the open file on disk

Task 1 is Go and stands alone: a watch on one path that survives an editor's atomic save.
Tasks 2 to 5 thread it through the platform seam, one mechanism each, and Task 6 documents
it. Every task leaves `main` shippable, and the web viewer and `emod diagram --serve` behave
exactly as they do today at every one of them (Decision 1).

| Slice | Tasks | Exit |
|---|---|---|
| **The watch** | 1 | `internal/desktop.OpenFileWatch` reports a change or a missing file for one path, an atomic save reads as a change, and nothing calls it yet |
| **What is watched** | 2 | The shell watches exactly the file the window has open, from every point the viewer commits one, and nothing else once a page starts over |
| **Reload** | 3 | A change to the open file with nothing unsaved is on screen with no click, drawn in place, and the app's own saves reload nothing |
| **The choice** | 4 | A change under unsaved edits asks Keep or Reload, and never while another question is on screen |
| **Missing** | 5 | A moved or deleted file is reported, its model stays, and it can be saved only to a place the user chooses |
| **Documentation** | 6 | README and architecture describe what the app does, and a guard reads the README's paragraph |

---

## Story Reference

`user-stories/emod-desktop.md` → **US-013: Notice when the open file changes on disk**
(`:192-202`). It depends on US-004, delivered
(`tasks/completed/us-004-know-when-there-are-unsaved-changes.md`), whose Boundaries left this
story exactly this work: "Noticing that the file changed on disk while it was open. That is
US-013; the marker compares the panel against the bytes the file arrived with, not against
the file now." US-005, US-006, US-008, US-009, US-011 and US-012 are delivered too, so every
way of opening a model exists and the diagram writes into the panel.

Open Question 5 (`:267`) asks whether this story survives US-009's in-app editing. It is in
scope and built here.

| Story criterion | Task |
|---|---|
| When the open file changes on disk and the app has no unsaved edits, the app reloads it and reports that it did | 3 (1 notices the change, 2 tells the shell which file to watch) |
| When the open file changes on disk and the app has unsaved edits, the app reports the conflict and offers to keep its version or reload from disk | 4 (3 leaves the edits alone until then) |
| The app's own saves never trigger a reload of the write it just made | 3 (Decision 3) |
| Deleting or renaming the open file is reported and leaves the in-app model intact, so it can still be saved elsewhere | 1 notices it, 3 leaves the model intact, 5 reports it and makes Save ask where |
| A reload preserves the current pan and zoom where the model still allows it | 3, and 4's Reload draws the same way (Decision 6) |

---

## Boundaries

**Out of scope** (the story set's non-goals that bear on this story, verbatim from
`user-stories/emod-desktop.md:248-259`):

- **No replacement of existing distributions.** `emod diagram --serve` and the hosted web viewer keep working unchanged; desktop is a third option, not a migration.
- **No desktop-only diagram features.** Anything added to the canvas stays available in all three distributions.
- **No multi-window, tabs, or project workspaces.** One model per window.
- **No automated tests driving the desktop window.** Manual smoke testing of the shell, backed by the existing Go and browser suites.

Watching a file adds nothing to the canvas, so the second of these is untouched: the diagram
is the same in all three distributions.

**Out of scope** (added by this breakdown):

- Watching in the web viewer or under `emod diagram --serve` (Decision 1). The CLI's server
  could watch the file it serves; that is a CLI feature this story does not ask for.
- Following a renamed file to its new name. A rename is reported the way a delete is
  (Decision 7).
- Merging the app's version with the disk's, or showing what differs between them. The
  choice is one version or the other, whole.
- Watching any file but the open one: other files in its directory, the entries of the
  recent list.
- A directory that is itself deleted or renamed. The open file is reported missing and the
  watch ends; the directory coming back is not noticed.
- Filesystems that raise no change events — many network mounts (NFS, SMB) and some FUSE
  filesystems. Nothing reaches the app from them, and a file kept on one behaves as it did
  before this story.
- Fitting the reloaded diagram into view when the kept pan leaves it off screen
  (Decision 6).
- Anything on screen that says the watch is running.

**Deferred:**

- **Checking the disk at the moment of a save.** The watch reports a change once the
  platform's events for it pause, and a save that lands inside that pause writes over the
  change before it is compared. So does the Save answered in an unsaved-changes question
  that was already on screen when the change arrived. Closing both needs Save itself to
  compare the file on disk with what the app last held before it writes, which changes
  US-003's write path. No story carries it yet; it is the follow-up that makes "never save
  over work done outside it" hold with no window at all.
- **The keep-or-reload question on Windows** → US-016. Windows' message box ignores custom
  button labels (US-004 Decision 6), so the answer is one the desktop module does not
  recognise and is read as Keep; Reload is unreachable there.

---

## Decisions

### Decision 1 — the behaviour is the desktop app's; the shared code is inert elsewhere

| Behaviour | Desktop app | Web viewer and `emod diagram --serve` |
|---|---|---|
| Watching the open file (Tasks 1, 2) | `internal/desktop.OpenFileWatch`, started and stopped through `watchOpenFile` | `watchOpenFile` resolves and does nothing. No file a page opens has a location: a drop yields contents only (`internal/frontend/static/platform.browser.js:102-128`), and `--serve` hands the page a diagram, not a file (`internal/frontend/static/viewer.js:1129-1133`) |
| A change on disk reaching the viewer (Task 3) | two shell events, delivered to `onOpenFileChanged` | the handler is accepted and never called, as `onFileOpened` already is (`platform.browser.js:170-184`) |
| The keep-or-reload question (Task 4) | a native two-button question | never asked; answers keep, the outcome that changes nothing |
| A missing open file (Task 5) | reported; the model stays and Save asks where | cannot happen |

The frontend is one copy, so the viewer's new code ships to all three distributions — and
every line of it is reached only through a change the browser never reports. Unlike US-009
and US-012 (`tasks/learnings.md:1263-1269`) there is therefore no goal tension to put to the
user: the web viewer and `--serve` show nothing new.

### Decision 2 — the watch is on the directory, picks the file out by name, and decides once the events pause

`github.com/fsnotify/fsnotify` is not in `go.mod` or `go.sum`. It joins `go.mod` pinned at
`v1.10.1`, the newest tag (`go list -m -versions`).

| Option | Problem |
|---|---|
| fsnotify on the file itself | An atomic save — a working file renamed over the original, which `desktop.FileService.Write` does itself (`internal/desktop/file_service.go:116-152`) — removes the watched inode, so the watch is lost on the save most editors make, and adding it again races the next write |
| **fsnotify on the parent directory, picking the file out by name** | **Under kqueue (macOS) a directory watch holds one descriptor per file in that directory** |
| `rjeczalik/notify` (FSEvents on macOS) | Its macOS backend is cgo, and `internal/desktop` links no CGO (`internal/desktop/service.go:1-6`); it is in `go.sum` only as an indirect dependency of the `wails3` tool |
| Polling the file's modification time | A second write within the timestamp's resolution is missed, and every tick costs a read |

**Chosen: the second.** fsnotify's own README recommends it ("Watching a file doesn't work
well") with `cmd/fsnotify/file.go` in the module as its example.

What happened is decided once the platform's events for the file have paused: the path
either names a file, which is a change, or it does not, which is missing. That one rule is
what makes an atomic save, whose name vanishes and returns inside one burst, a change, and a
delete a missing file. The pause lives in Go, so the viewer arms no timer
(`tasks/learnings.md:1221-1227`).

A watch begins with one report about the file as it is. The gap between the read that
opened a file and the watch starting can be as long as the unsaved-changes question takes to
answer, and a change made inside it would otherwise never be seen.

A path reached through a symbolic link is watched where its bytes live, resolved the way
Save resolves it (`internal/desktop/file_service.go:189-214`), so the watched file is the one
a save writes. Every report names the path the page asked for, because that is what the
viewer compares against the open file's.

The descriptor cost is real and bounded: Go raises the soft descriptor limit to the hard one
at start, and a directory too large to watch makes the watch fail, which Task 2 reports. The
app then behaves for that file as it did before this story.

A report says "look at this file", not "this file changed": Decision 3 is what decides
whether it did.

**Deviation recorded during delivery — the watch polls the path instead of watching the directory.**
Task 1 builds `OpenFileWatch` without fsnotify, and `go.mod` gains no dependency. Each check stats
the open file's path once a second (a ticker in `cmd/emod-desktop`) and compares the file's
identity, size and modification time with what the last report saw, so an atomic save, a write in
place and a rename away all show. Two checks in a row with no file make it missing, which takes the
place of the pause above. The reasons: under kqueue a directory watch holds one descriptor for each
file in the directory for as long as the app runs, and a network filesystem raises no events, while
a stat works on both. The objection in the table, a second write within the timestamp's resolution,
is answered by `coarsestTimeStep`: a match is not trusted while the last report came less than two
seconds (FAT's step) from the file's modification time, so such a write is reported on the next
check. As a result, Task 1's two criteria about fsnotify in `go.mod` and about a directory that
cannot be watched do not apply, a file kept on a network mount is watched, and `Watch` answers no
error. clerk records the change as design note D1.

### Decision 3 — a change is real only when the bytes differ, judged in the viewer after the saves already queued

The viewer already holds what the open file contains as far as the app knows.
`store.currentFile.content` is the text the file arrived with
(`internal/frontend/static/viewer.js:27-48`), and a save replaces it with exactly what it
wrote (`viewer.js:1003-1005`). So the viewer reads the file again through the host — through
`desktop.FileService.Read`, the read every open uses, with its refusal of a file that is not
UTF-8 (`tasks/learnings.md:894-899`) — and compares those bytes with that, never with the
panel's text (`tasks/learnings.md:930-935`). Equal bytes mean nothing happened.

That is the whole of how the app's own saves are told apart. The comparison waits for every
save queued before the report arrived, and each of those adopts the bytes it wrote before
the comparison runs, so the write a save just made always compares equal.

| Alternative | Why not |
|---|---|
| Ignore reports for a while after a save | Drops a real write that lands inside the window, and fires late on a slow disk |
| Tell the Go watch what was written | A second copy of bytes the viewer owns, and it would count a save the viewer did not adopt — another file opened while it wrote (`viewer.js:998-1013`) — as the open file's |

"The app's own saves" is a category, so its members are read off the source rather than
listed from examples: every caller of the viewer's save, which is `onSaveRequested` (Save and
Save As) and the Save chosen in the unsaved-changes question that `clearedToReplace` raises
for an open, a close and a quit (`viewer.js:661-689`, `:1025`). Export is not one of them: it
writes the model re-serialised from the diagram and adopts nothing (`viewer.js:1044-1062`,
`README.md:486-489`). An Export aimed by its dialog at the open file changes that file the
way another program would, and is reloaded or asked about like any other change.

### Decision 4 — the question reuses US-004's mechanism, not its operation

US-004's unsaved-changes prompt, `resolveUnsavedEdits`, offers Save, Discard and Cancel.
Its shape does not fit this question. Its Save writes the app's version over the file — the
one write this story exists to stop — and its message speaks of changes not yet saved rather
than a file changed by someone else. Reading Discard as Reload and Cancel as Keep would put
two of three buttons under labels that describe something else.

What fits is how it is built, and Task 4 builds `resolveExternalChange` the same way:

- a seam operation the browser answers inertly (`internal/frontend/static/platform.browser.js:163-168`)
- a native question answered by the label pressed, with every answer it does not recognise
  read as the outcome that loses nothing (`internal/frontend/desktop/platform.desktop.js:265-286`)
- a turn on the queue every other question already takes (`viewer.js:624-656`), so two
  questions are never on screen at once

Its buttons are Keep my version, which is both the default and the way out, and Reload from
disk.

An in-page notice over the canvas, the way US-011 shows its conflict (`ui.js` `showConflict`),
was the other candidate. It would put markup into the web viewer that the web viewer can never
show, and "conflict" already names that notice's concept.

### Decision 5 — which surface each report uses

"Reports" in this story means where the user can read it, which is the bar along the bottom
(`#save-status` in `#stats`), not `#render-status` inside the collapsed data panel
(`tasks/learnings.md:972-977`).

| Report | Surface |
|---|---|
| The open file was reloaded | the bar |
| It changed while the window holds unsaved edits | the question itself — the story's offer needs an answer — and then the bar says which choice was made |
| The user kept the app's version | the bar; the unsaved marker still shows |
| The open file is missing | the bar, in its failed style |
| A reload could not read the file, or found it empty | the bar, in its failed style |
| The shell refused to watch the file | the bar, by the rule a refused recording follows (`viewer.js:876-907`, `tasks/learnings.md:1092-1097`) |

None of them reveals the data panel. A failure reveals it when it answers something the user
just did; these answer something another program did, and opening the panel would take the
canvas away from someone in the middle of work. Each report leads with the file's name and
what happened, because the bar cuts its text at 30 characters
(`internal/frontend/static/viewer.html:895-916`) and keeps the whole of it in its title. A
reload or missing report is newer than anything in the bar and replaces it.

### Decision 6 — a reload draws in place, and that is what "where the model still allows it" means

Nothing in the viewer moves the pan or the zoom except the user's own gestures, Fit, and
centring on a node (`interaction.js`, `minimap.js`), so any render keeps them already. What an
opening render throws away is the arrangement: `Model.setModelData` starts dragged offsets
and hidden nodes empty (`internal/frontend/static/model.js:19-21`), while the in-place redraw
US-009 uses carries both to each node the new model still has, by its name and its parent
(`model.js:23-33`, `viewer.js:353-359`).

So a reload draws in place, and "where the model still allows it" reads: pan and zoom are
always kept, each node the reloaded file still has keeps its offset and whether it is
hidden, and a node the file no longer has takes both with it. A reloaded file that does not
parse is treated the way a pause in typing treats one (US-009): the last diagram stays,
marked out of date, and the panel lists the diagnostics.

Not done: fitting a diagram that the kept pan leaves off screen. Fit does that in one click,
and moving a view the user chose is what the criterion asks the app not to do.

### Decision 7 — a missing open file is unsaved work, and Save asks where to put it

Once its file is gone, the window holds the only copy of the model. So it counts as unsaved
even with nothing typed, and closing, quitting and opening another model ask US-004's
question first rather than lose it in silence.

Save does not write back to the old path. After a delete that would bring back a file the
user removed; after a rename it would leave a second copy beside the one they moved. Save
asks for a location instead, offering the file's own name — the story's "saved elsewhere".
The window keeps its name and the path in the bar, because that is still the model on
screen.

A file that appears at the path again — an undo in the file manager, a checkout, an editor
slow to finish an atomic save — ends the state. With the bytes the app last held it is the
open file again; with other bytes it is a change on disk like any other.

### Decision 8 — what a change on disk can lose to

A render nobody asked for loses races (`tasks/learnings.md:1214-1220`, `:1256-1262`), and a
delivery parked behind a question is no longer reading (`:1032-1037`). Every window a change
can wait in, and what holds there:

| While the change waits | What holds |
|---|---|
| A save already queued | The comparison runs after it, so its own write compares equal (Decision 3) |
| A diagram edit being applied | It is waited for, and then counts as unsaved |
| An unsaved-changes question on screen | The change is handled after the answer, against whatever is open then |
| A model opened, dropped or chosen from recent files | The change applies only to the file it was about, and claims no gesture number (`platform.desktop.js:61-70`), so it can never supersede the user's own Open |
| Typing during the read | There are unsaved edits now: no reload (Task 3), the question (Task 4) |
| Typing while the reload is drawn | What was typed is kept, on top of the reloaded text |
| A Save asked for while the keep-or-reload question is on screen | It waits for the answer (Task 4) |

One fact about the code bounds how this is built: `clearedToReplace` runs a save inside a
turn of the question queue (`viewer.js:672-689`). A save must therefore never wait on
something that itself waits for that queue, or the first unsaved-changes question answered
with Save leaves the window waiting forever.

---

## Codebase Context

Verified against the working tree on 2026-11-02, branch `us-013-external-file-changes`, with
US-001 to US-012 landed.

**The open file.** `store.currentFile` is `{name, path, content}`. `content` is the text the
file arrived with, and a save replaces the whole record with what it wrote
(`internal/frontend/static/viewer.js:1003-1005`). The viewer commits it in exactly two places:
the opening branch of `renderPanelSource` once the parse resolves (`:295-299`), and
`writeModel` when a save adopts its target (`:998-1013`). Both already call `rememberFile`,
which is how US-006 records every way of opening a model from one branch — the shape Task 2
follows for the watch. A file the browser opens always has an empty `path`.

**The gap between the panel's text and the file's identity.** `renderPanelSource` writes
the panel's text the instant a file arrives and commits the file only once the parse
resolves (`:243-347`). Anything reading both reads across that gap
(`tasks/learnings.md:936-941`), which is why the comparison in Decision 3 waits rather than
reading at once.

**Two queues.** `saveQueue` (`:941-955`) orders saves behind each other and behind renders
and applies through `rendersSettled` and `appliesSettled` (`:957-975`). `guardQueue`
(`:624-645`) orders the unsaved-changes question and the act it authorises, for an open
(`openModel`, `:650-656`) and for a close or a quit (`onLeaveRequested`, `:661-663`).
`clearedToReplace` (`:672-689`) runs a save from inside a guard turn.

**Two kinds of render.** An opening render replaces the model through `Model.setModelData`,
which empties dragged offsets and hidden nodes (`internal/frontend/static/model.js:19-21`). A
revalidation redraws in place through `redrawInPlace` (`viewer.js:353-359`), which carries both
by node identity (`model.js:23-33`). A revalidation of text that does not parse keeps the last
diagram marked out of date (`viewer.js:290-294`, `:381-387`). `store.viewport` is never reset
by either. `writeAppliedSource` (`:579-606`) is the one place the viewer replaces the panel's
text while keeping its scroll position.

**Unsaved work.** `hasUnsavedEdits` (`:426-432`) is what the marker reports
(`reportModified`, `:416-424`) and what the unsaved-changes question asks about: the panel
differing from the file's bytes through `sourceToSave` (`:27-61`), or a diagram edit the panel
does not hold yet.

**The bar.** `reportInBar` (`:861-868`) writes `#save-status`, cut at 30 characters with the
full text in its title (`viewer.html:895-916`). An opening render clears it
(`clearSaveConfirmation`, `:852-859`). Its writers today are a save's confirmation, a refused
recording and the reformat report, and each decides how it ranks against the others
(`:870-907`).

**The platform seam.** `platform.js` re-exports seventeen names from `platform.browser.js`;
`internal/frontend/desktop/platform.desktop.js:481` exports the same seventeen. The contract's
list is written out in `internal/frontend/tests/viewer.test.js:5904-5935`, so a new operation
lands in the contract, both implementations, the viewer's mock (`viewer.test.js:1-81`) and
that list in one change (`tasks/learnings.md:1008-1013`). The browser's inert operations carry
their reason beside them (`platform.browser.js:147-188`). The desktop module queues every call
to a shell service because each binding call is served on its own goroutine
(`platform.desktop.js:179-263`), and it reads a file named by the shell through a `read` the
viewer calls when it is ready (`droppedFileAt`, `:97-110`). The test stubs record the order
calls land in, not only the order they are sent (`internal/frontend/tests/bindings-stub.js:44-79`,
`tasks/learnings.md:1020-1025`).

**`internal/desktop` and its guards.** The package imports no GUI framework and links no CGO
(`service.go:1-6`). A collaborator the shell supplies is a small interface the service takes
at construction, with `nil` allowed and a recording fake in the test — `WindowMarker`
(`window_service.go:5-38`) and `RecentMenu` (`recent_files.go:14-58`). It is imported by
`cmd/emod-desktop` alone — `go list -deps ./cmd/emod ./cmd/emod-wasm` names it nowhere — so a
dependency added here reaches neither the CLI nor the WebAssembly build. Four guards cross the
language boundary:

- `TestBindingNames` (`binding_names_test.go:22-36`) requires, for every service `main.go`
  registers, that `platform.desktop.js` calls at least one of its methods and that every one
  it calls is exported. A service registered with no caller fails it, so registration and
  first caller land together (`tasks/learnings.md:912-917`).
- `TestServiceRegistrations` (`:65-115`) resolves a registered variable through a
  `desktop.New<Type>(` constructor or a `&desktop.<Type>{` literal and nothing else.
- `TestShellEventNames` (`event_names_test.go:23-33`) requires the events every file in
  `cmd/emod-desktop` emits through `EmitEvent("literal"` to equal the events
  `platform.desktop.js` subscribes to, in both directions.
- `TestServiceWireKeys` (`binding_names_test.go:162-190`) pairs each `opened.<field>` read in
  `viewer.js` with a JSON tag `openedFile` writes.

`task test:race` runs `internal/desktop` under the race detector (`Taskfile.yml:120-127`).

**The shell.** `cmd/emod-desktop/main.go` registers services built after the window at
`:133-159` and emits a resolved drop's paths with data at `:161-171`; a Wails event's data
reaches the page as `event.data`. `window.EmitEvent` may be called from any goroutine once the
app runs. The watch starts only from a binding call, which cannot arrive before `Run`, so the
`InvokeAsync`-before-`Run` panic (`tasks/learnings.md:1056-1061`) cannot reach it. The
package is in no test target; `window_marker_darwin.go:24-44` is the shape of an adapter
implementing an `internal/desktop` interface.

**fsnotify v1.10.1.** Requires Go 1.23 and `golang.org/x/sys` v0.13 or later; the module
already carries v0.46.0. Its kqueue backend opens a descriptor for every file in a watched
directory; its inotify backend holds one watch for the directory; on Linux a removed file
still held open somewhere reports a CHMOD before its REMOVE.

---

## Tasks

### Task 1: Report what happens on disk to a watched file

**Behavior:** `internal/desktop` gains `OpenFileWatch`, which watches one path and tells an
`OpenFileListener` the shell supplies when the file there has changed or is missing, once the
platform's events for it have paused. It watches the directory and picks the file out by
name, so an editor's atomic save reads as a change and the watch carries on through it.
`github.com/fsnotify/fsnotify` joins `go.mod` pinned at `v1.10.1`. Nothing outside the package
calls the watch yet.

**Acceptance Criteria:**
- [ ] `go.mod` requires `github.com/fsnotify/fsnotify` at exactly `v1.10.1`, `go.sum` carries its hashes, and `go mod tidy` leaves both files byte-identical
- [x] Writing the watched file in place is reported as a change
- [x] Replacing the watched file as an atomic save does — a working file written beside it and renamed over it — is reported as a change and never as missing, and a write after it is still reported
- [x] Renaming the watched file away and writing a new file at its name, as an editor that keeps a backup saves, is reported as a change and never as missing
- [x] Deleting the watched file, and renaming it away with nothing put in its place, are each reported as missing
- [x] A file that appears at the watched path after it was reported missing is reported as a change
- [x] One write that raises several platform events in quick succession is reported once
- [x] Creating, writing, renaming or deleting another file in the same directory reports nothing
- [x] A watch begins with one report of the file as it is — a change when a file is there, missing when none is — so a change made before the watch started is not lost
- [x] A path reached through a symbolic link is watched where its bytes live, the file a save through that link writes, and every report names the path the watch was asked for
- [x] Watching a second path ends the watch on the first, and once Stop returns nothing more is reported
- [ ] A path whose directory cannot be watched answers the reason and reports nothing
- [x] Watch, Stop and the platform's events reach the watch from different goroutines, and `task test:race` is clean
- [x] `CGO_ENABLED=0 go build ./internal/desktop/...` succeeds, and the package imports no GUI framework
- [x] `task test:unit` is green

**Affected Files/Modules:**
- `go.mod`, `go.sum` — the new requirement
- `internal/desktop/open_file_watch.go` — new: `OpenFileWatch`, `OpenFileListener` and the constructor
- `internal/desktop/open_file_watch_test.go` — new: one `TestOpenFileWatch` umbrella driving real temporary directories through what an editor does, with a recording listener

**Patterns to Follow:**
- `internal/desktop/window_service.go:5-38` for an interface the shell supplies, taken at construction, with `nil` allowed
- `internal/desktop/recent_files.go:14-58` for the same shape around state kept between calls
- `internal/desktop/file_service.go:189-214` for following symbolic links to the file a save writes
- `internal/desktop/window_service_test.go:16-34` for a recording fake, and `:71-90` for a leaf run under the race detector
- `internal/desktop/recent_files_test.go:23-60` for a fake read back as a sequence, and fixtures under `t.TempDir()`
- `internal/desktop/file_service_test.go:283-320` for symbolic-link fixtures
- `tasks/learnings.md:942-947` — a leaf whose point is that nothing happened must be able to fail
- `tasks/learnings.md:876-881` — `Watch` and `Stop` are called only through the generated bindings, which `clerk verify` reads as dead code

**Testable:** Yes — an Exported API leaf per behaviour, through `Watch`, `Stop` and the listener, against a real directory.

**Certainty:** low — nothing in this repository watches the filesystem, the dependency is new, and the rule that tells an atomic save from a delete is this task's own design (Decision 2).

**Blast radius:** low — it reads directory events and reports; it writes nothing and decides nothing about the user's work.

**Verification:** `task test:unit` and `task test:race` green on macOS, and `task test:unit` green on Linux, where the inotify backend runs. `go mod tidy` leaves `go.mod` and `go.sum` unchanged; `CGO_ENABLED=0 go build ./internal/desktop/...` succeeds.

**Depends on:** None

---

### Task 2: Tell the shell which file the window has open

**Behavior:** Each time the viewer commits a file as the open file — an opening render, or a
save that adopts a new location — it tells the host to watch that file's path, or to watch
nothing when the file has no location. A page that starts tells the host to watch nothing, so
a watch a previous page left behind ends. The desktop sends each request to `OpenFileWatch`,
which the shell now registers; the browser accepts it and does nothing. Nothing is yet told
what the watch sees.

**Acceptance Criteria:**
- [x] A file the host opened is watched once its render has committed it, and not before: a file whose render the pipeline rejects is never watched, and the file open before it stays watched
- [x] The request is made from the branch that commits an opened file, so every entry point that delivers a file reaches it with no request of its own — witnessed by a file the host opened and a file the host dropped with a real path
- [x] Opening a model with no location — a drop the page read itself — tells the host to watch nothing, so the file open before it is no longer watched
- [x] A save that adopts a new location — Save As, and pasted source saved for the first time — tells the host to watch the new path; a save back to the open file, and a save whose dialog was cancelled, tell it nothing
- [x] As the page starts it tells the host to watch nothing, before it asks for its initial state
- [x] A watch the host refuses is reported in the bar by the rule a refused recording follows — after an open only while the bar is still empty, after a save added to its confirmation while the bar still shows it — and the model stays open either way
- [x] The desktop implementation sends a path to `OpenFileWatch.Watch` and no path to `Stop`, sends every request with no record of what the shell already holds, and leaves the shell holding the last request whatever order the calls finish in — asserted on the order they land
- [x] A refused request is raised to the caller and does not hold the next request behind it
- [x] The browser implementation resolves and does nothing, with its reason beside it
- [x] The seam contract test names the new operation, and both implementations satisfy it
- [x] The shell registers `OpenFileWatch` in a shape `TestServiceRegistrations` resolves, with no listener yet; `TestBindingNames` covers `Watch` and `Stop`
- [x] `task test:viewer`, `task test:unit` and `task test:race` are green, and `task build:desktop` succeeds

**Affected Files/Modules:**
- `internal/frontend/static/viewer.js` — the request beside the two places an opened file is remembered, and once as the page starts
- `internal/frontend/static/platform.js` — the contract gains `watchOpenFile`
- `internal/frontend/static/platform.browser.js` — the inert implementation
- `internal/frontend/desktop/platform.desktop.js` — the queued `Watch` and `Stop` calls
- `cmd/emod-desktop/main.go` — registers the service
- `internal/frontend/tests/viewer.test.js` — the leaves, the mock and the contract's list
- `internal/frontend/tests/platform.desktop.test.js` — the requests and their landing order
- `internal/frontend/tests/bindings-stub.js` — `OpenFileWatch`, with a landing order

**Patterns to Follow:**
- `internal/frontend/static/viewer.js:295-299` and `:998-1018` for the two places an opened file is committed and remembered
- `internal/frontend/static/viewer.js:870-907` for how a refused recording is reported on the open path and on the save path
- `internal/frontend/desktop/platform.desktop.js:247-263` for a queued call to a shell service that sends every request and raises a refusal, and `:179-222` for ordering by landing
- `internal/frontend/static/platform.browser.js:156-161` for an inert operation that resolves
- `cmd/emod-desktop/main.go:133-159` for registering a service built after the window
- `internal/desktop/binding_names_test.go:76-115` for the registration shapes the guard resolves
- `internal/frontend/tests/viewer.test.js:4064-4237` for the leaves that pin remembering at the commit points
- `internal/frontend/tests/platform.desktop.test.js:1022-1072` and `:250-335` for a queued binding's leaves and landing order
- `internal/frontend/tests/bindings-stub.js:44-79` for a binding that records when it lands
- `tasks/learnings.md:990-995` — the shell's watch outlives the page, so the page's start is a moment to undo it
- `tasks/learnings.md:912-917` — the registration lands with its first caller
- `tasks/learnings.md:1092-1097` — the bar's ranking between a refusal and a confirmation

**Testable:** Yes — the viewer through `init()` and the mocked seam, the desktop module through the bindings stub, and the registration through the Go guards.

**Certainty:** medium — `rememberOpenedFile` is the precedent at the same two commit points through the same kind of queued shell call, but nothing it tells the shell is ever taken back, while a watch must end when the open file loses its location and when a page starts over.

**Blast radius:** low — it says which file to watch; nothing acts on what the watch sees yet.

**Verification:** `task test:viewer`, `task test:unit` and `task test:race` green; `task build:desktop` succeeds and the app opens, opens a file and saves it as before.

**Depends on:** Task 1

---

### Task 3: Reload the open file when it changes on disk with nothing unsaved

**Behavior:** The shell emits what the watch sees to the page as two events,
`file:changed-on-disk` and `file:missing-from-disk`, each carrying the path. On a change the
viewer reads the file again through the host and compares its bytes with the bytes it last
read from it or wrote to it. When they differ and nothing is unsaved, the file's new text
replaces the panel's, the diagram is redrawn in place — pan, zoom and node arrangement kept —
and the bar says the file was reloaded. When they are equal nothing happens at all, which is
why the app's own saves never reload. With unsaved edits nothing is replaced: Task 4 asks. A
missing file changes nothing on screen: Task 5 reports it.

**Acceptance Criteria:**
- [x] A change on disk to the open file, with nothing unsaved, puts the file's new text in the panel and redraws the diagram from it with no click; the window's name and the path in the bar stay the file's
- [x] The bar says the open file was reloaded, leading with its name; the data panel stays open or collapsed as it was, and the source panel keeps its scroll position
- [x] After a reload the marker shows nothing unsaved, and a Save with nothing typed leaves the file byte-for-byte as the other program wrote it, line endings included
- [x] Pan and zoom stay exactly as they were; a node the reloaded file still has keeps its dragged offset and stays hidden if it was hidden; a node the file no longer has takes both with it
- [x] A reloaded file that does not parse leaves the last diagram on screen marked out of date, with the panel holding the file's text and listing its diagnostics, and it is still the open file Save compares against
- [x] A change whose bytes equal what the app last read or wrote changes nothing: no redraw, and nothing new in the bar
- [x] No write by any caller of the viewer's save reloads anything — witnessed by the shell's Save, its Save As back onto the open file, and the Save answered in the unsaved-changes question — including when the change its write raised is delivered before the save's answer has been adopted
- [x] A change to a path that is not the open file's — a file opened since, or none open — reads nothing and changes nothing
- [x] With unsaved edits — anything the unsaved marker counts — a change replaces nothing: the panel, the diagram and the marker stay as they were
- [x] A file that cannot be read when its change arrives — refused, or not valid UTF-8 — or that is now empty leaves the model as it is and says why in the bar; a later change is still compared against what the app last held
- [x] A change is never applied over a model the user opened, dropped or chose from recent files after it arrived, and it is handled only after an unsaved-changes question already on screen has been answered, against whatever is open then
- [x] A diagram edit being applied when a change arrives is waited for, and the change is then one made over unsaved edits
- [x] Text typed while a reload is being drawn is kept, on top of the reloaded text, and the marker measures it against the reloaded file
- [x] A report that the open file is missing changes nothing on screen
- [x] The desktop implementation delivers both shell events to the handler the viewer registered — a change with a read that goes through `FileService.Read` and answers a reason rather than rejecting, a missing file with none — and discards either one that arrives before the viewer has registered, without throwing
- [x] Neither event claims a gesture number, so an Open, a drop or a recent entry in flight is still delivered
- [x] The shell gives the watch a listener that emits both events, and `TestShellEventNames` holds in both directions
- [x] The browser implementation accepts the handler and never calls it; the seam contract test names the operation and both implementations satisfy it
- [x] `task test:viewer` and `task test:unit` are green, and `task build:desktop` succeeds

**Affected Files/Modules:**
- `cmd/emod-desktop/open_file_events.go` — new: the listener that emits the two events
- `cmd/emod-desktop/main.go` — hands that listener to the watch
- `internal/frontend/desktop/platform.desktop.js` — subscribes to both events and hands the viewer each one, a change with its read
- `internal/frontend/static/platform.js`, `internal/frontend/static/platform.browser.js` — the contract gains `onOpenFileChanged`, inert in the browser
- `internal/frontend/static/viewer.js` — the comparison, the reload in place, and the bar's report
- `internal/frontend/tests/viewer.test.js` — the leaves, the mock and the contract's list
- `internal/frontend/tests/platform.desktop.test.js` — both events reaching the handler, and the read

**Patterns to Follow:**
- `internal/frontend/static/viewer.js:243-347` for an opening render that commits the file and a revalidation that draws in place, and `:353-359` for the in-place redraw
- `internal/frontend/static/viewer.js:579-606` for replacing the panel's text while keeping its scroll position
- `internal/frontend/static/viewer.js:941-975` for the save queue and the waits on renders and applies, `:977-1023` for a save adopting the bytes it wrote, and `:624-689` for the question queue
- `internal/frontend/static/viewer.js:814-849` for how an unreadable and an empty file are reported on opening
- `internal/frontend/desktop/platform.desktop.js:78-110` for a shell event carrying data handed to the viewer with a read that answers a reason, and `:61-70` for the gesture number a change must not claim
- `cmd/emod-desktop/main.go:161-171` for emitting an event with data, and `cmd/emod-desktop/window_marker_darwin.go:24-44` for a shell adapter implementing an `internal/desktop` interface
- `internal/desktop/event_names_test.go:23-33` for the guard both events must satisfy
- `internal/frontend/tests/viewer.test.js:4852-5039` for leaves on what an in-place redraw keeps, `:1580-1742` for a save overlapping something else, `:3983-4063` for two arrivals at once, and `:5325-5527` for a render nobody asked for against an open, a save and a reported failure
- `internal/frontend/tests/platform.desktop.test.js:485-617` for leaves on a shell event and its read
- `task:2` for the watch request this task's events answer
- `tasks/learnings.md:930-941` — compare against bytes on disk, and across the panel-to-file gap
- `tasks/learnings.md:894-899` — the read goes through the same refusal every open does
- `tasks/learnings.md:1214-1220` and `:1256-1262` — a render nobody asked for, and the audit rounds it costs
- `tasks/learnings.md:1032-1037` — every window a delivery can wait in
- `tasks/learnings.md:1068-1073` and `:1194-1200` — driving the built app with a probe, and reading what it rendered

**Testable:** Yes — the viewer through `init()` and the mocked seam, the desktop module through the runtime and bindings stubs, the event names through the Go guard.

**Certainty:** low — no render here both commits a file and draws in place, and none started by the host rather than the user has had to order itself against opens, saves, typing, diagram edits and the unsaved-changes question at once; US-009's revalidation, the nearest shape, took three audit rounds for its ordering alone.

**Blast radius:** high — it replaces the user's work on screen without asking, so a wrong "nothing unsaved" answer is an irreversible write over typed edits the textarea's own undo cannot bring back.

**Verification:** `task test:viewer` and `task test:unit` green; `task build:desktop`. In the built app: open a model, change it from another program both in place (`printf … >`) and by an atomic replace (write a copy, `mv` it over), and watch the panel and the bar follow each; save with ⌘S and confirm with `shasum` that the file holds the other program's bytes. Where nothing can look at the window, a probe that triggers `file:save-requested` after the change and the bytes on disk are the oracle (`tasks/learnings.md:1068-1073`).

**Depends on:** Task 2

---

### Task 4: Ask whether to keep the app's version or reload when the file changes under unsaved edits

**Behavior:** A change on disk to the open file while the window holds unsaved edits asks the
host which version to keep, naming the file. Keep my version leaves the panel, the diagram
and the marker exactly as they were and writes nothing; a later Save writes the app's version.
Reload from disk replaces them with the file as it is when the answer is given, drawn in place
as Task 3 draws, and writes nothing. The question is never on screen beside another one, and a
Save asked for while it is waits for the answer.

**Acceptance Criteria:**
- [x] A change on disk to the open file while the window holds unsaved edits — anything the unsaved marker counts — asks the host to choose between keeping the app's version and reloading, naming the file; nothing on screen changes before the answer
- [x] Reload replaces the panel and the diagram with the file as it is when the answer is given, drawn in place as Task 3 draws, clears the marker and any disagreement between the diagram and the panel, writes nothing, and says in the bar that the file was reloaded
- [x] A file that cannot be read when Reload is chosen, or that is now empty, leaves the model and the marker as they were and says why in the bar
- [x] Keep leaves the panel, the diagram and the marker exactly as they were, writes nothing, and says in the bar that the app's version was kept while the file differs
- [x] After Keep, a further report of the same bytes asks nothing, a later change to other bytes asks again, and a Save writes the app's version
- [x] A change that arrives while the unsaved-changes question is on screen is asked about only after that question is answered, and not at all once the file it was about is no longer the open file
- [x] A model opened, dropped, chosen from recent files or asked for by the system while this question is on screen raises its own question only after this one is answered
- [x] A Save asked for while this question is on screen writes only after it is answered
- [x] When the edits are no longer unsaved by the time the question's turn comes — saved, or typed back to the file's text — the change is handled as Task 3 handles one, and nothing is asked
- [x] The desktop question names the file and says it changed on disk while the window holds unsaved edits; it offers Keep my version, as the default and the way out, and Reload from disk, and reads a dismissed dialog, a label it does not recognise and a failed dialog as Keep
- [x] The browser implementation asks nothing and answers keep; the seam contract test names the operation and both implementations satisfy it
- [x] `task test:viewer` is green, and `task build:desktop` succeeds

**Affected Files/Modules:**
- `internal/frontend/static/viewer.js` — the question in its turn, and what each answer does
- `internal/frontend/static/platform.js`, `internal/frontend/static/platform.browser.js` — the contract gains `resolveExternalChange`, inert in the browser
- `internal/frontend/desktop/platform.desktop.js` — the two-button question and the mapping from the label pressed to the outcome
- `internal/frontend/tests/viewer.test.js` — the leaves, the mock and the contract's list
- `internal/frontend/tests/platform.desktop.test.js` — the question's shape and its answers

**Patterns to Follow:**
- `internal/frontend/desktop/platform.desktop.js:265-286` for a question answered by the label pressed, with every unrecognised answer read as the outcome that loses nothing
- `internal/frontend/static/viewer.js:624-689` for questions taking turns and the act each answer authorises
- `internal/frontend/static/viewer.js:564-574` for setting aside diagram edits and the disagreement notice
- `internal/frontend/static/platform.browser.js:163-168` for an inert question with its reason
- `internal/frontend/tests/viewer.test.js:3828-3982` for leaves on a question raised over unsaved edits, and `:4238-4317` for one the host starts
- `internal/frontend/tests/platform.desktop.test.js:725-780` for leaves on a question's buttons and answers
- `internal/frontend/tests/wails-runtime-stub.js:36-53` for the question dialog in the stub
- `task:3` for the reload and the waits this question's turn sits behind
- `tasks/learnings.md:1032-1037` — a delivery parked on a question is no longer reading

**Testable:** Yes — the viewer through `init()` and the mocked seam, the question through the runtime stub.

**Certainty:** medium — `resolveUnsavedEdits` and the question queue are the precedent for a question answered by label and acted on in its turn, but here the question is raised by the host rather than a gesture, a Save asked for while it is on screen must wait for it, and its Reload acts on the disk as it is when answered rather than when asked.

**Blast radius:** high — Reload discards the user's unsaved edits, so a wrong mapping from button to outcome, or an answer acted on against a stale file, throws away work the user chose to keep.

**Verification:** `task test:viewer` green; `task build:desktop`. In the built app: open a model, type, change the file from another program, and exercise both buttons; after Keep confirm with `shasum` that nothing was written until ⌘S; after Reload confirm the panel holds the other program's text.

**Depends on:** Task 3

---

### Task 5: Report a missing open file and keep its model saveable elsewhere

**Behavior:** When the open file is deleted or renamed away, the bar says so and the model
stays exactly as it is. From then the window counts as holding unsaved work, so closing,
quitting and opening another model ask first, and Save asks where to write rather than
putting the file back where it was. A file that appears at the path again ends that state.

**Acceptance Criteria:**
- [x] A report that the open file is missing says in the bar, in its failed style, that the file was moved or deleted, leading with its name; the panel, the diagram, the window's name and the path in the bar stay as they were
- [x] From then the window counts as holding unsaved work even with nothing typed: the marker shows it, and closing, quitting and opening another model each ask the unsaved-changes question first
- [x] Save asks for a location, offering the file's own name, and does not write to where the file was; the location chosen becomes the open file, the window takes its name, the marker clears, and the host is told to watch it
- [x] A Save whose dialog is cancelled leaves everything as it was, the file still missing
- [x] Discard in the unsaved-changes question goes ahead and writes nothing
- [x] A file that appears at the open file's path again with the bytes the app last read or wrote ends the missing state: the marker goes back to what the panel says, and Save writes there with no dialog
- [x] A file that appears there with other bytes is a change on disk to the open file: reloaded when the panel holds no edits of its own, asked about when it does
- [x] A report that a path other than the open file's is missing changes nothing
- [x] `task test:viewer` is green

**Affected Files/Modules:**
- `internal/frontend/static/viewer.js` — the missing state, its report, what it does to unsaved work and to Save, and its end
- `internal/frontend/tests/viewer.test.js` — the leaves

**Patterns to Follow:**
- `internal/frontend/static/viewer.js:416-432` for the one answer the marker and the unsaved-changes question both read
- `internal/frontend/static/viewer.js:977-1023` for a save with no file to write back to, which asks where and adopts the answer
- `internal/frontend/static/viewer.js:624-689` for the question an open, a close and a quit raise
- `internal/frontend/tests/viewer.test.js:1800-1981` for leaves on the marker, `:1356-1579` for leaves on where a save writes, and `:4238-4317` for leaves on a close or quit over unsaved work
- `task:3` for the reports reaching the viewer, and `task:4` for a change made over the panel's own edits
- `tasks/learnings.md:972-977` — the bar is where the user reads it

**Testable:** Yes — the viewer through `init()` and the mocked seam.

**Certainty:** medium — `hasUnsavedEdits` and `writeModel`'s no-file branch are the precedent for "counts as unsaved" and "Save asks where", but no file here has been named by the window while Save must not write back to it, and none has been able to come back.

**Blast radius:** high — it decides whether a model with no copy left on disk can be closed without a question, and where Save writes it.

**Verification:** `task test:viewer` green; `task build:desktop`. In the built app: open a model, delete it in the file manager, read the bar, try ⌘W and confirm the question, then ⌘S and confirm the dialog offers the name; restore the file from the Trash and confirm ⌘S writes back to it with no dialog.

**Depends on:** Task 3, Task 4

---

### Task 6: Document how the app follows the open file on disk

**Behavior:** The README's desktop section and the architecture document describe what the
app now does when the open file changes, disappears or is saved by the app itself, and a guard
holds the README's paragraph to the labels the question shows.

**Acceptance Criteria:**
- [ ] The README's desktop section gains a paragraph saying that a change another program makes to the open file is reloaded when nothing is unsaved, keeping pan, zoom and dragged nodes, and that the bar says so
- [ ] That paragraph says that over unsaved edits the app asks, naming both choices by the labels the dialog shows, and that the app's own saves reload nothing
- [ ] It says that a moved or deleted file is reported, that its model stays and counts as unsaved, and that Save asks where to put it
- [ ] It says that the browser viewer and `emod diagram --serve` do not watch their file
- [ ] A guard captures that paragraph by its opening rather than reading the whole README, and requires it to name the two labels read out of the desktop module's own table of outcomes, so renaming a button fails it
- [ ] `docs/architecture.md`'s list of what a host provides names watching the open file, delivering its changes, and asking what to do about one, and says how the browser answers each; its paragraph on the shell names `OpenFileWatch` and the two events the event-name guard covers
- [ ] Neither document says the question works on Windows, or that a renamed file is followed to its new name
- [ ] The diff touches `README.md`, `docs/architecture.md` and the one test file that holds the guard, and nothing else

**Affected Files/Modules:**
- `README.md` — the desktop section, beside the unsaved-changes paragraph
- `docs/architecture.md` — the seam paragraph and the shell paragraph
- `internal/desktop/packaging_task_test.go` — the README guard, beside the README guards it already holds

**Patterns to Follow:**
- `README.md:387-509` for the section's voice and the paragraphs this one sits among
- `docs/architecture.md:233-247` for the seam's list of what a host provides, and `:303-336` for the shell paragraph and the guards it lists
- `internal/desktop/packaging_task_test.go:100-122` for a README paragraph captured by its opening and held to labels read out of the frontend
- `tasks/learnings.md:1138-1144` — prose a criterion rests on needs a reader
- `tasks/learnings.md:1173-1179` — a guard's tokens must be unique to its paragraph

**Testable:** Yes — the guard reads the README paragraph; the rest is prose.

**Certainty:** high — US-004 to US-008 extended this section and its guards in place, the closest being the double-click paragraph's guard (`internal/desktop/packaging_task_test.go:100-122`).

**Blast radius:** low — documentation and a guard over it.

**Verification:** `task test:unit` green. Read both sections against the branch: every sentence names behaviour the code has; delete the new paragraph and confirm the guard fails.

**Depends on:** Task 5

---

## Summary

**Six tasks**, ordered dependency-first and, within that, so that each later task rests on a
mechanism already proven:

- Task 1 is the riskiest piece and stands alone — a new dependency and a rule about filesystem
  events that no code here has needed — so it lands and is tested before anything relies on
  it.
- Task 2 makes the shell watch the right file and nothing else, from the two places the viewer
  commits one. It is the first task that registers the service, because the binding guard
  fails on a registered service with no caller.
- Task 3 is the core slice: a change reaches the viewer and is reloaded. It carries the
  self-save criterion, because the byte comparison that tells a save from a change is the
  same comparison that decides whether to reload.
- Task 4 and Task 5 are each one more answer to a change, and Task 5 follows Task 4 because a
  file that comes back with other bytes under the panel's edits is asked about.
- Task 6 follows the repository's convention of documenting a desktop story in its closing
  task.

**Story coverage.** All five criteria are covered, and none is deferred. Two are read in a way
stated in full above:

- "The app's own saves" is every caller of the viewer's save, and Export is not one of them
  (Decision 3).
- "Where the model still allows it" is pan and zoom always, and the arrangement of each node
  the file still has (Decision 6).

**Assessments to overrule before they are built on.**

- **Task 1 — low certainty.** Nothing here watches the filesystem; the directory-and-pause
  rule is this breakdown's design, and its fd cost on macOS is accepted rather than avoided.
- **Task 3 — low certainty, high blast radius.** A render the host starts has to hold its
  own against every other claim on the panel, and a wrong "nothing unsaved" answer replaces
  typed work with no undo.
- **Task 4 — high blast radius.** Reload throws away the user's edits.
- **Task 5 — high blast radius.** It decides whether a model with no copy on disk can be closed
  without a question, and where Save writes.

**Judgments worth a second look.**

- **The save-time check is deferred.** A save inside the watch's pause, or one answered in an
  unsaved-changes question already on screen when the change arrived, still writes over the
  change. Making Save compare the disk first would close it, at the cost of changing US-003's
  write path; it could be a seventh task here.
- **A missing file counts as unsaved, and Save asks where** (Decision 7) rather than writing
  back to the old path.
- **An Export aimed at the open file is reloaded like any other program's change**
  (Decision 3).

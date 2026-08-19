# emod Desktop App

## Overview

Ship `emod` as a native desktop application alongside the CLI viewer and the hosted web viewer. Today a model author viewing a diagram either runs `emod diagram --serve` (which starts a localhost server and opens a browser) or uses the web viewer, and in both cases the browser can never write back to the file the model came from — the best it can do is download a copy. The desktop app closes that gap: real file dialogs, save back to the file you opened, recent files, `.emod` files that open on double-click, and a source panel that validates as you type.

The three distributions share one frontend and one Go pipeline. Desktop is the only genuinely new runtime — no WASM, no HTTP server, real filesystem access. Design detail is in [`docs/proposals/emod-desktop-proposal.md`](../docs/proposals/emod-desktop-proposal.md).

Stories are listed in recommended implementation order: prove the runtime seam first, then deliver the native file workflow that justifies the app, then the in-app editing surface, then the platform builds.

## Goals

- A model author can view and edit models in a standalone app with no browser, no localhost server, and no terminal
- Save writes back to the file that was opened, so the desktop app fits an existing working copy instead of a Downloads folder
- Models open the way desktop documents open: file dialog, drag-drop, recent files, double-click in the file manager
- The source panel validates continuously, reporting the same diagnostics the CLI reports, with navigation between a diagnostic and the source that caused it
- Diagram edits and source edits converge on one model, and Save writes what the author sees
- macOS, Linux (Ubuntu 24.04+) and Windows builds are downloadable without a Go toolchain
- The CLI, the LSP, and the web viewer keep working exactly as they do today, from a single copy of the frontend

## User Stories

### US-001: Render a model in a native desktop window
**Description:** As a model author, I want a standalone emod app that renders my model so that I can work with diagrams without starting a local server or opening a browser.

**Acceptance Criteria:**
- [ ] Launching the app opens a window with the same viewer interface as `emod diagram --serve` — source panel, diagram canvas, minimap, visibility toggles, diagnostics badge
- [ ] Pasting `.emod` source into the panel and rendering produces a diagram identical to the browser viewer's for the same source
- [ ] Pan, zoom, fit-to-view, node selection, the detail panel, layout reset, and the diagram context actions behave as they do in the browser viewer
- [ ] Invalid source fills the diagnostics badge and panel with the same messages, severities, and locations as the browser viewer
- [ ] The app renders with no network access, no local HTTP server, and no listening port
- [ ] `emod diagram --serve` and the published web viewer behave exactly as before this story
- [ ] A change to a shared viewer UI file appears in all three distributions without editing a second copy of that file
- [ ] The desktop framework version is pinned exactly in both `go.mod` and the tool manifest, with no floating or `latest` reference

**Context:** This is the walking skeleton — done when the window renders, even though every native capability that justifies the app arrives later. It carries the whole structural cost: the viewer's browser-specific behaviour (loading the Go core, reading a dropped file, downloading an export, receiving injected initial state, waiting for readiness) has to move behind one seam so a native implementation can be swapped in. The proposal inventories that surface as five touch points across three files (§4.1) and phases the work in §10. The existing frontend unit tests already mock at exactly this boundary. The single-source criterion is the one that keeps the three distributions from drifting into three forks; the pin is what keeps an alpha framework from moving underfoot between builds.

### US-002: Open a model with a native file dialog
**Description:** As a model author, I want to open a `.emod` file through the operating system's file picker so that I can load models from anywhere on disk without pasting their contents in.

**Acceptance Criteria:**
- [ ] The app offers Open in its menu bar and responds to the platform's standard open shortcut
- [ ] Open shows the OS file picker, filtered to `.emod` and `.json` files
- [ ] Selecting a file renders it immediately — no intermediate paste or render step
- [ ] The window title shows the opened file's name and the full path is discoverable in the window
- [ ] Cancelling the dialog leaves the currently displayed model untouched
- [ ] A file that cannot be read (missing, no permission) reports the reason in the status area and leaves the current model on screen
- [ ] A file with validation errors still opens: the diagram renders what it can and the diagnostics panel lists the errors, exactly as pasted source with the same errors does

**Depends on:** US-001

### US-003: Save a model back to the file it came from
**Description:** As a model author, I want Save to write back to the file I opened so that my edits land in my working copy instead of a copy in my downloads folder.

**Acceptance Criteria:**
- [ ] Save is in the menu bar and on the platform's standard save shortcut
- [ ] With a file open, Save writes to that exact path with no dialog and confirms in the status area
- [ ] With no file open — pasted source — Save prompts for a location, writes there, and that path becomes the save target for subsequent saves
- [ ] Save As is separately available and retargets subsequent saves to the newly chosen path
- [ ] Opening a file and saving it with no edits leaves the file byte-for-byte unchanged
- [ ] A failed write (read-only file, permission denied, no space) reports the reason and leaves the file on disk unchanged
- [ ] The browser viewer's existing download-based `.emod` export is unchanged

**Context:** Save-in-place is the payoff that justifies the whole port — a browser cannot do it, because a file it reads through a drop or a picker never carries a real path. The byte-for-byte criterion is deliberate: it forces an early answer to what Save actually writes, which US-011 then works through in full.

**Depends on:** US-002

### US-004: Know when there are unsaved changes
**Description:** As a model author, I want the app to show me when I have unsaved edits and stop me losing them so that closing the window or opening another model never silently discards work.

**Acceptance Criteria:**
- [ ] Any change that would alter the saved file marks the window as modified using the platform's convention (a dot in the close button on macOS, `*` in the title elsewhere)
- [ ] A successful save clears the modified marker
- [ ] Closing the window or quitting with unsaved changes prompts with Save, Discard, and Cancel; Cancel aborts the close
- [ ] Opening another model — dialog, recent files, or drop — with unsaved changes goes through the same prompt
- [ ] Discarding leaves the file on disk unchanged
- [ ] Moving a node for layout alone does not mark the file as modified

**Depends on:** US-003

### US-005: Open a model by dropping it on the window
**Description:** As a model author, I want to drag a `.emod` file from my file manager onto the app so that opening a model is one gesture, and the file I dropped is the file I save back to.

**Acceptance Criteria:**
- [ ] Dropping a `.emod` or `.json` file renders it and makes it the current file, exactly as opening it through the dialog would
- [ ] Save then writes back to the dropped file's real location with no dialog
- [ ] Dropping a file with any other extension reports `Only .emod and .json files are supported` and leaves the current model on screen
- [ ] Dropping while unsaved changes exist goes through the unsaved-changes prompt
- [ ] Dropping several files at once opens the first supported one and names the file it opened
- [ ] The drop-target highlight behaves as it does in the browser viewer

**Context:** The browser viewer already accepts drops, but reads them through the browser's file reader, which yields contents and never a path — the reason the browser build can only download a copy. On the desktop the same gesture carries a real path, which is what makes the second criterion possible.

**Depends on:** US-004

### US-006: Reopen a recently opened model
**Description:** As a model author, I want the app to remember models I have opened so that returning to the two or three I am actively working on takes one click instead of a trip through the file picker.

**Acceptance Criteria:**
- [ ] The app lists recently opened models in a menu, most recent first
- [ ] Selecting an entry opens that model exactly as the file dialog would, including becoming the save target
- [ ] The list survives quitting and relaunching the app
- [ ] The list holds at most 10 entries; reopening a listed file moves it to the top rather than duplicating it
- [ ] Selecting an entry whose file no longer exists reports that and removes the entry from the list
- [ ] The list can be cleared

**Depends on:** US-002

### US-007: Install and run a packaged app on macOS
**Description:** As a macOS user, I want a double-clickable emod app so that I can run it without a Go toolchain or a terminal.

**Acceptance Criteria:**
- [ ] A single build command produces an app bundle carrying the emod name and icon
- [ ] The bundle launches from Finder on a machine with no Go toolchain, no Node, and no emod CLI installed
- [ ] Copying the bundle to another Mac and launching it works after one documented step
- [ ] The README documents that step — Open Anyway, or `xattr -dr com.apple.quarantine` — and explains why it appears and that it is one-time per machine
- [ ] The app shows the emod name and icon in the Dock and the app switcher
- [ ] Building and releasing the CLI and the web viewer are unaffected by the desktop build

**Context:** Signing and notarization are deliberately out of scope (proposal §7.3). On Apple Silicon an ad-hoc signature is applied automatically, so a locally built app runs with no ceremony; only a copy that arrives through a browser download gets quarantined, which is what the documented step addresses.

**Depends on:** US-001

### US-008: Open a model by double-clicking it in the file manager
**Description:** As a model author, I want `.emod` files to open in the app when I double-click them so that models behave like documents rather than arguments to a command.

**Acceptance Criteria:**
- [ ] After installing the app, `.emod` files show emod as an opener in the OS
- [ ] Double-clicking a `.emod` file launches the app with that model rendered and set as the save target
- [ ] Double-clicking while the app is already running opens the model in the running app — subject to the unsaved-changes prompt — rather than starting a second copy
- [ ] `.emod` files display the app's document icon in the file manager
- [ ] The association does not take `.json` files away from other applications
- [ ] Launching the app with no file shows the same empty state as before

**Depends on:** US-007, US-004

### US-009: Edit source in the app and see diagnostics as you type
**Description:** As a model author, I want the source panel to validate continuously so that I can write and fix `.emod` in the app itself instead of switching to an editor to find out what is wrong.

**Acceptance Criteria:**
- [ ] Typing in the source panel revalidates automatically after a short pause, with no Render click
- [ ] The diagnostics badge and panel update on each revalidation with the same rule messages, severities, and locations the CLI reports for the same source
- [ ] Source that validates cleanly re-renders the diagram in place, preserving the current pan, zoom, and node layout
- [ ] Source that fails to parse keeps the last successfully rendered diagram on screen and marks it as stale, rather than blanking the canvas
- [ ] Typing stays responsive while validation runs, on the largest bundled example model
- [ ] An explicit render control remains available

**Context:** The app already has a source panel and a render button; the change is that validation stops being a button press. The same lex → parse → validate → lint result the CLI and LSP produce is what has to reach the panel, so a rule's message and severity never differ between surfaces.

**Depends on:** US-001

### US-010: Jump between a diagnostic and the source that caused it
**Description:** As a model author, I want to click a diagnostic and land on the line that caused it so that fixing an error does not mean hunting for it.

**Acceptance Criteria:**
- [ ] Each diagnostic in the panel shows its line, and its column where one is known
- [ ] Clicking a diagnostic moves the source panel's caret to that position and scrolls it into view
- [ ] Clicking a diagnostic still highlights the matching diagram element when one exists, as it does today
- [ ] Lines carrying a diagnostic are marked in the source panel, visually distinguishing error, warning, and info severities
- [ ] Putting the caret on a marked line surfaces that diagnostic's message
- [ ] A diagnostic with no position reports that it applies to the whole file rather than jumping to line 1

**Depends on:** US-009

### US-011: Keep the source panel and the diagram in agreement
**Description:** As a model author, I want edits made on the diagram and edits typed in the source to converge on one model so that Save writes what I am looking at, whichever surface I edited.

**Acceptance Criteria:**
- [ ] Adding, moving between parents, or deleting elements on the diagram updates the source panel to match
- [ ] Editing the source and revalidating updates the diagram, as in US-009
- [ ] An edit on one surface never silently discards an unapplied edit on the other; when both have diverged the app says so and offers a resolution
- [ ] Save writes exactly the source shown in the panel
- [ ] A file that was `emod fmt`-clean before a diagram edit is still `emod fmt`-clean after the app rewrites its source
- [ ] Purely visual state — node offsets, pan, zoom, panel collapse — never appears in the saved source

**Context:** The viewer can already edit models structurally on the canvas and turn the result back into `.emod` text. Adding a live source panel creates a second editing surface over the same model, and this story is where the two are reconciled. It is the highest-risk story in the set: what happens to hand-written comments when a diagram edit rewrites the source is not yet decided — see Open Questions.

**Depends on:** US-009

### US-012: Read source with syntax highlighting
**Description:** As a model author, I want `.emod` source in the panel to be highlighted so that reading a model in the app is no worse than reading it in my editor.

**Acceptance Criteria:**
- [ ] Keywords, identifiers, strings, numbers, and comments are visually distinguished in the source panel
- [ ] The same construct is highlighted as the same category the editor integrations already use, so colours do not contradict between the app and an editor
- [ ] Highlighting keeps up with typing on the largest bundled example model
- [ ] Source that fails to parse degrades to plain or partially highlighted text rather than breaking the panel
- [ ] Selection, copy, paste, and undo in the source panel behave as in a plain text field

**Depends on:** US-009

### US-013: Notice when the open file changes on disk
**Description:** As a model author who also edits `.emod` in an editor, I want the app to notice external changes so that I never save over work done outside it.

**Acceptance Criteria:**
- [ ] When the open file changes on disk and the app has no unsaved edits, the app reloads it and reports that it did
- [ ] When the open file changes on disk and the app has unsaved edits, the app reports the conflict and offers to keep its version or reload from disk
- [ ] The app's own saves never trigger a reload of the write it just made
- [ ] Deleting or renaming the open file is reported and leaves the in-app model intact, so it can still be saved elsewhere
- [ ] A reload preserves the current pan and zoom where the model still allows it

**Depends on:** US-004

### US-014: Run the desktop app on Linux
**Description:** As a Linux user, I want a desktop build for my distribution so that I get the same app my colleagues run on macOS.

**Acceptance Criteria:**
- [ ] A single build command on Ubuntu 24.04 produces a runnable desktop artifact
- [ ] The artifact launches on a clean Ubuntu 24.04 installation with only the documented runtime libraries installed
- [ ] The README names those runtime library packages, distinguishing them from the development packages needed to build
- [ ] The README states the supported floor (Ubuntu 24.04 and later) and names the symptom a user on an older release will see
- [ ] File dialogs, save-in-place, recent files, drag-drop, and `.emod` association behave as on macOS, using Linux conventions
- [ ] The diagram and the source panel are identical to the macOS build

**Context:** The Linux support floor is a decision, not an accident (proposal §7.5): the app targets the current GTK4 / WebKitGTK 6.0 stack with no fallback for older distributions, so the build stays on the best-supported path. Runtime library naming in this stack has churned repeatedly, which is why the README criterion is specific about runtime versus development packages.

**Depends on:** US-001 — the file-workflow criteria assume US-002, US-003, and US-008 are delivered

### US-015: Download a desktop build without building it
**Description:** As someone the author wants to hand emod to, I want prebuilt macOS and Linux downloads so that I can try the app without a Go toolchain.

**Acceptance Criteria:**
- [ ] A manually triggered build produces downloadable macOS and Linux artifacts from a single run
- [ ] The Linux artifact is built against the declared support floor, not whatever the build environment happens to default to
- [ ] Everyday pushes and the existing CLI release are unaffected — no added jobs, no added wall-clock on the normal loop
- [ ] Artifact names make the platform and the commit they came from obvious
- [ ] The download instructions cover the macOS first-launch step and the Linux runtime requirements
- [ ] A failing desktop build never blocks the CLI release or the web viewer deployment

**Depends on:** US-007, US-014

### US-016: Run the desktop app on Windows
**Description:** As a Windows user, I want a desktop build so that emod is usable on my machine without WSL or a browser.

**Acceptance Criteria:**
- [ ] A build produces a runnable Windows artifact
- [ ] The artifact launches on a clean Windows machine with no Go toolchain and no emod CLI, and the download instructions name any system component it depends on
- [ ] File dialogs, save-in-place, unsaved-change marking, recent files, and drag-drop behave as on macOS and Linux, using Windows conventions
- [ ] `.emod` files can be associated with the app and open on double-click
- [ ] The unsigned-application warning a user will see on first launch is documented alongside the download
- [ ] Windows artifacts come from the same on-demand build as macOS and Linux
- [ ] The CLI and web distributions are unaffected

**Context:** Windows was excluded in the proposal and is added here as the last platform, after the macOS and Linux builds have settled. It is the only story in the set the author may not be able to verify on their own hardware — see Open Questions.

**Depends on:** US-015

## Non-Goals

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

## Open Questions

1. **What happens to comments and formatting when a diagram edit rewrites the source (US-011)?** Turning a diagram back into text may not preserve hand-written comments or layout. If it cannot, the fallback is to warn before the first diagram edit on a file that has comments — but that needs deciding before US-011 starts.
2. **Should the app expose the CLI's other outputs** — `export -f json` / `-f cue`, draw.io and SVG diagrams — through native save dialogs? Not currently storied; it is a natural fit once native saving exists.
3. **Where do recent files, window geometry, and preferences live?** A config location shared with the CLI, or a desktop-only one.
4. **Is there a Windows machine available to verify US-016?** If not, that artifact ships built-but-unverified, and the story should say so rather than imply it was tested.
5. **Does US-013 survive US-009?** File watching assumes editing happens elsewhere. If the in-app source panel becomes good enough, external editing may stop being the common case.
6. **Should the app surface the DSL version a file pins?** Files can now declare one, and a version the app does not support is a different failure from a parse error.

Assumptions carried from the proposal and the answers to its open questions: unsigned distribution everywhere; macOS and Linux first with Windows following; Ubuntu 24.04 as the Linux floor; and rendering parity between desktop and browser, so a diagram never differs between distributions.

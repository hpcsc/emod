# US-007: Install and run a packaged app on macOS

## Contents
1. Task 1: Assemble a launchable `emod.app` bundle from a single package command
2. Task 2: Give the bundle the emod icon
3. Task 3: Document packaging and the first-launch quarantine step in the README

## Story Reference

`user-stories/emod-desktop.md` — **US-007: Install and run a packaged app on macOS**.
Design detail in `docs/proposals/emod-desktop-proposal.md` §7.3 (what "unsigned" costs
users) and §10 Phase 4 (packaging). Depends on US-001, already delivered.

## Boundaries

**Out of scope** (the story file's non-goals, carried verbatim):
- **No code signing or notarization** on any platform — no Apple Developer account, no App Store, no Windows signing certificate. Users take a documented one-time step on first launch.
- **No auto-update.** New versions are downloaded manually.
- **No replacement of existing distributions.** `emod diagram --serve` and the hosted web viewer keep working unchanged.
- **No automated tests driving the desktop window.** Manual smoke testing of the shell, backed by the existing Go and browser suites.

**Out of scope** (added by this decomposition):
- **No adoption of `wails3 package` or the wails Taskfile tree.** Measured, not assumed: `commands.Package` in the pinned module (`internal/commands/task_wrapper.go`) does nothing but run the *project's own* `package` task through the wails-patched task runner, dispatching to `darwin:package`. Adopting it would mean this repo's root `Taskfile.yml` defining `build`/`package` dispatch tasks and a `build/Taskfile.yml` tree — and this repo's root `build` task is the one that produces the CLI. The bundle assembly is mirrored from wails' own `create:app:bundle` instead, which is ~8 shell lines.
- **No `.dmg`, no installer, no universal binary.** The bundle carries the build host's architecture only.
- **No `Assets.car` / Icon Composer icon set.** A single `.icns` in `Contents/Resources`.
- **No version injection.** The plist carries a static version string; deriving it from a tag or a build flag is deferred — nothing in the repo injects a version into a Go binary today.
- **No deployment-target floor.** `build:desktop` sets no `MACOSX_DEPLOYMENT_TARGET`, so the bundle targets whatever the build host's SDK defaults to. Wails' own darwin task pins 12.0; matching it is deferred.
- **No rewrite of the README's `## Quick Start` / `### Install` section.** The packaging step is documented in the existing `### Desktop app` section.

**Deferred:**
- Linux packaging → US-014. Windows → US-016. CI release artifacts → US-015 (proposal §7.5 / Phase 5). File associations → US-008 — note that US-008 will bind to the bundle identifier this story chooses.

## Codebase Context

**What exists.** `task build:desktop` (`Taskfile.yml:48-65`) assembles
`cmd/emod-desktop/frontend`, generates bindings with `go tool wails3`, and builds
`./bin/emod-desktop` with `-tags production` and `CGO_ENABLED=1`. It is the only CGO
binary here. `internal/desktop` holds the testable logic and no GUI framework import;
`cmd/emod-desktop` holds framework calls and adapters only. There is no `build/`
directory, no icon, and no image asset of any kind anywhere in the repo.

**Measured during exploration** (each of these was run, not inferred):

- `otool -L bin/emod-desktop` lists only `/System/Library/Frameworks/*` and
  `/usr/lib/*`. Nothing from Homebrew, no Go runtime dependency, and the frontend is
  already inside the binary via `//go:embed all:frontend`. This is the concrete
  evidence available for the story's "no Go toolchain, no Node, no emod CLI" criterion.
- `codesign -dv bin/emod-desktop` reports `Signature=adhoc`, `flags=…(adhoc,linker-signed)`,
  `Identifier=a.out`, `Info.plist=not bound`, `Sealed Resources=none`. So the Go linker's
  ad-hoc signature is real (as §7.3 says), but it carries `a.out` as its identifier and
  seals neither a plist nor resources — which is what the bundle-level ad-hoc `codesign`
  step in wails' `create:app:bundle` changes.
- `go tool wails3 generate icons -input <png> -macfilename <icns> -windowsfilename ""`
  exits 0 and writes a valid `.icns` — but the result holds a **single** `ic10`
  (1024×1024) representation, confirmed with `sips` and `file`. macOS downsamples from
  it; if the Dock icon looks poor at small sizes the macOS-native `iconutil` route is the
  fallback. `go tool wails3` resolves only from inside the module, which the Taskfile
  already relies on at `Taskfile.yml:58`.
- Wails' own bundle assembly is `create:app:bundle` in the pinned module at
  `internal/commands/build_assets/darwin/Taskfile.yml` (module cache
  `github.com/wailsapp/wails/v3@v3.0.0-beta.9`). Read it: it is the exact sequence —
  the two `Contents` directories, the icns and optional `Assets.car`, the executable,
  the plist, then an ad-hoc `codesign`. Its plist template is not in the module; the
  `examples/*/build/darwin/Info.plist` files show the key set it emits.
- `/usr/bin/lsappinfo` is present and `lsappinfo list` reports each running app's name,
  bundle id and bundle path. That is the way to read what the Dock and app switcher show
  without AppleScript, which learning 7 records as refused from this harness.

**Config-guard precedents.** `internal/desktop/event_names_test.go` and
`binding_names_test.go` are exactly the "Not Every Test Has a Caller" shape: `//go:build
unit`, `package desktop_test`, reading repo files by relative path (`../../cmd/emod-desktop`,
`../frontend/desktop/platform.desktop.js`) and requiring two independently-maintained
artifacts to agree. `internal/desktop` is in the `test:unit` package set; `cmd/emod-desktop`
is excluded from it. A guard placed in `internal/desktop` therefore runs on CI (ubuntu), so
it must read tracked files only and shell out to nothing.

**What CI runs.** `.github/workflows/ci.yml` runs `task build`, `test:unit`, `test:race`,
`test:integration`, `test:grammar`, `test:vscode`, `test:viewer`, `test:e2e:viewer`,
`test:e2e` on `ubuntu-latest`, and deploys `task build:web` to Pages.
`.github/workflows/release.yml` runs goreleaser, whose `.goreleaser.yaml` builds
`./cmd/emod` only, `CGO_ENABLED=0`, linux+darwin. None of these may acquire a macOS or
CGO dependency.

## Tasks

### Task 1: Assemble a launchable `emod.app` bundle from a single package command

**Behavior:** `task package:desktop` builds the desktop binary and wraps it in
`bin/emod.app` — a double-clickable, ad-hoc-signed macOS application bundle that opens
the viewer window from Finder and identifies itself as `emod` in the Dock and the app
switcher. A parity guard pins the bundle's plist against what the build actually
produces, and pins the CLI and web build paths against acquiring a desktop dependency.

The bundle needs `Contents/MacOS/<executable>`, `Contents/Info.plist`, and (from Task 2)
`Contents/Resources`. The plist's key set is the contract with macOS, not this repo's
choice of shape: `CFBundlePackageType`, `CFBundleName`, `CFBundleExecutable`,
`CFBundleIdentifier`, `CFBundleVersion`, `CFBundleShortVersionString`,
`NSHighResolutionCapable`, `NSHumanReadableCopyright`. `CFBundleName` is what the Dock
and app switcher display. `CFBundleIdentifier` is the identifier macOS records in
LaunchServices and the one US-008's file associations will bind to — choose it once,
deliberately, from the module path.

The packaging task is macOS-only (`codesign`), and must be declared so rather than
failing on Linux.

**Acceptance Criteria:**
- [x] `task package:desktop`, run alone on a tree with no `bin/`, produces `bin/emod.app` containing `Contents/MacOS/<the binary build:desktop produces>` and `Contents/Info.plist`
- [x] `open bin/emod.app` opens the viewer window, and while it runs `lsappinfo list` reports it with the name `emod` and the chosen bundle identifier
- [x] `codesign --verify --deep --strict bin/emod.app` exits 0, and `codesign -dv bin/emod.app` reports `Signature=adhoc` with an identifier derived from `CFBundleIdentifier` rather than `a.out`, and sealed resources rather than `Sealed Resources=none`
- [x] `otool -L bin/emod.app/Contents/MacOS/<binary>` lists only paths under `/System/Library` and `/usr/lib` — no Homebrew prefix, no path inside the repo
- [x] A copy of `bin/emod.app` placed outside the repo (e.g. `/Applications` or a temporary directory) launches and renders a diagram there
- [x] A guard in `internal/desktop` requires `build/darwin/Info.plist`'s `CFBundleExecutable` to name the binary `Taskfile.yml`'s `build:desktop` produces; editing either side alone makes it fail (revert after checking)
- [x] The same guard requires `CFBundleName` to be `emod`, read from the plist by name rather than by walking a directory
- [x] The same guard requires both `go list` exclusion pipelines in `Taskfile.yml` to carry `-e`, and requires the CLI and web build paths (`build`, `build:wasm`, `build:web`, and `.goreleaser.yaml`'s build entry) to name no desktop target and keep `CGO_ENABLED` off; removing the `-e` from either pipeline makes it fail (revert after checking)
- [x] `task test:unit` passes, and the new guard reads tracked files only — it starts no subprocess and needs no macOS
- [x] `task build`, `task build:wasm` and `task build:web` still succeed and produce what they did before
- [x] The only files this task changes are `Taskfile.yml`, the new `build/darwin/Info.plist`, and the new
      guard files `internal/desktop/bundle_plist_test.go` and `internal/desktop/desktop_isolation_test.go`
      — two files rather than one, because the bundle's plist parity and the CLI/web isolation share no
      reader and the guideline puts unrelated declarations in separate files

**Affected Files/Modules:**
- `Taskfile.yml` — a new `package:desktop` task, depending on `build:desktop`, declared macOS-only
- `build/darwin/Info.plist` — new, tracked; the bundle's plist
- `internal/desktop/bundle_plist_test.go` — new; the bundle plist against the packaging task
- `internal/desktop/desktop_isolation_test.go` — new; the CLI and web build paths against the desktop build

**Patterns to Follow:**
- The bundle assembly sequence, including the ad-hoc `codesign` step: `create:app:bundle` in `internal/commands/build_assets/darwin/Taskfile.yml` of the pinned wails module in the module cache. Read it there rather than from the v3 docs.
- The plist key set: any of `examples/*/build/darwin/Info.plist` in the same module cache.
- Task shape, `deps`, and env conventions: `Taskfile.yml:48-65`.
- Guard shape — build tag, package, relative-path reads, umbrella test with sentence-named subtests, `require`: `internal/desktop/event_names_test.go` and `internal/desktop/binding_names_test.go:1-36`.
- What the CLI release must keep looking like: `.goreleaser.yaml`.

**Testable:** Yes — the parity guard is a config-drift check between `Taskfile.yml`, `.goreleaser.yaml` and the plist. It witnesses that the three artifacts agree; it cannot witness that the bundle launches, which the manual criteria above close.

**Certainty:** medium — `Taskfile.yml:48-65` is the precedent for a task that shells out and sets CGO, but no task here assembles a directory-shaped artifact or shells to `codesign`, and no guard here reads a plist.

**Blast radius:** high — `CFBundleIdentifier` is a contract consumed outside this repository: macOS records it in LaunchServices and US-008's file associations will bind to it, so changing it after anyone has run the app strands that state.

**Verification:** `task package:desktop`; `open bin/emod.app` and observe the window, the Dock tile and the app switcher; `lsappinfo list`, `codesign -dv`, `codesign --verify --deep --strict`, `otool -L` as the criteria state; copy the bundle outside the repo and launch it there; `task test:unit`; `task build && task build:web`. Budget one launch-and-read cycle for the native checks — learning 4 records that reading a framework's source settles what the call does, not what AppKit does around it.

**Depends on:** None

---

### Task 2: Give the bundle the emod icon

**Behavior:** The packaged app carries an emod icon in Finder, the Dock and the app
switcher. A tracked source image under `build/` is the only place the icon is authored;
the `.icns` the bundle carries is derived from it at package time by the pinned
`wails3` tool, and is not tracked.

The repository has no image asset at all today, so the source image is authored as part
of this task. `go tool wails3 generate icons` reads a PNG; the measured result carries a
single 1024×1024 representation, which macOS downsamples. If the Dock icon reads poorly
at small sizes, `iconutil` over an `.iconset` is the macOS-native fallback — but only
after looking, not before.

LaunchServices caches a bundle's icon per path: a rebuilt bundle at the same path can
keep showing the previous icon, so confirm from a freshly-copied path before concluding
the icon is wrong.

**Acceptance Criteria:**
- [ ] A square PNG of at least 1024×1024 is tracked under `build/` and is the only image the packaging reads — `sips -g pixelWidth -g pixelHeight` on it reports both dimensions
- [ ] `task package:desktop` derives the `.icns` from it with `go tool wails3 generate icons` and places it in `bin/emod.app/Contents/Resources/`
- [ ] `build/darwin/Info.plist` names that file in `CFBundleIconFile`, and the Task 1 guard gains a subtest requiring the name in the plist and the name the package task writes to agree; changing either alone makes it fail (revert after checking)
- [ ] The derived `.icns` is ignored by git — `git check-ignore` on its path exits 0
- [ ] After `task package:desktop`, `bin/emod.app` shows the icon in Finder, and a launched copy shows it in the Dock and in the app switcher
- [ ] `task test:unit` passes
- [ ] The only files this task changes are the new tracked PNG, `build/darwin/Info.plist`, `Taskfile.yml`, `.gitignore`, and the guard file Task 1 added

**Affected Files/Modules:**
- `build/appicon.png` — new, tracked (name it for the `wails3 generate icons` default input if you keep that default)
- `build/darwin/Info.plist` — gains `CFBundleIconFile`
- `Taskfile.yml` — icon generation and the Resources copy inside `package:desktop`
- `.gitignore` — the derived `.icns`
- `internal/desktop/` — the guard file from Task 1 gains the icon subtest

**Patterns to Follow:**
- Invoking the pinned CLI through the module rather than `PATH`: `Taskfile.yml:58`.
- The icns and `Assets.car` handling the bundle expects: `create:app:bundle` in the wails module cache (same file as Task 1); note its default output name differs from `wails3 generate icons`' own default, so name the file explicitly on both sides.
- The bundle assembly this extends: `task:1`.
- Guard shape: `internal/desktop/event_names_test.go`.

**Testable:** Yes — the plist-to-package-task name parity is checkable in `test:unit`. That the icon actually renders in the Dock is native state no suite can witness, and is closed by the observation criteria above.

**Certainty:** medium — `Taskfile.yml:58` is the precedent for calling `go tool wails3` from a task and the icns route was run during exploration, but nothing in this repo has ever observed Dock or app-switcher state, and LaunchServices caches an icon per bundle path.

**Blast radius:** low — a wrong icon is visible immediately and costs a rebuild.

**Verification:** `task package:desktop`; look at the bundle in Finder; launch it and look at the Dock tile and ⌘-Tab; confirm from a copy at a fresh path if the icon looks stale; `git check-ignore` on the derived file; `task test:unit`.

**Depends on:** Task 1

---

### Task 3: Document packaging and the first-launch quarantine step in the README

**Behavior:** The README tells a reader how to produce the app and what to do the first
time a copy that arrived through a browser download refuses to open — why it happens,
which two ways out there are, and that it is once per machine rather than once per launch.
The paragraph listing what the desktop app cannot do yet stops claiming there is no
packaged `.app`.

**Acceptance Criteria:**
- [ ] The existing `### Desktop app` section (README.md:371) documents the packaging command and the bundle it produces
- [ ] It documents the first-launch step for a downloaded copy — **Open Anyway** under System Settings → Privacy & Security, or `xattr -dr com.apple.quarantine` against the bundle — and says that either one is enough
- [ ] It says why the block appears: the app is unsigned and un-notarized, so Gatekeeper stops a copy carrying `com.apple.quarantine`, which is what a browser download attaches and what a locally-built bundle does not have
- [ ] It says the step is one-time per machine
- [ ] The "What it does not do yet" paragraph (README.md:452-455) no longer says there is no packaged `.app`
- [ ] The only file this task changes is `README.md`

**Affected Files/Modules:**
- `README.md` — `### Desktop app` (371) and the closing paragraph (452-455)

**Patterns to Follow:**
- Register, density and heading level of the surrounding prose: `README.md:371-455` — the section states observed behaviour in full sentences and does not hedge.
- The claims about Gatekeeper, ad-hoc signing and quarantine: proposal §7.3 (`docs/proposals/emod-desktop-proposal.md`).
- The bundle and command being documented: `task:1`.

**Testable:** No — this is prose in an existing section; a regex over README wording would pin the sentences a later edit is meant to improve.

**Certainty:** high — `README.md:371-455` already documents the desktop app in exactly this register, and the Gatekeeper claims are stated by proposal §7.3 rather than decided here.

**Blast radius:** low — a README edit, reversible in one commit.

**Verification:** Read the section back. Then check that the documented step works rather than assuming it: copy the bundle to another location, attach `com.apple.quarantine` to the copy with `xattr -w` to stand in for a browser download, confirm the launch is blocked, run the documented `xattr -dr` command, and confirm it then launches. No second Mac is available in this environment, so that simulation plus Task 1's linkage evidence is what closes the story's "copying to another Mac" criterion — report it as the simulation it is rather than implying a second machine was used. If a hand-attached attribute does not reproduce the block, report that and leave the documented step as §7.3 states it.

**Depends on:** Task 1

## Summary

**Total tasks:** 3.

**Ordering rationale:** dependency-first, and the risk is concentrated in Task 1 — the
bundle either launches from Finder or it does not, and everything else decorates or
describes it. Task 2 adds the one artifact that has no precedent in this repository at
all (an image), and Task 3 documents what the first two produced and verifies the
documented escape hatch actually works.

**Story criteria coverage:**

| Story criterion | Task | How it closes |
|---|---|---|
| 1. A single build command produces an app bundle carrying the emod name and icon | 1 (name), 2 (icon) | `task package:desktop`; `CFBundleName`; `CFBundleIconFile` |
| 2. Launches from Finder with no Go toolchain, no Node, no emod CLI | 1 | `otool -L` showing only system frameworks, the embedded frontend, plus a manual Finder launch. No unit test can witness this. |
| 3. Copying to another Mac works after one documented step | 1 (relocatable bundle), 3 (the step, and the quarantine simulation) | A second Mac is not available; the evidence is the relocation run plus a hand-attached quarantine attribute |
| 4. README documents the step, why it appears, and that it is one-time per machine | 3 | The existing `### Desktop app` section, confirmed to exist at README.md:371 |
| 5. The emod name and icon in the Dock and the app switcher | 1 (name), 2 (icon) | `lsappinfo list` for the name; observation for the icon |
| 6. Building and releasing the CLI and the web viewer are unaffected | 1 | The isolation guard, plus re-running `task build` / `build:wasm` / `build:web` |

**Nothing is deferred out of the story's criteria.** The two that no automated check can
close (2 and 3) are stated as inspection plus a named manual run rather than dressed up
as testable, and the guard for criterion 6 is written so that removing the `-e` from a
`go list` pipeline turns it red — the exact regression learning 2 records as invisible to
whoever introduces it.

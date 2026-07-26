# Proposal: A desktop distribution for `emod`

**Status:** Draft for review
**Date:** 2026-07-26
**Repo:** `github.com/hpcsc/emod` (analysed at branch `us-002-construct-descriptions`)
**Author:** Drafted with Claude Code

---

## 1. Summary

Adding a desktop app (Wails) to `emod` is **low-risk and unusually cheap**, because
the repo has already paid the structural cost without meaning to: the frontend is
framework-free, the Go pipeline is already transport-agnostic, and the JS↔Go
boundary is a single 71-line module with three importers.

**Recommendation: do it, as a third distribution behind a shared platform
adapter — not as a fork of the viewer.**

**Decided (2026-07-26):** target **Wails v3**; **no code signing or notarization**
(unsigned local/CI builds, no Apple Developer account, no App Store); ship
**macOS + Linux**, with Linux pinned to an **Ubuntu 24.04+ floor**.

The work splits cleanly:

| | Effort | Risk |
|---|---|---|
| Restructure + platform adapter (no behaviour change) | ~1.5 days | Low — pure refactor, tests guard it |
| Wails v3 shell + desktop adapter | ~1.5 days | Low — mirrors existing WASM shim |
| Native file I/O (the actual user-facing win) | ~1.5 days | Low |
| Packaging (`wails3 package`, unsigned) | ~0.5 day | Low |
| CI: macOS + Linux artifacts | ~0.5 day | Low — **$0**, public repo (§7.5) |

Roughly **5.5 days** to a working, CI-built desktop app for macOS and Linux.
Dropping signing removes what was the single largest source of schedule risk; the
remaining risk is Wails v3's alpha status (§9), which is managed by pinning.

---

## 2. Why this is cheap: what already exists

This is not a greenfield GUI project. Four properties of the current codebase do
the heavy lifting.

### 2.1 The core is already surface-agnostic

`internal/{lexer,parser,validator,linter,export,formatter,importer,diagram}` have no
knowledge of how they're invoked. They already serve **four** consumers today —
CLI, LSP (`internal/lsp/`), WASM, and the embedded viewer. Desktop becomes a fifth
with no changes below the pipeline layer.

### 2.2 The pipeline was written for exactly this

`internal/wasm/pipeline.go` — despite the package name — is pure Go types. Its own
doc comment states the intent:

> Package wasm extracts the emod pipeline (lex → parse → validate → lint → export)
> into functions that accept and return standard Go types, enabling testing
> independent of `syscall/js`.

`cmd/emod-wasm/main.go` is a thin `js.FuncOf` wrapper over it. A Wails v3 **service**
is the *same wrapper with a different calling convention* — the three entry points
map 1:1:

| WASM global | Pipeline function | v3 service method |
|---|---|---|
| `parseEmod` | `pipeline.RunPipelineExportDiagram` | `(*ModelService).ParseEmod` |
| `exportJSON` | `pipeline.RunPipelineExportJSON` | `(*ModelService).ExportJSON` |
| `exportEmod` | `pipeline.ExportEmodJSON` | `(*ModelService).ExportEmod` |

In v3 a service is a plain struct whose exported methods are callable from the
frontend — no registration boilerplate beyond listing it in `application.Options`.

### 2.3 The frontend has no framework and no build step

`internal/viewer/static/` is ~3,700 lines of plain ES modules — `renderer.js`,
`layout.js`, `interaction.js`, `minimap.js`, `model.js`, `store.js`, `ui.js`,
`bus.js`, `ctx-actions.js`, `config.js`. No React, no bundler, no transpile.
`npm` appears only for vitest.

Consequence: **the same files load unmodified in a Wails webview.** There is no
build-tooling migration.

### 2.4 Generated-vs-source discipline already exists

`.gitignore` contains `/web` and `internal/viewer/generated/`. Both bundles are
build outputs assembled by `task build:web` from one source of truth. Desktop
slots in as a third assembly target using the identical pattern.

---

## 3. The key framing: 3 distributions, 2 runtimes

CLI-viewer and web are **not** different platforms. Both are a browser loading the
same WASM binary and the same JS. They differ only in *delivery*:

- **CLI** (`emod diagram --viewer`): Go embeds the assets, serves them from
  `127.0.0.1:<port>` via `internal/viewer/serve.go`, injects state server-side.
- **Web** (GitHub Pages): the same assets copied to `/web`, hosted statically, no
  server-side injection.

Desktop is the only genuinely new **runtime**: native Go calls, no WASM, no HTTP,
real filesystem access.

So the adapter has **two implementations**, not three. CLI and web share one.

```
                      ┌──────────────────────────────┐
                      │  internal/{lexer,parser,     │  pure Go domain,
                      │  validator,linter,export,    │  surface-agnostic
                      │  formatter,importer,diagram} │
                      └──────────────┬───────────────┘
                                     │
                      ┌──────────────▼───────────────┐
                      │  internal/pipeline           │  lex→parse→validate
                      │  (today: internal/wasm)      │  →lint→export
                      └──────────────┬───────────────┘
                                     │
              ┌──────────────────────┼──────────────────────┐
              │                      │                      │
    ┌─────────▼────────┐   ┌─────────▼────────┐   ┌─────────▼────────┐
    │ cmd/emod         │   │ cmd/emod-wasm    │   │ cmd/emod-desktop │
    │ CLI + LSP +      │   │ js.FuncOf shims  │   │ Wails bindings   │
    │ localhost viewer │   │                  │   │                  │
    │ CGO=0            │   │ GOOS=js CGO=0    │   │ CGO=1            │
    └─────────┬────────┘   └─────────┬────────┘   └─────────┬────────┘
              │                      │                      │
              └──────────┬───────────┘                      │
                  browser runtime                     native runtime
                  (platform.browser.js)             (platform.desktop.js)
                         │                                  │
                    ┌────▼──────────────────────────────────▼────┐
                    │  internal/frontend/static/  (~3.7k lines)  │
                    │  ONE source of truth, zero forks           │
                    └────────────────────────────────────────────┘
```

---

## 4. The platform seam

### 4.1 Complete inventory of platform-coupled code

I audited every file in `internal/viewer/static/`. The entire coupling surface is
**five touch points across three files**. Everything else is pure.

| # | What | Location | Browser today | Desktop |
|---|---|---|---|---|
| 1 | Go bridge | `static/wasm.js` (whole file, 71 lines) | `fetch('generated/emod.wasm')` + `globalThis.*` | Wails bindings |
| 2 | Drag-drop open | `viewer.js:165–183` | `dataTransfer.files[0]` + `FileReader` | native path, `os.ReadFile` |
| 3 | Export/save | `viewer.js:203–216` | `Blob` + `a.download` | native save dialog, save-in-place |
| 4 | Initial state | `viewer.js:266` `INITIAL_DATA` global | injected by `serve.go` `buildHTML` | bindings call at startup |
| 5 | Readiness gate | `viewer.js:277,282`; `model.js:105`; `emod-export.js:4` | async WASM load | already-resolved promise |

### 4.2 The three importers of `wasm.js`

Worth stating precisely, because the refactor touches exactly these:

```
viewer.js:11        import { ready, isReady } from './wasm.js';        // static
emod-export.js:1    import { ready, exportEmod } from './wasm.js';     // static
model.js:104        import('./wasm.js').then(...)                      // DYNAMIC
```

`model.js` uses a **deferred dynamic import**, with the comment *"dynamic import to
defer init side effects"*. That is a tell: the current design already treats the
WASM module as something with awkward initialization side effects. Those side
effects vanish entirely in the desktop build, and the adapter lets `model.js` keep
the deferral without caring which implementation it gets.

### 4.3 Proposed interface

One new module, `platform.js`, with a stable contract:

```js
// The contract every platform implementation satisfies.
export const ready;              // Promise<void>  — resolves when the core is callable
export const isReady;            // boolean
export function parseEmod(source);          // → Promise<{diagnostics, diagram}>
export function exportEmod(diagram);        // → Promise<string>
export function openFile();                 // → Promise<{path, content} | null>
export function saveFile(name, content);    // → Promise<{path} | null>
export function initialState();             // → Promise<object | null>
```

The first four already exist in `wasm.js` and keep their exact JSON-string
semantics. The last three are new abstractions over touch points 2–4.

### 4.4 Before / after

**Before** (`emod-export.js`, browser-only):

```js
import { ready, exportEmod } from './wasm.js';
```

**After** (identical body, platform-neutral import):

```js
import { ready, exportEmod } from './platform.js';
```

**Before** (`viewer.js:203–216`, forced browser download):

```js
Export.exportToEmodString(store).then(function(content) {
  var blob = new Blob([content], { type: "text/plain" });
  var url = URL.createObjectURL(blob);
  var a = document.createElement("a");
  a.href = url;
  a.download = (store.modelName || "diagram") + ".emod";
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(url);
})
```

**After** (one call, each platform does the right thing):

```js
Export.exportToEmodString(store).then(function(content) {
  return saveFile((store.modelName || "diagram") + ".emod", content);
})
```

`platform.browser.js` keeps the blob dance verbatim. `platform.desktop.js` calls a
native save dialog and writes through Go — including **save back to the file you
opened**, which the browser fundamentally cannot do.

### 4.5 Selecting the implementation

No bundler exists, so module resolution happens at **assembly time**. This matches
the idiom `task build:web` already uses (it copies `static/*` and even does
`mv viewer.html index.html`):

```
build        (CLI)     → cp platform.browser.js → internal/frontend/static/platform.js
build:web    (web)     → cp platform.browser.js → web/static/platform.js
build:desktop          → cp platform.desktop.js → build/frontend/platform.js
```

Result: no conditionals in shipped code, no dead branches, no desktop code in the
Pages deploy.

**Alternative considered:** runtime detection — a single `platform.js` could branch
on whether the Wails runtime is present. Simpler (no build step, nothing to drift)
but ships both implementations everywhere and puts environment sniffing into the
hot path. *Recommendation: assembly-time copy*, matching the existing build idiom;
revisit if the copy step proves annoying in dev.

### 4.6 The Wails v3 shell, concretely

`cmd/emod-desktop/main.go` — the whole shell is roughly this:

```go
//go:embed all:frontend
var assets embed.FS

func main() {
    app := application.New(application.Options{
        Name: "emod",
        Assets: application.AssetOptions{
            Handler: application.AssetFileServerFS(assets),
        },
        Services: []application.Service{
            application.NewService(&ModelService{}),
        },
    })

    app.Window.NewWithOptions(application.WebviewWindowOptions{
        Title:  "emod",
        Width:  1400,
        Height: 900,
    })

    if err := app.Run(); err != nil {
        log.Fatal(err)
    }
}
```

The service is a direct transliteration of `cmd/emod-wasm/main.go`, minus
`js.Value`:

```go
type ModelService struct{}

func (s *ModelService) ParseEmod(source string) (string, error) {
    b, err := pipeline.RunPipelineExportDiagram(source)
    return string(b), err
}

func (s *ModelService) ExportEmod(diagramJSON string) string {
    return pipeline.ExportEmodJSON(diagramJSON)
}
```

Note the WASM shims must hand JSON *strings* across the boundary and wrap errors in
an `{"error": "..."}` envelope (`pipeline.ErrorJSON`). v3 marshals real Go types and
propagates `error` as a rejected promise — so the desktop side could be cleaner.
**Keep the string/envelope contract anyway in v1**, so both adapters satisfy an
identical interface and `model.js` / `emod-export.js` stay untouched. Revisit only
if the desktop surface diverges later.

**Native dialogs** are built in via `DialogManager` — this is what `platform.desktop.js`
sits on:

```go
func (s *ModelService) OpenFile() (FileResult, error) {
    path, err := application.Get().Dialog.OpenFile().
        SetTitle("Open model").
        AddFilter("emod model", "*.emod;*.json").
        PromptForSingleSelection()
    if err != nil || path == "" {
        return FileResult{}, err
    }
    content, err := os.ReadFile(path)
    return FileResult{Path: path, Content: string(content)}, err
}
```

`SaveFile()` is the mirror image, and because the service returns the real `path`,
**save-in-place** — the thing a browser fundamentally cannot do — is just remembering
that string in `store`.

**Generated bindings.** `wails3 generate bindings` emits JS under
`frontend/bindings/...`, imported like:

```js
import { ModelService } from '../bindings/emod/modelservice.js';
const result = await ModelService.ParseEmod(source);
```

These are generated artifacts — never hand-edited, and `.gitignore`'d like
`internal/viewer/generated/` already is. `platform.desktop.js` is the only file that
imports them, keeping the generated surface behind the adapter.

---

## 5. Package reorganization

Two renames, both mechanical, both making the sharing legible:

### 5.1 `internal/wasm` → `internal/pipeline`

The package doc already says it exists to be *independent* of `syscall/js`. Once
desktop calls it natively, the name actively misleads. Pure rename; no logic
changes.

### 5.2 Split `internal/viewer`

`internal/viewer` currently conflates two unrelated concerns:

- **shared frontend assets** (`static/`, `embed.go`, `tests/`, `package.json`) —
  used by all three distributions
- **the CLI's HTTP delivery** (`serve.go`, `serve_test.go`) — used by the CLI only

Proposed:

```
internal/frontend/     static/, embed.go, tests/, package.json, vitest.config.js
internal/viewer/       serve.go, serve_test.go        (CLI delivery only)
```

This makes it structurally obvious that the desktop app depends on `frontend` and
*not* on `viewer` — no localhost server, no port, no `buildHTML` templating.

### 5.3 Resulting layout

```
cmd/
  emod/                CLI + LSP + localhost viewer    CGO=0
  emod-wasm/           browser shim                    GOOS=js
  emod-desktop/        Wails v3 shell                  CGO=1   ← new
    main.go              application.New + window
    service.go           ModelService (pipeline + dialogs)
    frontend/            assembled at build time (gitignored)
      bindings/          wails3 generate bindings output (gitignored)
internal/
  pipeline/            shared orchestration            ← renamed from wasm/
  frontend/            shared UI, single source        ← extracted from viewer/
    static/
      platform.js          (assembled per target)
      platform.browser.js  ← new (today's wasm.js + blob/FileReader logic)
      platform.desktop.js  ← new (v3 bindings + native dialogs)
  viewer/              serve.go only — CLI delivery
  lexer/ parser/ validator/ linter/ export/ formatter/ importer/ diagram/ lsp/ ...
```

---

## 6. Build pipeline

**Fortunate alignment: Wails v3 orchestrates its own builds with go-task —
the same tool this repo already uses.** v3 generates a `Taskfile.yml` where icon
generation, `Info.plist` handling, binding generation and packaging are ordinary,
editable task steps rather than opaque CLI magic. So the desktop build composes
into the existing `Taskfile.yml` instead of sitting beside it as a foreign system.

Existing tasks keep working unchanged:

```yaml
build:            # unchanged — CLI, CGO_ENABLED=0, deps on build:wasm
build:wasm:       # unchanged — GOOS=js GOARCH=wasm
build:web:        # + copy platform.browser.js → platform.js
build:desktop:    # NEW — assemble frontend, wails3 generate bindings,
                  #       wails3 build; CGO_ENABLED=1
package:desktop:  # NEW — wails3 package → .app / AppImage (unsigned)
```

The desktop target notably **does not depend on `build:wasm`** — it calls Go
natively, dropping the multi-MB `.wasm` payload and the `wasm_exec.js` glue.

`mise.toml` gains `wails3` alongside `tree-sitter-cli`, `cue`, `goreleaser`, `task`.
**Pin the exact alpha version** (see §9) — `mise.toml` already pins every other tool
exactly, so this matches existing practice.

---

## 7. Release / distribution

With signing and notarization off the table, this section collapses from "the main
risk" to "a build task". One structural constraint remains.

### 7.1 The CGO constraint (unavoidable)

Everything shipped today is `CGO_ENABLED=0`, and `.goreleaser.yaml` cross-compiles
both `linux` and `darwin` from a single `ubuntu-latest` runner. Wails requires CGO
(WebKit on macOS, WebKitGTK on Linux). **That cross-compile cannot produce a Wails
binary** — a desktop artifact must be built *on* its target OS.

Mitigation is quarantine, not migration: keep desktop in its own build target and
its own CI job, so the existing CLI release is untouched.

| Distribution | Runner | CGO | Artifact | Change |
|---|---|---|---|---|
| CLI | `ubuntu-latest`, cross-compiled | `0` | `tar.gz` via goreleaser | **none** |
| Web | `ubuntu-latest` | `0` (`GOOS=js`) | GitHub Pages | **none** |
| Desktop | `macos-latest` **+** `ubuntu-24.04` | `1` | `.app` zip, Linux binary | new job |

Note goreleaser isn't needed for desktop — `wails3 package` produces the bundles
directly. Leave `.goreleaser.yaml` alone.

### 7.2 Recommended: start with no release pipeline at all

Given this is a personal project with no App Store ambitions, **the simplest thing
that works is to skip CI for desktop entirely at first**:

```
task build:desktop      # builds and runs locally on your Mac
```

No runners, no secrets, no matrix, no macOS CI minutes. Add a CI job only when you
actually want to hand the app to someone else. This makes Phase 4 nearly free, and
is the reason the estimate dropped from ~1–2 days to ~0.5 day.

When you do want shareable artifacts: one workflow, `macos-latest`, `wails3 package`,
`upload-artifact`. Your existing **On Demand Build** workflow is already exactly this
shape (`workflow_dispatch` → build → `upload-artifact`) and is the natural place to
add it.

### 7.3 What "unsigned" actually costs your users

Worth being clear-eyed, since this is the tradeoff being accepted:

- **Ad-hoc signing is automatic and sufficient to *run*.** On Apple Silicon every
  binary must carry at least an ad-hoc signature; the Go linker applies one for
  `darwin/arm64` automatically. So a locally-built app runs fine on your own machine —
  no ceremony at all.
- **Gatekeeper only bites on *downloaded* apps.** Files fetched by a browser get a
  `com.apple.quarantine` attribute. An unsigned, un-notarized app then gets blocked
  on first launch, and the user must either use **Open Anyway** (System Settings →
  Privacy & Security) or run:
  ```
  xattr -dr com.apple.quarantine /Applications/emod.app
  ```
- **This is a per-user, one-time annoyance**, and entirely acceptable for a dev tool
  distributed to people who already install Go CLIs.

The escape hatch, if it ever matters: **Developer ID + notarization is a separate
thing from the App Store** ($99/yr, no App Store review) and removes the friction
entirely. Deliberately deferred — nothing in this design blocks adding it later,
since it's a signing step appended to the packaging task.

### 7.4 Other notes

- **Windows** stays out of scope (`.goreleaser.yaml` targets linux+darwin only). v3
  supports it via WebView2; add later if wanted.

### 7.5 CI cost for macOS + Linux binaries (unsigned)

**Decided (2026-07-26): both macOS and Linux are in scope.**

**Dollar cost: $0.** Standard GitHub-hosted runners are **free and unmetered on
public repositories**, macOS included — and `hpcsc/emod` is public. The widely-quoted
10× macOS multiplier and $0.062/min rate apply only to *private* repos drawing on a
monthly allowance.

> If the repo ever goes private this flips sharply: a 5-minute macOS build consumes
> 50 minutes of a 2,000-minute Free-plan allowance (~40 builds/month before
> exhaustion). Worth knowing; not worth designing around today.

**Wall-clock cost: zero on the normal loop, if placed correctly.** CI currently runs
**~2m50s**. Put desktop builds behind `workflow_dispatch` — the existing **On Demand
Build** workflow is already exactly this shape — and everyday pushes are unaffected.

| Placement | Cost to push loop | Verdict |
|---|---|---|
| Every push | 2 extra jobs per commit, incl. docs-only | ✗ noise |
| `workflow_dispatch` (On Demand Build) | none | ✓ **recommended** |
| On tags (Release) | none | ✓ when cutting artifacts |

Estimated runtime: **~3–4 min Linux** (incl. ~1 min apt), **~4–6 min macOS** (CGO
linking is slower). They run in parallel.

**Setup effort: ~40 lines of YAML, ~half a day** including first-run debugging —
almost entirely Linux WebKit deps.

```yaml
strategy:
  matrix:
    include:
      - os: macos-latest    # arm64
      - os: ubuntu-24.04    # pinned, not -latest — see support floor below
runs-on: ${{ matrix.os }}
steps:
  - uses: actions/checkout@v6
  - uses: actions/setup-go@v6
    with: { go-version-file: 'go.mod' }
  - if: runner.os == 'Linux'
    run: |
      sudo apt-get update
      sudo apt-get install -y libgtk-4-dev libwebkitgtk-6.0-dev
  - run: go install github.com/wailsapp/wails/v3/cmd/wails3@<PINNED>
  - run: task package:desktop
  - uses: actions/upload-artifact@v7
```

#### The Linux compatibility floor

**Decided (2026-07-26): Ubuntu 24.04+ only.** Build on the GTK4 / WebKitGTK 6.0
stack that Wails v3 defaults to, with no legacy fallback.

This means:

- **No `-tags gtk3`, no second matrix entry, no dual-stack testing.** The GTK3 /
  WebKit2GTK 4.1 opt-in exists for Ubuntu 22.04 LTS, Debian 12, Fedora ≤39 and
  RHEL 9 — all explicitly out of scope.
- **The build is the default path**, which is the best-supported and least
  bug-prone configuration in an alpha project.
- WebKit links dynamically, so **artifacts will not run on 22.04.** Accepted.

**Pin `ubuntu-24.04` explicitly — do not use `ubuntu-latest`.** The label will
eventually roll to 26.04, which would silently raise the floor for users without
any change on your side. The support floor should move when you decide it moves:

```yaml
- os: ubuntu-24.04    # NOT ubuntu-latest — pins the support floor
```

Two follow-ups this implies:

1. **Document the runtime requirement** in the README — Linux users need the
   *runtime* libraries (`libgtk-4-1`, `libwebkitgtk-6.0-4`), not the `-dev` packages
   the CI job installs.
2. **Confirm package names with `wails3 doctor`** on the runner. They have churned
   (`webkit2gtk-4.0` → `4.1` → `webkitgtk-6.0`) and are a recurring source of broken
   Wails builds.

Escape hatch if it ever matters: adding a 22.04 + `-tags gtk3` job later is an extra
matrix entry and a build tag — still $0, and nothing in this design blocks it.

#### Ongoing cost

Runner images update and the WebKit dependency naming has a history of breaking
builds. Budget occasional unplanned fixes — a **Linux-specific** cost. macOS needs
nothing beyond the preinstalled toolchain.

---

## 8. Testing strategy

| Layer | Tool | Change |
|---|---|---|
| Go core & pipeline | `go test` | none — already covers all surfaces |
| Frontend units | vitest, `internal/frontend/tests/` | mock `platform.js` instead of `wasm.js` |
| Browser e2e | Playwright, `e2e-viewer/` | none — still runs the web bundle |
| CLI e2e | Docker, `e2e/` | none |
| Desktop e2e | — | **gap, see below** |

### 8.1 Frontend tests

Seven test files currently `vi.mock('../static/wasm.js', ...)`:
`wasm.test.js`, `wasm.init-success.test.js`, `wasm.init-http-error.test.js`,
`wasm.fallback.test.js`, `viewer.test.js`, `emod-export.test.js`, `model.test.js`.

They already mock at **precisely the boundary being introduced**, so the change is
a redirect of the mock target. The four `wasm.*.test.js` files — which test WASM
fetch/instantiate/fallback specifically — should move to `platform.browser.test.js`,
since that behaviour is browser-only.

### 8.2 The desktop e2e gap

Wails has no first-class Playwright story. Three options:

1. **Accept manual smoke testing** of the shell. Defensible: the desktop-specific
   surface is only the five touch points, and everything below is covered by the
   existing browser suite. *Recommended initially.*
2. Go-level tests against the bindings struct — cheap, catches contract drift,
   doesn't exercise the webview.
3. Full driver automation — high effort, low marginal value at this stage.

---

## 9. Risks and open questions

| Risk | Severity | Mitigation |
|---|---|---|
| **Wails v3 is alpha** — API churn between releases | **Medium** | Pin the exact version in `go.mod` **and** `mise.toml`; read the changelog before any bump. See below. |
| CGO breaks a shared build path | Low | Quarantined in its own `cmd/` + build task; CLI never links CGO |
| Frontend forks over time | Low | Single source enforced by build assembly; no `platform.*` logic in shared modules |
| Refactor destabilizes the viewer | Low | Phase 1 is behaviour-preserving with full test coverage |
| Desktop regressions go unnoticed | Medium | Accepted initially; revisit if the desktop surface grows |
| Generated bindings drift from services | Low | `wails3 generate bindings` runs in `build:desktop`; only `platform.desktop.js` imports them |

### 9.1 On the v3 alpha risk

Verified at the time of writing: the current release is **`v3.0.0-alpha2.117`
(published 8 Jul 2026)** — v3 has been in alpha for a long stretch, with the API
described as reasonably stable and production apps shipping on it, but nightly
alpha tags and moving details.

What this means practically:

- **Pin exactly.** `go.mod` gets the precise `v3.0.0-alphaN.NNN`; `mise.toml` gets
  the matching `wails3` CLI version. Never float. This repo already pins every tool
  exactly, so it's consistent with existing practice.
- **Use the Manager API style** — `app.Window.New*`, `app.Dialog.*`, `app.Event.*`,
  `app.Menu.*`. Earlier alphas exposed flat top-level functions that no longer
  exist, so older blog posts and tutorials will mislead. Some third-party v3
  references are already stale — one I checked while writing this claimed v3 has no
  built-in dialog runtime, which is **wrong** (`DialogManager` is documented in the
  package API).
- **Blast radius is small.** All v3-specific code lives in `cmd/emod-desktop/` plus
  `platform.desktop.js`. An API break touches ~100 lines and cannot reach the CLI,
  the web build, or `internal/`.

**Resolved (2026-07-26):**

1. ~~Wails v2 or v3?~~ → **v3**, pinned to a specific alpha.
2. ~~Signed/notarized for v1?~~ → **No.** Unsigned; no Apple Developer account. See §7.3.

3. ~~Linux desktop, or macOS-only?~~ → **Both macOS and Linux.** CI cost analysed
   in §7.5 ($0 — public repo).
4. ~~Linux support floor?~~ → **Ubuntu 24.04+ only**, GTK4 / WebKitGTK 6.0 default,
   no `-tags gtk3` fallback. Pin `ubuntu-24.04` in CI, not `ubuntu-latest`. See §7.5.

**Still open for the reviewer:**

5. **Is Windows in scope?** Currently excluded.
6. **Does the desktop app need the LSP** embedded (edit `.emod` text in-app with
   diagnostics), or is it diagram-only? Diagram-only assumed here.
7. **Should `internal/frontend` become a published npm-style package?** Assumed no —
   copy-assembly is sufficient and simpler.

---

## 10. Phased plan

Each phase is independently mergeable and leaves `main` shippable.

### Phase 0 — Restructure (~0.5 day)
Rename `internal/wasm` → `internal/pipeline`. Split `internal/viewer` into
`internal/frontend` + `internal/viewer`. Update `Taskfile.yml` paths, CI
`node-version-file`, embed directives.
**Exit:** all existing tests green, zero behaviour change.

### Phase 1 — Platform adapter (~1 day)
Introduce `platform.js` contract. Move today's `wasm.js` to `platform.browser.js`
and fold in the `FileReader` / `Blob` logic behind `openFile`/`saveFile`. Repoint
the three importers. Redirect vitest mocks.
**Exit:** CLI viewer and web bundle behave identically to today; no desktop code yet.

### Phase 2 — Wails v3 shell (~1.5 days)
Add `cmd/emod-desktop` with `ModelService` over `internal/pipeline` (§4.6). Add
`platform.desktop.js` against the generated bindings. Add `task build:desktop`.
Pin `wails3` in `mise.toml` and the module in `go.mod`.
**Exit:** a desktop window renders a diagram from pasted source. Feature parity
with the browser, nothing more.

### Phase 3 — Native file I/O (~1.5 days)
The actual user-facing payoff, via `app.Dialog.*`: native open/save dialogs,
**save-in-place**, recent-files, `.emod` file association, menu bar, optional
file-watch reload.
**Exit:** the desktop app is meaningfully better than the browser viewer.

### Phase 4 — Packaging (~0.5 day)
`task package:desktop` wrapping `wails3 package` → unsigned `.app`. No CI, no
signing, no secrets.
**Exit:** a double-clickable app on your own machine.

### Phase 5 — CI artifacts (~0.5 day)
A 2-job matrix (`macos-latest` + pinned `ubuntu-24.04`) on the existing **On Demand
Build** workflow, uploading packaged apps. Document the Linux runtime deps in the
README. See §7.5 for cost and the YAML sketch.
**Exit:** downloadable artifacts for both platforms. macOS recipients do the
one-time `xattr`/Open-Anyway dance (§7.3); Linux requires 24.04+.

---

## 11. Bottom line

The expensive parts of "add a desktop app" — extracting a reusable core, removing
framework lock-in, defining a clean UI/logic boundary — are **already done** in this
repo, apparently as a side effect of supporting WASM well. Adding Wails is mostly
writing ~40 lines of adapter per platform.

Two decisions removed most of the remaining cost. Skipping signing deletes the
schedule's biggest unknown and turns release into a build task. Choosing v3 buys a
first-class `DialogManager` for the native file I/O that justifies the port, and
a build system this repo already speaks — at the price of tracking an alpha, which
pinning contains to ~100 lines of shell code.

The architecture ends up as: **one core, one frontend, one adapter interface, three
thin shells** — with the only real duplication being the per-platform adapter, and
the only structural constraint a CGO-shaped hole isolated to `cmd/emod-desktop/`.

---

## Appendix: sources

- [Wails v3 — pkg.go.dev `pkg/application`](https://pkg.go.dev/github.com/wailsapp/wails/v3/pkg/application) — `DialogManager`, manager API, version `v3.0.0-alpha2.117` (8 Jul 2026)
- [Wails v3 documentation](https://v3.wails.io/) and [changelog](https://v3.wails.io/changelog/)
- [Wails v3 Dialogs API](https://v3alpha.wails.io/reference/dialogs/)
- [`wails3` CLI reference](https://pkg.go.dev/github.com/wailsapp/wails/v3/cmd/wails3)
- [spf13/go-skills — Wails skill](https://github.com/spf13/go-skills/blob/main/wails/SKILL.md) — useful for v3 idioms, but its dialog claim is inaccurate (see §9.1)
- [Wails releases](https://github.com/wailsapp/wails/releases)
- [Wails v3 installation / Linux deps](https://v3.wails.io/quick-start/installation/) — GTK4 + WebKitGTK 6.0 default, `-tags gtk3` fallback
- [wails#4339 — v3 build dependencies per distribution](https://github.com/wailsapp/wails/issues/4339)
- [wails#3513 / #3581 — libwebkit2gtk churn on Ubuntu](https://github.com/wailsapp/wails/issues/3513)
- [GitHub Actions billing docs](https://docs.github.com/billing/managing-billing-for-github-actions/about-billing-for-github-actions) — standard runners free on public repos
- [GitHub Actions 2026 pricing changes](https://github.com/resources/insights/2026-pricing-changes-for-github-actions)

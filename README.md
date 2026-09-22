# emod

Event Modeling DSL & visualization CLI tool. Write `.emod` files to model event-driven architectures, validate against anti-patterns, and generate diagrams.

## Quick Start

### Install

```bash
go install github.com/hpcsc/emod/cmd/emod@latest
```

Or build from source:

```bash
git clone https://github.com/hpcsc/emod.git
cd emod
go build -o ./bin/emod ./cmd/emod
```

### Version

```bash
emod version   # print the tag this build came from
```

A release carries the tag it was built from. A build from source has no tag, so
`emod version` prints the commit instead, and marks it `-dirty` when the working
tree held changes.

### Update

`emod update` replaces the running binary with the latest release on GitHub. It
checks the download against the release's `checksums.txt` before it writes, and
it renames the new binary over the old one, so the binary on disk is never half
written.

```bash
emod update               # install the latest release
emod update --check       # say what an update installs, and install nothing
emod update --prerelease  # install the latest build of main
emod update --force       # replace a build from a commit
```

`emod update` stops on a build from a commit and names `--force` instead of
replacing it. Each push to main publishes a prerelease, and the repository keeps
the 5 newest. `GITHUB_TOKEN` or `GH_TOKEN` in the environment authorises the
requests to GitHub.

### Write a model

Create `reservation.emod`:

```emod
emod = 1

model "Hotel Reservation" {
}

actor "Guest" {
}

context "Reservations" {
  aggregate "Reservation" {
    slice "Reserve a Room" {
      trigger "Reservation Form" {
        actor = Guest
        reads = AvailableRoomsView
      }

      command "ReserveRoom" {
        fields {
          roomId    = required(string)
          guestName = required(string)
          checkIn   = required(date)
          checkOut  = required(date)
        }
      }

      event "RoomReserved" {
        fields {
          reservationId = required(string)
          roomId        = required(string)
          guestName     = required(string)
          reservedAt    = required(timestamp)
        }
      }

      flow = <<-FLOW
        command -> event:    ReserveRoom -> RoomReserved
      FLOW
    }

    slice "View Available Rooms" {
      view "AvailableRoomsView" {
        subscribes = [RoomReserved]

        fields {
          roomId     = required(string)
          roomNumber = required(string)
          status     = required(string)
        }
      }
    }

    slice "Send Confirmation Email" {
      view "PendingConfirmationsView" {
        subscribes = [RoomReserved]

        fields {
          reservationId = required(string)
          guestName     = required(string)
          reservedAt    = required(timestamp)
        }
      }

      automation "ConfirmationEmailReactor" {
        on      = RoomReserved
        reads   = PendingConfirmationsView
        command = SendConfirmationEmail

        target {
          context = Notifications
        }
      }
    }
  }
}

context "Notifications" {
  aggregate "Notification" {
    slice "Send Confirmation" {
      command "SendConfirmationEmail" {
        fields {
          reservationId = required(string)
          guestName     = required(string)
          email         = required(string)
        }
      }
    }
  }
}
```

### Validate

```bash
emod validate reservation.emod
```

### Lint for anti-patterns

```bash
emod lint reservation.emod
```

### Format

```bash
emod fmt reservation.emod          # format in place
emod fmt --check reservation.emod  # check only (CI)
```

### Dynamic Consistency Boundary (DCB) models

For cross-cutting consistency boundaries, use `mode = dcb` to define slices directly under a context with tagged events and tag-scoped decision queries:

```
context "Fulfillment" {
  mode = dcb

  slice "Place Order" {
    command "PlaceOrder" { ... }

    event "OrderPlaced" {
      tags {
        entity = customerId
      }

      fields {
        orderId    = string
        customerId = string
      }
    }

    flow = <<-FLOW
      command -> event: PlaceOrder -> OrderPlaced
    FLOW
  }

  slice "Authorize Payment" {
    command "AuthorizePayment" {
      decides_on {
        events = [OrderPlaced]
        where  = tag(entity, customerId)
      }

      fields { ... }
    }

    event "PaymentAuthorized" { ... }
  }
}
```

See [examples/dcb_model.emod](/examples/dcb_model.emod) for a complete DCB example.

### Generate diagrams

```bash
emod diagram reservation.emod --format drawio   # draw.io XML (default)
emod diagram reservation.emod --format mermaid  # Mermaid markdown
emod diagram reservation.emod --format svg      # standalone SVG
emod diagram reservation.emod --format ascii    # terminal preview
emod diagram reservation.emod --specs           # …with each slice's specs as a
                                                # Given-When-Then card (drawio and svg only)
```

### Draw the event flow

The four formats above draw the whole model, lane by lane. `--format event-flow`
draws one question instead: which events the system appends, and what reacts to
them.

```bash
emod diagram reservation.emod --format event-flow          # writes reservation.event-flow.svg
emod diagram reservation.emod --format event-flow-mermaid  # the same flow as Mermaid, on stdout
```

The commands are collapsed away — an automation is joined straight to the events
its command emits — and the views are left out. What is left is an orange pill
for each event, a bare gear for each automation and translation reactor, a
rounded box for whatever issues a command from outside them (a trigger with the
actor who works it, an external system, or a command nothing issues), and a
dashed red box for an invariant that refuses a command. An event nothing reads
is drawn dashed, so a reader is not left to assume something reacts to it. An
event only a view reads carries no mark and no arrow onward, because the picture
draws no views.

The SVG carries its own colours and a dark-mode counterpart, so it reads on a
page of either theme, and it is written beside the lane diagram rather than over
it: `--format svg` writes `reservation.svg`, this writes
`reservation.event-flow.svg`.

`--format event-flow-mermaid` states the same graph as Mermaid, for a reader who
wants to lay it out again or to keep it in a document beside the source. It asks
for the ELK layout, which is what keeps a flow of this shape from tangling, and
its gear is a plain glyph rather than an icon, so GitHub renders it too. Where
the SVG writes a caption under a shape, Mermaid writes it as a second line
inside the shape, Mermaid placing every node itself.

### Export

```bash
emod export reservation.emod --format json  # JSON
emod export reservation.emod --format cue   # CUE schema
```

### List slices

```bash
emod slices list reservation.emod
```

### Arrange slices

Reorders each container's slices so the model reads forward — as far as
possible, every reference a slice makes points at a slice declared before it.
Only view slices move: a view projects events rather than stating a step of the
process, so it has no place of its own in the story, while the process slices
keep the order their author gave them.

```bash
emod slices arrange reservation.emod           # rewrite the file
emod slices arrange --check reservation.emod   # exit 1 if a view is out of place
```

Both forms report the references still pointing backward. Some cannot be removed
by any ordering — two slices producing one event means the second points back at
the declaration whichever way round they go — so the report says what the order
genuinely costs rather than claiming the model is now free of them.

### Render a glossary

```bash
emod glossary reservation.emod                # markdown (default)
emod glossary reservation.emod --format json  # JSON
```

## Editor Setup

The `emod` binary must be on your `PATH` for editor integrations to work.

A model is written in HCL, so any editor that colours HCL colours an `.emod`
file: point it at the HCL grammar and set the file type. Everything else comes
from the language server:

- **Diagnostics** — parser and validator errors as squiggly underlines
- **Completion** — the entries a block accepts, and the names a value slot admits
- **Go-to-definition** — jump from references to their definition
- **Find references** — find all usages of a command, event, or view
- **Hover** — contextual information on names and keywords
- **Semantic highlighting** — a context, an aggregate, a command, an event and a view, each painted as what it is
- **Format on save** — auto-format via `emod fmt`

### VS Code — symlink (recommended)

```bash
task setup:vscode
```

### VS Code — .vsix package

```bash
npx @vscode/vsce package --cwd editors/vscode
code --install-extension emod-*.vsix
```

### JetBrains (GoLand / IntelliJ)

Register `.emod` with the HCL file type under **Settings → Editor → File Types**,
then add the language server via the
[LSP4IJ](https://plugins.jetbrains.com/plugin/23257-lsp4ij) plugin:

1. Install the LSP4IJ plugin from the JetBrains Marketplace
2. Open **Settings → Languages & Frameworks → Language Servers**
3. Add a new server with command `emod` and argument `lsp`, file type `emod`

### Neovim

Colour comes from the HCL parser, and the rest from the language server:

```lua
vim.filetype.add({ extension = { emod = "hcl" } })

vim.api.nvim_create_autocmd('FileType', {
  pattern = 'hcl',
  callback = function()
    if vim.fn.expand('%:e') ~= 'emod' then
      return
    end
    vim.lsp.start({
      name = 'emod',
      cmd = { 'emod', 'lsp' },
    })
  end,
})
```

`:TSInstall hcl` installs the parser nvim-treesitter uses for the colour.

### Zed

```json
{
  "file_types": { "HCL": ["emod"] },
  "lsp": { "emod": { "binary": { "path": "emod", "arguments": ["lsp"] } } }
}
```

### Helix

```toml
[[language]]
name = "hcl"
file-types = ["hcl", "tf", "emod"]
language-servers = ["emod"]

[language-server.emod]
command = "emod"
args = ["lsp"]
```

## Development

```bash
go test -tags unit ./...  # unit tests
go test -tags unit -count=1 ./...  # bypass cache
```

### Desktop app

The viewer also runs as a native window, with no browser and no local server:

```bash
task build:desktop   # assembles the frontend, generates bindings, builds
./bin/emod-desktop
```

It renders the same diagrams the browser viewer does, from the same frontend
files, but reaches the Go pipeline directly instead of through WebAssembly.
Building it needs a C toolchain — it is the only binary here that links CGO —
and on Linux the GTK4 and WebKitGTK development packages.

**The source panel** checks what is typed into it as it goes. Once typing
pauses, the panel's text runs through the same lex, parse, validate and lint
chain `emod validate` runs, and the badge and the diagnostics panel list what
that reports — the same messages under the same rule names, at the same
severities and lines. Source that parses redraws the diagram where it is,
keeping pan, zoom, dragged nodes and whatever the visibility panel hides. Source
that does not parse leaves the last diagram on screen, dimmed and marked out of
date, until it parses again. **Render** stays below the panel and redraws at
once. Because the diagram is redrawn from the panel, typing replaces any change
made on the diagram itself since it was last drawn. The browser viewer and
`emod diagram --serve` share the frontend and behave the same way.

**File ▸ Open** (⌘O on macOS, Ctrl+O on Linux and Windows) opens a model through the
operating system's own file picker, filtered to `.emod` and `.json`. The chosen
file renders straight away — no paste, no Render click — and the window takes
the file's name while the bar along the bottom shows its full path. A file that
will not parse cleanly still opens: the diagram shows what it can and the
diagnostics panel lists the same errors `emod validate` reports. A file that
cannot be read at all says why and leaves the model already on screen alone.

**Dragging a file onto the window** opens it exactly as choosing it through the
picker does, from where it lives on disk: the window takes its name, the bar
along the bottom shows its real path, and Save writes back there with no dialog.
Release it anywhere in the window — an overlay says where it will land while a
file is over it. A file whose name ends in neither extension reports `Only .emod
and .json files are supported` and leaves the model on screen. Several files at
once open the first `.emod` or `.json` among them, and the window names that one.
A drop over unsaved changes goes through the same question every other way of
replacing the model does. The browser viewer takes drops too, but reads them
through the browser's own file reader, which yields contents and never a
location — which is why it can only ever offer a model back as a download.

**File ▸ Open Recent** lists the models opened most recently, newest first and
at most ten, and choosing one opens it exactly as the picker does: the window
takes its name, the bar shows its path, and Save writes back to it with no
dialog. A model opened again moves to the top rather than appearing twice, a
model saved to a new location joins the list, and two files that share a name
are told apart by their directory. The list is kept between runs as
`recent-files.json` in the platform's own per-user configuration directory —
`~/Library/Application Support/emod` on macOS, `~/.config/emod` on Linux,
`%AppData%\emod` on Windows. An entry whose file is no longer there says so
when chosen and leaves the list; one the filesystem refuses for any other
reason says why and stays. **Clear Menu**, at the bottom of the list, empties
it.

**File ▸ Save** (⌘S, Ctrl+S) writes the model back to the file it was opened
from — that exact path, no dialog — and confirms in the bar along the bottom.
With no file open, and for **File ▸ Save As** (⇧⌘S, Ctrl+Shift+S), the operating
system's save dialog asks where, and the file chosen becomes the target every
later Save writes to. What Save writes is the source in the panel, so a file
opened and saved with nothing edited is unchanged byte for byte, line endings
included. A write the filesystem refuses — a read-only file, a directory that
may not be written into — says why and leaves what is on disk untouched.

Save and Export answer different questions. Export writes the model
re-serialised from the diagram, which is canonically formatted and carries no
comments; Save writes the text in the source panel. Reconciling the two, once
the diagram and the panel can both be edited, is still to come.

**Unsaved changes** are what the source panel holds that the file behind it does
not — Save writes the panel, so editing the panel is what raises the marker and
a successful save is what clears it. Editing the panel back to exactly the text
the file arrived with clears it too, line endings included. Source pasted into
an empty window counts as unsaved until it has been written somewhere. macOS
shows the marker as a dot in the window's close button; elsewhere the window's
name carries a `*`.

Closing the window, quitting, and opening another model each ask before they
discard anything: **Save** writes the open file and then goes ahead, **Discard**
goes ahead and leaves the file on disk untouched, and **Cancel** does nothing at
all. A save the filesystem refuses, and one whose location dialog is cancelled,
both leave the model where it was. With nothing unsaved, none of the three asks.
This confirmation is a native three-button dialog, which macOS and Linux show as
written; Windows is not there yet.

The framework version is pinned in `go.mod`, which carries both the library
requirement and a `tool` directive for the `wails3` CLI, so the two cannot
drift and neither is resolved from `PATH`. Run the CLI as `go tool wails3`.

**Packaging it as a macOS app** wraps that binary in a double-clickable bundle:

```bash
task package:desktop   # builds, then assembles ./bin/emod.app
open ./bin/emod.app
```

`bin/emod.app` is a normal macOS application. It carries the emod name and icon
in Finder, the Dock and the app switcher, and it runs on a Mac with no Go
toolchain, no Node and no emod CLI: the frontend is compiled into the binary and
the only libraries it links are the ones macOS itself ships. Copy it to
`/Applications`, or anywhere else, and it keeps working — nothing in it points
back at the directory it was built in.

**A copy that arrives through a browser needs one step before it will open.**
The app is not signed with an Apple Developer certificate and is not notarized,
which is a deliberate trade — see §7.3 of
[docs/proposals/emod-desktop-proposal.md](docs/proposals/emod-desktop-proposal.md).
Gatekeeper only
asks about files carrying `com.apple.quarantine`, which is an attribute the
browser attaches to what it downloads and which a bundle you built yourself
never has. So a locally built `emod.app` opens with no ceremony, and a
downloaded one is refused on its first launch. Either of these clears it:

- Open **System Settings ▸ Privacy & Security**, find the message naming emod,
  and press **Open Anyway**.
- Or strip the attribute directly:
  ```bash
  xattr -dr com.apple.quarantine /Applications/emod.app
  ```

You do this once for a copy you have downloaded, not on every launch — after it,
that copy opens like any other application. Downloading a later build gives you
a new copy, which arrives quarantined in its turn.

**Double-clicking a `.emod` file opens it in emod.** The bundle declares the
`.emod` type as its own, so Finder draws models with the app's document icon and
opens them here. If emod is already running the model opens in the window that is
already there, rather than starting a second copy — and if that window holds
unsaved edits, it asks the same Save, Discard or Cancel question every other way
of replacing the model asks. Launching the app on its own still opens empty.

macOS learns this from the bundle, and only once it has seen the bundle: move
`emod.app` to your Applications folder, or open it once from wherever you built
it. A bundle that has sat unopened in `bin/` is not registered, and
double-clicking a model will not find it. To check that it has been:

```bash
mdls -name kMDItemContentType yourmodel.emod
```

Once emod is registered that reports `au.pnguyen.emod.model`. Before it, models
are an anonymous `dyn.…` type belonging to no application.

**emod offers itself for `.json` files without taking them over.** It appears in
Finder's **Open With** menu for a `.json` file, below whatever already opens
them, and the default is left exactly as it was — a machine where `.json` opens
in an editor keeps opening it there. To send one file to emod instead, use
**Get Info** on it and choose emod under **Open With**; the **Change All** button
beside it does the same for every `.json` on the machine, which is a bigger
change than it looks.

The association is macOS-only today. The Linux and Windows builds are still to
come, and neither registers a file type yet.

What it does not do yet: a diagram edit does not reach the source panel and so
is neither what Save writes nor what counts as an unsaved change; no installer;
and no prebuilt download, so it has to be built from source.

How the repository fits together — packages, the language pipeline, renderers,
the viewer's three distributions and the editor grammars — is described in
[docs/architecture.md](docs/architecture.md), with the WebAssembly subsystem
covered in depth by [docs/wasm-architecture.md](docs/wasm-architecture.md).
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

### Write a model

Create `reservation.emod`:

```emod
model "Hotel Reservation"

actor Guest

context Reservations {
  aggregate Reservation {
    slice "Reserve a Room" {
      trigger UI "Reservation Form" {
        actor Guest
        reads AvailableRoomsView
      }

      command ReserveRoom {
        fields {
          roomId    string required
          guestName string required
          checkIn   date   required
          checkOut  date   required
        }
      }

      event RoomReserved {
        fields {
          reservationId string required
          roomId        string required
          guestName     string required
          reservedAt    timestamp required
        }
      }

      flow {
        command -> event: ReserveRoom -> RoomReserved
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

### Generate diagrams

```bash
emod diagram reservation.emod -f drawio   # draw.io XML (default)
emod diagram reservation.emod -f mermaid  # Mermaid markdown
emod diagram reservation.emod -f svg      # standalone SVG
emod diagram reservation.emod -f ascii    # terminal preview
```

### Export

```bash
emod export reservation.emod -f json  # JSON
emod export reservation.emod -f cue   # CUE schema
```

### List slices

```bash
emod slices reservation.emod
```

## Editor Setup

The `emod` binary must be on your `PATH` for editor integrations to work.

Once set up, you'll get:
- **Syntax highlighting** — keywords, strings, comments, identifiers
- **Diagnostics** — parser and validator errors as squiggly underlines
- **Completion** — context-aware keyword suggestions
- **Go-to-definition** — jump from references to their definition
- **Find references** — find all usages of a command, event, or view
- **Hover** — contextual information on names and keywords
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

**Syntax highlighting** via TextMate bundle:

1. Open **Settings → Editor → TextMate Bundles**
2. Click **+** and browse to `editors/vscode/`

**LSP features** via the [LSP4IJ](https://plugins.jetbrains.com/plugin/23257-lsp4ij) plugin:

1. Install the LSP4IJ plugin from the JetBrains Marketplace
2. Open **Settings → Languages & Frameworks → Language Servers**
3. Add a new server with command `emod` and argument `lsp`, file type `emod`

### Neovim

Using [nvim-lspconfig](https://github.com/neovim/nvim-lspconfig):

```lua
vim.api.nvim_create_autocmd('FileType', {
  pattern = 'emod',
  callback = function()
    vim.lsp.start({
      name = 'emod',
      cmd = { 'emod', 'lsp' },
    })
  end,
})
```

### Zed

Add to `~/.zed/settings.json`:

```json
{
  "languages": {
    "EMOD": {
      "language_servers": ["emod"]
    }
  },
  "lsp": {
    "emod": {
      "command": "emod",
      "args": ["lsp"]
    }
  }
}
```

### Helix

Add to `languages.toml`:

```toml
[[language]]
name = "emod"
scope = "source.emod"
file-types = ["emod"]
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
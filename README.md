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

actor "Guest"

context "Reservations" {
  aggregate "Reservation" {
    slice "Reserve a Room" {
      trigger "Reservation Form" {
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

    slice "View Available Rooms" {
      view AvailableRoomsView {
        fields {
          roomId     string required
          roomNumber string required
          status     string required
        }
        subscribes [RoomReserved]
      }
    }

    slice "Send Confirmation Email" {
      view PendingConfirmationsView {
        fields {
          reservationId string    required
          guestName     string    required
          reservedAt    timestamp required
        }
        subscribes [RoomReserved]
      }

      automation ConfirmationEmailReactor {
        on RoomReserved
        reads PendingConfirmationsView
        command SendConfirmationEmail
        target context Notifications
      }
    }
  }
}

context "Notifications" {
  aggregate "Notification" {
    slice "Send Confirmation" {
      command SendConfirmationEmail {
        fields {
          reservationId string required
          guestName     string required
          email         string required
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

For cross-cutting consistency boundaries, use `mode dcb` to define slices directly under a context with tagged events and tag-scoped decision queries:

```
context "Fulfillment" mode dcb {
  slice "Place Order" {
    command PlaceOrder { ... }
    event OrderPlaced {
      tags { entity: customerId }
      fields { orderId string required; customerId string required; ... }
    }
    flow { command -> event: PlaceOrder -> OrderPlaced }
  }

  slice "Authorize Payment" {
    command AuthorizePayment {
      decides_on {
        events [OrderPlaced]
        where tag(entity = customerId)
      }
      fields { ... }
    }
    event PaymentAuthorized { ... }
    flow { ... }
  }
}
```

See [examples/dcb_model.emod](/examples/dcb_model.emod) for a complete DCB example.

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
emod slices list reservation.emod
```

### Render a glossary

```bash
emod glossary reservation.emod          # markdown (default)
emod glossary reservation.emod -f json  # JSON
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

### Tree-sitter grammar (Neovim, Zed, Helix)

The tree-sitter grammar at `editors/tree-sitter-emod/` provides syntax highlighting for editors that use tree-sitter (Neovim, Zed, Helix).

Build the parser:

```bash
cd editors/tree-sitter-emod
tree-sitter generate
```

### Neovim

**Prerequisite:** Build the parser first (see [Tree-sitter grammar](#tree-sitter-grammar-neovim-zed-helix) above).

**Syntax highlighting** via [nvim-treesitter](https://github.com/nvim-treesitter/nvim-treesitter):

```lua
-- Add the emod parser (adjust the url to your local clone path)
local parser_config = require("nvim-treesitter.parsers").get_parser_configs()
parser_config.emod = {
  install_info = {
    url = "~/path/to/emod/editors/tree-sitter-emod",
    files = { "src/parser.c" },
  },
  filetype = "emod",
}

-- Then run: :TSInstall emod
```

**Structural selection** via [nvim-treesitter-textobjects](https://github.com/nvim-treesitter/nvim-treesitter-textobjects):

```lua
require("nvim-treesitter.configs").setup({
  textobjects = {
    select = {
      enable = true,
      lookahead = true,
      keymaps = {
        -- Block-level text objects (e.g., "ab" / "ib" for around/inner block)
        ["ab"] = { query = "@block.outer", desc = "select around block" },
        ["ib"] = { query = "@block.inner", desc = "select inner block" },
      },
    },
  },
})
```

This enables selecting entire structural blocks (slice, fields, command, event,
etc.) with `vab` (around block) or `vib` (inner block).

**LSP integration** via [nvim-lspconfig](https://github.com/neovim/nvim-lspconfig):

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

**Verification:** Open an `.emod` file and run `:Inspect` to confirm the tree-sitter parser is active (you should see the language set to `emod` and AST node types highlighted). Test highlighting by checking that keywords, strings, and comments are colored. Test text objects by placing your cursor inside a block and pressing `vib` — the inner content should be selected.

### Zed

**Syntax highlighting** — add the grammar path to `~/.config/zed/languages/emod.scm` (symlink or copy the highlight queries):

```bash
ln -sf "$(pwd)/editors/tree-sitter-emod/queries/highlights.scm" ~/.config/zed/languages/emod/highlights.scm
```

**LSP** — add to `~/.zed/settings.json`:

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

**Syntax highlighting** — add to `languages.toml`:

```toml
[[grammar]]
name = "emod"
source = { path = "/path/to/editors/tree-sitter-emod" }

[[language]]
name = "emod"
scope = "source.emod"
file-types = ["emod"]
language-servers = ["emod"]
grammar = "emod"

[language-server.emod]
command = "emod"
args = ["lsp"]
```

Then run `hx --grammar fetch && hx --grammar build` to compile the parser.

**Highlight queries** are loaded from `editors/tree-sitter-emod/queries/highlights.scm`. Helix expects them at `runtime/queries/emod/highlights.scm` relative to the grammar source — the path above resolves automatically.

## Development

```bash
go test -tags unit ./...  # unit tests
go test -tags unit -count=1 ./...  # bypass cache
```

How the repository fits together — packages, the language pipeline, renderers,
the browser viewer and the editor grammars — is described in
[docs/architecture.md](docs/architecture.md), with the WebAssembly subsystem
covered in depth by [docs/wasm-architecture.md](docs/wasm-architecture.md).
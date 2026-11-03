---
title: Editor setup
weight: 4
---

The `emod` binary must be on your `PATH` for editor integrations to work.

A model is written in HCL, so any editor that colours HCL colours an `.emod` file: point it at
the HCL grammar and set the file type. Everything else comes from the language server:

- **Diagnostics** — parser and validator errors as squiggly underlines
- **Completion** — the entries a block accepts, and the names a value slot admits
- **Go-to-definition** — jump from references to their definition
- **Find references** — find all usages of a command, event, or view
- **Hover** — contextual information on names and keywords
- **Semantic highlighting** — each element painted as what it is
- **Format on save** — auto-format via `emod fmt`

## VS Code — symlink (recommended)

```sh
task setup:vscode
```

## VS Code — .vsix package

```sh
npx @vscode/vsce package --cwd editors/vscode
code --install-extension emod-*.vsix
```

## JetBrains (GoLand / IntelliJ)

Register `.emod` with the HCL file type under **Settings → Editor → File Types**, then add the
language server via the [LSP4IJ](https://plugins.jetbrains.com/plugin/23257-lsp4ij) plugin:

1. Install the LSP4IJ plugin from the JetBrains Marketplace.
2. Open **Settings → Languages & Frameworks → Language Servers**.
3. Add a new server with command `emod` and argument `lsp`, file type `emod`.

## Neovim

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

## Zed

```json
{
  "file_types": { "HCL": ["emod"] },
  "lsp": { "emod": { "binary": { "path": "emod", "arguments": ["lsp"] } } }
}
```

## Helix

```toml
[[language]]
name = "hcl"
file-types = ["hcl", "tf", "emod"]
language-servers = ["emod"]

[language-server.emod]
command = "emod"
args = ["lsp"]
```

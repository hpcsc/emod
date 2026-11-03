---
title: Web viewer
weight: 5
---

The diagram viewer runs in three places: as a browser page, as `emod diagram --serve`, and as a
native desktop window. All three share one frontend, so they render and edit the same way.

## The browser viewer

Open the [hosted viewer](/viewer/) to load a diagram with no local install. Drop a `.emod` or
`.json` file onto the page, or paste source into the panel, and press **Render**.

The source panel checks what is typed into it as it goes. Once typing pauses, the text runs
through the same lex, parse, validate and lint chain `emod validate` runs, and the badge and the
diagnostics panel list what that reports. Source that parses redraws the diagram in place,
keeping pan, zoom and dragged nodes.

Changes made on the diagram — adding, deleting or renaming an element, editing its fields,
reordering slices, or drawing, repointing or deleting an arrow — are written back into the source
panel. In the browser, the model can be offered back as a download; it has no file on disk to
write to.

## Serve a model locally

`emod diagram --serve` opens the same viewer, pointed at a model on disk, so its diagram edits
can be saved back to the file.

## The desktop app

The viewer also runs as a native window, with no browser and no local server:

```sh
task build:desktop   # assembles the frontend, generates bindings, builds
./bin/emod-desktop
```

The desktop app reaches the Go pipeline directly instead of through WebAssembly, and can open,
watch and save `.emod` files through the operating system. Packaging it as a macOS app is
described in the [repository README](https://github.com/hpcsc/emod#desktop-app).

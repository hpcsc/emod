---
title: Diagrams
weight: 3
---

Every diagram emod draws comes from the same `.emod` source. Pick the format that suits the
moment.

## Draw the whole model

```sh
emod diagram reservation.emod --format svg      # standalone SVG (default)
emod diagram reservation.emod --format mermaid  # Mermaid markdown
emod diagram reservation.emod --format ascii    # terminal preview
emod diagram reservation.emod --specs           # …with each slice's specs as a
                                                # Given-When-Then card (svg only)
```

The three formats draw the whole model, lane by lane.

## Draw the event flow

The formats above draw every element. `--format event-flow` answers one question instead: which
events the system appends, and what reacts to them.

```sh
emod diagram reservation.emod --format event-flow                   # writes reservation.event-flow.svg
emod diagram reservation.emod --format event-flow-mermaid           # the same flow as Mermaid, on stdout
emod diagram reservation.emod --format event-flow-mermaid-no-icons  # the Mermaid for GitHub, with no icon
```

Commands are collapsed away — an automation is joined straight to the events its command emits —
and views are left out. What remains is an orange pill for each event, a bare gear for each
automation, a rounded box for whatever issues a command from outside, and a dashed red box for an
invariant that refuses a command.

The SVG carries its own colours and a dark-mode counterpart, so it reads on a page of either
theme, and it is written beside the lane diagram rather than over it: `--format svg` writes
`reservation.svg`, `--format event-flow` writes `reservation.event-flow.svg`.

## Export

```sh
emod export reservation.emod --format json  # JSON
emod export reservation.emod --format cue   # CUE schema
```

## List and arrange slices

```sh
emod slices list reservation.emod            # list slices
emod slices arrange reservation.emod         # reorder slices so the model reads forward
emod slices arrange --check reservation.emod # exit 1 if a view is out of place
```

## Render a glossary

```sh
emod glossary reservation.emod                # markdown (default)
emod glossary reservation.emod --format json  # JSON
```

## View in the browser

`emod diagram --serve` opens the model in the interactive browser viewer — or open the
[hosted viewer](/viewer/) with no local install.

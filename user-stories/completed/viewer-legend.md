# Diagram Legend in the Viewer

## Overview

The viewer encodes meaning in colour and line style: a command is blue, an event orange, a view green, an automation purple; a subscription is a dashed green arrow, an automation trigger a dashed red one, a translation command a solid orange one. None of it is labelled. A reader who did not write the model — or who wrote it a month ago — has to infer the vocabulary from context or read the source, which is the same gap the diagram exists to close.

This set adds a legend panel to the viewer, listing every element type and every connection type with the exact swatch the canvas draws, derived from the palette the renderer itself reads so the two cannot drift apart.

## Goals

- A reader can name every colour on the diagram without opening the source or the docs
- A reader can tell the seven connection types apart, including the three that differ only by colour and dash pattern
- The legend is built from the renderer's own palette, so a new element or connection type appears in it without a second edit
- The legend stays out of the way: off by default, toggled like the minimap and visibility panels

## User Stories

### US-001: Read what each element colour means
**Description:** As a model author sharing a diagram, I want a legend naming the colour of each element type so that a reader who does not know the notation can tell a command from an event from a view.

**Acceptance Criteria:**
- [ ] A `Legend` button in the toolbar toggles a legend panel, matching the existing minimap and visibility toggles in placement and behaviour
- [ ] The panel is hidden on first load, and closes either from its own header close button or from the toolbar button that opened it
- [ ] The panel lists every element type in `nodePalette` — trigger, command, event, view, automation, translation — each as a swatch drawn with that type's fill and stroke, beside its name
- [ ] Swatch colours are read from `nodePalette` at render time rather than restated in markup or CSS, so a palette edit or a new element type reaches the legend with no second change
- [ ] Element names read as the model vocabulary a user would recognise, not the internal node-type strings where the two differ

**Context:** The viewer palette lives in `config.js:30-37`; the Go renderers keep their own copy of the same hex values in `internal/diagram/layout.go:72-82`. The legend reads the former — introducing a third copy is what the derived-swatch criterion exists to prevent.

---

### US-002: Read what each connection style means
**Description:** As a model author sharing a diagram, I want the legend to explain the arrows so that a reader can tell a subscription from an automation trigger, which differ only by colour.

**Acceptance Criteria:**
- [ ] The legend panel carries a second section listing every connection type in `edgeConfig` — flow, subscription, automation trigger, automation command, trigger command, reads, translation command
- [ ] Each row draws a short line sample using that type's stroke colour, dash pattern, and arrowhead marker, so a dashed green subscription and a dashed red automation trigger are distinguishable in the legend exactly as they are on the canvas
- [ ] Line samples are read from `edgeConfig` at render time, on the same terms as the element swatches
- [ ] Connection names describe the relationship in model terms rather than repeating the edge-type key
- [ ] The two sections are separately headed, so elements and connections do not read as one list

**Depends on:** US-001

## Non-Goals

- A legend on the static SVG and draw.io exports — the same gap exists there and is worth its own story, but those renderers keep a separate palette and their output is fixed-size, so the layout question is different
- Filtering the diagram by clicking a legend entry
- Showing only the element types present in the current model rather than the full vocabulary
- Explaining the container chrome — context swimlane, aggregate row, slice box — which carry labels on the canvas already
- Unifying the viewer and Go palettes into one shared definition

## Open Questions

- Should the legend list the full vocabulary always, or only the types the open model actually uses? Listed as a non-goal on the assumption that a legend teaches the notation, but a large model with three element types makes the case for the other reading.
- Should the panel remember its open state across reloads, as a reader reviewing several models might want it pinned?
- Do the container colours — the dark context header, the grey aggregate row, the dashed slice outline — need a third section, or do their on-canvas labels make them self-explanatory?

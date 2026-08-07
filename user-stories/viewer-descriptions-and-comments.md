# Descriptions and Comments in the Viewer

## Overview

`.emod` files carry two kinds of prose that the viewer discards: a `description` attribute on every construct, and `#` comments attached to whichever construct follows them. Both reach the CLI's other surfaces — descriptions become glossary entries, draw.io tooltips and SVG `<title>` elements; comments survive `emod fmt` — but neither reaches the browser. The viewer is fed diagram JSON, whose node shape carries neither field, so the prose is dropped before the page ever loads. Worse, the viewer's export button reconstructs `.emod` text from that same node shape, so editing a model in the viewer deletes every description and every comment the file had.

This set carries both through the diagram JSON in each direction, then surfaces them where the reader is already looking: descriptions inline on context and aggregate headers, and a hover marker for the constructs whose headers have no room.

## Goals

- A model author reading a diagram can see what a construct means without opening the source file
- A model author can tell at a glance which constructs are documented and which are not
- Comments written in the source are readable in the viewer, attached to the construct they describe
- Editing a model in the viewer and exporting it back no longer deletes descriptions or comments
- A model with no descriptions and no comments renders exactly as it does today

## User Stories

### US-001: Carry descriptions through the diagram JSON
**Description:** As a viewer, I want each diagram node to carry its construct's description so that I can display it without a second parse of the source.

**Acceptance Criteria:**
- [ ] `jsonDiagramNode` gains an optional `description` field, populated for every node type that can carry one: context, aggregate, slice, command, event, trigger, view, automation, and translation
- [ ] `emod export -f diagram-json` on a model with a description on every construct emits each one on its node
- [ ] The importer reads `description` back, so a diagram-JSON round-trip through `ImportDiagram` and `formatter.Format` preserves every description
- [ ] A construct without a description emits no `description` key, and a model with no descriptions produces byte-identical diagram JSON to today

---

### US-002: Preserve comments through a viewer round-trip
**Description:** As a model author, I want the comments I wrote to survive a visual edit so that exporting from the viewer does not silently delete the notes my team left in the file.

**Acceptance Criteria:**
- [ ] `jsonDiagramNode` gains an optional `comments` field carrying each comment's text and source line, populated for every node type whose AST construct holds comments
- [ ] The importer reads `comments` back onto the AST construct, so a round-trip through `ImportDiagram` and `formatter.Format` re-emits them above the construct they belong to
- [ ] A comment on a construct the viewer deletes is dropped with that construct, not reattached elsewhere
- [ ] A model with no comments produces byte-identical diagram JSON and byte-identical formatted output to today

**Context:** Comments attach to constructs rather than to byte offsets (`parser.go:1460-1476`), so they travel with a node rather than needing position repair. Comments belonging to constructs with no diagram node — invariants, specs, `decides_on`, and the model itself — have no node to ride and stay out of scope; see Open Questions.

---

### US-003: Read a construct's description in the detail panel
**Description:** As a model author, I want a selected node's description in the detail panel so that I can read the full text without hovering or opening the source.

**Acceptance Criteria:**
- [ ] Selecting a node with a description shows it in a labelled block between the panel header and the Fields section
- [ ] The text wraps over as many lines as it needs, and the sections below it keep their order
- [ ] Selecting a node without a description shows no description block and no empty placeholder
- [ ] The block renders for every selectable node type, including those with no fields

**Depends on:** US-001

---

### US-004: See context and aggregate descriptions on their headers
**Description:** As a model author, I want a context's and an aggregate's description beside its name so that the boundary's meaning reads off the diagram without any interaction.

**Acceptance Criteria:**
- [ ] A context with a description renders it beside the context name in the swimlane header, visually subordinate to the name
- [ ] An aggregate with a description renders it beside the aggregate name in the aggregate label row
- [ ] An aggregate description too wide for its row is truncated with an ellipsis so it never paints over the neighbouring aggregate's label
- [ ] Neither header grows in height, and a model with no descriptions lays out identically to today

**Context:** The context header has room to spare — it spans the full width of the context's slices. The aggregate row is 22px tall and floored at 100px wide, and the renderer already carries a note about narrow aggregates overlapping their neighbours (`renderer.js:231-233`), which is what the truncation criterion protects.

**Depends on:** US-001

---

### US-005: Spot and read a description without opening the panel
**Description:** As a model author, I want a marker on every construct that has a description, and its text on hover, so that I can see which parts of my model are documented and read any of them in one gesture.

**Acceptance Criteria:**
- [ ] A node with a description renders a marker in its top-right corner, clear of the label and of the automation cadence and translation external-system rows
- [ ] A slice with a description renders the same marker in its header, leaving the slice name centred where it is today
- [ ] Hovering the marker shows the description in the viewer's own tooltip, not a native browser tooltip
- [ ] A construct with no description renders no marker
- [ ] Hovering a node that has a description but no fields shows a tooltip, where today no tooltip appears at all

**Context:** Slice descriptions stay out of the header text because the slice name is centred and the header is sized from the slice's contents — inline text there would shift every slice name off-centre and change the layout constants the positions are computed from.

**Depends on:** US-001, US-003

---

### US-006: Spot and read comments on any construct
**Description:** As a model author, I want a marker on every construct that carries comments, and their text on hover, so that the notes in the source are visible on the diagram instead of only in the editor.

**Acceptance Criteria:**
- [ ] A node with comments renders a comment marker distinguishable from the description marker
- [ ] Context, aggregate and slice headers render the same marker when their construct carries comments
- [ ] Hovering the marker shows the comments in the viewer's own tooltip, in source order, one per line, with the leading `#` stripped
- [ ] A construct carrying both a description and comments shows both markers, and each shows only its own text
- [ ] Comments are read-only in the viewer: no editing affordance appears in the tooltip or the detail panel

**Context:** Comments stay read-only because a comment's meaning is tied to a source position that viewer editing can invalidate — the node it is attached to can be deleted, and slices can be reordered underneath it.

**Depends on:** US-002, US-005

## Non-Goals

- Editing descriptions in the viewer — reading them is the gap; writing them back is a separate story once the round-trip is proven
- Editing or repositioning comments in the viewer
- Rendering description text inside a node's box, which is sized from its label and would have to grow
- Field-level descriptions and field-level comments, neither of which the AST carries
- Surfacing comments attached to invariants, specs, `decides_on` clauses, or the model declaration, none of which have a diagram node
- Preserving blank lines and comment indentation exactly as authored

## Open Questions

- Comments inside a `fields` block currently migrate onto the next construct, and a comment trailing the last construct in a block is dropped, both because `ast.Field` holds no comments and the pending buffer is only drained at the next construct. Should the viewer stories inherit that behaviour, or should the parser hold those comments first?
- Should a construct with comments but no description be flagged by a lint rule as documented-in-the-wrong-place, nudging authors toward `description`?
- Should the two markers collapse into one when a construct has both, with a single tooltip showing description then comments?
- Is a file-level notes panel worth it for the comments that no node can carry, or is the source file the right place to read those?

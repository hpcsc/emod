# emod — Event Modeling DSL & Diagram Generation

## Overview

emod is a CLI tool that provides a domain-specific language (`.emod` files) for modeling event-driven architectures. Authors — both humans and AI agents — write declarative models capturing actors, bounded contexts, aggregates, slices, commands, events, views, and reactors. The tool parses these files, validates them against known event modeling anti-patterns, and generates visual diagrams (draw.io, Mermaid, SVG). The DSL uses a custom human-readable syntax, with CUE as the intermediate validated representation.

## Goals

- Enable humans and AI agents to author event models in a version-controllable text format
- Validate models against established event modeling anti-patterns automatically
- Generate visual diagrams from the textual model in multiple output formats
- Provide both human-readable and machine-readable output for terminal and CI use

## User Stories

### US-001: Parse a minimal .emod file
**Description:** As a model author, I want to write a `.emod` file with a model name, one actor, one context, one aggregate, and one command-pattern slice so that the tool can parse it into a structured representation.

**Acceptance Criteria:**
- [ ] `emod validate minimal.emod` exits with code 0 and prints no errors for a syntactically correct file
- [ ] The parser recognizes `model`, `actor`, `context`, `aggregate`, `slice`, `command`, `event`, and `fields` blocks
- [ ] Comments (lines starting with `#`) are ignored
- [ ] Quoted strings (e.g. model name, slice name, stream pattern) are parsed correctly
- [ ] Unrecognized keywords or unclosed braces produce an error message with file name, line number, and a description of what was expected

**Context:** This is the foundational story. The custom `.emod` syntax uses brace-delimited blocks with keyword identifiers. The parser should produce an internal AST/model that downstream stories consume.

---

### US-002: Parse all four event modeling patterns
**Description:** As a model author, I want to express Command, View, Automation, and Translation patterns in slices so that the full range of event modeling scenarios is captured.

**Acceptance Criteria:**
- [ ] Command pattern slices accept `trigger`, `command`, `event`, and flow declarations (`command -> event: X -> Y`)
- [ ] View pattern slices accept `view` blocks with `fields` and `subscribes` lists
- [ ] Automation pattern slices accept `automation` blocks with `trigger` (event name), `command`, and optional `target context`
- [ ] Translation pattern slices accept `translation` blocks with `external_system`, `reads` (view name), `command`, and inline `event` definitions
- [ ] A file containing all four patterns parses and validates without errors
- [ ] Missing required sub-blocks within a pattern produce a specific error (e.g. "automation block requires a trigger event")

**Depends on:** US-001

---

### US-003: Parse cross-context references and external sources
**Description:** As a model author, I want to reference aggregates and commands across bounded contexts, and mark events as originating from external systems, so that multi-context models and external integrations are expressible.

**Acceptance Criteria:**
- [ ] An automation's `target context` can reference a different context defined in the same model
- [ ] An event with `source external "Provider Name"` is parsed and stored with its external source metadata
- [ ] Referencing a context name that does not exist in the model produces a validation error
- [ ] Referencing a command or event name that does not exist produces a validation error

**Depends on:** US-002

---

### US-004: Format .emod files consistently
**Description:** As a model author, I want to auto-format my `.emod` files so that models have consistent indentation and structure regardless of who wrote them.

**Acceptance Criteria:**
- [ ] `emod fmt reservation.emod` rewrites the file with consistent indentation (2-space indent per nesting level)
- [ ] Keyword alignment within `fields` blocks is normalized (field names, types, and modifiers column-aligned)
- [ ] Blank lines between slices are normalized to exactly one
- [ ] Comments are preserved in their original positions
- [ ] Running `emod fmt` on an already-formatted file produces no changes
- [ ] `emod fmt --check reservation.emod` exits with code 1 if formatting changes are needed, 0 if already formatted (for CI use)

**Depends on:** US-001

---

### US-005: Lint models for naming convention violations
**Description:** As a model author, I want the linter to catch naming problems in my events, commands, and views so that my model follows event modeling conventions.

**Acceptance Criteria:**
- [ ] Events with names ending in `Updated`, `Changed`, or `Modified` produce a "state obsession" warning with the event name and a suggestion to use a specific business fact name
- [ ] Events with names matching `[Entity][Field]Changed` produce a "property sourcing" warning
- [ ] Events with names ending in `Initiated` produce a warning that these are likely commands in disguise
- [ ] Commands that are not in imperative form (e.g. past tense) produce a warning
- [ ] Views whose names do not end in `View` produce a warning
- [ ] Each warning includes the file path, line number, rule name, and a human-readable explanation
- [ ] `emod lint` exits with code 0 when no warnings, code 1 when warnings are found

**Depends on:** US-001

---

### US-006: Lint models for structural anti-patterns
**Description:** As a model author, I want the linter to detect structural problems like the left chair, right chair, and clickbait event patterns so that my model avoids known architectural pitfalls.

**Acceptance Criteria:**
- [ ] A command that produces 3 or more events triggers a "left chair" warning suggesting the command may contain multiple decisions
- [ ] A view that subscribes to 5 or more events triggers a "right chair" / "god view" warning
- [ ] An event whose fields contain only a single ID field triggers a "clickbait event" warning suggesting the payload should include relevant business data
- [ ] Warnings are categorized by severity: `error` for structural violations, `warning` for naming conventions
- [ ] `emod lint --format json` outputs a JSON array of diagnostics with `file`, `line`, `rule`, `severity`, and `message` fields
- [ ] Exit code is 2 for errors, 1 for warnings only, 0 for clean

**Depends on:** US-005

---

### US-007: Validate model completeness
**Description:** As a model author, I want validation to catch incomplete or inconsistent models so that I know my model is structurally sound before generating diagrams.

**Acceptance Criteria:**
- [ ] A command that is not connected to any event produces an error
- [ ] An event that is not produced by any command or external source produces an error
- [ ] A view whose `subscribes` list references a non-existent event produces an error
- [ ] An automation whose trigger references a non-existent event produces an error
- [ ] `emod validate` runs all lint rules plus completeness checks
- [ ] `emod validate --format json` outputs structured diagnostics matching the lint JSON format

**Depends on:** US-003, US-006

---

### US-008: Export model as JSON
**Description:** As an AI agent, I want to export a parsed and validated model as JSON so that I can consume and manipulate event models programmatically.

**Acceptance Criteria:**
- [ ] `emod export reservation.emod -f json` outputs the full model as a JSON document to stdout
- [ ] The JSON structure includes `model.name`, `model.actors`, `model.contexts` with nested aggregates, slices, commands, events, views, automations, and translations
- [ ] Field types and modifiers (required, optional) are preserved in the JSON output
- [ ] Cross-context references and external source metadata are included
- [ ] If the model has validation errors, the command exits with a non-zero code and prints errors to stderr instead of emitting JSON

**Depends on:** US-007

---

### US-009: Convert .emod to CUE for external validation
**Description:** As a model author, I want to export my model as CUE so that I can use CUE's constraint system for additional custom validation beyond the built-in rules.

**Acceptance Criteria:**
- [ ] `emod export reservation.emod -f cue` outputs a CUE file that conforms to the emod CUE schema
- [ ] The exported CUE file can be validated with `cue vet` against the emod schema without errors
- [ ] Round-trip fidelity: exporting to CUE and re-importing produces an equivalent model
- [ ] The emod CUE schema definition is bundled with the tool and can be printed via `emod schema --format cue`

**Depends on:** US-008

---

### US-010: Generate draw.io diagrams
**Description:** As a model author, I want to generate a draw.io XML file from my model so that I can open and share the event model as a visual diagram.

**Acceptance Criteria:**
- [ ] `emod diagram reservation.emod` writes a `.drawio` file to the same directory as the input
- [ ] The diagram uses swimlanes: "UI / Triggers" (top), "Commands / Views" (middle), "Events" (bottom)
- [ ] Events are rendered as orange boxes, commands as blue boxes, views as green boxes, triggers as white boxes
- [ ] Slices are laid out left-to-right in the order they appear in the model
- [ ] Connections between elements follow the flow: trigger -> command -> event, event -> view, event -> reactor -> command
- [ ] Automation reactors are rendered with a gear icon
- [ ] External systems are rendered as gray dashed boxes
- [ ] Output path can be overridden with `-o path/to/output.drawio`

**Depends on:** US-007

---

### US-011: Generate Mermaid diagrams
**Description:** As a model author, I want to generate Mermaid diagram markup from my model so that I can embed event model diagrams in markdown documents and pull requests.

**Acceptance Criteria:**
- [ ] `emod diagram reservation.emod -f mermaid` outputs Mermaid markup to stdout
- [ ] The diagram uses Mermaid's flowchart syntax with styled nodes for events (orange), commands (blue), views (green), and triggers (white)
- [ ] Each slice is visually grouped using Mermaid subgraphs
- [ ] Connections between elements match the event modeling flow patterns
- [ ] The output renders correctly when pasted into a GitHub markdown file
- [ ] Output can be written to a file with `-o path/to/output.md`

**Depends on:** US-007

---

### US-012: Generate SVG diagrams
**Description:** As a model author, I want to generate an SVG image from my model so that I can embed the diagram in documentation or view it without specialized tooling.

**Acceptance Criteria:**
- [ ] `emod diagram reservation.emod -f svg` writes an SVG file
- [ ] The SVG uses the same color scheme and layout as the draw.io output (orange events, blue commands, green views, white triggers)
- [ ] Swimlane labels are present in the SVG
- [ ] Connections between elements are rendered as styled arrows
- [ ] The SVG is self-contained (no external dependencies) and renders in any modern browser
- [ ] Output path can be overridden with `-o path/to/output.svg`

**Depends on:** US-010

---

### US-013: List implementation slices
**Description:** As a model author, I want to list all slices in my model with their pattern type so that I can plan implementation work and see the scope of the model at a glance.

**Acceptance Criteria:**
- [ ] `emod slices reservation.emod` prints a table with columns: slice name, pattern type (command/view/automation/translation), context, and key elements (command name, event name, or view name)
- [ ] Slices are listed in the order they appear in the model, grouped by context
- [ ] `emod slices --format json` outputs the same information as a JSON array
- [ ] If the model has parse errors, the command fails with a descriptive message

**Depends on:** US-001

---

### US-014: Print terminal ASCII preview
**Description:** As a model author, I want a quick text-based preview of my event model in the terminal so that I can review the model's structure without opening a diagram tool.

**Acceptance Criteria:**
- [ ] `emod diagram reservation.emod -f ascii` prints an ASCII representation to stdout
- [ ] The preview shows slices left-to-right (or top-to-bottom if terminal is narrow) with trigger -> command -> event flows
- [ ] Views show which events feed into them
- [ ] Automations show the event -> reactor -> command chain
- [ ] Elements are labeled with their names and visually distinguished (e.g. `[Command]`, `(Event)`, `{View}`)

**Depends on:** US-007

## Non-Goals

- Code generation from models (Go structs, aggregate skeletons) — planned for a future phase
- Desktop/GUI application for interactive model editing — planned for a future phase
- Importing from existing draw.io/Miro diagrams back into `.emod` format
- Runtime event validation or enforcement — emod is a design-time tool
- Multi-file model composition (splitting a model across files) — may be added later

## Open Questions

- Should `emod fmt` sort elements within a context (e.g. alphabetical aggregates) or preserve author ordering?
- What is the right threshold for the "right chair" warning — 5 subscribed events, or should it be configurable?
- Should the draw.io layout auto-calculate element positions based on the number of slices, or use fixed spacing?
- How should the CUE round-trip handle custom CUE constraints that a user adds to the exported file — preserve them on re-import or discard?
- Should `emod validate` have a `--strict` mode that treats warnings as errors?

# Specs, Invariants, and Model Metadata

## Overview

Let `.emod` files describe behaviour and meaning, not just structure. Slices gain Given-When-Then specs (with optional example payloads), aggregates and DCB contexts declare named invariants that rejections reference by name, flows and diagrams show rejection paths, every construct can carry a description feeding a generated glossary, files pin themselves to a DSL version, events bind to their wire-level type, and automations can fire on elapsed time. Every existing `.emod` file stays valid with unchanged meaning.

Stories are listed in recommended implementation order, following the proposal's phasing: metadata foundation, invariants and specs, payload literals, wire types and timers, then tooling.

## Goals

- Model authors can attach Given-When-Then specs to slices, covering all four slice patterns, with optional example payload data
- Business rules live in the model as named invariants; spec rejections and flow rejection edges reference them by name, and the validator checks every reference
- The timeline can show error paths: a command that can be rejected no longer renders identically to one that cannot
- Every construct can carry a description, and `emod glossary` renders the ubiquitous language from them
- Files declare the DSL version they target, so a future breaking grammar change fails with a clear message instead of a parse error
- Exports carry wire-level event types, making them usable as schema-registry and code-generation input
- Relative time-driven behaviour ("release the hold 24 hours after `RoomHeld`") is expressible on automations, completing the absolute schedules the grammar already carries
- Every existing `.emod` file remains valid and unchanged in meaning

## User Stories

### US-001: Pin files to a DSL version
**Description:** As a model author, I want my file to declare which DSL version it targets so that a future breaking grammar change fails with a clear versioning message instead of a confusing parse error deep in the file.

**Acceptance Criteria:**
- [ ] A file may open with `emod 1` on its own line before the `model` declaration, and parses and validates as before
- [ ] A file without the header is treated as version 1, with no diagnostics
- [ ] A file declaring a version higher than the tool supports is rejected with a single "unsupported version" diagnostic naming the declared and the supported version — not a parse error later in the file
- [ ] `emod fmt` inserts `emod 1` when the header is absent, so formatted files are always pinned
- [ ] Every existing header-less `.emod` file keeps its exact meaning

**Context:** Versioning follows the Kubernetes convention: additive grammar changes (new optional keywords) do not bump the version; breaking changes do. This is the cheapest item in the proposal and the most expensive to retrofit later — header adoption is driven entirely by `emod fmt`.

### US-002: Describe constructs where they are declared
**Description:** As a model author, I want to attach a `description` to any construct so that the model itself carries the ubiquitous language instead of a wiki page that drifts.

**Acceptance Criteria:**
- [ ] `context`, `aggregate`, `slice`, `command`, `event`, `view`, `automation`, `translation`, and `trigger` each accept an optional `description` string attribute
- [ ] `actor` and `model` gain an optional block form that holds a description; the existing single-line form remains valid
- [ ] `emod export -f json` and `-f cue` carry descriptions through
- [ ] draw.io diagrams attach descriptions as tooltips; SVG diagrams as `<title>` elements
- [ ] A file with no descriptions parses, validates, exports, and renders exactly as before

### US-003: Use reserved words as field names
**Description:** As a model author, I want fields named `type`, `description`, or any other DSL keyword to remain valid so that new grammar keywords never force me to rename my domain's fields.

**Acceptance Criteria:**
- [ ] A `fields` block accepts each new keyword (`type`, `description`, `spec`, `given`, `when`, `then`, `rejected`, `invariant`, `after`, `emod`) as a field name
- [ ] Words that were already reserved (e.g. `events`, `source`, `on`, `every`, `wireframe`) are also accepted in field-name position
- [ ] Such fields behave like any other field in validation, formatting, exports, and diagrams

**Context:** `fields { type string required }` is a plausible declaration in existing models. Adding `type` and `description` as keywords must not break it — the fix is general, so every current and future keyword gets the same courtesy, including ones added by later grammar work.

### US-004: Generate a glossary from the model
**Description:** As a team member, I want `emod glossary` to render the ubiquitous language from the model so that the whole team shares one vocabulary that cannot drift from the source.

**Acceptance Criteria:**
- [ ] `emod glossary <file>` writes markdown to stdout, with terms grouped by context
- [ ] Each context lists its aggregates, commands, events, views, and actors, each with its description
- [ ] `emod glossary <file> -f json` emits the same content as structured output
- [ ] A construct without a description appears with an empty definition, making gaps visible without a lint rule

**Depends on:** US-002

### US-005: Declare named invariants
**Description:** As a model author, I want aggregates and DCB contexts to declare the business rules they protect so that a rejection is a checkable reference to a declared rule, not free text.

**Acceptance Criteria:**
- [ ] An `aggregate` accepts `invariant <identifier> "<prose statement>"` entries
- [ ] A context in DCB mode accepts the same entries directly on the context
- [ ] Two invariants with the same name in the same scope produce a clear validation error
- [ ] A declared invariant that nothing references is not a validation error (a lint rule flags it later)
- [ ] `emod glossary` lists invariants under their aggregate or context

**Context:** Invariants do nothing on their own — they exist to be referenced by spec rejections (US-006) and flow rejection edges (US-009), and to be listed in the glossary. The rule "a room can hold one active reservation" finally has a place to live.

**Depends on:** US-004 (for the glossary criterion only)

### US-006: Write Given-When-Then specs on command slices
**Description:** As a model author, I want to write named Given-When-Then specs inside a slice so that the behaviour discovered in workshops lives next to the structure it describes instead of drifting in wiki pages or test files.

**Acceptance Criteria:**
- [ ] A slice accepts any number of `spec "<name>" { ... }` blocks
- [ ] `given` takes an ordered list of event names; `given []` and omitting `given` entirely are equivalent (empty history)
- [ ] `when` names the command under test
- [ ] `then [EventA, EventB]` states the events appended on success
- [ ] `then rejected <invariantName>` states the command fails; the name must resolve to an invariant on the enclosing aggregate (or context in DCB mode), and an unresolved name is a validation error with location
- [ ] Every event in `given` and `then`, and the command in `when`, must be defined in the model; unknown names produce an error with location

**Depends on:** US-005

### US-007: Write specs for view, automation, and translation slices
**Description:** As a model author, I want spec shapes matched to each slice pattern so that all four patterns — not just commands — can state their expected behaviour.

**Acceptance Criteria:**
- [ ] In a view slice, a spec omits `when` and concludes with `then view <ViewName>`
- [ ] In an event-driven automation slice, `when` names the automation's `on` event and the spec concludes with `then command <CommandName>`
- [ ] In a schedule-driven automation slice, a spec omits `when` — the `every` expression is the activation, and there is no event to name — and concludes with `then command <CommandName>`, read as: given this history, the next firing issues this command
- [ ] In a translation slice, a spec takes the given/when/then-events form
- [ ] The named view or command outcome must resolve to a construct defined in the model
- [ ] A `then` shape that does not match the slice pattern (e.g. a `view` outcome inside a command slice) is a validation error

**Depends on:** US-006

### US-008: Lint spec coverage and boundary assumptions
**Description:** As a model author, I want lint rules that flag missing specs and specs assuming impossible history so that coverage gaps and boundary violations surface before implementation starts.

**Acceptance Criteria:**
- [ ] `spec/command-without-spec` (info) fires for a command no spec exercises
- [ ] `spec/no-rejection-path` (info) fires for a command with specs but no rejection spec — happy-path-only coverage
- [ ] `spec/invariant-never-exercised` (warning) fires for a declared invariant no `rejected` spec references
- [ ] `spec/given-outside-boundary` (warning) fires in aggregate mode when a `given` event belongs to a different aggregate
- [ ] `spec/given-outside-boundary` (warning) fires in DCB mode when a `given` event's type is not matched by the `when` command's `decides_on`
- [ ] All four rules respect the existing severity configuration and `emod lint --explain <rule>` machinery

**Context:** `spec/given-outside-boundary` is the rule that pays for the feature: it catches specs that assume history the consistency boundary cannot see. US-011 later sharpens it with payload values.

**Depends on:** US-006, US-007

### US-009: Show rejection paths on the timeline
**Description:** As a model author, I want a flow entry from a command to an invariant so that a command that can be rejected no longer renders identically to one that cannot, and the error handling discovered in workshops has somewhere to land.

**Acceptance Criteria:**
- [ ] A `flow` block accepts `command -> rejected: <CommandName> -> <invariantName>` entries alongside the existing entry kind
- [ ] The invariant name resolves against the enclosing aggregate (or context in DCB mode), the same rule as `rejected` in specs; an unresolved name is a validation error
- [ ] Diagrams render the edge dashed, ending in a rejection badge whose tooltip (draw.io) or `<title>` (SVG) carries the invariant's prose
- [ ] `flow/rejection-without-spec` (info) fires when a rejection edge has no spec on the slice exercising that rejection
- [ ] A rejection edge counts as a reference for `spec/invariant-never-exercised`, which otherwise would flag the invariant as unused

**Context:** The edge covers invariant rejections only — the command fails and nothing is appended. A failure the business cares about (a payment declined) is an event and already has a flow entry; the rejection edge exists for the case where the timeline is otherwise silent.

**Depends on:** US-005, US-008 (for the lint interplay)

### US-010: State example payloads in specs
**Description:** As a model author, I want event and command references in a spec to carry example field values so that the scenario's meaning — the repeated room id, the overlapping dates — is stated in the spec itself.

**Acceptance Criteria:**
- [ ] Any event or command reference in `given`, `when`, or `then` accepts a `{ field: value, ... }` block
- [ ] Payloads are partial: fields declared `required` may be omitted — a spec is an example, not an instance
- [ ] Names-only specs remain valid; payloads are additive per element reference
- [ ] A payload field name not declared on the referenced construct's `fields` is a validation error
- [ ] Literal kinds are checked against the declared field type: strings satisfy `string`, `date`, `timestamp`, and `uuid` (the value must parse as the format); numbers satisfy `int` (no fractional part) and `decimal`; `true`/`false` satisfy `bool`
- [ ] Fields of domain types accept any literal unchecked — they are opaque to the model

**Depends on:** US-006

### US-011: Value-aware boundary checking in DCB mode
**Description:** As a model author working in DCB mode, I want `spec/given-outside-boundary` to compare tagged field values so that a spec assuming history outside the command's boundary is caught even when the event type alone would pass.

**Acceptance Criteria:**
- [ ] For a `when` command with a `where tag(key = field)` predicate, a `given` event whose payload states a different value for the tagged field than the `when` payload triggers the `spec/given-outside-boundary` warning
- [ ] Matching values produce no warning
- [ ] When either payload omits the tagged field, only the type-level check from US-008 applies — no value-based warning

**Context:** This is where specs stop being documentation and start being model-checking: with payloads, the rule verifies not just that a `given` event's type is matched by `decides_on`, but that its tagged value falls inside the boundary the command actually reads.

**Depends on:** US-008, US-010

### US-012: Bind model events to wire-level types
**Description:** As a developer consuming exports, I want each event to declare the type it is published under so that JSON and CUE exports become usable input to schema registries and code generation.

**Acceptance Criteria:**
- [ ] An `event` accepts an optional `type "<string>"` attribute; the value is an opaque string
- [ ] `emod validate` errors when two events share the same wire type
- [ ] `emod export -f json` and `-f cue` emit the wire type
- [ ] `wire/type-format` (info) nudges toward reverse-DNS kebab-case (the CloudEvents convention) without enforcing it
- [ ] Events without a wire type validate and export exactly as before

### US-013: Fire automations after elapsed time
**Description:** As a model author, I want an automation to fire a fixed duration after an event so that time-driven behaviour like "release the hold 24 hours after `RoomHeld`" is expressible in the model.

**Acceptance Criteria:**
- [ ] An automation's `on` clause accepts an optional `after "<duration>"` suffix, read as: the duration after each `on` event occurrence, issue the command
- [ ] The duration is a string in Go duration syntax (`"30m"`, `"24h"`, `"72h"`); a value that does not parse as a duration is a validation error with location
- [ ] Without `after`, behaviour is unchanged: the automation reacts immediately
- [ ] `after` on a schedule-driven automation is a validation error — `every` is already absolute, so `every "0 2 * * *" after "24h"` has no reading
- [ ] Diagrams carry the duration on the `event -> automation` edge; the clock badge on the automation box stays reserved for the `every` expression, so a relative delay and a wall-clock schedule read differently

**Context:** `after` is the relative half of automation timing — `every` covers the absolute half and is a precondition for this story. The two never combine. How the timer is implemented — durable scheduling, delivery guarantees, idempotency — is a runtime concern and stays out of the model, the same line the DCB proposal drew for append-condition checking.

### US-014: Format the new constructs consistently
**Description:** As a model author, I want `emod fmt` to canonicalise all the new syntax so that team models stay uniform without style debates.

**Acceptance Criteria:**
- [ ] A file using every new construct formats round-trip without data loss
- [ ] Attributes inside a block follow the canonical order: `description` first, then pattern-specific attributes, then `fields`, then `spec` blocks last
- [ ] `given` / `when` / `then` keywords align within a spec
- [ ] The `:` aligns across `command -> event:` and `command -> rejected:` entries within a `flow` block
- [ ] Payloads stay on one line when they fit; otherwise one field per line with values aligned, the same convention as `fields` blocks

**Depends on:** US-001, US-002, US-006, US-009, US-010

### US-015: Navigate and complete the new constructs in the editor
**Description:** As a model author using an LSP-capable editor, I want hover, completion, and navigation for specs and invariants so that I can author them without switching to documentation.

**Acceptance Criteria:**
- [ ] Hovering any construct shows its kind and description
- [ ] Hovering `rejected <name>` — in a spec or on a flow rejection edge — shows the invariant's prose
- [ ] Completion offers invariant names after `rejected`, event names inside `given [...]`, and field names inside payload braces scoped to the referenced construct's `fields`
- [ ] Go-to-definition works from spec event/command references and from invariant references, including flow rejection edges
- [ ] Find-references on an invariant lists the specs and flow edges that reference it

**Depends on:** US-002, US-006, US-009, US-010

### US-016: Render specs on diagrams
**Description:** As a model author, I want the option to render a slice's specs on the diagram so that a workshop artefact can show behaviour alongside structure when the audience needs it.

**Acceptance Criteria:**
- [ ] A `--specs` flag renders each slice's specs as a Given-When-Then card under the slice
- [ ] The flag is off by default; without it, diagram output is unchanged
- [ ] The card shows given, when, and then — including rejected outcomes with the invariant name
- [ ] The card renders in both SVG and draw.io outputs

**Depends on:** US-006, US-007

### US-017: Highlight the new syntax in editors
**Description:** As a model author, I want syntax highlighting for the new keywords so that specs and invariants read as structure, not prose.

**Acceptance Criteria:**
- [ ] The new keywords (`spec`, `given`, `when`, `then`, `rejected`, `invariant`, `description`, `type`, `after`, `emod`) are highlighted in the VS Code extension and the tree-sitter grammar
- [ ] Numbers and `true`/`false` in payload position are highlighted as literals
- [ ] A keyword used in field-name position is highlighted as a field name, not a keyword

**Depends on:** US-006, US-010

### US-018: Learn the new constructs from examples and the reference
**Description:** As a model author new to these features, I want the examples and DSL reference to cover every new construct so that I can learn them from working models instead of the grammar.

**Acceptance Criteria:**
- [ ] `examples/all_patterns.emod` exercises every new construct and passes `emod validate`
- [ ] A new `examples/specs_hotel.emod` mirrors the proposal's worked example — invariants, specs with payloads, a rejection flow edge, a wire type, and a timer — and passes `emod validate`
- [ ] `docs/dsl-reference.md` documents each new construct with at least one example

**Depends on:** US-001 through US-013

## Non-Goals

- Executing specs against an implementation, or exporting test skeletons (a later, separate feature)
- Adopting ESDM's file layout: YAML, one-artifact-per-file, and `scope` back-references — the nested single-file grammar stays
- Structural DDD kinds (`entity`, `value-object`, `domain-service`, `subdomain`)
- A process-manager construct — Event Modeling decomposes it into a view plus an automation that reads it, which the DSL expresses directly; only the elapsed-time clause is adopted
- Context-mapping relationship labels (ACL, conformist, customer-supplier), deferred until diagram grouping gives them somewhere to appear
- Expected view-state payloads on `then view` outcomes — modeling rows and collections is a much larger literal grammar, deferred until scalar payloads prove themselves
- Variable binding in specs (`let`) — payload linkage is by repetition of the same value
- Field-level descriptions
- A spec include mechanism (`specs "reservation_specs.emod"`) — watch real usage first
- Modeling timer runtime properties (delivery guarantees, idempotency, clock skew)

## Open Questions

- Duplicate invariant names in the same scope: the proposal is silent; US-005 assumes a validation error. Confirm this is the intended behaviour.
- Should a schedule-driven automation's spec assert the contents of the todo-list view it reads, rather than only the `given` events that project into it? Assumed not, since view-state payloads are already a non-goal — but a scheduled automation is the one case where the view's contents are the entire precondition.
- Should the `version/missing-header` lint rule (info) ship with US-001, or wait until a version 2 exists? Assumed deferred, since `emod fmt` already drives adoption.
- Should `emod glossary` also show wire types alongside events? Assumed not, until a consumer asks for it.
- Scope assumption: these stories cover the full proposal, sequenced by its five implementation phases. If only Phases 1–2 are in scope for now, US-010 through US-018 can be parked without affecting the earlier stories.

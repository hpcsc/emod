# User story progress

Which story in each in-progress story file is delivered. Files under
`user-stories/completed/` are finished in full and are not tracked here.

**21 of 171 delivered.**

## [emod-desktop.md](./emod-desktop.md) — 0/16

- [ ] US-001: Render a model in a native desktop window
- [ ] US-002: Open a model with a native file dialog
- [ ] US-003: Save a model back to the file it came from
- [ ] US-004: Know when there are unsaved changes
- [ ] US-005: Open a model by dropping it on the window
- [ ] US-006: Reopen a recently opened model
- [ ] US-007: Install and run a packaged app on macOS
- [ ] US-008: Open a model by double-clicking it in the file manager
- [ ] US-009: Edit source in the app and see diagnostics as you type
- [ ] US-010: Jump between a diagnostic and the source that caused it
- [ ] US-011: Keep the source panel and the diagram in agreement
- [ ] US-012: Read source with syntax highlighting
- [ ] US-013: Notice when the open file changes on disk
- [ ] US-014: Run the desktop app on Linux
- [ ] US-015: Download a desktop build without building it
- [ ] US-016: Run the desktop app on Windows

## [specs-and-metadata.md](./specs-and-metadata.md) — 6/18

Delivered on `main`: the `emod <n>` version header, `description` on every construct, keywords
usable as field names, `emod glossary`, named invariants on aggregates and DCB contexts, and
Given-When-Then specs on command slices.

- [x] US-001: Pin files to a DSL version
- [x] US-002: Describe constructs where they are declared
- [x] US-003: Use reserved words as field names
- [x] US-004: Generate a glossary from the model
- [x] US-005: Declare named invariants
- [x] US-006: Write Given-When-Then specs on command slices
- [ ] US-007: Write specs for view, automation, and translation slices
- [ ] US-008: Lint spec coverage and boundary assumptions
- [ ] US-009: Show rejection paths on the timeline
- [ ] US-010: State example payloads in specs
- [ ] US-011: Value-aware boundary checking in DCB mode
- [ ] US-012: Bind model events to wire-level types
- [ ] US-013: Fire automations after elapsed time
- [ ] US-014: Format the new constructs consistently
- [ ] US-015: Navigate and complete the new constructs in the editor
- [ ] US-016: Render specs on diagrams
- [ ] US-017: Highlight the new syntax in editors
- [ ] US-018: Learn the new constructs from examples and the reference

## [triggers-and-automations.md](./triggers-and-automations.md) — 10/11

Delivered on `main`: `reads` on an automation, `on` and `every` as its two activation forms, the
trigger without a kind slot, the `reads` edge and the human-only top lane on diagrams, one palette
across the renderers, the `automation/missing-todo-list` rule, editor completion and navigation for
automations, and highlighting for the realigned syntax. The grammar `specs-and-metadata.md` US-007
and US-013 build on is in place.

Breakdown in flight at `tasks/us-011-learn-the-realignment-from-examples-and-the-reference.md`.

- [x] US-001: Declare the view an automation reads
- [x] US-002: Name an automation's activation event with `on`
- [x] US-003: Activate an automation on a schedule
- [x] US-004: Drop the trigger kind slot
- [x] US-005: Draw the view a trigger or automation reads
- [x] US-006: Read the top lane as human-only
- [x] US-007: One palette for element types
- [x] US-008: Flag automations with no todo list
- [x] US-009: Complete and navigate automations in the editor
- [x] US-010: Highlight the realigned syntax
- [ ] US-011: Learn the realignment from examples and the reference

## [00-llm-foundation.md](./ai/00-llm-foundation.md) — 5/10

Breakdown in flight at `tasks/00-llm-foundation.md` (tasks 1-5 of 10 checked off).

- [x] US-001: Define the `llm.Model` port
- [x] US-002: Mock model for network-free tests
- [x] US-003: Concrete Bedrock adapter
- [x] US-004: Single AI configuration block
- [x] US-005: Single deterministic correctness oracle
- [ ] US-006: Generate → validate → lint → repair loop
- [ ] US-007: Schema-conformant structured output
- [ ] US-008: Cost and token usage reporting
- [ ] US-009: AI stays opt-in; existing paths and WASM stay provider-free
- [ ] US-010: End-to-end smoke command proving the seam

## [01-nl-to-model-generation.md](./ai/01-nl-to-model-generation.md) — 0/8

- [ ] US-001: Generate a valid .emod file from a prose description
- [ ] US-002: Self-correct generated output until it validates and lints clean
- [ ] US-003: Report attempt progress and token cost per run
- [ ] US-004: Degrade honestly when the loop cannot converge
- [ ] US-005: Accept descriptions from stdin and piped prose
- [ ] US-006: Emit machine-readable results for scripting
- [ ] US-007: Ground generation in idiomatic exemplars and the bundled schema
- [ ] US-008: Generate DCB-style models when prose signals per-decision boundaries

## [02-model-import-reverse-engineering.md](./ai/02-model-import-reverse-engineering.md) — 0/11

- [ ] US-001: Import a structured artifact into a draft model
- [ ] US-002: Choose the input family explicitly
- [ ] US-003: Map detected concepts onto emod constructs
- [ ] US-004: Map external integrations to translations
- [ ] US-005: Default to aggregate style, infer DCB only on strong signal
- [ ] US-006: Force a single bounded context
- [ ] US-007: Always emit a clean draft via validate and repair
- [ ] US-008: Surface inherited naming smells in lint
- [ ] US-009: Inspect the condensed brief before spending a model call
- [ ] US-010: Extract signals from source code
- [ ] US-011: Condense large inputs without losing the inventory

## [03-semantic-model-reviewer.md](./ai/03-semantic-model-reviewer.md) — 0/15

- [ ] US-001: Run a semantic review on an existing model
- [ ] US-002: Classify findings into a stable semantic-smell taxonomy
- [ ] US-003: Detect semantic smells the regex linter cannot
- [ ] US-004: Each finding includes a direction and located evidence
- [ ] US-005: Confidence and capped severity on every finding
- [ ] US-006: Filter findings by confidence threshold
- [ ] US-007: Filter findings by severity
- [ ] US-008: JSON output matching the lint format
- [ ] US-009: Opt-in with graceful degradation when AI is not configured
- [ ] US-010: CI-friendly exit codes
- [ ] US-011: Reproducible reviews for CI via caching
- [ ] US-012: Adversarial self-check to reduce false positives
- [ ] US-013: Report token cost and latency of a review
- [ ] US-014: On-demand AI review in the editor
- [ ] US-015: Suppress findings that overlap deterministic lint

## [04-lint-quickfixes-lsp.md](./ai/04-lint-quickfixes-lsp.md) — 0/15

- [ ] US-001: Lint findings carry their rule identity to the editor
- [ ] US-002: "Explain this rule" quick-fix appears on every lint finding
- [ ] US-003: Lightbulb appears instantly on fixable findings without a model call
- [ ] US-004: Rename a state-change event to a business fact
- [ ] US-005: Rename property-sourcing and command-in-disguise events
- [ ] US-006: Rename a past-tense command to imperative form
- [ ] US-007: Rename a view to end in `View`
- [ ] US-008: Offer multiple ranked rename candidates, cleanest first
- [ ] US-009: Fix a clickbait event by adding fields or inlining its identifier
- [ ] US-010: Add tags to an untagged DCB event
- [ ] US-011: Narrow an overly broad DCB query
- [ ] US-012: Resolve an orphan DCB tag key
- [ ] US-013: Split a god-view into focused views
- [ ] US-014: Specialize a command used across too many flows (left-chair)
- [ ] US-015: Show progress and report cost during a quick-fix

## [05-dcb-modeling-assistant.md](./ai/05-dcb-modeling-assistant.md) — 0/8

- [ ] US-001: Suggest tags for an untagged event
- [ ] US-002: Suggest a narrower decision query for a too-broad command
- [ ] US-003: Preview suggestions as a diff or apply them on request
- [ ] US-004: Filter suggestions by rule or target
- [ ] US-005: Resolve an orphan tag key by routing or removal
- [ ] US-006: Introduce an additional tag dimension for single-tag-everywhere
- [ ] US-007: Convert an aggregate-mode context to DCB
- [ ] US-008: Surface DCB suggestions as editor quick-fixes

## [06-mcp-server.md](./ai/06-mcp-server.md) — 0/17

- [ ] US-MCP-001: Start the MCP server over stdio
- [ ] US-MCP-002: Validate a model via an MCP tool
- [ ] US-MCP-003: Lint a model via an MCP tool
- [ ] US-MCP-004: Explain a lint rule via an MCP tool
- [ ] US-MCP-005: Retrieve the slices of a model via an MCP tool
- [ ] US-MCP-006: Retrieve the CUE schema via an MCP tool
- [ ] US-MCP-007: Self-correct a generated model host-side using the validate and lint tools
- [ ] US-MCP-008: Export a model via an MCP tool
- [ ] US-MCP-009: Render a diagram via an MCP tool
- [ ] US-MCP-010: Format a model via a read-only MCP tool
- [ ] US-MCP-011: Expose the schema, examples, and docs as MCP resources
- [ ] US-MCP-012: Ship authoring and review prompts
- [ ] US-MCP-013: Scope file-path tools to a root directory
- [ ] US-MCP-014: Gate disk writes behind an explicit opt-in
- [ ] US-MCP-015: Register the emod server with an MCP host
- [ ] US-MCP-016: Tool output matches the CLI for the same input
- [ ] US-MCP-017: Self-describe the server for debugging

## [07-talk-to-your-model-qa.md](./ai/07-talk-to-your-model-qa.md) — 0/12

- [ ] US-001: Ask a one-shot question grounded on the model export
- [ ] US-002: Get token cost and usage feedback for an answer
- [ ] US-003: Faithful trace and reachability answers across the graph
- [ ] US-004: Flag invented model element names in an answer
- [ ] US-005: Answer producer/consumer questions about an element
- [ ] US-006: Answer cross-context dependency questions
- [ ] US-007: Continue asking in an interactive REPL session
- [ ] US-008: Warn when the file changes during a REPL session
- [ ] US-009: Machine-readable answer output for tooling
- [ ] US-010: Impact and rename analysis for a model element
- [ ] US-011: Decline gracefully when the answer is not in the model
- [ ] US-012: Tune answer effort for simple versus multi-hop questions

## [08-docs-generation.md](./ai/08-docs-generation.md) — 0/9

- [ ] US-001: Refuse to document an invalid model
- [ ] US-002: Generate a faithful per-context reference catalogue
- [ ] US-003: Compute and narrate cross-context edges
- [ ] US-004: Generate narrative prose per slice grounded on its comment
- [ ] US-005: Enforce faithfulness so no flow is invented
- [ ] US-006: Embed emod-generated diagrams with explanatory prose
- [ ] US-007: Synthesize an executive summary and onboarding walkthrough
- [ ] US-008: Keep documentation in sync with a CI drift gate
- [ ] US-009: Choose between a docs tree and a single file

## [09-bdd-test-generation.md](./ai/09-bdd-test-generation.md) — 0/9

- [ ] US-001: Generate Given/When/Then scenarios from a model's slices
- [ ] US-002: Fill scenario payloads that conform to the typed field schema
- [ ] US-003: Emit Gherkin feature files and JSON fixtures
- [ ] US-004: Scope generation to a single slice
- [ ] US-005: Generate negative scenarios for missing required fields
- [ ] US-006: Keep generated scenarios faithful to the model's flows
- [ ] US-007: Regenerate fixtures that fail the conformance check
- [ ] US-008: Surface implied edge and branch cases the model gestures at
- [ ] US-009: Report token cost and machine-readable results per run

## [10-conversational-viewer-editing.md](./ai/10-conversational-viewer-editing.md) — 0/12

- [ ] US-001: Show or hide the chat panel based on backend availability
- [ ] US-002: Instruct an edit in natural language and see the proposed result
- [ ] US-003: Only valid edits are ever proposed
- [ ] US-004: Review a proposed edit as a diff before applying
- [ ] US-005: Accept a proposed edit and see the diagram re-render live
- [ ] US-006: Reject a proposed edit
- [ ] US-007: Knowingly accept an edit that carries a residual lint warning
- [ ] US-008: Undo an applied edit
- [ ] US-009: Ask read-only questions about the model
- [ ] US-010: See progress and cost for each turn
- [ ] US-011: Target an edit at the selected element or context
- [ ] US-012: Invalidate a pending proposal when the model changes underneath it

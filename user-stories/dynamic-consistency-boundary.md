# Dynamic Consistency Boundary Support

## Overview

Allow `.emod` files to express Dynamic Consistency Boundary (DCB) models — where events carry tags rather than belonging to a single aggregate, and each command declares its own consistency boundary via a query over the tag space. Aggregate-style models remain valid and idiomatic; DCB is additive.

## Goals

- Model authors can write DCB-style slices with tagged events and tag-scoped decision queries
- Existing aggregate-style models continue to work without modification
- The tool surfaces DCB-specific anti-patterns (overly broad queries, untagged events, single-tag-everywhere)
- Diagrams and formatting handle both styles from a single internal representation

## User Stories

### US-001: Author a DCB context with tagged events and decision queries
**Description:** As a model author, I want to define a context that uses DCB style — with slices directly under the context, tagged events, and commands that declare a `decides_on` query — so that I can model cross-cutting consistency boundaries without inventing artificial aggregates.

**Acceptance Criteria:**
- [ ] A context with `mode dcb` accepts a `slice` block directly under it (no aggregate parent) without error
- [ ] An event in a DCB context can declare a `tags` clause with one or more `key:field` entries
- [ ] A command in a DCB context can declare a `decides_on` block listing event types and a `where` predicate using `tag(key = fieldRef)`, `and`, `or`, and `not`
- [ ] A context with `mode aggregate` (or no mode set) warns when a slice lacks an aggregate parent or uses DCB-only constructs (`tags`, `decides_on`)
- [ ] A context with `mode dcb` warns when it contains an `aggregate` block
- [ ] A context with `mode mixed` accepts both DCB and aggregate constructs without warnings
- [ ] Invalid references — a tag key not declared on any referenced event, a field reference that doesn't match a declared field, or an event name in `decides_on.events` that doesn't exist — each produce a clear error with location
- [ ] An existing aggregate-based `.emod` file parses successfully with no new errors or DCB-related warnings

**Context:** This is the primary authoring experience. The parser accepts the new grammar. The mode flag controls linter warnings; the grammar always accepts both forms. Validation catches structural errors (bad references, mode mismatches) at parse time so the author gets immediate feedback. Backward compatibility is inherent — aggregate models require no changes.

### US-002: Surface DCB anti-patterns via linting
**Description:** As a model author, I want to run `emod lint` on a DCB model and get actionable warnings about common anti-patterns, so that I can keep my models clean and meaningful.

**Acceptance Criteria:**
- [ ] An event in a DCB-context slice without a `tags` clause triggers an `dcb/untagged-event` error
- [ ] A `decides_on` that references more than 5 event types triggers a `dcb/query-too-broad` warning
- [ ] A `decides_on` with a missing predicate or a predicate that is always `true` triggers a `dcb/query-too-broad` warning
- [ ] A context where every command uses only one distinct tag key triggers an `dcb/single-tag-everywhere` info message
- [ ] A tag key declared on events but never referenced in any command's `decides_on` triggers a `dcb/orphan-tag-key` warning
- [ ] Running `emod lint --explain dcb/query-too-broad` prints a description of the rule
- [ ] These rules fire only in `dcb` or `mixed` mode; they do not fire in `aggregate` mode (no false positives on existing models)

**Depends on:** US-001

### US-003: Visualize DCB models in diagrams
**Description:** As a model author, I want to generate diagrams from a DCB model that show tags and decision queries, so that I can communicate the cross-cutting consistency boundaries to my team.

**Acceptance Criteria:**
- [ ] Running `emod diagram` on a DCB context produces tag-projected swim lanes (one lane per primary tag key) by default
- [ ] An event with multiple tags appears in multiple lanes with a visible connector
- [ ] Running `emod diagram --style=dcb` produces a query-lens view with events on a single timeline, tags as colored badges, and commands rendered as labelled brackets showing the event types and predicate
- [ ] Mermaid and draw.io output both support the projected swim-lane variant
- [ ] Aggregate-based models continue to produce aggregate-grouped swim lanes with no change

**Depends on:** US-001

**Context:** DCB slices have no aggregate to group by, so diagrams need a different layout. Two styles ship: projected swim lanes (familiar layout, one lane per tag) and query-lens (flat timeline, tag badges, command brackets).

### US-004: Format DCB constructs consistently
**Description:** As a model author, I want `emod fmt` to format DCB constructs consistently — preserving tags, `decides_on`, and `where` predicates — so that my team's models follow a uniform style.

**Acceptance Criteria:**
- [ ] A DCB model with `tags`, `decides_on`, and `where` parses and formats round-trip without data loss
- [ ] `emod fmt` applies consistent indentation and line-breaking to `tags [...]`, `decides_on { ... }`, and `where` predicates
- [ ] `emod fmt` does not reorder tag entries or predicate structure
- [ ] The formatter handles both DCB and aggregate constructs in a `mode mixed` context

**Depends on:** US-001

## Non-Goals

- Generating runtime DCB infrastructure (event store, append-condition checking)
- Auto-migrating existing aggregate models to DCB
- Adding set-membership predicates (`tag(course in courseIds)`) — deferred until a concrete model demands it
- LSP autocomplete for tag keys or hover on `decides_on` events (Phase 4)
- VS Code syntax highlighting for the new keywords (Phase 4)

## Open Questions

- None — the proposal covers scope, DSL surface, and implementation phases thoroughly. Assumptions were noted where trade-offs exist.

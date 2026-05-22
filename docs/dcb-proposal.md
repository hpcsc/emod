# Dynamic Consistency Boundary Support — Proposal

## Problem

`emod` currently organises slices under aggregates: `Context → Aggregate → Slice`. Events implicitly belong to their parent aggregate, and consistency is implied by the aggregate's `stream "X-{id}"` pattern. This bakes a single worldview — "one aggregate owns the events and the decision" — into the AST, parser, linter, and diagram output.

Dynamic Consistency Boundary (DCB) inverts that model:

- Events are not owned by a single aggregate. They carry one or more **tags** (e.g. `course:c1`, `student:s42`).
- Each command/decision declares its own **query** over the tag space.
- That query *is* the consistency boundary — it determines which events are loaded to build state and which appended events would cause an optimistic-concurrency conflict.

Modelling DCB-style systems in `emod` today forces authors to either:

1. Pick an "owning" aggregate per slice and hide the cross-cutting nature in prose, or
2. Duplicate the same event across multiple aggregates.

Both lose the signal that makes DCB useful: that the consistency boundary is *per decision*, not *per type*.

## Goals

- Allow `.emod` files to express DCB-style slices: tagged events, tag-scoped commands, decision queries.
- Keep existing aggregate-style models valid and idiomatic — DCB is additive, not a replacement.
- Surface DCB-specific anti-patterns in `emod lint` (query too broad, untagged event, single-tag-everywhere).
- Provide a single internal representation so `fmt`, `lint`, and diagram generation don't fork into two pipelines.

## Non-Goals

- Generating runtime DCB infrastructure (event store, append-condition checking). `emod` describes the model; runtime is out of scope.
- Auto-migrating existing aggregate-style models to DCB.
- Inventing new DCB semantics. This proposal follows the established meaning of tags, queries, and append conditions from the DCB literature.

---

## DSL Surface

Two new affordances and one relaxation.

### 1. Slices may live directly under a context

Today, every slice must sit inside an aggregate. Relax the grammar so a `slice` block is valid directly under `context`:

```emod
context CourseEnrollment {
  slice "Subscribe Student to Course" {
    # no aggregate parent
    ...
  }
}
```

Aggregate-nested slices remain valid and unchanged.

### 2. Events declare tags

Events gain an optional `tags` clause. A tag is `key:fieldName` where `fieldName` references a field declared on the event (and, by convention, on related commands):

```emod
event StudentSubscribed {
  tags [course:courseId, student:studentId]
  fields {
    courseId    CourseID    required
    studentId   StudentID   required
    subscribedAt Timestamp  required
  }
}
```

When an event sits inside an `aggregate X { stream "X-{id}" }`, the aggregate's stream pattern desugars to an implicit tag `x:id` on every contained event. This is the bridge that lets the two styles share one internal representation.

### 3. Commands declare a decision query

Commands gain an optional `decides_on` block:

```emod
command SubscribeStudent {
  tags [course:courseId, student:studentId]
  decides_on {
    events [StudentSubscribed, StudentUnsubscribed, CoursePublished, CourseClosed]
    where  tag(course = courseId) or tag(student = studentId)
  }
  fields {
    courseId   CourseID   required
    studentId  StudentID  required
  }
}
```

- `events [...]` — the event types loaded to build decision state.
- `where` — a tag predicate combining the command's own tags with `and` / `or`. Grammar is intentionally small: `tag(key = fieldRef)`, parenthesisation, `and`, `or`, `not`.

If a slice is under an aggregate and no `decides_on` is given, the implicit query is "all events in this aggregate stream" — i.e. today's behaviour.

### 4. Optional context-level mode hint

For tooling clarity, a context may declare a mode:

```emod
context CourseEnrollment {
  mode dcb
  ...
}
```

Modes:

- `mode aggregate` — default. Slices require an aggregate parent. DCB-only constructs (`tags`, `decides_on`) warn.
- `mode dcb` — slices may live directly under context. Aggregate-only assumptions (single-event ownership) warn.
- `mode mixed` — explicit opt-in to both styles in one context; neither side warns.

The mode flag is a *linter hint*, not a parser switch. The grammar always accepts both forms; the mode determines which anti-patterns fire.

---

## Internal Representation

### AST changes (`internal/ast/ast.go`)

- `Context.Slices []*Slice` — slices allowed directly under context.
- `Context.Mode string` (`""`, `"aggregate"`, `"dcb"`, `"mixed"`).
- `Event.Tags []*TagDecl` where `TagDecl{ Key string; FieldRef string; Position }`.
- `Command.Tags []*TagDecl`.
- `Command.DecidesOn *DecisionQuery` where:

```go
type DecisionQuery struct {
    Events    []string          // event type names
    EventPos  []Position
    Predicate *TagPredicate     // tree of and/or/not over tag(key = fieldRef)
    OpenPos   Position
    ClosePos  Position
}

type TagPredicate struct {
    Op       string // "and", "or", "not", "match"
    Children []*TagPredicate
    Key      string // for "match"
    FieldRef string // for "match"
    Position
}
```

Aggregate stream patterns desugar at a normalisation pass after parsing, before linting/validation: every event under `aggregate X { stream "X-{id}" }` gains an implicit `tags [x:id]`, and every command under it gains an implicit `decides_on { events [<all events in aggregate>] where tag(x = id) }`. From the linter's point of view, both styles look the same.

### Parser changes

- Allow `slice` token at context scope.
- New grammar productions for `tags [...]`, `decides_on { ... }`, and `where`-predicate expressions.
- Reserved words: `tags`, `decides_on`, `where`, `tag`, `and`, `or`, `not`, `mode`, `dcb`, `aggregate`, `mixed`.

### Validator changes (`internal/validator`)

- **Tag key resolution**: every `tag(k = f)` in a `decides_on` predicate must have a matching declared tag key `k` on at least one event referenced in `events [...]`.
- **Field reference resolution**: `fieldRef` in any `tag(k = fieldRef)` must resolve to a field on the surrounding command (for command tags) or event (for event tags).
- **Event type resolution**: every name in `decides_on.events` must reference an event defined in the model.
- **Mode consistency**: when `mode aggregate` is set, error on slices that lack an aggregate parent or declare `decides_on`. When `mode dcb` is set, error on `aggregate` blocks.

### Linter changes (`internal/linter`)

New rules, all gated by mode where appropriate:

| Rule | Mode | Severity | Check |
|---|---|---|---|
| `dcb/untagged-event` | dcb, mixed | error | Events in a DCB slice must declare `tags`. |
| `dcb/query-too-broad` | dcb, mixed | warning | `decides_on.events` lists more than 5 event types, or predicate is missing/`true`. |
| `dcb/single-tag-everywhere` | dcb | info | All commands in the context use only one tag key — DCB is acting as aggregates with extra steps. |
| `dcb/orphan-tag-key` | dcb, mixed | warning | A tag key is declared on events but never used in any command's `decides_on`. |
| `aggregate/cross-aggregate-command` | aggregate | info | A command's flow references events from another aggregate — DCB candidate. |

Existing rules (past tense events, imperative commands, no state obsession, etc.) continue to apply unchanged.

### Diagram changes

Existing swim-lane diagrams group columns by aggregate. DCB slices have no aggregate to group by. Two additions:

1. **Tag-projected swim lanes** (default for `mode dcb`): one lane per primary tag key encountered in the slice. An event tagged `[course, student]` appears in both lanes with a connector.
2. **DCB query lens** (`--style=dcb`): events on a single horizontal timeline, tags rendered as coloured badges; each command rendered as a labelled bracket showing the event types and tag predicate it queries.

Mermaid and draw.io outputs both support the projected swim-lane variant. The query-lens style ships first as SVG only.

---

## Worked Example

```emod
model "Course Enrollment" {
  actor Student

  context Enrollment {
    mode dcb

    slice "Subscribe Student to Course" {
      trigger UI "Course Signup Form" {
        actor Student
      }

      command SubscribeStudent {
        tags [course:courseId, student:studentId]
        decides_on {
          events [
            StudentSubscribed,
            StudentUnsubscribed,
            CoursePublished,
            CourseClosed,
          ]
          where tag(course = courseId) or tag(student = studentId)
        }
        fields {
          courseId   CourseID   required
          studentId  StudentID  required
        }
      }

      event StudentSubscribed {
        tags [course:courseId, student:studentId]
        fields {
          courseId     CourseID     required
          studentId    StudentID    required
          subscribedAt Timestamp    required
        }
      }

      command -> event: SubscribeStudent -> StudentSubscribed
    }
  }
}
```

A `decides_on` query that loads `StudentSubscribed`, `StudentUnsubscribed`, `CoursePublished`, and `CourseClosed` would, in aggregate style, force the modeller to invent a `CourseEnrollment` aggregate that owns events touching both Course and Student lifecycles — a fiction the DCB form avoids.

---

## Phased Implementation

### Phase 1: Grammar and AST

- AST fields for tags, `decides_on`, mode.
- Parser productions and tests.
- Normalisation pass that desugars aggregate stream patterns into implicit tags.
- `emod fmt` round-trips both forms.

### Phase 2: Validation and Linting

- Validator: tag/field/event resolution, mode consistency.
- Linter: DCB rules listed above.
- `emod lint --explain dcb/query-too-broad` documents each rule.

### Phase 3: Diagrams

- Tag-projected swim lanes (Mermaid + draw.io).
- `--style=dcb` query-lens SVG.

### Phase 4: Tooling and Docs

- Update `examples/all_patterns.emod` to include a DCB context.
- New example: `examples/dcb_course_enrollment.emod`.
- LSP additions (autocomplete for tag keys, hover on `decides_on` events).
- `editors/vscode` syntax highlighting for the new keywords.

---

## Risks and Open Questions

- **Predicate grammar scope.** Starting with `and`/`or`/`not` over `tag(k = f)` is conservative. Real DCB systems sometimes need set membership (`tag(course in courseIds)`) or value matching beyond field equality. Worth deferring until a concrete model demands it.
- **Aggregate desugaring fidelity.** Treating an aggregate as "implicit tag + implicit query over all aggregate events" is correct for most cases but loses the snapshot/projection semantics some teams attach to aggregates. If users rely on those, expose them explicitly rather than implying them.
- **Diagram readability.** Tag-projected swim lanes can sprawl when slices touch 4+ tag keys. The query-lens view handles wide slices better but is unfamiliar. Both should ship; let usage decide the default.
- **Mode inference vs explicit `mode` keyword.** Inferring mode from the presence of `decides_on` is tempting but makes lint behaviour depend on subtle authoring choices. Explicit `mode` is verbose but predictable. Proposal picks explicit; revisit after Phase 2 usage data.
- **Migration story.** No automated migration. An `emod fmt --to=dcb` flag is feasible later if demand appears; it would lift aggregate slices to context level and materialise the implicit tags. Out of scope for Phase 1.

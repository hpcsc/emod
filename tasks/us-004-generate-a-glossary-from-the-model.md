# US-004: Generate a glossary from the model

## Progress
- [x] Task 1: Render a markdown glossary of the model, its contexts and their aggregates
- [x] Task 2: List each context's commands, events, views and actors in the glossary
- [x] Task 3: Emit the same glossary as structured JSON under `-f json`
- [ ] Task 4: Document `emod glossary` in the README and the DSL reference

---

## Story Reference

`user-stories/specs-and-metadata.md` → **US-004: Generate a glossary from the model** (fourth story
of "Specs, Invariants, and Model Metadata"). Design notes: `docs/proposals/specs-and-metadata-proposal.md`
§2 "Descriptions and Glossary", lines 86-101.

**In scope:** a new `emod glossary <file>` subcommand; markdown to stdout grouped by context; each
context's aggregates, commands, events, views and actors with their descriptions; the same content
as structured JSON under `-f json`; an empty definition wherever a construct carries no description.

**Out of scope:** named invariants and their glossary entries (US-005 — the glossary's per-context
entry list is the seam they slot into, and nothing in this story should make that awkward); wire
types on events (US-012, and the story's open questions record "assumed not, until a consumer asks");
descriptions on `slice`, `trigger`, `automation` and `translation`, which the story's criteria do not
name (the event nested inside a `translation` is still an event and does appear); LSP hover
(US-015); the WASM surface in `internal/wasm/pipeline.go`, which exposes diagram and JSON exports
only; `e2e/tests/`, which covers `validate` alone; any change to the existing `validate`, `fmt`,
`lint`, `export`, `diagram`, `slices` or `schema` commands.

**Discovered, deliberately not fixed here:** `README.md:114-128` documents `-f json` / `-f cue` for
`export` and `diagram`, and neither works — no command defines an `f` alias, and urfave/cli v2 stops
flag parsing at the first positional argument (verified: `emod export <file> -f cue` prints JSON, and
`emod slices <file> --format json` prints the text table). This story's criteria are written as
`emod glossary <file> -f json`, so Task 1 makes that spelling work **for the glossary command**;
retrofitting the other six commands is a separate change.

**Learnings folded in** from `tasks/learnings.md`: CLI diagnostic tests must assert the distinguishing
message text, not just the path and position; acceptance criteria never reference commit, branch or
remote state; the shared undescribed/described fixture pair is the witness for "gaps are visible";
de-duplicate before a fan-out edit (one document, two renderers); `docs/dsl-reference.md` anchors
embed the section number, so nothing may be renumbered; run repo tooling through `mise exec --`.

---

## Codebase Context

**Command surface.** `internal/cli/app.go` (`NewApp`) registers every subcommand on one
`urfave.App`; each `Action` reads the positional path plus a `format` string flag, calls a
`Run<Command>` function, and maps a returned `*cli.LintError` onto `urfave.Exit` with its exit code.
`cmd/emod/main.go:10` is the only caller of `NewApp().Run`. No test in the repo exercises `NewApp`
today — every CLI test calls `RunValidate` / `RunSlices` / … directly.

**Nearest sibling.** `internal/cli/slices.go` is the closest shape to this story: it rejects an
unsupported format with `ErrUnsupportedFormat` (`:22-28`), rejects an empty path with
`ErrMissingFileArgument` (`:30-36`), reports a read failure naming the path (`:38-44`), runs
lexer + parser only — no validator, no linter — and turns any diagnostics into a single `LintError`
carrying every diagnostic line (`:46-61`). `internal/cli/export.go:47-57` shows the other choice
(validate + lint before writing); a read-only rendering command follows `slices`.

**Error and format plumbing.** `internal/cli/lint.go:17-35` declares `ErrMissingFileArgument`,
`ErrUnsupportedFormat` and `LintError` (`Message`, `ExitCode`, `Cause`, with `Unwrap`). Output goes
to stdout via `fmt.Println`; diagnostics reach the user through the returned error, which
`app.go` prints to stderr.

**AST.** `internal/ast/ast.go` carries `Description` (plus `DescriptionPos`) on `Model` (:23),
`Actor` (:35), `Context` (:45), `Aggregate` (:59), `Slice` (:70), `Command` (:88), `Event` (:100),
`Trigger` (:135), `View` (:149), `Automation` (:162) and `Translation` (:178). Slices have two homes:
`Context.Aggregates[].Slices` (aggregate mode) and `Context.Slices` (DCB mode, `mode dcb`).
`internal/diagram/drawio.go:619-631` (`collectSlices`) and `convertModelToDiagram`
(`internal/export/export.go:895-940`) enumerate both. Two places visit only the first and are the
shape *not* to copy: `internal/cli/slices.go:146-159`, and `convertContext`
(`internal/export/export.go:318-331`), which silently drops a DCB context's slices from the JSON
export — a pre-existing gap, out of scope here. An `Event` also occurs nested at
`Translation.Event`, and `docs/dsl-reference.md:409` records
that it accepts a description like any other event. Actors are declared only at model scope
(`Model.Actors`) and referenced only from `Trigger.Actor`; nothing in `internal/validator` resolves
that reference, so a trigger may name an actor the model never declares.

**Renderer packages.** `internal/export` (`ExportJSON`, `ExportCUE`) and `internal/diagram`
(`ExportDrawio`, `ExportSVG`, `ExportMermaid`, `ExportASCII`) take `*ast.Model` and return
`([]byte, error)`; `internal/cli/*.go` stays a thin adapter over them. `internal/export/export.go:16-189`
declares document types (`jsonModel`, `jsonContext`, …) separate from the AST so serialization
concerns never leak into the domain types — every `Description` there is tagged `omitempty`, which is
the opposite of what this story's fourth criterion needs.

**Test conventions.** `internal/cli` tests are `//go:build unit`, package `cli_test`, one umbrella
`Test<Command>` per file with `t.Run` groups, `testify/require` throughout. Shared helpers and
fixtures already visible to a new file in that package: `captureStdout` (`internal/cli/lint_test.go:133`),
`writeTemp` (`internal/cli/validate_test.go:514`), and the constants `validEmod` /
`describedEmod` / `invalidEmod` (`internal/cli/validate_test.go:20-23`, aliasing
`test.HotelReservation`, `test.DescribedHotelReservation`, `test.Unparseable`) plus
`singleTagDCBEmod` (`internal/cli/lint_test.go:21`), a two-slice DCB context with no aggregate.
`internal/test/fixtures.go` needs no new fixture: `HotelReservation` and
`DescribedHotelReservation` are the same model without and with descriptions on every construct, and
that pair is exactly the witness the "empty definition" criterion wants. Neither has more than one
context — a multi-context case is built as an `ast.Model` literal (as `internal/export/export_test.go:24-60`
does) or parsed from a small inline source (as `internal/cli/lint_test.go:21` does).

**Diagnostic wording.** A parse failure on `test.Unparseable` reports
`<path>:1: unrecognized keyword "foobar"; expected one of: actor, context, model` — the token text
and the expected set are what a test must pin, per the first entry in `tasks/learnings.md`.

---

## Tasks

### Task 1: Render a markdown glossary of the model, its contexts and their aggregates

**Behavior:** `emod glossary <file>` writes a markdown glossary to stdout — the model with its
description, then one section per context in declaration order with the context's description and
the aggregates declared in it, each with its description. A construct with no description renders
with an empty definition rather than being dropped. The command is registered in `NewApp` with a
format flag that is honoured in both the `--format` and `-f` spellings and in either position
relative to the file argument; `markdown` is the default and the only value this task accepts.

**Acceptance Criteria:**
- [ ] Running the glossary over a model with two contexts writes markdown to stdout naming both
      contexts in declaration order, each followed by its description
- [ ] The aggregates declared in a context appear under that context, in declaration order, each
      paired with its description
- [ ] Over `test.HotelReservation` (a model whose constructs carry no descriptions) the same model
      name, context and aggregate all still appear, each with an empty definition
- [ ] Over `test.DescribedHotelReservation` each of those terms appears with the description written
      in the fixture
- [ ] An empty path fails with an error unwrapping to `cli.ErrMissingFileArgument` and a
      `*cli.LintError` whose `ExitCode` is 1
- [ ] A path that cannot be read fails with a message naming the path and `ExitCode` 1
- [ ] A file that does not parse fails with `ExitCode` 1, prints no glossary, and the message names
      the offending token text and the keywords the parser expected — not merely the path and line
- [ ] A format other than `markdown` fails with an error unwrapping to `cli.ErrUnsupportedFormat`
      whose message names the supported formats
- [ ] `emod glossary <file> -f xml` through `NewApp` — flag written after the file argument — fails
      with that unsupported-format error, and `emod glossary --format markdown <file>` succeeds,
      showing the flag is read in either position and under either spelling
- [ ] `emod --help` lists `glossary` among the commands

**Affected Files/Modules:**
- `internal/glossary/` (new) — the glossary document built from `*ast.Model`, and its markdown
  rendering
- `internal/glossary/glossary_test.go` (new) — behaviour of the exported rendering entry point
- `internal/cli/glossary.go` (new) — `RunGlossary`, the thin adapter: read, lex, parse, render
- `internal/cli/glossary_test.go` (new) — CLI behaviour and the error surface
- `internal/cli/app.go` — register the `glossary` command and its format flag

**Patterns to Follow:**
- Command shape, error surface and the lex/parse-only pipeline: `internal/cli/slices.go:21-72`
- Renderer package boundary and signature: `internal/diagram/mermaid.go:18` and
  `internal/export/export.go:250`; document types kept separate from the AST as in
  `internal/export/export.go:16-189`
- Flag and `LintError` handling inside an `Action`: `internal/cli/app.go:189-216`
- Test file layout, umbrella/`t.Run` grouping, `captureStdout` and `writeTemp` reuse:
  `internal/cli/slices_test.go`
- Assert the distinguishing text of a diagnostic, not just path and position — `tasks/learnings.md`
  "CLI diagnostic tests must assert the distinguishing message text", with
  `internal/cli/validate_test.go:253-258` as the model
- Do not add a fixture to `internal/test/fixtures.go`: this story adds no language construct, and
  `HotelReservation` / `DescribedHotelReservation` already form the undescribed/described pair

**Testable:** Yes — through `cli.RunGlossary`, the exported renderer, and `cli.NewApp().Run` for the
flag-position criteria.

**Verification:** `mise exec -- go test -tags unit ./internal/glossary/... ./internal/cli/...`;
`go build ./...`; `go run ./cmd/emod glossary examples/all_patterns.emod`.

**Depends on:** None

---

### Task 2: List each context's commands, events, views and actors in the glossary

**Behavior:** Every context section additionally lists the commands, events, views and actors that
belong to it, each with its description, so the glossary carries the model's whole named vocabulary.
Commands, events and views are collected from the slices in the context's aggregates and from the
slices declared directly on the context in DCB mode; the event declared inside a `translation` counts
as an event. Actors are those the context's triggers reference, resolved to their model-level
declaration for the description and listed once however many triggers name them; an actor the model
declares but no trigger references still appears in the glossary, so no declared term is lost.

**Acceptance Criteria:**
- [ ] Over `test.DescribedHotelReservation`, the context section lists `MakeReservation`,
      `ConfirmReservation` and `ImportBooking` as commands, `ReservationMade` and `BookingImported`
      as events, `ReservationsView` as a view and `Guest` as an actor, each with the description the
      fixture gives it
- [ ] `BookingImported`, declared inside the `translation` block, is among those events
- [ ] Over `test.HotelReservation` every one of those terms still appears, each with an empty
      definition
- [ ] Over `singleTagDCBEmod`, whose slices hang directly off a `mode dcb` context and which has no
      aggregate, `PlaceOrder`, `AuthorizePayment`, `OrderPlaced` and `PaymentAuthorized` appear under
      the `Fulfillment` context
- [ ] An actor referenced by two triggers within one context is listed once for that context
- [ ] An actor a model declares and no trigger references appears in the glossary with its
      description
- [ ] An actor named by a trigger but never declared appears with an empty definition rather than
      being dropped
- [ ] Rendering the same file twice produces byte-identical output, so ordering is declaration order
      and never map order

**Affected Files/Modules:**
- `internal/glossary/` — the walk that collects a context's terms, and the markdown sections for them
- `internal/glossary/glossary_test.go` — the collection and grouping behaviour
- `internal/cli/glossary_test.go` — the terms as they reach stdout for the shared fixtures

**Patterns to Follow:**
- Visit both slice homes: `internal/diagram/drawio.go:619-631` (`collectSlices`), not
  `internal/cli/slices.go:146-159` or `internal/export/export.go:318-331`, both of which visit only
  `Context.Aggregates`
- Kind-by-kind traversal of a slice's constructs: `internal/export/export.go:370-390`
  (`convertSlice`)
- Keep one collected document behind both renderings rather than a second walk for the JSON
  rendering that follows — `tasks/learnings.md` "De-duplicate before a fan-out edit"
- Name any extracted helper after the contract its callers rely on — `tasks/learnings.md` "Name an
  extracted helper after the contract its callers rely on"
- Leave room beneath the aggregate and the context for the invariants US-005 adds, rather than a
  structure keyed only to the four term kinds named here

**Testable:** Yes — through the exported renderer and `cli.RunGlossary`.

**Verification:** `mise exec -- go test -tags unit ./internal/glossary/... ./internal/cli/...`;
`go run ./cmd/emod glossary examples/dcb_model.emod`.

**Depends on:** Task 1

---

### Task 3: Emit the same glossary as structured JSON under `-f json`

**Behavior:** `emod glossary <file> -f json` writes one JSON document to stdout carrying the same
terms, groupings and descriptions as the markdown rendering. A construct with no description carries
an explicit empty description in the JSON rather than an omitted key, so a consumer can see the gap.

**Acceptance Criteria:**
- [ ] `emod glossary <file> -f json`, with the flag after the file argument, writes a single valid
      JSON document to stdout and exits 0 for a file that parses
- [ ] For `test.DescribedHotelReservation`, every context, aggregate, command, event, view and actor
      the markdown names is present in the decoded JSON, carrying the same description text
- [ ] For `test.HotelReservation`, each of those terms is present in the JSON with a description key
      that is present and empty — the key is not omitted, unlike the `omitempty` descriptions in
      `internal/export`
- [ ] Term order within the JSON matches the order the markdown lists them in
- [ ] `json` and `markdown` are both named by the unsupported-format message; a third value still
      fails with `cli.ErrUnsupportedFormat`
- [ ] The markdown rendering for both fixtures is unchanged by this task

**Affected Files/Modules:**
- `internal/glossary/` — the JSON rendering over the document Task 2 completed
- `internal/glossary/glossary_test.go` — parity between the two renderings, and the empty-description
  key
- `internal/cli/glossary.go` — accept and dispatch the `json` format
- `internal/cli/glossary_test.go` — the JSON as it reaches stdout, and the `-f json` spelling through
  `NewApp`

**Patterns to Follow:**
- Two renderings of one model asserted against each other: the "CUE and JSON exports describe the
  same model" subtest, `internal/export/export_test.go:3332`
- Decode and assert on the document rather than on the serialized text:
  `internal/export/export_test.go:24-60`
- JSON emission and marshalling-failure handling in a CLI adapter: `internal/cli/slices.go:117-139`
- Struct tags for the document type: `internal/export/export.go:16-189` — but the description key
  here must survive an empty value, which `omitempty` there does not

**Testable:** Yes — through the exported renderer, `cli.RunGlossary` and `cli.NewApp().Run`.

**Verification:** `mise exec -- go test -tags unit ./internal/glossary/... ./internal/cli/...`;
`go run ./cmd/emod glossary examples/all_patterns.emod -f json | jq .`

**Depends on:** Task 2

---

### Task 4: Document `emod glossary` in the README and the DSL reference

**Behavior:** The README's command walkthrough covers `emod glossary` in both formats, and the DSL
reference lists the glossary among the consumers of `description`, so a reader learning descriptions
finds the command that renders them.

**Acceptance Criteria:**
- [ ] `README.md` gains a glossary section alongside "List slices" (`README.md:130-134`) showing the
      markdown invocation and the `-f json` invocation
- [ ] The consumer bullets in `docs/dsl-reference.md` §10 "Descriptions" (`:456-462`) name
      `emod glossary` and what it renders
- [ ] No numbered heading in `docs/dsl-reference.md` is added, removed or reordered, so every
      existing `(#<n>-<slug>)` link in the document still resolves to its heading
- [ ] Every command shown in the new documentation runs as written against
      `examples/all_patterns.emod`

**Affected Files/Modules:**
- `README.md` — a glossary section in the usage walkthrough
- `docs/dsl-reference.md` — a consumer bullet in §10

**Patterns to Follow:**
- Section voice and the fenced-invocation style of the surrounding usage sections:
  `README.md:114-134`
- Consumer-bullet phrasing already used for exports and diagrams:
  `docs/dsl-reference.md:460-462`
- `tasks/learnings.md` "`docs/dsl-reference.md` anchors embed the section number" — a bullet inside
  an existing section is safe; a new numbered section would renumber everything below it and break
  four in-document links

**Testable:** No — prose only; correctness is that the documented invocations run.

**Verification:** run each documented command against `examples/all_patterns.emod`; confirm the
`^## [0-9]+\.` heading list and the `(#[0-9]+-` link list in `docs/dsl-reference.md` still agree.

**Depends on:** Task 3

---

## Summary

**Total tasks:** 4

**Ordering rationale:** dependency-first, thinnest working slice first. Task 1 stands the whole
pipeline up end to end — package, command, flag surface, error surface — over the smallest content
that is still a glossary (model, contexts, aggregates), so the risky parts (flag position, diagnostic
handling, command registration) are settled before any term collection is written. Task 2 fills in
the term kinds the story's second criterion names, over both slice homes. Task 3 adds the second
rendering once there is a complete document to render, which is what makes the parity assertion
meaningful. Task 4 is documentation and depends on the finished surface.

**Coverage of the story's acceptance criteria:**

| Criterion | Task |
|---|---|
| `emod glossary <file>` writes markdown to stdout, terms grouped by context | 1 |
| Each context lists its aggregates, commands, events, views and actors, each with its description | 1 (aggregates), 2 (the rest) |
| `emod glossary <file> -f json` emits the same content as structured output | 3 (flag surface prepared in 1) |
| A construct without a description appears with an empty definition | 1 (markdown), 2 (all term kinds), 3 (JSON) |

Nothing is deferred. US-005's criterion that the glossary lists invariants under their aggregate or
context stays with US-005; Task 2 keeps the per-aggregate and per-context entry structure open for it.

# Tasks: AI Foundation — LLM Integration Seam

## Progress
- [x] Task 1: Define the `llm.Model` port
- [x] Task 2: Extract the deterministic correctness oracle
- [x] Task 3: Add a mock `llm.Model` for network-free tests
- [x] Task 4: Add the Bedrock-backed `llm.Model` adapter
- [x] Task 5: Add the single AI configuration block
- [ ] Task 6: Support schema-conformant structured output in the adapter
- [ ] Task 7: Add cost and token usage reporting
- [ ] Task 8: Build the generate → validate → lint → repair loop
- [ ] Task 9: Gate AI as opt-in and keep WASM provider-free
- [ ] Task 10: Add the `emod ai` end-to-end smoke command

## Story Reference

Derived from `user-stories/ai/00-llm-foundation.md` (US-001 through US-010). The story
builds the shared LLM integration seam every AI feature (`01`–`10`) depends on: the
`llm.Model` port, a Bedrock adapter, a mock, the deterministic generate → validate →
lint → repair loop, configuration, cost reporting, opt-in gating, and a closing smoke
command. Features `01`–`10` themselves are out of scope.

## Codebase Context

- **Module:** `github.com/hpcsc/emod`, Go 1.25.5. Dependencies are minimal
  (`urfave/cli/v2`, `go-cmp`, `testify`); there is no provider SDK in `go.mod`/`go.sum`
  today, and US-003 introduces the only new heavy dependency (the official Anthropic SDK).
- **Deterministic pipeline (US-005):** assembled inline in two CLI commands.
  `internal/cli/validate.go` (`RunValidate`, lines 40–50) chains
  `lexer.Scan` → `parser.New`/`Parse` → `validator.Validate` → `linter.Lint`, appending
  each stage's `[]*diagnostic.Entry`. `internal/cli/lint.go` (`RunLint`, lines 114–122)
  runs the same lex/parse stages but only lints when there are no earlier diagnostics.
  `internal/wasm/pipeline.go` (`runPipeline`, lines 38–53) assembles the same chain a
  third time. Stage signatures: `lexer.Scan(input, filename string) ([]*lexer.Token, []*diagnostic.Entry)`,
  `parser.New(tokens, filename string) *parser.Instance` with `Parse() (*ast.Model, []*diagnostic.Entry)`,
  `validator.Validate(*ast.Model) []*diagnostic.Entry`, `linter.Lint(*ast.Model) []*diagnostic.Entry`.
- **Diagnostics:** `internal/diagnostic/entry.go` defines `Entry` (Filename, Line, Column,
  Message, Severity, RuleName) and `Severity` (`Error`, `Warning`, `Info`). Exit-code
  policy lives in `internal/cli/lint.go` (`formatJSON`/`formatText`): non-empty diagnostics
  exit 1, presence of an `Error` severity exits 2.
- **CLI:** `internal/cli/app.go` (`NewApp`) registers every command on a
  `urfave/cli/v2` `App`. Each command's `Action` calls a `RunX` function in its own file
  and unwraps `*LintError` (defined in `lint.go`) to map `Message`/`ExitCode` onto
  `urfave.Exit`. `cmd/emod/main.go` is the only entry point that calls `NewApp().Run`.
- **WASM (US-009):** real target at `cmd/emod-wasm/main.go`; the build-tag-free,
  `syscall/js`-free logic lives in `internal/wasm/pipeline.go`. `task test:unit` already
  excludes `/cmd/emod-wasm` from the unit run. The WASM binary is built with
  `GOOS=js GOARCH=wasm` via the `build:wasm` Taskfile task.
- **Config (US-004):** no configuration or environment handling exists anywhere in the
  codebase today; `os.Getenv`/`os.Environ` have zero call sites. This story introduces
  the first config block.
- **Tests:** unit tests use the `//go:build unit` build tag and run via `task test:unit`
  (`go test -tags unit $(go list ./... | grep -v /cmd/emod-wasm)`). Convention (see
  `internal/wasm/pipeline_test.go`, global CLAUDE.md): external `_test` package, one
  umbrella `Test{Type}` per type, `t.Run` groups by operation with behavior-named
  scenarios, `testify/require`, fresh fixtures per leaf subtest.
- **Cost source (US-008):** the `bedrock-cost` tooling already models per-model Bedrock
  pricing and is the assumed source for turning token counts into a cost estimate.

---

## Tasks

### Task 1: Define the `llm.Model` port

**Language:** Go

**Behavior:** A new `llm` package exposes the request/response contract and the `Model`
interface, so feature code can be written and compiled against `llm.Model` alone with no
reference to any concrete client and no provider SDK.

**Acceptance Criteria:**
- [ ] An `internal/llm` package exposes `Message` (role + content), `GenerateRequest`
  (system prompt, messages, optional schema, effort), `Response` (text + usage), and a
  `Model` interface with a single `Generate(ctx, request)` method.
- [ ] `Effort` accepts the documented levels (`low`, `medium`, `high`, `xhigh`); an unset
  value behaves as a sensible default.
- [ ] The `llm` package imports no provider SDK, performs no network or filesystem I/O,
  and adds no new dependency to `go.mod`.
- [ ] Every exported field and the optional schema and effort levels are documented.

**Affected Files/Modules:**
- `internal/llm/` — new package directory.
- `internal/llm/model.go` — new: `Message`, `GenerateRequest`, `Response`, usage type,
  `Effort` and its levels, the default-when-unset behavior, and the `Model` interface.
- `internal/llm/model_test.go` — new: covers the only behavior the package owns at this
  stage — that an unset `Effort` resolves to the documented default.

**Patterns to Follow:**
- For documented-type-plus-`String()` and a constant set with an unset/default case, follow
  `internal/diagnostic/entry.go:5-22` (`Severity` constants and `String()`).
- For exported request/response Go-type contracts kept free of `syscall/js`, follow the
  package doc and type shape in `internal/wasm/pipeline.go:1-34`.

**Testable:** Yes — the effort-default resolution is observable through the package's
public API. Tests live in this task. The struct shapes and the interface are structural
contracts other code compiles against; do not add tests that only assert a struct exists.

**Verification:** `task test:unit` passes; `go build ./...` succeeds; `go.mod` is unchanged.

**Depends on:** None

---

### Task 2: Extract the deterministic correctness oracle

**Language:** Go

**Behavior:** A single function accepts emod source text and returns the combined
diagnostics from the lexer, parser, validator, and linter; `validate` and `lint` adopt it
and keep their current diagnostics and exit codes.

**Acceptance Criteria:**
- [ ] One exported function accepts emod source text (and a filename) and returns the
  combined `[]*diagnostic.Entry` from the lex/parse/validate/lint chain.
- [ ] Clean input returns an empty diagnostic list; broken input returns entries carrying
  position, severity, and message.
- [ ] `RunValidate` and `RunLint` produce the same diagnostics and the same exit codes as
  today after adopting the shared oracle.
- [ ] The oracle imports no LLM package and performs no network I/O.

**Affected Files/Modules:**
- `internal/oracle/oracle.go` — new: the consolidated lex → parse → validate → lint
  function returning combined diagnostics.
- `internal/oracle/oracle_test.go` — new: clean source returns no diagnostics; broken
  source returns positioned, severity-bearing entries.
- `internal/cli/validate.go` — `RunValidate` (lines 40–50) calls the oracle instead of
  assembling the chain inline.
- `internal/cli/lint.go` — `RunLint` (lines 114–122) reuses the oracle while preserving
  its "lint only when earlier stages are clean" behavior.

**Patterns to Follow:**
- The exact chain and append order to consolidate is in `internal/wasm/pipeline.go:38-53`
  (`runPipeline`) and `internal/cli/validate.go:40-50` (`RunValidate`).
- The lint command's conditional-lint behavior to preserve is in
  `internal/cli/lint.go:114-122`. Exit-code mapping to leave unchanged is in
  `internal/cli/lint.go:32-77`.

**Testable:** Yes — diagnostics for a given source are an observable contract (Exported
API caller); the unchanged `validate`/`lint` behavior is covered by the existing
`internal/cli/validate_test.go` and `internal/cli/lint_test.go`.

**Verification:** `task test:unit` passes (including the existing CLI tests unchanged);
`go build ./...` succeeds.

**Depends on:** None

---

### Task 3: Add a mock `llm.Model` for network-free tests

**Language:** Go

**Behavior:** A test double implements `llm.Model`, returns pre-configured responses in
sequence (or a configured error), and records the requests it received so tests can assert
on how a prompt was assembled — all with no network and no credentials.

**Acceptance Criteria:**
- [ ] A type implements `llm.Model` and returns pre-configured responses in call order.
- [ ] The mock records each received request (system, messages, schema, effort) for
  assertion.
- [ ] The mock can be configured to return an error so failure handling can be exercised.
- [ ] The mock returns a different response per call, so repair-loop convergence and
  non-convergence can both be driven later.
- [ ] Tests exercising the mock run with no network access and no credentials.

**Affected Files/Modules:**
- `internal/llm/mock.go` — new: the recording, sequenced-response, error-injecting double
  (placement resolves the story's open question by keeping the mock in the `llm` package
  as an exported helper; revisit only if it pulls in test-only imports).
- `internal/llm/mock_test.go` — new: sequencing returns responses in order, requests are
  recorded, a configured error surfaces, and running past the configured responses behaves
  as documented.

**Patterns to Follow:**
- For a recording/sequenced test double, follow the Test Double Patterns guidance in
  `~/.config/ai/guidelines/go/testing-patterns.md` (Test Double Patterns section) and the
  caller-driven contract shaped by `internal/llm/model.go` from Task 1.
- For umbrella-test structure and `testify/require` usage, follow
  `internal/wasm/pipeline_test.go:1-35`.

**Testable:** Yes — the mock's sequencing, recording, and error behavior are the contract
its future callers depend on, and they are observable through `Generate` and the recorded
requests.

**Verification:** `task test:unit` passes; the mock package imports nothing under a network
or provider SDK.

**Depends on:** Task 1

---

### Task 4: Add the Bedrock-backed `llm.Model` adapter

**Language:** Go

**Behavior:** A concrete adapter implements `llm.Model` by wrapping the official Anthropic
Go SDK's messages surface over the Bedrock client; a `Generate` call returns the model's
text and token usage, using adaptive thinking with `Effort` mapped to the provider's effort
setting.

**Acceptance Criteria:**
- [ ] An adapter implements `llm.Model` over the Anthropic Go SDK's messages surface via
  the Bedrock client.
- [ ] `Generate` returns the model's text and input/output token usage in `Response`.
- [ ] The adapter uses adaptive thinking and maps each `llm.Effort` level to the provider's
  effort setting; no parameter rejected by the model generation in use is sent.
- [ ] The default model is the documented Opus generation, with a cheaper model selectable
  for low-stakes passes.
- [ ] The only new heavy dependency added to `go.mod` is the official Anthropic SDK.

**Affected Files/Modules:**
- `internal/llm/bedrock/adapter.go` — new: the adapter type, its constructor taking the
  resolved settings (region, model IDs), and the `Generate` implementation (effort mapping,
  adaptive thinking, text + usage extraction). Kept in a subpackage so the SDK import is
  isolated from the SDK-free `internal/llm` core and excludable from the WASM build.
- `internal/llm/bedrock/adapter_test.go` — new: effort-to-provider mapping is correct for
  each documented level and the unset default; the adapter satisfies `llm.Model`.
- `go.mod` / `go.sum` — add the official Anthropic Go SDK.

**Patterns to Follow:**
- Implement the `llm.Model` interface defined in `internal/llm/model.go` (Task 1).
- For Bedrock specifics referenced by the story: the SDK's `Messages.New` surface targets
  the Bedrock Mantle client, and model IDs carry the `anthropic.` prefix on Bedrock (e.g.
  `anthropic.claude-opus-4-8`) — read US-003 Context in
  `user-stories/ai/00-llm-foundation.md:61-72` and the official SDK docs directly rather
  than inferring call shapes.
- For asserting a translation/mapping without a network, follow the deterministic-mapping
  test style in `internal/diagram/style_test.go`.

**Testable:** Yes for the pure parts — effort-to-provider mapping and `llm.Model`
conformance are testable without a network. The live round-trip itself is not unit-tested
here (no network in unit tests); it is exercised end-to-end by the smoke command in Task 10.
Do not mock the SDK's own types; assert only on inputs the adapter computes.

**Verification:** `task test:unit` passes; `go build ./...` succeeds; `go.mod` gains only
the Anthropic SDK; `task build:wasm` is not run here (WASM exclusion is Task 9).

**Depends on:** Task 1

---

### Task 5: Add the single AI configuration block

**Language:** Go

**Behavior:** AI configuration is resolved once from environment variables, an
`~/.config/emod` file, and flags, with a defined precedence, and the resolved settings can
be handed to the Bedrock adapter; invalid or missing configuration produces a message that
names exactly what is missing.

**Acceptance Criteria:**
- [ ] A single resolver combines env, an `~/.config/emod` file, and flags into one settings
  value covering Bedrock region, default model, cheap model, default effort, and an optional
  gateway endpoint URL.
- [ ] Precedence between flags, env, and file is implemented and documented (story's assumed
  order: flags > env > file — confirm and document the chosen order).
- [ ] Settings keys `EMOD_AI_REGION`, `EMOD_AI_MODEL`, `EMOD_AI_MODEL_CHEAP`,
  `EMOD_AI_EFFORT`, `EMOD_AI_ENDPOINT` are recognized.
- [ ] Missing or invalid configuration yields a clear message naming exactly what is missing.
- [ ] The resolved settings carry the optional gateway endpoint so a later wiring change can
  route calls through it with no further code change.

**Affected Files/Modules:**
- `internal/llm/config/config.go` — new: the settings struct, the documented precedence
  resolver across env/file/flags, parsing of the documented keys (effort parsed via the
  `llm.Effort` levels from Task 1), and the missing/invalid messages.
- `internal/llm/config/config_test.go` — new: precedence is honored across the three
  sources; each documented key is read; invalid effort and missing required keys produce
  messages naming the offending key; the endpoint is carried through when set.

**Patterns to Follow:**
- This is the first configuration code in the repo; there is no in-repo precedent to copy
  for env/file resolution. For the message-and-exit-code shape, follow the `*LintError`
  pattern in `internal/cli/lint.go:15-22` so config errors map onto exit codes the same way
  other CLI errors do.
- Reuse the `Effort` levels and default from `internal/llm/model.go` (Task 1) for parsing
  `EMOD_AI_EFFORT`.

**Testable:** Yes — precedence, key recognition, and the named-missing-config messages are
observable behaviors (Exported API caller). Drive env and file inputs through the resolver's
parameters/temp files rather than mutating real process env or the real home directory.

**Verification:** `task test:unit` passes; `go build ./...` succeeds.

**Depends on:** Task 4

---

### Task 6: Support schema-conformant structured output in the adapter

**Language:** Go

**Behavior:** When a `GenerateRequest` carries a schema, the adapter requests structured
output and returns content conforming to that schema; non-conforming output surfaces as an
error; requests without a schema continue to return plain text.

**Acceptance Criteria:**
- [ ] A `GenerateRequest` carrying a schema causes the adapter to request structured output
  and return content conforming to that schema.
- [ ] Output that does not conform to the schema is surfaced as an error, not silently
  returned.
- [ ] A feature can supply a schema and receive a validated object it consumes directly.
- [ ] Requests without a schema continue to return plain text.

**Affected Files/Modules:**
- `internal/llm/bedrock/adapter.go` — extend `Generate` to take the strict-tool-use /
  structured-output path when `GenerateRequest.Schema` is set, validate the returned content
  against the schema before returning, and convert a non-conforming result into an error.
- `internal/llm/bedrock/adapter_test.go` — extend: schema-bearing requests select the
  structured-output path; conformance is validated; non-conforming content yields an error;
  schema-less requests stay on the plain-text path.

**Patterns to Follow:**
- Build on the `Generate` implementation and effort mapping from Task 4 in
  `internal/llm/bedrock/adapter.go`; the schema lives on `GenerateRequest.Schema` from
  Task 1 (`internal/llm/model.go`).
- For the SDK's structured-output / strict-tool-use surface, read US-007 Context in
  `user-stories/ai/00-llm-foundation.md:115-126` and the official SDK docs directly.

**Testable:** Yes for the schema-selection and conformance-validation decisions, which the
adapter computes locally; the live structured round-trip is covered end-to-end later. Assert
on the adapter's own validation decision (conforming vs error), not on SDK internals.

**Verification:** `task test:unit` passes; `go build ./...` succeeds.

**Depends on:** Task 1, Task 4

---

### Task 7: Add cost and token usage reporting

**Language:** Go

**Behavior:** Token counts from a `Response` (and accumulated across multiple attempts) are
turned into a reported tokens-used figure and an estimated cost that reflects the model
actually used.

**Acceptance Criteria:**
- [ ] Every `Response` carries input and output token counts (confirmed from Task 1's
  contract and Task 4's population of it).
- [ ] A reporting function turns token counts plus the model used into an estimated cost.
- [ ] The reported cost reflects the model actually used for the run.
- [ ] Usage from every attempt of a run can be accumulated, so a later repair loop reports
  the sum across attempts, not just the final attempt.

**Affected Files/Modules:**
- `internal/llm/cost/cost.go` — new: a per-model pricing table (reusing the `bedrock-cost`
  tooling's pricing as the source), accumulation of usage across attempts, and the
  cost-estimate computation keyed by model ID.
- `internal/llm/cost/cost_test.go` — new: cost is computed from token counts and matches the
  pricing for the model used; a different model yields a different cost; accumulated usage
  across attempts sums correctly; an unknown model is handled with a clear outcome.

**Patterns to Follow:**
- Source the per-model Bedrock pricing from the `bedrock-cost` tooling
  (`~/.claude/plugins/cache/dn-claude/bedrock-cost-plugin/1.0.0/skills/bedrock-cost/SKILL.md`)
  rather than inventing numbers; pricing values are load-bearing test inputs and must come
  from that table, not from the code under test.
- Consume the usage fields from `llm.Response` defined in `internal/llm/model.go` (Task 1).

**Testable:** Yes — cost computation and usage accumulation are deterministic, observable
outputs (Exported API caller). Expected cost values come from the pricing table (domain
knowledge), not from running the function under test.

**Verification:** `task test:unit` passes; `go build ./...` succeeds.

**Depends on:** Task 1, Task 4

---

### Task 8: Build the generate → validate → lint → repair loop

**Language:** Go

**Behavior:** A reusable loop asks the model for an emod source, runs it through the oracle,
returns the source as soon as there are no diagnostics, otherwise feeds the offending source
and diagnostics back for another attempt, and stops with a clear "did not converge" outcome
after the maximum attempts — depending only on `llm.Model` and the oracle.

**Acceptance Criteria:**
- [ ] Given a prompt and a maximum attempt count, the loop calls the model, runs output
  through the oracle, and returns the source as soon as there are no diagnostics.
- [ ] When diagnostics remain, the loop feeds the offending source and the diagnostics back
  to the model for another attempt.
- [ ] After the maximum attempts the loop returns a clear "did not converge" outcome.
- [ ] The loop imports only `internal/llm` and the oracle — no provider SDK and no bedrock
  subpackage.
- [ ] Both convergence and non-convergence are demonstrated with the mock model and no
  network.

**Affected Files/Modules:**
- `internal/llm/repair/repair.go` — new: the `GenerateAndRepair`-style loop taking an
  `llm.Model`, a prompt, and a max-attempts bound; calling the oracle each round; building
  the next request from the prior source plus diagnostics; and returning either the clean
  source or a "did not converge" outcome (carrying accumulated usage for Task 7/10).
- `internal/llm/repair/repair_test.go` — new: converges on the first clean response;
  converges after a repair round; reports non-convergence at the attempt bound; feeds prior
  diagnostics into the follow-up request (asserted via the mock's recorded requests).

**Patterns to Follow:**
- Drive both convergence and non-convergence with the per-call sequenced responses and
  recorded requests of the mock from Task 3 (`internal/llm/mock.go`).
- Call the oracle from Task 2 (`internal/oracle/oracle.go`) as the correctness check; build
  on `llm.GenerateRequest`/`Message` from Task 1 for the follow-up request.
- For the "loop depends only on the interface plus the deterministic pipeline" principle,
  read US-006 Context in `user-stories/ai/00-llm-foundation.md:101-113`.

**Testable:** Yes — convergence, repair-on-diagnostics, and non-convergence are the loop's
core observable behaviors and are fully drivable with the mock (Async/Exported API caller),
no network. This is the primary harness later features test against.

**Verification:** `task test:unit` passes; `go build ./...` succeeds; the `repair` package
imports no provider SDK.

**Depends on:** Task 1, Task 2, Task 3

---

### Task 9: Gate AI as opt-in and keep WASM provider-free

**Language:** Go

**Behavior:** With no AI configuration present, the existing commands run exactly as today
and AI commands are absent or clearly report "AI not configured"; the WASM build excludes
the provider SDK and credentials and still builds; no existing non-AI path gains a hard
dependency on an LLM or the network.

**Acceptance Criteria:**
- [ ] With no AI configuration, `validate`, `lint`, `diagram`, `export`, and `lsp` behave
  exactly as today.
- [ ] With no AI configuration, AI commands are absent or clearly report that AI is not
  configured, and never break the other commands.
- [ ] The WASM build (`GOOS=js GOARCH=wasm`) excludes the provider SDK and credentials and
  builds successfully.
- [ ] No existing non-AI code path imports an LLM package or the network.

**Affected Files/Modules:**
- `internal/cli/ai.go` — new (or extend `app.go`): wiring that constructs the adapter only
  when config resolves, otherwise yields a clear "AI not configured" outcome; AI command
  registration that depends on `internal/llm/config` (Task 5) and `internal/llm/bedrock`
  (Task 4) but keeps those imports off every non-AI command path.
- `cmd/emod-wasm/main.go` and `internal/wasm/pipeline.go` — verify (and, if any import now
  reaches the bedrock subpackage transitively, restructure) so the WASM build excludes the
  Anthropic SDK; rely on build tags or import isolation to keep `internal/wasm` free of the
  provider subpackage.
- `internal/cli/ai_test.go` — new: a missing-config resolution yields the "not configured"
  outcome without error and leaves the other commands' behavior untouched.

**Patterns to Follow:**
- For provider-SDK-free WASM logic, follow the package boundary in
  `internal/wasm/pipeline.go:1-34` and the `cmd/emod-wasm/main.go` entry point — the
  bedrock subpackage from Task 4 must stay out of anything reachable from there.
- For command registration and the `RunX` + `*LintError` unwrap convention, follow
  `internal/cli/app.go:12-256` and `internal/cli/lint.go:15-22`.
- The non-AI commands to leave byte-for-byte unchanged are covered by existing tests
  (`internal/cli/validate_test.go`, `internal/cli/lint_test.go`,
  `internal/cli/diagram_test.go`, `internal/cli/export_test.go`).

**Testable:** Yes — the "AI not configured" outcome and the untouched behavior of existing
commands are observable (Inbound/CLI caller). The WASM provider-free guarantee is a build
guard (no runtime caller): verify it with `task build:wasm`, not a unit test.

**Verification:** `task test:unit` passes; `task build:wasm` succeeds and the resulting
binary links no Anthropic SDK; `go build ./...` succeeds; existing CLI tests pass unchanged.

**Depends on:** Task 4, Task 5

---

### Task 10: Add the `emod ai` end-to-end smoke command

**Language:** Go

**Behavior:** `emod ai` exercises the whole seam: with AI configured it makes one real
round-trip, runs one generate → validate → repair round on a built-in trivial prompt, and
reports success, the model used, token usage, cost, and whether the result passed the oracle
or did not converge; with no AI configuration it prints a clear setup message and a
non-error "not configured" outcome without affecting other commands.

**Acceptance Criteria:**
- [ ] With AI configured, the command makes one real round-trip and reports success, the
  model used, and the token usage and cost.
- [ ] The command runs one generate → validate → repair round on a built-in trivial prompt
  and shows whether the result passed the oracle or did not converge (default attempt bound
  is a small number such as 3).
- [ ] With no AI configuration, the command prints a clear setup message and a non-error
  "not configured" outcome, without affecting other commands.
- [ ] The command is a health/smoke check only and implements no real authoring feature.

**Affected Files/Modules:**
- `internal/cli/ai.go` — extend: the `RunAI` smoke action that resolves config (Task 5),
  builds the adapter (Task 4) when configured, runs the repair loop (Task 8) on a built-in
  trivial prompt with the small default bound, reports cost (Task 7), and prints the
  "not configured" message otherwise.
- `internal/cli/app.go` — register the `ai` command in `NewApp` following the existing
  command shape.
- `internal/cli/ai_test.go` — extend: the not-configured path prints the setup message and
  returns the non-error outcome; the configured run path (driven through the seam with the
  mock `llm.Model` from Task 3, not a live call) reports model, usage, cost, and the
  pass/did-not-converge result, and accumulates usage across attempts.

**Patterns to Follow:**
- For command registration and the `Action` → `RunX` → `*LintError` unwrap convention,
  follow `internal/cli/app.go:12-256` (e.g. the `lsp` command at lines 243–253 for a
  no-argument command, and the `validate` command at lines 17–44 for error mapping).
- Compose the existing pieces: repair loop (`internal/llm/repair`), oracle
  (`internal/oracle`), cost (`internal/llm/cost`), config (`internal/llm/config`), and the
  opt-in gate (Task 9). For the assumed smoke-command shape (one round-trip plus one repair
  round, default bound ~3), read US-010 Context and Open Questions in
  `user-stories/ai/00-llm-foundation.md:154-183`.

**Testable:** Yes — the not-configured CLI outcome and the configured-run reporting are both
observable (Inbound/CLI caller); the configured path is driven through the seam with the
mock so no network is needed in unit tests. The single live round-trip is the manual/health
behavior verified by running the command with real config, not in unit tests.

**Verification:** `task test:unit` passes; `go build ./...` succeeds; manual: `emod ai`
with no config prints the setup message and exits non-error; `emod ai` with real Bedrock
config completes one round and reports model, usage, and cost.

**Depends on:** Task 4, Task 5, Task 6, Task 7, Task 8, Task 9

---

## Summary

- **Total tasks:** 10 (one implementation task per user story, all Go).
- **Ordering rationale:** dependency-first, then risk-first. Task 1 (`llm.Model` port,
  US-001) unblocks everything and ships first. Task 2 (oracle, US-005) is standalone and
  low-risk, pulled early so the repair loop has its correctness check ready. Task 3 (mock,
  US-002) follows the port. Task 4 (Bedrock adapter, US-003) is sequenced right after the
  port to retire the highest external risk (SDK behavior, model IDs, effort mapping) early.
  Tasks 5–7 (config US-004, structured output US-007, cost US-008) build on the adapter.
  Task 8 (repair loop, US-006) needs the port, mock, and oracle. Task 9 (opt-in + WASM,
  US-009) needs the adapter and config. Task 10 (smoke command, US-010) sits last because it
  depends on config, the repair loop, cost, and the opt-in gate. This preserves the story's
  declared edges: US-001 → US-002/US-003; US-003 → US-004/US-007/US-008; US-005 standalone;
  US-006 ← US-001/US-002/US-005; US-009 ← US-003/US-004; US-010 ← US-004/US-006/US-008/US-009.
- **Acceptance-criteria coverage:** every criterion in US-001 through US-010 maps to exactly
  one task (US-001→T1, US-005→T2, US-002→T3, US-003→T4, US-004→T5, US-007→T6, US-008→T7,
  US-006→T8, US-009→T9, US-010→T10). None deferred.
- **Verification by guideline:** Tasks 1–3 and 5–8 deliver behavior testable through their
  package public APIs (Exported API caller) with no network. Task 4 and Task 6 are testable
  only for their pure mapping/validation decisions; their live round-trips are not
  unit-tested (no network in unit tests) and are exercised end-to-end by Task 10. Task 9's
  WASM provider-free guarantee is a build guard with no runtime caller — verified by
  `task build:wasm`, not a unit test — while its opt-in behavior is CLI-testable. Task 10's
  configured path is driven through the mock; its single live round-trip is a manual health
  check.

# US-003: Give the agent the lint rules

## Contents
1. Task 1: List the names of the lint rules, and guard that each reported rule has a description
2. Task 2: Explain a rule with `emod_explain_rule`, and list the rules in `emod://rules`
3. Task 3: Document `emod_explain_rule` and `emod://rules`

---

## Story Reference

`user-stories/mcp-server.md` → **US-003: Give the agent the lint rules**. The design is in
`docs/proposals/ai/01-mcp-server.md`, section "Tool Surface". The proposal sets the direction, and this story
sets the scope.

The story's acceptance criteria, numbered for the Summary:

- **AC1.** `emod_explain_rule` takes a rule name, for example `state-obsession`, and returns `rule` and
  `description`. The description is the same text that `emod lint --explain state-obsession` prints.
- **AC2.** An unknown rule name returns an error result that names the rule, as `emod lint --explain` does.
- **AC3.** The resource `emod://rules` lists each lint rule with its description.
- **AC4.** `emod://rules` lists each rule that the linter can report, which includes the `dcb/*` and `spec/*`
  rules, and no other rule. A new lint rule appears in the list with no change to the server.

## Boundaries

**Out of scope:**

- The grammar resources, the examples and `emod_schema`. US-004 adds them.
- Any prompt. The non-goals of the story set keep the prompts out.
- A change to the text of a rule description.
- A resource for each rule, such as `emod://rules/{name}`. The tool gives one rule, and the resource gives all
  of them.
- Added by this breakdown:
  - **The list holds the two orphan rules.** The validator reports `orphan-command` and `orphan-event`, and
    `emod lint` prints them with their rule names, because `emod lint` runs the whole check. So "each rule
    that the linter can report" means each rule name that `emod lint` can print, and
    `emod lint --explain` already describes all of them.
  - **`emod://rules` is JSON**, with the MIME type `application/json`:
    `{"rules":[{"rule":"...","description":"..."}]}`, in the order of the rule names. Each entry has the
    same two keys as the result of `emod_explain_rule`.
  - **The error text of an unknown rule is the text of the CLI**: `unknown rule "<name>"`.

## Codebase Context

- **The descriptions.** `internal/linter/descriptions.go` holds `ruleDescriptions`, a map from the rule name
  to its text, and `RuleDescription(name)`. `emod lint --explain` (`RunLintExplain` in
  `internal/cli/lint.go:80-91`) prints the text, or returns `unknown rule %q` with exit code 1.
- **The rule names.** The linter (`internal/linter/linter.go`) reports each finding through `info`,
  `warning` or `errorEntry`, which take the rule name as the second argument, always a string literal. The
  validator stamps a rule name only in `orphanDiagnostics` (`internal/validator/validator.go:188`), as the
  third argument, with the literals `orphan-command` and `orphan-event`. No other code sets
  `diagnostic.Entry.RuleName`.
- **No guard keeps the two sides together.** The leaf "all rules have descriptions" in
  `internal/cli/lint_test.go:1921` names 24 rules by hand. A new rule that has no description passes it, and
  `emod lint --explain` then answers `unknown rule` for a name that `emod lint` prints.
- **The server.** `internal/mcp/server.go` registers `emod_check` with `sdk.AddTool`. The learning "An MCP
  tool over the go-sdk typed AddTool is routine, not low certainty" describes the SDK. `sdk.Server.AddResource`
  takes an `*sdk.Resource` and a `ResourceHandler`, and the SDK then advertises the resources capability
  with `listChanged`, also when `ServerOptions.Capabilities` is not nil.
- **The tests.** `internal/mcp/server_test.go` drives the server through an SDK client (`connect`). Two
  leaves assume one tool: "lists emod_check as the one tool" and "describes each input", and one leaf
  asserts the capabilities. `cmd/emod/main_test.go` drives the built binary, and its leaf "the binary lists
  emod_check" assumes one tool.

## Tasks

### Task 1: List the names of the lint rules, and guard that each reported rule has a description

**Behavior:** `linter.RuleNames()` returns the name of each lint rule that has a description, in
alphabetical order. A guard test reads the source of the linter and the validator, collects each rule name
that they report, and requires that the set is the same as `RuleNames()`. So a new rule with no
description fails the unit suite, and a description with no rule fails it too.

**Acceptance Criteria:**
- [x] `RuleNames()` returns 24 names in alphabetical order, which include `orphan-command`, `dcb/orphan-tag-key`
  and `spec/given-outside-boundary`.
- [x] Each name that `RuleNames()` returns has a description from `RuleDescription`.
- [x] The guard collects the rule names from each call of `info`, `warning` and `errorEntry` in
  `internal/linter/linter.go`, and of `orphanDiagnostics` in `internal/validator/validator.go`.
- [x] The guard fails when a rule argument is not a string literal, so that a rule name cannot pass by it.
- [x] The guard fails when it finds no call of one of the four functions, so that a rename does not empty it.

**Patterns to follow:**
- `internal/linter/descriptions.go`
- `internal/linter/linter_test.go` (build tag, package `linter_test`)

**Affected files:**
- `internal/linter/descriptions.go`
- `internal/linter/descriptions_test.go` (new)

**Certainty:** high. The map is there, and the guard reads Go source with `go/parser`. **Blast radius:**
low. Nothing reads `RuleNames()` yet.

**Depends on:** none

### Task 2: Explain a rule with `emod_explain_rule`, and list the rules in `emod://rules`

**Behavior:** The server registers the tool `emod_explain_rule`, which takes `rule` and returns `rule` and
`description`. An unknown name gives the error result `unknown rule "<name>"`. The server registers the
resource `emod://rules`, which returns each name of `RuleNames()` with its description as JSON.

**Acceptance Criteria:**
- [x] `tools/list` lists `emod_check` and `emod_explain_rule`. The input of `emod_explain_rule` is one
  required string, `rule`, and no other input.
- [x] `emod_explain_rule` with `state-obsession` returns `{"rule":"state-obsession","description":...}`
  with the description of the linter, in the structured content and in the text block.
- [x] `emod_explain_rule` with an unknown name returns an error result with the text `unknown rule "<name>"`.
- [x] `resources/list` lists `emod://rules` with the MIME type `application/json` and a description.
- [x] `resources/read` of `emod://rules` returns the 24 rules, each with its description, in the order of
  the names.
- [x] The initialize result offers tools and resources, and no logging.
- [x] Through the built binary, the description of `emod_explain_rule` for `state-obsession` is the text
  that `emod lint --explain state-obsession` prints, with no trailing newline.

**Patterns to follow:**
- `internal/mcp/server.go` (`checkTool`, `sdk.AddTool`)
- `internal/mcp/server_test.go` (`connect`, `textOf`)
- `cmd/emod/main_test.go` (`startServer`)
- task:1

**Affected files:**
- `internal/mcp/server.go`
- `internal/mcp/server_test.go`
- `cmd/emod/main_test.go`

**Certainty:** high. The learning on the typed `AddTool` and the existing `emod_check` are the precedent,
and the resource API is in the SDK. **Blast radius:** low. Both are read-only and take no path.

**Depends on:** 1

### Task 3: Document `emod_explain_rule` and `emod://rules`

**Behavior:** The MCP page of the docs describes the tool and the resource, and the README names them.

**Acceptance Criteria:**
- [ ] `site/content/docs/mcp.md` has a section on `emod_explain_rule` (input, result, error) and on
  `emod://rules` (the JSON shape), and says that an agent reads the description of a rule before it fixes a
  finding.
- [ ] The README section "Check models from an agent (MCP)" names the tool and the resource.

**Patterns to follow:**
- `site/content/docs/mcp.md`

**Affected files:**
- `site/content/docs/mcp.md`
- `README.md`

**Certainty:** high. **Blast radius:** low.

**Depends on:** 2

## Summary

| AC | Task |
| --- | --- |
| AC1 | 2 |
| AC2 | 2 |
| AC3 | 2 |
| AC4 | 1, 2 |

---
title: MCP server
weight: 7
---

`emod mcp` starts a server that speaks the Model Context Protocol (MCP) over stdin and stdout. An MCP host,
such as Claude Code, starts the server and gives its tools and resources to the model that the host runs.
The agent reads the grammar of emod from the server before its first draft, checks the draft with emod
before it writes the file, and reads why a lint rule exists before it fixes a finding.

The server calls no model. The host brings the model, so the server needs no API key, no `EMOD_AI_*`
setting and no network access.

## Register the server with Claude Code

The `emod` binary must be on your `PATH`. Then run:

```sh
claude mcp add emod -- emod mcp
```

`claude mcp list` shows `emod` when Claude Code can start the server.

## The `emod_check` tool

`emod_check` runs the check of `emod validate` on one model: the parser, the validator and the linter. The
call gives the model as text in `content`, or as the path of a file in `path`.

| Input | Required | What it is |
| --- | --- | --- |
| `content` | One of `content` and `path` | The `.emod` text to check |
| `path` | One of `content` and `path` | The path of an `.emod` file under the root, relative to the root or absolute |
| `name` | No | The file name that the records of `content` carry. The default is `path`, else `inline.emod`. |

When a call gives both `content` and `path`, the tool checks `content` and does not read `path`. So an agent
can check the draft of a new file before it writes the file, and the records carry the path of that file.
The records of a `path` carry the path as the call gives it.

The result holds a list of diagnostics under `diagnostics`. Each record is the same record that
`emod validate --format json` prints for the same model:

```json
{
  "diagnostics": [
    {
      "file": "ordering.emod",
      "line": 16,
      "rule": "state-obsession",
      "severity": "warning",
      "message": "event \"OrderUpdated\" uses a generic state-change suffix; prefer a name that describes a specific business fact"
    }
  ]
}
```

- An empty list means that the draft passes the check.
- Errors and warnings are normal results, not failures of the tool. A record has a `rule` key only when a
  rule name goes with the diagnostic, as with each lint finding. `emod_explain_rule` describes the rule.
- The tool returns an error result only when it cannot do the check: for example, when the call gives
  neither `content` nor `path`, or a path that the server cannot read.

The agent drafts a model, calls `emod_check`, fixes the cause of each record, and calls the tool again until
the list is empty. Then the agent writes the file with its own file tool.

## The grammar of emod

The server gives the agent the schema, the examples and the DSL reference of emod, so that the first draft
of the agent uses valid syntax and you paste no reference:

| Resource | MIME type | What it returns |
| --- | --- | --- |
| `emod://schema` | `text/plain` | The CUE schema of emod, the same text that `emod schema` prints |
| `emod://examples/all_patterns` | `text/plain` | A hotel model that shows the four slice patterns: command, view, automation and translation |
| `emod://examples/dcb_model` | `text/plain` | An order model in a `dcb` context, with `tags` on events and `decides_on` on commands |
| `emod://examples/specs_hotel` | `text/plain` | A hotel model with Given-When-Then specs, with example payloads and rejections |
| `emod://docs/dsl-reference` | `text/markdown` | The DSL reference: each construct, its syntax, the slice patterns and the specs |

Each example is the same text as its file in the `examples/` folder of the emod repository, and each one
passes `emod_check` with an empty list. The server does not serve `examples/error_diagnostics_test.emod`,
because that file has errors on purpose.

The text comes from the `emod` binary, not from a file on disk. So the server gives the same text from any
working directory, and the agent learns the grammar of the emod version that it calls.

Some hosts do not read a resource unless you ask for it. So the tool `emod_schema` gives the schema too. It
takes no input, and it returns the text of `emod://schema` as one text block.

## The `emod_slices_list` tool

`emod_slices_list` lists the slices of a model, so that the agent answers a question about the structure of
the model from facts. It takes the same `content`, `path` and `name` as `emod_check`.

The result holds one record for each slice under `slices`, in source order. Each record is the same record
that `emod slices list --format json` prints for the same model:

```json
{
  "slices": [
    {
      "name": "Reserve Room",
      "pattern": "command",
      "context": "Reservations",
      "keyElements": "ReserveRoom, RoomReserved"
    }
  ]
}
```

- `pattern` is `command`, `view`, `automation`, `translation` or `unknown`.
- `keyElements` names the parts that make the slice its pattern: for example the command and its event, the
  view, or the activation and the command of an automation.
- A model with no slice gives an empty list.
- A source that does not parse gives its parse diagnostics under `diagnostics`, as the records of
  `emod_check`, and no `slices`. This is a normal result, not an error result. A lint warning or a
  validation error does not stop the list.

## The `emod_export` tool

`emod_export` gives the model as a document, so that the agent reads the structure of the model and does not
parse the text. It takes the same `content`, `path` and `name` as `emod_check`, and a `format`:

| `format` | What the tool gives |
| --- | --- |
| `json` (the default) | The JSON document of the model, with the diagnostics of the check in it |
| `diagram-json` | Each node and each edge of the diagram, with the diagnostics of the check in it |
| `cue` | The CUE text of the model, for a source with no diagnostic |

- Each format gives the same output as `emod export --format <format>`.
- `json` and `diagram-json` give the document also for a source with errors. The document is the structured
  content of the result, and the text block holds the same JSON.
- `cue` gives the CUE text as one text block. A source with any diagnostic, also a warning, gives the
  diagnostics under `diagnostics`, as the records of `emod_check`, and no CUE.
- An unknown format gives an error result that lists the three formats.
- `emod://schema` gives the CUE schema of the model: the document under `model` in `json`, and the text of
  `cue`. It does not describe the `diagram-json` document.

## The `emod_glossary` tool

`emod_glossary` gives the terms that the model defines, so that the agent uses the words of the model when
it writes a document or answers a question. It takes the same `content`, `path` and `name` as `emod_check`,
and a `format`:

| `format` | What the tool gives |
| --- | --- |
| `markdown` (the default) | The glossary as one text block |
| `json` | The same terms as a document: the structured content, with the same JSON in the text block |

- Each format gives the same text as `emod glossary --format <format>`.
- A source that does not parse gives its parse diagnostics under `diagnostics`, and no glossary.
- An unknown format gives an error result that lists the two formats.

Neither `emod_export` nor `emod_glossary` writes a file.

## The `emod_diagram` tool

`emod_diagram` draws a model, so that you see the shape of a change in the chat before you open the viewer.
It takes the same `content`, `path` and `name` as `emod_check`, and these inputs:

| Input | Values | Default |
| --- | --- | --- |
| `format` | `svg`, `mermaid`, `ascii`, `event-flow`, `event-flow-mermaid`, `event-flow-mermaid-no-icons` | `svg` |
| `style` | `auto`, `projected`, `dcb` | `auto` |
| `specs` | `true` draws the Given-When-Then cards of the specs | `false` |

- Each format gives the same text as `emod diagram --format <format>`, with the same `--style` and `--specs`.
- The result holds the diagram text under `diagram`, and the first text block holds the same text. A host
  shows `mermaid` and `ascii` in the chat.
- For `svg` and `event-flow`, the tool gives the SVG text and writes no file. The host saves it with its own
  file tool when you want a file.
- A source with only warnings gives the diagram and the warnings under `diagnostics`. A second text block
  holds the warnings, one line each, as `emod diagram` prints them.
- A source with an error gives its diagnostics under `diagnostics`, as the records of `emod_check`, and no
  diagram.
- Only `svg` draws the spec cards. `specs: true` with another format gives an error result that says so.
- The tool starts no server. `emod diagram --serve` has no match in the tool.

## The `emod_fmt` tool

`emod_fmt` gives a model in the canonical layout of `emod fmt`, so that a file that the agent writes passes
`emod fmt --check` in CI. It takes the same `content`, `path` and `name` as `emod_check`.

The result holds the text under `formatted`, and the answer of the check under `changed`:

```json
{
  "formatted": "emod = 1\n\nmodel \"Ordering\" {\n}\n",
  "changed": true
}
```

- `formatted` is the same text that `emod fmt` writes for the source. The text block holds the same JSON.
- `changed` is true when `formatted` differs from the source, and false when the source is already in the
  canonical layout. It gives the same answer as the exit code of `emod fmt --check`.
- The tool writes the file only in the [write mode](#the-write-mode). Else the agent writes `formatted` with
  the file tool of the host.
- A source that does not parse gives its parse diagnostics under `diagnostics`, as the records of
  `emod_check`, and no `formatted`. This is a normal result, not an error result. A lint warning or a
  validation error does not stop the tool.

## The `emod_slices_arrange` tool

`emod_slices_arrange` puts the slices of each context and each aggregate in an order in which each reference
points forward, as `emod slices arrange` does. So a file that the agent writes passes
`emod slices arrange --check` in CI. It takes the same `content`, `path` and `name` as `emod_check`.

The result holds the arranged text and the report of the arrange:

```json
{
  "arranged": "emod = 1\n\n# Hotel Reservation System — exercises all four slice patterns\nmodel \"Hotel Reservation\" {\n...",
  "changed": true,
  "moved": 1,
  "backwardBefore": 3,
  "backwardAfter": 2,
  "backward": [
    {
      "kind": "reads",
      "label": "AvailableRoomsView -> trigger Reservation Form",
      "from": "View Available Rooms",
      "to": "Reserve a Room"
    },
    {
      "kind": "reads",
      "label": "UnconfirmedReservationsView -> UnconfirmedReservationExpirer",
      "from": "View Unconfirmed Reservations",
      "to": "Expire Unconfirmed Reservations"
    }
  ]
}
```

- Only view slices move. Each other slice stays where its author put it.
- `arranged` is the same text that `emod slices arrange` writes. When a slice moves, it is the whole model in
  the layout of `emod fmt`. When no slice moves, it is the source as it is.
- `changed` is true when a slice moves. It gives the same answer as the exit code of
  `emod slices arrange --check`.
- `moved`, `backwardBefore` and `backwardAfter` are the counts of the report that `emod slices arrange`
  prints.
- `backward` lists each reference that points backward after the arrange, with its `kind` (`subscribes`,
  `reads`, `on` or `flow`), its `label`, and the names of its two slices in `from` and `to`. The list is
  empty when no reference points backward. Because only view slices move, the arrange cannot fix such a
  reference, for example a reference between two slices that are not views. The agent does not try a new
  order for it.
- The tool writes the file only in the [write mode](#the-write-mode). Else the agent writes `arranged` with
  the file tool of the host.
- A source that does not parse gives its parse diagnostics under `diagnostics`, as the records of
  `emod_check`, and no report. A lint warning or a validation error does not stop the tool.

## The write mode

By default, the server writes no file, so you can register it for all projects. When you trust the agent in
one project, start the server with `--allow-write`:

```sh
claude mcp add emod -- emod mcp --allow-write
```

Then `emod_fmt` and `emod_slices_arrange` write their result to the file in `path` for a call that gives
`write: true`, so the agent fixes the layout of a file in one call:

```json
{ "path": "ordering.emod", "write": true }
```

- You decide about writes when you start the server, not the model. Without `--allow-write`, a call with
  `write: true` gives an error result that names the flag, and the file does not change.
- A call with `write: true` must give `path` and no `content`. The tool writes only a new layout of the file
  on disk, never text from the call. The host has its own file tools for other writes.
- The tool writes the file only when `changed` is true. When the file is already in the canonical layout, or
  when no slice moves, the tool writes nothing, and the modification time of the file does not change.
- The result is the same as for a call without `write`.
- The tool never writes a source that does not parse. It gives the parse diagnostics.
- The write follows the rules of [the root](#the-root). The tool refuses a path outside the root and writes
  nothing.
- The tool writes only a file that exists, in place, so the file keeps its mode.
- Just before it writes, the tool checks that the file still holds the text that it read. If a person saved
  the file after the read, for example in the desktop app, the tool gives an error result, and the file keeps
  the text of the person. The check does not see a save in the short time between the check and the write.
- One server writes one file at a time, and a read never sees half of a write. When two calls write one file,
  a call that read the file before the other call wrote it gives the error result for a file that changed.
- No other tool writes to disk, with or without `--allow-write`.

## The `emod_explain_rule` tool

`emod_explain_rule` tells why a lint rule exists and how to fix a finding of the rule. The call gives the
name from the `rule` key of a record in `rule`:

| Input | Required | What it is |
| --- | --- | --- |
| `rule` | Yes | The name of a lint rule, for example `state-obsession` or `dcb/untagged-event` |

The result holds the name under `rule` and its text under `description`. The text is the same text that
`emod lint --explain <rule>` prints:

```json
{
  "rule": "state-obsession",
  "description": "Events named with generic state-change suffixes like \"Updated\", \"Changed\", or \"Modified\" hide the business meaning of what happened. Prefer names that describe a specific business fact (e.g. \"OrderShipped\" instead of \"OrderUpdated\")."
}
```

A name that is not a lint rule gives an error result, for example `unknown rule "dcb/nonexistent"`.

So for a record with the rule `state-obsession` on the event `OrderUpdated`, the agent reads that the name
must tell the business fact, and it names the event `OrderShipped` in its next draft.

## The `emod://rules` resource

The resource `emod://rules` lists each lint rule that emod can report, with its description. This
includes the `dcb/*` and `spec/*` rules, and `orphan-command` and `orphan-event`, which the validator
reports. The agent reads the list before its first draft, to keep the draft free of the findings.

The resource is JSON, with the MIME type `application/json`. Each entry has the same keys as the result of
`emod_explain_rule`, and the entries are in the alphabetical order of the names:

```json
{
  "rules": [
    {
      "rule": "aggregate-in-dcb-mode",
      "description": "Aggregate blocks contain slices and represent the traditional aggregate modeling style. ..."
    }
  ]
}
```

Some hosts do not read a resource unless the person asks for it. The agent can then call
`emod_explain_rule` for each rule in its records.

## The root

The server reads files only under its root. The root is the working directory of `emod mcp`, and
`emod mcp --root <dir>` sets another root:

```sh
claude mcp add emod -- emod mcp --root /path/to/project
```

- The tool refuses a path outside the root with an error result that names the root. This includes a path
  with `..` that leaves the root, an absolute path in another directory, and a symbolic link to a file outside
  the root.
- Under the root, the server follows only relative symbolic links. It refuses an absolute symbolic link
  under the root as outside the root, also when the link points to a file under the root.
- The server reads only a regular file of at most 1 MiB. It refuses a directory, a named pipe and a larger
  file.
- A path under the root that does not exist, or that the server cannot read, gives an error result that
  names the path and the cause.
- The server writes to a file only in the [write mode](#the-write-mode), and only to a regular file under the
  root that exists.

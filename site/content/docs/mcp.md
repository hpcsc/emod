---
title: MCP server
weight: 7
---

`emod mcp` starts a server that speaks the Model Context Protocol (MCP) over stdin and stdout. An MCP host,
such as Claude Code, starts the server and gives its tools to the model that the host runs. The agent then
checks a draft model with emod before it writes the file.

The server calls no model. The host brings the model, so the server needs no API key, no `EMOD_AI_*`
setting and no network access.

## Register the server with Claude Code

The `emod` binary must be on your `PATH`. Then run:

```sh
claude mcp add emod -- emod mcp
```

`claude mcp list` shows `emod` when Claude Code can start the server.

## The `emod_check` tool

`emod_check` runs the check of `emod validate` on `.emod` text: the parser, the validator and the linter.
It reads no file.

| Input | Required | What it is |
| --- | --- | --- |
| `content` | Yes | The `.emod` text to check |
| `name` | No | The file name that each record carries. The default is `inline.emod`. |

The result holds a list of diagnostics under `diagnostics`. Each record is the same record that
`emod validate --format json` prints for the same text:

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
  rule name goes with the diagnostic, as with each lint finding. `emod lint --explain <rule>` describes a
  lint rule.
- The tool returns an error result only when it cannot do the check, for example when the call gives no
  `content`.

The agent drafts a model, calls `emod_check`, fixes the cause of each record, and calls the tool again until
the list is empty. Then the agent writes the file with its own file tool.

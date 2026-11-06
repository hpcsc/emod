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
  rule name goes with the diagnostic, as with each lint finding. `emod lint --explain <rule>` describes a
  lint rule.
- The tool returns an error result only when it cannot do the check: for example, when the call gives
  neither `content` nor `path`, or a path that the server cannot read.

The agent drafts a model, calls `emod_check`, fixes the cause of each record, and calls the tool again until
the list is empty. Then the agent writes the file with its own file tool.

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
- The server never writes to a file.

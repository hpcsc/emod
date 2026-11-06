---
title: AI
weight: 6
---

emod can call a language model. AI is opt-in: with no AI setting, emod makes no model call, and every
command works as it does with no AI.

The check of emod controls each answer of the model. The parser, the validator, the linter and the CUE
schema check the source that the model writes. When they report a diagnostic, emod sends the diagnostics
back to the model and asks again, up to a fixed number of attempts.

Two commands use AI now:

| Command | What it does |
| --- | --- |
| `emod ai status` | Shows the AI settings and the source of the API key, but not the key. It calls no model. |
| `emod ai` | Checks the setup with one round of model calls: the model writes a small model, and the check of emod repairs it. |

## Choose a provider

| Provider | Servers | API key | Model |
| --- | --- | --- | --- |
| `anthropic`, the default | Claude through the Anthropic API, or a proxy that speaks the Messages API | Required | `claude-opus-4-8` by default |
| `openai` | OpenAI, Azure OpenAI, Gemini, OpenRouter, LiteLLM, Ollama, LM Studio, vLLM | Optional | Required |

### Claude through the Anthropic API

```sh
export EMOD_AI_API_KEY=<your Anthropic API key>
emod ai
```

If `ANTHROPIC_API_KEY` already holds your key, set the provider instead:

```sh
export EMOD_AI_PROVIDER=anthropic
emod ai
```

### OpenAI

```sh
export EMOD_AI_PROVIDER=openai
export EMOD_AI_MODEL=<a model of your OpenAI account>
export EMOD_AI_API_KEY=<your OpenAI API key>
emod ai
```

If `OPENAI_API_KEY` already holds your key, you do not need `EMOD_AI_API_KEY`.

### A local server, such as Ollama

```sh
export EMOD_AI_PROVIDER=openai
export EMOD_AI_MODEL=llama3.1
export EMOD_AI_ENDPOINT=http://localhost:11434/v1
emod ai
```

A local server needs no API key, and the model source does not leave your machine.

### AWS Bedrock, Google Vertex AI and other platforms

emod has no provider for each cloud platform. Run a gateway, such as LiteLLM, in front of the platform:

1. Set `EMOD_AI_ENDPOINT` to the URL of the gateway. See the table below for the path.
2. Set `EMOD_AI_API_KEY` to the key of the gateway.
3. Set `EMOD_AI_MODEL` to a model name from the configuration of the gateway.

Use the provider that matches the API of the gateway:

| API of the gateway | Provider | `EMOD_AI_ENDPOINT` | emod adds |
| --- | --- | --- | --- |
| The Messages API | `anthropic` | `http://localhost:4000`, with no `/v1` | `/v1/messages` |
| The OpenAI-compatible API | `openai` | `http://localhost:4000/v1` | `/chat/completions` |

## Settings

| Key | Flag | Default | Meaning |
| --- | --- | --- | --- |
| `EMOD_AI_PROVIDER` | `--ai-provider` | `anthropic` | `anthropic` or `openai` |
| `EMOD_AI_API_KEY` | none | none | The API key. See [The API key](#the-api-key). |
| `EMOD_AI_MODEL` | `--ai-model` | `claude-opus-4-8` for `anthropic`. No default for `openai`. | The main model |
| `EMOD_AI_MODEL_CHEAP` | `--ai-model-cheap` | `claude-haiku-4-5` for `anthropic`. The main model for `openai`. | The model for small passes |
| `EMOD_AI_EFFORT` | `--ai-effort` | `medium` | `low`, `medium`, `high` or `xhigh` |
| `EMOD_AI_ENDPOINT` | `--ai-endpoint` | The Anthropic API, or `https://api.openai.com/v1` for `openai` | The base URL of the server or the gateway. For `anthropic`, give it with no `/v1`. For `openai`, give it with `/v1`. |

emod takes each setting from the first source that sets it: the flag, then the environment variable, then
the config file, then the default.

AI is on when one `EMOD_AI_*` key is set. `ANTHROPIC_API_KEY` or `OPENAI_API_KEY` alone does not turn AI
on, because you can hold it for another tool.

### The config file

The file `~/.config/emod/config` holds one `KEY=VALUE` line for each setting. A line that starts with `#`
is a comment.

```sh
# ~/.config/emod/config
EMOD_AI_PROVIDER=openai
EMOD_AI_MODEL=llama3.1
EMOD_AI_ENDPOINT=http://localhost:11434/v1
EMOD_AI_EFFORT=high
```

emod removes the spaces around the key and the value, and a pair of quotes of the same kind, `"` or `'`,
around the value. A `#` after a value is part of the value, not a comment.

The file must not hold the API key. If it holds `EMOD_AI_API_KEY`, emod stops and tells you to remove
the line.

### The API key

emod takes the API key only from the environment:

| Provider | Order of the sources |
| --- | --- |
| `anthropic` | `EMOD_AI_API_KEY`, then `ANTHROPIC_API_KEY` |
| `openai` | `EMOD_AI_API_KEY`, then `OPENAI_API_KEY` |

- The key has no flag, because a flag shows in the process list and in the shell history.
- emod sends `EMOD_AI_API_KEY` to the provider that is set. It does not send `ANTHROPIC_API_KEY` to an
  `openai` server, or `OPENAI_API_KEY` to the Anthropic API.
- emod reads no other variable of the Anthropic SDK. `ANTHROPIC_AUTH_TOKEN`, `ANTHROPIC_BASE_URL` and the
  profiles of `ant auth login` have no effect on emod.

## Check the setup

`emod ai status` shows the settings. It does not show the key, and it calls no model:

```text
AI is configured.
  Provider:    anthropic
  Model:       claude-opus-4-8
  Cheap model: claude-haiku-4-5
  Effort:      medium
  Endpoint:    the Anthropic API
  API key:     from EMOD_AI_API_KEY
```

`emod ai` shows the same settings, then calls the model. Each attempt can take many seconds, so emod
prints a line when each attempt ends. For example:

```text
emod asks the model for an emod source. Each attempt can take many seconds.
Attempt 1 of 3: the source has 2 diagnostics. emod asks the model again. Tokens: 1210 input, 380 output.
Attempt 2 of 3: the source passes the oracle. Tokens: 1650 input, 350 output.
The source passed the oracle on attempt 2 of 3.
Tokens: 2860 input, 730 output
```

| Exit code | When |
| --- | --- |
| 0 | An answer passes the check, or AI is not configured. With no configuration, `emod ai` prints the setup steps. |
| 1 | No answer passes the check after 3 attempts, a setting is not valid, or a model call fails |

## Data that leaves your machine

`emod ai` sends a fixed prompt: an instruction, a small example model, and a request for a model of a
library. It sends none of your files.

With a remote provider, the prompt goes to the host of the endpoint. `emod ai status` names that host.
With a local server, no data leaves your machine.

## Tested and compatible models

| Tier | Models | What emod promises |
| --- | --- | --- |
| Tested | Claude through the `anthropic` provider | emod tests its AI features on these models and sets their defaults. |
| Compatible | Any model behind the `openai` provider that supports JSON Schema output | The check of emod applies to each answer. emod does not measure the quality of the judgement. |

With the `openai` provider, `emod ai` prints a notice about the tier on each run.

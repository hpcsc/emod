# emod AI Proposals

Ideas for using LLMs in emod. The premise across all of them: emod is an unusually
good *target* for an LLM because it has a deterministic correctness oracle
(`emod validate`, `emod lint`, the CUE schema), so generated or edited models can be
machine-checked and the errors fed back until clean.

Read [`00-llm-foundation.md`](./00-llm-foundation.md) first — it defines the shared
`llm.Model` port, the Bedrock-backed Claude adapter, and the generate → validate →
lint → repair loop that the feature proposals build on.

## Proposals

| # | Proposal | One-line |
|---|----------|----------|
| 00 | [LLM foundation](./00-llm-foundation.md) | The shared `llm.Model` port, provider decision, and repair loop |
| 01 | [NL → model generation](./01-nl-to-model-generation.md) | Plain-English description → validated `.emod`, via the self-correcting loop |
| 02 | [Model import / reverse-engineering](./02-model-import-reverse-engineering.md) | Existing code or structured artifacts → `.emod` |
| 03 | [Semantic model reviewer](./03-semantic-model-reviewer.md) | LLM modeling-smell review beyond the regex linter |
| 04 | [Lint quick-fixes (LSP)](./04-lint-quickfixes-lsp.md) | AI code actions that fix the linter's existing findings |
| 05 | [DCB modeling assistant](./05-dcb-modeling-assistant.md) | Suggest tags / `decides_on`, fix DCB anti-patterns |
| 06 | [MCP server](./06-mcp-server.md) | Expose parse/validate/lint/export as MCP tools — no provider code |
| 07 | [Talk to your model (Q&A)](./07-talk-to-your-model-qa.md) | Grounded question answering over a model |
| 08 | [Docs / onboarding generation](./08-docs-generation.md) | Narrative docs from a `.emod` file |
| 09 | [BDD / test generation](./09-bdd-test-generation.md) | Given/When/Then scenarios and sample payloads from flows |
| 10 | [Conversational viewer editing](./10-conversational-viewer-editing.md) | Chat-driven live model editing in the WASM viewer |

## Suggested sequence

1. **06 (MCP server)** — smallest, reuses everything already built, immediately lets
   existing agents generate and self-check models. No LLM dependency in emod itself.
2. **01 (NL → model)** — the differentiated capability emod is unusually suited for;
   establishes the repair loop the others reuse.
3. Then layer review/assist features (03, 04, 05) and the consumption features
   (07, 08, 09), with 02 and 10 as larger follow-ons.

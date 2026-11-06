# User story progress

Which story in each in-progress story file is delivered. Files under
`user-stories/completed/` are finished in full and are not tracked here.

**5 of 26 delivered.**

## [emod-desktop.md](./emod-desktop.md) — 0/16

- [ ] US-001: Render a model in a native desktop window
- [ ] US-002: Open a model with a native file dialog
- [ ] US-003: Save a model back to the file it came from
- [ ] US-004: Know when there are unsaved changes
- [ ] US-005: Open a model by dropping it on the window
- [ ] US-006: Reopen a recently opened model
- [ ] US-007: Install and run a packaged app on macOS
- [ ] US-008: Open a model by double-clicking it in the file manager
- [ ] US-009: Edit source in the app and see diagnostics as you type
- [ ] US-010: Jump between a diagnostic and the source that caused it
- [ ] US-011: Keep the source panel and the diagram in agreement
- [ ] US-012: Read source with syntax highlighting
- [ ] US-013: Notice when the open file changes on disk
- [ ] US-014: Run the desktop app on Linux
- [ ] US-015: Download a desktop build without building it
- [ ] US-016: Run the desktop app on Windows

## [00-llm-foundation.md](./ai/00-llm-foundation.md) — 5/10

Breakdown in flight at `tasks/00-llm-foundation.md` (tasks 1-5 of 10 checked off). `internal/llm`
carries the port, its mock and broken doubles, the Bedrock adapter and the configuration block;
`internal/oracle` is the deterministic correctness check the repair loop will call.

- [x] US-001: Define the `llm.Model` port
- [x] US-002: Mock model for network-free tests
- [x] US-003: Concrete Bedrock adapter
- [x] US-004: Single AI configuration block
- [x] US-005: Single deterministic correctness oracle
- [ ] US-006: Generate → validate → lint → repair loop
- [ ] US-007: Schema-conformant structured output
- [ ] US-008: Cost and token usage reporting
- [ ] US-009: AI stays opt-in; existing paths and WASM stay provider-free
- [ ] US-010: End-to-end smoke command proving the seam

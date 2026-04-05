# Expand Tool

## Overview

The `expand` tool enables progressive discovery of detailed information about **tools**. It retrieves `AdvanceDescription`, optional `Troubleshooting`, and per-action `DetailsAbout` text on demand so system prompts can stay concise. Tools are provided to the LLM through native tool definitions; `expand` is for extra detail when the short prompt and basic tool metadata are not enough.

## Tool contract

- **subject_type** must be `"tool"`.
- **subject_name** is the tool’s `GetName()` (case-sensitive).
- Optional **details_about**, **troubleshoot** behave as implemented in `src/tools/expand/tool.go`.

## Agent context

The executor supplies `tools` on the agent context. There is no sub-agent list.

## Discoverable

Tools that support progressive discovery implement `agentforge.Discoverable` (`BasicDescription`, `AdvanceDescription`, `Troubleshooting`, `DetailsAbout`).

## Tests

```bash
go test ./src/tools/expand -v
```

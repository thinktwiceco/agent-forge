# Overview & Purpose

**Agent Forge** is a Go framework for building intelligent agents with LLM integration, tool execution, and multi-agent collaboration.

## Key Technologies

- **Language:** Go 1.21+
- **LLM Providers:** OpenAI, TogetherAI, DeepSeek, OpenAI-compatible APIs
- **Architecture:** Interface-based design, separation of concerns, modular packages

## Core Principles

1. **Reusability** - Extracted packages (`history`, `telemetry`) usable outside agent context
2. **Extensibility** - Strategy patterns (truncation), plugin system, interface segregation
3. **Maintainability** - Clear package boundaries, single responsibilities
4. **Separation of Concerns** - Domain-based organization (execution, context, prompts, handlers)
5. **Interface Segregation** - Split large interfaces (Plugin → HookProvider, ToolProvider, PromptProvider)

## Architecture Highlights

- Extracted history management to `src/history/`
- Telemetry package `src/telemetry/` for observability
- Agent responsibilities split into sub-packages (execution, context, prompts, handlers)
- Interface-based design with mocks for testing
- Strategy patterns for extensibility (truncation, plugins)

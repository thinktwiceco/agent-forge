# File Structure Reorganization

This document describes the reorganized file structure of the agent-forge codebase.

## Overview

The codebase has been reorganized to follow **separation of concerns** and **modular design principles**:
- ✅ Domain-based organization (vs. prefix-based naming)
- ✅ Clear package boundaries with interfaces
- ✅ Extracted packages for reusability (`history`, `telemetry`)
- ✅ Reduced God Object pattern (Agent responsibilities split)
- ✅ Strategy patterns for extensibility (truncation, plugins)

## Current Structure

```
src/
├── agents/                       # Agent implementation
│   ├── agent.go                 # Main Agent orchestrator (reduced responsibilities)
│   ├── agentConfig.go           # Agent configuration
│   ├── agentHistory.go          # Agent-level history methods (delegates to history pkg)
│   ├── agentHooks.go            # Hook registration and management
│   ├── agentSubagent.go         # Subagent handling
│   ├── agentTruncation.go       # History truncation for context limits
│   ├── agentChat.go             # Chat execution wrappers (delegates to executor)
│   ├── builder.go               # Fluent Builder API
│   ├── constants.go             # Constants
│   ├── interfaces.go           # Internal interfaces
│   ├── systemAgentConstructors.go # System agent factory functions
│   ├── systemHandlers.go        # System handler registration
│   ├── testing_helpers.go       # Test utilities
│   ├── *_test.go                # Test files
│   │
│   ├── context/                 # Context management & truncation
│   │   ├── manager.go           # Context manager (implements ContextManager)
│   │   ├── strategies.go       # Truncation strategies (SlidingWindow, Noop)
│   │   ├── truncation.go       # Truncation interface
│   │   └── *_test.go           # Tests
│   │
│   ├── execution/               # Execution engine
│   │   ├── executor.go         # Chat loop & tool execution (implements ExecutionEngine)
│   │   └── hooks.go            # Hook runner for executor
│   │
│   ├── handlers/                # System event handlers
│   │   ├── system.go           # System handlers (encapsulated, testable)
│   │   └── system_test.go      # Handler tests
│   │
│   ├── mocks/                   # Mock implementations for testing
│   │   ├── execution_engine.go # Mock ExecutionEngine
│   │   ├── prompt_builder.go   # Mock PromptBuilder
│   │   ├── context_manager.go  # Mock ContextManager
│   │   └── history_manager.go  # Mock HistoryManager
│   │
│   ├── prompts/                 # Prompt building
│   │   ├── builder.go          # Prompt builder (implements PromptBuilder)
│   │   └── config.go           # Prompt configuration
│   │
│   └── system/                  # System agent templates
│       ├── constants.go        # System agent constants
│       ├── systemAgentTemplate.go # Template struct & factory
│       ├── saCodingAgent.go     # Coding agent template
│       ├── saGitAgent.go        # Git agent template
│       ├── saOsAgent.go        # OS agent template
│       ├── saReasoning.go      # Reasoning agent template
│       └── saWebAgent.go       # Web agent template
│
├── core/                        # Core abstractions
│   ├── interfaces.go           # SubAgent, Plugin (split interfaces)
│   ├── tool.go                 # Tool implementation
│   ├── agentContext.go         # AgentContext struct
│   ├── response.go             # Response channels
│   ├── embeddings.go           # Embedding interface
│   └── vectordb.go             # Vector DB interface
│
├── history/                     # Conversation history (extracted)
│   ├── manager.go              # History manager (implements history.Manager)
│   └── manager_test.go         # Tests
│
├── telemetry/                   # Observability
│   ├── tracer.go               # Tracer interface & event types
│   ├── noop.go                 # NoopTracer (default)
│   ├── logger.go               # LogTracer (log-based)
│   ├── metrics.go              # MetricsTracer (OpenTelemetry)
│   └── tracer_test.go          # Tests
│
├── llms/                        # LLM engines
├── tools/                       # Built-in tools
├── persistence/                 # Persistence backends
├── integrations/                # External integrations
└── plugins/                     # Plugin implementations
```

## Key Changes

### 1. System Agents Package

System agent implementations have been moved to `src/agents/system/`:

- **Templates**: Each system agent (`saCodingAgent.go`, `saGitAgent.go`, `saOsAgent.go`, `saReasoning.go`, `saWebAgent.go`) provides a template creation function (e.g., `CreateCodingAgentTemplate()`)
- **Constants**: Agent names, traces, and tones are defined in `system/constants.go`
- **Constructors**: Actual agent constructors (e.g., `CodingAgent()`, `GitAgent()`) remain in the agents package in `systemAgentConstructors.go`

This separation breaks circular imports while maintaining clean domain boundaries.

### 2. Context Management

Context-related code moved to `src/agents/context/`:

- **Manager**: Context lifecycle management
- **Strategies**: Truncation strategy implementations
- **Truncation**: Truncation logic and utilities

### 3. Execution Engine

Execution logic moved to `src/agents/execution/`:

- **Executor**: Main chat execution engine
- **Hooks**: Hook runner interface for executor

### 4. Handlers

Handler implementations moved to `src/agents/handlers/`:

- **System Handlers**: System-level event handlers

### 5. Prompts

Prompt building logic moved to `src/agents/prompts/`:

- **Builder**: Prompt construction logic
- **Config**: Prompt configuration

### 6. Mocks

Mock implementations for testing moved to `src/agents/mocks/`:

- All interface mocks for testing purposes

## Import Patterns

### Agents Package Imports System Package

```go
import "github.com/thinktwiceco/agent-forge/src/agents/system"

// Use system agent templates
template := system.CreateCodingAgentTemplate()
```

### System Package is Self-Contained

The system package does NOT import the agents package to avoid circular dependencies. Constants are duplicated where necessary.

### External Packages Import Agents Package

```go
import "github.com/thinktwiceco/agent-forge/src/agents"

// Use agent constructors from agents package
agent := agents.CodingAgent(llmEngine, rootDir)
```

## Key Architectural Changes

### 1. Extracted Packages

**History Package** (`src/history/`)
- Standalone conversation history management
- Reusable outside agent context
- Implements `history.Manager` interface

**Telemetry Package** (`src/telemetry/`)
- Structured observability for agent behavior
- Pluggable tracers (Noop, Log, Metrics)
- Tracks tool execution, tokens, truncation

### 2. Separated Concerns (Agent Package)

**Context Management** (`src/agents/context/`)
- Context lifecycle & building
- Truncation strategies (strategy pattern)
- Token counting integration

**Execution Engine** (`src/agents/execution/`)
- Chat loop with tool execution
- Streaming response handling
- Iteration management

**Prompt Building** (`src/agents/prompts/`)
- System prompt construction
- Tool/sub-agent descriptions
- Tone application

**Handlers** (`src/agents/handlers/`)
- System event handlers
- Encapsulated & testable
- Minimal `AgentOperations` interface

### 3. Interface-Based Design

All major components have interfaces:
- `HistoryManager` - History operations
- `ExecutionEngine` - Chat/tool execution
- `PromptBuilder` - Prompt construction
- `ContextManager` - Context management
- `HookRegistry` - Event system

Mock implementations available in `src/agents/mocks/`

### 4. Split Interfaces (Core Package)

**SubAgent Composition:**
```go
type Identifier interface { Name() string }
type Executable interface { ChatStream(...) *ResponseCh }
type SubAgent interface {
    Identifier
    agentforge.Discoverable
    Executable
}
```

**Plugin Segregation:**
```go
type Plugin interface { Name() string }
type HookProvider interface { Plugin; Hooks() map[Event]AgentHookFn }
type ToolProvider interface { Plugin; Tools() []llms.Tool }
type PromptProvider interface { Plugin; SystemPrompt() string }
```

## Benefits

1. **Better Organization**: Domain-based packages with clear responsibilities
2. **Improved Discoverability**: Intuitive package names and structure
3. **Clear Boundaries**: Interfaces define contracts between modules
4. **No Circular Imports**: Clean dependency graph
5. **Better Testing**: Mock implementations for all interfaces
6. **Reusability**: Extracted packages (`history`, `telemetry`) usable elsewhere
7. **Extensibility**: Strategy patterns for truncation, plugins
8. **Maintainability**: Smaller, focused files with single responsibilities

## Agent Responsibilities Comparison

**Before (God Object):**
- Configuration management
- LLM interaction
- Tool execution
- Chat loop management
- History management
- Context building
- Prompt generation
- Token counting/truncation
- Hook system

**After (Orchestrator):**
- Configuration
- LLM interaction
- Tool registry
- Sub-agent registry
- Component orchestration (delegates to interfaces)

**Extracted to:**
- `execution.Executor` - Chat loop, tool execution
- `prompts.Builder` - Prompt generation
- `context.Manager` - Context building, truncation
- `history.Manager` - History management
- `handlers.SystemHandlers` - Event handlers

## Migration Notes

- ✅ All public APIs remain backward compatible
- ✅ Agent constructor functions unchanged
- ✅ Tests updated and passing
- ✅ New Builder API available (recommended)
- ⚠️ Some internal APIs changed (use interfaces)

**History vs Persistence:** `src/history/` manages conversation history; `src/persistence/` provides storage backends (e.g. JSON). The history package uses persistence implementations for save/load.

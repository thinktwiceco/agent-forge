# Architecture Changelog

This document outlines the major architectural changes from the original version (commit c20d426) to the current implementation.

## Overview

The library has undergone significant refactoring to improve modularity, performance, and extensibility. The changes focus on better separation of concerns, improved type safety, and enhanced lifecycle management.

## Major Architectural Changes

### 1. Tool System Refactoring

**Before:** Tools were created by extending `tools.BaseTool` struct with inheritance pattern.

**After:** Tools are now created using the `core.Tool` struct directly via struct literals.

**Impact:**
- Removed `tools.BaseTool` - replaced with `core.Tool` struct
- Tools are now created with `&core.Tool{...}` instead of extending BaseTool
- Unified tool implementation that implements both `llms.Tool` and `agentforge.Discoverable` interfaces
- Tool responses moved to `core/tool_response.go` with helper functions

**Migration:**
```go
// Old way (no longer works)
type MyTool struct {
    *tools.BaseTool
}

// New way
myTool := &core.Tool{
    Name: "my_tool",
    Description: "Tool description",
    // ... other fields
    Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
        return core.NewSuccessResponse("result")
    },
}
```

### 2. Agent Context System

**Before:** Agent context was built dynamically on every tool call, potentially rebuilding the same information repeatedly.

**After:** Introduced `core.AgentContext` struct that builds static context once at initialization.

**Impact:**
- Context built once at agent initialization instead of per-tool-call
- Improved performance by avoiding redundant context building
- `AgentContext` includes: AgentName, Trace, Model, Tools, SubAgents
- `BuildContext()` method merges static context with session-specific `responseCh`

**Benefits:**
- Reduced memory allocations
- Faster tool execution
- Consistent context across tool calls

### 3. Sub-Agent Interface

**Before:** Sub-agents were stored as `[]*Agent`, requiring direct agent struct access.

**After:** Introduced `core.SubAgent` interface for type-safe delegation.

**Impact:**
- Changed from `[]*Agent` to `[]*core.SubAgent` for better type safety
- Agents now implement `SubAgent` interface (Name, BasicDescription, AdvanceDescription, Troubleshooting, ChatStream)
- Better encapsulation and interface-based design

**Migration:**
```go
// Old way
subAgents := []*Agent{agent1, agent2}

// New way
subAgent1 := agent1.AgentAsSubAgent()
subAgents := []*core.SubAgent{subAgent1, subAgent2}
```

### 4. Hook System

**Before:** No lifecycle hooks - agent behavior was hardcoded.

**After:** Comprehensive lifecycle hook system (`AgentHooks`) for extensibility.

**Impact:**
- Added `AgentHooks` struct with multiple hook types
- Events: `contextBuild`, `beforeToolExecution`, `toolExecution`, `newUserMessage`, `addSystemAgent`, `addedSystemAgent`
- System callbacks registered automatically via `registerSystemCallbacks()`
- Hook errors logged but don't abort execution (non-blocking)

**New Capabilities:**
- Custom logic injection at key lifecycle points
- Monitoring and logging hooks
- Custom validation hooks
- Event-driven architecture

### 5. Response Channel Refactoring

**Before:** Response channels were in `agents/agentResponse.go` with basic chunk handling.

**After:** Moved to `core/response.go` with enhanced functionality.

**Impact:**
- Enhanced with `ExtendedChunkResponse` (includes AgentName and Trace)
- Added `Start()` method for convenient channel iteration
- Better channel lifecycle management with mutex protection
- Improved error handling

**Migration:**
```go
// Old way
for chunkBytes := range responseCh.Response {
    // manual deserialization
}

// New way
for chunk := range responseCh.Start() {
    // chunk is already ExtendedChunkResponse
    fmt.Printf("[%s] %s", chunk.AgentName, chunk.Content)
}
```

### 6. Initialization Refactoring

**Before:** Monolithic `NewAgent()` function with all initialization logic in one place.

**After:** Split into focused initialization methods for better maintainability.

**Impact:**
- `ensureConfig()` - validation and defaults
- `ensureHooks()` - hook system setup
- `addSystemAgents()` - reasoning agent creation
- `initSystemTools()` - meta and expand tools
- `loadDelegateTool()` - dynamic delegate tool loading
- `setResponseCh()` - response channel setup
- `initAgentContext()` - static context building
- `registerSystemCallbacks()` - system hook registration
- `ensureSystemPrompt()` - prompt building

**Benefits:**
- Better code organization
- Easier to test individual components
- Clearer initialization flow
- Easier to extend

### 7. Agent Structure Changes

**Before:** Agent struct held individual fields directly.

**After:** Agent holds `*AgentConfig` reference and uses getters.

**Impact:**
- Agent now holds `*AgentConfig` instead of individual fields
- Removed direct field access, uses config getters
- `responseCh` stored as agent field instead of created per message
- `agentContext` built once and reused

**Benefits:**
- Better encapsulation
- Consistent configuration access
- Reduced memory footprint

### 8. System Tools

**Before:** Tools were manually added to agents.

**After:** System tools are automatically managed.

**Impact:**
- Meta tool always added automatically
- Expand tool conditionally added based on `CanExpand` config
- Delegate tool dynamically loaded when sub-agents exist

**Benefits:**
- Less boilerplate code
- Consistent tool availability
- Automatic tool management

### 9. History Management

**Before:** History logic mixed with agent logic.

**After:** Separated into dedicated files with better structure.

**Impact:**
- Separated concerns into `history.go` and `agentHistory.go`
- Better integration with persistence layer
- Token tracking added to message storage

**Benefits:**
- Better code organization
- Improved testability
- Enhanced persistence features

### 10. Tool Organization

**Before:** Tools were in flat structure with `baseTool.go`.

**After:** Tools reorganized into packages.

**Impact:**
- Tools reorganized into packages: `delegate/`, `expand/`, `meta/`, `fs/`
- Each tool package has its own `tool.go` with `NewXTool()` constructor
- Tools use `core.Tool` struct instead of extending BaseTool

**Benefits:**
- Better code organization
- Clearer tool boundaries
- Easier to find and maintain tools

## Breaking Changes

1. **Tool Creation:** `core.NewTool()` function doesn't exist - use `&core.Tool{...}` struct literals
2. **Sub-Agents:** Changed from `[]*Agent` to `[]*core.SubAgent`
3. **Agent Config:** `NewAgent()` now takes `*AgentConfig` (pointer) instead of `AgentConfig` (value)
4. **Response Channels:** Must use `Start()` method for iteration instead of direct channel access
5. **Tool Execution Context:** Replaced with `AgentContext` system - context is now built automatically

## Migration Guide

### Updating Tool Creation

```go
// Old
import "github.com/thinktwice/agentForge/src/tools"

type MyTool struct {
    *tools.BaseTool
}

// New
import "github.com/thinktwice/agentForge/src/core"

myTool := &core.Tool{
    Name: "my_tool",
    Description: "Tool description",
    AdvanceDesc: "Advanced details",
    TroubleshootingInfo: "Troubleshooting info",
    Parameters: []core.Parameter{
        {Name: "param", Type: "string", Required: true},
    },
    Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
        return core.NewSuccessResponse("result")
    },
}
```

### Updating Agent Creation

```go
// Old
agent := agents.NewAgent(agents.AgentConfig{
    // config
})

// New (same, but config is now passed by pointer internally)
agent := agents.NewAgent(&agents.AgentConfig{
    // config
})
```

### Updating Response Handling

```go
// Old
responseCh := agent.ChatStream("message")
for chunkBytes := range responseCh.Response {
    var chunk llms.ChunkResponse
    json.Unmarshal(chunkBytes, &chunk)
    // process chunk
}

// New
responseCh := agent.ChatStream("message")
for chunk := range responseCh.Start() {
    // chunk is ExtendedChunkResponse with AgentName and Trace
    fmt.Printf("[%s] %s", chunk.AgentName, chunk.Content)
}
```

## Performance Improvements

1. **Context Building:** Reduced from per-call to once-per-agent initialization
2. **Memory:** Reduced allocations by reusing agent context
3. **Channel Management:** Better lifecycle management reduces goroutine leaks

## New Features

1. **Hook System:** Extensible lifecycle hooks for custom behavior
2. **Agent Context:** Structured context system with static building
3. **Enhanced Response Channels:** Better chunk handling with agent metadata
4. **System Tools:** Automatic tool management

## Deprecated Features

- `tools.BaseTool` - Use `core.Tool` instead
- Direct response channel access - Use `Start()` method
- Manual context building - Now automatic via `AgentContext`


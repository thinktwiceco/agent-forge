# Unit Test Analysis Report

## Executive Summary
The current state of the unit testing suite is critical. While there are some test files present, the **functional code coverage is effectively near 0%** for the core business logic, API server, and tools.

Most existing tests are:
1.  **Skipped by default** (require specific API keys).
2.  **Integration tests** masquerading as unit tests (depend on live LLM APIs).
3.  **Missing entirely** for major components (`persistence`, `tools`, `core`, `apis`).

**Total Project Coverage**: ~14.6% (heavily skewed by helper utils in the root `src` package).
**Core Component Coverage**: < 1%.

## Component Breakdown

### 1. Agents (`src/agents`)
- **Status**: 🔴 Critical
- **Coverage**: Low.
- **Analysis**:
  - Contains `agent_test.go` and `agent_reasoning_test.go`.
  - `TestAgent_Reasoning_TwoTraces` is permanently skipped awaiting "delegate tool implementation".
  - `TestAgent_fooTool_WithRealLLM` requires live `togetherai` or `deepseek` keys and network access.
  - **No pure unit tests** for `Agent` state machine, `agentChat.go` logic, or sub-agent orchestrators (`saCodingAgent.go`, etc.).

### 2. Core Framework (`src/core`)
- **Status**: 🔴 Critical
- **Coverage**: **0.9%**
- **Analysis**:
  - Foundational files like `agentContext.go`, `response.go`, and `tool.go` have almost zero coverage.
  - These are high-risk files as they handle the internal plumbing of the agent (context rehydration, tool validation, response channel management).

### 3. Tools (`src/tools`)
- **Status**: 🔴 Critical
- **Coverage**: **0.0%** (except `expand` tool)
- **Analysis**:
  - Complex tools like `src/tools/git` (Git operations), `src/tools/web` (Scraping), and `src/tools/fs` (Filesystem) have **no test files**.
  - These components interact with the OS and external world, making them prime candidates for bugs, yet they are completely verifying logic.

### 4. APIs & Server (`src/apis`, `cmd`)
- **Status**: 🔴 Critical
- **Coverage**: **0.0%**
- **Analysis**:
  - `src/apis/server.go` (the main HTTP entrypoint) is untested.
  - Endpoints logic (`handleChat`, `InitializeAgentFromConfig`) is not verified.

### 5. Persistence (`src/persistence`)
- **Status**: 🔴 Critical
- **Coverage**: **0.0%** (No test files found)
- **Analysis**:
  - No verification for data storage or retrieval.

## Deep Dive: Existing Tests
Two main test files exist in `src/agents`:
- `agent_test.go`: Tests `AgentConfig` validation (good) and executes a real `ChatStream` with a live LLM (bad for unit testing).
- `agent_reasoning_test.go`: Completely skipped.

**Example of problematic test pattern:**
```go
func TestAgent_fooTool_WithRealLLM(t *testing.T) {
    if !hasAPIKey("togetherai") {
        t.Skip("Skipping test - TogetherAI API key not available")
    }
    // ... makes real network calls
}
```
This pattern prevents CI from running meaningful logic checks unless secrets are injected, and makes tests flaky/slow.

## Recommendations

### Short Term (Immediate Actions)
1.  **Mocking Infrastructure**: Create a `mock` package or use `gomock` to mock `LLMEngine` and `Tool` interfaces.
2.  **Test Core Logic**: Ad tests for `src/core/agentContext.go` and `src/core/response.go` as they don't require external dependencies.
3.  **Refactor Agent Tests**:
    - Split `agent_test.go` into `agent_test.go` (unit tests using mocks) and `agent_integration_test.go` (live API tests).
    - Write a unit test for `ChatStream` that uses a Mock LLM to return a predefined stream and verifies the agent's state transitions.

### Medium Term
1.  **Tool Testing**: Implement tests for `tools/fs` and `tools/git` using temporary directories and `os/exec` references or mocks.
2.  **API Tests**: Add `httptest` based tests for `src/apis/server.go` to verify route handling without spinning up a full server.

# Agentforge Python SDK

Python SDK for interacting with the Agentforge Agent API server.

## Features

- **Automatic Server Management**: SDK can automatically start and stop the Go server
- **Cross-platform Support**: Works on Linux, macOS, and Windows
- **Streaming Responses**: Handle real-time streaming responses from agents
- **Tool Call Support**: Full support for tool calls and tool execution tracking
- **Simple API**: Clean, Pythonic interface

## Installation

### From Source

```bash
# Install using uv (recommended)
uv pip install -e python-sdk/

# Or using pip
pip install -e python-sdk/
```

### Dependencies

- Python 3.8+
- `requests` library

## Quick Start

```python
from agentforge_sdk import AgentforgeClient

# Initialize client (auto-starts server if needed)
client = AgentforgeClient(auto_start=True)

# List available agents
agents = client.list_agents()
print(f"Available agents: {agents}")

# Chat with an agent
for chunk in client.chat("test-agent", "Hello, how are you?"):
    if chunk.status == "streaming":
        print(chunk.content, end="", flush=True)
    elif chunk.status == "completed":
        print(f"\n\nFull response: {chunk.full_content}")
```

## Usage

### Basic Chat

```python
from agentforge_sdk import AgentforgeClient
from agentforge_sdk.constants import StatusStreaming, StatusCompleted

client = AgentforgeClient(auto_start=True)

for chunk in client.chat("test-agent", "What is 2+2?"):
    if chunk.status == StatusStreaming:
        print(chunk.content, end="", flush=True)
    elif chunk.status == StatusCompleted:
        print(f"\n\nTokens used: {chunk.total_tokens}")
```

### Server Lifecycle Management

```python
from agentforge_sdk import AgentforgeClient
from agentforge_sdk.server_manager import ServerManager

# Manual server management
manager = ServerManager(port=8080)
manager.start(wait_for_ready=True)

# Use client with manual server management
client = AgentforgeClient(auto_start=False)
agents = client.list_agents()

# Stop server
manager.stop()
```

### Handling Tool Calls

```python
from agentforge_sdk import AgentforgeClient
from agentforge_sdk.constants import (
    StatusToolCall,
    StatusToolExecuting,
    StatusToolResult,
)

client = AgentforgeClient(auto_start=True)

for chunk in client.chat("test-agent", "List files in current directory"):
    if chunk.status == StatusToolCall:
        print("Tool calls requested:")
        for tool_call in chunk.tool_calls:
            print(f"  - {tool_call.name}")
    
    elif chunk.status == StatusToolExecuting:
        print(f"Executing: {chunk.tool_executing.name}")
    
    elif chunk.status == StatusToolResult:
        for result in chunk.tool_results:
            if result.success:
                print(f"✓ {result.tool_name}: {result.result}")
            else:
                print(f"✗ {result.tool_name}: {result.error}")
```

## API Reference

### AgentforgeClient

Main client class for interacting with the Agentforge server.

#### Methods

- `list_agents() -> List[str]`: List all available agents
- `chat(agent_name: str, message: str) -> Iterator[ChunkResponse]`: Send a chat message and stream responses
- `start_server() -> bool`: Start the server if not running
- `stop_server() -> bool`: Stop the server

#### Parameters

- `base_url` (str): Base URL of the server (default: "http://localhost:8080")
- `server_path` (Optional[str]): Path to server binary (default: None, auto-detects)
- `auto_start` (bool): Automatically start server if not running (default: True)
- `port` (int): Port number for the server (default: 8080)

### ServerManager

Manages the lifecycle of the Agentforge server process.

#### Methods

- `start(wait_for_ready: bool = True, timeout: int = 30) -> bool`: Start server as daemon
- `stop(force: bool = False) -> bool`: Stop server
- `restart() -> bool`: Restart server
- `is_running() -> bool`: Check if server is running
- `get_status() -> ServerStatus`: Get server status
- `get_logs(lines: int = 100) -> List[str]`: Get recent server logs

### ChunkResponse

Represents a streaming response chunk from the agent.

#### Fields

- `content` (str): Current chunk content
- `delta` (str): Incremental delta
- `full_content` (str): Accumulated full content
- `status` (str): Status (see constants)
- `type` (str): Response type (see constants)
- `tool_calls` (Optional[List[ToolCall]]): Tool calls requested
- `tool_executing` (Optional[ToolCall]): Tool currently executing
- `tool_results` (Optional[List[ToolResult]]): Tool execution results
- `prompt_tokens` (Optional[int]): Input tokens consumed
- `completion_tokens` (Optional[int]): Output tokens generated
- `total_tokens` (Optional[int]): Total tokens used
- `agent_name` (str): Name of the agent
- `trace` (str): Trace information

## Constants

### Status Constants

- `StatusStreaming`: Content is actively being streamed
- `StatusCompleted`: Streaming response has finished
- `StatusError`: An error occurred
- `StatusToolCall`: LLM is requesting tool executions
- `StatusToolExecuting`: A tool is currently being executed
- `StatusToolResult`: Tool execution has completed

### Type Constants

- `TypeContent`: Regular content chunk
- `TypeToolCall`: Tool call request chunk
- `TypeToolExecuting`: Tool execution progress chunk
- `TypeToolResult`: Tool result chunk
- `TypeCompletion`: Final completion chunk

## Exceptions

- `AgentforgeError`: Base exception for all SDK errors
- `APIError`: Raised when an API request fails
- `AgentNotFoundError`: Raised when an agent is not found
- `ServerError`: Raised when server operations fail
- `ServerNotRunningError`: Raised when server is not running
- `BinaryNotFoundError`: Raised when server binary is not found

## Examples

See the `examples/` directory for more examples:

- `basic_chat.py`: Basic chat example
- `streaming_chat.py`: Advanced streaming with tool call handling
- `server_lifecycle.py`: Server lifecycle management

## Development

### Setup

```bash
# Install development dependencies
uv pip install -e "python-sdk/[dev]"
```

### Running Tests

```bash
cd python-sdk
uv run pytest tests/ -v
```

### Linting

```bash
cd python-sdk
ruff check .
```

## Versioning

The SDK uses independent versioning from the Go server. Version is stored in `VERSION` file and follows semantic versioning (e.g., `0.1.0`).

## License

MIT

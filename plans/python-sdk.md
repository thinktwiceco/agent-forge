# Python SDK Plan for ThinkTwice Agent

## Overview
Create a Python SDK that communicates with the compiled Go executable from `cmd/server/main.go`. The SDK will provide a clean Python interface to interact with the ThinkTwice Agent API server.

## Recommended Project Structure

```
thinktwice-agent/
├── python-sdk/              # New Python SDK directory
│   ├── thinktwice_sdk/      # Main package
│   │   ├── __init__.py
│   │   ├── client.py        # Main client class
│   │   ├── models.py        # Data models (ChunkResponse, etc.)
│   │   ├── exceptions.py    # Custom exceptions
│   │   └── constants.py     # Status/Type constants
│   ├── examples/
│   │   ├── basic_chat.py
│   │   └── streaming_chat.py
│   ├── tests/
│   │   ├── __init__.py
│   │   └── test_client.py
│   ├── README.md
│   ├── pyproject.toml       # Modern Python packaging
│   ├── requirements.txt     # For pip install
│   └── setup.py             # Optional, for older tools
```

## Key Design Decisions

1. **Package Location**: `python-sdk/` at the root level keeps it separate from Go code
2. **Package Name**: `thinktwice_sdk` (Python-friendly naming)
3. **Dependencies**: Use `requests` for synchronous HTTP, optionally `httpx` for async
4. **Streaming**: Handle NDJSON (newline-delimited JSON) line-by-line

## Implementation Approach

### Option A: Simple Synchronous Client (Recommended to Start)
- Uses `requests` library
- Simple streaming with `iter_lines()`
- Easy to use and debug

### Option B: Async Client (For Advanced Use)
- Uses `httpx` or `aiohttp`
- Better for concurrent operations
- More complex but more powerful

## API Endpoints

Based on `src/apis/server.go`:

1. **GET `/api/server/agents`**
   - Returns: `{"agents": ["agent1", "agent2", ...]}`

2. **POST `/api/server/{agentname}/chat`**
   - Request: `{"message": "user message"}`
   - Response: Streaming NDJSON with `ChunkResponse` objects

## Data Models

### ChunkResponse Structure
```python
@dataclass
class ChunkResponse:
    content: str = ""
    delta: str = ""
    full_content: str = ""
    status: str = ""  # streaming, completed, error, tool-call, tool-executing, tool-result
    type: str = ""    # content, tool-call, tool-executing, tool-result, completion
    tool_calls: Optional[List[ToolCall]] = None
    tool_executing: Optional[ToolCall] = None
    tool_results: Optional[List[ToolResult]] = None
    prompt_tokens: Optional[int] = None
    completion_tokens: Optional[int] = None
    total_tokens: Optional[int] = None
    agent_name: str = ""
    trace: str = ""
```

### Status Constants
- `StatusStreaming = "streaming"`
- `StatusCompleted = "completed"`
- `StatusError = "error"`
- `StatusToolCall = "tool-call"`
- `StatusToolExecuting = "tool-executing"`
- `StatusToolResult = "tool-result"`

### Type Constants
- `TypeContent = "content"`
- `TypeToolCall = "tool-call"`
- `TypeToolExecuting = "tool-executing"`
- `TypeToolResult = "tool-result"`
- `TypeCompletion = "completion"`

## Sample Client Implementation

```python
# python-sdk/thinktwice_sdk/client.py
import requests
import json
from typing import Iterator, List, Optional
from .models import ChunkResponse
from .exceptions import AgentNotFoundError, APIError

class ThinkTwiceClient:
    def __init__(self, base_url: str = "http://localhost:8080"):
        self.base_url = base_url.rstrip('/')
    
    def list_agents(self) -> List[str]:
        """List all available agents."""
        response = requests.get(f"{self.base_url}/api/server/agents")
        response.raise_for_status()
        return response.json()["agents"]
    
    def chat(self, agent_name: str, message: str) -> Iterator[ChunkResponse]:
        """Send a chat message and stream responses."""
        url = f"{self.base_url}/api/server/{agent_name}/chat"
        payload = {"message": message}
        
        response = requests.post(url, json=payload, stream=True)
        
        if response.status_code == 404:
            raise AgentNotFoundError(f"Agent '{agent_name}' not found")
        response.raise_for_status()
        
        # Parse NDJSON (one JSON object per line)
        for line in response.iter_lines():
            if line:
                chunk_data = json.loads(line)
                yield ChunkResponse(**chunk_data)
```

## Setup Files

### pyproject.toml
```toml
[build-system]
requires = ["setuptools>=61.0", "wheel"]
build-backend = "setuptools.build_meta"

[project]
name = "thinktwice-sdk"
version = "0.1.0"
description = "Python SDK for ThinkTwice Agent"
readme = "README.md"
requires-python = ">=3.8"
dependencies = [
    "requests>=2.28.0",
]

[project.optional-dependencies]
async = ["httpx>=0.24.0"]
```

### requirements.txt
```
requests>=2.28.0
```

## Benefits of This Structure

1. **Separation**: Python code is isolated from Go code
2. **Installable**: Can be installed with `pip install -e python-sdk/`
3. **Testable**: Clear test structure
4. **Extensible**: Easy to add async support later
5. **Standard**: Follows Python packaging conventions

## Integration with Go Executable

The SDK assumes the Go server is running. Options:
- Document that users need to run `./main -port 8080` first
- Add a helper to check if server is running
- Optionally add a subprocess wrapper to start/stop the server

## Example Usage

```python
from thinktwice_sdk import ThinkTwiceClient

client = ThinkTwiceClient("http://localhost:8080")

# List agents
agents = client.list_agents()
print(f"Available agents: {agents}")

# Chat with agent
for chunk in client.chat("test-agent", "Hello, how are you?"):
    if chunk.status == "streaming":
        print(chunk.content, end="", flush=True)
    elif chunk.status == "completed":
        print(f"\n\nFull response: {chunk.full_content}")
    elif chunk.status == "tool-executing":
        print(f"\n[Executing tool: {chunk.tool_executing.name}]")
    elif chunk.status == "error":
        print(f"\nError: {chunk.content}")
```

## Next Steps

1. Create the directory structure
2. Implement basic models and constants
3. Implement synchronous client
4. Add error handling and exceptions
5. Write examples and tests
6. Add documentation
7. (Optional) Add async support later


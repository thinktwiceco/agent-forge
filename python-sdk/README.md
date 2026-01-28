# Agent Forge Python SDK

The official Python SDK for interacting with the Agent Forge server.

## Installation

```bash
pip install agent-forge-sdk
```

## Usage

### Client Initialization

The `AgentForgeClient` manages the connection to the server. It can automatically start the server binary if it's not already running.

```python
from agentforge_sdk import AgentForgeClient

# Initialize client (auto-starts server by default)
client = AgentForgeClient()

# Initialize with custom settings
client = AgentForgeClient(
    base_url="http://localhost:8080",
    auto_start=True
)
```

### List Available Agents

Retrieve a list of all available agents on the server.

```python
try:
    agents = client.list_agents()
    print("Available agents:", agents)
except Exception as e:
    print(f"Error listing agents: {e}")
```

### Chat with an Agent

Send a message to an agent and stream the response.

```python
from agentforge_sdk.constants import TypeContent

agent_name = "Assistant"
message = "Hello, how can you help me?"

try:
    for chunk in client.chat(agent_name, message):
        if chunk.type == TypeContent:
            print(chunk.content, end="", flush=True)
except Exception as e:
    print(f"Chat error: {e}")
```

### Server Management

You can manually control the server process if needed.

```python
# Start the server
client.start_server()

# Stop the server
client.stop_server()
```

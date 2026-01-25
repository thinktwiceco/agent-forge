from agentforge_sdk import AgentforgeClient

import os

# Path to the server binary
server_path = os.path.abspath(os.path.join(os.path.dirname(__file__), "../python-sdk/bin/agentforge-server-linux-amd64"))

client = AgentforgeClient("http://localhost:8080", server_path=server_path)

# Chat with the agent
for chunk in client.chat("test-agent", "Hello!"):
    print(chunk.content, end="", flush=True)
"""Basic chat example with Agentforge SDK."""

from agentforge_sdk import AgentforgeClient
from agentforge_sdk.constants import StatusCompleted, StatusError, StatusStreaming


def main():
    """Run basic chat example."""
    # Initialize client (auto-starts server if needed)
    client = AgentforgeClient(auto_start=True)

    # List available agents
    print("Available agents:")
    agents = client.list_agents()
    for agent in agents:
        print(f"  - {agent}")

    if not agents:
        print("No agents available. Make sure the server is configured with agents.")
        return

    # Use the first available agent
    agent_name = agents[0]
    print(f"\nChatting with agent: {agent_name}\n")

    # Send a message
    message = "Hello! How are you?"
    print(f"You: {message}\n")
    print("Agent: ", end="", flush=True)

    # Stream response
    full_response = ""
    for chunk in client.chat(agent_name, message):
        if chunk.status == StatusStreaming:
            # Print incremental content
            if chunk.content:
                print(chunk.content, end="", flush=True)
                full_response += chunk.content
        elif chunk.status == StatusCompleted:
            # Print final content if any
            if chunk.full_content and chunk.full_content != full_response:
                remaining = chunk.full_content[len(full_response) :]
                print(remaining, end="", flush=True)
            print("\n")
        elif chunk.status == StatusError:
            print(f"\nError: {chunk.content}")
            break

    print(f"\nFull response: {full_response}")


if __name__ == "__main__":
    main()

"""Advanced streaming chat example with tool call handling."""

from agentforge_sdk import AgentforgeClient
from agentforge_sdk.constants import (
    StatusCompleted,
    StatusError,
    StatusStreaming,
    StatusToolCall,
    StatusToolExecuting,
    StatusToolResult,
)


def main():
    """Run streaming chat example with tool call handling."""
    # Initialize client
    client = AgentforgeClient(auto_start=True)

    # List agents
    agents = client.list_agents()
    if not agents:
        print("No agents available.")
        return

    agent_name = agents[0]
    print(f"Chatting with agent: {agent_name}\n")

    # Send a message that might trigger tool calls
    message = "What files are in the current directory?"
    print(f"You: {message}\n")
    print("Agent: ", end="", flush=True)

    # Stream response with tool call handling
    for chunk in client.chat(agent_name, message):
        if chunk.status == StatusStreaming:
            # Print content as it streams
            if chunk.content:
                print(chunk.content, end="", flush=True)

        elif chunk.status == StatusToolCall:
            # Tool calls requested
            print("\n\n[Tool Calls Requested]")
            if chunk.tool_calls:
                for tool_call in chunk.tool_calls:
                    print(f"  - {tool_call.name}({tool_call.arguments})")

        elif chunk.status == StatusToolExecuting:
            # Tool is executing
            if chunk.tool_executing:
                print(f"\n[Executing: {chunk.tool_executing.name}]", end="", flush=True)

        elif chunk.status == StatusToolResult:
            # Tool results available
            print("\n[Tool Results]")
            if chunk.tool_results:
                for result in chunk.tool_results:
                    status = "✓" if result.success else "✗"
                    print(f"  {status} {result.tool_name}: {result.result[:100]}...")

        elif chunk.status == StatusCompleted:
            # Response completed
            print("\n\n[Completed]")
            if chunk.total_tokens:
                print(f"Tokens used: {chunk.total_tokens}")

        elif chunk.status == StatusError:
            print(f"\n[Error: {chunk.content}]")
            break


if __name__ == "__main__":
    main()

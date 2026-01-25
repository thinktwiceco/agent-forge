"""Server lifecycle management example."""

from agentforge_sdk import AgentforgeClient
from agentforge_sdk.server_manager import ServerManager, ServerStatus


def main():
    """Demonstrate server lifecycle management."""
    print("=== Server Lifecycle Management Example ===\n")

    # Create server manager
    manager = ServerManager(port=8080)

    # Check initial status
    status = manager.get_status()
    print(f"Initial status: {status.value}")

    # Start server
    print("\nStarting server...")
    try:
        manager.start(wait_for_ready=True, timeout=30)
        print("Server started successfully!")
    except Exception as e:
        print(f"Failed to start server: {e}")
        return

    # Check status
    status = manager.get_status()
    print(f"Status after start: {status.value}")

    # Get logs
    print("\nRecent server logs:")
    logs = manager.get_logs(lines=10)
    for log_line in logs[-5:]:  # Show last 5 lines
        print(f"  {log_line}")

    # Use client to interact with server
    print("\n--- Using client ---")
    client = AgentforgeClient(auto_start=False)  # Don't auto-start, we manage it

    try:
        agents = client.list_agents()
        print(f"Available agents: {agents}")
    except Exception as e:
        print(f"Error listing agents: {e}")

    # Stop server
    print("\nStopping server...")
    try:
        manager.stop()
        print("Server stopped successfully!")
    except Exception as e:
        print(f"Failed to stop server: {e}")

    # Check final status
    status = manager.get_status()
    print(f"Final status: {status.value}")


if __name__ == "__main__":
    main()

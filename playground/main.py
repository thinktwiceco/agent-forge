from agentforge_sdk import AgentforgeClient

import os

# Path to the server binary
server_path = os.path.abspath(os.path.join(os.path.dirname(__file__), "../python-sdk/bin/agentforge-server-linux-amd64"))

client = AgentforgeClient("http://localhost:8080", server_path=server_path)


def chat(message: str):
    # Chat with the agent
    for chunk in client.chat("test-agent", message):
        print(chunk.content, end="", flush=True)


if __name__ == "__main__":
    import argparse

    parser = argparse.ArgumentParser(description="CLI for interacting with agentforge agent")
    parser.add_argument("--stop", action="store_true", help="Stop the underlying server")
    parser.add_argument("message", nargs="?", help="Message to send to the agent")

    args = parser.parse_args()

    if args.stop:
        # Stop the server and exit
        print("Stopping the server...")
        client.stop_server()
        print("Server stopped.")
    else:
        # Forward message to the agent
        if not args.message:
            parser.error("Message is required when not using --stop")
        
        try:
            chat(args.message)
        finally:
            # Stop the underlying server
            client.stop_server()


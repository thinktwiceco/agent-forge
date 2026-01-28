from agentforge_sdk import AgentForgeClient

import os

# Path to the server binary
server_path = os.path.abspath(os.path.join(os.path.dirname(__file__), "../python-sdk/bin/agentforge-server-linux-amd64"))

client = AgentForgeClient("http://localhost:8080", server_path=server_path)


def chat(message: str):
    # Chat with the agent
    last_trace_id = None
    for chunk in client.chat("test-agent", message):
        # Only print trace header if it changes
        current_trace_id = (chunk.agent_name, chunk.trace)
        if chunk.trace and current_trace_id != last_trace_id:
            print(f"\n[{chunk.agent_name} - {chunk.trace}]", end=" ", flush=True)
            last_trace_id = current_trace_id
        
        if chunk.tool_calls:
            for tc in chunk.tool_calls:
                print(f"\n[Tool Call] {tc.name}({tc.arguments})", flush=True)
        
        if chunk.tool_executing:
            print(f"\n[Executing] {chunk.tool_executing.name}...", flush=True)
            
        if chunk.tool_results:
            for tr in chunk.tool_results:
                status = "Success" if tr.success else "Failed"
                print(f"\n[Tool Result] {tr.tool_name}: {status}", flush=True)
                if not tr.success:
                    print(f"Error: {tr.error}", flush=True)

        if chunk.delta:
            print(chunk.delta, end="", flush=True)


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


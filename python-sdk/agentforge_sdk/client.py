"""Main client class for Agent Forge SDK."""

from __future__ import annotations

import json
from typing import TYPE_CHECKING

import requests

from agentforge_sdk.constants import StatusError
from agentforge_sdk.exceptions import (
    AgentNotFoundError,
    APIError,
    ServerNotRunningError,
)
from agentforge_sdk.models import ChunkResponse
from agentforge_sdk.server_manager import ServerManager

if TYPE_CHECKING:
    from collections.abc import Iterator


class AgentForgeClient:
    """Client for interacting with the Agent Forge API server."""

    def __init__(
        self,
        base_url: str = "http://localhost:8080",
        server_path: str | None = None,
        auto_start: bool = True,
        port: int = 8080,
    ):
        """Initialize Agent Forge client.

        Args:
            base_url: Base URL of the server (default: http://localhost:8080).
            server_path: Optional path to server binary. If None, auto-detects from bin/.
            auto_start: If True, automatically start server if not running (default: True).
            port: Port number for the server (default: 8080). Only used if server_path is provided.
        """
        self.base_url = base_url.rstrip("/")
        self.auto_start = auto_start
        self.server_manager: ServerManager | None = None

        # Only create server manager if server_path is provided or auto_start is enabled
        if server_path is not None or auto_start:
            # Extract port from base_url if not explicitly provided
            if port == 8080 and ":" in base_url:
                try:
                    port = int(base_url.split(":")[-1].split("/")[0])
                except ValueError:
                    pass  # Use default

            self.server_manager = ServerManager(server_path=server_path, port=port)

            # Auto-start if enabled and server is not running
            if auto_start and not self.server_manager.is_running():
                self.server_manager.start()

    def list_agents(self) -> list[str]:
        """List all available agents.

        Returns:
            List of agent names.

        Raises:
            APIError: If the API request fails.
            ServerNotRunningError: If server is not running and auto_start is False.
        """
        self._ensure_server_running()

        try:
            response = requests.get(f"{self.base_url}/api/server/agents", timeout=10)
            response.raise_for_status()
            data = response.json()
            return data.get("agents", [])
        except requests.exceptions.RequestException as e:
            raise APIError(f"Failed to list agents: {e}", response_text=str(e)) from e

    def chat(self, agent_name: str, message: str) -> Iterator[ChunkResponse]:
        """Send a chat message and stream responses.

        Args:
            agent_name: Name of the agent to chat with.
            message: User message to send.

        Yields:
            ChunkResponse objects as they are received.

        Raises:
            AgentNotFoundError: If the agent is not found.
            APIError: If the API request fails.
            ServerNotRunningError: If server is not running and auto_start is False.
        """
        self._ensure_server_running()

        url = f"{self.base_url}/api/server/{agent_name}/chat"
        payload = {"message": message}

        try:
            response = requests.post(url, json=payload, stream=True, timeout=30)
        except requests.exceptions.RequestException as e:
            raise APIError(f"Failed to send chat message: {e}") from e

        if response.status_code == 404:
            raise AgentNotFoundError(agent_name)

        response.raise_for_status()

        # Parse NDJSON (one JSON object per line)
        for line in response.iter_lines():
            if line:
                try:
                    chunk_data = json.loads(line)
                    chunk = ChunkResponse.from_dict(chunk_data)
                    yield chunk

                    # Stop on error status
                    if chunk.status == StatusError:
                        break
                except json.JSONDecodeError:
                    # Invalid JSON, skip this line
                    continue

    def start_server(self) -> bool:
        """Start the server if not running.

        Returns:
            True if server started successfully, False otherwise.

        Raises:
            ServerError: If server manager is not available.
        """
        if self.server_manager is None:
            raise ServerNotRunningError("Server manager not initialized")

        return self.server_manager.start()

    def stop_server(self) -> bool:
        """Stop the server.

        Returns:
            True if server stopped successfully, False otherwise.

        Raises:
            ServerError: If server manager is not available.
        """
        if self.server_manager is None:
            raise ServerNotRunningError("Server manager not initialized")

        return self.server_manager.stop()

    def _ensure_server_running(self) -> None:
        """Ensure server is running, raise exception if not."""
        if self.server_manager is not None:
            if not self.server_manager.is_running():
                if self.auto_start:
                    self.server_manager.start()
                else:
                    raise ServerNotRunningError(
                        "Server is not running. Set auto_start=True or call start_server() first."
                    )
        else:
            # No server manager, assume server is managed externally
            # Just check if it's reachable
            try:
                response = requests.get(f"{self.base_url}/health", timeout=2)
                response.raise_for_status()
            except requests.exceptions.RequestException:
                raise ServerNotRunningError(
                    "Server is not reachable. Ensure the server is running."
                )


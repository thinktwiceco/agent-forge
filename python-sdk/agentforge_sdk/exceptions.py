"""Custom exceptions for Agent Forge SDK."""

from __future__ import annotations


class AgentForgeError(Exception):
    """Base exception for all Agent Forge SDK errors."""

    pass


class APIError(AgentForgeError):
    """Raised when an API request fails."""

    def __init__(self, message: str, status_code: int | None = None, response_text: str | None = None):
        super().__init__(message)
        self.status_code = status_code
        self.response_text = response_text


class AgentNotFoundError(APIError):
    """Raised when an agent is not found."""

    def __init__(self, agent_name: str):
        super().__init__(f"Agent '{agent_name}' not found", status_code=404)
        self.agent_name = agent_name


class ServerError(AgentForgeError):
    """Raised when server operations fail."""

    pass


class ServerNotRunningError(ServerError):
    """Raised when attempting to use a server that is not running."""

    def __init__(self, message: str = "Server is not running"):
        super().__init__(message)


class ServerStartError(ServerError):
    """Raised when server fails to start."""

    def __init__(self, message: str, exit_code: int | None = None):
        super().__init__(message)
        self.exit_code = exit_code


class ServerStopError(ServerError):
    """Raised when server fails to stop."""

    pass


class BinaryNotFoundError(ServerError):
    """Raised when server binary is not found."""

    def __init__(self, platform: str, arch: str):
        super().__init__(f"Server binary not found for platform {platform}/{arch}")
        self.platform = platform
        self.arch = arch


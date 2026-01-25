"""Server lifecycle management for ThinkTwice SDK.

Handles starting, stopping, and monitoring the Go server process as a daemon.
"""

import os
import platform
import subprocess
import sys
import tempfile
import time
from enum import Enum
from pathlib import Path
from typing import List, Optional

import requests

from thinktwice_sdk.exceptions import (
    BinaryNotFoundError,
    ServerError,
    ServerNotRunningError,
    ServerStartError,
    ServerStopError,
)


class ServerStatus(Enum):
    """Server status enumeration."""

    STOPPED = "stopped"
    RUNNING = "running"
    STARTING = "starting"
    STOPPING = "stopping"
    ERROR = "error"


class ServerManager:
    """Manages the lifecycle of the ThinkTwice server process."""

    def __init__(self, server_path: Optional[str] = None, port: int = 8080):
        """Initialize server manager.

        Args:
            server_path: Optional path to server binary. If None, auto-detects from bin/ directory.
            port: Port number for the server (default: 8080).
        """
        self.port = port
        self.base_url = f"http://localhost:{port}"
        self.server_path = server_path or self._detect_binary()
        self.pid_file = Path(tempfile.gettempdir()) / f"thinktwice-server-{port}.pid"
        self.log_file = Path(tempfile.gettempdir()) / f"thinktwice-server-{port}.log"
        self._process: Optional[subprocess.Popen] = None

    def _detect_binary(self) -> str:
        """Detect the appropriate server binary for the current platform."""
        system = platform.system().lower()
        machine = platform.machine().lower()

        # Map platform to binary name
        if system == "linux":
            if machine in ("x86_64", "amd64"):
                binary_name = "thinktwice-server-linux-amd64"
            else:
                raise BinaryNotFoundError(system, machine)
        elif system == "darwin":
            if machine in ("arm64", "aarch64"):
                binary_name = "thinktwice-server-darwin-arm64"
            elif machine in ("x86_64", "amd64"):
                binary_name = "thinktwice-server-darwin-amd64"
            else:
                raise BinaryNotFoundError(system, machine)
        elif system == "windows":
            if machine in ("x86_64", "amd64"):
                binary_name = "thinktwice-server-windows-amd64.exe"
            else:
                raise BinaryNotFoundError(system, machine)
        else:
            raise BinaryNotFoundError(system, machine)

        # Get the SDK package directory
        sdk_dir = Path(__file__).parent.parent
        binary_path = sdk_dir / "bin" / binary_name

        if not binary_path.exists():
            raise BinaryNotFoundError(system, machine)

        return str(binary_path)

    def _get_pid(self) -> Optional[int]:
        """Get the PID from the PID file."""
        if not self.pid_file.exists():
            return None

        try:
            with open(self.pid_file, "r") as f:
                pid = int(f.read().strip())
                # Check if process is still running
                if sys.platform == "win32":
                    # Windows: try to get process info
                    try:
                        os.kill(pid, 0)
                        return pid
                    except (OSError, ProcessLookupError):
                        return None
                else:
                    # Unix: use kill(0) to check if process exists
                    try:
                        os.kill(pid, 0)
                        return pid
                    except (OSError, ProcessLookupError):
                        return None
        except (ValueError, IOError):
            return None

    def _save_pid(self, pid: int) -> None:
        """Save PID to file."""
        with open(self.pid_file, "w") as f:
            f.write(str(pid))

    def _remove_pid(self) -> None:
        """Remove PID file."""
        if self.pid_file.exists():
            self.pid_file.unlink()

    def _check_health(self, timeout: float = 1.0) -> bool:
        """Check if server is healthy by making a request to /api/server/agents."""
        try:
            response = requests.get(f"{self.base_url}/api/server/agents", timeout=timeout)
            return response.status_code == 200
        except (requests.RequestException, requests.Timeout):
            return False

    def start(self, wait_for_ready: bool = True, timeout: int = 30) -> bool:
        """Start server as daemon.

        Args:
            wait_for_ready: Whether to wait for server to be ready before returning.
            timeout: Maximum time to wait for server to be ready (seconds).

        Returns:
            True if server started successfully, False otherwise.

        Raises:
            ServerStartError: If server fails to start.
        """
        if self.is_running():
            return True

        # Prepare environment
        env = os.environ.copy()

        # Start process
        if sys.platform == "win32":
            # Windows: use CREATE_NEW_PROCESS_GROUP
            creation_flags = subprocess.CREATE_NEW_PROCESS_GROUP
            startupinfo = subprocess.STARTUPINFO()
            startupinfo.dwFlags |= subprocess.STARTF_USESHOWWINDOW
            startupinfo.wShowWindow = subprocess.SW_HIDE

            with open(self.log_file, "w") as log_file:
                process = subprocess.Popen(
                    [self.server_path, "-port", str(self.port)],
                    stdout=log_file,
                    stderr=subprocess.STDOUT,
                    env=env,
                    creationflags=creation_flags,
                    startupinfo=startupinfo,
                )
        else:
            # Unix (Linux/macOS): detach process
            with open(self.log_file, "w") as log_file:
                process = subprocess.Popen(
                    [self.server_path, "-port", str(self.port)],
                    stdout=log_file,
                    stderr=subprocess.STDOUT,
                    env=env,
                    start_new_session=True,
                )

        self._process = process
        self._save_pid(process.pid)

        if wait_for_ready:
            # Wait for server to be ready
            start_time = time.time()
            while time.time() - start_time < timeout:
                if self._check_health():
                    return True
                time.sleep(0.5)

            # If we get here, server didn't become ready
            # Check if process is still running
            if process.poll() is not None:
                # Process exited
                exit_code = process.returncode
                raise ServerStartError(
                    f"Server process exited with code {exit_code}", exit_code=exit_code
                )
            else:
                # Process is running but not responding
                raise ServerStartError(
                    f"Server started but did not become ready within {timeout} seconds"
                )

        return True

    def stop(self, force: bool = False) -> bool:
        """Stop server.

        Args:
            force: If True, force kill the process. If False, send SIGTERM and wait.

        Returns:
            True if server stopped successfully, False otherwise.

        Raises:
            ServerStopError: If server fails to stop.
        """
        if not self.is_running():
            return True

        pid = self._get_pid()
        if pid is None:
            self._remove_pid()
            return True

        try:
            if force or sys.platform == "win32":
                # Force kill
                if sys.platform == "win32":
                    subprocess.run(["taskkill", "/F", "/PID", str(pid)], check=False)
                else:
                    os.kill(pid, 9)  # SIGKILL
            else:
                # Graceful shutdown
                os.kill(pid, 15)  # SIGTERM
                # Wait up to 10 seconds for process to exit
                for _ in range(20):
                    try:
                        os.kill(pid, 0)  # Check if process exists
                        time.sleep(0.5)
                    except (OSError, ProcessLookupError):
                        # Process exited
                        break
                else:
                    # Process didn't exit, force kill
                    os.kill(pid, 9)  # SIGKILL

            # Clean up
            self._remove_pid()
            self._process = None
            return True
        except (OSError, ProcessLookupError) as e:
            raise ServerStopError(f"Failed to stop server: {e}") from e

    def restart(self) -> bool:
        """Restart server.

        Returns:
            True if server restarted successfully, False otherwise.
        """
        self.stop()
        time.sleep(1)  # Brief pause between stop and start
        return self.start()

    def is_running(self) -> bool:
        """Check if server is running.

        Returns:
            True if server is running and healthy, False otherwise.
        """
        return self._check_health()

    def get_status(self) -> ServerStatus:
        """Get server status.

        Returns:
            ServerStatus enum value.
        """
        if self.is_running():
            return ServerStatus.RUNNING
        elif self._get_pid() is not None:
            return ServerStatus.ERROR
        else:
            return ServerStatus.STOPPED

    def get_logs(self, lines: int = 100) -> List[str]:
        """Get recent server logs.

        Args:
            lines: Number of lines to retrieve (default: 100).

        Returns:
            List of log lines (most recent first).
        """
        if not self.log_file.exists():
            return []

        try:
            with open(self.log_file, "r") as f:
                all_lines = f.readlines()
                # Return last N lines
                return [line.rstrip("\n") for line in all_lines[-lines:]]
        except IOError:
            return []


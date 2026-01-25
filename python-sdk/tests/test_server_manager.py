"""Tests for server manager."""

import os
import platform
import tempfile
from pathlib import Path
from unittest.mock import MagicMock, patch

import pytest

from agentforge_sdk.exceptions import BinaryNotFoundError, ServerStartError
from agentforge_sdk.server_manager import ServerManager, ServerStatus


def test_server_manager_initialization():
    """Test server manager initialization."""
    with patch("agentforge_sdk.server_manager.Path.exists", return_value=True):
        manager = ServerManager(port=8080)
        assert manager.port == 8080
        assert manager.base_url == "http://localhost:8080"


def test_detect_binary_linux():
    """Test binary detection for Linux."""
    with patch("platform.system", return_value="Linux"), patch(
        "platform.machine", return_value="x86_64"
    ), patch("agentforge_sdk.server_manager.Path.exists", return_value=True), patch(
        "agentforge_sdk.server_manager.Path.__truediv__",
        return_value=Path("/test/bin/agentforge-server-linux-amd64"),
    ):
        manager = ServerManager()
        assert "linux-amd64" in manager.server_path or manager.server_path.endswith(
            "agentforge-server-linux-amd64"
        )


def test_detect_binary_darwin_arm64():
    """Test binary detection for macOS ARM64."""
    with patch("platform.system", return_value="Darwin"), patch(
        "platform.machine", return_value="arm64"
    ), patch("agentforge_sdk.server_manager.Path.exists", return_value=True), patch(
        "agentforge_sdk.server_manager.Path.__truediv__",
        return_value=Path("/test/bin/agentforge-server-darwin-arm64"),
    ):
        manager = ServerManager()
        assert "darwin-arm64" in manager.server_path or manager.server_path.endswith(
            "agentforge-server-darwin-arm64"
        )


def test_detect_binary_not_found():
    """Test binary detection when binary doesn't exist."""
    with patch("platform.system", return_value="Linux"), patch(
        "platform.machine", return_value="x86_64"
    ), patch("agentforge_sdk.server_manager.Path.exists", return_value=False):
        with pytest.raises(BinaryNotFoundError):
            ServerManager()


def test_check_health_success():
    """Test health check when server is healthy."""
    manager = ServerManager(server_path="/fake/path", port=8080)

    with patch("requests.get") as mock_get:
        mock_response = MagicMock()
        mock_response.status_code = 200
        mock_get.return_value = mock_response

        assert manager._check_health() is True


def test_check_health_failure():
    """Test health check when server is not healthy."""
    manager = ServerManager(server_path="/fake/path", port=8080)

    with patch("requests.get", side_effect=Exception("Connection error")):
        assert manager._check_health() is False


def test_is_running():
    """Test is_running method."""
    manager = ServerManager(server_path="/fake/path", port=8080)

    with patch.object(manager, "_check_health", return_value=True):
        assert manager.is_running() is True

    with patch.object(manager, "_check_health", return_value=False):
        assert manager.is_running() is False


def test_get_status():
    """Test get_status method."""
    manager = ServerManager(server_path="/fake/path", port=8080)

    with patch.object(manager, "_check_health", return_value=True):
        assert manager.get_status() == ServerStatus.RUNNING

    with patch.object(manager, "_check_health", return_value=False), patch.object(
        manager, "_get_pid", return_value=None
    ):
        assert manager.get_status() == ServerStatus.STOPPED


def test_get_logs():
    """Test get_logs method."""
    manager = ServerManager(server_path="/fake/path", port=8080)

    # Create temporary log file
    with tempfile.NamedTemporaryFile(mode="w", delete=False) as f:
        f.write("Line 1\nLine 2\nLine 3\n")
        log_path = f.name

    manager.log_file = Path(log_path)

    try:
        logs = manager.get_logs(lines=2)
        assert len(logs) == 2
        assert "Line 2" in logs
        assert "Line 3" in logs
    finally:
        os.unlink(log_path)


def test_get_logs_file_not_exists():
    """Test get_logs when log file doesn't exist."""
    manager = ServerManager(server_path="/fake/path", port=8080)
    manager.log_file = Path("/nonexistent/file.log")

    logs = manager.get_logs()
    assert logs == []

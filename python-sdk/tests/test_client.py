"""Tests for Agentforge client."""

import json
from unittest.mock import MagicMock, patch

import pytest
import requests

from agentforge_sdk.client import AgentforgeClient
from agentforge_sdk.exceptions import AgentNotFoundError, APIError, ServerNotRunningError
from agentforge_sdk.models import ChunkResponse


def test_client_initialization():
    """Test client initialization."""
    client = AgentforgeClient(base_url="http://localhost:8080", auto_start=False)
    assert client.base_url == "http://localhost:8080"
    assert client.auto_start is False


def test_list_agents_success():
    """Test successful agent listing."""
    client = AgentforgeClient(auto_start=False)

    mock_response = MagicMock()
    mock_response.json.return_value = {"agents": ["agent1", "agent2"]}
    mock_response.status_code = 200

    with patch("requests.get", return_value=mock_response):
        agents = client.list_agents()
        assert agents == ["agent1", "agent2"]


def test_list_agents_error():
    """Test agent listing with error."""
    client = AgentforgeClient(auto_start=False)

    with patch("requests.get", side_effect=requests.RequestException("Connection error")):
        with pytest.raises(APIError):
            client.list_agents()


def test_chat_success():
    """Test successful chat."""
    client = AgentforgeClient(auto_start=False)

    # Mock streaming response
    mock_response = MagicMock()
    mock_response.status_code = 200
    mock_response.iter_lines.return_value = [
        json.dumps({"content": "Hello", "status": "streaming", "type": "content"}).encode(),
        json.dumps({"content": " World", "status": "streaming", "type": "content"}).encode(),
        json.dumps({"fullContent": "Hello World", "status": "completed", "type": "completion"}).encode(),
    ]

    with patch("requests.post", return_value=mock_response):
        chunks = list(client.chat("test-agent", "Hello"))
        assert len(chunks) == 3
        assert chunks[0].content == "Hello"
        assert chunks[1].content == " World"
        assert chunks[2].status == "completed"


def test_chat_agent_not_found():
    """Test chat with non-existent agent."""
    client = AgentforgeClient(auto_start=False)

    mock_response = MagicMock()
    mock_response.status_code = 404

    with patch("requests.post", return_value=mock_response):
        with pytest.raises(AgentNotFoundError) as exc_info:
            list(client.chat("nonexistent", "Hello"))
        assert exc_info.value.agent_name == "nonexistent"


def test_chat_api_error():
    """Test chat with API error."""
    client = AgentforgeClient(auto_start=False)

    with patch("requests.post", side_effect=requests.RequestException("Connection error")):
        with pytest.raises(APIError):
            list(client.chat("test-agent", "Hello"))


def test_ensure_server_running_with_manager():
    """Test _ensure_server_running with server manager."""
    client = AgentforgeClient(auto_start=False)

    with patch.object(client.server_manager, "is_running", return_value=False):
        with patch.object(client.server_manager, "start", return_value=True):
            # Should not raise when auto_start is False but manager exists
            # (it will try to start if not running)
            pass


def test_ensure_server_running_without_manager():
    """Test _ensure_server_running without server manager."""
    client = AgentforgeClient(auto_start=False)
    client.server_manager = None

    with patch("requests.get", side_effect=requests.RequestException("Connection error")):
        with pytest.raises(ServerNotRunningError):
            client.list_agents()

"""Data models for Agentforge SDK.

These models match the Go structures in src/core/response.go and src/llms/models.go.
"""

from dataclasses import dataclass, field
from typing import Any, Dict, List, Optional


@dataclass
class ToolCall:
    """Represents a tool call request from the LLM.

    Matches Go struct: src/llms/models.go ToolCall
    """

    id: str  # Tool call ID
    name: str  # Tool name
    arguments: Dict[str, Any]  # Tool arguments


@dataclass
class ToolResult:
    """Represents the result of a tool execution.

    Matches Go struct: src/llms/models.go ToolResult
    """

    tool_call_id: str  # ID of the tool call this result is for
    tool_name: str  # Name of the tool that was executed
    success: bool  # Whether the tool executed successfully
    result: str  # Result data from the tool
    error: str = ""  # Error message if tool failed
    ephemeral: bool = False  # Whether the result is ephemeral


@dataclass
class ChunkResponse:
    """Represents a streaming response chunk.

    This matches ExtendedChunkResponse from src/core/response.go which extends
    ChunkResponse with agentName and trace fields.
    """

    content: str = ""  # Current chunk content
    delta: str = ""  # Incremental delta
    full_content: str = ""  # Accumulated full content
    status: str = ""  # Status: see constants.py Status* constants
    type: str = ""  # Response type: see constants.py Type* constants
    tool_calls: Optional[List[ToolCall]] = None  # Tool calls (when Type is "tool-call")
    tool_executing: Optional[ToolCall] = None  # Tool being executed (when Status is "tool-executing")
    tool_results: Optional[List[ToolResult]] = None  # Tool execution results (when Status is "tool-result")
    prompt_tokens: Optional[int] = None  # Input tokens consumed
    completion_tokens: Optional[int] = None  # Output tokens generated
    total_tokens: Optional[int] = None  # Total tokens used
    agent_name: str = ""  # Name of the agent producing this chunk
    trace: str = ""  # Trace information (e.g., "thinking", "response")

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "ChunkResponse":
        """Create ChunkResponse from dictionary (e.g., from JSON)."""
        # Handle tool_calls
        tool_calls = None
        if "toolCalls" in data and data["toolCalls"]:
            tool_calls = [ToolCall(**tc) for tc in data["toolCalls"]]

        # Handle tool_executing
        tool_executing = None
        if "toolExecuting" in data and data["toolExecuting"]:
            tool_executing = ToolCall(**data["toolExecuting"])

        # Handle tool_results
        tool_results = None
        if "toolResults" in data and data["toolResults"]:
            tool_results = []
            for tr in data["toolResults"]:
                # Convert toolCallId to tool_call_id
                tr_dict = {
                    "tool_call_id": tr.get("toolCallId", ""),
                    "tool_name": tr.get("toolName", ""),
                    "success": tr.get("success", False),
                    "result": tr.get("result", ""),
                    "error": tr.get("error", ""),
                    "ephemeral": tr.get("ephemeral", False),
                }
                tool_results.append(ToolResult(**tr_dict))

        return cls(
            content=data.get("content", ""),
            delta=data.get("delta", ""),
            full_content=data.get("fullContent", ""),
            status=data.get("status", ""),
            type=data.get("type", ""),
            tool_calls=tool_calls,
            tool_executing=tool_executing,
            tool_results=tool_results,
            prompt_tokens=data.get("promptTokens"),
            completion_tokens=data.get("completionTokens"),
            total_tokens=data.get("totalTokens"),
            agent_name=data.get("agentName", ""),
            trace=data.get("trace", ""),
        )

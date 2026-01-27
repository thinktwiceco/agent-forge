"""Constants for Agent Forge SDK.

These constants match the Go implementation in src/llms/constants.go.
"""

# ChunkResponse Status Constants
#
# These constants define the possible status values for ChunkResponse.Status field.
# The Status field indicates the current state of the streaming response.

# StatusStreaming indicates that content is actively being streamed from the LLM.
# This status is used during the content generation phase when the LLM is producing text.
StatusStreaming = "streaming"

# StatusCompleted indicates that the streaming response has finished successfully.
# This is the final status sent when the LLM has completed generating its response
# and no further chunks will be sent (unless tool calls require another iteration).
StatusCompleted = "completed"

# StatusError indicates that an error occurred during the streaming process.
# This status is used to signal failures in deserialization, network issues,
# or other errors that prevent normal response processing.
StatusError = "error"

# StatusToolCall indicates that the LLM is requesting one or more tool executions.
# This status signals that the model wants to use external tools to gather information
# or perform actions before continuing its response.
StatusToolCall = "tool-call"

# StatusToolExecuting indicates that a specific tool is currently being executed.
# This status is emitted by the agent to provide real-time feedback about
# which tool is running, allowing consumers to track execution progress.
StatusToolExecuting = "tool-executing"

# StatusToolResult indicates that tool execution has completed and results are available.
# This status is emitted after each tool finishes executing, containing the
# success/failure status and the tool's output or error message.
StatusToolResult = "tool-result"

# ChunkResponse Type Constants
#
# These constants define the possible type values for ChunkResponse.Type field.
# The Type field categorizes the kind of response chunk being sent.

# TypeContent indicates a chunk containing regular content being streamed from the LLM.
# This is the most common type during normal text generation.
TypeContent = "content"

# TypeToolCall indicates a chunk containing tool call requests from the LLM.
# The LLM has decided to use external tools and is providing the tool names
# and arguments needed to execute them.
TypeToolCall = "tool-call"

# TypeToolExecuting indicates a chunk signaling that a tool is currently being executed.
# This type is used for progress tracking and allows consumers to monitor
# which tools are running in real-time.
TypeToolExecuting = "tool-executing"

# TypeToolResult indicates a chunk containing the results of a tool execution.
# This includes both successful results and error information if the tool failed.
TypeToolResult = "tool-result"

# TypeCompletion indicates the final chunk signaling that the response is complete.
# This type marks the end of the streaming response when no more data will be sent.
TypeCompletion = "completion"


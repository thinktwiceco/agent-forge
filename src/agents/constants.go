package agents

// System-defined agent names
const (
	// AgentNameSystemReasoning is the name of the system reasoning agent
	AgentNameSystemReasoning = "system-reasoning"
	// AgentNameSystemOS is the name of the system OS agent
	AgentNameSystemOS = "system-os"
	// AgentNameSystemCoding is the name of the system coding agent
	AgentNameSystemCoding = "system-coding"
	// AgentNameSystemVector is the name of the system vector agent
	AgentNameSystemVector = "system-vector"
	// SystemAgentPrefix is the prefix used for all system-defined agents
	SystemAgentPrefix = "system-"
)

// System-defined trace values
const (
	// TraceReasoning is the trace identifier for reasoning agent operations
	TraceReasoning = "reasoning"
	// TraceOS is the trace identifier for OS agent operations
	TraceOS = "os"
	// TraceCoding is the trace identifier for coding agent operations
	TraceCoding = "coding"
	// TraceVector is the trace identifier for vector agent operations
	TraceVector = "vector"
	// TraceResponse is the trace identifier for main agent responses
	TraceResponse = "response"
	// TraceThinking is a conceptual trace identifier for thinking/reasoning processes
	// This is used for pattern matching in formatting logic
	TraceThinking = "thinking"
)

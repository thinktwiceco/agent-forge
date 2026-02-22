package system

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
	// AgentNameSystemGit is the name of the system git agent
	AgentNameSystemGit = "system-git"
	// AgentNameSystemWeb is the name of the system web agent
	AgentNameSystemWeb = "system-web"
	// AgentNameSystemVision is the name of the system vision agent
	AgentNameSystemVision = "system-vision"
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
	// TraceGit is the trace identifier for git agent operations
	TraceGit = "git"
	// TraceWeb is the trace identifier for web agent operations
	TraceWeb = "web"
	// TraceVision is the trace identifier for vision agent operations
	TraceVision = "vision"
)

// ToneSystemAgent instructs the agent to provide detailed responses for system agents
const ToneSystemAgent = "system-agent"

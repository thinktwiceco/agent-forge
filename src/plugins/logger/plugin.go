package logger

import (
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/thinktwice/agentForge/src/agents"
	"github.com/thinktwice/agentForge/src/core"
	"github.com/thinktwice/agentForge/src/llms"
)

const (
	// ANSI color codes
	ColorReset   = "\033[0m"
	ColorCyan    = "\033[36m"
	ColorYellow  = "\033[33m"
	ColorRed     = "\033[31m"
	ColorGreen   = "\033[32m"
	ColorBlue    = "\033[34m"
	ColorMagenta = "\033[35m"
	ColorBold    = "\033[1m"
	ColorDim     = "\033[2m"
)

// LoggerPlugin provides configurable formatting for chunks based on agent name and trace patterns
type LoggerPlugin struct {
	colorRules            []ColorRule
	labelRules            []LabelRule
	output                io.Writer
	currentAgent          string
	currentTrace          string
	currentChunkAgentName string     // Agent name from current chunk being processed
	currentChunkTrace     string     // Trace from current chunk being processed
	mu                    sync.Mutex // Protects all fields
}

// NewPlugin creates a new logger plugin with the specified color and label rules
func NewPlugin(colorRules []ColorRule, labelRules []LabelRule, output io.Writer) *LoggerPlugin {
	return &LoggerPlugin{
		colorRules:   colorRules,
		labelRules:   labelRules,
		output:       output,
		currentAgent: "",
		currentTrace: "",
	}
}

// Name implements the core.Plugin interface
func (p *LoggerPlugin) Name() string {
	return "logger"
}

// On implements the core.Plugin interface
func (p *LoggerPlugin) On(event core.Event) core.AgentHookFn {
	switch event {
	case core.EventNewChunk:
		fn := agents.OnNewChunkHook(p.handleNewChunk)
		return fn
	}
	return nil
}

// Tools implements the core.Plugin interface
func (p *LoggerPlugin) Tools() []llms.Tool {
	return []llms.Tool{} // Logger plugin doesn't provide tools
}

// handleNewChunk is called when a new chunk is created
// This hook formats and prints chunks based on the plugin's rules
func (p *LoggerPlugin) handleNewChunk(a *agents.Agent, chunk *llms.ChunkResponse) error {
	if p.output == nil {
		return nil // No output writer configured, skip formatting
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Use stored agent name/trace from current chunk if available, otherwise use agent's name/trace
	agentName := p.currentChunkAgentName
	trace := p.currentChunkTrace
	if agentName == "" {
		agentName = a.Name()
	}
	if trace == "" {
		trace = a.Trace()
	}

	// Determine color based on agent name/trace
	color := getColorWithRules(agentName, trace, p.colorRules)

	// Print agent header if agent changed
	if agentName != p.currentAgent || trace != p.currentTrace {
		p.currentAgent = agentName
		p.currentTrace = trace

		// Print agent name header
		agentLabel := formatLabelWithRules(agentName, trace, p.labelRules)
		fmt.Fprintf(p.output, "\n%s%s%s%s\n", ColorBold, color, agentLabel, ColorReset)
	}

	// Handle different chunk types
	switch chunk.Type {
	case llms.TypeContent:
		// Stream content as it arrives
		if chunk.Content != "" || chunk.Delta != "" {
			content := chunk.Content
			if content == "" {
				content = chunk.Delta
			}
			fmt.Fprintf(p.output, "%s%s%s", color, content, ColorReset)
		}

	case llms.TypeCompletion:
		// Final completion - display token usage if available
		if chunk.TotalTokens > 0 {
			fmt.Fprintf(p.output, "\n%s%s📊 Tokens: %d prompt + %d completion = %d total%s\n",
				ColorBlue, ColorDim,
				chunk.PromptTokens, chunk.CompletionTokens, chunk.TotalTokens,
				ColorReset)
		}

	case llms.TypeToolExecuting:
		// Show tool execution (suppress for delegate tool when delegating to sub-agents)
		if chunk.ToolExecuting != nil && chunk.ToolExecuting.Name != "delegate" {
			fmt.Fprintf(p.output, "\n%s%s⚙️  Executing tool: %s%s\n", ColorMagenta, ColorBold, chunk.ToolExecuting.Name, ColorReset)
		}

	case llms.TypeToolResult:
		// Show tool results (suppress for delegate tool when delegating to sub-agents)
		if len(chunk.ToolResults) > 0 {
			for _, result := range chunk.ToolResults {
				if result.ToolName == "delegate" {
					// Skip verbose delegate tool completion messages for sub-agents
					continue
				}
				if result.Success {
					fmt.Fprintf(p.output, "%s%s✓ Tool completed: %s%s\n", ColorGreen, ColorBold, result.ToolName, ColorReset)
				} else {
					fmt.Fprintf(p.output, "%s%s✗ Tool failed: %s - %s%s\n", ColorRed, ColorBold, result.ToolName, result.Error, ColorReset)
				}
			}
		}
	}

	// Flush output if it's a file or buffered writer
	if flusher, ok := p.output.(interface{ Flush() error }); ok {
		flusher.Flush()
	}
	// Note: os.Stdout and os.Stderr are *os.File and will auto-flush on newline
	// But we can force sync if needed
	if file, ok := p.output.(*os.File); ok && (file == os.Stdout || file == os.Stderr) {
		file.Sync()
	}

	// Clear chunk-specific state after processing to prevent stale data
	p.currentChunkAgentName = ""
	p.currentChunkTrace = ""

	return nil
}

// ResetState resets the internal state tracking (useful for new conversations)
func (p *LoggerPlugin) ResetState() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.currentAgent = ""
	p.currentTrace = ""
	p.currentChunkAgentName = ""
	p.currentChunkTrace = ""
}

// SetCurrentChunkInfo stores the agent name and trace for the current chunk being processed
// This is called before the hook processes the chunk
func (p *LoggerPlugin) SetCurrentChunkInfo(agentName, trace string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.currentChunkAgentName = agentName
	p.currentChunkTrace = trace
}

// GetColorForAgent returns the color code for the given agent name and trace
// This is a convenience method that uses the plugin's color rules
func (p *LoggerPlugin) GetColorForAgent(agentName, trace string) string {
	return getColorWithRules(agentName, trace, p.colorRules)
}

// FormatAgentLabel returns a formatted label for the given agent name and trace
// This is a convenience method that uses the plugin's label rules
func (p *LoggerPlugin) FormatAgentLabel(agentName, trace string) string {
	return formatLabelWithRules(agentName, trace, p.labelRules)
}

// Global instance for convenience functions (used when plugin instance is not available)
var defaultPlugin *LoggerPlugin

// init initializes the default plugin with default rules
func init() {
	defaultPlugin = NewPlugin(DefaultColorRules(), DefaultLabelRules(), nil)
}

// GetColor returns the color code for the given agent name and trace using default rules
// This is a convenience function for use in cmd/chat/main.go when plugin instance is not available
func GetColor(agentName, trace string) string {
	return defaultPlugin.GetColorForAgent(agentName, trace)
}

// FormatLabel returns a formatted label for the given agent name and trace using default rules
// This is a convenience function for use in cmd/chat/main.go when plugin instance is not available
func FormatLabel(agentName, trace string) string {
	return defaultPlugin.FormatAgentLabel(agentName, trace)
}

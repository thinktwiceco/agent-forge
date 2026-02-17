package logger

import (
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/thinktwiceco/agent-forge/src/agents"
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
	"github.com/thinktwiceco/agent-forge/src/tools/delegate"
)

const (
	// Plugin name
	PLUGIN_NAME = "logger"

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
	colorRules   []ColorRule
	labelRules   []LabelRule
	output       io.Writer
	currentAgent string
	currentTrace string
	mu           sync.Mutex // Protects all fields
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
	return PLUGIN_NAME
}

// Hooks implements the core.HookProvider interface
// Returns a map of event hooks that this plugin provides
func (p *LoggerPlugin) Hooks() map[core.Event]core.AgentHookFn {
	return map[core.Event]core.AgentHookFn{
		core.EventNewChunk: agents.OnNewChunkHook(p.handleNewChunk),
	}
}

// Note: Logger plugin does not implement ToolProvider or PromptProvider
// as it only provides event hooks for logging purposes.

// handleNewChunk is called when a new chunk is created
// This hook formats and prints chunks based on the plugin's rules
func (p *LoggerPlugin) handleNewChunk(a *agents.Agent, extendedChunk *core.ExtendedChunkResponse) error {
	if p.output == nil {
		return nil // No output writer configured, skip formatting
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Use agent name/trace from extended chunk if available, otherwise use agent's name/trace
	agentName := extendedChunk.AgentName
	trace := extendedChunk.Trace
	if agentName == "" && a != nil {
		agentName = a.Name()
	}
	if trace == "" && a != nil {
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
		_, _ = fmt.Fprintf(p.output, "\n%s%s%s%s\n", ColorBold, color, agentLabel, ColorReset)
	}

	// Handle different chunk types
	switch extendedChunk.Type {
	case llms.TypeContent:
		// Stream content as it arrives
		if extendedChunk.Content != "" || extendedChunk.Delta != "" {
			content := extendedChunk.Content
			if content == "" {
				content = extendedChunk.Delta
			}
			_, _ = fmt.Fprintf(p.output, "%s%s%s", color, content, ColorReset)
		}

	case llms.TypeCompletion:
		// Final completion - display token usage if available
		if extendedChunk.TotalTokens > 0 {
			_, _ = fmt.Fprintf(p.output, "\n%s%s📊 Tokens: %d prompt + %d completion = %d total%s\n",
				ColorBlue, ColorDim,
				extendedChunk.PromptTokens, extendedChunk.CompletionTokens, extendedChunk.TotalTokens,
				ColorReset)
		}

	case llms.TypeToolExecuting:
		// Show tool execution (suppress for delegate tool when delegating to sub-agents)
		if extendedChunk.ToolExecuting != nil && extendedChunk.ToolExecuting.Name != delegate.DELEGATE_TOOL {
			_, _ = fmt.Fprintf(p.output, "\n%s%s⚙️  Executing tool: %s%s\n", ColorMagenta, ColorBold, extendedChunk.ToolExecuting.Name, ColorReset)
		}

	case llms.TypeToolResult:
		// Show tool results (suppress for delegate tool when delegating to sub-agents)
		if len(extendedChunk.ToolResults) > 0 {
			for _, result := range extendedChunk.ToolResults {
				if result.ToolName == delegate.DELEGATE_TOOL {
					// Skip verbose delegate tool completion messages for sub-agents
					continue
				}
				if result.Success {
					_, _ = fmt.Fprintf(p.output, "%s%s✓ Tool completed: %s%s\n", ColorGreen, ColorBold, result.ToolName, ColorReset)
				} else {
					_, _ = fmt.Fprintf(p.output, "%s%s✗ Tool failed: %s - %s%s\n", ColorRed, ColorBold, result.ToolName, result.Error, ColorReset)
				}
			}
		}
	}

	// Flush output if it's a file or buffered writer
	if flusher, ok := p.output.(interface{ Flush() error }); ok {
		_ = flusher.Flush()
	}
	// Note: os.Stdout and os.Stderr are *os.File and will auto-flush on newline
	// But we can force sync if needed
	if file, ok := p.output.(*os.File); ok && (file == os.Stdout || file == os.Stderr) {
		_ = file.Sync() // Ignore sync errors for stdout/stderr as they're typically non-critical
	}

	return nil
}

// ResetState resets the internal state tracking (useful for new conversations)
func (p *LoggerPlugin) ResetState() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.currentAgent = ""
	p.currentTrace = ""
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

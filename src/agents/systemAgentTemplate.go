package agents

import (
	"fmt"
	"strings"

	"github.com/thinktwice/agentForge/src/llms"
)

// SystemAgentTemplate defines a template for creating system agents.
//
// This struct formalizes the structure of system agents by defining their
// core components: name, trace identifier, system prompt, and description.
// It provides a structured way to build system prompts and descriptions
// from their constituent parts.
type SystemAgentTemplate struct {
	// Name is the unique identifier for the system agent (e.g., "system-reasoning")
	Name string

	// Trace is the trace identifier used for tracking agent operations (e.g., "reasoning")
	Trace string

	// systemPrompt contains the complete behavioral instructions for the agent.
	// Built using AddSystemPrompt method.
	systemPrompt string

	// description contains usage instructions that explain when and how to
	// delegate tasks to this agent. Built using AddDescription method.
	description string

	// advanceDescription contains detailed information about the agent's
	// capabilities, behavior, and advanced usage patterns.
	advanceDescription string

	// troubleshooting contains debugging tips, common issues, and
	// configuration guidance for this agent.
	troubleshooting string
}

// NewSystemAgentTemplate creates a new SystemAgentTemplate with validation.
//
// Parameters:
//   - name: Unique identifier for the system agent
//   - trace: Trace identifier for tracking
//
// Returns:
//   - *SystemAgentTemplate: A new template instance
//   - error: Validation error if any required field is empty
func NewSystemAgentTemplate(name, trace string) (*SystemAgentTemplate, error) {
	if name == "" {
		return nil, fmt.Errorf("name is required but was empty")
	}
	if trace == "" {
		return nil, fmt.Errorf("trace is required but was empty")
	}

	return &SystemAgentTemplate{
		Name:  name,
		Trace: trace,
	}, nil
}

// AddSystemPrompt builds the system prompt from structured components.
//
// This method constructs the system prompt by assembling the provided
// components in a consistent format.
//
// Parameters:
//   - incipit: Introduction text explaining the agent's purpose
//   - steps: List of step descriptions for the agent's process
//   - output: Output format specification
//   - examples: List of example interactions
//   - critical: List of critical rules and constraints
//
// Returns:
//   - *SystemAgentTemplate: Returns self for method chaining
func (t *SystemAgentTemplate) AddSystemPrompt(incipit string, steps []string, output string, examples []string, critical []string) *SystemAgentTemplate {
	var builder strings.Builder

	// Add incipit
	if incipit != "" {
		builder.WriteString(incipit)
		builder.WriteString("\n\n")
	}

	// Add steps
	if len(steps) > 0 {
		builder.WriteString("STEPS:\n")
		for i, step := range steps {
			builder.WriteString(fmt.Sprintf("- Step %d: %s\n", i+1, step))
		}
		builder.WriteString("\n")
	}

	// Add output format
	if output != "" {
		builder.WriteString("[OUTPUT]\n")
		builder.WriteString(output)
		builder.WriteString("\n\n")
	}

	// Add examples
	if len(examples) > 0 {
		builder.WriteString("[EXAMPLES]\n")
		for _, example := range examples {
			builder.WriteString(example)
			builder.WriteString("\n\n")
		}
	}

	// Add critical rules
	if len(critical) > 0 {
		builder.WriteString("[CRITICAL]\n")
		for _, rule := range critical {
			builder.WriteString(rule)
			builder.WriteString("\n")
		}
	}

	t.systemPrompt = builder.String()
	return t
}

// AddDescription builds the description from structured components.
//
// This method constructs the agent description that explains when and
// how to use the agent.
//
// Parameters:
//   - incipit: Introduction text explaining the agent's usage
//   - examples: List of usage examples (dos and don'ts)
//
// Returns:
//   - *SystemAgentTemplate: Returns self for method chaining
func (t *SystemAgentTemplate) AddDescription(incipit string, examples []string) *SystemAgentTemplate {
	var builder strings.Builder

	// Add incipit
	if incipit != "" {
		builder.WriteString(incipit)
		builder.WriteString("\n")
	}

	// Add examples
	if len(examples) > 0 {
		builder.WriteString("[EXAMPLES]\n")
		for _, example := range examples {
			builder.WriteString(example)
			builder.WriteString("\n")
		}
	}

	t.description = builder.String()
	return t
}

// SystemPrompt returns the built system prompt.
func (t *SystemAgentTemplate) SystemPrompt() string {
	return t.systemPrompt
}

// Description returns the built description.
func (t *SystemAgentTemplate) Description() string {
	return t.description
}

// AddAdvanceDescription sets the advanced description for the agent.
//
// This method sets detailed information about the agent's capabilities,
// behavior, and advanced usage patterns.
//
// Parameters:
//   - advanceDescription: Detailed agent capabilities and usage information
//
// Returns:
//   - *SystemAgentTemplate: Returns self for method chaining
func (t *SystemAgentTemplate) AddAdvanceDescription(advanceDescription string) *SystemAgentTemplate {
	t.advanceDescription = advanceDescription
	return t
}

// AddTroubleshooting sets the troubleshooting information for the agent.
//
// This method sets debugging tips, common issues, and configuration
// guidance for the agent.
//
// Parameters:
//   - troubleshooting: Troubleshooting and debugging information
//
// Returns:
//   - *SystemAgentTemplate: Returns self for method chaining
func (t *SystemAgentTemplate) AddTroubleshooting(troubleshooting string) *SystemAgentTemplate {
	t.troubleshooting = troubleshooting
	return t
}

// AdvanceDescription returns the built advance description.
func (t *SystemAgentTemplate) AdvanceDescription() string {
	return t.advanceDescription
}

// Troubleshooting returns the built troubleshooting information.
func (t *SystemAgentTemplate) Troubleshooting() string {
	return t.troubleshooting
}

// ToAgentConfig converts the template to an AgentConfig ready for agent creation.
//
// This method creates an AgentConfig populated with the template's fields,
// allowing easy instantiation of an agent from the template.
//
// Parameters:
//   - llmEngine: The LLM engine to use for this agent
//
// Returns:
//   - AgentConfig: Configuration ready to pass to NewAgent()
func (t *SystemAgentTemplate) ToAgentConfig(llmEngine llms.LLMEngine) AgentConfig {
	return AgentConfig{
		LLMEngine:          llmEngine,
		AgentName:          t.Name,
		Trace:              t.Trace,
		SystemPrompt:       t.systemPrompt,
		Description:        t.description,
		AdvanceDescription: t.advanceDescription,
		Troubleshooting:    t.troubleshooting,
		MainAgent:          false,
	}
}

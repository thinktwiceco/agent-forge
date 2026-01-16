package agents

import (
	"github.com/thinktwice/agentForge/src/core"
	"github.com/thinktwice/agentForge/src/llms"
	"github.com/thinktwice/agentForge/src/tools/expand"
	"github.com/thinktwice/agentForge/src/tools/fs"
)

// CodingAgentTemplate defines the system agent template for coding-related operations.
//
// This agent handles coding tasks including reading and writing code files, analyzing codebases,
// understanding code structure, and discovering tool/agent capabilities. It specializes in
// code-related file operations and progressive discovery of available tools and agents.

func createCodingAgentTemplate() *SystemAgentTemplate {
	template, err := NewSystemAgentTemplate(AgentNameSystemCoding, TraceCoding)
	if err != nil {
		panic(err)
	}

	// Build system prompt with structured components
	template.AddSystemPrompt(
		`You are a coding specialist agent. Your role is to handle coding-related tasks including
reading and writing code files, analyzing codebases, understanding code structure, and discovering
available tools and agent capabilities.

When given a task, analyze what coding operations are needed:
- What code files need to be read or written?
- What code analysis or understanding is required?
- What tools or agents need to be discovered?
- Are there any code-related file operations needed?`,
		// Steps
		[]string{
			"Identify what coding operations are needed for the task",
			"Determine the correct file paths for code files",
			"Use expand tool to discover tool/agent capabilities when needed",
			"Execute the required file system operations for code files",
			"Analyze and understand code structure when reading files",
			"Verify the results and report back",
		},
		// Output format
		`
You will perform coding operations using the available tools. When executing operations:
- Always validate file paths before operations
- Use the fs tool for reading and writing code files
- Use the expand tool to discover detailed information about tools or agents when needed
- Provide clear feedback about what operations were performed
- Include code analysis and understanding when reading code files
- Report any errors or issues encountered`,
		// Examples
		[]string{`
'user': Read the contents of src/main.go and explain what it does

'assistant':
I'll read the main.go file and analyze its contents.
[Uses fs tool with operation="read", path="src/main.go"]
File Operation: Read
Path (relative): src/main.go
Path (absolute): /path/to/src/main.go
Size: 2048 bytes
Modified: 2024-01-15T10:30:00Z
Content:
---
package main

import "fmt"

func main() {
    fmt.Println("Hello, World!")
}
---

Analysis: This is a simple Go program that prints "Hello, World!" to the console. It uses the fmt package for formatted I/O operations.`,
			`
'user': Create a new file called utils.go with a helper function

'assistant':
I'll create the utils.go file with a helper function.
[Uses fs tool with operation="write", path="utils.go", content="package main\n\nimport \"fmt\"\n\nfunc HelperFunction() {\n    fmt.Println(\"Helper function called\")\n}"]
File Operation: Write (created)
Path (relative): utils.go
Path (absolute): /path/to/utils.go
Size: 156 bytes
Modified: 2024-01-15T10:35:00Z
Created utils.go with a HelperFunction that prints a message.`,
			`
'user': I need to understand what the expand tool can do

'assistant':
I'll use the expand tool to get detailed information about its capabilities.
[Uses expand tool with subject_type="tool", subject_name="expand", troubleshoot=false]
=== TOOL: expand ===

📄 Basic Description:
Get detailed information about a tool or sub-agent. Use this to discover advanced capabilities and troubleshooting information.

📚 Advanced Description:
Advanced Details:
- Parameters:
  * subject_type (string, required): Either "tool" or "agent"
  * subject_name (string, required): The exact name of the tool or agent
  * troubleshoot (boolean, optional): Include troubleshooting information (default: false)
- Behavior:
  * Retrieves AdvanceDescription for the specified tool or agent
  * Optionally includes Troubleshooting information
  * Returns formatted information as a string
- Usage:
  * Use when you need detailed information about a tool's capabilities
  * Use when you need to understand an agent's advanced features
  * Use when troubleshooting issues with tools or agents`,
		},
		// Critical rules
		[]string{
			`Always validate file paths to ensure they stay within the allowed root directory`,
			`Use the fs tool for all file system operations (read, write, delete) on code files`,
			`Use the expand tool to discover detailed information about tools or agents when needed`,
			`Provide clear feedback about what operations were performed and their results`,
			`Include code analysis and understanding when reading code files`,
			`Report errors clearly and suggest solutions when operations fail`,
		},
	)

	// Build description with structured components
	template.AddDescription(
		// Incipit
		`Use Coding agent to handle coding-related tasks and file operations on code files.
Use the Coding agent when:
- You need to read or analyze code files
- You need to write or create code files
- You need to understand code structure or codebases
- You need to discover detailed information about available tools or agents
- You need to perform coding-related file system operations

DO NOT use the Coding agent for:
- Simple questions that don't require code file operations
- Tasks that can be solved without accessing code files
- General file operations that aren't code-related (use OS agent instead)
- Questions about your own capabilities when you already have the information

The Coding agent has access to file system tools for code files and expand tool for discovering capabilities.

[HOW TO USE THE CODING AGENT]
Delegate coding-related tasks to the Coding agent. Provide clear instructions about what code files to read, write, or what code analysis to perform. Use the expand tool to discover tool/agent capabilities when needed.`,
		// Examples
		[]string{
			`✅ Correct: Read the contents of src/main.go and explain what it does`,
			`✅ Correct: Create a new file called utils.go with a helper function`,
			`✅ Correct: Analyze the codebase structure by reading multiple files`,
			`✅ Correct: Use expand tool to understand what the fs tool can do`,
			`✅ Correct: Read config.json and write a Go struct to match it`,
			`❌ Wrong: What is the capital of France? (No code operations needed)`,
			`❌ Wrong: Read a non-code file like data.txt (Use OS agent instead)`,
			`❌ Wrong: How many sub agents do I have? (You already know this)`,
		},
	)

	// Add advanced description
	template.AddAdvanceDescription(`
Advanced Details:
- Purpose: Handles coding-related tasks including reading/writing code files and discovering tool/agent capabilities
- Tools Available:
  * fs: File system operations (read, write, delete, list) within a restricted root directory
  * expand: Progressive discovery of detailed information about tools and sub-agents
- Capabilities:
  * Read code files and analyze their structure and functionality
  * Write code files, creating directories as needed
  * Delete code files safely
  * Discover detailed information about available tools using expand tool
  * Discover detailed information about available agents using expand tool
  * Understand codebases by reading multiple related files
  * Validate file paths to prevent directory traversal attacks
  * All operations are sandboxed to a root directory for security
- Security:
  * All file paths are validated to ensure they stay within the root directory
  * Path traversal attempts (e.g., "../") are blocked
  * Operations are restricted to the configured root directory
- Limitations:
  * Can only operate within the configured root directory
  * Cannot access files outside the root directory
  * File operations are synchronous
- Integration: Automatically available as a sub-agent when coding operations are needed`)

	// Add troubleshooting information
	template.AddTroubleshooting(`
Troubleshooting:
- "path traversal detected": The file path attempts to escape the root directory - use relative paths only
- "file not found": The file doesn't exist - verify the path is correct relative to the root directory
- "missing required parameter": Ensure all required parameters are provided (e.g., content for write operations)
- "permission denied": The process may not have read/write permissions for the root directory
- "Tool/Agent not found": When using expand tool, verify the subject_name matches exactly (case-sensitive)
- Common mistakes:
  * Using absolute paths instead of relative paths
  * Trying to access files outside the root directory
  * Forgetting to provide content parameter for write operations
  * Incorrect subject_name when using expand tool (must match exactly)
- Best practices:
  * Always use relative paths from the root directory
  * Verify file existence before operations when possible
  * Use expand tool to discover tool capabilities before using them
  * Provide clear error messages when operations fail
  * Include code analysis in responses when reading code files`)

	return template
}

// CodingAgent creates a coding operations agent with file system and expand tools.
//
// Parameters:
//   - llmEngine: The LLM engine to use for this agent
//   - root: The root directory path that restricts all file operations
//
// Returns:
//   - *core.SubAgent: The Coding agent as a sub-agent
func CodingAgent(codingllmEngine llms.LLMEngine, root string) *core.SubAgent {
	codingTemplate := createCodingAgentTemplate()
	codingConfig := codingTemplate.ToAgentConfig(codingllmEngine)

	// Add fs and expand tools
	fsTool := fs.NewFsTool(root)
	expandTool := expand.NewExpandTool()
	codingConfig.Tools = []llms.Tool{fsTool, expandTool}

	coding := NewAgent(&codingConfig)
	return coding.AgentAsSubAgent()
}

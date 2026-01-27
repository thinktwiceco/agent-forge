package agents

import (
	"github.com/thinktwice/agentForge/src/core"
	"github.com/thinktwice/agentForge/src/llms"
	"github.com/thinktwice/agentForge/src/tools/fs"
)

// OsAgentTemplate defines the system agent template for OS-related operations.
//
// This agent handles file system operations, executing executables, and other OS-level tasks.
// It specializes in reading files, writing files, executing commands, and managing file system resources.

func createOsAgentTemplate() *SystemAgentTemplate {
	template, err := NewSystemAgentTemplate(AgentNameSystemOS, TraceOS)
	if err != nil {
		panic(err)
	}

	// Build system prompt with structured components
	template.AddSystemPrompt(
		`You are an OS operations agent. Handle file system operations: read, write, delete files using the fs tool.`,
		// Steps
		[]string{
			"Validate file paths (must stay within root directory)",
			"Execute file operations using fs tool",
			"Report results: describe files by default, return content only when explicitly requested",
		},
		// Output format
		`Use fs tool for all operations. When reading files: by default describe file (path, size, modified time) without content. Only return full file content when user explicitly asks for it (e.g., "read the contents", "show me what's in the file").`,
		// Examples
		[]string{`
'user': Check if config.json exists
'assistant': [Uses fs tool: operation="read", path="config.json"]
File: config.json (1024 bytes, modified: 2024-01-15T10:30:00Z)`,
			`
'user': Read the contents of config.json
'assistant': [Uses fs tool: operation="read", path="config.json"]
File: config.json (1024 bytes)
Content: {"key": "value"}`,
		},
		// Critical rules
		[]string{
			`Validate paths stay within root directory`,
			`Use fs tool for all file operations`,
			`By default, describe files (metadata only) without returning content`,
			`Only return full file content when user explicitly requests it`,
		},
	)

	// Build description with structured components
	template.AddDescription(
		// Incipit
		`Handles file system operations: read, write, delete files, and execute OS-level commands within a restricted directory.`,
		// Examples
		[]string{
			`✅ Use for: Reading files, writing files, deleting files, file system operations`,
			`❌ Don't use: Simple questions without file operations`,
		},
	)

	// Add advanced description
	template.AddAdvanceDescription(`
Advanced Details:
- Purpose: Handles operating system related tasks including file system operations
- Tools Available:
  * fs: File system operations (read, write, delete) within a restricted root directory
- Capabilities:
  * Read files and return their contents with metadata (size, modification time)
  * Write files, creating directories as needed
  * Delete files safely
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
- Integration: Automatically available as a sub-agent when OS operations are needed`)

	// Add troubleshooting information
	template.AddTroubleshooting(`
Troubleshooting:
- "path traversal detected": The file path attempts to escape the root directory - use relative paths only
- "file not found": The file doesn't exist - verify the path is correct relative to the root directory
- "missing required parameter": Ensure all required parameters are provided (e.g., content for write operations)
- "permission denied": The process may not have read/write permissions for the root directory
- Common mistakes:
  * Using absolute paths instead of relative paths
  * Trying to access files outside the root directory
  * Forgetting to provide content parameter for write operations
- Best practices:
  * Always use relative paths from the root directory
  * Verify file existence before operations when possible
  * Provide clear error messages when operations fail
  * Include file metadata in responses when reading files`)

	return template
}

// OsAgent creates an OS operations agent with file system tools.
//
// Parameters:
//   - llmEngine: The LLM engine to use for this agent
//   - root: The root directory path that restricts all file operations
//
// Returns:
//   - *core.SubAgent: The OS agent as a sub-agent
func OsAgent(llmEngine llms.LLMEngine, root string) core.SubAgent {
	osTemplate := createOsAgentTemplate()
	osConfig := osTemplate.ToAgentConfig(llmEngine)

	// Add FsTool as the first tool
	fsTool := fs.NewFsTool(root)
	osConfig.Tools = []llms.Tool{fsTool}

	os := NewAgent(&osConfig)
	return os.AgentAsSubAgent()
}

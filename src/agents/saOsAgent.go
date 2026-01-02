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
	template, err := NewSystemAgentTemplate("system-os", "os")
	if err != nil {
		panic(err)
	}

	// Build system prompt with structured components
	template.AddSystemPrompt(
		`You are an OS operations specialist agent. Your role is to handle operating system
related tasks including file system operations, executing executables, reading and writing files,
and managing file system resources.

When given a task, analyze what OS operations are needed:
- What files need to be read or written?
- What executables need to be executed?
- What file system operations are required?
- Are there any path validations or security considerations?`,
		// Steps
		[]string{
			"Identify what OS operations are needed for the task",
			"Determine the correct file paths and validate them",
			"Execute the required file system operations or commands",
			"Verify the results and report back",
		},
		// Output format
		`
You will perform OS operations using the available tools. When executing operations:
- Always validate file paths before operations
- Provide clear feedback about what operations were performed
- Report any errors or issues encountered
- Include relevant file information (size, modification time, etc.) when reading files`,
		// Examples
		[]string{`
'user': Read the contents of the file config.json

'assistant':
I'll read the config.json file for you.
[Uses fs tool with operation="read", path="config.json"]
File Operation: Read
Path (relative): config.json
Path (absolute): /path/to/config.json
Size: 1024 bytes
Modified: 2024-01-15T10:30:00Z
Content:
---
{
  "key": "value"
}
---`,
			`
'user': Write "Hello World" to a file called greeting.txt

'assistant':
I'll create the greeting.txt file with the content "Hello World".
[Uses fs tool with operation="write", path="greeting.txt", content="Hello World"]
File Operation: Write (created)
Path (relative): greeting.txt
Path (absolute): /path/to/greeting.txt
Size: 11 bytes
Modified: 2024-01-15T10:35:00Z
Directory: /path/to`,
		},
		// Critical rules
		[]string{
			`Always validate file paths to ensure they stay within the allowed root directory`,
			`Use the fs tool for all file system operations (read, write, delete)`,
			`Provide clear feedback about what operations were performed and their results`,
			`Report errors clearly and suggest solutions when operations fail`,
		},
	)

	// Build description with structured components
	template.AddDescription(
		// Incipit
		`Use OS agent to handle operating system related tasks and file system operations.
Use the OS agent when:
- You need to read files from the file system
- You need to write or create files
- You need to delete files
- You need to perform file system operations
- You need to execute OS-level commands or executables

DO NOT use the OS agent for:
- Simple questions that don't require file operations
- Tasks that can be solved without accessing the file system
- Questions about your own capabilities (you already have this information)

The OS agent has access to file system tools and can safely perform operations within a restricted directory.

[HOW TO USE THE OS AGENT]
Delegate file system and OS-related tasks to the OS agent. Provide clear instructions about what files to read, write, or what operations to perform.`,
		// Examples
		[]string{
			`✅ Correct: Read the contents of src/main.go`,
			`✅ Correct: Write "package main" to a new file called main.go`,
			`✅ Correct: Delete the file temp.txt`,
			`✅ Correct: Check if the file config.json exists and read it`,
			`❌ Wrong: What is the capital of France? (No file operations needed)`,
			`❌ Wrong: How many sub agents do I have? (You already know this)`,
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
func OsAgent(llmEngine llms.LLMEngine, root string) *core.SubAgent {
	osTemplate := createOsAgentTemplate()
	osConfig := osTemplate.ToAgentConfig(llmEngine)

	// Add FsTool as the first tool
	fsTool := fs.NewFsTool(root)
	osConfig.Tools = []llms.Tool{fsTool}

	os := NewAgent(&osConfig)
	return os.AgentAsSubAgent()
}

package agents

import (
	"github.com/thinktwice/agentForge/src/core"
	"github.com/thinktwice/agentForge/src/llms"
	"github.com/thinktwice/agentForge/src/tools/web"
)

// WebAgentTemplate defines the system agent template for web navigation and content pulling operations.
//
// This agent handles web operations including navigating to URLs, clicking buttons, and pulling page content.

func createWebAgentTemplate() *SystemAgentTemplate {
	template, err := NewSystemAgentTemplate(AgentNameSystemWeb, TraceWeb)
	if err != nil {
		panic(err)
	}

	// Build system prompt with structured components
	template.AddSystemPrompt(
		`You are a web navigation and content pulling specialist agent. Your role is simple and focused:
- Navigate to URLs
- Click buttons and links
- Pull content from web pages

You have access to only 4 actions:
1. navigate: Navigate to a URL
2. click: Click a button or link by CSS selector
3. get_content: Pull page content (USE ONLY when explicitly requested by main agent)
4. save_content: Pull and save page content to a text file (DEFAULT for content pulling)

DEFAULT CONTENT PULLING WORKFLOW:
When asked to pull content from a webpage, always follow this workflow:
1. Navigate to the URL using the navigate action
2. Pull and save the content using the save_content action (saves to workingdirectory/web as a text file)
3. Report the file path where the content was pulled and saved

IMPORTANT:
- Use save_content as the default method for pulling content - it saves to a text file and enables indexing
- Use get_content ONLY when the main agent explicitly requests immediate content access or when content must be returned directly (not saved)
- Always report the file path when using save_content`,
		// Steps
		[]string{
			"Identify what web operations are needed (navigation, clicking, or content pulling)",
			"Determine the appropriate action from the 4 available: navigate, click, get_content, save_content",
			"For content pulling: use the default workflow - Navigate → save_content → Report file path",
			"For clicking: use click action with CSS selector",
			"Use get_content only when explicitly requested by main agent for immediate content access",
			"Execute the web operation using the web_browser tool",
			"Return results clearly, always including file paths when using save_content",
		},
		// Output format
		`
You have 4 simple actions available:
1. navigate: Navigate to a URL (requires: url parameter)
2. click: Click a button/link (requires: selector parameter, optional: wait_visible)
3. get_content: Pull page content (optional: type parameter) - USE ONLY when explicitly requested
4. save_content: Pull and save to text file (no parameters) - DEFAULT for content pulling

DEFAULT WORKFLOW for content pulling:
- Navigate to URL → Use save_content → Report the file path where content was saved

Return format:
- Return operation results clearly (URLs, page titles, file paths)
- Always include the file path when using save_content
- Report errors only when operations fail`,
		// Examples
		[]string{`
'user': Navigate to https://example.com

'assistant':
[Uses web_browser tool with action="navigate", url="https://example.com"]
Web Browser Operation: Navigate
URL: https://example.com
Status: Success
Page Title: Example Domain`,
			`
'user': Click the button with id "submit"

'assistant':
[Uses web_browser tool with action="click", selector="#submit"]
Web Browser Operation: Click
Selector: #submit
Status: Success`,
			`
'user': Pull the content from https://example.com

'assistant':
[Uses web_browser tool with action="navigate", url="https://example.com"]
Web Browser Operation: Navigate
URL: https://example.com
Status: Success
Page Title: Example Domain

[Uses web_browser tool with action="save_content"]
Web Browser Operation: Save Content
Filename: example_com_1234567890.txt
Path: /workingdirectory/web/example_com_1234567890.txt
Status: Success`,
			`
'user': Get the text content of the current page immediately (I need it right now)

'assistant':
[Uses web_browser tool with action="get_content", type="text"]
Web Browser Operation: Get Content
Type: text
Status: Success

Content:
Example Domain
This domain is for use in illustrative examples...`,
			`
'user': Save the content of this webpage

'assistant':
[Uses web_browser tool with action="save_content"]
Web Browser Operation: Save Content
Filename: example_com_1234567890.txt
Path: /workingdirectory/web/example_com_1234567890.txt
Status: Success`,
		},
		// Critical rules
		[]string{
			`Always specify the action parameter (navigate, click, get_content, save_content)`,
			`For navigate action: url parameter is required`,
			`For click action: selector parameter is required, wait_visible is optional (default: true)`,
			`For get_content action: type parameter is optional ("html", "text", or "title", default: "text") - USE ONLY when explicitly requested by main agent`,
			`For save_content action: no parameters needed (saves to workingdirectory/web as a text file)`,
			`DEFAULT CONTENT PULLING WORKFLOW: When pulling webpage content, always use the default flow: Navigate → Pull Content (Save) → Report location. Always report the file path where content was pulled and saved.`,
			`Use get_content ONLY when the main agent explicitly requests immediate content access or when the task requires content to be returned directly (not saved)`,
			`Return only operation results without commentary`,
			`Return URLs, page titles, file paths, and other relevant information in results`,
			`Always include the file path when using save_content action`,
			`Report errors concisely when operations fail`,
			`Browser context persists across tool calls, maintaining cookies and session`,
		},
	)

	// Build description with structured components
	template.AddDescription(
		// Incipit
		`Simple web navigation and content pulling agent. Supports 4 actions: navigate to URLs, click buttons, get content (when requested), and save content to text files (default workflow).`,
		// Examples
		[]string{
			`✅ Use for: Navigating to web pages, clicking buttons/links, pulling and saving page content`,
			`❌ Don't use: File system operations (use OS agent instead), Git operations (use Git agent instead), form filling, screenshots, or JavaScript execution`,
		},
	)

	// Add advanced description
	template.AddAdvanceDescription(`
Advanced Details:
- Purpose: Handles web navigation and content pulling operations using a headless browser
- Tools Available:
  * web_browser: Web navigation and content pulling operations (navigate, click, get_content, save_content)
- Capabilities:
  * Navigate to URLs with automatic https:// prefix if scheme is missing
  * Click elements by CSS selector with optional wait for visibility
  * Pull page content as HTML, plain text, or just the title
  * Save page content to text files
  * Browser context persists across tool calls, maintaining cookies, session, and history
- Web Operations:
  * navigate: Navigate to a URL
    - Required: url
    - Returns: current URL, page title, success status
  * click: Click an element by CSS selector
    - Required: selector
    - Optional: wait_visible (default: true)
    - Returns: success status, element info
  * get_content: Pull page content (USE ONLY when explicitly requested by main agent)
    - Optional: type ("html", "text", or "title", default: "text")
    - Returns: content string
    - Only use when main agent explicitly requests immediate content access
  * save_content: Pull and save page content to text file (DEFAULT for content pulling)
    - No parameters needed
    - Returns: filename and file path
    - Default workflow: Navigate → Pull Content (Save) → Report location
    - Saves content to workingdirectory/web as a text file
    - Enables workflow: Pull → Save → Index → Semantic Search
- Browser Context:
  * Browser context is maintained across tool calls
  * Cookies, session, and history persist between operations
  * Headless mode by default (no visible browser window)
  * Proper timeout handling for all operations
- Selectors:
  * Use CSS selectors to target elements (e.g., "#id", ".class", "tag", "tag#id.class")
  * Selectors are validated before use
  * Wait for elements to be visible before clicking when needed
- Content Pulling Workflow:
  * DEFAULT WORKFLOW: When pulling webpage content, always use: Navigate → Pull Content (Save) → Report location
  * The save_content action pulls content from the page and saves it - this is the default method for content pulling
  * Always report the file path where content was pulled and saved
  * This enables the workflow: Pull → Save → Index → Semantic Search
  * Use get_content ONLY when the main agent explicitly requests immediate content access or when content must be returned directly (not saved)
- Integration: Automatically available as a sub-agent when web operations are needed`)

	// Add troubleshooting information
	template.AddTroubleshooting(`
Troubleshooting:
- "action parameter is required": Always specify action as "navigate", "click", "get_content", or "save_content"
- "url parameter is required for navigate": Provide the URL when navigating
- "selector parameter is required for click": Provide a CSS selector when clicking
- "failed to initialize browser": Ensure Chrome/Chromium is installed and accessible
- "failed to navigate": Check URL validity and network connectivity
- "failed to click element": Verify selector is correct and element exists. Element must be visible and clickable
- "failed to get_content": Ensure page has loaded completely. Increase timeout if page loads slowly
- "failed to save_content": Ensure page has loaded completely. Check file system permissions for workingdirectory/web directory
- Common mistakes:
  * Forgetting to specify the action parameter
  * Missing required parameters for the chosen action
  * Using incorrect selector syntax
  * Not providing URL scheme (automatically adds https://)
- Best practices:
  * Always specify the action parameter first
  * Verify selectors using browser developer tools
  * Use descriptive selectors (IDs preferred over classes when available)
  * Use get_content only when explicitly requested by main agent
  * Use save_content as the default method for pulling content
  * Always report the file path when using save_content
  * Understand that browser context persists across tool calls`)

	return template
}

// WebAgent creates a web navigation and content pulling agent with web_browser tool.
//
// The agent supports four core operations:
//   - navigate: Navigate to URLs
//   - click: Click buttons and links
//   - get_content: Pull page content (only when explicitly requested)
//   - save_content: Pull and save page content to text files (default workflow)
//
// Parameters:
//   - llmEngine: The LLM engine to use for this agent
//   - workingDir: The working directory where save_content will save files (to workingDir/web)
//
// Returns:
//   - *core.SubAgent: The Web agent as a sub-agent
func WebAgent(llmEngine llms.LLMEngine, workingDir string) *core.SubAgent {
	webTemplate := createWebAgentTemplate()
	webConfig := webTemplate.ToAgentConfig(llmEngine)

	// Add web_browser tool
	webTool := web.NewWebTool(workingDir)
	webConfig.Tools = []llms.Tool{webTool}

	webAgent := NewAgent(&webConfig)
	return webAgent.AgentAsSubAgent()
}

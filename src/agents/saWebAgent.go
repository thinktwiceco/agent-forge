package agents

import (
	"github.com/thinktwice/agentForge/src/core"
	"github.com/thinktwice/agentForge/src/llms"
	"github.com/thinktwice/agentForge/src/tools/web"
)

// WebAgentTemplate defines the system agent template for web navigation and automation operations.
//
// This agent handles web operations including navigating to URLs, clicking elements, filling forms,
// taking screenshots, extracting content, waiting for elements, browser history navigation, and executing JavaScript.

func createWebAgentTemplate() *SystemAgentTemplate {
	template, err := NewSystemAgentTemplate(AgentNameSystemWeb, TraceWeb)
	if err != nil {
		panic(err)
	}

	// Build system prompt with structured components
	template.AddSystemPrompt(
		`You are a web navigation and automation specialist agent. Your role is to handle web operations
including navigating to URLs, interacting with web pages (clicking, typing), taking screenshots,
extracting page content, waiting for elements to load, navigating browser history, and executing JavaScript.

When given a task, analyze what web operations are needed:
- What URL needs to be navigated to?
- What elements need to be clicked or interacted with?
- What forms need to be filled out?
- What content needs to be extracted from pages?
- What screenshots need to be taken?
- What JavaScript needs to be executed?
- Do elements need to wait for before interacting?

IMPORTANT - CONTENT RETRIEVAL WITH INDEXING:
- Check if a vector/indexing system is available (look for "system-vector" in available sub-agents)
- If indexing system is present, use save_content as the PREFERRED method for retrieving webpage content instead of get_content
- This enables the workflow: Save → Index → Semantic Search
- After saving content with save_content, the main agent can delegate to the vector agent to index it for semantic search capabilities
- Use get_content only when you need immediate content access without indexing (e.g., quick verification)`,
		// Steps
		[]string{
			"Identify what web operations are needed for the task",
			"Check if vector/indexing system is available (look for 'system-vector' in available sub-agents)",
			"Determine the appropriate action (navigate, click, type, screenshot, get_content, save_content, wait, back, forward, evaluate)",
			"If content retrieval is needed and indexing system is available, prefer save_content over get_content",
			"Extract or prepare the required parameters for the operation",
			"Execute the web operation using the web_browser tool",
			"Format and return the results clearly",
		},
		// Output format
		`
You will perform web operations using the web_browser tool. When executing operations:
- Always specify the action parameter (navigate, click, type, screenshot, get_content, save_content, wait, back, forward, evaluate)
- For navigate: provide url parameter
- For click: provide selector parameter (optionally wait_visible)
- For type: provide selector and text parameters (optionally clear)
- For screenshot: optionally provide path and selector parameters
- For get_content: optionally provide type parameter ("html", "text", or "title")
- For save_content: no additional parameters needed (saves to workingdirectory/web)
- CONTENT RETRIEVAL: If a vector/indexing system is available (check available sub-agents for "system-vector"), use save_content as the PREFERRED method for retrieving webpage content instead of get_content. This enables indexing and semantic search capabilities.
- For wait: optionally provide selector and timeout parameters
- For back/forward: no parameters needed
- For evaluate: provide script parameter (optionally return_value)
- Provide clear feedback about what operations were performed
- Include relevant information from results (URLs, page titles, screenshot paths, file paths, etc.)
- Report any errors or issues encountered`,
		// Examples
		[]string{`
'user': Navigate to https://example.com

'assistant':
I'll navigate to https://example.com for you.
[Uses web_browser tool with action="navigate", url="https://example.com"]
Web Browser Operation: Navigate
URL: https://example.com
Status: Success
Page Title: Example Domain

Successfully navigated to https://example.com. The page title is "Example Domain".`,
			`
'user': Click the button with id "submit"

'assistant':
I'll click the submit button for you.
[Uses web_browser tool with action="click", selector="#submit"]
Web Browser Operation: Click
Selector: #submit
Status: Success

Successfully clicked the submit button.`,
			`
'user': Type "hello world" into the input field with class "search-box"

'assistant':
I'll type "hello world" into the search box.
[Uses web_browser tool with action="type", selector=".search-box", text="hello world"]
Web Browser Operation: Type
Selector: .search-box
Status: Success

Successfully typed "hello world" into the search box.`,
			`
'user': Take a screenshot of the page

'assistant':
I'll take a screenshot of the current page.
[Uses web_browser tool with action="screenshot"]
Web Browser Operation: Screenshot
Path: /tmp/screenshot_1234567890.png
Size: 245760 bytes
Status: Success

Screenshot saved to /tmp/screenshot_1234567890.png (245760 bytes).`,
			`
'user': Get the text content of the page

'assistant':
I'll extract the text content from the current page.
[Uses web_browser tool with action="get_content", type="text"]
Web Browser Operation: Get Content
Type: text
Status: Success

Content:
Example Domain
This domain is for use in illustrative examples...

Successfully extracted the text content from the page.`,
			`
'user': Save the content of this webpage for later search

'assistant':
I'll save the page content to a file. Since a vector/indexing system is available, I'll use save_content to enable indexing and semantic search capabilities.
[Uses web_browser tool with action="save_content"]
Web Browser Operation: Save Content
Filename: example_com_1234567890.txt
Path: /workingdirectory/web/example_com_1234567890.txt
Status: Success

Successfully saved the page content to example_com_1234567890.txt. The content can now be indexed via the vector agent for semantic search.`,
			`
'user': Wait for the element with id "loading" to disappear, then click the "continue" button

'assistant':
I'll wait for the loading element to disappear, then click the continue button.
[Uses web_browser tool with action="wait", selector="#loading", timeout=30]
Web Browser Operation: Wait
Selector: #loading
Timeout: 30 seconds
Waited: 2.45 seconds
Status: Success

[Uses web_browser tool with action="click", selector="#continue"]
Web Browser Operation: Click
Selector: #continue
Status: Success

Successfully waited for the loading element and clicked the continue button.`,
			`
'user': Navigate back in the browser history

'assistant':
I'll navigate back in the browser history.
[Uses web_browser tool with action="back"]
Web Browser Operation: back
URL: https://previous-page.com
Status: Success

Successfully navigated back to https://previous-page.com.`,
			`
'user': Execute JavaScript to get the page title: document.title

'assistant':
I'll execute the JavaScript to get the page title.
[Uses web_browser tool with action="evaluate", script="document.title"]
Web Browser Operation: Evaluate JavaScript
Status: Success

Result:
"Example Domain"

The page title is "Example Domain".`,
		},
		// Critical rules
		[]string{
			`Always specify the action parameter (navigate, click, type, screenshot, get_content, save_content, wait, back, forward, evaluate)`,
			`For navigate action: url parameter is required`,
			`For click action: selector parameter is required, wait_visible is optional (default: true)`,
			`For type action: selector and text parameters are required, clear is optional (default: true)`,
			`For screenshot action: path and selector are optional (saves to temp file if path not provided)`,
			`For get_content action: type parameter is optional ("html", "text", or "title", default: "text")`,
			`For save_content action: no parameters needed (saves to workingdirectory/web)`,
			`CONTENT RETRIEVAL PREFERENCE: If a vector/indexing system is available (check available sub-agents for "system-vector"), use save_content as the PREFERRED method for retrieving webpage content instead of get_content. This enables the workflow: Save → Index → Semantic Search`,
			`Use get_content only when you need immediate content access without indexing (e.g., quick verification, simple text extraction)`,
			`For wait action: selector and timeout are optional (default timeout: 30 seconds)`,
			`For back/forward actions: no parameters needed`,
			`For evaluate action: script parameter is required, return_value is optional (default: true)`,
			`Provide clear feedback about what operations were performed and their results`,
			`Include URLs, page titles, screenshot paths, file paths, and other relevant information in responses`,
			`Report errors clearly and suggest solutions when operations fail`,
			`Use wait action before interacting with dynamically loaded elements`,
			`Browser context persists across tool calls, maintaining cookies and session`,
		},
	)

	// Build description with structured components
	template.AddDescription(
		// Incipit
		`Handles web navigation and automation: navigate to URLs, interact with pages (click, type), take screenshots, extract content, wait for elements, navigate history, and execute JavaScript.`,
		// Examples
		[]string{
			`✅ Use for: Web navigation, form filling, page interaction, content extraction, screenshots, JavaScript execution`,
			`❌ Don't use: File system operations (use OS agent instead), Git operations (use Git agent instead)`,
		},
	)

	// Add advanced description
	template.AddAdvanceDescription(`
Advanced Details:
- Purpose: Handles web navigation and automation operations using a headless browser
- Tools Available:
  * web_browser: Web navigation and automation operations (navigate, click, type, screenshot, get_content, wait, back, forward, evaluate)
- Capabilities:
  * Navigate to URLs with automatic https:// prefix if scheme is missing
  * Click elements by CSS selector with optional wait for visibility
  * Type text into input fields with optional field clearing
  * Take screenshots of entire pages or specific elements
  * Extract page content as HTML, plain text, or just the title
  * Wait for elements to appear or pages to load with configurable timeouts
  * Navigate browser history (back and forward)
  * Execute JavaScript code and return results
  * Browser context persists across tool calls, maintaining cookies, session, and history
- Web Operations:
  * navigate: Navigate to a URL
    - Required: url
    - Returns: current URL, page title, success status
  * click: Click an element by CSS selector
    - Required: selector
    - Optional: wait_visible (default: true)
    - Returns: success status, element info
  * type: Type text into an input field
    - Required: selector, text
    - Optional: clear (default: true)
    - Returns: success status
  * screenshot: Take a screenshot
    - Optional: path (defaults to temp file), selector (for element screenshot)
    - Returns: screenshot path, file size
  * get_content: Get page content
    - Optional: type ("html", "text", or "title", default: "text")
    - Returns: content string
  * save_content: Save page content to file (PREFERRED when vector/indexing system is available)
    - No parameters needed
    - Returns: filename and file path
    - Enables workflow: Save → Index → Semantic Search when indexing system is present
  * wait: Wait for condition
    - Optional: selector, timeout (seconds, default: 30)
    - Returns: success status, waited duration
  * back: Navigate back in browser history
    - No parameters
    - Returns: new URL, success status
  * forward: Navigate forward in browser history
    - No parameters
    - Returns: new URL, success status
  * evaluate: Execute JavaScript
    - Required: script
    - Optional: return_value (default: true)
    - Returns: JavaScript execution result
- Browser Context:
  * Browser context is maintained across tool calls
  * Cookies, session, and history persist between operations
  * Headless mode by default (no visible browser window)
  * Proper timeout handling for all operations
- Selectors:
  * Use CSS selectors to target elements (e.g., "#id", ".class", "tag", "tag#id.class")
  * Selectors are validated before use
  * Wait for elements to be visible before interacting when needed
- Screenshots:
  * Full page screenshots by default
  * Element-specific screenshots when selector is provided
  * Saved to temp directory if path not specified
  * PNG format
- JavaScript Execution:
  * Execute arbitrary JavaScript code
  * Results are JSON-serialized for complex types
  * Can execute code without returning values
- Content Retrieval with Indexing:
  * When a vector/indexing system is available (check available sub-agents for "system-vector"), use save_content as the PREFERRED method for retrieving webpage content
  * This enables the workflow: Save → Index → Semantic Search
  * After saving content with save_content, the main agent can delegate to the vector agent to index it for semantic search capabilities
  * Use get_content only when you need immediate content access without indexing (e.g., quick verification, simple text extraction)
- Integration: Automatically available as a sub-agent when web operations are needed`)

	// Add troubleshooting information
	template.AddTroubleshooting(`
Troubleshooting:
- "action parameter is required": Always specify action as "navigate", "click", "type", "screenshot", "get_content", "wait", "back", "forward", or "evaluate"
- "url parameter is required for navigate": Provide the URL when navigating
- "selector parameter is required for click/type": Provide a CSS selector when clicking or typing
- "text parameter is required for type": Provide the text to type when using type action
- "script parameter is required for evaluate": Provide JavaScript code when evaluating
- "failed to initialize browser": Ensure Chrome/Chromium is installed and accessible
- "failed to navigate": Check URL validity and network connectivity
- "failed to click element": Verify selector is correct and element exists. Use wait action if element loads dynamically
- "failed to type": Ensure selector targets an input field and element is visible/enabled
- "failed to take screenshot": Check file system permissions if custom path is provided
- "wait failed": Element may not appear within timeout. Increase timeout or verify selector
- "failed to navigate back/forward": Browser history may be empty. Navigate to pages first
- "failed to evaluate JavaScript": Check JavaScript syntax. Some browser APIs may not be available
- Common mistakes:
  * Forgetting to specify the action parameter
  * Missing required parameters for the chosen action
  * Using incorrect selector syntax
  * Not waiting for dynamically loaded elements before interacting
  * Expecting immediate results without waiting for page loads
  * Not providing URL scheme (automatically adds https://)
- Best practices:
  * Always specify the action parameter first
  * Use wait action before interacting with dynamically loaded elements
  * Verify selectors using browser developer tools
  * Use descriptive selectors (IDs preferred over classes when available)
  * Take screenshots to verify page state when debugging
  * Use get_content to verify page content before interactions
  * Increase timeout for slow-loading pages
  * Use browser history navigation (back/forward) to navigate between pages
  * Execute JavaScript for complex interactions or data extraction
  * Understand that browser context persists across tool calls`)

	return template
}

// WebAgent creates a web navigation and automation agent with web_browser tool.
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

package web

import (
	"fmt"

	agentforge "github.com/thinktwice/agentForge/src"
	"github.com/thinktwice/agentForge/src/core"
	"github.com/thinktwice/agentForge/src/llms"
)

// GetSessionMetrics returns metrics about browser session usage.
// This can be used for monitoring and observability.
func GetSessionMetrics() SessionMetrics {
	return globalSessionManager.GetMetrics()
}

// NewWebTool creates a new web browser tool that allows web navigation and automation.
//
// Parameters:
//   - workingDir: The working directory where save_content will save files (to workingDir/web)
//
// Available actions:
//   - navigate: Navigate to a URL
//   - click: Click an element by CSS selector
//   - type: Type text into an input field
//   - screenshot: Take a screenshot of the page or element
//   - get_content: Get page content (HTML, text, or title)
//   - save_content: Extract content from webpage and save it to a file
//   - wait: Wait for an element to appear or page to load
//   - back: Navigate back in browser history
//   - forward: Navigate forward in browser history
//   - evaluate: Execute JavaScript and return result
func NewWebTool(workingDir string) llms.Tool {
	w := NewWebBrowser(workingDir)

	return &core.Tool{
		Name:        "web_browser",
		Description: "Navigate the web, interact with pages, take screenshots, and execute JavaScript using a headless browser.",
		AdvanceDesc: `Advanced Details:
- Actions:
  * navigate: Navigate to a URL. Automatically adds https:// if scheme is missing.
  * click: Click an element by CSS selector. Optionally waits for element to be visible first.
  * type: Type text into an input field. Optionally clears the field first.
  * screenshot: Take a screenshot of the entire page or a specific element. Saves to temp directory if path not provided.
  * get_content: Extract content from current page or navigate to a URL and get its content as HTML, plain text, or title.
  * save_content: Extract plain text content from current page and save it to a file in workingdirectory/web. Returns the filename. PREFERRED method for retrieving webpage content when a vector database/indexing system is available, as it enables the workflow: Save → Index → Semantic Search. When vector indexing is present, use save_content instead of get_content for content that may need to be searched later. After saving, content can be indexed via the vector agent for semantic search capabilities.
  * wait: Wait for an element to appear or page to load. Configurable timeout.
  * back: Navigate back in browser history.
  * forward: Navigate forward in browser history.
  * evaluate: Execute JavaScript code and return the result.
  * close: Close the browser session and free resources.
- Parameters:
  * action (required): The action to perform: "navigate", "click", "type", "screenshot", "get_content", "save_content", "wait", "back", "forward", "evaluate", or "close"
  * url: The URL to navigate to
    - REQUIRED for 'navigate' action
    - OPTIONAL for 'get_content' action (if not provided, extracts from current page; if provided, navigates first then extracts)
  * selector (required for click/type, optional for screenshot/wait): CSS selector for the element
  * text (required for type): Text to type into the input field
  * clear (optional for type): Whether to clear the field before typing (default: true)
  * wait_visible (optional for click): Whether to wait for element to be visible before clicking (default: true)
  * path (optional for screenshot): File path to save screenshot (defaults to temp file)
  * type (optional for get_content): Content type - "html", "text", or "title" (default: "text")
  * timeout (optional for get_content/save_content/wait): Timeout in seconds (default: 60 for get_content/save_content, 30 for wait)
  * script (required for evaluate): JavaScript code to execute
  * return_value (optional for evaluate): Whether to return the result (default: true)
- Behavior:
  * Browser context is maintained across tool calls, preserving cookies, session, and history
  * All operations include proper timeout handling
  * Screenshots are saved as PNG files
  * JavaScript evaluation returns JSON-serialized results for complex types
- Usage:
  * WORKFLOW 1 (Two-step - for interaction): Use navigate to go to a webpage, then use other actions (click, type, screenshot, get_content, save_content)
  * WORKFLOW 2 (One-step - for quick content extraction): Use get_content with url parameter to navigate and extract in one call
  * After navigation, get_content without url parameter extracts from current page
  * CONTENT RETRIEVAL PREFERENCE: When a vector database/indexing system is available (check available sub-agents), use save_content as the PREFERRED method for retrieving webpage content instead of get_content. This enables the workflow: Save → Index → Semantic Search. The saved content can then be indexed via the vector agent for semantic search capabilities.
  * Use get_content when you only need immediate content access without indexing (e.g., quick verification, simple text extraction)
  * Use save_content when content may need to be searched later or when vector indexing is available
  * Use click to interact with buttons and links
  * Use type to fill forms
  * Use screenshot to capture page state
  * Use wait to ensure elements are loaded before interaction
  * Use back/forward to navigate browser history
  * Use evaluate to execute custom JavaScript
  * Use close when done to free browser resources`,
		TroubleshootingInfo: `Troubleshooting:
- If navigate fails: Ensure URL is valid and accessible. Check network connectivity.
- If click fails: Verify selector is correct and element exists on the page. Use wait action first if element loads dynamically.
- If type fails: Ensure selector targets an input field. Check if element is visible and enabled.
- If screenshot fails: Check file system permissions if custom path is provided.
- If get_content fails: Ensure page has loaded completely. Use wait action if needed.
- If wait fails: Element may not appear within timeout. Increase timeout or verify selector.
- If back/forward fails: Browser history may be empty. Navigate to pages first to build history.
- If evaluate fails: Check JavaScript syntax. Some browser APIs may not be available.
- Browser initialization errors: Ensure Chrome/Chromium is installed and accessible.
- Timeout errors: Increase timeout parameter or check network connectivity.
- Selector errors: Use browser developer tools to verify CSS selector syntax.`,
		Parameters: []core.Parameter{
			{
				Name:        "action",
				Type:        "string",
				Description: "The action to perform: 'navigate', 'click', 'type', 'screenshot', 'get_content', 'save_content', 'wait', 'back', 'forward', 'evaluate', or 'close'",
				Required:    true,
				Validator:   validateAction,
			},
			{
				Name:        "url",
				Type:        "string",
				Description: "The URL to navigate to. REQUIRED for 'navigate'. OPTIONAL for 'get_content' (if omitted, extracts from current page)",
				Required:    false,
			},
			{
				Name:        "selector",
				Type:        "string",
				Description: "CSS selector for the element (required for 'click'/'type', optional for 'screenshot'/'wait')",
				Required:    false,
			},
			{
				Name:        "text",
				Type:        "string",
				Description: "Text to type into the input field (required for 'type' action)",
				Required:    false,
			},
			{
				Name:        "clear",
				Type:        "boolean",
				Description: "Whether to clear the field before typing (optional for 'type' action, default: true)",
				Required:    false,
			},
			{
				Name:        "wait_visible",
				Type:        "boolean",
				Description: "Whether to wait for element to be visible before clicking (optional for 'click' action, default: true)",
				Required:    false,
			},
			{
				Name:        "path",
				Type:        "string",
				Description: "File path to save screenshot (optional for 'screenshot' action, defaults to temp file)",
				Required:    false,
			},
			{
				Name:        "type",
				Type:        "string",
				Description: "Content type: 'html', 'text', or 'title' (optional for 'get_content' action, default: 'text')",
				Required:    false,
			},
			{
				Name:        "timeout",
				Type:        "number",
				Description: "Timeout in seconds (optional for 'get_content'/'wait' actions, default: 60 for get_content, 30 for wait)",
				Required:    false,
			},
			{
				Name:        "script",
				Type:        "string",
				Description: "JavaScript code to execute (required for 'evaluate' action)",
				Required:    false,
			},
			{
				Name:        "return_value",
				Type:        "boolean",
				Description: "Whether to return the JavaScript execution result (optional for 'evaluate' action, default: true)",
				Required:    false,
			},
		},
		Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
			action, ok := args["action"].(string)
			if !ok {
				return core.NewErrorResponse("action parameter is required and must be a string")
			}

			agentforge.Info("Action: ---> %s", action)

			switch action {
			case "navigate":
				return w.navigate(agentContext, args)
			case "click":
				return w.click(agentContext, args)
			case "type":
				return w.typeAction(agentContext, args)
			case "screenshot":
				return w.screenshot(agentContext, args)
			case "get_content":
				return w.getContent(agentContext, args)
			case "save_content":
				return w.saveContent(agentContext, args)
			case "wait":
				return w.wait(agentContext, args)
			case "back":
				return w.back(agentContext, args)
			case "forward":
				return w.forward(agentContext, args)
			case "evaluate":
				return w.evaluate(agentContext, args)
			case "close":
				return w.close(agentContext, args)
			default:
				return core.NewErrorResponse(fmt.Sprintf("unknown action: %s. Valid actions are: navigate, click, type, screenshot, get_content, save_content, wait, back, forward, evaluate, close", action))
			}
		},
	}
}

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

// NewWebTool creates a new web browser tool that allows web navigation and content pulling.
//
// Parameters:
//   - workingDir: The working directory where save_content will save files (to workingDir/web)
//
// Available actions:
//   - navigate: Navigate to a URL
//   - click: Click an element by CSS selector
//   - get_content: Get page content (HTML, text, or title)
//   - save_content: Pull content from webpage and save it to a text file
func NewWebTool(workingDir string) llms.Tool {
	w := NewWebBrowser(workingDir)

	return &core.Tool{
		Name:        "web_browser",
		Description: "Navigate the web, pull content from pages, and interact with buttons using a headless browser.",
		AdvanceDesc: `Advanced Details:
- Actions:
  * navigate: Navigate to a URL. Automatically adds https:// if scheme is missing.
  * click: Click an element by CSS selector. Optionally waits for element to be visible first.
  * get_content: Pull content from current page or navigate to a URL and pull its content as HTML, plain text, or title. USE ONLY when explicitly requested by main agent for immediate content access.
  * save_content: Pull plain text content from current page and save it to a text file in workingdirectory/web. Returns the filename and file path. DEFAULT method for pulling webpage content. Default workflow: Navigate → Pull Content (Save) → Report location. Enables the workflow: Pull → Save → Index → Semantic Search.
- Parameters:
  * action (required): The action to perform: "navigate", "click", "get_content", or "save_content"
  * url: The URL to navigate to
    - REQUIRED for 'navigate' action
    - OPTIONAL for 'get_content' action (if not provided, pulls from current page; if provided, navigates first then pulls)
  * selector (required for click): CSS selector for the element to click
  * wait_visible (optional for click): Whether to wait for element to be visible before clicking (default: true)
  * type (optional for get_content): Content type - "html", "text", or "title" (default: "text")
  * timeout (optional for get_content/save_content): Timeout in seconds (default: 60)
- Behavior:
  * Browser context is maintained across tool calls, preserving cookies, session, and history
  * All operations include proper timeout handling
- Usage:
  * DEFAULT CONTENT PULLING WORKFLOW: Navigate → Pull Content (Save) → Report location. Always use this workflow when pulling webpage content. Always report the file path where content was pulled and saved.
  * WORKFLOW 1 (Content pulling): Use navigate to go to a webpage, then use save_content to pull and save the content and report the file path
  * Use get_content ONLY when explicitly requested by main agent for immediate content access or when content must be returned directly (not saved)
  * Use save_content as the default method for pulling content - it pulls content from the page and saves it to a text file, enabling the workflow: Pull → Save → Index → Semantic Search
  * Use click to interact with buttons and links`,
		TroubleshootingInfo: `Troubleshooting:
- If navigate fails: Ensure URL is valid and accessible. Check network connectivity.
- If click fails: Verify selector is correct and element exists on the page. Element must be visible and clickable.
- If get_content fails: Ensure page has loaded completely. Increase timeout if page loads slowly.
- If save_content fails: Ensure page has loaded completely. Check file system permissions for workingdirectory/web directory.
- Browser initialization errors: Ensure Chrome/Chromium is installed and accessible.
- Timeout errors: Increase timeout parameter or check network connectivity.
- Selector errors: Use browser developer tools to verify CSS selector syntax.`,
		Parameters: []core.Parameter{
			{
				Name:        "action",
				Type:        "string",
				Description: "The action to perform: 'navigate', 'click', 'get_content', or 'save_content'",
				Required:    true,
				Validator:   validateAction,
			},
			{
				Name:        "url",
				Type:        "string",
				Description: "The URL to navigate to. REQUIRED for 'navigate'. OPTIONAL for 'get_content' (if omitted, pulls from current page)",
				Required:    false,
			},
			{
				Name:        "selector",
				Type:        "string",
				Description: "CSS selector for the element (required for 'click' action)",
				Required:    false,
			},
			{
				Name:        "wait_visible",
				Type:        "boolean",
				Description: "Whether to wait for element to be visible before clicking (optional for 'click' action, default: true)",
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
				Description: "Timeout in seconds (optional for 'get_content'/'save_content' actions, default: 60)",
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
			case "get_content":
				return w.getContent(agentContext, args)
			case "save_content":
				return w.saveContent(agentContext, args)
			default:
				return core.NewErrorResponse(fmt.Sprintf("unknown action: %s. Valid actions are: navigate, click, get_content, save_content", action))
			}
		},
	}
}

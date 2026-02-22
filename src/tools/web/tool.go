package web

import (
	"fmt"
	"os"

	agentforge "github.com/thinktwiceco/agent-forge/src"
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

// GetSessionMetrics returns metrics about browser session usage.
// This can be used for monitoring and observability.
func GetSessionMetrics() SessionMetrics {
	return globalSessionManager.GetMetrics()
}

// NewWebTool creates a new web browser tool that allows web navigation and content pulling.
//
// Parameters:
//   - dir: The directory this tool operates in (agent working_dir/web). Saved content is written here.
//
// Available actions:
//   - navigate: Navigate to a URL
//   - click: Click an element by CSS selector
//   - get_content: Get page content (HTML, text, or title)
//   - save_content: Pull content from webpage and save it to a text file
//   - web_search: Search the web using the Brave Search API (requires AF_BRAVE_API_KEY env var)
func NewWebTool(dir string) llms.Tool {
	_ = os.MkdirAll(dir, 0755)
	w := NewWebBrowser(dir)

	return &core.Tool{
		Name:        "web_browser",
		Description: "Navigate the web, pull content from pages, interact with buttons and form inputs using a headless browser, and search the web via the Brave Search API.",
		AdvanceDesc: `Advanced Details:
- Actions:
  * navigate: Navigate to a URL. Automatically adds https:// if scheme is missing.
  * click: Click an element by CSS selector. Optionally waits for element to be visible first.
  * fill: Clear and type text into a form input by CSS selector. Optionally waits for element to be visible first.
  * fill_secret: Same as fill but the text value is supplied via resolveSecretValue and auto-decrypted by the vault plugin before execution.
  * get_content: Pull content from current page or navigate to a URL and pull its content as HTML, plain text, or title. USE ONLY when explicitly requested by main agent for immediate content access.
  * save_content: Pull plain text content from current page and save it to a text file in workingdirectory/web. Returns the filename and file path. DEFAULT method for pulling webpage content. Default workflow: Navigate → Pull Content (Save) → Report location. Enables the workflow: Pull → Save → Index → Semantic Search.
  * web_search: Search the web using the Brave Search API. Returns a list of results with title, URL, and description. Requires braveAPIKey to be configured.
- Parameters:
  * action (required): The action to perform: "navigate", "click", "fill", "fill_secret", "get_content", "save_content", or "web_search"
  * url: The URL to navigate to
    - REQUIRED for 'navigate' action
    - OPTIONAL for 'get_content' action (if not provided, pulls from current page; if provided, navigates first then pulls)
  * selector (required for click/fill/fill_secret): CSS selector for the element
  * wait_visible (optional for click/fill/fill_secret): Whether to wait for element to be visible before acting (default: true)
  * value (required for fill): Text to type into the form input
  * resolveSecretValue (required for fill_secret): Vault key whose decrypted value will be typed into the element. Resolved automatically by the vault plugin.
  * type (optional for get_content): Content type - "html", "text", or "title" (default: "text")
  * timeout (optional for get_content/save_content): Timeout in seconds (default: 60)
  * query (required for web_search): The search query string
  * count (optional for web_search): Number of results to return, 1-20 (default: 10)
  * offset (optional for web_search): Pagination offset
  * country (optional for web_search): 2-letter country code, e.g. "US"
  * search_lang (optional for web_search): Language code, e.g. "en"
  * freshness (optional for web_search): Time filter - "pd" (past day), "pw" (past week), "pm" (past month), "py" (past year)
- Behavior:
  * Browser context is maintained across tool calls, preserving cookies, session, and history
  * All operations include proper timeout handling
  * web_search calls the Brave Search API directly (no browser required)
- Usage:
  * DEFAULT CONTENT PULLING WORKFLOW: Navigate → Pull Content (Save) → Report location. Always use this workflow when pulling webpage content. Always report the file path where content was pulled and saved.
  * WORKFLOW 1 (Content pulling): Use navigate to go to a webpage, then use save_content to pull and save the content and report the file path
  * Use get_content ONLY when explicitly requested by main agent for immediate content access or when content must be returned directly (not saved)
  * Use save_content as the default method for pulling content - it pulls content from the page and saves it to a text file, enabling the workflow: Pull → Save → Index → Semantic Search
  * Use click to interact with buttons and links
  * Use web_search to find pages before navigating to them`,
		TroubleshootingInfo: `Troubleshooting:
- If navigate fails: Ensure URL is valid and accessible. Check network connectivity.
- If click fails: Verify selector is correct and element exists on the page. Element must be visible and clickable.
- If fill/fill_secret fails: Verify selector targets an input, textarea, or other editable element. Element must be visible and interactable.
- If get_content fails: Ensure page has loaded completely. Increase timeout if page loads slowly.
- If save_content fails: Ensure page has loaded completely. Check file system permissions for workingdirectory/web directory.
- If web_search fails with "braveAPIKey not configured": Set braveAPIKey in the tool configuration.
- If web_search returns a 401/403 error: Verify the Brave API key is valid and has an active subscription.
- Browser initialization errors: Ensure Chrome/Chromium is installed and accessible.
- Timeout errors: Increase timeout parameter or check network connectivity.
- Selector errors: Use browser developer tools to verify CSS selector syntax.`,
		Parameters: []core.Parameter{
			{
				Name:        "action",
				Type:        "string",
				Description: "The action to perform: 'navigate', 'click', 'fill', 'fill_secret', 'get_content', 'save_content', or 'web_search'",
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
				Description: "Whether to wait for element to be visible before acting (optional for 'click'/'fill'/'fill_secret' actions, default: true)",
				Required:    false,
			},
			{
				Name:        "value",
				Type:        "string",
				Description: "Text value to type into the element (required for 'fill' action)",
				Required:    false,
			},
			{
				Name:        "resolveSecretValue",
				Type:        "string",
				Description: "Vault key whose decrypted value will be typed into the element (required for 'fill_secret' action). Resolved automatically by the vault plugin.",
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
			{
				Name:        "query",
				Type:        "string",
				Description: "Search query string (required for 'web_search' action)",
				Required:    false,
			},
			{
				Name:        "count",
				Type:        "number",
				Description: "Number of search results to return, 1-20 (optional for 'web_search', default: 10)",
				Required:    false,
			},
			{
				Name:        "offset",
				Type:        "number",
				Description: "Pagination offset (optional for 'web_search')",
				Required:    false,
			},
			{
				Name:        "country",
				Type:        "string",
				Description: "2-letter country code to localise results, e.g. 'US' (optional for 'web_search')",
				Required:    false,
			},
			{
				Name:        "search_lang",
				Type:        "string",
				Description: "Language code for results, e.g. 'en' (optional for 'web_search')",
				Required:    false,
			},
			{
				Name:        "freshness",
				Type:        "string",
				Description: "Time filter: 'pd' (past day), 'pw' (past week), 'pm' (past month), 'py' (past year) (optional for 'web_search')",
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
			case "fill":
				return w.fill(agentContext, args)
			case "fill_secret":
				return w.fillSecret(agentContext, args)
			case "get_content":
				return w.getContent(agentContext, args)
			case "save_content":
				return w.saveContent(agentContext, args)
			case "web_search":
				return w.webSearch(agentContext, args)
			default:
				return core.NewErrorResponse(fmt.Sprintf("unknown action: %s. Valid actions are: navigate, click, fill, fill_secret, get_content, save_content, web_search", action))
			}
		},
	}
}

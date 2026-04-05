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
//   - defaultHeadless: when non-nil, default headless mode for new sessions and implicit startup; nil uses WEB_TOOL_HEADLESS or true.
//
// Available actions:
//   - navigate: Navigate to a URL
//   - click: Click an element by CSS selector
//   - get_content: Get page content (HTML, text, title, or interactive_tree)
//   - get_snapshot: Get Chrome accessibility tree (semantic, SPA-friendly)
//   - fetch: HTTP GET without a browser (static HTML, JSON APIs)
//   - save_content: Pull content from webpage and save it to a text file
//   - web_search: Search the web using the Brave Search API (requires AF_BRAVE_API_KEY env var)
//   - upload_file: Set a local file on an <input type="file"> element via CDP (no OS dialog)
//   - refresh: Reload the current browser page
//   - list_sessions: List all active browser sessions with their last-used time
//   - close_session: Close the browser session specified by the session parameter
func NewWebTool(dir string, defaultHeadless *bool) llms.Tool {
	_ = os.MkdirAll(dir, 0755)
	w := NewWebBrowser(dir, defaultHeadless)

	detailsAbout := func(item string) string {
		switch item {
		case "navigate":
			return `navigate: Navigate to a URL.
- Required: url (string) — destination URL; https:// is added automatically if scheme is missing
- Optional: settle_ms (number, default 500) — network-idle threshold after document.readyState complete; 0 skips network-idle wait
- Optional: session (string) — browser session name`
		case "click":
			return `click: Click an element by CSS selector.
- Required: selector (string) — CSS selector for the element to click
- Optional: wait_visible (boolean, default true) — wait for element to be visible before clicking
- Optional: session (string) — browser session name`
		case "fill":
			return `fill: Clear and type text into a form input by CSS selector.
- Required: selector (string) — CSS selector for the input element
- Required: value (string) — text to type into the element
- Optional: wait_visible (boolean, default true) — wait for element to be visible before filling
- Optional: session (string) — browser session name`
		case "fill_secret":
			return `fill_secret: Type a vault secret into a form input. NOT the same as fill — do NOT put the actual password here.
- Required: selector (string) — CSS selector for the input element
- Required: resolveSecretVaultKey (string) — the vault KEY NAME from listSecrets(), NOT the actual secret value
  WRONG: resolveSecretVaultKey: "user@example.com"   ← plaintext, will fail
  RIGHT: resolveSecretVaultKey: "gmail-username"      ← key name from listSecrets()
- Optional: wait_visible (boolean, default true)
- Optional: session (string) — browser session name`
		case "get_content":
			return `get_content: Pull content from the current page (or navigate first) and return it directly.
- Optional: url (string) — if provided, navigates to this URL before pulling content
- Optional: type (string) — "html", "text", "title", or "interactive_tree" (default: "text"). "interactive_tree" returns a condensed list of interactive elements like inputs and buttons.
- Optional: strip (boolean, default true) — strip noise elements; set false to keep nav/buttons visible (only applies to text/html). When a SPA is detected, type "text" uses an accessibility snapshot instead of body text.
- Optional: timeout (number, default 60) — timeout in seconds
- Optional: settle_ms (number, default 500) — network-idle threshold (ms) after readyState complete; 0 disables network-idle wait; increase for heavy SPAs
- Optional: session (string) — browser session name`
		case "get_snapshot":
			return `get_snapshot: Return the Chrome accessibility tree of the current page as JSON.
Produces a compact semantic view that works on JS-heavy SPAs and social sites regardless of CSS obfuscation or DOM virtualization.
Merges AX nodes from all frames (OOPIFs/iframes). Waits for network-idle like navigate/get_content, then polls briefly if the tree is still empty.
If the semantic tree stays minimal, appends a dom_interactive_fallback group built from visible inputs/buttons (same heuristics as get_content interactive_tree).
- Optional: session (string)
- Optional: timeout (number, default 60)
- Optional: settle_ms (number, default 500) — post-readyState network-idle threshold in ms; 0 skips network-idle wait; raise for heavy SPAs (e.g. 1500–3000)
- Preferred over get_content type "text" on React/Next.js/Vue/Angular pages when you need explicit structure.`
		case "fetch":
			return `fetch: HTTP GET without launching Chrome — for static HTML, documentation, or JSON/XML APIs.
- Required: url (string)
- Optional env: WEB_FETCH_USER_AGENT, WEB_FETCH_MAX_BYTES (default 2 MiB cap on response body)
- Prefer navigate + get_snapshot for SPAs, auth-gated pages, or pages that need interaction.`
		case "save_content":
			return `save_content: Pull content from the current page and save it to a file (DEFAULT content-pulling method).
- Saves to workingdirectory/web as .txt (type "text") or .html (type "html")
- Returns filename and file path — always report this to the user
- Workflow: Navigate → save_content → report file path
- Optional: type (string) — "text" (default) or "html"
- Optional: strip (boolean, default true) — strip noise; set false to preserve interactive elements
- Optional: timeout (number, default 60) — timeout in seconds
- Optional: settle_ms (number, default 500) — ms to wait after readyState complete
- Optional: session (string) — browser session name`
		case "web_search":
			return `web_search: Search the web (no browser required). Uses Brave if AF_BRAVE_API_KEY is set, otherwise Tavily if AF_TAVILY_API_KEY is set.
- Required: query (string) — the search query
- Optional: count (number, 1-20, default 10) — number of results
- Optional: offset (number) — pagination offset (Brave)
- Optional: country (string) — 2-letter country code, e.g. "US" (Brave)
- Optional: search_lang (string) — language code, e.g. "en" (Brave)
- Optional: freshness (string) — "pd" (past day), "pw" (past week), "pm" (past month), "py" (past year) (Brave)
- Optional env: WEB_SEARCH_CACHE_TTL_S — in-process cache TTL in seconds (default 300; set 0 to disable)`
		case "upload_file":
			return `upload_file: Set a local file on an <input type="file"> element via CDP (bypasses OS file picker).
- Required: selector (string) — CSS selector for the file input element
- Required: file_path (string) — absolute path to the file on the host machine
- Optional: session (string) — browser session name`
		case "refresh":
			return `refresh: Reload the current browser page. No additional parameters required.
- Optional: session (string) — browser session name`
		case "list_sessions":
			return `list_sessions: List all currently active browser sessions with their last-used timestamps and idle duration. No additional parameters required.`
		case "close_session":
			return `close_session: Close the browser session identified by the session parameter.
- Optional: session (string) — name of the session to close (closes default session if omitted)
- Use this to free resources when a session is no longer needed`
		case "open_session":
			return `open_session: Open a new named browser session or resume an existing one.
- Optional: session (string) — name for the session (default: "default"); reuse the same name to resume
- Optional: headless (boolean) — run browser headlessly; defaults to agent config tool headless, else WEB_TOOL_HEADLESS, else true
- Returns confirmation of whether the session was newly created or resumed`
		default:
			return fmt.Sprintf("Nothing to add about %s", item)
		}
	}

	return core.NewTool(core.ToolConfig{
		Name:        "web_browser",
		Description: "Navigate the web, pull content (including accessibility snapshots), run lightweight HTTP fetches, interact with page elements, and search the web (Brave or Tavily).",
		AdvanceDesc: `Advanced Details:
- Available actions: open_session, navigate, click, fill, fill_secret, get_content, get_snapshot, save_content, fetch, web_search, upload_file, refresh, list_sessions, close_session
  Use expand tool with details_about="[action]" for full parameter details on any action.
- Common parameters:
  * action (required): the action to perform
  * session (optional): browser session name — omit for default session; reuse the same name to share cookies/history
- Key workflows:
  * DEFAULT content pulling: navigate → save_content → report file path
  * Use get_content or get_snapshot when content must be returned inline (not saved); use fetch for static/API URLs without Chrome
  * Use web_search to find URLs before navigating
  * Use click/fill to interact with page elements
- Behavior:
  * New browser sessions use headless from config.yaml agent.tools[].headless when set for the web tool; else WEB_TOOL_HEADLESS; else headless true
  * Passing headless explicitly to open_session overrides that default
  * Browser context (cookies, history) is preserved across calls within the same session
  * web_search uses Brave or Tavily from environment keys; results may be cached (WEB_SEARCH_CACHE_TTL_S)`,
		DetailsAboutFunc: detailsAbout,
		TroubleshootingInfo: `Troubleshooting:
- If navigate fails: Ensure URL is valid and accessible. Check network connectivity.
- If click fails: Verify selector is correct and element exists on the page. Element must be visible and clickable.
- If fill/fill_secret fails: Verify selector targets an input, textarea, or other editable element. Element must be visible and interactable.
- If get_content fails: Ensure page has loaded completely. Increase timeout if page loads slowly.
- If get_content returns empty or partial content on a JS-heavy page: Increase settle_ms (e.g. 2000) to allow async data fetches and re-renders to complete after document.readyState === 'complete'.
- If save_content fails: Ensure page has loaded completely. Check file system permissions for workingdirectory/web directory.
- If save_content returns empty or partial content on a JS-heavy page: Increase settle_ms (e.g. 2000).
- If web_search fails with missing API key: Set AF_BRAVE_API_KEY and/or AF_TAVILY_API_KEY.
- If web_search returns a 401/403 error: Verify the API key is valid and has quota.
- Browser initialization errors: Ensure Chrome/Chromium is installed and accessible.
- Timeout errors: Increase timeout parameter or check network connectivity.
- Selector errors: Use browser developer tools to verify CSS selector syntax.`,
		Parameters: []core.Parameter{
			{
				Name:        "action",
				Type:        "string",
				Description: "The action to perform: 'open_session', 'navigate', 'click', 'fill', 'fill_secret', 'get_content', 'get_snapshot', 'save_content', 'fetch', 'web_search', 'upload_file', 'refresh', 'list_sessions', or 'close_session'",
				Required:    true,
				Validator:   validateAction,
			},
			{
				Name:        "session",
				Type:        "string",
				Description: "Optional name for the browser session. Allows managing multiple independent browser sessions. Omit to use the default session. Use 'open_session' to create/resume, 'list_sessions' to see active sessions, and 'close_session' to close one.",
				Required:    false,
			},
			{
				Name:        "headless",
				Type:        "boolean",
				Description: "Whether to run the browser in headless mode (optional for 'open_session'; defaults to config tool headless, else WEB_TOOL_HEADLESS, else true)",
				Required:    false,
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
				Description: "CSS selector for the element (required for 'click', 'fill', 'fill_secret', and 'upload_file' actions)",
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
				Name:        "resolveSecretVaultKey",
				Type:        "string",
				Description: "Vault KEY NAME to look up and decrypt (required for 'fill_secret' action). Must be a key from listSecrets() — e.g. \"gmail-password\". NOT the actual password or plaintext value. If the model outputs resolve_secret_vault_key, it is normalized automatically before execution.",
				Required:    false,
			},
			{
				Name:        "type",
				Type:        "string",
				Description: "Content type (optional, default: 'text'). For 'get_content': 'html', 'text', or 'title'. For 'save_content': 'text' (saves .txt) or 'html' (saves .html).",
				Required:    false,
			},
			{
				Name:        "strip",
				Type:        "boolean",
				Description: "Whether to strip noise elements (ads, banners, cookie popups) before extracting content (optional for 'get_content'/'save_content', default: true). Set to false to preserve all interactive elements such as nav links and buttons.",
				Required:    false,
			},
			{
				Name:        "timeout",
				Type:        "number",
				Description: "Timeout in seconds (optional for 'get_content'/'save_content' actions, default: 60)",
				Required:    false,
			},
			{
				Name:        "settle_ms",
				Type:        "number",
				Description: "Network-idle threshold (ms) after document.readyState === 'complete' (optional for 'navigate', 'get_content', 'save_content'; default: 500, set to 0 to skip network-idle wait). Increase for data-heavy SPAs.",
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
			{
				Name:        "file_path",
				Type:        "string",
				Description: "Absolute path to the file on the host machine (required for 'upload_file' action)",
				Required:    false,
			},
		},
		Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
			action, ok := args["action"].(string)
			if !ok {
				return core.NewErrorResponse("action parameter is required and must be a string")
			}

			// Inject named session into agentContext so all action handlers
			// transparently resolve the correct browser session via getSessionKey.
			if s, ok := args["session"].(string); ok && s != "" {
				agentContext["browserSession"] = s
			}

			agentforge.Info("Action: ---> %s", action)

			switch action {
			case "open_session":
				return w.openSession(agentContext, args)
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
			case "get_snapshot":
				return w.getSnapshot(agentContext, args)
			case "save_content":
				return w.saveContent(agentContext, args)
			case "fetch":
				return w.fetch(agentContext, args)
			case "web_search":
				return w.webSearch(agentContext, args)
			case "upload_file":
				return w.uploadFile(agentContext, args)
			case "refresh":
				return w.refresh(agentContext)
			case "list_sessions":
				return w.listSessions()
			case "close_session":
				return w.closeSession(agentContext)
			default:
				return core.NewErrorResponse(fmt.Sprintf("unknown action: %s. Valid actions are: open_session, navigate, click, fill, fill_secret, get_content, get_snapshot, save_content, fetch, web_search, upload_file, refresh, list_sessions, close_session", action))
			}
		},
	})
}

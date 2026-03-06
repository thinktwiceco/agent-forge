package web

import (
	"context"
	"fmt"

	"github.com/chromedp/chromedp"
	agentforge "github.com/thinktwiceco/agent-forge/src"
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

// Global session manager instance
var globalSessionManager = NewSessionManager()

// getOrCreateBrowser gets an existing browser context or creates a new one.
// headless controls whether the browser runs in headless mode (default: false).
// The browser context persists across tool calls.
func getOrCreateBrowser(agentContext map[string]any, headless ...bool) (context.Context, error) {
	return globalSessionManager.GetOrCreateBrowser(agentContext, headless...)
}

// CleanupAllBrowsers cleans up all browser sessions.
// Useful for graceful shutdown.
func CleanupAllBrowsers() {
	globalSessionManager.CloseAllBrowsers()
}

// getBrowserContext returns the existing browser context without creating a new one.
// Returns nil if no browser context exists.
// This function is currently unused but kept for potential future use.
//
//nolint:unused // Reserved for future use
func getBrowserContext(agentContext map[string]any) context.Context {
	return globalSessionManager.GetBrowserContext(agentContext)
}

// stripResult is returned by stripUnwantedContent.
// When FallbackText is non-empty the DOM was over-stripped and the caller
// should use FallbackText instead of re-reading the (now empty) body.
type stripResult struct {
	Skipped      bool
	OverStripped bool
	FallbackText string
}

// stripUnwantedContent removes navigational noise (nav, header, footer, ads, sidebars, etc.)
// from the page before content extraction.
//
// Heuristics applied:
//  1. <style> and <script> tags are never removed — they do not appear in innerText so
//     stripping them has no extraction benefit, but removing <style> destroys CSS and
//     makes the page go visually black.
//  2. SPA detection runs first. If a JS framework is detected (React, Next.js, Vue,
//     Nuxt, Remix, Angular) semantic noise stripping is skipped entirely, because SPA
//     pages rarely use semantic HTML landmarks and the pattern matching would
//     over-aggressively delete real content.
//  3. On non-SPA pages the semantic noise selectors are applied, followed by a content
//     ratio guard: if the visible text shrinks by more than 50 % the stripping is
//     considered too aggressive. In that case the pre-strip text is returned as
//     FallbackText so the caller can use it instead of the now-empty DOM.
func stripUnwantedContent(ctx context.Context) (stripResult, error) {
	var raw any
	if err := chromedp.Run(ctx, chromedp.Evaluate(getScript("strip_unwanted_content"), &raw)); err != nil {
		return stripResult{}, err
	}

	m, ok := raw.(map[string]any)
	if !ok {
		return stripResult{}, nil
	}

	res := stripResult{}

	if skipped, _ := m["skipped"].(bool); skipped {
		res.Skipped = true
		agentforge.Info("stripUnwantedContent: SPA detected, semantic noise stripping skipped")
		return res, nil
	}

	if overStripped, _ := m["overStripped"].(bool); overStripped {
		res.OverStripped = true
		ratio, _ := m["ratio"].(float64)
		agentforge.Info("stripUnwantedContent: content ratio %.2f < 0.50 — using pre-strip text as fallback", ratio)
		res.FallbackText, _ = m["fallbackText"].(string)
	}

	return res, nil
}

// WebBrowser represents the web browser tool instance.
type WebBrowser struct {
	sessionManager *SessionManager
	// dir is the directory this tool operates in (agent working_dir/web).
	dir string
}

// NewWebBrowser creates a new web browser tool instance.
func NewWebBrowser(dir string) *WebBrowser {
	return &WebBrowser{
		sessionManager: globalSessionManager,
		dir:            dir,
	}
}

// listSessions returns information about all currently active browser sessions.
func (w *WebBrowser) listSessions() llms.ToolReturn {
	infos := w.sessionManager.ListSessions()

	if len(infos) == 0 {
		return core.NewSuccessResponse("Active browser sessions: none")
	}

	out := fmt.Sprintf("Active browser sessions (%d):\n\n", len(infos))
	for _, s := range infos {
		out += fmt.Sprintf("  Session: %s\n    Created:   %s\n    Last used: %s\n    Idle for:  %s\n\n",
			s.Key,
			s.Created.Format("2006-01-02 15:04:05"),
			s.LastUsed.Format("2006-01-02 15:04:05"),
			s.IdleFor,
		)
	}
	return core.NewSuccessResponse(out)
}

// closeSession closes the browser session resolved from agentContext.
func (w *WebBrowser) closeSession(agentContext map[string]any) llms.ToolReturn {
	if err := w.sessionManager.CloseBrowser(agentContext); err != nil {
		return core.NewErrorResponse(fmt.Sprintf("failed to close session: %v", err))
	}
	sessionName, _ := agentContext["browserSession"].(string)
	if sessionName == "" {
		sessionName = "default"
	}
	return core.NewSuccessResponse(fmt.Sprintf("Browser session '%s' closed successfully.", sessionName))
}

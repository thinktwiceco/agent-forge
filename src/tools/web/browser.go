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
	script := `
		(function() {
			// --- Signal A: SPA / framework detection ---
			const isSPA = !!(
				window.__NEXT_DATA__   ||
				window.__nuxt__        ||
				window.__remixContext  ||
				window.angular         ||
				document.querySelector('#__next, #app, [data-reactroot], [data-v-app]')
			);

			if (isSPA) {
				return { isSPA: true, skipped: true, removedCount: 0, overStripped: false, fallbackText: '' };
			}

			// --- Capture content before any DOM mutation ---
			const beforeText = (document.body.innerText || '').trim();
			const beforeLen  = beforeText.length;

			// Never remove an element that contains interactive children — the agent
			// needs links, buttons, and inputs to know what to click next.
			function hasInteractive(el) {
				return !!el.querySelector('a[href], button, input, select, textarea, [role="button"], [onclick]');
			}

			function safeRemove(el) {
				if (el.closest('main, article, [role="main"], [role="article"]')) return false;
				if (hasInteractive(el)) return false;
				el.remove();
				return true;
			}

			// --- Semantic noise selectors (static / server-rendered pages) ---
			// nav/header/footer are intentionally excluded — they hold navigation links.
			const selectors = [
				'aside', '[role="complementary"]',
				'[class*="advertisement"]', '[class*="promo"]',
				'[id*="advertisement"]', '[id*="promo"]',
				'[class*="sponsor"]', '[id*="sponsor"]',
				'[class*="cookie"]', '[id*="cookie"]',
				'[class*="consent"]', '[id*="consent"]',
				'[class*="gdpr"]', '[id*="gdpr"]',
				'[class*="share"]', '[id*="share"]',
				'[class*="comment"]', '[id*="comment"]',
				'[class*="discussion"]', '[id*="discussion"]',
				'[class*="newsletter"]', '[id*="newsletter"]',
				'[class*="breadcrumb"]', '[id*="breadcrumb"]',
				'nav[aria-label*="breadcrumb"]',
				'[class*="skip"]', '[id*="skip"]',
				'[style*="display: none"]', '[style*="display:none"]',
				'[hidden]',
			];

			let removedCount = 0;
			selectors.forEach(selector => {
				try {
					document.querySelectorAll(selector).forEach(el => {
						if (safeRemove(el)) removedCount++;
					});
				} catch (e) {}
			});

			// Pattern-based removal — deliberately conservative patterns only
			const unwantedPatterns = [
				/\bpromo\b/i, /\bsponsor\b/i, /\bbanner\b/i,
				/\bcookie\b/i, /\bconsent\b/i,
			];

			document.querySelectorAll('*').forEach(el => {
				const combined = ((el.className || '') + ' ' + (el.id || '')).toLowerCase();
				for (const pattern of unwantedPatterns) {
					if (pattern.test(combined) && safeRemove(el)) { removedCount++; break; }
				}
			});

			// --- Signal B: content ratio guard ---
			const afterLen = (document.body.innerText || '').trim().length;
			const ratio = beforeLen > 0 ? afterLen / beforeLen : 1;
			const overStripped = ratio < 0.5;

			return {
				isSPA: false, skipped: false, removedCount,
				beforeLen, afterLen, ratio,
				overStripped,
				fallbackText: overStripped ? beforeText : '',
			};
		})();
	`

	var raw any
	if err := chromedp.Run(ctx, chromedp.Evaluate(script, &raw)); err != nil {
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

package web

import (
	"context"

	"github.com/chromedp/chromedp"
)

// Global session manager instance
var globalSessionManager = NewSessionManager()

// getOrCreateBrowser gets an existing browser context or creates a new one.
// headless controls whether the browser runs in headless mode (default: false).
// The browser context persists across tool calls.
func getOrCreateBrowser(agentContext map[string]any, headless ...bool) (context.Context, error) {
	return globalSessionManager.GetOrCreateBrowser(agentContext, headless...)
}

// cleanupBrowser cleans up browser resources for a specific agent.
// This should only be called when explicitly closing the browser session.
func cleanupBrowser(agentContext map[string]any) error {
	return globalSessionManager.CloseBrowser(agentContext)
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

// stripUnwantedContent removes common unwanted elements from the webpage before content extraction.
// This includes navigation, headers, footers, ads, sidebars, cookie banners, etc.
// Errors are logged but not returned to allow content extraction to continue.
func stripUnwantedContent(ctx context.Context) error {
	// JavaScript to remove unwanted elements
	script := `
		(function() {
			// Common selectors for unwanted content
			const selectors = [
				// Navigation and headers
				'nav', 'header', 'footer',
				// Sidebars
				'aside', '[role="complementary"]',
				// Ads and promotional content
				'[class*="ad"]', '[class*="advertisement"]', '[class*="promo"]',
				'[id*="ad"]', '[id*="advertisement"]', '[id*="promo"]',
				'[class*="sponsor"]', '[id*="sponsor"]',
				// Cookie banners and consent
				'[class*="cookie"]', '[id*="cookie"]',
				'[class*="consent"]', '[id*="consent"]',
				'[class*="gdpr"]', '[id*="gdpr"]',
				// Social media widgets
				'[class*="social"]', '[id*="social"]',
				'[class*="share"]', '[id*="share"]',
				'[class*="follow"]', '[id*="follow"]',
				// Comments sections
				'[class*="comment"]', '[id*="comment"]',
				'[class*="discussion"]', '[id*="discussion"]',
				// Related/recommended content
				'[class*="related"]', '[id*="related"]',
				'[class*="recommend"]', '[id*="recommend"]',
				'[class*="popular"]', '[id*="popular"]',
				// Newsletter signups
				'[class*="newsletter"]', '[id*="newsletter"]',
				'[class*="subscribe"]', '[id*="subscribe"]',
				// Breadcrumbs
				'[class*="breadcrumb"]', '[id*="breadcrumb"]',
				'nav[aria-label*="breadcrumb"]',
				// Skip links and accessibility helpers
				'[class*="skip"]', '[id*="skip"]',
				// Hidden elements (already hidden but remove for safety)
				'[style*="display: none"]', '[style*="display:none"]',
				'[hidden]',
			];
			
			let removedCount = 0;
			
			// Remove elements matching selectors
			selectors.forEach(selector => {
				try {
					const elements = document.querySelectorAll(selector);
					elements.forEach(el => {
						// Don't remove if it's the main content area
						if (el.closest('main, article, [role="main"], [role="article"]')) {
							return;
						}
						el.remove();
						removedCount++;
					});
				} catch (e) {
					// Invalid selector, skip
				}
			});
			
			// Remove script and style tags (they don't contribute to text content)
			const scripts = document.querySelectorAll('script, style, noscript');
			scripts.forEach(el => el.remove());
			
			// Remove elements with common unwanted classes/IDs (case-insensitive partial match)
			const unwantedPatterns = [
				/ad[s]?/i, /promo/i, /sponsor/i, /banner/i,
				/cookie/i, /consent/i, /popup/i, /modal/i,
				/sidebar/i, /widget/i, /toolbar/i,
			];
			
			const allElements = document.querySelectorAll('*');
			allElements.forEach(el => {
				const className = el.className || '';
				const id = el.id || '';
				const combined = (className + ' ' + id).toLowerCase();
				
				// Skip main content areas
				if (el.closest('main, article, [role="main"], [role="article"]')) {
					return;
				}
				
				// Check if element matches unwanted patterns
				for (const pattern of unwantedPatterns) {
					if (pattern.test(combined)) {
						el.remove();
						removedCount++;
						break;
					}
				}
			});
			
			return removedCount;
		})();
	`

	// Execute the script to remove unwanted content
	var result interface{}
	err := chromedp.Evaluate(script, &result).Do(ctx)
	if err != nil {
		return err
	}

	return nil
}

// WebBrowser represents the web browser tool instance.
type WebBrowser struct {
	sessionManager *SessionManager
	workingDir     string
}

// NewWebBrowser creates a new web browser tool instance.
func NewWebBrowser(workingDir string) *WebBrowser {
	return &WebBrowser{
		sessionManager: globalSessionManager,
		workingDir:     workingDir,
	}
}

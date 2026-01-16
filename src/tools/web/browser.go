package web

import (
	"context"
	"time"

	"github.com/chromedp/chromedp"
)

const (
	webBrowserCtxKey    = "web_browser_ctx"
	webBrowserCancelKey = "web_browser_cancel"
	defaultTimeout      = 60 * time.Second
)

// getOrCreateBrowser gets an existing browser context from agentContext or creates a new one.
// headless controls whether the browser runs in headless mode (default: true).
func getOrCreateBrowser(agentContext map[string]any, headless ...bool) (context.Context, context.CancelFunc, error) {
	// Check if browser context already exists
	if ctx, ok := agentContext[webBrowserCtxKey].(context.Context); ok {
		if cancel, ok := agentContext[webBrowserCancelKey].(context.CancelFunc); ok {
			return ctx, cancel, nil
		}
	}

	// Determine headless mode (default to true if not specified)
	headlessMode := true
	if len(headless) > 0 {
		headlessMode = headless[0]
	}

	// Create new browser context
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", headlessMode),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("no-sandbox", true),
	)

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	ctx, cancelCtx := chromedp.NewContext(allocCtx, chromedp.WithLogf(func(format string, v ...interface{}) {
		// Log browser messages if needed
	}))

	// Store in agentContext for reuse
	agentContext[webBrowserCtxKey] = ctx
	agentContext[webBrowserCancelKey] = func() {
		cancelCtx()
		cancelAlloc()
	}

	return ctx, func() {
		cancelCtx()
		cancelAlloc()
	}, nil
}

// cleanupBrowser cleans up browser resources.
func cleanupBrowser(agentContext map[string]any) error {
	if cancel, ok := agentContext[webBrowserCancelKey].(context.CancelFunc); ok {
		cancel()
		delete(agentContext, webBrowserCtxKey)
		delete(agentContext, webBrowserCancelKey)
	}
	return nil
}

// stripUnwantedContent removes common unwanted elements from the webpage before content extraction.
// This includes navigation, headers, footers, ads, sidebars, cookie banners, etc.
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
type WebBrowser struct{}

// NewWebBrowser creates a new web browser tool instance.
func NewWebBrowser() *WebBrowser {
	return &WebBrowser{}
}

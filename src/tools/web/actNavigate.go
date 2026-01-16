package web

import (
	"context"
	"fmt"
	"strings"

	"github.com/chromedp/chromedp"
	"github.com/thinktwice/agentForge/src/core"
	"github.com/thinktwice/agentForge/src/llms"
)

// navigate handles the navigate action for the web browser tool.
func (w *WebBrowser) navigate(agentContext map[string]any, args map[string]any) llms.ToolReturn {
	// Extract URL (required)
	urlStr, ok := args["url"].(string)
	if !ok || urlStr == "" {
		return core.NewErrorResponse("url parameter is required for navigate action and must be a non-empty string")
	}

	// Validate URL
	if err := validateURL(urlStr); err != nil {
		return core.NewErrorResponse(fmt.Sprintf("invalid URL: %v", err))
	}

	// Add scheme if missing
	if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
		urlStr = "https://" + urlStr
	}

	// Get or create browser context with headless=false to follow navigation
	ctx, cancel, err := getOrCreateBrowser(agentContext, false)
	if err != nil {
		return core.NewErrorResponse(fmt.Sprintf("failed to initialize browser: %v", err))
	}
	defer func() {
		// Don't cancel here - we want to keep the browser alive
		_ = cancel
	}()

	// Create timeout context
	timeoutCtx, timeoutCancel := context.WithTimeout(ctx, defaultTimeout)
	defer timeoutCancel()

	var currentURL, pageTitle string

	// Navigate and get page info
	err = chromedp.Run(timeoutCtx,
		chromedp.Navigate(urlStr),
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.Location(&currentURL),
		chromedp.Title(&pageTitle),
	)

	if err != nil {
		return core.NewErrorResponse(fmt.Sprintf("failed to navigate to %s: %v", urlStr, err))
	}

	response := &NavigateResponse{
		Operation: "navigate",
		URL:       currentURL,
		Title:     pageTitle,
		Success:   true,
	}

	return core.NewSuccessResponse(response.String())
}

package web

import (
	"context"
	"fmt"

	"github.com/chromedp/chromedp"
	agentforge "github.com/thinktwiceco/agent-forge/src"
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

// navigate handles the navigate action for the web browser tool.
func (w *WebBrowser) navigate(agentContext map[string]any, args map[string]any) llms.ToolReturn {
	// Extract URL (required)
	urlStr, ok := args["url"].(string)
	if !ok || urlStr == "" {
		w.sessionManager.RecordOperation(false)
		return core.NewErrorResponse("url parameter is required for navigate action and must be a non-empty string")
	}

	// Normalize and validate URL
	normalizedURL, err := normalizeURL(urlStr)
	if err != nil {
		w.sessionManager.RecordOperation(false)
		return core.NewErrorResponse(fmt.Sprintf("invalid URL: %v", err))
	}

	// Get or create browser context with headless=false to follow navigation
	ctx, err := getOrCreateBrowser(agentContext, false)
	if err != nil {
		w.sessionManager.RecordOperation(false)
		return core.NewErrorResponse(fmt.Sprintf("failed to initialize browser: %v", err))
	}

	// Create timeout context for this operation only
	timeoutCtx, timeoutCancel := context.WithTimeout(ctx, defaultTimeout)
	defer timeoutCancel()

	var currentURL, pageTitle string

	// Navigate and get page info
	err = chromedp.Run(timeoutCtx,
		chromedp.Navigate(normalizedURL),
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.Location(&currentURL),
		chromedp.Title(&pageTitle),
	)

	if err != nil {
		w.sessionManager.RecordOperation(false)
		return core.NewErrorResponse(fmt.Sprintf("failed to navigate to %s: %v", normalizedURL, err))
	}

	response := &navigateResponse{
		Operation: "navigate",
		URL:       currentURL,
		Title:     pageTitle,
		Success:   true,
	}

	w.sessionManager.RecordOperation(true)
	agentforge.Info("Navigated to %s", currentURL)
	agentforge.Info("Page title: %s", pageTitle)
	agentforge.Info("Response: %s", response.String())

	return core.NewSuccessResponse(response.String())
}

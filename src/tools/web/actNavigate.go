package web

import (
	"context"
	"fmt"
	"time"

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

	// Get or create browser context using the configured default headless mode.
	ctx, err := w.getOrCreateBrowser(agentContext)
	if err != nil {
		w.sessionManager.RecordOperation(false)
		return core.NewErrorResponse(fmt.Sprintf("failed to initialize browser: %v", err))
	}

	// Create timeout context for this operation only
	timeoutCtx, timeoutCancel := context.WithTimeout(ctx, defaultTimeout)
	defer timeoutCancel()

	settleDelay := defaultSettleDelay
	if sm, ok := args["settle_ms"]; ok {
		var ms float64
		switch v := sm.(type) {
		case float64:
			ms = v
		case int:
			ms = float64(v)
		case int64:
			ms = float64(v)
		default:
			w.sessionManager.RecordOperation(false)
			return core.NewErrorResponse("settle_ms parameter must be a number")
		}
		if ms < 0 {
			w.sessionManager.RecordOperation(false)
			return core.NewErrorResponse("settle_ms must be >= 0")
		}
		settleDelay = time.Duration(ms) * time.Millisecond
	}

	var currentURL, pageTitle string

	// Inject network counter before navigation so initial XHR/fetch are tracked.
	err = chromedp.Run(timeoutCtx,
		chromedp.Evaluate(getScript("network_idle"), nil),
		chromedp.Navigate(normalizedURL),
	)
	if err == nil {
		err = waitForPageReady(timeoutCtx, settleDelay)
	}
	if err == nil {
		err = chromedp.Run(timeoutCtx,
			chromedp.Location(&currentURL),
			chromedp.Title(&pageTitle),
		)
	}

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

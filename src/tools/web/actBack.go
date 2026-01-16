package web

import (
	"context"
	"fmt"

	"github.com/chromedp/chromedp"
	"github.com/thinktwice/agentForge/src/core"
	"github.com/thinktwice/agentForge/src/llms"
)

// back handles the back action for the web browser tool.
func (w *WebBrowser) back(agentContext map[string]any, args map[string]any) llms.ToolReturn {
	// Get browser context
	ctx, _, err := getOrCreateBrowser(agentContext)
	if err != nil {
		return core.NewErrorResponse(fmt.Sprintf("failed to get browser context: %v", err))
	}

	// Create timeout context
	timeoutCtx, timeoutCancel := context.WithTimeout(ctx, defaultTimeout)
	defer timeoutCancel()

	var currentURL string

	// Navigate back and get URL
	err = chromedp.Run(timeoutCtx,
		chromedp.NavigateBack(),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.Location(&currentURL),
	)

	if err != nil {
		return core.NewErrorResponse(fmt.Sprintf("failed to navigate back: %v", err))
	}

	response := &HistoryResponse{
		Operation: "back",
		URL:       currentURL,
		Success:   true,
	}

	return core.NewSuccessResponse(response.String())
}


package web

import (
	"context"
	"fmt"

	"github.com/chromedp/chromedp"
	agentforge "github.com/thinktwiceco/agent-forge/src"
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

// refresh reloads the current browser page.
func (w *WebBrowser) refresh(agentContext map[string]any) llms.ToolReturn {
	ctx, err := w.getOrCreateBrowser(agentContext)
	if err != nil {
		w.sessionManager.RecordOperation(false)
		return core.NewErrorResponse(fmt.Sprintf("failed to initialize browser: %v", err))
	}

	timeoutCtx, timeoutCancel := context.WithTimeout(ctx, defaultTimeout)
	defer timeoutCancel()

	var currentURL, pageTitle string

	err = chromedp.Run(timeoutCtx,
		chromedp.Reload(),
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.Location(&currentURL),
		chromedp.Title(&pageTitle),
	)
	if err != nil {
		w.sessionManager.RecordOperation(false)
		return core.NewErrorResponse(fmt.Sprintf("failed to refresh page: %v", err))
	}

	response := &refreshResponse{
		URL:     currentURL,
		Title:   pageTitle,
		Success: true,
	}

	w.sessionManager.RecordOperation(true)
	agentforge.Info("Refreshed page: %s", currentURL)
	agentforge.Info("Page title: %s", pageTitle)

	return core.NewSuccessResponse(response.String())
}

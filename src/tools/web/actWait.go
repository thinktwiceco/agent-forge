package web

import (
	"context"
	"fmt"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/thinktwice/agentForge/src/core"
	"github.com/thinktwice/agentForge/src/llms"
)

// wait handles the wait action for the web browser tool.
func (w *WebBrowser) wait(agentContext map[string]any, args map[string]any) llms.ToolReturn {
	// Extract selector (optional)
	selector, hasSelector := args["selector"].(string)
	if hasSelector && selector != "" {
		if err := validateSelector(selector); err != nil {
			w.sessionManager.RecordOperation(false)
			return core.NewErrorResponse(fmt.Sprintf("invalid selector: %v", err))
		}
	}

	// Extract timeout (optional, default: 30 seconds)
	defaultWaitTimeout := 30 * time.Second
	timeoutDuration, err := parseTimeout(args, "timeout", defaultWaitTimeout)
	if err != nil {
		w.sessionManager.RecordOperation(false)
		return core.NewErrorResponse(err.Error())
	}

	// Get browser context
	ctx, err := getOrCreateBrowser(agentContext)
	if err != nil {
		w.sessionManager.RecordOperation(false)
		return core.NewErrorResponse(fmt.Sprintf("failed to get browser context: %v", err))
	}

	// Create timeout context
	timeoutCtx, timeoutCancel := context.WithTimeout(ctx, timeoutDuration)
	defer timeoutCancel()

	startTime := time.Now()

	// Wait based on selector or just wait for page load
	if hasSelector && selector != "" {
		err = chromedp.Run(timeoutCtx,
			chromedp.WaitVisible(selector, chromedp.ByQuery),
		)
	} else {
		// Wait for page to be ready
		err = chromedp.Run(timeoutCtx,
			chromedp.WaitReady("body", chromedp.ByQuery),
		)
	}

	waited := time.Since(startTime).Seconds()

	if err != nil {
		w.sessionManager.RecordOperation(false)
		return core.NewErrorResponse(fmt.Sprintf("wait failed: %v", err))
	}

	response := &waitResponse{
		Operation: "wait",
		Selector:  selector,
		Timeout:   int(timeoutDuration.Seconds()),
		Waited:    waited,
		Success:   true,
	}

	w.sessionManager.RecordOperation(true)
	return core.NewSuccessResponse(response.String())
}

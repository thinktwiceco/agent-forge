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
			return core.NewErrorResponse(fmt.Sprintf("invalid selector: %v", err))
		}
	}

	// Extract timeout (optional, default: 30 seconds)
	timeoutSeconds := 30
	if ts, ok := args["timeout"]; ok {
		switch v := ts.(type) {
		case float64:
			timeoutSeconds = int(v)
		case int:
			timeoutSeconds = v
		case int64:
			timeoutSeconds = int(v)
		default:
			return core.NewErrorResponse("timeout parameter must be a number")
		}
		if timeoutSeconds <= 0 {
			return core.NewErrorResponse("timeout must be greater than 0")
		}
	}

	// Get browser context
	ctx, _, err := getOrCreateBrowser(agentContext)
	if err != nil {
		return core.NewErrorResponse(fmt.Sprintf("failed to get browser context: %v", err))
	}

	// Create timeout context
	timeoutDuration := time.Duration(timeoutSeconds) * time.Second
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
		return core.NewErrorResponse(fmt.Sprintf("wait failed: %v", err))
	}

	response := &WaitResponse{
		Operation: "wait",
		Selector:  selector,
		Timeout:   timeoutSeconds,
		Waited:    waited,
		Success:   true,
	}

	return core.NewSuccessResponse(response.String())
}


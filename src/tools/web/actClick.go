package web

import (
	"context"
	"fmt"

	"github.com/chromedp/chromedp"
	"github.com/thinktwice/agentForge/src/core"
	"github.com/thinktwice/agentForge/src/llms"
)

// click handles the click action for the web browser tool.
func (w *WebBrowser) click(agentContext map[string]any, args map[string]any) llms.ToolReturn {
	// Extract selector (required)
	selector, ok := args["selector"].(string)
	if !ok || selector == "" {
		return core.NewErrorResponse("selector parameter is required for click action and must be a non-empty string")
	}

	// Validate selector
	if err := validateSelector(selector); err != nil {
		return core.NewErrorResponse(fmt.Sprintf("invalid selector: %v", err))
	}

	// Extract wait_visible (optional, default: true)
	waitVisible := true
	if wv, ok := args["wait_visible"]; ok {
		if b, ok := wv.(bool); ok {
			waitVisible = b
		}
	}

	// Get browser context
	ctx, _, err := getOrCreateBrowser(agentContext)
	if err != nil {
		return core.NewErrorResponse(fmt.Sprintf("failed to get browser context: %v", err))
	}

	// Create timeout context
	timeoutCtx, timeoutCancel := context.WithTimeout(ctx, defaultTimeout)
	defer timeoutCancel()

	// Build actions
	actions := []chromedp.Action{
		chromedp.Click(selector, chromedp.ByQuery),
	}

	// Optionally wait for element to be visible first
	if waitVisible {
		actions = append([]chromedp.Action{
			chromedp.WaitVisible(selector, chromedp.ByQuery),
		}, actions...)
	}

	err = chromedp.Run(timeoutCtx, actions...)

	if err != nil {
		return core.NewErrorResponse(fmt.Sprintf("failed to click element '%s': %v", selector, err))
	}

	response := &ClickResponse{
		Operation: "click",
		Selector:  selector,
		Success:   true,
	}

	return core.NewSuccessResponse(response.String())
}


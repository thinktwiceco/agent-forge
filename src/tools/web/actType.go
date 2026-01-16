package web

import (
	"context"
	"fmt"

	"github.com/chromedp/chromedp"
	"github.com/thinktwice/agentForge/src/core"
	"github.com/thinktwice/agentForge/src/llms"
)

// typeAction handles the type action for the web browser tool.
func (w *WebBrowser) typeAction(agentContext map[string]any, args map[string]any) llms.ToolReturn {
	// Extract selector (required)
	selector, ok := args["selector"].(string)
	if !ok || selector == "" {
		return core.NewErrorResponse("selector parameter is required for type action and must be a non-empty string")
	}

	// Extract text (required)
	text, ok := args["text"].(string)
	if !ok {
		return core.NewErrorResponse("text parameter is required for type action and must be a string")
	}

	// Validate selector
	if err := validateSelector(selector); err != nil {
		return core.NewErrorResponse(fmt.Sprintf("invalid selector: %v", err))
	}

	// Extract clear (optional, default: true)
	clear := true
	if c, ok := args["clear"]; ok {
		if b, ok := c.(bool); ok {
			clear = b
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
		chromedp.WaitVisible(selector, chromedp.ByQuery),
	}

	if clear {
		actions = append(actions, chromedp.Clear(selector, chromedp.ByQuery))
	}

	actions = append(actions, chromedp.SendKeys(selector, text, chromedp.ByQuery))

	err = chromedp.Run(timeoutCtx, actions...)

	if err != nil {
		return core.NewErrorResponse(fmt.Sprintf("failed to type into element '%s': %v", selector, err))
	}

	response := &TypeResponse{
		Operation: "type",
		Selector:  selector,
		Success:   true,
	}

	return core.NewSuccessResponse(response.String())
}


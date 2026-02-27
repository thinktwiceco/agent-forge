package web

import (
	"context"
	"fmt"

	"github.com/chromedp/chromedp"
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

// fill handles the fill action: clears and types text into a form input.
func (w *WebBrowser) fill(agentContext map[string]any, args map[string]any) llms.ToolReturn {
	return w.doFill(agentContext, args, "fill", "value")
}

// fillSecret handles the fill_secret action: same as fill but the value is
// pre-decrypted by the vault plugin via the resolveSecretValue argument key.
func (w *WebBrowser) fillSecret(agentContext map[string]any, args map[string]any) llms.ToolReturn {
	return w.doFill(agentContext, args, "fill_secret", "resolveSecretValue")
}

// doFill is the shared implementation for fill and fill_secret.
// valueKey is the argument name that holds the text to type.
func (w *WebBrowser) doFill(agentContext map[string]any, args map[string]any, operation, valueKey string) llms.ToolReturn {
	selector, ok := args["selector"].(string)
	if !ok || selector == "" {
		w.sessionManager.RecordOperation(false)
		return core.NewErrorResponse("selector parameter is required for " + operation + " action and must be a non-empty string")
	}

	if err := validateSelector(selector); err != nil {
		w.sessionManager.RecordOperation(false)
		return core.NewErrorResponse(fmt.Sprintf("invalid selector: %v", err))
	}

	value, ok := args[valueKey].(string)
	if !ok {
		w.sessionManager.RecordOperation(false)
		return core.NewErrorResponse(valueKey + " parameter is required for " + operation + " action and must be a string")
	}

	waitVisible := true
	if wv, ok := args["wait_visible"]; ok {
		if b, ok := wv.(bool); ok {
			waitVisible = b
		}
	}

	ctx, err := getOrCreateBrowser(agentContext)
	if err != nil {
		w.sessionManager.RecordOperation(false)
		return core.NewErrorResponse(fmt.Sprintf("failed to get browser context: %v", err))
	}

	timeoutCtx, timeoutCancel := context.WithTimeout(ctx, defaultTimeout)
	defer timeoutCancel()

	actions := []chromedp.Action{
		chromedp.Clear(selector, chromedp.ByQuery),
		chromedp.SendKeys(selector, value, chromedp.ByQuery),
	}

	if waitVisible {
		actions = append([]chromedp.Action{
			chromedp.WaitVisible(selector, chromedp.ByQuery),
		}, actions...)
	}

	if err := chromedp.Run(timeoutCtx, actions...); err != nil {
		w.sessionManager.RecordOperation(false)
		return core.NewErrorResponse(fmt.Sprintf("failed to fill element '%s': %v", selector, err))
	}

	w.sessionManager.RecordOperation(true)
	return core.NewSuccessResponse((&fillResponse{
		Operation: operation,
		Selector:  selector,
		Success:   true,
	}).String())
}

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
// pre-decrypted by the vault plugin via the resolveSecretVaultKey argument key.
// Returns an error immediately if the vault plugin is not registered, so that
// plaintext values can never slip through as a silent fallback.
func (w *WebBrowser) fillSecret(agentContext map[string]any, args map[string]any) llms.ToolReturn {
	sessionStorage, _ := agentContext["sessionStorage"].(map[string]any)
	if _, hasVault := sessionStorage["resolveSecret"]; !hasVault {
		w.sessionManager.RecordOperation(false)
		return core.NewErrorResponse("fill_secret requires the vault plugin — no vault found in this agent. Register the vault plugin or use 'fill' with a non-secret value")
	}
	return w.doFill(agentContext, args, "fill_secret", "resolveSecretVaultKey")
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

	// Use the React-compatible native value setter to clear the field.
	// chromedp.Clear only sets element.value = '' at the DOM level, which React
	// ignores — on re-render it restores its internal state and SendKeys appends
	// to the old value, causing repeated concatenation on SPA pages like Instagram.
	// The nativeInputValueSetter trick fires through React's synthetic event system
	// so the framework sees the field as truly empty before we type.
	clearScript := fmt.Sprintf(getScript("clear_input"), selector)

	actions := []chromedp.Action{
		chromedp.Evaluate(clearScript, nil),
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

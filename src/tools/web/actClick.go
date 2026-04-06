package web

import (
	"context"
	"fmt"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

const clickPollInterval = 150 * time.Millisecond

type clickElementResult struct {
	OK     bool   `json:"ok"`
	Reason string `json:"reason"`
}

// click handles the click action for the web browser tool.
func (w *WebBrowser) click(agentContext map[string]any, args map[string]any) llms.ToolReturn {
	selector, ok := args["selector"].(string)
	if !ok || selector == "" {
		w.sessionManager.RecordOperation(false)
		return core.NewErrorResponse("selector parameter is required for click action and must be a non-empty string")
	}

	if err := validateSelector(selector); err != nil {
		w.sessionManager.RecordOperation(false)
		return core.NewErrorResponse(fmt.Sprintf("invalid selector: %v", err))
	}

	waitVisible := true
	if wv, ok := args["wait_visible"]; ok {
		if b, ok := wv.(bool); ok {
			waitVisible = b
		}
	}

	timeoutDuration, err := parseTimeout(args, "timeout", defaultTimeout)
	if err != nil {
		w.sessionManager.RecordOperation(false)
		return core.NewErrorResponse(err.Error())
	}

	ctx, err := w.getOrCreateBrowser(agentContext)
	if err != nil {
		w.sessionManager.RecordOperation(false)
		return core.NewErrorResponse(fmt.Sprintf("failed to get browser context: %v", err))
	}

	timeoutCtx, timeoutCancel := context.WithTimeout(ctx, timeoutDuration)
	defer timeoutCancel()

	script := fmt.Sprintf(getScript("click_element"), selector, waitVisible)

	var last clickElementResult
	for {
		if err := timeoutCtx.Err(); err != nil {
			w.sessionManager.RecordOperation(false)
			msg := fmt.Sprintf("failed to click element '%s': %v", selector, err)
			if last.Reason != "" {
				msg = fmt.Sprintf("failed to click element '%s': %v (last probe: %s)", selector, err, last.Reason)
			}
			return core.NewErrorResponse(msg)
		}

		var res clickElementResult
		if err := chromedp.Run(timeoutCtx, chromedp.Evaluate(script, &res)); err != nil {
			w.sessionManager.RecordOperation(false)
			return core.NewErrorResponse(fmt.Sprintf("failed to click element '%s': %v", selector, err))
		}
		last = res

		if res.OK {
			response := &clickResponse{
				Operation: "click",
				Selector:  selector,
				Success:   true,
			}
			w.sessionManager.RecordOperation(true)
			return core.NewSuccessResponse(response.String())
		}

		if !waitVisible {
			w.sessionManager.RecordOperation(false)
			return core.NewErrorResponse(fmt.Sprintf("failed to click element '%s': %s (set wait_visible=true to wait, or fix selector)", selector, res.Reason))
		}

		if err := chromedp.Run(timeoutCtx, chromedp.Sleep(clickPollInterval)); err != nil {
			w.sessionManager.RecordOperation(false)
			return core.NewErrorResponse(fmt.Sprintf("failed to click element '%s': %v", selector, err))
		}
	}
}

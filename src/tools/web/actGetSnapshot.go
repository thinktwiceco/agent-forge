package web

import (
	"context"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

// getSnapshot returns the Chrome accessibility (AX) tree for the current page as JSON.
func (w *WebBrowser) getSnapshot(agentContext map[string]any, args map[string]any) llms.ToolReturn {
	ctx, err := w.getOrCreateBrowser(agentContext)
	if err != nil {
		w.sessionManager.RecordOperation(false)
		return core.NewErrorResponse("failed to get browser context: " + err.Error())
	}

	timeoutDuration, err := parseTimeout(args, "timeout", defaultTimeout)
	if err != nil {
		w.sessionManager.RecordOperation(false)
		return core.NewErrorResponse(err.Error())
	}

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

	timeoutCtx, cancel := context.WithTimeout(ctx, timeoutDuration)
	defer cancel()

	if err := waitForPageReady(timeoutCtx, settleDelay); err != nil {
		w.sessionManager.RecordOperation(false)
		return core.NewErrorResponse("page not ready for snapshot: " + err.Error())
	}
	// One short paint / AX pass after network-idle so React SPAs expose controls.
	if settleDelay > 0 {
		if err := chromedp.Run(timeoutCtx, chromedp.Sleep(50*time.Millisecond)); err != nil {
			w.sessionManager.RecordOperation(false)
			return core.NewErrorResponse("snapshot settle: " + err.Error())
		}
	}

	out, err := accessibilitySnapshotJSON(timeoutCtx)
	if err != nil {
		w.sessionManager.RecordOperation(false)
		return core.NewErrorResponse("failed to get accessibility tree: " + err.Error())
	}

	w.sessionManager.RecordOperation(true)
	return core.NewSuccessResponse(out)
}

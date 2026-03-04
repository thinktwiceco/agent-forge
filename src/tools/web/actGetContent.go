package web

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	agentforge "github.com/thinktwiceco/agent-forge/src"
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

// getContent handles the get_content action for the web browser tool.
func (w *WebBrowser) getContent(agentContext map[string]any, args map[string]any) llms.ToolReturn {
	// Extract type (optional, default: "text")
	contentType := "text"
	if ct, ok := args["type"].(string); ok && ct != "" {
		validTypes := map[string]bool{
			"html":             true,
			"text":             true,
			"title":            true,
			"interactive_tree": true,
		}
		if !validTypes[ct] {
			w.sessionManager.RecordOperation(false)
			return core.NewErrorResponse(fmt.Sprintf("invalid content type: %s. Must be one of: html, text, title, interactive_tree", ct))
		}
		contentType = ct
	}

	// Extract timeout (optional, default: 60 seconds)
	timeoutDuration, err := parseTimeout(args, "timeout", defaultTimeout)
	if err != nil {
		w.sessionManager.RecordOperation(false)
		return core.NewErrorResponse(err.Error())
	}

	// Extract settle_ms (optional, default: 500ms). Controls the post-readyState
	// settle delay that lets async re-renders finish on JS-heavy pages.
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

	// Extract URL (optional - if provided, navigate first; if not, use current page)
	urlStr, hasURL := args["url"].(string)
	var normalizedURL string
	if hasURL && urlStr != "" {
		// Normalize and validate URL
		normalizedURL, err = normalizeURL(urlStr)
		if err != nil {
			w.sessionManager.RecordOperation(false)
			return core.NewErrorResponse(fmt.Sprintf("invalid URL: %v", err))
		}
	}

	// Get or create browser context (reuses existing browser from previous navigate)
	ctx, err := getOrCreateBrowser(agentContext)
	if err != nil {
		w.sessionManager.RecordOperation(false)
		return core.NewErrorResponse(fmt.Sprintf("failed to get browser context: %v", err))
	}

	// Create timeout context for this operation only
	timeoutCtx, timeoutCancel := context.WithTimeout(ctx, timeoutDuration)
	defer timeoutCancel()

	var content string

	// Navigate to URL if provided, otherwise use current page
	if hasURL && normalizedURL != "" {
		err = chromedp.Run(timeoutCtx,
			chromedp.Navigate(normalizedURL),
		)
		if err != nil {
			w.sessionManager.RecordOperation(false)
			return core.NewErrorResponse(fmt.Sprintf("failed to navigate to %s: %v", normalizedURL, err))
		}
		if err = waitForPageReady(timeoutCtx, settleDelay); err != nil {
			w.sessionManager.RecordOperation(false)
			return core.NewErrorResponse(fmt.Sprintf("page did not become ready at %s: %v", normalizedURL, err))
		}
	} else {
		// Ensure current page is fully ready before extracting content
		if err = waitForPageReady(timeoutCtx, settleDelay); err != nil {
			w.sessionManager.RecordOperation(false)
			return core.NewErrorResponse(fmt.Sprintf("no page loaded in browser. Use navigate action first or provide url parameter: %v", err))
		}
	}

	// strip parameter: default true. Set to false to skip noise stripping entirely
	// (useful when the agent needs to see nav/interactive elements to decide what to click).
	doStrip := true
	if sv, ok := args["strip"].(bool); ok {
		doStrip = sv
	}

	// Strip unwanted content (ads, cookie banners, etc.) before extraction.
	// Only do this for text and html content types, not for title.
	var stripRes stripResult
	if doStrip && (contentType == "text" || contentType == "html") {
		stripRes, err = stripUnwantedContent(timeoutCtx)
		if err != nil {
			agentforge.Info("Warning: failed to strip unwanted content: %v", err)
		}
	}

	// If stripping was too aggressive, use the captured pre-strip text directly
	// instead of re-reading the now-empty DOM.
	if stripRes.OverStripped && stripRes.FallbackText != "" {
		content = stripRes.FallbackText
	} else {
		// Get content based on type
		switch contentType {
		case "html":
			err = chromedp.Run(timeoutCtx,
				chromedp.OuterHTML("html", &content, chromedp.ByQuery),
			)
		case "text":
			err = chromedp.Run(timeoutCtx,
				chromedp.Text("body", &content, chromedp.ByQuery),
			)
		case "title":
			err = chromedp.Run(timeoutCtx,
				chromedp.Title(&content),
			)
		case "interactive_tree":
			err = chromedp.Run(timeoutCtx,
				chromedp.Evaluate(getScript("interactive_tree"), &content),
			)
		}
	}

	if err != nil {
		w.sessionManager.RecordOperation(false)
		return core.NewErrorResponse(fmt.Sprintf("failed to get %s content: %v", contentType, err))
	}

	// Clean up content
	content = strings.TrimSpace(content)

	response := &contentResponse{
		Operation: "get_content",
		Type:      contentType,
		Content:   content,
		Success:   true,
	}

	w.sessionManager.RecordOperation(true)
	return core.NewSuccessResponse(response.String())
}

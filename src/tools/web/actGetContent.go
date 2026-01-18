package web

import (
	"context"
	"fmt"
	"strings"

	"github.com/chromedp/chromedp"
	agentforge "github.com/thinktwice/agentForge/src"
	"github.com/thinktwice/agentForge/src/core"
	"github.com/thinktwice/agentForge/src/llms"
)

// getContent handles the get_content action for the web browser tool.
func (w *WebBrowser) getContent(agentContext map[string]any, args map[string]any) llms.ToolReturn {
	// Extract type (optional, default: "text")
	contentType := "text"
	if ct, ok := args["type"].(string); ok && ct != "" {
		validTypes := map[string]bool{
			"html":  true,
			"text":  true,
			"title": true,
		}
		if !validTypes[ct] {
			w.sessionManager.RecordOperation(false)
			return core.NewErrorResponse(fmt.Sprintf("invalid content type: %s. Must be one of: html, text, title", ct))
		}
		contentType = ct
	}

	// Extract timeout (optional, default: 60 seconds)
	timeoutDuration, err := parseTimeout(args, "timeout", defaultTimeout)
	if err != nil {
		w.sessionManager.RecordOperation(false)
		return core.NewErrorResponse(err.Error())
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
			chromedp.WaitVisible("body", chromedp.ByQuery),
		)
		if err != nil {
			w.sessionManager.RecordOperation(false)
			return core.NewErrorResponse(fmt.Sprintf("failed to navigate to %s: %v", normalizedURL, err))
		}
	} else {
		// Ensure current page is ready
		err = chromedp.Run(timeoutCtx,
			chromedp.WaitVisible("body", chromedp.ByQuery),
		)
		if err != nil {
			w.sessionManager.RecordOperation(false)
			return core.NewErrorResponse(fmt.Sprintf("no page loaded in browser. Use navigate action first or provide url parameter: %v", err))
		}
	}

	// Strip unwanted content (navigation, ads, sidebars, etc.) before extraction
	// Only do this for text and html content types, not for title
	if contentType == "text" || contentType == "html" {
		err = stripUnwantedContent(timeoutCtx)
		if err != nil {
			// Log error but don't fail - continue with content extraction
			// Some pages might have elements that can't be removed, which is fine
			agentforge.Info("Warning: failed to strip unwanted content: %v", err)
		}
	}

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

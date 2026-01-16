package web

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
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
			return core.NewErrorResponse(fmt.Sprintf("invalid content type: %s. Must be one of: html, text, title", ct))
		}
		contentType = ct
	}

	// Extract timeout (optional, default: 60 seconds)
	timeoutDuration := defaultTimeout
	if ts, ok := args["timeout"]; ok {
		switch v := ts.(type) {
		case float64:
			timeoutDuration = time.Duration(v) * time.Second
		case int:
			timeoutDuration = time.Duration(v) * time.Second
		case int64:
			timeoutDuration = time.Duration(v) * time.Second
		}
		if timeoutDuration <= 0 {
			return core.NewErrorResponse("timeout must be greater than 0")
		}
	}

	// Get browser context
	ctx, _, err := getOrCreateBrowser(agentContext)
	if err != nil {
		return core.NewErrorResponse(fmt.Sprintf("failed to get browser context: %v", err))
	}

	// Create timeout context
	timeoutCtx, timeoutCancel := context.WithTimeout(ctx, timeoutDuration)
	defer timeoutCancel()

	var content string

	// Wait for page to be ready before extracting content
	// This ensures the page has loaded and is ready for content extraction
	err = chromedp.Run(timeoutCtx,
		chromedp.WaitVisible("body", chromedp.ByQuery),
	)
	if err != nil {
		return core.NewErrorResponse(fmt.Sprintf("failed to wait for page to load: %v", err))
	}

	// Strip unwanted content (navigation, ads, sidebars, etc.) before extraction
	// Only do this for text and html content types, not for title
	if contentType == "text" || contentType == "html" {
		err = stripUnwantedContent(timeoutCtx)
		if err != nil {
			// Log error but don't fail - continue with content extraction
			// Some pages might have elements that can't be removed, which is fine
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
		return core.NewErrorResponse(fmt.Sprintf("failed to get %s content: %v", contentType, err))
	}

	// Clean up content
	content = strings.TrimSpace(content)

	response := &ContentResponse{
		Operation: "get_content",
		Type:      contentType,
		Content:   content,
		Success:   true,
	}

	return core.NewSuccessResponse(response.String())
}


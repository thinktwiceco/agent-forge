package web

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	agentforge "github.com/thinktwiceco/agent-forge/src"
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

// saveContent handles the save_content action for the web browser tool.
func (w *WebBrowser) saveContent(agentContext map[string]any, args map[string]any) llms.ToolReturn {
	// Validate working directory
	if w.dir == "" {
		w.sessionManager.RecordOperation(false)
		return core.NewErrorResponse("working directory is not set")
	}

	// Extract timeout (optional, default: 60 seconds)
	timeoutDuration, err := parseTimeout(args, "timeout", defaultTimeout)
	if err != nil {
		w.sessionManager.RecordOperation(false)
		return core.NewErrorResponse(err.Error())
	}

	// Extract settle_ms (optional, default: 500ms).
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

	// Get or create browser context
	ctx, err := getOrCreateBrowser(agentContext)
	if err != nil {
		w.sessionManager.RecordOperation(false)
		return core.NewErrorResponse(fmt.Sprintf("failed to get browser context: %v", err))
	}

	// Create timeout context for this operation only
	timeoutCtx, timeoutCancel := context.WithTimeout(ctx, timeoutDuration)
	defer timeoutCancel()

	// Ensure current page is fully ready before extracting content
	if err = waitForPageReady(timeoutCtx, settleDelay); err != nil {
		w.sessionManager.RecordOperation(false)
		return core.NewErrorResponse(fmt.Sprintf("no page loaded in browser. Use navigate action first: %v", err))
	}

	// Get current URL to derive filename
	var currentURL string
	err = chromedp.Run(timeoutCtx,
		chromedp.Location(&currentURL),
	)
	if err != nil {
		w.sessionManager.RecordOperation(false)
		return core.NewErrorResponse(fmt.Sprintf("failed to get current URL: %v", err))
	}

	// Extract type (optional, default: "text")
	contentType := "text"
	if ct, ok := args["type"].(string); ok && ct != "" {
		if ct != "text" && ct != "html" {
			w.sessionManager.RecordOperation(false)
			return core.NewErrorResponse(fmt.Sprintf("invalid content type: %s. Must be one of: text, html", ct))
		}
		contentType = ct
	}

	// strip parameter: default true. Skipped for html to preserve full document.
	doStrip := contentType == "text"
	if sv, ok := args["strip"].(bool); ok {
		doStrip = sv
	}

	// Strip unwanted content before extraction.
	var stripRes stripResult
	if doStrip {
		stripRes, err = stripUnwantedContent(timeoutCtx)
		if err != nil {
			agentforge.Info("Warning: failed to strip unwanted content: %v", err)
		}
	}

	// Extract content, falling back to pre-strip text when over-stripped.
	var content string
	if contentType == "text" && stripRes.OverStripped && stripRes.FallbackText != "" {
		content = stripRes.FallbackText
	} else {
		switch contentType {
		case "html":
			err = chromedp.Run(timeoutCtx,
				chromedp.OuterHTML("html", &content, chromedp.ByQuery),
			)
		default:
			err = chromedp.Run(timeoutCtx,
				chromedp.Text("body", &content, chromedp.ByQuery),
			)
		}
		if err != nil {
			w.sessionManager.RecordOperation(false)
			return core.NewErrorResponse(fmt.Sprintf("failed to get %s content: %v", contentType, err))
		}
	}

	// Clean up content
	content = strings.TrimSpace(content)

	// Generate filename from URL domain + timestamp
	ext := ".txt"
	if contentType == "html" {
		ext = ".html"
	}
	filename, err := generateFilenameFromURL(currentURL, ext)
	if err != nil {
		w.sessionManager.RecordOperation(false)
		return core.NewErrorResponse(fmt.Sprintf("failed to generate filename from URL: %v", err))
	}

	saveDir := w.dir
	err = os.MkdirAll(saveDir, 0755)
	if err != nil {
		w.sessionManager.RecordOperation(false)
		return core.NewErrorResponse(fmt.Sprintf("failed to create directory %s: %v", saveDir, err))
	}

	// Full path to save file
	filePath := filepath.Join(saveDir, filename)

	// Save content to file
	err = os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		w.sessionManager.RecordOperation(false)
		return core.NewErrorResponse(fmt.Sprintf("failed to save content to %s: %v", filePath, err))
	}

	response := &saveContentResponse{
		Operation: "save_content",
		Filename:  filename,
		Path:      filePath,
		Success:   true,
	}

	w.sessionManager.RecordOperation(true)
	return core.NewSuccessResponse(response.String())
}

// generateFilenameFromURL generates a filename from a URL by extracting the domain
// and appending a timestamp. Example: https://example.com/page -> example_com_1234567890.txt
func generateFilenameFromURL(urlStr string, ext string) (string, error) {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	// Extract domain (host)
	domain := parsedURL.Host
	if domain == "" {
		domain = "unknown"
	}

	// Remove port if present
	if idx := strings.Index(domain, ":"); idx != -1 {
		domain = domain[:idx]
	}

	// Replace dots and special characters with underscores
	re := regexp.MustCompile(`[^a-zA-Z0-9]`)
	domain = re.ReplaceAllString(domain, "_")

	// Generate timestamp
	timestamp := time.Now().Unix()

	// Create filename: domain_timestamp.ext
	filename := fmt.Sprintf("%s_%d%s", domain, timestamp, ext)

	return filename, nil
}

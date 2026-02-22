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

	// Get or create browser context
	ctx, err := getOrCreateBrowser(agentContext)
	if err != nil {
		w.sessionManager.RecordOperation(false)
		return core.NewErrorResponse(fmt.Sprintf("failed to get browser context: %v", err))
	}

	// Create timeout context for this operation only
	timeoutCtx, timeoutCancel := context.WithTimeout(ctx, timeoutDuration)
	defer timeoutCancel()

	// Ensure current page is ready
	err = chromedp.Run(timeoutCtx,
		chromedp.WaitVisible("body", chromedp.ByQuery),
	)
	if err != nil {
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

	// Strip unwanted content before extraction
	err = stripUnwantedContent(timeoutCtx)
	if err != nil {
		// Log error but don't fail - continue with content extraction
		agentforge.Info("Warning: failed to strip unwanted content: %v", err)
	}

	// Extract plain text content
	var content string
	err = chromedp.Run(timeoutCtx,
		chromedp.Text("body", &content, chromedp.ByQuery),
	)
	if err != nil {
		w.sessionManager.RecordOperation(false)
		return core.NewErrorResponse(fmt.Sprintf("failed to get text content: %v", err))
	}

	// Clean up content
	content = strings.TrimSpace(content)

	// Generate filename from URL domain + timestamp
	filename, err := generateFilenameFromURL(currentURL)
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
func generateFilenameFromURL(urlStr string) (string, error) {
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

	// Create filename: domain_timestamp.txt
	filename := fmt.Sprintf("%s_%d.txt", domain, timestamp)

	return filename, nil
}

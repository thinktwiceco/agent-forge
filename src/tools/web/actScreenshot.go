package web

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/thinktwice/agentForge/src/core"
	"github.com/thinktwice/agentForge/src/llms"
)

// screenshot handles the screenshot action for the web browser tool.
func (w *WebBrowser) screenshot(agentContext map[string]any, args map[string]any) llms.ToolReturn {
	// Extract path (optional)
	var screenshotPath string
	if p, ok := args["path"].(string); ok && p != "" {
		screenshotPath = p
	} else {
		// Create temp file if path not provided
		tmpDir := os.TempDir()
		screenshotPath = filepath.Join(tmpDir, fmt.Sprintf("screenshot_%d.png", time.Now().Unix()))
	}

	// Extract selector (optional, for element screenshot)
	selector, hasSelector := args["selector"].(string)
	if hasSelector && selector != "" {
		if err := validateSelector(selector); err != nil {
			return core.NewErrorResponse(fmt.Sprintf("invalid selector: %v", err))
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

	var buf []byte

	// Take screenshot
	if hasSelector && selector != "" {
		// Element screenshot
		err = chromedp.Run(timeoutCtx,
			chromedp.WaitVisible(selector, chromedp.ByQuery),
			chromedp.Screenshot(selector, &buf, chromedp.ByQuery),
		)
	} else {
		// Full page screenshot
		err = chromedp.Run(timeoutCtx, chromedp.CaptureScreenshot(&buf))
	}

	if err != nil {
		return core.NewErrorResponse(fmt.Sprintf("failed to take screenshot: %v", err))
	}

	// Save screenshot
	err = os.WriteFile(screenshotPath, buf, 0644)
	if err != nil {
		return core.NewErrorResponse(fmt.Sprintf("failed to save screenshot to %s: %v", screenshotPath, err))
	}

	// Get file info
	fileInfo, err := os.Stat(screenshotPath)
	if err != nil {
		return core.NewErrorResponse(fmt.Sprintf("failed to get screenshot file info: %v", err))
	}

	response := &ScreenshotResponse{
		Operation: "screenshot",
		Path:      screenshotPath,
		Size:      fileInfo.Size(),
		Success:   true,
	}

	return core.NewSuccessResponse(response.String())
}


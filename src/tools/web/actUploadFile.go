package web

import (
	"context"
	"fmt"
	"os"

	"github.com/chromedp/chromedp"
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

// uploadFile handles the upload_file action: sets a file on an <input type="file"> element
// using the CDP protocol, bypassing the OS file picker dialog entirely.
func (w *WebBrowser) uploadFile(agentContext map[string]any, args map[string]any) llms.ToolReturn {
	selector, ok := args["selector"].(string)
	if !ok || selector == "" {
		w.sessionManager.RecordOperation(false)
		return core.NewErrorResponse("selector parameter is required for upload_file action and must be a non-empty string")
	}

	if err := validateSelector(selector); err != nil {
		w.sessionManager.RecordOperation(false)
		return core.NewErrorResponse(fmt.Sprintf("invalid selector: %v", err))
	}

	filePath, ok := args["file_path"].(string)
	if !ok || filePath == "" {
		w.sessionManager.RecordOperation(false)
		return core.NewErrorResponse("file_path parameter is required for upload_file action and must be a non-empty string")
	}

	if _, err := os.Stat(filePath); err != nil {
		w.sessionManager.RecordOperation(false)
		return core.NewErrorResponse(fmt.Sprintf("file not found at path '%s': %v", filePath, err))
	}

	waitVisible := true
	if wv, ok := args["wait_visible"]; ok {
		if b, ok := wv.(bool); ok {
			waitVisible = b
		}
	}

	ctx, err := w.getOrCreateBrowser(agentContext)
	if err != nil {
		w.sessionManager.RecordOperation(false)
		return core.NewErrorResponse(fmt.Sprintf("failed to get browser context: %v", err))
	}

	timeoutCtx, timeoutCancel := context.WithTimeout(ctx, defaultTimeout)
	defer timeoutCancel()

	actions := []chromedp.Action{
		chromedp.SendKeys(selector, filePath, chromedp.ByQuery),
	}

	if waitVisible {
		actions = append([]chromedp.Action{
			chromedp.WaitVisible(selector, chromedp.ByQuery),
		}, actions...)
	}

	if err := chromedp.Run(timeoutCtx, actions...); err != nil {
		w.sessionManager.RecordOperation(false)
		return core.NewErrorResponse(fmt.Sprintf("failed to upload file to element '%s': %v", selector, err))
	}

	w.sessionManager.RecordOperation(true)
	return core.NewSuccessResponse((&uploadFileResponse{
		Selector: selector,
		FilePath: filePath,
		Success:  true,
	}).String())
}

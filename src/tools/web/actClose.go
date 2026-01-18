package web

import (
	"github.com/thinktwice/agentForge/src/core"
	"github.com/thinktwice/agentForge/src/llms"
)

// close handles the close action for the web browser tool.
func (w *WebBrowser) close(agentContext map[string]any, args map[string]any) llms.ToolReturn {
	// Clean up browser resources
	err := cleanupBrowser(agentContext)
	if err != nil {
		w.sessionManager.RecordOperation(false)
		return core.NewErrorResponse("failed to close browser: " + err.Error())
	}

	w.sessionManager.RecordOperation(true)
	return core.NewSuccessResponse("Browser session closed successfully")
}

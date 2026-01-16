package web

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/chromedp/chromedp"
	"github.com/thinktwice/agentForge/src/core"
	"github.com/thinktwice/agentForge/src/llms"
)

// evaluate handles the evaluate action for the web browser tool.
func (w *WebBrowser) evaluate(agentContext map[string]any, args map[string]any) llms.ToolReturn {
	// Extract script (required)
	script, ok := args["script"].(string)
	if !ok || script == "" {
		return core.NewErrorResponse("script parameter is required for evaluate action and must be a non-empty string")
	}

	// Extract return_value (optional, default: true)
	returnValue := true
	if rv, ok := args["return_value"]; ok {
		if b, ok := rv.(bool); ok {
			returnValue = b
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

	var result interface{}

	// Execute JavaScript
	if returnValue {
		err = chromedp.Run(timeoutCtx,
			chromedp.Evaluate(script, &result),
		)
	} else {
		err = chromedp.Run(timeoutCtx,
			chromedp.Evaluate(script, nil),
		)
		result = "Script executed successfully (no return value)"
	}

	if err != nil {
		return core.NewErrorResponse(fmt.Sprintf("failed to evaluate JavaScript: %v", err))
	}

	// Convert result to string
	var resultStr string
	if result != nil {
		if str, ok := result.(string); ok {
			resultStr = str
		} else {
			// Try to marshal to JSON for complex types
			jsonBytes, err := json.Marshal(result)
			if err != nil {
				resultStr = fmt.Sprintf("%v", result)
			} else {
				resultStr = string(jsonBytes)
			}
		}
	} else {
		resultStr = "null"
	}

	response := &EvaluateResponse{
		Operation: "evaluate",
		Result:    resultStr,
		Success:   true,
	}

	return core.NewSuccessResponse(response.String())
}


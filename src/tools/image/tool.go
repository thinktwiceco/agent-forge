package image

import (
	"fmt"

	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

// ImageTool loads images from disk and returns them as base64 data URIs
// suitable for passing to vision-capable LLMs via UserMessageWithImages.
type ImageTool struct {
	dir string
}

// NewImageTool creates an image tool sandboxed to dir.
// The returned data URI can be passed directly to llms.UserMessageWithImages.
func NewImageTool(dir string) llms.Tool {
	t := &ImageTool{dir: dir}

	return &core.Tool{
		Name:        "image",
		Description: "Load an image file and return it as a base64 data URI for vision-capable LLMs.",
		AdvanceDesc: `Advanced Details:
- Parameters:
  * operation (string, required): The operation to perform - currently only "load"
  * path (string, required): Image file path relative to the sandbox root
- Behavior:
  * Reads the image file and returns a base64-encoded data URI (data:<mime>;base64,<data>)
  * The returned data URI can be passed to the LLM via a multimodal user message
  * All paths are validated to stay within the sandbox root directory
  * Path traversal attempts (e.g., "../") are blocked for security
- Supported formats: jpg, jpeg, png, gif, webp
- Usage:
  * Use "load" to read an image and obtain its data URI`,
		TroubleshootingInfo: `Troubleshooting:
- "path traversal detected": The path attempts to escape the root - use relative paths only
- "image not found": The file does not exist - verify the path is correct
- "unsupported image format": Only jpg, jpeg, png, gif, webp are supported
- "invalid operation": Operation must be "load"`,
		Parameters: []core.Parameter{
			{
				Name:        "operation",
				Type:        "string",
				Description: "The operation to perform: 'load'",
				Required:    true,
			},
			{
				Name:        "path",
				Type:        "string",
				Description: "Image file path relative to the sandbox root",
				Required:    true,
			},
		},
		Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
			operation, ok := args["operation"].(string)
			if !ok {
				return core.NewErrorResponse("operation must be a string")
			}

			if operation != "load" {
				return core.NewErrorResponse(fmt.Sprintf("invalid operation '%s': must be 'load'", operation))
			}

			path, ok := args["path"].(string)
			if !ok {
				return core.NewErrorResponse("missing required parameter: path")
			}

			return t.loadImage(path)
		},
	}
}

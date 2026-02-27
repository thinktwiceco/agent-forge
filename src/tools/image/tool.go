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

	detailsAbout := func(item string) string {
		switch item {
		case "load":
			return `load: Read an image file and return it as a base64 data URI.
- Required: path (string) — image file path relative to the sandbox root
- Returns: data URI in the form data:<mime>;base64,<data>
- Supported formats: jpg, jpeg, png, gif, webp
- The returned data URI can be passed to the LLM via a multimodal user message`
		default:
			return fmt.Sprintf("Nothing to add about %s", item)
		}
	}

	return &core.Tool{
		Name:        "image",
		Description: "Load an image file and return it as a base64 data URI for vision-capable LLMs.",
		AdvanceDesc: `Advanced Details:
- Available operations: load
  Use expand tool with details_about="load" for full parameter details.
- Common parameters:
  * operation (string, required): the operation to perform
- Behavior:
  * All paths are validated and sandboxed to the root directory; path traversal ("../") is blocked`,
		DetailsAboutFunc: detailsAbout,
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

package image

import (
	"encoding/base64"
	"fmt"
	"os"

	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

// loadImage reads an image file from the sandbox, encodes it as a base64 data URI,
// and returns it as a tool response ready to be passed to a vision-capable LLM.
func (t *ImageTool) loadImage(path string) llms.ToolReturn {
	validatedPath, err := t.validatePath(path)
	if err != nil {
		return core.NewErrorResponse(err.Error())
	}

	mimeType, err := mimeTypeForPath(path)
	if err != nil {
		return core.NewErrorResponse(err.Error())
	}

	data, err := os.ReadFile(validatedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return core.NewErrorResponse(fmt.Sprintf("image not found: %s", path))
		}
		return core.NewErrorResponse(fmt.Sprintf("failed to read image '%s': %v", path, err))
	}

	encoded := base64.StdEncoding.EncodeToString(data)
	dataURI := fmt.Sprintf("data:%s;base64,%s", mimeType, encoded)

	return core.NewSuccessResponse(dataURI)
}

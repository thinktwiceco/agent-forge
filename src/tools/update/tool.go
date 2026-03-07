package update

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

const updateScriptName = "update-release.sh"

// Update runs the installation's update-release.sh script from the agent root.
type Update struct {
	dir string
}

// NewUpdateTool creates a built-in tool that executes update-release.sh in the
// agent working directory root.
func NewUpdateTool(dir string) llms.Tool {
	updateTool := &Update{dir: dir}

	return &core.Tool{
		Name:        "update",
		Description: "Run the root update-release.sh script to update the current agent installation.",
		AdvanceDesc: `Advanced Details:
- Parameters: none
- Behavior:
  * Resolves exactly ./update-release.sh in the agent root
  * Refuses to run if the script is missing or is not a regular file
  * Executes the script with bash from the agent root
  * Returns captured stdout and stderr so the caller can inspect update results`,
		TroubleshootingInfo: `Troubleshooting:
- "update-release.sh not found": The agent installation does not contain the update script in its root directory
- "update-release.sh is not a regular file": The path exists but is not a normal file
- "bash is required": Install bash or ensure it is available in PATH
- Non-zero exit: Inspect stdout/stderr returned by the tool for the underlying update error`,
		Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
			result, err := updateTool.run()
			if err != nil {
				return core.NewErrorResponse(err.Error())
			}
			return core.NewSuccessResponse(result)
		},
	}
}

func (u *Update) validateScriptPath() (string, error) {
	absRoot, err := filepath.Abs(u.dir)
	if err != nil {
		return "", fmt.Errorf("invalid root directory: %w", err)
	}

	scriptPath := filepath.Join(absRoot, updateScriptName)
	absScriptPath, err := filepath.Abs(scriptPath)
	if err != nil {
		return "", fmt.Errorf("invalid update script path: %w", err)
	}

	relPath, err := filepath.Rel(absRoot, absScriptPath)
	if err != nil {
		return "", fmt.Errorf("path validation failed: %w", err)
	}
	if len(relPath) >= 2 && relPath[:2] == ".." {
		return "", fmt.Errorf("path traversal detected while resolving %s", updateScriptName)
	}

	info, err := os.Stat(absScriptPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("%s not found in agent root: %s", updateScriptName, absRoot)
		}
		return "", fmt.Errorf("failed to stat %s: %w", updateScriptName, err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("%s is not a regular file", updateScriptName)
	}

	return absScriptPath, nil
}

func (u *Update) run() (string, error) {
	absRoot, err := filepath.Abs(u.dir)
	if err != nil {
		return "", fmt.Errorf("invalid root directory: %w", err)
	}

	scriptPath, err := u.validateScriptPath()
	if err != nil {
		return "", err
	}

	if _, err := exec.LookPath("bash"); err != nil {
		return "", fmt.Errorf("bash is required to run %s: %w", updateScriptName, err)
	}

	cmd := exec.Command("bash", scriptPath)
	cmd.Dir = absRoot

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	stdoutStr := strings.TrimSpace(stdout.String())
	stderrStr := strings.TrimSpace(stderr.String())

	if err != nil {
		var details []string
		if stdoutStr != "" {
			details = append(details, "stdout:\n"+stdoutStr)
		}
		if stderrStr != "" {
			details = append(details, "stderr:\n"+stderrStr)
		}
		if len(details) == 0 {
			return "", fmt.Errorf("%s failed: %w", updateScriptName, err)
		}
		return "", fmt.Errorf("%s failed: %w\n\n%s", updateScriptName, err, strings.Join(details, "\n\n"))
	}

	parts := []string{fmt.Sprintf("Update script completed successfully: %s", updateScriptName)}
	if stdoutStr != "" {
		parts = append(parts, "stdout:\n"+stdoutStr)
	}
	if stderrStr != "" {
		parts = append(parts, "stderr:\n"+stderrStr)
	}

	return strings.Join(parts, "\n\n"), nil
}

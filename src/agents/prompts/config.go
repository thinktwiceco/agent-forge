package prompts

import "github.com/thinktwiceco/agent-forge/src/llms"

// Config holds configuration for prompt building.
type Config struct {
	SystemPrompt     string
	MainAgent        bool
	Tone             string
	Tools            []llms.Tool
	CanSpawnSubagent bool
}

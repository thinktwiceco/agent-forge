package main

import (
	"fmt"
	"os"

	"github.com/thinktwiceco/agent-forge/src/tools/api"
)

// RegisterApiHooks registers all API authentication hooks
// Call this function before building the agent
func RegisterApiHooks() {
	// Register HelpCrunch authentication hook
	// Reads API key from HELPCRUNCH_API_KEY environment variable
	api.RegisterHook("helpcrunch_auth", func(url string, headers map[string]string, body string) (map[string]string, error) {
		apiKey := os.Getenv("HELPCRUNCH_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("HELPCRUNCH_API_KEY environment variable not set")
		}

		if headers == nil {
			headers = make(map[string]string)
		}

		headers["Authorization"] = fmt.Sprintf("Bearer %s", apiKey)
		headers["Content-Type"] = "application/json"

		return headers, nil
	})
}

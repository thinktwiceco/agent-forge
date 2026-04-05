package vault

import (
	"encoding/json"
	"fmt"

	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

const (
	saveSecretTool   = "saveSecret"
	listSecretsTool  = "listSecrets"
	deleteSecretTool = "deleteSecret"
)

func newSaveSecretTool(p *VaultPlugin) llms.Tool {
	return core.NewTool(core.ToolConfig{
		Name:        saveSecretTool,
		Description: "Encrypts a secret value and stores it in the vault under the given key. Use this to securely persist API keys, passwords, or any sensitive information.",
		AdvanceDesc: `Advanced Details:
- The secret is encrypted with AES-256-GCM before being written to disk.
- The key is a human-readable identifier (e.g. "openai-api-key", "db-password").
- Once saved, the key can be passed to tools that accept resolveSecret* parameters.
- Overwriting an existing key replaces the stored secret.`,
		TroubleshootingInfo: `Troubleshooting:
- Ensure VAULT_MASTER_KEY is set in the environment.
- The 'key' must be a non-empty string.
- The 'value' must be a non-empty string.`,
		Parameters: []core.Parameter{
			{
				Name:        "key",
				Type:        "string",
				Description: "Unique identifier for the secret (e.g. 'openai-api-key')",
				Required:    true,
			},
			{
				Name:        "value",
				Type:        "string",
				Description: "The secret value to encrypt and store",
				Required:    true,
			},
		},
		Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
			key, ok := args["key"].(string)
			if !ok || key == "" {
				return core.NewErrorResponse("'key' must be a non-empty string")
			}

			value, ok := args["value"].(string)
			if !ok || value == "" {
				return core.NewErrorResponse("'value' must be a non-empty string")
			}

			if err := p.storeSecret(key, value); err != nil {
				return core.NewErrorResponse(fmt.Sprintf("failed to save secret: %v", err))
			}

			return core.NewSuccessResponse(fmt.Sprintf("Secret '%s' saved successfully", key))
		},
	})
}

func newDeleteSecretTool(p *VaultPlugin) llms.Tool {
	return core.NewTool(core.ToolConfig{
		Name:        deleteSecretTool,
		Description: "Deletes a secret from the vault by its key. The encrypted value is permanently removed from disk.",
		AdvanceDesc: `Advanced Details:
- The secret is removed from the in-memory cache and from vault/secrets.json.
- This operation is irreversible; the plaintext cannot be recovered after deletion.`,
		TroubleshootingInfo: `Troubleshooting:
- Returns an error if the key does not exist in the vault.
- The 'key' must be a non-empty string matching a key returned by listSecrets().`,
		Parameters: []core.Parameter{
			{
				Name:        "key",
				Type:        "string",
				Description: "Identifier of the secret to delete (e.g. 'openai-api-key')",
				Required:    true,
			},
		},
		Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
			key, ok := args["key"].(string)
			if !ok || key == "" {
				return core.NewErrorResponse("'key' must be a non-empty string")
			}

			if err := p.deleteSecret(key); err != nil {
				return core.NewErrorResponse(fmt.Sprintf("failed to delete secret: %v", err))
			}

			return core.NewSuccessResponse(fmt.Sprintf("Secret '%s' deleted successfully", key))
		},
	})
}

func newListSecretsTool(p *VaultPlugin) llms.Tool {
	return core.NewTool(core.ToolConfig{
		Name:        listSecretsTool,
		Description: "Lists all secret keys stored in the vault. Returns only the key identifiers, never the secret values.",
		AdvanceDesc: `Advanced Details:
- Returns a JSON array of key strings.
- The secret values are never revealed; only the identifiers are returned.
- Use the returned keys when calling tools that accept resolveSecret* parameters.`,
		TroubleshootingInfo: `Troubleshooting:
- Returns an empty array if no secrets have been saved yet.`,
		Parameters: []core.Parameter{},
		Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
			keys := p.listSecretKeys()

			data, err := json.Marshal(keys)
			if err != nil {
				return core.NewErrorResponse(fmt.Sprintf("failed to serialize secret keys: %v", err))
			}

			return core.NewEphemeralResponse(string(data))
		},
	})
}

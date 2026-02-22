package vault

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/thinktwiceco/agent-forge/src/agents"
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
	"github.com/thinktwiceco/agent-forge/src/plugins/registry"
)

const (
	PLUGIN_NAME       = "vault"
	masterKeyEnv      = "VAULT_MASTER_KEY"
	sessionStorageKey = "resolveSecret"
)

// VaultPlugin stores encrypted secrets in a local vault/ directory.
// Secrets are encrypted with AES-256-GCM using a master key from VAULT_MASTER_KEY env var.
//
// The plugin provides:
//   - saveSecret tool: encrypt and persist a secret
//   - listSecrets tool: return all stored key identifiers
//   - EventContextBuild hook: inject a resolveSecret(key) function into SessionStorage
//   - EventBeforeToolExecution hook: auto-decrypt any tool argument whose name starts with "resolveSecret"
type VaultPlugin struct {
	dir       string
	masterKey []byte
	secrets   map[string]string
	mu        sync.RWMutex
}

// NewVaultPlugin creates a new VaultPlugin.
// The plugin operates in the "vault" subdirectory of workingDir.
// The VAULT_MASTER_KEY environment variable must be set before the agent initializes.
//
// Parameters:
//   - workingDir: The agent working directory. The plugin will use workingDir/vault.
func NewVaultPlugin(workingDir string) *VaultPlugin {
	dir := filepath.Join(workingDir, "vault")
	return &VaultPlugin{
		dir:     dir,
		secrets: make(map[string]string),
	}
}

// Name implements core.Plugin.
func (p *VaultPlugin) Name() string {
	return PLUGIN_NAME
}

// Hooks implements core.HookProvider.
func (p *VaultPlugin) Hooks() map[core.Event]core.AgentHookFn {
	return map[core.Event]core.AgentHookFn{
		core.EventAgentInitialized:    agents.OnAgentInitializedHook(p.onAgentInitialized),
		core.EventContextBuild:        agents.OnContextBuildHook(p.onContextBuild),
		core.EventBeforeToolExecution: agents.BeforeToolExecutionHook(p.onBeforeToolExecution),
	}
}

// Tools implements core.ToolProvider.
func (p *VaultPlugin) Tools() []llms.Tool {
	return []llms.Tool{
		newSaveSecretTool(p),
		newListSecretsTool(p),
	}
}

// SystemPrompt implements core.PromptProvider.
func (p *VaultPlugin) SystemPrompt() string {
	return `
[VAULT]
- Encrypted secret storage (API keys, passwords)
- saveSecret(key, value): Store secret. key = identifier (e.g. "openai-api-key"), value = plaintext
- listSecrets(): List stored key identifiers (never values)

[RESOLVE SECRETS IN TOOLS]
- Tool params starting with resolveSecret* (e.g. resolveSecretApiKey, resolveSecretPassword)
- Steps: 1) Call listSecrets() 2) Pass vault key as arg value. Example: resolveSecretApiKey: "openai-api-key"
- Runtime auto-decrypts before tool execution. Never see or handle plaintext.
`
}

// onAgentInitialized loads the master key and existing secrets from disk.
func (p *VaultPlugin) onAgentInitialized(a *agents.Agent) error {
	rawKey := os.Getenv(masterKeyEnv)
	if rawKey == "" {
		return fmt.Errorf("vault plugin: %s environment variable is not set", masterKeyEnv)
	}

	key, err := base64.StdEncoding.DecodeString(rawKey)
	if err != nil {
		return fmt.Errorf("vault plugin: %s must be base64-encoded: %w", masterKeyEnv, err)
	}

	if len(key) != 32 {
		return fmt.Errorf("vault plugin: %s must decode to exactly 32 bytes for AES-256 (got %d)", masterKeyEnv, len(key))
	}

	p.mu.Lock()
	p.masterKey = key
	p.mu.Unlock()

	if err := os.MkdirAll(p.dir, 0700); err != nil {
		return fmt.Errorf("vault plugin: failed to create vault directory: %w", err)
	}

	secrets, err := loadSecrets(p.dir)
	if err != nil {
		return err
	}

	p.mu.Lock()
	p.secrets = secrets
	p.mu.Unlock()

	return nil
}

// onContextBuild injects the resolveSecret function into SessionStorage so tools
// can programmatically decrypt secrets at runtime.
func (p *VaultPlugin) onContextBuild(a *agents.Agent, agentContext *core.AgentContext) error {
	if agentContext.SessionStorage == nil {
		agentContext.SessionStorage = make(map[string]any)
	}

	agentContext.SessionStorage[sessionStorageKey] = func(key string) (string, error) {
		return p.resolveSecret(key)
	}

	return nil
}

// onBeforeToolExecution scans tool arguments for keys starting with "resolveSecret"
// and replaces their values with the decrypted secret in-place.
func (p *VaultPlugin) onBeforeToolExecution(a *agents.Agent, toolCall *llms.ToolCall) error {
	for k, v := range toolCall.Arguments {
		if !strings.HasPrefix(k, "resolveSecret") {
			continue
		}

		vaultKey, ok := v.(string)
		if !ok {
			continue
		}

		decrypted, err := p.resolveSecret(vaultKey)
		if err != nil {
			return fmt.Errorf("vault: failed to resolve secret for argument '%s': %w", k, err)
		}

		toolCall.Arguments[k] = decrypted
	}

	return nil
}

// resolveSecret decrypts a stored secret by its vault key.
func (p *VaultPlugin) resolveSecret(key string) (string, error) {
	p.mu.RLock()
	encrypted, exists := p.secrets[key]
	masterKey := p.masterKey
	p.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("vault: secret '%s' not found", key)
	}

	return decrypt(masterKey, encrypted)
}

// storeSecret encrypts and persists a secret under the given key.
func (p *VaultPlugin) storeSecret(key, value string) error {
	p.mu.RLock()
	masterKey := p.masterKey
	p.mu.RUnlock()

	if masterKey == nil {
		return fmt.Errorf("vault: master key not loaded; ensure the agent is initialized")
	}

	encrypted, err := encrypt(masterKey, value)
	if err != nil {
		return err
	}

	p.mu.Lock()
	p.secrets[key] = encrypted
	secretsCopy := make(map[string]string, len(p.secrets))
	for k, v := range p.secrets {
		secretsCopy[k] = v
	}
	p.mu.Unlock()

	return saveSecrets(p.dir, secretsCopy)
}

func init() {
	registry.Register(PLUGIN_NAME, func(workingDir string) core.Plugin {
		return NewVaultPlugin(workingDir)
	})
}

// listSecretKeys returns all stored secret key identifiers.
func (p *VaultPlugin) listSecretKeys() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	keys := make([]string, 0, len(p.secrets))
	for k := range p.secrets {
		keys = append(keys, k)
	}
	return keys
}

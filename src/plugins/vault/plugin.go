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
		newDeleteSecretTool(p),
	}
}

// SystemPrompt implements core.PromptProvider.
func (p *VaultPlugin) SystemPrompt() string {
	return `
[VAULT]
- Encrypted secret storage for sensitive values (passwords, API keys, tokens).
- saveSecret(key, value): Encrypt and persist a secret. key = short identifier, value = plaintext secret.
- listSecrets(): List stored key identifiers only — never the values.
- deleteSecret(key): Permanently remove a secret from the vault.

[USING SECRETS IN TOOLS — fill_secret]
To type a secret into a web form without exposing the plaintext:
  1. Call listSecrets() to find the vault key name (e.g. "github-password").
  2. Use action "fill_secret" with the vault key as resolveSecretValue — NOT the actual secret.
     Example:
       action: "fill_secret"
       selector: "#password"
       resolveSecretValue: "github-password"
  3. The runtime decrypts it automatically before the tool runs. You never handle the plaintext.

IMPORTANT:
- resolveSecretVaultKey must be a vault KEY NAME (from listSecrets), NOT the actual password/token.
- Never put plaintext secrets into any tool argument. Use fill_secret instead of fill for passwords.
- Any tool argument whose name starts with "resolveSecret" is auto-decrypted the same way.

COMMON MISTAKE — do not do this:
  WRONG: resolveSecretVaultKey: "user@example.com"   ← plaintext email, not a vault key
  WRONG: resolveSecretVaultKey: "mysecretpassword"   ← plaintext password, not a vault key
  RIGHT: resolveSecretVaultKey: "gmail-username"     ← key name returned by listSecrets()
  RIGHT: resolveSecretVaultKey: "gmail-password"     ← key name returned by listSecrets()
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

// sensitiveSuffixes is the list of substrings that make a vault key off-limits
// for direct SessionStorage access. Keys matching any of these can only flow
// through the onBeforeToolExecution hook (i.e. fill_secret), where the decrypted
// value is written directly to the browser and never returned to the LLM.
var sensitiveSuffixes = []string{"password", "token", "secret"}

// onContextBuild injects the resolveSecret function into SessionStorage so tools
// can programmatically decrypt secrets at runtime.
// Keys whose names contain "password", "token", or "secret" are blocked here —
// they may only be resolved through the onBeforeToolExecution hook (fill_secret).
func (p *VaultPlugin) onContextBuild(a *agents.Agent, agentContext *core.AgentContext) error {
	if agentContext.SessionStorage == nil {
		agentContext.SessionStorage = make(map[string]any)
	}

	agentContext.SessionStorage[sessionStorageKey] = func(key string) (string, error) {
		lower := strings.ToLower(key)
		for _, word := range sensitiveSuffixes {
			if strings.Contains(lower, word) {
				return "", fmt.Errorf("vault: direct decryption of key %q is not allowed — keys containing %q can only be used via fill_secret", key, word)
			}
		}
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
			return fmt.Errorf("vault: key '%s' not found for argument '%s'. Available keys: %v. Pass a key name from listSecrets(), not the actual secret value", vaultKey, k, p.listSecretKeys())
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

// deleteSecret removes a secret from the in-memory cache and from disk.
func (p *VaultPlugin) deleteSecret(key string) error {
	p.mu.Lock()
	if _, exists := p.secrets[key]; !exists {
		p.mu.Unlock()
		return fmt.Errorf("vault: secret '%s' not found", key)
	}
	delete(p.secrets, key)
	secretsCopy := make(map[string]string, len(p.secrets))
	for k, v := range p.secrets {
		secretsCopy[k] = v
	}
	p.mu.Unlock()

	return saveSecrets(p.dir, secretsCopy)
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

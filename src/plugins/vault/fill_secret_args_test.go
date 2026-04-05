package vault

import (
	"testing"

	"github.com/thinktwiceco/agent-forge/src/llms"
)

func TestNormalizeWebBrowserFillSecretVaultArg(t *testing.T) {
	tc := &llms.ToolCall{
		Name: "web_browser",
		Arguments: map[string]any{
			"action":                   "fill_secret",
			"resolve_secret_vault_key": "instagram-username",
		},
	}
	normalizeWebBrowserFillSecretVaultArg(tc)
	if _, ok := tc.Arguments["resolve_secret_vault_key"]; ok {
		t.Fatal("expected snake_case key to be removed after normalize")
	}
	got, ok := tc.Arguments["resolveSecretVaultKey"].(string)
	if !ok || got != "instagram-username" {
		t.Fatalf("resolveSecretVaultKey = %q, ok=%v", got, ok)
	}
}

func TestNormalizeWebBrowserFillSecretVaultArg_keepsCamelWhenBothWouldExist(t *testing.T) {
	tc := &llms.ToolCall{
		Name: "web_browser",
		Arguments: map[string]any{
			"action":                   "fill_secret",
			"resolveSecretVaultKey":    "primary",
			"resolve_secret_vault_key": "ignored",
		},
	}
	normalizeWebBrowserFillSecretVaultArg(tc)
	got, _ := tc.Arguments["resolveSecretVaultKey"].(string)
	if got != "primary" {
		t.Fatalf("want primary, got %q", got)
	}
}

func TestNormalizeWebBrowserFillSecretVaultArg_noOpOtherTools(t *testing.T) {
	tc := &llms.ToolCall{
		Name: "fs",
		Arguments: map[string]any{
			"resolve_secret_vault_key": "x",
		},
	}
	normalizeWebBrowserFillSecretVaultArg(tc)
	if _, ok := tc.Arguments["resolveSecretVaultKey"]; ok {
		t.Fatal("should not add resolveSecretVaultKey for non-web_browser tool")
	}
}

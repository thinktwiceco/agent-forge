package telegram

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// resetEnv clears relevant env vars and restores them after the test.
func resetEnv(t *testing.T) {
	t.Helper()
	origToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	origSecret := os.Getenv("WEBHOOK_SECRET_TELEGRAM")
	t.Cleanup(func() {
		_ = os.Setenv("TELEGRAM_BOT_TOKEN", origToken)
		_ = os.Setenv("WEBHOOK_SECRET_TELEGRAM", origSecret)
	})
	_ = os.Unsetenv("TELEGRAM_BOT_TOKEN")
	_ = os.Unsetenv("WEBHOOK_SECRET_TELEGRAM")
}

// --- register_token tests ---

func TestRegisterToken_MissingToken(t *testing.T) {
	result := actRegisterToken(map[string]any{})
	if result.Success() {
		t.Fatal("expected failure, got success")
	}
	if !strings.Contains(result.Error(), "token is required") {
		t.Errorf("expected 'token is required' error, got: %s", result.Error())
	}
}

func TestRegisterToken_InvalidToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":          false,
			"description": "Unauthorized",
		})
	}))
	defer srv.Close()

	origBase := telegramAPIBase
	telegramAPIBase = srv.URL
	defer func() { telegramAPIBase = origBase }()

	result := actRegisterToken(map[string]any{"token": "bad-token"})
	if result.Success() {
		t.Fatal("expected failure, got success")
	}
	if !strings.Contains(result.Error(), "Telegram API error") {
		t.Errorf("expected Telegram API error, got: %s", result.Error())
	}
}

func TestRegisterToken_Success(t *testing.T) {
	resetEnv(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok": true,
			"result": map[string]any{
				"id":         123456789,
				"is_bot":     true,
				"first_name": "TestBot",
				"username":   "test_bot",
			},
		})
	}))
	defer srv.Close()

	origBase := telegramAPIBase
	telegramAPIBase = srv.URL
	defer func() { telegramAPIBase = origBase }()

	result := actRegisterToken(map[string]any{"token": "valid-token"})
	if !result.Success() {
		t.Fatalf("expected success, got error: %s", result.Error())
	}
	if !strings.Contains(result.Data(), "Token registered successfully") {
		t.Errorf("expected success message, got: %s", result.Data())
	}
	if got := os.Getenv("TELEGRAM_BOT_TOKEN"); got != "valid-token" {
		t.Errorf("expected TELEGRAM_BOT_TOKEN=valid-token, got: %q", got)
	}
}

// --- start_ngrok tests ---

func TestStartNgrok_MissingToken(t *testing.T) {
	resetEnv(t)
	tool := &telegramTool{port: "8080"}
	result := tool.actStartNgrok(map[string]any{})
	if result.Success() {
		t.Fatal("expected failure, got success")
	}
	if !strings.Contains(result.Error(), "TELEGRAM_BOT_TOKEN is not set") {
		t.Errorf("expected missing token error, got: %s", result.Error())
	}
}

func TestStartNgrok_MissingWebhookSecret(t *testing.T) {
	resetEnv(t)
	_ = os.Setenv("TELEGRAM_BOT_TOKEN", "some-token")

	tool := &telegramTool{port: "8080"}
	result := tool.actStartNgrok(map[string]any{})
	if result.Success() {
		t.Fatal("expected failure, got success")
	}
	if !strings.Contains(result.Error(), "WEBHOOK_SECRET_TELEGRAM is required") {
		t.Errorf("expected webhook secret error, got: %s", result.Error())
	}
}

func TestStartNgrok_NgrokNotFound(t *testing.T) {
	resetEnv(t)
	_ = os.Setenv("TELEGRAM_BOT_TOKEN", "some-token")
	_ = os.Setenv("WEBHOOK_SECRET_TELEGRAM", "hook-secret")

	// Override PATH so ngrok cannot be found.
	origPath := os.Getenv("PATH")
	_ = os.Setenv("PATH", "")
	defer func() { _ = os.Setenv("PATH", origPath) }()

	tool := &telegramTool{port: "8080"}
	result := tool.actStartNgrok(map[string]any{})
	if result.Success() {
		t.Fatal("expected failure, got success")
	}
	if !strings.Contains(result.Error(), "ngrok not found") {
		t.Errorf("expected ngrok not found error, got: %s", result.Error())
	}
}

func TestStartNgrok_NgrokTimeout(t *testing.T) {
	// Serve an empty tunnels list so polling always returns nothing.
	ngrokSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"tunnels": []any{}})
	}))
	defer ngrokSrv.Close()

	origNgrok := ngrokAPIBase
	ngrokAPIBase = ngrokSrv.URL
	defer func() { ngrokAPIBase = origNgrok }()

	_, err := waitForNgrokTunnel()
	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Errorf("expected timeout error, got: %v", err)
	}
}

func TestStartNgrok_Success(t *testing.T) {
	resetEnv(t)
	_ = os.Setenv("TELEGRAM_BOT_TOKEN", "valid-token")
	_ = os.Setenv("WEBHOOK_SECRET_TELEGRAM", "hook-secret")

	// Mock Telegram Bot API for setWebhook.
	tgSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "description": "Webhook was set"})
	}))
	defer tgSrv.Close()

	// Mock ngrok local API returning an HTTPS tunnel URL.
	ngrokSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"tunnels": []map[string]any{
				{"public_url": "https://abc123.ngrok.io", "proto": "https"},
			},
		})
	}))
	defer ngrokSrv.Close()

	origTG := telegramAPIBase
	origNgrok := ngrokAPIBase
	telegramAPIBase = tgSrv.URL
	ngrokAPIBase = ngrokSrv.URL
	defer func() {
		telegramAPIBase = origTG
		ngrokAPIBase = origNgrok
	}()

	publicURL, err := waitForNgrokTunnel()
	if err != nil {
		t.Fatalf("waitForNgrokTunnel error: %v", err)
	}
	if publicURL != "https://abc123.ngrok.io" {
		t.Errorf("unexpected public URL: %s", publicURL)
	}

	if err := registerWebhook("valid-token", publicURL+"/api/webhooks/telegram", "hook-secret"); err != nil {
		t.Fatalf("registerWebhook error: %v", err)
	}
}

func TestRegisterWebhook_RequiresSecret(t *testing.T) {
	err := registerWebhook("t", "https://x/api/webhooks/telegram", "")
	if err == nil || !strings.Contains(err.Error(), "webhook secret is required") {
		t.Fatalf("expected secret required error, got: %v", err)
	}
}

func TestSetWebhook_MissingToken(t *testing.T) {
	resetEnv(t)
	tool := &telegramTool{port: "8080"}
	r := tool.actSetWebhook(map[string]any{})
	if r.Success() || !strings.Contains(r.Error(), "TELEGRAM_BOT_TOKEN is not set") {
		t.Fatalf("unexpected: %v / %s", r.Success(), r.Error())
	}
}

func TestSetWebhook_MissingSecret(t *testing.T) {
	resetEnv(t)
	_ = os.Setenv("TELEGRAM_BOT_TOKEN", "tok")
	tool := &telegramTool{port: "8080"}
	r := tool.actSetWebhook(map[string]any{})
	if r.Success() || !strings.Contains(r.Error(), "WEBHOOK_SECRET_TELEGRAM is required") {
		t.Fatalf("unexpected: %v / %s", r.Success(), r.Error())
	}
}

func TestSetWebhook_WithPublicURL(t *testing.T) {
	resetEnv(t)
	_ = os.Setenv("TELEGRAM_BOT_TOKEN", "valid-token")
	_ = os.Setenv("WEBHOOK_SECRET_TELEGRAM", "hook-secret")

	tgSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer tgSrv.Close()

	origBase := telegramAPIBase
	telegramAPIBase = tgSrv.URL
	defer func() { telegramAPIBase = origBase }()

	tool := &telegramTool{port: "8080"}
	r := tool.actSetWebhook(map[string]any{"webhook_public_url": "https://abc.ngrok.io"})
	if !r.Success() {
		t.Fatalf("expected success: %s", r.Error())
	}
	if !strings.Contains(r.Data(), "Webhook updated") {
		t.Errorf("unexpected: %s", r.Data())
	}
}

func TestSetWebhook_InvalidPublicURL(t *testing.T) {
	resetEnv(t)
	_ = os.Setenv("TELEGRAM_BOT_TOKEN", "valid-token")
	_ = os.Setenv("WEBHOOK_SECRET_TELEGRAM", "hook-secret")

	tool := &telegramTool{port: "8080"}
	r := tool.actSetWebhook(map[string]any{"webhook_public_url": "http://insecure.example"})
	if r.Success() || !strings.Contains(r.Error(), "https://") {
		t.Fatalf("unexpected: %v / %s", r.Success(), r.Error())
	}
}

// --- health_status tests ---

func TestHealthStatus_NgrokUp(t *testing.T) {
	resetEnv(t)

	ngrokSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tunnels" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"tunnels": []any{}})
	}))
	defer ngrokSrv.Close()

	origNgrok := ngrokAPIBase
	ngrokAPIBase = ngrokSrv.URL
	defer func() { ngrokAPIBase = origNgrok }()

	result := actHealthStatus()
	if !result.Success() {
		t.Fatalf("expected success, got: %s", result.Error())
	}
	if !strings.Contains(result.Data(), "ngrok local API: up") {
		t.Errorf("expected ngrok up, got: %s", result.Data())
	}
	if !strings.Contains(result.Data(), "TELEGRAM_BOT_TOKEN: missing") {
		t.Errorf("expected token missing, got: %s", result.Data())
	}
	if !strings.Contains(result.Data(), "WEBHOOK_SECRET_TELEGRAM: missing") {
		t.Errorf("expected webhook secret missing, got: %s", result.Data())
	}
	if !strings.Contains(result.Data(), "no HTTPS tunnel") {
		t.Errorf("expected no HTTPS tunnel line, got: %s", result.Data())
	}
	if !strings.Contains(result.Data(), "[BRAIN]:") {
		t.Errorf("expected [BRAIN] nudges when unhealthy, got: %s", result.Data())
	}
}

func TestHealthStatus_AllGreen_NoBrainNudge(t *testing.T) {
	resetEnv(t)
	_ = os.Setenv("TELEGRAM_BOT_TOKEN", "t")
	_ = os.Setenv("WEBHOOK_SECRET_TELEGRAM", "s")

	ngrokSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"tunnels": []any{
				map[string]any{"public_url": "https://abc.ngrok.io", "proto": "https"},
			},
		})
	}))
	defer ngrokSrv.Close()

	origNgrok := ngrokAPIBase
	ngrokAPIBase = ngrokSrv.URL
	defer func() { ngrokAPIBase = origNgrok }()

	result := actHealthStatus()
	if !result.Success() {
		t.Fatalf("expected success, got: %s", result.Error())
	}
	if strings.Contains(result.Data(), "[BRAIN]:") {
		t.Errorf("did not expect [BRAIN] when token, secret, and HTTPS tunnel are ok: %s", result.Data())
	}
	if !strings.Contains(result.Data(), "ngrok local API: up") {
		t.Errorf("expected ngrok up: %s", result.Data())
	}
}

func TestHealthStatus_SecretsSetNoHTTPS_StartNgrokPermissionNudge(t *testing.T) {
	resetEnv(t)
	_ = os.Setenv("TELEGRAM_BOT_TOKEN", "t")
	_ = os.Setenv("WEBHOOK_SECRET_TELEGRAM", "s")

	ngrokSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"tunnels": []any{}})
	}))
	defer ngrokSrv.Close()

	origNgrok := ngrokAPIBase
	ngrokAPIBase = ngrokSrv.URL
	defer func() { ngrokAPIBase = origNgrok }()

	result := actHealthStatus()
	if !result.Success() {
		t.Fatalf("expected success, got: %s", result.Error())
	}
	if !strings.Contains(result.Data(), "start_ngrok") {
		t.Errorf("expected start_ngrok permission nudge: %s", result.Data())
	}
	if !strings.Contains(result.Data(), "save_short_term_memory") {
		t.Errorf("expected save_short_term_memory hint: %s", result.Data())
	}
}

func TestHealthStatus_NgrokDown_InvalidJSON(t *testing.T) {
	resetEnv(t)

	ngrokSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("not-json"))
	}))
	defer ngrokSrv.Close()

	origNgrok := ngrokAPIBase
	ngrokAPIBase = ngrokSrv.URL
	defer func() { ngrokAPIBase = origNgrok }()

	result := actHealthStatus()
	if !result.Success() {
		t.Fatalf("expected success, got: %s", result.Error())
	}
	if !strings.Contains(result.Data(), "ngrok local API: down") {
		t.Errorf("expected ngrok down, got: %s", result.Data())
	}
	if !strings.Contains(result.Data(), "[BRAIN]:") {
		t.Errorf("expected [BRAIN] nudges when ngrok down, got: %s", result.Data())
	}
}

func TestHealthStatus_NgrokDown_BadStatus(t *testing.T) {
	resetEnv(t)

	ngrokSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer ngrokSrv.Close()

	origNgrok := ngrokAPIBase
	ngrokAPIBase = ngrokSrv.URL
	defer func() { ngrokAPIBase = origNgrok }()

	result := actHealthStatus()
	if !result.Success() {
		t.Fatalf("expected success, got: %s", result.Error())
	}
	if !strings.Contains(result.Data(), "HTTP 503") {
		t.Errorf("expected HTTP 503 in output, got: %s", result.Data())
	}
}

func TestHealthStatus_TokenSet(t *testing.T) {
	resetEnv(t)
	_ = os.Setenv("TELEGRAM_BOT_TOKEN", "secret-token")

	ngrokSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"tunnels": []any{}})
	}))
	defer ngrokSrv.Close()

	origNgrok := ngrokAPIBase
	ngrokAPIBase = ngrokSrv.URL
	defer func() { ngrokAPIBase = origNgrok }()

	result := actHealthStatus()
	if !result.Success() {
		t.Fatalf("expected success, got: %s", result.Error())
	}
	if !strings.Contains(result.Data(), "TELEGRAM_BOT_TOKEN: set") {
		t.Errorf("expected token set, got: %s", result.Data())
	}
	if strings.Contains(result.Data(), "secret-token") {
		t.Error("response must not contain the token value")
	}
}

func TestHealthStatus_Handler(t *testing.T) {
	resetEnv(t)

	ngrokSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"tunnels": []any{}})
	}))
	defer ngrokSrv.Close()

	origNgrok := ngrokAPIBase
	ngrokAPIBase = ngrokSrv.URL
	defer func() { ngrokAPIBase = origNgrok }()

	tool := &telegramTool{port: "8080"}
	tr := tool.handler(nil, map[string]any{"action": "health_status"})
	if !tr.Success() {
		t.Fatalf("expected success, got: %s", tr.Error())
	}
	if !strings.Contains(tr.Data(), "ngrok local API: up") {
		t.Errorf("unexpected: %s", tr.Data())
	}
}

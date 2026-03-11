# Authentication & Security — Public Deployment Plan

## Current State

The server is a Gin-based HTTP API with **no authentication**. It is designed for local/trusted-network use. The following risks exist if exposed to a public network:

| Risk | Location |
|---|---|
| All endpoints unauthenticated | Every route |
| API keys sent + stored over plain HTTP | `PUT /api/config/providers` |
| Postgres connection string returned in plaintext | `GET /api/config` |
| Config rewrite without auth (changes `workingDir`) | `PUT /api/config` |
| Arbitrary file write via upload | `POST /api/upload` |
| Wildcard CORS (`*`) | `corsMiddleware()` |
| Webhook signature verification silently skipped | `handlers_providers.go` |
| Symlink escape possible in FS endpoints | `GET /api/fs/*` |
| No TLS | `server.go` |

---

## Design Principles

1. **Backward compatible** — Auth is `disabled` by default. Local usage is unchanged.
2. **No external dependencies** — No Redis, no LDAP, no OAuth provider required. Self-contained Go.
3. **Defense in depth** — TLS + Auth + Filesystem sandbox + Secret masking, each independently valuable.
4. **Single-user first** — Personal agent tool. One admin user configured via env vars (not `config.yaml`, which is committed to git).

---

## Layer 1 — TLS

Add TLS support to `server.go`. Two modes:

**Manual certs** — operator provides files (e.g. from certbot):

```bash
AF_TLS_ENABLED=true
AF_TLS_CERT_FILE=/etc/letsencrypt/live/myagent.example.com/fullchain.pem
AF_TLS_KEY_FILE=/etc/letsencrypt/live/myagent.example.com/privkey.pem
```

**Auto TLS** — uses `golang.org/x/crypto/acme/autocert` to fetch + renew Let's Encrypt certificates automatically:

```bash
AF_TLS_AUTO_DOMAIN=myagent.example.com
AF_TLS_ACME_EMAIL=me@example.com
```

When TLS is enabled, a plain HTTP listener on port 80 responds only with an HTTP→HTTPS redirect. All other traffic is rejected.

---

## Layer 2 — Authentication

### Mechanism: Session Cookie + Bearer Token

Two parallel auth paths to support browser UI and programmatic/API access:

```
Request
  ├── Has Authorization: Bearer <token>?
  │     └── Hash token → compare SHA-256 to stored hash → ALLOW / 401
  └── Has session cookie?
        └── Lookup session in memory store → ALLOW / redirect to /login
```

### Configuration (env vars, not config.yaml)

```bash
AUTH_USERNAME=admin
AUTH_PASSWORD_HASH=$2a$12$...     # bcrypt hash of your password
AUTH_SESSION_TTL=24h              # optional, default 24h
AUTH_COOKIE_NAME=localforge_session  # optional
AUTH_COOKIE_SECURE=true           # optional; auto-detected from TLS/X-Forwarded-Proto
```

To generate a bcrypt hash for your password:

```bash
# Using Go (from the project root):
go run - <<'GO'
package main
import ("fmt"; "golang.org/x/crypto/bcrypt")
func main() {
  h, _ := bcrypt.GenerateFromPassword([]byte("your-password"), bcrypt.DefaultCost)
  fmt.Println(string(h))
}
GO
```

### Session Store

Pure in-memory, no external state. Sessions are keyed by `SHA-256(token)` — the raw token is never stored on the server.

```go
// cmd/localforge/src/auth/session_store.go

type Session struct {
    TokenHash string        // SHA-256 of the token presented to client
    UserID    string
    CreatedAt time.Time
    ExpiresAt time.Time
    IPAddr    string        // logged only, not enforced (VPN/mobile)
}

type SessionStore struct {
    mu       sync.RWMutex
    sessions map[string]*Session
}
```

A background goroutine sweeps expired sessions every hour. Sessions do not survive a server restart (by design — forces re-login).

### Auth Endpoints (exempt from middleware)

```
POST /api/auth/login     { username, password } → Set-Cookie: session=<token>; HttpOnly; Secure; SameSite=Strict
POST /api/auth/logout    → Delete cookie, remove session from store
GET  /api/auth/me        → { username, expiresAt } or 401
GET  /login              → Serves login.html
```

### Auth Middleware

Applied to all routes except `/login`, `/api/auth/login`, and `/api/auth/logout`.

```go
// cmd/localforge/src/auth/middleware.go

func AuthMiddleware(store *SessionStore, cfg *AuthConfig) gin.HandlerFunc {
    return func(c *gin.Context) {
        if !cfg.Enabled {
            c.Next()
            return
        }
        // 1. Bearer token (API / programmatic)
        if token := extractBearer(c); token != "" {
            if verifySHA256(token, cfg.APIKeyHash) {
                c.Set("auth_method", "api_key")
                c.Next()
                return
            }
            c.AbortWithStatusJSON(401, gin.H{"error": "invalid token"})
            return
        }
        // 2. Session cookie (browser)
        if cookie, err := c.Cookie("session"); err == nil {
            if session := store.Get(cookie); session != nil {
                c.Set("session", session)
                c.Next()
                return
            }
        }
        // 3. Reject
        if isAPIPath(c.Request.URL.Path) {
            c.AbortWithStatusJSON(401, gin.H{"error": "unauthorized"})
        } else {
            c.Redirect(302, "/login?next="+url.QueryEscape(c.Request.URL.Path))
        }
    }
}
```

### Login Rate Limiting

Simple per-IP in-memory counter on `POST /api/auth/login`:

- 5 failed attempts per IP per 15-minute window → `429 Too Many Requests`
- Counter resets on successful login
- No library required — ~50 lines of Go

---

## Layer 3 — CORS Hardening

Replace the wildcard CORS with an explicit allowlist when auth is enabled:

```bash
AF_CORS_ORIGINS=https://myagent.example.com,https://admin.example.com
```

When `AF_AUTH_ENABLED=false` the current wildcard behavior is preserved for local dev. When auth is enabled and `AF_CORS_ORIGINS` is unset, CORS is denied entirely (same-origin only).

---

## Layer 4 — Filesystem Sandbox

The existing traversal check (`strings.HasPrefix(absPath, workingDir)`) is correct but has one gap: **symlinks can escape the sandbox**. A symlink inside `workingDir` pointing to `/etc/passwd` passes the prefix check before resolution.

**Hardened helper** (replaces the current inline check):

```go
// cmd/localforge/src/auth/safe_path.go

func SafePath(workingDir, requestedPath string) (string, error) {
    joined := filepath.Join(workingDir, filepath.Clean("/"+requestedPath))
    // Resolve symlinks BEFORE the sandbox check
    resolved, err := filepath.EvalSymlinks(joined)
    if err != nil {
        return "", err
    }
    // Ensure trailing separator to prevent /workingdir-evil prefix match
    sandboxPrefix := workingDir + string(os.PathSeparator)
    if !strings.HasPrefix(resolved+string(os.PathSeparator), sandboxPrefix) {
        return "", ErrPathEscape
    }
    return resolved, nil
}
```

Applied to: `GET /api/fs/list`, `GET /api/fs/read`.

**Upload hardening:**

- Uploaded files must resolve under `workingDir/uploaded/` specifically (not just `workingDir`)
- Add a configurable max file size (`AF_UPLOAD_MAX_BYTES`, default 50 MB)
- Block writes to protected filenames regardless of upload path: `config.yaml`, `.env`, `*.db`, `*.key`

---

## Layer 5 — Secret Masking

Currently `GET /api/config` returns `PostgresURL` verbatim. Fix: always redact in API responses.

**Masking rules:**

| Field | Browser session | Bearer API key client |
|---|---|---|
| `postgresURL` | `postgresql://***:***@host/db` (host only) | Full value |
| Provider API keys | `••••••••1234` (last 4) | Full value |
| System prompt | Returned as-is | Returned as-is |

The distinction: a Bearer token client is a trusted machine caller (e.g. CI pipeline managing config). A browser session may be vulnerable to XSS, so secrets are masked there.

---

## Layer 6 — Webhook Secret Enforcement

Currently, if `WEBHOOK_SECRET_<PROVIDER>` is absent, signature verification is **silently skipped**. When `AF_AUTH_ENABLED=true`, make the webhook secret **required**:

```go
if cfg.AuthEnabled && secret == "" {
    c.AbortWithStatusJSON(403, gin.H{
        "error": "webhook secret not configured — set WEBHOOK_SECRET_<PROVIDER>",
    })
    return
}
```

---

## Files to Create / Modify

### New files

| File | Purpose |
|---|---|
| `cmd/localforge/src/auth/session_store.go` | In-memory session CRUD with expiry sweep |
| `cmd/localforge/src/auth/middleware.go` | Gin auth middleware (session + bearer) |
| `cmd/localforge/src/auth/login_limiter.go` | Per-IP brute-force rate limiter |
| `cmd/localforge/src/auth/safe_path.go` | Symlink-aware filesystem sandbox helper |
| `cmd/localforge/src/handlers_auth.go` | `/api/auth/*` and `/login` handlers |
| `cmd/localforge/src/static/login.html` | Login page (matches existing UI style) |

### Modified files

| File | Change |
|---|---|
| `server.go` | Wire auth middleware; TLS startup; CORS allowlist; register `/api/auth/*` and `/login` routes |
| `config_manager.go` | Add `AuthConfig` struct; load from env vars at startup |
| `types.go` | Add `AuthConfig`, `Session`, `SessionStore` types |
| `handlers_config.go` | Mask `PostgresURL`; full reveal only for Bearer clients |
| `handlers_providers.go` | Same masking logic; enforce webhook secret requirement |

---

## Deployment Reference

Minimum `.env` for public deployment:

```bash
# Auth (required to enable; auth is disabled if either is missing)
AUTH_USERNAME=admin
AUTH_PASSWORD_HASH=$2a$12$...    # bcrypt hash of your password
AUTH_SESSION_TTL=24h             # optional, default 24h
AUTH_COOKIE_SECURE=true          # set if not behind a TLS-terminating proxy

# Webhook secrets (per-provider; optional but strongly recommended)
WEBHOOK_SECRET_GITHUB=...
WEBHOOK_SECRET_STRIPE=...
WEBHOOK_SECRET_TELEGRAM=...
```

Local usage: omit `AUTH_USERNAME` and `AUTH_PASSWORD_HASH` → auth disabled, all routes open.

---

## Out of Scope (future work)

- Multi-user support with per-user conversation isolation
- OAuth2 / OIDC login (Google, GitHub)
- Audit log (who called what, when)
- Read-only role (view conversations/knowledge, no config writes)

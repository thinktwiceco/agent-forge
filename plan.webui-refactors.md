# Plan: Web UI Refactors (post-efficiency)

Status: **implemented**

## Goal

Apply the seven refactor proposals from the post-efficiency review without behavior changes.

## Scope

1. Split `chat.js` — extract `sse.js`, `tool-format.js`, `chat-turns.js`
2. Unify SSE chunk event routing in `ChatManager`
3. Extract persistent SSE reconnect loop (`subscribeSSE`)
4. Move title loading to `JSONConversationMetadata`
5. Add `GetTailHistory` to `JSONPersistence`
6. Identity-aware `ConversationRegistry` (mirror push registry)
7. Extract server-side SSE stream loop in `sse.go`

## Non-goals

- Turn-queue refactors
- Changing persistence interface
- SSE push for todos

## Validation

- `go test ./cmd/localforge/src/...`
- `go test ./src/persistence/...` (new tail/title tests)

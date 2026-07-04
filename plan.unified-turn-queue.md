# Plan: Unified Turn Queue

## Goal

Route **all agent turns** (web chat, webhooks, heartbeat, scheduler) through a single turn queue before the executor/LLM loop. Use **per-`chatId` serialization** so different conversations can run in parallel while the same conversation stays ordered.

## Feature type

- **Prompt / agent flow** — `src/agents/*`, `src/queue/*`
- **Plugin** — heartbeat, scheduler (`InboxAware` wiring)
- **Architecture docs** — `docs/agents/architecture.md`

## Scope

### In scope

1. Add `TurnRequest` and `TurnQueue` in `src/queue/` with global ingress and per-`chatId` mutex scheduling.
2. Extract `executeTurn` from `ChatStream`; worker goroutine is the only path into the executor.
3. `ChatStream` enqueues a turn and returns `ResponseCh` immediately (API unchanged).
4. Plugins (heartbeat, scheduler) enqueue via the same `TurnQueue` using existing `Enqueue(body, chatId, headers)` shape.
5. Autonomous turns (no external consumer) route chunks via existing `chunkRouter` / `turnCompleteRouter`.
6. Cancel queued turns when `TurnRequest.Ctx` is cancelled.
7. Unit tests for queue scheduling, ordering, and cancel-while-queued.
8. Update `docs/agents/architecture.md` and `src/plugins/README.md`.

### Non-goals

- Priority queue (user > heartbeat) — defer to follow-up.
- Localforge SSE "queued" status chunk — defer; silent wait is acceptable for v1.
- Serializing brain dreaming LLM calls (bypasses agent layer today).
- Changing `spawn_subagent` — child agents are separate `Agent` instances with their own queue.
- Removing `Agent.Drain()` — keep for now but document that it bypasses the unified queue (test/advanced use only) OR reimplement as external ingress to the same `TurnQueue` (preferred: wire `Drain` to submit turns).

## Current behavior (baseline)

- `ChatStream` starts executor immediately in a goroutine.
- Internal `inbox` (`queue.New(64)`) + `startBackgroundDrain` serializes plugin messages only.
- Direct `ChatStream` and inbox drain can run **concurrently**.
- Heartbeat skips when `inbox.Len() > 0`.

## Target architecture

```
Producers → TurnQueue.Submit(TurnRequest) → dispatcher (per-chatId lock) → executeTurn → executor
```

- `ChatStream`: creates `ResponseCh`, submits turn with caller's `ctx`.
- Plugins: submit turn with `ResponseCh=nil`; worker routes output internally.
- One worker pool pattern: each submitted turn runs in its own goroutine but acquires a per-`chatId` mutex before `executeTurn`.

## Implementation sequence

### Step 1 — Inbox interface + agent turn queue (no `queue`→`core` import)

**Import cycle fix:** `TurnQueue` lives in `src/agents/turn_queue.go` (holds `*core.ResponseCh`). `src/queue` only defines:

```go
type Inbox interface {
    Enqueue(body, chatId string, headers map[string]string)
    Len() int
}
```

`agents.TurnQueue` implements `queue.Inbox` for plugins.

```go
type turnRequest struct {
    ID         string
    Ctx        context.Context
    Body       string
    ChatID     string
    ResponseCh *core.ResponseCh  // nil for autonomous turns
    Source     string
}
```

- `NewTurnQueue(bufSize int, handler func(context.Context, turnRequest))`
- `Submit(req turnRequest) error` — blocks when ingress full
- `Enqueue(body, chatId, headers)` — `FormatHeaders` + UUID if empty `chatId` (plugin compat)
- Dispatcher: goroutine per ingress item; `acquireChat(chatID)` mutex map
- Cancel while queued: if `req.Ctx.Done()`, fail `ResponseCh` and skip execution
- `Len()` = ingress depth only (document heartbeat semantic shift)

Files: create `src/queue/inbox.go`, `src/agents/turn_queue.go`, `src/agents/turn_queue_test.go`.

### Step 2 — Extract turn execution (`src/agents/agent_turn.go`)

Move from `agent.go`:
- `executeTurn(ctx, req TurnRequest)` — history setup, hooks, executor, save, final chatId chunk.
- `startTurnWorker()` — starts `TurnQueue` consumer callback that calls `executeTurn` and handles nil-`ResponseCh` routing (today's background-drain loop body).

Keep `agent.go` under ~400 LOC (currently 503).

Files: create `src/agents/agent_turn.go`, modify `src/agents/agent.go`.

### Step 3 — Wire agent construction

In `NewAgent`:
- Replace `inbox *queue.Queue` + `inboxCancel` with `turnQueue *TurnQueue`.
- `startTurnWorker()` registers `executeTurn` as handler; no separate background drain.
- `core.InboxAware.SetInbox(q queue.Inbox)` — inject `turnQueue`.
- On agent discard/reload: cancel turn worker context so goroutines exit cleanly.

**Hooks:** `newUserMessageEvent` / `chatStartEvent` move from enqueue-time `ChatStream` into **start of `executeTurn`** (same timing relative to executor as today).

Files: `src/core/interfaces.go`, `src/agents/agentInit.go`, `src/agents/agent.go`.

### Step 4 — Update plugins

- Heartbeat: no logic change if `Inbox` interface + `Len()` reflects pending work globally.
- Scheduler consumer: still calls `inbox.Enqueue` — works via `TurnQueue`.
- Update plugin tests that construct raw `queue.Queue` where needed.

Files: `src/plugins/heartbeat/plugin.go`, `src/plugins/scheduler/*.go`, tests.

### Step 5 — `ChatStream` becomes enqueue-only

```go
func (a *Agent) ChatStream(ctx, message, chatId) *ResponseCh {
    // pre-assign chatId UUID (unchanged)
    responseCh := core.NewResponseCh(...)
    a.turnQueue.Submit(TurnRequest{Ctx: ctx, Body: message, ChatID: chatId, ResponseCh: responseCh, Source: "direct"})
    return responseCh
}
```

Remove recursive `ChatStream` from old drain loop.

### Step 6 — `Drain()` integration

Reimplement `Drain(ctx, q *queue.Queue)`:
- Do **not** cancel the turn worker.
- Read external `q.C()`, submit each message via `turnQueue.Submit` with `ResponseCh=nil`, `Ctx=ctx`.
- Do not replace `a.inbox` / swap internal queue (plugins keep init-time `turnQueue` reference).

### Step 7 — Tests

- `src/agents/turn_queue_test.go`: FIFO per chatId; parallel different chatIds; cancel while queued; full buffer blocks.
- Cross-source ordering: direct `ChatStream` + plugin `Enqueue` same `chatId` interleaved — must serialize.
- `src/agents/agent_turn_test.go` or extend `agent_test.go`: ChatStream still returns streaming response.
- Update `src/plugins/heartbeat/plugin_test.go` busy-inbox test to use `TurnQueue`.

Run: `go test ./src/queue/... ./src/agents/... ./src/plugins/heartbeat/... ./src/plugins/scheduler/... -race`

### Step 8 — Docs

- `docs/agents/architecture.md`: replace dual-path diagram with unified turn queue.
- `src/plugins/README.md`: note `Inbox` interface and global turn admission.

## Files summary

| Action | File |
|--------|------|
| Create | `src/queue/inbox.go` |
| Create | `src/agents/turn_queue.go`, `src/agents/turn_queue_test.go` |
| Create | `src/agents/agent_turn.go`, `src/agents/agent_turn_test.go` |
| Modify | `src/agents/agent.go`, `src/agents/agentInit.go` |
| Modify | `src/core/interfaces.go` |
| Modify | `src/queue/queue.go` (optional `Inbox` interface) |
| Modify | `src/plugins/heartbeat/plugin.go`, tests |
| Modify | `src/plugins/scheduler/consumers.go`, `scheduler.go` |
| Modify | `docs/agents/architecture.md`, `src/plugins/README.md` |

## Risks and mitigations

| Risk | Mitigation |
|------|------------|
| Subagent deadlock | Child agents are separate instances — each has own `TurnQueue`. |
| Same-chatId history race | Per-chatId mutex before load/save. |
| Increased latency | Acceptable for v1; optional queued-status SSE later. |
| `agent.go` size | Extract to `agent_turn.go`. |
| Heartbeat floods queue | Keep `Len() > 0` gate on `TurnQueue`. |

## Validation checklist

- [ ] `go test ./src/queue/... ./src/agents/... -race` passes
- [ ] Plugin tests pass
- [ ] Localforge chat + heartbeat still work (manual)
- [ ] Stop-chat cancellation still works via request ctx

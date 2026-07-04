package agents

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	agentforge "github.com/thinktwiceco/agent-forge/src"
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/queue"
)

// TurnHandler runs one agent turn. It is invoked by TurnQueue after per-chatId serialization.
type TurnHandler func(ctx context.Context, req turnRequest)

type turnRequest struct {
	ID         string
	Ctx        context.Context
	Body       string
	ChatID     string
	ResponseCh *core.ResponseCh
	Source     string
}

// TurnQueue is the single admission point for all agent turns.
// Different chatIds may run concurrently; the same chatId is processed FIFO by a dedicated worker.
type TurnQueue struct {
	ingress      chan turnRequest
	handler      TurnHandler
	workerCtx    context.Context
	workerCancel context.CancelFunc

	mu     sync.Mutex
	chatCh map[string]chan turnRequest
}

// NewTurnQueue creates a turn queue. Call Start after the handler's agent dependencies are ready.
func NewTurnQueue(bufSize int, handler TurnHandler) *TurnQueue {
	ctx, cancel := context.WithCancel(context.Background())
	return &TurnQueue{
		ingress:      make(chan turnRequest, bufSize),
		handler:      handler,
		workerCtx:    ctx,
		workerCancel: cancel,
		chatCh:       make(map[string]chan turnRequest),
	}
}

// Start launches the ingress dispatcher goroutine.
func (tq *TurnQueue) Start() {
	go tq.runDispatcher()
}

// Stop shuts down the dispatcher and per-chat workers.
func (tq *TurnQueue) Stop() {
	tq.workerCancel()
}

func (tq *TurnQueue) runDispatcher() {
	for {
		select {
		case <-tq.workerCtx.Done():
			return
		case req, ok := <-tq.ingress:
			if !ok {
				return
			}
			if err := req.Ctx.Err(); err != nil {
				tq.failCancelled(req, err)
				continue
			}
			ch := tq.chatIngress(req.ChatID)
			select {
			case <-tq.workerCtx.Done():
				return
			case ch <- req:
			}
		}
	}
}

func (tq *TurnQueue) chatIngress(chatID string) chan turnRequest {
	tq.mu.Lock()
	defer tq.mu.Unlock()
	if ch, ok := tq.chatCh[chatID]; ok {
		return ch
	}
	ch := make(chan turnRequest, 64)
	tq.chatCh[chatID] = ch
	go tq.runChatLoop(ch)
	return ch
}

func (tq *TurnQueue) runChatLoop(ch chan turnRequest) {
	for {
		select {
		case <-tq.workerCtx.Done():
			return
		case req, ok := <-ch:
			if !ok {
				return
			}
			if err := req.Ctx.Err(); err != nil {
				tq.failCancelled(req, err)
				continue
			}
			tq.handler(req.Ctx, req)
		}
	}
}

func (tq *TurnQueue) failCancelled(req turnRequest, err error) {
	if req.ResponseCh == nil {
		return
	}
	req.ResponseCh.Error <- err
	req.ResponseCh.Close()
}

// Submit enqueues a turn. Blocks when the buffer is full.
func (tq *TurnQueue) Submit(req turnRequest) error {
	if req.ID == "" {
		req.ID = uuid.NewString()
	}
	select {
	case <-tq.workerCtx.Done():
		return fmt.Errorf("turn queue stopped")
	case tq.ingress <- req:
		return nil
	}
}

// submitSpawnResult submits a subagent completion turn and logs submit failures.
func (tq *TurnQueue) submitSpawnResult(body, chatId string, headers map[string]string) error {
	if chatId == "" {
		return fmt.Errorf("submitSpawnResult: empty chatId")
	}
	source := ""
	if headers != nil {
		source = headers["sender"]
	}
	err := tq.Submit(turnRequest{
		Ctx:    context.Background(),
		Body:   queue.FormatHeaders(body, headers),
		ChatID: chatId,
		Source: source,
	})
	if err != nil {
		spawnID := ""
		if headers != nil {
			spawnID = headers["spawn_id"]
		}
		agentforge.Debug("[spawn] submitSpawnResult failed spawn_id=%s chatId=%s: %v", spawnID, chatId, err)
	}
	return err
}

// Enqueue implements queue.Inbox for plugin compatibility.
func (tq *TurnQueue) Enqueue(body, chatId string, headers map[string]string) {
	if chatId == "" {
		chatId = uuid.NewString()
	}
	source := ""
	if headers != nil {
		source = headers["sender"]
	}
	_ = tq.Submit(turnRequest{
		Ctx:    context.Background(),
		Body:   queue.FormatHeaders(body, headers),
		ChatID: chatId,
		Source: source,
	})
}

// Len returns the number of turns waiting in the ingress buffer (not in-flight or per-chat buffers).
func (tq *TurnQueue) Len() int {
	return len(tq.ingress)
}

var _ queue.Inbox = (*TurnQueue)(nil)

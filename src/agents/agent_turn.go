package agents

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	agentforge "github.com/thinktwiceco/agent-forge/src"
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
	"github.com/thinktwiceco/agent-forge/src/queue"
	"github.com/thinktwiceco/agent-forge/src/sessionlog"
)

func (a *Agent) startTurnWorker() {
	a.turnQueue.Start()
}

func (a *Agent) handleTurn(ctx context.Context, req turnRequest) {
	responseCh := req.ResponseCh
	if responseCh == nil {
		responseCh = core.NewResponseCh(a.config.AgentName, a.config.Trace, req.ChatID, a.onChunkReadHook())
		go a.executeTurn(ctx, req.Body, req.ChatID, responseCh)

		var sb strings.Builder
		finalChatId := req.ChatID
		for chunk := range responseCh.Start() {
			a.routeChunk(chunk)
			if chunk.Content != "" {
				sb.WriteString(chunk.Content)
			}
			if chunk.ChatId != "" {
				finalChatId = chunk.ChatId
			}
		}
		if a.turnCompleteRouter != nil && sb.Len() > 0 {
			a.turnCompleteRouter(finalChatId, sb.String())
		}
		return
	}

	a.executeTurn(ctx, req.Body, req.ChatID, responseCh)
}

func (a *Agent) onChunkReadHook() func(*core.ExtendedChunkResponse) error {
	return func(extendedChunk *core.ExtendedChunkResponse) error {
		errs := a.hooks.newChunkEvent(a, extendedChunk)
		for _, err := range errs {
			if err != nil {
				return err
			}
		}
		return nil
	}
}

func (a *Agent) executeTurn(ctx context.Context, message, chatId string, responseCh *core.ResponseCh) {
	releaseSessionLog := sessionlog.BindTurn(chatId, a.config.WorkingDir, a.config.AgentName)
	defer releaseSessionLog()
	defer responseCh.Close()

	hm := a.createHistory(chatId)
	a.injectSystemPrompt(hm)
	prevMessages := append([]*llms.UnifiedMessage(nil), hm.Messages()...)
	hm.AddUserMessage(message)
	errs := a.hooks.newUserMessageEvent(a, message)
	logHookErrors(errs)

	errs = a.hooks.chatStartEvent(a, chatId)
	logHookErrors(errs)

	execResult, err := a.executor.ExecuteChatWithTools(ctx, hm, responseCh)
	if err != nil {
		responseCh.Error <- err
	} else if execResult.HeartbeatAckSuppressed {
		hm.SetMessages(prevMessages)
	}

	var finalChatId string
	if err == nil && !execResult.HeartbeatAckSuppressed {
		saveId, saveErr := hm.Save()
		if saveErr != nil {
			agentforge.Error("Failed to save history: %v", saveErr)
		}
		finalChatId = saveId
		responseCh.SetChatId(finalChatId)
	}

	if finalChatId != "" {
		chatIdChunk := core.ExtendedChunkResponse{
			ChatId:    finalChatId,
			Status:    llms.StatusCompleted,
			Type:      llms.TypeCompletion,
			Content:   "",
			AgentName: a.config.AgentName,
			Trace:     a.config.Trace,
		}
		if chunkBytes, err := json.Marshal(chatIdChunk); err == nil {
			responseCh.TrySend(chunkBytes)
		}
	}
}

// Drain reads messages from an external queue and submits them through the turn queue,
// blocking until each turn completes. It returns when ctx is cancelled or q is closed.
func (a *Agent) Drain(ctx context.Context, q *queue.Queue) {
	a.toolsMu.RLock()
	a.executor.UpdateTools(a.tools)
	a.toolsMu.RUnlock()

	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-q.C():
			if !ok {
				return
			}
			chatId := msg.ChatId
			if chatId == "" {
				chatId = uuid.NewString()
			}
			responseCh := core.NewResponseCh(a.config.AgentName, a.config.Trace, chatId, a.onChunkReadHook())
			if err := a.turnQueue.Submit(turnRequest{
				Ctx:        ctx,
				Body:       msg.Format(),
				ChatID:     chatId,
				ResponseCh: responseCh,
				Source:     "drain",
			}); err != nil {
				return
			}
			for chunk := range responseCh.Start() {
				a.routeChunk(chunk)
			}
		}
	}
}

package agents

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	agentforge "github.com/thinktwiceco/agent-forge/src"
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
	"github.com/thinktwiceco/agent-forge/src/plugins/registry"
	"github.com/thinktwiceco/agent-forge/src/tools/spawn"
)

// subagentReservedToolNames are always provided by the subagent builder (system tools + todo plugin).
var subagentReservedToolNames = map[string]struct{}{
	"meta":            {},
	"expand":          {},
	"spawn_subagent":  {},
	"todo_handler":    {},
}

func filterToolsForSubagent(tools []llms.Tool) []llms.Tool {
	filtered := make([]llms.Tool, 0, len(tools))
	for _, t := range tools {
		if _, reserved := subagentReservedToolNames[t.GetName()]; reserved {
			continue
		}
		filtered = append(filtered, t)
	}
	return dedupeToolsByName(filtered)
}

func dedupeToolsByName(tools []llms.Tool) []llms.Tool {
	seen := make(map[string]struct{}, len(tools))
	out := make([]llms.Tool, 0, len(tools))
	for _, t := range tools {
		name := t.GetName()
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, t)
	}
	return out
}

// chatStreamDirect runs a turn immediately without enqueueing on TurnQueue.
// Used by spawn_subagent so the child agent does not wait behind the parent's queued turns.
// Completion is delivered back to the parent via TurnQueue.submitSpawnResult.
func (a *Agent) chatStreamDirect(ctx context.Context, message, chatId string) *core.ResponseCh {
	if chatId == "" {
		chatId = uuid.NewString()
	}
	responseCh := core.NewResponseCh(a.config.AgentName, a.config.Trace, chatId, a.onChunkReadHook())
	go a.executeTurn(ctx, message, chatId, responseCh)
	return responseCh
}

// drainSubagentResponse collects the final text from a subagent stream.
func drainSubagentResponse(stream <-chan core.ExtendedChunkResponse) (string, error) {
	var sb strings.Builder
	var full string
	for chunk := range stream {
		if chunk.Status == llms.StatusError {
			if chunk.Content != "" {
				return "", fmt.Errorf("%s", chunk.Content)
			}
			return "", fmt.Errorf("subagent stream failed")
		}
		if chunk.FullContent != "" {
			full = chunk.FullContent
		}
		sb.WriteString(chunk.Content)
		sb.WriteString(chunk.Delta)
	}
	if full != "" {
		return full, nil
	}
	text := strings.TrimSpace(sb.String())
	if text == "" {
		return "", fmt.Errorf("subagent returned empty response")
	}
	return text, nil
}

type spawnJob struct {
	spawnID      string
	parentChatID string
	cancel       context.CancelFunc
}

func (a *Agent) newAsyncSubagentSpawner() spawn.AsyncSubagentSpawner {
	return a.spawnSubagentAsync
}

func (a *Agent) buildSubagent(tools []llms.Tool) (*Agent, error) {
	b := NewBuilder(a.llmEngine, "subagent").
		WithCanExpand(true).
		WithTools(filterToolsForSubagent(tools)...)

	if todoFactory, err := registry.Get("todo"); err == nil {
		b = b.WithPlugins(todoFactory(a.config.WorkingDir))
	}

	return b.Build()
}

func (a *Agent) spawnSubagentAsync(parentChatID, prompt string, tools []llms.Tool) (string, error) {
	if parentChatID == "" {
		return "", fmt.Errorf("spawn_subagent requires an active conversation chatId")
	}

	sub, err := a.buildSubagent(tools)
	if err != nil {
		return "", err
	}

	spawnID := uuid.NewString()
	jobCtx, cancel := context.WithCancel(context.Background())

	a.spawnMu.Lock()
	if a.spawnRegistry == nil {
		a.spawnRegistry = make(map[string]spawnJob)
	}
	a.spawnRegistry[spawnID] = spawnJob{
		spawnID:      spawnID,
		parentChatID: parentChatID,
		cancel:       cancel,
	}
	a.spawnMu.Unlock()

	go a.runSpawnJob(jobCtx, spawnID, parentChatID, prompt, sub)

	return spawnID, nil
}

func (a *Agent) removeSpawnJob(spawnID string) {
	a.spawnMu.Lock()
	delete(a.spawnRegistry, spawnID)
	a.spawnMu.Unlock()
}

func (a *Agent) runSpawnJob(ctx context.Context, spawnID, parentChatID, prompt string, sub *Agent) {
	defer sub.Stop()
	defer a.removeSpawnJob(spawnID)

	stream := sub.chatStreamDirect(ctx, prompt, "").Start()
	text, err := drainSubagentResponse(stream)

	if ctx.Err() != nil {
		agentforge.Debug("[spawn] job %s cancelled, skip submit", spawnID)
		return
	}

	headers := map[string]string{
		"sender":    "subagent",
		"task_type": "subagent_result",
		"spawn_id":  spawnID,
	}

	if err != nil {
		headers["status"] = "error"
		body := err.Error()
		if a.turnQueue == nil {
			agentforge.Debug("[spawn] job %s completed with error but turn queue is nil", spawnID)
			return
		}
		_ = a.turnQueue.submitSpawnResult(body, parentChatID, headers)
		return
	}

	headers["status"] = "success"
	if a.turnQueue == nil {
		agentforge.Debug("[spawn] job %s completed but turn queue is nil", spawnID)
		return
	}
	_ = a.turnQueue.submitSpawnResult(text, parentChatID, headers)
}

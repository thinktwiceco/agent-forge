package heartbeat

import (
	"context"
	"time"

	agentforge "github.com/thinktwiceco/agent-forge/src"
	"github.com/thinktwiceco/agent-forge/src/agents"
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/heartbeatack"
	"github.com/thinktwiceco/agent-forge/src/llms"
	"github.com/thinktwiceco/agent-forge/src/plugins/registry"
	"github.com/thinktwiceco/agent-forge/src/queue"
)

const pluginName = "heartbeat"

// HeartbeatPlugin enqueues a synthetic inbox message on a fixed interval so
// the agent can perform periodic check-ins without a human initiating the turn.
type HeartbeatPlugin struct {
	cfg        HeartbeatConfig
	workingDir string
	inbox      *queue.Queue
	cancel     context.CancelFunc
	manager    *HeartbeatManager
}

// NewHeartbeatPlugin creates a plugin instance with the given configuration.
func NewHeartbeatPlugin(cfg HeartbeatConfig) *HeartbeatPlugin {
	return &HeartbeatPlugin{cfg: cfg, manager: NewHeartbeatManager()}
}

// Name implements core.Plugin.
func (p *HeartbeatPlugin) Name() string { return pluginName }

// SetWorkingDir implements core.WorkingDirAware.
func (p *HeartbeatPlugin) SetWorkingDir(dir string) {
	p.workingDir = dir
	if dir == "" {
		return
	}
	created, err := ensureDefaultHeartbeatFile(dir)
	if err != nil {
		agentforge.Debug("[heartbeat] could not create %s: %v", heartbeatFilename, err)
		return
	}
	if created {
		agentforge.Debug("[heartbeat] created default %s", heartbeatFilename)
	}
}

// SetInbox implements core.InboxAware.
func (p *HeartbeatPlugin) SetInbox(q *queue.Queue) {
	p.inbox = q
}

// Tools implements core.ToolProvider.
func (p *HeartbeatPlugin) Tools() []llms.Tool {
	return []llms.Tool{newHeartbeatManagerTool(p.manager)}
}

// SystemPrompt implements core.PromptProvider.
func (p *HeartbeatPlugin) SystemPrompt() string {
	return `[HEARTBEAT]
- Heartbeat messages come from sender=heartbeat.
- Read HEARTBEAT.md for the current checklist.
- If nothing needs attention, reply exactly HEARTBEAT_OK.
- Do not repeat old tasks unless HEARTBEAT.md still lists them.
- Use the heartbeat_manager tool to add, remove, or list named recurring instructions.`
}

// Hooks implements core.HookProvider.
func (p *HeartbeatPlugin) Hooks() map[core.Event]core.AgentHookFn {
	return map[core.Event]core.AgentHookFn{
		// Start the ticker after the agent is fully initialised.
		core.EventAgentInitialized: agents.OnAgentInitializedHook(func(_ *agents.Agent) error {
			interval, err := parseInterval(p.cfg.Every)
			if err != nil || interval == 0 {
				agentforge.Debug("[heartbeat] disabled (every=%q)", p.cfg.Every)
				return nil
			}
			ctx, cancel := context.WithCancel(context.Background())
			p.cancel = cancel
			go p.runTicker(ctx, interval)
			agentforge.Debug("[heartbeat] started (every=%s)", interval)
			return nil
		}),
	}
}

// runTicker drives the periodic heartbeat loop until ctx is cancelled.
func (p *HeartbeatPlugin) runTicker(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case t := <-ticker.C:
			p.maybeFire(t)
		}
	}
}

// maybeFire evaluates all gates and enqueues a heartbeat message when all pass.
func (p *HeartbeatPlugin) maybeFire(now time.Time) {
	if p.cfg.ActiveHours != nil && !isWithinActiveHours(p.cfg.ActiveHours, now) {
		agentforge.Debug("[heartbeat] skipped: outside active hours")
		return
	}
	if p.inbox != nil && p.inbox.Len() > 0 {
		agentforge.Debug("[heartbeat] skipped: inbox busy")
		return
	}
	prompt, skip := resolvePrompt(p.cfg, p.workingDir)
	if skip {
		agentforge.Debug("[heartbeat] skipped: HEARTBEAT.md effectively empty")
		return
	}
	if p.inbox == nil {
		agentforge.Debug("[heartbeat] skipped: inbox not set")
		return
	}
	chatId := "heartbeat-" + now.UTC().Format("20060102-150405")
	p.inbox.Enqueue(prompt, chatId, map[string]string{
		"sender":    "heartbeat",
		"task_type": heartbeatack.HeartbeatTickHeader,
		"fired_at":  now.UTC().Format(time.RFC3339),
	})
	agentforge.Debug("[heartbeat] fired at %s", now.UTC().Format(time.RFC3339))
}

func init() {
	registry.Register(pluginName, func(workingDir string) core.Plugin {
		return NewHeartbeatPlugin(DefaultConfig())
	})
}

package scheduler

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	agentforge "github.com/thinktwiceco/agent-forge/src"
	"github.com/thinktwiceco/agent-forge/src/agents"
	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
	"github.com/thinktwiceco/agent-forge/src/plugins/registry"
	"github.com/thinktwiceco/agent-forge/src/queue"
)

const pluginName = "scheduler"

// SchedulerPlugin wires together the scheduler, tool, and agent lifecycle hooks.
type SchedulerPlugin struct {
	workingDir string
	inbox      *queue.Queue // stored during EventAgentInitialization, applied when scheduler is created
	sched      *scheduler
	ctx        context.Context
	cancel     context.CancelFunc
}

func NewSchedulerPlugin(workingDir string) *SchedulerPlugin {
	ctx, cancel := context.WithCancel(context.Background())
	return &SchedulerPlugin{
		workingDir: filepath.Join(workingDir, "scheduler"),
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Name implements core.Plugin.
func (p *SchedulerPlugin) Name() string {
	return pluginName
}

// SetWorkingDir implements core.WorkingDirAware.
func (p *SchedulerPlugin) SetWorkingDir(dir string) {
	p.workingDir = filepath.Join(dir, "scheduler")
}

// SetInbox implements core.InboxAware.
// Called during EventAgentInitialization before the scheduler is created;
// the reference is stored and forwarded once the scheduler is ready.
func (p *SchedulerPlugin) SetInbox(q *queue.Queue) {
	p.inbox = q
	if p.sched != nil {
		p.sched.setInbox(q)
	}
}

// Hooks implements core.HookProvider.
// The scheduler is created on EventAgentInitialized so that the working
// directory and inbox are guaranteed to be set first.
func (p *SchedulerPlugin) Hooks() map[core.Event]core.AgentHookFn {
	return map[core.Event]core.AgentHookFn{
		core.EventAgentInitialized: agents.OnAgentInitializedHook(func(_ *agents.Agent) error {
			if err := os.MkdirAll(p.workingDir, 0o755); err != nil {
				return fmt.Errorf("[scheduler] failed to create dir %s: %w", p.workingDir, err)
			}

			s, err := newScheduler(p.workingDir)
			if err != nil {
				return fmt.Errorf("[scheduler] init failed: %w", err)
			}
			p.sched = s
			p.sched.setInbox(p.inbox)

			agentforge.Debug("[scheduler] starting poller (dir=%s)", p.workingDir)
			s.start(p.ctx)
			return nil
		}),
	}
}

// Tools implements core.ToolProvider.
func (p *SchedulerPlugin) Tools() []llms.Tool {
	return []llms.Tool{newScheduleTool(p)}
}

// SystemPrompt implements core.PromptProvider.
func (p *SchedulerPlugin) SystemPrompt() string {
	return `[SCHEDULER]
- Tool: schedule
- Use to create a reminder that fires at a specific future time
- Parameters: message (reminder text), scheduled_at (RFC3339 datetime e.g. 2026-03-05T15:00:00Z), chat_id (optional)
- Returns a task_id for reference
- When the reminder fires you receive an inbox message: sender=scheduler, task_type=agent_reminder`
}

func init() {
	registry.Register(pluginName, func(workingDir string) core.Plugin {
		return NewSchedulerPlugin(workingDir)
	})
}

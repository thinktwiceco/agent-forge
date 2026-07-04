package scheduler

import (
	"fmt"
	"strconv"

	"github.com/thinktwiceco/agent-forge/src/queue"
)

// ConsumerFn processes a due task by routing it to the agent inbox.
type ConsumerFn func(task *Task, inbox queue.Inbox) error

func defaultConsumers() map[string]ConsumerFn {
	return map[string]ConsumerFn{
		TaskTypeAgentReminder: agentReminderConsumer,
	}
}

// agentReminderConsumer delivers the task payload as a message to the agent inbox.
// The agent receives it like any other async message (sender=scheduler).
func agentReminderConsumer(task *Task, inbox queue.Inbox) error {
	if inbox == nil {
		return fmt.Errorf("inbox is nil")
	}
	inbox.Enqueue(task.Payload, task.ChatID, map[string]string{
		"sender":    "scheduler",
		"task_type": TaskTypeAgentReminder,
		"task_id":   strconv.FormatInt(task.ID, 10),
	})
	return nil
}

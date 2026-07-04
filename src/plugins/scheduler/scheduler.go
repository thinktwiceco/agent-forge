package scheduler

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
	agentforge "github.com/thinktwiceco/agent-forge/src"
	"github.com/thinktwiceco/agent-forge/src/queue"
)

const (
	dbFileName   = "scheduler.db"
	pollInterval = 5 * time.Second
)

type scheduler struct {
	db        *sql.DB
	inbox     queue.Inbox
	consumers map[string]ConsumerFn
}

func newScheduler(workingDir string) (*scheduler, error) {
	dbPath := filepath.Join(workingDir, dbFileName)
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_foreign_keys=1")
	if err != nil {
		return nil, fmt.Errorf("failed to open scheduler db: %w", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to connect to scheduler db: %w", err)
	}
	if err := ensureTable(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &scheduler{
		db:        db,
		consumers: defaultConsumers(),
	}, nil
}

func (s *scheduler) setInbox(q queue.Inbox) {
	s.inbox = q
}

// start launches the polling goroutine. Stops when ctx is cancelled.
func (s *scheduler) start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(pollInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.poll()
			case <-ctx.Done():
				return
			}
		}
	}()
}

//nolint:unused // reserved for plugin shutdown
func (s *scheduler) close() {
	if s.db != nil {
		_ = s.db.Close()
	}
}

func (s *scheduler) poll() {
	if s.inbox == nil {
		return
	}
	tasks, err := pendingDueTasks(s.db)
	if err != nil {
		agentforge.Debug("[scheduler] poll error: %v", err)
		return
	}
	for _, task := range tasks {
		consumer, ok := s.consumers[task.TaskType]
		if !ok {
			agentforge.Debug("[scheduler] unknown task_type %q for task %d", task.TaskType, task.ID)
			_ = markTaskFailed(s.db, task.ID, fmt.Sprintf("unknown task_type: %s", task.TaskType))
			continue
		}
		if err := consumer(task, s.inbox); err != nil {
			agentforge.Debug("[scheduler] task %d failed: %v", task.ID, err)
			_ = markTaskFailed(s.db, task.ID, err.Error())
			continue
		}
		_ = markTaskExecuted(s.db, task.ID)
		agentforge.Debug("[scheduler] task %d (%s) executed", task.ID, task.TaskType)
	}
}

func (s *scheduler) schedule(taskType, payload, chatID string, scheduledAt time.Time) (int64, error) {
	return insertTask(s.db, &Task{
		TaskType:    taskType,
		Payload:     payload,
		ChatID:      chatID,
		ScheduledAt: scheduledAt,
	})
}

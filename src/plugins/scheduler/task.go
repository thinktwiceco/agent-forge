package scheduler

import (
	"database/sql"
	"fmt"
	"time"
)

const (
	TaskTypeAgentReminder = "agent_reminder"

	statusPending  = "pending"
	statusExecuted = "executed"
	statusFailed   = "failed"
)

// Task represents a scheduled job stored in SQLite.
type Task struct {
	ID          int64
	TaskType    string
	Payload     string
	ChatID      string
	ScheduledAt time.Time
	CreatedAt   time.Time
	ExecutedAt  *time.Time
	Status      string
	Error       string
}

func ensureTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS scheduler_tasks (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			task_type    TEXT    NOT NULL,
			payload      TEXT    NOT NULL,
			chat_id      TEXT    NOT NULL DEFAULT '',
			scheduled_at DATETIME NOT NULL,
			created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			executed_at  DATETIME,
			status       TEXT    NOT NULL DEFAULT 'pending',
			error        TEXT    NOT NULL DEFAULT ''
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create scheduler_tasks table: %w", err)
	}

	_, err = db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_scheduler_status_sched
		ON scheduler_tasks (status, scheduled_at)
	`)
	return err
}

func insertTask(db *sql.DB, t *Task) (int64, error) {
	res, err := db.Exec(`
		INSERT INTO scheduler_tasks (task_type, payload, chat_id, scheduled_at, status)
		VALUES (?, ?, ?, ?, ?)
	`, t.TaskType, t.Payload, t.ChatID, t.ScheduledAt.UTC().Format(time.RFC3339), statusPending)
	if err != nil {
		return 0, fmt.Errorf("failed to insert task: %w", err)
	}
	return res.LastInsertId()
}

func pendingDueTasks(db *sql.DB) ([]*Task, error) {
	rows, err := db.Query(`
		SELECT id, task_type, payload, chat_id, scheduled_at, created_at
		FROM scheduler_tasks
		WHERE status = ? AND scheduled_at <= ?
		ORDER BY scheduled_at ASC
	`, statusPending, time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return nil, fmt.Errorf("failed to query due tasks: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var tasks []*Task
	for rows.Next() {
		var t Task
		var scheduledAt, createdAt string
		if err := rows.Scan(&t.ID, &t.TaskType, &t.Payload, &t.ChatID, &scheduledAt, &createdAt); err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}
		t.ScheduledAt, _ = time.Parse(time.RFC3339, scheduledAt)
		t.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		t.Status = statusPending
		tasks = append(tasks, &t)
	}
	return tasks, rows.Err()
}

func markTaskExecuted(db *sql.DB, id int64) error {
	_, err := db.Exec(`
		UPDATE scheduler_tasks SET status = ?, executed_at = ? WHERE id = ?
	`, statusExecuted, time.Now().UTC().Format(time.RFC3339), id)
	return err
}

func markTaskFailed(db *sql.DB, id int64, errMsg string) error {
	_, err := db.Exec(`
		UPDATE scheduler_tasks SET status = ?, executed_at = ?, error = ? WHERE id = ?
	`, statusFailed, time.Now().UTC().Format(time.RFC3339), errMsg, id)
	return err
}

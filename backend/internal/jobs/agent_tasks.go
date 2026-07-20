package jobs

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/domain"
)

const (
	AgentTaskStatusOpen = "open"
	AgentTaskStatusDone = "done"

	AgentTaskKindReviewStrongMatch = "review_strong_match"
	AgentTaskKindDecideManualJob   = "decide_manual_job"
	AgentTaskKindInspectSource     = "inspect_source"
	AgentTaskKindRunCrawl          = "run_crawl"
)

type AgentTask struct {
	ID          int64      `json:"id"`
	TaskDate    string     `json:"task_date"`
	Kind        string     `json:"kind"`
	Title       string     `json:"title"`
	Detail      string     `json:"detail"`
	Status      string     `json:"status"`
	Priority    int        `json:"priority"`
	Count       int        `json:"count"`
	SubjectID   int64      `json:"subject_id"`
	JobID       int64      `json:"job_id"`
	SourceID    int64      `json:"source_id"`
	Action      string     `json:"action"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

type AgentTaskInput struct {
	TaskDate  string
	Kind      string
	Title     string
	Detail    string
	Priority  int
	Count     int
	SubjectID int64
	JobID     int64
	SourceID  int64
	Action    string
}

func (r *Repository) SyncAgentTasks(ctx context.Context, now time.Time) ([]AgentTask, error) {
	taskDate := agentTaskDate(now)
	desired, err := r.buildDesiredAgentTasks(ctx, taskDate)
	if err != nil {
		return nil, err
	}
	desiredKeys := map[string]bool{}
	for _, input := range desired {
		desiredKeys[agentTaskKey(input.Kind, input.SubjectID)] = true
		if err := r.upsertAgentTask(ctx, input); err != nil {
			return nil, err
		}
	}
	if err := r.closeStaleOpenAgentTasks(ctx, taskDate, desiredKeys); err != nil {
		return nil, err
	}
	return r.ListAgentTasks(ctx, taskDate)
}

func (r *Repository) ListAgentTasks(ctx context.Context, taskDate string) ([]AgentTask, error) {
	taskDate = strings.TrimSpace(taskDate)
	if taskDate == "" {
		taskDate = agentTaskDate(time.Now().UTC())
	}
	rows, err := r.db.QueryContext(ctx, selectAgentTaskSQL()+`
		WHERE task_date = ?
		ORDER BY status ASC, priority DESC, id ASC
	`, taskDate)
	if err != nil {
		return nil, fmt.Errorf("list agent tasks: %w", err)
	}
	defer rows.Close()

	tasks := []AgentTask{}
	for rows.Next() {
		task, err := scanAgentTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate agent tasks: %w", err)
	}
	return tasks, nil
}

func (r *Repository) UpdateAgentTaskStatus(ctx context.Context, id int64, status string) error {
	status = normalizeAgentTaskStatus(status)
	var completedAt any
	if status == AgentTaskStatusDone {
		completedAt = time.Now().UTC()
	}
	result, err := r.db.ExecContext(ctx, `
		UPDATE agent_tasks
		SET status = ?, completed_at = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, status, completedAt, id)
	if err != nil {
		return fmt.Errorf("update agent task status: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read rows affected: %w", err)
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *Repository) buildDesiredAgentTasks(ctx context.Context, taskDate string) ([]AgentTaskInput, error) {
	jobList, err := r.ListJobs(ctx, ListFilter{})
	if err != nil {
		return nil, err
	}
	sources, err := r.ListSources(ctx, false)
	if err != nil {
		return nil, err
	}
	runs, err := r.ListRuns(ctx)
	if err != nil {
		return nil, err
	}

	tasks := []AgentTaskInput{}
	for _, job := range jobList {
		if job.MatchScore >= 70 && job.Status == domain.StatusNew {
			tasks = append(tasks, AgentTaskInput{
				TaskDate:  taskDate,
				Kind:      AgentTaskKindReviewStrongMatch,
				Title:     "Review " + fallbackText(job.Company, "Unknown company"),
				Detail:    fallbackText(job.Title, "Untitled role") + " / " + fallbackText(job.City, "Unknown city"),
				Priority:  90 + job.MatchScore,
				Count:     1,
				SubjectID: job.ID,
				JobID:     job.ID,
				Action:    "review_strong_matches",
			})
		}
		if job.Status == domain.StatusManualCheck {
			tasks = append(tasks, AgentTaskInput{
				TaskDate:  taskDate,
				Kind:      AgentTaskKindDecideManualJob,
				Title:     "Decide " + fallbackText(job.Company, "Unknown company"),
				Detail:    fallbackText(job.Title, "Untitled role") + " / " + firstText(job.PenaltyReasons, "Needs classification"),
				Priority:  120,
				Count:     1,
				SubjectID: job.ID,
				JobID:     job.ID,
				Action:    "review_manual_check",
			})
		}
	}
	for _, source := range sources {
		if !source.Enabled || (source.HealthStatus != SourceHealthBroken && source.HealthStatus != SourceHealthWarning) {
			continue
		}
		tasks = append(tasks, AgentTaskInput{
			TaskDate:  taskDate,
			Kind:      AgentTaskKindInspectSource,
			Title:     "Inspect " + fallbackText(source.Name, "source"),
			Detail:    source.HealthStatus + " / " + fallbackText(source.HealthReason, "Source needs attention"),
			Priority:  200 + source.ConsecutiveFailures,
			Count:     1,
			SubjectID: source.ID,
			SourceID:  source.ID,
			Action:    "inspect_failed_sources",
		})
	}
	if len(runs) == 0 {
		tasks = append(tasks, AgentTaskInput{
			TaskDate:  taskDate,
			Kind:      AgentTaskKindRunCrawl,
			Title:     "Run the first crawl",
			Detail:    "Sources exist, but I have no crawl record to work from yet.",
			Priority:  80,
			Count:     1,
			SubjectID: 0,
			Action:    "run_crawl",
		})
	}
	sort.Slice(tasks, func(i, j int) bool {
		if tasks[i].Priority == tasks[j].Priority {
			return tasks[i].Title < tasks[j].Title
		}
		return tasks[i].Priority > tasks[j].Priority
	})
	return tasks, nil
}

func (r *Repository) upsertAgentTask(ctx context.Context, input AgentTaskInput) error {
	if input.Count <= 0 {
		input.Count = 1
	}
	input.Kind = strings.TrimSpace(input.Kind)
	input.Title = strings.TrimSpace(input.Title)
	input.Detail = strings.TrimSpace(input.Detail)
	input.Action = strings.TrimSpace(input.Action)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO agent_tasks (
			task_date, kind, title, detail, status, priority, count, subject_id,
			job_id, source_id, action
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(task_date, kind, subject_id) DO UPDATE SET
			title = excluded.title,
			detail = excluded.detail,
			priority = excluded.priority,
			count = excluded.count,
			job_id = excluded.job_id,
			source_id = excluded.source_id,
			action = excluded.action,
			updated_at = CURRENT_TIMESTAMP
	`, input.TaskDate, input.Kind, input.Title, input.Detail, AgentTaskStatusOpen,
		input.Priority, input.Count, input.SubjectID, input.JobID, input.SourceID, input.Action)
	if err != nil {
		return fmt.Errorf("upsert agent task: %w", err)
	}
	return nil
}

func (r *Repository) closeStaleOpenAgentTasks(ctx context.Context, taskDate string, desiredKeys map[string]bool) error {
	tasks, err := r.ListAgentTasks(ctx, taskDate)
	if err != nil {
		return err
	}
	for _, task := range tasks {
		if task.Status != AgentTaskStatusOpen || desiredKeys[agentTaskKey(task.Kind, task.SubjectID)] {
			continue
		}
		if err := r.UpdateAgentTaskStatus(ctx, task.ID, AgentTaskStatusDone); err != nil {
			return err
		}
	}
	return nil
}

func selectAgentTaskSQL() string {
	return `
		SELECT id, task_date, kind, title, detail, status, priority, count,
			subject_id, job_id, source_id, action, created_at, updated_at, completed_at
		FROM agent_tasks`
}

func scanAgentTask(scanner jobScanner) (AgentTask, error) {
	var task AgentTask
	if err := scanner.Scan(
		&task.ID,
		&task.TaskDate,
		&task.Kind,
		&task.Title,
		&task.Detail,
		&task.Status,
		&task.Priority,
		&task.Count,
		&task.SubjectID,
		&task.JobID,
		&task.SourceID,
		&task.Action,
		&task.CreatedAt,
		&task.UpdatedAt,
		&task.CompletedAt,
	); err != nil {
		return AgentTask{}, fmt.Errorf("scan agent task: %w", err)
	}
	task.Status = normalizeAgentTaskStatus(task.Status)
	return task, nil
}

func agentTaskDate(now time.Time) string {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return now.UTC().Format("2006-01-02")
}

func agentTaskKey(kind string, subjectID int64) string {
	return strings.TrimSpace(kind) + ":" + fmt.Sprint(subjectID)
}

func normalizeAgentTaskStatus(status string) string {
	switch strings.TrimSpace(status) {
	case AgentTaskStatusDone:
		return AgentTaskStatusDone
	default:
		return AgentTaskStatusOpen
	}
}

package jobs

import (
	"context"
	"testing"
	"time"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/db"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/domain"
)

func TestRepositorySyncAgentTasksBuildsDailyQueue(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo := NewRepository(conn)

	strong, err := repo.CreateJob(ctx, domain.Job{
		Company:    "Tencent",
		Title:      "Go Backend Engineer",
		City:       "Shenzhen",
		MatchScore: 88,
		Status:     domain.StatusNew,
	})
	if err != nil {
		t.Fatalf("seed strong job: %v", err)
	}
	manual, err := repo.CreateJob(ctx, domain.Job{
		Company:        "Portal",
		Title:          "Campus page",
		City:           "Unknown",
		MatchScore:     42,
		Status:         domain.StatusManualCheck,
		PenaltyReasons: []string{"Unclear city"},
	})
	if err != nil {
		t.Fatalf("seed manual job: %v", err)
	}
	source, err := repo.CreateSource(ctx, SourceInput{
		Name:    "Meituan Campus",
		URL:     "https://campus.meituan.com/",
		Enabled: true,
	})
	if err != nil {
		t.Fatalf("seed source: %v", err)
	}
	if err := repo.UpdateSourceHealthByURL(ctx, source.URL, SourceHealthInput{
		Status:  SourceHealthBroken,
		Reason:  "HTTP 502",
		Success: false,
	}); err != nil {
		t.Fatalf("mark source broken: %v", err)
	}

	tasks, err := repo.SyncAgentTasks(ctx, time.Date(2026, 7, 20, 9, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("sync tasks: %v", err)
	}

	assertTask(t, tasks, AgentTaskKindReviewStrongMatch, strong.ID)
	assertTask(t, tasks, AgentTaskKindDecideManualJob, manual.ID)
	assertTask(t, tasks, AgentTaskKindInspectSource, source.ID)
}

func TestRepositorySyncAgentTasksKeepsCompletedTasksDone(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo := NewRepository(conn)

	job, err := repo.CreateJob(ctx, domain.Job{
		Company:    "Tencent",
		Title:      "Go Backend Engineer",
		City:       "Shenzhen",
		MatchScore: 90,
		Status:     domain.StatusNew,
	})
	if err != nil {
		t.Fatalf("seed job: %v", err)
	}
	if _, err := repo.CreateRun(ctx, "manual", time.Date(2026, 7, 20, 8, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("seed run: %v", err)
	}
	now := time.Date(2026, 7, 20, 9, 0, 0, 0, time.UTC)
	tasks, err := repo.SyncAgentTasks(ctx, now)
	if err != nil {
		t.Fatalf("sync tasks: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected one task, got %#v", tasks)
	}
	if err := repo.UpdateAgentTaskStatus(ctx, tasks[0].ID, AgentTaskStatusDone); err != nil {
		t.Fatalf("complete task: %v", err)
	}

	tasks, err = repo.SyncAgentTasks(ctx, now.Add(2*time.Hour))
	if err != nil {
		t.Fatalf("resync tasks: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected completed task not duplicated, got %#v", tasks)
	}
	if tasks[0].Status != AgentTaskStatusDone || tasks[0].JobID != job.ID {
		t.Fatalf("expected completed job task to stay done, got %#v", tasks[0])
	}
}

func TestRepositoryEscalatesOpenTasksBySLA(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo := NewRepository(conn)
	if _, err := repo.CreateJob(ctx, domain.Job{
		Company:    "Tencent",
		Title:      "Go Backend Engineer",
		City:       "Shenzhen",
		MatchScore: 90,
		Status:     domain.StatusNew,
	}); err != nil {
		t.Fatalf("seed job: %v", err)
	}
	now := time.Date(2026, 7, 21, 12, 0, 0, 0, time.UTC)
	if _, err := repo.CreateRun(ctx, "manual", now.Add(-time.Hour)); err != nil {
		t.Fatalf("seed run: %v", err)
	}
	tasks, err := repo.SyncAgentTasks(ctx, now)
	if err != nil {
		t.Fatalf("sync tasks: %v", err)
	}
	if len(tasks) == 0 {
		t.Fatal("expected synced task")
	}
	if _, err := repo.db.ExecContext(ctx, `UPDATE agent_tasks SET created_at = ? WHERE id = ?`, now.Add(-5*time.Hour), tasks[0].ID); err != nil {
		t.Fatalf("age task: %v", err)
	}

	result, err := repo.EscalateAgentTasks(ctx, now, Settings{TaskSLAHours: 4})
	if err != nil {
		t.Fatalf("escalate tasks: %v", err)
	}
	if result.Stale != 1 || result.Escalated != 0 {
		t.Fatalf("expected one stale task, got %#v", result)
	}
	updated, err := repo.GetAgentTask(ctx, tasks[0].ID)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if updated.Status != AgentTaskStatusStale {
		t.Fatalf("expected stale task, got %#v", updated)
	}

	if _, err := repo.db.ExecContext(ctx, `UPDATE agent_tasks SET status = ?, created_at = ? WHERE id = ?`, AgentTaskStatusOpen, now.Add(-9*time.Hour), tasks[0].ID); err != nil {
		t.Fatalf("reset task age: %v", err)
	}
	result, err = repo.EscalateAgentTasks(ctx, now, Settings{TaskSLAHours: 4})
	if err != nil {
		t.Fatalf("escalate old task: %v", err)
	}
	if result.Escalated != 1 {
		t.Fatalf("expected one escalated task, got %#v", result)
	}
	updated, err = repo.GetAgentTask(ctx, tasks[0].ID)
	if err != nil {
		t.Fatalf("get escalated task: %v", err)
	}
	if updated.Status != AgentTaskStatusEscalated || updated.EscalatedAt == nil {
		t.Fatalf("expected escalated task with timestamp, got %#v", updated)
	}
}

func TestRepositorySnoozesTasksAndKeepsCompletionReason(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo := NewRepository(conn)
	if _, err := repo.CreateJob(ctx, domain.Job{
		Company:    "Tencent",
		Title:      "Go Backend Engineer",
		City:       "Shenzhen",
		MatchScore: 90,
		Status:     domain.StatusNew,
	}); err != nil {
		t.Fatalf("seed job: %v", err)
	}
	now := time.Date(2026, 7, 21, 12, 0, 0, 0, time.UTC)
	if _, err := repo.CreateRun(ctx, "manual", now.Add(-time.Hour)); err != nil {
		t.Fatalf("seed run: %v", err)
	}
	tasks, err := repo.SyncAgentTasks(ctx, now)
	if err != nil {
		t.Fatalf("sync tasks: %v", err)
	}
	snoozedUntil := now.Add(24 * time.Hour)
	if err := repo.UpdateAgentTask(ctx, tasks[0].ID, AgentTaskUpdate{
		Status:       AgentTaskStatusSnoozed,
		SnoozedUntil: &snoozedUntil,
	}); err != nil {
		t.Fatalf("snooze task: %v", err)
	}
	if _, err := repo.db.ExecContext(ctx, `UPDATE agent_tasks SET created_at = ? WHERE id = ?`, now.Add(-12*time.Hour), tasks[0].ID); err != nil {
		t.Fatalf("age task: %v", err)
	}
	result, err := repo.EscalateAgentTasks(ctx, now, Settings{TaskSLAHours: 4})
	if err != nil {
		t.Fatalf("escalate snoozed task: %v", err)
	}
	if result.Stale != 0 || result.Escalated != 0 {
		t.Fatalf("expected snoozed task not to escalate, got %#v", result)
	}
	updated, err := repo.GetAgentTask(ctx, tasks[0].ID)
	if err != nil {
		t.Fatalf("get snoozed task: %v", err)
	}
	if updated.Status != AgentTaskStatusSnoozed || updated.SnoozedUntil == nil {
		t.Fatalf("expected snoozed task, got %#v", updated)
	}

	if err := repo.UpdateAgentTask(ctx, tasks[0].ID, AgentTaskUpdate{
		Status:           AgentTaskStatusDone,
		CompletionReason: "Applied from career portal",
	}); err != nil {
		t.Fatalf("complete task: %v", err)
	}
	updated, err = repo.GetAgentTask(ctx, tasks[0].ID)
	if err != nil {
		t.Fatalf("get completed task: %v", err)
	}
	if updated.Status != AgentTaskStatusDone || updated.CompletionReason != "Applied from career portal" || updated.CompletedAt == nil {
		t.Fatalf("expected completed task with reason, got %#v", updated)
	}
}

func assertTask(t *testing.T, tasks []AgentTask, kind string, subjectID int64) {
	t.Helper()
	for _, task := range tasks {
		if task.Kind == kind && task.SubjectID == subjectID {
			if task.Status != AgentTaskStatusOpen {
				t.Fatalf("expected open task for %s/%d, got %#v", kind, subjectID, task)
			}
			return
		}
	}
	t.Fatalf("missing task kind=%s subject=%d in %#v", kind, subjectID, tasks)
}

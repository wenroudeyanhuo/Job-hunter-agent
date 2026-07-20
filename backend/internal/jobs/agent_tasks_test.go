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

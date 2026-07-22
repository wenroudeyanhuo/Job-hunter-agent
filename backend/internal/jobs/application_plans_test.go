package jobs

import (
	"context"
	"testing"
	"time"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/db"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/domain"
)

func TestRepositorySyncsApplicationPlansForInterestedStrongMatches(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo := NewRepository(conn)
	job, err := repo.CreateJob(ctx, domain.Job{
		Company:          "Tencent",
		Title:            "Go Backend Engineer",
		City:             "Shenzhen",
		DirectionTags:    []string{"go", "backend"},
		Description:      "Campus recruitment role with job description and apply online.",
		ApplyURL:         "https://careers.example.com/jobs/1",
		DiscoveredAt:     time.Date(2026, 7, 22, 9, 0, 0, 0, time.UTC),
		MatchScore:       88,
		RecommendReasons: []string{"Strong profile fit", "Clear application URL"},
		Status:           domain.StatusInterested,
	})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}

	plans, err := repo.SyncApplicationPlans(ctx, time.Date(2026, 7, 22, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("sync plans: %v", err)
	}
	if len(plans) != 1 {
		t.Fatalf("expected one application plan, got %#v", plans)
	}
	if plans[0].JobID != job.ID || plans[0].Status != ApplicationPlanStatusPrepare {
		t.Fatalf("unexpected plan: %#v", plans[0])
	}
	if plans[0].NextAction == "" || len(plans[0].Checklist) == 0 {
		t.Fatalf("expected plan guidance, got %#v", plans[0])
	}

	tasks, err := repo.SyncAgentTasks(ctx, time.Date(2026, 7, 22, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("sync tasks: %v", err)
	}
	if !containsTaskKind(tasks, AgentTaskKindPrepareApplication) {
		t.Fatalf("expected prepare application task, got %#v", tasks)
	}
}

func containsTaskKind(tasks []AgentTask, kind string) bool {
	for _, task := range tasks {
		if task.Kind == kind {
			return true
		}
	}
	return false
}

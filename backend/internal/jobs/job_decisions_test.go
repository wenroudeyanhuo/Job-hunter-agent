package jobs

import (
	"context"
	"testing"
	"time"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/db"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/domain"
)

func TestRepositoryRecordsJobStatusDecision(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo := NewRepository(conn)
	job := createDecisionTestJob(t, ctx, repo)

	if err := repo.UpdateStatus(ctx, job.ID, domain.StatusInterested); err != nil {
		t.Fatalf("update status: %v", err)
	}

	decisions, err := repo.ListJobDecisions(ctx, job.ID)
	if err != nil {
		t.Fatalf("list decisions: %v", err)
	}
	if len(decisions) != 1 {
		t.Fatalf("expected one decision, got %d", len(decisions))
	}
	if decisions[0].Action != "status_changed" || decisions[0].FromStatus != "new" || decisions[0].ToStatus != "interested" {
		t.Fatalf("unexpected decision: %#v", decisions[0])
	}
}

func TestRepositoryRecordsJobNoteDecision(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo := NewRepository(conn)
	job := createDecisionTestJob(t, ctx, repo)

	if err := repo.UpdateNotes(ctx, job.ID, "Resume should emphasize Go and distributed systems."); err != nil {
		t.Fatalf("update notes: %v", err)
	}

	decisions, err := repo.ListJobDecisions(ctx, job.ID)
	if err != nil {
		t.Fatalf("list decisions: %v", err)
	}
	if len(decisions) != 1 {
		t.Fatalf("expected one decision, got %d", len(decisions))
	}
	if decisions[0].Action != "notes_updated" || decisions[0].Notes != "Resume should emphasize Go and distributed systems." {
		t.Fatalf("unexpected decision: %#v", decisions[0])
	}
}

func createDecisionTestJob(t *testing.T, ctx context.Context, repo *Repository) domain.Job {
	t.Helper()
	job, err := repo.CreateJob(ctx, domain.Job{
		Company:      "Tencent",
		Title:        "Go Backend Engineer",
		City:         "Shenzhen",
		SourceName:   "test",
		SourceURL:    "https://example.com/jobs",
		ApplyURL:     "https://example.com/jobs/1",
		DiscoveredAt: time.Date(2026, 7, 21, 9, 0, 0, 0, time.UTC),
		MatchScore:   82,
		Status:       domain.StatusNew,
	})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}
	return job
}

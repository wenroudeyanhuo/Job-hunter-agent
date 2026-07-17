package crawl

import (
	"context"
	"errors"
	"testing"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/db"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/domain"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/jobs"
)

func TestRunnerContinuesWhenCollectorFails(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo := jobs.NewRepository(conn)
	runner := NewRunner(repo, []Collector{
		fakeCollector{name: "valid", jobs: []domain.Job{{
			Company:     "Tencent",
			Title:       "Go Backend Engineer 2027 Campus",
			City:        "Shenzhen",
			Description: "Campus recruitment for backend microservices with Go.",
			ApplyURL:    "https://example.com/apply",
			SourceName:  "valid",
			SourceURL:   "https://example.com/source",
		}}},
		fakeCollector{name: "broken", err: errors.New("network failed")},
	})

	summary, err := runner.Run(ctx, "manual")
	if err != nil {
		t.Fatalf("run crawler: %v", err)
	}
	if summary.SourcesTotal != 2 {
		t.Fatalf("expected 2 sources, got %d", summary.SourcesTotal)
	}
	if summary.SourcesSuccess != 1 {
		t.Fatalf("expected 1 success, got %d", summary.SourcesSuccess)
	}
	if summary.SourcesFailed != 1 {
		t.Fatalf("expected 1 failure, got %d", summary.SourcesFailed)
	}
	if summary.JobsCreated != 1 {
		t.Fatalf("expected 1 created job, got %d", summary.JobsCreated)
	}

	list, err := repo.ListJobs(ctx, jobs.ListFilter{})
	if err != nil {
		t.Fatalf("list jobs: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected one stored job, got %d", len(list))
	}
	if list[0].MatchScore == 0 {
		t.Fatal("expected job to be scored")
	}
}

type fakeCollector struct {
	name string
	jobs []domain.Job
	err  error
}

func (f fakeCollector) Name() string {
	return f.name
}

func (f fakeCollector) Collect(context.Context) ([]domain.Job, error) {
	return f.jobs, f.err
}

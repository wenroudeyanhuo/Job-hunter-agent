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
	if len(summary.RecommendedJobs) != 1 {
		t.Fatalf("expected one recommended job, got %d", len(summary.RecommendedJobs))
	}
	if summary.RecommendedJobs[0].Title != "Go Backend Engineer 2027 Campus" {
		t.Fatalf("unexpected recommended job: %#v", summary.RecommendedJobs[0])
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
	runs, err := repo.ListRuns(ctx)
	if err != nil {
		t.Fatalf("list runs: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("expected one run, got %d", len(runs))
	}
	sourceResults, err := repo.ListRunSources(ctx, runs[0].ID)
	if err != nil {
		t.Fatalf("list run source results: %v", err)
	}
	if len(sourceResults) != 2 {
		t.Fatalf("expected two source results, got %d", len(sourceResults))
	}
}

func TestRunnerUsesSavedSettingsForFiltering(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo := jobs.NewRepository(conn)
	settings := jobs.DefaultSettings()
	settings.ExcludedKeywords = []string{"remote-only"}
	if _, err := repo.SaveSettings(ctx, settings); err != nil {
		t.Fatalf("save settings: %v", err)
	}

	runner := NewRunner(repo, []Collector{
		fakeCollector{name: "valid", jobs: []domain.Job{{
			Company:     "Example",
			Title:       "Go Backend Engineer",
			City:        "Shenzhen",
			Description: "This is a remote-only contractor role.",
			ApplyURL:    "https://example.com/apply",
			SourceName:  "valid",
			SourceURL:   "https://example.com/source",
		}}},
	})

	summary, err := runner.Run(ctx, "manual")
	if err != nil {
		t.Fatalf("run crawler: %v", err)
	}
	if summary.JobsCreated != 0 {
		t.Fatalf("expected configured excluded keyword to prevent creation, got %d created", summary.JobsCreated)
	}
	runs, err := repo.ListRuns(ctx)
	if err != nil {
		t.Fatalf("list runs: %v", err)
	}
	sourceResults, err := repo.ListRunSources(ctx, runs[0].ID)
	if err != nil {
		t.Fatalf("list run source results: %v", err)
	}
	if len(sourceResults) != 1 || sourceResults[0].JobsFiltered != 1 {
		t.Fatalf("expected one filtered job source result, got %#v", sourceResults)
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

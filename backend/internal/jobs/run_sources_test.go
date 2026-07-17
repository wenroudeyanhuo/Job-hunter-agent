package jobs

import (
	"context"
	"testing"
	"time"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/db"
)

func TestRepositoryCreateAndListRunSourceResults(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo := NewRepository(conn)
	run, err := repo.CreateRun(ctx, "manual", time.Now().UTC())
	if err != nil {
		t.Fatalf("create run: %v", err)
	}

	created, err := repo.CreateRunSourceResult(ctx, RunSourceResultInput{
		JobRunID:         run.ID,
		SourceName:       "example.com",
		SourceURL:        "https://example.com",
		Status:           "success",
		JobsFound:        2,
		JobsCreated:      1,
		JobsDuplicated:   1,
		JobsFiltered:     0,
		ManualCheckCount: 0,
	})
	if err != nil {
		t.Fatalf("create run source result: %v", err)
	}
	if created.ID == 0 {
		t.Fatal("expected result ID")
	}

	results, err := repo.ListRunSources(ctx, run.ID)
	if err != nil {
		t.Fatalf("list run source results: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected one result, got %d", len(results))
	}
	if results[0].JobsCreated != 1 || results[0].JobsDuplicated != 1 {
		t.Fatalf("unexpected result counts: %#v", results[0])
	}
}

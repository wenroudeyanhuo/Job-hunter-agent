package jobs

import (
	"context"
	"testing"
	"time"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/db"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/domain"
)

func TestRepositoryCreateListAndUpdateStatus(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo := NewRepository(conn)

	created, err := repo.CreateJob(ctx, domain.Job{
		Company:          "Tencent",
		Title:            "Backend Engineer",
		City:             "Shenzhen",
		DirectionTags:    []string{"backend", "go"},
		SourceName:       "manual",
		SourceURL:        "https://example.com/source",
		ApplyURL:         "https://example.com/apply",
		DiscoveredAt:     time.Date(2026, 7, 17, 9, 0, 0, 0, time.UTC),
		MatchScore:       90,
		RecommendReasons: []string{"Shenzhen", "backend"},
		Status:           domain.StatusNew,
	})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}
	if created.ID == 0 {
		t.Fatal("expected created ID")
	}

	if err := repo.UpdateStatus(ctx, created.ID, domain.StatusInterested); err != nil {
		t.Fatalf("update status: %v", err)
	}

	list, err := repo.ListJobs(ctx, ListFilter{Status: domain.StatusInterested})
	if err != nil {
		t.Fatalf("list jobs: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected one job, got %d", len(list))
	}
	if list[0].Status != domain.StatusInterested {
		t.Fatalf("expected interested, got %s", list[0].Status)
	}
	if len(list[0].DirectionTags) != 2 || list[0].DirectionTags[1] != "go" {
		t.Fatalf("direction tags did not round trip: %#v", list[0].DirectionTags)
	}
	if len(list[0].RecommendReasons) != 2 {
		t.Fatalf("recommend reasons did not round trip: %#v", list[0].RecommendReasons)
	}
}

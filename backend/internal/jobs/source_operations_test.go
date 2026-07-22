package jobs

import (
	"context"
	"testing"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/db"
)

func TestRepositoryBuildsSourceOperationsSummary(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo := NewRepository(conn)

	if _, err := repo.CreateSource(ctx, SourceInput{Name: "Healthy Source", URL: "https://healthy.example.com/jobs", Enabled: true}); err != nil {
		t.Fatalf("create healthy source: %v", err)
	}
	broken, err := repo.CreateSource(ctx, SourceInput{Name: "Broken Source", URL: "https://broken.example.com/jobs", Enabled: true})
	if err != nil {
		t.Fatalf("create broken source: %v", err)
	}
	if err := repo.UpdateSourceHealthByURL(ctx, broken.URL, SourceHealthInput{Status: SourceHealthBroken, Reason: "HTTP 500", Success: false}); err != nil {
		t.Fatalf("mark source broken: %v", err)
	}
	if _, err := repo.DiscoverSourceCandidates(ctx, SourceDiscoveryInput{TargetCities: []string{"Shenzhen"}, TargetDirections: []string{"go"}}); err != nil {
		t.Fatalf("discover candidates: %v", err)
	}

	summary, err := repo.BuildSourceOperationsSummary(ctx)
	if err != nil {
		t.Fatalf("build source operations summary: %v", err)
	}
	if summary.TotalSources != 2 || summary.BrokenSources != 1 || summary.PendingCandidates == 0 {
		t.Fatalf("unexpected source operations summary: %#v", summary)
	}
	if len(summary.Actions) == 0 || summary.Actions[0].Type == "" {
		t.Fatalf("expected recommended source actions, got %#v", summary.Actions)
	}
}

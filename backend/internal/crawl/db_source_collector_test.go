package crawl

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/db"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/jobs"
)

func TestDBSourceCollectorReadsEnabledSources(t *testing.T) {
	sourceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><head><title>Tencent Go Backend Engineer 2027 Campus - Shenzhen</title></head></html>`))
	}))
	defer sourceServer.Close()

	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo := jobs.NewRepository(conn)
	if _, err := repo.CreateSource(context.Background(), jobs.SourceInput{
		Name:    "enabled",
		URL:     sourceServer.URL,
		Enabled: true,
	}); err != nil {
		t.Fatalf("create enabled source: %v", err)
	}
	if _, err := repo.CreateSource(context.Background(), jobs.SourceInput{
		Name:    "disabled",
		URL:     "https://example.com/disabled",
		Enabled: false,
	}); err != nil {
		t.Fatalf("create disabled source: %v", err)
	}

	collector := NewDBSourceCollector(repo, sourceServer.Client())
	collected, err := collector.Collect(context.Background())
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if len(collected) != 1 {
		t.Fatalf("expected one collected job, got %d", len(collected))
	}
	if collected[0].Title != "Tencent Go Backend Engineer 2027 Campus - Shenzhen" {
		t.Fatalf("unexpected title %q", collected[0].Title)
	}
}

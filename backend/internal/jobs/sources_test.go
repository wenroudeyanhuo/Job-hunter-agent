package jobs

import (
	"context"
	"testing"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/db"
)

func TestRepositoryCreateListAndToggleSource(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo := NewRepository(conn)

	created, err := repo.CreateSource(ctx, SourceInput{
		Name:       "Tencent Campus",
		Type:       "public_url",
		URL:        "https://example.com/tencent",
		Enabled:    true,
		ParserType: "generic",
	})
	if err != nil {
		t.Fatalf("create source: %v", err)
	}
	if created.ID == 0 {
		t.Fatal("expected source ID")
	}

	sources, err := repo.ListSources(ctx, false)
	if err != nil {
		t.Fatalf("list sources: %v", err)
	}
	if len(sources) != 1 {
		t.Fatalf("expected one source, got %d", len(sources))
	}
	if !sources[0].Enabled {
		t.Fatal("expected source to be enabled")
	}

	if err := repo.UpdateSourceEnabled(ctx, created.ID, false); err != nil {
		t.Fatalf("toggle source: %v", err)
	}
	enabled, err := repo.ListSources(ctx, true)
	if err != nil {
		t.Fatalf("list enabled: %v", err)
	}
	if len(enabled) != 0 {
		t.Fatalf("expected no enabled sources, got %d", len(enabled))
	}
}

func TestRepositorySeedSourcesDeduplicatesURLs(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo := NewRepository(conn)

	if err := repo.SeedPublicURLSources(ctx, []string{"https://example.com/a", "https://example.com/a"}); err != nil {
		t.Fatalf("seed sources: %v", err)
	}
	sources, err := repo.ListSources(ctx, false)
	if err != nil {
		t.Fatalf("list sources: %v", err)
	}
	if len(sources) != 1 {
		t.Fatalf("expected one deduped source, got %d", len(sources))
	}
}

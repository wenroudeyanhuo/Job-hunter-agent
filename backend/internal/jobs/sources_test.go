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

func TestRepositorySeedRecommendedSources(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo := NewRepository(conn)

	result, err := repo.SeedRecommendedSources(ctx)
	if err != nil {
		t.Fatalf("seed recommended sources: %v", err)
	}
	if result.Created == 0 {
		t.Fatal("expected recommended sources to be created")
	}
	if result.Total != len(RecommendedSources()) {
		t.Fatalf("expected total recommended sources, got %#v", result)
	}

	second, err := repo.SeedRecommendedSources(ctx)
	if err != nil {
		t.Fatalf("seed recommended sources again: %v", err)
	}
	if second.Created != 0 || second.Duplicated != result.Total {
		t.Fatalf("expected second seed to dedupe, got %#v", second)
	}
}

func TestRepositorySeedRecommendedSourcesRefreshesExistingParserTypes(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo := NewRepository(conn)

	if _, err := repo.CreateSource(ctx, SourceInput{
		Name:       "OPPO Careers",
		URL:        "https://careers.oppo.com/",
		Enabled:    true,
		ParserType: "generic",
	}); err != nil {
		t.Fatalf("create existing OPPO source: %v", err)
	}

	if _, err := repo.SeedRecommendedSources(ctx); err != nil {
		t.Fatalf("seed recommended sources: %v", err)
	}

	sources, err := repo.ListSources(ctx, false)
	if err != nil {
		t.Fatalf("list sources: %v", err)
	}
	for _, source := range sources {
		if source.Name == "OPPO Careers" {
			if source.ParserType != "oppo_api" {
				t.Fatalf("expected OPPO parser to be refreshed, got %q", source.ParserType)
			}
			return
		}
	}
	t.Fatal("expected OPPO Careers source")
}

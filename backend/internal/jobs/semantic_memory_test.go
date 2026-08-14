package jobs

import (
	"context"
	"strings"
	"testing"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/db"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/domain"
)

func TestRepositoryRebuildsAndSearchesSemanticJobMemory(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo := NewRepository(conn)
	if _, err := repo.CreateJob(ctx, domain.Job{
		Company:       "Tencent",
		Title:         "Go Backend Platform Engineer",
		City:          "Shenzhen",
		DirectionTags: []string{"go", "backend"},
		Description:   "Build distributed service platform and storage systems.",
		MatchScore:    88,
		Status:        domain.StatusNew,
	}); err != nil {
		t.Fatalf("create backend job: %v", err)
	}
	if _, err := repo.CreateJob(ctx, domain.Job{
		Company:       "Pixel Studio",
		Title:         "Frontend Design Engineer",
		City:          "Guangzhou",
		DirectionTags: []string{"frontend"},
		Description:   "Build animation-heavy web experience.",
		MatchScore:    54,
		Status:        domain.StatusNew,
	}); err != nil {
		t.Fatalf("create frontend job: %v", err)
	}

	result, err := repo.RebuildSemanticMemory(ctx)
	if err != nil {
		t.Fatalf("rebuild semantic memory: %v", err)
	}
	if result.Created != 2 {
		t.Fatalf("expected two memory items, got %#v", result)
	}

	matches, err := repo.SearchSemanticMemory(ctx, SemanticMemoryQuery{Query: "golang backend distributed service in shenzhen", Limit: 3})
	if err != nil {
		t.Fatalf("search semantic memory: %v", err)
	}
	if len(matches) == 0 {
		t.Fatal("expected memory matches")
	}
	if matches[0].Kind != SemanticMemoryKindJob || matches[0].ReferenceID == 0 {
		t.Fatalf("expected top match to reference a job, got %#v", matches[0])
	}
	if !strings.Contains(strings.ToLower(matches[0].Title), "backend") {
		t.Fatalf("expected backend job as top match, got %#v", matches[0])
	}
	if matches[0].Score <= 0 {
		t.Fatalf("expected positive similarity score, got %#v", matches[0])
	}
}

func TestBuildAgentMemoryIncludesSemanticStoreStats(t *testing.T) {
	stats := SemanticMemoryStats{TotalItems: 4, JobItems: 3, Provider: "local_hash", Dimension: 64}

	memory := BuildAgentMemoryWithSemanticStats(nil, nil, stats)

	if memory.SemanticTotalItems != 4 || memory.SemanticJobItems != 3 {
		t.Fatalf("expected semantic memory stats, got %#v", memory)
	}
	if memory.SemanticProvider != "local_hash" || memory.SemanticDimension != 64 {
		t.Fatalf("expected semantic provider details, got %#v", memory)
	}
}

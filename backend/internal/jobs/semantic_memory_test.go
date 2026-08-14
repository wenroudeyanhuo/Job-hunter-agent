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

func TestRepositoryIndexesSemanticMemoryWhenJobIsCreated(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo := NewRepository(conn)

	created, err := repo.CreateJob(ctx, domain.Job{
		Company:       "DeepAI",
		Title:         "Agent Platform Go Engineer",
		City:          "Shenzhen",
		DirectionTags: []string{"go", "ai_application"},
		Description:   "Develop agent runtime, tool orchestration, and backend platform services.",
		MatchScore:    93,
		Status:        domain.StatusNew,
	})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}

	stats, err := repo.GetSemanticMemoryStats(ctx)
	if err != nil {
		t.Fatalf("get semantic stats: %v", err)
	}
	if stats.JobItems != 1 {
		t.Fatalf("expected created job to be indexed automatically, got %#v", stats)
	}
	matches, err := repo.SearchSemanticMemory(ctx, SemanticMemoryQuery{Query: "agent runtime go backend", Limit: 3})
	if err != nil {
		t.Fatalf("search semantic memory: %v", err)
	}
	if len(matches) == 0 || matches[0].ReferenceID != created.ID {
		t.Fatalf("expected created job memory to be searchable, got %#v", matches)
	}
}

func TestRepositoryRefreshesSemanticMemoryWhenJobChanges(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo := NewRepository(conn)
	created, err := repo.CreateJob(ctx, domain.Job{
		Company:       "InfraWorks",
		Title:         "Backend Engineer",
		City:          "Shenzhen",
		DirectionTags: []string{"backend"},
		Description:   "Build service reliability platform.",
		MatchScore:    75,
		Status:        domain.StatusNew,
	})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}

	if err := repo.UpdateStatus(ctx, created.ID, domain.StatusInterested); err != nil {
		t.Fatalf("update status: %v", err)
	}
	if err := repo.UpdateNotes(ctx, created.ID, "Emphasize Kubernetes, Go, and distributed tracing experience."); err != nil {
		t.Fatalf("update notes: %v", err)
	}

	item, err := repo.GetSemanticMemoryItemByReference(ctx, SemanticMemoryKindJob, created.ID)
	if err != nil {
		t.Fatalf("get memory item: %v", err)
	}
	if item.Metadata["status"] != string(domain.StatusInterested) {
		t.Fatalf("expected semantic metadata status to refresh, got %#v", item.Metadata)
	}
	if !strings.Contains(item.Content, "distributed tracing") {
		t.Fatalf("expected semantic memory content to include updated notes, got %q", item.Content)
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

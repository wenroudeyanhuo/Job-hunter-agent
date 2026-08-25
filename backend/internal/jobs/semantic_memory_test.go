package jobs

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

func TestRepositoryUsesQdrantSemanticMemoryWhenConfigured(t *testing.T) {
	ctx := context.Background()
	var upsertCalled bool
	var searchCalled bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/collections/job_hunter_test_memory":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"result":true}`))
		case r.Method == http.MethodPut && r.URL.Path == "/collections/job_hunter_test_memory/points":
			upsertCalled = true
			if r.URL.Query().Get("wait") != "true" {
				t.Fatalf("expected qdrant upsert to wait, got query %q", r.URL.RawQuery)
			}
			var payload struct {
				Points []struct {
					ID      int64          `json:"id"`
					Vector  []float64      `json:"vector"`
					Payload map[string]any `json:"payload"`
				} `json:"points"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode qdrant upsert: %v", err)
			}
			if len(payload.Points) != 1 || len(payload.Points[0].Vector) != SemanticMemoryDimensions {
				t.Fatalf("expected one qdrant point with embedding, got %#v", payload.Points)
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"result":{"operation_id":1,"status":"completed"}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/collections/job_hunter_test_memory/points/search":
			searchCalled = true
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"result":[{"id":7,"score":0.91,"payload":{"kind":"job","reference_id":42,"title":"Qdrant Go Backend","content":"Go backend in Shenzhen","metadata":{"company":"QdrantCo"},"embedding_provider":"qdrant","embedding_dimension":64,"updated_at":"2026-08-25T09:00:00Z"}}]}`))
		default:
			t.Fatalf("unexpected qdrant request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()
	t.Setenv("SEMANTIC_MEMORY_PROVIDER", SemanticMemoryProviderQdrant)
	t.Setenv("QDRANT_URL", server.URL)
	t.Setenv("QDRANT_COLLECTION", "job_hunter_test_memory")

	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo := NewRepository(conn)
	if _, err := repo.UpsertSemanticMemoryItem(ctx, SemanticMemoryItem{
		Kind:        SemanticMemoryKindJob,
		ReferenceID: 42,
		Title:       "Qdrant Go Backend",
		Content:     "Go backend in Shenzhen",
		Metadata:    map[string]string{"company": "QdrantCo"},
	}); err != nil {
		t.Fatalf("upsert semantic memory: %v", err)
	}
	if !upsertCalled {
		t.Fatal("expected qdrant upsert to be called")
	}

	matches, err := repo.SearchSemanticMemory(ctx, SemanticMemoryQuery{Query: "go backend shenzhen", Kind: SemanticMemoryKindJob, Limit: 3})
	if err != nil {
		t.Fatalf("search semantic memory: %v", err)
	}
	if !searchCalled {
		t.Fatal("expected qdrant search to be called")
	}
	if len(matches) != 1 || matches[0].ReferenceID != 42 || matches[0].Score != 0.91 {
		t.Fatalf("expected qdrant match to be returned, got %#v", matches)
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

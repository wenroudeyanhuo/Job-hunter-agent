package jobs

import (
	"testing"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/domain"
)

func TestSemanticMemoryUsesProviderBoundary(t *testing.T) {
	item := SemanticMemoryItemFromJobWithProvider(domain.Job{
		ID:          7,
		Company:     "Tencent",
		Title:       "Go Backend Engineer",
		City:        "Shenzhen",
		Description: "Distributed systems and AI application platform.",
	}, NewHashEmbeddingProvider())

	if item.EmbeddingProvider != SemanticMemoryProviderLocalHash {
		t.Fatalf("expected local hash provider, got %#v", item)
	}
	if item.EmbeddingDimension != SemanticMemoryDimensions || len(item.Embedding) != SemanticMemoryDimensions {
		t.Fatalf("expected provider dimension to round trip, got %#v", item)
	}
	if item.Title == "" || item.Content == "" {
		t.Fatalf("expected job memory content, got %#v", item)
	}
}

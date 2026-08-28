package jobs

import (
	"context"
	"strings"
	"testing"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/db"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/domain"
)

func TestRepositoryRefreshesMemoryReflectionsFromUserDecisions(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo := NewRepository(conn)

	backendJob, err := repo.CreateJob(ctx, domain.Job{
		Company:       "Tencent",
		Title:         "Go Backend Engineer",
		City:          "Shenzhen",
		DirectionTags: []string{"go", "backend"},
		MatchScore:    88,
		Status:        domain.StatusNew,
	})
	if err != nil {
		t.Fatalf("create backend job: %v", err)
	}
	trainingJob, err := repo.CreateJob(ctx, domain.Job{
		Company:       "Training Co",
		Title:         "Java Training Assistant",
		City:          "Remote",
		DirectionTags: []string{"java"},
		MatchScore:    20,
		Status:        domain.StatusNew,
	})
	if err != nil {
		t.Fatalf("create training job: %v", err)
	}
	if _, err := repo.RecordJobDecision(ctx, JobDecisionInput{JobID: backendJob.ID, Action: "mark_interested", ToStatus: string(domain.StatusInterested), Reason: "good go backend shenzhen"}); err != nil {
		t.Fatalf("record interested decision: %v", err)
	}
	if _, err := repo.RecordJobDecision(ctx, JobDecisionInput{JobID: trainingJob.ID, Action: "mark_ignore", ToStatus: string(domain.StatusIgnored), Reason: "training content"}); err != nil {
		t.Fatalf("record ignored decision: %v", err)
	}

	result, err := repo.RefreshMemoryReflections(ctx)
	if err != nil {
		t.Fatalf("refresh memory reflections: %v", err)
	}
	if result.Created != 1 {
		t.Fatalf("expected one reflection memory item, got %#v", result)
	}
	matches, err := repo.SearchSemanticMemory(ctx, SemanticMemoryQuery{Query: "user prefers go backend shenzhen and ignores training", Kind: SemanticMemoryKindPreferenceReflection, Limit: 3})
	if err != nil {
		t.Fatalf("search reflections: %v", err)
	}
	if len(matches) == 0 {
		t.Fatal("expected reflection match")
	}
	content := strings.ToLower(matches[0].Content)
	if !strings.Contains(content, "tencent") || !strings.Contains(content, "training co") || !strings.Contains(content, "go") {
		t.Fatalf("expected reflection to mention learned preference signals, got %q", matches[0].Content)
	}
	stats, err := repo.GetSemanticMemoryStats(ctx)
	if err != nil {
		t.Fatalf("get semantic stats: %v", err)
	}
	if stats.PreferenceReflectionItems != 1 {
		t.Fatalf("expected preference reflection stats, got %#v", stats)
	}
}

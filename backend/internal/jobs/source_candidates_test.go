package jobs

import (
	"context"
	"testing"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/db"
)

func TestRepositoryDiscoversAndAcceptsSourceCandidates(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo := NewRepository(conn)

	result, err := repo.DiscoverSourceCandidates(ctx, SourceDiscoveryInput{
		TargetCities:     []string{"Shenzhen"},
		TargetDirections: []string{"go", "ai_application"},
	})
	if err != nil {
		t.Fatalf("discover candidates: %v", err)
	}
	if result.Created == 0 {
		t.Fatalf("expected discovered candidates, got %#v", result)
	}

	second, err := repo.DiscoverSourceCandidates(ctx, SourceDiscoveryInput{
		TargetCities:     []string{"Shenzhen"},
		TargetDirections: []string{"go", "ai_application"},
	})
	if err != nil {
		t.Fatalf("discover candidates again: %v", err)
	}
	if second.Created != 0 || second.Duplicated == 0 {
		t.Fatalf("expected candidates to dedupe, got %#v", second)
	}

	candidates, err := repo.ListSourceCandidates(ctx, SourceCandidateFilter{Status: SourceCandidateStatusPending})
	if err != nil {
		t.Fatalf("list candidates: %v", err)
	}
	if len(candidates) == 0 {
		t.Fatal("expected pending candidates")
	}
	if candidates[0].Confidence <= 0 || candidates[0].Reason == "" || candidates[0].ValidationStatus == "" {
		t.Fatalf("expected candidate metadata, got %#v", candidates[0])
	}

	accepted, source, err := repo.AcceptSourceCandidate(ctx, candidates[0].ID)
	if err != nil {
		t.Fatalf("accept candidate: %v", err)
	}
	if accepted.Status != SourceCandidateStatusAccepted || source.ID == 0 {
		t.Fatalf("expected accepted candidate and source, got candidate=%#v source=%#v", accepted, source)
	}
}

func TestRepositoryRejectsSourceCandidate(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo := NewRepository(conn)
	result, err := repo.DiscoverSourceCandidates(ctx, SourceDiscoveryInput{TargetCities: []string{"Shenzhen"}})
	if err != nil {
		t.Fatalf("discover candidates: %v", err)
	}
	if result.Created == 0 {
		t.Fatal("expected candidates")
	}
	candidates, err := repo.ListSourceCandidates(ctx, SourceCandidateFilter{})
	if err != nil {
		t.Fatalf("list candidates: %v", err)
	}

	rejected, err := repo.UpdateSourceCandidateStatus(ctx, candidates[0].ID, SourceCandidateStatusRejected)
	if err != nil {
		t.Fatalf("reject candidate: %v", err)
	}
	if rejected.Status != SourceCandidateStatusRejected {
		t.Fatalf("expected rejected candidate, got %#v", rejected)
	}
}

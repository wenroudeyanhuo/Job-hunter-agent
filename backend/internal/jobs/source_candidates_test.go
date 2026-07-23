package jobs

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestRepositoryValidatesSourceCandidateWithRecruitmentSignals(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><head><title>Campus recruitment</title></head><body>
<a href="/jobs/go-backend-shenzhen">Go Backend Engineer Shenzhen Campus</a>
<a href="/positions/ai-application">AI application intern job description</a>
<p>Apply online for campus recruitment roles.</p>
</body></html>`))
	}))
	defer server.Close()

	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo := NewRepository(conn)
	if _, err := repo.createSourceCandidateIfMissing(ctx, sourceCandidateInput{
		Name:       "Example careers",
		URL:        server.URL,
		Category:   "official",
		ParserType: "generic",
		Reason:     "Test candidate",
		Confidence: 50,
	}); err != nil {
		t.Fatalf("create candidate: %v", err)
	}
	candidates, err := repo.ListSourceCandidates(ctx, SourceCandidateFilter{})
	if err != nil {
		t.Fatalf("list candidates: %v", err)
	}

	validated, err := repo.ValidateSourceCandidate(ctx, candidates[0].ID, server.Client())
	if err != nil {
		t.Fatalf("validate candidate: %v", err)
	}
	if validated.ValidationStatus != SourceCandidateValidationGood {
		t.Fatalf("expected verified candidate, got %#v", validated)
	}
	if validated.Confidence <= 50 {
		t.Fatalf("expected confidence to increase, got %d", validated.Confidence)
	}
	if validated.LastCheckedAt == nil || validated.ValidationReason == "" {
		t.Fatalf("expected validation metadata, got %#v", validated)
	}
}

func TestRepositoryValidatesSourceCandidateWithStructuredJobCards(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><body>
<a class="job-card" href="/roles/go-backend-2027">
  <h3>Go 后端开发工程师 深圳</h3>
  <p>负责云服务和微服务开发。</p>
</a>
</body></html>`))
	}))
	defer server.Close()

	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo := NewRepository(conn)
	if _, err := repo.createSourceCandidateIfMissing(ctx, sourceCandidateInput{
		Name:       "Structured careers",
		URL:        server.URL,
		Category:   "official",
		ParserType: "generic",
		Reason:     "Structured listing candidate",
		Confidence: 50,
	}); err != nil {
		t.Fatalf("create candidate: %v", err)
	}
	candidates, err := repo.ListSourceCandidates(ctx, SourceCandidateFilter{})
	if err != nil {
		t.Fatalf("list candidates: %v", err)
	}

	validated, err := repo.ValidateSourceCandidate(ctx, candidates[0].ID, server.Client())
	if err != nil {
		t.Fatalf("validate candidate: %v", err)
	}
	if validated.ValidationStatus != SourceCandidateValidationGood {
		t.Fatalf("expected structured job cards to verify the candidate, got %#v", validated)
	}
	if !strings.Contains(validated.ValidationReason, "job cards") {
		t.Fatalf("expected validation reason to mention job cards, got %q", validated.ValidationReason)
	}
}

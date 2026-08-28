package jobs

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/db"
)

func TestBuildActiveSourceSearchQueriesUsesProfile(t *testing.T) {
	queries := BuildActiveSourceSearchQueries(SourceDiscoveryInput{
		TargetCities:     []string{"Shenzhen"},
		TargetDirections: []string{"go", "ai_application"},
		SearchLimit:      3,
	})

	if len(queries) == 0 {
		t.Fatal("expected active source search queries")
	}
	joined := strings.Join(queries, "\n")
	if !strings.Contains(joined, "Shenzhen") || !strings.Contains(strings.ToLower(joined), "go") || !strings.Contains(strings.ToLower(joined), "ai") {
		t.Fatalf("expected queries to include profile city and directions, got %#v", queries)
	}
}

func TestRepositoryDiscoversActiveSourceCandidatesFromPublicSearch(t *testing.T) {
	ctx := context.Background()
	searchServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Query().Get("q"), "Shenzhen") {
			t.Fatalf("expected profile query, got %q", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`<html><body>
<a href="https://startup.example.com/careers">Startup Careers - Shenzhen Go campus hiring</a>
<a href="/url?q=https%3A%2F%2Fai.example.com%2Fjobs%2Fcampus&sa=U">AI Example 校招 Go 后端</a>
<a href="https://irrelevant.example.com/about">About us</a>
</body></html>`))
	}))
	defer searchServer.Close()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo := NewRepository(conn)

	result, err := repo.DiscoverSourceCandidates(ctx, SourceDiscoveryInput{
		TargetCities:     []string{"Shenzhen"},
		TargetDirections: []string{"go"},
		EnableWebSearch:  true,
		SearchLimit:      2,
		SearchEndpoint:   searchServer.URL + "/search",
	})
	if err != nil {
		t.Fatalf("discover source candidates: %v", err)
	}
	if result.Created == 0 {
		t.Fatalf("expected active search candidates, got %#v", result)
	}
	if result.WebSearchCandidates == 0 {
		t.Fatalf("expected web search candidate count, got %#v", result)
	}

	candidates, err := repo.ListSourceCandidates(ctx, SourceCandidateFilter{})
	if err != nil {
		t.Fatalf("list candidates: %v", err)
	}
	if !hasCandidateURL(candidates, "https://startup.example.com/careers") || !hasCandidateURL(candidates, "https://ai.example.com/jobs/campus") {
		t.Fatalf("expected search result candidates, got %#v", candidates)
	}
	for _, candidate := range candidates {
		if strings.Contains(candidate.URL, "startup.example.com") && candidate.DiscoveredBy != SourceCandidateDiscoveredByWebSearch {
			t.Fatalf("expected web search discovered_by, got %#v", candidate)
		}
	}
}

func hasCandidateURL(candidates []SourceCandidate, want string) bool {
	for _, candidate := range candidates {
		if candidate.URL == want {
			return true
		}
	}
	return false
}

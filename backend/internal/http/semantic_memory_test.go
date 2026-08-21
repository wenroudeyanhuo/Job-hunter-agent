package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/db"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/domain"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/jobs"
)

func TestSemanticMemoryAPIRebuildsAndSearches(t *testing.T) {
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo := jobs.NewRepository(conn)
	if _, err := repo.CreateJob(t.Context(), domain.Job{
		Company:       "ByteDance",
		Title:         "AI Application Backend Engineer",
		City:          "Shenzhen",
		DirectionTags: []string{"ai_application", "backend", "go"},
		Description:   "Build AI agent application platform and backend services.",
		MatchScore:    91,
		Status:        domain.StatusNew,
	}); err != nil {
		t.Fatalf("create job: %v", err)
	}
	router := NewRouter(&Handlers{Repo: repo})

	rebuildReq := httptest.NewRequest(http.MethodPost, "/api/agent/memory/rebuild", nil)
	rebuildRec := httptest.NewRecorder()
	router.ServeHTTP(rebuildRec, rebuildReq)
	if rebuildRec.Code != http.StatusOK {
		t.Fatalf("expected rebuild 200, got %d: %s", rebuildRec.Code, rebuildRec.Body.String())
	}

	searchReq := httptest.NewRequest(http.MethodGet, "/api/agent/memory/search?q=agent%20backend%20go", nil)
	searchRec := httptest.NewRecorder()
	router.ServeHTTP(searchRec, searchReq)
	if searchRec.Code != http.StatusOK {
		t.Fatalf("expected search 200, got %d: %s", searchRec.Code, searchRec.Body.String())
	}
	var matches []jobs.SemanticMemoryMatch
	if err := json.Unmarshal(searchRec.Body.Bytes(), &matches); err != nil {
		t.Fatalf("decode matches: %v", err)
	}
	if len(matches) != 1 || matches[0].Kind != jobs.SemanticMemoryKindJob {
		t.Fatalf("expected one job memory match, got %#v", matches)
	}
}

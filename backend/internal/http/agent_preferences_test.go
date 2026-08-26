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

func TestAgentPreferenceInsightsAPI(t *testing.T) {
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo := jobs.NewRepository(conn)
	job, err := repo.CreateJob(t.Context(), domain.Job{
		Company:       "Tencent",
		Title:         "Go Backend Engineer",
		City:          "Shenzhen",
		DirectionTags: []string{"backend", "go"},
		MatchScore:    90,
		Status:        domain.StatusNew,
	})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}
	if err := repo.UpdateStatus(t.Context(), job.ID, domain.StatusInterested); err != nil {
		t.Fatalf("mark interested: %v", err)
	}

	router := NewRouter(&Handlers{Repo: repo, LLM: &jobs.LLMConfig{}})
	req := httptest.NewRequest(http.MethodGet, "/api/agent/preferences", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var response jobs.AgentPreferenceInsights
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.TotalDecisions != 1 || len(response.InterestedCompanies) == 0 {
		t.Fatalf("expected preference insights response, got %#v", response)
	}
	var raw map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatalf("decode raw response: %v", err)
	}
	if _, ok := raw["recommended_jobs"]; !ok {
		t.Fatalf("expected recommended_jobs API field, got %s", rec.Body.String())
	}
}

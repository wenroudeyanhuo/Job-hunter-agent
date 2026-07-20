package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/domain"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/jobs"
)

func TestAgentBriefingAPI(t *testing.T) {
	repo, handler := testRouter(t, nil)
	if _, err := repo.CreateJob(t.Context(), domain.Job{
		Company:    "Tencent",
		Title:      "Go Backend Engineer",
		City:       "Shenzhen",
		MatchScore: 88,
		Status:     domain.StatusNew,
	}); err != nil {
		t.Fatalf("seed job: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/agent/briefing", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var response struct {
		Tone    string `json:"tone"`
		Metrics struct {
			TotalJobs int `json:"total_jobs"`
		} `json:"metrics"`
		NextActions []struct {
			Action string `json:"action"`
		} `json:"next_actions"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Metrics.TotalJobs != 1 {
		t.Fatalf("expected one job in briefing, got %#v", response)
	}
	if len(response.NextActions) == 0 {
		t.Fatalf("expected next actions, got %#v", response)
	}
}

func TestAgentDutyReportAPI(t *testing.T) {
	repo, handler := testRouter(t, nil)
	if _, err := repo.CreateJob(t.Context(), domain.Job{
		Company:    "Tencent",
		Title:      "Go Backend Engineer",
		City:       "Shenzhen",
		MatchScore: 88,
		Status:     domain.StatusNew,
	}); err != nil {
		t.Fatalf("seed job: %v", err)
	}
	if _, err := repo.CreateSource(t.Context(), jobs.SourceInput{
		Name:       "Meituan Campus",
		URL:        "https://campus.meituan.com/",
		Enabled:    true,
		ParserType: "meituan_api",
	}); err != nil {
		t.Fatalf("seed source: %v", err)
	}
	if err := repo.UpdateSourceHealthByURL(t.Context(), "https://campus.meituan.com/", jobs.SourceHealthInput{
		Status:  jobs.SourceHealthBroken,
		Reason:  "HTTP 502",
		Success: false,
	}); err != nil {
		t.Fatalf("mark source broken: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/agent/report", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var response jobs.AgentDutyReport
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Summary.StrongMatches != 1 || response.Summary.SourceIssues != 1 {
		t.Fatalf("unexpected report summary: %#v", response.Summary)
	}
	if response.NextBestAction.Action != "inspect_failed_sources" {
		t.Fatalf("expected source inspection action, got %#v", response.NextBestAction)
	}
}

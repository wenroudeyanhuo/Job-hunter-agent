package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/crawl"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/domain"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/jobs"
)

func TestAgentEventsAPI(t *testing.T) {
	repo, handler := testRouter(t, nil)
	if _, err := repo.CreateAgentEvent(t.Context(), jobs.AgentEventInput{
		Type:    "status_updated",
		Title:   "Marked interested",
		Summary: "You marked a backend role as interested.",
		Level:   "info",
	}); err != nil {
		t.Fatalf("seed event: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/agent/events", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var events []jobs.AgentEvent
	if err := json.Unmarshal(rec.Body.Bytes(), &events); err != nil {
		t.Fatalf("decode events: %v", err)
	}
	if len(events) != 1 || events[0].Type != "status_updated" {
		t.Fatalf("unexpected events: %#v", events)
	}
}

func TestRunCrawlRecordsAgentEvent(t *testing.T) {
	repo, handler := testRouter(t, fakeRunner{summary: crawl.RunSummary{JobsCreated: 2, JobsDuplicated: 1}})

	req := httptest.NewRequest(http.MethodPost, "/api/crawl/run", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	events, err := repo.ListAgentEvents(t.Context(), 10)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(events) != 1 || events[0].Type != "crawl_completed" {
		t.Fatalf("expected crawl event, got %#v", events)
	}
}

func TestUpdateJobStatusRecordsAgentEvent(t *testing.T) {
	repo, handler := testRouter(t, nil)
	job, err := repo.CreateJob(t.Context(), jobsTestJob())
	if err != nil {
		t.Fatalf("seed job: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/api/jobs/"+strconv.FormatInt(job.ID, 10)+"/status", bytes.NewBufferString(`{"status":"interested"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}

	events, err := repo.ListAgentEvents(t.Context(), 10)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(events) != 1 || events[0].Type != "job_status_updated" {
		t.Fatalf("expected status update event, got %#v", events)
	}
}

func jobsTestJob() domain.Job {
	return domain.Job{
		Company:    "Tencent",
		Title:      "Go Backend Engineer",
		City:       "Shenzhen",
		ApplyURL:   "https://example.com/apply",
		MatchScore: 88,
		Status:     domain.StatusNew,
	}
}

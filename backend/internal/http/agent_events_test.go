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
	if !containsAgentEvent(events, "crawl_completed") {
		t.Fatalf("expected crawl event, got %#v", events)
	}
}

func TestRunCrawlAutomaticallyRecordsAgentCycle(t *testing.T) {
	repo, handler := testRouter(t, fakeRunner{summary: crawl.RunSummary{JobsCreated: 2, JobsDuplicated: 1}})
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
		Name:       "Tencent Careers",
		URL:        "https://careers.tencent.com/",
		Enabled:    true,
		ParserType: "generic",
	}); err != nil {
		t.Fatalf("seed source: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/crawl/run", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	cycles, err := repo.ListAgentCycles(t.Context(), 10)
	if err != nil {
		t.Fatalf("list cycles: %v", err)
	}
	if len(cycles) != 1 || len(cycles[0].Trace) != 4 {
		t.Fatalf("expected automatic multi-agent cycle, got %#v", cycles)
	}
	if cycles[0].Summary == "" || cycles[0].OrchestratorProvider != jobs.MultiAgentOrchestratorEinoReady {
		t.Fatalf("expected cycle metadata, got %#v", cycles[0])
	}
	requests, err := repo.ListAgentActionRequests(t.Context(), jobs.AgentActionRequestStatusPending)
	if err != nil {
		t.Fatalf("list action requests: %v", err)
	}
	if len(requests) == 0 {
		t.Fatalf("expected cycle to create approval requests")
	}
	events, err := repo.ListAgentEvents(t.Context(), 10)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if !containsAgentEvent(events, "multi_agent_cycle_completed") {
		t.Fatalf("expected multi-agent cycle event, got %#v", events)
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

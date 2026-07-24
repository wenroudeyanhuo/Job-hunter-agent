package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/crawl"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/jobs"
)

func TestAgentActionRequestsAPIListsAndUpdatesRequests(t *testing.T) {
	repo, handler := testRouter(t, nil)
	if err := repo.RecordAgentActionRequests(t.Context(), "chat", []jobs.AgentCommandAction{
		{Type: "sync_application_plans", Target: "applications", Detail: "准备投递计划"},
	}); err != nil {
		t.Fatalf("seed action request: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/agent/actions?status=pending", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 list, got %d: %s", rec.Code, rec.Body.String())
	}
	var requests []jobs.AgentActionRequest
	if err := json.Unmarshal(rec.Body.Bytes(), &requests); err != nil {
		t.Fatalf("decode requests: %v", err)
	}
	if len(requests) != 1 || requests[0].ActionType != "sync_application_plans" {
		t.Fatalf("unexpected requests: %#v", requests)
	}

	req = httptest.NewRequest(http.MethodPatch, "/api/agent/actions/"+strconv.FormatInt(requests[0].ID, 10), strings.NewReader(`{"status":"dismissed"}`))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 update, got %d: %s", rec.Code, rec.Body.String())
	}
	var updated jobs.AgentActionRequest
	if err := json.Unmarshal(rec.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode updated request: %v", err)
	}
	if updated.Status != jobs.AgentActionRequestStatusDismissed || updated.ResolvedAt == nil {
		t.Fatalf("expected dismissed request, got %#v", updated)
	}
}

func TestAgentActionRequestApprovalExecutesRunCrawl(t *testing.T) {
	runner := &countingRunner{summary: crawl.RunSummary{JobsCreated: 2}}
	repo, handler := testRouter(t, runner)
	if err := repo.RecordAgentActionRequests(t.Context(), "chat", []jobs.AgentCommandAction{
		{Type: "run_crawl", Target: "sources", Detail: "Run a manual crawl."},
	}); err != nil {
		t.Fatalf("seed action request: %v", err)
	}
	requests, err := repo.ListAgentActionRequests(t.Context(), jobs.AgentActionRequestStatusPending)
	if err != nil {
		t.Fatalf("list requests: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/api/agent/actions/"+strconv.FormatInt(requests[0].ID, 10), strings.NewReader(`{"status":"approved"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 approve, got %d: %s", rec.Code, rec.Body.String())
	}
	if runner.calls != 1 || runner.lastTrigger != "agent_action" {
		t.Fatalf("expected approved action to run crawl once, got calls=%d trigger=%q", runner.calls, runner.lastTrigger)
	}
	events, err := repo.ListAgentEvents(t.Context(), 10)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if !containsAgentEvent(events, "agent_action_executed") {
		t.Fatalf("expected execution event, got %#v", events)
	}
	approved, err := repo.ListAgentActionRequests(t.Context(), jobs.AgentActionRequestStatusApproved)
	if err != nil {
		t.Fatalf("list approved requests: %v", err)
	}
	if len(approved) != 1 || approved[0].ExecutionStatus != jobs.AgentActionExecutionSucceeded || !strings.Contains(approved[0].ExecutionMessage, "Created 2 jobs") {
		t.Fatalf("expected execution receipt on approved action, got %#v", approved)
	}
}

func TestAgentActionRequestApprovalKeepsPendingWhenExecutionFails(t *testing.T) {
	runner := &countingRunner{err: errors.New("source is busy")}
	repo, handler := testRouter(t, runner)
	if err := repo.RecordAgentActionRequests(t.Context(), "chat", []jobs.AgentCommandAction{
		{Type: "run_crawl", Target: "sources", Detail: "Run a manual crawl."},
	}); err != nil {
		t.Fatalf("seed action request: %v", err)
	}
	requests, err := repo.ListAgentActionRequests(t.Context(), jobs.AgentActionRequestStatusPending)
	if err != nil {
		t.Fatalf("list requests: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/api/agent/actions/"+strconv.FormatInt(requests[0].ID, 10), strings.NewReader(`{"status":"approved"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 execution failure, got %d: %s", rec.Code, rec.Body.String())
	}
	after, err := repo.ListAgentActionRequests(t.Context(), jobs.AgentActionRequestStatusPending)
	if err != nil {
		t.Fatalf("list pending requests: %v", err)
	}
	if len(after) != 1 || after[0].Status != jobs.AgentActionRequestStatusPending || after[0].ResolvedAt != nil {
		t.Fatalf("expected failed action to remain pending, got %#v", after)
	}
	if after[0].ExecutionStatus != jobs.AgentActionExecutionFailed || !strings.Contains(after[0].ExecutionMessage, "source is busy") {
		t.Fatalf("expected failed execution receipt, got %#v", after[0])
	}
}

func TestAgentChatRecordsSuggestedActionRequests(t *testing.T) {
	repo, handler := testRouter(t, nil)
	body := bytes.NewBufferString(`{"message":"今天该做什么","active_view":"dashboard"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/agent/chat", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 chat, got %d: %s", rec.Code, rec.Body.String())
	}

	requests, err := repo.ListAgentActionRequests(t.Context(), jobs.AgentActionRequestStatusPending)
	if err != nil {
		t.Fatalf("list action requests: %v", err)
	}
	if len(requests) == 0 {
		t.Fatalf("expected chat suggested actions to be persisted")
	}
}

func TestAgentChatPersistsWorkPlanForSuggestedActions(t *testing.T) {
	repo, handler := testRouter(t, nil)
	body := bytes.NewBufferString(`{"message":"run crawl","active_view":"dashboard"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/agent/chat", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 chat, got %d: %s", rec.Code, rec.Body.String())
	}

	plans, err := repo.ListAgentPlans(t.Context(), jobs.AgentPlanStatusWaitingApproval, 10)
	if err != nil {
		t.Fatalf("list plans: %v", err)
	}
	if len(plans) != 1 {
		t.Fatalf("expected one pending work plan, got %#v", plans)
	}
	if plans[0].Goal != "run crawl" || len(plans[0].Steps) != 1 || plans[0].Steps[0].ActionType != "run_crawl" {
		t.Fatalf("unexpected work plan: %#v", plans[0])
	}

	req = httptest.NewRequest(http.MethodGet, "/api/agent/plans?status=waiting_approval", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 list plans, got %d: %s", rec.Code, rec.Body.String())
	}
	var response []jobs.AgentPlan
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode plans: %v", err)
	}
	if len(response) != 1 || response[0].Steps[0].ActionType != "run_crawl" {
		t.Fatalf("unexpected plan response: %#v", response)
	}
}

type countingRunner struct {
	summary     crawl.RunSummary
	err         error
	calls       int
	lastTrigger string
}

func (r *countingRunner) Run(_ context.Context, trigger string) (crawl.RunSummary, error) {
	r.calls++
	r.lastTrigger = trigger
	return r.summary, r.err
}

func containsAgentEvent(events []jobs.AgentEvent, eventType string) bool {
	for _, event := range events {
		if event.Type == eventType {
			return true
		}
	}
	return false
}

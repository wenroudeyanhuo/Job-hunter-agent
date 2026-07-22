package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

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

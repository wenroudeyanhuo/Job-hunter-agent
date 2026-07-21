package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/jobs"
)

func TestAgentChatAPIFallsBackToLocalReply(t *testing.T) {
	_, handler := testRouter(t, nil)
	body := bytes.NewBufferString(`{"message":"今天该做什么","active_view":"dashboard"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/agent/chat", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var response struct {
		Message jobs.AgentChatMessage `json:"message"`
		Reply   jobs.AgentChatReply   `json:"reply"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Message.Role != jobs.AgentChatRoleAssistant || response.Reply.Content == "" {
		t.Fatalf("unexpected chat response: %#v", response)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/agent/chat/messages", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var messages []jobs.AgentChatMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &messages); err != nil {
		t.Fatalf("decode messages: %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("expected user and assistant messages, got %#v", messages)
	}
}

func TestAgentChatStatusReportsLocalMode(t *testing.T) {
	_, handler := testRouter(t, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/agent/chat/status", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var status jobs.AgentChatStatus
	if err := json.Unmarshal(rec.Body.Bytes(), &status); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	if status.Mode != "local_rules" || status.Configured {
		t.Fatalf("expected local mode, got %#v", status)
	}
}

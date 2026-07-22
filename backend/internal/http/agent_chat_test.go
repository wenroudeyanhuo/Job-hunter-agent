package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/db"
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
	if !containsHTTPCommandAction(response.Reply.Actions, "sync_application_plans") {
		t.Fatalf("expected application sync action, got %#v", response.Reply.Actions)
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

func TestAgentChatAPIParsesModelSuggestedActions(t *testing.T) {
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"content\":\"我建议同步投递计划。\",\"actions\":[{\"type\":\"sync_application_plans\",\"target\":\"applications\",\"detail\":\"准备投递\"},{\"type\":\"auto_apply_resume\",\"target\":\"external\",\"detail\":\"危险动作\"}]}"}}]}`))
	}))
	defer model.Close()

	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo := jobs.NewRepository(conn)
	handler := NewRouter(&Handlers{
		Repo: repo,
		LLM:  jobs.LLMConfig{APIKey: "test-key", BaseURL: model.URL, Model: "test-model"},
	})

	body := bytes.NewBufferString(`{"message":"帮我准备投递","active_view":"applications"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/agent/chat", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var response struct {
		Reply jobs.AgentChatReply `json:"reply"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Reply.Source != "model" || !containsHTTPCommandAction(response.Reply.Actions, "sync_application_plans") {
		t.Fatalf("expected safe model action, got %#v", response.Reply)
	}
	if containsHTTPCommandAction(response.Reply.Actions, "auto_apply_resume") {
		t.Fatalf("unsafe model action should be filtered: %#v", response.Reply.Actions)
	}
}

func containsHTTPCommandAction(actions []jobs.AgentCommandAction, actionType string) bool {
	for _, action := range actions {
		if action.Type == actionType {
			return true
		}
	}
	return false
}

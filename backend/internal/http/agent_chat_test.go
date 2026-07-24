package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/db"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/domain"
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

func TestAgentChatModelPromptIncludesRecommendedJobs(t *testing.T) {
	var requestPayload struct {
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&requestPayload); err != nil {
			t.Fatalf("decode model request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"content\":\"我建议先看腾讯 Go 后端。\"}"}}]}`))
	}))
	defer model.Close()

	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo := jobs.NewRepository(conn)
	if _, err := repo.CreateJob(t.Context(), domain.Job{
		Company:    "Tencent",
		Title:      "Go Backend Engineer",
		City:       "Shenzhen",
		ApplyURL:   "https://careers.example.com/tencent/go-backend",
		MatchScore: 92,
	}); err != nil {
		t.Fatalf("seed job: %v", err)
	}
	handler := NewRouter(&Handlers{
		Repo: repo,
		LLM:  jobs.LLMConfig{APIKey: "test-key", BaseURL: model.URL, Model: "test-model"},
	})

	body := bytes.NewBufferString(`{"message":"今天最值得看什么岗位？","active_view":"opportunities"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/agent/chat", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if len(requestPayload.Messages) == 0 || !strings.Contains(requestPayload.Messages[0].Content, "Tencent - Go Backend Engineer - Shenzhen - score 92") {
		t.Fatalf("expected model prompt to include recommended job context, got %#v", requestPayload.Messages)
	}
}

func TestAgentChatModelReceivesRecentConversationHistory(t *testing.T) {
	var requestPayload struct {
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&requestPayload); err != nil {
			t.Fatalf("decode model request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"content\":\"我会接着上次的筛选继续分析。\"}"}}]}`))
	}))
	defer model.Close()

	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo := jobs.NewRepository(conn)
	if _, err := repo.RecordAgentChatMessage(t.Context(), jobs.AgentChatMessageInput{
		Role:    jobs.AgentChatRoleUser,
		Content: "\u6211\u60f3\u5148\u770b Go \u540e\u7aef\u5c97\u4f4d",
		Source:  "user",
	}); err != nil {
		t.Fatalf("seed user message: %v", err)
	}
	if _, err := repo.RecordAgentChatMessage(t.Context(), jobs.AgentChatMessageInput{
		Role:    jobs.AgentChatRoleAssistant,
		Content: "\u4e0a\u6b21\u6211\u5efa\u8bae\u4f60\u5148\u770b\u817e\u8baf\u548c\u7f8e\u56e2",
		Source:  "model",
	}); err != nil {
		t.Fatalf("seed assistant message: %v", err)
	}
	handler := NewRouter(&Handlers{
		Repo: repo,
		LLM:  jobs.LLMConfig{APIKey: "test-key", BaseURL: model.URL, Model: "test-model"},
	})

	body := bytes.NewBufferString(`{"message":"那算法岗位呢？","active_view":"opportunities"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/agent/chat", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !containsModelMessage(requestPayload.Messages, jobs.AgentChatRoleUser, "\u6211\u60f3\u5148\u770b Go \u540e\u7aef\u5c97\u4f4d") {
		t.Fatalf("expected previous user message in model request, got %#v", requestPayload.Messages)
	}
	if !containsModelMessage(requestPayload.Messages, jobs.AgentChatRoleAssistant, "\u4e0a\u6b21\u6211\u5efa\u8bae\u4f60\u5148\u770b\u817e\u8baf\u548c\u7f8e\u56e2") {
		t.Fatalf("expected previous assistant message in model request, got %#v", requestPayload.Messages)
	}
	if !containsModelMessage(requestPayload.Messages, jobs.AgentChatRoleUser, "\u90a3\u7b97\u6cd5\u5c97\u4f4d\u5462\uff1f") {
		t.Fatalf("expected current user message in model request, got %#v", requestPayload.Messages)
	}
}

func TestAgentChatHealthcheckCallsConfiguredModel(t *testing.T) {
	called := false
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("unexpected model path %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("unexpected authorization header %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer model.Close()

	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	handler := NewRouter(&Handlers{
		Repo: jobs.NewRepository(conn),
		LLM:  jobs.LLMConfig{Provider: "deepseek", APIKey: "test-key", BaseURL: model.URL, Model: "deepseek-chat"},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/agent/chat/healthcheck", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var response jobs.AgentChatHealthcheck
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode healthcheck: %v", err)
	}
	if !called || response.Status != "ok" || response.Provider != "deepseek" {
		t.Fatalf("expected successful model healthcheck, got called=%v response=%#v", called, response)
	}
}

func TestAgentChatHealthcheckSkipsWhenModelMissing(t *testing.T) {
	_, handler := testRouter(t, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/agent/chat/healthcheck", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var response jobs.AgentChatHealthcheck
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode healthcheck: %v", err)
	}
	if response.Status != "skipped" || response.Configured {
		t.Fatalf("expected skipped local healthcheck, got %#v", response)
	}
}

func containsModelMessage(messages []struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}, role string, content string) bool {
	for _, message := range messages {
		if message.Role == role && message.Content == content {
			return true
		}
	}
	return false
}

func containsHTTPCommandAction(actions []jobs.AgentCommandAction, actionType string) bool {
	for _, action := range actions {
		if action.Type == actionType {
			return true
		}
	}
	return false
}

package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/domain"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/jobs"
)

func TestSendFeishuTestUsesSavedWebhookURL(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	repo, handler := testRouter(t, nil)
	settings := jobs.DefaultSettings()
	settings.FeishuWebhookURL = server.URL
	if _, err := repo.SaveSettings(t.Context(), settings); err != nil {
		t.Fatalf("save settings: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/notifications/feishu/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !called {
		t.Fatal("expected saved webhook URL to be called")
	}
}

func TestSendFeishuReportUsesSavedWebhookURL(t *testing.T) {
	var payload struct {
		Content struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	repo, handler := testRouter(t, nil)
	settings := jobs.DefaultSettings()
	settings.FeishuWebhookURL = server.URL
	if _, err := repo.SaveSettings(t.Context(), settings); err != nil {
		t.Fatalf("save settings: %v", err)
	}
	if _, err := repo.CreateJob(t.Context(), domain.Job{
		Company:    "Tencent",
		Title:      "Go Backend Engineer",
		City:       "Shenzhen",
		MatchScore: 88,
		Status:     domain.StatusNew,
	}); err != nil {
		t.Fatalf("seed job: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/notifications/feishu/report", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(payload.Content.Text, "Strong matches: 1") {
		t.Fatalf("expected report summary in Feishu payload, got %q", payload.Content.Text)
	}
}

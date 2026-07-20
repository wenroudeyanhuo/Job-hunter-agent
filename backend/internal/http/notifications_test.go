package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

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

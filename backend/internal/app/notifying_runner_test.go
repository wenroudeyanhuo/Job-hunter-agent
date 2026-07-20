package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/crawl"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/db"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/domain"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/jobs"
)

func TestNotifyingRunnerSendsFeishuSummary(t *testing.T) {
	var payload struct {
		MsgType string `json:"msg_type"`
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

	runner := newNotifyingRunner(fakeSummaryRunner{summary: crawl.RunSummary{
		JobsCreated: 1,
		RecommendedJobs: []domain.Job{{
			Company:    "Tencent",
			Title:      "Backend Engineer",
			City:       "Shenzhen",
			MatchScore: 90,
		}},
	}}, nil, server.URL)

	if _, err := runner.Run(context.Background(), "manual"); err != nil {
		t.Fatalf("run: %v", err)
	}
	if payload.MsgType != "text" {
		t.Fatalf("expected text message, got %q", payload.MsgType)
	}
	if !strings.Contains(payload.Content.Text, "Tencent - Backend Engineer - Shenzhen - 90") {
		t.Fatalf("summary text missing job: %s", payload.Content.Text)
	}
}

func TestNotifyingRunnerSkipsEmptySummary(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer server.Close()

	runner := newNotifyingRunner(fakeSummaryRunner{summary: crawl.RunSummary{}}, nil, server.URL)
	if _, err := runner.Run(context.Background(), "manual"); err != nil {
		t.Fatalf("run: %v", err)
	}
	if called {
		t.Fatal("expected empty summary not to send notification")
	}
}

func TestNotifyingRunnerUsesSavedFeishuWebhook(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo := jobs.NewRepository(conn)
	settings := jobs.DefaultSettings()
	settings.FeishuWebhookURL = server.URL
	if _, err := repo.SaveSettings(context.Background(), settings); err != nil {
		t.Fatalf("save settings: %v", err)
	}

	runner := newNotifyingRunner(fakeSummaryRunner{summary: crawl.RunSummary{JobsCreated: 1}}, repo, "")
	if _, err := runner.Run(context.Background(), "manual"); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !called {
		t.Fatal("expected saved Feishu webhook to be called")
	}
}

type fakeSummaryRunner struct {
	summary crawl.RunSummary
	err     error
}

func (f fakeSummaryRunner) Run(context.Context, string) (crawl.RunSummary, error) {
	return f.summary, f.err
}

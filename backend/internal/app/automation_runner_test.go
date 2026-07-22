package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/db"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/domain"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/jobs"
)

func TestAutomationRunnerSendsDueDutyReportOnce(t *testing.T) {
	calls := 0
	var text string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var payload struct {
			Content struct {
				Text string `json:"text"`
			} `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		text = payload.Content.Text
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
	settings.AutoDutyReportEnabled = true
	settings.DutyReportTime = "18:00"
	if _, err := repo.SaveSettings(context.Background(), settings); err != nil {
		t.Fatalf("save settings: %v", err)
	}
	if _, err := repo.CreateJob(context.Background(), domain.Job{
		Company:    "Tencent",
		Title:      "Go Backend Engineer",
		City:       "Shenzhen",
		MatchScore: 88,
		Status:     domain.StatusNew,
	}); err != nil {
		t.Fatalf("seed job: %v", err)
	}

	runner := newAutomationRunner(repo, "")
	now := time.Date(2026, 7, 21, 18, 1, 0, 0, time.UTC)
	sent, err := runner.Tick(context.Background(), now)
	if err != nil {
		t.Fatalf("tick: %v", err)
	}
	if !sent || calls != 1 {
		t.Fatalf("expected one sent report, sent=%v calls=%d", sent, calls)
	}
	if !strings.Contains(text, "Job Hunter Agent duty report") {
		t.Fatalf("expected duty report text, got %q", text)
	}

	sent, err = runner.Tick(context.Background(), now.Add(10*time.Minute))
	if err != nil {
		t.Fatalf("second tick: %v", err)
	}
	if sent || calls != 1 {
		t.Fatalf("expected second same-day tick to skip, sent=%v calls=%d", sent, calls)
	}
}

func TestAutomationRunnerDiscoversSourcesWhenDue(t *testing.T) {
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo := jobs.NewRepository(conn)
	settings := jobs.DefaultSettings()
	settings.AutoSourceDiscoveryEnabled = true
	settings.SourceDiscoveryIntervalHours = 1
	if _, err := repo.SaveSettings(context.Background(), settings); err != nil {
		t.Fatalf("save settings: %v", err)
	}

	runner := newAutomationRunner(repo, "")
	ran, err := runner.Tick(context.Background(), time.Date(2026, 7, 21, 9, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("tick: %v", err)
	}
	if !ran {
		t.Fatal("expected automation tick to run source discovery")
	}
	candidates, err := repo.ListSourceCandidates(context.Background(), jobs.SourceCandidateFilter{})
	if err != nil {
		t.Fatalf("list source candidates: %v", err)
	}
	if len(candidates) == 0 {
		t.Fatal("expected discovered source candidates")
	}
	updated, err := repo.GetSettings(context.Background())
	if err != nil {
		t.Fatalf("get settings: %v", err)
	}
	if updated.LastSourceDiscoveryAt == nil {
		t.Fatalf("expected last source discovery time to be persisted")
	}
}

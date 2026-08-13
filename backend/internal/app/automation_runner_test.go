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
	settings.AutoSourceDiscoveryEnabled = false
	settings.DutyReportTime = "18:00"
	settings.TimeZone = "UTC"
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

func TestAutomationRunnerCreatesDailyWorkPlanOnce(t *testing.T) {
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo := jobs.NewRepository(conn)
	settings := jobs.DefaultSettings()
	settings.AutoDutyReportEnabled = true
	settings.AutoSourceDiscoveryEnabled = false
	settings.DutyReportTime = "09:00"
	settings.TimeZone = "UTC"
	if _, err := repo.SaveSettings(context.Background(), settings); err != nil {
		t.Fatalf("save settings: %v", err)
	}
	if _, err := repo.CreateSource(context.Background(), jobs.SourceInput{
		Name:       "Tencent Careers",
		URL:        "https://careers.tencent.com/",
		Enabled:    true,
		ParserType: "tencent_api",
	}); err != nil {
		t.Fatalf("seed source: %v", err)
	}

	runner := newAutomationRunner(repo, "")
	now := time.Date(2026, 8, 13, 9, 1, 0, 0, time.UTC)
	ran, err := runner.Tick(context.Background(), now)
	if err != nil {
		t.Fatalf("tick: %v", err)
	}
	if !ran {
		t.Fatal("expected automation tick to create a daily plan")
	}
	plans, err := repo.ListAgentPlans(context.Background(), jobs.AgentPlanStatusWaitingApproval, 10)
	if err != nil {
		t.Fatalf("list plans: %v", err)
	}
	if len(plans) != 1 || plans[0].Source != "automation" || len(plans[0].Steps) == 0 {
		t.Fatalf("expected one automation work plan, got %#v", plans)
	}
	requests, err := repo.ListAgentActionRequests(context.Background(), jobs.AgentActionRequestStatusPending)
	if err != nil {
		t.Fatalf("list action requests: %v", err)
	}
	if len(requests) != len(plans[0].Steps) || requests[0].PlanID != plans[0].ID {
		t.Fatalf("expected linked approval requests, got requests=%#v plans=%#v", requests, plans)
	}

	ran, err = runner.Tick(context.Background(), now.Add(30*time.Minute))
	if err != nil {
		t.Fatalf("second tick: %v", err)
	}
	if ran {
		t.Fatal("expected same-day automation plan to be skipped")
	}
	plans, err = repo.ListAgentPlans(context.Background(), "", 10)
	if err != nil {
		t.Fatalf("list all plans: %v", err)
	}
	if len(plans) != 1 {
		t.Fatalf("expected no duplicate daily plans, got %#v", plans)
	}
}

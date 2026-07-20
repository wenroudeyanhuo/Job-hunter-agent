package jobs

import (
	"testing"
	"time"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/domain"
)

func TestBuildAgentDutyReportPrioritizesWork(t *testing.T) {
	now := time.Date(2026, 7, 20, 9, 30, 0, 0, time.UTC)
	report := BuildAgentDutyReport([]domain.Job{
		{Company: "Tencent", Title: "Go Backend Engineer", City: "Shenzhen", MatchScore: 86, Status: domain.StatusNew, DiscoveredAt: now},
		{Company: "ByteDance", Title: "AI Application Engineer", City: "Shenzhen", MatchScore: 78, Status: domain.StatusNew, DiscoveredAt: now},
		{Company: "DJI", Title: "Campus portal", City: "Unknown", MatchScore: 35, Status: domain.StatusManualCheck, DiscoveredAt: now},
	}, []Source{
		{Name: "Meituan Campus", Enabled: true, HealthStatus: SourceHealthBroken, HealthReason: "HTTP 502", ConsecutiveFailures: 2},
		{Name: "OPPO Careers", Enabled: true, HealthStatus: SourceHealthWarning, HealthReason: "No jobs found in latest run"},
	}, []domain.JobRun{
		{Status: "partial_success", JobsCreated: 2, JobsDuplicated: 1, ManualCheckCount: 1, SourcesFailed: 1, StartedAt: now},
	})

	if report.Tone != "needs_attention" {
		t.Fatalf("expected needs_attention tone, got %q", report.Tone)
	}
	if report.Summary.JobsToReview != 3 || report.Summary.StrongMatches != 2 || report.Summary.SourceIssues != 2 {
		t.Fatalf("unexpected summary: %#v", report.Summary)
	}
	if len(report.TodaysWork) == 0 || report.TodaysWork[0].Kind != "inspect_failed_sources" {
		t.Fatalf("expected source inspection work first, got %#v", report.TodaysWork)
	}
	if len(report.NeedsDecision) != 1 || report.NeedsDecision[0].JobTitle != "Campus portal" {
		t.Fatalf("expected manual-check job decision, got %#v", report.NeedsDecision)
	}
	if len(report.SourceIssues) != 2 || report.SourceIssues[0].Status != SourceHealthBroken {
		t.Fatalf("expected broken source issue first, got %#v", report.SourceIssues)
	}
	if report.NextBestAction.Action != "inspect_failed_sources" {
		t.Fatalf("expected failed source action, got %#v", report.NextBestAction)
	}
}

func TestBuildAgentDutyReportAsksForCrawlWhenNoRunExists(t *testing.T) {
	report := BuildAgentDutyReport(nil, []Source{
		{Name: "Tencent Careers", Enabled: true, HealthStatus: SourceHealthUnknown},
	}, nil)

	if report.Tone != "needs_work" {
		t.Fatalf("expected needs_work tone, got %q", report.Tone)
	}
	if report.NextBestAction.Action != "run_crawl" {
		t.Fatalf("expected run crawl action, got %#v", report.NextBestAction)
	}
}

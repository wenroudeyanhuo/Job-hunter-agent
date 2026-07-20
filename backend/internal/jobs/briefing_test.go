package jobs

import (
	"testing"
	"time"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/domain"
)

func TestBuildAgentBriefingRecommendsCrawlWhenNoSources(t *testing.T) {
	briefing := BuildAgentBriefing(nil, nil, nil)

	if briefing.Tone != "needs_setup" {
		t.Fatalf("expected needs_setup tone, got %q", briefing.Tone)
	}
	if len(briefing.NextActions) == 0 || briefing.NextActions[0].Action != "add_recommended_and_crawl" {
		t.Fatalf("expected recommended crawl action, got %#v", briefing.NextActions)
	}
}

func TestBuildAgentBriefingSummarizesWorkQueue(t *testing.T) {
	now := time.Date(2026, 7, 20, 9, 0, 0, 0, time.UTC)
	briefing := BuildAgentBriefing([]domain.Job{
		{Title: "Go Backend Engineer", MatchScore: 82, Status: domain.StatusNew},
		{Title: "Algorithm Engineer", MatchScore: 91, Status: domain.StatusManualCheck},
		{Title: "Frontend Engineer", MatchScore: 55, Status: domain.StatusIgnored},
	}, []Source{
		{Name: "Tencent Careers", Enabled: true},
		{Name: "Disabled Source", Enabled: false},
	}, []domain.JobRun{
		{
			StartedAt:        now,
			Status:           "success",
			JobsCreated:      2,
			JobsDuplicated:   1,
			ManualCheckCount: 1,
			SourcesFailed:    0,
		},
	})

	if briefing.Metrics.TotalJobs != 3 {
		t.Fatalf("expected total jobs, got %#v", briefing.Metrics)
	}
	if briefing.Metrics.StrongMatches != 2 {
		t.Fatalf("expected two strong matches, got %#v", briefing.Metrics)
	}
	if briefing.Metrics.ManualCheckJobs != 1 {
		t.Fatalf("expected one manual check job, got %#v", briefing.Metrics)
	}
	if briefing.Metrics.EnabledSources != 1 {
		t.Fatalf("expected one enabled source, got %#v", briefing.Metrics)
	}
	if briefing.LatestRun == nil || briefing.LatestRun.JobsCreated != 2 {
		t.Fatalf("expected latest run summary, got %#v", briefing.LatestRun)
	}
	if len(briefing.NextActions) == 0 || briefing.NextActions[0].Action != "review_manual_check" {
		t.Fatalf("expected manual check action, got %#v", briefing.NextActions)
	}
}

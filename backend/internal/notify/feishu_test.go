package notify

import (
	"strings"
	"testing"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/crawl"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/domain"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/jobs"
)

func TestBuildFeishuSummary(t *testing.T) {
	text := BuildFeishuSummary(crawl.RunSummary{
		JobsCreated:      3,
		ManualCheckCount: 1,
		SourcesFailed:    2,
	}, []domain.Job{{
		Company:          "Tencent",
		Title:            "Backend Engineer",
		City:             "Shenzhen",
		MatchScore:       92,
		RecommendReasons: []string{"Shenzhen role", "Clear application URL"},
		ApplyURL:         "https://example.com/apply",
	}})

	wants := []string{
		"Jobs created: 3",
		"Strong matches: 1",
		"Manual check: 1",
		"Failed sources: 2",
		"Tencent - Backend Engineer - Shenzhen - 92",
		"Shenzhen role",
		"https://example.com/apply",
	}
	for _, want := range wants {
		if !strings.Contains(text, want) {
			t.Fatalf("expected summary to contain %q, got:\n%s", want, text)
		}
	}
}

func TestBuildFeishuDutyReportIncludesTrendSummary(t *testing.T) {
	text := BuildFeishuDutyReport(jobs.AgentDutyReport{
		Headline:     "I found work that needs your decision today.",
		TrendSummary: "Compared with the previous snapshot: strong matches +2, source issues -1, open tasks 0.",
		NextBestAction: jobs.AgentReportAction{
			Label:  "Review strong matches",
			Reason: "These are the most promising roles.",
		},
	})

	if !strings.Contains(text, "Trend:") || !strings.Contains(text, "strong matches +2") {
		t.Fatalf("expected trend summary in duty report, got:\n%s", text)
	}
}

func TestBuildFeishuDutyReportWithCycleIncludesAgentWork(t *testing.T) {
	cycle := jobs.AgentCycleRecord{
		Summary:              "Ran 4 recruiting agents and proposed 2 approval-gated actions.",
		ReadinessScore:       76,
		OrchestratorProvider: jobs.MultiAgentOrchestratorModelEnhanced,
		Trace: []jobs.MultiAgentTrace{
			{AgentKey: jobs.MultiAgentSourceScout, Decision: "Repair one source."},
			{AgentKey: jobs.MultiAgentJobAnalyst, Decision: "Review strong matches."},
		},
		Actions: []jobs.AgentCommandAction{{Type: "review_strong_matches", Target: "opportunities"}},
	}
	text := BuildFeishuDutyReportWithCycle(jobs.AgentDutyReport{
		Headline: "I found work that needs your decision today.",
		NextBestAction: jobs.AgentReportAction{
			Label:  "Review strong matches",
			Reason: "These are promising.",
		},
	}, &cycle)

	wants := []string{"Latest agent cycle", "76", "model_enhanced", "Source Scout", "review_strong_matches"}
	for _, want := range wants {
		if !strings.Contains(text, want) {
			t.Fatalf("expected duty report with cycle to contain %q, got:\n%s", want, text)
		}
	}
}

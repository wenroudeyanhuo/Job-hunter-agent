package jobs

import (
	"time"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/domain"
)

type AgentBriefing struct {
	GeneratedAt time.Time         `json:"generated_at"`
	Tone        string            `json:"tone"`
	Headline    string            `json:"headline"`
	Metrics     AgentMetrics      `json:"metrics"`
	LatestRun   *domain.JobRun    `json:"latest_run,omitempty"`
	Highlights  []string          `json:"highlights"`
	NextActions []AgentNextAction `json:"next_actions"`
}

type AgentMetrics struct {
	TotalJobs       int `json:"total_jobs"`
	StrongMatches   int `json:"strong_matches"`
	ManualCheckJobs int `json:"manual_check_jobs"`
	InterestedJobs  int `json:"interested_jobs"`
	AppliedJobs     int `json:"applied_jobs"`
	EnabledSources  int `json:"enabled_sources"`
}

type AgentNextAction struct {
	Action   string `json:"action"`
	Label    string `json:"label"`
	Reason   string `json:"reason"`
	Priority int    `json:"priority"`
}

func BuildAgentBriefing(jobList []domain.Job, sources []Source, runs []domain.JobRun) AgentBriefing {
	briefing := AgentBriefing{
		GeneratedAt: time.Now().UTC(),
		Tone:        "steady",
		Headline:    "I am monitoring your autumn recruitment pipeline.",
		Highlights:  []string{},
		NextActions: []AgentNextAction{},
	}

	for _, source := range sources {
		if source.Enabled {
			briefing.Metrics.EnabledSources++
		}
	}
	for _, job := range jobList {
		briefing.Metrics.TotalJobs++
		if job.MatchScore >= 70 {
			briefing.Metrics.StrongMatches++
		}
		switch job.Status {
		case domain.StatusManualCheck:
			briefing.Metrics.ManualCheckJobs++
		case domain.StatusInterested:
			briefing.Metrics.InterestedJobs++
		case domain.StatusApplied:
			briefing.Metrics.AppliedJobs++
		}
	}
	if len(runs) > 0 {
		latest := runs[0]
		briefing.LatestRun = &latest
	}

	if briefing.Metrics.EnabledSources == 0 {
		briefing.Tone = "needs_setup"
		briefing.Headline = "I do not have active sources yet."
		briefing.NextActions = append(briefing.NextActions, AgentNextAction{
			Action:   "add_recommended_and_crawl",
			Label:    "Add recommended sources and crawl",
			Reason:   "I need official recruitment sources before I can continuously find jobs.",
			Priority: 100,
		})
		return briefing
	}

	if briefing.Metrics.ManualCheckJobs > 0 {
		briefing.Tone = "needs_review"
		briefing.NextActions = append(briefing.NextActions, AgentNextAction{
			Action:   "review_manual_check",
			Label:    "Review manual-check jobs",
			Reason:   "Some pages look relevant but need your decision before I treat them as strong opportunities.",
			Priority: 90,
		})
	}
	if briefing.Metrics.StrongMatches > briefing.Metrics.InterestedJobs+briefing.Metrics.AppliedJobs {
		briefing.NextActions = append(briefing.NextActions, AgentNextAction{
			Action:   "review_strong_matches",
			Label:    "Review strong matches",
			Reason:   "I found high-score roles that have not been marked interested or applied.",
			Priority: 80,
		})
	}
	if briefing.LatestRun == nil {
		briefing.NextActions = append(briefing.NextActions, AgentNextAction{
			Action:   "run_crawl",
			Label:    "Run a crawl",
			Reason:   "Sources are ready, but I have not recorded a crawl yet.",
			Priority: 70,
		})
	} else if briefing.LatestRun.SourcesFailed > 0 {
		briefing.Tone = "needs_attention"
		briefing.NextActions = append(briefing.NextActions, AgentNextAction{
			Action:   "inspect_failed_sources",
			Label:    "Inspect failed sources",
			Reason:   "The last crawl had source failures, so some opportunities may be missing.",
			Priority: 85,
		})
	}

	if briefing.Metrics.TotalJobs == 0 {
		briefing.Headline = "Sources are ready, but I have not found stored jobs yet."
	} else {
		briefing.Highlights = append(briefing.Highlights, strongMatchHighlight(briefing.Metrics.StrongMatches))
		if briefing.LatestRun != nil {
			briefing.Highlights = append(briefing.Highlights, latestRunHighlight(*briefing.LatestRun))
		}
	}
	if len(briefing.NextActions) == 0 {
		briefing.NextActions = append(briefing.NextActions, AgentNextAction{
			Action:   "keep_monitoring",
			Label:    "Keep monitoring",
			Reason:   "The current queue looks under control.",
			Priority: 10,
		})
	}
	return briefing
}

func strongMatchHighlight(count int) string {
	if count == 0 {
		return "No strong matches are waiting right now."
	}
	return "Strong matches waiting: " + itoa(count) + "."
}

func latestRunHighlight(run domain.JobRun) string {
	return "Latest crawl created " + itoa(run.JobsCreated) + " jobs and found " + itoa(run.JobsDuplicated) + " duplicates."
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	digits := []byte{}
	for value > 0 {
		digits = append([]byte{byte('0' + value%10)}, digits...)
		value /= 10
	}
	return string(digits)
}

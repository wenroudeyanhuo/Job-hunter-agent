package jobs

import (
	"sort"
	"time"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/domain"
)

type AgentDutyReport struct {
	GeneratedAt    time.Time           `json:"generated_at"`
	Tone           string              `json:"tone"`
	Headline       string              `json:"headline"`
	Summary        AgentDutySummary    `json:"summary"`
	TodaysWork     []AgentWorkItem     `json:"todays_work"`
	NeedsDecision  []AgentDecisionItem `json:"needs_decision"`
	SourceIssues   []AgentSourceIssue  `json:"source_issues"`
	NextBestAction AgentReportAction   `json:"next_best_action"`
	LatestRun      *domain.JobRun      `json:"latest_run,omitempty"`
}

type AgentDutySummary struct {
	JobsToReview  int `json:"jobs_to_review"`
	StrongMatches int `json:"strong_matches"`
	ManualCheck   int `json:"manual_check"`
	SourceIssues  int `json:"source_issues"`
	NewJobs       int `json:"new_jobs"`
}

type AgentWorkItem struct {
	Kind     string `json:"kind"`
	Title    string `json:"title"`
	Detail   string `json:"detail"`
	Priority int    `json:"priority"`
	Count    int    `json:"count"`
}

type AgentDecisionItem struct {
	JobID    int64  `json:"job_id"`
	Company  string `json:"company"`
	JobTitle string `json:"job_title"`
	City     string `json:"city"`
	Reason   string `json:"reason"`
	Score    int    `json:"score"`
}

type AgentSourceIssue struct {
	SourceID            int64  `json:"source_id"`
	Name                string `json:"name"`
	URL                 string `json:"url"`
	Status              string `json:"status"`
	Reason              string `json:"reason"`
	ConsecutiveFailures int    `json:"consecutive_failures"`
	LastFoundCount      int    `json:"last_found_count"`
}

type AgentReportAction struct {
	Action string `json:"action"`
	Label  string `json:"label"`
	Reason string `json:"reason"`
}

func BuildAgentDutyReport(jobList []domain.Job, sources []Source, runs []domain.JobRun) AgentDutyReport {
	report := AgentDutyReport{
		GeneratedAt:   time.Now().UTC(),
		Tone:          "steady",
		Headline:      "I am on duty and watching your recruitment pipeline.",
		TodaysWork:    []AgentWorkItem{},
		NeedsDecision: []AgentDecisionItem{},
		SourceIssues:  []AgentSourceIssue{},
	}
	if len(runs) > 0 {
		latest := runs[0]
		report.LatestRun = &latest
	}

	for _, job := range jobList {
		if job.Status == domain.StatusNew || job.Status == domain.StatusManualCheck {
			report.Summary.JobsToReview++
		}
		if job.Status == domain.StatusNew {
			report.Summary.NewJobs++
		}
		if job.MatchScore >= 70 && job.Status != domain.StatusApplied && job.Status != domain.StatusIgnored {
			report.Summary.StrongMatches++
		}
		if job.Status == domain.StatusManualCheck {
			report.Summary.ManualCheck++
			report.NeedsDecision = append(report.NeedsDecision, AgentDecisionItem{
				JobID:    job.ID,
				Company:  fallbackText(job.Company, "Unknown company"),
				JobTitle: fallbackText(job.Title, "Untitled role"),
				City:     fallbackText(job.City, "Unknown"),
				Reason:   firstText(job.PenaltyReasons, "I need your decision before treating this as a concrete opportunity."),
				Score:    job.MatchScore,
			})
		}
	}

	for _, source := range sources {
		if !source.Enabled {
			continue
		}
		if source.HealthStatus != SourceHealthBroken && source.HealthStatus != SourceHealthWarning {
			continue
		}
		report.SourceIssues = append(report.SourceIssues, AgentSourceIssue{
			SourceID:            source.ID,
			Name:                fallbackText(source.Name, "Unnamed source"),
			URL:                 source.URL,
			Status:              source.HealthStatus,
			Reason:              fallbackText(source.HealthReason, "Source needs attention"),
			ConsecutiveFailures: source.ConsecutiveFailures,
			LastFoundCount:      source.LastFoundCount,
		})
	}
	sort.Slice(report.SourceIssues, func(i, j int) bool {
		return sourceIssuePriority(report.SourceIssues[i]) > sourceIssuePriority(report.SourceIssues[j])
	})
	report.Summary.SourceIssues = len(report.SourceIssues)

	if report.Summary.StrongMatches > 0 {
		report.TodaysWork = append(report.TodaysWork, AgentWorkItem{
			Kind:     "review_strong_matches",
			Title:    "Review strong matches",
			Detail:   "High-score roles are waiting before the autumn recruitment window moves on.",
			Priority: 90,
			Count:    report.Summary.StrongMatches,
		})
	}
	if report.Summary.ManualCheck > 0 {
		report.TodaysWork = append(report.TodaysWork, AgentWorkItem{
			Kind:     "review_manual_check",
			Title:    "Decide manual-check jobs",
			Detail:   "Some collected pages need your judgement before I can classify them confidently.",
			Priority: 80,
			Count:    report.Summary.ManualCheck,
		})
	}
	if report.Summary.SourceIssues > 0 {
		report.TodaysWork = append(report.TodaysWork, AgentWorkItem{
			Kind:     "inspect_failed_sources",
			Title:    "Inspect unhealthy sources",
			Detail:   "A broken or warning source can hide fresh openings from the queue.",
			Priority: 95,
			Count:    report.Summary.SourceIssues,
		})
	}
	if report.LatestRun == nil {
		report.TodaysWork = append(report.TodaysWork, AgentWorkItem{
			Kind:     "run_crawl",
			Title:    "Run the first crawl",
			Detail:   "Sources exist, but I have no crawl record to work from yet.",
			Priority: 70,
			Count:    1,
		})
	}
	sort.Slice(report.TodaysWork, func(i, j int) bool {
		return report.TodaysWork[i].Priority > report.TodaysWork[j].Priority
	})

	report.NextBestAction = nextReportAction(report)
	if report.Summary.SourceIssues > 0 {
		report.Tone = "needs_attention"
		report.Headline = "Some sources need attention before I can watch reliably."
	} else if report.Summary.JobsToReview > 0 || report.LatestRun == nil {
		report.Tone = "needs_work"
		report.Headline = "I found work that needs your decision today."
	}
	return report
}

func nextReportAction(report AgentDutyReport) AgentReportAction {
	if len(report.SourceIssues) > 0 {
		return AgentReportAction{
			Action: "inspect_failed_sources",
			Label:  "Inspect source issues",
			Reason: "Fixing source visibility first protects future job discovery.",
		}
	}
	if report.Summary.StrongMatches > 0 {
		return AgentReportAction{
			Action: "review_strong_matches",
			Label:  "Review strong matches",
			Reason: "These are the most promising roles in the current queue.",
		}
	}
	if report.Summary.ManualCheck > 0 {
		return AgentReportAction{
			Action: "review_manual_check",
			Label:  "Review manual-check jobs",
			Reason: "Your decision lets me clean up uncertain collected pages.",
		}
	}
	if report.LatestRun == nil {
		return AgentReportAction{
			Action: "run_crawl",
			Label:  "Run crawl",
			Reason: "I need a crawl result before I can prepare a useful queue.",
		}
	}
	return AgentReportAction{
		Action: "keep_monitoring",
		Label:  "Keep monitoring",
		Reason: "The current queue is under control.",
	}
}

func sourceIssuePriority(issue AgentSourceIssue) int {
	if issue.Status == SourceHealthBroken {
		return 100 + issue.ConsecutiveFailures
	}
	return 50
}

func firstText(values []string, fallback string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return fallback
}

func fallbackText(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

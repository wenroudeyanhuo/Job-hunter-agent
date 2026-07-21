package jobs

import (
	"sort"
	"time"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/domain"
)

type AgentReview struct {
	GeneratedAt time.Time             `json:"generated_at"`
	Health      AgentReviewHealth     `json:"health"`
	Focus       AgentReviewFocus      `json:"focus"`
	Findings    []AgentReviewFinding  `json:"findings"`
	Decisions   []AgentReviewDecision `json:"decisions"`
	NextSteps   []AgentReviewStep     `json:"next_steps"`
}

type AgentReviewHealth struct {
	Score  int    `json:"score"`
	Label  string `json:"label"`
	Reason string `json:"reason"`
}

type AgentReviewFocus struct {
	Title  string `json:"title"`
	Detail string `json:"detail"`
	Action string `json:"action"`
}

type AgentReviewFinding struct {
	Kind   string `json:"kind"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
	Level  string `json:"level"`
	Metric int    `json:"metric"`
}

type AgentReviewDecision struct {
	Question string `json:"question"`
	Context  string `json:"context"`
	Action   string `json:"action"`
}

type AgentReviewStep struct {
	Label  string `json:"label"`
	Reason string `json:"reason"`
	Action string `json:"action"`
}

func BuildAgentReview(jobList []domain.Job, sources []Source, runs []domain.JobRun, tasks []AgentTask) AgentReview {
	review := AgentReview{
		GeneratedAt: time.Now().UTC(),
		Health: AgentReviewHealth{
			Score:  100,
			Label:  "Healthy",
			Reason: "The recruiting pipeline is under control.",
		},
		Focus: AgentReviewFocus{
			Title:  "Keep monitoring",
			Detail: "No urgent action is blocking the current workflow.",
			Action: "keep_monitoring",
		},
		Findings:  []AgentReviewFinding{},
		Decisions: []AgentReviewDecision{},
		NextSteps: []AgentReviewStep{},
	}

	var strongMatches, manualJobs, newJobs, appliedJobs, interestedJobs int
	for _, job := range jobList {
		if job.Status == domain.StatusNew {
			newJobs++
		}
		if job.Status == domain.StatusInterested {
			interestedJobs++
		}
		if job.Status == domain.StatusApplied {
			appliedJobs++
		}
		if job.MatchScore >= 70 && job.Status != domain.StatusApplied && job.Status != domain.StatusIgnored {
			strongMatches++
		}
		if job.Status == domain.StatusManualCheck {
			manualJobs++
		}
	}

	var enabledSources, sourceIssues int
	for _, source := range sources {
		if !source.Enabled {
			continue
		}
		enabledSources++
		if source.HealthStatus == SourceHealthWarning || source.HealthStatus == SourceHealthBroken {
			sourceIssues++
		}
	}

	var openTasks, staleTasks, escalatedTasks int
	for _, task := range tasks {
		switch task.Status {
		case AgentTaskStatusDone:
			continue
		case AgentTaskStatusStale:
			openTasks++
			staleTasks++
		case AgentTaskStatusEscalated:
			openTasks++
			escalatedTasks++
		default:
			openTasks++
		}
	}

	if enabledSources == 0 {
		review.addFinding("setup", "No active source pool", "I cannot collect new openings until recommended or custom sources are enabled.", "critical", 0)
		review.addDecision("Should I add the recommended company source pool now?", "This is the fastest way to start collecting real openings for other users too.", "add_recommended_and_crawl")
	}
	if len(runs) == 0 {
		review.addFinding("crawl", "No crawl has run yet", "Sources may be configured, but I do not have a crawl result to evaluate.", "warning", 0)
	}
	if sourceIssues > 0 {
		review.addFinding("source_health", "Unhealthy sources detected", "Some enabled sources are warning or broken, so fresh roles may be missing.", "critical", sourceIssues)
		review.addDecision("Should I inspect failed sources before you review jobs?", "Source health affects every future recommendation.", "inspect_failed_sources")
	}
	if strongMatches > 0 {
		review.addFinding("recommendation", "Strong matches are waiting", "High-score roles are available and have not all been marked interested or applied.", "positive", strongMatches)
	}
	if manualJobs > 0 {
		review.addFinding("decision_queue", "Manual decisions are blocking cleanup", "Some collected pages need your judgement before I can classify them confidently.", "warning", manualJobs)
		review.addDecision("Can I open the manual-check queue for your decision?", "Your decision lets me reduce noise and improve the queue.", "review_manual_check")
	}
	if staleTasks > 0 || escalatedTasks > 0 {
		review.addFinding("follow_up", "Some tasks are aging", "Open work has passed the configured SLA and needs attention.", "warning", staleTasks+escalatedTasks)
	}
	if appliedJobs == 0 && interestedJobs > 0 {
		review.addFinding("conversion", "Interested jobs have not converted to applications", "The next bottleneck is moving selected roles into applied status.", "warning", interestedJobs)
	}
	if len(review.Findings) == 0 {
		review.addFinding("steady", "No blocking issue found", "I will keep watching scheduled crawls and report changes.", "info", openTasks)
	}

	review.NextSteps = buildReviewSteps(strongMatches, manualJobs, sourceIssues, len(runs), enabledSources, openTasks)
	review.Focus = buildReviewFocus(review.NextSteps)
	review.Health = buildReviewHealth(enabledSources, sourceIssues, manualJobs, openTasks, staleTasks, escalatedTasks, len(runs))
	sort.SliceStable(review.Findings, func(i, j int) bool {
		return findingWeight(review.Findings[i]) > findingWeight(review.Findings[j])
	})
	return review
}

func (r *AgentReview) addFinding(kind string, title string, detail string, level string, metric int) {
	r.Findings = append(r.Findings, AgentReviewFinding{
		Kind:   kind,
		Title:  title,
		Detail: detail,
		Level:  level,
		Metric: metric,
	})
}

func (r *AgentReview) addDecision(question string, context string, action string) {
	r.Decisions = append(r.Decisions, AgentReviewDecision{
		Question: question,
		Context:  context,
		Action:   action,
	})
}

func buildReviewSteps(strongMatches int, manualJobs int, sourceIssues int, runCount int, enabledSources int, openTasks int) []AgentReviewStep {
	steps := []AgentReviewStep{}
	if enabledSources == 0 {
		steps = append(steps, AgentReviewStep{Label: "Add recommended sources and crawl", Reason: "Without sources I cannot work autonomously.", Action: "add_recommended_and_crawl"})
	}
	if sourceIssues > 0 {
		steps = append(steps, AgentReviewStep{Label: "Inspect source issues", Reason: "Fixing source visibility protects future discovery.", Action: "inspect_failed_sources"})
	}
	if runCount == 0 {
		steps = append(steps, AgentReviewStep{Label: "Run the first crawl", Reason: "A crawl result gives me evidence to score and prioritize.", Action: "run_crawl"})
	}
	if strongMatches > 0 {
		steps = append(steps, AgentReviewStep{Label: "Review strong matches", Reason: "High-score roles should be decided before they go stale.", Action: "review_strong_matches"})
	}
	if manualJobs > 0 {
		steps = append(steps, AgentReviewStep{Label: "Clear manual decisions", Reason: "This lowers noise and improves future recommendations.", Action: "review_manual_check"})
	}
	if openTasks > 0 {
		steps = append(steps, AgentReviewStep{Label: "Refresh the daily queue", Reason: "Keep the work list synchronized with the latest pipeline state.", Action: "refresh_tasks"})
	}
	if len(steps) == 0 {
		steps = append(steps, AgentReviewStep{Label: "Keep monitoring", Reason: "The current queue is under control.", Action: "keep_monitoring"})
	}
	if len(steps) > 4 {
		return steps[:4]
	}
	return steps
}

func buildReviewFocus(steps []AgentReviewStep) AgentReviewFocus {
	if len(steps) == 0 {
		return AgentReviewFocus{Title: "Keep monitoring", Detail: "No urgent action is blocking the current workflow.", Action: "keep_monitoring"}
	}
	return AgentReviewFocus{
		Title:  steps[0].Label,
		Detail: steps[0].Reason,
		Action: steps[0].Action,
	}
}

func buildReviewHealth(enabledSources int, sourceIssues int, manualJobs int, openTasks int, staleTasks int, escalatedTasks int, runCount int) AgentReviewHealth {
	score := 100
	if enabledSources == 0 {
		score -= 35
	}
	if runCount == 0 {
		score -= 20
	}
	score -= sourceIssues * 15
	score -= manualJobs * 5
	score -= staleTasks * 8
	score -= escalatedTasks * 15
	if openTasks > 8 {
		score -= 10
	}
	if score < 0 {
		score = 0
	}
	if sourceIssues > 0 || manualJobs > 0 || staleTasks > 0 || escalatedTasks > 0 {
		return AgentReviewHealth{Score: score, Label: "Needs review", Reason: "The agent can work, but decisions or source issues need attention."}
	}
	switch {
	case score >= 85:
		return AgentReviewHealth{Score: score, Label: "Healthy", Reason: "Collection, review, and follow-up are in a good state."}
	case score >= 60:
		return AgentReviewHealth{Score: score, Label: "Needs review", Reason: "The agent can work, but decisions or source issues need attention."}
	default:
		return AgentReviewHealth{Score: score, Label: "Needs setup", Reason: "Autonomous work is blocked until sources, crawls, or decisions are handled."}
	}
}

func findingWeight(item AgentReviewFinding) int {
	switch item.Level {
	case "critical":
		return 300 + item.Metric
	case "warning":
		return 200 + item.Metric
	case "positive":
		return 100 + item.Metric
	default:
		return item.Metric
	}
}

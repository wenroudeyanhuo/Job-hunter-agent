package jobs

import (
	"context"
	"strings"
	"time"
)

type OnboardingHealth struct {
	GeneratedAt       time.Time              `json:"generated_at"`
	DatabaseReady     bool                   `json:"database_ready"`
	SchedulerExpected bool                   `json:"scheduler_expected"`
	FeishuConfigured  bool                   `json:"feishu_configured"`
	ModelConfigured   bool                   `json:"model_configured"`
	SourcePoolReady   bool                   `json:"source_pool_ready"`
	ProfileReady      bool                   `json:"profile_ready"`
	HasCrawlHistory   bool                   `json:"has_crawl_history"`
	OpenTasks         int                    `json:"open_tasks"`
	ReadinessScore    int                    `json:"readiness_score"`
	NextSteps         []string               `json:"next_steps"`
	WizardSteps       []OnboardingWizardStep `json:"wizard_steps"`
}

type OnboardingWizardStep struct {
	Key       string `json:"key"`
	Title     string `json:"title"`
	Detail    string `json:"detail"`
	Done      bool   `json:"done"`
	Action    string `json:"action"`
	ActionURL string `json:"action_url"`
}

func (r *Repository) BuildOnboardingHealth(ctx context.Context, schedulerExpected bool, feishuConfigured bool, modelConfigured bool, now time.Time) (OnboardingHealth, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	health := OnboardingHealth{
		GeneratedAt:       now.UTC(),
		DatabaseReady:     true,
		SchedulerExpected: schedulerExpected,
		FeishuConfigured:  feishuConfigured,
		ModelConfigured:   modelConfigured,
	}
	sources, err := r.ListSources(ctx, false)
	if err != nil {
		return OnboardingHealth{}, err
	}
	runs, err := r.ListRuns(ctx)
	if err != nil {
		return OnboardingHealth{}, err
	}
	settings, err := r.GetSettings(ctx)
	if err != nil {
		return OnboardingHealth{}, err
	}
	tasks, err := r.ListAgentTasks(ctx, agentTaskDate(now))
	if err != nil {
		return OnboardingHealth{}, err
	}
	profile, err := r.GetCandidateProfile(ctx)
	if err != nil {
		return OnboardingHealth{}, err
	}
	profileSaved, err := r.hasSavedCandidateProfile(ctx)
	if err != nil {
		return OnboardingHealth{}, err
	}
	for _, source := range sources {
		if source.Enabled {
			health.SourcePoolReady = true
			break
		}
	}
	health.HasCrawlHistory = len(runs) > 0
	health.ProfileReady = profileSaved && len(profile.TargetCities) > 0 && len(profile.TargetDirections) > 0 && (len(profile.Skills) > 0 || strings.TrimSpace(profile.Notes) != "")
	for _, task := range tasks {
		if task.Status != AgentTaskStatusDone {
			health.OpenTasks++
		}
	}
	if !health.SourcePoolReady {
		health.NextSteps = append(health.NextSteps, "Add recommended sources or run source discovery.")
	}
	if !health.HasCrawlHistory {
		health.NextSteps = append(health.NextSteps, "Run the first crawl to create a baseline.")
	}
	if !health.ProfileReady {
		health.NextSteps = append(health.NextSteps, "Complete your candidate profile.")
	}
	if !health.FeishuConfigured {
		health.NextSteps = append(health.NextSteps, "Optional: configure a Feishu webhook for reports.")
	}
	if !settings.AutoDutyReportEnabled {
		health.NextSteps = append(health.NextSteps, "Optional: enable automatic duty reports.")
	}
	health.WizardSteps = buildOnboardingWizardSteps(health, settings)
	health.ReadinessScore = onboardingReadinessScore(health)
	return health, nil
}

func (r *Repository) hasSavedCandidateProfile(ctx context.Context) (bool, error) {
	var count int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM candidate_profiles WHERE id = ?`, candidateProfileID).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func buildOnboardingWizardSteps(health OnboardingHealth, settings Settings) []OnboardingWizardStep {
	return []OnboardingWizardStep{
		{
			Key:       "profile",
			Title:     "Complete personal profile",
			Detail:    "Set target cities, directions, skills, preferred companies, and blocked keywords.",
			Done:      health.ProfileReady,
			Action:    "Open profile",
			ActionURL: "/profile",
		},
		{
			Key:       "sources",
			Title:     "Prepare source pool",
			Detail:    "Seed recommended sources or let Source Scout discover broader company and platform candidates.",
			Done:      health.SourcePoolReady,
			Action:    "Discover sources",
			ActionURL: "/settings",
		},
		{
			Key:       "crawl",
			Title:     "Run first crawl",
			Detail:    "Create the baseline job dataset so the agent can analyze real opportunities.",
			Done:      health.HasCrawlHistory,
			Action:    "Run crawl",
			ActionURL: "/runs",
		},
		{
			Key:       "model",
			Title:     "Connect DeepSeek or compatible model",
			Detail:    "Enable model chat for natural explanations while keeping local rules as fallback.",
			Done:      health.ModelConfigured,
			Action:    "Configure model",
			ActionURL: "/settings",
		},
		{
			Key:       "reports",
			Title:     "Enable daily reports",
			Detail:    "Send daily tasks, source issues, recommended jobs, and Agent Cycle summaries to Feishu.",
			Done:      health.FeishuConfigured && settings.AutoDutyReportEnabled,
			Action:    "Open reports settings",
			ActionURL: "/settings",
		},
	}
}

func onboardingReadinessScore(health OnboardingHealth) int {
	score := 0
	if health.DatabaseReady {
		score += 20
	}
	if health.SchedulerExpected {
		score += 10
	}
	if health.SourcePoolReady {
		score += 20
	}
	if health.ProfileReady {
		score += 20
	}
	if health.HasCrawlHistory {
		score += 20
	}
	if health.FeishuConfigured {
		score += 5
	}
	if health.ModelConfigured {
		score += 5
	}
	if score > 100 {
		return 100
	}
	return score
}

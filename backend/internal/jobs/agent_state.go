package jobs

import (
	"time"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/domain"
)

type AgentState struct {
	GeneratedAt    time.Time              `json:"generated_at"`
	Profile        AgentProfile           `json:"profile"`
	Mode           string                 `json:"mode"`
	Focus          string                 `json:"focus"`
	MaturityScore  int                    `json:"maturity_score"`
	Workload       AgentWorkload          `json:"workload"`
	Capabilities   []AgentCapability      `json:"capabilities"`
	Gaps           []AgentCapabilityGap   `json:"gaps"`
	OperatingCycle []AgentOperatingMoment `json:"operating_cycle"`
}

type AgentProfile struct {
	Name     string `json:"name"`
	Role     string `json:"role"`
	Mission  string `json:"mission"`
	Avatar   string `json:"avatar"`
	Presence string `json:"presence"`
}

type AgentWorkload struct {
	OpenTasks       int `json:"open_tasks"`
	DoneTasks       int `json:"done_tasks"`
	StrongMatches   int `json:"strong_matches"`
	ManualDecisions int `json:"manual_decisions"`
	SourceIssues    int `json:"source_issues"`
}

type AgentCapability struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	Status   string `json:"status"`
	Level    int    `json:"level"`
	Evidence string `json:"evidence"`
}

type AgentCapabilityGap struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	Why      string `json:"why"`
	NextStep string `json:"next_step"`
}

type AgentOperatingMoment struct {
	Time  string `json:"time"`
	Title string `json:"title"`
	State string `json:"state"`
}

func BuildAgentState(jobList []domain.Job, sources []Source, runs []domain.JobRun, tasks []AgentTask, settings Settings) AgentState {
	state := AgentState{
		GeneratedAt: time.Now().UTC(),
		Profile: AgentProfile{
			Name:     "Qiu Zhao",
			Role:     "Recruiting digital employee",
			Mission:  "Watch openings, prioritize matches, and turn signals into daily job-hunting work.",
			Avatar:   "/assets/job-agent-avatar.png",
			Presence: "online",
		},
		Mode:           "monitoring",
		Focus:          "Keep the recruitment pipeline moving.",
		OperatingCycle: buildOperatingCycle(settings.CrawlSchedule),
	}

	for _, task := range tasks {
		if task.Status == AgentTaskStatusDone {
			state.Workload.DoneTasks++
		} else {
			state.Workload.OpenTasks++
		}
	}
	for _, job := range jobList {
		if job.MatchScore >= 70 && job.Status != domain.StatusApplied && job.Status != domain.StatusIgnored {
			state.Workload.StrongMatches++
		}
		if job.Status == domain.StatusManualCheck {
			state.Workload.ManualDecisions++
		}
	}
	enabledSources := 0
	for _, source := range sources {
		if !source.Enabled {
			continue
		}
		enabledSources++
		if source.HealthStatus == SourceHealthBroken || source.HealthStatus == SourceHealthWarning {
			state.Workload.SourceIssues++
		}
	}

	state.Capabilities = []AgentCapability{
		{
			Key:      "collection",
			Label:    "Public source collection",
			Status:   capabilityStatus(enabledSources > 0),
			Level:    capabilityLevel(enabledSources > 0, 75),
			Evidence: itoa(enabledSources) + " enabled sources",
		},
		{
			Key:      "screening",
			Label:    "Scoring and filtering",
			Status:   capabilityStatus(len(jobList) > 0),
			Level:    capabilityLevel(len(jobList) > 0, 70),
			Evidence: itoa(len(jobList)) + " tracked jobs",
		},
		{
			Key:      "work_loop",
			Label:    "Daily task loop",
			Status:   capabilityStatus(len(tasks) > 0),
			Level:    capabilityLevel(len(tasks) > 0, 68),
			Evidence: itoa(state.Workload.OpenTasks) + " open tasks",
		},
		{
			Key:      "notification",
			Label:    "Feishu notification",
			Status:   capabilityStatus(settings.FeishuWebhookURL != ""),
			Level:    capabilityLevel(settings.FeishuWebhookURL != "", 55),
			Evidence: notificationEvidence(settings.FeishuWebhookURL),
		},
		{
			Key:      "memory",
			Label:    "Local memory",
			Status:   "active",
			Level:    60,
			Evidence: "SQLite stores jobs, runs, sources, tasks, settings, and events",
		},
	}
	state.Gaps = []AgentCapabilityGap{
		{
			Key:      "conversation",
			Label:    "Conversational command center",
			Why:      "Mainstream digital employees usually accept natural-language instructions and explain decisions.",
			NextStep: "Add a command inbox that turns user intent into task updates and settings changes.",
		},
		{
			Key:      "autonomy",
			Label:    "Autonomous follow-up",
			Why:      "The agent can plan daily work, but it still waits for manual refresh and explicit review actions.",
			NextStep: "Add reminders, stale-task escalation, and automatic daily Feishu duty reports.",
		},
		{
			Key:      "application_assist",
			Label:    "Resume and application assistance",
			Why:      "It does not yet match a resume to jobs or prepare application material.",
			NextStep: "Add resume profile, fit analysis, and human-approved application drafts.",
		},
	}
	state.MaturityScore = averageCapabilityLevel(state.Capabilities)
	if state.Workload.SourceIssues > 0 {
		state.Mode = "needs_attention"
		state.Focus = "Source health is blocking reliable monitoring."
	} else if state.Workload.OpenTasks > 0 {
		state.Mode = "on_duty"
		state.Focus = "There is recruiting work waiting for your decision."
	}
	return state
}

func buildOperatingCycle(schedule []string) []AgentOperatingMoment {
	if len(schedule) == 0 {
		schedule = []string{"09:00", "12:00", "18:00"}
	}
	out := make([]AgentOperatingMoment, 0, len(schedule))
	for _, item := range schedule {
		out = append(out, AgentOperatingMoment{
			Time:  item,
			Title: "Collect and refresh queue",
			State: "scheduled",
		})
	}
	return out
}

func capabilityStatus(ready bool) string {
	if ready {
		return "active"
	}
	return "setup_needed"
}

func capabilityLevel(ready bool, level int) int {
	if ready {
		return level
	}
	return 15
}

func notificationEvidence(webhookURL string) string {
	if webhookURL == "" {
		return "Webhook not configured"
	}
	return "Webhook configured in settings"
}

func averageCapabilityLevel(items []AgentCapability) int {
	if len(items) == 0 {
		return 0
	}
	total := 0
	for _, item := range items {
		total += item.Level
	}
	return total / len(items)
}

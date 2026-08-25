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
	Automation     AgentAutomationState   `json:"automation"`
	Memory         AgentMemory            `json:"memory"`
	Cycle          AgentCycleState        `json:"cycle"`
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
	OpenTasks        int `json:"open_tasks"`
	DoneTasks        int `json:"done_tasks"`
	StrongMatches    int `json:"strong_matches"`
	ManualDecisions  int `json:"manual_decisions"`
	SourceIssues     int `json:"source_issues"`
	ActivePlans      int `json:"active_plans"`
	PendingApprovals int `json:"pending_approvals"`
	CompletedPlans   int `json:"completed_plans"`
}

type AgentMemory struct {
	LastReviewAt       *time.Time `json:"last_review_at,omitempty"`
	LastTriggerType    string     `json:"last_trigger_type"`
	LastFocusTitle     string     `json:"last_focus_title"`
	LastFocusAction    string     `json:"last_focus_action"`
	TrendSummary       string     `json:"trend_summary"`
	RecentActionCount  int        `json:"recent_action_count"`
	SemanticTotalItems int        `json:"semantic_total_items"`
	SemanticJobItems   int        `json:"semantic_job_items"`
	SemanticProvider   string     `json:"semantic_provider"`
	SemanticDimension  int        `json:"semantic_dimension"`
}

type AgentCycleState struct {
	LastCycleAt          *time.Time `json:"last_cycle_at,omitempty"`
	Summary              string     `json:"summary"`
	ReadinessScore       int        `json:"readiness_score"`
	TraceCount           int        `json:"trace_count"`
	ActionCount          int        `json:"action_count"`
	OrchestratorProvider string     `json:"orchestrator_provider"`
	OrchestratorPattern  string     `json:"orchestrator_pattern"`
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
	return BuildAgentStateWithMemory(jobList, sources, runs, tasks, settings, nil, nil)
}

func BuildAgentStateWithMemory(jobList []domain.Job, sources []Source, runs []domain.JobRun, tasks []AgentTask, settings Settings, snapshots []AgentReviewSnapshot, events []AgentEvent) AgentState {
	return BuildAgentStateWithAgentWork(jobList, sources, runs, tasks, settings, snapshots, events, nil, nil)
}

func BuildAgentStateWithAgentWork(jobList []domain.Job, sources []Source, runs []domain.JobRun, tasks []AgentTask, settings Settings, snapshots []AgentReviewSnapshot, events []AgentEvent, plans []AgentPlan, actionRequests []AgentActionRequest) AgentState {
	return BuildAgentStateWithSemanticMemory(jobList, sources, runs, tasks, settings, snapshots, events, plans, actionRequests, SemanticMemoryStats{})
}

func BuildAgentStateWithSemanticMemory(jobList []domain.Job, sources []Source, runs []domain.JobRun, tasks []AgentTask, settings Settings, snapshots []AgentReviewSnapshot, events []AgentEvent, plans []AgentPlan, actionRequests []AgentActionRequest, semanticStats SemanticMemoryStats) AgentState {
	return BuildAgentStateWithCycles(jobList, sources, runs, tasks, settings, snapshots, events, plans, actionRequests, semanticStats, nil)
}

func BuildAgentStateWithCycles(jobList []domain.Job, sources []Source, runs []domain.JobRun, tasks []AgentTask, settings Settings, snapshots []AgentReviewSnapshot, events []AgentEvent, plans []AgentPlan, actionRequests []AgentActionRequest, semanticStats SemanticMemoryStats, cycles []AgentCycleRecord) AgentState {
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
		Automation:     BuildAgentAutomationState(settings, tasks, time.Now().UTC()),
		Memory:         BuildAgentMemoryWithSemanticStats(snapshots, events, semanticStats),
		Cycle:          BuildAgentCycleState(cycles),
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
	for _, plan := range plans {
		switch plan.Status {
		case AgentPlanStatusDone:
			state.Workload.CompletedPlans++
		case AgentPlanStatusFailed:
			continue
		default:
			state.Workload.ActivePlans++
		}
	}
	for _, request := range actionRequests {
		if request.Status == AgentActionRequestStatusPending {
			state.Workload.PendingApprovals++
		}
	}

	state.Capabilities = []AgentCapability{
		{
			Key:      "planning",
			Label:    "Plan and approval loop",
			Status:   capabilityStatus(len(plans) > 0),
			Level:    capabilityLevel(len(plans) > 0, 72),
			Evidence: itoa(state.Workload.ActivePlans) + " active plans / " + itoa(state.Workload.PendingApprovals) + " pending approvals",
		},
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
			Key:      "source_discovery",
			Label:    "Autonomous source discovery",
			Status:   capabilityStatus(settings.AutoSourceDiscoveryEnabled),
			Level:    capabilityLevel(settings.AutoSourceDiscoveryEnabled, 62),
			Evidence: sourceDiscoveryEvidence(settings),
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
			Label:    "Semantic memory",
			Status:   "active",
			Level:    semanticMemoryCapabilityLevel(semanticStats),
			Evidence: semanticMemoryEvidence(semanticStats),
		},
		{
			Key:      "multi_agent_cycle",
			Label:    "Multi-agent operating cycle",
			Status:   capabilityStatus(state.Cycle.LastCycleAt != nil),
			Level:    agentCycleCapabilityLevel(state.Cycle),
			Evidence: agentCycleEvidence(state.Cycle),
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
			Why:      "The agent can run scheduled reports and source discovery, but it still needs human approval before application actions.",
			NextStep: "Add approval-gated resume matching and application draft preparation.",
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
	} else if state.Workload.PendingApprovals > 0 {
		state.Mode = "waiting_approval"
		state.Focus = "I have planned work waiting for your approval."
	} else if state.Workload.OpenTasks > 0 || state.Workload.ActivePlans > 0 {
		state.Mode = "on_duty"
		state.Focus = "There is recruiting work waiting for your decision."
	}
	if state.Cycle.LastCycleAt != nil && state.Workload.PendingApprovals > 0 {
		state.Focus = "My latest multi-agent cycle produced work waiting for your approval."
	}
	return state
}

func BuildAgentCycleState(cycles []AgentCycleRecord) AgentCycleState {
	if len(cycles) == 0 {
		return AgentCycleState{}
	}
	latest := cycles[0]
	generatedAt := latest.GeneratedAt
	return AgentCycleState{
		LastCycleAt:          &generatedAt,
		Summary:              latest.Summary,
		ReadinessScore:       latest.ReadinessScore,
		TraceCount:           len(latest.Trace),
		ActionCount:          len(latest.Actions),
		OrchestratorProvider: latest.OrchestratorProvider,
		OrchestratorPattern:  latest.OrchestratorPattern,
	}
}

func BuildAgentMemory(snapshots []AgentReviewSnapshot, events []AgentEvent) AgentMemory {
	return BuildAgentMemoryWithSemanticStats(snapshots, events, SemanticMemoryStats{})
}

func BuildAgentMemoryWithSemanticStats(snapshots []AgentReviewSnapshot, events []AgentEvent, semanticStats SemanticMemoryStats) AgentMemory {
	memory := AgentMemory{
		TrendSummary:       "No review memory yet. Save or generate a review snapshot after meaningful work.",
		SemanticProvider:   defaultText(semanticStats.Provider, SemanticMemoryProvider),
		SemanticDimension:  semanticStats.Dimension,
		SemanticTotalItems: semanticStats.TotalItems,
		SemanticJobItems:   semanticStats.JobItems,
	}
	if memory.SemanticDimension == 0 {
		memory.SemanticDimension = SemanticMemoryDimensions
	}
	if len(snapshots) > 0 {
		latest := snapshots[0]
		capturedAt := latest.CapturedAt
		memory.LastReviewAt = &capturedAt
		memory.LastTriggerType = latest.TriggerType
		memory.LastFocusTitle = latest.FocusTitle
		memory.LastFocusAction = latest.FocusAction
		memory.TrendSummary = BuildAgentReviewHistory(snapshots).Summary
	}
	for _, event := range events {
		if event.Type == "agent_action_executed" || event.Type == "crawl_completed" || event.Type == "agent_action_crawl_completed" || event.Type == "feishu_report_sent" {
			memory.RecentActionCount++
		}
	}
	return memory
}

func semanticMemoryCapabilityLevel(stats SemanticMemoryStats) int {
	if stats.TotalItems > 0 {
		return 72
	}
	return 35
}

func semanticMemoryEvidence(stats SemanticMemoryStats) string {
	if stats.TotalItems == 0 {
		return "No vectorized memory yet"
	}
	return itoa(stats.TotalItems) + " vectorized memories / " + itoa(stats.JobItems) + " job memories"
}

func agentCycleCapabilityLevel(cycle AgentCycleState) int {
	if cycle.LastCycleAt == nil {
		return 20
	}
	if cycle.ReadinessScore > 0 {
		return maxInt(55, cycle.ReadinessScore)
	}
	return 55
}

func agentCycleEvidence(cycle AgentCycleState) string {
	if cycle.LastCycleAt == nil {
		return "No multi-agent cycle recorded yet"
	}
	return itoa(cycle.TraceCount) + " agents / " + itoa(cycle.ActionCount) + " proposed actions"
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

func sourceDiscoveryEvidence(settings Settings) string {
	if !settings.AutoSourceDiscoveryEnabled {
		return "Automatic discovery is disabled"
	}
	return "Runs every " + itoa(settings.SourceDiscoveryIntervalHours) + "h"
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

func maxInt(left int, right int) int {
	if left > right {
		return left
	}
	return right
}

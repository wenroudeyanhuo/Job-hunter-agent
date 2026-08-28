package jobs

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/domain"
)

const (
	MultiAgentSourceScout  = "source_scout"
	MultiAgentJobAnalyst   = "job_analyst"
	MultiAgentMemoryKeeper = "memory_keeper"
	MultiAgentPlanner      = "planner"
	MultiAgentObserver     = "observer"

	MultiAgentOrchestratorEinoReady     = "eino_ready"
	MultiAgentOrchestratorModelEnhanced = "model_enhanced"

	AgentAutonomyStepWaitingApproval = "waiting_approval"
	AgentAutonomyStepObserved        = "observed"
)

type RecruitingOrchestrator interface {
	Run(input MultiAgentCycleInput) MultiAgentCycle
}

type deterministicRecruitingOrchestrator struct{}

func DefaultRecruitingOrchestrator() RecruitingOrchestrator {
	return deterministicRecruitingOrchestrator{}
}

func (deterministicRecruitingOrchestrator) Run(input MultiAgentCycleInput) MultiAgentCycle {
	return RunRecruitingAgentCycle(input)
}

type MultiAgentTeam struct {
	Orchestrator MultiAgentOrchestrator `json:"orchestrator"`
	Agents       []MultiAgentRole       `json:"agents"`
}

type MultiAgentOrchestrator struct {
	Provider     string   `json:"provider"`
	Pattern      string   `json:"pattern"`
	Graph        []string `json:"graph"`
	NextStep     string   `json:"next_step"`
	ApprovalMode string   `json:"approval_mode"`
}

type MultiAgentRole struct {
	Key          string   `json:"key"`
	Name         string   `json:"name"`
	Mission      string   `json:"mission"`
	Inputs       []string `json:"inputs"`
	Outputs      []string `json:"outputs"`
	Guardrails   []string `json:"guardrails"`
	EinoNodeName string   `json:"eino_node_name"`
}

type MultiAgentCycleInput struct {
	Jobs    []domain.Job
	Sources []Source
	Tasks   []AgentTask
	Plans   []AgentPlan
	Memory  SemanticMemoryStats
	Now     time.Time
}

type MultiAgentCycle struct {
	GeneratedAt    time.Time            `json:"generated_at"`
	Team           MultiAgentTeam       `json:"team"`
	Summary        string               `json:"summary"`
	ReadinessScore int                  `json:"readiness_score"`
	Trace          []MultiAgentTrace    `json:"trace"`
	Actions        []AgentCommandAction `json:"actions"`
	AutonomyPlan   AgentAutonomyPlan    `json:"autonomy_plan"`
}

type MultiAgentTrace struct {
	AgentKey    string               `json:"agent_key"`
	Observation string               `json:"observation"`
	Decision    string               `json:"decision"`
	Actions     []AgentCommandAction `json:"actions"`
}

type ModelAgentInsight struct {
	AgentKey  string                    `json:"agent_key"`
	Decision  string                    `json:"decision"`
	ToolCalls []AgentStructuredToolCall `json:"tool_calls"`
	Actions   []AgentCommandAction      `json:"actions"`
}

type AgentAutonomyPlan struct {
	Mode                 string              `json:"mode"`
	Summary              string              `json:"summary"`
	NeedsApproval        bool                `json:"needs_approval"`
	ReplanAfterExecution bool                `json:"replan_after_execution"`
	Steps                []AgentAutonomyStep `json:"steps"`
}

type AgentAutonomyStep struct {
	Order            int    `json:"order"`
	Tool             string `json:"tool"`
	Target           string `json:"target"`
	Detail           string `json:"detail"`
	RiskLevel        string `json:"risk_level"`
	RequiresApproval bool   `json:"requires_approval"`
	Status           string `json:"status"`
	ObserverHint     string `json:"observer_hint"`
}

func ParseModelAgentInsights(raw string) []ModelAgentInsight {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var payload struct {
		Insights []ModelAgentInsight `json:"insights"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil
	}
	out := make([]ModelAgentInsight, 0, len(payload.Insights))
	for _, insight := range payload.Insights {
		insight.AgentKey = strings.TrimSpace(insight.AgentKey)
		insight.Decision = strings.TrimSpace(insight.Decision)
		if insight.AgentKey == "" || insight.Decision == "" {
			continue
		}
		insight.Actions = appendUniqueAgentActions(safeAgentActions(insight.Actions), actionsFromToolCalls(insight.ToolCalls)...)
		out = append(out, insight)
	}
	return out
}

func actionsFromToolCalls(calls []AgentStructuredToolCall) []AgentCommandAction {
	return ValidateAgentToolCalls(calls, NewDefaultAgentToolRegistry()).Actions
}

func BuildRecruitingAgentTeam() MultiAgentTeam {
	graph := []string{MultiAgentSourceScout, MultiAgentJobAnalyst, MultiAgentMemoryKeeper, MultiAgentPlanner, "tool_executor", MultiAgentObserver}
	return MultiAgentTeam{
		Orchestrator: MultiAgentOrchestrator{
			Provider:     MultiAgentOrchestratorEinoReady,
			Pattern:      "plan_tool_approval_observe_replan",
			Graph:        graph,
			NextStep:     "Planner creates approval-gated tool plans, executor waits for user approval, observer feeds results into the next cycle.",
			ApprovalMode: "human_approved_actions",
		},
		Agents: []MultiAgentRole{
			{
				Key:          MultiAgentSourceScout,
				Name:         "Source Scout",
				Mission:      "Watch source health and expand the recruiting source pool.",
				Inputs:       []string{"sources", "source candidates", "crawl runs"},
				Outputs:      []string{"source health decisions", "source discovery actions"},
				Guardrails:   []string{"Do not bypass anti-bot systems", "Only propose public or user-approved sources"},
				EinoNodeName: "source_scout_node",
			},
			{
				Key:          MultiAgentJobAnalyst,
				Name:         "Job Analyst",
				Mission:      "Score and explain which jobs deserve attention.",
				Inputs:       []string{"jobs", "candidate profile", "semantic memory"},
				Outputs:      []string{"strong match decisions", "manual review actions"},
				Guardrails:   []string{"Do not submit resumes", "Keep low-confidence jobs in manual review"},
				EinoNodeName: "job_analyst_node",
			},
			{
				Key:          MultiAgentMemoryKeeper,
				Name:         "Memory Keeper",
				Mission:      "Maintain semantic memory and retrieve relevant recruiting context.",
				Inputs:       []string{"jobs", "decisions", "notes", "chat history"},
				Outputs:      []string{"memory health", "retrieval readiness"},
				Guardrails:   []string{"Keep memory local by default", "Never store secrets in semantic memory"},
				EinoNodeName: "memory_keeper_node",
			},
			{
				Key:          MultiAgentPlanner,
				Name:         "Planner",
				Mission:      "Turn observations into approval-gated daily recruiting work.",
				Inputs:       []string{"agent observations", "tasks", "plans"},
				Outputs:      []string{"safe action requests", "daily work plan"},
				Guardrails:   []string{"All external actions require approval", "Prefer reversible workflow actions"},
				EinoNodeName: "planner_node",
			},
			{
				Key:          MultiAgentObserver,
				Name:         "Observer",
				Mission:      "Summarize executed tool results and decide whether the next cycle should adjust the plan.",
				Inputs:       []string{"tool execution receipts", "agent events", "latest tasks"},
				Outputs:      []string{"execution observations", "follow-up cycle triggers"},
				Guardrails:   []string{"Do not execute tools directly", "Escalate failed tool results for human review"},
				EinoNodeName: "observer_node",
			},
		},
	}
}

func BuildAgentToolObserverCycle(request AgentActionRequest, executionStatus string, executionMessage string, now time.Time) MultiAgentCycle {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	executionStatus = normalizeAgentActionExecutionStatus(executionStatus)
	success := executionStatus == AgentActionExecutionSucceeded
	decision := "Tool execution completed and the work queue can continue."
	if !success {
		decision = "Tool execution failed and should be reviewed before retrying."
	}
	cycle := MultiAgentCycle{
		GeneratedAt:    now,
		Team:           BuildRecruitingAgentTeam(),
		ReadinessScore: 78,
		Trace: []MultiAgentTrace{
			{
				AgentKey:    MultiAgentObserver,
				Observation: request.ActionType + " returned " + executionStatus + ": " + executionMessage,
				Decision:    decision,
			},
			{
				AgentKey:    MultiAgentPlanner,
				Observation: "Observed latest tool execution receipt for " + request.Target + ".",
				Decision:    "Continue with approval-gated workflow actions.",
			},
		},
	}
	cycle.Team.Orchestrator.Provider = "tool_observer"
	cycle.Team.Orchestrator.Pattern = "execute_observe_replan"
	cycle.Team.Orchestrator.NextStep = "Feed this observation into the next Eino graph cycle."
	if !success {
		cycle.ReadinessScore = 55
		cycle.Actions = append(cycle.Actions, AgentCommandAction{Type: request.ActionType, Target: request.Target, Detail: "Review failed execution before retrying: " + executionMessage})
	}
	cycle.Actions = appendUniqueAgentActions(cycle.Actions, BuildObserverReplanActions(request, executionStatus, executionMessage, now)...)
	cycle.AutonomyPlan = BuildAgentAutonomyPlan(cycle.Actions, "observe_replan")
	if success {
		cycle.AutonomyPlan = AgentAutonomyPlan{
			Mode:                 "observe_replan",
			Summary:              "Observer recorded a successful tool result and prepared a re-plan proposal from the execution receipt.",
			ReplanAfterExecution: true,
			Steps:                BuildAgentAutonomyPlan(cycle.Actions, "observe_replan").Steps,
		}
		if len(cycle.AutonomyPlan.Steps) == 0 {
			cycle.AutonomyPlan.Steps = []AgentAutonomyStep{{
				Order:        1,
				Tool:         request.ActionType,
				Target:       request.Target,
				Detail:       executionMessage,
				Status:       AgentAutonomyStepObserved,
				ObserverHint: "Use this receipt as context for the next plan.",
			}}
		}
	}
	cycle.Summary = "Observer reviewed " + request.ActionType + " execution: " + executionStatus + "."
	return cycle
}

func BuildObserverReplanActions(request AgentActionRequest, executionStatus string, executionMessage string, _ time.Time) []AgentCommandAction {
	executionStatus = normalizeAgentActionExecutionStatus(executionStatus)
	message := strings.ToLower(executionMessage)
	if executionStatus == AgentActionExecutionFailed {
		return safeAgentActions([]AgentCommandAction{
			{Type: "review_parser_gaps", Target: "opportunities", Detail: "Inspect the failed tool receipt before retrying: " + executionMessage},
		})
	}
	switch strings.TrimSpace(request.ActionType) {
	case "run_crawl", "add_recommended_and_crawl":
		if strings.Contains(message, "created 0") || strings.Contains(message, "0 jobs") || strings.Contains(message, "found 0") {
			return safeAgentActions([]AgentCommandAction{
				{Type: "discover_sources", Target: "sources", Detail: "Crawl produced no new jobs; expand the source pool before the next run."},
				{Type: "validate_source_candidates", Target: "sources", Detail: "Validate pending sources so the next crawl has better inputs."},
			})
		}
		return safeAgentActions([]AgentCommandAction{
			{Type: "review_strong_matches", Target: "opportunities", Detail: "Review newly collected strong matches before they go stale."},
			{Type: "sync_application_plans", Target: "applications", Detail: "Prepare application plans for promising jobs after the latest crawl."},
		})
	case "discover_sources":
		return safeAgentActions([]AgentCommandAction{
			{Type: "validate_source_candidates", Target: "sources", Detail: "Validate newly discovered source candidates before adding them to the crawl pool."},
		})
	case "refresh_tasks":
		return safeAgentActions([]AgentCommandAction{
			{Type: "send_feishu_report", Target: "notification", Detail: "Send the refreshed daily work queue to Feishu if configured."},
		})
	case "rebuild_semantic_memory":
		return safeAgentActions([]AgentCommandAction{
			{Type: "review_strong_matches", Target: "opportunities", Detail: "Use refreshed memory to review the best matching opportunities."},
		})
	default:
		return nil
	}
}

func RunRecruitingAgentCycle(input MultiAgentCycleInput) MultiAgentCycle {
	now := input.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	cycle := MultiAgentCycle{
		GeneratedAt: now,
		Team:        BuildRecruitingAgentTeam(),
	}
	cycle.Trace = append(cycle.Trace, runSourceScoutAgent(input))
	cycle.Trace = append(cycle.Trace, runJobAnalystAgent(input))
	cycle.Trace = append(cycle.Trace, runMemoryKeeperAgent(input))
	cycle.Trace = append(cycle.Trace, runPlannerAgent(input))
	for _, trace := range cycle.Trace {
		cycle.Actions = appendUniqueAgentActions(cycle.Actions, trace.Actions...)
	}
	cycle.ReadinessScore = multiAgentReadinessScore(input, cycle.Actions)
	cycle.AutonomyPlan = BuildAgentAutonomyPlan(cycle.Actions, "approval_gated_replan")
	cycle.Summary = buildMultiAgentCycleSummary(input, cycle)
	return cycle
}

func BuildAgentAutonomyPlan(actions []AgentCommandAction, mode string) AgentAutonomyPlan {
	if strings.TrimSpace(mode) == "" {
		mode = "approval_gated_replan"
	}
	registry := NewDefaultAgentToolRegistry()
	plan := AgentAutonomyPlan{
		Mode:                 mode,
		ReplanAfterExecution: len(actions) > 0,
		Steps:                []AgentAutonomyStep{},
	}
	for _, action := range safeAgentActions(actions) {
		tool, _ := registry.Get(action.Type)
		step := AgentAutonomyStep{
			Order:            len(plan.Steps) + 1,
			Tool:             action.Type,
			Target:           action.Target,
			Detail:           action.Detail,
			RiskLevel:        defaultText(tool.RiskLevel, AgentToolRiskLow),
			RequiresApproval: true,
			Status:           AgentAutonomyStepWaitingApproval,
			ObserverHint:     "After approval and execution, Observer should summarize the receipt and trigger a new planning pass.",
		}
		if tool.RequiresApproval {
			plan.NeedsApproval = true
			step.RequiresApproval = true
		}
		plan.Steps = append(plan.Steps, step)
	}
	if len(plan.Steps) == 0 {
		plan.Summary = "No tool calls are needed right now; the agent will observe the next crawl or user decision."
		return plan
	}
	plan.Summary = fmt.Sprintf("Prepared %d approval-gated tool steps. Execute only after user approval, then observe and re-plan.", len(plan.Steps))
	return plan
}

func ApplyModelAgentInsights(cycle MultiAgentCycle, insights []ModelAgentInsight) MultiAgentCycle {
	if len(insights) == 0 {
		return cycle
	}
	cycle.Team.Orchestrator.Provider = MultiAgentOrchestratorModelEnhanced
	cycle.Team.Orchestrator.NextStep = "Model insights enhanced specialist decisions; deterministic fallback remains available."
	for _, insight := range insights {
		agentKey := strings.TrimSpace(insight.AgentKey)
		for index := range cycle.Trace {
			if cycle.Trace[index].AgentKey != agentKey {
				continue
			}
			if strings.TrimSpace(insight.Decision) != "" {
				cycle.Trace[index].Decision = strings.TrimSpace(insight.Decision)
			}
			safeActions := safeAgentActions(insight.Actions)
			cycle.Trace[index].Actions = appendUniqueAgentActions(cycle.Trace[index].Actions, safeActions...)
			cycle.Actions = appendUniqueAgentActions(cycle.Actions, safeActions...)
		}
	}
	cycle.ReadinessScore = cycle.ReadinessScore - 2
	if cycle.ReadinessScore < 0 {
		cycle.ReadinessScore = 0
	}
	cycle.AutonomyPlan = BuildAgentAutonomyPlan(cycle.Actions, "model_enhanced_replan")
	cycle.Summary = cycle.Summary + " Model insights were applied to specialist decisions."
	return cycle
}

func safeAgentActions(actions []AgentCommandAction) []AgentCommandAction {
	out := make([]AgentCommandAction, 0, len(actions))
	for _, action := range actions {
		allowed, ok := allowedModelActionTypes[strings.TrimSpace(action.Type)]
		if !ok {
			continue
		}
		if strings.TrimSpace(action.Target) != "" {
			allowed.Target = strings.TrimSpace(action.Target)
		}
		if strings.TrimSpace(action.Detail) != "" {
			allowed.Detail = strings.TrimSpace(action.Detail)
		}
		out = append(out, allowed)
	}
	return out
}

func runSourceScoutAgent(input MultiAgentCycleInput) MultiAgentTrace {
	enabled, unhealthy := 0, 0
	for _, source := range input.Sources {
		if !source.Enabled {
			continue
		}
		enabled++
		if source.HealthStatus == SourceHealthBroken || source.HealthStatus == SourceHealthWarning {
			unhealthy++
		}
	}
	trace := MultiAgentTrace{
		AgentKey:    MultiAgentSourceScout,
		Observation: fmt.Sprintf("%d enabled sources, %d unhealthy sources", enabled, unhealthy),
		Decision:    "Source pool is usable.",
	}
	if enabled == 0 || unhealthy > 0 || enabled < 8 {
		trace.Decision = "Source pool should be expanded or repaired."
		trace.Actions = append(trace.Actions, AgentCommandAction{Type: "inspect_source_health", Target: "sources", Detail: "Inspect source health before deciding whether to repair or expand the source pool."})
		trace.Actions = append(trace.Actions, AgentCommandAction{Type: "discover_sources", Target: "sources", Detail: "Find additional recruiting source candidates."})
	}
	if unhealthy > 0 || enabled < 8 {
		trace.Actions = append(trace.Actions, AgentCommandAction{Type: "validate_source_candidates", Target: "sources", Detail: "Validate pending source candidates before promoting them into the crawl pool."})
	}
	return trace
}

func runJobAnalystAgent(input MultiAgentCycleInput) MultiAgentTrace {
	strong, manual, parserGaps := 0, 0, 0
	for _, job := range input.Jobs {
		if job.MatchScore >= 70 && job.Status != domain.StatusApplied && job.Status != domain.StatusIgnored {
			strong++
		}
		if job.Status == domain.StatusManualCheck {
			manual++
		}
		if job.Status == domain.StatusManualCheck && jobLooksLikeParserGap(job) {
			parserGaps++
		}
	}
	trace := MultiAgentTrace{
		AgentKey:    MultiAgentJobAnalyst,
		Observation: fmt.Sprintf("%d strong matches, %d manual decisions, %d parser gaps", strong, manual, parserGaps),
		Decision:    "No immediate job review required.",
	}
	if strong > 0 {
		trace.Decision = "Strong matches should be reviewed before they go stale."
		trace.Actions = append(trace.Actions, AgentCommandAction{Type: "review_strong_matches", Target: "opportunities", Detail: "Review high-score jobs."})
	}
	if manual > 0 {
		trace.Actions = append(trace.Actions, AgentCommandAction{Type: "review_manual_check", Target: "opportunities", Detail: "Resolve jobs that need manual decisions."})
	}
	if parserGaps > 0 {
		trace.Decision = "Some collected pages look like parser gaps and should be inspected so future crawls improve."
		trace.Actions = append(trace.Actions, AgentCommandAction{Type: "review_parser_gaps", Target: "opportunities", Detail: "Inspect low-confidence manual-check pages and decide which parser/source needs improvement."})
	}
	return trace
}

func runMemoryKeeperAgent(input MultiAgentCycleInput) MultiAgentTrace {
	trace := MultiAgentTrace{
		AgentKey:    MultiAgentMemoryKeeper,
		Observation: fmt.Sprintf("%d semantic memories, %d job memories", input.Memory.TotalItems, input.Memory.JobItems),
		Decision:    "Semantic memory is available for retrieval.",
	}
	if input.Memory.JobItems < len(input.Jobs) {
		trace.Decision = "Semantic memory should be rebuilt or backfilled."
		trace.Actions = append(trace.Actions, AgentCommandAction{Type: "rebuild_semantic_memory", Target: "memory", Detail: "Rebuild semantic memory from tracked jobs."})
	}
	return trace
}

func runPlannerAgent(input MultiAgentCycleInput) MultiAgentTrace {
	openTasks, activePlans := 0, 0
	for _, task := range input.Tasks {
		if task.Status != AgentTaskStatusDone {
			openTasks++
		}
	}
	for _, plan := range input.Plans {
		if plan.Status != AgentPlanStatusDone && plan.Status != AgentPlanStatusFailed {
			activePlans++
		}
	}
	trace := MultiAgentTrace{
		AgentKey:    MultiAgentPlanner,
		Observation: fmt.Sprintf("%d open tasks, %d active plans", openTasks, activePlans),
		Decision:    "Planning loop is under control.",
	}
	if openTasks == 0 {
		trace.Actions = append(trace.Actions, AgentCommandAction{Type: "generate_daily_plan", Target: "daily_tasks", Detail: "Generate today's recruiting plan from latest jobs, sources, and memory."})
		trace.Actions = append(trace.Actions, AgentCommandAction{Type: "refresh_tasks", Target: "daily_tasks", Detail: "Refresh today's task queue."})
	}
	if activePlans == 0 {
		trace.Actions = append(trace.Actions, AgentCommandAction{Type: "run_crawl", Target: "sources", Detail: "Run a crawl if the queue needs fresh evidence."})
	}
	return trace
}

func appendUniqueAgentActions(existing []AgentCommandAction, next ...AgentCommandAction) []AgentCommandAction {
	seen := map[string]bool{}
	for _, action := range existing {
		seen[action.Type+"|"+action.Target] = true
	}
	for _, action := range next {
		key := action.Type + "|" + action.Target
		if seen[key] {
			continue
		}
		seen[key] = true
		existing = append(existing, action)
	}
	return existing
}

func multiAgentReadinessScore(input MultiAgentCycleInput, actions []AgentCommandAction) int {
	score := 100
	score -= len(actions) * 8
	if len(input.Sources) == 0 {
		score -= 20
	}
	if input.Memory.JobItems == 0 && len(input.Jobs) > 0 {
		score -= 15
	}
	if score < 0 {
		return 0
	}
	return score
}

func buildMultiAgentCycleSummary(input MultiAgentCycleInput, cycle MultiAgentCycle) string {
	names := make([]string, 0, len(cycle.Trace))
	for _, trace := range cycle.Trace {
		names = append(names, trace.AgentKey)
	}
	return fmt.Sprintf("Ran %d recruiting agents (%s), proposed %d approval-gated actions for %d tracked jobs.", len(cycle.Trace), strings.Join(names, " -> "), len(cycle.Actions), len(input.Jobs))
}

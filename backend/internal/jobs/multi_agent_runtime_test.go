package jobs

import (
	"testing"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/domain"
)

func TestBuildRecruitingAgentTeamDefinesSpecializedAgents(t *testing.T) {
	team := BuildRecruitingAgentTeam()

	if len(team.Agents) != 5 {
		t.Fatalf("expected five specialized agents, got %#v", team.Agents)
	}
	if team.Agents[0].Key != MultiAgentSourceScout || team.Agents[1].Key != MultiAgentJobAnalyst || team.Agents[2].Key != MultiAgentMemoryKeeper || team.Agents[3].Key != MultiAgentPlanner || team.Agents[4].Key != MultiAgentObserver {
		t.Fatalf("unexpected agent order: %#v", team.Agents)
	}
	if team.Orchestrator.Provider != MultiAgentOrchestratorEinoReady {
		t.Fatalf("expected Eino-ready orchestration boundary, got %#v", team.Orchestrator)
	}
}

func TestRunRecruitingAgentCycleProducesTraceAndActions(t *testing.T) {
	cycle := RunRecruitingAgentCycle(MultiAgentCycleInput{
		Jobs: []domain.Job{
			{Company: "DeepAI", Title: "Agent Platform Go Engineer", City: "Shenzhen", MatchScore: 91, Status: domain.StatusNew},
			{Company: "Noise", Title: "Sales Intern", City: "Unknown", MatchScore: 22, Status: domain.StatusManualCheck},
		},
		Sources: []Source{
			{Name: "Tencent Careers", Enabled: true, HealthStatus: SourceHealthHealthy},
			{Name: "Broken Source", Enabled: true, HealthStatus: SourceHealthBroken},
		},
		Memory: SemanticMemoryStats{TotalItems: 1, JobItems: 1, Provider: SemanticMemoryProvider, Dimension: SemanticMemoryDimensions},
	})

	if cycle.Team.Orchestrator.Provider != MultiAgentOrchestratorEinoReady {
		t.Fatalf("expected Eino-ready team, got %#v", cycle.Team)
	}
	if len(cycle.Trace) != 4 {
		t.Fatalf("expected one trace item per agent, got %#v", cycle.Trace)
	}
	if !multiAgentHasAction(cycle.Actions, "discover_sources") {
		t.Fatalf("expected source scout to propose source discovery, got %#v", cycle.Actions)
	}
	if !multiAgentHasAction(cycle.Actions, "review_strong_matches") {
		t.Fatalf("expected analyst to propose strong match review, got %#v", cycle.Actions)
	}
	if cycle.Summary == "" || cycle.ReadinessScore <= 0 {
		t.Fatalf("expected cycle summary and readiness score, got %#v", cycle)
	}
	if cycle.AutonomyPlan.Mode != "approval_gated_replan" {
		t.Fatalf("expected cycle to build an autonomy plan, got %#v", cycle.AutonomyPlan)
	}
	if len(cycle.AutonomyPlan.Steps) == 0 || !cycle.AutonomyPlan.NeedsApproval || !cycle.AutonomyPlan.ReplanAfterExecution {
		t.Fatalf("expected approval-gated replan steps, got %#v", cycle.AutonomyPlan)
	}
	if cycle.AutonomyPlan.Steps[0].Status != AgentAutonomyStepWaitingApproval {
		t.Fatalf("expected tool plan to wait for approval, got %#v", cycle.AutonomyPlan.Steps[0])
	}
}

func TestDefaultRecruitingOrchestratorRunsDeterministicFallback(t *testing.T) {
	cycle := DefaultRecruitingOrchestrator().Run(MultiAgentCycleInput{
		Sources: []Source{{Name: "Tencent Careers", Enabled: true, HealthStatus: SourceHealthBroken}},
	})

	if cycle.Team.Orchestrator.Provider != MultiAgentOrchestratorEinoReady {
		t.Fatalf("expected fallback orchestrator to preserve Eino-ready boundary, got %#v", cycle.Team.Orchestrator)
	}
	if len(cycle.Trace) != 4 {
		t.Fatalf("expected deterministic trace, got %#v", cycle.Trace)
	}
}

func TestApplyModelAgentInsightsAnnotatesCycleAndFiltersActions(t *testing.T) {
	cycle := RunRecruitingAgentCycle(MultiAgentCycleInput{
		Jobs: []domain.Job{{Company: "Tencent", Title: "Go Backend Engineer", City: "Shenzhen", MatchScore: 90, Status: domain.StatusNew}},
	})

	enhanced := ApplyModelAgentInsights(cycle, []ModelAgentInsight{
		{
			AgentKey: MultiAgentJobAnalyst,
			Decision: "Model says Tencent backend should be reviewed today.",
			Actions: []AgentCommandAction{
				{Type: "review_strong_matches", Target: "opportunities", Detail: "Review Tencent backend first."},
				{Type: "auto_apply_resume", Target: "external", Detail: "Unsafe action"},
			},
		},
	})

	if enhanced.Team.Orchestrator.Provider != MultiAgentOrchestratorModelEnhanced {
		t.Fatalf("expected model-enhanced provider, got %#v", enhanced.Team.Orchestrator)
	}
	if !multiAgentHasAction(enhanced.Actions, "review_strong_matches") {
		t.Fatalf("expected safe model action to be kept, got %#v", enhanced.Actions)
	}
	if multiAgentHasAction(enhanced.Actions, "auto_apply_resume") {
		t.Fatalf("expected unsafe model action to be filtered, got %#v", enhanced.Actions)
	}
	if enhanced.Trace[1].Decision != "Model says Tencent backend should be reviewed today." {
		t.Fatalf("expected model decision to annotate matching trace, got %#v", enhanced.Trace[1])
	}
}

func TestRunRecruitingAgentCycleProactivelyPlansDailySelfCheck(t *testing.T) {
	cycle := RunRecruitingAgentCycle(MultiAgentCycleInput{
		Jobs: []domain.Job{
			{Company: "Tencent", Title: "Go Backend Engineer", City: "Shenzhen", MatchScore: 86, Status: domain.StatusNew},
			{Company: "ParserGap", Title: "Career home", City: "", MatchScore: 15, Status: domain.StatusManualCheck, PenaltyReasons: []string{"Low confidence job posting"}},
		},
		Sources: []Source{
			{Name: "Broken Careers", Enabled: true, HealthStatus: SourceHealthBroken},
		},
		Memory: SemanticMemoryStats{TotalItems: 0, JobItems: 0},
	})

	if !multiAgentHasAction(cycle.Actions, "validate_source_candidates") {
		t.Fatalf("expected proactive source validation action, got %#v", cycle.Actions)
	}
	if !multiAgentHasAction(cycle.Actions, "review_parser_gaps") {
		t.Fatalf("expected proactive parser-gap review action, got %#v", cycle.Actions)
	}
	if cycle.AutonomyPlan.Summary == "" || len(cycle.AutonomyPlan.Steps) < 3 {
		t.Fatalf("expected autonomy plan to include proactive daily work, got %#v", cycle.AutonomyPlan)
	}
}

func TestParseModelAgentInsightsAcceptsStructuredJSON(t *testing.T) {
	insights := ParseModelAgentInsights(`{
		"insights": [
			{
				"agent_key": "job_analyst",
				"decision": "Review Tencent first.",
				"actions": [
					{"type":"review_strong_matches","target":"opportunities","detail":"Review high score jobs."},
					{"type":"auto_apply_resume","target":"external","detail":"Unsafe"}
				]
			}
		]
	}`)

	if len(insights) != 1 {
		t.Fatalf("expected one insight, got %#v", insights)
	}
	if insights[0].AgentKey != MultiAgentJobAnalyst || len(insights[0].Actions) != 1 {
		t.Fatalf("expected parsed and filtered insight, got %#v", insights[0])
	}
}

func multiAgentHasAction(actions []AgentCommandAction, actionType string) bool {
	for _, action := range actions {
		if action.Type == actionType {
			return true
		}
	}
	return false
}

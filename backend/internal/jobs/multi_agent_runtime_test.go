package jobs

import (
	"testing"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/domain"
)

func TestBuildRecruitingAgentTeamDefinesSpecializedAgents(t *testing.T) {
	team := BuildRecruitingAgentTeam()

	if len(team.Agents) != 4 {
		t.Fatalf("expected four specialized agents, got %#v", team.Agents)
	}
	if team.Agents[0].Key != MultiAgentSourceScout || team.Agents[1].Key != MultiAgentJobAnalyst || team.Agents[2].Key != MultiAgentMemoryKeeper || team.Agents[3].Key != MultiAgentPlanner {
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
}

func multiAgentHasAction(actions []AgentCommandAction, actionType string) bool {
	for _, action := range actions {
		if action.Type == actionType {
			return true
		}
	}
	return false
}

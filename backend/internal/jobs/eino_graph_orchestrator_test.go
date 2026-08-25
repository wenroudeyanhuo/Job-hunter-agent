//go:build eino

package jobs

import (
	"context"
	"testing"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/domain"
)

func TestEinoRecruitingOrchestratorRunsGraph(t *testing.T) {
	orchestrator, err := NewEinoRecruitingOrchestrator(context.Background())
	if err != nil {
		t.Fatalf("new eino orchestrator: %v", err)
	}
	cycle := orchestrator.Run(MultiAgentCycleInput{
		Jobs:    []domain.Job{{Company: "Tencent", Title: "Go Backend Engineer", City: "Shenzhen", MatchScore: 90, Status: domain.StatusNew}},
		Sources: []Source{{Name: "Tencent Careers", Enabled: true, HealthStatus: SourceHealthHealthy}},
	})

	if cycle.Team.Orchestrator.Provider != "eino_graph" {
		t.Fatalf("expected eino graph provider, got %#v", cycle.Team.Orchestrator)
	}
	if len(cycle.Trace) != 6 || cycle.Summary == "" {
		t.Fatalf("expected complete graph cycle, got %#v", cycle)
	}
	if cycle.Trace[4].AgentKey != MultiAgentPlanner || cycle.Trace[5].AgentKey != MultiAgentObserver {
		t.Fatalf("expected tool planner and observer nodes, got %#v", cycle.Trace)
	}
	if cycle.AutonomyPlan.Mode != "eino_graph" || !cycle.AutonomyPlan.ReplanAfterExecution {
		t.Fatalf("expected Eino graph to build autonomous plan, got %#v", cycle.AutonomyPlan)
	}
}

func TestConfiguredRecruitingOrchestratorUsesEinoGraph(t *testing.T) {
	orchestrator := NewConfiguredRecruitingOrchestrator(context.Background(), "eino_graph")
	cycle := orchestrator.Run(MultiAgentCycleInput{})

	if cycle.Team.Orchestrator.Provider != "eino_graph" {
		t.Fatalf("expected configured eino graph orchestrator, got %#v", cycle.Team.Orchestrator)
	}
}

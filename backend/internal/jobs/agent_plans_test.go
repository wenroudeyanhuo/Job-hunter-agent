package jobs

import (
	"context"
	"testing"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/db"
)

func TestRepositoryCreatesAndListsAgentPlans(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo := NewRepository(conn)

	created, err := repo.CreateAgentPlan(ctx, AgentPlanInput{
		Source:        "chat",
		Goal:          "帮我刷新并整理今天的岗位",
		Summary:       "先采集，再刷新今日任务。",
		RiskLevel:     AgentPlanRiskApprovalRequired,
		NeedsApproval: true,
		Steps: []AgentPlanStep{
			{Order: 1, ActionType: "run_crawl", Target: "sources", Detail: "Run a manual crawl.", Status: AgentPlanStepStatusPending},
			{Order: 2, ActionType: "refresh_tasks", Target: "tasks", Detail: "Refresh today's tasks.", Status: AgentPlanStepStatusPending},
		},
	})
	if err != nil {
		t.Fatalf("create agent plan: %v", err)
	}
	if created.Status != AgentPlanStatusWaitingApproval || len(created.Steps) != 2 {
		t.Fatalf("unexpected created plan: %#v", created)
	}

	plans, err := repo.ListAgentPlans(ctx, AgentPlanStatusWaitingApproval, 10)
	if err != nil {
		t.Fatalf("list agent plans: %v", err)
	}
	if len(plans) != 1 {
		t.Fatalf("expected one plan, got %#v", plans)
	}
	if plans[0].Goal != "帮我刷新并整理今天的岗位" || plans[0].Steps[0].ActionType != "run_crawl" {
		t.Fatalf("plan fields were not persisted: %#v", plans[0])
	}
}

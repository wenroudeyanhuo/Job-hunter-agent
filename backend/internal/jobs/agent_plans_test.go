package jobs

import (
	"context"
	"testing"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/db"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/domain"
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

func TestRepositoryFindsAgentPlanForDay(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo := NewRepository(conn)

	created, err := repo.CreateAgentPlan(ctx, AgentPlanInput{
		Source:        "automation",
		Goal:          "daily plan",
		Summary:       "Plan the day.",
		RiskLevel:     AgentPlanRiskApprovalRequired,
		NeedsApproval: true,
		Steps: []AgentPlanStep{
			{ActionType: "run_crawl", Target: "sources", Detail: "Run a manual crawl."},
		},
	})
	if err != nil {
		t.Fatalf("create plan: %v", err)
	}

	exists, err := repo.HasAgentPlanForDay(ctx, "automation", "daily plan", created.CreatedAt.Format("2006-01-02"))
	if err != nil {
		t.Fatalf("check plan for day: %v", err)
	}
	if !exists {
		t.Fatal("expected plan lookup to find created plan")
	}
}

func TestBuildAgentPlanInputFromReviewKeepsSafeNextSteps(t *testing.T) {
	review := BuildAgentReview([]domain.Job{
		{Title: "Go Backend Engineer", MatchScore: 88, Status: domain.StatusNew},
	}, nil, nil, nil)

	input := BuildAgentPlanInputFromReview(review)

	if input.Goal != "今日秋招工作计划" || !input.NeedsApproval {
		t.Fatalf("expected approval-gated daily plan, got %#v", input)
	}
	if len(input.Steps) == 0 {
		t.Fatalf("expected safe review steps to become plan steps")
	}
	if input.Steps[0].ActionType != "add_recommended_and_crawl" {
		t.Fatalf("expected source bootstrap to be planned first, got %#v", input.Steps)
	}
	for _, step := range input.Steps {
		if step.ActionType == "keep_monitoring" || step.ActionType == "inspect_failed_sources" {
			t.Fatalf("unsafe or unsupported step should not be planned directly: %#v", step)
		}
	}
}

package jobs

import (
	"context"
	"testing"
	"time"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/db"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/domain"
)

func TestAgentRuntimeCreatesReviewPlanAndActionRequests(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo := NewRepository(conn)
	runtime := NewAgentRuntime(repo)
	review := BuildAgentReview([]domain.Job{
		{Title: "AI Application Engineer", Company: "Example", City: "Shenzhen", MatchScore: 86, Status: domain.StatusNew},
	}, nil, nil, nil)

	result, err := runtime.CreateReviewPlan(ctx, AgentReviewPlanRequest{
		Review: review,
		Source: "manual",
		Now:    time.Date(2026, 8, 14, 9, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("create review plan: %v", err)
	}

	if !result.Created {
		t.Fatalf("expected plan to be created, got %#v", result)
	}
	if result.Plan.ID == 0 || result.Plan.Source != "manual" || len(result.Plan.Steps) == 0 {
		t.Fatalf("unexpected plan: %#v", result.Plan)
	}
	requests, err := repo.ListAgentActionRequests(ctx, AgentActionRequestStatusPending)
	if err != nil {
		t.Fatalf("list action requests: %v", err)
	}
	if len(requests) != len(result.Plan.Steps) {
		t.Fatalf("expected one action request per step, got requests=%#v steps=%#v", requests, result.Plan.Steps)
	}
	if requests[0].PlanID != result.Plan.ID {
		t.Fatalf("expected request to link to plan %d, got %#v", result.Plan.ID, requests[0])
	}
}

func TestAgentRuntimeSkipsDuplicateReviewPlanForSameDay(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo := NewRepository(conn)
	runtime := NewAgentRuntime(repo)
	review := BuildAgentReview(nil, nil, nil, nil)
	now := time.Date(2026, 8, 14, 9, 0, 0, 0, time.UTC)

	first, err := runtime.CreateReviewPlan(ctx, AgentReviewPlanRequest{
		Review: review,
		Source: "automation",
		Now:    now,
		Dedupe: true,
	})
	if err != nil {
		t.Fatalf("create first plan: %v", err)
	}
	second, err := runtime.CreateReviewPlan(ctx, AgentReviewPlanRequest{
		Review: review,
		Source: "automation",
		Now:    now.Add(2 * time.Hour),
		Dedupe: true,
	})
	if err != nil {
		t.Fatalf("create duplicate plan: %v", err)
	}

	if !first.Created {
		t.Fatalf("expected first plan to be created, got %#v", first)
	}
	if second.Created || second.Plan.ID != 0 {
		t.Fatalf("expected duplicate plan to be skipped, got %#v", second)
	}
	plans, err := repo.ListAgentPlans(ctx, "", 10)
	if err != nil {
		t.Fatalf("list plans: %v", err)
	}
	if len(plans) != 1 {
		t.Fatalf("expected one stored plan, got %#v", plans)
	}
}

func TestAgentRuntimeRunsAndRecordsMultiAgentCycle(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo := NewRepository(conn)
	runtime := NewAgentRuntime(repo)

	result, err := runtime.RunMultiAgentCycle(ctx, MultiAgentCycleRequest{
		Input: MultiAgentCycleInput{
			Now: time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC),
			Jobs: []domain.Job{
				{Company: "ByteDance", Title: "AI Application Engineer", City: "Shenzhen", MatchScore: 90, Status: domain.StatusNew},
			},
			Sources: []Source{
				{Name: "ByteDance Campus", Enabled: true, HealthStatus: SourceHealthBroken},
			},
			Memory: SemanticMemoryStats{TotalItems: 0, JobItems: 0},
		},
		Source:               "scheduler",
		RecordActionRequests: true,
	})
	if err != nil {
		t.Fatalf("run multi-agent cycle: %v", err)
	}
	if result.Cycle.ID == 0 || len(result.Cycle.Trace) != 4 {
		t.Fatalf("expected recorded cycle with trace, got %#v", result.Cycle)
	}
	if result.ActionRequestsCreated == 0 {
		t.Fatalf("expected safe action requests to be created, got %#v", result)
	}

	cycles, err := repo.ListAgentCycles(ctx, 10)
	if err != nil {
		t.Fatalf("list cycles: %v", err)
	}
	if len(cycles) != 1 || cycles[0].ID != result.Cycle.ID {
		t.Fatalf("expected persisted runtime cycle, got %#v", cycles)
	}
	requests, err := repo.ListAgentActionRequests(ctx, AgentActionRequestStatusPending)
	if err != nil {
		t.Fatalf("list action requests: %v", err)
	}
	if len(requests) != result.ActionRequestsCreated {
		t.Fatalf("expected runtime-created action requests, got result=%#v requests=%#v", result, requests)
	}
}

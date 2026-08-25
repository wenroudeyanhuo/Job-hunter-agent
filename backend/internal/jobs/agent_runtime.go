package jobs

import (
	"context"
	"strings"
	"time"
)

type AgentRuntime struct {
	repo *Repository
}

type AgentReviewPlanRequest struct {
	Review AgentReview
	Source string
	Now    time.Time
	Dedupe bool
}

type AgentPlanResult struct {
	Plan    AgentPlan `json:"plan"`
	Created bool      `json:"created"`
}

type MultiAgentCycleRequest struct {
	Input                MultiAgentCycleInput `json:"input"`
	Source               string               `json:"source"`
	RecordActionRequests bool                 `json:"record_action_requests"`
}

type MultiAgentCycleResult struct {
	Cycle                 AgentCycleRecord `json:"cycle"`
	ActionRequestsCreated int              `json:"action_requests_created"`
}

func NewAgentRuntime(repo *Repository) *AgentRuntime {
	return &AgentRuntime{repo: repo}
}

func (r *AgentRuntime) RunMultiAgentCycle(ctx context.Context, request MultiAgentCycleRequest) (MultiAgentCycleResult, error) {
	if r == nil || r.repo == nil {
		return MultiAgentCycleResult{}, nil
	}
	cycle := RunRecruitingAgentCycle(request.Input)
	record, err := r.repo.RecordAgentCycle(ctx, cycle)
	if err != nil {
		return MultiAgentCycleResult{}, err
	}
	result := MultiAgentCycleResult{Cycle: record}
	if !request.RecordActionRequests {
		return result, nil
	}
	source := strings.TrimSpace(request.Source)
	if source == "" {
		source = "multi_agent"
	}
	if err := r.repo.RecordAgentActionRequests(ctx, source, cycle.Actions); err != nil {
		return MultiAgentCycleResult{}, err
	}
	result.ActionRequestsCreated = countAllowedAgentActions(cycle.Actions)
	return result, nil
}

func (r *AgentRuntime) CreateReviewPlan(ctx context.Context, request AgentReviewPlanRequest) (AgentPlanResult, error) {
	if r == nil || r.repo == nil {
		return AgentPlanResult{}, nil
	}
	input := BuildAgentPlanInputFromReview(request.Review)
	input.Source = strings.TrimSpace(request.Source)
	if input.Source == "" {
		input.Source = "agent"
	}
	if len(input.Steps) == 0 {
		return AgentPlanResult{}, nil
	}
	now := request.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	input.CreatedAt = now.UTC()
	if request.Dedupe {
		exists, err := r.repo.HasAgentPlanForDay(ctx, input.Source, input.Goal, now.UTC().Format("2006-01-02"))
		if err != nil {
			return AgentPlanResult{}, err
		}
		if exists {
			return AgentPlanResult{Created: false}, nil
		}
	}
	plan, err := r.repo.CreateAgentPlan(ctx, input)
	if err != nil {
		return AgentPlanResult{}, err
	}
	if err := r.repo.RecordAgentActionRequestsForPlan(ctx, plan.ID, plan.Source, actionsFromPlanSteps(plan.Steps)); err != nil {
		return AgentPlanResult{}, err
	}
	return AgentPlanResult{Plan: plan, Created: true}, nil
}

func (r *AgentRuntime) CreateChatPlan(ctx context.Context, goal string, reply AgentChatReply) (AgentPlanResult, error) {
	if r == nil || r.repo == nil || len(reply.Actions) == 0 {
		return AgentPlanResult{}, nil
	}
	plan, err := r.repo.CreateAgentPlan(ctx, BuildAgentPlanInputFromReply(goal, reply))
	if err != nil {
		return AgentPlanResult{}, err
	}
	if err := r.repo.RecordAgentActionRequestsForPlan(ctx, plan.ID, reply.Source, reply.Actions); err != nil {
		return AgentPlanResult{}, err
	}
	return AgentPlanResult{Plan: plan, Created: true}, nil
}

func countAllowedAgentActions(actions []AgentCommandAction) int {
	count := 0
	for _, action := range actions {
		if _, ok := allowedModelActionTypes[strings.TrimSpace(action.Type)]; ok {
			count++
		}
	}
	return count
}

func actionsFromPlanSteps(steps []AgentPlanStep) []AgentCommandAction {
	actions := make([]AgentCommandAction, 0, len(steps))
	for _, step := range steps {
		actions = append(actions, AgentCommandAction{
			Type:   step.ActionType,
			Target: step.Target,
			Detail: step.Detail,
		})
	}
	return actions
}

package jobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	AgentPlanStatusDraft           = "draft"
	AgentPlanStatusWaitingApproval = "waiting_approval"
	AgentPlanStatusExecuting       = "executing"
	AgentPlanStatusDone            = "done"
	AgentPlanStatusFailed          = "failed"

	AgentPlanStepStatusPending = "pending"
	AgentPlanStepStatusDone    = "done"
	AgentPlanStepStatusFailed  = "failed"
	AgentPlanStepStatusSkipped = "skipped"

	AgentPlanRiskLow              = "low"
	AgentPlanRiskApprovalRequired = "approval_required"
)

type AgentPlan struct {
	ID            int64           `json:"id"`
	Source        string          `json:"source"`
	Goal          string          `json:"goal"`
	Summary       string          `json:"summary"`
	Status        string          `json:"status"`
	RiskLevel     string          `json:"risk_level"`
	NeedsApproval bool            `json:"needs_approval"`
	Steps         []AgentPlanStep `json:"steps"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
	CompletedAt   *time.Time      `json:"completed_at,omitempty"`
}

type AgentPlanInput struct {
	Source        string
	Goal          string
	Summary       string
	Status        string
	RiskLevel     string
	NeedsApproval bool
	Steps         []AgentPlanStep
}

type AgentPlanStep struct {
	Order      int    `json:"order"`
	ActionType string `json:"action_type"`
	Target     string `json:"target"`
	Detail     string `json:"detail"`
	Status     string `json:"status"`
	Message    string `json:"message"`
}

func (r *Repository) CreateAgentPlan(ctx context.Context, input AgentPlanInput) (AgentPlan, error) {
	input.Source = defaultText(input.Source, "agent")
	input.Goal = strings.TrimSpace(input.Goal)
	input.Summary = strings.TrimSpace(input.Summary)
	input.Status = normalizeAgentPlanStatus(input.Status)
	if input.Status == "" {
		if input.NeedsApproval {
			input.Status = AgentPlanStatusWaitingApproval
		} else {
			input.Status = AgentPlanStatusDraft
		}
	}
	input.RiskLevel = normalizeAgentPlanRisk(input.RiskLevel)
	steps := normalizeAgentPlanSteps(input.Steps)
	stepsJSON, err := marshalAgentPlanSteps(steps)
	if err != nil {
		return AgentPlan{}, err
	}
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO agent_plans (source, goal, summary, status, risk_level, needs_approval, steps_json)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, input.Source, input.Goal, input.Summary, input.Status, input.RiskLevel, boolToInt(input.NeedsApproval), stepsJSON)
	if err != nil {
		return AgentPlan{}, fmt.Errorf("insert agent plan: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return AgentPlan{}, fmt.Errorf("read agent plan id: %w", err)
	}
	return r.GetAgentPlan(ctx, id)
}

func (r *Repository) GetAgentPlan(ctx context.Context, id int64) (AgentPlan, error) {
	row := r.db.QueryRowContext(ctx, selectAgentPlanSQL()+` WHERE id = ?`, id)
	plan, err := scanAgentPlan(row)
	if err != nil {
		return AgentPlan{}, fmt.Errorf("get agent plan %d: %w", id, err)
	}
	return plan, nil
}

func (r *Repository) ListAgentPlans(ctx context.Context, status string, limit int) ([]AgentPlan, error) {
	status = normalizeAgentPlanStatus(status)
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	query := selectAgentPlanSQL()
	args := []any{}
	if status != "" {
		query += ` WHERE status = ?`
		args = append(args, status)
	}
	query += ` ORDER BY created_at DESC, id DESC LIMIT ?`
	args = append(args, limit)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list agent plans: %w", err)
	}
	defer rows.Close()

	plans := []AgentPlan{}
	for rows.Next() {
		plan, err := scanAgentPlan(rows)
		if err != nil {
			return nil, err
		}
		plans = append(plans, plan)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate agent plans: %w", err)
	}
	return plans, nil
}

func (r *Repository) HasAgentPlanForDay(ctx context.Context, source string, goal string, day string) (bool, error) {
	source = strings.TrimSpace(source)
	goal = strings.TrimSpace(goal)
	day = strings.TrimSpace(day)
	if source == "" || goal == "" || day == "" {
		return false, nil
	}
	var count int
	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM agent_plans
		WHERE source = ? AND goal = ? AND date(created_at) = date(?)
	`, source, goal, day).Scan(&count); err != nil {
		return false, fmt.Errorf("count agent plans for day: %w", err)
	}
	return count > 0, nil
}

func (r *Repository) RecordAgentPlanStepExecution(ctx context.Context, planID int64, actionType string, status string, message string) (AgentPlan, error) {
	if planID <= 0 {
		return AgentPlan{}, nil
	}
	plan, err := r.GetAgentPlan(ctx, planID)
	if err != nil {
		return AgentPlan{}, err
	}
	actionType = strings.TrimSpace(actionType)
	status = normalizeAgentPlanStepStatus(status)
	message = strings.TrimSpace(message)
	updated := false
	for index := range plan.Steps {
		if plan.Steps[index].ActionType != actionType {
			continue
		}
		if plan.Steps[index].Status == AgentPlanStepStatusDone {
			continue
		}
		plan.Steps[index].Status = status
		plan.Steps[index].Message = message
		updated = true
		break
	}
	if !updated {
		return plan, nil
	}
	nextStatus := deriveAgentPlanStatus(plan.Steps)
	var completedAt any
	if nextStatus == AgentPlanStatusDone {
		completedAt = time.Now().UTC()
	}
	stepsJSON, err := marshalAgentPlanSteps(plan.Steps)
	if err != nil {
		return AgentPlan{}, err
	}
	result, err := r.db.ExecContext(ctx, `
		UPDATE agent_plans
		SET steps_json = ?, status = ?, completed_at = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, stepsJSON, nextStatus, completedAt, plan.ID)
	if err != nil {
		return AgentPlan{}, fmt.Errorf("update agent plan step: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return AgentPlan{}, fmt.Errorf("read rows affected: %w", err)
	}
	if affected == 0 {
		return AgentPlan{}, sql.ErrNoRows
	}
	return r.GetAgentPlan(ctx, plan.ID)
}

func BuildAgentPlanInputFromReply(goal string, reply AgentChatReply) AgentPlanInput {
	steps := make([]AgentPlanStep, 0, len(reply.Actions))
	for index, action := range reply.Actions {
		if _, ok := allowedModelActionTypes[strings.TrimSpace(action.Type)]; !ok {
			continue
		}
		steps = append(steps, AgentPlanStep{
			Order:      index + 1,
			ActionType: strings.TrimSpace(action.Type),
			Target:     strings.TrimSpace(action.Target),
			Detail:     strings.TrimSpace(action.Detail),
			Status:     AgentPlanStepStatusPending,
		})
	}
	return AgentPlanInput{
		Source:        reply.Source,
		Goal:          goal,
		Summary:       reply.Content,
		RiskLevel:     AgentPlanRiskApprovalRequired,
		NeedsApproval: len(steps) > 0,
		Steps:         steps,
	}
}

func BuildAgentPlanInputFromReview(review AgentReview) AgentPlanInput {
	steps := []AgentPlanStep{}
	for _, nextStep := range review.NextSteps {
		allowed, ok := allowedModelActionTypes[strings.TrimSpace(nextStep.Action)]
		if !ok {
			continue
		}
		detail := strings.TrimSpace(nextStep.Reason)
		if detail == "" {
			detail = allowed.Detail
		}
		steps = append(steps, AgentPlanStep{
			Order:      len(steps) + 1,
			ActionType: allowed.Type,
			Target:     allowed.Target,
			Detail:     detail,
			Status:     AgentPlanStepStatusPending,
		})
	}
	return AgentPlanInput{
		Source:        "review",
		Goal:          "今日秋招工作计划",
		Summary:       review.Focus.Title + ": " + review.Focus.Detail,
		RiskLevel:     AgentPlanRiskApprovalRequired,
		NeedsApproval: len(steps) > 0,
		Steps:         steps,
	}
}

func normalizeAgentPlanStatus(status string) string {
	switch strings.TrimSpace(status) {
	case AgentPlanStatusDraft:
		return AgentPlanStatusDraft
	case AgentPlanStatusWaitingApproval:
		return AgentPlanStatusWaitingApproval
	case AgentPlanStatusExecuting:
		return AgentPlanStatusExecuting
	case AgentPlanStatusDone:
		return AgentPlanStatusDone
	case AgentPlanStatusFailed:
		return AgentPlanStatusFailed
	default:
		return ""
	}
}

func normalizeAgentPlanRisk(risk string) string {
	switch strings.TrimSpace(risk) {
	case AgentPlanRiskApprovalRequired:
		return AgentPlanRiskApprovalRequired
	default:
		return AgentPlanRiskLow
	}
}

func normalizeAgentPlanSteps(steps []AgentPlanStep) []AgentPlanStep {
	out := make([]AgentPlanStep, 0, len(steps))
	for index, step := range steps {
		step.ActionType = strings.TrimSpace(step.ActionType)
		step.Target = strings.TrimSpace(step.Target)
		step.Detail = strings.TrimSpace(step.Detail)
		step.Message = strings.TrimSpace(step.Message)
		step.Status = normalizeAgentPlanStepStatus(step.Status)
		if step.Status == "" {
			step.Status = AgentPlanStepStatusPending
		}
		if step.Order <= 0 {
			step.Order = index + 1
		}
		out = append(out, step)
	}
	return out
}

func normalizeAgentPlanStepStatus(status string) string {
	switch strings.TrimSpace(status) {
	case AgentPlanStepStatusPending:
		return AgentPlanStepStatusPending
	case AgentPlanStepStatusDone:
		return AgentPlanStepStatusDone
	case AgentPlanStepStatusFailed:
		return AgentPlanStepStatusFailed
	case AgentPlanStepStatusSkipped:
		return AgentPlanStepStatusSkipped
	default:
		return ""
	}
}

func deriveAgentPlanStatus(steps []AgentPlanStep) string {
	if len(steps) == 0 {
		return AgentPlanStatusDraft
	}
	allDone := true
	for _, step := range steps {
		switch step.Status {
		case AgentPlanStepStatusFailed:
			return AgentPlanStatusFailed
		case AgentPlanStepStatusDone, AgentPlanStepStatusSkipped:
		default:
			allDone = false
		}
	}
	if allDone {
		return AgentPlanStatusDone
	}
	return AgentPlanStatusWaitingApproval
}

func selectAgentPlanSQL() string {
	return `SELECT id, source, goal, summary, status, risk_level, needs_approval, steps_json, created_at, updated_at, completed_at FROM agent_plans`
}

func scanAgentPlan(scanner jobScanner) (AgentPlan, error) {
	var plan AgentPlan
	var needsApproval int
	var stepsJSON string
	if err := scanner.Scan(
		&plan.ID,
		&plan.Source,
		&plan.Goal,
		&plan.Summary,
		&plan.Status,
		&plan.RiskLevel,
		&needsApproval,
		&stepsJSON,
		&plan.CreatedAt,
		&plan.UpdatedAt,
		&plan.CompletedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return AgentPlan{}, err
		}
		return AgentPlan{}, fmt.Errorf("scan agent plan: %w", err)
	}
	steps, err := unmarshalAgentPlanSteps(stepsJSON)
	if err != nil {
		return AgentPlan{}, err
	}
	plan.NeedsApproval = needsApproval != 0
	plan.Status = normalizeAgentPlanStatus(plan.Status)
	if plan.Status == "" {
		plan.Status = AgentPlanStatusDraft
	}
	plan.RiskLevel = normalizeAgentPlanRisk(plan.RiskLevel)
	plan.Steps = steps
	return plan, nil
}

func marshalAgentPlanSteps(steps []AgentPlanStep) (string, error) {
	payload, err := json.Marshal(steps)
	if err != nil {
		return "", fmt.Errorf("marshal agent plan steps: %w", err)
	}
	return string(payload), nil
}

func unmarshalAgentPlanSteps(raw string) ([]AgentPlanStep, error) {
	if strings.TrimSpace(raw) == "" {
		return []AgentPlanStep{}, nil
	}
	var steps []AgentPlanStep
	if err := json.Unmarshal([]byte(raw), &steps); err != nil {
		return nil, fmt.Errorf("unmarshal agent plan steps: %w", err)
	}
	return normalizeAgentPlanSteps(steps), nil
}

func defaultText(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type AgentCycleRecord struct {
	ID                   int64                `json:"id"`
	GeneratedAt          time.Time            `json:"generated_at"`
	Summary              string               `json:"summary"`
	ReadinessScore       int                  `json:"readiness_score"`
	OrchestratorProvider string               `json:"orchestrator_provider"`
	OrchestratorPattern  string               `json:"orchestrator_pattern"`
	Trace                []MultiAgentTrace    `json:"trace"`
	Actions              []AgentCommandAction `json:"actions"`
	AutonomyPlan         AgentAutonomyPlan    `json:"autonomy_plan"`
	CreatedAt            time.Time            `json:"created_at"`
}

func (r *Repository) RecordAgentCycle(ctx context.Context, cycle MultiAgentCycle) (AgentCycleRecord, error) {
	if cycle.GeneratedAt.IsZero() {
		cycle.GeneratedAt = time.Now().UTC()
	}
	traceJSON, err := json.Marshal(cycle.Trace)
	if err != nil {
		return AgentCycleRecord{}, fmt.Errorf("marshal agent cycle trace: %w", err)
	}
	actionsJSON, err := json.Marshal(cycle.Actions)
	if err != nil {
		return AgentCycleRecord{}, fmt.Errorf("marshal agent cycle actions: %w", err)
	}
	autonomyPlanJSON, err := json.Marshal(cycle.AutonomyPlan)
	if err != nil {
		return AgentCycleRecord{}, fmt.Errorf("marshal agent cycle autonomy plan: %w", err)
	}
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO agent_cycles (
			generated_at, summary, readiness_score, orchestrator_provider,
			orchestrator_pattern, trace_json, actions_json, autonomy_plan_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, cycle.GeneratedAt, cycle.Summary, cycle.ReadinessScore, cycle.Team.Orchestrator.Provider,
		cycle.Team.Orchestrator.Pattern, string(traceJSON), string(actionsJSON), string(autonomyPlanJSON))
	if err != nil {
		return AgentCycleRecord{}, fmt.Errorf("record agent cycle: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return AgentCycleRecord{}, fmt.Errorf("read agent cycle id: %w", err)
	}
	return r.GetAgentCycle(ctx, id)
}

func (r *Repository) GetAgentCycle(ctx context.Context, id int64) (AgentCycleRecord, error) {
	row := r.db.QueryRowContext(ctx, selectAgentCycleSQL()+` WHERE id = ?`, id)
	record, err := scanAgentCycle(row)
	if err != nil {
		return AgentCycleRecord{}, fmt.Errorf("get agent cycle %d: %w", id, err)
	}
	return record, nil
}

func (r *Repository) ListAgentCycles(ctx context.Context, limit int) ([]AgentCycleRecord, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := r.db.QueryContext(ctx, selectAgentCycleSQL()+`
		ORDER BY generated_at DESC, id DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("list agent cycles: %w", err)
	}
	defer rows.Close()

	records := []AgentCycleRecord{}
	for rows.Next() {
		record, err := scanAgentCycle(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate agent cycles: %w", err)
	}
	return records, nil
}

func selectAgentCycleSQL() string {
	return `
		SELECT id, generated_at, summary, readiness_score, orchestrator_provider,
			orchestrator_pattern, trace_json, actions_json, autonomy_plan_json, created_at
		FROM agent_cycles`
}

func scanAgentCycle(scanner jobScanner) (AgentCycleRecord, error) {
	var record AgentCycleRecord
	var traceJSON string
	var actionsJSON string
	var autonomyPlanJSON string
	if err := scanner.Scan(
		&record.ID,
		&record.GeneratedAt,
		&record.Summary,
		&record.ReadinessScore,
		&record.OrchestratorProvider,
		&record.OrchestratorPattern,
		&traceJSON,
		&actionsJSON,
		&autonomyPlanJSON,
		&record.CreatedAt,
	); err != nil {
		return AgentCycleRecord{}, fmt.Errorf("scan agent cycle: %w", err)
	}
	if err := json.Unmarshal([]byte(traceJSON), &record.Trace); err != nil {
		return AgentCycleRecord{}, fmt.Errorf("decode agent cycle trace: %w", err)
	}
	if err := json.Unmarshal([]byte(actionsJSON), &record.Actions); err != nil {
		return AgentCycleRecord{}, fmt.Errorf("decode agent cycle actions: %w", err)
	}
	if strings.TrimSpace(autonomyPlanJSON) != "" {
		if err := json.Unmarshal([]byte(autonomyPlanJSON), &record.AutonomyPlan); err != nil {
			return AgentCycleRecord{}, fmt.Errorf("decode agent cycle autonomy plan: %w", err)
		}
	}
	return record, nil
}

package jobs

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type JobDecision struct {
	ID         int64     `json:"id"`
	JobID      int64     `json:"job_id"`
	Action     string    `json:"action"`
	Reason     string    `json:"reason"`
	FromStatus string    `json:"from_status"`
	ToStatus   string    `json:"to_status"`
	Notes      string    `json:"notes"`
	CreatedAt  time.Time `json:"created_at"`
}

type JobDecisionInput struct {
	JobID      int64
	Action     string
	Reason     string
	FromStatus string
	ToStatus   string
	Notes      string
}

func (r *Repository) RecordJobDecision(ctx context.Context, input JobDecisionInput) (JobDecision, error) {
	input.Action = strings.TrimSpace(input.Action)
	input.Reason = strings.TrimSpace(input.Reason)
	input.FromStatus = strings.TrimSpace(input.FromStatus)
	input.ToStatus = strings.TrimSpace(input.ToStatus)
	input.Notes = strings.TrimSpace(input.Notes)
	if input.Action == "" {
		input.Action = "decision_recorded"
	}
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO job_decisions (job_id, action, reason, from_status, to_status, notes)
		VALUES (?, ?, ?, ?, ?, ?)
	`, input.JobID, input.Action, input.Reason, input.FromStatus, input.ToStatus, input.Notes)
	if err != nil {
		return JobDecision{}, fmt.Errorf("record job decision: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return JobDecision{}, fmt.Errorf("read job decision id: %w", err)
	}
	return r.GetJobDecision(ctx, id)
}

func (r *Repository) GetJobDecision(ctx context.Context, id int64) (JobDecision, error) {
	row := r.db.QueryRowContext(ctx, selectJobDecisionSQL()+` WHERE id = ?`, id)
	decision, err := scanJobDecision(row)
	if err != nil {
		return JobDecision{}, fmt.Errorf("get job decision %d: %w", id, err)
	}
	return decision, nil
}

func (r *Repository) ListJobDecisions(ctx context.Context, jobID int64) ([]JobDecision, error) {
	rows, err := r.db.QueryContext(ctx, selectJobDecisionSQL()+`
		WHERE job_id = ?
		ORDER BY created_at DESC, id DESC
	`, jobID)
	if err != nil {
		return nil, fmt.Errorf("list job decisions: %w", err)
	}
	defer rows.Close()
	out := []JobDecision{}
	for rows.Next() {
		decision, err := scanJobDecision(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, decision)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate job decisions: %w", err)
	}
	return out, nil
}

func selectJobDecisionSQL() string {
	return `
		SELECT id, job_id, action, reason, from_status, to_status, notes, created_at
		FROM job_decisions`
}

func scanJobDecision(scanner jobScanner) (JobDecision, error) {
	var decision JobDecision
	if err := scanner.Scan(
		&decision.ID,
		&decision.JobID,
		&decision.Action,
		&decision.Reason,
		&decision.FromStatus,
		&decision.ToStatus,
		&decision.Notes,
		&decision.CreatedAt,
	); err != nil {
		return JobDecision{}, fmt.Errorf("scan job decision: %w", err)
	}
	return decision, nil
}

package jobs

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/domain"
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
	if id <= 0 {
		if err := r.db.QueryRowContext(ctx, `SELECT id FROM job_decisions WHERE job_id = ? ORDER BY id DESC LIMIT 1`, input.JobID).Scan(&id); err != nil {
			return JobDecision{}, fmt.Errorf("read latest job decision id: %w", err)
		}
	}
	decision, err := r.GetJobDecision(ctx, id)
	if err != nil {
		return JobDecision{}, err
	}
	if job, err := r.GetJob(ctx, input.JobID); err == nil {
		_, _ = r.UpsertSemanticMemoryItem(ctx, SemanticMemoryItemFromDecision(decision, job))
	}
	return decision, nil
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

func (r *Repository) BuildJobPreferenceFeedback(ctx context.Context, limit int) (JobPreferenceFeedback, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT j.company, j.direction_tags, d.to_status, d.action
		FROM job_decisions d
		JOIN jobs j ON j.id = d.job_id
		WHERE d.to_status IN (?, ?, ?)
		ORDER BY d.created_at DESC, d.id DESC
		LIMIT ?
	`, string(domain.StatusInterested), string(domain.StatusApplied), string(domain.StatusIgnored), limit)
	if err != nil {
		return JobPreferenceFeedback{}, fmt.Errorf("build job preference feedback: %w", err)
	}
	defer rows.Close()
	feedback := JobPreferenceFeedback{}
	for rows.Next() {
		var company string
		var tagsJSON string
		var toStatus string
		var action string
		if err := rows.Scan(&company, &tagsJSON, &toStatus, &action); err != nil {
			return JobPreferenceFeedback{}, fmt.Errorf("scan job preference feedback: %w", err)
		}
		tags := unmarshalStrings(tagsJSON)
		switch toStatus {
		case string(domain.StatusInterested), string(domain.StatusApplied):
			feedback.InterestedCompanies = mergeStrings(feedback.InterestedCompanies, []string{company})
			feedback.InterestedDirections = mergeStrings(feedback.InterestedDirections, tags)
		case string(domain.StatusIgnored):
			feedback.IgnoredCompanies = mergeStrings(feedback.IgnoredCompanies, []string{company})
			feedback.IgnoredDirections = mergeStrings(feedback.IgnoredDirections, tags)
		}
	}
	if err := rows.Err(); err != nil {
		return JobPreferenceFeedback{}, fmt.Errorf("iterate job preference feedback: %w", err)
	}
	return feedback, nil
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

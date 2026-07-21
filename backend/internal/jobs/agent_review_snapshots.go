package jobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type AgentReviewSnapshot struct {
	ID          int64            `json:"id"`
	TriggerType string           `json:"trigger_type"`
	CapturedAt  time.Time        `json:"captured_at"`
	HealthScore int              `json:"health_score"`
	HealthLabel string           `json:"health_label"`
	FocusTitle  string           `json:"focus_title"`
	FocusAction string           `json:"focus_action"`
	Stats       AgentReviewStats `json:"stats"`
	Review      AgentReview      `json:"review"`
	CreatedAt   time.Time        `json:"created_at"`
}

type AgentReviewHistory struct {
	GeneratedAt time.Time             `json:"generated_at"`
	Snapshots   []AgentReviewSnapshot `json:"snapshots"`
	Delta       AgentReviewStats      `json:"delta"`
	Summary     string                `json:"summary"`
}

func BuildAgentReviewHistory(snapshots []AgentReviewSnapshot) AgentReviewHistory {
	history := AgentReviewHistory{
		GeneratedAt: time.Now().UTC(),
		Snapshots:   snapshots,
		Summary:     "Not enough snapshots to compare yet.",
	}
	if len(snapshots) < 2 {
		return history
	}
	latest := snapshots[0].Stats
	previous := snapshots[1].Stats
	history.Delta = AgentReviewStats{
		TrackedJobs:     latest.TrackedJobs - previous.TrackedJobs,
		NewJobs:         latest.NewJobs - previous.NewJobs,
		StrongMatches:   latest.StrongMatches - previous.StrongMatches,
		ManualDecisions: latest.ManualDecisions - previous.ManualDecisions,
		SourceIssues:    latest.SourceIssues - previous.SourceIssues,
		OpenTasks:       latest.OpenTasks - previous.OpenTasks,
		AppliedJobs:     latest.AppliedJobs - previous.AppliedJobs,
	}
	history.Summary = reviewHistorySummary(history.Delta)
	return history
}

func (r *Repository) CreateAgentReviewSnapshot(ctx context.Context, review AgentReview, triggerType string) (AgentReviewSnapshot, error) {
	triggerType = strings.TrimSpace(triggerType)
	if triggerType == "" {
		triggerType = "manual"
	}
	if review.GeneratedAt.IsZero() {
		review.GeneratedAt = time.Now().UTC()
	}
	raw, err := json.Marshal(review)
	if err != nil {
		return AgentReviewSnapshot{}, fmt.Errorf("marshal agent review: %w", err)
	}
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO agent_review_snapshots (
			trigger_type, captured_at, health_score, health_label, focus_title, focus_action,
			tracked_jobs, new_jobs, strong_matches, manual_decisions, source_issues,
			open_tasks, applied_jobs, review_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, triggerType, review.GeneratedAt, review.Health.Score, review.Health.Label, review.Focus.Title, review.Focus.Action,
		review.Stats.TrackedJobs, review.Stats.NewJobs, review.Stats.StrongMatches, review.Stats.ManualDecisions,
		review.Stats.SourceIssues, review.Stats.OpenTasks, review.Stats.AppliedJobs, string(raw))
	if err != nil {
		return AgentReviewSnapshot{}, fmt.Errorf("create agent review snapshot: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return AgentReviewSnapshot{}, fmt.Errorf("read agent review snapshot id: %w", err)
	}
	return r.GetAgentReviewSnapshot(ctx, id)
}

func reviewHistorySummary(delta AgentReviewStats) string {
	return "Compared with the previous snapshot: strong matches " + signedInt(delta.StrongMatches) +
		", source issues " + signedInt(delta.SourceIssues) +
		", open tasks " + signedInt(delta.OpenTasks) + "."
}

func signedInt(value int) string {
	if value > 0 {
		return "+" + itoa(value)
	}
	return itoa(value)
}

func (r *Repository) GetAgentReviewSnapshot(ctx context.Context, id int64) (AgentReviewSnapshot, error) {
	row := r.db.QueryRowContext(ctx, selectAgentReviewSnapshotSQL()+` WHERE id = ?`, id)
	snapshot, err := scanAgentReviewSnapshot(row)
	if err != nil {
		return AgentReviewSnapshot{}, fmt.Errorf("get agent review snapshot %d: %w", id, err)
	}
	return snapshot, nil
}

func (r *Repository) ListAgentReviewSnapshots(ctx context.Context, limit int) ([]AgentReviewSnapshot, error) {
	if limit <= 0 || limit > 100 {
		limit = 14
	}
	rows, err := r.db.QueryContext(ctx, selectAgentReviewSnapshotSQL()+`
		ORDER BY captured_at DESC, id DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("list agent review snapshots: %w", err)
	}
	defer rows.Close()

	out := []AgentReviewSnapshot{}
	for rows.Next() {
		snapshot, err := scanAgentReviewSnapshot(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, snapshot)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate agent review snapshots: %w", err)
	}
	return out, nil
}

func selectAgentReviewSnapshotSQL() string {
	return `
		SELECT id, trigger_type, captured_at, health_score, health_label, focus_title,
			focus_action, tracked_jobs, new_jobs, strong_matches, manual_decisions,
			source_issues, open_tasks, applied_jobs, review_json, created_at
		FROM agent_review_snapshots`
}

func scanAgentReviewSnapshot(scanner interface {
	Scan(dest ...any) error
}) (AgentReviewSnapshot, error) {
	var snapshot AgentReviewSnapshot
	var rawReview string
	if err := scanner.Scan(
		&snapshot.ID,
		&snapshot.TriggerType,
		&snapshot.CapturedAt,
		&snapshot.HealthScore,
		&snapshot.HealthLabel,
		&snapshot.FocusTitle,
		&snapshot.FocusAction,
		&snapshot.Stats.TrackedJobs,
		&snapshot.Stats.NewJobs,
		&snapshot.Stats.StrongMatches,
		&snapshot.Stats.ManualDecisions,
		&snapshot.Stats.SourceIssues,
		&snapshot.Stats.OpenTasks,
		&snapshot.Stats.AppliedJobs,
		&rawReview,
		&snapshot.CreatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return AgentReviewSnapshot{}, err
		}
		return AgentReviewSnapshot{}, fmt.Errorf("scan agent review snapshot: %w", err)
	}
	if err := json.Unmarshal([]byte(rawReview), &snapshot.Review); err != nil {
		return AgentReviewSnapshot{}, fmt.Errorf("decode agent review snapshot: %w", err)
	}
	return snapshot, nil
}

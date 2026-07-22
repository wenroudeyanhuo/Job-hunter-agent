package jobs

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/domain"
)

const (
	ApplicationPlanStatusPrepare = "prepare"
	ApplicationPlanStatusReady   = "ready"
	ApplicationPlanStatusApplied = "applied"
	ApplicationPlanStatusPaused  = "paused"
)

type ApplicationPlan struct {
	ID              int64     `json:"id"`
	JobID           int64     `json:"job_id"`
	Status          string    `json:"status"`
	Priority        int       `json:"priority"`
	NextAction      string    `json:"next_action"`
	Checklist       []string  `json:"checklist"`
	BlockerNotes    string    `json:"blocker_notes"`
	ResumeVersion   string    `json:"resume_version"`
	DraftNotes      string    `json:"draft_notes"`
	TargetApplyDate string    `json:"target_apply_date"`
	FollowUpDate    string    `json:"follow_up_date"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type ApplicationPlanUpdate struct {
	Status          string   `json:"status"`
	NextAction      string   `json:"next_action"`
	Checklist       []string `json:"checklist"`
	BlockerNotes    string   `json:"blocker_notes"`
	ResumeVersion   string   `json:"resume_version"`
	DraftNotes      string   `json:"draft_notes"`
	TargetApplyDate string   `json:"target_apply_date"`
	FollowUpDate    string   `json:"follow_up_date"`
}

func (r *Repository) SyncApplicationPlans(ctx context.Context, now time.Time) ([]ApplicationPlan, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	jobs, err := r.ListJobs(ctx, ListFilter{})
	if err != nil {
		return nil, err
	}
	for _, job := range jobs {
		if !shouldPrepareApplicationPlan(job) {
			continue
		}
		if err := r.upsertApplicationPlan(ctx, buildApplicationPlanInput(job, now)); err != nil {
			return nil, err
		}
	}
	return r.ListApplicationPlans(ctx, "")
}

func (r *Repository) ListApplicationPlans(ctx context.Context, status string) ([]ApplicationPlan, error) {
	query := selectApplicationPlanSQL()
	args := []any{}
	if strings.TrimSpace(status) != "" {
		query += " WHERE status = ?"
		args = append(args, normalizeApplicationPlanStatus(status))
	}
	query += " ORDER BY priority DESC, updated_at DESC, id DESC"
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list application plans: %w", err)
	}
	defer rows.Close()
	out := []ApplicationPlan{}
	for rows.Next() {
		plan, err := scanApplicationPlan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, plan)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate application plans: %w", err)
	}
	return out, nil
}

func (r *Repository) GetApplicationPlanByJobID(ctx context.Context, jobID int64) (ApplicationPlan, bool, error) {
	row := r.db.QueryRowContext(ctx, selectApplicationPlanSQL()+` WHERE job_id = ?`, jobID)
	plan, err := scanApplicationPlan(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return ApplicationPlan{}, false, nil
		}
		return ApplicationPlan{}, false, fmt.Errorf("get application plan by job %d: %w", jobID, err)
	}
	return plan, true, nil
}

func (r *Repository) UpdateApplicationPlan(ctx context.Context, id int64, input ApplicationPlanUpdate) (ApplicationPlan, error) {
	status := normalizeApplicationPlanStatus(input.Status)
	checklist, err := marshalStrings(cleanStringList(input.Checklist))
	if err != nil {
		return ApplicationPlan{}, err
	}
	_, err = r.db.ExecContext(ctx, `
		UPDATE application_plans
		SET status = ?, next_action = ?, checklist = ?, blocker_notes = ?, resume_version = ?,
			draft_notes = ?, target_apply_date = ?, follow_up_date = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, status, strings.TrimSpace(input.NextAction), checklist, strings.TrimSpace(input.BlockerNotes),
		strings.TrimSpace(input.ResumeVersion), strings.TrimSpace(input.DraftNotes),
		strings.TrimSpace(input.TargetApplyDate), strings.TrimSpace(input.FollowUpDate), id)
	if err != nil {
		return ApplicationPlan{}, fmt.Errorf("update application plan: %w", err)
	}
	row := r.db.QueryRowContext(ctx, selectApplicationPlanSQL()+` WHERE id = ?`, id)
	return scanApplicationPlan(row)
}

type applicationPlanInput struct {
	JobID           int64
	Status          string
	Priority        int
	NextAction      string
	Checklist       []string
	ResumeVersion   string
	DraftNotes      string
	TargetApplyDate string
	FollowUpDate    string
}

func (r *Repository) upsertApplicationPlan(ctx context.Context, input applicationPlanInput) error {
	checklist, err := marshalStrings(cleanStringList(input.Checklist))
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO application_plans (
			job_id, status, priority, next_action, checklist, resume_version, draft_notes,
			target_apply_date, follow_up_date
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(job_id) DO UPDATE SET
			priority = excluded.priority,
			next_action = CASE WHEN application_plans.status IN (?, ?) THEN application_plans.next_action ELSE excluded.next_action END,
			checklist = CASE WHEN application_plans.status IN (?, ?) THEN application_plans.checklist ELSE excluded.checklist END,
			resume_version = CASE WHEN application_plans.resume_version != '' THEN application_plans.resume_version ELSE excluded.resume_version END,
			draft_notes = CASE WHEN application_plans.draft_notes != '' THEN application_plans.draft_notes ELSE excluded.draft_notes END,
			follow_up_date = CASE WHEN application_plans.follow_up_date != '' THEN application_plans.follow_up_date ELSE excluded.follow_up_date END,
			updated_at = CURRENT_TIMESTAMP
	`, input.JobID, normalizeApplicationPlanStatus(input.Status), input.Priority, strings.TrimSpace(input.NextAction), checklist,
		strings.TrimSpace(input.ResumeVersion), strings.TrimSpace(input.DraftNotes), input.TargetApplyDate, input.FollowUpDate,
		ApplicationPlanStatusReady, ApplicationPlanStatusApplied, ApplicationPlanStatusReady, ApplicationPlanStatusApplied)
	if err != nil {
		return fmt.Errorf("upsert application plan: %w", err)
	}
	return nil
}

func shouldPrepareApplicationPlan(job domain.Job) bool {
	return job.ID > 0 && job.Status == domain.StatusInterested && job.MatchScore >= 70
}

func buildApplicationPlanInput(job domain.Job, now time.Time) applicationPlanInput {
	checklist := []string{"Review job detail", "Confirm resume version", "Prepare application notes"}
	if strings.TrimSpace(job.ApplyURL) != "" {
		checklist = append(checklist, "Open application link")
	}
	return applicationPlanInput{
		JobID:           job.ID,
		Status:          ApplicationPlanStatusPrepare,
		Priority:        100 + job.MatchScore,
		NextAction:      "Prepare resume and application notes for " + fallbackText(job.Company, "this company"),
		Checklist:       checklist,
		ResumeVersion:   "default",
		DraftNotes:      "Draft a short application note based on the job detail and profile signals.",
		TargetApplyDate: now.Add(24 * time.Hour).Format("2006-01-02"),
		FollowUpDate:    now.Add(7 * 24 * time.Hour).Format("2006-01-02"),
	}
}

func normalizeApplicationPlanStatus(status string) string {
	switch strings.TrimSpace(status) {
	case ApplicationPlanStatusReady:
		return ApplicationPlanStatusReady
	case ApplicationPlanStatusApplied:
		return ApplicationPlanStatusApplied
	case ApplicationPlanStatusPaused:
		return ApplicationPlanStatusPaused
	default:
		return ApplicationPlanStatusPrepare
	}
}

func selectApplicationPlanSQL() string {
	return `
		SELECT id, job_id, status, priority, next_action, checklist, blocker_notes,
			resume_version, draft_notes, target_apply_date, follow_up_date, created_at, updated_at
		FROM application_plans`
}

func scanApplicationPlan(scanner jobScanner) (ApplicationPlan, error) {
	var plan ApplicationPlan
	var checklist string
	if err := scanner.Scan(
		&plan.ID,
		&plan.JobID,
		&plan.Status,
		&plan.Priority,
		&plan.NextAction,
		&checklist,
		&plan.BlockerNotes,
		&plan.ResumeVersion,
		&plan.DraftNotes,
		&plan.TargetApplyDate,
		&plan.FollowUpDate,
		&plan.CreatedAt,
		&plan.UpdatedAt,
	); err != nil {
		return ApplicationPlan{}, err
	}
	plan.Status = normalizeApplicationPlanStatus(plan.Status)
	plan.Checklist = unmarshalStrings(checklist)
	return plan, nil
}

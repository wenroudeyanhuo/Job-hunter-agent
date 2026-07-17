package jobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/domain"
)

type Repository struct {
	db *sql.DB
}

type ListFilter struct {
	Status domain.JobStatus
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateJob(ctx context.Context, job domain.Job) (domain.Job, error) {
	if job.DiscoveredAt.IsZero() {
		job.DiscoveredAt = time.Now().UTC()
	}
	if job.Status == "" {
		job.Status = domain.StatusNew
	}
	directionTags, err := marshalStrings(job.DirectionTags)
	if err != nil {
		return domain.Job{}, err
	}
	recommendReasons, err := marshalStrings(job.RecommendReasons)
	if err != nil {
		return domain.Job{}, err
	}
	penaltyReasons, err := marshalStrings(job.PenaltyReasons)
	if err != nil {
		return domain.Job{}, err
	}

	result, err := r.db.ExecContext(ctx, `
		INSERT INTO jobs (
			company, title, city, direction_tags, description, source_name, source_url, apply_url,
			published_at, deadline_at, discovered_at, match_score, recommend_reasons,
			penalty_reasons, status, notes
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, job.Company, job.Title, job.City, directionTags, job.Description, job.SourceName,
		job.SourceURL, job.ApplyURL, job.PublishedAt, job.DeadlineAt, job.DiscoveredAt,
		job.MatchScore, recommendReasons, penaltyReasons, string(job.Status), job.Notes)
	if err != nil {
		return domain.Job{}, fmt.Errorf("insert job: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return domain.Job{}, fmt.Errorf("read inserted job id: %w", err)
	}
	return r.GetJob(ctx, id)
}

func (r *Repository) GetJob(ctx context.Context, id int64) (domain.Job, error) {
	row := r.db.QueryRowContext(ctx, selectJobSQL()+` WHERE id = ?`, id)
	job, err := scanJob(row)
	if err != nil {
		return domain.Job{}, fmt.Errorf("get job %d: %w", id, err)
	}
	return job, nil
}

func (r *Repository) ListJobs(ctx context.Context, filter ListFilter) ([]domain.Job, error) {
	query := selectJobSQL()
	args := []any{}
	if filter.Status != "" {
		query += " WHERE status = ?"
		args = append(args, string(filter.Status))
	}
	query += " ORDER BY match_score DESC, discovered_at DESC, id DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list jobs: %w", err)
	}
	defer rows.Close()

	var out []domain.Job
	for rows.Next() {
		job, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, job)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate jobs: %w", err)
	}
	return out, nil
}

func (r *Repository) UpdateStatus(ctx context.Context, id int64, status domain.JobStatus) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE jobs
		SET status = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, string(status), id)
	if err != nil {
		return fmt.Errorf("update job status: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read rows affected: %w", err)
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

type jobScanner interface {
	Scan(dest ...any) error
}

func selectJobSQL() string {
	return `
		SELECT id, company, title, city, direction_tags, description, source_name, source_url,
			apply_url, published_at, deadline_at, discovered_at, match_score, recommend_reasons,
			penalty_reasons, status, notes, created_at, updated_at
		FROM jobs`
}

func scanJob(scanner jobScanner) (domain.Job, error) {
	var job domain.Job
	var directionTags string
	var recommendReasons string
	var penaltyReasons string
	var status string
	if err := scanner.Scan(
		&job.ID,
		&job.Company,
		&job.Title,
		&job.City,
		&directionTags,
		&job.Description,
		&job.SourceName,
		&job.SourceURL,
		&job.ApplyURL,
		&job.PublishedAt,
		&job.DeadlineAt,
		&job.DiscoveredAt,
		&job.MatchScore,
		&recommendReasons,
		&penaltyReasons,
		&status,
		&job.Notes,
		&job.CreatedAt,
		&job.UpdatedAt,
	); err != nil {
		return domain.Job{}, fmt.Errorf("scan job: %w", err)
	}
	job.Status = domain.JobStatus(status)
	job.DirectionTags = unmarshalStrings(directionTags)
	job.RecommendReasons = unmarshalStrings(recommendReasons)
	job.PenaltyReasons = unmarshalStrings(penaltyReasons)
	return job, nil
}

func marshalStrings(values []string) (string, error) {
	if values == nil {
		values = []string{}
	}
	data, err := json.Marshal(values)
	if err != nil {
		return "", fmt.Errorf("marshal string list: %w", err)
	}
	return string(data), nil
}

func unmarshalStrings(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []string{}
	}
	var values []string
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return []string{}
	}
	return values
}

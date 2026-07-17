package domain

import "time"

type JobStatus string

const (
	StatusNew         JobStatus = "new"
	StatusInterested  JobStatus = "interested"
	StatusApplied     JobStatus = "applied"
	StatusIgnored     JobStatus = "ignored"
	StatusManualCheck JobStatus = "manual_check"
	StatusExpired     JobStatus = "expired"
)

type Job struct {
	ID               int64      `json:"id"`
	Company          string     `json:"company"`
	Title            string     `json:"title"`
	City             string     `json:"city"`
	DirectionTags    []string   `json:"direction_tags"`
	Description      string     `json:"description"`
	SourceName       string     `json:"source_name"`
	SourceURL        string     `json:"source_url"`
	ApplyURL         string     `json:"apply_url"`
	PublishedAt      *time.Time `json:"published_at,omitempty"`
	DeadlineAt       *time.Time `json:"deadline_at,omitempty"`
	DiscoveredAt     time.Time  `json:"discovered_at"`
	MatchScore       int        `json:"match_score"`
	RecommendReasons []string   `json:"recommend_reasons"`
	PenaltyReasons   []string   `json:"penalty_reasons"`
	Status           JobStatus  `json:"status"`
	Notes            string     `json:"notes"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type JobRun struct {
	ID               int64      `json:"id"`
	TriggerType      string     `json:"trigger_type"`
	StartedAt        time.Time  `json:"started_at"`
	FinishedAt       *time.Time `json:"finished_at,omitempty"`
	Status           string     `json:"status"`
	SourcesTotal     int        `json:"sources_total"`
	SourcesSuccess   int        `json:"sources_success"`
	SourcesFailed    int        `json:"sources_failed"`
	JobsFound        int        `json:"jobs_found"`
	JobsCreated      int        `json:"jobs_created"`
	JobsDuplicated   int        `json:"jobs_duplicated"`
	ManualCheckCount int        `json:"manual_check_count"`
	ErrorSummary     string     `json:"error_summary"`
}

type JobRunSource struct {
	ID               int64     `json:"id"`
	JobRunID         int64     `json:"job_run_id"`
	SourceName       string    `json:"source_name"`
	SourceURL        string    `json:"source_url"`
	Status           string    `json:"status"`
	JobsFound        int       `json:"jobs_found"`
	JobsCreated      int       `json:"jobs_created"`
	JobsDuplicated   int       `json:"jobs_duplicated"`
	JobsFiltered     int       `json:"jobs_filtered"`
	ManualCheckCount int       `json:"manual_check_count"`
	ErrorMessage     string    `json:"error_message"`
	CreatedAt        time.Time `json:"created_at"`
}

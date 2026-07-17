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
	ID               int64
	Company          string
	Title            string
	City             string
	DirectionTags    []string
	Description      string
	SourceName       string
	SourceURL        string
	ApplyURL         string
	PublishedAt      *time.Time
	DeadlineAt       *time.Time
	DiscoveredAt     time.Time
	MatchScore       int
	RecommendReasons []string
	PenaltyReasons   []string
	Status           JobStatus
	Notes            string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type JobRun struct {
	ID               int64
	TriggerType      string
	StartedAt        time.Time
	FinishedAt       *time.Time
	Status           string
	SourcesTotal     int
	SourcesSuccess   int
	SourcesFailed    int
	JobsFound        int
	JobsCreated      int
	JobsDuplicated   int
	ManualCheckCount int
	ErrorSummary     string
}

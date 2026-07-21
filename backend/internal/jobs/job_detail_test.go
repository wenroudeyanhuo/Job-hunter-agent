package jobs

import (
	"testing"
	"time"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/domain"
)

func TestBuildJobDetailUsesProfileSignals(t *testing.T) {
	job := domain.Job{
		ID:               7,
		Company:          "Tencent",
		Title:            "Go Backend Engineer",
		City:             "Shenzhen",
		DirectionTags:    []string{"backend", "go"},
		MatchScore:       82,
		RecommendReasons: []string{"Target city: Shenzhen", "Matches target direction"},
		PenaltyReasons:   []string{"Unclear deadline"},
		Status:           domain.StatusNew,
	}
	profile := CandidateProfile{
		TargetCities:       []string{"Shenzhen"},
		TargetDirections:   []string{"backend", "go"},
		Skills:             []string{"Go", "Distributed Systems"},
		PreferredCompanies: []string{"Tencent"},
		BlockedKeywords:    []string{"外包"},
	}
	decisions := []JobDecision{{
		JobID:     7,
		Action:    "status_changed",
		ToStatus:  "interested",
		CreatedAt: time.Date(2026, 7, 21, 10, 0, 0, 0, time.UTC),
	}}

	detail := BuildJobDetail(job, profile, decisions)
	if detail.Fit.Score <= job.MatchScore {
		t.Fatalf("expected profile fit to lift score, got %#v", detail.Fit)
	}
	if len(detail.Fit.Strengths) == 0 {
		t.Fatalf("expected fit strengths, got %#v", detail.Fit)
	}
	if len(detail.Decisions) != 1 {
		t.Fatalf("expected decision history, got %#v", detail.Decisions)
	}
	if detail.SuggestedAction.Action == "" {
		t.Fatalf("expected suggested action, got %#v", detail.SuggestedAction)
	}
}

package jobs

import (
	"context"
	"testing"
	"time"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/db"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/domain"
)

func TestRepositoryBuildsAgentPreferenceInsights(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo := NewRepository(conn)

	goJob := createInsightJob(t, ctx, repo, domain.Job{
		Company:       "Tencent",
		Title:         "Go Backend Engineer",
		City:          "Shenzhen",
		DirectionTags: []string{"backend", "go"},
		MatchScore:    90,
		Status:        domain.StatusNew,
	})
	frontendJob := createInsightJob(t, ctx, repo, domain.Job{
		Company:       "AdStudio",
		Title:         "Frontend Engineer",
		City:          "Shenzhen",
		DirectionTags: []string{"frontend"},
		MatchScore:    45,
		Status:        domain.StatusNew,
	})
	if err := repo.UpdateStatus(ctx, goJob.ID, domain.StatusInterested); err != nil {
		t.Fatalf("mark interested: %v", err)
	}
	if err := repo.UpdateStatus(ctx, frontendJob.ID, domain.StatusIgnored); err != nil {
		t.Fatalf("mark ignored: %v", err)
	}

	insights, err := repo.BuildAgentPreferenceInsights(ctx, time.Date(2026, 8, 26, 9, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("build insights: %v", err)
	}
	if insights.Summary == "" || insights.TotalDecisions != 2 {
		t.Fatalf("expected summary and decision count, got %#v", insights)
	}
	if !hasPreferenceSignal(insights.InterestedCompanies, "Tencent") {
		t.Fatalf("expected Tencent positive company signal, got %#v", insights.InterestedCompanies)
	}
	if !hasPreferenceSignal(insights.InterestedDirections, "go") || !hasPreferenceSignal(insights.IgnoredDirections, "frontend") {
		t.Fatalf("expected learned direction signals, got positive=%#v ignored=%#v", insights.InterestedDirections, insights.IgnoredDirections)
	}
	if len(insights.Recommendations) == 0 || insights.Recommendations[0].JobID != goJob.ID {
		t.Fatalf("expected interested-like job recommendation insight, got %#v", insights.Recommendations)
	}
	if len(insights.Recommendations[0].Reasons) == 0 {
		t.Fatalf("expected recommendation reasons, got %#v", insights.Recommendations[0])
	}
}

func createInsightJob(t *testing.T, ctx context.Context, repo *Repository, job domain.Job) domain.Job {
	t.Helper()
	job.DiscoveredAt = time.Date(2026, 8, 26, 8, 0, 0, 0, time.UTC)
	if job.ApplyURL == "" {
		job.ApplyURL = "https://example.com/" + job.Company + "/" + job.Title
	}
	created, err := repo.CreateJob(ctx, job)
	if err != nil {
		t.Fatalf("create job: %v", err)
	}
	return created
}

func hasPreferenceSignal(signals []PreferenceSignal, name string) bool {
	for _, signal := range signals {
		if signal.Name == name && signal.Count > 0 {
			return true
		}
	}
	return false
}

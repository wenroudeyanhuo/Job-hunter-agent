package jobs

import (
	"testing"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/db"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/domain"
)

func TestBuildAgentReviewAsksForSetupWithoutSources(t *testing.T) {
	review := BuildAgentReview(nil, nil, nil, nil)

	if review.Health.Label != "Needs setup" {
		t.Fatalf("expected setup health label, got %q", review.Health.Label)
	}
	if review.Focus.Action != "add_recommended_and_crawl" {
		t.Fatalf("expected recommended crawl focus, got %#v", review.Focus)
	}
	if len(review.Decisions) == 0 {
		t.Fatal("expected a setup decision")
	}
}

func TestBuildAgentReviewPrioritizesSourceIssues(t *testing.T) {
	review := BuildAgentReview(
		[]domain.Job{{
			ID:         1,
			Company:    "Tencent",
			Title:      "Go Backend Engineer",
			Status:     domain.StatusNew,
			MatchScore: 82,
		}},
		[]Source{{
			ID:                  2,
			Name:                "Tencent Careers",
			Enabled:             true,
			HealthStatus:        SourceHealthBroken,
			HealthReason:        "parser failed",
			ConsecutiveFailures: 2,
		}},
		[]domain.JobRun{{ID: 1}},
		nil,
	)

	if review.Focus.Action != "inspect_failed_sources" {
		t.Fatalf("expected source issue focus, got %#v", review.Focus)
	}
	if review.Findings[0].Kind != "source_health" {
		t.Fatalf("expected source health to be first finding, got %#v", review.Findings)
	}
	if review.Health.Score >= 100 {
		t.Fatalf("expected source issue to lower health score, got %d", review.Health.Score)
	}
	if review.Health.Label != "Needs review" {
		t.Fatalf("expected review health label, got %q", review.Health.Label)
	}
}

func TestBuildAgentReviewRecommendsStrongMatches(t *testing.T) {
	review := BuildAgentReview(
		[]domain.Job{{
			ID:         1,
			Company:    "DJI",
			Title:      "AI Application Engineer",
			Status:     domain.StatusNew,
			MatchScore: 90,
		}},
		[]Source{{ID: 1, Name: "DJI Careers", Enabled: true, HealthStatus: SourceHealthHealthy}},
		[]domain.JobRun{{ID: 1}},
		[]AgentTask{{ID: 1, Status: AgentTaskStatusOpen}},
	)

	if review.Focus.Action != "review_strong_matches" {
		t.Fatalf("expected strong match focus, got %#v", review.Focus)
	}
	if review.Findings[0].Kind != "recommendation" {
		t.Fatalf("expected recommendation finding, got %#v", review.Findings)
	}
	if review.Stats.StrongMatches != 1 || review.Stats.OpenTasks != 1 {
		t.Fatalf("expected review stats, got %#v", review.Stats)
	}
}

func TestRepositoryCreatesAndListsAgentReviewSnapshots(t *testing.T) {
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo := NewRepository(conn)
	review := BuildAgentReview(
		[]domain.Job{{
			ID:         1,
			Company:    "Tencent",
			Title:      "Backend Engineer",
			Status:     domain.StatusNew,
			MatchScore: 88,
		}},
		[]Source{{ID: 1, Name: "Tencent Careers", Enabled: true, HealthStatus: SourceHealthHealthy}},
		[]domain.JobRun{{ID: 1}},
		[]AgentTask{{ID: 1, Status: AgentTaskStatusOpen}},
	)

	created, err := repo.CreateAgentReviewSnapshot(t.Context(), review, "test")
	if err != nil {
		t.Fatalf("create snapshot: %v", err)
	}
	if created.ID == 0 || created.TriggerType != "test" {
		t.Fatalf("unexpected created snapshot: %#v", created)
	}

	snapshots, err := repo.ListAgentReviewSnapshots(t.Context(), 10)
	if err != nil {
		t.Fatalf("list snapshots: %v", err)
	}
	if len(snapshots) != 1 {
		t.Fatalf("expected one snapshot, got %d", len(snapshots))
	}
	if snapshots[0].Stats.StrongMatches != 1 || snapshots[0].Stats.OpenTasks != 1 {
		t.Fatalf("expected snapshot stats to round trip, got %#v", snapshots[0].Stats)
	}
	if snapshots[0].Review.Focus.Action != "review_strong_matches" {
		t.Fatalf("expected embedded review to round trip, got %#v", snapshots[0].Review)
	}
}

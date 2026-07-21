package jobs

import (
	"testing"

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
}

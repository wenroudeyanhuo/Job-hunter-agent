package jobs

import (
	"strings"
	"testing"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/domain"
)

func TestScoreJobStrongShenzhenBackendGo(t *testing.T) {
	result := ScoreJob(domain.Job{
		Company:     "Tencent",
		Title:       "Go Backend Engineer 2027 Campus",
		City:        "Shenzhen",
		Description: "Campus recruitment for backend microservices with Go.",
		ApplyURL:    "https://example.com/apply",
	})

	if result.HardFiltered {
		t.Fatalf("expected job to pass filter: %s", result.HardFilterReason)
	}
	if result.Job.MatchScore < 70 {
		t.Fatalf("expected strong score >= 70, got %d", result.Job.MatchScore)
	}
	if !contains(result.Job.DirectionTags, "backend") || !contains(result.Job.DirectionTags, "go") {
		t.Fatalf("expected backend and go tags, got %#v", result.Job.DirectionTags)
	}
	if len(result.Job.RecommendReasons) == 0 {
		t.Fatal("expected recommendation reasons")
	}
}

func TestScoreJobAlgorithmAndAIExtraPriority(t *testing.T) {
	algorithm := ScoreJob(domain.Job{
		Company:     "AI Lab",
		Title:       "Algorithm Engineer 2027 Campus",
		City:        "Shenzhen",
		Description: "Machine learning, LLM, RAG, and recommendation systems.",
		ApplyURL:    "https://example.com/apply",
	})
	backend := ScoreJob(domain.Job{
		Company:     "AI Lab",
		Title:       "Backend Engineer 2027 Campus",
		City:        "Shenzhen",
		Description: "Backend service development.",
		ApplyURL:    "https://example.com/apply",
	})

	if algorithm.Job.MatchScore <= backend.Job.MatchScore {
		t.Fatalf("expected algorithm score > backend score, got %d <= %d", algorithm.Job.MatchScore, backend.Job.MatchScore)
	}
	if !contains(algorithm.Job.DirectionTags, "algorithm") || !contains(algorithm.Job.DirectionTags, "ai_application") {
		t.Fatalf("expected algorithm and ai tags, got %#v", algorithm.Job.DirectionTags)
	}
}

func TestScoreJobHardFiltersOutsourcingAndTraining(t *testing.T) {
	cases := []domain.Job{
		{Company: "Some Outsourcing Co", Title: "Java Engineer", Description: "outsourcing role"},
		{Company: "Course Sales", Title: "Frontend Training", Description: "培训课程就业保障"},
	}

	for _, tc := range cases {
		result := ScoreJob(tc)
		if !result.HardFiltered {
			t.Fatalf("expected hard filter for %#v", tc)
		}
		if strings.TrimSpace(result.HardFilterReason) == "" {
			t.Fatal("expected hard filter reason")
		}
	}
}

func TestScoreJobHardFiltersUnclearInternConversion(t *testing.T) {
	result := ScoreJob(domain.Job{
		Company:     "Example",
		Title:       "Backend Intern",
		City:        "Shenzhen",
		Description: "实习岗位，转正不明",
	})

	if !result.HardFiltered {
		t.Fatal("expected unclear conversion internship to be hard filtered")
	}
}

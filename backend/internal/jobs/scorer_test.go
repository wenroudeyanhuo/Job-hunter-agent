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

func TestScoreJobDoesNotTreatDomainAsAICompany(t *testing.T) {
	result := ScoreJob(domain.Job{
		Company:  "Example",
		Title:    "Example Domain",
		ApplyURL: "https://example.com",
	})

	for _, reason := range result.Job.RecommendReasons {
		if reason == "Preferred company category" {
			t.Fatalf("did not expect preferred company category for generic domain: %#v", result.Job.RecommendReasons)
		}
	}
}

func TestScoreJobWithSettingsUsesTargetCitiesAndDirections(t *testing.T) {
	settings := DefaultSettings()
	settings.TargetCities = []string{"Guangzhou"}
	settings.TargetDirections = []string{"backend", "go"}

	guangzhou := ScoreJobWithSettings(domain.Job{
		Company:     "Example",
		Title:       "Go Backend Engineer 2027 Campus",
		City:        "Guangzhou",
		Description: "Campus recruitment for backend microservices with Go.",
		ApplyURL:    "https://example.com/apply",
	}, settings)
	shenzhen := ScoreJobWithSettings(domain.Job{
		Company:     "Example",
		Title:       "Go Backend Engineer 2027 Campus",
		City:        "Shenzhen",
		Description: "Campus recruitment for backend microservices with Go.",
		ApplyURL:    "https://example.com/apply",
	}, settings)

	if guangzhou.Job.MatchScore <= shenzhen.Job.MatchScore {
		t.Fatalf("expected configured target city to score higher, got %d <= %d", guangzhou.Job.MatchScore, shenzhen.Job.MatchScore)
	}
	if !contains(guangzhou.Job.RecommendReasons, "Target city: Guangzhou") {
		t.Fatalf("expected target city reason, got %#v", guangzhou.Job.RecommendReasons)
	}
	if !contains(guangzhou.Job.RecommendReasons, "Matches target direction") {
		t.Fatalf("expected target direction reason, got %#v", guangzhou.Job.RecommendReasons)
	}
}

func TestScoreJobWithSettingsDoesNotRewardUnselectedDirections(t *testing.T) {
	settings := DefaultSettings()
	settings.TargetDirections = []string{"algorithm"}

	frontend := ScoreJobWithSettings(domain.Job{
		Company:     "Example",
		Title:       "Frontend Engineer 2027 Campus",
		City:        "Shenzhen",
		Description: "React and TypeScript web development.",
		ApplyURL:    "https://example.com/apply",
	}, settings)
	algorithm := ScoreJobWithSettings(domain.Job{
		Company:     "Example",
		Title:       "Algorithm Engineer 2027 Campus",
		City:        "Shenzhen",
		Description: "Machine learning and recommendation systems.",
		ApplyURL:    "https://example.com/apply",
	}, settings)

	if frontend.Job.MatchScore >= algorithm.Job.MatchScore {
		t.Fatalf("expected selected algorithm direction to score higher, got frontend %d algorithm %d", frontend.Job.MatchScore, algorithm.Job.MatchScore)
	}
	if contains(frontend.Job.RecommendReasons, "Matches target direction") {
		t.Fatalf("did not expect unselected frontend direction to be rewarded: %#v", frontend.Job.RecommendReasons)
	}
}

func TestScoreJobWithSettingsHardFiltersExcludedKeywords(t *testing.T) {
	settings := DefaultSettings()
	settings.ExcludedKeywords = []string{"remote-only"}

	result := ScoreJobWithSettings(domain.Job{
		Company:     "Example",
		Title:       "Go Backend Engineer",
		City:        "Shenzhen",
		Description: "This is a remote-only contractor role.",
	}, settings)

	if !result.HardFiltered {
		t.Fatal("expected configured excluded keyword to hard filter job")
	}
	if result.HardFilterReason != "Matched excluded keyword: remote-only" {
		t.Fatalf("unexpected hard filter reason %q", result.HardFilterReason)
	}
}

func TestScoreJobMarksGenericCareerHomeForManualCheck(t *testing.T) {
	result := ScoreJob(domain.Job{
		Company:     "Tencent",
		Title:       "Tencent Careers",
		Description: "Explore career opportunities and learn about our company culture.",
		ApplyURL:    "https://careers.tencent.com/",
		SourceURL:   "https://careers.tencent.com/",
	})

	if result.HardFiltered {
		t.Fatalf("expected generic career page to stay reviewable: %s", result.HardFilterReason)
	}
	if result.Job.Status != domain.StatusManualCheck {
		t.Fatalf("expected generic career page to need manual_check, got %q", result.Job.Status)
	}
	if !contains(result.Job.PenaltyReasons, "Low confidence job posting") {
		t.Fatalf("expected low confidence penalty, got %#v", result.Job.PenaltyReasons)
	}
}

func TestScoreJobKeepsConcretePostingAsNew(t *testing.T) {
	result := ScoreJob(domain.Job{
		Company:     "Tencent",
		Title:       "Go Backend Engineer 2027 Campus - Shenzhen",
		City:        "Shenzhen",
		Description: "Campus recruitment role building backend microservices with Go. Apply online before the deadline.",
		ApplyURL:    "https://careers.tencent.com/job/123",
	})

	if result.Job.Status != domain.StatusNew {
		t.Fatalf("expected concrete posting to stay new, got %q with penalties %#v", result.Job.Status, result.Job.PenaltyReasons)
	}
	if contains(result.Job.PenaltyReasons, "Low confidence job posting") {
		t.Fatalf("did not expect low confidence penalty: %#v", result.Job.PenaltyReasons)
	}
}

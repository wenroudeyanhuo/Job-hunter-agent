package jobs

import (
	"context"
	"testing"
	"time"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/db"
)

func TestOnboardingHealthIncludesPersonalWizardSteps(t *testing.T) {
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo := NewRepository(conn)

	health, err := repo.BuildOnboardingHealth(context.Background(), true, false, false, time.Date(2026, 8, 25, 9, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("build onboarding health: %v", err)
	}

	if len(health.WizardSteps) == 0 {
		t.Fatalf("expected wizard steps, got %#v", health)
	}
	if !hasWizardStep(health.WizardSteps, "profile") || !hasWizardStep(health.WizardSteps, "sources") || !hasWizardStep(health.WizardSteps, "model") {
		t.Fatalf("expected personal setup wizard steps, got %#v", health.WizardSteps)
	}
	if health.WizardSteps[0].Done {
		t.Fatalf("expected first setup step to remain open for a fresh install, got %#v", health.WizardSteps[0])
	}
}

func hasWizardStep(steps []OnboardingWizardStep, key string) bool {
	for _, step := range steps {
		if step.Key == key {
			return true
		}
	}
	return false
}

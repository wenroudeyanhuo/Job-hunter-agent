package crawl

import (
	"context"
	"testing"
)

func TestStartSchedulerAcceptsValidSpecs(t *testing.T) {
	stop, err := StartScheduler(context.Background(), nil, []string{"0 9 * * *", "0 12 * * *", "0 18 * * *"})
	if err != nil {
		t.Fatalf("expected valid specs, got %v", err)
	}
	stop()
}

func TestStartSchedulerRejectsInvalidSpec(t *testing.T) {
	_, err := StartScheduler(context.Background(), nil, []string{"not a cron spec"})
	if err == nil {
		t.Fatal("expected invalid spec error")
	}
}

package jobs

import (
	"context"
	"testing"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/db"
)

func TestRepositoryCreatesAndListsAgentEvents(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo := NewRepository(conn)

	created, err := repo.CreateAgentEvent(ctx, AgentEventInput{
		Type:    "crawl_completed",
		Title:   "Crawl completed",
		Summary: "Created 3 jobs and found 2 duplicates.",
		Level:   "success",
	})
	if err != nil {
		t.Fatalf("create agent event: %v", err)
	}
	if created.ID == 0 || created.CreatedAt.IsZero() {
		t.Fatalf("expected stored event metadata, got %#v", created)
	}

	events, err := repo.ListAgentEvents(ctx, 10)
	if err != nil {
		t.Fatalf("list agent events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected one event, got %d", len(events))
	}
	if events[0].Type != "crawl_completed" || events[0].Level != "success" {
		t.Fatalf("unexpected event: %#v", events[0])
	}
}

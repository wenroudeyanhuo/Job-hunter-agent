package jobs

import (
	"context"
	"testing"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/db"
)

func TestRepositoryRecordsAndUpdatesAgentActionRequests(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo := NewRepository(conn)

	if err := repo.RecordAgentActionRequests(ctx, "chat", []AgentCommandAction{
		{Type: "sync_application_plans", Target: "applications", Detail: "准备投递计划"},
		{Type: "auto_apply_resume", Target: "external", Detail: "危险动作"},
	}); err != nil {
		t.Fatalf("record action requests: %v", err)
	}
	requests, err := repo.ListAgentActionRequests(ctx, AgentActionRequestStatusPending)
	if err != nil {
		t.Fatalf("list action requests: %v", err)
	}
	if len(requests) != 1 {
		t.Fatalf("expected one safe pending request, got %#v", requests)
	}
	if requests[0].ActionType != "sync_application_plans" || requests[0].Status != AgentActionRequestStatusPending {
		t.Fatalf("unexpected request: %#v", requests[0])
	}

	updated, err := repo.UpdateAgentActionRequestStatus(ctx, requests[0].ID, AgentActionRequestStatusApproved)
	if err != nil {
		t.Fatalf("approve action request: %v", err)
	}
	if updated.Status != AgentActionRequestStatusApproved || updated.ResolvedAt == nil {
		t.Fatalf("expected approved request with resolved timestamp, got %#v", updated)
	}
}

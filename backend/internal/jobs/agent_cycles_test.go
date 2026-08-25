package jobs

import (
	"context"
	"testing"
	"time"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/db"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/domain"
)

func TestRepositoryRecordsAndListsAgentCycles(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo := NewRepository(conn)

	cycle := RunRecruitingAgentCycle(MultiAgentCycleInput{
		Now: time.Date(2026, 8, 25, 9, 0, 0, 0, time.UTC),
		Jobs: []domain.Job{
			{Company: "Tencent", Title: "Go Backend Engineer", City: "Shenzhen", MatchScore: 86, Status: domain.StatusNew},
		},
		Sources: []Source{
			{Name: "Tencent Careers", Enabled: true, HealthStatus: SourceHealthHealthy},
		},
		Memory: SemanticMemoryStats{TotalItems: 1, JobItems: 1},
	})

	record, err := repo.RecordAgentCycle(ctx, cycle)
	if err != nil {
		t.Fatalf("record agent cycle: %v", err)
	}
	if record.ID == 0 {
		t.Fatal("expected persisted cycle id")
	}
	if record.Summary == "" || record.ReadinessScore == 0 {
		t.Fatalf("expected cycle summary and readiness score, got %#v", record)
	}
	if record.OrchestratorProvider != MultiAgentOrchestratorEinoReady {
		t.Fatalf("expected orchestrator provider to round trip, got %q", record.OrchestratorProvider)
	}
	if len(record.Trace) != 4 {
		t.Fatalf("expected four agent trace entries, got %#v", record.Trace)
	}
	if len(record.Actions) == 0 {
		t.Fatalf("expected approval-gated actions, got %#v", record.Actions)
	}

	list, err := repo.ListAgentCycles(ctx, 10)
	if err != nil {
		t.Fatalf("list agent cycles: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected one cycle, got %d", len(list))
	}
	if list[0].ID != record.ID || list[0].Trace[0].AgentKey != MultiAgentSourceScout {
		t.Fatalf("expected latest cycle to round trip, got %#v", list[0])
	}
}

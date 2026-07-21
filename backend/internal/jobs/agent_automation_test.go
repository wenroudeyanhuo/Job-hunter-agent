package jobs

import (
	"testing"
	"time"
)

func TestBuildAgentAutomationStateFindsStaleTasks(t *testing.T) {
	now := time.Date(2026, 7, 21, 10, 0, 0, 0, time.UTC)
	settings := DefaultSettings()
	settings.AutoDutyReportEnabled = true
	settings.DutyReportTime = "18:30"
	settings.TaskSLAHours = 4

	state := BuildAgentAutomationState(settings, []AgentTask{
		{
			ID:        1,
			Title:     "Review Tencent",
			Detail:    "Go Backend Engineer / Shenzhen",
			Status:    AgentTaskStatusOpen,
			CreatedAt: now.Add(-5 * time.Hour),
		},
		{
			ID:        2,
			Title:     "Done task",
			Status:    AgentTaskStatusDone,
			CreatedAt: now.Add(-8 * time.Hour),
		},
	}, now)

	if !state.DutyReportEnabled || state.DutyReportTime != "18:30" {
		t.Fatalf("expected duty report settings, got %#v", state)
	}
	if state.StaleTaskCount != 1 || len(state.StaleTasks) != 1 {
		t.Fatalf("expected one stale open task, got %#v", state)
	}
	if state.NextDutyReportAt == "" {
		t.Fatalf("expected next report time")
	}
}
